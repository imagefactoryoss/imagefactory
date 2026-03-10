package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	appbuild "github.com/srikarm/image-factory/internal/application/build"
	"github.com/srikarm/image-factory/internal/domain/workflow"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

type mockWorkflowRepository struct {
	instancesBySubject map[string]*workflow.Instance
	stepsBySubject     map[string][]workflow.Step
	calls              map[string]int
}

func (m *mockWorkflowRepository) key(subjectType string, subjectID uuid.UUID) string {
	return subjectType + ":" + subjectID.String()
}

func (m *mockWorkflowRepository) ClaimNextRunnableStep(ctx context.Context) (*workflow.Step, error) {
	return nil, errors.New("not implemented")
}

func (m *mockWorkflowRepository) UpdateStep(ctx context.Context, step *workflow.Step) error {
	return errors.New("not implemented")
}

func (m *mockWorkflowRepository) AppendEvent(ctx context.Context, event *workflow.Event) error {
	return errors.New("not implemented")
}

func (m *mockWorkflowRepository) UpsertDefinition(ctx context.Context, name string, version int, definition map[string]interface{}) (uuid.UUID, error) {
	return uuid.Nil, errors.New("not implemented")
}

func (m *mockWorkflowRepository) CreateInstance(ctx context.Context, definitionID uuid.UUID, tenantID *uuid.UUID, subjectType string, subjectID uuid.UUID, status workflow.InstanceStatus) (uuid.UUID, error) {
	return uuid.Nil, errors.New("not implemented")
}

func (m *mockWorkflowRepository) CreateSteps(ctx context.Context, instanceID uuid.UUID, steps []workflow.StepDefinition) error {
	return errors.New("not implemented")
}

func (m *mockWorkflowRepository) UpdateInstanceStatus(ctx context.Context, instanceID uuid.UUID, status workflow.InstanceStatus) error {
	return errors.New("not implemented")
}

func (m *mockWorkflowRepository) UpdateStepStatus(ctx context.Context, instanceID uuid.UUID, stepKey string, status workflow.StepStatus, errMsg *string) error {
	return errors.New("not implemented")
}

func (m *mockWorkflowRepository) GetInstanceWithStepsBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) (*workflow.Instance, []workflow.Step, error) {
	if m.calls == nil {
		m.calls = make(map[string]int)
	}
	m.calls[subjectType]++
	key := m.key(subjectType, subjectID)
	instance, ok := m.instancesBySubject[key]
	if !ok || instance == nil {
		return nil, nil, errors.New("not found")
	}
	return instance, m.stepsBySubject[key], nil
}

func (m *mockWorkflowRepository) GetBlockedStepDiagnostics(ctx context.Context, subjectType string) (*workflow.BlockedStepDiagnostics, error) {
	return &workflow.BlockedStepDiagnostics{SubjectType: subjectType}, nil
}

func TestGetWorkflow_UsesBuildSubjectOnly(t *testing.T) {
	buildID := uuid.New()
	instanceID := uuid.New()
	now := time.Now().UTC()

	repo := &mockWorkflowRepository{
		instancesBySubject: map[string]*workflow.Instance{
			"build:" + buildID.String(): {
				ID:        instanceID,
				Status:    workflow.InstanceStatusRunning,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		stepsBySubject: map[string][]workflow.Step{
			"build:" + buildID.String(): {
				{
					ID:        uuid.New(),
					StepKey:   "build.dispatch",
					Status:    workflow.StepStatusRunning,
					Attempts:  1,
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
	}

	h := NewBuildHandler(nil, appbuild.NewService(&stubDomainBuildServiceNoop{}, zap.NewNop()), nil, repo, nil, nil, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/builds/"+buildID.String()+"/workflow", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", buildID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{UserID: uuid.New(), TenantID: uuid.New()}))
	w := httptest.NewRecorder()

	h.GetWorkflow(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var response BuildWorkflowResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, instanceID.String(), response.InstanceID)
	require.Equal(t, string(workflow.InstanceStatusRunning), response.Status)
	require.Len(t, response.Steps, 1)
	require.Equal(t, "build.dispatch", response.Steps[0].StepKey)
	require.Equal(t, 1, repo.calls["build"])
	require.Equal(t, 0, repo.calls["build_execution"])
}

func TestGetWorkflow_DoesNotFallbackToBuildExecutionSubject(t *testing.T) {
	buildID := uuid.New()
	executionID := uuid.New()
	now := time.Now().UTC()

	repo := &mockWorkflowRepository{
		instancesBySubject: map[string]*workflow.Instance{
			"build_execution:" + executionID.String(): {
				ID:        uuid.New(),
				Status:    workflow.InstanceStatusRunning,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		stepsBySubject: map[string][]workflow.Step{
			"build_execution:" + executionID.String(): {
				{
					ID:        uuid.New(),
					StepKey:   "queue_build",
					Status:    workflow.StepStatusSucceeded,
					Attempts:  1,
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
	}

	h := NewBuildHandler(nil, appbuild.NewService(&stubDomainBuildServiceNoop{}, zap.NewNop()), nil, repo, nil, nil, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/builds/"+buildID.String()+"/workflow?execution_id="+executionID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", buildID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{UserID: uuid.New(), TenantID: uuid.New()}))
	w := httptest.NewRecorder()

	h.GetWorkflow(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var response BuildWorkflowResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Empty(t, response.InstanceID)
	require.Empty(t, response.Status)
	require.Empty(t, response.Steps)
	require.Equal(t, 1, repo.calls["build"])
	require.Equal(t, 0, repo.calls["build_execution"])
}
