package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/domain/eprregistration"
	"go.uber.org/zap"
)

type EPRRegistrationRequestRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewEPRRegistrationRequestRepository(db *sqlx.DB, logger *zap.Logger) *EPRRegistrationRequestRepository {
	return &EPRRegistrationRequestRepository{db: db, logger: logger}
}

type eprRegistrationRequestRow struct {
	ID                    uuid.UUID      `db:"id"`
	TenantID              uuid.UUID      `db:"tenant_id"`
	EPRRecordID           string         `db:"epr_record_id"`
	ProductName           string         `db:"product_name"`
	TechnologyName        string         `db:"technology_name"`
	BusinessJustification sql.NullString `db:"business_justification"`
	RequestedByUserID     uuid.UUID      `db:"requested_by_user_id"`
	Status                string         `db:"status"`
	DecidedByUserID       sql.NullString `db:"decided_by_user_id"`
	DecisionReason        sql.NullString `db:"decision_reason"`
	DecidedAt             sql.NullTime   `db:"decided_at"`
	ApprovedAt            sql.NullTime   `db:"approved_at"`
	ExpiresAt             sql.NullTime   `db:"expires_at"`
	LifecycleStatus       string         `db:"lifecycle_status"`
	SuspensionReason      sql.NullString `db:"suspension_reason"`
	LastReviewedAt        sql.NullTime   `db:"last_reviewed_at"`
	CreatedAt             time.Time      `db:"created_at"`
	UpdatedAt             time.Time      `db:"updated_at"`
}

type eprLifecycleTransitionRow struct {
	RequestID         uuid.UUID `db:"id"`
	TenantID          uuid.UUID `db:"tenant_id"`
	EPRRecordID       string    `db:"epr_record_id"`
	RequestedByUserID uuid.UUID `db:"requested_by_user_id"`
	LifecycleStatus   string    `db:"lifecycle_status"`
}

func (r *EPRRegistrationRequestRepository) Create(ctx context.Context, req *eprregistration.Request) error {
	query := `
		INSERT INTO epr_registration_requests (
			id, tenant_id, epr_record_id, product_name, technology_name, business_justification,
			requested_by_user_id, status, decided_by_user_id, decision_reason, decided_at,
			approved_at, expires_at, lifecycle_status, suspension_reason, last_reviewed_at, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :epr_record_id, :product_name, :technology_name, :business_justification,
			:requested_by_user_id, :status, :decided_by_user_id, :decision_reason, :decided_at,
			:approved_at, :expires_at, :lifecycle_status, :suspension_reason, :last_reviewed_at, :created_at, :updated_at
		)`
	params := map[string]interface{}{
		"id":                     req.ID,
		"tenant_id":              req.TenantID,
		"epr_record_id":          req.EPRRecordID,
		"product_name":           req.ProductName,
		"technology_name":        req.TechnologyName,
		"business_justification": nullableString(req.BusinessJustification),
		"requested_by_user_id":   req.RequestedByUserID,
		"status":                 string(req.Status),
		"decided_by_user_id":     req.DecidedByUserID,
		"decision_reason":        nullableString(req.DecisionReason),
		"decided_at":             req.DecidedAt,
		"approved_at":            req.ApprovedAt,
		"expires_at":             req.ExpiresAt,
		"lifecycle_status":       string(req.LifecycleStatus),
		"suspension_reason":      nullableString(req.SuspensionReason),
		"last_reviewed_at":       req.LastReviewedAt,
		"created_at":             req.CreatedAt.UTC(),
		"updated_at":             req.UpdatedAt.UTC(),
	}
	if _, err := r.db.NamedExecContext(ctx, query, params); err != nil {
		return err
	}
	return nil
}

func (r *EPRRegistrationRequestRepository) GetByID(ctx context.Context, id uuid.UUID) (*eprregistration.Request, error) {
	query := `
		SELECT id, tenant_id, epr_record_id, product_name, technology_name, business_justification,
		       requested_by_user_id, status, decided_by_user_id, decision_reason, decided_at,
		       approved_at, expires_at, lifecycle_status, suspension_reason, last_reviewed_at, created_at, updated_at
		FROM epr_registration_requests
		WHERE id = $1`
	var row eprRegistrationRequestRow
	if err := r.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, eprregistration.ErrNotFound
		}
		return nil, err
	}
	return mapEPRRegistrationRow(row)
}

