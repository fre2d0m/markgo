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
	inline := func(kind mdeditor.InlineKind, delimiter string) bool {
		if a.preview {
			return a.writer(file).ToggleInline(kind)
		}
		return mdeditor.ToggleInline(st, delimiter)
	}
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
		button("bold", "B", "Bold", func() bool { return inline(mdeditor.InlineBold, "**") })
		button("italic", "I", "Italic", func() bool { return inline(mdeditor.InlineItalic, "*") })
		button("code", "</>", "Inline code", func() bool { return inline(mdeditor.InlineCode, "`") })
		button("link", "Link", "Insert link", func() bool {
			if a.preview {
				return a.writer(file).OpenLink()
			}
			return mdeditor.InsertLink(st)
		})
	})
}

func (a *App) blockFormatting(c *ui.Context, file *mdeditor.File) {
	t := c.Theme()
	apply := func(action func(*mdeditor.Typesetter)) func() bool {
		return func() bool {
			w := a.writer(file)
			if !a.preview {
				w.FromSource()
			} else {
				w.SyncSelection()
			}
			action(w)
			return true
		}
	}
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		ui.Text(c, "Block style").FontSize(11).FontWeight(500).TextColor(t.TextMuted).Padding(6, 10, 8)
		for _, command := range []struct {
			id, glyph, label string
			action           func() bool
		}{
			{"paragraph", "¶", "Paragraph", apply(func(w *mdeditor.Typesetter) { w.Heading(0) })},
			{"heading1", "H1", "Heading 1", apply(func(w *mdeditor.Typesetter) { w.Heading(1) })},
			{"heading2", "H2", "Heading 2", apply(func(w *mdeditor.Typesetter) { w.Heading(2) })},
			{"heading3", "H3", "Heading 3", apply(func(w *mdeditor.Typesetter) { w.Heading(3) })},
			{"list", "", "Bullet list", apply(func(w *mdeditor.Typesetter) { w.SetList(mdeditor.BulletedList) })},
			{"numbered-list", "", "Numbered list", apply(func(w *mdeditor.Typesetter) { w.SetList(mdeditor.NumberedList) })},
			{"task-list", "", "Task list", apply(func(w *mdeditor.Typesetter) { w.SetList(mdeditor.TaskList) })},
			{"quote", "", "Quote", apply(func(w *mdeditor.Typesetter) { w.Quote() })},
			{"code", "", "Code block", apply(func(w *mdeditor.Typesetter) { w.CodeBlock() })},
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
