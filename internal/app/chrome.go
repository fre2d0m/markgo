package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/xddcode/markgo/internal/mdeditor"
	"github.com/xddcode/markgo/internal/writingui"
	"time"
)

func (a *App) header(c *ui.Context) {
	t := c.Theme()
	file := a.current()
	ui.Row(c.Key("title-bar")).FillWidth().Height(titleBarHeight).DragWindow().AlignItems(ui.Center).Children(func() {
		if a.focusMode {
			ui.Text(c, file.Name).FontSize(13).TextColor(t.TextMuted).Grow(1).MinWidth(0).PaddingStart(28).SingleLine().Ellipsis("…")
			iconButton(c, "exit-focus", "exit-focus", "Exit focus mode").OnClick(func() { a.focusMode = false })
			a.captionButtons(c)
			return
		}
		ui.Row(c.Key("workspace")).Width(sidebarWidth).Shrink(0).FillHeight().Padding(0, 12).AlignItems(ui.Center).Children(func() {
			button := writingui.Interactive(c, ui.ButtonBase(c.Key("workspace-menu")), false).Height(44).Padding(0, 6).KeepFocus().Label("Document menu").Tooltip("Document menu")
			button.Children(func() { ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() { brand(c) }) })
			button.PopoverMenuButton(a.documentMenu, ui.PopoverMenuButtonOptions{Panel: func(panel ui.Element) { writingui.Surface(c, panel).Padding(8) }})
			ui.Spacer(c)
			iconButton(c, "sidebar", "sidebar", "Toggle file sidebar").OnClick(func() {
				a.sidebar = !a.sidebar
				if !a.sidebar && !a.reading {
					file.Focus.Focus(c)
				}
			})
		})
		services := c.Services()
		documentTabs(c, &a.tabs, a.files, a.selected, a.fileBusy, a.selectDocument, a.closeDocument, a.newDocument, func(menu *ui.Menu, file *mdeditor.File) { a.fileMenu(services, menu, file) })
		ui.Row(c.Key("window-tools")).Shrink(0).PaddingStart(8).Gap(2).AlignItems(ui.Center).Children(func() {
			iconButton(c, "find", "search", "Find in document").OnClick(func() { a.reading = false; file.State.Run(editor.Find) })
			iconButton(c, "focus", "focus", "Focus mode").OnClick(func() {
				a.focusMode = true
				if !a.reading {
					file.Focus.Focus(c)
				}
			})
			appearance := stateIconButton(c, "appearance", "appearance", "Appearance", a.appearanceOpen)
			if appearance.Clicked() {
				a.appearanceOpen = !a.appearanceOpen
				a.formatOpen = false
			}
			var writingFocus *ui.Handle
			if !a.reading {
				writingFocus = &file.Focus
			}
			a.appearancePopover(c, appearance, writingFocus)
		})
		a.captionButtons(c)
	})
}

func (a *App) documentToolbar(c *ui.Context, file *mdeditor.File) {
	t := c.Theme()
	ui.Row(c.Key("document-toolbar")).FillWidth().Height(56).Padding(6, 16).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Row(c.Key("writing-mode")).Height(44).Padding(2).Gap(2).Radius(10).Background(t.Surface).Shrink(0).Children(func() {
			for _, mode := range []struct {
				label, name string
				reading     bool
			}{{"Edit document", "Write", false}, {"Read document", "Read", true}} {
				button := appearanceSegment(c.Key(mode.name), mode.label, a.reading == mode.reading).Width(62)
				button.Children(func() { ui.Text(c, mode.name).FontSize(12).FontWeight(500) })
				button.OnClick(func() {
					a.reading = mode.reading
					if !a.reading {
						file.Focus.Focus(c)
					}
				})
			}
		})
		if !a.reading {
			ui.Box(c).Size(1, 20).Background(t.Border)
			ui.Row(c.Key("quick-formatting")).Width(166).Shrink(0).Children(func() { a.editorToolbar(c, file) })
		}
		ui.Spacer(c)
		format := stateIconButton(c, "format", "more", "Formatting", a.formatOpen)
		if format.Clicked() {
			a.formatOpen = !a.formatOpen
			a.appearanceOpen = false
			a.reading = false
		}
		ui.PopoverBase(c, format, &a.formatOpen, func(panel ui.Element) {
			writingui.Surface(c, panel).Width(248).Padding(8).Margin(8, 0, 0).Children(func() { a.blockFormatting(c, file) })
		})
	})
	ui.Box(c).FillWidth().Height(1).MarginX(16).Background(t.Border.Alpha(0.55))
}

