package mdeditor

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

type typesetBlockUI struct {
	languageOpen                 bool
	languageAt, copiedAt         int
	languageVersion              uint64
	languageDraft, languageError string
	copiedUntil                  time.Time
}

// Original native vectors share the workbench's 24-unit, 1.65-stroke grid.
var codeIcons = map[string]*ui.SVG{
	"chevron":  ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round"><path d="m8 10 4 4 4-4"/></svg>`)),
	"copy":     ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round"><rect x="8" y="8" width="12" height="13" rx="2"/><path d="M16 8V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h3"/></svg>`)),
	"check":    ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round"><path d="m5 12 4.5 4.5L19 7"/></svg>`)),
	"continue": ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round"><path d="M19 4v8a3 3 0 0 1-3 3H5m4-4-4 4 4 4"/></svg>`)),
}

func codeButton(c *ui.Context, label string) ui.Element {
	th := c.Theme()
	b := ui.ButtonBase(c).Height(40).MinWidth(40).KeepFocus().Radius(6).Label(label)
	b.HandleInput(func(ui.InputEvent) bool { return false })
	fill := ui.Transparent
	if b.Hovered() {
		fill = th.SurfaceHover
	}
	if b.Pressed() {
		fill = th.SurfacePressed
	}
	b.Background(b.AnimateColor("hover", fill, ui.Animation{Duration: 100 * time.Millisecond, NonMotion: true}))
	scale := float32(1)
	if b.Pressed() {
		scale = 0.96
	}
	scale = b.AnimateTo("press", scale, ui.Animation{Duration: 90 * time.Millisecond, Ease: ui.EaseOut})
	return b.Scale(scale, scale)
}

func (t *Typesetter) codeHeaders(c *ui.Context, view ui.Element) {
	u, th := &t.blockUI, c.Theme()
	if u.languageOpen && u.languageVersion != t.Source.Buffer.Version() {
		u.languageOpen = false
	}
	boxes := t.Document.VisibleParagraphs()
	services := c.Services()
	view.Children(func() {
		for _, box := range boxes {
			line, ok := t.projection.shownLine(box.Index)
			if !ok || line.block.code == 0 {
				continue
			}
			code := t.projection.codes[line.block.code-1]
			if t.projection.visible[box.Index] != code.first {
				continue
			}
			name := ""
			if fields := strings.Fields(code.info); len(fields) > 0 {
				name = fields[0]
			}
			label := name
			if label == "" {
				label = "Plain text"
			}
			ui.Row(c.Key(fmt.Sprintf("code-header-%d", code.open))).Absolute().Left(8).Right(8).Top(box.Top - 44).Height(40).Gap(2).AlignItems(ui.Center).Children(func() {
				language := codeButton(c.Key("language"), "Code language: "+label).Padding(0, 8).MaxWidth(180).Tooltip("Change code language")
				language.Children(func() {
					ui.Row(c).Fill().Gap(4).AlignItems(ui.Center).Children(func() {
						ui.Text(c, label).FontSize(12).TextColor(th.TextMuted).SingleLine().Ellipsis("…").MinWidth(0)
						ui.Icon(c, codeIcons["chevron"]).Size(14, 14).TextColor(th.TextMuted).Shrink(0)
					})
				})
				language.OnClick(func() {
					t.CancelComposition()
					u.languageOpen = !u.languageOpen || u.languageAt != code.open
					u.languageAt, u.languageDraft, u.languageError = code.open, name, ""
					u.languageVersion = t.Source.Buffer.Version()
				})
				if u.languageAt == code.open {
					t.languagePopover(c, language)
				}
				ui.Spacer(c)
				copied := u.copiedAt == code.open && c.Now().Before(u.copiedUntil)
				if copied {
					c.After(u.copiedUntil.Sub(c.Now()))
				}
				icon, hint := "copy", "Copy code"
				if copied {
					icon, hint = "check", "Code copied"
				}
				copy := codeButton(c.Key("copy"), hint).Width(40).Center().Tooltip(hint)
				copy.Children(func() { ui.Icon(c, codeIcons[icon]).Size(16, 16).TextColor(th.TextMuted) })
				copy.OnClick(func() {
					if text, ok := t.CodeText(code.open); ok {
						services.WriteClipboard(text)
						u.copiedAt, u.copiedUntil = code.open, c.Now().Add(1500*time.Millisecond)
					}
				})
				next := codeButton(c.Key("continue"), "Continue after code").Width(40).Center().Tooltip("Continue writing · Cmd/Ctrl+Enter")
				next.Children(func() { ui.Icon(c, codeIcons["continue"]).Size(17, 17).TextColor(th.TextMuted) })
				next.OnClick(func() {
					t.ContinueAfterCode(code.open)
					t.input.handle.Focus(c)
				})
			})
		}
	})
}

