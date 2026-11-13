package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/xddcode/markgo/internal/writingui"
)

func (a *App) showExitPrompt(quit, failed bool) {
	a.closing, a.fileBusy = true, true
	a.recentOpen, a.appearanceOpen, a.formatOpen = false, false, false
	a.exitPrompt, a.exitPromptError, a.exitQuit = true, failed, quit
	if a.window != nil {
		a.window.Invalidate()
	}
}

func (a *App) cancelExitPrompt() {
	a.exitPrompt, a.closing, a.fileBusy = false, false, false
	if !a.reading {
		a.returnToWriting = true
	}
}

func (a *App) chooseExit(action string) {
	a.exitPrompt, a.closing, a.fileBusy = false, false, false
	switch action {
	case "keep":
		a.persistAndExit(a.exitQuit, false)
	case "discard":
		a.finishExit(a.exitQuit)
	case "save":
		a.saveAllAndExit(a.exitQuit)
	case "exit":
		a.closeNow(a.exitQuit)
	}
}

func (a *App) exitDialog(c *ui.Context) {
	t := c.Theme()
	dialog := ui.DialogBase(c.Key("exit-confirmation"), &a.exitPrompt, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.18))
		writingui.Surface(c, panel).Width(560).Padding(24).Gap(20).Label("Close writing session").Children(func() {
			title, detail := "Leave your writing here?", "Save your documents or discard their unsaved changes before closing."
			if a.sessionStore != nil {
				detail = "Keep drafts on this device and continue next time, or save your Markdown files before closing."
			}
			if a.exitPromptError {
				title = "Your drafts could not be backed up"
				detail = "Stay open to save your work or retry. Exiting now may lose newer edits or bring back an older recovery copy."
			}
			ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons["recovery"]).Size(23, 23).TextColor(t.TextMuted)
				ui.Text(c, title).Grow(1).FontSize(19).FontWeight(500).LetterSpacing(-0.3)
			})
			ui.Text(c, detail).FillWidth().FontSize(13).LineHeight(1.6).TextColor(t.TextMuted)
			if !a.exitPromptError {
				count := 0
				for _, file := range a.files {
					if file.Modified() {
						count++
					}
				}
				label := fmt.Sprintf("%d documents have unsaved changes", count)
				if count == 1 {
					label = "1 document has unsaved changes"
				}
				ui.Text(c, label).FontSize(12).TextColor(t.TextMuted)
			}
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				if a.exitPromptError {
					writingui.Button(c, "Exit anyway", false).TextColor(t.Danger).Label("Exit without updating recovery").OnClick(func() { a.chooseExit("exit") })
					ui.Spacer(c)
					writingui.Button(c, "Stay open", true).Label("Stay open after recovery failure").AutoFocus().OnClick(a.cancelExitPrompt)
					return
				}
				writingui.Button(c, "Cancel", false).Label("Cancel closing").OnClick(a.cancelExitPrompt)
				ui.Spacer(c)
				writingui.Button(c, "Discard", false).TextColor(t.Danger).Label("Discard unsaved changes").Tooltip("Discard unsaved changes and their recovery copies").OnClick(func() { a.chooseExit("discard") })
				if a.sessionStore != nil {
					writingui.Button(c, "Save all", false).Label("Save all before closing").OnClick(func() { a.chooseExit("save") })
					writingui.Button(c, "Keep drafts", true).Label("Keep drafts and close").AutoFocus().OnClick(func() { a.chooseExit("keep") })
				} else {
					writingui.Button(c, "Save all", true).Label("Save all before closing").AutoFocus().OnClick(func() { a.chooseExit("save") })
				}
			})
		})
	})
	if dialog.Dismissed() {
		a.cancelExitPrompt()
	}
}
