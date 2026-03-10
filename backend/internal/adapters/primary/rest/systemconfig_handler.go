package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// SystemConfigHandler handles system configuration HTTP requests
type SystemConfigHandler struct {
	systemConfigService *systemconfig.Service
	auditService        *audit.Service
	logger              *zap.Logger
	serverEnvironment   string
}

// NewSystemConfigHandler creates a new system configuration handler
func NewSystemConfigHandler(systemConfigService *systemconfig.Service, auditService *audit.Service, logger *zap.Logger, serverEnvironment string) *SystemConfigHandler {
	return &SystemConfigHandler{
		systemConfigService: systemConfigService,
		auditService:        auditService,
		logger:              logger,
		serverEnvironment:   serverEnvironment,
	}
}

// CreateConfigRequest represents a configuration creation request
type CreateConfigRequest struct {
	TenantID    *string     `json:"tenant_id,omitempty" validate:"omitempty,uuid"`
	ConfigType  string      `json:"config_type" validate:"required,oneof=ldap smtp general security build tool_settings messaging runtime_services"`
	ConfigKey   string      `json:"config_key" validate:"required"`
	ConfigValue interface{} `json:"config_value" validate:"required"`
	Description string      `json:"description,omitempty"`
}

// CreateConfigResponse represents a configuration creation response
type CreateConfigResponse struct {
	Config SystemConfigResponse `json:"config"`
}

// SystemConfigResponse represents system configuration information
type SystemConfigResponse struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenant_id"`
	ConfigType  string      `json:"config_type"`
	ConfigKey   string      `json:"config_key"`
	ConfigValue interface{} `json:"config_value"`
	Status      string      `json:"status"`
	Description string      `json:"description"`
	IsDefault   bool        `json:"is_default"`
	IsActive    bool        `json:"is_active"`
	CreatedBy   string      `json:"created_by"`
	UpdatedBy   string      `json:"updated_by"`
	CreatedAt   string      `json:"created_at"`
	UpdatedAt   string      `json:"updated_at"`
	Version     int         `json:"version"`
}

// ListConfigsResponse represents a list of configurations response
type ListConfigsResponse struct {
	Configs []SystemConfigResponse `json:"configs"`
	Total   int                    `json:"total"`
}

// UpdateConfigRequest represents a configuration update request
type UpdateConfigRequest struct {
	ConfigValue *interface{} `json:"config_value,omitempty"`
	Description *string      `json:"description,omitempty"`
}

// TestConnectionRequest represents a connection test request
type TestConnectionRequest struct {
	ConfigKey string `json:"config_key" validate:"required"`
}

type rebootResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (h *SystemConfigHandler) resolveTenantScope(authCtx *middleware.AuthContext, tenantIDStr string, allowAllTenants bool) (*uuid.UUID, int, string) {
	if authCtx == nil {
		return nil, http.StatusUnauthorized, "Authentication required"
	}
	tenantID := authCtx.TenantID
	if tenantIDStr == "" {
		if allowAllTenants && authCtx.IsSystemAdmin {
			return nil, 0, ""
		}
		return &tenantID, 0, ""
	}

	parsedTenantID, err := uuid.Parse(tenantIDStr)
	if err != nil || parsedTenantID == uuid.Nil {
		return nil, http.StatusBadRequest, "Invalid tenant ID"
	}
	if !authCtx.IsSystemAdmin && parsedTenantID != authCtx.TenantID {
		return nil, http.StatusForbidden, "Access denied to this tenant"
	}
	return &parsedTenantID, 0, ""
}

// Corporate LDAP scope is global-only (tenant_id must not be provided in query or body).
func enforceGlobalLDAPScope(r *http.Request, bodyTenantID *string) error {
	if strings.TrimSpace(r.URL.Query().Get("tenant_id")) != "" {
		return errors.New("LDAP configuration is global; tenant_id query parameter is not allowed")
	}
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("all_tenants")), "true") {
		return errors.New("LDAP configuration is global; all_tenants query parameter is not allowed")
	}
	if bodyTenantID != nil && strings.TrimSpace(*bodyTenantID) != "" {
		return errors.New("LDAP configuration is global; tenant_id is not allowed")
	}
	return nil
}

// Corporate Tekton scope is global-only (tenant_id must not be provided in query or body).
func enforceGlobalTektonScope(r *http.Request, bodyTenantID *string) error {
	if strings.TrimSpace(r.URL.Query().Get("tenant_id")) != "" {
		return errors.New("Tekton configuration is global; tenant_id query parameter is not allowed")
	}
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("all_tenants")), "true") {
		return errors.New("Tekton configuration is global; all_tenants query parameter is not allowed")
	}
	if bodyTenantID != nil && strings.TrimSpace(*bodyTenantID) != "" {
		return errors.New("Tekton configuration is global; tenant_id is not allowed")
	}
	return nil
}

