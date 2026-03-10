package worker

import (
	"fmt"
	"time"
)

// WorkerConfig holds configuration for email worker pool
type WorkerConfig struct {
	// WorkerCount is the number of concurrent workers to spawn
	WorkerCount int `json:"worker_count"`

	// PollInterval is how often to check for new emails when queue is empty
	PollInterval time.Duration `json:"poll_interval"`

	// MaxRetries is the maximum number of retry attempts for failed emails
	MaxRetries int `json:"max_retries"`

	// RetryBaseDelay is the base delay for exponential backoff (e.g., 1 second)
	RetryBaseDelay time.Duration `json:"retry_base_delay"`

	// RetryMaxDelay is the maximum delay for exponential backoff (e.g., 60 seconds)
	RetryMaxDelay time.Duration `json:"retry_max_delay"`

	// ShutdownTimeout is how long to wait for workers to finish during graceful shutdown
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`

	// HealthCheckPeriod is how often to perform health checks
	HealthCheckPeriod time.Duration `json:"health_check_period"`
}

// DefaultWorkerConfig returns a WorkerConfig with sensible defaults
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		WorkerCount:       5,
		PollInterval:      1 * time.Second,
		MaxRetries:        3,
		RetryBaseDelay:    1 * time.Second,
		RetryMaxDelay:     60 * time.Second,
		ShutdownTimeout:   30 * time.Second,
		HealthCheckPeriod: 10 * time.Second,
	}
}

// Validate checks if the configuration is valid
func (c WorkerConfig) Validate() error {
	if c.WorkerCount < 1 {
		return fmt.Errorf("worker_count must be at least 1, got %d", c.WorkerCount)
	}
	if c.WorkerCount > 100 {
		return fmt.Errorf("worker_count must not exceed 100, got %d", c.WorkerCount)
	}
	if c.PollInterval < 100*time.Millisecond {
		return fmt.Errorf("poll_interval must be at least 100ms, got %v", c.PollInterval)
	}
	if c.PollInterval > 1*time.Minute {
		return fmt.Errorf("poll_interval must not exceed 1 minute, got %v", c.PollInterval)
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative, got %d", c.MaxRetries)
	}
	if c.MaxRetries > 10 {
		return fmt.Errorf("max_retries must not exceed 10, got %d", c.MaxRetries)
	}
	if c.RetryBaseDelay < 100*time.Millisecond {
		return fmt.Errorf("retry_base_delay must be at least 100ms, got %v", c.RetryBaseDelay)
	}
	if c.RetryMaxDelay < c.RetryBaseDelay {
		return fmt.Errorf("retry_max_delay must be >= retry_base_delay, got %v < %v", c.RetryMaxDelay, c.RetryBaseDelay)
	}
	if c.ShutdownTimeout < 1*time.Second {
		return fmt.Errorf("shutdown_timeout must be at least 1 second, got %v", c.ShutdownTimeout)
	}
	if c.HealthCheckPeriod < 1*time.Second {
		return fmt.Errorf("health_check_period must be at least 1 second, got %v", c.HealthCheckPeriod)
	}
	return nil
}
