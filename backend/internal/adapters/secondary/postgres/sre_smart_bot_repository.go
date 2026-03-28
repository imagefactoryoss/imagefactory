package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/domain/sresmartbot"
	"go.uber.org/zap"
)

type SRESmartBotRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewSRESmartBotRepository(db *sqlx.DB, logger *zap.Logger) *SRESmartBotRepository {
	return &SRESmartBotRepository{db: db, logger: logger}
}

type sreIncidentModel struct {
	ID              uuid.UUID       `db:"id"`
	TenantID        *uuid.UUID      `db:"tenant_id"`
	CorrelationKey  string          `db:"correlation_key"`
	Domain          string          `db:"domain"`
	IncidentType    string          `db:"incident_type"`
	DisplayName     string          `db:"display_name"`
	Summary         string          `db:"summary"`
	Severity        string          `db:"severity"`
	Confidence      string          `db:"confidence"`
	Status          string          `db:"status"`
	Source          string          `db:"source"`
	FirstObservedAt time.Time       `db:"first_observed_at"`
	LastObservedAt  time.Time       `db:"last_observed_at"`
	ContainedAt     *time.Time      `db:"contained_at"`
	ResolvedAt      *time.Time      `db:"resolved_at"`
	SuppressedUntil *time.Time      `db:"suppressed_until"`
	Metadata        json.RawMessage `db:"metadata"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

type sreFindingModel struct {
	ID         uuid.UUID       `db:"id"`
	IncidentID *uuid.UUID      `db:"incident_id"`
	Source     string          `db:"source"`
	SignalType string          `db:"signal_type"`
	SignalKey  string          `db:"signal_key"`
	Severity   string          `db:"severity"`
	Confidence string          `db:"confidence"`
	Title      string          `db:"title"`
	Message    string          `db:"message"`
	RawPayload json.RawMessage `db:"raw_payload"`
	OccurredAt time.Time       `db:"occurred_at"`
	CreatedAt  time.Time       `db:"created_at"`
}

type sreEvidenceModel struct {
	ID           uuid.UUID       `db:"id"`
	IncidentID   uuid.UUID       `db:"incident_id"`
	EvidenceType string          `db:"evidence_type"`
	Summary      string          `db:"summary"`
	Payload      json.RawMessage `db:"payload"`
	CapturedAt   time.Time       `db:"captured_at"`
	CreatedAt    time.Time       `db:"created_at"`
}

type sreActionAttemptModel struct {
	ID               uuid.UUID       `db:"id"`
	IncidentID       uuid.UUID       `db:"incident_id"`
	ActionKey        string          `db:"action_key"`
	ActionClass      string          `db:"action_class"`
	TargetKind       string          `db:"target_kind"`
	TargetRef        string          `db:"target_ref"`
	Status           string          `db:"status"`
	ActorType        string          `db:"actor_type"`
	ActorID          sql.NullString  `db:"actor_id"`
	ApprovalRequired bool            `db:"approval_required"`
	RequestedAt      time.Time       `db:"requested_at"`
	StartedAt        *time.Time      `db:"started_at"`
	CompletedAt      *time.Time      `db:"completed_at"`
	ErrorMessage     sql.NullString  `db:"error_message"`
	ResultPayload    json.RawMessage `db:"result_payload"`
	CreatedAt        time.Time       `db:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at"`
}

type sreApprovalModel struct {
	ID                uuid.UUID      `db:"id"`
	IncidentID        uuid.UUID      `db:"incident_id"`
	ActionAttemptID   *uuid.UUID     `db:"action_attempt_id"`
	ChannelProviderID string         `db:"channel_provider_id"`
	Status            string         `db:"status"`
	RequestMessage    string         `db:"request_message"`
	RequestedBy       sql.NullString `db:"requested_by"`
	DecidedBy         sql.NullString `db:"decided_by"`
	DecisionComment   sql.NullString `db:"decision_comment"`
	RequestedAt       time.Time      `db:"requested_at"`
	DecidedAt         *time.Time     `db:"decided_at"`
	ExpiresAt         *time.Time     `db:"expires_at"`
	CreatedAt         time.Time      `db:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at"`
}

