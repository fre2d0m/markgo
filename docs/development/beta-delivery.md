# 1.0 beta delivery plan

This is an engineering plan, not a list of shipped features. A beta tag means the writing, data-safety and native-platform gates below have passed. Passing CI or finishing the window shell is not enough.

## Product promise

A quiet, fully drawn desktop Markdown editor that feels immediate, keeps ordinary files intact, and lets a writer return to unfinished work. The writing page carries the identity. Navigation, formatting and status explain themselves through proportion, placement and useful feedback. AI, cloud accounts and decorative dashboards are outside this beta.

## Baseline and remaining release gates

| Area | Available foundation | Required before 1.0 beta |
| --- | --- | --- |
| Documents | Manual open/save/save-copy, folders, external-change guards, recent documents, session restore, local recovery and Keep drafts on exit | Optional guarded autosave; native recovery/exit acceptance, including OS shutdown and file-dialog cancellation |
| Editing | One Markdown source buffer, supported syntax hiding, lists/tasks/quotes, undoable formatting, find/replace | Directly editable heading sizes, paragraphs, lists, quotes and code; continuous selection and undo across blocks; stable Chinese IME |
| Images and tables | Formatted in Read mode | Local image insertion/paste/drop with relative paths; directly editable basic GFM tables and keyboard navigation |
| Navigation | File list, folders, outline, focus mode | Reliable source/read positions, complete keyboard flow, purposeful empty/error/loading states, discoverable row actions |
| Output | Portable Markdown files | HTML and PDF export, with image/Unicode/page-break acceptance fixtures |
| Presentation | Three palettes, typography and width preferences, self-drawn window controls | Coherent motion and feedback in every writing flow; Chinese/English interface and accessible names/focus; actual high-DPI acceptance |
| Release | Three-platform builds and headless tests | Windows-first native acceptance, then native acceptance for each additional platform advertised by the beta; documented installation/update path |

## Delivery order

### 1. Make everyday interactions coherent

- Sidebar motion must reverse from its current position; closing navigation immediately stops taking input. Retain document identity, selection, scroll and undo throughout.
- Tab indicators explain the destination; small content changes use a short fade, not a page entrance animation.
- Put document close and context actions by the document. The same unsaved guard applies to every close entry point.
- Confirm a successful current-version save briefly. Never show success for a failed save or an older snapshot while newer edits remain.
- The outline must navigate both editing and reading. Current-section emphasis in editing follows the caret. The current reader anchor API resolves repeated identical heading slugs to the first match; exact duplicate-heading navigation remains an editor-model acceptance case.
- Menus, search, formatting and preferences use the same self-drawn theme. Reduced-motion settings suppress movement.

Acceptance: a recorded, repeatable open → write → format → navigate → save → switch → close sequence, including interruption and keyboard-only use. Headless recordings demonstrate layout/state transitions; they do not certify a desktop's window system or IME.

### 2. Make unfinished writing safe

Recovery, recent documents and session restoration are implemented with source/version and process-termination tests plus rendered UI evidence. Native OS shutdown, actual file-dialog interactions, and target-platform filesystem failures remain acceptance work. Implement opt-in automatic disk saving only after these recovery gates. Recovery snapshots live in the application data directory, separately from the user's Markdown files. Keep source, path and version together; use atomic writes and bounded retention. Recovered content opens for review and never silently replaces a changed disk file. Save/discard decisions must clean up only the corresponding recovered revision.

Recent documents and session restoration use real local paths, with review and removal of unavailable entries. Next add opt-in autosave for named files through the existing snapshot/conflict pipeline. Untitled documents remain recoverable drafts. A canceled dialog, full disk, permission failure, crash during a write, and an external editor changing the same file are required tests.

Acceptance: forcibly terminate a sandbox instance with both named and untitled changes; recover them on relaunch; verify disk bytes and repeat the recovery without duplicates. Normal close, explicit discard and later edits must each preserve the intended version.

### 3. Resolve native editable typesetting

The current `ui/editor` adapter has one font size and line-height model. That is a limitation of the chosen path, not a limit on MyGo drawing. The pinned MyGo 0.9.0 also includes `ui/richtext.EditorBase`, per-heading styles, paragraph layout, attachments, rich selection, undo and IME support.

However, direct `ParseMarkdown(source).Markdown()` replacement is unsuitable for existing files. A local probe against the pinned code produced these results without making any edits:

| Input | Round-trip result |
| --- | --- |
| Heading and ordinary paragraph | Preserved in the tested fixture |
| Fenced block tagged `go` | Code language omitted |
| GFM table | Converted to tab-separated paragraphs |
| Reference link | Rewritten as an inline link |
| YAML front matter | Reinterpreted as a heading |

A runnable, sample-only prototype now explores a source-preserving adapter over native rich text. See the [native typesetting checkpoint](native-typesetting.md) for the implementation decision, commands, screenshots, tests and limitations. It directly edits headings/paragraphs, maps edits to the canonical source, and shares source-mode history. Incremental parsing, compact line maps and retained paragraph layouts reduce long-document input allocations; fixed-workload benchmarks and fresh-parse comparisons cover this checkpoint. The production editor is unchanged. Native IME/selection acceptance, desktop latency and sustained-memory checks remain required before integration. Do not introduce a WebView, replace the editor with a whole-document serializer, or maintain two competing authoritative document models.

The list checkpoint adds source-backed bullets, decimal numbering, tasks, subtree indentation, empty-item exit, task controls and shared clipboard/undo behavior in that isolated preview. Quotes and code are the next structured blocks; native-platform acceptance remains a separate gate. See the checkpoint's 69-state rendering matrix and list behavior coverage for the current evidence.

The chosen approach must pass these gates before integration:

1. Untouched source ranges, unsupported syntax, reference definitions, code languages and line endings remain intact.
2. Heading/body selection, CJK composition, delete across a block boundary, paste and undo form a continuous editing experience.
3. Formatting, typing and source-mode changes share a coherent undo history; toggling mode does not clear history or move the caret unexpectedly.
4. Image/table actions modify the same canonical source through one transaction, with relative paths and undo.
5. A 100,000-character document stays usable. Record the test hardware and actual native input latency; target p95 below 50 ms and opening below 2 s. These are acceptance targets, not existing measurements.

### 4. Finish output and platform acceptance

Add HTML/PDF export, localization, accessibility and packaging after the editor model stabilizes. Validate real Windows dragging, resize edges, Snap, multi-monitor DPI, shortcuts, clipboard, file dialogs and Chinese composition. Apply equivalent native checks before advertising macOS or Linux beta support. Record failures and platform-specific limits explicitly.

## After the beta

Advanced tables, math/diagram editing, footnote tools, typewriter mode, broader theme customization and reviewable AI work follow the core gates. Existing unsupported syntax must still remain intact. The beta does not include an account, sync service, marketplace or plugin host.

## Working rule

Each delivery names one user flow, its observable acceptance cases, its actual evidence and its remaining limitations. Keep changes in reviewable PRs. Update this plan as evidence changes; do not substitute cosmetic PR completion for a release milestone.
