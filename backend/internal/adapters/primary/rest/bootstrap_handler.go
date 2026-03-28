package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	appbootstrap "github.com/srikarm/image-factory/internal/application/bootstrap"
	"github.com/srikarm/image-factory/internal/domain/sso"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

// BootstrapHandler handles first-run setup bootstrap APIs.
type BootstrapHandler struct {
	service             *appbootstrap.Service
	systemConfigService *systemconfig.Service
	ssoService          *sso.Service
	logger              *zap.Logger
}

func NewBootstrapHandler(service *appbootstrap.Service, systemConfigService *systemconfig.Service, ssoService *sso.Service, logger *zap.Logger) *BootstrapHandler {
	return &BootstrapHandler{service: service, systemConfigService: systemConfigService, ssoService: ssoService, logger: logger}
}

type BootstrapStatusResponse struct {
	SetupRequired  bool                `json:"setup_required"`
	Status         string              `json:"status"`
	State          *appbootstrap.State `json:"state,omitempty"`
	RequiredSteps  []string            `json:"required_steps,omitempty"`
	CompletedSteps []string            `json:"completed_steps,omitempty"`
	MissingSteps   []string            `json:"missing_steps,omitempty"`
}

type BootstrapSaveStepRequest struct {
	ConfigKey string                 `json:"config_key,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty"`
	Type      string                 `json:"type,omitempty"`
}

type BootstrapSaveAllRequest struct {
	General         map[string]interface{} `json:"general"`
	SMTP            map[string]interface{} `json:"smtp"`
	LDAP            map[string]interface{} `json:"ldap"`
	RuntimeServices map[string]interface{} `json:"runtime_services"`
	ExternalService map[string]interface{} `json:"external_service,omitempty"`
	SSOType         string                 `json:"sso_type,omitempty"`
	OIDC            map[string]interface{} `json:"oidc,omitempty"`
	SAML            map[string]interface{} `json:"saml,omitempty"`
}

type BootstrapDefaultsResponse struct {
	General         map[string]interface{} `json:"general"`
	SMTP            map[string]interface{} `json:"smtp"`
	LDAP            map[string]interface{} `json:"ldap"`
	RuntimeServices map[string]interface{} `json:"runtime_services"`
	ExternalService map[string]interface{} `json:"external_service"`
	SSO             map[string]interface{} `json:"sso"`
}

