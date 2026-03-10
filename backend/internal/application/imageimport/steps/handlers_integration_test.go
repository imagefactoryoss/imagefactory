package steps

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	domainimageimport "github.com/srikarm/image-factory/internal/domain/imageimport"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"github.com/srikarm/image-factory/internal/testutil"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func openImportMonitorIntegrationDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("IF_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set IF_TEST_POSTGRES_DSN to run import monitor integration tests")
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

func createImportMonitorTempTables(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TEMP TABLE external_image_imports (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL,
			requested_by_user_id UUID NOT NULL,
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
		CREATE TEMP TABLE catalog_images (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL,
			name TEXT NOT NULL,
			repository_url TEXT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TEMP TABLE catalog_image_sbom (
			id UUID PRIMARY KEY,
			image_id UUID NOT NULL UNIQUE,
			sbom_format TEXT NOT NULL,
			sbom_version TEXT NULL,
			sbom_content TEXT NOT NULL,
			generated_by_tool TEXT NULL,
			tool_version TEXT NULL,
			sbom_checksum TEXT NULL,
			scan_timestamp TIMESTAMPTZ NULL,
			scan_duration_seconds INT NULL,
			status TEXT NULL
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
			scan_report_json TEXT NULL,
			error_message TEXT NULL
		);
		CREATE TEMP TABLE catalog_image_metadata (
			id UUID PRIMARY KEY,
			image_id UUID NOT NULL UNIQUE,
			docker_manifest_digest TEXT NULL,
			packages_count INT NULL,
			vulnerabilities_high_count INT NULL,
			vulnerabilities_medium_count INT NULL,
			vulnerabilities_low_count INT NULL,
			layers_evidence_status TEXT NULL,
			sbom_evidence_status TEXT NULL,
			vulnerability_evidence_status TEXT NULL,
			sbom_evidence_updated_at TIMESTAMPTZ NULL,
			vulnerability_evidence_updated_at TIMESTAMPTZ NULL
		);
	`)
	if err != nil {
		t.Fatalf("failed creating temp import monitor tables: %v", err)
	}
}

type importMonitorRunReaderStub struct {
	run *tektonv1.PipelineRun
}

func (s *importMonitorRunReaderStub) GetPipelineRun(ctx context.Context, req *domainimageimport.ImportRequest) (*tektonv1.PipelineRun, error) {
	return s.run, nil
}

type importMonitorRunSequenceReaderStub struct {
	runs []*tektonv1.PipelineRun
	i    int
}

func (s *importMonitorRunSequenceReaderStub) GetPipelineRun(ctx context.Context, req *domainimageimport.ImportRequest) (*tektonv1.PipelineRun, error) {
	if len(s.runs) == 0 {
		return nil, nil
	}
	if s.i >= len(s.runs) {
		return s.runs[len(s.runs)-1], nil
	}
	run := s.runs[s.i]
	s.i++
	return run, nil
}

func TestImportMonitorHandler_PipelineInProgressThenQuarantined_WithPersistedPipelineRefs(t *testing.T) {
	db := openImportMonitorIntegrationDB(t)
	createImportMonitorTempTables(t, db)

	tenantID := uuid.New()
	importID := uuid.New()
	now := time.Now().UTC()
	internalRef := "registry.local/published/team-a/base:latest"
	pipelineRunName := "pr-running-1"
	pipelineNamespace := "ns-a"

	_, err := db.Exec(`
		INSERT INTO external_image_imports (
			id, tenant_id, requested_by_user_id, sor_record_id, source_registry, source_image_ref,
			status, internal_image_ref, pipeline_run_name, pipeline_namespace, created_at, updated_at
		) VALUES ($1, $2, $3, 'APP-1', 'ghcr.io', 'ghcr.io/acme/base:2.0.0', 'importing', $4, $5, $6, $7, $7)
	`, importID, tenantID, uuid.New(), internalRef, pipelineRunName, pipelineNamespace, now)
	if err != nil {
		t.Fatalf("failed seeding external_image_imports: %v", err)
	}

	// Seed catalog image up-front so terminal pass can sync evidence immediately.
	imageID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO catalog_images (id, tenant_id, name, repository_url, updated_at)
		VALUES ($1, $2, 'base', 'registry.local/published/team-a/base', NOW())
	`, imageID, tenantID)
	if err != nil {
		t.Fatalf("failed seeding catalog_images: %v", err)
	}

	reader := &importMonitorRunSequenceReaderStub{
		runs: []*tektonv1.PipelineRun{
			{
				Status: tektonv1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{Type: apis.ConditionSucceeded, Status: corev1.ConditionUnknown},
						},
					},
				},
			},
			{
				Status: tektonv1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{Type: apis.ConditionSucceeded, Status: corev1.ConditionTrue},
						},
					},
					PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
						Results: []tektonv1.PipelineRunResult{
							{Name: "decision", Value: tektonv1.ResultValue{StringVal: "quarantine"}},
							{Name: "scan-summary", Value: tektonv1.ResultValue{StringVal: `{"tool":"trivy","vulnerabilities":{"critical":2,"high":1,"medium":0,"low":0}}`}},
							{Name: "sbom-evidence", Value: tektonv1.ResultValue{StringVal: `{"format":"spdx","generator":"syft","version":"1.2.1"}`}},
							{Name: "source-image-digest", Value: tektonv1.ResultValue{StringVal: "sha256:def456"}},
						},
					},
				},
			},
		},
	}

	repo := postgres.NewImageImportRepository(db, zap.NewNop())
	handler := NewImportMonitorHandler(repo, reader, zap.NewNop())
	step := &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": importID.String(),
		},
	}

	first, err := handler.Execute(context.Background(), step)
	if err != nil {
		t.Fatalf("expected no execute error on first pass, got %v", err)
	}
	if first.Status != domainworkflow.StepStatusBlocked {
		t.Fatalf("expected blocked status on first pass, got %s", first.Status)
	}
	if !strings.Contains(first.Error, "still in progress") {
		t.Fatalf("expected in-progress blocked error, got %q", first.Error)
	}

	var stateAfterFirst struct {
		Status            string `db:"status"`
		PipelineRunName   string `db:"pipeline_run_name"`
		PipelineNamespace string `db:"pipeline_namespace"`
	}
	if err := db.Get(&stateAfterFirst, `
		SELECT status, pipeline_run_name, pipeline_namespace
		FROM external_image_imports
		WHERE id = $1
	`, importID); err != nil {
		t.Fatalf("failed reading import status after first pass: %v", err)
	}
	if stateAfterFirst.Status != "importing" {
		t.Fatalf("expected status importing after first pass, got %s", stateAfterFirst.Status)
	}
	if stateAfterFirst.PipelineRunName != pipelineRunName || stateAfterFirst.PipelineNamespace != pipelineNamespace {
		t.Fatalf("expected pipeline refs to remain persisted after blocked pass, got name=%q namespace=%q", stateAfterFirst.PipelineRunName, stateAfterFirst.PipelineNamespace)
	}

	second, err := handler.Execute(context.Background(), step)
	if err != nil {
		t.Fatalf("expected no execute error on second pass, got %v", err)
	}
	if second.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded status on second pass, got %s", second.Status)
	}

	var finalState struct {
		Status            string `db:"status"`
		PipelineRunName   string `db:"pipeline_run_name"`
		PipelineNamespace string `db:"pipeline_namespace"`
	}
	if err := db.Get(&finalState, `
		SELECT status, pipeline_run_name, pipeline_namespace
		FROM external_image_imports
		WHERE id = $1
	`, importID); err != nil {
		t.Fatalf("failed reading final import status: %v", err)
	}
	if finalState.Status != "quarantined" {
		t.Fatalf("expected final status quarantined, got %s", finalState.Status)
	}
	if finalState.PipelineRunName != pipelineRunName || finalState.PipelineNamespace != pipelineNamespace {
		t.Fatalf("expected pipeline refs to remain persisted at terminal state, got name=%q namespace=%q", finalState.PipelineRunName, finalState.PipelineNamespace)
	}
}

