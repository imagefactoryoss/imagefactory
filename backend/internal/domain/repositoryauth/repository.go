package repositoryauth

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for repository authentication persistence
type Repository interface {
	// Save persists a repository authentication configuration
	Save(ctx context.Context, auth *RepositoryAuth) error

	// FindByID retrieves a repository authentication by ID
	FindByID(ctx context.Context, id uuid.UUID) (*RepositoryAuth, error)

	// FindByProjectID retrieves all repository authentications for a project
	FindByProjectID(ctx context.Context, projectID uuid.UUID) ([]*RepositoryAuth, error)

	// FindByTenantID retrieves all tenant-scoped repository authentications for a tenant
	FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*RepositoryAuth, error)

	// FindByProjectIDWithTenant retrieves project-scoped auths with optional tenant-scoped fallback
	FindByProjectIDWithTenant(ctx context.Context, projectID uuid.UUID, includeTenant bool) ([]*RepositoryAuth, error)

	// FindActiveByProjectID retrieves the active repository authentication for a project
	FindActiveByProjectID(ctx context.Context, projectID uuid.UUID) (*RepositoryAuth, error)

	// FindByNameAndProjectID retrieves a repository authentication by name and project ID
	FindByNameAndProjectID(ctx context.Context, name string, projectID uuid.UUID) (*RepositoryAuth, error)

	// Update updates an existing repository authentication
	Update(ctx context.Context, auth *RepositoryAuth) error

	// Delete performs soft delete (deactivate)
	Delete(ctx context.Context, id uuid.UUID) error

	// ExistsByNameAndProjectID checks if a repository authentication name exists for a project
	ExistsByNameAndProjectID(ctx context.Context, name string, projectID uuid.UUID) (bool, error)

	// ExistsByNameInScope checks if a repository authentication name exists in tenant/project scope
	ExistsByNameInScope(ctx context.Context, tenantID uuid.UUID, projectID *uuid.UUID, name string) (bool, error)

	// FindSummariesByTenantID retrieves repository authentications for a tenant
	FindSummariesByTenantID(ctx context.Context, tenantID uuid.UUID) ([]RepositoryAuthSummary, error)

	// FindActiveProjectUsages retrieves active projects currently referencing this auth.
	FindActiveProjectUsages(ctx context.Context, authID uuid.UUID) ([]ProjectUsage, error)
}
