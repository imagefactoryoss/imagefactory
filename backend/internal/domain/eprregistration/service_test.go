package eprregistration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type repoStub struct {
	created            *Request
	byID               map[uuid.UUID]*Request
	approved           map[string]bool
	lifecycleByKey     map[string]*LifecycleStatus
	listTenant         []*Request
	listAll            []*Request
	updateCalled       bool
	transitionExpiring []LifecycleTransitionRecord
	transitionExpired  []LifecycleTransitionRecord
}

func (r *repoStub) Create(ctx context.Context, req *Request) error {
	r.created = req
	if r.byID == nil {
		r.byID = map[uuid.UUID]*Request{}
	}
	r.byID[req.ID] = req
	return nil
}

func (r *repoStub) GetByID(ctx context.Context, id uuid.UUID) (*Request, error) {
	if req, ok := r.byID[id]; ok {
		copy := *req
		return &copy, nil
	}
	return nil, ErrNotFound
}

func (r *repoStub) ListByTenant(ctx context.Context, tenantID uuid.UUID, status *Status, limit, offset int) ([]*Request, error) {
	return r.listTenant, nil
}

func (r *repoStub) ListAll(ctx context.Context, status *Status, limit, offset int) ([]*Request, error) {
	return r.listAll, nil
}

func (r *repoStub) UpdateDecision(ctx context.Context, req *Request) error {
	r.updateCalled = true
	if r.byID == nil {
		r.byID = map[uuid.UUID]*Request{}
	}
	r.byID[req.ID] = req
	return nil
}

func (r *repoStub) UpdateLifecycle(ctx context.Context, req *Request) error {
	r.updateCalled = true
	if r.byID == nil {
		r.byID = map[uuid.UUID]*Request{}
	}
	r.byID[req.ID] = req
	return nil
}

func (r *repoStub) HasApprovedRegistration(ctx context.Context, tenantID uuid.UUID, eprRecordID string) (bool, error) {
	return r.approved[tenantID.String()+":"+eprRecordID], nil
}

func (r *repoStub) GetApprovedRegistrationLifecycleStatus(ctx context.Context, tenantID uuid.UUID, eprRecordID string) (*LifecycleStatus, error) {
	if r.lifecycleByKey == nil {
		return nil, nil
	}
	return r.lifecycleByKey[tenantID.String()+":"+eprRecordID], nil
}

func (r *repoStub) TransitionLifecycleStates(ctx context.Context, now time.Time, expiringBefore time.Time) (expiring []LifecycleTransitionRecord, expired []LifecycleTransitionRecord, err error) {
	return r.transitionExpiring, r.transitionExpired, nil
}

func TestServiceCreateApproveAndLookup(t *testing.T) {
	repo := &repoStub{approved: map[string]bool{}}
	svc := NewService(repo, zap.NewNop())
	tenantID := uuid.New()
	userID := uuid.New()

	created, err := svc.CreateRequest(context.Background(), CreateInput{
		TenantID:              tenantID,
		RequestedByUserID:     userID,
		EPRRecordID:           "SOR-900",
		ProductName:           "Product A",
		TechnologyName:        "nginx",
		BusinessJustification: "needed for baseline",
	})
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	if created.Status != StatusPending {
		t.Fatalf("expected pending status, got %s", created.Status)
	}

	approved, err := svc.Approve(context.Background(), created.ID, uuid.New(), "validated")
	if err != nil {
		t.Fatalf("approve failed: %v", err)
	}
	if approved.Status != StatusApproved {
		t.Fatalf("expected approved status, got %s", approved.Status)
	}
	if !repo.updateCalled {
		t.Fatal("expected decision update call")
	}

	repo.approved[tenantID.String()+":SOR-900"] = true
	ok, err := svc.IsApprovedEPRRegistration(context.Background(), tenantID, "SOR-900")
	if err != nil {
		t.Fatalf("approval lookup failed: %v", err)
	}
	if !ok {
		t.Fatal("expected approved epr registration lookup to be true")
	}
}

func TestServiceRejectAlreadyDecided(t *testing.T) {
	req, err := NewRequest(uuid.New(), uuid.New(), "SOR-1", "Prod", "Tech", "")
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	_ = req.Approve(uuid.New(), "")
	repo := &repoStub{byID: map[uuid.UUID]*Request{req.ID: req}, approved: map[string]bool{}}
	svc := NewService(repo, zap.NewNop())

	_, err = svc.Reject(context.Background(), req.ID, uuid.New(), "no")
	if err == nil {
		t.Fatal("expected reject to fail for already decided request")
	}
	if err != ErrAlreadyDecided {
		t.Fatalf("expected ErrAlreadyDecided, got %v", err)
	}
}

