package sresmartbot

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type IncidentStatus string
type IncidentSeverity string
type IncidentConfidence string

const (
	IncidentStatusObserved   IncidentStatus = "observed"
	IncidentStatusTriaged    IncidentStatus = "triaged"
	IncidentStatusContained  IncidentStatus = "contained"
	IncidentStatusRecovering IncidentStatus = "recovering"
	IncidentStatusResolved   IncidentStatus = "resolved"
	IncidentStatusSuppressed IncidentStatus = "suppressed"
	IncidentStatusEscalated  IncidentStatus = "escalated"
)

const (
	IncidentSeverityInfo     IncidentSeverity = "info"
	IncidentSeverityWarning  IncidentSeverity = "warning"
	IncidentSeverityCritical IncidentSeverity = "critical"
)

const (
	IncidentConfidenceLow    IncidentConfidence = "low"
	IncidentConfidenceMedium IncidentConfidence = "medium"
	IncidentConfidenceHigh   IncidentConfidence = "high"
)

type Incident struct {
	ID              uuid.UUID          `json:"id"`
	TenantID        *uuid.UUID         `json:"tenant_id,omitempty"`
	CorrelationKey  string             `json:"correlation_key"`
	Domain          string             `json:"domain"`
	IncidentType    string             `json:"incident_type"`
	DisplayName     string             `json:"display_name"`
	Summary         string             `json:"summary"`
	Severity        IncidentSeverity   `json:"severity"`
	Confidence      IncidentConfidence `json:"confidence"`
	Status          IncidentStatus     `json:"status"`
	Source          string             `json:"source"`
	FirstObservedAt time.Time          `json:"first_observed_at"`
	LastObservedAt  time.Time          `json:"last_observed_at"`
	ContainedAt     *time.Time         `json:"contained_at,omitempty"`
	ResolvedAt      *time.Time         `json:"resolved_at,omitempty"`
	SuppressedUntil *time.Time         `json:"suppressed_until,omitempty"`
	Metadata        json.RawMessage    `json:"metadata,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type Finding struct {
	ID         uuid.UUID          `json:"id"`
	IncidentID *uuid.UUID         `json:"incident_id,omitempty"`
	Source     string             `json:"source"`
	SignalType string             `json:"signal_type"`
	SignalKey  string             `json:"signal_key"`
	Severity   IncidentSeverity   `json:"severity"`
	Confidence IncidentConfidence `json:"confidence"`
	Title      string             `json:"title"`
	Message    string             `json:"message"`
	RawPayload json.RawMessage    `json:"raw_payload,omitempty"`
	OccurredAt time.Time          `json:"occurred_at"`
	CreatedAt  time.Time          `json:"created_at"`
}

type Evidence struct {
	ID           uuid.UUID       `json:"id"`
	IncidentID   uuid.UUID       `json:"incident_id"`
	EvidenceType string          `json:"evidence_type"`
	Summary      string          `json:"summary"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	CapturedAt   time.Time       `json:"captured_at"`
	CreatedAt    time.Time       `json:"created_at"`
}

