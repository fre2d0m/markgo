package mdeditor

import (
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown/parser"
)

var (
	headingPrefix = regexp.MustCompile(`^[ \t]{0,3}#{1,6}(?:[ \t]+|$)`)
	itemPrefix    = regexp.MustCompile(`^[ \t]*(?:[-+*]|[0-9]{1,9}[.)])[ \t]+`)
	taskPrefix    = regexp.MustCompile(`^[ \t]*(?:[-+*]|[0-9]{1,9}[.)])[ \t]+\[[ xX]\]`)
)

type markerKind uint8

const (
	hiddenMarker markerKind = iota
	bulletMarker
	taskMarker
)

type previewMarker struct {
	start, end             int
	revealStart, revealEnd int
	kind                   markerKind
}

// Analysis is a source-preserving view of a Markdown snapshot. Its offsets are
// byte offsets in that exact source and must not be applied to another version.
type Analysis struct {
	Headings []Heading
	Stats    Statistics
	markers  []previewMarker
	source   string
}

// Source returns the immutable source already flattened by the analysis worker.
func (a Analysis) Source() string { return a.source }

// Analyze parses a snapshot without changing it. It is safe to run on a worker.
func Analyze(source string) Analysis {
	doc := parser.Parse(source)
	frontEnd := frontMatterEnd(source)
	result := Analysis{source: source}
	var visible strings.Builder
	add := func(start, end, revealStart, revealEnd int, kind markerKind) {
		if start >= frontEnd && start >= 0 && end > start && end <= len(source) {
			result.markers = append(result.markers, previewMarker{start, end, revealStart, revealEnd, kind})
		}
	}
	for i := 1; i < doc.Len(); i++ {
		block := doc.Block(i)
		if block.Start < frontEnd || block.Start < 0 || block.End > len(source) || block.End < block.Start {
			continue
		}
		switch block.Kind {
		case parser.Heading:
			title := strings.TrimSpace(parser.PlainText(doc.Inlines(i)))
			if title == "" {
				title = "Untitled heading"
			}
			result.Headings = append(result.Headings, Heading{Title: title, Level: block.Level, Offset: block.Start})
			line := firstLine(source[block.Start:block.End])
			if match := headingPrefix.FindStringIndex(line); match != nil {
				add(block.Start+match[0], block.Start+match[1], block.Start, block.End, hiddenMarker)
			}
		case parser.Item:
			line := firstLine(source[block.Start:block.End])
			if block.Task != parser.NoTask {
				if match := taskPrefix.FindStringIndex(line); match != nil {
					start, end := block.Start+match[0], block.Start+match[1]
					add(start, end, start, end, taskMarker)
				}
			} else if match := itemPrefix.FindStringIndex(line); match != nil {
				prefix := line[match[0]:match[1]]
				for offset, ch := range prefix {
					if ch == '-' || ch == '+' || ch == '*' {
						start := block.Start + match[0] + offset
						add(start, start+1, start, start+1, bulletMarker)
						break
					}
				}
			}
		}
		switch block.Kind {
		case parser.Paragraph, parser.Heading, parser.TableCell:
			visible.WriteString(parser.PlainText(doc.Inlines(i)))
			visible.WriteByte('\n')
		case parser.CodeBlock, parser.MathBlock, parser.HTMLBlock:
			visible.WriteString(doc.Lines(i))
			visible.WriteByte('\n')
		}
		if block.Kind != parser.Paragraph && block.Kind != parser.Heading && block.Kind != parser.TableCell {
			continue
		}
		inlines := doc.Inlines(i)
		for j, in := range inlines {
			if in.Start < frontEnd || in.Start < 0 || in.End > len(source) || in.End <= in.Start {
				continue
			}
			raw := source[in.Start:in.End]
			switch in.Kind {
			case parser.Strong, parser.Emphasis, parser.Strikethrough:
				n := 1
				if in.Kind == parser.Strong || (in.Kind == parser.Strikethrough && strings.HasPrefix(raw, "~~")) {
					n = 2
				}
				if len(raw) > 2*n && raw[:n] == raw[len(raw)-n:] {
					add(in.Start, in.Start+n, in.Start, in.End, hiddenMarker)
					add(in.End-n, in.End, in.Start, in.End, hiddenMarker)
				}
			case parser.Code:
				n := len(raw) - len(strings.TrimLeft(raw, "`"))
				if n > 0 && len(raw) > 2*n && strings.HasSuffix(raw, strings.Repeat("`", n)) {
					add(in.Start, in.Start+n, in.Start, in.End, hiddenMarker)
					add(in.End-n, in.End, in.Start, in.End, hiddenMarker)
				}
			case parser.Link:
				if in.Auto || j+1 >= in.Next || in.Next > len(inlines) || !strings.HasPrefix(raw, "[") {
					continue
				}
				start, end := in.End, in.Start
				for _, child := range inlines[j+1 : in.Next] {
					if child.Start >= in.Start && child.End <= in.End {
						start, end = min(start, child.Start), max(end, child.End)
					}
				}
				if start < end {
					add(in.Start, start, in.Start, in.End, hiddenMarker)
					add(end, in.End, in.Start, in.End, hiddenMarker)
				}
			}
		}
	}
	result.Stats = CountText(strings.TrimRight(visible.String(), "\n"))
	return result
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

// frontMatterEnd keeps YAML metadata out of the writing preview and outline.
// An unfinished front matter block stays literal until the closing marker.
func frontMatterEnd(source string) int {
	first, rest, ok := strings.Cut(source, "\n")
	if !ok || strings.TrimPrefix(first, "\ufeff") != "---" {
		return 0
	}
	offset := len(first) + 1
	for rest != "" {
		line, next, hasNewline := strings.Cut(rest, "\n")
		offset += len(line)
		if hasNewline {
			offset++
		}
		if line == "---" || line == "..." {
			return offset
		}
		if !hasNewline {
			break
		}
		rest = next
	}
	return len(source)
}

// Replacements hides only the syntax whose element is not currently being
// edited. The user's selection and source bytes remain in the editor buffer.
func (a Analysis) Replacements(st *editor.State) []editor.Replacement {
	selections := st.Selections()
	var replacements []editor.Replacement
	for _, marker := range a.markers {
		reveal := false
		for _, selection := range selections {
			if selection.Empty() {
				reveal = selection.Start() >= marker.revealStart && selection.Start() <= marker.revealEnd
			} else {
				reveal = selection.Start() < marker.revealEnd && selection.End() > marker.revealStart
			}
			if reveal {
				break
			}
		}
		if reveal {
			continue
		}
		switch marker.kind {
		case bulletMarker:
			replacements = append(replacements, editor.Replacement{Start: marker.start, End: marker.end, Text: "•"})
		case taskMarker:
			replacements = append(replacements, taskReplacement(st, marker.start, marker.end))
		default:
			replacements = append(replacements, editor.Replacement{Start: marker.start, End: marker.end})
		}
	}
	return replacements
}

// LivePreview is convenient for callers that do not maintain a cached analysis.
func LivePreview(st *editor.State) []editor.Replacement {
	return Analyze(st.Buffer.String()).Replacements(st)
}

func taskReplacement(st *editor.State, start, end int) editor.Replacement {
	done := strings.Contains(strings.ToLower(st.Buffer.Slice(start, end)), "x")
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
