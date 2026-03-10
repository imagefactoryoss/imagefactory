package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/subosito/gotenv"

	emailWorkerConfig "github.com/srikarm/image-factory/cmd/email-worker/config"
	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	"github.com/srikarm/image-factory/internal/domain/email"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/config"
	"github.com/srikarm/image-factory/internal/infrastructure/database"
	"github.com/srikarm/image-factory/internal/infrastructure/logger"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	var envFile = flag.String("env", "", "Path to environment file (e.g., .env.development, .env.production)")
	flag.Parse()

	// Load environment file if specified
	if *envFile != "" {
		// Convert to absolute path if relative
		if !filepath.IsAbs(*envFile) {
			if absPath, err := filepath.Abs(*envFile); err == nil {
				*envFile = absPath
			}
		}

		// Check if file exists
		if _, err := os.Stat(*envFile); err != nil {
			log.Fatalf("Environment file not found: %s (%v)", *envFile, err)
		}

		// Load the environment file
		if err := gotenv.Load(*envFile); err != nil {
			log.Fatalf("Failed to load environment file %s: %v", *envFile, err)
		}

		log.Printf("Loaded environment from: %s", *envFile)
	}

	// Load configuration
	cfg, err := emailWorkerConfig.LoadFromEnv()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	loggerConfig := config.LoggerConfig{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	}
	logger, err := logger.NewLogger(loggerConfig)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting Email Worker", zap.String("version", "1.0.0"))

	// Build database config from URL
	dbConfig := config.DatabaseConfig{
		URL:             cfg.DatabaseURL,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	// Initialize database
	db, err := database.NewConnection(dbConfig)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()
	expectedSchema := strings.TrimSpace(os.Getenv("IF_DATABASE_SCHEMA"))
	if expectedSchema == "" {
		expectedSchema = "public"
	}
	if readiness, readinessErr := database.CheckSchemaReadiness(context.Background(), db); readinessErr != nil {
		logger.Warn("Database schema readiness check failed", zap.Error(readinessErr))
	} else {
		if readiness.CurrentSchema != expectedSchema {
			logger.Warn("Database schema mismatch detected",
				zap.String("configured_schema", expectedSchema),
				zap.String("current_schema", readiness.CurrentSchema),
			)
		}
		logger.Info("Database schema readiness check completed",
			zap.String("current_schema", readiness.CurrentSchema),
			zap.Bool("schema_migrations_present", readiness.SchemaMigrations),
			zap.Bool("tenants_domain_column", readiness.TenantsDomainColumn),
			zap.Bool("tenants_domain_not_null", readiness.TenantsDomainNotNull),
		)
	}

	// Load SMTP configuration from database if needed
	if cfg.LoadSMTPFromDB {
		logger.Info("Loading SMTP configuration from database")
		systemConfigRepo := postgres.NewSystemConfigRepository(db, logger)

		// For now, use nil tenant ID to get universal configs
		// TODO: In a multi-tenant setup, this should be configurable
		smtpConfigs, err := systemConfigRepo.FindActiveByType(context.Background(), uuid.Nil, systemconfig.ConfigTypeSMTP)
		if err != nil {
			logger.Fatal("Failed to load SMTP configuration from database", zap.Error(err))
		}

		if len(smtpConfigs) > 0 {
			config := smtpConfigs[0]
			var smtpConfig map[string]interface{}
			if err := json.Unmarshal(config.ConfigValue(), &smtpConfig); err != nil {
				logger.Fatal("Failed to unmarshal SMTP configuration", zap.Error(err))
			}

			// Update config with database values
			if host, ok := smtpConfig["host"].(string); ok {
				cfg.SMTP.Host = host
			}
			if port, ok := smtpConfig["port"].(float64); ok {
				cfg.SMTP.Port = int(port)
			}
			if username, ok := smtpConfig["username"].(string); ok {
				cfg.SMTP.Username = username
			}
			if password, ok := smtpConfig["password"].(string); ok {
				cfg.SMTP.Password = password
			}
			if from, ok := smtpConfig["from"].(string); ok {
				cfg.SMTP.From = from
			}
			if startTLS, ok := smtpConfig["start_tls"].(bool); ok {
				cfg.SMTP.UseTLS = startTLS
			}
			if ssl, ok := smtpConfig["ssl"].(bool); ok && ssl {
				cfg.SMTP.UseTLS = true
			}

			logger.Info("Loaded SMTP configuration from database",
				zap.String("host", cfg.SMTP.Host),
				zap.Int("port", cfg.SMTP.Port),
				zap.String("from", cfg.SMTP.From))
		} else {
			logger.Warn("No SMTP configuration found in database, using defaults")
		}
	}

	// Log the final SMTP configuration being used
	logger.Info("SMTP configuration loaded",
		zap.String("source", func() string {
			if !cfg.LoadSMTPFromDB {
				return "environment_variables"
			}
			return "database"
		}()),
		zap.String("host", cfg.SMTP.Host),
		zap.Int("port", cfg.SMTP.Port),
		zap.String("from", cfg.SMTP.From),
		zap.Bool("use_tls", cfg.SMTP.UseTLS),
		zap.Bool("has_username", cfg.SMTP.Username != ""),
		zap.Bool("has_password", cfg.SMTP.Password != ""))

	// Initialize repositories and services
	emailRepo := postgres.NewEmailRepository(db, logger)
	emailService := email.NewServiceWithSMTP(emailRepo, logger, cfg.SMTP.Host, cfg.SMTP.Port)
	if bus := initEventBusFromEnv(logger); bus != nil {
		emailService.SetEventBus(bus)
		defer closeEventBus(bus)
	}

	// Start email processing worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start email processor in background
	go func() {
		ticker := time.NewTicker(30 * time.Second) // Process every 30 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Process pending emails
				if err := emailService.ProcessEmailQueue(ctx, 10); err != nil {
					logger.Error("Failed to process email queue", zap.Error(err))
				}

				// Process retry queue
				if err := emailService.ProcessRetryQueue(ctx, 5); err != nil {
					logger.Error("Failed to process retry queue", zap.Error(err))
				}
			}
		}
	}()

	// Create HTTP server for health checks
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"email-worker","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))
	})

	server := &http.Server{
		Addr:    ":8081", // Different port from main server and other services
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting Email Worker HTTP server", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start email worker server", zap.Error(err))
		}
	}()

	logger.Info("Email Worker started successfully")

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down Email Worker...")

	// Give outstanding requests a deadline for completion
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("Email Worker forced to shutdown", zap.Error(err))
	}

	logger.Info("Email Worker exited")
}

