package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

type inMemorySystemConfigRepo struct {
	items map[string]*systemconfig.SystemConfig
}

func newInMemorySystemConfigRepo() *inMemorySystemConfigRepo {
	return &inMemorySystemConfigRepo{items: make(map[string]*systemconfig.SystemConfig)}
}

func (r *inMemorySystemConfigRepo) key(tenantID *uuid.UUID, configType systemconfig.ConfigType, configKey string) string {
	tenant := "global"
	if tenantID != nil {
		tenant = tenantID.String()
	}
	return tenant + "|" + string(configType) + "|" + configKey
}

func (r *inMemorySystemConfigRepo) Save(ctx context.Context, config *systemconfig.SystemConfig) error {
	if config == nil {
		return errors.New("config is required")
	}
	r.items[r.key(config.TenantID(), config.ConfigType(), config.ConfigKey())] = config
	return nil
}

func (r *inMemorySystemConfigRepo) SaveAll(ctx context.Context, configs []*systemconfig.SystemConfig) error {
	for _, cfg := range configs {
		if err := r.Save(ctx, cfg); err != nil {
			return err
		}
	}
	return nil
}

func (r *inMemorySystemConfigRepo) FindByID(ctx context.Context, id uuid.UUID) (*systemconfig.SystemConfig, error) {
	for _, cfg := range r.items {
		if cfg.ID() == id {
			return cfg, nil
		}
	}
	return nil, systemconfig.ErrConfigNotFound
}

func (r *inMemorySystemConfigRepo) FindByKey(ctx context.Context, tenantID *uuid.UUID, configKey string) (*systemconfig.SystemConfig, error) {
	for _, cfg := range r.items {
		if cfg.ConfigKey() != configKey {
			continue
		}
		if (tenantID == nil && cfg.TenantID() == nil) || (tenantID != nil && cfg.TenantID() != nil && *tenantID == *cfg.TenantID()) {
			return cfg, nil
		}
	}
	return nil, systemconfig.ErrConfigNotFound
}

func (r *inMemorySystemConfigRepo) FindByTypeAndKey(ctx context.Context, tenantID *uuid.UUID, configType systemconfig.ConfigType, configKey string) (*systemconfig.SystemConfig, error) {
	cfg, ok := r.items[r.key(tenantID, configType, configKey)]
	if !ok {
		return nil, systemconfig.ErrConfigNotFound
	}
	return cfg, nil
}

func (r *inMemorySystemConfigRepo) FindByType(ctx context.Context, tenantID *uuid.UUID, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	out := make([]*systemconfig.SystemConfig, 0)
	for _, cfg := range r.items {
		if cfg.ConfigType() != configType {
			continue
		}
		if tenantID == nil {
			if cfg.TenantID() == nil {
				out = append(out, cfg)
			}
			continue
		}
		if cfg.TenantID() != nil && *cfg.TenantID() == *tenantID {
			out = append(out, cfg)
		}
	}
	return out, nil
}

func (r *inMemorySystemConfigRepo) FindAllByType(ctx context.Context, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	out := make([]*systemconfig.SystemConfig, 0)
	for _, cfg := range r.items {
		if cfg.ConfigType() == configType {
			out = append(out, cfg)
		}
	}
	return out, nil
}

func (r *inMemorySystemConfigRepo) FindUniversalByType(ctx context.Context, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	return r.FindByType(ctx, nil, configType)
}

func (r *inMemorySystemConfigRepo) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*systemconfig.SystemConfig, error) {
	out := make([]*systemconfig.SystemConfig, 0)
	for _, cfg := range r.items {
		if cfg.TenantID() != nil && *cfg.TenantID() == tenantID {
			out = append(out, cfg)
		}
	}
	return out, nil
}

func (r *inMemorySystemConfigRepo) FindAll(ctx context.Context) ([]*systemconfig.SystemConfig, error) {
	out := make([]*systemconfig.SystemConfig, 0, len(r.items))
	for _, cfg := range r.items {
		out = append(out, cfg)
	}
	return out, nil
}

