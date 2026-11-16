# MarkGo

MarkGo is a free, open-source, local-first Markdown editor. The default client uses Go and fully drawn MyGo native UI. `native/` contains the Rust + GPUI Kit migration client, including a real editable workbench; it has not yet reached editor/recovery/platform parity with the Go client. Core writing takes priority; AI integration is deferred.

## Engineering rules

- Keep changes focused, tested, and honest about implemented versus planned behavior.
- Keep `cmd/markgo` thin. `internal/app` owns application state and presentation; `internal/mdeditor` owns Markdown editor adapters and live-preview behavior; `internal/storage` owns file and directory I/O; `internal/preferences` owns atomic application-setting writes; `internal/session` owns bounded, versioned recovery storage.
- Add packages only with their first real behavior. Do not create empty architecture stubs.
- Plain Markdown source is the document's source of truth. Preview must not silently rewrite it.
- Save immutable snapshots off the UI thread. Only clear a document's modified state if the current buffer is the version actually saved. Never overwrite an externally changed file or an existing Save as destination.
- Keep integration-test files in `t.TempDir()`. Set `MARKGO_EPHEMERAL=1` for isolated native checks to avoid loading or persisting real window state, appearance preferences, sessions, and recent documents.
- Keep file I/O, model calls, indexing, and other slow work off the UI thread.
- AI context and edits must be explicit, reviewable, and undoable. No telemetry or mandatory account.
- Do not commit credentials, local machine paths, build output, or unknown-origin assets.
- Update relevant documentation and this file when commands, boundaries, or behavior change.
- Run `make check` before delivery; run race, native, or security checks when relevant.
- For every visible flow, capture and inspect the actual application view before delivery. Cover normal, empty, error, narrow-window and relevant theme states; fix issues and capture again. Use headless screenshots as layout evidence, and report native-platform checks separately.
- Recovery never writes the user's Markdown files. Keep source, original bytes and version together; discard only the chosen revision, and do not overwrite unreadable or unsupported recovery data.

## Design authority

Before changing a visible surface, typography, theme, icon or interaction, read [DESIGN.md](DESIGN.md). It is the single product-design contract for the native client. [docs/design/design-system.md](docs/design/design-system.md) records current implementation values; it does not override the contract's targets. Follow [docs/design/review.md](docs/design/review.md) for affected state, geometry and evidence checks.

Keep product commitments, proposed visual measurements and implemented behavior distinct. External brand analyses and generic UI skills are references, not additional authorities. Apply shared changes consistently, update the implementation record with actual code changes, and do not claim a document-only change as visual or native acceptance.

## MyGo UI

Read `.agents/skills/mygo-maintenance/SKILL.md` before changing UI code. Confirm APIs against the MyGo version pinned in `go.mod`.

- Views build from persistent application state on the main thread and may run several times for one event.
- Do not store `ui.Element`, `ui.Context`, or drawing parts in application state. Keep widget state and `ui.Handle` instead.
- Give stateful and repeated controls stable identities with `c.Key`.
- Use `Window.Update` to publish background results. Never block the UI thread or start untracked work while building a view.
- Prefer MyGo widgets and base widgets so keyboard, focus, input methods, and accessibility keep working.
- Shared control feedback, fields and floating surfaces live in `internal/writingui`; keep rich-text child controls' input/focus policy at their call sites. Theme IDs come from the palette catalog, not hardcoded enum lists or positions.
- Document tabs reuse the existing File identities, close guards and source history. Key the native tab's parent by file ID, retain each document's selection/scroll/undo, and reveal the selected tab after navigation or resizing. Preview tabs keep closed sample edits in memory and never attach real stores.
- Verify every theme's normal, hover and pressed text contrast, preference round trip and OS light/dark choice. Appearance controls must fit at 960 × 640 and retain keyboard reachability, cancellation, source, selection, scroll and undo.
- Use `ui.NewTester` for behavior tests without opening a window.
- Typesetting projection changes must retain untouched source and paragraph layouts; compare incremental edits against a fresh parse, including distant fence/front-matter effects and Unicode offsets. Benchmark fixed edit/undo workloads separately from native input latency.
- Structured lists come from parser-confirmed items. Preserve original markers and written numbers; display semantic decimal numbering at every depth. Prefix/indentation changes, task toggles and their necessary Markdown separators belong to source history, never a rich-text serializer.
- Task controls keep the writing focus and selection, use non-overlapping 40 DIP targets, and leave the document still. Verify pointer cancellation, keyboard toggling, source-mode undo, whole-item clipboard syntax and native text-service hit geometry after layout changes.
- Structured quotes and top-level fenced code share the same source history. Hidden fence lines must not create fake editable paragraphs; empty fences need a virtual caret without no-op source changes. Retain fence characters, info suffixes, body whitespace and line endings.
- Literal fences typed or pasted into code extend both delimiters together. Edits across one boundary remove its surviving paired fence in the same transaction; verify multi-block, reversed-selection undo.
- Code headers use 40 DIP targets and native vector icons. Language changes, Escape and Continue return focus to the preserved writing selection. Keep clipboard feedback local to the code header and do not animate the document.
- Inline presentation uses parser-confirmed spans and sparse hidden source intervals. Preserve delimiter spelling, link titles/references and line endings. Selection endpoints need different marker affinity; crossing one boundary removes its surviving paired syntax in the same history step.
- Keep inline attribute caches relative to their paragraph and invalidate changed/composing paragraphs. Do not enumerate global text runs for every span on every input. Verify cached attributes against fresh projections and benchmark a format-dense 100,000-character document.
- Link editing must retain the writing selection and return focus after Apply, Cancel or Escape. Never open an address implicitly while editing. Reference-link changes apply only to the chosen occurrence, without rewriting its shared definition.
- Ordinary documents default to the source-preserving Typesetter. Each visited File keeps its own Typesetter and hidden keyed view so MyGo's private rich-text viewport survives tab/view changes. Suspended views do not parse, paint or take input; release them when a file closes. Source and reading are secondary View-menu choices, not a persistent Write/Read segment.
- Slash insertion arms only on a typed slash in an empty ordinary paragraph, yields to composition, and never opens for pasted paths or literal code. Query deletion and block conversion share one source transaction. Keep the popup anchored horizontally, avoid the caret line when flipping above, and test whole-row bounds at the minimum size.
- Search in visual editing uses the canonical buffer, with UTF-8 scope anchors reconciled against the last immutable search text. Preserve replaced text at scope edges through undo, explicit syntax and focus. Release search baselines when closing search or switching documents. Save/recovery snapshots exclude preedit without cancelling it; navigation cancels preedit before native focus-loss unmark can commit it.

