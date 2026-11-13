package app

import (
	"math"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

func TestSidebarMotionCanReverseWhileWriting(t *testing.T) {
	a := New()
	f := a.current()
	f.State.Buffer.Set(strings.Repeat("A quiet paragraph.\n\n", 100))
	tt := ui.NewTester(a.View, 1320, 840)
	f.State.Select(200, 210)
	tt.Frame()
	f.State.SetScrollY(200)
	tt.Frame()
	before, selection, scroll := f.State.Buffer.String(), f.State.Selection(), f.State.ScrollY()
	tt.Key(ui.Cmd, ui.KeyBackslash)
	tt.Advance(50 * time.Millisecond)
	middle, ok := tt.Find("File sidebar")
	if !ok || middle.W <= 0 || middle.W >= sidebarWidth {
		t.Fatalf("sidebar did not animate: %+v", middle)
	}
	count := len(a.files)
	_ = tt.Click("New document")
	if len(a.files) != count {
		t.Fatal("closing sidebar still took a click")
	}
	tt.Key(ui.Cmd, ui.KeyBackslash)
	reversed, _ := tt.Find("File sidebar")
	if math.Abs(float64(reversed.W-middle.W)) > 1 {
		t.Fatalf("reversing jumped: %v to %v", middle.W, reversed.W)
	}
	tt.Advance(50 * time.Millisecond)
	opening, _ := tt.Find("File sidebar")
	if opening.W <= middle.W || opening.W >= sidebarWidth {
		t.Fatalf("sidebar did not reverse smoothly: %v", opening.W)
	}
	tt.Advance(200 * time.Millisecond)
	if f.State.Buffer.String() != before || f.State.Selection() != selection || math.Abs(f.State.ScrollY()-scroll) > 0.1 {
		t.Fatal("sidebar animation changed writing state")
	}
	tt.Type("a new thought")
	if f.State.Buffer.String() == before || !f.State.Buffer.Undo() || f.State.Buffer.String() != before {
		t.Fatal("animation lost input focus or undo history")
	}
}

func TestMotionFollowsReducedMotionPreference(t *testing.T) {
	a := New()
	tt := ui.NewTester(a.View, 960, 640)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Key(ui.Cmd, ui.KeyBackslash)
	if _, ok := tt.Find("File sidebar"); ok {
		t.Fatal("reduced motion left a collapsing sidebar")
	}
	tt.Key(ui.Cmd, ui.KeyBackslash)
	if r, ok := tt.Find("File sidebar"); !ok || r.W != sidebarWidth {
		t.Fatalf("sidebar did not open immediately: %+v", r)
	}
	click(t, tt, "Outline")
	r, _ := tt.Find("Sidebar tab indicator")
	if r.X != 100 {
		t.Fatalf("reduced-motion indicator did not settle: %+v", r)
	}
}

func TestTypingContinuesDuringNarrowSidebarMotion(t *testing.T) {
	a := New()
	tt := ui.NewTester(a.View, 960, 640)
	tt.Type("开头 ")
	tt.Key(ui.Cmd, ui.KeyBackslash)
	tt.Advance(30 * time.Millisecond)
	tt.Type("中途输入")
	tt.Key(ui.Cmd, ui.KeyBackslash)
	tt.Advance(30 * time.Millisecond)
	tt.Type("继续")
	tt.Advance(200 * time.Millisecond)
	if a.current().State.Buffer.String() != "开头 中途输入继续" {
		t.Fatalf("typing during motion: %q", a.current().State.Buffer.String())
	}
	if active, _ := tt.TextInputActive(); !active {
		t.Fatal("animation stopped text input")
	}
}

func TestFileRowCloseKeepsDirtyDocuments(t *testing.T) {
	a := exampleApp()
	path := filepath.Join(t.TempDir(), "Markdown Reference.md")
	a.files[2].SetPath(path)
	tt := ui.NewTester(a.View, 960, 640)
	click(t, tt, "Close Welcome.md")
	if len(a.files) != 2 || a.current().Name != "Writing Guide.md" {
		t.Fatal("row close did not select the next document")
	}
	f := a.current()
	f.State.Buffer.Edit(editor.Edit{Text: "Unsaved paragraph\n"})
	tt.Frame()
	click(t, tt, "Close Writing Guide.md")
	if len(a.files) != 2 || a.current() != f || !f.Modified() {
		t.Fatal("row close bypassed unsaved protection")
	}
	if err := tt.RightClick("Markdown Reference.md"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Copy file path"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != path {
		t.Fatalf("copied path: %q", tt.Clipboard())
	}
	if err := tt.RightClick("Markdown Reference.md"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Close document"); err != nil {
		t.Fatal(err)
	}
	if len(a.files) != 1 || a.current() != f {
		t.Fatal("context menu closed the wrong document")
	}
}

func TestOutlineNavigationStaysInReadingMode(t *testing.T) {
	a := exampleApp()
	tt := ui.NewTester(a.View, 960, 640)
	f := a.current()
	before, selection := f.State.Buffer.String(), f.State.Selection()
	click(t, tt, "Read document")
	click(t, tt, "Outline")
	click(t, tt, "Go to Try the editor")
	first := f.Reader.AnchorVersion
	if !a.reading || f.Reader.Anchor != "try-the-editor" || first == 0 {
		t.Fatal("outline did not address the reading view")
	}
	click(t, tt, "Go to Try the editor")
	if f.Reader.AnchorVersion <= first {
		t.Fatal("choosing the same heading did not request a new jump")
	}
	if f.State.Buffer.String() != before || f.State.Selection() != selection {
		t.Fatal("reading navigation changed editing state")
	}
}

func TestSaveAcknowledgmentExpiresAndNeverHidesNewerEdits(t *testing.T) {
	a := New()
	f := a.current()
	f.SetPath(filepath.Join(t.TempDir(), "note.md"))
	tt := ui.NewTester(a.View, 960, 640)
	tt.Type("A sentence.")
	version := f.State.Buffer.Version()
	if !f.MarkSavedVersion(version) {
		t.Fatal("could not mark the simulated saved version")
	}
	a.acknowledgeSave(f)
	tt.Frame()
	if !tt.HasText("Saved just now") {
		t.Fatal("successful current save was not acknowledged")
	}
	tt.Advance(2100 * time.Millisecond)
	if tt.HasText("Saved just now") || !tt.HasText("Saved locally") {
		t.Fatal("acknowledgment did not settle quietly")
	}
	tt.Type(" A newer edit.")
	if f.MarkSavedVersion(version) {
		t.Fatal("older snapshot marked a newer edit saved")
	}
	a.acknowledgeSave(f)
	tt.Frame()
	if tt.HasText("Saved just now") || !tt.HasText("Unsaved changes") {
		t.Fatal("save feedback hid newer unsaved edits")
	}
}
