package rest

import (
	"context"
	"net/http"
	"reflect"
	"time"

	appdispatcher "github.com/srikarm/image-factory/internal/application/dispatcher"
	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
)

// ExecutionPipelineHealthHandler provides runtime visibility for execution pipeline background components.
type ExecutionPipelineHealthHandler struct {
	dispatcherController DispatcherController
	runtimeReader        DispatcherRuntimeReader
	processStatus        runtimehealth.Provider
	dispatcherEnabled    bool
	workflowRepo         WorkflowDiagnosticsReader
}

// WorkflowDiagnosticsReader provides control-plane workflow blockage diagnostics.
type WorkflowDiagnosticsReader interface {
	GetBlockedStepDiagnostics(ctx context.Context, subjectType string) (*domainworkflow.BlockedStepDiagnostics, error)
}

func NewExecutionPipelineHealthHandler(
	dispatcherController DispatcherController,
	runtimeReader DispatcherRuntimeReader,
	processStatus runtimehealth.Provider,
	dispatcherEnabled bool,
	workflowRepo WorkflowDiagnosticsReader,
) *ExecutionPipelineHealthHandler {
	return &ExecutionPipelineHealthHandler{
		dispatcherController: dispatcherController,
		runtimeReader:        runtimeReader,
		processStatus:        processStatus,
		dispatcherEnabled:    dispatcherEnabled,
		workflowRepo:         workflowRepo,
	}
}

type pipelineComponentHealth struct {
	Enabled      bool             `json:"enabled"`
	Running      bool             `json:"running"`
	Available    bool             `json:"available"`
	Mode         string           `json:"mode,omitempty"`
	Source       string           `json:"source,omitempty"`
	LastActivity string           `json:"last_activity,omitempty"`
	Message      string           `json:"message,omitempty"`
	Metrics      map[string]int64 `json:"metrics,omitempty"`
}

type workflowControlPlaneDiagnostics struct {
	SubjectType             string `json:"subject_type"`
	BlockedStepCount        int    `json:"blocked_step_count"`
	DispatchBlocked         int    `json:"dispatch_blocked"`
	MonitorBlocked          int    `json:"monitor_blocked"`
	FinalizeBlocked         int    `json:"finalize_blocked"`
	OldestBlockedAt         string `json:"oldest_blocked_at,omitempty"`
	OldestBlockedAgeSeconds int64  `json:"oldest_blocked_age_seconds,omitempty"`
	RecoveryAction          string `json:"recovery_action,omitempty"`
	RecoveryHint            string `json:"recovery_hint,omitempty"`
}

func (h *ExecutionPipelineHealthHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	components := map[string]pipelineComponentHealth{
		"dispatcher":                            h.dispatcherHealth(r.Context()),
		"workflow_orchestrator":                 h.processHealth("workflow_orchestrator"),
		"stale_execution_watchdog":              h.processHealth("stale_execution_watchdog"),
		"build_monitor_event_subscriber":        h.processHealth("build_monitor_event_subscriber"),
		"build_notification_event_subscriber":   h.processHealth("build_notification_event_subscriber"),
		"image_catalog_event_subscriber":        h.processHealth("image_catalog_event_subscriber"),
		"build_monitor_sweeper":                 h.processHealth("build_monitor_sweeper"),
		"messaging_outbox_relay":                h.processHealth("messaging_outbox_relay"),
		"provider_readiness_watcher":            h.processHealth("provider_readiness_watcher"),
		"tenant_asset_drift_watcher":            h.processHealth("tenant_asset_drift_watcher"),
		"quarantine_release_compliance_watcher": h.processHealth("quarantine_release_compliance_watcher"),
		"runtime_dependency_watcher":            h.processHealth("runtime_dependency_watcher"),
		"internal_registry_gc_worker":           h.processHealth("internal_registry_gc_worker"),
	}
	workflowDiagnostics := h.workflowDiagnostics(r.Context(), components["dispatcher"], components["workflow_orchestrator"])

	resp := map[string]interface{}{
		"checked_at": time.Now().UTC().Format(time.RFC3339),
		"components": components,
	}
	if workflowDiagnostics != nil {
		resp["workflow_control_plane"] = workflowDiagnostics
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *ExecutionPipelineHealthHandler) processHealth(name string) pipelineComponentHealth {
	if h.processStatus == nil {
		return pipelineComponentHealth{
			Enabled:   false,
			Running:   false,
			Available: false,
			Message:   "runtime status unavailable",
		}
	}

	status, ok := h.processStatus.GetStatus(name)
	if !ok {
		return pipelineComponentHealth{
			Enabled:   false,
			Running:   false,
			Available: false,
			Message:   "status not reported",
		}
	}

	resp := pipelineComponentHealth{
		Enabled:   status.Enabled,
		Running:   status.Running,
		Available: true,
		Message:   status.Message,
		Metrics:   status.Metrics,
	}
	if !status.LastActivity.IsZero() {
		resp.LastActivity = status.LastActivity.UTC().Format(time.RFC3339)
	}
	if !status.Enabled {
		resp.Message = "disabled"
	}
	return resp
}

func (h *ExecutionPipelineHealthHandler) dispatcherHealth(ctx context.Context) pipelineComponentHealth {
	if !isNilDispatcherController(h.dispatcherController) {
		running := h.dispatcherController.Status()
		resp := pipelineComponentHealth{
			Enabled:   true,
			Running:   running,
			Available: true,
			Mode:      appdispatcher.DispatcherModeEmbedded,
			Source:    "in_process",
		}
		if status := h.lookupRuntimeStatus(ctx, appdispatcher.DispatcherModeEmbedded); status != nil {
			resp.LastActivity = status.LastHeartbeat.UTC().Format(time.RFC3339)
		}
		if !running {
			resp.Message = "embedded dispatcher is not running"
		}
		return resp
	}

	externalStatus := h.lookupRuntimeStatus(ctx, appdispatcher.DispatcherModeExternal)
	if externalStatus != nil {
		fresh := time.Since(externalStatus.LastHeartbeat) <= 90*time.Second
		resp := pipelineComponentHealth{
			Enabled:      true,
			Running:      externalStatus.Running && fresh,
			Available:    fresh,
			Mode:         appdispatcher.DispatcherModeExternal,
			Source:       "heartbeat",
			LastActivity: externalStatus.LastHeartbeat.UTC().Format(time.RFC3339),
		}
		if !fresh {
			resp.Message = "external dispatcher heartbeat is stale"
		}
		return resp
	}

	return pipelineComponentHealth{
		Enabled:   h.dispatcherEnabled,
		Running:   false,
		Available: false,
		Mode:      appdispatcher.DispatcherModeEmbedded,
		Message:   "dispatcher status unavailable",
	}
}

func isNilDispatcherController(controller DispatcherController) bool {
	if controller == nil {
		return true
	}

	value := reflect.ValueOf(controller)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func (h *ExecutionPipelineHealthHandler) lookupRuntimeStatus(ctx context.Context, mode string) *appdispatcher.RuntimeStatus {
	if h.runtimeReader == nil {
		return nil
	}
	status, err := h.runtimeReader.GetLatestRuntimeStatus(ctx, mode)
	if err != nil {
		return nil
	}
	return status
}

func (h *ExecutionPipelineHealthHandler) workflowDiagnostics(ctx context.Context, dispatcher pipelineComponentHealth, orchestrator pipelineComponentHealth) *workflowControlPlaneDiagnostics {
	if h.workflowRepo == nil {
		return nil
	}

	diag, err := h.workflowRepo.GetBlockedStepDiagnostics(ctx, "build")
	if err != nil || diag == nil {
		return nil
	}

	out := &workflowControlPlaneDiagnostics{
		SubjectType:      diag.SubjectType,
		BlockedStepCount: diag.BlockedStepCount,
		DispatchBlocked:  diag.DispatchBlocked,
		MonitorBlocked:   diag.MonitorBlocked,
		FinalizeBlocked:  diag.FinalizeBlocked,
	}
	if diag.OldestBlockedAt != nil {
		ts := diag.OldestBlockedAt.UTC()
		out.OldestBlockedAt = ts.Format(time.RFC3339)
		out.OldestBlockedAgeSeconds = int64(time.Since(ts).Seconds())
	}

	if out.BlockedStepCount > 0 {
		switch {
		case !orchestrator.Enabled || !orchestrator.Running:
			out.RecoveryAction = "start_orchestrator"
			out.RecoveryHint = "Workflow orchestrator is not running. Start it to resume blocked control-plane steps."
		case !dispatcher.Enabled || !dispatcher.Running:
			out.RecoveryAction = "start_dispatcher"
			out.RecoveryHint = "Dispatcher is not running. Start it so dispatch steps can hand off build execution."
		default:
			out.RecoveryAction = "retry_step"
			out.RecoveryHint = "Pipeline services are running; inspect step errors and retry stuck workflow steps."
		}
	}

	return out
}
