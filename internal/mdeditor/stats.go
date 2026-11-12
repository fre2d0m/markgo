package mdeditor

import (
	"unicode"
	"unicode/utf8"
)

type Statistics struct {
	Words      int
	Characters int
}

// CountText counts CJK characters individually and groups other letters and
// digits into words. It is intended for extracted document text, not Markdown
// syntax or link destinations.
func CountText(text string) Statistics {
	stats := Statistics{Characters: utf8.RuneCountInString(text)}
	inWord := false
	runes := []rune(text)
	for i, r := range runes {
		switch {
		case unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul):
			stats.Words++
			inWord = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if !inWord {
				stats.Words++
			}
			inWord = true
		case unicode.IsMark(r) && inWord:
			// Combining marks belong to the preceding word.
		case (r == '\'' || r == '’') && inWord && i+1 < len(runes) && unicode.IsLetter(runes[i+1]):
			// Keep contractions such as "don't" together.
		default:
			inWord = false
		}
	}
	return stats
}
