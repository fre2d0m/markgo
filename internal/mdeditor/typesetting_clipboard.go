package mdeditor

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// Whole paragraphs retain their original Markdown, including list markers,
// indentation and task state. A selection within a sentence copies plain text.
// Keyboard, context-menu and native Edit-menu commands all share this boundary.
func (t *Typesetter) copySelection(services ui.Services, cut bool) {
	t.ToSource()
	anchor, caret := t.Document.Selection()
	a, z := min(anchor, caret), max(anchor, caret)
	if a == z {
		return
	}
	p := &t.projection
	start, end := p.sourceOffset(a), p.sourceOffset(z)
	first, last := p.lines[p.sourceLine(start)], p.lines[p.sourceLine(end)]
	whole := a == first.viewStart && (z == last.viewStart+last.runes || z == last.viewStart)
	text := t.Document.Text().Slice(a, z).String()
	if whole {
		start = first.start
		if z == last.viewStart {
			end = last.start
		}
		text = p.source[start:end]
	}
	services.WriteClipboard(text)
	if cut {
		if !whole {
			t.DeleteSelection()
			return
		}
		if anchor > caret {
			t.Source.Select(end, start)
		} else {
			t.Source.Select(start, end)
		}
		t.Source.Run(editor.Delete)
		t.FromSource()
	}
}
