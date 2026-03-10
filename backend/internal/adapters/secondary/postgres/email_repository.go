package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/email"
)

// EmailRepository implements the email.Repository interface
type EmailRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewEmailRepository creates a new email repository
func NewEmailRepository(db *sqlx.DB, logger *zap.Logger) *EmailRepository {
	return &EmailRepository{
		db:     db,
		logger: logger,
	}
}

// CreateEmail creates a new email in the database
func (r *EmailRepository) CreateEmail(ctx context.Context, email *email.Email) error {
	query := `
		INSERT INTO email_queue (
			id, tenant_id, to_email, cc_email, from_email, subject, body_text, body_html,
			email_type, priority, status, retry_count, max_retries,
			smtp_host, smtp_port, smtp_username, smtp_password, smtp_use_tls,
			metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19, $20, $21
		)`

	// Handle nil metadata by converting to empty JSON object
	metadata := email.Metadata()
	if metadata == nil {
		metadata = []byte("{}")
	}

	// Handle empty CC as NULL
	ccEmail := email.CCEmail()
	var ccEmailPtr *string
	if ccEmail != "" {
		ccEmailPtr = &ccEmail
	}

	_, err := r.db.ExecContext(ctx, query,
		email.ID(),
		email.TenantID(),
		email.ToEmail(),
		ccEmailPtr,
		email.FromEmail(),
		email.Subject(),
		email.BodyText(),
		email.BodyHTML(),
		email.EmailType(),
		email.Priority(),
		email.Status(),
		email.RetryCount(),
		email.MaxRetries(),
		email.SMTPHost(),
		email.SMTPPort(),
		email.SMTPUsername(),
		email.SMTPPassword(),
		email.SMTPUseTLS(),
		metadata,
		email.CreatedAt(),
		email.UpdatedAt(),
	)

	if err != nil {
		r.logger.Error("Failed to create email", zap.Error(err))
		return err
	}

	return nil
}

// GetEmailByID retrieves an email by ID
func (r *EmailRepository) GetEmailByID(ctx context.Context, id uuid.UUID) (*email.Email, error) {
	query := `
		SELECT id, tenant_id, to_email, from_email, subject, body_text, body_html,
			   email_type, priority, status, retry_count, max_retries, last_error,
			   next_retry_at, smtp_host, smtp_port, smtp_username, smtp_password,
			   smtp_use_tls, metadata, created_at, updated_at, sent_at, processed_at
		FROM email_queue
		WHERE id = $1`

	var e emailRow
	err := r.db.GetContext(ctx, &e, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, email.ErrEmailNotFound
		}
		r.logger.Error("Failed to get email", zap.String("email_id", id.String()), zap.Error(err))
		return nil, err
	}

	return e.toDomain(), nil
}

// UpdateEmail updates an email in the database
func (r *EmailRepository) UpdateEmail(ctx context.Context, email *email.Email) error {
	query := `
		UPDATE email_queue SET
			status = $1, retry_count = $2, last_error = $3, next_retry_at = $4,
			sent_at = $5, processed_at = $6, updated_at = $7
		WHERE id = $8`

	_, err := r.db.ExecContext(ctx, query,
		email.Status(),
		email.RetryCount(),
		email.LastError(),
		email.NextRetryAt(),
		email.SentAt(),
		email.ProcessedAt(),
		email.UpdatedAt(),
		email.ID(),
	)

	if err != nil {
		r.logger.Error("Failed to update email", zap.String("email_id", email.ID().String()), zap.Error(err))
		return err
	}

	return nil
}

// DeleteEmail deletes an email from the database
func (r *EmailRepository) DeleteEmail(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM email_queue WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete email", zap.String("email_id", id.String()), zap.Error(err))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return email.ErrEmailNotFound
	}

	return nil
}