func (a *App) documentMenu(m *ui.Menu) {
	if writingui.MenuItem(m, "New document").Icon(menuIcons["plus"]).Shortcut(ui.Cmd, ui.KeyN).Disabled(a.fileBusy).Chosen() {
		a.newDocument()
	}
	if writingui.MenuItem(m, "Open document…").Icon(menuIcons["file"]).Shortcut(ui.Cmd, ui.KeyO).Disabled(a.fileBusy).Chosen() {
		a.openDialog()
	}
	if writingui.MenuItem(m, "Recent documents…").Chosen() {
		a.recentOpen = true
		a.recentQuery = ""
	}
	if writingui.MenuItem(m, "Open folder…").Icon(menuIcons["folder"]).Disabled(a.fileBusy).Chosen() {
		a.openFolderDialog()
	}
	m.Separator()
	if writingui.MenuItem(m, "Save").Icon(menuIcons["save"]).Shortcut(ui.Cmd, ui.KeyS).Disabled(a.fileBusy).Chosen() {
		a.saveDocument(false)
	}
	if writingui.MenuItem(m, "Save as…").Shortcut(ui.Cmd|ui.Shift, ui.KeyS).Disabled(a.fileBusy).Chosen() {
		a.saveDocument(true)
	}
	if writingui.MenuItem(m, "Close document").Shortcut(ui.Cmd, ui.KeyW).Disabled(a.fileBusy).Chosen() {
		a.closeDocument(a.current())
	}
	m.Separator()
	if writingui.MenuItem(m, "Show Markdown source").Checked(!a.preview).Chosen() {
		a.preview = !a.preview
		a.reading = false
	}
	m.Separator()
	if writingui.MenuItem(m, "Writing guide").Chosen() {
		a.openGuides()
	}
}

func (a *App) fileSidebar(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(sidebarWidth).Shrink(0).FillHeight().Padding(12, 12, 12).Gap(12).Background(t.Surface).Children(func() {
		ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
			tabs := ui.Row(c.Key("navigation-tabs")).Width(128).Height(40)
			x := tabs.AnimateTo("indicator", float32(a.sideView)*64+24, ui.Animation{Duration: 160 * time.Millisecond, Ease: ui.EaseOut})
			tabs.Children(func() {
				for index, label := range []string{"Files", "Outline"} {
					button := writingui.Interactive(c, ui.ButtonBase(c.Key(label)), false).Size(64, 40).Center().Label(label)
					color := t.TextMuted
					if button.Hovered() {
						button.Background(t.SurfaceHover)
						color = t.Text
					}
					if a.sideView == index {
						color = t.Text
					}
					button.Children(func() {
						ui.Text(c, label).FontSize(12).FontWeight(500).TextColor(color)
					})
					if button.Clicked() {
						a.sideView = index
					}
				}
				ui.Box(c.Key("tab-indicator")).Absolute().Left(x).Bottom(0).Size(16, 2).Radius(1).Background(t.Text).PassThrough().Label("Sidebar tab indicator")
			})
			ui.Spacer(c)
			if iconButton(c, "new", "plus", "New document").Disabled(a.fileBusy).Clicked() {
				a.newDocument()
			}
		})
		ui.Column(c.Key(a.sideView)).Grow(1).MinHeight(0).FillWidth().Gap(12).
			Transition(ui.ElementTransition{Duration: 120 * time.Millisecond, Colors: true, Enter: &ui.Motion{Y: 4, Opacity: 0}, ExitDuration: -1}).Children(func() {
			if a.sideView == 1 {
				a.outlineSidebar(c, a.current())
				return
			}
			a.documentList(c)
		})
	})
}

