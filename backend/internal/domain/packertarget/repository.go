package packertarget

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, profile *Profile) error
	Update(ctx context.Context, profile *Profile) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Profile, error)
	List(ctx context.Context, tenantID uuid.UUID, allTenants bool, provider string) ([]*Profile, error)
	UpdateValidation(ctx context.Context, tenantID, id uuid.UUID, result ValidationResult) error
}
