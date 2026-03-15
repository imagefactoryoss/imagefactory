package appsignals

import (
	"context"
	"time"
)

type HTTPWindowSnapshot struct {
	RequestCount     int64
	ServerErrorCount int64
	ClientErrorCount int64
	TotalLatencyMs   int64
	AverageLatencyMs int64
	MaxLatencyMs     int64
	WindowStartedAt  time.Time
	WindowEndedAt    time.Time
}

type HTTPWindowRecord struct {
	RequestCount     int64     `db:"request_count"`
	ServerErrorCount int64     `db:"server_error_count"`
	ClientErrorCount int64     `db:"client_error_count"`
	TotalLatencyMs   int64     `db:"total_latency_ms"`
	AverageLatencyMs int64     `db:"average_latency_ms"`
	MaxLatencyMs     int64     `db:"max_latency_ms"`
	WindowStartedAt  time.Time `db:"window_started_at"`
	WindowEndedAt    time.Time `db:"window_ended_at"`
	CreatedAt        time.Time `db:"created_at"`
}

type Repository interface {
	StoreHTTPWindow(ctx context.Context, snapshot HTTPWindowSnapshot, retentionDays int) error
	ListRecentHTTPWindows(ctx context.Context, limit int) ([]HTTPWindowRecord, error)
}
