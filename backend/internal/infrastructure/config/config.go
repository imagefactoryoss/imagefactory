package config

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Logger     LoggerConfig     `mapstructure:"logger"`
	NATS       NATSConfig       `mapstructure:"nats"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Auth       AuthConfig       `mapstructure:"auth"`
	Build      BuildConfig      `mapstructure:"build"`
	SMTP       SMTPConfig       `mapstructure:"smtp"`
	Frontend   FrontendConfig   `mapstructure:"frontend"`
	Messaging  MessagingConfig  `mapstructure:"messaging"`
	Dispatcher DispatcherConfig `mapstructure:"dispatcher"`
	Workflow   WorkflowConfig   `mapstructure:"workflow"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Host         string        `mapstructure:"host"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	Version      string        `mapstructure:"version"`
	Environment  string        `mapstructure:"environment"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Name            string        `mapstructure:"name"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	Schema          string        `mapstructure:"schema"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	URL             string        `mapstructure:"url"`
}

// LoggerConfig represents logger configuration
type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
}

// MessagingConfig represents event bus configuration
type MessagingConfig struct {
	SchemaVersion  string `mapstructure:"schema_version"`
	ValidateEvents bool   `mapstructure:"validate_events"`
	EnableNATS     bool   `mapstructure:"enable_nats"`
}

