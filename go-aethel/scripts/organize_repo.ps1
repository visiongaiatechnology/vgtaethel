# VGT AETHEL — repository cleanup / folder layout
# Run from go-aethel root:
#   powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\organize_repo.ps1
$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
if (-not (Test-Path (Join-Path $Root 'go.mod'))) {
  $Root = Get-Location
}
Set-Location $Root
Write-Host "Organizing: $Root"

function Ensure-Dir($p) {
  New-Item -ItemType Directory -Force -Path $p | Out-Null
}

function Move-IfExists($src, $dstDir) {
  if (-not (Test-Path -LiteralPath $src)) { return }
  Ensure-Dir $dstDir
  $name = Split-Path $src -Leaf
  $dest = Join-Path $dstDir $name
  if ((Resolve-Path $src).Path -eq (Join-Path (Resolve-Path $dstDir) $name)) { return }
  Move-Item -LiteralPath $src -Destination $dest -Force
  Write-Host "  moved $src -> $dest"
}

# --- folders ---
Ensure-Dir (Join-Path $Root 'docs\project')
Ensure-Dir (Join-Path $Root 'docs\scratch')
Ensure-Dir (Join-Path $Root 'scripts')
Ensure-Dir (Join-Path $Root '_archive')
Ensure-Dir (Join-Path $Root '_logs')
Ensure-Dir (Join-Path $Root 'frontend\assets')

# --- Earth basemap ---
$earthDest = Join-Path $Root 'frontend\assets\earth_day.jpg'
$needEarth = $true
if (Test-Path -LiteralPath $earthDest) {
  if ((Get-Item -LiteralPath $earthDest).Length -gt 10KB) { $needEarth = $false }
}
if ($needEarth) {
  foreach ($cand in @('1.jpg', 'frontend\assets\1.jpg')) {
    $p = Join-Path $Root $cand
    if (-not (Test-Path -LiteralPath $p)) { continue }
    if ((Get-Item -LiteralPath $p).Length -le 10KB) { continue }
    Copy-Item -LiteralPath $p -Destination $earthDest -Force
    Write-Host "  earth texture: $cand -> frontend\assets\earth_day.jpg ($((Get-Item -LiteralPath $earthDest).Length) bytes)"
    break
  }
}

# --- docs ---
Move-IfExists (Join-Path $Root 'BUILD_SHERPA.md') (Join-Path $Root 'docs\project')
Move-IfExists (Join-Path $Root 'COMPATIBILITY.md') (Join-Path $Root 'docs\project')
Move-IfExists (Join-Path $Root 'RELEASE_NOTES.md') (Join-Path $Root 'docs\project')
Move-IfExists (Join-Path $Root 'task.md') (Join-Path $Root 'docs\project')

# --- scratch / archive ---
if (Test-Path (Join-Path $Root 'SCRATCH')) {
  Move-IfExists (Join-Path $Root 'SCRATCH') (Join-Path $Root 'docs')
  if (Test-Path (Join-Path $Root 'docs\SCRATCH')) {
    Rename-Item (Join-Path $Root 'docs\SCRATCH') 'scratch' -Force -ErrorAction SilentlyContinue
  }
}
Move-IfExists (Join-Path $Root 'pre-A-B-backup-2026-07-11') (Join-Path $Root '_archive')

# --- logs ---
foreach ($f in @('aethel_debug.log','stderr.log','stdout.log','verification_evidence.log')) {
  Move-IfExists (Join-Path $Root $f) (Join-Path $Root '_logs')
}

# --- root clutter binaries (keep one copy under build\bin) ---
Ensure-Dir (Join-Path $Root 'build\bin')
foreach ($f in @('AETHEL.exe','go-aethel.exe')) {
  $p = Join-Path $Root $f
  if (Test-Path $p) {
    $dest = Join-Path $Root "build\bin\$f"
    if (-not (Test-Path $dest)) {
      Copy-Item $p $dest -Force
      Write-Host "  staged $f -> build\bin\"
    }
    # keep root AETHEL.exe for convenience run; remove go-aethel.exe dupe if both exist
    if ($f -eq 'go-aethel.exe') {
      Remove-Item $p -Force -ErrorAction SilentlyContinue
      Write-Host "  removed root go-aethel.exe (use build\bin or AETHEL.exe)"
    }
  }
}

# --- DLLs at root: leave (runtime next to exe) but also ensure build\bin has them ---
foreach ($dll in @('onnxruntime.dll','sherpa-onnx-c-api.dll','sherpa-onnx-cxx-api.dll')) {
  $p = Join-Path $Root $dll
  if (Test-Path $p) {
    $dest = Join-Path $Root "build\bin\$dll"
    if (-not (Test-Path $dest)) { Copy-Item $p $dest -Force }
  }
}

# --- scripts ---
Move-IfExists (Join-Path $Root 'build_aethel.bat') (Join-Path $Root 'scripts')

# --- frontend assets junk from failed copy probes ---
$assets = Join-Path $Root 'frontend\assets'
Get-ChildItem $assets -Force -ErrorAction SilentlyContinue | Where-Object {
  $_.Name -like '_bin*' -or $_.Name -like '_copy*' -or $_.Name -like '_do_*' -or
  $_.Name -like '_null*' -or $_.Name -like '_trigger*' -or $_.Name -like '_write*'
} | ForEach-Object {
  Remove-Item $_.FullName -Force -ErrorAction SilentlyContinue
  Write-Host "  cleaned $($_.Name)"
}

# --- root 1.jpg after install to assets ---
if (Test-Path -LiteralPath $earthDest) {
  if ((Get-Item -LiteralPath $earthDest).Length -gt 10KB) {
    $rootJpg = Join-Path $Root '1.jpg'
    if (Test-Path -LiteralPath $rootJpg) {
      Remove-Item -LiteralPath $rootJpg -Force -ErrorAction SilentlyContinue
      Write-Host "  removed root 1.jpg (now in frontend\assets\earth_day.jpg)"
    }
  }
}

# --- brand image ---
$rootWebp = Join-Path $Root 'aethel.webp'
$feWebp = Join-Path $Root 'frontend\aethel.webp'
if ((Test-Path -LiteralPath $rootWebp) -and -not (Test-Path -LiteralPath $feWebp)) {
  Move-Item -LiteralPath $rootWebp -Destination $feWebp -Force
}

Write-Host "Done. See docs\STRUCTURE.md"
