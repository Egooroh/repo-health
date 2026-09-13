param([string]$CacheRoot)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'environment.ps1') -CacheRoot $CacheRoot
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    go run ./cmd/repo-health
    if ($LASTEXITCODE -ne 0) { throw 'Repo Health exited with an error.' }
} finally { Pop-Location }
