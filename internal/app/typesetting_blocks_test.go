package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestTypesettingBlocksMenuSharesSourceHistory(t *testing.T) {
	for _, sourceMode := range []bool{false, true} {
		for _, action := range []string{"Quote", "Code block"} {
			p := NewTypesettingPreview()
			p.selected = 3
			page := p.current()
			page.file.State.Buffer.Set("a thought\nanother thought")
			page.typesetter.FromSource()
			tt := ui.NewTester(p.View, 960, 640)
			if sourceMode {
				click(t, tt, "Source mode")
				page.file.State.Select(page.file.State.Buffer.Len(), 0)
			} else {
				page.typesetter.Document.SetSelection(page.typesetter.Document.Text().Len(), 0)
			}
			tt.Frame()
			click(t, tt, "Paragraph style")
			for _, label := range []string{"Quote", "Code block", "Task list", "Heading 1"} {
				r, ok := tt.Find(label)
				if !ok || r.H < 40 || r.Y < 0 || r.Y+r.H > 640 {
					t.Fatalf("format action clipped: %s %+v", label, r)
				}
			}
			click(t, tt, action)
			want := "> a thought\n> another thought"
			if action == "Code block" {
				want = "```\na thought\nanother thought\n```"
			}
			if got := page.file.State.Buffer.String(); got != want {
				t.Fatalf("%s from source=%v: %q", action, sourceMode, got)
			}
			if sourceMode {
				click(t, tt, "Typeset mode")
			}
			if a, z := page.typesetter.Document.Selection(); a <= z {
				t.Fatal("formatting lost reversed selection")
			}
			tt.Type("new")
			tt.Command("undo")
			tt.Command("undo")
			if got := page.file.State.Buffer.String(); got != "a thought\nanother thought" {
				t.Fatalf("format menu lost focus or history: %q", got)
			}
		}
	}
}
