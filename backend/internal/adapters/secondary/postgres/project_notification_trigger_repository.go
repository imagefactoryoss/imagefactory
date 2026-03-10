package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/domain/buildnotification"
	"go.uber.org/zap"
)

type ProjectNotificationTriggerRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewProjectNotificationTriggerRepository(db *sqlx.DB, logger *zap.Logger) *ProjectNotificationTriggerRepository {
	return &ProjectNotificationTriggerRepository{db: db, logger: logger}
}

type projectNotificationTriggerModel struct {
	ID                   uuid.UUID      `db:"id"`
	TenantID             uuid.UUID      `db:"tenant_id"`
	ProjectID            uuid.UUID      `db:"project_id"`
	TriggerID            string         `db:"trigger_id"`
	Enabled              bool           `db:"enabled"`
	Channels             []byte         `db:"channels"`
	RecipientPolicy      string         `db:"recipient_policy"`
	CustomRecipientUsers sql.NullString `db:"custom_recipient_user_ids"`
	SeverityOverride     sql.NullString `db:"severity_override"`
	CreatedAt            time.Time      `db:"created_at"`
	UpdatedAt            time.Time      `db:"updated_at"`
}

type tenantNotificationTriggerModel struct {
	ID                   uuid.UUID      `db:"id"`
	TenantID             uuid.UUID      `db:"tenant_id"`
	TriggerID            string         `db:"trigger_id"`
	Enabled              bool           `db:"enabled"`
	Channels             []byte         `db:"channels"`
	RecipientPolicy      string         `db:"recipient_policy"`
	CustomRecipientUsers sql.NullString `db:"custom_recipient_user_ids"`
	SeverityOverride     sql.NullString `db:"severity_override"`
}

