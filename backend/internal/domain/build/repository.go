package build

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for build persistence
type Repository interface {
	// Save persists a build
	Save(ctx context.Context, build *Build) error

	// FindByID retrieves a build by ID
	FindByID(ctx context.Context, id uuid.UUID) (*Build, error)

	// FindByIDsBatch retrieves multiple builds by IDs in a single query (avoids N+1)
	FindByIDsBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*Build, error)

	// FindByTenantID retrieves builds for a tenant
	FindByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Build, error)

	// FindByProjectID retrieves builds for a project
	FindByProjectID(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*Build, error)

	// FindByStatus retrieves builds by status
	FindByStatus(ctx context.Context, status BuildStatus, limit, offset int) ([]*Build, error)

	// Update updates an existing build
	Update(ctx context.Context, build *Build) error

	// UpdateStatus updates the status and execution timestamps for a build
	UpdateStatus(ctx context.Context, id uuid.UUID, status BuildStatus, startedAt, completedAt *time.Time, errorMessage *string) error

	// ClaimNextQueuedBuild atomically claims the next queued build and marks it running
	ClaimNextQueuedBuild(ctx context.Context) (*Build, error)

	// RequeueBuild returns a build to the queue with a next run time.
	RequeueBuild(ctx context.Context, id uuid.UUID, nextRunAt time.Time, errorMessage *string) error

	// Delete removes a build
	Delete(ctx context.Context, id uuid.UUID) error

	// CountByTenantID counts builds for a tenant
	CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error)

	// CountByStatus counts builds by status for a tenant
	CountByStatus(ctx context.Context, tenantID uuid.UUID, status BuildStatus) (int, error)

	// CountByProjectID counts builds for a project
	CountByProjectID(ctx context.Context, projectID uuid.UUID) (int, error)

	// FindRunningBuilds retrieves all running builds
	FindRunningBuilds(ctx context.Context) ([]*Build, error)

	// SaveBuildConfig persists build configuration
	SaveBuildConfig(ctx context.Context, config *BuildConfigData) error

	// GetBuildConfig retrieves build configuration by build ID
	GetBuildConfig(ctx context.Context, buildID uuid.UUID) (*BuildConfigData, error)

	// UpdateBuildConfig updates existing build configuration
	UpdateBuildConfig(ctx context.Context, config *BuildConfigData) error

	// DeleteBuildConfig deletes build configuration
	DeleteBuildConfig(ctx context.Context, buildID uuid.UUID) error

	// UpdateInfrastructureSelection updates the infrastructure selection for a build
	UpdateInfrastructureSelection(ctx context.Context, build *Build) error
}

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	PublishBuildCreated(ctx context.Context, event *BuildCreated) error
	PublishBuildStarted(ctx context.Context, event *BuildStarted) error
	PublishBuildCompleted(ctx context.Context, event *BuildCompleted) error
	PublishBuildStatusUpdated(ctx context.Context, event *BuildStatusUpdated) error
}

// BuildExecutor defines the interface for executing builds
type BuildExecutor interface {
	Execute(ctx context.Context, build *Build) (*BuildResult, error)
	Cancel(ctx context.Context, buildID uuid.UUID) error
}
