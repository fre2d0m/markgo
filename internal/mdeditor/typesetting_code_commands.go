package mdeditor

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui/editor"
)

func (t *Typesetter) currentCode() (typesetCode, bool) {
	t.ToSource()
	first, last := t.selectedLines()
	code, ok := t.projection.codeAt(first)
	other, same := t.projection.codeAt(last)
	return code, ok && same && code.open == other.open
}

func (t *Typesetter) CodeBlock() {
	if code, ok := t.currentCode(); ok {
		t.unwrapCode(code)
		return
	}
	first, last := t.selectedLines()
	for i := first; i <= last; i++ {
		if !t.projection.canList(i) || t.projection.lines[i].list.kind != NoList {
			t.Notice = "Choose ordinary paragraphs to make a code block."
			return
		}
	}
	p := &t.projection
	a, z := p.lines[first].start, p.lines[last].end
	body := p.source[a:z]
	length := 3
	for line := range strings.SplitSeq(body, "\n") {
		length = max(length, fenceRun(line, '`')+1)
	}
	marker := strings.Repeat("`", length)
	prefix, suffix := marker+"\n", "\n"+marker
	if first > 0 && strings.TrimSpace(t.Source.Buffer.Line(first-1)) != "" {
		prefix = "\n" + prefix
	}
	if last+1 < len(p.lines) && strings.TrimSpace(t.Source.Buffer.Line(last+1)) != "" {
		suffix += "\n"
	}
	sel := t.Source.Selection()
	t.Source.Select(a, z)
	t.Source.Insert(prefix + body + suffix)
	t.Source.Select(sel.Anchor+len(prefix), sel.Head+len(prefix))
	t.FromSource()
	t.Notice = ""
}

func (t *Typesetter) unwrapCode(code typesetCode) {
	p := &t.projection
	sel := t.Source.Selection()
	body := p.source[code.bodyStart:code.bodyEnd]
	prefix, suffix := "", ""
	if code.open > 0 && strings.TrimSpace(t.Source.Buffer.Line(code.open-1)) != "" {
		prefix = "\n"
	}
	last := code.close
	if last < 0 {
		last = code.last
	}
	if last+1 < len(p.lines) && strings.TrimSpace(t.Source.Buffer.Line(last+1)) != "" {
		suffix = "\n"
	}
	move := func(pos int) int { return code.start + len(prefix) + min(max(0, pos-code.bodyStart), len(body)) }
	t.Source.Select(code.start, code.end)
	if text := prefix + body + suffix; text != "" {
		t.Source.Insert(text)
	} else {
		t.Source.Run(editor.Delete)
	}
	t.Source.Select(move(sel.Anchor), move(sel.Head))
	t.FromSource()
	t.Notice = ""
}

func (t *Typesetter) codeEnter() bool {
	code, ok := t.currentCode()
	if !ok {
		return false
	}
	i := t.projection.sourceLine(t.Source.Cursor())
	if t.projection.lines[i].block.virtual {
		at := t.projection.lines[code.open].end
		text := "\n"
		if code.close < 0 {
			text += "\n" + code.marker
		}
		t.Source.Select(at, at)
		t.Source.Insert(text)
		t.Source.Select(at+1, at+1)
		t.FromSource()
		return true
	}
	raw := t.Source.Buffer.Line(i)
	indent := raw[:len(raw)-len(strings.TrimLeft(raw, " \t"))]
	line := t.projection.lines[i]
	indent = indent[:min(len(indent), max(0, t.Source.Selection().Start()-line.start))]
	t.Insert("\n" + indent)
	return true
}

func (t *Typesetter) indentCode(remove bool) bool {
	code, ok := t.currentCode()
	if !ok {
		return false
	}
	first, last := t.selectedLines()
	step, marker := 4, "    "
	name := ""
	if parts := strings.Fields(code.info); len(parts) > 0 {
		name = parts[0]
	}
	if language := editor.LanguageNamed(name); language != nil {
		if language.TabSize > 0 {
			step = language.TabSize
		}
		marker = strings.Repeat(" ", step)
		if language.HardTabs {
			marker = "\t"
		}
	}
	if !remove && t.Source.Selection().Empty() {
		t.Insert(marker)
		return true
	}
	lines := map[int]string{}
	for i := first; i <= last; i++ {
		raw := t.Source.Buffer.Line(i)
		if remove {
			lines[i] = shiftListIndent(raw, -step)
		} else {
			lines[i] = marker + raw
		}
	}
	t.applyLineChanges(lines)
	return true
}

// ContinueAfterCode is also available from the code header when the caret is
// elsewhere. Its one source insertion, if needed, is undoable.
func (t *Typesetter) ContinueAfterCode(open int) bool {
	t.ToSource()
	if open < 0 {
		open = t.projection.sourceLine(t.Source.Cursor())
	}
	code, ok := t.projection.codeAt(open)
	if !ok {
		return false
	}
	if code.close >= 0 && code.close+1 < len(t.projection.lines) && strings.TrimSpace(t.Source.Buffer.Line(code.close+1)) == "" {
		at := t.projection.lines[code.close+1].start
		t.Source.Select(at, at)
		t.FromSource()
		return true
	}
	text := "\n\n"
	if code.close < 0 {
		text = "\n" + code.marker + "\n\n"
		if code.bodyStart == code.bodyEnd {
			text = "\n" + text
		}
	}
	t.Source.Select(code.end, code.end)
	t.Source.Insert(text)
	t.FromSource()
	t.Notice = ""
	return true
}

func (t *Typesetter) SetCodeLanguage(open int, language string) bool {
	language = strings.TrimSpace(language)
	if utf8.RuneCountInString(language) > 32 || strings.ContainsAny(language, "`~") || strings.ContainsFunc(language, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		t.Notice = "Use a short language name, such as go, python or javascript."
		return false
	}
	t.ToSource()
	code, ok := t.projection.codeAt(open)
	if !ok {
		return false
	}
	raw := t.Source.Buffer.Line(code.open)
	start := strings.Index(raw, code.marker) + len(code.marker)
	at := start
	for at < len(raw) && (raw[at] == ' ' || raw[at] == '\t') {
		at++
	}
	end := at
	for end < len(raw) && raw[end] != ' ' && raw[end] != '\t' {
		end++
	}
	if language == "" && strings.TrimSpace(raw[end:]) != "" {
		language = "text" // retain trailing info without turning it into a language
	}
	t.applyLineChanges(map[int]string{code.open: raw[:at] + language + raw[end:]})
	return true
}

func (t *Typesetter) CodeText(open int) (string, bool) {
	code, ok := t.projection.codeAt(open)
	if !ok {
		return "", false
	}
	return t.projection.source[code.bodyStart:code.bodyEnd], true
}
