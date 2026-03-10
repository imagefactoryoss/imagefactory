package middleware

import (
	"net/http"
	"time"

	"github.com/srikarm/image-factory/internal/infrastructure/metrics"
	"go.uber.org/zap"
)

// HTTPMetricsMiddleware tracks HTTP request metrics
type HTTPMetricsMiddleware struct {
	operationMetrics *metrics.OperationMetrics
	logger           *zap.Logger
}

// NewHTTPMetricsMiddleware creates a new HTTP metrics middleware
func NewHTTPMetricsMiddleware(logger *zap.Logger) *HTTPMetricsMiddleware {
	return &HTTPMetricsMiddleware{
		operationMetrics: &metrics.OperationMetrics{},
		logger:           logger,
	}
}

// Handler wraps an HTTP handler to track metrics
func (hm *HTTPMetricsMiddleware) Handler(name string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // default
		}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start)
		durationMs := int64(duration.Milliseconds())
		hm.operationMetrics.RecordOperation(durationMs)

		hm.logger.Debug("HTTP request completed",
			zap.String("handler", name),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", wrapped.statusCode),
			zap.Int64("duration_ms", durationMs),
		)
	})
}

// GetMetrics returns current metrics
func (hm *HTTPMetricsMiddleware) GetMetrics() (count int64, avgMs float64) {
	count, _, _, _, avgMs = hm.operationMetrics.GetStats()
	return
}

// GetDetailedMetrics returns all operation metrics
func (hm *HTTPMetricsMiddleware) GetDetailedMetrics() (count, totalMs, minMs, maxMs int64, avgMs float64) {
	return hm.operationMetrics.GetStats()
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	if !w.written {
		w.statusCode = statusCode
		w.written = true
		w.ResponseWriter.WriteHeader(statusCode)
	}
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}
