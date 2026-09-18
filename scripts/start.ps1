param(
    [ValidateSet('Auto','Go','Docker')][string]$Runtime = 'Auto',
    [ValidateRange(0,65535)][int]$Port = 0,
    [switch]$NoTokenPrompt,
    [string]$CacheRoot,
    [string]$DatabasePath
)
$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
$saved = @{}
foreach ($name in @('SOURCECRAFT_TOKEN','HTTP_ADDR','DATABASE_PATH','REPO_HEALTH_PORT','GOCACHE','GOMODCACHE','GOTMPDIR')) {
    $saved[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
$secureToken = $null
$executable = $null
$exitCode = 0
Push-Location -LiteralPath $root
try {
    if ($Port -eq 0) {
        $Port = 8080
        if ($env:REPO_HEALTH_PORT) {
            $parsedPort = 0
            if (-not [int]::TryParse($env:REPO_HEALTH_PORT, [ref]$parsedPort) -or $parsedPort -lt 1 -or $parsedPort -gt 65535) { throw 'REPO_HEALTH_PORT должен быть числом от 1 до 65535.' }
            $Port = $parsedPort
        }
    }
    if ($Runtime -eq 'Auto') {
        if (Get-Command go -ErrorAction SilentlyContinue) { $Runtime = 'Go' }
        elseif (Get-Command docker -ErrorAction SilentlyContinue) { $Runtime = 'Docker' }
        else { throw 'Установите Go (https://go.dev/dl/) или Docker Desktop с Compose (https://docs.docker.com/get-started/get-docker/), затем откройте новое окно терминала.' }
    }
    if ($Runtime -eq 'Go') {
        if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw 'Go не найден. Установите Go по версии из go.mod или выберите -Runtime Docker.' }
        $probe = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, $Port)
        try { $probe.Start() }
        catch { throw "Порт $Port занят. Остановите предыдущий запуск через Ctrl+C или выберите -Port 8081. Другие процессы не остановлены." }
        finally { $probe.Stop() }
    } else {
        if ($DatabasePath -or $CacheRoot) { throw 'DatabasePath и CacheRoot применимы только к запуску через Go. Docker хранит данные в собственном томе.' }
        if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { throw 'Docker не найден. Установите Docker с Compose или выберите -Runtime Go.' }
        $probePreference = $ErrorActionPreference
        try {
            $ErrorActionPreference = 'Continue'
            & docker compose version *> $null
            $composeExit = $LASTEXITCODE
            & docker info *> $null
            $daemonExit = $LASTEXITCODE
        } finally { $ErrorActionPreference = $probePreference }
        if ($composeExit -ne 0) { throw 'Нужен Docker Compose v2: команда docker compose недоступна.' }
        if ($daemonExit -ne 0) { throw 'Docker не запущен или недоступен. Запустите Docker Desktop (Linux containers) и повторите запуск.' }
    }
    Write-Host 'Личный токен подключается на сайте кнопкой «Подключить токен».'
    Write-Host "Repo Health: $Runtime, адрес http://127.0.0.1:$Port" -ForegroundColor Green
    Write-Host 'Первый запуск может потребовать интернет для зависимостей. Остановка: Ctrl+C.'
    if ($Runtime -eq 'Docker') {
        $env:REPO_HEALTH_PORT = [string]$Port
        & docker compose --project-directory $root -f (Join-Path $root 'compose.yaml') up --build --exit-code-from repo-health
        $exitCode = $LASTEXITCODE
    } else {
        . (Join-Path $PSScriptRoot 'environment.ps1') -CacheRoot $CacheRoot
        $env:HTTP_ADDR = "127.0.0.1:$Port"
        if (-not $DatabasePath) { $DatabasePath = $saved['DATABASE_PATH'] }
        if (-not $DatabasePath) { $DatabasePath = 'data/repo-health.db' }
        if (-not [System.IO.Path]::IsPathRooted($DatabasePath)) { $DatabasePath = Join-Path $root $DatabasePath }
        $env:DATABASE_PATH = [System.IO.Path]::GetFullPath($DatabasePath)
        $buildDir = Join-Path $env:GOTMPDIR 'launcher'
        New-Item -ItemType Directory -Path $buildDir -Force | Out-Null
        $suffix = ''
        if ($env:OS -eq 'Windows_NT') { $suffix = '.exe' }
        $executable = Join-Path $buildDir ('repo-health-' + [guid]::NewGuid().ToString('N') + $suffix)
        Write-Host 'Сборка текущих исходников...'
        & go build -trimpath -o $executable ./cmd/repo-health
        if ($LASTEXITCODE -ne 0) { throw 'Сборка не удалась. Проверьте ошибку Go выше, версию из go.mod и доступ к зависимостям. Старый бинарный файл не запускается.' }
        & $executable
        $exitCode = $LASTEXITCODE
    }
} catch {
    Write-Host $_.Exception.Message -ForegroundColor Red
    $exitCode = 1
} finally {
    foreach ($name in $saved.Keys) { [Environment]::SetEnvironmentVariable($name, $saved[$name], 'Process') }
    if ($secureToken) { $secureToken.Dispose() }
    if ($executable -and (Test-Path -LiteralPath $executable)) { Remove-Item -LiteralPath $executable -Force -ErrorAction SilentlyContinue }
    Pop-Location
}
exit $exitCode
