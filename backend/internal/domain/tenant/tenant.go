package tenant

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Domain errors
var (
	ErrTenantNotFound    = errors.New("tenant not found")
	ErrTenantExists      = errors.New("tenant already exists")
	ErrInvalidTenantID   = errors.New("invalid tenant ID")
	ErrInvalidTenantName = errors.New("invalid tenant name")
)

// TenantStatus represents the status of a tenant
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusPending   TenantStatus = "pending"
	TenantStatusDeleted   TenantStatus = "deleted"
)

// ResourceQuota represents resource limits for a tenant
type ResourceQuota struct {
	MaxBuilds         int     `json:"max_builds"`
	MaxImages         int     `json:"max_images"`
	MaxStorageGB      float64 `json:"max_storage_gb"`
	MaxConcurrentJobs int     `json:"max_concurrent_jobs"`
}

// TenantConfig represents tenant-specific configuration
type TenantConfig struct {
	BuildTimeout         time.Duration          `json:"build_timeout"`
	AllowedImageTypes    []string               `json:"allowed_image_types"`
	SecurityPolicies     map[string]interface{} `json:"security_policies"`
	NotificationSettings map[string]interface{} `json:"notification_settings"`
}

// Tenant represents the tenant aggregate root
type Tenant struct {
	id          uuid.UUID
	numericID   int
	companyID   uuid.UUID
	tenantCode  string
	name        string
	slug        string
	description string
	status      TenantStatus
	quota       ResourceQuota
	config      TenantConfig
	createdAt   time.Time
	updatedAt   time.Time
	version     int // for optimistic concurrency control
}

// NewTenant creates a new tenant aggregate
func NewTenant(id uuid.UUID, companyID uuid.UUID, tenantCode, name, slug, description string) (*Tenant, error) {
	if name == "" {
		return nil, ErrInvalidTenantName
	}
	if tenantCode == "" {
		return nil, errors.New("tenant code cannot be empty")
	}
	if len(tenantCode) > 8 {
		return nil, errors.New("tenant code cannot exceed 8 characters")
	}

	// Generate default resource quota
	defaultQuota := ResourceQuota{
		MaxBuilds:         100,
		MaxImages:         500,
		MaxStorageGB:      100.0,
		MaxConcurrentJobs: 5,
	}

	// Generate default config
	defaultConfig := TenantConfig{
		BuildTimeout:         30 * time.Minute,
		AllowedImageTypes:    []string{"container", "vm"},
		SecurityPolicies:     make(map[string]interface{}),
		NotificationSettings: make(map[string]interface{}),
	}

	return &Tenant{
		id:          id,
		companyID:   companyID,
		tenantCode:  tenantCode,
		name:        name,
		slug:        slug,
		description: description,
		status:      TenantStatusPending,
		quota:       defaultQuota,
		config:      defaultConfig,
		createdAt:   time.Now().UTC(),
		updatedAt:   time.Now().UTC(),
		version:     1,
	}, nil
}

// NewTenantFromExisting creates a tenant from existing data (for repository reconstruction)
func NewTenantFromExisting(id uuid.UUID, numericID int, companyID uuid.UUID, tenantCode, name, slug, description string, status TenantStatus, quota ResourceQuota, config TenantConfig, createdAt, updatedAt time.Time, version int) (*Tenant, error) {
	if name == "" {
		return nil, ErrInvalidTenantName
	}
	if tenantCode == "" {
		return nil, errors.New("tenant code cannot be empty")
	}

	return &Tenant{
		id:          id,
		numericID:   numericID,
		companyID:   companyID,
		tenantCode:  tenantCode,
		name:        name,
		slug:        slug,
		description: description,
		status:      status,
		quota:       quota,
		config:      config,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		version:     version,
	}, nil
}

// ID returns the tenant ID
func (t *Tenant) ID() uuid.UUID {
	return t.id
}

// NumericID returns the tenant's unique numeric ID
func (t *Tenant) NumericID() int {
	return t.numericID
}