func (r *EPRRegistrationRequestRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, status *eprregistration.Status, limit, offset int) ([]*eprregistration.Request, error) {
	query := `
		SELECT id, tenant_id, epr_record_id, product_name, technology_name, business_justification,
		       requested_by_user_id, status, decided_by_user_id, decision_reason, decided_at,
		       approved_at, expires_at, lifecycle_status, suspension_reason, last_reviewed_at, created_at, updated_at
		FROM epr_registration_requests
		WHERE tenant_id = $1
		  AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`
	statusValue := ""
	if status != nil {
		statusValue = string(*status)
	}
	rows := make([]eprRegistrationRequestRow, 0)
	if err := r.db.SelectContext(ctx, &rows, query, tenantID, statusValue, limit, offset); err != nil {
		return nil, err
	}
	return mapEPRRegistrationRows(rows)
}

func (r *EPRRegistrationRequestRepository) ListAll(ctx context.Context, status *eprregistration.Status, limit, offset int) ([]*eprregistration.Request, error) {
	query := `
		SELECT id, tenant_id, epr_record_id, product_name, technology_name, business_justification,
		       requested_by_user_id, status, decided_by_user_id, decision_reason, decided_at,
		       approved_at, expires_at, lifecycle_status, suspension_reason, last_reviewed_at, created_at, updated_at
		FROM epr_registration_requests
		WHERE ($1 = '' OR status = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	statusValue := ""
	if status != nil {
		statusValue = string(*status)
	}
	rows := make([]eprRegistrationRequestRow, 0)
	if err := r.db.SelectContext(ctx, &rows, query, statusValue, limit, offset); err != nil {
		return nil, err
	}
	return mapEPRRegistrationRows(rows)
}

func (r *EPRRegistrationRequestRepository) UpdateDecision(ctx context.Context, req *eprregistration.Request) error {
	query := `
		UPDATE epr_registration_requests
		SET status = $2,
		    decided_by_user_id = $3,
		    decision_reason = $4,
		    decided_at = $5,
		    approved_at = $6,
		    last_reviewed_at = $7,
		    updated_at = $8
		WHERE id = $1
		  AND status = 'pending'`
	result, err := r.db.ExecContext(
		ctx,
		query,
		req.ID,
		string(req.Status),
		nullableUUID(req.DecidedByUserID),
		nullableString(req.DecisionReason),
		req.DecidedAt,
		req.ApprovedAt,
		req.LastReviewedAt,
		req.UpdatedAt.UTC(),
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return eprregistration.ErrAlreadyDecided
	}
	return nil
}

func (r *EPRRegistrationRequestRepository) UpdateLifecycle(ctx context.Context, req *eprregistration.Request) error {
	query := `
		UPDATE epr_registration_requests
		SET lifecycle_status = $2,
		    suspension_reason = $3,
		    decision_reason = $4,
		    decided_by_user_id = $5,
		    last_reviewed_at = $6,
		    updated_at = $7
		WHERE id = $1
		  AND status = 'approved'`
	result, err := r.db.ExecContext(
		ctx,
		query,
		req.ID,
		string(req.LifecycleStatus),
		nullableString(req.SuspensionReason),
		nullableString(req.DecisionReason),
		nullableUUID(req.DecidedByUserID),
		req.LastReviewedAt,
		req.UpdatedAt.UTC(),
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return eprregistration.ErrLifecycleAction
	}
	return nil
}

func (r *EPRRegistrationRequestRepository) TransitionLifecycleStates(ctx context.Context, now time.Time, expiringBefore time.Time) (expiring []eprregistration.LifecycleTransitionRecord, expired []eprregistration.LifecycleTransitionRecord, err error) {
	expiringQuery := `
		UPDATE epr_registration_requests
		SET lifecycle_status = 'expiring',
		    updated_at = $2
		WHERE status = 'approved'
		  AND lifecycle_status = 'active'
		  AND expires_at IS NOT NULL
		  AND expires_at > $1
		  AND expires_at <= $3
		RETURNING id, tenant_id, epr_record_id, requested_by_user_id, lifecycle_status`
	expiredQuery := `
		UPDATE epr_registration_requests
		SET lifecycle_status = 'expired',
		    updated_at = $2
		WHERE status = 'approved'
		  AND lifecycle_status IN ('active', 'expiring')
		  AND expires_at IS NOT NULL
		  AND expires_at <= $1
		RETURNING id, tenant_id, epr_record_id, requested_by_user_id, lifecycle_status`

	expiringRows := make([]eprLifecycleTransitionRow, 0)
	if selectErr := r.db.SelectContext(ctx, &expiringRows, expiringQuery, now.UTC(), now.UTC(), expiringBefore.UTC()); selectErr != nil {
		return nil, nil, selectErr
	}
	expiredRows := make([]eprLifecycleTransitionRow, 0)
	if selectErr := r.db.SelectContext(ctx, &expiredRows, expiredQuery, now.UTC(), now.UTC()); selectErr != nil {
		return nil, nil, selectErr
	}

	expiring = make([]eprregistration.LifecycleTransitionRecord, 0, len(expiringRows))
	for _, row := range expiringRows {
		lifecycleStatus, parseErr := eprregistration.ParseLifecycleStatus(row.LifecycleStatus)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		expiring = append(expiring, eprregistration.LifecycleTransitionRecord{
			RequestID:         row.RequestID,
			TenantID:          row.TenantID,
			EPRRecordID:       row.EPRRecordID,
			RequestedByUserID: row.RequestedByUserID,
			LifecycleStatus:   lifecycleStatus,
		})
	}

	expired = make([]eprregistration.LifecycleTransitionRecord, 0, len(expiredRows))
	for _, row := range expiredRows {
		lifecycleStatus, parseErr := eprregistration.ParseLifecycleStatus(row.LifecycleStatus)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		expired = append(expired, eprregistration.LifecycleTransitionRecord{
			RequestID:         row.RequestID,
			TenantID:          row.TenantID,
			EPRRecordID:       row.EPRRecordID,
			RequestedByUserID: row.RequestedByUserID,
			LifecycleStatus:   lifecycleStatus,
		})
	}

	return expiring, expired, nil
}

func (r *EPRRegistrationRequestRepository) HasApprovedRegistration(ctx context.Context, tenantID uuid.UUID, eprRecordID string) (bool, error) {
	eprRecordID = strings.TrimSpace(eprRecordID)
	if eprRecordID == "" {
		return false, nil
	}
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM epr_registration_requests
			WHERE tenant_id = $1
			  AND epr_record_id = $2
			  AND status = 'approved'
			  AND lifecycle_status IN ('active', 'expiring')
		)`
	var exists bool
	if err := r.db.GetContext(ctx, &exists, query, tenantID, eprRecordID); err != nil {
		return false, fmt.Errorf("failed to check approved epr registration: %w", err)
	}
	return exists, nil
}

