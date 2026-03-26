package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"

	domainEmail "github.com/srikarm/image-factory/internal/domain/email"
)

// NotificationTemplate represents an email template
type NotificationTemplate struct {
	ID              uuid.UUID
	CompanyID       uuid.UUID
	TemplateType    string
	Name            string
	Description     string
	SubjectTemplate string
	BodyTemplate    string
	HTMLTemplate    string
	IsDefault       bool
	Enabled         bool
}

// TenantOnboardingData contains data for tenant onboarding email
type TenantOnboardingData struct {
	ContactName  string
	TenantName   string
	TenantID     string
	Status       string
	Company      string
	ContactEmail string
	AdminEmail   string // CC'd recipient (optional)
	APIRateLimit int
	StorageLimit int
	MaxUsers     int
	DashboardURL string
}

// UserAddedToGroupData contains data for user added to group email
type UserAddedToGroupData struct {
	UserEmail    string
	UserName     string
	GroupName    string
	TenantName   string
	TenantID     uuid.UUID
	DashboardURL string
}

// UserAddedToTenantData contains data for user added to tenant email
type UserAddedToTenantData struct {
	UserEmail    string
	UserName     string
	TenantName   string
	TenantID     uuid.UUID
	Role         string
	DashboardURL string
}

// UserRoleChangedData contains data for user role changed email
type UserRoleChangedData struct {
	UserEmail    string
	UserName     string
	TenantName   string
	TenantID     uuid.UUID
	OldRole      string
	NewRole      string
	DashboardURL string
}

// UserRemovedFromTenantData contains data for user removed from tenant email
type UserRemovedFromTenantData struct {
	UserEmail  string
	UserName   string
	TenantName string
	TenantID   uuid.UUID
}

// UserSuspendedData contains data for user account suspended email
type UserSuspendedData struct {
	UserEmail   string
	UserName    string
	TenantName  string
	TenantID    uuid.UUID
	SuspendedAt string
	Reason      string
}

// UserActivatedData contains data for user account activated email
type UserActivatedData struct {
	UserEmail     string
	UserName      string
	TenantName    string
	TenantID      uuid.UUID
	ReactivatedAt string
	DashboardURL  string
}

// NotificationService handles enqueueing email notifications to the worker queue
// Instead of sending emails directly, it persists them to email_queue table
// for async processing by the email worker service on port 8081
type NotificationService struct {
	emailService        *domainEmail.Service
	eventBus            messaging.EventBus
	publishEvents       bool
	logger              *zap.Logger
	fromEmail           string
	companyID           uuid.UUID
	systemConfigService *systemconfig.Service
}

// NewNotificationService creates a new notification service that uses email queue
func NewNotificationService(
	emailService *domainEmail.Service,
	eventBus messaging.EventBus,
	publishEvents bool,
	logger *zap.Logger,
	fromEmail string,
	companyID uuid.UUID,
	systemConfigService *systemconfig.Service,
) *NotificationService {
	return &NotificationService{
		emailService:        emailService,
		eventBus:            eventBus,
		publishEvents:       publishEvents,
		logger:              logger,
		fromEmail:           fromEmail,
		companyID:           companyID,
		systemConfigService: systemConfigService,
	}
}

func (s *NotificationService) requestNotification(ctx context.Context, notificationType string, emailReq domainEmail.CreateEmailRequest, fields ...zap.Field) error {
	if !s.smtpEnabled(ctx, emailReq.TenantID) {
		s.logger.Warn("SMTP disabled; skipping notification dispatch", append(fields, zap.String("notification_type", notificationType))...)
		return nil
	}
	if !s.publishEvents || s.eventBus == nil {
		return s.enqueueEmail(ctx, emailReq, fields...)
	}

	payload := map[string]interface{}{
		"notification_type": notificationType,
		"channel":           "email",
		"tenant_id":         emailReq.TenantID.String(),
		"to":                emailReq.ToEmail,
		"cc":                emailReq.CCEmail,
		"from":              emailReq.FromEmail,
		"subject":           emailReq.Subject,
		"body_text":         emailReq.BodyText,
		"body_html":         emailReq.BodyHTML,
		"email_type":        string(emailReq.EmailType),
		"priority":          emailReq.Priority,
	}

	if len(emailReq.Metadata) > 0 {
		payload["metadata"] = json.RawMessage(emailReq.Metadata)
	}

	event := messaging.Event{
		Type:     "notification.requested",
		TenantID: emailReq.TenantID.String(),
		Payload:  payload,
	}

	if err := s.eventBus.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish notification event, falling back to enqueue",
			append(fields, zap.Error(err))...)
		return s.enqueueEmail(ctx, emailReq, fields...)
	}

	s.logger.Info("Notification event published", fields...)
	return nil
}

