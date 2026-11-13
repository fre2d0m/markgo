package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/xddcode/markgo/internal/writingui"
)

func (a *App) appearancePopover(c *ui.Context, anchor ui.Element, writingFocus *ui.Handle) {
	t := c.Theme()
	opening := a.appearanceOpen && !a.appearanceWasOpen
	close := func() {
		a.appearanceOpen = false
		if writingFocus != nil {
			writingFocus.Focus(c)
		}
	}
	ui.PopoverBase(c, anchor, &a.appearanceOpen, func(panel ui.Element) {
		panel.Bind(&a.appearancePanel).OnShortcut(0, ui.KeyEscape, close)
		writingui.Surface(c, panel).Width(408).Padding(16).Gap(8).Margin(8, 0, 0).Label("Appearance panel").Children(func() {
			ui.Row(c).FillWidth().Height(40).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Grow(1).Gap(3).Children(func() {
					ui.Text(c, "Appearance").FontSize(15).FontWeight(600)
					ui.Text(c, paletteFor(a.appearance.Theme, a.systemDark).Name).FontSize(12).TextColor(t.TextMuted).Role(ui.RoleStatus)
				})
				iconButton(c, "close-appearance", "close", "Close appearance").Bind(&a.appearanceFirst).OnClick(close)
				if opening {
					a.appearanceFirst.Focus(c)
				}
			})
			ui.Column(c.Key("theme-gallery")).FillWidth().Gap(8).Children(func() {
				for row := 0; row < len(palettes); row += 4 {
					ui.Row(c.Key(row)).FillWidth().Gap(8).Children(func() {
						for _, p := range palettes[row:min(row+4, len(palettes))] {
							a.themeCard(c, p)
						}
					})
				}
			})
			follow := a.appearance.Theme == "system"
			followRow := writingui.Interactive(c, ui.CheckboxBase(c.Key("system-theme"), &follow), false).
				FillWidth().Height(40).Padding(0, 8).KeepFocus().Label("Follow system appearance")
			followRow.OnChange(func() {
				if follow {
					a.appearance.Theme = "system"
				} else {
					a.appearance.Theme = paletteFor("system", a.systemDark).ID
				}
				a.saveAppearance()
			}).Children(func() {
				ui.Text(c, "Follow system appearance").FontSize(12).Grow(1).TextColor(t.Text)
				box := ui.Box(c).Size(18, 18).Radius(5).Border(1, t.TextMuted).Center()
				if follow {
					box.Background(t.Accent).Border(1, t.Accent)
				}
				box.Children(func() {
					if follow {
						ui.Icon(c, icons["check"]).Size(13, 13).TextColor(t.AccentText)
					}
				})
			})
			ui.Box(c).FillWidth().Height(1).Background(t.Border)
			ui.Column(c.Key("typography-controls")).FillWidth().Gap(4).Children(func() {
				ui.Row(c).FillWidth().Height(40).Padding(0, 4).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "Text size").FontSize(12).Grow(1).TextColor(t.TextMuted)
					if iconButton(c, "smaller", "minus", "Smaller text").Disabled(a.appearance.FontSize <= 14).Clicked() {
						a.appearance.FontSize--
						a.saveAppearance()
					}
					ui.Textf(c, "%.0f", a.appearance.FontSize).Width(32).TextAlign(ui.Center).FontFeatures("tnum").FontSize(13).Role(ui.RoleStatus)
					if iconButton(c, "larger", "plus", "Larger text").Disabled(a.appearance.FontSize >= 24).Clicked() {
						a.appearance.FontSize++
						a.saveAppearance()
					}
				})
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Text(c, "Typeface").Width(76).PaddingStart(4).FontSize(12).TextColor(t.TextMuted)
					ui.Row(c).Grow(1).Padding(2).Radius(10).Background(t.Surface).Gap(2).Children(func() {
						for _, serif := range []bool{false, true} {
							label, font := "Modern", "system-ui"
							if serif {
								label, font = "Book", "serif"
							}
							b := appearanceSegment(c.Key(label), label+" typography", a.appearance.Serif == serif).Grow(1)
							b.Children(func() { ui.Text(c, label).Font(font).FontSize(13) })
							if b.Clicked() {
								a.appearance.Serif = serif
								a.saveAppearance()
							}
						}
					})
				})
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Text(c, "Page width").Width(76).PaddingStart(4).FontSize(12).TextColor(t.TextMuted)
					ui.Row(c).Grow(1).Padding(2).Radius(10).Background(t.Surface).Gap(2).Children(func() {
						for i, width := range []float32{640, 720, 840} {
							label := []string{"Narrow", "Comfort", "Wide"}[i]
							b := appearanceSegment(c.Key(label), label+" page width", a.appearance.Width == width).Grow(1)
							b.Children(func() { ui.Text(c, label).FontSize(12) })
							if b.Clicked() {
								a.appearance.Width = width
								a.saveAppearance()
							}
						}
					})
				})
			})
		})
		if panel.OverlayShortcut(0, ui.KeyEscape) {
			close()
		}
	})
	// Outside clicks may focus another input; leave that focus alone. Return
	// only when a dismissed/interrupted panel still owns the keyboard. This
	// also covers the anchor toggle and switching directly to Formatting.
	if !a.appearanceOpen && a.appearancePanel.FocusWithin(c) {
		if writingFocus != nil {
			writingFocus.Focus(c)
		} else {
			anchor.Focus()
		}
	}
	a.appearanceWasOpen = a.appearanceOpen
}

func (a *App) themeCard(c *ui.Context, p palette) {
	selected := paletteFor(a.appearance.Theme, a.systemDark).ID == p.ID
	b := ui.ButtonBase(c.Key(p.ID)).Grow(1).MinWidth(0).Height(60).Padding(8).Radius(8).KeepFocus().
		Label(p.Name + " theme").Tooltip(p.Description).Description(p.Description).Cursor(ui.CursorPointer)
	border := ui.Hex(p.Border)
	if b.Hovered() {
		border = ui.Hex(p.Muted)
	}
	if selected {
		border = ui.Hex(p.Accent)
		b.Description(p.Description + " · Selected")
	}
	writingui.Interactive(c, b, false).Background(ui.Hex(p.Paper)).Border(1, border)
	if selected {
		b.Border(1.5, border)
	}
	b.Children(func() {
		ui.Column(c).Fill().Justify(ui.SpaceBetween).Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Aa").Font("serif").FontSize(19).TextColor(ui.Hex(p.Ink)).Grow(1)
				if selected {
					ui.Box(c).Size(16, 16).Radius(8).Background(ui.Hex(p.Accent)).Center().Children(func() {
						ui.Icon(c, icons["check"]).Size(11, 11).TextColor(ui.Hex(p.Paper))
					})
				}
			})
			ui.Text(c, p.Name).FontSize(11).SingleLine().TextColor(ui.Hex(p.Ink))
		})
	})
	b.OnClick(func() { a.appearance.Theme = p.ID; a.saveAppearance() })
}

func appearanceSegment(c *ui.Context, label string, selected bool) ui.Element {
	t := c.Theme()
	b := writingui.Interactive(c, ui.ButtonBase(c), selected).MinWidth(0).Height(40).Center().KeepFocus().Label(label)
	if selected {
		b.Background(t.Background).Border(1, t.Border).TextColor(t.Text).Description("Selected")
	}
	return b
}
