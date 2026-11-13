package mdeditor

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func blockClick(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
}

func TestTypesettingCodeHeaderActionsKeepWritingPlace(t *testing.T) {
	source := "```go title=sample\n中文🙂\n```\n\nafter"
	s, tt := typesetTester(source)
	selectShown(s, tt, 2, 1)
	before, _ := s.Document.CaretRect()
	copy, _ := tt.Find("Copy code")
	next, _ := tt.Find("Continue after code")
	language, _ := tt.Find("Code language: go")
	if copy.W < 40 || copy.H < 40 || next.W < 40 || next.H < 40 || language.W < 40 || language.H < 40 || language.X+language.W > copy.X || copy.X+copy.W > next.X {
		t.Fatalf("code targets overlap or are too small: %+v %+v %+v", language, copy, next)
	}
	tt.SetClipboard("unchanged")
	tt.Press(copy.X+20, copy.Y+20)
	tt.Move(700, 500)
	tt.Release(700, 500)
	if tt.Clipboard() != "unchanged" {
		t.Fatal("cancelled copy activated")
	}
	blockClick(t, tt, "Copy code")
	if tt.Clipboard() != "中文🙂" {
		t.Fatalf("header copy included fences: %q", tt.Clipboard())
	}
	if a, z := s.Document.Selection(); a != 2 || z != 1 {
		t.Fatalf("copy moved reversed selection: %d,%d", a, z)
	}
	after, _ := s.Document.CaretRect()
	if math.Abs(float64(before.Y-after.Y)) > 1 {
		t.Fatal("copy moved the page")
	}
	if _, ok := tt.Find("Code copied"); !ok {
		t.Fatal("missing copy feedback")
	}
	tt.Advance(1600 * time.Millisecond)
	if _, ok := tt.Find("Copy code"); !ok {
		t.Fatal("copy feedback did not expire")
	}
	blockClick(t, tt, "Code language: go")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("python")
	tt.Key(0, ui.KeyEnter)
	wantSource(t, s, strings.Replace(source, "go title", "python title", 1))
	if a, z := s.Document.Selection(); a != 2 || z != 1 {
		t.Fatalf("language moved selection: %d,%d", a, z)
	}
	tt.Type("字")
	wantSource(t, s, strings.Replace(strings.Replace(source, "go title", "python title", 1), "中文", "中字", 1))
	tt.Command("undo")
	tt.Command("undo")
	wantSource(t, s, source)
	blockClick(t, tt, "Code language: go")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("invalid language")
	blockClick(t, tt, "Apply code language")
	wantSource(t, s, source)
	if !s.blockUI.languageOpen || s.blockUI.languageError == "" {
		t.Fatal("invalid language closed the input without feedback")
	}
	tt.Key(0, ui.KeyEscape)
	if s.blockUI.languageOpen {
		t.Fatal("Escape left language popup open")
	}
	if !tt.Focused("Typeset document") {
		t.Fatal("Escape did not restore the writing focus")
	}
	tt.Type("字")
	wantSource(t, s, strings.Replace(source, "中文", "中字", 1))
	tt.Command("undo")
	blockClick(t, tt, "Continue after code")
	tt.Type("new paragraph")
	wantSource(t, s, "```go title=sample\n中文🙂\n```\nnew paragraph\nafter")
	tt.Command("undo")
	wantSource(t, s, source)
}

func TestTypesettingBlocksNativeGeometryAndComposition(t *testing.T) {
	for _, size := range []float32{14, 18, 24} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			source := "> > 引用中文🙂\n\n```go\n代码中文🙂\n```\n\nafter"
			s := NewTypesetter(NewFile("blocks.md", source).State)
			tt := ui.NewTester(func(c *ui.Context) { TypesettingView(c, s, ui.Font{Family: "serif", Size: size}).Fill().AutoFocus() }, 400, 560)
			tt.SetScale(map[float32]float32{14: 1, 18: 1.5, 24: 2}[size])
			tt.SetPreferences(ui.Preferences{ReduceMotion: size == 24})
			for _, box := range s.Document.VisibleParagraphs() {
				line, _ := s.projection.shownLine(box.Index)
				inset := float32(0)
				if line.block.quote > 0 {
					inset = float32(line.block.quote) * s.input.gutter
				} else if line.block.code > 0 {
					inset = 16
				} else {
					continue
				}
				at, ok := s.input.IndexForPoint(ui.Point{X: inset + 0.1, Y: box.Top + box.Height/2})
				if want := nativeOffset(s.Document.Text().String(), box.Start); !ok || at != want {
					t.Fatalf("block native hit: %d/%v, want %d", at, ok, want)
				}
			}
			at := shownAt(s, "代码")
			selectShown(s, tt, at+5, at+2)
			selection := s.input.Selection()
			if !selection.Reversed || selection.Range.End-selection.Range.Start != 4 {
				t.Fatalf("native selection lost surrogate or direction: %+v", selection)
			}
			s.input.SetMarkedText(nil, "nihao", ui.TextInputRange{Start: 5, End: 5})
			tt.Frame()
			wantSource(t, s, source)
			s.input.ReplaceText(nil, "你好🙂")
			tt.Frame()
			wantSource(t, s, strings.Replace(source, "代码中文", "代码你好", 1))
			tt.Command("undo")
			wantSource(t, s, source)
		})
	}
}

func TestTypesettingCodeLongReadRetainsPlace(t *testing.T) {
	source := strings.Repeat("> 一点想法 and a quiet sentence.\n\n```go\nfmt.Println(\"中文🙂\")\n```\n\n", 1800)
	s, tt := typesetTester(source)
	selectShown(s, tt, s.Document.Text().Len()/2, s.Document.Text().Len()/2)
	before, _ := s.Document.CaretRect()
	tt.Type("中🙂")
	tt.Command("undo")
	wantSource(t, s, source)
	after, ok := s.Document.CaretRect()
	if !ok || math.Abs(float64(before.Y-after.Y)) > 1 {
		t.Fatalf("block undo moved page: %+v -> %+v", before, after)
	}
}

func TestTypesettingBlocksWrappedNativeGeometry(t *testing.T) {
	for _, source := range []string{
		"> > " + strings.Repeat("中文🙂 has room to continue. ", 8),
		"```go\n" + strings.Repeat("中文🙂 has room to continue. ", 8) + "\n```",
	} {
		s := NewTypesetter(NewFile("wrapped.md", source).State)
		tt := ui.NewTester(func(c *ui.Context) { TypesettingView(c, s, ui.Font{Family: "serif", Size: 24}).Fill().AutoFocus() }, 400, 560)
		for _, at := range []int{0, 4, 26, 60, 90, 120, 180} {
			selectShown(s, tt, at, at)
			r, ok := s.Document.CaretRect()
			if !ok {
				t.Fatal("wrapped caret missing")
			}
			index, hit := s.input.IndexForPoint(ui.Point{X: r.X + 0.1, Y: r.Y + r.H/2})
			if want := nativeOffset(s.Document.Text().String(), at); !hit || index != want {
				t.Fatalf("wrapped hit at %d: %d/%v, want %d, caret %+v", at, index, hit, want, r)
			}
		}
	}
}
