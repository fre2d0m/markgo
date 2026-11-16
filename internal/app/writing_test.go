package app

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/xddcode/markgo/internal/session"
	"github.com/xddcode/markgo/internal/storage"
)

func TestVisualDocumentSaveRecoveryAndSourceRoundTrip(t *testing.T) {
	a := New()
	dir := t.TempDir()
	path := filepath.Join(dir, "draft.md")
	original := "# 标题🙂\r\n\r\n**Original** and [ref][id].\r\n\r\n\r\n\r\n[id]: <https://example.test> \"kept\"\r\n"
	f := openFixture(t, a, path, original)
	tt := ui.NewTester(a.View, 960, 640)
	w := a.writer(f)
	if !a.preview || a.reading || strings.Contains(w.Document.Text().String(), "**Original**") {
		t.Fatal("real file did not open in visual editing")
	}
	at := strings.Index(f.State.Buffer.String(), "\n\n\n\n") + 2
	f.State.Select(at, at)
	tt.Frame()
	tt.Type("/")
	tt.Type("task")
	tt.Key(0, ui.KeyEnter)
	tt.Type("记下这个想法🙂")
	expected := strings.ReplaceAll(strings.Replace(original, "\r\n\r\n\r\n\r\n", "\r\n\r\n- [ ] 记下这个想法🙂\r\n\r\n", 1), "\r\n", "\n")
	if f.State.Buffer.String() != expected {
		t.Fatalf("real-file edit: %q", f.State.Buffer.String())
	}
	for range 2 {
		click(t, tt, "View options")
		click(t, tt, "Markdown source")
		if a.preview || !tt.Focused(f.Name) {
			t.Fatal("source menu did not focus source")
		}
		click(t, tt, "View options")
		click(t, tt, "Visual editing")
		if f.State.Buffer.String() != expected {
			t.Fatal("view round trip rewrote source")
		}
	}
	checkpoint := &session.Store{Path: filepath.Join(dir, "recovery.json")}
	selection := f.State.Selection()
	saveCheckpoint(t, a, checkpoint, 1, false)
	b := New()
	if err := b.loadSession(checkpoint); err != nil {
		t.Fatal(err)
	}
	ui.NewTester(b.View, 960, 640)
	if b.current().State.Buffer.String() != expected || b.current().State.Selection() != selection || !b.preview {
		t.Fatal("visual draft recovery lost source, selection or mode")
	}
	captured := f.CaptureForSave()
	data, err := captured.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != strings.ReplaceAll(expected, "\n", "\r\n") {
		t.Fatal("visual edits changed CRLF or reference syntax")
	}
	if _, err := storage.Save(a.disk[f.ID], data); err != nil {
		t.Fatal(err)
	}
	reopened, _, err := loadDocument(path)
	if err != nil || reopened.State.Buffer.String() != expected {
		t.Fatal("saved bytes did not reopen", err)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil || string(onDisk) != string(data) {
		t.Fatal("disk differs from immutable snapshot", err)
	}
	tt.Command("undo")
	tt.Command("undo")
	if !strings.Contains(f.State.Buffer.String(), "/task") {
		t.Fatal("save and mode changes broke the single slash undo step")
	}
}

func TestVisualTabsRetainActualViewportAndModeSelection(t *testing.T) {
	a := New()
	first := a.current()
	first.State.Buffer.Set(strings.Repeat("## A section\n\nA paragraph to keep in place.\n\n", 80))
	a.newDocument()
	second := a.current()
	a.selected = first.ID
	tt := ui.NewTester(a.View, 960, 640)
	w := a.writer(first)
	tt.Scroll(650, 400, 0, 1300)
	before := w.Document.VisibleParagraphs()
	if len(before) == 0 || before[0].Start == 0 {
		t.Fatal("fixture did not scroll rich editor")
	}
	index, top := before[0].Index, before[0].Top
	click(t, tt, "Tab "+second.Name)
	tt.Type("Separate history")
	click(t, tt, "Tab "+first.Name)
	after := w.Document.VisibleParagraphs()
	if len(after) == 0 || after[0].Index != index || math.Abs(float64(after[0].Top-top)) > 0.1 {
		t.Fatalf("rich viewport reset on tab switch: %v -> %v", before, after)
	}
	click(t, tt, "View options")
	click(t, tt, "Markdown source")
	click(t, tt, "View options")
	click(t, tt, "Visual editing")
	after = w.Document.VisibleParagraphs()
	if len(after) == 0 || after[0].Index != index || math.Abs(float64(after[0].Top-top)) > 0.1 {
		t.Fatal("rich viewport reset on source round trip")
	}
	click(t, tt, "Tab "+second.Name)
	tt.Command("undo")
	if second.State.Buffer.Len() != 0 || !strings.Contains(first.State.Buffer.String(), "## A section") {
		t.Fatal("tabs shared undo")
	}
	// A clean closed tab must release both document and retained native view.
	second.State.Buffer.MarkSaved()
	a.closeDocument(second)
	tt.Frame()
	if a.writers[second.ID] != nil {
		t.Fatal("closed tab retained typesetter")
	}
}

func TestVisualCompositionRecoveryAndTabSwitch(t *testing.T) {
	a := New()
	first := a.current()
	first.State.Buffer.Set("# 标题\n\n原文")
	a.newDocument()
	second := a.current()
	a.selected = first.ID
	tt := ui.NewTester(a.View, 960, 640)
	w := a.writer(first)
	w.Document.SetSelection(w.Document.Text().Len(), w.Document.Text().Len())
	tt.Frame()
	tt.Compose("nihao", 5)
	state, err := a.captureRecovery(false).serialize()
	if err != nil {
		t.Fatal(err)
	}
	if state.Documents[0].Source != "# 标题\n\n原文" || !strings.Contains(w.Document.Text().String(), "nihao") {
		t.Fatal("recovery committed or cancelled preedit")
	}
	click(t, tt, "Tab "+second.Name)
	tt.Type("second")
	click(t, tt, "Tab "+first.Name)
	if first.State.Buffer.String() != "# 标题\n\n原文" || strings.Contains(w.Document.Text().String(), "nihao") {
		t.Fatalf("tab switch carried unfinished input: source=%q shown=%q second=%q", first.State.Buffer.String(), w.Document.Text().String(), second.State.Buffer.String())
	}
	tt.Type("你好")
	if first.State.Buffer.String() != "# 标题\n\n原文你好" {
		t.Fatal("switch lost original caret")
	}
	tt.Command("undo")
	if first.State.Buffer.String() != "# 标题\n\n原文" {
		t.Fatal("IME transition broke undo")
	}
}

func TestVisualFindRegexSelectionAndTabIsolation(t *testing.T) {
	a := New()
	f := a.current()
	f.State.Buffer.Set("# Example\n\ncat scatter Cat\n\ncat **cat**\n")
	tt := ui.NewTester(a.View, 960, 640)
	before := f.State.Buffer.String()
	start := strings.Index(before, "cat scatter")
	f.State.Select(start, start+len("cat scatter Cat"))
	tt.Frame()
	a.openFind()
	a.find.query, a.find.wholeWord, a.find.inSelection, a.find.regex = "(cat)", true, true, true
	a.find.replacement = "${1}s"
	a.refreshVisualFind(true)
	if len(a.find.matches) != 2 {
		t.Fatalf("scope and whole word: %v", a.find.matches)
	}
	a.replaceVisualMatches(true)
	if got := f.State.Buffer.String(); got != strings.Replace(before, "cat scatter Cat", "cats scatter Cats", 1) {
		t.Fatalf("regex replacement %q", got)
	}
	visualFindControls{a}.Close()
	tt.Frame()
	tt.Command("undo")
	if f.State.Buffer.String() != before {
		t.Fatal("replace-all did not undo together")
	}
	a.openFind()
	a.find.query = "["
	a.find.regex = true
	a.refreshVisualFind(true)
	if a.find.err == nil || len(a.find.matches) != 0 {
		t.Fatal("invalid pattern kept old results")
	}
	a.newDocument()
	tt.Frame()
	if a.find.open {
		t.Fatal("search remained attached to another tab")
	}
	if a.writer(f).Document.Text().String() == "" {
		t.Fatal("search cleared old document")
	}
}

func TestVisualCompositionEndsBeforeNavigation(t *testing.T) {
	for _, action := range []string{"sidebar", "source", "find", "new"} {
		t.Run(action, func(t *testing.T) {
			a := New()
			first := a.current()
			first.State.Buffer.Set("Original")
			a.newDocument()
			second := a.current()
			a.selected = first.ID
			tt := ui.NewTester(a.View, 960, 640)
			first.State.Select(8, 8)
			tt.Frame()
			tt.Compose("nihao", 5)
			switch action {
			case "sidebar":
				click(t, tt, "Document "+second.Name)
			case "source":
				click(t, tt, "View options")
				click(t, tt, "Markdown source")
			case "find":
				click(t, tt, "Find in document")
			case "new":
				click(t, tt, "New document tab")
			}
			if first.State.Buffer.String() != "Original" {
				t.Fatalf("navigation committed unfinished input: %q", first.State.Buffer.String())
			}
		})
	}
}

func TestVisualCloseLastTabStartsFreshDocument(t *testing.T) {
	a := New()
	first := a.current()
	tt := ui.NewTester(a.View, 960, 640)
	click(t, tt, "Close tab "+first.Name)
	if len(a.files) != 1 || a.current().ID == first.ID || a.writers[first.ID] != nil {
		t.Fatal("closing last clean tab retained old state")
	}
	tt.Type("A new thought")
	if a.current().State.Buffer.String() != "A new thought" {
		t.Fatal("fresh document did not take typing")
	}
}

func TestVisualFindScopeTracksEditsAndUndo(t *testing.T) {
	a := New()
	f := a.current()
	f.State.Buffer.Set("outside cat\n\ncat inside cat\n\ncat outside")
	tt := ui.NewTester(a.View, 960, 640)
	start := strings.Index(f.State.Buffer.String(), "cat inside")
	f.State.Select(start, start+len("cat inside cat"))
	tt.Frame()
	a.openFind()
	a.find.query, a.find.inSelection = "cat", true
	a.refreshVisualFind(true)
	f.State.Buffer.Edit(editor.Edit{Start: 0, End: 0, Text: "cat before\n"})
	a.refreshVisualFind(false)
	if len(a.find.matches) != 2 || a.find.matches[0][0] != start+len("cat before\n") {
		t.Fatal("selection scope did not follow preceding edit")
	}
	f.State.Buffer.Undo()
	a.refreshVisualFind(false)
	if len(a.find.matches) != 2 || a.find.matches[0][0] != start {
		t.Fatal("selection scope did not follow undo")
	}
	a.find.replacement = "longer cat"
	a.replaceVisualMatches(true)
	if len(a.find.matches) != 2 {
		t.Fatalf("selection scope did not expand with replacement: matches=%v scope=%d:%d", a.find.matches, a.find.scopeStart, a.find.scopeEnd)
	}
	f.State.Buffer.Undo()
	a.refreshVisualFind(false)
	if len(a.find.matches) != 2 || a.find.scopeStart != start || a.find.scopeEnd != start+len("cat inside cat") {
		t.Fatal("replacement undo lost scope")
	}
	// Distant changes on both sides are ambiguous in a snapshot diff. They
	// must never silently widen a scoped replace-all to the entire document.
	f.State.Buffer.Edit(editor.Edit{Start: 0, End: 0, Text: "cat\n"}, editor.Edit{Start: f.State.Buffer.Len(), End: f.State.Buffer.Len(), Text: "\ncat"})
	a.refreshVisualFind(false)
	if !a.find.scopeInvalid || len(a.find.matches) != 0 {
		t.Fatal("ambiguous edits widened search scope")
	}
	before := f.State.Buffer.String()
	a.replaceVisualMatches(true)
	if f.State.Buffer.String() != before {
		t.Fatal("invalid scope allowed replacement")
	}
}