func (s *NotificationService) smtpEnabled(ctx context.Context, tenantID uuid.UUID) bool {
	if s.systemConfigService == nil {
		return true
	}

	config, err := s.systemConfigService.GetConfigByTypeAndKey(ctx, &tenantID, systemconfig.ConfigTypeSMTP, "smtp")
	if err != nil {
		if err == systemconfig.ErrConfigNotFound {
			config, err = s.systemConfigService.GetConfigByTypeAndKey(ctx, nil, systemconfig.ConfigTypeSMTP, "smtp")
		}
		if err != nil {
			return true
		}
	}

	smtpConfig, err := config.GetSMTPConfig()
	if err != nil {
		return true
	}

	return smtpConfig.Enabled
}

func (s *NotificationService) enqueueEmail(ctx context.Context, emailReq domainEmail.CreateEmailRequest, fields ...zap.Field) error {
	_, err := s.emailService.CreateEmail(ctx, emailReq)
	if err != nil {
		s.logger.Error("Failed to enqueue email", append(fields, zap.Error(err))...)
		return err
	}
	s.logger.Info("Email enqueued successfully", fields...)
	return nil
}

// SendTenantOnboardingEmail enqueues a tenant onboarding notification email to the worker queue
// The email-worker service on port 8081 will pick it up every 30 seconds and process it
// Non-blocking: logs warning if enqueue fails but doesn't fail the request
func (s *NotificationService) SendTenantOnboardingEmail(ctx context.Context, data *TenantOnboardingData, tenantID uuid.UUID) error {
	// Prepare template data
	templateData := map[string]interface{}{
		"ContactName":  data.ContactName,
		"TenantName":   data.TenantName,
		"TenantID":     data.TenantID,
		"Industry":     data.Status,
		"Country":      data.Company,
		"APIRateLimit": data.APIRateLimit,
		"StorageLimit": data.StorageLimit,
		"MaxUsers":     data.MaxUsers,
		"DashboardURL": data.DashboardURL,
		"ContactEmail": data.ContactEmail,
		"AdminEmail":   data.AdminEmail,
	}

	// Render the template using the notification service
	rendered, err := s.emailService.RenderTenantOnboardingTemplate(ctx, s.companyID, tenantID, templateData)
	if err != nil {
		s.logger.Error("Failed to render tenant onboarding template", zap.Error(err))
		return fmt.Errorf("failed to render tenant onboarding template: %w", err)
	}

	// Create email request for queue processing
	// Tenant ID will be used to route email to correct tenant's SMTP settings
	emailReq := domainEmail.CreateEmailRequest{
		TenantID:  tenantID,
		ToEmail:   data.ContactEmail,
		CCEmail:   data.AdminEmail, // CC the tenant admin if provided
		FromEmail: s.fromEmail,
		Subject:   rendered.Subject,
		BodyText:  rendered.Body,
		EmailType: domainEmail.EmailTypeNotification,
		Priority:  5, // Medium priority
	}

	// Set HTML body if available
	if rendered.HTML != nil {
		emailReq.BodyHTML = *rendered.HTML
	}

	err = s.requestNotification(ctx, "tenant_onboarding", emailReq,
		zap.String("to", data.ContactEmail),
		zap.String("cc", data.AdminEmail),
		zap.String("tenant_name", data.TenantName),
		zap.String("tenant_id", tenantID.String()))
	return err
}