// GetStatus handles GET /api/v1/bootstrap/status
func (h *BootstrapHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	state, err := h.service.GetState(r.Context())
	if err != nil {
		h.logger.Error("Failed to get bootstrap state", zap.Error(err))
		http.Error(w, "Failed to get bootstrap status", http.StatusInternalServerError)
		return
	}

	resp := BootstrapStatusResponse{SetupRequired: false, Status: "not_initialized", State: nil}
	if state != nil {
		resp.SetupRequired = state.SetupRequired
		resp.Status = state.Status
		resp.State = state
		required, completed, missing := h.getSetupStepProgress(r.Context())
		resp.RequiredSteps = required
		resp.CompletedSteps = completed
		resp.MissingSteps = missing
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetDefaults handles GET /api/v1/bootstrap/defaults
func (h *BootstrapHandler) GetDefaults(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil || !authCtx.IsSystemAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	ldapDefaults := resolveBootstrapLDAPDefaults()

	resp := BootstrapDefaultsResponse{
		General: map[string]interface{}{
			"system_name":        envOr("IF_SYSTEM_NAME", "Image Factory"),
			"system_description": envOr("IF_SYSTEM_DESCRIPTION", "Container image factory platform"),
			"admin_email":        envOr("IF_SYSTEM_ADMIN_EMAIL", "admin@imgfactory.com"),
			"support_email":      envOr("IF_SYSTEM_SUPPORT_EMAIL", "support@imgfactory.com"),
			"time_zone":          envOr("IF_SYSTEM_TIME_ZONE", "UTC"),
			"date_format":        envOr("IF_SYSTEM_DATE_FORMAT", "YYYY-MM-DD"),
			"default_language":   envOr("IF_SYSTEM_DEFAULT_LANGUAGE", "en"),
			"maintenance_mode":   envBoolOr("IF_SYSTEM_MAINTENANCE_MODE", false),
		},
		SMTP: map[string]interface{}{
			"host":      envOr("IF_SMTP_HOST", ""),
			"port":      envIntOr("IF_SMTP_PORT", 587),
			"username":  envOr("IF_SMTP_USERNAME", ""),
			"password":  envOr("IF_SMTP_PASSWORD", ""),
			"from":      envOr("IF_SMTP_FROM_EMAIL", "noreply@imgfactory.com"),
			"start_tls": envBoolOr("IF_SMTP_START_TLS", true),
			"ssl":       envBoolOr("IF_SMTP_USE_TLS", false),
			"enabled":   envOr("IF_SMTP_HOST", "") != "",
		},
		LDAP: map[string]interface{}{
			"provider_name":     envOrAny("Active Directory", "IF_AUTH_LDAP_PROVIDER_NAME", "LDAP_PROVIDER_NAME"),
			"provider_type":     envOrAny("active_directory", "IF_AUTH_LDAP_PROVIDER_TYPE", "LDAP_PROVIDER_TYPE"),
			"host":              ldapDefaults.Host,
			"port":              ldapDefaults.Port,
			"base_dn":           ldapDefaults.BaseDN,
			"user_search_base":  ldapDefaults.UserSearchBase,
			"group_search_base": ldapDefaults.GroupSearchBase,
			"bind_dn":           ldapDefaults.BindDN,
			"bind_password":     ldapDefaults.BindPassword,
			"user_filter":       envOrAny("(uid=%s)", "IF_AUTH_LDAP_USER_FILTER", "LDAP_USER_FILTER"),
			"group_filter":      envOrAny("(member=%s)", "IF_AUTH_LDAP_GROUP_FILTER", "LDAP_GROUP_FILTER"),
			"start_tls":         ldapDefaults.StartTLS,
			"ssl":               ldapDefaults.UseTLS,
			"allowed_domains":   ldapDefaults.AllowedDomains,
			"enabled":           ldapDefaults.Enabled,
		},
		RuntimeServices: map[string]interface{}{
			"dispatcher_url":                              envOr("IF_DISPATCHER_URL", "http://localhost"),
			"dispatcher_port":                             envIntOr("IF_DISPATCHER_PORT", 8084),
			"dispatcher_mtls_enabled":                     envBoolOr("IF_DISPATCHER_MTLS_ENABLED", false),
			"dispatcher_ca_cert":                          envOr("IF_DISPATCHER_CA_CERT", ""),
			"dispatcher_client_cert":                      envOr("IF_DISPATCHER_CLIENT_CERT", ""),
			"dispatcher_client_key":                       envOr("IF_DISPATCHER_CLIENT_KEY", ""),
			"workflow_orchestrator_enabled":               envBoolOr("IF_WORKFLOW_ENABLED", true),
			"email_worker_url":                            envOr("IF_EMAIL_WORKER_URL", "http://localhost"),
			"email_worker_port":                           envIntOr("IF_EMAIL_WORKER_PORT", 8081),
			"email_worker_tls_enabled":                    envBoolOr("IF_EMAIL_WORKER_TLS_ENABLED", false),
			"notification_worker_url":                     envOr("IF_NOTIFICATION_WORKER_URL", "http://localhost"),
			"notification_worker_port":                    envIntOr("IF_NOTIFICATION_WORKER_PORT", 8083),
			"notification_tls_enabled":                    envBoolOr("IF_NOTIFICATION_WORKER_TLS_ENABLED", false),
			"health_check_timeout_seconds":                envIntOr("IF_RUNTIME_HEALTH_TIMEOUT_SECONDS", 5),
			"provider_readiness_watcher_enabled":          envBoolOr("IF_PROVIDER_READINESS_WATCHER_ENABLED", true),
			"provider_readiness_watcher_interval_seconds": envIntOr("IF_PROVIDER_READINESS_WATCHER_INTERVAL_SECONDS", 180),
			"provider_readiness_watcher_timeout_seconds":  envIntOr("IF_PROVIDER_READINESS_WATCHER_TIMEOUT_SECONDS", 90),
			"provider_readiness_watcher_batch_size":       envIntOr("IF_PROVIDER_READINESS_WATCHER_BATCH_SIZE", 200),
			"tenant_asset_reconcile_policy":               envOr("IF_TENANT_ASSET_RECONCILE_POLICY", "full_reconcile_on_prepare"),
			"tenant_asset_drift_watcher_enabled":          envBoolOr("IF_TENANT_ASSET_DRIFT_WATCHER_ENABLED", true),
			"tenant_asset_drift_watcher_interval_seconds": envIntOr("IF_TENANT_ASSET_DRIFT_WATCHER_INTERVAL_SECONDS", 300),
			"storage_profiles": map[string]interface{}{
				"internal_registry": map[string]interface{}{
					"type":              envOr("IF_INTERNAL_REGISTRY_STORAGE_TYPE", "hostPath"),
					"host_path":         envOr("IF_INTERNAL_REGISTRY_STORAGE_HOST_PATH", "/var/lib/image-factory/registry"),
					"host_path_type":    envOr("IF_INTERNAL_REGISTRY_STORAGE_HOST_PATH_TYPE", "DirectoryOrCreate"),
					"pvc_name":          envOr("IF_INTERNAL_REGISTRY_STORAGE_PVC_NAME", "image-factory-registry-data"),
					"pvc_size":          envOr("IF_INTERNAL_REGISTRY_STORAGE_PVC_SIZE", "20Gi"),
					"pvc_storage_class": envOr("IF_INTERNAL_REGISTRY_STORAGE_PVC_STORAGE_CLASS", ""),
					"pvc_access_modes":  envCSVOr("IF_INTERNAL_REGISTRY_STORAGE_PVC_ACCESS_MODES", []string{"ReadWriteOnce"}),
				},
				"trivy_cache": map[string]interface{}{
					"type":              envOr("IF_TRIVY_CACHE_STORAGE_TYPE", "emptyDir"),
					"host_path":         envOr("IF_TRIVY_CACHE_STORAGE_HOST_PATH", ""),
					"host_path_type":    envOr("IF_TRIVY_CACHE_STORAGE_HOST_PATH_TYPE", "DirectoryOrCreate"),
					"pvc_name":          envOr("IF_TRIVY_CACHE_STORAGE_PVC_NAME", "image-factory-trivy-cache"),
					"pvc_size":          envOr("IF_TRIVY_CACHE_STORAGE_PVC_SIZE", "10Gi"),
					"pvc_storage_class": envOr("IF_TRIVY_CACHE_STORAGE_PVC_STORAGE_CLASS", ""),
					"pvc_access_modes":  envCSVOr("IF_TRIVY_CACHE_STORAGE_PVC_ACCESS_MODES", []string{"ReadWriteOnce"}),
				},
			},
		},
		ExternalService: map[string]interface{}{
			"name":        envOr("IF_EXTERNAL_SERVICE_NAME", "tenant-service"),
			"description": envOr("IF_EXTERNAL_SERVICE_DESCRIPTION", "Tenant service integration"),
			"url":         envOr("IF_EXTERNAL_SERVICE_URL", ""),
			"api_key":     envOr("IF_EXTERNAL_SERVICE_API_KEY", ""),
			"enabled":     false,
		},
		SSO: map[string]interface{}{
			"type": "oidc",
			"oidc": map[string]interface{}{
				"name":              "PingFed",
				"issuer":            envOr("IF_AUTH_OIDC_ISSUER", ""),
				"client_id":         envOr("IF_AUTH_OIDC_CLIENT_ID", ""),
				"client_secret":     envOr("IF_AUTH_OIDC_CLIENT_SECRET", ""),
				"authorization_url": envOr("IF_AUTH_OIDC_AUTHORIZATION_URL", ""),
				"token_url":         envOr("IF_AUTH_OIDC_TOKEN_URL", ""),
				"userinfo_url":      envOr("IF_AUTH_OIDC_USERINFO_URL", ""),
				"jwks_url":          envOr("IF_AUTH_OIDC_JWKS_URL", ""),
				"redirect_uris":     []string{envOr("IF_AUTH_OIDC_REDIRECT_URI", "http://localhost:3000/auth/callback")},
				"scopes":            []string{"openid", "profile", "email"},
				"response_types":    []string{"code"},
				"grant_types":       []string{"authorization_code"},
				"attributes":        map[string]interface{}{},
				"enabled":           envBoolOr("IF_AUTH_OIDC_ENABLED", false),
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// StartSetup handles POST /api/v1/bootstrap/start
func (h *BootstrapHandler) StartSetup(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil || !authCtx.IsSystemAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.service.MarkSetupInProgress(r.Context()); err != nil {
		h.logger.Error("Failed to mark setup in progress", zap.Error(err))
		http.Error(w, "Failed to update bootstrap status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "status": appbootstrap.StatusSetupInProgress})
}

// SaveStep handles POST /api/v1/bootstrap/steps/{step}/save
func (h *BootstrapHandler) SaveStep(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil || !authCtx.IsSystemAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	step := strings.TrimSpace(r.PathValue("step"))
	if step == "" {
		http.Error(w, "Step is required", http.StatusBadRequest)
		return
	}

	var req BootstrapSaveStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Config) == 0 {
		http.Error(w, "Config payload is required", http.StatusBadRequest)
		return
	}

	if err := h.saveSetupStep(r.Context(), authCtx.UserID, step, req); err != nil {
		var validationErr *setupValidationError
		if strings.Contains(err.Error(), "Unsupported setup step") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if ok := errorsAs(err, &validationErr); ok {
			http.Error(w, validationErr.Error(), http.StatusBadRequest)
			return
		}
		h.logger.Error("Failed to save setup step", zap.String("step", step), zap.Error(err))
		http.Error(w, "Failed to save setup step", http.StatusInternalServerError)
		return
	}

	_ = h.service.MarkSetupInProgress(r.Context())

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "step": step})
}

// SaveAll handles POST /api/v1/bootstrap/save-all
func (h *BootstrapHandler) SaveAll(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil || !authCtx.IsSystemAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req BootstrapSaveAllRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	required := map[string]map[string]interface{}{
		"general":          req.General,
		"smtp":             req.SMTP,
		"ldap":             req.LDAP,
		"runtime_services": req.RuntimeServices,
	}
	for step, cfg := range required {
		if len(cfg) == 0 {
			http.Error(w, "Missing required config section: "+step, http.StatusBadRequest)
			return
		}
		if err := h.saveSetupStep(r.Context(), authCtx.UserID, step, BootstrapSaveStepRequest{
			Config:    cfg,
			ConfigKey: "ldap_active_directory",
		}); err != nil {
			var validationErr *setupValidationError
			if ok := errorsAs(err, &validationErr); ok {
				http.Error(w, "Invalid "+step+" config: "+validationErr.Error(), http.StatusBadRequest)
				return
			}
			h.logger.Error("Failed to save required setup section", zap.String("step", step), zap.Error(err))
			http.Error(w, "Failed to save required section: "+step, http.StatusInternalServerError)
			return
		}
	}

	skipped := make([]string, 0)
	if shouldPersistExternalService(req.ExternalService) {
		if err := h.saveSetupStep(r.Context(), authCtx.UserID, "external_services", BootstrapSaveStepRequest{
			Config: req.ExternalService,
		}); err != nil {
			h.logger.Warn("Skipping external service save during bootstrap save-all", zap.Error(err))
			skipped = append(skipped, "external_services")
		}
	} else {
		skipped = append(skipped, "external_services")
	}

	if ssoReq, ok := buildSSOSaveRequest(req); ok {
		if err := h.saveSetupStep(r.Context(), authCtx.UserID, "sso", ssoReq); err != nil {
			h.logger.Warn("Skipping SSO save during bootstrap save-all", zap.Error(err))
			skipped = append(skipped, "sso")
		}
	} else {
		skipped = append(skipped, "sso")
	}

	_ = h.service.MarkSetupInProgress(r.Context())

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"saved_steps": []string{
			"general", "smtp", "ldap", "runtime_services",
		},
		"skipped_optional_steps": skipped,
	})
}

// CompleteSetup handles POST /api/v1/bootstrap/complete
func (h *BootstrapHandler) CompleteSetup(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil || !authCtx.IsSystemAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	required, _, missing := h.getSetupStepProgress(r.Context())
	if len(missing) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":        false,
			"error":          "setup is incomplete",
			"required_steps": required,
			"missing_steps":  missing,
		})
		return
	}

	if err := h.service.MarkSetupComplete(r.Context()); err != nil {
		h.logger.Error("Failed to mark setup complete", zap.Error(err))
		http.Error(w, "Failed to complete bootstrap setup", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          true,
		"status":           appbootstrap.StatusSetupComplete,
		"restart_required": true,
	})
}

