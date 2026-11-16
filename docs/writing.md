# Writing with MarkGo

## Documents and folders

MarkGo restores your previous documents and unfinished drafts. The first launch starts on a blank, focused document. Begin typing, or open the **Document menu** (the MarkGo icon and name at the top left) and choose **Open document…**. The plus after the document tabs or beside Files creates another document. **Open a folder** at the foot of the sidebar shows Markdown files in a local folder and its subfolders. Built-in examples are available through **Document menu → Writing guide**. Filtering reveals matching files even inside collapsed folders; clearing the filter restores the previous expansion state. Use the refresh button after changing the folder outside MarkGo.

Each open document has a tab across the top. Switching tabs keeps each document's caret, selection, editing scroll and undo history. The dot marks unsaved changes; hovering the close target reveals its cross. Hover a title to see its full path. When many tabs are open, scroll horizontally or use **All open documents** beside the plus. The menu distinguishes identical filenames by their folder paths. **Control+Tab** and **Control+Shift+Tab** cycle documents on every platform. The selected tab stays visible after switching or resizing. Closing a tab follows the same unsaved-change protection as closing from the sidebar.

In the open-document list, use the arrow keys, Home, and End to browse without losing list focus. Press Enter to begin writing in the selected document.

The current or hovered document shows a close button. Right-click a document to save it, copy its path, or close it. These actions address that document; closing unsaved work keeps the same confirmation behavior as the document menu.

Packaged builds register Markdown file types. Files sent by the operating system or dropped on the native window are queued while another file operation is running. These OS integrations require verification on your target platform.

The sidebar's **Outline** tab lists headings in the current document. Choose a heading to move the editing caret there, or jump the page there in Read mode. While editing, the current section follows the caret. Code-fence contents and YAML front matter are excluded from the outline.

Read-mode anchors currently choose the first matching title when headings repeat; editing-mode outline navigation uses exact source positions.

## Writing and reading

Documents open in **Visual editing**. Headings, paragraphs, ordinary lists and tasks, simple quotations, fenced code, bold/italic/strikethrough, inline code and links are editable in their formatted presentation. Moving the caret into supported text keeps that presentation. Every change updates the same Markdown source and undo history.

The book-shaped **View options** button, or **Document menu → View**, offers **Visual editing**, **Markdown source** and **Reading view**. Cmd/Ctrl+Shift+M switches visual editing and source. Reading is an optional read-only page that also displays tables and local images; **Return to writing** resumes editing. Images, tables, front matter and more complex Markdown remain literal source in visual editing. Formulas remain TeX text. Remote images are not automatically fetched; local image paths resolve relative to the saved document.

