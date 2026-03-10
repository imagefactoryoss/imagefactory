package image

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Domain errors
var (
	ErrImageNotFound         = errors.New("image not found")
	ErrImageExists           = errors.New("image already exists")
	ErrInvalidImageID        = errors.New("invalid image ID")
	ErrInvalidTenantID       = errors.New("invalid tenant ID")
	ErrInvalidImageName      = errors.New("invalid image name")
	ErrInvalidVisibility     = errors.New("invalid visibility level")
	ErrInvalidStatus         = errors.New("invalid status")
	ErrPermissionDenied      = errors.New("permission denied")
	ErrVersionNotFound       = errors.New("image version not found")
	ErrVersionExists         = errors.New("image version already exists")
	ErrInvalidVersion        = errors.New("invalid version format")
	ErrInvalidImageVersionID = errors.New("invalid image version ID")
	ErrImageTagNotFound      = errors.New("image tag not found")
	ErrImageVersionNotFound  = errors.New("image version not found")
)

// ImageVisibility represents the visibility level of an image
type ImageVisibility string

const (
	VisibilityPublic  ImageVisibility = "public"  // Visible to all authenticated users
	VisibilityTenant  ImageVisibility = "tenant"  // Visible only to tenant members
	VisibilityPrivate ImageVisibility = "private" // Visible only to specific users/groups (future)
)

// ImageStatus represents the lifecycle status of an image
type ImageStatus string

const (
	StatusDraft      ImageStatus = "draft"      // Being created/edited
	StatusPublished  ImageStatus = "published"  // Available for use
	StatusDeprecated ImageStatus = "deprecated" // Marked for removal but still available
	StatusArchived   ImageStatus = "archived"   // No longer available for new use
)

// ValidateVisibility validates image visibility
func ValidateVisibility(visibility ImageVisibility) error {
	switch visibility {
	case VisibilityPublic, VisibilityTenant, VisibilityPrivate:
		return nil
	default:
		return ErrInvalidVisibility
	}
}

// ValidateStatus validates image status
func ValidateStatus(status ImageStatus) error {
	switch status {
	case StatusDraft, StatusPublished, StatusDeprecated, StatusArchived:
		return nil
	default:
		return ErrInvalidStatus
	}
}

// Image represents a container image in the catalog
type Image struct {
	id               uuid.UUID
	tenantID         uuid.UUID
	name             string
	description      string
	visibility       ImageVisibility
	status           ImageStatus
	repositoryURL    *string
	registryProvider *string
	architecture     *string
	os               *string
	language         *string
	framework        *string
	version          *string
	tags             []string
	metadata         map[string]interface{}
	sizeBytes        *int64
	pullCount        int64
	createdBy        uuid.UUID
	updatedBy        uuid.UUID
	createdAt        time.Time
	updatedAt        time.Time
	deprecatedAt     *time.Time
	archivedAt       *time.Time
}

// ImageVersion represents a specific version of an image
type ImageVersion struct {
	id           uuid.UUID
	imageID      uuid.UUID
	version      string
	description  *string
	digest       *string
	sizeBytes    *int64
	manifest     map[string]interface{}
	config       map[string]interface{}
	layers       []map[string]interface{}
	tags         []string
	metadata     map[string]interface{}
	createdBy    uuid.UUID
	createdAt    time.Time
	publishedAt  time.Time
	deprecatedAt *time.Time
}

// ImageTag represents a tag associated with an image for search optimization
type ImageTag struct {
	ID        uuid.UUID `json:"id"`
	ImageID   uuid.UUID `json:"image_id"`
	Tag       string    `json:"tag"`
	Category  string    `json:"category"` // user, auto, system
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// NewImage creates a new image aggregate
func NewImage(tenantID uuid.UUID, name, description string, visibility ImageVisibility, createdBy uuid.UUID) (*Image, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}

	if name == "" {
		return nil, ErrInvalidImageName
	}

	if err := ValidateVisibility(visibility); err != nil {
		return nil, err
	}

	now := time.Now()
	image := &Image{
		id:          uuid.New(),
		tenantID:    tenantID,
		name:        name,
		description: description,
		visibility:  visibility,
		status:      StatusDraft,
		tags:        []string{},
		metadata:    make(map[string]interface{}),
		pullCount:   0,
		createdBy:   createdBy,
		updatedBy:   createdBy,
		createdAt:   now,
		updatedAt:   now,
	}

	return image, nil
}

