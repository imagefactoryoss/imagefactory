package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"github.com/stretchr/testify/assert"
)

func TestNewHTTPError(t *testing.T) {
	err := NewHTTPError(http.StatusBadRequest, "BAD_REQUEST", "Invalid input")
	
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Equal(t, "BAD_REQUEST", err.Code)
	assert.Equal(t, "Invalid input", err.Message)
	assert.NotNil(t, err.Details)
	assert.Equal(t, 0, len(err.Details))
}

func TestBadRequest(t *testing.T) {
	err := BadRequest("Invalid request")
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Equal(t, "BAD_REQUEST", err.Code)
}

func TestUnauthorized(t *testing.T) {
	err := Unauthorized("Unauthorized access")
	assert.Equal(t, http.StatusUnauthorized, err.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", err.Code)
}

func TestForbidden(t *testing.T) {
	err := Forbidden("Access forbidden")
	assert.Equal(t, http.StatusForbidden, err.StatusCode)
	assert.Equal(t, "FORBIDDEN", err.Code)
}

func TestNotFound(t *testing.T) {
	err := NotFound("Resource not found")
	assert.Equal(t, http.StatusNotFound, err.StatusCode)
	assert.Equal(t, "NOT_FOUND", err.Code)
}

func TestConflict(t *testing.T) {
	err := Conflict("Resource conflict")
	assert.Equal(t, http.StatusConflict, err.StatusCode)
	assert.Equal(t, "CONFLICT", err.Code)
}

func TestValidationErrorHTTP(t *testing.T) {
	err := ValidationErrorHTTP("Validation failed")
	assert.Equal(t, http.StatusUnprocessableEntity, err.StatusCode)
	assert.Equal(t, "VALIDATION_ERROR", err.Code)
}

func TestInternalServer(t *testing.T) {
	err := InternalServer("Internal error")
	assert.Equal(t, http.StatusInternalServerError, err.StatusCode)
	assert.Equal(t, "INTERNAL_ERROR", err.Code)
}

func TestServiceUnavailable(t *testing.T) {
	err := ServiceUnavailable("Service down")
	assert.Equal(t, http.StatusServiceUnavailable, err.StatusCode)
	assert.Equal(t, "SERVICE_UNAVAILABLE", err.Code)
}

func TestWithDetails(t *testing.T) {
	err := BadRequest("Invalid input").WithDetails(map[string]interface{}{
		"field": "email",
		"reason": "invalid format",
	})
	
	assert.Equal(t, 2, len(err.Details))
	assert.Equal(t, "email", err.Details["field"])
	assert.Equal(t, "invalid format", err.Details["reason"])
}

func TestWithDetail(t *testing.T) {
	err := BadRequest("Invalid input").WithDetail("field", "email")
	
	assert.Equal(t, 1, len(err.Details))
	assert.Equal(t, "email", err.Details["field"])
}

func TestWithCause(t *testing.T) {
	underlyingErr := assert.AnError
	err := InternalServer("Failed to process").WithCause(underlyingErr)
	
	assert.Equal(t, underlyingErr, err.Err)
	assert.Contains(t, err.Error(), "Failed to process")
}

func TestResponse(t *testing.T) {
	err := NotFound("User not found").WithDetail("user_id", "123")
	resp := err.Response()
	
	assert.Equal(t, "NOT_FOUND", resp.Code)
	assert.Equal(t, "User not found", resp.Message)
	assert.Equal(t, "123", resp.Details["user_id"])
}

func TestError_Interface(t *testing.T) {
	err := BadRequest("Invalid input")
	
	// Test that HTTPError implements the error interface
	var _ error = err
	assert.NotNil(t, err.Error())
}

func TestResponseWithContext(t *testing.T) {
	traceID := "test-trace-id-12345"
	ctx := context.WithValue(context.Background(), middleware.TraceIDKey, traceID)

	err := BadRequest("Invalid request")
	resp := err.ResponseWithContext(ctx)

	assert.Equal(t, "BAD_REQUEST", resp.Code)
	assert.Equal(t, "Invalid request", resp.Message)
	assert.Equal(t, traceID, resp.TraceID)
}

func TestResponseWithContext_NoTraceID(t *testing.T) {
	ctx := context.Background()

	err := Unauthorized("Unauthorized")
	resp := err.ResponseWithContext(ctx)

	assert.Equal(t, "UNAUTHORIZED", resp.Code)
	assert.Equal(t, "", resp.TraceID) // Should be empty if no trace ID in context
}

func TestWriteError(t *testing.T) {
	traceID := "write-error-trace-id"
	ctx := context.WithValue(context.Background(), middleware.TraceIDKey, traceID)

	err := NotFound("User not found")
	recorder := httptest.NewRecorder()

	WriteError(recorder, ctx, err)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	// Verify response contains trace ID
	assert.Contains(t, recorder.Body.String(), traceID)
	assert.Contains(t, recorder.Body.String(), "NOT_FOUND")
}

func TestWriteError_WithDetails(t *testing.T) {
	ctx := context.Background()
	err := ValidationErrorHTTP("Field validation failed").WithDetail("field", "email")

	recorder := httptest.NewRecorder()
	WriteError(recorder, ctx, err)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "VALIDATION_ERROR")
	assert.Contains(t, recorder.Body.String(), "email")
}

func TestWriteError_InternalServer(t *testing.T) {
	ctx := context.Background()
	err := InternalServer("Database error").WithCause(context.DeadlineExceeded)

	recorder := httptest.NewRecorder()
	WriteError(recorder, ctx, err)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "INTERNAL_ERROR")
}