func (r *inMemorySystemConfigRepo) FindActiveByType(ctx context.Context, tenantID uuid.UUID, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	out := make([]*systemconfig.SystemConfig, 0)
	for _, cfg := range r.items {
		if cfg.ConfigType() == configType && cfg.TenantID() != nil && *cfg.TenantID() == tenantID && cfg.IsActive() {
			out = append(out, cfg)
		}
	}
	return out, nil
}

func (r *inMemorySystemConfigRepo) Update(ctx context.Context, config *systemconfig.SystemConfig) error {
	return r.Save(ctx, config)
}

func (r *inMemorySystemConfigRepo) Delete(ctx context.Context, id uuid.UUID) error {
	for key, cfg := range r.items {
		if cfg.ID() == id {
			delete(r.items, key)
			return nil
		}
	}
	return systemconfig.ErrConfigNotFound
}

func (r *inMemorySystemConfigRepo) ExistsByKey(ctx context.Context, tenantID *uuid.UUID, configKey string) (bool, error) {
	_, err := r.FindByKey(ctx, tenantID, configKey)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, systemconfig.ErrConfigNotFound) {
		return false, nil
	}
	return false, err
}

func (r *inMemorySystemConfigRepo) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	configs, _ := r.FindByTenantID(ctx, tenantID)
	return len(configs), nil
}

func (r *inMemorySystemConfigRepo) CountByType(ctx context.Context, tenantID uuid.UUID, configType systemconfig.ConfigType) (int, error) {
	configs, _ := r.FindByType(ctx, &tenantID, configType)
	return len(configs), nil
}

func TestSystemConfigHandler_UpdateCategoryConfig_AllowsMessagingType(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := &SystemConfigHandler{
		logger: logger,
	}

	userID := uuid.New()
	authCtx := &middleware.AuthContext{
		UserID: userID,
	}

	payload := map[string]interface{}{
		"config_type":  "messaging",
		"config_key":   "messaging",
		"config_value": map[string]interface{}{"enable_nats": true},
	}

	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/system-configs/category", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), "auth", authCtx))

	w := httptest.NewRecorder()

	defer func() {
		if r := recover(); r != nil {
			// Expected panic due to nil service dependencies.
			// This confirms the handler accepted the config type and proceeded.
		}
		assert.NotEqual(t, http.StatusBadRequest, w.Code)
	}()

	handler.UpdateCategoryConfig(w, req)
}

func TestSystemConfigHandler_UpdateCategoryConfig_InvalidType(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := &SystemConfigHandler{
		logger: logger,
	}

	payload := map[string]interface{}{
		"config_type":  "invalid_type",
		"config_key":   "messaging",
		"config_value": map[string]interface{}{"enable_nats": true},
	}

	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/system-configs/category", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.UpdateCategoryConfig(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSystemConfigHandler_RebootServer_DryRun_NonProd(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := &SystemConfigHandler{
		logger:            logger,
		serverEnvironment: "development",
	}

	req := httptest.NewRequest("POST", "/api/v1/admin/system/reboot?dry_run=true", nil)
	w := httptest.NewRecorder()

	handler.RebootServer(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "dry_run", resp["status"])
}

