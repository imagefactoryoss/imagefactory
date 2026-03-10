package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/srikarm/image-factory/internal/infrastructure/tektoncompat"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/domain/infrastructure/connectors"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// InfrastructureProviderHandler handles infrastructure provider HTTP requests
type InfrastructureProviderHandler struct {
	service *infrastructure.Service
	logger  *zap.Logger
}

type scopedProviderAccess struct {
	authCtx     *middleware.AuthContext
	scopeTenant uuid.UUID
	allTenants  bool
	providerID  uuid.UUID
	provider    *infrastructure.Provider
}

const maxProvidersWithPrepareSummary = 500

var providerPrepareStreamUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NewInfrastructureProviderHandler creates a new infrastructure provider handler
func NewInfrastructureProviderHandler(service *infrastructure.Service, logger *zap.Logger) *InfrastructureProviderHandler {
	return &InfrastructureProviderHandler{
		service: service,
		logger:  logger,
	}
}

// ============================================================================
// Request/Response Types
// ============================================================================

// InfrastructureProviderResponse represents a provider in API responses
type InfrastructureProviderResponse struct {
	ID                           string                 `json:"id"`
	TenantID                     string                 `json:"tenant_id"`
	IsGlobal                     bool                   `json:"is_global"`
	ProviderType                 string                 `json:"provider_type"`
	Name                         string                 `json:"name"`
	DisplayName                  string                 `json:"display_name"`
	Config                       map[string]interface{} `json:"config"`
	Status                       string                 `json:"status"`
	Capabilities                 []string               `json:"capabilities"`
	CreatedBy                    string                 `json:"created_by"`
	CreatedAt                    string                 `json:"created_at"`
	UpdatedAt                    string                 `json:"updated_at"`
	LastHealthCheck              *string                `json:"last_health_check,omitempty"`
	HealthStatus                 *string                `json:"health_status,omitempty"`
	ReadinessStatus              *string                `json:"readiness_status,omitempty"`
	ReadinessLastChecked         *string                `json:"readiness_last_checked,omitempty"`
	ReadinessMissingPrereqs      []string               `json:"readiness_missing_prereqs,omitempty"`
	BootstrapMode                string                 `json:"bootstrap_mode"`
	CredentialScope              string                 `json:"credential_scope"`
	TargetNamespace              *string                `json:"target_namespace,omitempty"`
	IsSchedulable                bool                   `json:"is_schedulable"`
	SchedulableReason            *string                `json:"schedulable_reason,omitempty"`
	BlockedBy                    []string               `json:"blocked_by,omitempty"`
	LatestPrepareRunID           *string                `json:"latest_prepare_run_id,omitempty"`
	LatestPrepareStatus          *string                `json:"latest_prepare_status,omitempty"`
	LatestPrepareUpdatedAt       *string                `json:"latest_prepare_updated_at,omitempty"`
	LatestPrepareError           *string                `json:"latest_prepare_error,omitempty"`
	LatestPrepareCheckCategory   *string                `json:"latest_prepare_check_category,omitempty"`
	LatestPrepareCheckSeverity   *string                `json:"latest_prepare_check_severity,omitempty"`
	LatestPrepareRemediationHint *string                `json:"latest_prepare_remediation_hint,omitempty"`
}

type TenantNamespacePrepareResponse struct {
	ID                    string                 `json:"id"`
	ProviderID            string                 `json:"provider_id"`
	TenantID              string                 `json:"tenant_id"`
	Namespace             string                 `json:"namespace"`
	RequestedBy           *string                `json:"requested_by,omitempty"`
	Status                string                 `json:"status"`
	ResultSummary         map[string]interface{} `json:"result_summary,omitempty"`
	ErrorMessage          *string                `json:"error_message,omitempty"`
	DesiredAssetVersion   *string                `json:"desired_asset_version,omitempty"`
	InstalledAssetVersion *string                `json:"installed_asset_version,omitempty"`
	AssetDriftStatus      string                 `json:"asset_drift_status"`
	StartedAt             *string                `json:"started_at,omitempty"`
	CompletedAt           *string                `json:"completed_at,omitempty"`
	CreatedAt             *string                `json:"created_at,omitempty"`
	UpdatedAt             *string                `json:"updated_at,omitempty"`
}

type ReconcileSelectedTenantNamespacesRequest struct {
	TenantIDs []string `json:"tenant_ids"`
}

// CreateProviderRequest represents a request to create a provider
type CreateProviderRequest struct {
	ProviderType    string                 `json:"provider_type" validate:"required"`
	Name            string                 `json:"name" validate:"required,min=3,max=100"`
	DisplayName     string                 `json:"display_name" validate:"required,min=3,max=255"`
	Config          map[string]interface{} `json:"config" validate:"required"`
	Capabilities    []string               `json:"capabilities,omitempty"`
	IsGlobal        bool                   `json:"is_global"`
	BootstrapMode   string                 `json:"bootstrap_mode,omitempty"`
	CredentialScope string                 `json:"credential_scope,omitempty"`
	TargetNamespace *string                `json:"target_namespace,omitempty"`
}

// UpdateProviderRequest represents a request to update a provider
type UpdateProviderRequest struct {
	DisplayName     *string                 `json:"display_name,omitempty"`
	Config          *map[string]interface{} `json:"config,omitempty"`
	Capabilities    *[]string               `json:"capabilities,omitempty"`
	Status          *string                 `json:"status,omitempty"`
	IsGlobal        *bool                   `json:"is_global,omitempty"`
	BootstrapMode   *string                 `json:"bootstrap_mode,omitempty"`
	CredentialScope *string                 `json:"credential_scope,omitempty"`
	TargetNamespace *string                 `json:"target_namespace,omitempty"`
}

// TestConnectionResponse represents the response from testing a connection
type TestConnectionResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ProviderHealthResponse represents provider health information
type ProviderHealthResponse struct {
	ProviderID string                 `json:"provider_id"`
	Status     string                 `json:"status"`
	LastCheck  string                 `json:"last_check"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Metrics    *HealthMetrics         `json:"metrics,omitempty"`
}

type ProviderReadinessCheck struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

type ProviderReadinessResponse struct {
	ProviderID      string                   `json:"provider_id"`
	Status          string                   `json:"status"`
	CheckedAt       string                   `json:"checked_at"`
	TenantNamespace string                   `json:"tenant_namespace"`
	MissingPrereqs  []string                 `json:"missing_prereqs"`
	Checks          []ProviderReadinessCheck `json:"checks"`
}

type ProviderQuarantineDispatchReadinessResponse struct {
	ProviderID                 string                   `json:"provider_id"`
	ProviderName               string                   `json:"provider_name"`
	ProviderType               string                   `json:"provider_type"`
	ProviderStatus             string                   `json:"provider_status"`
	TenantNamespace            string                   `json:"tenant_namespace"`
	Status                     string                   `json:"status"`
	DispatchReady              bool                     `json:"dispatch_ready"`
	TektonEnabled              bool                     `json:"tekton_enabled"`
	QuarantineDispatchEnabled  bool                     `json:"quarantine_dispatch_enabled"`
	CheckedAt                  string                   `json:"checked_at"`
	MissingPrereqs             []string                 `json:"missing_prereqs"`
	Checks                     []ProviderReadinessCheck `json:"checks"`
	ReadinessStatus            *string                  `json:"readiness_status,omitempty"`
	ReadinessMissingPrereqs    []string                 `json:"readiness_missing_prereqs,omitempty"`
	IsSchedulable              bool                     `json:"is_schedulable"`
	SchedulableReason          *string                  `json:"schedulable_reason,omitempty"`
}

type ProviderPermissionResponse struct {
	ID         string  `json:"id"`
	ProviderID string  `json:"provider_id"`
	TenantID   string  `json:"tenant_id"`
	Permission string  `json:"permission"`
	GrantedBy  string  `json:"granted_by"`
	GrantedAt  string  `json:"granted_at"`
	ExpiresAt  *string `json:"expires_at,omitempty"`
}

type GrantPermissionRequest struct {
	TenantID   string `json:"tenant_id"`
	Permission string `json:"permission"`
}

type RevokePermissionRequest struct {
	TenantID   string `json:"tenant_id"`
	Permission string `json:"permission"`
}

type TektonInstallerActionRequest struct {
	InstallMode    string `json:"install_mode,omitempty"`
	AssetVersion   string `json:"asset_version,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type TektonInstallerRetryRequest struct {
	JobID string `json:"job_id"`
}

type TektonInstallerJobResponse struct {
	ID           string  `json:"id"`
	ProviderID   string  `json:"provider_id"`
	TenantID     string  `json:"tenant_id"`
	RequestedBy  string  `json:"requested_by"`
	InstallMode  string  `json:"install_mode"`
	AssetVersion string  `json:"asset_version"`
	Status       string  `json:"status"`
	ErrorMessage *string `json:"error_message,omitempty"`
	StartedAt    *string `json:"started_at,omitempty"`
	CompletedAt  *string `json:"completed_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type TektonInstallerJobEventResponse struct {
	ID        string                 `json:"id"`
	JobID     string                 `json:"job_id"`
	EventType string                 `json:"event_type"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	CreatedBy *string                `json:"created_by,omitempty"`
	CreatedAt string                 `json:"created_at"`
}

type TektonProviderStatusResponse struct {
	ProviderID              string                            `json:"provider_id"`
	ReadinessStatus         *string                           `json:"readiness_status,omitempty"`
	ReadinessLastChecked    *string                           `json:"readiness_last_checked,omitempty"`
	ReadinessMissingPrereqs []string                          `json:"readiness_missing_prereqs,omitempty"`
	RequiredTasks           []string                          `json:"required_tasks,omitempty"`
	RequiredPipelines       []string                          `json:"required_pipelines,omitempty"`
	ActiveJob               *TektonInstallerJobResponse       `json:"active_job,omitempty"`
	RecentJobs              []TektonInstallerJobResponse      `json:"recent_jobs"`
	ActiveJobEvents         []TektonInstallerJobEventResponse `json:"active_job_events,omitempty"`
}

