package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/xddcode/markgo/internal/mdeditor"
)

// TypesettingPreview is an explicitly launched sample workspace. It has no
// file-open/save hooks, recovery store, single-instance handoff or preferences
// store. Closing it discards only its in-memory samples.
type TypesettingPreview struct {
	shell         *App
	pages         []*previewPage
	selected      int
	sourceMode    bool
	sidebar       bool
	formatOpen    bool
	railSelection int
}

type previewPage struct {
	file         *mdeditor.File
	typesetter   *mdeditor.Typesetter
	description  string
	stats        mdeditor.Statistics
	statsVersion uint64
	statsReady   bool
}

func NewTypesettingPreview() *TypesettingPreview {
	p := &TypesettingPreview{shell: New(), sidebar: true, railSelection: -1}
	p.shell.appearance.Theme = "paper"
	for _, sample := range []struct{ name, description, source string }{
		{"A quieter morning", "An ordinary beginning", quietMorning},
		{"慢一点，写下来", "让文字有自己的节奏", quietChinese},
		{"Markdown notes", "Headings, code & references", typesettingMarkdown},
		{"Blank page", "Room for your own words", ""},
		{"Long read", "A longer writing session", "# Room to think\n\n" + strings.Repeat("## An unhurried thought\n\nThere is no need to hurry the page. One observation leads to another, until the shape of an idea begins to emerge. 写作，是让模糊的想法慢慢变得清晰。\n\n", 700)},
		{"A little structure", "Ideas, steps & small intentions", typesettingLists},
		{"Words worth keeping", "Quotations & small experiments", typesettingBlocks},
	} {
		f := mdeditor.NewFile(sample.name+".md", sample.source)
		p.pages = append(p.pages, &previewPage{file: f, typesetter: mdeditor.NewTypesetter(f.State), description: sample.description})
	}
	return p
}

func (p *TypesettingPreview) WindowOptions() mygo.WindowOptions {
	o := p.shell.WindowOptions("")
	o.Title = "MarkGo · Writing preview"
	o.Content = ui.View(p.View)
	return o
}

func (p *TypesettingPreview) Attach(window *mygo.Window) {
	// Deliberately do not call App.Attach: that owns the real file lifecycle.
	p.shell.window, p.shell.windowCommands = window, window
	window.OnMaximize(func() { p.shell.windowMaximized = true; window.Invalidate() })
	window.OnUnmaximize(func() { p.shell.windowMaximized = false; window.Invalidate() })
}

func (p *TypesettingPreview) current() *previewPage { return p.pages[p.selected] }

func (p *TypesettingPreview) setMode(source bool) {
	if p.sourceMode == source {
		return
	}
	if source {
		p.current().typesetter.ToSource()
	} else {
		p.current().typesetter.FromSource()
	}
	p.sourceMode = source
}

func (p *TypesettingPreview) View(c *ui.Context) {
	a := p.shell
	a.systemDark = c.Theme().Dark
	c.SetTheme(writingTheme(c.Theme(), a.appearance.Theme))
	c.SetMenuPresentation(ui.PopoverMenus)
	t := c.Theme()
	page := p.current()
	if !p.sourceMode {
		page.typesetter.Sync()
	}
	if c.Shortcut(ui.Cmd, ui.KeyBackslash) {
		p.sidebar = !p.sidebar
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyM) {
		p.setMode(!p.sourceMode)
		page.file.Focus.Focus(c)
	}
	if c.Shortcut(ui.Cmd, ui.KeyComma) {
		a.appearanceOpen = !a.appearanceOpen
		p.formatOpen = false
	}
	if (a.appearanceOpen || p.formatOpen) && c.Shortcut(0, ui.KeyEscape) {
		a.appearanceOpen = false
		p.formatOpen = false
	}
	root := ui.Column(c.Key("writing-preview")).Fill().Background(t.Background)
	target := float32(0)
	if p.sidebar {
		target = sidebarWidth
	}
	width := root.AnimateTo("rail", target, ui.Animation{Duration: 180 * time.Millisecond, Ease: ui.EaseOut})
	root.Children(func() {
		p.header(c)
		ui.Row(c.Key("preview-body")).Grow(1).FillWidth().MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			if width > 0.5 {
				ui.Column(c.Key("preview-rail")).Width(width).Shrink(0).Clip().FillHeight().Disabled(!p.sidebar).Children(func() { p.rail(c) })
			}
			ui.Column(c.Key("preview-page")).Grow(1).MinWidth(0).FillHeight().Children(func() {
				if page.typesetter.Notice != "" {
					ui.Row(c.Key("editing-notice")).FillWidth().Padding(0, 24).MinHeight(44).Gap(8).AlignItems(ui.Center).Background(t.Surface).Children(func() {
						ui.Text(c, page.typesetter.Notice).FontSize(12).TextColor(t.TextMuted).Grow(1).MinWidth(0).Role(ui.RoleStatus)
						if iconButton(c, "dismiss", "close", "Dismiss writing notice").Clicked() {
							page.typesetter.Notice = ""
						}
					})
				}
				ui.Row(c.Key("page-inset")).Grow(1).FillWidth().MinHeight(0).Padding(52, 48, 28).Justify(ui.Center).AlignItems(ui.Stretch).Children(func() {
					f := page.file
					font := ui.Font{Family: "system-ui", Size: a.appearance.FontSize}
					if a.appearance.Serif {
						font.Family = "serif"
					}
					var view ui.Element
					if p.sourceMode {
						f.State.SetReplacements(nil)
						f.State.Font = ui.Font{Family: "monospace", Size: 15}
						f.State.Theme = proseColors(t)
						view = editor.View(c.Key("source-"+f.ID), f.State).Label("Markdown source")
					} else {
						view = mdeditor.TypesettingView(c.Key("typeset-"+f.ID), page.typesetter, font)
					}
					view.Bind(&f.Focus).Grow(1).MaxWidth(a.appearance.Width).MinWidth(0).FillHeight().AutoFocus()
				})
			})
		})
		p.footer(c)
	})
}

