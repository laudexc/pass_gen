# Release Process

This document defines the default release workflow for `pass_gen`.

## Preconditions

Before creating a release tag:

1. CI on target branch is green.
2. `go test ./...` passes locally.
3. OpenAPI validation passes:

```bash
go run ./cmd/openapicheck docs/openapi.yaml
```

4. Migration check passes against target DB:

```bash
PASSGEN_TEST_DSN='<dsn>' go run ./cmd/migrationcheck
```

5. `CHANGELOG.md` contains release notes under `[Unreleased]`.

## Create a release tag

Use semantic versioning (`vMAJOR.MINOR.PATCH`).

```bash
git tag v1.0.0
git push origin v1.0.0
```

Pushing the tag triggers `.github/workflows/release.yml` which:
- runs tests,
- builds binaries for multiple OS/arch targets,
- publishes GitHub release assets,
- builds and pushes Docker image to GHCR.

## Post-release verification

After release job succeeds:

1. Verify release assets are present in GitHub Release page.
2. Verify image exists in GHCR.
3. Deploy the new version to environment.
4. Run smoke checks:

```bash
curl -fsS http://<host>:8080/healthz
curl -fsS http://<host>:8080/metrics | head
```

5. Confirm no regressions in:
- `5xx` ratio,
- p95 latency,
- `429` rate.

You can also run GitHub workflow `Post-Release Smoke` with target `base_url` for automated checks.

## Rollback policy

If post-release checks fail:

1. Redeploy previous known-good tag.
2. Keep new rollout paused.
3. Collect evidence (`request_id`, logs, alert snapshots).
4. Create follow-up patch release.

## Changelog flow

1. Add all user-visible changes into `CHANGELOG.md` under `[Unreleased]`.
2. At release time, move entries into a new version section.
3. Start a fresh `[Unreleased]` section for next cycle.
