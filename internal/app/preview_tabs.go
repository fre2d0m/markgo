package app

import (
	"fmt"
	"slices"

	"github.com/egoist/mygo/ui"
	"github.com/xddcode/markgo/internal/mdeditor"
	"github.com/xddcode/markgo/internal/writingui"
)

func (p *TypesettingPreview) ensureOpen(index int) {
	if !slices.Contains(p.openPages, index) {
		p.openPages = append(p.openPages, index)
	}
}

func (p *TypesettingPreview) selectPage(index int, writing bool) {
	if index < 0 || index >= len(p.pages) {
		return
	}
	if p.selected != index {
		if !p.sourceMode {
			p.current().typesetter.ToSource()
			p.current().typesetter.CancelInlineIntent()
		}
		p.selected = index
		if !p.sourceMode {
			p.current().typesetter.FromSource()
		}
		p.shell.appearanceOpen, p.formatOpen = false, false
	}
	p.ensureOpen(index)
	if writing {
		p.current().file.Focus.Focus()
	}
}

func (p *TypesettingPreview) newPage() {
	p.nextPage++
	f := mdeditor.NewFile(fmt.Sprintf("Untitled %d.md", p.nextPage), "")
	f.ID = fmt.Sprintf("preview-new-%d", p.nextPage)
	p.pages = append(p.pages, &previewPage{file: f, typesetter: mdeditor.NewTypesetter(f.State), description: "Room for your own words"})
	p.selectPage(len(p.pages)-1, true)
}

// Closing a sample's tab does not delete its edits. It remains in Sample pages
// and can be reopened, while the ordinary app uses its guarded close lifecycle.
func (p *TypesettingPreview) closePage(file *mdeditor.File) {
	for i, index := range p.openPages {
		if p.pages[index].file != file {
			continue
		}
		p.openPages = slices.Delete(p.openPages, i, i+1)
		if len(p.openPages) == 0 {
			p.newPage()
		} else if p.selected == index {
			p.selectPage(p.openPages[min(i, len(p.openPages)-1)], true)
		}
		return
	}
}

func (p *TypesettingPreview) cyclePage(direction int) {
	index := slices.Index(p.openPages, p.selected)
	if index >= 0 {
		p.selectPage(p.openPages[(index+direction+len(p.openPages))%len(p.openPages)], true)
	}
}

func (p *TypesettingPreview) header(c *ui.Context) {
	a := p.shell
	ui.Row(c.Key("preview-titlebar")).FillWidth().Height(titleBarHeight).DragWindow().AlignItems(ui.Center).Children(func() {
		ui.Row(c).Width(sidebarWidth).Shrink(0).FillHeight().Padding(0, 18).Gap(10).AlignItems(ui.Center).Children(func() {
			brand(c)
			ui.Spacer(c)
			iconButton(c, "sidebar", "sidebar", "Toggle sample pages").OnClick(func() { p.sidebar = !p.sidebar })
		})
		files := make([]*mdeditor.File, len(p.openPages))
		for i, index := range p.openPages {
			files[i] = p.pages[index].file
		}
		documentTabs(c, &p.tabs, files, p.current().file.ID, false, func(i int, writing bool) { p.selectPage(p.openPages[i], writing) }, p.closePage, p.newPage, func(menu *ui.Menu, file *mdeditor.File) {
			if writingui.MenuItem(menu, "Close sample tab").Chosen() {
				p.closePage(file)
			}
		})
		appearance := stateIconButton(c, "appearance", "appearance", "Appearance", a.appearanceOpen).MarginStart(8)
		if appearance.Clicked() {
			a.appearanceOpen = !a.appearanceOpen
			p.formatOpen = false
		}
		a.appearancePopover(c, appearance, &p.current().file.Focus)
		a.captionButtons(c)
	})
}