type sreDetectorRuleSuggestionModel struct {
	ID              uuid.UUID       `db:"id"`
	TenantID        *uuid.UUID      `db:"tenant_id"`
	IncidentID      *uuid.UUID      `db:"incident_id"`
	Fingerprint     string          `db:"fingerprint"`
	Name            string          `db:"name"`
	Description     string          `db:"description"`
	Query           string          `db:"query"`
	Threshold       int             `db:"threshold"`
	Domain          string          `db:"domain"`
	IncidentType    string          `db:"incident_type"`
	Severity        string          `db:"severity"`
	Confidence      string          `db:"confidence"`
	SignalKey       sql.NullString  `db:"signal_key"`
	Source          string          `db:"source"`
	Status          string          `db:"status"`
	AutoCreated     bool            `db:"auto_created"`
	Reason          sql.NullString  `db:"reason"`
	EvidencePayload json.RawMessage `db:"evidence_payload"`
	ProposedBy      sql.NullString  `db:"proposed_by"`
	ReviewedBy      sql.NullString  `db:"reviewed_by"`
	ReviewedAt      *time.Time      `db:"reviewed_at"`
	ActivatedRuleID sql.NullString  `db:"activated_rule_id"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

type sreRemediationPackRunModel struct {
	ID              uuid.UUID       `db:"id"`
	TenantID        *uuid.UUID      `db:"tenant_id"`
	IncidentID      uuid.UUID       `db:"incident_id"`
	PackKey         string          `db:"pack_key"`
	PackVersion     string          `db:"pack_version"`
	RunKind         string          `db:"run_kind"`
	Status          string          `db:"status"`
	RequestedBy     sql.NullString  `db:"requested_by"`
	RequestID       sql.NullString  `db:"request_id"`
	ApprovalID      *uuid.UUID      `db:"approval_id"`
	ActionAttemptID *uuid.UUID      `db:"action_attempt_id"`
	Summary         string          `db:"summary"`
	ResultPayload   json.RawMessage `db:"result_payload"`
	StartedAt       *time.Time      `db:"started_at"`
	CompletedAt     *time.Time      `db:"completed_at"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

func (r *SRESmartBotRepository) CreateIncident(ctx context.Context, incident *sresmartbot.Incident) error {
	query := `
		INSERT INTO sre_incidents (
			id, tenant_id, correlation_key, domain, incident_type, display_name, summary,
			severity, confidence, status, source, first_observed_at, last_observed_at,
			contained_at, resolved_at, suppressed_until, metadata, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19
		)`
	_, err := r.db.ExecContext(ctx, query,
		incident.ID, incident.TenantID, incident.CorrelationKey, incident.Domain, incident.IncidentType, incident.DisplayName, incident.Summary,
		string(incident.Severity), string(incident.Confidence), string(incident.Status), incident.Source, incident.FirstObservedAt, incident.LastObservedAt,
		incident.ContainedAt, incident.ResolvedAt, incident.SuppressedUntil, normalizeJSON(incident.Metadata), incident.CreatedAt, incident.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create sre incident: %w", err)
	}
	return nil
}

func (r *SRESmartBotRepository) UpdateIncident(ctx context.Context, incident *sresmartbot.Incident) error {
	query := `
		UPDATE sre_incidents
		SET tenant_id = $2,
		    domain = $3,
		    incident_type = $4,
		    display_name = $5,
		    summary = $6,
		    severity = $7,
		    confidence = $8,
		    status = $9,
		    source = $10,
		    first_observed_at = $11,
		    last_observed_at = $12,
		    contained_at = $13,
		    resolved_at = $14,
		    suppressed_until = $15,
		    metadata = $16,
		    updated_at = $17
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query,
		incident.ID, incident.TenantID, incident.Domain, incident.IncidentType, incident.DisplayName, incident.Summary,
		string(incident.Severity), string(incident.Confidence), string(incident.Status), incident.Source, incident.FirstObservedAt, incident.LastObservedAt,
		incident.ContainedAt, incident.ResolvedAt, incident.SuppressedUntil, normalizeJSON(incident.Metadata), incident.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update sre incident: %w", err)
	}
	return nil
}

func (r *SRESmartBotRepository) GetIncident(ctx context.Context, id uuid.UUID) (*sresmartbot.Incident, error) {
	var model sreIncidentModel
	query := `
		SELECT id, tenant_id, correlation_key, domain, incident_type, display_name, summary,
		       severity, confidence, status, source, first_observed_at, last_observed_at,
		       contained_at, resolved_at, suppressed_until, metadata, created_at, updated_at
		FROM sre_incidents
		WHERE id = $1`
	if err := r.db.GetContext(ctx, &model, query, id); err != nil {
		return nil, fmt.Errorf("get sre incident: %w", err)
	}
	incident := incidentModelToDomain(model)
	return &incident, nil
}

func (r *SRESmartBotRepository) GetIncidentByCorrelationKey(ctx context.Context, correlationKey string) (*sresmartbot.Incident, error) {
	var model sreIncidentModel
	query := `
		SELECT id, tenant_id, correlation_key, domain, incident_type, display_name, summary,
		       severity, confidence, status, source, first_observed_at, last_observed_at,
		       contained_at, resolved_at, suppressed_until, metadata, created_at, updated_at
		FROM sre_incidents
		WHERE correlation_key = $1`
	if err := r.db.GetContext(ctx, &model, query, correlationKey); err != nil {
		return nil, fmt.Errorf("get sre incident by correlation key: %w", err)
	}
	incident := incidentModelToDomain(model)
	return &incident, nil
}

func (r *SRESmartBotRepository) ListIncidents(ctx context.Context, filter sresmartbot.IncidentFilter) ([]*sresmartbot.Incident, error) {
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	query := `
		SELECT id, tenant_id, correlation_key, domain, incident_type, display_name, summary,
		       severity, confidence, status, source, first_observed_at, last_observed_at,
		       contained_at, resolved_at, suppressed_until, metadata, created_at, updated_at
		FROM sre_incidents`
	clauses := make([]string, 0, 4)
	args := make([]interface{}, 0, 6)
	arg := 1
	if filter.TenantID != nil {
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", arg))
		args = append(args, *filter.TenantID)
		arg++
	}
	if strings.TrimSpace(filter.Domain) != "" {
		clauses = append(clauses, fmt.Sprintf("domain = $%d", arg))
		args = append(args, strings.TrimSpace(filter.Domain))
		arg++
	}
	if strings.TrimSpace(filter.Status) != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", arg))
		args = append(args, strings.TrimSpace(filter.Status))
		arg++
	}
	if strings.TrimSpace(filter.Severity) != "" {
		clauses = append(clauses, fmt.Sprintf("severity = $%d", arg))
		args = append(args, strings.TrimSpace(filter.Severity))
		arg++
	}
	if strings.TrimSpace(filter.Search) != "" {
		clauses = append(clauses, fmt.Sprintf("(display_name ILIKE $%d OR summary ILIKE $%d OR incident_type ILIKE $%d)", arg, arg, arg))
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		arg++
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY last_observed_at DESC LIMIT $%d OFFSET $%d", arg, arg+1)
	args = append(args, filter.Limit, filter.Offset)

	models := make([]sreIncidentModel, 0)
	if err := r.db.SelectContext(ctx, &models, query, args...); err != nil {
		return nil, fmt.Errorf("list sre incidents: %w", err)
	}
	out := make([]*sresmartbot.Incident, 0, len(models))
	for _, model := range models {
		incident := incidentModelToDomain(model)
		out = append(out, &incident)
	}
	return out, nil
}

func (r *SRESmartBotRepository) CountIncidents(ctx context.Context, filter sresmartbot.IncidentFilter) (int, error) {
	query := `SELECT COUNT(*) FROM sre_incidents`
	clauses := make([]string, 0, 4)
	args := make([]interface{}, 0, 5)
	arg := 1
	if filter.TenantID != nil {
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", arg))
		args = append(args, *filter.TenantID)
		arg++
	}
	if strings.TrimSpace(filter.Domain) != "" {
		clauses = append(clauses, fmt.Sprintf("domain = $%d", arg))
		args = append(args, strings.TrimSpace(filter.Domain))
		arg++
	}
	if strings.TrimSpace(filter.Status) != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", arg))
		args = append(args, strings.TrimSpace(filter.Status))
		arg++
	}
	if strings.TrimSpace(filter.Severity) != "" {
		clauses = append(clauses, fmt.Sprintf("severity = $%d", arg))
		args = append(args, strings.TrimSpace(filter.Severity))
		arg++
	}
	if strings.TrimSpace(filter.Search) != "" {
		clauses = append(clauses, fmt.Sprintf("(display_name ILIKE $%d OR summary ILIKE $%d OR incident_type ILIKE $%d)", arg, arg, arg))
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		arg++
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	var count int
	if err := r.db.GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("count sre incidents: %w", err)
	}
	return count, nil
}

func (r *SRESmartBotRepository) CreateFinding(ctx context.Context, finding *sresmartbot.Finding) error {
	query := `
		INSERT INTO sre_findings (
			id, incident_id, source, signal_type, signal_key, severity, confidence,
			title, message, raw_payload, occurred_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
	_, err := r.db.ExecContext(ctx, query,
		finding.ID, finding.IncidentID, finding.Source, finding.SignalType, finding.SignalKey,
		string(finding.Severity), string(finding.Confidence), finding.Title, finding.Message,
		normalizeJSON(finding.RawPayload), finding.OccurredAt, finding.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create sre finding: %w", err)
	}
	return nil
}

func (r *SRESmartBotRepository) ListFindingsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*sresmartbot.Finding, error) {
	models := make([]sreFindingModel, 0)
	query := `
		SELECT id, incident_id, source, signal_type, signal_key, severity, confidence,
		       title, message, raw_payload, occurred_at, created_at
		FROM sre_findings
		WHERE incident_id = $1
		ORDER BY occurred_at DESC`
	if err := r.db.SelectContext(ctx, &models, query, incidentID); err != nil {
		return nil, fmt.Errorf("list sre findings: %w", err)
	}
	out := make([]*sresmartbot.Finding, 0, len(models))
	for _, model := range models {
		finding := findingModelToDomain(model)
		out = append(out, &finding)
	}
	return out, nil
}

func (r *SRESmartBotRepository) AddEvidence(ctx context.Context, evidence *sresmartbot.Evidence) error {
	query := `
		INSERT INTO sre_incident_evidence (
			id, incident_id, evidence_type, summary, payload, captured_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7)`
	_, err := r.db.ExecContext(ctx, query,
		evidence.ID, evidence.IncidentID, evidence.EvidenceType, evidence.Summary,
		normalizeJSON(evidence.Payload), evidence.CapturedAt, evidence.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("add sre evidence: %w", err)
	}
	return nil
}

func (r *SRESmartBotRepository) ListEvidenceByIncident(ctx context.Context, incidentID uuid.UUID) ([]*sresmartbot.Evidence, error) {
	models := make([]sreEvidenceModel, 0)
	query := `
		SELECT id, incident_id, evidence_type, summary, payload, captured_at, created_at
		FROM sre_incident_evidence
		WHERE incident_id = $1
		ORDER BY captured_at DESC`
	if err := r.db.SelectContext(ctx, &models, query, incidentID); err != nil {
		return nil, fmt.Errorf("list sre evidence: %w", err)
	}
	out := make([]*sresmartbot.Evidence, 0, len(models))
	for _, model := range models {
		evidence := evidenceModelToDomain(model)
		out = append(out, &evidence)
	}
	return out, nil
}

func (r *SRESmartBotRepository) CreateActionAttempt(ctx context.Context, attempt *sresmartbot.ActionAttempt) error {
	query := `
		INSERT INTO sre_action_attempts (
			id, incident_id, action_key, action_class, target_kind, target_ref, status,
			actor_type, actor_id, approval_required, requested_at, started_at, completed_at,
			error_message, result_payload, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`
	_, err := r.db.ExecContext(ctx, query,
		attempt.ID, attempt.IncidentID, attempt.ActionKey, attempt.ActionClass, attempt.TargetKind, attempt.TargetRef,
		attempt.Status, attempt.ActorType, nullableOptionalString(attempt.ActorID), attempt.ApprovalRequired, attempt.RequestedAt,
		attempt.StartedAt, attempt.CompletedAt, nullableOptionalString(attempt.ErrorMessage), normalizeJSON(attempt.ResultPayload),
		attempt.CreatedAt, attempt.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create sre action attempt: %w", err)
	}
	return nil
}

func (r *SRESmartBotRepository) UpdateActionAttempt(ctx context.Context, attempt *sresmartbot.ActionAttempt) error {
	query := `
		UPDATE sre_action_attempts
		SET status = $2,
		    actor_type = $3,
		    actor_id = $4,
		    approval_required = $5,
		    requested_at = $6,
		    started_at = $7,
		    completed_at = $8,
		    error_message = $9,
		    result_payload = $10,
		    updated_at = $11
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query,
		attempt.ID, attempt.Status, attempt.ActorType, nullableOptionalString(attempt.ActorID), attempt.ApprovalRequired,
		attempt.RequestedAt, attempt.StartedAt, attempt.CompletedAt, nullableOptionalString(attempt.ErrorMessage),
		normalizeJSON(attempt.ResultPayload), attempt.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update sre action attempt: %w", err)
	}
	return nil
}

func (r *SRESmartBotRepository) ListActionAttemptsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*sresmartbot.ActionAttempt, error) {
	models := make([]sreActionAttemptModel, 0)
	query := `
		SELECT id, incident_id, action_key, action_class, target_kind, target_ref, status,
		       actor_type, actor_id, approval_required, requested_at, started_at, completed_at,
		       error_message, result_payload, created_at, updated_at
		FROM sre_action_attempts
		WHERE incident_id = $1
		ORDER BY requested_at DESC`
	if err := r.db.SelectContext(ctx, &models, query, incidentID); err != nil {
		return nil, fmt.Errorf("list sre action attempts: %w", err)
	}
	out := make([]*sresmartbot.ActionAttempt, 0, len(models))
	for _, model := range models {
		action := actionAttemptModelToDomain(model)
		out = append(out, &action)
	}
	return out, nil
}

func (r *SRESmartBotRepository) CreateApproval(ctx context.Context, approval *sresmartbot.Approval) error {
	query := `
		INSERT INTO sre_approvals (
			id, incident_id, action_attempt_id, channel_provider_id, status,
			request_message, requested_by, decided_by, decision_comment,
			requested_at, decided_at, expires_at, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`
	_, err := r.db.ExecContext(ctx, query,
		approval.ID, approval.IncidentID, approval.ActionAttemptID, approval.ChannelProviderID, approval.Status,
		approval.RequestMessage, nullableOptionalString(approval.RequestedBy), nullableOptionalString(approval.DecidedBy),
		nullableOptionalString(approval.DecisionComment), approval.RequestedAt, approval.DecidedAt, approval.ExpiresAt,
		approval.CreatedAt, approval.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create sre approval: %w", err)
	}
	return nil
}

func (r *SRESmartBotRepository) UpdateApproval(ctx context.Context, approval *sresmartbot.Approval) error {
	query := `
		UPDATE sre_approvals
		SET channel_provider_id = $2,
		    status = $3,
		    request_message = $4,
		    requested_by = $5,
		    decided_by = $6,
		    decision_comment = $7,
		    requested_at = $8,
		    decided_at = $9,
		    expires_at = $10,
		    updated_at = $11
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query,
		approval.ID, approval.ChannelProviderID, approval.Status, approval.RequestMessage,
		nullableOptionalString(approval.RequestedBy), nullableOptionalString(approval.DecidedBy), nullableOptionalString(approval.DecisionComment),
		approval.RequestedAt, approval.DecidedAt, approval.ExpiresAt, approval.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update sre approval: %w", err)
	}
	return nil
}

func (r *SRESmartBotRepository) ListApprovalsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*sresmartbot.Approval, error) {
	models := make([]sreApprovalModel, 0)
	query := `
		SELECT id, incident_id, action_attempt_id, channel_provider_id, status,
		       request_message, requested_by, decided_by, decision_comment,
		       requested_at, decided_at, expires_at, created_at, updated_at
		FROM sre_approvals
		WHERE incident_id = $1
		ORDER BY requested_at DESC`
	if err := r.db.SelectContext(ctx, &models, query, incidentID); err != nil {
		return nil, fmt.Errorf("list sre approvals: %w", err)
	}
	out := make([]*sresmartbot.Approval, 0, len(models))
	for _, model := range models {
		approval := approvalModelToDomain(model)
		out = append(out, &approval)
	}
	return out, nil
}

