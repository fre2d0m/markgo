package app

import (
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/xddcode/markgo/internal/mdeditor"
)

func TestDocumentTabsPreserveIndependentWritingAndCloseGuards(t *testing.T) {
	a := New()
	first := a.current()
	first.State.Buffer.Set(strings.Repeat("A quiet paragraph.\n\n", 80))
	a.newDocument()
	second := a.current()
	second.State.Buffer.Edit(editor.Edit{Text: "Another thought"})
	a.newDocument()
	third := a.current()
	a.selected = first.ID
	tt := ui.NewTester(a.View, 1320, 840)
	first.State.Select(200, 210)
	tt.Frame()
	first.State.SetScrollY(200)
	tt.Frame()
	source, selection, scroll := first.State.Buffer.String(), first.State.Selection(), first.State.ScrollY()
	click(t, tt, "Tab "+second.Name)
	if a.current() != second {
		t.Fatal("tab did not select its document")
	}
	tt.Command("undo")
	if second.State.Buffer.String() != "" || first.State.Buffer.String() != source {
		t.Fatal("tabs shared undo history")
	}
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyTab)
	if a.current() != first || first.State.Selection() != selection || math.Abs(first.State.ScrollY()-scroll) > 0.1 {
		t.Fatal("switching tabs lost the writing position")
	}
	tt.Type("replacement")
	tt.Command("undo")
	if first.State.Buffer.String() != source {
		t.Fatal("tab switch lost the writing focus")
	}
	click(t, tt, "Close tab "+third.Name)
	if len(a.files) != 2 || a.current() != first {
		t.Fatal("closing an inactive tab changed the active document")
	}
	click(t, tt, "Close tab "+first.Name)
	if len(a.files) != 2 || a.current() != first {
		t.Fatal("tab close bypassed unsaved protection")
	}
	click(t, tt, "New document tab")
	if len(a.files) != 3 || a.current() == first {
		t.Fatal("new tab did not create a fresh document")
	}
	tt.Type("New page")
	if a.current().State.Buffer.String() != "New page" {
		t.Fatal("new tab did not receive typing")
	}
}

func TestDocumentTabOverflowRemainsReachable(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		a := tabsExample(24)
		tt := ui.NewTester(a.View, 960, 640)
		tt.SetScale(scale)
		tt.Key(ui.Ctrl|ui.Shift, ui.KeyTab)
		last := a.current()
		if last != a.files[23] {
			t.Fatal("reverse cycling did not wrap")
		}
		assertTabVisible(t, tt, last.Name)
		tt.SetSize(1320, 840)
		tt.SetSize(960, 640)
		assertTabVisible(t, tt, last.Name)
		for _, label := range []string{"New document tab", "All open documents", "Appearance", "Minimize window", "Close window"} {
			r, ok := tt.Find(label)
			if !ok || r.W < 40 || r.H < 40 || r.X < 0 || r.X+r.W > 960 {
				t.Fatalf("scale %v: %s unavailable: %+v", scale, label, r)
			}
		}
		click(t, tt, "All open documents")
		tt.Key(0, ui.KeyEnd)
		tt.Key(0, ui.KeyEnter)
		if a.current() != last {
			t.Fatal("overflow menu keyboard could not reach the last document")
		}
		click(t, tt, "All open documents")
		if err := tt.ChooseMenuItem(a.files[0].Name); err != nil {
			t.Fatal(err)
		}
		if a.current() != a.files[0] {
			t.Fatal("overflow menu chose the wrong file")
		}
		assertTabVisible(t, tt, a.current().Name)
		tt.SetSize(1320, 840)
		tt.SetSize(960, 640)
		click(t, tt, "Close tab "+a.current().Name)
		if len(a.files) != 23 || a.current() != a.files[0] {
			t.Fatal("active close did not choose the adjacent document")
		}
		assertTabVisible(t, tt, a.current().Name)
	}
}

func assertTabVisible(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	tt.Frame()
	tab, ok := tt.Find("Tab " + name)
	strip, _ := tt.Find("Document tab strip")
	if !ok || tab.X < strip.X-1 || tab.X+tab.W > strip.X+strip.W+1 {
		t.Fatalf("active tab is outside the strip: %+v in %+v", tab, strip)
	}
}

func TestPreviewTabsKeepClosedSampleEdits(t *testing.T) {
	p := NewTypesettingPreview()
	p.selected = 3
	tt := ui.NewTester(p.View, 1320, 840)
	tt.Type("A new page")
	page := p.current()
	click(t, tt, "New document tab")
	if p.current() == page || p.current().file.State.Buffer.String() != "" {
		t.Fatal("new preview tab reused a buffer")
	}
	click(t, tt, "Close tab "+p.current().file.Name)
	if p.current() != page {
		t.Fatal("closing the new tab lost the previous page")
	}
	click(t, tt, "Close tab "+page.file.Name)
	click(t, tt, "Open sample Blank page")
	if p.current() != page || page.file.State.Buffer.String() != "A new page" {
		t.Fatal("closing a sample tab discarded its edits")
	}
	tt.Command("undo")
	if page.file.State.Buffer.String() != "" {
		t.Fatal("reopening a sample lost undo history")
	}
	if p.shell.sessionStore != nil || p.shell.preferenceStore != nil {
		t.Fatal("sample tabs attached persistent storage")
	}
}

