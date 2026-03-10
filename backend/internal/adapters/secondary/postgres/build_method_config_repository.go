package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/domain/build"
	"go.uber.org/zap"
)

// BuildMethodConfigRepository implements build.BuildMethodConfigRepository using build_configs.
type BuildMethodConfigRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewBuildMethodConfigRepository creates a new repository backed by build_configs.
func NewBuildMethodConfigRepository(db *sqlx.DB, logger *zap.Logger) build.BuildMethodConfigRepository {
	return &BuildMethodConfigRepository{
		db:     db,
		logger: logger,
	}
}

type buildConfigRow struct {
	ID             uuid.UUID `db:"id"`
	BuildID        uuid.UUID `db:"build_id"`
	BuildMethod    string    `db:"build_method"`
	Dockerfile     *string   `db:"dockerfile"`
	BuildContext   *string   `db:"build_context"`
	CacheRepo      *string   `db:"cache_repo"`
	BuildArgs      []byte    `db:"build_args"`
	Secrets        []byte    `db:"secrets"`
	Platforms      []byte    `db:"platforms"`
	CacheFrom      []byte    `db:"cache_from"`
	CacheTo        *string   `db:"cache_to"`
	Builder        *string   `db:"builder"`
	Buildpacks     []byte    `db:"buildpacks"`
	PackerTemplate *string   `db:"packer_template"`
	Metadata       []byte    `db:"metadata"`
}

func (r *BuildMethodConfigRepository) SavePacker(ctx context.Context, config *build.PackerConfig) error {
	return errors.New("SavePacker is not supported for build_configs storage")
}

func (r *BuildMethodConfigRepository) SaveBuildx(ctx context.Context, config *build.BuildxConfig) error {
	return errors.New("SaveBuildx is not supported for build_configs storage")
}

func (r *BuildMethodConfigRepository) SaveKaniko(ctx context.Context, config *build.KanikoConfig) error {
	return errors.New("SaveKaniko is not supported for build_configs storage")
}

func (r *BuildMethodConfigRepository) FindByBuildID(ctx context.Context, buildID uuid.UUID) (build.BuildMethodConfig, error) {
	query := `
		SELECT id, build_id, build_method, dockerfile, build_context, cache_repo,
		       build_args, secrets, platforms, cache_from, cache_to, builder, buildpacks, packer_template, metadata
		FROM build_configs
		WHERE build_id = $1`

	var row buildConfigRow
	if err := r.db.GetContext(ctx, &row, query, buildID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, build.ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to load build config: %w", err)
	}

	return r.configRowToMethodConfig(&row, build.BuildMethod(row.BuildMethod))
}

func (r *BuildMethodConfigRepository) FindByBuildIDAndMethod(ctx context.Context, buildID uuid.UUID, method build.BuildMethod) (build.BuildMethodConfig, error) {
	query := `
		SELECT id, build_id, build_method, dockerfile, build_context, cache_repo,
		       build_args, secrets, platforms, cache_from, cache_to, builder, buildpacks, packer_template, metadata
		FROM build_configs
		WHERE build_id = $1 AND build_method = $2`

	var row buildConfigRow
	if err := r.db.GetContext(ctx, &row, query, buildID, string(method)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, build.ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to load build config: %w", err)
	}

	return r.configRowToMethodConfig(&row, method)
}

func (r *BuildMethodConfigRepository) DeleteByBuildID(ctx context.Context, buildID uuid.UUID) error {
	return errors.New("DeleteByBuildID is not supported for build_configs storage")
}

func (r *BuildMethodConfigRepository) ListByMethod(ctx context.Context, projectID uuid.UUID, method build.BuildMethod) ([]build.BuildMethodConfig, error) {
	return nil, errors.New("ListByMethod is not supported for build_configs storage")
}

