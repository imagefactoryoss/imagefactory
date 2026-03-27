package rest

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

type VMImageHandler struct {
	db     *sqlx.DB
	logger *zap.Logger
}

type vmImageCatalogItem struct {
	ExecutionID                 uuid.UUID           `db:"execution_id" json:"execution_id"`
	BuildID                     uuid.UUID           `db:"build_id" json:"build_id"`
	ProjectID                   uuid.UUID           `db:"project_id" json:"project_id"`
	ProjectName                 string              `db:"project_name" json:"project_name"`
	BuildNumber                 int                 `db:"build_number" json:"build_number"`
	BuildStatus                 string              `db:"build_status" json:"build_status"`
	ExecutionStatus             string              `db:"execution_status" json:"execution_status"`
	CreatedAt                   time.Time           `db:"created_at" json:"created_at"`
	StartedAt                   *time.Time          `db:"started_at" json:"started_at,omitempty"`
	CompletedAt                 *time.Time          `db:"completed_at" json:"completed_at,omitempty"`
	TargetProvider              string              `db:"target_provider" json:"target_provider"`
	TargetProfileID             string              `db:"target_profile_id" json:"target_profile_id"`
	ProviderArtifactIdentifiers map[string][]string `json:"provider_artifact_identifiers"`
	ArtifactValues              []string            `json:"artifact_values"`
	LifecycleState              string              `json:"lifecycle_state"`
}

type vmImageCatalogListResponse struct {
	Data       []vmImageCatalogItem `json:"data"`
	TotalCount int                  `json:"total_count"`
	Limit      int                  `json:"limit"`
	Offset     int                  `json:"offset"`
}

func NewVMImageHandler(db *sqlx.DB, logger *zap.Logger) *VMImageHandler {
	return &VMImageHandler{db: db, logger: logger}
}

