package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const TraceIDKey = "trace_id"

// TraceIDMiddleware adds a trace ID to each request
type TraceIDMiddleware struct {
	logger *zap.Logger
}

// NewTraceIDMiddleware creates a new trace ID middleware
func NewTraceIDMiddleware(logger *zap.Logger) *TraceIDMiddleware {
	return &TraceIDMiddleware{
		logger: logger,
	}
}

// Wrap adds trace ID to request context and headers
func (tm *TraceIDMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate or get existing trace ID
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// Add to response headers
		w.Header().Set("X-Trace-ID", traceID)

		// Add to context
		ctx := context.WithValue(r.Context(), TraceIDKey, traceID)

		// Call next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetTraceID extracts trace ID from context
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}
