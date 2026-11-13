package app

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/xddcode/markgo/internal/mdeditor"
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

func TestTypesettingListMenuSourceHistoryAndFocus(t *testing.T) {
	for _, sourceMode := range []bool{false, true} {
		p := NewTypesettingPreview()
		p.selected = 3
		p.current().file.State.Buffer.Set("a thought\nanother thought")
		p.current().typesetter.FromSource()
		tt := ui.NewTester(p.View, 960, 640)
		page := p.current()
		if sourceMode {
			click(t, tt, "Source mode")
			page.file.State.Select(0, page.file.State.Buffer.Len())
		} else {
			page.typesetter.Document.SetSelection(0, page.typesetter.Document.Text().Len())
		}
		tt.Frame()
		click(t, tt, "Paragraph style")
		for _, label := range []string{"Paragraph", "Heading 3", "Bulleted list", "Numbered list", "Task list"} {
			r, ok := tt.Find(label)
			if !ok || r.H < 40 || r.Y < 0 || r.Y+r.H > 640 {
				t.Fatalf("format action outside narrow window: %s %+v", label, r)
			}
		}
		click(t, tt, "Task list")
		if got := page.file.State.Buffer.String(); got != "- [ ] a thought\n- [ ] another thought" {
			t.Fatalf("list action in source mode %v: %q", sourceMode, got)
		}
		if sourceMode {
			click(t, tt, "Typeset mode")
		}
		page.typesetter.Document.SetSelection(page.typesetter.Document.Text().Len(), page.typesetter.Document.Text().Len())
		tt.Frame()
		tt.Type("!")
		tt.Command("undo")
		tt.Command("undo")
		if got := page.file.State.Buffer.String(); got != "a thought\nanother thought" {
			t.Fatalf("list menu lost focus or source undo: %q", got)
		}
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
		for _, scene := range []string{"writing", "chinese", "selection", "source", "appearance", "paragraph", "empty", "notice", "focus", "composition", "syntax", "long", "long-middle", "large-book", "list", "list-narrow", "list-tasks", "list-selection", "list-empty", "list-source", "list-large", "list-menu", "list-notice"} {
			p := NewTypesettingPreview()
			p.shell.appearance.Theme = theme
			w, h := 1320, 840
			if scene != "writing" && scene != "chinese" && scene != "list" {
				w, h = 960, 640
			}
			if strings.HasPrefix(scene, "list") {
				p.selected = 5
			}
			switch scene {
			case "chinese", "composition":
				p.selected = 1
			case "empty":
				p.selected = 3
			case "syntax":
				p.selected = 2
			case "long", "long-middle":
				p.selected = 4
			case "large-book":
				p.shell.appearance.FontSize = 24
				p.shell.appearance.Serif = true
				p.shell.appearance.Width = 840
			case "notice":
				p.current().typesetter.Notice = "Choose one paragraph to change its heading."
			case "focus":
				p.sidebar = false
			case "list-empty":
				p.current().file.State.Buffer.Set("# Room for a thought\n\n- [ ] ")
				p.current().typesetter.FromSource()
			case "list-large":
				p.shell.appearance.FontSize = 24
				p.shell.appearance.Serif = true
				p.current().file.State.Buffer.Set("# A few small steps\n\n999999999. Keep the long marker readable.\n\n- [ ] 给还没想清楚的念头留一点空间，让它慢慢生长。\n- [x] Make room for writing.\n")
				p.current().typesetter.FromSource()
			case "list-notice":
				p.current().file.State.Buffer.Set("# Markdown stays yours\n\n```md\n- This is code, not an editable list.\n```\n")
				p.current().typesetter.FromSource()
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
			case "long-middle":
				doc := p.current().typesetter.Document
				at := doc.Text().Len() / 2
				doc.SetSelection(at, at)
				tt.Frame()
				tt.Type("继续写下去。")
			case "list-menu":
				click(t, tt, "Paragraph style")
			case "list-source":
				click(t, tt, "Source mode")
			case "list-selection":
				doc := p.current().typesetter.Document
				shown := doc.Text().String()
				start := strings.Index(shown, "Start with")
				doc.SetSelection(start, start+75)
				tt.Frame()
			case "list-empty", "list-tasks":
				doc := p.current().typesetter.Document
				doc.SetSelection(doc.Text().Len(), doc.Text().Len())
				tt.Frame()
				if scene == "list-tasks" {
					click(t, tt, "Toggle task: 写下一个还没想清楚的念头。")
					if r, ok := tt.Find("Toggle task: Come back when you have something to add."); ok {
						tt.Move(r.X+r.W/2, r.Y+r.H/2)
					}
				}
			case "list-notice":
				doc := p.current().typesetter.Document
				doc.SetSelection(doc.Text().Len()-2, doc.Text().Len()-2)
				p.current().typesetter.SetList(mdeditor.TaskList)
				tt.Frame()
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

func BenchmarkTypesettingPreviewEditCycle100K(b *testing.B) {
	p := NewTypesettingPreview()
	p.selected = 4
	tt := ui.NewTester(p.View, 960, 640)
	doc := p.current().typesetter.Document
	at := doc.Text().Len() / 2
	doc.SetSelection(at, at)
	tt.Frame()
	original := p.current().file.State.Buffer.String()
	b.ReportAllocs()
	for b.Loop() {
		tt.Type("中🙂")
		tt.Key(ui.Cmd, ui.KeyZ)
	}
	if p.current().file.State.Buffer.String() != original {
		b.Fatal("benchmark did not restore its fixture")
	}
}