func (p *TypesettingPreview) header(c *ui.Context) {
	t, a := c.Theme(), p.shell
	page := p.current()
	ui.Row(c.Key("preview-titlebar")).FillWidth().Height(titleBarHeight).DragWindow().AlignItems(ui.Center).Children(func() {
		brand := ui.Row(c).Width(sidebarWidth).Shrink(0).FillHeight().Padding(0, 20).Gap(8).AlignItems(ui.Center)
		if p.sidebar {
			brand.Background(t.Surface)
		}
		brand.Children(func() {
			ui.Icon(c, icons["mark"]).Size(17, 17).TextColor(t.Text)
			ui.Text(c, "markgo").FontSize(13).FontWeight(600).LetterSpacing(-0.3)
			ui.Spacer(c)
			if iconButton(c, "sidebar", "sidebar", "Toggle sample pages").Clicked() {
				p.sidebar = !p.sidebar
			}
		})
		ui.Text(c, strings.TrimSuffix(page.file.Name, ".md")).FontSize(12).TextColor(t.TextMuted).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").MarginX(20)
		ui.Row(c.Key("view-modes")).Height(40).Gap(2).AlignItems(ui.Center).Children(func() {
			for _, mode := range []struct {
				label  string
				source bool
			}{{"Typeset", false}, {"Source", true}} {
				b := ui.ButtonBase(c.Key(mode.label)).Size(68, 40).KeepFocus().Center().Radius(7).Label(mode.label + " mode").Tooltip("Switch view · Cmd/Ctrl+Shift+M")
				fill, ink := ui.Transparent, t.TextMuted
				if p.sourceMode == mode.source {
					fill, ink = t.SurfaceHover, t.Text
				}
				if b.Hovered() {
					fill = t.SurfaceHover
					ink = t.Text
				}
				b.Background(b.AnimateColor("fill", fill, ui.Animation{Duration: 110 * time.Millisecond, NonMotion: true}))
				b.Children(func() { ui.Text(c, mode.label).FontSize(12).FontWeight(500).TextColor(ink) })
				if b.Clicked() {
					p.setMode(mode.source)
					page.file.Focus.Focus(c)
				}
			}
		})
		format := stateIconButton(c, "format", "format", "Paragraph style", p.formatOpen)
		if format.Clicked() {
			p.formatOpen = !p.formatOpen
			a.appearanceOpen = false
		}
		ui.PopoverBase(c, format, &p.formatOpen, func(panel ui.Element) {
			floatingPanel(panel).Width(220).Padding(6).Gap(2).Margin(8, 0, 0).Radius(12).Background(t.Background).Border(1, t.Border.Alpha(0.7)).Shadow(0, 8, 28, 0, ui.RGBA(0, 0, 0, 0.09)).Children(func() {
				for level, label := range []string{"Paragraph", "Heading 1", "Heading 2", "Heading 3"} {
					b := ui.ButtonBase(c.Key(label)).FillWidth().Height(44).Padding(0, 12).Radius(6).KeepFocus().Label(label)
					if b.Hovered() {
						b.Background(t.SurfaceHover)
					}
					b.Children(func() {
						ui.Row(c).Fill().AlignItems(ui.Center).Children(func() {
							ui.Text(c, label).FontSize(13).Grow(1)
							ui.Textf(c, "%d", level).FontSize(11).TextColor(t.TextMuted)
						})
					})
					if b.Clicked() {
						if p.sourceMode {
							page.typesetter.FromSource()
						}
						page.typesetter.Heading(level)
						p.formatOpen = false
						page.file.Focus.Focus(c)
					}
				}
				ui.Box(c).Height(1).FillWidth().Margin(4, 6).Background(t.Border)
				for _, action := range []struct {
					label, icon, hint string
					kind              mdeditor.ListKind
				}{
					{"Bulleted list", "list", "⇧8", mdeditor.BulletedList},
					{"Numbered list", "numbered-list", "⇧7", mdeditor.NumberedList},
					{"Task list", "task-list", "⇧9", mdeditor.TaskList},
				} {
					b := ui.ButtonBase(c.Key(action.label)).FillWidth().Height(44).Padding(0, 12).Radius(6).KeepFocus().Label(action.label).Tooltip("Cmd/Ctrl+" + action.hint)
					if b.Hovered() {
						b.Background(t.SurfaceHover)
					}
					b.Children(func() {
						ui.Row(c).Fill().Gap(10).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, icons[action.icon]).Size(16, 16).TextColor(t.TextMuted)
							ui.Text(c, action.label).FontSize(13).Grow(1)
							ui.Text(c, action.hint).FontSize(11).TextColor(t.TextMuted)
						})
					})
					if b.Clicked() {
						if p.sourceMode {
							page.typesetter.FromSource()
						}
						page.typesetter.SetList(action.kind)
						p.formatOpen = false
						page.file.Focus.Focus(c)
					}
				}
				ui.Box(c).Height(1).FillWidth().Margin(4, 6).Background(t.Border)
				for _, action := range []struct {
					label, icon, hint string
					apply             func()
				}{
					{"Quote", "quote", "⇧Q", page.typesetter.Quote},
					{"Code block", "code", "⇧K", page.typesetter.CodeBlock},
				} {
					b := ui.ButtonBase(c.Key(action.label)).FillWidth().Height(44).Padding(0, 12).Radius(6).KeepFocus().Label(action.label).Tooltip("Cmd/Ctrl+" + action.hint)
					if b.Hovered() {
						b.Background(t.SurfaceHover)
					}
					b.Children(func() {
						ui.Row(c).Fill().Gap(10).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, icons[action.icon]).Size(16, 16).TextColor(t.TextMuted)
							ui.Text(c, action.label).FontSize(13).Grow(1)
							ui.Text(c, action.hint).FontSize(11).TextColor(t.TextMuted)
						})
					})
					b.OnClick(func() {
						if p.sourceMode {
							page.typesetter.FromSource()
						}
						action.apply()
						p.formatOpen = false
						page.file.Focus.Focus(c)
					})
				}
			})
		})
		if iconButton(c, "undo", "undo", "Undo writing change").Disabled(!page.file.State.Buffer.CanUndo()).Clicked() {
			if p.sourceMode {
				page.file.State.Run(editor.Undo)
			} else {
				page.typesetter.Undo(false)
			}
		}
		appearance := stateIconButton(c, "appearance", "appearance", "Appearance", a.appearanceOpen)
		if appearance.Clicked() {
			a.appearanceOpen = !a.appearanceOpen
			p.formatOpen = false
		}
		a.appearancePopover(c, appearance)
		a.captionButtons(c)
	})
}

