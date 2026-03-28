package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	appsresmartbot "github.com/srikarm/image-factory/internal/application/sresmartbot"
	"github.com/srikarm/image-factory/internal/domain/sresmartbot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type stubSRERepository struct {
	sresmartbot.Repository
	incident  *sresmartbot.Incident
	err       error
	runs      []*sresmartbot.RemediationPackRun
	approvals []*sresmartbot.Approval
}

func (s *stubSRERepository) GetIncident(ctx context.Context, id uuid.UUID) (*sresmartbot.Incident, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.incident == nil {
		return nil, errors.New("not found")
	}
	return s.incident, nil
}

func (s *stubSRERepository) CreateRemediationPackRun(ctx context.Context, run *sresmartbot.RemediationPackRun) error {
	s.runs = append(s.runs, run)
	return nil
}

func (s *stubSRERepository) ListRemediationPackRunsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*sresmartbot.RemediationPackRun, error) {
	filtered := make([]*sresmartbot.RemediationPackRun, 0)
	for _, run := range s.runs {
		if run != nil && run.IncidentID == incidentID {
			filtered = append(filtered, run)
		}
	}
	return filtered, nil
}

func (s *stubSRERepository) ListApprovalsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*sresmartbot.Approval, error) {
	return s.approvals, nil
}

