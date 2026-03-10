package gitprovider

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

type mockProviderRepo struct {
    providers []*Provider
    byKey     map[string]*Provider
}

func (m *mockProviderRepo) FindActive(ctx context.Context) ([]*Provider, error) {
    return m.providers, nil
}

func (m *mockProviderRepo) FindByKey(ctx context.Context, key string) (*Provider, error) {
    if m.byKey == nil {
        return nil, nil
    }
    return m.byKey[key], nil
}

func TestServiceListActiveProviders(t *testing.T) {
    t.Parallel()

    provider := NewProviderFromExisting(
        uuid.New(),
        "github",
        "GitHub",
        ProviderTypeHosted,
        "https://api.github.com",
        true,
        true,
    )

    repo := &mockProviderRepo{providers: []*Provider{provider}}
    service := NewService(repo)

    providers, err := service.ListActiveProviders(context.Background())
    require.NoError(t, err)
    require.Len(t, providers, 1)
    assert.Equal(t, provider.Key(), providers[0].Key())
}

func TestServiceGetActiveProviderByKey(t *testing.T) {
    t.Parallel()

    provider := NewProviderFromExisting(
        uuid.New(),
        "gitlab",
        "GitLab",
        ProviderTypeHosted,
        "https://gitlab.com/api/v4",
        true,
        true,
    )

    repo := &mockProviderRepo{byKey: map[string]*Provider{"gitlab": provider}}
    service := NewService(repo)

    found, err := service.GetActiveProviderByKey(context.Background(), "gitlab")
    require.NoError(t, err)
    require.NotNil(t, found)
    assert.Equal(t, "gitlab", found.Key())
}
