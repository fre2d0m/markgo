package mdeditor

import (
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"
	"github.com/egoist/mygo/ui/richtext"
)

// A freshly projected caret retains its exact source side of a hidden marker.
// Pointer placement instead chooses the right side. This lets typing continue
// inside a format, and an arrow leave it without consuming a visible character.
func (t *Typesetter) sourceCaret(at int) int {
	sel := t.Source.Selection()
	line := t.projection.lines[t.projection.sourceLine(sel.Head)]
	if t.caretAffinity && sel.Empty() && sel.Head >= line.content && !line.block.hidden && t.projection.viewOffset(sel.Head) == at {
		return sel.Head
	}
	return t.projection.sourceOffset(at)
}

func (t *Typesetter) inlineArrow(key ui.Key) bool {
	a, z := t.Document.Selection()
	if a != z {
		return false
	}
	at := t.sourceCaret(a)
	line := t.projection.sourceLine(at)
	for _, h := range t.projection.inlineLines[line].holes {
		next := at
		if key == ui.KeyRight && at >= h.start && at < h.end {
			next = h.end
		}
		if key == ui.KeyLeft && at > h.start && at <= h.end {
			next = h.start
		}
		if next != at {
			t.Source.Select(next, next)
			t.FromSource()
			return true
		}
	}
	t.caretAffinity = false
	return false
}

// InlineKind is a source-backed inline formatting command.
type InlineKind uint8

const (
	InlineBold InlineKind = iota + 1
	InlineItalic
	InlineStrike
	InlineCode
	InlineLink
)

type typesetInline struct {
	kind                            InlineKind
	start, content, contentEnd, end int
	destination, title              string
	reference                       bool
}

type inlineHole struct{ start, end int }
type inlineLine struct {
	holes []inlineHole
	marks []int
}

func (p *typesetProjection) projectInlines(source string, frontEnd int) {
	p.inlines = p.inlines[:0]
	clear(p.inlineLines)
	if p.inlineLines == nil {
		p.inlineLines = map[int]inlineLine{}
	}
	if !strings.ContainsAny(source, "*_~`[") {
		return
	}
	for i := 1; i < p.parsed.Len(); i++ {
		b := p.parsed.Block(i)
		if b.Start < frontEnd || (b.Kind != parser.Paragraph && b.Kind != parser.Heading) {
			continue
		}
		first, last := p.sourceLine(b.Start), p.sourceLine(b.End)
		allowed := true
		for line := first; line <= last; line++ {
			l := p.lines[line]
			if l.block.code > 0 || (b.Parent != 0 && l.list.kind == NoList && l.block.quote == 0) || (b.Kind == parser.Heading && l.heading == 0) {
				allowed = false
				break
			}
		}
		if !allowed {
			continue
		}
		if !strings.ContainsAny(source[b.Start:b.End], "*_~`[") {
			continue
		}
		in := p.parsed.Inlines(i)
		for j := 0; j < len(in); j++ {
			x := in[j]
			m := typesetInline{start: x.Start, end: x.End}
			raw := source[x.Start:x.End]
			n := 0
			switch x.Kind {
			case parser.Strong:
				m.kind, n = InlineBold, 2
			case parser.Emphasis:
				m.kind, n = InlineItalic, 1
			case parser.Strikethrough:
				m.kind, n = InlineStrike, 1
				if strings.HasPrefix(raw, "~~") {
					n = 2
				}
			case parser.Code:
				if strings.Contains(raw, "\n") {
					continue
				}
				m.kind = InlineCode
				n = len(raw) - len(strings.TrimLeft(raw, "`"))
			case parser.Link:
				if x.Auto || x.Pending || strings.Contains(raw, "\n") || !strings.HasPrefix(raw, "[") || j+1 >= x.Next {
					j = x.Next - 1
					continue
				}
				lastChild := j + 1
				for next := in[lastChild].Next; next < x.Next; next = in[next].Next {
					lastChild = next
				}
				m.kind, m.content, m.contentEnd = InlineLink, x.Start+1, in[lastChild].End
				if m.contentEnd >= x.End || source[m.contentEnd] != ']' {
					j = x.Next - 1
					continue
				}
				m.destination, m.title = x.Dest, x.Title
				m.reference = m.contentEnd+1 >= m.end || source[m.contentEnd+1] != '('
			case parser.Image, parser.HTML, parser.Math:
				j = x.Next - 1
				continue
			default:
				continue
			}
			if n > 0 {
				if len(raw) < 2*n || raw[:n] != raw[len(raw)-n:] {
					continue
				}
				m.content, m.contentEnd = x.Start+n, x.End-n
				if m.kind == InlineCode && m.contentEnd-m.content >= 2 {
					body := source[m.content:m.contentEnd]
					if body[0] == ' ' && body[len(body)-1] == ' ' && strings.Trim(body, " ") != "" {
						m.content++
						m.contentEnd--
					}
				}
			}
			if m.content >= m.contentEnd {
				continue
			}
			idx := len(p.inlines)
			p.inlines = append(p.inlines, m)
			for line := p.sourceLine(m.start); line <= p.sourceLine(m.end); line++ {
				v := p.inlineLines[line]
				v.marks = append(v.marks, idx)
				for _, hole := range []inlineHole{{m.start, m.content}, {m.contentEnd, m.end}} {
					a, z := max(hole.start, p.lines[line].content), min(hole.end, p.lines[line].end)
					if z > a {
						v.holes = append(v.holes, inlineHole{a, z})
					}
				}
				p.inlineLines[line] = v
			}
		}
	}
	for line, v := range p.inlineLines {
		slices.SortFunc(v.holes, func(a, b inlineHole) int { return a.start - b.start })
		holes := v.holes[:0]
		for _, h := range v.holes {
			if n := len(holes); n > 0 && h.start <= holes[n-1].end {
				holes[n-1].end = max(holes[n-1].end, h.end)
			} else {
				holes = append(holes, h)
			}
		}
		v.holes = holes
		p.inlineLines[line] = v
	}
}

