package build

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// BuildPolicyService defines the business logic for build policy management
type BuildPolicyService struct {
	repository BuildPolicyRepository
	logger     *zap.Logger
}

// NewBuildPolicyService creates a new build policy service
func NewBuildPolicyService(repository BuildPolicyRepository, logger *zap.Logger) *BuildPolicyService {
	return &BuildPolicyService{
		repository: repository,
		logger:     logger,
	}
}

// CreatePolicy creates a new build policy
func (s *BuildPolicyService) CreatePolicy(ctx context.Context, tenantID uuid.UUID, policyType PolicyType, policyKey string, policyValue PolicyValue, userID *uuid.UUID) (*BuildPolicy, error) {
	s.logger.Info("Creating build policy",
		zap.String("tenant_id", tenantID.String()),
		zap.String("policy_type", string(policyType)),
		zap.String("policy_key", policyKey))

	// Check if policy already exists
	exists, err := s.repository.ExistsByTenantAndKey(ctx, tenantID, policyKey)
	if err != nil {
		s.logger.Error("Failed to check policy existence", zap.Error(err))
		return nil, fmt.Errorf("failed to check policy existence: %w", err)
	}

	if exists {
		return nil, ErrDuplicatePolicyKey
	}

	// Create new policy
	policy, err := NewBuildPolicy(tenantID, policyType, policyKey, policyValue)
	if err != nil {
		s.logger.Error("Failed to create build policy", zap.Error(err))
		return nil, fmt.Errorf("failed to create build policy: %w", err)
	}

	// Set created by user
	policy.SetUpdatedBy(userID)

	// Save to repository
	if err := s.repository.Save(ctx, policy); err != nil {
		s.logger.Error("Failed to save build policy", zap.Error(err))
		return nil, fmt.Errorf("failed to save build policy: %w", err)
	}

	s.logger.Info("Build policy created successfully",
		zap.String("policy_id", policy.ID().String()),
		zap.String("policy_key", policyKey))

	return policy, nil
}

// GetPolicy retrieves a policy by ID
func (s *BuildPolicyService) GetPolicy(ctx context.Context, policyID uuid.UUID) (*BuildPolicy, error) {
	s.logger.Debug("Getting build policy", zap.String("policy_id", policyID.String()))

	policy, err := s.repository.FindByID(ctx, policyID)
	if err != nil {
		s.logger.Error("Failed to get build policy", zap.Error(err), zap.String("policy_id", policyID.String()))
		return nil, err
	}

	return policy, nil
}

