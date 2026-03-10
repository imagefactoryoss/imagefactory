package gitprovider

import (
    "errors"
    "strings"
    "time"

    "github.com/google/uuid"
)

type ProviderType string

const (
    ProviderTypeGeneric ProviderType = "generic"
    ProviderTypeHosted  ProviderType = "hosted"
)

var (
    ErrInvalidProviderKey   = errors.New("provider key is required")
    ErrInvalidDisplayName   = errors.New("provider display name is required")
    ErrInvalidProviderType  = errors.New("provider type is invalid")
)

// Provider represents a supported Git provider configuration.
type Provider struct {
    id          uuid.UUID
    key         string
    displayName string
    providerType ProviderType
    apiBaseURL  string
    supportsAPI bool
    isActive    bool
    createdAt   time.Time
    updatedAt   time.Time
}

func NewProvider(
    key, displayName string,
    providerType ProviderType,
    apiBaseURL string,
    supportsAPI bool,
) (*Provider, error) {
    if strings.TrimSpace(key) == "" {
        return nil, ErrInvalidProviderKey
    }
    if strings.TrimSpace(displayName) == "" {
        return nil, ErrInvalidDisplayName
    }
    if providerType != ProviderTypeGeneric && providerType != ProviderTypeHosted {
        return nil, ErrInvalidProviderType
    }

    now := time.Now().UTC()
    return &Provider{
        id:          uuid.New(),
        key:         key,
        displayName: displayName,
        providerType: providerType,
        apiBaseURL:  apiBaseURL,
        supportsAPI: supportsAPI,
        isActive:    true,
        createdAt:   now,
        updatedAt:   now,
    }, nil
}

func NewProviderFromExisting(
    id uuid.UUID,
    key, displayName string,
    providerType ProviderType,
    apiBaseURL string,
    supportsAPI bool,
    isActive bool,
) *Provider {
    now := time.Now().UTC()
    return &Provider{
        id:          id,
        key:         key,
        displayName: displayName,
        providerType: providerType,
        apiBaseURL:  apiBaseURL,
        supportsAPI: supportsAPI,
        isActive:    isActive,
        createdAt:   now,
        updatedAt:   now,
    }
}

func (p *Provider) ID() uuid.UUID {
    return p.id
}

func (p *Provider) Key() string {
    return p.key
}

func (p *Provider) DisplayName() string {
    return p.displayName
}

func (p *Provider) ProviderType() ProviderType {
    return p.providerType
}

func (p *Provider) APIBaseURL() string {
    return p.apiBaseURL
}

func (p *Provider) SupportsAPI() bool {
    return p.supportsAPI
}

func (p *Provider) IsActive() bool {
    return p.isActive
}
