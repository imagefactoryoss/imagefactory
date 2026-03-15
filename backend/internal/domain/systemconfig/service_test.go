package systemconfig

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Save(ctx context.Context, config *SystemConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockRepository) FindByID(ctx context.Context, id uuid.UUID) (*SystemConfig, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SystemConfig), args.Error(1)
}

func (m *MockRepository) FindByKey(ctx context.Context, tenantID *uuid.UUID, configKey string) (*SystemConfig, error) {
	args := m.Called(ctx, tenantID, configKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SystemConfig), args.Error(1)
}

func (m *MockRepository) FindByTypeAndKey(ctx context.Context, tenantID *uuid.UUID, configType ConfigType, configKey string) (*SystemConfig, error) {
	args := m.Called(ctx, tenantID, configType, configKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SystemConfig), args.Error(1)
}

func (m *MockRepository) FindByType(ctx context.Context, tenantID *uuid.UUID, configType ConfigType) ([]*SystemConfig, error) {
	args := m.Called(ctx, tenantID, configType)
	return args.Get(0).([]*SystemConfig), args.Error(1)
}

func (m *MockRepository) FindAllByType(ctx context.Context, configType ConfigType) ([]*SystemConfig, error) {
	args := m.Called(ctx, configType)
	return args.Get(0).([]*SystemConfig), args.Error(1)
}

func (m *MockRepository) FindUniversalByType(ctx context.Context, configType ConfigType) ([]*SystemConfig, error) {
	args := m.Called(ctx, configType)
	return args.Get(0).([]*SystemConfig), args.Error(1)
}

func (m *MockRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*SystemConfig, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*SystemConfig), args.Error(1)
}

func (m *MockRepository) FindAll(ctx context.Context) ([]*SystemConfig, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*SystemConfig), args.Error(1)
}

func (m *MockRepository) FindActiveByType(ctx context.Context, tenantID uuid.UUID, configType ConfigType) ([]*SystemConfig, error) {
	args := m.Called(ctx, tenantID, configType)
	return args.Get(0).([]*SystemConfig), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, config *SystemConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ExistsByKey(ctx context.Context, tenantID *uuid.UUID, configKey string) (bool, error) {
	args := m.Called(ctx, tenantID, configKey)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) CountByType(ctx context.Context, tenantID uuid.UUID, configType ConfigType) (int, error) {
	args := m.Called(ctx, tenantID, configType)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) SaveAll(ctx context.Context, configs []*SystemConfig) error {
	args := m.Called(ctx, configs)
	return args.Error(0)
}

func TestService_CreateConfig(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	configType := ConfigTypeLDAP
	configKey := "test-ldap-config"
	configValue := LDAPConfig{Host: "ldap.example.com", Port: 389}
	description := "Test LDAP configuration"
	createdBy := uuid.New()

	req := CreateConfigRequest{
		TenantID:    &tenantID,
		ConfigType:  configType,
		ConfigKey:   configKey,
		ConfigValue: configValue,
		Description: description,
		CreatedBy:   createdBy,
	}

	t.Run("success", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("ExistsByKey", ctx, &tenantID, configKey).Return(false, nil).Once()
		mockRepo.On("Save", ctx, mock.AnythingOfType("*systemconfig.SystemConfig")).Return(nil).Once()

		// Execute
		result, err := service.CreateConfig(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, &tenantID, result.TenantID())
		assert.Equal(t, configType, result.ConfigType())
		assert.Equal(t, configKey, result.ConfigKey())
		assert.Equal(t, description, result.Description())

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})

	t.Run("config already exists", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("ExistsByKey", ctx, &tenantID, configKey).Return(true, nil).Once()

		// Execute
		result, err := service.CreateConfig(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrConfigAlreadyExists, err)
		assert.Nil(t, result)

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid tenant ID", func(t *testing.T) {
		invalidReq := req
		invalidReq.TenantID = nil

		result, err := service.CreateConfig(ctx, invalidReq)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "tenant ID is required")
	})

	t.Run("empty config key", func(t *testing.T) {
		invalidReq := req
		invalidReq.ConfigKey = ""

		result, err := service.CreateConfig(ctx, invalidReq)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrInvalidConfigKey, err)
	})

	t.Run("runtime services cleanup retention out of bounds", func(t *testing.T) {
		invalidReq := req
		invalidReq.ConfigType = ConfigTypeRuntimeServices
		invalidReq.ConfigKey = "runtime_services"
		invalidReq.ConfigValue = map[string]interface{}{
			"image_import_notification_receipt_retention_days": 0,
		}

		result, err := service.CreateConfig(ctx, invalidReq)

		assert.Error(t, err)
		assert.Nil(t, result)
		var validationErr *ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "must be between 1 and 3650", validationErr.FieldErrors["image_import_notification_receipt_retention_days"])
	})
}

