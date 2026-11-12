package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/xddcode/markgo/internal/mdeditor"
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
		item := ui.ButtonBase(c.Key(id)).MinWidth(36).Height(32).Padding(5, 9).
			Radius(6).KeepFocus().Label(label).Tooltip(label)
		if item.Hovered() {
			item.Background(t.SurfaceHover)
		}
		item.Children(func() {
			ui.Text(c, text).FontSize(13).TextColor(t.Text)
		})
		if item.Clicked() {
			a.format(file, action)
			file.Focus.Focus(c)
		}
	}
	separator := func() { ui.Box(c).Width(1).Height(18).MarginX(4).Background(t.Border) }
	ui.Row(c).FillWidth().Height(42).Padding(0, 22).Gap(2).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(t.Border.Alpha(0.5)).Children(func() {
		button("heading1", "H1", "Heading 1", func() bool { return mdeditor.SetHeading(st, 1) })
		button("heading2", "H2", "Heading 2", func() bool { return mdeditor.SetHeading(st, 2) })
		separator()
		button("bold", "B", "Bold", func() bool { return mdeditor.ToggleInline(st, "**") })
		button("italic", "I", "Italic", func() bool { return mdeditor.ToggleInline(st, "*") })
		button("code", "</>", "Inline code", func() bool { return mdeditor.ToggleInline(st, "`") })
		button("link", "Link", "Insert link", func() bool { return mdeditor.InsertLink(st) })
		separator()
		button("list", "• List", "Bullet list", func() bool { return mdeditor.ToggleLinePrefix(st, "- ") })
		button("quote", "❝", "Quote", func() bool { return mdeditor.ToggleLinePrefix(st, "> ") })
		ui.Spacer(c)
		if ui.Button(c.Key("find"), "Find").Height(32).KeepFocus().Label("Find in document").Tooltip("Find and replace in this document").Clicked() {
			st.Run(editor.Find)
		}
	})
}
