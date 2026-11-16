package mdeditor

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/xddcode/markgo/internal/writingui"
)

type slashCommand struct {
	id, label, keywords, glyph string
	heading                    int
	list                       ListKind
}

var slashCommands = []slashCommand{
	{id: "paragraph", label: "Paragraph", keywords: "text plain normal 段落 正文 普通", glyph: "paragraph"},
	{id: "h1", label: "Heading 1", keywords: "h1 title 标题 一级 大标题", glyph: "H1", heading: 1},
	{id: "h2", label: "Heading 2", keywords: "h2 subtitle 标题 二级", glyph: "H2", heading: 2},
	{id: "h3", label: "Heading 3", keywords: "h3 标题 三级", glyph: "H3", heading: 3},
	{id: "bullet", label: "Bullet list", keywords: "bullets unordered list 列表 无序 项目", glyph: "list", list: BulletedList},
	{id: "number", label: "Numbered list", keywords: "ordered numbers list 列表 有序 编号", glyph: "number", list: NumberedList},
	{id: "task", label: "Task list", keywords: "todo checkbox check 待办 任务 清单 列表", glyph: "task", list: TaskList},
	{id: "quote", label: "Quote", keywords: "blockquote 引用 引述", glyph: "quote"},
	{id: "code", label: "Code block", keywords: "code fence pre 代码 代码块", glyph: "code"},
}

var slashIcons = func() map[string]*ui.SVG {
	paths := map[string]string{
		"paragraph": `<path d="M13 4H9a5 5 0 0 0 0 10h4M13 4v16M18 4v16M10 4h11"/>`,
		"list":      `<path d="M9 6h12M9 12h12M9 18h12"/><circle cx="3.5" cy="6" r=".6"/><circle cx="3.5" cy="12" r=".6"/><circle cx="3.5" cy="18" r=".6"/>`,
		"number":    `<path d="M10 6h11M10 12h11M10 18h11M3 4l2-1v6M3 9h4M3 15c0-3 4-3 4 0 0 2-4 2-4 5h4"/>`,
		"task":      `<rect x="4" y="4" width="16" height="16" rx="3"/><path d="m8 12 3 3 5-6"/>`,
		"quote":     `<path d="M9 7H5v7h4v-4H5m14-3h-4v7h4v-4h-4M9 14c0 3-2 4-4 4m14-4c0 3-2 4-4 4"/>`,
		"code":      `<path d="m8 6-5 6 5 6m8-12 5 6-5 6M14 4l-4 16"/>`,
	}
	icons := make(map[string]*ui.SVG, len(paths))
	for name, path := range paths {
		icons[name] = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round">` + path + `</svg>`))
	}
	return icons
}()

type slashMenu struct {
	armed, open, reveal bool
	start, end, chosen  int
	query               string
	matches             []int
	anchorX             float32
}

// Arm only from an actual typed slash in an empty, ordinary source line.
// Loading/pasting a path, code, metadata and composition never arm the menu.
func (t *Typesetter) armSlash() {
	if t.composing {
		return
	}
	t.Sync()
	a, z := t.Document.Selection()
	if a != z {
		return
	}
	i := t.projection.sourceLine(t.sourceCaret(z))
	line := t.projection.lines[i]
	if !t.slashLine(i) || strings.TrimSpace(t.projection.source[line.start:line.end]) != "" {
		return
	}
	caret, _ := t.Document.CaretRect()
	t.slash = slashMenu{armed: true, start: line.start, reveal: true, anchorX: caret.X}
}

func (t *Typesetter) slashLine(i int) bool {
	line := t.projection.lines[i]
	return line.heading == 0 && line.list.kind == NoList && line.block == (typesetBlockLine{}) && t.projection.canList(i)
}

func (t *Typesetter) DismissSlash() { t.slash = slashMenu{} }

func (t *Typesetter) updateSlash() bool {
	u := &t.slash
	if !u.armed || t.composing {
		return false
	}
	t.Sync()
	a, z := t.Document.Selection()
	i := t.projection.sourceLine(t.sourceCaret(z))
	line := t.projection.lines[i]
	if a != z || line.start != u.start || !t.slashLine(i) || t.sourceCaret(z) != line.end {
		t.DismissSlash()
		return false
	}
	raw := strings.TrimLeft(t.projection.source[line.start:line.end], " \t")
	if !strings.HasPrefix(raw, "/") || utf8.RuneCountInString(raw) > 65 {
		t.DismissSlash()
		return false
	}
	query := raw[1:]
	for _, r := range query {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != ' ' && r != '-' {
			t.DismissSlash()
			return false
		}
	}
	if !u.open || query != u.query {
		u.query, u.chosen, u.reveal = query, 0, true
		u.matches = u.matches[:0]
		for i, cmd := range slashCommands {
			if strings.Contains(strings.ToLower(cmd.label+" "+cmd.keywords), strings.ToLower(strings.TrimSpace(query))) {
				u.matches = append(u.matches, i)
			}
		}
	}
	u.open, u.end = true, line.end
	return true
}

