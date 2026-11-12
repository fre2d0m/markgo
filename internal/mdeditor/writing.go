package mdeditor

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// writingSyntax reuses the editor's incremental tokenizer to distinguish
// ordinary Markdown from fenced code and comments before continuing a list.
// No whole-document parse runs on a key press.
type writingSyntax struct {
	editor.Highlighter
	text  editor.Snapshot
	plain bool
}

func (s *writingSyntax) Edit(old, next editor.Snapshot, changes []editor.Change) {
	s.text = next
	s.Highlighter.Edit(old, next, changes)
}

func (f *File) SetLanguage(language *editor.Language) {
	f.syntax = nil
	if language == nil || language.Name != "Markdown" || language.Tokenizer == nil {
		f.State.Buffer.SetLanguage(language)
		return
	}
	copy := *language
	copy.Highlighter = func(text editor.Snapshot, _ func()) editor.Highlighter {
		syntax := &writingSyntax{text: text}
		tokenizer := editor.TokenizerFunc(func(line string, state uint64, tokens []editor.Token) ([]editor.Token, uint64) {
			// MyGo's Markdown tokenizer uses state zero for ordinary prose;
			// fenced code and multiline comments carry a nonzero state.
			syntax.plain = state == 0
			return language.Tokenizer.Tokenize(line, state, tokens)
		})
		syntax.Highlighter = editor.NewTokenHighlighter(tokenizer, text)
		f.syntax = syntax
		return syntax
	}
	f.State.Buffer.SetLanguage(&copy)
}

func (f *File) writingInput(event ui.InputEvent) bool {
	st := f.State
	if event.Kind != ui.InputKeyDown || event.Key != ui.KeyEnter || event.Mods != 0 || st.ReadOnly {
		return false
	}
	if len(st.Selections()) != 1 || !st.Selection().Empty() {
		return false
	}
	position := st.CursorPos()
	syntax := f.syntax
	if syntax == nil || syntax.text.Version() != st.Buffer.Version() || !syntax.Highlight(position.Line, time.Millisecond) {
		return false
	}
	line := st.Buffer.Line(position.Line)
	_, ready := syntax.Tokens(position.Line, line, nil)
	if !ready || !syntax.plain {
		return false
	}
	edit, ok := continuation(line, position.Col)
	if !ok {
		return false
	}
	offset := st.Buffer.Offset(editor.Pos{Line: position.Line})
	st.Select(offset+edit.Start, offset+edit.End)
	st.Insert(edit.Text)
	return true
}

var listStart = regexp.MustCompile(`^([-+*]|([0-9]{1,9})([.)]))([ \t]+)(?:\[([ xX])\]([ \t]+))?`)

// continuation edits only the current line. Indented code (four spaces or a
// tab) and selected text retain the editor's normal Return behavior.
func continuation(line string, column int) (editor.Edit, bool) {
	indent := len(line) - len(strings.TrimLeft(line, " "))
	if indent > 3 || column < indent || column > len(line) || strings.HasPrefix(line[indent:], "\t") {
		return editor.Edit{}, false
	}
	baseEnd, previousQuote := indent, indent
	for baseEnd < len(line) && line[baseEnd] == '>' {
		previousQuote = baseEnd
		baseEnd++
		if baseEnd < len(line) && (line[baseEnd] == ' ' || line[baseEnd] == '\t') {
			baseEnd++
		}
	}
	prefixEnd := baseEnd
	match := listStart.FindStringSubmatch(line[baseEnd:])
	if match == nil && baseEnd == indent {
		return editor.Edit{}, false
	}
	if match != nil {
		prefixEnd += len(match[0])
	}
	if column < prefixEnd {
		return editor.Edit{}, false
	}
	if strings.TrimSpace(line[prefixEnd:]) == "" && column == len(line) {
		start := baseEnd
		if match == nil {
			start = previousQuote
		}
		return editor.Edit{Start: start, End: len(line)}, true
	}
	prefix := line[:baseEnd]
	if match != nil {
		marker := match[1]
		if match[2] != "" {
			number, _ := strconv.Atoi(match[2])
			if number == 999999999 {
				return editor.Edit{}, false
			}
			marker = strconv.Itoa(number+1) + match[3]
		}
		prefix += marker + match[4]
		if match[5] != "" {
			prefix += "[ ]" + match[6]
		}
	}
	return editor.Edit{Start: column, End: column, Text: "\n" + prefix}, true
}