const typesettingLists = `# A little structure

Some thoughts need a little room. Others need a list.

## Gather a few ideas

- Start with something you noticed today.
- Let the next thought follow naturally.
  - 一点观察，一点好奇。
  - Leave room for the unexpected.

## One step at a time

3. Find the sentence you want to keep.
4. Give it a little more attention.

## Small intentions

- [ ] 写下一个还没想清楚的念头。
- [x] Make a little space for writing.
- [ ] Come back when you have something to add.
`

const typesettingBlocks = "# Words worth keeping\n\nA quotation leaves room for another voice.\n\n> 留下真正打动你的那一句。\n> A few honest words can change the shape of a day.\n> > Let the thought unfold.\n\n```go\nfunc begin() {\n    fmt.Println(\"慢一点，写下来。\")\n}\n```\n\nThe page is yours to continue.\n"

func (p *TypesettingPreview) rail(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(sidebarWidth).FillHeight().Padding(22, 12, 16).Gap(10).Background(t.Surface).Children(func() {
		ui.Text(c, "Sample pages").FontSize(11).FontWeight(500).TextColor(t.TextMuted).Margin(0, 10, 6)
		ui.Scroll(c.Key("sample-list")).FillWidth().Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().Gap(10).Children(func() {
				for i, page := range p.pages {
					b := ui.ButtonBase(c.Key(page.file.ID)).FillWidth().Height(58).Padding(10).Radius(8).Label("Open sample " + strings.TrimSuffix(page.file.Name, ".md"))
					fill := ui.Transparent
					if p.selected == i {
						fill = t.SurfacePressed.Alpha(0.65)
						if p.railSelection != i {
							b.ScrollIntoView()
							p.railSelection = i
						}
					} else if b.Hovered() {
						fill = t.SurfaceHover
					}
					if b.Pressed() {
						fill = t.SurfacePressed
					}
					b.Background(b.AnimateColor("fill", fill, ui.Animation{Duration: 110 * time.Millisecond, NonMotion: true}))
					b.Children(func() {
						ui.Column(c).FillWidth().Gap(5).Children(func() {
							ui.Text(c, strings.TrimSuffix(page.file.Name, ".md")).FontSize(12).FontWeight(500).SingleLine().Ellipsis("…")
							ui.Text(c, page.description).FontSize(10).TextColor(t.TextMuted).SingleLine().Ellipsis("…")
						})
					})
					if b.Clicked() {
						if !p.sourceMode {
							p.current().typesetter.ToSource()
						}
						p.selected = i
						if !p.sourceMode {
							page.typesetter.FromSource()
						}
						page.file.Focus.Focus(c)
					}
				}
			})
		})
		ui.Column(c).Padding(10).Gap(7).Children(func() {
			ui.Text(c, "A space to try things.").FontSize(11).TextColor(t.TextMuted)
			ui.Text(c, "These pages stay in memory.\nYour files stay untouched.").FontSize(11).TextColor(t.TextMuted).LineHeight(1.6)
		})
	})
}

