package mdeditor

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/richtext"
)

func TestTypesettingInlineProjection(t *testing.T) {
	for _, tc := range []struct{ source, shown string }{
		{"A **bold *inside*** and __strong__ ~~gone~~ ` code `.", "A bold inside and strong gone code."},
		{"[**中文🙂**](<https://example.test/a(b)> \"Title\") [ref][id]\n\n[id]: /dest", "中文🙂 ref\n\n[id]: /dest"},
		{"- [ ] a **bold**\n> _quote_\n\n## **Title**", "a bold\nquote\n\nTitle"},
		{"**a\nb**", "a\nb"},
		{"a \\*escaped\\* &amp; **c**", "a \\*escaped\\* &amp; c"},
		{"```md\n**literal**\n```", "**literal**"},
		{"![**alt**](image.png)", "![**alt**](image.png)"},
		{"`` `code` `` and `   `", "`code` and    "},
	} {
		p := projectTypesetting(tc.source)
		if p.text != tc.shown {
			t.Errorf("%q: got %q, want %q", tc.source, p.text, tc.shown)
		}
		for at := 0; at <= p.len(); at++ {
			if p.viewOffset(p.sourceOffset(at)) != at {
				t.Fatalf("map at %d in %q", at, tc.source)
			}
		}
		p.update("前言🙂\n\n" + tc.source)
		checkProjection(t, &p, "前言🙂\n\n"+tc.source)
		p.update(tc.source)
		checkProjection(t, &p, tc.source)
	}
}

func TestTypesettingInlineAttributesAndPartialToggle(t *testing.T) {
	s, _ := typesetTester("a **bold _inside_** ~~gone~~ `code` [link](dest)")
	for _, tc := range []struct {
		word  string
		check func(richtext.Attributes) bool
	}{
		{"bold", func(a richtext.Attributes) bool { return a.Weight == 700 }},
		{"inside", func(a richtext.Attributes) bool { return a.Weight == 700 && a.Italic }},
		{"gone", func(a richtext.Attributes) bool { return a.Strikethrough }},
		{"code", func(a richtext.Attributes) bool { return a.Code }},
		{"link", func(a richtext.Attributes) bool { return a.Underline && a.Link == "" }},
	} {
		at := strings.Index(s.Document.Text().String(), tc.word)
		if !tc.check(s.Document.Text().AttributesAt(at)) {
			t.Fatalf("lost %s attributes", tc.word)
		}
	}
	s, tt := typesetTester("a **bold words** z")
	selectShown(s, tt, 7, 11)
	tt.Key(ui.Cmd, ui.KeyB)
	wantSource(t, s, "a **bold** word**s** z")
	tt.Key(ui.Cmd, ui.KeyZ)
	wantSource(t, s, "a **bold words** z")
}

func TestTypesettingInlineNativeCommandsClipboardAndIME(t *testing.T) {
	for _, command := range []string{"bold", "italic", "strikethrough", "code"} {
		s, tt := typesetTester("a 中文🙂 z")
		selectShown(s, tt, 5, 2)
		tt.Command(command)
		if s.Document.Text().String() != "a 中文🙂 z" || s.Source.Buffer.String() == "a 中文🙂 z" {
			t.Fatalf("native %s did not edit source", command)
		}
		tt.Command("undo")
		wantSource(t, s, "a 中文🙂 z")
	}
	s, tt := typesetTester("a **中文🙂** z\n\nlast")
	selectShown(s, tt, 5, 2)
	tt.Command("copy")
	if tt.Clipboard() != "中文🙂" {
		t.Fatalf("partial clipboard %q", tt.Clipboard())
	}
	tt.Compose("nihao", 5)
	wantSource(t, s, "a **中文🙂** z\n\nlast")
	tt.Key(0, ui.KeyEscape)
	wantSource(t, s, "a **中文🙂** z\n\nlast")
	tt.Compose("shijie", 6)
	tt.Type("世界")
	wantSource(t, s, "a **世界** z\n\nlast")
	tt.Command("undo")
	wantSource(t, s, "a **中文🙂** z\n\nlast")
	selectShown(s, tt, 0, 7)
	tt.Command("copy")
	if tt.Clipboard() != "a **中文🙂** z" {
		t.Fatalf("whole clipboard %q", tt.Clipboard())
	}
	tt.SetClipboard("**替换**")
	selectShown(s, tt, 5, 2)
	tt.Command("paste")
	if strings.Contains(s.Document.Text().String(), "*") {
		t.Fatalf("paste damaged nesting: %q", s.Source.Buffer.String())
	}
}