func TestService_CreateOrUpdateCategoryConfig_RuntimeServicesValidation(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()

	_, err := service.CreateOrUpdateCategoryConfig(
		ctx,
		&tenantID,
		ConfigTypeRuntimeServices,
		"runtime_services",
		map[string]interface{}{
			"image_import_notification_receipt_cleanup_interval_hours": 999,
		},
		uuid.New(),
	)

	assert.Error(t, err)
	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "must be between 1 and 168", validationErr.FieldErrors["image_import_notification_receipt_cleanup_interval_hours"])
	mockRepo.AssertNotCalled(t, "ExistsByKey", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestService_CreateOrUpdateCategoryConfig_RuntimeServicesProviderWatcherValidation(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()

	_, err := service.CreateOrUpdateCategoryConfig(
		ctx,
		&tenantID,
		ConfigTypeRuntimeServices,
		"runtime_services",
		map[string]interface{}{
			"provider_readiness_watcher_interval_seconds": 20,
			"provider_readiness_watcher_timeout_seconds":  20,
			"provider_readiness_watcher_batch_size":       1001,
		},
		uuid.New(),
	)

	assert.Error(t, err)
	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "must be at least 30", validationErr.FieldErrors["provider_readiness_watcher_interval_seconds"])
	assert.Equal(t, "must be less than provider_readiness_watcher_interval_seconds", validationErr.FieldErrors["provider_readiness_watcher_timeout_seconds"])
	assert.Equal(t, "must be between 1 and 1000", validationErr.FieldErrors["provider_readiness_watcher_batch_size"])
	mockRepo.AssertNotCalled(t, "ExistsByKey", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestService_CreateOrUpdateCategoryConfig_RuntimeServicesTektonCleanupValidation(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()

	_, err := service.CreateOrUpdateCategoryConfig(
		ctx,
		&tenantID,
		ConfigTypeRuntimeServices,
		"runtime_services",
		map[string]interface{}{
			"tekton_history_cleanup_keep_pipelineruns": 0,
			"tekton_history_cleanup_keep_taskruns":     0,
			"tekton_history_cleanup_keep_pods":         0,
		},
		uuid.New(),
	)

	assert.Error(t, err)
	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "must be at least 1", validationErr.FieldErrors["tekton_history_cleanup_keep_pipelineruns"])
	assert.Equal(t, "must be at least 1", validationErr.FieldErrors["tekton_history_cleanup_keep_taskruns"])
	assert.Equal(t, "must be at least 1", validationErr.FieldErrors["tekton_history_cleanup_keep_pods"])
	mockRepo.AssertNotCalled(t, "ExistsByKey", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestService_CreateOrUpdateCategoryConfig_RuntimeServicesPortAndHealthValidation(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()

	_, err := service.CreateOrUpdateCategoryConfig(
		ctx,
		&tenantID,
		ConfigTypeRuntimeServices,
		"runtime_services",
		map[string]interface{}{
			"dispatcher_port":              0,
			"email_worker_port":            70000,
			"notification_worker_port":     0,
			"health_check_timeout_seconds": 0,
		},
		uuid.New(),
	)

	assert.Error(t, err)
	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "must be between 1 and 65535", validationErr.FieldErrors["dispatcher_port"])
	assert.Equal(t, "must be between 1 and 65535", validationErr.FieldErrors["email_worker_port"])
	assert.Equal(t, "must be between 1 and 65535", validationErr.FieldErrors["notification_worker_port"])
	assert.Equal(t, "must be at least 1", validationErr.FieldErrors["health_check_timeout_seconds"])
	mockRepo.AssertNotCalled(t, "ExistsByKey", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestService_CreateOrUpdateCategoryConfig_RuntimeServicesTektonCleanupScheduleValidation(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()

	_, err := service.CreateOrUpdateCategoryConfig(
		ctx,
		&tenantID,
		ConfigTypeRuntimeServices,
		"runtime_services",
		map[string]interface{}{
			"tekton_history_cleanup_schedule": "bad cron expr!",
		},
		uuid.New(),
	)

	assert.Error(t, err)
	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "must contain exactly 5 cron fields", validationErr.FieldErrors["tekton_history_cleanup_schedule"])
	mockRepo.AssertNotCalled(t, "ExistsByKey", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestService_CreateOrUpdateCategoryConfig_RuntimeServicesURLValidation(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()

	_, err := service.CreateOrUpdateCategoryConfig(
		ctx,
		&tenantID,
		ConfigTypeRuntimeServices,
		"runtime_services",
		map[string]interface{}{
			"dispatcher_url":          "://bad",
			"email_worker_url":        "ftp://localhost",
			"notification_worker_url": "",
		},
		uuid.New(),
	)

	assert.Error(t, err)
	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "must be a valid absolute URL", validationErr.FieldErrors["dispatcher_url"])
	assert.Equal(t, "must use http or https scheme", validationErr.FieldErrors["email_worker_url"])
	assert.Equal(t, "must be a non-empty absolute URL", validationErr.FieldErrors["notification_worker_url"])
	mockRepo.AssertNotCalled(t, "ExistsByKey", mock.Anything, mock.Anything, mock.Anything)
	mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestService_GetConfig(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	configID := uuid.New()
	tenantID := uuid.New()
	expectedConfig, _ := NewSystemConfig(&tenantID, ConfigTypeLDAP, "test-config", LDAPConfig{}, "Test", uuid.New())

	t.Run("success", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("FindByID", ctx, configID).Return(expectedConfig, nil).Once()

		// Execute
		result, err := service.GetConfig(ctx, configID)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expectedConfig, result)

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})

	t.Run("config not found", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("FindByID", ctx, configID).Return(nil, ErrConfigNotFound).Once()

		// Execute
		result, err := service.GetConfig(ctx, configID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrConfigNotFound, err)
		assert.Nil(t, result)

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid config ID", func(t *testing.T) {
		result, err := service.GetConfig(ctx, uuid.Nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "config ID is required")
	})
}

func TestService_GetConfigByKey(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	configKey := "test-config"
	expectedConfig, _ := NewSystemConfig(&tenantID, ConfigTypeSMTP, configKey, SMTPConfig{}, "Test", uuid.New())

	t.Run("success", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("FindByKey", ctx, &tenantID, configKey).Return(expectedConfig, nil).Once()

		// Execute
		result, err := service.GetConfigByKey(ctx, &tenantID, configKey)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expectedConfig, result)

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})

	t.Run("nil tenant ID", func(t *testing.T) {
		// Setup mock expectations for universal config
		var nilTenantID *uuid.UUID
		mockRepo.On("FindByKey", ctx, nilTenantID, configKey).Return(expectedConfig, nil).Once()

		// Execute
		result, err := service.GetConfigByKey(ctx, nil, configKey)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expectedConfig, result)

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty config key", func(t *testing.T) {
		result, err := service.GetConfigByKey(ctx, &tenantID, "")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrInvalidConfigKey, err)
	})
}

func TestService_GetConfigsByType(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	configType := ConfigTypeLDAP

	config1, _ := NewSystemConfig(&tenantID, configType, "config1", LDAPConfig{}, "Test 1", uuid.New())
	config2, _ := NewSystemConfig(&tenantID, configType, "config2", LDAPConfig{}, "Test 2", uuid.New())
	expectedConfigs := []*SystemConfig{config1, config2}

	t.Run("success", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("FindByType", ctx, &tenantID, configType).Return(expectedConfigs, nil).Once()
		mockRepo.On("FindUniversalByType", ctx, configType).Return([]*SystemConfig{}, nil).Once()

		// Execute
		result, err := service.GetConfigsByType(ctx, &tenantID, configType)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expectedConfigs, result)

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})

	t.Run("universal configs", func(t *testing.T) {
		// Setup mock expectations for universal configs
		mockRepo.On("FindUniversalByType", ctx, configType).Return(expectedConfigs, nil).Once()

		// Execute with nil tenantID (universal configs)
		result, err := service.GetConfigsByType(ctx, nil, configType)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expectedConfigs, result)

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})
}

func TestService_GetSORRegistrationConfig_DefaultWhenMissing(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	mockRepo.On("FindByKey", ctx, &tenantID, "sor_registration").Return(nil, ErrConfigNotFound).Once()
	mockRepo.On("FindByKey", ctx, (*uuid.UUID)(nil), "sor_registration").Return(nil, ErrConfigNotFound).Once()

	cfg, err := service.GetSORRegistrationConfig(ctx, &tenantID)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.True(t, cfg.Enforce)
	assert.Equal(t, "error", cfg.RuntimeErrorMode)
	mockRepo.AssertExpectations(t)
}

func TestService_UpdateSORRegistrationConfig_InvalidMode(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	cfg := &SORRegistrationConfig{
		Enforce:          true,
		RuntimeErrorMode: "invalid",
	}

	updated, err := service.UpdateSORRegistrationConfig(context.Background(), nil, cfg, uuid.New())
	require.Error(t, err)
	assert.Nil(t, updated)
}

func TestService_UpdateSORRegistrationConfig_NormalizesMode(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	cfg := &SORRegistrationConfig{
		Enforce:          true,
		RuntimeErrorMode: " DeNy ",
	}
	updatedBy := uuid.New()

	mockRepo.On("ExistsByKey", ctx, (*uuid.UUID)(nil), "sor_registration").Return(false, nil).Once()
	mockRepo.On("Save", ctx, mock.AnythingOfType("*systemconfig.SystemConfig")).Return(nil).Once()

	updated, err := service.UpdateSORRegistrationConfig(ctx, nil, cfg, updatedBy)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "deny", updated.RuntimeErrorMode)
	mockRepo.AssertExpectations(t)
}

