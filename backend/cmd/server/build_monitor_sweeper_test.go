package main

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
	buildsteps "github.com/srikarm/image-factory/internal/application/build/steps"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"github.com/srikarm/image-factory/internal/testutil"
)

type sweeperRepoStub struct {
	updateErrByInstance map[uuid.UUID]error
	updateCalls         int
	updates             []sweeperUpdate
	appendCalls         int
	events              []*domainworkflow.Event
}

type sweeperUpdate struct {
	InstanceID uuid.UUID
	StepKey    string
	Status     domainworkflow.StepStatus
	ErrMsg     *string
}

func (s *sweeperRepoStub) ClaimNextRunnableStep(ctx context.Context) (*domainworkflow.Step, error) {
	return nil, nil
}
func (s *sweeperRepoStub) UpdateStep(ctx context.Context, step *domainworkflow.Step) error {
	return nil
}
func (s *sweeperRepoStub) AppendEvent(ctx context.Context, event *domainworkflow.Event) error {
	s.appendCalls++
	s.events = append(s.events, event)
	return nil
}
func (s *sweeperRepoStub) UpsertDefinition(ctx context.Context, name string, version int, definition map[string]interface{}) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (s *sweeperRepoStub) CreateInstance(ctx context.Context, definitionID uuid.UUID, tenantID *uuid.UUID, subjectType string, subjectID uuid.UUID, status domainworkflow.InstanceStatus) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (s *sweeperRepoStub) CreateSteps(ctx context.Context, instanceID uuid.UUID, steps []domainworkflow.StepDefinition) error {
	return nil
}
func (s *sweeperRepoStub) UpdateInstanceStatus(ctx context.Context, instanceID uuid.UUID, status domainworkflow.InstanceStatus) error {
	return nil
}
func (s *sweeperRepoStub) UpdateStepStatus(ctx context.Context, instanceID uuid.UUID, stepKey string, status domainworkflow.StepStatus, errMsg *string) error {
	s.updateCalls++
	s.updates = append(s.updates, sweeperUpdate{InstanceID: instanceID, StepKey: stepKey, Status: status, ErrMsg: errMsg})
	if s.updateErrByInstance != nil {
		if err := s.updateErrByInstance[instanceID]; err != nil {
			return err
		}
	}
	return nil
}
func (s *sweeperRepoStub) GetInstanceWithStepsBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) (*domainworkflow.Instance, []domainworkflow.Step, error) {
	return nil, nil, nil
}
func (s *sweeperRepoStub) GetBlockedStepDiagnostics(ctx context.Context, subjectType string) (*domainworkflow.BlockedStepDiagnostics, error) {
	return nil, nil
}

func openSweeperTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("IF_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set IF_TEST_POSTGRES_DSN to run monitor sweeper DB tests")
	}
	testutil.RequireSafeTestDSN(t, dsn, "IF_TEST_POSTGRES_DSN")
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("failed connecting to postgres: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createSweeperTempTables(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TEMP TABLE workflow_instances (
			id UUID PRIMARY KEY,
			subject_id UUID NOT NULL,
			subject_type TEXT NOT NULL,
			status TEXT NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TEMP TABLE workflow_steps (
			id UUID PRIMARY KEY,
			instance_id UUID NOT NULL,
			step_key TEXT NOT NULL,
			status TEXT NOT NULL
		);
		CREATE TEMP TABLE builds (
			id UUID PRIMARY KEY,
			status TEXT NOT NULL,
			error_message TEXT NULL
		);
	`)
	if err != nil {
		t.Fatalf("failed creating temp tables: %v", err)
	}
}

func TestRunBuildMonitorSweeperTick_ReconcilesTerminalBuilds(t *testing.T) {
	db := openSweeperTestDB(t)
	createSweeperTempTables(t, db)
	repo := &sweeperRepoStub{}

	instCompleted := uuid.New()
	instFailed := uuid.New()
	instIgnored := uuid.New()
	buildCompleted := uuid.New()
	buildFailed := uuid.New()
	buildRunning := uuid.New()
	stepCompleted := uuid.New()
	stepFailed := uuid.New()
	stepIgnored := uuid.New()

	_, err := db.Exec(`
		INSERT INTO workflow_instances (id, subject_id, subject_type, status, updated_at) VALUES
		($1, $2, 'build', 'running', NOW() - INTERVAL '2 minutes'),
		($3, $4, 'build', 'running', NOW() - INTERVAL '1 minute'),
		($5, $6, 'build', 'running', NOW());

		INSERT INTO workflow_steps (id, instance_id, step_key, status) VALUES
		($7, $1, 'build.monitor', 'pending'),
		($8, $3, 'build.monitor', 'running'),
		($9, $5, 'build.monitor', 'pending');

		INSERT INTO builds (id, status, error_message) VALUES
		($2, 'completed', NULL),
		($4, 'failed', 'kaniko task failed'),
		($6, 'running', NULL);
	`, instCompleted, buildCompleted, instFailed, buildFailed, instIgnored, buildRunning, stepCompleted, stepFailed, stepIgnored)
	if err != nil {
		t.Fatalf("failed seeding sweep data: %v", err)
	}

	attempted, reconciled, failed, runErr := runBuildMonitorSweeperTick(context.Background(), db, repo, nil, 50)
	if runErr != nil {
		t.Fatalf("runBuildMonitorSweeperTick returned error: %v", runErr)
	}
	if attempted != 2 {
		t.Fatalf("expected attempted=2, got %d", attempted)
	}
	if reconciled != 2 {
		t.Fatalf("expected reconciled=2, got %d", reconciled)
	}
	if failed != 0 {
		t.Fatalf("expected failed=0, got %d", failed)
	}
	if repo.updateCalls != 2 {
		t.Fatalf("expected 2 update calls, got %d", repo.updateCalls)
	}
	if repo.appendCalls != 2 {
		t.Fatalf("expected 2 append calls, got %d", repo.appendCalls)
	}

	var seenCompleted, seenFailed bool
	for _, up := range repo.updates {
		if up.StepKey != buildsteps.StepMonitorBuild {
			t.Fatalf("unexpected step key %s", up.StepKey)
		}
		switch up.InstanceID {
		case instCompleted:
			seenCompleted = true
			if up.Status != domainworkflow.StepStatusSucceeded {
				t.Fatalf("expected completed build -> succeeded, got %s", up.Status)
			}
		case instFailed:
			seenFailed = true
			if up.Status != domainworkflow.StepStatusFailed {
				t.Fatalf("expected failed build -> failed, got %s", up.Status)
			}
			if up.ErrMsg == nil || *up.ErrMsg != "kaniko task failed" {
				t.Fatalf("expected failed reason propagated, got %+v", up.ErrMsg)
			}
		default:
			t.Fatalf("unexpected instance updated: %s", up.InstanceID)
		}
	}
	if !seenCompleted || !seenFailed {
		t.Fatalf("expected updates for completed and failed candidates, seenCompleted=%v seenFailed=%v", seenCompleted, seenFailed)
	}
}

func TestRunBuildMonitorSweeperTick_TracksUpdateFailures(t *testing.T) {
	db := openSweeperTestDB(t)
	createSweeperTempTables(t, db)

	instOK := uuid.New()
	instFail := uuid.New()
	buildOK := uuid.New()
	buildFail := uuid.New()
	stepOK := uuid.New()
	stepFail := uuid.New()

	_, err := db.Exec(`
		INSERT INTO workflow_instances (id, subject_id, subject_type, status, updated_at) VALUES
		($1, $2, 'build', 'running', NOW() - INTERVAL '2 minutes'),
		($3, $4, 'build', 'running', NOW() - INTERVAL '1 minute');

		INSERT INTO workflow_steps (id, instance_id, step_key, status) VALUES
		($5, $1, 'build.monitor', 'pending'),
		($6, $3, 'build.monitor', 'pending');

		INSERT INTO builds (id, status, error_message) VALUES
		($2, 'completed', NULL),
		($4, 'failed', 'remote registry timeout');
	`, instOK, buildOK, instFail, buildFail, stepOK, stepFail)
	if err != nil {
		t.Fatalf("failed seeding data: %v", err)
	}

	repo := &sweeperRepoStub{
		updateErrByInstance: map[uuid.UUID]error{instFail: sql.ErrTxDone},
	}
	attempted, reconciled, failed, runErr := runBuildMonitorSweeperTick(context.Background(), db, repo, nil, 10)
	if runErr != nil {
		t.Fatalf("runBuildMonitorSweeperTick returned error: %v", runErr)
	}
	if attempted != 2 {
		t.Fatalf("expected attempted=2, got %d", attempted)
	}
	if reconciled != 1 {
		t.Fatalf("expected reconciled=1, got %d", reconciled)
	}
	if failed != 1 {
		t.Fatalf("expected failed=1, got %d", failed)
	}
	if repo.appendCalls != 1 {
		t.Fatalf("expected append only for successful update, got %d", repo.appendCalls)
	}

	if len(repo.events) != 1 {
		t.Fatalf("expected exactly one event, got %d", len(repo.events))
	}
	if repo.events[0].Type != "workflow.step.succeeded" {
		t.Fatalf("expected successful path event type, got %s", repo.events[0].Type)
	}
	if repo.events[0].CreatedAt.Before(time.Now().UTC().Add(-1 * time.Minute)) {
		t.Fatalf("expected recent CreatedAt on appended event, got %s", repo.events[0].CreatedAt)
	}
}
