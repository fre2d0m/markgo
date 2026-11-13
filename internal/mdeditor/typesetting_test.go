package mdeditor

import (
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

func typesetTester(source string) (*Typesetter, *ui.Tester) {
	f := NewFile("sample.md", source)
	t := NewTypesetter(f.State)
	tt := ui.NewTester(func(c *ui.Context) {
		TypesettingView(c, t, ui.Font{Family: "system-ui", Size: 18}).Fill().AutoFocus()
	}, 760, 560)
	return t, tt
}

func selectShown(t *Typesetter, tt *ui.Tester, a, z int) { t.Document.SetSelection(a, z); tt.Frame() }

func wantSource(t *testing.T, s *Typesetter, want string) {
	t.Helper()
	if got := s.Source.Buffer.String(); got != want {
		t.Fatalf("source\n got %q\nwant %q", got, want)
	}
}

func TestTypesettingPreservesUneditedSyntaxAndLineEndings(t *testing.T) {
	source := "---\n# metadata: literal\n---\n\n# 标题 ###\n\n正文🙂\n\n```go\n# not a heading\nfmt.Println(1)\n```\n\n| a | b |\n| - | - |\n| 1 | 2 |\n\n[link][ref]\n\n[ref]: <https://example.test> \"title\"\n\nSetext\n======\n"
	for _, ending := range []string{"\n", "\r\n"} {
		f := NewFile("sample.md", strings.ReplaceAll(source, "\n", ending))
		s := NewTypesetter(f.State)
		tt := ui.NewTester(func(c *ui.Context) { TypesettingView(c, s, ui.Font{Size: 18}).Fill().AutoFocus() }, 760, 560)
		shown := s.Document.Text().String()
		if strings.Contains(shown, "# 标题") || !strings.Contains(shown, "# metadata") || !strings.Contains(shown, "# not a heading") || !strings.Contains(shown, "======") {
			t.Fatalf("projection changed unsupported syntax: %q", shown)
		}
		at := utf8.RuneCountInString(shown[:strings.Index(shown, "正文")]) + 2
		selectShown(s, tt, at, at)
		tt.Type("，继续")
		want := strings.Replace(source, "正文", "正文，继续", 1)
		wantSource(t, s, want)
		data, err := f.CaptureForSave().Bytes()
		if err != nil || string(data) != strings.ReplaceAll(want, "\n", ending) {
			t.Fatalf("save bytes changed: %q, %v", data, err)
		}
		tt.Key(ui.Cmd, ui.KeyZ)
		wantSource(t, s, source)
	}
}

func TestTypesettingCrossBlockEditsAndSelection(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		a, z         int
		key          ui.Key
		text, want   string
	}{
		{"join heading", "one\n## two", 3, 3, ui.KeyDelete, "", "onetwo"},
		{"cross selection", "# One\n\n## Two\n\nlast", 1, 9, 0, "中🙂", "# O中🙂\nlast"},
		{"reversed selection", "# One\n\n## Two\n\nlast", 9, 1, 0, "中🙂", "# O中🙂\nlast"},
		{"select all", "# One\n\n## Two", 0, 8, ui.KeyBackspace, "", ""},
		{"unstyle first", "## Title", 0, 0, ui.KeyBackspace, "", "Title"},
		{"split start", "## Title", 0, 0, ui.KeyEnter, "", "\n## Title"},
		{"split middle", "## Title", 2, 2, ui.KeyEnter, "", "## Ti\ntle"},
		{"split end", "## Title", 5, 5, ui.KeyEnter, "", "## Title\n"},
		{"emoji offset", "# 你🙂好", 2, 2, 0, "世界", "# 你🙂世界好"},
		{"repeat boundary", "# aaa\n\n# aaa", 5, 5, 0, "a", "# aaa\n\n# aaaa"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, tt := typesetTester(tc.source)
			selectShown(s, tt, tc.a, tc.z)
			if tc.key != 0 {
				tt.Key(0, tc.key)
			} else {
				tt.Type(tc.text)
			}
			wantSource(t, s, tc.want)
			tt.Key(ui.Cmd, ui.KeyZ)
			wantSource(t, s, tc.source)
			tt.Key(ui.Cmd|ui.Shift, ui.KeyZ)
			wantSource(t, s, tc.want)
		})
	}
}

