package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/srikarm/image-factory/internal/domain/build"
	"go.uber.org/zap"
)

type BuildRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewBuildRepository creates a new PostgreSQL build repository
func NewBuildRepository(db *sqlx.DB, logger *zap.Logger) build.Repository {
	return &BuildRepository{
		db:     db,
		logger: logger,
	}
}

// Save persists a build to the database
func (r *BuildRepository) Save(ctx context.Context, b *build.Build) error {
	r.logger.Info("Saving build", zap.String("build_id", b.ID().String()), zap.String("tenant_id", b.TenantID().String()))

	// Get the next build number for this project
	var buildNumber int
	countQuery := `SELECT COALESCE(MAX(build_number), 0) + 1 FROM builds WHERE project_id = $1`
	err := r.db.GetContext(ctx, &buildNumber, countQuery, b.ProjectID())
	if err != nil {
		r.logger.Error("Failed to get next build number", zap.Error(err), zap.String("project_id", b.ProjectID().String()))
		return fmt.Errorf("failed to get build number: %w", err)
	}

	query := `
		INSERT INTO builds (id, tenant_id, project_id, build_number, status, created_at, updated_at,
		                   triggered_by_user_id,
		                   infrastructure_type, infrastructure_reason, infrastructure_provider_id, selected_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			triggered_by_user_id = EXCLUDED.triggered_by_user_id,
			updated_at = EXCLUDED.updated_at,
			infrastructure_type = EXCLUDED.infrastructure_type,
			infrastructure_reason = EXCLUDED.infrastructure_reason,
			infrastructure_provider_id = EXCLUDED.infrastructure_provider_id,
			selected_at = EXCLUDED.selected_at`

	now := time.Now().UTC()

	_, err = r.db.ExecContext(ctx, query,
		b.ID(),
		b.TenantID(),
		b.ProjectID(),
		buildNumber,
		string(b.Status()),
		now,
		now,
		b.CreatedBy(),
		r.nullStringFromString(b.InfrastructureType()),
		r.nullStringFromString(b.InfrastructureReason()),
		b.InfrastructureProviderID(),
		r.nullTimeFromTime(b.SelectedAt()),
	)

	if err != nil {
		r.logger.Error("Failed to save build", zap.Error(err), zap.String("build_id", b.ID().String()))
		return fmt.Errorf("failed to save build: %w", err)
	}

	r.logger.Info("Build saved successfully", zap.String("build_id", b.ID().String()), zap.Int("build_number", buildNumber))
	return nil
}

// FindByID retrieves a build by ID
func (r *BuildRepository) FindByID(ctx context.Context, id uuid.UUID) (*build.Build, error) {
	query := `
		SELECT id, tenant_id, project_id, build_number, git_branch, triggered_by_user_id,
			   status, created_at, started_at, completed_at, error_message, updated_at,
			   infrastructure_type, infrastructure_reason, infrastructure_provider_id, selected_at,
			   dispatch_attempts, dispatch_next_run_at
		FROM builds
		WHERE id = $1`

	var buildData dbBuild
	err := r.db.GetContext(ctx, &buildData, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("Failed to find build by ID", zap.Error(err), zap.String("build_id", id.String()))
		return nil, fmt.Errorf("failed to find build: %w", err)
	}

	var config *build.BuildConfigData
	cfg, err := r.GetBuildConfig(ctx, buildData.ID)
	if err != nil {
		r.logger.Warn("Failed to load build config", zap.Error(err), zap.String("build_id", buildData.ID.String()))
	} else {
		config = cfg
	}

	manifest := r.buildManifestFromDB(buildData, config)
	b := build.NewBuildFromDB(
		buildData.ID,
		buildData.TenantID,
		buildData.ProjectID,
		manifest,
		build.BuildStatus(buildData.Status),
		buildData.CreatedAt,
		buildData.UpdatedAt,
		buildData.TriggeredByUserID,
	)
	errorMessage := ""
	if buildData.ErrorMessage != nil {
		errorMessage = *buildData.ErrorMessage
	}
	b.RestoreLifecycleState(buildData.StartedAt, buildData.CompletedAt, errorMessage)

	// Restore infrastructure selection fields (these are stored outside the manifest and
	// are required for executor selection at runtime).
	if buildData.InfrastructureType != nil && buildData.InfrastructureReason != nil {
		b.SetInfrastructureSelectionWithProvider(*buildData.InfrastructureType, *buildData.InfrastructureReason, buildData.InfrastructureProviderID)
	} else if buildData.InfrastructureType != nil {
		// Defensive: allow restoring type even if reason is missing.
		b.SetInfrastructureSelectionWithProvider(*buildData.InfrastructureType, "restored", buildData.InfrastructureProviderID)
	} else if buildData.InfrastructureProviderID != nil {
		b.SetInfrastructureProviderID(buildData.InfrastructureProviderID)
	}
	b.SetDispatchState(buildData.DispatchAttempts, buildData.DispatchNextRunAt)
	if config != nil {
		b.SetConfig(config)
	}

	return b, nil
}

