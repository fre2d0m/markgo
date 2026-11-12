package mdeditor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/editor"
)

func TestLivePreview(t *testing.T) {
	st := &editor.State{Buffer: editor.NewBuffer("a **b** c\n# Title\n- [ ] task\n- item\n")}
	st.Select(4, 4)
	replaced := func() []string {
		var source []string
		for _, replacement := range LivePreview(st) {
			source = append(source, st.Buffer.Slice(replacement.Start, replacement.End))
		}
		return source
	}
	got := replaced()
	want := []string{"# ", "- [ ]", "-"}
	if len(got) != len(want) {
		t.Fatalf("replaced %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("replaced %q, want %q", got, want)
		}
	}
	st.Select(st.Buffer.Len(), st.Buffer.Len())
	if got := replaced(); len(got) != 5 || got[0] != "**" || got[1] != "**" {
		t.Fatalf("replaced %q", got)
	}
}

func TestLivePreviewRevealsOnlyTheElementBeingEdited(t *testing.T) {
	source := "Before **bold** and *italic* after"
	st := &editor.State{Buffer: editor.NewBuffer(source)}
	st.Select(0, 0)
	if got := LivePreview(st); len(got) != 4 {
		t.Fatalf("expected both formatted spans to hide their delimiters, got %#v", got)
	}
	insideBold := strings.Index(source, "bold") + 1
	st.Select(insideBold, insideBold)
	got := LivePreview(st)
	if len(got) != 2 || st.Buffer.Slice(got[0].Start, got[0].End) != "*" || st.Buffer.Slice(got[1].Start, got[1].End) != "*" {
		t.Fatalf("only italic delimiters should remain hidden, got %#v", got)
	}
	if st.Buffer.String() != source {
		t.Fatal("preview changed the underlying Markdown")
	}
}

func TestAnalysisProtectsFrontMatterEscapesAndCodeBlocks(t *testing.T) {
	source := "---\ntitle: **raw**\n---\n\n# Visible\n\n\\*escaped\\* and **real**\n\n```md\n# not a heading\n**not bold**\n```\n"
	analysis := Analyze(source)
	if len(analysis.Headings) != 1 || analysis.Headings[0].Title != "Visible" {
		t.Fatalf("headings: %#v", analysis.Headings)
	}
	st := &editor.State{Buffer: editor.NewBuffer(source)}
	st.Select(len(source), len(source))
	for _, replacement := range analysis.Replacements(st) {
		text := source[replacement.Start:replacement.End]
		if replacement.Start < strings.Index(source, "# Visible") || replacement.Start >= strings.Index(source, "```md") {
			t.Fatalf("unexpected replacement %q at %d", text, replacement.Start)
		}
	}
}

func TestTaskMarkerRecognizesUppercaseCheckedState(t *testing.T) {
	source := "- [X] task\n"
	st := &editor.State{Buffer: editor.NewBuffer(source)}
	st.Select(len(source), len(source))
	got := LivePreview(st)
	if len(got) != 1 || !got[0].Interactive || got[0].Label != "Done" {
		t.Fatalf("checked task: %#v", got)
	}
}
