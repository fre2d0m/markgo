package mdeditor

import (
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown/parser"
)

func (p *typesetProjection) itemAt(line int) (typesetListItem, bool) {
	if line < 0 || line >= len(p.listLines) || p.listLines[line].owner == 0 {
		return typesetListItem{}, false
	}
	return p.items[p.listLines[line].owner-1], true
}

func (t *Typesetter) listEnter() bool {
	t.ToSource()
	sel := t.Source.Selection()
	lineIndex := t.projection.sourceLine(sel.Start())
	item, ok := t.projection.itemAt(lineIndex)
	if !ok {
		return false
	}
	first := t.projection.lines[item.line]
	if sel.Empty() && lineIndex == item.line && strings.TrimSpace(t.projection.source[first.content:first.end]) == "" && item.end <= first.end {
		return t.IndentList(true)
	}
	if first.list.number >= 999999999 {
		t.Notice = "Markdown list numbers can contain up to nine digits."
		return true
	}
	prefix := item.prefix.next(t.projection.source[first.start:first.end], first.list.number)
	// An empty first child cannot interrupt its parent's paragraph (a dash
	// would even become a Setext underline). Separate it without renumbering
	// or rewriting other items, in the same undo step as Return.
	separate := item.parent >= 0 && lineIndex == item.line && sel.Start() == first.content && strings.TrimSpace(t.Source.Buffer.Line(item.line-1)) != ""
	for _, previous := range t.projection.items {
		if previous.line < item.line && previous.list == item.list {
			separate = false
		}
	}
	t.Source.Buffer.Transact(func() {
		t.Source.Insert("\n" + prefix)
		if separate {
			caret := t.Source.Selection()
			t.Source.Buffer.Edit(editor.Edit{Start: first.start, End: first.start, Text: "\n"})
			t.Source.Select(caret.Anchor+1, caret.Head+1)
		}
	})
	t.FromSource()
	t.Notice = ""
	return true
}

func indentColumns(s string) int {
	n := 0
	for _, c := range s {
		if c == '\t' {
			n += 4 - n%4
		} else {
			n++
		}
	}
	return n
}

func shiftListIndent(line string, columns int) string {
	if columns == 0 {
		return line
	}
	at := len(line) - len(strings.TrimLeft(line, " \t"))
	// Prepending spaces to a tab can leave its visual column unchanged.
	// Normalize only the indentation explicitly moved by this command.
	width := max(0, indentColumns(line[:at])+columns)
	return strings.Repeat(" ", width) + line[at:]
}

// applyListLines changes only affected prefixes/indentation. Buffer.Edit keeps
// all replacements in one source-history step; selections keep their direction.
func (t *Typesetter) applyListLines(lines map[int]string) {
	indices := make([]int, 0, len(lines))
	for i := range lines {
		indices = append(indices, i)
	}
	slices.Sort(indices)
	var edits []editor.Edit
	for _, i := range indices {
		line := t.projection.lines[i]
		old, next := t.projection.source[line.start:line.end], lines[i]
		if old == next {
			continue
		}
		a, z, n := changedText(old, next, len(old))
		edits = append(edits, editor.Edit{Start: line.start + a, End: line.start + z, Text: next[a:n]})
	}
	if len(edits) == 0 {
		return
	}
	sel := t.Source.Selection()
	move := func(pos int) int {
		delta := 0
		for _, e := range edits {
			if pos < e.Start {
				break
			}
			if pos < e.End {
				return e.Start + delta + len(e.Text)
			}
			delta += len(e.Text) - (e.End - e.Start)
		}
		return pos + delta
	}
	t.Source.Buffer.Edit(edits...)
	t.Source.Select(move(sel.Anchor), move(sel.Head))
	t.FromSource()
	t.Notice = ""
}

func (t *Typesetter) selectedListItems() ([]typesetListItem, bool) {
	sel := t.Source.Selection()
	first, last := t.projection.sourceLine(sel.Start()), t.projection.sourceLine(sel.End())
	if !sel.Empty() && last > first && sel.End() <= t.projection.lines[last].content {
		last--
	}
	var chosen []typesetListItem
	for line := first; line <= last; line++ {
		item, ok := t.projection.itemAt(line)
		if !ok {
			if strings.TrimSpace(t.Source.Buffer.Line(line)) == "" {
				continue
			}
			return nil, false
		}
		if len(chosen) > 0 && item.line <= t.projection.sourceLine(chosen[len(chosen)-1].end) {
			continue // selected parent already carries its complete subtree
		}
		chosen = append(chosen, item)
	}
	return chosen, len(chosen) > 0
}

