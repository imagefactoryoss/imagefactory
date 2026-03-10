package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/srikarm/image-factory/internal/infrastructure/worker"
)

// Config holds all configuration for the email worker binary
type Config struct {
	// Database
	DatabaseURL string

	// Worker Configuration
	Worker worker.WorkerConfig

	// SMTP Configuration
	SMTP SMTPConfig

	// Flag to indicate if SMTP config should be loaded from database
	LoadSMTPFromDB bool

	// Template Configuration
	Template TemplateConfig

	// Health Check
	HealthCheck HealthCheckConfig

	// Logging
	Logging LoggingConfig
}

// SMTPConfig holds SMTP server configuration
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
	Timeout  time.Duration
}

// TemplateConfig holds template cache configuration
type TemplateConfig struct {
	CacheEnabled bool
	CacheTTL     time.Duration
}

// HealthCheckConfig holds health check endpoint configuration
type HealthCheckConfig struct {
	Port int
	Path string
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	config := &Config{
		// Worker defaults (will be overridden by environment)
		Worker: worker.DefaultWorkerConfig(),

		// SMTP defaults
		SMTP: SMTPConfig{
			UseTLS:  true,
			Timeout: 30 * time.Second,
		},

		// Template defaults
		Template: TemplateConfig{
			CacheEnabled: true,
			CacheTTL:     5 * time.Minute,
		},

		// Health check defaults
		HealthCheck: HealthCheckConfig{
			Port: 8081,
			Path: "/health",
		},

		// Logging defaults
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	// Required: Database URL
	config.DatabaseURL = os.Getenv("DATABASE_URL")
	if config.DatabaseURL == "" {
		// Try building from IF_DATABASE_* environment variables (like main server)
		dbHost := getEnvOrDefault("IF_DATABASE_HOST", "localhost")
		dbPort := getEnvOrDefault("IF_DATABASE_PORT", "5432")
		dbName := getEnvOrDefault("IF_DATABASE_NAME", "image_factory_dev")
		dbUser := getEnvOrDefault("IF_DATABASE_USER", "postgres")
		dbPassword := getEnvOrDefault("IF_DATABASE_PASSWORD", "postgres")
		dbSSLMode := getEnvOrDefault("IF_DATABASE_SSL_MODE", "disable")
		dbSchema := getEnvOrDefault("IF_DATABASE_SCHEMA", "public")

		config.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode)
		config.DatabaseURL = ensureSearchPath(config.DatabaseURL, dbSchema)
	}

	// Check if SMTP environment variables are provided
	smtpHost := os.Getenv("EMAIL_SMTP_HOST")
	if smtpHost == "" {
		smtpHost = os.Getenv("IF_EMAIL_SMTP_HOST")
	}
	if smtpHost == "" {
		smtpHost = os.Getenv("IF_SMTP_HOST")
	}

	smtpFrom := os.Getenv("EMAIL_FROM_ADDRESS")
	if smtpFrom == "" {
		smtpFrom = os.Getenv("IF_EMAIL_FROM_ADDRESS")
	}
	if smtpFrom == "" {
		smtpFrom = os.Getenv("IF_SMTP_FROM_EMAIL")
	}

	// If SMTP environment variables are provided, use them
	if smtpHost != "" && smtpFrom != "" {
		config.SMTP.Host = smtpHost
		config.SMTP.From = smtpFrom
		config.LoadSMTPFromDB = false

		// Load other SMTP settings from environment
		if val := os.Getenv("EMAIL_SMTP_PORT"); val != "" {
			port, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid EMAIL_SMTP_PORT: %w", err)
			}
			config.SMTP.Port = port
		} else if val := os.Getenv("IF_EMAIL_SMTP_PORT"); val != "" {
			port, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid IF_EMAIL_SMTP_PORT: %w", err)
			}
			config.SMTP.Port = port
		} else if val := os.Getenv("IF_SMTP_PORT"); val != "" {
			port, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid IF_SMTP_PORT: %w", err)
			}
			config.SMTP.Port = port
		} else {
			config.SMTP.Port = 1025 // Default port
		}

		config.SMTP.Username = os.Getenv("EMAIL_SMTP_USERNAME")
		if config.SMTP.Username == "" {
			config.SMTP.Username = os.Getenv("IF_EMAIL_SMTP_USERNAME")
		}
		if config.SMTP.Username == "" {
			config.SMTP.Username = os.Getenv("IF_SMTP_USERNAME")
		}

		config.SMTP.Password = os.Getenv("EMAIL_SMTP_PASSWORD")
		if config.SMTP.Password == "" {
			config.SMTP.Password = os.Getenv("IF_EMAIL_SMTP_PASSWORD")
		}
		if config.SMTP.Password == "" {
			config.SMTP.Password = os.Getenv("IF_SMTP_PASSWORD")
		}

		if val := os.Getenv("EMAIL_SMTP_TLS_ENABLED"); val != "" {
			tls, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("invalid EMAIL_SMTP_TLS_ENABLED: %w", err)
			}
			config.SMTP.UseTLS = tls
		} else if val := os.Getenv("IF_EMAIL_SMTP_TLS_ENABLED"); val != "" {
			tls, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("invalid IF_EMAIL_SMTP_TLS_ENABLED: %w", err)
			}
			config.SMTP.UseTLS = tls
		} else if val := os.Getenv("IF_SMTP_USE_TLS"); val != "" {
			tls, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("invalid IF_SMTP_USE_TLS: %w", err)
			}
			config.SMTP.UseTLS = tls
		}
	} else {
		// No SMTP environment variables provided, load from database
		config.LoadSMTPFromDB = true
		// Set defaults that will be overridden by database config
		config.SMTP.Host = "localhost"
		config.SMTP.Port = 1025
		config.SMTP.From = "noreply@image-factory.com"
		config.SMTP.UseTLS = false
	}

	// Optional: Worker configuration (with defaults)
	if val := os.Getenv("EMAIL_QUEUE_WORKER_COUNT"); val != "" {
		count, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid EMAIL_QUEUE_WORKER_COUNT: %w", err)
		}
		config.Worker.WorkerCount = count
	}

	if val := os.Getenv("EMAIL_QUEUE_POLL_INTERVAL"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("invalid EMAIL_QUEUE_POLL_INTERVAL: %w", err)
		}
		config.Worker.PollInterval = duration
	}

	if val := os.Getenv("EMAIL_QUEUE_MAX_RETRIES"); val != "" {
		retries, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid EMAIL_QUEUE_MAX_RETRIES: %w", err)
		}
		config.Worker.MaxRetries = retries
	}

	if val := os.Getenv("EMAIL_QUEUE_RETRY_BASE_DELAY"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("invalid EMAIL_QUEUE_RETRY_BASE_DELAY: %w", err)
		}
		config.Worker.RetryBaseDelay = duration
	}

	if val := os.Getenv("EMAIL_QUEUE_RETRY_MAX_DELAY"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("invalid EMAIL_QUEUE_RETRY_MAX_DELAY: %w", err)
		}
		config.Worker.RetryMaxDelay = duration
	}

	if val := os.Getenv("EMAIL_QUEUE_SHUTDOWN_TIMEOUT"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("invalid EMAIL_QUEUE_SHUTDOWN_TIMEOUT: %w", err)
		}
		config.Worker.ShutdownTimeout = duration
	}

	if val := os.Getenv("EMAIL_SMTP_TIMEOUT"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("invalid EMAIL_SMTP_TIMEOUT: %w", err)
		}
		config.SMTP.Timeout = duration
	}

	// Optional: Template configuration
	if val := os.Getenv("EMAIL_TEMPLATE_CACHE_ENABLED"); val != "" {
		enabled, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("invalid EMAIL_TEMPLATE_CACHE_ENABLED: %w", err)
		}
		config.Template.CacheEnabled = enabled
	}

	if val := os.Getenv("EMAIL_TEMPLATE_CACHE_TTL"); val != "" {
		duration, err := time.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("invalid EMAIL_TEMPLATE_CACHE_TTL: %w", err)
		}
		config.Template.CacheTTL = duration
	}

	// Optional: Health check configuration
	if val := os.Getenv("HEALTH_CHECK_PORT"); val != "" {
		port, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid HEALTH_CHECK_PORT: %w", err)
		}
		config.HealthCheck.Port = port
	}

	if val := os.Getenv("HEALTH_CHECK_PATH"); val != "" {
		config.HealthCheck.Path = val
	}

	// Optional: Logging configuration
	if val := os.Getenv("LOG_LEVEL"); val != "" {
		config.Logging.Level = val
	}

	if val := os.Getenv("LOG_FORMAT"); val != "" {
		config.Logging.Format = val
	} else if val := os.Getenv("LOG_JSON"); val != "" {
		// Backward compatibility
		logJSON, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("invalid LOG_JSON: %w", err)
		}
		if logJSON {
			config.Logging.Format = "json"
		} else {
			config.Logging.Format = "console"
		}
	}

	return config, nil
}

