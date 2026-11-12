# MarkGo

MarkGo is a free, open-source, local-first Markdown editor built with Go and fully drawn MyGo native UI. Core writing takes priority; AI integration is deferred.

## Engineering rules

- Keep changes focused, tested, and honest about implemented versus planned behavior.
- Keep `cmd/markgo` thin. `internal/app` owns application state and presentation; `internal/mdeditor` owns Markdown editor adapters and live-preview behavior; `internal/storage` owns file and directory I/O.
- Add packages only with their first real behavior. Do not create empty architecture stubs.
- Plain Markdown source is the document's source of truth. Preview must not silently rewrite it.
- Save immutable snapshots off the UI thread. Only clear a document's modified state if the current buffer is the version actually saved. Never overwrite an externally changed file or an existing Save as destination.
- Keep integration-test files in `t.TempDir()`. Set `MARKGO_EPHEMERAL=1` for isolated native checks to avoid persisting real window state.
- Keep file I/O, model calls, indexing, and other slow work off the UI thread.
- AI context and edits must be explicit, reviewable, and undoable. No telemetry or mandatory account.
- Do not commit credentials, local machine paths, build output, or unknown-origin assets.
- Update relevant documentation and this file when commands, boundaries, or behavior change.
- Run `make check` before delivery; run race, native, or security checks when relevant.

## MyGo UI

Read `.agents/skills/mygo-maintenance/SKILL.md` before changing UI code. Confirm APIs against the MyGo version pinned in `go.mod`.

- Views build from persistent application state on the main thread and may run several times for one event.
- Do not store `ui.Element`, `ui.Context`, or drawing parts in application state. Keep widget state and `ui.Handle` instead.
- Give stateful and repeated controls stable identities with `c.Key`.
- Use `Window.Update` to publish background results. Never block the UI thread or start untracked work while building a view.
- Prefer MyGo widgets and base widgets so keyboard, focus, input methods, and accessibility keep working.
- Use `ui.NewTester` for behavior tests without opening a window.

## Commands

```sh
go run ./cmd/markgo
go tool mygo dev .
make check
make race
make security
go tool mygo build
```

## Repository layout

- `cmd/markgo`: executable entry and window configuration.
- `internal/app`: workbench state, views, themes, and UI behavior tests.
- `internal/mdeditor`: Markdown document editor state and live-preview replacements.
- `internal/storage`: UTF-8 file snapshots, guarded saving, and folder enumeration.
- `resources`: application resources used by MyGo packaging.
- `docs`: architecture, design, development, and testing documentation.
- `.github`: issue and pull-request contracts, dependency automation, and CI.

See `docs/architecture/overview.md` and `docs/development/setup.md` for details.
