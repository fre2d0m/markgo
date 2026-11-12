package app

import "github.com/xddcode/markgo/internal/mdeditor"

func sampleDocuments() []*mdeditor.File {
	return []*mdeditor.File{
		mdeditor.NewFile("Welcome.md", welcomeDocument),
		mdeditor.NewFile("Writing Guide.md", writingGuideDocument),
		mdeditor.NewFile("AI Notes.md", aiNotesDocument),
	}
}

const welcomeDocument = `# Welcome to MarkGo

A **free and open-source** Markdown editor for focused writing and AI-assisted thinking.

## A calm place to write

MarkGo keeps your words in plain Markdown files. There is no account, no proprietary document format, and no required cloud service.

- [x] Native interface drawn by MyGo
- [x] Real editing with undo, search and multiple carets
- [x] Live Markdown preview
- [ ] Local AI provider integration

> The best writing tool gets out of the way until you need it.

## Try the editor

Write **bold**, *italic*, or ` + "`inline code`" + ` text. Move the caret to a formatted line to reveal its source. Use the sidebar to switch documents.

` + "```go\nfunc main() {\n\tfmt.Println(\"Hello, MarkGo!\")\n}\n```" + `

[Explore MyGo](https://github.com/egoist/mygo) and see what powers the native interface.
`

const writingGuideDocument = `# Writing Guide

## Keep the source truthful

The editor always stores the original Markdown. Preview changes how text appears, never what the file contains.

## Make every interaction count

1. Keep the writing surface quiet.
2. Reveal complexity only when it is needed.
3. Let every action be undone.

### Shortcuts

Use the editor's standard shortcuts for selection, find and replace, multiple carets, undo and redo.
`

const aiNotesDocument = `# AI Notes

MarkGo treats AI as a **careful collaborator**, not an invisible replacement for the writer.

## The editing contract

- Read only the context the user chooses.
- Show a diff before changing the source.
- Keep every accepted edit undoable.
- Support local models without requiring an account.

> AI can help shape the words. The writer keeps the final say.
`
