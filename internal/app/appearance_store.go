package app

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/xddcode/markgo/internal/preferences"
)

// LoadAppearance runs at startup before the native event loop begins. Tests and
// ephemeral launches leave preferences detached from the user's real settings.
func (a *App) LoadAppearance() error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	return a.loadAppearance(&preferences.Store{Path: filepath.Join(dir, "MarkGo", "appearance.json")})
}

func (a *App) loadAppearance(store *preferences.Store) error {
	a.preferenceStore = store
	data, err := store.Load()
	if err != nil || data == nil {
		return err
	}
	var appearance appearance
	if err := json.Unmarshal(data, &appearance); err != nil {
		return err
	}
	a.appearance = normalizeAppearance(appearance)
	return nil
}

func normalizeAppearance(p appearance) appearance {
	if _, known := knownPalette(p.Theme); !known && p.Theme != "system" {
		p.Theme = "system"
	}
	if p.FontSize < 14 || p.FontSize > 24 {
		p.FontSize = 18
	}
	if p.Width != 640 && p.Width != 720 && p.Width != 840 {
		p.Width = 720
	}
	return p
}

func (a *App) saveAppearance() {
	if a.window != nil && mygo.Theme.Source() != a.ThemeSource() {
		mygo.Theme.SetSource(a.ThemeSource())
	}
	a.appearanceRevision++
	if a.preferenceStore == nil {
		return
	}
	store, revision, prefs, window := a.preferenceStore, a.appearanceRevision, a.appearance, a.window
	a.preferenceWorkers.Go(func() {
		data, err := json.Marshal(prefs)
		if err == nil {
			err = store.Save(revision, data)
		}
		if err != nil && window != nil {
			window.Update(func() {
				if a.appearanceRevision == revision {
					a.notice = "Your appearance is active, but it could not be remembered: " + err.Error()
				}
			})
		}
	})
}

// WaitForPreferences drains small settings writes after the event loop exits,
// so choosing an appearance and immediately quitting still remembers it.
// Never call this from a view or input callback.
func (a *App) WaitForPreferences() { a.preferenceWorkers.Wait() }

// ThemeSource also applies the chosen appearance to native window chrome and
// dialogs. The palette itself remains owned by our native-drawn view.
func (a *App) ThemeSource() mygo.ThemeSource {
	if p, known := knownPalette(a.appearance.Theme); known {
		if p.Dark {
			return mygo.ThemeDark
		}
		return mygo.ThemeLight
	}
	return mygo.ThemeSystem
}
