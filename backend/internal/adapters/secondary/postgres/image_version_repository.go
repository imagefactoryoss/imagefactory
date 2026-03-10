package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/image"
)

// Helper functions for dereferencing pointers
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

// ImageVersionRepository implements the image.VersionRepository interface
type ImageVersionRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewImageVersionRepository creates a new image version repository
func NewImageVersionRepository(db *sqlx.DB, logger *zap.Logger) *ImageVersionRepository {
	return &ImageVersionRepository{
		db:     db,
		logger: logger,
	}
}

// imageVersionModel represents the database model for image version
type imageVersionModel struct {
	ID          uuid.UUID       `db:"id"`
	ImageID     uuid.UUID       `db:"image_id"`
	Version     string          `db:"version"`
	Description sql.NullString  `db:"description"`
	Digest      string          `db:"digest"`
	SizeBytes   int64           `db:"size_bytes"`
	Manifest    json.RawMessage `db:"manifest"`
	Config      json.RawMessage `db:"config"`
	Layers      json.RawMessage `db:"layers"`
	CreatedBy   uuid.UUID       `db:"created_by"`
	CreatedAt   time.Time       `db:"created_at"`
}

// toDomain converts database model to domain model
func (m *imageVersionModel) toDomain() (*image.ImageVersion, error) {
	var manifest map[string]interface{}
	if len(m.Manifest) > 0 {
		if err := json.Unmarshal(m.Manifest, &manifest); err != nil {
			return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
		}
	}

	var config map[string]interface{}
	if len(m.Config) > 0 {
		if err := json.Unmarshal(m.Config, &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	var layers []map[string]interface{}
	if len(m.Layers) > 0 {
		if err := json.Unmarshal(m.Layers, &layers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal layers: %w", err)
		}
	}

	var digestPtr *string
	if m.Digest != "" {
		digest := m.Digest
		digestPtr = &digest
	}
	var sizePtr *int64
	if m.SizeBytes > 0 {
		size := m.SizeBytes
		sizePtr = &size
	}
	version, err := image.ReconstructImageVersion(
		m.ID,
		m.ImageID,
		m.Version,
		nil,
		digestPtr,
		sizePtr,
		manifest,
		config,
		layers,
		map[string]interface{}{},
		m.CreatedBy,
		m.CreatedAt,
		m.CreatedAt,
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Set additional fields
	if m.Description.Valid {
		desc := m.Description.String
		version.SetDescription(&desc)
	}

	return version, nil
}

// fromDomain converts domain model to database model
func fromVersionDomain(version *image.ImageVersion) *imageVersionModel {
	manifest, _ := json.Marshal(version.Manifest())
	config, _ := json.Marshal(version.Config())
	layers, _ := json.Marshal(version.Layers())

	return &imageVersionModel{
		ID:          version.ID(),
		ImageID:     version.ImageID(),
		Version:     version.Version(),
		Description: sql.NullString{String: derefString(version.Description()), Valid: version.Description() != nil},
		Digest:      derefString(version.Digest()),
		SizeBytes:   derefInt64(version.SizeBytes()),
		Manifest:    manifest,
		Config:      config,
		Layers:      layers,
		CreatedBy:   version.CreatedBy(),
		CreatedAt:   version.CreatedAt(),
	}
}

// Save persists an image version
func (r *ImageVersionRepository) Save(ctx context.Context, version *image.ImageVersion) error {
	model := fromVersionDomain(version)

	query := `
		INSERT INTO catalog_image_versions (
			id, catalog_image_id, version, description, digest, size_bytes,
			manifest, config, layers, created_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		model.ID, model.ImageID, model.Version, model.Description, model.Digest, model.SizeBytes,
		model.Manifest, model.Config, model.Layers, model.CreatedBy, model.CreatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to save image version",
			zap.String("version_id", version.ID().String()),
			zap.String("image_id", version.ImageID().String()),
			zap.String("version", version.Version()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save image version: %w", err)
	}

	r.logger.Info("Image version saved successfully",
		zap.String("version_id", version.ID().String()),
		zap.String("version", version.Version()),
	)

	return nil
}

// FindByID retrieves an image version by ID
func (r *ImageVersionRepository) FindByID(ctx context.Context, id uuid.UUID) (*image.ImageVersion, error) {
	query := `
		SELECT id, catalog_image_id AS image_id, version, description, digest, size_bytes,
		       manifest, config, layers, created_by, created_at
		FROM catalog_image_versions
		WHERE id = $1
	`

	var model imageVersionModel
	err := r.db.GetContext(ctx, &model, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to find image version by ID",
			zap.String("version_id", id.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find image version: %w", err)
	}

	return model.toDomain()
}

// FindByImageID retrieves all versions for an image
func (r *ImageVersionRepository) FindByImageID(ctx context.Context, imageID uuid.UUID) ([]*image.ImageVersion, error) {
	query := `
		SELECT id, catalog_image_id AS image_id, version, description, digest, size_bytes,
		       manifest, config, layers, created_by, created_at
		FROM catalog_image_versions
		WHERE catalog_image_id = $1
		ORDER BY created_at DESC
	`

	var models []imageVersionModel
	err := r.db.SelectContext(ctx, &models, query, imageID)
	if err != nil {
		r.logger.Error("Failed to find image versions by image ID",
			zap.String("image_id", imageID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find image versions: %w", err)
	}

	var versions []*image.ImageVersion
	for _, model := range models {
		version, err := model.toDomain()
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}

	return versions, nil
}

// FindByImageIDAndVersion retrieves a specific version for an image
func (r *ImageVersionRepository) FindByImageIDAndVersion(ctx context.Context, imageID uuid.UUID, version string) (*image.ImageVersion, error) {
	query := `
		SELECT id, catalog_image_id AS image_id, version, description, digest, size_bytes,
		       manifest, config, layers, created_by, created_at
		FROM catalog_image_versions
		WHERE catalog_image_id = $1 AND version = $2
	`

	var model imageVersionModel
	err := r.db.GetContext(ctx, &model, query, imageID, version)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find image version: %w", err)
	}

	return model.toDomain()
}

// FindByImageAndVersion retrieves a specific version for an image.
// Kept to satisfy the domain repository interface.
func (r *ImageVersionRepository) FindByImageAndVersion(ctx context.Context, imageID uuid.UUID, version string) (*image.ImageVersion, error) {
	return r.FindByImageIDAndVersion(ctx, imageID, version)
}

// FindLatestByImageID retrieves the latest version for an image
func (r *ImageVersionRepository) FindLatestByImageID(ctx context.Context, imageID uuid.UUID) (*image.ImageVersion, error) {
	query := `
		SELECT id, catalog_image_id AS image_id, version, description, digest, size_bytes,
		       manifest, config, layers, created_by, created_at
		FROM catalog_image_versions
		WHERE catalog_image_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var model imageVersionModel
	err := r.db.GetContext(ctx, &model, query, imageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to find latest image version",
			zap.String("image_id", imageID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find latest image version: %w", err)
	}

	return model.toDomain()
}

// Delete removes an image version
func (r *ImageVersionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM catalog_image_versions WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete image version",
			zap.String("version_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete image version: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return image.ErrImageVersionNotFound
	}

	r.logger.Info("Image version deleted successfully",
		zap.String("version_id", id.String()),
	)

	return nil
}

// ExistsByImageIDAndVersion checks if a version exists for an image
func (r *ImageVersionRepository) ExistsByImageIDAndVersion(ctx context.Context, imageID uuid.UUID, version string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM catalog_image_versions WHERE catalog_image_id = $1 AND version = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, imageID, version)
	if err != nil {
		return false, fmt.Errorf("failed to check image version existence: %w", err)
	}

	return exists, nil
}

// CountByImageID returns the number of versions for an image
func (r *ImageVersionRepository) CountByImageID(ctx context.Context, imageID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM catalog_image_versions WHERE catalog_image_id = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, imageID)
	if err != nil {
		return 0, fmt.Errorf("failed to count image versions: %w", err)
	}

	return count, nil
}

// Update updates an image version by id, with natural-key fallback.
func (r *ImageVersionRepository) Update(ctx context.Context, version *image.ImageVersion) error {
	model := fromVersionDomain(version)

	query := `
		UPDATE catalog_image_versions
		SET version = $1,
		    description = $2,
		    digest = $3,
		    size_bytes = $4,
		    manifest = $5,
		    config = $6,
		    layers = $7,
		    created_by = $8
		WHERE id = $9
		   OR (catalog_image_id = $10 AND version = $11)
	`

	result, err := r.db.ExecContext(ctx, query,
		model.Version, model.Description, model.Digest, model.SizeBytes,
		model.Manifest, model.Config, model.Layers, model.CreatedBy, model.ID, model.ImageID, model.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to update image version: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return image.ErrImageVersionNotFound
	}

	return nil
}
