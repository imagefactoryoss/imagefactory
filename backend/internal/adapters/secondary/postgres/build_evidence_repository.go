package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/application/imagecatalog"
	"go.uber.org/zap"
)

// BuildEvidenceRepository persists normalized build evidence rows.
type BuildEvidenceRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewBuildEvidenceRepository(db *sqlx.DB, logger *zap.Logger) *BuildEvidenceRepository {
	return &BuildEvidenceRepository{db: db, logger: logger}
}

func (r *BuildEvidenceRepository) PersistBuildEvidence(ctx context.Context, evidence *imagecatalog.BuildEvidence) error {
	if evidence == nil || evidence.BuildID == uuid.Nil || evidence.ImageID == uuid.Nil {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin evidence tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	working := *evidence
	policy, err := r.resolveEvidenceWritePolicy(ctx, tx, evidence)
	if err != nil {
		return err
	}
	if !policy.AllowLayers {
		working.Layers = nil
	}
	if !policy.AllowSBOM {
		working.SBOM = nil
	}
	if !policy.AllowVulnerability {
		working.VulnerabilityScan = nil
	}

	if err := r.replaceBuildArtifacts(ctx, tx, &working); err != nil {
		return err
	}
	if err := r.replaceBuildMetrics(ctx, tx, &working); err != nil {
		return err
	}
	if err := r.upsertImageMetadata(ctx, tx, &working); err != nil {
		return err
	}
	if err := r.replaceImageLayers(ctx, tx, &working); err != nil {
		return err
	}
	if err := r.replaceLayerEvidence(ctx, tx, &working); err != nil {
		return err
	}
	sbomID, err := r.upsertImageSBOM(ctx, tx, &working)
	if err != nil {
		return err
	}
	if err := r.replaceSBOMPackages(ctx, tx, &working, sbomID); err != nil {
		return err
	}
	if err := r.replaceLayerPackages(ctx, tx, &working); err != nil {
		return err
	}
	if err := r.replaceLayerVulnerabilityMappings(ctx, tx, &working); err != nil {
		return err
	}
	if err := r.replaceVulnerabilityScan(ctx, tx, &working); err != nil {
		return err
	}
	if err := r.updateEvidenceFreshness(ctx, tx, &working, policy); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit evidence tx: %w", err)
	}
	return nil
}

