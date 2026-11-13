package app

import (
	"bufio"
	"errors"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/xddcode/markgo/internal/mdeditor"
	"github.com/xddcode/markgo/internal/session"
	"github.com/xddcode/markgo/internal/storage"
)

func saveCheckpoint(t *testing.T, a *App, store *session.Store, revision uint64, discard bool) session.State {
	t.Helper()
	state, err := a.captureRecovery(discard).serialize()
	if err == nil {
		err = store.Save(revision, state)
	}
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func openFixture(t *testing.T, a *App, path, source string) *mdeditor.File {
	t.Helper()
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	file, snapshot, err := loadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	a.addOpened(file, snapshot)
	return file
}

func TestRecoveryRestoresDraftsPositionsAndDiskGuards(t *testing.T) {
	dir := t.TempDir()
	store := &session.Store{Path: filepath.Join(dir, "session.json")}
	a := New()
	a.current().State.Buffer.Set("尚未命名的想法\n")
	path := filepath.Join(dir, "essay.md")
	f := openFixture(t, a, path, "# A page\r\n\r\nOriginal.\r\n")
	f.State.Buffer.Set("# A page\n\nMy unfinished paragraph.\n")
	f.State.Select(3, 6)
	a.reading, a.sidebar, a.sideView = true, false, 1
	saveCheckpoint(t, a, store, 1, false)
	b := New()
	if err := b.loadSession(store); err != nil {
		t.Fatal(err)
	}
	if len(b.files) != 2 || b.current().ID != f.ID || !b.current().Modified() || b.sidebar || !b.reading || b.sideView != 1 {
		t.Fatalf("workspace did not restore: %+v", b)
	}
	if b.current().State.Buffer.String() != f.State.Buffer.String() || b.current().State.Selection() != f.State.Selection() {
		t.Fatal("source or selection changed during recovery")
	}
	if b.files[0].State.Buffer.String() != "尚未命名的想法\n" || !b.files[0].Modified() || len(b.recovered) != 2 {
		t.Fatal("untitled draft was lost or falsely marked saved")
	}
	data, err := b.current().CaptureForSave().Bytes()
	if err != nil || !strings.Contains(string(data), "\r\n") {
		t.Fatalf("recovery lost CRLF: %q %v", data, err)
	}
	if err := storage.Check(b.disk[f.ID]); err != nil {
		t.Fatal("unchanged disk did not retain its guarded snapshot", err)
	}
	saveCheckpoint(t, b, store, 2, false)
	c := New()
	if err := c.loadSession(store); err != nil || len(c.files) != 2 {
		t.Fatalf("second recovery duplicated drafts: %d %v", len(c.files), err)
	}
	c.newDocument()
	if c.current().ID == f.ID || c.current().ID == a.files[0].ID {
		t.Fatal("new document reused a recovered identity")
	}
	disk, _ := os.ReadFile(path)
	if string(disk) != "# A page\r\n\r\nOriginal.\r\n" {
		t.Fatal("recovery wrote into the original file")
	}
}

func TestRecoveryKeepsExternalAndMissingOriginalsSeparate(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(fmt.Sprint(remove), func(t *testing.T) {
			dir := t.TempDir()
			store := &session.Store{Path: filepath.Join(dir, "session.json")}
			a := New()
			path := filepath.Join(dir, "note.md")
			file := openFixture(t, a, path, "original")
			file.State.Buffer.Set("recovered edits")
			saveCheckpoint(t, a, store, 1, false)
			var err error
			if remove {
				err = os.Remove(path)
			} else {
				err = os.WriteFile(path, []byte("external edits"), 0o600)
			}
			if err != nil {
				t.Fatal(err)
			}
			b := New()
			if err := b.loadSession(store); err != nil {
				t.Fatal(err)
			}
			if b.current().State.Buffer.String() != "recovered edits" || b.disk[file.ID].Path != "" || !b.conflicts[file.ID] {
				t.Fatal("recovery granted overwrite permission over an unavailable or changed file")
			}
			tt := ui.NewTester(b.View, 960, 640)
			if !tt.HasText("Save a copy") || !tt.HasText("Recovered draft") {
				t.Fatalf("recovery does not explain the conflict: %q", tt.Texts())
			}
		})
	}
}

