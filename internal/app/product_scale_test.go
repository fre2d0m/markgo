package app

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestProductScaleScreenshots(t *testing.T) {
	dir := os.Getenv("MARKGO_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set MARKGO_SCREENSHOT_DIR to inspect display scale")
	}
	dir = filepath.Join(dir, "scale")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"paper", "sakura", "graphite", "warm"} {
		for _, scale := range []float32{1.5, 2} {
			a := New()
			a.appearance.Theme = theme
			a.current().Name = "A quieter morning.md"
			a.current().State.Buffer.Set(quietMorning)
			tt := ui.NewTester(a.View, 960, 640)
			tt.SetScale(scale)
			label := "Appearance"
			if theme == "warm" {
				label = "Formatting"
			}
			click(t, tt, label)
			tt.Move(0, 639)
			tt.Advance(200 * time.Millisecond)
			out, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s-%gx.png", theme, scale)))
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
