# MarkGo roadmap

The roadmap describes direction, not release promises. The [beta delivery plan](docs/development/beta-delivery.md) defines the engineering order and acceptance gates; features below are not all implemented.

## M0 — Engineering foundation

- Go/MyGo application boundaries and a native workbench shell.
- Editable sample documents and initial live Markdown preview.
- Apache-2.0 licensing, repository health files, and bilingual project overview.
- Quality, cross-platform, security, and dependency automation.
- Design tokens and UI behavior verification.

## M1 — Real documents

- Open and save Markdown files and local folders.
- Atomic saving, autosave, crash recovery, and external-change handling.
- Recent documents, file association, and unsaved-change workflows.
- Document fixtures and end-to-end file lifecycle tests.

## M2 — Writing experience

- Incremental CommonMark/GFM parsing and complete source/live-preview behavior.
- Images, tables, code blocks, footnotes, and math rendering.
- Outline, search, focus and typewriter modes.
- HTML/PDF export and accessible, localized interface foundations.
- Large-document benchmarks and input-latency budgets.

## M3 — Release quality

- Verified Windows, macOS, and Linux packages for each platform advertised by the release.
- Reproducible release provenance and signing where credentials are available.
- Real-platform compatibility, accessibility, performance, and recovery testing.

## After 1.0 beta — Reviewable AI editing

- Ollama, LM Studio, and other user-configured OpenAI-compatible providers.
- Selection rewrite, summarization, restructuring, and translation.
- Explicit context selection, streaming responses, cancellation, and redaction.
- Preview diffs, per-hunk decisions, and undoable accepted edits.
- Local document indexing without a mandatory account or MarkGo server.

## Explicit non-goals

- Paid features, subscriptions, advertisements, or telemetry.
- A proprietary document format or mandatory cloud workspace.
- An AI chat panel that bypasses source review and undo.
- A general-purpose IDE or an unbounded third-party plugin system in the initial releases.
