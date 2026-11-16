#!/bin/sh
# Portable development bundle; no install or launch.
set -eu
repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_dir"
profile=debug
case "${1:-}" in
    --release) profile=release; cargo build --locked --release --bin markgo ;;
    "") cargo build --locked --bin markgo ;;
    *) printf 'Use --release or no argument\n' >&2; exit 2 ;;
esac
bundle="$repo_dir/build/MarkGo-linux"
mkdir -p "$bundle/licenses"
cp "target/$profile/markgo" "$bundle/markgo"
cp resources/icon.png "$bundle/markgo.png"
cp LICENSE NOTICE THIRD_PARTY.md THIRD_PARTY_NOTICES.md LICENSES/* "$bundle/licenses/"
cat > "$bundle/dev.markgo.app.desktop" <<'DESKTOP'
[Desktop Entry]
Name=MarkGo
Comment=Write local Markdown documents
Exec=markgo %F
Icon=markgo
Type=Application
Terminal=false
Categories=Office;TextEditor;
MimeType=text/markdown;text/x-markdown;
StartupWMClass=dev.markgo.app
DESKTOP
printf 'Install the binary on PATH and desktop/icon assets using your desktop environment. This script does not install or launch.\n' > "$bundle/INSTALL.txt"
tar -czf "$repo_dir/build/MarkGo-linux.tar.gz" -C "$repo_dir/build" MarkGo-linux
printf 'Built: %s\n' "$bundle"