func (r *SRESmartBotRepository) ListApprovals(ctx context.Context, filter sresmartbot.ApprovalFilter) ([]*sresmartbot.Approval, error) {
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	query := `
		SELECT a.id, a.incident_id, a.action_attempt_id, a.channel_provider_id, a.status,
		       a.request_message, a.requested_by, a.decided_by, a.decision_comment,
		       a.requested_at, a.decided_at, a.expires_at, a.created_at, a.updated_at
		FROM sre_approvals a
		INNER JOIN sre_incidents i ON i.id = a.incident_id
		LEFT JOIN sre_action_attempts aa ON aa.id = a.action_attempt_id`
	clauses := make([]string, 0, 4)
	args := make([]interface{}, 0, 6)
	arg := 1
	if filter.TenantID != nil {
		clauses = append(clauses, fmt.Sprintf("i.tenant_id = $%d", arg))
		args = append(args, *filter.TenantID)
		arg++
	}
	if strings.TrimSpace(filter.Status) != "" {
		clauses = append(clauses, fmt.Sprintf("a.status = $%d", arg))
		args = append(args, strings.TrimSpace(filter.Status))
		arg++
	}
	if strings.TrimSpace(filter.ChannelProviderID) != "" {
		clauses = append(clauses, fmt.Sprintf("a.channel_provider_id = $%d", arg))
		args = append(args, strings.TrimSpace(filter.ChannelProviderID))
		arg++
	}
	if strings.TrimSpace(filter.Search) != "" {
		clauses = append(clauses, fmt.Sprintf("(a.request_message ILIKE $%d OR i.display_name ILIKE $%d OR i.incident_type ILIKE $%d OR aa.action_key ILIKE $%d OR aa.target_ref ILIKE $%d)", arg, arg, arg, arg, arg))
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		arg++
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY CASE WHEN a.status = 'pending' THEN 0 ELSE 1 END, a.requested_at DESC LIMIT $%d OFFSET $%d", arg, arg+1)
	args = append(args, filter.Limit, filter.Offset)

	models := make([]sreApprovalModel, 0)
	if err := r.db.SelectContext(ctx, &models, query, args...); err != nil {
		return nil, fmt.Errorf("list sre approvals: %w", err)
	}
	out := make([]*sresmartbot.Approval, 0, len(models))
	for _, model := range models {
		approval := approvalModelToDomain(model)
		out = append(out, &approval)
	}
	return out, nil
}

func (r *SRESmartBotRepository) CountApprovals(ctx context.Context, filter sresmartbot.ApprovalFilter) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM sre_approvals a
		INNER JOIN sre_incidents i ON i.id = a.incident_id
		LEFT JOIN sre_action_attempts aa ON aa.id = a.action_attempt_id`
	clauses := make([]string, 0, 4)
	args := make([]interface{}, 0, 4)
	arg := 1
	if filter.TenantID != nil {
		clauses = append(clauses, fmt.Sprintf("i.tenant_id = $%d", arg))
		args = append(args, *filter.TenantID)
		arg++
	}
	if strings.TrimSpace(filter.Status) != "" {
		clauses = append(clauses, fmt.Sprintf("a.status = $%d", arg))
		args = append(args, strings.TrimSpace(filter.Status))
		arg++
	}
	if strings.TrimSpace(filter.ChannelProviderID) != "" {
		clauses = append(clauses, fmt.Sprintf("a.channel_provider_id = $%d", arg))
		args = append(args, strings.TrimSpace(filter.ChannelProviderID))
		arg++
	}
	if strings.TrimSpace(filter.Search) != "" {
		clauses = append(clauses, fmt.Sprintf("(a.request_message ILIKE $%d OR i.display_name ILIKE $%d OR i.incident_type ILIKE $%d OR aa.action_key ILIKE $%d OR aa.target_ref ILIKE $%d)", arg, arg, arg, arg, arg))
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		arg++
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	var count int
	if err := r.db.GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("count sre approvals: %w", err)
	}
	return count, nil
}

func (r *SRESmartBotRepository) CreateDetectorRuleSuggestion(ctx context.Context, suggestion *sresmartbot.DetectorRuleSuggestion) error {
	query := `
		INSERT INTO sre_detector_rule_suggestions (
			id, tenant_id, incident_id, fingerprint, name, description, query, threshold,
			domain, incident_type, severity, confidence, signal_key, source, status, auto_created,
			reason, evidence_payload, proposed_by, reviewed_by, reviewed_at, activated_rule_id, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24
		)`
	_, err := r.db.ExecContext(ctx, query,
		suggestion.ID, suggestion.TenantID, suggestion.IncidentID, suggestion.Fingerprint, suggestion.Name, suggestion.Description,
		suggestion.Query, suggestion.Threshold, suggestion.Domain, suggestion.IncidentType, string(suggestion.Severity),
		string(suggestion.Confidence), nullableOptionalString(suggestion.SignalKey), suggestion.Source, string(suggestion.Status),
		suggestion.AutoCreated, nullableOptionalString(suggestion.Reason), normalizeJSON(suggestion.EvidencePayload),
		nullableOptionalString(suggestion.ProposedBy), nullableOptionalString(suggestion.ReviewedBy), suggestion.ReviewedAt,
		nullableOptionalString(suggestion.ActivatedRuleID), suggestion.CreatedAt, suggestion.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create sre detector rule suggestion: %w", err)
	}
	return nil
}

func (r *SRESmartBotRepository) UpdateDetectorRuleSuggestion(ctx context.Context, suggestion *sresmartbot.DetectorRuleSuggestion) error {
	query := `
		UPDATE sre_detector_rule_suggestions
		SET tenant_id = $2,
		    incident_id = $3,
		    name = $4,
		    description = $5,
		    query = $6,
		    threshold = $7,
		    domain = $8,
		    incident_type = $9,
		    severity = $10,
		    confidence = $11,
		    signal_key = $12,
		    source = $13,
		    status = $14,
		    auto_created = $15,
		    reason = $16,
		    evidence_payload = $17,
		    proposed_by = $18,
		    reviewed_by = $19,
		    reviewed_at = $20,
		    activated_rule_id = $21,
		    updated_at = $22
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query,
		suggestion.ID, suggestion.TenantID, suggestion.IncidentID, suggestion.Name, suggestion.Description, suggestion.Query,
		suggestion.Threshold, suggestion.Domain, suggestion.IncidentType, string(suggestion.Severity), string(suggestion.Confidence),
		nullableOptionalString(suggestion.SignalKey), suggestion.Source, string(suggestion.Status), suggestion.AutoCreated,
		nullableOptionalString(suggestion.Reason), normalizeJSON(suggestion.EvidencePayload), nullableOptionalString(suggestion.ProposedBy),
		nullableOptionalString(suggestion.ReviewedBy), suggestion.ReviewedAt, nullableOptionalString(suggestion.ActivatedRuleID), suggestion.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update sre detector rule suggestion: %w", err)
	}
	return nil
}

