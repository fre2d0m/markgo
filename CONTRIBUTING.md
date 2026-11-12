# Contributing to MarkGo

Thank you for contributing. MarkGo uses GitHub Issues, Pull Requests, Actions, and Releases for collaboration.

## Before making a change

1. Search existing issues and pull requests.
2. Open an issue for a feature, compatibility problem, or substantial fix.
3. Describe the user scenario and observable acceptance criteria.
4. Discuss the architecture first when changing document storage, editor behavior, AI context boundaries, or release workflows.

Small documentation and obvious defect fixes still need a pull request and CI.

## Development setup

Use Go 1.27.4 or newer. MyGo and development tools are pinned by `go.mod`; no sibling framework checkout is required.

```sh
go run ./cmd/markgo
go tool mygo dev .
make check
make race
```

See [docs/development/setup.md](docs/development/setup.md) for platform notes.

## Pull requests

- Create a focused branch from an up-to-date `main`.
- Use a Conventional Commit title, such as `feat: open Markdown files`.
- Link the issue and describe the actual behavior change.
- Add or update tests for success, failure, and boundary behavior.
- Update documentation and `AGENTS.md` when applicable.
- Attach real screenshots for visual changes and identify the platform and display scale.
- List new dependencies and their licenses.
- Do not include unrelated formatting, local configuration, or build output.

All required checks must pass. Do not claim an unrun command or unimplemented feature as complete.

## AI-assisted contributions

AI-generated changes follow the same review standard as other contributions. Never weaken tests to hide a regression, copy code with unknown provenance, or add telemetry or undisclosed network behavior.

## Tests and quality

At minimum, run:

```sh
make check
make race
```

UI changes need `ui.Tester` behavior coverage and real visual verification. Parser changes need malformed-input and boundary coverage. Native integration changes need corresponding platform verification.

## License and attribution

By submitting a contribution, you agree that it is licensed under Apache-2.0. Only submit code, documents, icons, fonts, or other assets whose origin and redistribution terms are known and compatible. Update `THIRD_PARTY_NOTICES.md` when distributed third-party material changes.
