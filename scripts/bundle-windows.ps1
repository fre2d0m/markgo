param([switch]$Release)
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
Push-Location $repo
try {
    $profile = 'debug'
    if ($Release) { $profile = 'release'; cargo build --locked --release --bin markgo }
    else { cargo build --locked --bin markgo }
    if ($LASTEXITCODE -ne 0) { throw 'Cargo build failed' }
    $bundle = Join-Path $repo 'build/MarkGo-windows'
    New-Item -ItemType Directory -Force $bundle, (Join-Path $bundle 'Licenses') | Out-Null
    Copy-Item "target/$profile/markgo.exe" $bundle -Force
    Copy-Item 'resources/icon.png' $bundle -Force
    Copy-Item 'LICENSE','NOTICE','THIRD_PARTY.md','THIRD_PARTY_NOTICES.md','LICENSES/*' (Join-Path $bundle 'Licenses') -Force
    Copy-Item 'scripts/register-markdown.ps1' $bundle -Force
    Compress-Archive -Path $bundle -DestinationPath 'build/MarkGo-windows.zip' -Force
    Write-Host "Built: $bundle"
} finally { Pop-Location }