func TestService_UpdateConfig(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	configID := uuid.New()
	tenantID := uuid.New()
	existingConfig, _ := NewSystemConfig(&tenantID, ConfigTypeLDAP, "test-config", LDAPConfig{Host: "old.example.com"}, "Old config", uuid.New())

	updatedValue := LDAPConfig{Host: "new.example.com", Port: 636}
	newDescription := "Updated configuration"
	updatedBy := uuid.New()

	req := UpdateConfigRequest{
		ID:          configID,
		ConfigValue: &updatedValue,
		Description: &newDescription,
		UpdatedBy:   updatedBy,
	}

	t.Run("success", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("FindByID", ctx, configID).Return(existingConfig, nil).Once()
		mockRepo.On("Update", ctx, mock.AnythingOfType("*systemconfig.SystemConfig")).Return(nil).Once()

		// Execute
		result, err := service.UpdateConfig(ctx, req)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, updatedBy, result.UpdatedBy())
		assert.Equal(t, 3, result.Version())

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})

	t.Run("config not found", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("FindByID", ctx, configID).Return(nil, ErrConfigNotFound).Once()

		// Execute
		result, err := service.UpdateConfig(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrConfigNotFound, err)
		assert.Nil(t, result)

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid config ID", func(t *testing.T) {
		invalidReq := req
		invalidReq.ID = uuid.Nil

		result, err := service.UpdateConfig(ctx, invalidReq)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "config ID is required")
	})

	t.Run("invalid updated by", func(t *testing.T) {
		invalidReq := req
		invalidReq.UpdatedBy = uuid.Nil

		result, err := service.UpdateConfig(ctx, invalidReq)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "updated by user ID is required")
	})

	t.Run("runtime services cleanup bounds validation", func(t *testing.T) {
		runtimeConfig, _ := NewSystemConfig(&tenantID, ConfigTypeRuntimeServices, "runtime_services", RuntimeServicesConfig{}, "Runtime config", uuid.New())
		invalidValue := map[string]interface{}{
			"image_import_notification_receipt_cleanup_interval_hours": 0,
		}
		invalidReq := UpdateConfigRequest{
			ID:          configID,
			ConfigValue: invalidValue,
			UpdatedBy:   updatedBy,
		}
		mockRepo.On("FindByID", ctx, configID).Return(runtimeConfig, nil).Once()

		result, err := service.UpdateConfig(ctx, invalidReq)

		assert.Error(t, err)
		assert.Nil(t, result)
		var validationErr *ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "must be between 1 and 168", validationErr.FieldErrors["image_import_notification_receipt_cleanup_interval_hours"])
	})
}

func TestService_DeleteConfig(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	configID := uuid.New()
	tenantID := uuid.New()
	existingConfig, _ := NewSystemConfig(&tenantID, ConfigTypeLDAP, "test-config", LDAPConfig{}, "Test", uuid.New())

	t.Run("success", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("FindByID", ctx, configID).Return(existingConfig, nil).Once()
		mockRepo.On("Delete", ctx, configID).Return(nil).Once()

		// Execute
		err := service.DeleteConfig(ctx, configID)

		// Assert
		assert.NoError(t, err)

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})

	t.Run("config not found", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("FindByID", ctx, configID).Return(nil, ErrConfigNotFound).Once()

		// Execute
		err := service.DeleteConfig(ctx, configID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrConfigNotFound, err)

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})
}

func TestService_ActivateConfig(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	configID := uuid.New()
	tenantID := uuid.New()
	existingConfig, _ := NewSystemConfig(&tenantID, ConfigTypeLDAP, "test-config", LDAPConfig{}, "Test", uuid.New())
	updatedBy := uuid.New()

	t.Run("success", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("FindByID", ctx, configID).Return(existingConfig, nil).Once()
		mockRepo.On("Update", ctx, mock.AnythingOfType("*systemconfig.SystemConfig")).Return(nil).Once()

		// Execute
		result, err := service.ActivateConfig(ctx, configID, updatedBy)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, ConfigStatusActive, result.Status())
		assert.True(t, result.IsActive())

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})
}

func TestService_DeactivateConfig(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	configID := uuid.New()
	tenantID := uuid.New()
	existingConfig, _ := NewSystemConfig(&tenantID, ConfigTypeLDAP, "test-config", LDAPConfig{}, "Test", uuid.New())
	updatedBy := uuid.New()

	t.Run("success", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("FindByID", ctx, configID).Return(existingConfig, nil).Once()
		mockRepo.On("Update", ctx, mock.AnythingOfType("*systemconfig.SystemConfig")).Return(nil).Once()

		// Execute
		result, err := service.DeactivateConfig(ctx, configID, updatedBy)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, ConfigStatusInactive, result.Status())
		assert.False(t, result.IsActive())

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})
}

func TestService_TestLDAPConnection(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	configKey := "test-ldap-config"
	ldapConfig, _ := NewSystemConfig(&tenantID, ConfigTypeLDAP, configKey, LDAPConfig{Host: "ldap.example.com", Port: 389}, "Test LDAP", uuid.New())

	t.Run("success", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("FindByKey", ctx, &tenantID, configKey).Return(ldapConfig, nil).Once()
		mockRepo.On("Update", ctx, mock.AnythingOfType("*systemconfig.SystemConfig")).Return(nil).Once()

		// Execute
		err := service.TestLDAPConnection(ctx, &tenantID, configKey)

		// Assert
		assert.NoError(t, err)

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong config type", func(t *testing.T) {
		smtpConfig, _ := NewSystemConfig(&tenantID, ConfigTypeSMTP, configKey, SMTPConfig{}, "Test SMTP", uuid.New())

		// Setup mock expectations
		mockRepo.On("FindByKey", ctx, &tenantID, configKey).Return(smtpConfig, nil).Once()

		// Execute
		err := service.TestLDAPConnection(ctx, &tenantID, configKey)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuration is not LDAP type")

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid LDAP config", func(t *testing.T) {
		invalidLDAPConfig, _ := NewSystemConfig(&tenantID, ConfigTypeLDAP, configKey, LDAPConfig{Host: "", Port: 0}, "Invalid LDAP", uuid.New())

		// Setup mock expectations
		mockRepo.On("FindByKey", ctx, &tenantID, configKey).Return(invalidLDAPConfig, nil).Once()

		// Execute
		err := service.TestLDAPConnection(ctx, &tenantID, configKey)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "LDAP host is required")

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})
}

func TestService_TestSMTPConnection(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	configKey := "test-smtp-config"
	smtpConfig, _ := NewSystemConfig(&tenantID, ConfigTypeSMTP, configKey, SMTPConfig{Host: "smtp.example.com", Port: 587}, "Test SMTP", uuid.New())

	t.Run("success", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("FindByKey", ctx, &tenantID, configKey).Return(smtpConfig, nil).Once()
		mockRepo.On("Update", ctx, mock.AnythingOfType("*systemconfig.SystemConfig")).Return(nil).Once()

		// Execute
		err := service.TestSMTPConnection(ctx, &tenantID, configKey)

		// Assert
		assert.NoError(t, err)

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong config type", func(t *testing.T) {
		ldapConfig, _ := NewSystemConfig(&tenantID, ConfigTypeLDAP, configKey, LDAPConfig{}, "Test LDAP", uuid.New())

		// Setup mock expectations
		mockRepo.On("FindByKey", ctx, &tenantID, configKey).Return(ldapConfig, nil).Once()

		// Execute
		err := service.TestSMTPConnection(ctx, &tenantID, configKey)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuration is not SMTP type")

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})
}

