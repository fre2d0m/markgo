package app

import (
	"fmt"
	"regexp"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

type visualFind struct {
	open, replacing, caseSensitive, wholeWord, regex, inSelection bool
	query, replacement, fileID, searched                          string
	options                                                       [4]bool
	version                                                       uint64
	matches                                                       [][]int
	current, scopeStart, scopeEnd                                 int
	scopeText                                                     string
	scopeInvalid                                                  bool
	pattern                                                       *regexp.Regexp
	err                                                           error
}

func (a *App) openFind() {
	if a.reading {
		a.setWritingMode(true, false)
	}
	f := a.current()
	if !a.preview {
		f.State.Run(editor.Find)
		return
	}
	a.writer(f).Suspend()
	sel := f.State.Selection()
	query := a.find.query
	if !sel.Empty() && f.State.Buffer.Pos(sel.Start()).Line == f.State.Buffer.Pos(sel.End()).Line {
		query = f.State.SelectedText()
	}
	a.find = visualFind{open: true, query: query, fileID: f.ID, scopeStart: sel.Start(), scopeEnd: sel.End(), scopeText: f.State.Buffer.String()}
	a.refreshVisualFind(true)
}

func (a *App) dismissVisualFind() {
	a.find = visualFind{query: a.find.query}
}

func (a *App) refreshVisualFind(force bool) {
	f, doc := &a.find, a.current()
	if !f.open || f.fileID != doc.ID {
		return
	}
	opts := [4]bool{f.caseSensitive, f.wholeWord, f.regex, f.inSelection}
	version := doc.State.Buffer.Version()
	if !force && f.version == version && f.searched == f.query && f.options == opts {
		return
	}
	text := doc.State.Buffer.String()
	f.moveScope(text)
	f.version, f.searched, f.options = version, f.query, opts
	f.matches, f.err, f.pattern = nil, nil, nil
	if f.query == "" || (f.inSelection && f.scopeInvalid) {
		return
	}
	pattern := f.query
	if !f.regex {
		pattern = regexp.QuoteMeta(pattern)
	}
	if !f.caseSensitive {
		pattern = "(?i)" + pattern
	}
	f.pattern, f.err = regexp.Compile("(?m)" + pattern)
	if f.err != nil {
		return
	}
	word := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }
	for _, m := range f.pattern.FindAllStringSubmatchIndex(text, 100000) {
		if m[0] == m[1] || (f.inSelection && (m[0] < f.scopeStart || m[1] > f.scopeEnd || f.scopeStart == f.scopeEnd)) {
			continue
		}
		if f.wholeWord {
			before, _ := utf8.DecodeLastRuneInString(text[:m[0]])
			after, _ := utf8.DecodeRuneInString(text[m[1]:])
			if word(before) || word(after) {
				continue
			}
		}
		f.matches = append(f.matches, m)
	}
	f.current = 0
	for i, m := range f.matches {
		if m[0] >= doc.State.Selection().Start() {
			f.current = i
			break
		}
	}
	a.selectFindMatch()
}

// Keep the original search passage inclusive of replacements at its edges.
// Native editor selections exclude edge insertions; using one for scope would
// silently drop the first/last replacement. This immutable last-search string
// is only an offset baseline; Source remains the sole editable authority.
func (f *visualFind) moveScope(next string) {
	old := f.scopeText
	if old == next {
		return
	}
	a := 0
	for a < min(len(old), len(next)) && old[a] == next[a] {
		a++
	}
	for a > 0 && a < len(old) && !utf8.RuneStart(old[a]) {
		a--
	}
	z, n := len(old), len(next)
	for z > a && n > a && old[z-1] == next[n-1] {
		z--
		n--
	}
	for z < len(old) && !utf8.RuneStart(old[z]) {
		z++
		n++
	}
	// A diff crossing a scope boundary may combine several distant edits
	// (e.g. undoing an earlier replace-all). Never widen the search to text
	// outside the chosen passage when the boundary cannot be retained.
	if (a < f.scopeStart && z > f.scopeStart) || (a < f.scopeEnd && z > f.scopeEnd) {
		f.scopeInvalid = true
	}
	move := func(at int, end bool) int {
		if at < a {
			return at
		}
		if at > z {
			return at + n - z
		}
		if end {
			return n
		}
		return a
	}
	empty := f.scopeStart == f.scopeEnd
	f.scopeStart, f.scopeEnd = move(f.scopeStart, false), move(f.scopeEnd, true)
	if empty {
		f.scopeEnd = f.scopeStart
	}
	f.scopeText = next
}