func TestTypesettingInlineWrappedNativeGeometry(t *testing.T) {
	for _, size := range []float32{14, 18, 24} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			source := "## **A bold heading**\n\n" + strings.Repeat("A **bold 中文🙂** and *italic phrase* with `code(value)` and [a link](note.md). ", 7)
			s := NewTypesetter(NewFile("inline.md", source).State)
			tt := ui.NewTester(func(c *ui.Context) { TypesettingView(c, s, ui.Font{Family: "serif", Size: size}).Fill().AutoFocus() }, 400, 560)
			tt.SetScale(map[float32]float32{14: 1, 18: 1.5, 24: 2}[size])
			for _, at := range []int{1, 5, 22, 30, 48, 65, 90, 120, 180} {
				selectShown(s, tt, at, at)
				r, ok := s.Document.CaretRect()
				if !ok {
					t.Fatal("missing caret")
				}
				index, hit := s.input.IndexForPoint(ui.Point{X: r.X + 0.1, Y: r.Y + r.H/2})
				if want := nativeOffset(s.Document.Text().String(), at); !hit || index != want {
					t.Fatalf("at %d: %d/%v, want %d at %+v", at, index, hit, want, r)
				}
			}
			at := s.Document.Text().Len() / 2
			selectShown(s, tt, at, at)
			before, _ := s.Document.CaretRect()
			tt.Type("中文🙂")
			tt.Command("undo")
			wantSource(t, s, source)
			after, _ := s.Document.CaretRect()
			if math.Abs(float64(before.Y-after.Y)) > 1 {
				t.Fatal("undo moved writing place")
			}
		})
	}
}

