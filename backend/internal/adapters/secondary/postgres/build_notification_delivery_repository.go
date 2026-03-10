package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	buildnotifications "github.com/srikarm/image-factory/internal/application/buildnotifications"
	"go.uber.org/zap"
)

type BuildNotificationDeliveryRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewBuildNotificationDeliveryRepository(db *sqlx.DB, logger *zap.Logger) *BuildNotificationDeliveryRepository {
	return &BuildNotificationDeliveryRepository{
		db:     db,
		logger: logger,
	}
}

func (r *BuildNotificationDeliveryRepository) ListProjectMemberUserIDs(ctx context.Context, projectID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT DISTINCT user_id
		FROM project_members
		WHERE project_id = $1
		ORDER BY user_id`

	rows, err := r.db.QueryxContext(ctx, query, projectID)
	if err != nil {
		r.logger.Error("Failed to list project member user ids", zap.String("project_id", projectID.String()), zap.Error(err))
		return nil, fmt.Errorf("failed to list project member user ids: %w", err)
	}
	defer rows.Close()

	var out []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if scanErr := rows.Scan(&userID); scanErr != nil {
			return nil, fmt.Errorf("failed to scan project member user id: %w", scanErr)
		}
		out = append(out, userID)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("failed reading project member user ids: %w", rowsErr)
	}
	return out, nil
}

func (r *BuildNotificationDeliveryRepository) ListTenantAdminUserIDs(ctx context.Context, tenantID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		WITH tenant_admin_users AS (
			SELECT DISTINCT ura.user_id
			FROM user_role_assignments ura
			INNER JOIN rbac_roles rr ON rr.id = ura.role_id
			WHERE ura.tenant_id = $1
			  AND REPLACE(LOWER(rr.name), ' ', '_') IN ('owner', 'administrator')
			UNION
			SELECT DISTINCT gm.user_id
			FROM group_members gm
			INNER JOIN tenant_groups tg ON tg.id = gm.group_id
			WHERE tg.tenant_id = $1
			  AND tg.status = 'active'
			  AND tg.role_type IN ('owner', 'administrator')
			  AND gm.removed_at IS NULL
		)
		SELECT user_id
		FROM tenant_admin_users
		ORDER BY user_id`

	rows, err := r.db.QueryxContext(ctx, query, tenantID)
	if err != nil {
		r.logger.Error("Failed to list tenant admin user ids", zap.String("tenant_id", tenantID.String()), zap.Error(err))
		return nil, fmt.Errorf("failed to list tenant admin user ids: %w", err)
	}
	defer rows.Close()

	var out []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if scanErr := rows.Scan(&userID); scanErr != nil {
			return nil, fmt.Errorf("failed to scan tenant admin user id: %w", scanErr)
		}
		out = append(out, userID)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("failed reading tenant admin user ids: %w", rowsErr)
	}
	return out, nil
}

func (r *BuildNotificationDeliveryRepository) ListSystemAdminUserIDs(ctx context.Context) ([]uuid.UUID, error) {
	query := `
		WITH system_admin_users AS (
			SELECT DISTINCT ura.user_id
			FROM user_role_assignments ura
			INNER JOIN rbac_roles rr ON rr.id = ura.role_id
			WHERE REPLACE(LOWER(rr.name), ' ', '_') = 'system_administrator'
			  AND (rr.tenant_id IS NULL OR rr.is_system = true)
			UNION
			SELECT DISTINCT gm.user_id
			FROM group_members gm
			INNER JOIN tenant_groups tg ON tg.id = gm.group_id
			WHERE tg.status = 'active'
			  AND tg.role_type = 'system_administrator'
			  AND gm.removed_at IS NULL
		)
		SELECT user_id
		FROM system_admin_users
		ORDER BY user_id`

	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		r.logger.Error("Failed to list system admin user ids", zap.Error(err))
		return nil, fmt.Errorf("failed to list system admin user ids: %w", err)
	}
	defer rows.Close()

	var out []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if scanErr := rows.Scan(&userID); scanErr != nil {
			return nil, fmt.Errorf("failed to scan system admin user id: %w", scanErr)
		}
		out = append(out, userID)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("failed reading system admin user ids: %w", rowsErr)
	}
	return out, nil
}

func (r *BuildNotificationDeliveryRepository) ListSecurityReviewerUserIDs(ctx context.Context) ([]uuid.UUID, error) {
	query := `
		WITH reviewer_users AS (
			SELECT DISTINCT ura.user_id
			FROM user_role_assignments ura
			INNER JOIN rbac_roles rr ON rr.id = ura.role_id
			WHERE REPLACE(LOWER(rr.name), ' ', '_') = 'security_reviewer'
			  AND (rr.tenant_id IS NULL OR rr.is_system = true)
			UNION
			SELECT DISTINCT gm.user_id
			FROM group_members gm
			INNER JOIN tenant_groups tg ON tg.id = gm.group_id
			WHERE tg.status = 'active'
			  AND tg.role_type = 'security_reviewer'
			  AND gm.removed_at IS NULL
		)
		SELECT user_id
		FROM reviewer_users
		ORDER BY user_id`

	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		r.logger.Error("Failed to list security reviewer user ids", zap.Error(err))
		return nil, fmt.Errorf("failed to list security reviewer user ids: %w", err)
	}
	defer rows.Close()

	var out []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if scanErr := rows.Scan(&userID); scanErr != nil {
			return nil, fmt.Errorf("failed to scan security reviewer user id: %w", scanErr)
		}
		out = append(out, userID)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("failed reading security reviewer user ids: %w", rowsErr)
	}
	return out, nil
}

