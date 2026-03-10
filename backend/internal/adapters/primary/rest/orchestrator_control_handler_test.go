package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/srikarm/image-factory/internal/application/runtimehealth"
)

type stubOrchestratorController struct {
	enabled bool
	running bool
	started int
	stopped int
}

func (s *stubOrchestratorController) Start(ctx context.Context) bool {
	if !s.enabled || s.running {
		return false
	}
	s.running = true
	s.started++
	return true
}

func (s *stubOrchestratorController) Status() bool {
	return s.running
}

func (s *stubOrchestratorController) Stop() bool {
	if !s.running {
		return false
	}
	s.running = false
	s.stopped++
	return true
}

func (s *stubOrchestratorController) Enabled() bool {
	return s.enabled
}

func TestOrchestratorControlHandler_StartOrchestrator(t *testing.T) {
	controller := &stubOrchestratorController{enabled: true, running: false}
	handler := NewOrchestratorControlHandler(controller, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/orchestrator/start", nil)
	rec := httptest.NewRecorder()

	handler.StartOrchestrator(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if running, ok := response["running"].(bool); !ok || !running {
		t.Fatalf("expected running=true in response, got %#v", response["running"])
	}
	if controller.started != 1 {
		t.Fatalf("expected controller to start once, got %d", controller.started)
	}
}

func TestOrchestratorControlHandler_StartOrchestrator_Disabled(t *testing.T) {
	store := runtimehealth.NewStore()
	store.Upsert("workflow_orchestrator", runtimehealth.ProcessStatus{
		Enabled:      false,
		Running:      false,
		LastActivity: time.Now().UTC(),
		Message:      "disabled",
	})
	handler := NewOrchestratorControlHandler(nil, store, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/orchestrator/start", nil)
	rec := httptest.NewRecorder()

	handler.StartOrchestrator(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if enabled, ok := response["enabled"].(bool); !ok || enabled {
		t.Fatalf("expected enabled=false in response, got %#v", response["enabled"])
	}
}

func TestOrchestratorControlHandler_StopOrchestrator(t *testing.T) {
	controller := &stubOrchestratorController{enabled: true, running: true}
	handler := NewOrchestratorControlHandler(controller, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/orchestrator/stop", nil)
	rec := httptest.NewRecorder()

	handler.StopOrchestrator(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if running, ok := response["running"].(bool); !ok || running {
		t.Fatalf("expected running=false in response, got %#v", response["running"])
	}
	if controller.stopped != 1 {
		t.Fatalf("expected controller to stop once, got %d", controller.stopped)
	}
}
