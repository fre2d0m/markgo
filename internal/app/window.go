package app

import (
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const titleBarHeight = 60
const sidebarWidth = 240

// Keep the OS frame's resize edges, rounded corners and shadow while drawing
// the title bar and all three controls ourselves. A borderless window would
// lose Windows' top resize edge.
func (a *App) WindowOptions(stateKey string) mygo.WindowOptions {
	return mygo.WindowOptions{
		Title: "MarkGo", Width: 1320, Height: 840, MinWidth: 960, MinHeight: 640,
		StateKey: stateKey, TitleBarStyle: mygo.TitleBarHidden,
		TitleBarHeight: titleBarHeight, HideWindowButtons: true,
		BackgroundColor: "light-dark(#ffffff, #191b20)", Content: ui.View(a.View),
	}
}

// The real window implements these actions. A fake lets headless tests check
// dispatch without opening windows or bypassing the normal close event.
type windowCommands interface {
	Minimize()
	ToggleMaximize()
	Close()
}

func (a *App) captionButtons(c *ui.Context) {
	t := c.Theme()
	ui.Row(c.Key("caption-buttons")).Height(titleBarHeight).MarginStart(10).Children(func() {
		button := func(key, glyph, label string, action func()) {
			b := ui.ButtonBase(c.Key(key)).Size(44, titleBarHeight).KeepFocus().Center().Label(label).Tooltip(label)
			ink, fill := t.TextMuted, ui.Transparent
			if b.Hovered() {
				ink, fill = t.Text, t.SurfaceHover
			}
			if b.Pressed() {
				fill = t.SurfacePressed
			}
			if key == "close-window" && (b.Hovered() || b.Pressed()) {
				ink, fill = ui.Hex("#ffffff"), ui.Hex("#c94146")
			}
			if key == "maximize-window" {
				b.MaximizeButton()
			}
			b.Background(b.AnimateColor("fill", fill, ui.Animation{Duration: 110 * time.Millisecond, NonMotion: true}))
			b.Children(func() { ui.Icon(c, icons[glyph]).Size(12, 12).TextColor(ink) })
			if b.Clicked() && a.windowCommands != nil {
				action()
			}
		}
		button("minimize-window", "window-minimize", "Minimize window", func() { a.windowCommands.Minimize() })
		glyph, label := "window-maximize", "Maximize window"
		if a.windowMaximized {
			glyph, label = "window-restore", "Restore window"
		}
		button("maximize-window", glyph, label, func() { a.windowCommands.ToggleMaximize() })
		button("close-window", "window-close", "Close window", func() { a.windowCommands.Close() })
	})
}
