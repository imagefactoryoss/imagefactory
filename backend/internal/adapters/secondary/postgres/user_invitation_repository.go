package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/user"
)

// UserInvitationRepository implements the user.UserInvitationRepository interface
type UserInvitationRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewUserInvitationRepository creates a new user invitation repository
func NewUserInvitationRepository(db *sqlx.DB, logger *zap.Logger) *UserInvitationRepository {
	return &UserInvitationRepository{
		db:     db,
		logger: logger,
	}
}

// userInvitationModel represents the database model for user invitation
type userInvitationModel struct {
	ID          uuid.UUID  `db:"id"`
	Email       string     `db:"email"`
	TenantID    uuid.UUID  `db:"tenant_id"`
	RoleID      *uuid.UUID `db:"role_id"`
	InviteToken string     `db:"invite_token"`
	InvitedByID uuid.UUID  `db:"invited_by_id"`
	Status      string     `db:"status"`
	AcceptedAt  *time.Time `db:"accepted_at"`
	ExpiresAt   time.Time  `db:"expires_at"`
	CreatedAt   time.Time  `db:"created_at"`
	Message     string     `db:"message"`
}

// CreateInvitation creates a new user invitation in the database
func (r *UserInvitationRepository) CreateInvitation(ctx context.Context, invitation *user.UserInvitation, plainToken string) error {
	query := `
		INSERT INTO user_invitations (
			id, email, tenant_id, role_id, invite_token, invited_by_id,
			status, expires_at, message
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`

	_, err := r.db.ExecContext(ctx, query,
		invitation.ID().String(),
		invitation.Email(),
		invitation.TenantID().String(),
		invitation.RoleID().String(),
		invitation.TokenHash(), // Store the hash, not the plain token
		invitation.InvitedBy().String(),
		string(invitation.Status()),
		invitation.ExpiresAt(),
		invitation.Message(),
	)

	if err != nil {
		r.logger.Error("Failed to create user invitation",
			zap.String("invitation_id", invitation.ID().String()),
			zap.String("email", invitation.Email()),
			zap.Error(err))
		return err
	}

	r.logger.Info("User invitation created successfully",
		zap.String("invitation_id", invitation.ID().String()),
		zap.String("email", invitation.Email()))

	return nil
}

// GetInvitationByToken retrieves a user invitation by token
func (r *UserInvitationRepository) GetInvitationByToken(ctx context.Context, token string) (*user.UserInvitation, error) {
	// Hash the provided token to match against stored hash
	hasher := sha256.New()
	hasher.Write([]byte(token))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	query := `
		SELECT id, email, tenant_id, role_id, invite_token, invited_by_id,
			   status, accepted_at, expires_at, created_at, message
		FROM user_invitations
		WHERE invite_token = $1`

	var model userInvitationModel
	err := r.db.GetContext(ctx, &model, query, tokenHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, user.ErrInvitationNotFound
		}
		r.logger.Error("Failed to get invitation by token", zap.String("token_hash", tokenHash), zap.Error(err))
		return nil, err
	}

	return r.modelToUserInvitation(&model), nil
}

// GetInvitationByID retrieves a user invitation by ID
func (r *UserInvitationRepository) GetInvitationByID(ctx context.Context, id uuid.UUID) (*user.UserInvitation, error) {
	query := `
		SELECT id, email, tenant_id, role_id, invite_token, invited_by_id,
			   status, accepted_at, expires_at, created_at, message
		FROM user_invitations
		WHERE id = $1`

	var model userInvitationModel
	err := r.db.GetContext(ctx, &model, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, user.ErrInvitationNotFound
		}
		r.logger.Error("Failed to get invitation by ID", zap.String("id", id.String()), zap.Error(err))
		return nil, err
	}

	return r.modelToUserInvitation(&model), nil
}