// ReconstructImage creates an image aggregate from database data (for loading existing images)
func ReconstructImage(
	id, tenantID uuid.UUID,
	name, description string,
	visibility ImageVisibility,
	status ImageStatus,
	repositoryURL, registryProvider, architecture, os, language, framework, version *string,
	tags []string,
	metadata map[string]interface{},
	sizeBytes *int64,
	pullCount int64,
	createdBy, updatedBy uuid.UUID,
	createdAt, updatedAt time.Time,
	deprecatedAt, archivedAt *time.Time,
) (*Image, error) {
	if name == "" {
		return nil, ErrInvalidImageName
	}

	if err := ValidateVisibility(visibility); err != nil {
		return nil, err
	}

	if err := ValidateStatus(status); err != nil {
		return nil, err
	}

	image := &Image{
		id:               id,
		tenantID:         tenantID,
		name:             name,
		description:      description,
		visibility:       visibility,
		status:           status,
		repositoryURL:    repositoryURL,
		registryProvider: registryProvider,
		architecture:     architecture,
		os:               os,
		language:         language,
		framework:        framework,
		version:          version,
		tags:             tags,
		metadata:         metadata,
		sizeBytes:        sizeBytes,
		pullCount:        pullCount,
		createdBy:        createdBy,
		updatedBy:        updatedBy,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
		deprecatedAt:     deprecatedAt,
		archivedAt:       archivedAt,
	}

	return image, nil
}

// NewImageVersion creates a new image version
func NewImageVersion(imageID uuid.UUID, version string, digest *string, sizeBytes *int64, createdBy uuid.UUID) (*ImageVersion, error) {
	if version == "" {
		return nil, ErrInvalidVersion
	}

	now := time.Now()
	imageVersion := &ImageVersion{
		id:          uuid.New(),
		imageID:     imageID,
		version:     version,
		digest:      digest,
		sizeBytes:   sizeBytes,
		tags:        []string{},
		metadata:    make(map[string]interface{}),
		createdBy:   createdBy,
		createdAt:   now,
		publishedAt: now,
	}

	return imageVersion, nil
}

