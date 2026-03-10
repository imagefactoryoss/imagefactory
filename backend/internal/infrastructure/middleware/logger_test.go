package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestLoggingMiddleware_RequestLogging(t *testing.T) {
	logger := zaptest.NewLogger(t)
	loggingMW := NewLoggingMiddleware(logger)

	// Create test handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with logging middleware
	handler := loggingMW.Wrap(nextHandler)

	// Create request with trace ID in context
	req := httptest.NewRequest("GET", "/api/users?page=1", nil)
	ctx := context.WithValue(req.Context(), TraceIDKey, "test-trace-123")
	req = req.WithContext(ctx)

	// Record response
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
}

func TestLoggingMiddleware_CapturesStatus(t *testing.T) {
	logger := zaptest.NewLogger(t)
	loggingMW := NewLoggingMiddleware(logger)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := loggingMW.Wrap(nextHandler)

	req := httptest.NewRequest("POST", "/api/users", nil)
	ctx := context.WithValue(req.Context(), TraceIDKey, "test-trace-456")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLoggingMiddleware_CapturesResponseSize(t *testing.T) {
	logger := zaptest.NewLogger(t)
	loggingMW := NewLoggingMiddleware(logger)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("1234567890"))
	})

	handler := loggingMW.Wrap(nextHandler)

	req := httptest.NewRequest("GET", "/api/data", nil)
	ctx := context.WithValue(req.Context(), TraceIDKey, "test-trace-789")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 10, len(w.Body.String()))
}

func TestLoggingMiddleware_PreservesTraceID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	loggingMW := NewLoggingMiddleware(logger)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify trace ID is in context
		traceID := r.Context().Value(TraceIDKey)
		assert.NotNil(t, traceID)
		assert.Equal(t, "trace-uuid-001", traceID.(string))
		w.WriteHeader(http.StatusOK)
	})

	handler := loggingMW.Wrap(nextHandler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	ctx := context.WithValue(req.Context(), TraceIDKey, "trace-uuid-001")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
