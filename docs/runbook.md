# Operational Runbook

This runbook describes first-response actions for common production incidents.

## Quick context

- API base: `http://<host>:8080`
- Health endpoint: `GET /healthz`
- Metrics endpoint: `GET /metrics`
- Required tracing header: `X-Request-ID`
- DB DSN env: `PASSGEN_DB_DSN`
- Rate limit env:
  - `PASSGEN_RATE_LIMIT_RPS`
  - `PASSGEN_RATE_LIMIT_BURST`

---

## Incident: API returns 5xx

### Symptoms

- Alert `PassgenHighErrorRate` firing.
- Increased 5xx in Prometheus metric `passgen_http_requests_total{status=~"5.."}`.

### Immediate checks

1. Verify service health:

```bash
curl -i http://localhost:8080/healthz
```

2. Check recent logs for panic or DB errors:

```bash
docker compose logs app --tail=200
```

3. Confirm DB connectivity from app env values:

```bash
docker compose exec app sh -lc 'echo "$PASSGEN_DB_DSN"'
```

### Recovery actions

1. If DB connectivity issue: restore DB availability/network first.
2. If app panic loop: rollback to last stable release tag.
3. If release-specific regression: disable new rollout and deploy previous image.

### Verify recovery

- 5xx ratio returns to baseline.
- `/healthz` stable.
- No new panic logs.

---

## Incident: High latency (p95)

### Symptoms

- Alert `PassgenHighP95Latency` firing.
- p95 latency > 750ms.

### Immediate checks

1. Confirm endpoint distribution:

```promql
sum(rate(passgen_http_requests_total[5m])) by (path)
```

2. Confirm per-path p95 latency:

```promql
histogram_quantile(0.95, sum(rate(passgen_http_request_duration_seconds_bucket[5m])) by (le, path))
```

3. Inspect DB saturation / slow operations.

### Recovery actions

1. Scale app replicas if CPU saturated.
2. Tune DB pool limits if DB is bottleneck.
3. Reduce expensive traffic burst using tighter ingress controls.

### Verify recovery

- p95 drops below SLO threshold.
- no elevated 5xx while latency decreases.

---

## Incident: Rate-limit flood (429 spike)

### Symptoms

- Alert `PassgenRateLimitFlood` firing.
- High `429` in `passgen_http_requests_total{status="429"}`.

### Immediate checks

1. Confirm source pattern from ingress/proxy logs.
2. Determine if legitimate traffic increase or abuse.

### Recovery actions

1. For abuse:
   - block/limit upstream source at gateway/WAF.
2. For legitimate load:
   - temporarily raise:
     - `PASSGEN_RATE_LIMIT_RPS`
     - `PASSGEN_RATE_LIMIT_BURST`
   - scale application instances.

### Verify recovery

- 429 ratio normalizes.
- latency and 5xx remain healthy.

---

## Incident: DB unavailable

### Symptoms

- app startup fails or requests return errors tied to DB.
- migration check fails.

### Immediate checks

1. DB health:

```bash
docker compose ps db
docker compose logs db --tail=200
```

2. Validate DSN credentials and host.

3. Validate schema presence:

```bash
PASSGEN_TEST_DSN='<dsn>' go run ./cmd/migrationcheck
```

### Recovery actions

1. Restore DB service first.
2. Re-apply schema check with `cmd/migrationcheck`.
3. Restart app after DB recovers.

### Verify recovery

- `migration check passed`.
- `/healthz` and write endpoints return success.

---

## Deployment rollback procedure

1. Identify last known good tag `vX.Y.Z`.
2. Redeploy previous release artifact/image.
3. Watch:
   - error rate
   - latency
   - 429 rate
4. Keep rollout frozen until root cause is documented in postmortem.

---

## Post-incident checklist

1. Capture timeline and blast radius.
2. Attach representative `request_id` samples.
3. Add action items (code, tests, alerts, docs).
4. Update `CHANGELOG.md` and this runbook if workflow changes.
