package template

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EmailTemplate represents an email template from the database
type EmailTemplate struct {
	ID               uuid.UUID
	CompanyID        uuid.UUID
	TenantID         *uuid.UUID
	TemplateName     string
	NotificationType string
	SubjectTemplate  string
	BodyTemplate     string
	ChannelType      string
	HTMLTemplate     string
	AvailableVars    []string
	IsActive         bool
	IsDefault        bool
	Locale           string
	Metadata         map[string]interface{}
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// TemplateRepository defines the interface for template storage operations
type TemplateRepository interface {
	// GetTemplate retrieves a template by name, company, tenant, and locale
	GetTemplate(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID, templateName, locale string) (*EmailTemplate, error)

	// LoadTemplates loads all templates for a company/tenant combination
	LoadTemplates(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID) ([]*EmailTemplate, error)

	// CreateTemplate creates a new template
	CreateTemplate(ctx context.Context, template *EmailTemplate) error

	// UpdateTemplate updates an existing template
	UpdateTemplate(ctx context.Context, template *EmailTemplate) error

	// DeleteTemplate deletes a template
	DeleteTemplate(ctx context.Context, id uuid.UUID) error

	// InvalidateCache manually invalidates cached templates
	InvalidateCache(companyID uuid.UUID, tenantID uuid.UUID, templateName string)
}

// TemplateNotFoundError is returned when a template doesn't exist
type TemplateNotFoundError struct {
	CompanyID    uuid.UUID
	TenantID     uuid.UUID
	TemplateName string
	Locale       string
}

func (e *TemplateNotFoundError) Error() string {
	return "template not found: " + e.TemplateName
}

// IsTemplateNotFoundError checks if an error is a TemplateNotFoundError
func IsTemplateNotFoundError(err error) bool {
	_, ok := err.(*TemplateNotFoundError)
	return ok
}
