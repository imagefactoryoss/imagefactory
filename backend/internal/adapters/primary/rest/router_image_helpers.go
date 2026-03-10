package rest

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/image"
	"github.com/srikarm/image-factory/internal/domain/rbac"
	"github.com/srikarm/image-factory/internal/domain/tenant"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
)

// rbacPermissionChecker adapts rbac.Service to image.PermissionChecker interface.
type rbacPermissionChecker struct {
	rbacService *rbac.Service
	tenantRepo  tenant.Repository
}

func (r *rbacPermissionChecker) HasPermission(ctx context.Context, userID, tenantID *uuid.UUID, resource, action string) (bool, error) {
	if userID == nil {
		return false, nil
	}

	hasPermission, err := r.rbacService.CheckUserPermission(ctx, *userID, resource, action)
	if err != nil {
		return false, err
	}
	if !hasPermission {
		return false, nil
	}

	if tenantID != nil && *tenantID != uuid.Nil {
		tenantEntity, err := r.tenantRepo.FindByID(ctx, *tenantID)
		if err != nil {
			return false, fmt.Errorf("invalid tenant: %w", err)
		}
		if tenantEntity == nil {
			return false, fmt.Errorf("invalid tenant: tenant not found")
		}
	}

	return true, nil
}

// auditLoggerAdapter adapts audit.Service to image.AuditLogger interface.
type auditLoggerAdapter struct {
	auditService *audit.Service
}

func (a *auditLoggerAdapter) LogEvent(ctx context.Context, eventType, category, severity, resource, action, message string, details map[string]interface{}) error {
	var tenantID uuid.UUID
	if details != nil {
		if tid, ok := details["tenant_id"].(uuid.UUID); ok {
			tenantID = tid
		}
	}

	var auditEventType audit.AuditEventType
	switch eventType {
	case "user_create":
		auditEventType = audit.AuditEventUserCreate
	default:
		auditEventType = audit.AuditEventUserCreate
	}

	return a.auditService.LogSystemAction(ctx, tenantID, auditEventType, resource, action, message, details)
}

// Stub implementations for image service dependencies.
type stubPermissionChecker struct{}

func (s *stubPermissionChecker) HasPermission(ctx context.Context, userID, tenantID *uuid.UUID, resource, action string) (bool, error) {
	return true, nil
}

type stubAuditLogger struct{}

func (s *stubAuditLogger) LogEvent(ctx context.Context, eventType, category, severity, resource, action, message string, details map[string]interface{}) error {
	return nil
}

type stubImageVersionRepository struct{}

func (s *stubImageVersionRepository) Save(ctx context.Context, version *image.ImageVersion) error {
	return nil
}
func (s *stubImageVersionRepository) FindByID(ctx context.Context, id uuid.UUID) (*image.ImageVersion, error) {
	return nil, nil
}
func (s *stubImageVersionRepository) FindByImageID(ctx context.Context, imageID uuid.UUID) ([]*image.ImageVersion, error) {
	return nil, nil
}
func (s *stubImageVersionRepository) FindByImageAndVersion(ctx context.Context, imageID uuid.UUID, version string) (*image.ImageVersion, error) {
	return nil, nil
}
func (s *stubImageVersionRepository) FindLatestByImageID(ctx context.Context, imageID uuid.UUID) (*image.ImageVersion, error) {
	return nil, nil
}
func (s *stubImageVersionRepository) Update(ctx context.Context, version *image.ImageVersion) error {
	return nil
}
func (s *stubImageVersionRepository) Delete(ctx context.Context, id uuid.UUID) error { return nil }

type stubImageTagRepository struct{}

func (s *stubImageTagRepository) Save(ctx context.Context, tag *image.ImageTag) error { return nil }
func (s *stubImageTagRepository) FindByID(ctx context.Context, id uuid.UUID) (*image.ImageTag, error) {
	return nil, nil
}
func (s *stubImageTagRepository) FindByImageID(ctx context.Context, imageID uuid.UUID) ([]*image.ImageTag, error) {
	return nil, nil
}
func (s *stubImageTagRepository) FindByTag(ctx context.Context, tag string) ([]uuid.UUID, error) {
	return nil, nil
}
func (s *stubImageTagRepository) DeleteByImageID(ctx context.Context, imageID uuid.UUID) error {
	return nil
}
func (s *stubImageTagRepository) Delete(ctx context.Context, id uuid.UUID) error { return nil }