func (r *BuildNotificationDeliveryRepository) ListUserEmailsByIDs(ctx context.Context, userIDs []uuid.UUID) ([]string, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	query, args, err := sqlx.In(`
		SELECT DISTINCT email
		FROM users
		WHERE id IN (?)
		  AND status = 'active'
		ORDER BY email`, userIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to build user email query: %w", err)
	}
	query = r.db.Rebind(query)

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to list user emails", zap.Error(err))
		return nil, fmt.Errorf("failed to list user emails: %w", err)
	}
	defer rows.Close()

	out := make([]string, 0, len(userIDs))
	for rows.Next() {
		var email string
		if scanErr := rows.Scan(&email); scanErr != nil {
			return nil, fmt.Errorf("failed to scan user email: %w", scanErr)
		}
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}
		out = append(out, email)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("failed reading user emails: %w", rowsErr)
	}
	return out, nil
}

func (r *BuildNotificationDeliveryRepository) InsertInAppNotifications(ctx context.Context, rows []buildnotifications.InAppNotificationRow) error {
	if len(rows) == 0 {
		return nil
	}

	query := `
		INSERT INTO notifications (
			id, user_id, tenant_id, title, message,
			notification_type, related_resource_type, related_resource_id,
			channel, created_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,NOW()
		)`

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin notification transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, row := range rows {
		if _, execErr := tx.ExecContext(
			ctx,
			query,
			row.ID,
			row.UserID,
			row.TenantID,
			row.Title,
			row.Message,
			row.NotificationType,
			row.RelatedResourceType,
			row.RelatedResourceID,
			row.Channel,
		); execErr != nil {
			r.logger.Error("Failed to insert in-app notification",
				zap.String("notification_id", row.ID.String()),
				zap.String("user_id", row.UserID.String()),
				zap.String("tenant_id", row.TenantID.String()),
				zap.Error(execErr))
			return fmt.Errorf("failed to insert in-app notification: %w", execErr)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit notification transaction: %w", err)
	}
	return nil
}

func (r *BuildNotificationDeliveryRepository) TryClaimImageImportNotification(ctx context.Context, tenantID, userID uuid.UUID, eventType, idempotencyKey string) (bool, error) {
	query := `
		INSERT INTO image_import_notification_receipts (
			id, tenant_id, user_id, event_type, idempotency_key, created_at
		) VALUES (
			$1,$2,$3,$4,$5,NOW()
		)
		ON CONFLICT (tenant_id, user_id, event_type, idempotency_key) DO NOTHING`

	res, err := r.db.ExecContext(ctx, query, uuid.New(), tenantID, userID, strings.TrimSpace(eventType), strings.TrimSpace(idempotencyKey))
	if err != nil {
		return false, fmt.Errorf("failed to claim image import notification idempotency key: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed reading image import notification claim rows: %w", err)
	}
	return affected == 1, nil
}

func (r *BuildNotificationDeliveryRepository) DeleteImageImportNotificationReceiptsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	query := `
		DELETE FROM image_import_notification_receipts
		WHERE created_at < $1`

	res, err := r.db.ExecContext(ctx, query, cutoff.UTC())
	if err != nil {
		return 0, fmt.Errorf("failed to delete image import notification receipts: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed reading deleted image import notification receipts count: %w", err)
	}
	return affected, nil
}

func (r *BuildNotificationDeliveryRepository) CountImageImportNotificationReceipts(ctx context.Context, tenantID uuid.UUID, eventType, idempotencyKey string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM image_import_notification_receipts
		WHERE tenant_id = $1
		  AND event_type = $2
		  AND idempotency_key = $3`

	var count int
	if err := r.db.GetContext(ctx, &count, query, tenantID, strings.TrimSpace(eventType), strings.TrimSpace(idempotencyKey)); err != nil {
		return 0, fmt.Errorf("failed to count image import notification receipts: %w", err)
	}
	return count, nil
}

func (r *BuildNotificationDeliveryRepository) CountImageImportInAppNotifications(ctx context.Context, tenantID, importID uuid.UUID, notificationType string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM notifications
		WHERE tenant_id = $1
		  AND related_resource_type = 'external_image_import'
		  AND related_resource_id = $2
		  AND notification_type = $3`

	var count int
	if err := r.db.GetContext(ctx, &count, query, tenantID, importID, strings.TrimSpace(notificationType)); err != nil {
		return 0, fmt.Errorf("failed to count image import in-app notifications: %w", err)
	}
	return count, nil
}
