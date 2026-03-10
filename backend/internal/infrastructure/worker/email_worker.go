package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/srikarm/image-factory/internal/domain/notification"
)

// EmailSender defines the interface for sending emails
// This follows Interface Segregation Principle - focused on sending only
// Implementations can be SMTP, AWS SES, SendGrid, etc.
type EmailSender interface {
	// Send delivers an email task
	// Returns error if send fails (temporary or permanent)
	Send(ctx context.Context, task *notification.EmailTask) error
}

// EmailWorker processes email tasks from the queue
// Single Responsibility: Dequeue → Send → Mark Processed/Retry
type EmailWorker struct {
	id      int                            // Worker ID (0-based index)
	queue   notification.EmailQueueBackend // Queue backend
	sender  EmailSender                    // Email sender
	metrics *WorkerMetrics                 // Shared metrics
	config  WorkerConfig                   // Worker configuration
}

// NewEmailWorker creates a new email worker
func NewEmailWorker(
	id int,
	queue notification.EmailQueueBackend,
	sender EmailSender,
	metrics *WorkerMetrics,
	config WorkerConfig,
) *EmailWorker {
	return &EmailWorker{
		id:      id,
		queue:   queue,
		sender:  sender,
		metrics: metrics,
		config:  config,
	}
}

// Run starts the worker processing loop
// Runs until context is cancelled
// Gracefully exits on context cancellation
func (w *EmailWorker) Run(ctx context.Context) {
	fmt.Printf("Worker %d: Starting run loop\n", w.id)

	// Set initial status
	w.metrics.SetWorkerStatus(w.id, WorkerStatusIdle)
	defer w.metrics.SetWorkerStatus(w.id, WorkerStatusStopped)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Graceful shutdown - exit immediately
			fmt.Printf("Worker %d: Context cancelled, shutting down\n", w.id)
			return

		case <-ticker.C:
			// Poll queue for next task
			fmt.Printf("Worker %d: Polling for tasks\n", w.id)
			w.processNextTask(ctx)
		}
	}
}

// processNextTask dequeues and processes a single email task
func (w *EmailWorker) processNextTask(ctx context.Context) {
	// Dequeue next task
	task, err := w.queue.Dequeue(ctx)
	if err != nil {
		fmt.Printf("Worker %d: Error dequeuing task: %v\n", w.id, err)
		w.metrics.SetWorkerStatus(w.id, WorkerStatusError)
		return
	}

	// No task available - queue is empty
	if task == nil {
		w.metrics.SetWorkerStatus(w.id, WorkerStatusIdle)
		return
	}

	fmt.Printf("Worker %d: Dequeued email task %s to %s\n", w.id, task.ID, task.ToAddress)

	// Process the task
	w.metrics.SetWorkerStatus(w.id, WorkerStatusProcessing)
	w.processTask(ctx, task)
	w.metrics.SetWorkerStatus(w.id, WorkerStatusIdle)
}

// processTask sends an email and handles success/failure
func (w *EmailWorker) processTask(ctx context.Context, task *notification.EmailTask) {
	// Log processing attempt
	fmt.Printf("Worker %d: Processing email to %s (attempt %d/%d)\n", w.id, task.ToAddress, task.Attempts+1, task.MaxAttempts)

	// Attempt to send email
	err := w.sender.Send(ctx, task)

	if err == nil {
		// Success - mark as sent
		fmt.Printf("Worker %d: Successfully sent email to %s\n", w.id, task.ToAddress)
		w.handleSuccess(ctx, task)
		return
	}

	// Failure - check if we should retry or mark as permanently failed
	fmt.Printf("Worker %d: Failed to send email to %s: %v\n", w.id, task.ToAddress, err)
	w.handleFailure(ctx, task, err)
}

// handleSuccess marks email as sent and updates metrics
func (w *EmailWorker) handleSuccess(ctx context.Context, task *notification.EmailTask) {
	err := w.queue.MarkProcessed(ctx, task.ID, notification.EmailStatusSent, "")
	if err != nil {
		fmt.Printf("Worker %d: Error marking email as sent: %v\n", w.id, err)
		return
	}

	// Update metrics
	w.metrics.IncrementEmailsSent()
}

// handleFailure decides whether to retry or permanently fail an email
func (w *EmailWorker) handleFailure(ctx context.Context, task *notification.EmailTask, sendErr error) {
	// Determine max attempts (use task's MaxAttempts if set, otherwise use config's MaxRetries)
	maxAttempts := task.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = w.config.MaxRetries
	}

	// Check if max retries exceeded
	if task.Attempts >= maxAttempts {
		// Permanent failure - no more retries
		errorMsg := fmt.Sprintf("max retries exceeded: %v", sendErr)
		fmt.Printf("Worker %d: Email permanently failed: %s\n", w.id, errorMsg)
		err := w.queue.MarkProcessed(ctx, task.ID, notification.EmailStatusFailed, errorMsg)
		if err != nil {
			fmt.Printf("Worker %d: Error marking email as failed: %v\n", w.id, err)
			return
		}

		// Update metrics
		w.metrics.IncrementEmailsFailed()
		return
	}

	// Schedule retry with exponential backoff
	nextAttemptAt := CalculateNextAttempt(task.Attempts, w.config.RetryBaseDelay, w.config.RetryMaxDelay)
	fmt.Printf("Worker %d: Scheduling retry for email, next attempt at %s\n", w.id, nextAttemptAt.Format("15:04:05"))
	err := w.queue.Retry(ctx, task.ID, nextAttemptAt)
	if err != nil {
		fmt.Printf("Worker %d: Error scheduling retry: %v\n", w.id, err)
		return
	}

	// Update metrics
	w.metrics.IncrementEmailsRetried()
}