func TestTypesettingIMEDoesNotCommitPreeditOrLoseSelection(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		s, tt := typesetTester("# 标题\n\n原文🙂\n\n结尾")
		selectShown(s, tt, 4, 7)
		tt.Compose("ni", 2)
		wantSource(t, s, "# 标题\n\n原文🙂\n\n结尾")
		tt.Compose("nihao", 5)
		wantSource(t, s, "# 标题\n\n原文🙂\n\n结尾")
		if active, noIME := tt.TextInputActive(); !active || noIME {
			t.Fatal("native input method unavailable")
		}
		if cancel {
			tt.Compose("", 0)
			wantSource(t, s, "# 标题\n\n原文🙂\n\n结尾")
			tt.Type("替换")
			wantSource(t, s, "# 标题\n\n替换\n\n结尾")
		} else {
			tt.Type("你好")
			wantSource(t, s, "# 标题\n\n你好\n\n结尾")
		}
		tt.Key(ui.Cmd, ui.KeyZ)
		wantSource(t, s, "# 标题\n\n原文🙂\n\n结尾")
	}
}

func TestTypesettingNativeCommandsAndPlainPaste(t *testing.T) {
	s, tt := typesetTester("# Title\n\ntext")
	selectShown(s, tt, 7, 11)
	paste := "## 中🙂\n\n```go\ncode\n```"
	tt.SetClipboardData(transfer.New(transfer.NewItem(transfer.Bytes(transfer.HTML, []byte("<h1>Wrong</h1>")), transfer.Bytes(transfer.Text, []byte(paste)))))
	tt.Command("paste")
	wantSource(t, s, "# Title\n\n"+paste)
	tt.Command("undo")
	wantSource(t, s, "# Title\n\ntext")
	tt.Command("redo")
	wantSource(t, s, "# Title\n\n"+paste)
	tt.Command("bold")
	wantSource(t, s, "# Title\n\n"+paste)
	if s.Notice == "" {
		t.Fatal("unsupported formatting needs feedback")
	}
	tt.Command("selectAll")
	tt.Command("delete")
	wantSource(t, s, "")
	tt.Command("undo")
	wantSource(t, s, "# Title\n\n"+paste)
}

func TestTypesettingSourceModeSharesHistoryAndReversedSelection(t *testing.T) {
	s, tt := typesetTester("# Title\n\n正文")
	selectShown(s, tt, 7, 9)
	tt.Type("变化")
	s.ToSource()
	s.Source.Select(s.Source.Buffer.Len(), s.Source.Buffer.Len())
	s.Source.Insert(" source")
	s.FromSource()
	tt.Frame()
	tt.Command("undo")
	wantSource(t, s, "# Title\n\n变化")
	tt.Command("undo")
	wantSource(t, s, "# Title\n\n正文")
	selectShown(s, tt, 9, 1)
	s.ToSource()
	if got := s.Source.Selection(); got.Anchor != 15 || got.Head != 3 {
		t.Fatalf("reverse selection: %+v", got)
	}
	s.FromSource()
	if a, z := s.Document.Selection(); a != 9 || z != 1 {
		t.Fatalf("reverse projection: %d,%d", a, z)
	}
	// Source editor commands still use the very same history.
	s.Source.Run(editor.Redo)
	s.FromSource()
	wantSource(t, s, "# Title\n\n变化")
}

func BenchmarkTypesettingProjection(b *testing.B) {
	source := strings.Repeat("## A quiet paragraph\n\nSome words, 一些中文 and an emoji 🙂.\n\n", 1600)
	b.SetBytes(int64(len(source)))
	b.ReportAllocs()
	for b.Loop() {
		_ = projectTypesetting(source)
	}
}

