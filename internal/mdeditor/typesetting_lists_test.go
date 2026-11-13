package mdeditor

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

func TestTypesettingListReturn(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		at                 int // -1 means end
	}{
		{"bullet", "- alpha", "- alpha\n- ", -1},
		{"split", "- abcd", "- ab\n- cd", 2},
		{"start", "- alpha", "- \n- alpha", 0},
		{"empty", "- ", "", 0},
		{"checked task", "- [X] done", "- [X] done\n- [ ] ", -1},
		{"empty task", "- [ ] ", "", 0},
		{"marker", "+ one", "+ one\n+ ", -1},
		{"number", "7) first", "7) first\n8) ", -1},
		{"semantic number", "3. one\n1. two", "3. one\n1. two\n5. ", -1},
		{"nested", "- parent\n  - child", "- parent\n  - child\n  - ", -1},
		{"split first child", "- parent\n  - child", "- parent\n\n  - \n  - child", 7},
		{"nested empty", "- parent\n  - child\n  - ", "- parent\n  - child\n- ", -1},
		{"continuation", "- first\n  second", "- first\n  second\n- ", -1},
		{"code first", "-     code", "-     code\n", -1},
		{"Unicode", "- 中文🙂", "- 中\n- 文🙂", 1},
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
			tt.Key(ui.Cmd, ui.KeyZ)
			wantSource(t, s, tc.source)
			tt.Key(ui.Cmd|ui.Shift, ui.KeyZ)
			wantSource(t, s, tc.want)
		})
	}
}

func TestTypesettingListClipboardAndNativeCommands(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(fmt.Sprint(native), func(t *testing.T) {
			source := "# Notes\n\n3) first\n1) second\n\n- [X] 中文🙂\n- [ ] next\n"
			s, tt := typesetTester(source)
			start := s.projection.viewOffset(strings.Index(source, "3)"))
			end := s.projection.viewOffset(strings.Index(source, "second") + len("second"))
			selectShown(s, tt, end, start)
			command := func(name string, key ui.Key) {
				if native {
					tt.Command(name)
				} else {
					tt.Key(ui.Cmd, key)
				}
			}
			command("copy", ui.KeyC)
			if got := tt.Clipboard(); got != "3) first\n1) second" {
				t.Fatalf("copy lost list syntax: %q", got)
			}
			wantSource(t, s, source)
			command("cut", ui.KeyX)
			wantSource(t, s, strings.Replace(source, "3) first\n1) second", "", 1))
			tt.Command("undo")
			wantSource(t, s, source)
			selectShown(s, tt, start+1, start+4)
			command("copy", ui.KeyC)
			if tt.Clipboard() != "irs" {
				t.Fatalf("partial word copied markup: %q", tt.Clipboard())
			}
			selectShown(s, tt, start, end)
			tt.Command("tasks")
			wantSource(t, s, strings.Replace(source, "3) first\n1) second", "- [ ] first\n- [ ] second", 1))
			tt.Command("undo")
			wantSource(t, s, source)
		})
	}
}

func TestTypesettingListHitTargetsAndTextGeometry(t *testing.T) {
	for _, size := range []float32{14, 18, 24} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			source := "999999999. A long decimal marker\n\n- [ ] first\n- [ ] 中文🙂\n\nwrite here"
			s := NewTypesetter(NewFile("list.md", source).State)
			tt := ui.NewTester(func(c *ui.Context) {
				TypesettingView(c, s, ui.Font{Family: "system-ui", Size: size}).Fill().AutoFocus()
			}, 400, 560)
			scale := map[float32]float32{14: 1, 18: 1.5, 24: 2}[size]
			tt.SetScale(scale)
			tt.SetPreferences(ui.Preferences{ReduceMotion: size == 24})
			first, _ := tt.Find("Toggle task: first")
			second, _ := tt.Find("Toggle task: 中文🙂")
			if first.W < 40 || first.H < 40 || first.Y+first.H > second.Y {
				t.Fatalf("overlapping task targets: %+v %+v", first, second)
			}
			if s.input.gutter <= 40 {
				t.Fatal("long decimal marker did not reserve enough room")
			}
			selectShown(s, tt, s.Document.Text().Len(), s.Document.Text().Len())
			tt.Press(first.X+20, first.Y+20)
			tt.Move(390, 550)
			tt.Release(390, 550)
			wantSource(t, s, source) // dragging off cancels the toggle
			if err := tt.Click("Toggle task: 中文🙂"); err != nil {
				t.Fatal(err)
			}
			wantSource(t, s, strings.Replace(source, "[ ] 中文", "[x] 中文", 1))
			for _, box := range s.Document.VisibleParagraphs() {
				if box.Index != 3 {
					continue
				}
				at, ok := s.input.IndexForPoint(ui.Point{X: s.input.gutter + 0.1, Y: box.Top + box.Height/2})
				want := nativeOffset(s.Document.Text().String(), box.Start)
				if !ok || at != want {
					t.Fatalf("native task text hit: %d/%v, want %d", at, ok, want)
				}
			}
		})
	}
}

