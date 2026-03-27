package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/infrastructure/ldap"
)

func TestLDAPServiceCreateUserFromLDAP_DoesNotPersistPasswordHash(t *testing.T) {
	repo := NewMockRepository()
	service := NewLDAPService(repo, zap.NewNop(), "test-secret", nil, nil)

	ldapUser := &ldap.UserInfo{
		Username:  "alice",
		FirstName: "Alice",
		LastName:  "Admin",
		Email:     "alice@example.com",
	}

	createdUser, err := service.createUserFromLDAP(context.Background(), "alice@example.com", ldapUser)
	require.NoError(t, err)
	require.NotNil(t, createdUser)

	assert.Equal(t, AuthMethodLDAP, createdUser.AuthMethod())
	assert.Empty(t, createdUser.PasswordHash())
	assert.False(t, createdUser.VerifyPassword("SomeKnownPassword123"))

	savedUser, err := repo.FindByEmail(context.Background(), "alice@example.com")
	require.NoError(t, err)
	assert.Empty(t, savedUser.PasswordHash())
	assert.Equal(t, AuthMethodLDAP, savedUser.AuthMethod())
}
