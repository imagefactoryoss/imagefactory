package rest

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/worker"
)

// WorkerHandler handles worker pool HTTP requests
type WorkerHandler struct {
	poolService *worker.PoolService
	logger      *zap.Logger
}

// NewWorkerHandler creates a new worker handler
func NewWorkerHandler(
	poolService *worker.PoolService,
	logger *zap.Logger,
) *WorkerHandler {
	return &WorkerHandler{
		poolService: poolService,
		logger:      logger,
	}
}

// ============================================================================
// Request Types
// ============================================================================

// RegisterWorkerRequest represents a request to register a worker
type RegisterWorkerRequest struct {
	TenantID   string `json:"tenant_id" validate:"required,uuid"`
	Name       string `json:"name" validate:"required,min=1"`
	WorkerType string `json:"worker_type" validate:"required"`
	Capacity   int    `json:"capacity" validate:"required,gt=0"`
}

// HeartbeatRequest represents a worker heartbeat request
type HeartbeatRequest struct {
	WorkerID string `json:"worker_id" validate:"required,uuid"`
	TenantID string `json:"tenant_id" validate:"required,uuid"`
	Healthy  bool   `json:"healthy"`
}

// ============================================================================
// Response Types
// ============================================================================

// WorkerResponse represents a worker in responses
type WorkerResponse struct {
	ID                    string `json:"id"`
	TenantID              string `json:"tenant_id"`
	Name                  string `json:"name"`
	WorkerType            string `json:"worker_type"`
	Capacity              int    `json:"capacity"`
	CurrentLoad           int    `json:"current_load"`
	Status                string `json:"status"`
	LastHeartbeat         string `json:"last_heartbeat,omitempty"`
	ConsecutiveFailures   int    `json:"consecutive_failures"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
}

// PoolStatsResponse represents worker pool statistics
type PoolStatsResponse struct {
	TotalWorkers          int     `json:"total_workers"`
	HealthyWorkers        int     `json:"healthy_workers"`
	UnhealthyWorkers      int     `json:"unhealthy_workers"`
	TotalCapacity         int     `json:"total_capacity"`
	CurrentLoad           int     `json:"current_load"`
	AvailableCapacity     int     `json:"available_capacity"`
	AverageLoad           float64 `json:"average_load"`
}

// ============================================================================
// Handler Methods
// ============================================================================

// Register handles POST /api/v1/workers/register
func (h *WorkerHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RegisterWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse UUIDs
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant_id")
		return
	}

	// Parse worker type
	workerType := worker.WorkerType(req.WorkerType)

	// Call service
	serviceReq := &worker.RegisterWorkerRequest{
		TenantID:   tenantID,
		Name:       req.Name,
		WorkerType: workerType,
		Capacity:   req.Capacity,
	}

	registeredWorker, err := h.poolService.RegisterWorker(ctx, serviceReq)
	if err != nil {
		h.handleServiceError(w, err, "Failed to register worker")
		return
	}

	// Return response
	h.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"worker": h.workerToResponse(registeredWorker),
	})

	h.logger.Info("Worker registered",
		zap.String("worker_id", registeredWorker.ID().String()),
		zap.String("tenant_id", tenantID.String()),
		zap.String("worker_name", req.Name),
	)
}

// Unregister handles DELETE /api/v1/workers/{id}
func (h *WorkerHandler) Unregister(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get worker_id from query params
	workerIDStr := r.URL.Query().Get("worker_id")
	if workerIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "worker_id query parameter required")
		return
	}

	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "tenant_id query parameter required")
		return
	}

	// Parse UUIDs
	workerID, err := uuid.Parse(workerIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid worker_id")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant_id")
		return
	}

	// Call service
	err = h.poolService.UnregisterWorker(ctx, workerID, tenantID)
	if err != nil {
		h.handleServiceError(w, err, "Failed to unregister worker")
		return
	}

	w.WriteHeader(http.StatusNoContent)

	h.logger.Info("Worker unregistered",
		zap.String("worker_id", workerID.String()),
		zap.String("tenant_id", tenantID.String()),
	)
}

// Heartbeat handles POST /api/v1/workers/heartbeat
func (h *WorkerHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse UUIDs
	workerID, err := uuid.Parse(req.WorkerID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid worker_id")
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant_id")
		return
	}

	// Call service
	serviceReq := &worker.RecordHeartbeatRequest{
		WorkerID: workerID,
		TenantID: tenantID,
		Healthy:  req.Healthy,
	}

	workerAggregate, err := h.poolService.RecordHeartbeat(ctx, serviceReq)
	if err != nil {
		h.handleServiceError(w, err, "Failed to record heartbeat")
		return
	}

	// Return response
	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"worker": h.workerToResponse(workerAggregate),
	})

	status := "healthy"
	if !req.Healthy {
		status = "unhealthy"
	}

	h.logger.Info("Heartbeat recorded",
		zap.String("worker_id", workerID.String()),
		zap.String("status", status),
	)
}

// GetStats handles GET /api/v1/workers/stats
func (h *WorkerHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tenant_id from query params
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "tenant_id query parameter required")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant_id")
		return
	}

	// Call service
	stats, err := h.poolService.GetPoolStats(ctx, tenantID)
	if err != nil {
		h.handleServiceError(w, err, "Failed to fetch pool stats")
		return
	}

	// Convert to response
	response := PoolStatsResponse{
		TotalWorkers:      stats.TotalWorkers,
		HealthyWorkers:    stats.HealthyWorkers,
		UnhealthyWorkers:  stats.UnhealthyCount,
		TotalCapacity:     stats.TotalCapacity,
		CurrentLoad:       stats.CurrentLoad,
		AvailableCapacity: stats.AvailableCapacity,
		AverageLoad:       stats.AverageLoad,
	}

	h.respondJSON(w, http.StatusOK, response)

	h.logger.Debug("Pool stats retrieved",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("total_workers", stats.TotalWorkers),
		zap.Int("healthy_workers", stats.HealthyWorkers),
	)
}

// ============================================================================
// Helper Methods
// ============================================================================

// workerToResponse converts a domain Worker to response DTO
func (h *WorkerHandler) workerToResponse(w *worker.Worker) WorkerResponse {
	resp := WorkerResponse{
		ID:                  w.ID().String(),
		TenantID:            w.TenantID().String(),
		Name:                w.Name(),
		WorkerType:          string(w.WorkerType()),
		Capacity:            int(w.Capacity()),
		CurrentLoad:         w.CurrentLoad(),
		Status:              string(w.Status()),
		ConsecutiveFailures: w.ConsecutiveFailures(),
		CreatedAt:           w.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:           w.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	if !w.LastHeartbeat().IsZero() {
		resp.LastHeartbeat = w.LastHeartbeat().Format("2006-01-02T15:04:05Z07:00")
	}

	return resp
}

// handleServiceError translates domain errors to HTTP responses
func (h *WorkerHandler) handleServiceError(w http.ResponseWriter, err error, defaultMsg string) {
	errStr := err.Error()

	switch {
	case errStr == "invalid request" || errStr == "invalid tenant id" || errStr == "invalid worker id" ||
		errStr == "worker name is required" || errStr == "capacity must be greater than 0":
		h.respondError(w, http.StatusBadRequest, errStr)

	case errStr == "worker not found" || errStr == "tenant not found":
		h.respondError(w, http.StatusNotFound, errStr)

	case errStr == "tenant mismatch":
		h.respondError(w, http.StatusForbidden, "Tenant mismatch")

	case errStr == "invalid load" || errStr == "load exceeds capacity":
		h.respondError(w, http.StatusConflict, errStr)

	default:
		h.respondError(w, http.StatusInternalServerError, defaultMsg)
		h.logger.Error("Service error", zap.Error(err))
	}
}

// respondJSON sends a JSON response
func (h *WorkerHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends a JSON error response
func (h *WorkerHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
