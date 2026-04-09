package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"pass_gen/internal/repository/postgres"
	"pass_gen/internal/security/password"
	"pass_gen/internal/transport/httpserver"
	"pass_gen/internal/usecase"
)

func TestServerIntegration_Healthz(t *testing.T) {
	ts, _, cleanup := setupIntegrationServer(t)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID header")
	}
	if resp.Header.Get("X-API-Version") != "v1" {
		t.Fatalf("expected X-API-Version=v1, got %q", resp.Header.Get("X-API-Version"))
	}
}

func TestServerIntegration_MetricsEndpoint(t *testing.T) {
	ts, _, cleanup := setupIntegrationServer(t)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read metrics body failed: %v", err)
	}
	if !strings.Contains(string(body), "passgen_http_requests_total") {
		t.Fatal("expected passgen metrics to be present")
	}
}

func TestServerIntegration_RegisterAndGenerate(t *testing.T) {
	ts, repo, cleanup := setupIntegrationServer(t)
	defer cleanup()

	registerBody := []byte(`{"password":"Abc!1234"}`)
	registerResp := postJSON(t, ts.URL+"/v1/passwords/register", registerBody)
	defer registerResp.Body.Close()

	if registerResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 on register, got %d", registerResp.StatusCode)
	}
	if !strings.Contains(registerResp.Header.Get("Content-Type"), "application/json") {
		t.Fatalf("expected application/json, got %q", registerResp.Header.Get("Content-Type"))
	}
	if registerResp.Header.Get("X-API-Version") != "v1" {
		t.Fatalf("expected X-API-Version=v1, got %q", registerResp.Header.Get("X-API-Version"))
	}

	var registerPayload map[string]any
	if err := json.NewDecoder(registerResp.Body).Decode(&registerPayload); err != nil {
		t.Fatalf("decode register response failed: %v", err)
	}
	if registerPayload["stored"] != true {
		t.Fatalf("expected stored=true, got %v", registerPayload["stored"])
	}
	if registerPayload["transport_ciphertext"] == "" {
		t.Fatal("expected non-empty transport_ciphertext")
	}
	if _, exists := registerPayload["password"]; exists {
		t.Fatal("response must not contain plaintext password")
	}

	ctx := context.Background()
	assertTableCount(t, repo, ctx, "password_hashes", 1)

	generateBody := []byte(`{"length":12,"count":2}`)
	generateResp := postJSON(t, ts.URL+"/v1/passwords/generate", generateBody)
	defer generateResp.Body.Close()

	if generateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 on generate, got %d", generateResp.StatusCode)
	}

	var generatePayload struct {
		Stored               bool     `json:"stored"`
		Count                int      `json:"count"`
		TransportCiphertexts []string `json:"transport_ciphertexts"`
	}
	if err := json.NewDecoder(generateResp.Body).Decode(&generatePayload); err != nil {
		t.Fatalf("decode generate response failed: %v", err)
	}
	if !generatePayload.Stored {
		t.Fatal("expected stored=true")
	}
	if generatePayload.Count != 2 || len(generatePayload.TransportCiphertexts) != 2 {
		t.Fatalf("expected 2 generated ciphertexts, got count=%d len=%d", generatePayload.Count, len(generatePayload.TransportCiphertexts))
	}

	assertTableCount(t, repo, ctx, "password_hashes", 3)
	assertTableCount(t, repo, ctx, "generation_audit", 1)
}

func TestServerIntegration_ErrorContractIncludesRequestID(t *testing.T) {
	ts, _, cleanup := setupIntegrationServer(t)
	defer cleanup()

	resp := postJSON(t, ts.URL+"/v1/passwords/register", []byte(`{"password":""}`))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		t.Fatalf("expected application/json, got %q", resp.Header.Get("Content-Type"))
	}
	if resp.Header.Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID header")
	}
	if resp.Header.Get("X-API-Version") != "v1" {
		t.Fatalf("expected X-API-Version=v1, got %q", resp.Header.Get("X-API-Version"))
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error response failed: %v", err)
	}
	if payload["error"] == "" {
		t.Fatal("expected error field in response")
	}
	if payload["request_id"] == "" {
		t.Fatal("expected request_id field in response body")
	}
}