func (p *TypesettingPreview) footer(c *ui.Context) {
	t, page := c.Theme(), p.current()
	if version := page.file.State.Buffer.Version(); !page.statsReady || page.statsVersion != version {
		page.stats = mdeditor.CountText(page.file.State.Buffer.String())
		page.statsVersion = version
		page.statsReady = true
	}
	ui.Row(c).Height(30).FillWidth().Padding(0, 22).Gap(18).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Writing preview · changes stay in memory").FontSize(11).TextColor(t.TextMuted).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
		ui.Textf(c, "%d characters", page.stats.Characters).FontSize(11).FontFeatures("tnum").TextColor(t.TextMuted)
	})
}

const quietMorning = `# A quieter morning

The house is still. There is time to finish a sentence, to find the right word, to leave a little space.

I used to begin with a list of everything I needed to do. Today, I opened the window first. The air was cool, and somewhere below, a bicycle passed through a patch of sunlight.

## Notice the ordinary

The first cup of tea. A half-read book. The sound of rain before you see it. There is more to write about than we think, if we give it a moment.

Writing does not always begin with an idea. Sometimes it begins with paying attention.

## Leave room for tomorrow

Not every thought needs an ending today. A few honest lines are enough to return to.
`

const quietChinese = `# 慢一点，写下来

清晨的房间很安静。窗外有风，杯子里的水还温着。我想把一些细小的感受留下来，不必着急，也不必一开始就找到答案。

## 从一个句子开始

写作有时候不是为了表达一个已经想清楚的观点，而是让模糊的想法慢慢变得清晰。删去一个多余的词，停顿一下，再往前走。

我们需要的也许只是一张干净的纸，一段不被打扰的时间，和愿意认真观察生活的心情。

## 留一点空白

没有写完的句子，可以留到明天。好的文字有自己的节奏，就像一条熟悉的小路，不需要每一步都走得很快。

中英文可以自然地放在一起：a little space to think。偶尔也可以留下一个小小的记号 ☀️。
`

const typesettingMarkdown = "---\ntitle: A small notebook\n---\n\n# Markdown stays yours\n\nHeadings, lists, simple quotations and fenced code are editable here. Links and tables keep their Markdown syntax.\n\n## A code fragment\n\n```go\nfmt.Println(\"Hello, writing\")\n```\n\n## A reference\n\nRead [a quiet page][note] when you have a moment.\n\n[note]: https://example.test/writing \"Notes\"\n\n| Thought | Next step |\n| --- | --- |\n| Begin | Write one sentence |\n"
