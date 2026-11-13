package mdeditor

import (
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui/markdown/parser"
)

type typesetLine struct {
	start, content, end int // UTF-8 source offsets
	viewStart, viewByte int // projected rune and byte offsets
	runes, heading      int // visible length; ATX level
	list                typesetListStyle
}

type typesetProjection struct {
	source, text string
	lines        []typesetLine
	parsed       *parser.Document
	styleVersion int
	listLines    []typesetListLine
	items        []typesetListItem
	listNumber   int
}

func projectTypesetting(source string) typesetProjection {
	var p typesetProjection
	p.update(source)
	return p
}

// changedText bounds one replacement in UTF-8 bytes. The common prefix stops
// at limit so repeated text cannot move an edit past the original selection.
// Neither end may split a rune (different CJK characters can share bytes).
func changedText(old, next string, limit int) (start, oldEnd, newEnd int) {
	for start < min(len(old), len(next), limit) && old[start] == next[start] {
		start++
	}
	for start > 0 && start < len(old) && !utf8.RuneStart(old[start]) {
		start--
	}
	oldEnd, newEnd = len(old), len(next)
	for oldEnd > start && newEnd > start && old[oldEnd-1] == next[newEnd-1] {
		oldEnd--
		newEnd--
	}
	for oldEnd < len(old) && !utf8.RuneStart(old[oldEnd]) {
		oldEnd++
		newEnd++
	}
	return
}

// update reparses the affected Markdown blocks using the pinned parser's
// incremental API, replaces the affected physical lines, and retains the
// others. A fence or front-matter edit can change headings much farther away:
// reconcile all top-level block kinds before narrowing the display change.
// Offsets after the edit still shift linearly; unchanged text is not decoded
// again, nor is a document-sized per-rune source map allocated.
func (p *typesetProjection) update(source string) {
	if p.parsed == nil {
		p.parsed = parser.New(parser.Default)
		p.lines = []typesetLine{{}}
		p.styleVersion = 1
	} else if source == p.source {
		return
	}
	start, oldEnd, newEnd := changedText(p.source, source, len(p.source))
	first, last := p.sourceLine(start), p.sourceLine(oldEnd)
	oldFirst, oldLast := p.lines[first], p.lines[last]
	oldStartByte, oldEndByte := oldFirst.viewByte, oldLast.viewByte+oldLast.end-oldLast.content
	delta := newEnd - oldEnd
	regionStart, regionEnd := oldFirst.start, oldLast.end+delta
	region := source[regionStart:regionEnd]
	replaced := make([]typesetLine, 0, strings.Count(region, "\n")+1)
	offset := regionStart
	for line := range strings.SplitSeq(region, "\n") {
		heading := 0
		if first+len(replaced) <= last {
			heading = p.lines[first+len(replaced)].heading
		}
		list := typesetListStyle{}
		if first+len(replaced) <= last {
			list = p.lines[first+len(replaced)].list
		}
		replaced = append(replaced, typesetLine{start: offset, content: offset, end: offset + len(line), heading: heading, list: list})
		offset += len(line) + 1
	}
	styleChanged := len(replaced) != last-first+1
	p.lines = slices.Replace(p.lines, first, last+1, replaced...)
	newLast := first + len(replaced) - 1
	for i := newLast + 1; i < len(p.lines); i++ {
		p.lines[i].start += delta
		p.lines[i].content += delta
		p.lines[i].end += delta
	}
	p.parsed.Set(source)
	frontEnd := frontMatterEnd(source)
	p.projectLists(source, frontEnd)
	block, viewStart, viewByte := 1, 0, 0
	changedFirst, changedLast := first, newLast
	for i := range p.lines {
		line := &p.lines[i]
		for block < p.parsed.Len() && p.parsed.Block(block).Start < line.start {
			block = p.parsed.Block(block).Next
		}
		level := 0
		if block < p.parsed.Len() {
			b := p.parsed.Block(block)
			if b.Start == line.start && b.Kind == parser.Heading && b.Start >= frontEnd {
				level = b.Level
			}
		}
		touched := i >= first && i <= newLast
		content := line.content
		if touched || level != line.heading {
			content = line.start
			if level > 0 {
				if prefix := headingPrefix.FindStringIndex(source[line.start:line.end]); prefix != nil {
					content += prefix[1]
				} else {
					level = 0 // Setext headings are still literal.
				}
			}
		}
		list := p.listLines[i]
		if list.style.kind != NoList {
			content = line.start + list.prefix
		} else if line.list.kind != NoList && level == 0 {
			content = line.start
		}
		if touched || level != line.heading || list.style != line.list || content != line.content {
			if !touched {
				oldStartByte = min(oldStartByte, line.viewByte)
				oldEndByte = max(oldEndByte, line.viewByte+line.end-line.content)
				changedFirst, changedLast = min(changedFirst, i), max(changedLast, i)
			}
			styleChanged = styleChanged || level != line.heading || list.style != line.list
			line.content, line.heading, line.list = content, level, list.style
			line.runes = utf8.RuneCountInString(source[content:line.end])
		}
		line.viewStart, line.viewByte = viewStart, viewByte
		viewStart += line.runes + 1
		viewByte += line.end - line.content + 1
	}
	if styleChanged {
		p.styleVersion++
	}
	var shown strings.Builder
	shown.Grow(p.lines[changedLast].end - p.lines[changedFirst].start)
	for i := changedFirst; i <= changedLast; i++ {
		if i != changedFirst {
			shown.WriteByte('\n')
		}
		line := p.lines[i]
		shown.WriteString(source[line.content:line.end])
	}
	p.text = p.text[:oldStartByte] + shown.String() + p.text[oldEndByte:]
	p.source = source
}

func (p *typesetProjection) len() int {
	line := p.lines[len(p.lines)-1]
	return line.viewStart + line.runes
}

func (p *typesetProjection) sourceLine(offset int) int {
	return max(0, sort.Search(len(p.lines), func(i int) bool { return p.lines[i].start > offset })-1)
}

// Maps are compact per-line records. At a heading's start, the source offset
// is after its hidden prefix, so joining across the preceding newline consumes
// that prefix as well. Only the selected line needs rune/byte conversion.
func (p *typesetProjection) sourceOffset(runeOffset int) int {
	runeOffset = max(0, min(runeOffset, p.len()))
	i := max(0, sort.Search(len(p.lines), func(i int) bool { return p.lines[i].viewStart > runeOffset })-1)
	line := p.lines[i]
	return line.content + runeByte(p.source[line.content:line.end], runeOffset-line.viewStart)
}

func (p *typesetProjection) viewOffset(byteOffset int) int {
	byteOffset = max(0, min(byteOffset, len(p.source)))
	line := p.lines[p.sourceLine(byteOffset)]
	return line.viewStart + utf8.RuneCountInString(p.source[line.content:max(line.content, min(byteOffset, line.end))])
}
