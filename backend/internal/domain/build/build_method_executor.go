package build

import (
	"context"
	"encoding/json"
)

// BuildMethodExecutor defines the interface for executing builds with different methods
type BuildMethodExecutor interface {
	// Execute runs a build with the given configuration and returns execution output
	Execute(ctx context.Context, configID string, method BuildMethod) (*MethodExecutionOutput, error)

	// Cancel stops a running execution
	Cancel(ctx context.Context, executionID string) error

	// GetStatus retrieves the current status of an execution
	GetStatus(ctx context.Context, executionID string) (ExecutionStatus, error)

	// Supports checks if this executor supports the given method
	Supports(method BuildMethod) bool
}

// MethodExecutionOutput represents the output from a build method executor
type MethodExecutionOutput struct {
	ExecutionID  string      `json:"execution_id"`
	Status       ExecutionStatus `json:"status"`
	Output       string      `json:"output"`
	ErrorMessage string      `json:"error_message,omitempty"`
	Artifacts    []MethodArtifact `json:"artifacts,omitempty"`
	StartTime    int64       `json:"start_time"`
	EndTime      int64       `json:"end_time,omitempty"`
	Duration     int         `json:"duration_seconds"`
}

// MethodArtifact represents an artifact produced by a build method
type MethodArtifact struct {
	Name        string          `json:"name"`
	Type        string          `json:"type"` // e.g., "image", "binary", "archive"
	Size        int64           `json:"size"`
	Path        string          `json:"path"`
	Checksum    string          `json:"checksum"`
	ContentType string          `json:"content_type"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   int64           `json:"created_at"`
}

// BuildMethodExecutorFactory creates executors for different build methods
type BuildMethodExecutorFactory interface {
	// CreateExecutor creates an executor for the given build method
	CreateExecutor(method BuildMethod) (BuildMethodExecutor, error)

	// GetSupportedMethods returns a list of supported build methods
	GetSupportedMethods() []BuildMethod
}
