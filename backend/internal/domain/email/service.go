package email

import (
	"context"
	"encoding/json"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/notification"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

// NotificationService defines the interface for notification operations
type NotificationService interface {
	RenderTemplate(ctx context.Context, companyID, tenantID uuid.UUID, notificationType, channelType, locale string, templateData map[string]interface{}) (*notification.RenderedTemplate, error)
}

// HealthService defines the interface for health check operations
type HealthService interface {
	GetTemplateData(ctx context.Context) (map[string]interface{}, error)
}

// Service handles email business logic
type Service struct {
	repo                Repository
	logger              *zap.Logger
	smtpHost            string
	smtpPort            int
	fromEmail           string
	notificationService NotificationService
	healthService       HealthService
	eventBus            messaging.EventBus
}

// NewService creates a new email service
func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// NewServiceWithSMTP creates a new email service with SMTP configuration
func NewServiceWithSMTP(repo Repository, logger *zap.Logger, smtpHost string, smtpPort int) *Service {
	return &Service{
		repo:     repo,
		logger:   logger,
		smtpHost: smtpHost,
		smtpPort: smtpPort,
	}
}

// NewServiceWithNotification creates a new email service with notification support
func NewServiceWithNotification(repo Repository, logger *zap.Logger, smtpHost string, smtpPort int, fromEmail string, notificationService NotificationService) *Service {
	return &Service{
		repo:                repo,
		logger:              logger,
		smtpHost:            smtpHost,
		smtpPort:            smtpPort,
		fromEmail:           fromEmail,
		notificationService: notificationService,
	}
}

// NewServiceWithHealth creates a new email service with health check support
func NewServiceWithHealth(repo Repository, logger *zap.Logger, smtpHost string, smtpPort int, fromEmail string, notificationService NotificationService, healthService HealthService) *Service {
	return &Service{
		repo:                repo,
		logger:              logger,
		smtpHost:            smtpHost,
		smtpPort:            smtpPort,
		fromEmail:           fromEmail,
		notificationService: notificationService,
		healthService:       healthService,
	}
}

// SetEventBus configures the event bus for publish-on-send notifications.
func (s *Service) SetEventBus(bus messaging.EventBus) {
	s.eventBus = bus
}

// CreateEmail creates a new email in the queue
func (s *Service) CreateEmail(ctx context.Context, req CreateEmailRequest) (*Email, error) {
	email, err := NewEmail(req)
	if err != nil {
		s.logger.Error("Failed to create email", zap.Error(err))
		return nil, err
	}

	if err := s.repo.CreateEmail(ctx, email); err != nil {
		s.logger.Error("Failed to save email", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Email created successfully",
		zap.String("email_id", email.ID().String()),
		zap.String("to", email.ToEmail()),
		zap.String("type", string(email.EmailType())))

	return email, nil
}

// GetEmail retrieves an email by ID
func (s *Service) GetEmail(ctx context.Context, id uuid.UUID) (*Email, error) {
	email, err := s.repo.GetEmailByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get email", zap.String("email_id", id.String()), zap.Error(err))
		return nil, err
	}
	return email, nil
}

// UpdateEmailStatus updates the status of an email
func (s *Service) UpdateEmailStatus(ctx context.Context, id uuid.UUID, req UpdateEmailStatusRequest) error {
	email, err := s.repo.GetEmailByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get email for status update", zap.String("email_id", id.String()), zap.Error(err))
		return err
	}

	// Update status based on request
	switch req.Status {
	case EmailStatusSent:
		if err := email.MarkAsSent(); err != nil {
			return err
		}
		if req.SentAt != nil {
			email.sentAt = req.SentAt
		}
	case EmailStatusFailed:
		if err := email.MarkAsFailed(req.LastError); err != nil {
			return err
		}
	case EmailStatusProcessing:
		if err := email.MarkAsProcessing(); err != nil {
			return err
		}
		if req.ProcessedAt != nil {
			email.processedAt = req.ProcessedAt
		}
	}

	if err := s.repo.UpdateEmail(ctx, email); err != nil {
		s.logger.Error("Failed to update email status", zap.String("email_id", id.String()), zap.Error(err))
		return err
	}

	s.logger.Info("Email status updated",
		zap.String("email_id", id.String()),
		zap.String("status", string(req.Status)))

	return nil
}

