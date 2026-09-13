param([string]$CacheRoot)
$ErrorActionPreference = 'Stop'
if (-not $CacheRoot) { $CacheRoot = Join-Path (Split-Path $PSScriptRoot -Parent) '.cache' }
$CacheRoot = [System.IO.Path]::GetFullPath($CacheRoot)
foreach ($part in @('go-build', 'go-mod', 'tmp')) {
    New-Item -ItemType Directory -Path (Join-Path $CacheRoot $part) -Force | Out-Null
}
$env:GOCACHE = Join-Path $CacheRoot 'go-build'
$env:GOMODCACHE = Join-Path $CacheRoot 'go-mod'
$env:GOTMPDIR = Join-Path $CacheRoot 'tmp'

