package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/srikarm/image-factory/internal/domain/notification"
)

// PostgresEmailQueue implements EmailQueueBackend using PostgreSQL
// This implementation follows SOLID principles:
// - Single Responsibility: Only handles email queue operations
// - Dependency Inversion: Depends on sqlx.DB interface
// - No N+1 queries: All operations use single SQL statements
type PostgresEmailQueue struct {
	db *sqlx.DB
}

// NewPostgresEmailQueue creates a new PostgreSQL email queue backend
func NewPostgresEmailQueue(db *sqlx.DB) notification.EmailQueueBackend {
	return &PostgresEmailQueue{
		db: db,
	}
}

// Enqueue adds a new email task to the queue
// Single INSERT query - no N+1 issues
// Uses prepared statement to prevent SQL injection
func (q *PostgresEmailQueue) Enqueue(ctx context.Context, task *notification.EmailTask) error {
	// Validation
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}
	if task.TenantID == uuid.Nil {
		return fmt.Errorf("tenant_id is required")
	}
	if task.ToAddress == "" {
		return fmt.Errorf("to_address is required")
	}
	if task.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	// Validate priority if set (allow 0 to use default)
	if task.Priority != 0 && (task.Priority < 1 || task.Priority > 10) {
		return fmt.Errorf("priority must be between 1 and 10")
	}

	// Set defaults
	now := time.Now()
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	if task.Priority == 0 {
		task.Priority = 5 // Default priority
	}
	if task.MaxAttempts == 0 {
		task.MaxAttempts = 3 // Default max attempts
	}
	if task.ScheduledAt.IsZero() {
		task.ScheduledAt = now
	}
	if task.Status == "" {
		task.Status = notification.EmailStatusPending
	}
	task.CreatedAt = now
	task.UpdatedAt = now

	// Serialize template data to JSONB
	var templateDataJSON interface{}
	if task.TemplateData != nil {
		data, err := json.Marshal(task.TemplateData)
		if err != nil {
			return fmt.Errorf("failed to marshal template_data: %w", err)
		}
		templateDataJSON = data
	} else {
		templateDataJSON = nil // Explicitly set to NULL for database
	}

	// Single INSERT query - no N+1
	query := `
		INSERT INTO email_queue (
			id, tenant_id, to_address, cc_address, subject, body_html, body_text,
			template_name, template_data, status, priority,
			attempts, max_attempts, scheduled_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14,
			$15, $16
		)
	`

	_, err := q.db.ExecContext(ctx, query,
		task.ID,
		task.TenantID,
		task.ToAddress,
		task.CCAddress,
		task.Subject,
		task.BodyHTML,
		task.BodyText,
		task.TemplateName,
		templateDataJSON,
		task.Status,
		task.Priority,
		task.Attempts,
		task.MaxAttempts,
		task.ScheduledAt,
		task.CreatedAt,
		task.UpdatedAt,
	)

	if err != nil {
		// Handle PostgreSQL-specific errors
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505": // unique_violation
				return fmt.Errorf("email task with ID %s already exists: %w", task.ID, err)
			case "23503": // foreign_key_violation
				return fmt.Errorf("tenant_id %s does not exist: %w", task.TenantID, err)
			case "23514": // check_violation
				return fmt.Errorf("invalid data (check constraint violation): %w", err)
			}
		}
		return fmt.Errorf("failed to enqueue email: %w", err)
	}

	return nil
}

