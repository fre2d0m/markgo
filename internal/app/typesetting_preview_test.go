package app

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestTypesettingPreviewModeHistoryAndAppearance(t *testing.T) {
	p := NewTypesettingPreview()
	tt := ui.NewTester(p.View, 960, 640)
	page := p.current()
	original := page.file.State.Buffer.String()
	page.typesetter.Document.SetSelection(0, 0)
	tt.Frame()
	tt.Type("New ")
	click(t, tt, "Source mode")
	tt.Type("source ")
	click(t, tt, "Typeset mode")
	tt.Command("undo")
	if got := page.file.State.Buffer.String(); got != "# New "+original[2:] {
		t.Fatalf("source undo: %q", got)
	}
	click(t, tt, "Appearance")
	click(t, tt, "Graphite theme")
	click(t, tt, "Book typography")
	tt.Key(0, ui.KeyEscape)
	tt.Command("undo")
	if got := page.file.State.Buffer.String(); got != original {
		t.Fatalf("history lost after appearance: %q", got)
	}
	click(t, tt, "Open sample Blank page")
	tt.Type("# 新的一页")
	if got := p.current().typesetter.Document.Text().String(); got != "新的一页" {
		t.Fatalf("typed heading not projected: %q", got)
	}
	click(t, tt, "Open sample A quieter morning")
	if got := p.current().file.State.Buffer.String(); got != original {
		t.Fatal("switching sample changed its text")
	}
	if p.shell.sessionStore != nil || p.shell.preferenceStore != nil || p.WindowOptions().StateKey != "" {
		t.Fatal("preview attached real persistence")
	}
}

func TestTypesettingPreviewCompositionAndMinimumWindow(t *testing.T) {
	p := NewTypesettingPreview()
	tt := ui.NewTester(p.View, 960, 640)
	page := p.current()
	page.typesetter.Document.SetSelection(0, 1)
	tt.Frame()
	original := page.file.State.Buffer.String()
	tt.Compose("ceshi", 5)
	tt.Key(0, ui.KeyEscape)
	tt.Type("A")
	if got := page.file.State.Buffer.String(); got != original {
		t.Fatal("Escape lost original selection")
	}
	page.typesetter.Document.SetSelection(0, 1)
	tt.Frame()
	tt.Compose("nihao", 5)
	click(t, tt, "Source mode")
	if got := page.file.State.Buffer.String(); got != original {
		t.Fatal("mode switch committed unfinished composition")
	}
	click(t, tt, "Typeset mode")
	click(t, tt, "Toggle sample pages")
	tt.Advance(75 * time.Millisecond)
	tt.Type("新")
	click(t, tt, "Toggle sample pages")
	tt.Advance(200 * time.Millisecond)
	for _, label := range []string{"Typeset mode", "Source mode", "Paragraph style", "Appearance", "Minimize window", "Maximize window", "Close window"} {
		r, ok := tt.Find(label)
		if !ok || r.W < 40 || r.H < 40 || r.X < 0 || r.X+r.W > 960 || r.Y+r.H > 640 {
			t.Fatalf("inaccessible %s: %+v", label, r)
		}
	}
	if !strings.HasPrefix(page.file.State.Buffer.String(), "# 新 quieter") {
		t.Fatalf("lost focus or composition selection: %q", page.file.State.Buffer.String())
	}
}

func TestTypesettingScreenshots(t *testing.T) {
	dir := os.Getenv("MARKGO_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set MARKGO_SCREENSHOT_DIR to capture rendered evidence")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"paper", "warm", "graphite"} {
		for _, scene := range []string{"writing", "chinese", "selection", "source", "appearance", "paragraph", "empty", "notice", "focus", "composition", "syntax", "long", "large-book"} {
			p := NewTypesettingPreview()
			p.shell.appearance.Theme = theme
			w, h := 1320, 840
			if scene != "writing" && scene != "chinese" {
				w, h = 960, 640
			}
			switch scene {
			case "chinese", "composition":
				p.selected = 1
			case "empty":
				p.selected = 3
			case "syntax":
				p.selected = 2
			case "long":
				p.selected = 4
			case "large-book":
				p.shell.appearance.FontSize = 24
				p.shell.appearance.Serif = true
				p.shell.appearance.Width = 840
			case "notice":
				p.current().typesetter.Notice = "Choose one paragraph to change its heading."
			case "focus":
				p.sidebar = false
			}
			tt := ui.NewTester(p.View, w, h)
			switch scene {
			case "selection":
				p.current().typesetter.Document.SetSelection(14, 55)
				tt.Frame()
			case "source":
				click(t, tt, "Source mode")
			case "appearance":
				click(t, tt, "Appearance")
			case "paragraph":
				click(t, tt, "Paragraph style")
			case "composition":
				p.current().typesetter.Document.SetSelection(12, 12)
				tt.Frame()
				tt.Compose("xiezuo", 6)
			}
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