func (r *BuildEvidenceRepository) replaceBuildArtifacts(ctx context.Context, tx *sqlx.Tx, evidence *imagecatalog.BuildEvidence) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM build_artifacts WHERE build_id = $1`, evidence.BuildID); err != nil {
		return fmt.Errorf("delete build_artifacts: %w", err)
	}
	if len(evidence.Artifacts) == 0 {
		return nil
	}
	query := `
		INSERT INTO build_artifacts (
			id, build_id, artifact_type, artifact_name, artifact_version, artifact_location,
			artifact_mime_type, artifact_size_bytes, sha256_digest, is_available, image_id
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11
		)
	`
	for _, artifact := range evidence.Artifacts {
		artifactType := clampString(artifact.Type, 50)
		if artifactType == "" {
			artifactType = "build_output"
		}
		name := clampString(artifact.Name, 255)
		if name == "" {
			name = clampString(artifact.Location, 255)
		}
		if name == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, query,
			uuid.New(),
			evidence.BuildID,
			artifactType,
			name,
			nullIfEmpty(clampString(artifact.Version, 100)),
			nullIfEmpty(clampString(artifact.Location, 500)),
			nullIfEmpty(clampString(artifact.MimeType, 100)),
			artifact.SizeBytes,
			nullIfEmpty(normalizeSHA256(artifact.SHA256, artifact.Name)),
			artifact.IsAvailable,
			artifact.ImageID,
		); err != nil {
			return fmt.Errorf("insert build_artifact: %w", err)
		}
	}
	return nil
}

func (r *BuildEvidenceRepository) replaceBuildMetrics(ctx context.Context, tx *sqlx.Tx, evidence *imagecatalog.BuildEvidence) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM build_metrics WHERE build_id = $1`, evidence.BuildID); err != nil {
		return fmt.Errorf("delete build_metrics: %w", err)
	}
	totalLayers := len(evidence.Layers)
	var finalSize interface{}
	if evidence.ImageSizeBytes != nil {
		finalSize = *evidence.ImageSizeBytes
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO build_metrics (
			id, build_id, total_duration_seconds, total_layers, final_image_size_bytes
		) VALUES (
			$1, $2, $3, $4, $5
		)
	`, uuid.New(), evidence.BuildID, nullableInt(evidence.BuildDurationSeconds), nullableInt(totalLayers), finalSize); err != nil {
		return fmt.Errorf("insert build_metrics: %w", err)
	}
	return nil
}

func (r *BuildEvidenceRepository) upsertImageMetadata(ctx context.Context, tx *sqlx.Tx, evidence *imagecatalog.BuildEvidence) error {
	var totalLayers interface{}
	if hasLayerEvidence(evidence) {
		totalLayers = len(evidence.Layers)
	}
	var packagesCount interface{}
	if hasSBOMContent(evidence) {
		packagesCount = len(evidence.SBOM.Packages)
	}
	var vulnHigh, vulnMedium, vulnLow interface{}
	if evidence.VulnerabilityScan != nil {
		vulnHigh = evidence.VulnerabilityScan.High
		vulnMedium = evidence.VulnerabilityScan.Medium
		vulnLow = evidence.VulnerabilityScan.Low
	}
	var compressed interface{}
	if evidence.ImageSizeBytes != nil {
		compressed = *evidence.ImageSizeBytes
	}
	var scanTool interface{}
	if evidence.VulnerabilityScan != nil {
		scanTool = nullIfEmpty(evidence.ScanTool)
	}
	_, err := tx.ExecContext(ctx, `
			INSERT INTO catalog_image_metadata (
				id, image_id, docker_manifest_digest, total_layer_count, compressed_size_bytes,
				packages_count, vulnerabilities_high_count, vulnerabilities_medium_count, vulnerabilities_low_count,
				scan_tool, last_scanned_at
			) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10::text, CASE WHEN $10::text IS NULL THEN NULL ELSE NOW() END
		)
		ON CONFLICT (image_id) DO UPDATE SET
			docker_manifest_digest = COALESCE(EXCLUDED.docker_manifest_digest, catalog_image_metadata.docker_manifest_digest),
			total_layer_count = COALESCE(EXCLUDED.total_layer_count, catalog_image_metadata.total_layer_count),
			compressed_size_bytes = COALESCE(EXCLUDED.compressed_size_bytes, catalog_image_metadata.compressed_size_bytes),
			packages_count = COALESCE(EXCLUDED.packages_count, catalog_image_metadata.packages_count),
			vulnerabilities_high_count = COALESCE(EXCLUDED.vulnerabilities_high_count, catalog_image_metadata.vulnerabilities_high_count),
			vulnerabilities_medium_count = COALESCE(EXCLUDED.vulnerabilities_medium_count, catalog_image_metadata.vulnerabilities_medium_count),
			vulnerabilities_low_count = COALESCE(EXCLUDED.vulnerabilities_low_count, catalog_image_metadata.vulnerabilities_low_count),
			scan_tool = COALESCE(EXCLUDED.scan_tool, catalog_image_metadata.scan_tool),
			last_scanned_at = COALESCE(EXCLUDED.last_scanned_at, catalog_image_metadata.last_scanned_at)
	`,
		uuid.New(),
		evidence.ImageID,
		nullIfEmpty(evidence.ImageDigest),
		totalLayers,
		compressed,
		packagesCount,
		vulnHigh,
		vulnMedium,
		vulnLow,
		scanTool,
	)
	if err != nil {
		return fmt.Errorf("upsert catalog_image_metadata: %w", err)
	}
	return nil
}

func (r *BuildEvidenceRepository) replaceImageLayers(ctx context.Context, tx *sqlx.Tx, evidence *imagecatalog.BuildEvidence) error {
	if !hasLayerEvidence(evidence) {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM catalog_image_layers WHERE image_id = $1`, evidence.ImageID); err != nil {
		return fmt.Errorf("delete catalog_image_layers: %w", err)
	}
	query := `
		INSERT INTO catalog_image_layers (
			id, image_id, layer_number, layer_digest, layer_size_bytes, media_type
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
	`
	for _, layer := range evidence.Layers {
		if strings.TrimSpace(layer.Digest) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, query,
			uuid.New(),
			evidence.ImageID,
			layer.LayerNumber,
			clampString(layer.Digest, 255),
			layer.SizeBytes,
			nullIfEmpty(clampString(layer.MediaType, 100)),
		); err != nil {
			return fmt.Errorf("insert catalog_image_layer: %w", err)
		}
	}
	return nil
}

