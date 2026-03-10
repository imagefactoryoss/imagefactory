package email

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Domain errors
var (
	ErrEmailNotFound      = errors.New("email not found")
	ErrEmailAlreadySent   = errors.New("email already sent")
	ErrEmailSendFailed    = errors.New("email send failed")
	ErrInvalidEmailStatus = errors.New("invalid email status")
	ErrMaxRetriesExceeded = errors.New("maximum retries exceeded")
)

// EmailStatus represents the status of an email
type EmailStatus string

const (
	EmailStatusPending    EmailStatus = "pending"
	EmailStatusProcessing EmailStatus = "processing"
	EmailStatusSent       EmailStatus = "sent"
	EmailStatusFailed     EmailStatus = "failed"
	EmailStatusCancelled  EmailStatus = "cancelled"
)

// EmailType represents the type of email
type EmailType string

const (
	EmailTypeHealthCheck   EmailType = "health_check"
	EmailTypeNotification  EmailType = "notification"
	EmailTypeAlert         EmailType = "alert"
	EmailTypeWelcome       EmailType = "welcome"
	EmailTypePasswordReset EmailType = "password_reset"
	EmailTypeGeneral       EmailType = "general"
)

// Email represents an email to be sent
type Email struct {
	id           uuid.UUID
	tenantID     uuid.UUID
	toEmail      string
	ccEmail      string // CC'd recipient (optional)
	fromEmail    string
	subject      string
	bodyText     string
	bodyHTML     string
	emailType    EmailType
	priority     int
	status       EmailStatus
	retryCount   int
	maxRetries   int
	lastError    string
	nextRetryAt  *time.Time
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	smtpUseTLS   bool
	metadata     json.RawMessage
	createdAt    time.Time
	updatedAt    time.Time
	sentAt       *time.Time
	processedAt  *time.Time
}

// EmailTemplate represents a reusable email template
type EmailTemplate struct {
	id               uuid.UUID
	tenantID         uuid.UUID
	name             string
	description      string
	templateType     EmailType
	subjectTemplate  string
	bodyTextTemplate string
	bodyHTMLTemplate string
	variablesSchema  json.RawMessage
	isActive         bool
	metadata         json.RawMessage
	createdAt        time.Time
	updatedAt        time.Time
}

// CreateEmailRequest represents a request to create a new email
type CreateEmailRequest struct {
	TenantID     uuid.UUID
	ToEmail      string
	CCEmail      string // Optional CC'd recipient
	FromEmail    string
	Subject      string
	BodyText     string
	BodyHTML     string
	EmailType    EmailType
	Priority     int
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPUseTLS   bool
	Metadata     json.RawMessage
}

// UpdateEmailStatusRequest represents a request to update email status
type UpdateEmailStatusRequest struct {
	Status      EmailStatus
	LastError   string
	SentAt      *time.Time
	ProcessedAt *time.Time
}

// CreateEmailTemplateRequest represents a request to create an email template
type CreateEmailTemplateRequest struct {
	TenantID         uuid.UUID
	Name             string
	Description      string
	TemplateType     EmailType
	SubjectTemplate  string
	BodyTextTemplate string
	BodyHTMLTemplate string
	VariablesSchema  json.RawMessage
	Metadata         json.RawMessage
}

// NewEmailTemplate creates a new email template
func NewEmailTemplate(req CreateEmailTemplateRequest) (*EmailTemplate, error) {
	if req.TenantID == uuid.Nil {
		return nil, errors.New("tenant ID is required")
	}
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if req.SubjectTemplate == "" {
		return nil, errors.New("subject template is required")
	}

	now := time.Now().UTC()
	template := &EmailTemplate{
		id:               uuid.New(),
		tenantID:         req.TenantID,
		name:             req.Name,
		description:      req.Description,
		templateType:     req.TemplateType,
		subjectTemplate:  req.SubjectTemplate,
		bodyTextTemplate: req.BodyTextTemplate,
		bodyHTMLTemplate: req.BodyHTMLTemplate,
		variablesSchema:  req.VariablesSchema,
		isActive:         true,
		metadata:         req.Metadata,
		createdAt:        now,
		updatedAt:        now,
	}

	if template.templateType == "" {
		template.templateType = EmailTypeGeneral
	}

	return template, nil
}

