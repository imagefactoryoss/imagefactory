package imageimport

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type serviceTestRepository struct {
	created            []*ImportRequest
	byID               map[uuid.UUID]*ImportRequest
	releasedArtifacts  []*ReleasedArtifact
	releasedTotal      int
	lastReleasedTenant uuid.UUID
	lastReleasedSearch string
	lastReleasedLimit  int
	lastReleasedOffset int
}

func (r *serviceTestRepository) Create(ctx context.Context, req *ImportRequest) error {
	r.created = append(r.created, req)
	return nil
}

func (r *serviceTestRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*ImportRequest, error) {
	if r.byID != nil {
		if item, ok := r.byID[id]; ok && item.TenantID == tenantID {
			return item, nil
		}
	}
	return nil, ErrImportNotFound
}

func (r *serviceTestRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, requestType RequestType, limit, offset int) ([]*ImportRequest, error) {
	if r.byID == nil {
		return []*ImportRequest{}, nil
	}
	rows := make([]*ImportRequest, 0, len(r.byID))
	for _, item := range r.byID {
		if item == nil || item.TenantID != tenantID {
			continue
		}
		if requestType != "" && item.RequestType != requestType {
			continue
		}
		rows = append(rows, item)
	}
	return rows, nil
}

func (r *serviceTestRepository) ListAll(ctx context.Context, requestType RequestType, limit, offset int) ([]*ImportRequest, error) {
	return []*ImportRequest{}, nil
}

func (r *serviceTestRepository) ListReleasedByTenant(ctx context.Context, tenantID uuid.UUID, search string, limit, offset int) ([]*ReleasedArtifact, int, error) {
	r.lastReleasedTenant = tenantID
	r.lastReleasedSearch = search
	r.lastReleasedLimit = limit
	r.lastReleasedOffset = offset
	return r.releasedArtifacts, r.releasedTotal, nil
}

func (r *serviceTestRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status Status, errorMessage, internalImageRef string) error {
	if r.byID != nil {
		if item, ok := r.byID[id]; ok && item.TenantID == tenantID {
			item.Status = status
			item.ErrorMessage = errorMessage
			item.InternalImageRef = internalImageRef
			return nil
		}
	}
	return ErrImportNotFound
}

func TestServiceWithdrawImportRequest_Success(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	importID := uuid.New()
	repo := &serviceTestRepository{
		byID: map[uuid.UUID]*ImportRequest{
			importID: {
				ID:                importID,
				TenantID:          tenantID,
				RequestedByUserID: uuid.New(),
				RequestType:       RequestTypeQuarantine,
				Status:            StatusPending,
			},
		},
	}
	service := NewService(repo, nil, nil, nil, zap.NewNop())

	updated, err := service.WithdrawImportRequest(context.Background(), tenantID, actorID, importID, "submitted in error")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Status != StatusFailed {
		t.Fatalf("expected status failed after withdraw, got %s", updated.Status)
	}
	if updated.ErrorMessage != "Withdrawn: submitted in error" {
		t.Fatalf("expected withdraw message to persist, got %q", updated.ErrorMessage)
	}
}

func TestServiceWithdrawImportRequest_OnlyPending(t *testing.T) {
	tenantID := uuid.New()
	importID := uuid.New()
	repo := &serviceTestRepository{
		byID: map[uuid.UUID]*ImportRequest{
			importID: {
				ID:                importID,
				TenantID:          tenantID,
				RequestedByUserID: uuid.New(),
				RequestType:       RequestTypeQuarantine,
				Status:            StatusApproved,
			},
		},
	}
	service := NewService(repo, nil, nil, nil, zap.NewNop())

	_, err := service.WithdrawImportRequest(context.Background(), tenantID, uuid.New(), importID, "")
	if !errors.Is(err, ErrImportNotWithdrawable) {
		t.Fatalf("expected ErrImportNotWithdrawable, got %v", err)
	}
}

func (r *serviceTestRepository) UpdatePipelineRefs(ctx context.Context, tenantID, id uuid.UUID, pipelineRunName, pipelineNamespace string) error {
	return nil
}

