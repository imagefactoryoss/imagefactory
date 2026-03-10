package notification

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service handles notification template operations
type Service struct {
	repo   Repository
	logger *zap.Logger
}

// NewService creates a new notification service
func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// RenderedTemplate represents a rendered notification template
type RenderedTemplate struct {
	Subject string
	Body    string
	HTML    *string
}

// RenderTemplate renders a notification template with the provided data
func (s *Service) RenderTemplate(ctx context.Context, companyID, tenantID uuid.UUID, notificationType, channelType, locale string, templateData map[string]interface{}) (*RenderedTemplate, error) {
	// Get the template
	tmpl, err := s.repo.GetNotificationTemplate(ctx, companyID, tenantID, notificationType, channelType, locale)
	if err != nil {
		s.logger.Error("Failed to get notification template", zap.Error(err))
		return nil, fmt.Errorf("failed to get notification template: %w", err)
	}

	if tmpl == nil {
		s.logger.Warn("No template found for notification",
			zap.String("notification_type", notificationType),
			zap.String("channel_type", channelType),
			zap.String("locale", locale))
		return nil, fmt.Errorf("no template found for notification type %s, channel %s, locale %s", notificationType, channelType, locale)
	}

	// Prepare template data with defaults
	data := s.prepareTemplateData(templateData)

	// Render subject
	subject, err := s.renderTemplate(tmpl.SubjectTemplate, data)
	if err != nil {
		s.logger.Error("Failed to render subject template", zap.Error(err))
		return nil, fmt.Errorf("failed to render subject template: %w", err)
	}

	// Render body
	body, err := s.renderTemplate(tmpl.BodyTemplate, data)
	if err != nil {
		s.logger.Error("Failed to render body template", zap.Error(err))
		return nil, fmt.Errorf("failed to render body template: %w", err)
	}

	rendered := &RenderedTemplate{
		Subject: strings.TrimSpace(subject),
		Body:    strings.TrimSpace(body),
	}

	// Render HTML if available
	if tmpl.HTMLTemplate != nil && *tmpl.HTMLTemplate != "" {
		html, err := s.renderTemplate(*tmpl.HTMLTemplate, data)
		if err != nil {
			s.logger.Error("Failed to render HTML template", zap.Error(err))
			return nil, fmt.Errorf("failed to render HTML template: %w", err)
		}
		html = strings.TrimSpace(html)
		rendered.HTML = &html
	}

	return rendered, nil
}

// prepareTemplateData prepares template data with common defaults
func (s *Service) prepareTemplateData(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		data = make(map[string]interface{})
	}

	// Add current timestamp if not provided
	if _, exists := data["timestamp"]; !exists {
		data["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	}

	// Add current date if not provided
	if _, exists := data["date"]; !exists {
		data["date"] = time.Now().UTC().Format("2006-01-02")
	}

	// Add current time if not provided
	if _, exists := data["time"]; !exists {
		data["time"] = time.Now().UTC().Format("15:04:05")
	}

	return data
}

// renderTemplate renders a Go template with the provided data
func (s *Service) renderTemplate(templateStr string, data map[string]interface{}) (string, error) {
	// Create template with function map that includes the data variables
	funcMap := template.FuncMap{}
	for key, value := range data {
		funcMap[key] = func() interface{} { return value }
	}

	tmpl, err := template.New("notification").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
