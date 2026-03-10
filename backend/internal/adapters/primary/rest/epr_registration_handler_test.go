package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/eprregistration"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

func TestMapEPRRegistrationResponse_IncludesLifecycleFields(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	expires := now.Add(24 * time.Hour)
	decider := uuid.New()

	req := &eprregistration.Request{
		ID:                    uuid.New(),
		TenantID:              uuid.New(),
		EPRRecordID:           "EPR-TEST-001",
		ProductName:           "Product",
		TechnologyName:        "Tech",
		BusinessJustification: "Need",
		RequestedByUserID:     uuid.New(),
		Status:                eprregistration.StatusApproved,
		LifecycleStatus:       eprregistration.LifecycleStatusExpiring,
		ApprovedAt:            &now,
		ExpiresAt:             &expires,
		LastReviewedAt:        &now,
		SuspensionReason:      "",
		DecidedByUserID:       &decider,
		DecisionReason:        "ok",
		DecidedAt:             &now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	resp := mapEPRRegistrationResponse(req)
	if resp.LifecycleStatus != string(eprregistration.LifecycleStatusExpiring) {
		t.Fatalf("expected lifecycle_status %q, got %q", eprregistration.LifecycleStatusExpiring, resp.LifecycleStatus)
	}
	if resp.ApprovedAt == "" || resp.ExpiresAt == "" || resp.LastReviewedAt == "" {
		t.Fatal("expected approved_at/expires_at/last_reviewed_at to be present")
	}
}

type eprRegistrationRepoStub struct {
	byID map[uuid.UUID]*eprregistration.Request
}

func (r *eprRegistrationRepoStub) Create(ctx context.Context, req *eprregistration.Request) error {
	return nil
}
func (r *eprRegistrationRepoStub) GetByID(ctx context.Context, id uuid.UUID) (*eprregistration.Request, error) {
	req, ok := r.byID[id]
	if !ok {
		return nil, eprregistration.ErrNotFound
	}
	copyReq := *req
	return &copyReq, nil
}
func (r *eprRegistrationRepoStub) ListByTenant(ctx context.Context, tenantID uuid.UUID, status *eprregistration.Status, limit, offset int) ([]*eprregistration.Request, error) {
	return nil, nil
}
func (r *eprRegistrationRepoStub) ListAll(ctx context.Context, status *eprregistration.Status, limit, offset int) ([]*eprregistration.Request, error) {
	return nil, nil
}
func (r *eprRegistrationRepoStub) UpdateDecision(ctx context.Context, req *eprregistration.Request) error {
	return nil
}
func (r *eprRegistrationRepoStub) UpdateLifecycle(ctx context.Context, req *eprregistration.Request) error {
	r.byID[req.ID] = req
	return nil
}
func (r *eprRegistrationRepoStub) TransitionLifecycleStates(ctx context.Context, now time.Time, expiringBefore time.Time) ([]eprregistration.LifecycleTransitionRecord, []eprregistration.LifecycleTransitionRecord, error) {
	return nil, nil, nil
}
func (r *eprRegistrationRepoStub) HasApprovedRegistration(ctx context.Context, tenantID uuid.UUID, eprRecordID string) (bool, error) {
	return false, nil
}
func (r *eprRegistrationRepoStub) GetApprovedRegistrationLifecycleStatus(ctx context.Context, tenantID uuid.UUID, eprRecordID string) (*eprregistration.LifecycleStatus, error) {
	return nil, nil
}

func TestEPRRegistrationHandlerSuspendRequest_Success(t *testing.T) {
	now := time.Now().UTC()
	reqID := uuid.New()
	repo := &eprRegistrationRepoStub{
		byID: map[uuid.UUID]*eprregistration.Request{
			reqID: {
				ID:                reqID,
				TenantID:          uuid.New(),
				EPRRecordID:       "EPR-100",
				ProductName:       "Product",
				TechnologyName:    "Tech",
				RequestedByUserID: uuid.New(),
				Status:            eprregistration.StatusApproved,
				LifecycleStatus:   eprregistration.LifecycleStatusActive,
				CreatedAt:         now,
				UpdatedAt:         now,
			},
		},
	}
	svc := eprregistration.NewService(repo, zap.NewNop())
	handler := NewEPRRegistrationHandler(svc, zap.NewNop())

	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/epr/registration-requests/"+reqID.String()+"/suspend", nil)
	request = request.WithContext(context.WithValue(request.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
	}))
	request = withEPRURLParam(request, "id", reqID.String())
	recorder := httptest.NewRecorder()

	handler.SuspendRequest(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, recorder.Code)
	}
	if repo.byID[reqID].LifecycleStatus != eprregistration.LifecycleStatusSuspended {
		t.Fatalf("expected lifecycle to be suspended, got %q", repo.byID[reqID].LifecycleStatus)
	}
}

