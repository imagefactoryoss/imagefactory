package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/domain/build"
	"go.uber.org/zap"
)

type BuildPolicyRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewBuildPolicyRepository creates a new PostgreSQL build policy repository
func NewBuildPolicyRepository(db *sqlx.DB, logger *zap.Logger) build.BuildPolicyRepository {
	return &BuildPolicyRepository{
		db:     db,
		logger: logger,
	}
}

// dbBuildPolicy represents the database structure for build policies
type dbBuildPolicy struct {
	ID          uuid.UUID  `db:"id"`
	TenantID    uuid.UUID  `db:"tenant_id"`
	PolicyType  string     `db:"policy_type"`
	PolicyKey   string     `db:"policy_key"`
	PolicyValue []byte     `db:"policy_value"`
	Description *string    `db:"description"`
	IsActive    bool       `db:"is_active"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	CreatedBy   *uuid.UUID `db:"created_by"`
	UpdatedBy   *uuid.UUID `db:"updated_by"`
}

// Save persists a build policy to the database
func (r *BuildPolicyRepository) Save(ctx context.Context, policy *build.BuildPolicy) error {
	r.logger.Info("Saving build policy",
		zap.String("policy_id", policy.ID().String()),
		zap.String("tenant_id", policy.TenantID().String()),
		zap.String("policy_key", policy.PolicyKey()))

	// Marshal policy value to JSON
	policyValueJSON, err := json.Marshal(policy.PolicyValue())
	if err != nil {
		r.logger.Error("Failed to marshal policy value", zap.Error(err))
		return fmt.Errorf("failed to marshal policy value: %w", err)
	}

	query := `
		INSERT INTO build_policies (id, tenant_id, policy_type, policy_key, policy_value, description, is_active, created_at, updated_at, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (tenant_id, policy_key) DO UPDATE SET
			policy_value = EXCLUDED.policy_value,
			description = EXCLUDED.description,
			is_active = EXCLUDED.is_active,
			updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by`

	now := time.Now().UTC()
	description := policy.Description()
	var descPtr *string
	if description != "" {
		descPtr = &description
	}

	_, err = r.db.ExecContext(ctx, query,
		policy.ID(),
		policy.TenantID(),
		string(policy.PolicyType()),
		policy.PolicyKey(),
		policyValueJSON,
		descPtr,
		policy.IsActive(),
		policy.CreatedAt(),
		now,
		policy.CreatedBy(),
		policy.UpdatedBy(),
	)

	if err != nil {
		r.logger.Error("Failed to save build policy", zap.Error(err), zap.String("policy_id", policy.ID().String()))
		return fmt.Errorf("failed to save build policy: %w", err)
	}

	r.logger.Info("Build policy saved successfully", zap.String("policy_id", policy.ID().String()))
	return nil
}

// FindByID retrieves a build policy by ID
func (r *BuildPolicyRepository) FindByID(ctx context.Context, id uuid.UUID) (*build.BuildPolicy, error) {
	query := `
		SELECT id, tenant_id, policy_type, policy_key, policy_value, description, is_active, created_at, updated_at, created_by, updated_by
		FROM build_policies
		WHERE id = $1`

	var policyData dbBuildPolicy
	err := r.db.GetContext(ctx, &policyData, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, build.ErrBuildPolicyNotFound
		}
		r.logger.Error("Failed to find build policy by ID", zap.Error(err), zap.String("policy_id", id.String()))
		return nil, fmt.Errorf("failed to find build policy: %w", err)
	}

	return r.buildPolicyFromDB(policyData), nil
}

// FindByTenantID retrieves all policies for a tenant
func (r *BuildPolicyRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*build.BuildPolicy, error) {
	query := `
		SELECT id, tenant_id, policy_type, policy_key, policy_value, description, is_active, created_at, updated_at, created_by, updated_by
		FROM build_policies
		WHERE tenant_id = $1
		ORDER BY policy_type, policy_key`

	var policiesData []dbBuildPolicy
	err := r.db.SelectContext(ctx, &policiesData, query, tenantID)
	if err != nil {
		r.logger.Error("Failed to find build policies by tenant", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil, fmt.Errorf("failed to find build policies: %w", err)
	}

	policies := make([]*build.BuildPolicy, len(policiesData))
	for i, data := range policiesData {
		policies[i] = r.buildPolicyFromDB(data)
	}

	return policies, nil
}

// FindAll retrieves all policies across tenants.
func (r *BuildPolicyRepository) FindAll(ctx context.Context) ([]*build.BuildPolicy, error) {
	query := `
		SELECT id, tenant_id, policy_type, policy_key, policy_value, description, is_active, created_at, updated_at, created_by, updated_by
		FROM build_policies
		ORDER BY tenant_id, policy_type, policy_key`

	var policiesData []dbBuildPolicy
	err := r.db.SelectContext(ctx, &policiesData, query)
	if err != nil {
		r.logger.Error("Failed to find all build policies", zap.Error(err))
		return nil, fmt.Errorf("failed to find build policies: %w", err)
	}

	policies := make([]*build.BuildPolicy, len(policiesData))
	for i, data := range policiesData {
		policies[i] = r.buildPolicyFromDB(data)
	}

	return policies, nil
}

// FindByTenantAndKey retrieves a specific policy by tenant and key
func (r *BuildPolicyRepository) FindByTenantAndKey(ctx context.Context, tenantID uuid.UUID, policyKey string) (*build.BuildPolicy, error) {
	query := `
		SELECT id, tenant_id, policy_type, policy_key, policy_value, description, is_active, created_at, updated_at, created_by, updated_by
		FROM build_policies
		WHERE tenant_id = $1 AND policy_key = $2`

	var policyData dbBuildPolicy
	err := r.db.GetContext(ctx, &policyData, query, tenantID, policyKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, build.ErrBuildPolicyNotFound
		}
		r.logger.Error("Failed to find build policy by tenant and key", zap.Error(err), zap.String("tenant_id", tenantID.String()), zap.String("policy_key", policyKey))
		return nil, fmt.Errorf("failed to find build policy: %w", err)
	}

	return r.buildPolicyFromDB(policyData), nil
}

// FindByType retrieves policies by type for a tenant
func (r *BuildPolicyRepository) FindByType(ctx context.Context, tenantID uuid.UUID, policyType build.PolicyType) ([]*build.BuildPolicy, error) {
	query := `
		SELECT id, tenant_id, policy_type, policy_key, policy_value, description, is_active, created_at, updated_at, created_by, updated_by
		FROM build_policies
		WHERE tenant_id = $1 AND policy_type = $2
		ORDER BY policy_key`

	var policiesData []dbBuildPolicy
	err := r.db.SelectContext(ctx, &policiesData, query, tenantID, string(policyType))
	if err != nil {
		r.logger.Error("Failed to find build policies by type", zap.Error(err), zap.String("tenant_id", tenantID.String()), zap.String("policy_type", string(policyType)))
		return nil, fmt.Errorf("failed to find build policies: %w", err)
	}

	policies := make([]*build.BuildPolicy, len(policiesData))
	for i, data := range policiesData {
		policies[i] = r.buildPolicyFromDB(data)
	}

	return policies, nil
}

// FindAllByType retrieves policies by type across tenants.
func (r *BuildPolicyRepository) FindAllByType(ctx context.Context, policyType build.PolicyType) ([]*build.BuildPolicy, error) {
	query := `
		SELECT id, tenant_id, policy_type, policy_key, policy_value, description, is_active, created_at, updated_at, created_by, updated_by
		FROM build_policies
		WHERE policy_type = $1
		ORDER BY tenant_id, policy_key`

	var policiesData []dbBuildPolicy
	err := r.db.SelectContext(ctx, &policiesData, query, string(policyType))
	if err != nil {
		r.logger.Error("Failed to find build policies by type across tenants", zap.Error(err), zap.String("policy_type", string(policyType)))
		return nil, fmt.Errorf("failed to find build policies: %w", err)
	}

	policies := make([]*build.BuildPolicy, len(policiesData))
	for i, data := range policiesData {
		policies[i] = r.buildPolicyFromDB(data)
	}

	return policies, nil
}

// FindActiveByTenantID retrieves only active policies for a tenant
func (r *BuildPolicyRepository) FindActiveByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*build.BuildPolicy, error) {
	query := `
		SELECT id, tenant_id, policy_type, policy_key, policy_value, description, is_active, created_at, updated_at, created_by, updated_by
		FROM build_policies
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY policy_type, policy_key`

	var policiesData []dbBuildPolicy
	err := r.db.SelectContext(ctx, &policiesData, query, tenantID)
	if err != nil {
		r.logger.Error("Failed to find active build policies by tenant", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil, fmt.Errorf("failed to find active build policies: %w", err)
	}

	policies := make([]*build.BuildPolicy, len(policiesData))
	for i, data := range policiesData {
		policies[i] = r.buildPolicyFromDB(data)
	}

	return policies, nil
}

// FindAllActive retrieves only active policies across tenants.
func (r *BuildPolicyRepository) FindAllActive(ctx context.Context) ([]*build.BuildPolicy, error) {
	query := `
		SELECT id, tenant_id, policy_type, policy_key, policy_value, description, is_active, created_at, updated_at, created_by, updated_by
		FROM build_policies
		WHERE is_active = true
		ORDER BY tenant_id, policy_type, policy_key`

	var policiesData []dbBuildPolicy
	err := r.db.SelectContext(ctx, &policiesData, query)
	if err != nil {
		r.logger.Error("Failed to find active build policies across tenants", zap.Error(err))
		return nil, fmt.Errorf("failed to find active build policies: %w", err)
	}

	policies := make([]*build.BuildPolicy, len(policiesData))
	for i, data := range policiesData {
		policies[i] = r.buildPolicyFromDB(data)
	}

	return policies, nil
}

// Update updates an existing build policy
func (r *BuildPolicyRepository) Update(ctx context.Context, policy *build.BuildPolicy) error {
	r.logger.Info("Updating build policy",
		zap.String("policy_id", policy.ID().String()),
		zap.String("policy_key", policy.PolicyKey()))

	// Marshal policy value to JSON
	policyValueJSON, err := json.Marshal(policy.PolicyValue())
	if err != nil {
		r.logger.Error("Failed to marshal policy value", zap.Error(err))
		return fmt.Errorf("failed to marshal policy value: %w", err)
	}

	query := `
		UPDATE build_policies
		SET policy_value = $1, description = $2, is_active = $3, updated_at = $4, updated_by = $5
		WHERE id = $6`

	description := policy.Description()
	var descPtr *string
	if description != "" {
		descPtr = &description
	}

	result, err := r.db.ExecContext(ctx, query,
		policyValueJSON,
		descPtr,
		policy.IsActive(),
		policy.UpdatedAt(),
		policy.UpdatedBy(),
		policy.ID(),
	)

	if err != nil {
		r.logger.Error("Failed to update build policy", zap.Error(err), zap.String("policy_id", policy.ID().String()))
		return fmt.Errorf("failed to update build policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected", zap.Error(err))
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return build.ErrBuildPolicyNotFound
	}

	r.logger.Info("Build policy updated successfully", zap.String("policy_id", policy.ID().String()))
	return nil
}

// Delete removes a build policy
func (r *BuildPolicyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.logger.Info("Deleting build policy", zap.String("policy_id", id.String()))

	query := `DELETE FROM build_policies WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete build policy", zap.Error(err), zap.String("policy_id", id.String()))
		return fmt.Errorf("failed to delete build policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Failed to get rows affected", zap.Error(err))
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return build.ErrBuildPolicyNotFound
	}

	r.logger.Info("Build policy deleted successfully", zap.String("policy_id", id.String()))
	return nil
}

