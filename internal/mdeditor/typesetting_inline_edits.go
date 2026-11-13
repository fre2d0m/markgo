package mdeditor

import (
	"slices"
	"strings"

	"github.com/egoist/mygo/ui/editor"
)

func (p *typesetProjection) inlineEditPairs(a, z int, text string) []editor.Edit {
	var extra []editor.Edit
	add := func(start, end int, replacement string) {
		if end <= start {
			return
		}
		// Only surviving syntax is changed; the primary selection owns its part.
		if start < a {
			extra = append(extra, editor.Edit{Start: start, End: min(a, end), Text: replacement})
		}
		if end > z {
			extra = append(extra, editor.Edit{Start: max(z, start), End: end, Text: replacement})
		}
	}
	for _, m := range p.inlines {
		openHit, closeHit := a < m.content && z > m.start, a < m.end && z > m.contentEnd
		empty := text == "" && p.viewOffset(a) <= p.viewOffset(m.content) && p.viewOffset(z) >= p.viewOffset(m.contentEnd)
		if z > a && (openHit != closeHit || empty) {
			add(m.start, m.content, "")
			add(m.contentEnd, m.end, "")
			continue
		}
		if m.kind != InlineCode || a < m.content || z > m.contentEnd {
			continue
		}
		body := p.source[m.content:a] + text + p.source[z:m.contentEnd]
		prefix, suffix := inlineDelimiters(InlineCode, body)
		oldPrefix, oldSuffix := p.source[m.start:m.content], p.source[m.contentEnd:m.end]
		length := max(len(strings.TrimSpace(prefix)), len(strings.TrimSpace(oldPrefix)))
		pad := strings.HasSuffix(prefix, " ") || strings.HasSuffix(oldPrefix, " ") && strings.Trim(body, " ") != ""
		prefix, suffix = strings.Repeat("`", length), strings.Repeat("`", length)
		if pad {
			prefix += " "
			suffix = " " + suffix
		}
		// Preserve a valid original delimiter and optional padding. Extend
		// the run for literal backticks, and shield significant edge spaces
		// from CommonMark's one-space normalization.
		if prefix != oldPrefix || suffix != oldSuffix {
			add(m.start, m.content, prefix)
			add(m.contentEnd, m.end, suffix)
		}
	}
	slices.SortFunc(extra, func(a, b editor.Edit) int { return a.Start - b.Start })
	// Nested formats can share a boundary. Merge only overlapping removals.
	out := extra[:0]
	for _, e := range extra {
		if n := len(out); n > 0 && e.Start <= out[n-1].End && e.Text == "" && out[n-1].Text == "" {
			out[n-1].End = max(out[n-1].End, e.End)
		} else {
			out = append(out, e)
		}
	}
	return out
}

func inlineDelimiters(kind InlineKind, body string) (string, string) {
	switch kind {
	case InlineBold:
		return "**", "**"
	case InlineItalic:
		return "_", "_"
	case InlineStrike:
		return "~~", "~~"
	case InlineCode:
		longest, run := 0, 0
		for _, c := range body {
			if c == '`' {
				run++
				longest = max(longest, run)
			} else {
				run = 0
			}
		}
		mark := strings.Repeat("`", longest+1)
		if strings.Trim(body, " ") != "" && (strings.HasPrefix(body, "`") || strings.HasSuffix(body, "`") || strings.HasPrefix(body, " ") && strings.HasSuffix(body, " ")) {
			return mark + " ", " " + mark
		}
		return mark, mark
	}
	return "", ""
}
