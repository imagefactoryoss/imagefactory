package sresmartbot

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	domainsresmartbot "github.com/srikarm/image-factory/internal/domain/sresmartbot"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type DetectorEventSubscriber struct {
	service *Service
	logger  *zap.Logger
}

func NewDetectorEventSubscriber(service *Service, logger *zap.Logger) *DetectorEventSubscriber {
	return &DetectorEventSubscriber{
		service: service,
		logger:  logger,
	}
}

func RegisterDetectorEventSubscriber(bus messaging.EventBus, subscriber *DetectorEventSubscriber) func() {
	if bus == nil || subscriber == nil {
		return func() {}
	}
	unsubObserved := bus.Subscribe(messaging.EventTypeSREDetectorFindingObserved, subscriber.HandleFindingEvent)
	unsubRecovered := bus.Subscribe(messaging.EventTypeSREDetectorFindingRecovered, subscriber.HandleRecoveryEvent)
	return func() {
		unsubObserved()
		unsubRecovered()
	}
}

func (s *DetectorEventSubscriber) HandleFindingEvent(ctx context.Context, event messaging.Event) {
	if s == nil || s.service == nil {
		return
	}
	observation, evidenceType, evidenceSummary, evidencePayload, ok := mapDetectorEvent(event)
	if !ok {
		return
	}
	if err := s.service.RecordObservation(ctx, observation); err != nil {
		s.logWarn("Failed to record detector finding observation",
			zap.String("event_type", event.Type),
			zap.String("correlation_key", observation.CorrelationKey),
			zap.Error(err))
		return
	}
	if strings.TrimSpace(evidenceType) != "" {
		if err := s.service.AddEvidence(ctx, observation.CorrelationKey, evidenceType, evidenceSummary, evidencePayload, observation.OccurredAt); err != nil {
			s.logWarn("Failed to record detector finding evidence",
				zap.String("event_type", event.Type),
				zap.String("correlation_key", observation.CorrelationKey),
				zap.Error(err))
		}
	}
}

func (s *DetectorEventSubscriber) HandleRecoveryEvent(ctx context.Context, event messaging.Event) {
	if s == nil || s.service == nil {
		return
	}
	payload := event.Payload
	correlationKey := strings.TrimSpace(stringPayload(payload, "correlation_key"))
	domain := strings.TrimSpace(stringPayload(payload, "domain"))
	incidentType := strings.TrimSpace(stringPayload(payload, "incident_type"))
	summary := strings.TrimSpace(stringPayload(payload, "summary"))
	if correlationKey == "" || domain == "" || incidentType == "" {
		return
	}
	observedAt := parseDetectorTime(payload, "resolved_at")
	if observedAt.IsZero() {
		observedAt = event.OccurredAt
	}
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	metadata := nestedMap(payload, "metadata")
	if len(metadata) == 0 {
		metadata = map[string]interface{}{}
	}
	metadata["source"] = defaultString(stringPayload(payload, "source"), "detector")
	if err := s.service.ResolveIncident(ctx, correlationKey, observedAt, summary, metadata); err != nil {
		s.logWarn("Failed to resolve detector-driven incident",
			zap.String("event_type", event.Type),
			zap.String("correlation_key", correlationKey),
			zap.Error(err))
	}
}

