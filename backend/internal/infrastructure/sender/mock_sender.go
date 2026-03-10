package sender

import (
	"context"
	"fmt"
	"sync"

	"github.com/srikarm/image-factory/internal/domain/notification"
)

// MockEmailSender is a mock implementation of EmailSender for testing
type MockEmailSender struct {
	mu         sync.RWMutex
	sentEmails []*notification.EmailTask
	sendError  error
	sendDelay  bool
}

// NewMockEmailSender creates a new mock email sender
func NewMockEmailSender() *MockEmailSender {
	return &MockEmailSender{
		sentEmails: make([]*notification.EmailTask, 0),
	}
}

// Send implements the EmailSender interface
func (m *MockEmailSender) Send(ctx context.Context, task *notification.EmailTask) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	// Return configured error if set
	if m.sendError != nil {
		return m.sendError
	}

	// Basic validation
	if task.ToAddress == "" {
		return fmt.Errorf("recipient address is required")
	}
	if task.BodyHTML == "" && task.BodyText == "" {
		return fmt.Errorf("email body is required")
	}

	// Store the sent email
	// Make a copy to avoid issues with pointers
	emailCopy := *task
	m.sentEmails = append(m.sentEmails, &emailCopy)

	return nil
}

// SetSendError configures the sender to return an error on Send()
func (m *MockEmailSender) SetSendError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendError = err
}

// GetSentEmails returns all emails that were sent
func (m *MockEmailSender) GetSentEmails() []*notification.EmailTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modifications
	emails := make([]*notification.EmailTask, len(m.sentEmails))
	copy(emails, m.sentEmails)
	return emails
}

// GetSentEmailCount returns the number of emails sent
func (m *MockEmailSender) GetSentEmailCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sentEmails)
}

// GetLastSentEmail returns the last email that was sent, or nil if none
func (m *MockEmailSender) GetLastSentEmail() *notification.EmailTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.sentEmails) == 0 {
		return nil
	}
	return m.sentEmails[len(m.sentEmails)-1]
}

// Reset clears all sent emails and errors
func (m *MockEmailSender) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentEmails = make([]*notification.EmailTask, 0)
	m.sendError = nil
}

// WasSentTo checks if an email was sent to a specific address
func (m *MockEmailSender) WasSentTo(address string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, email := range m.sentEmails {
		if email.ToAddress == address {
			return true
		}
	}
	return false
}

// GetEmailsSentTo returns all emails sent to a specific address
func (m *MockEmailSender) GetEmailsSentTo(address string) []*notification.EmailTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var emails []*notification.EmailTask
	for _, email := range m.sentEmails {
		if email.ToAddress == address {
			emailCopy := *email
			emails = append(emails, &emailCopy)
		}
	}
	return emails
}
