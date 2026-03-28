package user

import (
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Domain errors
var (
	ErrUserNotFound            = errors.New("user not found")
	ErrUserExists              = errors.New("user already exists")
	ErrInvalidUserID           = errors.New("invalid user ID")
	ErrInvalidEmail            = errors.New("invalid email address")
	ErrInvalidPassword         = errors.New("invalid password")
	ErrEmailNotVerified        = errors.New("email not verified")
	ErrAccountLocked           = errors.New("account is locked")
	ErrAccountDisabled         = errors.New("account is disabled")
	ErrPasswordResetNotAllowed = errors.New("password reset not allowed for this account")
)

// Validation rules
const (
	minPasswordLength = 8
	maxPasswordLength = 128
)

// Email validation regex (simplified RFC 5322)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	if len(email) > 254 {
		return ErrInvalidEmail
	}
	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if password == "" {
		return ErrInvalidPassword
	}
	if len(password) < minPasswordLength {
		return ErrInvalidPassword
	}
	if len(password) > maxPasswordLength {
		return ErrInvalidPassword
	}
	return nil
}

// UserStatus represents the status of a user account
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusPending   UserStatus = "pending"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusDisabled  UserStatus = "disabled"
	UserStatusLocked    UserStatus = "locked"
)

// MFAType represents the type of multi-factor authentication
type MFAType string

const (
	MFATypeNone  MFAType = "none"
	MFATypeTOTP  MFAType = "totp"
	MFATypeSMS   MFAType = "sms"
	MFATypeEmail MFAType = "email"
)

// AuthMethod represents the authentication method used by the user
type AuthMethod string

const (
	AuthMethodCredentials AuthMethod = "credentials" // Local username/password
	AuthMethodLDAP        AuthMethod = "ldap"        // LDAP/Directory
	AuthMethodOIDC        AuthMethod = "oidc"        // OpenID Connect
	AuthMethodAPIKey      AuthMethod = "api_key"     // API Key authentication
)

// User represents the user aggregate root
type User struct {
	id                 uuid.UUID
	email              string
	passwordHash       string
	firstName          string
	lastName           string
	status             UserStatus
	emailVerified      bool
	emailVerifiedAt    *time.Time
	mfaEnabled         bool
	mfaType            MFAType
	mfaSecret          string
	failedLoginCount   int
	lastLoginAt        *time.Time
	passwordChangedAt  time.Time
	mustChangePassword bool
	lockedUntil        *time.Time
	authMethod         AuthMethod // How the user authenticates (credentials, ldap, oidc, etc.)
	createdAt          time.Time
	updatedAt          time.Time
	version            int
}

// NewUser creates a new user aggregate
func NewUser(email, firstName, lastName, password string) (*User, error) {
	// Note: users are assigned to tenants via RBAC (user_role_assignments or groups).
	// The User aggregate no longer stores a default tenant — tenancy is RBAC-driven.

	// Validate email
	if err := ValidateEmail(email); err != nil {
		return nil, err
	}

	if firstName == "" || lastName == "" {
		return nil, errors.New("first name and last name are required")
	}

	// Validate password
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}

	// Hash the password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	return &User{
		id:                 uuid.New(),
		email:              email,
		passwordHash:       string(passwordHash),
		firstName:          firstName,
		lastName:           lastName,
		status:             UserStatusPending,
		emailVerified:      false,
		mfaEnabled:         false,
		mfaType:            MFATypeNone,
		failedLoginCount:   0,
		passwordChangedAt:  now,
		mustChangePassword: false,
		authMethod:         AuthMethodCredentials, // Default to credentials authentication
		createdAt:          now,
		updatedAt:          now,
		version:            1,
	}, nil
}

// NewUserFromExisting creates a user from existing data (for repository reconstruction)
func NewUserFromExisting(
	id uuid.UUID,
	email, passwordHash, firstName, lastName string,
	status UserStatus,
	emailVerified bool,
	emailVerifiedAt *time.Time,
	mfaEnabled bool,
	mfaType MFAType,
	mfaSecret string,
	failedLoginCount int,
	lastLoginAt *time.Time,
	passwordChangedAt time.Time,
	mustChangePassword bool,
	lockedUntil *time.Time,
	authMethod AuthMethod,
	createdAt, updatedAt time.Time,
	version int,
) (*User, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	// Users do not store a default tenant in the aggregate; tenancy is RBAC-driven.
	if email == "" {
		return nil, ErrInvalidEmail
	}

	return &User{
		id:                 id,
		email:              email,
		passwordHash:       passwordHash,
		firstName:          firstName,
		lastName:           lastName,
		status:             status,
		emailVerified:      emailVerified,
		emailVerifiedAt:    emailVerifiedAt,
		mfaEnabled:         mfaEnabled,
		mfaType:            mfaType,
		mfaSecret:          mfaSecret,
		failedLoginCount:   failedLoginCount,
		lastLoginAt:        lastLoginAt,
		passwordChangedAt:  passwordChangedAt,
		mustChangePassword: mustChangePassword,
		lockedUntil:        lockedUntil,
		authMethod:         authMethod,
		createdAt:          createdAt,
		updatedAt:          updatedAt,
		version:            version,
	}, nil
}

