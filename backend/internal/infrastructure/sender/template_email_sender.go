package sender

import (
	"context"
	"fmt"

	"github.com/srikarm/image-factory/internal/domain/notification"
)

// TemplateEmailSender wraps SMTPEmailSender with template rendering capabilities
type TemplateEmailSender struct {
	smtpSender *SMTPEmailSender
	renderer   TemplateRenderer

	// In-memory template storage (for simple cases)
	htmlTemplates map[string]string
	textTemplates map[string]string
}

// NewTemplateEmailSender creates a new template-enabled email sender
func NewTemplateEmailSender(config SMTPConfig, renderer TemplateRenderer) *TemplateEmailSender {
	return &TemplateEmailSender{
		smtpSender:    NewSMTPEmailSender(config),
		renderer:      renderer,
		htmlTemplates: make(map[string]string),
		textTemplates: make(map[string]string),
	}
}

// AddHTMLTemplate adds an HTML template by name
func (s *TemplateEmailSender) AddHTMLTemplate(name, template string) {
	s.htmlTemplates[name] = template
}

// AddTextTemplate adds a text template by name
func (s *TemplateEmailSender) AddTextTemplate(name, template string) {
	s.textTemplates[name] = template
}

// Send implements the worker.EmailSender interface with template support
func (s *TemplateEmailSender) Send(ctx context.Context, task *notification.EmailTask) error {
	// If template is specified, render it first
	if err := s.RenderTemplates(ctx, task); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	// Delegate to SMTP sender
	return s.smtpSender.Send(ctx, task)
}

// RenderTemplates renders the template (if specified) and populates BodyHTML/BodyText
func (s *TemplateEmailSender) RenderTemplates(ctx context.Context, task *notification.EmailTask) error {
	// Check context first
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	// If no template name, assume direct body content
	if task.TemplateName == "" {
		return nil
	}

	// Look up templates
	htmlTemplate, hasHTML := s.htmlTemplates[task.TemplateName]
	textTemplate, hasText := s.textTemplates[task.TemplateName]

	if !hasHTML && !hasText {
		return fmt.Errorf("template not found: %s", task.TemplateName)
	}

	// Render HTML template if available
	if hasHTML {
		rendered, err := s.renderer.RenderHTML(ctx, htmlTemplate, task.TemplateData)
		if err != nil {
			return fmt.Errorf("failed to render HTML template %s: %w", task.TemplateName, err)
		}
		task.BodyHTML = rendered
	}

	// Render text template if available
	if hasText {
		rendered, err := s.renderer.RenderText(ctx, textTemplate, task.TemplateData)
		if err != nil {
			return fmt.Errorf("failed to render text template %s: %w", task.TemplateName, err)
		}
		task.BodyText = rendered
	}

	return nil
}
