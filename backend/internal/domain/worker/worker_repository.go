package worker

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// Repository interface defines persistence operations for Worker aggregate
type Repository interface {
	// Write operations
	Save(ctx context.Context, worker *Worker) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Read operations
	FindByID(ctx context.Context, id uuid.UUID) (*Worker, error)
	FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*Worker, error)
	FindAvailable(ctx context.Context, tenantID uuid.UUID) ([]*Worker, error)
	FindByType(ctx context.Context, tenantID uuid.UUID, workerType string) ([]*Worker, error)

	// Health queries
	FindUnhealthy(ctx context.Context, tenantID uuid.UUID) ([]*Worker, error)
	FindStale(ctx context.Context, threshold string) ([]*Worker, error)
}

// RepositoryError represents persistence-related errors
var (
	ErrWorkerNotFound     = errors.New("worker not found")
	ErrTenantNotFound     = errors.New("tenant not found")
	ErrPersistenceFailed  = errors.New("persistence operation failed")
	ErrInvalidTenantID    = errors.New("invalid tenant id")
	ErrInvalidWorkerID    = errors.New("invalid worker id")
	ErrInvalidRequest     = errors.New("invalid request")
	ErrWorkerNameRequired = errors.New("worker name is required")
	ErrInvalidCapacity    = errors.New("capacity must be greater than 0")
	ErrTenantMismatch     = errors.New("tenant mismatch")
	ErrInvalidLoad        = errors.New("invalid load")
	ErrLoadExceedsCapacity = errors.New("load exceeds capacity")
)
