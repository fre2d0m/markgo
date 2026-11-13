package mdeditor

import "strings"

func (t *Typesetter) selectedLines() (int, int) {
	sel := t.Source.Selection()
	first, last := t.projection.sourceLine(sel.Start()), t.projection.sourceLine(sel.End())
	if !sel.Empty() && last > first && sel.End() <= t.projection.lines[last].content {
		last--
	}
	return first, last
}

// Quote toggles one level of quotation across ordinary selected paragraphs.
func (t *Typesetter) Quote() {
	t.ToSource()
	first, last := t.selectedLines()
	remove := true
	for i := first; i <= last; i++ {
		line := t.projection.lines[i]
		if line.block.quote == 0 && (!t.projection.canList(i) || line.list.kind != NoList || line.heading > 0) {
			t.Notice = "Choose paragraphs or ordinary quotations. Other Markdown stays unchanged."
			return
		}
		remove = remove && line.block.quote > 0
	}
	t.changeQuote(first, last, remove)
}

func (t *Typesetter) changeQuote(first, last int, remove bool) {
	lines, depths := map[int]string{}, map[int]int{}
	for i := first; i <= last; i++ {
		line := t.projection.lines[i]
		depth := line.block.quote + 1
		if remove {
			depth = max(0, line.block.quote-1)
		}
		if depth > 8 {
			t.Notice = "This preview supports up to eight quotation levels."
			return
		}
		content := line.content
		if line.heading > 0 {
			content = line.start // quoting a heading keeps its original Markdown
		}
		lines[i] = strings.Repeat("> ", depth) + t.projection.source[content:line.end]
		depths[i] = depth
	}
	for i := first; i <= last; i++ {
		if i == 0 || strings.TrimSpace(t.Source.Buffer.Line(i-1)) == "" {
			continue
		}
		previous := t.projection.lines[i-1].block.quote
		if depth, changed := depths[i-1]; changed {
			previous = depth
		}
		if previous > depths[i] {
			lines[i] = strings.TrimSpace(strings.Repeat("> ", depths[i])) + "\n" + lines[i]
		}
	}
	t.applyLineChanges(lines)
}

func (t *Typesetter) quoteEnter() bool {
	t.ToSource()
	sel := t.Source.Selection()
	i := t.projection.sourceLine(sel.Start())
	line := t.projection.lines[i]
	if line.block.quote == 0 {
		return false
	}
	if sel.Empty() && strings.TrimSpace(t.projection.source[line.content:line.end]) == "" {
		t.changeQuote(i, i, true)
		return true
	}
	// Insert owns quote prefixes for every inserted newline, including paste.
	t.Insert("\n")
	return true
}

func (t *Typesetter) indentQuote(remove bool) bool {
	t.ToSource()
	first, last := t.selectedLines()
	for i := first; i <= last; i++ {
		if t.projection.lines[i].block.quote == 0 {
			return false
		}
	}
	t.changeQuote(first, last, remove)
	return true
}