func (r *ProjectNotificationTriggerRepository) ListTenantTriggerPreferences(ctx context.Context, tenantID uuid.UUID) ([]buildnotification.TenantTriggerPreference, error) {
	query := `
		SELECT id, tenant_id, trigger_id, enabled, channels,
		       recipient_policy, custom_recipient_user_ids::text AS custom_recipient_user_ids,
		       severity_override
		FROM tenant_notification_trigger_prefs
		WHERE tenant_id = $1
		ORDER BY trigger_id ASC`

	var models []tenantNotificationTriggerModel
	if err := r.db.SelectContext(ctx, &models, query, tenantID); err != nil {
		r.logger.Error("Failed to list tenant notification trigger preferences", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil, err
	}

	out := make([]buildnotification.TenantTriggerPreference, 0, len(models))
	for _, model := range models {
		pref, err := r.tenantModelToDomain(model)
		if err != nil {
			r.logger.Error("Failed to decode tenant notification trigger preference", zap.Error(err), zap.String("trigger_id", model.TriggerID))
			return nil, err
		}
		out = append(out, pref)
	}
	return out, nil
}

func (r *ProjectNotificationTriggerRepository) UpsertTenantTriggerPreferences(ctx context.Context, tenantID, actorID uuid.UUID, prefs []buildnotification.TenantTriggerPreference) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	query := `
		INSERT INTO tenant_notification_trigger_prefs (
			id, tenant_id, trigger_id, enabled, channels,
			recipient_policy, custom_recipient_user_ids, severity_override,
			created_by, updated_by, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5::jsonb,$6,$7::jsonb,$8,$9,$10,NOW(),NOW()
		)
		ON CONFLICT (tenant_id, trigger_id)
		DO UPDATE SET
			enabled = EXCLUDED.enabled,
			channels = EXCLUDED.channels,
			recipient_policy = EXCLUDED.recipient_policy,
			custom_recipient_user_ids = EXCLUDED.custom_recipient_user_ids,
			severity_override = EXCLUDED.severity_override,
			updated_by = EXCLUDED.updated_by,
			updated_at = NOW()`

	for _, pref := range prefs {
		id := pref.ID
		if id == uuid.Nil {
			id = uuid.New()
		}
		channelsJSON, err := json.Marshal(pref.Channels)
		if err != nil {
			return fmt.Errorf("marshal channels: %w", err)
		}
		var usersJSON interface{}
		if len(pref.CustomRecipientUsers) > 0 {
			encodedUsers, marshalErr := json.Marshal(pref.CustomRecipientUsers)
			if marshalErr != nil {
				return fmt.Errorf("marshal custom recipients: %w", marshalErr)
			}
			usersJSON = string(encodedUsers)
		} else {
			// Persist SQL NULL (not JSON null) so DB check constraints treat it as unset.
			usersJSON = nil
		}

		var severity interface{}
		if pref.SeverityOverride != nil {
			severity = string(*pref.SeverityOverride)
		}

		if _, err := tx.ExecContext(
			ctx,
			query,
			id,
			tenantID,
			string(pref.TriggerID),
			pref.Enabled,
			string(channelsJSON),
			string(pref.RecipientPolicy),
			usersJSON,
			severity,
			actorID,
			actorID,
		); err != nil {
			r.logger.Error("Failed to upsert tenant notification trigger preference", zap.Error(err), zap.String("trigger_id", string(pref.TriggerID)))
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *ProjectNotificationTriggerRepository) ListProjectTriggerPreferences(ctx context.Context, tenantID, projectID uuid.UUID) ([]buildnotification.ProjectTriggerPreference, error) {
	query := `
		SELECT id, tenant_id, project_id, trigger_id, enabled, channels,
		       recipient_policy, custom_recipient_user_ids::text AS custom_recipient_user_ids,
		       severity_override, created_at, updated_at
		FROM project_notification_trigger_prefs
		WHERE tenant_id = $1 AND project_id = $2
		ORDER BY trigger_id ASC`

	var models []projectNotificationTriggerModel
	if err := r.db.SelectContext(ctx, &models, query, tenantID, projectID); err != nil {
		r.logger.Error("Failed to list project notification trigger preferences", zap.Error(err), zap.String("tenant_id", tenantID.String()), zap.String("project_id", projectID.String()))
		return nil, err
	}

	out := make([]buildnotification.ProjectTriggerPreference, 0, len(models))
	for _, model := range models {
		pref, err := r.modelToDomain(model)
		if err != nil {
			r.logger.Error("Failed to decode project notification trigger preference", zap.Error(err), zap.String("trigger_id", model.TriggerID))
			return nil, err
		}
		out = append(out, pref)
	}
	return out, nil
}

func (r *ProjectNotificationTriggerRepository) UpsertProjectTriggerPreferences(ctx context.Context, tenantID, projectID, actorID uuid.UUID, prefs []buildnotification.ProjectTriggerPreference) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	query := `
		INSERT INTO project_notification_trigger_prefs (
			id, tenant_id, project_id, trigger_id, enabled, channels,
			recipient_policy, custom_recipient_user_ids, severity_override,
			created_by, updated_by, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6::jsonb,$7,$8::jsonb,$9,$10,$11,NOW(),NOW()
		)
		ON CONFLICT (project_id, trigger_id)
		DO UPDATE SET
			enabled = EXCLUDED.enabled,
			channels = EXCLUDED.channels,
			recipient_policy = EXCLUDED.recipient_policy,
			custom_recipient_user_ids = EXCLUDED.custom_recipient_user_ids,
			severity_override = EXCLUDED.severity_override,
			updated_by = EXCLUDED.updated_by,
			updated_at = NOW()`

	for _, pref := range prefs {
		id := pref.ID
		if id == uuid.Nil {
			id = uuid.New()
		}

		channelsJSON, err := json.Marshal(pref.Channels)
		if err != nil {
			return fmt.Errorf("marshal channels: %w", err)
		}

		var usersJSON interface{}
		if len(pref.CustomRecipientUsers) > 0 {
			encodedUsers, marshalErr := json.Marshal(pref.CustomRecipientUsers)
			if marshalErr != nil {
				return fmt.Errorf("marshal custom recipients: %w", marshalErr)
			}
			usersJSON = string(encodedUsers)
		} else {
			// Persist SQL NULL (not JSON null) so DB check constraints treat it as unset.
			usersJSON = nil
		}

		var severity interface{}
		if pref.SeverityOverride != nil {
			severity = string(*pref.SeverityOverride)
		}

		if _, err := tx.ExecContext(
			ctx,
			query,
			id,
			tenantID,
			projectID,
			string(pref.TriggerID),
			pref.Enabled,
			string(channelsJSON),
			string(pref.RecipientPolicy),
			usersJSON,
			severity,
			actorID,
			actorID,
		); err != nil {
			r.logger.Error("Failed to upsert project notification trigger preference", zap.Error(err), zap.String("trigger_id", string(pref.TriggerID)))
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *ProjectNotificationTriggerRepository) DeleteProjectTriggerPreference(ctx context.Context, tenantID, projectID uuid.UUID, triggerID buildnotification.TriggerID) error {
	query := `
		DELETE FROM project_notification_trigger_prefs
		WHERE tenant_id = $1 AND project_id = $2 AND trigger_id = $3`
	_, err := r.db.ExecContext(ctx, query, tenantID, projectID, string(triggerID))
	if err != nil {
		r.logger.Error(
			"Failed to delete project notification trigger preference",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
			zap.String("project_id", projectID.String()),
			zap.String("trigger_id", string(triggerID)),
		)
		return err
	}
	return nil
}

func (r *ProjectNotificationTriggerRepository) modelToDomain(model projectNotificationTriggerModel) (buildnotification.ProjectTriggerPreference, error) {
	pref := buildnotification.ProjectTriggerPreference{
		ID:              model.ID,
		TenantID:        model.TenantID,
		ProjectID:       model.ProjectID,
		TriggerID:       buildnotification.TriggerID(model.TriggerID),
		Enabled:         model.Enabled,
		RecipientPolicy: buildnotification.RecipientPolicy(model.RecipientPolicy),
	}

	if len(model.Channels) > 0 {
		if err := json.Unmarshal(model.Channels, &pref.Channels); err != nil {
			return pref, err
		}
	}

	if model.CustomRecipientUsers.Valid && model.CustomRecipientUsers.String != "" && model.CustomRecipientUsers.String != "null" {
		if err := json.Unmarshal([]byte(model.CustomRecipientUsers.String), &pref.CustomRecipientUsers); err != nil {
			return pref, err
		}
	}

	if model.SeverityOverride.Valid {
		severity := buildnotification.Severity(model.SeverityOverride.String)
		pref.SeverityOverride = &severity
	}

	return pref, nil
}

func (r *ProjectNotificationTriggerRepository) tenantModelToDomain(model tenantNotificationTriggerModel) (buildnotification.TenantTriggerPreference, error) {
	pref := buildnotification.TenantTriggerPreference{
		ID:              model.ID,
		TenantID:        model.TenantID,
		TriggerID:       buildnotification.TriggerID(model.TriggerID),
		Enabled:         model.Enabled,
		RecipientPolicy: buildnotification.RecipientPolicy(model.RecipientPolicy),
	}

	if len(model.Channels) > 0 {
		if err := json.Unmarshal(model.Channels, &pref.Channels); err != nil {
			return pref, err
		}
	}

	if model.CustomRecipientUsers.Valid && model.CustomRecipientUsers.String != "" && model.CustomRecipientUsers.String != "null" {
		if err := json.Unmarshal([]byte(model.CustomRecipientUsers.String), &pref.CustomRecipientUsers); err != nil {
			return pref, err
		}
	}

	if model.SeverityOverride.Valid {
		severity := buildnotification.Severity(model.SeverityOverride.String)
		pref.SeverityOverride = &severity
	}

	return pref, nil
}
