# Development setup

## Requirements

- Go 1.27.4 or newer.
- Windows 10/11, macOS, or Linux with MyGo's native desktop dependencies.
- Git. Make is recommended for the shared quality commands.

The Go module pins MyGo, Staticcheck, and Govulncheck. A sibling MyGo checkout is not required. For local framework co-development, use an untracked `go.work`; do not commit workstation-specific `replace` directives.

## Run and test

```sh
go run ./cmd/markgo
go tool mygo dev .
CGO_ENABLED=0 go test ./... -count=1
go tool mygo build
```

Run the local quality gate before submitting a pull request:

```sh
make check
make race
make security
```

`make check` verifies formatting, module tidiness, Go vet, MyGo's stored-value checks, static analysis, and ordinary tests. Race and vulnerability checks are separate because they take longer or query vulnerability data.

## Repository layout

- `cmd/markgo`: executable entry point.
- `internal/app`: workbench state, views, theme, sample documents, and UI tests.
- `internal/mdeditor`: Markdown editor state and live-preview behavior.
- `internal/storage`: guarded UTF-8 file saving and folder enumeration.
- `resources`: application resources used by MyGo packaging.
- `docs`: architecture, design, development, and testing guides.
- `.github`: issue and PR contracts, dependency automation, and workflows.

Additional packages are created with their first real behavior, following [the architecture overview](../architecture/overview.md).

## Definition of done

A change is complete when its acceptance criteria are met, relevant tests and documentation are updated, required checks pass, and no unrelated or generated files are included. UI changes additionally need real visual evidence and behavior coverage. MarkGo does not claim cross-platform verification from a build alone; record the actual platforms exercised.
