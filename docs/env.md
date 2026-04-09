# Environment Variables

## Runtime variables

- `PASSGEN_DB_DSN`
  - PostgreSQL DSN used by `passgen server`.
  - Example: `postgres://user:pass@host:5432/db?sslmode=disable`

- `PASSGEN_TRANSPORT_KEY_BASE64`
  - 32-byte encryption key encoded as base64.
  - Generate via:
    - `go run ./cmd/passgen keygen`

- `PASSGEN_RATE_LIMIT_RPS`
  - Request refill rate per second for token limiter.
  - Default: `30`

- `PASSGEN_RATE_LIMIT_BURST`
  - Burst capacity for token limiter.
  - Default: `60`

## Test and tooling variables

- `PASSGEN_TEST_DSN`
  - DSN used by integration tests and `cmd/migrationcheck` by default.

## Notes

- Do not commit `.env` with secrets.
- Prefer secret manager / CI secret store for production values.
