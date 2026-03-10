package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/domain/workflow"
	"go.uber.org/zap"
)

type WorkflowRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewWorkflowRepository(db *sqlx.DB, logger *zap.Logger) *WorkflowRepository {
	return &WorkflowRepository{db: db, logger: logger}
}

type dbWorkflowStep struct {
	ID          uuid.UUID  `db:"id"`
	InstanceID  uuid.UUID  `db:"instance_id"`
	StepKey     string     `db:"step_key"`
	Payload     []byte     `db:"payload"`
	Status      string     `db:"status"`
	Attempts    int        `db:"attempts"`
	LastError   *string    `db:"last_error"`
	StartedAt   *time.Time `db:"started_at"`
	CompletedAt *time.Time `db:"completed_at"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

func (r *WorkflowRepository) ClaimNextRunnableStep(ctx context.Context) (*workflow.Step, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	query := `
		SELECT ws.id, ws.instance_id, ws.step_key, ws.payload, ws.status, ws.attempts, ws.last_error, ws.started_at, ws.completed_at, ws.created_at, ws.updated_at
		FROM workflow_steps ws
		JOIN workflow_instances wi ON wi.id = ws.instance_id
		WHERE ws.status = $1 AND wi.status = $2
		ORDER BY wi.created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1`

	var step dbWorkflowStep
	if err = tx.GetContext(ctx, &step, query, string(workflow.StepStatusPending), string(workflow.InstanceStatusRunning)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = tx.Rollback()
			return nil, nil
		}
		return nil, fmt.Errorf("failed to select runnable step: %w", err)
	}

	now := time.Now().UTC()
	updateQuery := `
		UPDATE workflow_steps
		SET status = $2, attempts = attempts + 1, started_at = $3
		WHERE id = $1 AND status = $4`
	res, err := tx.ExecContext(ctx, updateQuery, step.ID, string(workflow.StepStatusRunning), now, string(workflow.StepStatusPending))
	if err != nil {
		return nil, fmt.Errorf("failed to claim workflow step: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to read claim result: %w", err)
	}
	if affected == 0 {
		_ = tx.Rollback()
		return nil, nil
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit claim: %w", err)
	}

	step.Status = string(workflow.StepStatusRunning)
	step.Attempts++
	step.StartedAt = &now
	step.UpdatedAt = now

	var payload map[string]interface{}
	if len(step.Payload) > 0 {
		if err := json.Unmarshal(step.Payload, &payload); err != nil {
			r.logger.Warn("Failed to unmarshal workflow step payload", zap.Error(err), zap.String("step_id", step.ID.String()))
		}
	}
	return &workflow.Step{
		ID:          step.ID,
		InstanceID:  step.InstanceID,
		StepKey:     step.StepKey,
		Payload:     payload,
		Status:      workflow.StepStatus(step.Status),
		Attempts:    step.Attempts,
		LastError:   step.LastError,
		StartedAt:   step.StartedAt,
		CompletedAt: step.CompletedAt,
		CreatedAt:   step.CreatedAt,
		UpdatedAt:   step.UpdatedAt,
	}, nil
}

func (r *WorkflowRepository) UpdateStep(ctx context.Context, step *workflow.Step) error {
	payload, err := json.Marshal(step.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow step payload: %w", err)
	}

	query := `
		UPDATE workflow_steps
		SET status = $2,
		    attempts = $3,
		    last_error = $4,
		    started_at = $5,
		    completed_at = $6,
		    payload = COALESCE(NULLIF($7, 'null'::jsonb), payload),
		    updated_at = $8
		WHERE id = $1`
	_, err = r.db.ExecContext(ctx, query,
		step.ID,
		string(step.Status),
		step.Attempts,
		step.LastError,
		step.StartedAt,
		step.CompletedAt,
		payload,
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to update workflow step: %w", err)
	}
	return nil
}

func (r *WorkflowRepository) AppendEvent(ctx context.Context, event *workflow.Event) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow event payload: %w", err)
	}
	query := `
		INSERT INTO workflow_events (id, instance_id, step_id, type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = r.db.ExecContext(ctx, query,
		event.ID,
		event.InstanceID,
		event.StepID,
		event.Type,
		payload,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to append workflow event: %w", err)
	}
	return nil
}

func (r *WorkflowRepository) UpsertDefinition(ctx context.Context, name string, version int, definition map[string]interface{}) (uuid.UUID, error) {
	payload, err := json.Marshal(definition)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal workflow definition: %w", err)
	}

	var definitionID uuid.UUID
	query := `
		INSERT INTO workflow_definitions (name, version, definition)
		VALUES ($1, $2, $3)
		ON CONFLICT (name, version)
		DO UPDATE SET definition = EXCLUDED.definition, updated_at = now()
		RETURNING id`
	if err := r.db.GetContext(ctx, &definitionID, query, name, version, payload); err != nil {
		return uuid.Nil, fmt.Errorf("failed to upsert workflow definition: %w", err)
	}
	return definitionID, nil
}

