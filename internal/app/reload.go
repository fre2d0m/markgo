package app

import (
	"errors"

	"github.com/egoist/mygo"
	"github.com/xddcode/markgo/internal/mdeditor"
	"github.com/xddcode/markgo/internal/storage"
)

func (a *App) checkExternalChange(file *mdeditor.File) {
	if a.window == nil || a.saving || a.checking[file.ID] {
		return
	}
	snapshot, ok := a.disk[file.ID]
	if !ok {
		return
	}
	a.checking[file.ID] = true
	window, id := a.window, file.ID
	go func() {
		err := storage.Check(snapshot)
		window.Update(func() {
			delete(a.checking, id)
			current, stillOpen := a.disk[id]
			if !stillOpen || current != snapshot {
				return
			}
			if errors.Is(err, storage.ErrExternalChange) {
				a.conflicts[id] = true
				a.notice = "The file changed outside MarkGo."
			} else if err == nil {
				delete(a.conflicts, id)
			} else {
				a.notice = "Could not check file: " + err.Error()
			}
		})
	}()
}

func (a *App) reloadDocument(file *mdeditor.File) {
	if a.window == nil || a.fileBusy || file.Path == "" {
		return
	}
	a.fileBusy = true
	window, path, id := a.window, file.Path, file.ID
	version, dirty := file.State.Buffer.Version(), file.Modified()
	go func() {
		if dirty {
			result, err := mygo.Dialog.Message(mygo.MessageOptions{
				Parent: window, Type: mygo.MessageWarning, Title: "Reload document",
				Message: "Discard your unsaved changes and reload this file?",
				Detail:  "The version on disk will replace the text currently in the editor.",
				Buttons: []string{"Cancel", "Reload"}, DefaultButton: 0, CancelButton: 0, DestructiveButtons: []int{1},
			})
			if err != nil || result.Button != 1 {
				window.Update(func() {
					a.fileBusy = false
					if err != nil {
						a.notice = "Could not reload document: " + err.Error()
					}
				})
				return
			}
		}
		loaded, snapshot, err := loadDocument(path)
		window.Update(func() {
			a.fileBusy = false
			if err != nil {
				a.notice = "Could not reload document: " + err.Error()
				return
			}
			a.applyReload(id, version, loaded, snapshot)
		})
	}()
}

func (a *App) applyReload(id string, version uint64, loaded *mdeditor.File, snapshot storage.Snapshot) bool {
	for index, file := range a.files {
		if file.ID != id {
			continue
		}
		if file.State.Buffer.Version() != version {
			a.notice = "The editor changed while reloading; your newer text was kept."
			return false
		}
		loaded.ID = file.ID
		loaded.Focus = file.Focus
		a.files[index] = loaded
		a.disk[id] = snapshot
		delete(a.analyses, id)
		delete(a.conflicts, id)
		a.notice = "Reloaded " + loaded.Name
		return true
	}
	return false
}