## Rust and GPUI migration

Read [the migration decision](docs/architecture/gpui-migration.md) and [native foundation guide](native/README.md) before extending `native/`. Keep the Go client runnable until the native migration gates pass. The shared design authority applies to either renderer; MyGo-specific APIs above apply only to the Go client.

- Keep committed Markdown, source history and composition staging in the renderer-independent Rust model. Snapshots must exclude preedit. Validate UTF-8 ranges and reject stale projections before mutation.
- The first projection supports top-level ATX headings and paragraphs; unsupported syntax stays literal. Preserve parser-confirmed context, closing delimiters, line endings and reversed selections through edit/undo/redo.
- Pin Rust and dependency versions and commit `Cargo.lock`. Introduce GPUI dependencies with their first actual integration; do not add unused packages or claim the current core is a GPUI application.
- Run `make native-check` for Rust changes. Model tests do not establish GPUI text-service behavior, candidate geometry, grapheme navigation, visual polish or native performance.
- The native desktop feature uses GPUI Kit 0.9.1 with the default upstream engine. Keep text-service events in `desktop/writer`, source/display mapping in `editing`/`projection`, file guards in `storage` and ordered preference writes in `preferences`/`desktop/workbench`.
- Use `Theme::update` to synchronize Kit's component and Base semantic tokens. Direct mutation alone leaves stale control colors. Theme preview swatches use opaque exact palette colors.
- The title strip must have an explicit available width inside Kit's title-bar wrapper; test overflow with at least four tabs at 960 DIP so utility controls cannot move offscreen.
- The native preview is sample-only. `--preview` or `MARKGO_EPHEMERAL=1` never reads/writes preferences or real documents. `make native-render-check` uses hidden windows with the real Metal/WGPU renderer; it does not operate a host desktop application.
- Run `cargo clippy --locked --features render-check --all-targets -- -D warnings`, the core tests and native-render-check after native interaction changes. Screenshots must include empty/normal/narrow, slash, appearance, search, close and error states. Real installed-IME and OS-window acceptance remain separate.

## Commands

```sh
go run ./cmd/markgo
go tool mygo dev .
make check
make native-check
make native-build
make native-preview
make native-render-check
make race
make security
go tool mygo build
go run ./cmd/markgo-preview
go tool mygo build -skip-dmg ./cmd/markgo-preview
```

## Repository layout

- `cmd/markgo`: executable entry; `internal/app.WindowOptions` owns the self-drawn window configuration.
- `cmd/markgo-preview`: isolated, sample-only native typesetting executable. Never attach real file, recovery or preference stores to it; its state is intentionally in memory.
- `internal/app`: workbench state, views, themes, and UI behavior tests.
- `internal/mdeditor`: Markdown document editor state and live-preview replacements.
- `internal/writingui`: shared native control styling, field validation states and floating-surface motion; no document state.
- `internal/storage`: UTF-8 file snapshots, guarded saving, and folder enumeration.
- `internal/preferences`: bounded preference loading and ordered atomic setting writes.
- `internal/session`: bounded recovery checkpoints and session metadata; `internal/app` coordinates restoration and UI feedback.
- `resources`: application resources used by MyGo packaging.
- `native`: Rust source/history/input/projection model, guarded local I/O and GPUI desktop workbench; the existing Go client remains the default until parity gates pass.
- `docs`: architecture, design, development, and testing documentation.
- `.github`: issue and pull-request contracts, dependency automation, and CI.

See `docs/architecture/overview.md` and `docs/development/setup.md` for details.