// FindByIDsBatch retrieves multiple builds by IDs in a single query (avoids N+1)
func (r *BuildRepository) FindByIDsBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*build.Build, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*build.Build), nil
	}

	query := `
		SELECT id, tenant_id, project_id, build_number, git_branch, triggered_by_user_id,
			   status, created_at, updated_at,
			   infrastructure_type, infrastructure_reason, infrastructure_provider_id, selected_at
		FROM builds
		WHERE id = ANY($1)
		ORDER BY created_at DESC`

	var buildDataList []dbBuild
	err := r.db.SelectContext(ctx, &buildDataList, query, pq.Array(ids))
	if err != nil {
		r.logger.Error("Failed to find builds in batch",
			zap.Int("build_count", len(ids)),
			zap.Error(err))
		return nil, fmt.Errorf("failed to find builds in batch: %w", err)
	}

	// Create result map
	result := make(map[uuid.UUID]*build.Build)
	configByBuildID, cfgErr := r.getBuildConfigsByBuildIDs(ctx, ids)
	if cfgErr != nil {
		r.logger.Warn("Failed to load build configs for batch build lookup", zap.Error(cfgErr))
	}
	for _, buildData := range buildDataList {
		result[buildData.ID] = r.buildFromDBWithConfig(buildData, configByBuildID[buildData.ID])
	}

	return result, nil
}

// FindByTenantID retrieves builds for a tenant with pagination.
func (r *BuildRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*build.Build, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}

	query := `
		SELECT b.id, b.project_id, b.build_number, b.git_branch, b.triggered_by_user_id, b.status, b.created_at, b.updated_at, p.tenant_id
		FROM builds b
		JOIN projects p ON b.project_id = p.id
		WHERE p.tenant_id = $1
		ORDER BY b.created_at DESC
		LIMIT $2 OFFSET $3`
	args := []interface{}{tenantID, limit, offset}

	var buildData []dbBuild
	err := r.db.SelectContext(ctx, &buildData, query, args...)
	if err != nil {
		r.logger.Error("Failed to find builds by tenant ID", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return nil, fmt.Errorf("failed to find builds by tenant: %w", err)
	}

	builds := make([]*build.Build, len(buildData))
	buildIDs := make([]uuid.UUID, 0, len(buildData))
	for _, data := range buildData {
		buildIDs = append(buildIDs, data.ID)
	}
	configByBuildID, cfgErr := r.getBuildConfigsByBuildIDs(ctx, buildIDs)
	if cfgErr != nil {
		r.logger.Warn("Failed to load build configs for tenant build list", zap.Error(cfgErr), zap.String("tenant_id", tenantID.String()))
	}
	for i, data := range buildData {
		builds[i] = r.buildFromDBWithConfig(data, configByBuildID[data.ID])
	}

	return builds, nil
}

// FindAll retrieves builds across all tenants with pagination.
func (r *BuildRepository) FindAll(ctx context.Context, limit, offset int) ([]*build.Build, error) {
	query := `
		SELECT id, tenant_id, project_id, build_number, git_branch, triggered_by_user_id,
			   status, created_at, updated_at
		FROM builds
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	var buildData []dbBuild
	err := r.db.SelectContext(ctx, &buildData, query, limit, offset)
	if err != nil {
		r.logger.Error("Failed to find builds across all tenants", zap.Error(err))
		return nil, fmt.Errorf("failed to find builds across all tenants: %w", err)
	}

	builds := make([]*build.Build, len(buildData))
	buildIDs := make([]uuid.UUID, 0, len(buildData))
	for _, data := range buildData {
		buildIDs = append(buildIDs, data.ID)
	}
	configByBuildID, cfgErr := r.getBuildConfigsByBuildIDs(ctx, buildIDs)
	if cfgErr != nil {
		r.logger.Warn("Failed to load build configs for all-tenant build list", zap.Error(cfgErr))
	}
	for i, data := range buildData {
		builds[i] = r.buildFromDBWithConfig(data, configByBuildID[data.ID])
	}

	return builds, nil
}

// FindByProjectID retrieves builds for a project with pagination
func (r *BuildRepository) FindByProjectID(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*build.Build, error) {
	query := `
		SELECT id, tenant_id, project_id, build_number, git_branch, triggered_by_user_id,
			   status, created_at, updated_at
		FROM builds
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	var buildData []dbBuild
	err := r.db.SelectContext(ctx, &buildData, query, projectID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to find builds by project ID", zap.Error(err), zap.String("project_id", projectID.String()))
		return nil, fmt.Errorf("failed to find builds by project: %w", err)
	}

	builds := make([]*build.Build, len(buildData))
	buildIDs := make([]uuid.UUID, 0, len(buildData))
	for _, data := range buildData {
		buildIDs = append(buildIDs, data.ID)
	}
	configByBuildID, cfgErr := r.getBuildConfigsByBuildIDs(ctx, buildIDs)
	if cfgErr != nil {
		r.logger.Warn("Failed to load build configs for project build list", zap.Error(cfgErr), zap.String("project_id", projectID.String()))
	}
	for i, data := range buildData {
		builds[i] = r.buildFromDBWithConfig(data, configByBuildID[data.ID])
	}

	return builds, nil
}