func (r *SRESmartBotRepository) GetDetectorRuleSuggestion(ctx context.Context, id uuid.UUID) (*sresmartbot.DetectorRuleSuggestion, error) {
	var model sreDetectorRuleSuggestionModel
	query := `
		SELECT id, tenant_id, incident_id, fingerprint, name, description, query, threshold,
		       domain, incident_type, severity, confidence, signal_key, source, status, auto_created,
		       reason, evidence_payload, proposed_by, reviewed_by, reviewed_at, activated_rule_id, created_at, updated_at
		FROM sre_detector_rule_suggestions
		WHERE id = $1`
	if err := r.db.GetContext(ctx, &model, query, id); err != nil {
		return nil, fmt.Errorf("get sre detector rule suggestion: %w", err)
	}
	suggestion := detectorRuleSuggestionModelToDomain(model)
	return &suggestion, nil
}

func (r *SRESmartBotRepository) GetDetectorRuleSuggestionByFingerprint(ctx context.Context, fingerprint string) (*sresmartbot.DetectorRuleSuggestion, error) {
	var model sreDetectorRuleSuggestionModel
	query := `
		SELECT id, tenant_id, incident_id, fingerprint, name, description, query, threshold,
		       domain, incident_type, severity, confidence, signal_key, source, status, auto_created,
		       reason, evidence_payload, proposed_by, reviewed_by, reviewed_at, activated_rule_id, created_at, updated_at
		FROM sre_detector_rule_suggestions
		WHERE fingerprint = $1`
	if err := r.db.GetContext(ctx, &model, query, strings.TrimSpace(fingerprint)); err != nil {
		return nil, fmt.Errorf("get sre detector rule suggestion by fingerprint: %w", err)
	}
	suggestion := detectorRuleSuggestionModelToDomain(model)
	return &suggestion, nil
}