func TestImportMonitorHandler_PipelineInProgressThenFailed_WithPersistedPipelineRefs(t *testing.T) {
	db := openImportMonitorIntegrationDB(t)
	createImportMonitorTempTables(t, db)

	tenantID := uuid.New()
	importID := uuid.New()
	now := time.Now().UTC()
	internalRef := "registry.local/published/team-a/base:latest"
	pipelineRunName := "pr-running-2"
	pipelineNamespace := "ns-a"

	_, err := db.Exec(`
		INSERT INTO external_image_imports (
			id, tenant_id, requested_by_user_id, sor_record_id, source_registry, source_image_ref,
			status, internal_image_ref, pipeline_run_name, pipeline_namespace, created_at, updated_at
		) VALUES ($1, $2, $3, 'APP-2', 'ghcr.io', 'ghcr.io/acme/base:2.1.0', 'importing', $4, $5, $6, $7, $7)
	`, importID, tenantID, uuid.New(), internalRef, pipelineRunName, pipelineNamespace, now)
	if err != nil {
		t.Fatalf("failed seeding external_image_imports: %v", err)
	}

	reader := &importMonitorRunSequenceReaderStub{
		runs: []*tektonv1.PipelineRun{
			{
				Status: tektonv1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{Type: apis.ConditionSucceeded, Status: corev1.ConditionUnknown},
						},
					},
				},
			},
			{
				Status: tektonv1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:    apis.ConditionSucceeded,
								Status:  corev1.ConditionFalse,
								Reason:  "Failed",
								Message: "image pull failed",
							},
						},
					},
				},
			},
		},
	}

	repo := postgres.NewImageImportRepository(db, zap.NewNop())
	handler := NewImportMonitorHandler(repo, reader, zap.NewNop())
	step := &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": importID.String(),
		},
	}

	first, err := handler.Execute(context.Background(), step)
	if err != nil {
		t.Fatalf("expected no execute error on first pass, got %v", err)
	}
	if first.Status != domainworkflow.StepStatusBlocked {
		t.Fatalf("expected blocked status on first pass, got %s", first.Status)
	}

	second, err := handler.Execute(context.Background(), step)
	if err != nil {
		t.Fatalf("expected no execute error on second pass, got %v", err)
	}
	if second.Status != domainworkflow.StepStatusFailed {
		t.Fatalf("expected failed status on second pass, got %s", second.Status)
	}
	if !strings.Contains(second.Error, "image pull failed") {
		t.Fatalf("expected failed message propagated, got %q", second.Error)
	}

	var finalState struct {
		Status            string `db:"status"`
		ErrorMessage      string `db:"error_message"`
		PipelineRunName   string `db:"pipeline_run_name"`
		PipelineNamespace string `db:"pipeline_namespace"`
	}
	if err := db.Get(&finalState, `
		SELECT status, error_message, pipeline_run_name, pipeline_namespace
		FROM external_image_imports
		WHERE id = $1
	`, importID); err != nil {
		t.Fatalf("failed reading final import status: %v", err)
	}
	if finalState.Status != "failed" {
		t.Fatalf("expected final status failed, got %s", finalState.Status)
	}
	if !strings.Contains(strings.ToLower(finalState.ErrorMessage), "image pull failed") {
		t.Fatalf("expected persisted error message, got %q", finalState.ErrorMessage)
	}
	if finalState.PipelineRunName != pipelineRunName || finalState.PipelineNamespace != pipelineNamespace {
		t.Fatalf("expected pipeline refs to remain persisted at failed terminal state, got name=%q namespace=%q", finalState.PipelineRunName, finalState.PipelineNamespace)
	}
}

