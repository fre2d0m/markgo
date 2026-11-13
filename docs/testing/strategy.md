# Testing strategy

## Headless behavior tests

MyGo's `ui.NewTester` drives the application view without opening a desktop window. Tests cover document and folder filtering, outline navigation, formatting and undo, task-marker edits, find/replace and undo, focus restoration, source/read switching, focus mode, and simulated input-method composition. Native IME acceptance is still separate. Appearance tests exercise all palettes, system/explicit theme behavior, selection/scroll/undo preservation, font and width controls, Escape dismissal, new documents from reading mode, and hit targets in a 960 × 640 window with a long filename.

Test observable behavior and persistent state rather than incidental pixel coordinates. Keep stable control labels and identities.

Custom-title-bar tests use fake window commands to verify exactly-once dispatch, maximize/restore labels, and preservation of writing state. They exercise control reachability at 1×, 1.5×, and 2× display scale in a minimum-sized window, including Focus mode. Panel tests interrupt enter/exit transitions with and without reduced motion, then check selection and continued typing. File-list tests verify consecutive keyboard navigation and Enter-to-edit behavior.

Interaction tests advance the actual animation clock, reverse a sidebar mid-flight, reject input in closing navigation, and verify that typing and undo continue afterward. They exercise row/context close protection, copying a specific document path, reader-anchor navigation, and save acknowledgment expiry/precedence over stale saves. Recorded headless frames can demonstrate motion; they remain distinct from native desktop acceptance.

## Editor and parser tests

Preview replacements must preserve the source buffer and reveal the active element's marks. Cover CJK byte offsets, escapes, code fences, tasks, links, nested delimiters, YAML front matter, heading offsets, and statistics. List continuation must not alter fenced code or comments. Golden rendering tests complement rather than replace editing behavior tests.

Preference tests use temporary directories for rapid changes, latest-write ordering, JSON round trips, invalid-setting normalization, and interrupted-write cleanup. They never load or save the real user preferences.

Storage tests use temporary directories to verify bytes, CRLF, permissions, invalid UTF-8, symlink rejection, external-change detection, exclusive creation, and cleanup. Save and reload tests protect edits made after a snapshot was captured. Never use a shared writing folder as a test fixture.

Run `go test ./internal/mdeditor -run '^$' -bench BenchmarkAnalyze -benchmem` to measure full-snapshot parser work. These measurements do not establish native input latency or large-document usability.

## Native verification

Headless tests do not prove native window, font, input-method, clipboard, file-dialog, or GPU behavior. Changes in those areas require a real application run and platform-specific evidence. Record the operating system, architecture, and display scale; do not infer macOS or Linux behavior from Windows verification alone.

For the custom title bar, check dragging from blank areas and the document title, double-click maximize/restore, all resize edges, minimize/restore, maximize icon changes, and close with unsaved text. On Windows 11, check Snap layouts over the maximize button and at 100%, 150%, and 200% display scaling. In Focus mode the window buttons must remain reachable. Clicking or dragging a writing control must not drag the window. Neither a compiled executable nor a headless render establishes those OS behaviors.

## Quality gates

```sh
make check
make race
make security
go tool mygo build
```

- `make check`: formatting, module tidiness, Go and MyGo vet, Staticcheck, and tests.
- `make race`: shared-state race detection.
- `make security`: Go vulnerability analysis.
- `go tool mygo build`: the real application and platform packaging path.

Build output stays under ignored `build/` directories and is not committed. Use `MARKGO_EPHEMERAL=1` when launching an isolated native check; it bypasses both native window state and appearance preferences. `go tool mygo build -skip-dmg` builds the macOS app bundle without producing a disk image. Headless screenshots are layout evidence only; label them accordingly. Temporary verification helpers live under ignored `.scratch/`; documentation screenshots are deliberate assets.
