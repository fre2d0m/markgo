package resources

import (
	"bytes"
	"image/png"
	"testing"
)

func TestAppIcon(t *testing.T) {
	config, err := png.DecodeConfig(bytes.NewReader(AppIconPNG))
	if err != nil {
		t.Fatal(err)
	}
	if config.Width != config.Height || config.Width < 1024 {
		t.Fatalf("icon dimensions %dx%d, want a square of at least 1024 pixels", config.Width, config.Height)
	}
}
