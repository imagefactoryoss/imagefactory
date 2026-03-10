package rest

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/image"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

type imageHandlerTestRepository struct {
	existsCalled bool
	saveCalled   bool
	findByIDImg  *image.Image
}

func (r *imageHandlerTestRepository) Save(ctx context.Context, img *image.Image) error {
	r.saveCalled = true
	return nil
}

func (r *imageHandlerTestRepository) FindByID(ctx context.Context, id uuid.UUID) (*image.Image, error) {
	return r.findByIDImg, nil
}

func (r *imageHandlerTestRepository) FindByTenantAndName(ctx context.Context, tenantID uuid.UUID, name string) (*image.Image, error) {
	return nil, nil
}

func (r *imageHandlerTestRepository) FindByVisibility(ctx context.Context, tenantID *uuid.UUID, includePublic bool) ([]*image.Image, error) {
	return nil, nil
}

func (r *imageHandlerTestRepository) Search(ctx context.Context, query string, tenantID *uuid.UUID, filters image.SearchFilters) ([]*image.Image, error) {
	return nil, nil
}

func (r *imageHandlerTestRepository) FindPopular(ctx context.Context, tenantID *uuid.UUID, limit int) ([]*image.Image, error) {
	return nil, nil
}

func (r *imageHandlerTestRepository) FindRecent(ctx context.Context, tenantID *uuid.UUID, limit int) ([]*image.Image, error) {
	return nil, nil
}

func (r *imageHandlerTestRepository) Update(ctx context.Context, img *image.Image) error {
	return nil
}

func (r *imageHandlerTestRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (r *imageHandlerTestRepository) ExistsByTenantAndName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	r.existsCalled = true
	return false, nil
}

func (r *imageHandlerTestRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return 0, nil
}

func (r *imageHandlerTestRepository) CountByVisibility(ctx context.Context, tenantID *uuid.UUID) (map[image.ImageVisibility]int, error) {
	return map[image.ImageVisibility]int{}, nil
}

func (r *imageHandlerTestRepository) IncrementPullCount(ctx context.Context, id uuid.UUID) error {
	return nil
}

type imageHandlerPermissionChecker struct{}

func (c *imageHandlerPermissionChecker) HasPermission(ctx context.Context, userID, tenantID *uuid.UUID, resource, action string) (bool, error) {
	return true, nil
}

type imageHandlerAuditLogger struct{}

func (l *imageHandlerAuditLogger) LogEvent(ctx context.Context, eventType, category, severity, resource, action, message string, details map[string]interface{}) error {
	return nil
}

type imageHandlerOperationCapabilityCheckerStub struct {
	entitled bool
	err      error
}

func (c *imageHandlerOperationCapabilityCheckerStub) IsOnDemandScanEntitled(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	if c.err != nil {
		return false, c.err
	}
	return c.entitled, nil
}

