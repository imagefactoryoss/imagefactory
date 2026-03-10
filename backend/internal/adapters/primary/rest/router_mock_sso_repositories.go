package rest

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/sso"
	"go.uber.org/zap"
)

// mockSAMLRepository is a temporary mock implementation for development.
type mockSAMLRepository struct {
	providers map[uuid.UUID]*sso.SAMLProvider
	logger    *zap.Logger
}

func (m *mockSAMLRepository) Save(ctx context.Context, provider *sso.SAMLProvider) error {
	m.providers[provider.ID] = provider
	return nil
}

func (m *mockSAMLRepository) Update(ctx context.Context, provider *sso.SAMLProvider) error {
	m.providers[provider.ID] = provider
	return nil
}

func (m *mockSAMLRepository) FindByID(ctx context.Context, id uuid.UUID) (*sso.SAMLProvider, error) {
	provider, exists := m.providers[id]
	if !exists {
		return nil, errors.New("SAML provider not found")
	}
	return provider, nil
}

func (m *mockSAMLRepository) FindByEntityID(ctx context.Context, entityID string) (*sso.SAMLProvider, error) {
	for _, provider := range m.providers {
		if provider.EntityID == entityID {
			return provider, nil
		}
	}
	return nil, errors.New("SAML provider not found")
}

func (m *mockSAMLRepository) FindAll(ctx context.Context) ([]*sso.SAMLProvider, error) {
	providers := make([]*sso.SAMLProvider, 0, len(m.providers))
	for _, provider := range m.providers {
		providers = append(providers, provider)
	}
	return providers, nil
}

func (m *mockSAMLRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.providers, id)
	return nil
}

// mockOIDCRepository is a temporary mock implementation for development.
type mockOIDCRepository struct {
	providers map[uuid.UUID]*sso.OpenIDConnectProvider
	logger    *zap.Logger
}

func (m *mockOIDCRepository) Save(ctx context.Context, provider *sso.OpenIDConnectProvider) error {
	m.providers[provider.ID] = provider
	return nil
}

func (m *mockOIDCRepository) Update(ctx context.Context, provider *sso.OpenIDConnectProvider) error {
	m.providers[provider.ID] = provider
	return nil
}

func (m *mockOIDCRepository) FindByID(ctx context.Context, id uuid.UUID) (*sso.OpenIDConnectProvider, error) {
	provider, exists := m.providers[id]
	if !exists {
		return nil, errors.New("OIDC provider not found")
	}
	return provider, nil
}

func (m *mockOIDCRepository) FindByIssuer(ctx context.Context, issuer string) (*sso.OpenIDConnectProvider, error) {
	for _, provider := range m.providers {
		if provider.Issuer == issuer {
			return provider, nil
		}
	}
	return nil, errors.New("OIDC provider not found")
}

func (m *mockOIDCRepository) FindAll(ctx context.Context) ([]*sso.OpenIDConnectProvider, error) {
	providers := make([]*sso.OpenIDConnectProvider, 0, len(m.providers))
	for _, provider := range m.providers {
		providers = append(providers, provider)
	}
	return providers, nil
}

func (m *mockOIDCRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.providers, id)
	return nil
}
