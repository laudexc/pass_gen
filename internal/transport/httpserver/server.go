package httpserver

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"

	"pass_gen/internal/usecase"
)

type Server struct {
	processor    *usecase.PasswordProcessor
	transportKey []byte
}

type registerRequest struct {
	Password string `json:"password"`
}

type validateRequest struct {
	Password string `json:"password"`
	Hash     string `json:"hash"`
}

type strengthRequest struct {
	Password string `json:"password"`
}

type generateRequest struct {
	Length int `json:"length"`
	Count  int `json:"count"`
}

func New(processor *usecase.PasswordProcessor, transportKey []byte) (*Server, error) {
	if processor == nil {
		return nil, errors.New("processor is required")
	}
	if len(transportKey) == 0 {
		return nil, errors.New("transport key is required")
	}
	return &Server{processor: processor, transportKey: transportKey}, nil
}

func DecodeTransportKeyBase64(raw string) ([]byte, error) {
	if raw == "" {
		return nil, errors.New("transport key base64 is required")
	}
	key, err := base64.RawStdEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, errors.New("transport key must be 32 bytes")
	}
	return key, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /v1/passwords/register", s.handleRegister)
	mux.HandleFunc("POST /v1/passwords/generate", s.handleGenerate)
	mux.HandleFunc("POST /v1/passwords/validate", s.handleValidate)
	mux.HandleFunc("POST /v1/passwords/strength", s.handleStrength)
	return chain(
		mux,
		requestIDMiddleware,
		recoveryMiddleware,
		rateLimitMiddleware(newTokenRateLimiter(30, 60)),
	)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}

	result, err := s.processor.RegisterPassword(r.Context(), req.Password, s.transportKey)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"stored":               true,
		"transport_ciphertext": result.TransportCiphertext,
	})
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}

	results, err := s.processor.GenerateAndRegister(r.Context(), req.Length, req.Count, s.transportKey)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	payload := make([]string, 0, len(results))
	for _, item := range results {
		payload = append(payload, item.TransportCiphertext)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"stored":                true,
		"count":                 len(payload),
		"transport_ciphertexts": payload,
	})
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	var req validateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}

	valid, err := s.processor.VerifyPassword(r.Context(), req.Password, req.Hash)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"valid": valid})
}

func (s *Server) handleStrength(w http.ResponseWriter, r *http.Request) {
	var req strengthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}

	strength, err := s.processor.PasswordStrength(r.Context(), req.Password)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, strength)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	payload := map[string]string{"error": msg}
	if requestID := requestIDFromContext(r.Context()); requestID != "" {
		payload["request_id"] = requestID
	}
	writeJSON(w, status, payload)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