func (r *serviceTestRepository) UpdateEvidence(ctx context.Context, tenantID, id uuid.UUID, evidence ImportEvidence) error {
	return nil
}

func (r *serviceTestRepository) UpdateReleaseState(ctx context.Context, tenantID, id uuid.UUID, state ReleaseState, blockerReason string, actorUserID *uuid.UUID, reason string, requestedAt, releasedAt *time.Time) error {
	if r.byID != nil {
		if item, ok := r.byID[id]; ok && item.TenantID == tenantID {
			item.ReleaseState = state
			item.ReleaseBlockerReason = blockerReason
			item.ReleaseActorUserID = actorUserID
			item.ReleaseReason = reason
			item.ReleaseRequestedAt = requestedAt
			item.ReleasedAt = releasedAt
			return nil
		}
	}
	return ErrImportNotFound
}

func (r *serviceTestRepository) SyncEvidenceToCatalog(ctx context.Context, tenantID, id uuid.UUID) error {
	return nil
}

type serviceTestSORValidator struct {
	result    bool
	err       error
	callCount int
}

func (v *serviceTestSORValidator) ValidateRegistration(ctx context.Context, tenantID uuid.UUID, sorRecordID string) (bool, error) {
	v.callCount++
	if v.err != nil {
		return false, v.err
	}
	return v.result, nil
}

type serviceTestCapabilityChecker struct {
	entitled  bool
	err       error
	callCount int
}

func (c *serviceTestCapabilityChecker) IsImportEntitled(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	c.callCount++
	if c.err != nil {
		return false, c.err
	}
	return c.entitled, nil
}

type serviceTestApprovalRequester struct {
	called    bool
	err       error
	callCount int
}

func (a *serviceTestApprovalRequester) CreateImportApproval(ctx context.Context, req *ImportRequest) error {
	a.called = true
	a.callCount++
	return a.err
}

func TestServiceCreateImportRequest_ReturnsSORRequired(t *testing.T) {
	repo := &serviceTestRepository{}
	validator := &serviceTestSORValidator{result: false}
	capabilityChecker := &serviceTestCapabilityChecker{entitled: true}
	approver := &serviceTestApprovalRequester{}
	service := NewService(repo, validator, capabilityChecker, approver, zap.NewNop())

	_, err := service.CreateImportRequest(context.Background(), CreateImportRequestInput{
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-123",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/org/app:1.0.0",
	})
	if !errors.Is(err, ErrSORRegistrationRequired) {
		t.Fatalf("expected ErrSORRegistrationRequired, got %v", err)
	}
	if len(repo.created) != 0 {
		t.Fatalf("expected no create call when SOR validation fails")
	}
	if capabilityChecker.callCount != 1 {
		t.Fatalf("expected capability checker to be called once, got %d", capabilityChecker.callCount)
	}
	if validator.callCount != 1 {
		t.Fatalf("expected SOR validator to be called once, got %d", validator.callCount)
	}
	if approver.callCount != 0 {
		t.Fatalf("expected approval requester to not be called when SOR validation fails")
	}
}