func TestTypesettingDirectNativeIMEAndUnmarkCommit(t *testing.T) {
	for _, unmark := range []bool{false, true} {
		s, tt := typesetTester("# Title\n\n原文🙂")
		selectShown(s, tt, 7, 10)
		n := &s.input
		// Call the native client directly: no ObserveInput or Tester.Compose.
		n.SetMarkedText(nil, "nihao", ui.TextInputRange{Start: 5, End: 5})
		tt.Frame()
		wantSource(t, s, "# Title\n\n原文🙂")
		mark, ok := n.MarkedRange()
		if !ok || mark.Start != 7 || mark.End != 12 {
			t.Fatalf("mark: %+v, %v", mark, ok)
		}
		if rect, _, ok := n.BoundsForRange(mark); !ok || rect.H <= 0 || rect.W <= 0 {
			t.Fatalf("candidate bounds: %+v %v", rect, ok)
		}
		if unmark {
			n.UnmarkText()
			wantSource(t, s, "# Title\n\nnihao")
			n.ReplaceText(&mark, "你好🙂")
		} else {
			n.ReplaceText(nil, "你好🙂")
		}
		tt.Frame()
		wantSource(t, s, "# Title\n\n你好🙂")
		tt.Command("undo")
		wantSource(t, s, "# Title\n\n原文🙂")
		tt.Command("redo")
		wantSource(t, s, "# Title\n\n你好🙂")
	}
}

func TestTypesettingUnchangedCompositionDoesNotUndoEarlierWriting(t *testing.T) {
	s, tt := typesetTester("# Title\n\ntext")
	selectShown(s, tt, 0, 0)
	tt.Type("New ")
	selectShown(s, tt, 11, 15)
	n := &s.input
	n.SetMarkedText(nil, "text", ui.TextInputRange{Start: 4, End: 4})
	r, _ := n.MarkedRange()
	n.UnmarkText()
	n.ReplaceText(&r, "修改")
	wantSource(t, s, "# New Title\n\n修改")
	tt.Command("undo")
	wantSource(t, s, "# New Title\n\ntext")
	tt.Command("undo")
	wantSource(t, s, "# Title\n\ntext")
}

func TestTypesettingDirectNativeCancellationAndUTF16Replacement(t *testing.T) {
	s, tt := typesetTester("# 你🙂好\n\nbody")
	selectShown(s, tt, 1, 2)
	n := &s.input
	n.SetMarkedText(nil, "ni", ui.TextInputRange{Start: 2, End: 2})
	n.SetMarkedText(nil, "", ui.TextInputRange{})
	tt.Frame()
	wantSource(t, s, "# 你🙂好\n\nbody")
	if a, z := s.Document.Selection(); a != 1 || z != 2 {
		t.Fatalf("cancel lost selection %d,%d", a, z)
	}
	r := ui.TextInputRange{Start: 1, End: 3}
	n.ReplaceText(&r, "好🙂")
	tt.Frame()
	wantSource(t, s, "# 你好🙂好\n\nbody")
	tt.Command("undo")
	wantSource(t, s, "# 你🙂好\n\nbody")
}

func TestTypesettingTypingBurstsDoNotAbsorbNavigationOrIME(t *testing.T) {
	s, tt := typesetTester("# Title")
	selectShown(s, tt, 5, 5)
	for _, ch := range " abc" {
		s.input.ReplaceText(nil, string(ch))
	}
	wantSource(t, s, "# Title abc")
	tt.Key(0, ui.KeyLeft)
	tt.Key(0, ui.KeyRight)
	s.input.ReplaceText(nil, "d")
	s.input.SetMarkedText(nil, "ni", ui.TextInputRange{Start: 2, End: 2})
	s.input.ReplaceText(nil, "你")
	tt.Command("undo")
	wantSource(t, s, "# Title abcd")
	tt.Command("undo")
	wantSource(t, s, "# Title abc")
	tt.Command("undo")
	wantSource(t, s, "# Title")
}

