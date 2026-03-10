package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/image"
)

// ImageTagRepository implements the image.TagRepository interface
type ImageTagRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewImageTagRepository creates a new image tag repository
func NewImageTagRepository(db *sqlx.DB, logger *zap.Logger) *ImageTagRepository {
	return &ImageTagRepository{
		db:     db,
		logger: logger,
	}
}

// imageTagModel represents the database model for image tag
type imageTagModel struct {
	ID        uuid.UUID `db:"id"`
	ImageID   uuid.UUID `db:"image_id"`
	Tag       string    `db:"tag"`
	CreatedBy uuid.UUID `db:"created_by"`
	CreatedAt time.Time `db:"created_at"`
}

// toDomain converts database model to domain model
func (m *imageTagModel) toDomain() *image.ImageTag {
	return &image.ImageTag{
		ID:        m.ID,
		ImageID:   m.ImageID,
		Tag:       m.Tag,
		CreatedBy: m.CreatedBy,
		CreatedAt: m.CreatedAt,
	}
}

// fromDomain converts domain model to database model
func fromTagDomain(tag *image.ImageTag) *imageTagModel {
	return &imageTagModel{
		ID:        tag.ID,
		ImageID:   tag.ImageID,
		Tag:       tag.Tag,
		CreatedBy: tag.CreatedBy,
		CreatedAt: tag.CreatedAt,
	}
}

// Save persists an image tag
func (r *ImageTagRepository) Save(ctx context.Context, tag *image.ImageTag) error {
	model := fromTagDomain(tag)

	query := `
		INSERT INTO catalog_image_tags (id, catalog_image_id, tag, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		model.ID, model.ImageID, model.Tag, model.CreatedBy, model.CreatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to save image tag",
			zap.String("tag_id", tag.ID.String()),
			zap.String("image_id", tag.ImageID.String()),
			zap.String("tag", tag.Tag),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save image tag: %w", err)
	}

	r.logger.Info("Image tag saved successfully",
		zap.String("tag_id", tag.ID.String()),
		zap.String("tag", tag.Tag),
	)

	return nil
}

// FindByImageID retrieves all tags for an image
func (r *ImageTagRepository) FindByImageID(ctx context.Context, imageID uuid.UUID) ([]*image.ImageTag, error) {
	query := `
		SELECT id, catalog_image_id AS image_id, tag, created_by, created_at
		FROM catalog_image_tags
		WHERE catalog_image_id = $1
		ORDER BY created_at ASC
	`

	var models []imageTagModel
	err := r.db.SelectContext(ctx, &models, query, imageID)
	if err != nil {
		r.logger.Error("Failed to find image tags by image ID",
			zap.String("image_id", imageID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find image tags: %w", err)
	}

	var tags []*image.ImageTag
	for _, model := range models {
		tags = append(tags, model.toDomain())
	}

	return tags, nil
}

// FindByTag retrieves all images with a specific tag
func (r *ImageTagRepository) FindByTag(ctx context.Context, tag string) ([]uuid.UUID, error) {
	query := `
		SELECT DISTINCT catalog_image_id
		FROM catalog_image_tags
		WHERE tag = $1
	`

	var imageIDs []uuid.UUID
	err := r.db.SelectContext(ctx, &imageIDs, query, tag)
	if err != nil {
		r.logger.Error("Failed to find images by tag",
			zap.String("tag", tag),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find images by tag: %w", err)
	}

	return imageIDs, nil
}

// Delete removes an image tag
func (r *ImageTagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM catalog_image_tags WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete image tag",
			zap.String("tag_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete image tag: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return image.ErrImageTagNotFound
	}

	r.logger.Info("Image tag deleted successfully",
		zap.String("tag_id", id.String()),
	)

	return nil
}

// DeleteByImageID removes all tags for an image
func (r *ImageTagRepository) DeleteByImageID(ctx context.Context, imageID uuid.UUID) error {
	query := `DELETE FROM catalog_image_tags WHERE catalog_image_id = $1`

	_, err := r.db.ExecContext(ctx, query, imageID)
	if err != nil {
		r.logger.Error("Failed to delete image tags by image ID",
			zap.String("image_id", imageID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete image tags: %w", err)
	}

	r.logger.Info("Image tags deleted successfully",
		zap.String("image_id", imageID.String()),
	)

	return nil
}

// ExistsByImageIDAndTag checks if a tag exists for an image
func (r *ImageTagRepository) ExistsByImageIDAndTag(ctx context.Context, imageID uuid.UUID, tag string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM catalog_image_tags WHERE catalog_image_id = $1 AND tag = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, imageID, tag)
	if err != nil {
		return false, fmt.Errorf("failed to check image tag existence: %w", err)
	}

	return exists, nil
}

// GetPopularTags returns the most used tags
func (r *ImageTagRepository) GetPopularTags(ctx context.Context, limit int) ([]string, error) {
	query := `
		SELECT tag, COUNT(*) as count
		FROM catalog_image_tags
		GROUP BY tag
		ORDER BY count DESC
		LIMIT $1
	`

	type tagCount struct {
		Tag   string `db:"tag"`
		Count int    `db:"count"`
	}

	var results []tagCount
	err := r.db.SelectContext(ctx, &results, query, limit)
	if err != nil {
		r.logger.Error("Failed to get popular tags",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get popular tags: %w", err)
	}

	var tags []string
	for _, result := range results {
		tags = append(tags, result.Tag)
	}

	return tags, nil
}

// CountByImageID returns the number of tags for an image
func (r *ImageTagRepository) CountByImageID(ctx context.Context, imageID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM catalog_image_tags WHERE catalog_image_id = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, imageID)
	if err != nil {
		return 0, fmt.Errorf("failed to count image tags: %w", err)
	}

	return count, nil
}
