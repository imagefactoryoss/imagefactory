package sresmartbot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	domainsystemconfig "github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/logdetector"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type LogQueryClient interface {
	QueryRange(ctx context.Context, query string, start time.Time, end time.Time, limit int) (*logdetector.QueryResult, error)
}

type LogDetectorRule struct {
	Name         string
	Query        string
	Threshold    int
	Domain       string
	IncidentType string
	DisplayName  string
	Summary      string
	Severity     string
	Confidence   string
	SignalKey    string
}

type LogDetectorRunnerConfig struct {
	Enabled       bool
	BaseURL       string
	Interval      time.Duration
	Timeout       time.Duration
	Lookback      time.Duration
	MaxMatches    int
	EventSource   string
	SchemaVersion string
}

type LogDetectorPolicyReader interface {
	GetRobotSREPolicyConfig(ctx context.Context, tenantID *uuid.UUID) (*domainsystemconfig.RobotSREPolicyConfig, error)
}

type logFindingPublisher struct {
	bus           messaging.EventBus
	eventSource   string
	schemaVersion string
}

func newLogFindingPublisher(bus messaging.EventBus, eventSource string, schemaVersion string) *logFindingPublisher {
	return &logFindingPublisher{
		bus:           bus,
		eventSource:   eventSource,
		schemaVersion: schemaVersion,
	}
}

func (p *logFindingPublisher) PublishObserved(ctx context.Context, rule LogDetectorRule, matches []logdetector.QueryMatch, observedAt time.Time, lookback time.Duration) error {
	if p == nil || p.bus == nil {
		return nil
	}
	return p.bus.Publish(ctx, messaging.Event{
		Type:          messaging.EventTypeSREDetectorFindingObserved,
		Source:        defaultString(p.eventSource, "sre_log_detector"),
		OccurredAt:    observedAt,
		SchemaVersion: p.schemaVersion,
		CorrelationID: "log_detector:" + strings.TrimSpace(rule.Name),
		Payload: map[string]interface{}{
			"correlation_key": "log_detector:" + strings.TrimSpace(rule.Name),
			"domain":          strings.TrimSpace(rule.Domain),
			"incident_type":   strings.TrimSpace(rule.IncidentType),
			"display_name":    strings.TrimSpace(rule.DisplayName),
			"summary":         strings.TrimSpace(rule.Summary),
			"source":          "sre_log_detector",
			"severity":        strings.TrimSpace(rule.Severity),
			"confidence":      strings.TrimSpace(rule.Confidence),
			"finding_title":   strings.TrimSpace(rule.DisplayName),
			"finding_message": fmt.Sprintf("%s (%d matches in the last %s)", strings.TrimSpace(rule.Summary), len(matches), lookback),
			"signal_type":     "log_signature",
			"signal_key":      defaultString(rule.SignalKey, rule.Name),
			"metadata": map[string]interface{}{
				"detector_name": "sre_log_detector",
				"rule_name":     rule.Name,
				"query_ref":     rule.Query,
				"match_count":   len(matches),
			},
			"raw_payload": map[string]interface{}{
				"query": rule.Query,
			},
			"evidence_type":    "loki_log_match_window",
			"evidence_summary": fmt.Sprintf("%d matching log lines detected", len(matches)),
			"evidence_payload": map[string]interface{}{
				"match_count": len(matches),
				"matches":     compactMatches(matches),
			},
			"observed_at": observedAt.Format(time.RFC3339),
		},
	})
}

