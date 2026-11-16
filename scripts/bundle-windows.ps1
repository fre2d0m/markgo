param([switch]$Release)
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
Push-Location $repo
try {
    $profile = 'debug'
    if ($Release) { $profile = 'release'; cargo build --locked --release --bin markgo }
    else { cargo build --locked --bin markgo }
    if ($LASTEXITCODE -ne 0) { throw 'Cargo build failed' }
    # Verify the linked image, not just a build-script string. Native view
    # construction can overflow the default 1 MiB Windows executable stack.
    $image = [IO.File]::ReadAllBytes((Join-Path $repo "target/$profile/markgo.exe"))
    $optional = [BitConverter]::ToInt32($image, 0x3c) + 24
    $magic = [BitConverter]::ToUInt16($image, $optional)
    $stack = if ($magic -eq 0x20b) {
        [BitConverter]::ToUInt64($image, $optional + 72)
    } elseif ($magic -eq 0x10b) {
        [uint64][BitConverter]::ToUInt32($image, $optional + 72)
    } else { throw 'Unrecognized PE optional header' }
    if ($stack -lt 8388608) { throw "Insufficient executable stack reserve: $stack" }
    Write-Host "Verified PE stack reserve: $stack bytes"
    $bundle = Join-Path $repo 'build/MarkGo-windows'
    New-Item -ItemType Directory -Force $bundle, (Join-Path $bundle 'Licenses') | Out-Null
    Copy-Item "target/$profile/markgo.exe" $bundle -Force
    Copy-Item 'resources/icon.png' $bundle -Force
    Copy-Item 'LICENSE','NOTICE','THIRD_PARTY.md','THIRD_PARTY_NOTICES.md','LICENSES/*' (Join-Path $bundle 'Licenses') -Force
    Copy-Item 'scripts/register-markdown.ps1' $bundle -Force
    Compress-Archive -Path $bundle -DestinationPath 'build/MarkGo-windows.zip' -Force
    Write-Host "Built: $bundle"
} finally { Pop-Location }
