# Build and package

Install rustup, Git and native build tools. `rust-toolchain.toml` pins Rust 1.99.1 with rustfmt/Clippy; Cargo.lock pins dependencies.

- macOS: Command Line Tools or licensed Xcode; Metal-capable Mac. The bundle declares macOS 13 minimum; verify actual deployment requirements before distribution.
- Windows: Visual Studio C++ Build Tools, Windows SDK, MSVC Rust toolchain and supported GPU.
- Linux: pinned GPUI platform dependencies for Wayland/X11, graphics and fonts. Core CI runs on Linux; its desktop distribution remains experimental until native acceptance.

```sh
cargo run --locked
cargo run --locked -- --preview
make check
make render-check
cargo build --locked --release
```

Desktop is the default feature. `cargo test --locked --no-default-features` needs no GPU. Render-check needs native graphics services even with hidden windows.

## Packaging

macOS: `scripts/bundle-macos.sh` creates `build/MarkGo.app`; add `--preview` for `build/MarkGo GPUI Preview.app`, or `--release` for optimization. Normal ID: `dev.markgo.app`; isolated preview: `dev.markgo.writing-preview`. Only the normal bundle registers Markdown types. The script never launches an app.

Windows: `scripts/bundle-windows.ps1` (optional `-Release`) creates a portable folder/ZIP. Its optional `register-markdown.ps1` adds a current-user Open With registration without changing the default editor.

Linux: `scripts/bundle-linux.sh` (optional `--release`) creates a portable directory with desktop-entry/icon assets. Installing that entry is a separate user choice. Scripts build on their native host; no cross-compilation claim.

These are unsigned development artifacts. Release requires native acceptance, target-specific binary notices, signing/provenance and installation testing. Generated output is ignored.

## State and isolation

Settings: `MarkGo/native/appearance.conf`; recovery: adjacent `session.json`, below macOS Application Support, Windows APPDATA or Linux XDG_CONFIG_HOME (default ~/.config). Unknown/corrupt data is retained. Session schemas 1/2 are readable. If native files do not exist, the previous known MarkGo/session.json and MarkGo/appearance.json are read and retained unchanged. New checkpoints/settings use MarkGo/native. Unknown data blocks migration; development or other application directories are not searched.

`--preview` / `MARKGO_EPHEMERAL=1` bypass real documents, preferences, sessions and recent paths. All integration tests use temporary directories.
