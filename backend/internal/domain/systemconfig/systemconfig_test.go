package systemconfig

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSystemConfig(t *testing.T) {
	tenantID := uuid.New()
	configType := ConfigTypeLDAP
	configKey := "test-ldap-config"
	configValue := LDAPConfig{
		Host:   "ldap.example.com",
		Port:   389,
		BaseDN: "dc=example,dc=com",
	}
	description := "Test LDAP configuration"
	createdBy := uuid.New()

	t.Run("success", func(t *testing.T) {
		config, err := NewSystemConfig(&tenantID, configType, configKey, configValue, description, createdBy)

		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, &tenantID, config.TenantID())
		assert.Equal(t, configType, config.ConfigType())
		assert.Equal(t, configKey, config.ConfigKey())
		assert.Equal(t, description, config.Description())
		assert.Equal(t, ConfigStatusActive, config.Status())
		assert.False(t, config.IsDefault())
		assert.Equal(t, createdBy, config.CreatedBy())
		assert.Equal(t, createdBy, config.UpdatedBy())
		assert.NotEqual(t, uuid.Nil, config.ID())
		assert.True(t, config.CreatedAt().After(time.Now().Add(-time.Second)))
		assert.True(t, config.UpdatedAt().After(time.Now().Add(-time.Second)))
		assert.Equal(t, 1, config.Version())
	})

	t.Run("nil tenant ID", func(t *testing.T) {
		config, err := NewSystemConfig(nil, configType, configKey, configValue, description, createdBy)
		assert.NoError(t, err)
		assert.Nil(t, config.TenantID())
	})

	t.Run("empty config type", func(t *testing.T) {
		_, err := NewSystemConfig(&tenantID, "", configKey, configValue, description, createdBy)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config type is required")
	})

	t.Run("empty config key", func(t *testing.T) {
		_, err := NewSystemConfig(&tenantID, configType, "", configValue, description, createdBy)
		assert.Equal(t, ErrInvalidConfigKey, err)
	})

	t.Run("nil created by", func(t *testing.T) {
		_, err := NewSystemConfig(&tenantID, configType, configKey, configValue, description, uuid.Nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "created by user ID is required")
	})
}

func TestNewSystemConfigFromExisting(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	configType := ConfigTypeSMTP
	configKey := "test-smtp-config"
	status := ConfigStatusTesting
	description := "Test SMTP configuration"
	isDefault := true
	createdBy := uuid.New()
	updatedBy := uuid.New()
	createdAt := time.Now().Add(-time.Hour)
	updatedAt := time.Now()
	version := 2

	t.Run("success", func(t *testing.T) {
		config, err := NewSystemConfigFromExisting(
			id, &tenantID, configType, configKey, json.RawMessage(`{"host":"smtp.example.com","port":587}`),
			status, description, isDefault, createdBy, updatedBy, createdAt, updatedAt, version,
		)

		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, id, config.ID())
		assert.Equal(t, &tenantID, config.TenantID())
		assert.Equal(t, configType, config.ConfigType())
		assert.Equal(t, configKey, config.ConfigKey())
		assert.Equal(t, status, config.Status())
		assert.Equal(t, description, config.Description())
		assert.Equal(t, isDefault, config.IsDefault())
		assert.Equal(t, createdBy, config.CreatedBy())
		assert.Equal(t, updatedBy, config.UpdatedBy())
		assert.Equal(t, createdAt, config.CreatedAt())
		assert.Equal(t, updatedAt, config.UpdatedAt())
		assert.Equal(t, version, config.Version())
	})

	t.Run("nil ID", func(t *testing.T) {
		_, err := NewSystemConfigFromExisting(
			uuid.Nil, &tenantID, configType, configKey, json.RawMessage(`{}`),
			status, description, isDefault, createdBy, updatedBy, createdAt, updatedAt, version,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid config ID")
	})
}

func TestSystemConfig_GetLDAPConfig(t *testing.T) {
	ldapConfig := LDAPConfig{
		Host:   "ldap.example.com",
		Port:   389,
		BaseDN: "dc=example,dc=com",
	}

	tenantID := uuid.New()
	config, err := NewSystemConfig(&tenantID, ConfigTypeLDAP, "test-ldap", ldapConfig, "Test LDAP", uuid.New())
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		result, err := config.GetLDAPConfig()
		require.NoError(t, err)
		assert.Equal(t, ldapConfig.Host, result.Host)
		assert.Equal(t, ldapConfig.Port, result.Port)
		assert.Equal(t, ldapConfig.BaseDN, result.BaseDN)
	})

	t.Run("wrong config type", func(t *testing.T) {
		tenantID := uuid.New()
		smtpConfig, err := NewSystemConfig(&tenantID, ConfigTypeSMTP, "test-smtp", SMTPConfig{}, "Test SMTP", uuid.New())
		require.NoError(t, err)

		_, err = smtpConfig.GetLDAPConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuration is not LDAP type")
	})
}

