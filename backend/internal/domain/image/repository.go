package image

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for image persistence
type Repository interface {
	// Save persists an image
	Save(ctx context.Context, image *Image) error

	// FindByID retrieves an image by ID
	FindByID(ctx context.Context, id uuid.UUID) (*Image, error)

	// FindByTenantAndName retrieves an image by tenant and name
	FindByTenantAndName(ctx context.Context, tenantID uuid.UUID, name string) (*Image, error)

	// FindByVisibility retrieves images based on visibility rules
	// For public images: returns all public images
	// For tenant images: returns tenant's images + public images
	FindByVisibility(ctx context.Context, tenantID *uuid.UUID, includePublic bool) ([]*Image, error)

	// Search performs full-text search on images with visibility filtering
	Search(ctx context.Context, query string, tenantID *uuid.UUID, filters SearchFilters) ([]*Image, error)

	// FindPopular returns most popular images by pull count
	FindPopular(ctx context.Context, tenantID *uuid.UUID, limit int) ([]*Image, error)

	// FindRecent returns recently added/updated images
	FindRecent(ctx context.Context, tenantID *uuid.UUID, limit int) ([]*Image, error)

	// Update updates an existing image
	Update(ctx context.Context, image *Image) error

	// Delete removes an image
	Delete(ctx context.Context, id uuid.UUID) error

	// ExistsByTenantAndName checks if an image exists by tenant and name
	ExistsByTenantAndName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error)

	// CountByTenant returns the number of images for a tenant
	CountByTenant(ctx context.Context, tenantID uuid.UUID) (int, error)

	// CountByVisibility returns counts by visibility level
	CountByVisibility(ctx context.Context, tenantID *uuid.UUID) (map[ImageVisibility]int, error)

	// IncrementPullCount increments the pull count for an image
	IncrementPullCount(ctx context.Context, id uuid.UUID) error
}

// ImageVersionRepository defines the interface for image version persistence
type ImageVersionRepository interface {
	// Save persists an image version
	Save(ctx context.Context, version *ImageVersion) error

	// FindByID retrieves an image version by ID
	FindByID(ctx context.Context, id uuid.UUID) (*ImageVersion, error)

	// FindByImageID retrieves all versions for an image
	FindByImageID(ctx context.Context, imageID uuid.UUID) ([]*ImageVersion, error)

	// FindByImageAndVersion retrieves a specific version of an image
	FindByImageAndVersion(ctx context.Context, imageID uuid.UUID, version string) (*ImageVersion, error)

	// FindLatestByImageID retrieves the latest version of an image
	FindLatestByImageID(ctx context.Context, imageID uuid.UUID) (*ImageVersion, error)

	// Update updates an existing image version
	Update(ctx context.Context, version *ImageVersion) error

	// Delete removes an image version
	Delete(ctx context.Context, id uuid.UUID) error
}

// ImageTagRepository defines the interface for image tag persistence
type ImageTagRepository interface {
	// Save persists an image tag
	Save(ctx context.Context, tag *ImageTag) error

	// FindByImageID retrieves all tags for an image
	FindByImageID(ctx context.Context, imageID uuid.UUID) ([]*ImageTag, error)

	// FindByTag retrieves image IDs that have a specific tag
	FindByTag(ctx context.Context, tag string) ([]uuid.UUID, error)

	// DeleteByImageID removes all tags for an image
	DeleteByImageID(ctx context.Context, imageID uuid.UUID) error

	// Delete removes a specific tag
	Delete(ctx context.Context, id uuid.UUID) error
}

// SearchFilters defines filters for image search
type SearchFilters struct {
	Visibility     *ImageVisibility `json:"visibility,omitempty"`
	Status         *ImageStatus     `json:"status,omitempty"`
	RegistryProvider *string        `json:"registry_provider,omitempty"`
	Architecture   *string          `json:"architecture,omitempty"`
	OS             *string           `json:"os,omitempty"`
	Language       *string           `json:"language,omitempty"`
	Framework      *string           `json:"framework,omitempty"`
	Tags           []string         `json:"tags,omitempty"`
	SizeMin        *int64           `json:"size_min,omitempty"`
	SizeMax        *int64           `json:"size_max,omitempty"`
	CreatedAfter   *time.Time       `json:"created_after,omitempty"`
	CreatedBefore  *time.Time       `json:"created_before,omitempty"`
	SortBy         string           `json:"sort_by,omitempty"` // name, created_at, updated_at, pull_count, size
	SortOrder      string           `json:"sort_order,omitempty"` // asc, desc
	Limit          int              `json:"limit,omitempty"`
	Offset         int              `json:"offset,omitempty"`
}
