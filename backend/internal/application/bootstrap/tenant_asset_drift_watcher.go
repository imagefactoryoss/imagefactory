package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	domaininfrastructure "github.com/srikarm/image-factory/internal/domain/infrastructure"
	domainsystemconfig "github.com/srikarm/image-factory/internal/domain/systemconfig"
	"go.uber.org/zap"
)

type TenantAssetDriftWatcherConfig struct {
	DefaultEnabled         bool
	DefaultIntervalSeconds int
	DefaultTimeoutSeconds  int
	DefaultBatchSize       int
	OnTick                 func(ctx context.Context, result *domaininfrastructure.TenantAssetDriftWatchTickResult, metrics domaininfrastructure.TenantAssetDriftMetricsSnapshot, err error, observedAt time.Time)
}

type tenantAssetDriftWatcherRuntimeConfig struct {
	enabled  bool
	interval time.Duration
	timeout  time.Duration
	source   string
	batch    int
}

func StartTenantAssetDriftWatcher(
	logger *zap.Logger,
	processHealthStore *runtimehealth.Store,
	systemConfigService interface {
		GetConfigByTypeAndKey(ctx context.Context, tenantID *uuid.UUID, configType domainsystemconfig.ConfigType, configKey string) (*domainsystemconfig.SystemConfig, error)
	},
	infrastructureService interface {
		RunTenantAssetDriftWatchTick(ctx context.Context, pageSize int) (*domaininfrastructure.TenantAssetDriftWatchTickResult, error)
		GetTenantAssetDriftMetrics() domaininfrastructure.TenantAssetDriftMetricsSnapshot
	},
	cfg TenantAssetDriftWatcherConfig,
) {
	if processHealthStore == nil {
		return
	}

	loadConfig := func(ctx context.Context) tenantAssetDriftWatcherRuntimeConfig {
		out := tenantAssetDriftWatcherRuntimeConfig{
			enabled:  cfg.DefaultEnabled,
			interval: time.Duration(cfg.DefaultIntervalSeconds) * time.Second,
			timeout:  time.Duration(cfg.DefaultTimeoutSeconds) * time.Second,
			batch:    cfg.DefaultBatchSize,
			source:   "env",
		}
		if out.interval < 30*time.Second {
			out.interval = 30 * time.Second
		}
		if out.timeout < 10*time.Second {
			out.timeout = 10 * time.Second
		}
		if out.timeout >= out.interval {
			out.timeout = out.interval - time.Second
			if out.timeout < 10*time.Second {
				out.timeout = 10 * time.Second
			}
		}
		if out.batch < 1 {
			out.batch = 200
		}

		if systemConfigService == nil {
			return out
		}
		runtimeConfig, err := systemConfigService.GetConfigByTypeAndKey(ctx, nil, domainsystemconfig.ConfigTypeRuntimeServices, "runtime_services")
		if err != nil || runtimeConfig == nil {
			return out
		}
		runtimeServicesConfig, cfgErr := runtimeConfig.GetRuntimeServicesConfig()
		if cfgErr != nil || runtimeServicesConfig == nil {
			return out
		}
		if runtimeServicesConfig.TenantAssetDriftWatcherEnabled != nil {
			out.enabled = *runtimeServicesConfig.TenantAssetDriftWatcherEnabled
			out.source = "system_config"
		}
		if runtimeServicesConfig.TenantAssetDriftWatcherIntervalSeconds >= 30 {
			out.interval = time.Duration(runtimeServicesConfig.TenantAssetDriftWatcherIntervalSeconds) * time.Second
			out.source = "system_config"
		}
		if out.timeout >= out.interval {
			out.timeout = out.interval - time.Second
			if out.timeout < 10*time.Second {
				out.timeout = 10 * time.Second
			}
		}
		return out
	}

	initialCfg := loadConfig(context.Background())
	processHealthStore.Upsert("tenant_asset_drift_watcher", runtimehealth.ProcessStatus{
		Enabled:      initialCfg.enabled,
		Running:      initialCfg.enabled,
		LastActivity: time.Now().UTC(),
		Message:      "tenant asset drift watcher initialized",
	})
	if logger != nil {
		logger.Info("Background process starting",
			zap.String("component", "tenant_asset_drift_watcher"),
			zap.Duration("interval", initialCfg.interval),
			zap.Duration("timeout", initialCfg.timeout),
			zap.Int("batch_size", initialCfg.batch),
			zap.String("config_source", initialCfg.source),
			zap.Bool("enabled", initialCfg.enabled),
		)
	}

	go func() {
		ticker := time.NewTicker(initialCfg.interval)
		defer ticker.Stop()
		currentInterval := initialCfg.interval

		runTick := func() {
			currentCfg := loadConfig(context.Background())
			if !currentCfg.enabled {
				processHealthStore.Upsert("tenant_asset_drift_watcher", runtimehealth.ProcessStatus{
					Enabled:      false,
					Running:      false,
					LastActivity: time.Now().UTC(),
					Message:      "tenant asset drift watcher disabled via runtime_services config",
				})
				return
			}
			if currentCfg.interval != currentInterval {
				ticker.Reset(currentCfg.interval)
				currentInterval = currentCfg.interval
				if logger != nil {
					logger.Info("Tenant asset drift watcher interval updated",
						zap.Duration("interval", currentCfg.interval),
						zap.String("config_source", currentCfg.source),
					)
				}
			}
			if infrastructureService == nil {
				processHealthStore.Upsert("tenant_asset_drift_watcher", runtimehealth.ProcessStatus{
					Enabled:      true,
					Running:      false,
					LastActivity: time.Now().UTC(),
					Message:      "infrastructure service not configured",
				})
				return
			}

			tickCtx, cancel := context.WithTimeout(context.Background(), currentCfg.timeout)
			defer cancel()

			result, err := infrastructureService.RunTenantAssetDriftWatchTick(tickCtx, currentCfg.batch)
			driftMetrics := infrastructureService.GetTenantAssetDriftMetrics()
			observedAt := time.Now().UTC()
			if err != nil {
				if logger != nil {
					logger.Warn("Tenant asset drift watcher tick failed", zap.Error(err))
				}
				if cfg.OnTick != nil {
					cfg.OnTick(context.Background(), nil, driftMetrics, err, observedAt)
				}
				processHealthStore.Upsert("tenant_asset_drift_watcher", runtimehealth.ProcessStatus{
					Enabled:      true,
					Running:      true,
					LastActivity: observedAt,
					Message:      "watch tick failed",
					Metrics:      tenantAssetDriftMetricsMap(driftMetrics),
				})
				return
			}
			if cfg.OnTick != nil {
				cfg.OnTick(context.Background(), result, driftMetrics, nil, observedAt)
			}

			processHealthStore.Upsert("tenant_asset_drift_watcher", runtimehealth.ProcessStatus{
				Enabled:      true,
				Running:      true,
				LastActivity: observedAt,
				Message:      fmt.Sprintf("tick completed namespaces=%d current=%d stale=%d unknown=%d failed=%d", result.TotalNamespaces, result.Current, result.Stale, result.Unknown, result.Failed),
				Metrics:      tenantAssetDriftMetricsMap(driftMetrics),
			})
			if logger != nil && (result.TotalNamespaces > 0 || result.Failed > 0 || result.Stale > 0) {
				logger.Info("Tenant asset drift watcher tick completed",
					zap.Int("total_providers", result.TotalProviders),
					zap.Int("attempted", result.Attempted),
					zap.Int("succeeded", result.Succeeded),
					zap.Int("failed", result.Failed),
					zap.Int("skipped", result.Skipped),
					zap.Int("total_namespaces", result.TotalNamespaces),
					zap.Int("current", result.Current),
					zap.Int("stale", result.Stale),
					zap.Int("unknown", result.Unknown),
					zap.Duration("interval", currentCfg.interval),
					zap.Duration("timeout", currentCfg.timeout),
					zap.Int("batch_size", currentCfg.batch),
					zap.String("config_source", currentCfg.source),
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

func tenantAssetDriftMetricsMap(snapshot domaininfrastructure.TenantAssetDriftMetricsSnapshot) map[string]int64 {
	return map[string]int64{
		"tenant_asset_drift_watch_ticks_total":           snapshot.WatchTicksTotal,
		"tenant_asset_drift_watch_failures_total":        snapshot.WatchFailuresTotal,
		"tenant_asset_drift_namespaces_current":          snapshot.WatchCurrentNamespaces,
		"tenant_asset_drift_namespaces_stale":            snapshot.WatchStaleNamespaces,
		"tenant_asset_drift_namespaces_unknown":          snapshot.WatchUnknownNamespaces,
		"tenant_asset_reconcile_requests_total":          snapshot.ReconcileRequestsTotal,
		"tenant_asset_reconcile_requests_success_total":  snapshot.ReconcileRequestsSuccess,
		"tenant_asset_reconcile_requests_failures_total": snapshot.ReconcileRequestsFailures,
		"tenant_asset_drift_watch_duration_count":        snapshot.WatchDurationCount,
		"tenant_asset_drift_watch_duration_total_ms":     snapshot.WatchDurationTotalMs,
		"tenant_asset_drift_watch_duration_max_ms":       snapshot.WatchDurationMaxMs,
	}
}