func (p *logFindingPublisher) PublishRecovered(ctx context.Context, rule LogDetectorRule, observedAt time.Time, recoveredAt time.Time) error {
	if p == nil || p.bus == nil {
		return nil
	}
	return p.bus.Publish(ctx, messaging.Event{
		Type:          messaging.EventTypeSREDetectorFindingRecovered,
		Source:        defaultString(p.eventSource, "sre_log_detector"),
		OccurredAt:    recoveredAt,
		SchemaVersion: p.schemaVersion,
		CorrelationID: "log_detector:" + strings.TrimSpace(rule.Name),
		Payload: map[string]interface{}{
			"correlation_key": "log_detector:" + strings.TrimSpace(rule.Name),
			"domain":          strings.TrimSpace(rule.Domain),
			"incident_type":   strings.TrimSpace(rule.IncidentType),
			"summary":         fmt.Sprintf("%s recovered", strings.TrimSpace(rule.DisplayName)),
			"source":          "sre_log_detector",
			"resolved_at":     recoveredAt.Format(time.RFC3339),
			"metadata": map[string]interface{}{
				"detector_name": "sre_log_detector",
				"rule_name":     rule.Name,
				"previous_seen": observedAt.Format(time.RFC3339),
			},
		},
	})
}

type activeRuleState struct {
	lastObservedAt time.Time
}

func StartLogDetectorRunner(
	logger *zap.Logger,
	processHealthStore *runtimehealth.Store,
	queryClient LogQueryClient,
	bus messaging.EventBus,
	policyReader LogDetectorPolicyReader,
	cfg LogDetectorRunnerConfig,
) {
	if cfg.Interval < time.Minute {
		cfg.Interval = 2 * time.Minute
	}
	if cfg.Timeout < 5*time.Second {
		cfg.Timeout = 15 * time.Second
	}
	if cfg.Lookback < time.Minute {
		cfg.Lookback = 5 * time.Minute
	}
	if cfg.MaxMatches <= 0 {
		cfg.MaxMatches = 5
	}

	running := cfg.Enabled && queryClient != nil && bus != nil && strings.TrimSpace(cfg.BaseURL) != ""
	message := "sre log detector initialized"
	switch {
	case !cfg.Enabled:
		message = "sre log detector disabled"
	case strings.TrimSpace(cfg.BaseURL) == "":
		message = "sre log detector unavailable: Loki base URL not configured"
	case queryClient == nil || bus == nil:
		message = "sre log detector unavailable: dependencies not configured"
	}

	processHealthStore.Upsert("sre_log_detector", runtimehealth.ProcessStatus{
		Enabled:      cfg.Enabled,
		Running:      running,
		LastActivity: time.Now().UTC(),
		Message:      message,
		Metrics: map[string]int64{
			"sre_log_detector_ticks_total":      0,
			"sre_log_detector_failures_total":   0,
			"sre_log_detector_findings_total":   0,
			"sre_log_detector_active_rules":     0,
			"sre_log_detector_interval_seconds": int64(cfg.Interval / time.Second),
			"sre_log_detector_lookback_seconds": int64(cfg.Lookback / time.Second),
			"sre_log_detector_max_matches":      int64(cfg.MaxMatches),
		},
	})
	if !running {
		return
	}

	publisher := newLogFindingPublisher(bus, cfg.EventSource, cfg.SchemaVersion)
	initialRules := mergeLogDetectorRules(defaultLogDetectorRules(), loadCustomLogDetectorRules(policyReader))
	logger.Info("Background process starting",
		zap.String("component", "sre_log_detector"),
		zap.Duration("interval", cfg.Interval),
		zap.Duration("timeout", cfg.Timeout),
		zap.Duration("lookback", cfg.Lookback),
		zap.Int("rules", len(initialRules)),
	)

	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		active := map[string]activeRuleState{}
		var ticksTotal int64
		var failuresTotal int64
		var findingsTotal int64

		runTick := func() {
			ticksTotal++
			now := time.Now().UTC()
			start := now.Add(-cfg.Lookback)
			rules := mergeLogDetectorRules(defaultLogDetectorRules(), loadCustomLogDetectorRules(policyReader))
			activeCount, findingsDelta, failuresDelta := runLogDetectorTick(logger, queryClient, publisher, rules, active, start, now, cfg.Lookback, cfg.Timeout, cfg.MaxMatches)
			findingsTotal += findingsDelta
			failuresTotal += failuresDelta

			processHealthStore.Upsert("sre_log_detector", runtimehealth.ProcessStatus{
				Enabled:      true,
				Running:      true,
				LastActivity: now,
				Message:      fmt.Sprintf("evaluated %d rules; active=%d", len(rules), activeCount),
				Metrics: map[string]int64{
					"sre_log_detector_ticks_total":      ticksTotal,
					"sre_log_detector_failures_total":   failuresTotal,
					"sre_log_detector_findings_total":   findingsTotal,
					"sre_log_detector_active_rules":     int64(activeCount),
					"sre_log_detector_interval_seconds": int64(cfg.Interval / time.Second),
					"sre_log_detector_lookback_seconds": int64(cfg.Lookback / time.Second),
					"sre_log_detector_max_matches":      int64(cfg.MaxMatches),
				},
			})
		}

		runTick()
		for range ticker.C {
			runTick()
		}
	}()
}

