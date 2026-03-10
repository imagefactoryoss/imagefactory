package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/notification"
)

// NotificationRepository implements the notification.Repository interface
type NotificationRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *sqlx.DB, logger *zap.Logger) *NotificationRepository {
	return &NotificationRepository{
		db:     db,
		logger: logger,
	}
}

// NotificationTemplate represents a notification template from the database
type NotificationTemplate struct {
	ID               uuid.UUID      `db:"id"`
	CompanyID        *uuid.UUID     `db:"company_id"`
	TemplateType     string         `db:"template_type"`
	Name             string         `db:"name"`
	Description      sql.NullString `db:"description"`
	SubjectTemplate  string         `db:"subject_template"`
	BodyTemplate     sql.NullString `db:"body_template"`
	HTMLTemplate     sql.NullString `db:"html_template"`
	IsDefault        bool           `db:"is_default"`
	Enabled          bool           `db:"enabled"`
	CreatedAt        time.Time      `db:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at"`
}

// GetNotificationTemplate retrieves a notification template by type and channel
func (r *NotificationRepository) GetNotificationTemplate(ctx context.Context, companyID, tenantID uuid.UUID, notificationType, channelType, locale string) (*notification.Template, error) {
	query := `
		SELECT id, company_id, template_type, name, description,
		       subject_template, body_template, html_template,
		       is_default, enabled, created_at, updated_at
		FROM notification_templates
		WHERE (company_id = $1 OR company_id IS NULL)
		  AND template_type = $2
		  AND enabled = true
		ORDER BY company_id DESC NULLS LAST, is_default DESC
		LIMIT 1`

	var template NotificationTemplate
	err := r.db.GetContext(ctx, &template, query, companyID, notificationType)
	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Warn("No notification template found",
				zap.String("notification_type", notificationType),
				zap.String("channel_type", channelType),
				zap.String("locale", locale))
			return nil, nil
		}
		r.logger.Error("Failed to get notification template", zap.Error(err))
		return nil, err
	}

	// Convert to domain object
	domainTemplate := &notification.Template{
		ID:               template.ID,
		CompanyID:        uuid.UUID{}, // Default UUID for NULL company_id
		TenantID:         nil, // Not used in current schema
		TemplateName:     template.Name,
		NotificationType: template.TemplateType,
		SubjectTemplate:  template.SubjectTemplate,
		BodyTemplate:     template.BodyTemplate.String,
		ChannelType:      "email", // Hardcoded for now
		AvailableVariables: nil, // Not used in current schema
		IsActive:         template.Enabled,
		IsDefault:        template.IsDefault,
		Locale:           "en_US", // Default locale
		CreatedAt:        template.CreatedAt,
		UpdatedAt:        template.UpdatedAt,
	}

	// Set company ID if not NULL
	if template.CompanyID != nil {
		domainTemplate.CompanyID = *template.CompanyID
	}

	// Set HTML template if available
	if template.HTMLTemplate.Valid {
		domainTemplate.HTMLTemplate = &template.HTMLTemplate.String
	}

	// Set HTML template if available
	if template.HTMLTemplate.Valid {
		domainTemplate.HTMLTemplate = &template.HTMLTemplate.String
	}

	return domainTemplate, nil
}
