package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// renderFindBar keeps the editor's native search, selection and undo behavior
// while giving each action a distinct label and fitting the narrow workbench.
func renderFindBar(c *ui.Context, bar ui.Element, find *editor.FindBar) {
	t := c.Theme()
	bar.Width(460).Gap(8).Padding(12).Background(t.Background).Border(1, t.Border.Alpha(0.7)).
		Radius(12).Shadow(0, 8, 28, 0, ui.RGBA(0, 0, 0, 0.09))
	field := func(id, label string, value *string) ui.Element {
		return ui.TextInputBase(c.Key(id), value).Grow(1).MinWidth(80).Height(40).
			Padding(6, 9).Background(t.Surface).Border(1, t.Border).
			Radius(6).FontSize(13).Placeholder(label).Label(label)
	}
	button := func(id, text, label string) ui.Element {
		item := ui.ButtonBase(c.Key(id)).Height(40).MinWidth(40).Padding(4, 8).Center().
			Radius(6).KeepFocus().Label(label).Tooltip(label)
		if item.Hovered() {
			item.Background(t.SurfaceHover)
		}
		if item.Pressed() {
			item.Background(t.SurfacePressed)
		}
		item.Children(func() {
			if icon := icons[id]; icon != nil {
				ui.Icon(c, icon).Size(17, 17).TextColor(t.TextMuted)
			} else {
				ui.Text(c, text).FontSize(12)
			}
		})
		return item
	}
	bar.Children(func() {
		ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
			find.QueryInput(field("query", "Find text", find.Query()))
			ui.Text(c, find.Status()).FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum").MinWidth(62).Role(ui.RoleStatus)
			if button("previous", "↑", "Previous match").Clicked() {
				find.Previous()
			}
			if button("next", "↓", "Next match").Clicked() {
				find.Next()
			}
			if button("close", "×", "Close find").Clicked() {
				find.Close()
			}
		})
		ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
			toggle := func(id, text, label string, on *bool) {
				item := ui.ToggleBase(c.Key(id), on).Height(40).MinWidth(40).Padding(4, 8).Center().Radius(5).
					KeepFocus().Label(label).Tooltip(label)
				if *on || item.Hovered() {
					item.Background(t.SurfaceHover).TextColor(t.Text)
				}
				if item.Pressed() {
					item.Background(t.SurfacePressed)
				}
				item.Children(func() { ui.Text(c, text).FontSize(11) })
			}
			toggle("case", "Aa", "Match case", find.CaseSensitive())
			toggle("word", "Word", "Match whole word", find.WholeWord())
			toggle("regex", ".*", "Use regular expression", find.Regexp())
			toggle("selection", "Selection", "Find in selection", find.InSelection())
			ui.Spacer(c)
			if button("replacing", "Replace", "Show replace").Disabled(!find.CanReplace()).Clicked() {
				find.SetReplacing(!find.Replacing())
			}
		})
		if find.Replacing() {
			ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
				find.ReplacementInput(field("replacement", "Replace text", find.Replacement()))
				count, _ := find.Matches()
				if button("replace", "Replace", "Replace match").Disabled(count == 0).Clicked() {
					find.Replace()
				}
				if button("replace-all", "All", "Replace all").Disabled(count == 0).Clicked() {
					find.ReplaceAll()
				}
			})
		}
	})
}
