package project

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for project persistence
type Repository interface {
	// Save persists a project
	Save(ctx context.Context, project *Project) error

	// FindByID retrieves a project by ID
	FindByID(ctx context.Context, id uuid.UUID) (*Project, error)

	// FindByTenantID retrieves projects for a tenant, optionally filtering drafts by creator
	FindByTenantID(ctx context.Context, tenantID uuid.UUID, viewerID *uuid.UUID, limit, offset int) ([]*Project, error)

	// FindByNameAndTenantID retrieves a project by name and tenant ID
	FindByNameAndTenantID(ctx context.Context, name string, tenantID uuid.UUID) (*Project, error)

	// Update updates an existing project
	Update(ctx context.Context, project *Project) error

	// Delete performs soft delete
	Delete(ctx context.Context, id uuid.UUID) error

	// PurgeDeletedBefore permanently deletes projects deleted before cutoff
	PurgeDeletedBefore(ctx context.Context, cutoff time.Time) (int, error)

	// CountByTenantID counts projects for a tenant, optionally filtering drafts by creator
	CountByTenantID(ctx context.Context, tenantID uuid.UUID, viewerID *uuid.UUID) (int, error)

	// ExistsByNameAndTenantID checks if a project name exists for a tenant
	ExistsByNameAndTenantID(ctx context.Context, name string, tenantID uuid.UUID) (bool, error)

	// ExistsBySlugAndTenantID checks if a project slug exists for a tenant
	ExistsBySlugAndTenantID(ctx context.Context, slug string, tenantID uuid.UUID) (bool, error)
}

// EventPublisher defines the interface for publishing project domain events
type EventPublisher interface {
	PublishProjectCreated(ctx context.Context, event *ProjectCreated) error
	PublishProjectUpdated(ctx context.Context, event *ProjectUpdated) error
	PublishProjectDeleted(ctx context.Context, event *ProjectDeleted) error
}
