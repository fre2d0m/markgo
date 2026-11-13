package mdeditor

import (
	"math"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

func TestTypesettingLongDocumentRetainsEditPosition(t *testing.T) {
	source := "# Long read\n\n" + strings.Repeat("## A paragraph\n\nWriting, 中文 and emoji 🙂. A little room to think.\n\n", 1900)
	for _, location := range []string{"start", "middle", "end"} {
		t.Run(location, func(t *testing.T) {
			s, tt := typesetTester(source)
			at := s.Document.Text().Len()
			if location == "start" {
				at = 2
			} else if location == "middle" {
				at /= 2
			}
			selectShown(s, tt, at, at)
			before, ok := s.Document.CaretRect()
			if !ok {
				t.Fatal("caret not revealed")
			}
			byteOffset := s.projection.sourceOffset(at)
			tt.Type("中🙂")
			wantSource(t, s, source[:byteOffset]+"中🙂"+source[byteOffset:])
			after, ok := s.Document.CaretRect()
			if !ok || after.Y < 0 || after.Y+after.H > 560 || math.Abs(float64(after.Y-before.Y)) > 40 {
				t.Fatalf("typing moved the page: %+v -> %+v", before, after)
			}
			tt.Key(ui.Cmd, ui.KeyZ)
			wantSource(t, s, source)
			after, ok = s.Document.CaretRect()
			if !ok || math.Abs(float64(after.Y-before.Y)) > 1 {
				t.Fatalf("undo moved the page: %+v -> %+v", before, after)
			}
		})
	}
}

// A fresh full parse is the oracle for the incremental path. In particular,
// edits to a fence/front-matter delimiter must update untouched distant lines.
func checkProjection(t testing.TB, p *typesetProjection, source string) {
	t.Helper()
	want := projectTypesetting(source)
	if p.source != source || p.text != want.text || !slices.Equal(p.lines, want.lines) || !slices.Equal(p.listLines, want.listLines) || !slices.Equal(p.items, want.items) || p.listNumber != want.listNumber || !slices.Equal(p.blockLines, want.blockLines) || !slices.Equal(p.codes, want.codes) || !slices.Equal(p.visible, want.visible) {
		t.Fatalf("incremental projection differs from full parse for %q\n got %q\nwant %q", source, p.text, want.text)
	}
	for i := 0; i <= want.len(); i++ {
		if got, expected := p.sourceOffset(i), want.sourceOffset(i); got != expected {
			t.Fatalf("source offset at rune %d: %d, want %d", i, got, expected)
		}
	}
	for i := range source {
		if got, expected := p.viewOffset(i), want.viewOffset(i); got != expected {
			t.Fatalf("view offset at byte %d: %d, want %d", i, got, expected)
		}
	}
	if p.viewOffset(len(source)) != want.len() {
		t.Fatal("end-of-document mapping lost")
	}
}

func TestTypesettingIncrementalProjectionContextChanges(t *testing.T) {
	fixtures := []string{
		"", "# ", "## 标题🙂\n\n正文\n", "body\n## next\nbody\n",
		"# First\n\n```go\n# Literal\n```\n\n## Last\n",
		"---\n# metadata\n---\n\n# Title\n\n正文\n",
		"# First\n\n<div>\n# Literal\n</div>\n\n# Last",
		"# First\n\n- list\n  # nested\n\n# Last\n",
		"Setext\n======\n\n## Title ###\n\n[ref]: <https://example.test>\n",
		"3. one\n1. two\n\n   7) nested\n   1) next\n\n- [X] 中文🙂\n- [ ] \n",
		"- parent\n  - child\n\n  ```md\n  - literal\n  ```\n\n> - quote",
		"- parent\n\t- child\n\t  continued\n\n999999999. last",
		"> first\n> > 中文🙂\n> lazy\n>\n\n```\n```\n\nafter",
		"  ~~~~ go extra\n  # literal\n\t中🙂\n  ~~~~~~\n\n```python\n",
	}
	for _, source := range fixtures {
		for _, insert := range []string{"\n", "# ", "```\n", "---\n", "<!--\n", "中文🙂", "\t", "\n\n"} {
			for _, at := range []int{0, len(source) / 2, len(source)} {
				for at > 0 && at < len(source) && !utf8.RuneStart(source[at]) {
					at--
				}
				p := projectTypesetting(source)
				next := source[:at] + insert + source[at:]
				p.update(next)
				checkProjection(t, &p, next)
				p.update(source) // undo must repair both the visible text and map
				checkProjection(t, &p, source)
				p.update(next)
				checkProjection(t, &p, next)
			}
		}
	}
}

func FuzzTypesettingIncrementalProjection(f *testing.F) {
	for _, seed := range []string{
		"# 标题\n\n文字🙂\n\n## Tail",
		"---\n# metadata\n---\n\n```go\n# literal\n```\n\n# Tail",
		"- item\n  ## nested\n\n> # quote\n\nSetext\n====\n",
		"3. one\n1. two\n   - [X] child🙂\n   - [ ] \n\n999999999) last",
		"> first\n> > 中文🙂\n> lazy\n>\n\n```\n```\n\nafter",
		"  ~~~~ go extra\n  # literal\n\t中🙂\n  ~~~~~~\n\n```python\n",
	} {
		f.Add(seed, []byte{0, 3, 7, 12, 9, 30, 255, 1, 11})
	}
	f.Fuzz(func(t *testing.T, source string, edits []byte) {
		if len(source) > 4096 || len(edits) > 64 || !utf8.ValidString(source) {
			t.Skip()
		}
		p := projectTypesetting(source)
		snippets := []string{"", "\n", "# ", "```\n", "---\n", "🙂", "中", "文", "\t", "<!--", "-->", "[ref]: /note", "- ", "> ", "[X] ", "[ ] ", "3. ", "1) ", "  "}
		for i, action := range edits {
			runes := []rune(source)
			at := int(action) % (len(runes) + 1)
			end := min(len(runes), at+i%5)
			source = string(runes[:at]) + snippets[int(action)%len(snippets)] + string(runes[end:])
			p.update(source)
			checkProjection(t, &p, source)
		}
	})
}

func TestTypesettingTypingRetainsHeadingStylesAndClearsPreedit(t *testing.T) {
	s, tt := typesetTester("# Title\n\ntext\n\n## Next")
	style := s.projection.styleVersion
	selectShown(s, tt, 2, 2)
	tt.Type("中🙂")
	if s.projection.styleVersion != style {
		t.Fatal("typing in a heading invalidated every paragraph's style")
	}
	s.Heading(2)
	if s.projection.styleVersion == style {
		t.Fatal("changing heading level did not invalidate typography")
	}
	for _, cancel := range []bool{true, false} {
		s, tt = typesetTester("# Title\n\ntext\n\nnext")
		selectShown(s, tt, 7, 11)
		s.input.SetMarkedText(nil, "text", ui.TextInputRange{Start: 4, End: 4})
		if cancel {
			s.CancelComposition()
		} else {
			s.input.UnmarkText()
		}
		wantSource(t, s, "# Title\n\ntext\n\nnext")
		if s.Document.Text().AttributesAt(7).Underline {
			t.Fatal("committed/cancelled preedit left an underline in retained text")
		}
		// The next composition must not make the old preedit reappear.
		selectShown(s, tt, 13, 13)
		s.input.SetMarkedText(nil, "xin", ui.TextInputRange{Start: 3, End: 3})
		if s.Document.Text().AttributesAt(7).Underline {
			t.Fatal("previous preedit reappeared during a new composition")
		}
	}
}

func BenchmarkTypesettingIncrementalProjection100K(b *testing.B) {
	source := strings.Repeat("## A quiet paragraph\n\nSome words, 一些中文 and an emoji 🙂.\n\n", 1900)
	for _, location := range []string{"start", "middle", "end"} {
		b.Run(location, func(b *testing.B) {
			p := projectTypesetting(source)
			at := len(source)
			if location == "start" {
				at = 10
			} else if location == "middle" {
				at = len(source)/2 + 10
			}
			next := source[:at] + "中🙂" + source[at:]
			b.ReportAllocs()
			for b.Loop() {
				p.update(next)
				p.update(source)
			}
		})
	}
}

// A fixed-length edit-and-undo cycle makes before/after runs comparable.
// Unlike continuous typing, a faster implementation cannot grow a longer
// document or a larger undo group just by completing more iterations.
func BenchmarkTypesettingEditCycle100K(b *testing.B) {
	source := "# Long read\n\n" + strings.Repeat("## A paragraph\n\nWriting, 中文 and emoji 🙂. A little room to think.\n\n", 1900)
	for _, location := range []string{"start", "middle", "end"} {
		b.Run(location, func(b *testing.B) {
			s, tt := typesetTester(source)
			at := s.Document.Text().Len()
			if location == "start" {
				at = 2
			} else if location == "middle" {
				at /= 2
			}
			selectShown(s, tt, at, at)
			b.ReportAllocs()
			for b.Loop() {
				tt.Type("中🙂")
				tt.Key(ui.Cmd, ui.KeyZ)
			}
			if s.Source.Buffer.String() != source {
				b.Fatal("benchmark did not restore its fixture")
			}
		})
	}
}
