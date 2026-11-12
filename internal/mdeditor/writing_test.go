package mdeditor

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

func TestListContinuation(t *testing.T) {
	for _, test := range []struct{ source, want string }{
		{"- 中文", "- 中文\n- "},
		{"  * nested", "  * nested\n  * "},
		{"9. nine", "9. nine\n10. "},
		{"2) two", "2) two\n3) "},
		{"- [X] done", "- [X] done\n- [ ] "},
		{"- ", ""},
		{"- [ ] ", ""},
		{"> quoted", "> quoted\n> "},
		{"> - item", "> - item\n> - "},
		{"> - ", "> "},
		{"> ", ""},
		{"> > ", "> "},
	} {
		t.Run(test.source, func(t *testing.T) {
			edit, ok := continuation(test.source, len(test.source))
			if !ok {
				t.Fatal("Return was not handled")
			}
			got := test.source[:edit.Start] + edit.Text + test.source[edit.End:]
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
	for _, source := range []string{"paragraph", "    - code", "\t- code", "999999999. last"} {
		if _, ok := continuation(source, len(source)); ok {
			t.Fatalf("unexpected continuation for %q", source)
		}
	}
}

func TestReturnContinuesListsButNotCodeAndComments(t *testing.T) {
	for _, test := range []struct{ source, want string }{
		{"- one", "- one\n- "},
		{"- [x] done", "- [x] done\n- [ ] "},
		{"```md\n- literal", "```md\n- literal\n"},
		{"~~~\n- literal", "~~~\n- literal\n"},
		{"<!--\n- comment", "<!--\n- comment\n"},
	} {
		t.Run(test.source, func(t *testing.T) {
			file := NewFile("writing.md", test.source)
			tt := ui.NewTester(func(c *ui.Context) { editor.View(c, file.State).Fill().AutoFocus() }, 640, 480)
			file.State.Select(file.State.Buffer.Len(), file.State.Buffer.Len())
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			if got := file.State.Buffer.String(); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
			tt.Key(ui.Cmd, ui.KeyZ)
			if got := file.State.Buffer.String(); got != test.source {
				t.Fatalf("Return undo: %q", got)
			}
		})
	}
}
