package app

import (
	"path/filepath"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/xddcode/markgo/internal/mdeditor"
	"github.com/xddcode/markgo/internal/writingui"
	"github.com/xddcode/markgo/resources"
)

var brandImage = func() ui.ImageSource {
	image, err := ui.DecodeImage(resources.AppIconPNG)
	if err != nil {
		panic(err)
	}
	return image
}()

const documentTabWidth = 224

// Only navigation state lives here. Every tab still owns the same File,
// source buffer, selection, scroll position and undo history as the sidebar.
type documentTabState struct {
	focus       ui.Handle
	scroll      ui.ScrollState
	revealed    string
	count       int
	windowWidth float32
}

func brand(c *ui.Context) {
	ui.Image(c, brandImage).Size(28, 28).Radius(8)
	ui.Text(c, "MarkGo").FontSize(15).FontWeight(650).LetterSpacing(-0.35).TextColor(c.Theme().Text)
}

func documentTabs(c *ui.Context, state *documentTabState, files []*mdeditor.File, selected string, busy bool, choose func(int, bool), closeFile func(*mdeditor.File), create func(), contextMenu func(*ui.Menu, *mdeditor.File)) {
	t := c.Theme()
	index := 0
	for i, file := range files {
		if file.ID == selected {
			index = i
			break
		}
	}
	windowWidth, _ := c.Size()
	reveal := state.revealed != selected || state.count != len(files) || windowWidth != state.windowWidth
	ui.Row(c.Key("document-tabs")).Grow(1).MinWidth(0).FillHeight().Gap(4).AlignItems(ui.End).Children(func() {
		ui.ScrollHorizontal(c.Key("tab-scroll")).Width(float32(len(files)*(documentTabWidth+4) - 4)).Shrink(1).MinWidth(100).Height(titleBarHeight).TrackScroll(&state.scroll).Label("Document tab strip").Children(func() {
			tabs := ui.TabsBase(c.Key("tab-list"), &index, len(files))
			tabs.List.Bind(&state.focus).Height(titleBarHeight).Padding(12, 0, 0).Gap(4).OnChange(func() { choose(index, false) }).Children(func() {
				for i, file := range files {
					// A keyed parent keeps the native tab identity with its file
					// when another document is inserted or closed before it.
					ui.Row(c.Key(file.ID)).Shrink(0).Children(func() {
						active := file.ID == selected
						tab := tabs.Tab(i).Width(documentTabWidth).Height(48).Padding(0, 4, 0, 14).Gap(9).AlignItems(ui.Center).Radius(12, 12, 0, 0).Label("Tab " + file.Name)
						path := file.Path
						if path == "" {
							path = file.Name
						}
						status := "Saved"
						if file.Path == "" {
							status = "Not saved yet"
						}
						if file.Modified() {
							status = "Unsaved changes"
						}
						tab.Tooltip(path + " · " + status).Description(path + " · " + status)
						if contextMenu != nil {
							tab.ContextMenu(func(menu *ui.Menu) { contextMenu(menu, file) })
						}
						fill, ink := ui.Transparent, t.TextMuted
						if tab.Hovered() {
							fill, ink = t.SurfaceHover, t.Text
						}
						if active {
							fill, ink = t.Background, t.Text
						}
						if tab.Pressed() {
							fill = t.SurfacePressed
						}
						tab.Background(tab.AnimateColor("tab-fill", fill, ui.Animation{Duration: 110 * time.Millisecond, NonMotion: true})).TextColor(ink)
						if active && reveal {
							tab.ScrollIntoView()
						}
						tab.OnClick(func() { choose(i, true) })
						tab.Children(func() {
							ui.Box(c).Size(16, 16).Shrink(0).Children(func() {
								icon := ui.Icon(c, icons["file"]).Size(16, 16)
								if active {
									icon.TextColor(t.Accent)
								}
								if file.Modified() && c.Preferences().NoHover {
									ui.Box(c).Absolute().Top(-2).Right(-3).Size(5, 5).Radius(3).Background(t.Accent).Label("Unsaved tab " + file.Name)
								}
							})
							ui.Text(c, file.Name).FontSize(13).FontWeight(500).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
							close := writingui.Interactive(c, ui.ButtonBase(c.Key("close-"+file.ID)), false).Size(40, 40).Shrink(0).KeepFocus().Center().Disabled(busy).Label("Close tab " + file.Name).Tooltip("Close " + file.Name + " · Cmd/Ctrl+W")
							close.Children(func() {
								if file.Modified() && !close.Hovered() && !c.Preferences().NoHover {
									ui.Box(c).Size(6, 6).Radius(3).Background(t.Accent).Label("Unsaved tab " + file.Name)
								} else {
									ui.Icon(c, icons["close"]).Size(14, 14).Opacity(func() float32 {
										if active || tab.Hovered() || c.Preferences().NoHover {
											return 1
										}
										return 0.45
									}())
								}
							})
							close.OnClick(func() { closeFile(file) })
						})
					})
				}
			})
			state.focus.OnShortcut(c, 0, ui.KeyEnter, func() { choose(index, true) })
		})
		ui.Row(c.Key("tab-actions")).Height(48).Shrink(0).AlignItems(ui.Center).Children(func() {
			iconButton(c, "new-tab", "plus", "New document tab").Disabled(busy).OnClick(create)
			all := iconButton(c, "all-tabs", "chevron", "All open documents").Tooltip("All open documents · Ctrl+Tab to switch")
			all.PopoverMenuButton(func(menu *ui.Menu) {
				for i, file := range files {
					label := file.Name
					for _, other := range files {
						if other.ID != file.ID && other.Name == file.Name && file.Path != "" {
							label += " — " + filepath.Dir(file.Path)
							break
						}
					}
					if file.Modified() {
						label += "  •"
					}
					if writingui.MenuItem(menu, label).ID(file.ID).Checked(file.ID == selected).Chosen() {
						choose(i, true)
					}
				}
			}, ui.PopoverMenuButtonOptions{Panel: func(panel ui.Element) { writingui.Surface(c, panel).Padding(8).MaxWidth(440).MaxHeight(480) }})
		})
		ui.Spacer(c)
	})
	state.revealed, state.count = selected, len(files)
	state.windowWidth = windowWidth
}

func (a *App) selectDocument(index int, writing bool) {
	if index < 0 || index >= len(a.files) {
		return
	}
	file := a.files[index]
	if file.ID != a.selected {
		a.selected = file.ID
		a.appearanceOpen, a.formatOpen = false, false
		a.notice = ""
	}
	if writing && !a.reading {
		file.Focus.Focus()
	}
}

func (a *App) cycleDocument(direction int) {
	for i, file := range a.files {
		if file.ID == a.selected {
			a.selectDocument((i+direction+len(a.files))%len(a.files), true)
			return
		}
	}
}

func pageSurface(c *ui.Context, panel ui.Element) ui.Element {
	t := c.Theme()
	return panel.Background(t.Background).Radius(0, 16, 16, 16).MarginEnd(12).Border(1, t.Border.Alpha(0.55))
}
