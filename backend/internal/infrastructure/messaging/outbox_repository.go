package messaging

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type OutboxStore interface {
	Enqueue(ctx context.Context, event Event) error
	ClaimDue(ctx context.Context, limit int, claimOwner string, claimLease time.Duration) ([]OutboxMessage, error)
	MarkPublished(ctx context.Context, id uuid.UUID, claimOwner string) error
	MarkFailed(ctx context.Context, id uuid.UUID, claimOwner string, lastError string, nextAttemptAt time.Time) error
	PendingCount(ctx context.Context) (int64, error)
}

type OutboxMessage struct {
	ID              uuid.UUID
	Event           Event
	PublishAttempts int
}

type SQLOutboxStore struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewSQLOutboxStore(db *sqlx.DB, logger *zap.Logger) *SQLOutboxStore {
	return &SQLOutboxStore{db: db, logger: logger}
}

func (s *SQLOutboxStore) Enqueue(ctx context.Context, event Event) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("outbox store is not configured")
	}
	if event.Type == "" {
		return ErrEmptyEventType
	}
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal outbox payload: %w", err)
	}
	id, err := uuid.Parse(event.ID)
	if err != nil {
		id = uuid.New()
	}
	var tenantID *uuid.UUID
	if event.TenantID != "" {
		if parsed, parseErr := uuid.Parse(event.TenantID); parseErr == nil {
			tenantID = &parsed
		}
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO messaging_outbox (
			id, event_type, tenant_id, source, occurred_at, payload, schema_version,
			publish_attempts, next_attempt_at, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,0,NOW(),NOW(),NOW()
		)
	`, id, event.Type, tenantID, event.Source, event.OccurredAt.UTC(), payload, event.SchemaVersion)
	if err != nil {
		return fmt.Errorf("failed to enqueue outbox event: %w", err)
	}
	return nil
}

func (s *SQLOutboxStore) ClaimDue(ctx context.Context, limit int, claimOwner string, claimLease time.Duration) ([]OutboxMessage, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("outbox store is not configured")
	}
	if limit <= 0 {
		limit = 100
	}
	if claimLease <= 0 {
		claimLease = 30 * time.Second
	}
	claimLeaseSeconds := int(claimLease.Seconds())
	if claimLeaseSeconds <= 0 {
		claimLeaseSeconds = 30
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start outbox claim transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	rows, err := tx.QueryxContext(ctx, `
		WITH due AS (
			SELECT id
			FROM messaging_outbox
			WHERE published_at IS NULL
			  AND next_attempt_at <= NOW()
			  AND (claim_expires_at IS NULL OR claim_expires_at <= NOW())
			ORDER BY next_attempt_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE messaging_outbox mo
		SET claim_owner = $2,
			claim_expires_at = NOW() + ($3 * interval '1 second'),
			updated_at = NOW()
		FROM due
		WHERE mo.id = due.id
		RETURNING mo.id, mo.event_type, mo.tenant_id, mo.source, mo.occurred_at, mo.payload, mo.schema_version, mo.publish_attempts
	`, limit, claimOwner, claimLeaseSeconds)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch due outbox messages: %w", err)
	}
	defer rows.Close()

	out := make([]OutboxMessage, 0, limit)
	for rows.Next() {
		var (
			id              uuid.UUID
			eventType       string
			tenantID        sql.NullString
			source          sql.NullString
			occurredAt      time.Time
			payloadRaw      []byte
			schemaVersion   sql.NullString
			publishAttempts int
		)
		if scanErr := rows.Scan(&id, &eventType, &tenantID, &source, &occurredAt, &payloadRaw, &schemaVersion, &publishAttempts); scanErr != nil {
			return nil, fmt.Errorf("failed to scan outbox row: %w", scanErr)
		}
		payload := map[string]interface{}{}
		if len(payloadRaw) > 0 {
			_ = json.Unmarshal(payloadRaw, &payload)
		}
		event := Event{
			ID:            id.String(),
			Type:          eventType,
			TenantID:      tenantID.String,
			Source:        source.String,
			OccurredAt:    occurredAt.UTC(),
			SchemaVersion: schemaVersion.String,
			Payload:       payload,
		}
		out = append(out, OutboxMessage{
			ID:              id,
			Event:           event,
			PublishAttempts: publishAttempts,
		})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("failed iterating outbox rows: %w", rowsErr)
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return nil, fmt.Errorf("failed to commit outbox claim transaction: %w", commitErr)
	}
	return out, nil
}

func (s *SQLOutboxStore) MarkPublished(ctx context.Context, id uuid.UUID, claimOwner string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("outbox store is not configured")
	}
	query := `
		UPDATE messaging_outbox
		SET published_at = NOW(),
		    claim_owner = NULL,
		    claim_expires_at = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`
	args := []interface{}{id}
	if claimOwner != "" {
		query += " AND claim_owner = $2"
		args = append(args, claimOwner)
	}
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to mark outbox message published: %w", err)
	}
	return nil
}

func (s *SQLOutboxStore) MarkFailed(ctx context.Context, id uuid.UUID, claimOwner string, lastError string, nextAttemptAt time.Time) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("outbox store is not configured")
	}
	if nextAttemptAt.IsZero() {
		nextAttemptAt = time.Now().UTC().Add(5 * time.Second)
	}
	query := `
		UPDATE messaging_outbox
		SET publish_attempts = publish_attempts + 1,
		    last_error = $2,
		    next_attempt_at = $3,
		    claim_owner = NULL,
		    claim_expires_at = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`
	args := []interface{}{id, lastError, nextAttemptAt.UTC()}
	if claimOwner != "" {
		query += " AND claim_owner = $4"
		args = append(args, claimOwner)
	}
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to mark outbox message failed: %w", err)
	}
	return nil
}

func (s *SQLOutboxStore) PendingCount(ctx context.Context) (int64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("outbox store is not configured")
	}
	var count int64
	if err := s.db.GetContext(ctx, &count, `
		SELECT COUNT(*)
		FROM messaging_outbox
		WHERE published_at IS NULL
	`); err != nil {
		return 0, fmt.Errorf("failed to count pending outbox messages: %w", err)
	}
	return count, nil
}
