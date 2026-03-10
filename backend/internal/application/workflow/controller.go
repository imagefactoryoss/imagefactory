package workflow

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// ControllerConfig defines workflow controller runtime settings.
type ControllerConfig struct {
	PollInterval    time.Duration
	MaxStepsPerTick int
}

// ControllerHooks allows runtime callbacks for status tracking.
type ControllerHooks struct {
	OnStart func()
	OnTick  func()
	OnStop  func()
}

// Controller manages workflow orchestrator lifecycle.
type Controller struct {
	orchestrator *Orchestrator
	config       ControllerConfig
	hooks        ControllerHooks
	logger       *zap.Logger
	enabled      bool
	running      atomic.Bool
	cancel       context.CancelFunc
	mu           sync.Mutex
}

// NewController creates a workflow controller.
func NewController(orchestrator *Orchestrator, config ControllerConfig, hooks ControllerHooks, logger *zap.Logger) *Controller {
	if config.PollInterval <= 0 {
		config.PollInterval = 3 * time.Second
	}
	if config.MaxStepsPerTick <= 0 {
		config.MaxStepsPerTick = 1
	}
	return &Controller{
		orchestrator: orchestrator,
		config:       config,
		hooks:        hooks,
		logger:       logger,
		enabled:      orchestrator != nil,
	}
}

// Start begins orchestrator processing. Returns true if started.
func (c *Controller) Start(ctx context.Context) bool {
	if c.orchestrator == nil {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running.Load() {
		return false
	}

	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.running.Store(true)
	if c.hooks.OnStart != nil {
		c.hooks.OnStart()
	}

	go func() {
		defer func() {
			c.running.Store(false)
			if c.hooks.OnStop != nil {
				c.hooks.OnStop()
			}
		}()

		ticker := time.NewTicker(c.config.PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-runCtx.Done():
				return
			default:
			}

			for i := 0; i < c.config.MaxStepsPerTick; i++ {
				ran, err := c.orchestrator.RunOnce(runCtx)
				if err != nil {
					c.logger.Error("Workflow orchestrator step failed", zap.Error(err))
					break
				}
				if !ran {
					break
				}
			}

			if c.hooks.OnTick != nil {
				c.hooks.OnTick()
			}

			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	return true
}

// Stop halts orchestrator processing. Returns true if stopped.
func (c *Controller) Stop() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running.Load() {
		return false
	}
	if c.cancel != nil {
		c.cancel()
	}
	c.running.Store(false)
	return true
}

// Status reports whether the orchestrator is running.
func (c *Controller) Status() bool {
	return c.running.Load()
}

// Enabled reports whether the controller has an orchestrator configured.
func (c *Controller) Enabled() bool {
	return c.enabled
}
