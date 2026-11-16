package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/xddcode/markgo/internal/mdeditor"
	"github.com/xddcode/markgo/internal/writingui"
)

func (a *App) writer(file *mdeditor.File) *mdeditor.Typesetter {
	w := a.writers[file.ID]
	if w == nil || w.Source != file.State {
		w = mdeditor.NewTypesetter(file.State)
		a.writers[file.ID] = w
	}
	return w
}

func (a *App) syncWriting() {
	active := ""
	if a.preview && !a.reading {
		active = a.selected
	}
	if a.activeWriter != "" && a.activeWriter != active {
		if w := a.writers[a.activeWriter]; w != nil {
			w.Suspend()
		}
		a.dismissVisualFind()
	}
	a.activeWriter = active
	if active != "" {
		a.writer(a.current()).SyncSelection()
	}
}

func (a *App) flushWriting(file *mdeditor.File) {
	if w := a.writers[file.ID]; w != nil && a.activeWriter == file.ID {
		w.ToSource()
	}
}

// Popover menus and the file list take focus on pointer-down, before their
// chosen action runs. End preedit before MyGo releases the native text client,
// so its focus-loss UnmarkText cannot save an unconfirmed candidate.
func (a *App) endCompositionBeforePointer(ev ui.InputEvent) {
	if ev.Kind == ui.InputPointerDown {
		if w := a.writers[a.activeWriter]; w != nil {
			w.CancelComposition()
		}
	}
}

func (a *App) setWritingMode(visual, reading bool) {
	if a.preview == visual && a.reading == reading {
		return
	}
	a.flushWriting(a.current())
	a.preview, a.reading = visual, reading
	a.formatOpen, a.appearanceOpen = false, false
	a.dismissVisualFind()
	a.syncWriting()
	if !reading {
		a.current().Focus.Focus()
	}
}

func (a *App) viewMenu(m *ui.Menu) {
	if writingui.MenuItem(m, "Visual editing").Checked(a.preview && !a.reading).Chosen() {
		a.setWritingMode(true, false)
	}
	if writingui.MenuItem(m, "Markdown source").Checked(!a.preview && !a.reading).Shortcut(ui.Cmd|ui.Shift, ui.KeyM).Chosen() {
		a.setWritingMode(false, false)
	}
	if writingui.MenuItem(m, "Reading view").Checked(a.reading).Chosen() {
		a.setWritingMode(a.preview, true)
	}
}

// Hidden editors retain the toolkit's private layout/scroll state. Only
// previously visited documents are built, and suspended views do not parse,
// create overlays, draw, take focus or participate in accessibility.
func (a *App) visualEditors(c *ui.Context, current *mdeditor.File) {
	for _, file := range a.files {
		w := a.writers[file.ID]
		if w == nil {
			continue
		}
		active := file == current && a.preview && !a.reading
		if active {
			start, end := -1, -1
			if a.find.open && len(a.find.matches) > 0 {
				match := a.find.matches[a.find.current]
				start, end = match[0], match[1]
			}
			w.SetSearchMatch(start, end)
		}
		font := ui.Font{Family: "system-ui", Size: a.appearance.FontSize}
		if a.appearance.Serif {
			font.Family = "serif"
		}
		view := mdeditor.TypesettingView(c.Key("visual-"+file.ID), w, font, !active).
			Grow(1).MaxWidth(a.appearance.Width).MinWidth(0).FillHeight().Label(file.Name)
		if active {
			view.Bind(&file.Focus)
			if !a.filesFocus.FocusWithin(c) && !a.tabs.focus.FocusWithin(c) && !a.find.open {
				view.AutoFocus()
			}
		}
	}
}