// NATSConfig represents NATS configuration
type NATSConfig struct {
	URLs          []string      `mapstructure:"urls"`
	MaxReconnects int           `mapstructure:"max_reconnects"`
	ReconnectWait time.Duration `mapstructure:"reconnect_wait"`
	Timeout       time.Duration `mapstructure:"timeout"`
	Subject       string        `mapstructure:"subject"`
	ClusterID     string        `mapstructure:"cluster_id"`
	ClientID      string        `mapstructure:"client_id"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	JWTSecret           string        `mapstructure:"jwt_secret"`
	JWTExpiration       time.Duration `mapstructure:"jwt_expiration"`
	RefreshTokenExp     time.Duration `mapstructure:"refresh_token_exp"`
	LDAPEnabled         bool          `mapstructure:"ldap_enabled"`
	LDAPServer          string        `mapstructure:"ldap_server"`
	LDAPPort            int           `mapstructure:"ldap_port"`
	LDAPBaseDN          string        `mapstructure:"ldap_base_dn"`
	LDAPBindDN          string        `mapstructure:"ldap_bind_dn"`
	LDAPBindPassword    string        `mapstructure:"ldap_bind_password"`
	LDAPUserSearchBase  string        `mapstructure:"ldap_user_search_base"`
	LDAPGroupSearchBase string        `mapstructure:"ldap_group_search_base"`
	LDAPUseTLS          bool          `mapstructure:"ldap_use_tls"`
	LDAPStartTLS        bool          `mapstructure:"ldap_start_tls"`
	SAMLEnabled         bool          `mapstructure:"saml_enabled"`
	SAMLMetadataURL     string        `mapstructure:"saml_metadata_url"`
}

// BuildConfig represents build configuration
type BuildConfig struct {
	DefaultTimeout    time.Duration `mapstructure:"default_timeout"`
	MaxConcurrentJobs int           `mapstructure:"max_concurrent_jobs"`
	WorkerPoolSize    int           `mapstructure:"worker_pool_size"`
	StoragePath       string        `mapstructure:"storage_path"`
	RegistryURL       string        `mapstructure:"registry_url"`
	TektonEnabled     bool          `mapstructure:"tekton_enabled"`
	TektonKubeconfig  string        `mapstructure:"tekton_kubeconfig"`
}

// DispatcherConfig represents build dispatcher configuration
type DispatcherConfig struct {
	Enabled            bool          `mapstructure:"enabled"`
	PollInterval       time.Duration `mapstructure:"poll_interval"`
	MaxDispatchPerTick int           `mapstructure:"max_dispatch_per_tick"`
	MaxRetries         int           `mapstructure:"max_retries"`
	RetryBackoff       time.Duration `mapstructure:"retry_backoff"`
	RetryBackoffMax    time.Duration `mapstructure:"retry_backoff_max"`
}

// WorkflowConfig represents workflow orchestrator configuration
type WorkflowConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	PollInterval    time.Duration `mapstructure:"poll_interval"`
	MaxStepsPerTick int           `mapstructure:"max_steps_per_tick"`
}

// SMTPConfig represents SMTP configuration
type SMTPConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	UseTLS    bool   `mapstructure:"use_tls"`
	FromEmail string `mapstructure:"from_email"`
}

// FrontendConfig represents frontend configuration
type FrontendConfig struct {
	BaseURL      string `mapstructure:"base_url"`
	DashboardURL string `mapstructure:"dashboard_url"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/image-factory")

	// Set default values
	setDefaults()

	// Enable environment variable support
	viper.AutomaticEnv()
	viper.SetEnvPrefix("IF") // Image Factory prefix
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file if available
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Viper already handles environment variable binding with IF prefix and automatic env key replacement
	// No need for manual parsing - viper.AutomaticEnv() with SetEnvPrefix and SetEnvKeyReplacer handles this
	// The manual parsing below is redundant and can be removed

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Generate database URL if not provided
	if config.Database.URL == "" {
		config.Database.URL = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			config.Database.User,
			config.Database.Password,
			config.Database.Host,
			config.Database.Port,
			config.Database.Name,
			config.Database.SSLMode,
		)
	}
	config.Database.Schema = normalizeDatabaseSchema(config.Database.Schema)
	normalizedDatabaseURL, err := ensureDatabaseURLSearchPath(config.Database.URL, config.Database.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize database URL: %w", err)
	}
	config.Database.URL = normalizedDatabaseURL

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.version", "1.0.0")
	viper.SetDefault("server.environment", "development")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.name", "image_factory")
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.schema", "public")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "5m")

	// Logger defaults
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "json")
	viper.SetDefault("logger.output_path", "stdout")

	// Messaging defaults
	viper.SetDefault("messaging.schema_version", "v1")
	viper.SetDefault("messaging.validate_events", false)
	viper.SetDefault("messaging.enable_nats", false)

	// NATS defaults
	viper.SetDefault("nats.urls", []string{"nats://localhost:4222"})
	viper.SetDefault("nats.max_reconnects", 3)
	viper.SetDefault("nats.reconnect_wait", "2s")
	viper.SetDefault("nats.timeout", "10s")
	viper.SetDefault("nats.subject", "image-factory")
	viper.SetDefault("nats.cluster_id", "image-factory-cluster")
	viper.SetDefault("nats.client_id", "image-factory-server")

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.idle_timeout", "5m")
	viper.SetDefault("redis.read_timeout", "3s")
	viper.SetDefault("redis.write_timeout", "3s")

	// Auth defaults
	viper.SetDefault("auth.jwt_secret", "")
	viper.SetDefault("auth.jwt_expiration", "24h")
	viper.SetDefault("auth.refresh_token_exp", "168h") // 7 days
	viper.SetDefault("auth.ldap_enabled", false)
	viper.SetDefault("auth.ldap_server", "localhost")
	viper.SetDefault("auth.ldap_port", 389)
	viper.SetDefault("auth.ldap_base_dn", "dc=example,dc=com")
	viper.SetDefault("auth.ldap_bind_dn", "cn=admin,dc=example,dc=com")
	viper.SetDefault("auth.ldap_bind_password", "")
	viper.SetDefault("auth.ldap_user_search_base", "ou=users,dc=example,dc=com")
	viper.SetDefault("auth.ldap_group_search_base", "ou=groups,dc=example,dc=com")
	viper.SetDefault("auth.ldap_use_tls", false)
	viper.SetDefault("auth.ldap_start_tls", false)
	viper.SetDefault("auth.saml_enabled", false)

	// Build defaults
	viper.SetDefault("build.default_timeout", "30m")
	viper.SetDefault("build.max_concurrent_jobs", 10)
	viper.SetDefault("build.worker_pool_size", 5)
	viper.SetDefault("build.storage_path", "/tmp/image-factory")
	viper.SetDefault("build.registry_url", "localhost:5000")
	viper.SetDefault("build.tekton_enabled", false)
	viper.SetDefault("build.tekton_kubeconfig", "")

	// SMTP defaults
	viper.SetDefault("smtp.host", "localhost")
	viper.SetDefault("smtp.port", 1025)
	viper.SetDefault("smtp.username", "")
	viper.SetDefault("smtp.password", "")
	viper.SetDefault("smtp.use_tls", false)
	viper.SetDefault("smtp.from_email", "noreply@image-factory.com")

	// Frontend defaults
	viper.SetDefault("frontend.base_url", "http://localhost:3000")
	viper.SetDefault("frontend.dashboard_url", "https://dashboard.imgfactory.com")

	// Dispatcher defaults
	viper.SetDefault("dispatcher.enabled", true)
	viper.SetDefault("dispatcher.poll_interval", "3s")
	viper.SetDefault("dispatcher.max_dispatch_per_tick", 1)
	viper.SetDefault("dispatcher.max_retries", 3)
	viper.SetDefault("dispatcher.retry_backoff", "5s")
	viper.SetDefault("dispatcher.retry_backoff_max", "1m")

	// Workflow defaults
	viper.SetDefault("workflow.enabled", true)
	viper.SetDefault("workflow.poll_interval", "3s")
	viper.SetDefault("workflow.max_steps_per_tick", 1)
}

