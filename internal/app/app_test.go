package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/xddcode/markgo/internal/mdeditor"
)

func TestWorkspace(t *testing.T) {
	a := New()
	tt := ui.NewTester(a.View, 1320, 840)
	if !tt.HasText("MarkGo") || !tt.HasText("Welcome.md") {
		t.Fatalf("initial workspace texts %q", tt.Texts())
	}
	if err := tt.Click("AI Notes.md"); err != nil {
		t.Fatal(err)
	}
	if got := a.current().Name; got != "AI Notes.md" {
		t.Fatalf("selected document %q", got)
	}
	if err := tt.Click("Live preview"); err != nil {
		t.Fatal(err)
	}
	if a.preview {
		t.Fatal("live preview stayed enabled")
	}
	if err := tt.Click("AI assistant"); err != nil {
		t.Fatal(err)
	}
	if a.assistant {
		t.Fatal("AI assistant stayed open")
	}
	if err := tt.Click("Filter documents"); err != nil {
		t.Fatal(err)
	}
	tt.Type("Guide")
	if files := a.visibleFiles(); len(files) != 1 || files[0].Name != "Writing Guide.md" {
		t.Fatalf("filtered files %v", files)
	}
}

func TestSelectedText(t *testing.T) {
	f := mdeditor.NewFile("note.md", "hello **world**")
	f.State.Select(8, 13)
	if got := mdeditor.SelectedText(f.State); got != "world" {
		t.Fatalf("selected text %q", got)
	}
}