func ensureSearchPath(databaseURL, schema string) string {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return databaseURL
	}
	query := parsed.Query()
	if strings.TrimSpace(query.Get("search_path")) == "" {
		query.Set("search_path", strings.TrimSpace(schema))
		parsed.RawQuery = query.Encode()
	}
	return parsed.String()
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	// Validate worker configuration
	if err := c.Worker.Validate(); err != nil {
		return fmt.Errorf("worker config invalid: %w", err)
	}

	// Validate SMTP configuration
	if err := c.SMTP.Validate(); err != nil {
		return fmt.Errorf("SMTP config invalid: %w", err)
	}

	// Validate template configuration
	if err := c.Template.Validate(); err != nil {
		return fmt.Errorf("template config invalid: %w", err)
	}

	// Validate health check configuration
	if err := c.HealthCheck.Validate(); err != nil {
		return fmt.Errorf("health check config invalid: %w", err)
	}

	// Validate logging configuration
	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging config invalid: %w", err)
	}

	return nil
}

// Validate validates SMTP configuration
func (c *SMTPConfig) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("SMTP port must be 1-65535, got %d", c.Port)
	}

	if c.Timeout < 1*time.Second {
		return fmt.Errorf("SMTP timeout must be at least 1s, got %v", c.Timeout)
	}

	if c.Timeout > 5*time.Minute {
		return fmt.Errorf("SMTP timeout must not exceed 5 minutes, got %v", c.Timeout)
	}

	return nil
}

// Validate validates template configuration
func (c *TemplateConfig) Validate() error {
	if c.CacheTTL < 10*time.Second {
		return fmt.Errorf("template cache TTL must be at least 10s, got %v", c.CacheTTL)
	}

	if c.CacheTTL > 1*time.Hour {
		return fmt.Errorf("template cache TTL must not exceed 1 hour, got %v", c.CacheTTL)
	}

	return nil
}

// Validate validates health check configuration
func (c *HealthCheckConfig) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("health check port must be 1-65535, got %d", c.Port)
	}

	if c.Path == "" {
		return fmt.Errorf("health check path cannot be empty")
	}

	if c.Path[0] != '/' {
		return fmt.Errorf("health check path must start with '/', got %s", c.Path)
	}

	return nil
}

// Validate validates logging configuration
func (c *LoggingConfig) Validate() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLevels[c.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Level)
	}

	return nil
}

// getEnvOrDefault gets an environment variable value or returns a default
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
