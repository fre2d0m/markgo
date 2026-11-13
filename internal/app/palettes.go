package app

import "github.com/egoist/mygo/ui"

// Each palette describes the entire writing environment, including readable
// secondary ink. The order is the appearance gallery: two daylight rows,
// followed by one evening row. Persist IDs, never positions in this slice.
type palette struct {
	ID, Name, Description, Paper, Rail, Hover, Pressed, Border, Ink, Muted, Accent string
	Dark                                                                           bool
}

var palettes = []palette{
	{"paper", "Paper", "白纸 · Clear daylight", "#ffffff", "#f7f8fa", "#eef0f3", "#e2e5eb", "#e7e9ee", "#252932", "#5d6571", "#414b5a", false},
	{"warm", "Warm paper", "暖纸 · A softer page", "#fbf8f2", "#f3eee5", "#eee8dd", "#e4dccd", "#e5ddcf", "#38342e", "#675e52", "#77603f", false},
	{"sakura", "Sakura", "樱花粉 · A trace of blossom", "#fff8fa", "#f7edf1", "#f0e1e7", "#e8d3dc", "#ead8e0", "#42343c", "#715460", "#965471", false},
	{"sage", "Sage", "鼠尾草 · Quiet green", "#f7faf6", "#edf2eb", "#e3ecdf", "#d4e0d0", "#dce6d8", "#303c32", "#536553", "#526e50", false},
	{"mist", "Mist", "雾蓝 · An open sky", "#f6f9fc", "#edf2f7", "#e1eaf3", "#d1deeb", "#d9e3ee", "#2d3948", "#536274", "#506e91", false},
	{"lavender", "Lavender", "薰衣草 · A quiet afternoon", "#faf8fd", "#f1edf7", "#e9e2f2", "#dcd2e9", "#e4ddee", "#3c3547", "#665771", "#79628e", false},
	{"sand", "Sand", "沙丘 · Sun-warmed linen", "#faf7ef", "#f1edde", "#e9e2ce", "#ded3b6", "#e4dcc7", "#3e392b", "#625a42", "#796638", false},
	{"pearl", "Pearl", "珍珠灰 · Soft silver light", "#f6f7f7", "#edefef", "#e3e6e6", "#d5dada", "#dce1e1", "#323b3c", "#556163", "#536e70", false},
	{"graphite", "Graphite", "石墨 · A little less light", "#191b20", "#15171b", "#252830", "#303540", "#343943", "#e9ecf1", "#a6adba", "#bcc6d7", true},
	{"ink", "Midnight", "午夜蓝 · After the world slows", "#18212c", "#141b25", "#253345", "#304258", "#35465b", "#e4ecf5", "#a4b5ca", "#a8c3e6", true},
	{"pine", "Pine", "松林 · A sheltered evening", "#1b2521", "#161f1b", "#2a3830", "#35483d", "#394b40", "#e4eee6", "#a7baab", "#afcbb1", true},
	{"plum", "Plum", "暮紫 · The last light", "#26202b", "#201a25", "#362c3e", "#45384f", "#4a3c54", "#eee6f2", "#beabc8", "#cdb2da", true},
}

func knownPalette(id string) (palette, bool) {
	for _, p := range palettes {
		if p.ID == id {
			return p, true
		}
	}
	return palette{}, false
}

func paletteFor(id string, dark bool) palette {
	if id == "system" {
		id = "paper"
		if dark {
			id = "graphite"
		}
	}
	if p, ok := knownPalette(id); ok {
		return p
	}
	p, _ := knownPalette("paper")
	return p
}

func writingTheme(base *ui.Theme, id string) *ui.Theme {
	p := paletteFor(id, base.Dark)
	t := *base
	t.Dark = p.Dark
	t.Background, t.Surface = ui.Hex(p.Paper), ui.Hex(p.Rail)
	t.SurfaceHover, t.SurfacePressed, t.Border = ui.Hex(p.Hover), ui.Hex(p.Pressed), ui.Hex(p.Border)
	t.Text, t.TextMuted = ui.Hex(p.Ink), ui.Hex(p.Muted)
	// Tint is used sparingly: selected controls and the keyboard focus ring.
	// Body text, syntax and icons keep the palette's neutral ink.
	t.Accent, t.AccentText = ui.Hex(p.Accent), t.Background
	t.AccentHover, t.AccentPressed = t.Accent.Mix(t.Text, 0.2), t.Accent.Mix(t.Text, 0.4)
	t.InactiveAccent = t.Accent
	t.Selection, t.Focus = t.Accent.Alpha(0.2), t.Accent
	t.FocusRingWidth, t.FocusRingOffset = 2, 1
	t.Inverse, t.InverseText = t.Text, t.Background
	t.Scrollbar, t.ScrollbarWidth = t.TextMuted.Alpha(0.45), 5
	t.Danger, t.Warning, t.Success = ui.Hex("#a83f4c"), ui.Hex("#826025"), ui.Hex("#466b50")
	if p.Dark {
		t.Danger, t.Warning, t.Success = ui.Hex("#f0a2aa"), ui.Hex("#dec18b"), ui.Hex("#a5c9ab")
	}
	t.Radius, t.FontSize = 8, 13
	return &t
}
