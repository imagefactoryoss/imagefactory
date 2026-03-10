package buildnotification

import (
	"errors"

	"github.com/google/uuid"
)

type TriggerID string

type Channel string

type RecipientPolicy string

type Severity string

type TriggerPreferenceSource string

const (
	ChannelInApp Channel = "in_app"
	ChannelEmail Channel = "email"
)

const (
	RecipientInitiator     RecipientPolicy = "initiator"
	RecipientProjectMember RecipientPolicy = "project_members"
	RecipientTenantAdmins  RecipientPolicy = "tenant_admins"
	RecipientCustomUsers   RecipientPolicy = "custom_users"
)

const (
	SeverityLow    Severity = "low"
	SeverityNormal Severity = "normal"
	SeverityHigh   Severity = "high"
)

const (
	TriggerPreferenceSourceSystem  TriggerPreferenceSource = "system"
	TriggerPreferenceSourceTenant  TriggerPreferenceSource = "tenant"
	TriggerPreferenceSourceProject TriggerPreferenceSource = "project"
)

const (
	TriggerBuildQueued         TriggerID = "BN-001"
	TriggerBuildStarted        TriggerID = "BN-002"
	TriggerBuildCompleted      TriggerID = "BN-003"
	TriggerBuildFailed         TriggerID = "BN-004"
	TriggerBuildCancelled      TriggerID = "BN-005"
	TriggerBuildRetryStarted   TriggerID = "BN-006"
	TriggerBuildRetryFailed    TriggerID = "BN-007"
	TriggerBuildRetrySucceeded TriggerID = "BN-008"
	TriggerBuildRecovered      TriggerID = "BN-009"
	TriggerPreflightBlocked    TriggerID = "BN-010"
)

var AllTriggerIDs = []TriggerID{
	TriggerBuildQueued,
	TriggerBuildStarted,
	TriggerBuildCompleted,
	TriggerBuildFailed,
	TriggerBuildCancelled,
	TriggerBuildRetryStarted,
	TriggerBuildRetryFailed,
	TriggerBuildRetrySucceeded,
	TriggerBuildRecovered,
	TriggerPreflightBlocked,
}

var (
	ErrInvalidProjectID = errors.New("invalid project ID")
	ErrInvalidTenantID  = errors.New("invalid tenant ID")
	ErrInvalidUserID    = errors.New("invalid user ID")
	ErrInvalidTriggerID = errors.New("invalid trigger ID")
)

type ProjectTriggerPreference struct {
	ID                   uuid.UUID               `json:"id"`
	TenantID             uuid.UUID               `json:"tenant_id"`
	ProjectID            uuid.UUID               `json:"project_id"`
	TriggerID            TriggerID               `json:"trigger_id"`
	Source               TriggerPreferenceSource `json:"source,omitempty"`
	Enabled              bool                    `json:"enabled"`
	Channels             []Channel               `json:"channels"`
	RecipientPolicy      RecipientPolicy         `json:"recipient_policy"`
	CustomRecipientUsers []uuid.UUID             `json:"custom_recipient_user_ids,omitempty"`
	SeverityOverride     *Severity               `json:"severity_override,omitempty"`
}

type TenantTriggerPreference struct {
	ID                   uuid.UUID               `json:"id"`
	TenantID             uuid.UUID               `json:"tenant_id"`
	TriggerID            TriggerID               `json:"trigger_id"`
	Source               TriggerPreferenceSource `json:"source,omitempty"`
	Enabled              bool                    `json:"enabled"`
	Channels             []Channel               `json:"channels"`
	RecipientPolicy      RecipientPolicy         `json:"recipient_policy"`
	CustomRecipientUsers []uuid.UUID             `json:"custom_recipient_user_ids,omitempty"`
	SeverityOverride     *Severity               `json:"severity_override,omitempty"`
}

func DefaultTenantTriggerPreference(tenantID uuid.UUID, triggerID TriggerID) TenantTriggerPreference {
	channels := []Channel{ChannelInApp}
	if triggerID == TriggerBuildStarted || triggerID == TriggerBuildCompleted || triggerID == TriggerBuildFailed || triggerID == TriggerBuildCancelled {
		channels = []Channel{ChannelInApp, ChannelEmail}
	}

	return TenantTriggerPreference{
		TenantID:        tenantID,
		TriggerID:       triggerID,
		Enabled:         true,
		Channels:        channels,
		RecipientPolicy: RecipientInitiator,
	}
}

func DefaultProjectTriggerPreference(tenantID, projectID uuid.UUID, triggerID TriggerID) ProjectTriggerPreference {
	channels := []Channel{ChannelInApp}
	if triggerID == TriggerBuildStarted || triggerID == TriggerBuildCompleted || triggerID == TriggerBuildFailed || triggerID == TriggerBuildCancelled {
		channels = []Channel{ChannelInApp, ChannelEmail}
	}

	return ProjectTriggerPreference{
		TenantID:        tenantID,
		ProjectID:       projectID,
		TriggerID:       triggerID,
		Enabled:         true,
		Channels:        channels,
		RecipientPolicy: RecipientInitiator,
	}
}
