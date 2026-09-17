param([string]$CacheRoot, [int]$Port = 0, [string]$DatabasePath)
& (Join-Path $PSScriptRoot 'start.ps1') -Runtime Go -NoTokenPrompt -CacheRoot $CacheRoot -Port $Port -DatabasePath $DatabasePath
exit $LASTEXITCODE
