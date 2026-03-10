package sender

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/srikarm/image-factory/internal/domain/notification"
)

// SMTPEmailSender implements the EmailSender interface using SMTP
type SMTPEmailSender struct {
	config SMTPConfig
}

// NewSMTPEmailSender creates a new SMTP email sender
func NewSMTPEmailSender(config SMTPConfig) *SMTPEmailSender {
	return &SMTPEmailSender{
		config: config,
	}
}

// Send implements the worker.EmailSender interface
func (s *SMTPEmailSender) Send(ctx context.Context, task *notification.EmailTask) error {
	fmt.Printf("SMTP Send: Starting send to %s\n", task.ToAddress)
	fmt.Printf("SMTP Send: Config - Host:%s Port:%d UseTLS:%v Timeout:%v\n",
		s.config.Host, s.config.Port, s.config.UseTLS, s.config.Timeout)

	// Check context first
	if err := ctx.Err(); err != nil {
		fmt.Printf("SMTP Send: Context error: %v\n", err)
		return fmt.Errorf("context error: %w", err)
	}

	// Validate recipient
	if task.ToAddress == "" {
		return fmt.Errorf("recipient address is required")
	}
	if !emailRegex.MatchString(task.ToAddress) {
		return fmt.Errorf("invalid recipient email address: %s", task.ToAddress)
	}

	// Validate body (at least one must be present)
	if task.BodyHTML == "" && task.BodyText == "" {
		return fmt.Errorf("email body (HTML or text) is required")
	}

	fmt.Printf("SMTP Send: Building message...\n")

	// Build the email message
	message, err := s.buildMessage(task)
	if err != nil {
		fmt.Printf("SMTP Send: Failed to build message: %v\n", err)
		return fmt.Errorf("failed to build message: %w", err)
	}

	fmt.Printf("SMTP Send: Sending via SMTP...\n")

	// Send the email
	if err := s.sendSMTP(ctx, task.ToAddress, message); err != nil {
		fmt.Printf("SMTP Send: Send failed: %v\n", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Printf("SMTP Send: Successfully sent to %s\n", task.ToAddress)
	return nil
}

// buildMessage constructs the email message with headers and body
func (s *SMTPEmailSender) buildMessage(task *notification.EmailTask) ([]byte, error) {
	var builder strings.Builder

	// Headers
	builder.WriteString(fmt.Sprintf("From: %s\r\n", s.config.From))
	builder.WriteString(fmt.Sprintf("To: %s\r\n", task.ToAddress))
	if task.CCAddress != "" {
		builder.WriteString(fmt.Sprintf("Cc: %s\r\n", task.CCAddress))
	}
	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", task.Subject))
	builder.WriteString("MIME-Version: 1.0\r\n")

	// Determine content type based on what's provided
	if task.BodyHTML != "" && task.BodyText != "" {
		// Multipart/alternative (both HTML and plain text)
		boundary := "----=_NextPart_000_0000_01D0000.00000000"
		builder.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		builder.WriteString("\r\n")

		// Plain text part
		builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		builder.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
		builder.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
		builder.WriteString(task.BodyText)
		builder.WriteString("\r\n\r\n")

		// HTML part
		builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		builder.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
		builder.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
		builder.WriteString(task.BodyHTML)
		builder.WriteString("\r\n\r\n")

		// End boundary
		builder.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if task.BodyHTML != "" {
		// HTML only
		builder.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(task.BodyHTML)
	} else {
		// Plain text only
		builder.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(task.BodyText)
	}

	return []byte(builder.String()), nil
}

// sendSMTP sends the email using SMTP protocol
func (s *SMTPEmailSender) sendSMTP(ctx context.Context, to string, message []byte) error {
	// Determine timeout
	timeout := s.config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Dial with context support
	dialer := &net.Dialer{
		Timeout: timeout,
	}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// STARTTLS if enabled
	if s.config.UseTLS {
		tlsConfig := &tls.Config{
			ServerName:         s.config.Host,
			InsecureSkipVerify: false,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Authenticate
	if s.config.Username != "" && s.config.Password != "" {
		auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(s.config.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send message data
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %w", err)
	}

	if _, err := writer.Write(message); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	// Quit
	if err := client.Quit(); err != nil {
		return fmt.Errorf("failed to quit SMTP session: %w", err)
	}

	return nil
}
