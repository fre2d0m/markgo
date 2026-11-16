# Verification strategy

`make check` runs rustfmt, core tests and Clippy for core/desktop/render targets. Tests exercise source fidelity, UTF-16/Unicode, composition, paired delimiters/fences, list/task editing, regex/scope replacement, history, guarded writes, corrupt/conflicting/out-of-order recovery and preferences. Format-dense long-document edits are compared with fresh projections; this is not a native latency benchmark.

`make render-check` uses GPUI's hidden application windows, actual Metal/WGPU rendering, native text shaping and test event dispatch. Images go to `build/gpui-evidence/`. Inspect normal, empty, narrow, relevant themes, slash, appearance, search/replace, close/error and rich editing/reading states. Check target geometry, candidate bounds, contrast, clipping and focus. Fix and recapture. Fixtures never use the real writing library.

Windows/macOS CI builds and lints the application; Linux CI covers core. Native acceptance separately requires installed Chinese IME, emoji, clipboard, accessibility, window resize/Snap, file-drop/association, save dialogs, sleep/shutdown/restart, fractional scale and native latency on every advertised platform.

Use only session-authorized desktop tooling with per-action approval. Record missing apps/permissions. A successful build, headless image or source inspection cannot substitute for host acceptance.
