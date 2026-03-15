package sresmartbot

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	domainsresmartbot "github.com/srikarm/image-factory/internal/domain/sresmartbot"
	"go.uber.org/zap"
)

type Service struct {
	repo               domainsresmartbot.Repository
	publisher          EventPublisher
	logger             *zap.Logger
	suggestionObserver DetectorSuggestionObserver
}

type EventPublisher interface {
	PublishFindingObserved(ctx context.Context, incident *domainsresmartbot.Incident, finding *domainsresmartbot.Finding) error
	PublishIncidentResolved(ctx context.Context, incident *domainsresmartbot.Incident) error
	PublishEvidenceAdded(ctx context.Context, incident *domainsresmartbot.Incident, evidence *domainsresmartbot.Evidence) error
	PublishActionProposed(ctx context.Context, incident *domainsresmartbot.Incident, attempt *domainsresmartbot.ActionAttempt) error
}

type DetectorSuggestionObserver interface {
	ObserveSignal(ctx context.Context, incident *domainsresmartbot.Incident, observation SignalObservation, finding *domainsresmartbot.Finding)
}

type SignalObservation struct {
	TenantID       *uuid.UUID
	CorrelationKey string
	Domain         string
	IncidentType   string
	DisplayName    string
	Summary        string
	Source         string
	Severity       domainsresmartbot.IncidentSeverity
	Confidence     domainsresmartbot.IncidentConfidence
	OccurredAt     time.Time
	Metadata       map[string]interface{}
	FindingTitle   string
	FindingMessage string
	SignalType     string
	SignalKey      string
	RawPayload     map[string]interface{}
}

type ActionAttemptSpec struct {
	ActionKey        string
	ActionClass      string
	TargetKind       string
	TargetRef        string
	Status           string
	ActorType        string
	ActorID          string
	ApprovalRequired bool
	ResultPayload    map[string]interface{}
	ErrorMessage     string
}

func NewService(repo domainsresmartbot.Repository, publisher EventPublisher, logger *zap.Logger) *Service {
	return &Service{repo: repo, publisher: publisher, logger: logger}
}

func (s *Service) SetDetectorSuggestionObserver(observer DetectorSuggestionObserver) {
	if s == nil {
		return
	}
	s.suggestionObserver = observer
}

func (s *Service) RecordObservation(ctx context.Context, observation SignalObservation) error {
	if s == nil || s.repo == nil {
		return nil
	}
	if observation.CorrelationKey == "" || observation.Domain == "" || observation.IncidentType == "" {
		return errors.New("correlation key, domain, and incident type are required")
	}
	if observation.OccurredAt.IsZero() {
		observation.OccurredAt = time.Now().UTC()
	}

	incident, _, err := s.upsertIncident(ctx, observation)
	if err != nil {
		return err
	}

	finding := &domainsresmartbot.Finding{
		ID:         uuid.New(),
		IncidentID: &incident.ID,
		Source:     observation.Source,
		SignalType: observation.SignalType,
		SignalKey:  observation.SignalKey,
		Severity:   observation.Severity,
		Confidence: observation.Confidence,
		Title:      observation.FindingTitle,
		Message:    observation.FindingMessage,
		RawPayload: mustJSON(observation.RawPayload),
		OccurredAt: observation.OccurredAt,
		CreatedAt:  observation.OccurredAt,
	}
	if err := s.repo.CreateFinding(ctx, finding); err != nil {
		return fmt.Errorf("create finding: %w", err)
	}
	s.publishFindingObserved(ctx, incident, finding)
	if s.suggestionObserver != nil {
		s.suggestionObserver.ObserveSignal(ctx, incident, observation, finding)
	}
	return nil
}

