package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/imageimport"
)

type ImageImportRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewImageImportRepository(db *sqlx.DB, logger *zap.Logger) *ImageImportRepository {
	return &ImageImportRepository{db: db, logger: logger}
}

type imageImportRow struct {
	ID                   uuid.UUID      `db:"id"`
	TenantID             uuid.UUID      `db:"tenant_id"`
	RequestedByUserID    uuid.UUID      `db:"requested_by_user_id"`
	RequestType          string         `db:"request_type"`
	SORRecordID          string         `db:"sor_record_id"`
	SourceRegistry       string         `db:"source_registry"`
	SourceImageRef       string         `db:"source_image_ref"`
	RegistryAuthID       sql.NullString `db:"registry_auth_id"`
	Status               string         `db:"status"`
	ErrorMessage         sql.NullString `db:"error_message"`
	InternalImageRef     sql.NullString `db:"internal_image_ref"`
	PipelineRunName      sql.NullString `db:"pipeline_run_name"`
	PipelineNamespace    sql.NullString `db:"pipeline_namespace"`
	PolicyDecision       sql.NullString `db:"policy_decision"`
	PolicyReasonsJSON    sql.NullString `db:"policy_reasons_json"`
	PolicySnapshotJSON   sql.NullString `db:"policy_snapshot_json"`
	ScanSummaryJSON      sql.NullString `db:"scan_summary_json"`
	SBOMSummaryJSON      sql.NullString `db:"sbom_summary_json"`
	SBOMEvidenceJSON     sql.NullString `db:"sbom_evidence_json"`
	SourceImageDigest    sql.NullString `db:"source_image_digest"`
	ReleaseState         sql.NullString `db:"release_state"`
	ReleaseBlockerReason sql.NullString `db:"release_blocker_reason"`
	ReleaseActorUserID   sql.NullString `db:"release_actor_user_id"`
	ReleaseReason        sql.NullString `db:"release_reason"`
	ReleaseRequestedAt   sql.NullTime   `db:"release_requested_at"`
	ReleasedAt           sql.NullTime   `db:"released_at"`
	CreatedAt            time.Time      `db:"created_at"`
	UpdatedAt            time.Time      `db:"updated_at"`
}

type releasedArtifactRow struct {
	ID                 uuid.UUID      `db:"id"`
	TenantID           uuid.UUID      `db:"tenant_id"`
	RequestedByUserID  uuid.UUID      `db:"requested_by_user_id"`
	SORRecordID        string         `db:"sor_record_id"`
	SourceRegistry     string         `db:"source_registry"`
	SourceImageRef     string         `db:"source_image_ref"`
	InternalImageRef   sql.NullString `db:"internal_image_ref"`
	SourceImageDigest  sql.NullString `db:"source_image_digest"`
	PolicyDecision     sql.NullString `db:"policy_decision"`
	PolicySnapshotJSON sql.NullString `db:"policy_snapshot_json"`
	ReleaseState       sql.NullString `db:"release_state"`
	ReleaseReason      sql.NullString `db:"release_reason"`
	ReleaseActorUserID sql.NullString `db:"release_actor_user_id"`
	ReleaseRequestedAt sql.NullTime   `db:"release_requested_at"`
	ReleasedAt         sql.NullTime   `db:"released_at"`
	CreatedAt          time.Time      `db:"created_at"`
	UpdatedAt          time.Time      `db:"updated_at"`
}

