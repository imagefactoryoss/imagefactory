package build

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ConfigTemplateRepository interface for template data access
type ConfigTemplateRepository interface {
	SaveTemplate(ctx context.Context, template *ConfigTemplate) error
	GetTemplate(ctx context.Context, id uuid.UUID) (*ConfigTemplate, error)
	ListTemplatesByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*ConfigTemplate, int, error)
	UpdateTemplate(ctx context.Context, template *ConfigTemplate) error
	DeleteTemplate(ctx context.Context, id uuid.UUID) error
	ShareTemplate(ctx context.Context, share *ConfigTemplateShare) error
	GetSharesByTemplate(ctx context.Context, templateID uuid.UUID) ([]*ConfigTemplateShare, error)
	GetSharesByUser(ctx context.Context, userID uuid.UUID) ([]*ConfigTemplateShare, error)
	DeleteShare(ctx context.Context, templateID, userID uuid.UUID) error
}

// ConfigTemplateService interface defines template operations
type ConfigTemplateService interface {
	// Save and retrieve templates
	SaveAsTemplate(ctx context.Context, req *SaveTemplateRequest, createdByUserID uuid.UUID) (*ConfigTemplate, error)
	LoadTemplate(ctx context.Context, templateID uuid.UUID) (*ConfigTemplate, error)
	ListTemplatesByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*ConfigTemplate, int, error)
	UpdateTemplate(ctx context.Context, req *UpdateTemplateRequest) (*ConfigTemplate, error)
	DeleteTemplate(ctx context.Context, templateID uuid.UUID) error

	// Sharing operations
	ShareTemplate(ctx context.Context, req *ShareTemplateRequest) (*ConfigTemplateShare, error)
	UpdateShare(ctx context.Context, req *UpdateShareRequest) (*ConfigTemplateShare, error)
	RevokeShare(ctx context.Context, templateID, userID uuid.UUID) error
	GetSharedTemplatesForUser(ctx context.Context, userID uuid.UUID) ([]*ConfigTemplateShare, error)
	GetTemplateShares(ctx context.Context, templateID uuid.UUID) ([]*ConfigTemplateShare, error)

	// Permission checks
	CanUseTemplate(ctx context.Context, templateID, userID uuid.UUID) (bool, error)
	CanEditTemplate(ctx context.Context, templateID, userID uuid.UUID) (bool, error)
	CanDeleteTemplate(ctx context.Context, templateID, userID uuid.UUID) (bool, error)
}

// ConfigTemplateServiceImpl implements ConfigTemplateService
type ConfigTemplateServiceImpl struct {
	repo ConfigTemplateRepository
}

// NewConfigTemplateServiceImpl creates new template service
func NewConfigTemplateServiceImpl(repo ConfigTemplateRepository) *ConfigTemplateServiceImpl {
	return &ConfigTemplateServiceImpl{
		repo: repo,
	}
}

// ValidateBuildMethod checks if method is valid
func ValidateBuildMethod(method string) bool {
	validMethods := map[string]bool{
		"packer": true,
		"buildx": true,
		"kaniko": true,
		"docker": true,
		"nix":    true,
	}
	return validMethods[method]
}

// SaveAsTemplate saves build configuration as template
func (s *ConfigTemplateServiceImpl) SaveAsTemplate(ctx context.Context, req *SaveTemplateRequest, createdByUserID uuid.UUID) (*ConfigTemplate, error) {
	// Validate inputs
	if req == nil {
		return nil, ErrInvalidManifest
	}

	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrTemplateNameRequired
	}

	if req.ProjectID == uuid.Nil {
		return nil, ErrInvalidBuildID
	}

	if !ValidateBuildMethod(req.Method) {
		return nil, ErrInvalidBuildMethod
	}

	if req.TemplateData == nil {
		req.TemplateData = make(map[string]interface{})
	}

	template := &ConfigTemplate{
		ID:              uuid.New(),
		ProjectID:       req.ProjectID,
		CreatedByUserID: createdByUserID,
		Name:            strings.TrimSpace(req.Name),
		Description:     req.Description,
		Method:          req.Method,
		TemplateData:    req.TemplateData,
		IsShared:        req.IsShared,
		IsPublic:        req.IsPublic,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	return template, s.repo.SaveTemplate(ctx, template)
}

// LoadTemplate retrieves template by ID
func (s *ConfigTemplateServiceImpl) LoadTemplate(ctx context.Context, templateID uuid.UUID) (*ConfigTemplate, error) {
	if templateID == uuid.Nil {
		return nil, ErrInvalidConfigID
	}
	return s.repo.GetTemplate(ctx, templateID)
}

