package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/build"
)

// BuildExecutionRepository implements build.BuildExecutionRepository
type BuildExecutionRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewBuildExecutionRepository creates a new BuildExecutionRepository
func NewBuildExecutionRepository(db *sqlx.DB, logger *zap.Logger) build.BuildExecutionRepository {
	return &BuildExecutionRepository{
		db:     db,
		logger: logger,
	}
}

// SaveExecution saves a build execution to the database
func (r *BuildExecutionRepository) SaveExecution(ctx context.Context, execution *build.BuildExecution) error {
	query := `
		INSERT INTO build_executions (
			id, build_id, config_id, status, created_by, started_at,
			completed_at, error_message, artifacts, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`

	artifactsJSON, err := json.Marshal(execution.Artifacts)
	if err != nil {
		return fmt.Errorf("failed to marshal artifacts: %w", err)
	}

	metadataJSON, err := json.Marshal(execution.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		execution.ID,
		execution.BuildID,
		execution.ConfigID,
		execution.Status,
		execution.CreatedBy,
		execution.StartedAt,
		execution.CompletedAt,
		execution.ErrorMessage,
		artifactsJSON,
		metadataJSON,
		execution.CreatedAt,
		execution.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to save build execution", zap.Error(err), zap.String("execution_id", execution.ID.String()))
		return fmt.Errorf("failed to save build execution: %w", err)
	}

	r.logger.Info("Build execution saved", zap.String("execution_id", execution.ID.String()))
	return nil
}

// UpdateExecution updates a build execution in the database
func (r *BuildExecutionRepository) UpdateExecution(ctx context.Context, execution *build.BuildExecution) error {
	query := `
		UPDATE build_executions SET
			status = $1, completed_at = $2, error_message = $3,
			artifacts = $4, metadata = $5, updated_at = $6
		WHERE id = $7
	`

	artifactsJSON, err := json.Marshal(execution.Artifacts)
	if err != nil {
		return fmt.Errorf("failed to marshal artifacts: %w", err)
	}

	metadataJSON, err := json.Marshal(execution.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query,
		execution.Status,
		execution.CompletedAt,
		execution.ErrorMessage,
		artifactsJSON,
		metadataJSON,
		time.Now().UTC(),
		execution.ID,
	)

	if err != nil {
		r.logger.Error("Failed to update build execution", zap.Error(err), zap.String("execution_id", execution.ID.String()))
		return fmt.Errorf("failed to update build execution: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("build execution not found: %s", execution.ID)
	}

	r.logger.Info("Build execution updated", zap.String("execution_id", execution.ID.String()))
	return nil
}

// GetExecution retrieves a build execution by ID
func (r *BuildExecutionRepository) GetExecution(ctx context.Context, id uuid.UUID) (*build.BuildExecution, error) {
	query := `
		SELECT id, build_id, config_id, status, created_by, started_at,
			   completed_at, error_message, artifacts, metadata, created_at, updated_at
		FROM build_executions
		WHERE id = $1
	`

	var execution build.BuildExecution
	var artifactsJSON []byte
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&execution.ID,
		&execution.BuildID,
		&execution.ConfigID,
		&execution.Status,
		&execution.CreatedBy,
		&execution.StartedAt,
		&execution.CompletedAt,
		&execution.ErrorMessage,
		&artifactsJSON,
		&metadataJSON,
		&execution.CreatedAt,
		&execution.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("build execution not found: %s", id)
		}
		r.logger.Error("Failed to get build execution", zap.Error(err), zap.String("execution_id", id.String()))
		return nil, fmt.Errorf("failed to get build execution: %w", err)
	}

	if err := json.Unmarshal(artifactsJSON, &execution.Artifacts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal artifacts: %w", err)
	}
	if err := json.Unmarshal(metadataJSON, &execution.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &execution, nil
}

// GetBuildExecutions retrieves build executions for a build with pagination
func (r *BuildExecutionRepository) GetBuildExecutions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]build.BuildExecution, int64, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM build_executions WHERE build_id = $1`
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, buildID).Scan(&total)
	if err != nil {
		r.logger.Error("Failed to get build executions count", zap.Error(err), zap.String("build_id", buildID.String()))
		return nil, 0, fmt.Errorf("failed to get build executions count: %w", err)
	}

	// Get executions with pagination
	query := `
		SELECT id, build_id, config_id, status, created_by, started_at,
			   completed_at, error_message, artifacts, metadata, created_at, updated_at
		FROM build_executions
		WHERE build_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, buildID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to get build executions", zap.Error(err), zap.String("build_id", buildID.String()))
		return nil, 0, fmt.Errorf("failed to get build executions: %w", err)
	}
	defer rows.Close()

	var executions []build.BuildExecution
	for rows.Next() {
		var execution build.BuildExecution
		var artifactsJSON []byte
		var metadataJSON []byte

		err := rows.Scan(
			&execution.ID,
			&execution.BuildID,
			&execution.ConfigID,
			&execution.Status,
			&execution.CreatedBy,
			&execution.StartedAt,
			&execution.CompletedAt,
			&execution.ErrorMessage,
			&artifactsJSON,
			&metadataJSON,
			&execution.CreatedAt,
			&execution.UpdatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan build execution", zap.Error(err))
			return nil, 0, fmt.Errorf("failed to scan build execution: %w", err)
		}

		if err := json.Unmarshal(artifactsJSON, &execution.Artifacts); err != nil {
			r.logger.Error("Failed to unmarshal artifacts", zap.Error(err))
			return nil, 0, fmt.Errorf("failed to unmarshal artifacts: %w", err)
		}
		if err := json.Unmarshal(metadataJSON, &execution.Metadata); err != nil {
			r.logger.Error("Failed to unmarshal metadata", zap.Error(err))
			return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		executions = append(executions, execution)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating build executions", zap.Error(err))
		return nil, 0, fmt.Errorf("error iterating build executions: %w", err)
	}

	return executions, total, nil
}