// ReconstructImageVersion rebuilds a persisted image version aggregate from storage.
func ReconstructImageVersion(
	id uuid.UUID,
	imageID uuid.UUID,
	version string,
	description *string,
	digest *string,
	sizeBytes *int64,
	manifest map[string]interface{},
	config map[string]interface{},
	layers []map[string]interface{},
	metadata map[string]interface{},
	createdBy uuid.UUID,
	createdAt time.Time,
	publishedAt time.Time,
	deprecatedAt *time.Time,
) (*ImageVersion, error) {
	if version == "" {
		return nil, ErrInvalidVersion
	}
	if id == uuid.Nil {
		return nil, ErrInvalidImageVersionID
	}
	if imageID == uuid.Nil {
		return nil, ErrInvalidImageID
	}
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	if publishedAt.IsZero() {
		publishedAt = createdAt
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	return &ImageVersion{
		id:           id,
		imageID:      imageID,
		version:      version,
		description:  description,
		digest:       digest,
		sizeBytes:    sizeBytes,
		manifest:     manifest,
		config:       config,
		layers:       layers,
		tags:         []string{},
		metadata:     metadata,
		createdBy:    createdBy,
		createdAt:    createdAt,
		publishedAt:  publishedAt,
		deprecatedAt: deprecatedAt,
	}, nil
}

// Getters for Image
func (i *Image) ID() uuid.UUID                    { return i.id }
func (i *Image) TenantID() uuid.UUID              { return i.tenantID }
func (i *Image) Name() string                     { return i.name }
func (i *Image) Description() string              { return i.description }
func (i *Image) Visibility() ImageVisibility      { return i.visibility }
func (i *Image) Status() ImageStatus              { return i.status }
func (i *Image) RepositoryURL() *string           { return i.repositoryURL }
func (i *Image) RegistryProvider() *string        { return i.registryProvider }
func (i *Image) Architecture() *string            { return i.architecture }
func (i *Image) OS() *string                      { return i.os }
func (i *Image) Language() *string                { return i.language }
func (i *Image) Framework() *string               { return i.framework }
func (i *Image) Version() *string                 { return i.version }
func (i *Image) Tags() []string                   { return i.tags }
func (i *Image) Metadata() map[string]interface{} { return i.metadata }
func (i *Image) SizeBytes() *int64                { return i.sizeBytes }
func (i *Image) PullCount() int64                 { return i.pullCount }
func (i *Image) CreatedBy() uuid.UUID             { return i.createdBy }
func (i *Image) UpdatedBy() uuid.UUID             { return i.updatedBy }
func (i *Image) CreatedAt() time.Time             { return i.createdAt }
func (i *Image) UpdatedAt() time.Time             { return i.updatedAt }
func (i *Image) DeprecatedAt() *time.Time         { return i.deprecatedAt }
func (i *Image) ArchivedAt() *time.Time           { return i.archivedAt }

// Getters for ImageVersion
func (iv *ImageVersion) ID() uuid.UUID                    { return iv.id }
func (iv *ImageVersion) ImageID() uuid.UUID               { return iv.imageID }
func (iv *ImageVersion) Version() string                  { return iv.version }
func (iv *ImageVersion) Description() *string             { return iv.description }
func (iv *ImageVersion) Digest() *string                  { return iv.digest }
func (iv *ImageVersion) SizeBytes() *int64                { return iv.sizeBytes }
func (iv *ImageVersion) Manifest() map[string]interface{} { return iv.manifest }
func (iv *ImageVersion) Config() map[string]interface{}   { return iv.config }
func (iv *ImageVersion) Layers() []map[string]interface{} { return iv.layers }
func (iv *ImageVersion) Tags() []string                   { return iv.tags }
func (iv *ImageVersion) Metadata() map[string]interface{} { return iv.metadata }
func (iv *ImageVersion) CreatedBy() uuid.UUID             { return iv.createdBy }
func (iv *ImageVersion) CreatedAt() time.Time             { return iv.createdAt }
func (iv *ImageVersion) PublishedAt() time.Time           { return iv.publishedAt }
func (iv *ImageVersion) DeprecatedAt() *time.Time         { return iv.deprecatedAt }

// Setters for ImageVersion
func (iv *ImageVersion) SetDescription(desc *string)                 { iv.description = desc }
func (iv *ImageVersion) SetDigest(digest *string)                    { iv.digest = digest }
func (iv *ImageVersion) SetSizeBytes(size *int64)                    { iv.sizeBytes = size }
func (iv *ImageVersion) SetManifest(manifest map[string]interface{}) { iv.manifest = manifest }
func (iv *ImageVersion) SetConfig(config map[string]interface{})     { iv.config = config }
func (iv *ImageVersion) SetLayers(layers []map[string]interface{})   { iv.layers = layers }
func (iv *ImageVersion) SetTags(tags []string)                       { iv.tags = tags }
func (iv *ImageVersion) SetMetadata(metadata map[string]interface{}) { iv.metadata = metadata }
func (iv *ImageVersion) SetPublishedAt(publishedAt time.Time)        { iv.publishedAt = publishedAt }
func (iv *ImageVersion) SetDeprecatedAt(deprecatedAt *time.Time)     { iv.deprecatedAt = deprecatedAt }

// Getters for ImageTag

// Update methods for Image
func (i *Image) UpdateDescription(description string) {
	i.description = description
	i.updatedAt = time.Now()
}

func (i *Image) UpdateVisibility(visibility ImageVisibility) error {
	if err := ValidateVisibility(visibility); err != nil {
		return err
	}
	i.visibility = visibility
	i.updatedAt = time.Now()
	return nil
}

func (i *Image) UpdateStatus(status ImageStatus) error {
	if err := ValidateStatus(status); err != nil {
		return err
	}
	i.status = status
	i.updatedAt = time.Now()

	// Set timestamps based on status
	now := time.Now()
	switch status {
	case StatusDeprecated:
		i.deprecatedAt = &now
	case StatusArchived:
		i.archivedAt = &now
	}

	return nil
}

func (i *Image) UpdateMetadata(metadata map[string]interface{}) {
	i.metadata = metadata
	i.updatedAt = time.Now()
}

func (i *Image) AddTags(tags []string) {
	// Avoid duplicates
	tagSet := make(map[string]bool)
	for _, tag := range i.tags {
		tagSet[tag] = true
	}
	for _, tag := range tags {
		if !tagSet[tag] {
			i.tags = append(i.tags, tag)
			tagSet[tag] = true
		}
	}
	i.updatedAt = time.Now()
}

func (i *Image) RemoveTags(tags []string) {
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	var newTags []string
	for _, tag := range i.tags {
		if !tagSet[tag] {
			newTags = append(newTags, tag)
		}
	}
	i.tags = newTags
	i.updatedAt = time.Now()
}

func (i *Image) IncrementPullCount() {
	i.pullCount++
	i.updatedAt = time.Now()
}

func (i *Image) SetUpdatedBy(userID uuid.UUID) {
	i.updatedBy = userID
	i.updatedAt = time.Now()
}
