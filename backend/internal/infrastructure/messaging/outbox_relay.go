package messaging

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OutboxRelayConfig struct {
	Interval    time.Duration
	BatchSize   int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
	ClaimOwner  string
	ClaimLease  time.Duration
}

type OutboxRelaySnapshot struct {
	ReplaySuccessTotal int64
	ReplayFailureTotal int64
}

type OutboxRelay struct {
	store       OutboxStore
	externalBus EventBus
	config      OutboxRelayConfig
	logger      *zap.Logger
	successes   atomic.Int64
	failures    atomic.Int64
}

func NewOutboxRelay(store OutboxStore, externalBus EventBus, config OutboxRelayConfig, logger *zap.Logger) *OutboxRelay {
	if config.Interval <= 0 {
		config.Interval = 5 * time.Second
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.BaseBackoff <= 0 {
		config.BaseBackoff = 5 * time.Second
	}
	if config.MaxBackoff <= 0 {
		config.MaxBackoff = 5 * time.Minute
	}
	if config.ClaimLease <= 0 {
		config.ClaimLease = 30 * time.Second
	}
	if config.ClaimOwner == "" {
		host, _ := os.Hostname()
		if host == "" {
			host = "unknown-host"
		}
		config.ClaimOwner = fmt.Sprintf("%s-%s", host, uuid.NewString())
	}
	return &OutboxRelay{
		store:       store,
		externalBus: externalBus,
		config:      config,
		logger:      logger,
	}
}

func (r *OutboxRelay) Run(ctx context.Context) {
	if r == nil || r.store == nil || r.externalBus == nil {
		return
	}
	ticker := time.NewTicker(r.config.Interval)
	defer ticker.Stop()

	r.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.runOnce(ctx)
		}
	}
}

func (r *OutboxRelay) runOnce(ctx context.Context) {
	messages, err := r.store.ClaimDue(ctx, r.config.BatchSize, r.config.ClaimOwner, r.config.ClaimLease)
	if err != nil {
		if r.logger != nil {
			r.logger.Warn("Outbox relay failed to claim due messages", zap.Error(err))
		}
		return
	}
	if len(messages) == 0 {
		return
	}
	for _, msg := range messages {
		err := r.externalBus.Publish(ctx, msg.Event)
		if err == nil {
			r.successes.Add(1)
			_ = r.store.MarkPublished(ctx, msg.ID, r.config.ClaimOwner)
			continue
		}
		r.failures.Add(1)
		nextAttempt := time.Now().UTC().Add(r.backoffForAttempt(msg.PublishAttempts + 1))
		_ = r.store.MarkFailed(ctx, msg.ID, r.config.ClaimOwner, err.Error(), nextAttempt)
		if r.logger != nil {
			r.logger.Warn("Outbox relay publish failed",
				zap.String("event_type", msg.Event.Type),
				zap.String("event_id", msg.Event.ID),
				zap.String("next_attempt_at", nextAttempt.Format(time.RFC3339)),
				zap.Error(err))
		}
	}

	if r.logger != nil {
		pending, pendingErr := r.store.PendingCount(ctx)
		if pendingErr == nil {
			r.logger.Debug("Outbox relay cycle completed",
				zap.Int("processed", len(messages)),
				zap.Int64("pending", pending))
		}
	}
}

func (r *OutboxRelay) backoffForAttempt(attempt int) time.Duration {
	if attempt <= 0 {
		return r.config.BaseBackoff
	}
	backoff := r.config.BaseBackoff * time.Duration(1<<(attempt-1))
	if backoff <= 0 {
		return r.config.MaxBackoff
	}
	if backoff > r.config.MaxBackoff {
		return r.config.MaxBackoff
	}
	return backoff
}

func (r *OutboxRelay) ReplayOnce(ctx context.Context) (int, error) {
	if r == nil || r.store == nil || r.externalBus == nil {
		return 0, fmt.Errorf("outbox relay is not configured")
	}
	messages, err := r.store.ClaimDue(ctx, r.config.BatchSize, r.config.ClaimOwner, r.config.ClaimLease)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, msg := range messages {
		if pubErr := r.externalBus.Publish(ctx, msg.Event); pubErr != nil {
			r.failures.Add(1)
			nextAttempt := time.Now().UTC().Add(r.backoffForAttempt(msg.PublishAttempts + 1))
			_ = r.store.MarkFailed(ctx, msg.ID, r.config.ClaimOwner, pubErr.Error(), nextAttempt)
			continue
		}
		r.successes.Add(1)
		_ = r.store.MarkPublished(ctx, msg.ID, r.config.ClaimOwner)
		processed++
	}
	return processed, nil
}

func (r *OutboxRelay) Snapshot() OutboxRelaySnapshot {
	if r == nil {
		return OutboxRelaySnapshot{}
	}
	return OutboxRelaySnapshot{
		ReplaySuccessTotal: r.successes.Load(),
		ReplayFailureTotal: r.failures.Load(),
	}
}
