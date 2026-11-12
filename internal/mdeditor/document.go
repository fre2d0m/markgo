package mdeditor

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

type File struct {
	Name  string
	State *editor.State
}

func NewFile(name, source string) *File {
	buffer := editor.NewBuffer(source)
	buffer.SetLanguage(editor.LanguageForText(name, source))
	return &File{
		Name: name,
		State: &editor.State{
			Buffer:      buffer,
			Font:        ui.Font{Size: 15},
			LineHeight:  1.75,
			SoftWrap:    true,
			LineNumbers: editor.LineNumbersOff,
			NoFolding:   true,
			Placeholder: "Start writing in Markdown…",
			Label:       name,
		},
	}
}

func SelectedText(st *editor.State) string {
	for _, selection := range st.Selections() {
		if !selection.Empty() {
			return st.Buffer.Slice(selection.Start(), selection.End())
		}
	}
	return ""
}
