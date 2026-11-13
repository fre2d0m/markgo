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
	t.input.gutter = t.listGutterWidth(font)
	t.input.width = t.input.handle.Bounds(c).W
	services := c.Services()
	t.services = services
	th := c.Theme()
	options := richtext.EditorOptions{
		Label: "Typeset document", Placeholder: "A thought, a sentence, a beginning…",
		NoMarkdown: true, Font: font,
		Theme: &richtext.Theme{Text: th.Text, Caret: th.Text, Selection: th.Selection, ParagraphSpacing: 0.12,
			CodeFont: "monospace", CodeSize: 0.9, CodeBlockBackground: th.Surface, CodeBlockRadius: 8 / font.Size, CodeBlockPadding: [2]float32{16 / font.Size, 10 / font.Size}},
		OnKey:         func(key ui.Key, mods ui.Modifiers) bool { return t.key(key, mods, services) },
		OnLinkCommand: func() { t.OpenLink() },
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
				t.copySelection(services, false)
			}
			if m.Item("Cut").Disabled(a == z).Chosen() {
				t.copySelection(services, true)
			}
			if m.Item("Paste text").Chosen() {
				t.Insert(services.ReadClipboard())
			}
			m.Separator()
			for _, format := range []struct {
				label string
				kind  InlineKind
			}{{"Bold", InlineBold}, {"Italic", InlineItalic}, {"Strikethrough", InlineStrike}, {"Inline code", InlineCode}} {
				if m.Item(format.label).Chosen() {
					t.ToggleInline(format.kind)
				}
			}
			if m.Item("Edit link…").Chosen() {
				t.OpenLink()
			}
			m.Separator()
			if m.Item("Bulleted list").Chosen() {
				t.SetList(BulletedList)
			}
			if m.Item("Numbered list").Chosen() {
				t.SetList(NumberedList)
			}
			if m.Item("Task list").Chosen() {
				t.SetList(TaskList)
			}
			if m.Item("Toggle task").Chosen() {
				t.ToggleTask(-1)
			}
			if m.Item("Quote").Chosen() {
				t.Quote()
			}
			if m.Item("Code block").Chosen() {
				t.CodeBlock()
			}
			m.Separator()
			if m.Item("Select all").Chosen() {
				t.Document.SetSelection(0, t.Document.Text().Len())
			}
		},
		SpansVersion: t.projection.styleVersion,
		Spans: func(p int, _ richtext.ParagraphStyle, spans []ui.Span) []ui.Span {
			return t.styleSpans(p, font, spans, th)
		},
		Space: func(p int) float32 {
			line, ok := t.projection.shownLine(p)
			if !ok {
				return 0
			}
			if line.block.code > 0 && t.projection.visible[p] == t.projection.codes[line.block.code-1].first {
				return 44
			}
			if line.list.kind != NoList && line.list.kind != listContinuation {
				return max(0, 40-font.Size*font.LineHeight)
			}
			if p > 0 && line.heading > 0 {
				return font.Size * 0.65
			}
			return 0
		},
	}
	if t.Source.Buffer.Len() > 0 {
		options.Placeholder = ""
	}
	options.Theme.Indent = t.input.gutter / font.Size
	view := richtext.EditorBase(c, t.Document, options).Bind(&t.input.handle).HandleTextInput(&t.input).ObserveInput(func(ev ui.InputEvent) { t.observe(ev, services) }).Clip()
	t.listMarkers(c, view, font)
	t.quoteBars(c, view)
	t.codeHeaders(c, view)
	t.linkPopover(c, view)
	return view
}

func (t *Typesetter) observe(ev ui.InputEvent, services ui.Services) {
	t.Sync()
	if !t.composing {
		t.beforeAnchor, t.beforeCaret = t.Document.Selection()
	}
	switch ev.Kind {
	case ui.InputPointerDown:
		t.CancelComposition()
		t.caretAffinity = false
	case ui.InputCommand:
		switch ev.Text {
		case "delete", "selectAll":
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
	if !t.composing && key == ui.KeyEscape && t.pendingStyles != 0 {
		t.CancelInlineIntent()
		return true
	}
	if !t.composing && mods == 0 && (key == ui.KeyLeft || key == ui.KeyRight) && t.inlineArrow(key) {
		return true
	}
	if key == ui.KeyUp || key == ui.KeyDown || key == ui.KeyHome || key == ui.KeyEnd {
		t.caretAffinity = false
	}
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
		case ui.KeyC, ui.KeyX:
			if mods == ui.Cmd {
				t.copySelection(services, key == ui.KeyX)
				return true
			}
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
	if !t.composing {
		if mods == ui.Cmd {
			switch key {
			case ui.KeyK:
				t.OpenLink()
				return true
			case ui.KeyB:
				t.ToggleInline(InlineBold)
				return true
			case ui.KeyI:
				t.ToggleInline(InlineItalic)
				return true
			case ui.KeyE:
				t.ToggleInline(InlineCode)
				return true
			}
		}
		if mods == ui.Cmd|ui.Shift {
			switch key {
			case ui.KeyX:
				t.ToggleInline(InlineStrike)
				return true
			case ui.Key7:
				t.SetList(NumberedList)
				return true
			case ui.Key8:
				t.SetList(BulletedList)
				return true
			case ui.Key9, ui.KeyL:
				t.SetList(TaskList)
				return true
			case ui.KeyQ:
				t.Quote()
				return true
			case ui.KeyK:
				t.CodeBlock()
				return true
			}
		}
		if key == ui.KeyEnter && mods == ui.Cmd {
			if t.ContinueAfterCode(-1) {
				return true
			}
			return t.ToggleTask(-1)
		}
		if (key == ui.KeyTab && (mods == 0 || mods == ui.Shift)) || (mods == ui.Cmd && (key == ui.KeyBracketLeft || key == ui.KeyBracketRight)) {
			if t.indentBlocks(mods == ui.Shift || key == ui.KeyBracketLeft) {
				return true
			}
		}
		if key == ui.KeyEnter && mods == 0 && (t.codeEnter() || t.quoteEnter() || t.listEnter()) {
			return true
		}
	}
	// Keep unsupported rich formatting from creating invisible, unsaved state.
	if mods&ui.Cmd != 0 {
		switch key {
		case ui.KeyB, ui.KeyI, ui.KeyU, ui.KeyE, ui.KeyK, ui.KeyL, ui.KeyR, ui.Key7, ui.Key8, ui.KeyBracketLeft, ui.KeyBracketRight, ui.KeyBackslash:
			t.Notice = "Use Markdown source for this formatting."
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
	index := t.projection.sourceLine(t.projection.sourceOffset(z))
	line := t.projection.lines[index]
	if key == ui.KeyBackspace && z == line.viewStart {
		if code, ok := t.projection.codeAt(index); ok && index == code.first {
			t.CodeBlock()
			return true
		}
		if line.block.quote > 0 {
			t.ToSource()
			t.changeQuote(index, index, true)
			return true
		}
	}
	if key == ui.KeyBackspace && line.list.kind != NoList && z == line.viewStart {
		if line.list.kind == listContinuation && z > 0 {
			t.Document.SetSelection(z-1, z)
			t.DeleteSelection()
			return true
		}
		return t.IndentList(true)
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

func (t *Typesetter) indentBlocks(remove bool) bool {
	return t.indentCode(remove) || t.indentQuote(remove) || t.IndentList(remove)
}
