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

// The recording advances MyGo's own animation clock. It is rendered evidence,
// not a timing measurement of a native GPU or an interpolated mockup.
func TestProductPolishMotion(t *testing.T) {
	dir := os.Getenv("MARKGO_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set MARKGO_SCREENSHOT_DIR to record the UI")
	}
	dir = filepath.Join(dir, "motion")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	p := NewTypesettingPreview()
	tt := ui.NewTester(p.View, 960, 640)
	before := p.current().file.State.Buffer.String()
	frame := 0
	capture := func(delay time.Duration) {
		tt.Advance(delay)
		out, err := os.Create(filepath.Join(dir, fmt.Sprintf("%03d.png", frame)))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(out, tt.Image())
		_ = out.Close()
		if err != nil {
			t.Fatal(err)
		}
		frame++
	}
	settle := func() {
		for range 7 {
			capture(20 * time.Millisecond)
		}
	}
	capture(0)
	click(t, tt, "Appearance")
	settle()
	r, _ := tt.Find("Sakura theme")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	settle()
	tt.Press(r.X+r.W/2, r.Y+r.H/2)
	settle()
	tt.Release(r.X+r.W/2, r.Y+r.H/2)
	settle()
	tt.Key(0, ui.KeyEscape)
	settle()
	click(t, tt, "Paragraph style")
	settle()
	tt.Key(0, ui.KeyEscape)
	settle()
	if p.current().file.State.Buffer.String() != before {
		t.Fatal("presentation changed the source")
	}
}
