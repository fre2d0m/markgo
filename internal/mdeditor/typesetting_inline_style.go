package mdeditor

import "github.com/egoist/mygo/ui"

// One typography path for the editor and the OS text-service hit tester.
func (t *Typesetter) styleSpans(p int, font ui.Font, spans []ui.Span, th *ui.Theme) []ui.Span {
	size, weight := font.Size, 400
	line, exists := t.projection.shownLine(p)
	if exists && line.heading > 0 {
		size *= [...]float32{1, 2, 1.45, 1.2, 1.08, 1, 1}[line.heading]
		weight = 600
	}
	for i := range spans {
		s := &spans[i]
		code := s.Font == "monospace"
		s.Font, s.Size, s.Weight = font.Family, size, max(weight, s.Weight)
		if code || (exists && line.block.code > 0) {
			s.Font, s.Size = "monospace", font.Size*0.9
		}
		if code {
			s.Radius, s.PaddingX = 4, 3
		}
		if th != nil {
			s.Color = th.Text
			if line.list.checked {
				s.Color = th.TextMuted
			}
			if code {
				s.Background = th.SurfaceHover.Alpha(0.65)
			}
			if s.Underline {
				s.DecorationColor, s.UnderlineOffset = th.TextMuted.Alpha(0.6), 3
			}
		}
	}
	return spans
}
