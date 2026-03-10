package postgres

import (
    "context"
    "database/sql"

    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    "go.uber.org/zap"

    "github.com/srikarm/image-factory/internal/domain/gitprovider"
)

type GitProviderRepository struct {
    db     *sqlx.DB
    logger *zap.Logger
}

type gitProviderModel struct {
    ID           uuid.UUID `db:"id"`
    ProviderKey  string    `db:"provider_key"`
    DisplayName  string    `db:"display_name"`
    ProviderType string    `db:"provider_type"`
    APIBaseURL   sql.NullString `db:"api_base_url"`
    SupportsAPI  bool      `db:"supports_api"`
    IsActive     bool      `db:"is_active"`
}

func NewGitProviderRepository(db *sqlx.DB, logger *zap.Logger) *GitProviderRepository {
    return &GitProviderRepository{db: db, logger: logger}
}

func (r *GitProviderRepository) FindActive(ctx context.Context) ([]*gitprovider.Provider, error) {
    query := `
        SELECT id, provider_key, display_name, provider_type, api_base_url, supports_api, is_active
        FROM git_providers
        WHERE is_active = true
        ORDER BY display_name ASC
    `

    var models []gitProviderModel
    if err := r.db.SelectContext(ctx, &models, query); err != nil {
        r.logger.Error("Failed to list active git providers", zap.Error(err))
        return nil, err
    }

    providers := make([]*gitprovider.Provider, len(models))
    for i, model := range models {
        providers[i] = gitprovider.NewProviderFromExisting(
            model.ID,
            model.ProviderKey,
            model.DisplayName,
            gitprovider.ProviderType(model.ProviderType),
            model.APIBaseURL.String,
            model.SupportsAPI,
            model.IsActive,
        )
    }

    return providers, nil
}

func (r *GitProviderRepository) FindByKey(ctx context.Context, key string) (*gitprovider.Provider, error) {
    query := `
        SELECT id, provider_key, display_name, provider_type, api_base_url, supports_api, is_active
        FROM git_providers
        WHERE provider_key = $1 AND is_active = true
    `

    var model gitProviderModel
    if err := r.db.GetContext(ctx, &model, query, key); err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, err
    }

    return gitprovider.NewProviderFromExisting(
        model.ID,
        model.ProviderKey,
        model.DisplayName,
        gitprovider.ProviderType(model.ProviderType),
        model.APIBaseURL.String,
        model.SupportsAPI,
        model.IsActive,
    ), nil
}