// ListTemplatesByProject lists templates for a project with pagination
func (s *ConfigTemplateServiceImpl) ListTemplatesByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*ConfigTemplate, int, error) {
	if projectID == uuid.Nil {
		return nil, 0, ErrInvalidBuildID
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.ListTemplatesByProject(ctx, projectID, limit, offset)
}

// UpdateTemplate updates template details
func (s *ConfigTemplateServiceImpl) UpdateTemplate(ctx context.Context, req *UpdateTemplateRequest) (*ConfigTemplate, error) {
	if req == nil {
		return nil, ErrInvalidManifest
	}

	if req.TemplateID == uuid.Nil {
		return nil, ErrInvalidConfigID
	}

	// Load existing template
	template, err := s.repo.GetTemplate(ctx, req.TemplateID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if strings.TrimSpace(req.Name) != "" {
		template.Name = strings.TrimSpace(req.Name)
	}

	if req.Description != "" {
		template.Description = req.Description
	}

	if req.TemplateData != nil {
		template.TemplateData = req.TemplateData
	}

	template.IsShared = req.IsShared
	template.IsPublic = req.IsPublic
	template.UpdatedAt = time.Now()

	err = s.repo.UpdateTemplate(ctx, template)
	if err != nil {
		return nil, err
	}

	return template, nil
}

// DeleteTemplate removes template
func (s *ConfigTemplateServiceImpl) DeleteTemplate(ctx context.Context, templateID uuid.UUID) error {
	if templateID == uuid.Nil {
		return ErrInvalidConfigID
	}
	return s.repo.DeleteTemplate(ctx, templateID)
}

// ShareTemplate shares template with another user
func (s *ConfigTemplateServiceImpl) ShareTemplate(ctx context.Context, req *ShareTemplateRequest) (*ConfigTemplateShare, error) {
	if req == nil {
		return nil, ErrInvalidManifest
	}

	if req.TemplateID == uuid.Nil {
		return nil, ErrInvalidConfigID
	}

	if req.SharedWithUserID == uuid.Nil {
		return nil, ErrInvalidManifest
	}

	// Verify template exists
	_, err := s.repo.GetTemplate(ctx, req.TemplateID)
	if err != nil {
		return nil, err
	}

	share := &ConfigTemplateShare{
		ID:               uuid.New(),
		TemplateID:       req.TemplateID,
		SharedWithUserID: req.SharedWithUserID,
		CanUse:           req.CanUse,
		CanEdit:          req.CanEdit,
		CanDelete:        req.CanDelete,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	return share, s.repo.ShareTemplate(ctx, share)
}

// UpdateShare updates share permissions
func (s *ConfigTemplateServiceImpl) UpdateShare(ctx context.Context, req *UpdateShareRequest) (*ConfigTemplateShare, error) {
	if req == nil {
		return nil, ErrInvalidManifest
	}

	if req.TemplateID == uuid.Nil || req.SharedWithUserID == uuid.Nil {
		return nil, ErrInvalidManifest
	}

	// Get existing share
	shares, err := s.repo.GetSharesByTemplate(ctx, req.TemplateID)
	if err != nil {
		return nil, err
	}

	var share *ConfigTemplateShare
	for _, s := range shares {
		if s.SharedWithUserID == req.SharedWithUserID {
			share = s
			break
		}
	}

	if share == nil {
		return nil, ErrShareNotFound
	}

	share.CanUse = req.CanUse
	share.CanEdit = req.CanEdit
	share.CanDelete = req.CanDelete
	share.UpdatedAt = time.Now()

	// Update in repository (would need new method for this)
	// For now, recreate the share
	_ = s.repo.DeleteShare(ctx, share.TemplateID, share.SharedWithUserID)
	return share, s.repo.ShareTemplate(ctx, share)
}

// RevokeShare removes shared access
func (s *ConfigTemplateServiceImpl) RevokeShare(ctx context.Context, templateID, userID uuid.UUID) error {
	if templateID == uuid.Nil || userID == uuid.Nil {
		return ErrInvalidManifest
	}
	return s.repo.DeleteShare(ctx, templateID, userID)
}

// GetSharedTemplatesForUser retrieves templates shared with user
func (s *ConfigTemplateServiceImpl) GetSharedTemplatesForUser(ctx context.Context, userID uuid.UUID) ([]*ConfigTemplateShare, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidManifest
	}
	return s.repo.GetSharesByUser(ctx, userID)
}

// GetTemplateShares retrieves all shares for a template
func (s *ConfigTemplateServiceImpl) GetTemplateShares(ctx context.Context, templateID uuid.UUID) ([]*ConfigTemplateShare, error) {
	if templateID == uuid.Nil {
		return nil, ErrInvalidConfigID
	}
	return s.repo.GetSharesByTemplate(ctx, templateID)
}

// CanUseTemplate checks if user can use template
func (s *ConfigTemplateServiceImpl) CanUseTemplate(ctx context.Context, templateID, userID uuid.UUID) (bool, error) {
	template, err := s.repo.GetTemplate(ctx, templateID)
	if err != nil {
		return false, err
	}

	// Creator can always use
	if template.CreatedByUserID == userID {
		return true, nil
	}

	// Check if public
	if template.IsPublic {
		return true, nil
	}

	// Check shares
	shares, err := s.repo.GetSharesByTemplate(ctx, templateID)
	if err != nil {
		return false, err
	}

	for _, share := range shares {
		if share.SharedWithUserID == userID && share.CanUse {
			return true, nil
		}
	}

	return false, nil
}

// CanEditTemplate checks if user can edit template
func (s *ConfigTemplateServiceImpl) CanEditTemplate(ctx context.Context, templateID, userID uuid.UUID) (bool, error) {
	template, err := s.repo.GetTemplate(ctx, templateID)
	if err != nil {
		return false, err
	}

	// Creator can always edit
	if template.CreatedByUserID == userID {
		return true, nil
	}

	// Check shares for edit permission
	shares, err := s.repo.GetSharesByTemplate(ctx, templateID)
	if err != nil {
		return false, err
	}

	for _, share := range shares {
		if share.SharedWithUserID == userID && share.CanEdit {
			return true, nil
		}
	}

	return false, nil
}

// CanDeleteTemplate checks if user can delete template
func (s *ConfigTemplateServiceImpl) CanDeleteTemplate(ctx context.Context, templateID, userID uuid.UUID) (bool, error) {
	template, err := s.repo.GetTemplate(ctx, templateID)
	if err != nil {
		return false, err
	}

	// Creator can always delete
	if template.CreatedByUserID == userID {
		return true, nil
	}

	// Check shares for delete permission
	shares, err := s.repo.GetSharesByTemplate(ctx, templateID)
	if err != nil {
		return false, err
	}

	for _, share := range shares {
		if share.SharedWithUserID == userID && share.CanDelete {
			return true, nil
		}
	}

	return false, nil
}
