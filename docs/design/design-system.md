# Current design implementation

Recorded from application code at `075b696` on 2026-11-16.

This is an implementation inventory, not the design authority. Read [DESIGN.md](../../DESIGN.md) for the intended product rules and starting visual targets. The design-contract change edits documentation only; the values below have not been migrated by publishing that contract.

## Current workbench

| Area | Current implementation | Source |
| --- | --- | --- |
| Window | 1320 × 840 default; 960 × 640 minimum; hidden native title/buttons with OS resize frame retained | [window.go](../../internal/app/window.go) |
| Title and tabs | 48 DIP strip; 172 × 48 DIP tabs; 8 DIP upper corners and square lower corners; 40 DIP close target | [document_tabs.go](../../internal/app/document_tabs.go) |
| Brand | Existing 28 DIP icon and 15 DIP label | [document_tabs.go](../../internal/app/document_tabs.go) |
| Page toolbar | 52 DIP; 20 DIP horizontal inset and 4 DIP vertical padding; 1 DIP divider at 40% theme-edge opacity | [page_toolbar.go](../../internal/app/page_toolbar.go) |
| Sidebar | 216 DIP; 13 DIP document/outline labels and 11 DIP supporting labels | [window.go](../../internal/app/window.go), [chrome.go](../../internal/app/chrome.go) |
| Writing surface | 8 DIP outer gutter; 10 DIP corners; 48 DIP horizontal and 38 DIP vertical content padding | [app.go](../../internal/app/app.go), [document_tabs.go](../../internal/app/document_tabs.go) |
| Footer | 30 DIP; reserved count widths and tabular figures | [app.go](../../internal/app/app.go) |
| Default prose | 18 DIP system text, optional serif; 720 DIP column; 14–24 DIP text and 640/720/840 widths available | [appearance.go](../../internal/app/appearance.go), [appearance_panel.go](../../internal/app/appearance_panel.go) |
| Visual/reading rhythm | Visual line height 1.75; source/reading 1.85; reader H1/H2/H3 at 1.9/1.35/1.12 times prose | [typesetting_view.go](../../internal/mdeditor/typesetting_view.go), [appearance.go](../../internal/app/appearance.go) |
| Shared controls | 40 DIP action/field height; 8 DIP control and 16 DIP panel radius; 18 DIP workbench glyphs | [controls.go](../../internal/writingui/controls.go), [icons.go](../../internal/app/icons.go) |
| Floating surfaces | One theme edge and one shadow; 130/90 ms entry/exit, 4/2 DIP movement | [controls.go](../../internal/writingui/controls.go) |
| Feedback | 100 ms color transition; 90 ms button press to 0.96; 180 ms sidebar motion | [controls.go](../../internal/writingui/controls.go), [app.go](../../internal/app/app.go) |

The main app defaults to the source-preserving visual editor. The View menu provides source and reading choices. Writing Preview retains its sample-specific mode control and does not attach real file/recovery stores. Its current 44 DIP segmented group is one reason the shared toolbar cannot simply be shortened without reviewing both consumers.

## Current palette catalog

These are recorded code values, not a second set of target tokens. [palettes.go](../../internal/app/palettes.go) owns resolved colors, hover/pressed states, semantics and persisted IDs. Keep the role definitions in the root contract.

| ID | Theme | Paper | Rail | Ink | Muted ink | Accent |
| --- | --- | --- | --- | --- | --- | --- |
| paper | Paper · 白纸 | #ffffff | #f7f8fa | #252932 | #5d6571 | #414b5a |
| warm | Warm paper · 暖纸 | #fbf8f2 | #f3eee5 | #38342e | #675e52 | #77603f |
| sakura | Sakura · 樱花粉 | #fff8fa | #f7edf1 | #42343c | #715460 | #965471 |
| sage | Sage · 鼠尾草 | #f7faf6 | #edf2eb | #303c32 | #536553 | #526e50 |
| mist | Mist · 雾蓝 | #f6f9fc | #edf2f7 | #2d3948 | #536274 | #506e91 |
| lavender | Lavender · 薰衣草 | #faf8fd | #f1edf7 | #3c3547 | #665771 | #79628e |
| sand | Sand · 沙丘 | #faf7ef | #f1edde | #3e392b | #625a42 | #796638 |
| pearl | Pearl · 珍珠灰 | #f6f7f7 | #edefef | #323b3c | #556163 | #536e70 |
| graphite | Graphite · 石墨 | #191b20 | #15171b | #e9ecf1 | #a6adba | #bcc6d7 |
| ink | Midnight · 午夜蓝 | #18212c | #141b25 | #e4ecf5 | #a4b5ca | #a8c3e6 |
| pine | Pine · 松林 | #1b2521 | #161f1b | #e4eee6 | #a7baab | #afcbb1 |
| plum | Plum · 暮紫 | #26202b | #201a25 | #eee6f2 | #beabc8 | #cdb2da |