// NewEmail creates a new email
func NewEmail(req CreateEmailRequest) (*Email, error) {
	// Allow system tenant ID (all zeros UUID) as valid
	// No tenant ID validation required
	if req.ToEmail == "" {
		return nil, errors.New("to email is required")
	}
	if req.FromEmail == "" {
		return nil, errors.New("from email is required")
	}
	if req.Subject == "" {
		return nil, errors.New("subject is required")
	}

	now := time.Now().UTC()
	email := &Email{
		id:           uuid.New(),
		tenantID:     req.TenantID,
		toEmail:      req.ToEmail,
		ccEmail:      req.CCEmail,
		fromEmail:    req.FromEmail,
		subject:      req.Subject,
		bodyText:     req.BodyText,
		bodyHTML:     req.BodyHTML,
		emailType:    req.EmailType,
		priority:     req.Priority,
		status:       EmailStatusPending,
		retryCount:   0,
		maxRetries:   3,
		smtpHost:     req.SMTPHost,
		smtpPort:     req.SMTPPort,
		smtpUsername: req.SMTPUsername,
		smtpPassword: req.SMTPPassword,
		smtpUseTLS:   req.SMTPUseTLS,
		metadata:     req.Metadata,
		createdAt:    now,
		updatedAt:    now,
	}

	if email.priority == 0 {
		email.priority = 5 // Default priority
	}
	if email.emailType == "" {
		email.emailType = EmailTypeGeneral
	}

	return email, nil
}

// NewEmailFromExisting creates an email from existing data (used by repository)
func NewEmailFromExisting(
	id uuid.UUID,
	tenantID uuid.UUID,
	toEmail, ccEmail, fromEmail, subject, bodyText, bodyHTML string,
	emailType EmailType,
	priority int,
	status EmailStatus,
	retryCount, maxRetries int,
	lastError string,
	nextRetryAt *time.Time,
	smtpHost string,
	smtpPort int,
	smtpUsername, smtpPassword string,
	smtpUseTLS bool,
	metadata json.RawMessage,
	createdAt, updatedAt time.Time,
	sentAt, processedAt *time.Time,
) *Email {
	return &Email{
		id:           id,
		tenantID:     tenantID,
		toEmail:      toEmail,
		ccEmail:      ccEmail,
		fromEmail:    fromEmail,
		subject:      subject,
		bodyText:     bodyText,
		bodyHTML:     bodyHTML,
		emailType:    emailType,
		priority:     priority,
		status:       status,
		retryCount:   retryCount,
		maxRetries:   maxRetries,
		lastError:    lastError,
		nextRetryAt:  nextRetryAt,
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		smtpUsername: smtpUsername,
		smtpPassword: smtpPassword,
		smtpUseTLS:   smtpUseTLS,
		metadata:     metadata,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		sentAt:       sentAt,
		processedAt:  processedAt,
	}
}

// NewEmailTemplateFromExisting creates an email template from existing data (used by repository)
func NewEmailTemplateFromExisting(
	id uuid.UUID,
	tenantID uuid.UUID,
	name, description string,
	templateType EmailType,
	subjectTemplate, bodyTextTemplate, bodyHTMLTemplate string,
	variablesSchema json.RawMessage,
	isActive bool,
	metadata json.RawMessage,
	createdAt, updatedAt time.Time,
) *EmailTemplate {
	return &EmailTemplate{
		id:               id,
		tenantID:         tenantID,
		name:             name,
		description:      description,
		templateType:     templateType,
		subjectTemplate:  subjectTemplate,
		bodyTextTemplate: bodyTextTemplate,
		bodyHTMLTemplate: bodyHTMLTemplate,
		variablesSchema:  variablesSchema,
		isActive:         isActive,
		metadata:         metadata,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
	}
}

