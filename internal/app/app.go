package app

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/xddcode/markgo/internal/mdeditor"
	"github.com/xddcode/markgo/internal/storage"
)

type App struct {
	files     []*mdeditor.File
	selected  string
	query     string
	preview   bool
	reading   bool
	focusMode bool
	sidebar   bool
	window    *mygo.Window
	disk      map[string]storage.Snapshot
	nextID    int
	fileBusy  bool
	saving    bool
	notice    string
	closing   bool
	allowExit bool
	sideView  int
	folder    *storage.Entry
	expanded  map[string]*bool
	analyses  map[string]*analysisCache
	conflicts map[string]bool
	checking  map[string]bool
	openQueue []string
}

func New() *App {
	files := sampleDocuments()
	return &App{
		files:     files,
		selected:  files[0].Name,
		preview:   true,
		sidebar:   true,
		disk:      make(map[string]storage.Snapshot),
		expanded:  make(map[string]*bool),
		analyses:  make(map[string]*analysisCache),
		conflicts: make(map[string]bool),
		checking:  make(map[string]bool),
		notice:    "Example documents are editable. Save a copy to keep your changes.",
	}
}

func (a *App) current() *mdeditor.File {
	for _, f := range a.files {
		if f.ID == a.selected {
			return f
		}
	}
	return a.files[0]
}

func (a *App) visibleFiles() []*mdeditor.File {
	query := strings.ToLower(strings.TrimSpace(a.query))
	if query == "" {
		return a.files
	}
	var files []*mdeditor.File
	for _, f := range a.files {
		if strings.Contains(strings.ToLower(f.Name), query) {
			files = append(files, f)
		}
	}
	return files
}

func (a *App) View(c *ui.Context) {
	c.SetTheme(markgoTheme(c.Theme()))
	a.processOpenQueue()
	if c.Shortcut(ui.Cmd, ui.KeyN) && !a.fileBusy {
		a.newDocument()
	}
	if c.Shortcut(ui.Cmd, ui.KeyO) && !a.fileBusy {
		a.openDialog()
	}
	if c.Shortcut(ui.Cmd, ui.KeyS) && !a.fileBusy {
		a.saveDocument(false)
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyS) && !a.fileBusy {
		a.saveDocument(true)
	}
	if c.Shortcut(ui.Cmd, ui.KeyW) && !a.fileBusy {
		a.closeDocument(a.current())
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyF) {
		a.focusMode = !a.focusMode
	}
	ui.Column(c).Fill().Background(c.Theme().Background).Children(func() {
		a.header(c)
		ui.Row(c).Grow(1).FillWidth().AlignItems(ui.Stretch).Children(func() {
			if a.sidebar && !a.focusMode {
				a.fileSidebar(c)
			}
			a.editorPane(c, a.current())
		})
		if !a.focusMode {
			a.statusBar(c, a.current())
		}
	})
}

func markgoTheme(base *ui.Theme) *ui.Theme {
	t := *base
	t.Accent = ui.Hex("#6c5ce7")
	t.AccentHover = ui.Hex("#7c6cf0")
	t.AccentPressed = ui.Hex("#5948cf")
	t.AccentText = ui.Hex("#ffffff")
	t.Selection = t.Accent.Alpha(0.24)
	t.Focus = t.Accent.Alpha(0.58)
	t.Radius = 8
	return &t
}

