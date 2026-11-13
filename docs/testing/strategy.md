# Testing strategy

## Headless behavior tests

MyGo's `ui.NewTester` drives the application view without opening a desktop window. Tests cover document and folder filtering, outline navigation, formatting and undo, task-marker edits, find/replace and undo, focus restoration, source/read switching, focus mode, and simulated input-method composition. Native IME acceptance is still separate. Appearance tests exercise all palettes, system/explicit theme behavior, selection/scroll/undo preservation, font and width controls, Escape dismissal, new documents from reading mode, and hit targets in a 960 × 640 window with a long filename.

Test observable behavior and persistent state rather than incidental pixel coordinates. Keep stable control labels and identities.

The twelve-theme gallery checks at least 4.5:1 contrast for primary and secondary text on normal, hover and pressed surfaces, plus primary-action and semantic text. Each theme must survive preference persistence and resolve the correct native light/dark appearance. At 960 × 640 and 1×/1.5×/2× scale, all gallery targets must fit and remain disjoint. Exercise keyboard entry/activation/exit, pointer cancellation, rapid panel switching and both writing modes. [Product polish verification](../development/product-polish.md) includes the full render matrix and change comparison.

Custom-title-bar tests use fake window commands to verify exactly-once dispatch, maximize/restore labels, and preservation of writing state. They exercise control reachability at 1×, 1.5×, and 2× display scale in a minimum-sized window, including Focus mode. Panel tests interrupt enter/exit transitions with and without reduced motion, then check selection and continued typing. File-list tests verify consecutive keyboard navigation and Enter-to-edit behavior.

Interaction tests advance the actual animation clock, reverse a sidebar mid-flight, reject input in closing navigation, and verify that typing and undo continue afterward. They exercise row/context close protection, copying a specific document path, reader-anchor navigation, and save acknowledgment expiry/precedence over stale saves. Recorded headless frames can demonstrate motion; they remain distinct from native desktop acceptance.

## Editor and parser tests

Preview replacements must preserve the source buffer and reveal the active element's marks. Cover CJK byte offsets, escapes, code fences, tasks, links, nested delimiters, YAML front matter, heading offsets, and statistics. List continuation must not alter fenced code or comments. Golden rendering tests complement rather than replace editing behavior tests.

Preference tests use temporary directories for rapid changes, latest-write ordering, JSON round trips, invalid-setting normalization, and interrupted-write cleanup. They never load or save the real user preferences.

Storage tests use temporary directories to verify bytes, CRLF, permissions, invalid UTF-8, symlink rejection, external-change detection, exclusive creation, and cleanup. Save and reload tests protect edits made after a snapshot was captured. Never use a shared writing folder as a test fixture.

Run `go test ./internal/mdeditor -run '^$' -bench BenchmarkAnalyze -benchmem` to measure full-snapshot parser work. These measurements do not establish native input latency or large-document usability.

## Recovery verification

Recovery tests use temporary directories and immutable source captures. They cover named and untitled drafts, current selection, LF/CRLF and mixed endings, changed/missing originals, a disk save completed before the next checkpoint, repeated restoration, explicit discard, recent-history removal and failed or stale worker completions. Storage tests reject oversized, unsupported and malformed data without altering the last complete checkpoint, and exercise actual destination write failures.

`TestRecoverySurvivesKilledProcess` starts a child test process, waits for a real checkpoint to finish, forcibly kills the process, restores both drafts, compares the original file bytes and repeats restoration without duplicates. `TestCrashBeforeAtomicReplacementPreservesPreviousCheckpoint` pauses a child immediately before atomic replacement, after syncing the temporary file, kills it, and verifies that the last complete checkpoint loads and the abandoned temporary file is cleaned. These tests do not prove recovery of uncommitted keystrokes, power-loss durability, real full-volume behavior or native window shutdown.

