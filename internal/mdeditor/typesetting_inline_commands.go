package mdeditor

import (
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui/editor"
)

func (t *Typesetter) pendingInline(at int) uint8 {
	if t.pendingStyleVersion != t.Source.Buffer.Version() || t.pendingStyleAt != at {
		t.pendingStyles = 0
	}
	return t.pendingStyles
}

func (t *Typesetter) prepareInlineTyping(a, z int, v projectedInsertion) projectedInsertion {
	styles := t.pendingInline(a)
	if styles == 0 || a != z {
		return v
	}
	t.pendingStyles = 0
	if strings.TrimSpace(v.text) == "" {
		return v
	}
	prefix, suffix := "", ""
	for _, kind := range []InlineKind{InlineCode, InlineStrike, InlineItalic, InlineBold} {
		if styles&(1<<kind) == 0 {
			continue
		}
		left, right := inlineDelimiters(kind, prefix+v.text+suffix)
		prefix, suffix = left+prefix, suffix+right
	}
	plain := t.projection.source[:a] + v.text + t.projection.source[z:]
	candidate := t.projection.source[:a] + prefix + v.text + suffix + t.projection.source[z:]
	expected := projectTypesetting(plain).text
	if projectTypesetting(candidate).text != expected && styles&(1<<InlineItalic) != 0 {
		prefix, suffix = strings.ReplaceAll(prefix, "_", "*"), strings.ReplaceAll(suffix, "_", "*")
		candidate = t.projection.source[:a] + prefix + v.text + suffix + t.projection.source[z:]
	}
	if projectTypesetting(candidate).text != expected {
		t.Notice = "Select the words you want to format."
		return v
	}
	v.text = prefix + v.text + suffix
	v.prefix = prefix + v.prefix
	return v
}

// CancelInlineIntent leaves source and history untouched when leaving Typeset.
func (t *Typesetter) CancelInlineIntent() { t.pendingStyles = 0 }

func (t *Typesetter) inlineAt(kind InlineKind, a, z int) (typesetInline, bool) {
	for _, m := range t.projection.inlines {
		if m.kind == kind && a >= m.content && z <= m.contentEnd {
			return m, true
		}
	}
	return typesetInline{}, false
}

// InlineActive also reflects a formatting intent at an empty caret. That
// intent never writes placeholder words or empty syntax into the document.
func (t *Typesetter) InlineActive(kind InlineKind) bool {
	a, z := t.Document.Selection()
	start, end := t.projection.sourceRange(a, z)
	if a == z {
		start = t.sourceCaret(a)
		end = start
	}
	if a == z && t.pendingInline(start)&(1<<kind) != 0 {
		return true
	}
	_, ok := t.inlineAt(kind, start, end)
	return ok
}

// inlineRange includes complete nested wrappers while refusing selections
// that would cut across half of a different format.
func (t *Typesetter) inlineRange() (int, int, bool) {
	t.ToSource()
	sel := t.Source.Selection()
	a, z := sel.Start(), sel.End()
	if t.projection.sourceLine(a) != t.projection.sourceLine(z) || !t.projection.inlineEligible(t.projection.sourceLine(a)) {
		t.Notice = "Choose text within one paragraph to format it."
		return a, z, false
	}
	if a == z {
		return a, z, true
	}
	va, vz := t.Document.Selection()
	va, vz = min(va, vz), max(va, vz)
	for _, m := range t.projection.inlines {
		ma, mz := t.projection.viewOffset(m.content), t.projection.viewOffset(m.contentEnd)
		if va <= ma && vz >= mz {
			a, z = min(a, m.start), max(z, m.end)
		}
		if va > ma && va < mz && vz > mz || vz > ma && vz < mz && va < ma {
			t.Notice = "Keep the selection inside a format, or include it completely."
			return a, z, false
		}
	}
	return a, z, true
}

// ToggleInline applies a source-backed format. A caret inside a format clears
// that span; a selection clears only its selected portion. Empty plain carets
// arm the next insertion without introducing fake document content.
func (t *Typesetter) ToggleInline(kind InlineKind) bool {
	if kind < InlineBold || kind > InlineCode {
		return false
	}
	t.ToSource()
	sel := t.Source.Selection()
	a, z := sel.Start(), sel.End()
	if m, ok := t.inlineAt(kind, a, z); ok {
		body := t.projection.source[m.content:m.contentEnd]
		var start, end int
		if a != z && (a > m.content || z < m.contentEnd) {
			wrap := func(s string) string {
				trim := strings.TrimFunc(s, unicode.IsSpace)
				if trim == "" {
					return s
				}
				left, right := inlineDelimiters(kind, trim)
				at := strings.Index(s, trim)
				return s[:at] + left + trim + right + s[at+len(trim):]
			}
			left, right := wrap(t.projection.source[m.content:a]), wrap(t.projection.source[z:m.contentEnd])
			body = left + t.projection.source[a:z] + right
			start, end = m.start+len(left), m.start+len(left)+(z-a)
		} else {
			start, end = a-(m.content-m.start), z-(m.content-m.start)
		}
		return t.replaceInline(m.start, m.end, body, start, end, true)
	}
	a, z, ok := t.inlineRange()
	if !ok {
		return false
	}
	if a == z {
		t.pendingInline(a)
		t.pendingStyles ^= 1 << kind
		t.pendingStyleAt, t.pendingStyleVersion = a, t.Source.Buffer.Version()
		t.Notice = ""
		return true
	}
	body := t.projection.source[a:z]
	trim := strings.TrimFunc(body, unicode.IsSpace)
	if trim == "" {
		t.Notice = "Select a word or a sentence to format it."
		return false
	}
	first := strings.Index(body, trim)
	a += first
	z = a + len(trim)
	left, right := inlineDelimiters(kind, trim)
	if kind == InlineItalic {
		candidate := t.projection.source[:a] + left + trim + right + t.projection.source[z:]
		if projectTypesetting(candidate).text != t.projection.text {
			left, right = "*", "*"
		}
	}
	return t.replaceInline(a, z, left+trim+right, a+len(left), a+len(left)+len(trim), true)
}

func (t *Typesetter) replaceInline(a, z int, text string, start, end int, sameText bool) bool {
	candidate := t.projection.source[:a] + text + t.projection.source[z:]
	if sameText && projectTypesetting(candidate).text != t.projection.text {
		t.Notice = "This selection needs Markdown source to keep its nested formatting."
		return false
	}
	anchor, caret := start, end
	beforeA, beforeZ := t.Document.Selection()
	if beforeA > beforeZ {
		anchor, caret = end, start
		t.Source.Select(z, a)
	} else {
		t.Source.Select(a, z)
	}
	if text == "" {
		t.Source.Run(editor.Delete)
	} else {
		t.Source.Insert(text)
	}
	t.Source.Select(anchor, caret)
	t.FromSource()
	t.pendingStyles = 0
	t.Notice = ""
	return true
}