// Validate validates the configuration for required fields and reasonable values
func (c *Config) Validate() error {
	// JWT secret is required
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required (set IF_AUTH_JWT_SECRET environment variable)")
	}

	// Database configuration required
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if !databaseSchemaIdentifierPattern.MatchString(normalizeDatabaseSchema(c.Database.Schema)) {
		return fmt.Errorf("database schema must be a valid SQL identifier")
	}

	// Server port must be > 0
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535")
	}

	// Database port must be valid
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535")
	}

	// Redis port must be valid if configured
	if c.Redis.Port < 0 || c.Redis.Port > 65535 {
		return fmt.Errorf("redis port must be between 0 and 65535")
	}

	// Database pool sizes must be positive
	if c.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("database max_open_conns must be positive")
	}
	if c.Database.MaxIdleConns < 0 {
		return fmt.Errorf("database max_idle_conns must be non-negative")
	}

	// LDAP validation when enabled
	if c.Auth.LDAPEnabled {
		if c.Auth.LDAPServer == "" {
			return fmt.Errorf("LDAP server is required when LDAP is enabled")
		}
		if c.Auth.LDAPPort <= 0 || c.Auth.LDAPPort > 65535 {
			return fmt.Errorf("LDAP port must be between 1 and 65535 when LDAP is enabled")
		}
		if c.Auth.LDAPBaseDN == "" {
			return fmt.Errorf("LDAP base DN is required when LDAP is enabled")
		}
	}

	// Build configuration validation
	if c.Build.MaxConcurrentJobs <= 0 {
		return fmt.Errorf("build max_concurrent_jobs must be positive")
	}
	if c.Build.WorkerPoolSize <= 0 {
		return fmt.Errorf("build worker_pool_size must be positive")
	}

	if c.Dispatcher.MaxDispatchPerTick <= 0 {
		return fmt.Errorf("dispatcher max_dispatch_per_tick must be positive")
	}
	if c.Dispatcher.MaxRetries < 0 {
		return fmt.Errorf("dispatcher max_retries must be non-negative")
	}
	if c.Workflow.MaxStepsPerTick <= 0 {
		return fmt.Errorf("workflow max_steps_per_tick must be positive")
	}

	return nil
}

var databaseSchemaIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func normalizeDatabaseSchema(schema string) string {
	trimmed := strings.TrimSpace(schema)
	if trimmed == "" {
		return "public"
	}
	return trimmed
}

func ensureDatabaseURLSearchPath(databaseURL, schema string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid database URL: %w", err)
	}
	query := parsed.Query()
	if strings.TrimSpace(query.Get("search_path")) != "" {
		return parsed.String(), nil
	}
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