// SendHealthCheckEmail creates and queues a health check email using notification templates
func (s *Service) SendHealthCheckEmail(ctx context.Context, companyID, tenantID uuid.UUID, adminEmail string, smtpConfig map[string]interface{}) error {
	// For now, if we don't have notification service, fall back to hardcoded email
	if s.notificationService == nil {
		return s.sendHealthCheckEmailFallback(ctx, companyID, tenantID, adminEmail, smtpConfig)
	}

	// TODO: Get company ID from tenant ID - for now assume a default company ID
	// companyID := uuid.Nil // Default company ID

	// Get template data from health service
	var templateData map[string]interface{}
	if s.healthService != nil {
		var err error
		templateData, err = s.healthService.GetTemplateData(ctx)
		if err != nil {
			s.logger.Error("Failed to get health data for template", zap.Error(err))
			return s.sendHealthCheckEmailFallback(ctx, companyID, tenantID, adminEmail, smtpConfig)
		}
	} else {
		s.logger.Warn("Health service not available, falling back to hardcoded email")
		return s.sendHealthCheckEmailFallback(ctx, companyID, tenantID, adminEmail, smtpConfig)
	}

	// Render the template
	rendered, err := s.notificationService.RenderTemplate(ctx, companyID, tenantID, "system_alert", "email", "en_US", templateData)
	if err != nil {
		s.logger.Error("Failed to render health check template", zap.Error(err))
		return s.sendHealthCheckEmailFallback(ctx, companyID, tenantID, adminEmail, smtpConfig)
	}

	req := CreateEmailRequest{
		TenantID:     tenantID,
		ToEmail:      adminEmail,
		FromEmail:    s.fromEmail,
		Subject:      rendered.Subject,
		BodyText:     rendered.Body,
		EmailType:    EmailTypeHealthCheck,
		Priority:     1, // High priority
		SMTPHost:     s.smtpHost,
		SMTPPort:     s.smtpPort,
		SMTPUsername: "", // TODO: Get from config
		SMTPPassword: "", // TODO: Get from config
		SMTPUseTLS:   true,
		Metadata:     json.RawMessage(`{}`),
	}

	// Set HTML body if available
	if rendered.HTML != nil {
		req.BodyHTML = *rendered.HTML
	}

	_, err = s.CreateEmail(ctx, req)
	if err != nil {
		s.logger.Error("Failed to create health check email", zap.Error(err))
		return err
	}

	s.logger.Info("Health check email queued", zap.String("tenant_id", tenantID.String()), zap.String("admin_email", adminEmail))
	return nil
}

// sendHealthCheckEmailFallback sends a hardcoded health check email when templates are not available
func (s *Service) sendHealthCheckEmailFallback(ctx context.Context, companyID, tenantID uuid.UUID, adminEmail string, smtpConfig map[string]interface{}) error {
	subject := "Image Factory - Health Check"
	bodyText := fmt.Sprintf(`Hello,

This is an automated health check email from Image Factory.

Server Status: ✅ Healthy
Timestamp: %s
Tenant ID: %s

This email confirms that the email system is working correctly.

Best regards,
Image Factory System`, time.Now().UTC().Format(time.RFC3339), tenantID.String())

	bodyHTML := fmt.Sprintf(`<html>
<body>
<h2>Image Factory - Health Check</h2>
<p>Hello,</p>
<p>This is an automated health check email from Image Factory.</p>
<ul>
<li><strong>Server Status:</strong> ✅ Healthy</li>
<li><strong>Timestamp:</strong> %s</li>
<li><strong>Tenant ID:</strong> %s</li>
</ul>
<p>This email confirms that the email system is working correctly.</p>
<p>Best regards,<br>Image Factory System</p>
</body>
</html>`, time.Now().UTC().Format(time.RFC3339), tenantID.String())

	// Extract SMTP config
	var smtpHost, smtpUsername, smtpPassword string
	var smtpPort int
	var smtpUseTLS bool

	if host, ok := smtpConfig["host"].(string); ok {
		smtpHost = host
	}
	if port, ok := smtpConfig["port"].(float64); ok {
		smtpPort = int(port)
	}
	if username, ok := smtpConfig["username"].(string); ok {
		smtpUsername = username
	}
	if password, ok := smtpConfig["password"].(string); ok {
		smtpPassword = password
	}
	if useTLS, ok := smtpConfig["use_tls"].(bool); ok {
		smtpUseTLS = useTLS
	}

	// Use default from email if not specified
	fromEmail := smtpUsername
	if fromEmail == "" {
		fromEmail = s.fromEmail
	}

	req := CreateEmailRequest{
		TenantID:     tenantID,
		ToEmail:      adminEmail,
		FromEmail:    fromEmail,
		Subject:      subject,
		BodyText:     bodyText,
		BodyHTML:     bodyHTML,
		EmailType:    EmailTypeHealthCheck,
		Priority:     1, // High priority
		SMTPHost:     smtpHost,
		SMTPPort:     smtpPort,
		SMTPUsername: smtpUsername,
		SMTPPassword: smtpPassword,
		SMTPUseTLS:   smtpUseTLS,
		Metadata:     json.RawMessage(`{}`), // Initialize with empty JSON object
	}

	_, err := s.CreateEmail(ctx, req)
	if err != nil {
		s.logger.Error("Failed to create health check email", zap.Error(err))
		return err
	}

	s.logger.Info("Health check email queued", zap.String("tenant_id", tenantID.String()), zap.String("admin_email", adminEmail))
	return nil
}