func mapDetectorEvent(event messaging.Event) (SignalObservation, string, string, map[string]interface{}, bool) {
	payload := event.Payload
	if len(payload) == 0 {
		return SignalObservation{}, "", "", nil, false
	}

	occurredAt := parseDetectorTime(payload, "observed_at")
	if occurredAt.IsZero() {
		occurredAt = event.OccurredAt
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	rawPayload := nestedMap(payload, "raw_payload")
	if len(rawPayload) == 0 {
		rawPayload = payload
	}

	metadata := nestedMap(payload, "metadata")
	if len(metadata) == 0 {
		metadata = map[string]interface{}{}
	}
	if detectorName := strings.TrimSpace(stringPayload(payload, "detector_name")); detectorName != "" {
		metadata["detector_name"] = detectorName
	}
	if queryRef := strings.TrimSpace(stringPayload(payload, "query_ref")); queryRef != "" {
		metadata["query_ref"] = queryRef
	}

	observation := SignalObservation{
		CorrelationKey: strings.TrimSpace(stringPayload(payload, "correlation_key")),
		Domain:         strings.TrimSpace(stringPayload(payload, "domain")),
		IncidentType:   strings.TrimSpace(stringPayload(payload, "incident_type")),
		DisplayName:    defaultString(stringPayload(payload, "display_name"), strings.TrimSpace(stringPayload(payload, "finding_title"))),
		Summary:        strings.TrimSpace(stringPayload(payload, "summary")),
		Source:         strings.TrimSpace(stringPayload(payload, "source")),
		Severity:       parseSeverity(stringPayload(payload, "severity")),
		Confidence:     parseConfidence(stringPayload(payload, "confidence")),
		OccurredAt:     occurredAt,
		Metadata:       metadata,
		FindingTitle:   strings.TrimSpace(stringPayload(payload, "finding_title")),
		FindingMessage: strings.TrimSpace(stringPayload(payload, "finding_message")),
		SignalType:     strings.TrimSpace(stringPayload(payload, "signal_type")),
		SignalKey:      strings.TrimSpace(stringPayload(payload, "signal_key")),
		RawPayload:     rawPayload,
	}
	if observation.CorrelationKey == "" || observation.Domain == "" || observation.IncidentType == "" {
		return SignalObservation{}, "", "", nil, false
	}
	if observation.DisplayName == "" {
		observation.DisplayName = observation.IncidentType
	}
	if observation.Summary == "" {
		observation.Summary = observation.FindingMessage
	}
	if observation.Source == "" {
		observation.Source = "detector"
	}
	if observation.FindingTitle == "" {
		observation.FindingTitle = observation.DisplayName
	}
	if observation.FindingMessage == "" {
		observation.FindingMessage = observation.Summary
	}
	if observation.SignalType == "" {
		observation.SignalType = "detector_finding"
	}
	if observation.SignalKey == "" {
		observation.SignalKey = observation.CorrelationKey
	}

	evidenceType := strings.TrimSpace(stringPayload(payload, "evidence_type"))
	evidenceSummary := strings.TrimSpace(stringPayload(payload, "evidence_summary"))
	evidencePayload := nestedMap(payload, "evidence_payload")
	if evidenceType == "" && len(evidencePayload) > 0 {
		evidenceType = "detector_evidence"
	}
	if evidenceSummary == "" {
		evidenceSummary = observation.Summary
	}

	return observation, evidenceType, evidenceSummary, evidencePayload, true
}

func parseSeverity(value string) domainsresmartbot.IncidentSeverity {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(domainsresmartbot.IncidentSeverityCritical):
		return domainsresmartbot.IncidentSeverityCritical
	case string(domainsresmartbot.IncidentSeverityInfo):
		return domainsresmartbot.IncidentSeverityInfo
	default:
		return domainsresmartbot.IncidentSeverityWarning
	}
}

func parseConfidence(value string) domainsresmartbot.IncidentConfidence {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(domainsresmartbot.IncidentConfidenceLow):
		return domainsresmartbot.IncidentConfidenceLow
	case string(domainsresmartbot.IncidentConfidenceHigh):
		return domainsresmartbot.IncidentConfidenceHigh
	default:
		return domainsresmartbot.IncidentConfidenceMedium
	}
}

func parseDetectorTime(payload map[string]interface{}, key string) time.Time {
	raw := strings.TrimSpace(stringPayload(payload, key))
	if raw == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func nestedMap(payload map[string]interface{}, key string) map[string]interface{} {
	if payload == nil {
		return nil
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case map[string]interface{}:
		return typed
	case json.RawMessage:
		var decoded map[string]interface{}
		if err := json.Unmarshal(typed, &decoded); err == nil {
			return decoded
		}
	case string:
		var decoded map[string]interface{}
		if err := json.Unmarshal([]byte(typed), &decoded); err == nil {
			return decoded
		}
	}
	return nil
}

func stringPayload(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

func (s *DetectorEventSubscriber) logWarn(message string, fields ...zap.Field) {
	if s == nil || s.logger == nil {
		return
	}
	s.logger.Warn(message, fields...)
}
