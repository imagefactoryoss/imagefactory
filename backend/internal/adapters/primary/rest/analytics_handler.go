package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/build"
)

// AnalyticsHandler handles analytics and insights HTTP requests
type AnalyticsHandler struct {
	etaService        *build.ETAService
	historyRepository build.HistoryRepository
	logger            *zap.Logger
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(
	etaService *build.ETAService,
	historyRepository build.HistoryRepository,
	logger *zap.Logger,
) *AnalyticsHandler {
	return &AnalyticsHandler{
		etaService:        etaService,
		historyRepository: historyRepository,
		logger:            logger,
	}
}

// ============================================================================
// Request/Response Types
// ============================================================================

// ETAPredictionRequest represents a request to predict build duration
type ETAPredictionRequest struct {
	ProjectID   string `json:"project_id"`
	BuildMethod string `json:"build_method"`
}

// ETAPredictionResponse represents the ETA prediction result
type ETAPredictionResponse struct {
	EstimatedDuration string  `json:"estimated_duration"`
	DurationSeconds   int64   `json:"duration_seconds"`
	Confidence        float64 `json:"confidence"`
	SampleSize        int     `json:"sample_size"`
	AverageDuration   string  `json:"average_duration"`
	MedianDuration    string  `json:"median_duration"`
	Success           bool    `json:"success"`
	Message           string  `json:"message,omitempty"`
}

// PerformanceMetricsResponse represents build performance metrics
type PerformanceMetricsResponse struct {
	ProjectID        string  `json:"project_id"`
	BuildMethod      string  `json:"build_method"`
	TotalBuilds      int     `json:"total_builds"`
	SuccessfulBuilds int     `json:"successful_builds"`
	FailedBuilds     int     `json:"failed_builds"`
	SuccessRate      float64 `json:"success_rate"`
	AverageDuration  string  `json:"average_duration"`
	DurationSeconds  int64   `json:"duration_seconds"`
	Success          bool    `json:"success"`
	Message          string  `json:"message,omitempty"`
}

// ============================================================================
// Handler Methods
// ============================================================================

// PredictETA handles requests to predict build duration
// GET /api/v1/analytics/eta?project_id={id}&build_method={method}
func (h *AnalyticsHandler) PredictETA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	buildMethod := r.URL.Query().Get("build_method")

	if projectIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "Missing project_id parameter")
		return
	}

	if buildMethod == "" {
		h.respondError(w, http.StatusBadRequest, "Missing build_method parameter")
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project_id format")
		return
	}

	// Predict duration
	req := &build.PredictionRequest{
		ProjectID:   projectID,
		BuildMethod: buildMethod,
	}

	result, err := h.etaService.PredictDuration(r.Context(), req)
	if err != nil {
		h.logger.Error("failed to predict ETA",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("build_method", buildMethod),
		)
		h.respondError(w, http.StatusInternalServerError, "Failed to predict ETA")
		return
	}

	response := ETAPredictionResponse{
		EstimatedDuration: result.EstimatedDuration.String(),
		DurationSeconds:   int64(result.EstimatedDuration.Seconds()),
		Confidence:        result.Confidence,
		SampleSize:        result.SampleSize,
		AverageDuration:   result.AverageDuration.String(),
		MedianDuration:    result.MedianDuration.String(),
		Success:           true,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetPerformanceMetrics handles requests for build performance metrics
// GET /api/v1/analytics/performance?project_id={id}&build_method={method}
func (h *AnalyticsHandler) GetPerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	buildMethod := r.URL.Query().Get("build_method")

	if projectIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "Missing project_id parameter")
		return
	}

	if buildMethod == "" {
		h.respondError(w, http.StatusBadRequest, "Missing build_method parameter")
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project_id format")
		return
	}

	// Get average duration
	avgDuration, err := h.historyRepository.AverageDurationByMethod(r.Context(), projectID, buildMethod)
	if err != nil {
		h.logger.Error("failed to get average duration",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("build_method", buildMethod),
		)
		avgDuration = 0
	}

	// Get success rate
	successRate, err := h.historyRepository.SuccessRateByMethod(r.Context(), projectID, buildMethod)
	if err != nil {
		h.logger.Error("failed to get success rate",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("build_method", buildMethod),
		)
		successRate = 0
	}

	// Get total count
	totalCount, err := h.historyRepository.CountByProjectAndMethod(r.Context(), projectID, buildMethod)
	if err != nil {
		h.logger.Error("failed to get build count",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("build_method", buildMethod),
		)
		totalCount = 0
	}

	successCount := int(float64(totalCount) * successRate)
	failedCount := totalCount - successCount

	response := PerformanceMetricsResponse{
		ProjectID:        projectIDStr,
		BuildMethod:      buildMethod,
		TotalBuilds:      totalCount,
		SuccessfulBuilds: successCount,
		FailedBuilds:     failedCount,
		SuccessRate:      successRate * 100,
		AverageDuration:  time.Duration(avgDuration).String(),
		DurationSeconds:  int64(avgDuration.Seconds()),
		Success:          true,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetHealthScore handles requests for overall analytics health
// GET /api/v1/analytics/health?tenant_id={id}
func (h *AnalyticsHandler) GetHealthScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	tenantIDStr := r.URL.Query().Get("tenant_id")

	if tenantIDStr == "" {
		h.respondError(w, http.StatusBadRequest, "Missing tenant_id parameter")
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant_id format")
		return
	}

	// Get health statistics
	recentBuilds, err := h.historyRepository.FindRecent(r.Context(), tenantID, 24*time.Hour)
	if err != nil {
		h.logger.Error("failed to get recent builds",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
		)
		recentBuilds = []*build.BuildHistory{}
	}

	// Calculate metrics
	successful := 0
	for _, b := range recentBuilds {
		if b.Success() {
			successful++
		}
	}

	successRate := 0.0
	if len(recentBuilds) > 0 {
		successRate = float64(successful) / float64(len(recentBuilds)) * 100
	}

	response := map[string]interface{}{
		"tenant_id":         tenantIDStr,
		"recent_builds_24h": len(recentBuilds),
		"successful_builds": successful,
		"failed_builds":     len(recentBuilds) - successful,
		"success_rate":      successRate,
		"health_status":     "healthy",
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
		"success":           true,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// ============================================================================
// Helper Methods
// ============================================================================

func (h *AnalyticsHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
	}
}

func (h *AnalyticsHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]interface{}{
		"success": false,
		"message": message,
	})
}
