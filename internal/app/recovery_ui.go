package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/xddcode/markgo/internal/mdeditor"
	"github.com/xddcode/markgo/internal/writingui"
)

func (a *App) sessionWarning(c *ui.Context) {
	if a.sessionError == "" && len(a.recentMissing) == 0 {
		return
	}
	t := c.Theme()
	ui.Row(c.Key("session-warning")).FillWidth().Padding(8, 20).Gap(14).AlignItems(ui.Center).Background(t.Surface).Children(func() {
		message := fmt.Sprintf("%d previous documents could not be reopened.", len(a.recentMissing))
		if len(a.recentMissing) == 1 {
			message = "One previous document could not be reopened."
		}
		if a.sessionError != "" {
			message = "Recovery is unavailable. Save your documents to keep your work."
		}
		ui.Text(c, message).FontSize(12).TextColor(t.Text).Grow(1).MinWidth(0).Role(ui.RoleStatus).Tooltip(a.sessionError)
		if a.sessionError != "" {
			if a.sessionStore != nil {
				writingui.Button(c, "Try again", false).Label("Retry recovery backup").OnClick(func() {
					a.sessionError = ""
					a.sessionObserved = nil
				})
			}
		} else {
			writingui.Button(c, "Review", false).Label("Review unavailable documents").OnClick(func() { a.recentOpen = true })
		}
	})
}

func (a *App) recoveryNotice(c *ui.Context, file *mdeditor.File) {
	reason := a.recovered[file.ID]
	if reason == "" {
		return
	}
	t := c.Theme()
	ui.Row(c.Key("recovered-"+file.ID)).FillWidth().Padding(12, 24).Gap(14).AlignItems(ui.Center).Background(t.Surface).Children(func() {
		ui.Icon(c, icons["recovery"]).Size(18, 18).Shrink(0).TextColor(t.TextMuted)
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, "Recovered draft").FontSize(12).FontWeight(600).TextColor(t.Text).Role(ui.RoleStatus)
			ui.Text(c, reason).FontSize(12).LineHeight(1.45).TextColor(t.TextMuted)
		})
		copy := file.Path != "" && a.disk[file.ID].Path == ""
		label := "Save"
		if copy {
			label = "Save a copy"
		}
		writingui.Button(c, label, false).Background(t.Background).Border(1, t.Border).Shrink(0).Label("Save recovered draft").Disabled(a.fileBusy).OnClick(func() { a.saveFile(file, copy, nil) })
		iconButton(c, "dismiss-recovery", "close", "Dismiss recovery notice").OnClick(func() { delete(a.recovered, file.ID) })
	})
}

func (a *App) recentDialog(c *ui.Context) {
	t := c.Theme()
	ui.DialogBase(c.Key("recent-documents"), &a.recentOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.18))
		writingui.Surface(c, panel).Width(560).Height(488).Padding(24).Gap(16).Label("Recent documents dialog").Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Column(c).Grow(1).Gap(4).Children(func() {
					ui.Text(c, "Pick up where you left off").FontSize(18).FontWeight(500).LetterSpacing(-0.3)
					ui.Text(c, "Recent documents, kept on this device.").FontSize(12).TextColor(t.TextMuted)
				})
				iconButton(c, "close-recent", "close", "Close recent documents").OnClick(func() { a.recentOpen = false })
			})
			ui.SearchField(c.Key("recent-search"), &a.recentQuery).Height(40).FontSize(13).Radius(8).Background(t.Surface).Border(1, t.Border).Placeholder("Find by name or folder").Label("Search recent documents").AutoFocus().OnSubmit(func() {
				for _, path := range a.recent {
					if strings.Contains(strings.ToLower(path), strings.ToLower(strings.TrimSpace(a.recentQuery))) {
						a.OpenFile(path)
						return
					}
				}
			})
			matches := 0
			ui.Scroll(c).Grow(1).FillWidth().MinHeight(0).Children(func() {
				ui.Column(c).FillWidth().Gap(3).Children(func() {
					for _, path := range a.recent {
						if !strings.Contains(strings.ToLower(path), strings.ToLower(strings.TrimSpace(a.recentQuery))) {
							continue
						}
						matches++
						name := filepath.Base(path)
						row := ui.ButtonBase(c.Key(path)).FillWidth().MinHeight(58).Padding(9, 10).Radius(8).Label("Open recent " + name).Tooltip(path).Disabled(a.fileBusy)
						fill := ui.Transparent
						if row.Hovered() || row.Focused() {
							fill = t.SurfaceHover
						}
						if row.Pressed() {
							fill = t.SurfacePressed
						}
						row.Background(row.AnimateColor("recent-hover", fill, ui.Animation{Duration: 100 * time.Millisecond, NonMotion: true}))
						showRemove := row.Hovered() || row.Focused() || c.Preferences().NoHover || a.recentMissing[path] != ""
						row.Children(func() {
							ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
								ui.Icon(c, icons["file"]).Size(18, 18).Shrink(0).TextColor(t.TextMuted)
								ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
									ui.Text(c, name).FillWidth().FontSize(13).FontWeight(500).SingleLine().Ellipsis("…")
									ui.Text(c, filepath.Dir(path)).FillWidth().FontSize(11).TextColor(t.TextMuted).SingleLine().Ellipsis("…")
								})
								if problem := a.recentMissing[path]; problem != "" {
									ui.Text(c, "Unavailable").FontSize(11).TextColor(t.TextMuted).Tooltip(problem)
								}
								ui.Box(c).Size(40, 40).Shrink(0).Children(func() {
									if showRemove {
										iconButton(c, "forget-recent", "close", "Remove recent "+name).Tooltip("Remove from history; keep the file").OnClick(func() { a.forgetRecent(path) })
									}
								})
							})
						})
						row.OnClick(func() { a.OpenFile(path) })
					}
					if matches == 0 {
						ui.Column(c).FillWidth().Padding(36, 20).Gap(10).AlignItems(ui.Center).Children(func() {
							glyph := "recovery"
							if len(a.recent) > 0 {
								glyph = "search"
							}
							ui.Icon(c, icons[glyph]).Size(25, 25).TextColor(t.TextMuted)
							label, hint := "Your next page starts here", "Documents you open or save will appear here."
							if len(a.recent) > 0 {
								label, hint = "No matching documents", "Try a different name or folder."
							}
							ui.Text(c, label).FontSize(14).FontWeight(500)
							ui.Text(c, hint).FontSize(12).TextColor(t.TextMuted)
						})
					}
				})
			})
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				writingui.Button(c, "Clear history", false).Label("Clear recent history").Disabled(len(a.recent) == 0).Tooltip("Remove this list; your files and open drafts stay intact").OnClick(a.clearRecent)
				ui.Spacer(c)
				writingui.Button(c, "Open document…", true).Label("Browse for a document").Disabled(a.fileBusy).OnClick(func() { a.recentOpen = false; a.openDialog() })
			})
		})
	})
	if a.recentWasOpen && !a.recentOpen && !a.reading {
		a.current().Focus.Focus(c)
	}
	a.recentWasOpen = a.recentOpen
}