// Getters
func (e *Email) ID() uuid.UUID             { return e.id }
func (e *Email) TenantID() uuid.UUID       { return e.tenantID }
func (e *Email) ToEmail() string           { return e.toEmail }
func (e *Email) CCEmail() string           { return e.ccEmail }
func (e *Email) FromEmail() string         { return e.fromEmail }
func (e *Email) Subject() string           { return e.subject }
func (e *Email) BodyText() string          { return e.bodyText }
func (e *Email) BodyHTML() string          { return e.bodyHTML }
func (e *Email) EmailType() EmailType      { return e.emailType }
func (e *Email) Priority() int             { return e.priority }
func (e *Email) Status() EmailStatus       { return e.status }
func (e *Email) RetryCount() int           { return e.retryCount }
func (e *Email) MaxRetries() int           { return e.maxRetries }
func (e *Email) LastError() string         { return e.lastError }
func (e *Email) NextRetryAt() *time.Time   { return e.nextRetryAt }
func (e *Email) SMTPHost() string          { return e.smtpHost }
func (e *Email) SMTPPort() int             { return e.smtpPort }
func (e *Email) SMTPUsername() string      { return e.smtpUsername }
func (e *Email) SMTPPassword() string      { return e.smtpPassword }
func (e *Email) SMTPUseTLS() bool          { return e.smtpUseTLS }
func (e *Email) Metadata() json.RawMessage { return e.metadata }
func (e *Email) CreatedAt() time.Time      { return e.createdAt }
func (e *Email) UpdatedAt() time.Time      { return e.updatedAt }
func (e *Email) SentAt() *time.Time        { return e.sentAt }
func (e *Email) ProcessedAt() *time.Time   { return e.processedAt }

// Getters for EmailTemplate
func (t *EmailTemplate) ID() uuid.UUID                    { return t.id }
func (t *EmailTemplate) TenantID() uuid.UUID              { return t.tenantID }
func (t *EmailTemplate) Name() string                     { return t.name }
func (t *EmailTemplate) Description() string              { return t.description }
func (t *EmailTemplate) TemplateType() EmailType          { return t.templateType }
func (t *EmailTemplate) SubjectTemplate() string          { return t.subjectTemplate }
func (t *EmailTemplate) BodyTextTemplate() string         { return t.bodyTextTemplate }
func (t *EmailTemplate) BodyHTMLTemplate() string         { return t.bodyHTMLTemplate }
func (t *EmailTemplate) VariablesSchema() json.RawMessage { return t.variablesSchema }
func (t *EmailTemplate) IsActive() bool                   { return t.isActive }
func (t *EmailTemplate) Metadata() json.RawMessage        { return t.metadata }
func (t *EmailTemplate) CreatedAt() time.Time             { return t.createdAt }
func (t *EmailTemplate) UpdatedAt() time.Time             { return t.updatedAt }

// Business logic methods
func (e *Email) MarkAsProcessing() error {
	if e.status != EmailStatusPending && e.status != EmailStatusFailed {
		return ErrInvalidEmailStatus
	}
	e.status = EmailStatusProcessing
	e.processedAt = &time.Time{}
	*e.processedAt = time.Now().UTC()
	e.updatedAt = time.Now().UTC()
	return nil
}

func (e *Email) MarkAsSent() error {
	if e.status != EmailStatusProcessing {
		return ErrInvalidEmailStatus
	}
	e.status = EmailStatusSent
	e.sentAt = &time.Time{}
	*e.sentAt = time.Now().UTC()
	e.updatedAt = time.Now().UTC()
	return nil
}

func (e *Email) MarkAsFailed(errorMsg string) error {
	e.status = EmailStatusFailed
	e.lastError = errorMsg
	e.retryCount++

	if e.retryCount < e.maxRetries {
		// Schedule next retry with exponential backoff
		nextRetry := time.Now().UTC().Add(time.Duration(e.retryCount*e.retryCount) * time.Minute)
		e.nextRetryAt = &nextRetry
	}

	e.updatedAt = time.Now().UTC()
	return nil
}

func (e *Email) CanRetry() bool {
	return e.retryCount < e.maxRetries && e.status == EmailStatusFailed
}

func (e *Email) IsReadyForRetry() bool {
	return e.CanRetry() && e.nextRetryAt != nil && time.Now().UTC().After(*e.nextRetryAt)
}
