package steps

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	domainimageimport "github.com/srikarm/image-factory/internal/domain/imageimport"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

type imageImportRepoStub struct {
	item              *domainimageimport.ImportRequest
	lastStatus        domainimageimport.Status
	lastErrMsg        string
	lastIntRef        string
	lastEvidence      domainimageimport.ImportEvidence
	statusUpdates     int
	evidenceUpdates   int
	syncCatalogCalled bool
	syncCatalogErr    error
}

func (s *imageImportRepoStub) Create(ctx context.Context, req *domainimageimport.ImportRequest) error {
	return nil
}

func (s *imageImportRepoStub) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domainimageimport.ImportRequest, error) {
	if s.item == nil {
		return nil, domainimageimport.ErrImportNotFound
	}
	return s.item, nil
}

func (s *imageImportRepoStub) ListByTenant(ctx context.Context, tenantID uuid.UUID, requestType domainimageimport.RequestType, limit, offset int) ([]*domainimageimport.ImportRequest, error) {
	return nil, nil
}

func (s *imageImportRepoStub) ListAll(ctx context.Context, requestType domainimageimport.RequestType, limit, offset int) ([]*domainimageimport.ImportRequest, error) {
	return nil, nil
}

func (s *imageImportRepoStub) ListReleasedByTenant(ctx context.Context, tenantID uuid.UUID, search string, limit, offset int) ([]*domainimageimport.ReleasedArtifact, int, error) {
	return nil, 0, nil
}

func (s *imageImportRepoStub) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domainimageimport.Status, errorMessage, internalImageRef string) error {
	s.statusUpdates++
	s.lastStatus = status
	s.lastErrMsg = errorMessage
	s.lastIntRef = internalImageRef
	if s.item != nil {
		s.item.Status = status
		s.item.ErrorMessage = errorMessage
		s.item.InternalImageRef = internalImageRef
	}
	return nil
}

func (s *imageImportRepoStub) UpdatePipelineRefs(ctx context.Context, tenantID, id uuid.UUID, pipelineRunName, pipelineNamespace string) error {
	if s.item != nil {
		s.item.PipelineRunName = pipelineRunName
		s.item.PipelineNamespace = pipelineNamespace
	}
	return nil
}

func (s *imageImportRepoStub) UpdateEvidence(ctx context.Context, tenantID, id uuid.UUID, evidence domainimageimport.ImportEvidence) error {
	s.evidenceUpdates++
	s.lastEvidence = evidence
	if s.item != nil {
		s.item.PolicyDecision = evidence.PolicyDecision
		s.item.PolicyReasonsJSON = evidence.PolicyReasonsJSON
		s.item.PolicySnapshotJSON = evidence.PolicySnapshotJSON
		s.item.ScanSummaryJSON = evidence.ScanSummaryJSON
		s.item.SBOMSummaryJSON = evidence.SBOMSummaryJSON
		s.item.SBOMEvidenceJSON = evidence.SBOMEvidenceJSON
		s.item.SourceImageDigest = evidence.SourceImageDigest
	}
	return nil
}

func (s *imageImportRepoStub) UpdateReleaseState(ctx context.Context, tenantID, id uuid.UUID, state domainimageimport.ReleaseState, blockerReason string, actorUserID *uuid.UUID, reason string, requestedAt, releasedAt *time.Time) error {
	if s.item != nil {
		s.item.ReleaseState = state
		s.item.ReleaseBlockerReason = blockerReason
		s.item.ReleaseActorUserID = actorUserID
		s.item.ReleaseReason = reason
		s.item.ReleaseRequestedAt = requestedAt
		s.item.ReleasedAt = releasedAt
	}
	return nil
}

func (s *imageImportRepoStub) SyncEvidenceToCatalog(ctx context.Context, tenantID, id uuid.UUID) error {
	s.syncCatalogCalled = true
	return s.syncCatalogErr
}

type workflowRepoStub struct {
	stepKey string
	status  domainworkflow.StepStatus
}

