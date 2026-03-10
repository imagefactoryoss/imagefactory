package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/srikarm/image-factory/internal/domain/build"
)

// PostgresBuildExecutionRepository implements BuildExecutionRepository for PostgreSQL
type PostgresBuildExecutionRepository struct {
	db *sqlx.DB
}

// NewPostgresBuildExecutionRepository creates a new PostgreSQL execution repository
func NewPostgresBuildExecutionRepository(db *sqlx.DB) build.BuildExecutionRepository {
	return &PostgresBuildExecutionRepository{db: db}
}

// SaveExecution creates a new execution record
func (r *PostgresBuildExecutionRepository) SaveExecution(ctx context.Context, execution *build.BuildExecution) error {
	query := `
		INSERT INTO build_executions 
		(id, build_id, config_id, status, created_by, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		execution.ID,
		execution.BuildID,
		execution.ConfigID,
		string(execution.Status),
		execution.CreatedBy,
		execution.Metadata,
		execution.CreatedAt,
		execution.UpdatedAt,
	)

	return err
}

// UpdateExecution updates an existing execution record
func (r *PostgresBuildExecutionRepository) UpdateExecution(ctx context.Context, execution *build.BuildExecution) error {
	query := `
		UPDATE build_executions
		SET status = $1, 
		    started_at = $2, 
		    completed_at = $3, 
		    duration_seconds = $4,
		    output = $5,
		    error_message = $6,
		    artifacts = $7,
		    metadata = $8,
		    updated_at = $9
		WHERE id = $10
	`

	_, err := r.db.ExecContext(ctx, query,
		string(execution.Status),
		execution.StartedAt,
		execution.CompletedAt,
		execution.DurationSeconds,
		execution.Output,
		execution.ErrorMessage,
		execution.Artifacts,
		execution.Metadata,
		execution.UpdatedAt,
		execution.ID,
	)

	return err
}

// GetExecution retrieves a single execution by ID
func (r *PostgresBuildExecutionRepository) GetExecution(ctx context.Context, id uuid.UUID) (*build.BuildExecution, error) {
	query := `
		SELECT id, build_id, config_id, status, started_at, completed_at, 
		       duration_seconds, output, error_message, artifacts, metadata, created_by, created_at, updated_at
		FROM build_executions
		WHERE id = $1
	`

	var execution build.BuildExecution
	var status string

	err := r.db.QueryRowxContext(ctx, query, id).Scan(
		&execution.ID,
		&execution.BuildID,
		&execution.ConfigID,
		&status,
		&execution.StartedAt,
		&execution.CompletedAt,
		&execution.DurationSeconds,
		&execution.Output,
		&execution.ErrorMessage,
		&execution.Artifacts,
		&execution.Metadata,
		&execution.CreatedBy,
		&execution.CreatedAt,
		&execution.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, build.ErrExecutionNotFound
	}
	if err != nil {
		return nil, err
	}

	execution.Status = build.ExecutionStatus(status)
	return &execution, nil
}

// GetBuildExecutions retrieves all executions for a build with pagination
func (r *PostgresBuildExecutionRepository) GetBuildExecutions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]build.BuildExecution, int64, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM build_executions WHERE build_id = $1`
	var total int64
	if err := r.db.QueryRowxContext(ctx, countQuery, buildID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query := `
		SELECT id, build_id, config_id, status, started_at, completed_at, 
		       duration_seconds, output, error_message, artifacts, metadata, created_by, created_at, updated_at
		FROM build_executions
		WHERE build_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryxContext(ctx, query, buildID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	executions := []build.BuildExecution{}
	for rows.Next() {
		var execution build.BuildExecution
		var status string

		err := rows.Scan(
			&execution.ID,
			&execution.BuildID,
			&execution.ConfigID,
			&status,
			&execution.StartedAt,
			&execution.CompletedAt,
			&execution.DurationSeconds,
			&execution.Output,
			&execution.ErrorMessage,
			&execution.Artifacts,
			&execution.Metadata,
			&execution.CreatedBy,
			&execution.CreatedAt,
			&execution.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		execution.Status = build.ExecutionStatus(status)
		executions = append(executions, execution)
	}

	return executions, total, rows.Err()
}

// ListRunningExecutions retrieves all currently running executions
func (r *PostgresBuildExecutionRepository) ListRunningExecutions(ctx context.Context) ([]build.BuildExecution, error) {
	query := `
		SELECT id, build_id, config_id, status, started_at, completed_at, 
		       duration_seconds, output, error_message, artifacts, metadata, created_by, created_at, updated_at
		FROM build_executions
		WHERE status = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryxContext(ctx, query, string(build.ExecutionRunning))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	executions := []build.BuildExecution{}
	for rows.Next() {
		var execution build.BuildExecution
		var status string

		err := rows.Scan(
			&execution.ID,
			&execution.BuildID,
			&execution.ConfigID,
			&status,
			&execution.StartedAt,
			&execution.CompletedAt,
			&execution.DurationSeconds,
			&execution.Output,
			&execution.ErrorMessage,
			&execution.Artifacts,
			&execution.Metadata,
			&execution.CreatedBy,
			&execution.CreatedAt,
			&execution.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		execution.Status = build.ExecutionStatus(status)
		executions = append(executions, execution)
	}

	return executions, rows.Err()
}

// GetRunningExecutionForConfig retrieves the running execution for a config
func (r *PostgresBuildExecutionRepository) GetRunningExecutionForConfig(ctx context.Context, configID uuid.UUID) (*build.BuildExecution, error) {
	query := `
		SELECT id, build_id, config_id, status, started_at, completed_at, 
		       duration_seconds, output, error_message, artifacts, metadata, created_by, created_at, updated_at
		FROM build_executions
		WHERE config_id = $1 AND status = $2
		LIMIT 1
	`

	var execution build.BuildExecution
	var status string

	err := r.db.QueryRowxContext(ctx, query, configID, string(build.ExecutionRunning)).Scan(
		&execution.ID,
		&execution.BuildID,
		&execution.ConfigID,
		&status,
		&execution.StartedAt,
		&execution.CompletedAt,
		&execution.DurationSeconds,
		&execution.Output,
		&execution.ErrorMessage,
		&execution.Artifacts,
		&execution.Metadata,
		&execution.CreatedBy,
		&execution.CreatedAt,
		&execution.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, build.ErrExecutionNotFound
	}
	if err != nil {
		return nil, err
	}

	execution.Status = build.ExecutionStatus(status)
	return &execution, nil
}

// AddLog adds a log entry for an execution
func (r *PostgresBuildExecutionRepository) AddLog(ctx context.Context, log *build.ExecutionLog) error {
	query := `
		INSERT INTO build_execution_logs 
		(id, execution_id, timestamp, level, message, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		log.ID,
		log.ExecutionID,
		log.Timestamp,
		string(log.Level),
		log.Message,
		log.Metadata,
	)

	return err
}

// GetLogs retrieves logs for an execution with pagination
func (r *PostgresBuildExecutionRepository) GetLogs(ctx context.Context, executionID uuid.UUID, limit, offset int) ([]build.ExecutionLog, int64, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM build_execution_logs WHERE execution_id = $1`
	var total int64
	if err := r.db.QueryRowxContext(ctx, countQuery, executionID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query := `
		SELECT id, execution_id, timestamp, level, message, metadata
		FROM build_execution_logs
		WHERE execution_id = $1
		ORDER BY timestamp ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryxContext(ctx, query, executionID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	logs := []build.ExecutionLog{}
	for rows.Next() {
		var log build.ExecutionLog
		var level string

		err := rows.Scan(
			&log.ID,
			&log.ExecutionID,
			&log.Timestamp,
			&level,
			&log.Message,
			&log.Metadata,
		)
		if err != nil {
			return nil, 0, err
		}

		log.Level = build.LogLevel(level)
		logs = append(logs, log)
	}

	return logs, total, rows.Err()
}

// UpdateExecutionStatus updates only the status of an execution
func (r *PostgresBuildExecutionRepository) UpdateExecutionStatus(ctx context.Context, executionID uuid.UUID, status build.ExecutionStatus) error {
	query := `
		UPDATE build_executions
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, string(status), time.Now(), executionID)
	return err
}

// UpdateExecutionMetadata updates only metadata for an execution.
func (r *PostgresBuildExecutionRepository) UpdateExecutionMetadata(ctx context.Context, executionID uuid.UUID, metadata []byte) error {
	query := `
		UPDATE build_executions
		SET metadata = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, metadata, time.Now(), executionID)
	return err
}

func (r *PostgresBuildExecutionRepository) TryAcquireMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
	query := `
		UPDATE build_executions
		SET monitor_owner = $1,
		    monitor_lease_expires_at = NOW() + ($2 * INTERVAL '1 second'),
		    updated_at = NOW()
		WHERE id = $3
		  AND status IN ('pending', 'running')
		  AND (
			monitor_owner IS NULL
			OR monitor_owner = $1
			OR monitor_lease_expires_at IS NULL
			OR monitor_lease_expires_at < NOW()
		  )
	`
	result, err := r.db.ExecContext(ctx, query, owner, int(ttl.Seconds()), executionID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (r *PostgresBuildExecutionRepository) RenewMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
	query := `
		UPDATE build_executions
		SET monitor_lease_expires_at = NOW() + ($1 * INTERVAL '1 second'),
		    updated_at = NOW()
		WHERE id = $2
		  AND monitor_owner = $3
		  AND status IN ('pending', 'running')
	`
	result, err := r.db.ExecContext(ctx, query, int(ttl.Seconds()), executionID, owner)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (r *PostgresBuildExecutionRepository) ReleaseMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string) error {
	query := `
		UPDATE build_executions
		SET monitor_owner = NULL,
		    monitor_lease_expires_at = NULL,
		    updated_at = NOW()
		WHERE id = $1 AND monitor_owner = $2
	`
	_, err := r.db.ExecContext(ctx, query, executionID, owner)
	return err
}

// DeleteOldExecutions deletes executions older than the specified duration
func (r *PostgresBuildExecutionRepository) DeleteOldExecutions(ctx context.Context, olderThan time.Duration) error {
	query := `
		DELETE FROM build_executions
		WHERE created_at < $1
	`

	cutoff := time.Now().Add(-olderThan)
	_, err := r.db.ExecContext(ctx, query, cutoff)
	return err
}

// GetBuildIDFromConfig retrieves the build ID for a config
func (r *PostgresBuildExecutionRepository) GetBuildIDFromConfig(ctx context.Context, configID uuid.UUID) (uuid.UUID, error) {
	query := `
		SELECT build_id
		FROM build_configs
		WHERE id = $1
	`

	var buildID uuid.UUID
	err := r.db.QueryRowxContext(ctx, query, configID).Scan(&buildID)

	if err == sql.ErrNoRows {
		return uuid.Nil, build.ErrInvalidConfigID
	}

	return buildID, err
}
