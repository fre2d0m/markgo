# MarkGo

[![CI](https://github.com/xddcode/markgo/actions/workflows/ci.yml/badge.svg)](https://github.com/xddcode/markgo/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.27.4%2B-00ADD8.svg)](go.mod)

**English | [简体中文](README.zh-CN.md)**

MarkGo is a free, open-source, local-first AI Markdown editor built with Go and [MyGo](https://github.com/egoist/mygo). Its native interface uses no Electron runtime, embedded browser, or JavaScript layer.

> MarkGo is at the engineering-foundation stage. The current source provides a native workbench, editable in-memory sample documents, live Markdown preview, and UI behavior tests. It does not yet open or save files or connect to AI models.

## Principles

- **Plain Markdown:** documents remain portable files, not a proprietary database.
- **Local-first:** core writing works without an account or a MarkGo server.
- **AI with consent:** model context and edits must be visible, reviewable, and undoable.
- **Native and focused:** a quiet writing surface, responsive interaction, and no required web runtime.
- **Free and open source:** no subscriptions, feature locks, advertisements, or telemetry.

## Current status

| Area | Status |
| --- | --- |
| Native three-pane workbench | Available |
| Editable in-memory Markdown documents | Available |
| Document switching and filtering | Available |
| Live source-mark preview and clickable task markers | Available |
| Selection context and AI panel shell | Available |
| Headless UI behavior tests | Available |
| File opening, saving, and recovery | Planned |
| Full CommonMark/GFM editing and rendering | Planned |
| Local model providers and reviewable AI diffs | Planned |
| Published packages and releases | Planned |

See [ROADMAP.md](ROADMAP.md) for direction and non-goals.

## Build from source

Requirements: Go 1.27.4 or newer, Git, and a desktop platform supported by MyGo. Make is recommended for the shared quality commands.

```sh
git clone https://github.com/xddcode/markgo.git
cd markgo
go run ./cmd/markgo
```

Development and verification:

```sh
go tool mygo dev .
make check
make race
make security
go tool mygo build
```

See the [development guide](docs/development/setup.md) and [testing strategy](docs/testing/strategy.md).

## Documentation

- [Documentation index](docs/README.md)
- [Architecture](docs/architecture/overview.md)
- [Design system](docs/design/design-system.md)
- [Contributing](CONTRIBUTING.md)
- [Security policy](SECURITY.md)
- [Changelog](CHANGELOG.md)

## License

MarkGo is licensed under the [Apache License, Version 2.0](LICENSE). See [NOTICE](NOTICE) and [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for attribution. The MarkGo name and logo are not licensed under Apache-2.0.