// FindByStatus retrieves builds by status with pagination
func (r *BuildRepository) FindByStatus(ctx context.Context, status build.BuildStatus, limit, offset int) ([]*build.Build, error) {
	query := `
		SELECT id, tenant_id, project_id, build_number, git_branch, triggered_by_user_id,
			   status, created_at, updated_at
		FROM builds
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	var buildData []dbBuild
	err := r.db.SelectContext(ctx, &buildData, query, status, limit, offset)
	if err != nil {
		r.logger.Error("Failed to find builds by status", zap.Error(err), zap.String("status", string(status)))
		return nil, fmt.Errorf("failed to find builds by status: %w", err)
	}

	builds := make([]*build.Build, len(buildData))
	buildIDs := make([]uuid.UUID, 0, len(buildData))
	for _, data := range buildData {
		buildIDs = append(buildIDs, data.ID)
	}
	configByBuildID, cfgErr := r.getBuildConfigsByBuildIDs(ctx, buildIDs)
	if cfgErr != nil {
		r.logger.Warn("Failed to load build configs for status build list", zap.Error(cfgErr), zap.String("status", string(status)))
	}
	for i, data := range buildData {
		builds[i] = r.buildFromDBWithConfig(data, configByBuildID[data.ID])
	}

	return builds, nil
}

// Update modifies an existing build in the database
func (r *BuildRepository) Update(ctx context.Context, b *build.Build) error {
	query := `
		UPDATE builds SET
			status = $2,
			started_at = $3,
			completed_at = $4,
			error_message = $5,
			updated_at = $6
		WHERE id = $1`

	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, query,
		b.ID(),
		string(b.Status()),
		b.StartedAt(),
		b.CompletedAt(),
		r.nullStringFromString(b.ErrorMessage()),
		now,
	)
	if err != nil {
		r.logger.Error("Failed to update build", zap.Error(err), zap.String("build_id", b.ID().String()))
		return fmt.Errorf("failed to update build: %w", err)
	}

	r.logger.Info("Build updated successfully", zap.String("build_id", b.ID().String()))
	return nil
}

// UpdateStatus updates the status and execution timestamps for a build
func (r *BuildRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status build.BuildStatus, startedAt, completedAt *time.Time, errorMessage *string) error {
	query := `
		UPDATE builds SET
			status = $2,
			started_at = $3,
			completed_at = $4,
			error_message = $5,
			updated_at = $6
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		id,
		string(status),
		startedAt,
		completedAt,
		errorMessage,
		time.Now().UTC(),
	)
	if err != nil {
		r.logger.Error("Failed to update build status", zap.Error(err), zap.String("build_id", id.String()))
		return fmt.Errorf("failed to update build status: %w", err)
	}

	return nil
}

// ClaimNextQueuedBuild atomically claims the next queued build and marks it running
func (r *BuildRepository) ClaimNextQueuedBuild(ctx context.Context) (*build.Build, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	query := `
		SELECT id, tenant_id, project_id, build_number, git_branch, triggered_by_user_id,
			   status, created_at, updated_at,
			   infrastructure_type, infrastructure_reason, infrastructure_provider_id, selected_at,
			   dispatch_attempts, dispatch_next_run_at
		FROM builds
		WHERE status = $1
		  AND (dispatch_next_run_at IS NULL OR dispatch_next_run_at <= NOW())
		ORDER BY created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1`

	var buildData dbBuild
	if err = tx.GetContext(ctx, &buildData, query, string(build.BuildStatusQueued)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = tx.Rollback()
			return nil, nil
		}
		return nil, fmt.Errorf("failed to select queued build: %w", err)
	}

	var config *build.BuildConfigData
	cfg, cfgErr := r.GetBuildConfig(ctx, buildData.ID)
	if cfgErr != nil {
		r.logger.Warn("Failed to load build config during claim", zap.Error(cfgErr), zap.String("build_id", buildData.ID.String()))
	} else {
		config = cfg
	}

	manifest := r.buildManifestFromDB(buildData, config)
	b := build.NewBuildFromDB(
		buildData.ID,
		buildData.TenantID,
		buildData.ProjectID,
		manifest,
		build.BuildStatus(buildData.Status),
		buildData.CreatedAt,
		buildData.UpdatedAt,
		buildData.TriggeredByUserID,
	)

	// Restore infrastructure selection fields (required for executor selection).
	if buildData.InfrastructureType != nil && buildData.InfrastructureReason != nil {
		b.SetInfrastructureSelectionWithProvider(*buildData.InfrastructureType, *buildData.InfrastructureReason, buildData.InfrastructureProviderID)
	} else if buildData.InfrastructureType != nil {
		b.SetInfrastructureSelectionWithProvider(*buildData.InfrastructureType, "restored", buildData.InfrastructureProviderID)
	} else if buildData.InfrastructureProviderID != nil {
		b.SetInfrastructureProviderID(buildData.InfrastructureProviderID)
	}
	b.SetDispatchState(buildData.DispatchAttempts+1, nil)
	if config != nil {
		b.SetConfig(config)
	}

	if err = b.Start(); err != nil {
		return nil, fmt.Errorf("failed to start build in memory: %w", err)
	}

	updateQuery := `
		UPDATE builds
		SET status = $2,
		    started_at = $3,
		    updated_at = $4,
		    dispatch_attempts = dispatch_attempts + 1,
		    dispatch_next_run_at = NULL
		WHERE id = $1 AND status = $5`
	result, err := tx.ExecContext(ctx, updateQuery,
		b.ID(),
		string(build.BuildStatusRunning),
		b.StartedAt(),
		time.Now().UTC(),
		string(build.BuildStatusQueued),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to claim build: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to read claim result: %w", err)
	}
	if rows == 0 {
		_ = tx.Rollback()
		return nil, nil
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit claim: %w", err)
	}

	return b, nil
}

