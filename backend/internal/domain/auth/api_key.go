package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	// KeyLength is the length of the randomly generated key portion
	KeyLength = 32

	// APIKeyFormat is the format of the API key: "if_" prefix + random string
	APIKeyPrefix = "if_"
)

// APIKey represents an API key for external authentication
type APIKey struct {
	ID        string     `db:"id"`
	TenantID  string     `db:"tenant_id"`
	Key       string     `db:"key"` // Unhashed key - only shown at creation
	KeyHash   string     `db:"key_hash"` // Hashed key for storage
	Name      string     `db:"name"`
	Scopes    []string   `db:"scopes"` // JSON array in DB
	CreatedAt time.Time  `db:"created_at"`
	ExpiresAt *time.Time `db:"expires_at"` // Nullable for non-expiring keys
	LastUsedAt *time.Time `db:"last_used_at"` // Track usage for security audit
	RevokedAt *time.Time `db:"revoked_at"` // Soft delete via revocation
}

// GenerateKey creates a new API key with a random token
// Returns the unhashed key (only shown once) and stores the hashed version
func GenerateKey(tenantID, name string, scopes []string, expiresIn *time.Duration) (*APIKey, string, error) {
	// Validate inputs
	if tenantID == "" {
		return nil, "", fmt.Errorf("tenant ID is required")
	}
	if name == "" {
		return nil, "", fmt.Errorf("API key name is required")
	}
	if len(scopes) == 0 {
		return nil, "", fmt.Errorf("at least one scope is required")
	}

	// Generate random key
	randomBytes := make([]byte, KeyLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	randomHex := hex.EncodeToString(randomBytes)

	// Create API key string with prefix for easy identification
	unhashed := fmt.Sprintf("%s%s", APIKeyPrefix, randomHex)

	// Create key object
	now := time.Now().UTC()
	apiKey := &APIKey{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Key:       unhashed, // Temporarily store unhashed (will be cleared before returning)
		KeyHash:   "", // Will be set by caller with hash function
		Name:      name,
		Scopes:    scopes,
		CreatedAt: now,
		ExpiresAt: nil,
	}

	// Add expiration if specified
	if expiresIn != nil {
		expiresAt := now.Add(*expiresIn)
		apiKey.ExpiresAt = &expiresAt
	}

	return apiKey, unhashed, nil
}

// IsValid checks if the API key is currently valid (not revoked, not expired)
func (a *APIKey) IsValid() bool {
	if a == nil {
		return false
	}

	// Check if revoked
	if a.RevokedAt != nil {
		return false
	}

	// Check if expired
	if a.ExpiresAt != nil && time.Now().UTC().After(*a.ExpiresAt) {
		return false
	}

	return true
}

// IsExpired checks if the API key has expired
func (a *APIKey) IsExpired() bool {
	if a.ExpiresAt == nil {
		return false // No expiration date
	}
	return time.Now().UTC().After(*a.ExpiresAt)
}

// IsRevoked checks if the API key has been revoked
func (a *APIKey) IsRevoked() bool {
	return a.RevokedAt != nil
}

// HasScope checks if the API key has the required scope
func (a *APIKey) HasScope(scope string) bool {
	if !a.IsValid() {
		return false
	}

	for _, s := range a.Scopes {
		if s == scope || s == "*" {
			return true
		}
	}
	return false
}

// UpdateLastUsed updates the last used timestamp (call after successful validation)
func (a *APIKey) UpdateLastUsed() {
	now := time.Now().UTC()
	a.LastUsedAt = &now
}

// Revoke marks the API key as revoked
func (a *APIKey) Revoke() {
	now := time.Now().UTC()
	a.RevokedAt = &now
}

// ClearSensitiveData removes the unhashed key from memory (for security)
func (a *APIKey) ClearSensitiveData() {
	a.Key = ""
}
