package systemconfig

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Domain errors
var (
	ErrConfigNotFound                = errors.New("system configuration not found")
	ErrInvalidConfigKey              = errors.New("invalid configuration key")
	ErrInvalidConfigValue            = errors.New("invalid configuration value")
	ErrConfigAlreadyExists           = errors.New("configuration already exists")
	ErrInvalidToolAvailabilityConfig = errors.New("invalid tool availability configuration")
)

// ConfigType represents the type of configuration
type ConfigType string

const (
	ConfigTypeLDAP             ConfigType = "ldap"
	ConfigTypeSMTP             ConfigType = "smtp"
	ConfigTypeGeneral          ConfigType = "general"
	ConfigTypeSecurity         ConfigType = "security"
	ConfigTypeBuild            ConfigType = "build"
	ConfigTypeTekton           ConfigType = "tekton"
	ConfigTypeToolSettings     ConfigType = "tool_settings"
	ConfigTypeExternalServices ConfigType = "external_services"
	ConfigTypeMessaging        ConfigType = "messaging"
	ConfigTypeRuntimeServices  ConfigType = "runtime_services"
)

// ConfigStatus represents the status of a configuration
type ConfigStatus string

const (
	ConfigStatusActive   ConfigStatus = "active"
	ConfigStatusInactive ConfigStatus = "inactive"
	ConfigStatusTesting  ConfigStatus = "testing"
)

// LDAPConfig represents LDAP configuration settings
type LDAPConfig struct {
	ProviderName    string   `json:"provider_name,omitempty"`
	ProviderType    string   `json:"provider_type,omitempty"`
	Host            string   `json:"host"`
	Port            int      `json:"port"`
	BaseDN          string   `json:"base_dn"`
	UserSearchBase  string   `json:"user_search_base,omitempty"`
	GroupSearchBase string   `json:"group_search_base,omitempty"`
	BindDN          string   `json:"bind_dn"`
	BindPassword    string   `json:"bind_password"`
	UserFilter      string   `json:"user_filter"`
	GroupFilter     string   `json:"group_filter"`
	StartTLS        bool     `json:"start_tls"`
	SSL             bool     `json:"ssl"`
	AllowedDomains  []string `json:"allowed_domains,omitempty"` // Allowed email domains for LDAP users
	Enabled         bool     `json:"enabled"`
}

// SMTPConfig represents SMTP configuration settings
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	StartTLS bool   `json:"start_tls"`
	SSL      bool   `json:"ssl"`
	Enabled  bool   `json:"enabled"`
}

// GeneralConfig represents general system configuration
type GeneralConfig struct {
	SystemName              string `json:"system_name"`
	SystemDescription       string `json:"system_description"`
	AdminEmail              string `json:"admin_email"`
	SupportEmail            string `json:"support_email"`
	TimeZone                string `json:"time_zone"`
	DateFormat              string `json:"date_format"`
	DefaultLanguage         string `json:"default_language"`
	MessagingEnableNATS     *bool  `json:"messaging_enable_nats,omitempty"`
	WorkflowEnabled         *bool  `json:"workflow_enabled,omitempty"`
	WorkflowPollInterval    string `json:"workflow_poll_interval,omitempty"`
	WorkflowMaxStepsPerTick int    `json:"workflow_max_steps_per_tick,omitempty"`
	ProjectRetentionDays    int    `json:"project_retention_days,omitempty"`
	ProjectLastPurgeAt      string `json:"project_last_purge_at,omitempty"`
	ProjectLastPurgeCount   int    `json:"project_last_purge_count,omitempty"`
	MaintenanceMode         bool   `json:"maintenance_mode"`
}

// MessagingConfig represents messaging system configuration settings
type MessagingConfig struct {
	EnableNATS                 bool  `json:"enable_nats"`
	NATSRequired               *bool `json:"nats_required,omitempty"`
	ExternalOnly               *bool `json:"external_only,omitempty"`
	OutboxEnabled              *bool `json:"outbox_enabled,omitempty"`
	OutboxRelayIntervalSeconds *int  `json:"outbox_relay_interval_seconds,omitempty"`
	OutboxRelayBatchSize       *int  `json:"outbox_relay_batch_size,omitempty"`
	OutboxClaimLeaseSeconds    *int  `json:"outbox_claim_lease_seconds,omitempty"`
}