// RequeueBuild returns a build to the queue with a next run time.
func (r *BuildRepository) RequeueBuild(ctx context.Context, id uuid.UUID, nextRunAt time.Time, errorMessage *string) error {
	query := `
		UPDATE builds
		SET status = $2,
		    started_at = NULL,
		    completed_at = NULL,
		    error_message = $3,
		    dispatch_next_run_at = $4,
		    updated_at = $5
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		id,
		string(build.BuildStatusQueued),
		errorMessage,
		nextRunAt,
		time.Now().UTC(),
	)
	if err != nil {
		r.logger.Error("Failed to requeue build", zap.Error(err), zap.String("build_id", id.String()))
		return fmt.Errorf("failed to requeue build: %w", err)
	}
	return nil
}

// Delete removes a build
func (r *BuildRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM builds WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete build", zap.Error(err), zap.String("build_id", id.String()))
		return fmt.Errorf("failed to delete build: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("build not found")
	}

	r.logger.Info("Build deleted successfully", zap.String("build_id", id.String()))
	return nil
}

// CountByTenantID counts builds for a tenant.
func (r *BuildRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	if tenantID == uuid.Nil {
		return 0, fmt.Errorf("tenant_id is required")
	}
	query := `SELECT COUNT(*) FROM builds b JOIN projects p ON b.project_id = p.id WHERE p.tenant_id = $1`
	args := []interface{}{tenantID}

	var count int
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		r.logger.Error("Failed to count builds by tenant ID", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return 0, fmt.Errorf("failed to count builds: %w", err)
	}

	return count, nil
}

// CountAll counts builds across all tenants.
func (r *BuildRepository) CountAll(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM builds`

	var count int
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		r.logger.Error("Failed to count builds across all tenants", zap.Error(err))
		return 0, fmt.Errorf("failed to count builds across all tenants: %w", err)
	}

	return count, nil
}

// CountByStatus counts builds by status for a tenant
func (r *BuildRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID, status build.BuildStatus) (int, error) {
	query := `SELECT COUNT(*) FROM builds WHERE tenant_id = $1 AND status = $2`

	var count int
	err := r.db.GetContext(ctx, &count, query, tenantID, status)
	if err != nil {
		r.logger.Error("Failed to count builds by status", zap.Error(err), zap.String("tenant_id", tenantID.String()), zap.String("status", string(status)))
		return 0, fmt.Errorf("failed to count builds by status: %w", err)
	}

	return count, nil
}

// CountByProjectID counts builds for a project
func (r *BuildRepository) CountByProjectID(ctx context.Context, projectID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM builds WHERE project_id = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, projectID)
	if err != nil {
		r.logger.Error("Failed to count builds by project ID", zap.Error(err), zap.String("project_id", projectID.String()))
		return 0, fmt.Errorf("failed to count builds by project: %w", err)
	}

	return count, nil
}

// FindRunningBuilds retrieves all running builds
func (r *BuildRepository) FindRunningBuilds(ctx context.Context) ([]*build.Build, error) {
	query := `
		SELECT id, tenant_id, project_id, build_number, git_branch, triggered_by_user_id,
			   status, created_at, updated_at
		FROM builds
		WHERE status IN ('queued', 'running', 'in_progress')
		ORDER BY created_at ASC`

	var buildData []dbBuild
	err := r.db.SelectContext(ctx, &buildData, query)
	if err != nil {
		r.logger.Error("Failed to find running builds", zap.Error(err))
		return nil, fmt.Errorf("failed to find running builds: %w", err)
	}

	builds := make([]*build.Build, len(buildData))
	buildIDs := make([]uuid.UUID, 0, len(buildData))
	for _, data := range buildData {
		buildIDs = append(buildIDs, data.ID)
	}
	configByBuildID, cfgErr := r.getBuildConfigsByBuildIDs(ctx, buildIDs)
	if cfgErr != nil {
		r.logger.Warn("Failed to load build configs for running build list", zap.Error(cfgErr))
	}
	for i, data := range buildData {
		builds[i] = r.buildFromDBWithConfig(data, configByBuildID[data.ID])
	}

	return builds, nil
}

// SaveBuildConfig persists build configuration to the database
func (r *BuildRepository) SaveBuildConfig(ctx context.Context, config *build.BuildConfigData) error {
	if config == nil {
		return errors.New("build config is required")
	}

	if err := config.Validate(); err != nil {
		r.logger.Error("Invalid build config", zap.Error(err), zap.String("build_id", config.BuildID.String()))
		return fmt.Errorf("invalid build config: %w", err)
	}

	query := `
		INSERT INTO build_configs (
			id, build_id, source_id, ref_policy, fixed_ref, build_method, sbom_tool, scan_tool, registry_type, secret_manager_type, build_args, environment, secrets, metadata,
			dockerfile, build_context, cache_enabled, cache_repo,
			platforms, cache_from, cache_to,
			target_stage,
			builder, buildpacks,
			packer_template,
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18,
			$19, $20, $21,
			$22,
			$23, $24,
			$25,
			$26, $27
		)
		ON CONFLICT (build_id) DO UPDATE SET
			source_id = EXCLUDED.source_id,
			ref_policy = EXCLUDED.ref_policy,
			fixed_ref = EXCLUDED.fixed_ref,
			build_method = EXCLUDED.build_method,
			sbom_tool = EXCLUDED.sbom_tool,
			scan_tool = EXCLUDED.scan_tool,
			registry_type = EXCLUDED.registry_type,
			secret_manager_type = EXCLUDED.secret_manager_type,
			build_args = EXCLUDED.build_args,
			environment = EXCLUDED.environment,
			secrets = EXCLUDED.secrets,
			metadata = EXCLUDED.metadata,
			dockerfile = EXCLUDED.dockerfile,
			build_context = EXCLUDED.build_context,
			cache_enabled = EXCLUDED.cache_enabled,
			cache_repo = EXCLUDED.cache_repo,
			platforms = EXCLUDED.platforms,
			cache_from = EXCLUDED.cache_from,
			cache_to = EXCLUDED.cache_to,
			target_stage = EXCLUDED.target_stage,
			builder = EXCLUDED.builder,
			buildpacks = EXCLUDED.buildpacks,
			packer_template = EXCLUDED.packer_template,
			updated_at = EXCLUDED.updated_at`

	now := time.Now().UTC()
	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}

	// Convert slices and maps to JSON
	buildArgsJSON, _ := json.Marshal(config.BuildArgs)
	envJSON, _ := json.Marshal(config.Environment)
	secretsJSON, _ := json.Marshal(config.Secrets)
	metadataJSON, _ := json.Marshal(config.Metadata)
	platformsJSON, _ := json.Marshal(config.Platforms)
	cacheFromJSON, _ := json.Marshal(config.CacheFrom)
	buildpacksJSON, _ := json.Marshal(config.Buildpacks)

	_, err := r.db.ExecContext(ctx, query,
		config.ID,
		config.BuildID,
		config.SourceID,
		config.RefPolicy,
		nullIfEmpty(config.FixedRef),
		config.BuildMethod,
		config.SBOMTool,
		config.ScanTool,
		config.RegistryType,
		config.SecretManagerType,
		buildArgsJSON,
		envJSON,
		secretsJSON,
		metadataJSON,
		config.Dockerfile,
		config.BuildContext,
		config.CacheEnabled,
		config.CacheRepo,
		platformsJSON,
		cacheFromJSON,
		config.CacheTo,
		config.TargetStage,
		config.Builder,
		buildpacksJSON,
		config.PackerTemplate,
		now,
		now,
	)

	if err != nil {
		r.logger.Error("Failed to save build config",
			zap.Error(err),
			zap.String("build_id", config.BuildID.String()),
			zap.String("build_method", config.BuildMethod))
		return fmt.Errorf("failed to save build config: %w", err)
	}

	r.logger.Info("Build config saved successfully",
		zap.String("build_id", config.BuildID.String()),
		zap.String("build_method", config.BuildMethod))
	return nil
}

