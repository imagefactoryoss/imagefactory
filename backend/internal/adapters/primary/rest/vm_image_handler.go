package rest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	awscore "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

type VMImageHandler struct {
	db                *sqlx.DB
	logger            *zap.Logger
	lifecycleExecutor vmLifecycleExecutor
}

type vmLifecycleTransitionRequest struct {
	ExecutionID                 uuid.UUID
	TenantID                    uuid.UUID
	TargetProvider              string
	ProviderArtifactIdentifiers map[string][]string
	ArtifactValues              []string
	TargetState                 string
	Reason                      string
}

type vmLifecycleTransitionResult struct {
	TransitionMode string
}

type vmLifecycleExecutor interface {
	ExecuteTransition(ctx context.Context, req vmLifecycleTransitionRequest) (vmLifecycleTransitionResult, error)
}

type vmLifecycleExecutionMode string

const (
	vmLifecycleExecutionModeMetadataOnly          vmLifecycleExecutionMode = "metadata_only"
	vmLifecycleExecutionModePreferProviderNative  vmLifecycleExecutionMode = "prefer_provider_native"
	vmLifecycleExecutionModeRequireProviderNative vmLifecycleExecutionMode = "require_provider_native"
)

var errUnsupportedProviderLifecycleTransition = errors.New("provider lifecycle transition is not implemented")
var errInvalidProviderLifecycleTransitionInput = errors.New("provider lifecycle transition is missing required metadata")

var awsAMIIDPattern = regexp.MustCompile(`\bami-[0-9a-fA-F]{8,17}\b`)
var awsImageARNPattern = regexp.MustCompile(`^arn:aws[a-zA-Z-]*:ec2:([a-z0-9-]+):[0-9]*:image/(ami-[0-9a-fA-F]{8,17})$`)

type awsLifecycleImageReference struct {
	Region  string
	ImageID string
}

type vmAWSLifecycleClient interface {
	DeregisterImage(ctx context.Context, params *ec2.DeregisterImageInput, optFns ...func(*ec2.Options)) (*ec2.DeregisterImageOutput, error)
	EnableImageDeprecation(ctx context.Context, params *ec2.EnableImageDeprecationInput, optFns ...func(*ec2.Options)) (*ec2.EnableImageDeprecationOutput, error)
	DisableImageDeprecation(ctx context.Context, params *ec2.DisableImageDeprecationInput, optFns ...func(*ec2.Options)) (*ec2.DisableImageDeprecationOutput, error)
}

type vmDispatchLifecycleExecutor struct {
	mode             vmLifecycleExecutionMode
	awsClientFactory func(ctx context.Context, region string) (vmAWSLifecycleClient, error)
}

func (e vmDispatchLifecycleExecutor) ExecuteTransition(ctx context.Context, req vmLifecycleTransitionRequest) (vmLifecycleTransitionResult, error) {
	provider := strings.ToLower(strings.TrimSpace(req.TargetProvider))
	if e.mode == vmLifecycleExecutionModeMetadataOnly {
		return vmLifecycleTransitionResult{TransitionMode: "metadata_only"}, nil
	}

	if provider != "aws" || (req.TargetState != "deleted" && req.TargetState != "deprecated" && req.TargetState != "released") {
		if e.mode == vmLifecycleExecutionModeRequireProviderNative {
			if provider == "" {
				provider = "unknown"
			}
			return vmLifecycleTransitionResult{}, fmt.Errorf("%w for provider %s", errUnsupportedProviderLifecycleTransition, provider)
		}
		return vmLifecycleTransitionResult{TransitionMode: "metadata_only"}, nil
	}

	ref, err := selectAWSLifecycleImageReference(req.ProviderArtifactIdentifiers, req.ArtifactValues, strings.TrimSpace(os.Getenv("IF_VM_LIFECYCLE_AWS_REGION")))
	if err != nil {
		return vmLifecycleTransitionResult{}, err
	}

	clientFactory := e.awsClientFactory
	if clientFactory == nil {
		clientFactory = newVMAWSLifecycleClient
	}
	client, err := clientFactory(ctx, ref.Region)
	if err != nil {
		return vmLifecycleTransitionResult{}, err
	}
	if req.TargetState == "deleted" {
		if _, err := client.DeregisterImage(ctx, &ec2.DeregisterImageInput{ImageId: awscore.String(ref.ImageID)}); err != nil {
			return vmLifecycleTransitionResult{}, err
		}
	}

	if req.TargetState == "deprecated" {
		deprecateAt := resolveVMAWSDeprecationTimestamp()
		if _, err := client.EnableImageDeprecation(ctx, &ec2.EnableImageDeprecationInput{
			ImageId:     awscore.String(ref.ImageID),
			DeprecateAt: awscore.Time(deprecateAt),
		}); err != nil {
			return vmLifecycleTransitionResult{}, err
		}
	}
	if req.TargetState == "released" {
		if _, err := client.DisableImageDeprecation(ctx, &ec2.DisableImageDeprecationInput{
			ImageId: awscore.String(ref.ImageID),
		}); err != nil {
			return vmLifecycleTransitionResult{}, err
		}
	}

	return vmLifecycleTransitionResult{TransitionMode: "provider_native"}, nil
}

