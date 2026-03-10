package worker

import (
	"testing"
	"time"
)

func TestDefaultWorkerConfig(t *testing.T) {
	cfg := DefaultWorkerConfig()
	if cfg.WorkerCount != 5 || cfg.PollInterval != time.Second || cfg.MaxRetries != 3 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestWorkerConfigValidate(t *testing.T) {
	valid := DefaultWorkerConfig()
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid default config, got %v", err)
	}

	cases := []WorkerConfig{
		{WorkerCount: 0},
		{WorkerCount: 101},
		{WorkerCount: 1, PollInterval: 50 * time.Millisecond, RetryBaseDelay: time.Second, RetryMaxDelay: time.Second, ShutdownTimeout: time.Second, HealthCheckPeriod: time.Second},
		{WorkerCount: 1, PollInterval: 2 * time.Minute, RetryBaseDelay: time.Second, RetryMaxDelay: time.Second, ShutdownTimeout: time.Second, HealthCheckPeriod: time.Second},
		{WorkerCount: 1, PollInterval: time.Second, MaxRetries: -1, RetryBaseDelay: time.Second, RetryMaxDelay: time.Second, ShutdownTimeout: time.Second, HealthCheckPeriod: time.Second},
		{WorkerCount: 1, PollInterval: time.Second, MaxRetries: 11, RetryBaseDelay: time.Second, RetryMaxDelay: time.Second, ShutdownTimeout: time.Second, HealthCheckPeriod: time.Second},
		{WorkerCount: 1, PollInterval: time.Second, MaxRetries: 1, RetryBaseDelay: 50 * time.Millisecond, RetryMaxDelay: time.Second, ShutdownTimeout: time.Second, HealthCheckPeriod: time.Second},
		{WorkerCount: 1, PollInterval: time.Second, MaxRetries: 1, RetryBaseDelay: time.Second, RetryMaxDelay: 500 * time.Millisecond, ShutdownTimeout: time.Second, HealthCheckPeriod: time.Second},
		{WorkerCount: 1, PollInterval: time.Second, MaxRetries: 1, RetryBaseDelay: time.Second, RetryMaxDelay: time.Second, ShutdownTimeout: 500 * time.Millisecond, HealthCheckPeriod: time.Second},
		{WorkerCount: 1, PollInterval: time.Second, MaxRetries: 1, RetryBaseDelay: time.Second, RetryMaxDelay: time.Second, ShutdownTimeout: time.Second, HealthCheckPeriod: 500 * time.Millisecond},
	}
	for i, cfg := range cases {
		if err := cfg.Validate(); err == nil {
			t.Fatalf("case %d expected validation error, got nil", i)
		}
	}
}
