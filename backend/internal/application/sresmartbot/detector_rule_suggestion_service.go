package sresmartbot

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	domainsresmartbot "github.com/srikarm/image-factory/internal/domain/sresmartbot"
	domainsystemconfig "github.com/srikarm/image-factory/internal/domain/systemconfig"
	"go.uber.org/zap"
)

var detectorSuggestionNoise = regexp.MustCompile(`[\r\n\t]+`)

type DetectorRulePolicyStore interface {
	GetRobotSREPolicyConfig(ctx context.Context, tenantID *uuid.UUID) (*domainsystemconfig.RobotSREPolicyConfig, error)
	UpdateRobotSREPolicyConfig(ctx context.Context, tenantID *uuid.UUID, cfg *domainsystemconfig.RobotSREPolicyConfig, updatedBy uuid.UUID) (*domainsystemconfig.RobotSREPolicyConfig, error)
}

type DetectorRuleSuggestionService struct {
	repo        domainsresmartbot.Repository
	policyStore DetectorRulePolicyStore
	logger      *zap.Logger
}

func NewDetectorRuleSuggestionService(repo domainsresmartbot.Repository, policyStore DetectorRulePolicyStore, logger *zap.Logger) *DetectorRuleSuggestionService {
	return &DetectorRuleSuggestionService{repo: repo, policyStore: policyStore, logger: logger}
}

func (s *DetectorRuleSuggestionService) ObserveSignal(ctx context.Context, incident *domainsresmartbot.Incident, observation SignalObservation, finding *domainsresmartbot.Finding) {
	if s == nil || s.repo == nil || s.policyStore == nil || incident == nil {
		return
	}
	if strings.EqualFold(strings.TrimSpace(observation.Source), "sre_log_detector") {
		return
	}
	policy, err := s.policyStore.GetRobotSREPolicyConfig(ctx, incident.TenantID)
	if err != nil || policy == nil {
		return
	}
	mode := strings.TrimSpace(policy.DetectorLearningMode)
	if mode == "" || mode == "disabled" {
		return
	}
	suggestion, rule, ok := buildDetectorRuleSuggestion(incident, observation, finding)
	if !ok {
		return
	}
	if ruleAlreadyActive(policy, rule) {
		return
	}
	existing, err := s.repo.GetDetectorRuleSuggestionByFingerprint(ctx, suggestion.Fingerprint)
	if err == nil && existing != nil {
		return
	}
	if err := s.repo.CreateDetectorRuleSuggestion(ctx, suggestion); err != nil {
		if s.logger != nil {
			s.logger.Warn("Failed to create detector rule suggestion", zap.Error(err))
		}
		return
	}
	if mode == "training_auto_create" {
		if _, err := s.AcceptSuggestion(ctx, suggestion.ID, "sre-smart-bot", nil); err != nil && s.logger != nil {
			s.logger.Warn("Failed to auto-activate detector rule suggestion", zap.Error(err), zap.String("suggestion_id", suggestion.ID.String()))
		}
	}
}

func (s *DetectorRuleSuggestionService) ListSuggestions(ctx context.Context, filter domainsresmartbot.DetectorRuleSuggestionFilter) ([]*domainsresmartbot.DetectorRuleSuggestion, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	return s.repo.ListDetectorRuleSuggestions(ctx, filter)
}

