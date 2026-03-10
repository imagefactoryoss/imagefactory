package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	appbuild "github.com/srikarm/image-factory/internal/application/build"
	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/workflow"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

type stubBuildExecutionServiceTrace struct {
	execution *build.BuildExecution
}

func (s *stubBuildExecutionServiceTrace) StartBuild(ctx context.Context, configID uuid.UUID, createdBy uuid.UUID) (*build.BuildExecution, error) {
	return nil, nil
}
func (s *stubBuildExecutionServiceTrace) CancelBuild(ctx context.Context, executionID uuid.UUID) error {
	return nil
}
func (s *stubBuildExecutionServiceTrace) RetryBuild(ctx context.Context, executionID uuid.UUID, createdBy uuid.UUID) (*build.BuildExecution, error) {
	return nil, nil
}
func (s *stubBuildExecutionServiceTrace) GetExecution(ctx context.Context, executionID uuid.UUID) (*build.BuildExecution, error) {
	if s.execution != nil && s.execution.ID == executionID {
		return s.execution, nil
	}
	return nil, build.ErrExecutionNotFound
}
func (s *stubBuildExecutionServiceTrace) GetBuildExecutions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]build.BuildExecution, int64, error) {
	return nil, 0, nil
}
func (s *stubBuildExecutionServiceTrace) ListRunningExecutions(ctx context.Context) ([]build.BuildExecution, error) {
	return nil, nil
}
func (s *stubBuildExecutionServiceTrace) GetLogs(ctx context.Context, executionID uuid.UUID, limit, offset int) ([]build.ExecutionLog, int64, error) {
	return nil, 0, nil
}
func (s *stubBuildExecutionServiceTrace) AddLog(ctx context.Context, executionID uuid.UUID, level build.LogLevel, message string, metadata []byte) error {
	return nil
}
func (s *stubBuildExecutionServiceTrace) UpdateExecutionStatus(ctx context.Context, executionID uuid.UUID, status build.ExecutionStatus) error {
	return nil
}
func (s *stubBuildExecutionServiceTrace) UpdateExecutionMetadata(ctx context.Context, executionID uuid.UUID, metadata []byte) error {
	return nil
}
func (s *stubBuildExecutionServiceTrace) TryAcquireMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
	return false, nil
}
func (s *stubBuildExecutionServiceTrace) RenewMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
	return false, nil
}
func (s *stubBuildExecutionServiceTrace) ReleaseMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string) error {
	return nil
}
func (s *stubBuildExecutionServiceTrace) CompleteExecution(ctx context.Context, executionID uuid.UUID, success bool, errorMsg string, artifacts []byte) error {
	return nil
}
func (s *stubBuildExecutionServiceTrace) CleanupOldExecutions(ctx context.Context, olderThan time.Duration) error {
	return nil
}

func buildForTraceTests(buildID, tenantID uuid.UUID) *build.Build {
	now := time.Now().UTC().Add(-5 * time.Minute)
	manifest := build.BuildManifest{
		Name: "trace-build",
		Type: build.BuildTypeContainer,
	}
	return build.NewBuildFromDB(buildID, tenantID, uuid.New(), manifest, build.BuildStatusRunning, now, now, nil)
}

func requestWithAuthAndBuildID(path string, buildID uuid.UUID, authCtx *middleware.AuthContext) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", buildID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), "auth", authCtx))
	return req
}

