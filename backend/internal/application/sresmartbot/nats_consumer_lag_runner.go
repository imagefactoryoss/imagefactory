package sresmartbot

import (
	"context"
	"fmt"
	"time"

	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type NATSConsumerLagRunnerConfig struct {
	Enabled                  bool
	Interval                 time.Duration
	PendingMessagesThreshold int64
	AckPendingThreshold      int64
	StalledDuration          time.Duration
}

func StartNATSConsumerLagSignalRunner(
	logger *zap.Logger,
	processHealthStore *runtimehealth.Store,
	provider interface {
		ConsumerLagSnapshots(ctx context.Context) ([]messaging.NATSConsumerLagSnapshot, error)
	},
	sreSmartBotService *Service,
	cfg NATSConsumerLagRunnerConfig,
) {
	if processHealthStore == nil {
		return
	}
	if cfg.Interval < 30*time.Second {
		cfg.Interval = 60 * time.Second
	}
	if cfg.PendingMessagesThreshold < 1 {
		cfg.PendingMessagesThreshold = 25
	}
	if cfg.AckPendingThreshold < 1 {
		cfg.AckPendingThreshold = 10
	}
	if cfg.StalledDuration < 30*time.Second {
		cfg.StalledDuration = 5 * time.Minute
	}

	running := cfg.Enabled && provider != nil
	message := "nats consumer lag signal runner initialized"
	switch {
	case !cfg.Enabled:
		message = "nats consumer lag signal runner disabled"
	case provider == nil:
		message = "nats consumer lag signal runner unavailable: NATS consumer lag provider not configured"
	}

	processHealthStore.Upsert("nats_consumer_lag_signal_runner", runtimehealth.ProcessStatus{
		Enabled:      cfg.Enabled,
		Running:      running,
		LastActivity: time.Now().UTC(),
		Message:      message,
		Metrics: map[string]int64{
			"nats_consumer_lag_interval_seconds":  int64(cfg.Interval / time.Second),
			"nats_consumer_pending_threshold":     cfg.PendingMessagesThreshold,
			"nats_consumer_ack_pending_threshold": cfg.AckPendingThreshold,
			"nats_consumer_stalled_seconds":       int64(cfg.StalledDuration / time.Second),
			"nats_consumer_visible_count":         0,
			"nats_consumer_lagging_count":         0,
			"nats_consumer_max_pending":           0,
		},
	})

	if !cfg.Enabled || provider == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		previousIssues := map[string]NATSConsumerLagIssue{}

		runTick := func() {
			now := time.Now().UTC()
			ctx, cancel := context.WithTimeout(context.Background(), consumerLagRunnerTimeout(cfg.Interval))
			defer cancel()

			snapshots, err := provider.ConsumerLagSnapshots(ctx)
			if err != nil {
				processHealthStore.Upsert("nats_consumer_lag_signal_runner", runtimehealth.ProcessStatus{
					Enabled:      true,
					Running:      true,
					LastActivity: now,
					Message:      fmt.Sprintf("nats consumer lag snapshot failed: %v", err),
					Metrics: map[string]int64{
						"nats_consumer_lag_interval_seconds":  int64(cfg.Interval / time.Second),
						"nats_consumer_pending_threshold":     cfg.PendingMessagesThreshold,
						"nats_consumer_ack_pending_threshold": cfg.AckPendingThreshold,
						"nats_consumer_stalled_seconds":       int64(cfg.StalledDuration / time.Second),
					},
				})
				if logger != nil {
					logger.Warn("Failed to capture NATS consumer lag snapshots", zap.Error(err))
				}
				return
			}

			laggingCount := int64(0)
			maxPending := int64(0)
			for _, snapshot := range snapshots {
				pendingCount := int64(snapshot.PendingCount)
				if pendingCount > 0 || snapshot.AckPendingCount > 0 {
					laggingCount++
				}
				if pendingCount > maxPending {
					maxPending = pendingCount
				}
			}

			processHealthStore.Upsert("nats_consumer_lag_signal_runner", runtimehealth.ProcessStatus{
				Enabled:      true,
				Running:      true,
				LastActivity: now,
				Message:      fmt.Sprintf("nats consumer lag visible=%d lagging=%d max_pending=%d", len(snapshots), laggingCount, maxPending),
				Metrics: map[string]int64{
					"nats_consumer_lag_interval_seconds":  int64(cfg.Interval / time.Second),
					"nats_consumer_pending_threshold":     cfg.PendingMessagesThreshold,
					"nats_consumer_ack_pending_threshold": cfg.AckPendingThreshold,
					"nats_consumer_stalled_seconds":       int64(cfg.StalledDuration / time.Second),
					"nats_consumer_visible_count":         int64(len(snapshots)),
					"nats_consumer_lagging_count":         laggingCount,
					"nats_consumer_max_pending":           maxPending,
				},
			})

			previousIssues = ObserveNATSConsumerLagSignals(
				context.Background(),
				sreSmartBotService,
				logger,
				snapshots,
				now,
				previousIssues,
				NATSConsumerLagThresholds{
					PendingMessagesThreshold: cfg.PendingMessagesThreshold,
					AckPendingThreshold:      cfg.AckPendingThreshold,
					StalledDuration:          cfg.StalledDuration,
				},
			)
		}

		runTick()
		for range ticker.C {
			runTick()
		}
	}()
}

func consumerLagRunnerTimeout(interval time.Duration) time.Duration {
	if interval <= 15*time.Second {
		return interval
	}
	return 15 * time.Second
}
