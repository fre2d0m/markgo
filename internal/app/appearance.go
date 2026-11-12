package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
)

type appearance struct {
	Theme    string  `json:"theme"`
	FontSize float32 `json:"fontSize"`
	Width    float32 `json:"width"`
	Serif    bool    `json:"serif"`
}

func defaultAppearance() appearance { return appearance{Theme: "system", FontSize: 18, Width: 720} }

type palette struct {
	ID, Name, Description, Paper, Rail, Hover, Pressed, Border, Ink, Muted string
	Dark                                                                   bool
}

var palettes = []palette{
	{"paper", "Paper", "Clear, quiet daylight", "#fcfcfa", "#f4f4f1", "#eaeae5", "#deded7", "#e3e3dc", "#272824", "#6d6f66", false},
	{"warm", "Warm paper", "A softer page", "#f8f4eb", "#f0ebe0", "#e8e1d4", "#ddd4c3", "#e0d8ca", "#35312b", "#746b5e", false},
	{"graphite", "Graphite", "A little less light", "#21221f", "#1a1b18", "#30322c", "#3c3f36", "#363930", "#e9eae2", "#a7aa9d", true},
}

func paletteFor(id string, dark bool) palette {
	if id == "system" && dark {
		return palettes[2]
	}
	for _, p := range palettes {
		if p.ID == id {
			return p
		}
	}
	return palettes[0]
}

func writingTheme(base *ui.Theme, id string) *ui.Theme {
	p := paletteFor(id, base.Dark)
	t := *base
	t.Dark = p.Dark
	t.Background, t.Surface = ui.Hex(p.Paper), ui.Hex(p.Rail)
	t.SurfaceHover, t.SurfacePressed, t.Border = ui.Hex(p.Hover), ui.Hex(p.Pressed), ui.Hex(p.Border)
	t.Text, t.TextMuted = ui.Hex(p.Ink), ui.Hex(p.Muted)
	t.Accent, t.AccentHover, t.AccentPressed, t.AccentText = t.Text, t.Text.Alpha(0.85), t.Text.Alpha(0.72), t.Background
	t.InactiveAccent = t.Text
	t.Selection, t.Focus = t.Text.Alpha(0.14), t.Text.Alpha(0.55)
	t.Radius, t.FontSize = 8, 13
	return &t
}

func proseColors(t *ui.Theme) *editor.Theme {
	th := editor.DefaultTheme(t)
	for i := range th.Tokens {
		th.Tokens[i].Color = t.Text
	}
	th.Tokens[editor.TokenComment].Color = t.TextMuted
	th.Tokens[editor.TokenPunctuation].Color = t.TextMuted
	th.Tokens[editor.TokenKeyword].Bold = true
	th.Match, th.CurrentMatch = t.Text.Alpha(0.09), t.Text.Alpha(0.22)
	return th
}

func (a *App) applyTypography(f *markdown.Options, t *ui.Theme) {
	// Reuse resolved typography while appearance stays the same. Switching
	// documents still keeps each reader's own image cache.
	id := paletteFor(a.appearance.Theme, a.systemDark).ID
	if a.editorTheme != nil && a.cachedAppearance == a.appearance && a.cachedPalette == id {
		f.Theme = a.readerTheme
		return
	}
	a.cachedAppearance, a.cachedPalette = a.appearance, id
	a.editorTheme = proseColors(t)
	font := "system-ui"
	if a.appearance.Serif {
		font = "serif"
	}
	a.readerTheme = &markdown.Theme{
		Font: font, FontSize: a.appearance.FontSize, LineHeight: 1.85, BlockSpacing: 1.0,
		Headings: [6]markdown.HeadingStyle{{Size: 1.9, Weight: 600, Top: 1.1, Bottom: 0.65}, {Size: 1.35, Weight: 600, Top: 1.65, Bottom: 0.65}, {Size: 1.12, Weight: 600, Top: 1.2, Bottom: 0.6}},
		Text:     t.Text, Muted: t.TextMuted, Link: t.Text, Strong: t.Text,
		Code: t.Text, CodeBackground: t.Surface, Rule: t.Border, Mark: t.Selection,
		CodeColors: a.editorTheme, CodeBlockBackground: t.Surface, CodeBlockBorder: t.Border,
		CodeBlockRadius: 6, QuoteBar: t.Border, QuoteText: t.TextMuted, QuoteBarWidth: 2,
		TableHeader: t.Surface, TableBorder: t.Border, Checkbox: t.Text,
	}
	f.Theme = a.readerTheme
}