// RuntimeServicesConfig represents runtime background service endpoints and TLS settings.
type RuntimeServicesConfig struct {
	DispatcherURL                                      string                      `json:"dispatcher_url"`
	DispatcherPort                                     int                         `json:"dispatcher_port"`
	DispatcherMTLSEnabled                              bool                        `json:"dispatcher_mtls_enabled"`
	DispatcherCACert                                   string                      `json:"dispatcher_ca_cert,omitempty"`
	DispatcherClientCert                               string                      `json:"dispatcher_client_cert,omitempty"`
	DispatcherClientKey                                string                      `json:"dispatcher_client_key,omitempty"`
	WorkflowOrchestratorEnabled                        *bool                       `json:"workflow_orchestrator_enabled,omitempty"`
	EmailWorkerURL                                     string                      `json:"email_worker_url"`
	EmailWorkerPort                                    int                         `json:"email_worker_port"`
	EmailWorkerTLSEnabled                              bool                        `json:"email_worker_tls_enabled"`
	NotificationWorkerURL                              string                      `json:"notification_worker_url"`
	NotificationWorkerPort                             int                         `json:"notification_worker_port"`
	NotificationTLSEnabled                             bool                        `json:"notification_tls_enabled"`
	InternalRegistryGCWorkerURL                        string                      `json:"internal_registry_gc_worker_url,omitempty"`
	InternalRegistryGCWorkerPort                       int                         `json:"internal_registry_gc_worker_port,omitempty"`
	InternalRegistryGCWorkerTLSEnabled                 bool                        `json:"internal_registry_gc_worker_tls_enabled,omitempty"`
	HealthCheckTimeoutSecond                           int                         `json:"health_check_timeout_seconds"`
	InternalRegistryTempCleanupEnabled                 *bool                       `json:"internal_registry_temp_cleanup_enabled,omitempty"`
	InternalRegistryTempCleanupRetentionHours          int                         `json:"internal_registry_temp_cleanup_retention_hours,omitempty"`
	InternalRegistryTempCleanupIntervalMinutes         int                         `json:"internal_registry_temp_cleanup_interval_minutes,omitempty"`
	InternalRegistryTempCleanupBatchSize               int                         `json:"internal_registry_temp_cleanup_batch_size,omitempty"`
	InternalRegistryTempCleanupDryRun                  *bool                       `json:"internal_registry_temp_cleanup_dry_run,omitempty"`
	ProviderReadinessWatcherEnabled                    *bool                       `json:"provider_readiness_watcher_enabled,omitempty"`
	ProviderReadinessWatcherIntervalSeconds            int                         `json:"provider_readiness_watcher_interval_seconds,omitempty"`
	ProviderReadinessWatcherTimeoutSeconds             int                         `json:"provider_readiness_watcher_timeout_seconds,omitempty"`
	ProviderReadinessWatcherBatchSize                  int                         `json:"provider_readiness_watcher_batch_size,omitempty"`
	TenantAssetReconcilePolicy                         string                      `json:"tenant_asset_reconcile_policy,omitempty"`
	TenantAssetDriftWatcherEnabled                     *bool                       `json:"tenant_asset_drift_watcher_enabled,omitempty"`
	TenantAssetDriftWatcherIntervalSeconds             int                         `json:"tenant_asset_drift_watcher_interval_seconds,omitempty"`
	TektonHistoryCleanupEnabled                        *bool                       `json:"tekton_history_cleanup_enabled,omitempty"`
	TektonHistoryCleanupSchedule                       string                      `json:"tekton_history_cleanup_schedule,omitempty"`
	TektonHistoryCleanupKeepPipelineRuns               int                         `json:"tekton_history_cleanup_keep_pipelineruns,omitempty"`
	TektonHistoryCleanupKeepTaskRuns                   int                         `json:"tekton_history_cleanup_keep_taskruns,omitempty"`
	TektonHistoryCleanupKeepPods                       int                         `json:"tekton_history_cleanup_keep_pods,omitempty"`
	ImageImportNotificationReceiptCleanupEnabled       *bool                       `json:"image_import_notification_receipt_cleanup_enabled,omitempty"`
	ImageImportNotificationReceiptRetentionDays        int                         `json:"image_import_notification_receipt_retention_days,omitempty"`
	ImageImportNotificationReceiptCleanupIntervalHours int                         `json:"image_import_notification_receipt_cleanup_interval_hours,omitempty"`
	StorageProfiles                                    RuntimeAssetStorageProfiles `json:"storage_profiles,omitempty"`
	// AppHQ tenant lookup service configuration
	AppHQEnabled       *bool  `json:"apphq_enabled,omitempty"`
	AppHQOAuthTokenURL string `json:"apphq_oauth_token_url,omitempty"`
	AppHQClientID      string `json:"apphq_client_id,omitempty"`
	AppHQClientSecret  string `json:"apphq_client_secret,omitempty"`
	AppHQAPIURL        string `json:"apphq_api_url,omitempty"`
	AppHQSystem        string `json:"apphq_system,omitempty"`
	AppHQSystemName    string `json:"apphq_system_name,omitempty"`
	AppHQRun           string `json:"apphq_run,omitempty"`
	AppHQObjCode       string `json:"apphq_obj_cd,omitempty"`
}