func TestTypesettingLinkPanelHistoryAndFocus(t *testing.T) {
	source := "Read [**the notes**](<notes/a(b).md> \"Old title\") again."
	s, tt := typesetTester(source)
	selectShown(s, tt, 7, 7)
	tt.Key(ui.Cmd, ui.KeyK)
	if !s.linkUI.open || !s.linkUI.existing || s.linkUI.label != "the notes" {
		t.Fatalf("wrong link panel: %+v", s.linkUI)
	}
	for _, label := range []string{"Link text", "Link address", "Apply link", "Cancel link edit", "Remove link"} {
		r, ok := tt.Find(label)
		if !ok || r.X < 0 || r.X+r.W > 760 || r.Y < 0 || r.Y+r.H > 560 {
			t.Fatalf("clipped %s: %+v", label, r)
		}
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("next.md")
	blockClick(t, tt, "Apply link")
	wantSource(t, s, strings.Replace(source, "notes/a(b).md", "next.md", 1))
	if a, z := s.Document.Selection(); a != 7 || z != 7 {
		t.Fatalf("address change moved caret: %d,%d", a, z)
	}
	if !tt.Focused("Typeset document") {
		t.Fatal("apply did not restore focus")
	}
	tt.Command("undo")
	wantSource(t, s, source)
	tt.Key(ui.Cmd, ui.KeyK)
	blockClick(t, tt, "Link address")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	blockClick(t, tt, "Apply link")
	if !s.linkUI.open || s.linkUI.error == "" || s.linkUI.errorField != "address" {
		t.Fatal("invalid address closed panel")
	}
	wantSource(t, s, source)
	blockClick(t, tt, "Link text")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	blockClick(t, tt, "Apply link")
	if s.linkUI.errorField != "label" {
		t.Fatal("an invalid label marked the address instead")
	}
	wantSource(t, s, source)
	tt.Key(0, ui.KeyEscape)
	if !tt.Focused("Typeset document") {
		t.Fatal("Escape lost focus")
	}
	wantSource(t, s, source)
	tt.Key(ui.Cmd, ui.KeyK)
	blockClick(t, tt, "Remove link")
	wantSource(t, s, "Read **the notes** again.")
	tt.Command("undo")
	wantSource(t, s, source)
}

func TestTypesettingReferenceLinkEditIsLocal(t *testing.T) {
	source := "[one][ref] and [two][ref]\n\n[ref]: old.md \"Title\""
	s, tt := typesetTester(source)
	selectShown(s, tt, 1, 1)
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("next.md")
	tt.Key(0, ui.KeyEnter)
	wantSource(t, s, "[one](<next.md> \"Title\") and [two][ref]\n\n[ref]: old.md \"Title\"")
	tt.Command("undo")
	wantSource(t, s, source)
}

func BenchmarkTypesettingInlineEditCycle100K(b *testing.B) {
	source := strings.Repeat("## A **quiet** paragraph\n\nSome **words**, _一些中文🙂_ and `a call()` with [a note](notes.md).\n\n", 1600)
	s, tt := typesetTester(source)
	at := s.Document.Text().Len() / 2
	selectShown(s, tt, at, at)
	b.ReportAllocs()
	for b.Loop() {
		tt.Type("中🙂")
		tt.Key(ui.Cmd, ui.KeyZ)
	}
	if s.Source.Buffer.String() != source {
		b.Fatal("benchmark changed source")
	}
}

func TestTypesettingInlineCacheMatchesFreshStyles(t *testing.T) {
	source := "# A **heading**\n\n**中文🙂 _inside_**\n\n[link][ref]\n\n[ref]: old.md\n\n- [ ] ~~task~~\n\n`code`"
	s := NewTypesetter(NewFile("styles.md", source).State)
	check := func() {
		t.Helper()
		fresh := NewTypesetter(NewFile("fresh.md", s.Source.Buffer.String()).State)
		if s.Document.Text().String() != fresh.Document.Text().String() {
			t.Fatal("cache changed text")
		}
		for i := 0; i < s.Document.Text().Len(); i++ {
			if s.Document.Text().AttributesAt(i) != fresh.Document.Text().AttributesAt(i) {
				t.Fatalf("stale attributes at %d in %q", i, s.Source.Buffer.String())
			}
		}
	}
	for _, snippet := range []string{"\n", "**", "_", "a", "中文", "[ref]: new.md\n\n", "~~", "`"} {
		for _, where := range []int{0, 3, len(source) / 2, len(source)} {
			at := runeByte(source, where)
			s.Source.Buffer.Edit(editor.Edit{Start: at, End: at, Text: snippet})
			s.FromSource()
			check()
			s.Source.Run(editor.Undo)
			s.FromSource()
			check()
		}
	}
}

func TestTypesettingInlineEmptyIntentAndDirectNativeInput(t *testing.T) {
	s, tt := typesetTester("")
	tt.Key(ui.Cmd, ui.KeyB)
	wantSource(t, s, "")
	s.input.SetMarkedText(nil, "nihao", ui.TextInputRange{Start: 5, End: 5})
	tt.Frame()
	wantSource(t, s, "")
	s.input.ReplaceText(nil, "你好🙂")
	tt.Frame()
	wantSource(t, s, "**你好🙂**")
	tt.Command("undo")
	wantSource(t, s, "")
	s, tt = typesetTester("a [中文🙂](dest) z")
	selectShown(s, tt, 2, 5)
	s.input.SetMarkedText(nil, "中文🙂", ui.TextInputRange{Start: 4, End: 4})
	tt.Frame()
	s.input.UnmarkText()
	tt.Frame()
	wantSource(t, s, "a [中文🙂](dest) z")
	if !s.Document.Text().AttributesAt(3).Underline {
		t.Fatal("unchanged composition erased link style")
	}
	tt.Command("underline")
	if !s.Document.Text().AttributesAt(3).Underline {
		t.Fatal("unsupported native formatting erased link style")
	}
}

func TestTypesettingInlineIntentKeepsInvalidBoundariesLiteral(t *testing.T) {
	s, tt := typesetTester("code")
	selectShown(s, tt, 2, 2)
	tt.Key(ui.Cmd, ui.KeyI)
	tt.Type("字")
	wantSource(t, s, "co*字*de")
	s, tt = typesetTester("")
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Type(" padded ")
	wantSource(t, s, " padded ")
	s, tt = typesetTester("")
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Key(0, ui.KeyEscape)
	tt.Type("word")
	wantSource(t, s, "word")
	s, tt = typesetTester("")
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Type("word")
	tt.SetClipboard("s")
	tt.Command("paste")
	wantSource(t, s, "**words**")
	tt.Command("bold")
	wantSource(t, s, "words")
}

func TestTypesettingInlineSaveKeepsCRLFAndOriginalDelimiters(t *testing.T) {
	source := "## __Title__\n\na _word_ and [link][id].\n\n[id]: <notes/a(b).md> \"A title\"\n"
	f := NewFile("lines.md", strings.ReplaceAll(source, "\n", "\r\n"))
	s := NewTypesetter(f.State)
	s.Document.SetSelection(1, 1)
	s.Insert("中")
	want := strings.Replace(source, "Title", "T中itle", 1)
	data, err := f.CaptureForSave().Bytes()
	if err != nil || string(data) != strings.ReplaceAll(want, "\n", "\r\n") {
		t.Fatalf("changed line endings: %q / %v", data, err)
	}
	s.Undo(false)
	wantSource(t, s, source)
}

func TestTypesettingInlineEdits(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		a, z         int
		text, want   string
	}{
		{"inside", "a **bold** z", 3, 3, "中", "a **b中old** z"},
		{"replace", "a **bold** z", 2, 6, "new", "a **new** z"},
		{"delete full", "a **bold** z", 2, 6, "", "a  z"},
		{"leave", "a **bold** z", 4, 8, "!", "a bo!"},
		{"enter", "a **bold** z", 0, 4, "!", "!ld z"},
		{"nested", "a **bold _inside_** z", 2, 13, "", "a  z"},
		{"link", "a [text](target) z", 3, 5, "中", "a [t中t](target) z"},
		{"link crossing", "a [text](target) z", 4, 8, "!", "a te!"},
		{"code tick", "a `code` z", 4, 4, "`", "a ``co`de`` z"},
		{"retain code run", "a ```code``` z", 4, 4, "`", "a ```co`de``` z"},
		{"code spaces", "a `code` z", 2, 6, " code ", "a `  code  ` z"},
		{"keep code padding", "a ` code ` z", 2, 3, "C", "a ` Code ` z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, tt := typesetTester(tc.source)
			selectShown(s, tt, tc.a, tc.z)
			if tc.text == "" {
				tt.Key(0, ui.KeyBackspace)
			} else {
				tt.Type(tc.text)
			}
			wantSource(t, s, tc.want)
			checkProjection(t, &s.projection, tc.want)
			tt.Key(ui.Cmd, ui.KeyZ)
			wantSource(t, s, tc.source)
			tt.Key(ui.Cmd|ui.Shift, ui.KeyZ)
			wantSource(t, s, tc.want)
		})
	}
}

