param([string]$CacheRoot)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'environment.ps1') -CacheRoot $CacheRoot
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    go test ./... -count=1 -cover
    if ($LASTEXITCODE -ne 0) { throw 'Tests failed.' }
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Static checks failed.' }
    if (Get-Command node -ErrorAction SilentlyContinue) {
        node --check web/app.js
        if ($LASTEXITCODE -ne 0) { throw 'JavaScript syntax check failed.' }
        node --check web/access.js
        if ($LASTEXITCODE -ne 0) { throw 'Access UI syntax check failed.' }
    }
    New-Item -ItemType Directory -Path 'bin' -Force | Out-Null
    go build -trimpath -o 'bin/repo-health.exe' ./cmd/repo-health
    if ($LASTEXITCODE -ne 0) { throw 'Build failed.' }
} finally { Pop-Location }
