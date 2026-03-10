package gitprovider

import "context"

// Service provides access to git provider configurations.
type Service struct {
    repo Repository
}

func NewService(repo Repository) *Service {
    return &Service{repo: repo}
}

func (s *Service) ListActiveProviders(ctx context.Context) ([]*Provider, error) {
    return s.repo.FindActive(ctx)
}

func (s *Service) GetActiveProviderByKey(ctx context.Context, key string) (*Provider, error) {
    if key == "" {
        return nil, ErrInvalidProviderKey
    }
    return s.repo.FindByKey(ctx, key)
}
