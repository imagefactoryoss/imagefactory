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

type ProviderReadinessWatcherConfig struct {
	DefaultEnabled         bool
	DefaultIntervalSeconds int
	DefaultTimeoutSeconds  int
	DefaultBatchSize       int
	OnTick                 func(ctx context.Context, result *domaininfrastructure.ProviderReadinessWatchTickResult, err error, observedAt time.Time)
}

type providerReadinessWatcherRuntimeConfig struct {
	enabled  bool
	interval time.Duration
	timeout  time.Duration
	batch    int
	source   string
}

func StartProviderReadinessWatcher(
	logger *zap.Logger,
	processHealthStore *runtimehealth.Store,
	systemConfigService interface {
		GetConfigByTypeAndKey(ctx context.Context, tenantID *uuid.UUID, configType domainsystemconfig.ConfigType, configKey string) (*domainsystemconfig.SystemConfig, error)
	},
	infrastructureService interface {
		RunProviderReadinessWatchTick(ctx context.Context, pageSize int) (*domaininfrastructure.ProviderReadinessWatchTickResult, error)
	},
	cfg ProviderReadinessWatcherConfig,
) {
	if processHealthStore == nil {
		return
	}

	loadConfig := func(ctx context.Context) providerReadinessWatcherRuntimeConfig {
		out := providerReadinessWatcherRuntimeConfig{
			enabled:  cfg.DefaultEnabled,
			interval: time.Duration(cfg.DefaultIntervalSeconds) * time.Second,
			timeout:  time.Duration(cfg.DefaultTimeoutSeconds) * time.Second,
			batch:    cfg.DefaultBatchSize,
			source:   "env",
		}
		if out.interval < 30*time.Second {
			out.interval = 180 * time.Second
		}
		if out.timeout < 10*time.Second || out.timeout >= out.interval {
			out.timeout = out.interval - time.Second
			if out.timeout > 90*time.Second {
				out.timeout = 90 * time.Second
			}
			if out.timeout < 10*time.Second {
				out.timeout = 10 * time.Second
			}
		}
		if out.batch <= 0 {
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
		if runtimeServicesConfig.ProviderReadinessWatcherEnabled != nil {
			out.enabled = *runtimeServicesConfig.ProviderReadinessWatcherEnabled
			out.source = "system_config"
		}
		if runtimeServicesConfig.ProviderReadinessWatcherIntervalSeconds >= 30 {
			out.interval = time.Duration(runtimeServicesConfig.ProviderReadinessWatcherIntervalSeconds) * time.Second
			out.source = "system_config"
		}
		if runtimeServicesConfig.ProviderReadinessWatcherTimeoutSeconds >= 10 {
			timeout := time.Duration(runtimeServicesConfig.ProviderReadinessWatcherTimeoutSeconds) * time.Second
			if timeout < out.interval {
				out.timeout = timeout
				out.source = "system_config"
			}
		}
		if runtimeServicesConfig.ProviderReadinessWatcherBatchSize >= 1 && runtimeServicesConfig.ProviderReadinessWatcherBatchSize <= 1000 {
			out.batch = runtimeServicesConfig.ProviderReadinessWatcherBatchSize
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
	processHealthStore.Upsert("provider_readiness_watcher", runtimehealth.ProcessStatus{
		Enabled:      initialCfg.enabled,
		Running:      initialCfg.enabled,
		LastActivity: time.Now().UTC(),
		Message:      "provider readiness watcher initialized",
	})
	if logger != nil {
		logger.Info("Background process starting",
			zap.String("component", "provider_readiness_watcher"),
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
				processHealthStore.Upsert("provider_readiness_watcher", runtimehealth.ProcessStatus{
					Enabled:      false,
					Running:      false,
					LastActivity: time.Now().UTC(),
					Message:      "provider readiness watcher disabled via runtime_services config",
				})
				return
			}
			if currentCfg.interval != currentInterval {
				ticker.Reset(currentCfg.interval)
				currentInterval = currentCfg.interval
				if logger != nil {
					logger.Info("Provider readiness watcher interval updated",
						zap.Duration("interval", currentCfg.interval),
						zap.String("config_source", currentCfg.source),
					)
				}
			}

			if infrastructureService == nil {
				processHealthStore.Upsert("provider_readiness_watcher", runtimehealth.ProcessStatus{
					Enabled:      true,
					Running:      false,
					LastActivity: time.Now().UTC(),
					Message:      "infrastructure service not configured",
				})
				return
			}

			tickCtx, cancel := context.WithTimeout(context.Background(), currentCfg.timeout)
			defer cancel()

			result, err := infrastructureService.RunProviderReadinessWatchTick(tickCtx, currentCfg.batch)
			observedAt := time.Now().UTC()
			if err != nil {
				if logger != nil {
					logger.Warn("Provider readiness watcher tick failed", zap.Error(err))
				}
				if cfg.OnTick != nil {
					cfg.OnTick(context.Background(), nil, err, observedAt)
				}
				processHealthStore.Upsert("provider_readiness_watcher", runtimehealth.ProcessStatus{
					Enabled:      true,
					Running:      true,
					LastActivity: observedAt,
					Message:      "watch tick failed",
				})
				return
			}
			if cfg.OnTick != nil {
				cfg.OnTick(context.Background(), result, nil, observedAt)
			}
			processHealthStore.Upsert("provider_readiness_watcher", runtimehealth.ProcessStatus{
				Enabled:      true,
				Running:      true,
				LastActivity: observedAt,
				Message:      fmt.Sprintf("tick completed attempted=%d refresh_succeeded=%d failed=%d skipped=%d ready=%d not_ready=%d", result.Attempted, result.Succeeded, result.Failed, result.Skipped, result.Ready, result.NotReady),
			})
			if logger != nil && (result.Attempted > 0 || result.Failed > 0) {
				logger.Info("Provider readiness watcher tick completed",
					zap.Int("total_providers", result.TotalProviders),
					zap.Int("attempted", result.Attempted),
					zap.Int("refresh_succeeded", result.Succeeded),
					zap.Int("failed", result.Failed),
					zap.Int("skipped", result.Skipped),
					zap.Int("ready", result.Ready),
					zap.Int("not_ready", result.NotReady),
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
