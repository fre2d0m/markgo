# Current native implementation

Updated 2026-11-16. [DESIGN.md](../../DESIGN.md) is the authority; this is an implementation record, not native release acceptance.

| Surface | Current code |
| --- | --- |
| Workbench | 1320 × 840 default, 960 × 640 minimum; 48 DIP title and toolbar, 224 DIP sidebar, 28 DIP footer, 8 DIP outer gutter |
| Tabs | Persistent document identities, 2 DIP accent rule and paper surface for the active tab, muted inactive labels, 100 ms hover color, 40 DIP close target, dirty dot and square lower seams |
| Prose | 18 DIP default, 14–24 range; 720 DIP column with 640/840 options; 48/32 DIP horizontal/top padding, 1.75 line height; H1/H2/H3 at 2/1.5/1.25 scale; system/serif choice |
| Controls | Original 24-unit vector designs, 18 DIP primary glyphs, 40 DIP targets, 8 DIP controls, 16 DIP floating panels |
| Appearance | 12 stable palette IDs, immediate Kit/Base update, 440 DIP gallery panel, three columns of 124 × 48 samples; system light/dark follows Paper/Graphite |
| Editing | Source-preserving headings/inline/links/lists/tasks/quotes/top-level code; sparse mapping; source history; staged composition; slash insertion |
| Files | Worker I/O, guarded snapshot saves, exclusive Save-as, undoable stale-checked reload, ordered bounded recovery and explicit exit decisions |
| Search/reading | Case/word/regex/scope search above the page; native tables/local images in optional reading; rendered source anchors |
| Menus | 304 DIP view/format/document popovers; 360 DIP document switcher with 56 DIP file rows; 400 DIP recent list. 8 DIP padding, 40 DIP command rows, 2 DIP gaps, fixed icon/label/shortcut columns; outside dismissal and Escape restore writing |
| Feedback | Icon controls: 100 ms hover color, 90 ms glyph press to 0.96 with a fixed 40 DIP target. Floating surfaces: 130 ms entry with at most 4 DIP movement; 90 ms departure with at most 2 DIP movement. Closing disables panel actions immediately and allows writing and reopening. Reduced motion removes movement/scaling and resolves colors immediately; no animated prose |
| Windows UI font | Microsoft YaHei UI when installed, otherwise Segoe UI; separate from the user's prose font preference |

Sources: [workbench/views](../../src/desktop/views.rs), [writer](../../src/desktop/writer.rs), [themes](../../src/desktop/theme.rs), [projection](../../src/projection.rs), [recovery](../../src/desktop/recovery.rs).

Actual GPU screenshots are generated with `make render-check` under `build/gpui-evidence/`. The selected images in [images/gpui](images/gpui) are offscreen native renders. Event tests cover input/undo, tab preservation, slash, composition adapter geometry, twelve themes and dialogs. Popover tests cover intermediate frames, reversing a departure, immediate writing focus and reduced motion; menu tests check equal row geometry in Paper, Sakura and Graphite. Native installed IME, OS clipboard/window/file integration, screen readers, fractional scaling and frame pacing remain separate acceptance. Sidebar geometry does not yet animate.

The workbench polish captures include the [document switcher](images/gpui/tabs-paper.png), [view menu](images/gpui/view-paper.png), [dark view menu](images/gpui/view-graphite.png), [Sakura switcher](images/gpui/tabs-sakura.png) and [format menu](images/gpui/format-paper.png), at 960 × 640 DIP and 2× scale. These are actual GPU renders of isolated fixtures, not Windows desktop captures.
