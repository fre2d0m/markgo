# Architecture overview

## Goals

MarkGo is a fully drawn native desktop Markdown editor for focused writing. The architecture optimizes for source fidelity, responsive interaction, local data ownership, and testable behavior. AI integration is deferred until the core editor is ready.

## Current boundaries

```text
cmd/markgo
    |
internal/app ---- internal/mdeditor
    |                    |
    +--------------- MyGo native UI
    |
internal/storage
```

- `cmd/markgo` is the thin executable entry point. It creates the application model and configures the native window.
- `internal/app` owns workbench state, presentation, theme adaptation, document selection, and user interaction.
- `internal/mdeditor` owns MyGo Markdown editor state, source-preserving parser analysis, outline and statistics, formatting, list continuation, and live-preview replacements.
- `internal/writingui` shares native control feedback, fields and floating surfaces between the workbench and editable popovers; it owns no document state or I/O.
- `internal/storage` owns UTF-8 file snapshots, guarded saving, and recursive folder enumeration.
- `internal/preferences` atomically writes small local settings. Appearance write revisions reject older workers that finish late; queued writes are drained after the event loop exits.
- `internal/session` stores bounded recovery checkpoints atomically. The application holds immutable source snapshots, original disk bytes and buffer versions together, serializes them on a worker, and rejects stale write requests. Recent paths and session metadata share the same transaction.
- `resources` contains assets loaded by MyGo packaging.

Document tabs and the sidebar select the same persistent File objects. The shared tab strip owns only navigation, focus and horizontal scroll state; it reuses each document’s source, selection, undo and guarded close lifecycle. The sample-only Writing Preview uses the same strip with in-memory pages and keeps closed samples available for reopening.

Documents have stable identities independent of their names and paths. Each saved document has a disk snapshot with a content digest and permissions. Saves serialize immutable editor snapshots, write and sync a temporary file beside the destination, recheck the disk snapshot, and rename the temporary file. A concurrent external writer can still race the final check and rename; this is conflict detection, not filesystem locking. New files use an exclusive hard link of the completed temporary file and require filesystem support for hard links.

## Document model

Plain Markdown is the source of truth. Live preview uses parser-derived byte offsets and editor replacements to hide or replace source marks without changing the buffer. A caret or selection inside a formatted element reveals its source. Interactive task markers edit the original buffer and remain subject to undo. Stale parser results are rejected by buffer version; existing replacements follow edits while fresh analysis runs.

MyGo's Markdown parser supplies block and inline structure. This application currently parses a complete snapshot on a worker; it is not an incremental Markdown parser. The editor's highlighter remains incremental. List continuation reuses its confirmed tokenizer context to avoid changing fenced code or comments, with a bounded highlighting budget.

Reading uses MyGo's native Markdown renderer and per-document local-image cache. The flattened source is supplied by the analysis worker; MyGo maintains its own incremental rendering parse on the UI thread. It does not introduce a browser or WebView. Reading and editing are separate views of one source. The editor does not yet provide variable-height, directly editable headings, tables, images, or math.

The editor normalizes CRLF in memory. Save snapshots reproduce the original line-ending convention; unedited mixed-ending files preserve their original bytes. Edited mixed-ending documents can only be saved as an explicitly confirmed normalized copy. A save finishing after a later edit must not clear the dirty marker. Reload also checks the buffer version before replacing it.

## Execution model

MyGo's window thread owns UI state. Views may rebuild several times for one event and must not perform slow I/O or start untracked background work. File operations, dialogs, folder scanning, and parser analysis run in guarded workers and publish results through `Window.Update`. Save metadata is prepared before the UI callback. The headless test path computes parser results synchronously because it has no native event loop.

Appearance and session data load before the event loop starts. Normal launches request MyGo's single-instance lock before loading recovery state; subsequent launches forward their open requests. Recovery restores dirty buffers for review and only retains disk-save permission when the original bytes still match. A changed or unavailable original requires Save as. Clean documents are reloaded from disk rather than restored from stale text.

Recovery is debounced for 750 ms and scheduled at least every five seconds of continuous changes, with one routine checkpoint in flight. Explicit discard and final exit writes receive newer revisions. The workbench waits for the final write before closing. Keep drafts retains unsaved sources; Discard strips them while retaining named document paths. Failed final writes offer staying open or an explicit exit with the older checkpoint left intact. Checkpoints do not persist undo history or reader scroll positions.

Appearance settings load before the event loop starts. Theme and typography controls reuse persistent document state and cached editor/reader themes. A theme switch does not recreate a document or its image loader.

Elements and contexts are frame-owned. Persistent application state stores document and widget state, not `ui.Element` or `ui.Context` values.

## Future AI boundary

AI providers receive only context exposed by the user's action and provider settings. Model output is a proposed edit, not an implicit document write. A reviewable diff sits between the result and the editor buffer; accepting a change creates an ordinary undoable edit.

There is no MarkGo account, telemetry endpoint, or required application server. Provider credentials, when introduced, belong in the operating system's secret store rather than Markdown files or source control.
