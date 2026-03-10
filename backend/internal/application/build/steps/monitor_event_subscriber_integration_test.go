package steps

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"github.com/srikarm/image-factory/internal/testutil"
	"go.uber.org/zap"
)

func openMonitorSubscriberTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("IF_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set IF_TEST_POSTGRES_DSN to run monitor subscriber integration tests")
	}
	testutil.RequireSafeTestDSN(t, dsn, "IF_TEST_POSTGRES_DSN")
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("failed connecting test postgres: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createMonitorSubscriberTempTables(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TEMP TABLE workflow_instances (
			id UUID PRIMARY KEY,
			definition_id UUID NOT NULL,
			tenant_id UUID NULL,
			subject_type TEXT NOT NULL,
			subject_id UUID NOT NULL,
			status TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TEMP TABLE workflow_steps (
			id UUID PRIMARY KEY,
			instance_id UUID NOT NULL,
			step_key TEXT NOT NULL,
			payload JSONB NOT NULL DEFAULT '{}'::jsonb,
			status TEXT NOT NULL,
			attempts INT NOT NULL DEFAULT 0,
			last_error TEXT NULL,
			started_at TIMESTAMPTZ NULL,
			completed_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TEMP TABLE workflow_events (
			id UUID PRIMARY KEY,
			instance_id UUID NOT NULL,
			step_id UUID NULL,
			type TEXT NOT NULL,
			payload JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		t.Fatalf("failed creating temp workflow tables: %v", err)
	}
}

func TestBuildMonitorEventSubscriber_MultiInstanceSharedDBIdempotency(t *testing.T) {
	db := openMonitorSubscriberTestDB(t)
	createMonitorSubscriberTempTables(t, db)

	repo := postgres.NewWorkflowRepository(db, zap.NewNop())
	subscriberA := NewBuildMonitorEventSubscriber(repo, zap.NewNop())
	subscriberB := NewBuildMonitorEventSubscriber(repo, zap.NewNop())

	buildID := uuid.New()
	instanceID := uuid.New()
	monitorStepID := uuid.New()
	finalizeStepID := uuid.New()
	definitionID := uuid.New()

	_, err := db.Exec(`
		INSERT INTO workflow_instances (id, definition_id, tenant_id, subject_type, subject_id, status, created_at, updated_at)
		VALUES ($1, $2, NULL, 'build', $3, 'running', NOW() - INTERVAL '1 minute', NOW() - INTERVAL '1 minute');

		INSERT INTO workflow_steps (id, instance_id, step_key, payload, status, attempts, created_at, updated_at)
		VALUES
		($4, $1, 'build.monitor', '{}'::jsonb, 'pending', 0, NOW() - INTERVAL '50 seconds', NOW() - INTERVAL '50 seconds'),
		($5, $1, 'build.finalize', '{}'::jsonb, 'pending', 0, NOW() - INTERVAL '40 seconds', NOW() - INTERVAL '40 seconds');
	`, instanceID, definitionID, buildID, monitorStepID, finalizeStepID)
	if err != nil {
		t.Fatalf("failed seeding workflow instance and steps: %v", err)
	}

	event := messaging.Event{
		Type: messaging.EventTypeBuildExecutionCompleted,
		Payload: map[string]interface{}{
			"build_id": buildID.String(),
		},
		OccurredAt: time.Now().UTC(),
	}

	// Instance A performs transition.
	subscriberA.HandleExecutionTerminalEvent(context.Background(), event)

	var monitorStatus string
	if err := db.Get(&monitorStatus, `SELECT status FROM workflow_steps WHERE id = $1`, monitorStepID); err != nil {
		t.Fatalf("failed reading monitor step status after subscriber A: %v", err)
	}
	if monitorStatus != "succeeded" {
		t.Fatalf("expected monitor step succeeded after first transition, got %s", monitorStatus)
	}

	var eventCountAfterA int
	if err := db.Get(&eventCountAfterA, `SELECT COUNT(*) FROM workflow_events WHERE instance_id = $1 AND type = 'workflow.step.succeeded'`, instanceID); err != nil {
		t.Fatalf("failed counting workflow events after subscriber A: %v", err)
	}
	if eventCountAfterA != 1 {
		t.Fatalf("expected one terminal workflow event after subscriber A, got %d", eventCountAfterA)
	}

	// Instance B receives duplicate terminal signal and must no-op.
	subscriberB.HandleExecutionTerminalEvent(context.Background(), event)

	var eventCountAfterB int
	if err := db.Get(&eventCountAfterB, `SELECT COUNT(*) FROM workflow_events WHERE instance_id = $1 AND type = 'workflow.step.succeeded'`, instanceID); err != nil {
		t.Fatalf("failed counting workflow events after subscriber B: %v", err)
	}
	if eventCountAfterB != 1 {
		t.Fatalf("expected no duplicate terminal event after subscriber B, got %d", eventCountAfterB)
	}
	if subscriberB.Snapshot().NoopTerminal != 1 {
		t.Fatalf("expected subscriber B noop count 1, got %d", subscriberB.Snapshot().NoopTerminal)
	}
}

func TestBuildMonitorEventSubscriber_MultiInstanceSharedDBIdempotencyFailedPath(t *testing.T) {
	db := openMonitorSubscriberTestDB(t)
	createMonitorSubscriberTempTables(t, db)

	repo := postgres.NewWorkflowRepository(db, zap.NewNop())
	subscriberA := NewBuildMonitorEventSubscriber(repo, zap.NewNop())
	subscriberB := NewBuildMonitorEventSubscriber(repo, zap.NewNop())

	buildID := uuid.New()
	instanceID := uuid.New()
	monitorStepID := uuid.New()
	definitionID := uuid.New()

	_, err := db.Exec(`
		INSERT INTO workflow_instances (id, definition_id, tenant_id, subject_type, subject_id, status, created_at, updated_at)
		VALUES ($1, $2, NULL, 'build', $3, 'running', NOW() - INTERVAL '1 minute', NOW() - INTERVAL '1 minute');

		INSERT INTO workflow_steps (id, instance_id, step_key, payload, status, attempts, created_at, updated_at)
		VALUES
		($4, $1, 'build.monitor', '{}'::jsonb, 'running', 1, NOW() - INTERVAL '50 seconds', NOW() - INTERVAL '50 seconds');
	`, instanceID, definitionID, buildID, monitorStepID)
	if err != nil {
		t.Fatalf("failed seeding workflow instance and step: %v", err)
	}

	firstEvent := messaging.Event{
		Type: messaging.EventTypeBuildExecutionFailed,
		Payload: map[string]interface{}{
			"build_id": buildID.String(),
			"message":  "kaniko push failed",
		},
		OccurredAt: time.Now().UTC(),
	}
	duplicateEvent := messaging.Event{
		Type: messaging.EventTypeBuildExecutionFailed,
		Payload: map[string]interface{}{
			"build_id": buildID.String(),
			"message":  "different failure detail that should be ignored after terminal transition",
		},
		OccurredAt: time.Now().UTC().Add(1 * time.Second),
	}

	// Instance A performs failed transition.
	subscriberA.HandleExecutionTerminalEvent(context.Background(), firstEvent)

	var monitorStatus string
	var monitorErrMsg *string
	if err := db.QueryRowx(`
		SELECT status, last_error
		FROM workflow_steps
		WHERE id = $1
	`, monitorStepID).Scan(&monitorStatus, &monitorErrMsg); err != nil {
		t.Fatalf("failed reading monitor step after first failed transition: %v", err)
	}
	if monitorStatus != "failed" {
		t.Fatalf("expected monitor status failed, got %s", monitorStatus)
	}
	if monitorErrMsg == nil || *monitorErrMsg != "kaniko push failed" {
		t.Fatalf("expected monitor last_error from first event, got %+v", monitorErrMsg)
	}

	var failedEventCountAfterA int
	if err := db.Get(&failedEventCountAfterA, `SELECT COUNT(*) FROM workflow_events WHERE instance_id = $1 AND type = 'workflow.step.failed'`, instanceID); err != nil {
		t.Fatalf("failed counting failed workflow events after subscriber A: %v", err)
	}
	if failedEventCountAfterA != 1 {
		t.Fatalf("expected one failed workflow event after subscriber A, got %d", failedEventCountAfterA)
	}

	// Instance B receives duplicate failure signal and must no-op.
	subscriberB.HandleExecutionTerminalEvent(context.Background(), duplicateEvent)

	var failedEventCountAfterB int
	if err := db.Get(&failedEventCountAfterB, `SELECT COUNT(*) FROM workflow_events WHERE instance_id = $1 AND type = 'workflow.step.failed'`, instanceID); err != nil {
		t.Fatalf("failed counting failed workflow events after subscriber B: %v", err)
	}
	if failedEventCountAfterB != 1 {
		t.Fatalf("expected no duplicate failed workflow event after subscriber B, got %d", failedEventCountAfterB)
	}
	if subscriberB.Snapshot().NoopTerminal != 1 {
		t.Fatalf("expected subscriber B noop count 1, got %d", subscriberB.Snapshot().NoopTerminal)
	}

	var monitorErrMsgAfterDup *string
	if err := db.Get(&monitorErrMsgAfterDup, `SELECT last_error FROM workflow_steps WHERE id = $1`, monitorStepID); err != nil {
		t.Fatalf("failed reading monitor last_error after duplicate event: %v", err)
	}
	if monitorErrMsgAfterDup == nil || *monitorErrMsgAfterDup != "kaniko push failed" {
		t.Fatalf("expected original monitor last_error to remain unchanged, got %+v", monitorErrMsgAfterDup)
	}
}
