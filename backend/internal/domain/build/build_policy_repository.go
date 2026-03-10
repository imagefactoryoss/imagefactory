package build

import (
	"context"

	"github.com/google/uuid"
)

// BuildPolicyRepository defines the interface for build policy persistence
type BuildPolicyRepository interface {
	// Save persists a build policy
	Save(ctx context.Context, policy *BuildPolicy) error

	// FindByID retrieves a build policy by ID
	FindByID(ctx context.Context, id uuid.UUID) (*BuildPolicy, error)

	// FindByTenantID retrieves all policies for a tenant
	FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*BuildPolicy, error)
	// FindAll retrieves all policies across tenants
	FindAll(ctx context.Context) ([]*BuildPolicy, error)

	// FindByTenantAndKey retrieves a specific policy by tenant and key
	FindByTenantAndKey(ctx context.Context, tenantID uuid.UUID, policyKey string) (*BuildPolicy, error)

	// FindByType retrieves policies by type for a tenant
	FindByType(ctx context.Context, tenantID uuid.UUID, policyType PolicyType) ([]*BuildPolicy, error)
	// FindAllByType retrieves policies by type across tenants
	FindAllByType(ctx context.Context, policyType PolicyType) ([]*BuildPolicy, error)

	// FindActiveByTenantID retrieves only active policies for a tenant
	FindActiveByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*BuildPolicy, error)
	// FindAllActive retrieves only active policies across tenants
	FindAllActive(ctx context.Context) ([]*BuildPolicy, error)

	// Update updates an existing build policy
	Update(ctx context.Context, policy *BuildPolicy) error

	// Delete removes a build policy
	Delete(ctx context.Context, id uuid.UUID) error

	// ExistsByTenantAndKey checks if a policy exists for the tenant and key
	ExistsByTenantAndKey(ctx context.Context, tenantID uuid.UUID, policyKey string) (bool, error)

	// CountByTenantID counts policies for a tenant
	CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error)
}