func (h *BootstrapHandler) getSetupStepProgress(ctx context.Context) (required []string, completed []string, missing []string) {
	required = []string{"general", "smtp", "ldap", "runtime_services"}

	if cfg, err := h.systemConfigService.GetConfigByTypeAndKey(ctx, nil, systemconfig.ConfigTypeGeneral, "general"); err == nil && cfg.IsActive() {
		completed = append(completed, "general")
	}
	if cfg, err := h.systemConfigService.GetConfigByTypeAndKey(ctx, nil, systemconfig.ConfigTypeSMTP, "smtp"); err == nil && cfg.IsActive() {
		if smtpCfg, parseErr := cfg.GetSMTPConfig(); parseErr == nil && smtpCfg.Enabled {
			completed = append(completed, "smtp")
		}
	}
	if ldapConfigs, err := h.systemConfigService.GetConfigsByType(ctx, nil, systemconfig.ConfigTypeLDAP); err == nil {
		for _, cfg := range ldapConfigs {
			ldapCfg, parseErr := cfg.GetLDAPConfig()
			if parseErr == nil && ldapCfg.Enabled && cfg.IsActive() {
				completed = append(completed, "ldap")
				break
			}
		}
	}
	if serviceConfigs, err := h.systemConfigService.GetConfigsByType(ctx, nil, systemconfig.ConfigTypeExternalServices); err == nil {
		for _, cfg := range serviceConfigs {
			serviceCfg, parseErr := cfg.GetExternalServiceConfig()
			if parseErr == nil && serviceCfg.Enabled && cfg.IsActive() {
				completed = append(completed, "external_services")
				break
			}
		}
	}
	if cfg, err := h.systemConfigService.GetConfigByTypeAndKey(ctx, nil, systemconfig.ConfigTypeRuntimeServices, "runtime_services"); err == nil && cfg.IsActive() {
		completed = append(completed, "runtime_services")
	}
	if h.ssoService != nil {
		ssoComplete := false
		if oidcProviders, err := h.ssoService.GetAllOIDCProviders(ctx); err == nil {
			for _, p := range oidcProviders {
				if p.Enabled {
					ssoComplete = true
					break
				}
			}
		}
		if !ssoComplete {
			if samlProviders, err := h.ssoService.GetAllSAMLProviders(ctx); err == nil {
				for _, p := range samlProviders {
					if p.Enabled {
						ssoComplete = true
						break
					}
				}
			}
		}
		if ssoComplete {
			completed = append(completed, "sso")
		}
	}

	completedSet := map[string]bool{}
	for _, c := range completed {
		completedSet[c] = true
	}
	for _, step := range required {
		if !completedSet[step] {
			missing = append(missing, step)
		}
	}
	return required, completed, missing
}