func TestTypesettingInlineCommands(t *testing.T) {
	for _, tc := range []struct {
		kind InlineKind
		key  ui.Key
		mods ui.Modifiers
		want string
	}{
		{InlineBold, ui.KeyB, ui.Cmd, "a **中文🙂** z"},
		{InlineItalic, ui.KeyI, ui.Cmd, "a _中文🙂_ z"},
		{InlineStrike, ui.KeyX, ui.Cmd | ui.Shift, "a ~~中文🙂~~ z"},
		{InlineCode, ui.KeyE, ui.Cmd, "a `中文🙂` z"},
	} {
		s, tt := typesetTester("a 中文🙂 z")
		selectShown(s, tt, 5, 2)
		tt.Key(tc.mods, tc.key)
		wantSource(t, s, tc.want)
		if !s.InlineActive(tc.kind) {
			t.Fatal("format is not active")
		}
		if a, z := s.Document.Selection(); a != 5 || z != 2 {
			t.Fatalf("selection %d:%d", a, z)
		}
		tt.Key(tc.mods, tc.key)
		wantSource(t, s, "a 中文🙂 z")
		tt.Key(ui.Cmd, ui.KeyZ)
		wantSource(t, s, tc.want)
	}
}

func TestTypesettingInlineTypingIntentAndBoundary(t *testing.T) {
	s, tt := typesetTester("a ")
	selectShown(s, tt, 2, 2)
	tt.Key(ui.Cmd, ui.KeyB)
	wantSource(t, s, "a ")
	tt.Type("word")
	wantSource(t, s, "a **word**")
	tt.Key(0, ui.KeyRight)
	tt.Type(".")
	wantSource(t, s, "a **word**.")
	tt.Key(ui.Cmd, ui.KeyZ)
	wantSource(t, s, "a **word**")
	tt.Key(ui.Cmd, ui.KeyZ)
	wantSource(t, s, "a ")
	if strings.Contains(s.Document.Text().String(), "*") {
		t.Fatal("pending intent changed source")
	}
}