func (h *VMImageHandler) ListTenantVMImages(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil || authCtx.TenantID == uuid.Nil {
		writeVMImageJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if h.db == nil {
		writeVMImageJSON(w, http.StatusInternalServerError, map[string]string{"error": "vm image catalog is unavailable"})
		return
	}

	limit := clampIntQuery(r.URL.Query().Get("limit"), 25, 1, 100)
	offset := clampIntQuery(r.URL.Query().Get("offset"), 0, 0, 10000)
	providerFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("provider")))
	statusFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	countQuery := `
		SELECT COUNT(*)
		FROM build_executions be
		JOIN builds b ON b.id = be.build_id
		JOIN projects p ON p.id = b.project_id
		JOIN build_configs bc ON bc.id = be.config_id
		WHERE b.tenant_id = $1
		  AND bc.build_method = 'packer'
		  AND ($2 = '' OR LOWER(COALESCE(be.metadata #>> '{packer,target_provider}', '')) = $2)
		  AND ($3 = '' OR LOWER(be.status::text) = $3)
		  AND (
		        $4 = ''
		        OR p.name ILIKE '%' || $4 || '%'
		        OR COALESCE(be.metadata #>> '{packer,target_provider}', '') ILIKE '%' || $4 || '%'
		        OR COALESCE(be.metadata #>> '{packer,target_profile_id}', '') ILIKE '%' || $4 || '%'
		        OR COALESCE(be.metadata::text, '') ILIKE '%' || $4 || '%'
		        OR COALESCE(be.artifacts::text, '') ILIKE '%' || $4 || '%'
		      )
	`
	var total int
	if err := h.db.GetContext(r.Context(), &total, countQuery, authCtx.TenantID, providerFilter, statusFilter, search); err != nil {
		h.logger.Error("Failed to count tenant VM images", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()))
		writeVMImageJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list vm images"})
		return
	}

	type vmImageListRow struct {
		ExecutionID     uuid.UUID       `db:"execution_id"`
		BuildID         uuid.UUID       `db:"build_id"`
		ProjectID       uuid.UUID       `db:"project_id"`
		ProjectName     string          `db:"project_name"`
		BuildNumber     int             `db:"build_number"`
		BuildStatus     string          `db:"build_status"`
		ExecutionStatus string          `db:"execution_status"`
		CreatedAt       time.Time       `db:"created_at"`
		StartedAt       *time.Time      `db:"started_at"`
		CompletedAt     *time.Time      `db:"completed_at"`
		MetadataRaw     json.RawMessage `db:"metadata"`
		ArtifactsRaw    json.RawMessage `db:"artifacts"`
	}
	rows := make([]vmImageListRow, 0, limit)
	query := `
		SELECT
			be.id AS execution_id,
			b.id AS build_id,
			p.id AS project_id,
			COALESCE(NULLIF(TRIM(p.name), ''), 'Unknown project') AS project_name,
			b.build_number,
			b.status AS build_status,
			be.status AS execution_status,
			be.created_at,
			be.started_at,
			be.completed_at,
			COALESCE(be.metadata, '{}'::jsonb) AS metadata,
			COALESCE(be.artifacts, '[]'::jsonb) AS artifacts
		FROM build_executions be
		JOIN builds b ON b.id = be.build_id
		JOIN projects p ON p.id = b.project_id
		JOIN build_configs bc ON bc.id = be.config_id
		WHERE b.tenant_id = $1
		  AND bc.build_method = 'packer'
		  AND ($2 = '' OR LOWER(COALESCE(be.metadata #>> '{packer,target_provider}', '')) = $2)
		  AND ($3 = '' OR LOWER(be.status::text) = $3)
		  AND (
		        $4 = ''
		        OR p.name ILIKE '%' || $4 || '%'
		        OR COALESCE(be.metadata #>> '{packer,target_provider}', '') ILIKE '%' || $4 || '%'
		        OR COALESCE(be.metadata #>> '{packer,target_profile_id}', '') ILIKE '%' || $4 || '%'
		        OR COALESCE(be.metadata::text, '') ILIKE '%' || $4 || '%'
		        OR COALESCE(be.artifacts::text, '') ILIKE '%' || $4 || '%'
		      )
		ORDER BY COALESCE(be.completed_at, be.created_at) DESC
		LIMIT $5 OFFSET $6
	`
	if err := h.db.SelectContext(r.Context(), &rows, query, authCtx.TenantID, providerFilter, statusFilter, search, limit, offset); err != nil {
		h.logger.Error("Failed to list tenant VM images", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()))
		writeVMImageJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list vm images"})
		return
	}

	items := make([]vmImageCatalogItem, 0, len(rows))
	for _, row := range rows {
		targetProvider, targetProfileID, providerIdentifiers, lifecycleOverride := parsePackerMetadataFields(row.MetadataRaw)
		artifactValues := extractArtifactValues(row.ArtifactsRaw)
		items = append(items, vmImageCatalogItem{
			ExecutionID:                 row.ExecutionID,
			BuildID:                     row.BuildID,
			ProjectID:                   row.ProjectID,
			ProjectName:                 row.ProjectName,
			BuildNumber:                 row.BuildNumber,
			BuildStatus:                 row.BuildStatus,
			ExecutionStatus:             row.ExecutionStatus,
			CreatedAt:                   row.CreatedAt.UTC(),
			StartedAt:                   row.StartedAt,
			CompletedAt:                 row.CompletedAt,
			TargetProvider:              targetProvider,
			TargetProfileID:             targetProfileID,
			ProviderArtifactIdentifiers: providerIdentifiers,
			ArtifactValues:              artifactValues,
			LifecycleState:              vmImageLifecycleState(row.ExecutionStatus, lifecycleOverride),
		})
	}

	writeVMImageJSON(w, http.StatusOK, vmImageCatalogListResponse{
		Data:       items,
		TotalCount: total,
		Limit:      limit,
		Offset:     offset,
	})
}

