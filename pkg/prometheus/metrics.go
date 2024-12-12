package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var once = &sync.Once{}

var (
	requestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of response time for handler in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
)

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Metrics is a middleware that collects metrics for HTTP requests and exposes them for Prometheus
func Metrics(next http.Handler) http.Handler {
	once.Do(func() {
		prometheus.MustRegister(requestCount, requestDuration)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		srw := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(srw, r)
		duration := time.Since(start)

		requestCount.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(srw.status)).Inc()
		requestDuration.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(srw.status)).Observe(duration.Seconds())
	})
}

// ExposeHTTP returns a http.Handler that exposes the metrics
func ExposeHTTP() http.Handler {
	return promhttp.Handler()
}
