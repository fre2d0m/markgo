package app

import (
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/xddcode/markgo/internal/preferences"
)

// Readability is checked on actual interaction surfaces, including the
// selected/pressed sidebar row, rather than just white behind primary ink.
func TestPaletteReadabilityAndNativeAppearance(t *testing.T) {
	if len(palettes) < 12 {
		t.Fatal("the writing gallery needs twelve complete themes")
	}
	seen := map[string]bool{}
	for _, p := range palettes {
		t.Run(p.ID, func(t *testing.T) {
			if seen[p.ID] {
				t.Fatal("duplicate persisted theme ID")
			}
			seen[p.ID] = true
			th := writingTheme(ui.LightTheme(), p.ID)
			for name, ink := range map[string]ui.Color{"text": th.Text, "secondary": th.TextMuted} {
				for surface, paper := range map[string]ui.Color{"page": th.Background, "sidebar": th.Surface, "hover": th.SurfaceHover, "pressed": th.SurfacePressed} {
					if ratio := contrast(ink, paper); ratio < 4.5 {
						t.Errorf("%s on %s: %.2f:1; need 4.5:1", name, surface, ratio)
					}
				}
			}
			for state, fill := range map[string]ui.Color{"normal": th.Accent, "hover": th.AccentHover, "pressed": th.AccentPressed} {
				if ratio := contrast(th.AccentText, fill); ratio < 4.5 {
					t.Errorf("primary %s: %.2f:1", state, ratio)
				}
			}
			for name, ink := range map[string]ui.Color{"error": th.Danger, "warning": th.Warning, "success": th.Success, "focus": th.Focus} {
				if ratio := contrast(ink, th.Background); ratio < 4.5 {
					t.Errorf("%s on page: %.2f:1", name, ratio)
				}
			}
			// Every explicit color choice must also put OS dialogs in the
			// appropriate light/dark appearance and survive a real preference write.
			a := New()
			store := &preferences.Store{Path: filepath.Join(t.TempDir(), "appearance.json")}
			if err := a.loadAppearance(store); err != nil {
				t.Fatal(err)
			}
			a.appearance.Theme = p.ID
			a.saveAppearance()
			a.WaitForPreferences()
			b := New()
			if err := b.loadAppearance(store); err != nil {
				t.Fatal(err)
			}
			want := mygo.ThemeLight
			if p.Dark {
				want = mygo.ThemeDark
			}
			if b.appearance.Theme != p.ID || b.ThemeSource() != want {
				t.Fatalf("theme did not round trip: %+v", b.appearance)
			}
		})
	}
}

func contrast(a, b ui.Color) float64 {
	luminance := func(c ui.Color) float64 {
		channel := func(v uint8) float64 {
			x := float64(v) / 255
			if x <= 0.04045 {
				return x / 12.92
			}
			return math.Pow((x+0.055)/1.055, 2.4)
		}
		return 0.2126*channel(c.R) + 0.7152*channel(c.G) + 0.0722*channel(c.B)
	}
	x, y := luminance(a), luminance(b)
	return (max(x, y) + 0.05) / (min(x, y) + 0.05)
}

func TestThemeGalleryFitsAndPreservesWriting(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		a := New()
		f := a.current()
		f.State.Buffer.Set(strings.Repeat("给想法留一些空间。 A little room to think.\n\n", 100))
		tt := ui.NewTester(a.View, 960, 640)
		tt.SetScale(scale)
		f.State.Select(300, 330)
		tt.Frame()
		f.State.SetScrollY(300)
		tt.Frame()
		before, selection, scroll := f.State.Buffer.String(), f.State.Selection(), f.State.ScrollY()
		click(t, tt, "Appearance")
		tt.Advance(200 * time.Millisecond)
		labels := []string{"Close appearance", "Follow system appearance", "Smaller text", "Larger text", "Modern typography", "Book typography", "Narrow page width", "Comfort page width", "Wide page width"}
		for _, p := range palettes {
			labels = append(labels, p.Name+" theme")
		}
		var targets []ui.Rect
		for _, label := range labels {
			r, ok := tt.Find(label)
			if !ok || r.W < 40 || r.H < 40 || r.X < 0 || r.Y < 0 || r.X+r.W > 960 || r.Y+r.H > 640 {
				t.Fatalf("scale %v: %s unusable at %+v", scale, label, r)
			}
			for i, other := range targets {
				if min(r.X+r.W, other.X+other.W)-max(r.X, other.X) > 0.1 && min(r.Y+r.H, other.Y+other.H)-max(r.Y, other.Y) > 0.1 {
					t.Fatalf("%s overlaps %s", label, labels[i])
				}
			}
			targets = append(targets, r)
		}
		for _, p := range palettes {
			click(t, tt, p.Name+" theme")
			tt.Advance(150 * time.Millisecond)
			if a.appearance.Theme != p.ID || f.State.Buffer.String() != before || f.State.Selection() != selection || math.Abs(f.State.ScrollY()-scroll) > 0.01 {
				t.Fatalf("%s changed writing state", p.ID)
			}
		}
		click(t, tt, "Close appearance")
		tt.Type("继续")
		if !f.State.Buffer.Undo() || f.State.Buffer.String() != before {
			t.Fatal("closing the gallery lost writing focus or undo history")
		}
	}
}

