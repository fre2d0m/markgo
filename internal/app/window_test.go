package app

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

type testWindow struct {
	minimized, toggled, closed int
	onToggle                   func()
}

func (w *testWindow) Minimize()       { w.minimized++ }
func (w *testWindow) ToggleMaximize() { w.toggled++; w.onToggle() }
func (w *testWindow) Close()          { w.closed++ }

// These checks exercise dispatch and layout. Real window dragging, resizing,
// Snap layouts and the OS close dialog still need native platform acceptance.
func TestCaptionControlsKeepEditingState(t *testing.T) {
	a := New()
	f := a.current()
	f.Name = strings.Repeat("一篇很长的文档标题", 20) + ".md"
	w := &testWindow{onToggle: func() { a.windowMaximized = !a.windowMaximized }}
	a.windowCommands = w
	tt := ui.NewTester(a.View, 960, 640)
	tt.Type("A sentence to keep.")
	f.State.Select(2, 10)
	tt.Frame()
	before, selection := f.State.Buffer.String(), f.State.Selection()
	for _, label := range []string{"Minimize window", "Maximize window", "Restore window", "Close window"} {
		r, ok := tt.Find(label)
		if !ok || r.W < 40 || r.H != titleBarHeight || r.Y != 0 || r.X < 0 || r.X+r.W > 960 {
			t.Fatalf("caption %s: %+v, exists %v", label, r, ok)
		}
		click(t, tt, label)
		if f.State.Buffer.String() != before || f.State.Selection() != selection || a.current() != f {
			t.Fatalf("%s changed the document or its selection", label)
		}
	}
	if w.minimized != 1 || w.toggled != 2 || w.closed != 1 {
		t.Fatalf("caption actions were dropped or duplicated: %+v", w)
	}
	tt.Type("replacement")
	if f.State.Buffer.String() == before || !f.State.Buffer.Undo() || f.State.Buffer.String() != before {
		t.Fatal("window controls lost the writing focus or undo history")
	}
}

func TestCaptionControlsRemainReachableInWritingModes(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			a := New()
			tt := ui.NewTester(a.View, 960, 640)
			tt.SetScale(scale)
			for _, action := range []string{"Toggle file sidebar", "Focus mode", "Exit focus mode", "Toggle file sidebar", "Read document", "Edit document"} {
				click(t, tt, action)
				lastRight := float32(0)
				for _, label := range []string{"Minimize window", "Maximize window", "Close window"} {
					r, ok := tt.Find(label)
					if !ok || r.X < lastRight || r.X+r.W > 960 || r.Y+r.H > titleBarHeight {
						t.Fatalf("after %s, %s overlaps or leaves the title bar: %+v", action, label, r)
					}
					lastRight = r.X + r.W
				}
			}
		})
	}
}

func TestPanelsCanBeInterruptedWithoutLosingWriting(t *testing.T) {
	for _, reduced := range []bool{false, true} {
		t.Run(fmt.Sprint(reduced), func(t *testing.T) {
			a := New()
			tt := ui.NewTester(a.View, 960, 640)
			tt.SetPreferences(ui.Preferences{ReduceMotion: reduced})
			tt.Type("Keep the words.")
			f := a.current()
			f.State.Select(5, 8)
			tt.Frame()
			before, selection := f.State.Buffer.String(), f.State.Selection()
			for i := 0; i < 3; i++ {
				click(t, tt, "Appearance")
				tt.Advance(30 * time.Millisecond)
				click(t, tt, "Formatting")
				tt.Advance(30 * time.Millisecond)
				tt.Key(0, ui.KeyEscape)
				tt.Advance(200 * time.Millisecond)
				if a.appearanceOpen || a.formatOpen || f.State.Buffer.String() != before || f.State.Selection() != selection {
					t.Fatal("interrupted panels left an overlay open or changed the selection")
				}
			}
			tt.Type("these")
			if f.State.Buffer.String() != "Keep these words." {
				t.Fatalf("typing after panels: %q", f.State.Buffer.String())
			}
		})
	}
}

func TestSidebarKeyboardAndBottomAction(t *testing.T) {
	a := exampleApp()
	tt := ui.NewTester(a.View, 960, 640)
	click(t, tt, "Documents")
	tt.Key(0, ui.KeyDown)
	if a.current().Name != "Writing Guide.md" {
		t.Fatalf("arrow selection: %s", a.current().Name)
	}
	tt.Key(0, ui.KeyEnd)
	if a.current().Name != "Markdown Reference.md" {
		t.Fatalf("last document: %s", a.current().Name)
	}
	if !tt.Focused("Documents") {
		t.Fatal("changing documents stole list focus")
	}
	before := a.current().State.Buffer.String()
	tt.Key(0, ui.KeyEnter)
	tt.Type("Start writing ")
	if a.current().State.Buffer.String() == before {
		t.Fatal("Enter did not move from files to writing")
	}
	r, ok := tt.Find("Open folder")
	if !ok || r.Y < 550 || r.Y+r.H > 610 {
		t.Fatalf("folder action is not at the sidebar bottom: %+v", r)
	}
}
