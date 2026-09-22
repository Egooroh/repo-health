param([string]$CacheRoot)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'environment.ps1') -CacheRoot $CacheRoot
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    if (-not (Get-Command npm -ErrorAction SilentlyContinue)) { throw 'npm is required to build the frontend (web/dist is embedded into the Go binary).' }
    npm --prefix web/frontend ci
    if ($LASTEXITCODE -ne 0) { throw 'Frontend dependency install failed.' }
    npm --prefix web/frontend run build
    if ($LASTEXITCODE -ne 0) { throw 'Frontend build failed.' }
    go test ./... -count=1 -cover
    if ($LASTEXITCODE -ne 0) { throw 'Tests failed.' }
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Static checks failed.' }
    New-Item -ItemType Directory -Path 'bin' -Force | Out-Null
    go build -trimpath -o 'bin/repo-health.exe' ./cmd/repo-health
    if ($LASTEXITCODE -ne 0) { throw 'Build failed.' }
} finally { Pop-Location }
