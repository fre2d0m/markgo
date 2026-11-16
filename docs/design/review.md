# Design implementation review

[DESIGN.md](../../DESIGN.md) is the authority; [implementation values](design-system.md) describe current code.

1. Identify the visible problem and user/data contracts that must survive.
2. Implement a representative real scene with actual editor state.
3. Capture, inspect, fix and recapture before extending shared tokens.
4. Test affected behavior and shared consumers.
5. Record revision, scene, theme, size/scale, results and native limitations in the PR.

## Required scenes

| Area | Review |
| --- | --- |
| Workbench/tabs | Normal and 960 × 640; long/duplicate names; dirty inactive tabs; overflow; Focus; reachable window controls |
| Prose | Chinese/Latin/emoji; wrapping; headings; list indents; code; selection and candidate geometry |
| Controls | Normal/hover/press/focus/disabled/invalid/busy; 40 DIP non-overlapping targets; pointer cancellation |
| Slash/search | Empty/no result; composition; whole menu rows; top/bottom caret; regex errors; scoped replacement; visible match |
| Dialogs | Open/switch/cancel/error; long text; modal focus; no activation through overlays |
| Files | Save/version/conflict; reload; recovery; failed checkpoint; close/exit choices |
| Appearance | Paper/Warm/Sakura/Graphite and all palettes for shared colors; unchanged source/history |
| Platform | Fractional scale, native IME/clipboard/accessibility, window behavior, reduced motion and latency |

## Evidence

Run `make check` and `make render-check`. Tests cover source/history, semantic editing, composition, guarded files and recovery. `examples/render_check.rs` exercises actual hidden GPU windows and event dispatch. Inspect `build/gpui-evidence/` at full size; a file count is not inspection.

Keep all fixtures out of the real writing library. Source inspection, behavior tests, renders and native acceptance are separate. Use only authorized desktop/browser tools and approvals; record missing apps or permissions. Builds/screenshots alone do not certify beta quality.
