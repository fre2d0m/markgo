# Architecture

The root Cargo package is the only client. The default `desktop` feature uses GPUI Kit 0.9.1 with the upstream GPUI engine, Base behavior and native controls. `--no-default-features` builds the renderer-independent model. No WebView or cross-language bridge.

## Source and presentation

`document` owns UTF-8 source, selection, history and composition. `markdown` identifies parser-confirmed ranges; `projection` uses sparse hidden intervals; `editing` performs source transactions. There is no rich-text serializer. Unsupported structures stay literal.

`desktop/writer` owns UTF-16 text-service callbacks, grapheme movement, shaping/wrapping, selection and task/code controls. Reading uses Kit TextView and source-to-rendered anchors. Remote images are blocked; local paths resolve against the document. Native input/accessibility still require acceptance because MarkGo owns the editor model.

Each tab retains Writer state, original disk bytes and an optional guarded disk snapshot. Outline, search and file operations use source offsets. Search reconciles scope against an immutable baseline; cross-boundary changes invalidate scope rather than widening replacement.

## Files and recovery

File reads, writes, directory scans and journal serialization run on workers. Save syncs a sibling temporary file, rechecks original bytes/permissions and renames. Save-as exclusively hard-links the completed temporary file. Filesystems must support these operations. Another process can race the final check/rename: this is conflict detection, not filesystem locking.

Reload validates tab identity, source revision and bytes, then applies one undoable transaction. Dirty state compares current bytes with the saved snapshot. Reads reject symlinks and are bounded to 32 MiB. Directory traversal is bounded to depth 8 and 2,000 entries.

Recovery stores committed source, original bytes, cursor, view/scroll, active tab, folder and recent paths: at most 100 documents, 20 recent paths and 64 MiB. Idle checkpoints run after 750 ms or every five seconds of continuous changes. Ordered generations and a shared writer reject obsolete jobs. Recovery never writes document paths. Clean files reload from disk; missing/conflicting originals retain checkpoint text without overwrite permission.

Normal exit waits for its final keep/save/discard checkpoint. Discard prepares serialized state without destroying live buffers; save-all is sequential and cancellation cancels exit. The OS shutdown callback attempts the latest snapshot through the same ordered writer. GPUI imposes a timeout and cannot cancel OS shutdown. Force-kill/power loss can lose changes since the previous checkpoint. Undo history is not persisted.

Settings/session files remain under `MarkGo/native` in the platform configuration directory. On first use, absent native files can read the previous known MarkGo/session.json and MarkGo/appearance.json locations. Source files remain intact; subsequent writes use MarkGo/native. Unknown versions block migration instead of creating an empty journal. Other application directories are not searched. Unknown/corrupt journals disable writes. Concurrent instances do not coordinate UI state; external-change guards reject conflicting journals.

## Workbench and verification

`workbench` coordinates tabs, actions, dialogs and preferences; `recovery` coordinates checkpoints; `navigation` handles reload and file dispatch. Theme updates synchronize Kit/Base semantic tokens. Layout does no file I/O.

Preview is strictly in-memory. `render-check` uses hidden actual Metal/WGPU windows and event dispatch, not a host desktop app. It establishes rendered layout/adapter behavior only. See [testing](../testing/strategy.md).
