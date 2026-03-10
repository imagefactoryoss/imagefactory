package middleware

import (
	"net/http"

	"go.uber.org/zap"
)

// ErrorHandler handles panics and converts them to proper error responses
type ErrorHandler struct {
	logger *zap.Logger
}

// NewErrorHandler creates a new error handler middleware
func NewErrorHandler(logger *zap.Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
	}
}

// Wrap wraps an HTTP handler with panic recovery
func (eh *ErrorHandler) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				eh.logger.Error("Panic recovered",
					zap.String("method", r.Method),
					zap.String("path", r.RequestURI),
					zap.Any("panic", err),
				)

				// Send error response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"internal server error"}}`))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