func (p *typesetProjection) inlineEligible(line int) bool {
	l := p.lines[line]
	return l.block.code == 0 && !l.block.hidden && (l.list.kind != NoList || l.block.quote > 0 || p.canList(line))
}

func (p *typesetProjection) inlineSourceOffset(line, at int, right bool) int {
	l := p.lines[line]
	start := l.content
	for _, h := range p.inlineLines[line].holes {
		n := utf8.RuneCountInString(p.source[start:h.start])
		if at < n || (at == n && !right) {
			return start + runeByte(p.source[start:h.start], at)
		}
		at -= n
		start = h.end
	}
	return start + runeByte(p.source[start:l.end], at)
}

func (p *typesetProjection) inlineViewOffset(line, at int) int {
	l := p.lines[line]
	start, n := l.content, 0
	for _, h := range p.inlineLines[line].holes {
		if at <= h.start {
			return n + utf8.RuneCountInString(p.source[start:max(start, at)])
		}
		n += utf8.RuneCountInString(p.source[start:h.start])
		if at <= h.end {
			return n
		}
		start = h.end
	}
	return n + utf8.RuneCountInString(p.source[start:max(start, min(at, l.end))])
}

func (p *typesetProjection) sourceRange(a, z int) (int, int) {
	a, z = min(a, z), max(a, z)
	if a == z {
		at := p.sourceOffset(a)
		return at, at
	}
	start := p.sourceOffset(a)
	i := max(0, sort.Search(len(p.visible), func(i int) bool { return p.lines[p.visible[i]].viewStart > z })-1)
	line := p.visible[i]
	end := p.inlineSourceOffset(line, z-p.lines[line].viewStart, false)
	return start, max(start, end)
}

func inlineAttributes(m typesetInline, a richtext.Attributes) richtext.Attributes {
	switch m.kind {
	case InlineBold:
		a.Weight = 700
	case InlineItalic:
		a.Italic = true
	case InlineStrike:
		a.Strikethrough = true
	case InlineCode:
		a.Code = true
	case InlineLink:
		a.Underline = true // editing never opens a browser implicitly
	}
	return a
}

func (t *Typesetter) invalidateInlineStyles(first, last, next int) {
	shifted := make(map[int]uint64, len(t.inlinePainted))
	for i, paint := range t.inlinePainted {
		if i < first {
			shifted[i] = paint
		} else if i > last {
			shifted[i+next-last] = paint
		}
	}
	t.inlinePainted = shifted
	if t.inlineDirty == nil {
		t.inlineDirty = map[int]bool{}
	}
	for i := first; i <= next; i++ {
		t.inlineDirty[i] = true
	}
}

func (t *Typesetter) projectInlineStyles() {
	p := &t.projection
	next := make(map[int]uint64, len(p.inlineLines))
	t.Document.Edit(func(e *richtext.Edit) {
		for paragraph, lineIndex := range p.visible {
			marked := len(p.inlineLines[lineIndex].marks) > 0
			if !marked && t.inlinePainted[paragraph] == 0 && !t.inlineDirty[paragraph] {
				continue
			}
			line := p.lines[lineIndex]
			// Relative source boundaries are stable when earlier paragraphs
			// change. Dirty paragraphs still rebuild, even for equal-length
			// edits or a cancelled preedit with identical visible text.
			signature := uint64(14695981039346656037)
			for _, index := range p.inlineLines[lineIndex].marks {
				m := p.inlines[index]
				for _, n := range []int{int(m.kind), max(m.start, line.content) - line.content, max(m.content, line.content) - line.content, min(m.contentEnd, line.end) - line.content, min(m.end, line.end) - line.content} {
					signature = (signature ^ uint64(n)) * 1099511628211
				}
			}
			if marked {
				next[paragraph] = signature
			}
			if marked && t.inlinePainted[paragraph] == signature && !t.inlineDirty[paragraph] {
				continue
			}
			a, z := line.viewStart, line.viewStart+line.runes
			cuts := []int{a, z}
			for _, index := range p.inlineLines[lineIndex].marks {
				m := p.inlines[index]
				cuts = append(cuts, max(a, p.viewOffset(m.content)), min(z, p.viewOffset(m.contentEnd)))
			}
			slices.Sort(cuts)
			cuts = slices.Compact(cuts)
			for i := 1; i < len(cuts); i++ {
				start, end := cuts[i-1], cuts[i]
				if start == end {
					continue
				}
				attrs := richtext.Attributes{}
				for _, index := range p.inlineLines[lineIndex].marks {
					m := p.inlines[index]
					if p.viewOffset(m.content) <= start && p.viewOffset(m.contentEnd) >= end {
						attrs = inlineAttributes(m, attrs)
					}
				}
				e.Format(start, end, func(a *richtext.Attributes) { *a = attrs })
			}
		}
	})
	t.inlinePainted = next
	clear(t.inlineDirty)
}
