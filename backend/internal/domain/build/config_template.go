package build

import (
	"time"

	"github.com/google/uuid"
)

// ConfigTemplate represents a saved build configuration template
type ConfigTemplate struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	ProjectID        uuid.UUID              `json:"project_id" db:"project_id"`
	CreatedByUserID  uuid.UUID              `json:"created_by_user_id" db:"created_by_user_id"`
	Name             string                 `json:"name" db:"name"`
	Description      string                 `json:"description,omitempty" db:"description"`
	Method           string                 `json:"method" db:"method"` // packer, buildx, kaniko, docker, nix
	TemplateData     map[string]interface{} `json:"template_data" db:"template_data"`
	IsShared         bool                   `json:"is_shared" db:"is_shared"`
	IsPublic         bool                   `json:"is_public" db:"is_public"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

// ConfigTemplateShare represents shared access to a template
type ConfigTemplateShare struct {
	ID               uuid.UUID `json:"id" db:"id"`
	TemplateID       uuid.UUID `json:"template_id" db:"template_id"`
	SharedWithUserID uuid.UUID `json:"shared_with_user_id" db:"shared_with_user_id"`
	CanUse           bool      `json:"can_use" db:"can_use"`
	CanEdit          bool      `json:"can_edit" db:"can_edit"`
	CanDelete        bool      `json:"can_delete" db:"can_delete"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// API Request Types

// SaveTemplateRequest payload for saving build config as template
type SaveTemplateRequest struct {
	ProjectID    uuid.UUID              `json:"project_id" binding:"required"`
	Name         string                 `json:"name" binding:"required,min=1,max=255"`
	Description  string                 `json:"description" binding:"max=1000"`
	Method       string                 `json:"method" binding:"required,oneof=packer buildx kaniko docker nix"`
	TemplateData map[string]interface{} `json:"template_data" binding:"required"`
	IsShared     bool                   `json:"is_shared"`
	IsPublic     bool                   `json:"is_public"`
}

// UpdateTemplateRequest payload for updating template
type UpdateTemplateRequest struct {
	TemplateID   uuid.UUID              `json:"template_id" binding:"required"`
	Name         string                 `json:"name" binding:"max=255"`
	Description  string                 `json:"description" binding:"max=1000"`
	TemplateData map[string]interface{} `json:"template_data"`
	IsShared     bool                   `json:"is_shared"`
	IsPublic     bool                   `json:"is_public"`
}

// ShareTemplateRequest payload for sharing template
type ShareTemplateRequest struct {
	TemplateID       uuid.UUID `json:"template_id" binding:"required"`
	SharedWithUserID uuid.UUID `json:"shared_with_user_id" binding:"required"`
	CanUse           bool      `json:"can_use"`
	CanEdit          bool      `json:"can_edit"`
	CanDelete        bool      `json:"can_delete"`
}

// UpdateShareRequest payload for updating share permissions
type UpdateShareRequest struct {
	TemplateID       uuid.UUID `json:"template_id" binding:"required"`
	SharedWithUserID uuid.UUID `json:"shared_with_user_id" binding:"required"`
	CanUse           bool      `json:"can_use"`
	CanEdit          bool      `json:"can_edit"`
	CanDelete        bool      `json:"can_delete"`
}

// API Response Types

// ConfigTemplateResponse represents template in API responses
type ConfigTemplateResponse struct {
	ID              uuid.UUID              `json:"id"`
	ProjectID       uuid.UUID              `json:"project_id"`
	CreatedByUserID uuid.UUID              `json:"created_by_user_id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Method          string                 `json:"method"`
	TemplateData    map[string]interface{} `json:"template_data"`
	IsShared        bool                   `json:"is_shared"`
	IsPublic        bool                   `json:"is_public"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// ConfigTemplateListResponse paginated template list
type ConfigTemplateListResponse struct {
	Templates []*ConfigTemplateResponse `json:"templates"`
	Total     int                       `json:"total"`
	Limit     int                       `json:"limit"`
	Offset    int                       `json:"offset"`
}

// ConfigTemplateShareResponse represents share in API responses
type ConfigTemplateShareResponse struct {
	ID               uuid.UUID `json:"id"`
	TemplateID       uuid.UUID `json:"template_id"`
	SharedWithUserID uuid.UUID `json:"shared_with_user_id"`
	CanUse           bool      `json:"can_use"`
	CanEdit          bool      `json:"can_edit"`
	CanDelete        bool      `json:"can_delete"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Helper method to convert domain to response
func (ct *ConfigTemplate) ToResponse() *ConfigTemplateResponse {
	if ct == nil {
		return nil
	}
	return &ConfigTemplateResponse{
		ID:              ct.ID,
		ProjectID:       ct.ProjectID,
		CreatedByUserID: ct.CreatedByUserID,
		Name:            ct.Name,
		Description:     ct.Description,
		Method:          ct.Method,
		TemplateData:    ct.TemplateData,
		IsShared:        ct.IsShared,
		IsPublic:        ct.IsPublic,
		CreatedAt:       ct.CreatedAt,
		UpdatedAt:       ct.UpdatedAt,
	}
}

// Helper method to convert share domain to response
func (cs *ConfigTemplateShare) ToResponse() *ConfigTemplateShareResponse {
	if cs == nil {
		return nil
	}
	return &ConfigTemplateShareResponse{
		ID:               cs.ID,
		TemplateID:       cs.TemplateID,
		SharedWithUserID: cs.SharedWithUserID,
		CanUse:           cs.CanUse,
		CanEdit:          cs.CanEdit,
		CanDelete:        cs.CanDelete,
		CreatedAt:        cs.CreatedAt,
		UpdatedAt:        cs.UpdatedAt,
	}
}