func (s *Service) AddEvidence(ctx context.Context, correlationKey string, evidenceType string, summary string, payload map[string]interface{}, capturedAt time.Time) error {
	if s == nil || s.repo == nil || correlationKey == "" || strings.TrimSpace(evidenceType) == "" {
		return nil
	}
	if capturedAt.IsZero() {
		capturedAt = time.Now().UTC()
	}
	incident, err := s.repo.GetIncidentByCorrelationKey(ctx, correlationKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsNoRows(err) {
			return nil
		}
		return err
	}
	evidence := &domainsresmartbot.Evidence{
		ID:           uuid.New(),
		IncidentID:   incident.ID,
		EvidenceType: strings.TrimSpace(evidenceType),
		Summary:      summary,
		Payload:      mustJSON(payload),
		CapturedAt:   capturedAt,
		CreatedAt:    capturedAt,
	}
	if err := s.repo.AddEvidence(ctx, evidence); err != nil {
		return fmt.Errorf("create evidence: %w", err)
	}
	s.publishEvidenceAdded(ctx, incident, evidence)
	return nil
}

func (s *Service) EnsureActionAttempt(ctx context.Context, correlationKey string, spec ActionAttemptSpec, requestedAt time.Time) error {
	if s == nil || s.repo == nil || correlationKey == "" || strings.TrimSpace(spec.ActionKey) == "" {
		return nil
	}
	if requestedAt.IsZero() {
		requestedAt = time.Now().UTC()
	}
	incident, err := s.repo.GetIncidentByCorrelationKey(ctx, correlationKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsNoRows(err) {
			return nil
		}
		return err
	}
	existing, err := s.repo.ListActionAttemptsByIncident(ctx, incident.ID)
	if err != nil {
		return err
	}
	for _, attempt := range existing {
		if attempt == nil {
			continue
		}
		if attempt.ActionKey == spec.ActionKey && attempt.TargetKind == spec.TargetKind && attempt.TargetRef == spec.TargetRef {
			return nil
		}
	}
	action := &domainsresmartbot.ActionAttempt{
		ID:               uuid.New(),
		IncidentID:       incident.ID,
		ActionKey:        strings.TrimSpace(spec.ActionKey),
		ActionClass:      strings.TrimSpace(spec.ActionClass),
		TargetKind:       strings.TrimSpace(spec.TargetKind),
		TargetRef:        strings.TrimSpace(spec.TargetRef),
		Status:           strings.TrimSpace(spec.Status),
		ActorType:        strings.TrimSpace(spec.ActorType),
		ActorID:          strings.TrimSpace(spec.ActorID),
		ApprovalRequired: spec.ApprovalRequired,
		RequestedAt:      requestedAt,
		ErrorMessage:     strings.TrimSpace(spec.ErrorMessage),
		ResultPayload:    mustJSON(spec.ResultPayload),
		CreatedAt:        requestedAt,
		UpdatedAt:        requestedAt,
	}
	if action.ActionClass == "" {
		action.ActionClass = "recommendation"
	}
	if action.Status == "" {
		action.Status = "proposed"
	}
	if action.ActorType == "" {
		action.ActorType = "system"
	}
	if action.TargetKind == "" {
		action.TargetKind = "system"
	}
	if action.TargetRef == "" {
		action.TargetRef = "global"
	}
	if err := s.repo.CreateActionAttempt(ctx, action); err != nil {
		return fmt.Errorf("create action attempt: %w", err)
	}
	s.publishActionProposed(ctx, incident, action)
	return nil
}