func (a *App) documentList(c *ui.Context) {
	t := c.Theme()
	services := c.Services()
	ui.SearchField(c.Key("filter"), &a.query).Height(40).Background(t.Background).Border(1, t.Border).Radius(8).FontSize(13).Placeholder("Find a document").Label("Filter documents")
	ui.Row(c).FillWidth().Padding(2, 10).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Open documents").FontSize(11).FontWeight(500).TextColor(t.TextMuted).Grow(1)
		ui.Textf(c, "%d", len(a.files)).FontSize(11).FontFeatures("tnum").TextColor(t.TextMuted)
	})
	files := a.visibleFiles()
	// Preserve the Sidebar's keyboard model with a quieter selection color.
	listTheme := *t
	listTheme.AccentText = t.Text
	listTheme.SurfacePressed = t.Background
	listScope := ui.Themed(c, &listTheme).Background(t.Surface).MinHeight(0)
	if a.folder == nil {
		listScope.Grow(1)
	} else {
		listScope.MaxHeight(180).Shrink(1)
	}
	listScope.Children(func() {
		list := ui.Sidebar(c, &a.selected, func() {
			for _, file := range files {
				label := file.Name
				if file.Modified() {
					label += "  •"
				}
				row := ui.SidebarItem(c.Key(file.ID), file.ID, icons["file"], label).Height(40).PaddingEnd(0).Radius(8).FontSize(13).Label("Document " + file.Name)
				path := file.Path
				if path == "" {
					path = file.Name
				}
				if file.ID == a.selected {
					row.Background(t.Background).TextColor(t.Text).Border(1, t.Border)
				}
				row.Tooltip(path).ContextMenu(func(m *ui.Menu) { a.fileMenu(services, m, file) })
				showClose := row.Hovered() || file.ID == a.selected || c.Preferences().NoHover
				row.Children(func() {
					ui.Box(c).Size(40, 40).Children(func() {
						if showClose {
							close := iconButton(c, "close-file", "close", "Close "+file.Name).Disabled(a.fileBusy)
							close.Transition(ui.ElementTransition{Duration: 100 * time.Millisecond, Colors: true, Enter: &ui.Motion{Opacity: 0}, Exit: &ui.Motion{Opacity: 0}})
							close.OnClick(func() { a.closeDocument(file) })
						}
					})
				})
			}
		}).Label("Documents").Bind(&a.filesFocus).Grow(1).MinHeight(0).Padding(0)
		a.filesFocus.OnShortcut(c, 0, ui.KeyEnter, func() { a.reading = false; a.current().Focus.Focus(c) })
		list.Children(func() {
			if len(files) == 0 {
				ui.Text(c, "No matching documents").FontSize(12).TextColor(t.TextMuted).Padding(12, 10)
			}
		})
	})
	if a.folder != nil {
		ui.Row(c).FillWidth().PaddingStart(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.folder.Name).FontSize(11).FontWeight(500).TextColor(t.TextMuted).SingleLine().Ellipsis("…").Grow(1)
			if iconButton(c, "refresh", "refresh", "Refresh folder").Disabled(a.fileBusy).Clicked() {
				a.refreshFolder()
			}
		})
		ui.Scroll(c).Grow(1).MinHeight(0).Children(func() { ui.Tree(c, func() { a.folderItems(c, a.folder.Children) }).Label("Folder contents") })
	}
	open := writingui.Interactive(c, ui.ButtonBase(c.Key("open-folder")), false).FillWidth().Height(40).Padding(0, 10).Label("Open folder").Disabled(a.fileBusy)
	open.Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icons["folder"]).Size(18, 18)
			ui.Text(c, "Open a folder").FontSize(13)
		})
	})
	if open.Clicked() {
		a.openFolderDialog()
	}
}

func (a *App) fileMenu(services ui.Services, m *ui.Menu, file *mdeditor.File) {
	if writingui.MenuItem(m, "Save document").Disabled(a.fileBusy).Chosen() {
		a.saveFile(file, false, nil)
	}
	if writingui.MenuItem(m, "Copy file path").Disabled(file.Path == "").Chosen() {
		services.WriteClipboard(file.Path)
	}
	m.Separator()
	if writingui.MenuItem(m, "Close document").Disabled(a.fileBusy).Chosen() {
		a.closeDocument(file)
	}
}