// RuntimeAssetStorageProfile describes storage backend options for runtime assets.
type RuntimeAssetStorageProfile struct {
	Type            string   `json:"type,omitempty"` // hostPath | pvc | emptyDir
	HostPath        string   `json:"host_path,omitempty"`
	HostPathType    string   `json:"host_path_type,omitempty"`
	PVCName         string   `json:"pvc_name,omitempty"`
	PVCSize         string   `json:"pvc_size,omitempty"`
	PVCStorageClass string   `json:"pvc_storage_class,omitempty"`
	PVCAccessModes  []string `json:"pvc_access_modes,omitempty"`
}

// RuntimeAssetStorageProfiles groups storage profiles by runtime asset.
type RuntimeAssetStorageProfiles struct {
	InternalRegistry RuntimeAssetStorageProfile `json:"internal_registry,omitempty"`
	TrivyCache       RuntimeAssetStorageProfile `json:"trivy_cache,omitempty"`
}

// BuildConfig represents build system configuration settings
type BuildConfig struct {
	DefaultTimeoutMinutes     int   `json:"default_timeout_minutes"`
	MaxConcurrentJobs         int   `json:"max_concurrent_jobs"`
	WorkerPoolSize            int   `json:"worker_pool_size"`
	MaxQueueSize              int   `json:"max_queue_size"`
	ArtifactRetentionDays     int   `json:"artifact_retention_days"`
	TektonEnabled             bool  `json:"tekton_enabled"`
	MonitorEventDrivenEnabled *bool `json:"monitor_event_driven_enabled,omitempty"`
	EnableTempScanStage       bool  `json:"enable_temp_scan_stage"`
}

// TektonCoreConfig defines defaults for installing/checking Tekton "core" (CRDs/controllers)
// when preparing a Kubernetes infrastructure provider. Stored as a global system config
// (tenant_id = NULL) under config_type="build", config_key="tekton_core".
type TektonCoreConfig struct {
	// install_source: "manifest" (apply release YAML URLs), "helm" (admin installs), "preinstalled" (no install).
	InstallSource string `json:"install_source,omitempty"`

	// ManifestURLs is used when InstallSource="manifest".
	ManifestURLs []string `json:"manifest_urls,omitempty"`

	// Helm settings are used when InstallSource="helm" to generate guidance/hints.
	HelmRepoURL     string `json:"helm_repo_url,omitempty"`
	HelmChart       string `json:"helm_chart,omitempty"`
	HelmReleaseName string `json:"helm_release_name,omitempty"`
	HelmNamespace   string `json:"helm_namespace,omitempty"`

	// AssetsDir optionally overrides where Image Factory loads its Tekton tasks/pipelines manifests from.
	// This is useful for air-gapped deployments where the assets are mounted into the container.
	AssetsDir string `json:"assets_dir,omitempty"`
}

// TektonTaskImagesConfig defines overrideable container images used by Tekton tasks/pipelines.
// Stored as a global system config (tenant_id = NULL) under:
// config_type="tekton", config_key="tekton_task_images".
type TektonTaskImagesConfig struct {
	GitClone       string `json:"git_clone"`
	KanikoExecutor string `json:"kaniko_executor"`
	Buildkit       string `json:"buildkit"`
	Skopeo         string `json:"skopeo"`
	Trivy          string `json:"trivy"`
	Syft           string `json:"syft"`
	Cosign         string `json:"cosign"`
	Packer         string `json:"packer"`
	PythonAlpine   string `json:"python_alpine"`
	Alpine         string `json:"alpine"`
	CleanupKubectl string `json:"cleanup_kubectl"`
}