func (r *BuildEvidenceRepository) replaceLayerEvidence(ctx context.Context, tx *sqlx.Tx, evidence *imagecatalog.BuildEvidence) error {
	if !hasLayerEvidence(evidence) {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM catalog_image_layer_evidence WHERE image_id = $1`, evidence.ImageID); err != nil {
		return fmt.Errorf("delete catalog_image_layer_evidence: %w", err)
	}
	query := `
		INSERT INTO catalog_image_layer_evidence (
			id, image_id, layer_digest, layer_number, history_created_by, source_command, diff_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
	`
	for _, layer := range evidence.Layers {
		layerDigest := strings.TrimSpace(layer.Digest)
		if layerDigest == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, query,
			uuid.New(),
			evidence.ImageID,
			layerDigest,
			layer.LayerNumber,
			nullIfEmpty(layer.HistoryCreatedBy),
			nullIfEmpty(layer.SourceCommand),
			nullIfEmpty(layer.DiffID),
		); err != nil {
			return fmt.Errorf("insert catalog_image_layer_evidence: %w", err)
		}
	}
	return nil
}

func (r *BuildEvidenceRepository) upsertImageSBOM(ctx context.Context, tx *sqlx.Tx, evidence *imagecatalog.BuildEvidence) (*uuid.UUID, error) {
	if !hasSBOMContent(evidence) {
		return nil, nil
	}
	sbomID := uuid.New()
	var existing uuid.UUID
	if err := tx.GetContext(ctx, &existing, `SELECT id FROM catalog_image_sbom WHERE image_id = $1`, evidence.ImageID); err == nil {
		sbomID = existing
	}

	duration := evidence.SBOM.DurationSecs
	_, err := tx.ExecContext(ctx, `
		INSERT INTO catalog_image_sbom (
			id, image_id, sbom_format, sbom_version, sbom_content, generated_by_tool,
			tool_version, sbom_checksum, scan_timestamp, scan_duration_seconds, status
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, NOW(), $9, $10
		)
		ON CONFLICT (image_id) DO UPDATE SET
			sbom_format = EXCLUDED.sbom_format,
			sbom_version = EXCLUDED.sbom_version,
			sbom_content = EXCLUDED.sbom_content,
			generated_by_tool = EXCLUDED.generated_by_tool,
			tool_version = EXCLUDED.tool_version,
			sbom_checksum = EXCLUDED.sbom_checksum,
			scan_timestamp = EXCLUDED.scan_timestamp,
			scan_duration_seconds = EXCLUDED.scan_duration_seconds,
			status = EXCLUDED.status
	`,
		sbomID,
		evidence.ImageID,
		defaultIfEmpty(evidence.SBOM.Format, "spdx"),
		nullIfEmpty(evidence.SBOM.Version),
		evidence.SBOM.Content,
		nullIfEmpty(evidence.SBOM.GeneratedBy),
		nullIfEmpty(evidence.SBOM.ToolVersion),
		nullIfEmpty(normalizeSHA256(evidence.SBOM.Checksum, evidence.SBOM.Content)),
		duration,
		defaultIfEmpty(evidence.SBOM.Status, "valid"),
	)
	if err != nil {
		return nil, fmt.Errorf("upsert catalog_image_sbom: %w", err)
	}
	return &sbomID, nil
}

func (r *BuildEvidenceRepository) replaceSBOMPackages(ctx context.Context, tx *sqlx.Tx, evidence *imagecatalog.BuildEvidence, sbomID *uuid.UUID) error {
	if !hasSBOMPackages(evidence) {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sbom_packages WHERE image_id = $1`, evidence.ImageID); err != nil {
		return fmt.Errorf("delete sbom_packages: %w", err)
	}
	if sbomID == nil {
		return nil
	}
	query := `
		INSERT INTO sbom_packages (
			id, image_sbom_id, image_id, package_name, package_version, package_type,
			package_url, homepage_url, license_name, license_spdx_id, package_path,
			known_vulnerabilities_count, critical_vulnerabilities_count
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13
		)
	`
	for _, pkg := range evidence.SBOM.Packages {
		if strings.TrimSpace(pkg.Name) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, query,
			uuid.New(),
			*sbomID,
			evidence.ImageID,
			clampString(pkg.Name, 255),
			nullIfEmpty(clampString(pkg.Version, 100)),
			nullIfEmpty(clampString(pkg.Type, 50)),
			nullIfEmpty(clampString(pkg.PackageURL, 500)),
			nullIfEmpty(clampString(pkg.HomepageURL, 500)),
			nullIfEmpty(clampString(pkg.LicenseName, 255)),
			nullIfEmpty(clampString(pkg.LicenseSPDXID, 50)),
			nullIfEmpty(clampString(pkg.PackagePath, 500)),
			pkg.KnownVulnCount,
			pkg.CriticalCount,
		); err != nil {
			return fmt.Errorf("insert sbom_package: %w", err)
		}
	}
	return nil
}