func TestSystemConfig_GetSMTPConfig(t *testing.T) {
	smtpConfig := SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user@example.com",
		Password: "password",
		From:     "noreply@example.com",
		StartTLS: true,
	}

	tenantID := uuid.New()
	config, err := NewSystemConfig(&tenantID, ConfigTypeSMTP, "test-smtp", smtpConfig, "Test SMTP", uuid.New())
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		result, err := config.GetSMTPConfig()
		require.NoError(t, err)
		assert.Equal(t, smtpConfig.Host, result.Host)
		assert.Equal(t, smtpConfig.Port, result.Port)
		assert.Equal(t, smtpConfig.Username, result.Username)
		assert.Equal(t, smtpConfig.From, result.From)
		assert.Equal(t, smtpConfig.StartTLS, result.StartTLS)
	})
}

func TestSystemConfig_GetGeneralConfig(t *testing.T) {
	enableNATS := true
	generalConfig := GeneralConfig{
		SystemName:          "Image Factory",
		SystemDescription:   "Container Image Build Platform",
		AdminEmail:          "admin@example.com",
		SupportEmail:        "support@example.com",
		TimeZone:            "UTC",
		DateFormat:          "YYYY-MM-DD",
		DefaultLanguage:     "en",
		MessagingEnableNATS: &enableNATS,
	}

	tenantID := uuid.New()
	config, err := NewSystemConfig(&tenantID, ConfigTypeGeneral, "test-general", generalConfig, "Test General", uuid.New())
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		result, err := config.GetGeneralConfig()
		require.NoError(t, err)
		assert.Equal(t, generalConfig.SystemName, result.SystemName)
		assert.Equal(t, generalConfig.AdminEmail, result.AdminEmail)
		assert.Equal(t, generalConfig.TimeZone, result.TimeZone)
		assert.NotNil(t, result.MessagingEnableNATS)
		assert.Equal(t, enableNATS, *result.MessagingEnableNATS)
	})
}

func TestSystemConfig_GetMessagingConfig(t *testing.T) {
	natsRequired := true
	externalOnly := true
	outboxEnabled := false
	outboxRelayInterval := 3
	outboxBatchSize := 25
	outboxLease := 45
	messagingConfig := MessagingConfig{
		EnableNATS:                 true,
		NATSRequired:               &natsRequired,
		ExternalOnly:               &externalOnly,
		OutboxEnabled:              &outboxEnabled,
		OutboxRelayIntervalSeconds: &outboxRelayInterval,
		OutboxRelayBatchSize:       &outboxBatchSize,
		OutboxClaimLeaseSeconds:    &outboxLease,
	}

	tenantID := uuid.New()
	config, err := NewSystemConfig(&tenantID, ConfigTypeMessaging, "test-messaging", messagingConfig, "Test Messaging", uuid.New())
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		result, err := config.GetMessagingConfig()
		require.NoError(t, err)
		assert.Equal(t, messagingConfig.EnableNATS, result.EnableNATS)
		require.NotNil(t, result.NATSRequired)
		assert.Equal(t, natsRequired, *result.NATSRequired)
		require.NotNil(t, result.ExternalOnly)
		assert.Equal(t, externalOnly, *result.ExternalOnly)
		require.NotNil(t, result.OutboxEnabled)
		assert.Equal(t, outboxEnabled, *result.OutboxEnabled)
		require.NotNil(t, result.OutboxRelayIntervalSeconds)
		assert.Equal(t, outboxRelayInterval, *result.OutboxRelayIntervalSeconds)
		require.NotNil(t, result.OutboxRelayBatchSize)
		assert.Equal(t, outboxBatchSize, *result.OutboxRelayBatchSize)
		require.NotNil(t, result.OutboxClaimLeaseSeconds)
		assert.Equal(t, outboxLease, *result.OutboxClaimLeaseSeconds)
	})
}