// ID returns the user ID
func (u *User) ID() uuid.UUID {
	return u.id
}

// Email returns the user's email
func (u *User) Email() string {
	return u.email
}

// FirstName returns the user's first name
func (u *User) FirstName() string {
	return u.firstName
}

// LastName returns the user's last name
func (u *User) LastName() string {
	return u.lastName
}

// FullName returns the user's full name
func (u *User) FullName() string {
	return u.firstName + " " + u.lastName
}

// Status returns the user status
func (u *User) Status() UserStatus {
	return u.status
}

// IsActive returns true if the user account is active
func (u *User) IsActive() bool {
	return u.status == UserStatusActive
}

// AuthMethod returns the user's authentication method
func (u *User) AuthMethod() AuthMethod {
	return u.authMethod
}

// SetAuthMethod sets the user's authentication method
func (u *User) SetAuthMethod(method AuthMethod) {
	u.authMethod = method
	u.updatedAt = time.Now().UTC()
	u.version++
}

// ClearPasswordHash clears locally stored password credentials.
// This should be used for externally authenticated users (LDAP/OIDC).
func (u *User) ClearPasswordHash() {
	u.passwordHash = ""
	u.mustChangePassword = false
	u.updatedAt = time.Now().UTC()
	u.version++
}

// EnsureLDAPCredentials normalizes credentials for LDAP-authenticated users.
// Returns true when state changed and a persistence update is needed.
func (u *User) EnsureLDAPCredentials() bool {
	changed := false
	if u.authMethod != AuthMethodLDAP {
		u.authMethod = AuthMethodLDAP
		changed = true
	}
	if u.passwordHash != "" || u.mustChangePassword {
		u.passwordHash = ""
		u.mustChangePassword = false
		changed = true
	}
	if changed {
		u.updatedAt = time.Now().UTC()
		u.version++
	}
	return changed
}

// IsEmailVerified returns true if the user's email is verified
func (u *User) IsEmailVerified() bool {
	return u.emailVerified
}

// VerifyEmail marks the user's email as verified
func (u *User) VerifyEmail() {
	now := time.Now().UTC()
	u.emailVerified = true
	u.emailVerifiedAt = &now
	u.updatedAt = now
	u.version++
}

// ChangePassword changes the user's password
func (u *User) ChangePassword(newPassword string) error {
	// Validate password
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.passwordHash = string(passwordHash)
	u.passwordChangedAt = time.Now().UTC()
	u.mustChangePassword = false
	u.updatedAt = time.Now().UTC()
	u.version++
	return nil
}

// VerifyPassword verifies a password against the stored hash
func (u *User) VerifyPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.passwordHash), []byte(password))
	return err == nil
}

// RecordLogin records a successful login
func (u *User) RecordLogin() {
	now := time.Now().UTC()
	u.lastLoginAt = &now
	u.failedLoginCount = 0
	u.updatedAt = now
	u.version++
}

// RecordFailedLogin records a failed login attempt
func (u *User) RecordFailedLogin() {
	u.failedLoginCount++
	u.updatedAt = time.Now().UTC()

	// Lock account after 5 failed attempts
	if u.failedLoginCount >= 5 {
		u.status = UserStatusLocked
		u.lockedUntil = &time.Time{}
		*u.lockedUntil = time.Now().UTC().Add(30 * time.Minute) // Lock for 30 minutes
	}

	u.version++
}

// UpdateFirstName updates the user's first name
func (u *User) UpdateFirstName(firstName string) error {
	if firstName == "" {
		return errors.New("first name cannot be empty")
	}
	u.firstName = firstName
	u.updatedAt = time.Now().UTC()
	u.version++
	return nil
}

// UpdateLastName updates the user's last name
func (u *User) UpdateLastName(lastName string) error {
	if lastName == "" {
		return errors.New("last name cannot be empty")
	}
	u.lastName = lastName
	u.updatedAt = time.Now().UTC()
	u.version++
	return nil
}

