package rest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// BuildPolicyHandler handles build policy-related HTTP requests
type BuildPolicyHandler struct {
	buildPolicyService *build.BuildPolicyService
	auditService       *audit.Service
	logger             *zap.Logger
}

// NewBuildPolicyHandler creates a new build policy handler
func NewBuildPolicyHandler(
	buildPolicyService *build.BuildPolicyService,
	auditService *audit.Service,
	logger *zap.Logger,
) *BuildPolicyHandler {
	return &BuildPolicyHandler{
		buildPolicyService: buildPolicyService,
		auditService:       auditService,
		logger:             logger,
	}
}

// BuildPolicyResponse represents a build policy in API responses
type BuildPolicyResponse struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenant_id"`
	PolicyType  string      `json:"policy_type"`
	PolicyKey   string      `json:"policy_key"`
	PolicyValue interface{} `json:"policy_value"`
	Description string      `json:"description,omitempty"`
	IsActive    bool        `json:"is_active"`
	CreatedAt   string      `json:"created_at"`
	UpdatedAt   string      `json:"updated_at"`
	CreatedBy   *string     `json:"created_by,omitempty"`
	UpdatedBy   *string     `json:"updated_by,omitempty"`
}

// CreateBuildPolicyRequest represents a build policy creation request
type CreateBuildPolicyRequest struct {
	PolicyType  string      `json:"policy_type" validate:"required,oneof=resource_limit scheduling_rule approval_workflow"`
	PolicyKey   string      `json:"policy_key" validate:"required"`
	PolicyValue interface{} `json:"policy_value" validate:"required"`
	Description string      `json:"description,omitempty"`
}

// UpdateBuildPolicyRequest represents a build policy update request
type UpdateBuildPolicyRequest struct {
	PolicyValue interface{} `json:"policy_value,omitempty"`
	Description string      `json:"description,omitempty"`
	IsActive    *bool       `json:"is_active,omitempty"`
}

// ListBuildPoliciesResponse represents a list of build policies
type ListBuildPoliciesResponse struct {
	Policies []BuildPolicyResponse `json:"policies"`
	Total    int                   `json:"total"`
}

func (h *BuildPolicyHandler) resolveScope(r *http.Request, authCtx *middleware.AuthContext) (tenantID uuid.UUID, allTenants bool, status int, message string) {
	return resolveTenantScopeFromRequest(r, authCtx, true)
}