func TestService_TestExternalServiceConnection(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	configKey := "test-external-service-config"
	externalServiceConfig, _ := NewSystemConfig(&tenantID, ConfigTypeExternalServices, configKey, ExternalServiceConfig{
		Name:        "Test Service",
		Description: "Test external service",
		URL:         "https://httpbin.org/status/200",
		APIKey:      "test-api-key",
		Enabled:     true,
	}, "Test External Service", uuid.New())

	t.Run("success", func(t *testing.T) {
		// Setup mock expectations
		mockRepo.On("FindByKey", ctx, &tenantID, configKey).Return(externalServiceConfig, nil).Once()
		mockRepo.On("Update", ctx, mock.AnythingOfType("*systemconfig.SystemConfig")).Return(nil).Once()

		// Execute
		err := service.TestExternalServiceConnection(ctx, &tenantID, configKey)

		// Assert
		assert.NoError(t, err)

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong config type", func(t *testing.T) {
		ldapConfig, _ := NewSystemConfig(&tenantID, ConfigTypeLDAP, configKey, LDAPConfig{}, "Test LDAP", uuid.New())

		// Setup mock expectations
		mockRepo.On("FindByKey", ctx, &tenantID, configKey).Return(ldapConfig, nil).Once()

		// Execute
		err := service.TestExternalServiceConnection(ctx, &tenantID, configKey)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuration is not external service type")

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid external service config", func(t *testing.T) {
		invalidExternalServiceConfig, _ := NewSystemConfig(&tenantID, ConfigTypeExternalServices, configKey, ExternalServiceConfig{
			Name:        "",
			Description: "",
			URL:         "",
			APIKey:      "",
			Enabled:     false,
		}, "Invalid External Service", uuid.New())

		// Setup mock expectations
		mockRepo.On("FindByKey", ctx, &tenantID, configKey).Return(invalidExternalServiceConfig, nil).Once()

		// Execute
		err := service.TestExternalServiceConnection(ctx, &tenantID, configKey)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "external service URL is required")

		// Verify mock expectations
		mockRepo.AssertExpectations(t)
	})
}

func TestService_GetToolAvailabilityConfig_TenantSpecificIsolation(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantA := uuid.New()
	tenantB := uuid.New()

	tenantAConfigValue := ToolAvailabilityConfig{
		BuildMethods:   BuildMethodAvailability{Container: false, Kaniko: true, Buildx: false, Packer: false, Paketo: false, Nix: false},
		SBOMTools:      SBOMToolAvailability{Syft: true, Grype: false, Trivy: false},
		ScanTools:      ScanToolAvailability{Trivy: true, Clair: false, Grype: false, Snyk: false},
		RegistryTypes:  RegistryTypeAvailability{S3: true, Harbor: false, Quay: false, Artifactory: false},
		SecretManagers: SecretManagerAvailability{Vault: true, AWSSM: false, AzureKV: false, GCP: false},
	}
	tenantAConfig, err := NewSystemConfig(&tenantA, ConfigTypeToolSettings, "tool_availability", tenantAConfigValue, "Tenant A tools", uuid.New())
	require.NoError(t, err)

	tenantBConfigValue := ToolAvailabilityConfig{
		BuildMethods:   BuildMethodAvailability{Container: true, Kaniko: false, Buildx: true, Packer: false, Paketo: false, Nix: false},
		SBOMTools:      SBOMToolAvailability{Syft: true, Grype: false, Trivy: false},
		ScanTools:      ScanToolAvailability{Trivy: true, Clair: false, Grype: false, Snyk: false},
		RegistryTypes:  RegistryTypeAvailability{S3: true, Harbor: false, Quay: false, Artifactory: false},
		SecretManagers: SecretManagerAvailability{Vault: true, AWSSM: false, AzureKV: false, GCP: false},
	}
	tenantBConfig, err := NewSystemConfig(&tenantB, ConfigTypeToolSettings, "tool_availability", tenantBConfigValue, "Tenant B tools", uuid.New())
	require.NoError(t, err)

	mockRepo.On("FindByKey", ctx, &tenantA, "tool_availability").Return(tenantAConfig, nil).Once()
	mockRepo.On("FindByKey", ctx, &tenantB, "tool_availability").Return(tenantBConfig, nil).Once()

	gotA, err := service.GetToolAvailabilityConfig(ctx, &tenantA)
	require.NoError(t, err)
	require.NotNil(t, gotA)
	assert.True(t, gotA.BuildMethods.Kaniko)
	assert.False(t, gotA.BuildMethods.Buildx)

	gotB, err := service.GetToolAvailabilityConfig(ctx, &tenantB)
	require.NoError(t, err)
	require.NotNil(t, gotB)
	assert.False(t, gotB.BuildMethods.Kaniko)
	assert.True(t, gotB.BuildMethods.Buildx)

	mockRepo.AssertExpectations(t)
}

func TestService_GetToolAvailabilityConfig_TenantFallbackToGlobal(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	globalConfigValue := ToolAvailabilityConfig{
		BuildMethods:   BuildMethodAvailability{Container: true, Kaniko: true, Buildx: true, Packer: true, Paketo: true, Nix: true},
		SBOMTools:      SBOMToolAvailability{Syft: true, Grype: true, Trivy: true},
		ScanTools:      ScanToolAvailability{Trivy: true, Clair: true, Grype: true, Snyk: true},
		RegistryTypes:  RegistryTypeAvailability{S3: true, Harbor: true, Quay: true, Artifactory: true},
		SecretManagers: SecretManagerAvailability{Vault: true, AWSSM: true, AzureKV: true, GCP: true},
	}
	globalConfig, err := NewSystemConfig(nil, ConfigTypeToolSettings, "tool_availability", globalConfigValue, "Global tools", uuid.New())
	require.NoError(t, err)

	var nilTenant *uuid.UUID
	mockRepo.On("FindByKey", ctx, &tenantID, "tool_availability").Return(nil, ErrConfigNotFound).Once()
	mockRepo.On("FindByKey", ctx, nilTenant, "tool_availability").Return(globalConfig, nil).Once()

	got, err := service.GetToolAvailabilityConfig(ctx, &tenantID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.BuildMethods.Container)
	assert.True(t, got.BuildMethods.Kaniko)
	assert.True(t, got.BuildMethods.Nix)

	mockRepo.AssertExpectations(t)
}

