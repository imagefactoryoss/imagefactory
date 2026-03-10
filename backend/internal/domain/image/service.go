package image

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PermissionChecker defines the interface for checking permissions
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, tenantID *uuid.UUID, resource, action string) (bool, error)
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogEvent(ctx context.Context, eventType, category, severity, resource, action, message string, details map[string]interface{}) error
}

// Service defines the business logic for image catalog management
type Service struct {
	repository        Repository
	versionRepository ImageVersionRepository
	tagRepository     ImageTagRepository
	permissionChecker PermissionChecker
	auditLogger       AuditLogger
	logger            *zap.Logger
}

// NewService creates a new image service
func NewService(
	repository Repository,
	versionRepository ImageVersionRepository,
	tagRepository ImageTagRepository,
	permissionChecker PermissionChecker,
	auditLogger AuditLogger,
	logger *zap.Logger,
) *Service {
	return &Service{
		repository:        repository,
		versionRepository: versionRepository,
		tagRepository:     tagRepository,
		permissionChecker: permissionChecker,
		auditLogger:       auditLogger,
		logger:            logger,
	}
}

// CreateImage creates a new image in the catalog
func (s *Service) CreateImage(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, name, description string, visibility ImageVisibility) (*Image, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}

	// Check permissions
	hasPermission, err := s.permissionChecker.HasPermission(ctx, &userID, &tenantID, "image", "create")
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return nil, ErrPermissionDenied
	}

	// Check if image already exists
	exists, err := s.repository.ExistsByTenantAndName(ctx, tenantID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to check image existence: %w", err)
	}
	if exists {
		return nil, ErrImageExists
	}

	// Create the image
	image, err := NewImage(tenantID, name, description, visibility, userID)
	if err != nil {
		return nil, err
	}

	// Save to repository
	if err := s.repository.Save(ctx, image); err != nil {
		return nil, fmt.Errorf("failed to save image: %w", err)
	}

	// Log audit event
	s.auditLogger.LogEvent(ctx, "image.created", "catalog", "info", "image", "create",
		fmt.Sprintf("Image %s created by user %s", name, userID), map[string]interface{}{
			"image_id":   image.ID(),
			"tenant_id":  tenantID,
			"name":       name,
			"visibility": visibility,
		})

	s.logger.Info("Image created",
		zap.String("image_id", image.ID().String()),
		zap.String("tenant_id", tenantID.String()),
		zap.String("name", name),
		zap.String("user_id", userID.String()))

	return image, nil
}

// GetImage retrieves an image by ID with visibility checks
func (s *Service) GetImage(ctx context.Context, imageID uuid.UUID, userID, tenantID *uuid.UUID) (*Image, error) {
	image, err := s.repository.FindByID(ctx, imageID)
	if err != nil {
		return nil, err
	}
	if image == nil {
		return nil, ErrImageNotFound
	}

	// Check visibility permissions
	if err := s.checkImageVisibility(ctx, image, userID, tenantID); err != nil {
		return nil, err
	}

	return image, nil
}

// UpdateImage updates an image's metadata
func (s *Service) UpdateImage(ctx context.Context, imageID uuid.UUID, userID uuid.UUID, tenantID uuid.UUID, updates ImageUpdates) error {
	image, err := s.repository.FindByID(ctx, imageID)
	if err != nil {
		return err
	}
	if image == nil {
		return ErrImageNotFound
	}

	// Enforce explicit tenant context ownership for all mutating operations.
	// Cross-tenant updates must be performed by selecting the owning tenant context.
	imageTenantID := image.TenantID()
	if imageTenantID != tenantID {
		return ErrPermissionDenied
	}

	hasPermission, err := s.permissionChecker.HasPermission(ctx, &userID, &tenantID, "image", "update")
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return ErrPermissionDenied
	}

	// Apply updates
	if updates.Description != nil {
		image.UpdateDescription(*updates.Description)
	}
	if updates.Visibility != nil {
		if err := image.UpdateVisibility(*updates.Visibility); err != nil {
			return err
		}
	}
	if updates.Status != nil {
		if err := image.UpdateStatus(*updates.Status); err != nil {
			return err
		}
	}
	if updates.Metadata != nil {
		image.UpdateMetadata(updates.Metadata)
	}
	if updates.TagsToAdd != nil {
		image.AddTags(updates.TagsToAdd)
	}
	if updates.TagsToRemove != nil {
		image.RemoveTags(updates.TagsToRemove)
	}

	image.SetUpdatedBy(userID)

	// Save changes
	if err := s.repository.Update(ctx, image); err != nil {
		return fmt.Errorf("failed to update image: %w", err)
	}

	// Update tags in separate repository
	if updates.TagsToAdd != nil || updates.TagsToRemove != nil {
		if err := s.updateImageTags(ctx, image.ID(), updates.TagsToAdd, updates.TagsToRemove); err != nil {
			s.logger.Error("Failed to update image tags", zap.Error(err))
			// Don't fail the whole operation for tag update failures
		}
	}

	// Log audit event
	s.auditLogger.LogEvent(ctx, "image.updated", "catalog", "info", "image", "update",
		fmt.Sprintf("Image %s updated by user %s", image.Name(), userID), map[string]interface{}{
			"image_id": imageID,
			"updates":  updates,
		})

	return nil
}

