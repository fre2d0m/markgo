// Package writingui keeps the workbench and the native writing overlays on
// the same visual and interaction scale. It owns no document or widget state.
package writingui

import (
	"time"

	"github.com/egoist/mygo/ui"
)

const (
	ControlHeight = 40
	ControlRadius = 8
	PanelRadius   = 16
)

// Surface gives floating content a quiet edge, depth and reversible entrance.
// MyGo suppresses movement with the system's reduced-motion preference.
func Surface(c *ui.Context, panel ui.Element) ui.Element {
	t := c.Theme()
	shadow := float32(0.12)
	if t.Dark {
		shadow = 0.28
	}
	return panel.Background(t.Background).Radius(PanelRadius).Border(1, t.Border).
		NoShadow().Shadow(0, 10, 32, 0, ui.RGBA(0, 0, 0, shadow)).Transition(ui.ElementTransition{
		Duration: 130 * time.Millisecond, ExitDuration: 90 * time.Millisecond, Colors: true,
		Enter: &ui.Motion{Y: -4, Opacity: 0, Flip: true}, Exit: &ui.Motion{Y: -2, Opacity: 0, Flip: true},
	})
}

// Interactive styles a native base widget, retaining its focus, cancellation,
// keyboard and accessibility behavior. Layout and focus policy stay at the call site.
func Interactive(c *ui.Context, b ui.Element, selected bool) ui.Element {
	t := c.Theme()
	fill, ink := ui.Transparent, t.TextMuted
	if selected || b.Hovered() {
		fill, ink = t.SurfaceHover, t.Text
	}
	if b.Pressed() {
		fill, ink = t.SurfacePressed, t.Text
	}
	b.Background(b.AnimateColor("control-fill", fill, ui.Animation{Duration: 100 * time.Millisecond, NonMotion: true}))
	b.TextColor(ink).Radius(ControlRadius).Cursor(ui.CursorPointer)
	press(b)
	return b
}

func Primary(c *ui.Context, b ui.Element) ui.Element {
	t := c.Theme()
	fill := t.Accent
	if b.Hovered() {
		fill = t.AccentHover
	}
	if b.Pressed() {
		fill = t.AccentPressed
	}
	b.Background(b.AnimateColor("primary-fill", fill, ui.Animation{Duration: 100 * time.Millisecond, NonMotion: true})).
		TextColor(t.AccentText).Radius(ControlRadius).Cursor(ui.CursorPointer)
	press(b)
	return b
}

func press(b ui.Element) {
	scale := float32(1)
	if b.Pressed() {
		scale = 0.96
	}
	b.AnimatePose(ui.Pose{Scale: scale}, ui.Animation{Duration: 90 * time.Millisecond, Ease: ui.EaseOut})
}

func Button(c *ui.Context, label string, primary bool) ui.Element {
	b := ui.ButtonBase(c).Height(ControlHeight).Padding(0, 14).Center().Label(label)
	if primary {
		Primary(c, b)
	} else {
		Interactive(c, b, false).TextColor(c.Theme().Text)
	}
	b.Children(func() { ui.Text(c, label).FontSize(13).FontWeight(500).SingleLine() })
	return b
}

// Field keeps the native caret and keyboard focus ring; the border strengthens
// on hover and invalid fields also carry an explicit textual explanation.
func Field(c *ui.Context, field ui.Element, invalid bool) ui.Element {
	t := c.Theme()
	border := t.Border
	if field.Hovered() {
		border = t.TextMuted.Alpha(0.5)
	}
	if invalid {
		border = t.Danger
	}
	return field.Height(ControlHeight).Padding(6, 11).Background(t.Surface).
		Border(1, border).Radius(ControlRadius).FontSize(13)
}

// MenuItem keeps MyGo's menu keyboard model, shortcut column and accessible
// name. A 28 DIP label area plus its two 6 DIP insets gives a 40 DIP target.
func MenuItem(menu *ui.Menu, label string) *ui.MenuItem {
	return menu.Item(label).Content(func(content ui.MenuItemContent) {
		c := content.Context
		ui.Column(c).MinHeight(28).Justify(ui.Center).Children(func() {
			ui.Text(c, label).FontSize(13).SingleLine().Ellipsis("…")
		})
	})
}
