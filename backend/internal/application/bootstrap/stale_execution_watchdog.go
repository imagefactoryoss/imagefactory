package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type StaleExecutionWatchdogConfig struct {
	Interval      time.Duration
	BatchSize     int
	EventSource   string
	SchemaVersion string
}

type staleBuildRecoveryResult struct {
	BuildID     uuid.UUID `db:"build_id"`
	TenantID    uuid.UUID `db:"tenant_id"`
	ProjectID   uuid.UUID `db:"project_id"`
	BuildNumber int64     `db:"build_number"`
}

func StartStaleExecutionWatchdog(
	logger *zap.Logger,
	processHealthStore *runtimehealth.Store,
	db *sqlx.DB,
	eventBus messaging.EventBus,
	cfg StaleExecutionWatchdogConfig,
) {
	if processHealthStore == nil {
		return
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}

	go func() {
		processHealthStore.Upsert("stale_execution_watchdog", runtimehealth.ProcessStatus{
			Enabled:      true,
			Running:      true,
			LastActivity: time.Now().UTC(),
			Message:      "watchdog started",
		})
		reason := "Build execution monitor lease expired; marking execution as failed"
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		if logger != nil {
			logger.Info("Background process starting",
				zap.String("component", "stale_execution_lease_watchdog"),
				zap.Duration("interval", cfg.Interval),
				zap.Int("batch_limit", cfg.BatchSize),
			)
		}

		for {
			if _, err := recoverStaleExecutionLeases(
				context.Background(),
				db,
				logger,
				eventBus,
				cfg.EventSource,
				cfg.SchemaVersion,
				cfg.BatchSize,
				reason,
			); err != nil && logger != nil {
				logger.Warn("Stale execution lease recovery failed", zap.Error(err))
			}
			processHealthStore.Touch("stale_execution_watchdog")
			<-ticker.C
		}
	}()
}

func recoverStaleExecutionLeases(
	ctx context.Context,
	db *sqlx.DB,
	logger *zap.Logger,
	eventBus messaging.EventBus,
	eventSource string,
	schemaVersion string,
	limit int,
	reason string,
) (int, error) {
	if db == nil {
		return 0, nil
	}
	if limit <= 0 {
		limit = 100
	}
	if reason == "" {
		reason = "Build execution monitor lease expired; marking execution as failed"
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to start stale recovery transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	rows, err := tx.QueryxContext(ctx, `
		WITH stale AS (
			SELECT be.build_id, b.tenant_id, b.project_id, b.build_number
			FROM build_executions be
			INNER JOIN builds b ON b.id = be.build_id
			WHERE be.status IN ('running', 'monitoring')
			  AND be.monitor_lease_expires_at IS NOT NULL
			  AND be.monitor_lease_expires_at < NOW()
			ORDER BY be.monitor_lease_expires_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		),
		updated AS (
			UPDATE builds b
			SET status = 'failed',
			    error_message = $2,
			    completed_at = COALESCE(b.completed_at, NOW()),
			    updated_at = NOW()
			FROM stale s
			WHERE b.id = s.build_id
			  AND b.status NOT IN ('completed', 'failed', 'cancelled')
			RETURNING b.id AS build_id, b.tenant_id, b.project_id, b.build_number
		),
		exec_updated AS (
			UPDATE build_executions be
			SET status = 'failed',
			    completed_at = COALESCE(be.completed_at, NOW()),
			    error_message = $2,
			    updated_at = NOW()
			FROM updated u
			WHERE be.build_id = u.build_id
			  AND be.status IN ('running', 'monitoring')
			RETURNING be.build_id
		)
		SELECT u.build_id, u.tenant_id, u.project_id, u.build_number
		FROM updated u
	`, limit, reason)
	if err != nil {
		return 0, fmt.Errorf("failed to recover stale executions: %w", err)
	}
	defer rows.Close()

	results := make([]staleBuildRecoveryResult, 0)
	for rows.Next() {
		var rec staleBuildRecoveryResult
		if scanErr := rows.Scan(&rec.BuildID, &rec.TenantID, &rec.ProjectID, &rec.BuildNumber); scanErr != nil {
			return 0, fmt.Errorf("failed to scan stale recovery result: %w", scanErr)
		}
		results = append(results, rec)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return 0, fmt.Errorf("failed iterating stale recovery rows: %w", rowsErr)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit stale recovery transaction: %w", err)
	}

	for _, rec := range results {
		if eventBus == nil {
			continue
		}
		_ = eventBus.Publish(ctx, messaging.Event{
			Type:          messaging.EventTypeBuildExecutionFailed,
			Source:        eventSource,
			SchemaVersion: schemaVersion,
			TenantID:      rec.TenantID.String(),
			OccurredAt:    time.Now().UTC(),
			Payload: map[string]interface{}{
				"build_id":     rec.BuildID.String(),
				"build_number": fmt.Sprintf("%d", rec.BuildNumber),
				"project_id":   rec.ProjectID.String(),
				"status":       "failed",
				"message":      reason,
				"metadata": map[string]interface{}{
					"failure_type": "stale_execution_lease",
				},
			},
		})
	}

	if len(results) > 0 && logger != nil {
		logger.Warn("Recovered stale build executions after lease expiry", zap.Int("count", len(results)))
	}

	return len(results), nil
}
