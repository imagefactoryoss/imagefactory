package gitprovider

import "context"

// Repository defines persistence for git providers.
type Repository interface {
    FindActive(ctx context.Context) ([]*Provider, error)
    FindByKey(ctx context.Context, key string) (*Provider, error)
}
