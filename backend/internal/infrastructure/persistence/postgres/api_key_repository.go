package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/auth"
)

// APIKeyRepository implements auth.APIKeyRepository for PostgreSQL
type APIKeyRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(db *sql.DB, logger *zap.Logger) *APIKeyRepository {
	return &APIKeyRepository{
		db:     db,
		logger: logger,
	}
}

// Create stores a new API key in the database
func (r *APIKeyRepository) Create(apiKey *auth.APIKey) error {
	query := `
		INSERT INTO api_keys (id, tenant_id, key_hash, name, scopes, created_at, expires_at, last_used_at, revoked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(
		query,
		apiKey.ID,
		apiKey.TenantID,
		apiKey.KeyHash,
		apiKey.Name,
		pq.Array(apiKey.Scopes),
		apiKey.CreatedAt,
		apiKey.ExpiresAt,
		apiKey.LastUsedAt,
		apiKey.RevokedAt,
	)

	if err != nil {
		r.logger.Error("failed to create API key", zap.Error(err))
		return fmt.Errorf("failed to create API key: %w", err)
	}

	return nil
}

// GetByKeyHash retrieves an API key by its hashed key
func (r *APIKeyRepository) GetByKeyHash(keyHash string) (*auth.APIKey, error) {
	query := `
		SELECT id, tenant_id, key_hash, name, scopes, created_at, expires_at, last_used_at, revoked_at
		FROM api_keys
		WHERE key_hash = $1
	`

	apiKey := &auth.APIKey{}
	var scopes pq.StringArray

	err := r.db.QueryRow(query, keyHash).Scan(
		&apiKey.ID,
		&apiKey.TenantID,
		&apiKey.KeyHash,
		&apiKey.Name,
		&scopes,
		&apiKey.CreatedAt,
		&apiKey.ExpiresAt,
		&apiKey.LastUsedAt,
		&apiKey.RevokedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Key not found
		}
		r.logger.Error("failed to get API key by hash", zap.Error(err))
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	apiKey.Scopes = scopes
	return apiKey, nil
}

// GetByID retrieves an API key by its ID
func (r *APIKeyRepository) GetByID(id string) (*auth.APIKey, error) {
	query := `
		SELECT id, tenant_id, key_hash, name, scopes, created_at, expires_at, last_used_at, revoked_at
		FROM api_keys
		WHERE id = $1
	`

	apiKey := &auth.APIKey{}
	var scopes pq.StringArray

	err := r.db.QueryRow(query, id).Scan(
		&apiKey.ID,
		&apiKey.TenantID,
		&apiKey.KeyHash,
		&apiKey.Name,
		&scopes,
		&apiKey.CreatedAt,
		&apiKey.ExpiresAt,
		&apiKey.LastUsedAt,
		&apiKey.RevokedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Key not found
		}
		r.logger.Error("failed to get API key by ID", zap.Error(err))
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	apiKey.Scopes = scopes
	return apiKey, nil
}

// GetByTenantID retrieves all API keys for a tenant (including revoked)
func (r *APIKeyRepository) GetByTenantID(tenantID string) ([]*auth.APIKey, error) {
	query := `
		SELECT id, tenant_id, key_hash, name, scopes, created_at, expires_at, last_used_at, revoked_at
		FROM api_keys
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		r.logger.Error("failed to get API keys by tenant ID", zap.Error(err))
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []*auth.APIKey

	for rows.Next() {
		apiKey := &auth.APIKey{}
		var scopes pq.StringArray

		err := rows.Scan(
			&apiKey.ID,
			&apiKey.TenantID,
			&apiKey.KeyHash,
			&apiKey.Name,
			&scopes,
			&apiKey.CreatedAt,
			&apiKey.ExpiresAt,
			&apiKey.LastUsedAt,
			&apiKey.RevokedAt,
		)

		if err != nil {
			r.logger.Error("failed to scan API key", zap.Error(err))
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}

		apiKey.Scopes = scopes
		apiKeys = append(apiKeys, apiKey)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("error iterating API keys", zap.Error(err))
		return nil, fmt.Errorf("error iterating API keys: %w", err)
	}

	return apiKeys, nil
}

// Update updates an existing API key
func (r *APIKeyRepository) Update(apiKey *auth.APIKey) error {
	query := `
		UPDATE api_keys
		SET last_used_at = $1, revoked_at = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`

	result, err := r.db.Exec(query, apiKey.LastUsedAt, apiKey.RevokedAt, apiKey.ID)
	if err != nil {
		r.logger.Error("failed to update API key", zap.Error(err))
		return fmt.Errorf("failed to update API key: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

// Delete removes an API key from the database (hard delete)
func (r *APIKeyRepository) Delete(id string) error {
	query := `DELETE FROM api_keys WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		r.logger.Error("failed to delete API key", zap.Error(err))
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

// ListValid lists all valid (non-revoked, non-expired) API keys for a tenant
func (r *APIKeyRepository) ListValid(tenantID string) ([]*auth.APIKey, error) {
	now := time.Now().UTC()

	query := `
		SELECT id, tenant_id, key_hash, name, scopes, created_at, expires_at, last_used_at, revoked_at
		FROM api_keys
		WHERE tenant_id = $1
			AND revoked_at IS NULL
			AND (expires_at IS NULL OR expires_at > $2)
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, tenantID, now)
	if err != nil {
		r.logger.Error("failed to list valid API keys", zap.Error(err))
		return nil, fmt.Errorf("failed to list valid API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []*auth.APIKey

	for rows.Next() {
		apiKey := &auth.APIKey{}
		var scopes pq.StringArray

		err := rows.Scan(
			&apiKey.ID,
			&apiKey.TenantID,
			&apiKey.KeyHash,
			&apiKey.Name,
			&scopes,
			&apiKey.CreatedAt,
			&apiKey.ExpiresAt,
			&apiKey.LastUsedAt,
			&apiKey.RevokedAt,
		)

		if err != nil {
			r.logger.Error("failed to scan API key", zap.Error(err))
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}

		apiKey.Scopes = scopes
		apiKeys = append(apiKeys, apiKey)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("error iterating valid API keys", zap.Error(err))
		return nil, fmt.Errorf("error iterating valid API keys: %w", err)
	}

	return apiKeys, nil
}