// ToolAvailabilityConfig represents tool availability settings for administrative control
type ToolAvailabilityConfig struct {
	BuildMethods   BuildMethodAvailability   `json:"build_methods"`
	SBOMTools      SBOMToolAvailability      `json:"sbom_tools"`
	ScanTools      ScanToolAvailability      `json:"scan_tools"`
	RegistryTypes  RegistryTypeAvailability  `json:"registry_types"`
	SecretManagers SecretManagerAvailability `json:"secret_managers"`
	TrivyRuntime   TrivyRuntimeConfig        `json:"trivy_runtime"`
}

type TrivyRuntimeConfig struct {
	CacheMode        string `json:"cache_mode"`
	DBRepository     string `json:"db_repository"`
	JavaDBRepository string `json:"java_db_repository"`
}

// BuildCapabilitiesConfig represents tenant/global entitlement flags for
// advanced build capabilities beyond tool toggles.
type BuildCapabilitiesConfig struct {
	GPU            bool `json:"gpu"`
	Privileged     bool `json:"privileged"`
	MultiArch      bool `json:"multi_arch"`
	HighMemory     bool `json:"high_memory"`
	HostNetworking bool `json:"host_networking"`
	Premium        bool `json:"premium"`
}

// OperationCapabilitiesConfig represents tenant/global entitlement flags for
// operational workflows outside build execution itself.
type OperationCapabilitiesConfig struct {
	Build             bool `json:"build"`
	QuarantineRequest bool `json:"quarantine_request"`
	QuarantineRelease bool `json:"quarantine_release"`
	OnDemandImageScan bool `json:"ondemand_image_scanning"`
}

// CapabilitySurfaceSet describes effective UI/action surface keys allowed for
// the current tenant context.
type CapabilitySurfaceSet struct {
	NavKeys    []string `json:"nav_keys"`
	RouteKeys  []string `json:"route_keys"`
	ActionKeys []string `json:"action_keys"`
}

// CapabilitySurfaceDenial describes deterministic deny metadata for a denied
// route/action surface.
type CapabilitySurfaceDenial struct {
	ReasonCode string `json:"reason_code"`
	Capability string `json:"capability"`
	Message    string `json:"message"`
}

// CapabilitySurfacesResponse is the backend-driven capability surface contract
// for a tenant context.
type CapabilitySurfacesResponse struct {
	TenantID     uuid.UUID                          `json:"tenant_id"`
	Version      string                             `json:"version"`
	Capabilities OperationCapabilitiesConfig        `json:"capabilities"`
	Surfaces     CapabilitySurfaceSet               `json:"surfaces"`
	Denials      map[string]CapabilitySurfaceDenial `json:"denials"`
}

// QuarantinePolicySeverityMapping maps enterprise priority bands (P1..P4)
// to scanner severity labels.
type QuarantinePolicySeverityMapping struct {
	P1 []string `json:"p1"`
	P2 []string `json:"p2"`
	P3 []string `json:"p3"`
	P4 []string `json:"p4"`
}

// QuarantinePolicyConfig represents tenant/global policy thresholds used to
// evaluate quarantine scan outcomes.
type QuarantinePolicyConfig struct {
	Enabled         bool                            `json:"enabled"`
	Mode            string                          `json:"mode"` // enforce | dry_run
	MaxCritical     int                             `json:"max_critical"`
	MaxP2           int                             `json:"max_p2"`
	MaxP3           int                             `json:"max_p3"`
	MaxCVSS         float64                         `json:"max_cvss"`
	SeverityMapping QuarantinePolicySeverityMapping `json:"severity_mapping"`
}

// SORRegistrationConfig controls SOR prerequisite enforcement behavior.
// runtime_error_mode:
//   - "error" (default): surface integration/runtime errors
//   - "deny": treat runtime errors as not registered
//   - "allow": bypass SOR gate on runtime errors
type SORRegistrationConfig struct {
	Enforce          bool   `json:"enforce"`
	RuntimeErrorMode string `json:"runtime_error_mode"`
}

