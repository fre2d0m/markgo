package mdeditor

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

type typesetLinkUI struct {
	open, wasOpen           bool
	version                 uint64
	start, end              int
	anchor, caret           int
	existing                bool
	mark                    typesetInline
	label, address, error   string
	originalLabel, rawLabel string
}

// OpenLink edits the link under the selection, or turns a selected phrase
// into a link. A link is never launched as a side effect of editing it.
func (t *Typesetter) OpenLink() bool {
	t.ToSource()
	sel := t.Source.Selection()
	a, z := sel.Start(), sel.End()
	u := typesetLinkUI{open: true, version: t.Source.Buffer.Version(), start: a, end: z}
	u.anchor, u.caret = t.Document.Selection()
	for _, m := range t.projection.inlines {
		if m.kind == InlineLink && a >= m.content && z <= m.end && z >= m.content && a <= m.contentEnd {
			u.existing, u.mark = true, m
			u.start, u.end = m.start, m.end
			u.rawLabel = t.projection.source[m.content:m.contentEnd]
			u.label = t.Document.Text().Slice(t.projection.viewOffset(m.content), t.projection.viewOffset(m.contentEnd)).String()
			u.address = m.destination
			break
		}
	}
	if !u.existing {
		var ok bool
		u.start, u.end, ok = t.inlineRange()
		if !ok {
			return false
		}
		for _, m := range t.projection.inlines {
			if m.kind == InlineLink && u.start < m.end && u.end > m.start {
				t.Notice = "Choose one link at a time."
				return false
			}
		}
		u.rawLabel = t.projection.source[u.start:u.end]
		u.label = t.Document.Text().Slice(min(u.anchor, u.caret), max(u.anchor, u.caret)).String()
	}
	u.originalLabel = u.label
	t.linkUI = u
	t.Notice = ""
	return true
}

func (t *Typesetter) applyLink(remove bool) bool {
	u := &t.linkUI
	if u.version != t.Source.Buffer.Version() {
		u.error = "The text changed. Close this panel and choose the link again."
		return false
	}
	label := u.label
	if label == u.originalLabel {
		label = u.rawLabel
	}
	if remove {
		if !u.existing {
			return false
		}
		label = u.rawLabel
	} else if strings.TrimSpace(label) == "" || strings.ContainsAny(label, "\n\r") {
		u.error = "Add a short, single-line link label."
		return false
	}
	address := strings.TrimSpace(u.address)
	if !remove && (address == "" || strings.ContainsAny(address, "<>\n\r\t")) {
		u.error = "Enter a web address, a file path or a #section."
		return false
	}
	text, offset := label, 0
	if !remove {
		title := ""
		if u.existing && u.mark.title != "" {
			title = " \"" + strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(u.mark.title) + "\""
		}
		text, offset = "["+label+"](<"+strings.ReplaceAll(address, "\\", "\\\\")+">"+title+")", 1
		p := projectTypesetting(t.projection.source[:u.start] + text + t.projection.source[u.end:])
		valid := false
		for _, m := range p.inlines {
			if m.kind == InlineLink && m.start == u.start && m.end == u.start+len(text) {
				valid = true
				break
			}
		}
		if !valid {
			u.error = "That label or address cannot form a Markdown link."
			return false
		}
	}
	if u.existing && !remove && u.label == u.originalLabel && address == u.mark.destination {
		u.open = false
		return true
	}
	a, z := u.start+offset, u.start+offset+len(label)
	if u.anchor == u.caret {
		a = z
		if !remove {
			a = u.start + len(text)
			z = a
		}
	}
	t.Document.SetSelection(u.anchor, u.caret)
	t.replaceInline(u.start, u.end, text, a, z, false)
	if u.existing && label == u.rawLabel {
		t.Document.SetSelection(u.anchor, u.caret)
		t.caretAffinity = false
		t.ToSource()
	}
	u.open = false
	return true
}

