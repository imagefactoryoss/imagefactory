package build

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewWebhookTrigger tests webhook trigger creation
func TestNewWebhookTrigger(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()
	createdBy := uuid.New()

	t.Run("valid webhook trigger", func(t *testing.T) {
		trigger, err := NewWebhookTrigger(
			tenantID, projectID, buildID, createdBy,
			"GitHub Webhook", "Deploy on push",
			"https://webhook.example.com", "secret123",
			[]string{"push", "pull_request"},
		)

		assert.NoError(t, err)
		assert.NotNil(t, trigger)
		assert.Equal(t, TriggerTypeWebhook, trigger.Type)
		assert.Equal(t, "GitHub Webhook", trigger.Name)
		assert.Equal(t, "https://webhook.example.com", trigger.WebhookURL)
		assert.True(t, trigger.IsActive)
	})

	t.Run("missing webhook URL", func(t *testing.T) {
		trigger, err := NewWebhookTrigger(
			tenantID, projectID, buildID, createdBy,
			"Test", "", "", "", nil,
		)

		assert.Error(t, err)
		assert.Nil(t, trigger)
		assert.Contains(t, err.Error(), "webhook URL required")
	})

	t.Run("missing trigger name", func(t *testing.T) {
		trigger, err := NewWebhookTrigger(
			tenantID, projectID, buildID, createdBy,
			"", "", "https://webhook.example.com", "", nil,
		)

		assert.Error(t, err)
		assert.Nil(t, trigger)
		assert.Contains(t, err.Error(), "trigger name is required")
	})

	t.Run("invalid tenant ID", func(t *testing.T) {
		trigger, err := NewWebhookTrigger(
			uuid.Nil, projectID, buildID, createdBy,
			"Test", "", "https://webhook.example.com", "", nil,
		)

		assert.Error(t, err)
		assert.Nil(t, trigger)
		assert.Contains(t, err.Error(), "invalid IDs")
	})

	t.Run("invalid created_by ID", func(t *testing.T) {
		trigger, err := NewWebhookTrigger(
			tenantID, projectID, buildID, uuid.Nil,
			"Test", "", "https://webhook.example.com", "", nil,
		)

		assert.Error(t, err)
		assert.Nil(t, trigger)
		assert.Contains(t, err.Error(), "created_by required")
	})
}

// TestNewScheduledTrigger tests scheduled trigger creation
func TestNewScheduledTrigger(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()
	createdBy := uuid.New()

	t.Run("valid scheduled trigger", func(t *testing.T) {
		trigger, err := NewScheduledTrigger(
			tenantID, projectID, buildID, createdBy,
			"Nightly Build", "Build every night at midnight",
			"0 0 * * *", "America/New_York",
		)

		assert.NoError(t, err)
		assert.NotNil(t, trigger)
		assert.Equal(t, TriggerTypeSchedule, trigger.Type)
		assert.Equal(t, "Nightly Build", trigger.Name)
		assert.Equal(t, "0 0 * * *", trigger.CronExpr)
		assert.Equal(t, "America/New_York", trigger.Timezone)
		assert.True(t, trigger.IsActive)
	})

	t.Run("default timezone", func(t *testing.T) {
		trigger, err := NewScheduledTrigger(
			tenantID, projectID, buildID, createdBy,
			"Test", "", "0 0 * * *", "",
		)

		assert.NoError(t, err)
		assert.Equal(t, "UTC", trigger.Timezone)
	})

	t.Run("missing cron expression", func(t *testing.T) {
		trigger, err := NewScheduledTrigger(
			tenantID, projectID, buildID, createdBy,
			"Test", "", "", "UTC",
		)

		assert.Error(t, err)
		assert.Nil(t, trigger)
		assert.Contains(t, err.Error(), "cron expression required")
	})

	t.Run("missing trigger name", func(t *testing.T) {
		trigger, err := NewScheduledTrigger(
			tenantID, projectID, buildID, createdBy,
			"", "", "0 0 * * *", "UTC",
		)

		assert.Error(t, err)
		assert.Nil(t, trigger)
		assert.Contains(t, err.Error(), "trigger name is required")
	})
}