// GetPoliciesByTenant retrieves all policies for a tenant
func (s *BuildPolicyService) GetPoliciesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*BuildPolicy, error) {
	s.logger.Debug("Getting build policies for tenant", zap.String("tenant_id", tenantID.String()))

	policies, err := s.repository.FindByTenantID(ctx, tenantID)
	if err != nil {
		s.logger.Error("Failed to get build policies for tenant", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil, err
	}

	return policies, nil
}

// GetAllPolicies retrieves all policies across tenants.
func (s *BuildPolicyService) GetAllPolicies(ctx context.Context) ([]*BuildPolicy, error) {
	s.logger.Debug("Getting build policies for all tenants")
	policies, err := s.repository.FindAll(ctx)
	if err != nil {
		s.logger.Error("Failed to get build policies for all tenants", zap.Error(err))
		return nil, err
	}
	return policies, nil
}

// GetActivePoliciesByTenant retrieves only active policies for a tenant
func (s *BuildPolicyService) GetActivePoliciesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*BuildPolicy, error) {
	s.logger.Debug("Getting active build policies for tenant", zap.String("tenant_id", tenantID.String()))

	policies, err := s.repository.FindActiveByTenantID(ctx, tenantID)
	if err != nil {
		s.logger.Error("Failed to get active build policies for tenant", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil, err
	}

	return policies, nil
}

// GetAllActivePolicies retrieves only active policies across tenants.
func (s *BuildPolicyService) GetAllActivePolicies(ctx context.Context) ([]*BuildPolicy, error) {
	s.logger.Debug("Getting active build policies for all tenants")
	policies, err := s.repository.FindAllActive(ctx)
	if err != nil {
		s.logger.Error("Failed to get active build policies for all tenants", zap.Error(err))
		return nil, err
	}
	return policies, nil
}

// GetPolicyByKey retrieves a specific policy by tenant and key
func (s *BuildPolicyService) GetPolicyByKey(ctx context.Context, tenantID uuid.UUID, policyKey string) (*BuildPolicy, error) {
	s.logger.Debug("Getting build policy by key",
		zap.String("tenant_id", tenantID.String()),
		zap.String("policy_key", policyKey))

	policy, err := s.repository.FindByTenantAndKey(ctx, tenantID, policyKey)
	if err != nil {
		s.logger.Error("Failed to get build policy by key", zap.Error(err),
			zap.String("tenant_id", tenantID.String()), zap.String("policy_key", policyKey))
		return nil, err
	}

	return policy, nil
}

// GetPoliciesByType retrieves policies by type for a tenant
func (s *BuildPolicyService) GetPoliciesByType(ctx context.Context, tenantID uuid.UUID, policyType PolicyType) ([]*BuildPolicy, error) {
	s.logger.Debug("Getting build policies by type",
		zap.String("tenant_id", tenantID.String()),
		zap.String("policy_type", string(policyType)))

	policies, err := s.repository.FindByType(ctx, tenantID, policyType)
	if err != nil {
		s.logger.Error("Failed to get build policies by type", zap.Error(err),
			zap.String("tenant_id", tenantID.String()), zap.String("policy_type", string(policyType)))
		return nil, err
	}

	return policies, nil
}

// GetAllPoliciesByType retrieves policies by type across tenants.
func (s *BuildPolicyService) GetAllPoliciesByType(ctx context.Context, policyType PolicyType) ([]*BuildPolicy, error) {
	s.logger.Debug("Getting build policies by type for all tenants",
		zap.String("policy_type", string(policyType)))

	policies, err := s.repository.FindAllByType(ctx, policyType)
	if err != nil {
		s.logger.Error("Failed to get build policies by type for all tenants", zap.Error(err),
			zap.String("policy_type", string(policyType)))
		return nil, err
	}
	return policies, nil
}

// UpdatePolicy updates an existing policy
func (s *BuildPolicyService) UpdatePolicy(ctx context.Context, policyID uuid.UUID, policyValue PolicyValue, description string, isActive bool, userID *uuid.UUID) (*BuildPolicy, error) {
	s.logger.Info("Updating build policy", zap.String("policy_id", policyID.String()))

	// Get existing policy
	policy, err := s.repository.FindByID(ctx, policyID)
	if err != nil {
		s.logger.Error("Failed to find policy for update", zap.Error(err), zap.String("policy_id", policyID.String()))
		return nil, err
	}

	// Update policy value if provided
	if policyValue.Value != nil || policyValue.Data != nil {
		if err := policy.SetPolicyValue(policyValue); err != nil {
			s.logger.Error("Invalid policy value", zap.Error(err), zap.String("policy_id", policyID.String()))
			return nil, err
		}
	}

	// Update description if provided
	if description != "" {
		policy.SetDescription(description)
	}

	// Update active status
	policy.SetActive(isActive)

	// Set updated by user
	policy.SetUpdatedBy(userID)

	// Save changes
	if err := s.repository.Update(ctx, policy); err != nil {
		s.logger.Error("Failed to update build policy", zap.Error(err), zap.String("policy_id", policyID.String()))
		return nil, err
	}

	s.logger.Info("Build policy updated successfully", zap.String("policy_id", policyID.String()))
	return policy, nil
}

// DeletePolicy deletes a policy
func (s *BuildPolicyService) DeletePolicy(ctx context.Context, policyID uuid.UUID) error {
	s.logger.Info("Deleting build policy", zap.String("policy_id", policyID.String()))

	if err := s.repository.Delete(ctx, policyID); err != nil {
		s.logger.Error("Failed to delete build policy", zap.Error(err), zap.String("policy_id", policyID.String()))
		return err
	}

	s.logger.Info("Build policy deleted successfully", zap.String("policy_id", policyID.String()))
	return nil
}

// ValidatePolicyValue validates a policy value without creating the policy
func (s *BuildPolicyService) ValidatePolicyValue(ctx context.Context, policyType PolicyType, policyKey string, policyValue PolicyValue) error {
	s.logger.Debug("Validating policy value",
		zap.String("policy_type", string(policyType)),
		zap.String("policy_key", policyKey))

	return validatePolicyValue(policyType, policyKey, policyValue)
}

// GetDefaultPolicies returns a map of default policies for a tenant
func (s *BuildPolicyService) GetDefaultPolicies() map[string]struct {
	Type        PolicyType
	Value       PolicyValue
	Description string
} {
	return map[string]struct {
		Type        PolicyType
		Value       PolicyValue
		Description string
	}{
		"max_build_duration": {
			Type: PolicyTypeResourceLimit,
			Value: PolicyValue{
				Value: 2,
				Unit:  "hours",
			},
			Description: "Maximum duration a single build can run",
		},
		"concurrent_builds_per_tenant": {
			Type: PolicyTypeResourceLimit,
			Value: PolicyValue{
				Value: 5,
			},
			Description: "Maximum simultaneous builds allowed per tenant",
		},
		"storage_quota_per_build": {
			Type: PolicyTypeResourceLimit,
			Value: PolicyValue{
				Value: 10,
				Unit:  "GB",
			},
			Description: "Maximum disk space a build can use",
		},
		"maintenance_windows": {
			Type: PolicyTypeSchedulingRule,
			Value: PolicyValue{
				Data: map[string]interface{}{
					"schedule": "weekends 2-4 AM",
					"timezone": "UTC",
				},
			},
			Description: "Scheduled maintenance periods when builds are paused",
		},
		"priority_queuing": {
			Type: PolicyTypeSchedulingRule,
			Value: PolicyValue{
				Data: map[string]interface{}{
					"algorithm": "priority-based",
					"levels":    []string{"low", "normal", "high", "urgent"},
				},
			},
			Description: "How builds are prioritized in the queue",
		},
		"approval_required": {
			Type: PolicyTypeApprovalWorkflow,
			Value: PolicyValue{
				Data: map[string]interface{}{
					"enabled":    false,
					"conditions": []string{"production_deployment", "privileged_access"},
				},
			},
			Description: "Whether builds require approval before execution",
		},
		"auto_approval_threshold": {
			Type: PolicyTypeApprovalWorkflow,
			Value: PolicyValue{
				Data: map[string]interface{}{
					"max_duration":  "1h",
					"max_resources": "medium",
				},
			},
			Description: "Criteria for automatic approval of builds",
		},
	}
}