func (h *VMImageHandler) GetTenantVMImage(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil || authCtx.TenantID == uuid.Nil {
		writeVMImageJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if h.db == nil {
		writeVMImageJSON(w, http.StatusInternalServerError, map[string]string{"error": "vm image catalog is unavailable"})
		return
	}

	executionID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "executionId")))
	if err != nil {
		writeVMImageJSON(w, http.StatusBadRequest, map[string]string{"error": "executionId must be a valid uuid"})
		return
	}

	var row vmImageRow
	query := `
		SELECT
			be.id AS execution_id,
			b.id AS build_id,
			p.id AS project_id,
			COALESCE(NULLIF(TRIM(p.name), ''), 'Unknown project') AS project_name,
			b.build_number,
			b.status AS build_status,
			be.status AS execution_status,
			be.created_at,
			be.started_at,
			be.completed_at,
			COALESCE(be.metadata, '{}'::jsonb) AS metadata,
			COALESCE(be.artifacts, '[]'::jsonb) AS artifacts
		FROM build_executions be
		JOIN builds b ON b.id = be.build_id
		JOIN projects p ON p.id = b.project_id
		JOIN build_configs bc ON bc.id = be.config_id
		WHERE be.id = $1
		  AND b.tenant_id = $2
		  AND bc.build_method = 'packer'
	`
	if err := h.db.GetContext(r.Context(), &row, query, executionID, authCtx.TenantID); err != nil {
		writeVMImageJSON(w, http.StatusNotFound, map[string]string{"error": "vm image execution not found"})
		return
	}

	targetProvider, targetProfileID, providerIdentifiers, lifecycleOverride := parsePackerMetadataFields(row.MetadataRaw)
	item := vmImageCatalogItem{
		ExecutionID:                 row.ExecutionID,
		BuildID:                     row.BuildID,
		ProjectID:                   row.ProjectID,
		ProjectName:                 row.ProjectName,
		BuildNumber:                 row.BuildNumber,
		BuildStatus:                 row.BuildStatus,
		ExecutionStatus:             row.ExecutionStatus,
		CreatedAt:                   row.CreatedAt.UTC(),
		StartedAt:                   row.StartedAt,
		CompletedAt:                 row.CompletedAt,
		TargetProvider:              targetProvider,
		TargetProfileID:             targetProfileID,
		ProviderArtifactIdentifiers: providerIdentifiers,
		ArtifactValues:              extractArtifactValues(row.ArtifactsRaw),
		LifecycleState:              vmImageLifecycleState(row.ExecutionStatus, lifecycleOverride),
	}

	writeVMImageJSON(w, http.StatusOK, map[string]vmImageCatalogItem{"data": item})
}

func (h *VMImageHandler) PromoteTenantVMImage(w http.ResponseWriter, r *http.Request) {
	h.transitionTenantVMImageLifecycle(w, r, "released", false)
}

func (h *VMImageHandler) DeprecateTenantVMImage(w http.ResponseWriter, r *http.Request) {
	h.transitionTenantVMImageLifecycle(w, r, "deprecated", true)
}

func (h *VMImageHandler) DeleteTenantVMImage(w http.ResponseWriter, r *http.Request) {
	h.transitionTenantVMImageLifecycle(w, r, "deleted", true)
}

func clampIntQuery(raw string, fallback, min, max int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if parsed < min {
		return min
	}
	if parsed > max {
		return max
	}
	return parsed
}

func vmImageLifecycleState(executionStatus, lifecycleOverride string) string {
	lifecycleOverride = strings.ToLower(strings.TrimSpace(lifecycleOverride))
	switch lifecycleOverride {
	case "released", "deprecated", "deleted":
		return lifecycleOverride
	}

	switch strings.ToLower(strings.TrimSpace(executionStatus)) {
	case "success":
		return "available"
	case "running":
		return "building"
	case "pending":
		return "queued"
	case "cancelled":
		return "cancelled"
	case "failed":
		return "failed"
	default:
		return "unknown"
	}
}

func parsePackerMetadataFields(raw json.RawMessage) (targetProvider, targetProfileID string, providerIdentifiers map[string][]string, lifecycleOverride string) {
	type packerMetadata struct {
		TargetProvider              string              `json:"target_provider"`
		TargetProfileID             string              `json:"target_profile_id"`
		ProviderArtifactIdentifiers map[string][]string `json:"provider_artifact_identifiers"`
		LifecycleState              string              `json:"lifecycle_state"`
	}
	type executionMetadata struct {
		Packer packerMetadata `json:"packer"`
	}
	var parsed executionMetadata
	if len(raw) == 0 || json.Unmarshal(raw, &parsed) != nil {
		return "", "", map[string][]string{}, ""
	}
	targetProvider = strings.TrimSpace(parsed.Packer.TargetProvider)
	targetProfileID = strings.TrimSpace(parsed.Packer.TargetProfileID)
	lifecycleOverride = strings.TrimSpace(parsed.Packer.LifecycleState)
	out := make(map[string][]string, len(parsed.Packer.ProviderArtifactIdentifiers))
	for provider, values := range parsed.Packer.ProviderArtifactIdentifiers {
		normalized := make([]string, 0, len(values))
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value != "" {
				normalized = append(normalized, value)
			}
		}
		if len(normalized) == 0 {
			continue
		}
		sort.Strings(normalized)
		out[strings.ToLower(strings.TrimSpace(provider))] = normalized
	}
	return targetProvider, targetProfileID, out, lifecycleOverride
}

