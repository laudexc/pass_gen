# pass_gen

`pass_gen` — это сервис для безопасной работы с паролями.

Главное:
- plaintext пароль не возвращается клиенту,
- в БД хранится только Argon2id-хеш,
- между сервисами передается только шифртекст (AES-GCM + Base64).

## Что умеет

- `POST /v1/passwords/register` — принять пароль, сохранить хеш, вернуть шифртекст.
- `POST /v1/passwords/generate` — сгенерировать пароли, сохранить хеши, вернуть шифртексты.
- `POST /v1/passwords/validate` — проверить пароль против хеша.
- `POST /v1/passwords/strength` — оценить сложность пароля.
- `GET /healthz` — health check.
- `GET /metrics` — Prometheus metrics.

---

## Установка и запуск с нуля (очень просто)

Инструкция для Windows + PowerShell.

## 1) Установи зависимости

Нужно:
- Git
- Go
- Docker Desktop

Проверка:

```powershell
go version
docker --version
docker compose version
```

## 2) Склонируй репозиторий

По SSH:

```powershell
git clone git@github.com:laudexc/pass_gen.git
cd pass_gen
```

Или по HTTPS:

```powershell
git clone https://github.com/laudexc/pass_gen
cd pass_gen
```

## 3) Создай `.env`

```powershell
Copy-Item .env.example .env
```

## 4) Сгенерируй ключ и вставь в `.env`

Сгенерировать:

```powershell
go run .\cmd\passgen keygen
```

Открой `.env` и вставь в строку:

```env
PASSGEN_TRANSPORT_KEY_BASE64=СЮДА_КЛЮЧ
```

## 5) Запусти сервис и БД

```powershell
docker compose up --build
```

Сервис будет на `http://localhost:8080`.

## 6) Проверь что всё живо

```powershell
curl http://localhost:8080/healthz
```

Ожидаемо:

```json
{"status":"ok"}
```

---

## Как делать HTTP запросы БЕЗ Postman

Ниже 2 варианта: `Invoke-RestMethod` (самый удобный в PowerShell) и `curl`.

## Вариант A: PowerShell `Invoke-RestMethod`

## 1. Register

```powershell
$body = @{ password = "MyStrong!Pass123" } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "http://localhost:8080/v1/passwords/register" -ContentType "application/json" -Body $body
```

## 2. Generate

```powershell
$body = @{ length = 12; count = 3 } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "http://localhost:8080/v1/passwords/generate" -ContentType "application/json" -Body $body
```

## 3. Strength

```powershell
$body = @{ password = "MyStrong!Pass123" } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "http://localhost:8080/v1/passwords/strength" -ContentType "application/json" -Body $body
```

## 4. Validate

Сначала тебе нужен хеш (например из БД или предыдущего шага в твоем потоке).

```powershell
$body = @{ password = "MyStrong!Pass123"; hash = "<ARGON2ID_HASH>" } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "http://localhost:8080/v1/passwords/validate" -ContentType "application/json" -Body $body
```

## Вариант B: curl

```powershell
curl -X POST http://localhost:8080/v1/passwords/register `
  -H "Content-Type: application/json" `
  -d "{\"password\":\"MyStrong!Pass123\"}"
```

---

## Если хочешь через Postman

Можно, конечно:
- Method: `POST`/`GET`
- URL: `http://localhost:8080/...`
- Для POST: `Body -> raw -> JSON`
- Header: `Content-Type: application/json`

Полезные response headers:
- `X-Request-ID`
- `X-API-Version: v1`

---

## CLI (без HTTP)

```powershell
go run .\cmd\passgen keygen
go run .\cmd\passgen generate --length 12 --count 2 --json
go run .\cmd\passgen strength --password "MyStrong!Pass123" --json
```

---

## Проверки проекта

## Тесты

```powershell
go test ./...
```

## Проверка OpenAPI

```powershell
go run .\cmd\openapicheck docs/openapi.yaml
```

## Проверка миграций

```powershell
$env:PASSGEN_TEST_DSN="postgres://passgen:change-me@localhost:5432/passgen?sslmode=disable"
go run .\cmd\migrationcheck
```

---

## Мониторинг (опционально)

```powershell
docker compose --profile observability up --build
```

После запуска:
- App: `http://localhost:8080`
- Prometheus: `http://localhost:9090`
- Alertmanager: `http://localhost:9093`

---

## Частые ошибки

## Ошибка `directory not found` при `go run ./cmd/passgen ...`

Ты не в корне проекта.

```powershell
cd C:\.My\Golang_files\pass_gen
go run .\cmd\passgen keygen
```

Если ты уже в `cmd\passgen`, запускай так:

```powershell
go run . keygen
```

## Docker не стартует

Проверь, что Docker Desktop запущен.

## API не стартует

Проверь `.env`:
- `POSTGRES_DB`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `PASSGEN_TRANSPORT_KEY_BASE64`

---

## Безопасность

- Не коммить `.env`.
- Не логируй plaintext пароли.
- В проде используй secret manager, а не `.env`.
- Для API держи совместимость `v1`.

---

## Полезные документы

- OpenAPI: `docs/openapi.yaml`
- Версионирование API: `docs/api-versioning.md`
- Наблюдаемость: `docs/observability.md`
- SLO: `docs/slo.md`
- Runbook: `docs/runbook.md`
- Release process: `docs/release-process.md`
- Launch checklist: `docs/launch-checklist.md`
- Env reference: `docs/env.md`
