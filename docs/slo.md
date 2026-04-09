# Service Level Objectives (SLO)

This document defines baseline SLO targets for `pass_gen` API in production-like environments.

## Scope

Applies to HTTP endpoints:
- `GET /healthz`
- `POST /v1/passwords/register`
- `POST /v1/passwords/generate`
- `POST /v1/passwords/validate`
- `POST /v1/passwords/strength`

## SLI and targets

### Availability SLO

- SLI: successful request ratio for API endpoints.
- Formula:

  `1 - (5xx responses / total responses)`

- Target: **99.9% per rolling 30 days**.

### Latency SLO

- SLI: p95 request latency for API endpoints.
- Formula:

  `histogram_quantile(0.95, rate(passgen_http_request_duration_seconds_bucket[5m]))`

- Target: **p95 < 750ms** for rolling 30 days.

### Throttling SLO

- SLI: rate-limited response ratio (`429`).
- Target: **< 1%** of total requests over rolling 30 days.

## Error budget policy

- If availability error budget consumption exceeds 50% within half period, freeze non-critical feature rollouts.
- If exceeds 100%, prioritize reliability work until recovery trend is confirmed.

## Alert mapping

Baseline alert rules are in `deploy/prometheus/alerts.yml`:
- `PassgenHighErrorRate`
- `PassgenHighP95Latency`
- `PassgenRateLimitFlood`

These alerts are early warning signals and not direct SLO breach alarms.

## Operational notes

- Correlate incidents with `X-Request-ID` and structured HTTP logs.
- For DB-related incidents, verify `PASSGEN_DB_DSN`, DB saturation, and migration state first.
- For rate-limit alerts, verify traffic profile before tuning `PASSGEN_RATE_LIMIT_RPS` and `PASSGEN_RATE_LIMIT_BURST`.
