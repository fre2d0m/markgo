package mdeditor

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/richtext"
)

// Desktop input methods call TextInputClient directly, without ObserveInput.
// Own this boundary so preedit can never enter the canonical source/history.
// Richtext still owns shaping, painting, pointer selection and navigation.
type typesettingInput struct {
	owner      *Typesetter
	font       ui.Font
	handle     ui.Handle
	width      float32
	gutter     float32
	marked     bool
	start, end int // rune offsets in the staged display
	ended      *endedComposition
	typing     *typesettingTyping
}

type typesettingTyping struct {
	version            uint64
	anchor, caret, end int // canonical source bytes
	text               string
	at                 time.Time
}

type endedComposition struct {
	version       uint64
	range16       ui.TextInputRange
	anchor, caret int
	applied       bool
}

var _ ui.TextInputClient = (*typesettingInput)(nil)

func runeByte(s string, at int) int {
	n := 0
	for off := range s {
		if n >= at {
			return off
		}
		n++
	}
	return len(s)
}

func nativeOffset(s string, at int) int { return ui.UTF16Len(s[:runeByte(s, at)]) }

func nativeRange(s string, r ui.TextInputRange) (int, int) {
	a, z := ui.UTF16ByteOffset(s, min(r.Start, r.End)), ui.UTF16ByteOffset(s, max(r.Start, r.End))
	return utf8.RuneCountInString(s[:a]), utf8.RuneCountInString(s[:z])
}

func (n *typesettingInput) TextForRange(r ui.TextInputRange) (string, ui.TextInputRange) {
	s := n.owner.Document.Text().String()
	a, z := nativeRange(s, r)
	return s[runeByte(s, a):runeByte(s, z)], ui.TextInputRange{Start: nativeOffset(s, a), End: nativeOffset(s, z)}
}

func (n *typesettingInput) Selection() ui.TextInputSelection {
	s := n.owner.Document.Text().String()
	a, z := n.owner.Document.Selection()
	return ui.TextInputSelection{Range: ui.TextInputRange{Start: nativeOffset(s, min(a, z)), End: nativeOffset(s, max(a, z))}, Reversed: a > z}
}

func (n *typesettingInput) MarkedRange() (ui.TextInputRange, bool) {
	s := n.owner.Document.Text().String()
	return ui.TextInputRange{Start: nativeOffset(s, n.start), End: nativeOffset(s, n.end)}, n.marked
}

func (n *typesettingInput) replace(a, z int, s string, marked bool) {
	t := n.owner
	t.Document.Edit(func(e *richtext.Edit) {
		e.ReplaceString(min(a, z), max(a, z), s, richtext.Attributes{Underline: marked})
		caret := min(a, z) + utf8.RuneCountInString(s)
		e.Select(caret, caret)
	})
}

func (n *typesettingInput) ReplaceText(r *ui.TextInputRange, s string) {
	t := n.owner
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	plainTyping := r == nil && !t.composing && utf8.RuneCountInString(s) == 1 && !strings.ContainsAny(s, "\n\t`~")
	// GTK/IMM32 may unmark before sending the final replacement. If no
	// intervening action occurred, replace that one commit, not its history.
	if ended := n.ended; ended != nil && r != nil && *r == ended.range16 && t.Source.Buffer.Version() == ended.version {
		n.ended = nil
		if ended.applied {
			t.Source.Run(editor.Undo)
		}
		t.FromSource()
		t.Document.SetSelection(ended.anchor, ended.caret)
		r = nil
	}
	n.ended = nil
	if !t.composing {
		t.Sync()
		t.beforeAnchor, t.beforeCaret = t.Document.Selection()
	}
	anchor, caret := t.projection.sourceOffset(t.beforeAnchor), t.projection.sourceOffset(t.beforeCaret)
	now := time.Now()
	if group := n.typing; plainTyping && group != nil && group.version == t.Source.Buffer.Version() && anchor == caret && caret == group.end && now.Sub(group.at) < time.Second {
		// Buffer exposes transactions, but not an open typing group across
		// input callbacks. Replace our own latest step only when its exact
		// version and caret still match; never absorb another command's edit.
		t.Source.Run(editor.Undo)
		t.Source.Select(group.anchor, group.caret)
		t.Source.Insert(group.text + s)
		t.FromSource()
		group.text += s
		group.end = t.Source.Cursor()
		group.version = t.Source.Buffer.Version()
		group.at = now
		return
	}
	n.typing = nil
	beforeVersion, beforeLen := t.Source.Buffer.Version(), t.Source.Buffer.Len()
	a, z := t.Document.Selection()
	if r != nil {
		a, z = nativeRange(t.Document.Text().String(), *r)
		if !t.composing {
			t.beforeAnchor, t.beforeCaret = a, z
		}
	} else if n.marked {
		a, z = n.start, n.end
	}
	n.replace(a, z, s, false)
	t.composing, n.marked = false, false
	t.Sync()
	if plainTyping && t.Source.Buffer.Version() != beforeVersion && t.Source.Buffer.Len()-beforeLen == len(s)-(max(anchor, caret)-min(anchor, caret)) {
		n.typing = &typesettingTyping{t.Source.Buffer.Version(), anchor, caret, t.Source.Cursor(), s, now}
	}
}

