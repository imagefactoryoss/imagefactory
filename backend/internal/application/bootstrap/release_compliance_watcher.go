package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"github.com/srikarm/image-factory/internal/infrastructure/releasecompliance"
	"go.uber.org/zap"
)

type ReleaseComplianceWatcherConfig struct {
	Enabled       bool
	Interval      time.Duration
	Timeout       time.Duration
	BatchSize     int
	SchemaVersion string
	EventSource   string
	OnTick        func(ctx context.Context, detected []releasecompliance.DriftRecord, recovered []releasecompliance.DriftRecord, snapshot releasecompliance.Snapshot, err error, observedAt time.Time)
}

type ReleaseComplianceListFunc func(ctx context.Context, limit int) ([]releasecompliance.DriftRecord, error)
type ReleaseComplianceCountFunc func(ctx context.Context) (int64, error)
type ReleaseCompliancePublishFunc func(ctx context.Context, eventType string, record releasecompliance.DriftRecord, stateField string, stateValue string) error

func StartReleaseComplianceWatcher(
	logger *zap.Logger,
	processHealthStore *runtimehealth.Store,
	metrics *releasecompliance.Metrics,
	listCandidates ReleaseComplianceListFunc,
	countReleased ReleaseComplianceCountFunc,
	publish ReleaseCompliancePublishFunc,
	cfg ReleaseComplianceWatcherConfig,
) {
	if processHealthStore == nil {
		return
	}
	if cfg.Interval < 30*time.Second {
		cfg.Interval = 180 * time.Second
	}
	if cfg.Timeout < 10*time.Second {
		cfg.Timeout = 20 * time.Second
	}
	if cfg.Timeout >= cfg.Interval {
		cfg.Timeout = cfg.Interval - time.Second
		if cfg.Timeout < 10*time.Second {
			cfg.Timeout = 10 * time.Second
		}
	}
	if cfg.BatchSize < 1 {
		cfg.BatchSize = 500
	}

	processHealthStore.Upsert("quarantine_release_compliance_watcher", runtimehealth.ProcessStatus{
		Enabled:      cfg.Enabled,
		Running:      cfg.Enabled,
		LastActivity: time.Now().UTC(),
		Message:      "quarantine release compliance watcher initialized",
		Metrics:      releaseComplianceMetricsMap(snapshotMetrics(metrics)),
	})

	if !cfg.Enabled {
		if metrics != nil {
			metrics.RecordTick(0, 0)
		}
		processHealthStore.Upsert("quarantine_release_compliance_watcher", runtimehealth.ProcessStatus{
			Enabled:      false,
			Running:      false,
			LastActivity: time.Now().UTC(),
			Message:      "quarantine release compliance watcher disabled",
			Metrics:      releaseComplianceMetricsMap(snapshotMetrics(metrics)),
		})
		return
	}

	if logger != nil {
		logger.Info("Background process starting",
			zap.String("component", "quarantine_release_compliance_watcher"),
			zap.Duration("interval", cfg.Interval),
			zap.Duration("timeout", cfg.Timeout),
			zap.Int("batch_size", cfg.BatchSize),
		)
	}

	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()
		manager := releasecompliance.NewManager()

		runTick := func() {
			tickCtx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
			defer cancel()

			candidates, err := listCandidates(tickCtx, cfg.BatchSize)
			observedAt := time.Now().UTC()
			if err != nil {
				if metrics != nil {
					metrics.RecordFailure()
				}
				if cfg.OnTick != nil {
					cfg.OnTick(context.Background(), nil, nil, snapshotMetrics(metrics), err, observedAt)
				}
				processHealthStore.Upsert("quarantine_release_compliance_watcher", runtimehealth.ProcessStatus{
					Enabled:      true,
					Running:      true,
					LastActivity: observedAt,
					Message:      "watch tick failed",
					Metrics:      releaseComplianceMetricsMap(snapshotMetrics(metrics)),
				})
				if logger != nil {
					logger.Warn("Quarantine release compliance watcher tick failed", zap.Error(err))
				}
				return
			}

			releasedCount, err := countReleased(tickCtx)
			if err != nil {
				if metrics != nil {
					metrics.RecordFailure()
				}
				if logger != nil {
					logger.Warn("Quarantine release compliance watcher failed to count released artifacts", zap.Error(err))
				}
				return
			}

			detected, recovered := manager.Evaluate(candidates)
			if metrics != nil {
				metrics.AddDetected(int64(len(detected)))
				metrics.AddRecovered(int64(len(recovered)))
				metrics.RecordTick(int64(manager.ActiveCount()), releasedCount)
			}
			snapshot := snapshotMetrics(metrics)
			if cfg.OnTick != nil {
				cfg.OnTick(context.Background(), detected, recovered, snapshot, nil, observedAt)
			}

			for _, rec := range detected {
				if publish == nil {
					continue
				}
				if err := publish(context.Background(), messaging.EventTypeQuarantineReleaseDriftDetected, rec, "release_state", rec.ReleaseState); err != nil && logger != nil {
					logger.Warn("Failed to publish quarantine release drift detected event",
						zap.Error(err),
						zap.String("external_image_import_id", rec.ExternalImageImportID.String()),
					)
				}
			}
			for _, rec := range recovered {
				if publish == nil {
					continue
				}
				if err := publish(context.Background(), messaging.EventTypeQuarantineReleaseDriftRecovered, rec, "previous_release_state", rec.ReleaseState); err != nil && logger != nil {
					logger.Warn("Failed to publish quarantine release drift recovered event",
						zap.Error(err),
						zap.String("external_image_import_id", rec.ExternalImageImportID.String()),
					)
				}
			}

			processHealthStore.Upsert("quarantine_release_compliance_watcher", runtimehealth.ProcessStatus{
				Enabled:      true,
				Running:      true,
				LastActivity: observedAt,
				Message:      fmt.Sprintf("tick completed active_drift=%d detected=%d recovered=%d", snapshot.ActiveDriftCount, len(detected), len(recovered)),
				Metrics:      releaseComplianceMetricsMap(snapshot),
			})

			if logger != nil && (len(detected) > 0 || len(recovered) > 0 || snapshot.ActiveDriftCount > 0) {
				logger.Info("Quarantine release compliance watcher tick completed",
					zap.Int("detected", len(detected)),
					zap.Int("recovered", len(recovered)),
					zap.Int64("active_drift", snapshot.ActiveDriftCount),
					zap.Int64("released_count", snapshot.ReleasedCount),
				)
			}
		}

		runTick()
		for {
			<-ticker.C
			runTick()
		}
	}()
}

func snapshotMetrics(metrics *releasecompliance.Metrics) releasecompliance.Snapshot {
	if metrics == nil {
		return releasecompliance.Snapshot{}
	}
	return metrics.Snapshot()
}

func releaseComplianceMetricsMap(snapshot releasecompliance.Snapshot) map[string]int64 {
	return map[string]int64{
		"quarantine_release_compliance_watch_ticks_total":     snapshot.WatchTicksTotal,
		"quarantine_release_compliance_watch_failures_total":  snapshot.WatchFailuresTotal,
		"quarantine_release_compliance_drift_detected_total":  snapshot.DriftDetectedTotal,
		"quarantine_release_compliance_drift_recovered_total": snapshot.DriftRecoveredTotal,
		"quarantine_release_compliance_active_drift_count":    snapshot.ActiveDriftCount,
		"quarantine_release_compliance_released_count":        snapshot.ReleasedCount,
	}
}
