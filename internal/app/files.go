package app

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui/editor"
	"github.com/xddcode/markgo/internal/mdeditor"
	"github.com/xddcode/markgo/internal/storage"
)

// Attach connects native dialogs and close protection after the window exists.
func (a *App) Attach(window *mygo.Window) {
	a.window = window
	window.OnFileDrop(func(event *mygo.FileDropEvent) {
		for _, path := range event.Paths {
			a.OpenFile(path)
		}
	})
	window.OnFocus(func() { a.checkExternalChange(a.current()) })
	window.OnClose(func(event *mygo.CloseEvent) {
		if a.allowExit {
			return
		}
		if a.saving {
			event.PreventDefault()
			a.notice = "Please wait for the file operation to finish."
			window.Invalidate()
			return
		}
		if a.hasUnsavedChanges() {
			event.PreventDefault()
			a.confirmExit(false)
		}
	})
	mygo.App.OnBeforeQuit(func(event *mygo.QuitEvent) {
		if event.SessionEnding || a.allowExit {
			return
		}
		if a.saving {
			event.PreventDefault()
			a.notice = "Please wait for the file operation to finish."
			window.Invalidate()
			return
		}
		if a.hasUnsavedChanges() {
			event.PreventDefault()
			a.confirmExit(true)
		}
	})
}

// OpenFile queues a native file-association or file-drop request on the app
// thread. Requests arriving before the window or during a save are retained.
func (a *App) OpenFile(path string) {
	if path == "" {
		return
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		a.notice = "Could not open document: " + err.Error()
		return
	}
	for _, queued := range a.openQueue {
		if queued == absolute {
			return
		}
	}
	a.openQueue = append(a.openQueue, absolute)
	if a.window != nil {
		a.window.Invalidate()
	}
}

func (a *App) processOpenQueue() {
	if a.window == nil || a.closing || a.allowExit {
		return
	}
	for len(a.openQueue) > 0 && !a.fileBusy {
		path := a.openQueue[0]
		a.openQueue = a.openQueue[1:]
		a.openPath(path)
	}
}

func (a *App) hasUnsavedChanges() bool {
	for _, file := range a.files {
		if file.Modified() {
			return true
		}
	}
	return false
}

func (a *App) confirmExit(quit bool) {
	if a.closing || a.fileBusy || a.window == nil {
		return
	}
	a.closing = true
	a.fileBusy = true
	window := a.window
	go func() {
		result, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent: window, Type: mygo.MessageWarning,
			Title: "Unsaved changes", Message: "Some documents have unsaved changes.",
			Detail:  "Save all documents before closing, or discard their unsaved changes.",
			Buttons: []string{"Cancel", "Discard changes", "Save all"}, CancelButton: 0,
			DefaultButton: 2, DestructiveButtons: []int{1},
		})
		window.Update(func() {
			a.closing = false
			a.fileBusy = false
			if err != nil {
				a.notice = "Could not confirm closing: " + err.Error()
				return
			}
			if result.Button == 1 {
				a.finishExit(quit)
			} else if result.Button == 2 {
				a.saveAllAndExit(quit)
			}
		})
	}()
}

func (a *App) finishExit(quit bool) {
	a.allowExit = true
	if quit {
		mygo.App.Quit()
	} else {
		a.window.Close()
	}
}

// Save one document at a time. Cancellation, a failed save, or a newer edit
// stops the sequence and leaves the application open with its buffers intact.
func (a *App) saveAllAndExit(quit bool) {
	for _, file := range a.files {
		if file.Modified() {
			a.saveFile(file, false, func() { a.saveAllAndExit(quit) })
			return
		}
	}
	a.finishExit(quit)
}

func (a *App) newDocument() {
	a.reading, a.formatOpen, a.appearanceOpen = false, false, false
	a.query = ""
	a.nextID++
	file := mdeditor.NewFile(fmt.Sprintf("Untitled %d.md", a.nextID), "")
	file.ID = fmt.Sprintf("document-%d", a.nextID)
	a.files = append(a.files, file)
	a.selected = file.ID
	file.Focus.Focus()
	a.notice = ""
}