func (h *BootstrapHandler) enforceSingleActiveLDAPProvider(ctx context.Context, tenantID *uuid.UUID, activeConfigID uuid.UUID, updatedBy uuid.UUID) error {
	configs, err := h.systemConfigService.GetConfigsByType(ctx, tenantID, systemconfig.ConfigTypeLDAP)
	if err != nil {
		return err
	}

	for _, cfg := range configs {
		if cfg.ID() == activeConfigID {
			continue
		}
		ldapCfg, err := cfg.GetLDAPConfig()
		if err != nil || !ldapCfg.Enabled {
			continue
		}
		ldapCfg.Enabled = false
		if _, err := h.systemConfigService.UpdateConfig(ctx, systemconfig.UpdateConfigRequest{
			ID:          cfg.ID(),
			ConfigValue: ldapCfg,
			UpdatedBy:   updatedBy,
		}); err != nil {
			return err
		}
	}
	return nil
}

func decodeMapInto(input map[string]interface{}, out interface{}) bool {
	raw, err := json.Marshal(input)
	if err != nil {
		return false
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return false
	}
	return true
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "service"
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "service"
	}
	return out
}

func validateGeneralSetupConfig(cfg *systemconfig.GeneralConfig) error {
	if cfg == nil {
		return errBadSetupPayload("general config is required")
	}
	if strings.TrimSpace(cfg.SystemName) == "" {
		return errBadSetupPayload("system_name is required")
	}
	if strings.TrimSpace(cfg.AdminEmail) == "" {
		return errBadSetupPayload("admin_email is required")
	}
	if strings.TrimSpace(cfg.TimeZone) == "" {
		return errBadSetupPayload("time_zone is required")
	}
	return nil
}

