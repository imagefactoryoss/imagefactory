package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appdispatcher "github.com/srikarm/image-factory/internal/application/dispatcher"
	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
)

type stubDispatcherController struct {
	running bool
}

func (s *stubDispatcherController) Start(ctx context.Context) bool {
	s.running = true
	return true
}

func (s *stubDispatcherController) Stop() bool {
	s.running = false
	return true
}

func (s *stubDispatcherController) Status() bool {
	return s.running
}

type stubDispatcherRuntimeReader struct {
	byMode map[string]*appdispatcher.RuntimeStatus
}

func (s *stubDispatcherRuntimeReader) GetLatestRuntimeStatus(ctx context.Context, mode string) (*appdispatcher.RuntimeStatus, error) {
	if s.byMode == nil {
		return nil, nil
	}
	return s.byMode[mode], nil
}

type stubWorkflowDiagnosticsReader struct {
	diag *domainworkflow.BlockedStepDiagnostics
}

func (s *stubWorkflowDiagnosticsReader) GetBlockedStepDiagnostics(ctx context.Context, subjectType string) (*domainworkflow.BlockedStepDiagnostics, error) {
	return s.diag, nil
}

type executionPipelineHealthResponse struct {
	CheckedAt            string                             `json:"checked_at"`
	Components           map[string]pipelineComponentHealth `json:"components"`
	WorkflowControlPlane *workflowControlPlaneDiagnostics   `json:"workflow_control_plane,omitempty"`
}