func TestServiceCreateImportRequest_Success(t *testing.T) {
	repo := &serviceTestRepository{}
	validator := &serviceTestSORValidator{result: true}
	service := NewService(repo, validator, &serviceTestCapabilityChecker{entitled: true}, nil, zap.NewNop())

	created, err := service.CreateImportRequest(context.Background(), CreateImportRequestInput{
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-123",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/org/app:1.0.0",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if created == nil {
		t.Fatalf("expected created import request")
	}
	if created.Status != StatusPending {
		t.Fatalf("expected status %s, got %s", StatusPending, created.Status)
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected one create call, got %d", len(repo.created))
	}
}

func TestServiceCreateImportRequest_ReturnsCapabilityDenied(t *testing.T) {
	repo := &serviceTestRepository{}
	validator := &serviceTestSORValidator{result: true}
	capabilityChecker := &serviceTestCapabilityChecker{entitled: false}
	approver := &serviceTestApprovalRequester{}
	service := NewService(repo, validator, capabilityChecker, approver, zap.NewNop())

	_, err := service.CreateImportRequest(context.Background(), CreateImportRequestInput{
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-123",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/org/app:1.0.0",
	})
	if !errors.Is(err, ErrOperationNotEntitled) {
		t.Fatalf("expected ErrOperationNotEntitled, got %v", err)
	}
	if len(repo.created) != 0 {
		t.Fatalf("expected no create call when capability is denied")
	}
	if capabilityChecker.callCount != 1 {
		t.Fatalf("expected capability checker to be called once, got %d", capabilityChecker.callCount)
	}
	if validator.callCount != 0 {
		t.Fatalf("expected SOR validator to not be called when capability is denied")
	}
	if approver.callCount != 0 {
		t.Fatalf("expected approval requester to not be called when capability is denied")
	}
}

func TestServiceCreateImportRequest_CallsApprovalRequester(t *testing.T) {
	repo := &serviceTestRepository{}
	validator := &serviceTestSORValidator{result: true}
	approver := &serviceTestApprovalRequester{}
	service := NewService(repo, validator, &serviceTestCapabilityChecker{entitled: true}, approver, zap.NewNop())

	_, err := service.CreateImportRequest(context.Background(), CreateImportRequestInput{
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-123",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/org/app:1.0.0",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !approver.called {
		t.Fatalf("expected approval requester to be called")
	}
	if approver.callCount != 1 {
		t.Fatalf("expected approval requester to be called once, got %d", approver.callCount)
	}
	if validator.callCount != 1 {
		t.Fatalf("expected SOR validator to be called once, got %d", validator.callCount)
	}
}

func TestServiceRetryImportRequest_ReturnsCapabilityDeniedWithoutSideEffects(t *testing.T) {
	repo := &serviceTestRepository{byID: map[uuid.UUID]*ImportRequest{}}
	tenantID := uuid.New()
	existing := &ImportRequest{
		ID:                uuid.New(),
		TenantID:          tenantID,
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-123",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/org/app:1.0.0",
		Status:            StatusFailed,
	}
	repo.byID[existing.ID] = existing
	validator := &serviceTestSORValidator{result: true}
	capabilityChecker := &serviceTestCapabilityChecker{entitled: false}
	approver := &serviceTestApprovalRequester{}
	service := NewService(repo, validator, capabilityChecker, approver, zap.NewNop())

	_, err := service.RetryImportRequest(context.Background(), tenantID, uuid.New(), existing.ID)
	if !errors.Is(err, ErrOperationNotEntitled) {
		t.Fatalf("expected ErrOperationNotEntitled, got %v", err)
	}
	if len(repo.created) != 0 {
		t.Fatalf("expected no create call when retry capability is denied")
	}
	if capabilityChecker.callCount != 1 {
		t.Fatalf("expected capability checker call count=1, got %d", capabilityChecker.callCount)
	}
	if validator.callCount != 0 {
		t.Fatalf("expected sor validator call count=0, got %d", validator.callCount)
	}
	if approver.callCount != 0 {
		t.Fatalf("expected approval requester call count=0, got %d", approver.callCount)
	}
}

func TestServiceRetryImportRequest_ReturnsSORRequiredWithoutSideEffects(t *testing.T) {
	repo := &serviceTestRepository{byID: map[uuid.UUID]*ImportRequest{}}
	tenantID := uuid.New()
	existing := &ImportRequest{
		ID:                uuid.New(),
		TenantID:          tenantID,
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-123",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/org/app:1.0.0",
		Status:            StatusFailed,
	}
	repo.byID[existing.ID] = existing
	validator := &serviceTestSORValidator{result: false}
	capabilityChecker := &serviceTestCapabilityChecker{entitled: true}
	approver := &serviceTestApprovalRequester{}
	service := NewService(repo, validator, capabilityChecker, approver, zap.NewNop())

	_, err := service.RetryImportRequest(context.Background(), tenantID, uuid.New(), existing.ID)
	if !errors.Is(err, ErrSORRegistrationRequired) {
		t.Fatalf("expected ErrSORRegistrationRequired, got %v", err)
	}
	if len(repo.created) != 0 {
		t.Fatalf("expected no create call when retry SOR validation fails")
	}
	if capabilityChecker.callCount != 1 {
		t.Fatalf("expected capability checker call count=1, got %d", capabilityChecker.callCount)
	}
	if validator.callCount != 1 {
		t.Fatalf("expected sor validator call count=1, got %d", validator.callCount)
	}
	if approver.callCount != 0 {
		t.Fatalf("expected approval requester call count=0, got %d", approver.callCount)
	}
}

func TestServiceRetryImportRequest_BackoffActive(t *testing.T) {
	repo := &serviceTestRepository{byID: map[uuid.UUID]*ImportRequest{}}
	tenantID := uuid.New()
	existing := &ImportRequest{
		ID:                uuid.New(),
		TenantID:          tenantID,
		RequestedByUserID: uuid.New(),
		RequestType:       RequestTypeQuarantine,
		SORRecordID:       "APP-123",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/org/app:1.0.0",
		Status:            StatusFailed,
		ErrorMessage:      "dispatch_failed: context deadline exceeded",
		UpdatedAt:         time.Now().UTC(),
	}
	repo.byID[existing.ID] = existing
	service := NewService(repo, &serviceTestSORValidator{result: true}, &serviceTestCapabilityChecker{entitled: true}, nil, zap.NewNop())

	_, err := service.RetryImportRequest(context.Background(), tenantID, uuid.New(), existing.ID)
	var backoffErr *RetryBackoffError
	if !errors.As(err, &backoffErr) {
		t.Fatalf("expected RetryBackoffError, got %v", err)
	}
	if backoffErr.Remaining <= 0 {
		t.Fatalf("expected positive backoff remaining, got %v", backoffErr.Remaining)
	}
	if len(repo.created) != 0 {
		t.Fatalf("expected no create call while backoff is active")
	}
}

func TestServiceRetryImportRequest_AttemptLimitReached(t *testing.T) {
	repo := &serviceTestRepository{byID: map[uuid.UUID]*ImportRequest{}}
	tenantID := uuid.New()
	now := time.Now().UTC().Add(-2 * time.Hour)
	for i := 0; i < 5; i++ {
		id := uuid.New()
		repo.byID[id] = &ImportRequest{
			ID:                id,
			TenantID:          tenantID,
			RequestedByUserID: uuid.New(),
			RequestType:       RequestTypeQuarantine,
			SORRecordID:       "APP-123",
			SourceRegistry:    "ghcr.io",
			SourceImageRef:    "ghcr.io/org/app:1.0.0",
			Status:            StatusFailed,
			ErrorMessage:      "dispatch_failed: context deadline exceeded",
			UpdatedAt:         now,
		}
	}

	targetID := uuid.Nil
	for id := range repo.byID {
		targetID = id
		break
	}
	service := NewService(repo, &serviceTestSORValidator{result: true}, &serviceTestCapabilityChecker{entitled: true}, nil, zap.NewNop())

	_, err := service.RetryImportRequest(context.Background(), tenantID, uuid.New(), targetID)
	var limitErr *RetryAttemptLimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("expected RetryAttemptLimitError, got %v", err)
	}
	if limitErr.MaxAttempts != 5 {
		t.Fatalf("expected max attempts 5 for dispatch class, got %d", limitErr.MaxAttempts)
	}
	if len(repo.created) != 0 {
		t.Fatalf("expected no create call when attempt limit reached")
	}
}

func TestServiceMarkImportReleased_Success(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	importID := uuid.New()
	repo := &serviceTestRepository{
		byID: map[uuid.UUID]*ImportRequest{
			importID: {
				ID:                 importID,
				TenantID:           tenantID,
				RequestedByUserID:  uuid.New(),
				RequestType:        RequestTypeQuarantine,
				Status:             StatusSuccess,
				PolicyDecision:     "pass",
				PolicySnapshotJSON: `{"decision":"pass"}`,
				ScanSummaryJSON:    `{"critical":0}`,
				SBOMSummaryJSON:    `{"packages":42}`,
				SourceImageDigest:  "sha256:abcd",
				UpdatedAt:          time.Now().UTC(),
			},
		},
	}
	service := NewService(repo, nil, nil, nil, zap.NewNop())

	updated, err := service.MarkImportReleased(context.Background(), tenantID, importID, actorID, "security-approved")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.ReleaseState != ReleaseStateReleased {
		t.Fatalf("expected release state released, got %q", updated.ReleaseState)
	}
	if updated.ReleaseActorUserID == nil || *updated.ReleaseActorUserID != actorID {
		t.Fatalf("expected release actor to be persisted")
	}
	if updated.ReleaseRequestedAt == nil || updated.ReleasedAt == nil {
		t.Fatalf("expected release timestamps to be set")
	}
}

func TestServiceMarkImportReleased_NotEligible(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	importID := uuid.New()
	repo := &serviceTestRepository{
		byID: map[uuid.UUID]*ImportRequest{
			importID: {
				ID:                importID,
				TenantID:          tenantID,
				RequestedByUserID: uuid.New(),
				RequestType:       RequestTypeQuarantine,
				Status:            StatusPending,
			},
		},
	}
	service := NewService(repo, nil, nil, nil, zap.NewNop())

	_, err := service.MarkImportReleased(context.Background(), tenantID, importID, actorID, "")
	if !errors.Is(err, ErrReleaseNotEligible) {
		t.Fatalf("expected ErrReleaseNotEligible, got %v", err)
	}
}

func TestServiceMarkImportReleased_IdempotentWhenAlreadyReleased(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	importID := uuid.New()
	releasedAt := time.Now().UTC()
	repo := &serviceTestRepository{
		byID: map[uuid.UUID]*ImportRequest{
			importID: {
				ID:                 importID,
				TenantID:           tenantID,
				RequestedByUserID:  uuid.New(),
				RequestType:        RequestTypeQuarantine,
				Status:             StatusSuccess,
				ReleaseState:       ReleaseStateReleased,
				PolicyDecision:     "pass",
				PolicySnapshotJSON: `{"decision":"pass"}`,
				ScanSummaryJSON:    `{"critical":0}`,
				SBOMSummaryJSON:    `{"packages":42}`,
				SourceImageDigest:  "sha256:abcd",
				UpdatedAt:          time.Now().UTC(),
				ReleasedAt:         &releasedAt,
			},
		},
	}
	service := NewService(repo, nil, nil, nil, zap.NewNop())

	updated, err := service.MarkImportReleased(context.Background(), tenantID, importID, actorID, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.ReleaseState != ReleaseStateReleased {
		t.Fatalf("expected release state released, got %q", updated.ReleaseState)
	}
}

func TestServiceListReleasedArtifacts_RejectsInvalidTenant(t *testing.T) {
	service := NewService(&serviceTestRepository{}, nil, nil, nil, zap.NewNop())

	_, _, err := service.ListReleasedArtifacts(context.Background(), uuid.Nil, "", 20, 0)
	if !errors.Is(err, ErrInvalidTenantID) {
		t.Fatalf("expected ErrInvalidTenantID, got %v", err)
	}
}

func TestServiceListReleasedArtifacts_PassesValidatedInputs(t *testing.T) {
	tenantID := uuid.New()
	repo := &serviceTestRepository{
		releasedArtifacts: []*ReleasedArtifact{{ID: uuid.New(), TenantID: tenantID, RequestedByUserID: uuid.New()}},
		releasedTotal:     1,
	}
	service := NewService(repo, nil, nil, nil, zap.NewNop())

	items, total, err := service.ListReleasedArtifacts(context.Background(), tenantID, "nginx", 999, -5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(items) != 1 || total != 1 {
		t.Fatalf("expected one released artifact and total=1, got items=%d total=%d", len(items), total)
	}
	if repo.lastReleasedTenant != tenantID {
		t.Fatalf("expected tenant %s, got %s", tenantID, repo.lastReleasedTenant)
	}
	if repo.lastReleasedSearch != "nginx" {
		t.Fatalf("expected search to pass through, got %q", repo.lastReleasedSearch)
	}
	if repo.lastReleasedLimit != 200 {
		t.Fatalf("expected limit clamp to 200, got %d", repo.lastReleasedLimit)
	}
	if repo.lastReleasedOffset != 0 {
		t.Fatalf("expected offset clamp to 0, got %d", repo.lastReleasedOffset)
	}
}