type ActionAttempt struct {
	ID               uuid.UUID       `json:"id"`
	IncidentID       uuid.UUID       `json:"incident_id"`
	ActionKey        string          `json:"action_key"`
	ActionClass      string          `json:"action_class"`
	TargetKind       string          `json:"target_kind"`
	TargetRef        string          `json:"target_ref"`
	Status           string          `json:"status"`
	ActorType        string          `json:"actor_type"`
	ActorID          string          `json:"actor_id,omitempty"`
	ApprovalRequired bool            `json:"approval_required"`
	RequestedAt      time.Time       `json:"requested_at"`
	StartedAt        *time.Time      `json:"started_at,omitempty"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	ErrorMessage     string          `json:"error_message,omitempty"`
	ResultPayload    json.RawMessage `json:"result_payload,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type Approval struct {
	ID                uuid.UUID  `json:"id"`
	IncidentID        uuid.UUID  `json:"incident_id"`
	ActionAttemptID   *uuid.UUID `json:"action_attempt_id,omitempty"`
	ChannelProviderID string     `json:"channel_provider_id"`
	Status            string     `json:"status"`
	RequestMessage    string     `json:"request_message"`
	RequestedBy       string     `json:"requested_by,omitempty"`
	DecidedBy         string     `json:"decided_by,omitempty"`
	DecisionComment   string     `json:"decision_comment,omitempty"`
	RequestedAt       time.Time  `json:"requested_at"`
	DecidedAt         *time.Time `json:"decided_at,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type IncidentFilter struct {
	TenantID *uuid.UUID
	Domain   string
	Status   string
	Severity string
	Search   string
	Limit    int
	Offset   int
}

type ApprovalFilter struct {
	TenantID          *uuid.UUID
	Status            string
	ChannelProviderID string
	Search            string
	Limit             int
	Offset            int
}

type RemediationPackRun struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        *uuid.UUID      `json:"tenant_id,omitempty"`
	IncidentID      uuid.UUID       `json:"incident_id"`
	PackKey         string          `json:"pack_key"`
	PackVersion     string          `json:"pack_version"`
	RunKind         string          `json:"run_kind"`
	Status          string          `json:"status"`
	RequestedBy     string          `json:"requested_by,omitempty"`
	RequestID       string          `json:"request_id,omitempty"`
	ApprovalID      *uuid.UUID      `json:"approval_id,omitempty"`
	ActionAttemptID *uuid.UUID      `json:"action_attempt_id,omitempty"`
	Summary         string          `json:"summary,omitempty"`
	ResultPayload   json.RawMessage `json:"result_payload,omitempty"`
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type DetectorRuleSuggestionStatus string

const (
	DetectorRuleSuggestionStatusPending  DetectorRuleSuggestionStatus = "pending"
	DetectorRuleSuggestionStatusAccepted DetectorRuleSuggestionStatus = "accepted"
	DetectorRuleSuggestionStatusRejected DetectorRuleSuggestionStatus = "rejected"
)

type DetectorRuleSuggestion struct {
	ID              uuid.UUID                    `json:"id"`
	TenantID        *uuid.UUID                   `json:"tenant_id,omitempty"`
	IncidentID      *uuid.UUID                   `json:"incident_id,omitempty"`
	Fingerprint     string                       `json:"fingerprint"`
	Name            string                       `json:"name"`
	Description     string                       `json:"description"`
	Query           string                       `json:"query"`
	Threshold       int                          `json:"threshold"`
	Domain          string                       `json:"domain"`
	IncidentType    string                       `json:"incident_type"`
	Severity        IncidentSeverity             `json:"severity"`
	Confidence      IncidentConfidence           `json:"confidence"`
	SignalKey       string                       `json:"signal_key,omitempty"`
	Source          string                       `json:"source"`
	Status          DetectorRuleSuggestionStatus `json:"status"`
	AutoCreated     bool                         `json:"auto_created"`
	Reason          string                       `json:"reason,omitempty"`
	EvidencePayload json.RawMessage              `json:"evidence_payload,omitempty"`
	ProposedBy      string                       `json:"proposed_by,omitempty"`
	ReviewedBy      string                       `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time                   `json:"reviewed_at,omitempty"`
	ActivatedRuleID string                       `json:"activated_rule_id,omitempty"`
	CreatedAt       time.Time                    `json:"created_at"`
	UpdatedAt       time.Time                    `json:"updated_at"`
}

type DetectorRuleSuggestionFilter struct {
	TenantID *uuid.UUID
	Status   string
	Search   string
	Limit    int
	Offset   int
}

type Repository interface {
	CreateIncident(ctx context.Context, incident *Incident) error
	UpdateIncident(ctx context.Context, incident *Incident) error
	GetIncident(ctx context.Context, id uuid.UUID) (*Incident, error)
	GetIncidentByCorrelationKey(ctx context.Context, correlationKey string) (*Incident, error)
	ListIncidents(ctx context.Context, filter IncidentFilter) ([]*Incident, error)
	CountIncidents(ctx context.Context, filter IncidentFilter) (int, error)
	CreateFinding(ctx context.Context, finding *Finding) error
	ListFindingsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*Finding, error)
	AddEvidence(ctx context.Context, evidence *Evidence) error
	ListEvidenceByIncident(ctx context.Context, incidentID uuid.UUID) ([]*Evidence, error)
	CreateActionAttempt(ctx context.Context, attempt *ActionAttempt) error
	UpdateActionAttempt(ctx context.Context, attempt *ActionAttempt) error
	ListActionAttemptsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*ActionAttempt, error)
	CreateApproval(ctx context.Context, approval *Approval) error
	UpdateApproval(ctx context.Context, approval *Approval) error
	ListApprovalsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*Approval, error)
	ListApprovals(ctx context.Context, filter ApprovalFilter) ([]*Approval, error)
	CountApprovals(ctx context.Context, filter ApprovalFilter) (int, error)
	CreateDetectorRuleSuggestion(ctx context.Context, suggestion *DetectorRuleSuggestion) error
	UpdateDetectorRuleSuggestion(ctx context.Context, suggestion *DetectorRuleSuggestion) error
	GetDetectorRuleSuggestion(ctx context.Context, id uuid.UUID) (*DetectorRuleSuggestion, error)
	GetDetectorRuleSuggestionByFingerprint(ctx context.Context, fingerprint string) (*DetectorRuleSuggestion, error)
	ListDetectorRuleSuggestions(ctx context.Context, filter DetectorRuleSuggestionFilter) ([]*DetectorRuleSuggestion, error)
	CreateRemediationPackRun(ctx context.Context, run *RemediationPackRun) error
	ListRemediationPackRunsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*RemediationPackRun, error)
}
