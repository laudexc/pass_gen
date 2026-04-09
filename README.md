# pass_gen

## Текущий статус

Сделано:
- Генерация пароля через `crypto/rand`.
- Убрана предсказуемость позиции классов символов (добавлен secure shuffle).
- Ошибки из криптографии не игнорируются.
- Добавлен Argon2id (`golang.org/x/crypto/argon2`) для хеширования.
- Пароль не возвращается клиенту обратно.
- Для межсервисной передачи реализовано шифрование (AES-GCM) + Base64.
- В БД должен сохраняться только хеш, не открытый пароль.
- Выполнена базовая структура проекта:
  - `cmd/passgen/main.go` — entrypoint.
  - `internal/security/password` — генерация, hash/verify, transport encryption.
  - `internal/usecase` — сценарии `RegisterPassword`, `VerifyPassword`, `GenerateAndRegister`, `PasswordStrength`.
- Реализованы CLI-команды:
  - `generate`
  - `validate`
  - `strength`
  - `keygen`
  - `server` (запуск HTTP API)
- Подготовлен HTTP-слой:
  - `GET /healthz`
  - `POST /v1/passwords/register`
  - `POST /v1/passwords/generate`
  - `POST /v1/passwords/validate`
  - `POST /v1/passwords/strength`
- Добавлены middleware для HTTP:
  - `X-Request-ID` (корреляция запросов)
  - `panic recovery` (500 без утечки деталей)
  - `rate limit` (базовая защита от перегрузки)
- Подключён PostgreSQL-репозиторий и схема (`migrations/001_init.sql`) для:
  - хранения только Argon2id хешей
  - аудита генераций (length/count/created_at)
- Добавлен OpenAPI контракт: `docs/openapi.yaml`.
- Добавлены интеграционные тесты HTTP + PostgreSQL.
- Добавлен CI workflow `.github/workflows/ci.yml`:
  - `gofmt` check
  - `go vet`
  - OpenAPI check
  - migration check
  - unit + integration tests
  - build
- Добавлены инфраструктурные файлы:
  - `Dockerfile`
  - `docker-compose.yml`
  - `.env.example`
  - `Makefile`

## Что дальше по плану (следующий шаг)

Следующий приоритет: подготовка к релизу и эксплуатация.

1. Добавить structured logging и метрики (Prometheus).
2. Добавить release workflow (теги, артефакты, контейнерный образ).
3. Добавить контрактные тесты для внешних клиентов API.

## Жесткие правила проекта

- Пароли не логировать.
- Пароли не возвращать наружу после приёма на backend.
- В БД хранить только Argon2id-хеш + соль/параметры в encoded-формате.
- Межсервисно передавать только шифртекст (например Base64 поверх AES-GCM).

## Запуск интеграционных тестов (HTTP + PostgreSQL)

Интеграционные тесты используют переменную окружения `PASSGEN_TEST_DSN`.

Пример:

```powershell
$env:PASSGEN_TEST_DSN="postgres://user:pass@localhost:5432/passgen_test?sslmode=disable"
go test ./...
```

Если `PASSGEN_TEST_DSN` не задана, интеграционные тесты автоматически пропускаются.

## Локальный запуск через Docker Compose

1. Скопировать `.env.example` в `.env`.
2. Сгенерировать ключ:

```powershell
go run ./cmd/passgen keygen
```

3. Вставить ключ в `PASSGEN_TRANSPORT_KEY_BASE64` в `.env`.
4. Запустить сервисы:

```powershell
docker compose up --build
```

Сервер будет доступен на `http://localhost:8080`.

## Технический roadmap

1. Этап 1: стабилизация core и тесты.
2. Этап 2: CLI (`generate`, `validate`, `strength`).
3. Этап 3: HTTP API (`/v1/passwords/generate`, `/validate`, `/strength`, `/healthz`).
4. Этап 4: PostgreSQL + миграции + аудит (без хранения открытых паролей).
5. Этап 5: CI/CD, линтеры, безопасность, метрики.
6. Этап 6: минимальный Java-адаптер (только если реально нужен для интеграции).
