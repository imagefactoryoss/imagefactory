package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/srikarm/image-factory/internal/infrastructure/denialtelemetry"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"github.com/srikarm/image-factory/internal/infrastructure/releasecompliance"
	"github.com/srikarm/image-factory/internal/infrastructure/releasetelemetry"
	"go.uber.org/zap"
)

type eprStatsStub struct {
	metrics map[string]int64
}

func (s *eprStatsStub) GetLifecycleMetrics(ctx context.Context) (map[string]int64, error) {
	return s.metrics, nil
}

func TestSystemStatsHandlerGetSystemStats_IncludesReleaseMetrics(t *testing.T) {
	denials := denialtelemetry.NewMetrics()
	releases := releasetelemetry.NewMetrics()
	releases.Record(messaging.EventTypeQuarantineReleaseRequested)
	releases.Record(messaging.EventTypeQuarantineReleased)
	releases.Record(messaging.EventTypeQuarantineReleaseFailed)
	releases.Record(messaging.EventTypeQuarantineReleaseFailed)
	releases.Record(messaging.EventTypeQuarantineReleaseConsumed)

	handler := NewSystemStatsHandler(nil, nil, denials, releases, nil, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
	w := httptest.NewRecorder()
	handler.GetSystemStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var response SystemStatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.ReleaseMetrics == nil {
		t.Fatalf("expected release metrics in response")
	}
	if response.ReleaseMetrics.Requested != 1 || response.ReleaseMetrics.Released != 1 || response.ReleaseMetrics.Failed != 2 || response.ReleaseMetrics.Total != 4 || response.ReleaseMetrics.Consumed != 1 {
		t.Fatalf("unexpected release metrics: %+v", response.ReleaseMetrics)
	}
}

func TestSystemStatsHandlerGetSystemStats_IncludesEPRLifecycleMetrics(t *testing.T) {
	handler := NewSystemStatsHandler(nil, nil, nil, nil, nil, zap.NewNop())
	handler.SetEPRStatsReader(&eprStatsStub{
		metrics: map[string]int64{
			"total":     7,
			"active":    3,
			"expiring":  1,
			"expired":   2,
			"suspended": 1,
			"pending":   0,
			"approved":  6,
			"rejected":  1,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
	w := httptest.NewRecorder()
	handler.GetSystemStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var response SystemStatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.EPRLifecycleMetrics == nil {
		t.Fatal("expected epr_lifecycle_metrics in response")
	}
	if response.EPRLifecycleMetrics["active"] != 3 || response.EPRLifecycleMetrics["expired"] != 2 || response.EPRLifecycleMetrics["total"] != 7 {
		t.Fatalf("unexpected epr lifecycle metrics: %+v", response.EPRLifecycleMetrics)
	}
}

func TestSystemStatsHandlerGetSystemStats_RejectsNonGET(t *testing.T) {
	handler := NewSystemStatsHandler(nil, nil, nil, nil, nil, zap.NewNop())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/stats", nil)
	w := httptest.NewRecorder()

	handler.GetSystemStats(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestSystemStatsHandlerGetSystemStats_IncludesReleaseComplianceMetrics(t *testing.T) {
	compliance := releasecompliance.NewMetrics()
	compliance.RecordTick(2, 7)
	compliance.AddDetected(2)

	handler := NewSystemStatsHandler(nil, nil, nil, nil, compliance, zap.NewNop())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
	w := httptest.NewRecorder()
	handler.GetSystemStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var response SystemStatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.ReleaseCompliance == nil {
		t.Fatalf("expected release compliance metrics in response")
	}
	if response.ReleaseCompliance.ActiveDriftCount != 2 || response.ReleaseCompliance.ReleasedCount != 7 || response.ReleaseCompliance.DriftDetectedTotal != 2 {
		t.Fatalf("unexpected release compliance metrics: %+v", response.ReleaseCompliance)
	}
}
