package mdeditor

import (
	"fmt"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// The gutter is shared by every list kind. It reserves a 40 DIP task target
// and makes wrapped lines align with the text, even at the smallest font size.
const listGutter float32 = 40

func (t *Typesetter) listGutterWidth(font ui.Font) float32 {
	if t.projection.listNumber == 0 {
		return listGutter
	}
	width, _ := ui.ShapeText(strconv.Itoa(t.projection.listNumber)+".", font, 0).Size()
	return max(listGutter, width+18)
}

func (t *Typesetter) listMarkers(c *ui.Context, view ui.Element, font ui.Font) {
	boxes := t.Document.VisibleParagraphs()
	th := c.Theme()
	view.Children(func() {
		for _, box := range boxes {
			line, ok := t.projection.shownLine(box.Index)
			if !ok {
				continue
			}
			style := line.list
			if style.kind == NoList || style.kind == listContinuation {
				continue
			}
			gutter := t.input.gutter
			x := float32(style.depth) * gutter
			y := box.Top + (font.Size*font.LineHeight-listGutter)/2
			if style.kind == TaskList {
				idx := t.projection.visible[box.Index]
				label := "Toggle task: " + t.projection.source[line.content:line.end]
				button := ui.ButtonBase(c.Key(fmt.Sprintf("task-%d", idx))).Absolute().Left(x+gutter-listGutter).Top(y).Size(listGutter, listGutter).KeepFocus().TabStop(false).Role(ui.RoleCheckBox).Checked(style.checked).Label(label).Tooltip("Toggle task · Cmd/Ctrl+Enter").Radius(7)
				// Stop the ancestor rich-text handler from taking this press as
				// a text selection. Returning false leaves the base button's
				// normal click, cancellation and accessibility behavior in charge.
				button.HandleInput(func(ui.InputEvent) bool { return false })
				fill := ui.Transparent
				if button.Hovered() || button.Pressed() {
					fill = th.SurfaceHover
				}
				button.Background(button.AnimateColor("hover", fill, ui.Animation{Duration: 100 * time.Millisecond, NonMotion: true}))
				scale := float32(1)
				if button.Pressed() {
					scale = 0.96
				}
				scale = button.AnimateTo("press", scale, ui.Animation{Duration: 90 * time.Millisecond, Ease: ui.EaseOut})
				button.Scale(scale, scale)
				checked := style.checked
				button.Draw(func(p *ui.Painter, r ui.Rect) {
					box := ui.Rect{X: r.X + 11, Y: r.Y + 11, W: 18, H: 18}
					if checked {
						p.Fill(box, th.Text, 4)
						var check ui.Path
						check.MoveTo(box.X+4, box.Y+9).LineTo(box.X+7.5, box.Y+12.5).LineTo(box.X+14, box.Y+5.5)
						p.StrokePath(&check, 1.65, th.Background)
					} else {
						p.Stroke(box, th.TextMuted, 4, 1.25)
					}
				})
				button.OnClick(func() { t.ToggleTask(idx) })
				continue
			}
			marker := ui.Box(c.Key(fmt.Sprintf("marker-%d", box.Index))).Absolute().Left(x).Top(y).Size(gutter, listGutter)
			if style.kind == NumberedList {
				// Drawing the source-derived decimal ordinal avoids MyGo's
				// rich-text numbering rules (always 1, then letters/Roman).
				text := strconv.Itoa(style.number) + "."
				marker.Children(func() {
					ui.Text(c, text).FontSize(font.Size).Font(font.Family).TextColor(th.TextMuted).Absolute().Right(9).Top((listGutter - font.Size*font.LineHeight) / 2).LineHeight(font.LineHeight)
				})
			} else {
				marker.Draw(func(p *ui.Painter, r ui.Rect) {
					bullet := ui.Rect{X: r.X + r.W - 22, Y: r.Y + 18, W: 4, H: 4}
					if style.depth%2 == 0 {
						p.Fill(bullet, th.TextMuted, 2)
					} else {
						p.Stroke(bullet, th.TextMuted, 2, 1)
					}
				})
			}
		}
	})
}