func TestServiceCreateRequest_AutoGeneratesEPRRecordID(t *testing.T) {
	repo := &repoStub{approved: map[string]bool{}}
	svc := NewService(repo, zap.NewNop())

	created, err := svc.CreateRequest(context.Background(), CreateInput{
		TenantID:              uuid.New(),
		RequestedByUserID:     uuid.New(),
		EPRRecordID:           "",
		ProductName:           "Product A",
		TechnologyName:        "Tech A",
		BusinessJustification: "needed",
	})
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	if created.EPRRecordID == "" {
		t.Fatal("expected generated EPR record ID to be non-empty")
	}
	if !strings.HasPrefix(created.EPRRecordID, "EPR-") {
		t.Fatalf("expected generated ID to start with EPR-, got %q", created.EPRRecordID)
	}
}

func TestServiceWithdraw_PendingRequest(t *testing.T) {
	tenantID := uuid.New()
	req, err := NewRequest(tenantID, uuid.New(), "EPR-100", "Prod", "Tech", "")
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	repo := &repoStub{byID: map[uuid.UUID]*Request{req.ID: req}, approved: map[string]bool{}}
	svc := NewService(repo, zap.NewNop())

	updated, err := svc.Withdraw(context.Background(), tenantID, req.ID, uuid.New(), "duplicate submission")
	if err != nil {
		t.Fatalf("withdraw failed: %v", err)
	}
	if updated.Status != StatusWithdrawn {
		t.Fatalf("expected withdrawn after withdraw, got %s", updated.Status)
	}
	if updated.DecisionReason != "duplicate submission" {
		t.Fatalf("expected reason to persist, got %q", updated.DecisionReason)
	}
}

func TestServiceWithdraw_RejectsDifferentTenant(t *testing.T) {
	req, err := NewRequest(uuid.New(), uuid.New(), "EPR-100", "Prod", "Tech", "")
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	repo := &repoStub{byID: map[uuid.UUID]*Request{req.ID: req}, approved: map[string]bool{}}
	svc := NewService(repo, zap.NewNop())

	_, err = svc.Withdraw(context.Background(), uuid.New(), req.ID, uuid.New(), "")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for tenant mismatch, got %v", err)
	}
}

func TestServiceGetApprovedLifecycleStatus(t *testing.T) {
	tenantID := uuid.New()
	lifecycle := LifecycleStatusSuspended
	repo := &repoStub{
		approved:       map[string]bool{},
		lifecycleByKey: map[string]*LifecycleStatus{tenantID.String() + ":EPR-100": &lifecycle},
	}
	svc := NewService(repo, zap.NewNop())

	got, err := svc.GetApprovedEPRRegistrationLifecycleStatus(context.Background(), tenantID, "EPR-100")
	if err != nil {
		t.Fatalf("GetApprovedEPRRegistrationLifecycleStatus failed: %v", err)
	}
	if got == nil || *got != LifecycleStatusSuspended {
		t.Fatalf("expected suspended lifecycle, got %#v", got)
	}
}

func TestServiceSuspendAndReactivateLifecycle(t *testing.T) {
	tenantID := uuid.New()
	req, err := NewRequest(tenantID, uuid.New(), "EPR-200", "Prod", "Tech", "")
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	if err := req.Approve(uuid.New(), "approved"); err != nil {
		t.Fatalf("approve failed: %v", err)
	}

	repo := &repoStub{byID: map[uuid.UUID]*Request{req.ID: req}, approved: map[string]bool{}}
	svc := NewService(repo, zap.NewNop())

	suspended, err := svc.Suspend(context.Background(), req.ID, uuid.New(), "policy hold")
	if err != nil {
		t.Fatalf("suspend failed: %v", err)
	}
	if suspended.LifecycleStatus != LifecycleStatusSuspended {
		t.Fatalf("expected suspended lifecycle, got %s", suspended.LifecycleStatus)
	}

	reactivated, err := svc.Reactivate(context.Background(), req.ID, uuid.New(), "revalidated")
	if err != nil {
		t.Fatalf("reactivate failed: %v", err)
	}
	if reactivated.LifecycleStatus != LifecycleStatusActive {
		t.Fatalf("expected active lifecycle, got %s", reactivated.LifecycleStatus)
	}
}

func TestServiceRunLifecycleTransitions(t *testing.T) {
	repo := &repoStub{
		approved: map[string]bool{},
		transitionExpiring: []LifecycleTransitionRecord{
			{RequestID: uuid.New()},
			{RequestID: uuid.New()},
			{RequestID: uuid.New()},
		},
		transitionExpired: []LifecycleTransitionRecord{
			{RequestID: uuid.New()},
			{RequestID: uuid.New()},
		},
	}
	svc := NewService(repo, zap.NewNop())
	result, err := svc.RunLifecycleTransitions(context.Background(), time.Now().UTC(), 24*time.Hour)
	if err != nil {
		t.Fatalf("RunLifecycleTransitions failed: %v", err)
	}
	if result.ExpiringCount != 3 || result.ExpiredCount != 2 {
		t.Fatalf("unexpected transition result: %+v", result)
	}
}