// SendUserAddedToGroupEmail enqueues a notification email when a user is added to a group
// The email-worker service will process this email
// Non-blocking: logs warning if enqueue fails but doesn't fail the request
func (s *NotificationService) SendUserAddedToGroupEmail(ctx context.Context, data *UserAddedToGroupData) error {
	// Prepare template data
	templateData := map[string]interface{}{
		"UserName":     data.UserName,
		"UserEmail":    data.UserEmail,
		"GroupName":    data.GroupName,
		"TenantName":   data.TenantName,
		"DashboardURL": data.DashboardURL,
	}

	// Render the template using the notification service
	rendered, err := s.emailService.RenderUserAddedToGroupTemplate(ctx, s.companyID, data.TenantID, templateData)
	if err != nil {
		s.logger.Error("Failed to render user added to group template", zap.Error(err))
		return fmt.Errorf("failed to render user added to group template: %w", err)
	}

	// Create email request for queue processing
	emailReq := domainEmail.CreateEmailRequest{
		TenantID:  data.TenantID,
		ToEmail:   data.UserEmail,
		FromEmail: s.fromEmail,
		Subject:   rendered.Subject,
		BodyText:  rendered.Body,
		EmailType: domainEmail.EmailTypeNotification,
		Priority:  5, // Medium priority
	}

	// Set HTML body if available
	if rendered.HTML != nil {
		emailReq.BodyHTML = *rendered.HTML
	}

	err = s.requestNotification(ctx, "user_added_to_group", emailReq,
		zap.String("to", data.UserEmail),
		zap.String("group_name", data.GroupName),
		zap.String("tenant_name", data.TenantName))
	return err
}

// SendUserAddedToTenantEmail enqueues a notification email when a user is added to a tenant
// The email-worker service will process this email
// Non-blocking: logs warning if enqueue fails but doesn't fail the request
func (s *NotificationService) SendUserAddedToTenantEmail(ctx context.Context, data *UserAddedToTenantData) error {
	// Prepare template data
	templateData := map[string]interface{}{
		"UserName":     data.UserName,
		"UserEmail":    data.UserEmail,
		"TenantName":   data.TenantName,
		"Role":         data.Role,
		"DashboardURL": data.DashboardURL,
	}

	// Render the template using the notification service
	rendered, err := s.emailService.RenderUserAddedToTenantTemplate(ctx, s.companyID, data.TenantID, templateData)
	if err != nil {
		s.logger.Error("Failed to render user added to tenant template", zap.Error(err))
		// Don't fail the request if template fails - send a basic email instead
		// Create a basic email request
		emailReq := domainEmail.CreateEmailRequest{
			TenantID:  data.TenantID,
			ToEmail:   data.UserEmail,
			FromEmail: s.fromEmail,
			Subject:   "You've been added to " + data.TenantName,
			BodyText:  data.UserName + " has been added to " + data.TenantName + ". Visit the dashboard to get started.",
			EmailType: domainEmail.EmailTypeNotification,
			Priority:  4, // Normal priority
		}

		_ = s.requestNotification(ctx, "user_added_to_tenant", emailReq,
			zap.String("to", data.UserEmail),
			zap.String("tenant_name", data.TenantName))
		return nil // Don't fail the operation
	}

	// Create email request for queue processing
	emailReq := domainEmail.CreateEmailRequest{
		TenantID:  data.TenantID,
		ToEmail:   data.UserEmail,
		FromEmail: s.fromEmail,
		Subject:   rendered.Subject,
		BodyText:  rendered.Body,
		EmailType: domainEmail.EmailTypeNotification,
		Priority:  4, // Normal priority
	}

	// Set HTML body if available
	if rendered.HTML != nil {
		emailReq.BodyHTML = *rendered.HTML
	}

	_ = s.requestNotification(ctx, "user_added_to_tenant", emailReq,
		zap.String("to", data.UserEmail),
		zap.String("tenant_name", data.TenantName))
	return nil // Don't fail the operation
}