func TestDocumentTabPointerCancellationAndNoHover(t *testing.T) {
	a := tabsExample(3)
	tt := ui.NewTester(a.View, 1320, 840)
	first := a.current()
	first.State.Buffer.Edit(editor.Edit{Text: "Unsaved "})
	tt.Frame()
	r, _ := tt.Find("Tab " + a.files[1].Name)
	tt.Press(r.X+30, r.Y+20)
	tt.Move(900, 700)
	tt.Release(900, 700)
	if a.current() != first {
		t.Fatal("dragging out switched documents")
	}
	tt.SetPreferences(ui.Preferences{NoHover: true})
	if _, ok := tt.Find("Unsaved tab " + first.Name); !ok {
		t.Fatal("no-hover mode hid the dirty state")
	}
	click(t, tt, "Close tab "+first.Name)
	if len(a.files) != 3 || a.current() != first {
		t.Fatal("no-hover close bypassed unsaved protection")
	}
}

func TestDocumentTabKeyboardAndDuplicateNames(t *testing.T) {
	a := tabsExample(3)
	tt := ui.NewTester(a.View, 1320, 840)
	// Tab indents Markdown inside the editor. Start from the search field
	// to exercise ordinary focus traversal without changing the document.
	click(t, tt, "Filter documents")
	for i := 0; i < 40 && !tt.Focused("Tab "+a.files[0].Name); i++ {
		tt.Key(ui.Shift, ui.KeyTab)
	}
	if !tt.Focused("Tab " + a.files[0].Name) {
		t.Fatal("Tab could not reach the document strip")
	}
	tt.Key(0, ui.KeyRight)
	if a.current() != a.files[1] {
		t.Fatal("arrow navigation did not select the next tab")
	}
	tt.Key(0, ui.KeyRight)
	if a.current() != a.files[2] {
		t.Fatal("arrow navigation did not retain the tab keyboard focus")
	}
	tt.Key(0, ui.KeyLeft)
	before := a.current().State.Buffer.String()
	tt.Key(0, ui.KeyEnter)
	tt.Type("A thought. ")
	if a.current().State.Buffer.String() == before || a.files[0].State.Buffer.String() != before {
		t.Fatal("Enter did not return typing to the selected tab")
	}
	for i, folder := range []string{"notes", "journal"} {
		a.files[i].Name = "Today.md"
		a.files[i].Path = filepath.Join(string(filepath.Separator), folder, "Today.md")
	}
	tt.Frame()
	click(t, tt, "All open documents")
	label := "Today.md — " + filepath.Dir(a.files[0].Path)
	if err := tt.ChooseMenuItem(label); err != nil {
		t.Fatal(err)
	}
	if a.current() != a.files[0] {
		t.Fatal("duplicate filenames chose the wrong document")
	}
}

func tabsExample(count int) *App {
	a := New()
	a.files = nil
	names := []string{"慢一点，写下来.md", "设计规范与一些未完成的想法.md", "A quieter morning.md"}
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("Notebook %02d.md", i+1)
		if i < len(names) {
			name = names[i]
		}
		f := mdeditor.NewFile(name, quietChinese)
		f.ID = fmt.Sprintf("tab-example-%d", i)
		a.files = append(a.files, f)
	}
	a.selected = a.files[0].ID
	return a
}

func TestWorkbenchTabsScreenshots(t *testing.T) {
	dir := os.Getenv("MARKGO_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set MARKGO_SCREENSHOT_DIR for workbench evidence")
	}
	dir = filepath.Join(dir, "workbench")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, palette := range palettes {
		for _, scene := range []string{"writing", "minimum", "many", "reading"} {
			a := tabsExample(3)
			a.appearance.Theme = palette.ID
			w, h := 1320, 840
			if scene == "minimum" {
				w, h = 960, 640
				a.selected = a.files[2].ID
			}
			if scene == "many" {
				a = tabsExample(24)
				a.appearance.Theme = palette.ID
				w, h = 960, 640
			}
			if scene == "reading" {
				a.reading = true
				a.sideView = 1
			}
			tt := ui.NewTester(a.View, w, h)
			if scene == "many" {
				click(t, tt, "All open documents")
			}
			tt.Move(0, float32(h-1))
			tt.Advance(200 * time.Millisecond)
			out, err := os.Create(filepath.Join(dir, palette.ID+"-"+scene+".png"))
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(out, tt.Image())
			_ = out.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}
