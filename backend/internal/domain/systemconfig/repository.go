package systemconfig

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for system configuration persistence
type Repository interface {
	// Save persists a system configuration
	Save(ctx context.Context, config *SystemConfig) error

	// SaveAll persists multiple system configurations
	SaveAll(ctx context.Context, configs []*SystemConfig) error

	// FindByID retrieves a configuration by ID
	FindByID(ctx context.Context, id uuid.UUID) (*SystemConfig, error)

	// FindByKey retrieves a configuration by tenant and key
	FindByKey(ctx context.Context, tenantID *uuid.UUID, configKey string) (*SystemConfig, error)

	// FindByTypeAndKey retrieves a configuration by tenant, type, and key
	FindByTypeAndKey(ctx context.Context, tenantID *uuid.UUID, configType ConfigType, configKey string) (*SystemConfig, error)

	// FindByType retrieves all configurations of a specific type for a tenant
	FindByType(ctx context.Context, tenantID *uuid.UUID, configType ConfigType) ([]*SystemConfig, error)

	// FindAllByType retrieves all configurations of a specific type from all tenants
	FindAllByType(ctx context.Context, configType ConfigType) ([]*SystemConfig, error)

	// FindUniversalByType retrieves all universal configurations of a specific type (tenant_id IS NULL)
	FindUniversalByType(ctx context.Context, configType ConfigType) ([]*SystemConfig, error)

	// FindByTenantID retrieves all configurations for a tenant
	FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*SystemConfig, error)

	// FindAll retrieves all configurations from all tenants
	FindAll(ctx context.Context) ([]*SystemConfig, error)

	// FindActiveByType retrieves active configurations of a specific type for a tenant
	FindActiveByType(ctx context.Context, tenantID uuid.UUID, configType ConfigType) ([]*SystemConfig, error)

	// Update updates an existing configuration
	Update(ctx context.Context, config *SystemConfig) error

	// Delete removes a configuration
	Delete(ctx context.Context, id uuid.UUID) error

	// ExistsByKey checks if a configuration exists by tenant and key
	ExistsByKey(ctx context.Context, tenantID *uuid.UUID, configKey string) (bool, error)

	// CountByTenantID counts configurations for a tenant
	CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error)

	// CountByType counts configurations of a specific type for a tenant
	CountByType(ctx context.Context, tenantID uuid.UUID, configType ConfigType) (int, error)
}
