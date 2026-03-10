package infrastructure

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type TektonInstallerRunner interface {
	RunNextTektonInstallerJob(ctx context.Context) (bool, error)
}

type TektonInstallerDispatcherConfig struct {
	PollInterval   time.Duration
	MaxJobsPerTick int
}

type TektonInstallerDispatcher struct {
	runner TektonInstallerRunner
	config TektonInstallerDispatcherConfig
	logger *zap.Logger
}

func NewTektonInstallerDispatcher(runner TektonInstallerRunner, logger *zap.Logger, config TektonInstallerDispatcherConfig) *TektonInstallerDispatcher {
	if config.PollInterval <= 0 {
		config.PollInterval = 5 * time.Second
	}
	if config.MaxJobsPerTick <= 0 {
		config.MaxJobsPerTick = 1
	}
	return &TektonInstallerDispatcher{
		runner: runner,
		config: config,
		logger: logger,
	}
}

func (d *TektonInstallerDispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.config.PollInterval)
	defer ticker.Stop()

	d.logger.Info("Tekton installer dispatcher started",
		zap.Duration("poll_interval", d.config.PollInterval),
		zap.Int("max_jobs_per_tick", d.config.MaxJobsPerTick),
	)

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("Tekton installer dispatcher stopped")
			return
		default:
		}

		_, _ = d.RunOnce(ctx)

		select {
		case <-ctx.Done():
			d.logger.Info("Tekton installer dispatcher stopped")
			return
		case <-ticker.C:
		}
	}
}

func (d *TektonInstallerDispatcher) RunOnce(ctx context.Context) (int, error) {
	processed := 0
	for processed < d.config.MaxJobsPerTick {
		handled, err := d.runner.RunNextTektonInstallerJob(ctx)
		if err != nil {
			d.logger.Error("Failed to process tekton installer job", zap.Error(err))
			return processed, err
		}
		if !handled {
			break
		}
		processed++
	}
	return processed, nil
}
