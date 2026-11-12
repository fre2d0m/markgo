package mdeditor

import (
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

var (
	markdownEmphasis = regexp.MustCompile("(\\*\\*|\\*|`)([^*`]+)(\\*\\*|\\*|`)")
	markdownLink     = regexp.MustCompile(`(\[)[^\]]+(\]\([^)]*\))`)
	markdownHeading  = regexp.MustCompile(`^#{1,6} `)
	markdownBullet   = regexp.MustCompile(`^\s*(-) `)
	markdownTask     = regexp.MustCompile(`^\s*(- \[[ x]\])`)
)

func LivePreview(st *editor.State) []editor.Replacement {
	buffer := st.Buffer
	caretLines := map[int]bool{}
	for _, selection := range st.Selections() {
		for line := buffer.Pos(selection.Start()).Line; line <= buffer.Pos(selection.End()).Line; line++ {
			caretLines[line] = true
		}
	}
	var replacements []editor.Replacement
	fenced := false
	for lineNumber := range buffer.LineCount() {
		line := buffer.Line(lineNumber)
		if strings.HasPrefix(line, "```") {
			fenced = !fenced
			continue
		}
		if fenced || caretLines[lineNumber] {
			continue
		}
		start := buffer.Offset(editor.Pos{Line: lineNumber})
		if match := markdownHeading.FindStringIndex(line); match != nil {
			replacements = append(replacements, editor.Replacement{Start: start, End: start + match[1]})
		}
		if match := markdownTask.FindStringSubmatchIndex(line); match != nil {
			replacements = append(replacements, taskReplacement(st, start+match[2], start+match[3]))
		} else if match := markdownBullet.FindStringSubmatchIndex(line); match != nil {
			replacements = append(replacements, editor.Replacement{Start: start + match[2], End: start + match[3], Text: "•"})
		}
		for _, match := range markdownEmphasis.FindAllStringSubmatchIndex(line, -1) {
			if line[match[2]:match[3]] == line[match[6]:match[7]] {
				replacements = append(replacements,
					editor.Replacement{Start: start + match[2], End: start + match[3]},
					editor.Replacement{Start: start + match[6], End: start + match[7]},
				)
			}
		}
		for _, match := range markdownLink.FindAllStringSubmatchIndex(line, -1) {
			replacements = append(replacements,
				editor.Replacement{Start: start + match[2], End: start + match[3]},
				editor.Replacement{Start: start + match[4], End: start + match[5]},
			)
		}
	}
	return replacements
}

func taskReplacement(st *editor.State, start, end int) editor.Replacement {
	done := strings.Contains(st.Buffer.Slice(start, end), "x")
	label, mark := "Not done", "☐"
	if done {
		label, mark = "Done", "☑"
	}
	return editor.Replacement{
		Start:       start,
		End:         end,
		Label:       label,
		Interactive: true,
		Content: func(c *ui.Context, element ui.Element, replacement editor.Replacement) {
			element.Children(func() {
				button := ui.ButtonBase(c).Fill().Center().KeepFocus().Cursor(ui.CursorPointer).Label(label)
				button.Children(func() { ui.Text(c, mark) })
				if button.Clicked() {
					text := "x"
					if done {
						text = " "
					}
					st.Buffer.Edit(editor.Edit{Start: replacement.End - 2, End: replacement.End - 1, Text: text})
				}
			})
		},
	}
}
