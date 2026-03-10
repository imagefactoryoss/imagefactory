package repositorybranch

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/gitprovider"
	"github.com/srikarm/image-factory/internal/domain/repositoryauth"
)

type fakeAuthService struct {
	auth        *repositoryauth.RepositoryAuth
	credentials map[string]interface{}
}

func (f *fakeAuthService) GetRepositoryAuth(ctx context.Context, id uuid.UUID) (*repositoryauth.RepositoryAuth, error) {
	return f.auth, nil
}

func (f *fakeAuthService) DecryptCredentials(ctx context.Context, id uuid.UUID) (map[string]interface{}, error) {
	return f.credentials, nil
}

type fakeProviderService struct {
	provider *gitprovider.Provider
}

func (f *fakeProviderService) GetActiveProviderByKey(ctx context.Context, key string) (*gitprovider.Provider, error) {
	return f.provider, nil
}

type fakeHTTPClient struct {
	lastRequest *http.Request
	response    *http.Response
	err         error
}

func (f *fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	f.lastRequest = req
	if f.err != nil {
		return nil, f.err
	}
	return f.response, nil
}

type fakeGitRunner struct {
	called   bool
	branches []string
}

func (f *fakeGitRunner) ListRemoteBranches(ctx context.Context, repoURL string, auth GitAuth) ([]string, error) {
	f.called = true
	return f.branches, nil
}

func TestListBranches_UsesProviderAPI(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	authID := uuid.New()
	auth := repositoryauth.NewRepositoryAuthFromExisting(
		authID,
		uuid.New(),
		&projectID,
		"auth",
		"",
		string(repositoryauth.AuthTypeToken),
		[]byte("secret"),
		true,
		uuid.New(),
		now(),
		now(),
		1,
	)

	provider := gitprovider.NewProviderFromExisting(
		uuid.New(),
		"github",
		"GitHub",
		gitprovider.ProviderTypeHosted,
		"https://api.github.com",
		true,
		true,
	)

	httpClient := &fakeHTTPClient{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`[{"name":"main"},{"name":"develop"}]`)),
		},
	}

	service := NewService(
		&fakeAuthService{auth: auth, credentials: map[string]interface{}{"token": "ghp_123"}},
		&fakeProviderService{provider: provider},
		httpClient,
		&fakeGitRunner{branches: []string{"fallback"}},
		zap.NewNop(),
	)

	branches, err := service.ListBranches(context.Background(), ListBranchesRequest{
		ProjectID:   projectID,
		AuthID:      authID,
		RepoURL:     "https://github.com/acme/widget.git",
		ProviderKey: "github",
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"main", "develop"}, branches)
	require.NotNil(t, httpClient.lastRequest)
	assert.Equal(t, "https://api.github.com/repos/acme/widget/branches", httpClient.lastRequest.URL.String())
	assert.Equal(t, "Bearer ghp_123", httpClient.lastRequest.Header.Get("Authorization"))
}

func TestListBranches_UsesGitRunnerForGeneric(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	authID := uuid.New()
	auth := repositoryauth.NewRepositoryAuthFromExisting(
		authID,
		uuid.New(),
		&projectID,
		"auth",
		"",
		string(repositoryauth.AuthTypeToken),
		[]byte("secret"),
		true,
		uuid.New(),
		now(),
		now(),
		1,
	)

	provider := gitprovider.NewProviderFromExisting(
		uuid.New(),
		"generic",
		"Generic Git",
		gitprovider.ProviderTypeGeneric,
		"",
		false,
		true,
	)

	gitRunner := &fakeGitRunner{branches: []string{"main"}}

	service := NewService(
		&fakeAuthService{auth: auth, credentials: map[string]interface{}{"token": "tok"}},
		&fakeProviderService{provider: provider},
		&fakeHTTPClient{},
		gitRunner,
		zap.NewNop(),
	)

	branches, err := service.ListBranches(context.Background(), ListBranchesRequest{
		ProjectID:   projectID,
		AuthID:      authID,
		RepoURL:     "https://example.com/org/repo.git",
		ProviderKey: "generic",
	})

	require.NoError(t, err)
	assert.True(t, gitRunner.called)
	assert.Equal(t, []string{"main"}, branches)
}

func TestListBranches_RejectsAuthProjectMismatch(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	authProjectID := uuid.New()
	authID := uuid.New()
	auth := repositoryauth.NewRepositoryAuthFromExisting(
		authID,
		uuid.New(),
		&authProjectID,
		"auth",
		"",
		string(repositoryauth.AuthTypeToken),
		[]byte("secret"),
		true,
		uuid.New(),
		now(),
		now(),
		1,
	)

	provider := gitprovider.NewProviderFromExisting(
		uuid.New(),
		"generic",
		"Generic Git",
		gitprovider.ProviderTypeGeneric,
		"",
		false,
		true,
	)

	service := NewService(
		&fakeAuthService{auth: auth, credentials: map[string]interface{}{"token": "tok"}},
		&fakeProviderService{provider: provider},
		&fakeHTTPClient{},
		&fakeGitRunner{branches: []string{"main"}},
		zap.NewNop(),
	)

	_, err := service.ListBranches(context.Background(), ListBranchesRequest{
		ProjectID:   projectID,
		AuthID:      authID,
		RepoURL:     "https://example.com/org/repo.git",
		ProviderKey: "generic",
	})

	require.Error(t, err)
}

func now() time.Time {
	return time.Now().UTC()
}