func TestEPRRegistrationHandlerSuspendRequest_InvalidTransition(t *testing.T) {
	now := time.Now().UTC()
	reqID := uuid.New()
	repo := &eprRegistrationRepoStub{
		byID: map[uuid.UUID]*eprregistration.Request{
			reqID: {
				ID:                reqID,
				TenantID:          uuid.New(),
				EPRRecordID:       "EPR-101",
				ProductName:       "Product",
				TechnologyName:    "Tech",
				RequestedByUserID: uuid.New(),
				Status:            eprregistration.StatusPending,
				LifecycleStatus:   eprregistration.LifecycleStatusActive,
				CreatedAt:         now,
				UpdatedAt:         now,
			},
		},
	}
	svc := eprregistration.NewService(repo, zap.NewNop())
	handler := NewEPRRegistrationHandler(svc, zap.NewNop())

	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/epr/registration-requests/"+reqID.String()+"/suspend", nil)
	request = request.WithContext(context.WithValue(request.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
	}))
	request = withEPRURLParam(request, "id", reqID.String())
	recorder := httptest.NewRecorder()

	handler.SuspendRequest(recorder, request)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, recorder.Code)
	}
}

func TestEPRRegistrationHandlerBulkSuspendRequests_Success(t *testing.T) {
	now := time.Now().UTC()
	reqID1 := uuid.New()
	reqID2 := uuid.New()
	repo := &eprRegistrationRepoStub{
		byID: map[uuid.UUID]*eprregistration.Request{
			reqID1: {
				ID:                reqID1,
				TenantID:          uuid.New(),
				EPRRecordID:       "EPR-300",
				ProductName:       "Product",
				TechnologyName:    "Tech",
				RequestedByUserID: uuid.New(),
				Status:            eprregistration.StatusApproved,
				LifecycleStatus:   eprregistration.LifecycleStatusActive,
				CreatedAt:         now,
				UpdatedAt:         now,
			},
			reqID2: {
				ID:                reqID2,
				TenantID:          uuid.New(),
				EPRRecordID:       "EPR-301",
				ProductName:       "Product",
				TechnologyName:    "Tech",
				RequestedByUserID: uuid.New(),
				Status:            eprregistration.StatusApproved,
				LifecycleStatus:   eprregistration.LifecycleStatusExpiring,
				CreatedAt:         now,
				UpdatedAt:         now,
			},
		},
	}
	svc := eprregistration.NewService(repo, zap.NewNop())
	handler := NewEPRRegistrationHandler(svc, zap.NewNop())

	payload, _ := json.Marshal(map[string]interface{}{
		"request_ids": []string{reqID1.String(), reqID2.String()},
		"reason":      "bulk suspend",
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/epr/registration-requests/bulk/suspend", bytes.NewReader(payload))
	request = request.WithContext(context.WithValue(request.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
	}))
	recorder := httptest.NewRecorder()

	handler.BulkSuspendRequests(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, recorder.Code)
	}
	if repo.byID[reqID1].LifecycleStatus != eprregistration.LifecycleStatusSuspended || repo.byID[reqID2].LifecycleStatus != eprregistration.LifecycleStatusSuspended {
		t.Fatal("expected both rows to be suspended")
	}
}

func TestEPRRegistrationHandlerBulkSuspendRequests_InvalidID(t *testing.T) {
	repo := &eprRegistrationRepoStub{byID: map[uuid.UUID]*eprregistration.Request{}}
	svc := eprregistration.NewService(repo, zap.NewNop())
	handler := NewEPRRegistrationHandler(svc, zap.NewNop())

	payload, _ := json.Marshal(map[string]interface{}{
		"request_ids": []string{"not-a-uuid"},
		"reason":      "bulk suspend",
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/epr/registration-requests/bulk/suspend", bytes.NewReader(payload))
	request = request.WithContext(context.WithValue(request.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
	}))
	recorder := httptest.NewRecorder()

	handler.BulkSuspendRequests(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func withEPRURLParam(req *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}
