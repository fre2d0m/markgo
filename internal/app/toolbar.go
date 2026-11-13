package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/xddcode/markgo/internal/mdeditor"
	"github.com/xddcode/markgo/internal/writingui"
)

func (a *App) format(file *mdeditor.File, action func() bool) {
	if !action() {
		a.notice = "Choose a single selection or line to apply this format."
	}
}

func (a *App) editorToolbar(c *ui.Context, file *mdeditor.File) {
	t := c.Theme()
	st := file.State
	button := func(id, text, label string, action func() bool) {
		item := writingui.Interactive(c, ui.ButtonBase(c.Key(id)), false).Size(40, 40).Center().KeepFocus().Label(label).Tooltip(label)
		item.Children(func() {
			if icon := icons[id]; icon != nil {
				ui.Icon(c, icon).Size(18, 18).TextColor(t.Text)
			} else {
				ui.Text(c, text).FontSize(12).FontWeight(500).TextColor(t.Text)
			}
		})
		if item.Clicked() {
			a.format(file, action)
			a.formatOpen = false
			file.Focus.Focus(c)
		}
	}
	ui.Row(c).FillWidth().Height(40).Gap(2).AlignItems(ui.Center).Children(func() {
		button("bold", "B", "Bold", func() bool { return mdeditor.ToggleInline(st, "**") })
		button("italic", "I", "Italic", func() bool { return mdeditor.ToggleInline(st, "*") })
		button("code", "</>", "Inline code", func() bool { return mdeditor.ToggleInline(st, "`") })
		button("link", "Link", "Insert link", func() bool { return mdeditor.InsertLink(st) })
	})
}

func (a *App) blockFormatting(c *ui.Context, file *mdeditor.File) {
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		ui.Text(c, "Paragraph").FontSize(11).FontWeight(500).TextColor(t.TextMuted).Padding(6, 10, 8)
		for _, command := range []struct {
			id, glyph, label string
			action           func() bool
		}{
			{"heading1", "H1", "Heading 1", func() bool { return mdeditor.SetHeading(file.State, 1) }},
			{"heading2", "H2", "Heading 2", func() bool { return mdeditor.SetHeading(file.State, 2) }},
			{"list", "", "Bullet list", func() bool { return mdeditor.ToggleLinePrefix(file.State, "- ") }},
			{"quote", "", "Quote", func() bool { return mdeditor.ToggleLinePrefix(file.State, "> ") }},
		} {
			button := writingui.Interactive(c, ui.ButtonBase(c.Key(command.id)), false).FillWidth().Height(40).Padding(0, 10).KeepFocus().Label(command.label)
			button.Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
					ui.Box(c).Size(24, 24).Center().Children(func() {
						if icon := icons[command.id]; icon != nil {
							ui.Icon(c, icon).Size(18, 18)
						} else {
							ui.Text(c, command.glyph).FontSize(12).FontWeight(500)
						}
					})
					ui.Text(c, command.label).FontSize(13)
				})
			})
			button.OnClick(func() {
				a.format(file, command.action)
				a.formatOpen = false
				file.Focus.Focus(c)
			})
		}
	})
}
