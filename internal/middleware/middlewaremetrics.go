package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Matthew-Ing/MI_API_Gateway/internal/metrics"
)

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(c int) {
	w.code = c
	w.ResponseWriter.WriteHeader(c)
}

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, code: 200}
		next.ServeHTTP(sw, r)
		metrics.Requests.WithLabelValues(r.Method, strconv.Itoa(sw.code)).Inc()
		metrics.Duration.WithLabelValues(r.Method).Observe(time.Since(start).Seconds())
	})
}