// ReleaseGovernancePolicyConfig controls release failure alert thresholds.
// failure_ratio_threshold:
//   - decimal fraction from 0.0 to 1.0 (example: 0.2 = 20% failures)
type ReleaseGovernancePolicyConfig struct {
	Enabled                      bool    `json:"enabled"`
	FailureRatioThreshold        float64 `json:"failure_ratio_threshold"`
	ConsecutiveFailuresThreshold int     `json:"consecutive_failures_threshold"`
	MinimumSamples               int     `json:"minimum_samples"`
	WindowMinutes                int     `json:"window_minutes"`
}

// RobotSREOperatorRule represents an operator-defined detection/remediation rule
// managed from the admin UI. These are additive to built-in rules.
type RobotSREOperatorRule struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Domain             string            `json:"domain"`
	IncidentType       string            `json:"incident_type"`
	Severity           string            `json:"severity"`
	Enabled            bool              `json:"enabled"`
	Source             string            `json:"source"`
	MatchLabels        map[string]string `json:"match_labels,omitempty"`
	Threshold          int               `json:"threshold,omitempty"`
	ForDurationSeconds int               `json:"for_duration_seconds,omitempty"`
	SuggestedAction    string            `json:"suggested_action,omitempty"`
	AutoAllowed        bool              `json:"auto_allowed,omitempty"`
}

// RobotSREDetectorRule represents an active log detector rule used by the SRE
// Smart Bot log intelligence pipeline.
type RobotSREDetectorRule struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	Source          string `json:"source,omitempty"`
	Query           string `json:"query"`
	Threshold       int    `json:"threshold,omitempty"`
	Domain          string `json:"domain"`
	IncidentType    string `json:"incident_type"`
	Severity        string `json:"severity"`
	Confidence      string `json:"confidence,omitempty"`
	SignalKey       string `json:"signal_key,omitempty"`
	SuggestedAction string `json:"suggested_action,omitempty"`
	AutoCreated     bool   `json:"auto_created,omitempty"`
}

// RobotSREChannelProvider defines a configurable operator channel integration.
type RobotSREChannelProvider struct {
	ID                          string            `json:"id"`
	Name                        string            `json:"name"`
	Kind                        string            `json:"kind"`
	Enabled                     bool              `json:"enabled"`
	SupportsInteractiveApproval bool              `json:"supports_interactive_approval,omitempty"`
	ConfigRef                   string            `json:"config_ref,omitempty"`
	Settings                    map[string]string `json:"settings,omitempty"`
}

// RobotSREMCPServer defines a configurable MCP-style tool endpoint that the
// SRE Smart Bot agent/runtime may use for evidence gathering and bounded tool
// execution.
type RobotSREMCPServer struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Kind             string            `json:"kind"`
	Enabled          bool              `json:"enabled"`
	Transport        string            `json:"transport"`
	Endpoint         string            `json:"endpoint,omitempty"`
	ConfigRef        string            `json:"config_ref,omitempty"`
	AllowedTools     []string          `json:"allowed_tools,omitempty"`
	ReadOnly         bool              `json:"read_only,omitempty"`
	ApprovalRequired bool              `json:"approval_required,omitempty"`
	Settings         map[string]string `json:"settings,omitempty"`
}

// RobotSREAgentRuntimeConfig controls the optional AI/agent feature layer that
// sits on top of deterministic incident, policy, and approval logic.
type RobotSREAgentRuntimeConfig struct {
	Enabled                            bool   `json:"enabled"`
	Provider                           string `json:"provider,omitempty"`
	Model                              string `json:"model,omitempty"`
	BaseURL                            string `json:"base_url,omitempty"`
	SystemPromptRef                    string `json:"system_prompt_ref,omitempty"`
	OperatorSummaryEnabled             bool   `json:"operator_summary_enabled"`
	HypothesisRankingEnabled           bool   `json:"hypothesis_ranking_enabled"`
	DraftActionPlansEnabled            bool   `json:"draft_action_plans_enabled"`
	ConversationalApprovalSupport      bool   `json:"conversational_approval_support"`
	MaxToolCallsPerTurn                int    `json:"max_tool_calls_per_turn,omitempty"`
	MaxIncidentsPerSummary             int    `json:"max_incidents_per_summary,omitempty"`
	RequireHumanConfirmationForMessage bool   `json:"require_human_confirmation_for_message"`
}