func (r *ImageImportRepository) Create(ctx context.Context, req *imageimport.ImportRequest) error {
	query := `
		INSERT INTO external_image_imports (
			id,
			tenant_id,
			requested_by_user_id,
			request_type,
			sor_record_id,
			source_registry,
			source_image_ref,
			registry_auth_id,
			status,
			error_message,
			internal_image_ref,
			pipeline_run_name,
			pipeline_namespace,
			policy_decision,
			policy_reasons_json,
			policy_snapshot_json,
			scan_summary_json,
			sbom_summary_json,
			sbom_evidence_json,
			source_image_digest,
			release_state,
			release_blocker_reason,
			release_actor_user_id,
			release_reason,
			release_requested_at,
			released_at,
			created_at,
			updated_at
		) VALUES (
			:id,
			:tenant_id,
			:requested_by_user_id,
			:request_type,
			:sor_record_id,
			:source_registry,
			:source_image_ref,
			:registry_auth_id,
			:status,
			:error_message,
			:internal_image_ref,
			:pipeline_run_name,
			:pipeline_namespace,
			:policy_decision,
			:policy_reasons_json,
			:policy_snapshot_json,
			:scan_summary_json,
			:sbom_summary_json,
			:sbom_evidence_json,
			:source_image_digest,
			:release_state,
			:release_blocker_reason,
			:release_actor_user_id,
			:release_reason,
			:release_requested_at,
			:released_at,
			:created_at,
			:updated_at
		)`

	params := map[string]interface{}{
		"id":                     req.ID,
		"tenant_id":              req.TenantID,
		"requested_by_user_id":   req.RequestedByUserID,
		"request_type":           string(req.RequestType),
		"sor_record_id":          req.SORRecordID,
		"source_registry":        req.SourceRegistry,
		"source_image_ref":       req.SourceImageRef,
		"registry_auth_id":       req.RegistryAuthID,
		"status":                 string(req.Status),
		"error_message":          nullableString(req.ErrorMessage),
		"internal_image_ref":     nullableString(req.InternalImageRef),
		"pipeline_run_name":      nullableString(req.PipelineRunName),
		"pipeline_namespace":     nullableString(req.PipelineNamespace),
		"policy_decision":        nullableString(req.PolicyDecision),
		"policy_reasons_json":    nullableString(req.PolicyReasonsJSON),
		"policy_snapshot_json":   nullableString(req.PolicySnapshotJSON),
		"scan_summary_json":      nullableString(req.ScanSummaryJSON),
		"sbom_summary_json":      nullableString(req.SBOMSummaryJSON),
		"sbom_evidence_json":     nullableString(req.SBOMEvidenceJSON),
		"source_image_digest":    nullableString(req.SourceImageDigest),
		"release_state":          nullableString(string(req.ReleaseState)),
		"release_blocker_reason": nullableString(req.ReleaseBlockerReason),
		"release_actor_user_id":  req.ReleaseActorUserID,
		"release_reason":         nullableString(req.ReleaseReason),
		"release_requested_at":   req.ReleaseRequestedAt,
		"released_at":            req.ReleasedAt,
		"created_at":             req.CreatedAt.UTC(),
		"updated_at":             req.UpdatedAt.UTC(),
	}

	_, err := r.db.NamedExecContext(ctx, query, params)
	if err != nil {
		r.logger.Error("Failed to insert external image import request", zap.Error(err), zap.String("tenant_id", req.TenantID.String()))
		return err
	}
	return nil
}

func (r *ImageImportRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*imageimport.ImportRequest, error) {
	query := `
		SELECT id, tenant_id, requested_by_user_id, request_type, sor_record_id, source_registry, source_image_ref, registry_auth_id, status, error_message, internal_image_ref, pipeline_run_name, pipeline_namespace, policy_decision, policy_reasons_json, policy_snapshot_json, scan_summary_json, sbom_summary_json, sbom_evidence_json, source_image_digest, release_state, release_blocker_reason, release_actor_user_id, release_reason, release_requested_at, released_at, created_at, updated_at
		FROM external_image_imports
		WHERE id = $1 AND tenant_id = $2`

	var row imageImportRow
	if err := r.db.GetContext(ctx, &row, query, id, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, imageimport.ErrImportNotFound
		}
		return nil, err
	}
	return mapImageImportRow(row)
}

