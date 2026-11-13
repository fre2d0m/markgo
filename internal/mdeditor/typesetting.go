package mdeditor

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/richtext"
)

// Typesetter is an experimental, source-preserving editing projection. Source
// owns every edit and its history. Document is disposable presentation state;
// neither richtext.ParseMarkdown nor its Markdown serializer is used here.
// Keep it on the UI thread, in the isolated sample workspace only.
type Typesetter struct {
	Source   *editor.State
	Document *richtext.Document
	Notice   string

	projection                          typesetProjection
	sourceVersion, richVersion          uint64
	beforeAnchor, beforeCaret           int
	composing                           bool
	compositionAnchor, compositionCaret int
	pendingCommand, pendingText         string
	input                               typesettingInput
	appliedStyle                        int
	services                            ui.Services
	blockUI                             typesetBlockUI
}

func NewTypesetter(source *editor.State) *Typesetter {
	t := &Typesetter{Source: source, Document: richtext.NewDocument(richtext.Text{})}
	t.input.owner = t
	t.FromSource()
	return t
}

// FromSource updates the projection after source edits, undo, or a mode change.
// Only changed display text is replaced, so richtext retains the other
// paragraphs' layouts. Source remains the only authority for undo.
func (t *Typesetter) FromSource() {
	t.projection.update(t.Source.Buffer.String())
	t.input.clearMark()
	sel := t.Source.Selection()
	shown := t.Document.Text().String()
	if shown != t.projection.text {
		start, oldEnd, newEnd := changedText(shown, t.projection.text, len(shown))
		if strings.Contains(shown[start:oldEnd], "\n") || strings.Contains(t.projection.text[start:newEnd], "\n") {
			t.appliedStyle = 0
		}
		a, z := utf8.RuneCountInString(shown[:start]), utf8.RuneCountInString(shown[:oldEnd])
		t.Document.Edit(func(e *richtext.Edit) {
			e.ReplaceString(a, z, t.projection.text[start:newEnd], richtext.Attributes{})
		})
	}
	t.projectParagraphStyles()
	t.Document.SetSelection(t.projection.viewOffset(sel.Anchor), t.projection.viewOffset(sel.Head))
	t.sourceVersion, t.richVersion = t.Source.Buffer.Version(), t.Document.Version()
	t.beforeAnchor, t.beforeCaret = t.Document.Selection()
	t.composing = false
	t.input.marked = false
}

// Sync commits completed native edits as precise source replacements. While an
// IME is composing, even a temporarily removed selection stays in Source.
func (t *Typesetter) Sync() {
	if t.Source.Buffer.Version() != t.sourceVersion {
		t.FromSource()
		return
	}
	if t.pendingCommand != "" {
		command, text := t.pendingCommand, t.pendingText
		t.pendingCommand, t.pendingText = "", ""
		// Native menu commands bypass EditorOptions.OnKey. Their rich-text
		// changes are intentionally discarded, then the captured source command
		// runs once. This also prevents HTML/RTF paste from rewriting Markdown.
		t.Source.Select(t.projection.sourceOffset(t.beforeAnchor), t.projection.sourceOffset(t.beforeCaret))
		t.appliedStyle = 0
		t.FromSource()
		switch command {
		case "undo":
			t.Undo(false)
		case "redo":
			t.Undo(true)
		case "paste", "pasteAndMatchStyle":
			t.Insert(text)
		case "copy", "cut":
			t.copySelection(t.services, command == "cut")
		case "bullets":
			t.SetList(BulletedList)
		case "numbers":
			t.SetList(NumberedList)
		case "tasks":
			t.SetList(TaskList)
		case "quote":
			t.Quote()
		case "codeBlock":
			t.CodeBlock()
		case "indent", "outdent":
			t.indentBlocks(command == "outdent")
		default:
			t.Document.SetText(richtext.NewText(t.projection.text))
			t.appliedStyle = 0
			t.FromSource()
			t.Notice = "This formatting is available in Markdown source."
		}
		return
	}
	if t.composing || t.Document.Version() == t.richVersion {
		return
	}
	old, next := t.projection.text, t.Document.Text().String()
	a, z := t.Document.Selection()
	if old == next {
		// Unmapped rich formatting must never look saved without source syntax.
		t.Source.Select(t.projection.sourceOffset(a), t.projection.sourceOffset(z))
		t.Document.SetText(richtext.NewText(old))
		t.appliedStyle = 0
		t.FromSource()
		return
	}
	first, oldLast, newLast := changedText(old, next, runeByte(old, min(t.beforeAnchor, t.beforeCaret)))
	start := utf8.RuneCountInString(old[:first])
	oldEnd, newEnd := utf8.RuneCountInString(old[:oldLast]), utf8.RuneCountInString(next[:newLast])
	sourceStart, sourceEnd := t.projection.sourceOffset(start), t.projection.sourceOffset(oldEnd)
	if t.beforeAnchor != t.beforeCaret && min(t.beforeAnchor, t.beforeCaret) == 0 && max(t.beforeAnchor, t.beforeCaret) == t.projection.len() {
		sourceStart, sourceEnd, start, oldEnd, newEnd = 0, len(t.projection.source), 0, t.projection.len(), t.Document.Text().Len()
		first, newLast = 0, len(next)
	}
	inserted := next[first:newLast]
	prepared := t.projection.prepareInsertion(sourceStart, sourceEnd, inserted, sourceStart == 0 && sourceEnd == len(t.projection.source) && sourceStart != sourceEnd)
	mapSelection := func(pos int) int {
		if pos < start {
			return t.projection.sourceOffset(pos)
		}
		if pos <= newEnd {
			return sourceStart + prepared.offset(pos-start)
		}
		return t.projection.sourceOffset(pos-newEnd+oldEnd) + len(prepared.text) - (sourceEnd - sourceStart)
	}
	anchor, caret := mapSelection(a), mapSelection(z)
	t.applyProjectedEdit(sourceStart, sourceEnd, prepared.text, anchor, caret)
	t.FromSource()
}