func TestSystemConfig_GetRuntimeServicesConfig(t *testing.T) {
	workflowEnabled := true
	providerWatcherEnabled := true
	tenantDriftWatcherEnabled := true
	tektonCleanupEnabled := true
	receiptCleanupEnabled := true
	runtimeCfg := RuntimeServicesConfig{
		DispatcherURL:                                      "http://dispatcher.local",
		DispatcherPort:                                     8084,
		WorkflowOrchestratorEnabled:                        &workflowEnabled,
		ProviderReadinessWatcherEnabled:                    &providerWatcherEnabled,
		ProviderReadinessWatcherIntervalSeconds:            180,
		ProviderReadinessWatcherTimeoutSeconds:             90,
		ProviderReadinessWatcherBatchSize:                  200,
		TenantAssetDriftWatcherEnabled:                     &tenantDriftWatcherEnabled,
		TenantAssetDriftWatcherIntervalSeconds:             300,
		TektonHistoryCleanupEnabled:                        &tektonCleanupEnabled,
		TektonHistoryCleanupSchedule:                       "30 2 * * *",
		TektonHistoryCleanupKeepPipelineRuns:               120,
		TektonHistoryCleanupKeepTaskRuns:                   240,
		TektonHistoryCleanupKeepPods:                       240,
		ImageImportNotificationReceiptCleanupEnabled:       &receiptCleanupEnabled,
		ImageImportNotificationReceiptRetentionDays:        45,
		ImageImportNotificationReceiptCleanupIntervalHours: 12,
	}

	tenantID := uuid.New()
	config, err := NewSystemConfig(&tenantID, ConfigTypeRuntimeServices, "runtime_services", runtimeCfg, "Test Runtime Services", uuid.New())
	require.NoError(t, err)

	result, err := config.GetRuntimeServicesConfig()
	require.NoError(t, err)
	require.NotNil(t, result.ImageImportNotificationReceiptCleanupEnabled)
	assert.Equal(t, receiptCleanupEnabled, *result.ImageImportNotificationReceiptCleanupEnabled)
	assert.Equal(t, 45, result.ImageImportNotificationReceiptRetentionDays)
	assert.Equal(t, 12, result.ImageImportNotificationReceiptCleanupIntervalHours)
}

func TestSystemConfig_GetBuildConfig(t *testing.T) {
	monitorEventDrivenEnabled := true
	buildConfig := BuildConfig{
		DefaultTimeoutMinutes:     45,
		MaxConcurrentJobs:         20,
		WorkerPoolSize:            8,
		MaxQueueSize:              200,
		ArtifactRetentionDays:     14,
		TektonEnabled:             true,
		MonitorEventDrivenEnabled: &monitorEventDrivenEnabled,
		EnableTempScanStage:       true,
	}

	tenantID := uuid.New()
	config, err := NewSystemConfig(&tenantID, ConfigTypeBuild, "test-build", buildConfig, "Test Build", uuid.New())
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		result, err := config.GetBuildConfig()
		require.NoError(t, err)
		assert.Equal(t, buildConfig.DefaultTimeoutMinutes, result.DefaultTimeoutMinutes)
		assert.Equal(t, buildConfig.MaxConcurrentJobs, result.MaxConcurrentJobs)
		assert.Equal(t, buildConfig.WorkerPoolSize, result.WorkerPoolSize)
		assert.Equal(t, buildConfig.MaxQueueSize, result.MaxQueueSize)
		assert.Equal(t, buildConfig.ArtifactRetentionDays, result.ArtifactRetentionDays)
		assert.Equal(t, buildConfig.TektonEnabled, result.TektonEnabled)
		require.NotNil(t, result.MonitorEventDrivenEnabled)
		assert.Equal(t, monitorEventDrivenEnabled, *result.MonitorEventDrivenEnabled)
		assert.Equal(t, buildConfig.EnableTempScanStage, result.EnableTempScanStage)
	})
}

func TestSystemConfig_GetExternalServiceConfig(t *testing.T) {
	externalServiceConfig := ExternalServiceConfig{
		Name:        "Test Service",
		Description: "A test external service",
		URL:         "https://api.test.com",
		APIKey:      "test-api-key",
		Enabled:     true,
	}

	tenantID := uuid.New()
	config, err := NewSystemConfig(&tenantID, ConfigTypeExternalServices, "test-external-service", externalServiceConfig, "Test External Service", uuid.New())
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		result, err := config.GetExternalServiceConfig()
		require.NoError(t, err)
		assert.Equal(t, externalServiceConfig.Name, result.Name)
		assert.Equal(t, externalServiceConfig.Description, result.Description)
		assert.Equal(t, externalServiceConfig.URL, result.URL)
		assert.Equal(t, externalServiceConfig.APIKey, result.APIKey)
		assert.Equal(t, externalServiceConfig.Enabled, result.Enabled)
	})

	t.Run("wrong config type", func(t *testing.T) {
		tenantID := uuid.New()
		ldapConfig, err := NewSystemConfig(&tenantID, ConfigTypeLDAP, "test-ldap", LDAPConfig{}, "Test LDAP", uuid.New())
		require.NoError(t, err)

		_, err = ldapConfig.GetExternalServiceConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuration is not external_services type")
	})
}