func (a *App) closeDocument(file *mdeditor.File) {
	if a.fileBusy {
		return
	}
	if !file.Modified() {
		a.removeDocument(file.ID)
		return
	}
	if a.window == nil {
		return
	}
	a.fileBusy = true
	window, fileID, name := a.window, file.ID, file.Name
	go func() {
		result, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent: window, Type: mygo.MessageWarning, Title: "Unsaved changes",
			Message: "Save changes to " + name + "?",
			Detail:  "Save the document before closing, or discard its unsaved changes.",
			Buttons: []string{"Cancel", "Discard changes", "Save"}, DefaultButton: 2,
			CancelButton: 0, DestructiveButtons: []int{1},
		})
		window.Update(func() {
			a.fileBusy = false
			if err != nil {
				a.notice = "Could not close document: " + err.Error()
			} else if result.Button == 1 {
				a.removeDocument(fileID)
			} else if result.Button == 2 {
				a.saveFile(file, false, func() { a.removeDocument(fileID) })
			}
		})
	}()
}

func (a *App) removeDocument(id string) {
	for index, file := range a.files {
		if file.ID != id {
			continue
		}
		a.files = append(a.files[:index], a.files[index+1:]...)
		delete(a.disk, id)
		delete(a.analyses, id)
		delete(a.conflicts, id)
		delete(a.checking, id)
		if len(a.files) == 0 {
			a.newDocument()
		} else if a.selected == id {
			a.selected = a.files[min(index, len(a.files)-1)].ID
		}
		a.notice = ""
		return
	}
}

func (a *App) openDialog() {
	if a.window == nil || a.fileBusy {
		return
	}
	a.fileBusy = true
	window := a.window
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: window, Title: "Open Markdown document",
			Filters: []mygo.FileFilter{{Name: "Markdown and text", Extensions: []string{"md", "markdown", "txt"}}},
		})
		if err != nil || len(paths) == 0 {
			window.Update(func() {
				a.fileBusy = false
				if err != nil {
					a.notice = "Could not open document: " + err.Error()
				}
			})
			return
		}
		file, snapshot, err := loadDocument(paths[0])
		window.Update(func() {
			a.fileBusy = false
			if err != nil {
				a.notice = "Could not open document: " + err.Error()
				return
			}
			a.addOpened(file, snapshot)
		})
	}()
}

func (a *App) openPath(path string) {
	if a.window == nil || a.fileBusy {
		return
	}
	for _, file := range a.files {
		if file.Path == path {
			a.selected = file.ID
			a.notice = "This document is already open."
			return
		}
	}
	a.fileBusy = true
	window := a.window
	go func() {
		file, snapshot, err := loadDocument(path)
		window.Update(func() {
			a.fileBusy = false
			if err != nil {
				a.notice = "Could not open document: " + err.Error()
				return
			}
			a.addOpened(file, snapshot)
		})
	}()
}

func loadDocument(path string) (*mdeditor.File, storage.Snapshot, error) {
	snapshot, data, err := storage.Open(path)
	if err != nil {
		return nil, storage.Snapshot{}, err
	}
	file := mdeditor.NewFile(filepath.Base(snapshot.Path), string(data))
	file.SetPath(snapshot.Path)
	return file, snapshot, nil
}

func (a *App) addOpened(file *mdeditor.File, snapshot storage.Snapshot) {
	for _, current := range a.files {
		if current.Path != "" && current.Path == snapshot.Path {
			a.selected = current.ID
			a.notice = "This document is already open."
			return
		}
	}
	a.nextID++
	file.ID = fmt.Sprintf("document-%d", a.nextID)
	a.files = append(a.files, file)
	a.disk[file.ID] = snapshot
	a.selected = file.ID
	file.Focus.Focus()
	if file.HasMixedLineEndings() {
		a.notice = "Mixed line endings: use Save as to create a normalized copy after editing."
	} else {
		a.notice = ""
	}
}