func TestService_GetToolAvailabilityConfig_TenantMissingBuildMethodKeys_StrictFalse(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()

	tenantConfigValue := map[string]interface{}{
		"build_methods": map[string]interface{}{
			"kaniko": true,
		},
		"sbom_tools": map[string]interface{}{
			"syft": true, "grype": true, "trivy": true,
		},
		"scan_tools": map[string]interface{}{
			"trivy": true, "clair": true, "grype": true, "snyk": true,
		},
		"registry_types": map[string]interface{}{
			"s3": true, "harbor": true, "quay": true, "artifactory": true,
		},
		"secret_managers": map[string]interface{}{
			"vault": true, "aws_secretsmanager": true, "azure_keyvault": true, "gcp_secretmanager": true,
		},
	}
	tenantConfig, err := NewSystemConfig(&tenantID, ConfigTypeToolSettings, "tool_availability", tenantConfigValue, "Tenant partial tools", uuid.New())
	require.NoError(t, err)

	mockRepo.On("FindByKey", ctx, &tenantID, "tool_availability").Return(tenantConfig, nil).Once()

	got, err := service.GetToolAvailabilityConfig(ctx, &tenantID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.BuildMethods.Kaniko)
	assert.False(t, got.BuildMethods.Container)
	assert.False(t, got.BuildMethods.Nix)
	assert.False(t, got.BuildMethods.Packer)
	assert.False(t, got.BuildMethods.Paketo)
	assert.False(t, got.BuildMethods.Buildx)
}

func TestService_GetToolAvailabilityConfig_GlobalMissingContainerNix_DefaultsTrue(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()

	globalConfigValue := map[string]interface{}{
		"build_methods": map[string]interface{}{
			"kaniko": true,
		},
		"sbom_tools": map[string]interface{}{
			"syft": true, "grype": true, "trivy": true,
		},
		"scan_tools": map[string]interface{}{
			"trivy": true, "clair": true, "grype": true, "snyk": true,
		},
		"registry_types": map[string]interface{}{
			"s3": true, "harbor": true, "quay": true, "artifactory": true,
		},
		"secret_managers": map[string]interface{}{
			"vault": true, "aws_secretsmanager": true, "azure_keyvault": true, "gcp_secretmanager": true,
		},
	}
	globalConfig, err := NewSystemConfig(nil, ConfigTypeToolSettings, "tool_availability", globalConfigValue, "Global partial tools", uuid.New())
	require.NoError(t, err)

	mockRepo.On("FindByKey", ctx, (*uuid.UUID)(nil), "tool_availability").Return(globalConfig, nil).Once()

	got, err := service.GetToolAvailabilityConfig(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.BuildMethods.Kaniko)
	assert.True(t, got.BuildMethods.Container)
	assert.True(t, got.BuildMethods.Nix)
}

func TestService_GetBuildCapabilitiesConfig_TenantMissingKeysStrictFalse(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()

	tenantConfigValue := map[string]interface{}{
		"gpu": true,
	}
	tenantConfig, err := NewSystemConfig(&tenantID, ConfigTypeToolSettings, "build_capabilities", tenantConfigValue, "Tenant build capabilities", uuid.New())
	require.NoError(t, err)

	mockRepo.On("FindByKey", ctx, &tenantID, "build_capabilities").Return(tenantConfig, nil).Once()

	got, err := service.GetBuildCapabilitiesConfig(ctx, &tenantID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.GPU)
	assert.False(t, got.Privileged)
	assert.False(t, got.MultiArch)
	assert.False(t, got.HighMemory)
	assert.False(t, got.HostNetworking)
	assert.False(t, got.Premium)
}

func TestService_GetBuildCapabilitiesConfig_GlobalMissingKeysDefaultTrue(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()

	globalConfigValue := map[string]interface{}{
		"gpu": true,
	}
	globalConfig, err := NewSystemConfig(nil, ConfigTypeToolSettings, "build_capabilities", globalConfigValue, "Global build capabilities", uuid.New())
	require.NoError(t, err)

	mockRepo.On("FindByKey", ctx, (*uuid.UUID)(nil), "build_capabilities").Return(globalConfig, nil).Once()

	got, err := service.GetBuildCapabilitiesConfig(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.GPU)
	assert.True(t, got.Privileged)
	assert.True(t, got.MultiArch)
	assert.True(t, got.HighMemory)
	assert.True(t, got.HostNetworking)
	assert.True(t, got.Premium)
}

