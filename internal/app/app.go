package app

import (
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/xddcode/markgo/internal/mdeditor"
	"github.com/xddcode/markgo/internal/preferences"
	"github.com/xddcode/markgo/internal/storage"
	"github.com/xddcode/markgo/internal/writingui"
)

type App struct {
	workspaceSession
	files              []*mdeditor.File
	selected           string
	query              string
	preview            bool
	reading            bool
	focusMode          bool
	sidebar            bool
	filesFocus         ui.Handle
	tabs               documentTabState
	window             *mygo.Window
	windowCommands     windowCommands
	windowMaximized    bool
	disk               map[string]storage.Snapshot
	nextID             int
	fileBusy           bool
	saving             bool
	savingFileID       string
	savedEvents        map[string]uint64
	notice             string
	closing            bool
	allowExit          bool
	sideView           int
	folder             *storage.Entry
	expanded           map[string]*bool
	analyses           map[string]*analysisCache
	conflicts          map[string]bool
	checking           map[string]bool
	openQueue          []string
	appearance         appearance
	appearanceOpen     bool
	appearanceWasOpen  bool
	appearancePanel    ui.Handle
	appearanceFirst    ui.Handle
	formatOpen         bool
	appearanceRevision uint64
	preferenceStore    *preferences.Store
	systemDark         bool
	cachedAppearance   appearance
	cachedPalette      string
	editorTheme        *editor.Theme
	readerTheme        *markdown.Theme
	preferenceWorkers  sync.WaitGroup
}

