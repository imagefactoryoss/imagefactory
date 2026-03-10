package build

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// HistoryRepository interface defines persistence operations for BuildHistory value object
type HistoryRepository interface {
	// Write operations
	Save(ctx context.Context, history *BuildHistory) error

	// Read operations - Basic
	FindByID(ctx context.Context, id uuid.UUID) (*BuildHistory, error)
	FindByBuild(ctx context.Context, buildID uuid.UUID) (*BuildHistory, error)

	// Read operations - Query by filters
	FindByProject(ctx context.Context, projectID uuid.UUID) ([]*BuildHistory, error)
	FindByMethod(ctx context.Context, tenantID uuid.UUID, method string) ([]*BuildHistory, error)
	FindSuccessfulByMethod(ctx context.Context, projectID uuid.UUID, method string, limit int) ([]*BuildHistory, error)
	FindRecent(ctx context.Context, tenantID uuid.UUID, since time.Duration) ([]*BuildHistory, error)

	// Analytics
	AverageDurationByMethod(ctx context.Context, projectID uuid.UUID, method string) (time.Duration, error)
	SuccessRateByMethod(ctx context.Context, projectID uuid.UUID, method string) (float64, error)
	CountByProjectAndMethod(ctx context.Context, projectID uuid.UUID, method string) (int, error)
}

// RepositoryError represents persistence-related errors
var (
	ErrHistoryNotFound   = errors.New("build history not found")
	ErrTenantNotFound    = errors.New("tenant not found")
	ErrProjectNotFound   = errors.New("project not found")
	ErrPersistenceFailed = errors.New("persistence operation failed")
	ErrInvalidTenantID   = errors.New("invalid tenant id")
	ErrInvalidProjectID  = errors.New("invalid project id")
	ErrInvalidRequest    = errors.New("invalid request")
	ErrNoHistoryData     = errors.New("no history data available")
	ErrInvalidMethod     = errors.New("invalid build method")
)
