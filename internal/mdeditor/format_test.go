package mdeditor

import (
	"testing"

	"github.com/egoist/mygo/ui/editor"
)

func TestInlineFormattingIsUndoableAndPreservesUnicode(t *testing.T) {
	st := &editor.State{Buffer: editor.NewBuffer("前面的文字")}
	st.Select(len("前面"), len("前面的"))
	if !ToggleInline(st, "**") || st.Buffer.String() != "前面**的**文字" {
		t.Fatalf("formatted: %q", st.Buffer.String())
	}
	if got := st.SelectedText(); got != "的" {
		t.Fatalf("selection: %q", got)
	}
	if !st.Buffer.Undo() || st.Buffer.String() != "前面的文字" {
		t.Fatalf("undo: %q", st.Buffer.String())
	}
}

func TestInlineToggleAndEmptySelection(t *testing.T) {
	st := &editor.State{Buffer: editor.NewBuffer("**hello**")}
	st.Select(2, 7)
	if !ToggleInline(st, "**") || st.Buffer.String() != "hello" {
		t.Fatalf("toggle off: %q", st.Buffer.String())
	}
	st.Select(5, 5)
	if !ToggleInline(st, "*") || st.Buffer.String() != "hello**" || st.Selection().Start() != 6 {
		t.Fatalf("empty selection: %q, %#v", st.Buffer.String(), st.Selection())
	}
}

func TestHeadingAndLinkFormatting(t *testing.T) {
	st := &editor.State{Buffer: editor.NewBuffer("## Title\n\n正文")}
	st.Select(4, 4)
	if !SetHeading(st, 1) || st.Buffer.String() != "# Title\n\n正文" {
		t.Fatalf("heading: %q", st.Buffer.String())
	}
	start := len("# Title\n\n")
	st.Select(start, st.Buffer.Len())
	if !InsertLink(st) || st.Buffer.String() != "# Title\n\n[正文](url)" || st.SelectedText() != "url" {
		t.Fatalf("link: %q, selection %q", st.Buffer.String(), st.SelectedText())
	}
}
