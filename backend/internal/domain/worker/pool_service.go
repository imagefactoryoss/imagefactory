package worker

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PoolService defines business logic for worker pool management
type PoolService struct {
	repository Repository
	logger     *zap.Logger
}

// NewPoolService creates a new worker pool service
func NewPoolService(
	repository Repository,
	logger *zap.Logger,
) *PoolService {
	return &PoolService{
		repository: repository,
		logger:     logger,
	}
}

// RegisterWorkerRequest represents the request to register a worker
type RegisterWorkerRequest struct {
	TenantID   uuid.UUID
	Name       string
	WorkerType WorkerType
	Capacity   int
}

// RegisterWorker registers a new worker in the pool
func (s *PoolService) RegisterWorker(ctx context.Context, req *RegisterWorkerRequest) (*Worker, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}

	if req.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}

	if req.Name == "" {
		return nil, ErrWorkerNameRequired
	}

	if req.Capacity <= 0 {
		return nil, ErrInvalidCapacity
	}

	// Create new worker aggregate
	w, err := New(
		uuid.New(),
		req.TenantID,
		req.Name,
		req.WorkerType,
		Capacity(req.Capacity),
	)
	if err != nil {
		return nil, err
	}

	// Save to repository
	if err := s.repository.Save(ctx, w); err != nil {
		s.logger.Error("failed to register worker",
			zap.Error(err),
			zap.String("tenant_id", req.TenantID.String()),
			zap.String("worker_name", req.Name),
		)
		return nil, ErrPersistenceFailed
	}

	s.logger.Info("worker registered successfully",
		zap.String("worker_id", w.ID().String()),
		zap.String("tenant_id", req.TenantID.String()),
		zap.String("worker_name", req.Name),
		zap.Int("capacity", req.Capacity),
	)

	return w, nil
}

// UnregisterWorker removes a worker from the pool
func (s *PoolService) UnregisterWorker(ctx context.Context, workerID, tenantID uuid.UUID) error {
	if workerID == uuid.Nil {
		return ErrInvalidWorkerID
	}

	if tenantID == uuid.Nil {
		return ErrInvalidTenantID
	}

	// Verify worker exists and belongs to tenant
	worker, err := s.repository.FindByID(ctx, workerID)
	if err != nil {
		s.logger.Error("failed to fetch worker for unregistration",
			zap.Error(err),
			zap.String("worker_id", workerID.String()),
		)
		return ErrWorkerNotFound
	}

	if worker.TenantID() != tenantID {
		return ErrTenantMismatch
	}

	// Delete from repository
	if err := s.repository.Delete(ctx, workerID); err != nil {
		s.logger.Error("failed to delete worker",
			zap.Error(err),
			zap.String("worker_id", workerID.String()),
		)
		return ErrPersistenceFailed
	}

	s.logger.Info("worker unregistered successfully",
		zap.String("worker_id", workerID.String()),
		zap.String("tenant_id", tenantID.String()),
	)

	return nil
}

// UpdateLoadRequest represents the request to update worker load
type UpdateLoadRequest struct {
	WorkerID uuid.UUID
	TenantID uuid.UUID
	Delta    int // Positive to increase, negative to decrease
}

// UpdateLoad increments or decrements the worker's current load
func (s *PoolService) UpdateLoad(ctx context.Context, req *UpdateLoadRequest) (*Worker, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}

	if req.WorkerID == uuid.Nil {
		return nil, ErrInvalidWorkerID
	}

	if req.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}

	// Get worker
	worker, err := s.repository.FindByID(ctx, req.WorkerID)
	if err != nil {
		s.logger.Error("failed to fetch worker for load update",
			zap.Error(err),
			zap.String("worker_id", req.WorkerID.String()),
		)
		return nil, ErrWorkerNotFound
	}

	// Verify tenant match
	if worker.TenantID() != req.TenantID {
		return nil, ErrTenantMismatch
	}

	// Update load
	newLoad := worker.CurrentLoad() + req.Delta

	// Validate load within capacity
	if newLoad < 0 {
		return nil, ErrInvalidLoad
	}

	if newLoad > int(worker.Capacity()) {
		return nil, ErrLoadExceedsCapacity
	}

	// Apply change
	if req.Delta > 0 {
		if err := worker.IncrementLoad(req.Delta); err != nil {
			s.logger.Error("failed to increment load",
				zap.Error(err),
				zap.String("worker_id", req.WorkerID.String()),
			)
			return nil, ErrInvalidLoad
		}
	} else if req.Delta < 0 {
		if err := worker.DecrementLoad(-req.Delta); err != nil {
			s.logger.Error("failed to decrement load",
				zap.Error(err),
				zap.String("worker_id", req.WorkerID.String()),
			)
			return nil, ErrInvalidLoad
		}
	}

	// Save
	if err := s.repository.Save(ctx, worker); err != nil {
		s.logger.Error("failed to save worker load update",
			zap.Error(err),
			zap.String("worker_id", req.WorkerID.String()),
		)
		return nil, ErrPersistenceFailed
	}

	s.logger.Info("worker load updated",
		zap.String("worker_id", req.WorkerID.String()),
		zap.Int("old_load", worker.CurrentLoad()-req.Delta),
		zap.Int("new_load", newLoad),
		zap.Int("capacity", int(worker.Capacity())),
	)

	return worker, nil
}