func (s *workflowRepoStub) UpdateStepStatus(ctx context.Context, instanceID uuid.UUID, stepKey string, status domainworkflow.StepStatus, errMsg *string) error {
	s.stepKey = stepKey
	s.status = status
	return nil
}

type dispatcherStub struct {
	result DispatchResult
	err    error
	calls  int
}

func (s *dispatcherStub) Dispatch(ctx context.Context, req *domainimageimport.ImportRequest) (DispatchResult, error) {
	s.calls++
	return s.result, s.err
}

type runReaderStub struct {
	run *tektonv1.PipelineRun
	err error
	n   int
}

func (s *runReaderStub) GetPipelineRun(ctx context.Context, req *domainimageimport.ImportRequest) (*tektonv1.PipelineRun, error) {
	s.n++
	return s.run, s.err
}

type eventBusStub struct {
	events []messaging.Event
}

func (s *eventBusStub) Publish(ctx context.Context, event messaging.Event) error {
	s.events = append(s.events, event)
	return nil
}

func (s *eventBusStub) Subscribe(eventType string, handler messaging.Handler) (unsubscribe func()) {
	return func() {}
}

func TestApprovalRequestHandler_KeepsDecisionStepBlockedUntilManualDecision(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{ID: reqID, TenantID: tenantID},
	}
	workflowRepo := &workflowRepoStub{}
	handler := NewApprovalRequestHandler(repo, workflowRepo, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		InstanceID: uuid.New(),
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", result.Status)
	}
	if workflowRepo.stepKey != "" {
		t.Fatalf("expected approval.decision to remain blocked until manual decision")
	}
}

func TestApprovalRequestHandler_PublishesLifecycleEvent(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	requesterID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:                reqID,
			TenantID:          tenantID,
			RequestedByUserID: requesterID,
			Status:            domainimageimport.StatusPending,
		},
	}
	bus := &eventBusStub{}
	handler := NewApprovalRequestHandlerWithEvents(repo, &workflowRepoStub{}, bus, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		InstanceID: uuid.New(),
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", result.Status)
	}
	if len(bus.events) != 1 {
		t.Fatalf("expected one published event, got %d", len(bus.events))
	}
	if bus.events[0].Type != messaging.EventTypeExternalImageImportApprovalRequested {
		t.Fatalf("expected approval requested event type, got %s", bus.events[0].Type)
	}
}

func TestApprovalDecisionHandler_Approved_UnblocksDispatchAndMarksApproved(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{ID: reqID, TenantID: tenantID, Status: domainimageimport.StatusPending},
	}
	workflowRepo := &workflowRepoStub{}
	handler := NewApprovalDecisionHandler(repo, workflowRepo, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		InstanceID: uuid.New(),
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
			"approved":                 true,
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", result.Status)
	}
	if repo.lastStatus != domainimageimport.StatusApproved {
		t.Fatalf("expected status approved, got %s", repo.lastStatus)
	}
	if workflowRepo.stepKey != StepImportDispatch || workflowRepo.status != domainworkflow.StepStatusPending {
		t.Fatalf("expected import.dispatch to be unblocked")
	}
}

func TestApprovalDecisionHandler_RequiresExplicitDecision(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{ID: reqID, TenantID: tenantID, Status: domainimageimport.StatusPending},
	}
	workflowRepo := &workflowRepoStub{}
	handler := NewApprovalDecisionHandler(repo, workflowRepo, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		InstanceID: uuid.New(),
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusFailed {
		t.Fatalf("expected failed status when explicit decision is missing, got %s", result.Status)
	}
	if !strings.Contains(result.Error, "approval decision is required") {
		t.Fatalf("expected explicit decision error, got %q", result.Error)
	}
	if repo.statusUpdates != 0 {
		t.Fatalf("expected no status updates without explicit decision")
	}
	if workflowRepo.stepKey != "" {
		t.Fatalf("expected no workflow step updates without explicit decision")
	}
}

