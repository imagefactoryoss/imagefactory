package email

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewEmailValidationAndDefaults(t *testing.T) {
	_, err := NewEmail(CreateEmailRequest{})
	if err == nil {
		t.Fatal("expected validation error for missing required fields")
	}

	msg, err := NewEmail(CreateEmailRequest{
		TenantID:  uuid.New(),
		ToEmail:   "alice@example.com",
		FromEmail: "noreply@example.com",
		Subject:   "hello",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if msg.Priority() != 5 {
		t.Fatalf("expected default priority 5, got %d", msg.Priority())
	}
	if msg.EmailType() != EmailTypeGeneral {
		t.Fatalf("expected default email type general, got %s", msg.EmailType())
	}
	if msg.Status() != EmailStatusPending {
		t.Fatalf("expected pending status, got %s", msg.Status())
	}
}

func TestEmailStatusTransitions(t *testing.T) {
	msg, err := NewEmail(CreateEmailRequest{
		TenantID:  uuid.New(),
		ToEmail:   "alice@example.com",
		FromEmail: "noreply@example.com",
		Subject:   "hello",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if err := msg.MarkAsSent(); err != ErrInvalidEmailStatus {
		t.Fatalf("expected invalid status error, got %v", err)
	}

	if err := msg.MarkAsProcessing(); err != nil {
		t.Fatalf("expected mark processing success, got %v", err)
	}
	if msg.ProcessedAt() == nil {
		t.Fatal("expected processedAt to be populated")
	}

	if err := msg.MarkAsSent(); err != nil {
		t.Fatalf("expected mark sent success, got %v", err)
	}
	if msg.SentAt() == nil || msg.Status() != EmailStatusSent {
		t.Fatalf("unexpected sent state: status=%s sentAt=%v", msg.Status(), msg.SentAt())
	}
}

func TestMarkAsFailedAndRetryHelpers(t *testing.T) {
	msg := NewEmailFromExisting(
		uuid.New(),
		uuid.New(),
		"alice@example.com",
		"",
		"noreply@example.com",
		"subj",
		"body",
		"",
		EmailTypeGeneral,
		5,
		EmailStatusPending,
		0,
		3,
		"",
		nil,
		"",
		0,
		"",
		"",
		false,
		json.RawMessage(`{}`),
		time.Now().UTC(),
		time.Now().UTC(),
		nil,
		nil,
	)

	if err := msg.MarkAsFailed("smtp error"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if msg.Status() != EmailStatusFailed || msg.RetryCount() != 1 || msg.LastError() == "" {
		t.Fatalf("unexpected failed state: status=%s retry=%d err=%q", msg.Status(), msg.RetryCount(), msg.LastError())
	}
	if !msg.CanRetry() || msg.NextRetryAt() == nil {
		t.Fatal("expected message to be retryable with next retry time set")
	}

	past := time.Now().UTC().Add(-time.Minute)
	ready := NewEmailFromExisting(
		uuid.New(),
		uuid.New(),
		"alice@example.com",
		"",
		"noreply@example.com",
		"subj",
		"body",
		"",
		EmailTypeGeneral,
		5,
		EmailStatusFailed,
		1,
		3,
		"smtp",
		&past,
		"",
		0,
		"",
		"",
		false,
		json.RawMessage(`{}`),
		time.Now().UTC(),
		time.Now().UTC(),
		nil,
		nil,
	)
	if !ready.IsReadyForRetry() {
		t.Fatal("expected IsReadyForRetry=true when nextRetryAt is in past")
	}
}

func TestNewEmailTemplateValidationAndDefaultType(t *testing.T) {
	_, err := NewEmailTemplate(CreateEmailTemplateRequest{})
	if err == nil {
		t.Fatal("expected validation error for missing fields")
	}

	tmpl, err := NewEmailTemplate(CreateEmailTemplateRequest{
		TenantID:        uuid.New(),
		Name:            "welcome",
		SubjectTemplate: "Welcome {{.name}}",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if tmpl.TemplateType() != EmailTypeGeneral {
		t.Fatalf("expected default template type general, got %s", tmpl.TemplateType())
	}
	if !tmpl.IsActive() {
		t.Fatal("expected template active by default")
	}
}
