package mdeditor

import (
	"strings"
	"testing"
)

func TestOutlineUsesMarkdownStructureAndSourceOffsets(t *testing.T) {
	source := "# 中文 **标题**\n\n```md\n# not a heading\n```\n\nA [linked](https://example.org) title\n---\n\n> ### 引用标题\n"
	headings := Outline(source)
	if len(headings) != 3 {
		t.Fatalf("headings: %#v", headings)
	}
	for i, want := range []struct {
		title string
		level int
		start string
	}{
		{"中文 标题", 1, "# 中文"},
		{"A linked title", 2, "A [linked]"},
		{"引用标题", 3, "### 引用"},
	} {
		got := headings[i]
		if got.Title != want.title || got.Level != want.level || !strings.HasPrefix(source[got.Offset:], want.start) {
			t.Fatalf("heading %d: %#v, source %q", i, got, source[got.Offset:])
		}
	}
}
