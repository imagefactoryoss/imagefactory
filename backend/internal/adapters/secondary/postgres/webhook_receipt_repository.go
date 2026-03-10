package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

type WebhookReceipt struct {
	ID                  uuid.UUID `db:"id"`
	TenantID            uuid.UUID `db:"tenant_id"`
	ProjectID           uuid.UUID `db:"project_id"`
	Provider            string    `db:"provider"`
	DeliveryID          *string   `db:"delivery_id"`
	EventType           string    `db:"event_type"`
	EventRef            *string   `db:"event_ref"`
	EventBranch         *string   `db:"event_branch"`
	EventCommitSHA      *string   `db:"event_commit_sha"`
	RepoURL             *string   `db:"repo_url"`
	EventSHA            *string   `db:"event_sha"`
	SignatureValid      bool      `db:"signature_valid"`
	Status              string    `db:"status"`
	Reason              *string   `db:"reason"`
	MatchedTriggerCount int       `db:"matched_trigger_count"`
	TriggeredBuildIDs   []string  `db:"-"`
	TriggeredBuildRaw   []byte    `db:"triggered_build_ids"`
	ReceivedAt          time.Time `db:"received_at"`
}

type WebhookReceiptRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewWebhookReceiptRepository(db *sqlx.DB, logger *zap.Logger) *WebhookReceiptRepository {
	return &WebhookReceiptRepository{db: db, logger: logger}
}

func (r *WebhookReceiptRepository) Save(ctx context.Context, receipt *WebhookReceipt) error {
	if receipt == nil {
		return fmt.Errorf("receipt is required")
	}
	if receipt.ID == uuid.Nil {
		receipt.ID = uuid.New()
	}
	buildIDsJSON, _ := json.Marshal(receipt.TriggeredBuildIDs)

	query := `
		INSERT INTO webhook_receipts (
			id, tenant_id, project_id, provider, delivery_id, event_type, event_ref, event_branch, event_commit_sha, repo_url, event_sha,
			signature_valid, status, reason, matched_trigger_count, triggered_build_ids, received_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17
		)`

	_, err := r.db.ExecContext(ctx, query,
		receipt.ID, receipt.TenantID, receipt.ProjectID, receipt.Provider, receipt.DeliveryID, receipt.EventType, receipt.EventRef, receipt.EventBranch, receipt.EventCommitSHA, receipt.RepoURL, receipt.EventSHA,
		receipt.SignatureValid, receipt.Status, receipt.Reason, receipt.MatchedTriggerCount, buildIDsJSON, receipt.ReceivedAt,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *WebhookReceiptRepository) FindByProjectProviderDelivery(ctx context.Context, projectID uuid.UUID, provider, deliveryID string) (*WebhookReceipt, error) {
	query := `
		SELECT id, tenant_id, project_id, provider, delivery_id, event_type, event_ref, event_branch, event_commit_sha, repo_url, event_sha,
		       signature_valid, status, reason, matched_trigger_count, triggered_build_ids, received_at
		FROM webhook_receipts
		WHERE project_id = $1 AND provider = $2 AND delivery_id = $3
		LIMIT 1`

	var receipt WebhookReceipt
	if err := r.db.GetContext(ctx, &receipt, query, projectID, provider, deliveryID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if len(receipt.TriggeredBuildRaw) > 0 {
		_ = json.Unmarshal(receipt.TriggeredBuildRaw, &receipt.TriggeredBuildIDs)
	}
	return &receipt, nil
}

func (r *WebhookReceiptRepository) ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]WebhookReceipt, error) {
	query := `
		SELECT id, tenant_id, project_id, provider, delivery_id, event_type, event_ref, event_branch, event_commit_sha, repo_url, event_sha,
		       signature_valid, status, reason, matched_trigger_count, triggered_build_ids, received_at
		FROM webhook_receipts
		WHERE project_id = $1
		ORDER BY received_at DESC
		LIMIT $2 OFFSET $3`

	var receipts []WebhookReceipt
	if err := r.db.SelectContext(ctx, &receipts, query, projectID, limit, offset); err != nil {
		return nil, err
	}
	for i := range receipts {
		if len(receipts[i].TriggeredBuildRaw) > 0 {
			_ = json.Unmarshal(receipts[i].TriggeredBuildRaw, &receipts[i].TriggeredBuildIDs)
		}
	}
	return receipts, nil
}

func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	pqErr, ok := err.(*pq.Error)
	return ok && pqErr.Code == "23505"
}
