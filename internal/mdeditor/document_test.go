package mdeditor

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui/editor"
)

func TestCaptureForSavePreservesCRLF(t *testing.T) {
	f := NewFile("test.md", "# 中文\r\n\r\n原文\r\n")
	f.State.Buffer.Edit(editor.Edit{Start: f.State.Buffer.Len(), End: f.State.Buffer.Len(), Text: "新增\n"})
	got, err := f.CaptureForSave().Bytes()
	if err != nil || string(got) != "# 中文\r\n\r\n原文\r\n新增\r\n" {
		t.Fatalf("saved text %q, error %v", got, err)
	}
}

func TestCaptureForSaveProtectsMixedLineEndings(t *testing.T) {
	const original = "第一行\r\n第二行\n第三行\r\n"
	f := NewFile("test.md", original)
	got, err := f.CaptureForSave().Bytes()
	if err != nil || string(got) != original {
		t.Fatalf("unchanged text %q, error %v", got, err)
	}
	f.State.Buffer.Edit(editor.Edit{Start: 0, End: 0, Text: "修改"})
	if _, err := f.CaptureForSave().Bytes(); !errors.Is(err, ErrMixedLineEndings) {
		t.Fatalf("expected mixed-line-ending error, got %v", err)
	}
	normalized, err := f.CaptureForSave().NormalizedBytes()
	if err != nil || string(normalized) != "修改第一行\r\n第二行\r\n第三行\r\n" {
		t.Fatalf("normalized copy %q, error %v", normalized, err)
	}
	f.State.Buffer.Undo()
	got, err = f.CaptureForSave().Bytes()
	if err != nil || string(got) != original {
		t.Fatalf("undo text %q, error %v", got, err)
	}
}

func TestDisplayColumnCountsCharactersInsteadOfBytes(t *testing.T) {
	f := NewFile("test.md", "中文abc")
	f.State.Select(len("中文"), len("中文"))
	if got := DisplayColumn(f.State); got != 3 {
		t.Fatalf("column: %d", got)
	}
}

func TestSavingEarlierVersionNeverHidesNewerUnsavedText(t *testing.T) {
	f := NewFile("test.md", "original")
	f.State.Buffer.Set("written to disk")
	written := f.State.Buffer.Version()
	f.State.Buffer.Undo() // The buffer now matches the old baseline, not disk.
	if f.MarkSavedVersion(written) || !f.Modified() {
		t.Fatal("a save that finished late incorrectly cleared the dirty state")
	}
	if !f.MarkSavedVersion(f.State.Buffer.Version()) || f.Modified() {
		t.Fatal("saving the current version did not clear the dirty state")
	}
}