// GetBuildConfig retrieves build configuration by build ID
func (r *BuildRepository) GetBuildConfig(ctx context.Context, buildID uuid.UUID) (*build.BuildConfigData, error) {
	query := `
		SELECT 
			id, build_id, source_id, ref_policy, fixed_ref, build_method, sbom_tool, scan_tool, registry_type, secret_manager_type, build_args, environment, secrets, metadata,
			dockerfile, build_context, cache_enabled, cache_repo,
			platforms, cache_from, cache_to,
			target_stage,
			builder, buildpacks,
			packer_template,
			created_at, updated_at
		FROM build_configs
		WHERE build_id = $1`

	var dbConfig dbBuildConfig
	err := r.db.GetContext(ctx, &dbConfig, query, buildID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Debug("Build config not found", zap.String("build_id", buildID.String()))
			return nil, nil
		}
		r.logger.Error("Failed to get build config",
			zap.Error(err),
			zap.String("build_id", buildID.String()))
		return nil, fmt.Errorf("failed to get build config: %w", err)
	}

	config := r.dbBuildConfigToConfig(dbConfig)
	return config, nil
}

// UpdateBuildConfig updates existing build configuration
func (r *BuildRepository) UpdateBuildConfig(ctx context.Context, config *build.BuildConfigData) error {
	if config == nil {
		return errors.New("build config is required")
	}

	if config.ID == uuid.Nil {
		return errors.New("config ID is required for update")
	}

	if err := config.Validate(); err != nil {
		r.logger.Error("Invalid build config", zap.Error(err), zap.String("build_id", config.BuildID.String()))
		return fmt.Errorf("invalid build config: %w", err)
	}

	query := `
		UPDATE build_configs
		SET source_id = $2, ref_policy = $3, fixed_ref = $4, build_method = $5, sbom_tool = $6, scan_tool = $7, registry_type = $8, secret_manager_type = $9,
		    build_args = $10, environment = $11, secrets = $12, metadata = $13,
		    dockerfile = $14, build_context = $15, cache_enabled = $16, cache_repo = $17,
		    platforms = $18, cache_from = $19, cache_to = $20,
		    target_stage = $21,
		    builder = $22, buildpacks = $23,
		    packer_template = $24,
		    updated_at = $25
		WHERE id = $1`

	now := time.Now().UTC()

	// Convert slices and maps to JSON
	buildArgsJSON, _ := json.Marshal(config.BuildArgs)
	envJSON, _ := json.Marshal(config.Environment)
	secretsJSON, _ := json.Marshal(config.Secrets)
	metadataJSON, _ := json.Marshal(config.Metadata)
	platformsJSON, _ := json.Marshal(config.Platforms)
	cacheFromJSON, _ := json.Marshal(config.CacheFrom)
	buildpacksJSON, _ := json.Marshal(config.Buildpacks)

	result, err := r.db.ExecContext(ctx, query,
		config.ID,
		config.SourceID,
		config.RefPolicy,
		nullIfEmpty(config.FixedRef),
		config.BuildMethod,
		config.SBOMTool,
		config.ScanTool,
		config.RegistryType,
		config.SecretManagerType,
		buildArgsJSON,
		envJSON,
		secretsJSON,
		metadataJSON,
		config.Dockerfile,
		config.BuildContext,
		config.CacheEnabled,
		config.CacheRepo,
		platformsJSON,
		cacheFromJSON,
		config.CacheTo,
		config.TargetStage,
		config.Builder,
		buildpacksJSON,
		config.PackerTemplate,
		now,
	)

	if err != nil {
		r.logger.Error("Failed to update build config",
			zap.Error(err),
			zap.String("config_id", config.ID.String()))
		return fmt.Errorf("failed to update build config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("build config not found: %s", config.ID)
	}

	r.logger.Info("Build config updated successfully", zap.String("config_id", config.ID.String()))
	return nil
}

// DeleteBuildConfig deletes build configuration
func (r *BuildRepository) DeleteBuildConfig(ctx context.Context, buildID uuid.UUID) error {
	query := `DELETE FROM build_configs WHERE build_id = $1`

	result, err := r.db.ExecContext(ctx, query, buildID)
	if err != nil {
		r.logger.Error("Failed to delete build config",
			zap.Error(err),
			zap.String("build_id", buildID.String()))
		return fmt.Errorf("failed to delete build config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		r.logger.Info("Build config deleted successfully", zap.String("build_id", buildID.String()))
	}

	return nil
}

// dbBuild represents the database structure for builds
type dbBuild struct {
	ID                       uuid.UUID  `db:"id"`
	BuildNumber              int        `db:"build_number"`
	GitBranch                *string    `db:"git_branch"`
	TenantID                 uuid.UUID  `db:"tenant_id"`
	ProjectID                uuid.UUID  `db:"project_id"`
	TriggeredByUserID        *uuid.UUID `db:"triggered_by_user_id"`
	Status                   string     `db:"status"`
	CreatedAt                time.Time  `db:"created_at"`
	StartedAt                *time.Time `db:"started_at"`
	CompletedAt              *time.Time `db:"completed_at"`
	ErrorMessage             *string    `db:"error_message"`
	UpdatedAt                time.Time  `db:"updated_at"`
	InfrastructureType       *string    `db:"infrastructure_type"`
	InfrastructureReason     *string    `db:"infrastructure_reason"`
	InfrastructureProviderID *uuid.UUID `db:"infrastructure_provider_id"`
	SelectedAt               *time.Time `db:"selected_at"`
	DispatchAttempts         int        `db:"dispatch_attempts"`
	DispatchNextRunAt        *time.Time `db:"dispatch_next_run_at"`
}

// dbBuildConfig represents the database structure for build configs
type dbBuildConfig struct {
	ID                uuid.UUID  `db:"id"`
	BuildID           uuid.UUID  `db:"build_id"`
	SourceID          *uuid.UUID `db:"source_id"`
	RefPolicy         string     `db:"ref_policy"`
	FixedRef          *string    `db:"fixed_ref"`
	BuildMethod       string     `db:"build_method"`
	SBOMTool          *string    `db:"sbom_tool"`
	ScanTool          *string    `db:"scan_tool"`
	RegistryType      *string    `db:"registry_type"`
	SecretManagerType *string    `db:"secret_manager_type"`
	BuildArgs         []byte     `db:"build_args"`
	Environment       []byte     `db:"environment"`
	Secrets           []byte     `db:"secrets"`
	Metadata          []byte     `db:"metadata"`
	Dockerfile        *string    `db:"dockerfile"`
	BuildContext      *string    `db:"build_context"`
	CacheEnabled      bool       `db:"cache_enabled"`
	CacheRepo         *string    `db:"cache_repo"`
	Platforms         []byte     `db:"platforms"`
	CacheFrom         []byte     `db:"cache_from"`
	CacheTo           *string    `db:"cache_to"`
	TargetStage       *string    `db:"target_stage"`
	Builder           *string    `db:"builder"`
	Buildpacks        []byte     `db:"buildpacks"`
	PackerTemplate    *string    `db:"packer_template"`
	CreatedAt         time.Time  `db:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"`
}

// dbBuildConfigToConfig converts database representation to domain build config
func (r *BuildRepository) dbBuildConfigToConfig(db dbBuildConfig) *build.BuildConfigData {
	config := &build.BuildConfigData{
		ID:           db.ID,
		BuildID:      db.BuildID,
		SourceID:     db.SourceID,
		RefPolicy:    db.RefPolicy,
		BuildMethod:  db.BuildMethod,
		CacheEnabled: db.CacheEnabled,
		CreatedAt:    db.CreatedAt,
		UpdatedAt:    db.UpdatedAt,
	}
	if db.FixedRef != nil {
		config.FixedRef = *db.FixedRef
	}

	// Unmarshal JSON fields
	if len(db.BuildArgs) > 0 {
		json.Unmarshal(db.BuildArgs, &config.BuildArgs)
	}
	if len(db.Environment) > 0 {
		json.Unmarshal(db.Environment, &config.Environment)
	}
	if len(db.Secrets) > 0 {
		json.Unmarshal(db.Secrets, &config.Secrets)
	}
	if len(db.Metadata) > 0 {
		json.Unmarshal(db.Metadata, &config.Metadata)
	}
	if len(db.Platforms) > 0 {
		json.Unmarshal(db.Platforms, &config.Platforms)
	}
	if len(db.CacheFrom) > 0 {
		json.Unmarshal(db.CacheFrom, &config.CacheFrom)
	}
	if len(db.Buildpacks) > 0 {
		json.Unmarshal(db.Buildpacks, &config.Buildpacks)
	}

	if db.SBOMTool != nil {
		config.SBOMTool = build.SBOMTool(*db.SBOMTool)
	}
	if db.ScanTool != nil {
		config.ScanTool = build.ScanTool(*db.ScanTool)
	}
	if db.RegistryType != nil {
		config.RegistryType = build.RegistryType(*db.RegistryType)
	}
	if db.SecretManagerType != nil {
		config.SecretManagerType = build.SecretManagerType(*db.SecretManagerType)
	}

	// Set optional string fields
	if db.Dockerfile != nil {
		config.Dockerfile = *db.Dockerfile
	}
	if db.BuildContext != nil {
		config.BuildContext = *db.BuildContext
	}
	if db.CacheRepo != nil {
		config.CacheRepo = *db.CacheRepo
	}
	if db.CacheTo != nil {
		config.CacheTo = *db.CacheTo
	}
	if db.TargetStage != nil {
		config.TargetStage = *db.TargetStage
	}
	if db.Builder != nil {
		config.Builder = *db.Builder
	}
	if db.PackerTemplate != nil {
		config.PackerTemplate = *db.PackerTemplate
	}

	return config
}

// buildToDB converts a domain build to database representation
func (r *BuildRepository) buildToDB(b *build.Build) (dbBuild, error) {
	// TODO: Implement buildToDB to match current database schema
	return dbBuild{}, fmt.Errorf("buildToDB not implemented for current database schema")
}

// nullStringFromString converts a string to *string (nil if empty)
func (r *BuildRepository) nullStringFromString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// nullTimeFromTime converts a *time.Time to *pq.NullTime
func (r *BuildRepository) nullTimeFromTime(t *time.Time) *pq.NullTime {
	if t == nil {
		return nil
	}
	return &pq.NullTime{Time: *t, Valid: true}
}

// buildFromDB converts database representation to domain build
// NOTE: This creates a minimal build object since the current DB schema doesn't store full manifests
func (r *BuildRepository) buildFromDB(data dbBuild) *build.Build {
	return r.buildFromDBWithConfig(data, nil)
}

func (r *BuildRepository) buildFromDBWithConfig(data dbBuild, config *build.BuildConfigData) *build.Build {
	// Create a minimal manifest from available data
	manifest := r.buildManifestFromDB(data, config)

	// Create minimal build without manifest validation (DB doesn't store full manifest)
	b := build.NewBuildFromDB(
		data.ID,
		data.TenantID,
		data.ProjectID,
		manifest,
		build.BuildStatus(data.Status),
		data.CreatedAt,
		data.UpdatedAt,
		data.TriggeredByUserID,
	)
	errorMessage := ""
	if data.ErrorMessage != nil {
		errorMessage = *data.ErrorMessage
	}
	b.RestoreLifecycleState(data.StartedAt, data.CompletedAt, errorMessage)

	// Set infrastructure selection if available
	if data.InfrastructureType != nil && data.InfrastructureReason != nil {
		b.SetInfrastructureSelectionWithProvider(*data.InfrastructureType, *data.InfrastructureReason, data.InfrastructureProviderID)
		// Override the selected_at timestamp if it was set
		if data.SelectedAt != nil {
			// Note: In a complete implementation, we'd need a method to set the selected_at timestamp
			// For now, the SetInfrastructureSelection method sets it to now
		}
	}
	if data.InfrastructureType == nil && data.InfrastructureProviderID != nil {
		b.SetInfrastructureProviderID(data.InfrastructureProviderID)
	}
	b.SetDispatchState(data.DispatchAttempts, data.DispatchNextRunAt)
	if config != nil {
		b.SetConfig(config)
	}

	// Note: In a complete implementation, we would need domain methods to restore
	// the build to its previous state (status, timestamps, results, etc.)
	// For now, this returns a build in pending state

	return b
}

func (r *BuildRepository) getBuildConfigsByBuildIDs(ctx context.Context, buildIDs []uuid.UUID) (map[uuid.UUID]*build.BuildConfigData, error) {
	configs := make(map[uuid.UUID]*build.BuildConfigData, len(buildIDs))
	if len(buildIDs) == 0 {
		return configs, nil
	}

	query := `
		SELECT
			id, build_id, source_id, ref_policy, fixed_ref, build_method, sbom_tool, scan_tool, registry_type, secret_manager_type, build_args, environment, secrets, metadata,
			dockerfile, build_context, cache_enabled, cache_repo,
			platforms, cache_from, cache_to,
			target_stage,
			builder, buildpacks,
			packer_template,
			created_at, updated_at
		FROM build_configs
		WHERE build_id = ANY($1)
		ORDER BY updated_at DESC`

	rows := make([]dbBuildConfig, 0, len(buildIDs))
	if err := r.db.SelectContext(ctx, &rows, query, pq.Array(buildIDs)); err != nil {
		return nil, fmt.Errorf("failed to list build configs: %w", err)
	}

	for _, row := range rows {
		// Keep the most recent config per build_id when multiple rows exist.
		if _, exists := configs[row.BuildID]; exists {
			continue
		}
		configs[row.BuildID] = r.dbBuildConfigToConfig(row)
	}

	return configs, nil
}

func (r *BuildRepository) buildManifestFromDB(data dbBuild, config *build.BuildConfigData) build.BuildManifest {
	manifest := build.BuildManifest{
		Name: fmt.Sprintf("Build #%d", data.BuildNumber),
		Type: build.BuildTypeContainer,
	}
	if data.InfrastructureType != nil {
		manifest.InfrastructureType = *data.InfrastructureType
	}
	if data.InfrastructureProviderID != nil {
		manifest.InfrastructureProviderID = data.InfrastructureProviderID
	}

	if config == nil {
		return manifest
	}
	if config.SourceID != nil {
		if manifest.Metadata == nil {
			manifest.Metadata = map[string]interface{}{}
		}
		manifest.Metadata["source_id"] = config.SourceID.String()
	}
	if config.RefPolicy != "" {
		if manifest.Metadata == nil {
			manifest.Metadata = map[string]interface{}{}
		}
		manifest.Metadata["ref_policy"] = config.RefPolicy
	}
	if config.FixedRef != "" {
		if manifest.Metadata == nil {
			manifest.Metadata = map[string]interface{}{}
		}
		manifest.Metadata["fixed_ref"] = config.FixedRef
	}

	buildType := build.BuildType(config.BuildMethod)
	if config.BuildMethod == string(build.BuildMethodDocker) {
		buildType = build.BuildTypeContainer
	}
	if buildType != "" {
		manifest.Type = buildType
	}

	manifest.BuildConfig = &build.BuildConfig{
		BuildType:         buildType,
		SBOMTool:          config.SBOMTool,
		ScanTool:          config.ScanTool,
		RegistryType:      config.RegistryType,
		SecretManagerType: config.SecretManagerType,
		Dockerfile:        config.Dockerfile,
		BuildContext:      config.BuildContext,
		BuildArgs:         config.BuildArgs,
		Secrets:           config.Secrets,
		Platforms:         config.Platforms,
		Cache:             config.CacheEnabled,
		CacheRepo:         config.CacheRepo,
		CacheFrom:         config.CacheFrom,
		CacheTo:           config.CacheTo,
		Target:            config.TargetStage,
		PackerTemplate:    config.PackerTemplate,
	}

	if config.Metadata != nil {
		if registryRepo, ok := config.Metadata["registry_repo"].(string); ok {
			manifest.BuildConfig.RegistryRepo = registryRepo
		}
		if registryAuthID, ok := config.Metadata["registry_auth_id"].(string); ok && registryAuthID != "" {
			if parsed, err := uuid.Parse(registryAuthID); err == nil {
				manifest.BuildConfig.RegistryAuthID = &parsed
			}
		}
		if skipUnusedStages, ok := config.Metadata["skip_unused_stages"].(bool); ok {
			manifest.BuildConfig.SkipUnusedStages = skipUnusedStages
		}
		if nixExpression, ok := config.Metadata["nix_expression"].(string); ok {
			manifest.BuildConfig.NixExpression = nixExpression
		}
		if flakeURI, ok := config.Metadata["flake_uri"].(string); ok {
			manifest.BuildConfig.FlakeURI = flakeURI
		}
		if attributes, ok := config.Metadata["attributes"].([]interface{}); ok {
			out := make([]string, 0, len(attributes))
			for _, attr := range attributes {
				if s, ok := attr.(string); ok {
					out = append(out, s)
				}
			}
			manifest.BuildConfig.Attributes = out
		}
		if outputs, ok := config.Metadata["outputs"].(map[string]interface{}); ok {
			out := make(map[string]string, len(outputs))
			for k, v := range outputs {
				if s, ok := v.(string); ok {
					out[k] = s
				}
			}
			manifest.BuildConfig.Outputs = out
		}
		if cacheDir, ok := config.Metadata["cache_dir"].(string); ok {
			manifest.BuildConfig.CacheDir = cacheDir
		}
		if pure, ok := config.Metadata["pure"].(bool); ok {
			manifest.BuildConfig.Pure = pure
		}
		if showTrace, ok := config.Metadata["show_trace"].(bool); ok {
			manifest.BuildConfig.ShowTrace = showTrace
		}
		if gitURL, ok := config.Metadata["git_url"].(string); ok && gitURL != "" {
			if manifest.Metadata == nil {
				manifest.Metadata = map[string]interface{}{}
			}
			manifest.Metadata["git_url"] = gitURL
		}
		if gitBranch, ok := config.Metadata["git_branch"].(string); ok && gitBranch != "" {
			if manifest.Metadata == nil {
				manifest.Metadata = map[string]interface{}{}
			}
			manifest.Metadata["git_branch"] = gitBranch
		}
	}

	if config.Builder != "" || len(config.Buildpacks) > 0 {
		manifest.BuildConfig.PaketoConfig = &build.PaketoConfig{
			Builder:    config.Builder,
			Buildpacks: config.Buildpacks,
		}
		if config.Metadata != nil {
			if env, ok := config.Metadata["env"].(map[string]interface{}); ok {
				envMap := make(map[string]string, len(env))
				for k, v := range env {
					if s, ok := v.(string); ok {
						envMap[k] = s
					}
				}
				manifest.BuildConfig.PaketoConfig.Env = envMap
			}
			if buildArgs, ok := config.Metadata["build_args"].(map[string]interface{}); ok {
				argsMap := make(map[string]string, len(buildArgs))
				for k, v := range buildArgs {
					if s, ok := v.(string); ok {
						argsMap[k] = s
					}
				}
				manifest.BuildConfig.PaketoConfig.BuildArgs = argsMap
			}
		}
	}

	return manifest
}

// UpdateInfrastructureSelection updates the infrastructure selection for a build
func (r *BuildRepository) UpdateInfrastructureSelection(ctx context.Context, build *build.Build) error {
	query := `
		UPDATE builds
		SET infrastructure_type = $2, infrastructure_reason = $3, infrastructure_provider_id = $4, selected_at = $5, updated_at = $6
		WHERE id = $1`

	now := time.Now().UTC()

	result, err := r.db.ExecContext(ctx, query,
		build.ID(),
		r.nullStringFromString(build.InfrastructureType()),
		r.nullStringFromString(build.InfrastructureReason()),
		build.InfrastructureProviderID(),
		r.nullTimeFromTime(build.SelectedAt()),
		now,
	)

	if err != nil {
		r.logger.Error("Failed to update infrastructure selection",
			zap.Error(err),
			zap.String("build_id", build.ID().String()))
		return fmt.Errorf("failed to update infrastructure selection: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("build not found: %s", build.ID())
	}

	r.logger.Info("Infrastructure selection updated successfully", zap.String("build_id", build.ID().String()))
	return nil
}