func (r *BuildEvidenceRepository) replaceLayerPackages(ctx context.Context, tx *sqlx.Tx, evidence *imagecatalog.BuildEvidence) error {
	if !hasLayerPackageEvidence(evidence) {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM catalog_image_layer_packages WHERE image_id = $1`, evidence.ImageID); err != nil {
		return fmt.Errorf("delete catalog_image_layer_packages: %w", err)
	}
	query := `
		INSERT INTO catalog_image_layer_packages (
			id, image_id, layer_digest, package_name, package_version, package_type, package_path, source_command,
			known_vulnerabilities_count, critical_vulnerabilities_count
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`
	for _, pkg := range evidence.SBOM.Packages {
		layerDigest := strings.TrimSpace(pkg.LayerDigest)
		packageName := strings.TrimSpace(pkg.Name)
		if layerDigest == "" || packageName == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, query,
			uuid.New(),
			evidence.ImageID,
			clampString(layerDigest, 255),
			clampString(packageName, 255),
			defaultIfEmpty(clampString(pkg.Version, 100), ""),
			nullIfEmpty(clampString(pkg.Type, 50)),
			nullIfEmpty(clampString(pkg.PackagePath, 500)),
			nullIfEmpty(pkg.SourceCommand),
			pkg.KnownVulnCount,
			pkg.CriticalCount,
		); err != nil {
			return fmt.Errorf("insert catalog_image_layer_package: %w", err)
		}
	}
	return nil
}

func (r *BuildEvidenceRepository) replaceLayerVulnerabilityMappings(ctx context.Context, tx *sqlx.Tx, evidence *imagecatalog.BuildEvidence) error {
	if !hasLayerPackageEvidence(evidence) {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM catalog_image_layer_vulnerabilities WHERE image_id = $1`, evidence.ImageID); err != nil {
		return fmt.Errorf("delete catalog_image_layer_vulnerabilities: %w", err)
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO catalog_image_layer_vulnerabilities (
			id, image_id, layer_digest, package_name, package_version, cve_id, severity, cvss_v3_score, reference_url
		)
		SELECT
			uuid_generate_v4(),
			lp.image_id,
			lp.layer_digest,
			lp.package_name,
			lp.package_version,
			cve.cve_id,
			UPPER(COALESCE(cve.cvss_v3_severity, 'UNKNOWN')),
			cve.cvss_v3_score,
			CASE
				WHEN cve.cve_id IS NULL OR TRIM(cve.cve_id) = '' THEN NULL
				ELSE 'https://nvd.nist.gov/vuln/detail/' || cve.cve_id
			END AS reference_url
		FROM catalog_image_layer_packages lp
		JOIN package_vulnerabilities pv
		  ON pv.package_name = lp.package_name
		 AND COALESCE(pv.package_version, '') = COALESCE(lp.package_version, '')
		JOIN cve_database cve ON cve.cve_id = pv.cve_id
		WHERE lp.image_id = $1
	`, evidence.ImageID)
	if err != nil {
		return fmt.Errorf("insert catalog_image_layer_vulnerabilities: %w", err)
	}
	return nil
}

func (r *BuildEvidenceRepository) replaceVulnerabilityScan(ctx context.Context, tx *sqlx.Tx, evidence *imagecatalog.BuildEvidence) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM catalog_image_vulnerability_scans WHERE image_id = $1 AND build_id = $2`, evidence.ImageID, evidence.BuildID); err != nil {
		return fmt.Errorf("delete vulnerability_scan: %w", err)
	}
	if evidence.VulnerabilityScan == nil {
		return nil
	}
	scan := evidence.VulnerabilityScan
	_, err := tx.ExecContext(ctx, `
		INSERT INTO catalog_image_vulnerability_scans (
			id, image_id, build_id, scan_tool, tool_version, scan_status,
			started_at, completed_at, duration_seconds,
			vulnerabilities_critical, vulnerabilities_high, vulnerabilities_medium, vulnerabilities_low,
			vulnerabilities_negligible, vulnerabilities_unknown,
			pass_fail_result, compliance_check_passed, scan_report_json, error_message
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			NOW(), NOW(), NULL,
			$7, $8, $9, $10,
			$11, $12,
			$13, $14, $15, $16
		)
	`,
		uuid.New(),
		evidence.ImageID,
		evidence.BuildID,
		defaultIfEmpty(scan.Tool, "unknown"),
		nullIfEmpty(scan.ToolVersion),
		defaultIfEmpty(scan.Status, "completed"),
		scan.Critical,
		scan.High,
		scan.Medium,
		scan.Low,
		scan.Negligible,
		scan.Unknown,
		nullIfEmpty(scan.PassFail),
		scan.ComplianceOK,
		nullIfEmpty(scan.ReportJSON),
		nullIfEmpty(scan.ErrorMessage),
	)
	if err != nil {
		return fmt.Errorf("insert vulnerability_scan: %w", err)
	}
	return nil
}

func (r *BuildEvidenceRepository) updateEvidenceFreshness(ctx context.Context, tx *sqlx.Tx, evidence *imagecatalog.BuildEvidence, policy evidenceWritePolicy) error {
	layersFresh := hasLayerEvidence(evidence)
	sbomFresh := hasSBOMContent(evidence)
	vulnFresh := hasVulnerabilityEvidence(evidence)

	layersExists, err := r.evidenceTypeExists(ctx, tx, "catalog_image_layers", evidence.ImageID)
	if err != nil {
		return err
	}
	sbomExists, err := r.evidenceTypeExists(ctx, tx, "catalog_image_sbom", evidence.ImageID)
	if err != nil {
		return err
	}
	vulnExists, err := r.evidenceTypeExists(ctx, tx, "catalog_image_vulnerability_scans", evidence.ImageID)
	if err != nil {
		return err
	}

	if policy.AllowLayers {
		if err := r.updateEvidenceStatusForType(ctx, tx, evidence.ImageID, evidence.BuildID, "layers", layersFresh, layersExists); err != nil {
			return err
		}
	}
	if policy.AllowSBOM {
		if err := r.updateEvidenceStatusForType(ctx, tx, evidence.ImageID, evidence.BuildID, "sbom", sbomFresh, sbomExists); err != nil {
			return err
		}
	}
	if policy.AllowVulnerability {
		if err := r.updateEvidenceStatusForType(ctx, tx, evidence.ImageID, evidence.BuildID, "vulnerability", vulnFresh, vulnExists); err != nil {
			return err
		}
	}

	return nil
}

type evidenceWritePolicy struct {
	AllowLayers        bool
	AllowSBOM          bool
	AllowVulnerability bool
}

type metadataEvidenceOrderingRow struct {
	LayersEvidenceBuildID          *uuid.UUID `db:"layers_evidence_build_id"`
	LayersEvidenceUpdatedAt        *time.Time `db:"layers_evidence_updated_at"`
	SBOMEvidenceBuildID            *uuid.UUID `db:"sbom_evidence_build_id"`
	SBOMEvidenceUpdatedAt          *time.Time `db:"sbom_evidence_updated_at"`
	VulnerabilityEvidenceBuildID   *uuid.UUID `db:"vulnerability_evidence_build_id"`
	VulnerabilityEvidenceUpdatedAt *time.Time `db:"vulnerability_evidence_updated_at"`
}

func (r *BuildEvidenceRepository) resolveEvidenceWritePolicy(ctx context.Context, tx *sqlx.Tx, evidence *imagecatalog.BuildEvidence) (evidenceWritePolicy, error) {
	policy := evidenceWritePolicy{
		AllowLayers:        true,
		AllowSBOM:          true,
		AllowVulnerability: true,
	}
	if evidence == nil || evidence.ImageID == uuid.Nil {
		return policy, nil
	}
	var row metadataEvidenceOrderingRow
	if err := tx.GetContext(ctx, &row, `
		SELECT
			layers_evidence_build_id,
			layers_evidence_updated_at,
			sbom_evidence_build_id,
			sbom_evidence_updated_at,
			vulnerability_evidence_build_id,
			vulnerability_evidence_updated_at
		FROM catalog_image_metadata
		WHERE image_id = $1
	`, evidence.ImageID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return policy, nil
		}
		return policy, fmt.Errorf("load catalog_image_metadata ordering state: %w", err)
	}
	policy.AllowLayers = shouldWriteEvidenceType(evidence.BuildID, evidence.BuildCompletedAt, row.LayersEvidenceBuildID, row.LayersEvidenceUpdatedAt)
	policy.AllowSBOM = shouldWriteEvidenceType(evidence.BuildID, evidence.BuildCompletedAt, row.SBOMEvidenceBuildID, row.SBOMEvidenceUpdatedAt)
	policy.AllowVulnerability = shouldWriteEvidenceType(evidence.BuildID, evidence.BuildCompletedAt, row.VulnerabilityEvidenceBuildID, row.VulnerabilityEvidenceUpdatedAt)
	if r.logger != nil && (!policy.AllowLayers || !policy.AllowSBOM || !policy.AllowVulnerability) {
		r.logger.Warn("Skipping out-of-order evidence overwrite for catalog image",
			zap.String("image_id", evidence.ImageID.String()),
			zap.String("build_id", evidence.BuildID.String()),
			zap.Bool("allow_layers", policy.AllowLayers),
			zap.Bool("allow_sbom", policy.AllowSBOM),
			zap.Bool("allow_vulnerability", policy.AllowVulnerability),
		)
	}
	return policy, nil
}