// ListEmails retrieves emails with filtering
func (r *EmailRepository) ListEmails(ctx context.Context, tenantID uuid.UUID, status email.EmailStatus, limit, offset int) ([]*email.Email, error) {
	query := `
		SELECT id, tenant_id, to_email, from_email, subject, body_text, body_html,
			   email_type, priority, status, retry_count, max_retries, last_error,
			   next_retry_at, smtp_host, smtp_port, smtp_username, smtp_password,
			   smtp_use_tls, metadata, created_at, updated_at, sent_at, processed_at
		FROM email_queue
		WHERE tenant_id = $1`

	args := []interface{}{tenantID}
	argCount := 1

	if status != "" {
		argCount++
		query += ` AND status = $` + fmt.Sprintf("%d", argCount)
		args = append(args, status)
	}

	query += ` ORDER BY priority DESC, created_at DESC LIMIT $` + fmt.Sprintf("%d", argCount+1) + ` OFFSET $` + fmt.Sprintf("%d", argCount+2)
	args = append(args, limit, offset)

	var rows []emailRow
	err := r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		r.logger.Error("Failed to list emails", zap.Error(err))
		return nil, err
	}

	emails := make([]*email.Email, len(rows))
	for i, row := range rows {
		emails[i] = row.toDomain()
	}

	return emails, nil
}

// GetPendingEmails retrieves pending emails for processing
func (r *EmailRepository) GetPendingEmails(ctx context.Context, limit int) ([]*email.Email, error) {
	query := `
		SELECT id, tenant_id, to_email, cc_email, from_email, subject, body_text, body_html,
			   email_type, priority, status, retry_count, max_retries, last_error,
			   next_retry_at, smtp_host, smtp_port, smtp_username, smtp_password,
			   smtp_use_tls, metadata, created_at, updated_at, sent_at, processed_at
		FROM email_queue
		WHERE status = 'pending'
		ORDER BY priority DESC, created_at ASC
		LIMIT $1`

	var rows []emailRow
	err := r.db.SelectContext(ctx, &rows, query, limit)
	if err != nil {
		r.logger.Error("Failed to get pending emails", zap.Error(err))
		return nil, err
	}

	emails := make([]*email.Email, len(rows))
	for i, row := range rows {
		emails[i] = row.toDomain()
	}

	return emails, nil
}

// GetEmailsReadyForRetry retrieves emails ready for retry
func (r *EmailRepository) GetEmailsReadyForRetry(ctx context.Context, limit int) ([]*email.Email, error) {
	query := `
		SELECT id, tenant_id, to_email, cc_email, from_email, subject, body_text, body_html,
			   email_type, priority, status, retry_count, max_retries, last_error,
			   next_retry_at, smtp_host, smtp_port, smtp_username, smtp_password,
			   smtp_use_tls, metadata, created_at, updated_at, sent_at, processed_at
		FROM email_queue
		WHERE status = 'failed' AND retry_count < max_retries AND next_retry_at <= NOW()
		ORDER BY next_retry_at ASC
		LIMIT $1`

	var rows []emailRow
	err := r.db.SelectContext(ctx, &rows, query, limit)
	if err != nil {
		r.logger.Error("Failed to get emails ready for retry", zap.Error(err))
		return nil, err
	}

	emails := make([]*email.Email, len(rows))
	for i, row := range rows {
		emails[i] = row.toDomain()
	}

	return emails, nil
}

// CountEmailsByStatus counts emails by status
func (r *EmailRepository) CountEmailsByStatus(ctx context.Context, tenantID uuid.UUID, status email.EmailStatus) (int, error) {
	query := `SELECT COUNT(*) FROM email_queue WHERE tenant_id = $1 AND status = $2`

	var count int
	err := r.db.GetContext(ctx, &count, query, tenantID, status)
	if err != nil {
		r.logger.Error("Failed to count emails", zap.Error(err))
		return 0, err
	}

	return count, nil
}