func TestImportMonitorHandler_BlocksWhenPipelineRefsMissing(t *testing.T) {
	db := openImportMonitorIntegrationDB(t)
	createImportMonitorTempTables(t, db)

	tenantID := uuid.New()
	importID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(`
		INSERT INTO external_image_imports (
			id, tenant_id, requested_by_user_id, sor_record_id, source_registry, source_image_ref,
			status, internal_image_ref, pipeline_run_name, pipeline_namespace, created_at, updated_at
		) VALUES ($1, $2, $3, 'APP-3', 'ghcr.io', 'ghcr.io/acme/base:3.0.0', 'importing', $4, '', '', $5, $5)
	`, importID, tenantID, uuid.New(), "registry.local/published/team-a/base:latest", now)
	if err != nil {
		t.Fatalf("failed seeding external_image_imports: %v", err)
	}

	repo := postgres.NewImageImportRepository(db, zap.NewNop())
	handler := NewImportMonitorHandler(repo, &importMonitorRunReaderStub{}, zap.NewNop())
	step := &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": importID.String(),
		},
	}

	result, execErr := handler.Execute(context.Background(), step)
	if execErr != nil {
		t.Fatalf("expected no execute error, got %v", execErr)
	}
	if result.Status != domainworkflow.StepStatusBlocked {
		t.Fatalf("expected blocked status when pipeline refs are missing, got %s", result.Status)
	}
	if !strings.Contains(result.Error, "pipeline reference is not available yet") {
		t.Fatalf("expected missing pipeline ref error, got %q", result.Error)
	}

	var status string
	if err := db.Get(&status, `SELECT status FROM external_image_imports WHERE id = $1`, importID); err != nil {
		t.Fatalf("failed reading import status after blocked pass: %v", err)
	}
	if status != "importing" {
		t.Fatalf("expected status importing after missing-ref blocked pass, got %s", status)
	}
}