UI tests check filtering, Enter-to-open, Escape-to-write, safe history removal, Keep drafts/Discard checkpoint content, cancellation and modal focus. A simulated disk-full completion verifies the persistent warning. Final exit publication still requires native event-loop acceptance.

Capture the actual application view for visual inspection:

```sh
MARKGO_SCREENSHOT_DIR="$PWD/build/recovery-review" CGO_ENABLED=0 go test ./internal/app -run TestRecoveryScreenshots -count=1
```

This captures Paper, Warm paper and Graphite in recovery, recent, missing-file, empty, no-results, backup-error, conflicting-original, long-filename, exit and failed-exit states at 960 × 640, plus recovery/recent views at 1320 × 840. Inspect the images after each visible change and regenerate after fixes. The screenshots use fixture paths, not the user's writing library.

## Native verification

Headless tests do not prove native window, font, input-method, clipboard, file-dialog, or GPU behavior. Changes in those areas require a real application run and platform-specific evidence. Record the operating system, architecture, and display scale; do not infer macOS or Linux behavior from Windows verification alone.

For the custom title bar, check dragging from blank areas, and verify document tabs do not initiate a window drag, double-click maximize/restore, all resize edges, minimize/restore, maximize icon changes, and close with unsaved text. On Windows 11, check Snap layouts over the maximize button and at 100%, 150%, and 200% display scaling. In Focus mode the window buttons must remain reachable. Clicking or dragging a writing control must not drag the window. Neither a compiled executable nor a headless render establishes those OS behaviors.

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

Build output stays under ignored `build/` directories and is not committed. Use `MARKGO_EPHEMERAL=1` when launching an isolated native check; it bypasses native window state, appearance preferences, recovery, and recent documents. `go tool mygo build -skip-dmg` builds the macOS app bundle without producing a disk image. Headless screenshots are layout evidence only; label them accordingly. Temporary verification helpers live under ignored `.scratch/`; documentation screenshots are deliberate assets.

The typesetting experiment has a separate sample-only executable: `go run ./cmd/markgo-preview`. It never loads or saves real documents or application state. Its behavior tests include direct `TextInputClient` calls because native input methods can bypass keyboard observers. Coverage includes lists, quotations, code, paired-syntax edits, inline styles and caret affinity, reference-link isolation, link-panel validation/focus, source-mode history, Markdown clipboard ranges, wrapped Unicode text-service geometry and preserved LF/CRLF captures. Inline attribute caches are compared with fresh projections after edits and undo, including equal-length replacements and unchanged-text composition. Capture its 141 rendered states with:

```sh
MARKGO_SCREENSHOT_DIR="$PWD/build/typesetting-review" CGO_ENABLED=0 go test ./internal/app -run '^TestTypesettingScreenshots$' -count=1
go test ./internal/mdeditor ./internal/app -run '^$' -bench 'BenchmarkTypesetting(HeadlessInput100K|EditCycle100K|PreviewEditCycle100K|ListEditCycle100K|BlocksEditCycle100K)$' -benchmem -benchtime=200x -count=3
go test ./internal/mdeditor -run '^$' -fuzz '^FuzzTypesettingIncrementalProjection$' -fuzztime=20s -parallel=2
```

Inspect every captured scene across all three palettes. Fixed-length edit/undo cycles prevent faster implementations from benchmarking larger documents just because they complete more iterations. Compare the same workload, hardware, toolchain and iteration count, with no competing build/test jobs. Incremental projection tests compare against a fresh full parse, including edits to distant block context and Unicode source/view offsets. See [native typesetting](../development/native-typesetting.md) for evidence boundaries and the remaining native-platform gates.

Document-tab regression checks cover independent buffers and undo, retained caret/scroll, guarded close actions, inactive versus active closure, duplicate filenames, arrow/Enter navigation, pointer cancellation and no-hover close targets, 24-document overflow and keyboard selection, resizing, and in-memory preview reopening. `TestWorkbenchTabsScreenshots` renders four multi-document scenes in every palette.
