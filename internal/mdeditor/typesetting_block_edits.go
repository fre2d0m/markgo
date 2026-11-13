package mdeditor

import (
	"slices"
	"strings"

	"github.com/egoist/mygo/ui/editor"
)

type projectedInsertion struct {
	raw, text, prefix, quote string
}

func (v projectedInsertion) offset(runes int) int {
	at := runeByte(v.raw, runes)
	return len(v.prefix) + at + strings.Count(v.raw[:at], "\n")*len(v.quote)
}

func (p *typesetProjection) prepareInsertion(a, z int, text string, whole bool) projectedInsertion {
	v := projectedInsertion{raw: text, text: text}
	if whole || text == "" {
		return v
	}
	first, last := p.lines[p.sourceLine(a)], p.lines[p.sourceLine(z)]
	if first.block.virtual && a == z {
		v.prefix = "\n"
	}
	if first.block.quote > 0 && first.block.quote == last.block.quote {
		v.quote = strings.Repeat("> ", first.block.quote)
	}
	v.text = v.prefix + strings.ReplaceAll(text, "\n", "\n"+v.quote)
	return v
}

func fenceRun(line string, marker byte) int {
	at := 0
	for at < min(3, len(line)) && line[at] == ' ' {
		at++
	}
	start := at
	for at < len(line) && line[at] == marker {
		at++
	}
	return at - start
}

func (p *typesetProjection) lineAfter(i int) int {
	if i+1 < len(p.lines) {
		return p.lines[i+1].start
	}
	return len(p.source)
}

// Keep paired fence edits in the same history entry as the content edit.
// Crossing one boundary unwraps the surviving content; typing a literal fence
// inside code extends its delimiters instead of silently ending the block.
func (t *Typesetter) applyProjectedEdit(a, z int, text string, anchor, caret int) {
	p := &t.projection
	var extra []editor.Edit
	for _, code := range p.codes {
		open, close := p.lines[code.open], typesetLine{}
		if z > a && open.block.virtual && a == code.bodyStart {
			// An empty block's display caret is at the end of the opener.
			// A selection leaving it must remove that hidden prefix too.
			extra = append(extra, editor.Edit{Start: open.start, End: a})
		}
		openHit := a < p.lineAfter(code.open) && z > open.start
		closeHit := false
		if code.close >= 0 {
			close = p.lines[code.close]
			closeHit = a < p.lineAfter(code.close) && z > close.start
		}
		if z > a && openHit != closeHit {
			if closeHit {
				extra = append(extra, editor.Edit{Start: open.start, End: p.lineAfter(code.open)})
			} else if code.close >= 0 {
				extra = append(extra, editor.Edit{Start: close.start, End: p.lineAfter(code.close)})
			}
			continue
		}
		if a < code.bodyStart || z > code.bodyEnd {
			continue
		}
		body := p.source[code.bodyStart:a] + text + p.source[z:code.bodyEnd]
		length := len(code.marker)
		for line := range strings.SplitSeq(body, "\n") {
			length = max(length, fenceRun(line, code.marker[0])+1)
		}
		if length <= len(code.marker) {
			continue
		}
		at := open.start + strings.Index(p.source[open.start:open.end], code.marker)
		extra = append(extra, editor.Edit{Start: at, End: at + len(code.marker), Text: strings.Repeat(code.marker[:1], length)})
		if code.close >= 0 {
			raw := p.source[close.start:close.end]
			at = close.start + strings.IndexByte(raw, code.marker[0])
			n := fenceRun(raw, code.marker[0])
			if n < length {
				extra = append(extra, editor.Edit{Start: at, End: at + n, Text: strings.Repeat(code.marker[:1], length)})
			}
		}
	}
	t.Source.Select(a, z)
	if t.beforeAnchor > t.beforeCaret {
		t.Source.Select(z, a)
	}
	if len(extra) == 0 {
		if text == "" {
			t.Source.Run(editor.Delete)
		} else {
			t.Source.Insert(text)
		}
		t.Source.Select(anchor, caret)
		return
	}
	slices.SortFunc(extra, func(a, b editor.Edit) int { return a.Start - b.Start })
	delta := len(text) - (z - a)
	move := func(pos int) int {
		shift := 0
		for _, e := range extra {
			start, end := e.Start, e.End
			if start >= z {
				start, end = start+delta, end+delta
			}
			if pos < start {
				break
			}
			if pos < end {
				return start + shift + len(e.Text)
			}
			shift += len(e.Text) - (end - start)
		}
		return pos + shift
	}
	// Start with the source editor's edit so the transaction retains its
	// before-selection. The paired syntax changes then join that same step.
	t.Source.Buffer.Transact(func() {
		if text == "" {
			t.Source.Run(editor.Delete)
		} else {
			t.Source.Insert(text)
		}
		edits := slices.Clone(extra)
		for i := range edits {
			if edits[i].Start >= z {
				edits[i].Start += delta
				edits[i].End += delta
			}
		}
		t.Source.Buffer.Edit(edits...)
		t.Source.Select(move(anchor), move(caret))
	})
}

func (p *typesetProjection) wholeCodeRange(a, z int) (int, int) {
	for _, code := range p.codes {
		if a <= code.bodyStart && z >= code.bodyEnd {
			a, z = min(a, code.start), max(z, code.end)
		}
	}
	return a, z
}