// RenderUserInvitationTemplate renders a user invitation email template
func (s *Service) RenderUserInvitationTemplate(ctx context.Context, companyID, tenantID uuid.UUID, templateData map[string]interface{}) (*notification.RenderedTemplate, error) {
	if s.notificationService == nil {
		return nil, fmt.Errorf("notification service not available")
	}

	return s.notificationService.RenderTemplate(ctx, companyID, tenantID, "user_invitation", "email", "en_US", templateData)
}

// RenderUserInvitationCancelledTemplate renders a user invitation cancelled email template
func (s *Service) RenderUserInvitationCancelledTemplate(ctx context.Context, companyID, tenantID uuid.UUID, templateData map[string]interface{}) (*notification.RenderedTemplate, error) {
	if s.notificationService == nil {
		return nil, fmt.Errorf("notification service not available")
	}

	return s.notificationService.RenderTemplate(ctx, companyID, tenantID, "user_invitation_cancelled", "email", "en_US", templateData)
}

// RenderTenantOnboardingTemplate renders a tenant onboarding email template
func (s *Service) RenderTenantOnboardingTemplate(ctx context.Context, companyID, tenantID uuid.UUID, templateData map[string]interface{}) (*notification.RenderedTemplate, error) {
	if s.notificationService == nil {
		return nil, fmt.Errorf("notification service not available")
	}

	return s.notificationService.RenderTemplate(ctx, companyID, tenantID, "tenant_onboarding", "email", "en_US", templateData)
}

// RenderUserAddedToGroupTemplate renders a user added to group email template
func (s *Service) RenderUserAddedToGroupTemplate(ctx context.Context, companyID, tenantID uuid.UUID, templateData map[string]interface{}) (*notification.RenderedTemplate, error) {
	if s.notificationService == nil {
		return nil, fmt.Errorf("notification service not available")
	}

	return s.notificationService.RenderTemplate(ctx, companyID, tenantID, "user_added_to_group", "email", "en_US", templateData)
}

// RenderUserAddedToTenantTemplate renders a user added to tenant email template
func (s *Service) RenderUserAddedToTenantTemplate(ctx context.Context, companyID, tenantID uuid.UUID, templateData map[string]interface{}) (*notification.RenderedTemplate, error) {
	if s.notificationService == nil {
		return nil, fmt.Errorf("notification service not available")
	}

	return s.notificationService.RenderTemplate(ctx, companyID, tenantID, "user_added_to_tenant", "email", "en_US", templateData)
}

// RenderUserRoleChangedTemplate renders a user role changed email template
func (s *Service) RenderUserRoleChangedTemplate(ctx context.Context, companyID, tenantID uuid.UUID, templateData map[string]interface{}) (*notification.RenderedTemplate, error) {
	if s.notificationService == nil {
		return nil, fmt.Errorf("notification service not available")
	}

	return s.notificationService.RenderTemplate(ctx, companyID, tenantID, "user_role_changed", "email", "en_US", templateData)
}

// RenderUserRemovedFromTenantTemplate renders a user removed from tenant email template
func (s *Service) RenderUserRemovedFromTenantTemplate(ctx context.Context, companyID, tenantID uuid.UUID, templateData map[string]interface{}) (*notification.RenderedTemplate, error) {
	if s.notificationService == nil {
		return nil, fmt.Errorf("notification service not available")
	}

	return s.notificationService.RenderTemplate(ctx, companyID, tenantID, "user_removed_from_tenant", "email", "en_US", templateData)
}