func TestImportDispatchHandler_MarksImporting(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:       reqID,
			TenantID: tenantID,
			Status:   domainimageimport.StatusApproved,
		},
	}
	dispatcher := &dispatcherStub{
		result: DispatchResult{
			PipelineRunName:  "quarantine-import-abc123",
			Namespace:        "tenant-a",
			InternalImageRef: "registry.local/quarantine/tenant-a/example:latest",
		},
	}
	workflowRepo := &workflowRepoStub{}
	handler := NewImportDispatchHandler(repo, dispatcher, workflowRepo, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		InstanceID: uuid.New(),
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", result.Status)
	}
	if repo.lastStatus != domainimageimport.StatusImporting {
		t.Fatalf("expected importing status, got %s", repo.lastStatus)
	}
	if repo.lastIntRef == "" {
		t.Fatalf("expected internal image ref to be set")
	}
	if workflowRepo.stepKey != StepImportMonitor || workflowRepo.status != domainworkflow.StepStatusPending {
		t.Fatalf("expected import.monitor to be unblocked")
	}
	if repo.item.PipelineRunName == "" || repo.item.PipelineNamespace == "" {
		t.Fatalf("expected pipeline refs to be stored")
	}
}

func TestImportDispatchHandler_BlocksWhenDispatcherUnavailable(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:       reqID,
			TenantID: tenantID,
			Status:   domainimageimport.StatusApproved,
		},
	}
	handler := NewImportDispatchHandler(repo, nil, &workflowRepoStub{}, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		InstanceID: uuid.New(),
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusBlocked {
		t.Fatalf("expected blocked status, got %s", result.Status)
	}
	if !strings.Contains(result.Error, "waiting_for_dispatch") {
		t.Fatalf("expected waiting_for_dispatch error hint, got %q", result.Error)
	}
	if repo.statusUpdates != 0 {
		t.Fatalf("expected no status mutation while dispatcher unavailable, got %d updates", repo.statusUpdates)
	}
}

func TestImportDispatchHandler_ReplaysBlockedDispatchWhenDispatcherBecomesAvailable(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:       reqID,
			TenantID: tenantID,
			Status:   domainimageimport.StatusApproved,
		},
	}
	workflowRepo := &workflowRepoStub{}
	dispatcher := &dispatcherStub{
		err: errors.New("waiting_for_dispatch: no tekton-enabled quarantine dispatcher available"),
	}
	handler := NewImportDispatchHandler(repo, dispatcher, workflowRepo, zap.NewNop())
	step := &domainworkflow.Step{
		InstanceID: uuid.New(),
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	}

	first, err := handler.Execute(context.Background(), step)
	if err != nil {
		t.Fatalf("expected no execute error on blocked pass, got %v", err)
	}
	if first.Status != domainworkflow.StepStatusBlocked {
		t.Fatalf("expected blocked status on first pass, got %s", first.Status)
	}
	if !strings.Contains(first.Error, "waiting_for_dispatch") {
		t.Fatalf("expected waiting_for_dispatch error on first pass, got %q", first.Error)
	}
	if repo.item.Status != domainimageimport.StatusApproved {
		t.Fatalf("expected import status to remain approved on blocked pass, got %s", repo.item.Status)
	}
	if repo.statusUpdates != 0 {
		t.Fatalf("expected no status updates while blocked, got %d", repo.statusUpdates)
	}

	dispatcher.err = nil
	dispatcher.result = DispatchResult{
		PipelineRunName:  "pr-j09-replay",
		Namespace:        "image-factory",
		InternalImageRef: "registry.local/quarantine/acme/tooling:1.0.0",
	}

	second, err := handler.Execute(context.Background(), step)
	if err != nil {
		t.Fatalf("expected no execute error on replay pass, got %v", err)
	}
	if second.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded status on replay pass, got %s", second.Status)
	}
	if repo.item.Status != domainimageimport.StatusImporting {
		t.Fatalf("expected import status importing after replay, got %s", repo.item.Status)
	}
	if repo.item.PipelineRunName != "pr-j09-replay" || repo.item.PipelineNamespace != "image-factory" {
		t.Fatalf("expected pipeline refs persisted after replay, got name=%q namespace=%q", repo.item.PipelineRunName, repo.item.PipelineNamespace)
	}
	if workflowRepo.stepKey != StepImportMonitor || workflowRepo.status != domainworkflow.StepStatusPending {
		t.Fatalf("expected import.monitor to be unblocked after replay")
	}
	if dispatcher.calls != 2 {
		t.Fatalf("expected two dispatch attempts, got %d", dispatcher.calls)
	}
}