func (r *ImageImportRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, requestType imageimport.RequestType, limit, offset int) ([]*imageimport.ImportRequest, error) {
	query := `
		SELECT id, tenant_id, requested_by_user_id, request_type, sor_record_id, source_registry, source_image_ref, registry_auth_id, status, error_message, internal_image_ref, pipeline_run_name, pipeline_namespace, policy_decision, policy_reasons_json, policy_snapshot_json, scan_summary_json, sbom_summary_json, sbom_evidence_json, source_image_digest, release_state, release_blocker_reason, release_actor_user_id, release_reason, release_requested_at, released_at, created_at, updated_at
		FROM external_image_imports
		WHERE tenant_id = $1
		  AND ($2 = '' OR request_type = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows := make([]imageImportRow, 0)
	if err := r.db.SelectContext(ctx, &rows, query, tenantID, string(requestType), limit, offset); err != nil {
		return nil, err
	}

	result := make([]*imageimport.ImportRequest, 0, len(rows))
	for _, row := range rows {
		mapped, err := mapImageImportRow(row)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

func (r *ImageImportRepository) ListAll(ctx context.Context, requestType imageimport.RequestType, limit, offset int) ([]*imageimport.ImportRequest, error) {
	query := `
		SELECT id, tenant_id, requested_by_user_id, request_type, sor_record_id, source_registry, source_image_ref, registry_auth_id, status, error_message, internal_image_ref, pipeline_run_name, pipeline_namespace, policy_decision, policy_reasons_json, policy_snapshot_json, scan_summary_json, sbom_summary_json, sbom_evidence_json, source_image_digest, release_state, release_blocker_reason, release_actor_user_id, release_reason, release_requested_at, released_at, created_at, updated_at
		FROM external_image_imports
		WHERE ($1 = '' OR request_type = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows := make([]imageImportRow, 0)
	if err := r.db.SelectContext(ctx, &rows, query, string(requestType), limit, offset); err != nil {
		return nil, err
	}

	result := make([]*imageimport.ImportRequest, 0, len(rows))
	for _, row := range rows {
		mapped, err := mapImageImportRow(row)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

func (r *ImageImportRepository) ListReleasedByTenant(ctx context.Context, tenantID uuid.UUID, search string, limit, offset int) ([]*imageimport.ReleasedArtifact, int, error) {
	search = strings.TrimSpace(search)

	filters := `
		FROM external_image_imports
		WHERE tenant_id = $1
		  AND request_type = 'quarantine'
		  AND release_state = 'released'
	`
	args := []interface{}{tenantID}
	if search != "" {
		args = append(args, "%"+search+"%")
		filters += `
		  AND (
			source_registry ILIKE $2
			OR source_image_ref ILIKE $2
			OR COALESCE(internal_image_ref, '') ILIKE $2
			OR COALESCE(source_image_digest, '') ILIKE $2
		  )
		`
	}

	countQuery := "SELECT COUNT(*) " + filters
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	limitIdx := len(args) - 1
	offsetIdx := len(args)
	listQuery := fmt.Sprintf(`
		SELECT id, tenant_id, requested_by_user_id, sor_record_id, source_registry, source_image_ref,
		       internal_image_ref, source_image_digest, policy_decision, policy_snapshot_json,
		       release_state, release_reason, release_actor_user_id, release_requested_at, released_at,
		       created_at, updated_at
		%s
		ORDER BY COALESCE(released_at, updated_at) DESC, created_at DESC
		LIMIT $%d OFFSET $%d
	`, filters, limitIdx, offsetIdx)

	rows := make([]releasedArtifactRow, 0)
	if err := r.db.SelectContext(ctx, &rows, listQuery, args...); err != nil {
		return nil, 0, err
	}

	items := make([]*imageimport.ReleasedArtifact, 0, len(rows))
	for _, row := range rows {
		var releaseActorUserID *uuid.UUID
		if row.ReleaseActorUserID.Valid && strings.TrimSpace(row.ReleaseActorUserID.String) != "" {
			parsed, err := uuid.Parse(row.ReleaseActorUserID.String)
			if err != nil {
				return nil, 0, err
			}
			releaseActorUserID = &parsed
		}

		item := &imageimport.ReleasedArtifact{
			ID:                 row.ID,
			TenantID:           row.TenantID,
			RequestedByUserID:  row.RequestedByUserID,
			SORRecordID:        row.SORRecordID,
			SourceRegistry:     row.SourceRegistry,
			SourceImageRef:     row.SourceImageRef,
			InternalImageRef:   strings.TrimSpace(row.InternalImageRef.String),
			SourceImageDigest:  strings.TrimSpace(row.SourceImageDigest.String),
			PolicyDecision:     strings.TrimSpace(row.PolicyDecision.String),
			PolicySnapshotJSON: strings.TrimSpace(row.PolicySnapshotJSON.String),
			ReleaseState:       imageimport.ReleaseState(strings.TrimSpace(row.ReleaseState.String)),
			ReleaseReason:      strings.TrimSpace(row.ReleaseReason.String),
			ReleaseActorUserID: releaseActorUserID,
			CreatedAt:          row.CreatedAt,
			UpdatedAt:          row.UpdatedAt,
		}
		if row.ReleaseRequestedAt.Valid {
			ts := row.ReleaseRequestedAt.Time
			item.ReleaseRequestedAt = &ts
		}
		if row.ReleasedAt.Valid {
			ts := row.ReleasedAt.Time
			item.ReleasedAt = &ts
		}
		items = append(items, item)
	}

	return items, total, nil
}

func (r *ImageImportRepository) IsArtifactRefReleased(ctx context.Context, tenantID uuid.UUID, imageRef string) (bool, error) {
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return true, nil
	}
	query := `
		SELECT
			COUNT(*) AS row_count,
			COALESCE(BOOL_OR(release_state = 'released'), false) AS has_released
		FROM external_image_imports
		WHERE tenant_id = $1
		  AND request_type = 'quarantine'
		  AND internal_image_ref = $2
	`
	var rowCount int
	var hasReleased bool
	if err := r.db.QueryRowxContext(ctx, query, tenantID, imageRef).Scan(&rowCount, &hasReleased); err != nil {
		return false, err
	}
	if rowCount == 0 {
		return true, nil
	}
	return hasReleased, nil
}

func (r *ImageImportRepository) GetArtifactReleaseStateByRef(ctx context.Context, tenantID uuid.UUID, imageRef string) (string, error) {
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return "", nil
	}

	query := `
		SELECT COALESCE(release_state, '')
		FROM external_image_imports
		WHERE tenant_id = $1
		  AND request_type = 'quarantine'
		  AND internal_image_ref = $2
		ORDER BY updated_at DESC
		LIMIT 1
	`
	var releaseState string
	if err := r.db.QueryRowxContext(ctx, query, tenantID, imageRef).Scan(&releaseState); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(releaseState), nil
}

func (r *ImageImportRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status imageimport.Status, errorMessage, internalImageRef string) error {
	query := `
		UPDATE external_image_imports
		SET status = $3,
		    error_message = $4,
		    internal_image_ref = $5,
		    updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2`
	res, err := r.db.ExecContext(ctx, query, id, tenantID, string(status), nullableString(errorMessage), nullableString(internalImageRef))
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return imageimport.ErrImportNotFound
	}
	return nil
}