// ExistsByTenantAndKey checks if a policy exists for the tenant and key
func (r *BuildPolicyRepository) ExistsByTenantAndKey(ctx context.Context, tenantID uuid.UUID, policyKey string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM build_policies WHERE tenant_id = $1 AND policy_key = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, tenantID, policyKey)
	if err != nil {
		r.logger.Error("Failed to check policy existence", zap.Error(err), zap.String("tenant_id", tenantID.String()), zap.String("policy_key", policyKey))
		return false, fmt.Errorf("failed to check policy existence: %w", err)
	}

	return exists, nil
}

// CountByTenantID counts policies for a tenant
func (r *BuildPolicyRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM build_policies WHERE tenant_id = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, tenantID)
	if err != nil {
		r.logger.Error("Failed to count build policies", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return 0, fmt.Errorf("failed to count build policies: %w", err)
	}

	return count, nil
}

// buildPolicyFromDB converts database struct to domain object
func (r *BuildPolicyRepository) buildPolicyFromDB(data dbBuildPolicy) *build.BuildPolicy {
	// Unmarshal policy value from JSON
	var policyValue build.PolicyValue
	if err := json.Unmarshal(data.PolicyValue, &policyValue); err != nil {
		r.logger.Error("Failed to unmarshal policy value", zap.Error(err), zap.String("policy_id", data.ID.String()))
		// Return policy with empty value if unmarshaling fails
		policyValue = build.PolicyValue{}
	}

	// Create new policy with the data
	policy, err := build.NewBuildPolicy(data.TenantID, build.PolicyType(data.PolicyType), data.PolicyKey, policyValue)
	if err != nil {
		r.logger.Error("Failed to create build policy from database", zap.Error(err), zap.String("policy_id", data.ID.String()))
		return nil
	}

	// Set additional fields
	if data.Description != nil {
		policy.SetDescription(*data.Description)
	}
	if !data.IsActive {
		policy.SetActive(false)
	}

	// Note: In a complete implementation, we would need domain methods to restore
	// timestamps and user references. For now, this creates a policy with current timestamps.

	return policy
}
