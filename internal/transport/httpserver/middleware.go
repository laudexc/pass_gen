package httpserver

import (
	"context"
	"encoding/hex"
	"log"
	"net/http"
	"sync"
	"time"

	"crypto/rand"
)

const requestIDHeader = "X-Request-ID"

type middleware func(http.Handler) http.Handler

type contextKey string

const requestIDKey contextKey = "request_id"

func chain(handler http.Handler, middlewares ...middleware) http.Handler {
	wrapped := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}
	return wrapped
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(requestIDHeader)
		if requestID == "" {
			requestID = generateRequestID()
		}

		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		w.Header().Set(requestIDHeader, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered request_id=%s err=%v", requestIDFromContext(r.Context()), rec)
				writeError(w, r, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func rateLimitMiddleware(limiter *tokenRateLimiter) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				writeError(w, r, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requestIDFromContext(ctx context.Context) string {
	if value, ok := ctx.Value(requestIDKey).(string); ok {
		return value
	}
	return ""
}

func generateRequestID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}

type tokenRateLimiter struct {
	mu     sync.Mutex
	tokens int
	max    int
}

func newTokenRateLimiter(refillPerSecond int, burst int) *tokenRateLimiter {
	if refillPerSecond <= 0 {
		refillPerSecond = 1
	}
	if burst <= 0 {
		burst = refillPerSecond
	}

	limiter := &tokenRateLimiter{tokens: burst, max: burst}
	interval := time.Second / time.Duration(refillPerSecond)
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			limiter.mu.Lock()
			if limiter.tokens < limiter.max {
				limiter.tokens++
			}
			limiter.mu.Unlock()
		}
	}()
	return limiter
}

func (l *tokenRateLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.tokens <= 0 {
		return false
	}
	l.tokens--
	return true
}