// SendUserRoleChangedEmail enqueues a notification email when a user's role is changed in a tenant
// The email-worker service will process this email
// Non-blocking: logs warning if enqueue fails but doesn't fail the request
func (s *NotificationService) SendUserRoleChangedEmail(ctx context.Context, data *UserRoleChangedData) error {
	// Prepare template data
	templateData := map[string]interface{}{
		"UserName":     data.UserName,
		"UserEmail":    data.UserEmail,
		"TenantName":   data.TenantName,
		"OldRole":      data.OldRole,
		"NewRole":      data.NewRole,
		"DashboardURL": data.DashboardURL,
	}

	// Render the template using the notification service
	rendered, err := s.emailService.RenderUserRoleChangedTemplate(ctx, s.companyID, data.TenantID, templateData)
	if err != nil {
		s.logger.Error("Failed to render user role changed template", zap.Error(err))
		// Don't fail the request if template fails - send a basic email instead
		emailReq := domainEmail.CreateEmailRequest{
			TenantID:  data.TenantID,
			ToEmail:   data.UserEmail,
			FromEmail: s.fromEmail,
			Subject:   "Your role in " + data.TenantName + " has changed",
			BodyText:  "Your role in " + data.TenantName + " has been changed from " + data.OldRole + " to " + data.NewRole + ".",
			EmailType: domainEmail.EmailTypeNotification,
			Priority:  4, // Normal priority
		}

		_ = s.requestNotification(ctx, "user_role_changed", emailReq,
			zap.String("to", data.UserEmail),
			zap.String("tenant_name", data.TenantName))
		return nil // Don't fail the operation
	}

	// Create email request for queue processing
	emailReq := domainEmail.CreateEmailRequest{
		TenantID:  data.TenantID,
		ToEmail:   data.UserEmail,
		FromEmail: s.fromEmail,
		Subject:   rendered.Subject,
		BodyText:  rendered.Body,
		EmailType: domainEmail.EmailTypeNotification,
		Priority:  4, // Normal priority
	}

	// Set HTML body if available
	if rendered.HTML != nil {
		emailReq.BodyHTML = *rendered.HTML
	}

	_ = s.requestNotification(ctx, "user_role_changed", emailReq,
		zap.String("to", data.UserEmail),
		zap.String("tenant_name", data.TenantName),
		zap.String("old_role", data.OldRole),
		zap.String("new_role", data.NewRole))
	return nil // Don't fail the operation
}

// SendUserRemovedFromTenantEmail enqueues a notification email when a user is removed from a tenant
// The email-worker service will process this email
// Non-blocking: logs warning if enqueue fails but doesn't fail the request
func (s *NotificationService) SendUserRemovedFromTenantEmail(ctx context.Context, data *UserRemovedFromTenantData) error {
	// Prepare template data
	templateData := map[string]interface{}{
		"UserName":   data.UserName,
		"UserEmail":  data.UserEmail,
		"TenantName": data.TenantName,
	}

	// Render the template using the notification service
	rendered, err := s.emailService.RenderUserRemovedFromTenantTemplate(ctx, s.companyID, data.TenantID, templateData)
	if err != nil {
		s.logger.Error("Failed to render user removed from tenant template", zap.Error(err))
		// Don't fail the request if template fails - send a basic email instead
		emailReq := domainEmail.CreateEmailRequest{
			TenantID:  data.TenantID,
			ToEmail:   data.UserEmail,
			FromEmail: s.fromEmail,
			Subject:   "Removed from " + data.TenantName,
			BodyText:  "You have been removed from " + data.TenantName + ".",
			EmailType: domainEmail.EmailTypeNotification,
			Priority:  4, // Normal priority
		}

		_ = s.requestNotification(ctx, "user_removed_from_tenant", emailReq,
			zap.String("to", data.UserEmail),
			zap.String("tenant_name", data.TenantName))
		return nil // Don't fail the operation
	}

	// Create email request for queue processing
	emailReq := domainEmail.CreateEmailRequest{
		TenantID:  data.TenantID,
		ToEmail:   data.UserEmail,
		FromEmail: s.fromEmail,
		Subject:   rendered.Subject,
		BodyText:  rendered.Body,
		EmailType: domainEmail.EmailTypeNotification,
		Priority:  4, // Normal priority
	}

	// Set HTML body if available
	if rendered.HTML != nil {
		emailReq.BodyHTML = *rendered.HTML
	}

	_ = s.requestNotification(ctx, "user_removed_from_tenant", emailReq,
		zap.String("to", data.UserEmail),
		zap.String("tenant_name", data.TenantName))
	return nil // Don't fail the operation
}

// UserInvitationData contains data for user invitation email
type UserInvitationData struct {
	Email         string
	TenantName    string
	InviterName   string
	InvitationURL string
	Message       string
	ExpiresAt     string
}