func (r *WorkflowRepository) CreateInstance(ctx context.Context, definitionID uuid.UUID, tenantID *uuid.UUID, subjectType string, subjectID uuid.UUID, status workflow.InstanceStatus) (uuid.UUID, error) {
	var instanceID uuid.UUID
	query := `
		INSERT INTO workflow_instances (definition_id, tenant_id, subject_type, subject_id, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	if err := r.db.GetContext(ctx, &instanceID, query, definitionID, tenantID, subjectType, subjectID, string(status)); err != nil {
		return uuid.Nil, fmt.Errorf("failed to create workflow instance: %w", err)
	}
	return instanceID, nil
}

func (r *WorkflowRepository) CreateSteps(ctx context.Context, instanceID uuid.UUID, steps []workflow.StepDefinition) error {
	query := `
		INSERT INTO workflow_steps (instance_id, step_key, payload, status)
		VALUES ($1, $2, $3, $4)`
	for _, step := range steps {
		payload, err := json.Marshal(step.Payload)
		if err != nil {
			return fmt.Errorf("failed to marshal workflow step payload: %w", err)
		}
		status := step.Status
		if status == "" {
			status = workflow.StepStatusPending
		}
		if _, err := r.db.ExecContext(ctx, query, instanceID, step.StepKey, payload, string(status)); err != nil {
			return fmt.Errorf("failed to create workflow step: %w", err)
		}
	}
	return nil
}

func (r *WorkflowRepository) UpdateInstanceStatus(ctx context.Context, instanceID uuid.UUID, status workflow.InstanceStatus) error {
	query := `
		UPDATE workflow_instances
		SET status = $2, updated_at = now()
		WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, query, instanceID, string(status)); err != nil {
		return fmt.Errorf("failed to update workflow instance status: %w", err)
	}
	return nil
}

func (r *WorkflowRepository) UpdateStepStatus(ctx context.Context, instanceID uuid.UUID, stepKey string, status workflow.StepStatus, errMsg *string) error {
	startNow := status == workflow.StepStatusRunning
	completeNow := status == workflow.StepStatusSucceeded || status == workflow.StepStatusFailed || status == workflow.StepStatusBlocked

	query := `
		UPDATE workflow_steps
		SET status = $3,
		    last_error = $4,
		    started_at = CASE WHEN $5 AND started_at IS NULL THEN now() ELSE started_at END,
		    completed_at = CASE WHEN $6 THEN now() ELSE completed_at END,
		    updated_at = now()
		WHERE instance_id = $1 AND step_key = $2`
	if _, err := r.db.ExecContext(ctx, query, instanceID, stepKey, string(status), errMsg, startNow, completeNow); err != nil {
		return fmt.Errorf("failed to update workflow step status: %w", err)
	}
	return nil
}

