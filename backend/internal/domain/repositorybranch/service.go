package repositorybranch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/gitprovider"
	"github.com/srikarm/image-factory/internal/domain/repositoryauth"
)

type AuthService interface {
	GetRepositoryAuth(ctx context.Context, id uuid.UUID) (*repositoryauth.RepositoryAuth, error)
	DecryptCredentials(ctx context.Context, id uuid.UUID) (map[string]interface{}, error)
}

type ProviderService interface {
	GetActiveProviderByKey(ctx context.Context, key string) (*gitprovider.Provider, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type GitRunner interface {
	ListRemoteBranches(ctx context.Context, repoURL string, auth GitAuth) ([]string, error)
}

type GitAuth struct {
	AuthType repositoryauth.AuthType
	Username string
	Password string
	Token    string
	SSHKey   string
}

type ListBranchesRequest struct {
	ProjectID   uuid.UUID
	AuthID      uuid.UUID
	RepoURL     string
	ProviderKey string
}

type Service struct {
	authService     AuthService
	providerService ProviderService
	httpClient      HTTPClient
	gitRunner       GitRunner
	logger          *zap.Logger
}

func NewService(authService AuthService, providerService ProviderService, httpClient HTTPClient, gitRunner GitRunner, logger *zap.Logger) *Service {
	return &Service{
		authService:     authService,
		providerService: providerService,
		httpClient:      httpClient,
		gitRunner:       gitRunner,
		logger:          logger,
	}
}

func (s *Service) ListBranches(ctx context.Context, req ListBranchesRequest) ([]string, error) {
	if req.ProjectID == uuid.Nil {
		return nil, errors.New("project ID is required")
	}
	if req.AuthID == uuid.Nil {
		return nil, errors.New("auth ID is required")
	}
	if strings.TrimSpace(req.RepoURL) == "" {
		return nil, errors.New("repository URL is required")
	}
	if strings.TrimSpace(req.ProviderKey) == "" {
		return nil, errors.New("provider key is required")
	}

	auth, err := s.authService.GetRepositoryAuth(ctx, req.AuthID)
	if err != nil {
		return nil, fmt.Errorf("failed to load repository auth: %w", err)
	}
	if auth == nil {
		return nil, errors.New("repository auth not found")
	}
	if authProjectID := auth.GetProjectID(); authProjectID != nil && *authProjectID != req.ProjectID {
		return nil, errors.New("repository auth does not belong to project")
	}

	credentials, err := s.authService.DecryptCredentials(ctx, req.AuthID)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	provider, err := s.providerService.GetActiveProviderByKey(ctx, req.ProviderKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load git provider: %w", err)
	}
	if provider == nil {
		return nil, errors.New("git provider not found")
	}

	gitAuth := GitAuth{
		AuthType: auth.GetAuthType(),
		Username: getString(credentials, "username"),
		Password: getString(credentials, "password"),
		Token:    getString(credentials, "token"),
		SSHKey:   getString(credentials, "ssh_key"),
	}

	if provider.SupportsAPI() && provider.APIBaseURL() != "" {
		branches, apiErr := s.fetchBranchesFromProviderAPI(ctx, provider, req.RepoURL, gitAuth)
		if apiErr == nil {
			return branches, nil
		}
	}

	return s.gitRunner.ListRemoteBranches(ctx, req.RepoURL, gitAuth)
}

func (s *Service) fetchBranchesFromProviderAPI(ctx context.Context, provider *gitprovider.Provider, repoURL string, auth GitAuth) ([]string, error) {
	owner, repo, err := parseRepoOwnerAndName(repoURL)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(provider.Key()) {
	case "github":
		endpoint := fmt.Sprintf("%s/repos/%s/%s/branches", strings.TrimRight(provider.APIBaseURL(), "/"), owner, repo)
		return s.fetchGitHubBranches(ctx, endpoint, auth)
	case "gitlab":
		projectPath := url.PathEscape(fmt.Sprintf("%s/%s", owner, repo))
		endpoint := fmt.Sprintf("%s/projects/%s/repository/branches", strings.TrimRight(provider.APIBaseURL(), "/"), projectPath)
		return s.fetchGitLabBranches(ctx, endpoint, auth)
	default:
		return nil, errors.New("provider API not supported")
	}
}

func (s *Service) fetchGitHubBranches(ctx context.Context, endpoint string, auth GitAuth) ([]string, error) {
	resp, err := s.doRequestWithRetry(ctx, http.MethodGet, endpoint, auth, "github")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github API error: %s", resp.Status)
	}

	var payload []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	branches := make([]string, 0, len(payload))
	for _, branch := range payload {
		if branch.Name != "" {
			branches = append(branches, branch.Name)
		}
	}
	return branches, nil
}

func (s *Service) fetchGitLabBranches(ctx context.Context, endpoint string, auth GitAuth) ([]string, error) {
	resp, err := s.doRequestWithRetry(ctx, http.MethodGet, endpoint, auth, "gitlab")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gitlab API error: %s", resp.Status)
	}

	var payload []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	branches := make([]string, 0, len(payload))
	for _, branch := range payload {
		if branch.Name != "" {
			branches = append(branches, branch.Name)
		}
	}
	return branches, nil
}