// TestNewGitEventTrigger tests Git event trigger creation
func TestNewGitEventTrigger(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()
	createdBy := uuid.New()

	t.Run("valid GitHub trigger", func(t *testing.T) {
		trigger, err := NewGitEventTrigger(
			tenantID, projectID, buildID, createdBy,
			"GitHub Push", "Trigger on push to main",
			GitProviderGitHub,
			"https://github.com/owner/repo.git", "main",
		)

		assert.NoError(t, err)
		assert.NotNil(t, trigger)
		assert.Equal(t, TriggerTypeGitEvent, trigger.Type)
		assert.Equal(t, GitProviderGitHub, trigger.GitProvider)
		assert.Equal(t, "main", trigger.GitBranchPattern)
		assert.True(t, trigger.IsActive)
	})

	t.Run("default branch pattern", func(t *testing.T) {
		trigger, err := NewGitEventTrigger(
			tenantID, projectID, buildID, createdBy,
			"Test", "", GitProviderGitHub,
			"https://github.com/owner/repo.git", "",
		)

		assert.NoError(t, err)
		assert.Equal(t, "main", trigger.GitBranchPattern)
	})

	t.Run("missing repository URL", func(t *testing.T) {
		trigger, err := NewGitEventTrigger(
			tenantID, projectID, buildID, createdBy,
			"Test", "", GitProviderGitHub, "", "main",
		)

		assert.Error(t, err)
		assert.Nil(t, trigger)
		assert.Contains(t, err.Error(), "Git repository URL is required")
	})

	t.Run("missing Git provider", func(t *testing.T) {
		trigger, err := NewGitEventTrigger(
			tenantID, projectID, buildID, createdBy,
			"Test", "", "", "https://github.com/owner/repo.git", "main",
		)

		assert.Error(t, err)
		assert.Nil(t, trigger)
		assert.Contains(t, err.Error(), "Git provider required")
	})

	t.Run("all Git providers", func(t *testing.T) {
		providers := []GitProvider{
			GitProviderGitHub,
			GitProviderGitLab,
			GitProviderGitea,
			GitProviderBitbucket,
		}

		for _, provider := range providers {
			trigger, err := NewGitEventTrigger(
				tenantID, projectID, buildID, createdBy,
				"Test", "", provider,
				"https://github.com/owner/repo.git", "main",
			)

			assert.NoError(t, err, "failed for provider: %s", provider)
			assert.Equal(t, provider, trigger.GitProvider)
		}
	})
}

// TestBuildTriggerMutators tests state mutation methods
func TestBuildTriggerMutators(t *testing.T) {
	trigger, _ := NewWebhookTrigger(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		"Test", "", "https://webhook.example.com", "", nil,
	)

	t.Run("deactivate trigger", func(t *testing.T) {
		assert.True(t, trigger.IsActive)
		trigger.Deactivate()
		assert.False(t, trigger.IsActive)
	})

	t.Run("activate trigger", func(t *testing.T) {
		trigger.Activate()
		assert.True(t, trigger.IsActive)
	})

	t.Run("record trigger execution", func(t *testing.T) {
		assert.Nil(t, trigger.LastTriggered)
		assert.Nil(t, trigger.NextTrigger)

		nextTime := time.Now().Add(24 * time.Hour)
		trigger.RecordTrigger(&nextTime)

		assert.NotNil(t, trigger.LastTriggered)
		assert.NotNil(t, trigger.NextTrigger)
		assert.Equal(t, nextTime.Unix(), trigger.NextTrigger.Unix()) // Within 1 second
	})
}

// TestTriggerAccessors tests public accessor functions
func TestTriggerAccessors(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()
	createdBy := uuid.New()

	trigger, err := NewWebhookTrigger(
		tenantID, projectID, buildID, createdBy,
		"Test Trigger", "Test Description",
		"https://webhook.example.com", "secret",
		[]string{"push", "pr"},
	)

	require.NoError(t, err)

	t.Run("field accessors", func(t *testing.T) {
		assert.Equal(t, tenantID, trigger.TenantID)
		assert.Equal(t, projectID, trigger.ProjectID)
		assert.Equal(t, buildID, trigger.BuildID)
		assert.Equal(t, createdBy, trigger.CreatedBy)
		assert.Equal(t, "Test Trigger", trigger.Name)
		assert.Equal(t, "Test Description", trigger.Description)
		assert.Equal(t, TriggerTypeWebhook, trigger.Type)
		assert.Equal(t, "https://webhook.example.com", trigger.WebhookURL)
		assert.Equal(t, "secret", trigger.WebhookSecret)
		assert.Equal(t, []string{"push", "pr"}, trigger.WebhookEvents)
	})

	t.Run("timestamp fields", func(t *testing.T) {
		assert.False(t, trigger.CreatedAt.IsZero())
		assert.False(t, trigger.UpdatedAt.IsZero())
		assert.True(t, trigger.CreatedAt.Before(time.Now().Add(time.Second)))
	})
}
