package mdeditor

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

func shownAt(s *Typesetter, text string) int {
	shown := s.Document.Text().String()
	at := strings.Index(shown, text)
	if at < 0 {
		panic("missing fixture text: " + text)
	}
	return utf8.RuneCountInString(shown[:at])
}

func TestTypesettingStructuredBlocksPreserveSource(t *testing.T) {
	source := "# Notes\n\n> first\n>\n>> 中文🙂\n> lazy\n\n~~~~go title=sample\n# literal\n  text := `value`\n~~~~~\n\nafter\n"
	for _, ending := range []string{"\n", "\r\n"} {
		original := strings.ReplaceAll(source, "\n", ending)
		f := NewFile("blocks.md", original)
		s := NewTypesetter(f.State)
		if got := s.Document.Text().String(); got != "Notes\n\nfirst\n\n中文🙂\nlazy\n\n# literal\n  text := `value`\n\nafter\n" {
			t.Fatalf("projection: %q", got)
		}
		s.ToSource()
		s.FromSource()
		bytes, err := f.CaptureForSave().Bytes()
		if err != nil || string(bytes) != original {
			t.Fatalf("no-op changed source bytes: %q %v", bytes, err)
		}
		at := shownAt(s, "text :=")
		s.Document.SetSelection(at, at)
		s.Insert("new ")
		bytes, err = f.CaptureForSave().Bytes()
		if err != nil || string(bytes) != strings.Replace(original, "text :=", "new text :=", 1) {
			t.Fatalf("code edit changed untouched bytes or line endings: %q %v", bytes, err)
		}
		s.Undo(false)
		bytes, err = f.CaptureForSave().Bytes()
		if err != nil || string(bytes) != original {
			t.Fatalf("code undo changed saved bytes: %q %v", bytes, err)
		}
	}
}

func TestTypesettingQuoteWritingAndExit(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		at                 int
	}{
		{"continue", "> first", "> first\n> ", -1},
		{"split", "> abcd", "> ab\n> cd", 2},
		{"empty", "> ", "", 0},
		{"exit", "> first\n> ", "> first\n\n", -1},
		{"nested exit", ">> first\n>> ", ">> first\n>\n> ", -1},
		{"lazy", "> first\nlazy", "> first\nlazy\n> ", -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, tt := typesetTester(tc.source)
			at := tc.at
			if at < 0 {
				at = s.Document.Text().Len()
			}
			selectShown(s, tt, at, at)
			tt.Key(0, ui.KeyEnter)
			wantSource(t, s, tc.want)
			tt.Command("undo")
			wantSource(t, s, tc.source)
			tt.Command("redo")
			wantSource(t, s, tc.want)
		})
	}
	s, tt := typesetTester("> first\n> ")
	selectShown(s, tt, s.Document.Text().Len(), s.Document.Text().Len())
	tt.Key(0, ui.KeyEnter)
	tt.Type("paragraph")
	wantSource(t, s, "> first\n\nparagraph")
	if s.projection.lines[len(s.projection.lines)-1].block.quote != 0 {
		t.Fatal("quote exit became lazy continuation")
	}

	s, tt = typesetTester("> 中文🙂")
	selectShown(s, tt, 1, 2)
	tt.SetClipboard("new\n\nthought")
	tt.Command("paste")
	wantSource(t, s, "> 中new\n> \n> thought🙂")
	tt.Command("undo")
	wantSource(t, s, "> 中文🙂")
}

func TestTypesettingQuoteFormatIndentAndBackspace(t *testing.T) {
	s, tt := typesetTester("first\nsecond")
	selectShown(s, tt, s.Document.Text().Len(), 0)
	tt.Key(ui.Cmd|ui.Shift, ui.KeyQ)
	wantSource(t, s, "> first\n> second")
	tt.Key(0, ui.KeyTab)
	wantSource(t, s, "> > first\n> > second")
	tt.Key(ui.Shift, ui.KeyTab)
	wantSource(t, s, "> first\n> second")
	if a, z := s.Document.Selection(); a <= z {
		t.Fatal("quote indentation lost reversed selection")
	}
	selectShown(s, tt, shownAt(s, "second"), shownAt(s, "second"))
	tt.Key(0, ui.KeyBackspace)
	wantSource(t, s, "> first\n\nsecond")
	tt.Command("undo")
	wantSource(t, s, "> first\n> second")
	for _, source := range []string{"> - list", "> ```go\n> x\n> ```", "> [!NOTE]\n> alert", "```go\nx\n```", "# Heading"} {
		s, tt := typesetTester(source)
		selectShown(s, tt, 0, s.Document.Text().Len())
		s.Quote()
		wantSource(t, s, source)
		if s.Notice == "" {
			t.Fatalf("missing unsupported quote feedback for %q", source)
		}
	}
}

