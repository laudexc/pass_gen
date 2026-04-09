SHELL := /bin/sh

.PHONY: fmt-check lint test test-integration build openapi-check migration-check ci

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

openapi-check:
	go run ./cmd/openapicheck docs/openapi.yaml

migration-check:
	go run ./cmd/migrationcheck

ci: fmt-check lint openapi-check test build