func TestServerIntegration_ValidateAndStrength(t *testing.T) {
	ts, _, cleanup := setupIntegrationServer(t)
	defer cleanup()

	hash, err := password.HashArgon2id("Abc!1234")
	if err != nil {
		t.Fatalf("HashArgon2id failed: %v", err)
	}

	validateBody := []byte(`{"password":"Abc!1234","hash":"` + hash + `"}`)
	validateResp := postJSON(t, ts.URL+"/v1/passwords/validate", validateBody)
	defer validateResp.Body.Close()

	if validateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 on validate, got %d", validateResp.StatusCode)
	}

	var validatePayload struct {
		Valid bool `json:"valid"`
	}
	if err := json.NewDecoder(validateResp.Body).Decode(&validatePayload); err != nil {
		t.Fatalf("decode validate response failed: %v", err)
	}
	if !validatePayload.Valid {
		t.Fatal("expected valid=true")
	}

	strengthBody := []byte(`{"password":"Abc!1234"}`)
	strengthResp := postJSON(t, ts.URL+"/v1/passwords/strength", strengthBody)
	defer strengthResp.Body.Close()

	if strengthResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 on strength, got %d", strengthResp.StatusCode)
	}

	var strengthPayload struct {
		Score int    `json:"score"`
		Label string `json:"label"`
	}
	if err := json.NewDecoder(strengthResp.Body).Decode(&strengthPayload); err != nil {
		t.Fatalf("decode strength response failed: %v", err)
	}
	if strengthPayload.Score <= 0 {
		t.Fatalf("expected score > 0, got %d", strengthPayload.Score)
	}
	if strengthPayload.Label == "" {
		t.Fatal("expected non-empty label")
	}
}

func setupIntegrationServer(t *testing.T) (*httptest.Server, *postgres.Repository, func()) {
	t.Helper()

	dsn := os.Getenv("PASSGEN_TEST_DSN")
	if dsn == "" {
		t.Skip("PASSGEN_TEST_DSN is not set")
	}

	repo, err := postgres.NewRepository(dsn)
	if err != nil {
		t.Fatalf("NewRepository failed: %v", err)
	}

	ctx := context.Background()
	if err := repo.Ping(ctx); err != nil {
		_ = repo.Close()
		t.Fatalf("Ping failed: %v", err)
	}
	if err := repo.CreateSchema(ctx); err != nil {
		_ = repo.Close()
		t.Fatalf("CreateSchema failed: %v", err)
	}
	if _, err := repo.DB().ExecContext(ctx, "TRUNCATE TABLE password_hashes, generation_audit RESTART IDENTITY"); err != nil {
		_ = repo.Close()
		t.Fatalf("truncate failed: %v", err)
	}

	key, err := password.NewTransportKey()
	if err != nil {
		_ = repo.Close()
		t.Fatalf("NewTransportKey failed: %v", err)
	}

	processor := usecase.NewPasswordProcessor(repo)
	srv, err := httpserver.New(processor, key)
	if err != nil {
		_ = repo.Close()
		t.Fatalf("httpserver.New failed: %v", err)
	}

	ts := httptest.NewServer(srv.Routes())
	cleanup := func() {
		ts.Close()
		_, _ = repo.DB().ExecContext(context.Background(), "TRUNCATE TABLE password_hashes, generation_audit RESTART IDENTITY")
		_ = repo.Close()
	}
	return ts, repo, cleanup
}

func postJSON(t *testing.T, url string, body []byte) *http.Response {
	t.Helper()

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s failed: %v", url, err)
	}
	return resp
}

func assertTableCount(t *testing.T, repo *postgres.Repository, ctx context.Context, table string, expected int) {
	t.Helper()

	query := "SELECT COUNT(*) FROM " + table
	var count int
	if err := repo.DB().QueryRowContext(ctx, query).Scan(&count); err != nil {
		t.Fatalf("count query failed for %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("expected %d rows in %s, got %d", expected, table, count)
	}
}