func TestTypesettingCodeReturnIndentAndExit(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		key                ui.Key
		mods               ui.Modifiers
	}{
		{"indent continue", "```go\n\tcall()\n```", "```go\n\tcall()\n\t\n```", ui.KeyEnter, 0},
		{"empty pair", "```\n```", "```\n\n```", ui.KeyEnter, 0},
		{"open pair", "```go", "```go\n\n```", ui.KeyEnter, 0},
		{"literal task", "```md\n- [ ] code\n```", "```md\n- [ ] code\n\n```", ui.KeyEnter, 0},
		{"plain indent", "```\nx\n```", "```\nx    \n```", ui.KeyTab, 0},
		{"go indent", "```go\nx\n```", "```go\nx\t\n```", ui.KeyTab, 0},
		{"closed exit", "```go\nx\n```", "```go\nx\n```\n\n", ui.KeyEnter, ui.Cmd},
		{"open exit", "~~~go\nx", "~~~go\nx\n~~~\n\n", ui.KeyEnter, ui.Cmd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, tt := typesetTester(tc.source)
			end := s.Document.Text().Len()
			selectShown(s, tt, end, end)
			tt.Key(tc.mods, tc.key)
			wantSource(t, s, tc.want)
			tt.Command("undo")
			wantSource(t, s, tc.source)
		})
	}
	s, tt := typesetTester("```go\nfirst\nsecond\n```\n\nafter")
	selectShown(s, tt, 0, 12)
	tt.Key(0, ui.KeyTab)
	wantSource(t, s, "```go\n\tfirst\n\tsecond\n```\n\nafter")
	tt.Key(ui.Shift, ui.KeyTab)
	wantSource(t, s, "```go\nfirst\nsecond\n```\n\nafter")
}

func TestTypesettingCodeTypingAndLiteralFences(t *testing.T) {
	for _, source := range []string{"```\n```", "~~~go", "```go\n\n```"} {
		s, tt := typesetTester(source)
		tt.Type("中文🙂")
		if s.Document.Text().String() != "中文🙂" {
			t.Fatalf("empty code typing: %q", s.Document.Text().String())
		}
		if len(s.projection.codes) != 1 {
			t.Fatal("typing destroyed the empty code block")
		}
	}
	s, tt := typesetTester("```md\n\n```")
	tt.Type("```")
	wantSource(t, s, "````md\n```\n````")
	if s.Document.Text().String() != "```" {
		t.Fatal("literal fence was interpreted as structure")
	}
	tt.Command("undo")
	wantSource(t, s, "```md\n\n```")
	for range 3 {
		tt.Type("`")
	}
	wantSource(t, s, "````md\n```\n````")
	tt.Command("undo")
	wantSource(t, s, "```md\n``\n```")
	s, tt = typesetTester("```go title=sample\nold\n```")
	selectShown(s, tt, 0, 0)
	tt.SetClipboard("before\n`````\nafter\n")
	tt.Command("paste")
	wantSource(t, s, "``````go title=sample\nbefore\n`````\nafter\nold\n``````")
	tt.Command("undo")
	wantSource(t, s, "```go title=sample\nold\n```")
}

func TestTypesettingCodeCrossBoundaryEditing(t *testing.T) {
	source := "before\n\n```go\nabcd\n```\n\nafter"
	for _, from := range []bool{true, false} {
		s, tt := typesetTester(source)
		if from {
			selectShown(s, tt, 1, shownAt(s, "abcd")+2)
		} else {
			selectShown(s, tt, shownAt(s, "abcd")+2, shownAt(s, "after")+2)
		}
		tt.Type("X")
		want := "before\n\nabXter"
		if from {
			want = "bXcd\n\nafter"
		}
		wantSource(t, s, want)
		tt.Command("undo")
		wantSource(t, s, source)
	}
}

