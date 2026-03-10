package build

import (
	"errors"
	"strings"
)

// ValidationError represents a request-level error that should surface as HTTP 400.
type ValidationError struct {
	message string
	cause   error
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

func (e *ValidationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func newValidationError(message string, cause error) *ValidationError {
	return &ValidationError{
		message: message,
		cause:   cause,
	}
}

func classifyCreateBuildError(err error) error {
	if err == nil {
		return nil
	}

	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return err
	}

	if isCreateValidationErrorMessage(err.Error()) {
		return newValidationError(err.Error(), err)
	}

	return err
}

func isCreateValidationErrorMessage(message string) bool {
	lower := strings.ToLower(message)
	validationTokens := []string{
		"is required",
		"invalid",
		"unsupported",
		"must have tekton_enabled=true",
		"does not match infrastructure_type",
		"is not available for this tenant",
		"queue limit reached",
		"concurrent builds limit reached",
		"registry authentication is required",
		"tekton is disabled for this tenant",
	}

	for i := range validationTokens {
		if strings.Contains(lower, validationTokens[i]) {
			return true
		}
	}

	return false
}

