package infrastructure

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ProviderTenantNamespacePrepareStatus string

const (
	ProviderTenantNamespacePreparePending   ProviderTenantNamespacePrepareStatus = "pending"
	ProviderTenantNamespacePrepareRunning   ProviderTenantNamespacePrepareStatus = "running"
	ProviderTenantNamespacePrepareSucceeded ProviderTenantNamespacePrepareStatus = "succeeded"
	ProviderTenantNamespacePrepareFailed    ProviderTenantNamespacePrepareStatus = "failed"
	ProviderTenantNamespacePrepareCancelled ProviderTenantNamespacePrepareStatus = "cancelled"
)

type TenantAssetDriftStatus string

const (
	TenantAssetDriftStatusCurrent TenantAssetDriftStatus = "current"
	TenantAssetDriftStatusStale   TenantAssetDriftStatus = "stale"
	TenantAssetDriftStatusUnknown TenantAssetDriftStatus = "unknown"
)

// ProviderTenantNamespacePrepare tracks readiness/provisioning for a single provider+tenant namespace pair.
// This is required because Tekton Tasks/Pipelines and runtime RBAC are namespace-scoped.
type ProviderTenantNamespacePrepare struct {
	ID                    uuid.UUID                            `json:"id" db:"id"`
	ProviderID            uuid.UUID                            `json:"provider_id" db:"provider_id"`
	TenantID              uuid.UUID                            `json:"tenant_id" db:"tenant_id"`
	Namespace             string                               `json:"namespace" db:"namespace"`
	RequestedBy           *uuid.UUID                           `json:"requested_by,omitempty" db:"requested_by"`
	Status                ProviderTenantNamespacePrepareStatus `json:"status" db:"status"`
	ResultSummary         map[string]interface{}               `json:"result_summary,omitempty" db:"result_summary"`
	ErrorMessage          *string                              `json:"error_message,omitempty" db:"error_message"`
	DesiredAssetVersion   *string                              `json:"desired_asset_version,omitempty" db:"desired_asset_version"`
	InstalledAssetVersion *string                              `json:"installed_asset_version,omitempty" db:"installed_asset_version"`
	AssetDriftStatus      TenantAssetDriftStatus               `json:"asset_drift_status" db:"asset_drift_status"`
	StartedAt             *time.Time                           `json:"started_at,omitempty" db:"started_at"`
	CompletedAt           *time.Time                           `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt             time.Time                            `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time                            `json:"updated_at" db:"updated_at"`
}

type ProviderTenantNamespacePrepareRepository interface {
	UpsertTenantNamespacePrepare(ctx context.Context, prepare *ProviderTenantNamespacePrepare) error
	GetTenantNamespacePrepare(ctx context.Context, providerID, tenantID uuid.UUID) (*ProviderTenantNamespacePrepare, error)
}