type ProviderPrepareActionRequest struct {
	RequestedActions map[string]interface{} `json:"requested_actions,omitempty"`
}

type ProviderPrepareRunResponse struct {
	ID               string                 `json:"id"`
	ProviderID       string                 `json:"provider_id"`
	TenantID         string                 `json:"tenant_id"`
	RequestedBy      string                 `json:"requested_by"`
	Status           string                 `json:"status"`
	RequestedActions map[string]interface{} `json:"requested_actions,omitempty"`
	ResultSummary    map[string]interface{} `json:"result_summary,omitempty"`
	ErrorMessage     *string                `json:"error_message,omitempty"`
	StartedAt        *string                `json:"started_at,omitempty"`
	CompletedAt      *string                `json:"completed_at,omitempty"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
}

type ProviderPrepareRunCheckResponse struct {
	ID        string                 `json:"id"`
	RunID     string                 `json:"run_id"`
	CheckKey  string                 `json:"check_key"`
	Category  string                 `json:"category"`
	Severity  string                 `json:"severity"`
	OK        bool                   `json:"ok"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	CreatedAt string                 `json:"created_at"`
}

type ProviderPrepareStatusResponse struct {
	ProviderID string                            `json:"provider_id"`
	ActiveRun  *ProviderPrepareRunResponse       `json:"active_run,omitempty"`
	Checks     []ProviderPrepareRunCheckResponse `json:"checks,omitempty"`
}

type ProviderPrepareSummaryResponse struct {
	ProviderID            string  `json:"provider_id"`
	RunID                 *string `json:"run_id,omitempty"`
	Status                *string `json:"status,omitempty"`
	UpdatedAt             *string `json:"updated_at,omitempty"`
	Error                 *string `json:"error_message,omitempty"`
	LatestCheckCategory   *string `json:"latest_prepare_check_category,omitempty"`
	LatestCheckSeverity   *string `json:"latest_prepare_check_severity,omitempty"`
	LatestRemediationHint *string `json:"latest_prepare_remediation_hint,omitempty"`
}

type ProviderPrepareSummaryBatchMetricsResponse struct {
	BatchCount        int64   `json:"batch_count"`
	BatchTotalMs      int64   `json:"batch_total_ms"`
	BatchMinMs        int64   `json:"batch_min_ms"`
	BatchMaxMs        int64   `json:"batch_max_ms"`
	BatchAvgMs        float64 `json:"batch_avg_ms"`
	BatchErrors       int64   `json:"batch_errors"`
	ProvidersTotal    int64   `json:"providers_total"`
	RepositoryBatches int64   `json:"repository_batches"`
	FallbackBatches   int64   `json:"fallback_batches"`
}

// HealthMetrics represents health metrics
type HealthMetrics struct {
	TotalNodes            *int     `json:"total_nodes,omitempty"`
	HealthyNodes          *int     `json:"healthy_nodes,omitempty"`
	TotalCPUCapacity      *float64 `json:"total_cpu_capacity,omitempty"`
	UsedCPUCapacity       *float64 `json:"used_cpu_cores,omitempty"`
	TotalMemoryCapacityGB *float64 `json:"total_memory_capacity_gb,omitempty"`
	UsedMemoryGB          *float64 `json:"used_memory_gb,omitempty"`
}

// PaginatedProvidersResponse represents a paginated list of providers
type PaginatedProvidersResponse struct {
	Data       []InfrastructureProviderResponse `json:"data"`
	Pagination struct {
		Page       int `json:"page"`
		Limit      int `json:"limit"`
		Total      int `json:"total"`
		TotalPages int `json:"total_pages"`
	} `json:"pagination"`
}

// ============================================================================
// Helper Functions
// ============================================================================

// toResponse converts a domain provider to API response
func (h *InfrastructureProviderHandler) toResponse(p *infrastructure.Provider) InfrastructureProviderResponse {
	response := InfrastructureProviderResponse{
		ID:                p.ID.String(),
		TenantID:          p.TenantID.String(),
		IsGlobal:          p.IsGlobal,
		ProviderType:      string(p.ProviderType),
		Name:              p.Name,
		DisplayName:       p.DisplayName,
		Config:            p.Config,
		Status:            string(p.Status),
		Capabilities:      p.Capabilities,
		CreatedBy:         p.CreatedBy.String(),
		CreatedAt:         p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         p.UpdatedAt.Format(time.RFC3339),
		BootstrapMode:     p.BootstrapMode,
		CredentialScope:   p.CredentialScope,
		TargetNamespace:   p.TargetNamespace,
		IsSchedulable:     p.IsSchedulable,
		SchedulableReason: p.SchedulableReason,
		BlockedBy:         p.BlockedBy,
	}

	if p.LastHealthCheck != nil {
		lastCheck := p.LastHealthCheck.Format(time.RFC3339)
		response.LastHealthCheck = &lastCheck
	}
	if p.HealthStatus != nil {
		response.HealthStatus = p.HealthStatus
	}
	if p.ReadinessStatus != nil {
		response.ReadinessStatus = p.ReadinessStatus
	}
	if p.ReadinessLastChecked != nil {
		readinessLastChecked := p.ReadinessLastChecked.Format(time.RFC3339)
		response.ReadinessLastChecked = &readinessLastChecked
	}
	if len(p.ReadinessMissingPrereqs) > 0 {
		response.ReadinessMissingPrereqs = p.ReadinessMissingPrereqs
	}

	return response
}

func (h *InfrastructureProviderHandler) toTektonInstallerJobResponse(job *infrastructure.TektonInstallerJob) *TektonInstallerJobResponse {
	if job == nil {
		return nil
	}

	response := &TektonInstallerJobResponse{
		ID:           job.ID.String(),
		ProviderID:   job.ProviderID.String(),
		TenantID:     job.TenantID.String(),
		RequestedBy:  job.RequestedBy.String(),
		InstallMode:  string(job.InstallMode),
		AssetVersion: job.AssetVersion,
		Status:       string(job.Status),
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    job.UpdatedAt.Format(time.RFC3339),
	}
	if job.StartedAt != nil {
		startedAt := job.StartedAt.Format(time.RFC3339)
		response.StartedAt = &startedAt
	}
	if job.CompletedAt != nil {
		completedAt := job.CompletedAt.Format(time.RFC3339)
		response.CompletedAt = &completedAt
	}
	return response
}

func (h *InfrastructureProviderHandler) toTektonInstallerJobEventResponse(event *infrastructure.TektonInstallerJobEvent) *TektonInstallerJobEventResponse {
	if event == nil {
		return nil
	}
	response := &TektonInstallerJobEventResponse{
		ID:        event.ID.String(),
		JobID:     event.JobID.String(),
		EventType: event.EventType,
		Message:   event.Message,
		Details:   event.Details,
		CreatedAt: event.CreatedAt.Format(time.RFC3339),
	}
	if event.CreatedBy != nil {
		createdBy := event.CreatedBy.String()
		response.CreatedBy = &createdBy
	}
	return response
}

func (h *InfrastructureProviderHandler) toProviderPrepareRunResponse(run *infrastructure.ProviderPrepareRun) *ProviderPrepareRunResponse {
	if run == nil {
		return nil
	}
	response := &ProviderPrepareRunResponse{
		ID:               run.ID.String(),
		ProviderID:       run.ProviderID.String(),
		TenantID:         run.TenantID.String(),
		RequestedBy:      run.RequestedBy.String(),
		Status:           string(run.Status),
		RequestedActions: run.RequestedActions,
		ResultSummary:    run.ResultSummary,
		ErrorMessage:     run.ErrorMessage,
		CreatedAt:        run.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        run.UpdatedAt.Format(time.RFC3339),
	}
	if run.StartedAt != nil {
		startedAt := run.StartedAt.Format(time.RFC3339)
		response.StartedAt = &startedAt
	}
	if run.CompletedAt != nil {
		completedAt := run.CompletedAt.Format(time.RFC3339)
		response.CompletedAt = &completedAt
	}
	return response
}

func (h *InfrastructureProviderHandler) toProviderPrepareRunCheckResponse(check *infrastructure.ProviderPrepareRunCheck) *ProviderPrepareRunCheckResponse {
	if check == nil {
		return nil
	}
	return &ProviderPrepareRunCheckResponse{
		ID:        check.ID.String(),
		RunID:     check.RunID.String(),
		CheckKey:  check.CheckKey,
		Category:  check.Category,
		Severity:  check.Severity,
		OK:        check.OK,
		Message:   check.Message,
		Details:   check.Details,
		CreatedAt: check.CreatedAt.Format(time.RFC3339),
	}
}

func (h *InfrastructureProviderHandler) toTenantNamespacePrepareResponse(prepare *infrastructure.ProviderTenantNamespacePrepare) *TenantNamespacePrepareResponse {
	if prepare == nil {
		return nil
	}
	assetDriftStatus := prepare.AssetDriftStatus
	if assetDriftStatus == "" {
		assetDriftStatus = infrastructure.TenantAssetDriftStatusUnknown
	}
	resp := &TenantNamespacePrepareResponse{
		ID:                    prepare.ID.String(),
		ProviderID:            prepare.ProviderID.String(),
		TenantID:              prepare.TenantID.String(),
		Namespace:             prepare.Namespace,
		Status:                string(prepare.Status),
		ResultSummary:         prepare.ResultSummary,
		ErrorMessage:          prepare.ErrorMessage,
		DesiredAssetVersion:   prepare.DesiredAssetVersion,
		InstalledAssetVersion: prepare.InstalledAssetVersion,
		AssetDriftStatus:      string(assetDriftStatus),
	}
	createdAt := prepare.CreatedAt.Format(time.RFC3339)
	updatedAt := prepare.UpdatedAt.Format(time.RFC3339)
	resp.CreatedAt = &createdAt
	resp.UpdatedAt = &updatedAt
	if prepare.RequestedBy != nil {
		id := prepare.RequestedBy.String()
		resp.RequestedBy = &id
	}
	if prepare.StartedAt != nil {
		startedAt := prepare.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &startedAt
	}
	if prepare.CompletedAt != nil {
		completedAt := prepare.CompletedAt.Format(time.RFC3339)
		resp.CompletedAt = &completedAt
	}
	return resp
}

