# Architecture overview

## Goals

MarkGo is a native desktop Markdown editor for focused writing and reviewable AI assistance. The architecture optimizes for source fidelity, responsive interaction, local data ownership, and testable behavior.

## Current boundaries

```text
cmd/markgo
    |
internal/app ---- internal/mdeditor
    |                    |
    +---------------- MyGo native UI
```

- `cmd/markgo` is the thin executable entry point. It creates the application model and configures the native window.
- `internal/app` owns workbench state, presentation, theme adaptation, document selection, and user interaction.
- `internal/mdeditor` owns MyGo Markdown editor state, selection extraction, and live-preview replacements.
- `resources` contains assets loaded by MyGo packaging.

The current documents are editable in-memory samples. Real file ownership, persistence, parsing, export, and AI provider packages will be introduced with their first implementation rather than as empty stubs.

## Document model

Plain Markdown is the source of truth. Live preview uses editor replacements to hide or replace source marks without changing the underlying buffer. The line containing a caret keeps its source visible. Interactive replacements, such as task markers, edit the original buffer and remain subject to undo.

The current regular-expression preview is an initial shell, not a complete CommonMark parser. Incremental CommonMark/GFM parsing belongs to the editor implementation milestone and must preserve arbitrary source text.

## Execution model

MyGo's window thread owns UI state. Views may rebuild several times for one event and must not perform slow I/O or start untracked background work. Future file operations, parsing, indexing, and model calls run outside the view builder and publish results through `Window.Update`.

Elements and contexts are frame-owned. Persistent application state stores document and widget state, not `ui.Element` or `ui.Context` values.

## Future AI boundary

AI providers receive only context exposed by the user's action and provider settings. Model output is a proposed edit, not an implicit document write. A reviewable diff sits between the result and the editor buffer; accepting a change creates an ordinary undoable edit.

There is no MarkGo account, telemetry endpoint, or required application server. Provider credentials, when introduced, belong in the operating system's secret store rather than Markdown files or source control.