func (r *EPRRegistrationRequestRepository) GetApprovedRegistrationLifecycleStatus(ctx context.Context, tenantID uuid.UUID, eprRecordID string) (*eprregistration.LifecycleStatus, error) {
	eprRecordID = strings.TrimSpace(eprRecordID)
	if eprRecordID == "" {
		return nil, nil
	}
	query := `
		SELECT lifecycle_status
		FROM epr_registration_requests
		WHERE tenant_id = $1
		  AND epr_record_id = $2
		  AND status = 'approved'
		ORDER BY COALESCE(approved_at, decided_at, updated_at) DESC
		LIMIT 1`
	var raw string
	if err := r.db.GetContext(ctx, &raw, query, tenantID, eprRecordID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load approved epr lifecycle status: %w", err)
	}
	parsed, err := eprregistration.ParseLifecycleStatus(raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (r *EPRRegistrationRequestRepository) GetLifecycleMetrics(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'pending') AS pending,
			COUNT(*) FILTER (WHERE status = 'approved') AS approved,
			COUNT(*) FILTER (WHERE status = 'rejected') AS rejected,
			COUNT(*) FILTER (WHERE status = 'withdrawn') AS withdrawn,
			COUNT(*) FILTER (WHERE lifecycle_status = 'active') AS active,
			COUNT(*) FILTER (WHERE lifecycle_status = 'expiring') AS expiring,
			COUNT(*) FILTER (WHERE lifecycle_status = 'expired') AS expired,
			COUNT(*) FILTER (WHERE lifecycle_status = 'suspended') AS suspended
		FROM epr_registration_requests`
	row := struct {
		Total     int64 `db:"total"`
		Pending   int64 `db:"pending"`
		Approved  int64 `db:"approved"`
		Rejected  int64 `db:"rejected"`
		Withdrawn int64 `db:"withdrawn"`
		Active    int64 `db:"active"`
		Expiring  int64 `db:"expiring"`
		Expired   int64 `db:"expired"`
		Suspended int64 `db:"suspended"`
	}{}
	if err := r.db.GetContext(ctx, &row, query); err != nil {
		return nil, fmt.Errorf("failed to fetch epr lifecycle metrics: %w", err)
	}
	return map[string]int64{
		"total":     row.Total,
		"pending":   row.Pending,
		"approved":  row.Approved,
		"rejected":  row.Rejected,
		"withdrawn": row.Withdrawn,
		"active":    row.Active,
		"expiring":  row.Expiring,
		"expired":   row.Expired,
		"suspended": row.Suspended,
	}, nil
}

func mapEPRRegistrationRows(rows []eprRegistrationRequestRow) ([]*eprregistration.Request, error) {
	out := make([]*eprregistration.Request, 0, len(rows))
	for _, row := range rows {
		mapped, err := mapEPRRegistrationRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
}

func mapEPRRegistrationRow(row eprRegistrationRequestRow) (*eprregistration.Request, error) {
	status := eprregistration.Status(strings.ToLower(strings.TrimSpace(row.Status)))
	switch status {
	case eprregistration.StatusPending, eprregistration.StatusApproved, eprregistration.StatusRejected, eprregistration.StatusWithdrawn:
	default:
		return nil, fmt.Errorf("invalid epr registration status %q", row.Status)
	}

	var decidedBy *uuid.UUID
	if row.DecidedByUserID.Valid {
		parsed, err := uuid.Parse(row.DecidedByUserID.String)
		if err == nil {
			decidedBy = &parsed
		}
	}
	var decidedAt *time.Time
	if row.DecidedAt.Valid {
		t := row.DecidedAt.Time.UTC()
		decidedAt = &t
	}
	var approvedAt *time.Time
	if row.ApprovedAt.Valid {
		t := row.ApprovedAt.Time.UTC()
		approvedAt = &t
	}
	var expiresAt *time.Time
	if row.ExpiresAt.Valid {
		t := row.ExpiresAt.Time.UTC()
		expiresAt = &t
	}
	var lastReviewedAt *time.Time
	if row.LastReviewedAt.Valid {
		t := row.LastReviewedAt.Time.UTC()
		lastReviewedAt = &t
	}
	lifecycleStatus := eprregistration.LifecycleStatus(strings.ToLower(strings.TrimSpace(row.LifecycleStatus)))
	switch lifecycleStatus {
	case eprregistration.LifecycleStatusActive, eprregistration.LifecycleStatusExpiring, eprregistration.LifecycleStatusExpired, eprregistration.LifecycleStatusSuspended:
	default:
		return nil, fmt.Errorf("invalid epr lifecycle status %q", row.LifecycleStatus)
	}

	return &eprregistration.Request{
		ID:                    row.ID,
		TenantID:              row.TenantID,
		EPRRecordID:           row.EPRRecordID,
		ProductName:           row.ProductName,
		TechnologyName:        row.TechnologyName,
		BusinessJustification: nullStringToValue(row.BusinessJustification),
		RequestedByUserID:     row.RequestedByUserID,
		Status:                status,
		DecidedByUserID:       decidedBy,
		DecisionReason:        nullStringToValue(row.DecisionReason),
		DecidedAt:             decidedAt,
		ApprovedAt:            approvedAt,
		ExpiresAt:             expiresAt,
		LifecycleStatus:       lifecycleStatus,
		SuspensionReason:      nullStringToValue(row.SuspensionReason),
		LastReviewedAt:        lastReviewedAt,
		CreatedAt:             row.CreatedAt.UTC(),
		UpdatedAt:             row.UpdatedAt.UTC(),
	}, nil
}

func nullableUUID(id *uuid.UUID) interface{} {
	if id == nil || *id == uuid.Nil {
		return nil
	}
	return *id
}

func nullStringToValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