func (n *typesettingInput) SetMarkedText(r *ui.TextInputRange, s string, selected ui.TextInputRange) {
	t := n.owner
	n.ended = nil
	n.typing = nil
	if s == "" {
		t.CancelComposition()
		return
	}
	if !t.composing {
		t.Sync()
		t.beforeAnchor, t.beforeCaret = t.Document.Selection()
		t.compositionAnchor, t.compositionCaret = t.beforeAnchor, t.beforeCaret
	}
	a, z := t.Document.Selection()
	if r != nil {
		a, z = nativeRange(t.Document.Text().String(), *r)
		if !t.composing {
			t.beforeAnchor, t.beforeCaret = a, z
			t.compositionAnchor, t.compositionCaret = a, z
		}
	} else if n.marked {
		a, z = n.start, n.end
	}
	t.composing, n.marked = true, true
	n.start, n.end = min(a, z), min(a, z)+utf8.RuneCountInString(s)
	n.replace(a, z, s, true)
	sa, sz := nativeRange(s, selected)
	t.Document.SetSelection(n.start+sa, n.start+sz)
}

func (n *typesettingInput) UnmarkText() {
	if !n.marked {
		return
	}
	t := n.owner
	r, _ := n.MarkedRange()
	a, z := t.compositionAnchor, t.compositionCaret
	before := t.Source.Buffer.Version()
	n.clearMark()
	t.composing, n.marked = false, false
	t.Sync()
	n.ended = &endedComposition{t.Source.Buffer.Version(), r, a, z, t.Source.Buffer.Version() != before}
}

func (n *typesettingInput) clearMark() {
	if !n.marked {
		return
	}
	n.owner.Document.Edit(func(e *richtext.Edit) {
		e.Format(n.start, n.end, func(a *richtext.Attributes) { a.Underline = false })
	})
}

func (n *typesettingInput) BoundsForRange(_ ui.TextInputRange) (ui.Rect, ui.TextInputRange, bool) {
	// MyGo supplies the shaped caret's actual geometry, including heading
	// sizes, wrapped lines and scroll. Returning the actual collapsed range
	// is allowed by TextInputClient and keeps candidate windows by the caret.
	r, ok := n.owner.Document.CaretRect()
	selection := n.Selection()
	caret := selection.Range.End
	if selection.Reversed {
		caret = selection.Range.Start
	}
	return r, ui.TextInputRange{Start: caret, End: caret}, ok
}

func (n *typesettingInput) IndexForPoint(point ui.Point) (int, bool) {
	t := n.owner
	for _, box := range t.Document.VisibleParagraphs() {
		if point.Y < box.Top || point.Y > box.Top+box.Height {
			continue
		}
		start, end := t.Document.Text().ParagraphRange(box.Start)
		s := t.Document.Text().Slice(start, end).String()
		size, weight := n.font.Size, 400
		family := n.font.Family
		line, exists := t.projection.shownLine(box.Index)
		if exists {
			level := line.heading
			if level > 0 {
				size *= [...]float32{1, 2, 1.45, 1.2, 1.08, 1, 1}[level]
				weight = 600
			}
			if line.block.code > 0 {
				family, size = "monospace", n.font.Size*0.9
			}
		}
		// Hit testing is used by native text services. The editor's ordinary
		// pointer events still use richtext's retained layout directly.
		indent, right := float32(0), float32(0)
		if exists && line.list.kind != NoList {
			indent = float32(line.list.depth+1) * n.gutter
		}
		if exists && line.block.quote > 0 {
			indent = float32(line.block.quote) * n.gutter
		}
		if exists && line.block.code > 0 {
			indent, right = 16, 16
		}
		font := n.font
		font.Family, font.Size = family, size
		layout := ui.ShapeRichText([]ui.Span{{Text: s, Font: family, Size: size, Weight: weight}}, font, max(1, n.width-indent-right))
		pos := layout.PositionAt(ui.Point{X: point.X - indent, Y: point.Y - box.Top})
		return nativeOffset(t.Document.Text().String(), start) + pos.Index, true
	}
	return 0, false
}