func (t *Typesetter) languagePopover(c *ui.Context, anchor ui.Element) {
	u, th := &t.blockUI, c.Theme()
	apply := func() {
		if t.SetCodeLanguage(u.languageAt, u.languageDraft) {
			u.languageOpen = false
			t.input.handle.Focus(c)
		} else {
			u.languageError, t.Notice = t.Notice, ""
		}
	}
	ui.PopoverBase(c, anchor, &u.languageOpen, func(panel ui.Element) {
		panel.Transition(ui.ElementTransition{Duration: 130 * time.Millisecond, ExitDuration: 90 * time.Millisecond, Colors: true, Enter: &ui.Motion{Y: -4, Opacity: 0, Flip: true}, Exit: &ui.Motion{Y: -2, Opacity: 0, Flip: true}})
		panel.Width(260).Padding(14).Gap(10).Margin(6, 0, 0).Radius(12).Background(th.Background).Border(1, th.Border.Alpha(0.7)).Shadow(0, 8, 28, 0, ui.RGBA(0, 0, 0, 0.09)).Children(func() {
			ui.Text(c, "Code language").FontSize(13).FontWeight(500)
			ui.TextInputBase(c.Key("language-name"), &u.languageDraft).FillWidth().Height(40).Padding(6, 9).Background(th.Surface).Border(1, th.Border).Radius(6).FontSize(13).Label("Language name").Placeholder("Plain text").AutoFocus().OnSubmit(apply).OnChange(func() { u.languageError = "" })
			if u.languageError != "" {
				ui.Text(c, u.languageError).FillWidth().FontSize(12).TextColor(th.TextMuted).LineHeight(1.5).Role(ui.RoleAlert)
			} else {
				ui.Text(c, "For example: go, python, javascript.\nLeave empty for plain text.").FillWidth().FontSize(12).TextColor(th.TextMuted).LineHeight(1.5)
			}
			ui.Row(c).FillWidth().Gap(6).Justify(ui.End).Children(func() {
				cancel := codeButton(c.Key("cancel"), "Cancel language change").Padding(0, 12).Center()
				cancel.Children(func() { ui.Text(c, "Cancel").FontSize(12).TextColor(th.TextMuted) })
				cancel.OnClick(func() { u.languageOpen = false; t.input.handle.Focus(c) })
				button := codeButton(c.Key("apply"), "Apply code language").Padding(0, 14).Background(th.SurfaceHover).Center()
				button.Children(func() { ui.Text(c, "Apply").FontSize(12).FontWeight(500) })
				button.OnClick(apply)
			})
		})
		if panel.OverlayShortcut(0, ui.KeyEscape) {
			u.languageOpen = false
			t.input.handle.Focus(c)
		}
	})
}

func (t *Typesetter) quoteBars(c *ui.Context, view ui.Element) {
	boxes := t.Document.VisibleParagraphs()
	ink, gutter := c.Theme().Border, t.input.gutter
	view.Children(func() {
		for _, box := range boxes {
			line, ok := t.projection.shownLine(box.Index)
			if !ok || line.block.quote == 0 {
				continue
			}
			depth := line.block.quote
			ui.Box(c.Key(fmt.Sprintf("quote-%d", box.Index))).Absolute().Left(0).Top(box.Top - 2).Width(float32(depth) * gutter).Height(box.Height + 4).Draw(func(p *ui.Painter, r ui.Rect) {
				for level := range depth {
					p.Fill(ui.Rect{X: r.X + float32(level)*gutter + 2, Y: r.Y, W: 2, H: r.H}, ink, 1)
				}
			})
		}
	})
}
