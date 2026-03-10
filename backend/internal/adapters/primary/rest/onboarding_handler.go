package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/tenant"
	"github.com/srikarm/image-factory/internal/domain/user"
)

// OnboardingHandler handles tenant onboarding HTTP requests
type OnboardingHandler struct {
	onboardingService *tenant.OnboardingService
	userService       *user.Service
	db                *sqlx.DB
	logger            *zap.Logger
}

// NewOnboardingHandler creates a new onboarding handler
func NewOnboardingHandler(
	onboardingService *tenant.OnboardingService,
	userService *user.Service,
	db *sqlx.DB,
	logger *zap.Logger,
) *OnboardingHandler {
	return &OnboardingHandler{
		onboardingService: onboardingService,
		userService:       userService,
		db:                db,
		logger:            logger,
	}
}

// StartOnboardingRequest represents a request to start tenant onboarding
type StartOnboardingRequest struct {
	TenantID       uuid.UUID                `json:"tenant_id" validate:"required"`
	CompanyID      uuid.UUID                `json:"company_id" validate:"required"`
	TenantCode     string                   `json:"tenant_code,omitempty" validate:"numeric,max=8"`
	TenantName     string                   `json:"tenant_name" validate:"required"`
	TenantSlug     string                   `json:"tenant_slug" validate:"required,alphanum"`
	AdminEmail     string                   `json:"admin_email" validate:"required,email"`
	AdminFirstName string                   `json:"admin_first_name" validate:"required"`
	AdminLastName  string                   `json:"admin_last_name" validate:"required"`
	AdminPassword  string                   `json:"admin_password" validate:"required,min=8"`
	Template       string                   `json:"template,omitempty"`
	CustomData     map[string]interface{}   `json:"custom_data,omitempty"`
}

// OnboardingStatusResponse represents the response for onboarding status
type OnboardingStatusResponse struct {
	ID          uuid.UUID                  `json:"id"`
	TenantID    uuid.UUID                  `json:"tenant_id"`
	TenantCode  string                     `json:"tenant_code,omitempty"`
	Status      string                     `json:"status"`
	Steps       []OnboardingStepResponse   `json:"steps"`
	StartedAt   string                     `json:"started_at"`
	CompletedAt *string                    `json:"completed_at,omitempty"`
}

// OnboardingStepResponse represents onboarding step information in responses
type OnboardingStepResponse struct {
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Status        string  `json:"status"`
	CompletedAt   *string `json:"completed_at,omitempty"`
}