func (a *App) header(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Height(56).Padding(0, 14).Gap(12).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
		if a.focusMode {
			ui.Text(c, a.current().Name).FontSize(13).TextColor(t.TextMuted)
			ui.Spacer(c)
			if ui.Button(c, "Exit focus").Label("Exit focus mode").KeepFocus().Clicked() {
				a.focusMode = false
			}
			return
		}
		if ui.Button(c, "Files").Label("Toggle file sidebar").KeepFocus().Clicked() {
			a.sidebar = !a.sidebar
		}
		ui.Row(c).Width(30).Height(30).Center().Radius(9).Background(t.Accent).Children(func() {
			ui.Text(c, "M").FontSize(16).Bold().TextColor(t.AccentText)
		})
		ui.Text(c, "MarkGo").FontSize(17).Bold()
		ui.Badge(c, "PREVIEW")
		ui.Spacer(c)
		if ui.Button(c, "New").Label("New document").KeepFocus().Disabled(a.fileBusy).Clicked() {
			a.newDocument()
		}
		if ui.Button(c, "Open").Label("Open document").KeepFocus().Disabled(a.fileBusy).Clicked() {
			a.openDialog()
		}
		if ui.Button(c, "Save").Label("Save document").KeepFocus().Disabled(a.fileBusy).Clicked() {
			a.saveDocument(false)
		}
		if ui.Button(c, "Save as").Label("Save document as").KeepFocus().Disabled(a.fileBusy).Clicked() {
			a.saveDocument(true)
		}
		previewAction := "Source"
		if !a.preview {
			previewAction = "Preview"
		}
		if ui.Button(c, previewAction).Label("Live preview").Tooltip("Switch between source and live preview").KeepFocus().Clicked() {
			a.preview = !a.preview
			a.reading = false
		}
		readAction := "Read"
		if a.reading {
			readAction = "Edit"
		}
		if ui.Button(c, readAction).Label("Read document").Tooltip("Switch between reading and editing").KeepFocus().Clicked() {
			a.reading = !a.reading
			if !a.reading {
				a.current().Focus.Focus(c)
			}
		}
		if ui.Button(c, "Focus").Label("Focus mode").Tooltip("Focus mode").KeepFocus().Clicked() {
			a.focusMode = true
		}
	})
}

func (a *App) fileSidebar(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(224).Shrink(0).FillHeight().Padding(14, 10).Gap(10).Background(t.Surface).BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "WORKSPACE").FontSize(11).Bold().TextColor(t.TextMuted).Grow(1)
			if ui.Button(c, "Folder").Label("Open folder").Disabled(a.fileBusy).Clicked() {
				a.openFolderDialog()
			}
		})
		ui.Segmented(c, &a.sideView, "Files", "Outline").Label("Sidebar view").FillWidth()
		if a.sideView == 1 {
			a.outlineSidebar(c, a.current())
			return
		}
		ui.SearchField(c, &a.query).Placeholder("Filter documents").Label("Filter documents")
		files := a.visibleFiles()
		ui.Text(c, "OPEN DOCUMENTS").FontSize(10).Bold().TextColor(t.TextMuted)
		list := ui.Sidebar(c, &a.selected, func() {
			for _, f := range files {
				label := f.Name
				if f.Modified() {
					label += "  •"
				}
				ui.SidebarItem(c.Key(f.ID), f.ID, nil, label)
			}
		}).Label("Documents")
		if a.folder == nil {
			list.Grow(1)
		} else {
			list.MaxHeight(180).Shrink(1)
		}
		list.Children(func() {
			if len(files) == 0 {
				ui.Text(c, "No matching documents").FontSize(12).TextColor(t.TextMuted).Padding(12, 8)
			}
		})
		if a.folder != nil {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Text(c, a.folder.Name).FontSize(10).Bold().TextColor(t.TextMuted).SingleLine().Ellipsis("…").Grow(1)
				refresh := ui.ButtonBase(c).Size(30, 30).Center().Label("Refresh folder").Tooltip("Refresh folder").Disabled(a.fileBusy)
				refresh.Children(func() { ui.Text(c, "↻").FontSize(16).TextColor(t.TextMuted) })
				if refresh.Clicked() {
					a.refreshFolder()
				}
			})
			ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
				ui.Tree(c, func() { a.folderItems(c, a.folder.Children) }).Label("Folder contents")
			})
		}
	})
}

