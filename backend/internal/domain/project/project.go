package project

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Domain errors
var (
	ErrProjectNotFound      = errors.New("project not found")
	ErrInvalidProjectID     = errors.New("invalid project ID")
	ErrDuplicateProjectName = errors.New("project name already exists in tenant")
	ErrDuplicateProjectSlug = errors.New("project slug already exists in tenant")
	ErrInvalidProjectData   = errors.New("invalid project data")
)

// ProjectStatus represents the status of a project
type ProjectStatus string

const (
	ProjectStatusActive    ProjectStatus = "active"
	ProjectStatusArchived  ProjectStatus = "archived"
	ProjectStatusSuspended ProjectStatus = "suspended"
)

// ProjectVisibility represents the visibility level of a project
type ProjectVisibility string

const (
	ProjectVisibilityPrivate  ProjectVisibility = "private"
	ProjectVisibilityInternal ProjectVisibility = "internal"
	ProjectVisibilityPublic   ProjectVisibility = "public"
)

// Project represents the project aggregate root
type Project struct {
	id          uuid.UUID
	tenantID    uuid.UUID
	name        string
	slug        string
	description string
	status      ProjectStatus
	visibility  ProjectVisibility
	gitRepo     string
	gitBranch   string
	gitProvider string
	repoAuthID  *uuid.UUID
	createdBy   *uuid.UUID
	isDraft     bool
	buildCount  int
	createdAt   time.Time
	updatedAt   time.Time
	deletedAt   *time.Time
	version     int
}

// NewProject creates a new project aggregate
func NewProject(tenantID uuid.UUID, name, slug, description, gitRepo, gitBranch, visibility, gitProvider string, repoAuthID *uuid.UUID, createdBy *uuid.UUID, isDraft bool) (*Project, error) {
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant ID is required")
	}

	if slug == "" {
		slug = generateSlug(name)
	}

	if err := validateProjectData(name, slug, description, gitRepo, gitBranch, gitProvider); err != nil {
		return nil, err
	}

	// Default visibility to private if not provided
	vis := ProjectVisibilityPrivate
	if visibility != "" {
		vis = ProjectVisibility(visibility)
	}

	if gitBranch == "" {
		gitBranch = "main"
	}

	if gitProvider == "" {
		gitProvider = "generic"
	}

	return &Project{
		id:          uuid.New(),
		tenantID:    tenantID,
		name:        name,
		slug:        slug,
		description: description,
		status:      ProjectStatusActive,
		visibility:  vis,
		gitRepo:     gitRepo,
		gitBranch:   gitBranch,
		gitProvider: gitProvider,
		repoAuthID:  repoAuthID,
		createdBy:   createdBy,
		isDraft:     isDraft,
		buildCount:  0,
		createdAt:   time.Now().UTC(),
		updatedAt:   time.Now().UTC(),
		version:     1,
	}, nil
}

// generateSlug converts a name to a URL-friendly slug
func generateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)
	// Replace spaces and underscores with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	// Remove any characters that aren't alphanumeric or hyphens
	re := regexp.MustCompile("[^a-z0-9-]")
	slug = re.ReplaceAllString(slug, "")
	// Replace multiple hyphens with single hyphen
	re = regexp.MustCompile("-+")
	slug = re.ReplaceAllString(slug, "-")
	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")
	// Limit to 100 characters
	if len(slug) > 100 {
		slug = slug[:100]
	}
	return slug
}

// GenerateSlug converts a value to a normalized URL-friendly slug.
func GenerateSlug(value string) string {
	return generateSlug(value)
}

// validateProjectData validates project input data
func validateProjectData(name, slug, description, gitRepo, gitBranch, gitProvider string) error {
	if name == "" {
		return errors.New("project name is required")
	}
	if len(name) < 3 || len(name) > 100 {
		return errors.New("project name must be between 3 and 100 characters")
	}
	if len(description) > 1000 {
		return errors.New("project description must not exceed 1000 characters")
	}
	if slug == "" {
		return errors.New("project slug is required")
	}
	if len(slug) > 100 {
		return errors.New("project slug must not exceed 100 characters")
	}
	if slug != generateSlug(slug) {
		return errors.New("project slug must contain only lowercase letters, numbers, and hyphens")
	}
	if gitRepo != "" && len(gitRepo) > 500 {
		return errors.New("git repository URL must not exceed 500 characters")
	}
	if gitBranch != "" && len(gitBranch) > 200 {
		return errors.New("git branch must not exceed 200 characters")
	}
	if gitProvider != "" && len(gitProvider) > 100 {
		return errors.New("git provider key must not exceed 100 characters")
	}
	// TODO: Add URL validation for gitRepo if provided
	return nil
}

// ID returns the project ID
func (p *Project) ID() uuid.UUID {
	return p.id
}

// TenantID returns the tenant ID
func (p *Project) TenantID() uuid.UUID {
	return p.tenantID
}

// Name returns the project name
func (p *Project) Name() string {
	return p.name
}

// Slug returns the project slug
func (p *Project) Slug() string {
	return p.slug
}

// Description returns the project description
func (p *Project) Description() string {
	return p.description
}

// Status returns the project status
func (p *Project) Status() ProjectStatus {
	return p.status
}

