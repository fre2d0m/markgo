package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestTypesettingInlineMenuSharesSourceHistory(t *testing.T) {
	for _, sourceMode := range []bool{false, true} {
		for _, tc := range []struct{ label, want string }{{"Bold", "a **thought**"}, {"Italic", "a _thought_"}, {"Strikethrough", "a ~~thought~~"}, {"Inline code", "a `thought`"}} {
			p := NewTypesettingPreview()
			p.selected = 3
			page := p.current()
			page.file.State.Buffer.Set("a thought")
			page.typesetter.FromSource()
			tt := ui.NewTester(p.View, 960, 640)
			if sourceMode {
				click(t, tt, "Source mode")
				page.file.State.Select(9, 2)
			} else {
				page.typesetter.Document.SetSelection(9, 2)
			}
			tt.Frame()
			click(t, tt, "Paragraph style")
			for _, label := range []string{"Bold", "Italic", "Strikethrough", "Inline code", "Edit link", "Quote", "Code block"} {
				r, ok := tt.Find(label)
				if !ok || r.W < 40 || r.H < 40 || r.X < 0 || r.X+r.W > 960 || r.Y < 0 || r.Y+r.H > 640 {
					t.Fatalf("clipped control %s: %+v", label, r)
				}
			}
			click(t, tt, tc.label)
			if got := page.file.State.Buffer.String(); got != tc.want {
				t.Fatalf("%s from source=%v: %q", tc.label, sourceMode, got)
			}
			if sourceMode {
				click(t, tt, "Typeset mode")
			}
			if a, z := page.typesetter.Document.Selection(); a != 9 || z != 2 {
				t.Fatalf("selection %d:%d", a, z)
			}
			tt.Command("undo")
			if got := page.file.State.Buffer.String(); got != "a thought" {
				t.Fatalf("undo %q", got)
			}
		}
	}
}

func TestTypesettingInlineLinkMenuFocus(t *testing.T) {
	p := NewTypesettingPreview()
	p.selected = 3
	page := p.current()
	page.file.State.Buffer.Set("a thought")
	page.typesetter.FromSource()
	tt := ui.NewTester(p.View, 960, 640)
	page.typesetter.Document.SetSelection(2, 9)
	tt.Frame()
	click(t, tt, "Paragraph style")
	click(t, tt, "Edit link")
	tt.Type("notes.md")
	click(t, tt, "Apply link")
	if got := page.file.State.Buffer.String(); got != "a [thought](<notes.md>)" {
		t.Fatalf("link %q", got)
	}
	tt.Command("undo")
	if got := page.file.State.Buffer.String(); got != "a thought" {
		t.Fatalf("undo %q", got)
	}
}
