package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/infrastructure/audit"
)

// AuditRepository implements the audit.Repository interface
type AuditRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(db *sqlx.DB, logger *zap.Logger) *AuditRepository {
	return &AuditRepository{
		db:     db,
		logger: logger,
	}
}

// auditEventModel represents the database model for audit events
type auditEventModel struct {
	ID        uuid.UUID      `db:"id"`
	TenantID  *uuid.UUID     `db:"tenant_id"`
	UserID    *uuid.UUID     `db:"user_id"`
	UserName  sql.NullString `db:"user_name"`
	EventType string         `db:"event_type"`
	Severity  string         `db:"severity"`
	Resource  string         `db:"resource"`
	Action    string         `db:"action"`
	IPAddress sql.NullString `db:"ip_address"`
	UserAgent sql.NullString `db:"user_agent"`
	Details   sql.NullString `db:"details"`
	Message   string         `db:"message"`
	Timestamp time.Time      `db:"timestamp"`
}

// SaveEvent persists an audit event
func (r *AuditRepository) SaveEvent(ctx context.Context, event *audit.AuditEvent) error {
	// Convert details map to JSON
	var detailsJSON interface{}
	if event.Details != nil {
		var err error
		detailsBytes, err := json.Marshal(event.Details)
		if err != nil {
			r.logger.Error("Failed to marshal audit event details", zap.Error(err))
			return fmt.Errorf("failed to marshal audit event details: %w", err)
		}
		detailsJSON = string(detailsBytes)
	} else {
		detailsJSON = nil
	}

	query := `
		INSERT INTO audit_events (
			id, tenant_id, user_id, event_type, severity, resource, action,
			ip_address, user_agent, details, message, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.ExecContext(ctx, query,
		event.ID,
		event.TenantID,
		event.UserID,
		string(event.EventType),
		string(event.Severity),
		event.Resource,
		event.Action,
		nullString(event.IPAddress),
		nullString(event.UserAgent),
		detailsJSON,
		event.Message,
		event.Timestamp,
	)

	if err != nil {
		tenantIDStr := "null"
		if event.TenantID != nil {
			tenantIDStr = event.TenantID.String()
		}
		r.logger.Error("Failed to save audit event",
			zap.String("event_id", event.ID.String()),
			zap.String("tenant_id", tenantIDStr),
			zap.String("event_type", string(event.EventType)),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save audit event: %w", err)
	}

	tenantIDStr := "null"
	if event.TenantID != nil {
		tenantIDStr = event.TenantID.String()
	}
	r.logger.Debug("Audit event saved successfully",
		zap.String("event_id", event.ID.String()),
		zap.String("tenant_id", tenantIDStr),
	)

	return nil
}

// QueryEvents queries audit events with filtering
func (r *AuditRepository) QueryEvents(ctx context.Context, tenantID *uuid.UUID, filter audit.AuditEventFilter, limit, offset int) ([]*audit.AuditEvent, error) {
	query := `
		SELECT ae.id, ae.tenant_id, ae.user_id, 
			   CONCAT(u.first_name, ' ', u.last_name) as user_name,
			   ae.event_type, ae.severity, ae.resource, ae.action,
			   ae.ip_address, ae.user_agent, ae.details, ae.message, ae.timestamp
		FROM audit_events ae
		LEFT JOIN users u ON ae.user_id = u.id
	`
	args := []interface{}{}
	argCount := 0
	hasWhere := false

	// Add tenant filter if specified (nil means query all events)
	if tenantID != nil {
		argCount++
		query += fmt.Sprintf(" WHERE ae.tenant_id = $%d", argCount)
		args = append(args, *tenantID)
		hasWhere = true
	}

	// Add filters
	if filter.UserID != nil {
		argCount++
		if hasWhere {
			query += fmt.Sprintf(" AND ae.user_id = $%d", argCount)
		} else {
			query += fmt.Sprintf(" WHERE ae.user_id = $%d", argCount)
			hasWhere = true
		}
		args = append(args, *filter.UserID)
	}

	if filter.EventType != nil {
		argCount++
		if hasWhere {
			query += fmt.Sprintf(" AND ae.event_type = $%d", argCount)
		} else {
			query += fmt.Sprintf(" WHERE ae.event_type = $%d", argCount)
			hasWhere = true
		}
		args = append(args, string(*filter.EventType))
	}

	if filter.Severity != nil {
		argCount++
		if hasWhere {
			query += fmt.Sprintf(" AND ae.severity = $%d", argCount)
		} else {
			query += fmt.Sprintf(" WHERE ae.severity = $%d", argCount)
			hasWhere = true
		}
		args = append(args, string(*filter.Severity))
	}

	if filter.Resource != nil {
		argCount++
		if hasWhere {
			query += fmt.Sprintf(" AND ae.resource = $%d", argCount)
		} else {
			query += fmt.Sprintf(" WHERE ae.resource = $%d", argCount)
			hasWhere = true
		}
		args = append(args, *filter.Resource)
	}

	if filter.Action != nil {
		argCount++
		if hasWhere {
			query += fmt.Sprintf(" AND ae.action = $%d", argCount)
		} else {
			query += fmt.Sprintf(" WHERE ae.action = $%d", argCount)
			hasWhere = true
		}
		args = append(args, *filter.Action)
	}

	if filter.Search != nil && *filter.Search != "" {
		argCount++
		searchTerm := "%" + *filter.Search + "%"
		if hasWhere {
			query += fmt.Sprintf(" AND (ae.message ILIKE $%d OR CONCAT(u.first_name, ' ', u.last_name) ILIKE $%d)", argCount, argCount)
		} else {
			query += fmt.Sprintf(" WHERE (ae.message ILIKE $%d OR CONCAT(u.first_name, ' ', u.last_name) ILIKE $%d)", argCount, argCount)
			hasWhere = true
		}
		args = append(args, searchTerm)
	}

	if filter.StartTime != nil {
		argCount++
		if hasWhere {
			query += fmt.Sprintf(" AND ae.timestamp >= $%d", argCount)
		} else {
			query += fmt.Sprintf(" WHERE ae.timestamp >= $%d", argCount)
			hasWhere = true
		}
		args = append(args, *filter.StartTime)
	}

	if filter.EndTime != nil {
		argCount++
		if hasWhere {
			query += fmt.Sprintf(" AND ae.timestamp <= $%d", argCount)
		} else {
			query += fmt.Sprintf(" WHERE ae.timestamp <= $%d", argCount)
			hasWhere = true
		}
		args = append(args, *filter.EndTime)
	}

	// Add ordering and pagination
	query += " ORDER BY ae.timestamp DESC"
	if limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
	}
	if offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	var models []auditEventModel
	err := r.db.SelectContext(ctx, &models, query, args...)
	if err != nil {
		tenantIDStr := "null"
		if tenantID != nil {
			tenantIDStr = tenantID.String()
		}
		r.logger.Error("Failed to query audit events",
			zap.String("tenant_id", tenantIDStr),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to query audit events: %w", err)
	}

	events := make([]*audit.AuditEvent, len(models))
	for i, model := range models {
		event, err := r.modelToAuditEvent(&model)
		if err != nil {
			r.logger.Error("Failed to convert audit event model",
				zap.String("event_id", model.ID.String()),
				zap.Error(err),
			)
			continue
		}
		events[i] = event
	}

	return events, nil
}

// DeleteOldEvents removes audit events older than the specified time
func (r *AuditRepository) DeleteOldEvents(ctx context.Context, tenantID uuid.UUID, before time.Time) error {
	query := `DELETE FROM audit_events WHERE tenant_id = $1 AND timestamp < $2`

	result, err := r.db.ExecContext(ctx, query, tenantID, before)
	if err != nil {
		r.logger.Error("Failed to delete old audit events",
			zap.String("tenant_id", tenantID.String()),
			zap.Time("before", before),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete old audit events: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	r.logger.Info("Old audit events deleted",
		zap.String("tenant_id", tenantID.String()),
		zap.Int64("deleted_count", rowsAffected),
	)

	return nil
}

// CountEvents counts audit events matching the filter
func (r *AuditRepository) CountEvents(ctx context.Context, tenantID *uuid.UUID, filter audit.AuditEventFilter) (int, error) {
	query := `SELECT COUNT(*) FROM audit_events ae LEFT JOIN users u ON ae.user_id = u.id`
	args := []interface{}{}
	argCount := 0
	hasWhere := false

	// Add tenant filter if specified (nil means count all events)
	if tenantID != nil {
		argCount++
		query += fmt.Sprintf(" WHERE ae.tenant_id = $%d", argCount)
		args = append(args, *tenantID)
		hasWhere = true
	}

	// Add filters (same as QueryEvents)
	if filter.UserID != nil {
		argCount++
		if hasWhere {
			query += fmt.Sprintf(" AND ae.user_id = $%d", argCount)
		} else {
			query += fmt.Sprintf(" WHERE ae.user_id = $%d", argCount)
			hasWhere = true
		}
		args = append(args, *filter.UserID)
	}

	if filter.EventType != nil {
		argCount++
		if hasWhere {
			query += fmt.Sprintf(" AND ae.event_type = $%d", argCount)
		} else {
			query += fmt.Sprintf(" WHERE ae.event_type = $%d", argCount)
			hasWhere = true
		}
		args = append(args, string(*filter.EventType))
	}

	if filter.Severity != nil {
		argCount++
		if hasWhere {
			query += fmt.Sprintf(" AND ae.severity = $%d", argCount)
		} else {
			query += fmt.Sprintf(" WHERE ae.severity = $%d", argCount)
			hasWhere = true
		}
		args = append(args, string(*filter.Severity))
	}

	if filter.Resource != nil {
		argCount++
		if hasWhere {
			query += fmt.Sprintf(" AND ae.resource = $%d", argCount)
		} else {
			query += fmt.Sprintf(" WHERE ae.resource = $%d", argCount)
			hasWhere = true
		}
		args = append(args, *filter.Resource)
	}

	if filter.Action != nil {
		argCount++
		if hasWhere {
			query += fmt.Sprintf(" AND ae.action = $%d", argCount)
		} else {
			query += fmt.Sprintf(" WHERE ae.action = $%d", argCount)
			hasWhere = true
		}
		args = append(args, *filter.Action)
	}

	if filter.Search != nil && *filter.Search != "" {
		argCount++
		searchTerm := "%" + *filter.Search + "%"
		if hasWhere {
			query += fmt.Sprintf(" AND (ae.message ILIKE $%d OR CONCAT(u.first_name, ' ', u.last_name) ILIKE $%d)", argCount, argCount)
		} else {
			query += fmt.Sprintf(" WHERE (ae.message ILIKE $%d OR CONCAT(u.first_name, ' ', u.last_name) ILIKE $%d)", argCount, argCount)
			hasWhere = true
		}
		args = append(args, searchTerm)
	}

	if filter.StartTime != nil {
		argCount++
		if hasWhere {
			query += fmt.Sprintf(" AND ae.timestamp >= $%d", argCount)
		} else {
			query += fmt.Sprintf(" WHERE ae.timestamp >= $%d", argCount)
			hasWhere = true
		}
		args = append(args, *filter.StartTime)
	}

	if filter.EndTime != nil {
		argCount++
		if hasWhere {
			query += fmt.Sprintf(" AND ae.timestamp <= $%d", argCount)
		} else {
			query += fmt.Sprintf(" WHERE ae.timestamp <= $%d", argCount)
			hasWhere = true
		}
		args = append(args, *filter.EndTime)
	}

	var count int
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		tenantIDStr := "null"
		if tenantID != nil {
			tenantIDStr = tenantID.String()
		}
		r.logger.Error("Failed to count audit events",
			zap.String("tenant_id", tenantIDStr),
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to count audit events: %w", err)
	}

	return count, nil
}

// modelToAuditEvent converts a database model to an audit event
func (r *AuditRepository) modelToAuditEvent(model *auditEventModel) (*audit.AuditEvent, error) {
	// Parse details JSON
	var details map[string]interface{}
	if model.Details.Valid && model.Details.String != "" {
		if err := json.Unmarshal([]byte(model.Details.String), &details); err != nil {
			r.logger.Warn("Failed to unmarshal audit event details",
				zap.String("event_id", model.ID.String()),
				zap.Error(err),
			)
			// Continue without details rather than failing
		}
	}

	event := &audit.AuditEvent{
		ID:        model.ID,
		TenantID:  model.TenantID,
		UserID:    model.UserID,
		UserName:  nullStringValue(model.UserName),
		EventType: audit.AuditEventType(model.EventType),
		Severity:  audit.AuditEventSeverity(model.Severity),
		Resource:  model.Resource,
		Action:    model.Action,
		IPAddress: nullStringValue(model.IPAddress),
		UserAgent: nullStringValue(model.UserAgent),
		Details:   details,
		Message:   model.Message,
		Timestamp: model.Timestamp,
	}

	return event, nil
}

// nullString creates a sql.NullString from a string
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// nullStringValue extracts string value from sql.NullString
func nullStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