// Dequeue retrieves the next pending email task from the queue
// Uses FOR UPDATE SKIP LOCKED for atomic dequeue
// Single query - no N+1 issues
func (q *PostgresEmailQueue) Dequeue(ctx context.Context) (*notification.EmailTask, error) {
	fmt.Println("DEBUG: Dequeue called")

	query := `
		UPDATE email_queue
		SET status = $1,
			processing_started_at = NOW(),
			attempts = attempts + 1,
			updated_at = NOW()
		WHERE id = (
			SELECT id FROM email_queue
			WHERE status = $2
			  AND scheduled_at <= NOW()
			ORDER BY priority ASC, created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, tenant_id, to_address, cc_address, subject, body_html, body_text,
		          template_name, template_data, status, priority,
		          attempts, max_attempts, scheduled_at,
		          processing_started_at, processed_at, error_message,
		          created_at, updated_at
	`

	fmt.Printf("DEBUG: Executing dequeue query with status: processing=%s, pending=%s\n",
		notification.EmailStatusProcessing, notification.EmailStatusPending)

	var task notification.EmailTask
	var templateDataJSON []byte
	var templateName sql.NullString
	var processingStartedAt, processedAt sql.NullTime
	var errorMessage sql.NullString

	err := q.db.QueryRowContext(ctx, query,
		notification.EmailStatusProcessing,
		notification.EmailStatusPending,
	).Scan(
		&task.ID,
		&task.TenantID,
		&task.ToAddress,
		&task.CCAddress,
		&task.Subject,
		&task.BodyHTML,
		&task.BodyText,
		&templateName,
		&templateDataJSON,
		&task.Status,
		&task.Priority,
		&task.Attempts,
		&task.MaxAttempts,
		&task.ScheduledAt,
		&processingStartedAt,
		&processedAt,
		&errorMessage,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// No pending emails - return nil task
		fmt.Println("DEBUG: No pending emails found")
		return nil, nil
	}
	if err != nil {
		fmt.Printf("DEBUG: Dequeue error: %v\n", err)
		return nil, fmt.Errorf("failed to dequeue email: %w", err)
	}

	fmt.Printf("DEBUG: Dequeued email %s to %s\n", task.ID, task.ToAddress)

	// Convert nullable template name
	if templateName.Valid {
		task.TemplateName = templateName.String
	}

	// Deserialize template data
	if len(templateDataJSON) > 0 {
		if err := json.Unmarshal(templateDataJSON, &task.TemplateData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal template_data: %w", err)
		}
	}

	// Convert nullable timestamps
	if processingStartedAt.Valid {
		task.ProcessingStartedAt = &processingStartedAt.Time
	}
	if processedAt.Valid {
		task.ProcessedAt = &processedAt.Time
	}

	// Convert nullable error message
	if errorMessage.Valid {
		task.ErrorMessage = errorMessage.String
	}

	return &task, nil
}

// MarkProcessed marks an email task as sent (success) or failed
// Single UPDATE query - no N+1 issues
func (q *PostgresEmailQueue) MarkProcessed(ctx context.Context, taskID uuid.UUID, status notification.EmailStatus, errorMsg string) error {
	if status != notification.EmailStatusSent && status != notification.EmailStatusFailed {
		return fmt.Errorf("status must be 'sent' or 'failed'")
	}

	// Handle NULL for empty error message
	var errorMsgParam interface{}
	if errorMsg == "" {
		errorMsgParam = nil
	} else {
		errorMsgParam = errorMsg
	}

	query := `
		UPDATE email_queue
		SET status = $1,
			processed_at = NOW(),
			error_message = $2,
			updated_at = NOW()
		WHERE id = $3
	`

	result, err := q.db.ExecContext(ctx, query, status, errorMsgParam, taskID)
	if err != nil {
		return fmt.Errorf("failed to mark email as processed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("email task %s not found", taskID)
	}

	return nil
}

// Retry schedules an email task for retry with exponential backoff
// Single UPDATE query - no N+1 issues
func (q *PostgresEmailQueue) Retry(ctx context.Context, taskID uuid.UUID, nextAttemptAt time.Time) error {
	query := `
		UPDATE email_queue
		SET status = $1,
			scheduled_at = $2,
			updated_at = NOW()
		WHERE id = $3
	`

	result, err := q.db.ExecContext(ctx, query,
		notification.EmailStatusPending,
		nextAttemptAt,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("failed to retry email: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("email task %s not found", taskID)
	}

	return nil
}

// Cancel marks an email task as cancelled
// Single UPDATE query - no N+1 issues
func (q *PostgresEmailQueue) Cancel(ctx context.Context, taskID uuid.UUID) error {
	query := `
		UPDATE email_queue
		SET status = $1,
			updated_at = NOW()
		WHERE id = $2
	`

	result, err := q.db.ExecContext(ctx, query,
		notification.EmailStatusCancelled,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("failed to cancel email: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("email task %s not found", taskID)
	}

	return nil
}

// GetQueueStats returns current queue statistics grouped by status
// Single query using GROUP BY - no N+1 issues
func (q *PostgresEmailQueue) GetQueueStats(ctx context.Context) (*notification.QueueStats, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM email_queue
		GROUP BY status
	`

	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}
	defer rows.Close()

	stats := &notification.QueueStats{}
	var total int64

	for rows.Next() {
		var status notification.EmailStatus
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan queue stats: %w", err)
		}

		total += count
		switch status {
		case notification.EmailStatusPending:
			stats.Pending = count
		case notification.EmailStatusProcessing:
			stats.Processing = count
		case notification.EmailStatusSent:
			stats.Sent = count
		case notification.EmailStatusFailed:
			stats.Failed = count
		case notification.EmailStatusCancelled:
			stats.Cancelled = count
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating queue stats: %w", err)
	}

	stats.Total = total
	return stats, nil
}

// GetQueueDepth returns the count of pending emails in the queue
// Uses partial index for high performance - single query, no N+1
func (q *PostgresEmailQueue) GetQueueDepth(ctx context.Context) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM email_queue
		WHERE status = $1
	`

	var count int64
	err := q.db.QueryRowContext(ctx, query, notification.EmailStatusPending).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue depth: %w", err)
	}

	return count, nil
}
