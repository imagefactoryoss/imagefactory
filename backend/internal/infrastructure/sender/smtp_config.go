package sender

import (
	"fmt"
	"regexp"
	"time"
)

// Email validation regex pattern
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// SMTPConfig holds SMTP server configuration
type SMTPConfig struct {
	// Host is the SMTP server hostname (e.g., "smtp.gmail.com")
	Host string `json:"host"`

	// Port is the SMTP server port (e.g., 587 for STARTTLS, 465 for SSL)
	Port int `json:"port"`

	// Username for SMTP authentication
	Username string `json:"username"`

	// Password for SMTP authentication
	Password string `json:"password"`

	// From is the default sender email address
	From string `json:"from"`

	// UseTLS enables TLS/STARTTLS
	UseTLS bool `json:"use_tls"`

	// Timeout is the connection timeout
	Timeout time.Duration `json:"timeout"`
}

// DefaultSMTPConfig returns SMTPConfig with sensible defaults
func DefaultSMTPConfig() SMTPConfig {
	return SMTPConfig{
		Port:    587,              // STARTTLS port
		UseTLS:  true,             // Enable TLS by default
		Timeout: 30 * time.Second, // 30 second timeout
	}
}

// Validate checks if the configuration is valid
func (c SMTPConfig) Validate() error {
	// Validate host
	if c.Host == "" {
		return fmt.Errorf("smtp host is required")
	}

	// Validate port
	if c.Port <= 0 {
		return fmt.Errorf("smtp port must be positive, got %d", c.Port)
	}
	if c.Port > 65535 {
		return fmt.Errorf("smtp port must be <= 65535, got %d", c.Port)
	}

	// Validate username
	if c.Username == "" {
		return fmt.Errorf("smtp username is required")
	}

	// Validate password
	if c.Password == "" {
		return fmt.Errorf("smtp password is required")
	}

	// Validate from address
	if c.From == "" {
		return fmt.Errorf("smtp from address is required")
	}
	if !emailRegex.MatchString(c.From) {
		return fmt.Errorf("smtp from address is invalid: %s", c.From)
	}

	// Validate timeout (allow zero for default, but not negative)
	if c.Timeout < 0 {
		return fmt.Errorf("smtp timeout must be non-negative, got %v", c.Timeout)
	}

	return nil
}
