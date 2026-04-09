# Observability

## Metrics endpoint

- `GET /metrics`
- Exposes Prometheus metrics.

## Key metrics

- `passgen_http_requests_total{method,path,status}`
- `passgen_http_request_duration_seconds{method,path,status}`
- `passgen_http_in_flight_requests`

## Suggested alerts

See: `deploy/prometheus/alerts.yml`

Included baseline alerts:
- high `5xx` ratio
- high `p95` latency
- excessive `429` throttling

## Logs

HTTP logs are emitted as structured JSON.
Required fields include:
- `request_id`
- `method`
- `path`
- `status`
- `duration_ms`
