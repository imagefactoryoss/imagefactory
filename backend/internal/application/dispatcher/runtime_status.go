package dispatcher

import (
	"context"
	"time"
)

const (
	DispatcherModeEmbedded = "embedded"
	DispatcherModeExternal = "external"
)

// RuntimeStatus captures liveness and metrics for a dispatcher instance.
type RuntimeStatus struct {
	InstanceID    string
	Mode          string
	Running       bool
	LastHeartbeat time.Time
	Metrics       DispatcherMetricsSnapshot
	UpdatedAt     time.Time
}

// RuntimeStatusStore persists dispatcher runtime snapshots.
type RuntimeStatusStore interface {
	UpsertRuntimeStatus(ctx context.Context, status RuntimeStatus) error
	GetLatestRuntimeStatus(ctx context.Context, mode string) (*RuntimeStatus, error)
}
