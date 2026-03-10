package build

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// BuildExecutionRepository defines the interface for persisting build executions
type BuildExecutionRepository interface {
	// Execution CRUD
	SaveExecution(ctx context.Context, execution *BuildExecution) error
	UpdateExecution(ctx context.Context, execution *BuildExecution) error
	GetExecution(ctx context.Context, id uuid.UUID) (*BuildExecution, error)
	GetBuildExecutions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]BuildExecution, int64, error)
	ListRunningExecutions(ctx context.Context) ([]BuildExecution, error)
	GetRunningExecutionForConfig(ctx context.Context, configID uuid.UUID) (*BuildExecution, error)

	// Log management
	AddLog(ctx context.Context, log *ExecutionLog) error
	GetLogs(ctx context.Context, executionID uuid.UUID, limit, offset int) ([]ExecutionLog, int64, error)

	// Status updates
	UpdateExecutionStatus(ctx context.Context, executionID uuid.UUID, status ExecutionStatus) error
	UpdateExecutionMetadata(ctx context.Context, executionID uuid.UUID, metadata []byte) error
	TryAcquireMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error)
	RenewMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error)
	ReleaseMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string) error

	// Cleanup
	DeleteOldExecutions(ctx context.Context, olderThan time.Duration) error

	// Helper
	GetBuildIDFromConfig(ctx context.Context, configID uuid.UUID) (uuid.UUID, error)
}
