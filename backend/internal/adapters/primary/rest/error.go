package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// ErrorResponse represents a standardized API error response
type ErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	TraceID string                 `json:"trace_id,omitempty"`
}

// HTTPError represents an HTTP error with status code
type HTTPError struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]interface{}
	Err        error
}

// Error implements the error interface
func (e *HTTPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(statusCode int, code, message string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		Details:    make(map[string]interface{}),
	}
}

// BadRequest creates a 400 Bad Request error
func BadRequest(message string) *HTTPError {
	return NewHTTPError(http.StatusBadRequest, "BAD_REQUEST", message)
}

// Unauthorized creates a 401 Unauthorized error
func Unauthorized(message string) *HTTPError {
	return NewHTTPError(http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden creates a 403 Forbidden error
func Forbidden(message string) *HTTPError {
	return NewHTTPError(http.StatusForbidden, "FORBIDDEN", message)
}

// NotFound creates a 404 Not Found error
func NotFound(message string) *HTTPError {
	return NewHTTPError(http.StatusNotFound, "NOT_FOUND", message)
}

// MethodNotAllowed creates a 405 Method Not Allowed error
func MethodNotAllowed(message string) *HTTPError {
	return NewHTTPError(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", message)
}

// Conflict creates a 409 Conflict error
func Conflict(message string) *HTTPError {
	return NewHTTPError(http.StatusConflict, "CONFLICT", message)
}

// ValidationError creates a 422 Unprocessable Entity error
func ValidationErrorHTTP(message string) *HTTPError {
	return NewHTTPError(http.StatusUnprocessableEntity, "VALIDATION_ERROR", message)
}

// InternalServer creates a 500 Internal Server Error
func InternalServer(message string) *HTTPError {
	return NewHTTPError(http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

// ServiceUnavailable creates a 503 Service Unavailable error
func ServiceUnavailable(message string) *HTTPError {
	return NewHTTPError(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", message)
}

// WithDetails adds details to the error
func (e *HTTPError) WithDetails(details map[string]interface{}) *HTTPError {
	e.Details = details
	return e
}

// WithDetail adds a single detail to the error
func (e *HTTPError) WithDetail(key string, value interface{}) *HTTPError {
	e.Details[key] = value
	return e
}

// WithCause wraps the underlying error
func (e *HTTPError) WithCause(err error) *HTTPError {
	e.Err = err
	return e
}

// Response returns the API response structure
func (e *HTTPError) Response() ErrorResponse {
	return ErrorResponse{
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
	}
}

// ResponseWithContext returns the API response structure with trace ID from context
func (e *HTTPError) ResponseWithContext(ctx context.Context) ErrorResponse {
	resp := e.Response()
	resp.TraceID = middleware.GetTraceID(ctx)
	return resp
}

// WriteError writes an HTTP error response
func WriteError(w http.ResponseWriter, ctx context.Context, err *HTTPError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode)
	resp := err.ResponseWithContext(ctx)
	// Best effort encoding - if it fails, we've already written headers
	_ = encodeJSON(w, resp)
}
