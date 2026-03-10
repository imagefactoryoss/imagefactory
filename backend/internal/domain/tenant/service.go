package tenant

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service defines the business logic for tenant management
type Service struct {
	repository     Repository
	eventPublisher EventPublisher
	logger         *zap.Logger
}

// NewService creates a new tenant service
func NewService(repository Repository, eventPublisher EventPublisher, logger *zap.Logger) *Service {
	return &Service{
		repository:     repository,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// CreateTenant creates a new tenant
func (s *Service) CreateTenant(ctx context.Context, companyID uuid.UUID, tenantCode, name, slug, description string) (*Tenant, error) {
	// Check if tenant already exists
	exists, err := s.repository.ExistsBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrTenantExists
	}

	// Create new tenant
	tenant, err := NewTenant(uuid.New(), companyID, tenantCode, name, slug, description)
	if err != nil {
		return nil, err
	}

	// Save tenant
	if err := s.repository.Save(ctx, tenant); err != nil {
		return nil, err
	}

	// Publish domain event
	event := NewTenantCreated(tenant.ID(), tenant.Name())
	if err := s.eventPublisher.PublishTenantCreated(ctx, event); err != nil {
		// Log error but don't fail the operation
		// In a real implementation, you might want to use an outbox pattern
	}

	return tenant, nil
}

// GetTenant retrieves a tenant by ID
func (s *Service) GetTenant(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	tenant, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}
	return tenant, nil
}

// GetTenantBySlug retrieves a tenant by slug
func (s *Service) GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error) {
	tenant, err := s.repository.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}
	return tenant, nil
}

// ActivateTenant activates a tenant
func (s *Service) ActivateTenant(ctx context.Context, id uuid.UUID) error {
	tenant, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if tenant == nil {
		return ErrTenantNotFound
	}

	if err := tenant.Activate(); err != nil {
		return err
	}

	if err := s.repository.Update(ctx, tenant); err != nil {
		return err
	}

	// Publish domain event
	event := NewTenantActivated(tenant.ID())
	if err := s.eventPublisher.PublishTenantActivated(ctx, event); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

// SuspendTenant suspends a tenant
func (s *Service) SuspendTenant(ctx context.Context, id uuid.UUID) error {
	tenant, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if tenant == nil {
		return ErrTenantNotFound
	}

	if err := tenant.Suspend(); err != nil {
		return err
	}

	return s.repository.Update(ctx, tenant)
}

// DeleteTenant marks a tenant as deleted (soft delete)
// This preserves data for audit trail while excluding the tenant from normal operations
func (s *Service) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	tenant, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if tenant == nil {
		return ErrTenantNotFound
	}

	if err := tenant.Delete(); err != nil {
		return err
	}

	return s.repository.Update(ctx, tenant)
}

// UpdateTenantQuota updates a tenant's resource quota
func (s *Service) UpdateTenantQuota(ctx context.Context, id uuid.UUID, quota ResourceQuota) error {
	tenant, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if tenant == nil {
		return ErrTenantNotFound
	}

	tenant.UpdateQuota(quota)
	return s.repository.Update(ctx, tenant)
}

// UpdateTenant updates tenant properties
func (s *Service) UpdateTenant(ctx context.Context, id uuid.UUID, name, slug, description string, status string, quota *ResourceQuota, config *TenantConfig) (*Tenant, error) {
	tenant, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}

	// Update name if provided
	if name != "" {
		tenant.name = name
	}

	// Update slug if provided
	if slug != "" {
		// Check if slug is already taken by another tenant
		existing, err := s.repository.FindBySlug(ctx, slug)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID() != id {
			return nil, ErrTenantExists
		}
		tenant.slug = slug
	}

	// Update description if provided
	if description != "" {
		tenant.description = description
	}

	// Update status if provided
	if status != "" {
		switch status {
		case "active":
			tenant.Activate()
		case "suspended":
			tenant.Suspend()
		case "deleted":
			tenant.Delete()
		default:
			return nil, errors.New("invalid status")
		}
	}

	// Update quota if provided
	if quota != nil {
		tenant.UpdateQuota(*quota)
	}

	// Update config if provided
	if config != nil {
		tenant.UpdateConfig(*config)
	}

	// Update timestamp
	tenant.updatedAt = time.Now().UTC()
	tenant.version++

	if err := s.repository.Update(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

// TenantFilter represents filtering options for tenant queries
type TenantFilter struct {
	CompanyID *uuid.UUID
	Status    *TenantStatus
	UserID    *uuid.UUID // Filter tenants where user is an owner/administrator
	Limit     int
	Offset    int
}

// ListTenants lists tenants with pagination and filtering
func (s *Service) ListTenants(ctx context.Context, filter TenantFilter) ([]*Tenant, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50 // default limit
	}
	if filter.Limit > 100 {
		filter.Limit = 100 // max limit
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	return s.repository.FindAll(ctx, filter)
}

// ValidateTenantAccess validates if a tenant can perform operations
func (s *Service) ValidateTenantAccess(ctx context.Context, tenantID uuid.UUID) error {
	tenant, err := s.repository.FindByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant == nil {
		return ErrTenantNotFound
	}

	if !tenant.CanPerformBuild() {
		return errors.New("tenant does not have access to perform builds")
	}

	return nil
}

// GetEventPublisher returns the event publisher
func (s *Service) GetEventPublisher() EventPublisher {
	return s.eventPublisher
}

// GetTotalTenantCount returns the total number of tenants in the system
func (s *Service) GetTotalTenantCount(ctx context.Context) (int, error) {
	return s.repository.GetTotalTenantCount(ctx)
}

// GetActiveTenantCount returns the number of active tenants in the system
func (s *Service) GetActiveTenantCount(ctx context.Context) (int, error) {
	return s.repository.GetActiveTenantCount(ctx)
}
