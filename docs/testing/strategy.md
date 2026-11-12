# Testing strategy

## Headless behavior tests

MyGo's `ui.NewTester` drives the application view without opening a desktop window. Tests cover document and folder filtering, outline navigation, formatting and undo, task-marker edits, find/replace and undo, focus restoration, source/read switching, focus mode, and simulated input-method composition. Native IME acceptance is still separate.

Test observable behavior and persistent state rather than incidental pixel coordinates. Keep stable control labels and identities.

## Editor and parser tests

Preview replacements must preserve the source buffer and reveal the active element's marks. Cover CJK byte offsets, escapes, code fences, tasks, links, nested delimiters, YAML front matter, heading offsets, and statistics. List continuation must not alter fenced code or comments. Golden rendering tests complement rather than replace editing behavior tests.

Storage tests use temporary directories to verify bytes, CRLF, permissions, invalid UTF-8, symlink rejection, external-change detection, exclusive creation, and cleanup. Save and reload tests protect edits made after a snapshot was captured. Never use a shared writing folder as a test fixture.

Run `go test ./internal/mdeditor -run '^$' -bench BenchmarkAnalyze -benchmem` to measure full-snapshot parser work. These measurements do not establish native input latency or large-document usability.

## Native verification

Headless tests do not prove native window, font, input-method, clipboard, file-dialog, or GPU behavior. Changes in those areas require a real application run and platform-specific evidence. Record the operating system, architecture, and display scale; do not infer macOS or Linux behavior from Windows verification alone.

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

Build output stays under ignored `build/` directories and is not committed. Use `MARKGO_EPHEMERAL=1` when launching an isolated native check. `go tool mygo build -skip-dmg` builds the macOS app bundle without producing a disk image. Headless screenshots are layout evidence only; label them accordingly. Temporary verification helpers live under ignored `.scratch/`; documentation screenshots are deliberate assets.