func newVMAWSLifecycleClient(ctx context.Context, region string) (vmAWSLifecycleClient, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(strings.TrimSpace(region)))
	if err != nil {
		return nil, err
	}
	return ec2.NewFromConfig(cfg), nil
}

func selectAWSLifecycleImageReference(providerArtifactIdentifiers map[string][]string, artifactValues []string, defaultRegion string) (awsLifecycleImageReference, error) {
	candidates := make([]string, 0, len(providerArtifactIdentifiers["aws"])+len(artifactValues))
	candidates = append(candidates, providerArtifactIdentifiers["aws"]...)
	candidates = append(candidates, artifactValues...)
	if len(candidates) == 0 {
		return awsLifecycleImageReference{}, fmt.Errorf("%w: aws image identifier", errInvalidProviderLifecycleTransitionInput)
	}

	for _, candidate := range candidates {
		if ref, ok := parseAWSLifecycleImageReference(candidate, defaultRegion); ok {
			return ref, nil
		}
	}

	return awsLifecycleImageReference{}, fmt.Errorf("%w: aws image identifier parse failed", errInvalidProviderLifecycleTransitionInput)
}

func parseAWSLifecycleImageReference(raw, defaultRegion string) (awsLifecycleImageReference, bool) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return awsLifecycleImageReference{}, false
	}

	if matches := awsImageARNPattern.FindStringSubmatch(candidate); len(matches) == 3 {
		return awsLifecycleImageReference{
			Region:  strings.ToLower(strings.TrimSpace(matches[1])),
			ImageID: strings.ToLower(strings.TrimSpace(matches[2])),
		}, true
	}

	if strings.Contains(candidate, ":") {
		parts := strings.SplitN(candidate, ":", 2)
		region := strings.ToLower(strings.TrimSpace(parts[0]))
		image := strings.TrimSpace(parts[1])
		if region != "" && awsAMIIDPattern.MatchString(image) {
			return awsLifecycleImageReference{
				Region:  region,
				ImageID: strings.ToLower(awsAMIIDPattern.FindString(image)),
			}, true
		}
	}

	if awsAMIIDPattern.MatchString(candidate) {
		region := strings.ToLower(strings.TrimSpace(defaultRegion))
		if region == "" {
			return awsLifecycleImageReference{}, false
		}
		return awsLifecycleImageReference{
			Region:  region,
			ImageID: strings.ToLower(awsAMIIDPattern.FindString(candidate)),
		}, true
	}

	return awsLifecycleImageReference{}, false
}

func resolveVMAWSDeprecationTimestamp() time.Time {
	hours := 24
	if raw := strings.TrimSpace(os.Getenv("IF_VM_LIFECYCLE_AWS_DEPRECATION_HOURS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 24*365 {
			hours = parsed
		}
	}
	return time.Now().UTC().Add(time.Duration(hours) * time.Hour)
}

