package user

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Domain errors for user invitations
var (
	ErrInvitationNotFound      = errors.New("user invitation not found")
	ErrInvitationExpired       = errors.New("invitation has expired")
	ErrInvitationAlreadyUsed   = errors.New("invitation has already been used")
	ErrInvitationAlreadyExists = errors.New("invitation already exists for this email")
	ErrInvalidInvitationToken  = errors.New("invalid invitation token")
)

// UserInvitationStatus represents the status of a user invitation
type UserInvitationStatus string

const (
	UserInvitationStatusPending   UserInvitationStatus = "pending"
	UserInvitationStatusAccepted  UserInvitationStatus = "accepted"
	UserInvitationStatusExpired   UserInvitationStatus = "expired"
	UserInvitationStatusCancelled UserInvitationStatus = "cancelled"
	UserInvitationStatusRevoked   UserInvitationStatus = "revoked"
)

// UserInvitation represents a user invitation aggregate
type UserInvitation struct {
	id            uuid.UUID
	tenantID      uuid.UUID
	email         string
	tokenHash     string
	roleID        uuid.UUID
	invitedBy     uuid.UUID
	status        UserInvitationStatus
	expiresAt     time.Time
	acceptedAt    *time.Time
	acceptedBy    *uuid.UUID
	message       string
	createdAt     time.Time
	updatedAt     time.Time
	version       int
}

// NewUserInvitation creates a new user invitation
func NewUserInvitation(
	tenantID uuid.UUID,
	email string,
	roleID uuid.UUID,
	invitedBy uuid.UUID,
	message string,
) (*UserInvitation, string, error) {
	if tenantID == uuid.Nil {
		return nil, "", errors.New("tenant ID is required")
	}
	if err := ValidateEmail(email); err != nil {
		return nil, "", err
	}
	if roleID == uuid.Nil {
		return nil, "", errors.New("role ID is required")
	}
	if invitedBy == uuid.Nil {
		return nil, "", errors.New("invited by user ID is required")
	}

	// Generate a secure random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", errors.New("failed to generate secure token")
	}
	token := hex.EncodeToString(tokenBytes)

	// Hash the token for storage
	hasher := sha256.New()
	hasher.Write([]byte(token))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	now := time.Now().UTC()
	expiresAt := now.Add(7 * 24 * time.Hour) // 7 day expiry

	return &UserInvitation{
		id:          uuid.New(),
		tenantID:    tenantID,
		email:       email,
		tokenHash:   tokenHash,
		roleID:      roleID,
		invitedBy:   invitedBy,
		status:      UserInvitationStatusPending,
		expiresAt:   expiresAt,
		message:     message,
		createdAt:   now,
		updatedAt:   now,
		version:     1,
	}, token, nil
}

// NewUserInvitationFromExisting creates a user invitation from existing data
func NewUserInvitationFromExisting(
	id, tenantID uuid.UUID,
	email, plainToken string,
	roleID, invitedBy uuid.UUID,
	status UserInvitationStatus,
	expiresAt time.Time,
	acceptedAt *time.Time,
	acceptedBy *uuid.UUID,
	message string,
	createdAt, updatedAt time.Time,
	version int,
) (*UserInvitation, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid invitation ID")
	}
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant ID is required")
	}
	if email == "" {
		return nil, ErrInvalidEmail
	}
	if roleID == uuid.Nil {
		return nil, errors.New("role ID is required")
	}
	if invitedBy == uuid.Nil {
		return nil, errors.New("invited by user ID is required")
	}

	// Hash the token for storage
	hasher := sha256.New()
	hasher.Write([]byte(plainToken))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	return &UserInvitation{
		id:          id,
		tenantID:    tenantID,
		email:       email,
		tokenHash:   tokenHash,
		roleID:      roleID,
		invitedBy:   invitedBy,
		status:      status,
		expiresAt:   expiresAt,
		acceptedAt:  acceptedAt,
		acceptedBy:  acceptedBy,
		message:     message,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		version:     version,
	}, nil
}

// ID returns the invitation ID
func (i *UserInvitation) ID() uuid.UUID {
	return i.id
}

