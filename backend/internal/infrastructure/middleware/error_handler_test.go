package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestErrorHandler_Wrap_PanicRecovery(t *testing.T) {
	logger := zap.NewNop()
	handler := NewErrorHandler(logger)

	// Handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	wrapped := handler.Wrap(panicHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	// Call handler (should not panic)
	wrapped.ServeHTTP(rec, req)

	// Verify response
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	body, _ := io.ReadAll(rec.Body)
	assert.Contains(t, string(body), "INTERNAL_ERROR")
}

func TestErrorHandler_Wrap_NormalExecution(t *testing.T) {
	logger := zap.NewNop()
	handler := NewErrorHandler(logger)

	// Normal handler
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	wrapped := handler.Wrap(okHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	// Call handler
	wrapped.ServeHTTP(rec, req)

	// Verify response
	assert.Equal(t, http.StatusOK, rec.Code)
	body, _ := io.ReadAll(rec.Body)
	assert.Equal(t, "success", string(body))
}
