package rest

import (
	"net/http"

	"github.com/srikarm/image-factory/internal/domain/health"
	"go.uber.org/zap"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	service *health.Service
	logger  *zap.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(service *health.Service, logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		service: service,
		logger:  logger,
	}
}

// HandleCheck handles GET /health requests
func (h *HealthHandler) HandleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"error":"method not allowed"}`))
		return
	}

	status, err := h.service.Check(r.Context())
	if err != nil {
		h.logger.Error("Health check failed", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status":"unhealthy","error":"health check failed"}`))
		return
	}

	// Return 200 if healthy, 503 if unhealthy
	statusCode := http.StatusOK
	if status.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encodeJSON(w, status)
}

// HandleReady handles GET /ready requests (Kubernetes readiness probe)
func (h *HealthHandler) HandleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"error":"method not allowed"}`))
		return
	}

	status, err := h.service.Check(r.Context())
	if err != nil {
		h.logger.Error("Readiness check failed", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		encodeJSON(w, map[string]interface{}{
			"ready":  false,
			"status": "unhealthy",
			"error":  "readiness check failed",
		})
		return
	}

	// For readiness, we check if the service is fully initialized
	// Return 200 only if all critical components are healthy
	if status.Status != "healthy" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		encodeJSON(w, map[string]interface{}{
			"ready":   false,
			"status":  status.Status,
			"message": "not ready",
			"build":   status.Build,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encodeJSON(w, map[string]interface{}{
		"ready":   true,
		"status":  status.Status,
		"build":   status.Build,
		"version": status.Version,
	})
}

// HandleAlive handles GET /alive requests (Kubernetes liveness probe)
func (h *HealthHandler) HandleAlive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"error":"method not allowed"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"alive":true}`))
}
