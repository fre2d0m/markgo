# Writing with MarkGo

## Documents

Cmd/Ctrl+O opens files; the plus creates a document. Tabs retain separate caret, scroll and undo. A dot marks unsaved changes. Hover titles for paths, scroll overflowing tabs or use the chevron to list all documents. Ctrl+Tab / Ctrl+Shift+Tab cycle tabs. Right-click a tab to save, copy its path, reload or close it.

Open a folder at the sidebar foot. Folder rows collapse; filtering reveals matching paths and Refresh rereads changes. Traversal stops at depth 8 or 2,000 entries. Outline navigates headings, including in reading view. View → Recent opens remembered paths; clearing history deletes no files or drafts.

The application accepts command-line paths, OS file requests and dropped files/folders. Packaged association and OS dispatch require verification on your platform. Preview ignores real files.

## Writing

Type ordinary Markdown. Top-level ATX headings, bold/italic/strike/code/links appear formatted; lists/tasks have hanging indents, quotes a quiet rule and top-level fences a code header. Untouched source bytes retain their marker spelling and line endings. Tables, images, metadata and unsupported complex structures remain literal in direct editing.

Type `/` in an empty paragraph for formats. Chinese/English search, arrows, Enter/Tab and pointer choice are supported. Escape retains literal input; pasted paths/code and unfinished composition do not trigger it.

Enter continues list/task/quote structure; empty entries exit or reduce nesting. Tab/Shift+Tab indents/outdents. Task controls or Cmd/Ctrl+Enter toggle completion. Code tools set language, copy body or continue below. Literal closing fences entered into code extend paired delimiters.

Use toolbar formatting or Cmd/Ctrl+B/I/E for bold/italic/code, Cmd/Ctrl+Shift+X for strike and Cmd/Ctrl+K for links. Link errors preserve input; Apply/Cancel returns focus. Editing a reference link changes that occurrence only. Undo/redo: Cmd/Ctrl+Z and Cmd/Ctrl+Shift+Z.

View offers source and read-only reading, without a permanent Write/Read switch. Reading renders tables and local images relative to the saved document. Remote images are blocked; HTTP(S) links open only on explicit click. Formulas remain text.

## Find and replace

Cmd/Ctrl+F searches canonical Markdown. Options: case, whole words, regex and selection scope. Select a passage before opening Find for scope. Invalid patterns show inline errors. Replace-all is one undo; regex replacements support `$1` captures. Invalidated scope must be selected again and never expands silently. Reading disables replacement.

## Save and recovery

Cmd/Ctrl+S saves; Cmd/Ctrl+Shift+S saves a new copy. Save-as never silently overwrites an existing destination. Changed original bytes/permissions block saving; save a copy or reload from the tab menu. Reload confirms dirty content, rejects stale results and is undoable.

Recovery checkpoints are separate local data. “Draft backed up” differs from “Saved locally”. Dirty drafts restore for review; changed/missing originals require a copy. Exit offers keep drafts, save all, discard or cancel and waits for its checkpoint. Cancelling Save as cancels save-and-exit. Write failure preserves buffers and keeps the app open.

Recovery holds at most 100 documents/64 MiB and 20 recent paths. It is unencrypted local data, not an archive; undo history is not persisted. Force-kill/power loss can lose the latest interval; OS shutdown is best effort within a platform timeout. Save important writing in named files. Locations are in [setup](development/setup.md).

## Appearance

Twelve paper/ink themes include Sakura. System appearance follows light/dark. Choose system/serif prose, 14–24 DIP size and three widths. Interface controls retain their own scale. Focus hides the sidebar while preserving window controls. Presentation changes leave Markdown/history intact.