func TestTypesettingCodeFormatLanguageClipboardAndIME(t *testing.T) {
	s, tt := typesetTester("print()")
	selectShown(s, tt, 0, 7)
	tt.Key(ui.Cmd|ui.Shift, ui.KeyK)
	wantSource(t, s, "```\nprint()\n```")
	tt.Command("undo")
	wantSource(t, s, "print()")

	source := "before\n\n~~~ go title=sample\n中文🙂\n~~~~\n\nafter"
	s, tt = typesetTester(source)
	at := shownAt(s, "中文")
	selectShown(s, tt, at, at+3)
	if !s.SetCodeLanguage(2, "python") {
		t.Fatal("language change rejected")
	}
	tt.Frame()
	wantSource(t, s, strings.Replace(source, " go ", " python ", 1))
	s.input.SetMarkedText(nil, "nihao", ui.TextInputRange{Start: 5, End: 5})
	wantSource(t, s, strings.Replace(source, " go ", " python ", 1))
	s.input.ReplaceText(nil, "你好🙂")
	tt.Frame()
	tt.Command("undo")
	tt.Command("undo")
	wantSource(t, s, source)
	selectShown(s, tt, at, at+3)
	tt.Command("copy")
	if tt.Clipboard() != "~~~ go title=sample\n中文🙂\n~~~~" {
		t.Fatalf("code clipboard: %q", tt.Clipboard())
	}
	tt.Command("cut")
	wantSource(t, s, "before\n\n\n\nafter")
	tt.Command("undo")
	wantSource(t, s, source)
}

func TestTypesettingCodeBoundariesAndReversedUndo(t *testing.T) {
	for _, fromEmpty := range []bool{true, false} {
		source := "before\n\n```go\n```\n\nafter"
		s, tt := typesetTester(source)
		after := shownAt(s, "after")
		want := "before\n\nXfter"
		if fromEmpty {
			selectShown(s, tt, after-2, after+1)
		} else {
			selectShown(s, tt, 1, after-2)
			want = "bX\n\nafter"
		}
		tt.Type("X")
		wantSource(t, s, want)
		tt.Command("undo")
		wantSource(t, s, source)
	}
	source := "before\n\n```go\nabcd\n```\n\nbetween\n\n~~~py\nwxyz\n~~~\n\nafter"
	s, tt := typesetTester(source)
	a, z := shownAt(s, "abcd")+2, shownAt(s, "wxyz")+2
	selectShown(s, tt, z, a)
	tt.Type("X")
	wantSource(t, s, "before\n\nabXyz\n\nafter")
	tt.Command("undo")
	wantSource(t, s, source)
	if anchor, caret := s.Document.Selection(); anchor != z || caret != a {
		t.Fatalf("paired edit undo lost reversed selection: %d,%d want %d,%d", anchor, caret, z, a)
	}
	tt.Command("redo")
	wantSource(t, s, "before\n\nabXyz\n\nafter")
	for _, source := range []string{"```\n```", "```go", "```go\n\n```"} {
		s, tt := typesetTester(source)
		tt.Key(0, ui.KeyBackspace)
		wantSource(t, s, "")
		tt.Command("undo")
		wantSource(t, s, source)
	}
	for _, source := range []string{"---\nexample: |\n  ```go\n  literal\n  ```\n---\n\ntext", "> # Heading\n> paragraph", "> - item\n>   - nested", "    ```go\n    literal\n    ```"} {
		p := projectTypesetting(source)
		if p.text != source || len(p.codes) != 0 {
			t.Fatalf("unsupported structure changed: %q -> %q", source, p.text)
		}
	}
}

func BenchmarkTypesettingBlocksEditCycle100K(b *testing.B) {
	source := strings.Repeat("> 一点想法 and a quiet sentence.\n\n```go\nfmt.Println(\"中文🙂\")\n```\n\n", 1800)
	s, tt := typesetTester(source)
	selectShown(s, tt, s.Document.Text().Len()/2, s.Document.Text().Len()/2)
	b.ReportAllocs()
	for b.Loop() {
		tt.Type("中🙂")
		tt.Key(ui.Cmd, ui.KeyZ)
	}
	if s.Source.Buffer.String() != source {
		b.Fatal("benchmark did not restore its fixture")
	}
}
