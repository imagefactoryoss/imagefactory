package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/worker"
)

// WorkerRepository implements the worker.Repository interface for PostgreSQL
type WorkerRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewWorkerRepository creates a new worker repository
func NewWorkerRepository(db *sqlx.DB, logger *zap.Logger) *WorkerRepository {
	return &WorkerRepository{
		db:     db,
		logger: logger,
	}
}

// workerModel represents the database model for worker
type workerModel struct {
	ID                   uuid.UUID  `db:"id"`
	TenantID             uuid.UUID  `db:"tenant_id"`
	WorkerName           string     `db:"worker_name"`
	WorkerType           string     `db:"worker_type"`
	Capacity             int        `db:"capacity"`
	CurrentLoad          int        `db:"current_load"`
	Status               string     `db:"status"`
	LastHeartbeat        *time.Time `db:"last_heartbeat"`
	ConsecutiveFailures  int        `db:"consecutive_failures"`
	CreatedAt            time.Time  `db:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at"`
}

// Save persists a worker to the database
func (r *WorkerRepository) Save(ctx context.Context, w *worker.Worker) error {
	if w == nil {
		return worker.ErrInvalidWorkerID
	}

	if w.TenantID() == uuid.Nil {
		return worker.ErrInvalidTenantID
	}

	model := r.toModel(w)

	query := `
		INSERT INTO workers (
			id, tenant_id, worker_name, worker_type, capacity, current_load,
			status, last_heartbeat, consecutive_failures, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		ON CONFLICT (id) DO UPDATE SET
			worker_name = EXCLUDED.worker_name,
			worker_type = EXCLUDED.worker_type,
			capacity = EXCLUDED.capacity,
			current_load = EXCLUDED.current_load,
			status = EXCLUDED.status,
			last_heartbeat = EXCLUDED.last_heartbeat,
			consecutive_failures = EXCLUDED.consecutive_failures,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		model.ID,
		model.TenantID,
		model.WorkerName,
		model.WorkerType,
		model.Capacity,
		model.CurrentLoad,
		model.Status,
		model.LastHeartbeat,
		model.ConsecutiveFailures,
		model.CreatedAt,
		model.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("failed to save worker", zap.Error(err), zap.String("worker_id", w.ID().String()))
		return worker.ErrPersistenceFailed
	}

	return nil
}

// Delete removes a worker from the database
func (r *WorkerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return worker.ErrInvalidWorkerID
	}

	query := `DELETE FROM workers WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("failed to delete worker", zap.Error(err), zap.String("worker_id", id.String()))
		return worker.ErrPersistenceFailed
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("failed to get rows affected", zap.Error(err), zap.String("worker_id", id.String()))
		return worker.ErrPersistenceFailed
	}

	if rowsAffected == 0 {
		return worker.ErrWorkerNotFound
	}

	return nil
}

// FindByID retrieves a worker by its ID
func (r *WorkerRepository) FindByID(ctx context.Context, id uuid.UUID) (*worker.Worker, error) {
	if id == uuid.Nil {
		return nil, worker.ErrInvalidWorkerID
	}

	query := `
		SELECT id, tenant_id, worker_name, worker_type, capacity, current_load,
		       status, last_heartbeat, consecutive_failures, created_at, updated_at
		FROM workers
		WHERE id = $1
	`

	var model workerModel
	err := r.db.GetContext(ctx, &model, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("worker not found", zap.String("worker_id", id.String()))
			return nil, worker.ErrWorkerNotFound
		}
		r.logger.Error("failed to find worker", zap.Error(err), zap.String("worker_id", id.String()))
		return nil, worker.ErrPersistenceFailed
	}

	return r.toDomain(&model), nil
}

