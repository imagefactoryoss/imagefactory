package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type HTTPRequestSignalSnapshot struct {
	WindowStartedAt  time.Time
	WindowEndedAt    time.Time
	RequestCount     int64
	ServerErrorCount int64
	ClientErrorCount int64
	TotalLatencyMs   int64
	MaxLatencyMs     int64
}

type HTTPRequestSignalStore struct {
	mu               sync.Mutex
	windowStartedAt  time.Time
	requestCount     int64
	serverErrorCount int64
	clientErrorCount int64
	totalLatencyMs   int64
	maxLatencyMs     int64
}

func NewHTTPRequestSignalStore() *HTTPRequestSignalStore {
	return &HTTPRequestSignalStore{
		windowStartedAt: time.Now().UTC(),
	}
}

func (s *HTTPRequestSignalStore) Middleware(next http.Handler) http.Handler {
	if s == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldSkipHTTPRequestSignals(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		wrapped := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)
		s.record(wrapped.statusCode, time.Since(start))
	})
}

func (s *HTTPRequestSignalStore) SnapshotAndReset(now time.Time) HTTPRequestSignalSnapshot {
	if s == nil {
		return HTTPRequestSignalSnapshot{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if now.IsZero() {
		now = time.Now().UTC()
	}
	snapshot := HTTPRequestSignalSnapshot{
		WindowStartedAt:  s.windowStartedAt,
		WindowEndedAt:    now,
		RequestCount:     s.requestCount,
		ServerErrorCount: s.serverErrorCount,
		ClientErrorCount: s.clientErrorCount,
		TotalLatencyMs:   s.totalLatencyMs,
		MaxLatencyMs:     s.maxLatencyMs,
	}
	s.windowStartedAt = now
	s.requestCount = 0
	s.serverErrorCount = 0
	s.clientErrorCount = 0
	s.totalLatencyMs = 0
	s.maxLatencyMs = 0
	return snapshot
}

func (s *HTTPRequestSignalStore) record(statusCode int, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.windowStartedAt.IsZero() {
		s.windowStartedAt = time.Now().UTC()
	}

	durationMs := duration.Milliseconds()
	s.requestCount++
	s.totalLatencyMs += durationMs
	if durationMs > s.maxLatencyMs {
		s.maxLatencyMs = durationMs
	}
	if statusCode >= 500 {
		s.serverErrorCount++
	} else if statusCode >= 400 {
		s.clientErrorCount++
	}
}

func shouldSkipHTTPRequestSignals(path string) bool {
	if path == "/health" || path == "/healthz" || path == "/ready" || path == "/alive" {
		return true
	}
	return strings.HasSuffix(path, "/logs/stream")
}
