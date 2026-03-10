package registryauth

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Scope string

const (
	ScopeTenant  Scope = "tenant"
	ScopeProject Scope = "project"
)

type AuthType string

const (
	AuthTypeBasicAuth        AuthType = "basic_auth"
	AuthTypeToken            AuthType = "token"
	AuthTypeDockerConfigJSON AuthType = "dockerconfigjson"
)

type RegistryAuth struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	ProjectID      *uuid.UUID `json:"project_id,omitempty"`
	Scope          Scope      `json:"scope"`
	Name           string     `json:"name"`
	Description    string     `json:"description,omitempty"`
	RegistryType   string     `json:"registry_type"`
	AuthType       AuthType   `json:"auth_type"`
	RegistryHost   string     `json:"registry_host"`
	credentialData []byte
	IsActive       bool      `json:"is_active"`
	IsDefault      bool      `json:"is_default"`
	CreatedBy      uuid.UUID `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type RegistryAuthCreate struct {
	TenantID     uuid.UUID  `json:"tenant_id"`
	ProjectID    *uuid.UUID `json:"project_id,omitempty"`
	Name         string     `json:"name"`
	Description  string     `json:"description,omitempty"`
	RegistryType string     `json:"registry_type"`
	AuthType     AuthType   `json:"auth_type"`
	RegistryHost string     `json:"registry_host"`
	IsDefault    bool       `json:"is_default"`
}

var (
	ErrRegistryAuthNotFound  = errors.New("registry authentication not found")
	ErrDuplicateName         = errors.New("registry authentication name already exists in scope")
	ErrNoRegistryAuthFound   = errors.New("no registry authentication found")
	ErrInvalidRegistryAuthID = errors.New("invalid registry authentication id")
)

func NewRegistryAuth(input RegistryAuthCreate, credentialData []byte, createdBy uuid.UUID) (*RegistryAuth, error) {
	if input.TenantID == uuid.Nil {
		return nil, errors.New("tenant_id is required")
	}
	if input.Name == "" {
		return nil, errors.New("name is required")
	}
	if input.RegistryType == "" {
		return nil, errors.New("registry_type is required")
	}
	if input.RegistryHost == "" {
		return nil, errors.New("registry_host is required")
	}
	if !input.AuthType.IsValid() {
		return nil, errors.New("invalid auth_type")
	}
	if len(credentialData) == 0 {
		return nil, errors.New("credential_data is required")
	}

	scope := ScopeTenant
	if input.ProjectID != nil {
		scope = ScopeProject
	}

	now := time.Now().UTC()
	return &RegistryAuth{
		ID:             uuid.New(),
		TenantID:       input.TenantID,
		ProjectID:      input.ProjectID,
		Scope:          scope,
		Name:           input.Name,
		Description:    input.Description,
		RegistryType:   input.RegistryType,
		AuthType:       input.AuthType,
		RegistryHost:   input.RegistryHost,
		credentialData: credentialData,
		IsActive:       true,
		IsDefault:      input.IsDefault,
		CreatedBy:      createdBy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func NewRegistryAuthFromExisting(
	id, tenantID uuid.UUID,
	projectID *uuid.UUID,
	name, description, registryType string,
	authType AuthType,
	registryHost string,
	credentialData []byte,
	isActive bool,
	isDefault bool,
	createdBy uuid.UUID,
	createdAt, updatedAt time.Time,
) *RegistryAuth {
	scope := ScopeTenant
	if projectID != nil {
		scope = ScopeProject
	}
	return &RegistryAuth{
		ID:             id,
		TenantID:       tenantID,
		ProjectID:      projectID,
		Scope:          scope,
		Name:           name,
		Description:    description,
		RegistryType:   registryType,
		AuthType:       authType,
		RegistryHost:   registryHost,
		credentialData: credentialData,
		IsActive:       isActive,
		IsDefault:      isDefault,
		CreatedBy:      createdBy,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}

func (a *RegistryAuth) CredentialData() []byte {
	data := make([]byte, len(a.credentialData))
	copy(data, a.credentialData)
	return data
}

func (at AuthType) IsValid() bool {
	switch at {
	case AuthTypeBasicAuth, AuthTypeToken, AuthTypeDockerConfigJSON:
		return true
	default:
		return false
	}
}