// ListInvitationsByTenant retrieves all invitations for a tenant
func (r *UserInvitationRepository) ListInvitationsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*user.UserInvitation, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}
	query := `
		SELECT id, email, tenant_id, role_id, invite_token, invited_by_id,
		       status, accepted_at, expires_at, created_at, message
		FROM user_invitations
		WHERE tenant_id = $1
		ORDER BY created_at DESC`
	args := []interface{}{tenantID}

	var models []userInvitationModel
	err := r.db.SelectContext(ctx, &models, query, args...)
	if err != nil {
		r.logger.Error("Failed to list invitations by tenant",
			zap.String("tenant_id", tenantID.String()), zap.Error(err))
		return nil, err
	}

	invitations := make([]*user.UserInvitation, len(models))
	for i, model := range models {
		invitations[i] = r.modelToUserInvitation(&model)
	}

	return invitations, nil
}

// UpdateInvitationStatus updates the status of a user invitation
func (r *UserInvitationRepository) UpdateInvitationStatus(ctx context.Context, id uuid.UUID, status user.UserInvitationStatus, acceptedAt *time.Time) error {
	query := `
		UPDATE user_invitations
		SET status = $1, accepted_at = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3`

	_, err := r.db.ExecContext(ctx, query, string(status), acceptedAt, id)
	if err != nil {
		r.logger.Error("Failed to update invitation status",
			zap.String("invitation_id", id.String()),
			zap.String("status", string(status)),
			zap.Error(err))
		return err
	}

	r.logger.Info("Invitation status updated",
		zap.String("invitation_id", id.String()),
		zap.String("status", string(status)))

	return nil
}

// UpdateInvitationToken updates the token hash for a user invitation
func (r *UserInvitationRepository) UpdateInvitationToken(ctx context.Context, id uuid.UUID, tokenHash string) error {
	query := `
		UPDATE user_invitations
		SET invite_token = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, tokenHash, id)
	if err != nil {
		r.logger.Error("Failed to update invitation token",
			zap.String("invitation_id", id.String()),
			zap.Error(err))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return user.ErrInvitationNotFound
	}

	r.logger.Info("Invitation token updated",
		zap.String("invitation_id", id.String()))

	return nil
}

// DeleteInvitation deletes a user invitation
func (r *UserInvitationRepository) DeleteInvitation(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM user_invitations WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete invitation", zap.String("id", id.String()), zap.Error(err))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return user.ErrInvitationNotFound
	}

	r.logger.Info("Invitation deleted", zap.String("invitation_id", id.String()))
	return nil
}

// ExistsByEmailAndTenant checks if an invitation exists for the given email and tenant
func (r *UserInvitationRepository) ExistsByEmailAndTenant(ctx context.Context, email string, tenantID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_invitations
			WHERE email = $1 AND tenant_id = $2 AND status = 'pending'
		)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, email, tenantID)
	if err != nil {
		r.logger.Error("Failed to check invitation existence",
			zap.String("email", email),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err))
		return false, err
	}

	return exists, nil
}

// modelToUserInvitation converts a database model to a domain UserInvitation
func (r *UserInvitationRepository) modelToUserInvitation(model *userInvitationModel) *user.UserInvitation {
	// Parse role ID
	var roleID uuid.UUID
	if model.RoleID != nil {
		roleID = *model.RoleID
	}

	invitation, err := user.NewUserInvitationFromExisting(
		model.ID,
		model.TenantID,
		model.Email,
		model.InviteToken,
		roleID,
		model.InvitedByID,
		user.UserInvitationStatus(model.Status),
		model.ExpiresAt,
		model.AcceptedAt,
		nil, // acceptedBy - not in DB yet
		model.Message,
		model.CreatedAt,
		model.CreatedAt, // updatedAt - use createdAt as default
		1,               // version - default to 1
	)

	if err != nil {
		r.logger.Error("Failed to create UserInvitation from model",
			zap.String("id", model.ID.String()), zap.Error(err))
		return nil
	}

	return invitation
}
