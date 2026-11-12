package app

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/xddcode/markgo/internal/preferences"
)

func click(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
}

func TestLaunchStartsWritingAndGuidesAreOptIn(t *testing.T) {
	a := New()
	tt := ui.NewTester(a.View, 960, 640)
	if len(a.files) != 1 || a.current().State.Buffer.String() != "" || a.reading {
		t.Fatal("launch did not start on an empty editable document")
	}
	tt.Type("A thought worth keeping.")
	first := a.current()
	if first.State.Buffer.String() != "A thought worth keeping." {
		t.Fatal("typing did not reach the new document")
	}
	click(t, tt, "Document menu")
	click(t, tt, "Writing guide")
	if !a.reading || a.current().Name != "Welcome.md" || len(a.files) != 4 {
		t.Fatal("guides did not open")
	}
	a.current().State.Buffer.Edit(editor.Edit{Text: "My guide\n"})
	a.openGuides()
	if len(a.files) != 4 || !strings.HasPrefix(a.current().State.Buffer.String(), "My guide") {
		t.Fatal("reopening guides discarded edits or duplicated them")
	}
	tt.Key(ui.Cmd, ui.KeyN)
	if a.reading {
		t.Fatal("new document opened in read mode")
	}
	tt.Type("Fresh page")
	if a.current().State.Buffer.String() != "Fresh page" || first.State.Buffer.String() != "A thought worth keeping." {
		t.Fatal("new document did not keep input and existing work separate")
	}
}

func TestThemeSwitchPreservesWritingState(t *testing.T) {
	a := New()
	f := a.current()
	f.State.Buffer.Set(strings.Repeat("A quiet paragraph with **some emphasis**.\n\n", 150))
	tt := ui.NewTester(a.View, 1320, 840)
	f.State.Select(900, 920)
	tt.Frame()
	f.State.SetScrollY(540)
	tt.Frame()
	before, selection, scroll := f.State.Buffer.String(), f.State.Selection(), f.State.ScrollY()
	click(t, tt, "Appearance")
	for _, theme := range []string{"Warm paper", "Graphite", "Paper"} {
		click(t, tt, theme+" theme")
		tt.Frame()
		if f.State.Buffer.String() != before || f.State.Selection() != selection || math.Abs(f.State.ScrollY()-scroll) > 0.01 {
			t.Fatalf("%s changed source, selection or scroll: %v vs %v", theme, f.State.ScrollY(), scroll)
		}
	}
	theme := a.editorTheme
	tt.Frame()
	if theme != a.editorTheme {
		t.Fatal("unchanged appearance recreated the editor theme")
	}
	tt.Key(0, ui.KeyEscape)
	if a.appearanceOpen {
		t.Fatal("Escape left appearance open")
	}
	tt.Type("继续写作")
	if f.State.Buffer.String() == before {
		t.Fatal("closing appearance lost editor focus")
	}
	if !f.State.Buffer.Undo() || f.State.Buffer.String() != before {
		t.Fatal("appearance interfered with undo")
	}
}

func TestExplicitAndSystemAppearance(t *testing.T) {
	a := New()
	tt := ui.NewTester(a.View, 960, 640)
	tt.SetDark(true)
	if a.cachedPalette != "graphite" {
		t.Fatal("system dark appearance was not followed")
	}
	click(t, tt, "Appearance")
	click(t, tt, "Paper theme")
	tt.SetDark(true)
	if a.cachedPalette != "paper" {
		t.Fatal("explicit Paper followed system dark")
	}
	click(t, tt, "Follow system appearance")
	if a.appearance.Theme != "system" || a.cachedPalette != "graphite" {
		t.Fatalf("system checkbox: %+v, %s", a.appearance, a.cachedPalette)
	}
	click(t, tt, "Follow system appearance")
	tt.SetDark(false)
	if a.appearance.Theme != "graphite" || a.cachedPalette != "graphite" {
		t.Fatal("turning system following off did not retain the current appearance")
	}
}

func TestTypographyControlsPreserveSourceAndFitMinimumWindow(t *testing.T) {
	a := New()
	f := a.current()
	f.Name = strings.Repeat("一个很长的文档名称", 20) + ".md"
	f.State.Buffer.Set("First paragraph.\n\nAnother paragraph.")
	tt := ui.NewTester(a.View, 960, 640)
	before := f.State.Buffer.String()
	f.State.Select(6, 15)
	selection := f.State.Selection()
	tt.Key(ui.Cmd, ui.KeyComma)
	if !a.appearanceOpen {
		t.Fatal("appearance shortcut failed")
	}
	for _, label := range []string{"Appearance", "Paper theme", "Warm paper theme", "Graphite theme", "Smaller text", "Larger text", "Wide page width"} {
		r, ok := tt.Find(label)
		if !ok || r.W < 40 || r.H < 40 || r.X < 0 || r.Y < 0 || r.X+r.W > 960 || r.Y+r.H > 640 {
			t.Fatalf("%s has an unusable hit target: %+v", label, r)
		}
	}
	click(t, tt, "Larger text")
	click(t, tt, "Book typography")
	click(t, tt, "Narrow page width")
	if a.appearance.FontSize != 19 || !a.appearance.Serif || a.appearance.Width != 640 || f.State.Font.Family != "serif" {
		t.Fatalf("typography was not applied: %+v", a.appearance)
	}
	if f.State.Buffer.String() != before || f.State.Selection() != selection {
		t.Fatal("typography changed the document")
	}
	tt.Key(0, ui.KeyEscape)
	click(t, tt, "Find in document")
	click(t, tt, "Show replace")
	for _, label := range []string{"Find text", "Replace text", "Previous match", "Next match", "Close find", "Replace match", "Replace all"} {
		r, ok := tt.Find(label)
		if !ok || r.H != 40 || r.X+r.W > 960 {
			t.Fatalf("%s find control: %+v", label, r)
		}
	}
}

func TestAppearanceRoundTripAndInvalidSettings(t *testing.T) {
	store := &preferences.Store{Path: filepath.Join(t.TempDir(), "appearance.json")}
	a := New()
	if err := a.loadAppearance(store); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		a.appearance = appearance{Theme: "warm", FontSize: float32(14 + i%11), Width: 840, Serif: true}
		a.saveAppearance()
	}
	a.WaitForPreferences()
	b := New()
	if err := b.loadAppearance(store); err != nil {
		t.Fatal(err)
	}
	if b.appearance != a.appearance {
		t.Fatalf("restored %+v, wanted %+v", b.appearance, a.appearance)
	}
	if err := os.WriteFile(store.Path, []byte(`{"theme":"unknown","fontSize":200,"width":-1}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := b.loadAppearance(store); err != nil {
		t.Fatal(err)
	}
	if b.appearance != defaultAppearance() {
		t.Fatalf("invalid settings were not normalized: %+v", b.appearance)
	}
	if err := os.WriteFile(store.Path, []byte(`broken json`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := b.loadAppearance(store); err == nil || b.appearance != defaultAppearance() {
		t.Fatal("corrupt preferences replaced the safe defaults")
	}
}
