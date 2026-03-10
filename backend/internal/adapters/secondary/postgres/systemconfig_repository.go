package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/systemconfig"
)

// SystemConfigRepository implements the systemconfig.Repository interface
type SystemConfigRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewSystemConfigRepository creates a new system configuration repository
func NewSystemConfigRepository(db *sqlx.DB, logger *zap.Logger) *SystemConfigRepository {
	return &SystemConfigRepository{
		db:     db,
		logger: logger,
	}
}

// systemConfigModel represents the database model for system configuration
type systemConfigModel struct {
	ID          uuid.UUID       `db:"id"`
	TenantID    *uuid.UUID      `db:"tenant_id"`
	ConfigType  string          `db:"config_type"`
	ConfigKey   string          `db:"config_key"`
	ConfigValue json.RawMessage `db:"config_value"`
	Status      string          `db:"status"`
	Description string          `db:"description"`
	IsDefault   bool            `db:"is_default"`
	CreatedBy   uuid.UUID       `db:"created_by"`
	UpdatedBy   uuid.UUID       `db:"updated_by"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
	Version     int             `db:"version"`
}

// Save persists a system configuration
func (r *SystemConfigRepository) Save(ctx context.Context, config *systemconfig.SystemConfig) error {
	query := `
		INSERT INTO system_configs (
			id, tenant_id, config_type, config_key, config_value, status,
			description, is_default, created_by, updated_by, created_at, updated_at, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.ExecContext(ctx, query,
		config.ID(),
		config.TenantID(),
		string(config.ConfigType()),
		config.ConfigKey(),
		config.ConfigValue(),
		string(config.Status()),
		config.Description(),
		config.IsDefault(),
		config.CreatedBy(),
		config.UpdatedBy(),
		config.CreatedAt(),
		config.UpdatedAt(),
		config.Version(),
	)

	if err != nil {
		r.logger.Error("Failed to save system config",
			zap.String("config_id", config.ID().String()),
			zap.String("config_key", config.ConfigKey()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save system config: %w", err)
	}

	r.logger.Info("System config saved successfully",
		zap.String("config_id", config.ID().String()),
		zap.String("config_key", config.ConfigKey()),
	)

	return nil
}

// SaveAll persists multiple system configurations
func (r *SystemConfigRepository) SaveAll(ctx context.Context, configs []*systemconfig.SystemConfig) error {
	// Start a transaction for atomicity
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		r.logger.Error("Failed to start transaction for saving multiple configs", zap.Error(err))
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	// Prepare the insert query
	query := `
		INSERT INTO system_configs (
			id, tenant_id, config_type, config_key, config_value, status,
			description, is_default, created_by, updated_by, created_at, updated_at, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	// Execute the query for each config
	for _, config := range configs {
		_, err := tx.ExecContext(ctx, query,
			config.ID(),
			config.TenantID(),
			string(config.ConfigType()),
			config.ConfigKey(),
			config.ConfigValue(),
			string(config.Status()),
			config.Description(),
			config.IsDefault(),
			config.CreatedBy(),
			config.UpdatedBy(),
			config.CreatedAt(),
			config.UpdatedAt(),
			config.Version(),
		)

		if err != nil {
			_ = tx.Rollback()
			r.logger.Error("Failed to save system config in batch",
				zap.String("config_id", config.ID().String()),
				zap.String("config_key", config.ConfigKey()),
				zap.Error(err),
			)
			return fmt.Errorf("failed to save system config: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		r.logger.Error("Failed to commit transaction for saving multiple configs", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("System configs saved successfully in batch", zap.Int("count", len(configs)))

	return nil
}

// FindByID retrieves a configuration by ID
func (r *SystemConfigRepository) FindByID(ctx context.Context, id uuid.UUID) (*systemconfig.SystemConfig, error) {
	var model systemConfigModel

	query := `
		SELECT id, tenant_id, config_type, config_key, config_value, status,
			   description, is_default, created_by, updated_by, created_at, updated_at, version
		FROM system_configs
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &model, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, systemconfig.ErrConfigNotFound
		}
		r.logger.Error("Failed to find system config by ID",
			zap.String("config_id", id.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find system config by ID: %w", err)
	}

	return r.modelToSystemConfig(&model)
}

// FindByKey retrieves a configuration by tenant and key
func (r *SystemConfigRepository) FindByKey(ctx context.Context, tenantID *uuid.UUID, configKey string) (*systemconfig.SystemConfig, error) {
	var model systemConfigModel

	var query string
	var args []interface{}

	if tenantID == nil {
		// For universal configs, find globally (tenant_id IS NULL)
		query = `
			SELECT id, tenant_id, config_type, config_key, config_value, status,
				   description, is_default, created_by, updated_by, created_at, updated_at, version
			FROM system_configs
			WHERE tenant_id IS NULL AND config_key = $1
		`
		args = []interface{}{configKey}
	} else {
		// For tenant-specific configs, find within tenant
		query = `
			SELECT id, tenant_id, config_type, config_key, config_value, status,
				   description, is_default, created_by, updated_by, created_at, updated_at, version
			FROM system_configs
			WHERE tenant_id = $1 AND config_key = $2
		`
		args = []interface{}{*tenantID, configKey}
	}

	err := r.db.GetContext(ctx, &model, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			if tenantID != nil {
				r.logger.Debug("System config not found by key",
					zap.String("tenant_id", tenantID.String()),
					zap.String("config_key", configKey),
				)
			} else {
				r.logger.Debug("Universal system config not found by key",
					zap.String("config_key", configKey),
				)
			}
			return nil, systemconfig.ErrConfigNotFound
		}
		if tenantID != nil {
			r.logger.Error("Failed to query system config by key",
				zap.String("tenant_id", tenantID.String()),
				zap.String("config_key", configKey),
				zap.Error(err),
			)
		} else {
			r.logger.Error("Failed to query universal system config by key",
				zap.String("config_key", configKey),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to find system config by key: %w", err)
	}

	return r.modelToSystemConfig(&model)
}

// FindByTypeAndKey retrieves a configuration by tenant, type, and key
func (r *SystemConfigRepository) FindByTypeAndKey(ctx context.Context, tenantID *uuid.UUID, configType systemconfig.ConfigType, configKey string) (*systemconfig.SystemConfig, error) {
	var model systemConfigModel

	var query string
	var args []interface{}

	if tenantID == nil {
		// For universal configs, find globally (tenant_id IS NULL)
		query = `
			SELECT id, tenant_id, config_type, config_key, config_value, status,
				   description, is_default, created_by, updated_by, created_at, updated_at, version
			FROM system_configs
			WHERE tenant_id IS NULL AND config_type = $1 AND config_key = $2
		`
		args = []interface{}{string(configType), configKey}
	} else {
		// For tenant-specific configs, find within tenant
		query = `
			SELECT id, tenant_id, config_type, config_key, config_value, status,
				   description, is_default, created_by, updated_by, created_at, updated_at, version
			FROM system_configs
			WHERE tenant_id = $1 AND config_type = $2 AND config_key = $3
		`
		args = []interface{}{*tenantID, string(configType), configKey}
	}

	err := r.db.GetContext(ctx, &model, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			if tenantID != nil {
				r.logger.Debug("System config not found by type and key",
					zap.String("tenant_id", tenantID.String()),
					zap.String("config_type", string(configType)),
					zap.String("config_key", configKey),
				)
			} else {
				r.logger.Debug("Universal system config not found by type and key",
					zap.String("config_type", string(configType)),
					zap.String("config_key", configKey),
				)
			}
			return nil, systemconfig.ErrConfigNotFound
		}
		if tenantID != nil {
			r.logger.Error("Failed to query system config by type and key",
				zap.String("tenant_id", tenantID.String()),
				zap.String("config_type", string(configType)),
				zap.String("config_key", configKey),
				zap.Error(err),
			)
		} else {
			r.logger.Error("Failed to query universal system config by type and key",
				zap.String("config_type", string(configType)),
				zap.String("config_key", configKey),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to find system config by type and key: %w", err)
	}

	return r.modelToSystemConfig(&model)
}

// FindByType retrieves all configurations of a specific type for a tenant
func (r *SystemConfigRepository) FindByType(ctx context.Context, tenantID *uuid.UUID, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	var models []systemConfigModel

	var query string
	var args []interface{}

	if tenantID == nil {
		// For universal configs, find globally (tenant_id IS NULL)
		query = `
			SELECT id, tenant_id, config_type, config_key, config_value, status,
				   description, is_default, created_by, updated_by, created_at, updated_at, version
			FROM system_configs
			WHERE tenant_id IS NULL AND config_type = $1
			ORDER BY config_key
		`
		args = []interface{}{string(configType)}
	} else {
		// For tenant-specific configs, find within tenant
		query = `
			SELECT id, tenant_id, config_type, config_key, config_value, status,
				   description, is_default, created_by, updated_by, created_at, updated_at, version
			FROM system_configs
			WHERE tenant_id = $1 AND config_type = $2
			ORDER BY config_key
		`
		args = []interface{}{*tenantID, string(configType)}
	}

	err := r.db.SelectContext(ctx, &models, query, args...)
	if err != nil {
		if tenantID != nil {
			r.logger.Error("Failed to find system configs by type",
				zap.String("tenant_id", tenantID.String()),
				zap.String("config_type", string(configType)),
				zap.Error(err),
			)
		} else {
			r.logger.Error("Failed to find universal system configs by type",
				zap.String("config_type", string(configType)),
				zap.Error(err))
		}
		return nil, fmt.Errorf("failed to find system configs by type: %w", err)
	}

	configs := make([]*systemconfig.SystemConfig, len(models))
	for i, model := range models {
		config, err := r.modelToSystemConfig(&model)
		if err != nil {
			return nil, err
		}
		configs[i] = config
	}

	return configs, nil
}

// FindAllByType retrieves all configurations of a specific type from all tenants
func (r *SystemConfigRepository) FindAllByType(ctx context.Context, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	var models []systemConfigModel

	query := `
		SELECT id, tenant_id, config_type, config_key, config_value, status,
			   description, is_default, created_by, updated_by, created_at, updated_at, version
		FROM system_configs
		WHERE config_type = $1
		ORDER BY tenant_id, config_key
	`

	err := r.db.SelectContext(ctx, &models, query, string(configType))
	if err != nil {
		r.logger.Error("Failed to find all system configs by type",
			zap.String("config_type", string(configType)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find all system configs by type: %w", err)
	}

	configs := make([]*systemconfig.SystemConfig, len(models))
	for i, model := range models {
		config, err := r.modelToSystemConfig(&model)
		if err != nil {
			return nil, err
		}
		configs[i] = config
	}

	return configs, nil
}

// FindUniversalByType retrieves all universal configurations of a specific type (tenant_id IS NULL)
func (r *SystemConfigRepository) FindUniversalByType(ctx context.Context, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	var models []systemConfigModel

	query := `
		SELECT id, tenant_id, config_type, config_key, config_value, status,
			   description, is_default, created_by, updated_by, created_at, updated_at, version
		FROM system_configs
		WHERE config_type = $1 AND tenant_id IS NULL
		ORDER BY config_key
	`

	err := r.db.SelectContext(ctx, &models, query, string(configType))
	if err != nil {
		r.logger.Error("Failed to find universal system configs by type",
			zap.String("config_type", string(configType)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find universal system configs by type: %w", err)
	}

	configs := make([]*systemconfig.SystemConfig, len(models))
	for i, model := range models {
		config, err := r.modelToSystemConfig(&model)
		if err != nil {
			return nil, err
		}
		configs[i] = config
	}

	return configs, nil
}

// FindByTenantID retrieves all configurations for a tenant
func (r *SystemConfigRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*systemconfig.SystemConfig, error) {
	var models []systemConfigModel

	query := `
		SELECT id, tenant_id, config_type, config_key, config_value, status,
			   description, is_default, created_by, updated_by, created_at, updated_at, version
		FROM system_configs
		WHERE tenant_id = $1
		ORDER BY config_type, config_key
	`

	err := r.db.SelectContext(ctx, &models, query, tenantID)
	if err != nil {
		r.logger.Error("Failed to find system configs by tenant ID",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find system configs by tenant ID: %w", err)
	}

	configs := make([]*systemconfig.SystemConfig, len(models))
	for i, model := range models {
		config, err := r.modelToSystemConfig(&model)
		if err != nil {
			return nil, err
		}
		configs[i] = config
	}

	return configs, nil
}

// FindAll retrieves all configurations from all tenants
func (r *SystemConfigRepository) FindAll(ctx context.Context) ([]*systemconfig.SystemConfig, error) {
	var models []systemConfigModel

	query := `
		SELECT id, tenant_id, config_type, config_key, config_value, status,
			   description, is_default, created_by, updated_by, created_at, updated_at, version
		FROM system_configs
		ORDER BY tenant_id, config_type, config_key
	`

	err := r.db.SelectContext(ctx, &models, query)
	if err != nil {
		r.logger.Error("Failed to find all system configs",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find all system configs: %w", err)
	}

	configs := make([]*systemconfig.SystemConfig, len(models))
	for i, model := range models {
		config, err := r.modelToSystemConfig(&model)
		if err != nil {
			return nil, err
		}
		configs[i] = config
	}

	return configs, nil
}

// FindActiveByType retrieves active configurations of a specific type for a tenant
func (r *SystemConfigRepository) FindActiveByType(ctx context.Context, tenantID uuid.UUID, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	var models []systemConfigModel

	query := `
		SELECT id, tenant_id, config_type, config_key, config_value, status,
			   description, is_default, created_by, updated_by, created_at, updated_at, version
		FROM system_configs
		WHERE tenant_id = $1 AND config_type = $2 AND status = 'active'
		ORDER BY config_key
	`

	err := r.db.SelectContext(ctx, &models, query, tenantID, string(configType))
	if err != nil {
		r.logger.Error("Failed to find active system configs by type",
			zap.String("tenant_id", tenantID.String()),
			zap.String("config_type", string(configType)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to find active system configs by type: %w", err)
	}

	configs := make([]*systemconfig.SystemConfig, len(models))
	for i, model := range models {
		config, err := r.modelToSystemConfig(&model)
		if err != nil {
			return nil, err
		}
		configs[i] = config
	}

	return configs, nil
}

// Update updates an existing configuration
func (r *SystemConfigRepository) Update(ctx context.Context, config *systemconfig.SystemConfig) error {
	query := `
		UPDATE system_configs
		SET config_value = $1, status = $2, description = $3, updated_by = $4,
		    updated_at = $5, version = $6
		WHERE id = $7 AND version = $8
	`

	result, err := r.db.ExecContext(ctx, query,
		config.ConfigValue(),
		string(config.Status()),
		config.Description(),
		config.UpdatedBy(),
		config.UpdatedAt(),
		config.Version(),
		config.ID(),
		config.Version()-1, // Optimistic concurrency check
	)

	if err != nil {
		r.logger.Error("Failed to update system config",
			zap.String("config_id", config.ID().String()),
			zap.String("config_key", config.ConfigKey()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update system config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("system config update failed: no rows affected (possible concurrent modification)")
	}

	r.logger.Info("System config updated successfully",
		zap.String("config_id", config.ID().String()),
		zap.String("config_key", config.ConfigKey()),
	)

	return nil
}

// Delete removes a configuration
func (r *SystemConfigRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM system_configs WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete system config",
			zap.String("config_id", id.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete system config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return systemconfig.ErrConfigNotFound
	}

	r.logger.Info("System config deleted successfully", zap.String("config_id", id.String()))
	return nil
}

// ExistsByKey checks if a configuration exists by tenant and key
func (r *SystemConfigRepository) ExistsByKey(ctx context.Context, tenantID *uuid.UUID, configKey string) (bool, error) {
	var count int

	var query string
	var args []interface{}

	if tenantID == nil {
		// For universal configs, check globally (tenant_id IS NULL)
		query = `SELECT COUNT(*) FROM system_configs WHERE tenant_id IS NULL AND config_key = $1`
		args = []interface{}{configKey}
	} else {
		// For tenant-specific configs, check within tenant
		query = `SELECT COUNT(*) FROM system_configs WHERE tenant_id = $1 AND config_key = $2`
		args = []interface{}{*tenantID, configKey}
	}

	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		if tenantID != nil {
			r.logger.Error("Failed to check if system config exists by key",
				zap.String("tenant_id", tenantID.String()),
				zap.String("config_key", configKey),
				zap.Error(err),
			)
		} else {
			r.logger.Error("Failed to check if universal system config exists by key",
				zap.String("config_key", configKey),
				zap.Error(err))
		}
		return false, fmt.Errorf("failed to check if system config exists by key: %w", err)
	}

	return count > 0, nil
}

// CountByTenantID counts configurations for a tenant
func (r *SystemConfigRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int

	query := `SELECT COUNT(*) FROM system_configs WHERE tenant_id = $1`

	err := r.db.GetContext(ctx, &count, query, tenantID)
	if err != nil {
		r.logger.Error("Failed to count system configs by tenant ID",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to count system configs by tenant ID: %w", err)
	}

	return count, nil
}

// CountByType counts configurations of a specific type for a tenant
func (r *SystemConfigRepository) CountByType(ctx context.Context, tenantID uuid.UUID, configType systemconfig.ConfigType) (int, error) {
	var count int

	query := `SELECT COUNT(*) FROM system_configs WHERE tenant_id = $1 AND config_type = $2`

	err := r.db.GetContext(ctx, &count, query, tenantID, string(configType))
	if err != nil {
		r.logger.Error("Failed to count system configs by type",
			zap.String("tenant_id", tenantID.String()),
			zap.String("config_type", string(configType)),
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to count system configs by type: %w", err)
	}

	return count, nil
}

// modelToSystemConfig converts a database model to a domain system config
func (r *SystemConfigRepository) modelToSystemConfig(model *systemConfigModel) (*systemconfig.SystemConfig, error) {
	configType := systemconfig.ConfigType(model.ConfigType)
	status := systemconfig.ConfigStatus(model.Status)

	return systemconfig.NewSystemConfigFromExisting(
		model.ID,
		model.TenantID,
		configType,
		model.ConfigKey,
		model.ConfigValue,
		status,
		model.Description,
		model.IsDefault,
		model.CreatedBy,
		model.UpdatedBy,
		model.CreatedAt,
		model.UpdatedAt,
		model.Version,
	)
}
