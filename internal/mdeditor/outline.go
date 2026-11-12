package mdeditor

// Heading points into the original, UTF-8 Markdown source, not rendered text.
type Heading struct {
	Title  string
	Level  int
	Offset int
}

// Outline extracts headings with the same parser used by MyGo's Markdown
// renderer. Fenced code and other non-heading blocks are not treated as titles.
func Outline(source string) []Heading {
	return Analyze(source).Headings
}
