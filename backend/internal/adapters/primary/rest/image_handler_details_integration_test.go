package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	"github.com/srikarm/image-factory/internal/domain/image"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"github.com/srikarm/image-factory/internal/testutil"
	"go.uber.org/zap"
)

func openImageDetailsTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("IF_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set IF_TEST_POSTGRES_DSN to run image details integration tests")
	}
	testutil.RequireSafeTestDSN(t, dsn, "IF_TEST_POSTGRES_DSN")
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("failed connecting test postgres: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createImageDetailsTempTables(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TEMP TABLE catalog_image_metadata (
			image_id UUID PRIMARY KEY,
			docker_config_digest TEXT NULL,
			docker_manifest_digest TEXT NULL,
			total_layer_count INT NULL,
			compressed_size_bytes BIGINT NULL,
			uncompressed_size_bytes BIGINT NULL,
			packages_count INT NULL,
			vulnerabilities_high_count INT NULL,
			vulnerabilities_medium_count INT NULL,
			vulnerabilities_low_count INT NULL,
			entrypoint TEXT NULL,
			cmd TEXT NULL,
			env_vars TEXT NULL,
			working_dir TEXT NULL,
			labels TEXT NULL,
			last_scanned_at TIMESTAMPTZ NULL,
			scan_tool TEXT NULL,
			layers_evidence_status TEXT NULL,
			layers_evidence_build_id UUID NULL,
			layers_evidence_updated_at TIMESTAMPTZ NULL,
			sbom_evidence_status TEXT NULL,
			sbom_evidence_build_id UUID NULL,
			sbom_evidence_updated_at TIMESTAMPTZ NULL,
			vulnerability_evidence_status TEXT NULL,
			vulnerability_evidence_build_id UUID NULL,
			vulnerability_evidence_updated_at TIMESTAMPTZ NULL
		);
		CREATE TEMP TABLE catalog_image_layers (
			image_id UUID NOT NULL,
			layer_number INT NOT NULL,
			layer_digest TEXT NOT NULL,
			layer_size_bytes BIGINT NULL,
			media_type TEXT NULL,
			is_base_layer BOOLEAN NOT NULL DEFAULT false,
			base_image_name TEXT NULL,
			base_image_tag TEXT NULL,
			used_in_builds_count INT NULL,
			last_used_in_build_at TIMESTAMPTZ NULL
		);
		CREATE TEMP TABLE catalog_image_layer_evidence (
			image_id UUID NOT NULL,
			layer_digest TEXT NOT NULL,
			history_created_by TEXT NULL,
			source_command TEXT NULL,
			diff_id TEXT NULL
		);
		CREATE TEMP TABLE catalog_image_layer_packages (
			image_id UUID NOT NULL,
			layer_digest TEXT NOT NULL,
			package_name TEXT NOT NULL,
			package_version TEXT NULL,
			package_type TEXT NULL,
			package_path TEXT NULL,
			source_command TEXT NULL,
			known_vulnerabilities_count INT NULL,
			critical_vulnerabilities_count INT NULL
		);
		CREATE TEMP TABLE catalog_image_layer_vulnerabilities (
			image_id UUID NOT NULL,
			layer_digest TEXT NOT NULL,
			package_name TEXT NOT NULL,
			package_version TEXT NULL,
			cve_id TEXT NOT NULL,
			severity TEXT NULL,
			cvss_v3_score DOUBLE PRECISION NULL,
			reference_url TEXT NULL
		);
		CREATE TEMP TABLE catalog_image_sbom (
			id UUID PRIMARY KEY,
			image_id UUID NOT NULL UNIQUE,
			sbom_format TEXT NOT NULL,
			sbom_version TEXT NULL,
			generated_by_tool TEXT NULL,
			tool_version TEXT NULL,
			scan_timestamp TIMESTAMPTZ NULL,
			scan_duration_seconds INT NULL,
			status TEXT NULL
		);
		CREATE TEMP TABLE sbom_packages (
			image_sbom_id UUID NOT NULL,
			package_name TEXT NOT NULL,
			package_version TEXT NULL,
			package_type TEXT NULL,
			known_vulnerabilities_count INT NULL,
			critical_vulnerabilities_count INT NULL
		);
		CREATE TEMP TABLE package_vulnerabilities (
			package_name TEXT NOT NULL,
			package_version TEXT NULL,
			cve_id TEXT NOT NULL
		);
		CREATE TEMP TABLE cve_database (
			cve_id TEXT PRIMARY KEY,
			cvss_v3_severity TEXT NULL,
			cvss_v3_score DOUBLE PRECISION NULL,
			cve_description TEXT NULL,
			published_date DATE NULL,
			cve_references TEXT NULL
		);
		CREATE TEMP TABLE catalog_image_vulnerability_scans (
			id UUID PRIMARY KEY,
			image_id UUID NOT NULL,
			build_id UUID NULL,
			scan_tool TEXT NOT NULL,
			tool_version TEXT NULL,
			scan_status TEXT NOT NULL,
			started_at TIMESTAMPTZ NULL,
			completed_at TIMESTAMPTZ NULL,
			duration_seconds INT NULL,
			vulnerabilities_critical INT NULL,
			vulnerabilities_high INT NULL,
			vulnerabilities_medium INT NULL,
			vulnerabilities_low INT NULL,
			vulnerabilities_negligible INT NULL,
			vulnerabilities_unknown INT NULL,
			pass_fail_result TEXT NULL,
			compliance_check_passed BOOLEAN NULL,
			scan_report_location TEXT NULL,
			error_message TEXT NULL
		);
		CREATE TEMP TABLE external_image_imports (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL,
			request_type TEXT NOT NULL,
			internal_image_ref TEXT NULL,
			status TEXT NOT NULL,
			release_state TEXT NULL
		);
	`)
	if err != nil {
		t.Fatalf("failed creating temp image details tables: %v", err)
	}
}

func ensureImageDetailsTenant(t *testing.T, db *sqlx.DB, tenantID uuid.UUID) {
	t.Helper()
	code := strings.ReplaceAll(tenantID.String(), "-", "")[:8]
	_, err := db.Exec(`
		INSERT INTO tenants (id, tenant_code, name, slug, status)
		VALUES ($1, $2, $3, $4, 'active')
		ON CONFLICT (id) DO NOTHING
	`, tenantID, code, "test-tenant-"+code, "test-tenant-"+code)
	if err != nil {
		t.Fatalf("failed to seed tenant row: %v", err)
	}
}

func ensureImageDetailsUser(t *testing.T, db *sqlx.DB, userID uuid.UUID) {
	t.Helper()
	email := "itest-" + strings.ReplaceAll(userID.String(), "-", "")[:12] + "@example.com"
	_, err := db.Exec(`
		INSERT INTO users (id, email, status)
		VALUES ($1, $2, 'active')
		ON CONFLICT (id) DO NOTHING
	`, userID, email)
	if err != nil {
		t.Fatalf("failed to seed user row: %v", err)
	}
}

func TestImageHandlerGetImageDetails_ReflectsQuarantineImportEvidence(t *testing.T) {
	db := openImageDetailsTestDB(t)
	createImageDetailsTempTables(t, db)

	tenantID := uuid.New()
	userID := uuid.New()
	imageID := uuid.New()
	sbomID := uuid.New()
	buildID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	// Seed security/evidence tables that the details handler reads.
	_, err := db.Exec(`
		INSERT INTO catalog_image_metadata (
			image_id, docker_manifest_digest, packages_count, vulnerabilities_high_count, vulnerabilities_medium_count,
			scan_tool, sbom_evidence_status, sbom_evidence_updated_at,
			vulnerability_evidence_status, vulnerability_evidence_updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);
	`, imageID, "sha256:testdigest", 42, 2, 3, "trivy", "fresh", now, "fresh", now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("failed seeding image details evidence rows: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO catalog_image_sbom (
			id, image_id, sbom_format, generated_by_tool, tool_version, scan_timestamp, status
		) VALUES ($1, $2, 'spdx', 'syft', '1.0.0', $3, 'valid');
	`, sbomID, imageID, now)
	if err != nil {
		t.Fatalf("failed seeding image sbom row: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO sbom_packages (image_sbom_id, package_name, package_version, package_type, known_vulnerabilities_count, critical_vulnerabilities_count)
		VALUES ($1, 'openssl', '3.0.0', 'library', 1, 1);
	`, sbomID)
	if err != nil {
		t.Fatalf("failed seeding sbom package row: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO catalog_image_vulnerability_scans (
			id, image_id, build_id, scan_tool, scan_status, started_at, completed_at,
			vulnerabilities_critical, vulnerabilities_high, vulnerabilities_medium, pass_fail_result
		) VALUES ($1, $2, $3, 'trivy', 'completed', $4, $5, 1, 2, 3, 'FAIL');
	`, uuid.New(), imageID, buildID, now, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("failed seeding vulnerability scan row: %v", err)
	}

	repo := postgres.NewImageRepository(db, zap.NewNop())
	svc := image.NewService(repo, nil, nil, &imageHandlerPermissionChecker{}, &imageHandlerAuditLogger{}, zap.NewNop())
	handler := NewImageHandler(svc, nil, nil, zap.NewNop())
	handler.SetDB(db)
	ensureImageDetailsTenant(t, db, tenantID)
	ensureImageDetailsUser(t, db, userID)

	repositoryURL := "registry.local/published/team-a/base:latest"
	img, err := image.ReconstructImage(
		imageID,
		tenantID,
		"base",
		"base image",
		image.VisibilityTenant,
		image.StatusPublished,
		&repositoryURL,
		nil, nil, nil, nil, nil, nil,
		[]string{"latest"},
		map[string]interface{}{},
		nil,
		0,
		userID,
		userID,
		now,
		now,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to reconstruct image: %v", err)
	}
	if err := repo.Save(context.Background(), img); err != nil {
		t.Fatalf("failed to save catalog image row: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/"+imageID.String()+"/details", nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   userID,
		TenantID: tenantID,
	}))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", imageID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.GetImageDetails(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response struct {
		Data struct {
			Metadata struct {
				SBOMEvidenceStatus          string `json:"sbom_evidence_status"`
				VulnerabilityEvidenceStatus string `json:"vulnerability_evidence_status"`
				PackagesCount               int    `json:"packages_count"`
			} `json:"metadata"`
			SBOM *struct {
				Format          string `json:"format"`
				GeneratedByTool string `json:"generated_by_tool"`
			} `json:"sbom"`
			VulnerabilityScans []struct {
				ScanTool                string `json:"scan_tool"`
				ScanStatus              string `json:"scan_status"`
				VulnerabilitiesCritical int    `json:"vulnerabilities_critical"`
				VulnerabilitiesHigh     int    `json:"vulnerabilities_high"`
				VulnerabilitiesMedium   int    `json:"vulnerabilities_medium"`
				PassFailResult          string `json:"pass_fail_result"`
			} `json:"vulnerability_scans"`
			Stats struct {
				VulnerabilityScans int `json:"vulnerability_scan_count"`
			} `json:"stats"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Data.Metadata.SBOMEvidenceStatus != "fresh" || response.Data.Metadata.VulnerabilityEvidenceStatus != "fresh" {
		t.Fatalf("expected fresh evidence statuses, got sbom=%q vuln=%q", response.Data.Metadata.SBOMEvidenceStatus, response.Data.Metadata.VulnerabilityEvidenceStatus)
	}
	if response.Data.SBOM == nil || response.Data.SBOM.Format != "spdx" || response.Data.SBOM.GeneratedByTool != "syft" {
		t.Fatalf("expected SBOM evidence in details response, got %+v", response.Data.SBOM)
	}
	if len(response.Data.VulnerabilityScans) != 1 {
		t.Fatalf("expected one vulnerability scan row, got %d", len(response.Data.VulnerabilityScans))
	}
	scan := response.Data.VulnerabilityScans[0]
	if scan.ScanTool != "trivy" || scan.ScanStatus != "completed" || scan.PassFailResult != "FAIL" {
		t.Fatalf("unexpected vulnerability scan mapping: %+v", scan)
	}
	if scan.VulnerabilitiesCritical != 1 || scan.VulnerabilitiesHigh != 2 || scan.VulnerabilitiesMedium != 3 {
		t.Fatalf("unexpected vulnerability counts: %+v", scan)
	}
	if response.Data.Stats.VulnerabilityScans != 1 {
		t.Fatalf("expected stats.vulnerability_scan_count=1, got %d", response.Data.Stats.VulnerabilityScans)
	}
}

func TestImageHandlerGetImageDetails_HidesUnreleasedQuarantineImportImage(t *testing.T) {
	db := openImageDetailsTestDB(t)
	createImageDetailsTempTables(t, db)

	tenantID := uuid.New()
	userID := uuid.New()
	imageID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	repositoryURL := "registry.local/quarantine/team-a/import-1:latest"

	repo := postgres.NewImageRepository(db, zap.NewNop())
	svc := image.NewService(repo, nil, nil, &imageHandlerPermissionChecker{}, &imageHandlerAuditLogger{}, zap.NewNop())
	handler := NewImageHandler(svc, nil, nil, zap.NewNop())
	handler.SetDB(db)
	ensureImageDetailsTenant(t, db, tenantID)
	ensureImageDetailsUser(t, db, userID)

	img, err := image.ReconstructImage(
		imageID,
		tenantID,
		"import-1",
		"quarantine import image",
		image.VisibilityTenant,
		image.StatusPublished,
		&repositoryURL,
		nil, nil, nil, nil, nil, nil,
		[]string{"latest"},
		map[string]interface{}{},
		nil,
		0,
		userID,
		userID,
		now,
		now,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to reconstruct image: %v", err)
	}
	if err := repo.Save(context.Background(), img); err != nil {
		t.Fatalf("failed to save catalog image row: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO external_image_imports (id, tenant_id, request_type, internal_image_ref, status, release_state)
		VALUES ($1, $2, 'quarantine', $3, 'success', 'ready_for_release')
	`, uuid.New(), tenantID, repositoryURL)
	if err != nil {
		t.Fatalf("failed to seed unreleased import row: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/"+imageID.String()+"/details", nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   userID,
		TenantID: tenantID,
	}))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", imageID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.GetImageDetails(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unreleased quarantine image, got %d: %s", w.Code, w.Body.String())
	}
}

