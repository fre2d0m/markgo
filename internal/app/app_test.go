package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/xddcode/markgo/internal/mdeditor"
	"github.com/xddcode/markgo/internal/storage"
)

func TestWorkspace(t *testing.T) {
	a := exampleApp()
	tt := ui.NewTester(a.View, 1320, 840)
	if !tt.HasText("MarkGo") || !tt.HasText("Welcome.md") {
		t.Fatalf("initial workspace texts %q", tt.Texts())
	}
	if err := tt.Click("Markdown Reference.md"); err != nil {
		t.Fatal(err)
	}
	if got := a.current().Name; got != "Markdown Reference.md" {
		t.Fatalf("selected document %q", got)
	}
	if err := tt.Click("Document menu"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Show Markdown source"); err != nil {
		t.Fatal(err)
	}
	if a.preview {
		t.Fatal("live preview stayed enabled")
	}
	if err := tt.Click("Filter documents"); err != nil {
		t.Fatal(err)
	}
	tt.Type("Guide")
	if files := a.visibleFiles(); len(files) != 1 || files[0].Name != "Writing Guide.md" {
		t.Fatalf("filtered files %v", files)
	}
}

func TestSelectedText(t *testing.T) {
	f := mdeditor.NewFile("note.md", "hello **world**")
	f.State.Select(8, 13)
	if got := mdeditor.SelectedText(f.State); got != "world" {
		t.Fatalf("selected text %q", got)
	}
}

