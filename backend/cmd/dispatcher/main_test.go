package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDispatcherHealthEndpointHealthy(t *testing.T) {
	state := &dispatcherHealthState{
		instanceID: "external-test",
		startedAt:  time.Now().Add(-5 * time.Second),
	}
	state.running.Store(true)

	rec := httptest.NewRecorder()
	respondDispatcherHealth(rec, state, false)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse health response: %v", err)
	}
	if payload["status"] != "healthy" {
		t.Fatalf("expected status=healthy, got %v", payload["status"])
	}
	if payload["service"] != "dispatcher" {
		t.Fatalf("expected service=dispatcher, got %v", payload["service"])
	}
	if payload["running"] != true {
		t.Fatalf("expected running=true, got %v", payload["running"])
	}
}

func TestDispatcherReadyEndpointNotReadyWhenStopped(t *testing.T) {
	state := &dispatcherHealthState{
		instanceID: "external-test",
		startedAt:  time.Now().Add(-5 * time.Second),
	}
	state.running.Store(false)

	rec := httptest.NewRecorder()
	respondDispatcherHealth(rec, state, true)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse readiness response: %v", err)
	}
	if payload["status"] != "not_ready" {
		t.Fatalf("expected status=not_ready, got %v", payload["status"])
	}
	if payload["running"] != false {
		t.Fatalf("expected running=false, got %v", payload["running"])
	}
}
