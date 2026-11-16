# MarkGo design contract

Version 1 · 2026-11-16 · Native desktop writing

MarkGo should feel like a good sheet of paper with capable tools close at hand. Its character comes from typography, proportion and dependable behavior. Writing stays prominent; the workbench stays quiet.

## Authority and scope

Read this file before designing or changing a visible MarkGo surface. It is the single authority for product design, alongside the engineering constraints in [AGENTS.md](AGENTS.md). It applies to the ordinary application and shared controls in Writing Preview. The preview remains an isolated sample application.

Distinguish three kinds of information:

| Kind | Meaning | Where it lives |
| --- | --- | --- |
| Product contract | Requirements to preserve: native rendering, writing first, document safety, established navigation and accessible interaction | This file |
| Design target | The intended visual rules and starting measurements for the next implementation pass; not a claim that the current application already meets them | This file |
| Implementation record | Values and capabilities currently present in code, with dated evidence and remaining acceptance limits | [Current implementation](docs/design/design-system.md) |

The user's explicit decisions take precedence. Reference websites, generic UI skills and old screenshots do not override this contract. If a target needs adjustment, record the reason, update the shared rule and verify its affected surfaces together. Do not silently introduce a second scale for one dialog.

This document establishes the design direction. Publishing it does not certify a visual redesign or a beta release. Use the [review procedure](docs/design/review.md) to verify implementations.

## Product commitments

- Fully drawn native UI; no WebView or web rendering layer. The current client uses MyGo; the [Rust/GPUI migration](docs/architecture/gpui-migration.md) preserves this same design contract.
- A local Markdown document is the center of the application. AI integration remains deferred.
- Open multiple documents with persistent tabs. Keep the established full-height tab shape and square lower corners; do not turn tabs into floating pills.
- Default to direct visual editing. Markdown source and optional reading belong in the View menu, not a permanent Write/Read switch.
- Keep the existing tool entry points recognizable. Improve their proportion and states without adding a second toolbar or duplicate navigation rail.
- Preserve document text, selection, undo and editing position through presentation changes. Never imply that a visual action saved or synchronized a file when it did not.
- Provide at least twelve coherent themes, including Sakura. A theme is one color environment, not a different component system.

## Visual direction

Claude's paper-and-ink relationship is the primary reference; Linear's restrained surface hierarchy is a supporting reference. Mintlify contributes document/navigation relationships. These are adaptations of third-party website analyses, not official product specifications. See [reference decisions and attribution](docs/design/references.md).

Use neutral white or lightly tinted paper, legible ink and one restrained accent per theme. Paper may remain neutral white; warmth does not require every theme to be beige. Dark themes use distinct, quiet surface steps rather than a near-black marketing backdrop.

The visual order is: document content, current document identity, immediate editing tools, navigation, then supporting status. A bright selected file, thick toolbar or decorative panel must not compete with the paragraph being written.

Do not add decorative gradients, glow, glass, scenery, mascots, oversized welcome cards, illustrated empty states or colored blocks behind ordinary prose. Do not copy a brand's logo, proprietary typeface, hero banner, pricing card or call-to-action treatment. There is no marketing hero inside the editor.

## Workbench and density

The window has one compact title/tab strip, a supporting sidebar, a writing surface with its existing tools, and a quiet footer. The OS retains resize edges, window shadow and platform window behavior; self-drawn chrome must preserve close guards and maximize/Snap behavior.

The following measurements are **starting targets for the next rendered implementation**, not frozen legacy values. Changes must retain non-overlapping targets and pass the narrow-window review. The implementation record deliberately lists different values where migration is still pending.

