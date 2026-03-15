package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	domainappsignals "github.com/srikarm/image-factory/internal/domain/appsignals"
	"go.uber.org/zap"
)

type AppSignalsRepository struct {
	db *sqlx.DB
}

func NewAppSignalsRepository(db *sqlx.DB, _ *zap.Logger) domainappsignals.Repository {
	return &AppSignalsRepository{db: db}
}

func (r *AppSignalsRepository) StoreHTTPWindow(ctx context.Context, snapshot domainappsignals.HTTPWindowSnapshot, retentionDays int) error {
	if r == nil || r.db == nil {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const insertWindow = `
		INSERT INTO app_http_signal_windows (
			request_count,
			server_error_count,
			client_error_count,
			total_latency_ms,
			average_latency_ms,
			max_latency_ms,
			window_started_at,
			window_ended_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`

	if _, err = tx.ExecContext(ctx, insertWindow,
		snapshot.RequestCount,
		snapshot.ServerErrorCount,
		snapshot.ClientErrorCount,
		snapshot.TotalLatencyMs,
		snapshot.AverageLatencyMs,
		snapshot.MaxLatencyMs,
		snapshot.WindowStartedAt.UTC(),
		snapshot.WindowEndedAt.UTC(),
	); err != nil {
		return fmt.Errorf("insert app http signal window: %w", err)
	}

	if retentionDays > 0 {
		cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
		if _, err = tx.ExecContext(ctx, `DELETE FROM app_http_signal_windows WHERE window_ended_at < $1`, cutoff); err != nil {
			return fmt.Errorf("delete expired app http signal windows: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit app http signal window transaction: %w", err)
	}
	return nil
}

func (r *AppSignalsRepository) ListRecentHTTPWindows(ctx context.Context, limit int) ([]domainappsignals.HTTPWindowRecord, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 12
	}

	rows := make([]domainappsignals.HTTPWindowRecord, 0, limit)
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT
			request_count,
			server_error_count,
			client_error_count,
			total_latency_ms,
			average_latency_ms,
			max_latency_ms,
			window_started_at,
			window_ended_at,
			created_at
		FROM app_http_signal_windows
		ORDER BY window_ended_at DESC
		LIMIT $1
	`, limit); err != nil {
		return nil, fmt.Errorf("list recent app http signal windows: %w", err)
	}
	return rows, nil
}