func (h *InfrastructureProviderHandler) resolveScopedProviderAccess(w http.ResponseWriter, r *http.Request, allowAllTenants bool) (*scopedProviderAccess, bool) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}

	scopeTenantID, allTenants, status, message := resolveTenantScopeFromRequest(r, authCtx, allowAllTenants)
	if status != 0 {
		http.Error(w, message, status)
		return nil, false
	}

	providerIDStr := chi.URLParam(r, "id")
	providerID, err := uuid.Parse(providerIDStr)
	if err != nil {
		http.Error(w, "Invalid provider ID", http.StatusBadRequest)
		return nil, false
	}

	provider, err := h.service.GetProvider(r.Context(), providerID)
	if err != nil {
		if err == infrastructure.ErrProviderNotFound {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return nil, false
		}
		h.logger.Error("Failed to get provider", zap.Error(err), zap.String("provider_id", providerID.String()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return nil, false
	}

	if !allTenants && provider.TenantID != scopeTenantID {
		http.Error(w, "Access denied to this tenant", http.StatusForbidden)
		return nil, false
	}

	return &scopedProviderAccess{
		authCtx:     authCtx,
		scopeTenant: scopeTenantID,
		allTenants:  allTenants,
		providerID:  providerID,
		provider:    provider,
	}, true
}

// ============================================================================
// HTTP Handlers
// ============================================================================

// ListProviders lists infrastructure providers for the current tenant
func (h *InfrastructureProviderHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in ListProviders")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	opts := &infrastructure.ListProvidersOptions{}
	tenantID := authCtx.TenantID
	allTenants := isAllTenantsScopeRequested(r, authCtx)
	includePrepareSummary := true

	if providerType := r.URL.Query().Get("provider_type"); providerType != "" {
		pType := infrastructure.ProviderType(providerType)
		opts.ProviderType = &pType
	}

	if status := r.URL.Query().Get("status"); status != "" {
		pStatus := infrastructure.ProviderStatus(status)
		opts.Status = &pStatus
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			opts.Page = page
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			opts.Limit = limit
		}
	}
	if includeSummaryRaw := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("include_prepare_summary"))); includeSummaryRaw != "" {
		switch includeSummaryRaw {
		case "false", "0", "no":
			includePrepareSummary = false
		case "true", "1", "yes":
			includePrepareSummary = true
		default:
			http.Error(w, "Invalid include_prepare_summary query parameter", http.StatusBadRequest)
			return
		}
	}

	// System admins may explicitly target a tenant via query param.
	if authCtx.IsSystemAdmin {
		if tenantIDRaw := r.URL.Query().Get("tenant_id"); tenantIDRaw != "" {
			parsedTenantID, parseErr := uuid.Parse(tenantIDRaw)
			if parseErr != nil || parsedTenantID == uuid.Nil {
				http.Error(w, "Invalid tenant_id query parameter", http.StatusBadRequest)
				return
			}
			tenantID = parsedTenantID
			allTenants = false
		}
	}

	var result *infrastructure.ListProvidersResult
	var err error
	if allTenants {
		result, err = h.service.ListProvidersAll(r.Context(), opts)
	} else {
		result, err = h.service.ListProviders(r.Context(), tenantID, opts)
	}
	if err != nil {
		h.logger.Error("Failed to list providers", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response
	providers := make([]InfrastructureProviderResponse, len(result.Providers))
	latestPrepareSummaries := map[uuid.UUID]*infrastructure.ProviderPrepareLatestSummary{}
	if includePrepareSummary {
		if len(result.Providers) > maxProvidersWithPrepareSummary {
			includePrepareSummary = false
			h.logger.Warn("Skipping prepare summary enrichment due to provider list size cap",
				zap.Int("provider_count", len(result.Providers)),
				zap.Int("max_with_summary", maxProvidersWithPrepareSummary),
			)
		} else {
			providerIDs := make([]uuid.UUID, 0, len(result.Providers))
			for _, p := range result.Providers {
				providerIDs = append(providerIDs, p.ID)
			}
			summaries, summaryErr := h.service.ListLatestProviderPrepareSummaries(r.Context(), providerIDs)
			if summaryErr != nil {
				if errors.Is(summaryErr, infrastructure.ErrProviderPrepareNotConfigured) {
					latestPrepareSummaries = map[uuid.UUID]*infrastructure.ProviderPrepareLatestSummary{}
				} else {
					h.logger.Warn("Failed to load batched provider prepare summaries for list", zap.Error(summaryErr))
				}
			} else {
				latestPrepareSummaries = summaries
			}
		}
	}
	for i, p := range result.Providers {
		response := h.toResponse(&p)
		if includePrepareSummary {
			if summary, ok := latestPrepareSummaries[p.ID]; ok && summary != nil {
				if summary.RunID != nil {
					runID := summary.RunID.String()
					response.LatestPrepareRunID = &runID
				}
				if summary.Status != nil {
					status := string(*summary.Status)
					response.LatestPrepareStatus = &status
				}
				if summary.UpdatedAt != nil {
					updatedAt := summary.UpdatedAt.Format(time.RFC3339)
					response.LatestPrepareUpdatedAt = &updatedAt
				}
				response.LatestPrepareError = summary.ErrorMessage
				response.LatestPrepareCheckCategory = summary.CheckCategory
				response.LatestPrepareCheckSeverity = summary.CheckSeverity
				response.LatestPrepareRemediationHint = summary.RemediationHint
			}
		}
		providers[i] = response
	}

	response := PaginatedProvidersResponse{
		Data: providers,
	}
	response.Pagination.Page = result.Page
	response.Pagination.Limit = result.Limit
	response.Pagination.Total = result.Total
	response.Pagination.TotalPages = result.TotalPages

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetProvider gets a specific infrastructure provider
func (h *InfrastructureProviderHandler) GetProvider(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	response := h.toResponse(access.provider)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"provider": response,
	})
}

// CreateProvider creates a new infrastructure provider
func (h *InfrastructureProviderHandler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in CreateProvider")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	scopeTenantID, _, status, message := resolveTenantScopeFromRequest(r, authCtx, false)
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	var req CreateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert to domain request
	domainReq := &infrastructure.CreateProviderRequest{
		ProviderType:    infrastructure.ProviderType(req.ProviderType),
		Name:            req.Name,
		DisplayName:     req.DisplayName,
		Config:          req.Config,
		Capabilities:    req.Capabilities,
		IsGlobal:        req.IsGlobal,
		BootstrapMode:   req.BootstrapMode,
		CredentialScope: req.CredentialScope,
		TargetNamespace: req.TargetNamespace,
	}

	provider, err := h.service.CreateProvider(r.Context(), scopeTenantID, authCtx.UserID, domainReq)
	if err != nil {
		if err == infrastructure.ErrProviderExists {
			http.Error(w, "Provider with this name already exists", http.StatusConflict)
			return
		}
		if err == infrastructure.ErrInvalidProviderType {
			http.Error(w, "Invalid provider type", http.StatusBadRequest)
			return
		}
		if errors.Is(err, infrastructure.ErrInvalidTargetNamespace) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err == infrastructure.ErrPermissionDenied {
			http.Error(w, "Permission denied", http.StatusForbidden)
			return
		}
		h.logger.Error("Failed to create provider", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := h.toResponse(provider)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"provider": response,
	})
}

