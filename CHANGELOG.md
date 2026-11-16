# Changelog

All notable changes to MarkGo are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- A root design contract, documented reference decisions and a state-based review matrix; current code measurements remain a separate implementation inventory. This documentation change does not alter the application UI.

- A caret-anchored slash menu for paragraphs, headings 1–3, lists, tasks, quotes and code; Chinese/English filtering, keyboard choice, cancellation and single-step undo.
- An empty-paragraph hint, native vector block icons, whole-row popup sizing and caret-safe upward placement.

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

- Real documents now default to source-preserving native visual editing, with variable-size headings and editable structured/inline formatting. Files, tabs, save snapshots, recovery, outline and undo use the same Markdown buffer.
- Removed the persistent Write/Read switch; visual editing, Markdown source and optional reading live in the View menu. Existing formatting tools remain available.
- Find/replace stays in visual editing, including regex, case, word and selection scope; source-byte anchors retain replaced text at scope edges through edits and undo. The panel reserves space above the page, and the current match stays highlighted while the query field has focus.
- Retained each visited document's visual viewport across tab/view switches and prevented navigation from committing unfinished input-method candidates. Native platform acceptance and directly editable images/tables remain pending.

- Refined document tabs as compact full-height controls with square lower corners and theme-driven active feedback, plus a separate rounded writing page and quieter sidebar selection.
- Unified native buttons, inputs and floating panels; refined sidebar type, segmented view controls, dialog spacing and stable footer metrics.
- Added restrained hover/press feedback throughout formatting, appearance, search and dialog actions, respecting reduced motion.
- Tuned primary and secondary text contrast across all twelve palettes, including hovered and pressed surfaces.
- Replaced the application icon and configured its high-resolution packaging asset.
- Rebuilt the workspace around a 48 DIP self-drawn tab bar, a 216 DIP navigation rail and a distinct paper surface; retained OS resize edges, close protection and maximize state.
- Refined Paper, Warm paper, and Graphite; reduced wordmark and icon weight, softened sidebar selection, and added active-tool states and interruptible, reduced-motion-aware panel transitions.
- Kept keyboard focus in the file list while moving between documents; Enter begins editing the selection.
- Moved writing modes and common inline actions into a shared 52 DIP page toolbar in the main application and Writing Preview, with labeled heading, list and quote actions in the Formatting panel; unified controls and find fields around 40 DIP targets.
- Refined sidebar, outline, reading typography, neutral syntax colors, hover feedback, and save-status presentation.
- Start with a blank editable document; examples are available from the document menu.

- Removed AI panel placeholders from the writing interface; AI work is deferred until the base editor is ready.

### Fixed

- Page toolbar dividers now stop inside both paper edges, with matching inset and contrast in the main application and Writing Preview.
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
