# Launch Checklist

This checklist is the final gate before and after shipping a tagged release.

## 1) Pre-release (local)

1. Export integration DSN:

```bash
export PASSGEN_TEST_DSN='postgres://user:pass@host:5432/db?sslmode=disable'
```

2. Run full pre-release checks:

```bash
make pre-release
```

3. Confirm `CHANGELOG.md` is updated.

## 2) Tag and publish

1. Create release tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

2. Verify GitHub `Release` workflow succeeded.

## 3) Deploy

1. Deploy released binary/image to target environment.
2. Ensure runtime env vars are present:
- `PASSGEN_DB_DSN`
- `PASSGEN_TRANSPORT_KEY_BASE64`
- `PASSGEN_RATE_LIMIT_RPS` (optional)
- `PASSGEN_RATE_LIMIT_BURST` (optional)

## 4) Post-release smoke

1. Run GitHub workflow `Post-Release Smoke` (`workflow_dispatch`).
2. Pass `base_url` of deployed service.
3. Confirm all checks pass:
- `/healthz`
- `/metrics`
- API version header
- register endpoint contract
- strength endpoint contract

## 5) Observe first hour

Monitor:
- 5xx error ratio
- p95 latency
- 429 rate
- DB errors in logs

If unstable, rollback per `docs/runbook.md`.