// ListRunningExecutions retrieves all running build executions
func (r *BuildExecutionRepository) ListRunningExecutions(ctx context.Context) ([]build.BuildExecution, error) {
	query := `
		SELECT id, build_id, config_id, status, created_by, started_at,
			   completed_at, error_message, artifacts, metadata, created_at, updated_at
		FROM build_executions
		WHERE status IN ('pending', 'running')
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("Failed to list running executions", zap.Error(err))
		return nil, fmt.Errorf("failed to list running executions: %w", err)
	}
	defer rows.Close()

	var executions []build.BuildExecution
	for rows.Next() {
		var execution build.BuildExecution
		var artifactsJSON []byte
		var metadataJSON []byte

		err := rows.Scan(
			&execution.ID,
			&execution.BuildID,
			&execution.ConfigID,
			&execution.Status,
			&execution.CreatedBy,
			&execution.StartedAt,
			&execution.CompletedAt,
			&execution.ErrorMessage,
			&artifactsJSON,
			&metadataJSON,
			&execution.CreatedAt,
			&execution.UpdatedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan running execution", zap.Error(err))
			return nil, fmt.Errorf("failed to scan running execution: %w", err)
		}

		if err := json.Unmarshal(artifactsJSON, &execution.Artifacts); err != nil {
			r.logger.Error("Failed to unmarshal artifacts for running execution", zap.Error(err))
			return nil, fmt.Errorf("failed to unmarshal artifacts: %w", err)
		}
		if err := json.Unmarshal(metadataJSON, &execution.Metadata); err != nil {
			r.logger.Error("Failed to unmarshal metadata for running execution", zap.Error(err))
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		executions = append(executions, execution)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating running executions", zap.Error(err))
		return nil, fmt.Errorf("error iterating running executions: %w", err)
	}

	return executions, nil
}

// GetRunningExecutionForConfig retrieves the running execution for a config
func (r *BuildExecutionRepository) GetRunningExecutionForConfig(ctx context.Context, configID uuid.UUID) (*build.BuildExecution, error) {
	query := `
		SELECT id, build_id, config_id, status, created_by, started_at,
			   completed_at, error_message, artifacts, metadata, created_at, updated_at
		FROM build_executions
		WHERE config_id = $1 AND status IN ('pending', 'running')
		ORDER BY created_at DESC
		LIMIT 1
	`

	var execution build.BuildExecution
	var artifactsJSON []byte
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, configID).Scan(
		&execution.ID,
		&execution.BuildID,
		&execution.ConfigID,
		&execution.Status,
		&execution.CreatedBy,
		&execution.StartedAt,
		&execution.CompletedAt,
		&execution.ErrorMessage,
		&artifactsJSON,
		&metadataJSON,
		&execution.CreatedAt,
		&execution.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No running execution found
		}
		r.logger.Error("Failed to get running execution for config", zap.Error(err), zap.String("config_id", configID.String()))
		return nil, fmt.Errorf("failed to get running execution for config: %w", err)
	}

	if err := json.Unmarshal(artifactsJSON, &execution.Artifacts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal artifacts: %w", err)
	}
	if err := json.Unmarshal(metadataJSON, &execution.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &execution, nil
}

// AddLog adds a log entry for a build execution
func (r *BuildExecutionRepository) AddLog(ctx context.Context, log *build.ExecutionLog) error {
	query := `
		INSERT INTO build_execution_logs (
			id, execution_id, level, message, metadata, timestamp
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
	`

	metadataJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal log metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		log.ID,
		log.ExecutionID,
		log.Level,
		log.Message,
		metadataJSON,
		log.Timestamp,
	)

	if err != nil {
		r.logger.Error("Failed to add execution log", zap.Error(err), zap.String("execution_id", log.ExecutionID.String()))
		return fmt.Errorf("failed to add execution log: %w", err)
	}

	return nil
}

// GetLogs retrieves logs for a build execution with pagination
func (r *BuildExecutionRepository) GetLogs(ctx context.Context, executionID uuid.UUID, limit, offset int) ([]build.ExecutionLog, int64, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM build_execution_logs WHERE execution_id = $1`
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, executionID).Scan(&total)
	if err != nil {
		r.logger.Error("Failed to get execution logs count", zap.Error(err), zap.String("execution_id", executionID.String()))
		return nil, 0, fmt.Errorf("failed to get execution logs count: %w", err)
	}

	// Get logs with pagination
	query := `
		SELECT id, execution_id, level, message, metadata, timestamp
		FROM build_execution_logs
		WHERE execution_id = $1
		ORDER BY timestamp ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, executionID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to get execution logs", zap.Error(err), zap.String("execution_id", executionID.String()))
		return nil, 0, fmt.Errorf("failed to get execution logs: %w", err)
	}
	defer rows.Close()

	var logs []build.ExecutionLog
	for rows.Next() {
		var log build.ExecutionLog
		var metadataJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.ExecutionID,
			&log.Level,
			&log.Message,
			&metadataJSON,
			&log.Timestamp,
		)

		if err != nil {
			r.logger.Error("Failed to scan execution log", zap.Error(err))
			return nil, 0, fmt.Errorf("failed to scan execution log: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &log.Metadata); err != nil {
			r.logger.Error("Failed to unmarshal log metadata", zap.Error(err))
			return nil, 0, fmt.Errorf("failed to unmarshal log metadata: %w", err)
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating execution logs", zap.Error(err))
		return nil, 0, fmt.Errorf("error iterating execution logs: %w", err)
	}

	return logs, total, nil
}

// UpdateExecutionStatus updates the status of a build execution
func (r *BuildExecutionRepository) UpdateExecutionStatus(ctx context.Context, executionID uuid.UUID, status build.ExecutionStatus) error {
	query := `
		UPDATE build_executions SET
			status = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, status, time.Now().UTC(), executionID)
	if err != nil {
		r.logger.Error("Failed to update execution status", zap.Error(err), zap.String("execution_id", executionID.String()))
		return fmt.Errorf("failed to update execution status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("build execution not found: %s", executionID)
	}

	r.logger.Info("Execution status updated", zap.String("execution_id", executionID.String()), zap.String("status", string(status)))
	return nil
}

