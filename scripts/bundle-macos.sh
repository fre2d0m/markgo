#!/bin/sh
# Build only. Never launch or register the application on the host.
set -eu
repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
preview=0
profile=debug
for option in "$@"; do
    case "$option" in
        --preview) preview=1 ;;
        --release) profile=release ;;
        *) printf 'Unknown option: %s\n' "$option" >&2; exit 2 ;;
    esac
done
cd "$repo_dir"
if [ "$profile" = release ]; then cargo build --locked --release --bin markgo; else cargo build --locked --bin markgo; fi
name=MarkGo
identifier=dev.markgo.app
launcher=markgo
if [ "$preview" = 1 ]; then name="MarkGo GPUI Preview"; identifier=dev.markgo.writing-preview; fi
bundle="$repo_dir/build/$name.app"
mkdir -p "$bundle/Contents/MacOS" "$bundle/Contents/Resources/Licenses"
cp "target/$profile/markgo" "$bundle/Contents/MacOS/markgo-bin"
cp resources/icon.png "$bundle/Contents/Resources/icon.png"
cp LICENSE THIRD_PARTY.md THIRD_PARTY_NOTICES.md NOTICE "$bundle/Contents/Resources/Licenses/"
cp LICENSES/* "$bundle/Contents/Resources/Licenses/"
if [ "$preview" = 1 ]; then
    cat > "$bundle/Contents/MacOS/$launcher" <<'LAUNCHER'
#!/bin/sh
bundle_bin=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
exec "$bundle_bin/markgo-bin" --preview
LAUNCHER
else
    cat > "$bundle/Contents/MacOS/$launcher" <<'LAUNCHER'
#!/bin/sh
bundle_bin=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
exec "$bundle_bin/markgo-bin" "$@"
LAUNCHER
fi
chmod +x "$bundle/Contents/MacOS/$launcher"
icon=icon.png
if command -v iconutil >/dev/null 2>&1 && command -v sips >/dev/null 2>&1; then
    iconset="$repo_dir/build/MarkGo.iconset"
    mkdir -p "$iconset"
    for size in 16 32 128 256 512; do
        sips -z "$size" "$size" resources/icon.png --out "$iconset/icon_${size}x${size}.png" >/dev/null
        doubled=$((size * 2))
        sips -z "$doubled" "$doubled" resources/icon.png --out "$iconset/icon_${size}x${size}@2x.png" >/dev/null
    done
    iconutil -c icns "$iconset" -o "$bundle/Contents/Resources/MarkGo.icns"
    icon=MarkGo.icns
fi
cat > "$bundle/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>$identifier</string>
<key>CFBundleName</key><string>$name</string>
<key>CFBundleDisplayName</key><string>$name</string>
<key>CFBundleExecutable</key><string>$launcher</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleShortVersionString</key><string>0.1.0</string>
<key>CFBundleVersion</key><string>1</string>
<key>CFBundleIconFile</key><string>$icon</string>
<key>NSHighResolutionCapable</key><true/>
<key>LSMinimumSystemVersion</key><string>13.0</string>
PLIST
if [ "$preview" = 0 ]; then
    cat >> "$bundle/Contents/Info.plist" <<'PLIST'
<key>CFBundleDocumentTypes</key><array><dict>
<key>CFBundleTypeName</key><string>Markdown Document</string>
<key>CFBundleTypeRole</key><string>Editor</string>
<key>LSHandlerRank</key><string>Alternate</string>
<key>LSItemContentTypes</key><array><string>net.daringfireball.markdown</string></array>
<key>CFBundleTypeExtensions</key><array><string>md</string><string>markdown</string></array>
</dict></array>
<key>UTImportedTypeDeclarations</key><array><dict>
<key>UTTypeIdentifier</key><string>net.daringfireball.markdown</string>
<key>UTTypeDescription</key><string>Markdown Document</string>
<key>UTTypeConformsTo</key><array><string>public.plain-text</string></array>
<key>UTTypeTagSpecification</key><dict>
<key>public.filename-extension</key><array><string>md</string><string>markdown</string></array>
<key>public.mime-type</key><string>text/markdown</string>
</dict></dict></array>
PLIST
fi
printf '</dict></plist>\n' >> "$bundle/Contents/Info.plist"
plutil -lint "$bundle/Contents/Info.plist"
if [ "$preview" = 0 ]; then ditto -c -k --sequesterRsrc --keepParent "$bundle" "$repo_dir/build/MarkGo-macos.zip"; fi
printf 'Built: %s\n' "$bundle"
