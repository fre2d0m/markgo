package mdeditor

import (
	"bytes"
	"errors"
	"net/url"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
)

var ErrMixedLineEndings = errors.New("this document has mixed line endings; saving an edited copy would change them")

type File struct {
	ID     string
	Name   string
	Path   string
	State  *editor.State
	Focus  ui.Handle
	Reader *markdown.Options
	syntax *writingSyntax

	original           string
	originalNormalized string
	mixedLineEndings   bool
	savedOutOfSync     bool
}

func NewFile(name, source string) *File {
	buffer := editor.NewBuffer(source)
	f := &File{
		ID:   name,
		Name: name,
		State: &editor.State{
			Buffer:         buffer,
			Font:           ui.Font{Family: "system-ui", Size: 16},
			LineHeight:     1.75,
			SoftWrap:       true,
			LineNumbers:    editor.LineNumbersOff,
			NoFolding:      true,
			NoActiveLine:   true,
			NoIndentGuides: true,
			Placeholder:    "Start writing in Markdown…",
			Label:          name,
		},
	}
	f.SetLanguage(editor.LanguageForText(name, source))
	f.State.OnInput = f.writingInput
	f.RecordSaved([]byte(source))
	f.SetPath("")
	return f
}

// SetPath creates a per-document image cache so relative images from two
// folders cannot be confused. Remote images are not fetched automatically.
func (f *File) SetPath(path string) {
	f.Path = path
	f.Reader = &markdown.Options{Theme: &markdown.Theme{FontSize: 16, LineHeight: 1.7, BlockSpacing: 0.9}}
	if path != "" {
		load := markdown.FileImages(filepath.Dir(path))
		f.Reader.LoadImage = func(location string) (ui.ImageSource, error) {
			decoded, err := url.PathUnescape(location)
			if err != nil {
				return nil, err
			}
			return load(decoded)
		}
	}
}

// RecordSaved records the bytes on disk after a successful load or save.
// The editor normalizes CRLF internally, so mixed line endings need special
// handling to avoid silently rewriting an existing document.
func (f *File) RecordSaved(data []byte) {
	f.AcceptSavedSource(PrepareSavedSource(data))
}

// SavedSource holds immutable line-ending metadata prepared on the I/O worker.
type SavedSource struct {
	original, normalized string
	mixed                bool
}

func PrepareSavedSource(data []byte) SavedSource {
	source := string(data)
	return SavedSource{
		original:   source,
		normalized: strings.ReplaceAll(source, "\r\n", "\n"),
		mixed:      strings.Contains(source, "\r\n") && strings.Contains(strings.ReplaceAll(source, "\r\n", ""), "\n"),
	}
}

// AcceptSavedSource only swaps immutable values; it does not scan the text.
func (f *File) AcceptSavedSource(source SavedSource) {
	f.original, f.originalNormalized, f.mixedLineEndings = source.original, source.normalized, source.mixed
}

func (f *File) HasMixedLineEndings() bool { return f.mixedLineEndings }

// RecoverFile opens a checkpoint as an unsaved document without manufacturing
// an undo transaction that could erase the entire recovered draft.
func RecoverFile(name, source, original string) *File {
	f := NewFile(name, source)
	f.RecordSaved([]byte(original))
	f.savedOutOfSync = true
	return f
}

// Modified remains conservative when an earlier snapshot finished saving
// after the buffer changed. Buffer.MarkSaved can only mark its current text.
func (f *File) Modified() bool { return f.savedOutOfSync || f.State.Buffer.Modified() }

// MarkSavedVersion marks the current buffer saved only if it is still the
// snapshot that was written. Otherwise it must remain visibly unsaved.
func (f *File) MarkSavedVersion(version uint64) bool {
	if f.State.Buffer.Version() != version {
		f.savedOutOfSync = true
		return false
	}
	f.State.Buffer.MarkSaved()
	f.savedOutOfSync = false
	return true
}

// DisplayColumn converts the editor's UTF-8 byte column to a character count.
func DisplayColumn(st *editor.State) int {
	position := st.CursorPos()
	line := st.Buffer.Line(position.Line)
	column := min(max(position.Col, 0), len(line))
	return utf8.RuneCountInString(line[:column]) + 1
}

// SaveSnapshot is immutable and can be serialized on a background goroutine.
type SaveSnapshot struct {
	Text               editor.Snapshot
	original           string
	originalNormalized string
	mixedLineEndings   bool
}

// OriginalSource is immutable and preserves the exact disk bytes, including
// mixed line endings. Recovery uses it to recognize external changes.
func (s SaveSnapshot) OriginalSource() string { return s.original }

func (f *File) CaptureForSave() SaveSnapshot {
	return SaveSnapshot{
		Text:               f.State.Buffer.Snapshot(),
		original:           f.original,
		originalNormalized: f.originalNormalized,
		mixedLineEndings:   f.mixedLineEndings,
	}
}

// Bytes writes the buffer using its original line-ending convention. For a
// mixed-ending file, only the unchanged source can be reproduced safely.
func (s SaveSnapshot) Bytes() ([]byte, error) {
	if s.mixedLineEndings {
		if s.Text.String() != s.originalNormalized {
			return nil, ErrMixedLineEndings
		}
		return []byte(s.original), nil
	}
	return s.NormalizedBytes()
}

// NormalizedBytes is used only after the user explicitly chooses to export a
// mixed-ending document as a new copy with a single line-ending convention.
func (s SaveSnapshot) NormalizedBytes() ([]byte, error) {
	var out bytes.Buffer
	if _, err := s.Text.WriteTo(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func SelectedText(st *editor.State) string {
	for _, selection := range st.Selections() {
		if !selection.Empty() {
			return st.Buffer.Slice(selection.Start(), selection.End())
		}
	}
	return ""
}
