package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/imageimport"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/domain/infrastructure/connectors"
	"github.com/srikarm/image-factory/internal/domain/registryauth"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/denialtelemetry"
	k8sinfra "github.com/srikarm/image-factory/internal/infrastructure/kubernetes"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"github.com/srikarm/image-factory/internal/infrastructure/releasealerts"
	"github.com/srikarm/image-factory/internal/infrastructure/releasetelemetry"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
)

const approvalDecisionStepKey = "approval.decision"
const (
	defaultQuarantineReleasePromotionPipelineName = "image-factory-quarantine-release-promote-v1"
	defaultQuarantineReleaseDockerConfigSecret    = "docker-config"
)

type imageImportWorkflowRepository interface {
	GetInstanceWithStepsBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) (*domainworkflow.Instance, []domainworkflow.Step, error)
	UpdateStep(ctx context.Context, step *domainworkflow.Step) error
}

type imageImportNotificationReconciliationRepository interface {
	ListTenantAdminUserIDs(ctx context.Context, tenantID uuid.UUID) ([]uuid.UUID, error)
	CountImageImportNotificationReceipts(ctx context.Context, tenantID uuid.UUID, eventType, idempotencyKey string) (int, error)
	CountImageImportInAppNotifications(ctx context.Context, tenantID, importID uuid.UUID, notificationType string) (int, error)
}

type imageImportReleaseCapabilityChecker interface {
	IsQuarantineReleaseEntitled(ctx context.Context, tenantID uuid.UUID) (bool, error)
}

type imageImportInfrastructureService interface {
	GetAvailableProviders(ctx context.Context, tenantID uuid.UUID) ([]*infrastructure.Provider, error)
}

type imageImportRegistryAuthService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*registryauth.RegistryAuth, error)
	ResolveDockerConfigJSON(ctx context.Context, id uuid.UUID) ([]byte, error)
}

type ImageImportHandler struct {
	service                 *imageimport.Service
	workflowRepo            imageImportWorkflowRepository
	notificationRepo        imageImportNotificationReconciliationRepository
	infraService            imageImportInfrastructureService
	registryAuthService     imageImportRegistryAuthService
	logger                  *zap.Logger
	audit                   *audit.Service
	denials                 *denialtelemetry.Metrics
	releases                *releasetelemetry.Metrics
	config                  *systemconfig.Service
	eventBus                messaging.EventBus
	releaseCapability       imageImportReleaseCapabilityChecker
	releaseAlerts           *releasealerts.Manager
	defaultRequestType      imageimport.RequestType
	capabilityKey           string
	capabilityDeniedMessage string
	requireSOR              bool
}

func NewImageImportHandler(service *imageimport.Service, logger *zap.Logger) *ImageImportHandler {
	return &ImageImportHandler{
		service:                 service,
		logger:                  logger,
		defaultRequestType:      imageimport.RequestTypeQuarantine,
		capabilityKey:           "quarantine_request",
		capabilityDeniedMessage: "quarantine request is not enabled for this tenant",
		releaseAlerts:           releasealerts.NewManager(),
		requireSOR:              true,
	}
}

func NewOnDemandScanRequestHandler(service *imageimport.Service, logger *zap.Logger) *ImageImportHandler {
	return &ImageImportHandler{
		service:                 service,
		logger:                  logger,
		defaultRequestType:      imageimport.RequestTypeScan,
		capabilityKey:           "ondemand_image_scanning",
		capabilityDeniedMessage: "on-demand image scanning is not enabled for this tenant",
		releaseAlerts:           releasealerts.NewManager(),
		requireSOR:              false,
	}
}

func (h *ImageImportHandler) SetAuditService(auditService *audit.Service) {
	h.audit = auditService
}

func (h *ImageImportHandler) SetDenialMetrics(metrics *denialtelemetry.Metrics) {
	h.denials = metrics
}

func (h *ImageImportHandler) SetReleaseMetrics(metrics *releasetelemetry.Metrics) {
	h.releases = metrics
}

func (h *ImageImportHandler) SetSystemConfigService(configService *systemconfig.Service) {
	h.config = configService
}

func (h *ImageImportHandler) SetWorkflowRepository(workflowRepo imageImportWorkflowRepository) {
	h.workflowRepo = workflowRepo
}

func (h *ImageImportHandler) SetNotificationReconciliationRepository(notificationRepo imageImportNotificationReconciliationRepository) {
	h.notificationRepo = notificationRepo
}

func (h *ImageImportHandler) SetReleaseCapabilityChecker(checker imageImportReleaseCapabilityChecker) {
	h.releaseCapability = checker
}

func (h *ImageImportHandler) SetEventBus(bus messaging.EventBus) {
	h.eventBus = bus
}

func (h *ImageImportHandler) SetReleaseAlertManager(manager *releasealerts.Manager) {
	h.releaseAlerts = manager
}

func (h *ImageImportHandler) SetInfrastructureService(infraService imageImportInfrastructureService) {
	h.infraService = infraService
}

func (h *ImageImportHandler) SetRegistryAuthService(registryAuthService imageImportRegistryAuthService) {
	h.registryAuthService = registryAuthService
}

type createImageImportRequest struct {
	RequestType    string  `json:"request_type,omitempty"`
	EPRRecordID    string  `json:"epr_record_id"`
	SourceRegistry string  `json:"source_registry"`
	SourceImageRef string  `json:"source_image_ref"`
	RegistryAuthID *string `json:"registry_auth_id,omitempty"`
}

type rejectImportRequest struct {
	Reason string `json:"reason"`
}

type consumeReleasedArtifactRequest struct {
	ProjectID string `json:"project_id,omitempty"`
	Notes     string `json:"notes,omitempty"`
}

type releaseImportRequest struct {
	DestinationImageRef       string  `json:"destination_image_ref"`
	DestinationRegistryAuthID *string `json:"destination_registry_auth_id,omitempty"`
}

type imageImportResponse struct {
	ID                         string                                 `json:"id"`
	TenantID                   string                                 `json:"tenant_id"`
	RequestedByUserID          string                                 `json:"requested_by_user_id"`
	RequestType                string                                 `json:"request_type"`
	EPRRecordID                string                                 `json:"epr_record_id"`
	SourceRegistry             string                                 `json:"source_registry"`
	SourceImageRef             string                                 `json:"source_image_ref"`
	RegistryAuthID             *string                                `json:"registry_auth_id,omitempty"`
	Status                     string                                 `json:"status"`
	ErrorMessage               string                                 `json:"error_message,omitempty"`
	InternalImageRef           string                                 `json:"internal_image_ref,omitempty"`
	PipelineRunName            string                                 `json:"pipeline_run_name,omitempty"`
	PipelineNamespace          string                                 `json:"pipeline_namespace,omitempty"`
	PolicyDecision             string                                 `json:"policy_decision,omitempty"`
	PolicyReasonsJSON          string                                 `json:"policy_reasons_json,omitempty"`
	PolicySnapshotJSON         string                                 `json:"policy_snapshot_json,omitempty"`
	ScanSummaryJSON            string                                 `json:"scan_summary_json,omitempty"`
	SBOMSummaryJSON            string                                 `json:"sbom_summary_json,omitempty"`
	SBOMEvidenceJSON           string                                 `json:"sbom_evidence_json,omitempty"`
	SourceImageDigest          string                                 `json:"source_image_digest,omitempty"`
	DecisionTimeline           *imageImportDecisionTimeline           `json:"decision_timeline,omitempty"`
	NotificationReconciliation *imageImportNotificationReconciliation `json:"notification_reconciliation,omitempty"`
	SyncState                  string                                 `json:"sync_state,omitempty"`
	ExecutionState             string                                 `json:"execution_state,omitempty"`
	ExecutionStateUpdatedAt    string                                 `json:"execution_state_updated_at,omitempty"`
	DispatchQueuedAt           string                                 `json:"dispatch_queued_at,omitempty"`
	PipelineStartedAt          string                                 `json:"pipeline_started_at,omitempty"`
	EvidenceReadyAt            string                                 `json:"evidence_ready_at,omitempty"`
	ReleaseReadyAt             string                                 `json:"release_ready_at,omitempty"`
	FailureClass               string                                 `json:"failure_class,omitempty"`
	FailureCode                string                                 `json:"failure_code,omitempty"`
	Retryable                  bool                                   `json:"retryable"`
	ReleaseState               string                                 `json:"release_state,omitempty"`
	ReleaseEligible            bool                                   `json:"release_eligible"`
	ReleaseBlockerReason       string                                 `json:"release_blocker_reason,omitempty"`
	ReleaseReason              string                                 `json:"release_reason,omitempty"`
	CreatedAt                  string                                 `json:"created_at"`
	UpdatedAt                  string                                 `json:"updated_at"`
}