func TestSystemConfigHandler_RebootServer_ProdForbidden(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := &SystemConfigHandler{
		logger:            logger,
		serverEnvironment: "production",
	}

	req := httptest.NewRequest("POST", "/api/v1/admin/system/reboot", nil)
	w := httptest.NewRecorder()

	handler.RebootServer(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSystemConfigHandler_RuntimeServices_UpdateAndList_IncludesCleanupFields(t *testing.T) {
	logger := zaptest.NewLogger(t)
	repo := newInMemorySystemConfigRepo()
	service := systemconfig.NewService(repo, logger)
	handler := &SystemConfigHandler{
		systemConfigService: service,
		logger:              logger,
	}

	tenantID := uuid.New()
	userID := uuid.New()
	authCtx := &middleware.AuthContext{
		UserID:   userID,
		TenantID: tenantID,
	}

	updatePayload := map[string]interface{}{
		"config_type": "runtime_services",
		"config_key":  "runtime_services",
		"config_value": map[string]interface{}{
			"image_import_notification_receipt_cleanup_enabled":        true,
			"image_import_notification_receipt_retention_days":         45,
			"image_import_notification_receipt_cleanup_interval_hours": 12,
		},
	}
	updateBody, err := json.Marshal(updatePayload)
	require.NoError(t, err)
	updateReq := httptest.NewRequest(http.MethodPost, "/api/v1/system-configs/category", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq = updateReq.WithContext(context.WithValue(updateReq.Context(), "auth", authCtx))
	updateResp := httptest.NewRecorder()

	handler.UpdateCategoryConfig(updateResp, updateReq)
	require.Equal(t, http.StatusOK, updateResp.Code, updateResp.Body.String())

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/system-configs?config_type=runtime_services", nil)
	listReq = listReq.WithContext(context.WithValue(listReq.Context(), "auth", authCtx))
	listResp := httptest.NewRecorder()

	handler.ListConfigs(listResp, listReq)
	require.Equal(t, http.StatusOK, listResp.Code, listResp.Body.String())

	var out struct {
		Configs []struct {
			ConfigType  string                 `json:"config_type"`
			ConfigKey   string                 `json:"config_key"`
			ConfigValue map[string]interface{} `json:"config_value"`
		} `json:"configs"`
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(listResp.Body.Bytes(), &out))
	require.Equal(t, 1, out.Total)
	require.Len(t, out.Configs, 1)
	assert.Equal(t, "runtime_services", out.Configs[0].ConfigType)
	assert.Equal(t, "runtime_services", out.Configs[0].ConfigKey)
	assert.Equal(t, true, out.Configs[0].ConfigValue["image_import_notification_receipt_cleanup_enabled"])
	assert.Equal(t, float64(45), out.Configs[0].ConfigValue["image_import_notification_receipt_retention_days"])
	assert.Equal(t, float64(12), out.Configs[0].ConfigValue["image_import_notification_receipt_cleanup_interval_hours"])

	raw := strings.TrimSpace(listResp.Body.String())
	assert.NotEmpty(t, raw)
}

func TestSystemConfigHandler_RuntimeServices_UpdateCategoryConfig_ValidationFieldErrors(t *testing.T) {
	logger := zaptest.NewLogger(t)
	repo := newInMemorySystemConfigRepo()
	service := systemconfig.NewService(repo, logger)
	handler := &SystemConfigHandler{
		systemConfigService: service,
		logger:              logger,
	}

	tenantID := uuid.New()
	userID := uuid.New()
	authCtx := &middleware.AuthContext{
		UserID:   userID,
		TenantID: tenantID,
	}

	updatePayload := map[string]interface{}{
		"config_type": "runtime_services",
		"config_key":  "runtime_services",
		"config_value": map[string]interface{}{
			"image_import_notification_receipt_retention_days": 0,
		},
	}
	updateBody, err := json.Marshal(updatePayload)
	require.NoError(t, err)
	updateReq := httptest.NewRequest(http.MethodPost, "/api/v1/system-configs/category", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq = updateReq.WithContext(context.WithValue(updateReq.Context(), "auth", authCtx))
	updateResp := httptest.NewRecorder()

	handler.UpdateCategoryConfig(updateResp, updateReq)
	require.Equal(t, http.StatusBadRequest, updateResp.Code, updateResp.Body.String())

	var out struct {
		Error       string            `json:"error"`
		Message     string            `json:"message"`
		FieldErrors map[string]string `json:"field_errors"`
	}
	require.NoError(t, json.Unmarshal(updateResp.Body.Bytes(), &out))
	assert.Equal(t, "validation_failed", out.Error)
	assert.Equal(t, "runtime_services validation failed", out.Message)
	assert.Equal(t, "must be between 1 and 3650", out.FieldErrors["image_import_notification_receipt_retention_days"])
}

func TestSystemConfigHandler_RuntimeServices_UpdateCategoryConfig_ValidationFieldErrors_ProviderWatcher(t *testing.T) {
	logger := zaptest.NewLogger(t)
	repo := newInMemorySystemConfigRepo()
	service := systemconfig.NewService(repo, logger)
	handler := &SystemConfigHandler{
		systemConfigService: service,
		logger:              logger,
	}

	tenantID := uuid.New()
	userID := uuid.New()
	authCtx := &middleware.AuthContext{
		UserID:   userID,
		TenantID: tenantID,
	}

	updatePayload := map[string]interface{}{
		"config_type": "runtime_services",
		"config_key":  "runtime_services",
		"config_value": map[string]interface{}{
			"provider_readiness_watcher_interval_seconds": 20,
			"provider_readiness_watcher_timeout_seconds":  30,
			"provider_readiness_watcher_batch_size":       1001,
		},
	}
	updateBody, err := json.Marshal(updatePayload)
	require.NoError(t, err)
	updateReq := httptest.NewRequest(http.MethodPost, "/api/v1/system-configs/category", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq = updateReq.WithContext(context.WithValue(updateReq.Context(), "auth", authCtx))
	updateResp := httptest.NewRecorder()

	handler.UpdateCategoryConfig(updateResp, updateReq)
	require.Equal(t, http.StatusBadRequest, updateResp.Code, updateResp.Body.String())

	var out struct {
		Error       string            `json:"error"`
		Message     string            `json:"message"`
		FieldErrors map[string]string `json:"field_errors"`
	}
	require.NoError(t, json.Unmarshal(updateResp.Body.Bytes(), &out))
	assert.Equal(t, "validation_failed", out.Error)
	assert.Equal(t, "runtime_services validation failed", out.Message)
	assert.Equal(t, "must be at least 30", out.FieldErrors["provider_readiness_watcher_interval_seconds"])
	assert.Equal(t, "must be less than provider_readiness_watcher_interval_seconds", out.FieldErrors["provider_readiness_watcher_timeout_seconds"])
	assert.Equal(t, "must be between 1 and 1000", out.FieldErrors["provider_readiness_watcher_batch_size"])
}

func TestSystemConfigHandler_RuntimeServices_UpdateCategoryConfig_ValidationFieldErrors_TektonCleanup(t *testing.T) {
	logger := zaptest.NewLogger(t)
	repo := newInMemorySystemConfigRepo()
	service := systemconfig.NewService(repo, logger)
	handler := &SystemConfigHandler{
		systemConfigService: service,
		logger:              logger,
	}

	tenantID := uuid.New()
	userID := uuid.New()
	authCtx := &middleware.AuthContext{
		UserID:   userID,
		TenantID: tenantID,
	}

	updatePayload := map[string]interface{}{
		"config_type": "runtime_services",
		"config_key":  "runtime_services",
		"config_value": map[string]interface{}{
			"tekton_history_cleanup_keep_pipelineruns": 0,
			"tekton_history_cleanup_keep_taskruns":     0,
			"tekton_history_cleanup_keep_pods":         0,
		},
	}
	updateBody, err := json.Marshal(updatePayload)
	require.NoError(t, err)
	updateReq := httptest.NewRequest(http.MethodPost, "/api/v1/system-configs/category", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq = updateReq.WithContext(context.WithValue(updateReq.Context(), "auth", authCtx))
	updateResp := httptest.NewRecorder()

	handler.UpdateCategoryConfig(updateResp, updateReq)
	require.Equal(t, http.StatusBadRequest, updateResp.Code, updateResp.Body.String())

	var out struct {
		Error       string            `json:"error"`
		Message     string            `json:"message"`
		FieldErrors map[string]string `json:"field_errors"`
	}
	require.NoError(t, json.Unmarshal(updateResp.Body.Bytes(), &out))
	assert.Equal(t, "validation_failed", out.Error)
	assert.Equal(t, "runtime_services validation failed", out.Message)
	assert.Equal(t, "must be at least 1", out.FieldErrors["tekton_history_cleanup_keep_pipelineruns"])
	assert.Equal(t, "must be at least 1", out.FieldErrors["tekton_history_cleanup_keep_taskruns"])
	assert.Equal(t, "must be at least 1", out.FieldErrors["tekton_history_cleanup_keep_pods"])
}

func TestSystemConfigHandler_RuntimeServices_UpdateCategoryConfig_ValidationFieldErrors_PortsAndHealth(t *testing.T) {
	logger := zaptest.NewLogger(t)
	repo := newInMemorySystemConfigRepo()
	service := systemconfig.NewService(repo, logger)
	handler := &SystemConfigHandler{
		systemConfigService: service,
		logger:              logger,
	}

	tenantID := uuid.New()
	userID := uuid.New()
	authCtx := &middleware.AuthContext{
		UserID:   userID,
		TenantID: tenantID,
	}

	updatePayload := map[string]interface{}{
		"config_type": "runtime_services",
		"config_key":  "runtime_services",
		"config_value": map[string]interface{}{
			"dispatcher_port":              0,
			"email_worker_port":            70000,
			"notification_worker_port":     0,
			"health_check_timeout_seconds": 0,
		},
	}
	updateBody, err := json.Marshal(updatePayload)
	require.NoError(t, err)
	updateReq := httptest.NewRequest(http.MethodPost, "/api/v1/system-configs/category", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq = updateReq.WithContext(context.WithValue(updateReq.Context(), "auth", authCtx))
	updateResp := httptest.NewRecorder()

	handler.UpdateCategoryConfig(updateResp, updateReq)
	require.Equal(t, http.StatusBadRequest, updateResp.Code, updateResp.Body.String())

	var out struct {
		Error       string            `json:"error"`
		Message     string            `json:"message"`
		FieldErrors map[string]string `json:"field_errors"`
	}
	require.NoError(t, json.Unmarshal(updateResp.Body.Bytes(), &out))
	assert.Equal(t, "validation_failed", out.Error)
	assert.Equal(t, "runtime_services validation failed", out.Message)
	assert.Equal(t, "must be between 1 and 65535", out.FieldErrors["dispatcher_port"])
	assert.Equal(t, "must be between 1 and 65535", out.FieldErrors["email_worker_port"])
	assert.Equal(t, "must be between 1 and 65535", out.FieldErrors["notification_worker_port"])
	assert.Equal(t, "must be at least 1", out.FieldErrors["health_check_timeout_seconds"])
}

func TestSystemConfigHandler_RuntimeServices_UpdateCategoryConfig_ValidationFieldErrors_TektonCleanupSchedule(t *testing.T) {
	logger := zaptest.NewLogger(t)
	repo := newInMemorySystemConfigRepo()
	service := systemconfig.NewService(repo, logger)
	handler := &SystemConfigHandler{
		systemConfigService: service,
		logger:              logger,
	}

	tenantID := uuid.New()
	userID := uuid.New()
	authCtx := &middleware.AuthContext{
		UserID:   userID,
		TenantID: tenantID,
	}

	updatePayload := map[string]interface{}{
		"config_type": "runtime_services",
		"config_key":  "runtime_services",
		"config_value": map[string]interface{}{
			"tekton_history_cleanup_schedule": "bad cron expr!",
		},
	}
	updateBody, err := json.Marshal(updatePayload)
	require.NoError(t, err)
	updateReq := httptest.NewRequest(http.MethodPost, "/api/v1/system-configs/category", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq = updateReq.WithContext(context.WithValue(updateReq.Context(), "auth", authCtx))
	updateResp := httptest.NewRecorder()

	handler.UpdateCategoryConfig(updateResp, updateReq)
	require.Equal(t, http.StatusBadRequest, updateResp.Code, updateResp.Body.String())

	var out struct {
		Error       string            `json:"error"`
		Message     string            `json:"message"`
		FieldErrors map[string]string `json:"field_errors"`
	}
	require.NoError(t, json.Unmarshal(updateResp.Body.Bytes(), &out))
	assert.Equal(t, "validation_failed", out.Error)
	assert.Equal(t, "runtime_services validation failed", out.Message)
	assert.Equal(t, "must contain exactly 5 cron fields", out.FieldErrors["tekton_history_cleanup_schedule"])
}

func TestSystemConfigHandler_RuntimeServices_UpdateCategoryConfig_ValidationFieldErrors_URLs(t *testing.T) {
	logger := zaptest.NewLogger(t)
	repo := newInMemorySystemConfigRepo()
	service := systemconfig.NewService(repo, logger)
	handler := &SystemConfigHandler{
		systemConfigService: service,
		logger:              logger,
	}

	tenantID := uuid.New()
	userID := uuid.New()
	authCtx := &middleware.AuthContext{
		UserID:   userID,
		TenantID: tenantID,
	}

	updatePayload := map[string]interface{}{
		"config_type": "runtime_services",
		"config_key":  "runtime_services",
		"config_value": map[string]interface{}{
			"dispatcher_url":          "://bad",
			"email_worker_url":        "ftp://localhost",
			"notification_worker_url": "",
		},
	}
	updateBody, err := json.Marshal(updatePayload)
	require.NoError(t, err)
	updateReq := httptest.NewRequest(http.MethodPost, "/api/v1/system-configs/category", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq = updateReq.WithContext(context.WithValue(updateReq.Context(), "auth", authCtx))
	updateResp := httptest.NewRecorder()

	handler.UpdateCategoryConfig(updateResp, updateReq)
	require.Equal(t, http.StatusBadRequest, updateResp.Code, updateResp.Body.String())

	var out struct {
		Error       string            `json:"error"`
		Message     string            `json:"message"`
		FieldErrors map[string]string `json:"field_errors"`
	}
	require.NoError(t, json.Unmarshal(updateResp.Body.Bytes(), &out))
	assert.Equal(t, "validation_failed", out.Error)
	assert.Equal(t, "runtime_services validation failed", out.Message)
	assert.Equal(t, "must be a valid absolute URL", out.FieldErrors["dispatcher_url"])
	assert.Equal(t, "must use http or https scheme", out.FieldErrors["email_worker_url"])
	assert.Equal(t, "must be a non-empty absolute URL", out.FieldErrors["notification_worker_url"])
}

func TestSystemConfigHandler_TektonTaskImages_GetDefaults(t *testing.T) {
	logger := zaptest.NewLogger(t)
	repo := newInMemorySystemConfigRepo()
	service := systemconfig.NewService(repo, logger)
	handler := &SystemConfigHandler{
		systemConfigService: service,
		logger:              logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/tekton-task-images", nil)
	resp := httptest.NewRecorder()

	handler.GetTektonTaskImages(resp, req)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

	var out systemconfig.TektonTaskImagesConfig
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
	assert.Equal(t, "docker.io/moby/buildkit:v0.13.2", out.Buildkit)
	assert.Equal(t, "quay.io/skopeo/stable:v1.15.0", out.Skopeo)
}

func TestSystemConfigHandler_TektonTaskImages_UpdateThenGet(t *testing.T) {
	logger := zaptest.NewLogger(t)
	repo := newInMemorySystemConfigRepo()
	service := systemconfig.NewService(repo, logger)
	handler := &SystemConfigHandler{
		systemConfigService: service,
		logger:              logger,
	}

	userID := uuid.New()
	authCtx := &middleware.AuthContext{UserID: userID}

	payload := systemconfig.TektonTaskImagesConfig{
		GitClone:       "registry.local/tools/alpine-git:2.45.2",
		KanikoExecutor: "registry.local/tools/kaniko:v1.23.2",
		Buildkit:       "registry.local/tools/buildkit:v0.13.2",
		Skopeo:         "registry.local/tools/skopeo:v1.15.0",
		Trivy:          "registry.local/security/trivy:0.57.1",
		Syft:           "registry.local/security/syft:v1.18.1",
		Cosign:         "registry.local/security/cosign:v2.4.1",
		Packer:         "registry.local/tools/packer:1.10.2",
		PythonAlpine:   "registry.local/base/python:3.12-alpine",
		Alpine:         "registry.local/base/alpine:3.20",
		CleanupKubectl: "registry.local/tools/kubectl:latest",
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/tekton-task-images", bytes.NewReader(body))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq = updateReq.WithContext(context.WithValue(updateReq.Context(), "auth", authCtx))
	updateResp := httptest.NewRecorder()

	handler.UpdateTektonTaskImages(updateResp, updateReq)
	require.Equal(t, http.StatusOK, updateResp.Code, updateResp.Body.String())

	var updated systemconfig.TektonTaskImagesConfig
	require.NoError(t, json.Unmarshal(updateResp.Body.Bytes(), &updated))
	assert.Equal(t, payload.Buildkit, updated.Buildkit)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/tekton-task-images", nil)
	getResp := httptest.NewRecorder()
	handler.GetTektonTaskImages(getResp, getReq)
	require.Equal(t, http.StatusOK, getResp.Code, getResp.Body.String())

	var fetched systemconfig.TektonTaskImagesConfig
	require.NoError(t, json.Unmarshal(getResp.Body.Bytes(), &fetched))
	assert.Equal(t, payload.Trivy, fetched.Trivy)
	assert.Equal(t, payload.KanikoExecutor, fetched.KanikoExecutor)
}

func TestSystemConfigHandler_TektonTaskImages_UpdateInvalidPayload(t *testing.T) {
	logger := zaptest.NewLogger(t)
	repo := newInMemorySystemConfigRepo()
	service := systemconfig.NewService(repo, logger)
	handler := &SystemConfigHandler{
		systemConfigService: service,
		logger:              logger,
	}

	userID := uuid.New()
	authCtx := &middleware.AuthContext{UserID: userID}

	raw := []byte(`{"git_clone":"not valid image ref"}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/tekton-task-images", bytes.NewReader(raw))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq = updateReq.WithContext(context.WithValue(updateReq.Context(), "auth", authCtx))
	updateResp := httptest.NewRecorder()

	handler.UpdateTektonTaskImages(updateResp, updateReq)
	require.Equal(t, http.StatusBadRequest, updateResp.Code, updateResp.Body.String())
}

func TestSystemConfigHandler_ProductInfoMetadata_GetDefaults(t *testing.T) {
	logger := zaptest.NewLogger(t)
	repo := newInMemorySystemConfigRepo()
	service := systemconfig.NewService(repo, logger)
	handler := &SystemConfigHandler{
		systemConfigService: service,
		logger:              logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/product-info-metadata", nil)
	resp := httptest.NewRecorder()

	handler.GetProductInfoMetadata(resp, req)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

	var out systemconfig.ProductInfoMetadataConfig
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
	assert.Equal(t, "", out.LastBacklogSync)
}

func TestSystemConfigHandler_ProductInfoMetadata_ConfiguredValue(t *testing.T) {
	logger := zaptest.NewLogger(t)
	repo := newInMemorySystemConfigRepo()
	service := systemconfig.NewService(repo, logger)
	handler := &SystemConfigHandler{
		systemConfigService: service,
		logger:              logger,
	}

	_, err := service.CreateOrUpdateCategoryConfig(
		context.Background(),
		nil,
		systemconfig.ConfigTypeToolSettings,
		"product_info_metadata",
		systemconfig.ProductInfoMetadataConfig{LastBacklogSync: "2026-03-17"},
		uuid.New(),
	)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/product-info-metadata", nil)
	resp := httptest.NewRecorder()

	handler.GetProductInfoMetadata(resp, req)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

	var out systemconfig.ProductInfoMetadataConfig
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
	assert.Equal(t, "2026-03-17", out.LastBacklogSync)
}
