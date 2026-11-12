package mdeditor

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui/editor"
)

var existingHeading = regexp.MustCompile(`^[ \t]{0,3}#{1,6}(?:[ \t]+|$)`)

// ToggleInline wraps one selection, or removes matching delimiters around it.
// Each command is a single buffer edit and therefore a single undo step.
func ToggleInline(st *editor.State, delimiter string) bool {
	selections := st.Selections()
	if len(selections) != 1 || delimiter == "" {
		return false
	}
	selection := selections[0]
	start, end := selection.Start(), selection.End()
	text := st.Buffer.String()
	if !validRange(text, start, end) {
		return false
	}
	chosen := text[start:end]
	n := len(delimiter)
	if len(chosen) >= 2*n && strings.HasPrefix(chosen, delimiter) && strings.HasSuffix(chosen, delimiter) {
		inner := chosen[n : len(chosen)-n]
		st.Buffer.Edit(editor.Edit{Start: start, End: end, Text: inner})
		st.Select(start, start+len(inner))
		return true
	}
	if start >= n && end+n <= len(text) && text[start-n:start] == delimiter && text[end:end+n] == delimiter {
		st.Buffer.Edit(editor.Edit{Start: start - n, End: end + n, Text: chosen})
		st.Select(start-n, end-n)
		return true
	}
	st.Buffer.Edit(editor.Edit{Start: start, End: end, Text: delimiter + chosen + delimiter})
	st.Select(start+n, end+n)
	return true
}

// SetHeading changes the current line's ATX heading marker, keeping its text.
func SetHeading(st *editor.State, level int) bool {
	if level < 0 || level > 6 || len(st.Selections()) != 1 {
		return false
	}
	selection := st.Selection()
	startLine := st.Buffer.Pos(selection.Start()).Line
	if st.Buffer.Pos(selection.End()).Line != startLine {
		return false
	}
	line := st.Buffer.Line(startLine)
	offset := st.Buffer.Offset(editor.Pos{Line: startLine})
	prefixEnd := 0
	if match := existingHeading.FindStringIndex(line); match != nil {
		prefixEnd = match[1]
	}
	prefix := ""
	if level > 0 {
		prefix = strings.Repeat("#", level) + " "
	}
	st.Buffer.Edit(editor.Edit{Start: offset, End: offset + prefixEnd, Text: prefix})
	delta := len(prefix) - prefixEnd
	st.Select(max(offset+len(prefix), selection.Start()+delta), max(offset+len(prefix), selection.End()+delta))
	return true
}

// ToggleLinePrefix adds or removes a prefix on the current line.
func ToggleLinePrefix(st *editor.State, prefix string) bool {
	if prefix == "" || len(st.Selections()) != 1 {
		return false
	}
	selection := st.Selection()
	lineNumber := st.Buffer.Pos(selection.Start()).Line
	if st.Buffer.Pos(selection.End()).Line != lineNumber {
		return false
	}
	line := st.Buffer.Line(lineNumber)
	offset := st.Buffer.Offset(editor.Pos{Line: lineNumber})
	remove := strings.HasPrefix(line, prefix)
	end, replacement, delta := offset, prefix, len(prefix)
	if remove {
		end, replacement, delta = offset+len(prefix), "", -len(prefix)
	}
	st.Buffer.Edit(editor.Edit{Start: offset, End: end, Text: replacement})
	st.Select(max(offset, selection.Start()+delta), max(offset, selection.End()+delta))
	return true
}

// InsertLink creates an editable Markdown link and selects its placeholder.
func InsertLink(st *editor.State) bool {
	selections := st.Selections()
	if len(selections) != 1 {
		return false
	}
	start, end := selections[0].Start(), selections[0].End()
	text := st.Buffer.String()
	if !validRange(text, start, end) {
		return false
	}
	label := text[start:end]
	var selectStart, selectEnd int
	if label == "" {
		label = "link text"
		selectStart = start + 1
		selectEnd = selectStart + len(label)
	} else {
		selectStart = start + len(label) + 3
		selectEnd = selectStart + len("url")
	}
	st.Buffer.Edit(editor.Edit{Start: start, End: end, Text: "[" + label + "](url)"})
	st.Select(selectStart, selectEnd)
	return true
}

func validRange(text string, start, end int) bool {
	return start >= 0 && end >= start && end <= len(text) &&
		(start == len(text) || utf8.RuneStart(text[start])) && (end == len(text) || utf8.RuneStart(text[end]))
}
