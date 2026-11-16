#!/bin/sh
# Build a separate, in-memory preview. This script never launches an app.
set -eu
native_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
repo_dir=$(dirname -- "$native_dir")
bundle="$repo_dir/build/MarkGo GPUI Preview.app"
cd "$native_dir"
cargo build --locked --features desktop --bin markgo
mkdir -p "$bundle/Contents/MacOS" "$bundle/Contents/Resources"
mkdir -p "$bundle/Contents/Resources/Licenses"
cp target/debug/markgo "$bundle/Contents/MacOS/markgo-bin"
cp "$repo_dir/resources/icon.png" "$bundle/Contents/Resources/icon.png"
cp "$repo_dir/LICENSE" "$bundle/Contents/Resources/Licenses/MarkGo.txt"
cp "$native_dir/THIRD_PARTY.md" "$bundle/Contents/Resources/Licenses/THIRD_PARTY.md"
cp "$native_dir/LICENSES/LUCIDE-1.43.0.txt" "$bundle/Contents/Resources/Licenses/LUCIDE-1.43.0.txt"
cat > "$bundle/Contents/MacOS/markgo-preview" <<'LAUNCHER'
#!/bin/sh
bundle_bin=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
exec "$bundle_bin/markgo-bin" --preview
LAUNCHER
chmod +x "$bundle/Contents/MacOS/markgo-preview"
cat > "$bundle/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>dev.markgo.writing-preview</string>
<key>CFBundleName</key><string>MarkGo GPUI Preview</string>
<key>CFBundleDisplayName</key><string>MarkGo GPUI Preview</string>
<key>CFBundleExecutable</key><string>markgo-preview</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleShortVersionString</key><string>0.1.0</string>
<key>CFBundleVersion</key><string>1</string>
<key>CFBundleIconFile</key><string>icon.png</string>
<key>NSHighResolutionCapable</key><true/>
<key>LSMinimumSystemVersion</key><string>13.0</string>
</dict></plist>
PLIST
printf '%s\n' "Built: $bundle"