System appearance resolves to Paper or Graphite. Explicit theme IDs persist independently of catalog position. The appearance gallery currently uses four columns in a 408 DIP panel with 60 DIP samples. Its behavior and contrast assertions live in [product_polish_test.go](../../internal/app/product_polish_test.go) and [appearance_test.go](../../internal/app/appearance_test.go).

## Current editing and overlay boundaries

| Area | Present behavior |
| --- | --- |
| Visual editing | Native headings, lists/tasks, simple quotes, fenced code and supported inline spans; canonical Markdown source owns edits, undo, save and recovery |
| Complex content | Images, tables, front matter and unsupported complex Markdown remain literal in visual editing; optional reading can display local images and tables |
| Slash insertion | Nine implemented commands; 280 DIP panel; 40 DIP rows with 2 DIP gaps; Chinese/English filtering, caret-aware flipping, one-step undo and composition yielding |
| Find/replace | 480 DIP shared controls; visual search reserves space above the page and keeps the current match highlighted; regex/case/word/scope options use source offsets |
| Link/code panels | Shared fields and panel treatment; explicit validation, local copy feedback, focus restoration |
| Tab/view retention | Visited rich views preserve private viewport state in-process; exact rich/reader pixel scroll and undo history are not restored across relaunches |
| Input and recovery | Save/recovery capture committed text without cancelling live composition; navigation cancels preedit before focus-loss unmark |
| Window lifecycle | Existing unsaved-close, save-copy, external-conflict and recovery guards remain in the main app; preview samples remain in memory |

See [visual writing](../development/visual-writing.md), [writing guide](../writing.md) and [architecture](../architecture/overview.md) for the full behavior boundaries.

## Applying the new contract

The next UI pass should compare the root contract's targets with this inventory, starting with the compact workbench and type hierarchy. Review the main and preview toolbar groups together; retain the target sizes and keyboard behavior when adjusting the surrounding frame. Then validate floating controls and all palette consumers.

The compact toolbar, revised nominal sidebar/footer, top breathing room and typography roles in the root contract are **not implemented by this documentation change**. Existing code values are starting evidence, not approved exceptions or unchangeable rules. Implementations must follow the [review procedure](review.md) before replacing this inventory with new measurements.

## Existing rendered evidence

The images below were generated from application views in earlier implementation checkpoints. They are headless rendering evidence, not native desktop acceptance, and do not show the new contract's proposed measurements.

- [Integrated visual editor](images/visual-writing/paper-writing.png)
- [Slash commands](images/visual-writing/paper-slash.png)
- [Chinese filtering in Graphite](images/visual-writing/graphite-filtered.png)
- [Search and visible result](images/visual-writing/paper-find-replace.png)
- [Theme gallery](images/appearance.png)
- [Earlier twelve-theme board](images/polish/themes.png)
- [Earlier shared-control animation](images/interaction.gif)
- [Recovery review](images/recovery/paper-recovered.png)

The [visual-writing checkpoint](../development/visual-writing.md) records 392 generated views, selected full-size and board inspection, geometry tests and its native-access limit. [Product polish](../development/product-polish.md) records the preceding shell/theme pass. Neither proves installed-IME, OS clipboard, screen-reader, native window or GPU timing behavior. The [beta delivery gates](../development/beta-delivery.md) still apply.
