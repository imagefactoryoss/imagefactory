package user

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Domain errors for password reset tokens
var (
	ErrInvalidToken         = errors.New("invalid password reset token")
	ErrTokenExpired         = errors.New("password reset token has expired")
	ErrTokenAlreadyUsed     = errors.New("password reset token has already been used")
	ErrTokenNotFound        = errors.New("password reset token not found")
	ErrTooManyResetAttempts = errors.New("too many password reset attempts")
)

// PasswordResetTokenStatus represents the status of a password reset token
type PasswordResetTokenStatus string

const (
	PasswordResetTokenStatusActive   PasswordResetTokenStatus = "active"
	PasswordResetTokenStatusUsed     PasswordResetTokenStatus = "used"
	PasswordResetTokenStatusExpired  PasswordResetTokenStatus = "expired"
	PasswordResetTokenStatusRevoked  PasswordResetTokenStatus = "revoked"
)

// PasswordResetToken represents a password reset token aggregate
type PasswordResetToken struct {
	id          uuid.UUID
	userID      uuid.UUID
	tokenHash   string
	status      PasswordResetTokenStatus
	expiresAt   time.Time
	usedAt      *time.Time
	ipAddress   string
	userAgent   string
	createdAt   time.Time
	updatedAt   time.Time
	version     int
}

// NewPasswordResetToken creates a new password reset token
func NewPasswordResetToken(userID uuid.UUID, ipAddress, userAgent string) (*PasswordResetToken, string, error) {
	if userID == uuid.Nil {
		return nil, "", ErrInvalidUserID
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
	expiresAt := now.Add(24 * time.Hour) // 24 hour expiry

	return &PasswordResetToken{
		id:          uuid.New(),
		userID:      userID,
		tokenHash:   tokenHash,
		status:      PasswordResetTokenStatusActive,
		expiresAt:   expiresAt,
		ipAddress:   ipAddress,
		userAgent:   userAgent,
		createdAt:   now,
		updatedAt:   now,
		version:     1,
	}, token, nil
}

// NewPasswordResetTokenFromExisting creates a password reset token from existing data
func NewPasswordResetTokenFromExisting(
	id, userID uuid.UUID,
	tokenHash string,
	status PasswordResetTokenStatus,
	expiresAt time.Time,
	usedAt *time.Time,
	ipAddress, userAgent string,
	createdAt, updatedAt time.Time,
	version int,
) (*PasswordResetToken, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid token ID")
	}
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	if tokenHash == "" {
		return nil, ErrInvalidToken
	}

	return &PasswordResetToken{
		id:          id,
		userID:      userID,
		tokenHash:   tokenHash,
		status:      status,
		expiresAt:   expiresAt,
		usedAt:      usedAt,
		ipAddress:   ipAddress,
		userAgent:   userAgent,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		version:     version,
	}, nil
}

// ID returns the token ID
func (t *PasswordResetToken) ID() uuid.UUID {
	return t.id
}

// UserID returns the associated user ID
func (t *PasswordResetToken) UserID() uuid.UUID {
	return t.userID
}

// Status returns the token status
func (t *PasswordResetToken) Status() PasswordResetTokenStatus {
	return t.status
}

// ExpiresAt returns the expiration time
func (t *PasswordResetToken) ExpiresAt() time.Time {
	return t.expiresAt
}

// UsedAt returns the time when the token was used
func (t *PasswordResetToken) UsedAt() *time.Time {
	return t.usedAt
}

// TokenHash returns the hashed token
func (t *PasswordResetToken) TokenHash() string {
	return t.tokenHash
}

// IPAddress returns the IP address associated with the token
func (t *PasswordResetToken) IPAddress() string {
	return t.ipAddress
}

// UserAgent returns the user agent associated with the token
func (t *PasswordResetToken) UserAgent() string {
	return t.userAgent
}

// Version returns the version for optimistic locking
func (t *PasswordResetToken) Version() int {
	return t.version
}

// IsExpired returns true if the token has expired
func (t *PasswordResetToken) IsExpired() bool {
	return time.Now().UTC().After(t.expiresAt)
}

// IsActive returns true if the token is active and not expired
func (t *PasswordResetToken) IsActive() bool {
	return t.status == PasswordResetTokenStatusActive && !t.IsExpired()
}

// VerifyToken verifies a provided token against the stored hash
func (t *PasswordResetToken) VerifyToken(token string) error {
	if !t.IsActive() {
		if t.status == PasswordResetTokenStatusUsed {
			return ErrTokenAlreadyUsed
		}
		if t.status == PasswordResetTokenStatusExpired || t.IsExpired() {
			return ErrTokenExpired
		}
		return ErrInvalidToken
	}

	hasher := sha256.New()
	hasher.Write([]byte(token))
	providedHash := hex.EncodeToString(hasher.Sum(nil))

	if providedHash != t.tokenHash {
		return ErrInvalidToken
	}

	return nil
}

// UseToken marks the token as used
func (t *PasswordResetToken) UseToken() error {
	if !t.IsActive() {
		if t.status == PasswordResetTokenStatusUsed {
			return ErrTokenAlreadyUsed
		}
		return ErrTokenExpired
	}

	now := time.Now().UTC()
	t.status = PasswordResetTokenStatusUsed
	t.usedAt = &now
	t.updatedAt = now
	t.version++
	return nil
}

// RevokeToken marks the token as revoked
func (t *PasswordResetToken) RevokeToken() {
	now := time.Now().UTC()
	t.status = PasswordResetTokenStatusRevoked
	t.updatedAt = now
	t.version++
}

// CreatedAt returns the creation time
func (t *PasswordResetToken) CreatedAt() time.Time {
	return t.createdAt
}

// UpdatedAt returns the last update time
func (t *PasswordResetToken) UpdatedAt() time.Time {
	return t.updatedAt
}

// MarkExpired marks the token as expired
func (t *PasswordResetToken) MarkExpired() {
	now := time.Now().UTC()
	t.status = PasswordResetTokenStatusExpired
	t.updatedAt = now
	t.version++
}

// MarkUsed marks the token as used with additional metadata
func (t *PasswordResetToken) MarkUsed(ipAddress, userAgent string, usedAt time.Time) {
	t.status = PasswordResetTokenStatusUsed
	t.usedAt = &usedAt
	t.ipAddress = ipAddress
	t.userAgent = userAgent
	t.updatedAt = usedAt
	t.version++
}