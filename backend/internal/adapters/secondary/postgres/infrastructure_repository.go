package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/infrastructure"
)

// InfrastructureRepository implements the infrastructure.Repository interface
type InfrastructureRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewInfrastructureRepository creates a new infrastructure repository
func NewInfrastructureRepository(db *sqlx.DB, logger *zap.Logger) *InfrastructureRepository {
	return &InfrastructureRepository{
		db:     db,
		logger: logger,
	}
}

// providerModel represents the database model for infrastructure provider
type providerModel struct {
	ID                      uuid.UUID  `db:"id"`
	TenantID                uuid.UUID  `db:"tenant_id"`
	IsGlobal                bool       `db:"is_global"`
	ProviderType            string     `db:"provider_type"`
	Name                    string     `db:"name"`
	DisplayName             string     `db:"display_name"`
	Config                  []byte     `db:"config"` // JSONB stored as []byte
	Status                  string     `db:"status"`
	Capabilities            []byte     `db:"capabilities"` // JSONB stored as []byte
	CreatedBy               uuid.UUID  `db:"created_by"`
	CreatedAt               time.Time  `db:"created_at"`
	UpdatedAt               time.Time  `db:"updated_at"`
	LastHealthCheck         *time.Time `db:"last_health_check"`
	HealthStatus            *string    `db:"health_status"`
	ReadinessStatus         *string    `db:"readiness_status"`
	ReadinessLastChecked    *time.Time `db:"readiness_last_checked"`
	ReadinessMissingPrereqs []byte     `db:"readiness_missing_prereqs"`
	BootstrapMode           string     `db:"bootstrap_mode"`
	CredentialScope         string     `db:"credential_scope"`
	TargetNamespace         *string    `db:"target_namespace"`
	IsSchedulable           bool       `db:"is_schedulable"`
	SchedulableReason       *string    `db:"schedulable_reason"`
	BlockedBy               []byte     `db:"blocked_by"`
}