func TestImportDispatchHandler_DispatchFailurePublishesDispatchFailedEvent(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:                reqID,
			TenantID:          tenantID,
			RequestedByUserID: uuid.New(),
			Status:            domainimageimport.StatusApproved,
		},
	}
	dispatcher := &dispatcherStub{err: context.DeadlineExceeded}
	bus := &eventBusStub{}
	handler := NewImportDispatchHandlerWithPolicyAndEvents(repo, dispatcher, &workflowRepoStub{}, nil, bus, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
			"dispatch_attempt":         3,
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if repo.lastStatus != domainimageimport.StatusFailed {
		t.Fatalf("expected failed status persisted, got %s", repo.lastStatus)
	}
	if !strings.HasPrefix(repo.lastErrMsg, "dispatch_failed:") {
		t.Fatalf("expected persisted error to include dispatch_failed prefix, got %q", repo.lastErrMsg)
	}
	if len(bus.events) != 1 {
		t.Fatalf("expected one event publish, got %d", len(bus.events))
	}
	if bus.events[0].Type != messaging.EventTypeExternalImageImportDispatchFailed {
		t.Fatalf("expected dispatch_failed event type, got %s", bus.events[0].Type)
	}
	if gotAttempt, ok := bus.events[0].Payload["dispatch_attempt"].(int); !ok || gotAttempt != 3 {
		t.Fatalf("expected dispatch_attempt=3, got %v", bus.events[0].Payload["dispatch_attempt"])
	}
	if gotClass, _ := bus.events[0].Payload["failure_class"].(string); gotClass != "dispatch" {
		t.Fatalf("expected failure_class=dispatch, got %q", gotClass)
	}
	if gotCode, _ := bus.events[0].Payload["failure_code"].(string); gotCode != "dispatch_timeout" {
		t.Fatalf("expected failure_code=dispatch_timeout, got %q", gotCode)
	}
	expectedKey := reqID.String() + ":dispatch_failed:3"
	if gotKey, _ := bus.events[0].Payload["idempotency_key"].(string); gotKey != expectedKey {
		t.Fatalf("expected idempotency_key=%s, got %q", expectedKey, gotKey)
	}
}

func TestImportDispatchHandler_FailedReplayDoesNotDispatchOrMutate(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:       reqID,
			TenantID: tenantID,
			Status:   domainimageimport.StatusFailed,
		},
	}
	dispatcher := &dispatcherStub{}
	bus := &eventBusStub{}
	handler := NewImportDispatchHandlerWithPolicyAndEvents(repo, dispatcher, &workflowRepoStub{}, nil, bus, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if !strings.Contains(result.Error, "not dispatchable") {
		t.Fatalf("expected non-dispatchable error, got %q", result.Error)
	}
	if dispatcher.calls != 0 {
		t.Fatalf("expected dispatcher not to be invoked on terminal replay, got %d calls", dispatcher.calls)
	}
	if repo.statusUpdates != 0 {
		t.Fatalf("expected no status mutations on terminal replay, got %d", repo.statusUpdates)
	}
	if len(bus.events) != 0 {
		t.Fatalf("expected no events on terminal replay, got %d", len(bus.events))
	}
}