func TestService_GetOperationCapabilitiesConfig_TenantMissingKeysStrictFalse(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()

	tenantConfigValue := map[string]interface{}{
		"quarantine_request": true,
	}
	tenantConfig, err := NewSystemConfig(&tenantID, ConfigTypeToolSettings, "operation_capabilities", tenantConfigValue, "Tenant operation capabilities", uuid.New())
	require.NoError(t, err)

	mockRepo.On("FindByKey", ctx, &tenantID, "operation_capabilities").Return(tenantConfig, nil).Once()

	got, err := service.GetOperationCapabilitiesConfig(ctx, &tenantID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.False(t, got.Build)
	assert.True(t, got.QuarantineRequest)
	assert.False(t, got.QuarantineRelease)
	assert.False(t, got.OnDemandImageScan)
}

func TestService_GetOperationCapabilitiesConfig_GlobalMissingKeysDefaultFalse(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()

	globalConfigValue := map[string]interface{}{
		"build": true,
	}
	globalConfig, err := NewSystemConfig(nil, ConfigTypeToolSettings, "operation_capabilities", globalConfigValue, "Global operation capabilities", uuid.New())
	require.NoError(t, err)

	mockRepo.On("FindByKey", ctx, (*uuid.UUID)(nil), "operation_capabilities").Return(globalConfig, nil).Once()

	got, err := service.GetOperationCapabilitiesConfig(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.Build)
	assert.False(t, got.QuarantineRequest)
	assert.False(t, got.QuarantineRelease)
	assert.False(t, got.OnDemandImageScan)
}

func TestService_GetCapabilitySurfaces_IncludesAuthManagementForQuarantineRequest(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()

	tenantConfigValue := map[string]interface{}{
		"build":              false,
		"quarantine_request": true,
	}
	tenantConfig, err := NewSystemConfig(&tenantID, ConfigTypeToolSettings, "operation_capabilities", tenantConfigValue, "Tenant operation capabilities", uuid.New())
	require.NoError(t, err)

	mockRepo.On("FindByKey", ctx, &tenantID, "operation_capabilities").Return(tenantConfig, nil).Once()

	got, err := service.GetCapabilitySurfaces(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Contains(t, got.Surfaces.NavKeys, "auth_management")
	assert.Contains(t, got.Surfaces.NavKeys, "quarantine_requests")
	assert.Contains(t, got.Surfaces.RouteKeys, "settings.auth")
	assert.NotContains(t, got.Surfaces.NavKeys, "builds")
}

func TestService_GetCapabilitySurfaces_DeniesAuthManagementWithoutBuildOrQuarantineRequest(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()

	tenantConfigValue := map[string]interface{}{
		"build":              false,
		"quarantine_request": false,
	}
	tenantConfig, err := NewSystemConfig(&tenantID, ConfigTypeToolSettings, "operation_capabilities", tenantConfigValue, "Tenant operation capabilities", uuid.New())
	require.NoError(t, err)

	mockRepo.On("FindByKey", ctx, &tenantID, "operation_capabilities").Return(tenantConfig, nil).Once()

	got, err := service.GetCapabilitySurfaces(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.NotContains(t, got.Surfaces.NavKeys, "auth_management")
	denial, ok := got.Denials["settings.auth"]
	require.True(t, ok)
	assert.Equal(t, "tenant_capability_not_entitled", denial.ReasonCode)
}

func TestService_GetQuarantinePolicyConfig_DefaultWhenMissing(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()

	mockRepo.On("FindByKey", ctx, (*uuid.UUID)(nil), "quarantine_policy").Return((*SystemConfig)(nil), ErrConfigNotFound).Once()

	got, err := service.GetQuarantinePolicyConfig(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "dry_run", got.Mode)
	assert.True(t, got.Enabled)
	assert.Equal(t, 0, got.MaxCritical)
	assert.Equal(t, 0, got.MaxP2)
	assert.Equal(t, 0, got.MaxP3)
	assert.Equal(t, 0.0, got.MaxCVSS)
	assert.Equal(t, []string{"critical"}, got.SeverityMapping.P1)
}

func TestService_UpdateQuarantinePolicyConfig_RejectsInvalidMode(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	updatedBy := uuid.New()
	policy := &QuarantinePolicyConfig{
		Enabled:     true,
		Mode:        "strict",
		MaxCritical: 0,
		MaxP2:       1,
		MaxP3:       2,
		MaxCVSS:     7.5,
		SeverityMapping: QuarantinePolicySeverityMapping{
			P1: []string{"critical"},
			P2: []string{"high"},
			P3: []string{"medium"},
			P4: []string{"low", "unknown"},
		},
	}

	got, err := service.UpdateQuarantinePolicyConfig(ctx, &tenantID, policy, updatedBy)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "mode")
}

func TestService_UpdateQuarantinePolicyConfig_RejectsDuplicateSeverityMapping(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	updatedBy := uuid.New()
	policy := &QuarantinePolicyConfig{
		Enabled:     true,
		Mode:        "enforce",
		MaxCritical: 0,
		MaxP2:       0,
		MaxP3:       0,
		MaxCVSS:     9.0,
		SeverityMapping: QuarantinePolicySeverityMapping{
			P1: []string{"critical"},
			P2: []string{"critical"},
			P3: []string{"medium"},
			P4: []string{"low", "unknown"},
		},
	}

	got, err := service.UpdateQuarantinePolicyConfig(ctx, &tenantID, policy, updatedBy)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "duplicates severity")
}

func TestService_UpdateQuarantinePolicyConfig_SavesValidatedPolicy(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	updatedBy := uuid.New()
	policy := &QuarantinePolicyConfig{
		Enabled:     true,
		Mode:        "enforce",
		MaxCritical: 0,
		MaxP2:       1,
		MaxP3:       3,
		MaxCVSS:     7.2,
		SeverityMapping: QuarantinePolicySeverityMapping{
			P1: []string{"critical"},
			P2: []string{"high"},
			P3: []string{"medium"},
			P4: []string{"low", "unknown"},
		},
	}

	mockRepo.On("ExistsByKey", ctx, &tenantID, "quarantine_policy").Return(false, nil).Once()
	mockRepo.On("Save", ctx, mock.AnythingOfType("*systemconfig.SystemConfig")).Return(nil).Once()

	got, err := service.UpdateQuarantinePolicyConfig(ctx, &tenantID, policy, updatedBy)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "enforce", got.Mode)
	assert.Equal(t, 7.2, got.MaxCVSS)
}

func TestService_ValidateQuarantinePolicy(t *testing.T) {
	logger := zap.NewNop()
	service := NewService(&MockRepository{}, logger)

	validPolicy := &QuarantinePolicyConfig{
		Enabled:     true,
		Mode:        "enforce",
		MaxCritical: 0,
		MaxP2:       1,
		MaxP3:       3,
		MaxCVSS:     7.2,
		SeverityMapping: QuarantinePolicySeverityMapping{
			P1: []string{"critical"},
			P2: []string{"high"},
			P3: []string{"medium"},
			P4: []string{"low", "unknown"},
		},
	}
	result := service.ValidateQuarantinePolicy(validPolicy)
	require.NotNil(t, result)
	assert.True(t, result.Valid)
	assert.Len(t, result.Errors, 0)

	invalidPolicy := &QuarantinePolicyConfig{
		Mode:        "bad",
		MaxCritical: -1,
	}
	result = service.ValidateQuarantinePolicy(invalidPolicy)
	require.NotNil(t, result)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestService_GetReleaseGovernancePolicyConfig_DefaultWhenMissing(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	mockRepo.On("FindByKey", ctx, (*uuid.UUID)(nil), "release_governance_policy").Return((*SystemConfig)(nil), ErrConfigNotFound).Once()

	got, err := service.GetReleaseGovernancePolicyConfig(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.Enabled)
	assert.Equal(t, 0.2, got.FailureRatioThreshold)
	assert.Equal(t, 3, got.ConsecutiveFailuresThreshold)
	assert.Equal(t, 10, got.MinimumSamples)
	assert.Equal(t, 60, got.WindowMinutes)
}

func TestService_UpdateReleaseGovernancePolicyConfig_RejectsInvalidThresholds(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	updatedBy := uuid.New()

	cfg := &ReleaseGovernancePolicyConfig{
		Enabled:                      true,
		FailureRatioThreshold:        1.5,
		ConsecutiveFailuresThreshold: 0,
		MinimumSamples:               0,
		WindowMinutes:                0,
	}

	got, err := service.UpdateReleaseGovernancePolicyConfig(ctx, &tenantID, cfg, updatedBy)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "failure_ratio_threshold")
}

func TestService_UpdateReleaseGovernancePolicyConfig_SavesValidatedConfig(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	updatedBy := uuid.New()
	cfg := &ReleaseGovernancePolicyConfig{
		Enabled:                      true,
		FailureRatioThreshold:        0.3,
		ConsecutiveFailuresThreshold: 5,
		MinimumSamples:               20,
		WindowMinutes:                120,
	}

	mockRepo.On("ExistsByKey", ctx, &tenantID, "release_governance_policy").Return(false, nil).Once()
	mockRepo.On("Save", ctx, mock.AnythingOfType("*systemconfig.SystemConfig")).Return(nil).Once()

	got, err := service.UpdateReleaseGovernancePolicyConfig(ctx, &tenantID, cfg, updatedBy)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 0.3, got.FailureRatioThreshold)
	assert.Equal(t, 5, got.ConsecutiveFailuresThreshold)
	assert.Equal(t, 20, got.MinimumSamples)
	assert.Equal(t, 120, got.WindowMinutes)
}

