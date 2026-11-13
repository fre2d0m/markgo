package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/xddcode/markgo/internal/mdeditor"
)

type savePulse struct {
	revision uint64
	started  time.Time
}

// Acknowledgments are emitted only after the saved snapshot matches the
// current document. A later edit immediately takes precedence in the footer.
func (a *App) acknowledgeSave(file *mdeditor.File) {
	if file.Path != "" && !file.Modified() {
		a.savedEvents[file.ID]++
	}
}

func (a *App) saveStatus(c *ui.Context, file *mdeditor.File, state string) {
	t := c.Theme()
	row := ui.Row(c.Key("save-state-" + file.ID)).Gap(6).AlignItems(ui.Center).Label("Save status").Tooltip(file.Path)
	if version, ok := a.backedUp[file.ID]; ok && version == file.State.Buffer.Version() && file.Modified() && a.sessionError == "" {
		state = "Draft backed up"
		row.Tooltip("A local recovery copy is available. Use Save to update your Markdown file.")
	}
	pulse := ui.Local(row, "saved", func() savePulse { return savePulse{revision: a.savedEvents[file.ID]} })
	if pulse.revision != a.savedEvents[file.ID] {
		pulse.revision = a.savedEvents[file.ID]
		pulse.started = c.Now()
	}
	remaining := 2*time.Second - c.Now().Sub(pulse.started)
	confirmed := file.Path != "" && !file.Modified() && !pulse.started.IsZero() && remaining > 0
	if a.saving && a.savingFileID == file.ID {
		state = "Saving…"
		confirmed = false
	} else if confirmed {
		state = "Saved just now"
		c.After(remaining)
	}
	row.Children(func() {
		if confirmed {
			ui.Icon(c.Key("saved-check"), icons["check"]).Size(13, 13).TextColor(t.Text).
				Transition(ui.ElementTransition{Duration: 140 * time.Millisecond, Colors: true, Enter: &ui.Motion{Y: 2, Opacity: 0}, Exit: &ui.Motion{Opacity: 0}})
		}
		ui.Text(c, state).FontSize(11).TextColor(t.TextMuted)
	})
}