func (a *App) saveDocument(saveAs bool) {
	a.saveFile(a.current(), saveAs, nil)
}

func (a *App) saveFile(file *mdeditor.File, saveAs bool, onSaved func()) {
	if a.window == nil || a.fileBusy {
		return
	}
	fileID := file.ID
	defaultName := file.Name
	if a.folder != nil && file.Path == "" {
		defaultName = filepath.Join(a.folder.Path, defaultName)
	}
	path := file.Path
	captured := file.CaptureForSave()
	disk, hasDisk := a.disk[fileID]
	needsNewPath := saveAs || path == "" || !hasDisk
	a.fileBusy = true
	a.saving = true
	a.notice = "Saving…"
	window := a.window
	go func() {
		if needsNewPath {
			var err error
			path, err = mygo.Dialog.Save(mygo.SaveDialogOptions{
				Parent: window, Title: "Save Markdown document", DefaultPath: defaultName,
				Filters: []mygo.FileFilter{{Name: "Markdown", Extensions: []string{"md", "markdown"}}},
			})
			if err != nil || path == "" {
				window.Update(func() {
					a.fileBusy = false
					a.saving = false
					if err != nil {
						a.notice = "Could not save document: " + err.Error()
					} else {
						a.notice = "Save canceled."
					}
				})
				return
			}
		}
		data, err := captured.Bytes()
		if needsNewPath && errors.Is(err, mdeditor.ErrMixedLineEndings) {
			result, dialogErr := mygo.Dialog.Message(mygo.MessageOptions{
				Parent: window, Type: mygo.MessageWarning, Title: "Mixed line endings",
				Message: "Save a copy with consistent line endings?",
				Detail:  "The new file will use CRLF line endings. Your original file will not be changed.",
				Buttons: []string{"Cancel", "Save copy"}, DefaultButton: 0, CancelButton: 0,
			})
			switch {
			case dialogErr != nil:
				err = dialogErr
			case result.Button == 1:
				data, err = captured.NormalizedBytes()
			default:
				window.Update(func() {
					a.fileBusy = false
					a.saving = false
					a.notice = "Save canceled. Your changes remain open."
				})
				return
			}
		}
		var saved storage.Snapshot
		var source mdeditor.SavedSource
		var language *editor.Language
		if err == nil {
			if needsNewPath {
				saved, err = storage.Create(path, data)
			} else {
				saved, err = storage.Save(disk, data)
			}
			if err == nil {
				source = mdeditor.PrepareSavedSource(data)
				language = editor.LanguageForText(filepath.Base(saved.Path), string(data))
			}
		}
		window.Update(func() {
			a.fileBusy = false
			a.saving = false
			if err != nil {
				switch {
				case errors.Is(err, storage.ErrExternalChange):
					a.conflicts[fileID] = true
					a.notice = "File changed outside MarkGo. Your changes are still in the editor."
				case errors.Is(err, storage.ErrAlreadyExists):
					a.notice = "That file already exists. Choose a new name to avoid overwriting it."
				case errors.Is(err, mdeditor.ErrMixedLineEndings):
					a.notice = "Mixed line endings: use Save as to create a normalized copy."
				default:
					a.notice = "Could not save document: " + err.Error()
				}
				return
			}
			file.SetPath(saved.Path)
			file.Name = filepath.Base(saved.Path)
			file.State.Label = file.Name
			file.AcceptSavedSource(source)
			file.SetLanguage(language)
			a.disk[fileID] = saved
			delete(a.conflicts, fileID)
			if file.MarkSavedVersion(captured.Text.Version()) {
				a.notice = ""
				if onSaved != nil {
					onSaved()
				}
			} else {
				a.notice = "Saved an earlier version. Newer changes are still unsaved."
			}
		})
	}()
}