| Element | Starting target | Design reason and boundary |
| --- | --- | --- |
| Title/tab strip | 48 DIP high | Compact identity and navigation; keep window controls reachable in Focus mode |
| Page toolbar | 48 DIP high; 20 DIP horizontal inset | Fit 40 DIP controls with 4 DIP above/below; divider ends at both content insets |
| Sidebar | 224 DIP nominal | Accommodate icons, readable labels and fixed close targets; long names ellipsize, not squeeze the editor |
| Outer page gutter | 8 DIP | Separate navigation from the writing surface without stacking frames |
| Footer | 28 DIP high | Quiet, readable status; changing counts do not move adjacent groups |
| Writing column | 720 DIP default; retain 640/720/840 choices | Support long-form reading and user preference; shrink to available width |
| Page breathing room | 48 DIP horizontal, 32 DIP top as a starting point | Enough room for text, without a large empty banner above the first paragraph |
| Action/menu target | At least 40 × 40 DIP | Small glyphs can have generous targets; targets must not overlap |

The smallest supported window remains 960 × 640 DIP. Keep window controls, active document identity and essential actions visible there. Use the existing overflow mechanisms instead of shrinking text or hiding the active document. Do not force a simultaneous file tree, floating outline and third sidebar into a narrow window.

Use a 4 DIP spacing rhythm: 4, 8, 12, 16, 20, 24, 32 and 48. Optical corrections and native text metrics may need exceptions; explain them at their shared definition. Equal roles use equal spacing. A pair of related controls stays closer than unrelated groups.

## Typography

Interface typography and document typography have separate roles. Appearance preferences for prose must not resize the toolbar or menus.

| Role | Starting target | Rules |
| --- | --- | --- |
| Document/sidebar titles | 14 DIP, 400–500 | Clear at normal display scale; preserve full names through tooltips/accessibility |
| Menu and button labels | 13 DIP, 400–500 | Consistent baseline, no bolding every control |
| Supporting metadata | 12 DIP, 400 | Secondary by color and position; never make essential status illegible |
| Brand label | 15 DIP, about 600 | Support recognition without becoming a heading above the document |
| Prose | 18 DIP, 400, 1.75 line height | Preserve the user's 14–24 DIP preference; inspect Chinese and mixed-language paragraphs |
| H1 / H2 / H3 | 2 / 1.5 / 1.25 times prose, about 600 | Paragraph hierarchy, not website display typography; verify spacing and wrapped headings |
| Code | About 90% of prose size, system monospace | Literal spacing stays meaningful; code controls remain in the interface scale |

Use the platform system sans for interface text. Offer system or serif document text without importing a brand font. Check actual Chinese fallback, punctuation, Latin text, digits and emoji on supported platforms. Do not apply negative tracking to Chinese prose or blindly reuse a website's display-font tracking.

Headings belong more closely to the following paragraph than to the preceding one. Lists use hanging indents; continuation lines align with their text. Quotes use a quiet rule. Code uses a restrained surface. Neither headings nor normal paragraphs need their own cards.

Dynamic counts use tabular digits and reserved widths. Selected text must remain legible across bold, code and links. A font-size or line-height adjustment must preserve meaningful caret/selection position even when text reflows.

## Color and themes

Use semantic roles, not arbitrary colors at call sites:

| Role | Purpose |
| --- | --- |
| Paper | Editable/readable document surface |
| Rail | Supporting window/sidebar surface |
| Hover / Pressed | Distinct, restrained feedback on an existing control |
| Ink / Muted ink | Primary and secondary text, both readable |
| Edge | Separation where tone alone is insufficient |
| Accent / On accent | Primary action, current control and keyboard focus |
| Selection | Text or item selection, retaining legible foreground |
| Danger / Warning / Success | Actual outcomes or risks, always accompanied by words or a meaningful icon |

All twelve themes use the same dimensions, icon family, typography roles and state meanings. Keep existing persisted IDs. Theme choice must update the whole application and compatible native-dialog appearance while leaving document data untouched.

Primary and secondary text must achieve at least 4.5:1 contrast on their actual normal, hover and pressed surfaces. Inspect composited selection and focus states as well; a passing token pair alone is insufficient. Color cannot be the only indicator of selection, failure or an unsaved document.

Use accent sparingly. Ordinary prose and syntax remain neutral ink. Sakura uses a pale paper/rail tint and controlled rose accent, not saturated pink buttons everywhere. Apply the same discipline to every theme. Focus must remain easy to see in dark and light modes.

## Surfaces, borders and icons

