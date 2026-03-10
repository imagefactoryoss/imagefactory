package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	appdispatcher "github.com/srikarm/image-factory/internal/application/dispatcher"
	"go.uber.org/zap"
)

// DispatcherRuntimeRepository stores dispatcher liveness + metrics snapshots.
type DispatcherRuntimeRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewDispatcherRuntimeRepository creates a dispatcher runtime repository.
func NewDispatcherRuntimeRepository(db *sqlx.DB, logger *zap.Logger) appdispatcher.RuntimeStatusStore {
	return &DispatcherRuntimeRepository{
		db:     db,
		logger: logger,
	}
}

func (r *DispatcherRuntimeRepository) UpsertRuntimeStatus(ctx context.Context, status appdispatcher.RuntimeStatus) error {
	metricsJSON, err := json.Marshal(status.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal dispatcher metrics: %w", err)
	}

	query := `
		INSERT INTO dispatcher_runtime_status (
			instance_id, mode, running, last_heartbeat, metrics, updated_at
		) VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (instance_id) DO UPDATE SET
			mode = EXCLUDED.mode,
			running = EXCLUDED.running,
			last_heartbeat = EXCLUDED.last_heartbeat,
			metrics = EXCLUDED.metrics,
			updated_at = NOW()
	`

	if _, err := r.db.ExecContext(ctx, query, status.InstanceID, status.Mode, status.Running, status.LastHeartbeat.UTC(), metricsJSON); err != nil {
		return fmt.Errorf("failed to upsert dispatcher runtime status: %w", err)
	}

	return nil
}

func (r *DispatcherRuntimeRepository) GetLatestRuntimeStatus(ctx context.Context, mode string) (*appdispatcher.RuntimeStatus, error) {
	query := `
		SELECT instance_id, mode, running, last_heartbeat, metrics, updated_at
		FROM dispatcher_runtime_status
		WHERE mode = $1
		ORDER BY last_heartbeat DESC
		LIMIT 1
	`

	var instanceID string
	var modeOut string
	var running bool
	var lastHeartbeat time.Time
	var metricsJSON []byte
	var updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, mode).Scan(
		&instanceID,
		&modeOut,
		&running,
		&lastHeartbeat,
		&metricsJSON,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get dispatcher runtime status: %w", err)
	}

	var metrics appdispatcher.DispatcherMetricsSnapshot
	if len(metricsJSON) > 0 {
		if unmarshalErr := json.Unmarshal(metricsJSON, &metrics); unmarshalErr != nil {
			r.logger.Warn("Failed to unmarshal dispatcher runtime metrics", zap.Error(unmarshalErr), zap.String("instance_id", instanceID))
		}
	}

	return &appdispatcher.RuntimeStatus{
		InstanceID:    instanceID,
		Mode:          modeOut,
		Running:       running,
		LastHeartbeat: lastHeartbeat,
		Metrics:       metrics,
		UpdatedAt:     updatedAt,
	}, nil
}
