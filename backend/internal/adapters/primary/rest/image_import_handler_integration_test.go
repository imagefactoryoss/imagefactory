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
	"github.com/srikarm/image-factory/internal/domain/imageimport"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"github.com/srikarm/image-factory/internal/testutil"
	"go.uber.org/zap"
)

func openImageImportHandlerIntegrationDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("IF_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set IF_TEST_POSTGRES_DSN to run image import handler integration tests")
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

func createImageImportHandlerTempTables(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TEMP TABLE external_image_imports (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL,
			requested_by_user_id UUID NOT NULL,
			request_type TEXT NOT NULL,
			sor_record_id TEXT NOT NULL,
			source_registry TEXT NOT NULL,
			source_image_ref TEXT NOT NULL,
			registry_auth_id UUID NULL,
			status TEXT NOT NULL,
			error_message TEXT NULL,
			internal_image_ref TEXT NULL,
			pipeline_run_name TEXT NULL,
			pipeline_namespace TEXT NULL,
			policy_decision TEXT NULL,
			policy_reasons_json TEXT NULL,
			policy_snapshot_json TEXT NULL,
			scan_summary_json TEXT NULL,
			sbom_summary_json TEXT NULL,
			sbom_evidence_json TEXT NULL,
			source_image_digest TEXT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		t.Fatalf("failed creating temp external_image_imports table: %v", err)
	}
}

func TestImageImportHandlerIntegration_GetImportRequest_NormalizesTerminalEvidenceFallbacks(t *testing.T) {
	db := openImageImportHandlerIntegrationDB(t)
	createImageImportHandlerTempTables(t, db)

	tenantID := uuid.New()
	userID := uuid.New()
	importID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	_, err := db.Exec(`
		INSERT INTO external_image_imports (
			id, tenant_id, requested_by_user_id, request_type, sor_record_id,
			source_registry, source_image_ref, status, error_message, created_at, updated_at
		) VALUES (
			$1, $2, $3, 'quarantine', 'APP-9', 'ghcr.io', 'ghcr.io/acme/app:9.0.0', 'failed', 'pipeline execution failed', $4, $4
		)
	`, importID, tenantID, userID, now)
	if err != nil {
		t.Fatalf("failed seeding external_image_imports: %v", err)
	}

	repo := postgres.NewImageImportRepository(db, zap.NewNop())
	svc := imageimport.NewService(repo, nil, nil, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/import-requests/"+importID.String(), nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   userID,
		TenantID: tenantID,
	}))
	req = withImageImportRouteParam(req, "id", importID.String())
	w := httptest.NewRecorder()

	handler.GetImportRequest(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var decoded struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed decoding response: %v", err)
	}
	data := decoded.Data
	if data["policy_decision"] == "" || data["policy_reasons_json"] == "" || data["policy_snapshot_json"] == "" {
		t.Fatalf("expected non-empty policy fallbacks, got %+v", data)
	}
	if data["scan_summary_json"] == "" || data["sbom_summary_json"] == "" || data["sbom_evidence_json"] == "" || data["source_image_digest"] == "" {
		t.Fatalf("expected non-empty evidence fallbacks, got %+v", data)
	}
}

func TestImageImportHandlerIntegration_ListImportRequests_NormalizesTerminalEvidenceFallbacks(t *testing.T) {
	db := openImageImportHandlerIntegrationDB(t)
	createImageImportHandlerTempTables(t, db)

	tenantID := uuid.New()
	userID := uuid.New()
	importID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	_, err := db.Exec(`
		INSERT INTO external_image_imports (
			id, tenant_id, requested_by_user_id, request_type, sor_record_id,
			source_registry, source_image_ref, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, 'quarantine', 'APP-10', 'ghcr.io', 'ghcr.io/acme/app:10.0.0', 'success', $4, $4
		)
	`, importID, tenantID, userID, now)
	if err != nil {
		t.Fatalf("failed seeding external_image_imports: %v", err)
	}

	repo := postgres.NewImageImportRepository(db, zap.NewNop())
	svc := imageimport.NewService(repo, nil, nil, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/import-requests", nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   userID,
		TenantID: tenantID,
	}))
	w := httptest.NewRecorder()

	handler.ListImportRequests(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var decoded struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed decoding response: %v", err)
	}
	if len(decoded.Data) != 1 {
		t.Fatalf("expected one row, got %d", len(decoded.Data))
	}
	row := decoded.Data[0]
	if row["policy_decision"] == "" || row["scan_summary_json"] == "" || row["sbom_summary_json"] == "" || row["sbom_evidence_json"] == "" || row["source_image_digest"] == "" {
		t.Fatalf("expected non-empty terminal evidence fallbacks in list response, got %+v", row)
	}
}

func withImageImportRouteParam(req *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}
