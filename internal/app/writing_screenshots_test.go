package app

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

func TestWritingCommandGeometryAndCancellation(t *testing.T) {
	for _, bottom := range []bool{false, true} {
		for _, scale := range []float32{1, 1.5, 2} {
			a := New()
			if bottom {
				a.current().State.Buffer.Set(strings.Repeat("A quiet line.\n\n", 40))
			}
			var editorBounds ui.Rect
			tt := ui.NewTester(func(c *ui.Context) { a.View(c); editorBounds = a.current().Focus.Bounds(c) }, 960, 640)
			tt.SetScale(scale)
			w := a.writer(a.current())
			w.Document.SetSelection(w.Document.Text().Len(), w.Document.Text().Len())
			tt.Frame()
			tt.Type("/")
			tt.Advance(200 * time.Millisecond)
			r, ok := tt.Find("Insert block menu")
			if !ok || r.W < 240 || r.X < 0 || r.Y < 0 || r.X+r.W > 960 || r.Y+r.H > 640 {
				t.Fatalf("bottom=%v scale=%v menu: %+v", bottom, scale, r)
			}
			caret, _ := w.Document.CaretRect()
			caretTop := editorBounds.Y + caret.Y
			if r.Y < caretTop+caret.H && r.Y+r.H > caretTop {
				t.Fatalf("menu covers writing line: %+v caret y=%v h=%v", r, caretTop, caret.H)
			}
			// Up wraps to the last item and reveals it in the menu's scroll area.
			tt.Key(0, ui.KeyUp)
			tt.Frame()
			item, ok := tt.Find("Insert Code block")
			if !ok || item.H < 40 || item.Y < r.Y || item.Y+item.H > r.Y+r.H {
				t.Fatalf("last command clipped: %+v in %+v", item, r)
			}
			before := a.current().State.Buffer.String()
			tt.Press(item.X+20, item.Y+20)
			tt.Move(5, 600)
			tt.Release(5, 600)
			if a.current().State.Buffer.String() != before {
				t.Fatal("dragging off a command applied it")
			}
			tt.Key(0, ui.KeyEscape)
			if _, ok := tt.Find("Insert block menu"); ok {
				t.Fatal("Escape did not dismiss menu")
			}
		}
	}
}

func TestVisualFindNeverCoversTheMatchedLine(t *testing.T) {
	a := New()
	f := a.current()
	f.State.Buffer.Set("# Heading\n\nFind this thought in the first paragraph.\n\nAnother paragraph.")
	var bounds ui.Rect
	tt := ui.NewTester(func(c *ui.Context) { a.View(c); bounds = f.Focus.Bounds(c) }, 960, 640)
	click(t, tt, "Find in document")
	tt.Type("thought")
	for _, replacing := range []bool{false, true} {
		if replacing {
			click(t, tt, "Show replace")
		}
		tt.Advance(200 * time.Millisecond)
		panel, ok := tt.Find("Find in document panel")
		caret, visible := a.writer(f).Document.CaretRect()
		if !ok || !visible || panel.Y+panel.H > bounds.Y+caret.Y || bounds.H <= 0 {
			t.Fatalf("search covers result: panel=%+v editor=%+v caret=%+v", panel, bounds, caret)
		}
	}
	tt.Key(0, ui.KeyEscape)
	if a.find.open || !tt.Focused(f.Name) {
		t.Fatal("Escape did not return to the writing selection")
	}
}

// The images are actual App views with real editing state, not mockups.
func TestVisualWritingScreenshots(t *testing.T) {
	dir := os.Getenv("MARKGO_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set MARKGO_SCREENSHOT_DIR for rendered evidence")
	}
	dir = filepath.Join(dir, "visual-writing")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"paper", "sakura", "graphite"} {
		for _, scene := range []string{"writing", "blank", "slash", "filtered", "no-results", "bottom", "source", "view-menu", "find-error", "find-replace", "find-scope", "code", "link"} {
			a := New()
			a.appearance.Theme = theme
			f := a.current()
			f.Name = "A quieter morning.md"
			f.State.Buffer.Set("# A quieter morning\n\nMake room for **a thought worth keeping**. Write it down, then find the words that feel right.\n\n## A small beginning\n\n- [x] Open a fresh page\n- [ ] Follow one idea\n\n> Good writing begins with paying attention.\n\n")
			if scene == "blank" || scene == "slash" || scene == "filtered" || scene == "no-results" {
				f.State.Buffer.Set("")
			}
			if scene == "bottom" {
				f.State.Buffer.Set(strings.Repeat("A paragraph to keep in place.\n\n", 40))
			}
			if scene == "code" {
				f.State.Buffer.Set("# A small experiment\n\n```go\nfunc main() {\n    fmt.Println(\"慢一点，写下来。\")\n}\n```\n\n")
			}
			a.newDocument()
			a.current().Name = "设计手记.md"
			a.newDocument()
			a.current().Name = "Writing guide.md"
			a.selected = f.ID
			width, height := 960, 640
			if scene == "writing" {
				width, height = 1320, 840
			}
			tt := ui.NewTester(a.View, width, height)
			w := a.writer(f)
			switch scene {
			case "slash", "filtered", "no-results", "bottom":
				w.Document.SetSelection(w.Document.Text().Len(), w.Document.Text().Len())
				tt.Frame()
				tt.Type("/")
				if scene == "filtered" {
					tt.Type("标题")
				}
				if scene == "no-results" {
					tt.Type("xyzzy")
				}
				if scene == "bottom" {
					tt.Key(0, ui.KeyUp)
				}
			case "source":
				click(t, tt, "View options")
				click(t, tt, "Markdown source")
			case "view-menu":
				click(t, tt, "View options")
			case "find-error":
				click(t, tt, "Find in document")
				tt.Type("[")
				click(t, tt, "Use regular expression")
			case "find-replace":
				click(t, tt, "Find in document")
				tt.Type("thought")
				click(t, tt, "Show replace")
			case "find-scope":
				start := strings.Index(f.State.Buffer.String(), "a thought worth keeping")
				f.State.Select(start, start+len("a thought worth keeping"))
				tt.Frame()
				click(t, tt, "Find in document")
				click(t, tt, "Find in selection")
				f.State.Buffer.Edit(editor.Edit{Text: "A preface.\n\n"}, editor.Edit{Start: f.State.Buffer.Len(), End: f.State.Buffer.Len(), Text: "A final thought."})
				tt.Frame()
				click(t, tt, "Show replace")
			case "link":
				start := strings.Index(f.State.Buffer.String(), "a thought worth keeping")
				f.State.Select(start, start+len("a thought worth keeping"))
				tt.Frame()
				click(t, tt, "Insert link")
			}
			tt.Move(0, float32(height-1))
			tt.Advance(200 * time.Millisecond)
			out, err := os.Create(filepath.Join(dir, theme+"-"+scene+".png"))
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
