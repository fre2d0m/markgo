package mdeditor

import (
	"strings"

	"github.com/egoist/mygo/ui/markdown/parser"
)

type typesetBlockLine struct {
	quote, code, prefix int // quote depth; code index + 1; hidden prefix bytes
	hidden, virtual     bool
}

type typesetCode struct {
	open, close, first, last int // physical lines; close is -1 for an open fence
	start, end               int // source range, without the following newline
	bodyStart, bodyEnd       int
	marker, info             string // original fence and unmodified info string
}

func quotePrefix(raw string, depth int) int {
	at, _ := quoteMarks(raw, depth)
	return at
}

func quoteMarks(raw string, depth int) (int, int) {
	at := 0
	for count := range depth {
		start := at
		for at < len(raw) && raw[at] == ' ' && at-start < 3 {
			at++
		}
		if at >= len(raw) || raw[at] != '>' {
			return start, count // a lazy continuation has no marker of its own
		}
		at++
		if at < len(raw) && (raw[at] == ' ' || raw[at] == '\t') {
			at++
		}
	}
	return at, depth
}

func (p *typesetProjection) projectBlocks(source string, frontEnd int) {
	if cap(p.blockLines) < len(p.lines) {
		p.blockLines = make([]typesetBlockLine, len(p.lines))
	} else {
		p.blockLines = p.blockLines[:len(p.lines)]
		clear(p.blockLines)
	}
	p.codes = p.codes[:0]
	doc := p.parsed
	for i := 1; i < doc.Len(); i++ {
		b := doc.Block(i)
		if b.Start < frontEnd {
			continue
		}
		if b.Kind == parser.Quote && b.Parent == 0 {
			allowed := true
			for child := i; child < b.Next; child++ {
				x := doc.Block(child)
				if (x.Kind != parser.Quote && x.Kind != parser.Paragraph) || x.Alert != parser.NoAlert {
					allowed = false
					break
				}
			}
			if allowed {
				for child := i; child < b.Next; child++ {
					x := doc.Block(child)
					if x.Kind != parser.Quote {
						continue
					}
					depth := 1
					for parent := x.Parent; parent > 0; parent = doc.Block(parent).Parent {
						depth++
					}
					if depth > 8 {
						continue
					}
					last := p.sourceLine(x.End)
					// The parser's range can end at the last content line, before
					// trailing empty quote markers. They still need a caret/exit.
					for last+1 < len(p.lines) {
						raw := source[p.lines[last+1].start:p.lines[last+1].end]
						prefix, count := quoteMarks(raw, 64)
						if count < depth || strings.TrimSpace(raw[prefix:]) != "" {
							break
						}
						last++
					}
					for line := p.sourceLine(x.Start); line <= last; line++ {
						raw := source[p.lines[line].start:p.lines[line].end]
						p.blockLines[line] = typesetBlockLine{quote: depth, prefix: quotePrefix(raw, depth)}
					}
				}
			}
			i = b.Next - 1
			continue
		}
		if b.Kind != parser.CodeBlock || !b.Fenced || b.Parent != 0 {
			continue
		}
		open, close := p.sourceLine(b.Start), -1
		raw := source[p.lines[open].start:p.lines[open].end]
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		at := indent
		for at < len(raw) && raw[at] == raw[indent] {
			at++
		}
		marker := raw[indent:at]
		last := p.sourceLine(b.End)
		if b.Open {
			last = len(p.lines) - 1
		} else if last > open {
			closing := strings.TrimSpace(source[p.lines[last].start:p.lines[last].end])
			if len(closing) >= len(marker) && strings.Trim(closing, marker[:1]) == "" {
				close = last
				last--
			}
		}
		code := typesetCode{open: open, close: close, first: open + 1, last: last, start: p.lines[open].start, end: b.End, marker: marker, info: strings.TrimSpace(raw[at:])}
		if b.Open {
			code.end = len(source)
		}
		idx := len(p.codes) + 1
		p.blockLines[open] = typesetBlockLine{code: idx, hidden: true, prefix: len(raw)}
		if close >= 0 {
			p.blockLines[close] = typesetBlockLine{code: idx, hidden: true, prefix: p.lines[close].end - p.lines[close].start}
		}
		if code.first > code.last {
			// An empty fenced block still needs one editable paragraph, without
			// changing the file merely to create a place for the caret.
			code.first, code.last = open, open
			code.bodyStart, code.bodyEnd = p.lines[open].end, p.lines[open].end
			p.blockLines[open].hidden, p.blockLines[open].virtual = false, true
		} else {
			code.bodyStart, code.bodyEnd = p.lines[code.first].start, p.lines[code.last].end
			for line := code.first; line <= code.last; line++ {
				p.blockLines[line] = typesetBlockLine{code: idx}
			}
		}
		p.codes = append(p.codes, code)
	}
}

func (p *typesetProjection) shownLine(paragraph int) (typesetLine, bool) {
	if paragraph < 0 || paragraph >= len(p.visible) {
		return typesetLine{}, false
	}
	return p.lines[p.visible[paragraph]], true
}

func (p *typesetProjection) codeAt(line int) (typesetCode, bool) {
	if line < 0 || line >= len(p.blockLines) || p.blockLines[line].code == 0 {
		return typesetCode{}, false
	}
	return p.codes[p.blockLines[line].code-1], true
}