func (r *WorkflowRepository) GetInstanceWithStepsBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) (*workflow.Instance, []workflow.Step, error) {
	var instance workflow.Instance
	instQuery := `
		SELECT id, definition_id, tenant_id, subject_type, subject_id, status, created_at, updated_at
		FROM workflow_instances
		WHERE subject_type = $1 AND subject_id = $2
		ORDER BY created_at DESC
		LIMIT 1`
	if err := r.db.GetContext(ctx, &instance, instQuery, subjectType, subjectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to load workflow instance: %w", err)
	}

	stepQuery := `
		SELECT id, instance_id, step_key, payload, status, attempts, last_error, started_at, completed_at, created_at, updated_at
		FROM workflow_steps
		WHERE instance_id = $1
		ORDER BY created_at ASC`
	rows, err := r.db.QueryxContext(ctx, stepQuery, instance.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load workflow steps: %w", err)
	}
	defer rows.Close()

	var steps []workflow.Step
	for rows.Next() {
		var (
			row     dbWorkflowStep
			payload map[string]interface{}
		)
		if err := rows.StructScan(&row); err != nil {
			return nil, nil, fmt.Errorf("failed to scan workflow step: %w", err)
		}
		if len(row.Payload) > 0 {
			if err := json.Unmarshal(row.Payload, &payload); err != nil {
				r.logger.Warn("Failed to unmarshal workflow step payload", zap.Error(err), zap.String("step_id", row.ID.String()))
			}
		}
		steps = append(steps, workflow.Step{
			ID:          row.ID,
			InstanceID:  row.InstanceID,
			StepKey:     row.StepKey,
			Payload:     payload,
			Status:      workflow.StepStatus(row.Status),
			Attempts:    row.Attempts,
			LastError:   row.LastError,
			StartedAt:   row.StartedAt,
			CompletedAt: row.CompletedAt,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return &instance, steps, nil
}

func (r *WorkflowRepository) GetBlockedStepDiagnostics(ctx context.Context, subjectType string) (*workflow.BlockedStepDiagnostics, error) {
	diag := &workflow.BlockedStepDiagnostics{
		SubjectType: subjectType,
	}

	// In current control-plane model, blocked handler outcomes are requeued as pending.
	// We treat pending steps with last_error on running instances as effectively blocked.
	query := `
		SELECT
			COUNT(*) AS blocked_count,
			MIN(ws.updated_at) AS oldest_blocked_at,
			COUNT(*) FILTER (WHERE ws.step_key = 'build.dispatch') AS dispatch_blocked,
			COUNT(*) FILTER (WHERE ws.step_key = 'build.monitor') AS monitor_blocked,
			COUNT(*) FILTER (WHERE ws.step_key = 'build.finalize') AS finalize_blocked
		FROM workflow_steps ws
		JOIN workflow_instances wi ON wi.id = ws.instance_id
		WHERE wi.subject_type = $1
		  AND wi.status = $2
		  AND ws.status = $3
		  AND ws.last_error IS NOT NULL`

	var (
		oldestBlocked sql.NullTime
	)
	if err := r.db.QueryRowxContext(
		ctx,
		query,
		subjectType,
		string(workflow.InstanceStatusRunning),
		string(workflow.StepStatusPending),
	).Scan(
		&diag.BlockedStepCount,
		&oldestBlocked,
		&diag.DispatchBlocked,
		&diag.MonitorBlocked,
		&diag.FinalizeBlocked,
	); err != nil {
		return nil, fmt.Errorf("failed to load blocked workflow diagnostics: %w", err)
	}

	if oldestBlocked.Valid {
		ts := oldestBlocked.Time.UTC()
		diag.OldestBlockedAt = &ts
	}

	return diag, nil
}

func (r *WorkflowRepository) RequeueBlockedImportDispatchSteps(ctx context.Context) (int64, error) {
	query := `
		UPDATE workflow_steps ws
		SET status = $1,
		    last_error = NULL,
		    completed_at = NULL,
		    updated_at = now()
		FROM workflow_instances wi
		WHERE ws.instance_id = wi.id
		  AND wi.status = $2
		  AND wi.subject_type = $3
		  AND ws.step_key = $4
		  AND ws.status = $5
		  AND ws.last_error LIKE $6`
	res, err := r.db.ExecContext(
		ctx,
		query,
		string(workflow.StepStatusPending),
		string(workflow.InstanceStatusRunning),
		"external_image_import",
		"import.dispatch",
		string(workflow.StepStatusBlocked),
		"waiting_for_dispatch:%",
	)
	if err != nil {
		return 0, fmt.Errorf("failed to requeue blocked import.dispatch steps: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read requeue rows affected: %w", err)
	}
	return affected, nil
}
