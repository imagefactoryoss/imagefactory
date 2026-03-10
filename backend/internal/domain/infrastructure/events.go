package infrastructure

import (
	"time"

	"github.com/google/uuid"
)

// ProviderCreated represents a domain event for provider creation
type ProviderCreated struct {
	ProviderID   uuid.UUID `json:"provider_id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	ProviderType string    `json:"provider_type"`
	Name         string    `json:"name"`
	CreatedBy    uuid.UUID `json:"created_by"`
	OccurredAt   time.Time `json:"occurred_at"`
}

// ProviderUpdated represents a domain event for provider updates
type ProviderUpdated struct {
	ProviderID uuid.UUID `json:"provider_id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	UpdatedBy  uuid.UUID `json:"updated_by"`
	OccurredAt time.Time `json:"occurred_at"`
}

// ProviderDeleted represents a domain event for provider deletion
type ProviderDeleted struct {
	ProviderID uuid.UUID `json:"provider_id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	DeletedBy  uuid.UUID `json:"deleted_by"`
	OccurredAt time.Time `json:"occurred_at"`
}