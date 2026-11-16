# MarkGo

MarkGo is a local-first Markdown editor written entirely in Rust with GPUI Kit. The root Cargo package is the only client. No WebView, JavaScript UI or second runtime. Core writing takes priority; AI is deferred.

## Engineering boundaries

- `document` owns canonical UTF-8 Markdown, source selection, undo/redo and composition staging. Snapshots exclude preedit.
- `markdown`, `projection`, `editing` discover parser-confirmed ranges and turn editing into source transactions. Never round-trip through a rich-text serializer.
- `desktop/` owns GPUI views, native input, workbench, navigation and recovery coordination. Confirm APIs against pinned dependency source.
- `storage` owns bounded reads and guarded file writes. Run I/O on workers; reject stale reloads. Save-as never overwrites an existing destination.
- `session` owns bounded, versioned, ordered checkpoints. Recovery never writes user documents; unknown journals and changed originals must remain intact.
- `preferences` owns bounded settings; retain persisted theme IDs and serialize writes.
- Preserve existing workspace changes. Do not discard user work, create empty stubs or commit credentials, machine paths, binaries or unknown-origin assets.
- Keep docs current. Public guides describe user behavior; development acceptance belongs in testing/release documents.

## Design and native behavior

Read [DESIGN.md](DESIGN.md) before visible changes. It is the single design authority. [Implementation values](docs/design/design-system.md) describe code; [review](docs/design/review.md) defines evidence. External analyses and generic skills do not override user choices.

- Follow shared dimensions, semantic colors, original vectors and 40 DIP targets. Preserve square lower tab corners and the page silhouette.
- Default to visual editing. Source/reading belong in View; retain each tab's selection, history and useful scroll.
- Synchronize Kit/Base colors through `Theme::update`. Prose preferences must not resize controls. Check all twelve themes after shared changes.
- Formatting, task/link/code operations use one undoable source transaction and return writing focus. Preserve original marker spelling, info suffixes and line endings.
- Boundary edits repair paired inline/fence syntax; literal fences typed into code extend together.
- Slash only opens on a typed slash in an empty ordinary paragraph, yields to composition and fits complete rows.
- Search uses canonical source, reconciles scope against an immutable baseline and never silently broadens it. Release baselines on dismissal/navigation. Reading navigation uses rendered source anchors.
- UI state stays on the UI thread. Slow I/O runs on workers and publishes only if identity/revision still match.
- Ordered checkpoint generations prevent obsolete workers resurrecting discarded text. Normal close waits for its final write; OS shutdown is best effort within the toolkit timeout and cannot cancel termination.
- `--preview` / `MARKGO_EPHEMERAL=1` never loads real documents, settings or sessions. Tests use temporary directories, never the writing library.
- For visible changes, capture and inspect actual rendered views, fix issues and recapture. Cover normal/empty/error/narrow and relevant themes.
- Headless GPU rendering is not proof of installed IME, OS clipboard, accessibility, native window behavior or frame pacing. Host operations use only the session-authorized tools and approvals. Build scripts never launch apps.

## Commands

```sh
cargo run --locked
cargo run --locked -- --preview
make check
make render-check
scripts/bundle-macos.sh --preview
```

Pin Rust/dependencies and commit Cargo.lock. Run model tests, formatting and Clippy for core and desktop. UI/input changes also need real GPU rendering. Keep [beta gates](docs/development/beta-delivery.md) honest: code, tests, builds, rendered screenshots and host acceptance are different evidence.