func (r *SRESmartBotRepository) ListDetectorRuleSuggestions(ctx context.Context, filter sresmartbot.DetectorRuleSuggestionFilter) ([]*sresmartbot.DetectorRuleSuggestion, error) {
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	query := `
		SELECT id, tenant_id, incident_id, fingerprint, name, description, query, threshold,
		       domain, incident_type, severity, confidence, signal_key, source, status, auto_created,
		       reason, evidence_payload, proposed_by, reviewed_by, reviewed_at, activated_rule_id, created_at, updated_at
		FROM sre_detector_rule_suggestions`
	clauses := make([]string, 0, 3)
	args := make([]interface{}, 0, 5)
	arg := 1
	if filter.TenantID != nil {
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", arg))
		args = append(args, *filter.TenantID)
		arg++
	}
	if strings.TrimSpace(filter.Status) != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", arg))
		args = append(args, strings.TrimSpace(filter.Status))
		arg++
	}
	if strings.TrimSpace(filter.Search) != "" {
		clauses = append(clauses, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d OR incident_type ILIKE $%d OR signal_key ILIKE $%d)", arg, arg, arg, arg))
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		arg++
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY CASE WHEN status = 'pending' THEN 0 ELSE 1 END, created_at DESC LIMIT $%d OFFSET $%d", arg, arg+1)
	args = append(args, filter.Limit, filter.Offset)

	models := make([]sreDetectorRuleSuggestionModel, 0)
	if err := r.db.SelectContext(ctx, &models, query, args...); err != nil {
		return nil, fmt.Errorf("list sre detector rule suggestions: %w", err)
	}
	out := make([]*sresmartbot.DetectorRuleSuggestion, 0, len(models))
	for _, model := range models {
		suggestion := detectorRuleSuggestionModelToDomain(model)
		out = append(out, &suggestion)
	}
	return out, nil
}