func TestRecoveryRecognizesCompletedSaveAndExplicitDiscard(t *testing.T) {
	dir := t.TempDir()
	store := &session.Store{Path: filepath.Join(dir, "session.json")}
	a := New()
	a.current().State.Buffer.Set("discard this untitled text")
	path := filepath.Join(dir, "note.md")
	f := openFixture(t, a, path, "original")
	f.State.Buffer.Set("saved immediately before a crash")
	saveCheckpoint(t, a, store, 1, false)
	if _, err := storage.Save(a.disk[f.ID], []byte(f.State.Buffer.String())); err != nil {
		t.Fatal(err)
	}
	b := New()
	if err := b.loadSession(store); err != nil {
		t.Fatal(err)
	}
	if b.current().Modified() || b.recovered[f.ID] != "" {
		t.Fatal("an already completed disk save was shown as an unsaved recovery")
	}
	b.current().State.Buffer.Set("discard this named edit")
	saveCheckpoint(t, b, store, 2, true)
	c := New()
	if err := c.loadSession(store); err != nil || len(c.files) != 1 || c.current().Modified() || c.current().State.Buffer.String() != f.State.Buffer.String() {
		t.Fatal("explicit discard resurrected old text or removed the original", err)
	}
}

func TestCheckpointCaptureDoesNotConsumeNewerEditsOrOtherDrafts(t *testing.T) {
	a := New()
	f := a.current()
	f.State.Buffer.Set("checkpoint")
	capture := a.captureRecovery(false)
	f.State.Buffer.Set("newer typing")
	state, err := capture.serialize()
	if err != nil || state.Documents[0].Source != "checkpoint" {
		t.Fatal("worker snapshot was mutable", err)
	}
	a.sessionRevision = 2
	a.acceptRecovery(1, state, nil)
	if len(a.backedUp) != 0 {
		t.Fatal("old completion superseded a newer checkpoint")
	}
	a.acceptRecovery(2, state, nil)
	tt := ui.NewTester(a.View, 960, 640)
	if tt.HasText("Draft backed up") || !f.Modified() || f.State.Buffer.String() != "newer typing" {
		t.Fatal("checkpoint completion consumed or confirmed newer edits")
	}
	a.newDocument()
	a.current().State.Buffer.Set("keep this second draft")
	state, err = a.captureRecovery(false).without(f.ID).serialize()
	if err != nil || len(state.Documents) != 1 || state.Documents[0].Source != "keep this second draft" {
		t.Fatal("discarding one checkpoint affected another", err)
	}
	a.acceptRecovery(2, state, errors.New("disk full"))
	tt.Frame()
	if !tt.HasText("Recovery is unavailable. Save your documents to keep your work.") {
		t.Fatal("failed backup was hidden")
	}
}

func TestRecoveryDebounceHasBoundedDelayAndNoIdleWrites(t *testing.T) {
	a := New()
	now := time.Unix(100, 0)
	if delay, ready := a.recoveryDue(now); ready || delay != recoveryDelay {
		t.Fatalf("first recovery scheduling: %v %v", delay, ready)
	}
	for i := 1; i < 10; i++ {
		a.current().State.Buffer.Set(fmt.Sprint(i))
		if _, ready := a.recoveryDue(now.Add(time.Duration(i) * 500 * time.Millisecond)); ready {
			t.Fatal("continuous input was not debounced")
		}
	}
	if _, ready := a.recoveryDue(now.Add(recoveryMaxDelay)); !ready {
		t.Fatal("continuous typing postponed recovery forever")
	}
	if _, ready := a.recoveryDue(now.Add(time.Minute)); ready {
		t.Fatal("idle view kept writing checkpoints")
	}
}