// StartOnboarding handles POST /onboarding/start
func (h *OnboardingHandler) StartOnboarding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StartOnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode onboarding request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Auto-generate tenant code if not provided
	if req.TenantCode == "" {
		code, err := tenant.GenerateTenantCode()
		if err != nil {
			h.logger.Error("Failed to generate tenant code", zap.Error(err))
			http.Error(w, "Failed to generate tenant code", http.StatusInternalServerError)
			return
		}
		req.TenantCode = code
	}

	// Validate required fields
	if err := h.validateStartOnboardingRequest(req); err != nil {
		h.logger.Warn("Invalid onboarding request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Start the onboarding process
	onboardingReq := tenant.OnboardingRequest{
		TenantID:      req.TenantID,
		CompanyID:     req.CompanyID,
		TenantCode:    req.TenantCode,
		TenantName:    req.TenantName,
		TenantSlug:    req.TenantSlug,
		AdminEmail:    req.AdminEmail,
		AdminFirstName: req.AdminFirstName,
		AdminLastName:  req.AdminLastName,
		AdminPassword: req.AdminPassword,
		Template:      req.Template,
		CustomData:    req.CustomData,
	}

	workflow, err := h.onboardingService.StartOnboarding(r.Context(), onboardingReq)
	if err != nil {
		h.logger.Error("Failed to start onboarding", 
			zap.String("tenant_id", req.TenantID.String()), 
			zap.Error(err))
		http.Error(w, "Failed to start onboarding process", http.StatusInternalServerError)
		return
	}

	// Create admin user
	var adminUserID uuid.UUID
	if h.userService != nil {
		adminUser, err := h.userService.CreateUser(
			r.Context(),
			req.TenantID,
			req.AdminEmail,
			req.AdminFirstName,
			req.AdminLastName,
			req.AdminPassword,
		)
		if err != nil {
			h.logger.Warn("Failed to create admin user after onboarding",
				zap.String("tenant_id", req.TenantID.String()),
				zap.String("admin_email", req.AdminEmail),
				zap.Error(err))
			// Don't fail the whole onboarding, just log the warning
		} else {
			adminUserID = adminUser.ID()
			h.logger.Info("Admin user created successfully",
				zap.String("user_id", adminUserID.String()),
				zap.String("admin_email", req.AdminEmail))
			
			// Add admin to tenant admin group
			if h.db != nil {
				if err := h.createAdminGroupAndAddUser(r.Context(), req.TenantID, adminUserID); err != nil {
					h.logger.Warn("Failed to add admin to group",
						zap.String("tenant_id", req.TenantID.String()),
						zap.String("user_id", adminUserID.String()),
						zap.Error(err))
				} else {
					h.logger.Info("Admin added to admin group successfully",
						zap.String("tenant_id", req.TenantID.String()),
						zap.String("user_id", adminUserID.String()))
				}
			}
		}
	}

	// Convert workflow to response format (include tenant code)
	response := h.convertWorkflowToResponse(workflow, req.TenantCode)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("Onboarding started successfully", 
		zap.String("workflow_id", workflow.ID.String()),
		zap.String("tenant_id", req.TenantID.String()),
		zap.String("tenant_code", req.TenantCode))
}

// GetOnboardingStatus handles GET /onboarding/status/{tenant_id}
func (h *OnboardingHandler) GetOnboardingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get tenant ID from URL path
	tenantIDStr := r.PathValue("tenant_id")
	if tenantIDStr == "" {
		http.Error(w, "Tenant ID is required", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.logger.Warn("Invalid tenant ID format", zap.String("tenant_id", tenantIDStr))
		http.Error(w, "Invalid tenant ID format", http.StatusBadRequest)
		return
	}

	// Get onboarding status
	workflow, err := h.onboardingService.GetOnboardingStatus(r.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to get onboarding status", 
			zap.String("tenant_id", tenantID.String()), 
			zap.Error(err))
		http.Error(w, "Failed to retrieve onboarding status", http.StatusInternalServerError)
		return
	}

	// Convert workflow to response format
	response := h.convertWorkflowToResponse(workflow, "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ResumeOnboarding handles POST /onboarding/resume/{tenant_id}
func (h *OnboardingHandler) ResumeOnboarding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get tenant ID from URL path
	tenantIDStr := r.PathValue("tenant_id")
	if tenantIDStr == "" {
		http.Error(w, "Tenant ID is required", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.logger.Warn("Invalid tenant ID format", zap.String("tenant_id", tenantIDStr))
		http.Error(w, "Invalid tenant ID format", http.StatusBadRequest)
		return
	}

	// Resume onboarding
	if err := h.onboardingService.ResumeOnboarding(r.Context(), tenantID); err != nil {
		h.logger.Error("Failed to resume onboarding", 
			zap.String("tenant_id", tenantID.String()), 
			zap.Error(err))
		http.Error(w, "Failed to resume onboarding process", http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message": "Onboarding resumed successfully",
		"tenant_id": tenantID.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("Onboarding resumed successfully", zap.String("tenant_id", tenantID.String()))
}

// validateStartOnboardingRequest validates the onboarding start request
func (h *OnboardingHandler) validateStartOnboardingRequest(req StartOnboardingRequest) error {
	if req.TenantID == uuid.Nil {
		return &ValidationError{Field: "tenant_id", Message: "Tenant ID is required"}
	}
	if req.CompanyID == uuid.Nil {
		return &ValidationError{Field: "company_id", Message: "Company ID is required"}
	}
	// tenant_code can be auto-generated, so we just check if it's valid (max 8 chars, numeric only)
	if req.TenantCode != "" && len(req.TenantCode) > 8 {
		return &ValidationError{Field: "tenant_code", Message: "Tenant code cannot exceed 8 digits"}
	}
	// Check if tenant code is numeric only (if provided)
	if req.TenantCode != "" {
		for _, ch := range req.TenantCode {
			if ch < '0' || ch > '9' {
				return &ValidationError{Field: "tenant_code", Message: "Tenant code must contain only digits"}
			}
		}
	}
	if req.TenantName == "" {
		return &ValidationError{Field: "tenant_name", Message: "Tenant name is required"}
	}
	if req.TenantSlug == "" {
		return &ValidationError{Field: "tenant_slug", Message: "Tenant slug is required"}
	}
	if req.AdminEmail == "" {
		return &ValidationError{Field: "admin_email", Message: "Admin email is required"}
	}
	if req.AdminFirstName == "" {
		return &ValidationError{Field: "admin_first_name", Message: "Admin first name is required"}
	}
	if req.AdminLastName == "" {
		return &ValidationError{Field: "admin_last_name", Message: "Admin last name is required"}
	}
	if req.AdminPassword == "" {
		return &ValidationError{Field: "admin_password", Message: "Admin password is required"}
	}

	return nil
}

// convertWorkflowToResponse converts a tenant workflow to response format
func (h *OnboardingHandler) convertWorkflowToResponse(workflow *tenant.OnboardingWorkflow, tenantCode string) OnboardingStatusResponse {
	steps := make([]OnboardingStepResponse, len(workflow.Steps))
	for i, step := range workflow.Steps {
		var completedAt *string
		if step.CompletedAt != nil {
			completedAtStr := step.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
			completedAt = &completedAtStr
		}

		steps[i] = OnboardingStepResponse{
			Name:          step.Name,
			Description:   step.Description,
			Status:        string(step.Status),
			CompletedAt:   completedAt,
		}
	}

	var completedAt *string
	if workflow.CompletedAt != nil {
		completedAtStr := workflow.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		completedAt = &completedAtStr
	}

	return OnboardingStatusResponse{
		ID:         workflow.ID,
		TenantID:   workflow.TenantID,
		TenantCode: tenantCode,
		Status:     string(workflow.Status),
		Steps:      steps,
		StartedAt:  workflow.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		CompletedAt: completedAt,
	}
}

// createAdminGroupAndAddUser creates an administrator group for the tenant and adds the admin user to it
func (h *OnboardingHandler) createAdminGroupAndAddUser(ctx context.Context, tenantID, userID uuid.UUID) error {
	// Create the owner group if it doesn't exist
	adminGroupID := uuid.New()
	
	createGroupQuery := `
		INSERT INTO tenant_groups (id, tenant_id, name, slug, description, role_type, is_system_group, status, created_at, updated_at)
		SELECT $1, $2, 'Owners', $3, 'Tenant owners with full administrative access', 'owner', true, 'active', NOW(), NOW()
		WHERE NOT EXISTS (
			SELECT 1 FROM tenant_groups 
			WHERE tenant_id = $2 AND role_type = 'owner'
		)
	`
	
	slug := fmt.Sprintf("admin-%s", tenantID.String()[:8])
	_, err := h.db.Exec(createGroupQuery, adminGroupID, tenantID, slug)
	if err != nil {
		return fmt.Errorf("failed to create admin group: %w", err)
	}
	
	// Get the owner group ID (either the one we just created or the existing one)
	var groupID uuid.UUID
	getGroupQuery := `
		SELECT id FROM tenant_groups 
		WHERE tenant_id = $1 AND role_type = 'owner'
		LIMIT 1
	`
	err = h.db.Get(&groupID, getGroupQuery, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get admin group: %w", err)
	}
	
	// Add the admin user to the group
	addMemberQuery := `
		INSERT INTO group_members (id, group_id, user_id, is_group_admin, added_at)
		VALUES ($1, $2, $3, true, NOW())
		ON CONFLICT (group_id, user_id) DO NOTHING
	`
	
	memberID := uuid.New()
	_, err = h.db.Exec(addMemberQuery, memberID, groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to add user to admin group: %w", err)
	}
	
	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}