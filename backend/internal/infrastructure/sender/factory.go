package sender

import (
	"context"
	"fmt"
	"time"

	"github.com/srikarm/image-factory/internal/domain/notification"
)

// SenderType represents the type of email sender
type SenderType string

const (
	SenderTypeSMTP SenderType = "smtp"
	SenderTypeMock SenderType = "mock"
	// Future: SenderTypeSES, SenderTypeSendGrid, etc.
)

// SenderConfig holds configuration for creating an email sender
type SenderConfig struct {
	Type     SenderType
	SMTP     *SMTPConfig
	Renderer TemplateRenderer
	// Future: SES config, SendGrid config, etc.
}

// EmailSenderFactory creates email senders based on configuration
type EmailSenderFactory struct {
	defaultRenderer TemplateRenderer
}

// NewEmailSenderFactory creates a new email sender factory
func NewEmailSenderFactory() *EmailSenderFactory {
	return &EmailSenderFactory{
		defaultRenderer: NewGoTemplateRenderer(),
	}
}

// NewEmailSenderFactoryWithRenderer creates a factory with a custom renderer
func NewEmailSenderFactoryWithRenderer(renderer TemplateRenderer) *EmailSenderFactory {
	return &EmailSenderFactory{
		defaultRenderer: renderer,
	}
}

// CreateSender creates an email sender based on the provided configuration
func (f *EmailSenderFactory) CreateSender(config SenderConfig) (interface {
	Send(context.Context, *notification.EmailTask) error
}, error) {
	// Use provided renderer or fall back to default
	renderer := config.Renderer
	if renderer == nil {
		renderer = f.defaultRenderer
	}

	switch config.Type {
	case SenderTypeSMTP:
		if config.SMTP == nil {
			return nil, fmt.Errorf("SMTP configuration is required for SMTP sender")
		}

		// Validate SMTP config
		if err := config.SMTP.Validate(); err != nil {
			return nil, fmt.Errorf("invalid SMTP configuration: %w", err)
		}

		// Create template-enabled SMTP sender
		return NewTemplateEmailSender(*config.SMTP, renderer), nil

	case SenderTypeMock:
		// Create mock sender for testing
		return NewMockEmailSender(), nil

	default:
		return nil, fmt.Errorf("unsupported sender type: %s", config.Type)
	}
}

// CreateSMTPSender is a convenience method for creating SMTP senders
func (f *EmailSenderFactory) CreateSMTPSender(config SMTPConfig) (*TemplateEmailSender, error) {
	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid SMTP configuration: %w", err)
	}

	return NewTemplateEmailSender(config, f.defaultRenderer), nil
}

// CreateMockSender is a convenience method for creating mock senders
func (f *EmailSenderFactory) CreateMockSender() *MockEmailSender {
	return NewMockEmailSender()
}

// Helper function to create an SMTP sender with default settings
func CreateDefaultSMTPSender(host string, port int, username, password, from string) (*TemplateEmailSender, error) {
	config := SMTPConfig{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		UseTLS:   true,
		Timeout:  30 * time.Second,
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	renderer := NewGoTemplateRenderer()
	return NewTemplateEmailSender(config, renderer), nil
}
