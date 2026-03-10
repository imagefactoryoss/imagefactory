package auth

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// APIKeyService handles API key operations
type APIKeyService struct {
	repository APIKeyRepository
	logger     *zap.Logger
}

// APIKeyRepository defines persistence operations for API keys
type APIKeyRepository interface {
	Create(apiKey *APIKey) error
	GetByKeyHash(keyHash string) (*APIKey, error)
	GetByID(id string) (*APIKey, error)
	GetByTenantID(tenantID string) ([]*APIKey, error)
	Update(apiKey *APIKey) error
	Delete(id string) error
	ListValid(tenantID string) ([]*APIKey, error)
}

// NewAPIKeyService creates a new API key service
func NewAPIKeyService(repository APIKeyRepository, logger *zap.Logger) *APIKeyService {
	return &APIKeyService{
		repository: repository,
		logger:     logger,
	}
}

// CreateAPIKey generates and stores a new API key
// Returns the key object and the unhashed key string (only shown once)
func (s *APIKeyService) CreateAPIKey(tenantID, name string, scopes []string, expiresIn *time.Duration) (*APIKey, string, error) {
	// Generate the key
	apiKey, unhashedKey, err := GenerateKey(tenantID, name, scopes, expiresIn)
	if err != nil {
		s.logger.Error("failed to generate API key", zap.Error(err))
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Hash the key for storage
	hashedKey, err := s.hashKey(unhashedKey)
	if err != nil {
		s.logger.Error("failed to hash API key", zap.Error(err))
		return nil, "", fmt.Errorf("failed to hash API key: %w", err)
	}
	apiKey.KeyHash = hashedKey

	// Store in database
	if err := s.repository.Create(apiKey); err != nil {
		s.logger.Error("failed to store API key", zap.String("tenantID", tenantID), zap.Error(err))
		return nil, "", fmt.Errorf("failed to store API key: %w", err)
	}

	s.logger.Info("API key created successfully", zap.String("tenantID", tenantID), zap.String("keyID", apiKey.ID))

	// Clear sensitive data before returning
	apiKey.ClearSensitiveData()

	return apiKey, unhashedKey, nil
}

// ValidateAPIKey validates an API key and checks its validity
// Returns the API key details if valid, nil if invalid
func (s *APIKeyService) ValidateAPIKey(unhashedKey string) (*APIKey, error) {
	if unhashedKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Hash the provided key to look it up
	hashedKey, err := s.hashKey(unhashedKey)
	if err != nil {
		s.logger.Debug("failed to hash provided API key", zap.Error(err))
		return nil, fmt.Errorf("invalid API key format")
	}

	// Look up key in database
	apiKey, err := s.repository.GetByKeyHash(hashedKey)
	if err != nil {
		s.logger.Debug("API key not found", zap.Error(err))
		return nil, fmt.Errorf("API key not found")
	}

	if apiKey == nil {
		return nil, fmt.Errorf("API key not found")
	}

	// Check validity
	if !apiKey.IsValid() {
		if apiKey.IsRevoked() {
			s.logger.Warn("attempt to use revoked API key", zap.String("keyID", apiKey.ID))
			return nil, fmt.Errorf("API key has been revoked")
		}
		if apiKey.IsExpired() {
			s.logger.Warn("attempt to use expired API key", zap.String("keyID", apiKey.ID))
			return nil, fmt.Errorf("API key has expired")
		}
		return nil, fmt.Errorf("API key is invalid")
	}

	// Update last used timestamp
	apiKey.UpdateLastUsed()
	if err := s.repository.Update(apiKey); err != nil {
		s.logger.Error("failed to update last used timestamp", zap.String("keyID", apiKey.ID), zap.Error(err))
		// Don't fail the validation, just log the error
	}

	return apiKey, nil
}

// GetAPIKey retrieves an API key by ID
func (s *APIKeyService) GetAPIKey(id string) (*APIKey, error) {
	apiKey, err := s.repository.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve API key: %w", err)
	}
	if apiKey == nil {
		return nil, fmt.Errorf("API key not found")
	}
	return apiKey, nil
}

// ListAPIKeys lists all valid API keys for a tenant
func (s *APIKeyService) ListAPIKeys(tenantID string) ([]*APIKey, error) {
	apiKeys, err := s.repository.ListValid(tenantID)
	if err != nil {
		s.logger.Error("failed to list API keys", zap.String("tenantID", tenantID), zap.Error(err))
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	return apiKeys, nil
}

// RevokeAPIKey revokes an API key by ID
func (s *APIKeyService) RevokeAPIKey(id string) error {
	apiKey, err := s.repository.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to retrieve API key: %w", err)
	}
	if apiKey == nil {
		return fmt.Errorf("API key not found")
	}

	apiKey.Revoke()
	if err := s.repository.Update(apiKey); err != nil {
		s.logger.Error("failed to revoke API key", zap.String("keyID", id), zap.Error(err))
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	s.logger.Info("API key revoked", zap.String("keyID", id))
	return nil
}

// DeleteAPIKey deletes an API key by ID (hard delete, different from revocation)
func (s *APIKeyService) DeleteAPIKey(id string) error {
	if err := s.repository.Delete(id); err != nil {
		s.logger.Error("failed to delete API key", zap.String("keyID", id), zap.Error(err))
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	s.logger.Info("API key deleted", zap.String("keyID", id))
	return nil
}

// hashKey hashes an API key using bcrypt for secure storage
func (s *APIKeyService) hashKey(key string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash key: %w", err)
	}
	return string(hash), nil
}

// CompareKey compares an unhashed key with a stored hash
func (s *APIKeyService) CompareKey(key, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(key))
	return err == nil
}
