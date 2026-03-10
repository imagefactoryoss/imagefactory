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
	"github.com/lib/pq"
	"github.com/srikarm/image-factory/internal/domain/build"
	"go.uber.org/zap"
)

type TriggerRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewTriggerRepository creates a new PostgreSQL trigger repository
func NewTriggerRepository(db *sqlx.DB, logger *zap.Logger) build.TriggerRepository {
	return &TriggerRepository{
		db:     db,
		logger: logger,
	}
}

// dbBuildTrigger represents a trigger row in the database
type dbBuildTrigger struct {
	ID                 uuid.UUID      `db:"id"`
	TenantID           uuid.UUID      `db:"tenant_id"`
	ProjectID          uuid.UUID      `db:"project_id"`
	BuildID            uuid.UUID      `db:"build_id"`
	CreatedBy          uuid.UUID      `db:"created_by"`
	TriggerType        string         `db:"trigger_type"`
	TriggerName        string         `db:"trigger_name"`
	TriggerDescription sql.NullString `db:"trigger_description"`
	WebhookURL         sql.NullString `db:"webhook_url"`
	WebhookSecret      sql.NullString `db:"webhook_secret"`
	WebhookEvents      pq.StringArray `db:"webhook_events"`
	CronExpression     sql.NullString `db:"cron_expression"`
	Timezone           string         `db:"timezone"`
	LastTriggeredAt    *time.Time     `db:"last_triggered_at"`
	NextTriggerAt      *time.Time     `db:"next_trigger_at"`
	GitProvider        sql.NullString `db:"git_provider"`
	GitRepositoryURL   sql.NullString `db:"git_repository_url"`
	GitBranchPattern   sql.NullString `db:"git_branch_pattern"`
	IsActive           bool           `db:"is_active"`
	CreatedAt          time.Time      `db:"created_at"`
	UpdatedAt          time.Time      `db:"updated_at"`
}

// dbBuildTriggerToTrigger converts a database row to a BuildTrigger domain object
func (r *TriggerRepository) dbBuildTriggerToTrigger(db *dbBuildTrigger) *build.BuildTrigger {
	// This is a simplified conversion. In a real implementation, you'd reconstruct
	// the domain object properly. For now, we create it via the constructor.
	var trigger *build.BuildTrigger

	triggerType := build.TriggerType(db.TriggerType)

	switch triggerType {
	case build.TriggerTypeWebhook:
		webhook, _ := build.NewWebhookTrigger(
			db.TenantID,
			db.ProjectID,
			db.BuildID,
			db.CreatedBy,
			db.TriggerName,
			db.TriggerDescription.String,
			db.WebhookURL.String,
			db.WebhookSecret.String,
			db.WebhookEvents,
		)
		trigger = webhook
	case build.TriggerTypeSchedule:
		schedule, _ := build.NewScheduledTrigger(
			db.TenantID,
			db.ProjectID,
			db.BuildID,
			db.CreatedBy,
			db.TriggerName,
			db.TriggerDescription.String,
			db.CronExpression.String,
			db.Timezone,
		)
		trigger = schedule
	case build.TriggerTypeGitEvent:
		gitTrigger, _ := build.NewGitEventTrigger(
			db.TenantID,
			db.ProjectID,
			db.BuildID,
			db.CreatedBy,
			db.TriggerName,
			db.TriggerDescription.String,
			build.GitProvider(db.GitProvider.String),
			db.GitRepositoryURL.String,
			db.GitBranchPattern.String,
		)
		trigger = gitTrigger
	}

	// Set the ID and timestamps directly on the domain object
	if trigger != nil {
		// Override the ID that was auto-generated in the constructor
		trigger.ID = db.ID
		trigger.CreatedAt = db.CreatedAt
		trigger.UpdatedAt = db.UpdatedAt
		trigger.IsActive = db.IsActive
		if db.LastTriggeredAt != nil {
			trigger.LastTriggered = db.LastTriggeredAt
		}
		if db.NextTriggerAt != nil {
			trigger.NextTrigger = db.NextTriggerAt
		}
	}

	return trigger
}

