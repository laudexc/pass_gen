# pass_gen

`pass_gen` — backend-сервис для безопасной работы с паролями.

Ключевая идея:
- пароль можно принять и обработать,
- но наружу plaintext-пароль не возвращается,
- в БД хранится только Argon2id-хеш,
- для межсервисной передачи используется шифртекст (AES-GCM + Base64).

---

## Что умеет проект

1. Сгенерировать безопасные пароли.
2. Принять пароль пользователя и сохранить только хеш.
3. Проверить пароль против хеша.
4. Оценить сложность пароля.
5. Отдать метрики для Prometheus.

HTTP API:
- `GET /healthz`
- `GET /metrics`
- `POST /v1/passwords/register`
- `POST /v1/passwords/generate`
- `POST /v1/passwords/validate`
- `POST /v1/passwords/strength`

---

## Быстрый старт (самый простой путь)

Инструкция рассчитана на Windows + PowerShell.

### 1. Что должно быть установлено

1. Git
2. Go (актуальная версия)
3. Docker Desktop

Проверка:

```powershell
go version
docker --version
docker compose version
```

### 2. Скачать проект

```powershell
git clone <URL_ТВОЕГО_РЕПО>
cd pass_gen
```

### 3. Создать `.env`

```powershell
Copy-Item .env.example .env
```

### 4. Сгенерировать ключ шифрования

```powershell
go run .\cmd\passgen keygen
```

Скопируй выведенную строку и вставь в `.env`:

```env
PASSGEN_TRANSPORT_KEY_BASE64=СЮДА_ВСТАВЬ_КЛЮЧ
```

### 5. Запустить сервис и БД

```powershell
docker compose up --build
```

После запуска сервис доступен на:
- `http://localhost:8080`

### 6. Проверить, что сервис жив

```powershell
curl http://localhost:8080/healthz
```

Ожидаемо:

```json
{"status":"ok"}
```

---

## Как пользоваться API (по шагам)

## Шаг 1. Зарегистрировать пароль

Запрос:

```powershell
curl -X POST http://localhost:8080/v1/passwords/register `
  -H "Content-Type: application/json" `
  -d "{\"password\":\"MyStrong!Pass123\"}"
```

Что происходит:
1. Пароль принимается backend-ом.
2. В БД сохраняется только Argon2id-хеш.
3. В ответ приходит только `transport_ciphertext`.

## Шаг 2. Сгенерировать пароли

```powershell
curl -X POST http://localhost:8080/v1/passwords/generate `
  -H "Content-Type: application/json" `
  -d "{\"length\":12,\"count\":3}"
```

Что получишь:
- список `transport_ciphertexts`.
- plaintext-пароли не возвращаются.

## Шаг 3. Проверить пароль против хеша

```powershell
curl -X POST http://localhost:8080/v1/passwords/validate `
  -H "Content-Type: application/json" `
  -d "{\"password\":\"MyStrong!Pass123\",\"hash\":\"<ARGON2ID_HASH>\"}"
```

Ожидаемо:

```json
{"valid":true}
```

## Шаг 4. Оценить сложность

```powershell
curl -X POST http://localhost:8080/v1/passwords/strength `
  -H "Content-Type: application/json" `
  -d "{\"password\":\"MyStrong!Pass123\"}"
```

Ожидаемо: вернется `score`, `label` и детали валидации.

## Шаг 5. Проверить метрики

```powershell
curl http://localhost:8080/metrics
```

---

## Если хочешь через Postman

1. Создай запрос.
2. Выбери метод (`GET` или `POST`).
3. URL: `http://localhost:8080/...`
4. Для `POST`: `Body -> raw -> JSON`.
5. Заголовок: `Content-Type: application/json`.

Проверь response headers:
- `X-Request-ID`
- `X-API-Version: v1`

---

## CLI команды (без HTTP)

```powershell
go run .\cmd\passgen keygen
go run .\cmd\passgen generate --length 12 --count 2 --json
go run .\cmd\passgen strength --password "MyStrong!Pass123" --json
```

---

## Тесты и проверки

## Юнит + интеграционные тесты

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

Запуск с профилем observability:

```powershell
docker compose --profile observability up --build
```

После запуска:
- App: `http://localhost:8080`
- Prometheus: `http://localhost:9090`
- Alertmanager: `http://localhost:9093`

---

## Частые проблемы

## Ошибка `directory not found` при `go run ./cmd/passgen ...`

Причина: ты находишься не в корне проекта.

Решение:

```powershell
cd C:\.My\Golang_files\pass_gen
go run .\cmd\passgen keygen
```

Если ты уже в `cmd\passgen`, запускай так:

```powershell
go run . keygen
```

## Ошибка Docker

Проверь, что Docker Desktop запущен.

## Не стартует API из-за БД

Проверь значения в `.env`:
- `POSTGRES_DB`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `PASSGEN_TRANSPORT_KEY_BASE64`

---

## Важно по безопасности

1. Не коммить `.env` в git.
2. Не логируй plaintext-пароли.
3. В проде храни секреты в секрет-менеджере, а не в файле.
4. Для изменений API держи совместимость `v1`.
