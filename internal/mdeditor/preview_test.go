package mdeditor

import (
	"testing"

	"github.com/egoist/mygo/ui/editor"
)

func TestLivePreview(t *testing.T) {
	st := &editor.State{Buffer: editor.NewBuffer("a **b** c\n# Title\n- [ ] task\n- item\n")}
	st.Select(0, 0)
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
