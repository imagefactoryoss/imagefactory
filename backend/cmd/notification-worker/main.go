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
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	domainEmail "github.com/srikarm/image-factory/internal/domain/email"
	"github.com/srikarm/image-factory/internal/infrastructure/config"
	"github.com/srikarm/image-factory/internal/infrastructure/database"
	"github.com/srikarm/image-factory/internal/infrastructure/logger"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"github.com/subosito/gotenv"
	"go.uber.org/zap"
)

type notificationRequestedPayload struct {
	NotificationType string          `json:"notification_type"`
	Channel          string          `json:"channel"`
	TenantID         string          `json:"tenant_id"`
	To               string          `json:"to"`
	CC               string          `json:"cc"`
	From             string          `json:"from"`
	Subject          string          `json:"subject"`
	BodyText         string          `json:"body_text"`
	BodyHTML         string          `json:"body_html"`
	EmailType        string          `json:"email_type"`
	Priority         int             `json:"priority"`
	Metadata         json.RawMessage `json:"metadata"`
}

func main() {
	var envFile = flag.String("env", "", "Path to environment file (e.g., .env.development, .env.production)")
	var healthPort = flag.Int("health-port", 8083, "Port for health check endpoint")
	flag.Parse()

	if *envFile != "" {
		if !filepath.IsAbs(*envFile) {
			if absPath, err := filepath.Abs(*envFile); err == nil {
				*envFile = absPath
			}
		}

		if _, err := os.Stat(*envFile); err != nil {
			log.Fatalf("Environment file not found: %s (%v)", *envFile, err)
		}

		if err := gotenv.Load(*envFile); err != nil {
			log.Fatalf("Failed to load environment file %s: %v", *envFile, err)
		}

		log.Printf("Loaded environment from: %s", *envFile)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	logger, err := logger.NewLogger(cfg.Logger)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	if !cfg.Messaging.EnableNATS {
		logger.Fatal("Notification worker requires messaging.enable_nats=true")
	}

	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()
	if readiness, readinessErr := database.CheckSchemaReadiness(context.Background(), db); readinessErr != nil {
		logger.Warn("Database schema readiness check failed", zap.Error(readinessErr))
	} else {
		if readiness.CurrentSchema != cfg.Database.Schema {
			logger.Warn("Database schema mismatch detected",
				zap.String("configured_schema", cfg.Database.Schema),
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

	emailRepo := postgres.NewEmailRepository(db, logger)
	emailService := domainEmail.NewService(emailRepo, logger)

	natsBus, err := messaging.NewNATSBus(cfg.NATS, logger)
	if err != nil {
		logger.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer natsBus.Close()

	subject := cfg.NATS.Subject
	if subject == "" {
		subject = "notification.requested"
	} else {
		subject = fmt.Sprintf("%s.notification.requested", subject)
	}

	natsBus.Subscribe("notification.requested", func(ctx context.Context, event messaging.Event) {
		payload, decodeErr := decodeNotificationPayload(event)
		if decodeErr != nil {
			logger.Warn("Failed to decode notification payload", zap.Error(decodeErr))
			return
		}

		if payload.Channel != "" && payload.Channel != "email" {
			logger.Info("Skipping non-email notification",
				zap.String("channel", payload.Channel),
				zap.String("notification_type", payload.NotificationType))
			return
		}

		req, buildErr := buildEmailRequest(event, payload)
		if buildErr != nil {
			logger.Warn("Invalid notification payload", zap.Error(buildErr))
			return
		}

		if _, err := emailService.CreateEmail(ctx, req); err != nil {
			logger.Error("Failed to enqueue notification email", zap.Error(err),
				zap.String("notification_type", payload.NotificationType),
				zap.String("to", payload.To))
			return
		}

		logger.Info("Notification email enqueued",
			zap.String("notification_type", payload.NotificationType),
			zap.String("to", payload.To),
			zap.String("event_id", event.ID))
	})

	// Health check server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"notification-worker","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))
	})
	healthServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *healthPort),
		Handler: mux,
	}
	go func() {
		logger.Info("Starting notification worker health server", zap.String("addr", healthServer.Addr))
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start notification worker health server", zap.Error(err))
		}
	}()

	logger.Info("Notification worker started", zap.String("subject", subject), zap.Int("health_port", *healthPort))

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	logger.Info("Notification worker shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = healthServer.Shutdown(shutdownCtx)
}

func decodeNotificationPayload(event messaging.Event) (notificationRequestedPayload, error) {
	var payload notificationRequestedPayload
	raw, err := json.Marshal(event.Payload)
	if err != nil {
		return payload, fmt.Errorf("marshal payload: %w", err)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return payload, fmt.Errorf("unmarshal payload: %w", err)
	}
	if payload.TenantID == "" && event.TenantID != "" {
		payload.TenantID = event.TenantID
	}
	return payload, nil
}

func buildEmailRequest(event messaging.Event, payload notificationRequestedPayload) (domainEmail.CreateEmailRequest, error) {
	var tenantID uuid.UUID
	if payload.TenantID != "" {
		parsed, err := uuid.Parse(payload.TenantID)
		if err != nil {
			return domainEmail.CreateEmailRequest{}, fmt.Errorf("invalid tenant_id: %w", err)
		}
		tenantID = parsed
	}

	if payload.To == "" || payload.From == "" || payload.Subject == "" {
		return domainEmail.CreateEmailRequest{}, fmt.Errorf("missing required fields")
	}

	metadata := payload.Metadata
	if len(metadata) == 0 {
		meta := map[string]interface{}{
			"notification_type": payload.NotificationType,
			"event_id":          event.ID,
			"correlation_id":    event.CorrelationID,
			"request_id":        event.RequestID,
		}
		encoded, _ := json.Marshal(meta)
		metadata = encoded
	}

	return domainEmail.CreateEmailRequest{
		TenantID:  tenantID,
		ToEmail:   payload.To,
		CCEmail:   payload.CC,
		FromEmail: payload.From,
		Subject:   payload.Subject,
		BodyText:  payload.BodyText,
		BodyHTML:  payload.BodyHTML,
		EmailType: domainEmail.EmailType(payload.EmailType),
		Priority:  payload.Priority,
		Metadata:  metadata,
	}, nil
}