func TestImportMonitorHandler_CompletesAsQuarantinedOnDecision(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:                reqID,
			TenantID:          tenantID,
			Status:            domainimageimport.StatusImporting,
			PipelineRunName:   "quarantine-import-abc123",
			PipelineNamespace: "tenant-a",
		},
	}
	reader := &runReaderStub{
		run: &tektonv1.PipelineRun{
			Status: tektonv1.PipelineRunStatus{
				Status: duckStatusTrue(),
				PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
					Results: []tektonv1.PipelineRunResult{
						{
							Name: "decision",
							Value: tektonv1.ResultValue{
								StringVal: "quarantine",
							},
						},
						{
							Name: "decision-reasons-json",
							Value: tektonv1.ResultValue{
								StringVal: `["critical_count(1) > max_critical(0)"]`,
							},
						},
						{
							Name: "scan-summary",
							Value: tektonv1.ResultValue{
								StringVal: `{"vulnerabilities":{"critical":1}}`,
							},
						},
					},
				},
			},
		},
	}
	handler := NewImportMonitorHandler(repo, reader, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", result.Status)
	}
	if repo.item.Status != domainimageimport.StatusQuarantined {
		t.Fatalf("expected quarantined status, got %s", repo.item.Status)
	}
	if repo.lastEvidence.PolicyDecision != "quarantine" {
		t.Fatalf("expected policy decision evidence to be stored")
	}
	if repo.lastEvidence.ScanSummaryJSON == "" {
		t.Fatalf("expected scan summary evidence to be stored")
	}
	if !repo.syncCatalogCalled {
		t.Fatalf("expected catalog sync to be triggered")
	}
}

func TestImportMonitorHandler_BlocksWhenCatalogImageNotReady(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:                reqID,
			TenantID:          tenantID,
			Status:            domainimageimport.StatusImporting,
			PipelineRunName:   "quarantine-import-abc123",
			PipelineNamespace: "tenant-a",
		},
		syncCatalogErr: domainimageimport.ErrCatalogImageNotReady,
	}
	reader := &runReaderStub{
		run: &tektonv1.PipelineRun{
			Status: tektonv1.PipelineRunStatus{
				Status: duckStatusTrue(),
				PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
					Results: []tektonv1.PipelineRunResult{
						{Name: "decision", Value: tektonv1.ResultValue{StringVal: "pass"}},
					},
				},
			},
		},
	}
	handler := NewImportMonitorHandler(repo, reader, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusBlocked {
		t.Fatalf("expected blocked status, got %s", result.Status)
	}
	if repo.item.Status != domainimageimport.StatusImporting {
		t.Fatalf("expected importing status to remain for retry, got %s", repo.item.Status)
	}
	if repo.item.ErrorMessage == "" || !strings.Contains(strings.ToLower(repo.item.ErrorMessage), "catalog image is not ready") {
		t.Fatalf("expected deferred sync reason to be persisted on import request, got %q", repo.item.ErrorMessage)
	}
}

func TestImportMonitorHandler_FailsWhenRunReaderMissing(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:                reqID,
			TenantID:          tenantID,
			Status:            domainimageimport.StatusImporting,
			PipelineRunName:   "quarantine-import-abc123",
			PipelineNamespace: "tenant-a",
		},
	}
	handler := NewImportMonitorHandler(repo, nil, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if !strings.Contains(strings.ToLower(result.Error), "run reader") {
		t.Fatalf("expected run reader missing error, got %q", result.Error)
	}
	if repo.item.Status != domainimageimport.StatusImporting {
		t.Fatalf("expected importing status unchanged, got %s", repo.item.Status)
	}
}