func TestTypesettingListPointerDragBackspaceAndLongRead(t *testing.T) {
	source := "- first\n- second\n  continued\n- third"
	s, tt := typesetTester(source)
	boxes := s.Document.VisibleParagraphs()
	tt.Press(listGutter+1, boxes[0].Top+boxes[0].Height/2)
	tt.Move(listGutter+45, boxes[1].Top+boxes[1].Height/2)
	tt.Release(listGutter+45, boxes[1].Top+boxes[1].Height/2)
	a, z := s.Document.Selection()
	if min(a, z) != 0 || max(a, z) <= 6 {
		t.Fatalf("list drag did not cross items: %d,%d", a, z)
	}
	tt.Type("new")
	tt.Command("undo")
	wantSource(t, s, source)
	at := s.projection.viewOffset(strings.Index(source, "continued"))
	selectShown(s, tt, at, at)
	tt.Key(0, ui.KeyBackspace)
	wantSource(t, s, "- first\n- secondcontinued\n- third")
	tt.Command("undo")
	at = s.projection.viewOffset(strings.Index(source, "second"))
	selectShown(s, tt, at, at)
	tt.Key(0, ui.KeyBackspace)
	wantSource(t, s, "- first\n\nsecond\ncontinued\n- third")
	tt.Command("undo")
	wantSource(t, s, source)

	source = strings.Repeat("- [ ] 一点想法 and a quiet sentence.\n", 2500)
	s, tt = typesetTester(source)
	at = s.Document.Text().Len() / 2
	selectShown(s, tt, at, at)
	before, _ := s.Document.CaretRect()
	tt.Type("中🙂")
	tt.Command("undo")
	wantSource(t, s, source)
	after, ok := s.Document.CaretRect()
	if !ok || math.Abs(float64(before.Y-after.Y)) > 1 {
		t.Fatalf("list undo moved page: %+v -> %+v", before, after)
	}
}

func TestTypesettingListIndentSubtreeAndSelection(t *testing.T) {
	for _, tc := range []struct {
		name, source, selected, want string
		out                          bool
	}{
		{"indent", "- one\n- two", "two", "- one\n  - two", false},
		{"wide parent", "10. one\n11. two", "two", "10. one\n\n    11. two", false},
		{"empty child", "- parent\n- ", "", "- parent\n\n  - ", false},
		{"task parent", "- [ ] one\n- [ ] two", "two", "- [ ] one\n  - [ ] two", false},
		{"tab indent", "- parent\n\t- one\n\t- two", "two", "- parent\n\t- one\n      - two", false},
		{"tab outdent", "- parent\n  - one\n\t- two", "two", "- parent\n  - one\n  - two", true},
		{"subtree", "- one\n- two\n  - child\n    continued", "two", "- one\n  - two\n    - child\n      continued", false},
		{"outdent", "- one\n  - two\n    - child", "two", "- one\n- two\n  - child", true},
		{"paragraph", "- one\n- two", "two", "- one\n\ntwo", true},
		{"unlist parent", "- one\n\n  7. child", "one", "one\n\n7. child", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, tt := typesetTester(tc.source)
			sourceAt := strings.Index(tc.source, tc.selected)
			if tc.selected == "" {
				sourceAt = len(tc.source)
			}
			at := s.projection.viewOffset(sourceAt)
			selectShown(s, tt, at, at)
			mods := ui.Modifiers(0)
			if tc.out {
				mods = ui.Shift
			}
			tt.Key(mods, ui.KeyTab)
			wantSource(t, s, tc.want)
			if !tc.out && len(s.projection.items) < 2 {
				t.Fatal("indent no longer parses as a list")
			}
			tt.Key(ui.Cmd, ui.KeyZ)
			wantSource(t, s, tc.source)
			tt.Key(ui.Cmd|ui.Shift, ui.KeyZ)
			wantSource(t, s, tc.want)
		})
	}
	s, tt := typesetTester("- one\n- two\n- three")
	selectShown(s, tt, s.Document.Text().Len(), 4)
	tt.Key(0, ui.KeyTab)
	wantSource(t, s, "- one\n  - two\n  - three")
	a, z := s.Document.Selection()
	if a <= z {
		t.Fatal("indent lost reversed selection")
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	wantSource(t, s, "- one\n- two\n- three")
}

