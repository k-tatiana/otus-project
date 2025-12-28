package metrics

import (
	"bufio"
	"net"
	"net/http"
	"strconv"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

func HTTPMetricsMiddleware(m *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}
			start := time.Now()

			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start).Seconds()
			endpoint := r.URL.Path
			method := r.Method
			status := strconv.Itoa(rw.statusCode)

			m.HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
			m.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
			m.HTTPResponseSize.WithLabelValues(method, endpoint).Observe(float64(rw.size))

			if r.ContentLength > 0 {
				m.HTTPRequestSize.WithLabelValues(method, endpoint).Observe(float64(r.ContentLength))
			}
		})
	}
}