// UpdateStatus updates the user's status
func (u *User) UpdateStatus(status UserStatus) error {
	validStatuses := map[UserStatus]bool{
		UserStatusActive:    true,
		UserStatusPending:   true,
		UserStatusSuspended: true,
		UserStatusDisabled:  true,
		UserStatusLocked:    true,
	}
	if !validStatuses[status] {
		return errors.New("invalid user status")
	}
	u.status = status
	u.updatedAt = time.Now().UTC()
	u.version++
	return nil
}

// SetPasswordHash updates the password hash and records the change time
func (u *User) SetPasswordHash(hash string) error {
	if hash == "" {
		return errors.New("password hash cannot be empty")
	}
	u.passwordHash = hash
	u.passwordChangedAt = time.Now().UTC()
	u.updatedAt = time.Now().UTC()
	u.version++
	return nil
}

// VerifyEmailAddress marks the email as verified
func (u *User) VerifyEmailAddress() {
	now := time.Now().UTC()
	u.emailVerified = true
	u.emailVerifiedAt = &now
	u.updatedAt = now
	u.version++
}

// SuspendAccount suspends the account
func (u *User) SuspendAccount() {
	u.status = UserStatusSuspended
	u.updatedAt = time.Now().UTC()
	u.version++
}

// ReactivateAccount reactivates a suspended or locked account
func (u *User) ReactivateAccount() error {
	if u.status == UserStatusPending || u.status == UserStatusDisabled {
		return errors.New("cannot reactivate account with status: " + string(u.status))
	}
	u.status = UserStatusActive
	u.failedLoginCount = 0
	u.lockedUntil = nil
	u.updatedAt = time.Now().UTC()
	u.version++
	return nil
}

// IsLocked returns true if the account is currently locked
func (u *User) IsLocked() bool {
	if u.status == UserStatusLocked && u.lockedUntil != nil {
		return time.Now().UTC().Before(*u.lockedUntil)
	}
	return false
}

// UnlockAccount unlocks a locked account
func (u *User) UnlockAccount() {
	u.status = UserStatusActive
	u.failedLoginCount = 0
	u.lockedUntil = nil
	u.updatedAt = time.Now().UTC()
	u.version++
}

// Deactivate deactivates the user account (admin action)
func (u *User) Deactivate() {
	u.status = UserStatusDisabled
	u.updatedAt = time.Now().UTC()
	u.version++
}

// Activate activates a deactivated user account
func (u *User) Activate() {
	u.status = UserStatusActive
	u.failedLoginCount = 0
	u.lockedUntil = nil
	u.updatedAt = time.Now().UTC()
	u.version++
}

// EnableMFA enables multi-factor authentication
func (u *User) EnableMFA(mfaType MFAType, secret string) {
	u.mfaEnabled = true
	u.mfaType = mfaType
	u.mfaSecret = secret
	u.updatedAt = time.Now().UTC()
	u.version++
}

// DisableMFA disables multi-factor authentication
func (u *User) DisableMFA() {
	u.mfaEnabled = false
	u.mfaType = MFATypeNone
	u.mfaSecret = ""
	u.updatedAt = time.Now().UTC()
	u.version++
}

// IsMFAEnabled returns true if MFA is enabled
func (u *User) IsMFAEnabled() bool {
	return u.mfaEnabled
}

// MFAType returns the MFA type
func (u *User) MFAType() MFAType {
	return u.mfaType
}

// MFASecret returns the MFA secret
func (u *User) MFASecret() string {
	return u.mfaSecret
}

// CreatedAt returns the creation timestamp
func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

// UpdatedAt returns the last update timestamp
func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}

// PasswordHash returns the password hash (for repository use only)
func (u *User) PasswordHash() string {
	return u.passwordHash
}

// EmailVerifiedAt returns the email verification timestamp
func (u *User) EmailVerifiedAt() *time.Time {
	return u.emailVerifiedAt
}

// FailedLoginCount returns the failed login count
func (u *User) FailedLoginCount() int {
	return u.failedLoginCount
}

// LastLoginAt returns the last login timestamp
func (u *User) LastLoginAt() *time.Time {
	return u.lastLoginAt
}

// PasswordChangedAt returns the password change timestamp
func (u *User) PasswordChangedAt() time.Time {
	return u.passwordChangedAt
}

// MustChangePassword indicates whether user must rotate password before continuing.
func (u *User) MustChangePassword() bool {
	return u.mustChangePassword
}

// MarkPasswordChangeRequired sets mandatory password rotation flag.
func (u *User) MarkPasswordChangeRequired() {
	u.mustChangePassword = true
	u.updatedAt = time.Now().UTC()
	u.version++
}

// LockedUntil returns the account lock timestamp
func (u *User) LockedUntil() *time.Time {
	return u.lockedUntil
}

// Version returns the aggregate version for optimistic concurrency
func (u *User) Version() int {
	return u.version
}