// RenderUserSuspendedTemplate renders a user suspended email template
func (s *Service) RenderUserSuspendedTemplate(ctx context.Context, companyID, tenantID uuid.UUID, templateData map[string]interface{}) (*notification.RenderedTemplate, error) {
	if s.notificationService == nil {
		return nil, fmt.Errorf("notification service not available")
	}

	return s.notificationService.RenderTemplate(ctx, companyID, tenantID, "user_suspended", "email", "en_US", templateData)
}

// RenderUserActivatedTemplate renders a user activated email template
func (s *Service) RenderUserActivatedTemplate(ctx context.Context, companyID, tenantID uuid.UUID, templateData map[string]interface{}) (*notification.RenderedTemplate, error) {
	if s.notificationService == nil {
		return nil, fmt.Errorf("notification service not available")
	}

	return s.notificationService.RenderTemplate(ctx, companyID, tenantID, "user_activated", "email", "en_US", templateData)
}

// RenderBuildNotificationTemplate renders a build notification email template by template type.
func (s *Service) RenderBuildNotificationTemplate(ctx context.Context, companyID, tenantID uuid.UUID, templateType string, templateData map[string]interface{}) (*notification.RenderedTemplate, error) {
	if s.notificationService == nil {
		return nil, fmt.Errorf("notification service not available")
	}
	return s.notificationService.RenderTemplate(ctx, companyID, tenantID, templateType, "email", "en_US", templateData)
}

// ProcessEmailQueue processes pending emails in the queue
func (s *Service) ProcessEmailQueue(ctx context.Context, batchSize int) error {
	// Get pending emails
	emails, err := s.repo.GetPendingEmails(ctx, batchSize)
	if err != nil {
		s.logger.Error("Failed to get pending emails", zap.Error(err))
		return err
	}

	if len(emails) == 0 {
		s.logger.Debug("No pending emails to process")
		return nil
	}

	s.logger.Info("Processing email queue", zap.Int("count", len(emails)))

	// Process each email
	for _, email := range emails {
		if err := s.processEmail(ctx, email); err != nil {
			s.logger.Error("Failed to process email",
				zap.String("email_id", email.ID().String()),
				zap.Error(err))
		}
	}

	return nil
}

// ProcessRetryQueue processes emails that are ready for retry
func (s *Service) ProcessRetryQueue(ctx context.Context, batchSize int) error {
	emails, err := s.repo.GetEmailsReadyForRetry(ctx, batchSize)
	if err != nil {
		s.logger.Error("Failed to get emails ready for retry", zap.Error(err))
		return err
	}

	if len(emails) == 0 {
		s.logger.Debug("No emails ready for retry")
		return nil
	}

	s.logger.Info("Processing retry queue", zap.Int("count", len(emails)))

	for _, email := range emails {
		if err := s.processEmail(ctx, email); err != nil {
			s.logger.Error("Failed to retry email",
				zap.String("email_id", email.ID().String()),
				zap.Error(err))
		}
	}

	return nil
}

