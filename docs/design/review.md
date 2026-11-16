# Design implementation review

Use this procedure when applying [DESIGN.md](../../DESIGN.md). The contract defines the target; [design-system.md](design-system.md) records current code. This checklist does not itself establish that either has been accepted on native platforms.

## Review sequence

1. State the user-visible problem and affected shared component. Identify which product contracts must survive the change.
2. Read the current implementation and its tests. Distinguish an intentional user decision from an old incidental measurement.
3. Implement one representative native scene using existing controls and real editor state. Include its keyboard and pointer states, not only the default appearance.
4. Capture and inspect the actual view. Correct clipping, misalignment, hierarchy and feedback before expanding the change to other surfaces.
5. Run the affected behavior and geometry regressions. Check every consumer of a changed shared token; do not create a slightly different copy of the component.
6. Record before/after evidence, scope and limitations in the PR. Update the implementation record only to match code that changed.

Documentation-only work needs link, consistency and attribution checks plus required repository checks; it does not need fabricated visual evidence or tests that merely repeat prose. For a small local UI fix, verify its affected flows and shared consumers. Broaden checks when shared rules change or a failure exposes a wider issue.

## Scene and state matrix

| Area | Required scenarios | Pass condition |
| --- | --- | --- |
| Window/workbench | 960 × 640 and normal desktop size; sidebar open/closed; Focus mode | Window controls reachable; active document identifiable; divider stays inside page edges; writing is not crowded out |
| Tabs | One and many documents; long Chinese/English names; duplicate filenames; dirty active/inactive files; overflow | Names/close targets do not jump; keyboard and pointer target the intended file; history and close guards survive |
| Prose | Chinese punctuation, Latin paragraphs, mixed scripts, emoji, wrapped headings, long lists, quotes and code | Consistent fallback and baseline; useful line length; correct hanging indents; readable selection and stable caret geometry |
| Controls | Default, hover, pressed, focused, disabled, invalid and busy where meaningful | Distinguishable states; at least 40 DIP non-overlapping targets; optical icon alignment; dragging outside cancels |
| Slash/menu | Empty paragraph; Chinese/English query; no result; top/bottom of viewport; keyboard and mouse; composition | Whole rows; no caret obstruction; no accidental input loss or preedit commit; dismissal and undo behave correctly |
| Search | First/last visible match; expanded Replace; regex error; scoped replacement; no matches | Query and result visible together; highlight remains legible; scope cannot silently widen; error text is usable |
| Popovers/dialogs | Open, switch panels, dismiss, invalid submit, long field/error text, reduced motion | No clipped labels or moving targets; focus returns correctly; hidden/closing controls and modal background cannot activate |
| Files and status | Save success/failure; edit after save; recovery notice; external conflict; close last/dirty document | Feedback reports the actual state; safe action is clear; no data loss, misleading saved badge or layout jump |
| Appearance | Paper, Warm paper, Sakura, Graphite; every palette for shared color changes | Same component geometry; readable normal/hover/pressed/selection/focus states; preference choice preserves the document |
| Display/platform | 1×, 1.5× and 2× for geometry; native platforms for affected text/window/input behavior | No fractional-scale clipping; correct native text services and window behavior; platform limits stated separately |

## Existing verification anchors

Use these as starting points; they are evidence for the behavior they actually exercise, not proof that a proposed target has been implemented.

| Concern | Existing tests/source |
| --- | --- |
| Theme contrast, gallery geometry and keyboard return | [product_polish_test.go](../../internal/app/product_polish_test.go), [appearance_test.go](../../internal/app/appearance_test.go) |
| Tab identity, overflow and history | [document_tabs_test.go](../../internal/app/document_tabs_test.go) |
| Slash placement, pointer cancellation and visible search matches | [writing_screenshots_test.go](../../internal/app/writing_screenshots_test.go) |
| Real temporary-file round trips, recovery, viewports and composition | [writing_test.go](../../internal/app/writing_test.go) |
| Fractional-scale renders | [product_scale_test.go](../../internal/app/product_scale_test.go) |
| Native text/layout projection and formatting behavior | [mdeditor tests](../../internal/mdeditor) |

For a shared workbench change, existing renders can be generated with:

```sh
MARKGO_SCREENSHOT_DIR="$PWD/build/design-review/workbench" CGO_ENABLED=0 go test ./internal/app -run '^Test(ProductPolishScreenshots|ProductScaleScreenshots|WorkbenchTabsScreenshots|VisualWritingScreenshots)$' -count=1
MARKGO_SCREENSHOT_DIR="$PWD/build/design-review/typesetting" CGO_ENABLED=0 go test ./internal/app -run '^TestTypesettingScreenshots$' -count=1
MARKGO_SCREENSHOT_DIR="$PWD/build/design-review/recovery" CGO_ENABLED=0 go test ./internal/app -run '^TestRecoveryScreenshots$' -count=1
```

Separate destinations prevent similarly named fixtures from overwriting evidence. Inspect relevant full-size images as well as boards. A generated file count is not a substitute for inspection. Keep all fixture documents and persistent state isolated from the real library.

## Evidence record

Every visible-change PR identifies:

- The exact revision, scene, theme, window dimensions and display scale.
- Before/after images from the actual affected view, with different fixtures or historical baselines clearly identified.
- The interaction exercised and the observed final state, including cancellation and errors where relevant.
- Checks actually run and their results; distinguish a source review, behavior test, render and native session.
- Unverified native behavior or capabilities still outside the implementation.

Native checks cover the affected operating-system behavior, including installed IME candidates, clipboard, focus restoration, accessibility, window drag/resize/Snap and frame pacing when relevant. Use only the desktop/browser path authorized for that session and wait for its required approvals. If the allowed application is not running or host permission is unavailable, report the concrete reason and leave native acceptance open. A successful build or headless screenshot cannot close that gap.

Do not mark the application as satisfying this design contract or ready for beta solely because the documentation, tokens or test suite exist. Review the user's actual writing workflow.