func (r *BuildMethodConfigRepository) configRowToMethodConfig(row *buildConfigRow, method build.BuildMethod) (build.BuildMethodConfig, error) {
	if row == nil {
		return nil, build.ErrConfigNotFound
	}

	buildArgs := map[string]string{}
	secrets := map[string]string{}
	platforms := []string{}
	cacheFrom := []string{}
	metadata := map[string]interface{}{}

	if len(row.BuildArgs) > 0 {
		_ = json.Unmarshal(row.BuildArgs, &buildArgs)
	}
	if len(row.Secrets) > 0 {
		_ = json.Unmarshal(row.Secrets, &secrets)
	}
	if len(row.Platforms) > 0 {
		_ = json.Unmarshal(row.Platforms, &platforms)
	}
	if len(row.CacheFrom) > 0 {
		_ = json.Unmarshal(row.CacheFrom, &cacheFrom)
	}
	if len(row.Metadata) > 0 {
		_ = json.Unmarshal(row.Metadata, &metadata)
	}
	buildpacks := []string{}
	if len(row.Buildpacks) > 0 {
		_ = json.Unmarshal(row.Buildpacks, &buildpacks)
	}

	switch method {
	case build.BuildMethodPacker:
		template := ""
		if row.PackerTemplate != nil {
			template = *row.PackerTemplate
		}
		cfg, err := build.NewPackerConfig(row.BuildID, template)
		if err != nil {
			return nil, err
		}
		if vars := getMetadataMap(metadata, "variables"); vars != nil {
			for key, val := range vars {
				_ = cfg.SetVariable(key, val)
			}
		}
		return cfg, nil

	case build.BuildMethodBuildx, build.BuildMethodDocker, build.BuildMethod("container"):
		dockerfile := ""
		buildContext := ""
		if row.Dockerfile != nil {
			dockerfile = *row.Dockerfile
		}
		if row.BuildContext != nil {
			buildContext = *row.BuildContext
		}
		cfg, err := build.NewBuildxConfig(row.BuildID, dockerfile, buildContext)
		if err != nil {
			return nil, err
		}
		for _, platform := range platforms {
			_ = cfg.AddPlatform(platform)
		}
		for key, val := range buildArgs {
			_ = cfg.SetBuildArg(key, val)
		}
		for key, val := range secrets {
			_ = cfg.SetSecret(key, val)
		}
		if len(cacheFrom) > 0 {
			cacheTo := ""
			if row.CacheTo != nil {
				cacheTo = *row.CacheTo
			}
			cfg.SetCache(cacheFrom[0], cacheTo)
		}
		return cfg, nil

	case build.BuildMethodKaniko:
		dockerfile := ""
		buildContext := ""
		if row.Dockerfile != nil {
			dockerfile = *row.Dockerfile
		}
		if row.BuildContext != nil {
			buildContext = *row.BuildContext
		}
		registryRepo := getMetadataString(metadata, "registry_repo", "registryRepo")
		if registryRepo == "" {
			return nil, fmt.Errorf("registry_repo is required for kaniko builds")
		}
		cfg, err := build.NewKanikoConfig(row.BuildID, dockerfile, buildContext, registryRepo)
		if err != nil {
			return nil, err
		}
		if row.CacheRepo != nil {
			cfg.SetCacheRepo(*row.CacheRepo)
		}
		for key, val := range buildArgs {
			_ = cfg.SetBuildArg(key, val)
		}
		cfg.SetSkipUnusedStages(getMetadataBool(metadata, "skip_unused_stages", "skipUnusedStages"))
		return cfg, nil

	case build.BuildMethodPaketo:
		builder := ""
		if row.Builder != nil {
			builder = *row.Builder
		}
		if builder == "" {
			return nil, fmt.Errorf("builder is required for paketo builds")
		}
		cfg, err := build.NewPaketoMethodConfig(row.BuildID, builder)
		if err != nil {
			return nil, err
		}
		for _, bp := range buildpacks {
			_ = cfg.AddBuildpack(bp)
		}
		if env, ok := metadata["env"].(map[string]interface{}); ok {
			for key, val := range env {
				if s, ok := val.(string); ok {
					_ = cfg.SetEnv(key, s)
				}
			}
		}
		if args, ok := metadata["build_args"].(map[string]interface{}); ok {
			for key, val := range args {
				if s, ok := val.(string); ok {
					_ = cfg.SetBuildArg(key, s)
				}
			}
		}
		return cfg, nil

	case build.BuildMethodNix:
		cfg, err := build.NewNixMethodConfig(row.BuildID)
		if err != nil {
			return nil, err
		}
		cfg.SetNixExpression(getMetadataString(metadata, "nix_expression", "nixExpression"))
		cfg.SetFlakeURI(getMetadataString(metadata, "flake_uri", "flakeURI", "flakeUri"))
		if attrs, ok := metadata["attributes"].([]interface{}); ok {
			for _, attr := range attrs {
				if s, ok := attr.(string); ok {
					_ = cfg.AddAttribute(s)
				}
			}
		}
		if outputs, ok := metadata["outputs"].(map[string]interface{}); ok {
			for key, val := range outputs {
				if s, ok := val.(string); ok {
					_ = cfg.SetOutput(key, s)
				}
			}
		}
		cfg.SetCacheDir(getMetadataString(metadata, "cache_dir", "cacheDir"))
		cfg.SetPure(getMetadataBool(metadata, "pure"))
		cfg.SetShowTrace(getMetadataBool(metadata, "show_trace", "showTrace"))
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return cfg, nil
	default:
		return nil, fmt.Errorf("unsupported build method: %s", method)
	}
}

func getMetadataString(metadata map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := metadata[key]; ok {
			if s, ok := val.(string); ok {
				return s
			}
		}
	}
	return ""
}

func getMetadataBool(metadata map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		if val, ok := metadata[key]; ok {
			if b, ok := val.(bool); ok {
				return b
			}
		}
	}
	return false
}

func getMetadataMap(metadata map[string]interface{}, key string) map[string]interface{} {
	if metadata == nil {
		return nil
	}
	raw, ok := metadata[key]
	if !ok {
		return nil
	}
	val, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	return val
}