func TestOutlineNavigatesToHeading(t *testing.T) {
	a := exampleApp()
	tt := ui.NewTester(a.View, 1320, 840)
	if err := tt.Click("Outline"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("A calm place to write") {
		t.Fatalf("outline headings: %q", tt.Texts())
	}
	if err := tt.Click("Go to Try the editor"); err != nil {
		t.Fatal(err)
	}
	pos := a.current().State.CursorPos()
	if got := a.current().State.Buffer.Line(pos.Line); got != "## Try the editor" {
		t.Fatalf("navigation landed on line %d: %q", pos.Line, got)
	}
}

func TestOpeningSameDocumentDoesNotDuplicateOrDiscardEdits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.md")
	if err := os.WriteFile(path, []byte("# Existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, snapshot, err := loadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	a := exampleApp()
	a.addOpened(file, snapshot)
	file.State.Buffer.Edit(editor.Edit{Start: 0, End: 0, Text: "New text\n"})
	count := len(a.files)
	duplicate, sameSnapshot, err := loadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	a.addOpened(duplicate, sameSnapshot)
	if len(a.files) != count || a.current() != file || !file.State.Buffer.Modified() {
		t.Fatalf("reopening lost or duplicated a document: %d files", len(a.files))
	}
	if !a.hasUnsavedChanges() {
		t.Fatal("edited document was not treated as unsaved")
	}
}

func TestFormattingToolbarAndShortcut(t *testing.T) {
	a := exampleApp()
	tt := ui.NewTester(a.View, 1320, 840)
	file := a.current()
	before := file.State.Buffer.String()
	start := strings.Index(before, "focused writing")
	file.State.Select(start, start+len("focused writing"))
	tt.Frame()
	if err := tt.Click("Bold"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(file.State.Buffer.String(), "**focused writing**") {
		t.Fatalf("toolbar did not format selection: %q", file.State.Buffer.String())
	}
	if !file.State.Buffer.Undo() || file.State.Buffer.String() != before {
		t.Fatal("toolbar formatting was not one undoable edit")
	}
	file.State.Select(start, start+len("focused writing"))
	tt.Frame()
	click(t, tt, "Formatting")
	click(t, tt, "Quote")
	if a.formatOpen || !strings.Contains(file.State.Buffer.String(), "> ") || !file.State.Buffer.Undo() || file.State.Buffer.String() != before {
		t.Fatal("paragraph menu did not close after one undoable action")
	}
	file.State.Select(start, start+len("focused writing"))
	file.Focus.Focus()
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyI)
	if !strings.Contains(file.State.Buffer.String(), "*focused writing*") {
		t.Fatalf("shortcut did not format selection: %q", file.State.Buffer.String())
	}
}

func TestFolderTreeExpandsAndFilters(t *testing.T) {
	a := exampleApp()
	a.folder = &storage.Entry{Name: "Workspace", Directory: true, Children: []storage.Entry{
		{Name: "notes", Path: "/workspace/notes", Directory: true, Children: []storage.Entry{{Name: "nested.md", Path: "/workspace/notes/nested.md"}}},
		{Name: "other.md", Path: "/workspace/other.md"},
	}}
	tt := ui.NewTester(a.View, 1320, 840)
	if err := tt.Click("notes"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("nested.md") {
		t.Fatalf("expanded tree: %q", tt.Texts())
	}
	if err := tt.Click("Filter documents"); err != nil {
		t.Fatal(err)
	}
	tt.Type("nested")
	if !tt.HasText("nested.md") || tt.HasText("other.md") {
		t.Fatalf("filtered tree: %q", tt.Texts())
	}
}

func TestCloseDocumentKeepsDirtyTextWithoutConfirmation(t *testing.T) {
	a := exampleApp()
	first := a.current()
	a.closeDocument(first)
	if len(a.files) != 2 || a.current() == first {
		t.Fatal("unmodified document was not closed")
	}
	dirty := a.current()
	dirty.State.Buffer.Edit(editor.Edit{Start: 0, End: 0, Text: "unsaved"})
	a.closeDocument(dirty) // No window means no confirmation can be shown.
	if len(a.files) != 2 || a.current() != dirty || !dirty.State.Buffer.Modified() {
		t.Fatal("unsaved document was closed without confirmation")
	}
}

func TestReadModeKeepsMarkdownSourceUntouched(t *testing.T) {
	a := exampleApp()
	tt := ui.NewTester(a.View, 1320, 840)
	before := a.current().State.Buffer.String()
	if err := tt.Click("Read document"); err != nil {
		t.Fatal(err)
	}
	if !a.reading || !tt.HasText("Welcome to MarkGo") {
		t.Fatalf("read mode: %q", tt.Texts())
	}
	if err := tt.Click("Edit document"); err != nil {
		t.Fatal(err)
	}
	if a.reading || a.current().State.Buffer.String() != before {
		t.Fatal("switching reading mode changed the document")
	}
}

func TestLivePreviewTaskClickEditsSourceAndCanUndo(t *testing.T) {
	a := exampleApp()
	tt := ui.NewTester(a.View, 1320, 840)
	file := a.current()
	before := file.State.Buffer.String()
	if err := tt.Click("Not done"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(file.State.Buffer.String(), "- [x] Write your first note") {
		t.Fatalf("task source was not updated: %q", file.State.Buffer.String())
	}
	if !file.State.Buffer.Undo() || file.State.Buffer.String() != before {
		t.Fatal("task change was not undoable")
	}
}

func TestFocusModeRestoresWorkspace(t *testing.T) {
	a := exampleApp()
	tt := ui.NewTester(a.View, 1320, 840)
	if err := tt.Click("Focus mode"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(200 * time.Millisecond)
	if !a.focusMode || tt.HasText("Open documents") {
		t.Fatalf("focus mode: %q", tt.Texts())
	}
	if err := tt.Click("Exit focus mode"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(200 * time.Millisecond)
	if a.focusMode || !tt.HasText("Open documents") {
		t.Fatalf("workspace did not return: %q", tt.Texts())
	}
}

func TestReloadKeepsNewerEditsAndPreservesDiskBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reload.md")
	if err := os.WriteFile(path, []byte("original\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, snapshot, err := loadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	a := exampleApp()
	a.addOpened(file, snapshot)
	version := file.State.Buffer.Version()
	updated := []byte("first\r\nsecond\nthird\r\n")
	if err := os.WriteFile(path, updated, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, external, err := loadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	file.State.Buffer.Set("newer edit")
	if a.applyReload(file.ID, version, loaded, external) || file.State.Buffer.String() != "newer edit" {
		t.Fatal("reload replaced an edit made while the file was being read")
	}
	if !a.applyReload(file.ID, file.State.Buffer.Version(), loaded, external) {
		t.Fatal("reload did not apply to the unchanged version")
	}
	got, err := a.current().CaptureForSave().Bytes()
	if err != nil || string(got) != string(updated) {
		t.Fatalf("reloaded bytes %q, error %v", got, err)
	}
}

func TestFindReplaceAndUndoInLivePreview(t *testing.T) {
	a := exampleApp()
	file := mdeditor.NewFile("search.md", "# 中文标题\n\n你好 world，你好 **world**。\n")
	a.files, a.selected = []*mdeditor.File{file}, file.ID
	tt := ui.NewTester(a.View, 960, 640)
	before := file.State.Buffer.String()
	if err := tt.Click("Find in document"); err != nil {
		t.Fatal(err)
	}
	tt.Type("你好")
	if count, _ := file.State.FindMatches(); count != 2 {
		t.Fatalf("Chinese find: %d matches; texts %q", count, tt.Texts())
	}
	if err := tt.Click("Next match"); err != nil {
		t.Fatal(err)
	}
	if got := mdeditor.SelectedText(file.State); got != "你好" {
		t.Fatalf("selected match %q", got)
	}
	if err := tt.Click("Show replace"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Replace text"); err != nil {
		t.Fatal(err)
	}
	tt.Type("再见")
	if err := tt.Click("Replace all"); err != nil {
		t.Fatal(err)
	}
	if got := file.State.Buffer.String(); got != strings.ReplaceAll(before, "你好", "再见") {
		t.Fatalf("replaced source %q", got)
	}
	if err := tt.Click("Close find"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	if file.State.Buffer.String() != before {
		t.Fatal("replace-all was not one undoable edit")
	}
	if !tt.Focused("search.md") {
		t.Fatal("closing find did not restore editor focus")
	}
}

func TestCompositionWithLivePreviewPreservesSourceAndUndo(t *testing.T) {
	a := exampleApp()
	file := mdeditor.NewFile("composition.md", "**标题**\n\n- ")
	a.files, a.selected = []*mdeditor.File{file}, file.ID
	tt := ui.NewTester(a.View, 960, 640)
	before := file.State.Buffer.String()
	file.State.Select(len(before), len(before))
	tt.Frame()
	if active, disabled := tt.TextInputActive(); !active || disabled {
		t.Fatal("input methods are not enabled")
	}
	tt.Compose("ni", 2)
	tt.Compose("你好", 2)
	tt.Type("你好")
	if got := file.State.Buffer.String(); got != before+"你好" {
		t.Fatalf("composition committed unexpected text: %q", got)
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	if got := file.State.Buffer.String(); got != before {
		t.Fatalf("composition undo damaged Markdown source: %q", got)
	}
}

func TestFolderFilterRevealsCollapsedMatchesWithoutChangingExpansion(t *testing.T) {
	a := exampleApp()
	a.folder = &storage.Entry{Name: "Workspace", Directory: true, Children: []storage.Entry{
		{Name: "notes", Path: "/workspace/notes", Directory: true, Children: []storage.Entry{{Name: "nested.md", Path: "/workspace/notes/nested.md"}}},
	}}
	tt := ui.NewTester(a.View, 960, 640)
	if tt.HasText("nested.md") {
		t.Fatal("folder should start collapsed")
	}
	if err := tt.Click("Filter documents"); err != nil {
		t.Fatal(err)
	}
	tt.Type("nested")
	if !tt.HasText("nested.md") {
		t.Fatal("filter hid a match in a collapsed folder")
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	if tt.HasText("nested.md") || *a.expanded["/workspace/notes"] {
		t.Fatal("clearing filter changed the previous expansion state")
	}
}

func TestNativeOpenRequestsAreRetainedBeforeWindowAndWhileBusy(t *testing.T) {
	a := exampleApp()
	dir := t.TempDir()
	first, second := filepath.Join(dir, "first.md"), filepath.Join(dir, "second.md")
	a.OpenFile(first)
	a.OpenFile(first)
	a.OpenFile(second)
	a.processOpenQueue()
	if len(a.openQueue) != 2 || a.openQueue[0] != first || a.openQueue[1] != second {
		t.Fatalf("startup file requests were lost or duplicated: %v", a.openQueue)
	}
	a.fileBusy = true
	a.processOpenQueue()
	if len(a.openQueue) != 2 {
		t.Fatal("busy workbench discarded pending files")
	}
}

func exampleApp() *App {
	a := New()
	a.files = sampleDocuments()
	a.selected = a.files[0].ID
	return a
}