// RobotSREPolicyConfig controls environment posture, operator channels, and
// operator-defined rules for the SRE Smart Bot subsystem.
type RobotSREPolicyConfig struct {
	DisplayName                      string                     `json:"display_name"`
	Enabled                          bool                       `json:"enabled"`
	EnvironmentMode                  string                     `json:"environment_mode"`
	DetectorLearningMode             string                     `json:"detector_learning_mode,omitempty"`
	DefaultChannel                   string                     `json:"default_channel,omitempty"`
	DefaultChannelProviderID         string                     `json:"default_channel_provider_id,omitempty"`
	AutoObserveEnabled               bool                       `json:"auto_observe_enabled"`
	AutoNotifyEnabled                bool                       `json:"auto_notify_enabled"`
	AutoContainEnabled               bool                       `json:"auto_contain_enabled"`
	AutoRecoverEnabled               bool                       `json:"auto_recover_enabled"`
	RequireApprovalForRecover        bool                       `json:"require_approval_for_recover"`
	RequireApprovalForDisruptive     bool                       `json:"require_approval_for_disruptive"`
	DuplicateAlertSuppressionSeconds int                        `json:"duplicate_alert_suppression_seconds"`
	ActionCooldownSeconds            int                        `json:"action_cooldown_seconds"`
	EnabledDomains                   []string                   `json:"enabled_domains"`
	ChannelProviders                 []RobotSREChannelProvider  `json:"channel_providers,omitempty"`
	MCPServers                       []RobotSREMCPServer        `json:"mcp_servers,omitempty"`
	AgentRuntime                     RobotSREAgentRuntimeConfig `json:"agent_runtime"`
	DetectorRules                    []RobotSREDetectorRule     `json:"detector_rules,omitempty"`
	OperatorRules                    []RobotSREOperatorRule     `json:"operator_rules,omitempty"`
}

// BuildMethodAvailability represents availability of build methods
type BuildMethodAvailability struct {
	Container bool `json:"container"`
	Packer    bool `json:"packer"`
	Paketo    bool `json:"paketo"`
	Kaniko    bool `json:"kaniko"`
	Buildx    bool `json:"buildx"`
	Nix       bool `json:"nix"`
}

// SBOMToolAvailability represents availability of SBOM generation tools
type SBOMToolAvailability struct {
	Syft  bool `json:"syft"`
	Grype bool `json:"grype"`
	Trivy bool `json:"trivy"`
}

// ScanToolAvailability represents availability of security scanning tools
type ScanToolAvailability struct {
	Trivy bool `json:"trivy"`
	Clair bool `json:"clair"`
	Grype bool `json:"grype"`
	Snyk  bool `json:"snyk"`
}

// RegistryTypeAvailability represents availability of registry backends
type RegistryTypeAvailability struct {
	S3          bool `json:"s3"`
	Harbor      bool `json:"harbor"`
	Quay        bool `json:"quay"`
	Artifactory bool `json:"artifactory"`
}

// SecretManagerAvailability represents availability of secret management backends
type SecretManagerAvailability struct {
	Vault   bool `json:"vault"`
	AWSSM   bool `json:"aws_secretsmanager"`
	AzureKV bool `json:"azure_keyvault"`
	GCP     bool `json:"gcp_secretmanager"`
}

// ExternalServiceConfig represents configuration for an external service
type ExternalServiceConfig struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	URL         string            `json:"url"`
	APIKey      string            `json:"api_key"`
	Headers     map[string]string `json:"headers,omitempty"`
	Enabled     bool              `json:"enabled"`
}

// SystemConfig represents the system configuration aggregate root
type SystemConfig struct {
	id          uuid.UUID
	tenantID    *uuid.UUID
	configType  ConfigType
	configKey   string
	configValue json.RawMessage
	status      ConfigStatus
	description string
	isDefault   bool
	createdBy   uuid.UUID
	updatedBy   uuid.UUID
	createdAt   time.Time
	updatedAt   time.Time
	version     int
}