// toDomain converts a provider model to domain object
func (m *providerModel) toDomain() (*infrastructure.Provider, error) {
	var config map[string]interface{}
	if len(m.Config) > 0 {
		if err := json.Unmarshal(m.Config, &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	var capabilities []string
	if len(m.Capabilities) > 0 {
		if err := json.Unmarshal(m.Capabilities, &capabilities); err != nil {
			return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
		}
	}

	var readinessMissingPrereqs []string
	if len(m.ReadinessMissingPrereqs) > 0 {
		if err := json.Unmarshal(m.ReadinessMissingPrereqs, &readinessMissingPrereqs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal readiness_missing_prereqs: %w", err)
		}
	}
	var blockedBy []string
	if len(m.BlockedBy) > 0 {
		if err := json.Unmarshal(m.BlockedBy, &blockedBy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal blocked_by: %w", err)
		}
	}

	return &infrastructure.Provider{
		ID:                      m.ID,
		TenantID:                m.TenantID,
		IsGlobal:                m.IsGlobal,
		ProviderType:            infrastructure.ProviderType(m.ProviderType),
		Name:                    m.Name,
		DisplayName:             m.DisplayName,
		Config:                  config,
		Status:                  infrastructure.ProviderStatus(m.Status),
		Capabilities:            capabilities,
		CreatedBy:               m.CreatedBy,
		CreatedAt:               m.CreatedAt,
		UpdatedAt:               m.UpdatedAt,
		LastHealthCheck:         m.LastHealthCheck,
		HealthStatus:            m.HealthStatus,
		ReadinessStatus:         m.ReadinessStatus,
		ReadinessLastChecked:    m.ReadinessLastChecked,
		ReadinessMissingPrereqs: readinessMissingPrereqs,
		BootstrapMode:           m.BootstrapMode,
		CredentialScope:         m.CredentialScope,
		TargetNamespace:         m.TargetNamespace,
		IsSchedulable:           m.IsSchedulable,
		SchedulableReason:       m.SchedulableReason,
		BlockedBy:               blockedBy,
	}, nil
}

// fromDomain converts a domain object to provider model
func providerModelFromDomain(p *infrastructure.Provider) (*providerModel, error) {
	configBytes, err := json.Marshal(p.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	capabilitiesBytes, err := json.Marshal(p.Capabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal capabilities: %w", err)
	}

	readinessMissingPrereqsBytes, err := json.Marshal(p.ReadinessMissingPrereqs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal readiness missing prereqs: %w", err)
	}
	blockedByBytes, err := json.Marshal(p.BlockedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal blocked_by: %w", err)
	}

	bootstrapMode := p.BootstrapMode
	if bootstrapMode == "" {
		bootstrapMode = "image_factory_managed"
	}
	credentialScope := p.CredentialScope
	if credentialScope == "" {
		credentialScope = "unknown"
	}

	return &providerModel{
		ID:                      p.ID,
		TenantID:                p.TenantID,
		IsGlobal:                p.IsGlobal,
		ProviderType:            string(p.ProviderType),
		Name:                    p.Name,
		DisplayName:             p.DisplayName,
		Config:                  configBytes,
		Status:                  string(p.Status),
		Capabilities:            capabilitiesBytes,
		CreatedBy:               p.CreatedBy,
		CreatedAt:               p.CreatedAt,
		UpdatedAt:               p.UpdatedAt,
		LastHealthCheck:         p.LastHealthCheck,
		HealthStatus:            p.HealthStatus,
		ReadinessStatus:         p.ReadinessStatus,
		ReadinessLastChecked:    p.ReadinessLastChecked,
		ReadinessMissingPrereqs: readinessMissingPrereqsBytes,
		BootstrapMode:           bootstrapMode,
		CredentialScope:         credentialScope,
		TargetNamespace:         p.TargetNamespace,
		IsSchedulable:           p.IsSchedulable,
		SchedulableReason:       p.SchedulableReason,
		BlockedBy:               blockedByBytes,
	}, nil
}

// SaveProvider persists an infrastructure provider
func (r *InfrastructureRepository) SaveProvider(ctx context.Context, provider *infrastructure.Provider) error {
	model, err := providerModelFromDomain(provider)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO infrastructure_providers (
			id, tenant_id, is_global, provider_type, name, display_name, config,
			status, capabilities, bootstrap_mode, credential_scope, target_namespace,
			is_schedulable, schedulable_reason, blocked_by, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	_, err = r.db.ExecContext(ctx, query,
		model.ID, model.TenantID, model.IsGlobal, model.ProviderType, model.Name,
		model.DisplayName, model.Config, model.Status, model.Capabilities,
		model.BootstrapMode, model.CredentialScope, model.TargetNamespace,
		model.IsSchedulable, model.SchedulableReason, model.BlockedBy, model.CreatedBy, model.CreatedAt, model.UpdatedAt)

	if err != nil {
		r.logger.Error("Failed to save infrastructure provider",
			zap.String("provider_id", provider.ID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to save provider: %w", err)
	}

	return nil
}

// FindProviderByID retrieves a provider by ID
func (r *InfrastructureRepository) FindProviderByID(ctx context.Context, id uuid.UUID) (*infrastructure.Provider, error) {
	query := `
		SELECT id, tenant_id, is_global, provider_type, name, display_name, config,
		       status, capabilities, created_by, created_at, updated_at,
		       last_health_check, health_status, readiness_status, readiness_last_checked, readiness_missing_prereqs,
		       bootstrap_mode, credential_scope, target_namespace, is_schedulable, schedulable_reason, blocked_by
		FROM infrastructure_providers
		WHERE id = $1
	`

	var model providerModel
	err := r.db.GetContext(ctx, &model, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find provider: %w", err)
	}

	return model.toDomain()
}

// FindProvidersByTenant retrieves providers for a tenant with optional filtering
func (r *InfrastructureRepository) FindProvidersByTenant(ctx context.Context, tenantID uuid.UUID, opts *infrastructure.ListProvidersOptions) (*infrastructure.ListProvidersResult, error) {
	query := `
		SELECT id, tenant_id, is_global, provider_type, name, display_name, config,
		       status, capabilities, created_by, created_at, updated_at,
		       last_health_check, health_status, readiness_status, readiness_last_checked, readiness_missing_prereqs,
		       bootstrap_mode, credential_scope, target_namespace, is_schedulable, schedulable_reason, blocked_by
		FROM infrastructure_providers
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argCount := 1

	// Add filters
	if opts != nil {
		if opts.ProviderType != nil {
			argCount++
			query += fmt.Sprintf(" AND provider_type = $%d", argCount)
			args = append(args, string(*opts.ProviderType))
		}
		if opts.Status != nil {
			argCount++
			query += fmt.Sprintf(" AND status = $%d", argCount)
			args = append(args, string(*opts.Status))
		}
	}

	// Add ordering
	query += " ORDER BY created_at DESC"

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS subquery", query)
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to count providers: %w", err)
	}

	// Add pagination
	page := 1
	limit := 20
	if opts != nil && opts.Page > 0 {
		page = opts.Page
	}
	if opts != nil && opts.Limit > 0 {
		limit = opts.Limit
	}

	offset := (page - 1) * limit
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	// Execute query
	var models []providerModel
	err = r.db.SelectContext(ctx, &models, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}

	// Convert to domain objects
	providers := make([]infrastructure.Provider, len(models))
	for i, model := range models {
		provider, err := model.toDomain()
		if err != nil {
			return nil, fmt.Errorf("failed to convert provider model: %w", err)
		}
		providers[i] = *provider
	}

	totalPages := (total + limit - 1) / limit

	return &infrastructure.ListProvidersResult{
		Providers:  providers,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// FindProvidersAll retrieves providers across all tenants with optional filtering.
func (r *InfrastructureRepository) FindProvidersAll(ctx context.Context, opts *infrastructure.ListProvidersOptions) (*infrastructure.ListProvidersResult, error) {
	query := `
		SELECT id, tenant_id, is_global, provider_type, name, display_name, config,
		       status, capabilities, created_by, created_at, updated_at,
		       last_health_check, health_status, readiness_status, readiness_last_checked, readiness_missing_prereqs,
		       bootstrap_mode, credential_scope, target_namespace, is_schedulable, schedulable_reason, blocked_by
		FROM infrastructure_providers
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 0

	if opts != nil {
		if opts.ProviderType != nil {
			argCount++
			query += fmt.Sprintf(" AND provider_type = $%d", argCount)
			args = append(args, string(*opts.ProviderType))
		}
		if opts.Status != nil {
			argCount++
			query += fmt.Sprintf(" AND status = $%d", argCount)
			args = append(args, string(*opts.Status))
		}
	}

	query += " ORDER BY created_at DESC"

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS subquery", query)
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to count providers: %w", err)
	}

	page := 1
	limit := 20
	if opts != nil && opts.Page > 0 {
		page = opts.Page
	}
	if opts != nil && opts.Limit > 0 {
		limit = opts.Limit
	}

	offset := (page - 1) * limit
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	var models []providerModel
	err = r.db.SelectContext(ctx, &models, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}

	providers := make([]infrastructure.Provider, len(models))
	for i, model := range models {
		provider, err := model.toDomain()
		if err != nil {
			return nil, fmt.Errorf("failed to convert provider model: %w", err)
		}
		providers[i] = *provider
	}

	totalPages := (total + limit - 1) / limit

	return &infrastructure.ListProvidersResult{
		Providers:  providers,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// UpdateProvider updates an existing provider
func (r *InfrastructureRepository) UpdateProvider(ctx context.Context, provider *infrastructure.Provider) error {
	model, err := providerModelFromDomain(provider)
	if err != nil {
		return err
	}

	query := `
		UPDATE infrastructure_providers
		SET display_name = $1, config = $2, capabilities = $3, status = $4,
		    is_global = $5, updated_at = $6, last_health_check = $7, health_status = $8,
		    readiness_status = $9, readiness_last_checked = $10, readiness_missing_prereqs = $11,
		    bootstrap_mode = $12, credential_scope = $13, target_namespace = $14,
		    is_schedulable = $15, schedulable_reason = $16, blocked_by = $17
		WHERE id = $18
	`

	_, err = r.db.ExecContext(ctx, query,
		model.DisplayName, model.Config, model.Capabilities, model.Status,
		model.IsGlobal, model.UpdatedAt, model.LastHealthCheck, model.HealthStatus,
		model.ReadinessStatus, model.ReadinessLastChecked, model.ReadinessMissingPrereqs,
		model.BootstrapMode, model.CredentialScope, model.TargetNamespace,
		model.IsSchedulable, model.SchedulableReason, model.BlockedBy, model.ID)

	if err != nil {
		r.logger.Error("Failed to update infrastructure provider",
			zap.String("provider_id", provider.ID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to update provider: %w", err)
	}

	return nil
}

// DeleteProvider removes a provider
func (r *InfrastructureRepository) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM infrastructure_providers WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete infrastructure provider",
			zap.String("provider_id", id.String()),
			zap.Error(err))
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("provider not found")
	}

	return nil
}

// ExistsProviderByName checks if a provider exists by name for a tenant
func (r *InfrastructureRepository) ExistsProviderByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM infrastructure_providers WHERE tenant_id = $1 AND name = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, tenantID, name)
	if err != nil {
		return false, fmt.Errorf("failed to check provider existence: %w", err)
	}

	return exists, nil
}

// permissionModel represents the database model for provider permission
type permissionModel struct {
	ID         uuid.UUID  `db:"id"`
	ProviderID uuid.UUID  `db:"provider_id"`
	TenantID   uuid.UUID  `db:"tenant_id"`
	Permission string     `db:"permission"`
	GrantedBy  uuid.UUID  `db:"granted_by"`
	GrantedAt  time.Time  `db:"granted_at"`
	ExpiresAt  *time.Time `db:"expires_at"`
}

type installerJobModel struct {
	ID           uuid.UUID  `db:"id"`
	ProviderID   uuid.UUID  `db:"provider_id"`
	TenantID     uuid.UUID  `db:"tenant_id"`
	RequestedBy  uuid.UUID  `db:"requested_by"`
	Operation    string     `db:"operation"`
	InstallMode  string     `db:"install_mode"`
	AssetVersion string     `db:"asset_version"`
	Status       string     `db:"status"`
	ErrorMessage *string    `db:"error_message"`
	StartedAt    *time.Time `db:"started_at"`
	CompletedAt  *time.Time `db:"completed_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

type installerJobEventModel struct {
	ID         uuid.UUID  `db:"id"`
	JobID      uuid.UUID  `db:"job_id"`
	ProviderID uuid.UUID  `db:"provider_id"`
	TenantID   uuid.UUID  `db:"tenant_id"`
	EventType  string     `db:"event_type"`
	Message    string     `db:"message"`
	Details    []byte     `db:"details"`
	CreatedBy  *uuid.UUID `db:"created_by"`
	CreatedAt  time.Time  `db:"created_at"`
}

type providerPrepareRunModel struct {
	ID               uuid.UUID  `db:"id"`
	ProviderID       uuid.UUID  `db:"provider_id"`
	TenantID         uuid.UUID  `db:"tenant_id"`
	RequestedBy      uuid.UUID  `db:"requested_by"`
	Status           string     `db:"status"`
	RequestedActions []byte     `db:"requested_actions"`
	ResultSummary    []byte     `db:"result_summary"`
	ErrorMessage     *string    `db:"error_message"`
	StartedAt        *time.Time `db:"started_at"`
	CompletedAt      *time.Time `db:"completed_at"`
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
}

type providerPrepareRunCheckModel struct {
	ID        uuid.UUID `db:"id"`
	RunID     uuid.UUID `db:"run_id"`
	CheckKey  string    `db:"check_key"`
	Category  string    `db:"category"`
	Severity  string    `db:"severity"`
	OK        bool      `db:"ok"`
	Message   string    `db:"message"`
	Details   []byte    `db:"details"`
	CreatedAt time.Time `db:"created_at"`
}

type providerPrepareLatestSummaryModel struct {
	ProviderID    uuid.UUID  `db:"provider_id"`
	RunID         *uuid.UUID `db:"run_id"`
	Status        *string    `db:"status"`
	UpdatedAt     *time.Time `db:"updated_at"`
	ErrorMessage  *string    `db:"error_message"`
	CheckCategory *string    `db:"check_category"`
	CheckSeverity *string    `db:"check_severity"`
	CheckDetails  []byte     `db:"check_details"`
}

type tenantNamespacePrepareModel struct {
	ID                    uuid.UUID  `db:"id"`
	ProviderID            uuid.UUID  `db:"provider_id"`
	TenantID              uuid.UUID  `db:"tenant_id"`
	Namespace             string     `db:"namespace"`
	RequestedBy           *uuid.UUID `db:"requested_by"`
	Status                string     `db:"status"`
	ResultSummary         []byte     `db:"result_summary"`
	ErrorMessage          *string    `db:"error_message"`
	DesiredAssetVersion   *string    `db:"desired_asset_version"`
	InstalledAssetVersion *string    `db:"installed_asset_version"`
	AssetDriftStatus      string     `db:"asset_drift_status"`
	StartedAt             *time.Time `db:"started_at"`
	CompletedAt           *time.Time `db:"completed_at"`
	CreatedAt             time.Time  `db:"created_at"`
	UpdatedAt             time.Time  `db:"updated_at"`
}

// SavePermission persists a provider permission
func (r *InfrastructureRepository) SavePermission(ctx context.Context, permission *infrastructure.ProviderPermission) error {
	query := `
		INSERT INTO provider_permissions (
			id, provider_id, tenant_id, permission, granted_by, granted_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (provider_id, tenant_id, permission) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query,
		permission.ID, permission.ProviderID, permission.TenantID, permission.Permission,
		permission.GrantedBy, permission.GrantedAt, permission.ExpiresAt)

	if err != nil {
		r.logger.Error("Failed to save provider permission",
			zap.String("provider_id", permission.ProviderID.String()),
			zap.String("tenant_id", permission.TenantID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to save permission: %w", err)
	}

	return nil
}

// FindPermissionsByTenant retrieves permissions for a tenant
func (r *InfrastructureRepository) FindPermissionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*infrastructure.ProviderPermission, error) {
	query := `
		SELECT id, provider_id, tenant_id, permission, granted_by, granted_at, expires_at
		FROM provider_permissions
		WHERE tenant_id = $1 AND (expires_at IS NULL OR expires_at > NOW())
	`

	var models []permissionModel
	err := r.db.SelectContext(ctx, &models, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to find permissions: %w", err)
	}

	permissions := make([]*infrastructure.ProviderPermission, len(models))
	for i, model := range models {
		permissions[i] = &infrastructure.ProviderPermission{
			ID:         model.ID,
			ProviderID: model.ProviderID,
			TenantID:   model.TenantID,
			Permission: model.Permission,
			GrantedBy:  model.GrantedBy,
			GrantedAt:  model.GrantedAt,
			ExpiresAt:  model.ExpiresAt,
		}
	}

	return permissions, nil
}

// FindPermissionsByProvider retrieves permissions for a provider
func (r *InfrastructureRepository) FindPermissionsByProvider(ctx context.Context, providerID uuid.UUID) ([]*infrastructure.ProviderPermission, error) {
	query := `
		SELECT id, provider_id, tenant_id, permission, granted_by, granted_at, expires_at
		FROM provider_permissions
		WHERE provider_id = $1 AND (expires_at IS NULL OR expires_at > NOW())
	`

	var models []permissionModel
	err := r.db.SelectContext(ctx, &models, query, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find permissions: %w", err)
	}

	permissions := make([]*infrastructure.ProviderPermission, len(models))
	for i, model := range models {
		permissions[i] = &infrastructure.ProviderPermission{
			ID:         model.ID,
			ProviderID: model.ProviderID,
			TenantID:   model.TenantID,
			Permission: model.Permission,
			GrantedBy:  model.GrantedBy,
			GrantedAt:  model.GrantedAt,
			ExpiresAt:  model.ExpiresAt,
		}
	}

	return permissions, nil
}

// DeletePermission removes a permission
func (r *InfrastructureRepository) DeletePermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) error {
	query := `DELETE FROM provider_permissions WHERE provider_id = $1 AND tenant_id = $2 AND permission = $3`

	result, err := r.db.ExecContext(ctx, query, providerID, tenantID, permission)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("permission not found")
	}

	return nil
}

// HasPermission checks if a tenant has a specific permission for a provider
func (r *InfrastructureRepository) HasPermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM provider_permissions
			WHERE provider_id = $1 AND tenant_id = $2 AND permission = $3
			AND (expires_at IS NULL OR expires_at > NOW())
		)
	`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, providerID, tenantID, permission)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	return exists, nil
}

// UpdateProviderHealth updates the health status of a provider
func (r *InfrastructureRepository) UpdateProviderHealth(ctx context.Context, providerID uuid.UUID, health *infrastructure.ProviderHealth) error {
	query := `
		UPDATE infrastructure_providers
		SET last_health_check = $1, health_status = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := r.db.ExecContext(ctx, query, health.LastCheck, health.Status, time.Now(), providerID)
	if err != nil {
		return fmt.Errorf("failed to update provider health: %w", err)
	}

	return nil
}

// GetProviderHealth retrieves the health status of a provider
func (r *InfrastructureRepository) GetProviderHealth(ctx context.Context, providerID uuid.UUID) (*infrastructure.ProviderHealth, error) {
	query := `
		SELECT last_health_check, health_status
		FROM infrastructure_providers
		WHERE id = $1
	`

	var lastCheck *time.Time
	var status *string
	err := r.db.QueryRowContext(ctx, query, providerID).Scan(&lastCheck, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("provider not found")
		}
		return nil, fmt.Errorf("failed to get provider health: %w", err)
	}

	if lastCheck == nil || status == nil {
		return nil, fmt.Errorf("no health data available")
	}

	return &infrastructure.ProviderHealth{
		ProviderID: providerID,
		Status:     *status,
		LastCheck:  *lastCheck,
	}, nil
}