func extractArtifactValues(raw json.RawMessage) []string {
	type artifact struct {
		Name  string `json:"name"`
		Value string `json:"value"`
		Path  string `json:"path"`
	}
	values := make([]string, 0, 16)
	seen := map[string]struct{}{}
	appendValue := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, exists := seen[v]; exists {
			return
		}
		seen[v] = struct{}{}
		values = append(values, v)
	}

	var artifacts []artifact
	if len(raw) > 0 && json.Unmarshal(raw, &artifacts) == nil {
		for _, item := range artifacts {
			appendValue(item.Name)
			appendValue(item.Value)
			appendValue(item.Path)
		}
	}
	if len(values) > 0 {
		sort.Strings(values)
		return values
	}

	var generic []string
	if len(raw) > 0 && json.Unmarshal(raw, &generic) == nil {
		for _, item := range generic {
			appendValue(item)
		}
	}
	sort.Strings(values)
	return values
}

func writeVMImageJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type vmImageLifecycleActionRequest struct {
	Reason string `json:"reason"`
}

func (h *VMImageHandler) transitionTenantVMImageLifecycle(w http.ResponseWriter, r *http.Request, targetState string, allowReason bool) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil || authCtx.TenantID == uuid.Nil {
		writeVMImageJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if h.db == nil {
		writeVMImageJSON(w, http.StatusInternalServerError, map[string]string{"error": "vm image catalog is unavailable"})
		return
	}

	executionID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "executionId")))
	if err != nil {
		writeVMImageJSON(w, http.StatusBadRequest, map[string]string{"error": "executionId must be a valid uuid"})
		return
	}

	reqBody := vmImageLifecycleActionRequest{}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
	}
	reason := strings.TrimSpace(reqBody.Reason)
	if reason == "" && allowReason {
		reason = fmt.Sprintf("lifecycle transitioned to %s", targetState)
	}

	row, err := h.getTenantVMImageRow(r, authCtx.TenantID, executionID)
	if err != nil {
		status := http.StatusInternalServerError
		message := "failed to load vm image"
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
			message = "vm image execution not found"
		}
		writeVMImageJSON(w, status, map[string]string{"error": message})
		return
	}

	_, _, _, lifecycleOverride := parsePackerMetadataFields(row.MetadataRaw)
	currentLifecycle := vmImageLifecycleState(row.ExecutionStatus, lifecycleOverride)
	if strings.EqualFold(row.ExecutionStatus, "running") || strings.EqualFold(row.ExecutionStatus, "pending") {
		writeVMImageJSON(w, http.StatusConflict, map[string]string{"error": "cannot transition lifecycle while build execution is active"})
		return
	}
	if currentLifecycle == "failed" || currentLifecycle == "cancelled" {
		writeVMImageJSON(w, http.StatusConflict, map[string]string{"error": "cannot transition lifecycle for failed or cancelled executions"})
		return
	}
	if currentLifecycle == "deleted" && targetState != "deleted" {
		writeVMImageJSON(w, http.StatusConflict, map[string]string{"error": "deleted vm image cannot be transitioned"})
		return
	}

	nextMetadata, err := updatePackerLifecycleMetadata(row.MetadataRaw, targetState, reason, authCtx.UserID, time.Now().UTC())
	if err != nil {
		writeVMImageJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to prepare lifecycle metadata"})
		return
	}

	updateQuery := `
		UPDATE build_executions AS be
		SET metadata = $1
		FROM builds b
		JOIN build_configs bc ON bc.id = be.config_id
		WHERE be.id = $2
		  AND b.id = be.build_id
		  AND b.tenant_id = $3
		  AND bc.build_method = 'packer'
	`
	if _, err := h.db.ExecContext(r.Context(), updateQuery, nextMetadata, executionID, authCtx.TenantID); err != nil {
		h.logger.Error("Failed to update VM image lifecycle metadata",
			zap.String("execution_id", executionID.String()),
			zap.String("tenant_id", authCtx.TenantID.String()),
			zap.String("target_state", targetState),
			zap.Error(err))
		writeVMImageJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update vm image lifecycle"})
		return
	}

	updatedRow, err := h.getTenantVMImageRow(r, authCtx.TenantID, executionID)
	if err != nil {
		writeVMImageJSON(w, http.StatusInternalServerError, map[string]string{"error": "vm image lifecycle updated but failed to reload response"})
		return
	}
	targetProvider, targetProfileID, providerIdentifiers, updatedLifecycleOverride := parsePackerMetadataFields(updatedRow.MetadataRaw)
	item := vmImageCatalogItem{
		ExecutionID:                 updatedRow.ExecutionID,
		BuildID:                     updatedRow.BuildID,
		ProjectID:                   updatedRow.ProjectID,
		ProjectName:                 updatedRow.ProjectName,
		BuildNumber:                 updatedRow.BuildNumber,
		BuildStatus:                 updatedRow.BuildStatus,
		ExecutionStatus:             updatedRow.ExecutionStatus,
		CreatedAt:                   updatedRow.CreatedAt.UTC(),
		StartedAt:                   updatedRow.StartedAt,
		CompletedAt:                 updatedRow.CompletedAt,
		TargetProvider:              targetProvider,
		TargetProfileID:             targetProfileID,
		ProviderArtifactIdentifiers: providerIdentifiers,
		ArtifactValues:              extractArtifactValues(updatedRow.ArtifactsRaw),
		LifecycleState:              vmImageLifecycleState(updatedRow.ExecutionStatus, updatedLifecycleOverride),
	}
	writeVMImageJSON(w, http.StatusOK, map[string]interface{}{
		"data":    item,
		"message": fmt.Sprintf("vm image lifecycle transitioned to %s", targetState),
	})
}