// SendUserInvitationEmail enqueues a user invitation notification email to the worker queue
func (s *NotificationService) SendUserInvitationEmail(ctx context.Context, email, tenantName, inviterName, invitationURL, message, expiresAt string, tenantID uuid.UUID) error {
	// Prepare template data
	templateData := map[string]interface{}{
		"Email":         email,
		"TenantName":    tenantName,
		"InviterName":   inviterName,
		"InvitationURL": invitationURL,
		"Message":       message,
		"ExpiresAt":     expiresAt,
	}

	// Render the template using the notification service
	rendered, err := s.emailService.RenderUserInvitationTemplate(ctx, s.companyID, tenantID, templateData)
	if err != nil {
		s.logger.Error("Failed to render user invitation template", zap.Error(err))
		return fmt.Errorf("failed to render user invitation template: %w", err)
	}

	// Create email request
	emailReq := domainEmail.CreateEmailRequest{
		TenantID:  tenantID,
		ToEmail:   email,
		FromEmail: s.fromEmail,
		Subject:   rendered.Subject,
		BodyText:  rendered.Body,
		EmailType: domainEmail.EmailTypeNotification,
		Priority:  3, // High priority for invitations
	}

	// Set HTML body if available
	if rendered.HTML != nil {
		emailReq.BodyHTML = *rendered.HTML
	}

	return s.requestNotification(ctx, "user_invitation", emailReq,
		zap.String("to", email),
		zap.String("tenant_name", tenantName))
}

// SendUserInvitationCancelledEmail enqueues a user invitation cancelled notification email to the worker queue
func (s *NotificationService) SendUserInvitationCancelledEmail(ctx context.Context, email, tenantName string, tenantID uuid.UUID) error {
	// Prepare template data
	templateData := map[string]interface{}{
		"Email":      email,
		"TenantName": tenantName,
	}

	// Render the template using the notification service
	rendered, err := s.emailService.RenderUserInvitationCancelledTemplate(ctx, s.companyID, tenantID, templateData)
	if err != nil {
		s.logger.Error("Failed to render user invitation cancelled template", zap.Error(err))
		return fmt.Errorf("failed to render user invitation cancelled template: %w", err)
	}

	// Create email request
	emailReq := domainEmail.CreateEmailRequest{
		TenantID:  tenantID,
		ToEmail:   email,
		FromEmail: s.fromEmail,
		Subject:   rendered.Subject,
		BodyText:  rendered.Body,
		EmailType: domainEmail.EmailTypeNotification,
		Priority:  2, // Medium priority for cancellations
	}

	// Set HTML body if available
	if rendered.HTML != nil {
		emailReq.BodyHTML = *rendered.HTML
	}

	return s.requestNotification(ctx, "user_invitation_cancelled", emailReq,
		zap.String("to", email),
		zap.String("tenant_name", tenantName))
}

// SendEmail sends a generic email using the notification service
func (s *NotificationService) SendEmail(ctx context.Context, tenantID uuid.UUID, toEmail, subject, bodyText, bodyHTML string, priority int) error {
	emailReq := domainEmail.CreateEmailRequest{
		TenantID:  tenantID,
		ToEmail:   toEmail,
		FromEmail: s.fromEmail,
		Subject:   subject,
		BodyText:  bodyText,
		BodyHTML:  bodyHTML,
		EmailType: domainEmail.EmailTypeNotification,
		Priority:  priority,
	}

	return s.requestNotification(ctx, "generic_email", emailReq,
		zap.String("to", toEmail),
		zap.String("subject", subject),
		zap.String("tenant_id", tenantID.String()))
}

// SendBuildNotificationEmail sends build notification emails using seeded notification templates.
// Falls back to plain subject/body if template rendering is unavailable.
func (s *NotificationService) SendBuildNotificationEmail(
	ctx context.Context,
	tenantID uuid.UUID,
	toEmail string,
	templateType string,
	templateData map[string]interface{},
	fallbackSubject string,
	fallbackBody string,
) error {
	return s.SendBuildNotificationEmailWithCC(ctx, tenantID, toEmail, "", templateType, templateData, fallbackSubject, fallbackBody)
}

