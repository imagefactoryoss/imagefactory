package infrastructure

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for infrastructure provider persistence
type Repository interface {
	// Provider operations
	SaveProvider(ctx context.Context, provider *Provider) error
	FindProviderByID(ctx context.Context, id uuid.UUID) (*Provider, error)
	FindProvidersByTenant(ctx context.Context, tenantID uuid.UUID, opts *ListProvidersOptions) (*ListProvidersResult, error)
	FindProvidersAll(ctx context.Context, opts *ListProvidersOptions) (*ListProvidersResult, error)
	UpdateProvider(ctx context.Context, provider *Provider) error
	DeleteProvider(ctx context.Context, id uuid.UUID) error
	ExistsProviderByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error)

	// Permission operations
	SavePermission(ctx context.Context, permission *ProviderPermission) error
	FindPermissionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*ProviderPermission, error)
	FindPermissionsByProvider(ctx context.Context, providerID uuid.UUID) ([]*ProviderPermission, error)
	DeletePermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) error
	HasPermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) (bool, error)

	// Health operations
	UpdateProviderHealth(ctx context.Context, providerID uuid.UUID, health *ProviderHealth) error
	GetProviderHealth(ctx context.Context, providerID uuid.UUID) (*ProviderHealth, error)
	UpdateProviderReadiness(ctx context.Context, providerID uuid.UUID, status string, checkedAt time.Time, missingPrereqs []string) error
}

// EventPublisher defines the interface for publishing infrastructure domain events
type EventPublisher interface {
	PublishProviderCreated(ctx context.Context, event *ProviderCreated) error
	PublishProviderUpdated(ctx context.Context, event *ProviderUpdated) error
	PublishProviderDeleted(ctx context.Context, event *ProviderDeleted) error
}