- Separate the workbench through tone and spacing first. Use thin edges to clarify boundaries; reserve shadow for surfaces that actually float above content. Do not stack toolkit and custom shadows.
- Use one shared radius scale: 4 DIP for small in-text details, 8 DIP for controls, 16 DIP for floating panels. With 8 DIP panel padding and 8 DIP inner controls, the corners are concentric. Optical exceptions must not spread into competing scales.
- Preserve square lower tab corners and the user's established window/page silhouette. Do not round every seam because a generic design guide recommends concentric corners.
- Popovers have one boundary and one subdued shadow. Code, fields and nested sections do not each need another raised card. A document surface is stationary.
- Use one native vector icon family with a 24-unit grid, approximately 1.65-unit rounded strokes and 18 DIP standard glyphs. Align asymmetric glyphs optically; share corrected assets rather than adding arbitrary offsets at every call site.
- Keep the existing application icon distinct from control icons. Do not replace controls with emoji or font-dependent symbols. H1/H2/H3 and familiar text search options may remain text.
- Icon-only controls require accessible names and useful tooltips. Hover decoration must not move their target or neighboring labels.

### Floating layout scale

| Surface | Starting anatomy | Placement and overflow |
| --- | --- | --- |
| Block/command menu | 280 DIP width, 8 DIP padding, 40 DIP rows, 2 DIP row gaps; 18 DIP icon and 12 DIP icon/label gap | Anchor to the initiating control or caret; fit complete rows, flip when needed and scroll the result list rather than clip labels |
| Utility panel | 16 DIP padding, 12 DIP section gaps, 15 DIP heading and 40 DIP fields/actions | Width follows the content, not a full-window overlay; preserve the anchor and enough room for errors; clamp to window bounds |
| Confirmation/recent dialog | Up to 560 DIP width, 24 DIP padding, 24 DIP section gaps and 8 DIP action gaps | At least 24 DIP from window edges; long content scrolls without hiding actions; preserve modal focus and cancellation |
| Tooltip | 12 DIP text and compact padding; wrap long paths instead of silently truncating them | Informational, never interactive; do not obstruct the next target or substitute for a field error |

Keep icons, labels and shortcut hints in stable columns. The footer is informational; do not place a 40 DIP action into its shorter row. A control that needs more space must receive it explicitly, not overlap an adjacent target. If a specific panel needs an exception to this scale, share and document that exception instead of spreading one-off dimensions.

## Component and interaction contracts

These behaviors are part of the design, not optional finishing work. They apply to implemented features; this file does not authorize placeholder controls for future capabilities.

