package infrastructure

import (
	"time"

	"github.com/google/uuid"
)

type TektonInstallMode string

const (
	TektonInstallModeGitOps                TektonInstallMode = "gitops"
	TektonInstallModeImageFactoryInstaller TektonInstallMode = "image_factory_installer"
)

type TektonInstallerOperation string

const (
	TektonInstallerOperationInstall  TektonInstallerOperation = "install"
	TektonInstallerOperationUpgrade  TektonInstallerOperation = "upgrade"
	TektonInstallerOperationValidate TektonInstallerOperation = "validate"
)

type TektonInstallerJobStatus string

const (
	TektonInstallerJobStatusPending   TektonInstallerJobStatus = "pending"
	TektonInstallerJobStatusRunning   TektonInstallerJobStatus = "running"
	TektonInstallerJobStatusSucceeded TektonInstallerJobStatus = "succeeded"
	TektonInstallerJobStatusFailed    TektonInstallerJobStatus = "failed"
	TektonInstallerJobStatusCancelled TektonInstallerJobStatus = "cancelled"
)

type TektonInstallerJob struct {
	ID           uuid.UUID                `json:"id" db:"id"`
	ProviderID   uuid.UUID                `json:"provider_id" db:"provider_id"`
	TenantID     uuid.UUID                `json:"tenant_id" db:"tenant_id"`
	RequestedBy  uuid.UUID                `json:"requested_by" db:"requested_by"`
	Operation    TektonInstallerOperation `json:"operation" db:"operation"`
	InstallMode  TektonInstallMode        `json:"install_mode" db:"install_mode"`
	AssetVersion string                   `json:"asset_version" db:"asset_version"`
	Status       TektonInstallerJobStatus `json:"status" db:"status"`
	ErrorMessage *string                  `json:"error_message,omitempty" db:"error_message"`
	StartedAt    *time.Time               `json:"started_at,omitempty" db:"started_at"`
	CompletedAt  *time.Time               `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt    time.Time                `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time                `json:"updated_at" db:"updated_at"`
}

type TektonInstallerJobEvent struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	JobID      uuid.UUID              `json:"job_id" db:"job_id"`
	ProviderID uuid.UUID              `json:"provider_id" db:"provider_id"`
	TenantID   uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	EventType  string                 `json:"event_type" db:"event_type"`
	Message    string                 `json:"message" db:"message"`
	Details    map[string]interface{} `json:"details,omitempty" db:"details"`
	CreatedBy  *uuid.UUID             `json:"created_by,omitempty" db:"created_by"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

type StartTektonInstallerJobRequest struct {
	Operation      TektonInstallerOperation `json:"operation"`
	InstallMode    TektonInstallMode        `json:"install_mode,omitempty"`
	AssetVersion   string                   `json:"asset_version,omitempty"`
	IdempotencyKey string                   `json:"idempotency_key,omitempty"`
}

type TektonInstallerStatus struct {
	ProviderID      uuid.UUID                  `json:"provider_id"`
	ActiveJob       *TektonInstallerJob        `json:"active_job,omitempty"`
	RecentJobs      []*TektonInstallerJob      `json:"recent_jobs"`
	ActiveJobEvents []*TektonInstallerJobEvent `json:"active_job_events,omitempty"`
}
