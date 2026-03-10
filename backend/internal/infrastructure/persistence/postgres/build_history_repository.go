package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/build"
)

// BuildHistoryRepository implements the build.Repository interface for PostgreSQL
type BuildHistoryRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewBuildHistoryRepository creates a new build history repository
func NewBuildHistoryRepository(db *sqlx.DB, logger *zap.Logger) *BuildHistoryRepository {
	return &BuildHistoryRepository{
		db:     db,
		logger: logger,
	}
}

// historyModel represents the database model for build history
type historyModel struct {
	ID             uuid.UUID  `db:"id"`
	BuildID        uuid.UUID  `db:"build_id"`
	TenantID       uuid.UUID  `db:"tenant_id"`
	ProjectID      uuid.UUID  `db:"project_id"`
	BuildMethod    string     `db:"build_method"`
	WorkerID       *uuid.UUID `db:"worker_id"`
	DurationSeconds int        `db:"duration_seconds"`
	Success        bool       `db:"success"`
	StartedAt      *time.Time `db:"started_at"`
	CompletedAt    time.Time  `db:"completed_at"`
	CreatedAt      time.Time  `db:"created_at"`
}

// Save persists a build history record to the database
func (r *BuildHistoryRepository) Save(ctx context.Context, history *build.BuildHistory) error {
	if history == nil {
		return build.ErrInvalidBuildID
	}

	if history.BuildID() == uuid.Nil {
		return build.ErrInvalidBuildID
	}

	if history.TenantID() == uuid.Nil {
		return build.ErrInvalidTenantID
	}

	if history.ProjectID() == uuid.Nil {
		return build.ErrInvalidBuildID
	}

	model := r.toModel(history)

	query := `
		INSERT INTO build_history (
			id, build_id, tenant_id, project_id, build_method,
			worker_id, duration_seconds, success, started_at, completed_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		ON CONFLICT (id) DO UPDATE SET
			duration_seconds = EXCLUDED.duration_seconds,
			success = EXCLUDED.success,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at
	`

	_, err := r.db.ExecContext(ctx, query,
		model.ID,
		model.BuildID,
		model.TenantID,
		model.ProjectID,
		model.BuildMethod,
		model.WorkerID,
		model.DurationSeconds,
		model.Success,
		model.StartedAt,
		model.CompletedAt,
		model.CreatedAt,
	)

	if err != nil {
		r.logger.Error("failed to save build history",
			zap.Error(err),
			zap.String("build_id", history.BuildID().String()),
			zap.String("tenant_id", history.TenantID().String()),
		)
		return build.ErrPersistenceFailed
	}

	return nil
}

// FindByID retrieves a build history record by its ID
func (r *BuildHistoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*build.BuildHistory, error) {
	if id == uuid.Nil {
		return nil, build.ErrInvalidBuildID
	}

	query := `
		SELECT id, build_id, tenant_id, project_id, build_method,
		       worker_id, duration_seconds, success, started_at, completed_at, created_at
		FROM build_history
		WHERE id = $1
	`

	var model historyModel
	err := r.db.GetContext(ctx, &model, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("build history not found", zap.String("history_id", id.String()))
			return nil, build.ErrHistoryNotFound
		}
		r.logger.Error("failed to find build history", zap.Error(err), zap.String("history_id", id.String()))
		return nil, build.ErrPersistenceFailed
	}

	return r.toDomain(&model), nil
}

// FindByBuild retrieves the history record for a specific build
func (r *BuildHistoryRepository) FindByBuild(ctx context.Context, buildID uuid.UUID) (*build.BuildHistory, error) {
	if buildID == uuid.Nil {
		return nil, build.ErrInvalidBuildID
	}

	query := `
		SELECT id, build_id, tenant_id, project_id, build_method,
		       worker_id, duration_seconds, success, started_at, completed_at, created_at
		FROM build_history
		WHERE build_id = $1
	`

	var model historyModel
	err := r.db.GetContext(ctx, &model, query, buildID)
	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("build history not found for build", zap.String("build_id", buildID.String()))
			return nil, build.ErrHistoryNotFound
		}
		r.logger.Error("failed to find build history by build id",
			zap.Error(err),
			zap.String("build_id", buildID.String()),
		)
		return nil, build.ErrPersistenceFailed
	}

	return r.toDomain(&model), nil
}