func (s *DetectorRuleSuggestionService) ProposeFromIncident(ctx context.Context, incidentID uuid.UUID, proposedBy string) (*domainsresmartbot.DetectorRuleSuggestion, error) {
	if s == nil || s.repo == nil || s.policyStore == nil {
		return nil, nil
	}
	incident, err := s.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	findings, err := s.repo.ListFindingsByIncident(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	if len(findings) == 0 {
		return nil, errors.New("incident has no findings to learn from")
	}
	lead := findings[0]
	observation := SignalObservation{
		TenantID:       incident.TenantID,
		CorrelationKey: incident.CorrelationKey,
		Domain:         incident.Domain,
		IncidentType:   incident.IncidentType,
		DisplayName:    incident.DisplayName,
		Summary:        incident.Summary,
		Source:         lead.Source,
		Severity:       lead.Severity,
		Confidence:     lead.Confidence,
		OccurredAt:     lead.OccurredAt,
		FindingTitle:   lead.Title,
		FindingMessage: lead.Message,
		SignalType:     lead.SignalType,
		SignalKey:      lead.SignalKey,
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(lead.RawPayload, &raw); err == nil {
		observation.RawPayload = raw
	}
	suggestion, _, ok := buildDetectorRuleSuggestion(incident, observation, lead)
	if !ok {
		return nil, errors.New("incident does not contain enough structured log evidence to propose a detector rule")
	}
	suggestion.ProposedBy = strings.TrimSpace(proposedBy)
	existing, err := s.repo.GetDetectorRuleSuggestionByFingerprint(ctx, suggestion.Fingerprint)
	if err == nil && existing != nil {
		return existing, nil
	}
	if err := s.repo.CreateDetectorRuleSuggestion(ctx, suggestion); err != nil {
		return nil, err
	}
	return suggestion, nil
}

// AcceptSuggestion promotes a detector rule suggestion into the active policy.
// policyTenantID overrides which tenant scope the policy is written to; pass nil to fall back to
// the suggestion's own tenant (the incident's original tenant).
func (s *DetectorRuleSuggestionService) AcceptSuggestion(ctx context.Context, suggestionID uuid.UUID, reviewedBy string, policyTenantID *uuid.UUID) (*domainsresmartbot.DetectorRuleSuggestion, error) {
	if s == nil || s.repo == nil || s.policyStore == nil {
		return nil, nil
	}
	suggestion, err := s.repo.GetDetectorRuleSuggestion(ctx, suggestionID)
	if err != nil {
		return nil, err
	}
	// Determine the tenant scope for the policy write.  Prefer the explicitly provided
	// policyTenantID (derived from the admin's current request context) so that a global
	// admin accepting a suggestion always writes to the correct scope.
	writeTenantID := suggestion.TenantID
	if policyTenantID != nil {
		writeTenantID = policyTenantID
	}
	policy, err := s.policyStore.GetRobotSREPolicyConfig(ctx, writeTenantID)
	if err != nil {
		return nil, err
	}
	rule := detectorRuleSuggestionToPolicyRule(suggestion)
	if !ruleAlreadyActive(policy, rule) {
		policy.DetectorRules = append(policy.DetectorRules, rule)
		updatedBy := uuid.Nil
		if parsed, parseErr := uuid.Parse(strings.TrimSpace(reviewedBy)); parseErr == nil {
			updatedBy = parsed
		}
		if _, err := s.policyStore.UpdateRobotSREPolicyConfig(ctx, writeTenantID, policy, updatedBy); err != nil {
			return nil, err
		}
	}
	now := time.Now().UTC()
	suggestion.Status = domainsresmartbot.DetectorRuleSuggestionStatusAccepted
	suggestion.ReviewedBy = strings.TrimSpace(reviewedBy)
	suggestion.ReviewedAt = &now
	suggestion.ActivatedRuleID = rule.ID
	suggestion.UpdatedAt = now
	if err := s.repo.UpdateDetectorRuleSuggestion(ctx, suggestion); err != nil {
		return nil, err
	}
	return suggestion, nil
}

func (s *DetectorRuleSuggestionService) RejectSuggestion(ctx context.Context, suggestionID uuid.UUID, reviewedBy string, reason string) (*domainsresmartbot.DetectorRuleSuggestion, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	suggestion, err := s.repo.GetDetectorRuleSuggestion(ctx, suggestionID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	suggestion.Status = domainsresmartbot.DetectorRuleSuggestionStatusRejected
	suggestion.ReviewedBy = strings.TrimSpace(reviewedBy)
	suggestion.ReviewedAt = &now
	suggestion.Reason = strings.TrimSpace(reason)
	suggestion.UpdatedAt = now
	if err := s.repo.UpdateDetectorRuleSuggestion(ctx, suggestion); err != nil {
		return nil, err
	}
	return suggestion, nil
}

func buildDetectorRuleSuggestion(incident *domainsresmartbot.Incident, observation SignalObservation, finding *domainsresmartbot.Finding) (*domainsresmartbot.DetectorRuleSuggestion, domainsystemconfig.RobotSREDetectorRule, bool) {
	phrase := extractDetectorPhrase(observation, finding)
	if phrase == "" {
		return nil, domainsystemconfig.RobotSREDetectorRule{}, false
	}
	if len(phrase) > 160 {
		phrase = phrase[:160]
	}
	query := fmt.Sprintf(`{} |= %q`, phrase)
	signalKey := strings.TrimSpace(observation.SignalKey)
	if signalKey == "" {
		signalKey = slugifyDetectorID(phrase)
	}
	ruleID := slugifyDetectorID(observation.Domain + "-" + observation.IncidentType + "-" + signalKey)
	fingerprintInput := strings.Join([]string{
		stringOrEmptyUUID(incident.TenantID),
		observation.Domain,
		observation.IncidentType,
		signalKey,
		query,
	}, "|")
	hash := sha1.Sum([]byte(fingerprintInput))
	fingerprint := hex.EncodeToString(hash[:])
	evidencePayload := mustJSON(map[string]interface{}{
		"phrase":        phrase,
		"query":         query,
		"source":        observation.Source,
		"signal_key":    signalKey,
		"finding_title": observation.FindingTitle,
		"finding_msg":   observation.FindingMessage,
		"raw_payload":   observation.RawPayload,
	})
	if finding != nil && len(finding.RawPayload) > 0 {
		evidencePayload = finding.RawPayload
	}
	severity := observation.Severity
	if severity == "" {
		severity = domainsresmartbot.IncidentSeverityWarning
	}
	confidence := observation.Confidence
	if confidence == "" {
		confidence = domainsresmartbot.IncidentConfidenceMedium
	}
	name := fmt.Sprintf("Learned detector: %s", defaultString(observation.DisplayName, incident.DisplayName))
	suggestion := &domainsresmartbot.DetectorRuleSuggestion{
		ID:              uuid.New(),
		TenantID:        incident.TenantID,
		IncidentID:      &incident.ID,
		Fingerprint:     fingerprint,
		Name:            name,
		Description:     fmt.Sprintf("Proposed from observed incident pattern: %s", defaultString(observation.Summary, incident.Summary)),
		Query:           query,
		Threshold:       1,
		Domain:          observation.Domain,
		IncidentType:    observation.IncidentType,
		Severity:        severity,
		Confidence:      confidence,
		SignalKey:       signalKey,
		Source:          defaultString(observation.Source, "sre_smart_bot_learning"),
		Status:          domainsresmartbot.DetectorRuleSuggestionStatusPending,
		AutoCreated:     true,
		Reason:          "Observed repeated or novel incident signal without an active matching detector rule.",
		EvidencePayload: evidencePayload,
		ProposedBy:      "sre-smart-bot",
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	rule := domainsystemconfig.RobotSREDetectorRule{
		ID:           ruleID,
		Name:         name,
		Enabled:      true,
		Source:       "learned",
		Query:        query,
		Threshold:    1,
		Domain:       observation.Domain,
		IncidentType: observation.IncidentType,
		Severity:     string(severity),
		Confidence:   string(confidence),
		SignalKey:    signalKey,
		AutoCreated:  true,
	}
	return suggestion, rule, true
}

func detectorRuleSuggestionToPolicyRule(s *domainsresmartbot.DetectorRuleSuggestion) domainsystemconfig.RobotSREDetectorRule {
	return domainsystemconfig.RobotSREDetectorRule{
		ID:           slugifyDetectorID(defaultString(s.ActivatedRuleID, s.Name+"-"+s.SignalKey)),
		Name:         s.Name,
		Enabled:      true,
		Source:       "learned",
		Query:        s.Query,
		Threshold:    positiveOrFallbackInt(s.Threshold, 1),
		Domain:       s.Domain,
		IncidentType: s.IncidentType,
		Severity:     string(s.Severity),
		Confidence:   string(s.Confidence),
		SignalKey:    s.SignalKey,
		AutoCreated:  s.AutoCreated,
	}
}

func ruleAlreadyActive(policy *domainsystemconfig.RobotSREPolicyConfig, rule domainsystemconfig.RobotSREDetectorRule) bool {
	if policy == nil {
		return false
	}
	for _, existing := range policy.DetectorRules {
		if !existing.Enabled {
			continue
		}
		if strings.EqualFold(existing.ID, rule.ID) || (strings.EqualFold(existing.Query, rule.Query) && strings.EqualFold(existing.IncidentType, rule.IncidentType)) {
			return true
		}
	}
	return false
}

func extractDetectorPhrase(observation SignalObservation, finding *domainsresmartbot.Finding) string {
	candidates := []string{
		strings.TrimSpace(observation.SignalKey),
		stringMapValue(observation.RawPayload, "last_error"),
		stringMapValue(observation.RawPayload, "error"),
		stringMapValue(observation.RawPayload, "message"),
		stringMapValue(observation.RawPayload, "line"),
		strings.TrimSpace(observation.FindingMessage),
	}
	if finding != nil {
		var raw map[string]interface{}
		if err := json.Unmarshal(finding.RawPayload, &raw); err == nil {
			candidates = append(candidates, stringMapValue(raw, "last_error"), stringMapValue(raw, "error"), stringMapValue(raw, "message"), stringMapValue(raw, "line"))
		}
		candidates = append(candidates, finding.Message, finding.Title)
	}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(detectorSuggestionNoise.ReplaceAllString(candidate, " "))
		if candidate == "" {
			continue
		}
		switch strings.ToLower(candidate) {
		case "error", "warning", "failed", "detected":
			continue
		}
		return candidate
	}
	return ""
}

func stringMapValue(payload map[string]interface{}, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func slugifyDetectorID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = detectorSuggestionNoise.ReplaceAllString(value, "-")
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteRune('-')
			lastDash = true
		}
	}
	result := strings.Trim(out.String(), "-")
	if result == "" {
		return "detector-rule"
	}
	if len(result) > 64 {
		return result[:64]
	}
	return result
}

func stringOrEmptyUUID(value *uuid.UUID) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func positiveOrFallbackInt(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
