# Current native implementation

Updated 2026-11-16. [DESIGN.md](../../DESIGN.md) is the authority; this is an implementation record, not native release acceptance.

| Surface | Current code |
| --- | --- |
| Workbench | 1320 × 840 default, 960 × 640 minimum; 48 DIP title and toolbar, 224 DIP sidebar, 28 DIP footer, 8 DIP outer gutter |
| Tabs | Persistent document identities, horizontal overflow/all-documents panel, 40 DIP close target, dirty dot, square lower seams, context actions |
| Prose | 18 DIP default, 14–24 range; 720 DIP column with 640/840 options; 48/32 DIP horizontal/top padding, 1.75 line height; H1/H2/H3 at 2/1.5/1.25 scale; system/serif choice |
| Controls | Original 24-unit vector designs, 18 DIP primary glyphs, 40 DIP targets, 8 DIP controls, 16 DIP floating panels |
| Appearance | 12 stable palette IDs, immediate Kit/Base update, 440 DIP gallery panel, three columns of 124 × 48 samples; system light/dark follows Paper/Graphite |
| Editing | Source-preserving headings/inline/links/lists/tasks/quotes/top-level code; sparse mapping; source history; staged composition; slash insertion |
| Files | Worker I/O, guarded snapshot saves, exclusive Save-as, undoable stale-checked reload, ordered bounded recovery and explicit exit decisions |
| Search/reading | Case/word/regex/scope search above the page; native tables/local images in optional reading; rendered source anchors |
| Feedback | Native hover/press/focus states, inline validation, local code-copy acknowledgment; no animated prose |

Sources: [workbench/views](../../src/desktop/views.rs), [writer](../../src/desktop/writer.rs), [themes](../../src/desktop/theme.rs), [projection](../../src/projection.rs), [recovery](../../src/desktop/recovery.rs).

Actual GPU screenshots are generated with `make render-check` under `build/gpui-evidence/`. The selected images in [images/gpui](images/gpui) are offscreen native renders. Event tests cover input/undo, tab preservation, slash, composition adapter geometry, twelve themes and dialogs. Native installed IME, OS clipboard/window/file integration, screen readers, fractional scaling and frame pacing remain separate acceptance. Shared motion timing targets are not yet implemented as a complete transition system.