func TestImportMonitorHandler_DeferredCatalogSyncEventuallySucceeds(t *testing.T) {
	db := openImportMonitorIntegrationDB(t)
	createImportMonitorTempTables(t, db)

	tenantID := uuid.New()
	importID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(`
		INSERT INTO external_image_imports (
			id, tenant_id, requested_by_user_id, sor_record_id, source_registry, source_image_ref,
			status, internal_image_ref, pipeline_run_name, pipeline_namespace, created_at, updated_at
		) VALUES ($1, $2, $3, 'APP-1', 'ghcr.io', 'ghcr.io/acme/base:1.0.0', 'importing', $4, 'pr-1', 'ns-a', $5, $5)
	`, importID, tenantID, uuid.New(), "registry.local/published/team-a/base:latest", now)
	if err != nil {
		t.Fatalf("failed seeding external_image_imports: %v", err)
	}

	reader := &importMonitorRunReaderStub{
		run: &tektonv1.PipelineRun{
			Status: tektonv1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{Type: apis.ConditionSucceeded, Status: corev1.ConditionTrue},
					},
				},
				PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
					Results: []tektonv1.PipelineRunResult{
						{Name: "decision", Value: tektonv1.ResultValue{StringVal: "pass"}},
						{Name: "scan-summary", Value: tektonv1.ResultValue{StringVal: `{"tool":"trivy","vulnerabilities":{"critical":1,"high":2,"medium":3,"low":4}}`}},
						{Name: "sbom-evidence", Value: tektonv1.ResultValue{StringVal: `{"format":"spdx","generator":"syft","version":"1.2.0"}`}},
						{Name: "source-image-digest", Value: tektonv1.ResultValue{StringVal: "sha256:abc123"}},
					},
				},
			},
		},
	}

	repo := postgres.NewImageImportRepository(db, zap.NewNop())
	handler := NewImportMonitorHandler(repo, reader, zap.NewNop())

	step := &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": importID.String(),
		},
	}

	// First pass should block because catalog image is not available yet.
	result, err := handler.Execute(context.Background(), step)
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusBlocked {
		t.Fatalf("expected blocked status on deferred sync, got %s", result.Status)
	}

	var statusAfterBlocked string
	if err := db.Get(&statusAfterBlocked, `SELECT status FROM external_image_imports WHERE id = $1`, importID); err != nil {
		t.Fatalf("failed reading import status after blocked pass: %v", err)
	}
	if statusAfterBlocked != "importing" {
		t.Fatalf("expected status to remain importing after blocked pass, got %s", statusAfterBlocked)
	}

	// Add catalog image row expected by repository resolution and run monitor again.
	imageID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO catalog_images (id, tenant_id, name, repository_url, updated_at)
		VALUES ($1, $2, 'base', 'registry.local/published/team-a/base', NOW())
	`, imageID, tenantID)
	if err != nil {
		t.Fatalf("failed seeding catalog_images: %v", err)
	}

	result, err = handler.Execute(context.Background(), step)
	if err != nil {
		t.Fatalf("expected no execute error on second pass, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded status on second pass, got %s", result.Status)
	}

	var finalStatus string
	if err := db.Get(&finalStatus, `SELECT status FROM external_image_imports WHERE id = $1`, importID); err != nil {
		t.Fatalf("failed reading final import status: %v", err)
	}
	if finalStatus != "success" {
		t.Fatalf("expected final import status success, got %s", finalStatus)
	}

	var sbomCount int
	if err := db.Get(&sbomCount, `SELECT COUNT(*) FROM catalog_image_sbom WHERE image_id = $1`, imageID); err != nil {
		t.Fatalf("failed reading sbom rows: %v", err)
	}
	if sbomCount != 1 {
		t.Fatalf("expected one catalog_image_sbom row, got %d", sbomCount)
	}

	var scan struct {
		Critical int    `db:"vulnerabilities_critical"`
		High     int    `db:"vulnerabilities_high"`
		Medium   int    `db:"vulnerabilities_medium"`
		Tool     string `db:"scan_tool"`
	}
	if err := db.Get(&scan, `
		SELECT vulnerabilities_critical, vulnerabilities_high, vulnerabilities_medium, scan_tool
		FROM catalog_image_vulnerability_scans
		WHERE image_id = $1
		ORDER BY completed_at DESC
		LIMIT 1
	`, imageID); err != nil {
		t.Fatalf("failed reading vulnerability scan row: %v", err)
	}
	if scan.Tool != "trivy" || scan.Critical != 1 || scan.High != 2 || scan.Medium != 3 {
		t.Fatalf("unexpected vulnerability scan projection: %+v", scan)
	}

	var metadata struct {
		SBOMStatus string `db:"sbom_evidence_status"`
		VulnStatus string `db:"vulnerability_evidence_status"`
	}
	if err := db.Get(&metadata, `
		SELECT sbom_evidence_status, vulnerability_evidence_status
		FROM catalog_image_metadata
		WHERE image_id = $1
	`, imageID); err != nil {
		t.Fatalf("failed reading metadata row: %v", err)
	}
	if metadata.SBOMStatus != "fresh" || metadata.VulnStatus != "fresh" {
		t.Fatalf("expected fresh metadata evidence statuses, got sbom=%s vuln=%s", metadata.SBOMStatus, metadata.VulnStatus)
	}
}

func TestImportMonitorHandler_PartialEvidence_ProjectsCatalogFallbacks(t *testing.T) {
	db := openImportMonitorIntegrationDB(t)
	createImportMonitorTempTables(t, db)

	tenantID := uuid.New()
	importID := uuid.New()
	now := time.Now().UTC()

	_, err := db.Exec(`
		INSERT INTO external_image_imports (
			id, tenant_id, requested_by_user_id, sor_record_id, source_registry, source_image_ref,
			status, internal_image_ref, pipeline_run_name, pipeline_namespace, created_at, updated_at
		) VALUES ($1, $2, $3, 'APP-8', 'ghcr.io', 'ghcr.io/acme/base:8.0.0', 'importing', $4, 'pr-8', 'ns-a', $5, $5)
	`, importID, tenantID, uuid.New(), "registry.local/published/team-a/base:latest", now)
	if err != nil {
		t.Fatalf("failed seeding external_image_imports: %v", err)
	}

	imageID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO catalog_images (id, tenant_id, name, repository_url, updated_at)
		VALUES ($1, $2, 'base', 'registry.local/published/team-a/base', NOW())
	`, imageID, tenantID)
	if err != nil {
		t.Fatalf("failed seeding catalog_images: %v", err)
	}

	// No sbom-evidence or scan-summary: monitor should fall back to sbom-summary for SBOM projection,
	// while leaving vulnerability projection unavailable.
	reader := &importMonitorRunReaderStub{
		run: &tektonv1.PipelineRun{
			Status: tektonv1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{Type: apis.ConditionSucceeded, Status: corev1.ConditionTrue},
					},
				},
				PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
					Results: []tektonv1.PipelineRunResult{
						{Name: "decision", Value: tektonv1.ResultValue{StringVal: "pass"}},
						{Name: "sbom-summary", Value: tektonv1.ResultValue{StringVal: `{"format":"cyclonedx","generator":"syft","version":"1.4.0","package_count":18}`}},
						{Name: "source-image-digest", Value: tektonv1.ResultValue{StringVal: "sha256:partial-evidence"}},
					},
				},
			},
		},
	}

	repo := postgres.NewImageImportRepository(db, zap.NewNop())
	handler := NewImportMonitorHandler(repo, reader, zap.NewNop())
	step := &domainworkflow.Step{
		Payload: map[string]interface{}{
			"tenant_id":                tenantID.String(),
			"external_image_import_id": importID.String(),
		},
	}

	result, err := handler.Execute(context.Background(), step)
	if err != nil {
		t.Fatalf("expected no execute error, got %v", err)
	}
	if result.Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", result.Status)
	}

	var finalStatus string
	if err := db.Get(&finalStatus, `SELECT status FROM external_image_imports WHERE id = $1`, importID); err != nil {
		t.Fatalf("failed reading final import status: %v", err)
	}
	if finalStatus != "success" {
		t.Fatalf("expected final status success, got %s", finalStatus)
	}

	var sbom struct {
		Format  string `db:"sbom_format"`
		Content string `db:"sbom_content"`
		Tool    string `db:"generated_by_tool"`
	}
	if err := db.Get(&sbom, `
		SELECT sbom_format, sbom_content, generated_by_tool
		FROM catalog_image_sbom
		WHERE image_id = $1
	`, imageID); err != nil {
		t.Fatalf("failed reading sbom projection row: %v", err)
	}
	if sbom.Format != "cyclonedx" || sbom.Tool != "syft" {
		t.Fatalf("unexpected sbom projection values: %+v", sbom)
	}
	if !strings.Contains(sbom.Content, `"package_count":18`) {
		t.Fatalf("expected sbom_summary fallback content to be projected, got %q", sbom.Content)
	}

	var vulnScanCount int
	if err := db.Get(&vulnScanCount, `SELECT COUNT(*) FROM catalog_image_vulnerability_scans WHERE image_id = $1`, imageID); err != nil {
		t.Fatalf("failed reading vulnerability scan rows: %v", err)
	}
	if vulnScanCount != 0 {
		t.Fatalf("expected no vulnerability scan rows for partial evidence without scan-summary, got %d", vulnScanCount)
	}

	var metadata struct {
		Digest     string `db:"docker_manifest_digest"`
		SBOMStatus string `db:"sbom_evidence_status"`
		VulnStatus string `db:"vulnerability_evidence_status"`
	}
	if err := db.Get(&metadata, `
		SELECT docker_manifest_digest, sbom_evidence_status, vulnerability_evidence_status
		FROM catalog_image_metadata
		WHERE image_id = $1
	`, imageID); err != nil {
		t.Fatalf("failed reading metadata projection row: %v", err)
	}
	if metadata.Digest != "sha256:partial-evidence" {
		t.Fatalf("expected digest fallback to be projected, got %q", metadata.Digest)
	}
	if metadata.SBOMStatus != "fresh" || metadata.VulnStatus != "unavailable" {
		t.Fatalf("expected sbom fresh + vulnerability unavailable statuses, got sbom=%s vuln=%s", metadata.SBOMStatus, metadata.VulnStatus)
	}
}
