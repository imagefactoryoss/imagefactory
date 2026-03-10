package build

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Domain errors for build policies
var (
	ErrBuildPolicyNotFound    = errors.New("build policy not found")
	ErrInvalidPolicyType      = errors.New("invalid policy type")
	ErrInvalidPolicyKey       = errors.New("invalid policy key")
	ErrDuplicatePolicyKey     = errors.New("policy key already exists for this tenant")
	ErrPolicyValidationFailed = errors.New("policy validation failed")
)

// PolicyType represents the type of build policy
type PolicyType string

const (
	PolicyTypeResourceLimit    PolicyType = "resource_limit"
	PolicyTypeSchedulingRule   PolicyType = "scheduling_rule"
	PolicyTypeApprovalWorkflow PolicyType = "approval_workflow"
)

// BuildPolicy represents a configurable build policy
type BuildPolicy struct {
	id          uuid.UUID
	tenantID    uuid.UUID
	policyType  PolicyType
	policyKey   string
	policyValue PolicyValue
	description string
	isActive    bool
	createdAt   time.Time
	updatedAt   time.Time
	createdBy   *uuid.UUID
	updatedBy   *uuid.UUID
	version     int
}

// PolicyValue represents the flexible value structure for policies
type PolicyValue struct {
	Value interface{}            `json:"value,omitempty"`
	Unit  string                 `json:"unit,omitempty"`
	Data  map[string]interface{} `json:"-"`
}

// MarshalJSON custom marshaling for PolicyValue
func (pv PolicyValue) MarshalJSON() ([]byte, error) {
	if pv.Data != nil {
		return json.Marshal(pv.Data)
	}

	// Simple value with unit
	if pv.Unit != "" {
		return json.Marshal(map[string]interface{}{
			"value": pv.Value,
			"unit":  pv.Unit,
		})
	}

	// Simple value only
	return json.Marshal(pv.Value)
}

// UnmarshalJSON custom unmarshaling for PolicyValue
func (pv *PolicyValue) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as complex object first
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil {
		pv.Data = obj
		if val, ok := obj["value"]; ok {
			pv.Value = val
		}
		if unit, ok := obj["unit"].(string); ok {
			pv.Unit = unit
		}
		return nil
	}

	// Try to unmarshal as simple value
	var simple interface{}
	if err := json.Unmarshal(data, &simple); err == nil {
		pv.Value = simple
		return nil
	}

	return errors.New("invalid policy value format")
}

// NewBuildPolicy creates a new build policy
func NewBuildPolicy(tenantID uuid.UUID, policyType PolicyType, policyKey string, policyValue PolicyValue) (*BuildPolicy, error) {
	// Note: uuid.Nil is allowed for system-level policies

	if policyType == "" {
		return nil, ErrInvalidPolicyType
	}

	if policyKey == "" {
		return nil, ErrInvalidPolicyKey
	}

	if err := validatePolicyType(policyType); err != nil {
		return nil, err
	}

	if err := validatePolicyValue(policyType, policyKey, policyValue); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &BuildPolicy{
		id:          uuid.New(),
		tenantID:    tenantID,
		policyType:  policyType,
		policyKey:   policyKey,
		policyValue: policyValue,
		isActive:    true,
		createdAt:   now,
		updatedAt:   now,
		version:     1,
	}, nil
}

// validatePolicyType validates the policy type
func validatePolicyType(policyType PolicyType) error {
	switch policyType {
	case PolicyTypeResourceLimit, PolicyTypeSchedulingRule, PolicyTypeApprovalWorkflow:
		return nil
	default:
		return ErrInvalidPolicyType
	}
}

// validatePolicyValue validates the policy value based on type and key
func validatePolicyValue(policyType PolicyType, policyKey string, value PolicyValue) error {
	switch policyType {
	case PolicyTypeResourceLimit:
		return validateResourceLimitValue(policyKey, value)
	case PolicyTypeSchedulingRule:
		return validateSchedulingRuleValue(policyKey, value)
	case PolicyTypeApprovalWorkflow:
		return validateApprovalWorkflowValue(policyKey, value)
	default:
		return ErrInvalidPolicyType
	}
}

