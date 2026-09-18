param(
    [string]$Repository = 'examples/self-hosted-worker',
    [switch]$Anonymous
)
$ErrorActionPreference = 'Stop'
if ($Repository -notmatch '^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$') { throw 'Укажите организацию/репозиторий без адреса сайта.' }
Add-Type -AssemblyName System.Net.Http
$handler = [System.Net.Http.HttpClientHandler]::new()
$handler.AllowAutoRedirect = $false
$client = [System.Net.Http.HttpClient]::new($handler)
$client.Timeout = [TimeSpan]::FromSeconds(15)
$client.MaxResponseContentBufferSize = 4194304
$client.DefaultRequestHeaders.UserAgent.ParseAdd('RepoHealth-AccessCheck/1.0')
$secure = $null
$credential = $null
$rows = [System.Collections.Generic.List[object]]::new()

function Read-Endpoint([string]$Name, [string]$Uri) {
    $response = $null
    try {
        $response = $client.GetAsync($Uri).GetAwaiter().GetResult()
        $status = [int]$response.StatusCode
        $detail = switch ($status) {
            200 { 'Доступ разрешён; полнота данных и метрик отдельно не проверялась.' }
            401 { 'Требуется авторизация либо токен не принят.' }
            403 { 'Запрос запрещён; точная причина требует проверки прав и настроек.' }
            404 { 'Не найдено либо скрыто правами; отсутствие данных не доказано.' }
            429 { 'Ограничение частоты запросов.' }
            default { 'Неожиданный HTTP-статус; содержимое ответа не выводится.' }
        }
        $rows.Add([pscustomobject]@{source=$Name;status=$status;detail=$detail})
        if ($status -eq 200 -and $Name -eq 'Репозиторий') {
            try { return ($response.Content.ReadAsStringAsync().GetAwaiter().GetResult() | ConvertFrom-Json) }
            catch { return $null }
        }
    } catch {
        $rows.Add([pscustomobject]@{source=$Name;status=$null;detail='Сетевая ошибка, превышение лимита ответа или тайм-аут. Содержимое ошибки скрыто.'})
    } finally { if ($response) { $response.Dispose() } }
    return $null
}

try {
    if (-not $Anonymous) {
        $credential = $env:SOURCECRAFT_TOKEN
        if ([string]::IsNullOrWhiteSpace($credential)) {
            Write-Host 'Проверка доступа: только чтение. Токен не сохраняется и не печатается.'
            $secure = Read-Host 'PAT SourceCraft (ввод скрыт)' -AsSecureString
            $credential = ([System.Net.NetworkCredential]::new('', $secure)).Password
        }
        if ([string]::IsNullOrWhiteSpace($credential)) { throw 'Токен не введён. Для анонимной проверки используйте -Anonymous.' }
        $client.DefaultRequestHeaders.Authorization = [System.Net.Http.Headers.AuthenticationHeaderValue]::new('Bearer', $credential.Trim())
    }
    $base = 'https://api.sourcecraft.tech'
    $repoPath = '/repos/' + $Repository
    $null = Read-Endpoint 'Каталог публичных репозиториев' ($base + '/repos?page_size=1')
    $meta = Read-Endpoint 'Репозиторий' ($base + $repoPath)
    foreach ($item in @(
        @('Дерево файлов','/trees?page_size=1'),
        @('CI','/cicd/runs?page_size=1'),
        @('Issues','/issues?page_size=1&filter=visibility%3Dpublic'),
        @('MR','/pulls?page_size=1'),
        @('Contributors','/contributors?page_size=1'),
        @('Releases','/releases?page_size=1')
    )) { $null = Read-Endpoint $item[0] ($base + $repoPath + $item[1]) }
    $repoID = $null
    if ($meta) { if ($meta.uuid) { $repoID = $meta.uuid } elseif ($meta.id) { $repoID = $meta.id } }
    if ($repoID) {
        $query = [Uri]::EscapeDataString([string]$repoID)
        $null = Read-Endpoint 'AppSec: сканирования' ('https://appsec.sourcecraft.tech/v1/scans?gitRepo=' + $query + '&pageSize=1&status=FINISHED')
        $null = Read-Endpoint 'AppSec: находки' ('https://appsec.sourcecraft.tech/v1/defect-groups?gitRepo=' + $query + '&pageSize=1')
    } else {
        $rows.Add([pscustomobject]@{source='AppSec';status=$null;detail='Проверка репозитория пропущена: API не вернул идентификатор.'})
    }
    $report = [pscustomobject]@{
        checked_at = [DateTime]::UtcNow.ToString('o')
        repository = $Repository
        authenticated_request = (-not $Anonymous)
        checks = @($rows.ToArray())
    }
    $root = Split-Path $PSScriptRoot -Parent
    $folder = Join-Path $root '.cache'
    New-Item -ItemType Directory -Path $folder -Force | Out-Null
    $file = Join-Path $folder 'sourcecraft-access.json'
    $report | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $file -Encoding UTF8
    $rows | Format-Table source,status,detail -Wrap -AutoSize
    Write-Host "Отчёт сохранён: $file"
    Write-Host 'HTTP 200 подтверждает доступ к методу, но не наличие данных для всех метрик. Токен и содержимое репозитория не сохранены.'
} finally {
    $credential = $null
    if ($secure) { $secure.Dispose() }
    $client.Dispose()
}
