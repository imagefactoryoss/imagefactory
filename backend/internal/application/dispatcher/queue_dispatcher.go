package dispatcher

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/infrastructure/metrics"
	"go.uber.org/zap"
)

// QueueDispatcherConfig controls dispatcher polling and throughput.
type QueueDispatcherConfig struct {
	PollInterval       time.Duration
	MaxDispatchPerTick int
	MaxRetries         int
	RetryBackoff       time.Duration
	RetryBackoffMax    time.Duration
}

// BuildDispatchService defines the minimal interface for dispatching a running build.
type BuildDispatchService interface {
	DispatchBuild(ctx context.Context, build *build.Build) error
}

// QueuedBuildDispatcher polls queued builds and dispatches them for execution.
type QueuedBuildDispatcher struct {
	repo                build.Repository
	service             BuildDispatchService
	systemConfigService build.SystemConfigService
	config              QueueDispatcherConfig
	logger              *zap.Logger
	metrics             *DispatcherMetrics
}

// NewQueuedBuildDispatcher creates a new dispatcher.
func NewQueuedBuildDispatcher(repo build.Repository, service BuildDispatchService, systemConfigService build.SystemConfigService, logger *zap.Logger, config QueueDispatcherConfig) *QueuedBuildDispatcher {
	if config.PollInterval <= 0 {
		config.PollInterval = 3 * time.Second
	}
	if config.MaxDispatchPerTick <= 0 {
		config.MaxDispatchPerTick = 1
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	if config.RetryBackoff <= 0 {
		config.RetryBackoff = 5 * time.Second
	}
	if config.RetryBackoffMax <= 0 {
		config.RetryBackoffMax = time.Minute
	}

	return &QueuedBuildDispatcher{
		repo:                repo,
		service:             service,
		systemConfigService: systemConfigService,
		config:              config,
		logger:              logger,
		metrics:             NewDispatcherMetrics(),
	}
}

// Run starts the dispatch loop until context cancellation.
func (d *QueuedBuildDispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.config.PollInterval)
	defer ticker.Stop()

	d.logger.Info("Build dispatcher started",
		zap.Duration("poll_interval", d.config.PollInterval),
		zap.Int("max_dispatch_per_tick", d.config.MaxDispatchPerTick),
	)

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("Build dispatcher stopped")
			return
		default:
		}

		_, _ = d.dispatchBatch(ctx)

		select {
		case <-ctx.Done():
			d.logger.Info("Build dispatcher stopped")
			return
		case <-ticker.C:
		}
	}
}

// RunOnce dispatches up to MaxDispatchPerTick builds and returns the count.
func (d *QueuedBuildDispatcher) RunOnce(ctx context.Context) (int, error) {
	return d.dispatchBatch(ctx)
}

func (d *QueuedBuildDispatcher) dispatchBatch(ctx context.Context) (int, error) {
	dispatched := 0
	for dispatched < d.config.MaxDispatchPerTick {
		claimMeasure := metrics.NewMeasurement(&d.metrics.claimLatency)
		buildToRun, err := d.repo.ClaimNextQueuedBuild(ctx)
		claimMeasure.Record()
		if err != nil {
			d.logger.Error("Failed to claim queued build", zap.Error(err))
			atomic.AddInt64(&d.metrics.claimErrors, 1)
			return dispatched, err
		}
		if buildToRun == nil {
			break
		}
		atomic.AddInt64(&d.metrics.claims, 1)
		d.logSchedulingOutcome(buildToRun, "claimed", build.BuildStatusQueued, build.BuildStatusRunning, "")

		if !d.withinConcurrencyLimit(ctx, buildToRun) {
			atomic.AddInt64(&d.metrics.skippedForLimit, 1)
			retryAt := time.Now().UTC().Add(d.config.RetryBackoff)
			_ = d.repo.RequeueBuild(ctx, buildToRun.ID(), retryAt, nil)
			d.logSchedulingOutcome(buildToRun, "requeued", build.BuildStatusRunning, build.BuildStatusQueued, "concurrency_limit_exceeded")
			return dispatched, nil
		}

		dispatchMeasure := metrics.NewMeasurement(&d.metrics.dispatchLatency)
		if err := d.service.DispatchBuild(ctx, buildToRun); err != nil {
			dispatchMeasure.Record()
			d.logger.Error("Failed to dispatch build", zap.Error(err), zap.String("build_id", buildToRun.ID().String()))
			atomic.AddInt64(&d.metrics.dispatchErrors, 1)
			if errors.Is(err, build.ErrBuildCapabilityNotEntitled) {
				errMsg := err.Error()
				_ = d.repo.UpdateStatus(ctx, buildToRun.ID(), build.BuildStatusFailed, nil, nil, &errMsg)
				d.logSchedulingOutcome(buildToRun, "failed", build.BuildStatusRunning, build.BuildStatusFailed, errMsg)
				continue
			}
			attempts := buildToRun.DispatchAttempts()
			if attempts >= d.config.MaxRetries {
				errMsg := err.Error()
				_ = d.repo.UpdateStatus(ctx, buildToRun.ID(), build.BuildStatusFailed, nil, nil, &errMsg)
				d.logSchedulingOutcome(buildToRun, "failed", build.BuildStatusRunning, build.BuildStatusFailed, errMsg)
			} else {
				retryAt := time.Now().UTC().Add(d.retryBackoff(attempts))
				errMsg := err.Error()
				_ = d.repo.RequeueBuild(ctx, buildToRun.ID(), retryAt, &errMsg)
				atomic.AddInt64(&d.metrics.requeues, 1)
				d.logSchedulingOutcome(buildToRun, "requeued", build.BuildStatusRunning, build.BuildStatusQueued, errMsg)
			}
			return dispatched, err
		}
		dispatchMeasure.Record()
		atomic.AddInt64(&d.metrics.dispatches, 1)
		d.logSchedulingOutcome(buildToRun, "dispatched", build.BuildStatusRunning, build.BuildStatusRunning, "")

		dispatched++
	}

	return dispatched, nil
}

