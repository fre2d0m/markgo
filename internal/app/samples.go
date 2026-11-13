package app

import "github.com/xddcode/markgo/internal/mdeditor"

// Guides are opt-in documents. Opening them again keeps any edits the writer
// made to their existing copies.
func (a *App) openGuides() {
	for _, file := range sampleDocuments() {
		found := false
		for _, current := range a.files {
			if current.ID == file.ID {
				found = true
				break
			}
		}
		if !found {
			a.files = append(a.files, file)
		}
	}
	a.selected = "Welcome.md"
	a.reading = true
}

func sampleDocuments() []*mdeditor.File {
	return []*mdeditor.File{
		mdeditor.NewFile("Welcome.md", welcomeDocument),
		mdeditor.NewFile("Writing Guide.md", writingGuideDocument),
		mdeditor.NewFile("Markdown Reference.md", markdownReferenceDocument),
	}
}

const welcomeDocument = `# Welcome to MarkGo

A **free and open-source** Markdown editor for focused writing and clear thinking.

## A calm place to write

MarkGo keeps your words in plain Markdown files. There is no account, no proprietary document format, and no required cloud service.

- [x] Native interface drawn by MyGo
- [x] Real editing with undo, search and multiple carets
- [x] Live Markdown preview
- [ ] Write your first note

> The best writing tool gets out of the way until you need it.

## Try the editor

Write **bold**, *italic*, or ` + "`inline code`" + ` text. Move the caret into formatted text to reveal its source. Use the sidebar to switch documents.

` + "```go\nfunc main() {\n\tfmt.Println(\"Hello, MarkGo!\")\n}\n```" + `

[Explore MyGo](https://github.com/egoist/mygo) and see what powers the native interface.
`

const writingGuideDocument = `# Writing Guide

## Make room for your words

Choose the half-filled circle in the header to open Appearance. Paper, Warm paper and Graphite change the whole interface. Adjust the text size, type and page width until the page feels comfortable. Your choices are remembered on this device.

## Start a document

Use the plus beside Files, or press Cmd/Ctrl+N. Open an existing document from the three-dot document menu, or press Cmd/Ctrl+O. Open a folder from the foot of the sidebar.

Select a phrase and use Formatting for bold, italic, a link or inline code. H1 and H2 change the current heading. These actions can be undone with Cmd/Ctrl+Z.

## Keep writing

Return continues a list or quote; another Return on an empty item ends it. Find opens search and replacement. The Outline tab takes you to a heading. Focus mode gives the page the whole window.

## Save your work

Press Cmd/Ctrl+S to save. A new document asks for a location. The footer tells you whether changes are unsaved. Saving your Markdown file is manual. Local recovery copies protect completed checkpoints; choose Keep drafts when closing to continue later.

## Read and edit

The book icon opens a formatted reading view. The pencil returns to editing. Tables and local images are formatted in reading mode; edit their Markdown in the writing view. The document menu can show all Markdown source marks.

### Shortcuts

- Appearance: Cmd/Ctrl+comma
- Toggle sidebar: Cmd/Ctrl+backslash
- Focus mode: Cmd/Ctrl+Shift+F
- Find: Cmd/Ctrl+F
- Bold and italic: Cmd/Ctrl+B and Cmd/Ctrl+I
`

const markdownReferenceDocument = `# Markdown Reference

## Lists and tasks

- Press Return to continue a list.
- Press Return again on an empty item to end it.
- [ ] New task items start unchecked.

## Tables

Use Read to view a formatted table, and Edit to change its Markdown source.

| Element | Markdown |
| --- | --- |
| Heading | # Heading |
| Bold | **bold** |
| Emphasis | *italic* |

## Local images

Images use paths relative to the saved document. Read mode displays local images; remote images are not downloaded automatically.

## Source fidelity

Code fences, links and Markdown punctuation remain in your file even when live preview hides their markers. Use Show Markdown source in the document menu to inspect the original text.
`