// RecordHeartbeatRequest represents the request to record a worker heartbeat
type RecordHeartbeatRequest struct {
	WorkerID uuid.UUID
	TenantID uuid.UUID
	Healthy  bool
}

// RecordHeartbeat updates the worker's last heartbeat time and health status
func (s *PoolService) RecordHeartbeat(ctx context.Context, req *RecordHeartbeatRequest) (*Worker, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}

	if req.WorkerID == uuid.Nil {
		return nil, ErrInvalidWorkerID
	}

	if req.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}

	// Get worker
	worker, err := s.repository.FindByID(ctx, req.WorkerID)
	if err != nil {
		s.logger.Error("failed to fetch worker for heartbeat",
			zap.Error(err),
			zap.String("worker_id", req.WorkerID.String()),
		)
		return nil, ErrWorkerNotFound
	}

	// Verify tenant match
	if worker.TenantID() != req.TenantID {
		return nil, ErrTenantMismatch
	}

	// Update heartbeat
	if req.Healthy {
		if err := worker.RecordHeartbeat(); err != nil {
			s.logger.Error("failed to record heartbeat",
				zap.Error(err),
				zap.String("worker_id", req.WorkerID.String()),
			)
			return nil, errors.New("failed to record heartbeat")
		}
	} else {
		if err := worker.RecordFailure(); err != nil {
			s.logger.Error("failed to record failure",
				zap.Error(err),
				zap.String("worker_id", req.WorkerID.String()),
			)
			return nil, errors.New("failed to record failure")
		}
	}

	// Save
	if err := s.repository.Save(ctx, worker); err != nil {
		s.logger.Error("failed to save worker heartbeat",
			zap.Error(err),
			zap.String("worker_id", req.WorkerID.String()),
		)
		return nil, ErrPersistenceFailed
	}

	return worker, nil
}

// CheckHealthRequest represents the request to check worker health
type CheckHealthRequest struct {
	TenantID           uuid.UUID
	UnhealthyThreshold int // Consecutive failures before marking unhealthy
	StaleThreshold     time.Duration
}

// CheckHealth validates all workers in a tenant and marks unhealthy ones
func (s *PoolService) CheckHealth(ctx context.Context, req *CheckHealthRequest) (healthy int, unhealthy int, err error) {
	if req == nil {
		return 0, 0, ErrInvalidRequest
	}

	if req.TenantID == uuid.Nil {
		return 0, 0, ErrInvalidTenantID
	}

	// Find all workers for tenant
	allWorkers, err := s.repository.FindByTenant(ctx, req.TenantID)
	if err != nil {
		s.logger.Error("failed to fetch workers for health check",
			zap.Error(err),
			zap.String("tenant_id", req.TenantID.String()),
		)
		return 0, 0, ErrPersistenceFailed
	}

	// Count healthy and unhealthy
	for _, worker := range allWorkers {
		if worker.Status() == StatusOffline {
			unhealthy++
		} else {
			healthy++
		}
	}

	s.logger.Info("health check completed",
		zap.String("tenant_id", req.TenantID.String()),
		zap.Int("healthy", healthy),
		zap.Int("unhealthy", unhealthy),
	)

	return healthy, unhealthy, nil
}

// UnhealthyWorkers returns a list of all unhealthy workers
func (s *PoolService) UnhealthyWorkers(ctx context.Context, tenantID uuid.UUID, threshold int) ([]*Worker, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}

	// Get all workers for tenant and filter by status
	workers, err := s.repository.FindByTenant(ctx, tenantID)
	if err != nil {
		s.logger.Error("failed to fetch workers",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
		)
		return nil, ErrPersistenceFailed
	}

	unhealthy := []*Worker{}
	for _, w := range workers {
		if w.Status() == StatusOffline {
			unhealthy = append(unhealthy, w)
		}
	}

	return unhealthy, nil
}

// PoolStats represents statistics about the worker pool
type PoolStats struct {
	TotalWorkers    int
	HealthyWorkers  int
	UnhealthyCount  int
	TotalCapacity   int
	CurrentLoad     int
	AvailableCapacity int
	AverageLoad     float64
}

// GetPoolStats returns statistics about the worker pool
func (s *PoolService) GetPoolStats(ctx context.Context, tenantID uuid.UUID) (*PoolStats, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}

	// Get all workers for tenant
	workers, err := s.repository.FindByTenant(ctx, tenantID)
	if err != nil {
		s.logger.Error("failed to fetch workers for stats",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
		)
		return nil, ErrPersistenceFailed
	}

	stats := &PoolStats{
		TotalWorkers:  len(workers),
		TotalCapacity: 0,
		CurrentLoad:   0,
	}

	for _, worker := range workers {
		stats.TotalCapacity += int(worker.Capacity())
		stats.CurrentLoad += worker.CurrentLoad()

		if worker.Status() != StatusOffline {
			stats.HealthyWorkers++
		} else {
			stats.UnhealthyCount++
		}
	}

	stats.AvailableCapacity = stats.TotalCapacity - stats.CurrentLoad
	if stats.TotalCapacity > 0 {
		stats.AverageLoad = float64(stats.CurrentLoad) / float64(stats.TotalCapacity)
	}

	return stats, nil
}
