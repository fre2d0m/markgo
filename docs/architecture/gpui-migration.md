# Rust and GPUI migration decision

Date: 2026-11-16. Target direction: Rust with GPUI Kit. Current runnable client: Go with MyGo.

## Decision and reasons

Move toward a Rust, fully drawn GPUI client in independently reviewable stages. The product owner's preference is to invest in a desktop ecosystem with broader maintenance support and commercial application experience. GPUI Kit's separation of unstyled behavior and styled components also fits MarkGo's need for coherent interaction and an independent visual identity.

Those are reasons to select a direction, not proof that a new client is more reliable. Update frequency is not a stability guarantee. Pin the framework and its matching dependency graph, review upgrades separately, and run the editor's regression corpus before adopting updates.

The reviewed GPUI Kit baseline is **0.9.1**, with its default upstream GPUI snapshot. Its [published manifest](https://github.com/longbridge/gpui-kit/blob/main/crates/kit/Cargo.toml) enables components and assets; WebView and the JavaScript extension host are separate packages. The first GPUI integration should use an exact Kit requirement and a checked-in Cargo lockfile. Do not add `gpui-fast`, a separate upstream GPUI version, WebView, or a JavaScript host by default.

The native Rust foundation uses Rust 1.92.0 and pulldown-cmark 0.13.6. The current library deliberately does not link GPUI yet: dependencies enter the application with their first real integration. [Installation requirements](https://gpui-kit.com/docs/installation/) and the target-platform input behavior must be checked again when introducing the window implementation.

## Architecture to preserve

- A single Markdown source owns edits, undo, snapshots and recovery. Rendering is a disposable projection; no Markdown serializer round trip is allowed.
- The Rust foundation retains source bytes, including line endings. Platform UTF-16 positions are translated at the input boundary; UTF-8 source offsets are never treated as character counts.
- Preedit is separate state. Save/recovery snapshots contain committed text, and a successful composition creates one undo transaction. Cancellation must not consume prior writing history.
- Each document retains its own selection, history and composition state. The future workbench must define and test cancellation on navigation before taking focus away.
- File operations remain asynchronous, snapshot-based and guarded against stale results and external changes. Existing Go recovery data is not migrated or overwritten by this foundation.
- Prefer a Rust client with native Rust services. A Go/Rust bridge is not the default architecture; it would add a second runtime, error boundary and packaging path.
- Keep the product design contract, twelve theme identities, original icon assets, keyboard behavior and writing-first scope. AI remains deferred.

## Delivery sequence

| Stage | Deliverable | Exit evidence |
| --- | --- | --- |
| 1. Source and history foundation | Isolated Rust library; immutable snapshots, composition staging, reversible source transactions and a narrow heading/paragraph projection | Portable tests, formatting and lint checks; no native input claim |
| 2. GPUI editing integration | Real editable heading/body layout using the same source/history; native text-service adapter and candidate geometry | Approved native checks for composition, cross-block selection/delete/paste, keyboard navigation, undo/redo and source preservation |
| 3. Editing and workbench parity | Lists, tasks, quotes, code, inline spans, slash/search; tabs, settings, file guards and recovery | Existing behavior cases ported with equivalent assertions; real-file tests in temporary directories; actual screenshots for each changed flow |
| 4. Release switch | Packaging, platform integration, performance and failure recovery | Windows-first native acceptance, then each additional advertised platform; only then replace the default executable and retire the Go client |

Stage 2 must exercise direct visual editing. A source editor beside a Markdown viewer does not satisfy this gate. Kit's [Editor](https://gpui-kit.com/component/editor/) and [TextView](https://gpui-kit.com/component/text-view/) provide useful primitives, but their documented capabilities are not a drop-in Typora editor.

Before switching clients, compare a fixed 100,000-character corpus on the same hardware. Record native input latency, opening time, sustained memory, scrolling, text reflow and tab retention. Existing beta targets remain targets, not measured GPUI results. Also record adapter complexity and any framework patches required; passing a small demonstration alone is insufficient.

## Evidence and current limits

Stage 1 lives in [`native/`](../../native/README.md). Its projection recognizes top-level ATX headings using parser-confirmed source ranges. Unsupported constructs remain literal. It rejects stale projections and invalid UTF-8 offsets, retains LF/CRLF and mixed endings, and repairs closing heading markers when a range edit consumes their opening prefix. It does not yet implement full structured Markdown editing, incremental parsing, native rendering, native IME event sequencing, clipboard, file I/O, or app/session migration.

The composition tests exercise the Rust model, not a macOS, Windows or Linux input method. A GPUI build, a headless test and an actual desktop observation are separate evidence. No performance or visual improvement is claimed by this first stage.

The Go client stays runnable while these gates are completed. Keep the new core independent of real user files and settings; do not replace a configured preview app or open a host application through shell commands. Native operations follow the meeting's approved desktop tool and application identifiers.
