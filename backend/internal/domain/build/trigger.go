package build

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Trigger domain errors
var (
	ErrInvalidTriggerID    = errors.New("invalid trigger ID")
	ErrInvalidTriggerType  = errors.New("invalid trigger type")
	ErrInvalidWebhookURL   = errors.New("webhook URL is required")
	ErrInvalidCronExpr     = errors.New("cron expression is required")
	ErrInvalidGitProvider  = errors.New("Git provider is required")
	ErrTriggerNotFound     = errors.New("trigger not found")
)

// TriggerType represents the type of trigger
type TriggerType string

const (
	TriggerTypeWebhook   TriggerType = "webhook"
	TriggerTypeSchedule  TriggerType = "schedule"
	TriggerTypeGitEvent  TriggerType = "git_event"
)

// GitProvider represents the Git provider platform
type GitProvider string

const (
	GitProviderGitHub    GitProvider = "github"
	GitProviderGitLab    GitProvider = "gitlab"
	GitProviderGitea     GitProvider = "gitea"
	GitProviderBitbucket GitProvider = "bitbucket"
)

// BuildTrigger is the aggregate root for build triggers
type BuildTrigger struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	ProjectID   uuid.UUID
	BuildID     uuid.UUID
	CreatedBy   uuid.UUID

	Type        TriggerType
	Name        string
	Description string

	// Webhook configuration
	WebhookURL    string
	WebhookSecret string
	WebhookEvents []string // Array of event types: "push", "pull_request", "release"

	// Schedule configuration
	CronExpr       string
	Timezone       string
	LastTriggered  *time.Time
	NextTrigger    *time.Time

	// Git event configuration
	GitProvider      GitProvider
	GitRepoURL       string
	GitBranchPattern string

	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewWebhookTrigger creates a new webhook trigger
func NewWebhookTrigger(
	tenantID, projectID, buildID, createdBy uuid.UUID,
	name, description, webhookURL, webhookSecret string,
	events []string,
) (*BuildTrigger, error) {
	if tenantID == uuid.Nil || projectID == uuid.Nil || buildID == uuid.Nil {
		return nil, fmt.Errorf("invalid IDs: %w", ErrInvalidTriggerID)
	}

	if createdBy == uuid.Nil {
		return nil, fmt.Errorf("created_by required: %w", ErrInvalidTriggerID)
	}

	if webhookURL == "" {
		return nil, fmt.Errorf("webhook URL required: %w", ErrInvalidWebhookURL)
	}

	if name == "" {
		return nil, errors.New("trigger name is required")
	}

	return &BuildTrigger{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ProjectID:     projectID,
		BuildID:       buildID,
		CreatedBy:     createdBy,
		Type:          TriggerTypeWebhook,
		Name:          name,
		Description:   description,
		WebhookURL:    webhookURL,
		WebhookSecret: webhookSecret,
		WebhookEvents: events,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

// NewScheduledTrigger creates a new scheduled trigger with cron expression
func NewScheduledTrigger(
	tenantID, projectID, buildID, createdBy uuid.UUID,
	name, description, cronExpr, timezone string,
) (*BuildTrigger, error) {
	if tenantID == uuid.Nil || projectID == uuid.Nil || buildID == uuid.Nil {
		return nil, fmt.Errorf("invalid IDs: %w", ErrInvalidTriggerID)
	}

	if createdBy == uuid.Nil {
		return nil, fmt.Errorf("created_by required: %w", ErrInvalidTriggerID)
	}

	if cronExpr == "" {
		return nil, fmt.Errorf("cron expression required: %w", ErrInvalidCronExpr)
	}

	if name == "" {
		return nil, errors.New("trigger name is required")
	}

	if timezone == "" {
		timezone = "UTC"
	}

	return &BuildTrigger{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ProjectID:   projectID,
		BuildID:     buildID,
		CreatedBy:   createdBy,
		Type:        TriggerTypeSchedule,
		Name:        name,
		Description: description,
		CronExpr:    cronExpr,
		Timezone:    timezone,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// NewGitEventTrigger creates a new Git event trigger
func NewGitEventTrigger(
	tenantID, projectID, buildID, createdBy uuid.UUID,
	name, description string,
	provider GitProvider,
	repoURL, branchPattern string,
) (*BuildTrigger, error) {
	if tenantID == uuid.Nil || projectID == uuid.Nil || buildID == uuid.Nil {
		return nil, fmt.Errorf("invalid IDs: %w", ErrInvalidTriggerID)
	}

	if createdBy == uuid.Nil {
		return nil, fmt.Errorf("created_by required: %w", ErrInvalidTriggerID)
	}

	if repoURL == "" {
		return nil, errors.New("Git repository URL is required")
	}

	if name == "" {
		return nil, errors.New("trigger name is required")
	}

	if provider == "" {
		return nil, fmt.Errorf("Git provider required: %w", ErrInvalidGitProvider)
	}

	if branchPattern == "" {
		branchPattern = "main"
	}

	return &BuildTrigger{
		ID:               uuid.New(),
		TenantID:         tenantID,
		ProjectID:        projectID,
		BuildID:          buildID,
		CreatedBy:        createdBy,
		Type:             TriggerTypeGitEvent,
		Name:             name,
		Description:      description,
		GitProvider:      provider,
		GitRepoURL:       repoURL,
		GitBranchPattern: branchPattern,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}, nil
}


// Mutators
func (t *BuildTrigger) Deactivate() {
	t.IsActive = false
	t.UpdatedAt = time.Now()
}

func (t *BuildTrigger) Activate() {
	t.IsActive = true
	t.UpdatedAt = time.Now()
}

// RecordTrigger marks that a trigger was executed
func (t *BuildTrigger) RecordTrigger(nextTrigger *time.Time) {
	now := time.Now()
	t.LastTriggered = &now
	t.NextTrigger = nextTrigger
	t.UpdatedAt = now
}
