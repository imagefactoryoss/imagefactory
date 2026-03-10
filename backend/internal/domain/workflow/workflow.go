package workflow

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidDefinition = errors.New("invalid workflow definition")
	ErrInvalidInstance   = errors.New("invalid workflow instance")
	ErrInvalidStep       = errors.New("invalid workflow step")
)

type InstanceStatus string

const (
	InstanceStatusRunning   InstanceStatus = "running"
	InstanceStatusBlocked   InstanceStatus = "blocked"
	InstanceStatusFailed    InstanceStatus = "failed"
	InstanceStatusCompleted InstanceStatus = "completed"
)

type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusSucceeded StepStatus = "succeeded"
	StepStatusFailed    StepStatus = "failed"
	StepStatusBlocked   StepStatus = "blocked"
)

type Definition struct {
	ID         uuid.UUID              `json:"id"`
	Name       string                 `json:"name"`
	Version    int                    `json:"version"`
	Definition map[string]interface{} `json:"definition"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

type Instance struct {
	ID           uuid.UUID      `json:"id" db:"id"`
	DefinitionID uuid.UUID      `json:"definition_id" db:"definition_id"`
	TenantID     *uuid.UUID     `json:"tenant_id,omitempty" db:"tenant_id"`
	SubjectType  string         `json:"subject_type" db:"subject_type"`
	SubjectID    uuid.UUID      `json:"subject_id" db:"subject_id"`
	Status       InstanceStatus `json:"status" db:"status"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}

type Step struct {
	ID          uuid.UUID              `json:"id"`
	InstanceID  uuid.UUID              `json:"instance_id"`
	StepKey     string                 `json:"step_key"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	Status      StepStatus             `json:"status"`
	Attempts    int                    `json:"attempts"`
	LastError   *string                `json:"last_error,omitempty"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type Event struct {
	ID         uuid.UUID              `json:"id"`
	InstanceID uuid.UUID              `json:"instance_id"`
	StepID     *uuid.UUID             `json:"step_id,omitempty"`
	Type       string                 `json:"type"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

type StepDefinition struct {
	StepKey string                 `json:"step_key"`
	Payload map[string]interface{} `json:"payload,omitempty"`
	Status  StepStatus             `json:"status"`
}

// BlockedStepDiagnostics summarizes workflow steps that are effectively blocked
// (pending with last_error) for a given subject type.
type BlockedStepDiagnostics struct {
	SubjectType      string     `json:"subject_type"`
	BlockedStepCount int        `json:"blocked_step_count"`
	OldestBlockedAt  *time.Time `json:"oldest_blocked_at,omitempty"`
	DispatchBlocked  int        `json:"dispatch_blocked"`
	MonitorBlocked   int        `json:"monitor_blocked"`
	FinalizeBlocked  int        `json:"finalize_blocked"`
}
