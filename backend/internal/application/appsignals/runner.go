package appsignals

import (
	"context"
	"fmt"
	"time"

	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	appsresmartbot "github.com/srikarm/image-factory/internal/application/sresmartbot"
	domainappsignals "github.com/srikarm/image-factory/internal/domain/appsignals"
	"go.uber.org/zap"
)

type HTTPSignalSnapshotter interface {
	SnapshotAndReset(now time.Time) HTTPRequestSignalSnapshot
}

type HTTPRequestSignalSnapshot struct {
	WindowStartedAt  time.Time
	WindowEndedAt    time.Time
	RequestCount     int64
	ServerErrorCount int64
	ClientErrorCount int64
	TotalLatencyMs   int64
	MaxLatencyMs     int64
}

type HTTPSignalSnapshotterFunc func(now time.Time) HTTPRequestSignalSnapshot

func (f HTTPSignalSnapshotterFunc) SnapshotAndReset(now time.Time) HTTPRequestSignalSnapshot {
	return f(now)
}

type RunnerConfig struct {
	Enabled                 bool
	Interval                time.Duration
	RetentionDays           int
	MinRequestCount         int64
	ErrorRatePercent        int
	AverageLatencyThreshold int64
	RequestVolumeThreshold  int64
}

func StartHTTPGoldenSignalRunner(
	logger *zap.Logger,
	processHealthStore *runtimehealth.Store,
	store HTTPSignalSnapshotter,
	repo domainappsignals.Repository,
	sreSmartBotService *appsresmartbot.Service,
	cfg RunnerConfig,
) {
	if cfg.Interval < time.Minute {
		cfg.Interval = 2 * time.Minute
	}
	if cfg.RetentionDays < 1 {
		cfg.RetentionDays = 7
	}
	if cfg.MinRequestCount < 1 {
		cfg.MinRequestCount = 20
	}
	if cfg.ErrorRatePercent < 1 {
		cfg.ErrorRatePercent = 10
	}
	if cfg.AverageLatencyThreshold < 1 {
		cfg.AverageLatencyThreshold = 800
	}
	if cfg.RequestVolumeThreshold < 1 {
		cfg.RequestVolumeThreshold = 250
	}

	running := cfg.Enabled && store != nil
	message := "http golden signal runner initialized"
	switch {
	case !cfg.Enabled:
		message = "http golden signal runner disabled"
	case store == nil:
		message = "http golden signal runner unavailable: request signal store not configured"
	}

	processHealthStore.Upsert("http_golden_signal_runner", runtimehealth.ProcessStatus{
		Enabled:      cfg.Enabled,
		Running:      running,
		LastActivity: time.Now().UTC(),
		Message:      message,
		Metrics: map[string]int64{
			"http_signal_interval_seconds":         int64(cfg.Interval / time.Second),
			"http_signal_min_request_count":        cfg.MinRequestCount,
			"http_signal_error_rate_percent":       int64(cfg.ErrorRatePercent),
			"http_signal_avg_latency_threshold_ms": cfg.AverageLatencyThreshold,
			"http_signal_request_volume_threshold": cfg.RequestVolumeThreshold,
		},
	})
	if !running {
		return
	}

	logger.Info("Background process starting",
		zap.String("component", "http_golden_signal_runner"),
		zap.Duration("interval", cfg.Interval),
		zap.Int64("min_request_count", cfg.MinRequestCount),
		zap.Int("error_rate_percent", cfg.ErrorRatePercent),
		zap.Int64("avg_latency_threshold_ms", cfg.AverageLatencyThreshold),
		zap.Int64("request_volume_threshold", cfg.RequestVolumeThreshold),
	)

	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		var previous map[string]appsresmartbot.HTTPGoldenSignalIssue
		var ticksTotal int64

		runTick := func() {
			ticksTotal++
			now := time.Now().UTC()
			snapshot := store.SnapshotAndReset(now)
			if repo != nil {
				_ = repo.StoreHTTPWindow(context.Background(), domainappsignals.HTTPWindowSnapshot{
					RequestCount:     snapshot.RequestCount,
					ServerErrorCount: snapshot.ServerErrorCount,
					ClientErrorCount: snapshot.ClientErrorCount,
					TotalLatencyMs:   snapshot.TotalLatencyMs,
					AverageLatencyMs: averageLatency(snapshot),
					MaxLatencyMs:     snapshot.MaxLatencyMs,
					WindowStartedAt:  snapshot.WindowStartedAt,
					WindowEndedAt:    snapshot.WindowEndedAt,
				}, cfg.RetentionDays)
			}
			previous = appsresmartbot.ObserveHTTPGoldenSignals(
				context.Background(),
				sreSmartBotService,
				logger,
				appsresmartbot.HTTPRequestSignalSnapshot{
					WindowStartedAt:  snapshot.WindowStartedAt,
					WindowEndedAt:    snapshot.WindowEndedAt,
					RequestCount:     snapshot.RequestCount,
					ServerErrorCount: snapshot.ServerErrorCount,
					ClientErrorCount: snapshot.ClientErrorCount,
					TotalLatencyMs:   snapshot.TotalLatencyMs,
					MaxLatencyMs:     snapshot.MaxLatencyMs,
				},
				now,
				previous,
				appsresmartbot.HTTPGoldenSignalThresholds{
					MinRequestCount:         cfg.MinRequestCount,
					ErrorRatePercent:        cfg.ErrorRatePercent,
					AverageLatencyThreshold: cfg.AverageLatencyThreshold,
					RequestVolumeThreshold:  cfg.RequestVolumeThreshold,
				},
			)
			processHealthStore.Upsert("http_golden_signal_runner", runtimehealth.ProcessStatus{
				Enabled:      true,
				Running:      true,
				LastActivity: now,
				Message:      fmt.Sprintf("processed %d requests in latest window", snapshot.RequestCount),
				Metrics: map[string]int64{
					"http_signal_ticks_total":             ticksTotal,
					"http_signal_last_request_count":      snapshot.RequestCount,
					"http_signal_last_server_error_count": snapshot.ServerErrorCount,
					"http_signal_last_client_error_count": snapshot.ClientErrorCount,
					"http_signal_last_total_latency_ms":   snapshot.TotalLatencyMs,
					"http_signal_last_avg_latency_ms":     averageLatency(snapshot),
					"http_signal_last_max_latency_ms":     snapshot.MaxLatencyMs,
				},
			})
		}

		runTick()
		for range ticker.C {
			runTick()
		}
	}()
}

func averageLatency(snapshot HTTPRequestSignalSnapshot) int64 {
	if snapshot.RequestCount <= 0 {
		return 0
	}
	return snapshot.TotalLatencyMs / snapshot.RequestCount
}