// ToSource transfers the continuous (possibly reversed) selection on a mode switch.
func (t *Typesetter) ToSource() {
	t.input.typing = nil
	t.CancelComposition()
	t.Sync()
	a, z := t.Document.Selection()
	t.Source.Select(t.projection.sourceOffset(a), t.projection.sourceOffset(z))
}

func (t *Typesetter) CancelComposition() {
	if !t.composing {
		return
	}
	a, z := t.compositionAnchor, t.compositionCaret
	t.FromSource()
	t.Document.SetSelection(a, z)
	t.Notice = "Unfinished input was cancelled. Your text is unchanged."
}

func (t *Typesetter) Insert(text string) {
	if text == "" {
		return
	}
	t.ToSource()
	a, z := t.Document.Selection()
	if min(a, z) == 0 && max(a, z) == utf8.RuneCountInString(t.projection.text) && a != z {
		t.Source.Select(0, t.Source.Buffer.Len())
	}
	sel := t.Source.Selection()
	prepared := t.projection.prepareInsertion(sel.Start(), sel.End(), text, sel.Start() == 0 && sel.End() == t.Source.Buffer.Len() && !sel.Empty())
	end := sel.Start() + len(prepared.text)
	t.applyProjectedEdit(sel.Start(), sel.End(), prepared.text, end, end)
	t.FromSource()
}

func (t *Typesetter) DeleteSelection() {
	t.ToSource()
	a, z := t.Document.Selection()
	if a == z {
		return
	}
	if min(a, z) == 0 && max(a, z) == t.Document.Text().Len() {
		t.Source.Select(0, t.Source.Buffer.Len())
	}
	sel := t.Source.Selection()
	t.applyProjectedEdit(sel.Start(), sel.End(), "", sel.Start(), sel.Start())
	t.FromSource()
}

func (t *Typesetter) Undo(redo bool) {
	if t.composing {
		t.CancelComposition()
		return
	}
	t.ToSource()
	cmd := editor.Undo
	if redo {
		cmd = editor.Redo
	}
	t.Source.Run(cmd)
	t.FromSource()
}

func (t *Typesetter) Heading(level int) {
	if level < 0 || level > 6 {
		return
	}
	t.ToSource()
	sel := t.Source.Selection()
	i := t.projection.sourceLine(sel.Start())
	if level == 0 {
		if code, ok := t.projection.codeAt(i); ok {
			t.unwrapCode(code)
			return
		}
		if t.projection.lines[i].block.quote > 0 {
			first, last := t.selectedLines()
			t.changeQuote(first, last, true)
			return
		}
	}
	if !t.projection.canList(i) {
		t.Notice = "Choose an ordinary paragraph to change its heading."
		return
	}
	if item, ok := t.projection.itemAt(i); ok {
		if level == 0 {
			t.SetList(NoList)
			return
		}
		line := t.projection.lines[i]
		if i != item.line || item.end > line.end || t.projection.sourceLine(sel.End()) != i {
			t.Notice = "Choose one item without nested content to make a heading."
			return
		}
		t.applyLineChanges(map[int]string{i: strings.Repeat("#", level) + " " + t.projection.source[line.content:line.end]})
		return
	}
	if !SetHeading(t.Source, level) {
		t.Notice = "Choose one paragraph to change its heading."
	}
	t.FromSource()
}
