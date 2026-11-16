# Native GPUI workbench

This directory contains MarkGo's Rust + GPUI Kit client. It draws a native workbench and directly edits headings and paragraphs through the source-preserving model. There is no WebView, embedded JavaScript application or copied Astrolith implementation. The ordinary Go executable remains available until the [migration gates](../docs/architecture/gpui-migration.md) pass.

Dependency and asset attribution is recorded in [THIRD_PARTY.md](THIRD_PARTY.md).

The current visual projection is intentionally narrower than the Go editor. Lists, code and inline delimiters remain literal; format commands edit their Markdown source. Recovery, advanced Markdown presentation and full platform acceptance are not complete. This client is not a 1.0 beta replacement yet.

## Check

Install Rust with rustup. Commands run from this directory select the pinned toolchain automatically:

```sh
cd native
cargo fmt --check
cargo clippy --locked --all-targets -- -D warnings
cargo test --locked
cargo clippy --locked --features render-check --all-targets -- -D warnings
cargo build --locked --features desktop --bin markgo
cargo run --locked --features desktop --bin markgo -- --preview
cargo run --locked --features render-check --example render_check -- ../build/gpui-evidence
```

From the repository root, use `make native-check`, `make native-build`, `make native-preview` and `make native-render-check`. Rust **1.99.1**, GPUI Kit **0.9.1** and its default upstream GPUI **0.3.12** dependency graph are pinned. The earlier Rust 1.92 baseline failed on a newer standard-library API used by GPUI's dependencies; no framework source patch or nightly feature is used.

`--preview` and `MARKGO_EPHEMERAL=1` use sample documents in memory, skip preferences and disable real-file dialogs/writes. Run without them to open/save local files. File operations run on workers; saves use immutable committed snapshots and reject changed destinations. New files are published exclusively instead of overwriting an existing Save as target. Existing-file replacement retains the Go client's double-check/atomic-rename policy; it is not an OS-wide transaction against uncooperative external writers.

On macOS, `sh native/scripts/bundle-macos.sh` builds `build/MarkGo GPUI Preview.app` without launching it. It uses the approved `dev.markgo.writing-preview` identifier and does not replace an installed application. Native window operations still require the meeting's desktop tool and approval.

## Implemented boundaries

- `document`: original Markdown bytes, directed selections, source replacement transactions, undo/redo and versioned immutable snapshots. History retains changed text rather than a whole document per edit.
- `Composition`: separate candidate text and selection. Updates leave committed snapshots untouched. Commit becomes one undo step; cancellation retains the prior source selection and redo history.
- `projection`: parser-confirmed top-level ATX headings and paragraphs, with compact per-line source/display maps. Other Markdown remains literal. UTF-8 validation and revision/source checks prevent a stale projection from editing a different document state.
- `source_contract`: cases adapted from the existing Go editing contracts, including CJK/emoji, reversed cross-block selection, paired heading delimiters, metadata/code/reference preservation, mixed line endings and independent document history.
- `editing`: visible/source/native-UTF-16 adaptation, grapheme navigation, composition replacement and cancellation, inline format toggles, block insertion and list/task/quote continuation.
- `desktop/writer`: wrapped GPUI text, variable heading metrics, hit testing, selection painting, input-method geometry, clipboard and keyboard actions. Typing bursts coalesce; composition commits remain separate undo transactions.
- `desktop/workbench` and `views`: persistent tab identities and scroll handles, file/folder navigation, heading outline, eight slash commands, plain-text find/replace, source/reading choices under View, close guards, 12 palettes and appearance settings. Existing MarkGo vector icons are reused.
- `storage` and `preferences`: guarded temporary-file writes, UTF-8 validation, symlink rejection, bounded folder enumeration and versioned appearance storage. Preferences use `MarkGo/native/appearance.conf` in the platform config location; Go settings/recovery files are not migrated or overwritten. Writes are ordered and errors retain the current in-memory appearance.

The source and projection offsets are UTF-8 bytes; platform text-service offsets are UTF-16 code units. `utf16_to_byte` exposes explicit rounding for a surrogate pair, and `byte_to_utf16` rejects an invalid byte boundary. Preedit remains separate from saved source, and its heading metrics stay stable while the candidate text changes.

## Validation and remaining gates

`render_check` uses the actual application and Metal on macOS / WGPU on Linux, with hidden windows and temporary in-memory documents. It exercises pointer and keyboard dispatch, tab switching, typing/undo, close cancellation, slash selection, all 12 themes, search, composition adapter calls and candidate geometry. It also captures normal, narrow, empty, appearance, search, slash, close and error views. The error image is an explicit component fixture; storage failures are tested separately with temporary files.

These tests do not operate an installed input method or prove OS clipboard, screen-reader, window-chrome or GPU latency behavior. Complete the approved desktop checks separately. Windows/Linux native acceptance, structured Markdown presentation, recovery/session persistence, scope/regex search, long-document virtualization and release packaging remain gates before retiring Go. The parser and layout currently process the document as a whole; no large-file performance claim is made. File opening is bounded at 32 MB; folder enumeration is bounded at 2,000 Markdown files and eight directory levels.
