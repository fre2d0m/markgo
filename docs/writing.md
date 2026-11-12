# Writing with MarkGo

## Documents and folders

Choose **Open** to open a UTF-8 Markdown or text file. **New** creates an untitled document. **Folder** shows Markdown files in a local folder and its subfolders. Filtering reveals matching files even inside collapsed folders; clearing the filter restores the previous expansion state. Use the refresh button after changing the folder outside MarkGo.

Packaged builds register Markdown file types. Files sent by the operating system or dropped on the native window are queued while another file operation is running. These OS integrations require verification on your target platform.

The sidebar's **Outline** tab lists headings in the current document. Choose a heading to move the editing caret there. Code-fence contents and YAML front matter are excluded from the outline.

## Writing and reading

Live preview keeps a single editable Markdown buffer and hides supported syntax when the caret is outside the formatted element. Moving into it reveals the original marks. **Source** shows all marks; **Preview** returns to live preview. Task checkboxes update the Markdown source and can be undone.

**Read** displays native Markdown typography, tables, code blocks, and local images. Choose **Edit** to change the source. Tables and images are not directly editable in the reading view. Formulas remain TeX text. Remote images are not automatically fetched; local image paths are resolved relative to the saved document.

The formatting toolbar works on one selection or the current line. Ordinary bullet and numbered lists, tasks, and quotes continue with Return. A second Return on an empty item exits that item. Four-space and tab-indented lines keep normal editor behavior, as do fenced code, comments, and multiple selections. Shift+Return always uses the editor's ordinary newline action.

**Find** opens document search. Its options include case matching, whole words, regular expressions, and selection scope. Choose **Replace** to show the replacement field. Replacing all matches is one undoable edit. Escape closes search and restores writing focus.

## Saving

**Save** updates an opened file. New documents prompt for a location. **Save as** creates a separate file; choose an unused filename because it does not overwrite an existing destination. Closing a modified document offers save, discard, or cancel. Quitting offers to save all modified documents in sequence. Canceling or failing a save leaves the application open.

MarkGo checks the existing file before replacing it. If it changed outside the application, save is stopped and the editor keeps your changes. The warning offers **Reload** or **Keep editing**; keeping the buffer does not grant permission to overwrite the changed file. Save a copy if you want to retain both versions. Reload asks before discarding edits and rejects its result if the buffer changed while the file was being read.

Saving preserves ordinary LF or CRLF line endings. An unchanged file with mixed endings retains its exact bytes. After editing mixed endings, **Save as** asks before creating a copy normalized to CRLF. The original file remains untouched.

Saving is manual. There is no automatic save or crash recovery. New-file creation requires hard-link support on the destination filesystem. Symbolic links and non-UTF-8 files are not supported.

## Keyboard shortcuts

Use Command on macOS and Control on Windows/Linux for the shortcuts below.

| Action | Shortcut |
| --- | --- |
| New document | Cmd/Ctrl+N |
| Open document | Cmd/Ctrl+O |
| Save | Cmd/Ctrl+S |
| Save as | Cmd/Ctrl+Shift+S |
| Close document | Cmd/Ctrl+W |
| Find | Cmd/Ctrl+F |
| Bold / italic | Cmd/Ctrl+B / I |
| Heading 1 / 2 | Cmd/Ctrl+1 / 2 |
| Undo | Cmd/Ctrl+Z |
| Focus mode | Cmd/Ctrl+Shift+F |

The editor also supplies platform-native selection, clipboard, redo, multi-caret, and navigation shortcuts.