Type **/** in an empty ordinary paragraph to insert a paragraph, heading 1–3, bullet list, numbered list, task list, quote or code block. Continue typing a Chinese or English keyword, such as **标题**, **待办**, **h2** or **code**, to filter. Up/Down chooses a result, Enter or Tab applies it, and Escape dismisses the menu while retaining your input. One Undo restores the complete slash query. Pasting a path or typing inside code does not open the menu. A quiet hint appears only on the focused empty paragraph.

The page toolbar keeps bold, italic, inline code and links within reach. **Formatting** at its right opens the block styles. Formatting returns focus to writing; the panel closes after an action. Ordinary lists, tasks and quotes continue with Return; Return in an empty item exits one level. Tab/Shift+Tab changes list or quote nesting. Fenced code retains indentation and has language, Copy and Continue controls. Inline commands work within a physical paragraph; unsupported or ambiguous selections produce an explanation without changing the document.

**Find** searches the Markdown source while keeping visual editing open. Its panel sits above the page and the current visible match stays highlighted while you type the query. Its options include case matching, whole words, regular expressions, and selection scope. Select a passage before opening Find to limit the scope; the scope follows subsequent edits. If a change crosses its boundaries, select the passage again and reopen Find. Choose **Replace** to show the replacement field. Replacing all matches is one undoable edit. Escape closes search and restores writing focus. Source syntax is searchable even where its marks are hidden in the formatted page.

## Appearance and focus

The half-filled circle opens **Appearance**. The gallery offers eight light themes—**Paper**, **Warm paper**, **Sakura**, **Sage**, **Mist**, **Lavender**, **Sand** and **Pearl**—and four dark themes: **Graphite**, **Midnight**, **Pine** and **Plum**. A check marks the active choice. **Follow system appearance** chooses Paper by day-mode settings and Graphite by dark-mode settings. These themes cover the whole interface.

Adjust text size, choose **Modern** (system type) or **Book** (serif), and select a Narrow, Comfort, or Wide page. Changes apply immediately and are remembered locally. Switching a color theme preserves the caret, selection, scroll, and undo history. Typography changes reflow the page without changing the Markdown. Tab moves through the appearance controls; Space or Enter chooses a theme. The close button and Escape return to writing.

Escape closes the panel, and clicking outside returns to the page. The **Focus mode** icon hides navigation and status; **Exit focus mode** restores the workspace. The sidebar icon toggles file navigation independently.

Sidebar and tab transitions can be interrupted and follow the operating system's reduced-motion preference. The page is never faded or scaled. File and text context menus share the chosen theme.

The title bar is drawn as part of the interface. Drag its empty space to move the window; double-click that space to toggle maximization. Document tabs and tools take clicks without dragging the window. Minimize, maximize/restore, and close remain at the top right, including in Focus mode. Closing the window keeps the normal unsaved-document protection.

## Saving

**Save** updates an opened file. New documents prompt for a location. **Save as** creates a separate file; choose an unused filename because it does not overwrite an existing destination. Closing a modified document offers save, discard, or cancel. Quitting offers **Keep drafts**, **Save all**, **Discard**, or **Cancel**. Keep drafts remembers unfinished writing on this device without changing the original files. Save all saves modified documents in sequence. Canceling or failing a save leaves the application open.

MarkGo checks the existing file before replacing it. If it changed outside the application, save is stopped and the editor keeps your changes. The warning offers **Reload** or **Keep editing**; keeping the buffer does not grant permission to overwrite the changed file. Save a copy if you want to retain both versions. Reload asks before discarding edits and rejects its result if the buffer changed while the file was being read.

Saving preserves ordinary LF or CRLF line endings. An unchanged file with mixed endings retains its exact bytes. After editing mixed endings, **Save as** asks before creating a copy normalized to CRLF. The original file remains untouched.

Saving to your Markdown files is manual. Local recovery copies are managed separately, as described below. New-file creation requires hard-link support on the destination filesystem. Symbolic links and non-UTF-8 files are not supported.

The footer briefly confirms **Saved just now** only after the current version finishes saving. A newer edit takes precedence and stays marked **Unsaved changes**.

## Recovery and recent documents

MarkGo keeps a separate recovery copy in `session.json` in the application's local data directory. It contains unfinished text and original disk bytes, along with document paths, the active document, editor positions, the open folder and view modes. Nothing is uploaded. This file is ordinary local data, not an encrypted vault or a version-history archive. Development builds use MyGo's separate development data directory.

Checkpoints are scheduled 750 ms after changes pause, or after five seconds of continuous changes. Disk speed and an in-progress write can delay completion. **Draft backed up** confirms a completed checkpoint for the current buffer; it does not mean the Markdown file has been saved. A sudden interruption can lose edits made after the latest completed checkpoint. The retained checkpoint has a 64 MiB budget, including original text, and supports up to 100 open documents. Exceeding a limit or failing to write shows a persistent warning; existing recovery data is not silently truncated.

On relaunch, **Recovered draft** marks unfinished work. Review it and choose **Save** when ready. If the original file changed or became unavailable, choose **Save a copy** to keep the recovered text without replacing another version. Clean documents reload their current disk contents. The active document, view choice and caret/selection are restored. Undo history and exact visual/reading scroll positions are not restored across relaunches; tab switches within the running app retain the visual viewport.

When closing the application, **Keep drafts** writes the current work before exiting, ready for the next launch. **Discard** removes the unsaved recovery content and reopens named files from disk next time; unnamed discarded drafts are not retained. Cancel leaves the writing session intact. If the final recovery write fails, **Stay open** is the default. **Exit anyway** leaves the older checkpoint in place and can lose newer text or bring back previously discarded text. An unreadable or unsupported recovery file is preserved untouched and automatic recovery writes remain disabled.

Choose **Document menu → Recent documents…** to search up to 20 recent paths by filename or folder. Enter opens the first match; Tab reaches individual results and actions. Escape returns to writing. Unavailable files stay visible so you can retry or remove that history entry. **Clear history** removes the list, not files, open documents or drafts. This is not secure erasure of document contents. The list is remembered with the next completed checkpoint or normal exit.

## Keyboard shortcuts

Use Command on macOS and Control on Windows/Linux for the shortcuts below.

| Action | Shortcut |
| --- | --- |
| New document | Cmd/Ctrl+N |
| Open document | Cmd/Ctrl+O |
| Save | Cmd/Ctrl+S |
| Save as | Cmd/Ctrl+Shift+S |
| Close document | Cmd/Ctrl+W |
| Next / previous document | Control+Tab / Control+Shift+Tab on all platforms |
| Find | Cmd/Ctrl+F |
| Bold / italic | Cmd/Ctrl+B / I |
| Paragraph / heading 1–3 | Cmd/Ctrl+0 / 1 / 2 / 3 in visual editing |
| Block menu | / in an empty ordinary paragraph |
| Visual editing / source | Cmd/Ctrl+Shift+M |
| Bullets / numbers / tasks | Cmd/Ctrl+Shift+8 / 7 / 9 in visual editing |
| Link | Cmd/Ctrl+K in visual editing |
| Undo | Cmd/Ctrl+Z |
| Focus mode | Cmd/Ctrl+Shift+F |
| Appearance | Cmd/Ctrl+, |
| Toggle sidebar | Cmd/Ctrl+Backslash |

The editor also supplies platform-native selection, clipboard, redo and navigation shortcuts. Multiple carets are available in Markdown source; visual editing uses one continuous selection.
