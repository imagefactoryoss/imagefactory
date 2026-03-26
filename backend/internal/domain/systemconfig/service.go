package systemconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service defines the business logic for system configuration management
type Service struct {
	repository Repository
	logger     *zap.Logger
}

// ValidationError represents field-level config validation failures that can be
// surfaced directly by REST handlers.
type ValidationError struct {
	Message     string
	FieldErrors map[string]string
}

var cronFieldPattern = regexp.MustCompile(`^[A-Za-z0-9*/,\-?]+$`)
var containerImageReferencePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/\-:@]+$`)

func (e *ValidationError) Error() string {
	if e == nil {
		return "validation failed"
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	return "validation failed"
}

// NewService creates a new system configuration service
func NewService(repository Repository, logger *zap.Logger) *Service {
	return &Service{
		repository: repository,
		logger:     logger,
	}
}

// CreateConfigRequest represents a request to create a new configuration
type CreateConfigRequest struct {
	TenantID    *uuid.UUID
	ConfigType  ConfigType
	ConfigKey   string
	ConfigValue interface{}
	Description string
	CreatedBy   uuid.UUID
}

// UpdateConfigRequest represents a request to update a configuration
type UpdateConfigRequest struct {
	ID          uuid.UUID
	ConfigValue interface{}
	Description *string
	UpdatedBy   uuid.UUID
}

// SecurityConfig represents security-related configuration settings
type SecurityConfig struct {
	JWTExpirationHours    int  // Hours for JWT access token expiration
	RefreshTokenHours     int  // Hours for refresh token expiration
	MaxLoginAttempts      int  // Maximum failed login attempts before lockout
	AccountLockDuration   int  // Minutes to lock account after max attempts
	PasswordMinLength     int  // Minimum password length
	RequireSpecialChars   bool // Require special characters in passwords
	RequireNumbers        bool // Require numbers in passwords
	RequireUppercase      bool // Require uppercase letters in passwords
	SessionTimeoutMinutes int  // Minutes before session timeout
}

// CreateConfig creates a new system configuration
func (s *Service) CreateConfig(ctx context.Context, req CreateConfigRequest) (*SystemConfig, error) {
	// Validate request
	if req.TenantID == nil {
		return nil, errors.New("tenant ID is required")
	}
	if req.ConfigKey == "" {
		return nil, ErrInvalidConfigKey
	}
	if req.CreatedBy == uuid.Nil {
		return nil, errors.New("created by user ID is required")
	}
	if err := validateCategoryConfigValue(req.ConfigType, req.ConfigKey, req.ConfigValue); err != nil {
		return nil, err
	}

	// Check if configuration already exists
	exists, err := s.repository.ExistsByKey(ctx, req.TenantID, req.ConfigKey)
	if err != nil {
		s.logger.Error("Failed to check if config exists", zap.Error(err))
		return nil, err
	}
	if exists {
		return nil, ErrConfigAlreadyExists
	}

	// Create the configuration
	config, err := NewSystemConfig(req.TenantID, req.ConfigType, req.ConfigKey, req.ConfigValue, req.Description, req.CreatedBy)
	if err != nil {
		s.logger.Error("Failed to create system config", zap.Error(err))
		return nil, err
	}

	// Save to repository
	if err := s.repository.Save(ctx, config); err != nil {
		s.logger.Error("Failed to save system config", zap.Error(err))
		return nil, err
	}

	s.logger.Info("System configuration created",
		zap.String("configKey", req.ConfigKey),
		zap.String("configType", string(req.ConfigType)),
		zap.String("tenantID", req.TenantID.String()))

	return config, nil
}

// CreateOrUpdateCategoryConfig creates or updates a category-based system configuration
func (s *Service) CreateOrUpdateCategoryConfig(ctx context.Context, tenantID *uuid.UUID, configType ConfigType, configKey string, configValue interface{}, createdBy uuid.UUID) (*SystemConfig, error) {
	if configKey == "" {
		return nil, ErrInvalidConfigKey
	}
	if createdBy == uuid.Nil {
		return nil, errors.New("created by user ID is required")
	}
	if err := validateCategoryConfigValue(configType, configKey, configValue); err != nil {
		return nil, err
	}

	// Check if configuration already exists
	exists, err := s.repository.ExistsByKey(ctx, tenantID, configKey)
	if err != nil {
		s.logger.Error("Failed to check if config exists", zap.Error(err))
		return nil, err
	}

	if exists {
		// Update existing configuration
		existingConfig, err := s.repository.FindByKey(ctx, tenantID, configKey)
		if err != nil {
			s.logger.Error("Failed to find existing config for update", zap.Error(err))
			return nil, err
		}

		// Update value (description is already set appropriately)
		if err := existingConfig.UpdateValue(configValue, createdBy); err != nil {
			s.logger.Error("Failed to update config value", zap.Error(err))
			return nil, err
		}

		// Save updated configuration
		if err := s.repository.Update(ctx, existingConfig); err != nil {
			s.logger.Error("Failed to save updated config", zap.Error(err))
			return nil, err
		}

		s.logger.Info("System configuration updated",
			zap.String("configKey", configKey),
			zap.String("configType", string(configType)))

		return existingConfig, nil
	} else {
		// Create new configuration
		description := fmt.Sprintf("%s configuration settings", configKey)
		config, err := NewSystemConfig(tenantID, configType, configKey, configValue, description, createdBy)
		if err != nil {
			s.logger.Error("Failed to create system config", zap.Error(err))
			return nil, err
		}

		// Save to repository
		if err := s.repository.Save(ctx, config); err != nil {
			s.logger.Error("Failed to save system config", zap.Error(err))
			return nil, err
		}

		s.logger.Info("System configuration created",
			zap.String("configKey", configKey),
			zap.String("configType", string(configType)))

		return config, nil
	}
}

// GetConfig retrieves a configuration by ID
func (s *Service) GetConfig(ctx context.Context, id uuid.UUID) (*SystemConfig, error) {
	if id == uuid.Nil {
		return nil, errors.New("config ID is required")
	}

	config, err := s.repository.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to find config by ID", zap.Error(err), zap.String("id", id.String()))
		return nil, err
	}

	return config, nil
}

// GetConfigByKey retrieves a configuration by tenant and key
func (s *Service) GetConfigByKey(ctx context.Context, tenantID *uuid.UUID, configKey string) (*SystemConfig, error) {
	if configKey == "" {
		return nil, ErrInvalidConfigKey
	}

	config, err := s.repository.FindByKey(ctx, tenantID, configKey)
	if err != nil {
		s.logger.Error("Failed to find config by key",
			zap.Error(err),
			zap.String("configKey", configKey))
		return nil, err
	}

	return config, nil
}

// GetConfigByTypeAndKey retrieves a configuration by tenant, type, and key
func (s *Service) GetConfigByTypeAndKey(ctx context.Context, tenantID *uuid.UUID, configType ConfigType, configKey string) (*SystemConfig, error) {
	if configKey == "" {
		return nil, ErrInvalidConfigKey
	}

	config, err := s.repository.FindByTypeAndKey(ctx, tenantID, configType, configKey)
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			s.logger.Debug("Config not found by type and key",
				zap.String("configType", string(configType)),
				zap.String("configKey", configKey))
		} else {
			s.logger.Error("Failed to find config by type and key",
				zap.Error(err),
				zap.String("configType", string(configType)),
				zap.String("configKey", configKey))
		}
		return nil, err
	}

	return config, nil
}

// GetConfigsByType retrieves all configurations of a specific type for a tenant
func (s *Service) GetConfigsByType(ctx context.Context, tenantID *uuid.UUID, configType ConfigType) ([]*SystemConfig, error) {
	if tenantID == nil {
		// For universal configs, return configs of this type where tenant_id IS NULL
		configs, err := s.repository.FindUniversalByType(ctx, configType)
		if err != nil {
			s.logger.Error("Failed to find universal configs by type",
				zap.Error(err),
				zap.String("configType", string(configType)))
			return nil, err
		}
		return configs, nil
	}

	tenantConfigs, err := s.repository.FindByType(ctx, tenantID, configType)
	if err != nil {
		s.logger.Error("Failed to find configs by type",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()),
			zap.String("configType", string(configType)))
		return nil, err
	}

	universalConfigs, err := s.repository.FindUniversalByType(ctx, configType)
	if err != nil {
		s.logger.Error("Failed to find universal configs by type for tenant merge",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()),
			zap.String("configType", string(configType)))
		return nil, err
	}

	return mergeConfigsWithTenantOverrides(universalConfigs, tenantConfigs), nil
}

// GetConfigsByTypeAllScopes retrieves all configurations of a specific type across all scopes.
// This is intended for cross-tenant platform capabilities (for example, auth provider discovery).
func (s *Service) GetConfigsByTypeAllScopes(ctx context.Context, configType ConfigType) ([]*SystemConfig, error) {
	configs, err := s.repository.FindAllByType(ctx, configType)
	if err != nil {
		s.logger.Error("Failed to find all configs by type",
			zap.Error(err),
			zap.String("configType", string(configType)))
		return nil, err
	}
	return configs, nil
}

// GetActiveConfigsByType retrieves active configurations of a specific type for a tenant
func (s *Service) GetActiveConfigsByType(ctx context.Context, tenantID uuid.UUID, configType ConfigType) ([]*SystemConfig, error) {
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant ID is required")
	}

	configs, err := s.repository.FindActiveByType(ctx, tenantID, configType)
	if err != nil {
		s.logger.Error("Failed to find active configs by type",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()),
			zap.String("configType", string(configType)))
		return nil, err
	}

	return configs, nil
}

// GetAllConfigs retrieves all configurations for a tenant
func (s *Service) GetAllConfigs(ctx context.Context, tenantID *uuid.UUID) ([]*SystemConfig, error) {
	if tenantID == nil {
		// For universal configs, return all configs (both tenant-specific and universal)
		configs, err := s.repository.FindAll(ctx)
		if err != nil {
			s.logger.Error("Failed to find all configs",
				zap.Error(err))
			return nil, err
		}
		return configs, nil
	}

	tenantConfigs, err := s.repository.FindByTenantID(ctx, *tenantID)
	if err != nil {
		s.logger.Error("Failed to find all configs",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()))
		return nil, err
	}

	universalConfigs, err := s.getAllUniversalConfigs(ctx)
	if err != nil {
		s.logger.Error("Failed to load universal configs for tenant merge",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()))
		return nil, err
	}

	return mergeConfigsWithTenantOverrides(universalConfigs, tenantConfigs), nil
}

func (s *Service) getAllUniversalConfigs(ctx context.Context) ([]*SystemConfig, error) {
	configTypes := []ConfigType{
		ConfigTypeLDAP,
		ConfigTypeSMTP,
		ConfigTypeGeneral,
		ConfigTypeSecurity,
		ConfigTypeBuild,
		ConfigTypeToolSettings,
		ConfigTypeExternalServices,
		ConfigTypeMessaging,
		ConfigTypeRuntimeServices,
	}

	configs := make([]*SystemConfig, 0)
	for _, configType := range configTypes {
		byType, err := s.repository.FindUniversalByType(ctx, configType)
		if err != nil {
			return nil, err
		}
		configs = append(configs, byType...)
	}

	return configs, nil
}

func mergeConfigsWithTenantOverrides(universal, tenantSpecific []*SystemConfig) []*SystemConfig {
	byKey := make(map[string]*SystemConfig, len(universal)+len(tenantSpecific))
	keys := make([]string, 0, len(universal)+len(tenantSpecific))

	for _, cfg := range universal {
		key := string(cfg.ConfigType()) + "::" + cfg.ConfigKey()
		if _, exists := byKey[key]; !exists {
			keys = append(keys, key)
		}
		byKey[key] = cfg
	}

	for _, cfg := range tenantSpecific {
		key := string(cfg.ConfigType()) + "::" + cfg.ConfigKey()
		if _, exists := byKey[key]; !exists {
			keys = append(keys, key)
		}
		byKey[key] = cfg
	}

	sort.Strings(keys)
	merged := make([]*SystemConfig, 0, len(keys))
	for _, key := range keys {
		merged = append(merged, byKey[key])
	}
	return merged
}

// UpdateConfig updates an existing configuration
func (s *Service) UpdateConfig(ctx context.Context, req UpdateConfigRequest) (*SystemConfig, error) {
	if req.ID == uuid.Nil {
		return nil, errors.New("config ID is required")
	}
	if req.UpdatedBy == uuid.Nil {
		return nil, errors.New("updated by user ID is required")
	}

	// Get existing configuration
	config, err := s.repository.FindByID(ctx, req.ID)
	if err != nil {
		s.logger.Error("Failed to find config for update", zap.Error(err), zap.String("id", req.ID.String()))
		return nil, err
	}

	// Update value if provided
	if req.ConfigValue != nil {
		if err := validateCategoryConfigValue(config.ConfigType(), config.ConfigKey(), req.ConfigValue); err != nil {
			return nil, err
		}
		if err := config.UpdateValue(req.ConfigValue, req.UpdatedBy); err != nil {
			s.logger.Error("Failed to update config value", zap.Error(err))
			return nil, err
		}
	}

	// Update description if provided
	if req.Description != nil {
		if err := config.UpdateDescription(*req.Description, req.UpdatedBy); err != nil {
			s.logger.Error("Failed to update config description", zap.Error(err))
			return nil, err
		}
	}

	// Save updated configuration
	if err := s.repository.Update(ctx, config); err != nil {
		s.logger.Error("Failed to save updated config", zap.Error(err))
		return nil, err
	}

	s.logger.Info("System configuration updated",
		zap.String("configKey", config.ConfigKey()),
		zap.String("configType", string(config.ConfigType())),
		zap.String("updatedBy", req.UpdatedBy.String()))

	return config, nil
}

func validateCategoryConfigValue(configType ConfigType, configKey string, configValue interface{}) error {
	if configType != ConfigTypeRuntimeServices || strings.TrimSpace(configKey) != "runtime_services" {
		return nil
	}

	encoded, err := json.Marshal(configValue)
	if err != nil {
		return fmt.Errorf("invalid runtime_services config payload: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(encoded, &raw); err != nil {
		return fmt.Errorf("invalid runtime_services config payload: %w", err)
	}

	var runtimeCfg RuntimeServicesConfig
	if err := json.Unmarshal(encoded, &runtimeCfg); err != nil {
		return fmt.Errorf("invalid runtime_services config payload: %w", err)
	}

	fieldErrors := make(map[string]string)

	if _, exists := raw["image_import_notification_receipt_retention_days"]; exists {
		if runtimeCfg.ImageImportNotificationReceiptRetentionDays < 1 || runtimeCfg.ImageImportNotificationReceiptRetentionDays > 3650 {
			fieldErrors["image_import_notification_receipt_retention_days"] = "must be between 1 and 3650"
		}
	}
	if _, exists := raw["image_import_notification_receipt_cleanup_interval_hours"]; exists {
		if runtimeCfg.ImageImportNotificationReceiptCleanupIntervalHours < 1 || runtimeCfg.ImageImportNotificationReceiptCleanupIntervalHours > 168 {
			fieldErrors["image_import_notification_receipt_cleanup_interval_hours"] = "must be between 1 and 168"
		}
	}
	if _, exists := raw["provider_readiness_watcher_interval_seconds"]; exists {
		if runtimeCfg.ProviderReadinessWatcherIntervalSeconds < 30 {
			fieldErrors["provider_readiness_watcher_interval_seconds"] = "must be at least 30"
		}
	}
	if _, exists := raw["provider_readiness_watcher_timeout_seconds"]; exists {
		if runtimeCfg.ProviderReadinessWatcherTimeoutSeconds < 10 {
			fieldErrors["provider_readiness_watcher_timeout_seconds"] = "must be at least 10"
		}
	}
	if _, exists := raw["provider_readiness_watcher_batch_size"]; exists {
		if runtimeCfg.ProviderReadinessWatcherBatchSize < 1 || runtimeCfg.ProviderReadinessWatcherBatchSize > 1000 {
			fieldErrors["provider_readiness_watcher_batch_size"] = "must be between 1 and 1000"
		}
	}
	if _, exists := raw["dispatcher_url"]; exists {
		if err := validateRuntimeServiceURL(runtimeCfg.DispatcherURL); err != nil {
			fieldErrors["dispatcher_url"] = err.Error()
		}
	}
	if _, exists := raw["email_worker_url"]; exists {
		if err := validateRuntimeServiceURL(runtimeCfg.EmailWorkerURL); err != nil {
			fieldErrors["email_worker_url"] = err.Error()
		}
	}
	if _, exists := raw["notification_worker_url"]; exists {
		if err := validateRuntimeServiceURL(runtimeCfg.NotificationWorkerURL); err != nil {
			fieldErrors["notification_worker_url"] = err.Error()
		}
	}
	if _, exists := raw["internal_registry_gc_worker_url"]; exists {
		if err := validateRuntimeServiceURL(runtimeCfg.InternalRegistryGCWorkerURL); err != nil {
			fieldErrors["internal_registry_gc_worker_url"] = err.Error()
		}
	}
	if _, exists := raw["dispatcher_port"]; exists {
		if runtimeCfg.DispatcherPort < 1 || runtimeCfg.DispatcherPort > 65535 {
			fieldErrors["dispatcher_port"] = "must be between 1 and 65535"
		}
	}
	if _, exists := raw["email_worker_port"]; exists {
		if runtimeCfg.EmailWorkerPort < 1 || runtimeCfg.EmailWorkerPort > 65535 {
			fieldErrors["email_worker_port"] = "must be between 1 and 65535"
		}
	}
	if _, exists := raw["notification_worker_port"]; exists {
		if runtimeCfg.NotificationWorkerPort < 1 || runtimeCfg.NotificationWorkerPort > 65535 {
			fieldErrors["notification_worker_port"] = "must be between 1 and 65535"
		}
	}
	if _, exists := raw["internal_registry_gc_worker_port"]; exists {
		if runtimeCfg.InternalRegistryGCWorkerPort < 1 || runtimeCfg.InternalRegistryGCWorkerPort > 65535 {
			fieldErrors["internal_registry_gc_worker_port"] = "must be between 1 and 65535"
		}
	}
	if _, exists := raw["health_check_timeout_seconds"]; exists {
		if runtimeCfg.HealthCheckTimeoutSecond < 1 {
			fieldErrors["health_check_timeout_seconds"] = "must be at least 1"
		}
	}
	if _, exists := raw["internal_registry_temp_cleanup_retention_hours"]; exists {
		if runtimeCfg.InternalRegistryTempCleanupRetentionHours < 1 || runtimeCfg.InternalRegistryTempCleanupRetentionHours > 8760 {
			fieldErrors["internal_registry_temp_cleanup_retention_hours"] = "must be between 1 and 8760"
		}
	}
	if _, exists := raw["internal_registry_temp_cleanup_interval_minutes"]; exists {
		if runtimeCfg.InternalRegistryTempCleanupIntervalMinutes < 1 || runtimeCfg.InternalRegistryTempCleanupIntervalMinutes > 10080 {
			fieldErrors["internal_registry_temp_cleanup_interval_minutes"] = "must be between 1 and 10080"
		}
	}
	if _, exists := raw["internal_registry_temp_cleanup_batch_size"]; exists {
		if runtimeCfg.InternalRegistryTempCleanupBatchSize < 1 || runtimeCfg.InternalRegistryTempCleanupBatchSize > 5000 {
			fieldErrors["internal_registry_temp_cleanup_batch_size"] = "must be between 1 and 5000"
		}
	}
	if _, exists := raw["tekton_history_cleanup_keep_pipelineruns"]; exists {
		if runtimeCfg.TektonHistoryCleanupKeepPipelineRuns < 1 {
			fieldErrors["tekton_history_cleanup_keep_pipelineruns"] = "must be at least 1"
		}
	}
	if _, exists := raw["tekton_history_cleanup_keep_taskruns"]; exists {
		if runtimeCfg.TektonHistoryCleanupKeepTaskRuns < 1 {
			fieldErrors["tekton_history_cleanup_keep_taskruns"] = "must be at least 1"
		}
	}
	if _, exists := raw["tekton_history_cleanup_keep_pods"]; exists {
		if runtimeCfg.TektonHistoryCleanupKeepPods < 1 {
			fieldErrors["tekton_history_cleanup_keep_pods"] = "must be at least 1"
		}
	}
	if _, exists := raw["tekton_history_cleanup_schedule"]; exists {
		if err := validateCronExpression(runtimeCfg.TektonHistoryCleanupSchedule); err != nil {
			fieldErrors["tekton_history_cleanup_schedule"] = err.Error()
		}
	}
	if _, exists := raw["storage_profiles"]; exists {
		validateRuntimeAssetStorageProfile("storage_profiles.internal_registry", runtimeCfg.StorageProfiles.InternalRegistry, fieldErrors)
		validateRuntimeAssetStorageProfile("storage_profiles.trivy_cache", runtimeCfg.StorageProfiles.TrivyCache, fieldErrors)
	}

	if _, timeoutExists := raw["provider_readiness_watcher_timeout_seconds"]; timeoutExists {
		if _, intervalExists := raw["provider_readiness_watcher_interval_seconds"]; intervalExists {
			if runtimeCfg.ProviderReadinessWatcherTimeoutSeconds >= runtimeCfg.ProviderReadinessWatcherIntervalSeconds {
				fieldErrors["provider_readiness_watcher_timeout_seconds"] = "must be less than provider_readiness_watcher_interval_seconds"
			}
		}
	}

	if len(fieldErrors) > 0 {
		return &ValidationError{
			Message:     "runtime_services validation failed",
			FieldErrors: fieldErrors,
		}
	}

	return nil
}

func defaultRobotSREPolicyConfig() RobotSREPolicyConfig {
	defaultAgentRuntimeBaseURL := strings.TrimSpace(os.Getenv("IF_SRE_AGENT_RUNTIME_BASE_URL"))
	if defaultAgentRuntimeBaseURL == "" {
		defaultAgentRuntimeBaseURL = "http://127.0.0.1:11434"
	}
	defaultAgentRuntimeModel := strings.TrimSpace(os.Getenv("IF_SRE_AGENT_RUNTIME_MODEL"))
	if defaultAgentRuntimeModel == "" {
		defaultAgentRuntimeModel = "llama3.2:3b"
	}
	return RobotSREPolicyConfig{
		DisplayName:                      "SRE Smart Bot",
		Enabled:                          true,
		EnvironmentMode:                  "demo",
		DetectorLearningMode:             "suggest_only",
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
		EnabledDomains: []string{
			"infrastructure",
			"runtime_services",
			"application_services",
			"network_ingress",
			"identity_security",
			"release_configuration",
			"operator_channels",
		},
		ChannelProviders: []RobotSREChannelProvider{
			{
				ID:                          "in-app-default",
				Name:                        "In-App Admin Notifications",
				Kind:                        "in_app",
				Enabled:                     true,
				SupportsInteractiveApproval: true,
			},
		},
		MCPServers: []RobotSREMCPServer{
			{
				ID:               "observability-default",
				Name:             "Observability Read Model",
				Kind:             "observability",
				Enabled:          true,
				Transport:        "embedded",
				AllowedTools:     []string{"incidents.list", "incidents.get", "findings.list", "evidence.list", "runtime_health.get", "logs.recent"},
				ReadOnly:         true,
				ApprovalRequired: false,
			},
			{
				ID:               "release-default",
				Name:             "Release Compliance Read Model",
				Kind:             "release",
				Enabled:          true,
				Transport:        "embedded",
				AllowedTools:     []string{"release_drift.summary"},
				ReadOnly:         true,
				ApprovalRequired: false,
			},
		},
		AgentRuntime: RobotSREAgentRuntimeConfig{
			Enabled:                            false,
			Provider:                           "ollama",
			Model:                              defaultAgentRuntimeModel,
			BaseURL:                            defaultAgentRuntimeBaseURL,
			SystemPromptRef:                    "sre_smart_bot_default",
			OperatorSummaryEnabled:             true,
			HypothesisRankingEnabled:           true,
			DraftActionPlansEnabled:            true,
			ConversationalApprovalSupport:      false,
			MaxToolCallsPerTurn:                6,
			MaxIncidentsPerSummary:             5,
			RequireHumanConfirmationForMessage: true,
		},
		DetectorRules: []RobotSREDetectorRule{},
		OperatorRules: []RobotSREOperatorRule{},
	}
}

// DefaultRobotSREPolicyConfig returns the deployment-aware default Robot SRE
// policy configuration for bootstrap and first-run setup flows.
func DefaultRobotSREPolicyConfig() RobotSREPolicyConfig {
	return defaultRobotSREPolicyConfig()
}

func applyRobotSREPolicyDefaults(cfg *RobotSREPolicyConfig) {
	if cfg == nil {
		return
	}
	defaultCfg := defaultRobotSREPolicyConfig()
	if strings.TrimSpace(cfg.DisplayName) == "" {
		cfg.DisplayName = defaultCfg.DisplayName
	}
	if strings.TrimSpace(cfg.EnvironmentMode) == "" {
		cfg.EnvironmentMode = defaultCfg.EnvironmentMode
	}
	if strings.TrimSpace(cfg.DetectorLearningMode) == "" {
		cfg.DetectorLearningMode = defaultCfg.DetectorLearningMode
	}
	if strings.TrimSpace(cfg.DefaultChannel) == "" {
		cfg.DefaultChannel = defaultCfg.DefaultChannel
	}
	if strings.TrimSpace(cfg.DefaultChannelProviderID) == "" {
		cfg.DefaultChannelProviderID = defaultCfg.DefaultChannelProviderID
	}
	if cfg.DuplicateAlertSuppressionSeconds <= 0 {
		cfg.DuplicateAlertSuppressionSeconds = defaultCfg.DuplicateAlertSuppressionSeconds
	}
	if cfg.ActionCooldownSeconds <= 0 {
		cfg.ActionCooldownSeconds = defaultCfg.ActionCooldownSeconds
	}
	if len(cfg.EnabledDomains) == 0 {
		cfg.EnabledDomains = append([]string(nil), defaultCfg.EnabledDomains...)
	}
	if len(cfg.ChannelProviders) == 0 {
		cfg.ChannelProviders = append([]RobotSREChannelProvider(nil), defaultCfg.ChannelProviders...)
	}
	if len(cfg.MCPServers) == 0 {
		cfg.MCPServers = append([]RobotSREMCPServer(nil), defaultCfg.MCPServers...)
	}
	if strings.TrimSpace(cfg.AgentRuntime.Provider) == "" {
		cfg.AgentRuntime.Provider = defaultCfg.AgentRuntime.Provider
	}
	if strings.TrimSpace(cfg.AgentRuntime.BaseURL) == "" {
		cfg.AgentRuntime.BaseURL = defaultCfg.AgentRuntime.BaseURL
	}
	if strings.TrimSpace(cfg.AgentRuntime.SystemPromptRef) == "" {
		cfg.AgentRuntime.SystemPromptRef = defaultCfg.AgentRuntime.SystemPromptRef
	}
	if cfg.AgentRuntime.MaxToolCallsPerTurn <= 0 {
		cfg.AgentRuntime.MaxToolCallsPerTurn = defaultCfg.AgentRuntime.MaxToolCallsPerTurn
	}
	if cfg.AgentRuntime.MaxIncidentsPerSummary <= 0 {
		cfg.AgentRuntime.MaxIncidentsPerSummary = defaultCfg.AgentRuntime.MaxIncidentsPerSummary
	}
	if cfg.OperatorRules == nil {
		cfg.OperatorRules = []RobotSREOperatorRule{}
	}
	if cfg.DetectorRules == nil {
		cfg.DetectorRules = []RobotSREDetectorRule{}
	}
}

func validateRobotSREPolicyConfig(cfg *RobotSREPolicyConfig) error {
	if cfg == nil {
		return errors.New("robot sre policy is required")
	}

	applyRobotSREPolicyDefaults(cfg)

	validEnvironmentModes := map[string]struct{}{
		"demo":       {},
		"staging":    {},
		"production": {},
	}
	validDetectorLearningModes := map[string]struct{}{
		"disabled":             {},
		"suggest_only":         {},
		"training_auto_create": {},
	}
	validChannels := map[string]struct{}{
		"in_app":   {},
		"email":    {},
		"webhook":  {},
		"slack":    {},
		"teams":    {},
		"telegram": {},
		"whatsapp": {},
		"custom":   {},
	}
	validMCPKinds := map[string]struct{}{
		"kubernetes":    {},
		"oci":           {},
		"database":      {},
		"release":       {},
		"chat":          {},
		"observability": {},
		"custom":        {},
	}
	validMCPTransports := map[string]struct{}{
		"embedded": {},
		"http":     {},
		"stdio":    {},
		"custom":   {},
	}
	validAgentProviders := map[string]struct{}{
		"custom": {},
		"ollama": {},
		"openai": {},
		"none":   {},
	}
	validDomains := map[string]struct{}{
		"infrastructure":        {},
		"runtime_services":      {},
		"application_services":  {},
		"network_ingress":       {},
		"identity_security":     {},
		"release_configuration": {},
		"operator_channels":     {},
	}
	validSeverities := map[string]struct{}{
		"info":     {},
		"warning":  {},
		"critical": {},
	}

	if _, ok := validEnvironmentModes[strings.TrimSpace(cfg.EnvironmentMode)]; !ok {
		return fmt.Errorf("environment_mode must be one of: demo, staging, production")
	}
	if _, ok := validDetectorLearningModes[strings.TrimSpace(cfg.DetectorLearningMode)]; !ok {
		return fmt.Errorf("detector_learning_mode must be one of: disabled, suggest_only, training_auto_create")
	}
	if _, ok := validChannels[strings.TrimSpace(cfg.DefaultChannel)]; !ok {
		return fmt.Errorf("default_channel must be one of: in_app, email, webhook, slack, teams, telegram, whatsapp, custom")
	}
	if cfg.DuplicateAlertSuppressionSeconds < 60 || cfg.DuplicateAlertSuppressionSeconds > 86400 {
		return fmt.Errorf("duplicate_alert_suppression_seconds must be between 60 and 86400")
	}
	if cfg.ActionCooldownSeconds < 60 || cfg.ActionCooldownSeconds > 86400 {
		return fmt.Errorf("action_cooldown_seconds must be between 60 and 86400")
	}

	seenDomains := make(map[string]struct{}, len(cfg.EnabledDomains))
	for _, domain := range cfg.EnabledDomains {
		trimmed := strings.TrimSpace(domain)
		if _, ok := validDomains[trimmed]; !ok {
			return fmt.Errorf("enabled_domains contains invalid domain %q", domain)
		}
		if _, exists := seenDomains[trimmed]; exists {
			return fmt.Errorf("enabled_domains contains duplicate domain %q", domain)
		}
		seenDomains[trimmed] = struct{}{}
	}

	seenProviderIDs := make(map[string]struct{}, len(cfg.ChannelProviders))
	defaultProviderFound := false
	for idx := range cfg.ChannelProviders {
		provider := &cfg.ChannelProviders[idx]
		provider.ID = strings.TrimSpace(provider.ID)
		provider.Name = strings.TrimSpace(provider.Name)
		provider.Kind = strings.TrimSpace(provider.Kind)
		provider.ConfigRef = strings.TrimSpace(provider.ConfigRef)

		if provider.ID == "" {
			return fmt.Errorf("channel_providers[%d].id is required", idx)
		}
		if _, exists := seenProviderIDs[provider.ID]; exists {
			return fmt.Errorf("channel_providers contains duplicate id %q", provider.ID)
		}
		seenProviderIDs[provider.ID] = struct{}{}
		if provider.Name == "" {
			return fmt.Errorf("channel_providers[%d].name is required", idx)
		}
		if _, ok := validChannels[provider.Kind]; !ok {
			return fmt.Errorf("channel_providers[%d].kind must be one of: in_app, email, webhook, slack, teams, telegram, whatsapp, custom", idx)
		}
		if provider.ID == cfg.DefaultChannelProviderID {
			defaultProviderFound = true
			if !provider.Enabled {
				return fmt.Errorf("default_channel_provider_id must reference an enabled provider")
			}
		}
	}
	if strings.TrimSpace(cfg.DefaultChannelProviderID) != "" && !defaultProviderFound {
		return fmt.Errorf("default_channel_provider_id must reference a configured provider")
	}

	seenMCPServerIDs := make(map[string]struct{}, len(cfg.MCPServers))
	for idx := range cfg.MCPServers {
		server := &cfg.MCPServers[idx]
		server.ID = strings.TrimSpace(server.ID)
		server.Name = strings.TrimSpace(server.Name)
		server.Kind = strings.TrimSpace(server.Kind)
		server.Transport = strings.TrimSpace(server.Transport)
		server.Endpoint = strings.TrimSpace(server.Endpoint)
		server.ConfigRef = strings.TrimSpace(server.ConfigRef)

		if server.ID == "" {
			return fmt.Errorf("mcp_servers[%d].id is required", idx)
		}
		if _, exists := seenMCPServerIDs[server.ID]; exists {
			return fmt.Errorf("mcp_servers contains duplicate id %q", server.ID)
		}
		seenMCPServerIDs[server.ID] = struct{}{}
		if server.Name == "" {
			return fmt.Errorf("mcp_servers[%d].name is required", idx)
		}
		if _, ok := validMCPKinds[server.Kind]; !ok {
			return fmt.Errorf("mcp_servers[%d].kind must be one of: kubernetes, oci, database, release, chat, observability, custom", idx)
		}
		if _, ok := validMCPTransports[server.Transport]; !ok {
			return fmt.Errorf("mcp_servers[%d].transport must be one of: embedded, http, stdio, custom", idx)
		}
		if server.Transport == "http" && server.Endpoint == "" {
			return fmt.Errorf("mcp_servers[%d].endpoint is required when transport=http", idx)
		}
	}

	cfg.AgentRuntime.Provider = strings.TrimSpace(cfg.AgentRuntime.Provider)
	cfg.AgentRuntime.Model = strings.TrimSpace(cfg.AgentRuntime.Model)
	cfg.AgentRuntime.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.AgentRuntime.BaseURL), "/")
	cfg.AgentRuntime.SystemPromptRef = strings.TrimSpace(cfg.AgentRuntime.SystemPromptRef)
	if _, ok := validAgentProviders[cfg.AgentRuntime.Provider]; !ok {
		return fmt.Errorf("agent_runtime.provider must be one of: custom, ollama, openai, none")
	}
	if cfg.AgentRuntime.MaxToolCallsPerTurn < 1 || cfg.AgentRuntime.MaxToolCallsPerTurn > 20 {
		return fmt.Errorf("agent_runtime.max_tool_calls_per_turn must be between 1 and 20")
	}
	if cfg.AgentRuntime.MaxIncidentsPerSummary < 1 || cfg.AgentRuntime.MaxIncidentsPerSummary > 20 {
		return fmt.Errorf("agent_runtime.max_incidents_per_summary must be between 1 and 20")
	}
	if cfg.AgentRuntime.Enabled && cfg.AgentRuntime.Provider != "none" && cfg.AgentRuntime.Model == "" {
		return fmt.Errorf("agent_runtime.model is required when agent runtime is enabled")
	}
	if cfg.AgentRuntime.Enabled && cfg.AgentRuntime.Provider == "ollama" && cfg.AgentRuntime.BaseURL == "" {
		return fmt.Errorf("agent_runtime.base_url is required when provider=ollama")
	}

	seenRuleIDs := make(map[string]struct{}, len(cfg.OperatorRules))
	for idx := range cfg.OperatorRules {
		rule := &cfg.OperatorRules[idx]
		rule.ID = strings.TrimSpace(rule.ID)
		rule.Name = strings.TrimSpace(rule.Name)
		rule.Domain = strings.TrimSpace(rule.Domain)
		rule.IncidentType = strings.TrimSpace(rule.IncidentType)
		rule.Severity = strings.TrimSpace(rule.Severity)
		rule.Source = strings.TrimSpace(rule.Source)
		if rule.Source == "" {
			rule.Source = "operator_defined"
		}

		if rule.ID == "" {
			return fmt.Errorf("operator_rules[%d].id is required", idx)
		}
		if _, exists := seenRuleIDs[rule.ID]; exists {
			return fmt.Errorf("operator_rules contains duplicate id %q", rule.ID)
		}
		seenRuleIDs[rule.ID] = struct{}{}
		if rule.Name == "" {
			return fmt.Errorf("operator_rules[%d].name is required", idx)
		}
		if _, ok := validDomains[rule.Domain]; !ok {
			return fmt.Errorf("operator_rules[%d].domain is invalid", idx)
		}
		if rule.IncidentType == "" {
			return fmt.Errorf("operator_rules[%d].incident_type is required", idx)
		}
		if _, ok := validSeverities[rule.Severity]; !ok {
			return fmt.Errorf("operator_rules[%d].severity must be one of: info, warning, critical", idx)
		}
		if rule.Threshold < 0 {
			return fmt.Errorf("operator_rules[%d].threshold must be >= 0", idx)
		}
		if rule.ForDurationSeconds < 0 || rule.ForDurationSeconds > 86400 {
			return fmt.Errorf("operator_rules[%d].for_duration_seconds must be between 0 and 86400", idx)
		}
	}

	seenDetectorRuleIDs := make(map[string]struct{}, len(cfg.DetectorRules))
	validConfidences := map[string]struct{}{
		"":       {},
		"low":    {},
		"medium": {},
		"high":   {},
	}
	for idx := range cfg.DetectorRules {
		rule := &cfg.DetectorRules[idx]
		rule.ID = strings.TrimSpace(rule.ID)
		rule.Name = strings.TrimSpace(rule.Name)
		rule.Query = strings.TrimSpace(rule.Query)
		rule.Domain = strings.TrimSpace(rule.Domain)
		rule.IncidentType = strings.TrimSpace(rule.IncidentType)
		rule.Severity = strings.TrimSpace(rule.Severity)
		rule.Confidence = strings.TrimSpace(rule.Confidence)
		rule.SignalKey = strings.TrimSpace(rule.SignalKey)
		rule.Source = strings.TrimSpace(rule.Source)
		if rule.Source == "" {
			rule.Source = "operator_defined"
		}
		if rule.ID == "" {
			return fmt.Errorf("detector_rules[%d].id is required", idx)
		}
		if _, exists := seenDetectorRuleIDs[rule.ID]; exists {
			return fmt.Errorf("detector_rules contains duplicate id %q", rule.ID)
		}
		seenDetectorRuleIDs[rule.ID] = struct{}{}
		if rule.Name == "" {
			return fmt.Errorf("detector_rules[%d].name is required", idx)
		}
		if rule.Query == "" {
			return fmt.Errorf("detector_rules[%d].query is required", idx)
		}
		if _, ok := validDomains[rule.Domain]; !ok {
			return fmt.Errorf("detector_rules[%d].domain is invalid", idx)
		}
		if rule.IncidentType == "" {
			return fmt.Errorf("detector_rules[%d].incident_type is required", idx)
		}
		if _, ok := validSeverities[rule.Severity]; !ok {
			return fmt.Errorf("detector_rules[%d].severity must be one of: info, warning, critical", idx)
		}
		if _, ok := validConfidences[rule.Confidence]; !ok {
			return fmt.Errorf("detector_rules[%d].confidence must be one of: low, medium, high", idx)
		}
		if rule.Threshold < 0 {
			return fmt.Errorf("detector_rules[%d].threshold must be >= 0", idx)
		}
	}

	return nil
}

func validateRuntimeAssetStorageProfile(prefix string, profile RuntimeAssetStorageProfile, fieldErrors map[string]string) {
	storageType := strings.TrimSpace(strings.ToLower(profile.Type))
	switch storageType {
	case "", "hostpath", "pvc", "emptydir":
	default:
		fieldErrors[prefix+".type"] = "must be one of: hostPath, pvc, emptyDir"
		return
	}

	switch storageType {
	case "hostpath":
		if strings.TrimSpace(profile.HostPath) == "" {
			fieldErrors[prefix+".host_path"] = "is required when type is hostPath"
		}
		hostPathType := strings.TrimSpace(profile.HostPathType)
		if hostPathType != "" {
			switch hostPathType {
			case "DirectoryOrCreate", "Directory", "FileOrCreate", "File":
			default:
				fieldErrors[prefix+".host_path_type"] = "must be one of: DirectoryOrCreate, Directory, FileOrCreate, File"
			}
		}
	case "pvc":
		if strings.TrimSpace(profile.PVCName) == "" {
			fieldErrors[prefix+".pvc_name"] = "is required when type is pvc"
		}
		for idx, mode := range profile.PVCAccessModes {
			trimmed := strings.TrimSpace(mode)
			switch trimmed {
			case "ReadWriteOnce", "ReadOnlyMany", "ReadWriteMany", "ReadWriteOncePod":
			default:
				fieldErrors[fmt.Sprintf("%s.pvc_access_modes.%d", prefix, idx)] = "must be one of: ReadWriteOnce, ReadOnlyMany, ReadWriteMany, ReadWriteOncePod"
			}
		}
	}
}

func validateCronExpression(expr string) error {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return errors.New("must be a non-empty cron expression with 5 fields")
	}
	fields := strings.Fields(trimmed)
	if len(fields) != 5 {
		return errors.New("must contain exactly 5 cron fields")
	}
	for _, field := range fields {
		if !cronFieldPattern.MatchString(field) {
			return errors.New("contains unsupported cron tokens")
		}
	}
	return nil
}

func validateRuntimeServiceURL(rawValue string) error {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return errors.New("must be a non-empty absolute URL")
	}
	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil || parsed == nil {
		return errors.New("must be a valid absolute URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("must use http or https scheme")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return errors.New("must include a host")
	}
	return nil
}

// DeleteConfig deletes a configuration
func (s *Service) DeleteConfig(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("config ID is required")
	}

	// Check if configuration exists
	_, err := s.repository.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to find config for deletion", zap.Error(err), zap.String("id", id.String()))
		return err
	}

	// Delete the configuration
	if err := s.repository.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete config", zap.Error(err), zap.String("id", id.String()))
		return err
	}

	s.logger.Info("System configuration deleted", zap.String("id", id.String()))
	return nil
}

// ActivateConfig activates a configuration
func (s *Service) ActivateConfig(ctx context.Context, id uuid.UUID, updatedBy uuid.UUID) (*SystemConfig, error) {
	if id == uuid.Nil {
		return nil, errors.New("config ID is required")
	}
	if updatedBy == uuid.Nil {
		return nil, errors.New("updated by user ID is required")
	}

	config, err := s.repository.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to find config for activation", zap.Error(err), zap.String("id", id.String()))
		return nil, err
	}

	if err := config.Activate(updatedBy); err != nil {
		s.logger.Error("Failed to activate config", zap.Error(err))
		return nil, err
	}

	if err := s.repository.Update(ctx, config); err != nil {
		s.logger.Error("Failed to save activated config", zap.Error(err))
		return nil, err
	}

	s.logger.Info("System configuration activated", zap.String("configKey", config.ConfigKey()))
	return config, nil
}

// DeactivateConfig deactivates a configuration
func (s *Service) DeactivateConfig(ctx context.Context, id uuid.UUID, updatedBy uuid.UUID) (*SystemConfig, error) {
	if id == uuid.Nil {
		return nil, errors.New("config ID is required")
	}
	if updatedBy == uuid.Nil {
		return nil, errors.New("updated by user ID is required")
	}

	config, err := s.repository.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to find config for deactivation", zap.Error(err), zap.String("id", id.String()))
		return nil, err
	}

	if err := config.Deactivate(updatedBy); err != nil {
		s.logger.Error("Failed to deactivate config", zap.Error(err))
		return nil, err
	}

	if err := s.repository.Update(ctx, config); err != nil {
		s.logger.Error("Failed to save deactivated config", zap.Error(err))
		return nil, err
	}

	s.logger.Info("System configuration deactivated", zap.String("configKey", config.ConfigKey()))
	return config, nil
}

// TestLDAPConnection tests LDAP configuration connectivity
func (s *Service) TestLDAPConnection(ctx context.Context, tenantID *uuid.UUID, configKey string) error {
	config, err := s.GetConfigByKey(ctx, tenantID, configKey)
	if err != nil {
		return err
	}

	if config.ConfigType() != ConfigTypeLDAP {
		return errors.New("configuration is not LDAP type")
	}

	ldapConfig, err := config.GetLDAPConfig()
	if err != nil {
		return err
	}

	// Set testing status
	if err := config.SetTestingStatus(config.CreatedBy()); err != nil {
		return err
	}

	// TODO: Implement actual LDAP connection test
	// For now, just validate the configuration structure
	if ldapConfig.Host == "" {
		return errors.New("LDAP host is required")
	}
	if ldapConfig.Port <= 0 {
		return errors.New("LDAP port must be greater than 0")
	}

	// Save testing status
	if err := s.repository.Update(ctx, config); err != nil {
		return err
	}

	s.logger.Info("LDAP configuration test completed", zap.String("configKey", configKey))
	return nil
}

// TestSMTPConnection tests SMTP configuration connectivity
func (s *Service) TestSMTPConnection(ctx context.Context, tenantID *uuid.UUID, configKey string) error {
	config, err := s.GetConfigByKey(ctx, tenantID, configKey)
	if err != nil {
		return err
	}

	if config.ConfigType() != ConfigTypeSMTP {
		return errors.New("configuration is not SMTP type")
	}

	smtpConfig, err := config.GetSMTPConfig()
	if err != nil {
		return err
	}

	// Set testing status
	if err := config.SetTestingStatus(config.CreatedBy()); err != nil {
		return err
	}

	// TODO: Implement actual SMTP connection test
	// For now, just validate the configuration structure
	if smtpConfig.Host == "" {
		return errors.New("SMTP host is required")
	}
	if smtpConfig.Port <= 0 {
		return errors.New("SMTP port must be greater than 0")
	}

	// Save testing status
	if err := s.repository.Update(ctx, config); err != nil {
		return err
	}

	s.logger.Info("SMTP configuration test completed", zap.String("configKey", configKey))
	return nil
}

// TestExternalServiceConnection tests external service connectivity and authorization
func (s *Service) TestExternalServiceConnection(ctx context.Context, tenantID *uuid.UUID, configKey string) error {
	config, err := s.GetConfigByKey(ctx, tenantID, configKey)
	if err != nil {
		return err
	}

	if config.ConfigType() != ConfigTypeExternalServices {
		return errors.New("configuration is not external service type")
	}

	externalConfig, err := config.GetExternalServiceConfig()
	if err != nil {
		return err
	}

	// Set testing status
	if err := config.SetTestingStatus(config.CreatedBy()); err != nil {
		return err
	}

	// Validate basic configuration
	if externalConfig.URL == "" {
		return errors.New("external service URL is required")
	}
	if externalConfig.APIKey == "" && len(externalConfig.Headers) == 0 {
		return errors.New("external service API key or custom headers are required")
	}

	// Test connectivity by making a simple HEAD request to the service URL
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", externalConfig.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	// Add authentication headers
	if len(externalConfig.Headers) > 0 {
		for key, value := range externalConfig.Headers {
			req.Header.Set(key, value)
		}
	} else {
		// Default to X-API-Key header
		req.Header.Set("X-API-Key", externalConfig.APIKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to external service: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful response (2xx status codes or 404 for connectivity)
	// 404 is acceptable for connectivity testing as it proves the service is reachable
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == 404 {
			// 404 is acceptable for connectivity - service is reachable but endpoint doesn't exist
			s.logger.Info("External service connectivity test passed (404 response)",
				zap.String("configKey", configKey),
				zap.String("url", externalConfig.URL),
				zap.Int("statusCode", resp.StatusCode))
		} else {
			return fmt.Errorf("external service returned status %d: %s", resp.StatusCode, resp.Status)
		}
	}

	// Save testing status
	if err := s.repository.Update(ctx, config); err != nil {
		return err
	}

	s.logger.Info("External service connection test completed", zap.String("configKey", configKey), zap.String("url", externalConfig.URL))
	return nil
}

// GetSecurityConfig retrieves security configuration for a tenant
func (s *Service) GetSecurityConfig(ctx context.Context, tenantID uuid.UUID) (*SecurityConfig, error) {
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant ID is required")
	}

	// Get all active security configurations for the tenant
	configs, err := s.GetActiveConfigsByType(ctx, tenantID, ConfigTypeSecurity)
	if err != nil {
		s.logger.Error("Failed to get security configs", zap.Error(err), zap.String("tenantID", tenantID.String()))
		return nil, err
	}

	// Build security config from individual settings
	securityConfig := &SecurityConfig{
		JWTExpirationHours:    24,  // Default values
		RefreshTokenHours:     168, // 7 days
		MaxLoginAttempts:      5,
		AccountLockDuration:   30, // 30 minutes
		PasswordMinLength:     8,
		RequireSpecialChars:   true,
		RequireNumbers:        true,
		RequireUppercase:      true,
		SessionTimeoutMinutes: 60,
	}

	// Override defaults with configured values
	for _, config := range configs {
		var value interface{}
		if err := json.Unmarshal(config.ConfigValue(), &value); err != nil {
			s.logger.Warn("Failed to unmarshal config value", zap.Error(err), zap.String("key", config.ConfigKey()))
			continue
		}

		switch config.ConfigKey() {
		case "jwt_expiration_hours":
			if val, ok := value.(float64); ok {
				securityConfig.JWTExpirationHours = int(val)
			}
		case "refresh_token_hours":
			if val, ok := value.(float64); ok {
				securityConfig.RefreshTokenHours = int(val)
			}
		case "max_login_attempts":
			if val, ok := value.(float64); ok {
				securityConfig.MaxLoginAttempts = int(val)
			}
		case "account_lock_duration_minutes":
			if val, ok := value.(float64); ok {
				securityConfig.AccountLockDuration = int(val)
			}
		case "password_min_length":
			if val, ok := value.(float64); ok {
				securityConfig.PasswordMinLength = int(val)
			}
		case "require_special_chars":
			if val, ok := value.(bool); ok {
				securityConfig.RequireSpecialChars = val
			}
		case "require_numbers":
			if val, ok := value.(bool); ok {
				securityConfig.RequireNumbers = val
			}
		case "require_uppercase":
			if val, ok := value.(bool); ok {
				securityConfig.RequireUppercase = val
			}
		case "session_timeout_minutes":
			if val, ok := value.(float64); ok {
				securityConfig.SessionTimeoutMinutes = int(val)
			}
		}
	}

	return securityConfig, nil
}

// GetGeneralConfig retrieves general configuration for a tenant
func (s *Service) GetGeneralConfig(ctx context.Context, tenantID uuid.UUID) (*GeneralConfig, error) {
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant ID is required")
	}

	configs, err := s.GetActiveConfigsByType(ctx, tenantID, ConfigTypeGeneral)
	if err != nil {
		s.logger.Error("Failed to get general configs", zap.Error(err), zap.String("tenantID", tenantID.String()))
		return nil, err
	}

	// Merge in universal general configs (tenant-specific takes precedence).
	universalConfigs, err := s.GetConfigsByType(ctx, nil, ConfigTypeGeneral)
	if err != nil {
		s.logger.Warn("Failed to load universal general configs", zap.Error(err))
	} else if len(universalConfigs) > 0 {
		configs = append(universalConfigs, configs...)
	}

	generalConfig := &GeneralConfig{
		SystemName:              "ImageFactory",
		SystemDescription:       "Internal Image Factory Platform",
		AdminEmail:              "admin@imgfactory.com",
		SupportEmail:            "support@imgfactory.com",
		TimeZone:                "UTC",
		DateFormat:              "YYYY-MM-DD",
		DefaultLanguage:         "en",
		WorkflowPollInterval:    "3s",
		WorkflowMaxStepsPerTick: 1,
		MaintenanceMode:         false,
	}

	for _, config := range configs {
		var value interface{}
		if err := json.Unmarshal(config.ConfigValue(), &value); err != nil {
			s.logger.Warn("Failed to unmarshal config value", zap.Error(err), zap.String("key", config.ConfigKey()))
			continue
		}

		switch config.ConfigKey() {
		case "general":
			if obj, ok := value.(map[string]interface{}); ok {
				if val, ok := obj["system_name"].(string); ok {
					generalConfig.SystemName = val
				}
				if val, ok := obj["system_description"].(string); ok {
					generalConfig.SystemDescription = val
				}
				if val, ok := obj["admin_email"].(string); ok {
					generalConfig.AdminEmail = val
				}
				if val, ok := obj["support_email"].(string); ok {
					generalConfig.SupportEmail = val
				}
				if val, ok := obj["time_zone"].(string); ok {
					generalConfig.TimeZone = val
				}
				if val, ok := obj["date_format"].(string); ok {
					generalConfig.DateFormat = val
				}
				if val, ok := obj["default_language"].(string); ok {
					generalConfig.DefaultLanguage = val
				}
				if val, ok := obj["maintenance_mode"].(bool); ok {
					generalConfig.MaintenanceMode = val
				}
				if val, ok := obj["workflow_enabled"].(bool); ok {
					generalConfig.WorkflowEnabled = &val
				}
				if val, ok := obj["workflow_poll_interval"].(string); ok {
					generalConfig.WorkflowPollInterval = val
				}
				if val, ok := obj["workflow_max_steps_per_tick"].(float64); ok {
					generalConfig.WorkflowMaxStepsPerTick = int(val)
				}
			}
		case "maintenance_mode":
			if val, ok := value.(bool); ok {
				generalConfig.MaintenanceMode = val
			}
		}
	}

	return generalConfig, nil
}

// GetBuildConfig retrieves build configuration for a tenant
func (s *Service) GetBuildConfig(ctx context.Context, tenantID uuid.UUID) (*BuildConfig, error) {
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant ID is required")
	}

	// Get all active build configurations for the tenant
	configs, err := s.GetActiveConfigsByType(ctx, tenantID, ConfigTypeBuild)
	if err != nil {
		s.logger.Error("Failed to get build configs", zap.Error(err), zap.String("tenantID", tenantID.String()))
		return nil, err
	}

	// Merge in universal build configs (tenant-specific takes precedence).
	universalConfigs, err := s.GetConfigsByType(ctx, nil, ConfigTypeBuild)
	if err != nil {
		s.logger.Warn("Failed to load universal build configs", zap.Error(err))
	} else if len(universalConfigs) > 0 {
		configs = append(universalConfigs, configs...)
	}

	// Build build config from individual settings
	monitorEventDrivenEnabledDefault := true
	buildConfig := &BuildConfig{
		DefaultTimeoutMinutes:     30, // Default values
		MaxConcurrentJobs:         10,
		WorkerPoolSize:            5,
		MaxQueueSize:              100,
		ArtifactRetentionDays:     30,
		TektonEnabled:             true,
		MonitorEventDrivenEnabled: &monitorEventDrivenEnabledDefault,
		EnableTempScanStage:       true,
	}

	// Override defaults with configured values
	for _, config := range configs {
		var value interface{}
		if err := json.Unmarshal(config.ConfigValue(), &value); err != nil {
			s.logger.Warn("Failed to unmarshal config value", zap.Error(err), zap.String("key", config.ConfigKey()))
			continue
		}

		switch config.ConfigKey() {
		case "build":
			if obj, ok := value.(map[string]interface{}); ok {
				applyBuildConfigOverrides(buildConfig, obj)
			}
		case "default_timeout_minutes":
			if val, ok := value.(float64); ok {
				buildConfig.DefaultTimeoutMinutes = int(val)
			}
		case "max_concurrent_jobs":
			if val, ok := value.(float64); ok {
				buildConfig.MaxConcurrentJobs = int(val)
			}
		case "worker_pool_size":
			if val, ok := value.(float64); ok {
				buildConfig.WorkerPoolSize = int(val)
			}
		case "max_queue_size":
			if val, ok := value.(float64); ok {
				buildConfig.MaxQueueSize = int(val)
			}
		case "artifact_retention_days":
			if val, ok := value.(float64); ok {
				buildConfig.ArtifactRetentionDays = int(val)
			}
		case "tekton_enabled":
			if val, ok := value.(bool); ok {
				buildConfig.TektonEnabled = val
			}
		case "monitor_event_driven_enabled":
			if val, ok := value.(bool); ok {
				buildConfig.MonitorEventDrivenEnabled = &val
			}
		case "enable_temp_scan_stage":
			if val, ok := value.(bool); ok {
				buildConfig.EnableTempScanStage = val
			}
		}
	}

	return buildConfig, nil
}

func applyBuildConfigOverrides(buildConfig *BuildConfig, values map[string]interface{}) {
	if val, ok := values["default_timeout_minutes"].(float64); ok {
		buildConfig.DefaultTimeoutMinutes = int(val)
	}
	if val, ok := values["max_concurrent_jobs"].(float64); ok {
		buildConfig.MaxConcurrentJobs = int(val)
	}
	if val, ok := values["worker_pool_size"].(float64); ok {
		buildConfig.WorkerPoolSize = int(val)
	}
	if val, ok := values["max_queue_size"].(float64); ok {
		buildConfig.MaxQueueSize = int(val)
	}
	if val, ok := values["artifact_retention_days"].(float64); ok {
		buildConfig.ArtifactRetentionDays = int(val)
	}
	if val, ok := values["tekton_enabled"].(bool); ok {
		buildConfig.TektonEnabled = val
	}
	if val, ok := values["monitor_event_driven_enabled"].(bool); ok {
		buildConfig.MonitorEventDrivenEnabled = &val
	}
	if val, ok := values["enable_temp_scan_stage"].(bool); ok {
		buildConfig.EnableTempScanStage = val
	}
	if val, ok := values["enableTempScanStage"].(bool); ok {
		buildConfig.EnableTempScanStage = val
	}
}

func defaultTektonTaskImagesConfig() TektonTaskImagesConfig {
	return TektonTaskImagesConfig{
		GitClone:       "docker.io/alpine/git:2.45.2",
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
	}
}

func applyTektonTaskImagesDefaults(cfg *TektonTaskImagesConfig) {
	if cfg == nil {
		return
	}
	defaults := defaultTektonTaskImagesConfig()
	if strings.TrimSpace(cfg.GitClone) == "" {
		cfg.GitClone = defaults.GitClone
	}
	if strings.TrimSpace(cfg.KanikoExecutor) == "" {
		cfg.KanikoExecutor = defaults.KanikoExecutor
	}
	if strings.TrimSpace(cfg.Buildkit) == "" {
		cfg.Buildkit = defaults.Buildkit
	}
	if strings.TrimSpace(cfg.Skopeo) == "" {
		cfg.Skopeo = defaults.Skopeo
	}
	if strings.TrimSpace(cfg.Trivy) == "" {
		cfg.Trivy = defaults.Trivy
	}
	if strings.TrimSpace(cfg.Syft) == "" {
		cfg.Syft = defaults.Syft
	}
	if strings.TrimSpace(cfg.Cosign) == "" {
		cfg.Cosign = defaults.Cosign
	}
	if strings.TrimSpace(cfg.Packer) == "" {
		cfg.Packer = defaults.Packer
	}
	if strings.TrimSpace(cfg.PythonAlpine) == "" {
		cfg.PythonAlpine = defaults.PythonAlpine
	}
	if strings.TrimSpace(cfg.Alpine) == "" {
		cfg.Alpine = defaults.Alpine
	}
	if strings.TrimSpace(cfg.CleanupKubectl) == "" {
		cfg.CleanupKubectl = defaults.CleanupKubectl
	}
}

func (s *Service) GetTektonTaskImagesConfig(ctx context.Context) (*TektonTaskImagesConfig, error) {
	config, err := s.repository.FindByKey(ctx, nil, "tekton_task_images")
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) || strings.Contains(err.Error(), "no rows in result set") {
			defaultCfg := defaultTektonTaskImagesConfig()
			return &defaultCfg, nil
		}
		s.logger.Error("Failed to get tekton task images config", zap.Error(err))
		return nil, err
	}
	if config.Status() != ConfigStatusActive {
		return nil, fmt.Errorf("tekton task images configuration is not active")
	}

	var cfg TektonTaskImagesConfig
	if err := json.Unmarshal(config.ConfigValue(), &cfg); err != nil {
		s.logger.Error("Failed to unmarshal tekton task images config", zap.Error(err))
		return nil, err
	}
	applyTektonTaskImagesDefaults(&cfg)
	normalizeLegacyTektonTaskImageRefs(&cfg)
	return &cfg, nil
}

func (s *Service) UpdateTektonTaskImagesConfig(ctx context.Context, cfg *TektonTaskImagesConfig, updatedBy uuid.UUID) (*TektonTaskImagesConfig, error) {
	if cfg == nil {
		return nil, errors.New("tekton task images config is required")
	}
	if updatedBy == uuid.Nil {
		return nil, errors.New("updated by user ID is required")
	}
	applyTektonTaskImagesDefaults(cfg)
	if err := s.validateTektonTaskImagesConfig(cfg); err != nil {
		s.logger.Error("Tekton task images config validation failed", zap.Error(err))
		return nil, err
	}

	if _, err := s.CreateOrUpdateCategoryConfig(ctx, nil, ConfigTypeTekton, "tekton_task_images", cfg, updatedBy); err != nil {
		s.logger.Error("Failed to save tekton task images config", zap.Error(err))
		return nil, err
	}
	s.logger.Info("Tekton task images configuration updated", zap.String("updatedBy", updatedBy.String()))
	return cfg, nil
}

// GetToolAvailabilityConfig retrieves tool availability configuration for a tenant
func (s *Service) GetToolAvailabilityConfig(ctx context.Context, tenantID *uuid.UUID) (*ToolAvailabilityConfig, error) {
	// Get tool settings configuration (global if tenantID is nil)
	config, err := s.repository.FindByKey(ctx, tenantID, "tool_availability")
	if err != nil {
		// Check for config not found error using errors.Is for proper error wrapping handling
		if errors.Is(err, ErrConfigNotFound) || strings.Contains(err.Error(), "no rows in result set") {
			// For tenant scope, fall back to global tool availability if tenant-specific
			// config has not been created yet.
			if tenantID != nil {
				s.logger.Info("Tenant tool availability config not found, using global default",
					zap.String("tenantID", tenantID.String()))
				config, err = s.repository.FindByKey(ctx, nil, "tool_availability")
				if err == nil {
					goto parseConfig
				}
			}
			// Global config should be seeded in the database.
			s.logger.Error("Global tool availability config not found in database - this should be seeded")
			return nil, fmt.Errorf("tool availability configuration not found")
		}
		if tenantID != nil {
			s.logger.Error("Failed to get tool availability config", zap.Error(err), zap.String("tenantID", tenantID.String()))
		} else {
			s.logger.Error("Failed to get global tool availability config", zap.Error(err))
		}
		return nil, err
	}

	if config.Status() != ConfigStatusActive {
		// Tool availability config should be active
		if tenantID != nil {
			s.logger.Error("Tool availability config is not active", zap.String("tenantID", tenantID.String()), zap.String("status", string(config.Status())))
		} else {
			s.logger.Error("Global tool availability config is not active", zap.String("status", string(config.Status())))
		}
		return nil, fmt.Errorf("tool availability configuration is not active")
	}

parseConfig:
	var toolConfig ToolAvailabilityConfig
	if err := json.Unmarshal(config.ConfigValue(), &toolConfig); err != nil {
		s.logger.Error("Failed to unmarshal tool availability config", zap.Error(err))
		return nil, err
	}
	applyToolAvailabilityDefaults(config.ConfigValue(), &toolConfig, config.TenantID() != nil)

	return &toolConfig, nil
}

func applyToolAvailabilityDefaults(raw json.RawMessage, cfg *ToolAvailabilityConfig, tenantScoped bool) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.TrivyRuntime.CacheMode) == "" {
		cfg.TrivyRuntime.CacheMode = "shared"
	}
	if strings.TrimSpace(cfg.TrivyRuntime.DBRepository) == "" {
		cfg.TrivyRuntime.DBRepository = "image-factory-registry:5000/security/trivy-db:2,mirror.gcr.io/aquasec/trivy-db:2"
	}
	if strings.TrimSpace(cfg.TrivyRuntime.JavaDBRepository) == "" {
		cfg.TrivyRuntime.JavaDBRepository = "image-factory-registry:5000/security/trivy-java-db:1,mirror.gcr.io/aquasec/trivy-java-db:1"
	}
	// Strict tenant override semantics: omitted build method keys remain false.
	if tenantScoped {
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return
	}

	buildMethods, _ := payload["build_methods"].(map[string]interface{})
	if _, ok := buildMethods["container"]; !ok {
		cfg.BuildMethods.Container = true
	}
	if _, ok := buildMethods["nix"]; !ok {
		cfg.BuildMethods.Nix = true
	}
}

// GetBuildCapabilitiesConfig retrieves build capability entitlements for a tenant.
// Tenant scope falls back to global scope when no tenant override exists.
func (s *Service) GetBuildCapabilitiesConfig(ctx context.Context, tenantID *uuid.UUID) (*BuildCapabilitiesConfig, error) {
	config, err := s.repository.FindByKey(ctx, tenantID, "build_capabilities")
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) || strings.Contains(err.Error(), "no rows in result set") {
			if tenantID != nil {
				s.logger.Info("Tenant build capabilities config not found, using global default",
					zap.String("tenantID", tenantID.String()))
				config, err = s.repository.FindByKey(ctx, nil, "build_capabilities")
				if err == nil {
					goto parseConfig
				}
			}
			defaultCfg := defaultBuildCapabilitiesConfig()
			return &defaultCfg, nil
		}
		if tenantID != nil {
			s.logger.Error("Failed to get build capabilities config", zap.Error(err), zap.String("tenantID", tenantID.String()))
		} else {
			s.logger.Error("Failed to get global build capabilities config", zap.Error(err))
		}
		return nil, err
	}

	if config.Status() != ConfigStatusActive {
		if tenantID != nil {
			s.logger.Error("Build capabilities config is not active", zap.String("tenantID", tenantID.String()), zap.String("status", string(config.Status())))
		} else {
			s.logger.Error("Global build capabilities config is not active", zap.String("status", string(config.Status())))
		}
		return nil, fmt.Errorf("build capabilities configuration is not active")
	}

parseConfig:
	var buildCapabilities BuildCapabilitiesConfig
	if err := json.Unmarshal(config.ConfigValue(), &buildCapabilities); err != nil {
		s.logger.Error("Failed to unmarshal build capabilities config", zap.Error(err))
		return nil, err
	}
	applyBuildCapabilitiesDefaults(config.ConfigValue(), &buildCapabilities, config.TenantID() != nil)
	return &buildCapabilities, nil
}

func defaultBuildCapabilitiesConfig() BuildCapabilitiesConfig {
	return BuildCapabilitiesConfig{
		GPU:            true,
		Privileged:     true,
		MultiArch:      true,
		HighMemory:     true,
		HostNetworking: true,
		Premium:        true,
	}
}

func applyBuildCapabilitiesDefaults(raw json.RawMessage, cfg *BuildCapabilitiesConfig, tenantScoped bool) {
	if cfg == nil || tenantScoped {
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return
	}
	if _, ok := payload["gpu"]; !ok {
		cfg.GPU = true
	}
	if _, ok := payload["privileged"]; !ok {
		cfg.Privileged = true
	}
	if _, ok := payload["multi_arch"]; !ok {
		cfg.MultiArch = true
	}
	if _, ok := payload["high_memory"]; !ok {
		cfg.HighMemory = true
	}
	if _, ok := payload["host_networking"]; !ok {
		cfg.HostNetworking = true
	}
	if _, ok := payload["premium"]; !ok {
		cfg.Premium = true
	}
}

// UpdateToolAvailabilityConfig updates tool availability configuration for a tenant
func (s *Service) UpdateToolAvailabilityConfig(ctx context.Context, tenantID *uuid.UUID, toolConfig *ToolAvailabilityConfig, updatedBy uuid.UUID) (*ToolAvailabilityConfig, error) {
	if toolConfig == nil {
		return nil, errors.New("tool config is required")
	}
	if updatedBy == uuid.Nil {
		return nil, errors.New("updated by user ID is required")
	}

	// Validate the tool configuration
	if err := s.validateToolAvailabilityConfig(toolConfig); err != nil {
		s.logger.Error("Tool availability config validation failed", zap.Error(err))
		return nil, err
	}

	// Create or update the configuration
	_, err := s.CreateOrUpdateCategoryConfig(ctx, tenantID, ConfigTypeToolSettings, "tool_availability", toolConfig, updatedBy)
	if err != nil {
		s.logger.Error("Failed to save tool availability config", zap.Error(err))
		return nil, err
	}

	if tenantID != nil {
		s.logger.Info("Tool availability configuration updated",
			zap.String("tenantID", tenantID.String()),
			zap.String("updatedBy", updatedBy.String()))
	} else {
		s.logger.Info("Global tool availability configuration updated",
			zap.String("updatedBy", updatedBy.String()))
	}

	return toolConfig, nil
}

// UpdateBuildCapabilitiesConfig updates build capability entitlements for a tenant.
func (s *Service) UpdateBuildCapabilitiesConfig(ctx context.Context, tenantID *uuid.UUID, buildCapabilities *BuildCapabilitiesConfig, updatedBy uuid.UUID) (*BuildCapabilitiesConfig, error) {
	if buildCapabilities == nil {
		return nil, errors.New("build capabilities config is required")
	}
	if updatedBy == uuid.Nil {
		return nil, errors.New("updated by user ID is required")
	}

	_, err := s.CreateOrUpdateCategoryConfig(ctx, tenantID, ConfigTypeToolSettings, "build_capabilities", buildCapabilities, updatedBy)
	if err != nil {
		s.logger.Error("Failed to save build capabilities config", zap.Error(err))
		return nil, err
	}

	if tenantID != nil {
		s.logger.Info("Build capabilities configuration updated",
			zap.String("tenantID", tenantID.String()),
			zap.String("updatedBy", updatedBy.String()))
	} else {
		s.logger.Info("Global build capabilities configuration updated",
			zap.String("updatedBy", updatedBy.String()))
	}

	return buildCapabilities, nil
}

// GetOperationCapabilitiesConfig retrieves operation capability entitlements for a tenant.
// Tenant scope falls back to global scope when no tenant override exists.
func (s *Service) GetOperationCapabilitiesConfig(ctx context.Context, tenantID *uuid.UUID) (*OperationCapabilitiesConfig, error) {
	config, err := s.repository.FindByKey(ctx, tenantID, "operation_capabilities")
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) || strings.Contains(err.Error(), "no rows in result set") {
			if tenantID != nil {
				s.logger.Info("Tenant operation capabilities config not found, using global default",
					zap.String("tenantID", tenantID.String()))
				config, err = s.repository.FindByKey(ctx, nil, "operation_capabilities")
				if err == nil {
					goto parseConfig
				}
			}
			defaultCfg := defaultOperationCapabilitiesConfig()
			return &defaultCfg, nil
		}
		if tenantID != nil {
			s.logger.Error("Failed to get operation capabilities config", zap.Error(err), zap.String("tenantID", tenantID.String()))
		} else {
			s.logger.Error("Failed to get global operation capabilities config", zap.Error(err))
		}
		return nil, err
	}

	if config.Status() != ConfigStatusActive {
		if tenantID != nil {
			s.logger.Error("Operation capabilities config is not active", zap.String("tenantID", tenantID.String()), zap.String("status", string(config.Status())))
		} else {
			s.logger.Error("Global operation capabilities config is not active", zap.String("status", string(config.Status())))
		}
		return nil, fmt.Errorf("operation capabilities configuration is not active")
	}

parseConfig:
	var operationCapabilities OperationCapabilitiesConfig
	if err := json.Unmarshal(config.ConfigValue(), &operationCapabilities); err != nil {
		s.logger.Error("Failed to unmarshal operation capabilities config", zap.Error(err))
		return nil, err
	}
	applyOperationCapabilitiesDefaults(config.ConfigValue(), &operationCapabilities, config.TenantID() != nil)
	return &operationCapabilities, nil
}

func defaultOperationCapabilitiesConfig() OperationCapabilitiesConfig {
	// Secure default: fail closed for sensitive operations.
	return OperationCapabilitiesConfig{
		Build:             false,
		QuarantineRequest: false,
		QuarantineRelease: false,
		OnDemandImageScan: false,
	}
}

func applyOperationCapabilitiesDefaults(_ json.RawMessage, cfg *OperationCapabilitiesConfig, _ bool) {
	if cfg == nil {
		return
	}
}

func capabilityDeniedMessage(label string) string {
	return fmt.Sprintf("This tenant is not entitled for %s capability.", label)
}

// GetCapabilitySurfaces returns tenant-effective capability flags and a
// derived backend-driven surface contract for nav/routes/actions.
func (s *Service) GetCapabilitySurfaces(ctx context.Context, tenantID uuid.UUID) (*CapabilitySurfacesResponse, error) {
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant ID is required")
	}

	cfg, err := s.GetOperationCapabilitiesConfig(ctx, &tenantID)
	if err != nil {
		return nil, err
	}

	surfaces := CapabilitySurfaceSet{
		NavKeys: []string{
			"dashboard",
			"notifications",
			"images",
		},
		RouteKeys: []string{
			"dashboard.view",
			"images.list",
			"images.detail",
			"profile.view",
			"notifications.view",
		},
		ActionKeys: []string{
			"images.view_catalog",
		},
	}

	denials := map[string]CapabilitySurfaceDenial{}

	if cfg.Build {
		surfaces.NavKeys = append(surfaces.NavKeys, "projects", "builds")
		surfaces.RouteKeys = append(surfaces.RouteKeys,
			"projects.list",
			"projects.create",
			"projects.detail",
			"projects.edit",
			"builds.list",
			"builds.create",
			"builds.detail",
		)
		surfaces.ActionKeys = append(surfaces.ActionKeys,
			"builds.create",
			"projects.manage",
		)
	} else {
		for _, deniedKey := range []string{
			"projects.list",
			"projects.create",
			"projects.detail",
			"projects.edit",
			"builds.list",
			"builds.create",
			"builds.detail",
		} {
			denials[deniedKey] = CapabilitySurfaceDenial{
				ReasonCode: "tenant_capability_not_entitled",
				Capability: "build",
				Message:    capabilityDeniedMessage("Image Build"),
			}
		}
	}

	if cfg.Build || cfg.QuarantineRequest {
		surfaces.NavKeys = append(surfaces.NavKeys, "auth_management")
		surfaces.RouteKeys = append(surfaces.RouteKeys, "settings.auth")
		surfaces.ActionKeys = append(surfaces.ActionKeys, "settings.auth.manage")
	} else {
		denials["settings.auth"] = CapabilitySurfaceDenial{
			ReasonCode: "tenant_capability_not_entitled",
			Capability: "quarantine_request",
			Message:    capabilityDeniedMessage("Registry Auth"),
		}
	}

	if cfg.QuarantineRequest {
		surfaces.NavKeys = append(surfaces.NavKeys, "quarantine_requests")
		surfaces.RouteKeys = append(surfaces.RouteKeys, "quarantine.request.list", "quarantine.request.create")
		surfaces.ActionKeys = append(surfaces.ActionKeys, "quarantine.request.create")
	} else {
		for _, deniedKey := range []string{
			"quarantine.request.list",
			"quarantine.request.create",
		} {
			denials[deniedKey] = CapabilitySurfaceDenial{
				ReasonCode: "tenant_capability_not_entitled",
				Capability: "quarantine_request",
				Message:    capabilityDeniedMessage("Quarantine Request"),
			}
		}
	}

	if cfg.QuarantineRelease {
		surfaces.RouteKeys = append(surfaces.RouteKeys, "quarantine.release")
		surfaces.ActionKeys = append(surfaces.ActionKeys, "quarantine.release")
	} else {
		denials["quarantine.release"] = CapabilitySurfaceDenial{
			ReasonCode: "tenant_capability_not_entitled",
			Capability: "quarantine_release",
			Message:    capabilityDeniedMessage("Quarantine Release"),
		}
	}

	if cfg.OnDemandImageScan {
		surfaces.RouteKeys = append(surfaces.RouteKeys, "images.scan.ondemand")
		surfaces.ActionKeys = append(surfaces.ActionKeys, "images.scan.ondemand")
	} else {
		denials["images.scan.ondemand"] = CapabilitySurfaceDenial{
			ReasonCode: "tenant_capability_not_entitled",
			Capability: "ondemand_image_scanning",
			Message:    capabilityDeniedMessage("On-Demand Image Scanning"),
		}
	}

	sort.Strings(surfaces.NavKeys)
	sort.Strings(surfaces.RouteKeys)
	sort.Strings(surfaces.ActionKeys)

	return &CapabilitySurfacesResponse{
		TenantID:     tenantID,
		Version:      "2026-02-20",
		Capabilities: *cfg,
		Surfaces:     surfaces,
		Denials:      denials,
	}, nil
}

func defaultQuarantinePolicyConfig() QuarantinePolicyConfig {
	return QuarantinePolicyConfig{
		Enabled:     true,
		Mode:        "dry_run",
		MaxCritical: 0,
		MaxP2:       0,
		MaxP3:       0,
		MaxCVSS:     0,
		SeverityMapping: QuarantinePolicySeverityMapping{
			P1: []string{"critical"},
			P2: []string{"high"},
			P3: []string{"medium"},
			P4: []string{"low", "unknown"},
		},
	}
}

func applyQuarantinePolicyDefaults(raw json.RawMessage, cfg *QuarantinePolicyConfig, tenantScoped bool) {
	if cfg == nil {
		return
	}

	defaultCfg := defaultQuarantinePolicyConfig()
	if strings.TrimSpace(cfg.Mode) == "" {
		cfg.Mode = defaultCfg.Mode
	}
	if len(cfg.SeverityMapping.P1) == 0 {
		cfg.SeverityMapping.P1 = defaultCfg.SeverityMapping.P1
	}
	if len(cfg.SeverityMapping.P2) == 0 {
		cfg.SeverityMapping.P2 = defaultCfg.SeverityMapping.P2
	}
	if len(cfg.SeverityMapping.P3) == 0 {
		cfg.SeverityMapping.P3 = defaultCfg.SeverityMapping.P3
	}
	if len(cfg.SeverityMapping.P4) == 0 {
		cfg.SeverityMapping.P4 = defaultCfg.SeverityMapping.P4
	}

	// Preserve secure fallback semantics for legacy/global payloads with missing keys.
	if !tenantScoped && len(raw) == 0 {
		*cfg = defaultCfg
	}
}

func validateQuarantinePolicyConfig(policy *QuarantinePolicyConfig) error {
	if policy == nil {
		return errors.New("quarantine policy config is required")
	}

	mode := strings.ToLower(strings.TrimSpace(policy.Mode))
	if mode != "enforce" && mode != "dry_run" {
		return fmt.Errorf("quarantine policy mode must be one of: enforce, dry_run")
	}
	if policy.MaxCritical < 0 {
		return fmt.Errorf("max_critical must be >= 0")
	}
	if policy.MaxP2 < 0 {
		return fmt.Errorf("max_p2 must be >= 0")
	}
	if policy.MaxP3 < 0 {
		return fmt.Errorf("max_p3 must be >= 0")
	}
	if policy.MaxCVSS < 0 || policy.MaxCVSS > 10 {
		return fmt.Errorf("max_cvss must be between 0 and 10")
	}

	allowedSeverity := map[string]struct{}{
		"critical": {},
		"high":     {},
		"medium":   {},
		"low":      {},
		"unknown":  {},
	}

	seen := map[string]string{}
	checkBucket := func(bucket string, values []string) error {
		if len(values) == 0 {
			return fmt.Errorf("severity_mapping.%s must not be empty", bucket)
		}
		for _, value := range values {
			normalized := strings.ToLower(strings.TrimSpace(value))
			if _, ok := allowedSeverity[normalized]; !ok {
				return fmt.Errorf("severity_mapping.%s contains invalid severity %q", bucket, value)
			}
			if prevBucket, exists := seen[normalized]; exists && prevBucket != bucket {
				return fmt.Errorf("severity_mapping duplicates severity %q across %s and %s", normalized, prevBucket, bucket)
			}
			seen[normalized] = bucket
		}
		return nil
	}

	if err := checkBucket("p1", policy.SeverityMapping.P1); err != nil {
		return err
	}
	if err := checkBucket("p2", policy.SeverityMapping.P2); err != nil {
		return err
	}
	if err := checkBucket("p3", policy.SeverityMapping.P3); err != nil {
		return err
	}
	if err := checkBucket("p4", policy.SeverityMapping.P4); err != nil {
		return err
	}

	policy.Mode = mode
	return nil
}

func defaultSORRegistrationConfig() SORRegistrationConfig {
	return SORRegistrationConfig{
		Enforce:          true,
		RuntimeErrorMode: "error",
	}
}

func defaultReleaseGovernancePolicyConfig() ReleaseGovernancePolicyConfig {
	return ReleaseGovernancePolicyConfig{
		Enabled:                      true,
		FailureRatioThreshold:        0.2,
		ConsecutiveFailuresThreshold: 3,
		MinimumSamples:               10,
		WindowMinutes:                60,
	}
}

func applySORRegistrationDefaults(cfg *SORRegistrationConfig) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.RuntimeErrorMode) == "" {
		cfg.RuntimeErrorMode = "error"
	}
}

func validateSORRegistrationConfig(cfg *SORRegistrationConfig) error {
	if cfg == nil {
		return errors.New("epr registration config is required")
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.RuntimeErrorMode))
	switch mode {
	case "error", "deny", "allow":
	default:
		return fmt.Errorf("runtime_error_mode must be one of: error, deny, allow")
	}

	cfg.RuntimeErrorMode = mode
	return nil
}

func applyReleaseGovernancePolicyDefaults(cfg *ReleaseGovernancePolicyConfig) {
	if cfg == nil {
		return
	}
	defaultCfg := defaultReleaseGovernancePolicyConfig()
	if cfg.FailureRatioThreshold == 0 {
		cfg.FailureRatioThreshold = defaultCfg.FailureRatioThreshold
	}
	if cfg.ConsecutiveFailuresThreshold == 0 {
		cfg.ConsecutiveFailuresThreshold = defaultCfg.ConsecutiveFailuresThreshold
	}
	if cfg.MinimumSamples == 0 {
		cfg.MinimumSamples = defaultCfg.MinimumSamples
	}
	if cfg.WindowMinutes == 0 {
		cfg.WindowMinutes = defaultCfg.WindowMinutes
	}
}

func validateReleaseGovernancePolicyConfig(cfg *ReleaseGovernancePolicyConfig) error {
	if cfg == nil {
		return errors.New("release governance policy config is required")
	}
	if cfg.FailureRatioThreshold < 0 || cfg.FailureRatioThreshold > 1 {
		return fmt.Errorf("failure_ratio_threshold must be between 0 and 1")
	}
	if cfg.ConsecutiveFailuresThreshold < 1 {
		return fmt.Errorf("consecutive_failures_threshold must be >= 1")
	}
	if cfg.MinimumSamples < 1 {
		return fmt.Errorf("minimum_samples must be >= 1")
	}
	if cfg.WindowMinutes < 1 {
		return fmt.Errorf("window_minutes must be >= 1")
	}
	return nil
}

// UpdateOperationCapabilitiesConfig updates operation capability entitlements for a tenant.
func (s *Service) UpdateOperationCapabilitiesConfig(ctx context.Context, tenantID *uuid.UUID, operationCapabilities *OperationCapabilitiesConfig, updatedBy uuid.UUID) (*OperationCapabilitiesConfig, error) {
	if operationCapabilities == nil {
		return nil, errors.New("operation capabilities config is required")
	}
	if updatedBy == uuid.Nil {
		return nil, errors.New("updated by user ID is required")
	}

	_, err := s.CreateOrUpdateCategoryConfig(ctx, tenantID, ConfigTypeToolSettings, "operation_capabilities", operationCapabilities, updatedBy)
	if err != nil {
		s.logger.Error("Failed to save operation capabilities config", zap.Error(err))
		return nil, err
	}

	if tenantID != nil {
		s.logger.Info("Operation capabilities configuration updated",
			zap.String("tenantID", tenantID.String()),
			zap.String("updatedBy", updatedBy.String()))
	} else {
		s.logger.Info("Global operation capabilities configuration updated",
			zap.String("updatedBy", updatedBy.String()))
	}

	return operationCapabilities, nil
}

// GetQuarantinePolicyConfig retrieves quarantine policy configuration for a tenant.
// Tenant scope falls back to global scope when no tenant override exists.
func (s *Service) GetQuarantinePolicyConfig(ctx context.Context, tenantID *uuid.UUID) (*QuarantinePolicyConfig, error) {
	config, err := s.repository.FindByKey(ctx, tenantID, "quarantine_policy")
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) || strings.Contains(err.Error(), "no rows in result set") {
			if tenantID != nil {
				s.logger.Info("Tenant quarantine policy config not found, using global default",
					zap.String("tenantID", tenantID.String()))
				config, err = s.repository.FindByKey(ctx, nil, "quarantine_policy")
				if err == nil {
					goto parseConfig
				}
			}
			defaultCfg := defaultQuarantinePolicyConfig()
			return &defaultCfg, nil
		}
		if tenantID != nil {
			s.logger.Error("Failed to get quarantine policy config", zap.Error(err), zap.String("tenantID", tenantID.String()))
		} else {
			s.logger.Error("Failed to get global quarantine policy config", zap.Error(err))
		}
		return nil, err
	}

	if config.Status() != ConfigStatusActive {
		if tenantID != nil {
			s.logger.Error("Quarantine policy config is not active", zap.String("tenantID", tenantID.String()), zap.String("status", string(config.Status())))
		} else {
			s.logger.Error("Global quarantine policy config is not active", zap.String("status", string(config.Status())))
		}
		return nil, fmt.Errorf("quarantine policy configuration is not active")
	}

parseConfig:
	var policy QuarantinePolicyConfig
	if err := json.Unmarshal(config.ConfigValue(), &policy); err != nil {
		s.logger.Error("Failed to unmarshal quarantine policy config", zap.Error(err))
		return nil, err
	}
	applyQuarantinePolicyDefaults(config.ConfigValue(), &policy, config.TenantID() != nil)
	if err := validateQuarantinePolicyConfig(&policy); err != nil {
		s.logger.Error("Invalid quarantine policy config persisted", zap.Error(err))
		return nil, err
	}
	return &policy, nil
}

// UpdateQuarantinePolicyConfig updates quarantine policy configuration for a tenant/global scope.
func (s *Service) UpdateQuarantinePolicyConfig(ctx context.Context, tenantID *uuid.UUID, policy *QuarantinePolicyConfig, updatedBy uuid.UUID) (*QuarantinePolicyConfig, error) {
	if updatedBy == uuid.Nil {
		return nil, errors.New("updated by user ID is required")
	}
	if err := validateQuarantinePolicyConfig(policy); err != nil {
		return nil, err
	}

	_, err := s.CreateOrUpdateCategoryConfig(ctx, tenantID, ConfigTypeToolSettings, "quarantine_policy", policy, updatedBy)
	if err != nil {
		s.logger.Error("Failed to save quarantine policy config", zap.Error(err))
		return nil, err
	}

	if tenantID != nil {
		s.logger.Info("Quarantine policy configuration updated",
			zap.String("tenantID", tenantID.String()),
			zap.String("updatedBy", updatedBy.String()))
	} else {
		s.logger.Info("Global quarantine policy configuration updated",
			zap.String("updatedBy", updatedBy.String()))
	}

	return policy, nil
}

type QuarantinePolicyValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

type QuarantinePolicySimulationResult struct {
	Decision string   `json:"decision"`
	Mode     string   `json:"mode"`
	Reasons  []string `json:"reasons"`
}

// ValidateQuarantinePolicy validates quarantine policy payload without persisting it.
func (s *Service) ValidateQuarantinePolicy(policy *QuarantinePolicyConfig) *QuarantinePolicyValidationResult {
	result := &QuarantinePolicyValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}
	if err := validateQuarantinePolicyConfig(policy); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
	}
	return result
}

// SimulateQuarantinePolicy evaluates a policy against scan-summary style payload.
func (s *Service) SimulateQuarantinePolicy(policy *QuarantinePolicyConfig, scanSummary map[string]interface{}) (*QuarantinePolicySimulationResult, error) {
	if err := validateQuarantinePolicyConfig(policy); err != nil {
		return nil, err
	}

	critical, high, medium, maxCVSS := extractSimulatedVulnerabilityCounts(scanSummary)
	reasons := make([]string, 0)
	if critical > policy.MaxCritical {
		reasons = append(reasons, fmt.Sprintf("critical_count(%d) > max_critical(%d)", critical, policy.MaxCritical))
	}
	if high > policy.MaxP2 {
		reasons = append(reasons, fmt.Sprintf("high_count(%d) > max_p2(%d)", high, policy.MaxP2))
	}
	if medium > policy.MaxP3 {
		reasons = append(reasons, fmt.Sprintf("medium_count(%d) > max_p3(%d)", medium, policy.MaxP3))
	}
	if policy.MaxCVSS > 0 && maxCVSS > policy.MaxCVSS {
		reasons = append(reasons, fmt.Sprintf("max_cvss(%.1f) > threshold(%.1f)", maxCVSS, policy.MaxCVSS))
	}

	decision := "pass"
	if len(reasons) > 0 && strings.EqualFold(strings.TrimSpace(policy.Mode), "enforce") {
		decision = "quarantine"
	}

	return &QuarantinePolicySimulationResult{
		Decision: decision,
		Mode:     strings.ToLower(strings.TrimSpace(policy.Mode)),
		Reasons:  reasons,
	}, nil
}

func extractSimulatedVulnerabilityCounts(scanSummary map[string]interface{}) (critical int, high int, medium int, maxCVSS float64) {
	if scanSummary == nil {
		return 0, 0, 0, 0
	}
	if vulnMap, ok := scanSummary["vulnerabilities"].(map[string]interface{}); ok {
		critical = mapToInt(vulnMap["critical"])
		high = mapToInt(vulnMap["high"])
		medium = mapToInt(vulnMap["medium"])
	}
	if critical == 0 {
		critical = mapToInt(scanSummary["critical_count"])
	}
	if high == 0 {
		high = mapToInt(scanSummary["high_count"])
	}
	if medium == 0 {
		medium = mapToInt(scanSummary["medium_count"])
	}
	maxCVSS = mapToFloat(scanSummary["max_cvss"])
	if maxCVSS == 0 {
		maxCVSS = mapToFloat(scanSummary["maxCvss"])
	}
	return critical, high, medium, maxCVSS
}

func mapToInt(raw interface{}) int {
	switch typed := raw.(type) {
	case float64:
		return int(math.Round(typed))
	case float32:
		return int(math.Round(float64(typed)))
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	default:
		return 0
	}
}

func mapToFloat(raw interface{}) float64 {
	switch typed := raw.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

// GetSORRegistrationConfig retrieves EPR registration prerequisite configuration for a tenant.
// Tenant scope falls back to global scope when no tenant override exists.
func (s *Service) GetSORRegistrationConfig(ctx context.Context, tenantID *uuid.UUID) (*SORRegistrationConfig, error) {
	config, err := s.repository.FindByKey(ctx, tenantID, "sor_registration")
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) || strings.Contains(err.Error(), "no rows in result set") {
			if tenantID != nil {
				s.logger.Info("Tenant EPR registration config not found, using global default",
					zap.String("tenantID", tenantID.String()))
				config, err = s.repository.FindByKey(ctx, nil, "sor_registration")
				if err == nil {
					goto parseConfig
				}
			}
			defaultCfg := defaultSORRegistrationConfig()
			return &defaultCfg, nil
		}
		if tenantID != nil {
			s.logger.Error("Failed to get EPR registration config", zap.Error(err), zap.String("tenantID", tenantID.String()))
		} else {
			s.logger.Error("Failed to get global EPR registration config", zap.Error(err))
		}
		return nil, err
	}

	if config.Status() != ConfigStatusActive {
		if tenantID != nil {
			s.logger.Error("EPR registration config is not active", zap.String("tenantID", tenantID.String()), zap.String("status", string(config.Status())))
		} else {
			s.logger.Error("Global EPR registration config is not active", zap.String("status", string(config.Status())))
		}
		return nil, fmt.Errorf("epr registration configuration is not active")
	}

parseConfig:
	var sorConfig SORRegistrationConfig
	if err := json.Unmarshal(config.ConfigValue(), &sorConfig); err != nil {
		s.logger.Error("Failed to unmarshal EPR registration config", zap.Error(err))
		return nil, err
	}
	applySORRegistrationDefaults(&sorConfig)
	if err := validateSORRegistrationConfig(&sorConfig); err != nil {
		s.logger.Error("Invalid EPR registration config persisted", zap.Error(err))
		return nil, err
	}
	return &sorConfig, nil
}

// UpdateSORRegistrationConfig updates EPR registration prerequisite configuration for tenant/global scope.
func (s *Service) UpdateSORRegistrationConfig(ctx context.Context, tenantID *uuid.UUID, cfg *SORRegistrationConfig, updatedBy uuid.UUID) (*SORRegistrationConfig, error) {
	if updatedBy == uuid.Nil {
		return nil, errors.New("updated by user ID is required")
	}
	applySORRegistrationDefaults(cfg)
	if err := validateSORRegistrationConfig(cfg); err != nil {
		return nil, err
	}

	_, err := s.CreateOrUpdateCategoryConfig(ctx, tenantID, ConfigTypeToolSettings, "sor_registration", cfg, updatedBy)
	if err != nil {
		s.logger.Error("Failed to save EPR registration config", zap.Error(err))
		return nil, err
	}

	if tenantID != nil {
		s.logger.Info("EPR registration configuration updated",
			zap.String("tenantID", tenantID.String()),
			zap.String("updatedBy", updatedBy.String()))
	} else {
		s.logger.Info("Global EPR registration configuration updated",
			zap.String("updatedBy", updatedBy.String()))
	}

	return cfg, nil
}

// GetReleaseGovernancePolicyConfig retrieves release-governance threshold configuration.
// Tenant scope falls back to global scope when no tenant override exists.
func (s *Service) GetReleaseGovernancePolicyConfig(ctx context.Context, tenantID *uuid.UUID) (*ReleaseGovernancePolicyConfig, error) {
	config, err := s.repository.FindByKey(ctx, tenantID, "release_governance_policy")
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) || strings.Contains(err.Error(), "no rows in result set") {
			if tenantID != nil {
				s.logger.Info("Tenant release governance policy config not found, using global default",
					zap.String("tenantID", tenantID.String()))
				config, err = s.repository.FindByKey(ctx, nil, "release_governance_policy")
				if err == nil {
					goto parseConfig
				}
			}
			defaultCfg := defaultReleaseGovernancePolicyConfig()
			return &defaultCfg, nil
		}
		if tenantID != nil {
			s.logger.Error("Failed to get release governance policy config", zap.Error(err), zap.String("tenantID", tenantID.String()))
		} else {
			s.logger.Error("Failed to get global release governance policy config", zap.Error(err))
		}
		return nil, err
	}

	if config.Status() != ConfigStatusActive {
		if tenantID != nil {
			s.logger.Error("Release governance policy config is not active", zap.String("tenantID", tenantID.String()), zap.String("status", string(config.Status())))
		} else {
			s.logger.Error("Global release governance policy config is not active", zap.String("status", string(config.Status())))
		}
		return nil, fmt.Errorf("release governance policy configuration is not active")
	}

parseConfig:
	var policyConfig ReleaseGovernancePolicyConfig
	if err := json.Unmarshal(config.ConfigValue(), &policyConfig); err != nil {
		s.logger.Error("Failed to unmarshal release governance policy config", zap.Error(err))
		return nil, err
	}
	applyReleaseGovernancePolicyDefaults(&policyConfig)
	if err := validateReleaseGovernancePolicyConfig(&policyConfig); err != nil {
		s.logger.Error("Invalid release governance policy config persisted", zap.Error(err))
		return nil, err
	}
	return &policyConfig, nil
}

// UpdateReleaseGovernancePolicyConfig updates release-governance threshold configuration.
func (s *Service) UpdateReleaseGovernancePolicyConfig(ctx context.Context, tenantID *uuid.UUID, cfg *ReleaseGovernancePolicyConfig, updatedBy uuid.UUID) (*ReleaseGovernancePolicyConfig, error) {
	if updatedBy == uuid.Nil {
		return nil, errors.New("updated by user ID is required")
	}
	applyReleaseGovernancePolicyDefaults(cfg)
	if err := validateReleaseGovernancePolicyConfig(cfg); err != nil {
		return nil, err
	}

	_, err := s.CreateOrUpdateCategoryConfig(ctx, tenantID, ConfigTypeToolSettings, "release_governance_policy", cfg, updatedBy)
	if err != nil {
		s.logger.Error("Failed to save release governance policy config", zap.Error(err))
		return nil, err
	}

	if tenantID != nil {
		s.logger.Info("Release governance policy configuration updated",
			zap.String("tenantID", tenantID.String()),
			zap.String("updatedBy", updatedBy.String()))
	} else {
		s.logger.Info("Global release governance policy configuration updated",
			zap.String("updatedBy", updatedBy.String()))
	}

	return cfg, nil
}

// GetRobotSREPolicyConfig retrieves Robot SRE policy configuration.
// Tenant scope falls back to global scope when no tenant override exists.
func (s *Service) GetRobotSREPolicyConfig(ctx context.Context, tenantID *uuid.UUID) (*RobotSREPolicyConfig, error) {
	config, err := s.repository.FindByKey(ctx, tenantID, "robot_sre_policy")
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) || strings.Contains(err.Error(), "no rows in result set") {
			if tenantID != nil {
				s.logger.Info("Tenant Robot SRE policy config not found, using global default",
					zap.String("tenantID", tenantID.String()))
				config, err = s.repository.FindByKey(ctx, nil, "robot_sre_policy")
				if err == nil {
					goto parseConfig
				}
			}
			defaultCfg := defaultRobotSREPolicyConfig()
			return &defaultCfg, nil
		}
		if tenantID != nil {
			s.logger.Error("Failed to get Robot SRE policy config", zap.Error(err), zap.String("tenantID", tenantID.String()))
		} else {
			s.logger.Error("Failed to get global Robot SRE policy config", zap.Error(err))
		}
		return nil, err
	}

	if config.Status() != ConfigStatusActive {
		if tenantID != nil {
			s.logger.Error("Robot SRE policy config is not active", zap.String("tenantID", tenantID.String()), zap.String("status", string(config.Status())))
		} else {
			s.logger.Error("Global Robot SRE policy config is not active", zap.String("status", string(config.Status())))
		}
		return nil, fmt.Errorf("robot sre policy configuration is not active")
	}

parseConfig:
	var policyConfig RobotSREPolicyConfig
	if err := json.Unmarshal(config.ConfigValue(), &policyConfig); err != nil {
		s.logger.Error("Failed to unmarshal Robot SRE policy config", zap.Error(err))
		return nil, err
	}
	applyRobotSREPolicyDefaults(&policyConfig)
	if err := validateRobotSREPolicyConfig(&policyConfig); err != nil {
		s.logger.Error("Invalid Robot SRE policy config persisted", zap.Error(err))
		return nil, err
	}
	return &policyConfig, nil
}

// GetRobotSREPolicyDefaults returns the current deployment-aware default policy.
func (s *Service) GetRobotSREPolicyDefaults() RobotSREPolicyConfig {
	return defaultRobotSREPolicyConfig()
}

// UpdateRobotSREPolicyConfig updates Robot SRE policy configuration.
func (s *Service) UpdateRobotSREPolicyConfig(ctx context.Context, tenantID *uuid.UUID, cfg *RobotSREPolicyConfig, updatedBy uuid.UUID) (*RobotSREPolicyConfig, error) {
	if updatedBy == uuid.Nil {
		return nil, errors.New("updated by user ID is required")
	}
	applyRobotSREPolicyDefaults(cfg)
	if err := validateRobotSREPolicyConfig(cfg); err != nil {
		return nil, err
	}

	_, err := s.CreateOrUpdateCategoryConfig(ctx, tenantID, ConfigTypeToolSettings, "robot_sre_policy", cfg, updatedBy)
	if err != nil {
		s.logger.Error("Failed to save Robot SRE policy config", zap.Error(err))
		return nil, err
	}

	if tenantID != nil {
		s.logger.Info("Robot SRE policy configuration updated",
			zap.String("tenantID", tenantID.String()),
			zap.String("updatedBy", updatedBy.String()))
	} else {
		s.logger.Info("Global Robot SRE policy configuration updated",
			zap.String("updatedBy", updatedBy.String()))
	}

	return cfg, nil
}

// CreateExternalService creates a new external service configuration
func (s *Service) CreateExternalService(ctx context.Context, tenantID *uuid.UUID, serviceConfig *ExternalServiceConfig, createdBy uuid.UUID) (*ExternalServiceConfig, error) {
	if serviceConfig == nil {
		return nil, errors.New("external service config is required")
	}
	if serviceConfig.Name == "" {
		return nil, errors.New("external service name is required")
	}
	if serviceConfig.URL == "" {
		return nil, errors.New("external service URL is required")
	}
	if createdBy == uuid.Nil {
		return nil, errors.New("created by user ID is required")
	}

	// Generate a unique key for the service
	configKey := fmt.Sprintf("external_service_%s", strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(serviceConfig.Name, " ", "_"), "-", "_")))

	// Create the configuration
	_, err := s.CreateOrUpdateCategoryConfig(ctx, tenantID, ConfigTypeExternalServices, configKey, serviceConfig, createdBy)
	if err != nil {
		s.logger.Error("Failed to create external service config", zap.Error(err))
		return nil, err
	}

	if tenantID != nil {
		s.logger.Info("External service configuration created",
			zap.String("serviceName", serviceConfig.Name),
			zap.String("tenantID", tenantID.String()),
			zap.String("createdBy", createdBy.String()))
	} else {
		s.logger.Info("Global external service configuration created",
			zap.String("serviceName", serviceConfig.Name),
			zap.String("createdBy", createdBy.String()))
	}

	return serviceConfig, nil
}

// UpdateExternalService updates an existing external service configuration
func (s *Service) UpdateExternalService(ctx context.Context, tenantID *uuid.UUID, serviceName string, serviceConfig *ExternalServiceConfig, updatedBy uuid.UUID) (*ExternalServiceConfig, error) {
	if serviceConfig == nil {
		return nil, errors.New("external service config is required")
	}
	if serviceName == "" {
		return nil, errors.New("external service name is required")
	}
	if serviceConfig.Name == "" {
		return nil, errors.New("external service name is required")
	}
	if serviceConfig.URL == "" {
		return nil, errors.New("external service URL is required")
	}
	if updatedBy == uuid.Nil {
		return nil, errors.New("updated by user ID is required")
	}

	// Generate the config key
	configKey := fmt.Sprintf("external_service_%s", strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(serviceName, " ", "_"), "-", "_")))

	// Update the configuration
	_, err := s.CreateOrUpdateCategoryConfig(ctx, tenantID, ConfigTypeExternalServices, configKey, serviceConfig, updatedBy)
	if err != nil {
		s.logger.Error("Failed to update external service config", zap.Error(err))
		return nil, err
	}

	if tenantID != nil {
		s.logger.Info("External service configuration updated",
			zap.String("serviceName", serviceConfig.Name),
			zap.String("tenantID", tenantID.String()),
			zap.String("updatedBy", updatedBy.String()))
	} else {
		s.logger.Info("Global external service configuration updated",
			zap.String("serviceName", serviceConfig.Name),
			zap.String("updatedBy", updatedBy.String()))
	}

	return serviceConfig, nil
}

// GetExternalServices retrieves all external service configurations for a tenant
func (s *Service) GetExternalServices(ctx context.Context, tenantID *uuid.UUID) ([]*ExternalServiceConfig, error) {
	configs, err := s.GetConfigsByType(ctx, tenantID, ConfigTypeExternalServices)
	if err != nil {
		s.logger.Error("Failed to get external service configs", zap.Error(err))
		return nil, err
	}

	var services []*ExternalServiceConfig
	for _, config := range configs {
		var serviceConfig ExternalServiceConfig
		if err := json.Unmarshal(config.ConfigValue(), &serviceConfig); err != nil {
			s.logger.Error("Failed to unmarshal external service config",
				zap.String("configKey", config.ConfigKey()),
				zap.Error(err))
			continue
		}
		services = append(services, &serviceConfig)
	}

	return services, nil
}

// GetExternalService retrieves a specific external service configuration
func (s *Service) GetExternalService(ctx context.Context, tenantID *uuid.UUID, serviceName string) (*ExternalServiceConfig, error) {
	if serviceName == "" {
		return nil, errors.New("external service name is required")
	}

	configKey := fmt.Sprintf("external_service_%s", strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(serviceName, " ", "_"), "-", "_")))

	config, err := s.GetConfigByKey(ctx, tenantID, configKey)
	if err != nil && tenantID != nil {
		// Fall back to universal scope – the config may have been seeded globally
		config, err = s.GetConfigByKey(ctx, nil, configKey)
	}
	if err != nil {
		s.logger.Error("Failed to get external service config",
			zap.String("serviceName", serviceName),
			zap.Error(err))
		return nil, err
	}

	var serviceConfig ExternalServiceConfig
	if err := json.Unmarshal(config.ConfigValue(), &serviceConfig); err != nil {
		s.logger.Error("Failed to unmarshal external service config", zap.Error(err))
		return nil, err
	}

	return &serviceConfig, nil
}

// DeleteExternalService deletes an external service configuration
func (s *Service) DeleteExternalService(ctx context.Context, tenantID *uuid.UUID, serviceName string, deletedBy uuid.UUID) error {
	if serviceName == "" {
		return errors.New("external service name is required")
	}
	if deletedBy == uuid.Nil {
		return errors.New("deleted by user ID is required")
	}

	configKey := fmt.Sprintf("external_service_%s", strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(serviceName, " ", "_"), "-", "_")))

	config, err := s.GetConfigByKey(ctx, tenantID, configKey)
	if err != nil && tenantID != nil {
		// Fall back to universal scope – the config may have been seeded globally
		config, err = s.GetConfigByKey(ctx, nil, configKey)
	}
	if err != nil {
		s.logger.Error("Failed to find external service config for deletion",
			zap.String("serviceName", serviceName),
			zap.Error(err))
		return err
	}

	if err := s.DeleteConfig(ctx, config.ID()); err != nil {
		s.logger.Error("Failed to delete external service config", zap.Error(err))
		return err
	}

	if tenantID != nil {
		s.logger.Info("External service configuration deleted",
			zap.String("serviceName", serviceName),
			zap.String("tenantID", tenantID.String()),
			zap.String("deletedBy", deletedBy.String()))
	} else {
		s.logger.Info("Global external service configuration deleted",
			zap.String("serviceName", serviceName),
			zap.String("deletedBy", deletedBy.String()))
	}

	return nil
}

// validateToolAvailabilityConfig validates the tool availability configuration
func (s *Service) validateToolAvailabilityConfig(config *ToolAvailabilityConfig) error {
	if config == nil {
		return errors.New("tool availability config cannot be nil")
	}

	// Ensure at least one build method is enabled
	if !config.BuildMethods.Container && !config.BuildMethods.Packer &&
		!config.BuildMethods.Paketo && !config.BuildMethods.Kaniko &&
		!config.BuildMethods.Buildx && !config.BuildMethods.Nix {
		return errors.New("at least one build method must be enabled")
	}

	// Ensure at least one SBOM tool is enabled
	if !config.SBOMTools.Syft && !config.SBOMTools.Grype && !config.SBOMTools.Trivy {
		return errors.New("at least one SBOM tool must be enabled")
	}

	// Ensure at least one scan tool is enabled
	if !config.ScanTools.Trivy && !config.ScanTools.Clair &&
		!config.ScanTools.Grype && !config.ScanTools.Snyk {
		return errors.New("at least one scan tool must be enabled")
	}

	// Ensure at least one registry type is enabled
	if !config.RegistryTypes.S3 && !config.RegistryTypes.Harbor &&
		!config.RegistryTypes.Quay && !config.RegistryTypes.Artifactory {
		return errors.New("at least one registry type must be enabled")
	}

	// Ensure at least one secret manager is enabled
	if !config.SecretManagers.Vault && !config.SecretManagers.AWSSM &&
		!config.SecretManagers.AzureKV && !config.SecretManagers.GCP {
		return errors.New("at least one secret manager must be enabled")
	}

	switch strings.ToLower(strings.TrimSpace(config.TrivyRuntime.CacheMode)) {
	case "", "shared", "direct":
	default:
		return errors.New("trivy cache mode must be one of: shared, direct")
	}

	return nil
}

func (s *Service) validateTektonTaskImagesConfig(config *TektonTaskImagesConfig) error {
	if config == nil {
		return errors.New("tekton task images config cannot be nil")
	}
	fields := map[string]string{
		"git_clone":       config.GitClone,
		"kaniko_executor": config.KanikoExecutor,
		"buildkit":        config.Buildkit,
		"skopeo":          config.Skopeo,
		"trivy":           config.Trivy,
		"syft":            config.Syft,
		"cosign":          config.Cosign,
		"packer":          config.Packer,
		"python_alpine":   config.PythonAlpine,
		"alpine":          config.Alpine,
		"cleanup_kubectl": config.CleanupKubectl,
	}
	for key, value := range fields {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fmt.Errorf("%s is required", key)
		}
		if !containerImageReferencePattern.MatchString(trimmed) {
			return fmt.Errorf("%s must be a valid image reference", key)
		}
		if !hasExplicitRegistryHost(trimmed) {
			return fmt.Errorf("%s must be a fully qualified image reference with registry host", key)
		}
	}
	return nil
}

func hasExplicitRegistryHost(imageRef string) bool {
	ref := strings.TrimSpace(imageRef)
	if ref == "" {
		return false
	}
	firstSlash := strings.Index(ref, "/")
	if firstSlash <= 0 {
		return false
	}
	host := ref[:firstSlash]
	return strings.Contains(host, ".") || strings.Contains(host, ":") || host == "localhost"
}

func normalizeLegacyTektonTaskImageRefs(cfg *TektonTaskImagesConfig) {
	if cfg == nil {
		return
	}
	legacyToQualified := map[string]string{
		"alpine/git:2.45.2":       "docker.io/alpine/git:2.45.2",
		"moby/buildkit:v0.13.2":   "docker.io/moby/buildkit:v0.13.2",
		"aquasec/trivy:0.57.1":    "docker.io/aquasec/trivy:0.57.1",
		"anchore/syft:v1.18.1":    "docker.io/anchore/syft:v1.18.1",
		"hashicorp/packer:1.10.2": "docker.io/hashicorp/packer:1.10.2",
		"python:3.12-alpine":      "docker.io/library/python:3.12-alpine",
		"alpine:3.20":             "docker.io/library/alpine:3.20",
		"bitnami/kubectl:latest":  "docker.io/bitnami/kubectl:latest",
	}
	normalize := func(v *string) {
		if v == nil {
			return
		}
		trimmed := strings.TrimSpace(*v)
		if replacement, ok := legacyToQualified[trimmed]; ok {
			*v = replacement
		}
	}
	normalize(&cfg.GitClone)
	normalize(&cfg.KanikoExecutor)
	normalize(&cfg.Buildkit)
	normalize(&cfg.Skopeo)
	normalize(&cfg.Trivy)
	normalize(&cfg.Syft)
	normalize(&cfg.Cosign)
	normalize(&cfg.Packer)
	normalize(&cfg.PythonAlpine)
	normalize(&cfg.Alpine)
	normalize(&cfg.CleanupKubectl)
}