type vmImageRow struct {
	ExecutionID     uuid.UUID       `db:"execution_id"`
	BuildID         uuid.UUID       `db:"build_id"`
	ProjectID       uuid.UUID       `db:"project_id"`
	ProjectName     string          `db:"project_name"`
	BuildNumber     int             `db:"build_number"`
	BuildStatus     string          `db:"build_status"`
	ExecutionStatus string          `db:"execution_status"`
	CreatedAt       time.Time       `db:"created_at"`
	StartedAt       *time.Time      `db:"started_at"`
	CompletedAt     *time.Time      `db:"completed_at"`
	MetadataRaw     json.RawMessage `db:"metadata"`
	ArtifactsRaw    json.RawMessage `db:"artifacts"`
}

func (h *VMImageHandler) getTenantVMImageRow(r *http.Request, tenantID, executionID uuid.UUID) (*vmImageRow, error) {
	var row vmImageRow
	query := `
		SELECT
			be.id AS execution_id,
			b.id AS build_id,
			p.id AS project_id,
			COALESCE(NULLIF(TRIM(p.name), ''), 'Unknown project') AS project_name,
			b.build_number,
			b.status AS build_status,
			be.status AS execution_status,
			be.created_at,
			be.started_at,
			be.completed_at,
			COALESCE(be.metadata, '{}'::jsonb) AS metadata,
			COALESCE(be.artifacts, '[]'::jsonb) AS artifacts
		FROM build_executions be
		JOIN builds b ON b.id = be.build_id
		JOIN projects p ON p.id = b.project_id
		JOIN build_configs bc ON bc.id = be.config_id
		WHERE be.id = $1
		  AND b.tenant_id = $2
		  AND bc.build_method = 'packer'
	`
	if err := h.db.GetContext(r.Context(), &row, query, executionID, tenantID); err != nil {
		return nil, err
	}
	return &row, nil
}

func updatePackerLifecycleMetadata(raw json.RawMessage, targetState, reason string, userID uuid.UUID, at time.Time) (json.RawMessage, error) {
	metadata := map[string]interface{}{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &metadata); err != nil {
			return nil, err
		}
	}
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	packer, _ := metadata["packer"].(map[string]interface{})
	if packer == nil {
		packer = map[string]interface{}{}
	}
	packer["lifecycle_state"] = strings.ToLower(strings.TrimSpace(targetState))
	packer["lifecycle_last_action_at"] = at.UTC().Format(time.RFC3339)
	packer["lifecycle_last_action_by"] = userID.String()
	if strings.TrimSpace(reason) != "" {
		packer["lifecycle_last_reason"] = strings.TrimSpace(reason)
	}
	metadata["packer"] = packer
	return json.Marshal(metadata)
}
