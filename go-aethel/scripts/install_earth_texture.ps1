# Install high-resolution Earth basemap for AETHEL Global Watch (local-only, no runtime CDN).
# Target: equirectangular 2:1 day map as frontend/assets/earth_day.jpg (and optional *_8k copy).
# Recommended size: 4k–8k, several MB up to ~15–20 MB — fine for a local desktop app.
#
# Usage:
#   .\scripts\install_earth_texture.ps1
#   .\scripts\install_earth_texture.ps1 -Source "D:\maps\blue_marble.jpg"
#   .\scripts\install_earth_texture.ps1 -Download

param(
  [string]$Source = "",
  [switch]$Download
)

$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
$Assets = Join-Path $Root 'frontend\assets'
$dest = Join-Path $Assets 'earth_day.jpg'
$dest8k = Join-Path $Assets 'earth_day_8k.jpg'
New-Item -ItemType Directory -Force -Path $Assets | Out-Null

function Copy-EarthMap([string]$src, [string]$label) {
  if (-not (Test-Path -LiteralPath $src)) { throw "Source missing: $src" }
  $len = (Get-Item -LiteralPath $src).Length
  if ($len -lt 200KB) { throw "Source too small ($len bytes) — need a real equirectangular map" }
  Copy-Item -LiteralPath $src -Destination $dest -Force
  if ($len -ge 3MB) {
    Copy-Item -LiteralPath $src -Destination $dest8k -Force
  }
  $item = Get-Item -LiteralPath $dest
  Write-Output "SOURCE=$src ($label)"
  Write-Output "DEST=$($item.FullName)"
  Write-Output ("SIZE={0} bytes (~{1:N1} MB)" -f $item.Length, ($item.Length / 1MB))
  Write-Output "TIP=Restart AETHEL / hard-reload Global Watch. HUD shows TEX:WxH/jpg when loaded."
}

if ($Source) {
  Copy-EarthMap $Source "operator-path"
  exit 0
}

# Local drop-ins first
$localCandidates = @(
  (Join-Path $Root '1.jpg'),
  (Join-Path $Root 'earth_day.jpg'),
  (Join-Path $Root 'earth_day_8k.jpg'),
  (Join-Path $Assets '1.jpg'),
  (Join-Path $Assets 'earth_day_8k.jpg'),
  (Join-Path $Assets 'earth_day_4k.jpg')
)
foreach ($c in $localCandidates) {
  if (Test-Path -LiteralPath $c) {
    $len = (Get-Item -LiteralPath $c).Length
    if ($len -gt 200KB) {
      Copy-EarthMap $c "local-drop-in"
      exit 0
    }
  }
}

if ($Download) {
  # Solar System Scope 8k day map (NASA-based pack; free for personal use).
  # Alternative: place any public-domain NASA Blue Marble equirectangular as earth_day.jpg yourself.
  $url = 'https://www.solarsystemscope.com/textures/download/8k_earth_daymap.jpg'
  $tmp = Join-Path $env:TEMP ('aethel_earth_' + [guid]::NewGuid().ToString('N') + '.jpg')
  Write-Output "Downloading 8k Earth day map (may take a minute)..."
  Write-Output "URL=$url"
  try {
    Invoke-WebRequest -Uri $url -OutFile $tmp -UseBasicParsing -TimeoutSec 600
    Copy-EarthMap $tmp "download-8k"
  } finally {
    if (Test-Path -LiteralPath $tmp) { Remove-Item -LiteralPath $tmp -Force -ErrorAction SilentlyContinue }
  }
  exit 0
}

Write-Output @"
No local Earth map found.

Options:
  1) Drop a high-res equirectangular (2:1) JPG as:
       $dest
     or root: $(Join-Path $Root '1.jpg')
     Recommended: 4096x2048 or 8192x4096, 3–15 MB.

  2) Download 8k (requires network once):
       .\scripts\install_earth_texture.ps1 -Download

  3) Explicit path:
       .\scripts\install_earth_texture.ps1 -Source 'D:\maps\bluemarble.jpg'

NASA Blue Marble is public domain (US Government work). Solar System Scope textures
are free for personal projects — see frontend/assets/THIRD_PARTY_NOTICES.md.
"@
exit 1
