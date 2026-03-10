package dispatcher

import (
	"context"
	"sync"
	"sync/atomic"
)

// Controller manages dispatcher lifecycle.
type Controller struct {
	dispatcher *QueuedBuildDispatcher
	running    atomic.Bool
	cancel     context.CancelFunc
	mu         sync.Mutex
}

// NewController creates a dispatcher controller.
func NewController(dispatcher *QueuedBuildDispatcher) *Controller {
	return &Controller{dispatcher: dispatcher}
}

// Start begins dispatcher processing. Returns true if started.
func (c *Controller) Start(ctx context.Context) bool {
	if c.dispatcher == nil {
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
	go func() {
		c.dispatcher.Run(runCtx)
		c.running.Store(false)
	}()

	return true
}

// Stop halts dispatcher processing. Returns true if stopped.
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

// Status reports whether the dispatcher is running.
func (c *Controller) Status() bool {
	return c.running.Load()
}

// DispatcherMetrics returns dispatcher metrics snapshot.
func (c *Controller) DispatcherMetrics() DispatcherMetricsSnapshot {
	if c.dispatcher == nil {
		return DispatcherMetricsSnapshot{}
	}
	return c.dispatcher.DispatcherMetrics()
}
