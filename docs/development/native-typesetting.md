# Native typesetting checkpoint

This is an isolated, runnable editing experiment, not a 1.0 beta release or the default document editor. It opens built-in samples only. Changes stay in memory and closing the preview discards them. It has no file associations, open/save callbacks, recovery store, appearance store or single-instance handoff.

## Try it

```sh
go run ./cmd/markgo-preview
go tool mygo build -skip-dmg ./cmd/markgo-preview
go tool mygo build -platform windows/amd64 -skip-installer ./cmd/markgo-preview
```

The standalone application is **MarkGo Writing Preview**, under `build/typesetting-preview/`. The ordinary MarkGo executable also accepts `--typesetting-preview` as its first argument, before any real workspace initialization. Normal launches retain the existing editor.

Edit the English or Chinese sample directly. Drag from a heading into a paragraph, replace the selection, undo, switch to Source, type, switch back, and undo again. The Blank page and Long read samples exercise the same editor. `Cmd/Ctrl+Shift+M` switches views; `Cmd/Ctrl+0/1/2/3` applies a paragraph or heading. Appearance and sample-sidebar controls retain the writing selection. Switching views or pages cancels unfinished composition and restores the original selection with an explanatory notice.

## Changes in this checkpoint

| Before | After |
| --- | --- |
| Editable source uses one font size; larger headings are available in the separate reader | The preview directly edits headings and paragraphs with native variable-size text layout, a 720 DIP page, 18 DIP body text and 1.75 line height |
| Reading and source have different roles | Typeset and Source edit the same canonical `editor.Buffer`; view switches transfer the primary selection and retain history |
| No runnable source-preserving rich-text adapter | A rune-to-source-byte map hides top-level ATX prefixes and applies precise edits without Markdown serialization |
| A straightforward rich-text round trip rewrites unsupported syntax | Untouched code languages, tables, reference definitions, front matter and line-ending metadata stay intact; unsupported syntax remains visible |
| Rich text and source would normally have separate undo stacks | Native menu commands, typing bursts, paste and composition commits route to source history; one input-method commit is one step |
| Keyboard-event simulation alone misses direct desktop text-service calls | An application-owned `TextInputClient` handles direct UTF-16 replacements, preedit, cancellation, and unmark-before-commit ordering; richtext supplies shaped caret geometry |
| The ordinary application is the only executable | A separate preview entry and bundle identifier allow sample-only inspection without touching the real workspace |
| Existing workbench appearance and motion | The preview reuses Paper, Warm paper, Graphite, custom titlebar, reversible sidebar motion, quiet hover feedback and small popover transitions; no page entrance motion |
| First prototype screenshots had tight body lines and an initial zero character count | Corrected line height and initialized statistics, then captured the rendered states again |

## Engineering decision

Continue the source-preserving native richtext adapter experiment. Keep it out of the real document lifecycle until the remaining gates pass.

| Approach | Evidence and tradeoff |
| --- | --- |
| Extend `ui/editor` block layout | The pinned editor has one font and line-height model. Variable-height blocks would require changes throughout wrapping, hit testing, scrolling and input geometry in the dependency. Existing source behavior remains useful as the canonical buffer and fallback view. |
| Serialize `richtext.ParseMarkdown` | Rejected: the earlier probe lost code language, rewrote references, and converted tables/front matter. No-op loading must not rewrite a file. |
| Project source into native richtext | Selected for this checkpoint. Reuses shaping, continuous pointer selection, wrapping and caret layout while keeping source authoritative. Requires explicit source maps and input/history adapters; these are exercised here, not assumed. |

`internal/mdeditor/typesetting*.go` implements the adapter. `Document` is disposable display state, including staged preedit. The source buffer changes only when an edit is committed. `FromSource` rebuilds display without editing source or clearing source history. Header commands, native Edit-menu commands and clipboard paste use that same history. Paste reads plain text rather than importing HTML/RTF attributes. Unsupported rich-format commands show feedback rather than creating formatting that cannot be saved in Markdown.

Native input methods can call `TextInputClient` without `ObserveInput`. The adapter therefore owns that client rather than relying on keyboard observations to identify composition. Preedit replaces display text only. Cancellation restores the selection and source; confirmation commits once. An immediate final replacement after Unmark replaces that exact commit only if the source version and native range still match. Typing bursts similarly merge only the adapter's own latest version at the same caret; navigation, commands and IME break the burst.

## Evidence

Behavior tests cover forward and reversed cross-block selections, pointer drag, deletion/joining, Enter at heading start/middle/end, select-all deletion, emoji and CJK offsets, plain-text paste from a mixed HTML/text clipboard, native menu undo/redo, shared source-mode history, direct native-client calls, composition cancellation and unmark/commit ordering. Unchanged replacement tests guard against accidentally undoing earlier writing. The long-document test uses more than 100,000 Unicode characters and verifies that typing/undo retains source and keeps the caret in the viewport.

Rendered evidence comes from the actual Go view through `ui.NewTester`, not a browser or design mockup. The screenshot matrix has **39 states**: 13 scenes × three palettes, including writing, Chinese, selection, source, appearance, paragraph menu, empty, notice, sidebar hidden, simulated composition, literal syntax, long document, and 24 DIP book typography. Writing/Chinese use 1320×840; the other scenes use 960×640. Screenshots were inspected, corrected and recaptured.

![Editable Chinese page, headless rendering](../design/images/typesetting/paper-chinese.png)

![Continuous selection, headless rendering](../design/images/typesetting/graphite-selection.png)

![Appearance at minimum window size, headless rendering](../design/images/typesetting/warm-appearance.png)

On an Apple M2 Max, Go 1.27.4, CGO disabled, one local benchmark run measured approximately **1.84 ms** to rebuild a 107,200-byte projection and **7.10 ms per headless input operation** on the 100,000+ character fixture. The input benchmark allocated about **16.1 MB per operation**. These are CPU/headless averages, not desktop input latency, p95, GPU frame time or a performance acceptance result. The allocation cost is a concrete reason to make projection updates incremental before production integration.

Local `go test ./...`, module consistency, Go vet, MyGo vet for both entries, staticcheck and macOS/Windows preview packaging are checked for this checkpoint. The local `make` shim is blocked by an unaccepted Xcode license, so its component checks are run individually; CI runs the actual Make target and race detector.

**Native acceptance is still pending.** The computer-use tool did not grant access to the standalone preview application. No real desktop screenshot, OS candidate-window check, Windows drag/Snap/DPI check or OS clipboard acceptance is claimed. Calling the native client directly in tests verifies the adapter contract, not the behavior of an installed input method.

## Limits and next gates

- Only top-level ATX heading prefixes receive structured presentation. Optional closing heading markers remain literal. Setext headings, inline syntax, lists, quotes, code, front matter, images and tables remain editable Markdown text.
- Physical source lines remain separate display paragraphs; soft-line-break paragraph fusion and blank-line spacing need a deliberate model before expanding block support.
- This prototype uses a complete projection rebuild after edits. Incremental projection, precise paragraph anchoring across mode switches, native text-service hit testing, multi-line preedit and long-document allocation/latency profiling remain integration gates.
- Current history is in memory; this preview does not save or recover edits. The production editor's recovery implementation is separate and unchanged.
- Next: native Windows/macOS IME and selection acceptance, then source-preserving list/quote/code editing. Images, tables and export follow the [beta delivery plan](beta-delivery.md). Do not promote this adapter to ordinary files just because the sample looks finished.
