package storage

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenSavePreservesBytesAndRejectsExternalChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "文稿.md")
	original := []byte("\ufeff# 标题\r\n\r\n| 列 | 值 |\r\n| --- | --- |\r\n| 甲 | **乙** |\r\n\r\n```go\r\nprint(1)\r\n```\r\n")
	if err := os.WriteFile(path, original, 0o640); err != nil {
		t.Fatal(err)
	}
	snapshot, got, err := Open(path)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatalf("open: %q, %v", got, err)
	}
	updated := append(bytes.Clone(original), []byte("\r\n新的内容\r\n")...)
	snapshot, err = Save(snapshot, updated)
	if err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(got, updated) {
		t.Fatalf("saved: %q, %v", got, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o640 {
			t.Fatalf("mode: %v, %v", info, err)
		}
	}
	external := []byte("external edit")
	if err := os.WriteFile(path, external, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := Check(snapshot); !errors.Is(err, ErrExternalChange) {
		t.Fatalf("expected external-change check, got %v", err)
	}
	if _, err := Save(snapshot, []byte("must not replace")); !errors.Is(err, ErrExternalChange) {
		t.Fatalf("expected external-change error, got %v", err)
	}
	got, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(got, external) {
		t.Fatalf("external content was changed: %q, %v", got, err)
	}
}

func TestCreateNeverOverwritesAnExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.md")
	if _, err := Create(path, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(path, []byte("second")); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected existing-file error, got %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "first" {
		t.Fatalf("existing content changed: %q, %v", got, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("unexpected temporary files: %v, %v", entries, err)
	}
}

func TestRejectInvalidUTF8AndSymbolicLinks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "document.md")
	if err := os.WriteFile(path, []byte{0xff}, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Open(path); !errors.Is(err, ErrInvalidUTF8) {
		t.Fatalf("expected UTF-8 error, got %v", err)
	}
	link := filepath.Join(dir, "link.md")
	if err := os.Symlink(path, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if _, _, err := Open(link); !errors.Is(err, ErrNotRegular) {
		t.Fatalf("expected regular-file error, got %v", err)
	}
}

func TestSaveDoesNotBypassReadOnlyPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits")
	}
	path := filepath.Join(t.TempDir(), "readonly.md")
	if err := os.WriteFile(path, []byte("original"), 0o400); err != nil {
		t.Fatal(err)
	}
	snapshot, _, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Save(snapshot, []byte("changed")); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("read-only save returned %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "original" {
		t.Fatalf("read-only source changed: %q, %v", got, err)
	}
}