func (t *Typesetter) linkPopover(c *ui.Context, view ui.Element) {
	u, th := &t.linkUI, c.Theme()
	if u.open && u.version != t.Source.Buffer.Version() {
		u.open = false
	}
	if u.wasOpen && !u.open {
		t.input.handle.Focus(c)
	}
	u.wasOpen = u.open
	if !u.open {
		return
	}
	caret, _ := t.Document.CaretRect()
	view.Children(func() {
		anchor := ui.Box(c.Key("link-anchor")).Absolute().Left(caret.X).Top(caret.Y+caret.H).Size(1, 1)
		close := func() { u.open = false; t.input.handle.Focus(c) }
		apply := func() {
			if t.applyLink(false) {
				t.input.handle.Focus(c)
			}
		}
		ui.PopoverBase(c, anchor, &u.open, func(panel ui.Element) {
			panel.Transition(ui.ElementTransition{Duration: 130 * time.Millisecond, ExitDuration: 90 * time.Millisecond, Colors: true, Enter: &ui.Motion{Y: -4, Opacity: 0, Flip: true}, Exit: &ui.Motion{Y: -2, Opacity: 0, Flip: true}})
			panel.Width(340).Padding(16).Gap(10).Margin(8, 0, 0).Radius(14).Background(th.Background).Border(1, th.Border.Alpha(0.7)).Shadow(0, 8, 28, 0, ui.RGBA(0, 0, 0, 0.09)).Children(func() {
				title := "Add a link"
				if u.existing {
					title = "Edit link"
				}
				ui.Text(c, title).FontSize(14).FontWeight(500)
				ui.Text(c, "Text").FontSize(12).TextColor(th.TextMuted)
				text := ui.TextInputBase(c.Key("link-label"), &u.label).FillWidth().Height(40).Padding(6, 10).Background(th.Surface).Border(1, th.Border).Radius(6).FontSize(13).Label("Link text").Placeholder("A few words").OnSubmit(apply).OnChange(func() { u.error = "" })
				ui.Text(c, "Address").FontSize(12).TextColor(th.TextMuted)
				address := ui.TextInputBase(c.Key("link-address"), &u.address).FillWidth().Height(40).Padding(6, 10).Background(th.Surface).Border(1, th.Border).Radius(6).FontSize(13).Label("Link address").Placeholder("https://, a file path, or #section").OnSubmit(apply).OnChange(func() { u.error = "" })
				if u.originalLabel == "" {
					text.AutoFocus()
				} else {
					address.AutoFocus()
				}
				hint := "A small connection to the next idea."
				if u.existing && u.mark.reference {
					hint = "Only this link will change."
				}
				if u.error != "" {
					hint = u.error
				}
				message := ui.Text(c, hint).FillWidth().MinHeight(36).FontSize(12).TextColor(th.TextMuted).LineHeight(1.5)
				if u.error != "" {
					message.Role(ui.RoleAlert)
				}
				ui.Row(c).FillWidth().Gap(4).Children(func() {
					if u.existing {
						b := codeButton(c.Key("remove-link"), "Remove link").Padding(0, 8).Center()
						b.Children(func() { ui.Text(c, "Remove").FontSize(12).TextColor(th.TextMuted) })
						b.OnClick(func() {
							if t.applyLink(true) {
								t.input.handle.Focus(c)
							}
						})
					}
					ui.Spacer(c)
					cancel := codeButton(c.Key("cancel-link"), "Cancel link edit").Padding(0, 12).Center()
					cancel.Children(func() { ui.Text(c, "Cancel").FontSize(12).TextColor(th.TextMuted) })
					cancel.OnClick(close)
					button := codeButton(c.Key("apply-link"), "Apply link").Padding(0, 14).Background(th.SurfaceHover).Center()
					button.Children(func() { ui.Text(c, "Apply").FontSize(12).FontWeight(500) })
					button.OnClick(apply)
				})
			})
			if panel.OverlayShortcut(0, ui.KeyEscape) {
				close()
			}
		})
	})
}
