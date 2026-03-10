package user

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		firstName   string
		lastName    string
		password    string
		expectError bool
		errorType   error
	}{
		{
			name:      "success",
			email:     "test@example.com",
			firstName: "John",
			lastName:  "Doe",
			password:  "password123",
		},
		{
			name:        "empty email",
			email:       "",
			firstName:   "John",
			lastName:    "Doe",
			password:    "password123",
			expectError: true,
			errorType:   ErrInvalidEmail,
		},
		{
			name:        "empty first name",
			email:       "test@example.com",
			firstName:   "",
			lastName:    "Doe",
			password:    "password123",
			expectError: true,
		},
		{
			name:        "empty last name",
			email:       "test@example.com",
			firstName:   "John",
			lastName:    "",
			password:    "password123",
			expectError: true,
		},
		{
			name:        "empty password",
			email:       "test@example.com",
			firstName:   "John",
			lastName:    "Doe",
			password:    "",
			expectError: true,
			errorType:   ErrInvalidPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := NewUser(tt.email, tt.firstName, tt.lastName, tt.password)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.email, user.Email())
				assert.Equal(t, tt.firstName, user.FirstName())
				assert.Equal(t, tt.lastName, user.LastName())
				assert.Equal(t, UserStatusPending, user.Status())
				assert.False(t, user.IsEmailVerified())
				assert.Equal(t, 1, user.Version())
			}
		})
	}
}

func TestNewUserFromExisting(t *testing.T) {
	tests := []struct {
		name        string
		id          uuid.UUID
		email       string
		expectError bool
		errorType   error
	}{
		{
			name:  "success",
			id:    uuid.New(),
			email: "test@example.com",
		},
		{
			name:        "empty ID",
			id:          uuid.Nil,
			email:       "test@example.com",
			expectError: true,
			errorType:   ErrInvalidUserID,
		},
		{
			name:        "empty email",
			id:          uuid.New(),
			email:       "",
			expectError: true,
			errorType:   ErrInvalidEmail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := NewUserFromExisting(
				tt.id,
				tt.email,
				"hashedpassword",
				"John",
				"Doe",
				UserStatusActive,
				true,
				nil,
				false,
				MFATypeNone,
				"",
				0,
				nil,
				time.Now(),
				false,
				nil,
				AuthMethodCredentials,
				time.Now(),
				time.Now(),
				1,
			)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.id, user.ID())
				assert.Equal(t, tt.email, user.Email())
			}
		})
	}
}

func TestUser_VerifyPassword(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	assert.True(t, user.VerifyPassword("password123"))
	assert.False(t, user.VerifyPassword("wrongpassword"))
	assert.False(t, user.VerifyPassword(""))
}

func TestUser_ChangePassword(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	initialVersion := user.Version()
	initialChangedAt := user.PasswordChangedAt()

	err = user.ChangePassword("newpassword456")
	assert.NoError(t, err)
	assert.True(t, user.VerifyPassword("newpassword456"))
	assert.False(t, user.VerifyPassword("password123"))
	assert.Equal(t, initialVersion+1, user.Version())
	assert.True(t, user.PasswordChangedAt().After(initialChangedAt))
}

func TestUser_RecordLogin(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	initialVersion := user.Version()

	user.RecordLogin()

	assert.Equal(t, initialVersion+1, user.Version())
	assert.Equal(t, 0, user.FailedLoginCount())
	assert.NotNil(t, user.LastLoginAt())
}

func TestUser_RecordFailedLogin(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	// First failed login
	user.RecordFailedLogin()
	assert.Equal(t, 1, user.FailedLoginCount())
	assert.Equal(t, UserStatusPending, user.Status())

	// Second failed login
	user.RecordFailedLogin()
	assert.Equal(t, 2, user.FailedLoginCount())

	// Third failed login
	user.RecordFailedLogin()
	assert.Equal(t, 3, user.FailedLoginCount())

	// Fourth failed login
	user.RecordFailedLogin()
	assert.Equal(t, 4, user.FailedLoginCount())

	// Fifth failed login - should lock account
	user.RecordFailedLogin()
	assert.Equal(t, 5, user.FailedLoginCount())
	assert.Equal(t, UserStatusLocked, user.Status())
	assert.NotNil(t, user.LockedUntil())
}

func TestUser_IsLocked(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	// Initially not locked
	assert.False(t, user.IsLocked())

	// Simulate 5 failed logins to lock account
	for i := 0; i < 5; i++ {
		user.RecordFailedLogin()
	}

	assert.True(t, user.IsLocked())
	assert.Equal(t, UserStatusLocked, user.Status())
}

func TestUser_UnlockAccount(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	// Lock the account
	for i := 0; i < 5; i++ {
		user.RecordFailedLogin()
	}
	assert.True(t, user.IsLocked())

	// Unlock the account
	user.UnlockAccount()
	assert.False(t, user.IsLocked())
	assert.Equal(t, UserStatusActive, user.Status())
	assert.Equal(t, 0, user.FailedLoginCount())
	assert.Nil(t, user.LockedUntil())
}

func TestUser_VerifyEmail(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	assert.False(t, user.IsEmailVerified())
	assert.Nil(t, user.EmailVerifiedAt())

	user.VerifyEmail()

	assert.True(t, user.IsEmailVerified())
	assert.NotNil(t, user.EmailVerifiedAt())
}

func TestUser_EnableMFA(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	assert.False(t, user.IsMFAEnabled())
	assert.Equal(t, MFATypeNone, user.MFAType())

	user.EnableMFA(MFATypeTOTP, "secret123")

	assert.True(t, user.IsMFAEnabled())
	assert.Equal(t, MFATypeTOTP, user.MFAType())
	assert.Equal(t, "secret123", user.MFASecret())
}

func TestUser_DisableMFA(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	user.EnableMFA(MFATypeTOTP, "secret123")
	assert.True(t, user.IsMFAEnabled())

	user.DisableMFA()

	assert.False(t, user.IsMFAEnabled())
	assert.Equal(t, MFATypeNone, user.MFAType())
	assert.Equal(t, "", user.MFASecret())
}

func TestUser_IsActive(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	assert.False(t, user.IsActive()) // Starts as pending

	// Manually set to active for testing
	user, err = NewUserFromExisting(
		uuid.New(),
		"test@example.com",
		"hashedpassword",
		"John",
		"Doe",
		UserStatusActive,
		false,
		nil,
		false,
		MFATypeNone,
		"",
		0,
		nil,
		time.Now(),
		false,
		nil,
		AuthMethodCredentials,
		time.Now(),
		time.Now(),
		1,
	)
	require.NoError(t, err)

	assert.True(t, user.IsActive())
}

func TestUser_FullName(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe", "password123")
	require.NoError(t, err)

	assert.Equal(t, "John Doe", user.FullName())
}
