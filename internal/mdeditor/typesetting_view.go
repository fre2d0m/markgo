package mdeditor

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

// TypesettingView uses MyGo's native text layout, selection and input-method
// client. The spans change typography only, never the characters being edited.
func TypesettingView(c *ui.Context, t *Typesetter, font ui.Font) ui.Element {
	t.Sync()
	if font.LineHeight == 0 {
		font.LineHeight = 1.75
	}
	font.Weight = 400
	font.Antialiased = true
	t.input.font = font
	t.input.width = t.input.handle.Bounds(c).W
	services := c.Services()
	th := c.Theme()
	options := richtext.EditorOptions{
		Label: "Typeset document", Placeholder: "A thought, a sentence, a beginning…",
		NoMarkdown: true, Font: font,
		Theme: &richtext.Theme{Text: th.Text, Caret: th.Text, Selection: th.Selection, ParagraphSpacing: 0.12},
		OnKey: func(key ui.Key, mods ui.Modifiers) bool { return t.key(key, mods, services) },
		Menu: func(m *ui.Menu) {
			if m.Item("Undo").Disabled(!t.Source.Buffer.CanUndo()).Chosen() {
				t.Undo(false)
			}
			if m.Item("Redo").Disabled(!t.Source.Buffer.CanRedo()).Chosen() {
				t.Undo(true)
			}
			m.Separator()
			a, z := t.Document.Selection()
			if m.Item("Copy").Disabled(a == z).Chosen() {
				services.WriteClipboard(t.Document.Text().Slice(min(a, z), max(a, z)).String())
			}
			if m.Item("Cut").Disabled(a == z).Chosen() {
				services.WriteClipboard(t.Document.Text().Slice(min(a, z), max(a, z)).String())
				t.DeleteSelection()
			}
			if m.Item("Paste text").Chosen() {
				t.Insert(services.ReadClipboard())
			}
			m.Separator()
			if m.Item("Select all").Chosen() {
				t.Document.SetSelection(0, t.Document.Text().Len())
			}
		},
		SpansVersion: t.projection.styleVersion,
		Spans: func(p int, _ richtext.ParagraphStyle, spans []ui.Span) []ui.Span {
			size, weight := font.Size, 400
			if p < len(t.projection.lines) {
				level := t.projection.lines[p].heading
				if level > 0 {
					size *= [...]float32{1, 2, 1.45, 1.2, 1.08, 1, 1}[level]
					weight = 600
				}
			}
			for i := range spans {
				spans[i] = ui.Span{Text: spans[i].Text, Font: font.Family, Size: size, Weight: weight, Color: th.Text, Underline: t.composing && spans[i].Underline}
			}
			return spans
		},
		Space: func(p int) float32 {
			if p == 0 || p >= len(t.projection.lines) {
				return 0
			}
			if t.projection.lines[p].heading > 0 {
				return font.Size * 0.65
			}
			return 0
		},
	}
	return richtext.EditorBase(c, t.Document, options).Bind(&t.input.handle).HandleTextInput(&t.input).ObserveInput(func(ev ui.InputEvent) { t.observe(ev, services) })
}

func (t *Typesetter) observe(ev ui.InputEvent, services ui.Services) {
	t.Sync()
	if !t.composing {
		t.beforeAnchor, t.beforeCaret = t.Document.Selection()
	}
	switch ev.Kind {
	case ui.InputPointerDown:
		t.CancelComposition()
	case ui.InputCommand:
		switch ev.Text {
		case "copy", "cut", "delete", "selectAll":
			if t.composing {
				t.CancelComposition()
			}
		default:
			t.CancelComposition()
			t.beforeAnchor, t.beforeCaret = t.Document.Selection()
			t.pendingCommand = ev.Text
			if ev.Text == "paste" || ev.Text == "pasteAndMatchStyle" {
				t.pendingText = services.ReadClipboard()
			}
		}
	}
	if ev.Kind != ui.InputText && ev.Kind != ui.InputCompose {
		t.input.ended = nil
	}
	if ev.Kind == ui.InputPointerDown || ev.Kind == ui.InputCommand || (ev.Kind == ui.InputKeyDown && (ev.Mods&ui.Cmd != 0 || ev.Key == ui.KeyLeft || ev.Key == ui.KeyRight || ev.Key == ui.KeyUp || ev.Key == ui.KeyDown || ev.Key == ui.KeyHome || ev.Key == ui.KeyEnd || ev.Key == ui.KeyEnter || ev.Key == ui.KeyBackspace || ev.Key == ui.KeyDelete)) {
		t.input.typing = nil
	}
}

func (t *Typesetter) key(key ui.Key, mods ui.Modifiers, services ui.Services) bool {
	if mods == ui.Cmd || mods == ui.Cmd|ui.Shift {
		switch key {
		case ui.KeyZ:
			t.Undo(mods&ui.Shift != 0)
			return true
		case ui.KeyY:
			t.Undo(true)
			return true
		case ui.KeyV:
			t.Insert(services.ReadClipboard())
			return true
		case ui.Key1:
			t.Heading(1)
			return true
		case ui.Key2:
			t.Heading(2)
			return true
		case ui.Key3:
			t.Heading(3)
			return true
		case ui.Key0:
			t.Heading(0)
			return true
		}
	}
	// Keep unsupported rich formatting from creating invisible, unsaved state.
	if mods&ui.Cmd != 0 {
		switch key {
		case ui.KeyB, ui.KeyI, ui.KeyU, ui.KeyE, ui.KeyK, ui.KeyL, ui.KeyR, ui.Key7, ui.Key8, ui.KeyBracketLeft, ui.KeyBracketRight, ui.KeyBackslash:
			t.Notice = "Use Markdown source for inline and list formatting."
			return true
		case ui.KeyX:
			if mods&ui.Shift != 0 {
				t.Notice = "Use Markdown source for inline formatting."
				return true
			}
		}
	}
	if key == ui.KeyEscape && t.composing {
		t.CancelComposition()
		return true
	}
	if mods != 0 || t.composing {
		return false
	}
	a, z := t.Document.Selection()
	if a != z {
		return false
	}
	for _, line := range t.projection.lines {
		if line.viewStart != z || line.heading == 0 {
			continue
		}
		switch key {
		case ui.KeyBackspace:
			t.Heading(0)
			return true
		case ui.KeyEnter:
			if line.content < line.end {
				t.Source.Select(line.start, line.start)
				t.Source.Insert("\n")
				t.Source.Select(line.content+1, line.content+1)
				t.FromSource()
				return true
			}
		}
	}
	return false
}