// validateResourceLimitValue validates resource limit policy values
func validateResourceLimitValue(key string, value PolicyValue) error {
	switch key {
	case "max_build_duration":
		if value.Value == nil {
			return errors.New("max_build_duration requires a numeric value")
		}
		if value.Unit == "" {
			return errors.New("max_build_duration requires a unit (e.g., 'hours')")
		}
	case "concurrent_builds_per_tenant":
		if value.Value == nil {
			return errors.New("concurrent_builds_per_tenant requires a numeric value")
		}
	case "storage_quota_per_build":
		if value.Value == nil {
			return errors.New("storage_quota_per_build requires a numeric value")
		}
		if value.Unit == "" {
			return errors.New("storage_quota_per_build requires a unit (e.g., 'GB')")
		}
	default:
		return errors.New("unknown resource limit key: " + key)
	}
	return nil
}

// validateSchedulingRuleValue validates scheduling rule policy values
func validateSchedulingRuleValue(key string, value PolicyValue) error {
	switch key {
	case "maintenance_windows":
		if value.Data == nil || value.Data["schedule"] == nil {
			return errors.New("maintenance_windows requires a schedule field")
		}
	case "priority_queuing":
		if value.Data == nil || value.Data["algorithm"] == nil {
			return errors.New("priority_queuing requires an algorithm field")
		}
	default:
		return errors.New("unknown scheduling rule key: " + key)
	}
	return nil
}

// validateApprovalWorkflowValue validates approval workflow policy values
func validateApprovalWorkflowValue(key string, value PolicyValue) error {
	switch key {
	case "approval_required":
		if value.Data == nil {
			return errors.New("approval_required requires configuration data")
		}
	case "auto_approval_threshold":
		if value.Data == nil {
			return errors.New("auto_approval_threshold requires configuration data")
		}
	default:
		return errors.New("unknown approval workflow key: " + key)
	}
	return nil
}

// Getters
func (bp *BuildPolicy) ID() uuid.UUID            { return bp.id }
func (bp *BuildPolicy) TenantID() uuid.UUID      { return bp.tenantID }
func (bp *BuildPolicy) PolicyType() PolicyType   { return bp.policyType }
func (bp *BuildPolicy) PolicyKey() string        { return bp.policyKey }
func (bp *BuildPolicy) PolicyValue() PolicyValue { return bp.policyValue }
func (bp *BuildPolicy) Description() string      { return bp.description }
func (bp *BuildPolicy) IsActive() bool           { return bp.isActive }
func (bp *BuildPolicy) CreatedAt() time.Time     { return bp.createdAt }
func (bp *BuildPolicy) UpdatedAt() time.Time     { return bp.updatedAt }
func (bp *BuildPolicy) CreatedBy() *uuid.UUID    { return bp.createdBy }
func (bp *BuildPolicy) UpdatedBy() *uuid.UUID    { return bp.updatedBy }
func (bp *BuildPolicy) Version() int             { return bp.version }

// Setters
func (bp *BuildPolicy) SetDescription(description string) {
	bp.description = description
	bp.updatedAt = time.Now().UTC()
}

func (bp *BuildPolicy) SetPolicyValue(value PolicyValue) error {
	if err := validatePolicyValue(bp.policyType, bp.policyKey, value); err != nil {
		return err
	}
	bp.policyValue = value
	bp.updatedAt = time.Now().UTC()
	return nil
}

func (bp *BuildPolicy) SetActive(active bool) {
	bp.isActive = active
	bp.updatedAt = time.Now().UTC()
}

func (bp *BuildPolicy) SetUpdatedBy(userID *uuid.UUID) {
	bp.updatedBy = userID
	bp.updatedAt = time.Now().UTC()
}

// UpdateVersion increments the version for optimistic locking
func (bp *BuildPolicy) UpdateVersion() {
	bp.version++
	bp.updatedAt = time.Now().UTC()
}