func (r *ImageImportRepository) UpdatePipelineRefs(ctx context.Context, tenantID, id uuid.UUID, pipelineRunName, pipelineNamespace string) error {
	query := `
		UPDATE external_image_imports
		SET pipeline_run_name = $3,
		    pipeline_namespace = $4,
		    updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2`
	res, err := r.db.ExecContext(ctx, query, id, tenantID, nullableString(pipelineRunName), nullableString(pipelineNamespace))
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return imageimport.ErrImportNotFound
	}
	return nil
}

func (r *ImageImportRepository) UpdateEvidence(ctx context.Context, tenantID, id uuid.UUID, evidence imageimport.ImportEvidence) error {
	query := `
		UPDATE external_image_imports
		SET policy_decision = $3,
		    policy_reasons_json = $4,
		    policy_snapshot_json = $5,
		    scan_summary_json = $6,
		    sbom_summary_json = $7,
		    sbom_evidence_json = $8,
		    source_image_digest = $9,
		    updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2`
	res, err := r.db.ExecContext(
		ctx,
		query,
		id,
		tenantID,
		nullableString(evidence.PolicyDecision),
		nullableString(evidence.PolicyReasonsJSON),
		nullableString(evidence.PolicySnapshotJSON),
		nullableString(evidence.ScanSummaryJSON),
		nullableString(evidence.SBOMSummaryJSON),
		nullableString(evidence.SBOMEvidenceJSON),
		nullableString(evidence.SourceImageDigest),
	)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return imageimport.ErrImportNotFound
	}
	return nil
}

func (r *ImageImportRepository) UpdateReleaseState(
	ctx context.Context,
	tenantID, id uuid.UUID,
	state imageimport.ReleaseState,
	blockerReason string,
	actorUserID *uuid.UUID,
	reason string,
	requestedAt, releasedAt *time.Time,
) error {
	query := `
		UPDATE external_image_imports
		SET release_state = $3,
		    release_blocker_reason = $4,
		    release_actor_user_id = $5,
		    release_reason = $6,
		    release_requested_at = $7,
		    released_at = $8,
		    updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2`
	res, err := r.db.ExecContext(
		ctx,
		query,
		id,
		tenantID,
		nullableString(string(state)),
		nullableString(blockerReason),
		actorUserID,
		nullableString(reason),
		requestedAt,
		releasedAt,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return imageimport.ErrImportNotFound
	}
	return nil
}