func validateSMTPSetupConfig(cfg *systemconfig.SMTPConfig) error {
	if cfg == nil {
		return errBadSetupPayload("smtp config is required")
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return errBadSetupPayload("smtp host is required")
	}
	if cfg.Port <= 0 {
		return errBadSetupPayload("smtp port must be greater than 0")
	}
	if strings.TrimSpace(cfg.From) == "" {
		return errBadSetupPayload("smtp from address is required")
	}
	return nil
}

func validateLDAPSetupConfig(cfg *systemconfig.LDAPConfig) error {
	if cfg == nil {
		return errBadSetupPayload("ldap config is required")
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return errBadSetupPayload("ldap host is required")
	}
	if cfg.Port <= 0 {
		return errBadSetupPayload("ldap port must be greater than 0")
	}
	if strings.TrimSpace(cfg.BaseDN) == "" {
		return errBadSetupPayload("ldap base_dn is required")
	}
	if strings.TrimSpace(cfg.UserFilter) == "" {
		return errBadSetupPayload("ldap user_filter is required")
	}
	return nil
}

func validateExternalServiceSetupConfig(cfg *systemconfig.ExternalServiceConfig) error {
	if cfg == nil {
		return errBadSetupPayload("external service config is required")
	}
	if strings.TrimSpace(cfg.Name) == "" {
		return errBadSetupPayload("external service name is required")
	}
	if strings.TrimSpace(cfg.URL) == "" {
		return errBadSetupPayload("external service url is required")
	}
	return nil
}

func validateRuntimeServicesSetupConfig(cfg *systemconfig.RuntimeServicesConfig) error {
	if cfg == nil {
		return errBadSetupPayload("runtime services config is required")
	}
	if strings.TrimSpace(cfg.DispatcherURL) == "" {
		return errBadSetupPayload("dispatcher_url is required")
	}
	if cfg.DispatcherPort <= 0 {
		return errBadSetupPayload("dispatcher_port must be greater than 0")
	}
	if strings.TrimSpace(cfg.EmailWorkerURL) == "" {
		return errBadSetupPayload("email_worker_url is required")
	}
	if cfg.EmailWorkerPort <= 0 {
		return errBadSetupPayload("email_worker_port must be greater than 0")
	}
	if strings.TrimSpace(cfg.NotificationWorkerURL) == "" {
		return errBadSetupPayload("notification_worker_url is required")
	}
	if cfg.NotificationWorkerPort <= 0 {
		return errBadSetupPayload("notification_worker_port must be greater than 0")
	}
	if cfg.HealthCheckTimeoutSecond <= 0 {
		return errBadSetupPayload("health_check_timeout_seconds must be greater than 0")
	}
	if cfg.ProviderReadinessWatcherIntervalSeconds > 0 && cfg.ProviderReadinessWatcherIntervalSeconds < 30 {
		return errBadSetupPayload("provider_readiness_watcher_interval_seconds must be at least 30")
	}
	if cfg.ProviderReadinessWatcherTimeoutSeconds > 0 && cfg.ProviderReadinessWatcherTimeoutSeconds < 10 {
		return errBadSetupPayload("provider_readiness_watcher_timeout_seconds must be at least 10")
	}
	if cfg.ProviderReadinessWatcherIntervalSeconds > 0 &&
		cfg.ProviderReadinessWatcherTimeoutSeconds > 0 &&
		cfg.ProviderReadinessWatcherTimeoutSeconds >= cfg.ProviderReadinessWatcherIntervalSeconds {
		return errBadSetupPayload("provider_readiness_watcher_timeout_seconds must be less than provider_readiness_watcher_interval_seconds")
	}
	if cfg.ProviderReadinessWatcherBatchSize > 0 &&
		(cfg.ProviderReadinessWatcherBatchSize < 1 || cfg.ProviderReadinessWatcherBatchSize > 1000) {
		return errBadSetupPayload("provider_readiness_watcher_batch_size must be between 1 and 1000")
	}
	if strings.TrimSpace(cfg.TenantAssetReconcilePolicy) != "" {
		policy := strings.TrimSpace(strings.ToLower(cfg.TenantAssetReconcilePolicy))
		if policy != "full_reconcile_on_prepare" && policy != "async_trigger_only" && policy != "manual_only" {
			return errBadSetupPayload("tenant_asset_reconcile_policy must be one of full_reconcile_on_prepare, async_trigger_only, manual_only")
		}
	}
	if cfg.TenantAssetDriftWatcherIntervalSeconds > 0 && cfg.TenantAssetDriftWatcherIntervalSeconds < 30 {
		return errBadSetupPayload("tenant_asset_drift_watcher_interval_seconds must be at least 30")
	}
	if err := validateRuntimeStorageProfileSetup("storage_profiles.internal_registry", cfg.StorageProfiles.InternalRegistry); err != nil {
		return errBadSetupPayload(err.Error())
	}
	if err := validateRuntimeStorageProfileSetup("storage_profiles.trivy_cache", cfg.StorageProfiles.TrivyCache); err != nil {
		return errBadSetupPayload(err.Error())
	}
	return nil
}

