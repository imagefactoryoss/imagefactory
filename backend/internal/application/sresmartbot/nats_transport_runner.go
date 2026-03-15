package sresmartbot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	domainsresmartbot "github.com/srikarm/image-factory/internal/domain/sresmartbot"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type NATSTransportRunnerConfig struct {
	Enabled            bool
	Interval           time.Duration
	ReconnectThreshold int64
}

func StartNATSTransportSignalRunner(
	logger *zap.Logger,
	processHealthStore *runtimehealth.Store,
	provider interface {
		TransportStatus() messaging.NATSTransportStatus
	},
	sreSmartBotService *Service,
	cfg NATSTransportRunnerConfig,
) {
	if processHealthStore == nil {
		return
	}
	if cfg.Interval < 15*time.Second {
		cfg.Interval = 30 * time.Second
	}
	if cfg.ReconnectThreshold < 1 {
		cfg.ReconnectThreshold = 3
	}

	running := cfg.Enabled && provider != nil
	message := "nats transport signal runner initialized"
	switch {
	case !cfg.Enabled:
		message = "nats transport signal runner disabled"
	case provider == nil:
		message = "nats transport signal runner unavailable: NATS provider not configured"
	}

	processHealthStore.Upsert("nats_transport_signal_runner", runtimehealth.ProcessStatus{
		Enabled:      cfg.Enabled,
		Running:      running,
		LastActivity: time.Now().UTC(),
		Message:      message,
		Metrics: map[string]int64{
			"nats_transport_interval_seconds": int64(cfg.Interval / time.Second),
			"nats_reconnect_threshold":        cfg.ReconnectThreshold,
			"nats_transport_reconnects":       0,
			"nats_transport_disconnects":      0,
		},
	})

	if !cfg.Enabled || provider == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		previousIssueKeys := map[string]struct{}{}

		runTick := func() {
			now := time.Now().UTC()
			status := provider.TransportStatus()

			processHealthStore.Upsert("nats_transport_signal_runner", runtimehealth.ProcessStatus{
				Enabled:      true,
				Running:      true,
				LastActivity: now,
				Message:      fmt.Sprintf("nats transport status=%s reconnects=%d disconnects=%d", status.Status, status.Reconnects, status.Disconnects),
				Metrics: map[string]int64{
					"nats_transport_interval_seconds": int64(cfg.Interval / time.Second),
					"nats_reconnect_threshold":        cfg.ReconnectThreshold,
					"nats_transport_reconnects":       status.Reconnects,
					"nats_transport_disconnects":      status.Disconnects,
				},
			})

			currentIssueKeys := map[string]struct{}{}
			if strings.EqualFold(status.Status, "disconnected") || strings.EqualFold(status.Status, "closed") {
				issueKey := "nats_transport_unhealthy"
				currentIssueKeys[issueKey] = struct{}{}
				recordNATSTransportIssue(context.Background(), sreSmartBotService, logger, issueKey, "Messaging transport is disconnected", status, now, domainsresmartbot.IncidentSeverityWarning)
			}
			if status.Reconnects >= cfg.ReconnectThreshold {
				issueKey := "nats_reconnect_storm"
				currentIssueKeys[issueKey] = struct{}{}
				recordNATSTransportIssue(context.Background(), sreSmartBotService, logger, issueKey, "Messaging transport is reconnecting repeatedly", status, now, domainsresmartbot.IncidentSeverityWarning)
			}

			for issueKey := range previousIssueKeys {
				if _, stillPresent := currentIssueKeys[issueKey]; stillPresent {
					continue
				}
				if sreSmartBotService != nil {
					if err := sreSmartBotService.ResolveIncident(context.Background(), "messaging_transport:"+issueKey, now, "NATS transport recovered", map[string]interface{}{
						"issue_key": issueKey,
						"source":    "nats_transport_signal_runner",
					}); err != nil && logger != nil {
						logger.Warn("Failed to resolve NATS transport incident", zap.String("issue_key", issueKey), zap.Error(err))
					}
				}
			}
			previousIssueKeys = currentIssueKeys
		}

		runTick()
		for range ticker.C {
			runTick()
		}
	}()
}

func recordNATSTransportIssue(ctx context.Context, svc *Service, logger *zap.Logger, issueKey string, summary string, status messaging.NATSTransportStatus, now time.Time, severity domainsresmartbot.IncidentSeverity) {
	if svc == nil {
		return
	}
	payload := map[string]interface{}{
		"status":             status.Status,
		"connected_url":      status.ConnectedURL,
		"reconnects":         status.Reconnects,
		"disconnects":        status.Disconnects,
		"last_error":         status.LastError,
		"last_disconnect_at": status.LastDisconnectAt,
		"last_reconnect_at":  status.LastReconnectAt,
	}
	if err := svc.RecordObservation(ctx, SignalObservation{
		CorrelationKey: "messaging_transport:" + issueKey,
		Domain:         "runtime_services",
		IncidentType:   "messaging_transport_degraded",
		DisplayName:    "Messaging transport degraded",
		Summary:        summary,
		Source:         "nats_transport_signal_runner",
		Severity:       severity,
		Confidence:     domainsresmartbot.IncidentConfidenceHigh,
		OccurredAt:     now,
		Metadata: map[string]interface{}{
			"issue_key": issueKey,
			"status":    status.Status,
		},
		FindingTitle:   "NATS transport issue detected",
		FindingMessage: summary,
		SignalType:     "nats_transport",
		SignalKey:      issueKey,
		RawPayload:     payload,
	}); err != nil && logger != nil {
		logger.Warn("Failed to record NATS transport incident", zap.String("issue_key", issueKey), zap.Error(err))
	}
	_ = svc.AddEvidence(ctx, "messaging_transport:"+issueKey, "nats_transport_status", summary, payload, now)
	_ = svc.EnsureActionAttempt(ctx, "messaging_transport:"+issueKey, ActionAttemptSpec{
		ActionKey:     "review_messaging_transport",
		ActionClass:   "recommendation",
		TargetKind:    "message_bus",
		TargetRef:     "nats",
		Status:        "proposed",
		ActorType:     "system",
		ResultPayload: payload,
	}, now)
}
