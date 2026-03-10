package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestHTTPMetricsMiddleware_Handler(t *testing.T) {
	logger := zap.NewNop()
	hm := NewHTTPMetricsMiddleware(logger)

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate work
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrappedHandler := hm.Handler("test_handler", handler)

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify metrics recorded
	count, avgMs := hm.GetMetrics()
	if count != 1 {
		t.Errorf("expected 1 request recorded, got %d", count)
	}

	if avgMs < 10 {
		t.Errorf("expected average duration >= 10ms, got %.2fms", avgMs)
	}
}

func TestHTTPMetricsMiddleware_MultipleRequests(t *testing.T) {
	logger := zap.NewNop()
	hm := NewHTTPMetricsMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := hm.Handler("test_handler", handler)

	// Make 3 requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}

	// Verify metrics
	count, _, _, _, avgMs := hm.GetDetailedMetrics()
	if count != 3 {
		t.Errorf("expected 3 requests recorded, got %d", count)
	}

	if avgMs < 5 {
		t.Errorf("expected average duration >= 5ms, got %.2fms", avgMs)
	}
}

func TestHTTPMetricsMiddleware_ErrorStatus(t *testing.T) {
	logger := zap.NewNop()
	hm := NewHTTPMetricsMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error"))
	})

	wrappedHandler := hm.Handler("error_handler", handler)

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	count, _ := hm.GetMetrics()
	if count != 1 {
		t.Errorf("expected 1 request recorded, got %d", count)
	}
}

func TestHTTPMetricsMiddleware_GetMetrics(t *testing.T) {
	logger := zap.NewNop()
	hm := NewHTTPMetricsMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := hm.Handler("test", handler)

	// No requests yet
	count, avgMs := hm.GetMetrics()
	if count != 0 || avgMs != 0 {
		t.Errorf("expected no metrics yet, got count=%d, avgMs=%.2f", count, avgMs)
	}

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Check metrics
	count, avgMs = hm.GetMetrics()
	if count != 1 {
		t.Errorf("expected 1 request, got %d", count)
	}
	if avgMs < 0 {
		t.Errorf("expected positive duration, got %.2f", avgMs)
	}
}

func TestHTTPMetricsMiddleware_GetDetailedMetrics(t *testing.T) {
	logger := zap.NewNop()
	hm := NewHTTPMetricsMiddleware(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Millisecond) // Ensure measurable duration
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := hm.Handler("test", handler)

	// Make 2 requests
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}

	count, totalMs, minMs, maxMs, avgMs := hm.GetDetailedMetrics()

	if count != 2 {
		t.Errorf("expected count=2, got %d", count)
	}

	if totalMs <= 0 {
		t.Errorf("expected totalMs > 0, got %d", totalMs)
	}

	if minMs < 0 {
		t.Errorf("expected minMs >= 0, got %d", minMs)
	}

	if maxMs < minMs {
		t.Errorf("expected maxMs >= minMs, got %d >= %d", maxMs, minMs)
	}

	if count > 0 {
		expectedAvg := float64(totalMs) / float64(count)
		if avgMs != expectedAvg {
			t.Errorf("expected avgMs=%.2f, got %.2f", expectedAvg, avgMs)
		}
	}
}

func TestResponseWriterWrapper(t *testing.T) {
	recorder := httptest.NewRecorder()
	wrapper := &responseWriterWrapper{
		ResponseWriter: recorder,
	}

	// Test WriteHeader
	wrapper.WriteHeader(http.StatusNotFound)
	if wrapper.statusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", wrapper.statusCode)
	}

	// Test that second WriteHeader is ignored
	wrapper.WriteHeader(http.StatusInternalServerError)
	if wrapper.statusCode != http.StatusNotFound {
		t.Errorf("expected status to remain 404, got %d", wrapper.statusCode)
	}
}

func TestResponseWriterWrapper_Write(t *testing.T) {
	recorder := httptest.NewRecorder()
	wrapper := &responseWriterWrapper{
		ResponseWriter: recorder,
	}

	// Write without explicit WriteHeader
	data := []byte("test data")
	n, err := wrapper.Write(data)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if n != len(data) {
		t.Errorf("expected %d bytes written, got %d", len(data), n)
	}

	// Status should default to OK
	if wrapper.statusCode != http.StatusOK {
		t.Errorf("expected default status 200, got %d", wrapper.statusCode)
	}
}
