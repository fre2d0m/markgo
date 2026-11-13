# Changelog

All notable changes to MarkGo are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Paper, Warm paper, and Graphite themes with a system-following option, local preference persistence, text-size controls, Modern/Book typography, and three page widths.
- Original monochrome interface icons and a paper-and-ink application icon.
- Appearance and sidebar shortcuts; source/selection/scroll/undo and preference-persistence regression coverage.

- Open, create, save, and save copies of local Markdown files; browse and filter local folders.
- Native file-association and file-drop requests, queued while another file operation runs.
- Disk snapshots, guarded saves, external-change warnings, reload protection, and unsaved-change dialogs with save-and-close actions.
- Parser-derived live-preview markers, headings outline, and visible-text statistics.
- Undoable formatting controls, list/task/quote continuation, and a compact native find-and-replace panel.
- Native reading mode, per-document local-image loading, focus mode, and centered system-font editing.
- File integrity, saved-version, reload, Unicode, composition, and editing regression tests.

### Changed

- Combined the native title bar and application header into one 48 DIP self-drawn bar; retained OS resize edges and wired custom window controls through the normal close protection and maximize state.
- Refined Paper, Warm paper, and Graphite; reduced wordmark and icon weight, softened sidebar selection, and added active-tool states and interruptible, reduced-motion-aware panel transitions.
- Kept keyboard focus in the file list while moving between documents; Enter begins editing the selection.
- Replaced stacked toolbars with one header and an on-demand formatting panel; unified controls and find fields around 40 DIP targets.
- Refined sidebar, outline, reading typography, neutral syntax colors, hover feedback, and save-status presentation.
- Start with a blank editable document; examples are available from the document menu.

- Removed AI panel placeholders from the writing interface; AI work is deferred until the base editor is ready.

### Fixed

- Switching a theme no longer caches the previous theme's editor colors under the new choice.

### Foundation

- Initial Go and MyGo native application structure.
- Three-pane workbench with document filtering, switching, and editor status.
- Editable in-memory Markdown samples and live source-mark preview.
- Clickable task markers, selection context, and an AI panel shell.
- Headless UI and preview behavior tests.
- Apache-2.0 licensing, contribution and security policies, bilingual documentation, and repository automation.
