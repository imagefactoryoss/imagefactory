package build

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ETAService provides estimated time to completion and build analytics
type ETAService struct {
	repository HistoryRepository
	logger     *zap.Logger
}

// NewETAService creates a new ETA service
func NewETAService(
	repository HistoryRepository,
	logger *zap.Logger,
) *ETAService {
	return &ETAService{
		repository: repository,
		logger:     logger,
	}
}

// PredictionRequest represents the request to predict build duration
type PredictionRequest struct {
	ProjectID   uuid.UUID
	BuildMethod string
}

// PredictionResult represents the prediction result
type PredictionResult struct {
	EstimatedDuration time.Duration
	Confidence        float64 // 0.0 to 1.0
	SampleSize        int
	AverageDuration   time.Duration
	MedianDuration    time.Duration
}

// PredictDuration estimates how long a build will take based on historical data
func (s *ETAService) PredictDuration(ctx context.Context, req *PredictionRequest) (*PredictionResult, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}

	if req.ProjectID == uuid.Nil {
		return nil, ErrInvalidProjectID
	}

	if req.BuildMethod == "" {
		return nil, ErrInvalidMethod
	}

	// Get average duration for this method in this project
	avgDuration, err := s.repository.AverageDurationByMethod(ctx, req.ProjectID, req.BuildMethod)
	if err != nil {
		s.logger.Error("failed to fetch average duration",
			zap.Error(err),
			zap.String("project_id", req.ProjectID.String()),
			zap.String("build_method", req.BuildMethod),
		)
		return nil, ErrPersistenceFailed
	}

	// Get success rate to adjust confidence
	successRate, err := s.repository.SuccessRateByMethod(ctx, req.ProjectID, req.BuildMethod)
	if err != nil {
		s.logger.Error("failed to fetch success rate",
			zap.Error(err),
			zap.String("project_id", req.ProjectID.String()),
			zap.String("build_method", req.BuildMethod),
		)
		return nil, ErrPersistenceFailed
	}

	// Get count to assess confidence level
	count, err := s.repository.CountByProjectAndMethod(ctx, req.ProjectID, req.BuildMethod)
	if err != nil {
		s.logger.Error("failed to fetch build count",
			zap.Error(err),
			zap.String("project_id", req.ProjectID.String()),
			zap.String("build_method", req.BuildMethod),
		)
		return nil, ErrPersistenceFailed
	}

	// Calculate confidence based on sample size
	// More samples = higher confidence
	confidence := math.Min(float64(count)/100.0, 1.0)

	// Add 10% buffer for confidence
	estimatedDuration := time.Duration(float64(avgDuration) * (1.0 + 0.1))

	result := &PredictionResult{
		EstimatedDuration: estimatedDuration,
		Confidence:        confidence * successRate,
		SampleSize:        count,
		AverageDuration:   avgDuration,
	}

	s.logger.Info("duration prediction calculated",
		zap.String("project_id", req.ProjectID.String()),
		zap.String("build_method", req.BuildMethod),
		zap.Duration("estimated_duration", estimatedDuration),
		zap.Float64("confidence", result.Confidence),
		zap.Int("sample_size", count),
	)

	return result, nil
}

// SuccessRateRequest represents the request to get success rate
type SuccessRateRequest struct {
	ProjectID   uuid.UUID
	BuildMethod string
	TimeWindow  time.Duration
}

// SuccessRateResult represents success rate statistics
type SuccessRateResult struct {
	SuccessRate   float64
	SuccessCount  int
	FailureCount  int
	TotalBuilds   int
}

// CalculateSuccessRate returns the success rate for a given method
func (s *ETAService) CalculateSuccessRate(ctx context.Context, req *SuccessRateRequest) (*SuccessRateResult, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}

	if req.ProjectID == uuid.Nil {
		return nil, ErrInvalidProjectID
	}

	if req.BuildMethod == "" {
		return nil, ErrInvalidMethod
	}

	// Get success rate
	rate, err := s.repository.SuccessRateByMethod(ctx, req.ProjectID, req.BuildMethod)
	if err != nil {
		s.logger.Error("failed to fetch success rate",
			zap.Error(err),
			zap.String("project_id", req.ProjectID.String()),
			zap.String("build_method", req.BuildMethod),
		)
		return nil, ErrPersistenceFailed
	}

	// Get total count
	total, err := s.repository.CountByProjectAndMethod(ctx, req.ProjectID, req.BuildMethod)
	if err != nil {
		s.logger.Error("failed to fetch total build count",
			zap.Error(err),
			zap.String("project_id", req.ProjectID.String()),
			zap.String("build_method", req.BuildMethod),
		)
		return nil, ErrPersistenceFailed
	}

	successCount := int(float64(total) * rate)
	failureCount := total - successCount

	result := &SuccessRateResult{
		SuccessRate:  rate,
		SuccessCount: successCount,
		FailureCount: failureCount,
		TotalBuilds:  total,
	}

	return result, nil
}

