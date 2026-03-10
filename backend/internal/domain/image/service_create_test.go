package image

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type testImageRepository struct {
	existsCalled bool
	saveCalled   bool
}

func (r *testImageRepository) Save(ctx context.Context, image *Image) error {
	r.saveCalled = true
	return nil
}

func (r *testImageRepository) FindByID(ctx context.Context, id uuid.UUID) (*Image, error) {
	return nil, nil
}

func (r *testImageRepository) FindByTenantAndName(ctx context.Context, tenantID uuid.UUID, name string) (*Image, error) {
	return nil, nil
}

func (r *testImageRepository) FindByVisibility(ctx context.Context, tenantID *uuid.UUID, includePublic bool) ([]*Image, error) {
	return nil, nil
}

func (r *testImageRepository) Search(ctx context.Context, query string, tenantID *uuid.UUID, filters SearchFilters) ([]*Image, error) {
	return nil, nil
}

func (r *testImageRepository) FindPopular(ctx context.Context, tenantID *uuid.UUID, limit int) ([]*Image, error) {
	return nil, nil
}

func (r *testImageRepository) FindRecent(ctx context.Context, tenantID *uuid.UUID, limit int) ([]*Image, error) {
	return nil, nil
}

func (r *testImageRepository) Update(ctx context.Context, image *Image) error {
	return nil
}

func (r *testImageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (r *testImageRepository) ExistsByTenantAndName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	r.existsCalled = true
	return false, nil
}

func (r *testImageRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return 0, nil
}

func (r *testImageRepository) CountByVisibility(ctx context.Context, tenantID *uuid.UUID) (map[ImageVisibility]int, error) {
	return map[ImageVisibility]int{}, nil
}

func (r *testImageRepository) IncrementPullCount(ctx context.Context, id uuid.UUID) error {
	return nil
}

type testPermissionChecker struct{}

func (c *testPermissionChecker) HasPermission(ctx context.Context, userID, tenantID *uuid.UUID, resource, action string) (bool, error) {
	return true, nil
}

type testAuditLogger struct{}

func (l *testAuditLogger) LogEvent(ctx context.Context, eventType, category, severity, resource, action, message string, details map[string]interface{}) error {
	return nil
}

func TestCreateImage_ReturnsInvalidTenantIDForNilTenant(t *testing.T) {
	repo := &testImageRepository{}
	svc := NewService(repo, nil, nil, &testPermissionChecker{}, &testAuditLogger{}, zap.NewNop())

	_, err := svc.CreateImage(context.Background(), uuid.Nil, uuid.New(), "test-image", "", VisibilityPrivate)
	if err != ErrInvalidTenantID {
		t.Fatalf("expected ErrInvalidTenantID, got %v", err)
	}

	if repo.existsCalled {
		t.Fatalf("expected repository existence check to be skipped for invalid tenant")
	}
	if repo.saveCalled {
		t.Fatalf("expected save to be skipped for invalid tenant")
	}
}

func TestNewImage_ReturnsInvalidTenantIDForNilTenant(t *testing.T) {
	_, err := NewImage(uuid.Nil, "test-image", "", VisibilityPrivate, uuid.New())
	if err != ErrInvalidTenantID {
		t.Fatalf("expected ErrInvalidTenantID, got %v", err)
	}
}

func TestNewImage_AcceptsValidTenantID(t *testing.T) {
	img, err := NewImage(uuid.New(), "test-image", "description", VisibilityPrivate, uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if img == nil {
		t.Fatal("expected image to be created")
	}
	if img.CreatedAt().IsZero() || img.UpdatedAt().IsZero() {
		t.Fatal("expected image timestamps to be set")
	}
	if img.UpdatedAt().Before(img.CreatedAt().Add(-1 * time.Second)) {
		t.Fatal("unexpected timestamp values")
	}
}