type releasedArtifactResponse struct {
	ID                 string  `json:"id"`
	TenantID           string  `json:"tenant_id"`
	RequestedByUserID  string  `json:"requested_by_user_id"`
	EPRRecordID        string  `json:"epr_record_id"`
	SourceRegistry     string  `json:"source_registry"`
	SourceImageRef     string  `json:"source_image_ref"`
	InternalImageRef   string  `json:"internal_image_ref,omitempty"`
	SourceImageDigest  string  `json:"source_image_digest,omitempty"`
	PolicyDecision     string  `json:"policy_decision,omitempty"`
	PolicySnapshotJSON string  `json:"policy_snapshot_json,omitempty"`
	ReleaseState       string  `json:"release_state"`
	ReleaseReason      string  `json:"release_reason,omitempty"`
	ReleaseActorUserID *string `json:"release_actor_user_id,omitempty"`
	ReleaseRequestedAt string  `json:"release_requested_at,omitempty"`
	ReleasedAt         string  `json:"released_at,omitempty"`
	ConsumptionReady   bool    `json:"consumption_ready"`
	ConsumptionBlocker string  `json:"consumption_blocker_reason,omitempty"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

type imageImportDecisionTimeline struct {
	DecisionStatus     string `json:"decision_status,omitempty"`
	DecisionReason     string `json:"decision_reason,omitempty"`
	DecidedByUserID    string `json:"decided_by_user_id,omitempty"`
	DecidedAt          string `json:"decided_at,omitempty"`
	WorkflowStepStatus string `json:"workflow_step_status,omitempty"`
}

type imageImportNotificationReconciliation struct {
	DecisionEventType      string `json:"decision_event_type,omitempty"`
	IdempotencyKey         string `json:"idempotency_key,omitempty"`
	ExpectedRecipients     int    `json:"expected_recipients"`
	ReceiptCount           int    `json:"receipt_count"`
	InAppNotificationCount int    `json:"in_app_notification_count"`
	DeliveryState          string `json:"delivery_state,omitempty"`
}

type imageImportLifecycleStageResponse struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	State       string `json:"state"`
	Timestamp   string `json:"timestamp,omitempty"`
}

type imageImportWorkflowResponse struct {
	InstanceID      string                              `json:"instance_id,omitempty"`
	Status          string                              `json:"status,omitempty"`
	Steps           []BuildWorkflowStepResponse         `json:"steps"`
	LifecycleStages []imageImportLifecycleStageResponse `json:"lifecycle_stages,omitempty"`
}

func (h *ImageImportHandler) CreateImportRequest(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	if authCtx.TenantID == uuid.Nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_tenant", "tenant context is required", nil)
		return
	}

	var req createImageImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_request", "invalid request body", nil)
		return
	}

	var registryAuthID *uuid.UUID
	if req.RegistryAuthID != nil && strings.TrimSpace(*req.RegistryAuthID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(*req.RegistryAuthID))
		if err != nil {
			writeImageImportError(w, http.StatusBadRequest, "invalid_registry_auth_id", "registry_auth_id must be a valid uuid", nil)
			return
		}
		registryAuthID = &parsed
	}

	created, err := h.service.CreateImportRequest(r.Context(), imageimport.CreateImportRequestInput{
		TenantID:          authCtx.TenantID,
		RequestedByUserID: authCtx.UserID,
		RequestType:       h.resolveRequestType(req.RequestType),
		SORRecordID:       strings.TrimSpace(req.EPRRecordID),
		SourceRegistry:    req.SourceRegistry,
		SourceImageRef:    req.SourceImageRef,
		RegistryAuthID:    registryAuthID,
	})
	if err != nil {
		switch {
		case errors.Is(err, imageimport.ErrOperationNotEntitled):
			h.recordDenial(r, authCtx.TenantID, authCtx.UserID, h.capabilityKey, "tenant_capability_not_entitled", audit.AuditEventCapabilityDenied, "Create import denied: capability not entitled", nil)
			writeImageImportError(w, http.StatusForbidden, "tenant_capability_not_entitled", h.capabilityDeniedMessage, map[string]interface{}{
				"tenant_id":      authCtx.TenantID.String(),
				"capability_key": h.capabilityKey,
			})
		case errors.Is(err, imageimport.ErrSORRegistrationRequired):
			h.recordDenial(
				r,
				authCtx.TenantID,
				authCtx.UserID,
				h.capabilityKey,
				"epr_registration_required",
				audit.AuditEventSORDenied,
				"Create import denied: EPR registration required",
				h.eprDenialLabels(r.Context(), authCtx.TenantID),
			)
			writeImageImportError(w, http.StatusPreconditionFailed, "epr_registration_required", "enterprise EPR registration is required before requesting quarantine import", map[string]interface{}{
				"tenant_id":     authCtx.TenantID.String(),
				"epr_record_id": strings.TrimSpace(req.EPRRecordID),
			})
		case errors.Is(err, imageimport.ErrInvalidSORRecordID),
			errors.Is(err, imageimport.ErrInvalidSourceRegistry),
			errors.Is(err, imageimport.ErrInvalidSourceImageRef):
			writeImageImportError(w, http.StatusBadRequest, "validation_failed", err.Error(), nil)
		default:
			h.logger.Error("Failed to create image import request", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()))
			writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to create import request", nil)
		}
		return
	}

	response := map[string]interface{}{
		"data": mapImportResponse(created, nil, nil),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *ImageImportHandler) ListImportRequests(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	if authCtx.TenantID == uuid.Nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_tenant", "tenant context is required", nil)
		return
	}

	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 20)
	offset := (page - 1) * limit

	items, err := h.service.ListImportRequests(r.Context(), authCtx.TenantID, h.defaultRequestType, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list image import requests", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()))
		writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to list import requests", nil)
		return
	}

	rows := make([]imageImportResponse, 0, len(items))
	for _, item := range items {
		timeline := h.loadDecisionTimeline(r.Context(), item.ID)
		reconciliation := h.loadNotificationReconciliation(r.Context(), item, timeline)
		rows = append(rows, mapImportResponse(item, timeline, reconciliation))
	}

	response := map[string]interface{}{
		"data": rows,
		"pagination": map[string]int{
			"page":  page,
			"limit": limit,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *ImageImportHandler) ListAllImportRequests(w http.ResponseWriter, r *http.Request) {
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 20)
	offset := (page - 1) * limit

	items, err := h.service.ListAllImportRequests(r.Context(), h.defaultRequestType, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list all image import requests", zap.Error(err))
		writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to list import requests", nil)
		return
	}

	rows := make([]imageImportResponse, 0, len(items))
	for _, item := range items {
		timeline := h.loadDecisionTimeline(r.Context(), item.ID)
		reconciliation := h.loadNotificationReconciliation(r.Context(), item, timeline)
		rows = append(rows, mapImportResponse(item, timeline, reconciliation))
	}

	response := map[string]interface{}{
		"data": rows,
		"pagination": map[string]int{
			"page":  page,
			"limit": limit,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *ImageImportHandler) ListReleasedArtifacts(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	if authCtx.TenantID == uuid.Nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_tenant", "tenant context is required", nil)
		return
	}

	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 20)
	offset := (page - 1) * limit
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	items, total, err := h.service.ListReleasedArtifacts(r.Context(), authCtx.TenantID, search, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list released artifacts", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()))
		writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to list released artifacts", nil)
		return
	}

	rows := make([]releasedArtifactResponse, 0, len(items))
	for _, item := range items {
		rows = append(rows, mapReleasedArtifactResponse(item))
	}

	response := map[string]interface{}{
		"data": rows,
		"pagination": map[string]int{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *ImageImportHandler) ConsumeReleasedArtifact(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	if authCtx.TenantID == uuid.Nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_tenant", "tenant context is required", nil)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_id", "released artifact id must be a valid uuid", nil)
		return
	}

	var req consumeReleasedArtifactRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	item, err := h.service.GetImportRequest(r.Context(), authCtx.TenantID, id)
	if err != nil {
		switch {
		case errors.Is(err, imageimport.ErrImportNotFound):
			writeImageImportError(w, http.StatusNotFound, "not_found", "released artifact not found", nil)
		default:
			h.logger.Error("Failed to load released artifact for consumption", zap.Error(err), zap.String("import_id", id.String()))
			writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to load released artifact", nil)
		}
		return
	}

	if item.RequestType != imageimport.RequestTypeQuarantine {
		writeImageImportError(w, http.StatusNotFound, "not_found", "released artifact not found", nil)
		return
	}

	releaseProjection := imageimport.ResolveReleaseProjection(item)
	if releaseProjection.State != imageimport.ReleaseStateReleased || strings.TrimSpace(item.InternalImageRef) == "" {
		writeImageImportError(w, http.StatusConflict, "release_not_eligible", "artifact is not consumable yet", map[string]interface{}{
			"release_state":            string(releaseProjection.State),
			"release_blocker_reason":   releaseProjection.BlockerReason,
			"consumption_blocker_hint": "artifact must be released with internal image reference",
		})
		return
	}

	extra := map[string]interface{}{}
	if projectID := strings.TrimSpace(req.ProjectID); projectID != "" {
		extra["project_id"] = projectID
	}
	if notes := strings.TrimSpace(req.Notes); notes != "" {
		extra["notes"] = notes
	}
	h.publishReleaseEvent(r.Context(), authCtx.UserID, item, messaging.EventTypeQuarantineReleaseConsumed, "released artifact selected for tenant build intake", extra)

	if h.audit != nil {
		_ = h.audit.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventAPICall, "image_import", "consumed", "Released artifact consumed by tenant workflow", map[string]interface{}{
			"import_id":           item.ID.String(),
			"project_id":          strings.TrimSpace(req.ProjectID),
			"internal_image_ref":  item.InternalImageRef,
			"source_image_digest": item.SourceImageDigest,
			"release_state":       string(releaseProjection.State),
		})
	}

	response := map[string]interface{}{
		"data": map[string]interface{}{
			"id":                 item.ID.String(),
			"consumed":           true,
			"internal_image_ref": item.InternalImageRef,
			"project_id":         strings.TrimSpace(req.ProjectID),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *ImageImportHandler) GetImportRequest(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	if authCtx.TenantID == uuid.Nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_tenant", "tenant context is required", nil)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_id", "import request id must be a valid uuid", nil)
		return
	}

	item, err := h.service.GetImportRequest(r.Context(), authCtx.TenantID, id)
	if err != nil {
		switch {
		case errors.Is(err, imageimport.ErrImportNotFound):
			writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		default:
			h.logger.Error("Failed to get image import request", zap.Error(err), zap.String("import_id", id.String()))
			writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to get import request", nil)
		}
		return
	}
	if h.defaultRequestType != "" && item.RequestType != h.defaultRequestType {
		writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	timeline := h.loadDecisionTimeline(r.Context(), item.ID)
	reconciliation := h.loadNotificationReconciliation(r.Context(), item, timeline)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": mapImportResponse(item, timeline, reconciliation)})
}

func (h *ImageImportHandler) GetImportRequestWorkflow(w http.ResponseWriter, r *http.Request) {
	item, authCtx, err := h.resolveImportRequestForRead(r.Context(), r, false)
	if err != nil {
		h.writeResolveImportError(w, err)
		return
	}
	if item == nil {
		writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		return
	}
	if authCtx != nil && authCtx.TenantID != uuid.Nil && item.TenantID != authCtx.TenantID {
		writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		return
	}

	resp := BuildWorkflowResponse{Steps: []BuildWorkflowStepResponse{}}
	if h.workflowRepo != nil {
		instance, steps, wfErr := h.workflowRepo.GetInstanceWithStepsBySubject(r.Context(), "external_image_import", item.ID)
		if wfErr == nil && instance != nil {
			resp = imageImportWorkflowToResponse(instance, steps)
		} else if wfErr != nil {
			h.logger.Warn("Failed to load import workflow instance",
				zap.Error(wfErr),
				zap.String("import_id", item.ID.String()),
			)
		}
	}
	lifecycleStages := buildImportLifecycleStages(item)
	out := imageImportWorkflowResponse{
		InstanceID:      resp.InstanceID,
		Status:          resp.Status,
		Steps:           resp.Steps,
		LifecycleStages: lifecycleStages,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
}

func (h *ImageImportHandler) GetImportRequestWorkflowAdmin(w http.ResponseWriter, r *http.Request) {
	item, _, err := h.resolveImportRequestForRead(r.Context(), r, true)
	if err != nil {
		h.writeResolveImportError(w, err)
		return
	}
	if item == nil {
		writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		return
	}

	resp := BuildWorkflowResponse{Steps: []BuildWorkflowStepResponse{}}
	if h.workflowRepo != nil {
		instance, steps, wfErr := h.workflowRepo.GetInstanceWithStepsBySubject(r.Context(), "external_image_import", item.ID)
		if wfErr == nil && instance != nil {
			resp = imageImportWorkflowToResponse(instance, steps)
		} else if wfErr != nil {
			h.logger.Warn("Failed to load import workflow instance",
				zap.Error(wfErr),
				zap.String("import_id", item.ID.String()),
			)
		}
	}
	lifecycleStages := buildImportLifecycleStages(item)
	out := imageImportWorkflowResponse{
		InstanceID:      resp.InstanceID,
		Status:          resp.Status,
		Steps:           resp.Steps,
		LifecycleStages: lifecycleStages,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
}

func (h *ImageImportHandler) GetImportRequestLogs(w http.ResponseWriter, r *http.Request) {
	item, authCtx, err := h.resolveImportRequestForRead(r.Context(), r, false)
	if err != nil {
		h.writeResolveImportError(w, err)
		return
	}
	if item == nil {
		writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		return
	}
	if authCtx != nil && authCtx.TenantID != uuid.Nil && item.TenantID != authCtx.TenantID {
		writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		return
	}
	h.writeImportLogsResponse(w, r, item)
}

func (h *ImageImportHandler) GetImportRequestLogsAdmin(w http.ResponseWriter, r *http.Request) {
	item, _, err := h.resolveImportRequestForRead(r.Context(), r, true)
	if err != nil {
		h.writeResolveImportError(w, err)
		return
	}
	if item == nil {
		writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		return
	}
	h.writeImportLogsResponse(w, r, item)
}

func (h *ImageImportHandler) RetryImportRequest(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	if authCtx.TenantID == uuid.Nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_tenant", "tenant context is required", nil)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_id", "import request id must be a valid uuid", nil)
		return
	}

	created, err := h.service.RetryImportRequest(r.Context(), authCtx.TenantID, authCtx.UserID, id)
	if err != nil {
		var retryBackoffErr *imageimport.RetryBackoffError
		var retryLimitErr *imageimport.RetryAttemptLimitError
		switch {
		case errors.Is(err, imageimport.ErrImportNotFound):
			writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		case errors.Is(err, imageimport.ErrImportNotRetryable):
			writeImageImportError(w, http.StatusBadRequest, "import_not_retryable", "import request is not retryable in its current state", nil)
		case errors.As(err, &retryBackoffErr):
			remaining := int(retryBackoffErr.Remaining.Seconds())
			if remaining < 1 {
				remaining = 1
			}
			writeImageImportError(w, http.StatusTooManyRequests, "retry_backoff_active", "retry backoff is active for this request", map[string]interface{}{
				"retry_after_seconds": remaining,
				"retry_available_at":  time.Now().UTC().Add(retryBackoffErr.Remaining).Format(timeRFC3339),
			})
		case errors.As(err, &retryLimitErr):
			writeImageImportError(w, http.StatusConflict, "retry_attempt_limit_reached", "retry attempt limit reached for this import request", map[string]interface{}{
				"max_attempts":     retryLimitErr.MaxAttempts,
				"current_attempts": retryLimitErr.Current,
			})
		case errors.Is(err, imageimport.ErrOperationNotEntitled):
			h.recordDenial(r, authCtx.TenantID, authCtx.UserID, h.capabilityKey, "tenant_capability_not_entitled", audit.AuditEventCapabilityDenied, "Retry import denied: capability not entitled", nil)
			writeImageImportError(w, http.StatusForbidden, "tenant_capability_not_entitled", h.capabilityDeniedMessage, map[string]interface{}{
				"tenant_id":      authCtx.TenantID.String(),
				"capability_key": h.capabilityKey,
			})
		case errors.Is(err, imageimport.ErrSORRegistrationRequired):
			h.recordDenial(
				r,
				authCtx.TenantID,
				authCtx.UserID,
				h.capabilityKey,
				"epr_registration_required",
				audit.AuditEventSORDenied,
				"Retry import denied: EPR registration required",
				h.eprDenialLabels(r.Context(), authCtx.TenantID),
			)
			writeImageImportError(w, http.StatusPreconditionFailed, "epr_registration_required", "enterprise EPR registration is required before requesting quarantine import", map[string]interface{}{
				"tenant_id": authCtx.TenantID.String(),
			})
		default:
			h.logger.Error("Failed to retry image import request", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()), zap.String("import_id", id.String()))
			writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to retry import request", nil)
		}
		return
	}
	if h.defaultRequestType != "" && created.RequestType != h.defaultRequestType {
		writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		return
	}

	response := map[string]interface{}{
		"data": mapImportResponse(created, nil, nil),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *ImageImportHandler) resolveImportRequestForRead(ctx context.Context, r *http.Request, admin bool) (*imageimport.ImportRequest, *middleware.AuthContext, error) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		return nil, nil, fmt.Errorf("unauthorized")
	}

	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, authCtx, fmt.Errorf("invalid_id")
	}

	if !admin {
		if authCtx.TenantID == uuid.Nil {
			return nil, authCtx, fmt.Errorf("invalid_tenant")
		}
		item, getErr := h.service.GetImportRequest(ctx, authCtx.TenantID, id)
		if getErr != nil {
			return nil, authCtx, getErr
		}
		if h.defaultRequestType != "" && item.RequestType != h.defaultRequestType {
			return nil, authCtx, imageimport.ErrImportNotFound
		}
		return item, authCtx, nil
	}

	item, findErr := h.findImportRequestAcrossTenants(ctx, id)
	if findErr != nil {
		return nil, authCtx, findErr
	}
	if item == nil {
		return nil, authCtx, imageimport.ErrImportNotFound
	}
	if h.defaultRequestType != "" && item.RequestType != h.defaultRequestType {
		return nil, authCtx, imageimport.ErrImportNotFound
	}
	return item, authCtx, nil
}

func (h *ImageImportHandler) writeResolveImportError(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		return
	case err.Error() == "unauthorized":
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
	case err.Error() == "invalid_tenant":
		writeImageImportError(w, http.StatusBadRequest, "invalid_tenant", "tenant context is required", nil)
	case err.Error() == "invalid_id":
		writeImageImportError(w, http.StatusBadRequest, "invalid_id", "import request id must be a valid uuid", nil)
	case errors.Is(err, imageimport.ErrImportNotFound):
		writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
	default:
		writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to get import request", nil)
	}
}

func (h *ImageImportHandler) findImportRequestAcrossTenants(ctx context.Context, id uuid.UUID) (*imageimport.ImportRequest, error) {
	const (
		pageSize = 200
		maxPages = 50
	)
	for page := 1; page <= maxPages; page++ {
		offset := (page - 1) * pageSize
		rows, err := h.service.ListAllImportRequests(ctx, h.defaultRequestType, pageSize, offset)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if row != nil && row.ID == id {
				return row, nil
			}
		}
		if len(rows) < pageSize {
			break
		}
	}
	return nil, imageimport.ErrImportNotFound
}

func imageImportWorkflowToResponse(instance *domainworkflow.Instance, steps []domainworkflow.Step) BuildWorkflowResponse {
	resp := BuildWorkflowResponse{
		Steps: []BuildWorkflowStepResponse{},
	}
	if instance == nil {
		return resp
	}
	resp.InstanceID = instance.ID.String()
	resp.Status = string(instance.Status)
	for _, step := range steps {
		stepCopy := step
		item := BuildWorkflowStepResponse{
			StepKey:     stepCopy.StepKey,
			Status:      string(stepCopy.Status),
			Attempts:    stepCopy.Attempts,
			StartedAt:   stepCopy.StartedAt,
			CompletedAt: stepCopy.CompletedAt,
			CreatedAt:   stepCopy.CreatedAt,
			UpdatedAt:   stepCopy.UpdatedAt,
		}
		item.LastError = stepCopy.LastError
		resp.Steps = append(resp.Steps, item)
	}
	sort.SliceStable(resp.Steps, func(i, j int) bool {
		if resp.Steps[i].CreatedAt.Equal(resp.Steps[j].CreatedAt) {
			return resp.Steps[i].StepKey < resp.Steps[j].StepKey
		}
		return resp.Steps[i].CreatedAt.Before(resp.Steps[j].CreatedAt)
	})
	return resp
}

func buildImportLifecycleStages(item *imageimport.ImportRequest) []imageImportLifecycleStageResponse {
	if item == nil {
		return []imageImportLifecycleStageResponse{}
	}
	syncState, _ := deriveImportSyncState(item)
	_, _, dispatchQueuedAt, pipelineStartedAt, evidenceReadyAt, releaseReadyAt :=
		deriveImportExecutionContract(item, imageimport.ResolveReleaseProjection(item), syncState)

	stageDefs := []struct {
		Key         string
		Label       string
		Description string
		Timestamp   string
	}{
		{
			Key:         "awaiting_approval",
			Label:       "Awaiting Approval",
			Description: "Security reviewer decision required",
		},
		{
			Key:         "awaiting_dispatch",
			Label:       "Dispatch Queue",
			Description: "Waiting for eligible Tekton provider",
			Timestamp:   strings.TrimSpace(dispatchQueuedAt),
		},
		{
			Key:         "pipeline_running",
			Label:       "Pipeline Running",
			Description: "Quarantine pipeline is executing",
			Timestamp:   strings.TrimSpace(pipelineStartedAt),
		},
		{
			Key:         "evidence_pending",
			Label:       "Evidence Processing",
			Description: "Scan/SBOM evidence ingest and policy evaluation",
			Timestamp:   strings.TrimSpace(evidenceReadyAt),
		},
		{
			Key:         "ready_for_release",
			Label:       "Ready For Release",
			Description: "Reviewer can release or block",
			Timestamp:   strings.TrimSpace(releaseReadyAt),
		},
		{
			Key:         "completed",
			Label:       "Completed",
			Description: "Terminal request state",
		},
	}

	currentKey := "awaiting_approval"
	if item.Status == imageimport.StatusFailed {
		switch syncState {
		case "dispatch_failed":
			currentKey = "awaiting_dispatch"
		case "pipeline_running":
			currentKey = "pipeline_running"
		default:
			currentKey = "evidence_pending"
		}
	} else if item.Status == imageimport.StatusSuccess || item.Status == imageimport.StatusQuarantined {
		currentKey = "completed"
	} else {
		switch syncState {
		case "awaiting_dispatch":
			currentKey = "awaiting_dispatch"
		case "pipeline_running":
			currentKey = "pipeline_running"
		case "catalog_sync_pending":
			currentKey = "evidence_pending"
		case "completed":
			currentKey = "completed"
		}
	}

	currentIndex := 0
	for i := range stageDefs {
		if stageDefs[i].Key == currentKey {
			currentIndex = i
			break
		}
	}

	stages := make([]imageImportLifecycleStageResponse, 0, len(stageDefs))
	for i, def := range stageDefs {
		state := "pending"
		if item.Status == imageimport.StatusFailed && i == currentIndex {
			state = "failed"
		} else if i < currentIndex {
			state = "complete"
		} else if i == currentIndex {
			state = "current"
		}
		stages = append(stages, imageImportLifecycleStageResponse{
			Key:         def.Key,
			Label:       def.Label,
			Description: def.Description,
			State:       state,
			Timestamp:   def.Timestamp,
		})
	}
	return stages
}

func (h *ImageImportHandler) writeImportLogsResponse(w http.ResponseWriter, r *http.Request, item *imageimport.ImportRequest) {
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 100)
	if limit > 1000 {
		limit = 1000
	}
	offset := parsePositiveInt(r.URL.Query().Get("offset"), 0)
	logFilter, filterErr := parseBuildLogsFilter(r.URL.Query())
	if filterErr != nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_request", filterErr.Error(), nil)
		return
	}

	filteredLogs := h.listImportLogs(r.Context(), item, logFilter)
	sort.SliceStable(filteredLogs, func(i, j int) bool {
		return filteredLogs[i].Timestamp > filteredLogs[j].Timestamp
	})
	total := len(filteredLogs)
	if offset >= total {
		filteredLogs = []LogEntry{}
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		filteredLogs = filteredLogs[offset:end]
	}
	hasMore := offset+limit < total
	response := map[string]interface{}{
		"import_request_id": item.ID.String(),
		"logs":              filteredLogs,
		"total":             total,
		"has_more":          hasMore,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *ImageImportHandler) buildImportLifecycleLogs(item *imageimport.ImportRequest, timeline *imageImportDecisionTimeline, reconciliation *imageImportNotificationReconciliation) []LogEntry {
	logs := make([]LogEntry, 0, 16)
	syncState, _ := deriveImportSyncState(item)
	releaseProjection := imageimport.ResolveReleaseProjection(item)
	executionState, _, dispatchQueuedAt, pipelineStartedAt, evidenceReadyAt, releaseReadyAt :=
		deriveImportExecutionContract(item, releaseProjection, syncState)
	appendLog := func(ts time.Time, level build.LogLevel, message string, metadata map[string]interface{}) {
		if ts.IsZero() {
			return
		}
		entry := LogEntry{
			Timestamp: ts.UTC().Format(time.RFC3339),
			Level:     string(level),
			Message:   message,
			Metadata:  metadata,
		}
		logs = append(logs, entry)
	}

	appendLog(item.CreatedAt, build.LogInfo, "quarantine request created", map[string]interface{}{
		"source":  "lifecycle",
		"status":  string(item.Status),
		"request": item.ID.String(),
	})
	appendLog(item.UpdatedAt, build.LogInfo, "quarantine request state updated", map[string]interface{}{
		"source":          "lifecycle",
		"status":          string(item.Status),
		"sync_state":      syncState,
		"execution_state": executionState,
	})
	if dispatchQueuedAt != "" {
		if ts, err := time.Parse(timeRFC3339, dispatchQueuedAt); err == nil {
			appendLog(ts, build.LogInfo, "request queued for dispatch", map[string]interface{}{
				"source": "lifecycle",
			})
		}
	}
	if pipelineStartedAt != "" {
		if ts, err := time.Parse(timeRFC3339, pipelineStartedAt); err == nil {
			appendLog(ts, build.LogInfo, "pipeline started", map[string]interface{}{
				"source":       "lifecycle",
				"pipeline_run": item.PipelineRunName,
				"namespace":    item.PipelineNamespace,
			})
		}
	}
	if evidenceReadyAt != "" {
		if ts, err := time.Parse(timeRFC3339, evidenceReadyAt); err == nil {
			appendLog(ts, build.LogInfo, "evidence became available", map[string]interface{}{
				"source": "lifecycle",
			})
		}
	}
	if releaseReadyAt != "" {
		if ts, err := time.Parse(timeRFC3339, releaseReadyAt); err == nil {
			appendLog(ts, build.LogInfo, "request is ready for release decision", map[string]interface{}{
				"source": "lifecycle",
			})
		}
	}
	if timeline != nil {
		if decidedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(timeline.DecidedAt)); err == nil {
			appendLog(decidedAt, build.LogInfo, "approval decision recorded", map[string]interface{}{
				"source":          "lifecycle",
				"decision_status": timeline.DecisionStatus,
				"workflow_step":   timeline.WorkflowStepStatus,
			})
		}
	}
	if reconciliation != nil {
		appendLog(item.UpdatedAt, build.LogInfo, "notification reconciliation updated", map[string]interface{}{
			"source":               "lifecycle",
			"delivery_state":       reconciliation.DeliveryState,
			"receipt_count":        reconciliation.ReceiptCount,
			"in_app_notifications": reconciliation.InAppNotificationCount,
		})
	}
	if strings.TrimSpace(item.ErrorMessage) != "" {
		appendLog(item.UpdatedAt, build.LogError, item.ErrorMessage, map[string]interface{}{
			"source": "lifecycle",
		})
	}
	return logs
}

func (h *ImageImportHandler) fetchImportTektonLogs(ctx context.Context, item *imageimport.ImportRequest) []LogEntry {
	if h.infraService == nil || item == nil {
		return nil
	}
	namespace := strings.TrimSpace(item.PipelineNamespace)
	pipelineRun := strings.TrimSpace(item.PipelineRunName)
	if namespace == "" || pipelineRun == "" {
		return nil
	}

	providers, err := h.infraService.GetAvailableProviders(ctx, item.TenantID)
	if err != nil {
		h.logger.Warn("Failed to list providers for import logs",
			zap.Error(err),
			zap.String("tenant_id", item.TenantID.String()),
			zap.String("import_id", item.ID.String()),
		)
		return nil
	}
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		restCfg, cfgErr := connectors.BuildRESTConfigFromProviderConfig(provider.Config)
		if cfgErr != nil {
			continue
		}
		k8sClient, kErr := k8sclient.NewForConfig(restCfg)
		tektonClient, tErr := tektonclient.NewForConfig(restCfg)
		if kErr != nil || tErr != nil {
			continue
		}
		pipelineMgr := k8sinfra.NewKubernetesPipelineManager(k8sClient, tektonClient, h.logger)
		if _, runErr := tektonClient.TektonV1().PipelineRuns(namespace).Get(ctx, pipelineRun, metav1.GetOptions{}); runErr != nil {
			if apierrors.IsNotFound(runErr) {
				continue
			}
			continue
		}
		logsMap, getErr := pipelineMgr.GetLogs(ctx, namespace, pipelineRun)
		statusRows := h.fetchImportTektonStatusSignals(ctx, tektonClient, namespace, pipelineRun, provider.ID.String())
		if getErr != nil && len(statusRows) == 0 {
			continue
		}
		rows := make([]LogEntry, 0, len(logsMap)+len(statusRows))
		for key, content := range logsMap {
			parts := strings.SplitN(key, "/", 2)
			taskRunName := parts[0]
			stepName := ""
			if len(parts) > 1 {
				stepName = parts[1]
			}
			lines := strings.Split(content, "\n")
			for _, rawLine := range lines {
				line := strings.TrimSpace(rawLine)
				if line == "" {
					continue
				}
				entryTs, msg := parseTektonTimestampedLine(line)
				if msg == "" {
					continue
				}
				if entryTs == "" {
					entryTs = time.Now().UTC().Format(time.RFC3339)
				}
				rows = append(rows, LogEntry{
					Timestamp: entryTs,
					Level:     string(build.LogInfo),
					Message:   msg,
					Metadata: map[string]interface{}{
						"source":       "tekton",
						"pipeline_run": pipelineRun,
						"namespace":    namespace,
						"task_run":     taskRunName,
						"step":         stepName,
						"provider_id":  provider.ID.String(),
					},
				})
			}
		}
		rows = append(rows, statusRows...)
		return rows
	}
	return nil
}

func (h *ImageImportHandler) fetchImportTektonStatusSignals(ctx context.Context, tektonClient tektonclient.Interface, namespace, pipelineRun, providerID string) []LogEntry {
	if tektonClient == nil || strings.TrimSpace(namespace) == "" || strings.TrimSpace(pipelineRun) == "" {
		return nil
	}
	taskRuns, err := tektonClient.TektonV1().TaskRuns(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "tekton.dev/pipelineRun=" + pipelineRun,
	})
	if err != nil || taskRuns == nil || len(taskRuns.Items) == 0 {
		return nil
	}

	rows := make([]LogEntry, 0, len(taskRuns.Items))
	for i := range taskRuns.Items {
		taskRun := taskRuns.Items[i]
		taskRunName := strings.TrimSpace(taskRun.Name)
		taskName := strings.TrimSpace(taskRun.Spec.TaskRef.Name)

		conditionReason := ""
		conditionMessage := ""
		conditionStatus := ""
		for _, cond := range taskRun.Status.Conditions {
			if strings.EqualFold(string(cond.Type), "Succeeded") {
				conditionStatus = strings.TrimSpace(string(cond.Status))
				conditionReason = strings.TrimSpace(cond.Reason)
				conditionMessage = strings.TrimSpace(cond.Message)
				break
			}
		}

		if conditionStatus == "False" {
			ts := time.Now().UTC().Format(time.RFC3339)
			if taskRun.Status.CompletionTime != nil && !taskRun.Status.CompletionTime.IsZero() {
				ts = taskRun.Status.CompletionTime.UTC().Format(time.RFC3339)
			}
			message := conditionMessage
			if message == "" {
				message = "TaskRun failed"
			}
			rows = append(rows, LogEntry{
				Timestamp: ts,
				Level:     string(build.LogError),
				Message:   message,
				Metadata: map[string]interface{}{
					"source":       "tekton",
					"signal_type":  "taskrun_status",
					"reason":       conditionReason,
					"status":       conditionStatus,
					"pipeline_run": pipelineRun,
					"namespace":    namespace,
					"task_run":     taskRunName,
					"task":         taskName,
					"provider_id":  providerID,
				},
			})
		}

		for _, stepState := range taskRun.Status.Steps {
			terminated := stepState.Terminated
			if terminated == nil || terminated.ExitCode == 0 {
				continue
			}
			stepName := strings.TrimSpace(stepState.Name)
			ts := time.Now().UTC().Format(time.RFC3339)
			if !terminated.FinishedAt.IsZero() {
				ts = terminated.FinishedAt.UTC().Format(time.RFC3339)
			}
			message := strings.TrimSpace(terminated.Message)
			if message == "" {
				message = "Step terminated with non-zero exit code"
			}
			rows = append(rows, LogEntry{
				Timestamp: ts,
				Level:     string(build.LogError),
				Message:   message,
				Metadata: map[string]interface{}{
					"source":       "tekton",
					"signal_type":  "step_status",
					"reason":       strings.TrimSpace(terminated.Reason),
					"exit_code":    terminated.ExitCode,
					"pipeline_run": pipelineRun,
					"namespace":    namespace,
					"task_run":     taskRunName,
					"task":         taskName,
					"step":         stepName,
					"provider_id":  providerID,
				},
			})
		}
	}
	return rows
}

func parseTektonTimestampedLine(line string) (timestamp string, message string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", ""
	}
	firstSpace := strings.Index(trimmed, " ")
	if firstSpace <= 0 {
		return "", trimmed
	}
	candidateTs := strings.TrimSpace(trimmed[:firstSpace])
	if _, err := time.Parse(time.RFC3339, candidateTs); err == nil {
		return candidateTs, strings.TrimSpace(trimmed[firstSpace+1:])
	}
	return "", trimmed
}

func (h *ImageImportHandler) WithdrawImportRequest(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	if authCtx.TenantID == uuid.Nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_tenant", "tenant context is required", nil)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_id", "import request id must be a valid uuid", nil)
		return
	}

	var req rejectImportRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	updated, err := h.service.WithdrawImportRequest(r.Context(), authCtx.TenantID, authCtx.UserID, id, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, imageimport.ErrImportNotFound):
			writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		case errors.Is(err, imageimport.ErrImportNotWithdrawable):
			writeImageImportError(w, http.StatusConflict, "import_not_withdrawable", "import request can only be withdrawn while pending", nil)
		default:
			h.logger.Error("Failed to withdraw image import request", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()), zap.String("import_id", id.String()))
			writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to withdraw import request", nil)
		}
		return
	}
	if h.defaultRequestType != "" && updated.RequestType != h.defaultRequestType {
		writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		return
	}

	response := map[string]interface{}{
		"data": mapImportResponse(updated, nil, nil),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *ImageImportHandler) ApproveImportRequest(w http.ResponseWriter, r *http.Request) {
	h.applyApprovalDecision(w, r, true, "", true)
}

func (h *ImageImportHandler) ApproveImportRequestAdmin(w http.ResponseWriter, r *http.Request) {
	h.applyApprovalDecision(w, r, true, "", false)
}

func (h *ImageImportHandler) RejectImportRequest(w http.ResponseWriter, r *http.Request) {
	var req rejectImportRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	h.applyApprovalDecision(w, r, false, req.Reason, true)
}

func (h *ImageImportHandler) RejectImportRequestAdmin(w http.ResponseWriter, r *http.Request) {
	var req rejectImportRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	h.applyApprovalDecision(w, r, false, req.Reason, false)
}

func (h *ImageImportHandler) ReleaseImportRequest(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	if authCtx.TenantID == uuid.Nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_tenant", "tenant context is required", nil)
		return
	}
	var releaseReq releaseImportRequest
	if err := json.NewDecoder(r.Body).Decode(&releaseReq); err != nil && !errors.Is(err, io.EOF) {
		writeImageImportError(w, http.StatusBadRequest, "invalid_request", "failed to parse release request", nil)
		return
	}
	destinationImageRef := strings.TrimSpace(releaseReq.DestinationImageRef)
	if destinationImageRef == "" {
		writeImageImportError(w, http.StatusBadRequest, "validation_failed", "destination_image_ref is required", map[string]interface{}{
			"field": "destination_image_ref",
		})
		return
	}
	if !isValidDestinationImageRef(destinationImageRef) {
		writeImageImportError(w, http.StatusBadRequest, "validation_failed", "destination_image_ref must be a fully qualified image reference", map[string]interface{}{
			"field": "destination_image_ref",
		})
		return
	}
	var destinationRegistryAuthID *uuid.UUID
	if releaseReq.DestinationRegistryAuthID != nil && strings.TrimSpace(*releaseReq.DestinationRegistryAuthID) != "" {
		parsed, parseErr := uuid.Parse(strings.TrimSpace(*releaseReq.DestinationRegistryAuthID))
		if parseErr != nil {
			writeImageImportError(w, http.StatusBadRequest, "invalid_registry_auth_id", "destination_registry_auth_id must be a valid uuid", nil)
			return
		}
		destinationRegistryAuthID = &parsed
	}
	if destinationRegistryAuthID == nil {
		writeImageImportError(w, http.StatusBadRequest, "validation_failed", "destination_registry_auth_id is required", map[string]interface{}{
			"field": "destination_registry_auth_id",
		})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_id", "import request id must be a valid uuid", nil)
		return
	}

	if h.releaseCapability == nil {
		writeImageImportError(w, http.StatusForbidden, "tenant_capability_not_entitled", "quarantine release is not enabled for this tenant", map[string]interface{}{
			"tenant_id":      authCtx.TenantID.String(),
			"capability_key": "quarantine_release",
		})
		return
	}

	entitled, capabilityErr := h.releaseCapability.IsQuarantineReleaseEntitled(r.Context(), authCtx.TenantID)
	if capabilityErr != nil {
		h.logger.Error("Failed to resolve quarantine release capability", zap.Error(capabilityErr), zap.String("tenant_id", authCtx.TenantID.String()))
		writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to evaluate release entitlement", nil)
		return
	}
	if !entitled {
		h.recordDenial(r, authCtx.TenantID, authCtx.UserID, "quarantine_release", "tenant_capability_not_entitled", audit.AuditEventCapabilityDenied, "Release import denied: capability not entitled", nil)
		writeImageImportError(w, http.StatusForbidden, "tenant_capability_not_entitled", "quarantine release is not enabled for this tenant", map[string]interface{}{
			"tenant_id":      authCtx.TenantID.String(),
			"capability_key": "quarantine_release",
		})
		return
	}

	item, err := h.service.GetImportRequest(r.Context(), authCtx.TenantID, id)
	if err != nil {
		switch {
		case errors.Is(err, imageimport.ErrImportNotFound):
			writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		default:
			h.logger.Error("Failed to load import request for release", zap.Error(err), zap.String("import_id", id.String()))
			writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to get import request", nil)
		}
		return
	}
	if h.defaultRequestType != "" && item.RequestType != h.defaultRequestType {
		writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		return
	}

	releaseProjection := imageimport.ResolveReleaseProjection(item)
	eligibilityProjection := imageimport.DeriveReleaseProjection(item)
	if !eligibilityProjection.Eligible {
		h.publishReleaseEvent(r.Context(), authCtx.UserID, item, messaging.EventTypeQuarantineReleaseFailed, "import request is not eligible for release", map[string]interface{}{
			"release_blocker_reason": eligibilityProjection.BlockerReason,
		})
		if h.audit != nil {
			_ = h.audit.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventAPICall, "image_import", "release_failed", "Release import denied: import request is not eligible", map[string]interface{}{
				"import_id":              item.ID.String(),
				"release_state":          string(eligibilityProjection.State),
				"release_blocker_reason": eligibilityProjection.BlockerReason,
			})
		}
		writeImageImportError(w, http.StatusConflict, "release_not_eligible", "import request is not eligible for release", map[string]interface{}{
			"import_id":              item.ID.String(),
			"import_status":          string(item.Status),
			"release_state":          string(eligibilityProjection.State),
			"release_blocker_reason": eligibilityProjection.BlockerReason,
		})
		return
	}
	if h.audit != nil {
		_ = h.audit.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventAPICall, "image_import", "release_requested", "Release import request accepted", map[string]interface{}{
			"import_id":             item.ID.String(),
			"release_state":         string(releaseProjection.State),
			"destination_image_ref": destinationImageRef,
		})
	}
	releaseReason := "destination=" + destinationImageRef
	if destinationRegistryAuthID != nil {
		releaseReason += ";destination_registry_auth_id=" + destinationRegistryAuthID.String()
	}
	releaseEventExtra := map[string]interface{}{
		"destination_image_ref": destinationImageRef,
	}
	if destinationRegistryAuthID != nil {
		releaseEventExtra["destination_registry_auth_id"] = destinationRegistryAuthID.String()
	}
	h.publishReleaseEvent(r.Context(), authCtx.UserID, item, messaging.EventTypeQuarantineReleaseRequested, "", releaseEventExtra)
	promotionMeta, promotionErr := h.promoteReleasedArtifact(r.Context(), item, destinationImageRef, destinationRegistryAuthID)
	if promotionErr != nil {
		errorMessage := "release promotion failed: " + promotionErr.Error()
		h.publishReleaseEvent(r.Context(), authCtx.UserID, item, messaging.EventTypeQuarantineReleaseFailed, errorMessage, releaseEventExtra)
		h.logger.Error("Failed to promote released artifact",
			zap.Error(promotionErr),
			zap.String("import_id", item.ID.String()),
			zap.String("tenant_id", item.TenantID.String()),
			zap.String("source_image_ref", strings.TrimSpace(item.InternalImageRef)),
			zap.String("destination_image_ref", destinationImageRef),
		)
		writeImageImportError(w, http.StatusBadGateway, "internal_error", "failed to promote image to destination registry", map[string]interface{}{
			"import_id":              item.ID.String(),
			"destination_image_ref":  destinationImageRef,
			"release_blocker_reason": "promotion_failed",
		})
		return
	}
	for key, value := range promotionMeta {
		releaseEventExtra[key] = value
	}

	releasedItem, err := h.service.MarkImportReleased(r.Context(), authCtx.TenantID, item.ID, authCtx.UserID, releaseReason)
	if err != nil {
		switch {
		case errors.Is(err, imageimport.ErrImportNotFound):
			writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		case errors.Is(err, imageimport.ErrReleaseNotEligible):
			eligibilityProjection = imageimport.DeriveReleaseProjection(item)
			writeImageImportError(w, http.StatusConflict, "release_not_eligible", "import request is not eligible for release", map[string]interface{}{
				"import_id":              item.ID.String(),
				"import_status":          string(item.Status),
				"release_state":          string(eligibilityProjection.State),
				"release_blocker_reason": eligibilityProjection.BlockerReason,
			})
		default:
			h.logger.Error("Failed to persist release state", zap.Error(err), zap.String("import_id", item.ID.String()))
			writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to persist release state", nil)
		}
		return
	}
	h.publishReleaseEvent(r.Context(), authCtx.UserID, releasedItem, messaging.EventTypeQuarantineReleased, "", releaseEventExtra)
	if h.audit != nil {
		_ = h.audit.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventAPICall, "image_import", "released", "Import release completed", map[string]interface{}{
			"import_id":             releasedItem.ID.String(),
			"release_state":         string(releasedItem.ReleaseState),
			"destination_image_ref": destinationImageRef,
		})
	}

	response := map[string]interface{}{
		"data": mapImportResponse(releasedItem, nil, nil),
		"release": map[string]interface{}{
			"status":                "completed",
			"target_state":          string(imageimport.ReleaseStateReleased),
			"destination_image_ref": destinationImageRef,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *ImageImportHandler) publishReleaseEvent(ctx context.Context, actorID uuid.UUID, req *imageimport.ImportRequest, eventType, message string, extra map[string]interface{}) {
	if h == nil || req == nil || strings.TrimSpace(eventType) == "" {
		return
	}
	if h.releases != nil {
		h.releases.Record(eventType)
	}
	if h.eventBus == nil {
		return
	}

	releaseProjection := imageimport.ResolveReleaseProjection(req)
	sourceImageDigest := strings.TrimSpace(req.SourceImageDigest)
	if sourceImageDigest == "" {
		sourceImageDigest = "unknown"
	}

	payload := map[string]interface{}{
		"external_image_import_id": req.ID.String(),
		"tenant_id":                req.TenantID.String(),
		"actor_id":                 actorID.String(),
		"release_state":            string(releaseProjection.State),
		"release_eligible":         releaseProjection.Eligible,
		"release_blocker_reason":   releaseProjection.BlockerReason,
		"status":                   string(req.Status),
		"source_image_digest":      sourceImageDigest,
		"idempotency_key":          req.ID.String() + ":" + strings.ReplaceAll(strings.TrimSpace(eventType), ".", "_"),
	}
	if trimmed := strings.TrimSpace(message); trimmed != "" {
		payload["message"] = trimmed
	}
	for key, value := range extra {
		payload[key] = value
	}

	if err := h.eventBus.Publish(ctx, messaging.Event{
		Type:          eventType,
		TenantID:      req.TenantID.String(),
		ActorID:       actorID.String(),
		Source:        "image-import.release",
		OccurredAt:    time.Now().UTC(),
		SchemaVersion: "1.0",
		Payload:       payload,
	}); err != nil {
		h.logger.Warn("Failed to publish quarantine release event", zap.String("event_type", eventType), zap.String("import_id", req.ID.String()), zap.Error(err))
	}
	h.publishReleaseOperationalAlert(ctx, actorID, req, eventType)
}

func (h *ImageImportHandler) publishReleaseOperationalAlert(ctx context.Context, actorID uuid.UUID, req *imageimport.ImportRequest, releaseEventType string) {
	if h == nil || req == nil || h.eventBus == nil || h.releases == nil || h.config == nil || h.releaseAlerts == nil {
		return
	}
	switch releaseEventType {
	case messaging.EventTypeQuarantineReleaseRequested, messaging.EventTypeQuarantineReleased, messaging.EventTypeQuarantineReleaseFailed:
	default:
		return
	}

	policy, err := h.config.GetReleaseGovernancePolicyConfig(ctx, &req.TenantID)
	if err != nil {
		h.logger.Warn("Failed to load release governance policy for alert evaluation", zap.String("tenant_id", req.TenantID.String()), zap.Error(err))
		return
	}
	if policy == nil {
		return
	}

	snapshot := h.releases.Snapshot()
	transition, shouldEmit := h.releaseAlerts.RecordAndEvaluate(req.TenantID, releaseEventType, snapshot, *policy)
	if !shouldEmit || transition == nil {
		return
	}

	alertEventType := messaging.EventTypeQuarantineReleaseRecovered
	alertState := "healthy"
	previousState := "degraded"
	if transition.CurrentDegraded {
		alertEventType = messaging.EventTypeQuarantineReleaseAlert
		alertState = "degraded"
		previousState = "healthy"
	}

	payload := map[string]interface{}{
		"tenant_id":                      req.TenantID.String(),
		"external_image_import_id":       req.ID.String(),
		"actor_id":                       actorID.String(),
		"state":                          alertState,
		"previous_state":                 previousState,
		"failure_ratio":                  transition.FailureRatio,
		"failure_ratio_threshold":        transition.FailureRatioThreshold,
		"consecutive_failures":           transition.ConsecutiveFailures,
		"consecutive_failures_threshold": transition.FailureBurstThreshold,
		"minimum_samples":                transition.MinimumSamples,
		"release_requested":              transition.ReleaseMetricsSnapshot.Requested,
		"release_released":               transition.ReleaseMetricsSnapshot.Released,
		"release_failed":                 transition.ReleaseMetricsSnapshot.Failed,
		"release_total":                  transition.ReleaseMetricsSnapshot.Total,
		"breach_failure_ratio":           transition.BreachByFailureRatio,
		"breach_failure_burst":           transition.BreachByFailureBurst,
		"idempotency_key":                req.TenantID.String() + ":" + alertState + ":" + strconv.FormatInt(transition.ReleaseMetricsSnapshot.Total, 10),
	}
	if err := h.eventBus.Publish(ctx, messaging.Event{
		Type:          alertEventType,
		TenantID:      req.TenantID.String(),
		ActorID:       actorID.String(),
		Source:        "image-import.release-alerts",
		OccurredAt:    time.Now().UTC(),
		SchemaVersion: "1.0",
		Payload:       payload,
	}); err != nil {
		h.logger.Warn("Failed to publish release governance alert transition", zap.String("tenant_id", req.TenantID.String()), zap.String("state", alertState), zap.Error(err))
	}
}

func (h *ImageImportHandler) applyApprovalDecision(w http.ResponseWriter, r *http.Request, approved bool, reason string, enforceTenantMatch bool) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil {
		writeImageImportError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	if h.workflowRepo == nil {
		writeImageImportError(w, http.StatusInternalServerError, "workflow_not_configured", "workflow repository is not configured", nil)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeImageImportError(w, http.StatusBadRequest, "invalid_id", "import request id must be a valid uuid", nil)
		return
	}

	instance, steps, err := h.workflowRepo.GetInstanceWithStepsBySubject(r.Context(), "external_image_import", id)
	if err != nil {
		h.logger.Error("Failed to load import workflow instance", zap.Error(err), zap.String("import_id", id.String()))
		writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to load approval workflow", nil)
		return
	}
	if instance == nil {
		writeImageImportError(w, http.StatusNotFound, "not_found", "import request not found", nil)
		return
	}
	if enforceTenantMatch && instance.TenantID != nil && *instance.TenantID != authCtx.TenantID {
		writeImageImportError(w, http.StatusForbidden, "tenant_context_mismatch", "selected tenant does not match import request tenant", nil)
		return
	}

	var decisionStep *domainworkflow.Step
	for i := range steps {
		if steps[i].StepKey == approvalDecisionStepKey {
			decisionStep = &steps[i]
			break
		}
	}
	if decisionStep == nil {
		writeImageImportError(w, http.StatusConflict, "approval_step_missing", "approval decision step is not available", nil)
		return
	}

	switch decisionStep.Status {
	case domainworkflow.StepStatusSucceeded, domainworkflow.StepStatusFailed:
		writeImageImportError(w, http.StatusConflict, "approval_already_decided", "approval decision is already finalized", nil)
		return
	case domainworkflow.StepStatusPending, domainworkflow.StepStatusRunning:
		writeImageImportError(w, http.StatusConflict, "approval_decision_in_progress", "approval decision is already in progress", nil)
		return
	case domainworkflow.StepStatusBlocked:
	default:
		writeImageImportError(w, http.StatusConflict, "approval_step_invalid_state", "approval decision step is not in a decisionable state", nil)
		return
	}

	if decisionStep.Payload == nil {
		decisionStep.Payload = map[string]interface{}{}
	}
	reason = strings.TrimSpace(reason)
	decisionStep.Payload["approved"] = approved
	if approved {
		decisionStep.Payload["approval_status"] = "approved"
		decisionStep.Payload["approval_reason"] = ""
	} else {
		decisionStep.Payload["approval_status"] = "rejected"
		decisionStep.Payload["approval_reason"] = reason
	}
	decisionStep.Payload["approved_by_user_id"] = authCtx.UserID.String()
	decisionStep.Payload["approved_at"] = time.Now().UTC().Format(timeRFC3339)
	decisionStep.Status = domainworkflow.StepStatusPending
	decisionStep.LastError = nil
	decisionStep.StartedAt = nil
	decisionStep.CompletedAt = nil

	if err := h.workflowRepo.UpdateStep(r.Context(), decisionStep); err != nil {
		h.logger.Error("Failed to persist approval decision", zap.Error(err), zap.String("import_id", id.String()))
		writeImageImportError(w, http.StatusInternalServerError, "internal_error", "failed to persist approval decision", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"import_request_id": id.String(),
			"approved":          approved,
			"status":            "decision_queued",
		},
	})
}

func (h *ImageImportHandler) loadDecisionTimeline(ctx context.Context, importID uuid.UUID) *imageImportDecisionTimeline {
	if h == nil || h.workflowRepo == nil || importID == uuid.Nil {
		return nil
	}
	instance, steps, err := h.workflowRepo.GetInstanceWithStepsBySubject(ctx, "external_image_import", importID)
	if err != nil || instance == nil {
		return nil
	}
	for i := range steps {
		if steps[i].StepKey != approvalDecisionStepKey {
			continue
		}
		step := steps[i]
		timeline := &imageImportDecisionTimeline{
			WorkflowStepStatus: string(step.Status),
		}
		if step.Payload != nil {
			if status, ok := step.Payload["approval_status"].(string); ok {
				timeline.DecisionStatus = strings.TrimSpace(status)
			}
			if timeline.DecisionStatus == "" {
				if approved, ok := step.Payload["approved"].(bool); ok {
					if approved {
						timeline.DecisionStatus = "approved"
					} else {
						timeline.DecisionStatus = "rejected"
					}
				}
			}
			if reason, ok := step.Payload["approval_reason"].(string); ok {
				timeline.DecisionReason = strings.TrimSpace(reason)
			}
			if by, ok := step.Payload["approved_by_user_id"].(string); ok {
				timeline.DecidedByUserID = strings.TrimSpace(by)
			}
			if at, ok := step.Payload["approved_at"].(string); ok {
				timeline.DecidedAt = strings.TrimSpace(at)
			}
		}
		if timeline.DecisionStatus == "" && timeline.DecisionReason == "" && timeline.DecidedByUserID == "" && timeline.DecidedAt == "" && timeline.WorkflowStepStatus == "" {
			return nil
		}
		return timeline
	}
	return nil
}

func (h *ImageImportHandler) loadNotificationReconciliation(ctx context.Context, req *imageimport.ImportRequest, timeline *imageImportDecisionTimeline) *imageImportNotificationReconciliation {
	if h == nil || h.notificationRepo == nil || req == nil || req.ID == uuid.Nil || req.TenantID == uuid.Nil || timeline == nil {
		return nil
	}
	decisionStatus := strings.ToLower(strings.TrimSpace(timeline.DecisionStatus))
	if decisionStatus != "approved" && decisionStatus != "rejected" {
		return nil
	}

	decisionEventType := "external.image.import." + decisionStatus
	idempotencyKey := req.ID.String() + ":" + decisionStatus
	notificationType := "external_image_import_" + decisionStatus

	adminUserIDs, err := h.notificationRepo.ListTenantAdminUserIDs(ctx, req.TenantID)
	if err != nil {
		return nil
	}
	recipientSet := make(map[uuid.UUID]struct{}, len(adminUserIDs)+1)
	for _, userID := range adminUserIDs {
		if userID != uuid.Nil {
			recipientSet[userID] = struct{}{}
		}
	}
	if req.RequestedByUserID != uuid.Nil {
		recipientSet[req.RequestedByUserID] = struct{}{}
	}
	expectedRecipients := len(recipientSet)
	if expectedRecipients == 0 {
		return nil
	}

	receiptCount, err := h.notificationRepo.CountImageImportNotificationReceipts(ctx, req.TenantID, decisionEventType, idempotencyKey)
	if err != nil {
		return nil
	}
	inAppCount, err := h.notificationRepo.CountImageImportInAppNotifications(ctx, req.TenantID, req.ID, notificationType)
	if err != nil {
		return nil
	}

	deliveryState := "pending"
	if receiptCount >= expectedRecipients && inAppCount >= expectedRecipients {
		deliveryState = "delivered"
	} else if receiptCount > 0 || inAppCount > 0 {
		deliveryState = "partial"
	}

	return &imageImportNotificationReconciliation{
		DecisionEventType:      decisionEventType,
		IdempotencyKey:         idempotencyKey,
		ExpectedRecipients:     expectedRecipients,
		ReceiptCount:           receiptCount,
		InAppNotificationCount: inAppCount,
		DeliveryState:          deliveryState,
	}
}

func mapImportResponse(req *imageimport.ImportRequest, decisionTimeline *imageImportDecisionTimeline, notificationReconciliation *imageImportNotificationReconciliation) imageImportResponse {
	var registryAuthID *string
	if req.RegistryAuthID != nil {
		value := req.RegistryAuthID.String()
		registryAuthID = &value
	}

	syncState, retryable := deriveImportSyncState(req)
	releaseProjection := imageimport.ResolveReleaseProjection(req)
	executionState, executionStateUpdatedAt, dispatchQueuedAt, pipelineStartedAt, evidenceReadyAt, releaseReadyAt :=
		deriveImportExecutionContract(req, releaseProjection, syncState)
	failureClass, failureCode := deriveImportFailureClassification(req, syncState)
	policyDecision := strings.TrimSpace(req.PolicyDecision)
	policyReasonsJSON := strings.TrimSpace(req.PolicyReasonsJSON)
	policySnapshotJSON := strings.TrimSpace(req.PolicySnapshotJSON)
	scanSummaryJSON := strings.TrimSpace(req.ScanSummaryJSON)
	sbomSummaryJSON := strings.TrimSpace(req.SBOMSummaryJSON)
	sbomEvidenceJSON := strings.TrimSpace(req.SBOMEvidenceJSON)
	sourceImageDigest := strings.TrimSpace(req.SourceImageDigest)

	if isTerminalImportStatus(req.Status) {
		if policyDecision == "" {
			switch req.Status {
			case imageimport.StatusSuccess:
				policyDecision = "pass"
			case imageimport.StatusQuarantined:
				policyDecision = "quarantine"
			default:
				policyDecision = "unknown"
			}
		}
		if policyReasonsJSON == "" {
			policyReasonsJSON = "{}"
		}
		if policySnapshotJSON == "" {
			policySnapshotJSON = "{}"
		}
		if scanSummaryJSON == "" {
			scanSummaryJSON = "{}"
		}
		if sbomSummaryJSON == "" {
			sbomSummaryJSON = "{}"
		}
		if sbomEvidenceJSON == "" {
			sbomEvidenceJSON = "{}"
		}
		if sourceImageDigest == "" {
			sourceImageDigest = "unknown"
		}
	}

	return imageImportResponse{
		ID:                         req.ID.String(),
		TenantID:                   req.TenantID.String(),
		RequestedByUserID:          req.RequestedByUserID.String(),
		RequestType:                string(req.RequestType),
		EPRRecordID:                req.SORRecordID,
		SourceRegistry:             req.SourceRegistry,
		SourceImageRef:             req.SourceImageRef,
		RegistryAuthID:             registryAuthID,
		Status:                     string(req.Status),
		ErrorMessage:               req.ErrorMessage,
		InternalImageRef:           req.InternalImageRef,
		PipelineRunName:            req.PipelineRunName,
		PipelineNamespace:          req.PipelineNamespace,
		PolicyDecision:             policyDecision,
		PolicyReasonsJSON:          policyReasonsJSON,
		PolicySnapshotJSON:         policySnapshotJSON,
		ScanSummaryJSON:            scanSummaryJSON,
		SBOMSummaryJSON:            sbomSummaryJSON,
		SBOMEvidenceJSON:           sbomEvidenceJSON,
		SourceImageDigest:          sourceImageDigest,
		DecisionTimeline:           decisionTimeline,
		NotificationReconciliation: notificationReconciliation,
		SyncState:                  syncState,
		ExecutionState:             executionState,
		ExecutionStateUpdatedAt:    executionStateUpdatedAt,
		DispatchQueuedAt:           dispatchQueuedAt,
		PipelineStartedAt:          pipelineStartedAt,
		EvidenceReadyAt:            evidenceReadyAt,
		ReleaseReadyAt:             releaseReadyAt,
		FailureClass:               failureClass,
		FailureCode:                failureCode,
		Retryable:                  retryable,
		ReleaseState:               string(releaseProjection.State),
		ReleaseEligible:            releaseProjection.Eligible,
		ReleaseBlockerReason:       releaseProjection.BlockerReason,
		ReleaseReason:              strings.TrimSpace(req.ReleaseReason),
		CreatedAt:                  req.CreatedAt.UTC().Format(timeRFC3339),
		UpdatedAt:                  req.UpdatedAt.UTC().Format(timeRFC3339),
	}
}

func isValidDestinationImageRef(ref string) bool {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return false
	}
	if strings.ContainsAny(trimmed, " \t\r\n") {
		return false
	}
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 {
		return false
	}
	host := strings.TrimSpace(parts[0])
	remainder := strings.TrimSpace(parts[1])
	if host == "" || remainder == "" {
		return false
	}
	if !(strings.Contains(host, ".") || strings.Contains(host, ":") || host == "localhost") {
		return false
	}
	return true
}

func (h *ImageImportHandler) promoteReleasedArtifact(
	ctx context.Context,
	item *imageimport.ImportRequest,
	destinationImageRef string,
	destinationRegistryAuthID *uuid.UUID,
) (map[string]interface{}, error) {
	if h == nil || item == nil {
		return nil, fmt.Errorf("import request is required")
	}
	if h.infraService == nil {
		return nil, nil
	}
	sourceImageRef := strings.TrimSpace(item.InternalImageRef)
	if sourceImageRef == "" {
		return nil, fmt.Errorf("internal image reference is required for promotion")
	}
	if destinationRegistryAuthID == nil || *destinationRegistryAuthID == uuid.Nil {
		return nil, fmt.Errorf("destination registry auth id is required for promotion")
	}
	namespace := strings.TrimSpace(item.PipelineNamespace)
	if namespace == "" {
		namespace = "default"
	}

	providers, err := h.infraService.GetAvailableProviders(ctx, item.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list available providers: %w", err)
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("no available providers found for tenant")
	}

	type promotionClient struct {
		providerID   string
		k8sClient    k8sclient.Interface
		pipelineMgr  build.PipelineManager
		tektonClient tektonclient.Interface
	}
	clients := make([]promotionClient, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		restCfg, cfgErr := connectors.BuildRESTConfigFromProviderConfig(provider.Config)
		if cfgErr != nil {
			continue
		}
		k8sClient, kErr := k8sclient.NewForConfig(restCfg)
		tekClient, tErr := tektonclient.NewForConfig(restCfg)
		if kErr != nil || tErr != nil {
			continue
		}
		clients = append(clients, promotionClient{
			providerID:   provider.ID.String(),
			k8sClient:    k8sClient,
			pipelineMgr:  k8sinfra.NewKubernetesPipelineManager(k8sClient, tekClient, h.logger),
			tektonClient: tekClient,
		})
	}
	if len(clients) == 0 {
		return nil, fmt.Errorf("no tekton clients available for promotion")
	}

	selected := clients[0]
	if run := strings.TrimSpace(item.PipelineRunName); run != "" {
		for _, candidate := range clients {
			if _, getErr := candidate.tektonClient.TektonV1().PipelineRuns(namespace).Get(ctx, run, metav1.GetOptions{}); getErr == nil {
				selected = candidate
				break
			}
		}
	}

	pipelineName := strings.TrimSpace(os.Getenv("IF_QUARANTINE_RELEASE_PROMOTION_PIPELINE_NAME"))
	if pipelineName == "" {
		pipelineName = defaultQuarantineReleasePromotionPipelineName
	}
	dockerConfigSecret := strings.TrimSpace(os.Getenv("IF_QUARANTINE_IMPORT_DOCKERCONFIG_SECRET"))
	if dockerConfigSecret == "" {
		dockerConfigSecret = defaultQuarantineReleaseDockerConfigSecret
	}
	if h.registryAuthService == nil {
		return nil, fmt.Errorf("registry auth service is not configured")
	}
	auth, authErr := h.registryAuthService.GetByID(ctx, *destinationRegistryAuthID)
	if authErr != nil {
		return nil, fmt.Errorf("failed to load destination registry auth: %w", authErr)
	}
	if auth == nil {
		return nil, fmt.Errorf("destination registry auth not found")
	}
	if auth.TenantID != item.TenantID {
		return nil, fmt.Errorf("destination registry auth belongs to a different tenant")
	}
	if !auth.IsActive {
		return nil, fmt.Errorf("destination registry auth is inactive")
	}
	dockerConfigJSON, cfgErr := h.registryAuthService.ResolveDockerConfigJSON(ctx, *destinationRegistryAuthID)
	if cfgErr != nil {
		return nil, fmt.Errorf("failed to resolve destination docker config json: %w", cfgErr)
	}
	tenantSecretName := releasePromotionSecretName(*destinationRegistryAuthID)
	if reconcileErr := reconcileDockerConfigSecretForPromotion(ctx, selected.k8sClient, namespace, tenantSecretName, dockerConfigJSON); reconcileErr != nil {
		return nil, fmt.Errorf("failed to reconcile destination registry auth secret: %w", reconcileErr)
	}
	dockerConfigSecret = tenantSecretName

	pipelineRunYAML := renderReleasePromotionPipelineRunYAML(
		item,
		pipelineName,
		sourceImageRef,
		destinationImageRef,
		dockerConfigSecret,
	)
	created, createErr := selected.pipelineMgr.CreatePipelineRun(ctx, namespace, pipelineRunYAML)
	if createErr != nil {
		return nil, fmt.Errorf("failed to create release promotion pipeline run: %w", createErr)
	}

	runName := strings.TrimSpace(created.Name)
	timeout := releasePromotionTimeout()
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	succeeded, failureMessage, waitErr := waitForPromotionPipelineRun(waitCtx, selected.tektonClient, namespace, runName)
	if waitErr != nil {
		return nil, waitErr
	}
	if !succeeded {
		if strings.TrimSpace(failureMessage) == "" {
			failureMessage = "promotion pipeline failed"
		}
		return nil, errors.New(failureMessage)
	}

	return map[string]interface{}{
		"promotion_pipeline_run": runName,
		"promotion_namespace":    namespace,
		"promotion_provider_id":  selected.providerID,
		"promotion_secret_name":  dockerConfigSecret,
	}, nil
}

func renderReleasePromotionPipelineRunYAML(
	item *imageimport.ImportRequest,
	pipelineName, sourceImageRef, destinationImageRef, dockerConfigSecret string,
) string {
	importID := ""
	if item != nil {
		importID = item.ID.String()
	}
	return fmt.Sprintf(`apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: quarantine-release-promote-
  labels:
    app.kubernetes.io/part-of: image-factory
    image-factory.io/workflow-subject: external-image-import-release
    image-factory.io/external-image-import-id: "%s"
spec:
  pipelineRef:
    name: %s
  params:
    - name: source-image-ref
      value: "%s"
    - name: destination-image-ref
      value: "%s"
  workspaces:
    - name: source
      volumeClaimTemplate:
        spec:
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 2Gi
    - name: dockerconfig
      secret:
        secretName: %s
`, importID, pipelineName, sourceImageRef, destinationImageRef, dockerConfigSecret)
}

func waitForPromotionPipelineRun(
	ctx context.Context,
	tektonClient tektonclient.Interface,
	namespace, pipelineRunName string,
) (bool, string, error) {
	if tektonClient == nil {
		return false, "", fmt.Errorf("tekton client is required")
	}
	if strings.TrimSpace(namespace) == "" || strings.TrimSpace(pipelineRunName) == "" {
		return false, "", fmt.Errorf("pipeline namespace and run name are required")
	}
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		run, err := tektonClient.TektonV1().PipelineRuns(namespace).Get(ctx, pipelineRunName, metav1.GetOptions{})
		if err == nil && run != nil {
			terminal, succeeded, message := promotionPipelineRunOutcome(run)
			if terminal {
				return succeeded, strings.TrimSpace(message), nil
			}
		}
		select {
		case <-ctx.Done():
			return false, "", fmt.Errorf("timed out waiting for promotion pipeline run %s/%s", namespace, pipelineRunName)
		case <-ticker.C:
		}
	}
}

func promotionPipelineRunOutcome(run *tektonv1.PipelineRun) (terminal bool, succeeded bool, message string) {
	if run == nil {
		return false, false, ""
	}
	for _, condition := range run.Status.Conditions {
		if condition.Type != "Succeeded" {
			continue
		}
		msg := strings.TrimSpace(condition.Message)
		if msg == "" {
			msg = strings.TrimSpace(condition.Reason)
		}
		if condition.Status == "True" {
			return true, true, msg
		}
		if condition.Status == "False" {
			return true, false, msg
		}
	}
	return false, false, ""
}

func releasePromotionTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("IF_QUARANTINE_RELEASE_PROMOTION_TIMEOUT_SECONDS"))
	if raw == "" {
		return 5 * time.Minute
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(seconds) * time.Second
}

func releasePromotionSecretName(registryAuthID uuid.UUID) string {
	token := strings.ReplaceAll(strings.ToLower(registryAuthID.String()), "-", "")
	if len(token) > 12 {
		token = token[:12]
	}
	if token == "" {
		token = "default"
	}
	return "release-regcred-" + token
}

func reconcileDockerConfigSecretForPromotion(
	ctx context.Context,
	k8s k8sclient.Interface,
	namespace, secretName string,
	dockerConfigJSON []byte,
) error {
	if k8s == nil {
		return fmt.Errorf("kubernetes client is required")
	}
	if strings.TrimSpace(namespace) == "" || strings.TrimSpace(secretName) == "" {
		return fmt.Errorf("namespace and secret name are required")
	}
	if len(dockerConfigJSON) == 0 {
		return fmt.Errorf("docker config json is required")
	}
	desired := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJSON,
		},
	}
	current, err := k8s.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, createErr := k8s.CoreV1().Secrets(namespace).Create(ctx, desired, metav1.CreateOptions{})
			return createErr
		}
		return err
	}
	current.Type = corev1.SecretTypeDockerConfigJson
	if current.Data == nil {
		current.Data = map[string][]byte{}
	}
	current.Data[corev1.DockerConfigJsonKey] = dockerConfigJSON
	_, updateErr := k8s.CoreV1().Secrets(namespace).Update(ctx, current, metav1.UpdateOptions{})
	return updateErr
}

func mapReleasedArtifactResponse(item *imageimport.ReleasedArtifact) releasedArtifactResponse {
	consumptionReady := item.ReleaseState == imageimport.ReleaseStateReleased && strings.TrimSpace(item.InternalImageRef) != ""
	consumptionBlocker := ""
	if !consumptionReady {
		if item.ReleaseState != imageimport.ReleaseStateReleased {
			consumptionBlocker = "artifact is not in released state"
		} else {
			consumptionBlocker = "internal image reference is not available yet"
		}
	}
	response := releasedArtifactResponse{
		ID:                 item.ID.String(),
		TenantID:           item.TenantID.String(),
		RequestedByUserID:  item.RequestedByUserID.String(),
		EPRRecordID:        item.SORRecordID,
		SourceRegistry:     item.SourceRegistry,
		SourceImageRef:     item.SourceImageRef,
		InternalImageRef:   item.InternalImageRef,
		SourceImageDigest:  item.SourceImageDigest,
		PolicyDecision:     item.PolicyDecision,
		PolicySnapshotJSON: item.PolicySnapshotJSON,
		ReleaseState:       string(item.ReleaseState),
		ReleaseReason:      item.ReleaseReason,
		ConsumptionReady:   consumptionReady,
		ConsumptionBlocker: consumptionBlocker,
		CreatedAt:          item.CreatedAt.UTC().Format(timeRFC3339),
		UpdatedAt:          item.UpdatedAt.UTC().Format(timeRFC3339),
	}
	if item.ReleaseActorUserID != nil && *item.ReleaseActorUserID != uuid.Nil {
		value := item.ReleaseActorUserID.String()
		response.ReleaseActorUserID = &value
	}
	if item.ReleaseRequestedAt != nil {
		response.ReleaseRequestedAt = item.ReleaseRequestedAt.UTC().Format(timeRFC3339)
	}
	if item.ReleasedAt != nil {
		response.ReleasedAt = item.ReleasedAt.UTC().Format(timeRFC3339)
	}
	return response
}

func isTerminalImportStatus(status imageimport.Status) bool {
	return status == imageimport.StatusSuccess || status == imageimport.StatusQuarantined || status == imageimport.StatusFailed
}

func (h *ImageImportHandler) resolveRequestType(raw string) imageimport.RequestType {
	if h == nil {
		return imageimport.RequestTypeQuarantine
	}
	candidate := imageimport.RequestType(strings.TrimSpace(strings.ToLower(raw)))
	switch candidate {
	case imageimport.RequestTypeQuarantine, imageimport.RequestTypeScan:
		return candidate
	default:
		return h.defaultRequestType
	}
}

func deriveImportSyncState(req *imageimport.ImportRequest) (state string, retryable bool) {
	if req == nil {
		return "", false
	}
	errorMessage := strings.ToLower(strings.TrimSpace(req.ErrorMessage))
	switch req.Status {
	case imageimport.StatusPending:
		return "awaiting_approval", false
	case imageimport.StatusApproved:
		return "awaiting_dispatch", true
	case imageimport.StatusImporting:
		if strings.Contains(errorMessage, "catalog image is not ready for evidence sync") {
			return "catalog_sync_pending", true
		}
		return "pipeline_running", true
	case imageimport.StatusSuccess:
		return "completed", false
	case imageimport.StatusQuarantined:
		return "completed", true
	case imageimport.StatusFailed:
		if strings.HasPrefix(errorMessage, "dispatch_failed:") {
			return "dispatch_failed", true
		}
		return "failed", true
	default:
		return "", false
	}
}

func deriveImportExecutionContract(
	req *imageimport.ImportRequest,
	releaseProjection imageimport.ReleaseProjection,
	syncState string,
) (state, stateUpdatedAt, dispatchQueuedAt, pipelineStartedAt, evidenceReadyAt, releaseReadyAt string) {
	if req == nil {
		return "", "", "", "", "", ""
	}

	dispatchQueuedAt = req.CreatedAt.UTC().Format(timeRFC3339)
	stateUpdatedAt = req.UpdatedAt.UTC().Format(timeRFC3339)

	switch req.Status {
	case imageimport.StatusPending:
		state = "awaiting_approval"
		dispatchQueuedAt = ""
	case imageimport.StatusApproved:
		state = "awaiting_dispatch"
	case imageimport.StatusImporting:
		if syncState == "catalog_sync_pending" {
			state = "evidence_pending"
		} else {
			state = "pipeline_running"
		}
		pipelineStartedAt = stateUpdatedAt
	case imageimport.StatusSuccess:
		pipelineStartedAt = stateUpdatedAt
		evidenceReadyAt = stateUpdatedAt
		if releaseProjection.State == imageimport.ReleaseStateReadyForRelease ||
			releaseProjection.State == imageimport.ReleaseStateReleaseApproved ||
			releaseProjection.State == imageimport.ReleaseStateReleased {
			state = "ready_for_release"
			releaseReadyAt = stateUpdatedAt
		} else {
			state = "completed"
		}
	case imageimport.StatusQuarantined, imageimport.StatusFailed:
		state = "completed"
		pipelineStartedAt = stateUpdatedAt
		evidenceReadyAt = stateUpdatedAt
	default:
		state = ""
	}

	return state, stateUpdatedAt, dispatchQueuedAt, pipelineStartedAt, evidenceReadyAt, releaseReadyAt
}

func deriveImportFailureClassification(req *imageimport.ImportRequest, syncState string) (failureClass, failureCode string) {
	if req == nil {
		return "", ""
	}

	message := strings.TrimSpace(strings.ToLower(req.ErrorMessage))
	if message == "" && req.Status != imageimport.StatusFailed {
		return "", ""
	}

	if req.Status == imageimport.StatusQuarantined {
		return "policy", "quarantined_by_policy"
	}
	if req.Status != imageimport.StatusFailed {
		return "", ""
	}

	if syncState == "dispatch_failed" || strings.HasPrefix(message, "dispatch_failed:") {
		switch {
		case strings.Contains(message, "deadline exceeded"), strings.Contains(message, "timeout"):
			return "dispatch", "dispatch_timeout"
		case strings.Contains(message, "no tekton-enabled quarantine dispatcher available"),
			strings.Contains(message, "waiting_for_dispatch"):
			return "dispatch", "dispatcher_unavailable"
		default:
			return "dispatch", "dispatch_error"
		}
	}

	switch {
	case strings.Contains(message, "forbidden"), strings.Contains(message, "unauthorized"), strings.Contains(message, "authentication"):
		return "auth", "auth_error"
	case strings.Contains(message, "no such host"), strings.Contains(message, "connection refused"), strings.Contains(message, "i/o timeout"), strings.Contains(message, "dial tcp"):
		return "connectivity", "connectivity_error"
	case strings.Contains(message, "policy"), strings.Contains(message, "quarantine"):
		return "policy", "policy_blocked"
	default:
		return "runtime", "runtime_failed"
	}
}

func writeImageImportError(w http.ResponseWriter, status int, code, message string, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"details": details,
		},
	})
}

func (h *ImageImportHandler) recordDenial(
	r *http.Request,
	tenantID uuid.UUID,
	userID uuid.UUID,
	capabilityKey string,
	reason string,
	eventType audit.AuditEventType,
	message string,
	labels map[string]string,
) {
	if h.denials != nil {
		h.denials.RecordDeniedWithLabels(tenantID, capabilityKey, reason, labels)
	}
	if h.audit != nil {
		data := map[string]interface{}{
			"tenant_id":      tenantID.String(),
			"capability_key": capabilityKey,
			"reason":         reason,
		}
		for key, value := range labels {
			data[key] = value
		}
		_ = h.audit.LogUserAction(r.Context(), tenantID, userID, eventType, "image_import", "deny", message, data)
	}
}

func (h *ImageImportHandler) eprDenialLabels(ctx context.Context, tenantID uuid.UUID) map[string]string {
	labels := map[string]string{
		"epr_runtime_mode": "unknown",
		"epr_policy_scope": "unknown",
	}
	if h.config == nil || tenantID == uuid.Nil {
		return labels
	}

	cfg, err := h.config.GetSORRegistrationConfig(ctx, &tenantID)
	if err != nil {
		h.logger.Warn("Failed to resolve EPR registration config for denial labels", zap.String("tenant_id", tenantID.String()), zap.Error(err))
		return labels
	}
	mode := strings.ToLower(strings.TrimSpace(cfg.RuntimeErrorMode))
	if mode == "" {
		mode = "error"
	}
	labels["epr_runtime_mode"] = mode

	scope := "global"
	_, err = h.config.GetConfigByKey(ctx, &tenantID, "sor_registration")
	switch {
	case err == nil:
		scope = "tenant"
	case errors.Is(err, systemconfig.ErrConfigNotFound):
		scope = "global"
	default:
		scope = "unknown"
		h.logger.Warn("Failed to resolve EPR policy scope for denial labels", zap.String("tenant_id", tenantID.String()), zap.Error(err))
	}
	labels["epr_policy_scope"] = scope

	return labels
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func parsePositiveInt(raw string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