func resolveVMLifecycleExecutionMode(logger *zap.Logger) vmLifecycleExecutionMode {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("IF_VM_LIFECYCLE_EXECUTION_MODE")))
	switch vmLifecycleExecutionMode(raw) {
	case "", vmLifecycleExecutionModeMetadataOnly:
		return vmLifecycleExecutionModeMetadataOnly
	case vmLifecycleExecutionModePreferProviderNative:
		return vmLifecycleExecutionModePreferProviderNative
	case vmLifecycleExecutionModeRequireProviderNative:
		return vmLifecycleExecutionModeRequireProviderNative
	default:
		if logger != nil {
			logger.Warn("Invalid IF_VM_LIFECYCLE_EXECUTION_MODE; defaulting to metadata_only",
				zap.String("value", raw))
		}
		return vmLifecycleExecutionModeMetadataOnly
	}
}

type vmImageCatalogItem struct {
	ExecutionID                 uuid.UUID                `db:"execution_id" json:"execution_id"`
	BuildID                     uuid.UUID                `db:"build_id" json:"build_id"`
	ProjectID                   uuid.UUID                `db:"project_id" json:"project_id"`
	ProjectName                 string                   `db:"project_name" json:"project_name"`
	BuildNumber                 int                      `db:"build_number" json:"build_number"`
	BuildStatus                 string                   `db:"build_status" json:"build_status"`
	ExecutionStatus             string                   `db:"execution_status" json:"execution_status"`
	CreatedAt                   time.Time                `db:"created_at" json:"created_at"`
	StartedAt                   *time.Time               `db:"started_at" json:"started_at,omitempty"`
	CompletedAt                 *time.Time               `db:"completed_at" json:"completed_at,omitempty"`
	TargetProvider              string                   `db:"target_provider" json:"target_provider"`
	TargetProfileID             string                   `db:"target_profile_id" json:"target_profile_id"`
	ProviderArtifactIdentifiers map[string][]string      `json:"provider_artifact_identifiers"`
	ArtifactValues              []string                 `json:"artifact_values"`
	LifecycleState              string                   `json:"lifecycle_state"`
	LifecycleLastActionAt       string                   `json:"lifecycle_last_action_at,omitempty"`
	LifecycleLastActionBy       string                   `json:"lifecycle_last_action_by,omitempty"`
	LifecycleLastReason         string                   `json:"lifecycle_last_reason,omitempty"`
	LifecycleTransitionMode     string                   `json:"lifecycle_transition_mode"`
	LifecycleHistory            []vmLifecycleHistory     `json:"lifecycle_history,omitempty"`
	ActionPermissions           vmImageActionPermissions `json:"action_permissions"`
}

type vmImageActionPermissions struct {
	CanPromote   bool `json:"can_promote"`
	CanDeprecate bool `json:"can_deprecate"`
	CanDelete    bool `json:"can_delete"`
}

type vmLifecycleHistory struct {
	State          string `json:"state"`
	Reason         string `json:"reason,omitempty"`
	ActorID        string `json:"actor_id,omitempty"`
	At             string `json:"at,omitempty"`
	TransitionMode string `json:"transition_mode,omitempty"`
}

type vmImageCatalogListResponse struct {
	Data       []vmImageCatalogItem `json:"data"`
	TotalCount int                  `json:"total_count"`
	Limit      int                  `json:"limit"`
	Offset     int                  `json:"offset"`
}