// TenantID returns the tenant ID
func (i *UserInvitation) TenantID() uuid.UUID {
	return i.tenantID
}

// Email returns the invited email
func (i *UserInvitation) Email() string {
	return i.email
}

// RoleID returns the assigned role ID
func (i *UserInvitation) RoleID() uuid.UUID {
	return i.roleID
}

// InvitedBy returns the user who sent the invitation
func (i *UserInvitation) InvitedBy() uuid.UUID {
	return i.invitedBy
}

// Status returns the invitation status
func (i *UserInvitation) Status() UserInvitationStatus {
	return i.status
}

// ExpiresAt returns the expiration time
func (i *UserInvitation) ExpiresAt() time.Time {
	return i.expiresAt
}

// IsExpired returns true if the invitation has expired
func (i *UserInvitation) IsExpired() bool {
	return time.Now().UTC().After(i.expiresAt)
}

// IsPending returns true if the invitation is still pending
func (i *UserInvitation) IsPending() bool {
	return i.status == UserInvitationStatusPending && !i.IsExpired()
}

// VerifyToken verifies a provided token against the stored hash
func (i *UserInvitation) VerifyToken(token string) error {
	if !i.IsPending() {
		switch i.status {
		case UserInvitationStatusAccepted:
			return ErrInvitationAlreadyUsed
		case UserInvitationStatusExpired:
			return ErrInvitationExpired
		case UserInvitationStatusCancelled, UserInvitationStatusRevoked:
			return ErrInvalidInvitationToken
		}
		return ErrInvalidInvitationToken
	}

	hasher := sha256.New()
	hasher.Write([]byte(token))
	providedHash := hex.EncodeToString(hasher.Sum(nil))

	if providedHash != i.tokenHash {
		return ErrInvalidInvitationToken
	}

	return nil
}

// AcceptInvitation marks the invitation as accepted by a user
func (i *UserInvitation) AcceptInvitation(acceptedBy uuid.UUID) error {
	if !i.IsPending() {
		if i.status == UserInvitationStatusAccepted {
			return ErrInvitationAlreadyUsed
		}
		return ErrInvitationExpired
	}

	now := time.Now().UTC()
	i.status = UserInvitationStatusAccepted
	i.acceptedAt = &now
	i.acceptedBy = &acceptedBy
	i.updatedAt = now
	i.version++
	return nil
}

// CancelInvitation marks the invitation as cancelled
func (i *UserInvitation) CancelInvitation() {
	now := time.Now().UTC()
	i.status = UserInvitationStatusCancelled
	i.updatedAt = now
	i.version++
}
// TokenHash returns the hashed token
func (i *UserInvitation) TokenHash() string {
	return i.tokenHash
}
// RevokeInvitation marks the invitation as revoked
func (i *UserInvitation) RevokeInvitation() {
	now := time.Now().UTC()
	i.status = UserInvitationStatusRevoked
	i.updatedAt = now
	i.version++
}

// RegenerateToken generates a new token for the invitation
func (i *UserInvitation) RegenerateToken() (string, error) {
	// Generate a secure random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", errors.New("failed to generate secure token")
	}
	token := hex.EncodeToString(tokenBytes)

	// Hash the token for storage
	hasher := sha256.New()
	hasher.Write([]byte(token))
	i.tokenHash = hex.EncodeToString(hasher.Sum(nil))

	now := time.Now().UTC()
	i.updatedAt = now
	i.version++

	return token, nil
}

// Message returns the invitation message
func (i *UserInvitation) Message() string {
	return i.message
}

// AcceptedAt returns when the invitation was accepted
func (i *UserInvitation) AcceptedAt() *time.Time {
	return i.acceptedAt
}

// AcceptedBy returns the user who accepted the invitation
func (i *UserInvitation) AcceptedBy() *uuid.UUID {
	return i.acceptedBy
}

// CreatedAt returns the creation time
func (i *UserInvitation) CreatedAt() time.Time {
	return i.createdAt
}

// UpdatedAt returns the last update time
func (i *UserInvitation) UpdatedAt() time.Time {
	return i.updatedAt
}

// Version returns the version for optimistic locking
func (i *UserInvitation) Version() int {
	return i.version
}