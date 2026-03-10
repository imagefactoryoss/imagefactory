package notification

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Template represents a notification template
type Template struct {
	ID                 uuid.UUID  `json:"id"`
	CompanyID          uuid.UUID  `json:"company_id"`
	TenantID           *uuid.UUID `json:"tenant_id,omitempty"`
	TemplateName       string     `json:"template_name"`
	NotificationType   string     `json:"notification_type"`
	SubjectTemplate    string     `json:"subject_template"`
	BodyTemplate       string     `json:"body_template"`
	ChannelType        string     `json:"channel_type"`
	HTMLTemplate       *string    `json:"html_template,omitempty"`
	AvailableVariables []string   `json:"available_variables"`
	IsActive           bool       `json:"is_active"`
	IsDefault          bool       `json:"is_default"`
	Locale             string     `json:"locale"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// Repository defines the interface for notification template operations
type Repository interface {
	GetNotificationTemplate(ctx context.Context, companyID, tenantID uuid.UUID, notificationType, channelType, locale string) (*Template, error)
}