func TestExecutionPipelineHealth_DispatcherDisabledNoStatus(t *testing.T) {
	handler := NewExecutionPipelineHealthHandler(nil, nil, nil, false, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/execution-pipeline/health", nil)
	w := httptest.NewRecorder()

	handler.GetHealth(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response executionPipelineHealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	dispatcher := response.Components["dispatcher"]
	if dispatcher.Enabled {
		t.Fatalf("expected dispatcher disabled")
	}
	if dispatcher.Running {
		t.Fatalf("expected dispatcher not running")
	}
	if dispatcher.Available {
		t.Fatalf("expected dispatcher unavailable")
	}
}

func TestExecutionPipelineHealth_OrchestratorExplicitlyDisabled(t *testing.T) {
	processStore := runtimehealth.NewStore()
	processStore.Upsert("workflow_orchestrator", runtimehealth.ProcessStatus{
		Enabled:      false,
		Running:      false,
		LastActivity: time.Now().UTC(),
		Message:      "configured disabled",
	})
	processStore.Upsert("stale_execution_watchdog", runtimehealth.ProcessStatus{
		Enabled:      true,
		Running:      true,
		LastActivity: time.Now().UTC(),
		Message:      "healthy",
	})

	handler := NewExecutionPipelineHealthHandler(nil, nil, processStore, true, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/execution-pipeline/health", nil)
	w := httptest.NewRecorder()

	handler.GetHealth(w, req)

	var response executionPipelineHealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	orchestrator := response.Components["workflow_orchestrator"]
	if orchestrator.Enabled {
		t.Fatalf("expected workflow_orchestrator disabled")
	}
	if orchestrator.Running {
		t.Fatalf("expected workflow_orchestrator not running")
	}
	if !orchestrator.Available {
		t.Fatalf("expected workflow_orchestrator available status to be reported")
	}
	if orchestrator.Message != "disabled" {
		t.Fatalf("expected disabled message, got %q", orchestrator.Message)
	}
}

func TestExecutionPipelineHealth_ExternalDispatcherStaleHeartbeat(t *testing.T) {
	stale := time.Now().UTC().Add(-2 * time.Minute)
	reader := &stubDispatcherRuntimeReader{
		byMode: map[string]*appdispatcher.RuntimeStatus{
			appdispatcher.DispatcherModeExternal: {
				Mode:          appdispatcher.DispatcherModeExternal,
				Running:       true,
				LastHeartbeat: stale,
			},
		},
	}
	handler := NewExecutionPipelineHealthHandler(nil, reader, nil, true, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/execution-pipeline/health", nil)
	w := httptest.NewRecorder()
	handler.GetHealth(w, req)

	var response executionPipelineHealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	dispatcher := response.Components["dispatcher"]
	if !dispatcher.Enabled {
		t.Fatalf("expected dispatcher enabled")
	}
	if dispatcher.Running {
		t.Fatalf("expected stale heartbeat to force running=false")
	}
	if dispatcher.Available {
		t.Fatalf("expected stale heartbeat to force available=false")
	}
	if dispatcher.Message != "external dispatcher heartbeat is stale" {
		t.Fatalf("unexpected dispatcher message: %q", dispatcher.Message)
	}
}

func TestExecutionPipelineHealth_PartialRestartMixedState(t *testing.T) {
	controller := &stubDispatcherController{running: true}
	processStore := runtimehealth.NewStore()
	processStore.Upsert("workflow_orchestrator", runtimehealth.ProcessStatus{
		Enabled:      true,
		Running:      false,
		LastActivity: time.Now().UTC().Add(-10 * time.Minute),
		Message:      "process not started",
	})
	processStore.Upsert("stale_execution_watchdog", runtimehealth.ProcessStatus{
		Enabled:      true,
		Running:      true,
		LastActivity: time.Now().UTC(),
		Message:      "healthy",
	})

	handler := NewExecutionPipelineHealthHandler(controller, nil, processStore, true, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/execution-pipeline/health", nil)
	w := httptest.NewRecorder()
	handler.GetHealth(w, req)

	var response executionPipelineHealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	dispatcher := response.Components["dispatcher"]
	if !dispatcher.Running || !dispatcher.Available {
		t.Fatalf("expected embedded dispatcher running and available")
	}

	orchestrator := response.Components["workflow_orchestrator"]
	if !orchestrator.Enabled || orchestrator.Running {
		t.Fatalf("expected orchestrator enabled but not running during partial restart")
	}

	watchdog := response.Components["stale_execution_watchdog"]
	if !watchdog.Enabled || !watchdog.Running {
		t.Fatalf("expected watchdog healthy during partial restart")
	}
}

func TestExecutionPipelineHealth_WorkflowDiagnosticsSuggestStartOrchestrator(t *testing.T) {
	processStore := runtimehealth.NewStore()
	processStore.Upsert("workflow_orchestrator", runtimehealth.ProcessStatus{
		Enabled: false,
		Running: false,
		Message: "disabled",
	})
	processStore.Upsert("stale_execution_watchdog", runtimehealth.ProcessStatus{
		Enabled: true,
		Running: true,
		Message: "healthy",
	})

	diagReader := &stubWorkflowDiagnosticsReader{
		diag: &domainworkflow.BlockedStepDiagnostics{
			SubjectType:      "build",
			BlockedStepCount: 3,
			DispatchBlocked:  2,
			MonitorBlocked:   1,
			OldestBlockedAt:  ptrTime(time.Now().UTC().Add(-3 * time.Minute)),
		},
	}

	handler := NewExecutionPipelineHealthHandler(nil, nil, processStore, true, diagReader)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/execution-pipeline/health", nil)
	w := httptest.NewRecorder()

	handler.GetHealth(w, req)

	var response executionPipelineHealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.WorkflowControlPlane == nil {
		t.Fatal("expected workflow control-plane diagnostics")
	}
	if response.WorkflowControlPlane.BlockedStepCount != 3 {
		t.Fatalf("expected blocked_step_count=3, got %d", response.WorkflowControlPlane.BlockedStepCount)
	}
	if response.WorkflowControlPlane.RecoveryAction != "start_orchestrator" {
		t.Fatalf("expected recovery action start_orchestrator, got %q", response.WorkflowControlPlane.RecoveryAction)
	}
}

func ptrTime(ts time.Time) *time.Time {
	return &ts
}

func TestExecutionPipelineHealth_ComponentMetricsIncluded(t *testing.T) {
	processStore := runtimehealth.NewStore()
	processStore.Upsert("messaging_outbox_relay", runtimehealth.ProcessStatus{
		Enabled: true,
		Running: true,
		Message: "relay healthy",
		Metrics: map[string]int64{
			"messaging_outbox_pending_count":         4,
			"messaging_outbox_replay_success_total":  12,
			"messaging_outbox_replay_failures_total": 2,
		},
	})
	processStore.Upsert("image_catalog_event_subscriber", runtimehealth.ProcessStatus{
		Enabled: true,
		Running: true,
		Message: "catalog subscriber healthy",
		Metrics: map[string]int64{
			"image_catalog_missing_evidence_total": 3,
		},
	})

	handler := NewExecutionPipelineHealthHandler(nil, nil, processStore, true, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/execution-pipeline/health", nil)
	w := httptest.NewRecorder()
	handler.GetHealth(w, req)

	var response executionPipelineHealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	relay := response.Components["messaging_outbox_relay"]
	if relay.Metrics["messaging_outbox_pending_count"] != 4 {
		t.Fatalf("expected pending_count=4, got %d", relay.Metrics["messaging_outbox_pending_count"])
	}
	catalog := response.Components["image_catalog_event_subscriber"]
	if catalog.Metrics["image_catalog_missing_evidence_total"] != 3 {
		t.Fatalf("expected image_catalog_missing_evidence_total=3, got %d", catalog.Metrics["image_catalog_missing_evidence_total"])
	}
}

func TestExecutionPipelineHealth_IncludesTenantAssetDriftWatcherComponent(t *testing.T) {
	processStore := runtimehealth.NewStore()
	processStore.Upsert("tenant_asset_drift_watcher", runtimehealth.ProcessStatus{
		Enabled: true,
		Running: true,
		Message: "drift watcher healthy",
		Metrics: map[string]int64{
			"tenant_asset_drift_namespaces_stale": 2,
		},
	})

	handler := NewExecutionPipelineHealthHandler(nil, nil, processStore, true, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/execution-pipeline/health", nil)
	w := httptest.NewRecorder()
	handler.GetHealth(w, req)

	var response executionPipelineHealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	component, ok := response.Components["tenant_asset_drift_watcher"]
	if !ok {
		t.Fatalf("expected tenant_asset_drift_watcher component in health response")
	}
	if !component.Enabled || !component.Running {
		t.Fatalf("expected enabled+running watcher component, got %+v", component)
	}
	if component.Metrics["tenant_asset_drift_namespaces_stale"] != 2 {
		t.Fatalf("expected stale namespace metric=2, got %d", component.Metrics["tenant_asset_drift_namespaces_stale"])
	}
}

func TestExecutionPipelineHealth_TenantAssetDriftWatcherDisabledState(t *testing.T) {
	processStore := runtimehealth.NewStore()
	processStore.Upsert("tenant_asset_drift_watcher", runtimehealth.ProcessStatus{
		Enabled: false,
		Running: false,
		Message: "tenant asset drift watcher disabled via runtime_services config",
	})

	handler := NewExecutionPipelineHealthHandler(nil, nil, processStore, true, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/execution-pipeline/health", nil)
	w := httptest.NewRecorder()
	handler.GetHealth(w, req)

	var response executionPipelineHealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	component, ok := response.Components["tenant_asset_drift_watcher"]
	if !ok {
		t.Fatalf("expected tenant_asset_drift_watcher component in health response")
	}
	if component.Enabled {
		t.Fatalf("expected disabled watcher component, got %+v", component)
	}
	if component.Running {
		t.Fatalf("expected non-running disabled watcher component, got %+v", component)
	}
	if component.Message != "disabled" {
		t.Fatalf("expected disabled message, got %q", component.Message)
	}
}

func TestExecutionPipelineHealth_IncludesProviderAndRuntimeDependencyWatchers(t *testing.T) {
	processStore := runtimehealth.NewStore()
	processStore.Upsert("provider_readiness_watcher", runtimehealth.ProcessStatus{
		Enabled: true,
		Running: true,
		Message: "provider readiness watcher healthy",
	})
	processStore.Upsert("runtime_dependency_watcher", runtimehealth.ProcessStatus{
		Enabled: true,
		Running: true,
		Message: "all dependencies healthy",
		Metrics: map[string]int64{
			"runtime_dependency_critical_count": 0,
		},
	})
	processStore.Upsert("internal_registry_gc_worker", runtimehealth.ProcessStatus{
		Enabled: true,
		Running: true,
		Message: "internal registry gc worker healthy",
		Metrics: map[string]int64{
			"last_run_deleted":      5,
			"total_reclaimed_bytes": 1024,
		},
	})

	handler := NewExecutionPipelineHealthHandler(nil, nil, processStore, true, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/execution-pipeline/health", nil)
	w := httptest.NewRecorder()
	handler.GetHealth(w, req)

	var response executionPipelineHealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if component, ok := response.Components["provider_readiness_watcher"]; !ok {
		t.Fatal("expected provider_readiness_watcher component in health response")
	} else if !component.Enabled || !component.Running {
		t.Fatalf("expected provider_readiness_watcher enabled+running, got %+v", component)
	}

	if component, ok := response.Components["runtime_dependency_watcher"]; !ok {
		t.Fatal("expected runtime_dependency_watcher component in health response")
	} else if component.Message == "" {
		t.Fatalf("expected runtime_dependency_watcher message, got %+v", component)
	}

	if component, ok := response.Components["internal_registry_gc_worker"]; !ok {
		t.Fatal("expected internal_registry_gc_worker component in health response")
	} else if component.Metrics["last_run_deleted"] != 5 {
		t.Fatalf("expected internal_registry_gc_worker metric last_run_deleted=5, got %+v", component)
	}
}
