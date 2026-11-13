package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/xddcode/markgo/internal/writingui"
)

// renderFindBar keeps the editor's native search, selection and undo behavior
// while giving each action a distinct label and fitting the narrow workbench.
func renderFindBar(c *ui.Context, bar ui.Element, find *editor.FindBar) {
	t := c.Theme()
	writingui.Surface(c, bar).Width(480).Gap(8).Padding(12)
	field := func(id, label string, value *string) ui.Element {
		return writingui.Field(c, ui.TextInputBase(c.Key(id), value), false).Grow(1).MinWidth(80).Placeholder(label).Label(label)
	}
	button := func(id, text, label string) ui.Element {
		item := writingui.Interactive(c, ui.ButtonBase(c.Key(id)), false).Height(40).MinWidth(40).Padding(4, 8).Center().KeepFocus().Label(label).Tooltip(label)
		item.Children(func() {
			if icon := icons[id]; icon != nil {
				ui.Icon(c, icon).Size(18, 18)
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
				item := writingui.Interactive(c, ui.ToggleBase(c.Key(id), on), *on).Height(40).MinWidth(40).Padding(4, 8).Center().
					KeepFocus().Label(label).Tooltip(label)
				if *on {
					item.Border(1, t.Border).Description("Enabled")
				}
				item.Children(func() { ui.Text(c, text).FontSize(12) })
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
