package app

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/xddcode/markgo/internal/mdeditor"
)

type App struct {
	files     []*mdeditor.File
	selected  string
	query     string
	preview   bool
	assistant bool
	sidebar   bool
}

func New() *App {
	files := sampleDocuments()
	return &App{
		files:     files,
		selected:  files[0].Name,
		preview:   true,
		assistant: true,
		sidebar:   true,
	}
}

func (a *App) current() *mdeditor.File {
	for _, f := range a.files {
		if f.Name == a.selected {
			return f
		}
	}
	return a.files[0]
}

func (a *App) visibleFiles() []*mdeditor.File {
	query := strings.ToLower(strings.TrimSpace(a.query))
	if query == "" {
		return a.files
	}
	var files []*mdeditor.File
	for _, f := range a.files {
		if strings.Contains(strings.ToLower(f.Name), query) {
			files = append(files, f)
		}
	}
	return files
}

func (a *App) View(c *ui.Context) {
	c.SetTheme(markgoTheme(c.Theme()))
	ui.Column(c).Fill().Background(c.Theme().Background).Children(func() {
		a.header(c)
		ui.Row(c).Grow(1).FillWidth().AlignItems(ui.Stretch).Children(func() {
			if a.sidebar {
				a.fileSidebar(c)
			}
			a.editorPane(c, a.current())
			if a.assistant {
				a.assistantPane(c, a.current())
			}
		})
		a.statusBar(c, a.current())
	})
}

func markgoTheme(base *ui.Theme) *ui.Theme {
	t := *base
	t.Accent = ui.Hex("#6c5ce7")
	t.AccentHover = ui.Hex("#7c6cf0")
	t.AccentPressed = ui.Hex("#5948cf")
	t.AccentText = ui.Hex("#ffffff")
	t.Selection = t.Accent.Alpha(0.24)
	t.Focus = t.Accent.Alpha(0.58)
	t.Radius = 8
	return &t
}

func (a *App) header(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Height(56).Padding(0, 14).Gap(12).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
		if ui.Button(c, "Files").Label("Toggle file sidebar").KeepFocus().Clicked() {
			a.sidebar = !a.sidebar
		}
		ui.Row(c).Width(30).Height(30).Center().Radius(9).Background(t.Accent).Children(func() {
			ui.Text(c, "M").FontSize(16).Bold().TextColor(t.AccentText)
		})
		ui.Text(c, "MarkGo").FontSize(17).Bold()
		ui.Badge(c, "ALPHA")
		ui.Spacer(c)
		ui.Switch(c, &a.preview).Label("Live preview")
		ai := ui.Button(c, "AI co-writer").Label("AI assistant").KeepFocus()
		if a.assistant {
			ai.Background(t.Accent.Alpha(0.14)).TextColor(t.Accent)
		}
		if ai.Clicked() {
			a.assistant = !a.assistant
		}
	})
}

func (a *App) fileSidebar(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(224).Shrink(0).FillHeight().Padding(14, 10).Gap(10).Background(t.Surface).BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "WORKSPACE").FontSize(11).Bold().TextColor(t.TextMuted).Grow(1)
			ui.Badge(c, fmt.Sprintf("%d", len(a.files)))
		})
		ui.SearchField(c, &a.query).Placeholder("Filter documents").Label("Filter documents")
		files := a.visibleFiles()
		ui.Sidebar(c, &a.selected, func() {
			for _, f := range files {
				label := f.Name
				if f.State.Buffer.Modified() {
					label += "  •"
				}
				ui.SidebarItem(c.Key(f.Name), f.Name, nil, label)
			}
		}).Grow(1).Label("Documents").Children(func() {
			if len(files) == 0 {
				ui.Text(c, "No matching documents").FontSize(12).TextColor(t.TextMuted).Padding(12, 8)
			}
		})
		ui.Column(c).Padding(10, 12).Gap(3).Radius(t.Radius).Background(t.Background).Children(func() {
			ui.Text(c, "Plain Markdown").FontSize(12).Bold()
			ui.Text(c, "Local files. No lock-in.").FontSize(11).TextColor(t.TextMuted)
		})
	})
}