func loadCustomLogDetectorRules(policyReader LogDetectorPolicyReader) []LogDetectorRule {
	if policyReader == nil {
		return nil
	}
	policy, err := policyReader.GetRobotSREPolicyConfig(context.Background(), nil)
	if err != nil || policy == nil || len(policy.DetectorRules) == 0 {
		return nil
	}
	out := make([]LogDetectorRule, 0, len(policy.DetectorRules))
	for _, rule := range policy.DetectorRules {
		if !rule.Enabled || strings.TrimSpace(rule.Query) == "" {
			continue
		}
		out = append(out, LogDetectorRule{
			Name:         strings.TrimSpace(rule.ID),
			Query:        strings.TrimSpace(rule.Query),
			Threshold:    maxInt(rule.Threshold, 1),
			Domain:       strings.TrimSpace(rule.Domain),
			IncidentType: strings.TrimSpace(rule.IncidentType),
			DisplayName:  strings.TrimSpace(rule.Name),
			Summary:      strings.TrimSpace(rule.Name),
			Severity:     strings.TrimSpace(rule.Severity),
			Confidence:   strings.TrimSpace(rule.Confidence),
			SignalKey:    strings.TrimSpace(rule.SignalKey),
		})
	}
	return out
}

func mergeLogDetectorRules(base []LogDetectorRule, custom []LogDetectorRule) []LogDetectorRule {
	if len(custom) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base))
	merged := make([]LogDetectorRule, 0, len(base)+len(custom))
	for _, rule := range base {
		seen[strings.TrimSpace(rule.Name)] = struct{}{}
		merged = append(merged, rule)
	}
	for _, rule := range custom {
		if _, exists := seen[strings.TrimSpace(rule.Name)]; exists {
			continue
		}
		merged = append(merged, rule)
	}
	return merged
}

func runLogDetectorTick(
	logger *zap.Logger,
	queryClient LogQueryClient,
	publisher *logFindingPublisher,
	rules []LogDetectorRule,
	active map[string]activeRuleState,
	start time.Time,
	end time.Time,
	lookback time.Duration,
	timeout time.Duration,
	maxMatches int,
) (activeCount int64, findingsDelta int64, failuresDelta int64) {
	for _, rule := range rules {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		result, err := queryClient.QueryRange(ctx, rule.Query, start, end, maxMatches)
		cancel()
		if err != nil {
			failuresDelta++
			if logger != nil {
				logger.Warn("SRE log detector query failed", zap.String("rule", rule.Name), zap.Error(err))
			}
			continue
		}
		matchCount := len(result.Matches)
		state, wasActive := active[rule.Name]
		if matchCount >= maxInt(rule.Threshold, 1) {
			activeCount++
			if err := publisher.PublishObserved(context.Background(), rule, result.Matches, end, lookback); err != nil {
				failuresDelta++
				if logger != nil {
					logger.Warn("Failed to publish detector finding", zap.String("rule", rule.Name), zap.Error(err))
				}
				continue
			}
			findingsDelta++
			active[rule.Name] = activeRuleState{lastObservedAt: end}
			continue
		}
		if wasActive {
			if err := publisher.PublishRecovered(context.Background(), rule, state.lastObservedAt, end); err != nil {
				failuresDelta++
				if logger != nil {
					logger.Warn("Failed to publish detector recovery", zap.String("rule", rule.Name), zap.Error(err))
				}
			}
			delete(active, rule.Name)
		}
	}
	return activeCount, findingsDelta, failuresDelta
}

