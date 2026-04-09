package httpserver

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metrics struct {
	inFlight     prometheus.Gauge
	httpRequests *prometheus.CounterVec
	httpLatency  *prometheus.HistogramVec
	registry     *prometheus.Registry
	once         sync.Once
}

func newMetrics() *metrics {
	registry := prometheus.NewRegistry()
	m := &metrics{
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "passgen_http_in_flight_requests",
			Help: "Current number of in-flight HTTP requests.",
		}),
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "passgen_http_requests_total",
			Help: "Total number of HTTP requests.",
		}, []string{"method", "path", "status"}),
		httpLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "passgen_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path", "status"}),
		registry: registry,
	}

	registry.MustRegister(m.inFlight, m.httpRequests, m.httpLatency)
	return m
}

func (m *metrics) handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *metrics) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m.inFlight.Inc()
		defer m.inFlight.Dec()

		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)

		status := strconv.Itoa(rec.statusCode)
		path := r.URL.Path
		method := r.Method
		duration := time.Since(start).Seconds()

		m.httpRequests.WithLabelValues(method, path, status).Inc()
		m.httpLatency.WithLabelValues(method, path, status).Observe(duration)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}