// UpdateExecutionMetadata updates metadata for a build execution.
func (r *BuildExecutionRepository) UpdateExecutionMetadata(ctx context.Context, executionID uuid.UUID, metadata []byte) error {
	query := `
		UPDATE build_executions SET
			metadata = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, metadata, time.Now().UTC(), executionID)
	if err != nil {
		r.logger.Error("Failed to update execution metadata", zap.Error(err), zap.String("execution_id", executionID.String()))
		return fmt.Errorf("failed to update execution metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("build execution not found: %s", executionID)
	}
	return nil
}

func (r *BuildExecutionRepository) TryAcquireMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
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
		return false, fmt.Errorf("failed to acquire monitoring lease: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return rowsAffected > 0, nil
}

func (r *BuildExecutionRepository) RenewMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
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
		return false, fmt.Errorf("failed to renew monitoring lease: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return rowsAffected > 0, nil
}

func (r *BuildExecutionRepository) ReleaseMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string) error {
	query := `
		UPDATE build_executions
		SET monitor_owner = NULL,
		    monitor_lease_expires_at = NULL,
		    updated_at = NOW()
		WHERE id = $1 AND monitor_owner = $2
	`
	if _, err := r.db.ExecContext(ctx, query, executionID, owner); err != nil {
		return fmt.Errorf("failed to release monitoring lease: %w", err)
	}
	return nil
}

// DeleteOldExecutions deletes build executions older than the specified duration
func (r *BuildExecutionRepository) DeleteOldExecutions(ctx context.Context, olderThan time.Duration) error {
	cutoffTime := time.Now().UTC().Add(-olderThan)

	query := `DELETE FROM build_executions WHERE created_at < $1`
	result, err := r.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		r.logger.Error("Failed to delete old executions", zap.Error(err))
		return fmt.Errorf("failed to delete old executions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	r.logger.Info("Old executions deleted", zap.Int64("count", rowsAffected))
	return nil
}

// GetBuildIDFromConfig retrieves the build ID for a given config ID
func (r *BuildExecutionRepository) GetBuildIDFromConfig(ctx context.Context, configID uuid.UUID) (uuid.UUID, error) {
	query := `
		SELECT b.id
		FROM builds b
		JOIN build_configs bc ON b.id = bc.build_id
		WHERE bc.id = $1
	`

	var buildID uuid.UUID
	err := r.db.QueryRowContext(ctx, query, configID).Scan(&buildID)
	if err != nil {
		if err == sql.ErrNoRows {
			return uuid.Nil, fmt.Errorf("build not found for config: %s", configID)
		}
		r.logger.Error("Failed to get build ID from config", zap.Error(err), zap.String("config_id", configID.String()))
		return uuid.Nil, fmt.Errorf("failed to get build ID from config: %w", err)
	}

	return buildID, nil
}
