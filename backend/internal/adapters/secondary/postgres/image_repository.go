package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/image"
)

const releasedQuarantineImagePredicate = `
NOT EXISTS (
	SELECT 1
	FROM external_image_imports eii
	WHERE eii.tenant_id = catalog_images.tenant_id
	  AND eii.request_type = 'quarantine'
	  AND eii.internal_image_ref IS NOT NULL
	  AND eii.internal_image_ref = catalog_images.repository_url
	  AND COALESCE(NULLIF(eii.release_state, ''), CASE
		WHEN eii.status = 'success' THEN 'ready_for_release'
		WHEN eii.status = 'quarantined' THEN 'release_blocked'
		WHEN eii.status = 'failed' THEN 'release_blocked'
		WHEN eii.status IN ('pending', 'approved', 'importing') THEN 'not_ready'
		ELSE 'unknown'
	  END) <> 'released'
)`

// ImageRepository implements the image.Repository interface
type ImageRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewImageRepository creates a new image repository
func NewImageRepository(db *sqlx.DB, logger *zap.Logger) *ImageRepository {
	return &ImageRepository{
		db:     db,
		logger: logger,
	}
}

// imageModel represents the database model for image
type imageModel struct {
	ID               uuid.UUID       `db:"id"`
	TenantID         uuid.UUID       `db:"tenant_id"`
	Name             string          `db:"name"`
	Description      sql.NullString  `db:"description"`
	Visibility       string          `db:"visibility"`
	Status           string          `db:"status"`
	RepositoryURL    sql.NullString  `db:"repository_url"`
	RegistryProvider sql.NullString  `db:"registry_provider"`
	Architecture     sql.NullString  `db:"architecture"`
	OS               sql.NullString  `db:"os"`
	Language         sql.NullString  `db:"language"`
	Framework        sql.NullString  `db:"framework"`
	Version          sql.NullString  `db:"version"`
	Tags             json.RawMessage `db:"tags"`
	Metadata         json.RawMessage `db:"metadata"`
	SizeBytes        sql.NullInt64   `db:"size_bytes"`
	PullCount        int64           `db:"pull_count"`
	CreatedBy        uuid.UUID       `db:"created_by"`
	UpdatedBy        uuid.UUID       `db:"updated_by"`
	CreatedAt        time.Time       `db:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at"`
	DeprecatedAt     sql.NullTime    `db:"deprecated_at"`
	ArchivedAt       sql.NullTime    `db:"archived_at"`
}