// NewSystemConfig creates a new system configuration
func NewSystemConfig(tenantID *uuid.UUID, configType ConfigType, configKey string, configValue interface{}, description string, createdBy uuid.UUID) (*SystemConfig, error) {
	if configType == "" {
		return nil, errors.New("config type is required")
	}
	if configKey == "" {
		return nil, ErrInvalidConfigKey
	}
	if createdBy == uuid.Nil {
		return nil, errors.New("created by user ID is required")
	}

	// Validate and marshal the config value
	valueBytes, err := json.Marshal(configValue)
	if err != nil {
		return nil, ErrInvalidConfigValue
	}

	now := time.Now().UTC()

	return &SystemConfig{
		id:          uuid.New(),
		tenantID:    tenantID,
		configType:  configType,
		configKey:   configKey,
		configValue: valueBytes,
		status:      ConfigStatusActive,
		description: description,
		isDefault:   false,
		createdBy:   createdBy,
		updatedBy:   createdBy,
		createdAt:   now,
		updatedAt:   now,
		version:     1,
	}, nil
}

// NewSystemConfigFromExisting creates a system config from existing data
func NewSystemConfigFromExisting(
	id uuid.UUID,
	tenantID *uuid.UUID,
	configType ConfigType,
	configKey string,
	configValue json.RawMessage,
	status ConfigStatus,
	description string,
	isDefault bool,
	createdBy, updatedBy uuid.UUID,
	createdAt, updatedAt time.Time,
	version int,
) (*SystemConfig, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid config ID")
	}
	if configType == "" {
		return nil, errors.New("config type is required")
	}
	if configKey == "" {
		return nil, ErrInvalidConfigKey
	}

	return &SystemConfig{
		id:          id,
		tenantID:    tenantID,
		configType:  configType,
		configKey:   configKey,
		configValue: configValue,
		status:      status,
		description: description,
		isDefault:   isDefault,
		createdBy:   createdBy,
		updatedBy:   updatedBy,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		version:     version,
	}, nil
}

// ID returns the configuration ID
func (c *SystemConfig) ID() uuid.UUID {
	return c.id
}

// TenantID returns the tenant ID
func (c *SystemConfig) TenantID() *uuid.UUID {
	return c.tenantID
}

// ConfigType returns the configuration type
func (c *SystemConfig) ConfigType() ConfigType {
	return c.configType
}

// ConfigKey returns the configuration key
func (c *SystemConfig) ConfigKey() string {
	return c.configKey
}

// ConfigValue returns the raw configuration value
func (c *SystemConfig) ConfigValue() json.RawMessage {
	return c.configValue
}