func TestTypesettingUnchangedReplacementDoesNotJoinEarlierHistory(t *testing.T) {
	s, tt := typesetTester("# Title")
	selectShown(s, tt, 5, 5)
	tt.Type("!")
	selectShown(s, tt, 5, 6)
	s.input.ReplaceText(nil, "!")
	s.input.ReplaceText(nil, "a")
	wantSource(t, s, "# Title!a")
	tt.Command("undo")
	wantSource(t, s, "# Title!")
	tt.Command("undo")
	wantSource(t, s, "# Title")
}

func TestTypesettingNativeReplacementAwayFromCaretKeepsOtherHeadings(t *testing.T) {
	s, tt := typesetTester("# One\n\n## Two\n\nbody")
	selectShown(s, tt, 0, 0)
	r := ui.TextInputRange{Start: 10, End: 14}
	s.input.ReplaceText(&r, "changed")
	tt.Frame()
	wantSource(t, s, "# One\n\n## Two\n\nchanged")
	tt.Command("undo")
	wantSource(t, s, "# One\n\n## Two\n\nbody")
	selectShown(s, tt, 0, 0)
	s.input.SetMarkedText(&r, "gai", ui.TextInputRange{Start: 3, End: 3})
	s.input.ReplaceText(nil, "改变")
	wantSource(t, s, "# One\n\n## Two\n\n改变")
}

func TestTypesettingPointerSelectionAcrossHeadingAndParagraph(t *testing.T) {
	s, tt := typesetTester("# Heading\n\nParagraph\n\n## Next")
	boxes := s.Document.VisibleParagraphs()
	if len(boxes) < 3 {
		t.Fatal("missing paragraph layout")
	}
	first, last := boxes[0], boxes[2]
	tt.Press(2, first.Top+first.Height/2)
	tt.Move(54, last.Top+last.Height/2)
	tt.Release(54, last.Top+last.Height/2)
	a, z := s.Document.Selection()
	if min(a, z) != 0 || max(a, z) <= 9 {
		t.Fatalf("drag did not cross blocks: %d,%d", a, z)
	}
	selected := s.Document.Text().Slice(min(a, z), max(a, z)).String()
	if !strings.Contains(selected, "Heading\n\n") {
		t.Fatalf("selected %q", selected)
	}
	tt.Type("Replacement")
	if strings.Contains(s.Source.Buffer.String(), "# Heading") {
		t.Fatal("drag selection did not edit source")
	}
	tt.Command("undo")
	wantSource(t, s, "# Heading\n\nParagraph\n\n## Next")
}

func TestTypesettingLongDocumentKeepsCaretInView(t *testing.T) {
	source := "# Long read\n\n" + strings.Repeat("## A paragraph\n\nWriting, 中文 and emoji 🙂. A little room to think.\n\n", 1900)
	if utf8.RuneCountInString(source) < 100000 {
		t.Fatal("fixture is too short")
	}
	s, tt := typesetTester(source)
	end := s.Document.Text().Len()
	selectShown(s, tt, end, end)
	before, ok := s.Document.CaretRect()
	if !ok {
		t.Fatal("caret not revealed")
	}
	tt.Type("One more thought.")
	after, ok := s.Document.CaretRect()
	if !ok || after.Y < 0 || after.Y+after.H > 560 || math.Abs(float64(after.Y-before.Y)) > 40 {
		t.Fatalf("caret jumped: %+v -> %+v", before, after)
	}
	wantSource(t, s, source+"One more thought.")
	tt.Command("undo")
	wantSource(t, s, source)
}

func BenchmarkTypesettingHeadlessInput100K(b *testing.B) {
	source := "# Long read\n\n" + strings.Repeat("## A paragraph\n\nWriting, 中文 and emoji 🙂. A little room to think.\n\n", 1900)
	s, tt := typesetTester(source)
	selectShown(s, tt, s.Document.Text().Len(), s.Document.Text().Len())
	b.ReportAllocs()
	b.ReportMetric(float64(utf8.RuneCountInString(source)), "chars")
	for b.Loop() {
		tt.Type("a")
	}
}
