package app

import "github.com/egoist/mygo/ui"

const pageToolbarInset = 20

// Both writing surfaces share the same frame. The 44 DIP mode group fits
// inside it with 4 DIP above and below, preserving its 40 DIP button targets.
func pageToolbar(c *ui.Context) ui.Element {
	return ui.Row(c).FillWidth().Height(52).Padding(4, pageToolbarInset).AlignItems(ui.Center)
}

func pageToolbarDivider(c *ui.Context) {
	// FillWidth already occupies the parent's content width. Keep the inset
	// inside that width so the rule cannot extend beyond the paper's edge.
	ui.Box(c.Key("toolbar-divider")).FillWidth().Height(1).PaddingX(pageToolbarInset).Children(func() {
		ui.Box(c).Fill().Background(c.Theme().Border.Alpha(0.4))
	})
}