func initEventBusFromEnv(logger *zap.Logger) messaging.EventBus {
	if !parseBoolEnv("IF_MESSAGING_ENABLE_NATS") {
		return nil
	}

	natsConfig := config.NATSConfig{
		URLs:          parseCSVEnv("IF_NATS_URLS", []string{"nats://localhost:4222"}),
		MaxReconnects: parseIntEnv("IF_NATS_MAX_RECONNECTS", 3),
		ReconnectWait: parseDurationEnv("IF_NATS_RECONNECT_WAIT", 2*time.Second),
		Timeout:       parseDurationEnv("IF_NATS_TIMEOUT", 10*time.Second),
		Subject:       getEnvOrDefault("IF_NATS_SUBJECT", "image-factory"),
		ClusterID:     getEnvOrDefault("IF_NATS_CLUSTER_ID", "image-factory-cluster"),
		ClientID:      getEnvOrDefault("IF_NATS_CLIENT_ID", "image-factory-email-worker"),
	}

	bus, err := messaging.NewNATSBus(natsConfig, logger)
	if err != nil {
		logger.Warn("Failed to initialize NATS for email worker", zap.Error(err))
		return nil
	}

	logger.Info("Email worker publishing notification outcomes", zap.Strings("nats_urls", natsConfig.URLs))
	return bus
}

func closeEventBus(bus messaging.EventBus) {
	if closable, ok := bus.(interface{ Close() }); ok {
		closable.Close()
	}
}

func parseBoolEnv(key string) bool {
	val := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return val == "true" || val == "1" || val == "yes" || val == "y"
}

func parseCSVEnv(key string, defaultValue []string) []string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return defaultValue
	}
	parts := strings.Split(val, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return defaultValue
	}
	return out
}

func parseIntEnv(key string, defaultValue int) int {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return defaultValue
	}
	if parsed, err := strconv.Atoi(val); err == nil {
		return parsed
	}
	return defaultValue
}

func parseDurationEnv(key string, defaultValue time.Duration) time.Duration {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return defaultValue
	}
	if parsed, err := time.ParseDuration(val); err == nil {
		return parsed
	}
	return defaultValue
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return defaultValue
}
