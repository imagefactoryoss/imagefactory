package middleware

import (
	"bufio"
	"fmt"
	"net/http"
	"net"
	"time"

	"go.uber.org/zap"
)

// LoggingMiddleware provides structured request/response logging with trace IDs
type LoggingMiddleware struct {
	logger *zap.Logger
}

// NewLoggingMiddleware creates a new LoggingMiddleware instance
func NewLoggingMiddleware(logger *zap.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
	}
}

// ResponseWriter wraps http.ResponseWriter to capture status code and body size
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
	bodySize   int64
}

// WriteHeader captures the HTTP status code
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures written bytes
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bodySize += int64(n)
	return n, err
}

// Hijack forwards websocket/connection upgrades to the underlying writer when supported.
func (rw *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("hijack: feature not supported")
	}
	return hijacker.Hijack()
}

// Flush forwards streaming flushes to the underlying writer when supported.
func (rw *ResponseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Push forwards HTTP/2 server push to the underlying writer when supported.
func (rw *ResponseWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := rw.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

// Wrap applies logging middleware to the next handler
func (lm *LoggingMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap response writer to capture status and size
		wrapped := &ResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			bodySize:       0,
		}
		
		// Extract trace ID from context
		traceID := r.Context().Value(TraceIDKey).(string)
		
		// Call next handler
		next.ServeHTTP(wrapped, r)
		
		// Calculate request duration
		duration := time.Since(start)
		
		// Log request/response with standard fields
		lm.logger.Info("HTTP request",
			zap.String("trace_id", traceID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("query", r.URL.RawQuery),
			zap.Int("status_code", wrapped.statusCode),
			zap.Int64("response_size", wrapped.bodySize),
			zap.Duration("duration_ms", duration),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
		)
	})
}