func TestService_GetRobotSREPolicyConfig_DefaultWhenMissing(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	mockRepo.On("FindByKey", ctx, (*uuid.UUID)(nil), "robot_sre_policy").Return((*SystemConfig)(nil), ErrConfigNotFound).Once()

	got, err := service.GetRobotSREPolicyConfig(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.Enabled)
	assert.Equal(t, "SRE Smart Bot", got.DisplayName)
	assert.Equal(t, "demo", got.EnvironmentMode)
	assert.Equal(t, "in_app", got.DefaultChannel)
	assert.Equal(t, "in-app-default", got.DefaultChannelProviderID)
	assert.True(t, got.AutoContainEnabled)
	assert.False(t, got.AutoRecoverEnabled)
	assert.Contains(t, got.EnabledDomains, "infrastructure")
	require.Len(t, got.ChannelProviders, 1)
	assert.Equal(t, "in_app", got.ChannelProviders[0].Kind)
	require.Len(t, got.MCPServers, 2)
	assert.Equal(t, "observability", got.MCPServers[0].Kind)
	assert.Equal(t, "release", got.MCPServers[1].Kind)
	assert.False(t, got.AgentRuntime.Enabled)
	assert.Equal(t, "ollama", got.AgentRuntime.Provider)
	assert.Equal(t, "llama3.2:3b", got.AgentRuntime.Model)
	assert.Equal(t, "http://127.0.0.1:11434", got.AgentRuntime.BaseURL)
	assert.Equal(t, "sre_smart_bot_default", got.AgentRuntime.SystemPromptRef)
}

func TestService_UpdateRobotSREPolicyConfig_RejectsInvalidEnvironmentMode(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	updatedBy := uuid.New()
	cfg := &RobotSREPolicyConfig{
		DisplayName:                      "SRE Smart Bot",
		Enabled:                          true,
		EnvironmentMode:                  "wild-west",
		DefaultChannel:                   "telegram",
		DefaultChannelProviderID:         "ops-telegram",
		AutoObserveEnabled:               true,
		AutoNotifyEnabled:                true,
		AutoContainEnabled:               true,
		AutoRecoverEnabled:               false,
		RequireApprovalForRecover:        true,
		RequireApprovalForDisruptive:     true,
		DuplicateAlertSuppressionSeconds: 900,
		ActionCooldownSeconds:            900,
		EnabledDomains:                   []string{"infrastructure"},
		ChannelProviders: []RobotSREChannelProvider{
			{ID: "ops-telegram", Name: "Ops Telegram", Kind: "telegram", Enabled: true},
		},
	}

	got, err := service.UpdateRobotSREPolicyConfig(ctx, &tenantID, cfg, updatedBy)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "environment_mode")
}

func TestService_GetRobotSREPolicyConfig_DefaultsAgentRuntimeFromEnv(t *testing.T) {
	t.Setenv("IF_SRE_AGENT_RUNTIME_BASE_URL", "http://image-factory-ollama.image-factory.svc.cluster.local:11434")
	t.Setenv("IF_SRE_AGENT_RUNTIME_MODEL", "llama3.2:3b")

	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	mockRepo.On("FindByKey", ctx, (*uuid.UUID)(nil), "robot_sre_policy").Return((*SystemConfig)(nil), ErrConfigNotFound).Once()

	got, err := service.GetRobotSREPolicyConfig(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "http://image-factory-ollama.image-factory.svc.cluster.local:11434", got.AgentRuntime.BaseURL)
	assert.Equal(t, "llama3.2:3b", got.AgentRuntime.Model)
}

func TestService_UpdateRobotSREPolicyConfig_SavesValidatedConfig(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	updatedBy := uuid.New()
	cfg := &RobotSREPolicyConfig{
		DisplayName:                      "SRE Smart Bot",
		Enabled:                          true,
		EnvironmentMode:                  "demo",
		DefaultChannel:                   "custom",
		DefaultChannelProviderID:         "ops-webhook",
		AutoObserveEnabled:               true,
		AutoNotifyEnabled:                true,
		AutoContainEnabled:               true,
		AutoRecoverEnabled:               false,
		RequireApprovalForRecover:        true,
		RequireApprovalForDisruptive:     true,
		DuplicateAlertSuppressionSeconds: 900,
		ActionCooldownSeconds:            900,
		EnabledDomains:                   []string{"infrastructure", "runtime_services", "application_services"},
		ChannelProviders: []RobotSREChannelProvider{
			{
				ID:                          "ops-webhook",
				Name:                        "Enterprise Incident Gateway",
				Kind:                        "webhook",
				Enabled:                     true,
				SupportsInteractiveApproval: true,
				ConfigRef:                   "external_service_robot_sre_gateway",
			},
		},
		MCPServers: []RobotSREMCPServer{
			{
				ID:           "ops-observability",
				Name:         "Observability Gateway",
				Kind:         "observability",
				Enabled:      true,
				Transport:    "http",
				Endpoint:     "https://ops.example.com/mcp/observability",
				AllowedTools: []string{"incidents.list", "evidence.list"},
				ReadOnly:     true,
			},
		},
		AgentRuntime: RobotSREAgentRuntimeConfig{
			Enabled:                            true,
			Provider:                           "openai",
			Model:                              "gpt-5.1-mini",
			SystemPromptRef:                    "sre_smart_bot_enterprise",
			OperatorSummaryEnabled:             true,
			HypothesisRankingEnabled:           true,
			DraftActionPlansEnabled:            true,
			ConversationalApprovalSupport:      true,
			MaxToolCallsPerTurn:                8,
			MaxIncidentsPerSummary:             6,
			RequireHumanConfirmationForMessage: true,
		},
		OperatorRules: []RobotSREOperatorRule{
			{
				ID:           "node-disk-pressure-demo",
				Name:         "Node disk pressure containment",
				Domain:       "infrastructure",
				IncidentType: "node_disk_pressure",
				Severity:     "critical",
				Enabled:      true,
				Source:       "operator_defined",
				Threshold:    3,
			},
		},
	}

	mockRepo.On("ExistsByKey", ctx, &tenantID, "robot_sre_policy").Return(false, nil).Once()
	mockRepo.On("Save", ctx, mock.AnythingOfType("*systemconfig.SystemConfig")).Return(nil).Once()

	got, err := service.UpdateRobotSREPolicyConfig(ctx, &tenantID, cfg, updatedBy)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "custom", got.DefaultChannel)
	assert.Equal(t, "ops-webhook", got.DefaultChannelProviderID)
	require.Len(t, got.ChannelProviders, 1)
	assert.Equal(t, "webhook", got.ChannelProviders[0].Kind)
	require.Len(t, got.MCPServers, 1)
	assert.Equal(t, "http", got.MCPServers[0].Transport)
	assert.Equal(t, "openai", got.AgentRuntime.Provider)
	assert.Equal(t, "gpt-5.1-mini", got.AgentRuntime.Model)
	require.Len(t, got.OperatorRules, 1)
	assert.Equal(t, "node-disk-pressure-demo", got.OperatorRules[0].ID)
}