func applyAuthHeaders(req *http.Request, auth GitAuth, providerKey string) {
	switch auth.AuthType {
	case repositoryauth.AuthTypeToken:
		token := auth.Token
		if token == "" {
			return
		}
		if providerKey == "gitlab" {
			req.Header.Set("PRIVATE-TOKEN", token)
			return
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	case repositoryauth.AuthTypeBasic:
		req.SetBasicAuth(auth.Username, auth.Password)
	}
}

const (
	providerAPIMaxAttempts = 3
	providerAPIBaseDelay   = 200 * time.Millisecond
)

func (s *Service) doRequestWithRetry(ctx context.Context, method, endpoint string, auth GitAuth, providerKey string) (*http.Response, error) {
	if s.httpClient == nil {
		return nil, errors.New("http client not configured")
	}

	var lastErr error
	for attempt := 1; attempt <= providerAPIMaxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
		if err != nil {
			return nil, err
		}
		applyAuthHeaders(req, auth, providerKey)

		resp, err := s.httpClient.Do(req)
		if err == nil && !shouldRetryStatus(resp.StatusCode) {
			return resp, nil
		}

		if err != nil {
			lastErr = err
			s.logRetry(attempt, endpoint, 0, err)
		} else {
			lastErr = fmt.Errorf("provider API error: %s", resp.Status)
			s.logRetry(attempt, endpoint, resp.StatusCode, lastErr)
			resp.Body.Close()
		}

		if attempt < providerAPIMaxAttempts {
			if sleepErr := sleepWithContext(ctx, backoffDelay(attempt, providerAPIBaseDelay)); sleepErr != nil {
				return nil, sleepErr
			}
		}
	}

	return nil, lastErr
}

func (s *Service) logRetry(attempt int, endpoint string, status int, err error) {
	if s.logger == nil {
		return
	}
	fields := []zap.Field{
		zap.Int("attempt", attempt),
		zap.String("endpoint", endpoint),
	}
	if status > 0 {
		fields = append(fields, zap.Int("status", status))
	}
	s.logger.Warn("Provider API request retry", append(fields, zap.Error(err))...)
}

func shouldRetryStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

func backoffDelay(attempt int, base time.Duration) time.Duration {
	multiplier := 1 << (attempt - 1)
	return time.Duration(multiplier) * base
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func parseRepoOwnerAndName(repoURL string) (string, string, error) {
	normalized := strings.TrimSpace(repoURL)
	if normalized == "" {
		return "", "", errors.New("repository URL is required")
	}

	if strings.HasPrefix(normalized, "git@") {
		parts := strings.SplitN(normalized, ":", 2)
		if len(parts) != 2 {
			return "", "", errors.New("invalid SSH repository URL")
		}
		path := strings.TrimSuffix(parts[1], ".git")
		segments := strings.Split(path, "/")
		if len(segments) < 2 {
			return "", "", errors.New("invalid repository path")
		}
		owner := strings.Join(segments[:len(segments)-1], "/")
		repo := segments[len(segments)-1]
		return owner, repo, nil
	}

	parsed, err := url.Parse(normalized)
	if err != nil {
		return "", "", err
	}

	path := strings.Trim(parsed.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", errors.New("invalid repository path")
	}

	owner := strings.Join(parts[:len(parts)-1], "/")
	repo := parts[len(parts)-1]

	return owner, repo, nil
}

func getString(data map[string]interface{}, key string) string {
	if data == nil {
		return ""
	}
	value, ok := data[key]
	if !ok {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}
