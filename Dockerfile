FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/passgen ./cmd/passgen

FROM alpine:3.22
WORKDIR /app
RUN adduser -D -H -u 10001 appuser
USER appuser

COPY --from=builder /out/passgen /app/passgen
EXPOSE 8080
ENTRYPOINT ["/app/passgen"]
CMD ["server", "--addr", ":8080"]