func defaultLogDetectorRules() []LogDetectorRule {
	return []LogDetectorRule{
		{
			Name:         "docker_hub_rate_limit",
			Query:        `{namespace="image-factory"} |= "toomanyrequests"`,
			Threshold:    1,
			Domain:       "runtime_services",
			IncidentType: "registry_pull_failure",
			DisplayName:  "Registry pull rate limit detected",
			Summary:      "Repeated Docker Hub pull-rate limit errors were detected",
			Severity:     "warning",
			Confidence:   "high",
			SignalKey:    "toomanyrequests",
		},
		{
			Name:         "node_disk_pressure",
			Query:        `{namespace="image-factory"} |= "FreeDiskSpaceFailed"`,
			Threshold:    1,
			Domain:       "infrastructure",
			IncidentType: "node_disk_pressure",
			DisplayName:  "Node disk pressure signature detected",
			Summary:      "Node disk pressure log patterns were detected",
			Severity:     "critical",
			Confidence:   "medium",
			SignalKey:    "free_disk_space_failed",
		},
		{
			Name:         "ldap_timeout",
			Query:        `{namespace="image-factory"} |= "i/o timeout" |= "ldap"`,
			Threshold:    1,
			Domain:       "identity_security",
			IncidentType: "identity_provider_unreachable",
			DisplayName:  "LDAP timeout signature detected",
			Summary:      "LDAP timeout patterns were detected in application logs",
			Severity:     "warning",
			Confidence:   "medium",
			SignalKey:    "ldap_timeout",
		},
		{
			Name:         "notification_delivery_failure",
			Query:        `{}` + ` |= "Failed to enqueue notification email"`,
			Threshold:    1,
			Domain:       "application_services",
			IncidentType: "notification_delivery_failure",
			DisplayName:  "Notification delivery failure detected",
			Summary:      "Notification delivery failures were detected in worker logs",
			Severity:     "warning",
			Confidence:   "high",
			SignalKey:    "notification_delivery_failure",
		},
		{
			Name:         "notification_queue_persistence_failure",
			Query:        `{}` + ` |= "Failed to save email" |= "email_queue"`,
			Threshold:    1,
			Domain:       "application_services",
			IncidentType: "notification_delivery_failure",
			DisplayName:  "Notification queue persistence failure detected",
			Summary:      "Notification emails failed to persist into the queue",
			Severity:     "critical",
			Confidence:   "high",
			SignalKey:    "notification_queue_persistence_failure",
		},
		{
			Name:         "api_5xx_burst",
			Query:        `{}` + ` |= "HTTP/1.1" |= " - 5"`,
			Threshold:    3,
			Domain:       "golden_signals",
			IncidentType: "error_pressure",
			DisplayName:  "API 5xx burst detected",
			Summary:      "Repeated 5xx-style API access log responses were detected",
			Severity:     "warning",
			Confidence:   "medium",
			SignalKey:    "api_5xx_burst",
		},
		{
			Name:         "backend_panic_or_fatal",
			Query:        `{}` + ` |= "panic:"`,
			Threshold:    1,
			Domain:       "application_services",
			IncidentType: "application_service_degraded",
			DisplayName:  "Backend panic detected",
			Summary:      "A backend panic signature was detected in application logs",
			Severity:     "critical",
			Confidence:   "high",
			SignalKey:    "backend_panic_or_fatal",
		},
	}
}

func compactMatches(matches []logdetector.QueryMatch) []map[string]interface{} {
	if len(matches) == 0 {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(matches))
	for _, match := range matches {
		out = append(out, map[string]interface{}{
			"timestamp": match.Timestamp.Format(time.RFC3339),
			"line":      match.Line,
			"labels":    match.Labels,
		})
	}
	return out
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
