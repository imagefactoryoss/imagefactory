package email

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for email data access
type Repository interface {
	// Email operations
	CreateEmail(ctx context.Context, email *Email) error
	GetEmailByID(ctx context.Context, id uuid.UUID) (*Email, error)
	UpdateEmail(ctx context.Context, email *Email) error
	DeleteEmail(ctx context.Context, id uuid.UUID) error
	ListEmails(ctx context.Context, tenantID uuid.UUID, status EmailStatus, limit, offset int) ([]*Email, error)
	GetPendingEmails(ctx context.Context, limit int) ([]*Email, error)
	GetEmailsReadyForRetry(ctx context.Context, limit int) ([]*Email, error)
	CountEmailsByStatus(ctx context.Context, tenantID uuid.UUID, status EmailStatus) (int, error)

	// Email template operations
	CreateEmailTemplate(ctx context.Context, template *EmailTemplate) error
	GetEmailTemplateByID(ctx context.Context, id uuid.UUID) (*EmailTemplate, error)
	GetEmailTemplateByName(ctx context.Context, tenantID uuid.UUID, name string) (*EmailTemplate, error)
	UpdateEmailTemplate(ctx context.Context, template *EmailTemplate) error
	DeleteEmailTemplate(ctx context.Context, id uuid.UUID) error
	ListEmailTemplates(ctx context.Context, tenantID uuid.UUID, activeOnly bool, limit, offset int) ([]*EmailTemplate, error)
}
