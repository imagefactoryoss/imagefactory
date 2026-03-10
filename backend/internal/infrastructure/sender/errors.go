package sender

import (
	"context"
	"errors"
	"strings"
)

// ErrorType represents the category of an email sending error
type ErrorType string

const (
	ErrorTypeConnection     ErrorType = "ConnectionError"
	ErrorTypeAuthentication ErrorType = "AuthenticationError"
	ErrorTypeValidation     ErrorType = "ValidationError"
	ErrorTypeTimeout        ErrorType = "TimeoutError"
)

// EmailError is the base interface for all email sending errors
type EmailError interface {
	error
	ErrorType() string
	IsRetryable() bool
}

// ConnectionError represents a connection failure
type ConnectionError struct {
	Err error
}

func (e *ConnectionError) Error() string {
	return "connection error: " + e.Err.Error()
}

func (e *ConnectionError) ErrorType() string {
	return string(ErrorTypeConnection)
}

func (e *ConnectionError) IsRetryable() bool {
	return true
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// NewConnectionError creates a new connection error
func NewConnectionError(err error) *ConnectionError {
	return &ConnectionError{Err: err}
}

// AuthenticationError represents an authentication failure
type AuthenticationError struct {
	Err error
}

func (e *AuthenticationError) Error() string {
	return "authentication error: " + e.Err.Error()
}

func (e *AuthenticationError) ErrorType() string {
	return string(ErrorTypeAuthentication)
}

func (e *AuthenticationError) IsRetryable() bool {
	return false
}

func (e *AuthenticationError) Unwrap() error {
	return e.Err
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(err error) *AuthenticationError {
	return &AuthenticationError{Err: err}
}

// ValidationError represents a validation failure
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return "validation error: " + e.Message
}

func (e *ValidationError) ErrorType() string {
	return string(ErrorTypeValidation)
}

func (e *ValidationError) IsRetryable() bool {
	return false
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}

// TimeoutError represents a timeout failure
type TimeoutError struct {
	Err error
}

func (e *TimeoutError) Error() string {
	return "timeout error: " + e.Err.Error()
}

func (e *TimeoutError) ErrorType() string {
	return string(ErrorTypeTimeout)
}

func (e *TimeoutError) IsRetryable() bool {
	return true
}

func (e *TimeoutError) Unwrap() error {
	return e.Err
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(err error) *TimeoutError {
	return &TimeoutError{Err: err}
}

// IsRetryableError determines if an error should trigger a retry
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for typed errors first
	var emailErr EmailError
	if errors.As(err, &emailErr) {
		return emailErr.IsRetryable()
	}

	// Check for context errors
	if errors.Is(err, context.Canceled) {
		return false // Don't retry if context was explicitly canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true // Retry on timeout
	}

	// Check error message for known patterns
	errMsg := strings.ToLower(err.Error())

	// Non-retryable errors (permanent failures)
	nonRetryablePatterns := []string{
		"authentication failed",
		"invalid credentials",
		"535", // SMTP auth failed
		"550", // Mailbox unavailable, recipient rejected
		"551", // User not local
		"552", // Exceeded storage allocation
		"553", // Mailbox name not allowed
		"554", // Transaction failed, relay denied
		"recipient rejected",
		"mailbox unavailable",
		"relay denied",
		"invalid recipient",
		"invalid email",
	}

	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return false
		}
	}

	// Retryable errors (temporary failures)
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"i/o timeout",
		"timeout",
		"temporary failure",
		"421", // Service not available
		"450", // Mailbox unavailable (temporary)
		"451", // Local error in processing
		"452", // Insufficient system storage
		"dial tcp",
		"no route to host",
		"network is unreachable",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	// Default to retryable for unknown errors (safer to retry)
	return true
}

// ClassifyError wraps an error with the appropriate error type based on its content
func ClassifyError(err error) error {
	if err == nil {
		return nil
	}

	// Already classified
	var emailErr EmailError
	if errors.As(err, &emailErr) {
		return err
	}

	errMsg := strings.ToLower(err.Error())

	// Check for authentication errors
	authPatterns := []string{
		"authentication failed",
		"invalid credentials",
		"535",
	}
	for _, pattern := range authPatterns {
		if strings.Contains(errMsg, pattern) {
			return NewAuthenticationError(err)
		}
	}

	// Check for timeout errors
	timeoutPatterns := []string{
		"timeout",
		"i/o timeout",
		"connection timeout",
	}
	for _, pattern := range timeoutPatterns {
		if strings.Contains(errMsg, pattern) {
			return NewTimeoutError(err)
		}
	}

	// Check for connection errors
	connPatterns := []string{
		"connection refused",
		"connection reset",
		"dial tcp",
		"no route to host",
		"network is unreachable",
	}
	for _, pattern := range connPatterns {
		if strings.Contains(errMsg, pattern) {
			return NewConnectionError(err)
		}
	}

	// Check for validation errors
	validationPatterns := []string{
		"invalid recipient",
		"invalid email",
		"recipient rejected",
		"550",
		"553",
	}
	for _, pattern := range validationPatterns {
		if strings.Contains(errMsg, pattern) {
			return NewValidationError(err.Error())
		}
	}

	// Return original error if no classification matches
	return err
}