func TestMissingAndInvalidSessionLeaveFilesReviewable(t *testing.T) {
	dir := t.TempDir()
	store := &session.Store{Path: filepath.Join(dir, "session.json")}
	state := session.State{Schema: 1, Sidebar: true, Documents: []session.Document{{ID: "x", Name: "gone.md", Path: filepath.Join(dir, "gone.md")}}}
	if err := store.Save(1, state); err != nil {
		t.Fatal(err)
	}
	a := New()
	if err := a.loadSession(store); err != nil || len(a.recentMissing) != 1 || len(a.recent) != 1 {
		t.Fatal("missing file was silently forgotten", err)
	}
	if err := os.WriteFile(store.Path, []byte("{truncated"), 0o600); err != nil {
		t.Fatal(err)
	}
	b := New()
	if err := b.loadSession(store); err == nil || b.sessionStore != nil || b.sessionError == "" {
		t.Fatal("invalid session was attached for automatic overwriting")
	}
}

func TestRecoverySurvivesKilledProcess(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestRecoveryProcessHelper$")
	cmd.Env = append(os.Environ(), "MARKGO_TEST_RECOVERY_DIR="+dir)
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
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	ready := make(chan string, 1)
	go func() { line, _ := bufio.NewReader(out).ReadString('\n'); ready <- line }()
	select {
	case line := <-ready:
		if line != "checkpoint ready\n" {
			t.Fatalf("child did not checkpoint: %q", line)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("child checkpoint timed out")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	a := New()
	store := &session.Store{Path: filepath.Join(dir, "session.json")}
	if err := a.loadSession(store); err != nil {
		t.Fatal(err)
	}
	if len(a.files) != 2 || a.files[0].State.Buffer.String() != "未命名草稿" || a.current().State.Buffer.String() != "named draft after typing" {
		t.Fatal("process termination lost complete checkpoints")
	}
	disk, _ := os.ReadFile(filepath.Join(dir, "named.md"))
	if string(disk) != "original disk bytes" {
		t.Fatal("checkpoint replaced the original file")
	}
	saveCheckpoint(t, a, store, 2, false)
	b := New()
	if err := b.loadSession(store); err != nil || len(b.files) != 2 {
		t.Fatal("repeat restart duplicated recovered drafts", err)
	}
}

func TestRecoveryProcessHelper(t *testing.T) {
	dir := os.Getenv("MARKGO_TEST_RECOVERY_DIR")
	if dir == "" {
		return
	}
	a := New()
	a.current().State.Buffer.Set("未命名草稿")
	f := openFixture(t, a, filepath.Join(dir, "named.md"), "original disk bytes")
	f.State.Buffer.Set("named draft after typing")
	saveCheckpoint(t, a, &session.Store{Path: filepath.Join(dir, "session.json")}, 1, false)
	fmt.Println("checkpoint ready")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

func TestRecentDocumentsKeyboardFilteringAndSafeClear(t *testing.T) {
	a := New()
	f := a.current()
	f.State.Buffer.Set("keep my writing")
	a.rememberRecent(filepath.Join(t.TempDir(), "Travel notes.md"))
	second := filepath.Join(t.TempDir(), "Book draft.md")
	a.rememberRecent(second)
	a.rememberRecent(second)
	if len(a.recent) != 2 {
		t.Fatal("recent history duplicated a file")
	}
	tt := ui.NewTester(a.View, 960, 640)
	click(t, tt, "Document menu")
	click(t, tt, "Recent documents…")
	tt.Advance(180 * time.Millisecond)
	tt.Type("book")
	if tt.HasText("Travel notes.md") || !tt.HasText("Book draft.md") {
		t.Fatalf("recent filtering: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if len(a.openQueue) != 1 || a.openQueue[0] != second {
		t.Fatal("recent action did not open the selected path")
	}
	click(t, tt, "Clear recent history")
	if len(a.recent) != 0 || len(a.files) != 1 || f.State.Buffer.String() != "keep my writing" || !tt.HasText("Your next page starts here") {
		t.Fatal("clearing history affected documents or left stale empty state")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Advance(180 * time.Millisecond)
	if a.recentOpen {
		t.Fatal("Escape did not dismiss recent documents")
	}
	if active, disabled := tt.TextInputActive(); !active || disabled {
		t.Fatal("closing recent documents did not return to writing")
	}
}

func TestMissingRecentCanBeRemovedWithoutOpeningOrDeletingFile(t *testing.T) {
	a := New()
	path := filepath.Join(t.TempDir(), "remains.md")
	openFixture(t, a, path, "keep this file")
	a.recentMissing[path] = "no longer available"
	a.recentOpen = true
	tt := ui.NewTester(a.View, 960, 640)
	click(t, tt, "Remove recent remains.md")
	if len(a.recent) != 0 || len(a.recentMissing) != 0 || len(a.openQueue) != 0 {
		t.Fatal("removing a recent entry also opened it or left the warning behind")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "keep this file" || a.current().Path != path {
		t.Fatal("history removal touched the document", err)
	}
}

func TestExitPromptCanCancelKeepOrDiscardDrafts(t *testing.T) {
	for _, action := range []string{"Cancel closing", "Keep drafts and close", "Discard unsaved changes"} {
		t.Run(action, func(t *testing.T) {
			a := New()
			a.sessionStore = &session.Store{Path: filepath.Join(t.TempDir(), "session.json")}
			file := a.current()
			file.State.Buffer.Set("unfinished writing")
			file.State.Select(3, 7)
			tt := ui.NewTester(a.View, 960, 640)
			a.confirmExit(false)
			tt.Frame()
			tt.Type("must not enter the disabled document")
			if file.State.Buffer.String() != "unfinished writing" {
				t.Fatal("closing prompt allowed background editing")
			}
			click(t, tt, action)
			if action == "Cancel closing" {
				tt.Advance(200 * time.Millisecond)
				if a.closing || a.fileBusy || a.allowExit || file.State.Selection().Anchor != 3 || file.State.Selection().Head != 7 {
					t.Fatal("cancel changed writing state or left the workbench disabled")
				}
				if active, disabled := tt.TextInputActive(); !active || disabled {
					t.Fatalf("cancel did not return to writing: active=%v disabled=%v focused=%v pending=%v", active, disabled, tt.Focused(file.Name), a.returnToWriting)
				}
				return
			}
			a.WaitForSession()
			state, err := a.sessionStore.Load()
			if err != nil {
				t.Fatal(err)
			}
			if action == "Keep drafts and close" && (len(state.Documents) != 1 || state.Documents[0].Source != "unfinished writing") {
				t.Fatal("keep drafts did not persist the current buffer")
			}
			if action == "Discard unsaved changes" && len(state.Documents) != 0 {
				t.Fatal("discard left a recoverable draft")
			}
		})
	}
	a := New()
	a.current().State.Buffer.Set("keep after failed backup")
	a.showExitPrompt(false, true)
	tt := ui.NewTester(a.View, 960, 640)
	tt.Key(0, ui.KeyEscape)
	tt.Advance(200 * time.Millisecond)
	if a.allowExit || a.exitPrompt || a.closing || a.current().State.Buffer.String() != "keep after failed backup" {
		t.Fatal("dismissing a failed backup dialog discarded writing")
	}
}

func TestRecoveredMixedLineEndingsStillRequireAnExplicitCopy(t *testing.T) {
	dir := t.TempDir()
	store := &session.Store{Path: filepath.Join(dir, "session.json")}
	a := New()
	f := openFixture(t, a, filepath.Join(dir, "mixed.md"), "one\r\ntwo\nthree\r\n")
	f.State.Buffer.Set("one\ntwo edited\nthree\n")
	saveCheckpoint(t, a, store, 1, false)
	b := New()
	if err := b.loadSession(store); err != nil {
		t.Fatal(err)
	}
	if _, err := b.current().CaptureForSave().Bytes(); !errors.Is(err, mdeditor.ErrMixedLineEndings) {
		t.Fatal("recovering a draft bypassed mixed-ending protection", err)
	}
	if got := b.current().CaptureForSave().OriginalSource(); got != "one\r\ntwo\nthree\r\n" {
		t.Fatalf("recovery lost exact original bytes: %q", got)
	}
}

// Opt-in screenshots use the actual application view, with no user files or
// desktop window. They are layout evidence, not native input-method evidence.
func TestRecoveryScreenshots(t *testing.T) {
	dir := os.Getenv("MARKGO_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set MARKGO_SCREENSHOT_DIR to capture UI evidence")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"paper", "warm", "graphite"} {
		for _, scene := range []string{"recovered", "recent", "missing", "empty", "no-results", "backup-error", "conflict", "recovered-long", "exit", "exit-error"} {
			a := New()
			a.appearance.Theme = theme
			f := mdeditor.RecoverFile("A quieter morning.md", "# A quieter morning\n\nSome thoughts are worth returning to.\n\nThe house is still. There is time to finish a sentence, to find the right word, to leave a little space.\n\n## Small beginnings\n\n- Open the window\n- Make a cup of tea\n- Write one honest paragraph\n", "")
			a.files, a.selected = []*mdeditor.File{f}, f.ID
			a.recovered[f.ID] = "Your unfinished writing is ready. Review it, then save when you are ready."
			a.backedUp[f.ID] = f.State.Buffer.Version()
			a.recent = []string{"/Writing/Essays/A quieter morning.md", "/Writing/Notebook/关于留白与慢下来的日常.md", "/Writing/Research/Reading notes.md", "/Writing/Projects/An exceptionally long document title that must stay within the panel.md"}
			a.recentOpen = scene == "recent" || scene == "missing" || scene == "empty" || scene == "no-results"
			if scene == "conflict" {
				f.SetPath("/Writing/Essays/A quieter morning.md")
				a.conflicts[f.ID] = true
				a.recovered[f.ID] = "The original is unavailable or has changed. Your draft is intact; save a copy to keep both."
			}
			if scene == "recovered-long" {
				f.Name = strings.Repeat("关于留白与慢下来的日常", 10) + ".md"
			}
			if scene == "missing" {
				a.recentMissing[a.recent[0]] = "The file is no longer available at this path."
			}
			if scene == "empty" {
				a.recent = nil
			}
			if scene == "no-results" {
				a.recentQuery = "not found"
			}
			if scene == "backup-error" {
				a.sessionError = "There is no space left on the device."
				a.sessionStore = &session.Store{}
			}
			if scene == "exit" || scene == "exit-error" {
				a.sessionStore = &session.Store{}
				if scene == "exit-error" {
					a.sessionError = "There is no space left on the device."
				}
				a.showExitPrompt(false, scene == "exit-error")
			}
			tt := ui.NewTester(a.View, 960, 640)
			tt.Advance(200 * time.Millisecond)
			labels := []string{"Save recovered draft", "Dismiss recovery notice"}
			if a.recentOpen {
				labels = []string{"Close recent documents", "Search recent documents", "Clear recent history", "Browse for a document"}
			}
			if scene == "exit" {
				labels = []string{"Cancel closing", "Discard unsaved changes", "Keep drafts and close", "Save all before closing"}
			} else if scene == "exit-error" {
				labels = []string{"Exit without updating recovery", "Stay open after recovery failure"}
			}
			for _, label := range labels {
				r, ok := tt.Find(label)
				if !ok || r.H < 40 || r.X < 0 || r.Y < 0 || r.X+r.W > 960 || r.Y+r.H > 640 {
					t.Fatalf("%s/%s: unusable %s at %+v", theme, scene, label, r)
				}
			}
			path := filepath.Join(dir, theme+"-"+scene+".png")
			out, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			if scene == "recovered" || scene == "recent" {
				wide := ui.NewTester(a.View, 1320, 840)
				wide.Advance(200 * time.Millisecond)
				out, err := os.Create(filepath.Join(dir, theme+"-"+scene+"-wide.png"))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(out, wide.Image())
				_ = out.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
			err = png.Encode(out, tt.Image())
			_ = out.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}