func TestService_UpdateRobotSREPolicyConfig_RejectsEnabledAgentRuntimeWithoutModel(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	updatedBy := uuid.New()
	cfg := &RobotSREPolicyConfig{
		DisplayName:                      "SRE Smart Bot",
		Enabled:                          true,
		EnvironmentMode:                  "demo",
		DefaultChannel:                   "in_app",
		DefaultChannelProviderID:         "in-app-default",
		AutoObserveEnabled:               true,
		AutoNotifyEnabled:                true,
		AutoContainEnabled:               true,
		AutoRecoverEnabled:               false,
		RequireApprovalForRecover:        true,
		RequireApprovalForDisruptive:     true,
		DuplicateAlertSuppressionSeconds: 900,
		ActionCooldownSeconds:            900,
		EnabledDomains:                   []string{"infrastructure"},
		ChannelProviders: []RobotSREChannelProvider{
			{ID: "in-app-default", Name: "In-App", Kind: "in_app", Enabled: true},
		},
		MCPServers: []RobotSREMCPServer{
			{ID: "ops-observability", Name: "Observability", Kind: "observability", Enabled: true, Transport: "embedded"},
		},
		AgentRuntime: RobotSREAgentRuntimeConfig{
			Enabled:                true,
			Provider:               "openai",
			SystemPromptRef:        "sre_default",
			MaxToolCallsPerTurn:    6,
			MaxIncidentsPerSummary: 5,
		},
	}

	got, err := service.UpdateRobotSREPolicyConfig(ctx, &tenantID, cfg, updatedBy)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "agent_runtime.model")
}

func TestService_SimulateQuarantinePolicy(t *testing.T) {
	logger := zap.NewNop()
	service := NewService(&MockRepository{}, logger)

	policy := &QuarantinePolicyConfig{
		Enabled:     true,
		Mode:        "enforce",
		MaxCritical: 0,
		MaxP2:       4,
		MaxP3:       10,
		MaxCVSS:     8.0,
		SeverityMapping: QuarantinePolicySeverityMapping{
			P1: []string{"critical"},
			P2: []string{"high"},
			P3: []string{"medium"},
			P4: []string{"low", "unknown"},
		},
	}

	result, err := service.SimulateQuarantinePolicy(policy, map[string]interface{}{
		"vulnerabilities": map[string]interface{}{
			"critical": 1,
			"high":     5,
			"medium":   2,
		},
		"max_cvss": 9.1,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "quarantine", result.Decision)
	assert.Equal(t, "enforce", result.Mode)
	assert.NotEmpty(t, result.Reasons)

	policy.Mode = "dry_run"
	result, err = service.SimulateQuarantinePolicy(policy, map[string]interface{}{
		"critical_count": 2,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "pass", result.Decision)
	assert.NotEmpty(t, result.Reasons)
}

func TestService_GetTektonTaskImagesConfig_DefaultsWhenMissing(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)
	ctx := context.Background()

	mockRepo.On("FindByKey", ctx, (*uuid.UUID)(nil), "tekton_task_images").Return(nil, ErrConfigNotFound).Once()

	cfg, err := service.GetTektonTaskImagesConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "docker.io/moby/buildkit:v0.13.2", cfg.Buildkit)
	assert.Equal(t, "quay.io/skopeo/stable:v1.15.0", cfg.Skopeo)
	assert.Equal(t, "docker.io/bitnami/kubectl:latest", cfg.CleanupKubectl)
}

func TestService_GetTektonTaskImagesConfig_NormalizesLegacyShortnames(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, logger)
	ctx := context.Background()

	configValue := TektonTaskImagesConfig{
		GitClone:       "alpine/git:2.45.2",
		KanikoExecutor: "gcr.io/kaniko-project/executor:v1.23.2",
		Buildkit:       "moby/buildkit:v0.13.2",
		Skopeo:         "quay.io/skopeo/stable:v1.15.0",
		Trivy:          "aquasec/trivy:0.57.1",
		Syft:           "anchore/syft:v1.18.1",
		Cosign:         "gcr.io/projectsigstore/cosign:v2.4.1",
		Packer:         "hashicorp/packer:1.10.2",
		PythonAlpine:   "python:3.12-alpine",
		Alpine:         "alpine:3.20",
		CleanupKubectl: "bitnami/kubectl:latest",
	}
	config, err := NewSystemConfig(nil, ConfigTypeTekton, "tekton_task_images", configValue, "Legacy shortnames", uuid.New())
	require.NoError(t, err)
	mockRepo.On("FindByKey", ctx, (*uuid.UUID)(nil), "tekton_task_images").Return(config, nil).Once()

	cfg, err := service.GetTektonTaskImagesConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "docker.io/alpine/git:2.45.2", cfg.GitClone)
	assert.Equal(t, "docker.io/moby/buildkit:v0.13.2", cfg.Buildkit)
	assert.Equal(t, "docker.io/aquasec/trivy:0.57.1", cfg.Trivy)
	assert.Equal(t, "docker.io/anchore/syft:v1.18.1", cfg.Syft)
	assert.Equal(t, "docker.io/hashicorp/packer:1.10.2", cfg.Packer)
	assert.Equal(t, "docker.io/library/python:3.12-alpine", cfg.PythonAlpine)
	assert.Equal(t, "docker.io/library/alpine:3.20", cfg.Alpine)
	assert.Equal(t, "docker.io/bitnami/kubectl:latest", cfg.CleanupKubectl)
}

func TestService_UpdateTektonTaskImagesConfig_RejectsInvalidImageReference(t *testing.T) {
	logger := zap.NewNop()
	service := NewService(&MockRepository{}, logger)

	updated, err := service.UpdateTektonTaskImagesConfig(context.Background(), &TektonTaskImagesConfig{
		GitClone:       "not valid image",
		KanikoExecutor: "gcr.io/kaniko-project/executor:v1.23.2",
		Buildkit:       "docker.io/moby/buildkit:v0.13.2",
		Skopeo:         "quay.io/skopeo/stable:v1.15.0",
		Trivy:          "docker.io/aquasec/trivy:0.57.1",
		Syft:           "docker.io/anchore/syft:v1.18.1",
		Cosign:         "gcr.io/projectsigstore/cosign:v2.4.1",
		Packer:         "docker.io/hashicorp/packer:1.10.2",
		PythonAlpine:   "docker.io/library/python:3.12-alpine",
		Alpine:         "docker.io/library/alpine:3.20",
		CleanupKubectl: "docker.io/bitnami/kubectl:latest",
	}, uuid.New())
	require.Error(t, err)
	assert.Nil(t, updated)
	assert.Contains(t, err.Error(), "git_clone")
}

func TestService_UpdateTektonTaskImagesConfig_RejectsShortImageNames(t *testing.T) {
	logger := zap.NewNop()
	service := NewService(&MockRepository{}, logger)

	updated, err := service.UpdateTektonTaskImagesConfig(context.Background(), &TektonTaskImagesConfig{
		GitClone:       "alpine/git:2.45.2",
		KanikoExecutor: "gcr.io/kaniko-project/executor:v1.23.2",
		Buildkit:       "docker.io/moby/buildkit:v0.13.2",
		Skopeo:         "quay.io/skopeo/stable:v1.15.0",
		Trivy:          "docker.io/aquasec/trivy:0.57.1",
		Syft:           "docker.io/anchore/syft:v1.18.1",
		Cosign:         "gcr.io/projectsigstore/cosign:v2.4.1",
		Packer:         "docker.io/hashicorp/packer:1.10.2",
		PythonAlpine:   "docker.io/library/python:3.12-alpine",
		Alpine:         "docker.io/library/alpine:3.20",
		CleanupKubectl: "docker.io/bitnami/kubectl:latest",
	}, uuid.New())
	require.Error(t, err)
	assert.Nil(t, updated)
	assert.Contains(t, err.Error(), "fully qualified")
	assert.Contains(t, err.Error(), "git_clone")
}