// CreateConfig handles POST /api/v1/system-configs
func (h *SystemConfigHandler) CreateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode create config request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ConfigType == "" || req.ConfigKey == "" || req.ConfigValue == nil {
		http.Error(w, "Required fields are missing", http.StatusBadRequest)
		return
	}

	// Get authenticated user
	authCtx, _ := middleware.GetAuthContext(r)

	// Resolve tenant scope (default auth tenant, explicit override only for system admins).
	tenantIDStr := ""
	if req.TenantID != nil {
		tenantIDStr = *req.TenantID
	}
	tenantID, status, message := h.resolveTenantScope(authCtx, tenantIDStr, false)
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	// Validate config type
	configTypeValue := strings.ToLower(strings.TrimSpace(req.ConfigType))
	var configType systemconfig.ConfigType
	switch configTypeValue {
	case "ldap":
		configType = systemconfig.ConfigTypeLDAP
	case "smtp":
		configType = systemconfig.ConfigTypeSMTP
	case "general":
		configType = systemconfig.ConfigTypeGeneral
	case "security":
		configType = systemconfig.ConfigTypeSecurity
	case "build":
		configType = systemconfig.ConfigTypeBuild
	case "tekton":
		configType = systemconfig.ConfigTypeTekton
	case "tool_settings":
		configType = systemconfig.ConfigTypeToolSettings
	case "external_services":
		configType = systemconfig.ConfigTypeExternalServices
	case "messaging":
		configType = systemconfig.ConfigTypeMessaging
	case "runtime_services":
		configType = systemconfig.ConfigTypeRuntimeServices
	default:
		h.logger.Warn("Invalid config type",
			zap.String("config_type", req.ConfigType),
			zap.String("normalized", configTypeValue))
		http.Error(w, "Invalid config type", http.StatusBadRequest)
		return
	}

	if configType == systemconfig.ConfigTypeLDAP {
		if tenantErr := enforceGlobalLDAPScope(r, req.TenantID); tenantErr != nil {
			http.Error(w, tenantErr.Error(), http.StatusBadRequest)
			return
		}
		tenantID = nil
	}
	if configType == systemconfig.ConfigTypeTekton {
		if tenantErr := enforceGlobalTektonScope(r, req.TenantID); tenantErr != nil {
			http.Error(w, tenantErr.Error(), http.StatusBadRequest)
			return
		}
		tenantID = nil
	}
	if configType == systemconfig.ConfigTypeTekton {
		if tenantErr := enforceGlobalTektonScope(r, req.TenantID); tenantErr != nil {
			http.Error(w, tenantErr.Error(), http.StatusBadRequest)
			return
		}
		tenantID = nil
	}

	createdBy := authCtx.UserID

	// Create configuration
	configReq := systemconfig.CreateConfigRequest{
		TenantID:    tenantID,
		ConfigType:  configType,
		ConfigKey:   req.ConfigKey,
		ConfigValue: req.ConfigValue,
		Description: req.Description,
		CreatedBy:   createdBy,
	}

	config, err := h.systemConfigService.CreateConfig(r.Context(), configReq)
	if err != nil {
		h.logger.Error("Failed to create system config", zap.Error(err))
		if err == systemconfig.ErrConfigAlreadyExists {
			http.Error(w, "Configuration already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create configuration", http.StatusInternalServerError)
		return
	}

	// Audit log the configuration creation
	if h.auditService != nil {
		tenantID := authCtx.TenantID
		h.auditService.LogUserAction(r.Context(), tenantID, authCtx.UserID, audit.AuditEventConfigChange, "system_config", "create",
			"System configuration created", map[string]interface{}{
				"config_id":   config.ID().String(),
				"config_type": string(config.ConfigType()),
				"config_key":  config.ConfigKey(),
			})
	}

	// Convert to response format
	configResp := h.convertToResponse(config)

	response := CreateConfigResponse{
		Config: configResp,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// UpdateCategoryConfig handles POST /api/v1/system-configs/category
func (h *SystemConfigHandler) UpdateCategoryConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ConfigType  string      `json:"config_type"`
		ConfigKey   string      `json:"config_key"`
		ConfigValue interface{} `json:"config_value"`
		TenantID    *string     `json:"tenant_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update category config request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Info("Received system config category update",
		zap.String("config_type", req.ConfigType),
		zap.String("config_key", req.ConfigKey))

	// Validate required fields
	if req.ConfigType == "" || req.ConfigKey == "" || req.ConfigValue == nil {
		h.logger.Warn("System config category request missing fields",
			zap.String("config_type", req.ConfigType),
			zap.String("config_key", req.ConfigKey))
		http.Error(w, "Required fields are missing", http.StatusBadRequest)
		return
	}

	// Validate config type
	var configType systemconfig.ConfigType
	switch req.ConfigType {
	case "ldap":
		configType = systemconfig.ConfigTypeLDAP
	case "smtp":
		configType = systemconfig.ConfigTypeSMTP
	case "general":
		configType = systemconfig.ConfigTypeGeneral
	case "security":
		configType = systemconfig.ConfigTypeSecurity
	case "build":
		configType = systemconfig.ConfigTypeBuild
	case "tekton":
		configType = systemconfig.ConfigTypeTekton
	case "tool_settings":
		configType = systemconfig.ConfigTypeToolSettings
	case "external_services":
		configType = systemconfig.ConfigTypeExternalServices
	case "messaging":
		configType = systemconfig.ConfigTypeMessaging
	case "runtime_services":
		configType = systemconfig.ConfigTypeRuntimeServices
	default:
		http.Error(w, "Invalid config type", http.StatusBadRequest)
		return
	}

	// Get authenticated user
	authCtx, _ := middleware.GetAuthContext(r)

	// Resolve tenant scope (default auth tenant, explicit override only for system admins).
	tenantIDStr := ""
	if req.TenantID != nil {
		tenantIDStr = *req.TenantID
	}
	tenantID, status, message := h.resolveTenantScope(authCtx, tenantIDStr, false)
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	if configType == systemconfig.ConfigTypeLDAP {
		if tenantErr := enforceGlobalLDAPScope(r, req.TenantID); tenantErr != nil {
			http.Error(w, tenantErr.Error(), http.StatusBadRequest)
			return
		}
		tenantID = nil
	}

	createdBy := authCtx.UserID

	// Create or update category configuration
	config, err := h.systemConfigService.CreateOrUpdateCategoryConfig(r.Context(), tenantID, configType, req.ConfigKey, req.ConfigValue, createdBy)
	if err != nil {
		var validationErr *systemconfig.ValidationError
		if errors.As(err, &validationErr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":        "validation_failed",
				"message":      validationErr.Error(),
				"field_errors": validationErr.FieldErrors,
			})
			return
		}
		h.logger.Error("Failed to create/update category config", zap.Error(err))
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	configResp := h.convertToResponse(config)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configResp)
}

// GetConfig handles GET /api/v1/system-configs/{id}
func (h *SystemConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	idStr := pathParts[len(pathParts)-1]
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid configuration ID", http.StatusBadRequest)
		return
	}

	config, err := h.systemConfigService.GetConfig(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get system config", zap.Error(err))
		if err == systemconfig.ErrConfigNotFound {
			http.Error(w, "Configuration not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get configuration", http.StatusInternalServerError)
		return
	}

	configResp := h.convertToResponse(config)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configResp)
}

// ListConfigs handles GET /api/v1/system-configs
func (h *SystemConfigHandler) ListConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, _ := middleware.GetAuthContext(r)
	tenantIDPtr, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	var err error

	// Get optional config type filter
	configTypeStr := r.URL.Query().Get("config_type")
	var configs []*systemconfig.SystemConfig

	if configTypeStr != "" {
		var configType systemconfig.ConfigType
		switch configTypeStr {
		case "ldap":
			configType = systemconfig.ConfigTypeLDAP
		case "smtp":
			configType = systemconfig.ConfigTypeSMTP
		case "general":
			configType = systemconfig.ConfigTypeGeneral
		case "security":
			configType = systemconfig.ConfigTypeSecurity
		case "build":
			configType = systemconfig.ConfigTypeBuild
		case "tekton":
			configType = systemconfig.ConfigTypeTekton
		case "tool_settings":
			configType = systemconfig.ConfigTypeToolSettings
		case "external_services":
			configType = systemconfig.ConfigTypeExternalServices
		case "runtime_services":
			configType = systemconfig.ConfigTypeRuntimeServices
		default:
			http.Error(w, "Invalid config type", http.StatusBadRequest)
			return
		}
		if configType == systemconfig.ConfigTypeLDAP {
			if tenantErr := enforceGlobalLDAPScope(r, nil); tenantErr != nil {
				http.Error(w, tenantErr.Error(), http.StatusBadRequest)
				return
			}
			tenantIDPtr = nil
		}
		if configType == systemconfig.ConfigTypeTekton {
			if tenantErr := enforceGlobalTektonScope(r, nil); tenantErr != nil {
				http.Error(w, tenantErr.Error(), http.StatusBadRequest)
				return
			}
			tenantIDPtr = nil
		}

		configs, err = h.systemConfigService.GetConfigsByType(r.Context(), tenantIDPtr, configType)
	} else {
		configs, err = h.systemConfigService.GetAllConfigs(r.Context(), tenantIDPtr)
	}

	if err != nil {
		h.logger.Error("Failed to list system configs", zap.Error(err))
		http.Error(w, "Failed to list configurations", http.StatusInternalServerError)
		return
	}

	configResponses := make([]SystemConfigResponse, len(configs))
	for i, config := range configs {
		configResponses[i] = h.convertToResponse(config)
	}

	response := ListConfigsResponse{
		Configs: configResponses,
		Total:   len(configResponses),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateConfig handles PUT /api/v1/system-configs/{id}
func (h *SystemConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	idStr := pathParts[len(pathParts)-1]
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid configuration ID", http.StatusBadRequest)
		return
	}

	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update config request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get authenticated user
	authCtx, _ := middleware.GetAuthContext(r)
	updatedBy := authCtx.UserID

	updateReq := systemconfig.UpdateConfigRequest{
		ID:          id,
		ConfigValue: req.ConfigValue,
		Description: req.Description,
		UpdatedBy:   updatedBy,
	}

	config, err := h.systemConfigService.UpdateConfig(r.Context(), updateReq)
	if err != nil {
		h.logger.Error("Failed to update system config", zap.Error(err))
		if err == systemconfig.ErrConfigNotFound {
			http.Error(w, "Configuration not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to update configuration", http.StatusInternalServerError)
		return
	}

	configResp := h.convertToResponse(config)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configResp)
}

// DeleteConfig handles DELETE /api/v1/system-configs/{id}
func (h *SystemConfigHandler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	idStr := pathParts[len(pathParts)-1]
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid configuration ID", http.StatusBadRequest)
		return
	}

	err = h.systemConfigService.DeleteConfig(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to delete system config", zap.Error(err))
		if err == systemconfig.ErrConfigNotFound {
			http.Error(w, "Configuration not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete configuration", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ActivateConfig handles POST /api/v1/system-configs/{id}/activate
func (h *SystemConfigHandler) ActivateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 || pathParts[len(pathParts)-1] != "activate" {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	idStr := pathParts[len(pathParts)-2]
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid configuration ID", http.StatusBadRequest)
		return
	}

	// Get authenticated user
	authCtx, _ := middleware.GetAuthContext(r)
	updatedBy := authCtx.UserID

	config, err := h.systemConfigService.ActivateConfig(r.Context(), id, updatedBy)
	if err != nil {
		h.logger.Error("Failed to activate system config", zap.Error(err))
		if err == systemconfig.ErrConfigNotFound {
			http.Error(w, "Configuration not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to activate configuration", http.StatusInternalServerError)
		return
	}

	configResp := h.convertToResponse(config)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configResp)
}

// DeactivateConfig handles POST /api/v1/system-configs/{id}/deactivate
func (h *SystemConfigHandler) DeactivateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 || pathParts[len(pathParts)-1] != "deactivate" {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	idStr := pathParts[len(pathParts)-2]
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid configuration ID", http.StatusBadRequest)
		return
	}

	// Get authenticated user
	authCtx, _ := middleware.GetAuthContext(r)
	updatedBy := authCtx.UserID

	config, err := h.systemConfigService.DeactivateConfig(r.Context(), id, updatedBy)
	if err != nil {
		h.logger.Error("Failed to deactivate system config", zap.Error(err))
		if err == systemconfig.ErrConfigNotFound {
			http.Error(w, "Configuration not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to deactivate configuration", http.StatusInternalServerError)
		return
	}

	configResp := h.convertToResponse(config)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configResp)
}

// TestConnection handles POST /api/v1/system-configs/test-connection
func (h *SystemConfigHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TestConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode test connection request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ConfigKey == "" {
		http.Error(w, "Config key is required", http.StatusBadRequest)
		return
	}

	authCtx, _ := middleware.GetAuthContext(r)
	tenantPtr, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	// Get configuration to determine type
	config, err := h.systemConfigService.GetConfigByKey(r.Context(), tenantPtr, req.ConfigKey)
	if err != nil {
		h.logger.Error("Failed to get config for testing", zap.Error(err))
		if err == systemconfig.ErrConfigNotFound {
			http.Error(w, "Configuration not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get configuration", http.StatusInternalServerError)
		return
	}

	// Test connection based on config type
	switch config.ConfigType() {
	case systemconfig.ConfigTypeLDAP:
		if tenantErr := enforceGlobalLDAPScope(r, nil); tenantErr != nil {
			http.Error(w, tenantErr.Error(), http.StatusBadRequest)
			return
		}
		tenantPtr = nil
		err = h.systemConfigService.TestLDAPConnection(r.Context(), tenantPtr, req.ConfigKey)
	case systemconfig.ConfigTypeSMTP:
		err = h.systemConfigService.TestSMTPConnection(r.Context(), tenantPtr, req.ConfigKey)
	case systemconfig.ConfigTypeExternalServices:
		err = h.systemConfigService.TestExternalServiceConnection(r.Context(), tenantPtr, req.ConfigKey)
	default:
		http.Error(w, "Connection testing not supported for this config type", http.StatusBadRequest)
		return
	}

	if err != nil {
		h.logger.Error("Connection test failed", zap.Error(err))
		response := map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Connection test successful",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTektonTaskImages handles GET /api/v1/admin/settings/tekton-task-images
func (h *SystemConfigHandler) GetTektonTaskImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg, err := h.systemConfigService.GetTektonTaskImagesConfig(r.Context())
	if err != nil {
		h.logger.Error("Failed to get tekton task images config", zap.Error(err))
		http.Error(w, "Failed to get tekton task images configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// UpdateTektonTaskImages handles PUT /api/v1/admin/settings/tekton-task-images
func (h *SystemConfigHandler) UpdateTektonTaskImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req systemconfig.TektonTaskImagesConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update tekton task images request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil || authCtx.UserID == uuid.Nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	updatedConfig, err := h.systemConfigService.UpdateTektonTaskImagesConfig(r.Context(), &req, authCtx.UserID)
	if err != nil {
		h.logger.Error("Failed to update tekton task images config", zap.Error(err))
		http.Error(w, "Failed to update tekton task images configuration", http.StatusBadRequest)
		return
	}

	if h.auditService != nil {
		auditTenantID := authCtx.TenantID
		auditData := map[string]interface{}{
			"is_global": true,
		}
		h.auditService.LogUserAction(r.Context(), auditTenantID, authCtx.UserID, audit.AuditEventConfigChange, "tekton_task_images", "update",
			"Tekton task images configuration updated", auditData)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedConfig)
}

// GetToolAvailability handles GET /api/v1/admin/settings/tools
func (h *SystemConfigHandler) GetToolAvailability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, _ := middleware.GetAuthContext(r)
	tenantIDPtr, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	// Get tool availability configuration
	toolConfig, err := h.systemConfigService.GetToolAvailabilityConfig(r.Context(), tenantIDPtr)
	if err != nil {
		h.logger.Error("Failed to get tool availability config", zap.Error(err))
		http.Error(w, "Failed to get tool availability configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toolConfig)
}

// UpdateToolAvailability handles PUT /api/v1/admin/settings/tools
func (h *SystemConfigHandler) UpdateToolAvailability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req systemconfig.ToolAvailabilityConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update tool availability request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	authCtx, _ := middleware.GetAuthContext(r)
	tenantIDPtr, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	updatedBy := authCtx.UserID

	// Update tool availability configuration
	updatedConfig, err := h.systemConfigService.UpdateToolAvailabilityConfig(r.Context(), tenantIDPtr, &req, updatedBy)
	if err != nil {
		h.logger.Error("Failed to update tool availability config", zap.Error(err))
		if err == systemconfig.ErrInvalidToolAvailabilityConfig {
			http.Error(w, "Invalid tool availability configuration", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to update tool availability configuration", http.StatusInternalServerError)
		return
	}

	// Audit log the configuration change
	if h.auditService != nil {
		auditTenantID := authCtx.TenantID // Use the original tenantID from auth context for audit
		auditData := map[string]interface{}{
			"is_global": false,
		}
		h.auditService.LogUserAction(r.Context(), auditTenantID, authCtx.UserID, audit.AuditEventConfigChange, "tool_availability", "update",
			"Tool availability configuration updated", auditData)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedConfig)
}

// GetBuildCapabilities handles GET /api/v1/admin/settings/build-capabilities
func (h *SystemConfigHandler) GetBuildCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, _ := middleware.GetAuthContext(r)
	tenantIDPtr, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	buildCapabilities, err := h.systemConfigService.GetBuildCapabilitiesConfig(r.Context(), tenantIDPtr)
	if err != nil {
		h.logger.Error("Failed to get build capabilities config", zap.Error(err))
		http.Error(w, "Failed to get build capabilities configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(buildCapabilities)
}

// UpdateBuildCapabilities handles PUT /api/v1/admin/settings/build-capabilities
func (h *SystemConfigHandler) UpdateBuildCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req systemconfig.BuildCapabilitiesConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update build capabilities request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	authCtx, _ := middleware.GetAuthContext(r)
	tenantIDPtr, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	updatedBy := authCtx.UserID

	updatedConfig, err := h.systemConfigService.UpdateBuildCapabilitiesConfig(r.Context(), tenantIDPtr, &req, updatedBy)
	if err != nil {
		h.logger.Error("Failed to update build capabilities config", zap.Error(err))
		http.Error(w, "Failed to update build capabilities configuration", http.StatusInternalServerError)
		return
	}

	if h.auditService != nil {
		auditTenantID := authCtx.TenantID
		auditData := map[string]interface{}{
			"is_global": false,
		}
		h.auditService.LogUserAction(r.Context(), auditTenantID, authCtx.UserID, audit.AuditEventConfigChange, "build_capabilities", "update",
			"Build capabilities configuration updated", auditData)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedConfig)
}

// GetOperationCapabilities handles GET /api/v1/admin/settings/operation-capabilities
func (h *SystemConfigHandler) GetOperationCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, _ := middleware.GetAuthContext(r)
	tenantIDPtr, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	operationCapabilities, err := h.systemConfigService.GetOperationCapabilitiesConfig(r.Context(), tenantIDPtr)
	if err != nil {
		h.logger.Error("Failed to get operation capabilities config", zap.Error(err))
		http.Error(w, "Failed to get operation capabilities configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(operationCapabilities)
}

// UpdateOperationCapabilities handles PUT /api/v1/admin/settings/operation-capabilities
func (h *SystemConfigHandler) UpdateOperationCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req systemconfig.OperationCapabilitiesConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update operation capabilities request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	authCtx, _ := middleware.GetAuthContext(r)
	tenantIDPtr, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	updatedBy := authCtx.UserID
	changeReason := strings.TrimSpace(r.URL.Query().Get("change_reason"))
	if changeReason == "" {
		changeReason = "unspecified"
	}

	previousConfig, previousErr := h.systemConfigService.GetOperationCapabilitiesConfig(r.Context(), tenantIDPtr)
	if previousErr != nil {
		h.logger.Warn("Failed to fetch previous operation capabilities config for audit delta", zap.Error(previousErr))
	}

	updatedConfig, err := h.systemConfigService.UpdateOperationCapabilitiesConfig(r.Context(), tenantIDPtr, &req, updatedBy)
	if err != nil {
		h.logger.Error("Failed to update operation capabilities config", zap.Error(err))
		http.Error(w, "Failed to update operation capabilities configuration", http.StatusInternalServerError)
		return
	}

	if h.auditService != nil {
		auditTenantID := authCtx.TenantID
		auditData := map[string]interface{}{
			"is_global":     tenantIDPtr == nil,
			"change_reason": changeReason,
			"target_tenant": "",
			"before_config": previousConfig,
			"after_config":  updatedConfig,
		}
		if tenantIDPtr != nil {
			auditData["target_tenant"] = tenantIDPtr.String()
		}
		h.auditService.LogUserAction(r.Context(), auditTenantID, authCtx.UserID, audit.AuditEventConfigChange, "operation_capabilities", "update",
			"Operation capabilities configuration updated", auditData)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedConfig)
}

// GetCapabilitySurfaces handles GET /api/v1/settings/capability-surfaces
// and /api/v1/admin/settings/capability-surfaces.
func (h *SystemConfigHandler) GetCapabilitySurfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, _ := middleware.GetAuthContext(r)
	tenantIDPtr, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	if tenantIDPtr == nil || *tenantIDPtr == uuid.Nil {
		http.Error(w, "Tenant context required", http.StatusBadRequest)
		return
	}

	response, err := h.systemConfigService.GetCapabilitySurfaces(r.Context(), *tenantIDPtr)
	if err != nil {
		h.logger.Error("Failed to get capability surfaces", zap.Error(err))
		http.Error(w, "Failed to get capability surfaces", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetQuarantinePolicy handles GET /api/v1/admin/settings/quarantine-policy
func (h *SystemConfigHandler) GetQuarantinePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, _ := middleware.GetAuthContext(r)
	tenantIDPtr, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	policy, err := h.systemConfigService.GetQuarantinePolicyConfig(r.Context(), tenantIDPtr)
	if err != nil {
		h.logger.Error("Failed to get quarantine policy config", zap.Error(err))
		http.Error(w, "Failed to get quarantine policy configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policy)
}

// UpdateQuarantinePolicy handles PUT /api/v1/admin/settings/quarantine-policy
func (h *SystemConfigHandler) UpdateQuarantinePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req systemconfig.QuarantinePolicyConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update quarantine policy request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	authCtx, _ := middleware.GetAuthContext(r)
	tenantIDPtr, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	updatedBy := authCtx.UserID

	updatedConfig, err := h.systemConfigService.UpdateQuarantinePolicyConfig(r.Context(), tenantIDPtr, &req, updatedBy)
	if err != nil {
		h.logger.Error("Failed to update quarantine policy config", zap.Error(err))
		http.Error(w, "Failed to update quarantine policy configuration", http.StatusBadRequest)
		return
	}

	if h.auditService != nil {
		auditTenantID := authCtx.TenantID
		auditData := map[string]interface{}{
			"is_global": tenantIDPtr == nil,
		}
		if tenantIDPtr != nil {
			auditData["target_tenant"] = tenantIDPtr.String()
		}
		h.auditService.LogUserAction(r.Context(), auditTenantID, authCtx.UserID, audit.AuditEventConfigChange, "quarantine_policy", "update",
			"Quarantine policy configuration updated", auditData)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedConfig)
}

// ValidateQuarantinePolicy handles POST /api/v1/admin/settings/quarantine-policy/validate
func (h *SystemConfigHandler) ValidateQuarantinePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	policy, err := decodeQuarantinePolicyRequest(r)
	if err != nil {
		h.logger.Error("Failed to decode quarantine policy validate request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result := h.systemConfigService.ValidateQuarantinePolicy(policy)
	status := http.StatusOK
	if !result.Valid {
		status = http.StatusBadRequest
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(result)
}

// SimulateQuarantinePolicy handles POST /api/v1/admin/settings/quarantine-policy/simulate
func (h *SystemConfigHandler) SimulateQuarantinePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Policy      map[string]interface{} `json:"policy"`
		ScanSummary map[string]interface{} `json:"scan_summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode quarantine policy simulate request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Policy == nil {
		http.Error(w, "policy is required", http.StatusBadRequest)
		return
	}
	if req.ScanSummary == nil {
		req.ScanSummary = map[string]interface{}{}
	}

	encodedPolicy, err := json.Marshal(req.Policy)
	if err != nil {
		http.Error(w, "Invalid policy payload", http.StatusBadRequest)
		return
	}
	var policy systemconfig.QuarantinePolicyConfig
	if err := json.Unmarshal(encodedPolicy, &policy); err != nil {
		http.Error(w, "Invalid policy payload", http.StatusBadRequest)
		return
	}

	result, err := h.systemConfigService.SimulateQuarantinePolicy(&policy, req.ScanSummary)
	if err != nil {
		h.logger.Error("Failed to simulate quarantine policy", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetSORRegistration handles GET /api/v1/admin/settings/epr-registration
func (h *SystemConfigHandler) GetSORRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, _ := middleware.GetAuthContext(r)
	tenantIDPtr, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	config, err := h.systemConfigService.GetSORRegistrationConfig(r.Context(), tenantIDPtr)
	if err != nil {
		h.logger.Error("Failed to get EPR registration config", zap.Error(err))
		http.Error(w, "Failed to get EPR registration configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// UpdateSORRegistration handles PUT /api/v1/admin/settings/epr-registration
func (h *SystemConfigHandler) UpdateSORRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req systemconfig.SORRegistrationConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update EPR registration request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	authCtx, _ := middleware.GetAuthContext(r)
	tenantIDPtr, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	updatedBy := authCtx.UserID

	updatedConfig, err := h.systemConfigService.UpdateSORRegistrationConfig(r.Context(), tenantIDPtr, &req, updatedBy)
	if err != nil {
		h.logger.Error("Failed to update EPR registration config", zap.Error(err))
		http.Error(w, "Failed to update epr registration configuration", http.StatusBadRequest)
		return
	}

	if h.auditService != nil {
		auditTenantID := authCtx.TenantID
		auditData := map[string]interface{}{
			"is_global": tenantIDPtr == nil,
		}
		if tenantIDPtr != nil {
			auditData["target_tenant"] = tenantIDPtr.String()
		}
		h.auditService.LogUserAction(r.Context(), auditTenantID, authCtx.UserID, audit.AuditEventConfigChange, "sor_registration", "update",
			"EPR registration configuration updated", auditData)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedConfig)
}

// GetReleaseGovernancePolicy handles GET /api/v1/admin/settings/release-governance-policy
func (h *SystemConfigHandler) GetReleaseGovernancePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, _ := middleware.GetAuthContext(r)
	tenantIDPtr, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	config, err := h.systemConfigService.GetReleaseGovernancePolicyConfig(r.Context(), tenantIDPtr)
	if err != nil {
		h.logger.Error("Failed to get release governance policy config", zap.Error(err))
		http.Error(w, "Failed to get release governance policy configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// UpdateReleaseGovernancePolicy handles PUT /api/v1/admin/settings/release-governance-policy
func (h *SystemConfigHandler) UpdateReleaseGovernancePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req systemconfig.ReleaseGovernancePolicyConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update release governance policy request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	authCtx, _ := middleware.GetAuthContext(r)
	tenantIDPtr, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	updatedBy := authCtx.UserID

	updatedConfig, err := h.systemConfigService.UpdateReleaseGovernancePolicyConfig(r.Context(), tenantIDPtr, &req, updatedBy)
	if err != nil {
		h.logger.Error("Failed to update release governance policy config", zap.Error(err))
		http.Error(w, "Failed to update release governance policy configuration", http.StatusBadRequest)
		return
	}

	if h.auditService != nil {
		auditTenantID := authCtx.TenantID
		auditData := map[string]interface{}{
			"is_global": tenantIDPtr == nil,
		}
		if tenantIDPtr != nil {
			auditData["target_tenant"] = tenantIDPtr.String()
		}
		h.auditService.LogUserAction(r.Context(), auditTenantID, authCtx.UserID, audit.AuditEventConfigChange, "release_governance_policy", "update",
			"Release governance policy configuration updated", auditData)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedConfig)
}

func decodeQuarantinePolicyRequest(r *http.Request) (*systemconfig.QuarantinePolicyConfig, error) {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, err
	}

	var policyRaw interface{} = payload
	if nested, ok := payload["policy"]; ok {
		policyRaw = nested
	}
	encoded, err := json.Marshal(policyRaw)
	if err != nil {
		return nil, err
	}

	var policy systemconfig.QuarantinePolicyConfig
	if err := json.Unmarshal(encoded, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// CreateExternalServiceRequest represents an external service creation request
type CreateExternalServiceRequest struct {
	TenantID    *string           `json:"tenant_id,omitempty" validate:"omitempty,uuid"`
	Name        string            `json:"name" validate:"required"`
	Description string            `json:"description,omitempty"`
	URL         string            `json:"url" validate:"required,url"`
	APIKey      string            `json:"api_key" validate:"required"`
	Headers     map[string]string `json:"headers,omitempty"`
	Enabled     bool              `json:"enabled"`
}

// UpdateExternalServiceRequest represents an external service update request
type UpdateExternalServiceRequest struct {
	Name        string            `json:"name" validate:"required"`
	Description string            `json:"description,omitempty"`
	URL         string            `json:"url" validate:"required,url"`
	APIKey      string            `json:"api_key" validate:"required"`
	Headers     map[string]string `json:"headers,omitempty"`
	Enabled     bool              `json:"enabled"`
}

// ExternalServiceResponse represents external service information
type ExternalServiceResponse struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	URL         string            `json:"url"`
	APIKey      string            `json:"api_key,omitempty"` // Include in individual responses, omit in list responses
	Headers     map[string]string `json:"headers,omitempty"`
	Enabled     bool              `json:"enabled"`
}

// CreateLDAPConfigRequest represents an LDAP configuration creation request
type CreateLDAPConfigRequest struct {
	TenantID        *string  `json:"tenant_id,omitempty" validate:"omitempty,uuid"`
	ConfigKey       string   `json:"config_key" validate:"required"`
	ProviderName    string   `json:"provider_name,omitempty"`
	ProviderType    string   `json:"provider_type,omitempty"`
	Host            string   `json:"host" validate:"required"`
	Port            int      `json:"port" validate:"required,min=1,max=65535"`
	BaseDN          string   `json:"base_dn" validate:"required"`
	UserSearchBase  string   `json:"user_search_base,omitempty"`
	GroupSearchBase string   `json:"group_search_base,omitempty"`
	BindDN          string   `json:"bind_dn,omitempty"`
	BindPassword    string   `json:"bind_password,omitempty"`
	UserFilter      string   `json:"user_filter" validate:"required"`
	GroupFilter     string   `json:"group_filter,omitempty"`
	StartTLS        bool     `json:"start_tls"`
	SSL             bool     `json:"ssl"`
	AllowedDomains  []string `json:"allowed_domains,omitempty"`
	Enabled         *bool    `json:"enabled,omitempty"`
	Description     string   `json:"description,omitempty"`
}

// UpdateLDAPConfigRequest represents an LDAP configuration update request
type UpdateLDAPConfigRequest struct {
	ProviderName    *string   `json:"provider_name,omitempty"`
	ProviderType    *string   `json:"provider_type,omitempty"`
	Host            *string   `json:"host,omitempty"`
	Port            *int      `json:"port,omitempty"`
	BaseDN          *string   `json:"base_dn,omitempty"`
	UserSearchBase  *string   `json:"user_search_base,omitempty"`
	GroupSearchBase *string   `json:"group_search_base,omitempty"`
	BindDN          *string   `json:"bind_dn,omitempty"`
	BindPassword    *string   `json:"bind_password,omitempty"`
	UserFilter      *string   `json:"user_filter,omitempty"`
	GroupFilter     *string   `json:"group_filter,omitempty"`
	StartTLS        *bool     `json:"start_tls,omitempty"`
	SSL             *bool     `json:"ssl,omitempty"`
	AllowedDomains  *[]string `json:"allowed_domains,omitempty"`
	Enabled         *bool     `json:"enabled,omitempty"`
	Description     *string   `json:"description,omitempty"`
}

// LDAPConfigResponse represents LDAP configuration information
type LDAPConfigResponse struct {
	ID              string   `json:"id"`
	TenantID        string   `json:"tenant_id"`
	ConfigKey       string   `json:"config_key"`
	ProviderName    string   `json:"provider_name,omitempty"`
	ProviderType    string   `json:"provider_type,omitempty"`
	Host            string   `json:"host"`
	Port            int      `json:"port"`
	BaseDN          string   `json:"base_dn"`
	UserSearchBase  string   `json:"user_search_base,omitempty"`
	GroupSearchBase string   `json:"group_search_base,omitempty"`
	BindDN          string   `json:"bind_dn"`
	UserFilter      string   `json:"user_filter"`
	GroupFilter     string   `json:"group_filter"`
	StartTLS        bool     `json:"start_tls"`
	SSL             bool     `json:"ssl"`
	AllowedDomains  []string `json:"allowed_domains"`
	Enabled         bool     `json:"enabled"`
	Status          string   `json:"status"`
	Description     string   `json:"description"`
	IsDefault       bool     `json:"is_default"`
	IsActive        bool     `json:"is_active"`
	CreatedBy       string   `json:"created_by"`
	UpdatedBy       string   `json:"updated_by"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
	Version         int      `json:"version"`
}

// CreateExternalService creates a new external service configuration
func (h *SystemConfigHandler) CreateExternalService(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Error("No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateExternalServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Name == "" {
		http.Error(w, "Service name is required", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "Service URL is required", http.StatusBadRequest)
		return
	}
	if req.APIKey == "" {
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	// Convert tenant ID
	var tenantID *uuid.UUID
	if req.TenantID != nil && *req.TenantID != "" {
		parsedTenantID, err := uuid.Parse(*req.TenantID)
		if err != nil {
			h.logger.Error("Invalid tenant ID", zap.Error(err))
			http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
			return
		}
		tenantID = &parsedTenantID
	}

	// Create service config
	serviceConfig := &systemconfig.ExternalServiceConfig{
		Name:        req.Name,
		Description: req.Description,
		URL:         req.URL,
		APIKey:      req.APIKey,
		Headers:     req.Headers,
		Enabled:     req.Enabled,
	}

	createdConfig, err := h.systemConfigService.CreateExternalService(r.Context(), tenantID, serviceConfig, authCtx.UserID)
	if err != nil {
		h.logger.Error("Failed to create external service", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit log
	auditTenantID := authCtx.TenantID
	if tenantID != nil {
		auditTenantID = *tenantID
	}
	auditData := map[string]interface{}{
		"service_name": createdConfig.Name,
		"service_url":  createdConfig.URL,
		"enabled":      createdConfig.Enabled,
	}
	h.auditService.LogUserAction(r.Context(), auditTenantID, authCtx.UserID, audit.AuditEventConfigChange, "external_service", "create",
		"External service configuration created", auditData)

	response := ExternalServiceResponse{
		Name:        createdConfig.Name,
		Description: createdConfig.Description,
		URL:         createdConfig.URL,
		APIKey:      createdConfig.APIKey, // Include API key in creation response
		Headers:     createdConfig.Headers,
		Enabled:     createdConfig.Enabled,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateExternalService updates an existing external service configuration
func (h *SystemConfigHandler) UpdateExternalService(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Error("No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract service name from URL path parameters
	serviceName := chi.URLParam(r, "name")
	if serviceName == "" {
		http.Error(w, "Service name required in path", http.StatusBadRequest)
		return
	}

	var req UpdateExternalServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Name == "" {
		http.Error(w, "Service name is required", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "Service URL is required", http.StatusBadRequest)
		return
	}
	if req.APIKey == "" {
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	tenantID, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	// Create service config
	serviceConfig := &systemconfig.ExternalServiceConfig{
		Name:        req.Name,
		Description: req.Description,
		URL:         req.URL,
		APIKey:      req.APIKey,
		Headers:     req.Headers,
		Enabled:     req.Enabled,
	}

	updatedConfig, err := h.systemConfigService.UpdateExternalService(r.Context(), tenantID, serviceName, serviceConfig, authCtx.UserID)
	if err != nil {
		h.logger.Error("Failed to update external service", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit log
	auditTenantID := authCtx.TenantID
	if tenantID != nil {
		auditTenantID = *tenantID
	}
	auditData := map[string]interface{}{
		"service_name": updatedConfig.Name,
		"service_url":  updatedConfig.URL,
		"enabled":      updatedConfig.Enabled,
	}
	h.auditService.LogUserAction(r.Context(), auditTenantID, authCtx.UserID, audit.AuditEventConfigChange, "external_service", "update",
		"External service configuration updated", auditData)

	response := ExternalServiceResponse{
		Name:        updatedConfig.Name,
		Description: updatedConfig.Description,
		URL:         updatedConfig.URL,
		APIKey:      updatedConfig.APIKey, // Include API key in update response
		Headers:     updatedConfig.Headers,
		Enabled:     updatedConfig.Enabled,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetExternalServices retrieves all external service configurations
func (h *SystemConfigHandler) GetExternalServices(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Error("No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tenantID, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	services, err := h.systemConfigService.GetExternalServices(r.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to get external services", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var responses []ExternalServiceResponse
	for _, service := range services {
		responses = append(responses, ExternalServiceResponse{
			Name:        service.Name,
			Description: service.Description,
			URL:         service.URL,
			Headers:     service.Headers,
			Enabled:     service.Enabled,
			// Don't include API key in list response for security
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"services": responses,
	})
}

// GetExternalService retrieves a specific external service configuration
func (h *SystemConfigHandler) GetExternalService(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Error("No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract service name from URL path parameters
	serviceName := chi.URLParam(r, "name")
	if serviceName == "" {
		http.Error(w, "Service name required in path", http.StatusBadRequest)
		return
	}

	tenantID, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	service, err := h.systemConfigService.GetExternalService(r.Context(), tenantID, serviceName)
	if err != nil {
		h.logger.Error("Failed to get external service", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := ExternalServiceResponse{
		Name:        service.Name,
		Description: service.Description,
		URL:         service.URL,
		APIKey:      service.APIKey, // Include API key for admin editing
		Headers:     service.Headers,
		Enabled:     service.Enabled,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteExternalService deletes an external service configuration
func (h *SystemConfigHandler) DeleteExternalService(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Error("No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract service name from URL path parameters
	serviceName := chi.URLParam(r, "name")
	if serviceName == "" {
		http.Error(w, "Service name required in path", http.StatusBadRequest)
		return
	}

	tenantID, status, message := h.resolveTenantScope(authCtx, r.URL.Query().Get("tenant_id"), isAllTenantsScopeRequested(r, authCtx))
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	err := h.systemConfigService.DeleteExternalService(r.Context(), tenantID, serviceName, authCtx.UserID)
	if err != nil {
		h.logger.Error("Failed to delete external service", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit log
	auditTenantID := authCtx.TenantID
	if tenantID != nil {
		auditTenantID = *tenantID
	}
	auditData := map[string]interface{}{
		"service_name": serviceName,
	}
	h.auditService.LogUserAction(r.Context(), auditTenantID, authCtx.UserID, audit.AuditEventConfigChange, "external_service", "delete",
		"External service configuration deleted", auditData)

	w.WriteHeader(http.StatusNoContent)
}

// CreateLDAPConfig creates a new LDAP configuration
func (h *SystemConfigHandler) CreateLDAPConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Error("No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateLDAPConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode create LDAP config request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ConfigKey == "" || req.Host == "" || req.BaseDN == "" || req.UserFilter == "" {
		http.Error(w, "Required fields are missing", http.StatusBadRequest)
		return
	}

	if tenantErr := enforceGlobalLDAPScope(r, req.TenantID); tenantErr != nil {
		http.Error(w, tenantErr.Error(), http.StatusBadRequest)
		return
	}
	tenantID := (*uuid.UUID)(nil)

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	// Create LDAP config value
	ldapConfig := systemconfig.LDAPConfig{
		ProviderName:    req.ProviderName,
		ProviderType:    req.ProviderType,
		Host:            req.Host,
		Port:            req.Port,
		BaseDN:          req.BaseDN,
		UserSearchBase:  req.UserSearchBase,
		GroupSearchBase: req.GroupSearchBase,
		BindDN:          req.BindDN,
		BindPassword:    req.BindPassword,
		UserFilter:      req.UserFilter,
		GroupFilter:     req.GroupFilter,
		StartTLS:        req.StartTLS,
		SSL:             req.SSL,
		AllowedDomains:  req.AllowedDomains,
		Enabled:         enabled,
	}

	config, err := h.systemConfigService.CreateOrUpdateCategoryConfig(
		r.Context(),
		tenantID,
		systemconfig.ConfigTypeLDAP,
		req.ConfigKey,
		ldapConfig,
		authCtx.UserID,
	)
	if err != nil {
		h.logger.Error("Failed to create LDAP config", zap.Error(err))
		http.Error(w, "Failed to create LDAP configuration", http.StatusInternalServerError)
		return
	}

	if ldapConfig.Enabled {
		if err := h.enforceSingleActiveLDAPProvider(r.Context(), tenantID, config.ID(), authCtx.UserID); err != nil {
			h.logger.Error("Failed to enforce single active LDAP provider", zap.Error(err))
			http.Error(w, "Failed to enforce LDAP provider activation rule", http.StatusInternalServerError)
			return
		}
	}

	// Audit log
	h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventConfigChange, "ldap_config", "create",
		"LDAP configuration created", map[string]interface{}{
			"config_id":  config.ID().String(),
			"config_key": config.ConfigKey(),
		})

	// Convert to response
	ldapResp := h.convertToLDAPResponse(config)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ldapResp)
}

// GetLDAPConfigs retrieves all LDAP configurations
func (h *SystemConfigHandler) GetLDAPConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Error("No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if tenantErr := enforceGlobalLDAPScope(r, nil); tenantErr != nil {
		http.Error(w, tenantErr.Error(), http.StatusBadRequest)
		return
	}
	tenantIDPtr := (*uuid.UUID)(nil)

	configs, err := h.systemConfigService.GetConfigsByType(r.Context(), tenantIDPtr, systemconfig.ConfigTypeLDAP)
	if err != nil {
		h.logger.Error("Failed to get LDAP configs", zap.Error(err))
		http.Error(w, "Failed to get LDAP configurations", http.StatusInternalServerError)
		return
	}

	responses := make([]LDAPConfigResponse, len(configs))
	for i, config := range configs {
		responses[i] = h.convertToLDAPResponse(config)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"configs": responses,
		"total":   len(responses),
	})
}

// GetLDAPConfig retrieves a specific LDAP configuration
func (h *SystemConfigHandler) GetLDAPConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Error("No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract config key from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		http.Error(w, "Config key required in path", http.StatusBadRequest)
		return
	}
	configKey := pathParts[len(pathParts)-1]

	if tenantErr := enforceGlobalLDAPScope(r, nil); tenantErr != nil {
		http.Error(w, tenantErr.Error(), http.StatusBadRequest)
		return
	}
	tenantID := (*uuid.UUID)(nil)

	config, err := h.systemConfigService.GetConfigByKey(r.Context(), tenantID, configKey)
	if err != nil {
		h.logger.Error("Failed to get LDAP config", zap.Error(err))
		if err == systemconfig.ErrConfigNotFound {
			http.Error(w, "LDAP configuration not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get LDAP configuration", http.StatusInternalServerError)
		return
	}

	if config.ConfigType() != systemconfig.ConfigTypeLDAP {
		http.Error(w, "Configuration is not LDAP type", http.StatusBadRequest)
		return
	}

	ldapResp := h.convertToLDAPResponse(config)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ldapResp)
}

// UpdateLDAPConfig updates an existing LDAP configuration
func (h *SystemConfigHandler) UpdateLDAPConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Error("No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract config key from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		http.Error(w, "Config key required in path", http.StatusBadRequest)
		return
	}
	configKey := pathParts[len(pathParts)-1]

	var req UpdateLDAPConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update LDAP config request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if tenantErr := enforceGlobalLDAPScope(r, nil); tenantErr != nil {
		http.Error(w, tenantErr.Error(), http.StatusBadRequest)
		return
	}
	tenantID := (*uuid.UUID)(nil)

	// Get existing config
	existingConfig, err := h.systemConfigService.GetConfigByKey(r.Context(), tenantID, configKey)
	if err != nil {
		h.logger.Error("Failed to get existing LDAP config", zap.Error(err))
		if err == systemconfig.ErrConfigNotFound {
			http.Error(w, "LDAP configuration not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get LDAP configuration", http.StatusInternalServerError)
		return
	}

	if existingConfig.ConfigType() != systemconfig.ConfigTypeLDAP {
		http.Error(w, "Configuration is not LDAP type", http.StatusBadRequest)
		return
	}

	// Get current LDAP config
	currentLDAP, err := existingConfig.GetLDAPConfig()
	if err != nil {
		h.logger.Error("Failed to get current LDAP config", zap.Error(err))
		http.Error(w, "Failed to parse current LDAP configuration", http.StatusInternalServerError)
		return
	}

	// Apply updates
	if req.ProviderName != nil {
		currentLDAP.ProviderName = *req.ProviderName
	}
	if req.ProviderType != nil {
		currentLDAP.ProviderType = *req.ProviderType
	}
	if req.Host != nil {
		currentLDAP.Host = *req.Host
	}
	if req.Port != nil {
		currentLDAP.Port = *req.Port
	}
	if req.BaseDN != nil {
		currentLDAP.BaseDN = *req.BaseDN
	}
	if req.UserSearchBase != nil {
		currentLDAP.UserSearchBase = *req.UserSearchBase
	}
	if req.GroupSearchBase != nil {
		currentLDAP.GroupSearchBase = *req.GroupSearchBase
	}
	if req.BindDN != nil {
		currentLDAP.BindDN = *req.BindDN
	}
	if req.BindPassword != nil {
		currentLDAP.BindPassword = *req.BindPassword
	}
	if req.UserFilter != nil {
		currentLDAP.UserFilter = *req.UserFilter
	}
	if req.GroupFilter != nil {
		currentLDAP.GroupFilter = *req.GroupFilter
	}
	if req.StartTLS != nil {
		currentLDAP.StartTLS = *req.StartTLS
	}
	if req.SSL != nil {
		currentLDAP.SSL = *req.SSL
	}
	if req.AllowedDomains != nil {
		currentLDAP.AllowedDomains = *req.AllowedDomains
	}
	if req.Enabled != nil {
		currentLDAP.Enabled = *req.Enabled
	}

	// Update configuration
	updateReq := systemconfig.UpdateConfigRequest{
		ID:          existingConfig.ID(),
		ConfigValue: currentLDAP,
		Description: req.Description,
		UpdatedBy:   authCtx.UserID,
	}

	updatedConfig, err := h.systemConfigService.UpdateConfig(r.Context(), updateReq)
	if err != nil {
		h.logger.Error("Failed to update LDAP config", zap.Error(err))
		if err == systemconfig.ErrConfigNotFound {
			http.Error(w, "LDAP configuration not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to update LDAP configuration", http.StatusInternalServerError)
		return
	}

	if currentLDAP.Enabled {
		if err := h.enforceSingleActiveLDAPProvider(r.Context(), tenantID, updatedConfig.ID(), authCtx.UserID); err != nil {
			h.logger.Error("Failed to enforce single active LDAP provider", zap.Error(err))
			http.Error(w, "Failed to enforce LDAP provider activation rule", http.StatusInternalServerError)
			return
		}
	}

	// Audit log
	h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventConfigChange, "ldap_config", "update",
		"LDAP configuration updated", map[string]interface{}{
			"config_id":  updatedConfig.ID().String(),
			"config_key": updatedConfig.ConfigKey(),
		})

	ldapResp := h.convertToLDAPResponse(updatedConfig)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ldapResp)
}

// DeleteLDAPConfig deletes an LDAP configuration
func (h *SystemConfigHandler) DeleteLDAPConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		h.logger.Error("No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract config key from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		http.Error(w, "Config key required in path", http.StatusBadRequest)
		return
	}
	configKey := pathParts[len(pathParts)-1]

	if tenantErr := enforceGlobalLDAPScope(r, nil); tenantErr != nil {
		http.Error(w, tenantErr.Error(), http.StatusBadRequest)
		return
	}
	tenantID := (*uuid.UUID)(nil)

	// Get config to verify it exists and is LDAP type
	config, err := h.systemConfigService.GetConfigByKey(r.Context(), tenantID, configKey)
	if err != nil {
		h.logger.Error("Failed to get LDAP config for deletion", zap.Error(err))
		if err == systemconfig.ErrConfigNotFound {
			http.Error(w, "LDAP configuration not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get LDAP configuration", http.StatusInternalServerError)
		return
	}

	if config.ConfigType() != systemconfig.ConfigTypeLDAP {
		http.Error(w, "Configuration is not LDAP type", http.StatusBadRequest)
		return
	}

	err = h.systemConfigService.DeleteConfig(r.Context(), config.ID())
	if err != nil {
		h.logger.Error("Failed to delete LDAP config", zap.Error(err))
		if err == systemconfig.ErrConfigNotFound {
			http.Error(w, "LDAP configuration not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete LDAP configuration", http.StatusInternalServerError)
		return
	}

	// Audit log
	h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventConfigChange, "ldap_config", "delete",
		"LDAP configuration deleted", map[string]interface{}{
			"config_key": configKey,
		})

	w.WriteHeader(http.StatusNoContent)
}

// RebootServer triggers a graceful shutdown for non-production environments.
func (h *SystemConfigHandler) RebootServer(w http.ResponseWriter, r *http.Request) {
	env := strings.ToLower(strings.TrimSpace(h.serverEnvironment))
	if env == "production" || env == "prod" {
		http.Error(w, "Reboot is disabled in production", http.StatusForbidden)
		return
	}

	dryRun := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("dry_run")), "true")
	authCtx, _ := middleware.GetAuthContext(r)
	details := map[string]interface{}{
		"environment": h.serverEnvironment,
		"dry_run":     dryRun,
	}
	if authCtx != nil {
		details["requested_by"] = authCtx.UserID.String()
	}

	if h.auditService != nil {
		if authCtx != nil {
			_ = h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID,
				audit.AuditEventServerRestart, "system", "reboot", "Server reboot requested", details)
		} else {
			_ = h.auditService.LogGlobalSystemAction(r.Context(),
				audit.AuditEventServerRestart, "system", "reboot", "Server reboot requested", details)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if dryRun {
		_ = json.NewEncoder(w).Encode(rebootResponse{
			Status:  "dry_run",
			Message: "Dry run: reboot not executed",
		})
		return
	}

	_ = json.NewEncoder(w).Encode(rebootResponse{
		Status:  "rebooting",
		Message: "Server is restarting",
	})

	go func() {
		time.Sleep(500 * time.Millisecond)
		if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil && h.logger != nil {
			h.logger.Error("Failed to signal reboot", zap.Error(err))
		}
	}()
}

// convertToLDAPResponse converts a domain config to LDAP response format
func (h *SystemConfigHandler) convertToLDAPResponse(config *systemconfig.SystemConfig) LDAPConfigResponse {
	ldapConfig, err := config.GetLDAPConfig()
	if err != nil {
		h.logger.Error("Failed to parse LDAP config", zap.Error(err))
		// Return empty response if parsing fails
		return LDAPConfigResponse{}
	}

	var tenantID string
	if config.TenantID() != nil {
		tenantID = config.TenantID().String()
	}

	return LDAPConfigResponse{
		ID:              config.ID().String(),
		TenantID:        tenantID,
		ConfigKey:       config.ConfigKey(),
		ProviderName:    ldapConfig.ProviderName,
		ProviderType:    ldapConfig.ProviderType,
		Host:            ldapConfig.Host,
		Port:            ldapConfig.Port,
		BaseDN:          ldapConfig.BaseDN,
		UserSearchBase:  ldapConfig.UserSearchBase,
		GroupSearchBase: ldapConfig.GroupSearchBase,
		BindDN:          ldapConfig.BindDN,
		UserFilter:      ldapConfig.UserFilter,
		GroupFilter:     ldapConfig.GroupFilter,
		StartTLS:        ldapConfig.StartTLS,
		SSL:             ldapConfig.SSL,
		AllowedDomains:  ldapConfig.AllowedDomains,
		Enabled:         ldapConfig.Enabled,
		Status:          string(config.Status()),
		Description:     config.Description(),
		IsDefault:       config.IsDefault(),
		IsActive:        config.IsActive(),
		CreatedBy:       config.CreatedBy().String(),
		UpdatedBy:       config.UpdatedBy().String(),
		CreatedAt:       config.CreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       config.UpdatedAt().Format("2006-01-02T15:04:05Z"),
		Version:         config.Version(),
	}
}

func (h *SystemConfigHandler) enforceSingleActiveLDAPProvider(ctx context.Context, tenantID *uuid.UUID, activeConfigID uuid.UUID, updatedBy uuid.UUID) error {
	configs, err := h.systemConfigService.GetConfigsByType(ctx, tenantID, systemconfig.ConfigTypeLDAP)
	if err != nil {
		return err
	}

	for _, cfg := range configs {
		if cfg.ID() == activeConfigID {
			continue
		}
		ldapCfg, err := cfg.GetLDAPConfig()
		if err != nil {
			continue
		}
		if !ldapCfg.Enabled {
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

// convertToResponse converts a domain config to response format
func (h *SystemConfigHandler) convertToResponse(config *systemconfig.SystemConfig) SystemConfigResponse {
	var tenantID string
	if config.TenantID() != nil {
		tenantID = config.TenantID().String()
	}

	return SystemConfigResponse{
		ID:          config.ID().String(),
		TenantID:    tenantID,
		ConfigType:  string(config.ConfigType()),
		ConfigKey:   config.ConfigKey(),
		ConfigValue: config.ConfigValue(),
		Status:      string(config.Status()),
		Description: config.Description(),
		IsDefault:   config.IsDefault(),
		IsActive:    config.IsActive(),
		CreatedBy:   config.CreatedBy().String(),
		UpdatedBy:   config.UpdatedBy().String(),
		CreatedAt:   config.CreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   config.UpdatedAt().Format("2006-01-02T15:04:05Z"),
		Version:     config.Version(),
	}
}
