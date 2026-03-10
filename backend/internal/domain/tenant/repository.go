package tenant

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for tenant persistence
type Repository interface {
	// Save persists a tenant
	Save(ctx context.Context, tenant *Tenant) error

	// FindByID retrieves a tenant by ID
	FindByID(ctx context.Context, id uuid.UUID) (*Tenant, error)

	// FindBySlug retrieves a tenant by slug
	FindBySlug(ctx context.Context, slug string) (*Tenant, error)

	// FindAll retrieves all tenants with optional filtering
	FindAll(ctx context.Context, filter TenantFilter) ([]*Tenant, error) // Update updates an existing tenant
	Update(ctx context.Context, tenant *Tenant) error

	// Delete removes a tenant
	Delete(ctx context.Context, id uuid.UUID) error

	// ExistsBySlug checks if a tenant exists by slug
	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	// GetTotalTenantCount returns the total number of tenants in the system
	GetTotalTenantCount(ctx context.Context) (int, error)

	// GetActiveTenantCount returns the number of active tenants in the system
	GetActiveTenantCount(ctx context.Context) (int, error)
}

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	PublishTenantCreated(ctx context.Context, event *TenantCreated) error
	PublishTenantActivated(ctx context.Context, event *TenantActivated) error
}
