package messaging

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/srikarm/image-factory/internal/testutil"
)

func openOutboxTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("IF_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set IF_TEST_POSTGRES_DSN to run SQLOutboxStore DB tests")
	}
	testutil.RequireSafeTestDSN(t, dsn, "IF_TEST_POSTGRES_DSN")
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to connect test postgres: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createTempOutboxTable(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TEMP TABLE messaging_outbox (
			id UUID PRIMARY KEY,
			event_type TEXT NOT NULL,
			tenant_id UUID NULL,
			source TEXT NULL,
			occurred_at TIMESTAMPTZ NOT NULL,
			payload JSONB NOT NULL,
			schema_version TEXT NOT NULL,
			publish_attempts INT NOT NULL DEFAULT 0,
			last_error TEXT NULL,
			next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			published_at TIMESTAMPTZ NULL,
			claim_owner TEXT NULL,
			claim_expires_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("failed creating temp outbox table: %v", err)
	}
}

func TestSQLOutboxStoreEnqueuePersistsEvent(t *testing.T) {
	db := openOutboxTestDB(t)
	createTempOutboxTable(t, db)
	store := NewSQLOutboxStore(db, nil)

	eventID := uuid.New().String()
	tenantID := uuid.New().String()
	occurredAt := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Second)
	event := Event{
		ID:            eventID,
		Type:          EventTypeBuildExecutionCompleted,
		TenantID:      tenantID,
		Source:        "unit-test",
		OccurredAt:    occurredAt,
		SchemaVersion: "v1",
		Payload: map[string]interface{}{
			"build_id": uuid.New().String(),
			"status":   "completed",
		},
	}

	if err := store.Enqueue(context.Background(), event); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	var row struct {
		ID            uuid.UUID      `db:"id"`
		EventType     string         `db:"event_type"`
		TenantID      sql.NullString `db:"tenant_id"`
		Source        sql.NullString `db:"source"`
		OccurredAt    time.Time      `db:"occurred_at"`
		SchemaVersion string         `db:"schema_version"`
		Attempts      int            `db:"publish_attempts"`
		PublishedAt   sql.NullTime   `db:"published_at"`
	}
	if err := db.Get(&row, `
		SELECT id, event_type, tenant_id::text AS tenant_id, source, occurred_at, schema_version, publish_attempts, published_at
		FROM messaging_outbox
		WHERE id = $1
	`, eventID); err != nil {
		t.Fatalf("failed selecting enqueued row: %v", err)
	}

	if row.ID.String() != eventID {
		t.Fatalf("expected id %s, got %s", eventID, row.ID.String())
	}
	if row.EventType != event.Type {
		t.Fatalf("expected event type %s, got %s", event.Type, row.EventType)
	}
	if !row.TenantID.Valid || row.TenantID.String != tenantID {
		t.Fatalf("expected tenant_id %s, got %+v", tenantID, row.TenantID)
	}
	if !row.Source.Valid || row.Source.String != "unit-test" {
		t.Fatalf("expected source unit-test, got %+v", row.Source)
	}
	if !row.OccurredAt.UTC().Equal(occurredAt) {
		t.Fatalf("expected occurred_at %s, got %s", occurredAt.Format(time.RFC3339), row.OccurredAt.UTC().Format(time.RFC3339))
	}
	if row.SchemaVersion != "v1" {
		t.Fatalf("expected schema_version v1, got %s", row.SchemaVersion)
	}
	if row.Attempts != 0 {
		t.Fatalf("expected publish_attempts 0, got %d", row.Attempts)
	}
	if row.PublishedAt.Valid {
		t.Fatalf("expected published_at null, got %s", row.PublishedAt.Time.Format(time.RFC3339))
	}
}

func TestSQLOutboxStoreClaimDueHonorsLease(t *testing.T) {
	db := openOutboxTestDB(t)
	createTempOutboxTable(t, db)
	store := NewSQLOutboxStore(db, nil)

	dueID := uuid.New()
	notDueID := uuid.New()
	claimedID := uuid.New()
	_, err := db.Exec(`
		INSERT INTO messaging_outbox (id, event_type, occurred_at, payload, schema_version, next_attempt_at)
		VALUES
		($1, 'build.execution.completed', NOW(), '{}'::jsonb, 'v1', NOW() - INTERVAL '10 seconds'),
		($2, 'build.execution.completed', NOW(), '{}'::jsonb, 'v1', NOW() + INTERVAL '5 minutes'),
		($3, 'build.execution.completed', NOW(), '{}'::jsonb, 'v1', NOW() - INTERVAL '10 seconds')
	`, dueID, notDueID, claimedID)
	if err != nil {
		t.Fatalf("seed insert failed: %v", err)
	}
	_, err = db.Exec(`
		UPDATE messaging_outbox
		SET claim_owner = 'other-relay', claim_expires_at = NOW() + INTERVAL '1 minute'
		WHERE id = $1
	`, claimedID)
	if err != nil {
		t.Fatalf("failed seeding claimed row: %v", err)
	}

	messages, err := store.ClaimDue(context.Background(), 10, "relay-a", 30*time.Second)
	if err != nil {
		t.Fatalf("ClaimDue failed: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 claimed message, got %d", len(messages))
	}
	if messages[0].ID != dueID {
		t.Fatalf("expected claimed id %s, got %s", dueID, messages[0].ID)
	}

	var claimOwner sql.NullString
	if err := db.Get(&claimOwner, `SELECT claim_owner FROM messaging_outbox WHERE id = $1`, dueID); err != nil {
		t.Fatalf("failed reading claim owner: %v", err)
	}
	if !claimOwner.Valid || claimOwner.String != "relay-a" {
		t.Fatalf("expected claim_owner relay-a, got %+v", claimOwner)
	}
}