func (d *QueuedBuildDispatcher) logSchedulingOutcome(b *build.Build, outcome string, fromStatus, toStatus build.BuildStatus, reason string) {
	fields := []zap.Field{
		zap.String("event_type", "build.scheduled"),
		zap.String("build_id", b.ID().String()),
		zap.String("tenant_id", b.TenantID().String()),
		zap.String("outcome", outcome),
		zap.String("from_status", string(fromStatus)),
		zap.String("to_status", string(toStatus)),
		zap.Int("dispatch_attempt", b.DispatchAttempts()),
	}
	if reason != "" {
		fields = append(fields, zap.String("reason", reason))
	}
	d.logger.Info("Build scheduling outcome", fields...)
}

func (d *QueuedBuildDispatcher) retryBackoff(attempts int) time.Duration {
	if attempts <= 0 {
		return d.config.RetryBackoff
	}
	backoff := d.config.RetryBackoff * time.Duration(attempts)
	if backoff > d.config.RetryBackoffMax {
		return d.config.RetryBackoffMax
	}
	return backoff
}

// DispatcherMetricsSnapshot represents a snapshot of dispatcher metrics.
type DispatcherMetricsSnapshot struct {
	Claims          int64   `json:"claims"`
	Dispatches      int64   `json:"dispatches"`
	ClaimErrors     int64   `json:"claim_errors"`
	DispatchErrors  int64   `json:"dispatch_errors"`
	Requeues        int64   `json:"requeues"`
	SkippedForLimit int64   `json:"skipped_for_limit"`
	ClaimCount      int64   `json:"claim_count"`
	ClaimMinMs      int64   `json:"claim_min_ms"`
	ClaimMaxMs      int64   `json:"claim_max_ms"`
	ClaimAvgMs      float64 `json:"claim_avg_ms"`
	DispatchCount   int64   `json:"dispatch_count"`
	DispatchMinMs   int64   `json:"dispatch_min_ms"`
	DispatchMaxMs   int64   `json:"dispatch_max_ms"`
	DispatchAvgMs   float64 `json:"dispatch_avg_ms"`
}

// DispatcherMetrics returns a snapshot of dispatcher metrics.
func (d *QueuedBuildDispatcher) DispatcherMetrics() DispatcherMetricsSnapshot {
	claimCount, _, claimMin, claimMax, claimAvg := d.metrics.claimLatency.GetStats()
	dispatchCount, _, dispatchMin, dispatchMax, dispatchAvg := d.metrics.dispatchLatency.GetStats()

	return DispatcherMetricsSnapshot{
		Claims:          atomic.LoadInt64(&d.metrics.claims),
		Dispatches:      atomic.LoadInt64(&d.metrics.dispatches),
		ClaimErrors:     atomic.LoadInt64(&d.metrics.claimErrors),
		DispatchErrors:  atomic.LoadInt64(&d.metrics.dispatchErrors),
		Requeues:        atomic.LoadInt64(&d.metrics.requeues),
		SkippedForLimit: atomic.LoadInt64(&d.metrics.skippedForLimit),
		ClaimCount:      claimCount,
		ClaimMinMs:      claimMin,
		ClaimMaxMs:      claimMax,
		ClaimAvgMs:      claimAvg,
		DispatchCount:   dispatchCount,
		DispatchMinMs:   dispatchMin,
		DispatchMaxMs:   dispatchMax,
		DispatchAvgMs:   dispatchAvg,
	}
}

func (d *QueuedBuildDispatcher) withinConcurrencyLimit(ctx context.Context, buildToRun *build.Build) bool {
	if d.systemConfigService == nil {
		return true
	}

	config, err := d.systemConfigService.GetBuildConfig(ctx, buildToRun.TenantID())
	if err != nil || config == nil {
		d.logger.Warn("Failed to load build config for concurrency check", zap.Error(err), zap.String("tenant_id", buildToRun.TenantID().String()))
		return true
	}

	runningCount, err := d.repo.CountByStatus(ctx, buildToRun.TenantID(), build.BuildStatusRunning)
	if err != nil {
		d.logger.Warn("Failed to count running builds for concurrency check", zap.Error(err), zap.String("tenant_id", buildToRun.TenantID().String()))
		return true
	}

	return runningCount <= config.MaxConcurrentJobs
}

// DispatcherMetrics tracks dispatcher performance and outcomes.
type DispatcherMetrics struct {
	claims          int64
	dispatches      int64
	claimErrors     int64
	dispatchErrors  int64
	requeues        int64
	skippedForLimit int64
	claimLatency    metrics.OperationMetrics
	dispatchLatency metrics.OperationMetrics
}

// NewDispatcherMetrics creates a new metrics container.
func NewDispatcherMetrics() *DispatcherMetrics {
	return &DispatcherMetrics{}
}