func TestImportMonitorHandler_EvaluatesPolicyFromScanSummaryWhenDecisionMissing_Enforce(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:                reqID,
			TenantID:          tenantID,
			Status:            domainimageimport.StatusImporting,
			PipelineRunName:   "quarantine-import-abc123",
			PipelineNamespace: "tenant-a",
		},
	}
	reader := &runReaderStub{
		run: &tektonv1.PipelineRun{
			Status: tektonv1.PipelineRunStatus{
				Status: duckStatusTrue(),
				PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
					Results: []tektonv1.PipelineRunResult{
						{Name: "scan-summary", Value: tektonv1.ResultValue{StringVal: `{"vulnerabilities":{"critical":1,"high":0,"medium":0}}`}},
					},
				},
			},
		},
	}
	policyProvider := QuarantinePolicyProviderFunc(func(ctx context.Context, tenantID uuid.UUID) (*QuarantinePolicy, error) {
		return &QuarantinePolicy{Mode: "enforce", MaxCritical: 0, MaxP2: 0, MaxP3: 0, MaxCVSS: 10}, nil
	})
	handler := NewImportMonitorHandlerWithPolicy(repo, reader, policyProvider, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", result.Status)
	}
	if repo.item.Status != domainimageimport.StatusQuarantined {
		t.Fatalf("expected quarantined status, got %s", repo.item.Status)
	}
	if repo.lastEvidence.PolicyDecision != "quarantine" {
		t.Fatalf("expected derived quarantine decision, got %q", repo.lastEvidence.PolicyDecision)
	}
	if repo.lastEvidence.PolicyReasonsJSON == "" {
		t.Fatalf("expected derived policy reasons to be stored")
	}
}

func TestImportMonitorHandler_EvaluatesPolicyFromScanSummaryWhenDecisionMissing_DryRun(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:                reqID,
			TenantID:          tenantID,
			Status:            domainimageimport.StatusImporting,
			PipelineRunName:   "quarantine-import-abc123",
			PipelineNamespace: "tenant-a",
		},
	}
	reader := &runReaderStub{
		run: &tektonv1.PipelineRun{
			Status: tektonv1.PipelineRunStatus{
				Status: duckStatusTrue(),
				PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
					Results: []tektonv1.PipelineRunResult{
						{Name: "scan-summary", Value: tektonv1.ResultValue{StringVal: `{"vulnerabilities":{"critical":2}}`}},
					},
				},
			},
		},
	}
	policyProvider := QuarantinePolicyProviderFunc(func(ctx context.Context, tenantID uuid.UUID) (*QuarantinePolicy, error) {
		return &QuarantinePolicy{Mode: "dry_run", MaxCritical: 0, MaxP2: 0, MaxP3: 0, MaxCVSS: 0}, nil
	})
	handler := NewImportMonitorHandlerWithPolicy(repo, reader, policyProvider, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", result.Status)
	}
	if repo.item.Status != domainimageimport.StatusSuccess {
		t.Fatalf("expected success status in dry_run mode, got %s", repo.item.Status)
	}
	if repo.lastEvidence.PolicyDecision != "pass" {
		t.Fatalf("expected pass policy decision in dry_run mode, got %q", repo.lastEvidence.PolicyDecision)
	}
}

func TestImportMonitorHandler_PublishesTerminalEvent(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:                reqID,
			TenantID:          tenantID,
			RequestedByUserID: uuid.New(),
			Status:            domainimageimport.StatusImporting,
			PipelineRunName:   "quarantine-import-abc123",
			PipelineNamespace: "tenant-a",
		},
	}
	reader := &runReaderStub{
		run: &tektonv1.PipelineRun{
			Status: tektonv1.PipelineRunStatus{
				Status: duckStatusTrue(),
				PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
					Results: []tektonv1.PipelineRunResult{
						{Name: "decision", Value: tektonv1.ResultValue{StringVal: "pass"}},
					},
				},
			},
		},
	}
	bus := &eventBusStub{}
	handler := NewImportMonitorHandlerWithPolicyAndEvents(repo, reader, nil, bus, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", result.Status)
	}
	deadline := time.Now().Add(100 * time.Millisecond)
	for len(bus.events) == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if len(bus.events) == 0 {
		t.Fatalf("expected terminal event publish")
	}
	if bus.events[0].Type != messaging.EventTypeExternalImageImportCompleted {
		t.Fatalf("expected completed event type, got %s", bus.events[0].Type)
	}
}

