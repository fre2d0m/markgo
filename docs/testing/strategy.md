# Verification strategy

`make check` runs rustfmt, core tests and Clippy for core/desktop/render targets. Tests exercise source fidelity, UTF-16/Unicode, composition, paired delimiters/fences, list/task editing, regex/scope replacement, history, guarded writes, corrupt/conflicting/out-of-order recovery and preferences. Format-dense long-document edits are compared with fresh projections; this is not a native latency benchmark.

`make render-check` uses GPUI's hidden application windows, actual Metal/WGPU rendering, native text shaping and test event dispatch. Images go to `build/gpui-evidence/`. Inspect normal, empty, narrow, relevant themes, slash, appearance, search/replace, close/error and rich editing/reading states. Check target geometry, candidate bounds, contrast, clipping and focus. Fix and recapture. Fixtures never use the real writing library.

Windows/macOS CI builds and lints the application and runs `render_check --events-only`: isolated input, pointer tabs, toolbar/dialogs, heading conversion, focus, composition and theme checks. This mode uses native text shaping with test windows, without GPU capture. Windows' pinned GPUI backend does not expose an offscreen renderer. Linux CI covers core.

Workbench regression scenes check aligned command/document rows in Paper, Sakura and Graphite, long paths, selected tabs and window clamping. Controlled-clock tests capture popup entry/departure, reverse a closing popup, continue typing during departure and honor reduced motion. Button tests retain their hit bounds through press feedback and cancel formatting when the pointer is released outside.

The macOS job repeats those events with a 1 MiB main-thread stack (`ulimit -s 1024`). This reproduces the Windows debug dialog stack-overflow budget instead of letting macOS' larger default stack hide it. Windows packaging also reads the linked PE header and checks the 8 MiB reserve. Both platforms rasterize all 38 embedded control icons and reject blank results; GPU mode also checks their actual painted pixels, including window controls.

Native acceptance separately requires installed Chinese IME, emoji, clipboard, accessibility, titlebar hit-testing/window drag/resize/Snap, file-drop/association, save dialogs, sleep/shutdown/restart, fractional scale and native latency on every advertised platform. Test pointer dispatch does not exercise Windows' `WM_NCHITTEST` routing.

Use only session-authorized desktop tooling with per-action approval. Record missing apps/permissions. A successful build, headless image or source inspection cannot substitute for host acceptance.