// FindByProject retrieves all history records for a project
func (r *BuildHistoryRepository) FindByProject(ctx context.Context, projectID uuid.UUID) ([]*build.BuildHistory, error) {
	if projectID == uuid.Nil {
		return nil, build.ErrInvalidBuildID
	}

	query := `
		SELECT id, build_id, tenant_id, project_id, build_method,
		       worker_id, duration_seconds, success, started_at, completed_at, created_at
		FROM build_history
		WHERE project_id = $1
		ORDER BY completed_at DESC
	`

	var models []historyModel
	err := r.db.SelectContext(ctx, &models, query, projectID)
	if err != nil {
		r.logger.Error("failed to find history by project",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
		)
		return nil, build.ErrPersistenceFailed
	}

	histories := make([]*build.BuildHistory, len(models))
	for i, model := range models {
		histories[i] = r.toDomain(&model)
	}

	return histories, nil
}

// FindByMethod retrieves all history records for a specific build method within a tenant
func (r *BuildHistoryRepository) FindByMethod(ctx context.Context, tenantID uuid.UUID, method string) ([]*build.BuildHistory, error) {
	if tenantID == uuid.Nil {
		return nil, build.ErrInvalidTenantID
	}

	if method == "" {
		return nil, build.ErrInvalidMethod
	}

	query := `
		SELECT id, build_id, tenant_id, project_id, build_method,
		       worker_id, duration_seconds, success, started_at, completed_at, created_at
		FROM build_history
		WHERE tenant_id = $1 AND build_method = $2
		ORDER BY completed_at DESC
	`

	var models []historyModel
	err := r.db.SelectContext(ctx, &models, query, tenantID, method)
	if err != nil {
		r.logger.Error("failed to find history by method",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
			zap.String("method", method),
		)
		return nil, build.ErrPersistenceFailed
	}

	histories := make([]*build.BuildHistory, len(models))
	for i, model := range models {
		histories[i] = r.toDomain(&model)
	}

	return histories, nil
}

// FindSuccessfulByMethod retrieves successful builds for a method up to limit
func (r *BuildHistoryRepository) FindSuccessfulByMethod(ctx context.Context, projectID uuid.UUID, method string, limit int) ([]*build.BuildHistory, error) {
	if projectID == uuid.Nil {
		return nil, build.ErrInvalidBuildID
	}

	if method == "" {
		return nil, build.ErrInvalidMethod
	}

	if limit <= 0 {
		limit = 100 // Default limit
	}

	query := `
		SELECT id, build_id, tenant_id, project_id, build_method,
		       worker_id, duration_seconds, success, started_at, completed_at, created_at
		FROM build_history
		WHERE project_id = $1 AND build_method = $2 AND success = true
		ORDER BY completed_at DESC
		LIMIT $3
	`

	var models []historyModel
	err := r.db.SelectContext(ctx, &models, query, projectID, method, limit)
	if err != nil {
		r.logger.Error("failed to find successful history by method",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("method", method),
		)
		return nil, build.ErrPersistenceFailed
	}

	histories := make([]*build.BuildHistory, len(models))
	for i, model := range models {
		histories[i] = r.toDomain(&model)
	}

	return histories, nil
}

// FindRecent retrieves recent build history records within the specified duration
func (r *BuildHistoryRepository) FindRecent(ctx context.Context, tenantID uuid.UUID, since time.Duration) ([]*build.BuildHistory, error) {
	if tenantID == uuid.Nil {
		return nil, build.ErrInvalidTenantID
	}

	if since <= 0 {
		since = 24 * time.Hour // Default to last 24 hours
	}

	query := `
		SELECT id, build_id, tenant_id, project_id, build_method,
		       worker_id, duration_seconds, success, started_at, completed_at, created_at
		FROM build_history
		WHERE tenant_id = $1 AND completed_at > NOW() - INTERVAL '1 microsecond' * $2
		ORDER BY completed_at DESC
	`

	var models []historyModel
	err := r.db.SelectContext(ctx, &models, query, tenantID, since.Microseconds())
	if err != nil {
		r.logger.Error("failed to find recent history",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
		)
		return nil, build.ErrPersistenceFailed
	}

	histories := make([]*build.BuildHistory, len(models))
	for i, model := range models {
		histories[i] = r.toDomain(&model)
	}

	return histories, nil
}

