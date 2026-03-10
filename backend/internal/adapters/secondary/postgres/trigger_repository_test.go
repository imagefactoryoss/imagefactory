package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/srikarm/image-factory/internal/domain/build"
)

// TriggerRepositoryTestSuite defines tests for trigger repository
type TriggerRepositoryTestSuite struct {
	suite.Suite
	ctx context.Context
}

func (suite *TriggerRepositoryTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

// TestSaveTrigger tests creating a webhook trigger
func (suite *TriggerRepositoryTestSuite) TestSaveTrigger() {
	// Create trigger
	trigger, err := build.NewWebhookTrigger(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		"Test Webhook", "Test Description",
		"https://webhook.example.com", "secret123",
		[]string{"push", "pull_request"},
	)
	require.NoError(suite.T(), err)

	assert.NotNil(suite.T(), trigger)
	assert.Equal(suite.T(), build.TriggerTypeWebhook, trigger.Type)
	assert.Equal(suite.T(), "Test Webhook", trigger.Name)
	assert.True(suite.T(), trigger.IsActive)
}

// TestTriggerTypes tests creating all three trigger types
func (suite *TriggerRepositoryTestSuite) TestAllTriggerTypes() {
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()
	createdBy := uuid.New()

	t := suite.T()

	t.Run("webhook trigger", func(t *testing.T) {
		webhook, err := build.NewWebhookTrigger(
			tenantID, projectID, buildID, createdBy,
			"Webhook", "", "https://example.com", "", nil,
		)
		assert.NoError(t, err)
		assert.Equal(t, build.TriggerTypeWebhook, webhook.Type)
	})

	t.Run("schedule trigger", func(t *testing.T) {
		schedule, err := build.NewScheduledTrigger(
			tenantID, projectID, buildID, createdBy,
			"Schedule", "", "0 0 * * *", "UTC",
		)
		assert.NoError(t, err)
		assert.Equal(t, build.TriggerTypeSchedule, schedule.Type)
	})

	t.Run("git event trigger", func(t *testing.T) {
		gitEvent, err := build.NewGitEventTrigger(
			tenantID, projectID, buildID, createdBy,
			"GitEvent", "", build.GitProviderGitHub,
			"https://github.com/owner/repo.git", "main",
		)
		assert.NoError(t, err)
		assert.Equal(t, build.TriggerTypeGitEvent, gitEvent.Type)
	})
}

// TestTriggerValidation tests trigger validation
func (suite *TriggerRepositoryTestSuite) TestTriggerValidation() {
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()
	createdBy := uuid.New()

	t := suite.T()

	t.Run("webhook validation", func(t *testing.T) {
		tests := []struct {
			name        string
			webhookURL  string
			expectError bool
		}{
			{"valid URL", "https://example.com", false},
			{"empty URL", "", true},
		}

		for _, test := range tests {
			trigger, err := build.NewWebhookTrigger(
				tenantID, projectID, buildID, createdBy,
				"Test", "", test.webhookURL, "", nil,
			)

			if test.expectError {
				assert.Error(t, err, test.name)
			} else {
				assert.NoError(t, err, test.name)
				assert.NotNil(t, trigger)
			}
		}
	})

	t.Run("schedule validation", func(t *testing.T) {
		tests := []struct {
			name        string
			cronExpr    string
			expectError bool
		}{
			{"valid cron", "0 0 * * *", false},
			{"empty cron", "", true},
		}

		for _, test := range tests {
			trigger, err := build.NewScheduledTrigger(
				tenantID, projectID, buildID, createdBy,
				"Test", "", test.cronExpr, "UTC",
			)

			if test.expectError {
				assert.Error(t, err, test.name)
			} else {
				assert.NoError(t, err, test.name)
				assert.NotNil(t, trigger)
			}
		}
	})

	t.Run("git event validation", func(t *testing.T) {
		tests := []struct {
			name        string
			repoURL     string
			expectError bool
		}{
			{"valid URL", "https://github.com/owner/repo.git", false},
			{"empty URL", "", true},
		}

		for _, test := range tests {
			trigger, err := build.NewGitEventTrigger(
				tenantID, projectID, buildID, createdBy,
				"Test", "", build.GitProviderGitHub, test.repoURL, "main",
			)

			if test.expectError {
				assert.Error(t, err, test.name)
			} else {
				assert.NoError(t, err, test.name)
				assert.NotNil(t, trigger)
			}
		}
	})
}

// TestTriggerStateTransitions tests trigger state management
func (suite *TriggerRepositoryTestSuite) TestTriggerStateTransitions() {
	trigger, _ := build.NewWebhookTrigger(
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		"Test", "", "https://example.com", "", nil,
	)

	t := suite.T()

	t.Run("initial state", func(t *testing.T) {
		assert.True(t, trigger.IsActive)
		assert.Nil(t, trigger.LastTriggered)
		assert.Nil(t, trigger.NextTrigger)
	})

	t.Run("deactivation", func(t *testing.T) {
		trigger.Deactivate()
		assert.False(t, trigger.IsActive)
		updatedTime := trigger.UpdatedAt
		assert.True(t, updatedTime.After(trigger.CreatedAt))
	})

	t.Run("reactivation", func(t *testing.T) {
		trigger.Activate()
		assert.True(t, trigger.IsActive)
	})

	t.Run("record execution", func(t *testing.T) {
		now := time.Now()
		nextTime := now.Add(24 * time.Hour)
		trigger.RecordTrigger(&nextTime)

		assert.NotNil(t, trigger.LastTriggered)
		assert.NotNil(t, trigger.NextTrigger)
		assert.True(t, trigger.LastTriggered.After(now.Add(-time.Second)))
		assert.True(t, trigger.NextTrigger.After(now.Add(23 * time.Hour)))
	})
}

// TestScheduledTriggerTimezone tests timezone handling
func (suite *TriggerRepositoryTestSuite) TestScheduledTriggerTimezone() {
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()
	createdBy := uuid.New()

	t := suite.T()

	timezones := []string{
		"UTC",
		"America/New_York",
		"Europe/London",
		"Asia/Tokyo",
		"Australia/Sydney",
	}

	for _, tz := range timezones {
		trigger, err := build.NewScheduledTrigger(
			tenantID, projectID, buildID, createdBy,
			"Test", "", "0 0 * * *", tz,
		)

		assert.NoError(t, err, "timezone: %s", tz)
		assert.Equal(t, tz, trigger.Timezone)
	}
}

// TestGitProviders tests all Git provider types
func (suite *TriggerRepositoryTestSuite) TestGitProviders() {
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()
	createdBy := uuid.New()

	t := suite.T()

	providers := []build.GitProvider{
		build.GitProviderGitHub,
		build.GitProviderGitLab,
		build.GitProviderGitea,
		build.GitProviderBitbucket,
	}

	for _, provider := range providers {
		trigger, err := build.NewGitEventTrigger(
			tenantID, projectID, buildID, createdBy,
			"Test", "", provider,
			"https://example.com/repo.git", "main",
		)

		assert.NoError(t, err, "provider: %s", provider)
		assert.Equal(t, provider, trigger.GitProvider)
	}
}

// TestTriggerEvents tests webhook events
func (suite *TriggerRepositoryTestSuite) TestWebhookEvents() {
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()
	createdBy := uuid.New()

	t := suite.T()

	eventSets := [][]string{
		{"push"},
		{"push", "pull_request"},
		{"push", "pull_request", "release"},
		{},
	}

	for i, events := range eventSets {
		trigger, err := build.NewWebhookTrigger(
			tenantID, projectID, buildID, createdBy,
			"Test", "", "https://example.com", "", events,
		)

		assert.NoError(t, err, "event set %d", i)
		assert.Equal(t, events, trigger.WebhookEvents)
	}
}

// TestInvalidIDs tests ID validation
func (suite *TriggerRepositoryTestSuite) TestInvalidIDs() {
	projectID := uuid.New()
	buildID := uuid.New()
	createdBy := uuid.New()

	t := suite.T()

	t.Run("nil tenant ID", func(t *testing.T) {
		trigger, err := build.NewWebhookTrigger(
			uuid.Nil, projectID, buildID, createdBy,
			"Test", "", "https://example.com", "", nil,
		)
		assert.Error(t, err)
		assert.Nil(t, trigger)
	})

	t.Run("nil project ID", func(t *testing.T) {
		trigger, err := build.NewWebhookTrigger(
			uuid.New(), uuid.Nil, buildID, createdBy,
			"Test", "", "https://example.com", "", nil,
		)
		assert.Error(t, err)
		assert.Nil(t, trigger)
	})

	t.Run("nil build ID", func(t *testing.T) {
		trigger, err := build.NewWebhookTrigger(
			uuid.New(), projectID, uuid.Nil, createdBy,
			"Test", "", "https://example.com", "", nil,
		)
		assert.Error(t, err)
		assert.Nil(t, trigger)
	})

	t.Run("nil created by", func(t *testing.T) {
		trigger, err := build.NewWebhookTrigger(
			uuid.New(), projectID, buildID, uuid.Nil,
			"Test", "", "https://example.com", "", nil,
		)
		assert.Error(t, err)
		assert.Nil(t, trigger)
	})
}

// TestMissingNames tests name validation
func (suite *TriggerRepositoryTestSuite) TestMissingNames() {
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()
	createdBy := uuid.New()

	t := suite.T()

	t.Run("webhook without name", func(t *testing.T) {
		trigger, err := build.NewWebhookTrigger(
			tenantID, projectID, buildID, createdBy,
			"", "", "https://example.com", "", nil,
		)
		assert.Error(t, err)
		assert.Nil(t, trigger)
	})

	t.Run("schedule without name", func(t *testing.T) {
		trigger, err := build.NewScheduledTrigger(
			tenantID, projectID, buildID, createdBy,
			"", "", "0 0 * * *", "UTC",
		)
		assert.Error(t, err)
		assert.Nil(t, trigger)
	})

	t.Run("git event without name", func(t *testing.T) {
		trigger, err := build.NewGitEventTrigger(
			tenantID, projectID, buildID, createdBy,
			"", "", build.GitProviderGitHub,
			"https://example.com/repo.git", "main",
		)
		assert.Error(t, err)
		assert.Nil(t, trigger)
	})
}

// Run the test suite
func TestTriggerRepository(t *testing.T) {
	suite.Run(t, new(TriggerRepositoryTestSuite))
}
