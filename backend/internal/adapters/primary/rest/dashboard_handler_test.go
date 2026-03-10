package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestTenantDashboardHandler_GetTenantSummary_RequiresAuthContext(t *testing.T) {
	handler := NewTenantDashboardHandler(nil, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/tenant/summary", nil)
	w := httptest.NewRecorder()

	handler.GetTenantSummary(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestTenantDashboardHandler_GetTenantActivity_RequiresAuthContext(t *testing.T) {
	handler := NewTenantDashboardHandler(nil, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/tenant/activity", nil)
	w := httptest.NewRecorder()

	handler.GetTenantActivity(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestFormatDashboardDuration(t *testing.T) {
	if got := formatDashboardDuration(nil, nil); got != "n/a" {
		t.Fatalf("expected n/a for nil start time, got %q", got)
	}

	start := time.Now().UTC().Add(-95 * time.Second)
	if got := formatDashboardDuration(&start, nil); got == "n/a" {
		t.Fatalf("expected duration label, got %q", got)
	}

	end := start.Add(47 * time.Second)
	if got := formatDashboardDuration(&start, &end); got != "47s" {
		t.Fatalf("expected 47s, got %q", got)
	}
}
