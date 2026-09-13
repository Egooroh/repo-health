$ErrorActionPreference = 'Stop'
$repoHealthRoot = Split-Path $PSScriptRoot -Parent
$repoHealthExecutable = Join-Path $repoHealthRoot 'bin\repo-health.exe'
$repoHealthSecureToken = $null
$repoHealthOldToken = $env:SOURCECRAFT_TOKEN
$repoHealthOldAddress = $env:HTTP_ADDR
$repoHealthOldDatabase = $env:DATABASE_PATH

try {
    if (-not (Test-Path -LiteralPath $repoHealthExecutable -PathType Leaf)) {
        throw 'Готовая сборка bin\repo-health.exe не найдена.'
    }
    Write-Host 'Repo Health — запуск с токеном SourceCraft' -ForegroundColor Green
    Write-Host 'Вставьте только значение PAT. Ввод скрыт; токен не сохраняется в файл.'
    $repoHealthSecureToken = Read-Host 'Токен SourceCraft' -AsSecureString
    $env:SOURCECRAFT_TOKEN = ([System.Net.NetworkCredential]::new('', $repoHealthSecureToken)).Password.Trim()
    if ([string]::IsNullOrWhiteSpace($env:SOURCECRAFT_TOKEN)) {
        throw 'Токен не введён. Предыдущее приложение не остановлено.'
    }
    $repoHealthProcesses = @(Get-Process -Name 'repo-health' -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq $repoHealthExecutable })
    foreach ($repoHealthProcess in $repoHealthProcesses) {
        Write-Host 'Перезапускаем предыдущий экземпляр Repo Health…'
        Stop-Process -InputObject $repoHealthProcess -ErrorAction Stop
        if (-not $repoHealthProcess.WaitForExit(5000)) { throw 'Предыдущий экземпляр ещё не завершился.' }
    }
    $repoHealthProbe = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 8080)
    try { $repoHealthProbe.Start() }
    catch { throw 'Порт 8080 занят другим процессом. Его остановка автоматически не выполняется.' }
    finally { $repoHealthProbe.Stop() }

    $env:HTTP_ADDR = '127.0.0.1:8080'
    $env:DATABASE_PATH = Join-Path $repoHealthRoot 'data\repo-health.db'
    Write-Host 'Откройте http://127.0.0.1:8080 и нажмите «Обновить оценку».' -ForegroundColor Green
    Write-Host 'Оставьте это окно открытым. Для остановки приложения нажмите Ctrl+C.'
    Push-Location $repoHealthRoot
    try {
        & $repoHealthExecutable
        if ($LASTEXITCODE -ne 0) { throw 'Приложение завершилось с ошибкой. Сообщение показано выше.' }
    } finally { Pop-Location }
} catch {
    Write-Host $_.Exception.Message -ForegroundColor Red
} finally {
    $env:SOURCECRAFT_TOKEN = $repoHealthOldToken
    $env:HTTP_ADDR = $repoHealthOldAddress
    $env:DATABASE_PATH = $repoHealthOldDatabase
    if ($repoHealthSecureToken) { $repoHealthSecureToken.Dispose() }
}