// CompanyID returns the company ID
func (t *Tenant) CompanyID() uuid.UUID {
	return t.companyID
}

// TenantCode returns the tenant code
func (t *Tenant) TenantCode() string {
	return t.tenantCode
}

// Name returns the tenant name
func (t *Tenant) Name() string {
	return t.name
}

// Slug returns the tenant slug
func (t *Tenant) Slug() string {
	return t.slug
}

// Description returns the tenant description
func (t *Tenant) Description() string {
	return t.description
}

// Status returns the tenant status
func (t *Tenant) Status() TenantStatus {
	return t.status
}

// Quota returns the resource quota
func (t *Tenant) Quota() ResourceQuota {
	return t.quota
}

// Config returns the tenant configuration
func (t *Tenant) Config() TenantConfig {
	return t.config
}

// CreatedAt returns the creation timestamp
func (t *Tenant) CreatedAt() time.Time {
	return t.createdAt
}

// UpdatedAt returns the last update timestamp
func (t *Tenant) UpdatedAt() time.Time {
	return t.updatedAt
}

// Version returns the version for concurrency control
func (t *Tenant) Version() int {
	return t.version
}

// Activate activates the tenant
func (t *Tenant) Activate() error {
	if t.status == TenantStatusDeleted {
		return errors.New("cannot activate deleted tenant")
	}

	t.status = TenantStatusActive
	t.updatedAt = time.Now().UTC()
	t.version++
	return nil
}

// Suspend suspends the tenant
func (t *Tenant) Suspend() error {
	if t.status == TenantStatusDeleted {
		return errors.New("cannot suspend deleted tenant")
	}

	t.status = TenantStatusSuspended
	t.updatedAt = time.Now().UTC()
	t.version++
	return nil
}

// Delete marks the tenant as deleted (soft delete)
// The tenant data is preserved in the database but marked as deleted
// This ensures referential integrity and maintains an audit trail
func (t *Tenant) Delete() error {
	if t.status == TenantStatusDeleted {
		return errors.New("tenant is already deleted")
	}

	t.status = TenantStatusDeleted
	t.updatedAt = time.Now().UTC()
	t.version++
	return nil
}

// UpdateQuota updates the resource quota
func (t *Tenant) UpdateQuota(quota ResourceQuota) {
	t.quota = quota
	t.updatedAt = time.Now().UTC()
	t.version++
}

// UpdateConfig updates the tenant configuration
func (t *Tenant) UpdateConfig(config TenantConfig) {
	t.config = config
	t.updatedAt = time.Now().UTC()
	t.version++
}

// IsActive returns true if the tenant is active
func (t *Tenant) IsActive() bool {
	return t.status == TenantStatusActive
}

// CanPerformBuild returns true if the tenant can perform builds
func (t *Tenant) CanPerformBuild() bool {
	return t.status == TenantStatusActive
}

// Domain events
type TenantEvent interface {
	TenantID() uuid.UUID
	OccurredAt() time.Time
}

type TenantCreated struct {
	tenantID   uuid.UUID
	tenantName string
	occurredAt time.Time
}

func NewTenantCreated(tenantID uuid.UUID, tenantName string) *TenantCreated {
	return &TenantCreated{
		tenantID:   tenantID,
		tenantName: tenantName,
		occurredAt: time.Now().UTC(),
	}
}

func (e *TenantCreated) TenantID() uuid.UUID {
	return e.tenantID
}

func (e *TenantCreated) TenantName() string {
	return e.tenantName
}

func (e *TenantCreated) OccurredAt() time.Time {
	return e.occurredAt
}

type TenantActivated struct {
	tenantID   uuid.UUID
	occurredAt time.Time
}

func NewTenantActivated(tenantID uuid.UUID) *TenantActivated {
	return &TenantActivated{
		tenantID:   tenantID,
		occurredAt: time.Now().UTC(),
	}
}

func (e *TenantActivated) TenantID() uuid.UUID {
	return e.tenantID
}

func (e *TenantActivated) OccurredAt() time.Time {
	return e.occurredAt
}
