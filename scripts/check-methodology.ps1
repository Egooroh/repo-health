param([string]$CacheRoot)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'environment.ps1') -CacheRoot $CacheRoot
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    go test ./internal/methodology -count=1 -cover
    if ($LASTEXITCODE -ne 0) { throw 'Methodology tests failed.' }
    go run ./cmd/methodology-check
    if ($LASTEXITCODE -ne 0) { throw 'Methodology scenarios failed.' }
} finally { Pop-Location }