// GetPolicies handles GET /api/v1/admin/builds/policies
func (h *BuildPolicyHandler) GetPolicies(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	scopeTenantID, allTenants, status, message := h.resolveScope(r, authCtx)
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	h.logger.Info("Getting build policies", zap.String("tenant_id", scopeTenantID.String()), zap.Bool("all_tenants", allTenants))

	// Get query parameters
	policyType := r.URL.Query().Get("type")
	activeOnly := r.URL.Query().Get("active_only") == "true"

	var policies []*build.BuildPolicy
	var err error

	if policyType != "" {
		pType := build.PolicyType(policyType)
		if allTenants {
			policies, err = h.buildPolicyService.GetAllPoliciesByType(r.Context(), pType)
		} else {
			policies, err = h.buildPolicyService.GetPoliciesByType(r.Context(), scopeTenantID, pType)
		}
	} else if activeOnly {
		if allTenants {
			policies, err = h.buildPolicyService.GetAllActivePolicies(r.Context())
		} else {
			policies, err = h.buildPolicyService.GetActivePoliciesByTenant(r.Context(), scopeTenantID)
		}
	} else {
		if allTenants {
			policies, err = h.buildPolicyService.GetAllPolicies(r.Context())
		} else {
			policies, err = h.buildPolicyService.GetPoliciesByTenant(r.Context(), scopeTenantID)
		}
	}

	if err != nil {
		h.logger.Error("Failed to get build policies", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	responses := make([]BuildPolicyResponse, len(policies))
	for i, policy := range policies {
		responses[i] = h.buildPolicyToResponse(policy)
	}

	response := ListBuildPoliciesResponse{
		Policies: responses,
		Total:    len(responses),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPolicy handles GET /api/v1/admin/builds/policies/{id}
func (h *BuildPolicyHandler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	scopeTenantID, allTenants, status, message := h.resolveScope(r, authCtx)
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	policyIDStr := chi.URLParam(r, "id")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		h.logger.Error("Invalid policy ID", zap.Error(err), zap.String("policy_id", policyIDStr))
		http.Error(w, "Invalid policy ID", http.StatusBadRequest)
		return
	}

	h.logger.Info("Getting build policy", zap.String("policy_id", policyID.String()))

	policy, err := h.buildPolicyService.GetPolicy(r.Context(), policyID)
	if err != nil {
		if err == build.ErrBuildPolicyNotFound {
			http.Error(w, "Policy not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to get build policy", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Verify policy belongs to tenant
	if !allTenants && policy.TenantID() != scopeTenantID {
		h.logger.Warn("Policy access denied", zap.String("policy_id", policyID.String()), zap.String("tenant_id", scopeTenantID.String()))
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	response := h.buildPolicyToResponse(policy)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreatePolicy handles POST /api/v1/admin/builds/policies
func (h *BuildPolicyHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	scopeTenantID, _, status, message := h.resolveScope(r, authCtx)
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	var req CreateBuildPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	h.logger.Info("Creating build policy",
		zap.String("tenant_id", scopeTenantID.String()),
		zap.String("policy_type", req.PolicyType),
		zap.String("policy_key", req.PolicyKey))

	// Convert policy value to domain type
	policyValue, err := h.convertToPolicyValue(req.PolicyValue)
	if err != nil {
		h.logger.Error("Invalid policy value", zap.Error(err))
		http.Error(w, "Invalid policy value", http.StatusBadRequest)
		return
	}

	// Create policy
	policy, err := h.buildPolicyService.CreatePolicy(r.Context(), scopeTenantID, build.PolicyType(req.PolicyType), req.PolicyKey, policyValue, &authCtx.UserID)
	if err != nil {
		if err == build.ErrDuplicatePolicyKey {
			http.Error(w, "Policy key already exists", http.StatusConflict)
			return
		}
		if err == build.ErrInvalidPolicyType || err == build.ErrInvalidPolicyKey || err == build.ErrPolicyValidationFailed {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.logger.Error("Failed to create build policy", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Audit log
	h.auditService.LogUserAction(r.Context(), authCtx.UserID, scopeTenantID, audit.AuditEventConfigChange, "build_policy", "create", "Build policy created", map[string]interface{}{
		"policy_key":  policy.PolicyKey(),
		"policy_type": policy.PolicyType(),
		"tenant_id":   scopeTenantID.String(),
	})

	response := h.buildPolicyToResponse(policy)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// UpdatePolicy handles PUT /api/v1/admin/builds/policies/{id}
func (h *BuildPolicyHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	scopeTenantID, allTenants, status, message := h.resolveScope(r, authCtx)
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	policyIDStr := chi.URLParam(r, "id")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		h.logger.Error("Invalid policy ID", zap.Error(err), zap.String("policy_id", policyIDStr))
		http.Error(w, "Invalid policy ID", http.StatusBadRequest)
		return
	}

	var req UpdateBuildPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	h.logger.Info("Updating build policy", zap.String("policy_id", policyID.String()))

	// Get existing policy to verify ownership
	existingPolicy, err := h.buildPolicyService.GetPolicy(r.Context(), policyID)
	if err != nil {
		if err == build.ErrBuildPolicyNotFound {
			http.Error(w, "Policy not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to get existing policy", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Verify policy belongs to tenant
	if !allTenants && existingPolicy.TenantID() != scopeTenantID {
		h.logger.Warn("Policy update denied", zap.String("policy_id", policyID.String()), zap.String("tenant_id", scopeTenantID.String()))
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Prepare update values
	var policyValue build.PolicyValue
	var description string
	var isActive bool

	if req.PolicyValue != nil {
		policyValue, err = h.convertToPolicyValue(req.PolicyValue)
		if err != nil {
			h.logger.Error("Invalid policy value", zap.Error(err))
			http.Error(w, "Invalid policy value", http.StatusBadRequest)
			return
		}
	}

	if req.Description != "" {
		description = req.Description
	} else {
		description = existingPolicy.Description()
	}

	if req.IsActive != nil {
		isActive = *req.IsActive
	} else {
		isActive = existingPolicy.IsActive()
	}

	// Update policy
	updatedPolicy, err := h.buildPolicyService.UpdatePolicy(r.Context(), policyID, policyValue, description, isActive, &authCtx.UserID)
	if err != nil {
		if err == build.ErrPolicyValidationFailed {
			http.Error(w, "Invalid policy value", http.StatusBadRequest)
			return
		}
		h.logger.Error("Failed to update build policy", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Audit log
	h.auditService.LogUserAction(r.Context(), authCtx.UserID, existingPolicy.TenantID(), audit.AuditEventConfigChange, "build_policy", "update", "Build policy updated", map[string]interface{}{
		"policy_key": updatedPolicy.PolicyKey(),
		"tenant_id":  existingPolicy.TenantID().String(),
	})

	response := h.buildPolicyToResponse(updatedPolicy)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeletePolicy handles DELETE /api/v1/admin/builds/policies/{id}
func (h *BuildPolicyHandler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	scopeTenantID, allTenants, status, message := h.resolveScope(r, authCtx)
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	policyIDStr := chi.URLParam(r, "id")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		h.logger.Error("Invalid policy ID", zap.Error(err), zap.String("policy_id", policyIDStr))
		http.Error(w, "Invalid policy ID", http.StatusBadRequest)
		return
	}

	h.logger.Info("Deleting build policy", zap.String("policy_id", policyID.String()))

	// Get existing policy to verify ownership
	existingPolicy, err := h.buildPolicyService.GetPolicy(r.Context(), policyID)
	if err != nil {
		if err == build.ErrBuildPolicyNotFound {
			http.Error(w, "Policy not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to get existing policy", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Verify policy belongs to tenant
	if !allTenants && existingPolicy.TenantID() != scopeTenantID {
		h.logger.Warn("Policy delete denied", zap.String("policy_id", policyID.String()), zap.String("tenant_id", scopeTenantID.String()))
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Delete policy
	if err := h.buildPolicyService.DeletePolicy(r.Context(), policyID); err != nil {
		h.logger.Error("Failed to delete build policy", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Audit log
	h.auditService.LogUserAction(r.Context(), authCtx.UserID, existingPolicy.TenantID(), audit.AuditEventConfigChange, "build_policy", "delete", "Build policy deleted", map[string]interface{}{
		"policy_key": existingPolicy.PolicyKey(),
		"tenant_id":  existingPolicy.TenantID().String(),
	})

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func (h *BuildPolicyHandler) buildPolicyToResponse(policy *build.BuildPolicy) BuildPolicyResponse {
	var createdBy, updatedBy *string
	if policy.CreatedBy() != nil {
		createdByStr := policy.CreatedBy().String()
		createdBy = &createdByStr
	}
	if policy.UpdatedBy() != nil {
		updatedByStr := policy.UpdatedBy().String()
		updatedBy = &updatedByStr
	}

	return BuildPolicyResponse{
		ID:          policy.ID().String(),
		TenantID:    policy.TenantID().String(),
		PolicyType:  string(policy.PolicyType()),
		PolicyKey:   policy.PolicyKey(),
		PolicyValue: policy.PolicyValue(),
		Description: policy.Description(),
		IsActive:    policy.IsActive(),
		CreatedAt:   policy.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   policy.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
		CreatedBy:   createdBy,
		UpdatedBy:   updatedBy,
	}
}

func (h *BuildPolicyHandler) convertToPolicyValue(value interface{}) (build.PolicyValue, error) {
	// Convert interface{} to JSON bytes for unmarshaling
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return build.PolicyValue{}, err
	}

	var policyValue build.PolicyValue
	if err := json.Unmarshal(jsonBytes, &policyValue); err != nil {
		return build.PolicyValue{}, err
	}

	return policyValue, nil
}
