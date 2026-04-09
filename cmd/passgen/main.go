package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"pass_gen/internal/repository/postgres"
	"pass_gen/internal/security/password"
	"pass_gen/internal/transport/httpserver"
	"pass_gen/internal/usecase"
)

const (
	exitOK            = 0
	exitValidationErr = 2
	exitRuntimeErr    = 1
)

func main() {
	code := run(os.Args[1:])
	os.Exit(code)
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return exitValidationErr
	}

	switch args[0] {
	case "generate":
		return runGenerate(args[1:])
	case "validate":
		return runValidate(args[1:])
	case "strength":
		return runStrength(args[1:])
	case "keygen":
		return runKeygen(args[1:])
	case "server":
		return runServer(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		printUsage()
		return exitValidationErr
	}
}

func runGenerate(args []string) int {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	length := fs.Int("length", 12, "password length")
	count := fs.Int("count", 1, "number of passwords")
	jsonOut := fs.Bool("json", false, "json output")
	transportKeyB64 := fs.String("transport-key-base64", "", "transport key in base64 (optional)")

	if err := fs.Parse(args); err != nil {
		return exitValidationErr
	}

	key, err := resolveTransportKey(*transportKeyB64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitValidationErr
	}

	processor := usecase.NewPasswordProcessor(nil)
	ctx := context.Background()
	results, err := processor.GenerateAndRegister(ctx, *length, *count, key)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitRuntimeErr
	}

	if *jsonOut {
		payload := struct {
			Stored               bool     `json:"stored"`
			Count                int      `json:"count"`
			TransportCiphertexts []string `json:"transport_ciphertexts"`
		}{Stored: true, Count: len(results), TransportCiphertexts: make([]string, 0, len(results))}
		for _, r := range results {
			payload.TransportCiphertexts = append(payload.TransportCiphertexts, r.TransportCiphertext)
		}
		return printJSON(payload)
	}

	fmt.Printf("stored=true count=%d\n", len(results))
	for i, r := range results {
		fmt.Printf("%d: %s\n", i+1, r.TransportCiphertext)
	}
	return exitOK
}

func runValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	plain := fs.String("password", "", "plaintext password")
	hash := fs.String("hash", "", "argon2id encoded hash")
	jsonOut := fs.Bool("json", false, "json output")

	if err := fs.Parse(args); err != nil {
		return exitValidationErr
	}

	processor := usecase.NewPasswordProcessor(nil)
	valid, err := processor.VerifyPassword(context.Background(), *plain, *hash)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitValidationErr
	}

	if *jsonOut {
		return printJSON(map[string]bool{"valid": valid})
	}

	fmt.Printf("valid=%t\n", valid)
	return exitOK
}

func runStrength(args []string) int {
	fs := flag.NewFlagSet("strength", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	plain := fs.String("password", "", "plaintext password")
	jsonOut := fs.Bool("json", false, "json output")

	if err := fs.Parse(args); err != nil {
		return exitValidationErr
	}

	processor := usecase.NewPasswordProcessor(nil)
	result, err := processor.PasswordStrength(context.Background(), *plain)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitValidationErr
	}

	if *jsonOut {
		return printJSON(result)
	}

	fmt.Printf("score=%d label=%s\n", result.Score, result.Label)
	fmt.Printf("length=%d classes=%d valid=%t\n", result.Validation.Length, result.Validation.ClassesMatched, result.Validation.Valid)
	return exitOK
}

func runServer(args []string) int {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	addr := fs.String("addr", ":8080", "http listen address")
	dsn := fs.String("db-dsn", os.Getenv("PASSGEN_DB_DSN"), "postgres dsn")
	transportKeyB64 := fs.String("transport-key-base64", os.Getenv("PASSGEN_TRANSPORT_KEY_BASE64"), "transport key in base64 (32-byte key)")
	rateLimitRPS := fs.Int("rate-limit-rps", envInt("PASSGEN_RATE_LIMIT_RPS", 30), "max requests per second")
	rateLimitBurst := fs.Int("rate-limit-burst", envInt("PASSGEN_RATE_LIMIT_BURST", 60), "max burst requests")

	if err := fs.Parse(args); err != nil {
		return exitValidationErr
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo, err := postgres.NewRepository(*dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitValidationErr
	}
	defer func() {
		_ = repo.Close()
	}()

	if err := repo.Ping(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitRuntimeErr
	}
	if err := repo.CreateSchema(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitRuntimeErr
	}

	key, err := resolveTransportKey(*transportKeyB64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitValidationErr
	}

	processor := usecase.NewPasswordProcessor(repo)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	srv, err := httpserver.New(
		processor,
		key,
		httpserver.WithLogger(logger),
		httpserver.WithRateLimit(*rateLimitRPS, *rateLimitBurst),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitRuntimeErr
	}

	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("passgen server listening on %s", *addr)
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, err)
		return exitRuntimeErr
	}
	return exitOK
}

func runKeygen(args []string) int {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitValidationErr
	}

	key, err := password.NewTransportKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitRuntimeErr
	}

	fmt.Println(base64.RawStdEncoding.EncodeToString(key))
	return exitOK
}

func resolveTransportKey(raw string) ([]byte, error) {
	if raw != "" {
		key, err := httpserver.DecodeTransportKeyBase64(raw)
		if err != nil {
			return nil, err
		}
		return key, nil
	}

	key, err := password.NewTransportKey()
	if err != nil {
		return nil, err
	}
	return key, nil
}

func printJSON(payload any) int {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitRuntimeErr
	}
	return exitOK
}

func printUsage() {
	fmt.Println("usage: passgen <command> [flags]")
	fmt.Println("commands:")
	fmt.Println("  generate --length 12 --count 1 [--json]")
	fmt.Println("  validate --password <plain> --hash <argon2id> [--json]")
	fmt.Println("  strength --password <plain> [--json]")
	fmt.Println("  keygen")
	fmt.Println("  server --addr :8080 --db-dsn <dsn> --transport-key-base64 <key> [--rate-limit-rps 30 --rate-limit-burst 60]")
}

func envInt(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
