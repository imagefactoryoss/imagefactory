package rest

import (
	"context"
	"net/http"
	"time"

	appdispatcher "github.com/srikarm/image-factory/internal/application/dispatcher"
)

// DispatcherController manages dispatcher lifecycle for admin endpoints.
type DispatcherController interface {
	Start(ctx context.Context) bool
	Stop() bool
	Status() bool
}

// DispatcherRuntimeReader retrieves persisted dispatcher runtime snapshots.
type DispatcherRuntimeReader interface {
	GetLatestRuntimeStatus(ctx context.Context, mode string) (*appdispatcher.RuntimeStatus, error)
}

// DispatcherControlHandler handles dispatcher lifecycle endpoints.
type DispatcherControlHandler struct {
	controller    DispatcherController
	runtimeReader DispatcherRuntimeReader
	wsHub         *WebSocketHub
}

// NewDispatcherControlHandler creates a new dispatcher control handler.
func NewDispatcherControlHandler(controller DispatcherController, runtimeReader DispatcherRuntimeReader, wsHub *WebSocketHub) *DispatcherControlHandler {
	return &DispatcherControlHandler{controller: controller, runtimeReader: runtimeReader, wsHub: wsHub}
}

func (h *DispatcherControlHandler) getExternalRuntimeStatus(ctx context.Context) (*appdispatcher.RuntimeStatus, bool) {
	if h.runtimeReader == nil {
		return nil, false
	}

	status, err := h.runtimeReader.GetLatestRuntimeStatus(ctx, appdispatcher.DispatcherModeExternal)
	if err != nil || status == nil {
		return nil, false
	}

	if time.Since(status.LastHeartbeat) > 90*time.Second {
		return status, false
	}

	return status, true
}

// StartDispatcher handles POST /api/v1/admin/dispatcher/start
func (h *DispatcherControlHandler) StartDispatcher(w http.ResponseWriter, r *http.Request) {
	if h.controller == nil {
		if externalStatus, fresh := h.getExternalRuntimeStatus(r.Context()); externalStatus != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"running":            externalStatus.Running && fresh,
				"available":          fresh,
				"mode":               appdispatcher.DispatcherModeExternal,
				"managed_externally": true,
				"message":            "external dispatcher is managed out-of-process",
				"last_heartbeat":     externalStatus.LastHeartbeat.UTC().Format(time.RFC3339),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"running":   false,
			"available": false,
			"mode":      appdispatcher.DispatcherModeEmbedded,
			"message":   "dispatcher unavailable",
		})
		return
	}
	h.controller.Start(r.Context())
	if h.wsHub != nil {
		h.wsHub.BroadcastSystemEvent(
			"pipeline.health.changed",
			"info",
			"dispatcher start requested",
			map[string]interface{}{"component": "dispatcher", "action": "start"},
		)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"running":   h.controller.Status(),
		"available": true,
		"mode":      appdispatcher.DispatcherModeEmbedded,
	})
}

// StopDispatcher handles POST /api/v1/admin/dispatcher/stop
func (h *DispatcherControlHandler) StopDispatcher(w http.ResponseWriter, r *http.Request) {
	if h.controller == nil {
		if externalStatus, fresh := h.getExternalRuntimeStatus(r.Context()); externalStatus != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"running":            externalStatus.Running && fresh,
				"available":          fresh,
				"mode":               appdispatcher.DispatcherModeExternal,
				"managed_externally": true,
				"message":            "external dispatcher is managed out-of-process",
				"last_heartbeat":     externalStatus.LastHeartbeat.UTC().Format(time.RFC3339),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"running":   false,
			"available": false,
			"mode":      appdispatcher.DispatcherModeEmbedded,
			"message":   "dispatcher unavailable",
		})
		return
	}
	h.controller.Stop()
	if h.wsHub != nil {
		h.wsHub.BroadcastSystemEvent(
			"pipeline.health.changed",
			"info",
			"dispatcher stop requested",
			map[string]interface{}{"component": "dispatcher", "action": "stop"},
		)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"running":   h.controller.Status(),
		"available": true,
		"mode":      appdispatcher.DispatcherModeEmbedded,
	})
}

// GetDispatcherStatus handles GET /api/v1/admin/dispatcher/status
func (h *DispatcherControlHandler) GetDispatcherStatus(w http.ResponseWriter, r *http.Request) {
	if h.controller == nil {
		if externalStatus, fresh := h.getExternalRuntimeStatus(r.Context()); externalStatus != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"running":        externalStatus.Running && fresh,
				"available":      fresh,
				"mode":           appdispatcher.DispatcherModeExternal,
				"source":         "heartbeat",
				"last_heartbeat": externalStatus.LastHeartbeat.UTC().Format(time.RFC3339),
				"stale":          !fresh,
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"running":   false,
			"available": false,
			"mode":      appdispatcher.DispatcherModeEmbedded,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"running":   h.controller.Status(),
		"available": true,
		"mode":      appdispatcher.DispatcherModeEmbedded,
		"source":    "in_process",
	})
}