// FindByTenant retrieves all workers for a tenant
func (r *WorkerRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*worker.Worker, error) {
	if tenantID == uuid.Nil {
		return nil, worker.ErrInvalidTenantID
	}

	query := `
		SELECT id, tenant_id, worker_name, worker_type, capacity, current_load,
		       status, last_heartbeat, consecutive_failures, created_at, updated_at
		FROM workers
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	var models []workerModel
	err := r.db.SelectContext(ctx, &models, query, tenantID)
	if err != nil {
		r.logger.Error("failed to find workers by tenant", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil, worker.ErrPersistenceFailed
	}

	workers := make([]*worker.Worker, len(models))
	for i, model := range models {
		workers[i] = r.toDomain(&model)
	}

	return workers, nil
}

// FindAvailable retrieves all available workers for a tenant (status='available' with available capacity)
func (r *WorkerRepository) FindAvailable(ctx context.Context, tenantID uuid.UUID) ([]*worker.Worker, error) {
	if tenantID == uuid.Nil {
		return nil, worker.ErrInvalidTenantID
	}

	query := `
		SELECT id, tenant_id, worker_name, worker_type, capacity, current_load,
		       status, last_heartbeat, consecutive_failures, created_at, updated_at
		FROM workers
		WHERE tenant_id = $1 AND status = 'available' AND current_load < capacity
		ORDER BY current_load ASC, created_at ASC
	`

	var models []workerModel
	err := r.db.SelectContext(ctx, &models, query, tenantID)
	if err != nil {
		r.logger.Error("failed to find available workers", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil, worker.ErrPersistenceFailed
	}

	workers := make([]*worker.Worker, len(models))
	for i, model := range models {
		workers[i] = r.toDomain(&model)
	}

	return workers, nil
}

// FindByType retrieves all workers of a specific type for a tenant
func (r *WorkerRepository) FindByType(ctx context.Context, tenantID uuid.UUID, workerType string) ([]*worker.Worker, error) {
	if tenantID == uuid.Nil {
		return nil, worker.ErrInvalidTenantID
	}

	if workerType == "" {
		return nil, fmt.Errorf("invalid worker type")
	}

	query := `
		SELECT id, tenant_id, worker_name, worker_type, capacity, current_load,
		       status, last_heartbeat, consecutive_failures, created_at, updated_at
		FROM workers
		WHERE tenant_id = $1 AND worker_type = $2
		ORDER BY created_at DESC
	`

	var models []workerModel
	err := r.db.SelectContext(ctx, &models, query, tenantID, workerType)
	if err != nil {
		r.logger.Error("failed to find workers by type",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
			zap.String("worker_type", workerType),
		)
		return nil, worker.ErrPersistenceFailed
	}

	workers := make([]*worker.Worker, len(models))
	for i, model := range models {
		workers[i] = r.toDomain(&model)
	}

	return workers, nil
}

// FindAvailableByType retrieves all available workers of a specific type for a tenant
func (r *WorkerRepository) FindAvailableByType(ctx context.Context, tenantID uuid.UUID, workerType worker.WorkerType) ([]*worker.Worker, error) {
	if tenantID == uuid.Nil {
		return nil, worker.ErrInvalidTenantID
	}

	query := `
		SELECT id, tenant_id, worker_name, worker_type, capacity, current_load,
		       status, last_heartbeat, consecutive_failures, created_at, updated_at
		FROM workers
		WHERE tenant_id = $1 AND worker_type = $2 AND status = 'available' AND current_load < capacity
		ORDER BY current_load ASC, created_at ASC
	`

	var models []workerModel
	err := r.db.SelectContext(ctx, &models, query, tenantID, string(workerType))
	if err != nil {
		r.logger.Error("failed to find available workers by type",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
			zap.String("worker_type", string(workerType)),
		)
		return nil, worker.ErrPersistenceFailed
	}

	workers := make([]*worker.Worker, len(models))
	for i, model := range models {
		workers[i] = r.toDomain(&model)
	}

	return workers, nil
}

// FindUnhealthy retrieves workers with status='offline' or consecutive_failures > threshold
func (r *WorkerRepository) FindUnhealthy(ctx context.Context, tenantID uuid.UUID) ([]*worker.Worker, error) {
	if tenantID == uuid.Nil {
		return nil, worker.ErrInvalidTenantID
	}

	query := `
		SELECT id, tenant_id, worker_name, worker_type, capacity, current_load,
		       status, last_heartbeat, consecutive_failures, created_at, updated_at
		FROM workers
		WHERE tenant_id = $1 AND (status = 'offline' OR consecutive_failures > 3)
		ORDER BY consecutive_failures DESC, last_heartbeat ASC
	`

	var models []workerModel
	err := r.db.SelectContext(ctx, &models, query, tenantID)
	if err != nil {
		r.logger.Error("failed to find unhealthy workers", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil, worker.ErrPersistenceFailed
	}

	workers := make([]*worker.Worker, len(models))
	for i, model := range models {
		workers[i] = r.toDomain(&model)
	}

	return workers, nil
}

// FindStale retrieves workers that haven't sent a heartbeat within the threshold duration
func (r *WorkerRepository) FindStale(ctx context.Context, threshold string) ([]*worker.Worker, error) {
	if threshold == "" {
		threshold = "5 minutes" // Default threshold
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, worker_name, worker_type, capacity, current_load,
		       status, last_heartbeat, consecutive_failures, created_at, updated_at
		FROM workers
		WHERE last_heartbeat < NOW() - INTERVAL '%s'
		ORDER BY last_heartbeat ASC
	`, threshold)

	var models []workerModel
	err := r.db.SelectContext(ctx, &models, query)
	if err != nil {
		r.logger.Error("failed to find stale workers", zap.Error(err), zap.String("threshold", threshold))
		return nil, worker.ErrPersistenceFailed
	}

	workers := make([]*worker.Worker, len(models))
	for i, model := range models {
		workers[i] = r.toDomain(&model)
	}

	return workers, nil
}

// Helper methods

// toDomain converts a database model to a domain aggregate
// Note: This reconstructs the aggregate but doesn't restore domain events
func (r *WorkerRepository) toDomain(model *workerModel) *worker.Worker {
	w, err := worker.New(
		model.ID,
		model.TenantID,
		model.WorkerName,
		worker.WorkerType(model.WorkerType),
		worker.Capacity(model.Capacity),
	)
	if err != nil {
		r.logger.Error("failed to reconstruct worker from database model",
			zap.Error(err),
			zap.String("worker_id", model.ID.String()),
		)
		return nil
	}
	// Note: Domain events are not restored during load - would require event sourcing
	return w
}

// toModel converts a domain model to a database model
func (r *WorkerRepository) toModel(w *worker.Worker) *workerModel {
	var status string
	switch s := w.Status(); s {
	case worker.StatusAvailable:
		status = "available"
	case worker.StatusBusy:
		status = "busy"
	case worker.StatusOffline:
		status = "offline"
	default:
		status = "available"
	}

	var workerType string
	switch wt := w.WorkerType(); wt {
	case worker.WorkerTypeDocker:
		workerType = "docker"
	case worker.WorkerTypeKubernetes:
		workerType = "kubernetes"
	case worker.WorkerTypeLambda:
		workerType = "lambda"
	default:
		workerType = "docker"
	}

	return &workerModel{
		ID:                  w.ID(),
		TenantID:            w.TenantID(),
		WorkerName:          w.Name(),
		WorkerType:          workerType,
		Capacity:            int(w.Capacity()),
		CurrentLoad:         w.CurrentLoad(),
		Status:              status,
		LastHeartbeat:       w.LastHeartbeat(),
		ConsecutiveFailures: w.ConsecutiveFailures(),
		CreatedAt:           w.CreatedAt(),
		UpdatedAt:           w.UpdatedAt(),
	}
}