// processEmail sends an individual email
func (s *Service) processEmail(ctx context.Context, email *Email) error {
	start := time.Now().UTC()
	// Mark as processing
	if err := s.UpdateEmailStatus(ctx, email.ID(), UpdateEmailStatusRequest{
		Status:      EmailStatusProcessing,
		ProcessedAt: &time.Time{},
	}); err != nil {
		return fmt.Errorf("failed to mark email as processing: %w", err)
	}

	// Send the email
	if err := s.sendEmail(email); err != nil {
		// Mark as failed
		s.UpdateEmailStatus(ctx, email.ID(), UpdateEmailStatusRequest{
			Status:    EmailStatusFailed,
			LastError: err.Error(),
		})
		s.publishNotificationOutcome(ctx, email, EmailStatusFailed, time.Since(start), err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	// Mark as sent
	if err := s.UpdateEmailStatus(ctx, email.ID(), UpdateEmailStatusRequest{
		Status: EmailStatusSent,
		SentAt: &time.Time{},
	}); err != nil {
		return fmt.Errorf("failed to mark email as sent: %w", err)
	}

	s.logger.Info("Email sent successfully",
		zap.String("email_id", email.ID().String()),
		zap.String("to", email.ToEmail()))
	s.publishNotificationOutcome(ctx, email, EmailStatusSent, time.Since(start), nil)

	return nil
}

func (s *Service) publishNotificationOutcome(ctx context.Context, email *Email, status EmailStatus, duration time.Duration, sendErr error) {
	if s.eventBus == nil {
		return
	}

	payload := map[string]interface{}{
		"email_id":    email.ID().String(),
		"tenant_id":   email.TenantID().String(),
		"to":          email.ToEmail(),
		"subject":     email.Subject(),
		"email_type":  string(email.EmailType()),
		"duration_ms": duration.Milliseconds(),
	}

	if email.CCEmail() != "" {
		payload["cc"] = email.CCEmail()
	}

	if len(email.Metadata()) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(email.Metadata(), &meta); err == nil {
			for k, v := range meta {
				if _, exists := payload[k]; !exists {
					payload[k] = v
				}
			}
		} else {
			payload["metadata"] = string(email.Metadata())
		}
	}

	eventType := "notification.sent"
	if status == EmailStatusFailed {
		eventType = "notification.failed"
		if sendErr != nil {
			payload["error"] = sendErr.Error()
		}
	}

	_ = s.eventBus.Publish(ctx, messaging.Event{
		Type:     eventType,
		TenantID: email.TenantID().String(),
		Payload:  payload,
	})
}

// sendEmail sends an email via SMTP
func (s *Service) sendEmail(email *Email) error {
	// Use email's SMTP settings or service defaults
	smtpHost := email.SMTPHost()
	smtpPort := email.SMTPPort()
	smtpUsername := email.SMTPUsername()
	smtpPassword := email.SMTPPassword()
	smtpUseTLS := email.SMTPUseTLS()

	// Use service defaults if email doesn't have SMTP config
	if smtpHost == "" && s.smtpHost != "" {
		smtpHost = s.smtpHost
	}
	if smtpPort == 0 && s.smtpPort != 0 {
		smtpPort = s.smtpPort
	}
	if !smtpUseTLS {
		smtpUseTLS = false // Mailpit doesn't use TLS
	}

	// Prepare authentication
	var auth smtp.Auth
	if smtpUsername != "" && smtpPassword != "" {
		auth = smtp.PlainAuth("", smtpUsername, smtpPassword, smtpHost)
	}

	// Prepare message
	var message strings.Builder
	message.WriteString(fmt.Sprintf("From: %s\r\n", email.FromEmail()))
	message.WriteString(fmt.Sprintf("To: %s\r\n", email.ToEmail()))
	if email.CCEmail() != "" {
		message.WriteString(fmt.Sprintf("Cc: %s\r\n", email.CCEmail()))
	}
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject()))

	// Add content type
	if email.BodyHTML() != "" {
		message.WriteString("MIME-Version: 1.0\r\n")
		message.WriteString("Content-Type: multipart/alternative; boundary=boundary123\r\n")
		message.WriteString("\r\n")
		message.WriteString("--boundary123\r\n")
		message.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		message.WriteString("\r\n")
		message.WriteString(email.BodyText())
		message.WriteString("\r\n")
		message.WriteString("--boundary123\r\n")
		message.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		message.WriteString("\r\n")
		message.WriteString(email.BodyHTML())
		message.WriteString("\r\n")
		message.WriteString("--boundary123--")
	} else {
		message.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		message.WriteString("\r\n")
		message.WriteString(email.BodyText())
	}

	// Send email
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)
	recipients := []string{email.ToEmail()}
	if email.CCEmail() != "" {
		recipients = append(recipients, email.CCEmail())
	}
	if err := smtp.SendMail(addr, auth, email.FromEmail(), recipients, []byte(message.String())); err != nil {
		return fmt.Errorf("SMTP send failed: %w", err)
	}

	return nil
}

// CreateEmailTemplate creates a new email template
func (s *Service) CreateEmailTemplate(ctx context.Context, req CreateEmailTemplateRequest) (*EmailTemplate, error) {
	template, err := NewEmailTemplate(req)
	if err != nil {
		s.logger.Error("Failed to create email template", zap.Error(err))
		return nil, err
	}

	if err := s.repo.CreateEmailTemplate(ctx, template); err != nil {
		s.logger.Error("Failed to save email template", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Email template created successfully",
		zap.String("template_id", template.ID().String()),
		zap.String("name", template.Name()))

	return template, nil
}

// GetEmailTemplate retrieves an email template by ID
func (s *Service) GetEmailTemplate(ctx context.Context, id uuid.UUID) (*EmailTemplate, error) {
	template, err := s.repo.GetEmailTemplateByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get email template", zap.String("template_id", id.String()), zap.Error(err))
		return nil, err
	}
	return template, nil
}
