package preferences

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestLatestPreferenceWinsAndNoTemporaryFilesRemain(t *testing.T) {
	s := &Store{Path: filepath.Join(t.TempDir(), "settings", "appearance.json")}
	if data, err := s.Load(); err != nil || data != nil {
		t.Fatalf("new store: %q %v", data, err)
	}
	var workers sync.WaitGroup
	for revision := uint64(1); revision <= 25; revision++ {
		workers.Go(func() {
			if err := s.Save(revision, []byte{byte(revision)}); err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
	data, err := s.Load()
	if err != nil || len(data) != 1 || data[0] != 25 {
		t.Fatalf("latest preferences: %v, %v", data, err)
	}
	entries, err := os.ReadDir(filepath.Dir(s.Path))
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files: %v, %v", entries, err)
	}
}

func TestFailedSaveLeavesNoPartialSettingsAndCanRetry(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Path: filepath.Join(dir, "appearance.json")}
	if err := os.Mkdir(s.Path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(1, []byte(`{"theme":"warm"}`)); err == nil {
		t.Fatal("save replaced a directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || !entries[0].IsDir() {
		t.Fatalf("failed write left partial settings: %v, %v", entries, err)
	}
	if err := os.Remove(s.Path); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(2, []byte(`{"theme":"graphite"}`)); err != nil {
		t.Fatal(err)
	}
	data, err := s.Load()
	if err != nil || string(data) != `{"theme":"graphite"}` {
		t.Fatalf("retry: %q, %v", data, err)
	}
}

func TestLoadRejectsNonRegularAndOversizedPreferences(t *testing.T) {
	s := &Store{Path: t.TempDir()}
	if _, err := s.Load(); err == nil {
		t.Fatal("loaded a directory as preferences")
	}
	s.Path = filepath.Join(s.Path, "large.json")
	if err := os.WriteFile(s.Path, make([]byte, 64*1024+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); err == nil {
		t.Fatal("loaded oversized preferences")
	}
}
