# Writing with MarkGo

## Documents and folders

MarkGo starts on a blank, focused document. Begin typing, or open the **Document menu** (the three dots beside the wordmark) and choose **Open document…**. The plus beside Files creates another document. **Open a folder** at the foot of the sidebar shows Markdown files in a local folder and its subfolders. Built-in examples are available through **Document menu → Writing guide**. Filtering reveals matching files even inside collapsed folders; clearing the filter restores the previous expansion state. Use the refresh button after changing the folder outside MarkGo.

Packaged builds register Markdown file types. Files sent by the operating system or dropped on the native window are queued while another file operation is running. These OS integrations require verification on your target platform.

The sidebar's **Outline** tab lists headings in the current document. Choose a heading to move the editing caret there. Code-fence contents and YAML front matter are excluded from the outline.

## Writing and reading

Live preview keeps a single editable Markdown buffer and hides supported syntax when the caret is outside the formatted element. Moving into it reveals the original marks. **Document menu → Show Markdown source** toggles all marks. Task checkboxes update the Markdown source and can be undone.

**Read** displays native Markdown typography, tables, code blocks, and local images. Choose **Edit** to change the source. Tables and images are not directly editable in the reading view. Formulas remain TeX text. Remote images are not automatically fetched; local image paths are resolved relative to the saved document.

The **Formatting** icon opens a small toolbar above the page. It works on one selection or the current line, closes after an action, and returns focus to writing. Ordinary bullet and numbered lists, tasks, and quotes continue with Return. A second Return on an empty item exits that item. Four-space and tab-indented lines keep normal editor behavior, as do fenced code, comments, and multiple selections. Shift+Return always uses the editor's ordinary newline action.

**Find** opens document search. Its options include case matching, whole words, regular expressions, and selection scope. Choose **Replace** to show the replacement field. Replacing all matches is one undoable edit. Escape closes search and restores writing focus.

## Appearance and focus

The half-filled circle opens **Appearance**. Choose **Paper**, **Warm paper**, or **Graphite**, or enable **Follow system appearance**. These themes cover the whole interface. Adjust text size, choose **Modern** (system type) or **Book** (serif), and select a Narrow, Comfort, or Wide page. Changes apply immediately and are remembered locally. Switching a color theme preserves the caret, selection, scroll, and undo history. Typography changes reflow the page without changing the Markdown.

Escape closes the panel, and clicking outside returns to the page. The **Focus mode** icon hides navigation and status; **Exit focus mode** restores the workspace. The sidebar icon toggles file navigation independently.

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
| Appearance | Cmd/Ctrl+, |
| Toggle sidebar | Cmd/Ctrl+Backslash |

The editor also supplies platform-native selection, clipboard, redo, multi-caret, and navigation shortcuts.
