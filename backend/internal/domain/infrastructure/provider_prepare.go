package infrastructure

import (
	"time"

	"github.com/google/uuid"
)

type ProviderPrepareRunStatus string

const (
	ProviderPrepareRunStatusPending   ProviderPrepareRunStatus = "pending"
	ProviderPrepareRunStatusRunning   ProviderPrepareRunStatus = "running"
	ProviderPrepareRunStatusSucceeded ProviderPrepareRunStatus = "succeeded"
	ProviderPrepareRunStatusFailed    ProviderPrepareRunStatus = "failed"
	ProviderPrepareRunStatusCancelled ProviderPrepareRunStatus = "cancelled"
)

type ProviderPrepareRun struct {
	ID               uuid.UUID                `json:"id" db:"id"`
	ProviderID       uuid.UUID                `json:"provider_id" db:"provider_id"`
	TenantID         uuid.UUID                `json:"tenant_id" db:"tenant_id"`
	RequestedBy      uuid.UUID                `json:"requested_by" db:"requested_by"`
	Status           ProviderPrepareRunStatus `json:"status" db:"status"`
	RequestedActions map[string]interface{}   `json:"requested_actions,omitempty" db:"requested_actions"`
	ResultSummary    map[string]interface{}   `json:"result_summary,omitempty" db:"result_summary"`
	ErrorMessage     *string                  `json:"error_message,omitempty" db:"error_message"`
	StartedAt        *time.Time               `json:"started_at,omitempty" db:"started_at"`
	CompletedAt      *time.Time               `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt        time.Time                `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time                `json:"updated_at" db:"updated_at"`
}

type ProviderPrepareRunCheck struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	RunID     uuid.UUID              `json:"run_id" db:"run_id"`
	CheckKey  string                 `json:"check_key" db:"check_key"`
	Category  string                 `json:"category" db:"category"`
	Severity  string                 `json:"severity" db:"severity"`
	OK        bool                   `json:"ok" db:"ok"`
	Message   string                 `json:"message" db:"message"`
	Details   map[string]interface{} `json:"details,omitempty" db:"details"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
}

type StartProviderPrepareRunRequest struct {
	RequestedActions map[string]interface{} `json:"requested_actions,omitempty"`
}

type ProviderPrepareStatus struct {
	ProviderID string                     `json:"provider_id"`
	ActiveRun  *ProviderPrepareRun        `json:"active_run,omitempty"`
	Checks     []*ProviderPrepareRunCheck `json:"checks,omitempty"`
}

type ProviderPrepareLatestSummary struct {
	ProviderID      uuid.UUID                 `json:"provider_id"`
	RunID           *uuid.UUID                `json:"run_id,omitempty"`
	Status          *ProviderPrepareRunStatus `json:"status,omitempty"`
	UpdatedAt       *time.Time                `json:"updated_at,omitempty"`
	ErrorMessage    *string                   `json:"error_message,omitempty"`
	CheckCategory   *string                   `json:"latest_prepare_check_category,omitempty"`
	CheckSeverity   *string                   `json:"latest_prepare_check_severity,omitempty"`
	RemediationHint *string                   `json:"latest_prepare_remediation_hint,omitempty"`
}
