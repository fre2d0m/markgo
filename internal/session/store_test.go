package session

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func draftState(source string) State {
	return State{Schema: SchemaVersion, Sidebar: true, Active: "draft", Documents: []Document{{ID: "draft", Name: "Untitled.md", Dirty: true, Source: source}}}
}

func TestCrashBeforeAtomicReplacementPreservesPreviousCheckpoint(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestInterruptedStoreProcessHelper$")
	cmd.Env = append(os.Environ(), "MARKGO_TEST_INTERRUPTED_DIR="+dir)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	ready := make(chan string, 1)
	go func() { line, _ := bufio.NewReader(out).ReadString('\n'); ready <- line }()
	select {
	case line := <-ready:
		if line != "temporary checkpoint synced\n" {
			t.Fatalf("unexpected child output: %q", line)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("interrupted checkpoint timed out")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	store := &Store{Path: filepath.Join(dir, "session.json")}
	state, err := store.Load()
	if err != nil || state.Documents[0].Source != "last complete draft" {
		t.Fatal("interrupted replacement damaged the previous checkpoint", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "session.json" {
		t.Fatalf("interrupted temporary file was not cleaned: %v %v", entries, err)
	}
}

func TestInterruptedStoreProcessHelper(t *testing.T) {
	dir := os.Getenv("MARKGO_TEST_INTERRUPTED_DIR")
	if dir == "" {
		return
	}
	store := &Store{Path: filepath.Join(dir, "session.json")}
	if err := store.Save(1, draftState("last complete draft")); err != nil {
		t.Fatal(err)
	}
	store.replace = func(from, to string) error {
		fmt.Println("temporary checkpoint synced")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		return os.Rename(from, to)
	}
	if err := store.Save(2, draftState("not yet committed")); err != nil {
		t.Fatal(err)
	}
}

func TestOrderedAtomicRecoveryDoesNotResurrectOldDrafts(t *testing.T) {
	store := &Store{Path: filepath.Join(t.TempDir(), "private", "session.json")}
	var workers sync.WaitGroup
	for revision := uint64(1); revision <= 40; revision++ {
		workers.Go(func() {
			state := draftState("an older draft")
			if revision == 40 {
				state.Documents = nil // Explicit discard is the newest decision.
			}
			if err := store.Save(revision, state); err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
	loaded, err := store.Load()
	if err != nil || len(loaded.Documents) != 0 {
		t.Fatalf("discarded revision resurrected: %+v, %v", loaded, err)
	}
	entries, err := os.ReadDir(filepath.Dir(store.Path))
	if err != nil || len(entries) != 1 || entries[0].Name() != "session.json" {
		t.Fatalf("temporary recovery files leaked: %v, %v", entries, err)
	}
}

func TestFailedRecoveryWriteKeepsLastCompleteCheckpoint(t *testing.T) {
	store := &Store{Path: filepath.Join(t.TempDir(), "session.json")}
	if err := store.Save(1, draftState("完整草稿\n")); err != nil {
		t.Fatal(err)
	}
	invalid := draftState("discarded")
	invalid.Schema = 999
	if err := store.Save(3, invalid); err == nil {
		t.Fatal("unsupported version was written")
	}
	if err := store.Save(2, draftState("late stale writer")); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil || loaded.Documents[0].Source != "完整草稿\n" {
		t.Fatalf("failed write damaged the complete checkpoint: %+v %v", loaded, err)
	}
	if err := store.Save(4, draftState("retry succeeds")); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.Load()
	if err != nil || loaded.Documents[0].Source != "retry succeeds" {
		t.Fatalf("retry was blocked: %+v %v", loaded, err)
	}
}

func TestInvalidRecoveryIsRejectedWithoutChangingItsBytes(t *testing.T) {
	for _, input := range []string{
		`{`, `{}`, `{"schema":2}`, `{"schema":1,"documents":[{"id":"a","name":"note.md","path":"relative.md"}]}`,
		`{"schema":1,"documents":[{"id":"a","name":"one"},{"id":"a","name":"two"}]}`,
		`{"schema":1,"documents":[{"id":"a","name":"one","anchor":-1}]}`,
	} {
		t.Run(input, func(t *testing.T) {
			store := &Store{Path: filepath.Join(t.TempDir(), "session.json")}
			if err := os.WriteFile(store.Path, []byte(input), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := store.Load(); err == nil {
				t.Fatal("invalid recovery was accepted")
			}
			data, _ := os.ReadFile(store.Path)
			if string(data) != input {
				t.Fatal("rejected recovery was overwritten")
			}
		})
	}
}

func TestRecoveryBoundsAndDestinationFailure(t *testing.T) {
	dir := t.TempDir()
	store := &Store{Path: filepath.Join(dir, "session.json")}
	if err := store.Save(1, draftState("safe")); err != nil {
		t.Fatal(err)
	}
	big := draftState(strings.Repeat("x", MaxBytes+1))
	if err := store.Save(2, big); err == nil {
		t.Fatal("unbounded recovery accepted")
	}
	loaded, err := store.Load()
	if err != nil || loaded.Documents[0].Source != "safe" {
		t.Fatal("size failure replaced previous checkpoint")
	}
	blocked := &Store{Path: filepath.Join(store.Path, "cannot-create.json")}
	if err := blocked.Save(1, draftState("failed")); err == nil {
		t.Fatal("write into a regular file unexpectedly succeeded")
	}
	if err := (&Store{Path: dir}).Save(1, draftState("failed")); err == nil {
		t.Fatal("directory was replaced by recovery data")
	}
}