func (r *InfrastructureRepository) UpdateProviderReadiness(ctx context.Context, providerID uuid.UUID, status string, checkedAt time.Time, missingPrereqs []string) error {
	missingJSON, err := json.Marshal(missingPrereqs)
	if err != nil {
		return fmt.Errorf("failed to marshal readiness missing prerequisites: %w", err)
	}

	query := `
		UPDATE infrastructure_providers
		SET readiness_status = $1,
		    readiness_last_checked = $2,
		    readiness_missing_prereqs = $3,
		    updated_at = $4
		WHERE id = $5
	`
	_, err = r.db.ExecContext(ctx, query, status, checkedAt, missingJSON, time.Now(), providerID)
	if err != nil {
		return fmt.Errorf("failed to update provider readiness: %w", err)
	}
	return nil
}

func (r *InfrastructureRepository) CreateInstallerJob(ctx context.Context, job *infrastructure.TektonInstallerJob) error {
	query := `
		INSERT INTO tekton_installer_jobs (
			id, provider_id, tenant_id, requested_by, operation, install_mode, asset_version, status,
			error_message, started_at, completed_at, created_at, updated_at
		) VALUES (
			:id, :provider_id, :tenant_id, :requested_by, :operation, :install_mode, :asset_version, :status,
			:error_message, :started_at, :completed_at, :created_at, :updated_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, job)
	if err != nil {
		return fmt.Errorf("failed to create installer job: %w", err)
	}
	return nil
}

func (r *InfrastructureRepository) ClaimNextPendingInstallerJob(ctx context.Context) (*infrastructure.TektonInstallerJob, error) {
	query := `
		WITH next_job AS (
			SELECT id
			FROM tekton_installer_jobs
			WHERE status = 'pending'
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE tekton_installer_jobs AS j
		SET status = 'running',
		    started_at = COALESCE(j.started_at, NOW()),
		    updated_at = NOW()
		FROM next_job
		WHERE j.id = next_job.id
		RETURNING j.id, j.provider_id, j.tenant_id, j.requested_by, j.operation, j.install_mode, j.asset_version, j.status,
		          j.error_message, j.started_at, j.completed_at, j.created_at, j.updated_at
	`
	var model installerJobModel
	if err := r.db.GetContext(ctx, &model, query); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to claim next pending installer job: %w", err)
	}
	return &infrastructure.TektonInstallerJob{
		ID:           model.ID,
		ProviderID:   model.ProviderID,
		TenantID:     model.TenantID,
		RequestedBy:  model.RequestedBy,
		Operation:    infrastructure.TektonInstallerOperation(model.Operation),
		InstallMode:  infrastructure.TektonInstallMode(model.InstallMode),
		AssetVersion: model.AssetVersion,
		Status:       infrastructure.TektonInstallerJobStatus(model.Status),
		ErrorMessage: model.ErrorMessage,
		StartedAt:    model.StartedAt,
		CompletedAt:  model.CompletedAt,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}, nil
}

func (r *InfrastructureRepository) FindInstallerJobByProviderAndIdempotencyKey(
	ctx context.Context,
	providerID uuid.UUID,
	operation infrastructure.TektonInstallerOperation,
	idempotencyKey string,
) (*infrastructure.TektonInstallerJob, error) {
	query := `
		SELECT j.id, j.provider_id, j.tenant_id, j.requested_by, j.install_mode, j.asset_version, j.status,
		       j.error_message, j.started_at, j.completed_at, j.created_at, j.updated_at
		FROM tekton_installer_jobs j
		INNER JOIN tekton_installer_job_events e ON e.job_id = j.id
		WHERE j.provider_id = $1
		  AND e.event_type = $2
		  AND e.details->>'idempotency_key' = $3
		ORDER BY j.created_at DESC
		LIMIT 1
	`
	var model installerJobModel
	eventType := fmt.Sprintf("%s.requested", operation)
	if err := r.db.GetContext(ctx, &model, query, providerID, eventType, idempotencyKey); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find installer job by idempotency key: %w", err)
	}
	return &infrastructure.TektonInstallerJob{
		ID:           model.ID,
		ProviderID:   model.ProviderID,
		TenantID:     model.TenantID,
		RequestedBy:  model.RequestedBy,
		Operation:    infrastructure.TektonInstallerOperation(model.Operation),
		InstallMode:  infrastructure.TektonInstallMode(model.InstallMode),
		AssetVersion: model.AssetVersion,
		Status:       infrastructure.TektonInstallerJobStatus(model.Status),
		ErrorMessage: model.ErrorMessage,
		StartedAt:    model.StartedAt,
		CompletedAt:  model.CompletedAt,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}, nil
}

func (r *InfrastructureRepository) UpdateInstallerJobStatus(ctx context.Context, id uuid.UUID, status infrastructure.TektonInstallerJobStatus, startedAt, completedAt *time.Time, errorMessage *string) error {
	query := `
		UPDATE tekton_installer_jobs
		SET status = $2,
		    started_at = COALESCE($3, started_at),
		    completed_at = $4,
		    error_message = $5,
		    updated_at = $6
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, string(status), startedAt, completedAt, errorMessage, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update installer job status: %w", err)
	}
	return nil
}