func TestAppearancePointerCancellationAndKeyboardReturn(t *testing.T) {
	for _, preview := range []bool{false, true} {
		for _, reduce := range []bool{false, true} {
			p := NewTypesettingPreview()
			a := p.shell
			view, source := a.View, a.current().State.Buffer
			if preview {
				view, source = p.View, p.current().file.State.Buffer
			}
			tt := ui.NewTester(view, 960, 640)
			tt.SetPreferences(ui.Preferences{ReduceMotion: reduce})
			before := source.String()
			click(t, tt, "Appearance")
			tt.Advance(200 * time.Millisecond)
			r, _ := tt.Find("Sakura theme")
			tt.Press(r.X+r.W/2, r.Y+r.H/2)
			tt.Move(920, 620)
			tt.Release(920, 620)
			if a.appearance.Theme != "paper" {
				t.Fatal("dragging out selected a theme")
			}
			// Tab moves through the panel. Escape must still restore the editor,
			// rather than leaving a hidden theme card owning the keyboard.
			tt.Key(0, ui.KeyTab)
			if !tt.Focused("Paper theme") {
				t.Fatal("Tab did not reach the theme gallery")
			}
			tt.Key(0, ui.KeyEscape)
			tt.Advance(200 * time.Millisecond)
			if a.appearanceOpen {
				t.Fatalf("preview=%v reduced=%v: Escape did not close appearance", preview, reduce)
			}
			tt.Type("A new thought")
			if source.String() == before {
				t.Fatalf("preview=%v reduced=%v: keyboard did not return to writing", preview, reduce)
			}
		}
	}
}

func TestThemeKeyboardChoiceAndCloseResumeBothEditors(t *testing.T) {
	for _, sourceMode := range []bool{false, true} {
		p := NewTypesettingPreview()
		p.selected = 3
		tt := ui.NewTester(p.View, 960, 640)
		if sourceMode {
			click(t, tt, "Source mode")
		}
		tt.Type("An unfinished thought.")
		before := p.current().file.State.Buffer.String()
		click(t, tt, "Appearance")
		for i := 0; i < 32 && !tt.Focused("Sakura theme"); i++ {
			tt.Key(0, ui.KeyTab)
		}
		if !tt.Focused("Sakura theme") {
			t.Fatal("theme gallery is not reachable by keyboard")
		}
		tt.Key(0, ui.KeySpace)
		if p.shell.appearance.Theme != "sakura" {
			t.Fatal("keyboard activation did not select theme")
		}
		click(t, tt, "Close appearance")
		tt.Type(" Continue here.")
		if p.current().file.State.Buffer.String() != before+" Continue here." {
			t.Fatalf("source=%v: closing lost the caret", sourceMode)
		}
		tt.Command("undo")
		if p.current().file.State.Buffer.String() != before {
			t.Fatal("theme selection polluted source history")
		}
	}
}

func TestDocumentMenuTargetsAndKeyboardDismissal(t *testing.T) {
	a := New()
	tt := ui.NewTester(a.View, 960, 640)
	tt.Type("A thought.")
	click(t, tt, "Document menu")
	tt.Advance(200 * time.Millisecond)
	for _, label := range []string{"New document", "Open document…", "Recent documents…", "Open folder…", "Save", "Save as…", "Close document", "Show Markdown source", "Writing guide"} {
		r, ok := tt.Find(label)
		if !ok || r.H < 40 || r.X < 0 || r.Y < 0 || r.Y+r.H > 640 {
			t.Fatalf("menu target %s: %+v", label, r)
		}
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEscape)
	tt.Type(" Continue.")
	if a.current().State.Buffer.String() != "A thought. Continue." {
		t.Fatal("menu dismissal lost writing focus")
	}
}

// Opt-in evidence uses the application's actual Go views, never a separate
// mockup. All palettes get both shells, typography extremes and the gallery.
func TestProductPolishScreenshots(t *testing.T) {
	dir := os.Getenv("MARKGO_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set MARKGO_SCREENSHOT_DIR for rendered UI evidence")
	}
	dir = filepath.Join(dir, "gallery")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, palette := range palettes {
		for _, scene := range []string{"writing", "appearance", "large-book", "main-writing", "main-reading", "main-find", "main-empty", "main-outline", "main-format", "main-appearance", "main-focus", "main-menu", "recent-empty"} {
			p := NewTypesettingPreview()
			a := p.shell
			a.appearance.Theme = palette.ID
			w, h := 960, 640
			if scene == "writing" {
				w, h = 1320, 840
				p.selected = 1
			}
			if scene == "large-book" {
				a.appearance.FontSize = 24
				a.appearance.Serif = true
				a.appearance.Width = 840
			}
			view := p.View
			if strings.HasPrefix(scene, "main-") || scene == "recent-empty" {
				view = a.View
				a.current().Name = "A quieter morning.md"
				a.current().State.Buffer.Set(quietMorning)
				if scene == "main-empty" {
					a.current().State.Buffer.Set("")
				}
				if scene == "main-reading" {
					a.reading = true
				}
				if scene == "main-outline" {
					a.sideView = 1
				}
				if scene == "recent-empty" {
					a.recentOpen = true
				}
			}
			tt := ui.NewTester(view, w, h)
			switch scene {
			case "appearance", "main-appearance":
				click(t, tt, "Appearance")
			case "main-focus":
				click(t, tt, "Focus mode")
			case "main-menu":
				click(t, tt, "Document menu")
			case "main-format":
				click(t, tt, "Formatting")
			case "main-find":
				click(t, tt, "Find in document")
				tt.Type("morning")
				click(t, tt, "Show replace")
			}
			tt.Move(0, 639)
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
