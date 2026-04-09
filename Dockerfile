# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags='-s -w' -o /out/passgen ./cmd/passgen

FROM scratch
WORKDIR /app
COPY --from=builder /out/passgen /app/passgen

# Non-root runtime user
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/app/passgen"]
CMD ["server", "--addr", ":8080"]
