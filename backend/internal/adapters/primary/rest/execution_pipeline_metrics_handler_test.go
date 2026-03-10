package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/srikarm/image-factory/internal/application/runtimehealth"
)

type executionPipelineMetricsResponse struct {
	CheckedAt string           `json:"checked_at"`
	Metrics   map[string]int64 `json:"metrics"`
}

func TestExecutionPipelineMetricsHandler_UsesRuntimeProcessMetrics(t *testing.T) {
	store := runtimehealth.NewStore()
	store.Upsert("build_monitor_event_subscriber", runtimehealth.ProcessStatus{
		Enabled: true,
		Running: true,
		Metrics: map[string]int64{
			"monitor_event_driven_transitions_total": 12,
			"monitor_event_driven_noop_total":        4,
			"monitor_event_driven_transition_errors": 1,
			"monitor_event_driven_parse_failures":    2,
		},
	})
	store.Upsert("build_monitor_sweeper", runtimehealth.ProcessStatus{
		Enabled: true,
		Running: true,
		Metrics: map[string]int64{
			"monitor_sweeper_reconciled_total": 7,
			"monitor_sweeper_failures_total":   3,
		},
	})
	store.Upsert("build_notification_event_subscriber", runtimehealth.ProcessStatus{
		Enabled: true,
		Running: true,
		Metrics: map[string]int64{
			"build_notification_events_received_total":  20,
			"build_notification_mapped_events_total":    17,
			"build_notification_in_app_delivered_total": 33,
			"build_notification_email_queued_total":     12,
			"build_notification_failures_total":         2,
		},
	})
	store.Upsert("image_catalog_event_subscriber", runtimehealth.ProcessStatus{
		Enabled: true,
		Running: true,
		Metrics: map[string]int64{
			"image_catalog_events_received_total":          41,
			"image_catalog_missing_execution_id_total":     3,
			"image_catalog_explicit_lookup_failures_total": 2,
			"image_catalog_fallback_attempts_total":        5,
			"image_catalog_fallback_success_total":         4,
			"image_catalog_fallback_skipped_total":         1,
			"image_catalog_missing_evidence_total":         6,
			"image_catalog_alerts_emitted_total":           7,
			"image_catalog_alert_delivery_failures_total":  1,
		},
	})
	store.Upsert("messaging_outbox_relay", runtimehealth.ProcessStatus{
		Enabled: true,
		Running: true,
		Metrics: map[string]int64{
			"messaging_outbox_pending_count":         11,
			"messaging_outbox_replay_success_total":  30,
			"messaging_outbox_replay_failures_total": 5,
		},
	})
	store.Upsert("tenant_asset_drift_watcher", runtimehealth.ProcessStatus{
		Enabled: true,
		Running: true,
		Metrics: map[string]int64{
			"tenant_asset_drift_watch_ticks_total":           8,
			"tenant_asset_drift_watch_failures_total":        2,
			"tenant_asset_drift_namespaces_current":          12,
			"tenant_asset_drift_namespaces_stale":            3,
			"tenant_asset_drift_namespaces_unknown":          1,
			"tenant_asset_reconcile_requests_total":          6,
			"tenant_asset_reconcile_requests_success_total":  5,
			"tenant_asset_reconcile_requests_failures_total": 1,
			"tenant_asset_drift_watch_duration_count":        8,
			"tenant_asset_drift_watch_duration_total_ms":     1600,
			"tenant_asset_drift_watch_duration_max_ms":       400,
		},
	})
	store.Upsert("runtime_dependency_watcher", runtimehealth.ProcessStatus{
		Enabled: true,
		Running: true,
		Metrics: map[string]int64{
			"runtime_dependency_checks_total":       14,
			"runtime_dependency_check_failures":     2,
			"runtime_dependency_degraded_count":     1,
			"runtime_dependency_critical_count":     1,
			"runtime_dependency_alerts_emitted":     3,
			"runtime_dependency_recoveries_emitted": 1,
		},
	})
	store.Upsert("quarantine_release_compliance_watcher", runtimehealth.ProcessStatus{
		Enabled: true,
		Running: true,
		Metrics: map[string]int64{
			"quarantine_release_compliance_watch_ticks_total":     5,
			"quarantine_release_compliance_watch_failures_total":  1,
			"quarantine_release_compliance_drift_detected_total":  2,
			"quarantine_release_compliance_drift_recovered_total": 1,
			"quarantine_release_compliance_active_drift_count":    1,
			"quarantine_release_compliance_released_count":        9,
		},
	})

	handler := NewExecutionPipelineMetricsHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/execution-pipeline/metrics", nil)
	w := httptest.NewRecorder()
	handler.GetMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response executionPipelineMetricsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.CheckedAt == "" {
		t.Fatal("expected checked_at to be populated")
	}
	if _, err := time.Parse(time.RFC3339, response.CheckedAt); err != nil {
		t.Fatalf("expected checked_at RFC3339 timestamp, got %q", response.CheckedAt)
	}
	if response.Metrics["monitor_event_driven_transitions_total"] != 12 {
		t.Fatalf("expected transitions_total=12, got %d", response.Metrics["monitor_event_driven_transitions_total"])
	}
	if response.Metrics["monitor_sweeper_reconciled_total"] != 7 {
		t.Fatalf("expected sweeper_reconciled_total=7, got %d", response.Metrics["monitor_sweeper_reconciled_total"])
	}
	if response.Metrics["build_notification_email_queued_total"] != 12 {
		t.Fatalf("expected build_notification_email_queued_total=12, got %d", response.Metrics["build_notification_email_queued_total"])
	}
	if response.Metrics["image_catalog_missing_evidence_total"] != 6 {
		t.Fatalf("expected image_catalog_missing_evidence_total=6, got %d", response.Metrics["image_catalog_missing_evidence_total"])
	}
	if response.Metrics["image_catalog_alerts_emitted_total"] != 7 {
		t.Fatalf("expected image_catalog_alerts_emitted_total=7, got %d", response.Metrics["image_catalog_alerts_emitted_total"])
	}
	if response.Metrics["messaging_outbox_pending_count"] != 11 {
		t.Fatalf("expected outbox_pending_count=11, got %d", response.Metrics["messaging_outbox_pending_count"])
	}
	if response.Metrics["tenant_asset_drift_namespaces_stale"] != 3 {
		t.Fatalf("expected tenant_asset_drift_namespaces_stale=3, got %d", response.Metrics["tenant_asset_drift_namespaces_stale"])
	}
	if response.Metrics["runtime_dependency_critical_count"] != 1 {
		t.Fatalf("expected runtime_dependency_critical_count=1, got %d", response.Metrics["runtime_dependency_critical_count"])
	}
	if response.Metrics["runtime_dependency_alerts_emitted"] != 3 {
		t.Fatalf("expected runtime_dependency_alerts_emitted=3, got %d", response.Metrics["runtime_dependency_alerts_emitted"])
	}
	if response.Metrics["quarantine_release_compliance_drift_detected_total"] != 2 {
		t.Fatalf("expected quarantine_release_compliance_drift_detected_total=2, got %d", response.Metrics["quarantine_release_compliance_drift_detected_total"])
	}
}

func TestExecutionPipelineMetricsHandler_DefaultsToZeroWhenUnavailable(t *testing.T) {
	handler := NewExecutionPipelineMetricsHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/execution-pipeline/metrics", nil)
	w := httptest.NewRecorder()
	handler.GetMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response executionPipelineMetricsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	for key, value := range response.Metrics {
		if value != 0 {
			t.Fatalf("expected %s to default to 0, got %d", key, value)
		}
	}
}
