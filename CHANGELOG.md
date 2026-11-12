# Changelog

All notable changes to MarkGo are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Open, create, save, and save copies of local Markdown files; browse and filter local folders.
- Native file-association and file-drop requests, queued while another file operation runs.
- Disk snapshots, guarded saves, external-change warnings, reload protection, and unsaved-change dialogs with save-and-close actions.
- Parser-derived live-preview markers, headings outline, and visible-text statistics.
- Undoable formatting controls, list/task/quote continuation, and a compact native find-and-replace panel.
- Native reading mode, per-document local-image loading, focus mode, and centered system-font editing.
- File integrity, saved-version, reload, Unicode, composition, and editing regression tests.

### Changed

- Removed AI panel placeholders from the writing interface; AI work is deferred until the base editor is ready.

### Foundation

- Initial Go and MyGo native application structure.
- Three-pane workbench with document filtering, switching, and editor status.
- Editable in-memory Markdown samples and live source-mark preview.
- Clickable task markers, selection context, and an AI panel shell.
- Headless UI and preview behavior tests.
- Apache-2.0 licensing, contribution and security policies, bilingual documentation, and repository automation.
