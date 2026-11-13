package mdeditor

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui/markdown/parser"
	"github.com/egoist/mygo/ui/richtext"
)

// ListKind describes source-backed list commands, not rich-text-only formats.
type ListKind uint8

const (
	NoList ListKind = iota
	BulletedList
	NumberedList
	TaskList
	listContinuation
)

type typesetListStyle struct {
	kind    ListKind
	depth   int
	number  int
	checked bool
}

type typesetListLine struct {
	style         typesetListStyle
	prefix, owner int // bytes hidden; index in items plus one (zero means none)
}

type typesetListItem struct {
	line, end, parent, list int // physical line; source end; parent line; parser list
	prefix                  listPrefix
}

type listPrefix struct {
	indent, markerEnd, content int // relative source bytes, including whitespace
	task                       int // relative byte of space/x/X, or -1
	marker                     string
	delimiter                  byte
}

// readListPrefix runs only on parser-confirmed items. Code, HTML, front matter
// and quoted lists never become editable markers just because a line starts '-'.
func readListPrefix(line string) (p listPrefix, ok bool) {
	p.task = -1
	p.indent = len(line) - len(strings.TrimLeft(line, " \t"))
	i := p.indent
	if i == len(line) {
		return p, false
	}
	if strings.ContainsRune("-*+", rune(line[i])) {
		i++
	} else {
		for i < len(line) && line[i] >= '0' && line[i] <= '9' {
			i++
		}
		if i == p.indent || i-p.indent > 9 || i == len(line) || (line[i] != '.' && line[i] != ')') {
			return p, false
		}
		p.delimiter = line[i]
		i++
	}
	p.marker = line[p.indent:i]
	if i < len(line) && line[i] != ' ' && line[i] != '\t' {
		return p, false
	}
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	p.markerEnd = i
	if i+3 <= len(line) && line[i] == '[' && line[i+2] == ']' && strings.ContainsRune(" xX", rune(line[i+1])) && (i+3 == len(line) || line[i+3] == ' ' || line[i+3] == '\t') {
		p.task = i + 1
		i += 3
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
	}
	p.content = i
	return p, true
}

func (p *typesetProjection) projectLists(source string, frontEnd int) {
	if cap(p.listLines) < len(p.lines) {
		p.listLines = make([]typesetListLine, len(p.lines))
	} else {
		p.listLines = p.listLines[:len(p.lines)]
		clear(p.listLines)
	}
	p.items = p.items[:0]
	p.listNumber = 0
	doc := p.parsed
	ordinals := map[int]int{}
	// Walking lists and their direct children keeps Markdown's starting number
	// and decimal numbering at every depth, including non-1 starts and 1,1,1.
	for i := 1; i < doc.Len(); i++ {
		b := doc.Block(i)
		if b.Kind != parser.Item || b.Start < frontEnd {
			continue
		}
		list := doc.Block(b.Parent)
		number := 0
		if list.Ordered {
			number = list.Number
			if previous, ok := ordinals[b.Parent]; ok {
				number = previous + 1
			}
			// Literal complex items still count in their siblings' sequence.
			ordinals[b.Parent] = number
		}
		depth, parent, allowed := 0, -1, true
		for ancestor := b.Parent; ancestor > 0; ancestor = doc.Block(ancestor).Parent {
			a := doc.Block(ancestor)
			if a.Kind == parser.Item {
				depth++
				if parent < 0 {
					parent = p.sourceLine(a.Start)
				}
			} else if a.Kind != parser.List {
				allowed = false
			}
		}
		if !allowed || depth > 7 || parent == p.sourceLine(b.Start) || (parent >= 0 && p.listLines[parent].owner == 0) {
			continue
		}
		if i+1 < b.Next {
			switch doc.Block(i + 1).Kind {
			case parser.CodeBlock, parser.HTMLBlock, parser.Quote, parser.Table, parser.MathBlock, parser.ThematicBreak:
				continue // keep complex block-first items literal for this checkpoint
			}
		}
		lineIndex := p.sourceLine(b.Start)
		line := p.lines[lineIndex]
		prefix, ok := readListPrefix(source[line.start:line.end])
		if !ok {
			continue
		}
		kind := BulletedList
		if list.Ordered {
			kind = NumberedList
			// Item.Number is the original written number. The displayed number
			// instead follows the preceding sibling's semantic ordinal.
		}
		if prefix.task >= 0 {
			kind = TaskList
		}
		if kind == NumberedList {
			p.listNumber = max(p.listNumber, number)
		}
		p.items = append(p.items, typesetListItem{lineIndex, b.End, parent, b.Parent, prefix})
		style := typesetListStyle{kind, depth, number, prefix.task >= 0 && source[line.start+prefix.task] != ' '}
		p.listLines[lineIndex] = typesetListLine{style, prefix.content, len(p.items)}
		// Paragraph continuation lines share the item's hanging indent. Their
		// own Markdown indentation is hidden; code/fences stay literal.
		for child := i + 1; child < b.Next; child = doc.Block(child).Next {
			if doc.Block(child).Kind != parser.Paragraph {
				continue
			}
			for k := 0; k < doc.LineCount(child); k++ {
				_, offset := doc.Line(child, k)
				idx := p.sourceLine(offset)
				if idx == lineIndex {
					continue
				}
				continuation := style
				continuation.kind = listContinuation
				p.listLines[idx] = typesetListLine{continuation, offset - p.lines[idx].start, len(p.items)}
			}
		}
	}
}

func (t *Typesetter) projectParagraphStyles() {
	if t.appliedStyle == t.projection.styleVersion {
		return
	}
	t.Document.Edit(func(e *richtext.Edit) {
		for _, i := range t.projection.visible {
			line := t.projection.lines[i]
			style := richtext.ParagraphStyle{}
			if line.list.kind != NoList {
				style.Indent = line.list.depth + 1
			}
			if line.block.quote > 0 {
				style.Indent = line.block.quote
			}
			style.CodeBlock = line.block.code > 0
			if e.Text().ParagraphStyleAt(line.viewStart) != style {
				e.FormatParagraphs(line.viewStart, line.viewStart, func(s *richtext.ParagraphStyle) { *s = style })
			}
		}
	})
	t.appliedStyle = t.projection.styleVersion
}

func (p listPrefix) next(raw string, number int) string {
	marker := p.marker
	if p.delimiter != 0 {
		marker = strconv.Itoa(number+1) + string(p.delimiter)
	}
	space := raw[p.indent+len(p.marker) : p.markerEnd]
	if space == "" {
		space = " "
	}
	text := raw[:p.indent] + marker + space
	if p.task >= 0 {
		text += "[ ] "
	}
	return text
}
