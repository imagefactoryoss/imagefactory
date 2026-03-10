package worker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/srikarm/image-factory/internal/domain/notification"
)

// EmailWorkerPool manages a pool of EmailWorker instances
// Responsibilities:
// - Create and manage multiple workers
// - Start/stop workers with graceful shutdown
// - Aggregate metrics from all workers
// - Coordinate worker lifecycle
type EmailWorkerPool struct {
	queue   notification.EmailQueueBackend
	sender  EmailSender
	config  WorkerConfig
	metrics *WorkerMetrics

	workers []*EmailWorker
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.RWMutex
	running bool
}

// NewEmailWorkerPool creates a new worker pool
func NewEmailWorkerPool(
	queue notification.EmailQueueBackend,
	sender EmailSender,
	config WorkerConfig,
) *EmailWorkerPool {
	metrics := NewWorkerMetrics(config.WorkerCount)

	return &EmailWorkerPool{
		queue:   queue,
		sender:  sender,
		config:  config,
		metrics: metrics,
		workers: make([]*EmailWorker, 0, config.WorkerCount),
	}
}

// Start starts the worker pool
// Creates worker goroutines and begins processing
// Returns error if pool is already running
func (p *EmailWorkerPool) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return errors.New("worker pool already running")
	}

	// Create cancellable context for workers
	workerCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	// Create workers
	p.workers = make([]*EmailWorker, p.config.WorkerCount)
	for i := 0; i < p.config.WorkerCount; i++ {
		p.workers[i] = NewEmailWorker(i, p.queue, p.sender, p.metrics, p.config)
	}

	// Start each worker in its own goroutine
	for i, worker := range p.workers {
		p.wg.Add(1)
		go func(w *EmailWorker, id int) {
			defer p.wg.Done()
			w.Run(workerCtx)
		}(worker, i)
	}

	// Start health check goroutine
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.runHealthChecks(workerCtx)
	}()

	p.running = true
	return nil
}

// Stop gracefully shuts down the worker pool
// Waits for workers to finish current tasks (up to ShutdownTimeout)
// Returns error if pool is not running
func (p *EmailWorkerPool) Stop() error {
	p.mu.Lock()

	if !p.running {
		p.mu.Unlock()
		return errors.New("worker pool not running")
	}

	// Cancel worker contexts (signals shutdown)
	if p.cancel != nil {
		p.cancel()
	}

	p.running = false
	p.mu.Unlock()

	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		// All workers stopped gracefully
		return nil
	case <-timeAfter(p.config.ShutdownTimeout):
		// Timeout reached - workers forced to stop
		return nil
	}
}

// GetMetrics returns the shared metrics for all workers
func (p *EmailWorkerPool) GetMetrics() *WorkerMetrics {
	return p.metrics
}

// GetConfig returns the pool configuration
func (p *EmailWorkerPool) GetConfig() WorkerConfig {
	return p.config
}

// IsRunning returns whether the pool is currently running
func (p *EmailWorkerPool) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// timeAfter is a helper to make time.After mockable in tests
// In production, this just calls time.After
var timeAfter = func(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// runHealthChecks performs periodic health checks while the pool is running
// Updates metrics with queue depth and last health check timestamp
func (p *EmailWorkerPool) runHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(p.config.HealthCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Pool is shutting down, exit health check loop
			return

		case <-ticker.C:
			// Perform health check
			p.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck executes a single health check
// Updates queue depth and last health check timestamp in metrics
func (p *EmailWorkerPool) performHealthCheck(ctx context.Context) {
	// Get queue depth from backend
	depth, err := p.queue.GetQueueDepth(ctx)
	if err != nil {
		// Log error (TODO: add proper logging)
		// For now, just skip this health check
		return
	}

	// Update metrics
	p.metrics.UpdateQueueDepth(depth)
	p.metrics.UpdateLastHealthCheck()
}