func (a *App) selectFindMatch() {
	f := &a.find
	if len(f.matches) == 0 {
		return
	}
	m := f.matches[f.current]
	a.current().State.Select(m[0], m[1])
	a.writer(a.current()).SyncSelection()
}

func (a *App) replaceVisualMatches(all bool) {
	a.refreshVisualFind(false)
	f := &a.find
	if len(f.matches) == 0 || f.err != nil {
		return
	}
	doc := a.current()
	text := doc.State.Buffer.String()
	chosen := f.matches
	if !all {
		chosen = f.matches[f.current : f.current+1]
	}
	edits := make([]editor.Edit, 0, len(chosen))
	for _, m := range chosen {
		replacement := f.replacement
		if f.regex {
			replacement = string(f.pattern.ExpandString(nil, replacement, text, m))
		}
		edits = append(edits, editor.Edit{Start: m[0], End: m[1], Text: replacement})
	}
	doc.State.Buffer.Edit(edits...)
	end := edits[0].Start + len(edits[0].Text)
	doc.State.Select(end, end)
	a.writer(doc).FromSource()
	a.refreshVisualFind(true)
}

func (a *App) visualFindBar(c *ui.Context) {
	if !a.find.open || a.current().ID != a.find.fileID || !a.preview || a.reading {
		return
	}
	a.refreshVisualFind(false)
	// Reserve real layout space. A floating panel can cover the very match
	// the rich editor just revealed, especially in the first paragraph.
	ui.Row(c.Key("visual-find")).FillWidth().Shrink(0).Padding(12, 20, 0).Justify(ui.End).Children(func() {
		bar := ui.Column(c).Label("Find in document panel")
		renderDocumentFind(c, bar, visualFindControls{a})
		if bar.Shortcut(0, ui.KeyEscape) {
			visualFindControls{a}.Close()
		}
	})
}

type visualFindControls struct{ app *App }

func (f visualFindControls) Query() *string       { return &f.app.find.query }
func (f visualFindControls) Replacement() *string { return &f.app.find.replacement }
func (f visualFindControls) CaseSensitive() *bool { return &f.app.find.caseSensitive }
func (f visualFindControls) WholeWord() *bool     { return &f.app.find.wholeWord }
func (f visualFindControls) Regexp() *bool        { return &f.app.find.regex }
func (f visualFindControls) InSelection() *bool   { return &f.app.find.inSelection }
func (f visualFindControls) Replacing() bool      { return f.app.find.replacing }
func (f visualFindControls) SetReplacing(on bool) { f.app.find.replacing = on }
func (f visualFindControls) CanReplace() bool     { return true }
func (f visualFindControls) Matches() (int, int)  { return len(f.app.find.matches), f.app.find.current }
func (f visualFindControls) Replace()             { f.app.replaceVisualMatches(false) }
func (f visualFindControls) ReplaceAll()          { f.app.replaceVisualMatches(true) }
func (f visualFindControls) Next()                { f.step(1) }
func (f visualFindControls) Previous()            { f.step(-1) }
func (f visualFindControls) Close() {
	f.app.dismissVisualFind()
	f.app.current().Focus.Focus()
}
func (f visualFindControls) Status() string {
	if f.app.find.inSelection && f.app.find.scopeInvalid {
		return "Select passage again"
	}
	if f.app.find.err != nil {
		return "Invalid pattern"
	}
	n := len(f.app.find.matches)
	if n == 0 {
		return "0 matches"
	}
	return fmt.Sprintf("%d of %d", f.app.find.current+1, n)
}
func (f visualFindControls) step(direction int) {
	f.app.refreshVisualFind(false)
	if n := len(f.app.find.matches); n > 0 {
		f.app.find.current = (f.app.find.current + direction + n) % n
		f.app.selectFindMatch()
	}
}
func (f visualFindControls) QueryInput(in ui.Element) ui.Element {
	return in.AutoFocus().OnChange(func() { f.app.refreshVisualFind(true) }).OnSubmit(f.Next)
}
func (f visualFindControls) ReplacementInput(in ui.Element) ui.Element {
	return in.OnSubmit(f.Replace)
}