func (s *Service) ResolveIncident(ctx context.Context, correlationKey string, occurredAt time.Time, resolutionSummary string, metadata map[string]interface{}) error {
	if s == nil || s.repo == nil || correlationKey == "" {
		return nil
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	incident, err := s.repo.GetIncidentByCorrelationKey(ctx, correlationKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || containsNoRows(err) {
			return nil
		}
		return err
	}
	incident.Status = domainsresmartbot.IncidentStatusResolved
	incident.Summary = resolutionSummary
	incident.ResolvedAt = &occurredAt
	incident.LastObservedAt = occurredAt
	incident.Metadata = mustJSON(metadata)
	incident.UpdatedAt = occurredAt
	if err := s.repo.UpdateIncident(ctx, incident); err != nil {
		return err
	}
	s.publishIncidentResolved(ctx, incident)
	return nil
}

func (s *Service) upsertIncident(ctx context.Context, observation SignalObservation) (*domainsresmartbot.Incident, bool, error) {
	incident, err := s.repo.GetIncidentByCorrelationKey(ctx, observation.CorrelationKey)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) && !containsNoRows(err) {
			return nil, false, err
		}
		incident = &domainsresmartbot.Incident{
			ID:              uuid.New(),
			TenantID:        observation.TenantID,
			CorrelationKey:  observation.CorrelationKey,
			Domain:          observation.Domain,
			IncidentType:    observation.IncidentType,
			DisplayName:     observation.DisplayName,
			Summary:         observation.Summary,
			Severity:        observation.Severity,
			Confidence:      observation.Confidence,
			Status:          domainsresmartbot.IncidentStatusObserved,
			Source:          observation.Source,
			FirstObservedAt: observation.OccurredAt,
			LastObservedAt:  observation.OccurredAt,
			Metadata:        mustJSON(observation.Metadata),
			CreatedAt:       observation.OccurredAt,
			UpdatedAt:       observation.OccurredAt,
		}
		if err := s.repo.CreateIncident(ctx, incident); err != nil {
			return nil, false, fmt.Errorf("create incident: %w", err)
		}
		return incident, true, nil
	}

	incident.DisplayName = observation.DisplayName
	incident.TenantID = observation.TenantID
	incident.Summary = observation.Summary
	incident.Severity = observation.Severity
	incident.Confidence = observation.Confidence
	incident.Status = domainsresmartbot.IncidentStatusObserved
	incident.Source = observation.Source
	incident.LastObservedAt = observation.OccurredAt
	incident.ResolvedAt = nil
	incident.ContainedAt = nil
	incident.Metadata = mustJSON(observation.Metadata)
	incident.UpdatedAt = observation.OccurredAt
	if err := s.repo.UpdateIncident(ctx, incident); err != nil {
		return nil, false, fmt.Errorf("update incident: %w", err)
	}
	return incident, false, nil
}

func mustJSON(payload map[string]interface{}) json.RawMessage {
	if len(payload) == 0 {
		return json.RawMessage(`{}`)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return encoded
}

func (s *Service) publishFindingObserved(ctx context.Context, incident *domainsresmartbot.Incident, finding *domainsresmartbot.Finding) {
	if s == nil || s.publisher == nil {
		return
	}
	if err := s.publisher.PublishFindingObserved(ctx, incident, finding); err != nil && s.logger != nil {
		s.logger.Warn("Failed to publish SRE finding observed event", zap.Error(err))
	}
}

func (s *Service) publishIncidentResolved(ctx context.Context, incident *domainsresmartbot.Incident) {
	if s == nil || s.publisher == nil {
		return
	}
	if err := s.publisher.PublishIncidentResolved(ctx, incident); err != nil && s.logger != nil {
		s.logger.Warn("Failed to publish SRE incident resolved event", zap.Error(err))
	}
}

func (s *Service) publishEvidenceAdded(ctx context.Context, incident *domainsresmartbot.Incident, evidence *domainsresmartbot.Evidence) {
	if s == nil || s.publisher == nil {
		return
	}
	if err := s.publisher.PublishEvidenceAdded(ctx, incident, evidence); err != nil && s.logger != nil {
		s.logger.Warn("Failed to publish SRE evidence added event", zap.Error(err))
	}
}

func (s *Service) publishActionProposed(ctx context.Context, incident *domainsresmartbot.Incident, action *domainsresmartbot.ActionAttempt) {
	if s == nil || s.publisher == nil {
		return
	}
	if err := s.publisher.PublishActionProposed(ctx, incident, action); err != nil && s.logger != nil {
		s.logger.Warn("Failed to publish SRE action proposed event", zap.Error(err))
	}
}

func containsNoRows(err error) bool {
	return err != nil && (errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows in result set") || strings.Contains(err.Error(), "sql: no rows"))
}
