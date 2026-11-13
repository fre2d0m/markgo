# Changelog

All notable changes to MarkGo are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- A native document tab strip with independent writing positions and undo, unsaved-state feedback, guarded closing, full-path tooltips, context actions, Ctrl+Tab cycling and a bounded all-documents menu.
- Shared sample tabs in Writing Preview; closed sample edits remain available in memory.
- Sakura, Sage, Mist, Lavender, Sand, Pearl, Midnight, Pine and Plum writing themes, alongside Paper, Warm paper and Graphite.
- A compact twelve-theme gallery with a selected check, keyboard navigation and an explicit close action.
- Local recovery checkpoints for named and untitled drafts, previous-session restoration, and a searchable recent-document history.
- Self-drawn exit choices to keep drafts, save all, discard or cancel; final checkpoints complete before closing, with visible failure handling.
- Recovery review and save-copy guidance for changed or unavailable originals; current-version backup status and safe history removal.
- Process-termination, stale-write, discard, line-ending, recovery-failure and keyboard-focus regression tests, plus theme/state screenshots.

- Interruptible sidebar motion, a moving Files/Outline indicator, file-row close and context actions, and brief confirmation after a current-version save.
- A beta delivery plan with data-safety, native editor, output and platform acceptance gates.
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

- Unified native buttons, inputs and floating panels; refined sidebar type, segmented view controls, dialog spacing and stable footer metrics.
- Added restrained hover/press feedback throughout formatting, appearance, search and dialog actions, respecting reduced motion.
- Tuned primary and secondary text contrast across all twelve palettes, including hovered and pressed surfaces.
- Replaced the application icon and configured its high-resolution packaging asset.
- Rebuilt the workspace around a 60 DIP self-drawn tab bar, a 240 DIP navigation rail and a distinct paper surface; retained OS resize edges, close protection and maximize state.
- Refined Paper, Warm paper, and Graphite; reduced wordmark and icon weight, softened sidebar selection, and added active-tool states and interruptible, reduced-motion-aware panel transitions.
- Kept keyboard focus in the file list while moving between documents; Enter begins editing the selection.
- Moved writing modes and common inline actions into a 56 DIP page toolbar, with labeled heading, list and quote actions in the Formatting panel; unified controls and find fields around 40 DIP targets.
- Refined sidebar, outline, reading typography, neutral syntax colors, hover feedback, and save-status presentation.
- Start with a blank editable document; examples are available from the document menu.

- Removed AI panel placeholders from the writing interface; AI work is deferred until the base editor is ready.

### Fixed

- Keyboard theme selection and dismissal now return to the preserved writing caret in both editing modes.
- Link validation marks the corresponding text or address field and leaves the document unchanged until Apply succeeds.
- Apply the application icon to native windows and taskbar buttons in development and packaged builds.
- Outline navigation in reading mode now addresses the reading view; editing mode emphasizes the caret's current section.
- Switching a theme no longer caches the previous theme's editor colors under the new choice.

### Foundation

- Initial Go and MyGo native application structure.
- Three-pane workbench with document filtering, switching, and editor status.
- Editable in-memory Markdown samples and live source-mark preview.
- Clickable task markers, selection context, and an AI panel shell.
- Headless UI and preview behavior tests.
- Apache-2.0 licensing, contribution and security policies, bilingual documentation, and repository automation.
