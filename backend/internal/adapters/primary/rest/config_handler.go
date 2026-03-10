package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/build"
)

// ConfigHandler handles build method configuration endpoints backed by build_configs.
type ConfigHandler struct {
	repo     build.Repository
	db       *sqlx.DB
	resolver *build.DefaultConfigResolver
	logger   *zap.Logger
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(repo build.Repository, db *sqlx.DB, logger *zap.Logger) *ConfigHandler {
	return &ConfigHandler{
		repo:     repo,
		db:       db,
		resolver: build.NewDefaultConfigResolver(nil),
		logger:   logger,
	}
}

// ============================================================================
// Request/Response Types
// ============================================================================

type CreatePackerConfigRequest struct {
	BuildID   string                 `json:"build_id"`
	Template  string                 `json:"template"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type CreateBuildxConfigRequest struct {
	BuildID        string            `json:"build_id"`
	Dockerfile     json.RawMessage   `json:"dockerfile"`
	BuildContext   string            `json:"build_context"`
	RegistryAuthID string            `json:"registry_auth_id,omitempty"`
	Platforms      []string          `json:"platforms,omitempty"`
	BuildArgs      map[string]string `json:"build_args,omitempty"`
	Secrets        map[string]string `json:"secrets,omitempty"`
	Cache          map[string]string `json:"cache,omitempty"`
	NoCache        bool              `json:"no_cache,omitempty"`
	Outputs        []string          `json:"outputs,omitempty"`
}

type CreateKanikoConfigRequest struct {
	BuildID          string            `json:"build_id"`
	Dockerfile       json.RawMessage   `json:"dockerfile"`
	BuildContext     string            `json:"build_context"`
	RegistryRepo     string            `json:"registry_repo"`
	RegistryAuthID   string            `json:"registry_auth_id,omitempty"`
	CacheRepo        string            `json:"cache_repo,omitempty"`
	BuildArgs        map[string]string `json:"build_args,omitempty"`
	SkipUnusedStages bool              `json:"skip_unused_stages,omitempty"`
}

type CreateDockerConfigRequest struct {
	BuildID         string            `json:"build_id"`
	Dockerfile      json.RawMessage   `json:"dockerfile"`
	BuildContext    string            `json:"build_context"`
	RegistryAuthID  string            `json:"registry_auth_id,omitempty"`
	TargetStage     string            `json:"target_stage,omitempty"`
	BuildArgs       map[string]string `json:"build_args,omitempty"`
	EnvironmentVars map[string]string `json:"environment_vars,omitempty"`
}

type CreatePaketoConfigRequest struct {
	BuildID        string            `json:"build_id"`
	Builder        string            `json:"builder"`
	RegistryAuthID string            `json:"registry_auth_id,omitempty"`
	Buildpacks     []string          `json:"buildpacks,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	BuildArgs      map[string]string `json:"build_args,omitempty"`
}

type CreateNixConfigRequest struct {
	BuildID       string            `json:"build_id"`
	NixExpression string            `json:"nix_expression,omitempty"`
	FlakeURI      string            `json:"flake_uri,omitempty"`
	Attributes    []string          `json:"attributes,omitempty"`
	Outputs       map[string]string `json:"outputs,omitempty"`
	CacheDir      string            `json:"cache_dir,omitempty"`
	Pure          bool              `json:"pure,omitempty"`
	ShowTrace     bool              `json:"show_trace,omitempty"`
}

type ConfigResponse struct {
	ID        string      `json:"id"`
	BuildID   string      `json:"build_id"`
	Method    string      `json:"method"`
	Config    interface{} `json:"config"`
	CreatedAt string      `json:"created_at,omitempty"`
	UpdatedAt string      `json:"updated_at,omitempty"`
}

type PresetResponse struct {
	Name        string      `json:"name"`
	Method      string      `json:"method"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ============================================================================
// Packer Configuration Endpoints
// ============================================================================

// CreatePackerConfig handles POST /api/v1/config/packer
func (h *ConfigHandler) CreatePackerConfig(w http.ResponseWriter, r *http.Request) {
	var req CreatePackerConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	buildID, err := uuid.Parse(req.BuildID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid build id format", err)
		return
	}

	config := &build.BuildConfigData{
		BuildID:        buildID,
		BuildMethod:    "packer",
		PackerTemplate: req.Template,
		Metadata:       map[string]interface{}{},
	}
	if len(req.Variables) > 0 {
		config.Metadata["variables"] = req.Variables
	}

	if err := h.repo.SaveBuildConfig(r.Context(), config); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to create packer config", err)
		return
	}

	h.respondJSON(w, http.StatusCreated, h.buildConfigToResponse(config))
}

// ============================================================================
// Buildx Configuration Endpoints
// ============================================================================

// CreateBuildxConfig handles POST /api/v1/config/buildx
func (h *ConfigHandler) CreateBuildxConfig(w http.ResponseWriter, r *http.Request) {
	var req CreateBuildxConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	buildID, err := uuid.Parse(req.BuildID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid build id format", err)
		return
	}

	dockerfile, err := parseDockerfile(req.Dockerfile)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid dockerfile", err)
		return
	}
	config := &build.BuildConfigData{
		BuildID:      buildID,
		BuildMethod:  "buildx",
		Dockerfile:   dockerfile,
		BuildContext: req.BuildContext,
		Platforms:    req.Platforms,
		BuildArgs:    req.BuildArgs,
		Secrets:      req.Secrets,
		Metadata:     map[string]interface{}{},
	}
	if req.RegistryAuthID != "" {
		config.Metadata["registry_auth_id"] = req.RegistryAuthID
	}

	if cacheFrom, ok := req.Cache["from"]; ok && cacheFrom != "" {
		config.CacheFrom = []string{cacheFrom}
	}
	if cacheTo, ok := req.Cache["to"]; ok && cacheTo != "" {
		config.CacheTo = cacheTo
	}
	if req.NoCache {
		config.CacheEnabled = false
		config.Metadata["no_cache"] = true
	} else if config.CacheTo != "" || len(config.CacheFrom) > 0 {
		config.CacheEnabled = true
	}
	if len(req.Outputs) > 0 {
		config.Metadata["outputs"] = req.Outputs
	}

	if err := h.repo.SaveBuildConfig(r.Context(), config); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to create buildx config", err)
		return
	}

	h.respondJSON(w, http.StatusCreated, h.buildConfigToResponse(config))
}

// ============================================================================
// Kaniko Configuration Endpoints
// ============================================================================

// CreateKanikoConfig handles POST /api/v1/config/kaniko
func (h *ConfigHandler) CreateKanikoConfig(w http.ResponseWriter, r *http.Request) {
	var req CreateKanikoConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	buildID, err := uuid.Parse(req.BuildID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid build id format", err)
		return
	}

	dockerfile, err := parseDockerfile(req.Dockerfile)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid dockerfile", err)
		return
	}
	if req.RegistryRepo == "" {
		h.respondError(w, http.StatusBadRequest, "registry_repo is required", nil)
		return
	}

	config := &build.BuildConfigData{
		BuildID:      buildID,
		BuildMethod:  "kaniko",
		Dockerfile:   dockerfile,
		BuildContext: req.BuildContext,
		CacheRepo:    req.CacheRepo,
		BuildArgs:    req.BuildArgs,
		Metadata:     map[string]interface{}{},
	}
	config.Metadata["registry_repo"] = req.RegistryRepo
	if req.RegistryAuthID != "" {
		config.Metadata["registry_auth_id"] = req.RegistryAuthID
	}
	if req.SkipUnusedStages {
		config.Metadata["skip_unused_stages"] = true
	}

	if err := h.repo.SaveBuildConfig(r.Context(), config); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to create kaniko config", err)
		return
	}

	h.respondJSON(w, http.StatusCreated, h.buildConfigToResponse(config))
}

// ============================================================================
// Docker Configuration Endpoints
// ============================================================================

// CreateDockerConfig handles POST /api/v1/config/docker
func (h *ConfigHandler) CreateDockerConfig(w http.ResponseWriter, r *http.Request) {
	var req CreateDockerConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	buildID, err := uuid.Parse(req.BuildID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid build id format", err)
		return
	}

	dockerfile, err := parseDockerfile(req.Dockerfile)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid dockerfile", err)
		return
	}

	config := &build.BuildConfigData{
		BuildID:      buildID,
		BuildMethod:  "docker",
		Dockerfile:   dockerfile,
		BuildContext: req.BuildContext,
		TargetStage:  req.TargetStage,
		BuildArgs:    req.BuildArgs,
		Environment:  req.EnvironmentVars,
		Metadata:     map[string]interface{}{},
	}
	if req.RegistryAuthID != "" {
		config.Metadata["registry_auth_id"] = req.RegistryAuthID
	}

	if err := h.repo.SaveBuildConfig(r.Context(), config); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to create docker config", err)
		return
	}

	h.respondJSON(w, http.StatusCreated, h.buildConfigToResponse(config))
}

// ============================================================================
// Nix Configuration Endpoints
// ============================================================================

// CreatePaketoConfig handles POST /api/v1/config/paketo
func (h *ConfigHandler) CreatePaketoConfig(w http.ResponseWriter, r *http.Request) {
	var req CreatePaketoConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	buildID, err := uuid.Parse(req.BuildID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid build id format", err)
		return
	}
	if req.Builder == "" {
		h.respondError(w, http.StatusBadRequest, "builder is required", nil)
		return
	}

	config := &build.BuildConfigData{
		BuildID:     buildID,
		BuildMethod: "paketo",
		Builder:     req.Builder,
		Buildpacks:  req.Buildpacks,
		Metadata: map[string]interface{}{
			"env":              req.Env,
			"build_args":       req.BuildArgs,
			"registry_auth_id": req.RegistryAuthID,
		},
	}

	if err := h.repo.SaveBuildConfig(r.Context(), config); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to create paketo config", err)
		return
	}

	h.respondJSON(w, http.StatusCreated, h.buildConfigToResponse(config))
}

// ============================================================================
// Nix Configuration Endpoints
// ============================================================================

// CreateNixConfig handles POST /api/v1/config/nix
func (h *ConfigHandler) CreateNixConfig(w http.ResponseWriter, r *http.Request) {
	var req CreateNixConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	buildID, err := uuid.Parse(req.BuildID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid build id format", err)
		return
	}

	config := &build.BuildConfigData{
		BuildID:     buildID,
		BuildMethod: "nix",
		Metadata: map[string]interface{}{
			"nix_expression": req.NixExpression,
			"flake_uri":      req.FlakeURI,
			"attributes":     req.Attributes,
			"outputs":        req.Outputs,
			"cache_dir":      req.CacheDir,
			"pure":           req.Pure,
			"show_trace":     req.ShowTrace,
		},
	}

	if err := h.repo.SaveBuildConfig(r.Context(), config); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to create nix config", err)
		return
	}

	h.respondJSON(w, http.StatusCreated, h.buildConfigToResponse(config))
}

// ============================================================================
// Get Configuration Endpoints
// ============================================================================

// GetConfig handles GET /api/v1/config/:buildId
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	buildIDStr := r.PathValue("buildId")
	if buildIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "build id required", nil)
		return
	}

	buildID, err := uuid.Parse(buildIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid build id format", err)
		return
	}

	config, err := h.repo.GetBuildConfig(r.Context(), buildID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to get config", err)
		return
	}
	if config == nil {
		h.respondError(w, http.StatusNotFound, "config not found", nil)
		return
	}

	h.respondJSON(w, http.StatusOK, h.buildConfigToResponse(config))
}

// ============================================================================
// List Configurations Endpoints
// ============================================================================

// ListConfigsByMethod handles GET /api/v1/config?project_id={id}&method={method}
func (h *ConfigHandler) ListConfigsByMethod(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.URL.Query().Get("project_id")
	method := r.URL.Query().Get("method")

	if projectIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "project_id query parameter required", nil)
		return
	}
	if method == "" {
		h.respondError(w, http.StatusBadRequest, "method query parameter required", nil)
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid project id format", err)
		return
	}

	queryMethod := method
	if method == "docker" {
		queryMethod = "container"
	}

	rows := []uuid.UUID{}
	query := `
		SELECT bc.build_id
		FROM build_configs bc
		INNER JOIN builds b ON b.id = bc.build_id
		WHERE b.project_id = $1 AND bc.build_method = $2
		ORDER BY bc.created_at DESC`
	if err := h.db.SelectContext(r.Context(), &rows, query, projectID, queryMethod); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to list configs", err)
		return
	}

	responses := make([]ConfigResponse, 0, len(rows))
	for _, buildID := range rows {
		config, err := h.repo.GetBuildConfig(r.Context(), buildID)
		if err != nil || config == nil {
			continue
		}
		responses = append(responses, h.buildConfigToResponse(config))
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"count":   len(responses),
		"method":  method,
		"configs": responses,
	})
}

// ============================================================================
// Delete Configuration Endpoints
// ============================================================================

// DeleteConfig handles DELETE /api/v1/config/:buildId
func (h *ConfigHandler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	buildIDStr := r.PathValue("buildId")
	if buildIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "build id required", nil)
		return
	}

	buildID, err := uuid.Parse(buildIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid build id format", err)
		return
	}

	if err := h.repo.DeleteBuildConfig(r.Context(), buildID); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to delete config", err)
		return
	}

	h.respondJSON(w, http.StatusNoContent, nil)
}

// ============================================================================
// Preset Endpoints
// ============================================================================

// GetPresets handles GET /api/v1/config/presets
func (h *ConfigHandler) GetPresets(w http.ResponseWriter, r *http.Request) {
	method := r.URL.Query().Get("method")
	presets := h.resolver.GetDefaultPresets()

	if method != "" {
		buildMethod := build.BuildMethod(method)
		if !buildMethod.IsValid() {
			h.respondError(w, http.StatusBadRequest, "invalid build method", nil)
			return
		}

		methodPresets := presets[buildMethod]
		responses := make([]PresetResponse, 0, len(methodPresets))
		for _, p := range methodPresets {
			responses = append(responses, PresetResponse{
				Name:        p.Name,
				Method:      string(p.Method),
				Description: p.Description,
				Parameters:  p.Parameters,
			})
		}

		h.respondJSON(w, http.StatusOK, map[string]interface{}{
			"method":  method,
			"presets": responses,
		})
		return
	}

	allResponses := make(map[string][]PresetResponse)
	for presetMethod, methodPresets := range presets {
		responses := make([]PresetResponse, 0, len(methodPresets))
		for _, p := range methodPresets {
			responses = append(responses, PresetResponse{
				Name:        p.Name,
				Method:      string(p.Method),
				Description: p.Description,
				Parameters:  p.Parameters,
			})
		}
		allResponses[string(presetMethod)] = responses
	}

	h.respondJSON(w, http.StatusOK, allResponses)
}

// ============================================================================
// Helper Methods
// ============================================================================

func parseDockerfile(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return asString, nil
	}

	var obj struct {
		Source   string `json:"source"`
		Path     string `json:"path"`
		Content  string `json:"content"`
		Filename string `json:"filename"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", err
	}
	if obj.Source == "path" {
		return obj.Path, nil
	}
	if obj.Content != "" {
		return obj.Content, nil
	}
	return obj.Filename, nil
}