func TestSQLOutboxStoreMarkPublishedRespectsClaimOwner(t *testing.T) {
	db := openOutboxTestDB(t)
	createTempOutboxTable(t, db)
	store := NewSQLOutboxStore(db, nil)

	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO messaging_outbox (id, event_type, occurred_at, payload, schema_version, claim_owner, claim_expires_at)
		VALUES ($1, 'build.execution.failed', NOW(), '{}'::jsonb, 'v1', 'relay-a', NOW() + INTERVAL '30 seconds')
	`, id)
	if err != nil {
		t.Fatalf("seed insert failed: %v", err)
	}

	if err := store.MarkPublished(context.Background(), id, "relay-b"); err != nil {
		t.Fatalf("MarkPublished with wrong owner returned error: %v", err)
	}

	var publishedAt sql.NullTime
	if err := db.Get(&publishedAt, `SELECT published_at FROM messaging_outbox WHERE id = $1`, id); err != nil {
		t.Fatalf("failed checking published_at after wrong owner mark: %v", err)
	}
	if publishedAt.Valid {
		t.Fatalf("expected published_at to remain null for wrong owner, got %s", publishedAt.Time.Format(time.RFC3339))
	}

	if err := store.MarkPublished(context.Background(), id, "relay-a"); err != nil {
		t.Fatalf("MarkPublished with correct owner returned error: %v", err)
	}
	if err := db.Get(&publishedAt, `SELECT published_at FROM messaging_outbox WHERE id = $1`, id); err != nil {
		t.Fatalf("failed checking published_at after correct owner mark: %v", err)
	}
	if !publishedAt.Valid {
		t.Fatal("expected published_at to be set for correct owner")
	}
}

func TestSQLOutboxStoreMarkFailedSchedulesRetryAndClearsClaim(t *testing.T) {
	db := openOutboxTestDB(t)
	createTempOutboxTable(t, db)
	store := NewSQLOutboxStore(db, nil)

	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO messaging_outbox (id, event_type, occurred_at, payload, schema_version, claim_owner, claim_expires_at, publish_attempts)
		VALUES ($1, 'build.execution.failed', NOW(), '{}'::jsonb, 'v1', 'relay-a', NOW() + INTERVAL '30 seconds', 2)
	`, id)
	if err != nil {
		t.Fatalf("seed insert failed: %v", err)
	}

	nextAttempt := time.Now().UTC().Add(45 * time.Second).Truncate(time.Second)
	if err := store.MarkFailed(context.Background(), id, "relay-a", "nats publish timeout", nextAttempt); err != nil {
		t.Fatalf("MarkFailed returned error: %v", err)
	}

	var row struct {
		Attempts      int            `db:"publish_attempts"`
		LastError     sql.NullString `db:"last_error"`
		NextAttemptAt time.Time      `db:"next_attempt_at"`
		ClaimOwner    sql.NullString `db:"claim_owner"`
		ClaimExpires  sql.NullTime   `db:"claim_expires_at"`
	}
	if err := db.Get(&row, `
		SELECT publish_attempts, last_error, next_attempt_at, claim_owner, claim_expires_at
		FROM messaging_outbox
		WHERE id = $1
	`, id); err != nil {
		t.Fatalf("failed selecting updated row: %v", err)
	}

	if row.Attempts != 3 {
		t.Fatalf("expected attempts to increment to 3, got %d", row.Attempts)
	}
	if !row.LastError.Valid || row.LastError.String != "nats publish timeout" {
		t.Fatalf("expected last_error to be set, got %+v", row.LastError)
	}
	if !row.NextAttemptAt.UTC().Equal(nextAttempt) {
		t.Fatalf("expected next_attempt_at %s, got %s", nextAttempt.Format(time.RFC3339), row.NextAttemptAt.UTC().Format(time.RFC3339))
	}
	if row.ClaimOwner.Valid {
		t.Fatalf("expected claim_owner cleared, got %+v", row.ClaimOwner)
	}
	if row.ClaimExpires.Valid {
		t.Fatalf("expected claim_expires_at cleared, got %s", row.ClaimExpires.Time.Format(time.RFC3339))
	}
}