func (a *App) appearancePopover(c *ui.Context, anchor ui.Element) {
	t := c.Theme()
	ui.PopoverBase(c, anchor, &a.appearanceOpen, func(panel ui.Element) {
		panel.Width(320).Padding(18).Gap(16).Margin(6, 0, 0).Background(t.Background).
			Radius(12).Border(1, t.Border).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.12)).Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Make yourself at home").FontSize(14).FontWeight(600).Grow(1)
			})
			ui.Text(c, "APPEARANCE").FontSize(10).LetterSpacing(0.6).TextColor(t.TextMuted)
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				for _, p := range palettes {
					p := p
					button := ui.ButtonBase(c.Key(p.ID)).Grow(1).Height(96).Padding(10).Radius(8).KeepFocus().
						Label(p.Name+" theme").Tooltip(p.Description).Background(ui.Hex(p.Paper)).Border(1, ui.Hex(p.Border))
					if a.appearance.Theme == p.ID {
						button.Border(2, t.Text)
					}
					button.Children(func() {
						ui.Column(c).Fill().Justify(ui.SpaceBetween).Children(func() {
							ui.Text(c, "Aa").Font("serif").FontSize(24).TextColor(ui.Hex(p.Ink))
							ui.Text(c, p.Name).FontSize(10).SingleLine().TextColor(ui.Hex(p.Ink))
						})
					})
					if button.Clicked() {
						a.appearance.Theme = p.ID
						a.saveAppearance()
					}
				}
			})
			follow := a.appearance.Theme == "system"
			ui.CheckboxBase(c.Key("system-theme"), &follow).FillWidth().Height(40).KeepFocus().Label("Follow system appearance").OnChange(func() {
				if follow {
					a.appearance.Theme = "system"
				} else {
					a.appearance.Theme = paletteFor("system", a.systemDark).ID
				}
				a.saveAppearance()
			}).Children(func() {
				ui.Text(c, "Follow system appearance").FontSize(12).Grow(1)
				box := ui.Box(c).Size(18, 18).Radius(4).Border(1, t.TextMuted).Center()
				if follow {
					box.Background(t.Text)
				}
				box.Children(func() {
					if follow {
						ui.Icon(c, icons["check"]).Size(14, 14).TextColor(t.Background)
					}
				})
			})
			ui.Box(c).FillWidth().Height(1).Background(t.Border)
			ui.Text(c, "TYPE & SPACE").FontSize(10).LetterSpacing(0.6).TextColor(t.TextMuted)
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Text size").FontSize(12).Grow(1)
				if iconButton(c, "smaller", "minus", "Smaller text").Disabled(a.appearance.FontSize <= 14).Clicked() {
					a.appearance.FontSize--
					a.saveAppearance()
				}
				ui.Textf(c, "%.0f", a.appearance.FontSize).Width(26).TextAlign(ui.Center).FontFeatures("tnum").FontSize(12)
				if iconButton(c, "larger", "plus", "Larger text").Disabled(a.appearance.FontSize >= 24).Clicked() {
					a.appearance.FontSize++
					a.saveAppearance()
				}
			})
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				for _, serif := range []bool{false, true} {
					label, font := "Modern", "system-ui"
					if serif {
						label, font = "Book", "serif"
					}
					b := ui.ButtonBase(c.Key(label)).Grow(1).Height(40).Radius(6).Center().KeepFocus().Label(label + " typography").Background(t.Surface)
					if a.appearance.Serif == serif {
						b.Border(1, t.TextMuted)
					}
					b.Children(func() { ui.Text(c, label).Font(font).FontSize(13) })
					if b.Clicked() {
						a.appearance.Serif = serif
						a.saveAppearance()
					}
				}
			})
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				for _, width := range []float32{640, 720, 840} {
					label := map[float32]string{640: "Narrow", 720: "Comfort", 840: "Wide"}[width]
					b := ui.ButtonBase(c.Key(label)).Grow(1).Height(40).Radius(6).Center().KeepFocus().Label(label + " page width")
					if a.appearance.Width == width {
						b.Background(t.SurfaceHover)
					}
					b.Children(func() { ui.Text(c, label).FontSize(11) })
					if b.Clicked() {
						a.appearance.Width = width
						a.saveAppearance()
					}
				}
			})
		})
	})
}