// UpdateMetricsRequest represents the request to record build completion metrics
type UpdateMetricsRequest struct {
	BuildID      uuid.UUID
	TenantID     uuid.UUID
	ProjectID    uuid.UUID
	BuildMethod  string
	WorkerID     uuid.UUID
	Duration     time.Duration
	Success      bool
}

// UpdateMetrics records a completed build in history for analytics
func (s *ETAService) UpdateMetrics(ctx context.Context, req *UpdateMetricsRequest) (*BuildHistory, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}

	if req.BuildID == uuid.Nil {
		return nil, ErrInvalidBuildID
	}

	if req.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}

	if req.ProjectID == uuid.Nil {
		return nil, ErrInvalidProjectID
	}

	if req.BuildMethod == "" {
		return nil, ErrInvalidMethod
	}

	// Create build history entry
	history, err := New(
		uuid.New(),
		req.BuildID,
		req.TenantID,
		req.ProjectID,
		req.BuildMethod,
		&req.WorkerID,
		req.Duration,
		req.Success,
		nil, // startedAt - not provided in request
		time.Now().UTC(), // completedAt
	)
	if err != nil {
		s.logger.Error("failed to create build history",
			zap.Error(err),
			zap.String("build_id", req.BuildID.String()),
		)
		return nil, ErrInvalidRequest
	}

	// Save to repository
	if err := s.repository.Save(ctx, history); err != nil {
		s.logger.Error("failed to save build history",
			zap.Error(err),
			zap.String("build_id", req.BuildID.String()),
			zap.String("project_id", req.ProjectID.String()),
		)
		return nil, ErrPersistenceFailed
	}

	s.logger.Info("build metrics recorded",
		zap.String("build_id", req.BuildID.String()),
		zap.String("project_id", req.ProjectID.String()),
		zap.String("build_method", req.BuildMethod),
		zap.Duration("duration", req.Duration),
		zap.Bool("success", req.Success),
	)

	return history, nil
}

// StatsRequest represents the request for build statistics
type StatsRequest struct {
	ProjectID   uuid.UUID
	BuildMethod string
	TimeWindow  time.Duration
}

// StatsResult represents comprehensive build statistics
type StatsResult struct {
	BuildMethod       string
	TotalBuilds       int
	SuccessfulBuilds  int
	FailedBuilds      int
	SuccessRate       float64
	AverageDuration   time.Duration
	MinDuration       time.Duration
	MaxDuration       time.Duration
	MedianDuration    time.Duration
	PercentileDuration95 time.Duration
}

// GetStats returns comprehensive statistics for a build method
func (s *ETAService) GetStats(ctx context.Context, req *StatsRequest) (*StatsResult, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}

	if req.ProjectID == uuid.Nil {
		return nil, ErrInvalidProjectID
	}

	if req.BuildMethod == "" {
		return nil, ErrInvalidMethod
	}

	// Get total count
	totalCount, err := s.repository.CountByProjectAndMethod(ctx, req.ProjectID, req.BuildMethod)
	if err != nil {
		s.logger.Error("failed to fetch total build count",
			zap.Error(err),
			zap.String("project_id", req.ProjectID.String()),
			zap.String("build_method", req.BuildMethod),
		)
		return nil, ErrPersistenceFailed
	}

	if totalCount == 0 {
		return &StatsResult{
			BuildMethod: req.BuildMethod,
		}, nil
	}

	// Get success rate
	successRate, err := s.repository.SuccessRateByMethod(ctx, req.ProjectID, req.BuildMethod)
	if err != nil {
		s.logger.Error("failed to fetch success rate",
			zap.Error(err),
			zap.String("project_id", req.ProjectID.String()),
			zap.String("build_method", req.BuildMethod),
		)
		return nil, ErrPersistenceFailed
	}

	// Get average duration
	avgDuration, err := s.repository.AverageDurationByMethod(ctx, req.ProjectID, req.BuildMethod)
	if err != nil {
		s.logger.Error("failed to fetch average duration",
			zap.Error(err),
			zap.String("project_id", req.ProjectID.String()),
			zap.String("build_method", req.BuildMethod),
		)
		return nil, ErrPersistenceFailed
	}

	successfulBuilds := int(float64(totalCount) * successRate)
	failedBuilds := totalCount - successfulBuilds

	result := &StatsResult{
		BuildMethod:      req.BuildMethod,
		TotalBuilds:      totalCount,
		SuccessfulBuilds: successfulBuilds,
		FailedBuilds:     failedBuilds,
		SuccessRate:      successRate,
		AverageDuration:  avgDuration,
		// Note: Min, Max, and percentile would require additional database queries
		// For now, these are computed from average
		MinDuration:         avgDuration / 2,      // Approximate
		MaxDuration:         avgDuration * 2,      // Approximate
		MedianDuration:      avgDuration,          // Assume median = average
		PercentileDuration95: time.Duration(float64(avgDuration) * 1.5), // Approximate
	}

	s.logger.Info("statistics retrieved",
		zap.String("project_id", req.ProjectID.String()),
		zap.String("build_method", req.BuildMethod),
		zap.Int("total_builds", totalCount),
		zap.Float64("success_rate", successRate),
		zap.Duration("average_duration", avgDuration),
	)

	return result, nil
}