// IndentList moves selected items and their descendants together. Outdenting
// at the outermost level turns them into paragraphs and promotes child lists.
func (t *Typesetter) IndentList(outdent bool) bool {
	t.ToSource()
	chosen, ok := t.selectedListItems()
	if !ok {
		return false
	}
	first := chosen[0]
	for _, item := range chosen {
		if item.parent != first.parent {
			t.Notice = "Select items at the same level to change their indentation."
			return true
		}
	}
	if outdent && first.parent < 0 {
		t.setListItems(chosen, NoList)
		return true
	}
	raw := t.Source.Buffer.Line(first.line)
	currentIndent := indentColumns(raw[:first.prefix.indent])
	delta := 0
	if outdent {
		parent, _ := t.projection.itemAt(first.parent)
		parentRaw := t.Source.Buffer.Line(parent.line)
		delta = indentColumns(parentRaw[:parent.prefix.indent]) - currentIndent
	} else {
		var previous typesetListItem
		found := false
		for _, item := range t.projection.items {
			if item.line >= first.line {
				break
			}
			if item.parent == first.parent {
				previous, found = item, true
			}
		}
		if !found || strings.TrimSpace(t.projection.source[previous.end:t.projection.lines[first.line].start]) != "" {
			t.Notice = "An item needs a preceding item to nest under."
			return true
		}
		previousRaw := t.Source.Buffer.Line(previous.line)
		delta = indentColumns(previousRaw[:previous.prefix.markerEnd]) - currentIndent
		if delta <= 0 {
			delta = 2
		}
		for _, selected := range chosen {
			for _, item := range t.projection.items {
				if item.line >= selected.line && t.projection.lines[item.line].start <= selected.end && t.projection.lines[item.line].list.depth >= 7 {
					t.Notice = "This preview supports up to eight list levels."
					return true
				}
			}
		}
	}
	lines := map[int]string{}
	for _, item := range chosen {
		for i := item.line; i <= t.projection.sourceLine(item.end); i++ {
			raw := t.Source.Buffer.Line(i)
			if strings.TrimSpace(raw) != "" {
				lines[i] = shiftListIndent(raw, delta)
			}
		}
	}
	if !outdent && first.line > 0 && strings.TrimSpace(t.Source.Buffer.Line(first.line-1)) != "" {
		// Empty items and non-1 ordered starts cannot interrupt a paragraph.
		// A blank line makes the requested nesting unambiguous Markdown.
		prefix, valid := readListPrefix(lines[first.line])
		if valid && (strings.TrimSpace(lines[first.line][prefix.content:]) == "" || (prefix.delimiter != 0 && strings.TrimRight(prefix.marker, ".)") != "1")) {
			lines[first.line] = "\n" + lines[first.line]
		}
	}
	t.applyListLines(lines)
	return true
}

func (p *typesetProjection) canList(lineIndex int) bool {
	if _, ok := p.itemAt(lineIndex); ok {
		return true
	}
	line := p.lines[lineIndex]
	if line.start < frontMatterEnd(p.source) {
		return false
	}
	for i := 1; i < p.parsed.Len(); i++ {
		b := p.parsed.Block(i)
		if b.Start > line.end || b.End <= line.start {
			continue
		}
		if b.Parent != 0 || (b.Kind != parser.Paragraph && !(b.Kind == parser.Heading && line.heading > 0)) {
			return false
		}
	}
	// Unparsed reference definitions and link definitions remain source text.
	raw := strings.TrimSpace(p.source[line.start:line.end])
	return !(strings.HasPrefix(raw, "[") && strings.Contains(raw, "]:"))
}

