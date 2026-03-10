package user

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for user persistence
type Repository interface {
	// Save persists a user
	Save(ctx context.Context, user *User) error

	// FindByID retrieves a user by ID
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)

	// FindByIDsBatch retrieves multiple users by IDs in a single query
	FindByIDsBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*User, error)

	// FindByEmail retrieves a user by email
	FindByEmail(ctx context.Context, email string) (*User, error)

	// FindByTenantID retrieves all users for a tenant
	FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*User, error)

	// FindAll retrieves all users in the system (used by system administrators)
	FindAll(ctx context.Context) ([]*User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *User) error

	// Delete removes a user
	Delete(ctx context.Context, id uuid.UUID) error

	// ExistsByEmail checks if a user exists by email
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// CountByTenantID counts users for a tenant
	CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error)

	// GetTotalUserCount returns the total number of users in the system
	GetTotalUserCount(ctx context.Context) (int, error)

	// GetActiveUserCount returns the number of users active within the specified number of days
	GetActiveUserCount(ctx context.Context, days int) (int, error)
}

// UserInvitationRepository defines the interface for user invitation persistence
type UserInvitationRepository interface {
	// CreateInvitation creates a new user invitation
	CreateInvitation(ctx context.Context, invitation *UserInvitation, plainToken string) error

	// GetInvitationByToken retrieves a user invitation by token
	GetInvitationByToken(ctx context.Context, token string) (*UserInvitation, error)

	// GetInvitationByID retrieves a user invitation by ID
	GetInvitationByID(ctx context.Context, id uuid.UUID) (*UserInvitation, error)

	// ListInvitationsByTenant retrieves all invitations for a tenant
	ListInvitationsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*UserInvitation, error)

	// UpdateInvitationStatus updates the status of a user invitation
	UpdateInvitationStatus(ctx context.Context, id uuid.UUID, status UserInvitationStatus, acceptedAt *time.Time) error

	// UpdateInvitationToken updates the token hash for a user invitation
	UpdateInvitationToken(ctx context.Context, id uuid.UUID, tokenHash string) error

	// DeleteInvitation deletes a user invitation
	DeleteInvitation(ctx context.Context, id uuid.UUID) error

	// ExistsByEmailAndTenant checks if an invitation exists for the given email and tenant
	ExistsByEmailAndTenant(ctx context.Context, email string, tenantID uuid.UUID) (bool, error)
}

// PasswordResetTokenRepository defines the interface for password reset token persistence
type PasswordResetTokenRepository interface {
	// Save persists a password reset token
	Save(ctx context.Context, token *PasswordResetToken) error

	// FindByTokenHash retrieves a password reset token by its hash
	FindByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)

	// FindByUserID retrieves the most recent active password reset token for a user
	FindByUserID(ctx context.Context, userID uuid.UUID) (*PasswordResetToken, error)

	// Update updates an existing password reset token
	Update(ctx context.Context, token *PasswordResetToken) error

	// Delete removes a password reset token
	Delete(ctx context.Context, id uuid.UUID) error

	// CleanupExpiredTokens removes expired tokens from the database
	CleanupExpiredTokens(ctx context.Context) error
}