func New() *App {
	file := mdeditor.NewFile("Untitled 1.md", "")
	file.ID = "document-1"
	return &App{
		workspaceSession: workspaceSession{backedUp: make(map[string]uint64), recovered: make(map[string]string), recentMissing: make(map[string]string)},
		files:            []*mdeditor.File{file},
		selected:         file.ID,
		nextID:           1,
		preview:          true,
		appearance:       defaultAppearance(),
		sidebar:          true,
		disk:             make(map[string]storage.Snapshot),
		expanded:         make(map[string]*bool),
		analyses:         make(map[string]*analysisCache),
		conflicts:        make(map[string]bool),
		checking:         make(map[string]bool),
		savedEvents:      make(map[string]uint64),
		notice:           "",
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
	a.systemDark = c.Theme().Dark
	c.SetTheme(writingTheme(c.Theme(), a.appearance.Theme))
	c.SetMenuPresentation(ui.PopoverMenus)
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
	if c.Shortcut(ui.Ctrl, ui.KeyTab) {
		a.cycleDocument(1)
	}
	if c.Shortcut(ui.Ctrl|ui.Shift, ui.KeyTab) {
		a.cycleDocument(-1)
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyF) {
		a.focusMode = !a.focusMode
		if a.focusMode && !a.reading {
			a.current().Focus.Focus(c)
		}
	}
	if c.Shortcut(ui.Cmd, ui.KeyComma) {
		a.focusMode = false
		a.appearanceOpen = !a.appearanceOpen
		a.formatOpen = false
	}
	if c.Shortcut(ui.Cmd, ui.KeyBackslash) {
		a.sidebar = !a.sidebar
		if !a.sidebar && !a.reading {
			a.current().Focus.Focus(c)
		}
	}
	root := ui.Column(c.Key("workbench")).Fill().Background(c.Theme().Surface).Disabled(a.closing && !a.exitPrompt)
	showSidebar := a.sidebar && !a.focusMode
	target := float32(0)
	if showSidebar {
		target = sidebarWidth
	}
	width := root.AnimateTo("sidebar-width", target, ui.Animation{Duration: 180 * time.Millisecond, Ease: ui.EaseOut})
	root.Children(func() {
		a.header(c)
		a.sessionWarning(c)
		ui.Row(c.Key("workspace-body")).Grow(1).FillWidth().AlignItems(ui.Stretch).Children(func() {
			if width > 0.5 {
				ui.Column(c.Key("sidebar-slot")).Width(width).Shrink(0).FillHeight().Clip().Disabled(!showSidebar).Label("File sidebar").Children(func() { a.fileSidebar(c) })
			}
			a.editorPane(c.Key("document-pane"), a.current())
		})
		if !a.focusMode {
			a.statusBar(c, a.current())
		}
	})
	a.recentDialog(c)
	a.exitDialog(c)
	if a.returnToWriting && !a.closing {
		a.current().Focus.Focus(c)
		a.returnToWriting = false
	}
	a.scheduleRecovery(c)
}

func (a *App) editorPane(c *ui.Context, f *mdeditor.File) {
	t := c.Theme()
	st := f.State
	st.RenderFindBar = renderFindBar
	f.Focus.OnShortcut(c, ui.Cmd, ui.KeyB, func() { a.format(f, func() bool { return mdeditor.ToggleInline(st, "**") }) })
	f.Focus.OnShortcut(c, ui.Cmd, ui.KeyI, func() { a.format(f, func() bool { return mdeditor.ToggleInline(st, "*") }) })
	f.Focus.OnShortcut(c, ui.Cmd, ui.Key1, func() { a.format(f, func() bool { return mdeditor.SetHeading(st, 1) }) })
	f.Focus.OnShortcut(c, ui.Cmd, ui.Key2, func() { a.format(f, func() bool { return mdeditor.SetHeading(st, 2) }) })
	if a.appearanceOpen || a.formatOpen {
		// The writing caret stays focused while these controls are used.
		// Consume Escape before the editor collapses the user's selection.
		f.Focus.OnShortcut(c, 0, ui.KeyEscape, func() {
			a.appearanceOpen, a.formatOpen = false, false
		})
	}
	st.SoftWrap = true
	st.Font.Size = a.appearance.FontSize
	st.Font.Weight = 400
	st.Font.Family = "system-ui"
	if a.appearance.Serif {
		st.Font.Family = "serif"
	}
	st.LineHeight = 1.85
	a.applyTypography(f.Reader, t)
	st.Theme = a.editorTheme
	analysis, ready := a.analysisFor(f)
	if a.preview {
		if ready {
			st.SetReplacements(analysis.Replacements(st))
		}
	} else {
		st.SetReplacements(nil)
	}
	pane := ui.Column(c).Grow(1).MinWidth(0).FillHeight()
	if a.focusMode {
		pane.Background(t.Background)
	} else {
		pageSurface(c, pane)
	}
	pane.Children(func() {
		if !a.focusMode {
			a.documentToolbar(c, f)
		}
		a.recoveryNotice(c, f)
		if a.conflicts[f.ID] && a.recovered[f.ID] == "" {
			ui.Row(c).FillWidth().Padding(8, 18).Gap(10).AlignItems(ui.Center).Background(t.Warning.Alpha(0.12)).Children(func() {
				ui.Text(c, "Changed outside MarkGo").FontSize(12).Grow(1)
				if writingui.Button(c, "Reload", false).Label("Reload changed document").Disabled(a.fileBusy).Clicked() {
					a.reloadDocument(f)
				}
				if writingui.Button(c, "Keep editing", false).Label("Keep editing changed document").Clicked() {
					delete(a.conflicts, f.ID)
				}
			})
		}
		if a.reading {
			if !ready {
				ui.Text(c, "Preparing document…").FontSize(13).TextColor(t.TextMuted).Padding(38, 48)
				return
			}
			ui.Scroll(c).Grow(1).FillWidth().MinHeight(0).Padding(38, 48).Children(func() {
				ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
					ui.Column(c).Grow(1).MaxWidth(a.appearance.Width).MinWidth(0).Children(func() {
						markdown.View(c.Key(f.ID), analysis.Source(), f.Reader).FillWidth()
					})
				})
			})
		} else {
			ui.Row(c).Grow(1).FillWidth().MinHeight(0).Padding(38, 48).Justify(ui.Center).AlignItems(ui.Stretch).Children(func() {
				view := editor.View(c.Key(f.ID), st).Bind(&f.Focus).Grow(1).MaxWidth(a.appearance.Width).MinWidth(0).FillHeight()
				// Arrow navigation previews documents without moving focus out of
				// the file list. Enter in that list explicitly starts writing.
				if (!a.filesFocus.FocusWithin(c) && !a.tabs.focus.FocusWithin(c)) || a.focusMode {
					view.AutoFocus()
				}
			})
		}
	})
}

func (a *App) statusBar(c *ui.Context, f *mdeditor.File) {
	t := c.Theme()
	pos := f.State.CursorPos()
	analysis, _ := a.analysisFor(f)
	state := "Saved locally"
	if f.Path == "" {
		state = "Not saved yet"
	} else if f.Modified() {
		state = "Unsaved changes"
	}
	ui.Row(c).FillWidth().Height(30).Padding(0, 22).Gap(18).AlignItems(ui.Center).Children(func() {
		a.saveStatus(c, f, state)
		notice := a.notice
		if a.saving && a.savingFileID == f.ID && notice == "Saving…" {
			notice = ""
		}
		ui.Text(c, notice).FontSize(11).TextColor(t.TextMuted).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(notice).Role(ui.RoleStatus)
		if !a.reading {
			ui.Textf(c, "%d : %d", pos.Line+1, mdeditor.DisplayColumn(f.State)).MinWidth(64).TextAlign(ui.End).FontSize(11).FontFeatures("tnum").TextColor(t.TextMuted).Tooltip("Line and column")
		}
		ui.Textf(c, "%d words", analysis.Stats.Words).MinWidth(76).TextAlign(ui.End).FontSize(11).FontFeatures("tnum").TextColor(t.TextMuted)
		ui.Textf(c, "%d characters", analysis.Stats.Characters).MinWidth(104).TextAlign(ui.End).FontSize(11).FontFeatures("tnum").TextColor(t.TextMuted)
	})
}
