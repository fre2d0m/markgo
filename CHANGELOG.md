# Changelog

## Unreleased

- Restore tab selection hierarchy, compact aligned command/document popovers, native UI font choice on Windows and restrained hover/press/popup motion with reduced-motion support.
- Retain canonical file identities through recovery to avoid plain/verbatim Windows path duplicates; keep distinct unsaved buffers and display readable paths.
- Separate document-tab/action hit targets from native titlebar dragging; reduce dialog construction stack usage and set a Windows executable stack reserve.
- Keep incomplete heading markers visible until a separating space; Backspace at a heading's start removes its heading format in one undoable edit.
- Embed window-control vectors alongside the other control icons; add painted-icon checks and cross-platform UI event regressions.
- Rust/GPUI Kit is the sole root application/build package.
- Source-preserving native headings, inline formats/links, lists/tasks, quotes and fenced code, including composition and paired delimiter repair.
- Independent tabs, overflow/context actions, folder filter/collapse/refresh, outline and recent files.
- Guarded asynchronous files, undoable reload, bounded ordered recovery and explicit exit decisions.
- Slash insertion, regex/case/word/scope search, source view and native reading.
- Twelve themes, system appearance, prose preferences and original vectors under DESIGN.md.
- Unified Cargo checks, dependency automation, native-host packaging and actual GPU interaction verification.

Release and platform limits remain explicit in [beta gates](docs/development/beta-delivery.md).