// Visibility returns the project visibility
func (p *Project) Visibility() ProjectVisibility {
	return p.visibility
}

// GitRepo returns the git repository URL
func (p *Project) GitRepo() string {
	return p.gitRepo
}

// GitBranch returns the git branch for the project
func (p *Project) GitBranch() string {
	return p.gitBranch
}

// GitProvider returns the selected git provider key
func (p *Project) GitProvider() string {
	return p.gitProvider
}

// RepositoryAuthID returns the active repository auth ID (if set)
func (p *Project) RepositoryAuthID() *uuid.UUID {
	return p.repoAuthID
}

// CreatedBy returns the creator user ID (if available)
func (p *Project) CreatedBy() *uuid.UUID {
	return p.createdBy
}

// IsDraft returns true if the project is a draft
func (p *Project) IsDraft() bool {
	return p.isDraft
}

// BuildCount returns the number of builds
func (p *Project) BuildCount() int {
	return p.buildCount
}

// CreatedAt returns the creation timestamp
func (p *Project) CreatedAt() time.Time {
	return p.createdAt
}

// UpdatedAt returns the last update timestamp
func (p *Project) UpdatedAt() time.Time {
	return p.updatedAt
}

// DeletedAt returns the soft delete timestamp
func (p *Project) DeletedAt() *time.Time {
	return p.deletedAt
}

// Version returns the version for concurrency control
func (p *Project) Version() int {
	return p.version
}

// Update updates the project with new data
func (p *Project) Update(name, slug, description, gitRepo, gitBranch, gitProvider string, repoAuthID *uuid.UUID) error {
	if err := validateProjectData(name, slug, description, gitRepo, gitBranch, gitProvider); err != nil {
		return err
	}

	p.name = name
	p.slug = slug
	p.description = description
	p.gitRepo = gitRepo
	if gitBranch != "" {
		p.gitBranch = gitBranch
	}
	if gitProvider != "" {
		p.gitProvider = gitProvider
	}
	p.repoAuthID = repoAuthID
	p.updatedAt = time.Now().UTC()
	p.version++
	return nil
}

// SetDraft updates the draft flag
func (p *Project) SetDraft(isDraft bool) {
	p.isDraft = isDraft
	p.updatedAt = time.Now().UTC()
	p.version++
}

// Archive marks the project as archived
func (p *Project) Archive() error {
	if p.status == ProjectStatusArchived {
		return errors.New("project is already archived")
	}
	p.status = ProjectStatusArchived
	p.updatedAt = time.Now().UTC()
	p.version++
	return nil
}

// Suspend marks the project as suspended
func (p *Project) Suspend() error {
	if p.status == ProjectStatusSuspended {
		return errors.New("project is already suspended")
	}
	p.status = ProjectStatusSuspended
	p.updatedAt = time.Now().UTC()
	p.version++
	return nil
}

// Activate marks the project as active
func (p *Project) Activate() error {
	if p.status == ProjectStatusActive {
		return errors.New("project is already active")
	}
	p.status = ProjectStatusActive
	p.updatedAt = time.Now().UTC()
	p.version++
	return nil
}

// Delete performs soft delete
func (p *Project) Delete() error {
	if p.deletedAt != nil {
		return errors.New("project is already deleted")
	}
	now := time.Now().UTC()
	p.deletedAt = &now
	p.updatedAt = now
	p.version++
	return nil
}

// Restore undoes soft delete
func (p *Project) Restore() error {
	if p.deletedAt == nil {
		return errors.New("project is not deleted")
	}
	p.deletedAt = nil
	p.updatedAt = time.Now().UTC()
	p.version++
	return nil
}

// IncrementBuildCount increments the build count
func (p *Project) IncrementBuildCount() {
	p.buildCount++
	p.updatedAt = time.Now().UTC()
	p.version++
}

// IsDeleted returns true if the project is soft deleted
func (p *Project) IsDeleted() bool {
	return p.deletedAt != nil
}

// IsActive returns true if the project is active
func (p *Project) IsActive() bool {
	return p.status == ProjectStatusActive && !p.IsDeleted()
}

// NewProjectFromExisting reconstructs a project from existing data (for repository use)
func NewProjectFromExisting(
	id, tenantID uuid.UUID,
	name, slug, description, gitRepo, gitBranch, gitProvider string,
	status ProjectStatus,
	visibility string,
	repoAuthID *uuid.UUID,
	createdBy *uuid.UUID,
	isDraft bool,
	buildCount int,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
	version int,
) *Project {
	vis := ProjectVisibility(visibility)
	if visibility == "" {
		vis = ProjectVisibilityPrivate
	}

	if gitBranch == "" {
		gitBranch = "main"
	}
	if gitProvider == "" {
		gitProvider = "generic"
	}

	return &Project{
		id:          id,
		tenantID:    tenantID,
		name:        name,
		slug:        slug,
		description: description,
		status:      status,
		visibility:  vis,
		gitRepo:     gitRepo,
		gitBranch:   gitBranch,
		gitProvider: gitProvider,
		repoAuthID:  repoAuthID,
		createdBy:   createdBy,
		isDraft:     isDraft,
		buildCount:  buildCount,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		deletedAt:   deletedAt,
		version:     version,
	}
}