func TestSystemConfig_UpdateValue(t *testing.T) {
	initialConfig := LDAPConfig{Host: "ldap.example.com", Port: 389}
	tenantID := uuid.New()
	config, err := NewSystemConfig(&tenantID, ConfigTypeLDAP, "test-ldap", initialConfig, "Test LDAP", uuid.New())
	require.NoError(t, err)

	updatedConfig := LDAPConfig{Host: "ldap2.example.com", Port: 636, SSL: true}
	updatedBy := uuid.New()

	t.Run("success", func(t *testing.T) {
		err := config.UpdateValue(updatedConfig, updatedBy)
		require.NoError(t, err)

		assert.Equal(t, updatedBy, config.UpdatedBy())
		assert.True(t, config.UpdatedAt().After(config.CreatedAt()))
		assert.Equal(t, 2, config.Version())

		// Verify the config value was updated
		result, err := config.GetLDAPConfig()
		require.NoError(t, err)
		assert.Equal(t, "ldap2.example.com", result.Host)
		assert.Equal(t, 636, result.Port)
		assert.True(t, result.SSL)
	})

	t.Run("nil updated by", func(t *testing.T) {
		err := config.UpdateValue(updatedConfig, uuid.Nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "updated by user ID is required")
	})
}

func TestSystemConfig_UpdateDescription(t *testing.T) {
	tenantID := uuid.New()
	config, err := NewSystemConfig(&tenantID, ConfigTypeLDAP, "test-ldap", LDAPConfig{}, "Initial description", uuid.New())
	require.NoError(t, err)

	newDescription := "Updated description"
	updatedBy := uuid.New()

	t.Run("success", func(t *testing.T) {
		err := config.UpdateDescription(newDescription, updatedBy)
		require.NoError(t, err)

		assert.Equal(t, newDescription, config.Description())
		assert.Equal(t, updatedBy, config.UpdatedBy())
		assert.True(t, config.UpdatedAt().After(config.CreatedAt()))
		assert.Equal(t, 2, config.Version())
	})
}

func TestSystemConfig_Activate(t *testing.T) {
	tenantID := uuid.New()
	config, err := NewSystemConfig(&tenantID, ConfigTypeLDAP, "test-ldap", LDAPConfig{}, "Test LDAP", uuid.New())
	require.NoError(t, err)

	// First set to inactive
	config.Deactivate(config.CreatedBy())

	updatedBy := uuid.New()

	t.Run("success", func(t *testing.T) {
		err := config.Activate(updatedBy)
		require.NoError(t, err)

		assert.Equal(t, ConfigStatusActive, config.Status())
		assert.True(t, config.IsActive())
		assert.Equal(t, updatedBy, config.UpdatedBy())
		assert.Equal(t, 3, config.Version())
	})
}

func TestSystemConfig_Deactivate(t *testing.T) {
	tenantID := uuid.New()
	config, err := NewSystemConfig(&tenantID, ConfigTypeLDAP, "test-ldap", LDAPConfig{}, "Test LDAP", uuid.New())
	require.NoError(t, err)

	updatedBy := uuid.New()

	t.Run("success", func(t *testing.T) {
		err := config.Deactivate(updatedBy)
		require.NoError(t, err)

		assert.Equal(t, ConfigStatusInactive, config.Status())
		assert.False(t, config.IsActive())
		assert.Equal(t, updatedBy, config.UpdatedBy())
		assert.Equal(t, 2, config.Version())
	})
}

func TestSystemConfig_SetTestingStatus(t *testing.T) {
	tenantID := uuid.New()
	config, err := NewSystemConfig(&tenantID, ConfigTypeLDAP, "test-ldap", LDAPConfig{}, "Test LDAP", uuid.New())
	require.NoError(t, err)

	updatedBy := uuid.New()

	t.Run("success", func(t *testing.T) {
		err := config.SetTestingStatus(updatedBy)
		require.NoError(t, err)

		assert.Equal(t, ConfigStatusTesting, config.Status())
		assert.Equal(t, updatedBy, config.UpdatedBy())
		assert.Equal(t, 2, config.Version())
	})
}