func (h *ConfigHandler) buildConfigToResponse(config *build.BuildConfigData) ConfigResponse {
	if config.Metadata == nil {
		config.Metadata = map[string]interface{}{}
	}

	method := config.BuildMethod
	configMap := map[string]interface{}{}

	switch method {
	case "packer":
		configMap["template"] = config.PackerTemplate
		if vars, ok := config.Metadata["variables"]; ok {
			configMap["variables"] = vars
		}
	case "buildx":
		configMap["dockerfile"] = config.Dockerfile
		configMap["build_context"] = config.BuildContext
		configMap["platforms"] = config.Platforms
		if len(config.BuildArgs) > 0 {
			configMap["build_args"] = config.BuildArgs
		}
		if len(config.Secrets) > 0 {
			configMap["secrets"] = config.Secrets
		}
		cache := map[string]interface{}{}
		if len(config.CacheFrom) > 0 {
			cache["from"] = config.CacheFrom[0]
		}
		if config.CacheTo != "" {
			cache["to"] = config.CacheTo
		}
		if len(cache) > 0 {
			configMap["cache"] = cache
		}
		if noCache, ok := config.Metadata["no_cache"].(bool); ok && noCache {
			configMap["no_cache"] = true
		}
		if outputs, ok := config.Metadata["outputs"]; ok {
			configMap["outputs"] = outputs
		}
		if registryAuthID, ok := config.Metadata["registry_auth_id"]; ok {
			configMap["registry_auth_id"] = registryAuthID
		}
	case "kaniko":
		configMap["dockerfile"] = config.Dockerfile
		configMap["build_context"] = config.BuildContext
		configMap["cache_repo"] = config.CacheRepo
		if len(config.BuildArgs) > 0 {
			configMap["build_args"] = config.BuildArgs
		}
		if registryRepo, ok := config.Metadata["registry_repo"]; ok {
			configMap["registry_repo"] = registryRepo
		}
		if skip, ok := config.Metadata["skip_unused_stages"]; ok {
			configMap["skip_unused_stages"] = skip
		}
		if registryAuthID, ok := config.Metadata["registry_auth_id"]; ok {
			configMap["registry_auth_id"] = registryAuthID
		}
	case "docker", "container":
		configMap["dockerfile"] = config.Dockerfile
		configMap["build_context"] = config.BuildContext
		configMap["target_stage"] = config.TargetStage
		if len(config.BuildArgs) > 0 {
			configMap["build_args"] = config.BuildArgs
		}
		if len(config.Environment) > 0 {
			configMap["environment_vars"] = config.Environment
		}
		if registryAuthID, ok := config.Metadata["registry_auth_id"]; ok {
			configMap["registry_auth_id"] = registryAuthID
		}
		if method == "container" {
			method = "docker"
		}
	case "nix":
		for key, value := range config.Metadata {
			configMap[key] = value
		}
	case "paketo":
		configMap["builder"] = config.Builder
		configMap["buildpacks"] = config.Buildpacks
		if env, ok := config.Metadata["env"]; ok {
			configMap["env"] = env
		}
		if buildArgs, ok := config.Metadata["build_args"]; ok {
			configMap["build_args"] = buildArgs
		}
		if registryAuthID, ok := config.Metadata["registry_auth_id"]; ok {
			configMap["registry_auth_id"] = registryAuthID
		}
	}

	response := ConfigResponse{
		ID:      config.ID.String(),
		BuildID: config.BuildID.String(),
		Method:  method,
		Config:  configMap,
	}
	if !config.CreatedAt.IsZero() {
		response.CreatedAt = config.CreatedAt.Format(time.RFC3339)
	}
	if !config.UpdatedAt.IsZero() {
		response.UpdatedAt = config.UpdatedAt.Format(time.RFC3339)
	}
	return response
}

func (h *ConfigHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			h.logger.Error("failed to encode response", zap.Error(err))
		}
	}
}

func (h *ConfigHandler) respondError(w http.ResponseWriter, status int, message string, err error) {
	response := map[string]interface{}{
		"error": message,
	}
	if err != nil {
		response["detail"] = err.Error()
		h.logger.Error(message, zap.Error(err))
	}
	h.respondJSON(w, status, response)
}
