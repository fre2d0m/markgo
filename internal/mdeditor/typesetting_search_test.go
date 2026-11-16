package mdeditor

import (
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSearchPresentationKeepsTextGeometryAndHistory(t *testing.T) {
	source := "中文🙂 **a thought** and `some code` to keep."
	s, tt := typesetTester(source)
	at := s.Document.Text().Len()
	selectShown(s, tt, at, at)
	before, ok := s.Document.CaretRect()
	if !ok {
		t.Fatal("fixture has no caret")
	}
	version, undo := s.Source.Buffer.Version(), s.Source.Buffer.CanUndo()
	for _, query := range []string{"中文🙂", "thought", "code"} {
		start := strings.Index(source, query)
		s.SetSearchMatch(start, start+len(query))
		tt.Frame()
		after, ok := s.Document.CaretRect()
		if !ok || math.Abs(float64(after.X-before.X)) > 0.5 || after.Y != before.Y || after.H != before.H {
			t.Fatalf("search reflowed %q: %+v -> %+v", query, before, after)
		}
		if s.Source.Buffer.String() != source || s.Source.Buffer.Version() != version || s.Source.Buffer.CanUndo() != undo {
			t.Fatal("highlight edited source or history")
		}
	}
	s.SetSearchMatch(-1, -1)
	tt.Frame()
	tt.Key(0, ui.KeyEnd)
	tt.Type("!")
	tt.Command("undo")
	wantSource(t, s, source)
}