// DeleteImage deletes an image from the catalog
func (s *Service) DeleteImage(ctx context.Context, imageID uuid.UUID, userID uuid.UUID, tenantID uuid.UUID) error {
	image, err := s.repository.FindByID(ctx, imageID)
	if err != nil {
		return err
	}
	if image == nil {
		return ErrImageNotFound
	}

	// Enforce explicit tenant context ownership for all mutating operations.
	imageTenantID := image.TenantID()
	if imageTenantID != tenantID {
		return ErrPermissionDenied
	}

	// Check permissions in current tenant context.
	hasPermission, err := s.permissionChecker.HasPermission(ctx, &userID, &tenantID, "image", "delete")
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return ErrPermissionDenied
	}

	// Delete from repository
	if err := s.repository.Delete(ctx, imageID); err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	// Log audit event
	s.auditLogger.LogEvent(ctx, "image.deleted", "catalog", "warning", "image", "delete",
		fmt.Sprintf("Image %s deleted by user %s", image.Name(), userID), map[string]interface{}{
			"image_id":  imageID,
			"tenant_id": image.TenantID(),
		})

	return nil
}

// SearchImages performs a search with visibility filtering
func (s *Service) SearchImages(ctx context.Context, query string, userID, tenantID *uuid.UUID, filters SearchFilters) ([]*Image, error) {
	// Check read permissions
	hasPermission, err := s.permissionChecker.HasPermission(ctx, userID, tenantID, "image", "read")
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return nil, ErrPermissionDenied
	}

	images, err := s.repository.Search(ctx, query, tenantID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search images: %w", err)
	}

	// Filter results based on visibility
	var filteredImages []*Image
	for _, image := range images {
		if err := s.checkImageVisibility(ctx, image, userID, tenantID); err == nil {
			filteredImages = append(filteredImages, image)
		}
	}

	return filteredImages, nil
}

// GetPopularImages returns popular images
func (s *Service) GetPopularImages(ctx context.Context, userID, tenantID *uuid.UUID, limit int) ([]*Image, error) {
	hasPermission, err := s.permissionChecker.HasPermission(ctx, userID, tenantID, "image", "read")
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return nil, ErrPermissionDenied
	}

	images, err := s.repository.FindPopular(ctx, tenantID, limit)
	if err != nil {
		return nil, err
	}

	// Filter by visibility
	var filteredImages []*Image
	for _, image := range images {
		if err := s.checkImageVisibility(ctx, image, userID, tenantID); err == nil {
			filteredImages = append(filteredImages, image)
		}
	}

	return filteredImages, nil
}

// GetRecentImages returns recently added images
func (s *Service) GetRecentImages(ctx context.Context, userID, tenantID *uuid.UUID, limit int) ([]*Image, error) {
	hasPermission, err := s.permissionChecker.HasPermission(ctx, userID, tenantID, "image", "read")
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return nil, ErrPermissionDenied
	}

	images, err := s.repository.FindRecent(ctx, tenantID, limit)
	if err != nil {
		return nil, err
	}

	// Filter by visibility
	var filteredImages []*Image
	for _, image := range images {
		if err := s.checkImageVisibility(ctx, image, userID, tenantID); err == nil {
			filteredImages = append(filteredImages, image)
		}
	}

	return filteredImages, nil
}

// IncrementPullCount increments the pull count for an image
func (s *Service) IncrementPullCount(ctx context.Context, imageID uuid.UUID) error {
	return s.repository.IncrementPullCount(ctx, imageID)
}

// ImageUpdates represents the fields that can be updated on an image
type ImageUpdates struct {
	Description  *string                `json:"description,omitempty"`
	Visibility   *ImageVisibility       `json:"visibility,omitempty"`
	Status       *ImageStatus           `json:"status,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	TagsToAdd    []string               `json:"tags_to_add,omitempty"`
	TagsToRemove []string               `json:"tags_to_remove,omitempty"`
}

// checkImageVisibility checks if a user can access an image based on visibility rules
func (s *Service) checkImageVisibility(ctx context.Context, image *Image, userID, tenantID *uuid.UUID) error {
	// Nil tenant context is invalid for tenant-scoped image visibility checks.
	if tenantID == nil {
		return ErrPermissionDenied
	}

	// System admins can see all images
	if userID != nil {
		hasPermission, err := s.permissionChecker.HasPermission(ctx, userID, nil, "image", "manage")
		if err != nil {
			return fmt.Errorf("failed to check admin permissions: %w", err)
		}
		if hasPermission {
			return nil
		}
	}

	switch image.Visibility() {
	case VisibilityPublic:
		// Public images are visible to all authenticated users
		return nil
	case VisibilityTenant:
		// Tenant images are visible to users in the same tenant
		if tenantID != nil && image.TenantID() == *tenantID {
			return nil
		}
		return ErrPermissionDenied
	case VisibilityPrivate:
		// Until per-user/group ACLs are implemented, keep private images tenant-scoped.
		if tenantID != nil && image.TenantID() == *tenantID {
			return nil
		}
		return ErrPermissionDenied
	default:
		return ErrInvalidVisibility
	}
}

// updateImageTags updates the tags for an image
func (s *Service) updateImageTags(ctx context.Context, imageID uuid.UUID, tagsToAdd, tagsToRemove []string) error {
	// Remove tags
	for _, tag := range tagsToRemove {
		// Find and delete existing tag
		tags, err := s.tagRepository.FindByImageID(ctx, imageID)
		if err != nil {
			return err
		}
		for _, existingTag := range tags {
			if existingTag.Tag == tag {
				if err := s.tagRepository.Delete(ctx, existingTag.ID); err != nil {
					return err
				}
				break
			}
		}
	}

	// Add new tags
	for _, tag := range tagsToAdd {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			continue
		}

		imageTag := &ImageTag{
			ID:        uuid.New(),
			ImageID:   imageID,
			Tag:       tag,
			Category:  "user",
			CreatedBy: uuid.Nil, // TODO: Pass userID to this method
			CreatedAt: time.Now(),
		}

		if err := s.tagRepository.Save(ctx, imageTag); err != nil {
			return err
		}
	}

	return nil
}
