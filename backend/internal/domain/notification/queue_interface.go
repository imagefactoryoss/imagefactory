package notification

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EmailStatus represents the current state of an email in the queue
type EmailStatus string

const (
	EmailStatusPending    EmailStatus = "pending"
	EmailStatusProcessing EmailStatus = "processing"
	EmailStatusSent       EmailStatus = "sent"
	EmailStatusFailed     EmailStatus = "failed"
	EmailStatusCancelled  EmailStatus = "cancelled"
)

// EmailTask represents an email task in the queue
type EmailTask struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	ToAddress string
	CCAddress string
	Subject   string
	BodyHTML  string
	BodyText  string

	// Template-based email fields
	TemplateName string
	TemplateData map[string]interface{}

	// Queue management
	Status      EmailStatus
	Priority    int // 1=highest, 10=lowest
	Attempts    int
	MaxAttempts int
	ScheduledAt time.Time

	// Processing tracking
	ProcessingStartedAt *time.Time
	ProcessedAt         *time.Time

	// Error tracking
	ErrorMessage string

	// Audit fields
	CreatedAt time.Time
	UpdatedAt time.Time
}

// QueueStats provides statistics about the email queue
type QueueStats struct {
	Pending    int64
	Processing int64
	Sent       int64
	Failed     int64
	Cancelled  int64
	Total      int64
}

// EmailQueueBackend defines the interface for email queue operations
// This follows Interface Segregation Principle - focused on queue operations only
// Implementations can be PostgreSQL, Redis, NATS, etc.
type EmailQueueBackend interface {
	// Enqueue adds a new email task to the queue
	// Returns error if task cannot be enqueued
	Enqueue(ctx context.Context, task *EmailTask) error

	// Dequeue retrieves the next pending email task from the queue
	// Uses atomic operations (e.g., FOR UPDATE SKIP LOCKED in PostgreSQL)
	// Returns nil task if queue is empty
	// Returns error if dequeue operation fails
	Dequeue(ctx context.Context) (*EmailTask, error)

	// MarkProcessed marks an email task as sent (success) or failed
	// Updates processed_at timestamp and final status
	// Returns error if update fails
	MarkProcessed(ctx context.Context, taskID uuid.UUID, status EmailStatus, errorMsg string) error

	// Retry schedules an email task for retry with exponential backoff
	// Updates attempts counter and scheduled_at for next attempt
	// Returns error if update fails
	Retry(ctx context.Context, taskID uuid.UUID, nextAttemptAt time.Time) error

	// Cancel marks an email task as cancelled
	// Used for administrative cancellation of pending/failed emails
	// Returns error if update fails
	Cancel(ctx context.Context, taskID uuid.UUID) error

	// GetQueueStats returns current queue statistics grouped by status
	// Single query using GROUP BY - no N+1 queries
	// Returns error if stats cannot be retrieved
	GetQueueStats(ctx context.Context) (*QueueStats, error)

	// GetQueueDepth returns the count of pending emails in the queue
	// Uses partial index for high performance
	// Returns error if count fails
	GetQueueDepth(ctx context.Context) (int64, error)
}
