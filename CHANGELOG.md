# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- Structured HTTP logging with request correlation.
- Prometheus metrics endpoint (`/metrics`).
- Configurable rate limiting (`PASSGEN_RATE_LIMIT_RPS`, `PASSGEN_RATE_LIMIT_BURST`).
- OpenAPI contract validation command (`cmd/openapicheck`).
- Migration check command (`cmd/migrationcheck`).
- Integration tests for HTTP + PostgreSQL.
- CI smoke server startup and release workflow.
- API versioning policy, observability docs, SLO, runbook, release process, and env reference.
- Prometheus and Alertmanager local compose profile/configs.
- Pre-release one-command gate (`make pre-release`).
- Manual GitHub post-release smoke workflow (`workflow_dispatch`).
- Launch checklist document for final release/deploy flow.

### Security
- Enforced no plaintext password in API responses.
- API error contract includes `request_id` for safe tracing.
- API responses include `X-API-Version: v1` for explicit contract signaling.
