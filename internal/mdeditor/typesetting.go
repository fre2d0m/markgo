package mdeditor

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown/parser"
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
}

type typesetLine struct {
	start, content, end int // UTF-8 source offsets
	viewStart, heading  int // projected rune offset; ATX level
}

type typesetProjection struct {
	source, text string
	// Each visible rune boundary maps to a source byte boundary. At a
	// heading's start it points after the hidden prefix. A selection crossing
	// the preceding newline therefore also consumes that prefix when joining.
	boundaries []int
	lines      []typesetLine
}

func projectTypesetting(source string) typesetProjection {
	p := typesetProjection{source: source}
	headings := make(map[int]int)
	doc := parser.Parse(source)
	frontEnd := frontMatterEnd(source)
	for i := 1; i < doc.Len(); i++ {
		b := doc.Block(i)
		if b.Kind == parser.Heading && b.Parent == 0 && b.Start >= frontEnd {
			headings[b.Start] = b.Level
		}
	}
	var shown strings.Builder
	start := 0
	for _, line := range strings.Split(source, "\n") {
		content, level := start, headings[start]
		if level > 0 {
			if prefix := headingPrefix.FindStringIndex(line); prefix != nil {
				content += prefix[1]
			} else {
				level = 0 // Setext headings remain literal in this prototype.
			}
		}
		p.lines = append(p.lines, typesetLine{start, content, start + len(line), len(p.boundaries), level})
		for off, r := range source[content : start+len(line)] {
			p.boundaries = append(p.boundaries, content+off)
			shown.WriteRune(r)
		}
		if start+len(line) < len(source) {
			p.boundaries = append(p.boundaries, start+len(line))
			shown.WriteByte('\n')
		}
		start += len(line) + 1
	}
	p.boundaries = append(p.boundaries, len(source))
	p.text = shown.String()
	return p
}

func (p typesetProjection) sourceOffset(runeOffset int) int {
	return p.boundaries[max(0, min(runeOffset, len(p.boundaries)-1))]
}

func (p typesetProjection) viewOffset(byteOffset int) int {
	i := sort.SearchInts(p.boundaries, byteOffset)
	return min(i, len(p.boundaries)-1)
}

func NewTypesetter(source *editor.State) *Typesetter {
	t := &Typesetter{Source: source, Document: richtext.NewDocument(richtext.Text{})}
	t.input.owner = t
	t.FromSource()
	return t
}

// FromSource rebuilds presentation after source edits, undo, or a mode change.
// It does not edit the source or clear its undo history.
func (t *Typesetter) FromSource() {
	t.projection = projectTypesetting(t.Source.Buffer.String())
	sel := t.Source.Selection()
	t.Document.SetText(richtext.NewText(t.projection.text))
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
		t.FromSource()
		switch command {
		case "undo":
			t.Undo(false)
		case "redo":
			t.Undo(true)
		case "paste", "pasteAndMatchStyle":
			t.Insert(text)
		default:
			t.Notice = "This formatting is available in Markdown source."
		}
		return
	}
	if t.composing || t.Document.Version() == t.richVersion {
		return
	}
	old, next := []rune(t.projection.text), []rune(t.Document.Text().String())
	a, z := t.Document.Selection()
	if string(old) == string(next) {
		// Unmapped rich formatting must never look saved without source syntax.
		t.Source.Select(t.projection.sourceOffset(a), t.projection.sourceOffset(z))
		t.FromSource()
		return
	}
	start := 0
	for start < min(len(old), len(next), min(t.beforeAnchor, t.beforeCaret)) && old[start] == next[start] {
		start++
	}
	oldEnd, newEnd := len(old), len(next)
	for oldEnd > start && newEnd > start && old[oldEnd-1] == next[newEnd-1] {
		oldEnd--
		newEnd--
	}
	sourceStart, sourceEnd := t.projection.sourceOffset(start), t.projection.sourceOffset(oldEnd)
	if t.beforeAnchor != t.beforeCaret && min(t.beforeAnchor, t.beforeCaret) == 0 && max(t.beforeAnchor, t.beforeCaret) == len(old) {
		sourceStart, sourceEnd, start, oldEnd, newEnd = 0, len(t.projection.source), 0, len(old), len(next)
	}
	inserted := string(next[start:newEnd])
	mapSelection := func(pos int) int {
		if pos < start {
			return t.projection.sourceOffset(pos)
		}
		if pos <= newEnd {
			return sourceStart + len(string(next[start:pos]))
		}
		return t.projection.sourceOffset(pos-newEnd+oldEnd) + len(inserted) - (sourceEnd - sourceStart)
	}
	anchor, caret := mapSelection(a), mapSelection(z)
	t.Source.Select(sourceStart, sourceEnd)
	if inserted == "" {
		t.Source.Run(editor.Delete)
	} else {
		t.Source.Insert(inserted)
	}
	t.Source.Select(anchor, caret)
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
	t.Source.Insert(text)
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
	t.Source.Run(editor.Delete)
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
	t.ToSource()
	if !SetHeading(t.Source, level) {
		t.Notice = "Choose one paragraph to change its heading."
	}
	t.FromSource()
}