func TestApprovalDecisionHandler_RejectsInvalidStatusTransition(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:       reqID,
			TenantID: tenantID,
			Status:   domainimageimport.StatusImporting,
		},
	}
	handler := NewApprovalDecisionHandler(repo, &workflowRepoStub{}, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		InstanceID: uuid.New(),
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
			"approved":                 true,
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if !strings.Contains(result.Error, "invalid import status transition") {
		t.Fatalf("expected invalid transition message, got %q", result.Error)
	}
}

func TestImportMonitorHandler_ReplayTerminalSuccess_NoDuplicateSideEffects(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:                reqID,
			TenantID:          tenantID,
			RequestedByUserID: uuid.New(),
			Status:            domainimageimport.StatusSuccess,
		},
	}
	reader := &runReaderStub{}
	bus := &eventBusStub{}
	handler := NewImportMonitorHandlerWithPolicyAndEvents(repo, reader, nil, bus, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", result.Status)
	}
	if repo.statusUpdates != 0 || repo.evidenceUpdates != 0 || repo.syncCatalogCalled {
		t.Fatalf("expected no repository side effects on replay, got status_updates=%d evidence_updates=%d sync_called=%v", repo.statusUpdates, repo.evidenceUpdates, repo.syncCatalogCalled)
	}
	if reader.n != 0 {
		t.Fatalf("expected no pipeline-read call on terminal replay, got %d", reader.n)
	}
	if len(bus.events) != 0 {
		t.Fatalf("expected no duplicate terminal events, got %d", len(bus.events))
	}
}

func TestImportMonitorHandler_ReplayTerminalQuarantined_NoDuplicateSideEffects(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:                reqID,
			TenantID:          tenantID,
			RequestedByUserID: uuid.New(),
			Status:            domainimageimport.StatusQuarantined,
		},
	}
	reader := &runReaderStub{}
	bus := &eventBusStub{}
	handler := NewImportMonitorHandlerWithPolicyAndEvents(repo, reader, nil, bus, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", result.Status)
	}
	if repo.statusUpdates != 0 || repo.evidenceUpdates != 0 || repo.syncCatalogCalled {
		t.Fatalf("expected no repository side effects on replay, got status_updates=%d evidence_updates=%d sync_called=%v", repo.statusUpdates, repo.evidenceUpdates, repo.syncCatalogCalled)
	}
	if reader.n != 0 {
		t.Fatalf("expected no pipeline-read call on terminal replay, got %d", reader.n)
	}
	if len(bus.events) != 0 {
		t.Fatalf("expected no duplicate terminal events, got %d", len(bus.events))
	}
}

func TestImportMonitorHandler_ReplayTerminalFailed_NoDuplicateSideEffects(t *testing.T) {
	tenantID := uuid.New()
	reqID := uuid.New()
	repo := &imageImportRepoStub{
		item: &domainimageimport.ImportRequest{
			ID:                reqID,
			TenantID:          tenantID,
			RequestedByUserID: uuid.New(),
			Status:            domainimageimport.StatusFailed,
			ErrorMessage:      "terminal failure",
		},
	}
	reader := &runReaderStub{}
	bus := &eventBusStub{}
	handler := NewImportMonitorHandlerWithPolicyAndEvents(repo, reader, nil, bus, zap.NewNop())

	result, err := handler.Execute(context.Background(), &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": reqID.String(),
		},
	})
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if repo.statusUpdates != 0 || repo.evidenceUpdates != 0 || repo.syncCatalogCalled {
		t.Fatalf("expected no repository side effects on replay, got status_updates=%d evidence_updates=%d sync_called=%v", repo.statusUpdates, repo.evidenceUpdates, repo.syncCatalogCalled)
	}
	if reader.n != 0 {
		t.Fatalf("expected no pipeline-read call on terminal replay, got %d", reader.n)
	}
	if len(bus.events) != 0 {
		t.Fatalf("expected no duplicate terminal events, got %d", len(bus.events))
	}
}

func duckStatusTrue() duckv1.Status {
	return duckv1.Status{
		Conditions: duckv1.Conditions{
			{
				Type:   apis.ConditionSucceeded,
				Status: corev1.ConditionTrue,
			},
		},
	}
}