// GetLDAPConfig returns the LDAP configuration
func (c *SystemConfig) GetLDAPConfig() (*LDAPConfig, error) {
	if c.configType != ConfigTypeLDAP {
		return nil, errors.New("configuration is not LDAP type")
	}

	var config LDAPConfig
	if err := json.Unmarshal(c.configValue, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetSMTPConfig returns the SMTP configuration
func (c *SystemConfig) GetSMTPConfig() (*SMTPConfig, error) {
	if c.configType != ConfigTypeSMTP {
		return nil, errors.New("configuration is not SMTP type")
	}

	var config SMTPConfig
	if err := json.Unmarshal(c.configValue, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetSecurityConfig returns the security configuration
func (c *SystemConfig) GetSecurityConfig() (*SecurityConfig, error) {
	if c.configType != ConfigTypeSecurity {
		return nil, errors.New("configuration is not security type")
	}

	var config SecurityConfig
	if err := json.Unmarshal(c.configValue, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetGeneralConfig returns the general configuration
func (c *SystemConfig) GetGeneralConfig() (*GeneralConfig, error) {
	if c.configType != ConfigTypeGeneral {
		return nil, errors.New("configuration is not general type")
	}

	var config GeneralConfig
	if err := json.Unmarshal(c.configValue, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetMessagingConfig returns the messaging configuration
func (c *SystemConfig) GetMessagingConfig() (*MessagingConfig, error) {
	if c.configType != ConfigTypeMessaging {
		return nil, errors.New("configuration is not messaging type")
	}

	var config MessagingConfig
	if err := json.Unmarshal(c.configValue, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetBuildConfig returns the build configuration.
func (c *SystemConfig) GetBuildConfig() (*BuildConfig, error) {
	if c.configType != ConfigTypeBuild {
		return nil, errors.New("configuration is not build type")
	}

	var config BuildConfig
	if err := json.Unmarshal(c.configValue, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetRuntimeServicesConfig returns the runtime services configuration.
func (c *SystemConfig) GetRuntimeServicesConfig() (*RuntimeServicesConfig, error) {
	if c.configType != ConfigTypeRuntimeServices {
		return nil, errors.New("configuration is not runtime_services type")
	}

	var config RuntimeServicesConfig
	if err := json.Unmarshal(c.configValue, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetExternalServiceConfig returns the external service configuration
func (c *SystemConfig) GetExternalServiceConfig() (*ExternalServiceConfig, error) {
	if c.configType != ConfigTypeExternalServices {
		return nil, errors.New("configuration is not external_services type")
	}

	var config ExternalServiceConfig
	if err := json.Unmarshal(c.configValue, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetRobotSREPolicyConfig returns the Robot SRE policy configuration.
func (c *SystemConfig) GetRobotSREPolicyConfig() (*RobotSREPolicyConfig, error) {
	if c.configType != ConfigTypeToolSettings {
		return nil, errors.New("configuration is not tool_settings type")
	}
	if c.configKey != "robot_sre_policy" {
		return nil, errors.New("configuration is not robot_sre_policy")
	}

	var config RobotSREPolicyConfig
	if err := json.Unmarshal(c.configValue, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// Status returns the configuration status
func (c *SystemConfig) Status() ConfigStatus {
	return c.status
}

// Description returns the configuration description
func (c *SystemConfig) Description() string {
	return c.description
}

// IsDefault returns true if this is a default configuration
func (c *SystemConfig) IsDefault() bool {
	return c.isDefault
}

// IsActive returns true if the configuration is active
func (c *SystemConfig) IsActive() bool {
	return c.status == ConfigStatusActive
}

// CreatedBy returns the user who created the configuration
func (c *SystemConfig) CreatedBy() uuid.UUID {
	return c.createdBy
}

// UpdatedBy returns the user who last updated the configuration
func (c *SystemConfig) UpdatedBy() uuid.UUID {
	return c.updatedBy
}

// CreatedAt returns the creation timestamp
func (c *SystemConfig) CreatedAt() time.Time {
	return c.createdAt
}

// UpdatedAt returns the last update timestamp
func (c *SystemConfig) UpdatedAt() time.Time {
	return c.updatedAt
}

// Version returns the aggregate version for optimistic concurrency
func (c *SystemConfig) Version() int {
	return c.version
}

// UpdateValue updates the configuration value
func (c *SystemConfig) UpdateValue(configValue interface{}, updatedBy uuid.UUID) error {
	if updatedBy == uuid.Nil {
		return errors.New("updated by user ID is required")
	}

	valueBytes, err := json.Marshal(configValue)
	if err != nil {
		return ErrInvalidConfigValue
	}

	c.configValue = valueBytes
	c.updatedBy = updatedBy
	c.updatedAt = time.Now().UTC()
	c.version++
	return nil
}

// UpdateDescription updates the configuration description
func (c *SystemConfig) UpdateDescription(description string, updatedBy uuid.UUID) error {
	if updatedBy == uuid.Nil {
		return errors.New("updated by user ID is required")
	}

	c.description = description
	c.updatedBy = updatedBy
	c.updatedAt = time.Now().UTC()
	c.version++
	return nil
}

// Activate activates the configuration
func (c *SystemConfig) Activate(updatedBy uuid.UUID) error {
	if updatedBy == uuid.Nil {
		return errors.New("updated by user ID is required")
	}

	c.status = ConfigStatusActive
	c.updatedBy = updatedBy
	c.updatedAt = time.Now().UTC()
	c.version++
	return nil
}

// Deactivate deactivates the configuration
func (c *SystemConfig) Deactivate(updatedBy uuid.UUID) error {
	if updatedBy == uuid.Nil {
		return errors.New("updated by user ID is required")
	}

	c.status = ConfigStatusInactive
	c.updatedBy = updatedBy
	c.updatedAt = time.Now().UTC()
	c.version++
	return nil
}

// SetTestingStatus sets the configuration to testing status
func (c *SystemConfig) SetTestingStatus(updatedBy uuid.UUID) error {
	if updatedBy == uuid.Nil {
		return errors.New("updated by user ID is required")
	}

	c.status = ConfigStatusTesting
	c.updatedBy = updatedBy
	c.updatedAt = time.Now().UTC()
	c.version++
	return nil
}
