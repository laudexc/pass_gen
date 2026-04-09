# API Versioning Policy

Current API version is `v1`.

## Compatibility rules

- All endpoints are namespaced under `/v1/...`.
- Non-breaking changes are allowed in `v1`:
  - adding optional request fields
  - adding response fields
  - adding new endpoints under `/v1`
- Breaking changes require a new major API namespace (`/v2`).

## Deprecation process

1. Mark endpoint/field as deprecated in OpenAPI.
2. Keep compatibility period for at least one release cycle.
3. Announce removal in release notes and `CHANGELOG.md`.
4. Remove only in the next major API namespace.

## Runtime signaling

- Server returns `X-API-Version: v1` for JSON API responses.
- `X-Request-ID` is returned for request tracing.
