# Проверка доступности SourceCraft — 18 сентября 2026

Только GET, без токена, без запуска сканирований и изменения репозиториев. Проверен публичный пример `examples/self-hosted-worker`. В окружении проверки SOURCECRAFT_TOKEN отсутствует, локальный сервер недоступен, файла .env в проекте нет. Права конкретного аккаунта не проверены. У владельца проекта пока нет собственных репозиториев.

## Живые ответы

| Источник | URL/метод | Результат |
| --- | --- | --- |
| Спецификация основного API | https://api.sourcecraft.tech/sourcecraft.swagger.json | 200, JSON |
| Документация AppSec | https://appsec.sourcecraft.tech/swagger-ui/ | 200, HTML |
| Спецификация AppSec | https://appsec.sourcecraft.tech/openapi | 200, JSON |
| Каталог | GET /repos?page_size=1 | 401 |
| Метаданные примера | GET /repos/examples/self-hosted-worker | 401 |
| Дерево | GET /repos/examples/self-hosted-worker/trees?page_size=1 | 401 |
| CI | GET /repos/examples/self-hosted-worker/cicd/runs?page_size=1 | 401 |
| Issues | GET /repos/examples/self-hosted-worker/issues?page_size=1&filter=visibility%3Dpublic | 401 |
| MR | GET /repos/examples/self-hosted-worker/pulls?page_size=1 | 401 |
| Contributors | GET /repos/examples/self-hosted-worker/contributors?page_size=1 | 401 |
| Releases | GET /repos/examples/self-hosted-worker/releases?page_size=1 | 401 |
| AppSec, проверка входа без параметра gitRepo | GET https://appsec.sourcecraft.tech/v1/scans | 401 |
| Git HTTPS | GET https://git.sourcecraft.dev/examples/self-hosted-worker.git/info/refs?service=git-upload-pack | 200, application/x-git-upload-pack-advertisement |

401 без токена не доказывает, что репозиторий приватный или что доступ возможен только владельцу. Публичный Git-интерфейс действительно ответил без авторизации; клонирование всего репозитория не выполнялось. 200 у спецификации не означает доступ к самим отчётам. AppSec-вызов без gitRepo проверяет только реакцию входной точки, не права на конкретный репозиторий.

## Что подтверждено спецификациями

- `GET /repos` (DiscoverRepositories): пустой filter возвращает публичные репозитории, есть пагинация, сортировка по created_at/rating. Перечень не ограничен собственными репозиториями по контракту метода. Нужен живой запрос с PAT для проверки фактического доступа.
- Есть методы дерева, issues/комментариев, MR/комментариев/ревьюеров, запусков CI, contributors, releases и tags. Наличие методов не подтверждает полноту необходимых событий, роли авторов ответов, все статусы и пригодность для метрик rc1.
- AppSec `GET /v1/scans`, `/v1/scans/latest`, `/v1/defect-groups`, `/v1/defect-groups/types`, `/v1/activities` документированы. gitRepo — внутренний ID или публичный UUID из метаданных репозитория; org/repo не является значением этого параметра.
- Списки AppSec используют pageSize/pageToken; основной API — page_size/page_token. Для AppSec документирован максимум pageSize=250.
- Фильтр severity AppSec включает NONE, LOW, MEDIUM, HIGH, CRITICAL; типов больше, чем три используемых нами. Встречаются статусы вроде TRIAGED_TP. Нужен явный адаптер в нормализованные enum методики, нельзя считать их уже совпадающими.
- Swagger AppSec содержит противоречия: некоторые параметры помечены required, но их описания допускают отсутствие/пустое значение. Ответ 400 при проверке требует сверки контракта, а не вывода об отсутствии прав. Способ аутентификации не описан securitySchemes в полученной спецификации; принятие PAT этим API ещё требует проверки.

## Следующая проверка

Из корня проекта: `powershell -NoProfile -File ./scripts/check-sourcecraft-access.ps1`. По умолчанию проверяется чужой публичный пример; собственный репозиторий не нужен. PAT вводится скрыто либо берётся из SOURCECRAFT_TOKEN текущего процесса. Не передавать токен в аргументе команды или в чате.

Скрипт использует только фиксированные официальные хосты, запрещает редиректы, ограничивает размер ответа и тайм-аут. Сохраняет только статусы/пояснения в `.cache/sourcecraft-access.json`, без токена, ответов API, кода или находок. Если метаданные не прочитаны, проверки AppSec для репозитория пропускаются. `-Anonymous` повторяет проверку без реквизитов; `-Repository org/repo` выбирает другой доступный репозиторий.

Для положительной проверки AppSec потребуется репозиторий с доступом к отчётам, например командный репозиторий с назначенными правами и выполненными проверками. Создавать такой репозиторий автоматически и менять доступ в рамках этой проверки не требуется.