// SetList toggles a homogeneous selection, or converts paragraphs/items to kind.
func (t *Typesetter) SetList(kind ListKind) {
	if kind > TaskList {
		return
	}
	t.ToSource()
	sel := t.Source.Selection()
	first, last := t.projection.sourceLine(sel.Start()), t.projection.sourceLine(sel.End())
	if !sel.Empty() && last > first && sel.End() <= t.projection.lines[last].content {
		last--
	}
	var items []typesetListItem
	allSame := kind != NoList
	seen := map[int]bool{}
	for i := first; i <= last; i++ {
		if first != last && strings.TrimSpace(t.Source.Buffer.Line(i)) == "" {
			continue
		}
		if !t.projection.canList(i) {
			t.Notice = "Choose ordinary paragraphs or list items. Code and other Markdown stay unchanged."
			return
		}
		item, ok := t.projection.itemAt(i)
		if !ok {
			line := t.projection.lines[i]
			raw := t.Source.Buffer.Line(i)
			indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
			item = typesetListItem{line: i, end: line.end, parent: -1, prefix: listPrefix{task: -1, indent: indent, content: max(indent, line.content-line.start)}}
			allSame = false
		} else {
			allSame = allSame && t.projection.lines[item.line].list.kind == kind
		}
		if !seen[item.line] {
			seen[item.line] = true
			items = append(items, item)
		}
	}
	if allSame {
		kind = NoList
	}
	t.setListItems(items, kind)
}

func (t *Typesetter) setListItems(items []typesetListItem, kind ListKind) {
	lines := map[int]string{}
	converted := map[int]bool{}
	for _, item := range items {
		converted[item.line] = true
	}
	// Changes to a parent's marker width also move its continuation lines and
	// children. Deeper selected items inherit those indentation adjustments.
	get := func(i int) string {
		if line, ok := lines[i]; ok {
			return line
		}
		return t.Source.Buffer.Line(i)
	}
	ordinals := map[int]int{}
	for _, item := range items {
		old := t.Source.Buffer.Line(item.line)
		raw := get(item.line)
		prefix := raw[:max(0, len(raw)-len(old)+item.prefix.indent)]
		base := 0
		if kind != NoList {
			marker := "- "
			if kind == NumberedList {
				ordinals[item.parent]++
				marker = strconv.Itoa(ordinals[item.parent]) + ". "
			}
			prefix += marker
			base = len(marker)
			if kind == TaskList {
				mark := " "
				if item.prefix.task >= 0 && old[item.prefix.task] != ' ' {
					mark = "x"
				}
				prefix += "[" + mark + "] "
			}
		}
		lines[item.line] = prefix + old[item.prefix.content:]
		delta := base - (indentColumns(old[:item.prefix.markerEnd]) - indentColumns(old[:item.prefix.indent]))
		if item.prefix.marker == "" {
			delta = 0 // a plain paragraph has no item continuation to move
		}
		for i := item.line + 1; i <= t.projection.sourceLine(item.end); i++ {
			if raw := get(i); strings.TrimSpace(raw) != "" {
				lines[i] = shiftListIndent(raw, delta)
			}
		}
		if kind == NoList {
			lastList := -1
			for _, child := range t.projection.items {
				if child.parent == item.line && child.list != lastList {
					if strings.TrimSpace(t.Source.Buffer.Line(child.line-1)) != "" {
						lines[child.line] = "\n" + get(child.line)
					}
					lastList = child.list
				}
			}
		}
	}
	if kind == NoList {
		for _, item := range items {
			if item.prefix.marker == "" || item.parent >= 0 || item.line == 0 || strings.TrimSpace(get(item.line-1)) == "" {
				continue
			}
			previous, ok := t.projection.itemAt(item.line - 1)
			if ok && converted[previous.line] {
				continue
			}
			// Without a blank separator the next keystroke would be a lazy
			// continuation of the preceding list, rather than a paragraph.
			if raw := get(item.line); !strings.HasPrefix(raw, "\n") {
				lines[item.line] = "\n" + raw
			}
		}
	}
	t.applyListLines(lines)
}

// ToggleTask changes exactly the source marker; clicking keeps the writing
// selection and focus. The keyboard form addresses the item at the caret.
func (t *Typesetter) ToggleTask(lineIndex int) bool {
	t.ToSource()
	if lineIndex < 0 {
		lineIndex = t.projection.sourceLine(t.Source.Cursor())
	}
	item, ok := t.projection.itemAt(lineIndex)
	if !ok || item.prefix.task < 0 {
		return false
	}
	raw := t.Source.Buffer.Line(item.line)
	at := item.prefix.task
	mark := "x"
	if raw[at] != ' ' {
		mark = " "
	}
	t.applyListLines(map[int]string{item.line: raw[:at] + mark + raw[at+1:]})
	return true
}