func (a *App) editorPane(c *ui.Context, f *mdeditor.File) {
	t := c.Theme()
	st := f.State
	st.SoftWrap = true
	if a.preview {
		st.SetReplacements(mdeditor.LivePreview(st))
	} else {
		st.SetReplacements(nil)
	}
	ui.Column(c).Grow(1).MinWidth(0).FillHeight().Children(func() {
		ui.Row(c).Height(44).Padding(0, 22).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, f.Name).FontSize(13).Bold()
			if st.Buffer.Modified() {
				ui.Text(c, "Edited").FontSize(11).TextColor(t.Accent)
			}
			ui.Spacer(c)
			ui.Text(c, "Markdown").FontSize(11).TextColor(t.TextMuted)
		})
		ui.Box(c).Grow(1).FillWidth().Padding(8, 28).Children(func() {
			editor.View(c.Key(f.Name), st).Fill().AutoFocus()
		})
	})
}

func (a *App) assistantPane(c *ui.Context, f *mdeditor.File) {
	t := c.Theme()
	ui.Column(c).Width(286).Shrink(0).FillHeight().Padding(16, 16).Gap(14).Background(t.Surface).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "AI co-writer").FontSize(15).Bold().Grow(1)
			ui.Badge(c, "LOCAL")
		})
		ui.Column(c).Padding(14).Gap(8).Radius(12).Background(t.Accent.Alpha(0.1)).Children(func() {
			ui.Text(c, "Write with a second mind.").FontSize(14).Bold().TextColor(t.Accent)
			ui.Text(c, "Polish, reshape and understand your documents without giving up control.").FontSize(12).LineHeight(1.5).TextColor(t.TextMuted)
		})
		selection := mdeditor.SelectedText(f.State)
		ui.Column(c).Padding(12).Gap(5).Radius(t.Radius).Border(1, t.Border).Children(func() {
			ui.Text(c, "CURRENT CONTEXT").FontSize(10).Bold().TextColor(t.TextMuted)
			if selection == "" {
				ui.Text(c, "Select text in the document to start an AI edit.").FontSize(12).LineHeight(1.5)
			} else {
				ui.Textf(c, "%d characters selected", utf8.RuneCountInString(selection)).FontSize(12)
				ui.Text(c, selection).FontSize(11).MaxLines(3).TextColor(t.TextMuted)
			}
		})
		ui.Column(c).Gap(10).Padding(4, 2).Children(func() {
			a.aiCapability(c, "01", "Review every edit", "Accept or reject changes through a diff.")
			a.aiCapability(c, "02", "Use local models", "Connect Ollama or LM Studio directly.")
			a.aiCapability(c, "03", "Keep your source", "Your document stays plain Markdown.")
		})
		ui.Spacer(c)
		ui.Text(c, "Local provider integration is the next milestone.").FontSize(11).LineHeight(1.5).TextColor(t.TextMuted)
	})
}

func (a *App) aiCapability(c *ui.Context, number, title, description string) {
	t := c.Theme()
	ui.Row(c).Gap(10).AlignItems(ui.Start).Children(func() {
		ui.Text(c, number).FontSize(10).Bold().TextColor(t.Accent).Width(20)
		ui.Column(c).Gap(2).Grow(1).Children(func() {
			ui.Text(c, title).FontSize(12).Bold()
			ui.Text(c, description).FontSize(11).LineHeight(1.4).TextColor(t.TextMuted)
		})
	})
}

func (a *App) statusBar(c *ui.Context, f *mdeditor.File) {
	t := c.Theme()
	pos := f.State.CursorPos()
	text := f.State.Buffer.String()
	ui.Row(c).Height(28).Padding(0, 14).Gap(16).AlignItems(ui.Center).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Textf(c, "Ln %d, Col %d", pos.Line+1, pos.Col+1).FontSize(11).TextColor(t.TextMuted)
		ui.Textf(c, "%d characters", utf8.RuneCountInString(text)).FontSize(11).TextColor(t.TextMuted)
		ui.Textf(c, "%d lines", f.State.Buffer.LineCount()).FontSize(11).TextColor(t.TextMuted)
		ui.Spacer(c)
		ui.Text(c, "Open source  ·  Local-first").FontSize(11).TextColor(t.TextMuted)
	})
}