// toDomain converts database model to domain model
func (m *imageModel) toDomain() (*image.Image, error) {
	// Parse metadata
	var metadata map[string]interface{}
	if len(m.Metadata) > 0 {
		if err := json.Unmarshal(m.Metadata, &metadata); err != nil {
			// If unmarshaling fails, use empty metadata
			metadata = make(map[string]interface{})
		}
	} else {
		metadata = make(map[string]interface{})
	}

	// Parse tags
	var tags []string
	if len(m.Tags) > 0 {
		// Try to unmarshal as array first
		if err := json.Unmarshal(m.Tags, &tags); err != nil {
			// If it fails, check if it's an object or other type
			var obj interface{}
			if json.Unmarshal(m.Tags, &obj) == nil {
				// If it's a valid JSON but not an array, treat as empty array
				tags = []string{}
			} else {
				return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
			}
		}
	} else {
		tags = []string{}
	}

	// Convert nullable strings to pointers
	var repositoryURL, registryProvider, architecture, os, language, framework, version *string
	if m.RepositoryURL.Valid {
		repositoryURL = &m.RepositoryURL.String
	}
	if m.RegistryProvider.Valid {
		registryProvider = &m.RegistryProvider.String
	}
	if m.Architecture.Valid {
		architecture = &m.Architecture.String
	}
	if m.OS.Valid {
		os = &m.OS.String
	}
	if m.Language.Valid {
		language = &m.Language.String
	}
	if m.Framework.Valid {
		framework = &m.Framework.String
	}
	if m.Version.Valid {
		version = &m.Version.String
	}

	// Convert nullable int64 to pointer
	var sizeBytes *int64
	if m.SizeBytes.Valid {
		sizeBytes = &m.SizeBytes.Int64
	}

	// Convert nullable times to pointers
	var deprecatedAt, archivedAt *time.Time
	if m.DeprecatedAt.Valid {
		deprecatedAt = &m.DeprecatedAt.Time
	}
	if m.ArchivedAt.Valid {
		archivedAt = &m.ArchivedAt.Time
	}

	// Ensure tags is never nil
	if tags == nil {
		tags = []string{}
	}

	// Reconstruct domain image with all fields
	img, err := image.ReconstructImage(
		m.ID,
		m.TenantID,
		m.Name,
		m.Description.String,
		image.ImageVisibility(m.Visibility),
		image.ImageStatus(m.Status),
		repositoryURL,
		registryProvider,
		architecture,
		os,
		language,
		framework,
		version,
		tags, // Use the parsed tags
		metadata,
		sizeBytes,
		m.PullCount,
		m.CreatedBy,
		m.UpdatedBy,
		m.CreatedAt,
		m.UpdatedAt,
		deprecatedAt,
		archivedAt,
	)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// fromDomain converts domain model to database model
func fromDomain(img *image.Image) *imageModel {
	metadata, _ := json.Marshal(img.Metadata())
	tags, _ := json.Marshal(img.Tags())

	return &imageModel{
		ID:               img.ID(),
		TenantID:         img.TenantID(),
		Name:             img.Name(),
		Description:      sql.NullString{String: img.Description(), Valid: img.Description() != ""},
		Visibility:       string(img.Visibility()),
		Status:           string(img.Status()),
		RepositoryURL:    sql.NullString{String: stringPtrToString(img.RepositoryURL()), Valid: img.RepositoryURL() != nil},
		RegistryProvider: sql.NullString{String: stringPtrToString(img.RegistryProvider()), Valid: img.RegistryProvider() != nil},
		Architecture:     sql.NullString{String: stringPtrToString(img.Architecture()), Valid: img.Architecture() != nil},
		OS:               sql.NullString{String: stringPtrToString(img.OS()), Valid: img.OS() != nil},
		Language:         sql.NullString{String: stringPtrToString(img.Language()), Valid: img.Language() != nil},
		Framework:        sql.NullString{String: stringPtrToString(img.Framework()), Valid: img.Framework() != nil},
		Version:          sql.NullString{String: stringPtrToString(img.Version()), Valid: img.Version() != nil},
		Tags:             tags,
		Metadata:         metadata,
		SizeBytes:        sql.NullInt64{Int64: int64PtrToInt64(img.SizeBytes()), Valid: img.SizeBytes() != nil},
		PullCount:        img.PullCount(),
		CreatedBy:        img.CreatedBy(),
		UpdatedBy:        img.UpdatedBy(),
		CreatedAt:        img.CreatedAt(),
		UpdatedAt:        img.UpdatedAt(),
		DeprecatedAt:     timePtrToNullTime(img.DeprecatedAt()),
		ArchivedAt:       timePtrToNullTime(img.ArchivedAt()),
	}
}

// Helper functions
func stringPtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func int64PtrToInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func timePtrToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// Save persists an image
func (r *ImageRepository) Save(ctx context.Context, img *image.Image) error {
	model := fromDomain(img)

	query := `
		INSERT INTO catalog_images (
			id, tenant_id, name, description, visibility, status,
			repository_url, registry_provider, architecture, os, language, framework, version,
			tags, metadata, size_bytes, pull_count, created_by, updated_by, created_at, updated_at,
			deprecated_at, archived_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		model.ID, model.TenantID, model.Name, model.Description, model.Visibility, model.Status,
		model.RepositoryURL, model.RegistryProvider, model.Architecture, model.OS, model.Language, model.Framework, model.Version,
		model.Tags, model.Metadata, model.SizeBytes, model.PullCount, model.CreatedBy, model.UpdatedBy, model.CreatedAt, model.UpdatedAt,
		model.DeprecatedAt, model.ArchivedAt,
	)

	if err != nil {
		r.logger.Error("Failed to save image",
			zap.String("image_id", img.ID().String()),
			zap.String("name", img.Name()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save image: %w", err)
	}

	r.logger.Info("Image saved successfully",
		zap.String("image_id", img.ID().String()),
		zap.String("name", img.Name()),
	)

	return nil
}

// FindByID retrieves an image by ID
func (r *ImageRepository) FindByID(ctx context.Context, id uuid.UUID) (*image.Image, error) {
	query := `
		SELECT id, tenant_id, name, description, visibility, status,
		       repository_url, registry_provider, architecture, os, language, framework, version,
		       tags, metadata, size_bytes, pull_count, created_by, updated_by, created_at, updated_at,
		       deprecated_at, archived_at
		FROM catalog_images
		WHERE id = $1
		  AND ` + releasedQuarantineImagePredicate + `
	`

	var model imageModel
	err := r.db.GetContext(ctx, &model, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to find image by ID",
			zap.String("image_id", id.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find image: %w", err)
	}

	return model.toDomain()
}

// FindByTenantAndName retrieves an image by tenant and name
func (r *ImageRepository) FindByTenantAndName(ctx context.Context, tenantID uuid.UUID, name string) (*image.Image, error) {
	query := `
		SELECT id, tenant_id, name, description, visibility, status,
		       repository_url, registry_provider, architecture, os, language, framework, version,
		       tags, metadata, size_bytes, pull_count, created_by, updated_by, created_at, updated_at,
		       deprecated_at, archived_at
		FROM catalog_images
		WHERE tenant_id = $1 AND name = $2
		  AND ` + releasedQuarantineImagePredicate + `
	`

	var model imageModel
	err := r.db.GetContext(ctx, &model, query, tenantID, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find image by tenant and name: %w", err)
	}

	return model.toDomain()
}

// FindByVisibility retrieves images based on visibility rules
func (r *ImageRepository) FindByVisibility(ctx context.Context, tenantID *uuid.UUID, includePublic bool) ([]*image.Image, error) {
	var query string
	var args []interface{}

	if tenantID != nil && includePublic {
		query = `
			SELECT id, tenant_id, name, description, visibility, status,
			       repository_url, registry_provider, architecture, os, language, framework, version,
			       tags, metadata, size_bytes, pull_count, created_by, updated_by, created_at, updated_at,
			       deprecated_at, archived_at
			FROM catalog_images
			WHERE (visibility = 'public' OR tenant_id = $1)
			      AND status != 'archived'
			      AND ` + releasedQuarantineImagePredicate + `
			ORDER BY created_at DESC
		`
		args = []interface{}{*tenantID}
	} else if tenantID != nil {
		query = `
			SELECT id, tenant_id, name, description, visibility, status,
			       repository_url, registry_provider, architecture, os, language, framework, version,
			       tags, metadata, size_bytes, pull_count, created_by, updated_by, created_at, updated_at,
			       deprecated_at, archived_at
			FROM catalog_images
			WHERE tenant_id = $1 AND status != 'archived'
			      AND ` + releasedQuarantineImagePredicate + `
			ORDER BY created_at DESC
		`
		args = []interface{}{*tenantID}
	} else {
		// No tenant scope provided: only return public images.
		query = `
			SELECT id, tenant_id, name, description, visibility, status,
			       repository_url, registry_provider, architecture, os, language, framework, version,
			       tags, metadata, size_bytes, pull_count, created_by, updated_by, created_at, updated_at,
			       deprecated_at, archived_at
			FROM catalog_images
			WHERE visibility = 'public' AND status != 'archived'
			      AND ` + releasedQuarantineImagePredicate + `
			ORDER BY created_at DESC
		`
	}

	var models []imageModel
	err := r.db.SelectContext(ctx, &models, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find images by visibility: %w", err)
	}

	var images []*image.Image
	for _, model := range models {
		img, err := model.toDomain()
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}

	return images, nil
}

// Search performs full-text search on images
func (r *ImageRepository) Search(ctx context.Context, queryStr string, tenantID *uuid.UUID, filters image.SearchFilters) ([]*image.Image, error) {
	whereConditions := []string{"status != 'archived'", releasedQuarantineImagePredicate}
	args := []interface{}{}
	argCount := 1

	// Add visibility filter
	if tenantID != nil {
		whereConditions = append(whereConditions, "(visibility = 'public' OR tenant_id = $"+fmt.Sprintf("%d", argCount)+")")
		args = append(args, *tenantID)
		argCount++
	} else {
		// No tenant scope provided: only public images.
		whereConditions = append(whereConditions, "visibility = 'public'")
	}

	// Add search query
	if queryStr != "" {
		whereConditions = append(whereConditions, "(name ILIKE $"+fmt.Sprintf("%d", argCount)+" OR description ILIKE $"+fmt.Sprintf("%d", argCount)+")")
		args = append(args, "%"+queryStr+"%")
		argCount++
	}

	// Add other filters
	if filters.Status != nil {
		whereConditions = append(whereConditions, "status = $"+fmt.Sprintf("%d", argCount))
		args = append(args, string(*filters.Status))
		argCount++
	}

	if filters.RegistryProvider != nil {
		whereConditions = append(whereConditions, "registry_provider = $"+fmt.Sprintf("%d", argCount))
		args = append(args, *filters.RegistryProvider)
		argCount++
	}

	if filters.Architecture != nil {
		whereConditions = append(whereConditions, "architecture = $"+fmt.Sprintf("%d", argCount))
		args = append(args, *filters.Architecture)
		argCount++
	}

	if filters.OS != nil {
		whereConditions = append(whereConditions, "os = $"+fmt.Sprintf("%d", argCount))
		args = append(args, *filters.OS)
		argCount++
	}

	if filters.Language != nil {
		whereConditions = append(whereConditions, "language = $"+fmt.Sprintf("%d", argCount))
		args = append(args, *filters.Language)
		argCount++
	}

	if filters.Framework != nil {
		whereConditions = append(whereConditions, "framework = $"+fmt.Sprintf("%d", argCount))
		args = append(args, *filters.Framework)
		argCount++
	}

	if len(filters.Tags) > 0 {
		tagConditions := []string{}
		for _, tag := range filters.Tags {
			tagConditions = append(tagConditions, "$"+fmt.Sprintf("%d", argCount)+" = ANY(tags)")
			args = append(args, tag)
			argCount++
		}
		whereConditions = append(whereConditions, "("+strings.Join(tagConditions, " OR ")+")")
	}

	// Add date filters
	if filters.CreatedAfter != nil {
		whereConditions = append(whereConditions, "created_at >= $"+fmt.Sprintf("%d", argCount))
		args = append(args, *filters.CreatedAfter)
		argCount++
	}

	if filters.CreatedBefore != nil {
		whereConditions = append(whereConditions, "created_at <= $"+fmt.Sprintf("%d", argCount))
		args = append(args, *filters.CreatedBefore)
		argCount++
	}

	// Build ORDER BY
	orderBy := "created_at DESC"
	if filters.SortBy != "" {
		orderBy = filters.SortBy
		if filters.SortOrder == "asc" {
			orderBy += " ASC"
		} else {
			orderBy += " DESC"
		}
	}

	// Build LIMIT/OFFSET
	limit := 50 // default
	if filters.Limit > 0 && filters.Limit <= 100 {
		limit = filters.Limit
	}
	offset := filters.Offset

	sql := fmt.Sprintf(`
		SELECT id, tenant_id, name, description, visibility, status,
		       repository_url, registry_provider, architecture, os, language, framework, version,
		       tags, metadata, size_bytes, pull_count, created_by, updated_by, created_at, updated_at,
		       deprecated_at, archived_at
		FROM catalog_images
		WHERE %s
		ORDER BY %s
		LIMIT %d OFFSET %d
	`, strings.Join(whereConditions, " AND "), orderBy, limit, offset)

	var models []imageModel
	err := r.db.SelectContext(ctx, &models, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search images: %w", err)
	}

	var images []*image.Image
	for _, model := range models {
		img, err := model.toDomain()
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}

	return images, nil
}

// FindPopular returns most popular images
func (r *ImageRepository) FindPopular(ctx context.Context, tenantID *uuid.UUID, limit int) ([]*image.Image, error) {
	var query string
	var args []interface{}

	if tenantID != nil {
		query = `
			SELECT id, tenant_id, name, description, visibility, status,
			       repository_url, registry_provider, architecture, os, language, framework, version,
			       tags, metadata, size_bytes, pull_count, created_by, updated_by, created_at, updated_at,
			       deprecated_at, archived_at
			FROM catalog_images
			WHERE (visibility = 'public' OR tenant_id = $1) AND status = 'published'
			  AND ` + releasedQuarantineImagePredicate + `
			ORDER BY pull_count DESC, created_at DESC
			LIMIT $2
		`
		args = []interface{}{*tenantID, limit}
	} else {
		query = `
			SELECT id, tenant_id, name, description, visibility, status,
			       repository_url, registry_provider, architecture, os, language, framework, version,
			       tags, metadata, size_bytes, pull_count, created_by, updated_by, created_at, updated_at,
			       deprecated_at, archived_at
			FROM catalog_images
			WHERE visibility = 'public' AND status = 'published'
			  AND ` + releasedQuarantineImagePredicate + `
			ORDER BY pull_count DESC, created_at DESC
			LIMIT $1
		`
		args = []interface{}{limit}
	}

	var models []imageModel
	err := r.db.SelectContext(ctx, &models, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find popular images: %w", err)
	}

	var images []*image.Image
	for _, model := range models {
		img, err := model.toDomain()
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}

	return images, nil
}

// FindRecent returns recently added images
func (r *ImageRepository) FindRecent(ctx context.Context, tenantID *uuid.UUID, limit int) ([]*image.Image, error) {
	var query string
	var args []interface{}

	if tenantID != nil {
		query = `
			SELECT id, tenant_id, name, description, visibility, status,
			       repository_url, registry_provider, architecture, os, language, framework, version,
			       tags, metadata, size_bytes, pull_count, created_by, updated_by, created_at, updated_at,
			       deprecated_at, archived_at
			FROM catalog_images
			WHERE (visibility = 'public' OR tenant_id = $1) AND status = 'published'
			  AND ` + releasedQuarantineImagePredicate + `
			ORDER BY created_at DESC
			LIMIT $2
		`
		args = []interface{}{*tenantID, limit}
	} else {
		query = `
			SELECT id, tenant_id, name, description, visibility, status,
			       repository_url, registry_provider, architecture, os, language, framework, version,
			       tags, metadata, size_bytes, pull_count, created_by, updated_by, created_at, updated_at,
			       deprecated_at, archived_at
			FROM catalog_images
			WHERE visibility = 'public' AND status = 'published'
			  AND ` + releasedQuarantineImagePredicate + `
			ORDER BY created_at DESC
			LIMIT $1
		`
		args = []interface{}{limit}
	}

	var models []imageModel
	err := r.db.SelectContext(ctx, &models, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find recent images: %w", err)
	}

	var images []*image.Image
	for _, model := range models {
		img, err := model.toDomain()
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}

	return images, nil
}

// Update updates an existing image
func (r *ImageRepository) Update(ctx context.Context, img *image.Image) error {
	model := fromDomain(img)

	query := `
		UPDATE catalog_images SET
			description = $1, visibility = $2, status = $3,
			repository_url = $4, registry_provider = $5, architecture = $6, os = $7,
			language = $8, framework = $9, version = $10, tags = $11, metadata = $12,
			size_bytes = $13, pull_count = $14, updated_by = $15, updated_at = $16,
			deprecated_at = $17, archived_at = $18
		WHERE id = $19
	`

	_, err := r.db.ExecContext(ctx, query,
		model.Description, model.Visibility, model.Status,
		model.RepositoryURL, model.RegistryProvider, model.Architecture, model.OS,
		model.Language, model.Framework, model.Version, model.Tags, model.Metadata,
		model.SizeBytes, model.PullCount, model.UpdatedBy, model.UpdatedAt,
		model.DeprecatedAt, model.ArchivedAt, model.ID,
	)

	if err != nil {
		r.logger.Error("Failed to update image",
			zap.String("image_id", img.ID().String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update image: %w", err)
	}

	return nil
}

// Delete removes an image
func (r *ImageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM catalog_images WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete image",
			zap.String("image_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete image: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return image.ErrImageNotFound
	}

	r.logger.Info("Image deleted successfully",
		zap.String("image_id", id.String()),
	)

	return nil
}

// ExistsByTenantAndName checks if an image exists
func (r *ImageRepository) ExistsByTenantAndName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM catalog_images WHERE tenant_id = $1 AND name = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, tenantID, name)
	if err != nil {
		return false, fmt.Errorf("failed to check image existence: %w", err)
	}

	return exists, nil
}

// CountByTenant returns the number of images for a tenant
func (r *ImageRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM catalog_images WHERE tenant_id = $1 AND status != 'archived' AND ` + releasedQuarantineImagePredicate

	var count int
	err := r.db.GetContext(ctx, &count, query, tenantID)
	if err != nil {
		return 0, fmt.Errorf("failed to count images by tenant: %w", err)
	}

	return count, nil
}

// CountByVisibility returns counts by visibility level
func (r *ImageRepository) CountByVisibility(ctx context.Context, tenantID *uuid.UUID) (map[image.ImageVisibility]int, error) {
	query := `
		SELECT visibility, COUNT(*) as count
		FROM catalog_images
		WHERE status != 'archived'
		  AND ` + releasedQuarantineImagePredicate + `
	`

	var args []interface{}
	if tenantID != nil {
		query += ` AND (visibility = 'public' OR tenant_id = $1)`
		args = append(args, *tenantID)
	} else {
		query += ` AND visibility = 'public'`
	}

	query += ` GROUP BY visibility`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to count images by visibility: %w", err)
	}
	defer rows.Close()

	counts := make(map[image.ImageVisibility]int)
	for rows.Next() {
		var visibility string
		var count int
		if err := rows.Scan(&visibility, &count); err != nil {
			return nil, err
		}
		counts[image.ImageVisibility(visibility)] = count
	}

	return counts, nil
}

// IncrementPullCount increments the pull count for an image
func (r *ImageRepository) IncrementPullCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE catalog_images SET pull_count = pull_count + 1, updated_at = NOW() WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to increment pull count",
			zap.String("image_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to increment pull count: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return image.ErrImageNotFound
	}

	return nil
}
