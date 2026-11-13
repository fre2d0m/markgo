package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/xddcode/markgo/internal/mdeditor"
	"github.com/xddcode/markgo/internal/storage"
)

type analysisCache struct {
	version  uint64
	valid    bool
	running  bool
	analysis mdeditor.Analysis
}

func (a *App) analysisFor(file *mdeditor.File) (mdeditor.Analysis, bool) {
	cache := a.analyses[file.ID]
	if cache == nil {
		cache = &analysisCache{}
		a.analyses[file.ID] = cache
	}
	snapshot := file.State.Buffer.Snapshot()
	if cache.valid && cache.version == snapshot.Version() {
		return cache.analysis, true
	}
	if cache.running {
		return cache.analysis, false
	}
	if a.window == nil { // Headless UI tests have no event-loop window.
		cache.analysis = mdeditor.Analyze(snapshot.String())
		cache.version, cache.valid = snapshot.Version(), true
		return cache.analysis, true
	}
	cache.running = true
	window := a.window
	go func() {
		analysis := mdeditor.Analyze(snapshot.String())
		window.Update(func() {
			cache.running = false
			if file.State.Buffer.Version() == snapshot.Version() {
				cache.analysis = analysis
				cache.version, cache.valid = snapshot.Version(), true
			} else {
				cache.valid = false
			}
		})
	}()
	return cache.analysis, false
}

func (a *App) outlineSidebar(c *ui.Context, file *mdeditor.File) {
	t := c.Theme()
	analysis, ready := a.analysisFor(file)
	if !ready && len(analysis.Headings) == 0 {
		ui.Text(c, "Updating outline…").FontSize(12).TextColor(t.TextMuted).Padding(10, 4)
		return
	}
	if len(analysis.Headings) == 0 {
		ui.Text(c, "Headings in this document will appear here.").FontSize(12).LineHeight(1.5).TextColor(t.TextMuted).Padding(10, 4)
		return
	}
	ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).FillWidth().Gap(2).Children(func() {
			active := -1
			if !a.reading && ready {
				for index, heading := range analysis.Headings {
					if heading.Offset <= file.State.Cursor() {
						active = index
					}
				}
			}
			for index, heading := range analysis.Headings {
				heading := heading
				item := ui.ButtonBase(c.Key(heading.Offset)).FillWidth().MinHeight(40).
					Padding(6, 8).PaddingStart(float32(8 + min(heading.Level-1, 4)*12)).
					Radius(t.Radius).Disabled(!ready).Label("Go to " + heading.Title)
				background := ui.Transparent
				if item.Hovered() || index == active {
					background = t.SurfaceHover
				}
				item.Background(item.AnimateColor("active-section", background, ui.Animation{Duration: 120 * time.Millisecond, NonMotion: true}))
				item.Children(func() {
					weight := 400
					if index == active {
						weight = 500
					}
					ui.Text(c, heading.Title).FillWidth().FontSize(12).FontWeight(weight).SingleLine().Ellipsis("…").TextColor(t.Text)
				})
				if item.Clicked() {
					if a.reading {
						file.Reader.Anchor = markdown.Slug(heading.Title)
						file.Reader.AnchorVersion++
					} else {
						file.State.GoTo(file.State.Buffer.Pos(heading.Offset))
						file.Focus.Focus(c)
					}
				}
			}
		})
	})
}

func (a *App) openFolderDialog() {
	if a.window == nil || a.fileBusy {
		return
	}
	a.fileBusy = true
	window := a.window
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: window, Title: "Open folder", Directory: true})
		if err != nil || len(paths) == 0 {
			window.Update(func() {
				a.fileBusy = false
				if err != nil {
					a.notice = "Could not open folder: " + err.Error()
				}
			})
			return
		}
		folder, err := storage.ListFolder(paths[0])
		window.Update(func() {
			a.fileBusy = false
			if err != nil {
				a.notice = "Could not open folder: " + err.Error()
				return
			}
			a.folder = &folder
			a.expanded = make(map[string]*bool)
			a.notice = "Opened folder " + folder.Name
		})
	}()
}

func (a *App) refreshFolder() {
	if a.window == nil || a.fileBusy || a.folder == nil {
		return
	}
	a.fileBusy = true
	path := a.folder.Path
	window := a.window
	go func() {
		folder, err := storage.ListFolder(path)
		window.Update(func() {
			a.fileBusy = false
			if err != nil {
				a.notice = "Could not refresh folder: " + err.Error()
				return
			}
			if a.folder != nil && a.folder.Path == path {
				a.folder = &folder
				a.notice = "Folder refreshed."
			}
		})
	}()
}

func (a *App) folderItems(c *ui.Context, entries []storage.Entry) {
	query := strings.ToLower(strings.TrimSpace(a.query))
	for _, entry := range entries {
		entry := entry
		if query != "" && !folderMatches(entry, query) {
			continue
		}
		if entry.Directory {
			open := a.expanded[entry.Path]
			if open == nil {
				open = new(bool)
				a.expanded[entry.Path] = open
			}
			if query != "" {
				// Filtering exposes matching descendants without changing the
				// user's saved expansion state when the filter is cleared.
				ui.Text(c.Key(entry.Path+"-filter"), entry.Name).FontSize(12).Bold().Padding(6, 8)
				ui.Column(c.Key(entry.Path)).PaddingStart(12).Children(func() {
					a.folderItems(c, entry.Children)
				})
				continue
			}
			if ui.TreeItem(c.Key(entry.Path), entry.Name, open, func() {
				a.folderItems(c, entry.Children)
			}).Height(40).FontSize(12).Clicked() {
				*open = !*open
			}
		} else if ui.TreeItem(c.Key(entry.Path), entry.Name, nil, nil).Height(40).FontSize(12).
			Selected(a.current().Path == entry.Path).Disabled(a.fileBusy).Clicked() {
			a.openPath(entry.Path)
		}
	}
}

func folderMatches(entry storage.Entry, query string) bool {
	if strings.Contains(strings.ToLower(entry.Name), query) {
		return true
	}
	for _, child := range entry.Children {
		if folderMatches(child, query) {
			return true
		}
	}
	return false
}