// UpdateProvider updates an existing infrastructure provider
func (h *InfrastructureProviderHandler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	var req UpdateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert to domain request
	domainReq := &infrastructure.UpdateProviderRequest{
		DisplayName:     req.DisplayName,
		Config:          req.Config,
		Capabilities:    req.Capabilities,
		BootstrapMode:   req.BootstrapMode,
		CredentialScope: req.CredentialScope,
		TargetNamespace: req.TargetNamespace,
	}
	if req.Status != nil {
		status := infrastructure.ProviderStatus(*req.Status)
		domainReq.Status = &status
	}
	if req.IsGlobal != nil {
		domainReq.IsGlobal = req.IsGlobal
	}

	provider, err := h.service.UpdateProvider(r.Context(), access.providerID, domainReq)
	if err != nil {
		if err == infrastructure.ErrProviderNotFound {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		if err == infrastructure.ErrInvalidProviderStatus {
			http.Error(w, "Invalid provider status", http.StatusBadRequest)
			return
		}
		if err == infrastructure.ErrPermissionDenied {
			http.Error(w, "Permission denied", http.StatusForbidden)
			return
		}
		h.logger.Error("Failed to update provider", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := h.toResponse(provider)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"provider": response,
	})
}

// DeleteProvider deletes an infrastructure provider
func (h *InfrastructureProviderHandler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	err := h.service.DeleteProvider(r.Context(), access.providerID)
	if err != nil {
		if err == infrastructure.ErrProviderNotFound {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to delete provider", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TestProviderConnection tests the connection to an infrastructure provider
func (h *InfrastructureProviderHandler) TestProviderConnection(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	result, err := h.service.TestProviderConnection(r.Context(), access.providerID)
	if err != nil {
		if err == infrastructure.ErrProviderNotFound {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to test provider connection", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := TestConnectionResponse{
		Success: result.Success,
		Message: result.Message,
		Details: result.Details,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetProviderHealth gets the health status of an infrastructure provider
func (h *InfrastructureProviderHandler) GetProviderHealth(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	health, err := h.service.GetProviderHealth(r.Context(), access.providerID)
	if err != nil {
		if err == infrastructure.ErrProviderNotFound {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to get provider health", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := ProviderHealthResponse{
		ProviderID: health.ProviderID.String(),
		Status:     health.Status,
		LastCheck:  health.LastCheck.Format(time.RFC3339),
		Details:    health.Details,
		Metrics:    (*HealthMetrics)(health.Metrics),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetProviderReadiness validates Tekton build prerequisites on a provider.
func (h *InfrastructureProviderHandler) GetProviderReadiness(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	provider := access.provider
	tenantID := access.scopeTenant

	checks := make([]ProviderReadinessCheck, 0, 8)
	missing := make([]string, 0, 8)
	addCheck := func(name string, ok bool, message string) {
		checks = append(checks, ProviderReadinessCheck{Name: name, OK: ok, Message: message})
		if !ok && message != "" {
			missing = append(missing, message)
		}
	}

	tektonEnabled, hasTektonEnabled := provider.Config["tekton_enabled"].(bool)
	if !hasTektonEnabled {
		addCheck("provider_config.tekton_enabled", false, "provider config missing tekton_enabled=true")
	} else if !tektonEnabled {
		addCheck("provider_config.tekton_enabled", false, "provider config has tekton_enabled=false")
	} else {
		addCheck("provider_config.tekton_enabled", true, "")
	}

	restConfig, err := connectors.BuildRESTConfigFromProviderConfig(provider.Config)
	if err != nil {
		addCheck("provider_kubeconfig", false, "invalid provider kubeconfig/config")
		h.respondReadiness(r.Context(), w, access.providerID, "", checks, missing)
		return
	}
	addCheck("provider_kubeconfig", true, "")

	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		addCheck("kubernetes_client", false, "failed to create kubernetes client")
		h.respondReadiness(r.Context(), w, access.providerID, "", checks, missing)
		return
	}
	addCheck("kubernetes_client", true, "")

	namespace := resolveProviderNamespace(provider, tenantID)

	k8sStart := time.Now()
	if _, err := k8sClient.Discovery().ServerVersion(); err != nil {
		addCheck("kubernetes_api", false, "kubernetes API unreachable")
	} else {
		addCheck("kubernetes_api", true, "")
		addCheck("kubernetes_api_latency", true, fmt.Sprintf("kubernetes API latency %dms", time.Since(k8sStart).Milliseconds()))
	}

	tektonClient, err := tektonclient.NewForConfig(restConfig)
	if err != nil {
		addCheck("tekton_client", false, "failed to create tekton client")
		h.respondReadiness(r.Context(), w, access.providerID, namespace, checks, missing)
		return
	}
	addCheck("tekton_client", true, "")
	version, versionErr := tektoncompat.DetectAPIVersion(r.Context(), tektonClient, namespace)
	if versionErr != nil {
		addCheck("tekton_api_version", false, fmt.Sprintf("failed to detect tekton api version: %v", versionErr))
		h.respondReadiness(r.Context(), w, access.providerID, namespace, checks, missing)
		return
	}
	compat := tektoncompat.New(tektonClient, version)

	if namespace != "" {
		tektonStart := time.Now()
		// Avoid LIST probes here; runtime identities may not have list permissions. Use a GET probe instead.
		if err := compat.GetTask(r.Context(), namespace, "git-clone"); err != nil && !apierrors.IsNotFound(err) {
			addCheck("tekton_api", false, fmt.Sprintf("tekton API probe failed for namespace %s: %v", namespace, err))
		} else {
			addCheck("tekton_api", true, "")
			addCheck("tekton_api_latency", true, fmt.Sprintf("tekton API latency %dms", time.Since(tektonStart).Milliseconds()))
		}
	}

	nodes, nodeErr := k8sClient.CoreV1().Nodes().List(r.Context(), metav1.ListOptions{})
	if nodeErr != nil {
		addCheck("cluster_capacity", false, fmt.Sprintf("failed to inspect node capacity: %v", nodeErr))
	} else {
		total := len(nodes.Items)
		ready := 0
		for _, node := range nodes.Items {
			for _, cond := range node.Status.Conditions {
				if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
					ready++
					break
				}
			}
		}
		if total == 0 {
			addCheck("cluster_capacity", false, "cluster has zero nodes")
		} else if ready == 0 {
			addCheck("cluster_capacity", false, fmt.Sprintf("no ready nodes (%d total)", total))
		} else {
			addCheck("cluster_capacity", true, fmt.Sprintf("ready nodes: %d/%d", ready, total))
		}
	}

	requiredTasks := requiredTektonTaskNamesForReadiness()
	for _, taskName := range requiredTasks {
		if namespace == "" {
			addCheck("task."+taskName, false, "tenant namespace unavailable to check tasks")
			continue
		}
		if err := compat.GetTask(r.Context(), namespace, taskName); err != nil {
			switch {
			case apierrors.IsNotFound(err):
				addCheck("task."+taskName, false, fmt.Sprintf("missing tekton task: %s in namespace %s", taskName, namespace))
			case apierrors.IsForbidden(err), apierrors.IsUnauthorized(err):
				addCheck("task."+taskName, false, fmt.Sprintf("tekton task access denied: %s in namespace %s (%v)", taskName, namespace, err))
			default:
				addCheck("task."+taskName, false, fmt.Sprintf("failed to check tekton task: %s in namespace %s (%v)", taskName, namespace, err))
			}
		} else {
			addCheck("task."+taskName, true, "")
		}
	}

	if namespace == "" {
		addCheck("secret.docker-config", false, "tenant namespace unavailable to check docker-config secret")
	} else {
		if _, err := k8sClient.CoreV1().Secrets(namespace).Get(r.Context(), "docker-config", metav1.GetOptions{}); err != nil {
			addCheck("secret.docker-config", false, fmt.Sprintf("missing required secret docker-config in namespace %s", namespace))
		} else {
			addCheck("secret.docker-config", true, "")
		}
	}

	if usePipelineRefMode(provider.Config) {
		for _, pipelineName := range requiredPipelineNames(provider.Config) {
			if namespace == "" {
				addCheck("pipeline."+pipelineName, false, "tenant namespace unavailable to check pipelines")
				continue
			}
			if err := compat.GetPipeline(r.Context(), namespace, pipelineName); err != nil {
				switch {
				case apierrors.IsNotFound(err):
					addCheck("pipeline."+pipelineName, false, fmt.Sprintf("missing tekton pipeline: %s in namespace %s", pipelineName, namespace))
				case apierrors.IsForbidden(err), apierrors.IsUnauthorized(err):
					addCheck("pipeline."+pipelineName, false, fmt.Sprintf("tekton pipeline access denied: %s in namespace %s (%v)", pipelineName, namespace, err))
				default:
					addCheck("pipeline."+pipelineName, false, fmt.Sprintf("failed to check tekton pipeline: %s in namespace %s (%v)", pipelineName, namespace, err))
				}
			} else {
				addCheck("pipeline."+pipelineName, true, "")
			}
		}
	}

	if shouldProbePVC(r) {
		if namespace == "" {
			addCheck("storage.pvc_probe", false, "tenant namespace unavailable for PVC probe")
		} else {
			if err := probePVC(r.Context(), k8sClient, namespace, provider.Config); err != nil {
				addCheck("storage.pvc_probe", false, fmt.Sprintf("PVC provisioning probe failed: %v", err))
			} else {
				addCheck("storage.pvc_probe", true, "")
			}
		}
	}

	if shouldProbeRBAC(r) {
		if namespace == "" {
			addCheck("namespace_rbac_probe", false, "tenant namespace unavailable for RBAC probe")
		} else {
			if err := probeNamespaceRBAC(r.Context(), k8sClient, namespace); err != nil {
				addCheck("namespace_rbac_probe", false, fmt.Sprintf("namespace RBAC probe failed: %v", err))
			} else {
				addCheck("namespace_rbac_probe", true, "")
			}
		}
	}

	h.respondReadiness(r.Context(), w, access.providerID, namespace, checks, missing)
}

// GetProviderQuarantineDispatchReadiness returns lightweight quarantine-dispatch eligibility checks.
func (h *InfrastructureProviderHandler) GetProviderQuarantineDispatchReadiness(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	provider := access.provider
	checks := make([]ProviderReadinessCheck, 0, 10)
	missing := make([]string, 0, 10)
	addCheck := func(name string, ok bool, message string) {
		checks = append(checks, ProviderReadinessCheck{Name: name, OK: ok, Message: message})
		if !ok && strings.TrimSpace(message) != "" {
			missing = append(missing, message)
		}
	}

	if isQuarantineDispatchProviderType(provider.ProviderType) {
		addCheck("provider_type", true, "")
	} else {
		addCheck("provider_type", false, fmt.Sprintf("provider type %s is not supported for quarantine dispatch", provider.ProviderType))
	}

	if provider.Status == infrastructure.ProviderStatusOnline {
		addCheck("provider_status", true, "")
	} else {
		addCheck("provider_status", false, fmt.Sprintf("provider status is %s (must be online)", provider.Status))
	}

	tektonEnabled, hasTektonEnabled := providerConfigBool(provider.Config, "tekton_enabled")
	switch {
	case !hasTektonEnabled:
		addCheck("provider_config.tekton_enabled", false, "provider config missing tekton_enabled=true")
	case !tektonEnabled:
		addCheck("provider_config.tekton_enabled", false, "provider config has tekton_enabled=false")
	default:
		addCheck("provider_config.tekton_enabled", true, "")
	}

	quarantineDispatchEnabled, hasQuarantineDispatchEnabled := providerConfigBool(provider.Config, "quarantine_dispatch_enabled")
	switch {
	case !hasQuarantineDispatchEnabled:
		addCheck("provider_config.quarantine_dispatch_enabled", false, "provider config missing quarantine_dispatch_enabled=true")
	case !quarantineDispatchEnabled:
		addCheck("provider_config.quarantine_dispatch_enabled", false, "provider config has quarantine_dispatch_enabled=false")
	default:
		addCheck("provider_config.quarantine_dispatch_enabled", true, "")
	}

	namespace := resolveProviderNamespace(provider, access.scopeTenant)
	if strings.TrimSpace(namespace) == "" {
		addCheck("target_namespace", false, "provider target namespace could not be resolved")
	} else {
		addCheck("target_namespace", true, "")
	}

	if _, err := connectors.BuildRESTConfigFromProviderConfig(provider.Config); err != nil {
		addCheck("runtime_auth", false, fmt.Sprintf("runtime auth config invalid: %v", err))
	} else {
		addCheck("runtime_auth", true, "")
	}

	readinessStatus := strings.TrimSpace(stringOrDefault(provider.ReadinessStatus, "unknown"))
	if readinessStatus == "ready" {
		addCheck("provider_readiness_status", true, "")
	} else {
		msg := fmt.Sprintf("provider readiness status is %s", readinessStatus)
		if len(provider.ReadinessMissingPrereqs) > 0 {
			msg = fmt.Sprintf("%s: %s", msg, strings.Join(provider.ReadinessMissingPrereqs, "; "))
		}
		addCheck("provider_readiness_status", false, msg)
	}

	if provider.IsSchedulable {
		addCheck("provider_schedulable", true, "")
	} else {
		reason := strings.TrimSpace(stringOrDefault(provider.SchedulableReason, "provider is not schedulable"))
		addCheck("provider_schedulable", false, reason)
	}

	status := "ready"
	if len(missing) > 0 {
		status = "not_ready"
	}
	checkedAt := time.Now().UTC()
	response := ProviderQuarantineDispatchReadinessResponse{
		ProviderID:                provider.ID.String(),
		ProviderName:              provider.Name,
		ProviderType:              string(provider.ProviderType),
		ProviderStatus:            string(provider.Status),
		TenantNamespace:           namespace,
		Status:                    status,
		DispatchReady:             len(missing) == 0,
		TektonEnabled:             tektonEnabled,
		QuarantineDispatchEnabled: quarantineDispatchEnabled,
		CheckedAt:                 checkedAt.Format(time.RFC3339),
		MissingPrereqs:            missing,
		Checks:                    checks,
		ReadinessStatus:           provider.ReadinessStatus,
		ReadinessMissingPrereqs:   provider.ReadinessMissingPrereqs,
		IsSchedulable:             provider.IsSchedulable,
		SchedulableReason:         provider.SchedulableReason,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func resolveProviderNamespace(provider *infrastructure.Provider, tenantID uuid.UUID) string {
	if provider != nil && provider.TargetNamespace != nil {
		if ns := strings.TrimSpace(*provider.TargetNamespace); ns != "" {
			return ns
		}
	}
	if provider != nil && provider.Config != nil {
		if ns, ok := provider.Config["tekton_target_namespace"].(string); ok && strings.TrimSpace(ns) != "" {
			return strings.TrimSpace(ns)
		}
		if ns, ok := provider.Config["system_namespace"].(string); ok && strings.TrimSpace(ns) != "" {
			return strings.TrimSpace(ns)
		}
	}
	if provider != nil && provider.TenantID != uuid.Nil {
		return fmt.Sprintf("image-factory-%s", provider.TenantID.String()[:8])
	}
	if tenantID != uuid.Nil {
		return fmt.Sprintf("image-factory-%s", tenantID.String()[:8])
	}
	return ""
}

func (h *InfrastructureProviderHandler) respondReadiness(
	ctx context.Context,
	w http.ResponseWriter,
	providerID uuid.UUID,
	namespace string,
	checks []ProviderReadinessCheck,
	missing []string,
) {
	status := "ready"
	if len(missing) > 0 {
		status = "not_ready"
	}

	checkedAt := time.Now().UTC()
	if err := h.service.UpdateProviderReadiness(ctx, providerID, status, checkedAt, missing); err != nil {
		h.logger.Error("Failed to persist provider readiness result",
			zap.Error(err),
			zap.String("provider_id", providerID.String()),
			zap.String("status", status),
		)
		http.Error(w, "Failed to persist provider readiness", http.StatusInternalServerError)
		return
	}

	response := ProviderReadinessResponse{
		ProviderID:      providerID.String(),
		Status:          status,
		CheckedAt:       checkedAt.Format(time.RFC3339),
		TenantNamespace: namespace,
		MissingPrereqs:  missing,
		Checks:          checks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func usePipelineRefMode(config map[string]interface{}) bool {
	if config == nil {
		return false
	}
	if val, ok := config["tekton_use_pipeline_ref"].(bool); ok {
		return val
	}
	if val, ok := config["tekton_pipeline_ref_mode"].(bool); ok {
		return val
	}
	return false
}

func requiredPipelineNames(config map[string]interface{}) []string {
	version := "v1"
	if config != nil {
		if val, ok := config["tekton_profile_version"].(string); ok && val != "" {
			version = val
		}
	}
	return []string{
		fmt.Sprintf("image-factory-build-%s-docker", version),
		fmt.Sprintf("image-factory-build-%s-buildx", version),
		fmt.Sprintf("image-factory-build-%s-kaniko", version),
		fmt.Sprintf("image-factory-build-%s-packer", version),
	}
}

func shouldProbePVC(r *http.Request) bool {
	raw := r.URL.Query().Get("probe_pvc")
	if raw == "" {
		return false
	}
	v, err := strconv.ParseBool(raw)
	return err == nil && v
}

func providerConfigBool(config map[string]interface{}, key string) (bool, bool) {
	if config == nil {
		return false, false
	}
	raw, ok := config[key]
	if !ok || raw == nil {
		return false, false
	}
	switch v := raw.(type) {
	case bool:
		return v, true
	case string:
		normalized := strings.TrimSpace(strings.ToLower(v))
		switch normalized {
		case "true", "1", "yes", "y":
			return true, true
		case "false", "0", "no", "n":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

func isQuarantineDispatchProviderType(providerType infrastructure.ProviderType) bool {
	switch providerType {
	case infrastructure.ProviderTypeKubernetes,
		infrastructure.ProviderTypeAWSEKS,
		infrastructure.ProviderTypeGCPGKE,
		infrastructure.ProviderTypeAzureAKS,
		infrastructure.ProviderTypeOCIOKE,
		infrastructure.ProviderTypeOpenShift,
		infrastructure.ProviderTypeRancher:
		return true
	default:
		return false
	}
}

func stringOrDefault(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func shouldProbeRBAC(r *http.Request) bool {
	raw := r.URL.Query().Get("probe_rbac")
	if raw == "" {
		return false
	}
	v, err := strconv.ParseBool(raw)
	return err == nil && v
}

func probePVC(ctx context.Context, k8sClient kubernetes.Interface, namespace string, config map[string]interface{}) error {
	quantity, err := resource.ParseQuantity("1Mi")
	if err != nil {
		return err
	}
	pvcName := fmt.Sprintf("if-readiness-%d", time.Now().UnixNano())

	var storageClassName *string
	if config != nil {
		if val, ok := config["tekton_storage_class"].(string); ok && val != "" {
			storageClassName = &val
		}
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvcName,
			Labels: map[string]string{
				"app":  "image-factory",
				"kind": "readiness-probe",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: storageClassName,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: quantity,
				},
			},
		},
	}

	if _, err := k8sClient.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{}); err != nil {
		return err
	}
	_ = k8sClient.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
	return nil
}

func probeNamespaceRBAC(ctx context.Context, k8sClient kubernetes.Interface, namespace string) error {
	name := fmt.Sprintf("if-rbac-probe-%d", time.Now().UnixNano())
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app":  "image-factory",
				"kind": "rbac-probe",
			},
		},
		Data: map[string]string{"probe": "ok"},
	}
	if _, err := k8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{}); err != nil {
		return err
	}
	_ = k8sClient.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	return nil
}

func (h *InfrastructureProviderHandler) PrepareProvider(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	req := ProviderPrepareActionRequest{}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	}

	run, err := h.service.StartProviderPrepareRun(r.Context(), access.providerID, access.provider.TenantID, access.authCtx.UserID, infrastructure.StartProviderPrepareRunRequest{
		RequestedActions: req.RequestedActions,
	})
	if err != nil {
		switch {
		case errors.Is(err, infrastructure.ErrProviderNotFound):
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		case errors.Is(err, infrastructure.ErrPermissionDenied):
			http.Error(w, "Permission denied", http.StatusForbidden)
			return
		case errors.Is(err, infrastructure.ErrProviderPrepareRunInProgress):
			http.Error(w, "Another provider prepare run is already in progress for this provider", http.StatusConflict)
			return
		case errors.Is(err, infrastructure.ErrProviderPrepareNotConfigured):
			http.Error(w, "Provider prepare is not configured on this server", http.StatusNotImplemented)
			return
		}
		h.logger.Error("Failed to start provider prepare run", zap.Error(err), zap.String("provider_id", access.providerID.String()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"run": h.toProviderPrepareRunResponse(run),
	})
}

func (h *InfrastructureProviderHandler) ProvisionTenantNamespace(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}
	if access.authCtx == nil || !access.authCtx.IsSystemAdmin {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	tenantIDStr := chi.URLParam(r, "tenant_id")
	tenantID, err := uuid.Parse(strings.TrimSpace(tenantIDStr))
	if err != nil || tenantID == uuid.Nil {
		http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
		return
	}

	prepare, err := h.service.EnsureTenantNamespaceReady(r.Context(), access.providerID, tenantID, &access.authCtx.UserID)
	if err != nil {
		if errors.Is(err, infrastructure.ErrProviderNotFound) {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, infrastructure.ErrPermissionDenied) {
			http.Error(w, "Permission denied", http.StatusForbidden)
			return
		}
		if errors.Is(err, infrastructure.ErrProviderPrepareNotConfigured) {
			http.Error(w, "Tenant namespace provisioning is not configured on this server", http.StatusNotImplemented)
			return
		}
		h.logger.Error("Failed to prepare tenant namespace",
			zap.Error(err),
			zap.String("provider_id", access.providerID.String()),
			zap.String("tenant_id", tenantID.String()),
		)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"prepare": h.toTenantNamespacePrepareResponse(prepare),
	})
}

// PrepareTenantNamespace is kept as a backward-compatible alias.
func (h *InfrastructureProviderHandler) PrepareTenantNamespace(w http.ResponseWriter, r *http.Request) {
	h.ProvisionTenantNamespace(w, r)
}

func (h *InfrastructureProviderHandler) DeprovisionTenantNamespace(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}
	if access.authCtx == nil || !access.authCtx.IsSystemAdmin {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	tenantIDStr := chi.URLParam(r, "tenant_id")
	tenantID, err := uuid.Parse(strings.TrimSpace(tenantIDStr))
	if err != nil || tenantID == uuid.Nil {
		http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
		return
	}

	prepare, err := h.service.DeprovisionTenantNamespace(r.Context(), access.providerID, tenantID, &access.authCtx.UserID)
	if err != nil {
		if errors.Is(err, infrastructure.ErrProviderNotFound) {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, infrastructure.ErrPermissionDenied) {
			http.Error(w, "Permission denied", http.StatusForbidden)
			return
		}
		if errors.Is(err, infrastructure.ErrProviderPrepareNotConfigured) {
			http.Error(w, "Tenant namespace provisioning is not configured on this server", http.StatusNotImplemented)
			return
		}
		h.logger.Error("Failed to deprovision tenant namespace",
			zap.Error(err),
			zap.String("provider_id", access.providerID.String()),
			zap.String("tenant_id", tenantID.String()),
		)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"prepare": h.toTenantNamespacePrepareResponse(prepare),
	})
}

func (h *InfrastructureProviderHandler) ReconcileStaleTenantNamespaces(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}
	if access.authCtx == nil || !access.authCtx.IsSystemAdmin {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	summary, err := h.service.ReconcileStaleTenantNamespaces(r.Context(), access.providerID, &access.authCtx.UserID)
	if err != nil {
		if errors.Is(err, infrastructure.ErrProviderNotFound) {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, infrastructure.ErrPermissionDenied) {
			http.Error(w, "Permission denied", http.StatusForbidden)
			return
		}
		if errors.Is(err, infrastructure.ErrProviderPrepareNotConfigured) {
			http.Error(w, "Tenant namespace provisioning is not configured on this server", http.StatusNotImplemented)
			return
		}
		h.logger.Warn("Stale tenant namespace reconcile completed with failures",
			zap.Error(err),
			zap.String("provider_id", access.providerID.String()),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"summary": summary,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"summary": summary,
	})
}

func (h *InfrastructureProviderHandler) ReconcileSelectedTenantNamespaces(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}
	if access.authCtx == nil || !access.authCtx.IsSystemAdmin {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	var req ReconcileSelectedTenantNamespacesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.TenantIDs) == 0 {
		http.Error(w, "tenant_ids is required", http.StatusBadRequest)
		return
	}

	tenantIDs := make([]uuid.UUID, 0, len(req.TenantIDs))
	seen := make(map[uuid.UUID]struct{})
	for _, raw := range req.TenantIDs {
		parsed, parseErr := uuid.Parse(strings.TrimSpace(raw))
		if parseErr != nil || parsed == uuid.Nil {
			http.Error(w, "Invalid tenant_ids entry", http.StatusBadRequest)
			return
		}
		if _, exists := seen[parsed]; exists {
			continue
		}
		seen[parsed] = struct{}{}
		tenantIDs = append(tenantIDs, parsed)
	}

	summary, err := h.service.ReconcileSelectedTenantNamespaces(r.Context(), access.providerID, tenantIDs, &access.authCtx.UserID)
	if err != nil {
		if errors.Is(err, infrastructure.ErrProviderNotFound) {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, infrastructure.ErrPermissionDenied) {
			http.Error(w, "Permission denied", http.StatusForbidden)
			return
		}
		if errors.Is(err, infrastructure.ErrProviderPrepareNotConfigured) {
			http.Error(w, "Tenant namespace provisioning is not configured on this server", http.StatusNotImplemented)
			return
		}
		h.logger.Warn("Selected tenant namespace reconcile completed with failures",
			zap.Error(err),
			zap.String("provider_id", access.providerID.String()),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"summary": summary,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"summary": summary,
	})
}

func (h *InfrastructureProviderHandler) GetTenantNamespacePrepareStatus(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}
	if access.authCtx == nil || !access.authCtx.IsSystemAdmin {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	tenantIDStr := chi.URLParam(r, "tenant_id")
	tenantID, err := uuid.Parse(strings.TrimSpace(tenantIDStr))
	if err != nil || tenantID == uuid.Nil {
		http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
		return
	}

	status, err := h.service.GetTenantNamespacePrepareStatus(r.Context(), access.providerID, tenantID)
	if err != nil {
		if errors.Is(err, infrastructure.ErrProviderPrepareNotConfigured) {
			http.Error(w, "Tenant namespace provisioning is not configured on this server", http.StatusNotImplemented)
			return
		}
		h.logger.Error("Failed to get tenant namespace prepare status",
			zap.Error(err),
			zap.String("provider_id", access.providerID.String()),
			zap.String("tenant_id", tenantID.String()),
		)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"prepare": h.toTenantNamespacePrepareResponse(status),
	})
}

func (h *InfrastructureProviderHandler) StreamTenantNamespacePrepareStatus(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}
	if access.authCtx == nil || !access.authCtx.IsSystemAdmin {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	tenantIDStr := chi.URLParam(r, "tenant_id")
	tenantID, err := uuid.Parse(strings.TrimSpace(tenantIDStr))
	if err != nil || tenantID == uuid.Nil {
		http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
		return
	}

	conn, err := providerPrepareStreamUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket connection for tenant namespace prepare stream",
			zap.Error(err),
			zap.String("provider_id", access.providerID.String()),
			zap.String("tenant_id", tenantID.String()),
		)
		return
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			if _, _, readErr := conn.ReadMessage(); readErr != nil {
				return
			}
		}
	}()

	sendSnapshot := func() error {
		status, statusErr := h.service.GetTenantNamespacePrepareStatus(r.Context(), access.providerID, tenantID)
		if statusErr != nil {
			return statusErr
		}

		payload := map[string]interface{}{
			"type":        "tenant_namespace_prepare_status",
			"provider_id": access.providerID.String(),
			"tenant_id":   tenantID.String(),
			"prepare":     h.toTenantNamespacePrepareResponse(status),
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		}
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		return conn.WriteJSON(payload)
	}

	if snapErr := sendSnapshot(); snapErr != nil {
		h.logger.Warn("Failed to write initial tenant namespace prepare stream snapshot",
			zap.Error(snapErr),
			zap.String("provider_id", access.providerID.String()),
			zap.String("tenant_id", tenantID.String()),
		)
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-readDone:
			return
		case <-ticker.C:
			if snapErr := sendSnapshot(); snapErr != nil {
				h.logger.Warn("Tenant namespace prepare stream snapshot failed",
					zap.Error(snapErr),
					zap.String("provider_id", access.providerID.String()),
					zap.String("tenant_id", tenantID.String()),
				)
				return
			}
		case <-pingTicker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if pingErr := conn.WriteMessage(websocket.PingMessage, nil); pingErr != nil {
				return
			}
		}
	}
}

func (h *InfrastructureProviderHandler) GetProviderPrepareStatus(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	status, err := h.service.GetProviderPrepareStatus(r.Context(), access.providerID)
	if err != nil {
		switch {
		case errors.Is(err, infrastructure.ErrProviderNotFound):
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		case errors.Is(err, infrastructure.ErrProviderPrepareNotConfigured):
			http.Error(w, "Provider prepare is not configured on this server", http.StatusNotImplemented)
			return
		}
		h.logger.Error("Failed to get provider prepare status", zap.Error(err), zap.String("provider_id", access.providerID.String()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := ProviderPrepareStatusResponse{
		ProviderID: status.ProviderID,
		Checks:     make([]ProviderPrepareRunCheckResponse, 0, len(status.Checks)),
	}
	response.ActiveRun = h.toProviderPrepareRunResponse(status.ActiveRun)
	for _, check := range status.Checks {
		if converted := h.toProviderPrepareRunCheckResponse(check); converted != nil {
			response.Checks = append(response.Checks, *converted)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *InfrastructureProviderHandler) ListProviderPrepareRuns(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	limit := 20
	offset := 0
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 || parsed > 200 {
			http.Error(w, "Invalid limit (must be between 1 and 200)", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	if rawOffset := r.URL.Query().Get("offset"); rawOffset != "" {
		parsed, err := strconv.Atoi(rawOffset)
		if err != nil || parsed < 0 {
			http.Error(w, "Invalid offset (must be >= 0)", http.StatusBadRequest)
			return
		}
		offset = parsed
	}

	runs, err := h.service.ListProviderPrepareRuns(r.Context(), access.providerID, limit, offset)
	if err != nil {
		switch {
		case errors.Is(err, infrastructure.ErrProviderNotFound):
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		case errors.Is(err, infrastructure.ErrProviderPrepareNotConfigured):
			http.Error(w, "Provider prepare is not configured on this server", http.StatusNotImplemented)
			return
		}
		h.logger.Error("Failed to list provider prepare runs", zap.Error(err), zap.String("provider_id", access.providerID.String()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]ProviderPrepareRunResponse, 0, len(runs))
	for _, run := range runs {
		if converted := h.toProviderPrepareRunResponse(run); converted != nil {
			response = append(response, *converted)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"runs": response,
	})
}

func (h *InfrastructureProviderHandler) GetProviderPrepareRun(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	runID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "run_id")))
	if err != nil || runID == uuid.Nil {
		http.Error(w, "Invalid run_id", http.StatusBadRequest)
		return
	}

	limit := 500
	offset := 0
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, parseErr := strconv.Atoi(rawLimit)
		if parseErr != nil || parsed <= 0 || parsed > 2000 {
			http.Error(w, "Invalid limit (must be between 1 and 2000)", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	if rawOffset := r.URL.Query().Get("offset"); rawOffset != "" {
		parsed, parseErr := strconv.Atoi(rawOffset)
		if parseErr != nil || parsed < 0 {
			http.Error(w, "Invalid offset (must be >= 0)", http.StatusBadRequest)
			return
		}
		offset = parsed
	}

	run, checks, err := h.service.GetProviderPrepareRunWithChecks(r.Context(), access.providerID, runID, limit, offset)
	if err != nil {
		switch {
		case errors.Is(err, infrastructure.ErrProviderNotFound):
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		case errors.Is(err, infrastructure.ErrProviderPrepareRunNotFound):
			http.Error(w, "Provider prepare run not found", http.StatusNotFound)
			return
		case errors.Is(err, infrastructure.ErrProviderPrepareNotConfigured):
			http.Error(w, "Provider prepare is not configured on this server", http.StatusNotImplemented)
			return
		}
		h.logger.Error("Failed to get provider prepare run",
			zap.Error(err),
			zap.String("provider_id", access.providerID.String()),
			zap.String("run_id", runID.String()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := ProviderPrepareStatusResponse{
		ProviderID: access.providerID.String(),
		ActiveRun:  h.toProviderPrepareRunResponse(run),
		Checks:     make([]ProviderPrepareRunCheckResponse, 0, len(checks)),
	}
	for _, check := range checks {
		if converted := h.toProviderPrepareRunCheckResponse(check); converted != nil {
			response.Checks = append(response.Checks, *converted)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *InfrastructureProviderHandler) StreamProviderPrepareStatus(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	conn, err := providerPrepareStreamUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket connection for provider prepare stream",
			zap.Error(err),
			zap.String("provider_id", access.providerID.String()),
		)
		return
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			if _, _, readErr := conn.ReadMessage(); readErr != nil {
				return
			}
		}
	}()

	sendSnapshot := func() error {
		status, statusErr := h.service.GetProviderPrepareStatus(r.Context(), access.providerID)
		if statusErr != nil {
			return statusErr
		}
		runs, runsErr := h.service.ListProviderPrepareRuns(r.Context(), access.providerID, 10, 0)
		if runsErr != nil {
			return runsErr
		}

		statusResponse := ProviderPrepareStatusResponse{
			ProviderID: status.ProviderID,
			Checks:     make([]ProviderPrepareRunCheckResponse, 0, len(status.Checks)),
		}
		statusResponse.ActiveRun = h.toProviderPrepareRunResponse(status.ActiveRun)
		for _, check := range status.Checks {
			if converted := h.toProviderPrepareRunCheckResponse(check); converted != nil {
				statusResponse.Checks = append(statusResponse.Checks, *converted)
			}
		}

		runResponse := make([]ProviderPrepareRunResponse, 0, len(runs))
		for _, run := range runs {
			if converted := h.toProviderPrepareRunResponse(run); converted != nil {
				runResponse = append(runResponse, *converted)
			}
		}

		payload := map[string]interface{}{
			"type":        "prepare_status",
			"provider_id": access.providerID.String(),
			"status":      statusResponse,
			"runs":        runResponse,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		}
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		return conn.WriteJSON(payload)
	}

	if snapErr := sendSnapshot(); snapErr != nil {
		h.logger.Warn("Failed to write initial provider prepare stream snapshot",
			zap.Error(snapErr),
			zap.String("provider_id", access.providerID.String()),
		)
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-readDone:
			return
		case <-ticker.C:
			if snapErr := sendSnapshot(); snapErr != nil {
				h.logger.Warn("Provider prepare stream snapshot failed",
					zap.Error(snapErr),
					zap.String("provider_id", access.providerID.String()),
				)
				return
			}
		case <-pingTicker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if pingErr := conn.WriteMessage(websocket.PingMessage, nil); pingErr != nil {
				return
			}
		}
	}
}

func (h *InfrastructureProviderHandler) GetProviderPrepareSummaries(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	scopeTenantID, allTenants, status, message := resolveTenantScopeFromRequest(r, authCtx, true)
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	rawProviderIDs := strings.TrimSpace(r.URL.Query().Get("provider_ids"))
	if rawProviderIDs == "" {
		http.Error(w, "provider_ids is required", http.StatusBadRequest)
		return
	}

	rawParts := strings.Split(rawProviderIDs, ",")
	if len(rawParts) > 200 {
		http.Error(w, "provider_ids supports at most 200 ids", http.StatusBadRequest)
		return
	}

	seen := make(map[uuid.UUID]struct{}, len(rawParts))
	providerIDs := make([]uuid.UUID, 0, len(rawParts))
	for _, raw := range rawParts {
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}
		providerID, parseErr := uuid.Parse(part)
		if parseErr != nil {
			http.Error(w, "Invalid provider_ids", http.StatusBadRequest)
			return
		}
		if _, exists := seen[providerID]; exists {
			continue
		}
		seen[providerID] = struct{}{}
		providerIDs = append(providerIDs, providerID)
	}
	if len(providerIDs) == 0 {
		http.Error(w, "provider_ids is required", http.StatusBadRequest)
		return
	}
	includeBatchMetrics := false
	if includeMetricsRaw := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("include_batch_metrics"))); includeMetricsRaw != "" {
		switch includeMetricsRaw {
		case "false", "0", "no":
			includeBatchMetrics = false
		case "true", "1", "yes":
			includeBatchMetrics = true
		default:
			http.Error(w, "Invalid include_batch_metrics query parameter", http.StatusBadRequest)
			return
		}
	}

	batchedSummaries, summaryErr := h.service.ListLatestProviderPrepareSummaries(r.Context(), providerIDs)
	if summaryErr != nil {
		switch {
		case errors.Is(summaryErr, infrastructure.ErrProviderPrepareNotConfigured):
			http.Error(w, "Provider prepare is not configured on this server", http.StatusNotImplemented)
			return
		}
		h.logger.Error("Failed to list batched provider prepare summaries", zap.Error(summaryErr))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	summaries := make([]ProviderPrepareSummaryResponse, 0, len(providerIDs))
	for _, providerID := range providerIDs {
		provider, providerErr := h.service.GetProvider(r.Context(), providerID)
		if providerErr != nil {
			if errors.Is(providerErr, infrastructure.ErrProviderNotFound) {
				continue
			}
			h.logger.Error("Failed to get provider for prepare summary", zap.Error(providerErr), zap.String("provider_id", providerID.String()))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if !allTenants && provider.TenantID != scopeTenantID {
			http.Error(w, "Access denied to this tenant", http.StatusForbidden)
			return
		}

		summary := ProviderPrepareSummaryResponse{
			ProviderID: providerID.String(),
		}
		if batchedSummary, ok := batchedSummaries[providerID]; ok && batchedSummary != nil {
			if batchedSummary.RunID != nil {
				runID := batchedSummary.RunID.String()
				summary.RunID = &runID
			}
			if batchedSummary.Status != nil {
				runStatus := string(*batchedSummary.Status)
				summary.Status = &runStatus
			}
			if batchedSummary.UpdatedAt != nil {
				updatedAt := batchedSummary.UpdatedAt.Format(time.RFC3339)
				summary.UpdatedAt = &updatedAt
			}
			summary.Error = batchedSummary.ErrorMessage
			summary.LatestCheckCategory = batchedSummary.CheckCategory
			summary.LatestCheckSeverity = batchedSummary.CheckSeverity
			summary.LatestRemediationHint = batchedSummary.RemediationHint
		}
		summaries = append(summaries, summary)
	}

	payload := map[string]interface{}{
		"summaries": summaries,
	}
	if includeBatchMetrics {
		snapshot := h.service.GetPrepareSummaryBatchMetrics()
		payload["batch_metrics"] = ProviderPrepareSummaryBatchMetricsResponse{
			BatchCount:        snapshot.BatchCount,
			BatchTotalMs:      snapshot.BatchTotalMs,
			BatchMinMs:        snapshot.BatchMinMs,
			BatchMaxMs:        snapshot.BatchMaxMs,
			BatchAvgMs:        snapshot.BatchAvgMs,
			BatchErrors:       snapshot.BatchErrors,
			ProvidersTotal:    snapshot.ProvidersTotal,
			RepositoryBatches: snapshot.RepositoryBatches,
			FallbackBatches:   snapshot.FallbackBatches,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

func (h *InfrastructureProviderHandler) InstallTekton(w http.ResponseWriter, r *http.Request) {
	h.startTektonInstallerJob(w, r, infrastructure.TektonInstallerOperationInstall)
}

func (h *InfrastructureProviderHandler) UpgradeTekton(w http.ResponseWriter, r *http.Request) {
	h.startTektonInstallerJob(w, r, infrastructure.TektonInstallerOperationUpgrade)
}

func (h *InfrastructureProviderHandler) ValidateTekton(w http.ResponseWriter, r *http.Request) {
	h.startTektonInstallerJob(w, r, infrastructure.TektonInstallerOperationValidate)
}

func (h *InfrastructureProviderHandler) RetryTektonJob(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	req := TektonInstallerRetryRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	jobID, err := uuid.Parse(req.JobID)
	if err != nil {
		http.Error(w, "Invalid job_id", http.StatusBadRequest)
		return
	}

	job, err := h.service.RetryTektonInstallerJob(r.Context(), access.providerID, jobID, access.provider.TenantID, access.authCtx.UserID)
	if err != nil {
		switch {
		case errors.Is(err, infrastructure.ErrProviderNotFound):
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		case errors.Is(err, infrastructure.ErrPermissionDenied):
			http.Error(w, "Permission denied", http.StatusForbidden)
			return
		case errors.Is(err, infrastructure.ErrTektonInstallerJobNotFound):
			http.Error(w, "Tekton installer job not found", http.StatusNotFound)
			return
		case errors.Is(err, infrastructure.ErrTektonInstallerJobNotRetryable):
			http.Error(w, "Tekton installer job is not retryable (only failed jobs can be retried)", http.StatusBadRequest)
			return
		case errors.Is(err, infrastructure.ErrTektonInstallerJobInProgress):
			http.Error(w, "Another tekton install/upgrade/validate job is already in progress for this provider", http.StatusConflict)
			return
		}
		h.logger.Error("Failed to retry tekton installer job", zap.Error(err), zap.String("provider_id", access.providerID.String()), zap.String("job_id", jobID.String()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job":       h.toTektonInstallerJobResponse(job),
		"operation": "retry",
	})
}

func (h *InfrastructureProviderHandler) startTektonInstallerJob(w http.ResponseWriter, r *http.Request, operation infrastructure.TektonInstallerOperation) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	req := TektonInstallerActionRequest{}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	}

	startReq := infrastructure.StartTektonInstallerJobRequest{
		Operation:      operation,
		AssetVersion:   req.AssetVersion,
		IdempotencyKey: req.IdempotencyKey,
	}
	if req.InstallMode != "" {
		switch infrastructure.TektonInstallMode(req.InstallMode) {
		case infrastructure.TektonInstallModeGitOps, infrastructure.TektonInstallModeImageFactoryInstaller:
			startReq.InstallMode = infrastructure.TektonInstallMode(req.InstallMode)
		default:
			http.Error(w, "Invalid install_mode", http.StatusBadRequest)
			return
		}
	}

	job, err := h.service.StartTektonInstallerJob(r.Context(), access.providerID, access.provider.TenantID, access.authCtx.UserID, startReq)
	if err != nil {
		switch {
		case errors.Is(err, infrastructure.ErrProviderNotFound):
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		case errors.Is(err, infrastructure.ErrPermissionDenied):
			http.Error(w, "Permission denied", http.StatusForbidden)
			return
		case errors.Is(err, infrastructure.ErrTektonInstallerJobInProgress):
			http.Error(w, "Another tekton install/upgrade/validate job is already in progress for this provider", http.StatusConflict)
			return
		case errors.Is(err, infrastructure.ErrInvalidTektonInstallerRequest):
			http.Error(w, "Invalid tekton installer request", http.StatusBadRequest)
			return
		}
		h.logger.Error("Failed to start tekton installer job", zap.Error(err), zap.String("provider_id", access.providerID.String()), zap.String("operation", string(operation)))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job":       h.toTektonInstallerJobResponse(job),
		"operation": string(operation),
	})
}

func (h *InfrastructureProviderHandler) GetTektonStatus(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	limit := 20
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 || parsed > 100 {
			http.Error(w, "Invalid limit (must be between 1 and 100)", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	status, err := h.service.GetTektonInstallerStatus(r.Context(), access.providerID, limit)
	if err != nil {
		if errors.Is(err, infrastructure.ErrProviderNotFound) {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to get tekton installer status", zap.Error(err), zap.String("provider_id", access.providerID.String()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := TektonProviderStatusResponse{
		ProviderID:              access.providerID.String(),
		ReadinessStatus:         access.provider.ReadinessStatus,
		ReadinessMissingPrereqs: access.provider.ReadinessMissingPrereqs,
		RequiredTasks:           requiredTektonTaskNamesForReadiness(),
		RequiredPipelines:       requiredPipelineNames(access.provider.Config),
		RecentJobs:              make([]TektonInstallerJobResponse, 0, len(status.RecentJobs)),
		ActiveJobEvents:         make([]TektonInstallerJobEventResponse, 0, len(status.ActiveJobEvents)),
	}
	if access.provider.ReadinessLastChecked != nil {
		lastChecked := access.provider.ReadinessLastChecked.Format(time.RFC3339)
		response.ReadinessLastChecked = &lastChecked
	}
	response.ActiveJob = h.toTektonInstallerJobResponse(status.ActiveJob)
	for _, job := range status.RecentJobs {
		if converted := h.toTektonInstallerJobResponse(job); converted != nil {
			response.RecentJobs = append(response.RecentJobs, *converted)
		}
	}
	for _, event := range status.ActiveJobEvents {
		if converted := h.toTektonInstallerJobEventResponse(event); converted != nil {
			response.ActiveJobEvents = append(response.ActiveJobEvents, *converted)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ToggleProviderStatus enables or disables an infrastructure provider
func (h *InfrastructureProviderHandler) ToggleProviderStatus(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Determine status based on enabled flag
	var status infrastructure.ProviderStatus
	if req.Enabled {
		status = infrastructure.ProviderStatusOnline
	} else {
		status = infrastructure.ProviderStatusOffline
	}

	updateReq := &infrastructure.UpdateProviderRequest{
		Status: &status,
	}

	provider, err := h.service.UpdateProvider(r.Context(), access.providerID, updateReq)
	if err != nil {
		if err == infrastructure.ErrProviderNotFound {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to update provider status", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := h.toResponse(provider)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"provider": response,
	})
}

func requiredTektonTaskNamesForReadiness() []string {
	return []string{
		"git-clone",
		"docker-build",
		"buildx",
		"kaniko-no-push",
		"scan-image",
		"generate-sbom",
		"push-image",
		"packer",
	}
}

// GetAvailableProviders gets providers available for selection by users
func (h *InfrastructureProviderHandler) GetAvailableProviders(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in GetAvailableProviders")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	providers, err := h.service.GetAvailableProviders(r.Context(), authCtx.TenantID)
	if err != nil {
		h.logger.Error("Failed to get available providers", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response
	response := make([]InfrastructureProviderResponse, len(providers))
	for i, p := range providers {
		response[i] = h.toResponse(p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": response,
	})
}

// ListProviderPermissions lists tenant permissions for a provider.
func (h *InfrastructureProviderHandler) ListProviderPermissions(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	perms, err := h.service.ListProviderPermissions(r.Context(), access.providerID)
	if err != nil {
		h.logger.Error("Failed to list provider permissions", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]ProviderPermissionResponse, len(perms))
	for i, perm := range perms {
		var expiresAt *string
		if perm.ExpiresAt != nil {
			val := perm.ExpiresAt.Format(time.RFC3339)
			expiresAt = &val
		}
		response[i] = ProviderPermissionResponse{
			ID:         perm.ID.String(),
			ProviderID: perm.ProviderID.String(),
			TenantID:   perm.TenantID.String(),
			Permission: perm.Permission,
			GrantedBy:  perm.GrantedBy.String(),
			GrantedAt:  perm.GrantedAt.Format(time.RFC3339),
			ExpiresAt:  expiresAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"permissions": response,
	})
}

// GrantProviderPermission grants a tenant permission for a provider.
func (h *InfrastructureProviderHandler) GrantProviderPermission(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	var req GrantPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Permission == "" {
		req.Permission = "infrastructure:select"
	}
	if req.Permission != "infrastructure:select" {
		http.Error(w, "Invalid permission", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}
	if !access.authCtx.IsSystemAdmin && tenantID != access.scopeTenant {
		http.Error(w, "Access denied to this tenant", http.StatusForbidden)
		return
	}

	if err := h.service.GrantPermission(r.Context(), access.providerID, tenantID, access.authCtx.UserID, req.Permission); err != nil {
		h.logger.Error("Failed to grant provider permission", zap.Error(err))
		http.Error(w, "Failed to grant permission", http.StatusInternalServerError)
		return
	}

	if access.authCtx.IsSystemAdmin {
		if asyncErr := h.service.TriggerTenantNamespacePrepareAsync(r.Context(), access.providerID, tenantID, &access.authCtx.UserID); asyncErr != nil && !errors.Is(asyncErr, infrastructure.ErrProviderPrepareNotConfigured) {
			h.logger.Warn("Failed to auto-trigger tenant namespace prepare after permission grant",
				zap.String("provider_id", access.providerID.String()),
				zap.String("tenant_id", tenantID.String()),
				zap.Error(asyncErr),
			)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// RevokeProviderPermission revokes a tenant permission for a provider.
func (h *InfrastructureProviderHandler) RevokeProviderPermission(w http.ResponseWriter, r *http.Request) {
	access, ok := h.resolveScopedProviderAccess(w, r, true)
	if !ok {
		return
	}

	var req RevokePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Permission == "" {
		req.Permission = "infrastructure:select"
	}
	if req.Permission != "infrastructure:select" {
		http.Error(w, "Invalid permission", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}
	if !access.authCtx.IsSystemAdmin && tenantID != access.scopeTenant {
		http.Error(w, "Access denied to this tenant", http.StatusForbidden)
		return
	}

	if err := h.service.RevokePermission(r.Context(), access.providerID, tenantID, req.Permission); err != nil {
		h.logger.Error("Failed to revoke provider permission", zap.Error(err))
		http.Error(w, "Failed to revoke permission", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
