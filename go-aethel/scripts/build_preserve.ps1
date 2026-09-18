# STATUS: DIAMANT VGT SUPREME
[CmdletBinding()]
param(
    [ValidatePattern('^[A-Za-z0-9._-]+\.exe$')]
    [string]$OutputName = 'AETHEL.exe'
)

$ErrorActionPreference = 'Stop'
$repositoryRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$buildRoot = [System.IO.Path]::GetFullPath((Join-Path $repositoryRoot 'build'))
$binaryRoot = [System.IO.Path]::GetFullPath((Join-Path $buildRoot 'bin'))
$archiveRoot = [System.IO.Path]::GetFullPath((Join-Path $buildRoot 'archive'))

if (-not $binaryRoot.StartsWith($buildRoot + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw 'Build binary path escaped the repository build jail.'
}

New-Item -ItemType Directory -Path $binaryRoot -Force | Out-Null
New-Item -ItemType Directory -Path $archiveRoot -Force | Out-Null

Get-ChildItem -LiteralPath $binaryRoot -File -Filter '*.exe' | ForEach-Object {
    $archiveTarget = Join-Path $archiveRoot $_.Name
    if (-not (Test-Path -LiteralPath $archiveTarget -PathType Leaf)) {
        Copy-Item -LiteralPath $_.FullName -Destination $archiveTarget
    }
}

Push-Location $repositoryRoot
try {
    & wails build -clean=false -o $OutputName
    if ($LASTEXITCODE -ne 0) {
        throw "Wails build failed with exit code $LASTEXITCODE."
    }
} finally {
    Pop-Location
}

Get-ChildItem -LiteralPath $archiveRoot -File -Filter '*.exe' | ForEach-Object {
    $binaryTarget = Join-Path $binaryRoot $_.Name
    if (-not (Test-Path -LiteralPath $binaryTarget -PathType Leaf)) {
        Copy-Item -LiteralPath $_.FullName -Destination $binaryTarget
    }
}

$newBuild = Join-Path $binaryRoot $OutputName
if (-not (Test-Path -LiteralPath $newBuild -PathType Leaf)) {
    throw 'Wails reported success but the expected executable is missing.'
}
Copy-Item -LiteralPath $newBuild -Destination (Join-Path $archiveRoot $OutputName) -Force
Write-Host "Preserved build: $newBuild"
