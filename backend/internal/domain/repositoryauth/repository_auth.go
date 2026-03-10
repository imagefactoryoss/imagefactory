package repositoryauth

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// AuthType represents the type of authentication
type AuthType string

const (
	AuthTypeSSH   AuthType = "ssh_key"
	AuthTypeToken AuthType = "token"
	AuthTypeBasic AuthType = "basic_auth"
	AuthTypeOAuth AuthType = "oauth"
)

// Scope represents repository auth scope.
type Scope string

const (
	ScopeTenant  Scope = "tenant"
	ScopeProject Scope = "project"
)

// RepositoryAuth represents repository authentication credentials
type RepositoryAuth struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	TenantID       uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	ProjectID      *uuid.UUID `json:"project_id,omitempty" db:"project_id"`
	Scope          Scope      `json:"scope"`
	Name           string     `json:"name" db:"name"`
	Description    string     `json:"description,omitempty" db:"description"`
	AuthType       AuthType   `json:"auth_type" db:"auth_type"`
	credentialData []byte
	Username       string    `json:"-" db:"username"` // Not exposed in JSON for security
	SSHKey         string    `json:"-" db:"ssh_key"`  // Not exposed in JSON for security
	Token          string    `json:"-" db:"token"`    // Not exposed in JSON for security
	Password       string    `json:"-" db:"password"` // Not exposed in JSON for security
	IsActive       bool      `json:"is_active" db:"is_active"`
	Version        int       `json:"version" db:"version"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
	CreatedBy      uuid.UUID `json:"created_by" db:"created_by"`
	UpdatedBy      uuid.UUID `json:"updated_by" db:"updated_by"`
}

// RepositoryAuthSummary represents a repository auth with project context.
type RepositoryAuthSummary struct {
	ID             uuid.UUID `json:"id"`
	ProjectID      uuid.UUID `json:"project_id"`
	ProjectName    string    `json:"project_name"`
	GitProviderKey string    `json:"git_provider_key,omitempty"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	AuthType       AuthType  `json:"auth_type"`
	IsActive       bool      `json:"is_active"`
	CreatedBy      uuid.UUID `json:"created_by"`
	CreatedByEmail string    `json:"created_by_email,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ProjectUsage represents an active project referencing a repository auth.
type ProjectUsage struct {
	ProjectID   uuid.UUID `json:"project_id"`
	ProjectName string    `json:"project_name"`
}

// RepositoryAuthCreate represents the data needed to create a new repository auth
type RepositoryAuthCreate struct {
	TenantID    uuid.UUID  `json:"tenant_id" validate:"required"`
	ProjectID   *uuid.UUID `json:"project_id,omitempty"`
	Name        string     `json:"name" validate:"required,min=1,max=100"`
	Description string     `json:"description,omitempty" validate:"max=500"`
	AuthType    AuthType   `json:"auth_type" validate:"required,oneof=ssh_key token basic_auth oauth"`
	Username    string     `json:"username,omitempty" validate:"max=100"`
	SSHKey      string     `json:"ssh_key,omitempty"`
	Token       string     `json:"token,omitempty"`
	Password    string     `json:"password,omitempty"`
}

// RepositoryAuthUpdate represents the data needed to update repository auth
type RepositoryAuthUpdate struct {
	Name        *string   `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string   `json:"description,omitempty" validate:"omitempty,max=500"`
	AuthType    *AuthType `json:"auth_type,omitempty" validate:"omitempty,oneof=ssh_key token basic_auth oauth"`
	Username    *string   `json:"username,omitempty" validate:"omitempty,max=100"`
	SSHKey      *string   `json:"ssh_key,omitempty"`
	Token       *string   `json:"token,omitempty"`
	Password    *string   `json:"password,omitempty"`
}

