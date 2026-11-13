package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/xddcode/markgo/internal/writingui"
)

// Original 24-unit drawings share one optical weight and round terminals.
// They are native vector paths, parsed once rather than text glyphs or bitmaps.
var icons = map[string]*ui.SVG{
	"undo":            iconPath(`<path d="M9 5 4 10l5 5M4 10h10a6 6 0 0 1 0 12"/>`),
	"recovery":        iconPath(`<path d="M4 8a8 8 0 1 1-1 7M4 3v5h5M12 7v5l3 2"/>`),
	"mark":            iconPath(`<path d="M5 19V5l7 7 7-7v14M5 19h14"/>`),
	"sidebar":         iconPath(`<rect x="3.5" y="4.5" width="17" height="15" rx="2"/><path d="M9 5v14"/>`),
	"search":          iconPath(`<circle cx="10.5" cy="10.5" r="6"/><path d="m15 15 5 5"/>`),
	"plus":            iconPath(`<path d="M12 5v14M5 12h14"/>`),
	"folder":          iconPath(`<path d="M3.5 7.5V6a2 2 0 0 1 2-2H10l2.5 3H19a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5.5a2 2 0 0 1-2-2V7.5Z"/>`),
	"file":            iconPath(`<path d="M14 3.5H6a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9.5L14 3.5ZM14 4v6h6M8 14h8M8 17h5"/>`),
	"save":            iconPath(`<path d="M5 3.5h12l3.5 3.5v12a1.5 1.5 0 0 1-1.5 1.5H5A1.5 1.5 0 0 1 3.5 19V5A1.5 1.5 0 0 1 5 3.5ZM8 4v6h8V4M8 20v-6h8v6"/>`),
	"more":            iconPath(`<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>`),
	"focus":           iconPath(`<path d="M8 4H4v4M16 4h4v4M4 16v4h4M20 16v4h-4"/>`),
	"exit-focus":      iconPath(`<path d="M4 8h4V4M20 8h-4V4M8 20v-4H4M16 20v-4h4"/>`),
	"appearance":      iconPath(`<circle cx="12" cy="12" r="8"/><path d="M12 4v16"/><path d="M12 4a8 8 0 0 1 0 16Z" fill="currentColor" stroke="none"/>`),
	"chevron":         iconPath(`<path d="m8 10 4 4 4-4"/>`),
	"close":           iconPath(`<path d="m6 6 12 12M6 18 18 6"/>`),
	"check":           iconPath(`<path d="m5 12 4.5 4.5L19 7"/>`),
	"refresh":         iconPath(`<path d="M19.5 10A8 8 0 1 0 19 17M19.5 4v6h-6"/>`),
	"bold":            iconPath(`<path d="M7 4h6a4 4 0 0 1 0 8H7V4ZM7 12h7a4 4 0 0 1 0 8H7v-8Z"/>`),
	"italic":          iconPath(`<path d="M10 4h8M6 20h8M14.5 4l-5 16"/>`),
	"strikethrough":   iconPath(`<path d="M17 6c-1.2-1.4-2.8-2-5-2-2.8 0-5 1.5-5 4 0 1.8 1.5 2.8 3.5 3.5M8 18c1.1 1.3 2.5 2 4.6 2 2.7 0 4.6-1.4 4.6-3.6 0-1.1-.5-2-1.6-2.6M4 12h16"/>`),
	"link":            iconPath(`<path d="m9 15 6-6M8 8l2-2a4.5 4.5 0 0 1 6.4 6.4l-1.8 1.8M16 16l-2 2a4.5 4.5 0 0 1-6.4-6.4l1.8-1.8"/>`),
	"code":            iconPath(`<path d="m8 7-5 5 5 5m8-10 5 5-5 5M14 4l-4 16"/>`),
	"list":            iconPath(`<path d="M9 6h11M9 12h11M9 18h11M4 6h.01M4 12h.01M4 18h.01"/>`),
	"numbered-list":   iconPath(`<path d="M10 6h10M10 12h10M10 18h10M3 4l2-1v6M3 9h4M3 15c0-3 4-3 4-1 0 2-4 3-4 6h4"/>`),
	"task-list":       iconPath(`<rect x="3" y="4" width="6" height="6" rx="1.4"/><path d="m4.5 7 1 1 2-2M13 7h8M13 17h8"/><rect x="3" y="14" width="6" height="6" rx="1.4"/>`),
	"quote":           iconPath(`<path d="M4 12h6v6H4v-6c0-5 1-6 5-6M14 12h6v6h-6v-6c0-5 1-6 5-6"/>`),
	"read":            iconPath(`<path d="M12 6c-3-2-6-2-9-1v14c3-1 6-1 9 1 3-2 6-2 9-1V5c-3-1-6-1-9 1ZM12 6v14"/>`),
	"edit":            iconPath(`<path d="m4 15 12-12 5 5L9 20l-6 1 1-6ZM13 6l5 5"/>`),
	"format":          iconPath(`<path d="M3 19 9 4l6 15M5 14h8M16 10h6M19 10v9"/>`),
	"minus":           iconPath(`<path d="M5 12h14"/>`),
	"previous":        iconPath(`<path d="m6 10 6-6 6 6M12 4v16"/>`),
	"next":            iconPath(`<path d="m6 14 6 6 6-6M12 20V4"/>`),
	"window-minimize": iconPath(`<path d="M3 12h18"/>`),
	"window-maximize": iconPath(`<rect x="3.5" y="3.5" width="17" height="17" rx="1"/>`),
	"window-restore":  iconPath(`<path d="M8 7V3.5h12.5V16H17"/><rect x="3.5" y="8" width="12.5" height="12.5" rx="1"/>`),
	"window-close":    iconPath(`<path d="m4 4 16 16M4 20 20 4"/>`),
}

var menuIcons = func() map[string]*mygo.MenuIcon {
	result := make(map[string]*mygo.MenuIcon)
	for _, name := range []string{"plus", "file", "folder", "save"} {
		icon, err := mygo.NewMenuIconSVG(icons[name], true)
		if err != nil {
			panic(err)
		}
		result[name] = icon
	}
	return result
}()

func iconPath(path string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round">` + path + `</svg>`))
}

func iconButton(c *ui.Context, key, icon, label string) ui.Element {
	return stateIconButton(c, key, icon, label, false)
}

func stateIconButton(c *ui.Context, key, icon, label string, active bool) ui.Element {
	b := writingui.Interactive(c, ui.ButtonBase(c.Key(key)), active).Size(40, 40).Center().KeepFocus().
		Label(label).Tooltip(label).Cursor(ui.CursorPointer)
	b.Children(func() { ui.Icon(c, icons[icon]).Size(18, 18) })
	return b
}