// AverageDurationByMethod calculates average duration for successful builds by method
func (r *BuildHistoryRepository) AverageDurationByMethod(ctx context.Context, projectID uuid.UUID, method string) (time.Duration, error) {
	if projectID == uuid.Nil {
		return 0, build.ErrInvalidBuildID
	}

	if method == "" {
		return 0, build.ErrInvalidMethod
	}

	query := `
		SELECT AVG(duration_seconds)
		FROM build_history
		WHERE project_id = $1 AND build_method = $2 AND success = true
	`

	var avgSeconds sql.NullFloat64
	err := r.db.GetContext(ctx, &avgSeconds, query, projectID, method)
	if err != nil {
		r.logger.Error("failed to calculate average duration",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("method", method),
		)
		return 0, build.ErrPersistenceFailed
	}

	if !avgSeconds.Valid || avgSeconds.Float64 == 0 {
		return 0, build.ErrNoHistoryData
	}

	return time.Duration(int64(avgSeconds.Float64)) * time.Second, nil
}

// SuccessRateByMethod calculates the success rate (0.0-1.0) for a build method
func (r *BuildHistoryRepository) SuccessRateByMethod(ctx context.Context, projectID uuid.UUID, method string) (float64, error) {
	if projectID == uuid.Nil {
		return 0, build.ErrInvalidBuildID
	}

	if method == "" {
		return 0, build.ErrInvalidMethod
	}

	query := `
		SELECT COALESCE(
			SUM(CASE WHEN success = true THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*)::float, 0),
			0
		)
		FROM build_history
		WHERE project_id = $1 AND build_method = $2
	`

	var successRate float64
	err := r.db.GetContext(ctx, &successRate, query, projectID, method)
	if err != nil {
		r.logger.Error("failed to calculate success rate",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("method", method),
		)
		return 0, build.ErrPersistenceFailed
	}

	return successRate, nil
}

// CountByProjectAndMethod returns the count of builds for a project and method
func (r *BuildHistoryRepository) CountByProjectAndMethod(ctx context.Context, projectID uuid.UUID, method string) (int, error) {
	if projectID == uuid.Nil {
		return 0, build.ErrInvalidBuildID
	}

	if method == "" {
		return 0, build.ErrInvalidMethod
	}

	query := `
		SELECT COUNT(*)
		FROM build_history
		WHERE project_id = $1 AND build_method = $2
	`

	var count int
	err := r.db.GetContext(ctx, &count, query, projectID, method)
	if err != nil {
		r.logger.Error("failed to count builds by project and method",
			zap.Error(err),
			zap.String("project_id", projectID.String()),
			zap.String("method", method),
		)
		return 0, build.ErrPersistenceFailed
	}

	return count, nil
}

// Helper methods

// toDomain converts a database model to a domain value object
func (r *BuildHistoryRepository) toDomain(model *historyModel) *build.BuildHistory {
	history, err := build.New(
		model.ID,
		model.BuildID,
		model.TenantID,
		model.ProjectID,
		model.BuildMethod,
		model.WorkerID,
		time.Duration(model.DurationSeconds)*time.Second,
		model.Success,
		model.StartedAt,
		model.CompletedAt,
	)
	if err != nil {
		r.logger.Error("failed to reconstruct build history from database model",
			zap.Error(err),
			zap.String("build_id", model.BuildID.String()),
		)
		return nil
	}
	return history
}

// toModel converts a domain value object to a database model
func (r *BuildHistoryRepository) toModel(history *build.BuildHistory) *historyModel {
	return &historyModel{
		ID:              history.ID(),
		BuildID:         history.BuildID(),
		TenantID:        history.TenantID(),
		ProjectID:       history.ProjectID(),
		BuildMethod:     history.BuildMethod(),
		WorkerID:        history.WorkerID(),
		DurationSeconds: int(history.Duration().Seconds()),
		Success:         history.Success(),
		StartedAt:       history.StartedAt(),
		CompletedAt:     history.CompletedAt(),
		CreatedAt:       history.CreatedAt(),
	}
}
