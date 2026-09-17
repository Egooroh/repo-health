# Backwards-compatible entry point; all launch logic lives in start.ps1.
& (Join-Path $PSScriptRoot 'start.ps1') @args
exit $LASTEXITCODE
