package registryauth

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Save(ctx context.Context, auth *RegistryAuth) error
	Update(ctx context.Context, auth *RegistryAuth) error
	FindByID(ctx context.Context, id uuid.UUID) (*RegistryAuth, error)
	ListByProjectID(ctx context.Context, projectID uuid.UUID, includeTenant bool) ([]*RegistryAuth, error)
	ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*RegistryAuth, error)
	FindDefaultByProjectID(ctx context.Context, projectID uuid.UUID) (*RegistryAuth, error)
	FindDefaultByTenantID(ctx context.Context, tenantID uuid.UUID) (*RegistryAuth, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsByNameInScope(ctx context.Context, tenantID uuid.UUID, projectID *uuid.UUID, name string) (bool, error)
}
