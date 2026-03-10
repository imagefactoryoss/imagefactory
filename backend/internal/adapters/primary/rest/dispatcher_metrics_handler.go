package rest

import (
	"net/http"
	"time"

	appdispatcher "github.com/srikarm/image-factory/internal/application/dispatcher"
	"go.uber.org/zap"
)

// DispatcherMetricsProvider defines the interface for dispatcher metrics retrieval.
type DispatcherMetricsProvider interface {
	DispatcherMetrics() appdispatcher.DispatcherMetricsSnapshot
}

// DispatcherMetricsHandler handles dispatcher metrics requests.
type DispatcherMetricsHandler struct {
	provider      DispatcherMetricsProvider
	runtimeReader DispatcherRuntimeReader
	logger        *zap.Logger
}

// NewDispatcherMetricsHandler creates a new dispatcher metrics handler.
func NewDispatcherMetricsHandler(provider DispatcherMetricsProvider, runtimeReader DispatcherRuntimeReader, logger *zap.Logger) *DispatcherMetricsHandler {
	return &DispatcherMetricsHandler{
		provider:      provider,
		runtimeReader: runtimeReader,
		logger:        logger,
	}
}

// GetMetrics handles GET /api/v1/admin/dispatcher/metrics
func (h *DispatcherMetricsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.provider == nil {
		if h.runtimeReader != nil {
			status, err := h.runtimeReader.GetLatestRuntimeStatus(r.Context(), appdispatcher.DispatcherModeExternal)
			if err == nil && status != nil {
				fresh := time.Since(status.LastHeartbeat) <= 90*time.Second
				encodeJSON(w, map[string]interface{}{
					"claims":            status.Metrics.Claims,
					"dispatches":        status.Metrics.Dispatches,
					"claim_errors":      status.Metrics.ClaimErrors,
					"dispatch_errors":   status.Metrics.DispatchErrors,
					"requeues":          status.Metrics.Requeues,
					"skipped_for_limit": status.Metrics.SkippedForLimit,
					"claim_count":       status.Metrics.ClaimCount,
					"claim_min_ms":      status.Metrics.ClaimMinMs,
					"claim_max_ms":      status.Metrics.ClaimMaxMs,
					"claim_avg_ms":      status.Metrics.ClaimAvgMs,
					"dispatch_count":    status.Metrics.DispatchCount,
					"dispatch_min_ms":   status.Metrics.DispatchMinMs,
					"dispatch_max_ms":   status.Metrics.DispatchMaxMs,
					"dispatch_avg_ms":   status.Metrics.DispatchAvgMs,
					"available":         fresh,
					"mode":              appdispatcher.DispatcherModeExternal,
					"source":            "heartbeat",
					"last_heartbeat":    status.LastHeartbeat.UTC().Format(time.RFC3339),
					"stale":             !fresh,
				})
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		encodeJSON(w, map[string]interface{}{
			"claims":            0,
			"dispatches":        0,
			"claim_errors":      0,
			"dispatch_errors":   0,
			"requeues":          0,
			"skipped_for_limit": 0,
			"claim_count":       0,
			"claim_min_ms":      0,
			"claim_max_ms":      0,
			"claim_avg_ms":      0,
			"dispatch_count":    0,
			"dispatch_min_ms":   0,
			"dispatch_max_ms":   0,
			"dispatch_avg_ms":   0,
			"available":         false,
			"mode":              appdispatcher.DispatcherModeEmbedded,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	metrics := h.provider.DispatcherMetrics()
	encodeJSON(w, map[string]interface{}{
		"claims":            metrics.Claims,
		"dispatches":        metrics.Dispatches,
		"claim_errors":      metrics.ClaimErrors,
		"dispatch_errors":   metrics.DispatchErrors,
		"requeues":          metrics.Requeues,
		"skipped_for_limit": metrics.SkippedForLimit,
		"claim_count":       metrics.ClaimCount,
		"claim_min_ms":      metrics.ClaimMinMs,
		"claim_max_ms":      metrics.ClaimMaxMs,
		"claim_avg_ms":      metrics.ClaimAvgMs,
		"dispatch_count":    metrics.DispatchCount,
		"dispatch_min_ms":   metrics.DispatchMinMs,
		"dispatch_max_ms":   metrics.DispatchMaxMs,
		"dispatch_avg_ms":   metrics.DispatchAvgMs,
		"available":         true,
		"mode":              appdispatcher.DispatcherModeEmbedded,
		"source":            "in_process",
	})
}