| Surface | Required behavior |
| --- | --- |
| Buttons and fields | Specify normal, hover, pressed, keyboard focus, disabled and invalid states. Press-and-drag outside cancels. Validation explains the problem next to the field and preserves input. Busy feedback reflects real work and prevents duplicate activation without blocking unrelated writing. |
| Document tabs | Active, hovered, focused and dirty are distinguishable. Reserve the close target so the dirty dot/cross never shifts the title. Context actions address that tab. Long names expose full paths; duplicate names are distinguishable in overflow. Ctrl+Tab cycling and active-tab reveal retain independent caret, scroll and undo. Closing an inactive tab keeps the current document; closing the last clean tab creates a fresh document. |
| Sidebar and outline | Selection and keyboard focus are distinct. Arrows/Home/End navigate without stealing writing focus; Enter starts editing. Keep labels aligned with their icons. Hidden controls stop accepting input. Current-section indication remains quiet; reading navigation uses reader anchors. |
| Formatting | Toolbar, keyboard and slash actions share one document operation. Preserve selection, perform one undoable edit and return focus to the document. Ordinary formatting gets local feedback, not a success toast. |
| Slash menu | Open on a typed slash in an empty ordinary paragraph. Keep horizontal anchoring stable, flip above without covering the caret, and show complete rows. Chinese/English search, empty results, arrows, Enter/Tab and pointer choice work together. Escape/outside dismissal retains literal input; unfinished composition takes precedence. No trigger for pasted paths, code or metadata. Only implemented commands appear. |
| Search/replace | Query, current result and replacement feedback remain visible together. The panel cannot cover the matched paragraph, including after expanding Replace. Keep the current match visible when query focus hides native selection. Invalid patterns are explained; scoped replacement never silently expands outside its passage. |
| View menu | The current mode is checked. Switching views preserves text/history and the mapped selection; document position remains useful. Do not use a reading screenshot to claim a block is directly editable. |
| Appearance | Choices apply immediately to the whole interface. Keyboard navigation and selection are visible; Escape/Close returns to writing. Preserve source, selection and undo; typography may reflow content. Appearance and Formatting cannot leave overlapping interactive panels. |
| Link and code controls | Keep writing selection while editing a link or language. Error states mark the relevant field. Apply, Cancel and Escape return focus predictably; editing a link never implicitly opens it. Copy receives brief, local feedback without moving the page. |
| Lists, tasks and quotes | Hanging indents and decimal numbering remain aligned through wrapping and long ordinals. Task hit targets do not overlap at the smallest prose size; toggling preserves writing focus. Enter continues structure and an empty item/quote exits predictably without changing unrelated source. |
| Save and recovery status | Reflect the actual saved buffer version. A short acknowledgment must yield immediately to new edits or errors. Distinguish Saved locally from Draft backed up; neither means cloud sync. Failed operations keep text and a usable recovery action. |
| Dialogs and close guards | Preserve the underlying selection and restore focus. Destructive actions are clearly worded and separate from the safe primary action; cancellation is easy. Exit offers the actual keep/save/discard choices and stays open on failure. Background controls do not activate through a modal. |
| Native input | Preserve text-service and input-method behavior. Presentation changes must not commit unfinished composition or move the candidate/caret geometry incorrectly. Save/recovery capture committed text without cancelling ongoing composition. Navigation follows the existing preedit cancellation policy. |

Tooltips supplement controls; they do not replace visible status or cover the next target. Empty document hints should be short and disappear naturally as writing begins. No-results states keep the query and offer a clear way back.

Use short action verbs and concrete status language. Preserve filenames exactly and show the relevant path when names collide. Errors explain what happened and the available next action without exposing implementation jargon. Keep a stable place for error text where practical; do not announce routine typing and formatting through repeated global notifications.

## Motion and feedback

Motion explains an interaction and stays peripheral to prose. Do not animate typing, paragraphs, the writing page or initial document restoration. Do not stagger menu rows, bounce tabs or blur text to create an impression of polish.

| Transition | Starting timing | Boundary |
| --- | --- | --- |
| Hover/active color | 100 ms | Only the affected control; focus remains visible immediately |
| Isolated button press | 90 ms, scale 1 → 0.96 | Visual response only; do not scale a whole tab strip, row layout or document |
| Floating panel | 130 ms in / 90 ms out, at most 4 / 2 DIP translation | Interruptible; closing disables input immediately |
| Sidebar | 180 ms ease-out | Preserve editor identity and editing position; can reverse midway |

Respect reduced motion: remove translation and scale while retaining clear state changes. Feedback never delays an action, intercepts typing or promises an outcome before it occurs. Native frame pacing and text rendering require native acceptance; headless animation frames do not prove them.

## Applying and reviewing a change

Start with one representative native writing scene containing a Chinese/English document, long tab names and real editor state. Include normal, empty, active, focused and error cases for the changed component. Compare at 960 × 640 and a comfortable desktop size, with 1×, 1.5× and 2× scale where relevant.

Use Paper, Warm paper, Sakura and Graphite to inspect tone and legibility; shared color/control changes must also check all twelve palettes. Preserve existing semantic, data-safety and platform behavior while changing presentation.

Implement, capture the actual view, inspect it, fix the mismatch and capture again before propagating a token change. Report which evidence is source inspection, behavior tests, rendered screenshots or native desktop observation. Do not claim native acceptance from a build, a mockup or headless output.

Before delivery, update the [implementation record](docs/design/design-system.md) only for changes that actually shipped in code. Use the [review matrix](docs/design/review.md) for the affected flows, and keep native or unimplemented behavior explicitly outstanding. A previous screenshot is historical evidence, not automatic approval of its measurements.
