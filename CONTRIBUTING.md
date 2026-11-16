# Contributing to MarkGo

Use GitHub issues/PRs for proposals and fixes. Describe the user scenario and observable acceptance criteria. Architecture changes need explicit discussion; current approved decisions are in AGENTS.md.

For visible changes read [DESIGN.md](DESIGN.md), [implementation values](docs/design/design-system.md) and [review procedure](docs/design/review.md). Product rules, target measurements and current evidence are different.

## Development

Install rustup and native platform tools. The root Cargo package pins Rust and GPUI Kit. See [setup](docs/development/setup.md).

```sh
cargo run --locked -- --preview
make check
make render-check
```

Keep tests isolated from real documents and configuration. UI changes require real rendered evidence; native input/platform changes require host acceptance separately. List commands actually run and limitations.

## Pull requests

- Branch from current main and preserve local work.
- Explain concrete before/after behavior and relevant tests.
- Update docs and AGENTS.md when boundaries change.
- Attach rendered screenshots for visible work, identified by platform/size/scale.
- Include compatible licenses and notices for dependencies/assets.
- Exclude unrelated changes, credentials, local configuration and build output.

AI-assisted code follows the same review standard. Do not weaken tests to hide regressions, import unknown-origin code/assets or add undisclosed network behavior.

Contributions are under Apache-2.0. Only submit material with known compatible redistribution rights. Update THIRD_PARTY_NOTICES.md and THIRD_PARTY.md when appropriate.