// Validation errors
var (
	ErrInvalidProjectID            = errors.New("invalid project ID")
	ErrInvalidTenantID             = errors.New("invalid tenant ID")
	ErrNameRequired                = errors.New("name is required")
	ErrAuthTypeRequired            = errors.New("auth type is required")
	ErrInvalidAuthType             = errors.New("invalid auth type")
	ErrSSHKeyRequired              = errors.New("SSH key is required for SSH auth type")
	ErrTokenRequired               = errors.New("token is required for token auth type")
	ErrBasicAuthRequired           = errors.New("username and password are required for basic auth type")
	ErrDuplicateRepositoryAuthName = errors.New("repository authentication name already exists for this scope")
	ErrRepositoryAuthNotFound      = errors.New("repository authentication not found")
)

// RepositoryAuthInUseError is returned when a tenant-scoped repository auth
// is still referenced by active projects.
type RepositoryAuthInUseError struct {
	AuthID   uuid.UUID
	AuthName string
	Projects []ProjectUsage
}

func (e *RepositoryAuthInUseError) Error() string {
	return "repository authentication is currently used by active projects"
}

// NewRepositoryAuth creates a new repository authentication configuration
func NewRepositoryAuth(tenantID uuid.UUID, projectID *uuid.UUID, name, description string, authType AuthType, credentialData []byte, createdBy uuid.UUID) (*RepositoryAuth, error) {
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant ID is required")
	}

	if err := validateRepositoryAuthData(name, description, authType, credentialData); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	scope := ScopeTenant
	if projectID != nil {
		scope = ScopeProject
	}

	return &RepositoryAuth{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ProjectID:      projectID,
		Scope:          scope,
		Name:           name,
		Description:    description,
		AuthType:       authType,
		credentialData: credentialData,
		Username:       "", // Will be populated from credentialData
		SSHKey:         "", // Will be populated from credentialData
		Token:          "", // Will be populated from credentialData
		Password:       "", // Will be populated from credentialData
		IsActive:       true,
		Version:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      createdBy,
		UpdatedBy:      createdBy,
	}, nil
}

// ID returns the repository authentication ID
func (r *RepositoryAuth) GetID() uuid.UUID {
	return r.ID
}

// TenantID returns the tenant ID
func (r *RepositoryAuth) GetTenantID() uuid.UUID {
	return r.TenantID
}

// ProjectID returns the project ID
func (r *RepositoryAuth) GetProjectID() *uuid.UUID {
	return r.ProjectID
}

// Name returns the authentication configuration name
func (r *RepositoryAuth) GetName() string {
	return r.Name
}

// Description returns the authentication configuration description
func (r *RepositoryAuth) GetDescription() string {
	return r.Description
}

// AuthType returns the authentication type
func (r *RepositoryAuth) GetAuthType() AuthType {
	return r.AuthType
}

// CredentialData returns the encrypted credential data
func (r *RepositoryAuth) CredentialData() []byte {
	// Return a copy to prevent external modification
	data := make([]byte, len(r.credentialData))
	copy(data, r.credentialData)
	return data
}

// CreatedAt returns the creation timestamp
func (r *RepositoryAuth) GetCreatedAt() time.Time {
	return r.CreatedAt
}

// UpdatedAt returns the last update timestamp
func (r *RepositoryAuth) GetUpdatedAt() time.Time {
	return r.UpdatedAt
}

// CreatedBy returns the user who created this configuration
func (r *RepositoryAuth) GetCreatedBy() uuid.UUID {
	return r.CreatedBy
}

// IsActive returns whether this authentication configuration is active
func (r *RepositoryAuth) GetIsActive() bool {
	return r.IsActive
}

// Version returns the version for concurrency control
func (r *RepositoryAuth) GetVersion() int {
	return r.Version
}