func (r *SRESmartBotRepository) CreateRemediationPackRun(ctx context.Context, run *sresmartbot.RemediationPackRun) error {
	query := `
		INSERT INTO sre_remediation_pack_runs (
			id, tenant_id, incident_id, pack_key, pack_version, run_kind, status,
			requested_by, request_id, approval_id, action_attempt_id, summary,
			result_payload, started_at, completed_at, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17
		)`
	_, err := r.db.ExecContext(ctx, query,
		run.ID,
		run.TenantID,
		run.IncidentID,
		run.PackKey,
		run.PackVersion,
		run.RunKind,
		run.Status,
		nullableOptionalString(run.RequestedBy),
		nullableOptionalString(run.RequestID),
		run.ApprovalID,
		run.ActionAttemptID,
		run.Summary,
		normalizeJSON(run.ResultPayload),
		run.StartedAt,
		run.CompletedAt,
		run.CreatedAt,
		run.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create remediation pack run: %w", err)
	}
	return nil
}

func (r *SRESmartBotRepository) ListRemediationPackRunsByIncident(ctx context.Context, incidentID uuid.UUID) ([]*sresmartbot.RemediationPackRun, error) {
	models := make([]sreRemediationPackRunModel, 0)
	query := `
		SELECT id, tenant_id, incident_id, pack_key, pack_version, run_kind, status,
		       requested_by, request_id, approval_id, action_attempt_id, summary,
		       result_payload, started_at, completed_at, created_at, updated_at
		FROM sre_remediation_pack_runs
		WHERE incident_id = $1
		ORDER BY created_at DESC`
	if err := r.db.SelectContext(ctx, &models, query, incidentID); err != nil {
		return nil, fmt.Errorf("list remediation pack runs by incident: %w", err)
	}

	out := make([]*sresmartbot.RemediationPackRun, 0, len(models))
	for _, model := range models {
		run := remediationPackRunModelToDomain(model)
		out = append(out, &run)
	}
	return out, nil
}