func TestSRESmartBotHandler_ListRemediationPacks_OK(t *testing.T) {
	h := &SRESmartBotHandler{
		remediationPackService: appsresmartbot.NewRemediationPackService(nil),
		logger:                 zaptest.NewLogger(t),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sre/remediation-packs", nil)
	w := httptest.NewRecorder()

	h.ListRemediationPacks(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body listSRERemediationPacksResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.NotEmpty(t, body.Packs)
	assert.Equal(t, "async_backlog_pressure_pack", body.Packs[0].Key)
}

func TestSRESmartBotHandler_ListRemediationPacks_ServiceMissing(t *testing.T) {
	h := &SRESmartBotHandler{logger: zaptest.NewLogger(t)}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sre/remediation-packs", nil)
	w := httptest.NewRecorder()

	h.ListRemediationPacks(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var errResp ErrorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "INTERNAL_ERROR", errResp.Code)
}

func TestSRESmartBotHandler_ListIncidentRemediationPacks_InvalidIncidentID(t *testing.T) {
	h := &SRESmartBotHandler{
		repo:                   &stubSRERepository{},
		remediationPackService: appsresmartbot.NewRemediationPackService(nil),
		logger:                 zaptest.NewLogger(t),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sre/incidents/not-a-uuid/remediation-packs", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	w := httptest.NewRecorder()

	h.ListIncidentRemediationPacks(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var errResp ErrorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "BAD_REQUEST", errResp.Code)
}

func TestSRESmartBotHandler_ListIncidentRemediationPacks_IncidentNotFound(t *testing.T) {
	h := &SRESmartBotHandler{
		repo:                   &stubSRERepository{err: errors.New("missing")},
		remediationPackService: appsresmartbot.NewRemediationPackService(nil),
		logger:                 zaptest.NewLogger(t),
	}

	incidentID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sre/incidents/"+incidentID.String()+"/remediation-packs", nil)
	req = withChiURLParam(req, "id", incidentID.String())
	w := httptest.NewRecorder()

	h.ListIncidentRemediationPacks(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	var errResp ErrorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "NOT_FOUND", errResp.Code)
}

func TestSRESmartBotHandler_ListIncidentRemediationPacks_OK(t *testing.T) {
	incident := &sresmartbot.Incident{
		ID:           uuid.New(),
		IncidentType: "email_queue_backlog_pressure",
	}
	h := &SRESmartBotHandler{
		repo: &stubSRERepository{
			incident: incident,
		},
		remediationPackService: appsresmartbot.NewRemediationPackService(nil),
		logger:                 zaptest.NewLogger(t),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sre/incidents/"+incident.ID.String()+"/remediation-packs", nil)
	req = withChiURLParam(req, "id", incident.ID.String())
	w := httptest.NewRecorder()

	h.ListIncidentRemediationPacks(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body listSRERemediationPacksResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Len(t, body.Packs, 1)
	assert.Equal(t, "async_backlog_pressure_pack", body.Packs[0].Key)
}

func TestSRESmartBotHandler_DryRunIncidentRemediationPack_OK(t *testing.T) {
	incident := &sresmartbot.Incident{
		ID:           uuid.New(),
		IncidentType: "email_queue_backlog_pressure",
		Status:       sresmartbot.IncidentStatusObserved,
	}
	repo := &stubSRERepository{incident: incident}
	h := &SRESmartBotHandler{
		repo:                   repo,
		remediationPackService: appsresmartbot.NewRemediationPackService(nil),
		logger:                 zaptest.NewLogger(t),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/sre/incidents/"+incident.ID.String()+"/remediation-packs/async_backlog_pressure_pack/dry-run", nil)
	req = withChiURLParams(req, map[string]string{"id": incident.ID.String(), "packKey": "async_backlog_pressure_pack"})
	w := httptest.NewRecorder()

	h.DryRunIncidentRemediationPack(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Len(t, repo.runs, 1)
	assert.Equal(t, "dry_run", repo.runs[0].RunKind)
	assert.Equal(t, "async_backlog_pressure_pack", repo.runs[0].PackKey)
}

func TestSRESmartBotHandler_ExecuteIncidentRemediationPack_RequiresApproval(t *testing.T) {
	incident := &sresmartbot.Incident{
		ID:           uuid.New(),
		IncidentType: "nats_transport_disconnect_storm",
		Status:       sresmartbot.IncidentStatusObserved,
	}
	h := &SRESmartBotHandler{
		repo:                   &stubSRERepository{incident: incident},
		remediationPackService: appsresmartbot.NewRemediationPackService(nil),
		logger:                 zaptest.NewLogger(t),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/sre/incidents/"+incident.ID.String()+"/remediation-packs/nats_transport_stability_pack/execute", bytes.NewBufferString(`{}`))
	req = withChiURLParams(req, map[string]string{"id": incident.ID.String(), "packKey": "nats_transport_stability_pack"})
	w := httptest.NewRecorder()

	h.ExecuteIncidentRemediationPack(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var errResp ErrorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&errResp))
	assert.Equal(t, "BAD_REQUEST", errResp.Code)
}

func TestSRESmartBotHandler_ExecuteIncidentRemediationPack_OK(t *testing.T) {
	incident := &sresmartbot.Incident{
		ID:           uuid.New(),
		IncidentType: "email_queue_backlog_pressure",
		Status:       sresmartbot.IncidentStatusObserved,
	}
	repo := &stubSRERepository{incident: incident}
	h := &SRESmartBotHandler{
		repo:                   repo,
		remediationPackService: appsresmartbot.NewRemediationPackService(nil),
		logger:                 zaptest.NewLogger(t),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/sre/incidents/"+incident.ID.String()+"/remediation-packs/async_backlog_pressure_pack/execute", bytes.NewBufferString(`{"request_id":"req-123"}`))
	req = withChiURLParams(req, map[string]string{"id": incident.ID.String(), "packKey": "async_backlog_pressure_pack"})
	w := httptest.NewRecorder()

	h.ExecuteIncidentRemediationPack(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Len(t, repo.runs, 1)
	assert.Equal(t, "execute", repo.runs[0].RunKind)
	assert.Equal(t, "req-123", repo.runs[0].RequestID)
}

func TestSRESmartBotHandler_ExecuteIncidentRemediationPack_WithApprovedApproval_OK(t *testing.T) {
	incident := &sresmartbot.Incident{
		ID:           uuid.New(),
		IncidentType: "nats_transport_disconnect_storm",
		Status:       sresmartbot.IncidentStatusObserved,
	}
	approvedID := uuid.New()
	repo := &stubSRERepository{
		incident: incident,
		approvals: []*sresmartbot.Approval{
			{ID: approvedID, IncidentID: incident.ID, Status: "approved"},
		},
	}
	h := &SRESmartBotHandler{
		repo:                   repo,
		remediationPackService: appsresmartbot.NewRemediationPackService(nil),
		logger:                 zaptest.NewLogger(t),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/sre/incidents/"+incident.ID.String()+"/remediation-packs/nats_transport_stability_pack/execute", bytes.NewBufferString(`{"approval_id":"`+approvedID.String()+`"}`))
	req = withChiURLParams(req, map[string]string{"id": incident.ID.String(), "packKey": "nats_transport_stability_pack"})
	w := httptest.NewRecorder()

	h.ExecuteIncidentRemediationPack(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Len(t, repo.runs, 1)
	assert.Equal(t, "execute", repo.runs[0].RunKind)
	require.NotNil(t, repo.runs[0].ApprovalID)
	assert.Equal(t, approvedID, *repo.runs[0].ApprovalID)
}

func TestSRESmartBotHandler_ListIncidentRemediationPackRuns_OK(t *testing.T) {
	incidentID := uuid.New()
	repo := &stubSRERepository{
		runs: []*sresmartbot.RemediationPackRun{
			{ID: uuid.New(), IncidentID: incidentID, RunKind: "dry_run"},
			{ID: uuid.New(), IncidentID: incidentID, RunKind: "execute"},
		},
	}
	h := &SRESmartBotHandler{repo: repo, logger: zaptest.NewLogger(t)}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sre/incidents/"+incidentID.String()+"/remediation-packs/runs", nil)
	req = withChiURLParam(req, "id", incidentID.String())
	w := httptest.NewRecorder()

	h.ListIncidentRemediationPackRuns(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body listSRERemediationPackRunsResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Len(t, body.Runs, 2)
}

func withChiURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func withChiURLParams(req *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