func (a *App) editorPane(c *ui.Context, f *mdeditor.File) {
	t := c.Theme()
	st := f.State
	st.RenderFindBar = renderFindBar
	f.Focus.OnShortcut(c, ui.Cmd, ui.KeyB, func() { a.format(f, func() bool { return mdeditor.ToggleInline(st, "**") }) })
	f.Focus.OnShortcut(c, ui.Cmd, ui.KeyI, func() { a.format(f, func() bool { return mdeditor.ToggleInline(st, "*") }) })
	f.Focus.OnShortcut(c, ui.Cmd, ui.Key1, func() { a.format(f, func() bool { return mdeditor.SetHeading(st, 1) }) })
	f.Focus.OnShortcut(c, ui.Cmd, ui.Key2, func() { a.format(f, func() bool { return mdeditor.SetHeading(st, 2) }) })
	st.SoftWrap = true
	textTheme := editor.DefaultTheme(t)
	textTheme.Tokens[editor.TokenHeading].Color = t.Text
	textTheme.Tokens[editor.TokenStrong].Color = t.Text
	textTheme.Tokens[editor.TokenEmphasis].Color = t.Text
	textTheme.Tokens[editor.TokenLink].Color = t.Accent
	st.Theme = textTheme
	analysis, ready := a.analysisFor(f)
	if a.preview {
		if ready {
			st.SetReplacements(analysis.Replacements(st))
		}
	} else {
		st.SetReplacements(nil)
	}
	ui.Column(c).Grow(1).MinWidth(0).FillHeight().Children(func() {
		ui.Row(c).Height(44).Padding(0, 22).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, f.Name).FontSize(13).Bold().SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
			if f.Modified() {
				ui.Text(c, "Edited").FontSize(11).TextColor(t.Accent)
			}
			ui.Spacer(c)
			ui.Text(c, "Markdown").FontSize(11).TextColor(t.TextMuted)
			closeButton := ui.ButtonBase(c).Size(30, 30).Center().Label("Close document").Tooltip("Close document").Disabled(a.fileBusy)
			closeButton.Children(func() { ui.Text(c, "×").FontSize(18).TextColor(t.TextMuted) })
			if closeButton.Clicked() {
				a.closeDocument(f)
			}
		})
		if a.conflicts[f.ID] {
			ui.Row(c).FillWidth().Padding(8, 18).Gap(10).AlignItems(ui.Center).Background(t.Warning.Alpha(0.12)).Children(func() {
				ui.Text(c, "Changed outside MarkGo").FontSize(12).Grow(1)
				if ui.Button(c, "Reload").Label("Reload changed document").Disabled(a.fileBusy).Clicked() {
					a.reloadDocument(f)
				}
				if ui.Button(c, "Keep editing").Label("Keep editing changed document").Clicked() {
					delete(a.conflicts, f.ID)
				}
			})
		}
		if a.reading {
			if !ready {
				ui.Text(c, "Preparing document…").FontSize(13).TextColor(t.TextMuted).Padding(24, 28)
				return
			}
			ui.Scroll(c).Grow(1).FillWidth().MinHeight(0).Padding(24, 28).Children(func() {
				ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
					ui.Column(c).Grow(1).MaxWidth(760).MinWidth(0).Children(func() {
						markdown.View(c.Key(f.ID), analysis.Source(), f.Reader).FillWidth()
					})
				})
			})
		} else {
			if !a.focusMode {
				a.editorToolbar(c, f)
			}
			ui.Row(c).Grow(1).FillWidth().MinHeight(0).Padding(16, 28).Justify(ui.Center).AlignItems(ui.Stretch).Children(func() {
				editor.View(c.Key(f.ID), st).Bind(&f.Focus).Grow(1).MaxWidth(760).MinWidth(0).FillHeight().AutoFocus()
			})
		}
	})
}

func (a *App) statusBar(c *ui.Context, f *mdeditor.File) {
	t := c.Theme()
	pos := f.State.CursorPos()
	analysis, _ := a.analysisFor(f)
	ui.Row(c).Height(28).Padding(0, 14).Gap(16).AlignItems(ui.Center).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Textf(c, "Ln %d, Col %d", pos.Line+1, mdeditor.DisplayColumn(f.State)).FontSize(11).TextColor(t.TextMuted)
		ui.Textf(c, "%d words", analysis.Stats.Words).FontSize(11).TextColor(t.TextMuted)
		ui.Textf(c, "%d characters", analysis.Stats.Characters).FontSize(11).TextColor(t.TextMuted)
		ui.Textf(c, "%d lines", f.State.Buffer.LineCount()).FontSize(11).TextColor(t.TextMuted)
		ui.Spacer(c)
		ui.Text(c, a.notice).FontSize(11).TextColor(t.TextMuted).MaxWidth(450).SingleLine().Ellipsis("…").Tooltip(a.notice).Role(ui.RoleStatus)
	})
}