// SaveTrigger creates or updates a trigger
func (r *TriggerRepository) SaveTrigger(ctx context.Context, trigger *build.BuildTrigger) error {
	r.logger.Info("Saving trigger",
		zap.String("trigger_id", trigger.ID.String()),
		zap.String("trigger_type", string(trigger.Type)),
		zap.String("build_id", trigger.BuildID.String()),
	)

	query := `
		INSERT INTO build_triggers (
			id, tenant_id, project_id, build_id, created_by,
			trigger_type, trigger_name, trigger_description,
			webhook_url, webhook_secret, webhook_events,
			cron_expression, timezone, last_triggered_at, next_trigger_at,
			git_provider, git_repository_url, git_branch_pattern,
			is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18,
			$19, $20, $21
		)
		ON CONFLICT (id) DO UPDATE SET
			trigger_name = EXCLUDED.trigger_name,
			trigger_description = EXCLUDED.trigger_description,
			is_active = EXCLUDED.is_active,
			last_triggered_at = EXCLUDED.last_triggered_at,
			next_trigger_at = EXCLUDED.next_trigger_at,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		trigger.ID,
		trigger.TenantID,
		trigger.ProjectID,
		trigger.BuildID,
		trigger.CreatedBy,
		string(trigger.Type),
		trigger.Name,
		trigger.Description,
		nullIfEmpty(trigger.WebhookURL),
		nullIfEmpty(trigger.WebhookSecret),
		pq.Array(trigger.WebhookEvents),
		nullIfEmpty(trigger.CronExpr),
		trigger.Timezone,
		trigger.LastTriggered,
		trigger.NextTrigger,
		nullIfEmpty(string(trigger.GitProvider)),
		nullIfEmpty(trigger.GitRepoURL),
		nullIfEmpty(trigger.GitBranchPattern),
		trigger.IsActive,
		trigger.CreatedAt,
		trigger.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to save trigger", zap.Error(err), zap.String("trigger_id", trigger.ID.String()))
		return fmt.Errorf("failed to save trigger: %w", err)
	}

	r.logger.Info("Trigger saved successfully", zap.String("trigger_id", trigger.ID.String()))
	return nil
}

// GetTrigger retrieves a trigger by ID
func (r *TriggerRepository) GetTrigger(ctx context.Context, triggerID uuid.UUID) (*build.BuildTrigger, error) {
	query := `
		SELECT id, tenant_id, project_id, build_id, created_by,
			   trigger_type, trigger_name, trigger_description,
			   webhook_url, webhook_secret, webhook_events,
			   cron_expression, timezone, last_triggered_at, next_trigger_at,
			   git_provider, git_repository_url, git_branch_pattern,
			   is_active, created_at, updated_at
		FROM build_triggers
		WHERE id = $1
	`

	var dbTrigger dbBuildTrigger
	err := r.db.GetContext(ctx, &dbTrigger, query, triggerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("Failed to get trigger", zap.Error(err), zap.String("trigger_id", triggerID.String()))
		return nil, fmt.Errorf("failed to get trigger: %w", err)
	}

	return r.dbBuildTriggerToTrigger(&dbTrigger), nil
}

// GetTriggersByBuild retrieves all triggers for a build
func (r *TriggerRepository) GetTriggersByBuild(ctx context.Context, buildID uuid.UUID) ([]*build.BuildTrigger, error) {
	query := `
		SELECT id, tenant_id, project_id, build_id, created_by,
			   trigger_type, trigger_name, trigger_description,
			   webhook_url, webhook_secret, webhook_events,
			   cron_expression, timezone, last_triggered_at, next_trigger_at,
			   git_provider, git_repository_url, git_branch_pattern,
			   is_active, created_at, updated_at
		FROM build_triggers
		WHERE build_id = $1
		ORDER BY created_at DESC
	`

	var dbTriggers []dbBuildTrigger
	err := r.db.SelectContext(ctx, &dbTriggers, query, buildID)
	if err != nil {
		r.logger.Error("Failed to get triggers for build", zap.Error(err), zap.String("build_id", buildID.String()))
		return nil, fmt.Errorf("failed to get triggers: %w", err)
	}

	triggers := make([]*build.BuildTrigger, len(dbTriggers))
	for i, dbTrigger := range dbTriggers {
		triggers[i] = r.dbBuildTriggerToTrigger(&dbTrigger)
	}

	return triggers, nil
}

// GetTriggersByProject retrieves all triggers for a project
func (r *TriggerRepository) GetTriggersByProject(ctx context.Context, projectID uuid.UUID) ([]*build.BuildTrigger, error) {
	query := `
		SELECT id, tenant_id, project_id, build_id, created_by,
			   trigger_type, trigger_name, trigger_description,
			   webhook_url, webhook_secret, webhook_events,
			   cron_expression, timezone, last_triggered_at, next_trigger_at,
			   git_provider, git_repository_url, git_branch_pattern,
			   is_active, created_at, updated_at
		FROM build_triggers
		WHERE project_id = $1
		ORDER BY created_at DESC
	`

	var dbTriggers []dbBuildTrigger
	err := r.db.SelectContext(ctx, &dbTriggers, query, projectID)
	if err != nil {
		r.logger.Error("Failed to get triggers for project", zap.Error(err), zap.String("project_id", projectID.String()))
		return nil, fmt.Errorf("failed to get triggers: %w", err)
	}

	triggers := make([]*build.BuildTrigger, len(dbTriggers))
	for i, dbTrigger := range dbTriggers {
		triggers[i] = r.dbBuildTriggerToTrigger(&dbTrigger)
	}

	return triggers, nil
}

// GetActiveScheduledTriggers retrieves all active scheduled triggers for a tenant
// This is used by the scheduler service to find triggers that need to run
func (r *TriggerRepository) GetActiveScheduledTriggers(ctx context.Context, tenantID uuid.UUID) ([]*build.BuildTrigger, error) {
	query := `
		SELECT id, tenant_id, project_id, build_id, created_by,
			   trigger_type, trigger_name, trigger_description,
			   webhook_url, webhook_secret, webhook_events,
			   cron_expression, timezone, last_triggered_at, next_trigger_at,
			   git_provider, git_repository_url, git_branch_pattern,
			   is_active, created_at, updated_at
		FROM build_triggers
		WHERE tenant_id = $1 
		  AND trigger_type = 'schedule'
		  AND is_active = true
		  AND (next_trigger_at IS NULL OR next_trigger_at <= NOW())
		ORDER BY next_trigger_at ASC
	`

	var dbTriggers []dbBuildTrigger
	err := r.db.SelectContext(ctx, &dbTriggers, query, tenantID)
	if err != nil {
		r.logger.Error("Failed to get active scheduled triggers", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil, fmt.Errorf("failed to get scheduled triggers: %w", err)
	}

	triggers := make([]*build.BuildTrigger, len(dbTriggers))
	for i, dbTrigger := range dbTriggers {
		triggers[i] = r.dbBuildTriggerToTrigger(&dbTrigger)
	}

	return triggers, nil
}

// UpdateTrigger updates an existing trigger
func (r *TriggerRepository) UpdateTrigger(ctx context.Context, trigger *build.BuildTrigger) error {
	r.logger.Info("Updating trigger", zap.String("trigger_id", trigger.ID.String()))

	query := `
		UPDATE build_triggers SET
			trigger_name = $1,
			trigger_description = $2,
			last_triggered_at = $3,
			next_trigger_at = $4,
			is_active = $5,
			webhook_url = $6,
			webhook_secret = $7,
			webhook_events = $8,
			updated_at = $9
		WHERE id = $10
	`

	_, err := r.db.ExecContext(ctx, query,
		trigger.Name,
		trigger.Description,
		trigger.LastTriggered,
		trigger.NextTrigger,
		trigger.IsActive,
		nullIfEmpty(trigger.WebhookURL),
		nullIfEmpty(trigger.WebhookSecret),
		pq.Array(trigger.WebhookEvents),
		trigger.UpdatedAt,
		trigger.ID,
	)

	if err != nil {
		r.logger.Error("Failed to update trigger", zap.Error(err), zap.String("trigger_id", trigger.ID.String()))
		return fmt.Errorf("failed to update trigger: %w", err)
	}

	r.logger.Info("Trigger updated successfully", zap.String("trigger_id", trigger.ID.String()))
	return nil
}

// DeleteTrigger deletes a trigger
func (r *TriggerRepository) DeleteTrigger(ctx context.Context, triggerID uuid.UUID) error {
	r.logger.Info("Deleting trigger", zap.String("trigger_id", triggerID.String()))

	query := `DELETE FROM build_triggers WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, triggerID)
	if err != nil {
		r.logger.Error("Failed to delete trigger", zap.Error(err), zap.String("trigger_id", triggerID.String()))
		return fmt.Errorf("failed to delete trigger: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("trigger not found: %s", triggerID.String())
	}

	r.logger.Info("Trigger deleted successfully", zap.String("trigger_id", triggerID.String()))
	return nil
}

// Helper function to convert empty strings to NULL for database
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Helper function to marshal trigger configuration to JSON
func marshalTriggerConfig(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

// Helper function to unmarshal trigger configuration from JSON
func unmarshalTriggerConfig(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
