package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/srikarm/image-factory/internal/application/runtimehealth"
)

// WorkflowOrchestratorController manages workflow orchestrator lifecycle for admin endpoints.
type WorkflowOrchestratorController interface {
	Start(ctx context.Context) bool
	Stop() bool
	Status() bool
	Enabled() bool
}

// OrchestratorControlHandler handles workflow orchestrator lifecycle endpoints.
type OrchestratorControlHandler struct {
	controller         WorkflowOrchestratorController
	processStatusStore runtimehealth.Provider
	wsHub              *WebSocketHub
}

// NewOrchestratorControlHandler creates a new orchestrator control handler.
func NewOrchestratorControlHandler(controller WorkflowOrchestratorController, processStatusStore runtimehealth.Provider, wsHub *WebSocketHub) *OrchestratorControlHandler {
	return &OrchestratorControlHandler{
		controller:         controller,
		processStatusStore: processStatusStore,
		wsHub:              wsHub,
	}
}

// StartOrchestrator handles POST /api/v1/admin/orchestrator/start.
func (h *OrchestratorControlHandler) StartOrchestrator(w http.ResponseWriter, r *http.Request) {
	if h.controller == nil || !h.controller.Enabled() {
		status, ok := runtimehealth.ProcessStatus{}, false
		if h.processStatusStore != nil {
			status, ok = h.processStatusStore.GetStatus("workflow_orchestrator")
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"running":       status.Running,
			"enabled":       status.Enabled,
			"available":     ok,
			"last_activity": status.LastActivity.UTC().Format(time.RFC3339),
			"message":       "workflow orchestrator is disabled by runtime configuration",
		})
		return
	}

	started := h.controller.Start(r.Context())
	if h.wsHub != nil {
		h.wsHub.BroadcastSystemEvent(
			"pipeline.health.changed",
			"info",
			"workflow orchestrator start requested",
			map[string]interface{}{"component": "workflow_orchestrator", "action": "start"},
		)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"running":   h.controller.Status(),
		"enabled":   true,
		"available": true,
		"message": map[bool]string{
			true:  "workflow orchestrator started",
			false: "workflow orchestrator already running",
		}[started],
	})
}

// StopOrchestrator handles POST /api/v1/admin/orchestrator/stop.
func (h *OrchestratorControlHandler) StopOrchestrator(w http.ResponseWriter, r *http.Request) {
	if h.controller == nil || !h.controller.Enabled() {
		status, ok := runtimehealth.ProcessStatus{}, false
		if h.processStatusStore != nil {
			status, ok = h.processStatusStore.GetStatus("workflow_orchestrator")
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"running":       status.Running,
			"enabled":       status.Enabled,
			"available":     ok,
			"last_activity": status.LastActivity.UTC().Format(time.RFC3339),
			"message":       "workflow orchestrator is disabled by runtime configuration",
		})
		return
	}

	stopped := h.controller.Stop()
	if h.wsHub != nil {
		h.wsHub.BroadcastSystemEvent(
			"pipeline.health.changed",
			"info",
			"workflow orchestrator stop requested",
			map[string]interface{}{"component": "workflow_orchestrator", "action": "stop"},
		)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"running":   h.controller.Status(),
		"enabled":   true,
		"available": true,
		"message": map[bool]string{
			true:  "workflow orchestrator stopped",
			false: "workflow orchestrator already stopped",
		}[stopped],
	})
}