func (r *InfrastructureRepository) GetInstallerJob(ctx context.Context, id uuid.UUID) (*infrastructure.TektonInstallerJob, error) {
	query := `
		SELECT id, provider_id, tenant_id, requested_by, operation, install_mode, asset_version, status,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM tekton_installer_jobs
		WHERE id = $1
	`
	var model installerJobModel
	if err := r.db.GetContext(ctx, &model, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get installer job: %w", err)
	}
	return &infrastructure.TektonInstallerJob{
		ID:           model.ID,
		ProviderID:   model.ProviderID,
		TenantID:     model.TenantID,
		RequestedBy:  model.RequestedBy,
		Operation:    infrastructure.TektonInstallerOperation(model.Operation),
		InstallMode:  infrastructure.TektonInstallMode(model.InstallMode),
		AssetVersion: model.AssetVersion,
		Status:       infrastructure.TektonInstallerJobStatus(model.Status),
		ErrorMessage: model.ErrorMessage,
		StartedAt:    model.StartedAt,
		CompletedAt:  model.CompletedAt,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}, nil
}

func (r *InfrastructureRepository) ListInstallerJobsByProvider(ctx context.Context, providerID uuid.UUID, limit, offset int) ([]*infrastructure.TektonInstallerJob, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	query := `
		SELECT id, provider_id, tenant_id, requested_by, operation, install_mode, asset_version, status,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM tekton_installer_jobs
		WHERE provider_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	var models []installerJobModel
	if err := r.db.SelectContext(ctx, &models, query, providerID, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list installer jobs: %w", err)
	}
	out := make([]*infrastructure.TektonInstallerJob, 0, len(models))
	for _, model := range models {
		out = append(out, &infrastructure.TektonInstallerJob{
			Operation:    infrastructure.TektonInstallerOperation(model.Operation),
			ID:           model.ID,
			ProviderID:   model.ProviderID,
			TenantID:     model.TenantID,
			RequestedBy:  model.RequestedBy,
			InstallMode:  infrastructure.TektonInstallMode(model.InstallMode),
			AssetVersion: model.AssetVersion,
			Status:       infrastructure.TektonInstallerJobStatus(model.Status),
			ErrorMessage: model.ErrorMessage,
			StartedAt:    model.StartedAt,
			CompletedAt:  model.CompletedAt,
			CreatedAt:    model.CreatedAt,
			UpdatedAt:    model.UpdatedAt,
		})
	}
	return out, nil
}

func (r *InfrastructureRepository) AddInstallerJobEvent(ctx context.Context, event *infrastructure.TektonInstallerJobEvent) error {
	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		return fmt.Errorf("failed to marshal installer event details: %w", err)
	}

	query := `
		INSERT INTO tekton_installer_job_events (
			id, job_id, provider_id, tenant_id, event_type, message, details, created_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`
	_, err = r.db.ExecContext(ctx, query,
		event.ID,
		event.JobID,
		event.ProviderID,
		event.TenantID,
		event.EventType,
		event.Message,
		detailsJSON,
		event.CreatedBy,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert installer job event: %w", err)
	}
	return nil
}

func (r *InfrastructureRepository) ListInstallerJobEvents(ctx context.Context, jobID uuid.UUID, limit, offset int) ([]*infrastructure.TektonInstallerJobEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	query := `
		SELECT id, job_id, provider_id, tenant_id, event_type, message, details, created_by, created_at
		FROM tekton_installer_job_events
		WHERE job_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`
	var models []installerJobEventModel
	if err := r.db.SelectContext(ctx, &models, query, jobID, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list installer job events: %w", err)
	}

	out := make([]*infrastructure.TektonInstallerJobEvent, 0, len(models))
	for _, model := range models {
		details := make(map[string]interface{})
		if len(model.Details) > 0 {
			if err := json.Unmarshal(model.Details, &details); err != nil {
				return nil, fmt.Errorf("failed to unmarshal installer event details: %w", err)
			}
		}
		out = append(out, &infrastructure.TektonInstallerJobEvent{
			ID:         model.ID,
			JobID:      model.JobID,
			ProviderID: model.ProviderID,
			TenantID:   model.TenantID,
			EventType:  model.EventType,
			Message:    model.Message,
			Details:    details,
			CreatedBy:  model.CreatedBy,
			CreatedAt:  model.CreatedAt,
		})
	}
	return out, nil
}

func (r *InfrastructureRepository) CreateProviderPrepareRun(ctx context.Context, run *infrastructure.ProviderPrepareRun) error {
	actionsJSON, err := json.Marshal(run.RequestedActions)
	if err != nil {
		return fmt.Errorf("failed to marshal provider prepare requested actions: %w", err)
	}
	summaryJSON, err := json.Marshal(run.ResultSummary)
	if err != nil {
		return fmt.Errorf("failed to marshal provider prepare result summary: %w", err)
	}
	query := `
		INSERT INTO provider_prepare_runs (
			id, provider_id, tenant_id, requested_by, status, requested_actions, result_summary,
			error_message, started_at, completed_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12
		)
	`
	_, err = r.db.ExecContext(ctx, query,
		run.ID, run.ProviderID, run.TenantID, run.RequestedBy, string(run.Status), actionsJSON, summaryJSON,
		run.ErrorMessage, run.StartedAt, run.CompletedAt, run.CreatedAt, run.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create provider prepare run: %w", err)
	}
	return nil
}

func (r *InfrastructureRepository) UpdateProviderPrepareRunStatus(ctx context.Context, id uuid.UUID, status infrastructure.ProviderPrepareRunStatus, startedAt, completedAt *time.Time, errorMessage *string, resultSummary map[string]interface{}) error {
	summaryJSON, err := json.Marshal(resultSummary)
	if err != nil {
		return fmt.Errorf("failed to marshal provider prepare result summary: %w", err)
	}
	query := `
		UPDATE provider_prepare_runs
		SET status = $2,
		    started_at = COALESCE($3, started_at),
		    completed_at = $4,
		    error_message = $5,
		    result_summary = $6,
		    updated_at = $7
		WHERE id = $1
	`
	_, err = r.db.ExecContext(ctx, query, id, string(status), startedAt, completedAt, errorMessage, summaryJSON, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update provider prepare run status: %w", err)
	}
	return nil
}

func (r *InfrastructureRepository) GetProviderPrepareRun(ctx context.Context, id uuid.UUID) (*infrastructure.ProviderPrepareRun, error) {
	query := `
		SELECT id, provider_id, tenant_id, requested_by, status, requested_actions, result_summary,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM provider_prepare_runs
		WHERE id = $1
	`
	var model providerPrepareRunModel
	if err := r.db.GetContext(ctx, &model, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get provider prepare run: %w", err)
	}
	return providerPrepareRunFromModel(&model)
}

func (r *InfrastructureRepository) FindActiveProviderPrepareRunByProvider(ctx context.Context, providerID uuid.UUID) (*infrastructure.ProviderPrepareRun, error) {
	query := `
		SELECT id, provider_id, tenant_id, requested_by, status, requested_actions, result_summary,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM provider_prepare_runs
		WHERE provider_id = $1
		  AND status IN ('pending', 'running')
		ORDER BY created_at DESC
		LIMIT 1
	`
	var model providerPrepareRunModel
	if err := r.db.GetContext(ctx, &model, query, providerID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find active provider prepare run: %w", err)
	}
	return providerPrepareRunFromModel(&model)
}

func (r *InfrastructureRepository) ListProviderPrepareRunsByProvider(ctx context.Context, providerID uuid.UUID, limit, offset int) ([]*infrastructure.ProviderPrepareRun, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	query := `
		SELECT id, provider_id, tenant_id, requested_by, status, requested_actions, result_summary,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM provider_prepare_runs
		WHERE provider_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	var models []providerPrepareRunModel
	if err := r.db.SelectContext(ctx, &models, query, providerID, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list provider prepare runs: %w", err)
	}
	out := make([]*infrastructure.ProviderPrepareRun, 0, len(models))
	for i := range models {
		run, err := providerPrepareRunFromModel(&models[i])
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, nil
}

func (r *InfrastructureRepository) AddProviderPrepareRunCheck(ctx context.Context, check *infrastructure.ProviderPrepareRunCheck) error {
	detailsJSON, err := json.Marshal(check.Details)
	if err != nil {
		return fmt.Errorf("failed to marshal provider prepare check details: %w", err)
	}
	query := `
		INSERT INTO provider_prepare_run_checks (
			id, run_id, check_key, category, severity, ok, message, details, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`
	_, err = r.db.ExecContext(ctx, query,
		check.ID, check.RunID, check.CheckKey, check.Category, check.Severity, check.OK, check.Message, detailsJSON, check.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert provider prepare run check: %w", err)
	}
	return nil
}

func (r *InfrastructureRepository) ListProviderPrepareRunChecks(ctx context.Context, runID uuid.UUID, limit, offset int) ([]*infrastructure.ProviderPrepareRunCheck, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	query := `
		SELECT id, run_id, check_key, category, severity, ok, message, details, created_at
		FROM provider_prepare_run_checks
		WHERE run_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`
	var models []providerPrepareRunCheckModel
	if err := r.db.SelectContext(ctx, &models, query, runID, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list provider prepare run checks: %w", err)
	}
	out := make([]*infrastructure.ProviderPrepareRunCheck, 0, len(models))
	for i := range models {
		details := map[string]interface{}{}
		if len(models[i].Details) > 0 {
			if err := json.Unmarshal(models[i].Details, &details); err != nil {
				return nil, fmt.Errorf("failed to unmarshal provider prepare run check details: %w", err)
			}
		}
		out = append(out, &infrastructure.ProviderPrepareRunCheck{
			ID:        models[i].ID,
			RunID:     models[i].RunID,
			CheckKey:  models[i].CheckKey,
			Category:  models[i].Category,
			Severity:  models[i].Severity,
			OK:        models[i].OK,
			Message:   models[i].Message,
			Details:   details,
			CreatedAt: models[i].CreatedAt,
		})
	}
	return out, nil
}

func (r *InfrastructureRepository) ListLatestProviderPrepareSummaries(ctx context.Context, providerIDs []uuid.UUID) (map[uuid.UUID]*infrastructure.ProviderPrepareLatestSummary, error) {
	out := make(map[uuid.UUID]*infrastructure.ProviderPrepareLatestSummary, len(providerIDs))
	if len(providerIDs) == 0 {
		return out, nil
	}

	query := `
		WITH latest_runs AS (
			SELECT DISTINCT ON (provider_id)
				provider_id, id AS run_id, status, updated_at, error_message
			FROM provider_prepare_runs
			WHERE provider_id = ANY($1::uuid[])
			ORDER BY provider_id, created_at DESC
		)
		SELECT
			lr.provider_id,
			lr.run_id,
			lr.status,
			lr.updated_at,
			lr.error_message,
			lc.category AS check_category,
			lc.severity AS check_severity,
			lc.details AS check_details
		FROM latest_runs lr
		LEFT JOIN LATERAL (
			SELECT category, severity, details
			FROM provider_prepare_run_checks
			WHERE run_id = lr.run_id
			ORDER BY created_at DESC
			LIMIT 1
		) lc ON TRUE
	`

	var models []providerPrepareLatestSummaryModel
	providerIDStrings := make([]string, 0, len(providerIDs))
	for _, providerID := range providerIDs {
		providerIDStrings = append(providerIDStrings, providerID.String())
	}
	if err := r.db.SelectContext(ctx, &models, query, pq.Array(providerIDStrings)); err != nil {
		return nil, fmt.Errorf("failed to list latest provider prepare summaries: %w", err)
	}

	for i := range models {
		model := models[i]
		summary := &infrastructure.ProviderPrepareLatestSummary{
			ProviderID:   model.ProviderID,
			RunID:        model.RunID,
			UpdatedAt:    model.UpdatedAt,
			ErrorMessage: model.ErrorMessage,
		}
		if model.Status != nil {
			status := infrastructure.ProviderPrepareRunStatus(*model.Status)
			summary.Status = &status
		}
		summary.CheckCategory = model.CheckCategory
		summary.CheckSeverity = model.CheckSeverity
		if len(model.CheckDetails) > 0 {
			var details map[string]interface{}
			if err := json.Unmarshal(model.CheckDetails, &details); err != nil {
				return nil, fmt.Errorf("failed to unmarshal latest prepare check details: %w", err)
			}
			if remediation, ok := details["remediation"].(string); ok {
				remediation = strings.TrimSpace(remediation)
				if remediation != "" {
					if len(remediation) > 180 {
						remediation = remediation[:180] + "..."
					}
					summary.RemediationHint = &remediation
				}
			}
		}
		out[model.ProviderID] = summary
	}

	for _, providerID := range providerIDs {
		if _, exists := out[providerID]; !exists {
			out[providerID] = &infrastructure.ProviderPrepareLatestSummary{
				ProviderID: providerID,
			}
		}
	}

	return out, nil
}

func (r *InfrastructureRepository) UpsertTenantNamespacePrepare(ctx context.Context, prepare *infrastructure.ProviderTenantNamespacePrepare) error {
	if prepare == nil {
		return fmt.Errorf("prepare is required")
	}
	if prepare.AssetDriftStatus == "" {
		prepare.AssetDriftStatus = infrastructure.TenantAssetDriftStatusUnknown
	}
	summaryJSON, err := json.Marshal(prepare.ResultSummary)
	if err != nil {
		return fmt.Errorf("failed to marshal tenant namespace prepare result summary: %w", err)
	}

	query := `
		INSERT INTO provider_tenant_namespace_prepares (
			id, provider_id, tenant_id, namespace, requested_by, status, result_summary,
			error_message, desired_asset_version, installed_asset_version, asset_drift_status,
			started_at, completed_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15
		)
		ON CONFLICT (provider_id, tenant_id) DO UPDATE SET
			namespace = EXCLUDED.namespace,
			requested_by = EXCLUDED.requested_by,
			status = EXCLUDED.status,
			result_summary = EXCLUDED.result_summary,
			error_message = EXCLUDED.error_message,
			desired_asset_version = EXCLUDED.desired_asset_version,
			installed_asset_version = EXCLUDED.installed_asset_version,
			asset_drift_status = EXCLUDED.asset_drift_status,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at,
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.db.ExecContext(ctx, query,
		prepare.ID, prepare.ProviderID, prepare.TenantID, prepare.Namespace, prepare.RequestedBy, string(prepare.Status), summaryJSON,
		prepare.ErrorMessage, prepare.DesiredAssetVersion, prepare.InstalledAssetVersion, string(prepare.AssetDriftStatus),
		prepare.StartedAt, prepare.CompletedAt, prepare.CreatedAt, prepare.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert tenant namespace prepare: %w", err)
	}
	return nil
}

func (r *InfrastructureRepository) GetTenantNamespacePrepare(ctx context.Context, providerID, tenantID uuid.UUID) (*infrastructure.ProviderTenantNamespacePrepare, error) {
	query := `
		SELECT id, provider_id, tenant_id, namespace, requested_by, status, result_summary,
		       error_message, desired_asset_version, installed_asset_version, asset_drift_status,
		       started_at, completed_at, created_at, updated_at
		FROM provider_tenant_namespace_prepares
		WHERE provider_id = $1 AND tenant_id = $2
	`

	var model tenantNamespacePrepareModel
	if err := r.db.GetContext(ctx, &model, query, providerID, tenantID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get tenant namespace prepare: %w", err)
	}

	summary := map[string]interface{}{}
	if len(model.ResultSummary) > 0 {
		if err := json.Unmarshal(model.ResultSummary, &summary); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tenant namespace prepare result summary: %w", err)
		}
	}
	assetDriftStatus := infrastructure.TenantAssetDriftStatus(model.AssetDriftStatus)
	if assetDriftStatus == "" {
		assetDriftStatus = infrastructure.TenantAssetDriftStatusUnknown
	}

	return &infrastructure.ProviderTenantNamespacePrepare{
		ID:                    model.ID,
		ProviderID:            model.ProviderID,
		TenantID:              model.TenantID,
		Namespace:             model.Namespace,
		RequestedBy:           model.RequestedBy,
		Status:                infrastructure.ProviderTenantNamespacePrepareStatus(model.Status),
		ResultSummary:         summary,
		ErrorMessage:          model.ErrorMessage,
		DesiredAssetVersion:   model.DesiredAssetVersion,
		InstalledAssetVersion: model.InstalledAssetVersion,
		AssetDriftStatus:      assetDriftStatus,
		StartedAt:             model.StartedAt,
		CompletedAt:           model.CompletedAt,
		CreatedAt:             model.CreatedAt,
		UpdatedAt:             model.UpdatedAt,
	}, nil
}

func (r *InfrastructureRepository) ListTenantNamespacePreparesByProvider(ctx context.Context, providerID uuid.UUID) ([]*infrastructure.ProviderTenantNamespacePrepare, error) {
	query := `
		SELECT id, provider_id, tenant_id, namespace, requested_by, status, result_summary,
		       error_message, desired_asset_version, installed_asset_version, asset_drift_status,
		       started_at, completed_at, created_at, updated_at
		FROM provider_tenant_namespace_prepares
		WHERE provider_id = $1
		ORDER BY updated_at DESC
	`

	var models []tenantNamespacePrepareModel
	if err := r.db.SelectContext(ctx, &models, query, providerID); err != nil {
		return nil, fmt.Errorf("failed to list tenant namespace prepares: %w", err)
	}

	out := make([]*infrastructure.ProviderTenantNamespacePrepare, 0, len(models))
	for i := range models {
		model := models[i]
		summary := map[string]interface{}{}
		if len(model.ResultSummary) > 0 {
			if err := json.Unmarshal(model.ResultSummary, &summary); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tenant namespace prepare result summary: %w", err)
			}
		}
		assetDriftStatus := infrastructure.TenantAssetDriftStatus(model.AssetDriftStatus)
		if assetDriftStatus == "" {
			assetDriftStatus = infrastructure.TenantAssetDriftStatusUnknown
		}
		out = append(out, &infrastructure.ProviderTenantNamespacePrepare{
			ID:                    model.ID,
			ProviderID:            model.ProviderID,
			TenantID:              model.TenantID,
			Namespace:             model.Namespace,
			RequestedBy:           model.RequestedBy,
			Status:                infrastructure.ProviderTenantNamespacePrepareStatus(model.Status),
			ResultSummary:         summary,
			ErrorMessage:          model.ErrorMessage,
			DesiredAssetVersion:   model.DesiredAssetVersion,
			InstalledAssetVersion: model.InstalledAssetVersion,
			AssetDriftStatus:      assetDriftStatus,
			StartedAt:             model.StartedAt,
			CompletedAt:           model.CompletedAt,
			CreatedAt:             model.CreatedAt,
			UpdatedAt:             model.UpdatedAt,
		})
	}

	return out, nil
}

func providerPrepareRunFromModel(model *providerPrepareRunModel) (*infrastructure.ProviderPrepareRun, error) {
	actions := map[string]interface{}{}
	if len(model.RequestedActions) > 0 {
		if err := json.Unmarshal(model.RequestedActions, &actions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal provider prepare requested actions: %w", err)
		}
	}
	resultSummary := map[string]interface{}{}
	if len(model.ResultSummary) > 0 {
		if err := json.Unmarshal(model.ResultSummary, &resultSummary); err != nil {
			return nil, fmt.Errorf("failed to unmarshal provider prepare result summary: %w", err)
		}
	}
	return &infrastructure.ProviderPrepareRun{
		ID:               model.ID,
		ProviderID:       model.ProviderID,
		TenantID:         model.TenantID,
		RequestedBy:      model.RequestedBy,
		Status:           infrastructure.ProviderPrepareRunStatus(model.Status),
		RequestedActions: actions,
		ResultSummary:    resultSummary,
		ErrorMessage:     model.ErrorMessage,
		StartedAt:        model.StartedAt,
		CompletedAt:      model.CompletedAt,
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}, nil
}

var _ infrastructure.ProviderTenantNamespacePrepareRepository = (*InfrastructureRepository)(nil)
