SHELL := /bin/sh

.PHONY: fmt-check lint test test-integration build openapi-check migration-check ci smoke-server pre-release

fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt check failed:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

lint:
	go vet ./...

test:
	go test ./...

test-integration:
	go test ./internal/repository/postgres ./internal/transport/httpserver

build:
	go build ./cmd/passgen

smoke-server:
	@PASSGEN_DB_DSN=$${PASSGEN_TEST_DSN} PASSGEN_TRANSPORT_KEY_BASE64=$$(go run ./cmd/passgen keygen) \
		go run ./cmd/passgen server --addr 127.0.0.1:18080 > /tmp/passgen-server.log 2>&1 & \
		pid=$$!; \
		trap "kill $$pid >/dev/null 2>&1 || true; cat /tmp/passgen-server.log >/dev/null 2>&1 || true" EXIT; \
		for i in 1 2 3 4 5 6 7 8 9 10; do \
			curl -fsS http://127.0.0.1:18080/healthz >/dev/null && break; \
			sleep 1; \
		done; \
		curl -fsS http://127.0.0.1:18080/healthz >/dev/null; \
		curl -fsS http://127.0.0.1:18080/metrics >/dev/null; \
		kill $$pid >/dev/null 2>&1 || true; \
		wait $$pid >/dev/null 2>&1 || true

openapi-check:
	go run ./cmd/openapicheck docs/openapi.yaml

migration-check:
	go run ./cmd/migrationcheck

pre-release:
	@if [ -z "$$PASSGEN_TEST_DSN" ]; then \
		echo "PASSGEN_TEST_DSN is required for pre-release checks"; \
		exit 2; \
	fi
	$(MAKE) fmt-check
	$(MAKE) lint
	$(MAKE) openapi-check
	$(MAKE) migration-check
	$(MAKE) test
	$(MAKE) build

ci: fmt-check lint openapi-check test build