func (r *ImageImportRepository) SyncEvidenceToCatalog(ctx context.Context, tenantID, id uuid.UUID) error {
	req, err := r.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if req == nil {
		return imageimport.ErrImportNotFound
	}
	if req.RequestType == imageimport.RequestTypeScan {
		return nil
	}

	imageID, err := r.resolveCatalogImageID(ctx, req.TenantID, req.InternalImageRef)
	if err != nil {
		return err
	}
	if imageID == uuid.Nil {
		return imageimport.ErrCatalogImageNotReady
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := r.upsertCatalogImageSBOM(ctx, tx, imageID, req); err != nil {
		return err
	}
	if err := r.insertCatalogVulnerabilityScan(ctx, tx, imageID, req); err != nil {
		return err
	}
	if err := r.upsertCatalogImageMetadata(ctx, tx, imageID, req); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func mapImageImportRow(row imageImportRow) (*imageimport.ImportRequest, error) {
	var registryAuthID *uuid.UUID
	if row.RegistryAuthID.Valid && row.RegistryAuthID.String != "" {
		parsed, err := uuid.Parse(row.RegistryAuthID.String)
		if err != nil {
			return nil, err
		}
		registryAuthID = &parsed
	}
	var releaseActorUserID *uuid.UUID
	if row.ReleaseActorUserID.Valid && row.ReleaseActorUserID.String != "" {
		parsed, err := uuid.Parse(row.ReleaseActorUserID.String)
		if err != nil {
			return nil, err
		}
		releaseActorUserID = &parsed
	}

	var releaseRequestedAt *time.Time
	if row.ReleaseRequestedAt.Valid {
		ts := row.ReleaseRequestedAt.Time
		releaseRequestedAt = &ts
	}
	var releasedAt *time.Time
	if row.ReleasedAt.Valid {
		ts := row.ReleasedAt.Time
		releasedAt = &ts
	}

	requestType := imageimport.RequestType(strings.TrimSpace(row.RequestType))
	if requestType == "" {
		requestType = imageimport.RequestTypeQuarantine
	}

	return &imageimport.ImportRequest{
		ID:                   row.ID,
		TenantID:             row.TenantID,
		RequestedByUserID:    row.RequestedByUserID,
		RequestType:          requestType,
		SORRecordID:          row.SORRecordID,
		SourceRegistry:       row.SourceRegistry,
		SourceImageRef:       row.SourceImageRef,
		RegistryAuthID:       registryAuthID,
		Status:               imageimport.Status(row.Status),
		ErrorMessage:         row.ErrorMessage.String,
		InternalImageRef:     row.InternalImageRef.String,
		PipelineRunName:      row.PipelineRunName.String,
		PipelineNamespace:    row.PipelineNamespace.String,
		PolicyDecision:       row.PolicyDecision.String,
		PolicyReasonsJSON:    row.PolicyReasonsJSON.String,
		PolicySnapshotJSON:   row.PolicySnapshotJSON.String,
		ScanSummaryJSON:      row.ScanSummaryJSON.String,
		SBOMSummaryJSON:      row.SBOMSummaryJSON.String,
		SBOMEvidenceJSON:     row.SBOMEvidenceJSON.String,
		SourceImageDigest:    row.SourceImageDigest.String,
		ReleaseState:         imageimport.ReleaseState(strings.TrimSpace(row.ReleaseState.String)),
		ReleaseBlockerReason: row.ReleaseBlockerReason.String,
		ReleaseActorUserID:   releaseActorUserID,
		ReleaseReason:        row.ReleaseReason.String,
		ReleaseRequestedAt:   releaseRequestedAt,
		ReleasedAt:           releasedAt,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}, nil
}

func nullableString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func (r *ImageImportRepository) resolveCatalogImageID(ctx context.Context, tenantID uuid.UUID, internalImageRef string) (uuid.UUID, error) {
	repositoryURL, imageName := parseRepositoryURLAndName(internalImageRef)
	if repositoryURL == "" && imageName == "" {
		return uuid.Nil, nil
	}

	var id uuid.UUID
	query := `
		SELECT id
		FROM catalog_images
		WHERE tenant_id = $1
		  AND (
		        ($2 <> '' AND repository_url = $2)
		     OR ($3 <> '' AND name = $3)
		  )
		ORDER BY updated_at DESC
		LIMIT 1`
	if err := r.db.GetContext(ctx, &id, query, tenantID, repositoryURL, imageName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, nil
		}
		return uuid.Nil, err
	}
	return id, nil
}

func (r *ImageImportRepository) upsertCatalogImageSBOM(ctx context.Context, tx *sqlx.Tx, imageID uuid.UUID, req *imageimport.ImportRequest) error {
	content := strings.TrimSpace(req.SBOMEvidenceJSON)
	if content == "" {
		content = strings.TrimSpace(req.SBOMSummaryJSON)
	}
	if content == "" {
		return nil
	}

	tool := ""
	version := ""
	format := "spdx"
	if parsed := parseJSONMap(content); parsed != nil {
		tool = stringMapValue(parsed, "generator")
		if tool == "" {
			tool = stringMapValue(parsed, "tool")
		}
		version = stringMapValue(parsed, "version")
		if v := stringMapValue(parsed, "format"); v != "" {
			format = v
		}
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO catalog_image_sbom (
			id, image_id, sbom_format, sbom_version, sbom_content, generated_by_tool,
			tool_version, scan_timestamp, status
		) VALUES (
			$1, $2, $3, NULL, $4, $5, $6, NOW(), 'valid'
		)
		ON CONFLICT (image_id) DO UPDATE SET
			sbom_format = EXCLUDED.sbom_format,
			sbom_content = EXCLUDED.sbom_content,
			generated_by_tool = COALESCE(EXCLUDED.generated_by_tool, catalog_image_sbom.generated_by_tool),
			tool_version = COALESCE(EXCLUDED.tool_version, catalog_image_sbom.tool_version),
			scan_timestamp = EXCLUDED.scan_timestamp,
			status = EXCLUDED.status`,
		uuid.New(), imageID, format, content, nullableString(tool), nullableString(version))
	return err
}

func (r *ImageImportRepository) insertCatalogVulnerabilityScan(ctx context.Context, tx *sqlx.Tx, imageID uuid.UUID, req *imageimport.ImportRequest) error {
	if strings.TrimSpace(req.ScanSummaryJSON) == "" {
		return nil
	}
	critical, high, medium, low, negligible, unknown := parseVulnerabilityCounts(req.ScanSummaryJSON)
	tool := "trivy"
	if parsed := parseJSONMap(req.ScanSummaryJSON); parsed != nil {
		if value := stringMapValue(parsed, "tool"); value != "" {
			tool = value
		}
	}

	passFail := "PASS"
	if strings.EqualFold(strings.TrimSpace(req.PolicyDecision), "quarantine") {
		passFail = "FAIL"
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO catalog_image_vulnerability_scans (
			id, image_id, build_id, scan_tool, tool_version, scan_status,
			started_at, completed_at, duration_seconds,
			vulnerabilities_critical, vulnerabilities_high, vulnerabilities_medium, vulnerabilities_low,
			vulnerabilities_negligible, vulnerabilities_unknown,
			pass_fail_result, compliance_check_passed, scan_report_json, error_message
		) VALUES (
			$1, $2, NULL, $3, NULL, 'completed',
			NOW(), NOW(), NULL,
			$4, $5, $6, $7,
			$8, $9,
			$10, NULL, $11, NULL
		)`,
		uuid.New(),
		imageID,
		tool,
		critical, high, medium, low, negligible, unknown,
		passFail,
		req.ScanSummaryJSON,
	)
	return err
}

func (r *ImageImportRepository) upsertCatalogImageMetadata(ctx context.Context, tx *sqlx.Tx, imageID uuid.UUID, req *imageimport.ImportRequest) error {
	critical, high, medium, low, _, _ := parseVulnerabilityCounts(req.ScanSummaryJSON)
	sbomFresh := strings.TrimSpace(req.SBOMEvidenceJSON) != "" || strings.TrimSpace(req.SBOMSummaryJSON) != ""
	vulnFresh := strings.TrimSpace(req.ScanSummaryJSON) != ""
	_, err := tx.ExecContext(ctx, `
		INSERT INTO catalog_image_metadata (
			id, image_id, docker_manifest_digest,
			packages_count, vulnerabilities_high_count, vulnerabilities_medium_count, vulnerabilities_low_count,
			layers_evidence_status, sbom_evidence_status, vulnerability_evidence_status,
			sbom_evidence_updated_at, vulnerability_evidence_updated_at
		) VALUES (
			$1, $2, $3, NULL, $4, $5, $6,
			'unavailable',
			CASE WHEN $7 THEN 'fresh' ELSE 'unavailable' END,
			CASE WHEN $8 THEN 'fresh' ELSE 'unavailable' END,
			CASE WHEN $7 THEN NOW() ELSE NULL END,
			CASE WHEN $8 THEN NOW() ELSE NULL END
		)
		ON CONFLICT (image_id) DO UPDATE SET
			docker_manifest_digest = COALESCE(EXCLUDED.docker_manifest_digest, catalog_image_metadata.docker_manifest_digest),
			vulnerabilities_high_count = COALESCE(EXCLUDED.vulnerabilities_high_count, catalog_image_metadata.vulnerabilities_high_count),
			vulnerabilities_medium_count = COALESCE(EXCLUDED.vulnerabilities_medium_count, catalog_image_metadata.vulnerabilities_medium_count),
			vulnerabilities_low_count = COALESCE(EXCLUDED.vulnerabilities_low_count, catalog_image_metadata.vulnerabilities_low_count),
			sbom_evidence_status = CASE WHEN $7 THEN 'fresh' ELSE catalog_image_metadata.sbom_evidence_status END,
			vulnerability_evidence_status = CASE WHEN $8 THEN 'fresh' ELSE catalog_image_metadata.vulnerability_evidence_status END,
			sbom_evidence_updated_at = CASE WHEN $7 THEN NOW() ELSE catalog_image_metadata.sbom_evidence_updated_at END,
			vulnerability_evidence_updated_at = CASE WHEN $8 THEN NOW() ELSE catalog_image_metadata.vulnerability_evidence_updated_at END`,
		uuid.New(),
		imageID,
		nullableString(req.SourceImageDigest),
		high+critical, medium, low,
		sbomFresh,
		vulnFresh,
	)
	return err
}

func parseRepositoryURLAndName(ref string) (string, string) {
	value := strings.TrimSpace(ref)
	if value == "" {
		return "", ""
	}
	repository := value
	if at := strings.Index(repository, "@"); at > 0 {
		repository = repository[:at]
	}
	lastSlash := strings.LastIndex(repository, "/")
	lastColon := strings.LastIndex(repository, ":")
	if lastColon > lastSlash {
		repository = repository[:lastColon]
	}
	repository = strings.TrimSpace(repository)
	imageName := ""
	if idx := strings.LastIndex(repository, "/"); idx >= 0 && idx+1 < len(repository) {
		imageName = strings.TrimSpace(repository[idx+1:])
	}
	return repository, imageName
}

func parseVulnerabilityCounts(raw string) (critical, high, medium, low, negligible, unknown int) {
	parsed := parseJSONMap(raw)
	if parsed == nil {
		return 0, 0, 0, 0, 0, 0
	}
	v, ok := parsed["vulnerabilities"].(map[string]interface{})
	if !ok {
		return 0, 0, 0, 0, 0, 0
	}
	critical = intMapValue(v, "critical")
	high = intMapValue(v, "high")
	medium = intMapValue(v, "medium")
	low = intMapValue(v, "low")
	negligible = intMapValue(v, "negligible")
	unknown = intMapValue(v, "unknown")
	return
}

func parseJSONMap(raw string) map[string]interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil
	}
	return m
}

func stringMapValue(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	raw, ok := m[key]
	if !ok || raw == nil {
		return ""
	}
	if text, ok := raw.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func intMapValue(m map[string]interface{}, key string) int {
	if m == nil {
		return 0
	}
	raw, ok := m[key]
	if !ok || raw == nil {
		return 0
	}
	switch v := raw.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case string:
		var out int
		_, _ = fmt.Sscanf(strings.TrimSpace(v), "%d", &out)
		return out
	default:
		return 0
	}
}