func validateRuntimeStorageProfileSetup(prefix string, profile systemconfig.RuntimeAssetStorageProfile) error {
	storageType := strings.ToLower(strings.TrimSpace(profile.Type))
	switch storageType {
	case "", "hostpath", "pvc", "emptydir":
	default:
		return fmt.Errorf("%s.type must be one of hostPath, pvc, emptyDir", prefix)
	}
	switch storageType {
	case "hostpath":
		if strings.TrimSpace(profile.HostPath) == "" {
			return fmt.Errorf("%s.host_path is required when type is hostPath", prefix)
		}
	case "pvc":
		if strings.TrimSpace(profile.PVCName) == "" {
			return fmt.Errorf("%s.pvc_name is required when type is pvc", prefix)
		}
	}
	return nil
}

func errBadSetupPayload(message string) error {
	return &setupValidationError{Message: message}
}

func envOr(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envOrAny(fallback string, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envIntOrAny(fallback int, keys ...string) int {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func envBoolOr(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func envBoolOrAny(fallback bool, keys ...string) bool {
	for _, key := range keys {
		value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
		if value == "" {
			continue
		}
		switch value {
		case "1", "true", "yes", "y", "on":
			return true
		case "0", "false", "no", "n", "off":
			return false
		}
	}
	return fallback
}

func envCSVOr(key string, fallback []string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(strings.ToLower(strings.TrimPrefix(p, "@")))
		if v == "" {
			continue
		}
		values = append(values, v)
	}
	if len(values) == 0 {
		return fallback
	}
	return values
}

func envCSVOrAny(fallback []string, keys ...string) []string {
	for _, key := range keys {
		if values := envCSVOr(key, nil); len(values) > 0 {
			return values
		}
	}
	return fallback
}

type bootstrapLDAPDefaults struct {
	Host            string
	Port            int
	BaseDN          string
	UserSearchBase  string
	GroupSearchBase string
	BindDN          string
	BindPassword    string
	StartTLS        bool
	UseTLS          bool
	Enabled         bool
	AllowedDomains  []string
}

func anyEnvSet(keys ...string) bool {
	for _, key := range keys {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}
	return false
}

func resolveBootstrapLDAPDefaults() bootstrapLDAPDefaults {
	explicitLDAPConfigured := anyEnvSet(
		"IF_AUTH_LDAP_SERVER", "LDAP_HOST",
		"IF_AUTH_LDAP_BASE_DN", "LDAP_BASE_DN",
		"IF_AUTH_LDAP_BIND_DN", "LDAP_BIND_DN",
	)

	glauthHost := envOrAny("", "IF_GLAUTH_HOST", "GLAUTH_HOST", "IMAGE_FACTORY_GLAUTH_SERVICE_HOST")
	glauthPort := envIntOrAny(3893, "IF_GLAUTH_PORT", "GLAUTH_PORT", "IMAGE_FACTORY_GLAUTH_SERVICE_PORT_LDAP", "IMAGE_FACTORY_GLAUTH_SERVICE_PORT")
	glauthEnabled := envBoolOrAny(false, "IF_GLAUTH_ENABLED", "GLAUTH_ENABLED")
	glauthDetected := glauthEnabled || glauthHost != ""

	ldapServerRaw := envOrAny("", "IF_AUTH_LDAP_SERVER", "LDAP_HOST")
	if ldapServerRaw == "" && glauthDetected {
		ldapServerRaw = fmt.Sprintf("ldap://%s:%d", glauthHost, glauthPort)
	}
	if ldapServerRaw == "" {
		ldapServerRaw = "localhost"
	}

	ldapHost, ldapPortFromServer, ldapTLSFromScheme := parseLDAPServer(ldapServerRaw)
	ldapPort := envIntOrAny(ldapPortFromServer, "IF_AUTH_LDAP_PORT", "LDAP_PORT")
	ldapBaseDN := envOrAny("", "IF_AUTH_LDAP_BASE_DN", "LDAP_BASE_DN")
	ldapUserSearchBase := envOrAny("", "IF_AUTH_LDAP_USER_SEARCH_BASE", "LDAP_USER_SEARCH_BASE")
	ldapGroupSearchBase := envOrAny("", "IF_AUTH_LDAP_GROUP_SEARCH_BASE", "LDAP_GROUP_SEARCH_BASE")
	ldapBindDN := envOrAny("", "IF_AUTH_LDAP_BIND_DN", "LDAP_BIND_DN")
	ldapBindPassword := envOrAny("", "IF_AUTH_LDAP_BIND_PASSWORD", "LDAP_BIND_PASSWORD")
	ldapUseTLS := envBoolOrAny(ldapTLSFromScheme, "IF_AUTH_LDAP_USE_TLS", "IF_AUTH_LDAP_USE_SSL", "LDAP_USE_TLS", "LDAP_SSL")
	ldapStartTLS := envBoolOrAny(false, "IF_AUTH_LDAP_START_TLS", "LDAP_START_TLS")
	ldapEnabled := envBoolOrAny(false, "IF_AUTH_LDAP_ENABLED", "LDAP_ENABLED")
	ldapAllowedDomains := envCSVOrAny(nil, "IF_AUTH_LDAP_ALLOWED_DOMAINS", "LDAP_ALLOWED_DOMAINS")

	// Fallbacks for in-cluster GLAuth bootstrap when explicit LDAP env is not provided.
	if !explicitLDAPConfigured && glauthDetected {
		if strings.TrimSpace(ldapBaseDN) == "" {
			ldapBaseDN = envOrAny("dc=imgfactory,dc=com", "IF_GLAUTH_BASE_DN", "GLAUTH_BASE_DN")
		}
		if strings.TrimSpace(ldapUserSearchBase) == "" && ldapBaseDN != "" {
			ldapUserSearchBase = "ou=people," + ldapBaseDN
		}
		if strings.TrimSpace(ldapGroupSearchBase) == "" && ldapBaseDN != "" {
			ldapGroupSearchBase = ldapBaseDN
		}
		if strings.TrimSpace(ldapBindDN) == "" && ldapBaseDN != "" {
			ldapBindDN = "cn=ldap_search,ou=svcaccts," + ldapBaseDN
		}
		if strings.TrimSpace(ldapBindPassword) == "" {
			ldapBindPassword = envOrAny("search_password", "IF_GLAUTH_BIND_PASSWORD", "GLAUTH_BIND_PASSWORD", "IF_GLAUTH_SEARCH_PASSWORD", "GLAUTH_SEARCH_PASSWORD")
		}
		ldapUseTLS = envBoolOrAny(false, "IF_AUTH_LDAP_USE_TLS", "IF_AUTH_LDAP_USE_SSL", "LDAP_USE_TLS", "LDAP_SSL")
		ldapStartTLS = envBoolOrAny(false, "IF_AUTH_LDAP_START_TLS", "LDAP_START_TLS")
		if !anyEnvSet("IF_AUTH_LDAP_ENABLED", "LDAP_ENABLED") {
			ldapEnabled = true
		}
	}

	if len(ldapAllowedDomains) == 0 {
		if domain := domainFromEmail(envOr("IF_SMTP_FROM_EMAIL", "")); domain != "" {
			ldapAllowedDomains = []string{domain}
		}
	}

	return bootstrapLDAPDefaults{
		Host:            ldapHost,
		Port:            ldapPort,
		BaseDN:          ldapBaseDN,
		UserSearchBase:  ldapUserSearchBase,
		GroupSearchBase: ldapGroupSearchBase,
		BindDN:          ldapBindDN,
		BindPassword:    ldapBindPassword,
		StartTLS:        ldapStartTLS,
		UseTLS:          ldapUseTLS,
		Enabled:         ldapEnabled,
		AllowedDomains:  ldapAllowedDomains,
	}
}

func parseLDAPServer(raw string) (host string, port int, ssl bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "localhost", 389, false
	}

	// Handle ldap://host:port and ldaps://host:port formats.
	if strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err == nil {
			scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
			host = parsed.Hostname()
			if p := parsed.Port(); p != "" {
				if parsedPort, convErr := strconv.Atoi(p); convErr == nil {
					port = parsedPort
				}
			}
			ssl = scheme == "ldaps"
		}
	}

	if host == "" {
		host = trimmed
		// Handle plain host:port values.
		if strings.Count(trimmed, ":") == 1 && !strings.Contains(trimmed, "]") {
			parts := strings.Split(trimmed, ":")
			if len(parts) == 2 && parts[0] != "" {
				host = parts[0]
				if parsedPort, err := strconv.Atoi(parts[1]); err == nil {
					port = parsedPort
				}
			}
		}
	}

	if port == 0 {
		if ssl {
			port = 636
		} else {
			port = 389
		}
	}
	return host, port, ssl
}

func domainFromEmail(email string) string {
	if !strings.Contains(email, "@") {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(parts[1]))
}

type setupValidationError struct {
	Message string
}

func (e *setupValidationError) Error() string {
	return e.Message
}

func (h *BootstrapHandler) saveSetupStep(ctx context.Context, userID uuid.UUID, step string, req BootstrapSaveStepRequest) error {
	switch step {
	case "general":
		var cfg systemconfig.GeneralConfig
		if !decodeMapInto(req.Config, &cfg) {
			return errBadSetupPayload("Invalid general config payload")
		}
		if err := validateGeneralSetupConfig(&cfg); err != nil {
			return err
		}
		_, err := h.systemConfigService.CreateOrUpdateCategoryConfig(ctx, nil, systemconfig.ConfigTypeGeneral, "general", cfg, userID)
		return err
	case "smtp":
		var cfg systemconfig.SMTPConfig
		if !decodeMapInto(req.Config, &cfg) {
			return errBadSetupPayload("Invalid SMTP config payload")
		}
		if err := validateSMTPSetupConfig(&cfg); err != nil {
			return err
		}
		_, err := h.systemConfigService.CreateOrUpdateCategoryConfig(ctx, nil, systemconfig.ConfigTypeSMTP, "smtp", cfg, userID)
		return err
	case "ldap":
		var cfg systemconfig.LDAPConfig
		if !decodeMapInto(req.Config, &cfg) {
			return errBadSetupPayload("Invalid LDAP config payload")
		}
		if err := validateLDAPSetupConfig(&cfg); err != nil {
			return err
		}
		configKey := strings.TrimSpace(req.ConfigKey)
		if configKey == "" {
			configKey = "ldap_active_directory"
		}
		saved, err := h.systemConfigService.CreateOrUpdateCategoryConfig(ctx, nil, systemconfig.ConfigTypeLDAP, configKey, cfg, userID)
		if err != nil {
			return err
		}
		if cfg.Enabled {
			if err := h.enforceSingleActiveLDAPProvider(ctx, nil, saved.ID(), userID); err != nil {
				return err
			}
		}
		return nil
	case "external_services":
		var cfg systemconfig.ExternalServiceConfig
		if !decodeMapInto(req.Config, &cfg) {
			return errBadSetupPayload("Invalid external service payload")
		}
		if err := validateExternalServiceSetupConfig(&cfg); err != nil {
			return err
		}
		configKey := strings.TrimSpace(req.ConfigKey)
		if configKey == "" {
			if cfg.Name == "" {
				return errBadSetupPayload("External service name is required")
			}
			configKey = "external_service_" + slugify(cfg.Name)
		}
		_, err := h.systemConfigService.CreateOrUpdateCategoryConfig(ctx, nil, systemconfig.ConfigTypeExternalServices, configKey, cfg, userID)
		return err
	case "sso":
		if h.ssoService == nil {
			return errBadSetupPayload("SSO service unavailable")
		}
		ssoType := strings.ToLower(strings.TrimSpace(req.Type))
		if ssoType == "" {
			ssoType = "oidc"
		}
		switch ssoType {
		case "oidc":
			var oidcReq sso.OIDCProviderCreateRequest
			if !decodeMapInto(req.Config, &oidcReq) {
				return errBadSetupPayload("Invalid OIDC payload")
			}
			_, err := h.ssoService.CreateOIDCProvider(ctx, oidcReq)
			return err
		case "saml":
			var samlReq sso.SAMLProviderCreateRequest
			if !decodeMapInto(req.Config, &samlReq) {
				return errBadSetupPayload("Invalid SAML payload")
			}
			_, err := h.ssoService.CreateSAMLProvider(ctx, samlReq)
			return err
		default:
			return errBadSetupPayload("Unsupported SSO provider type")
		}
	case "runtime_services":
		var cfg systemconfig.RuntimeServicesConfig
		if !decodeMapInto(req.Config, &cfg) {
			return errBadSetupPayload("Invalid runtime services payload")
		}
		if err := validateRuntimeServicesSetupConfig(&cfg); err != nil {
			return err
		}
		_, err := h.systemConfigService.CreateOrUpdateCategoryConfig(ctx, nil, systemconfig.ConfigTypeRuntimeServices, "runtime_services", cfg, userID)
		return err
	default:
		return errBadSetupPayload("Unsupported setup step")
	}
}

func shouldPersistExternalService(cfg map[string]interface{}) bool {
	if len(cfg) == 0 {
		return false
	}
	enabled := getBool(cfg, "enabled")
	if !enabled {
		return false
	}
	name := strings.TrimSpace(getString(cfg, "name"))
	url := strings.TrimSpace(getString(cfg, "url"))
	return name != "" && url != ""
}

func buildSSOSaveRequest(req BootstrapSaveAllRequest) (BootstrapSaveStepRequest, bool) {
	ssoType := strings.ToLower(strings.TrimSpace(req.SSOType))
	if ssoType == "" {
		ssoType = "oidc"
	}

	switch ssoType {
	case "saml":
		if len(req.SAML) == 0 || !getBool(req.SAML, "enabled") {
			return BootstrapSaveStepRequest{}, false
		}
		if strings.TrimSpace(getString(req.SAML, "entity_id")) == "" || strings.TrimSpace(getString(req.SAML, "sso_url")) == "" {
			return BootstrapSaveStepRequest{}, false
		}
		return BootstrapSaveStepRequest{Type: "saml", Config: req.SAML}, true
	default:
		if len(req.OIDC) == 0 || !getBool(req.OIDC, "enabled") {
			return BootstrapSaveStepRequest{}, false
		}
		if strings.TrimSpace(getString(req.OIDC, "issuer")) == "" || strings.TrimSpace(getString(req.OIDC, "client_id")) == "" {
			return BootstrapSaveStepRequest{}, false
		}
		return BootstrapSaveStepRequest{Type: "oidc", Config: req.OIDC}, true
	}
}

func getString(cfg map[string]interface{}, key string) string {
	value, ok := cfg[key]
	if !ok || value == nil {
		return ""
	}
	v, ok := value.(string)
	if !ok {
		return ""
	}
	return v
}

func getBool(cfg map[string]interface{}, key string) bool {
	value, ok := cfg[key]
	if !ok || value == nil {
		return false
	}
	v, ok := value.(bool)
	if !ok {
		return false
	}
	return v
}

func errorsAs(err error, target interface{}) bool {
	switch t := target.(type) {
	case **setupValidationError:
		validationErr, ok := err.(*setupValidationError)
		if !ok {
			return false
		}
		*t = validationErr
		return true
	default:
		return false
	}
}