func TestImageHandlerCreateImage_ReturnsBadRequestForNilTenant(t *testing.T) {
	repo := &imageHandlerTestRepository{}
	svc := image.NewService(repo, nil, nil, &imageHandlerPermissionChecker{}, &imageHandlerAuditLogger{}, zap.NewNop())
	handler := NewImageHandler(svc, nil, nil, zap.NewNop())

	reqBody := map[string]string{
		"name":       "asw",
		"visibility": string(image.VisibilityPrivate),
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	authCtx := &middleware.AuthContext{
		UserID:         uuid.New(),
		TenantID:       uuid.Nil,
		UserTenants:    []uuid.UUID{},
		Email:          "admin@example.com",
		HasMultiTenant: true,
		IsSystemAdmin:  true,
	}
	req = req.WithContext(context.WithValue(req.Context(), "auth", authCtx))

	w := httptest.NewRecorder()
	handler.CreateImage(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if repo.existsCalled {
		t.Fatal("expected existence check to be skipped for nil tenant")
	}
	if repo.saveCalled {
		t.Fatal("expected save to be skipped for nil tenant")
	}
}

func TestImageHandlerCreateImage_ReturnsUnauthorizedWithoutAuthContext(t *testing.T) {
	repo := &imageHandlerTestRepository{}
	svc := image.NewService(repo, nil, nil, &imageHandlerPermissionChecker{}, &imageHandlerAuditLogger{}, zap.NewNop())
	handler := NewImageHandler(svc, nil, nil, zap.NewNop())

	reqBody := map[string]string{
		"name":       "asw",
		"visibility": string(image.VisibilityPrivate),
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateImage(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	if repo.existsCalled || repo.saveCalled {
		t.Fatal("expected no repository calls when auth context is missing")
	}
}

func TestCatalogImageMetadataModelToResponse_MapsSecurityEvidenceFields(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	model := catalogImageMetadataModel{
		SBOMEvidenceStatus:             sql.NullString{String: "fresh", Valid: true},
		SBOMEvidenceBuildID:            sql.NullString{String: uuid.New().String(), Valid: true},
		SBOMEvidenceUpdatedAt:          sql.NullTime{Time: now, Valid: true},
		VulnerabilityEvidenceStatus:    sql.NullString{String: "fresh", Valid: true},
		VulnerabilityEvidenceBuildID:   sql.NullString{String: uuid.New().String(), Valid: true},
		VulnerabilityEvidenceUpdatedAt: sql.NullTime{Time: now, Valid: true},
	}

	resp := catalogImageMetadataModelToResponse(model)
	if resp == nil {
		t.Fatal("expected non-nil metadata response")
	}
	if resp.SBOMEvidenceStatus != "fresh" {
		t.Fatalf("expected sbom evidence status fresh, got %q", resp.SBOMEvidenceStatus)
	}
	if resp.VulnerabilityEvidenceStatus != "fresh" {
		t.Fatalf("expected vulnerability evidence status fresh, got %q", resp.VulnerabilityEvidenceStatus)
	}
	if resp.SBOMEvidenceUpdatedAt == "" || resp.VulnerabilityEvidenceUpdatedAt == "" {
		t.Fatal("expected evidence updated timestamps to be set")
	}
}

func TestCatalogImageVulnerabilityScanModelsToResponse_MapsSecurityRows(t *testing.T) {
	scanID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	rows := []catalogImageVulnerabilityScanModel{
		{
			ID:                      scanID,
			ScanTool:                "trivy",
			ScanStatus:              "completed",
			StartedAt:               sql.NullTime{Time: now, Valid: true},
			CompletedAt:             sql.NullTime{Time: now.Add(2 * time.Minute), Valid: true},
			VulnerabilitiesCritical: sql.NullInt64{Int64: 1, Valid: true},
			VulnerabilitiesHigh:     sql.NullInt64{Int64: 2, Valid: true},
			VulnerabilitiesMedium:   sql.NullInt64{Int64: 3, Valid: true},
			PassFailResult:          sql.NullString{String: "FAIL", Valid: true},
		},
	}

	resp := catalogImageVulnerabilityScanModelsToResponse(rows)
	if len(resp) != 1 {
		t.Fatalf("expected one scan response, got %d", len(resp))
	}
	if resp[0].ID != scanID.String() {
		t.Fatalf("expected scan id %s, got %s", scanID.String(), resp[0].ID)
	}
	if resp[0].ScanTool != "trivy" || resp[0].ScanStatus != "completed" {
		t.Fatalf("unexpected scan tool/status mapping: %s %s", resp[0].ScanTool, resp[0].ScanStatus)
	}
	if resp[0].VulnerabilitiesCritical != 1 || resp[0].VulnerabilitiesHigh != 2 || resp[0].VulnerabilitiesMedium != 3 {
		t.Fatalf("unexpected vulnerability count mapping: %+v", resp[0])
	}
	if resp[0].PassFailResult != "FAIL" {
		t.Fatalf("expected pass_fail_result FAIL, got %q", resp[0].PassFailResult)
	}
}

func TestImageHandlerTriggerOnDemandScan_CapabilityDenied(t *testing.T) {
	repo := &imageHandlerTestRepository{
		findByIDImg: &image.Image{},
	}
	svc := image.NewService(repo, nil, nil, &imageHandlerPermissionChecker{}, &imageHandlerAuditLogger{}, zap.NewNop())
	handler := NewImageHandler(svc, nil, nil, zap.NewNop())
	handler.SetOperationCapabilityChecker(&imageHandlerOperationCapabilityCheckerStub{entitled: false})

	imageID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+imageID.String()+"/scan", nil)
	authCtx := &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
	}
	req = req.WithContext(context.WithValue(req.Context(), "auth", authCtx))
	req = withImageHandlerURLParam(req, "id", imageID.String())
	w := httptest.NewRecorder()

	handler.TriggerOnDemandScan(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestImageHandlerTriggerOnDemandScan_AcceptedWhenEntitled(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	imageID := uuid.New()
	img, err := image.NewImage(tenantID, "nginx", "desc", image.VisibilityPrivate, userID)
	if err != nil {
		t.Fatalf("failed to create image: %v", err)
	}
	repo := &imageHandlerTestRepository{
		findByIDImg: img,
	}
	svc := image.NewService(repo, nil, nil, &imageHandlerPermissionChecker{}, &imageHandlerAuditLogger{}, zap.NewNop())
	handler := NewImageHandler(svc, nil, nil, zap.NewNop())
	handler.SetOperationCapabilityChecker(&imageHandlerOperationCapabilityCheckerStub{entitled: true})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+imageID.String()+"/scan", nil)
	authCtx := &middleware.AuthContext{
		UserID:   userID,
		TenantID: tenantID,
	}
	req = req.WithContext(context.WithValue(req.Context(), "auth", authCtx))
	req = withImageHandlerURLParam(req, "id", imageID.String())
	w := httptest.NewRecorder()

	handler.TriggerOnDemandScan(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, w.Code)
	}
}

func withImageHandlerURLParam(req *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}
