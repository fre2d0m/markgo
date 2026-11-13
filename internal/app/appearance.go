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
	// A theme choice can change in the header during this build, after View
	// resolved its context theme. Cache colors from the new choice, never the
	// previous frame's palette under the new palette's cache key.
	t = writingTheme(t, id)
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
