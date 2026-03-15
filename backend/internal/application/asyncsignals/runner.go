package asyncsignals

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	appsresmartbot "github.com/srikarm/image-factory/internal/application/sresmartbot"
	"go.uber.org/zap"
)

type RunnerConfig struct {
	Enabled                  bool
	Interval                 time.Duration
	BuildQueueThreshold      int64
	EmailQueueThreshold      int64
	MessagingOutboxThreshold int64
}

type backlogSnapshot struct {
	BuildQueueDepth        int64
	EmailQueueDepth        int64
	MessagingOutboxPending int64
}

func StartAsyncBacklogSignalRunner(
	logger *zap.Logger,
	db *sqlx.DB,
	processHealthStore *runtimehealth.Store,
	sreSmartBotService *appsresmartbot.Service,
	cfg RunnerConfig,
) {
	if processHealthStore == nil {
		return
	}
	if cfg.Interval < time.Minute {
		cfg.Interval = 2 * time.Minute
	}
	if cfg.BuildQueueThreshold < 1 {
		cfg.BuildQueueThreshold = 10
	}
	if cfg.EmailQueueThreshold < 1 {
		cfg.EmailQueueThreshold = 20
	}
	if cfg.MessagingOutboxThreshold < 1 {
		cfg.MessagingOutboxThreshold = 15
	}

	running := cfg.Enabled && db != nil
	message := "async backlog signal runner initialized"
	switch {
	case !cfg.Enabled:
		message = "async backlog signal runner disabled"
	case db == nil:
		message = "async backlog signal runner unavailable: database not configured"
	}

	processHealthStore.Upsert("async_backlog_signal_runner", runtimehealth.ProcessStatus{
		Enabled:      cfg.Enabled,
		Running:      running,
		LastActivity: time.Now().UTC(),
		Message:      message,
		Metrics: map[string]int64{
			"async_backlog_interval_seconds":     int64(cfg.Interval / time.Second),
			"build_queue_threshold":              cfg.BuildQueueThreshold,
			"email_queue_threshold":              cfg.EmailQueueThreshold,
			"messaging_outbox_threshold":         cfg.MessagingOutboxThreshold,
			"async_backlog_ticks_total":          0,
			"async_backlog_build_queue_depth":    0,
			"async_backlog_email_queue_depth":    0,
			"async_backlog_outbox_pending_count": 0,
		},
	})

	if !cfg.Enabled || db == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		ticksTotal := int64(0)
		previous := map[string]appsresmartbot.AsyncBacklogIssue{}

		runTick := func() {
			ticksTotal++
			now := time.Now().UTC()
			snapshot, err := collectBacklogSnapshot(context.Background(), db, processHealthStore)
			if err != nil {
				processHealthStore.Upsert("async_backlog_signal_runner", runtimehealth.ProcessStatus{
					Enabled:      true,
					Running:      true,
					LastActivity: now,
					Message:      fmt.Sprintf("failed to collect async backlog snapshot: %v", err),
					Metrics: map[string]int64{
						"async_backlog_interval_seconds": int64(cfg.Interval / time.Second),
						"build_queue_threshold":          cfg.BuildQueueThreshold,
						"email_queue_threshold":          cfg.EmailQueueThreshold,
						"messaging_outbox_threshold":     cfg.MessagingOutboxThreshold,
						"async_backlog_ticks_total":      ticksTotal,
					},
				})
				return
			}

			previous = appsresmartbot.ObserveAsyncBacklogSignals(
				context.Background(),
				sreSmartBotService,
				logger,
				appsresmartbot.AsyncBacklogSignalSnapshot{
					BuildQueueDepth:        snapshot.BuildQueueDepth,
					EmailQueueDepth:        snapshot.EmailQueueDepth,
					MessagingOutboxPending: snapshot.MessagingOutboxPending,
				},
				now,
				previous,
				appsresmartbot.AsyncBacklogThresholds{
					BuildQueueThreshold:      cfg.BuildQueueThreshold,
					EmailQueueThreshold:      cfg.EmailQueueThreshold,
					MessagingOutboxThreshold: cfg.MessagingOutboxThreshold,
				},
			)

			processHealthStore.Upsert("async_backlog_signal_runner", runtimehealth.ProcessStatus{
				Enabled:      true,
				Running:      true,
				LastActivity: now,
				Message:      fmt.Sprintf("async backlog snapshot processed: builds=%d emails=%d outbox=%d", snapshot.BuildQueueDepth, snapshot.EmailQueueDepth, snapshot.MessagingOutboxPending),
				Metrics: map[string]int64{
					"async_backlog_interval_seconds":     int64(cfg.Interval / time.Second),
					"build_queue_threshold":              cfg.BuildQueueThreshold,
					"email_queue_threshold":              cfg.EmailQueueThreshold,
					"messaging_outbox_threshold":         cfg.MessagingOutboxThreshold,
					"async_backlog_ticks_total":          ticksTotal,
					"async_backlog_build_queue_depth":    snapshot.BuildQueueDepth,
					"async_backlog_email_queue_depth":    snapshot.EmailQueueDepth,
					"async_backlog_outbox_pending_count": snapshot.MessagingOutboxPending,
				},
			})
		}

		runTick()
		for range ticker.C {
			runTick()
		}
	}()
}

func collectBacklogSnapshot(ctx context.Context, db *sqlx.DB, processHealthStore *runtimehealth.Store) (backlogSnapshot, error) {
	snapshot := backlogSnapshot{}

	if err := db.GetContext(ctx, &snapshot.BuildQueueDepth, `SELECT COALESCE(SUM(queue_depth), 0) FROM v_build_analytics`); err != nil && err != sql.ErrNoRows {
		return backlogSnapshot{}, fmt.Errorf("query build queue depth: %w", err)
	}

	if err := db.GetContext(ctx, &snapshot.EmailQueueDepth, `SELECT COUNT(*) FROM email_queue WHERE status = 'pending'`); err != nil && err != sql.ErrNoRows {
		return backlogSnapshot{}, fmt.Errorf("query email queue depth: %w", err)
	}

	if processHealthStore != nil {
		if status, ok := processHealthStore.GetStatus("messaging_outbox_relay"); ok && status.Metrics != nil {
			snapshot.MessagingOutboxPending = status.Metrics["messaging_outbox_pending_count"]
		}
	}

	return snapshot, nil
}