func (t *Typesetter) slashKey(key ui.Key, mods ui.Modifiers) bool {
	if mods != 0 || !t.updateSlash() {
		return false
	}
	u := &t.slash
	switch key {
	case ui.KeyEscape:
		t.DismissSlash()
		return true
	case ui.KeyDown, ui.KeyUp:
		if n := len(u.matches); n > 0 {
			direction := 1
			if key == ui.KeyUp {
				direction = -1
			}
			u.chosen = (u.chosen + direction + n) % n
			u.reveal = true
		}
		return true
	case ui.KeyEnter, ui.KeyTab:
		if len(u.matches) == 0 {
			t.DismissSlash()
			return false
		}
		t.applySlash(u.matches[u.chosen])
		return true
	}
	return false
}

func (t *Typesetter) applySlash(index int) {
	if !t.updateSlash() || index < 0 || index >= len(slashCommands) {
		return
	}
	start, end, command := t.slash.start, t.slash.end, slashCommands[index]
	t.DismissSlash()
	t.input.typing = nil
	t.ToSource()
	t.Source.Buffer.Transact(func() {
		t.Source.Select(start, end)
		t.Source.Run(editor.Delete)
		t.FromSource()
		switch {
		case command.list != NoList:
			t.SetList(command.list)
		case command.id == "quote":
			t.Quote()
		case command.id == "code":
			t.CodeBlock()
		default:
			t.Heading(command.heading)
		}
	})
	t.syncedSelection = t.Source.Selection()
}

func (t *Typesetter) slashPopover(c *ui.Context, view ui.Element) {
	if !t.updateSlash() || !view.Focused() {
		return
	}
	caret, visible := t.Document.CaretRect()
	if !visible {
		return
	}
	u, theme := &t.slash, c.Theme()
	_, height := c.Size()
	y := t.input.handle.Bounds(c).Y + caret.Y
	room := max(y-16, height-y-caret.H-16)
	// Fit whole 40 DIP rows and their 2 DIP gaps, never half a label.
	rows := max(1, min(len(slashCommands), int((room-80)/42)))
	listHeight := float32(rows*42 - 2)
	view.Children(func() {
		anchor := ui.Box(c.Key("slash-anchor")).Absolute().Left(u.anchorX).Top(caret.Y).Size(1, caret.H)
		panel := ui.PopoverBase(c.Key("slash-popup"), anchor, &u.open, func(panel ui.Element) {
			writingui.Surface(c, panel).Width(280).Padding(8).Gap(4).Margin(8, 0, 0).KeepFocus().Label("Insert block menu").Children(func() {
				ui.Text(c, "Insert a block").FontSize(12).FontWeight(500).TextColor(theme.Text).Padding(4, 8, 6)
				ui.Scroll(c.Key("commands")).FillWidth().MaxHeight(listHeight).MinHeight(40).Role(ui.RoleListBox).Label("Block styles").Children(func() {
					ui.Column(c).FillWidth().Gap(2).Children(func() {
						if len(u.matches) == 0 {
							ui.Text(c, "No matching blocks").FontSize(13).TextColor(theme.TextMuted).Padding(12, 8).Role(ui.RoleStatus)
						}
						for pos, index := range u.matches {
							command := slashCommands[index]
							b := codeButton(c.Key(command.id), "Insert "+command.label).FillWidth().Padding(0, 10).Role(ui.RoleOption).Selected(pos == u.chosen)
							if pos == u.chosen {
								b.Background(theme.SurfaceHover).Description("Selected")
								view.ActiveDescendant(b)
								if u.reveal {
									b.ScrollIntoView()
								}
							}
							b.Children(func() {
								ui.Row(c).Fill().Gap(12).AlignItems(ui.Center).Children(func() {
									ui.Box(c).Size(20, 20).Center().Children(func() {
										if icon := slashIcons[command.glyph]; icon != nil {
											ui.Icon(c, icon).Size(18, 18).TextColor(theme.TextMuted)
										} else {
											ui.Text(c, command.glyph).FontSize(12).FontWeight(500).TextColor(theme.TextMuted)
										}
									})
									ui.Text(c, command.label).FontSize(13).TextColor(theme.Text)
								})
							})
							b.OnClick(func() { t.applySlash(index); t.input.handle.Focus(c) })
						}
					})
				})
				ui.Text(c, "↑ ↓ Choose   Enter Insert   Esc Dismiss").FontSize(11).TextColor(theme.TextMuted).Padding(6, 8, 4)
			})
		})
		if panel.Dismissed() {
			t.DismissSlash()
		}
	})
	u.reveal = false
}

func (t *Typesetter) emptyParagraphHint(c *ui.Context, view ui.Element, font ui.Font) {
	if !view.Focused() || t.composing || t.slash.open {
		return
	}
	a, z := t.Document.Selection()
	i := t.projection.sourceLine(t.sourceCaret(z))
	line := t.projection.lines[i]
	if a != z || !t.slashLine(i) || line.start != line.end {
		return
	}
	caret, ok := t.Document.CaretRect()
	if !ok {
		return
	}
	view.Description("Type / to insert a block")
	view.Children(func() {
		ui.Text(c.Key("empty-paragraph-hint"), "Type / to insert a block").Absolute().Left(caret.X + 4).Top(caret.Y).
			Font(font.Family).FontSize(font.Size).TextColor(c.Theme().TextMuted).PassThrough().AccessHidden(true)
	})
}
