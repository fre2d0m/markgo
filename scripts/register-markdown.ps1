# Optional user-invoked Open With registration. Does not change defaults.
$ErrorActionPreference = 'Stop'
$exe = Join-Path $PSScriptRoot 'markgo.exe'
if (-not (Test-Path $exe -PathType Leaf)) { throw 'Keep this script beside markgo.exe' }
$root = 'HKCU:\Software\Classes'
$command = Join-Path $root 'Applications\markgo.exe\shell\open\command'
New-Item -Path $command -Force | Out-Null
Set-Item -Path $command -Value ('"' + $exe + '" "%1"')
foreach ($extension in '.md', '.markdown') {
    $key = Join-Path $root ($extension + '\OpenWithList\markgo.exe')
    New-Item -Path $key -Force | Out-Null
}
Write-Host 'MarkGo is available in Open With. Your default editor is unchanged.'