func TestTypesettingListFormattingAndGuards(t *testing.T) {
	for _, tc := range []struct {
		source, want string
		kind         ListKind
	}{
		{"one\ntwo", "- one\n- two", BulletedList},
		{"one\ntwo", "1. one\n2. two", NumberedList},
		{"# Heading", "- Heading", BulletedList},
		{"   one", "   - one", BulletedList},
		{"- one\n- two", "- [ ] one\n- [ ] two", TaskList},
		{"- [X] one", "one", TaskList},
		{"- one\n  - child", "1. one\n   1. child", NumberedList},
		{"- one\n  - child\n- two\n  - next", "1. one\n   1. child\n2. two\n   1. next", NumberedList},
		{"- one\n  - child", "one\n\nchild", NoList},
		{"- [ ] one\n  - [X] child", "one\n\nchild", TaskList},
		{"[text](url)", "- [text](url)", BulletedList},
	} {
		t.Run(tc.source, func(t *testing.T) {
			s, tt := typesetTester(tc.source)
			selectShown(s, tt, 0, s.Document.Text().Len())
			s.SetList(tc.kind)
			tt.Frame()
			wantSource(t, s, tc.want)
			tt.Key(ui.Cmd, ui.KeyZ)
			wantSource(t, s, tc.source)
		})
	}
	for _, source := range []string{"```md\n- literal\n```", "---\n- metadata\n---", "> - quote", "    - code", "[ref]: /note", "| a | b |\n| - | - |", "Setext\n======"} {
		s, tt := typesetTester(source)
		selectShown(s, tt, 0, s.Document.Text().Len())
		s.SetList(TaskList)
		tt.Frame()
		wantSource(t, s, source)
		if s.Notice == "" {
			t.Fatalf("unsupported block needs feedback: %q", source)
		}
	}
}

