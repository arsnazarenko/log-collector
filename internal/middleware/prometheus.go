package middleware

import (
	"net/http"
	"time"

	"github.com/arsnazarenko/log-collector/internal/metrics"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func PrometheusHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := NewResponseWriter(w)
		start := time.Now()

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		endpoint := r.URL.Path
		if r.URL.RawQuery != "" {
			endpoint += "?" + r.URL.RawQuery
		}

		metrics.RecordHTTPRequest(r.Method, endpoint, rw.statusCode, duration)
	})
}