func TestGetTrace_SelectsLatestExecutionAndIncludesRuntimeForAdmin(t *testing.T) {
	buildID := uuid.New()
	tenantID := uuid.New()
	execID := uuid.New()
	now := time.Now().UTC()

	repo := &mockWorkflowRepository{
		instancesBySubject: map[string]*workflow.Instance{
			"build:" + buildID.String(): {
				ID:        uuid.New(),
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

	h := NewBuildHandler(nil, appbuild.NewService(&stubDomainBuildServiceNoop{}, zap.NewNop()), &stubBuildExecutionServiceTrace{
		execution: &build.BuildExecution{ID: execID, BuildID: buildID},
	}, repo, nil, nil, zap.NewNop())
	h.SetTraceReadOverrides(
		func(ctx context.Context, id uuid.UUID) (*build.Build, error) {
			return buildForTraceTests(id, tenantID), nil
		},
		func(ctx context.Context, id uuid.UUID, limit, offset int) ([]build.BuildExecution, int64, error) {
			return []build.BuildExecution{
				{
					ID:        execID,
					BuildID:   buildID,
					Status:    build.ExecutionRunning,
					CreatedAt: now,
					UpdatedAt: now,
				},
			}, 1, nil
		},
	)
	processStore := runtimehealth.NewStore()
	processStore.Upsert("dispatcher", runtimehealth.ProcessStatus{
		Enabled:      true,
		Running:      true,
		LastActivity: now,
		Message:      "running",
	})
	h.SetProcessStatusProvider(processStore)

	authCtx := &middleware.AuthContext{
		UserID:        uuid.New(),
		TenantID:      tenantID,
		UserTenants:   []uuid.UUID{tenantID},
		IsSystemAdmin: true,
	}
	req := requestWithAuthAndBuildID("/api/v1/builds/"+buildID.String()+"/trace", buildID, authCtx)
	w := httptest.NewRecorder()

	h.GetTrace(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp BuildTraceResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, buildID.String(), resp.Build.ID)
	require.Equal(t, execID.String(), resp.SelectedExecutionID)
	require.Len(t, resp.Executions, 1)
	require.NotNil(t, resp.Runtime)
	require.Contains(t, resp.Runtime, "dispatcher")
	require.NotNil(t, resp.Correlation)
	require.NotEmpty(t, resp.Correlation.WorkflowInstanceID)
	require.Equal(t, execID.String(), resp.Correlation.ExecutionID)
	require.Equal(t, "build.dispatch", resp.Correlation.ActiveStepKey)
}

func TestGetTrace_HidesRuntimeForNonAdmin(t *testing.T) {
	buildID := uuid.New()
	tenantID := uuid.New()
	execID := uuid.New()
	now := time.Now().UTC()

	h := NewBuildHandler(nil, appbuild.NewService(&stubDomainBuildServiceNoop{}, zap.NewNop()), &stubBuildExecutionServiceTrace{
		execution: &build.BuildExecution{ID: execID, BuildID: buildID},
	}, &mockWorkflowRepository{}, nil, nil, zap.NewNop())
	h.SetTraceReadOverrides(
		func(ctx context.Context, id uuid.UUID) (*build.Build, error) {
			return buildForTraceTests(id, tenantID), nil
		},
		func(ctx context.Context, id uuid.UUID, limit, offset int) ([]build.BuildExecution, int64, error) {
			return []build.BuildExecution{
				{
					ID:        execID,
					BuildID:   buildID,
					Status:    build.ExecutionRunning,
					CreatedAt: now,
					UpdatedAt: now,
				},
			}, 1, nil
		},
	)
	processStore := runtimehealth.NewStore()
	processStore.Upsert("dispatcher", runtimehealth.ProcessStatus{
		Enabled:      true,
		Running:      true,
		LastActivity: now,
		Message:      "running",
	})
	h.SetProcessStatusProvider(processStore)

	authCtx := &middleware.AuthContext{
		UserID:        uuid.New(),
		TenantID:      tenantID,
		UserTenants:   []uuid.UUID{tenantID},
		IsSystemAdmin: false,
	}
	req := requestWithAuthAndBuildID("/api/v1/builds/"+buildID.String()+"/trace", buildID, authCtx)
	w := httptest.NewRecorder()

	h.GetTrace(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp BuildTraceResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Nil(t, resp.Runtime)
}

func TestGetTraceExport_ReturnsAttachmentJSON(t *testing.T) {
	buildID := uuid.New()
	tenantID := uuid.New()
	execID := uuid.New()
	now := time.Now().UTC()

	h := NewBuildHandler(nil, appbuild.NewService(&stubDomainBuildServiceNoop{}, zap.NewNop()), &stubBuildExecutionServiceTrace{
		execution: &build.BuildExecution{ID: execID, BuildID: buildID},
	}, &mockWorkflowRepository{}, nil, nil, zap.NewNop())
	h.SetTraceReadOverrides(
		func(ctx context.Context, id uuid.UUID) (*build.Build, error) {
			return buildForTraceTests(id, tenantID), nil
		},
		func(ctx context.Context, id uuid.UUID, limit, offset int) ([]build.BuildExecution, int64, error) {
			return []build.BuildExecution{
				{
					ID:        execID,
					BuildID:   buildID,
					Status:    build.ExecutionRunning,
					CreatedAt: now,
					UpdatedAt: now,
				},
			}, 1, nil
		},
	)

	authCtx := &middleware.AuthContext{
		UserID:      uuid.New(),
		TenantID:    tenantID,
		UserTenants: []uuid.UUID{tenantID},
	}
	req := requestWithAuthAndBuildID("/api/v1/builds/"+buildID.String()+"/trace/export", buildID, authCtx)
	w := httptest.NewRecorder()

	h.GetTraceExport(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Disposition"), "attachment;")
	require.Contains(t, w.Header().Get("Content-Disposition"), "build-trace-"+buildID.String()+".json")

	var resp BuildTraceResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, buildID.String(), resp.Build.ID)
}

func TestGetTrace_IncludesRepoConfigDiagnostics(t *testing.T) {
	buildID := uuid.New()
	tenantID := uuid.New()
	now := time.Now().UTC()

	manifest := build.BuildManifest{
		Name: "trace-build",
		Type: build.BuildTypeKaniko,
		Metadata: map[string]interface{}{
			"repo_config_applied":     false,
			"repo_config_path":        "image-factory.yaml",
			"repo_config_ref":         "main",
			"repo_config_error_stage": "start",
			"repo_config_error_at":    now.Format(time.RFC3339),
			"repo_config_error":       "invalid repo build config image-factory.yaml: unsupported version \"v99\"",
		},
	}

	h := NewBuildHandler(nil, appbuild.NewService(&stubDomainBuildServiceNoop{}, zap.NewNop()), &stubBuildExecutionServiceTrace{}, &mockWorkflowRepository{}, nil, nil, zap.NewNop())
	h.SetTraceReadOverrides(
		func(ctx context.Context, id uuid.UUID) (*build.Build, error) {
			return build.NewBuildFromDB(id, tenantID, uuid.New(), manifest, build.BuildStatusFailed, now.Add(-time.Minute), now.Add(-time.Minute), nil), nil
		},
		func(ctx context.Context, id uuid.UUID, limit, offset int) ([]build.BuildExecution, int64, error) {
			return nil, 0, nil
		},
	)

	authCtx := &middleware.AuthContext{
		UserID:      uuid.New(),
		TenantID:    tenantID,
		UserTenants: []uuid.UUID{tenantID},
	}
	req := requestWithAuthAndBuildID("/api/v1/builds/"+buildID.String()+"/trace", buildID, authCtx)
	w := httptest.NewRecorder()

	h.GetTrace(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp BuildTraceResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Diagnostics)
	require.NotNil(t, resp.Diagnostics.RepoConfig)
	require.Equal(t, "image-factory.yaml", resp.Diagnostics.RepoConfig.Path)
	require.Equal(t, "start", resp.Diagnostics.RepoConfig.Stage)
	require.Equal(t, "repo_build_config_invalid", resp.Diagnostics.RepoConfig.ErrorCode)
}