// SendBuildNotificationEmailWithCC sends build notification emails with optional CC.
func (s *NotificationService) SendBuildNotificationEmailWithCC(
	ctx context.Context,
	tenantID uuid.UUID,
	toEmail string,
	ccEmail string,
	templateType string,
	templateData map[string]interface{},
	fallbackSubject string,
	fallbackBody string,
) error {
	subject := fallbackSubject
	bodyText := fallbackBody
	bodyHTML := ""

	rendered, err := s.emailService.RenderBuildNotificationTemplate(ctx, s.companyID, tenantID, templateType, templateData)
	if err != nil {
		s.logger.Warn("Failed to render build notification template, falling back to plain email",
			zap.String("template_type", templateType),
			zap.String("tenant_id", tenantID.String()),
			zap.String("to", toEmail),
			zap.Error(err))
	} else if rendered != nil {
		if strings.TrimSpace(rendered.Subject) != "" {
			subject = rendered.Subject
		}
		if strings.TrimSpace(rendered.Body) != "" {
			bodyText = rendered.Body
		}
		if rendered.HTML != nil {
			bodyHTML = strings.TrimSpace(*rendered.HTML)
		}
	}

	emailReq := domainEmail.CreateEmailRequest{
		TenantID:  tenantID,
		ToEmail:   toEmail,
		CCEmail:   strings.TrimSpace(ccEmail),
		FromEmail: s.fromEmail,
		Subject:   subject,
		BodyText:  bodyText,
		BodyHTML:  bodyHTML,
		EmailType: domainEmail.EmailTypeNotification,
		Priority:  1,
	}

	return s.requestNotification(ctx, templateType, emailReq,
		zap.String("to", toEmail),
		zap.String("template_type", templateType),
		zap.String("tenant_id", tenantID.String()))
}

// SendUserSuspendedEmail sends an email notification when a user account is suspended
func (s *NotificationService) SendUserSuspendedEmail(ctx context.Context, data *UserSuspendedData) error {
	// Prepare template data
	templateData := map[string]interface{}{
		"UserName":    data.UserName,
		"UserEmail":   data.UserEmail,
		"TenantName":  data.TenantName,
		"SuspendedAt": data.SuspendedAt,
		"Reason":      data.Reason,
	}

	// Render the template using the notification service
	rendered, err := s.emailService.RenderUserSuspendedTemplate(ctx, s.companyID, data.TenantID, templateData)
	if err != nil {
		s.logger.Error("Failed to render user suspended template", zap.Error(err))
		return fmt.Errorf("failed to render user suspended template: %w", err)
	}

	// Create email request
	emailReq := domainEmail.CreateEmailRequest{
		TenantID:  data.TenantID,
		ToEmail:   data.UserEmail,
		FromEmail: s.fromEmail,
		Subject:   rendered.Subject,
		BodyText:  rendered.Body,
		EmailType: domainEmail.EmailTypeNotification,
		Priority:  1, // Normal priority for account notifications
	}

	// Set HTML body if available
	if rendered.HTML != nil {
		emailReq.BodyHTML = *rendered.HTML
	}

	return s.requestNotification(ctx, "user_suspended", emailReq,
		zap.String("to", data.UserEmail),
		zap.String("tenant_name", data.TenantName))
}

// SendUserActivatedEmail sends an email notification when a suspended user account is reactivated
func (s *NotificationService) SendUserActivatedEmail(ctx context.Context, data *UserActivatedData) error {
	// Prepare template data
	templateData := map[string]interface{}{
		"UserName":      data.UserName,
		"UserEmail":     data.UserEmail,
		"TenantName":    data.TenantName,
		"ReactivatedAt": data.ReactivatedAt,
		"DashboardURL":  data.DashboardURL,
	}

	// Render the template using the notification service
	rendered, err := s.emailService.RenderUserActivatedTemplate(ctx, s.companyID, data.TenantID, templateData)
	if err != nil {
		s.logger.Error("Failed to render user activated template", zap.Error(err))
		return fmt.Errorf("failed to render user activated template: %w", err)
	}

	// Create email request
	emailReq := domainEmail.CreateEmailRequest{
		TenantID:  data.TenantID,
		ToEmail:   data.UserEmail,
		FromEmail: s.fromEmail,
		Subject:   rendered.Subject,
		BodyText:  rendered.Body,
		EmailType: domainEmail.EmailTypeNotification,
		Priority:  1, // Normal priority for account notifications
	}

	// Set HTML body if available
	if rendered.HTML != nil {
		emailReq.BodyHTML = *rendered.HTML
	}

	return s.requestNotification(ctx, "user_activated", emailReq,
		zap.String("to", data.UserEmail),
		zap.String("tenant_name", data.TenantName))
}

// renderTemplate parses and executes a template string with data
func renderTemplate(templateStr string, data interface{}) (string, error) {
	tpl, err := template.New("email").Parse(templateStr)
	if err != nil {
		return "", err
	}

	var result bytes.Buffer
	if err := tpl.Execute(&result, data); err != nil {
		return "", err
	}

	return result.String(), nil
}
