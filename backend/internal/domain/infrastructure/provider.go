package infrastructure

import (
	"time"

	"github.com/google/uuid"
)

// ProviderType represents the type of infrastructure provider
type ProviderType string

const (
	ProviderTypeKubernetes ProviderType = "kubernetes"
	ProviderTypeAWSEKS     ProviderType = "aws-eks"
	ProviderTypeGCPGKE     ProviderType = "gcp-gke"
	ProviderTypeAzureAKS   ProviderType = "azure-aks"
	ProviderTypeOCIOKE     ProviderType = "oci-oke"
	ProviderTypeVMwareVKS  ProviderType = "vmware-vks"
	ProviderTypeOpenShift  ProviderType = "openshift"
	ProviderTypeRancher    ProviderType = "rancher"
	ProviderTypeBuildNodes ProviderType = "build_nodes"
)

// ProviderStatus represents the status of an infrastructure provider
type ProviderStatus string

const (
	ProviderStatusOnline      ProviderStatus = "online"
	ProviderStatusOffline     ProviderStatus = "offline"
	ProviderStatusMaintenance ProviderStatus = "maintenance"
	ProviderStatusPending     ProviderStatus = "pending"
)

// Provider represents an infrastructure provider
type Provider struct {
	ID                      uuid.UUID              `json:"id" db:"id"`
	TenantID                uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	IsGlobal                bool                   `json:"is_global" db:"is_global"`
	ProviderType            ProviderType           `json:"provider_type" db:"provider_type"`
	Name                    string                 `json:"name" db:"name"`
	DisplayName             string                 `json:"display_name" db:"display_name"`
	Config                  map[string]interface{} `json:"config" db:"config"`
	Status                  ProviderStatus         `json:"status" db:"status"`
	Capabilities            []string               `json:"capabilities" db:"capabilities"`
	CreatedBy               uuid.UUID              `json:"created_by" db:"created_by"`
	CreatedAt               time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time              `json:"updated_at" db:"updated_at"`
	LastHealthCheck         *time.Time             `json:"last_health_check" db:"last_health_check"`
	HealthStatus            *string                `json:"health_status" db:"health_status"`
	ReadinessStatus         *string                `json:"readiness_status,omitempty" db:"readiness_status"`
	ReadinessLastChecked    *time.Time             `json:"readiness_last_checked,omitempty" db:"readiness_last_checked"`
	ReadinessMissingPrereqs []string               `json:"readiness_missing_prereqs,omitempty" db:"readiness_missing_prereqs"`
	BootstrapMode           string                 `json:"bootstrap_mode" db:"bootstrap_mode"`
	CredentialScope         string                 `json:"credential_scope" db:"credential_scope"`
	TargetNamespace         *string                `json:"target_namespace,omitempty" db:"target_namespace"`
	IsSchedulable           bool                   `json:"is_schedulable" db:"is_schedulable"`
	SchedulableReason       *string                `json:"schedulable_reason,omitempty" db:"schedulable_reason"`
	BlockedBy               []string               `json:"blocked_by,omitempty" db:"blocked_by"`
}

// CreateProviderRequest represents a request to create a new provider
type CreateProviderRequest struct {
	ProviderType    ProviderType           `json:"provider_type" validate:"required"`
	Name            string                 `json:"name" validate:"required,min=3,max=100"`
	DisplayName     string                 `json:"display_name" validate:"required,min=3,max=255"`
	Config          map[string]interface{} `json:"config" validate:"required"`
	Capabilities    []string               `json:"capabilities,omitempty"`
	IsGlobal        bool                   `json:"is_global"`
	BootstrapMode   string                 `json:"bootstrap_mode,omitempty"`
	CredentialScope string                 `json:"credential_scope,omitempty"`
	TargetNamespace *string                `json:"target_namespace,omitempty"`
}

// UpdateProviderRequest represents a request to update a provider
type UpdateProviderRequest struct {
	DisplayName     *string                 `json:"display_name,omitempty"`
	Config          *map[string]interface{} `json:"config,omitempty"`
	Capabilities    *[]string               `json:"capabilities,omitempty"`
	Status          *ProviderStatus         `json:"status,omitempty"`
	IsGlobal        *bool                   `json:"is_global,omitempty"`
	BootstrapMode   *string                 `json:"bootstrap_mode,omitempty"`
	CredentialScope *string                 `json:"credential_scope,omitempty"`
	TargetNamespace *string                 `json:"target_namespace,omitempty"`
}

// ProviderPermission represents a permission for a tenant to access a provider
type ProviderPermission struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	ProviderID uuid.UUID  `json:"provider_id" db:"provider_id"`
	TenantID   uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Permission string     `json:"permission" db:"permission"`
	GrantedBy  uuid.UUID  `json:"granted_by" db:"granted_by"`
	GrantedAt  time.Time  `json:"granted_at" db:"granted_at"`
	ExpiresAt  *time.Time `json:"expires_at" db:"expires_at"`
}

// ProviderHealth represents the health status of a provider
type ProviderHealth struct {
	ProviderID uuid.UUID              `json:"provider_id"`
	Status     string                 `json:"status"`
	LastCheck  time.Time              `json:"last_check"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Metrics    *HealthMetrics         `json:"metrics,omitempty"`
}

// HealthMetrics represents health metrics for a provider
type HealthMetrics struct {
	TotalNodes            *int     `json:"total_nodes,omitempty"`
	HealthyNodes          *int     `json:"healthy_nodes,omitempty"`
	TotalCPUCapacity      *float64 `json:"total_cpu_capacity,omitempty"`
	UsedCPUCapacity       *float64 `json:"used_cpu_cores,omitempty"`
	TotalMemoryCapacityGB *float64 `json:"total_memory_capacity_gb,omitempty"`
	UsedMemoryGB          *float64 `json:"used_memory_gb,omitempty"`
}

// TestConnectionResponse represents the response from testing a provider connection
type TestConnectionResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ListProvidersOptions represents options for listing providers
type ListProvidersOptions struct {
	ProviderType *ProviderType   `json:"provider_type,omitempty"`
	Status       *ProviderStatus `json:"status,omitempty"`
	Page         int             `json:"page,omitempty"`
	Limit        int             `json:"limit,omitempty"`
}

// ListProvidersResult represents the result of listing providers
type ListProvidersResult struct {
	Providers  []Provider `json:"providers"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	TotalPages int        `json:"total_pages"`
}