// NewRepositoryAuthFromExisting reconstructs a repository authentication from existing data
func NewRepositoryAuthFromExisting(
	id, tenantID uuid.UUID,
	projectID *uuid.UUID,
	name, description, authType string,
	credentialData []byte,
	isActive bool,
	createdBy uuid.UUID,
	createdAt, updatedAt time.Time,
	version int,
) *RepositoryAuth {
	scope := ScopeTenant
	if projectID != nil {
		scope = ScopeProject
	}
	return &RepositoryAuth{
		ID:             id,
		TenantID:       tenantID,
		ProjectID:      projectID,
		Scope:          scope,
		Name:           name,
		Description:    description,
		AuthType:       AuthType(authType),
		credentialData: credentialData,
		Username:       "", // Will be populated from credentialData
		SSHKey:         "", // Will be populated from credentialData
		Token:          "", // Will be populated from credentialData
		Password:       "", // Will be populated from credentialData
		IsActive:       isActive,
		Version:        version,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		CreatedBy:      createdBy,
		UpdatedBy:      createdBy,
	}
}

// Update updates the repository authentication configuration
func (r *RepositoryAuth) Update(name, description string, credentialData []byte) error {
	if err := validateRepositoryAuthData(name, description, r.AuthType, credentialData); err != nil {
		return err
	}

	r.Name = name
	r.Description = description
	r.credentialData = credentialData
	r.UpdatedAt = time.Now().UTC()

	return nil
}

func validateRepositoryAuthData(name, description string, authType AuthType, credentialData []byte) error {
	if name == "" {
		return errors.New("repository authentication name is required")
	}
	if len(name) < 1 || len(name) > 255 {
		return errors.New("repository authentication name must be between 1 and 255 characters")
	}
	if len(description) > 1000 {
		return errors.New("description must not exceed 1000 characters")
	}
	if !authType.IsValid() {
		return ErrInvalidAuthType
	}
	if len(credentialData) == 0 {
		return errors.New("credential data is required")
	}
	if len(credentialData) > 10000 { // Reasonable limit for encrypted data
		return errors.New("credential data is too large")
	}
	return nil
}

// IsValid checks if the authentication type is valid
func (at AuthType) IsValid() bool {
	switch at {
	case AuthTypeSSH, AuthTypeToken, AuthTypeBasic, AuthTypeOAuth:
		return true
	default:
		return false
	}
}

// Validate performs validation on the RepositoryAuthCreate
func (r *RepositoryAuthCreate) Validate() error {
	if r.TenantID == uuid.Nil {
		return ErrInvalidTenantID
	}
	if r.Name == "" {
		return ErrNameRequired
	}
	if r.AuthType == "" {
		return ErrAuthTypeRequired
	}

	// Validate based on auth type
	switch r.AuthType {
	case AuthTypeSSH:
		if r.SSHKey == "" {
			return ErrSSHKeyRequired
		}
	case AuthTypeToken:
		if r.Token == "" {
			return ErrTokenRequired
		}
	case AuthTypeBasic:
		if r.Username == "" || r.Password == "" {
			return ErrBasicAuthRequired
		}
	case AuthTypeOAuth:
		// OAuth validation would go here if needed
	default:
		return ErrInvalidAuthType
	}

	return nil
}

// Validate performs validation on the RepositoryAuthUpdate
func (r *RepositoryAuthUpdate) Validate() error {
	if r.Name != nil && *r.Name == "" {
		return ErrNameRequired
	}
	if r.AuthType != nil && *r.AuthType == "" {
		return ErrAuthTypeRequired
	}

	// Validate based on auth type if provided
	if r.AuthType != nil {
		switch *r.AuthType {
		case AuthTypeSSH:
			if r.SSHKey != nil && *r.SSHKey == "" {
				return ErrSSHKeyRequired
			}
		case AuthTypeToken:
			if r.Token != nil && *r.Token == "" {
				return ErrTokenRequired
			}
		case AuthTypeBasic:
			if r.Username != nil && *r.Username == "" {
				return ErrBasicAuthRequired
			}
			if r.Password != nil && *r.Password == "" {
				return ErrBasicAuthRequired
			}
		case AuthTypeOAuth:
			// OAuth validation would go here if needed
		default:
			return ErrInvalidAuthType
		}
	}

	return nil
}
