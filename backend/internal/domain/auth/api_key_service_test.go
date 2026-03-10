package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockAPIKeyRepository is a mock implementation of APIKeyRepository
type MockAPIKeyRepository struct {
	mock.Mock
}

func (m *MockAPIKeyRepository) Create(apiKey *APIKey) error {
	args := m.Called(apiKey)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) GetByKeyHash(keyHash string) (*APIKey, error) {
	args := m.Called(keyHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) GetByID(id string) (*APIKey, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) GetByTenantID(tenantID string) ([]*APIKey, error) {
	args := m.Called(tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) Update(apiKey *APIKey) error {
	args := m.Called(apiKey)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) ListValid(tenantID string) ([]*APIKey, error) {
	args := m.Called(tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*APIKey), args.Error(1)
}

func TestCreateAPIKey(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := new(MockAPIKeyRepository)

	tests := []struct {
		name        string
		tenantID    string
		keyName     string
		scopes      []string
		expiresIn   *time.Duration
		setupMock   func()
		expectError bool
	}{
		{
			name:      "successful creation",
			tenantID:  "tenant-1",
			keyName:   "Prod API Key",
			scopes:    []string{"read:*", "write:builds"},
			expiresIn: nil,
			setupMock: func() {
				mockRepo.On("Create", mock.MatchedBy(func(ak *APIKey) bool {
					return ak.TenantID == "tenant-1" && ak.Name == "Prod API Key"
				})).Return(nil)
			},
			expectError: false,
		},
		{
			name:      "with expiration",
			tenantID:  "tenant-2",
			keyName:   "Temp Key",
			scopes:    []string{"read:builds"},
			expiresIn: func() *time.Duration { d := 7 * 24 * time.Hour; return &d }(),
			setupMock: func() {
				mockRepo.On("Create", mock.MatchedBy(func(ak *APIKey) bool {
					return ak.TenantID == "tenant-2" && ak.ExpiresAt != nil
				})).Return(nil)
			},
			expectError: false,
		},
		{
			name:        "empty tenant ID",
			tenantID:    "",
			keyName:     "Test",
			scopes:      []string{"read:*"},
			setupMock:   func() {},
			expectError: true,
		},
		{
			name:        "database error",
			tenantID:    "tenant-1",
			keyName:     "Test",
			scopes:      []string{"read:*"},
			setupMock: func() {
				mockRepo.On("Create", mock.Anything).Return(errors.New("db connection failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			defer func() { mockRepo.AssertExpectations(t) }()

			service := NewAPIKeyService(mockRepo, logger)
			apiKey, unhashedKey, err := service.CreateAPIKey(tt.tenantID, tt.keyName, tt.scopes, tt.expiresIn)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, apiKey)
				assert.Empty(t, unhashedKey)
			} else {
				require.NoError(t, err)
				require.NotNil(t, apiKey)
				assert.NotEmpty(t, unhashedKey)
				assert.True(t, len(unhashedKey) > len(APIKeyPrefix))
				assert.Equal(t, APIKeyPrefix, unhashedKey[:len(APIKeyPrefix)])
				assert.Empty(t, apiKey.Key) // Sensitive data cleared
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestValidateAPIKey(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := new(MockAPIKeyRepository)
	now := time.Now().UTC()

	tests := []struct {
		name        string
		keyInput    string
		setupMock   func()
		expectError bool
		expectValid bool
	}{
		{
			name:     "valid key",
			keyInput: "if_testkeyvalidation",
			setupMock: func() {
				mockRepo.On("GetByKeyHash", mock.Anything).Return(&APIKey{
					ID:        "key-1",
					TenantID:  "tenant-1",
					Name:      "Test",
					Scopes:    []string{"read:*"},
					CreatedAt: now,
					ExpiresAt: nil,
					RevokedAt: nil,
				}, nil).Once()
				mockRepo.On("Update", mock.Anything).Return(nil).Once()
			},
			expectError: false,
			expectValid: true,
		},
		{
			name:        "empty key",
			keyInput:    "",
			setupMock:   func() {},
			expectError: true,
			expectValid: false,
		},
		{
			name:     "revoked key",
			keyInput: "if_revokedkey",
			setupMock: func() {
				revokedAt := now.Add(-1 * time.Hour)
				mockRepo.On("GetByKeyHash", mock.Anything).Return(&APIKey{
					ID:        "key-2",
					TenantID:  "tenant-1",
					Name:      "Old",
					Scopes:    []string{"read:*"},
					CreatedAt: now,
					ExpiresAt: nil,
					RevokedAt: &revokedAt,
				}, nil)
			},
			expectError: true,
			expectValid: false,
		},
		{
			name:     "expired key",
			keyInput: "if_expiredkey",
			setupMock: func() {
				pastTime := now.Add(-1 * time.Hour)
				mockRepo.On("GetByKeyHash", mock.Anything).Return(&APIKey{
					ID:        "key-3",
					TenantID:  "tenant-1",
					Name:      "Old",
					Scopes:    []string{"read:*"},
					CreatedAt: now,
					ExpiresAt: &pastTime,
					RevokedAt: nil,
				}, nil)
			},
			expectError: true,
			expectValid: false,
		},
		{
			name:     "key not found",
			keyInput: "if_notfound",
			setupMock: func() {
				mockRepo.On("GetByKeyHash", mock.Anything).Return(nil, errors.New("not found"))
			},
			expectError: true,
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			service := NewAPIKeyService(mockRepo, logger)
			apiKey, err := service.ValidateAPIKey(tt.keyInput)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, apiKey)
			} else {
				require.NoError(t, err)
				require.NotNil(t, apiKey)
				assert.Equal(t, "tenant-1", apiKey.TenantID)
				assert.NotNil(t, apiKey.LastUsedAt) // Should be updated
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRevokeAPIKey(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := new(MockAPIKeyRepository)

	t.Run("successful revocation", func(t *testing.T) {
		mockRepo.On("GetByID", "key-1").Return(&APIKey{
			ID:        "key-1",
			TenantID:  "tenant-1",
			RevokedAt: nil,
		}, nil).Once()
		mockRepo.On("Update", mock.Anything).Return(nil).Once()

		service := NewAPIKeyService(mockRepo, logger)
		err := service.RevokeAPIKey("key-1")

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("key not found", func(t *testing.T) {
		mockRepo.On("GetByID", "nonexistent").Return(nil, errors.New("not found"))

		service := NewAPIKeyService(mockRepo, logger)
		err := service.RevokeAPIKey("nonexistent")

		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestDeleteAPIKey(t *testing.T) {
	logger := zap.NewNop()
	mockRepo := new(MockAPIKeyRepository)

	t.Run("successful deletion", func(t *testing.T) {
		mockRepo.ExpectedCalls = nil // Reset expectations
		mockRepo.On("Delete", "key-1").Return(nil)

		service := NewAPIKeyService(mockRepo, logger)
		err := service.DeleteAPIKey("key-1")

		assert.NoError(t, err)
	})

	t.Run("deletion error", func(t *testing.T) {
		mockRepo = new(MockAPIKeyRepository) // Fresh mock
		mockRepo.On("Delete", "key-1").Return(errors.New("db error"))

		service := NewAPIKeyService(mockRepo, logger)
		err := service.DeleteAPIKey("key-1")

		assert.Error(t, err)
	})
}

func TestListAPIKeys(t *testing.T) {
	logger := zap.NewNop()

	t.Run("list valid keys", func(t *testing.T) {
		mockRepo := new(MockAPIKeyRepository)
		keys := []*APIKey{
			{ID: "key-1", TenantID: "tenant-1", Name: "Key 1"},
			{ID: "key-2", TenantID: "tenant-1", Name: "Key 2"},
		}
		mockRepo.On("ListValid", "tenant-1").Return(keys, nil)

		service := NewAPIKeyService(mockRepo, logger)
		result, err := service.ListAPIKeys("tenant-1")

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "key-1", result[0].ID)
		assert.Equal(t, "key-2", result[1].ID)
	})

	t.Run("list keys error", func(t *testing.T) {
		mockRepo := new(MockAPIKeyRepository)
		mockRepo.On("ListValid", "tenant-1").Return(nil, errors.New("db error"))

		service := NewAPIKeyService(mockRepo, logger)
		result, err := service.ListAPIKeys("tenant-1")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