func TestImageHandlerSearchImages_ExcludesUnreleasedQuarantineImportImage(t *testing.T) {
	db := openImageDetailsTestDB(t)
	createImageDetailsTempTables(t, db)

	tenantID := uuid.New()
	userID := uuid.New()
	imageID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	repositoryURL := "registry.local/quarantine/team-a/import-2:latest"

	repo := postgres.NewImageRepository(db, zap.NewNop())
	svc := image.NewService(repo, nil, nil, &imageHandlerPermissionChecker{}, &imageHandlerAuditLogger{}, zap.NewNop())
	handler := NewImageHandler(svc, nil, nil, zap.NewNop())
	handler.SetDB(db)
	ensureImageDetailsTenant(t, db, tenantID)
	ensureImageDetailsUser(t, db, userID)

	img, err := image.ReconstructImage(
		imageID,
		tenantID,
		"import-2",
		"quarantine import image",
		image.VisibilityTenant,
		image.StatusPublished,
		&repositoryURL,
		nil, nil, nil, nil, nil, nil,
		[]string{"latest"},
		map[string]interface{}{},
		nil,
		0,
		userID,
		userID,
		now,
		now,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to reconstruct image: %v", err)
	}
	if err := repo.Save(context.Background(), img); err != nil {
		t.Fatalf("failed to save catalog image row: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO external_image_imports (id, tenant_id, request_type, internal_image_ref, status, release_state)
		VALUES ($1, $2, 'quarantine', $3, 'success', 'ready_for_release')
	`, uuid.New(), tenantID, repositoryURL)
	if err != nil {
		t.Fatalf("failed to seed unreleased import row: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/search?query=import", nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   userID,
		TenantID: tenantID,
	}))
	w := httptest.NewRecorder()

	handler.SearchImages(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response SearchImagesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(response.Images) != 0 {
		t.Fatalf("expected unreleased quarantine image to be excluded from search results, got %d rows", len(response.Images))
	}
}
