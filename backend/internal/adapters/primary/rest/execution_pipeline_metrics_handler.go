package rest

import (
	"net/http"
	"time"

	"github.com/srikarm/image-factory/internal/application/runtimehealth"
)

// ExecutionPipelineMetricsHandler exposes execution-pipeline counters in a stable shape
// suitable for dashboard/alert ingestion.
type ExecutionPipelineMetricsHandler struct {
	processStatus runtimehealth.Provider
}

func NewExecutionPipelineMetricsHandler(processStatus runtimehealth.Provider) *ExecutionPipelineMetricsHandler {
	return &ExecutionPipelineMetricsHandler{processStatus: processStatus}
}

func (h *ExecutionPipelineMetricsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	out := map[string]interface{}{
		"checked_at": time.Now().UTC().Format(time.RFC3339),
		"metrics": map[string]int64{
			"monitor_event_driven_transitions_total":              h.metric("build_monitor_event_subscriber", "monitor_event_driven_transitions_total"),
			"monitor_event_driven_noop_total":                     h.metric("build_monitor_event_subscriber", "monitor_event_driven_noop_total"),
			"monitor_event_driven_transition_errors":              h.metric("build_monitor_event_subscriber", "monitor_event_driven_transition_errors"),
			"monitor_event_driven_parse_failures":                 h.metric("build_monitor_event_subscriber", "monitor_event_driven_parse_failures"),
			"monitor_sweeper_reconciled_total":                    h.metric("build_monitor_sweeper", "monitor_sweeper_reconciled_total"),
			"monitor_sweeper_failures_total":                      h.metric("build_monitor_sweeper", "monitor_sweeper_failures_total"),
			"build_notification_events_received_total":            h.metric("build_notification_event_subscriber", "build_notification_events_received_total"),
			"build_notification_mapped_events_total":              h.metric("build_notification_event_subscriber", "build_notification_mapped_events_total"),
			"build_notification_in_app_delivered_total":           h.metric("build_notification_event_subscriber", "build_notification_in_app_delivered_total"),
			"build_notification_email_queued_total":               h.metric("build_notification_event_subscriber", "build_notification_email_queued_total"),
			"build_notification_failures_total":                   h.metric("build_notification_event_subscriber", "build_notification_failures_total"),
			"image_catalog_events_received_total":                 h.metric("image_catalog_event_subscriber", "image_catalog_events_received_total"),
			"image_catalog_missing_execution_id_total":            h.metric("image_catalog_event_subscriber", "image_catalog_missing_execution_id_total"),
			"image_catalog_explicit_lookup_failures_total":        h.metric("image_catalog_event_subscriber", "image_catalog_explicit_lookup_failures_total"),
			"image_catalog_fallback_attempts_total":               h.metric("image_catalog_event_subscriber", "image_catalog_fallback_attempts_total"),
			"image_catalog_fallback_success_total":                h.metric("image_catalog_event_subscriber", "image_catalog_fallback_success_total"),
			"image_catalog_fallback_skipped_total":                h.metric("image_catalog_event_subscriber", "image_catalog_fallback_skipped_total"),
			"image_catalog_missing_evidence_total":                h.metric("image_catalog_event_subscriber", "image_catalog_missing_evidence_total"),
			"image_catalog_alerts_emitted_total":                  h.metric("image_catalog_event_subscriber", "image_catalog_alerts_emitted_total"),
			"image_catalog_alert_delivery_failures_total":         h.metric("image_catalog_event_subscriber", "image_catalog_alert_delivery_failures_total"),
			"messaging_outbox_pending_count":                      h.metric("messaging_outbox_relay", "messaging_outbox_pending_count"),
			"messaging_outbox_replay_success_total":               h.metric("messaging_outbox_relay", "messaging_outbox_replay_success_total"),
			"messaging_outbox_replay_failures_total":              h.metric("messaging_outbox_relay", "messaging_outbox_replay_failures_total"),
			"tenant_asset_drift_watch_ticks_total":                h.metric("tenant_asset_drift_watcher", "tenant_asset_drift_watch_ticks_total"),
			"tenant_asset_drift_watch_failures_total":             h.metric("tenant_asset_drift_watcher", "tenant_asset_drift_watch_failures_total"),
			"tenant_asset_drift_namespaces_current":               h.metric("tenant_asset_drift_watcher", "tenant_asset_drift_namespaces_current"),
			"tenant_asset_drift_namespaces_stale":                 h.metric("tenant_asset_drift_watcher", "tenant_asset_drift_namespaces_stale"),
			"tenant_asset_drift_namespaces_unknown":               h.metric("tenant_asset_drift_watcher", "tenant_asset_drift_namespaces_unknown"),
			"tenant_asset_reconcile_requests_total":               h.metric("tenant_asset_drift_watcher", "tenant_asset_reconcile_requests_total"),
			"tenant_asset_reconcile_requests_success_total":       h.metric("tenant_asset_drift_watcher", "tenant_asset_reconcile_requests_success_total"),
			"tenant_asset_reconcile_requests_failures_total":      h.metric("tenant_asset_drift_watcher", "tenant_asset_reconcile_requests_failures_total"),
			"tenant_asset_drift_watch_duration_count":             h.metric("tenant_asset_drift_watcher", "tenant_asset_drift_watch_duration_count"),
			"tenant_asset_drift_watch_duration_total_ms":          h.metric("tenant_asset_drift_watcher", "tenant_asset_drift_watch_duration_total_ms"),
			"tenant_asset_drift_watch_duration_max_ms":            h.metric("tenant_asset_drift_watcher", "tenant_asset_drift_watch_duration_max_ms"),
			"quarantine_release_compliance_watch_ticks_total":     h.metric("quarantine_release_compliance_watcher", "quarantine_release_compliance_watch_ticks_total"),
			"quarantine_release_compliance_watch_failures_total":  h.metric("quarantine_release_compliance_watcher", "quarantine_release_compliance_watch_failures_total"),
			"quarantine_release_compliance_drift_detected_total":  h.metric("quarantine_release_compliance_watcher", "quarantine_release_compliance_drift_detected_total"),
			"quarantine_release_compliance_drift_recovered_total": h.metric("quarantine_release_compliance_watcher", "quarantine_release_compliance_drift_recovered_total"),
			"quarantine_release_compliance_active_drift_count":    h.metric("quarantine_release_compliance_watcher", "quarantine_release_compliance_active_drift_count"),
			"quarantine_release_compliance_released_count":        h.metric("quarantine_release_compliance_watcher", "quarantine_release_compliance_released_count"),
			"runtime_dependency_checks_total":                     h.metric("runtime_dependency_watcher", "runtime_dependency_checks_total"),
			"runtime_dependency_check_failures":                   h.metric("runtime_dependency_watcher", "runtime_dependency_check_failures"),
			"runtime_dependency_degraded_count":                   h.metric("runtime_dependency_watcher", "runtime_dependency_degraded_count"),
			"runtime_dependency_critical_count":                   h.metric("runtime_dependency_watcher", "runtime_dependency_critical_count"),
			"runtime_dependency_alerts_emitted":                   h.metric("runtime_dependency_watcher", "runtime_dependency_alerts_emitted"),
			"runtime_dependency_recoveries_emitted":               h.metric("runtime_dependency_watcher", "runtime_dependency_recoveries_emitted"),
		},
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *ExecutionPipelineMetricsHandler) metric(processName, metricKey string) int64 {
	if h == nil || h.processStatus == nil {
		return 0
	}
	status, ok := h.processStatus.GetStatus(processName)
	if !ok || status.Metrics == nil {
		return 0
	}
	return status.Metrics[metricKey]
}
