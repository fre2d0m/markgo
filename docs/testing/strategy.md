# Testing strategy

## Headless behavior tests

MyGo's `ui.NewTester` drives the real application view without opening a desktop window. Current tests cover document switching and filtering, live-preview and AI panel toggles, selection extraction, and caret-sensitive Markdown replacements.

Test observable behavior and persistent state rather than incidental pixel coordinates. Keep stable control labels and identities.

## Editor and parser tests

Preview replacements must preserve the source buffer and reveal marks on active lines. Future parsers need valid, malformed, boundary, and fuzz coverage, including CJK input, UTF-8 offsets, incomplete syntax, nested structures, and large documents. Golden rendering tests should complement rather than replace editing behavior tests.

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

Build output stays under ignored `build/` directories and is not committed. Temporary screenshots and scripts used only for verification should be removed; documentation screenshots are deliberate assets.
