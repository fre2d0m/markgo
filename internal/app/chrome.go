package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

func (a *App) header(c *ui.Context) {
	t := c.Theme()
	file := a.current()
	ui.Row(c).FillWidth().Height(56).AlignItems(ui.Center).Children(func() {
		if a.sidebar && !a.focusMode {
			ui.Row(c).Width(240).Shrink(0).FillHeight().Padding(0, 20).Gap(9).AlignItems(ui.Center).Background(t.Surface).Children(func() {
				ui.Icon(c, icons["mark"]).Size(23, 23).TextColor(t.Text)
				ui.Text(c, "markgo").FontSize(19).FontWeight(600).LetterSpacing(-0.5).Grow(1)
				iconButton(c, "workspace-menu", "more", "Document menu").PopoverMenuButton(a.documentMenu)
			})
		}
		ui.Row(c).Grow(1).MinWidth(0).FillHeight().Padding(0, 18).Gap(4).AlignItems(ui.Center).Children(func() {
			if a.focusMode {
				ui.Text(c, file.Name).FontSize(12).TextColor(t.TextMuted).SingleLine().Ellipsis("…").Shrink(1)
				ui.Spacer(c)
				if iconButton(c, "exit-focus", "exit-focus", "Exit focus mode").Clicked() {
					a.focusMode = false
				}
				return
			}
			if iconButton(c, "sidebar", "sidebar", "Toggle file sidebar").Clicked() {
				a.sidebar = !a.sidebar
			}
			if !a.sidebar {
				iconButton(c, "document-menu", "more", "Document menu").PopoverMenuButton(a.documentMenu)
			}
			ui.Text(c, file.Name).FontSize(12).TextColor(t.TextMuted).SingleLine().Ellipsis("…").Shrink(1).MinWidth(0).MarginStart(8)
			if file.Modified() {
				ui.Box(c).Size(5, 5).Radius(3).Background(t.TextMuted).Label("Unsaved changes").Tooltip("Unsaved changes — save with Cmd/Ctrl+S")
			}
			ui.Spacer(c)
			if iconButton(c, "find", "search", "Find in document").Clicked() {
				a.reading = false
				file.State.Run(editor.Find)
			}
			format := iconButton(c, "format", "format", "Formatting")
			if format.Clicked() {
				a.formatOpen = !a.formatOpen
				a.appearanceOpen = false
				a.reading = false
			}
			ui.PopoverBase(c, format, &a.formatOpen, func(panel ui.Element) {
				panel.Width(380).Padding(8).Margin(6, 0, 0).Background(t.Background).Radius(10).Border(1, t.Border).
					Shadow(0, 6, 18, 0, ui.RGBA(0, 0, 0, 0.12)).Children(func() { a.editorToolbar(c, file) })
			})
			readIcon, readLabel := "read", "Read document"
			if a.reading {
				readIcon, readLabel = "edit", "Edit document"
			}
			if iconButton(c, "read", readIcon, readLabel).Clicked() {
				a.reading = !a.reading
				if !a.reading {
					file.Focus.Focus(c)
				}
			}
			if iconButton(c, "focus", "focus", "Focus mode").Clicked() {
				a.focusMode = true
			}
			appearance := iconButton(c, "appearance", "appearance", "Appearance")
			if appearance.Clicked() {
				a.appearanceOpen = !a.appearanceOpen
				a.formatOpen = false
			}
			a.appearancePopover(c, appearance)
		})
	})
}

