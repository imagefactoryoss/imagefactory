package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
	tests := []struct {
		name        string
		tenantID    string
		keyName     string
		scopes      []string
		expiresIn   *time.Duration
		expectError bool
		expectMsg   string
	}{
		{
			name:      "valid key generation",
			tenantID:  "tenant-123",
			keyName:   "Production API Key",
			scopes:    []string{"read:builds", "write:projects"},
			expiresIn: nil,
		},
		{
			name:      "with expiration",
			tenantID:  "tenant-123",
			keyName:   "Temporary Key",
			scopes:    []string{"read:*"},
			expiresIn: func() *time.Duration { d := 24 * time.Hour; return &d }(),
		},
		{
			name:        "empty tenant ID",
			tenantID:    "",
			keyName:     "Test Key",
			scopes:      []string{"read:*"},
			expectError: true,
			expectMsg:   "tenant ID is required",
		},
		{
			name:        "empty key name",
			tenantID:    "tenant-123",
			keyName:     "",
			scopes:      []string{"read:*"},
			expectError: true,
			expectMsg:   "API key name is required",
		},
		{
			name:        "empty scopes",
			tenantID:    "tenant-123",
			keyName:     "Test Key",
			scopes:      []string{},
			expectError: true,
			expectMsg:   "at least one scope is required",
		},
		{
			name:        "nil scopes",
			tenantID:    "tenant-123",
			keyName:     "Test Key",
			scopes:      nil,
			expectError: true,
			expectMsg:   "at least one scope is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey, unhashedKey, err := GenerateKey(tt.tenantID, tt.keyName, tt.scopes, tt.expiresIn)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectMsg)
				assert.Nil(t, apiKey)
				assert.Empty(t, unhashedKey)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, apiKey)
			assert.NotEmpty(t, unhashedKey)

			// Verify key format
			assert.True(t, len(unhashedKey) > len(APIKeyPrefix)+KeyLength)
			assert.Equal(t, APIKeyPrefix, unhashedKey[:len(APIKeyPrefix)])

			// Verify API key object
			assert.NotEmpty(t, apiKey.ID)
			assert.Equal(t, tt.tenantID, apiKey.TenantID)
			assert.Equal(t, tt.keyName, apiKey.Name)
			assert.Equal(t, tt.scopes, apiKey.Scopes)
			assert.NotNil(t, apiKey.CreatedAt)
			assert.Nil(t, apiKey.RevokedAt)
			assert.Nil(t, apiKey.LastUsedAt)

			// Verify expiration
			if tt.expiresIn != nil {
				assert.NotNil(t, apiKey.ExpiresAt)
				assert.True(t, apiKey.ExpiresAt.After(apiKey.CreatedAt))
			} else {
				assert.Nil(t, apiKey.ExpiresAt)
			}
		})
	}
}

func TestAPIKeyIsValid(t *testing.T) {
	tests := []struct {
		name      string
		setupKey  func() *APIKey
		expectValid bool
	}{
		{
			name: "fresh key is valid",
			setupKey: func() *APIKey {
				now := time.Now().UTC()
				return &APIKey{
					ID:        "key-1",
					TenantID:  "tenant-1",
					KeyHash:   "hash",
					Name:      "Test",
					Scopes:    []string{"read:*"},
					CreatedAt: now,
					ExpiresAt: nil,
					RevokedAt: nil,
				}
			},
			expectValid: true,
		},
		{
			name: "revoked key is invalid",
			setupKey: func() *APIKey {
				now := time.Now().UTC()
				return &APIKey{
					ID:        "key-1",
					TenantID:  "tenant-1",
					KeyHash:   "hash",
					Name:      "Test",
					Scopes:    []string{"read:*"},
					CreatedAt: now,
					ExpiresAt: nil,
					RevokedAt: &now,
				}
			},
			expectValid: false,
		},
		{
			name: "expired key is invalid",
			setupKey: func() *APIKey {
				now := time.Now().UTC()
				expired := now.Add(-1 * time.Hour)
				return &APIKey{
					ID:        "key-1",
					TenantID:  "tenant-1",
					KeyHash:   "hash",
					Name:      "Test",
					Scopes:    []string{"read:*"},
					CreatedAt: now,
					ExpiresAt: &expired,
					RevokedAt: nil,
				}
			},
			expectValid: false,
		},
		{
			name: "not yet expired key is valid",
			setupKey: func() *APIKey {
				now := time.Now().UTC()
				future := now.Add(24 * time.Hour)
				return &APIKey{
					ID:        "key-1",
					TenantID:  "tenant-1",
					KeyHash:   "hash",
					Name:      "Test",
					Scopes:    []string{"read:*"},
					CreatedAt: now,
					ExpiresAt: &future,
					RevokedAt: nil,
				}
			},
			expectValid: true,
		},
		{
			name: "nil key is invalid",
			setupKey: func() *APIKey {
				return nil
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := tt.setupKey()
			assert.Equal(t, tt.expectValid, apiKey.IsValid())
		})
	}
}

func TestAPIKeyIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		setupKey  func() *APIKey
		expectExpired bool
	}{
		{
			name: "key with no expiration is not expired",
			setupKey: func() *APIKey {
				return &APIKey{
					ID:        "key-1",
					ExpiresAt: nil,
				}
			},
			expectExpired: false,
		},
		{
			name: "key expiring in future is not expired",
			setupKey: func() *APIKey {
				future := time.Now().UTC().Add(24 * time.Hour)
				return &APIKey{
					ID:        "key-1",
					ExpiresAt: &future,
				}
			},
			expectExpired: false,
		},
		{
			name: "key expired in past is expired",
			setupKey: func() *APIKey {
				past := time.Now().UTC().Add(-1 * time.Hour)
				return &APIKey{
					ID:        "key-1",
					ExpiresAt: &past,
				}
			},
			expectExpired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := tt.setupKey()
			assert.Equal(t, tt.expectExpired, apiKey.IsExpired())
		})
	}
}

func TestAPIKeyIsRevoked(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name      string
		key       *APIKey
		expectRevoked bool
	}{
		{
			name:      "key without revocation is not revoked",
			key:       &APIKey{ID: "key-1", RevokedAt: nil},
			expectRevoked: false,
		},
		{
			name:      "key with revocation is revoked",
			key:       &APIKey{ID: "key-1", RevokedAt: &now},
			expectRevoked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectRevoked, tt.key.IsRevoked())
		})
	}
}

func TestAPIKeyHasScope(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name      string
		setupKey  func() *APIKey
		checkScope string
		expectHas bool
	}{
		{
			name: "valid key with matching scope",
			setupKey: func() *APIKey {
				return &APIKey{
					ID:        "key-1",
					Scopes:    []string{"read:builds", "write:projects"},
					CreatedAt: now,
					ExpiresAt: nil,
					RevokedAt: nil,
				}
			},
			checkScope: "read:builds",
			expectHas:  true,
		},
		{
			name: "valid key with wildcard scope",
			setupKey: func() *APIKey {
				return &APIKey{
					ID:        "key-1",
					Scopes:    []string{"*"},
					CreatedAt: now,
					ExpiresAt: nil,
					RevokedAt: nil,
				}
			},
			checkScope: "any:scope",
			expectHas:  true,
		},
		{
			name: "valid key without matching scope",
			setupKey: func() *APIKey {
				return &APIKey{
					ID:        "key-1",
					Scopes:    []string{"read:builds"},
					CreatedAt: now,
					ExpiresAt: nil,
					RevokedAt: nil,
				}
			},
			checkScope: "write:projects",
			expectHas:  false,
		},
		{
			name: "revoked key cannot have scope",
			setupKey: func() *APIKey {
				return &APIKey{
					ID:        "key-1",
					Scopes:    []string{"read:*"},
					CreatedAt: now,
					ExpiresAt: nil,
					RevokedAt: &now,
				}
			},
			checkScope: "read:*",
			expectHas:  false,
		},
		{
			name: "expired key cannot have scope",
			setupKey: func() *APIKey {
				past := now.Add(-1 * time.Hour)
				return &APIKey{
					ID:        "key-1",
					Scopes:    []string{"read:*"},
					CreatedAt: now,
					ExpiresAt: &past,
					RevokedAt: nil,
				}
			},
			checkScope: "read:*",
			expectHas:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := tt.setupKey()
			assert.Equal(t, tt.expectHas, apiKey.HasScope(tt.checkScope))
		})
	}
}

func TestAPIKeyUpdateLastUsed(t *testing.T) {
	apiKey := &APIKey{
		ID:         "key-1",
		LastUsedAt: nil,
	}

	assert.Nil(t, apiKey.LastUsedAt)

	apiKey.UpdateLastUsed()

	assert.NotNil(t, apiKey.LastUsedAt)
	assert.True(t, time.Now().UTC().Sub(*apiKey.LastUsedAt) < 1*time.Second)
}

func TestAPIKeyRevoke(t *testing.T) {
	apiKey := &APIKey{
		ID:        "key-1",
		RevokedAt: nil,
	}

	assert.Nil(t, apiKey.RevokedAt)

	apiKey.Revoke()

	assert.NotNil(t, apiKey.RevokedAt)
	assert.True(t, time.Now().UTC().Sub(*apiKey.RevokedAt) < 1*time.Second)
}

func TestAPIKeyClearSensitiveData(t *testing.T) {
	apiKey := &APIKey{
		ID:  "key-1",
		Key: "if_somesecretkey",
	}

	assert.Equal(t, "if_somesecretkey", apiKey.Key)

	apiKey.ClearSensitiveData()

	assert.Empty(t, apiKey.Key)
}
