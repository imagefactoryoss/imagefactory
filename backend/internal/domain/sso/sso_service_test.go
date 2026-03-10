package sso

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockSAMLRepo struct {
	items map[uuid.UUID]*SAMLProvider
}

func newMockSAMLRepo() *mockSAMLRepo {
	return &mockSAMLRepo{items: map[uuid.UUID]*SAMLProvider{}}
}
func (m *mockSAMLRepo) Save(ctx context.Context, provider *SAMLProvider) error {
	m.items[provider.ID] = provider
	return nil
}
func (m *mockSAMLRepo) Update(ctx context.Context, provider *SAMLProvider) error {
	m.items[provider.ID] = provider
	return nil
}
func (m *mockSAMLRepo) FindByID(ctx context.Context, id uuid.UUID) (*SAMLProvider, error) {
	p, ok := m.items[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return p, nil
}
func (m *mockSAMLRepo) FindByEntityID(ctx context.Context, entityID string) (*SAMLProvider, error) {
	return nil, errors.New("not found")
}
func (m *mockSAMLRepo) FindAll(ctx context.Context) ([]*SAMLProvider, error) {
	out := make([]*SAMLProvider, 0, len(m.items))
	for _, p := range m.items {
		out = append(out, p)
	}
	return out, nil
}
func (m *mockSAMLRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.items, id)
	return nil
}

type mockOIDCRepo struct {
	items map[uuid.UUID]*OpenIDConnectProvider
}

func newMockOIDCRepo() *mockOIDCRepo {
	return &mockOIDCRepo{items: map[uuid.UUID]*OpenIDConnectProvider{}}
}
func (m *mockOIDCRepo) Save(ctx context.Context, provider *OpenIDConnectProvider) error {
	m.items[provider.ID] = provider
	return nil
}
func (m *mockOIDCRepo) Update(ctx context.Context, provider *OpenIDConnectProvider) error {
	m.items[provider.ID] = provider
	return nil
}
func (m *mockOIDCRepo) FindByID(ctx context.Context, id uuid.UUID) (*OpenIDConnectProvider, error) {
	p, ok := m.items[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return p, nil
}
func (m *mockOIDCRepo) FindByIssuer(ctx context.Context, issuer string) (*OpenIDConnectProvider, error) {
	return nil, errors.New("not found")
}
func (m *mockOIDCRepo) FindAll(ctx context.Context) ([]*OpenIDConnectProvider, error) {
	out := make([]*OpenIDConnectProvider, 0, len(m.items))
	for _, p := range m.items {
		out = append(out, p)
	}
	return out, nil
}
func (m *mockOIDCRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.items, id)
	return nil
}

func TestCreateSAMLProviderValidation(t *testing.T) {
	svc := NewService(newMockSAMLRepo(), newMockOIDCRepo(), zap.NewNop())
	_, err := svc.CreateSAMLProvider(context.Background(), SAMLProviderCreateRequest{
		Name: "bad",
	})
	if err == nil {
		t.Fatal("expected validation error for missing required fields")
	}
}

func TestCreateSAMLProviderAndEnableFlow(t *testing.T) {
	samlRepo := newMockSAMLRepo()
	oidcRepo := newMockOIDCRepo()
	oidcEnabled := &OpenIDConnectProvider{
		ID:      uuid.New(),
		Name:    "oidc1",
		Issuer:  "https://issuer.example.com",
		Enabled: true,
	}
	oidcRepo.items[oidcEnabled.ID] = oidcEnabled

	svc := NewService(samlRepo, oidcRepo, zap.NewNop())
	enabled := true
	provider, err := svc.CreateSAMLProvider(context.Background(), SAMLProviderCreateRequest{
		Name:        "idp1",
		EntityID:    "urn:test:idp1",
		SSOURL:      "https://idp.example.com/sso",
		SLOURL:      "https://idp.example.com/slo",
		Certificate: "-----BEGIN CERTIFICATE-----abc-----END CERTIFICATE-----",
		Position:    SSOPositionIdentityProvider,
		Enabled:     &enabled,
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if provider == nil || provider.Metadata == "" {
		t.Fatal("expected provider with generated metadata")
	}
	if oidcRepo.items[oidcEnabled.ID].Enabled {
		t.Fatal("expected previously enabled OIDC provider to be disabled")
	}
}

func TestCreateOIDCProviderValidation(t *testing.T) {
	svc := NewService(newMockSAMLRepo(), newMockOIDCRepo(), zap.NewNop())
	_, err := svc.CreateOIDCProvider(context.Background(), OIDCProviderCreateRequest{
		Name: "oidc",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSetOIDCProviderEnabledDisablesSAML(t *testing.T) {
	samlRepo := newMockSAMLRepo()
	oidcRepo := newMockOIDCRepo()
	samlEnabled := &SAMLProvider{
		ID:      uuid.New(),
		Name:    "saml1",
		Enabled: true,
	}
	samlRepo.items[samlEnabled.ID] = samlEnabled

	oidc := &OpenIDConnectProvider{
		ID:      uuid.New(),
		Name:    "oidc1",
		Issuer:  "https://issuer.example.com",
		Enabled: false,
	}
	oidcRepo.items[oidc.ID] = oidc

	svc := NewService(samlRepo, oidcRepo, zap.NewNop())
	if err := svc.SetOIDCProviderEnabled(context.Background(), oidc.ID, true); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !oidcRepo.items[oidc.ID].Enabled {
		t.Fatal("expected oidc provider enabled")
	}
	if samlRepo.items[samlEnabled.ID].Enabled {
		t.Fatal("expected enabled saml provider to be disabled")
	}
}