func incidentModelToDomain(model sreIncidentModel) sresmartbot.Incident {
	return sresmartbot.Incident{
		ID:              model.ID,
		TenantID:        model.TenantID,
		CorrelationKey:  model.CorrelationKey,
		Domain:          model.Domain,
		IncidentType:    model.IncidentType,
		DisplayName:     model.DisplayName,
		Summary:         model.Summary,
		Severity:        sresmartbot.IncidentSeverity(model.Severity),
		Confidence:      sresmartbot.IncidentConfidence(model.Confidence),
		Status:          sresmartbot.IncidentStatus(model.Status),
		Source:          model.Source,
		FirstObservedAt: model.FirstObservedAt,
		LastObservedAt:  model.LastObservedAt,
		ContainedAt:     model.ContainedAt,
		ResolvedAt:      model.ResolvedAt,
		SuppressedUntil: model.SuppressedUntil,
		Metadata:        model.Metadata,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}

func findingModelToDomain(model sreFindingModel) sresmartbot.Finding {
	return sresmartbot.Finding{
		ID:         model.ID,
		IncidentID: model.IncidentID,
		Source:     model.Source,
		SignalType: model.SignalType,
		SignalKey:  model.SignalKey,
		Severity:   sresmartbot.IncidentSeverity(model.Severity),
		Confidence: sresmartbot.IncidentConfidence(model.Confidence),
		Title:      model.Title,
		Message:    model.Message,
		RawPayload: model.RawPayload,
		OccurredAt: model.OccurredAt,
		CreatedAt:  model.CreatedAt,
	}
}

func evidenceModelToDomain(model sreEvidenceModel) sresmartbot.Evidence {
	return sresmartbot.Evidence{
		ID:           model.ID,
		IncidentID:   model.IncidentID,
		EvidenceType: model.EvidenceType,
		Summary:      model.Summary,
		Payload:      model.Payload,
		CapturedAt:   model.CapturedAt,
		CreatedAt:    model.CreatedAt,
	}
}

func actionAttemptModelToDomain(model sreActionAttemptModel) sresmartbot.ActionAttempt {
	return sresmartbot.ActionAttempt{
		ID:               model.ID,
		IncidentID:       model.IncidentID,
		ActionKey:        model.ActionKey,
		ActionClass:      model.ActionClass,
		TargetKind:       model.TargetKind,
		TargetRef:        model.TargetRef,
		Status:           model.Status,
		ActorType:        model.ActorType,
		ActorID:          model.ActorID.String,
		ApprovalRequired: model.ApprovalRequired,
		RequestedAt:      model.RequestedAt,
		StartedAt:        model.StartedAt,
		CompletedAt:      model.CompletedAt,
		ErrorMessage:     model.ErrorMessage.String,
		ResultPayload:    model.ResultPayload,
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}
}

func approvalModelToDomain(model sreApprovalModel) sresmartbot.Approval {
	return sresmartbot.Approval{
		ID:                model.ID,
		IncidentID:        model.IncidentID,
		ActionAttemptID:   model.ActionAttemptID,
		ChannelProviderID: model.ChannelProviderID,
		Status:            model.Status,
		RequestMessage:    model.RequestMessage,
		RequestedBy:       model.RequestedBy.String,
		DecidedBy:         model.DecidedBy.String,
		DecisionComment:   model.DecisionComment.String,
		RequestedAt:       model.RequestedAt,
		DecidedAt:         model.DecidedAt,
		ExpiresAt:         model.ExpiresAt,
		CreatedAt:         model.CreatedAt,
		UpdatedAt:         model.UpdatedAt,
	}
}

func detectorRuleSuggestionModelToDomain(model sreDetectorRuleSuggestionModel) sresmartbot.DetectorRuleSuggestion {
	return sresmartbot.DetectorRuleSuggestion{
		ID:              model.ID,
		TenantID:        model.TenantID,
		IncidentID:      model.IncidentID,
		Fingerprint:     model.Fingerprint,
		Name:            model.Name,
		Description:     model.Description,
		Query:           model.Query,
		Threshold:       model.Threshold,
		Domain:          model.Domain,
		IncidentType:    model.IncidentType,
		Severity:        sresmartbot.IncidentSeverity(model.Severity),
		Confidence:      sresmartbot.IncidentConfidence(model.Confidence),
		SignalKey:       model.SignalKey.String,
		Source:          model.Source,
		Status:          sresmartbot.DetectorRuleSuggestionStatus(model.Status),
		AutoCreated:     model.AutoCreated,
		Reason:          model.Reason.String,
		EvidencePayload: model.EvidencePayload,
		ProposedBy:      model.ProposedBy.String,
		ReviewedBy:      model.ReviewedBy.String,
		ReviewedAt:      model.ReviewedAt,
		ActivatedRuleID: model.ActivatedRuleID.String,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}

func remediationPackRunModelToDomain(model sreRemediationPackRunModel) sresmartbot.RemediationPackRun {
	return sresmartbot.RemediationPackRun{
		ID:              model.ID,
		TenantID:        model.TenantID,
		IncidentID:      model.IncidentID,
		PackKey:         model.PackKey,
		PackVersion:     model.PackVersion,
		RunKind:         model.RunKind,
		Status:          model.Status,
		RequestedBy:     model.RequestedBy.String,
		RequestID:       model.RequestID.String,
		ApprovalID:      model.ApprovalID,
		ActionAttemptID: model.ActionAttemptID,
		Summary:         model.Summary,
		ResultPayload:   model.ResultPayload,
		StartedAt:       model.StartedAt,
		CompletedAt:     model.CompletedAt,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}

func normalizeJSON(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte(`{}`)
	}
	return raw
}

func nullableOptionalString(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