func (a *App) documentMenu(m *ui.Menu) {
	if m.Item("New document").Icon(menuIcons["plus"]).Shortcut(ui.Cmd, ui.KeyN).Disabled(a.fileBusy).Chosen() {
		a.newDocument()
	}
	if m.Item("Open document…").Icon(menuIcons["file"]).Shortcut(ui.Cmd, ui.KeyO).Disabled(a.fileBusy).Chosen() {
		a.openDialog()
	}
	if m.Item("Open folder…").Icon(menuIcons["folder"]).Disabled(a.fileBusy).Chosen() {
		a.openFolderDialog()
	}
	m.Separator()
	if m.Item("Save").Icon(menuIcons["save"]).Shortcut(ui.Cmd, ui.KeyS).Disabled(a.fileBusy).Chosen() {
		a.saveDocument(false)
	}
	if m.Item("Save as…").Shortcut(ui.Cmd|ui.Shift, ui.KeyS).Disabled(a.fileBusy).Chosen() {
		a.saveDocument(true)
	}
	if m.Item("Close document").Shortcut(ui.Cmd, ui.KeyW).Disabled(a.fileBusy).Chosen() {
		a.closeDocument(a.current())
	}
	m.Separator()
	if m.Item("Show Markdown source").Checked(!a.preview).Chosen() {
		a.preview = !a.preview
		a.reading = false
	}
	m.Separator()
	if m.Item("Writing guide").Chosen() {
		a.openGuides()
	}
}

func (a *App) fileSidebar(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(240).Shrink(0).FillHeight().Padding(14, 12, 12).Gap(14).Background(t.Surface).Children(func() {
		ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
			for index, label := range []string{"Files", "Outline"} {
				button := ui.ButtonBase(c.Key(label)).Height(40).Padding(0, 12).Center().Radius(6).Label(label)
				color := t.TextMuted
				if a.sideView == index {
					color = t.Text
					button.Background(t.Background)
				}
				button.Children(func() { ui.Text(c, label).FontSize(12).FontWeight(500).TextColor(color) })
				if button.Clicked() {
					a.sideView = index
				}
			}
			ui.Spacer(c)
			if iconButton(c, "new", "plus", "New document").Disabled(a.fileBusy).Clicked() {
				a.newDocument()
			}
		})
		if a.sideView == 1 {
			a.outlineSidebar(c, a.current())
			return
		}
		ui.SearchField(c.Key("filter"), &a.query).Height(38).Placeholder("Find a document").Label("Filter documents")
		ui.Row(c).FillWidth().Padding(2, 10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "OPEN DOCUMENTS").FontSize(9).LetterSpacing(0.7).TextColor(t.TextMuted).Grow(1)
			ui.Textf(c, "%d", len(a.files)).FontSize(10).FontFeatures("tnum").TextColor(t.TextMuted)
		})
		files := a.visibleFiles()
		list := ui.Sidebar(c, &a.selected, func() {
			for _, file := range files {
				label := file.Name
				if file.Modified() {
					label += "  •"
				}
				ui.SidebarItem(c.Key(file.ID), file.ID, icons["file"], label).Height(40).Radius(7).FontSize(12)
			}
		}).Label("Documents")
		if a.folder == nil {
			list.Grow(1)
		} else {
			list.MaxHeight(180).Shrink(1)
		}
		list.Children(func() {
			if len(files) == 0 {
				ui.Text(c, "No matching documents").FontSize(12).TextColor(t.TextMuted).Padding(12, 10)
			}
		})
		if a.folder != nil {
			ui.Row(c).FillWidth().PaddingStart(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, a.folder.Name).FontSize(10).FontWeight(600).TextColor(t.TextMuted).SingleLine().Ellipsis("…").Grow(1)
				if iconButton(c, "refresh", "refresh", "Refresh folder").Disabled(a.fileBusy).Clicked() {
					a.refreshFolder()
				}
			})
			ui.Scroll(c).Grow(1).MinHeight(0).Children(func() { ui.Tree(c, func() { a.folderItems(c, a.folder.Children) }).Label("Folder contents") })
		}
		open := ui.ButtonBase(c.Key("open-folder")).FillWidth().Height(40).Padding(0, 10).Radius(7).Label("Open folder").Disabled(a.fileBusy)
		if open.Hovered() {
			open.Background(t.SurfaceHover)
		}
		open.Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons["folder"]).Size(17, 17).TextColor(t.TextMuted)
				ui.Text(c, "Open a folder").FontSize(12).TextColor(t.TextMuted)
			})
		})
		if open.Clicked() {
			a.openFolderDialog()
		}
	})
}
