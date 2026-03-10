package steps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	appworkflow "github.com/srikarm/image-factory/internal/application/workflow"
	domainimageimport "github.com/srikarm/image-factory/internal/domain/imageimport"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"go.uber.org/zap"
)

const (
	StepApprovalRequest  = "approval.request"
	StepApprovalDecision = "approval.decision"
	StepImportDispatch   = "import.dispatch"
	StepImportMonitor    = "import.monitor"
)

type WorkflowRepository interface {
	UpdateStepStatus(ctx context.Context, instanceID uuid.UUID, stepKey string, status domainworkflow.StepStatus, errMsg *string) error
}

type QuarantineDispatcher interface {
	Dispatch(ctx context.Context, req *domainimageimport.ImportRequest) (DispatchResult, error)
}

type QuarantineRunReader interface {
	GetPipelineRun(ctx context.Context, req *domainimageimport.ImportRequest) (*tektonv1.PipelineRun, error)
}

type QuarantinePolicy struct {
	Mode        string            `json:"mode"`
	MaxCritical int               `json:"max_critical"`
	MaxP2       int               `json:"max_p2"`
	MaxP3       int               `json:"max_p3"`
	MaxCVSS     float64           `json:"max_cvss"`
	Thresholds  map[string]int    `json:"thresholds,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type QuarantinePolicyProvider interface {
	Resolve(ctx context.Context, tenantID uuid.UUID) (*QuarantinePolicy, error)
}

type QuarantinePolicyProviderFunc func(ctx context.Context, tenantID uuid.UUID) (*QuarantinePolicy, error)

func (f QuarantinePolicyProviderFunc) Resolve(ctx context.Context, tenantID uuid.UUID) (*QuarantinePolicy, error) {
	return f(ctx, tenantID)
}

type DispatchResult struct {
	PipelineRunName  string
	Namespace        string
	InternalImageRef string
}

type approvalRequestHandler struct {
	importRepo   domainimageimport.Repository
	workflowRepo WorkflowRepository
	eventBus     messaging.EventBus
	logger       *zap.Logger
}

func NewApprovalRequestHandler(importRepo domainimageimport.Repository, workflowRepo WorkflowRepository, logger *zap.Logger) appworkflow.StepHandler {
	return NewApprovalRequestHandlerWithEvents(importRepo, workflowRepo, nil, logger)
}

func NewApprovalRequestHandlerWithEvents(importRepo domainimageimport.Repository, workflowRepo WorkflowRepository, eventBus messaging.EventBus, logger *zap.Logger) appworkflow.StepHandler {
	return &approvalRequestHandler{
		importRepo:   importRepo,
		workflowRepo: workflowRepo,
		eventBus:     eventBus,
		logger:       logger,
	}
}

func (h *approvalRequestHandler) Key() string { return StepApprovalRequest }

func (h *approvalRequestHandler) Execute(ctx context.Context, step *domainworkflow.Step) (appworkflow.StepResult, error) {
	req, err := h.resolveImportRequest(ctx, step)
	if err != nil {
		return stepFailure(err), nil
	}
	if h.logger != nil {
		h.logger.Info("External image import approval requested",
			zap.String("import_request_id", req.ID.String()),
			zap.String("tenant_id", req.TenantID.String()),
		)
	}
	h.publishImportEvent(ctx, messaging.EventTypeExternalImageImportApprovalRequested, req, "")
	return appworkflow.StepResult{
		Status: domainworkflow.StepStatusSucceeded,
		Data: map[string]interface{}{
			"import_request_id": req.ID.String(),
		},
	}, nil
}

type approvalDecisionHandler struct {
	importRepo   domainimageimport.Repository
	workflowRepo WorkflowRepository
	eventBus     messaging.EventBus
	logger       *zap.Logger
}

func NewApprovalDecisionHandler(importRepo domainimageimport.Repository, workflowRepo WorkflowRepository, logger *zap.Logger) appworkflow.StepHandler {
	return NewApprovalDecisionHandlerWithEvents(importRepo, workflowRepo, nil, logger)
}

func NewApprovalDecisionHandlerWithEvents(importRepo domainimageimport.Repository, workflowRepo WorkflowRepository, eventBus messaging.EventBus, logger *zap.Logger) appworkflow.StepHandler {
	return &approvalDecisionHandler{
		importRepo:   importRepo,
		workflowRepo: workflowRepo,
		eventBus:     eventBus,
		logger:       logger,
	}
}

func (h *approvalDecisionHandler) Key() string { return StepApprovalDecision }

func (h *approvalDecisionHandler) Execute(ctx context.Context, step *domainworkflow.Step) (appworkflow.StepResult, error) {
	req, err := h.resolveImportRequest(ctx, step)
	if err != nil {
		return stepFailure(err), nil
	}

	approved, reason, decided := parseApprovalDecision(step.Payload)
	if !decided {
		return stepFailure(errors.New("approval decision is required")), nil
	}
	if approved {
		if !canTransitionImportStatus(req.Status, domainimageimport.StatusApproved) {
			return stepFailure(fmt.Errorf("invalid import status transition: %s -> %s", req.Status, domainimageimport.StatusApproved)), nil
		}
		if err := h.importRepo.UpdateStatus(ctx, req.TenantID, req.ID, domainimageimport.StatusApproved, "", req.InternalImageRef); err != nil {
			return stepFailure(fmt.Errorf("failed to mark import approved: %w", err)), nil
		}
		req.Status = domainimageimport.StatusApproved
		req.ErrorMessage = ""
		if h.workflowRepo != nil {
			if err := h.workflowRepo.UpdateStepStatus(ctx, step.InstanceID, StepImportDispatch, domainworkflow.StepStatusPending, nil); err != nil {
				return stepFailure(fmt.Errorf("failed to unblock import.dispatch: %w", err)), nil
			}
		}
		h.publishImportEvent(ctx, messaging.EventTypeExternalImageImportApproved, req, "")
		return appworkflow.StepResult{
			Status: domainworkflow.StepStatusSucceeded,
			Data: map[string]interface{}{
				"decision": "approved",
			},
		}, nil
	}

	if reason == "" {
		reason = "external image import request was rejected"
	}
	if !canTransitionImportStatus(req.Status, domainimageimport.StatusFailed) {
		return stepFailure(fmt.Errorf("invalid import status transition: %s -> %s", req.Status, domainimageimport.StatusFailed)), nil
	}
	if err := h.importRepo.UpdateStatus(ctx, req.TenantID, req.ID, domainimageimport.StatusFailed, reason, req.InternalImageRef); err != nil {
		return stepFailure(fmt.Errorf("failed to mark import rejected: %w", err)), nil
	}
	req.Status = domainimageimport.StatusFailed
	req.ErrorMessage = reason
	if h.logger != nil {
		h.logger.Warn("External image import rejected by approval decision",
			zap.String("import_request_id", req.ID.String()),
			zap.String("tenant_id", req.TenantID.String()),
			zap.String("reason", reason),
		)
	}
	h.publishImportEvent(ctx, messaging.EventTypeExternalImageImportRejected, req, reason)
	return stepFailure(errors.New(reason)), nil
}

type importDispatchHandler struct {
	importRepo   domainimageimport.Repository
	dispatcher   QuarantineDispatcher
	workflowRepo WorkflowRepository
	policy       QuarantinePolicyProvider
	eventBus     messaging.EventBus
	logger       *zap.Logger
}

func NewImportDispatchHandler(importRepo domainimageimport.Repository, dispatcher QuarantineDispatcher, workflowRepo WorkflowRepository, logger *zap.Logger) appworkflow.StepHandler {
	return NewImportDispatchHandlerWithPolicyAndEvents(importRepo, dispatcher, workflowRepo, nil, nil, logger)
}

func NewImportDispatchHandlerWithPolicy(importRepo domainimageimport.Repository, dispatcher QuarantineDispatcher, workflowRepo WorkflowRepository, policy QuarantinePolicyProvider, logger *zap.Logger) appworkflow.StepHandler {
	return NewImportDispatchHandlerWithPolicyAndEvents(importRepo, dispatcher, workflowRepo, policy, nil, logger)
}

func NewImportDispatchHandlerWithPolicyAndEvents(importRepo domainimageimport.Repository, dispatcher QuarantineDispatcher, workflowRepo WorkflowRepository, policy QuarantinePolicyProvider, eventBus messaging.EventBus, logger *zap.Logger) appworkflow.StepHandler {
	return &importDispatchHandler{
		importRepo:   importRepo,
		dispatcher:   dispatcher,
		workflowRepo: workflowRepo,
		policy:       policy,
		eventBus:     eventBus,
		logger:       logger,
	}
}

func (h *importDispatchHandler) Key() string { return StepImportDispatch }

func (h *importDispatchHandler) Execute(ctx context.Context, step *domainworkflow.Step) (appworkflow.StepResult, error) {
	if h.dispatcher == nil {
		return appworkflow.StepResult{
			Status: domainworkflow.StepStatusBlocked,
			Error:  "waiting_for_dispatch: no tekton-enabled quarantine dispatcher available",
		}, nil
	}
	req, err := h.resolveImportRequest(ctx, step)
	if err != nil {
		return stepFailure(err), nil
	}

	if req.Status != domainimageimport.StatusApproved && req.Status != domainimageimport.StatusImporting {
		return stepFailure(fmt.Errorf("import is not dispatchable from status=%s", req.Status)), nil
	}
	if h.policy != nil {
		policyCfg, policyErr := h.policy.Resolve(ctx, req.TenantID)
		if policyErr != nil {
			return stepFailure(fmt.Errorf("failed to resolve quarantine policy: %w", policyErr)), nil
		}
		if policyCfg != nil {
			if encoded, marshalErr := json.Marshal(policyCfg); marshalErr == nil {
				req.PolicySnapshotJSON = strings.TrimSpace(string(encoded))
			}
		}
	}

	result, err := h.dispatcher.Dispatch(ctx, req)
	if err != nil {
		dispatchErr := strings.TrimSpace(err.Error())
		if strings.HasPrefix(dispatchErr, "waiting_for_dispatch:") {
			return appworkflow.StepResult{
				Status: domainworkflow.StepStatusBlocked,
				Error:  dispatchErr,
			}, nil
		}
		attempt := dispatchAttemptFromPayload(step.Payload)
		if dispatchErr == "" {
			dispatchErr = "unknown dispatch failure"
		}
		failureMessage := "dispatch_failed: " + dispatchErr
		if canTransitionImportStatus(req.Status, domainimageimport.StatusFailed) {
			_ = h.importRepo.UpdateStatus(ctx, req.TenantID, req.ID, domainimageimport.StatusFailed, failureMessage, req.InternalImageRef)
		}
		req.Status = domainimageimport.StatusFailed
		req.ErrorMessage = failureMessage
		h.publishDispatchFailedEvent(ctx, req, dispatchErr, attempt)
		return stepFailure(fmt.Errorf("failed to dispatch quarantine import: %w", err)), nil
	}

	internalRef := strings.TrimSpace(result.InternalImageRef)
	if internalRef == "" {
		internalRef = req.InternalImageRef
	}
	if !canTransitionImportStatus(req.Status, domainimageimport.StatusImporting) {
		return stepFailure(fmt.Errorf("invalid import status transition: %s -> %s", req.Status, domainimageimport.StatusImporting)), nil
	}
	if err := h.importRepo.UpdateStatus(ctx, req.TenantID, req.ID, domainimageimport.StatusImporting, "", internalRef); err != nil {
		return stepFailure(fmt.Errorf("failed to mark import as importing: %w", err)), nil
	}
	if err := h.importRepo.UpdatePipelineRefs(ctx, req.TenantID, req.ID, strings.TrimSpace(result.PipelineRunName), strings.TrimSpace(result.Namespace)); err != nil {
		return stepFailure(fmt.Errorf("failed to persist import pipeline refs: %w", err)), nil
	}
	if h.workflowRepo != nil {
		if err := h.workflowRepo.UpdateStepStatus(ctx, step.InstanceID, StepImportMonitor, domainworkflow.StepStatusPending, nil); err != nil {
			return stepFailure(fmt.Errorf("failed to unblock import.monitor: %w", err)), nil
		}
	}

	if h.logger != nil {
		h.logger.Info("External image import dispatched",
			zap.String("import_request_id", req.ID.String()),
			zap.String("tenant_id", req.TenantID.String()),
			zap.String("pipeline_run", result.PipelineRunName),
			zap.String("namespace", result.Namespace),
		)
	}

	return appworkflow.StepResult{
		Status: domainworkflow.StepStatusSucceeded,
		Data: map[string]interface{}{
			"pipeline_run_name":  result.PipelineRunName,
			"pipeline_namespace": result.Namespace,
			"internal_image_ref": internalRef,
		},
	}, nil
}

type importMonitorHandler struct {
	importRepo domainimageimport.Repository
	reader     QuarantineRunReader
	policy     QuarantinePolicyProvider
	eventBus   messaging.EventBus
	logger     *zap.Logger
}

func NewImportMonitorHandler(importRepo domainimageimport.Repository, reader QuarantineRunReader, logger *zap.Logger) appworkflow.StepHandler {
	return NewImportMonitorHandlerWithPolicyAndEvents(importRepo, reader, nil, nil, logger)
}

func NewImportMonitorHandlerWithPolicy(importRepo domainimageimport.Repository, reader QuarantineRunReader, policy QuarantinePolicyProvider, logger *zap.Logger) appworkflow.StepHandler {
	return NewImportMonitorHandlerWithPolicyAndEvents(importRepo, reader, policy, nil, logger)
}

func NewImportMonitorHandlerWithPolicyAndEvents(importRepo domainimageimport.Repository, reader QuarantineRunReader, policy QuarantinePolicyProvider, eventBus messaging.EventBus, logger *zap.Logger) appworkflow.StepHandler {
	return &importMonitorHandler{
		importRepo: importRepo,
		reader:     reader,
		policy:     policy,
		eventBus:   eventBus,
		logger:     logger,
	}
}

func (h *importMonitorHandler) Key() string { return StepImportMonitor }

func (h *importMonitorHandler) Execute(ctx context.Context, step *domainworkflow.Step) (appworkflow.StepResult, error) {
	req, err := resolveImportRequest(ctx, h.importRepo, step.Payload)
	if err != nil {
		return stepFailure(err), nil
	}

	switch req.Status {
	case domainimageimport.StatusSuccess, domainimageimport.StatusQuarantined:
		return appworkflow.StepResult{Status: domainworkflow.StepStatusSucceeded}, nil
	case domainimageimport.StatusFailed:
		msg := strings.TrimSpace(req.ErrorMessage)
		if msg == "" {
			msg = "external image import failed"
		}
		return stepFailure(errors.New(msg)), nil
	}

	if h.reader == nil {
		return stepFailure(errors.New("quarantine run reader is not configured")), nil
	}

	if strings.TrimSpace(req.PipelineNamespace) == "" || strings.TrimSpace(req.PipelineRunName) == "" {
		return appworkflow.StepResult{
			Status: domainworkflow.StepStatusBlocked,
			Error:  "pipeline reference is not available yet",
		}, nil
	}

	pipelineRun, err := h.reader.GetPipelineRun(ctx, req)
	if err != nil || pipelineRun == nil {
		return appworkflow.StepResult{
			Status: domainworkflow.StepStatusBlocked,
			Error:  fmt.Sprintf("pipeline run is not available yet: %v", err),
		}, nil
	}

	terminal, succeeded, message := pipelineRunOutcome(pipelineRun)
	if !terminal {
		return appworkflow.StepResult{
			Status: domainworkflow.StepStatusBlocked,
			Error:  "pipeline run is still in progress",
		}, nil
	}
	evidence := extractImportEvidence(pipelineRun)
	resolvedPolicy := policyFromEvidenceSnapshot(evidence.PolicySnapshotJSON)
	if resolvedPolicy == nil && h.policy != nil {
		resolvedPolicy, _ = h.policy.Resolve(ctx, req.TenantID)
	}
	if evidence.PolicySnapshotJSON == "" && resolvedPolicy != nil {
		if encoded, marshalErr := json.Marshal(resolvedPolicy); marshalErr == nil {
			evidence.PolicySnapshotJSON = strings.TrimSpace(string(encoded))
		}
	}
	decision := strings.TrimSpace(strings.ToLower(pipelineRunResultValue(pipelineRun, "decision")))
	if decision == "" {
		fallbackDecision, reasons := evaluateDecisionFromPolicyAndScanSummary(resolvedPolicy, evidence.ScanSummaryJSON)
		decision = fallbackDecision
		if evidence.PolicyReasonsJSON == "" && len(reasons) > 0 {
			if encoded, marshalErr := json.Marshal(reasons); marshalErr == nil {
				evidence.PolicyReasonsJSON = strings.TrimSpace(string(encoded))
			}
		}
	}
	evidence.PolicyDecision = decisionOrDefault(decision)
	if err := h.importRepo.UpdateEvidence(ctx, req.TenantID, req.ID, evidence); err != nil {
		return stepFailure(fmt.Errorf("failed to persist import evidence: %w", err)), nil
	}
	if !succeeded {
		if strings.TrimSpace(message) == "" {
			message = "quarantine pipeline failed"
		}
		if !canTransitionImportStatus(req.Status, domainimageimport.StatusFailed) {
			return stepFailure(fmt.Errorf("invalid import status transition: %s -> %s", req.Status, domainimageimport.StatusFailed)), nil
		}
		if err := h.importRepo.UpdateStatus(ctx, req.TenantID, req.ID, domainimageimport.StatusFailed, message, req.InternalImageRef); err != nil {
			return stepFailure(fmt.Errorf("failed to persist failed import status: %w", err)), nil
		}
		req.Status = domainimageimport.StatusFailed
		req.ErrorMessage = message
		h.publishImportEvent(ctx, messaging.EventTypeExternalImageImportFailed, req, message)
		return stepFailure(errors.New(message)), nil
	}

	targetStatus := domainimageimport.StatusSuccess
	if decision == "quarantine" {
		targetStatus = domainimageimport.StatusQuarantined
	}
	if err := h.importRepo.SyncEvidenceToCatalog(ctx, req.TenantID, req.ID); err != nil {
		if errors.Is(err, domainimageimport.ErrCatalogImageNotReady) {
			if !canTransitionImportStatus(req.Status, domainimageimport.StatusImporting) {
				return stepFailure(fmt.Errorf("invalid import status transition: %s -> %s", req.Status, domainimageimport.StatusImporting)), nil
			}
			if persistErr := h.importRepo.UpdateStatus(ctx, req.TenantID, req.ID, domainimageimport.StatusImporting, "catalog image is not ready for evidence sync", req.InternalImageRef); persistErr != nil {
				return stepFailure(fmt.Errorf("failed to persist deferred catalog sync state: %w", persistErr)), nil
			}
			return appworkflow.StepResult{
				Status: domainworkflow.StepStatusBlocked,
				Error:  "catalog image is not ready for evidence sync",
			}, nil
		}
		return stepFailure(fmt.Errorf("failed to sync import evidence to catalog: %w", err)), nil
	}
	if !canTransitionImportStatus(req.Status, targetStatus) {
		return stepFailure(fmt.Errorf("invalid import status transition: %s -> %s", req.Status, targetStatus)), nil
	}
	if err := h.importRepo.UpdateStatus(ctx, req.TenantID, req.ID, targetStatus, "", req.InternalImageRef); err != nil {
		return stepFailure(fmt.Errorf("failed to persist successful import status: %w", err)), nil
	}
	req.Status = targetStatus
	req.ErrorMessage = ""
	if targetStatus == domainimageimport.StatusQuarantined {
		h.publishImportEvent(ctx, messaging.EventTypeExternalImageImportQuarantined, req, "")
	} else {
		h.publishImportEvent(ctx, messaging.EventTypeExternalImageImportCompleted, req, "")
	}
	return appworkflow.StepResult{
		Status: domainworkflow.StepStatusSucceeded,
		Data: map[string]interface{}{
			"decision": decisionOrDefault(decision),
			"status":   string(targetStatus),
		},
	}, nil
}

func NewExternalImageImportWorkflowHandlers(
	importRepo domainimageimport.Repository,
	workflowRepo WorkflowRepository,
	dispatcher QuarantineDispatcher,
	reader QuarantineRunReader,
	logger *zap.Logger,
) []appworkflow.StepHandler {
	return NewExternalImageImportWorkflowHandlersWithPolicyAndEvents(importRepo, workflowRepo, dispatcher, reader, nil, nil, logger)
}

func NewExternalImageImportWorkflowHandlersWithPolicy(
	importRepo domainimageimport.Repository,
	workflowRepo WorkflowRepository,
	dispatcher QuarantineDispatcher,
	reader QuarantineRunReader,
	policy QuarantinePolicyProvider,
	logger *zap.Logger,
) []appworkflow.StepHandler {
	return NewExternalImageImportWorkflowHandlersWithPolicyAndEvents(importRepo, workflowRepo, dispatcher, reader, policy, nil, logger)
}

func NewExternalImageImportWorkflowHandlersWithPolicyAndEvents(
	importRepo domainimageimport.Repository,
	workflowRepo WorkflowRepository,
	dispatcher QuarantineDispatcher,
	reader QuarantineRunReader,
	policy QuarantinePolicyProvider,
	eventBus messaging.EventBus,
	logger *zap.Logger,
) []appworkflow.StepHandler {
	return []appworkflow.StepHandler{
		NewApprovalRequestHandlerWithEvents(importRepo, workflowRepo, eventBus, logger),
		NewApprovalDecisionHandlerWithEvents(importRepo, workflowRepo, eventBus, logger),
		NewImportDispatchHandlerWithPolicyAndEvents(importRepo, dispatcher, workflowRepo, policy, eventBus, logger),
		NewImportMonitorHandlerWithPolicyAndEvents(importRepo, reader, policy, eventBus, logger),
	}
}

func (h *approvalRequestHandler) resolveImportRequest(ctx context.Context, step *domainworkflow.Step) (*domainimageimport.ImportRequest, error) {
	return resolveImportRequest(ctx, h.importRepo, step.Payload)
}

func (h *approvalDecisionHandler) resolveImportRequest(ctx context.Context, step *domainworkflow.Step) (*domainimageimport.ImportRequest, error) {
	return resolveImportRequest(ctx, h.importRepo, step.Payload)
}

func (h *importDispatchHandler) resolveImportRequest(ctx context.Context, step *domainworkflow.Step) (*domainimageimport.ImportRequest, error) {
	return resolveImportRequest(ctx, h.importRepo, step.Payload)
}

func (h *importMonitorHandler) resolveImportRequest(ctx context.Context, step *domainworkflow.Step) (*domainimageimport.ImportRequest, error) {
	return resolveImportRequest(ctx, h.importRepo, step.Payload)
}

func resolveImportRequest(ctx context.Context, repo domainimageimport.Repository, payload map[string]interface{}) (*domainimageimport.ImportRequest, error) {
	if repo == nil {
		return nil, errors.New("image import repository is not configured")
	}
	tenantID, err := payloadUUID(payload, "tenant_id")
	if err != nil {
		return nil, err
	}
	importID, err := payloadUUID(payload, "external_image_import_id")
	if err != nil {
		return nil, err
	}
	req, err := repo.GetByID(ctx, tenantID, importID)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func payloadUUID(payload map[string]interface{}, key string) (uuid.UUID, error) {
	if payload == nil {
		return uuid.Nil, fmt.Errorf("payload is required")
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return uuid.Nil, fmt.Errorf("%s is required", key)
	}
	value, ok := raw.(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("%s must be a string", key)
	}
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s is invalid: %w", key, err)
	}
	return parsed, nil
}

func parseApprovalDecision(payload map[string]interface{}) (approved bool, reason string, decided bool) {
	approved = true
	reason = strings.TrimSpace(stringPayload(payload, "approval_reason"))

	if value, ok := payload["approved"].(bool); ok {
		approved = value
		return approved, reason, true
	}
	if status := strings.TrimSpace(strings.ToLower(stringPayload(payload, "approval_status"))); status != "" {
		switch status {
		case "approved", "allow", "accepted", "auto_approved":
			return true, reason, true
		case "rejected", "denied", "declined":
			return false, reason, true
		}
	}
	return false, reason, false
}

func stringPayload(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func stepFailure(err error) appworkflow.StepResult {
	if err == nil {
		return appworkflow.StepResult{Status: domainworkflow.StepStatusFailed}
	}
	return appworkflow.StepResult{
		Status: domainworkflow.StepStatusFailed,
		Error:  err.Error(),
	}
}

func pipelineRunOutcome(pr *tektonv1.PipelineRun) (terminal bool, succeeded bool, message string) {
	if pr == nil {
		return false, false, ""
	}
	for _, condition := range pr.Status.Conditions {
		if !strings.EqualFold(string(condition.Type), "Succeeded") {
			continue
		}
		switch strings.TrimSpace(string(condition.Status)) {
		case "True":
			return true, true, strings.TrimSpace(condition.Message)
		case "False":
			return true, false, strings.TrimSpace(condition.Message)
		default:
			return false, false, strings.TrimSpace(condition.Message)
		}
	}
	return false, false, ""
}

func pipelineRunResultValue(pr *tektonv1.PipelineRun, key string) string {
	if pr == nil {
		return ""
	}
	for _, result := range pr.Status.Results {
		if strings.TrimSpace(result.Name) != key {
			continue
		}
		value := strings.TrimSpace(result.Value.StringVal)
		if value != "" {
			return value
		}
		if len(result.Value.ArrayVal) > 0 {
			encoded, err := json.Marshal(result.Value.ArrayVal)
			if err == nil {
				return strings.TrimSpace(string(encoded))
			}
		}
		if len(result.Value.ObjectVal) > 0 {
			encoded, err := json.Marshal(result.Value.ObjectVal)
			if err == nil {
				return strings.TrimSpace(string(encoded))
			}
		}
	}
	return ""
}

func decisionOrDefault(decision string) string {
	trimmed := strings.TrimSpace(strings.ToLower(decision))
	if trimmed == "" {
		return "pass"
	}
	return trimmed
}

func dispatchAttemptFromPayload(payload map[string]interface{}) int {
	if payload == nil {
		return 1
	}
	switch raw := payload["dispatch_attempt"].(type) {
	case int:
		if raw > 0 {
			return raw
		}
	case float64:
		if raw > 0 {
			return int(raw)
		}
	}
	return 1
}

func canTransitionImportStatus(from, to domainimageimport.Status) bool {
	if from == to {
		return true
	}
	switch from {
	case domainimageimport.StatusPending:
		return to == domainimageimport.StatusApproved || to == domainimageimport.StatusFailed
	case domainimageimport.StatusApproved:
		return to == domainimageimport.StatusImporting || to == domainimageimport.StatusFailed
	case domainimageimport.StatusImporting:
		return to == domainimageimport.StatusSuccess || to == domainimageimport.StatusQuarantined || to == domainimageimport.StatusFailed
	default:
		return false
	}
}

func (h *approvalRequestHandler) publishImportEvent(ctx context.Context, eventType string, req *domainimageimport.ImportRequest, message string) {
	publishImportEvent(ctx, h.eventBus, h.logger, eventType, req, message)
}

func (h *approvalDecisionHandler) publishImportEvent(ctx context.Context, eventType string, req *domainimageimport.ImportRequest, message string) {
	publishImportEvent(ctx, h.eventBus, h.logger, eventType, req, message)
}

func (h *importDispatchHandler) publishImportEvent(ctx context.Context, eventType string, req *domainimageimport.ImportRequest, message string) {
	publishImportEvent(ctx, h.eventBus, h.logger, eventType, req, message)
}

func (h *importDispatchHandler) publishDispatchFailedEvent(ctx context.Context, req *domainimageimport.ImportRequest, message string, attempt int) {
	failureClass, failureCode := classifyImportFailure(messaging.EventTypeExternalImageImportDispatchFailed, message)
	extra := map[string]interface{}{
		"dispatch_attempt": attempt,
		"failure_class":    failureClass,
		"failure_code":     failureCode,
	}
	publishImportEventWithExtra(ctx, h.eventBus, h.logger, messaging.EventTypeExternalImageImportDispatchFailed, req, message, extra)
}

func (h *importMonitorHandler) publishImportEvent(ctx context.Context, eventType string, req *domainimageimport.ImportRequest, message string) {
	publishImportEvent(ctx, h.eventBus, h.logger, eventType, req, message)
}

func publishImportEvent(ctx context.Context, eventBus messaging.EventBus, logger *zap.Logger, eventType string, req *domainimageimport.ImportRequest, message string) {
	publishImportEventWithExtra(ctx, eventBus, logger, eventType, req, message, nil)
}

func publishImportEventWithExtra(ctx context.Context, eventBus messaging.EventBus, logger *zap.Logger, eventType string, req *domainimageimport.ImportRequest, message string, extra map[string]interface{}) {
	if eventBus == nil || req == nil || strings.TrimSpace(eventType) == "" {
		return
	}
	payload := map[string]interface{}{
		"external_image_import_id": req.ID.String(),
		"tenant_id":                req.TenantID.String(),
		"requested_by_user_id":     req.RequestedByUserID.String(),
		"request_type":             string(req.RequestType),
		"status":                   string(req.Status),
		"sor_record_id":            req.SORRecordID,
		"source_registry":          req.SourceRegistry,
		"source_image_ref":         req.SourceImageRef,
		"internal_image_ref":       req.InternalImageRef,
	}
	if trimmed := strings.TrimSpace(req.PolicyDecision); trimmed != "" {
		payload["policy_decision"] = trimmed
	}
	if trimmed := strings.TrimSpace(message); trimmed != "" {
		payload["message"] = trimmed
	}
	if eventType == messaging.EventTypeExternalImageImportDispatchFailed || eventType == messaging.EventTypeExternalImageImportFailed {
		failureClass, failureCode := classifyImportFailure(eventType, message)
		if failureClass != "" {
			payload["failure_class"] = failureClass
		}
		if failureCode != "" {
			payload["failure_code"] = failureCode
		}
	}
	for key, value := range extra {
		payload[key] = value
	}
	if idempotencyKey := importEventIdempotencyKey(eventType, req, payload); idempotencyKey != "" {
		payload["idempotency_key"] = idempotencyKey
	}
	if err := eventBus.Publish(ctx, messaging.Event{
		Type:          eventType,
		TenantID:      req.TenantID.String(),
		Source:        "image-import.workflow",
		OccurredAt:    time.Now().UTC(),
		SchemaVersion: "1.0",
		Payload:       payload,
	}); err != nil && logger != nil {
		logger.Warn("Failed to publish external image import event",
			zap.String("event_type", eventType),
			zap.String("import_request_id", req.ID.String()),
			zap.Error(err),
		)
	}
}

func classifyImportFailure(eventType, message string) (failureClass, failureCode string) {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if eventType == messaging.EventTypeExternalImageImportDispatchFailed {
		switch {
		case strings.Contains(normalized, "deadline exceeded"), strings.Contains(normalized, "timeout"):
			return "dispatch", "dispatch_timeout"
		case strings.Contains(normalized, "waiting_for_dispatch"), strings.Contains(normalized, "dispatcher unavailable"), strings.Contains(normalized, "no tekton-enabled quarantine dispatcher available"):
			return "dispatch", "dispatcher_unavailable"
		default:
			return "dispatch", "dispatch_error"
		}
	}
	switch {
	case strings.Contains(normalized, "forbidden"), strings.Contains(normalized, "unauthorized"), strings.Contains(normalized, "authentication"):
		return "auth", "auth_error"
	case strings.Contains(normalized, "no such host"), strings.Contains(normalized, "connection refused"), strings.Contains(normalized, "i/o timeout"), strings.Contains(normalized, "dial tcp"), strings.Contains(normalized, "deadline exceeded"), strings.Contains(normalized, "timeout"):
		return "connectivity", "connectivity_error"
	case strings.Contains(normalized, "policy"), strings.Contains(normalized, "quarantine"):
		return "policy", "policy_blocked"
	default:
		return "runtime", "runtime_failed"
	}
}

func importEventIdempotencyKey(eventType string, req *domainimageimport.ImportRequest, payload map[string]interface{}) string {
	if req == nil {
		return ""
	}
	switch eventType {
	case messaging.EventTypeExternalImageImportApprovalRequested:
		return req.ID.String() + ":approval_requested"
	case messaging.EventTypeExternalImageImportApproved:
		return req.ID.String() + ":approved"
	case messaging.EventTypeExternalImageImportRejected:
		return req.ID.String() + ":rejected"
	case messaging.EventTypeExternalImageImportDispatchFailed:
		attempt := dispatchAttemptFromPayload(payload)
		return fmt.Sprintf("%s:dispatch_failed:%d", req.ID.String(), attempt)
	case messaging.EventTypeExternalImageImportCompleted, messaging.EventTypeExternalImageImportQuarantined, messaging.EventTypeExternalImageImportFailed:
		return req.ID.String() + ":" + string(req.Status)
	default:
		return req.ID.String() + ":" + strings.ReplaceAll(strings.TrimSpace(eventType), ".", "_")
	}
}

func extractImportEvidence(pr *tektonv1.PipelineRun) domainimageimport.ImportEvidence {
	if pr == nil {
		return domainimageimport.ImportEvidence{}
	}
	return domainimageimport.ImportEvidence{
		PolicyDecision:     strings.TrimSpace(strings.ToLower(pipelineRunResultValue(pr, "decision"))),
		PolicyReasonsJSON:  strings.TrimSpace(pipelineRunResultValue(pr, "decision-reasons-json")),
		PolicySnapshotJSON: strings.TrimSpace(pipelineRunResultValue(pr, "policy-snapshot-json")),
		ScanSummaryJSON:    strings.TrimSpace(pipelineRunResultValue(pr, "scan-summary")),
		SBOMSummaryJSON:    strings.TrimSpace(pipelineRunResultValue(pr, "sbom-summary")),
		SBOMEvidenceJSON:   strings.TrimSpace(pipelineRunResultValue(pr, "sbom-evidence")),
		SourceImageDigest:  strings.TrimSpace(pipelineRunResultValue(pr, "source-image-digest")),
	}
}

func policyFromEvidenceSnapshot(snapshot string) *QuarantinePolicy {
	trimmed := strings.TrimSpace(snapshot)
	if trimmed == "" {
		return nil
	}
	var policy QuarantinePolicy
	if err := json.Unmarshal([]byte(trimmed), &policy); err != nil {
		return nil
	}
	if thresholds, ok := extractThresholdsFromSnapshot(trimmed); ok {
		policy.MaxCritical = thresholds["max_critical"]
		policy.MaxP2 = thresholds["max_p2"]
		policy.MaxP3 = thresholds["max_p3"]
	}
	if strings.TrimSpace(policy.Mode) == "" {
		policy.Mode = "dry_run"
	}
	return &policy
}

func extractThresholdsFromSnapshot(snapshot string) (map[string]int, bool) {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(snapshot), &payload); err != nil {
		return nil, false
	}
	raw, ok := payload["thresholds"].(map[string]interface{})
	if !ok {
		return nil, false
	}
	thresholds := map[string]int{
		"max_critical": 0,
		"max_p2":       0,
		"max_p3":       0,
	}
	for key := range thresholds {
		if value, exists := raw[key]; exists {
			switch typed := value.(type) {
			case float64:
				thresholds[key] = int(math.Round(typed))
			case int:
				thresholds[key] = typed
			}
		}
	}
	return thresholds, true
}

func evaluateDecisionFromPolicyAndScanSummary(policy *QuarantinePolicy, scanSummaryJSON string) (string, []string) {
	if policy == nil {
		return "pass", nil
	}
	critical, high, medium, maxCVSS := extractVulnerabilityCounts(scanSummaryJSON)
	var reasons []string
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
	if len(reasons) == 0 {
		return "pass", nil
	}
	if strings.EqualFold(strings.TrimSpace(policy.Mode), "enforce") {
		return "quarantine", reasons
	}
	return "pass", reasons
}

func extractVulnerabilityCounts(scanSummaryJSON string) (critical int, high int, medium int, maxCVSS float64) {
	trimmed := strings.TrimSpace(scanSummaryJSON)
	if trimmed == "" {
		return 0, 0, 0, 0
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return 0, 0, 0, 0
	}
	if vulnMap, ok := payload["vulnerabilities"].(map[string]interface{}); ok {
		critical = mapInt(vulnMap, "critical")
		high = mapInt(vulnMap, "high")
		medium = mapInt(vulnMap, "medium")
	}
	maxCVSS = mapFloat(payload, "max_cvss")
	if maxCVSS == 0 {
		maxCVSS = mapFloat(payload, "maxCvss")
	}
	return critical, high, medium, maxCVSS
}

func mapInt(values map[string]interface{}, key string) int {
	raw, ok := values[key]
	if !ok {
		return 0
	}
	switch typed := raw.(type) {
	case float64:
		return int(math.Round(typed))
	case int:
		return typed
	default:
		return 0
	}
}

func mapFloat(values map[string]interface{}, key string) float64 {
	raw, ok := values[key]
	if !ok {
		return 0
	}
	switch typed := raw.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	default:
		return 0
	}
}
