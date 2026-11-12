package app

import "github.com/xddcode/markgo/internal/mdeditor"

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

## Keep the source truthful

The editor always stores the original Markdown. Preview changes how text appears, never what the file contains.

## Make every interaction count

1. Keep the writing surface quiet.
2. Reveal complexity only when it is needed.
3. Let every action be undone.

### Shortcuts

Use the editor's standard shortcuts for selection, find and replace, multiple carets, undo and redo.
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

Code fences, links and Markdown punctuation remain in your file even when live preview hides their markers. Use Source to inspect the original text.
`