func shouldWriteEvidenceType(incomingBuildID uuid.UUID, incomingCompletedAt *time.Time, existingBuildID *uuid.UUID, existingUpdatedAt *time.Time) bool {
	if existingBuildID == nil || *existingBuildID == uuid.Nil {
		return true
	}
	if incomingBuildID != uuid.Nil && *existingBuildID == incomingBuildID {
		return true
	}
	if incomingCompletedAt == nil || incomingCompletedAt.IsZero() {
		return false
	}
	if existingUpdatedAt == nil || existingUpdatedAt.IsZero() {
		return true
	}
	incoming := incomingCompletedAt.UTC()
	existing := existingUpdatedAt.UTC()
	return incoming.Equal(existing) || incoming.After(existing)
}

func (r *BuildEvidenceRepository) evidenceTypeExists(ctx context.Context, tx *sqlx.Tx, tableName string, imageID uuid.UUID) (bool, error) {
	if strings.TrimSpace(tableName) == "" {
		return false, fmt.Errorf("empty table name for evidence existence check")
	}
	query := fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s WHERE image_id = $1)`, tableName)
	var exists bool
	if err := tx.GetContext(ctx, &exists, query, imageID); err != nil {
		return false, fmt.Errorf("check evidence existence on %s: %w", tableName, err)
	}
	return exists, nil
}

func (r *BuildEvidenceRepository) updateEvidenceStatusForType(ctx context.Context, tx *sqlx.Tx, imageID uuid.UUID, buildID uuid.UUID, evidenceType string, isFresh bool, dataExists bool) error {
	status := deriveEvidenceStatus(isFresh, dataExists)
	statusColumn := ""
	buildColumn := ""
	updatedColumn := ""
	switch evidenceType {
	case "layers":
		statusColumn = "layers_evidence_status"
		buildColumn = "layers_evidence_build_id"
		updatedColumn = "layers_evidence_updated_at"
	case "sbom":
		statusColumn = "sbom_evidence_status"
		buildColumn = "sbom_evidence_build_id"
		updatedColumn = "sbom_evidence_updated_at"
	case "vulnerability":
		statusColumn = "vulnerability_evidence_status"
		buildColumn = "vulnerability_evidence_build_id"
		updatedColumn = "vulnerability_evidence_updated_at"
	default:
		return fmt.Errorf("unknown evidence type %q", evidenceType)
	}

	if isFresh {
		query := fmt.Sprintf(`
			UPDATE catalog_image_metadata
			SET %s = $2, %s = $3, %s = NOW()
			WHERE image_id = $1
		`, statusColumn, buildColumn, updatedColumn)
		if _, err := tx.ExecContext(ctx, query, imageID, status, buildID); err != nil {
			return fmt.Errorf("update %s freshness: %w", evidenceType, err)
		}
		return nil
	}

	if status == evidenceStatusUnavailable {
		query := fmt.Sprintf(`
			UPDATE catalog_image_metadata
			SET %s = $2, %s = NULL, %s = NULL
			WHERE image_id = $1
		`, statusColumn, buildColumn, updatedColumn)
		if _, err := tx.ExecContext(ctx, query, imageID, status); err != nil {
			return fmt.Errorf("update %s freshness: %w", evidenceType, err)
		}
		return nil
	}

	query := fmt.Sprintf(`
		UPDATE catalog_image_metadata
		SET %s = $2
		WHERE image_id = $1
	`, statusColumn)
	if _, err := tx.ExecContext(ctx, query, imageID, status); err != nil {
		return fmt.Errorf("update %s freshness: %w", evidenceType, err)
	}
	return nil
}

func nullableInt(v int) interface{} {
	if v <= 0 {
		return nil
	}
	return v
}

func defaultIfEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func clampString(v string, maxLen int) string {
	trimmed := strings.TrimSpace(v)
	if maxLen <= 0 || trimmed == "" {
		return trimmed
	}
	runes := []rune(trimmed)
	if len(runes) <= maxLen {
		return trimmed
	}
	return string(runes[:maxLen])
}

func hasLayerEvidence(evidence *imagecatalog.BuildEvidence) bool {
	if evidence == nil || len(evidence.Layers) == 0 {
		return false
	}
	for _, layer := range evidence.Layers {
		if strings.TrimSpace(layer.Digest) != "" {
			return true
		}
	}
	return false
}

func hasSBOMContent(evidence *imagecatalog.BuildEvidence) bool {
	return evidence != nil && evidence.SBOM != nil && strings.TrimSpace(evidence.SBOM.Content) != ""
}

func hasSBOMPackages(evidence *imagecatalog.BuildEvidence) bool {
	if !hasSBOMContent(evidence) || len(evidence.SBOM.Packages) == 0 {
		return false
	}
	for _, pkg := range evidence.SBOM.Packages {
		if strings.TrimSpace(pkg.Name) != "" {
			return true
		}
	}
	return false
}

func hasLayerPackageEvidence(evidence *imagecatalog.BuildEvidence) bool {
	if !hasSBOMPackages(evidence) {
		return false
	}
	for _, pkg := range evidence.SBOM.Packages {
		if strings.TrimSpace(pkg.Name) != "" && strings.TrimSpace(pkg.LayerDigest) != "" {
			return true
		}
	}
	return false
}

const (
	evidenceStatusFresh       = "fresh"
	evidenceStatusStale       = "stale"
	evidenceStatusUnavailable = "unavailable"
)

func deriveEvidenceStatus(isFresh bool, dataExists bool) string {
	if isFresh {
		return evidenceStatusFresh
	}
	if dataExists {
		return evidenceStatusStale
	}
	return evidenceStatusUnavailable
}

func hasVulnerabilityEvidence(evidence *imagecatalog.BuildEvidence) bool {
	return evidence != nil && evidence.VulnerabilityScan != nil
}

func normalizeSHA256(candidate string, fallback string) string {
	value := strings.TrimSpace(candidate)
	if strings.HasPrefix(value, "sha256:") {
		value = strings.TrimPrefix(value, "sha256:")
	}
	if len(value) == 64 {
		return strings.ToLower(value)
	}
	if strings.TrimSpace(fallback) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(fallback))
	return hex.EncodeToString(sum[:])
}