func NewVMImageHandler(db *sqlx.DB, logger *zap.Logger) *VMImageHandler {
	mode := resolveVMLifecycleExecutionMode(logger)
	return &VMImageHandler{
		db:     db,
		logger: logger,
		lifecycleExecutor: vmDispatchLifecycleExecutor{
			mode:             mode,
			awsClientFactory: newVMAWSLifecycleClient,
		},
	}
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
	transitionModeFilter := vmImageLifecycleTransitionMode(r.URL.Query().Get("transition_mode"))
	if strings.TrimSpace(r.URL.Query().Get("transition_mode")) == "" {
		transitionModeFilter = ""
	}
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
		  AND ($4 = '' OR LOWER(COALESCE(NULLIF(TRIM(be.metadata #>> '{packer,lifecycle_transition_mode}'), ''), 'metadata_only')) = $4)
		  AND (
		        $5 = ''
		        OR p.name ILIKE '%' || $5 || '%'
		        OR COALESCE(be.metadata #>> '{packer,target_provider}', '') ILIKE '%' || $5 || '%'
		        OR COALESCE(be.metadata #>> '{packer,target_profile_id}', '') ILIKE '%' || $5 || '%'
		        OR COALESCE(be.metadata::text, '') ILIKE '%' || $5 || '%'
		        OR COALESCE(be.artifacts::text, '') ILIKE '%' || $5 || '%'
		      )
	`
	var total int
	if err := h.db.GetContext(r.Context(), &total, countQuery, authCtx.TenantID, providerFilter, statusFilter, transitionModeFilter, search); err != nil {
		h.logger.Error("Failed to count tenant VM images", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()))
		writeVMImageJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list vm images"})
		return
	}

	rows := make([]vmImageRow, 0, limit)
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
		  AND ($4 = '' OR LOWER(COALESCE(NULLIF(TRIM(be.metadata #>> '{packer,lifecycle_transition_mode}'), ''), 'metadata_only')) = $4)
		  AND (
		        $5 = ''
		        OR p.name ILIKE '%' || $5 || '%'
		        OR COALESCE(be.metadata #>> '{packer,target_provider}', '') ILIKE '%' || $5 || '%'
		        OR COALESCE(be.metadata #>> '{packer,target_profile_id}', '') ILIKE '%' || $5 || '%'
		        OR COALESCE(be.metadata::text, '') ILIKE '%' || $5 || '%'
		        OR COALESCE(be.artifacts::text, '') ILIKE '%' || $5 || '%'
		      )
		ORDER BY COALESCE(be.completed_at, be.created_at) DESC
		LIMIT $6 OFFSET $7
	`
	if err := h.db.SelectContext(r.Context(), &rows, query, authCtx.TenantID, providerFilter, statusFilter, transitionModeFilter, search, limit, offset); err != nil {
		h.logger.Error("Failed to list tenant VM images", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()))
		writeVMImageJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list vm images"})
		return
	}

	items := make([]vmImageCatalogItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, vmImageCatalogItemFromRow(row))
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

	item := vmImageCatalogItemFromRow(row)

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

func vmImageLifecycleActionPermissions(executionStatus, lifecycleState string) vmImageActionPermissions {
	exec := strings.ToLower(strings.TrimSpace(executionStatus))
	state := strings.ToLower(strings.TrimSpace(lifecycleState))
	if exec == "running" || exec == "pending" {
		return vmImageActionPermissions{}
	}
	switch state {
	case "failed", "cancelled", "unknown", "deleted":
		return vmImageActionPermissions{}
	}
	return vmImageActionPermissions{
		CanPromote:   state == "available" || state == "deprecated",
		CanDeprecate: state == "available" || state == "released",
		CanDelete:    state == "deprecated",
	}
}

type vmImageLifecycleMetadata struct {
	State          string
	LastActionAt   string
	LastActionBy   string
	LastReason     string
	TransitionMode string
	History        []vmLifecycleHistory
}

func defaultVMLifecycleMetadata() vmImageLifecycleMetadata {
	return vmImageLifecycleMetadata{
		TransitionMode: "metadata_only",
	}
}

func parsePackerMetadataFields(raw json.RawMessage) (targetProvider, targetProfileID string, providerIdentifiers map[string][]string, lifecycle vmImageLifecycleMetadata) {
	type packerMetadata struct {
		TargetProvider              string               `json:"target_provider"`
		TargetProfileID             string               `json:"target_profile_id"`
		ProviderArtifactIdentifiers map[string][]string  `json:"provider_artifact_identifiers"`
		LifecycleState              string               `json:"lifecycle_state"`
		LifecycleLastActionAt       string               `json:"lifecycle_last_action_at"`
		LifecycleLastActionBy       string               `json:"lifecycle_last_action_by"`
		LifecycleLastReason         string               `json:"lifecycle_last_reason"`
		LifecycleTransitionMode     string               `json:"lifecycle_transition_mode"`
		LifecycleHistory            []vmLifecycleHistory `json:"lifecycle_history"`
	}
	type executionMetadata struct {
		Packer packerMetadata `json:"packer"`
	}
	var parsed executionMetadata
	if len(raw) == 0 || json.Unmarshal(raw, &parsed) != nil {
		return "", "", map[string][]string{}, defaultVMLifecycleMetadata()
	}
	targetProvider = strings.TrimSpace(parsed.Packer.TargetProvider)
	targetProfileID = strings.TrimSpace(parsed.Packer.TargetProfileID)
	lifecycle = vmImageLifecycleMetadata{
		State:          strings.TrimSpace(parsed.Packer.LifecycleState),
		LastActionAt:   strings.TrimSpace(parsed.Packer.LifecycleLastActionAt),
		LastActionBy:   strings.TrimSpace(parsed.Packer.LifecycleLastActionBy),
		LastReason:     strings.TrimSpace(parsed.Packer.LifecycleLastReason),
		TransitionMode: vmImageLifecycleTransitionMode(parsed.Packer.LifecycleTransitionMode),
		History:        sanitizeLifecycleHistory(parsed.Packer.LifecycleHistory),
	}
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
	return targetProvider, targetProfileID, out, lifecycle
}

func vmImageLifecycleTransitionMode(raw string) string {
	mode := strings.ToLower(strings.TrimSpace(raw))
	switch mode {
	case "", "metadata_only", "provider_native", "hybrid":
		if mode == "" {
			return "metadata_only"
		}
		return mode
	default:
		return "metadata_only"
	}
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

	if len(values) == 0 {
		var anyPayload interface{}
		if len(raw) > 0 && json.Unmarshal(raw, &anyPayload) == nil {
			collectArtifactStringValues(anyPayload, appendValue)
		}
	}
	sort.Strings(values)
	return values
}

func collectArtifactStringValues(raw interface{}, appendValue func(string)) {
	switch typed := raw.(type) {
	case string:
		appendValue(typed)
	case []interface{}:
		for _, item := range typed {
			collectArtifactStringValues(item, appendValue)
		}
	case map[string]interface{}:
		for _, value := range typed {
			collectArtifactStringValues(value, appendValue)
		}
	}
}

func writeVMImageJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type vmImageLifecycleActionRequest struct {
	Reason string `json:"reason"`
}

const vmImageLifecycleReasonMaxLength = 500

func validateVMLifecycleReason(raw string, required bool) (string, error) {
	reason := strings.TrimSpace(raw)
	if required && reason == "" {
		return "", errors.New("reason is required for this lifecycle transition")
	}
	if len(reason) > vmImageLifecycleReasonMaxLength {
		return "", fmt.Errorf("reason must be %d characters or fewer", vmImageLifecycleReasonMaxLength)
	}
	return reason, nil
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
	reason, err := validateVMLifecycleReason(reqBody.Reason, allowReason)
	if err != nil {
		writeVMImageJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
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

	targetProvider, _, providerIdentifiers, lifecycle := parsePackerMetadataFields(row.MetadataRaw)
	currentLifecycle := vmImageLifecycleState(row.ExecutionStatus, lifecycle.State)
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
	if targetState == "deleted" && currentLifecycle != "deprecated" && currentLifecycle != "deleted" {
		writeVMImageJSON(w, http.StatusConflict, map[string]string{"error": "vm image must be deprecated before delete transition"})
		return
	}
	if currentLifecycle == targetState {
		writeVMImageJSON(w, http.StatusOK, map[string]interface{}{
			"data":    vmImageCatalogItemFromRow(*row),
			"message": fmt.Sprintf("vm image already in %s lifecycle state", targetState),
		})
		return
	}

	transitionMode := "metadata_only"
	if h.lifecycleExecutor != nil {
		result, execErr := h.lifecycleExecutor.ExecuteTransition(r.Context(), vmLifecycleTransitionRequest{
			ExecutionID:                 executionID,
			TenantID:                    authCtx.TenantID,
			TargetProvider:              targetProvider,
			ProviderArtifactIdentifiers: providerIdentifiers,
			ArtifactValues:              extractArtifactValues(row.ArtifactsRaw),
			TargetState:                 targetState,
			Reason:                      reason,
		})
		if execErr != nil {
			if errors.Is(execErr, errUnsupportedProviderLifecycleTransition) {
				writeVMImageJSON(w, http.StatusNotImplemented, map[string]string{"error": execErr.Error()})
				return
			}
			if errors.Is(execErr, errInvalidProviderLifecycleTransitionInput) {
				writeVMImageJSON(w, http.StatusBadRequest, map[string]string{"error": execErr.Error()})
				return
			}
			h.logger.Error("Failed provider lifecycle transition execution",
				zap.String("execution_id", executionID.String()),
				zap.String("tenant_id", authCtx.TenantID.String()),
				zap.String("target_state", targetState),
				zap.String("target_provider", targetProvider),
				zap.Error(execErr))
			writeVMImageJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to execute provider lifecycle transition"})
			return
		}
		transitionMode = vmImageLifecycleTransitionMode(result.TransitionMode)
	}

	nextMetadata, err := updatePackerLifecycleMetadata(row.MetadataRaw, targetState, reason, transitionMode, authCtx.UserID, time.Now().UTC())
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
	item := vmImageCatalogItemFromRow(*updatedRow)
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

func vmImageCatalogItemFromRow(row vmImageRow) vmImageCatalogItem {
	targetProvider, targetProfileID, providerIdentifiers, lifecycle := parsePackerMetadataFields(row.MetadataRaw)
	lifecycleState := vmImageLifecycleState(row.ExecutionStatus, lifecycle.State)
	lifecycleTransitionMode := vmImageLifecycleTransitionMode(lifecycle.TransitionMode)
	return vmImageCatalogItem{
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
		LifecycleState:              lifecycleState,
		LifecycleLastActionAt:       lifecycle.LastActionAt,
		LifecycleLastActionBy:       lifecycle.LastActionBy,
		LifecycleLastReason:         lifecycle.LastReason,
		LifecycleTransitionMode:     lifecycleTransitionMode,
		LifecycleHistory:            lifecycle.History,
		ActionPermissions:           vmImageLifecycleActionPermissions(row.ExecutionStatus, lifecycleState),
	}
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

func updatePackerLifecycleMetadata(raw json.RawMessage, targetState, reason, transitionMode string, userID uuid.UUID, at time.Time) (json.RawMessage, error) {
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
	transitionMode = vmImageLifecycleTransitionMode(transitionMode)
	packer["lifecycle_state"] = strings.ToLower(strings.TrimSpace(targetState))
	packer["lifecycle_transition_mode"] = transitionMode
	packer["lifecycle_last_action_at"] = at.UTC().Format(time.RFC3339)
	packer["lifecycle_last_action_by"] = userID.String()
	if strings.TrimSpace(reason) != "" {
		packer["lifecycle_last_reason"] = strings.TrimSpace(reason)
	}
	history := sanitizeLifecycleHistory(interfaceToLifecycleHistory(packer["lifecycle_history"]))
	history = append(history, vmLifecycleHistory{
		State:          strings.ToLower(strings.TrimSpace(targetState)),
		Reason:         strings.TrimSpace(reason),
		ActorID:        userID.String(),
		At:             at.UTC().Format(time.RFC3339),
		TransitionMode: transitionMode,
	})
	if len(history) > 25 {
		history = history[len(history)-25:]
	}
	packer["lifecycle_history"] = history
	metadata["packer"] = packer
	return json.Marshal(metadata)
}

func interfaceToLifecycleHistory(raw interface{}) []vmLifecycleHistory {
	if raw == nil {
		return nil
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var history []vmLifecycleHistory
	if err := json.Unmarshal(payload, &history); err != nil {
		return nil
	}
	return history
}

func sanitizeLifecycleHistory(history []vmLifecycleHistory) []vmLifecycleHistory {
	if len(history) == 0 {
		return nil
	}
	out := make([]vmLifecycleHistory, 0, len(history))
	for _, entry := range history {
		state := strings.ToLower(strings.TrimSpace(entry.State))
		if state == "" {
			continue
		}
		out = append(out, vmLifecycleHistory{
			State:          state,
			Reason:         strings.TrimSpace(entry.Reason),
			ActorID:        strings.TrimSpace(entry.ActorID),
			At:             strings.TrimSpace(entry.At),
			TransitionMode: vmImageLifecycleTransitionMode(entry.TransitionMode),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
