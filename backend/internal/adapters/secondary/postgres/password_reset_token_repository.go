package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/user"
)

// nullStringToString converts sql.NullString to string
func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// PasswordResetTokenRepository implements the user.PasswordResetTokenRepository interface
type PasswordResetTokenRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewPasswordResetTokenRepository creates a new password reset token repository
func NewPasswordResetTokenRepository(db *sqlx.DB, logger *zap.Logger) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{
		db:     db,
		logger: logger,
	}
}

// passwordResetTokenModel represents the database model for password reset tokens
type passwordResetTokenModel struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	Status    string    `db:"status"`
	ExpiresAt time.Time `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	IPAddress sql.NullString `db:"ip_address"`
	UserAgent sql.NullString `db:"user_agent"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Version   int       `db:"version"`
}

// Save persists a password reset token
func (r *PasswordResetTokenRepository) Save(ctx context.Context, token *user.PasswordResetToken) error {
	model := passwordResetTokenModel{
		ID:        token.ID(),
		UserID:    token.UserID(),
		TokenHash: token.TokenHash(),
		Status:    string(token.Status()),
		ExpiresAt: token.ExpiresAt(),
		UsedAt:    token.UsedAt(),
		IPAddress: stringToNullString(token.IPAddress()),
		UserAgent: stringToNullString(token.UserAgent()),
		CreatedAt: token.CreatedAt(),
		UpdatedAt: token.UpdatedAt(),
		Version:   token.Version(),
	}

	query := `
		INSERT INTO password_reset_tokens (
			id, user_id, token_hash, status, expires_at, used_at,
			ip_address, user_agent, created_at, updated_at, version
		) VALUES (
			:id, :user_id, :token_hash, :status, :expires_at, :used_at,
			:ip_address, :user_agent, :created_at, :updated_at, :version
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			used_at = EXCLUDED.used_at,
			ip_address = EXCLUDED.ip_address,
			user_agent = EXCLUDED.user_agent,
			updated_at = EXCLUDED.updated_at,
			version = EXCLUDED.version
		WHERE password_reset_tokens.version < EXCLUDED.version
	`

	_, err := r.db.NamedExecContext(ctx, query, model)
	if err != nil {
		r.logger.Error("Failed to save password reset token",
			zap.String("tokenID", token.ID().String()),
			zap.Error(err))
		return err
	}

	r.logger.Debug("Password reset token saved",
		zap.String("tokenID", token.ID().String()),
		zap.String("userID", token.UserID().String()))
	return nil
}

// FindByTokenHash retrieves a password reset token by its hash
func (r *PasswordResetTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*user.PasswordResetToken, error) {
	var model passwordResetTokenModel
	query := `SELECT * FROM password_reset_tokens WHERE token_hash = $1`

	err := r.db.GetContext(ctx, &model, query, tokenHash)
	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("Password reset token not found", zap.String("tokenHash", tokenHash))
			return nil, user.ErrTokenNotFound
		}
		r.logger.Error("Failed to find password reset token by hash", zap.Error(err))
		return nil, err
	}

	return user.NewPasswordResetTokenFromExisting(
		model.ID,
		model.UserID,
		model.TokenHash,
		user.PasswordResetTokenStatus(model.Status),
		model.ExpiresAt,
		model.UsedAt,
		nullStringToString(model.IPAddress),
		nullStringToString(model.UserAgent),
		model.CreatedAt,
		model.UpdatedAt,
		model.Version,
	)
}

// FindByUserID retrieves the most recent active password reset token for a user
func (r *PasswordResetTokenRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*user.PasswordResetToken, error) {
	var model passwordResetTokenModel
	query := `
		SELECT * FROM password_reset_tokens
		WHERE user_id = $1 AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1
	`

	err := r.db.GetContext(ctx, &model, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Debug("No active password reset token found for user", zap.String("userID", userID.String()))
			return nil, user.ErrTokenNotFound
		}
		r.logger.Error("Failed to find password reset token by user ID", zap.Error(err))
		return nil, err
	}

	return user.NewPasswordResetTokenFromExisting(
		model.ID,
		model.UserID,
		model.TokenHash,
		user.PasswordResetTokenStatus(model.Status),
		model.ExpiresAt,
		model.UsedAt,
		nullStringToString(model.IPAddress),
		nullStringToString(model.UserAgent),
		model.CreatedAt,
		model.UpdatedAt,
		model.Version,
	)
}

// Update updates an existing password reset token
func (r *PasswordResetTokenRepository) Update(ctx context.Context, token *user.PasswordResetToken) error {
	model := passwordResetTokenModel{
		ID:        token.ID(),
		UserID:    token.UserID(),
		TokenHash: token.TokenHash(),
		Status:    string(token.Status()),
		ExpiresAt: token.ExpiresAt(),
		UsedAt:    token.UsedAt(),
		IPAddress: stringToNullString(token.IPAddress()),
		UserAgent: stringToNullString(token.UserAgent()),
		CreatedAt: token.CreatedAt(),
		UpdatedAt: token.UpdatedAt(),
		Version:   token.Version(),
	}

	query := `
		UPDATE password_reset_tokens SET
			status = :status,
			used_at = :used_at,
			ip_address = :ip_address,
			user_agent = :user_agent,
			updated_at = :updated_at,
			version = :version
		WHERE id = :id AND version = :version - 1
	`

	result, err := r.db.NamedExecContext(ctx, query, model)
	if err != nil {
		r.logger.Error("Failed to update password reset token",
			zap.String("tokenID", token.ID().String()),
			zap.Error(err))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected for password reset token update", zap.Error(err))
		return err
	}

	if rowsAffected == 0 {
		r.logger.Warn("Password reset token update failed - concurrent modification or not found",
			zap.String("tokenID", token.ID().String()))
		return user.ErrTokenNotFound
	}

	r.logger.Debug("Password reset token updated",
		zap.String("tokenID", token.ID().String()))
	return nil
}

// Delete removes a password reset token
func (r *PasswordResetTokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM password_reset_tokens WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete password reset token",
			zap.String("tokenID", id.String()),
			zap.Error(err))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected for password reset token deletion", zap.Error(err))
		return err
	}

	if rowsAffected == 0 {
		r.logger.Warn("Password reset token deletion failed - not found",
			zap.String("tokenID", id.String()))
		return user.ErrTokenNotFound
	}

	r.logger.Debug("Password reset token deleted",
		zap.String("tokenID", id.String()))
	return nil
}

// CleanupExpiredTokens removes expired password reset tokens
func (r *PasswordResetTokenRepository) CleanupExpiredTokens(ctx context.Context) error {
	query := `DELETE FROM password_reset_tokens WHERE expires_at < NOW() AND status != 'used'`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		r.logger.Error("Failed to cleanup expired password reset tokens", zap.Error(err))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected for expired token cleanup", zap.Error(err))
		return err
	}

	r.logger.Info("Cleaned up expired password reset tokens",
		zap.Int64("count", rowsAffected))
	return nil
}