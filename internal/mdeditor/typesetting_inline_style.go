package mdeditor

import (
	"github.com/egoist/mygo/ui"
	"unicode/utf8"
)

// SetSearchMatch changes presentation only. Keeping the current match visible
// while the search field has focus must not add rich formatting or source undo.
func (t *Typesetter) SetSearchMatch(start, end int) {
	a, z := 0, 0
	if start >= 0 && end > start {
		a, z = t.projection.viewOffset(start), t.projection.viewOffset(end)
	}
	if a != t.searchStart || z != t.searchEnd {
		t.searchStart, t.searchEnd = a, z
		t.searchVersion++
	}
}

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
	if th == nil || !exists || t.searchEnd <= t.searchStart || t.searchEnd <= line.viewStart || t.searchStart >= line.viewStart+line.runes {
		return spans
	}
	var highlighted []ui.Span
	offset := line.viewStart
	for _, s := range spans {
		n := utf8.RuneCountInString(s.Text)
		a, z := max(0, t.searchStart-offset), min(n, t.searchEnd-offset)
		offset += n
		if a >= z {
			highlighted = append(highlighted, s)
			continue
		}
		if s.PaddingX != 0 {
			// An inline-code chip is one padded shape. Splitting it would
			// duplicate padding and move text/caret geometry during search.
			s.Background = th.Selection
			highlighted = append(highlighted, s)
			continue
		}
		left, right := runeByte(s.Text, a), runeByte(s.Text, z)
		if left > 0 {
			part := s
			part.Text = s.Text[:left]
			highlighted = append(highlighted, part)
		}
		part := s
		part.Text, part.Background = s.Text[left:right], th.Selection
		highlighted = append(highlighted, part)
		if right < len(s.Text) {
			part := s
			part.Text = s.Text[right:]
			highlighted = append(highlighted, part)
		}
	}
	return highlighted
}
