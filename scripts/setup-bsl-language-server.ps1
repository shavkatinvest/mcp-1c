<#
.SYNOPSIS
    Downloads and extracts BSL Language Server (https://github.com/1c-syntax/bsl-language-server)
    into tools-external/bsl-language-server for local BSL linting (see scripts/lint-bsl.cmd).

.DESCRIPTION
    The tool is not vendored into the repo (large, self-contained JRE bundle) — this script
    fetches the latest Windows release on demand. Idempotent: skips download if already present.

.PARAMETER Version
    Release tag to install, e.g. "v1.0.2". Defaults to the latest release.
#>
param(
    [string]$Version
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$destDir = Join-Path $root "tools-external\bsl-language-server"
$exe = Join-Path $destDir "bsl-language-server\bsl-language-server.exe"

if (Test-Path $exe) {
    Write-Output "BSL Language Server already installed at $exe"
    exit 0
}

if (-not $Version) {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/1c-syntax/bsl-language-server/releases/latest"
    $Version = $release.tag_name
}

$zipUrl = "https://github.com/1c-syntax/bsl-language-server/releases/download/$Version/bsl-language-server_win.zip"
$zipPath = Join-Path $root "tools-external\bsl-language-server_win.zip"

New-Item -ItemType Directory -Force -Path (Join-Path $root "tools-external") | Out-Null
Write-Output "Downloading BSL Language Server $Version ..."
Invoke-WebRequest -Uri $zipUrl -OutFile $zipPath

Write-Output "Extracting..."
Expand-Archive -Path $zipPath -DestinationPath $destDir -Force
Remove-Item $zipPath

if (-not (Test-Path $exe)) {
    Write-Error "Extraction did not produce expected binary at $exe"
    exit 1
}

Write-Output "Installed: $exe"