// CreateEmailTemplate creates a new email template
func (r *EmailRepository) CreateEmailTemplate(ctx context.Context, template *email.EmailTemplate) error {
	query := `
		INSERT INTO email_templates (
			id, tenant_id, name, description, template_type, subject_template,
			body_text_template, body_html_template, variables_schema, is_active,
			metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.db.ExecContext(ctx, query,
		template.ID(),
		template.TenantID(),
		template.Name(),
		template.Description(),
		template.TemplateType(),
		template.SubjectTemplate(),
		template.BodyTextTemplate(),
		template.BodyHTMLTemplate(),
		template.VariablesSchema(),
		template.IsActive(),
		template.Metadata(),
		template.CreatedAt(),
		template.UpdatedAt(),
	)

	if err != nil {
		r.logger.Error("Failed to create email template", zap.Error(err))
		return err
	}

	return nil
}

// GetEmailTemplateByID retrieves an email template by ID
func (r *EmailRepository) GetEmailTemplateByID(ctx context.Context, id uuid.UUID) (*email.EmailTemplate, error) {
	query := `
		SELECT id, tenant_id, name, description, template_type, subject_template,
			   body_text_template, body_html_template, variables_schema, is_active,
			   metadata, created_at, updated_at
		FROM email_templates
		WHERE id = $1`

	var t templateRow
	err := r.db.GetContext(ctx, &t, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, email.ErrEmailNotFound
		}
		r.logger.Error("Failed to get email template", zap.String("template_id", id.String()), zap.Error(err))
		return nil, err
	}

	return t.toDomain(), nil
}

// GetEmailTemplateByName retrieves an email template by name and tenant
func (r *EmailRepository) GetEmailTemplateByName(ctx context.Context, tenantID uuid.UUID, name string) (*email.EmailTemplate, error) {
	query := `
		SELECT id, tenant_id, name, description, template_type, subject_template,
			   body_text_template, body_html_template, variables_schema, is_active,
			   metadata, created_at, updated_at
		FROM email_templates
		WHERE tenant_id = $1 AND name = $2`

	var t templateRow
	err := r.db.GetContext(ctx, &t, query, tenantID, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, email.ErrEmailNotFound
		}
		r.logger.Error("Failed to get email template by name", zap.String("name", name), zap.Error(err))
		return nil, err
	}

	return t.toDomain(), nil
}

// UpdateEmailTemplate updates an email template
func (r *EmailRepository) UpdateEmailTemplate(ctx context.Context, template *email.EmailTemplate) error {
	query := `
		UPDATE email_templates SET
			name = $1, description = $2, template_type = $3, subject_template = $4,
			body_text_template = $5, body_html_template = $6, variables_schema = $7,
			is_active = $8, metadata = $9, updated_at = $10
		WHERE id = $11`

	_, err := r.db.ExecContext(ctx, query,
		template.Name(),
		template.Description(),
		template.TemplateType(),
		template.SubjectTemplate(),
		template.BodyTextTemplate(),
		template.BodyHTMLTemplate(),
		template.VariablesSchema(),
		template.IsActive(),
		template.Metadata(),
		template.UpdatedAt(),
		template.ID(),
	)

	if err != nil {
		r.logger.Error("Failed to update email template", zap.String("template_id", template.ID().String()), zap.Error(err))
		return err
	}

	return nil
}

// DeleteEmailTemplate deletes an email template
func (r *EmailRepository) DeleteEmailTemplate(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM email_templates WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete email template", zap.String("template_id", id.String()), zap.Error(err))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return email.ErrEmailNotFound
	}

	return nil
}

// ListEmailTemplates retrieves email templates
func (r *EmailRepository) ListEmailTemplates(ctx context.Context, tenantID uuid.UUID, activeOnly bool, limit, offset int) ([]*email.EmailTemplate, error) {
	query := `
		SELECT id, tenant_id, name, description, template_type, subject_template,
			   body_text_template, body_html_template, variables_schema, is_active,
			   metadata, created_at, updated_at
		FROM email_templates
		WHERE tenant_id = $1`

	args := []interface{}{tenantID}
	argCount := 1

	if activeOnly {
		argCount++
		query += ` AND is_active = $` + fmt.Sprintf("%d", argCount)
		args = append(args, true)
	}

	query += ` ORDER BY name ASC LIMIT $` + fmt.Sprintf("%d", argCount+1) + ` OFFSET $` + fmt.Sprintf("%d", argCount+2)
	args = append(args, limit, offset)

	var rows []templateRow
	err := r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		r.logger.Error("Failed to list email templates", zap.Error(err))
		return nil, err
	}

	templates := make([]*email.EmailTemplate, len(rows))
	for i, row := range rows {
		templates[i] = row.toDomain()
	}

	return templates, nil
}

// Database row structs
type emailRow struct {
	ID           uuid.UUID       `db:"id"`
	TenantID     uuid.UUID       `db:"tenant_id"`
	ToEmail      string          `db:"to_email"`
	CCEmail      sql.NullString  `db:"cc_email"`
	FromEmail    string          `db:"from_email"`
	Subject      string          `db:"subject"`
	BodyText     sql.NullString  `db:"body_text"`
	BodyHTML     sql.NullString  `db:"body_html"`
	EmailType    string          `db:"email_type"`
	Priority     int             `db:"priority"`
	Status       string          `db:"status"`
	RetryCount   int             `db:"retry_count"`
	MaxRetries   int             `db:"max_retries"`
	LastError    sql.NullString  `db:"last_error"`
	NextRetryAt  sql.NullTime    `db:"next_retry_at"`
	SMTPHost     sql.NullString  `db:"smtp_host"`
	SMTPPort     sql.NullInt32   `db:"smtp_port"`
	SMTPUsername sql.NullString  `db:"smtp_username"`
	SMTPPassword sql.NullString  `db:"smtp_password"`
	SMTPUseTLS   sql.NullBool    `db:"smtp_use_tls"`
	Metadata     json.RawMessage `db:"metadata"`
	CreatedAt    time.Time       `db:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at"`
	SentAt       sql.NullTime    `db:"sent_at"`
	ProcessedAt  sql.NullTime    `db:"processed_at"`
}

