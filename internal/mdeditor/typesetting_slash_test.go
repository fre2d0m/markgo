package mdeditor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSlashCommandsKeepSourceAndOneUndo(t *testing.T) {
	for _, tc := range []struct{ query, want string }{
		{"paragraph", ""}, {"h1", "# "}, {"h2", "## "}, {"h3", "### "},
		{"bullet", "- "}, {"number", "1. "}, {"task", "- [ ] "},
		{"quote", "> "}, {"code", "```\n\n```"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			before := "# 原文🙂\n\n\n\n[ref]: <https://example.test> \"untouched\""
			s, tt := typesetTester(before)
			s.Source.Select(len("# 原文🙂\n\n"), len("# 原文🙂\n\n"))
			tt.Frame()
			tt.Type("/")
			tt.Type(tc.query)
			if !s.slash.open || len(s.slash.matches) != 1 {
				t.Fatalf("query %q did not filter: %+v", tc.query, s.slash)
			}
			typed := s.Source.Buffer.String()
			tt.Key(0, ui.KeyEnter)
			wantSource(t, s, strings.Replace(before, "\n\n\n\n", "\n\n"+tc.want+"\n\n", 1))
			if s.slash.open || !tt.Focused("Typeset document") {
				t.Fatal("selection did not close menu and retain writing focus")
			}
			tt.Command("undo")
			wantSource(t, s, typed)
			if s.slash.open {
				t.Fatal("undo unexpectedly reopened menu")
			}
			tt.Command("redo")
			wantSource(t, s, strings.Replace(before, "\n\n\n\n", "\n\n"+tc.want+"\n\n", 1))
		})
	}
}

func TestSlashFilterKeyboardAndPointer(t *testing.T) {
	s, tt := typesetTester("")
	tt.Type("/")
	tt.Type("标题")
	if len(s.slash.matches) != 3 {
		t.Fatalf("Chinese query: %+v", s.slash)
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyTab)
	wantSource(t, s, "## ")
	tt.Type("标题")
	wantSource(t, s, "## 标题")
	s, tt = typesetTester("")
	tt.Type("/")
	tt.Type("待办")
	r, ok := tt.Find("Insert Task list")
	if !ok || r.H < 40 {
		t.Fatalf("task target: %+v", r)
	}
	if err := tt.Click("Insert Task list"); err != nil {
		t.Fatal(err)
	}
	tt.Type("Finish this thought")
	wantSource(t, s, "- [ ] Finish this thought")
}

func TestSlashDismissalPreservesLiteralInput(t *testing.T) {
	for _, method := range []string{"escape", "outside", "path", "move", "no-results"} {
		t.Run(method, func(t *testing.T) {
			s, tt := typesetTester("Other\n\n")
			selectShown(s, tt, 7, 7)
			tt.Type("/")
			if !s.slash.open {
				t.Fatal("menu not open")
			}
			switch method {
			case "escape":
				tt.Key(0, ui.KeyEscape)
			case "outside":
				tt.ClickAt(30, 10)
			case "path":
				tt.Type("tmp/")
			case "move":
				tt.Key(0, ui.KeyLeft)
			case "no-results":
				tt.Type("zzzz")
				if !tt.HasText("No matching blocks") {
					t.Fatal("missing empty feedback")
				}
				tt.Key(0, ui.KeyEnter)
			}
			for range 3 {
				tt.Frame()
			}
			if s.slash.open || s.slash.armed {
				t.Fatal("dismissed menu reopened")
			}
			if !strings.Contains(s.Source.Buffer.String(), "/") {
				t.Fatal("dismissal removed literal slash")
			}
		})
	}
}

func TestSlashDoesNotTriggerInSyntaxOrPaste(t *testing.T) {
	for _, source := range []string{"text", "https:", "- ", "# ", "> ", "```\n\n```", "---\n\n---", "    ", "/h2"} {
		t.Run(source, func(t *testing.T) {
			s, tt := typesetTester(source)
			at := s.Document.Text().Len()
			if strings.Contains(source, "\n") {
				at = strings.Index(s.Document.Text().String(), "\n") + 1
			}
			selectShown(s, tt, at, at)
			tt.Type("/")
			if s.slash.open {
				t.Fatalf("menu in syntax %q", s.Source.Buffer.String())
			}
		})
	}
	s, tt := typesetTester("")
	s.Insert("/h2")
	tt.Frame()
	if s.slash.open {
		t.Fatal("paste opened command menu")
	}
}

func TestSlashCompositionAndNativeInput(t *testing.T) {
	s, tt := typesetTester("")
	s.input.ReplaceText(nil, "/")
	tt.Frame()
	if !s.slash.open {
		t.Fatal("native slash did not open menu")
	}
	s.input.SetMarkedText(nil, "biaoti", ui.TextInputRange{Start: 6, End: 6})
	tt.Frame()
	if _, ok := tt.Find("Insert block menu"); ok {
		t.Fatal("menu must yield to input method")
	}
	wantSource(t, s, "/")
	s.input.ReplaceText(nil, "标题")
	tt.Frame()
	if len(s.slash.matches) != 3 {
		t.Fatalf("committed Chinese query %+v", s.slash)
	}
	tt.Key(0, ui.KeyEnter)
	wantSource(t, s, "# ")
	tt.Command("undo")
	wantSource(t, s, "/标题")
	s, tt = typesetTester("")
	s.input.SetMarkedText(nil, "/", ui.TextInputRange{Start: 1, End: 1})
	s.input.ReplaceText(nil, "/")
	tt.Frame()
	if s.slash.open {
		t.Fatal("composition commit armed slash")
	}
}
