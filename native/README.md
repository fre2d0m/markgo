# Native migration foundation

This is the isolated Rust editor foundation for the [GPUI migration](../docs/architecture/gpui-migration.md). It is a library, not a replacement client or a GUI demo. The ordinary MarkGo executable continues to use MyGo.

## Check

Install Rust with rustup. Commands run from this directory select the pinned toolchain automatically:

```sh
cd native
cargo fmt --check
cargo clippy --locked --all-targets -- -D warnings
cargo test --locked
```

From the repository root, `make native-check` runs the same checks. `Cargo.lock` is committed. Generated output stays in `native/target/`; the library has no filesystem, network or desktop side effects.

## Implemented boundaries

- `document`: original Markdown bytes, directed selections, source replacement transactions, undo/redo and versioned immutable snapshots. History retains changed text rather than a whole document per edit.
- `Composition`: separate candidate text and selection. Updates leave committed snapshots untouched. Commit becomes one undo step; cancellation retains the prior source selection and redo history.
- `projection`: parser-confirmed top-level ATX headings and paragraphs, with compact per-line source/display maps. Other Markdown remains literal. UTF-8 validation and revision/source checks prevent a stale projection from editing a different document state.
- `source_contract`: cases adapted from the existing Go editing contracts, including CJK/emoji, reversed cross-block selection, paired heading delimiters, metadata/code/reference preservation, mixed line endings and independent document history.

The source and projection offsets are UTF-8 bytes; platform text-service offsets are UTF-16 code units. `utf16_to_byte` exposes explicit rounding for a surrogate pair, and `byte_to_utf16` rejects an invalid byte boundary. Grapheme navigation belongs in the native input adapter and is not implemented here.

## Next integration boundary

Introduce GPUI Kit with the first real native editor integration, using the dependency baseline in the migration decision. The adapter must own native composition event ordering and geometry while this library owns committed source and history. Do not bind arbitrary widget change events directly to disk or treat a code-editor widget's private undo stack as a second authority.

The current projection reparses and retains a source snapshot. It is a correctness foundation, not the final large-document layout or performance implementation. Structured lists, code/inline editing, clipboard, rendering and real IME validation remain outstanding. No library test establishes native input, visual quality or release readiness.