func (r *emailRow) toDomain() *email.Email {
	bodyText := ""
	if r.BodyText.Valid {
		bodyText = r.BodyText.String
	}
	bodyHTML := ""
	if r.BodyHTML.Valid {
		bodyHTML = r.BodyHTML.String
	}
	ccEmail := ""
	if r.CCEmail.Valid {
		ccEmail = r.CCEmail.String
	}
	lastError := ""
	if r.LastError.Valid {
		lastError = r.LastError.String
	}
	var nextRetryAt *time.Time
	if r.NextRetryAt.Valid {
		nextRetryAt = &r.NextRetryAt.Time
	}
	smtpHost := ""
	if r.SMTPHost.Valid {
		smtpHost = r.SMTPHost.String
	}
	smtpPort := 0
	if r.SMTPPort.Valid {
		smtpPort = int(r.SMTPPort.Int32)
	}
	smtpUsername := ""
	if r.SMTPUsername.Valid {
		smtpUsername = r.SMTPUsername.String
	}
	smtpPassword := ""
	if r.SMTPPassword.Valid {
		smtpPassword = r.SMTPPassword.String
	}
	smtpUseTLS := true
	if r.SMTPUseTLS.Valid {
		smtpUseTLS = r.SMTPUseTLS.Bool
	}
	var sentAt *time.Time
	if r.SentAt.Valid {
		sentAt = &r.SentAt.Time
	}
	var processedAt *time.Time
	if r.ProcessedAt.Valid {
		processedAt = &r.ProcessedAt.Time
	}

	return email.NewEmailFromExisting(
		r.ID,
		r.TenantID,
		r.ToEmail,
		ccEmail,
		r.FromEmail,
		r.Subject,
		bodyText,
		bodyHTML,
		email.EmailType(r.EmailType),
		r.Priority,
		email.EmailStatus(r.Status),
		r.RetryCount,
		r.MaxRetries,
		lastError,
		nextRetryAt,
		smtpHost,
		smtpPort,
		smtpUsername,
		smtpPassword,
		smtpUseTLS,
		r.Metadata,
		r.CreatedAt,
		r.UpdatedAt,
		sentAt,
		processedAt,
	)
}

type templateRow struct {
	ID               uuid.UUID       `db:"id"`
	TenantID         uuid.UUID       `db:"tenant_id"`
	Name             string          `db:"name"`
	Description      sql.NullString  `db:"description"`
	TemplateType     string          `db:"template_type"`
	SubjectTemplate  string          `db:"subject_template"`
	BodyTextTemplate sql.NullString  `db:"body_text_template"`
	BodyHTMLTemplate sql.NullString  `db:"body_html_template"`
	VariablesSchema  json.RawMessage `db:"variables_schema"`
	IsActive         bool            `db:"is_active"`
	Metadata         json.RawMessage `db:"metadata"`
	CreatedAt        time.Time       `db:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at"`
}

func (r *templateRow) toDomain() *email.EmailTemplate {
	description := ""
	if r.Description.Valid {
		description = r.Description.String
	}
	bodyTextTemplate := ""
	if r.BodyTextTemplate.Valid {
		bodyTextTemplate = r.BodyTextTemplate.String
	}
	bodyHTMLTemplate := ""
	if r.BodyHTMLTemplate.Valid {
		bodyHTMLTemplate = r.BodyHTMLTemplate.String
	}

	return email.NewEmailTemplateFromExisting(
		r.ID,
		r.TenantID,
		r.Name,
		description,
		email.EmailType(r.TemplateType),
		r.SubjectTemplate,
		bodyTextTemplate,
		bodyHTMLTemplate,
		r.VariablesSchema,
		r.IsActive,
		r.Metadata,
		r.CreatedAt,
		r.UpdatedAt,
	)
}