func TestTypesettingListTaskPointerKeyboardAndIME(t *testing.T) {
	source := "# Notes\n\n- [ ] 中文🙂\n- [X] finished\n\nwriting"
	s, tt := typesetTester(source)
	end := s.Document.Text().Len()
	selectShown(s, tt, end, end)
	r, ok := tt.Find("Toggle task: 中文🙂")
	if !ok || r.W < 40 || r.H < 40 {
		t.Fatalf("missing accessible task target: %+v", r)
	}
	if err := tt.Click("Toggle task: 中文🙂"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(120 * time.Millisecond)
	wantSource(t, s, strings.Replace(source, "[ ]", "[x]", 1))
	if a, z := s.Document.Selection(); a != end || z != end {
		t.Fatalf("task click moved writing selection: %d,%d", a, z)
	}
	tt.Type(" continues")
	tt.Key(ui.Cmd, ui.KeyZ)
	wantSource(t, s, strings.Replace(source, "[ ]", "[x]", 1))
	tt.Key(ui.Cmd, ui.KeyZ)
	wantSource(t, s, source)
	at := s.projection.viewOffset(strings.Index(source, "中文"))
	selectShown(s, tt, at, at+3)
	s.input.SetMarkedText(nil, "nihao", ui.TextInputRange{Start: 5, End: 5})
	wantSource(t, s, source)
	s.input.ReplaceText(nil, "你好🙂")
	tt.Frame()
	wantSource(t, s, strings.Replace(source, "中文🙂", "你好🙂", 1))
	tt.Key(ui.Cmd, ui.KeyEnter)
	wantSource(t, s, strings.Replace(strings.Replace(source, "中文🙂", "你好🙂", 1), "[ ]", "[x]", 1))
	tt.Key(ui.Cmd, ui.KeyZ)
	tt.Key(ui.Cmd, ui.KeyZ)
	wantSource(t, s, source)
}

func TestTypesettingListSourceProjectionAndCrossSelection(t *testing.T) {
	source := "7. one\n1. two\n\n   4. nested\n   1. next\n\n- [X] done\n- [ ] 中文🙂\n"
	s, tt := typesetTester(source)
	if got := s.Document.Text().String(); got != "one\ntwo\n\nnested\nnext\n\ndone\n中文🙂\n" {
		t.Fatalf("list projection: %q", got)
	}
	for line, number := range map[int]int{0: 7, 1: 8, 3: 4, 4: 5} {
		if s.projection.lines[line].list.number != number {
			t.Fatalf("line %d number: %+v", line, s.projection.lines[line].list)
		}
	}
	wantSource(t, s, source)
	selectShown(s, tt, 1, 6)
	tt.Type("中🙂")
	wantSource(t, s, "7. o中🙂o\n\n   4. nested\n   1. next\n\n- [X] done\n- [ ] 中文🙂\n")
	tt.Key(ui.Cmd, ui.KeyZ)
	wantSource(t, s, source)
	tt.Command("selectAll")
	tt.Key(0, ui.KeyBackspace)
	wantSource(t, s, "")
	tt.Key(ui.Cmd, ui.KeyZ)
	wantSource(t, s, source)
	if utf8.RuneCountInString(s.Document.Text().String()) != s.projection.len() {
		t.Fatal("source/view map length differs after undo")
	}
}

func TestTypesettingListTypingPasteAndLiteralNumbering(t *testing.T) {
	s, tt := typesetTester("")
	tt.Type("- one")
	if s.Document.Text().String() != "one" {
		t.Fatal("typed Markdown marker was not projected")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Type("two")
	tt.Key(0, ui.KeyTab)
	wantSource(t, s, "- one\n  - two")
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeyEnter)
	wantSource(t, s, "- one\n  - two\n- ")
	tt.Key(0, ui.KeyEnter)
	wantSource(t, s, "- one\n  - two\n\n")
	tt.Type("a paragraph")
	if s.projection.lines[len(s.projection.lines)-1].list.kind != NoList {
		t.Fatal("typing after list exit became a lazy continuation")
	}
	tt.Command("undo")
	tt.SetClipboard("\n- [ ] pasted\n- [X] done")
	tt.Command("paste")
	wantSource(t, s, "- one\n  - two\n\n\n- [ ] pasted\n- [X] done")
	tt.Command("undo")
	wantSource(t, s, "- one\n  - two\n\n")

	p := projectTypesetting("3.     literal code\n1. ordinary item")
	if got := p.lines[1].list.number; got != 4 {
		t.Fatalf("literal item did not count toward numbering: %d", got)
	}
}

func BenchmarkTypesettingListEditCycle100K(b *testing.B) {
	source := strings.Repeat("- [ ] 一点想法 and a quiet sentence.\n", 3500)
	s, tt := typesetTester(source)
	at := s.Document.Text().Len() / 2
	selectShown(s, tt, at, at)
	b.ReportAllocs()
	for b.Loop() {
		tt.Type("中🙂")
		tt.Command("undo")
	}
	if s.Source.Buffer.String() != source {
		b.Fatal("list benchmark did not restore fixture")
	}
}

func TestTypesettingListCapturePreservesLineEndings(t *testing.T) {
	source := "+ first\n  - [X] 中文🙂\n  - [ ] next\n\n7) start\n1) repeated number\n"
	for _, ending := range []string{"\n", "\r\n"} {
		original := strings.ReplaceAll(source, "\n", ending)
		f := NewFile("lists.md", original)
		s := NewTypesetter(f.State)
		s.ToggleTask(1)
		s.Undo(false)
		data, err := f.CaptureForSave().Bytes()
		if err != nil || string(data) != original {
			t.Fatalf("list round trip changed bytes: %q, %v", data, err)
		}
	}
}
