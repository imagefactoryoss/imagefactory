package main

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/adapters/primary/rest"
	"github.com/srikarm/image-factory/internal/adapters/secondary/email"
	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	"github.com/srikarm/image-factory/internal/application/appsignals"
	"github.com/srikarm/image-factory/internal/application/asyncsignals"
	appbootstrap "github.com/srikarm/image-factory/internal/application/bootstrap"
	buildsteps "github.com/srikarm/image-factory/internal/application/build/steps"
	buildnotifications "github.com/srikarm/image-factory/internal/application/buildnotifications"
	"github.com/srikarm/image-factory/internal/application/clustermetrics"
	imageimportnotifications "github.com/srikarm/image-factory/internal/application/imageimportnotifications"
	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	appsresmartbot "github.com/srikarm/image-factory/internal/application/sresmartbot"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/buildnotification"
	domainEmail "github.com/srikarm/image-factory/internal/domain/email"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/domain/infrastructure/connectors"
	"github.com/srikarm/image-factory/internal/domain/notification"
	"github.com/srikarm/image-factory/internal/domain/project"
	"github.com/srikarm/image-factory/internal/domain/rbac"
	"github.com/srikarm/image-factory/internal/domain/registryauth"
	"github.com/srikarm/image-factory/internal/domain/repositoryauth"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/domain/tenant"
	"github.com/srikarm/image-factory/internal/domain/user"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/config"
	"github.com/srikarm/image-factory/internal/infrastructure/crypto"
	"github.com/srikarm/image-factory/internal/infrastructure/database"
	k8sinfra "github.com/srikarm/image-factory/internal/infrastructure/kubernetes"
	"github.com/srikarm/image-factory/internal/infrastructure/ldap"
	"github.com/srikarm/image-factory/internal/infrastructure/logdetector"
	"github.com/srikarm/image-factory/internal/infrastructure/logger"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"github.com/srikarm/image-factory/internal/infrastructure/releasecompliance"
	"github.com/subosito/gotenv"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	k8srest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

var (
	buildVersion = "dev"
	buildCommit  = ""
	buildTime    = ""
	buildDirty   = ""
)

type startupBuildInfo struct {
	Version     string
	Commit      string
	BuildTime   string
	Dirty       string
	Source      string
	Fingerprint string
	CommitShort string
}

func commitMatchesExpected(actual, expected string) bool {
	actual = strings.TrimSpace(actual)
	expected = strings.TrimSpace(expected)
	if actual == "" || expected == "" || actual == "unknown" || expected == "unknown" {
		return false
	}
	if actual == expected {
		return true
	}
	if len(actual) >= len(expected) && strings.HasPrefix(actual, expected) {
		return true
	}
	if len(expected) >= len(actual) && strings.HasPrefix(expected, actual) {
		return true
	}
	return false
}

func logStartupBuildVerification(logger *zap.Logger, info startupBuildInfo) {
	expectedCommit := strings.TrimSpace(os.Getenv("IF_EXPECTED_BUILD_COMMIT"))
	if expectedCommit == "" {
		if info.Commit == "unknown" {
			logger.Warn("Build verification inconclusive: commit is unknown",
				zap.String("build_commit", info.Commit),
				zap.String("build_source", info.Source),
				zap.String("build_fingerprint", info.Fingerprint),
				zap.String("hint", "rebuild with git metadata or set IF_EXPECTED_BUILD_COMMIT to verify startup code version"),
			)
			return
		}
		logger.Info("Build verification",
			zap.String("status", "commit_detected"),
			zap.String("build_commit", info.Commit),
			zap.String("build_commit_short", info.CommitShort),
			zap.String("build_source", info.Source),
			zap.String("build_fingerprint", info.Fingerprint),
		)
		return
	}

	if commitMatchesExpected(info.Commit, expectedCommit) {
		logger.Info("Build verification",
			zap.String("status", "expected_commit_match"),
			zap.String("expected_commit", expectedCommit),
			zap.String("build_commit", info.Commit),
			zap.String("build_fingerprint", info.Fingerprint),
		)
		return
	}

	logger.Warn("Build verification mismatch",
		zap.String("status", "expected_commit_mismatch"),
		zap.String("expected_commit", expectedCommit),
		zap.String("build_commit", info.Commit),
		zap.String("build_source", info.Source),
		zap.String("build_fingerprint", info.Fingerprint),
	)
}

func resolveStartupBuildInfo() startupBuildInfo {
	info := startupBuildInfo{
		Version:   strings.TrimSpace(buildVersion),
		Commit:    strings.TrimSpace(buildCommit),
		BuildTime: strings.TrimSpace(buildTime),
		Dirty:     strings.TrimSpace(buildDirty),
		Source:    "ldflags",
	}
	if info.Version == "" {
		info.Version = "dev"
	}
	if info.Commit == "" || info.BuildTime == "" || info.Dirty == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			for _, s := range bi.Settings {
				switch s.Key {
				case "vcs.revision":
					if info.Commit == "" {
						info.Commit = strings.TrimSpace(s.Value)
					}
				case "vcs.time":
					if info.BuildTime == "" {
						info.BuildTime = strings.TrimSpace(s.Value)
					}
				case "vcs.modified":
					if info.Dirty == "" {
						info.Dirty = strings.TrimSpace(s.Value)
					}
				}
			}
			if info.Source == "ldflags" {
				info.Source = "go_build_info"
			}
		}
	}
	if info.Commit == "" {
		info.Commit = "unknown"
	}
	if info.BuildTime == "" {
		info.BuildTime = "unknown"
	}
	if info.Dirty == "" {
		info.Dirty = "unknown"
	}
	info.CommitShort = info.Commit
	if len(info.CommitShort) > 12 {
		info.CommitShort = info.CommitShort[:12]
	}
	sum := sha1.Sum([]byte(strings.Join([]string{info.Version, info.Commit, info.BuildTime, info.Dirty}, "|")))
	info.Fingerprint = fmt.Sprintf("%x", sum[:6])
	return info
}

// Helper function to get integer from environment variable
func getIntEnv(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	val := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if val == "" {
		return defaultValue
	}
	switch val {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return defaultValue
	}
}

type releaseComplianceCandidateRow struct {
	ExternalImageImportID uuid.UUID      `db:"id"`
	TenantID              uuid.UUID      `db:"tenant_id"`
	ReleaseState          sql.NullString `db:"release_state"`
	InternalImageRef      sql.NullString `db:"internal_image_ref"`
	SourceImageDigest     sql.NullString `db:"source_image_digest"`
	ReleasedAt            time.Time      `db:"released_at"`
}

func listReleaseComplianceDriftCandidates(ctx context.Context, db *sqlx.DB, limit int) ([]releasecompliance.DriftRecord, error) {
	if db == nil {
		return nil, errors.New("database is not configured")
	}
	if limit <= 0 {
		limit = 200
	}
	rows := make([]releaseComplianceCandidateRow, 0)
	query := `
		SELECT id, tenant_id, release_state, internal_image_ref, source_image_digest, released_at
		FROM external_image_imports
		WHERE request_type = 'quarantine'
		  AND released_at IS NOT NULL
		  AND COALESCE(NULLIF(TRIM(release_state), ''), 'released') <> 'released'
		ORDER BY updated_at DESC
		LIMIT $1
	`
	if err := db.SelectContext(ctx, &rows, query, limit); err != nil {
		return nil, err
	}

	out := make([]releasecompliance.DriftRecord, 0, len(rows))
	for _, row := range rows {
		if row.ExternalImageImportID == uuid.Nil || row.TenantID == uuid.Nil {
			continue
		}
		out = append(out, releasecompliance.DriftRecord{
			ExternalImageImportID: row.ExternalImageImportID,
			TenantID:              row.TenantID,
			ReleaseState:          strings.TrimSpace(row.ReleaseState.String),
			InternalImageRef:      strings.TrimSpace(row.InternalImageRef.String),
			SourceImageDigest:     strings.TrimSpace(row.SourceImageDigest.String),
			ReleasedAt:            row.ReleasedAt.UTC(),
		})
	}
	return out, nil
}

func countReleaseComplianceReleasedArtifacts(ctx context.Context, db *sqlx.DB) (int64, error) {
	if db == nil {
		return 0, errors.New("database is not configured")
	}
	const query = `
		SELECT COUNT(*)
		FROM external_image_imports
		WHERE request_type = 'quarantine'
		  AND released_at IS NOT NULL
		  AND COALESCE(NULLIF(TRIM(release_state), ''), 'released') = 'released'
	`
	var releasedCount int64
	if err := db.GetContext(ctx, &releasedCount, query); err != nil {
		return 0, err
	}
	return releasedCount, nil
}

func publishReleaseComplianceEvent(
	ctx context.Context,
	eventBus messaging.EventBus,
	eventSource string,
	schemaVersion string,
	eventType string,
	record releasecompliance.DriftRecord,
	stateField string,
	stateValue string,
) error {
	if eventBus == nil || record.ExternalImageImportID == uuid.Nil || record.TenantID == uuid.Nil {
		return nil
	}

	payload := map[string]interface{}{
		"external_image_import_id": record.ExternalImageImportID.String(),
		"tenant_id":                record.TenantID.String(),
		"source_image_digest":      strings.TrimSpace(record.SourceImageDigest),
		"internal_image_ref":       strings.TrimSpace(record.InternalImageRef),
		"released_at":              record.ReleasedAt.UTC().Format(time.RFC3339),
		"idempotency_key":          fmt.Sprintf("%s:%s:%d", record.ExternalImageImportID.String(), eventType, record.ReleasedAt.UTC().Unix()),
	}
	if stateField != "" && strings.TrimSpace(stateValue) != "" {
		payload[stateField] = strings.TrimSpace(stateValue)
	}

	return eventBus.Publish(ctx, messaging.Event{
		Type:          eventType,
		Source:        eventSource,
		SchemaVersion: schemaVersion,
		TenantID:      record.TenantID.String(),
		OccurredAt:    time.Now().UTC(),
		Payload:       payload,
	})
}

func buildKubeConfig(kubeconfigPath string) (*k8srest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	return k8srest.InClusterConfig()
}

func shouldAttemptInClusterOrKubeconfig(kubeconfigPath string) bool {
	if kubeconfigPath != "" {
		return true
	}
	// Avoid noisy warnings when running locally; only attempt in-cluster config when
	// Kubernetes service env vars are present.
	return os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("KUBERNETES_SERVICE_PORT") != ""
}

func isKubernetesCapableWorkflowProviderType(providerType infrastructure.ProviderType) bool {
	switch providerType {
	case infrastructure.ProviderTypeKubernetes,
		infrastructure.ProviderTypeAWSEKS,
		infrastructure.ProviderTypeGCPGKE,
		infrastructure.ProviderTypeAzureAKS,
		infrastructure.ProviderTypeOCIOKE,
		infrastructure.ProviderTypeVMwareVKS,
		infrastructure.ProviderTypeOpenShift,
		infrastructure.ProviderTypeRancher:
		return true
	default:
		return false
	}
}

func isWorkflowTektonEnabled(provider *infrastructure.Provider) bool {
	if provider == nil || provider.Config == nil {
		return false
	}
	enabled, ok := provider.Config["tekton_enabled"].(bool)
	return ok && enabled
}

func logDatabaseSchemaReadiness(ctx context.Context, db *sqlx.DB, expectedSchema string, skipMigrations bool, logger *zap.Logger) {
	if db == nil || logger == nil {
		return
	}

	readiness, err := database.CheckSchemaReadiness(ctx, db)
	if err != nil {
		logger.Warn("Database schema readiness check failed", zap.Error(err))
		return
	}

	expected := strings.TrimSpace(expectedSchema)
	if expected == "" {
		expected = "public"
	}
	if readiness.CurrentSchema != expected {
		logger.Fatal("Database schema mismatch detected",
			zap.String("configured_schema", expected),
			zap.String("current_schema", readiness.CurrentSchema),
			zap.String("hint", "ensure database URL/search_path and IF_DATABASE_SCHEMA are aligned"),
		)
	}

	if skipMigrations && !readiness.SchemaMigrations {
		logger.Warn("Database migrations are skipped and schema_migrations is missing in active schema",
			zap.String("current_schema", readiness.CurrentSchema),
			zap.String("hint", "run backend migrations against the active schema before starting with --skip-migrations"),
		)
	}

	logger.Info("Database schema readiness check completed",
		zap.String("current_schema", readiness.CurrentSchema),
		zap.Bool("schema_migrations_present", readiness.SchemaMigrations),
		zap.Bool("tenants_domain_column", readiness.TenantsDomainColumn),
		zap.Bool("tenants_domain_not_null", readiness.TenantsDomainNotNull),
	)
}

func newWorkflowInfrastructurePreflight(
	infraService interface {
		GetAvailableProviders(ctx context.Context, tenantID uuid.UUID) ([]*infrastructure.Provider, error)
	},
	logger *zap.Logger,
) buildsteps.InfrastructurePreflight {
	return func(ctx context.Context, tenantID uuid.UUID, manifest *build.BuildManifest) error {
		if manifest == nil {
			return errors.New("build manifest is required")
		}
		if manifest.InfrastructureType == "" || manifest.InfrastructureType == "auto" {
			return errors.New("infrastructure selection is required")
		}
		if manifest.InfrastructureType == "kubernetes" && manifest.InfrastructureProviderID == nil {
			if infraService == nil {
				return errors.New("infrastructure_provider_id is required for kubernetes builds (no global tekton-ready provider available)")
			}

			providers, err := infraService.GetAvailableProviders(ctx, tenantID)
			if err != nil {
				logger.Error("Workflow preflight failed to fetch infrastructure providers", zap.Error(err), zap.String("tenant_id", tenantID.String()))
				return errors.New("failed to select default infrastructure provider")
			}

			var selected *infrastructure.Provider
			for i := range providers {
				provider := providers[i]
				if provider == nil || !provider.IsGlobal {
					continue
				}
				if !isKubernetesCapableWorkflowProviderType(provider.ProviderType) || !isWorkflowTektonEnabled(provider) {
					continue
				}
				if _, err := connectors.BuildRESTConfigFromProviderConfig(provider.Config); err != nil {
					continue
				}
				selected = provider
				break
			}

			if selected == nil {
				return errors.New("infrastructure_provider_id is required for kubernetes builds (no global tekton-ready provider available)")
			}
			providerID := selected.ID
			manifest.InfrastructureProviderID = &providerID
		}

		if manifest.InfrastructureProviderID == nil {
			return errors.New("infrastructure_provider_id is required when infrastructure_type is set")
		}
		if infraService == nil {
			return errors.New("infrastructure service not configured")
		}

		providers, err := infraService.GetAvailableProviders(ctx, tenantID)
		if err != nil {
			logger.Error("Workflow preflight failed to validate infrastructure provider", zap.Error(err), zap.String("tenant_id", tenantID.String()))
			return errors.New("failed to validate infrastructure provider")
		}

		var selected *infrastructure.Provider
		for i := range providers {
			provider := providers[i]
			if provider != nil && provider.ID == *manifest.InfrastructureProviderID {
				selected = provider
				break
			}
		}
		if selected == nil {
			return errors.New("selected infrastructure provider is not available for this tenant")
		}

		if manifest.InfrastructureType == "kubernetes" {
			if !isKubernetesCapableWorkflowProviderType(selected.ProviderType) {
				return errors.New("selected infrastructure provider does not match infrastructure_type")
			}
			if !isWorkflowTektonEnabled(selected) {
				return errors.New("selected infrastructure provider must have tekton_enabled=true")
			}
			if _, err := connectors.BuildRESTConfigFromProviderConfig(selected.Config); err != nil {
				return errors.New("selected infrastructure provider has invalid kubernetes configuration")
			}
		} else if string(selected.ProviderType) != manifest.InfrastructureType {
			return errors.New("selected infrastructure provider does not match infrastructure_type")
		}
		return nil
	}
}

type buildMonitorSweepCandidate struct {
	InstanceID        uuid.UUID      `db:"instance_id"`
	BuildID           uuid.UUID      `db:"build_id"`
	BuildStatus       string         `db:"build_status"`
	BuildErrorMessage sql.NullString `db:"build_error_message"`
	MonitorStatus     string         `db:"monitor_status"`
}

func runBuildMonitorSweeperTick(
	ctx context.Context,
	db *sqlx.DB,
	workflowRepo domainworkflow.Repository,
	logger *zap.Logger,
	limit int,
) (attempted int, reconciled int, failed int, err error) {
	if db == nil || workflowRepo == nil {
		return 0, 0, 0, nil
	}
	if limit <= 0 {
		limit = 100
	}

	candidates := make([]buildMonitorSweepCandidate, 0, limit)
	queryErr := db.SelectContext(ctx, &candidates, `
		SELECT
			wi.id AS instance_id,
			b.id AS build_id,
			b.status AS build_status,
			b.error_message AS build_error_message,
			ws.status AS monitor_status
		FROM workflow_instances wi
		JOIN workflow_steps ws
			ON ws.instance_id = wi.id
		JOIN builds b
			ON b.id = wi.subject_id
		WHERE wi.subject_type = 'build'
			AND wi.status = 'running'
			AND ws.step_key = 'build.monitor'
			AND ws.status IN ('pending', 'running')
			AND b.status IN ('completed', 'failed', 'cancelled')
		ORDER BY wi.updated_at ASC
		LIMIT $1
	`, limit)
	if queryErr != nil {
		return 0, 0, 0, fmt.Errorf("failed to query build monitor sweep candidates: %w", queryErr)
	}

	for _, candidate := range candidates {
		attempted++
		targetStatus := domainworkflow.StepStatusFailed
		var errMsg *string
		switch candidate.BuildStatus {
		case string(build.BuildStatusCompleted):
			targetStatus = domainworkflow.StepStatusSucceeded
		default:
			msg := strings.TrimSpace(candidate.BuildErrorMessage.String)
			if msg == "" {
				msg = fmt.Sprintf("build reached terminal state: %s", candidate.BuildStatus)
			}
			errMsg = &msg
		}

		updateErr := workflowRepo.UpdateStepStatus(ctx, candidate.InstanceID, buildsteps.StepMonitorBuild, targetStatus, errMsg)
		if updateErr != nil {
			failed++
			if logger != nil {
				logger.Warn("Build monitor sweeper failed to update monitor step",
					zap.String("instance_id", candidate.InstanceID.String()),
					zap.String("build_id", candidate.BuildID.String()),
					zap.String("build_status", candidate.BuildStatus),
					zap.Error(updateErr))
			}
			continue
		}
		reconciled++
		_ = workflowRepo.AppendEvent(ctx, &domainworkflow.Event{
			ID:         uuid.New(),
			InstanceID: candidate.InstanceID,
			Type:       "workflow.step." + string(targetStatus),
			Payload: map[string]interface{}{
				"source":                  "monitor_sweeper",
				"build_id":                candidate.BuildID.String(),
				"monitor_previous_status": candidate.MonitorStatus,
				"build_status":            candidate.BuildStatus,
			},
			CreatedAt: time.Now().UTC(),
		})
	}

	return attempted, reconciled, failed, nil
}

type runtimeDependencyAlertRecipient struct {
	TenantID uuid.UUID `db:"tenant_id"`
	UserID   uuid.UUID `db:"user_id"`
}

func listRuntimeDependencyAlertRecipients(ctx context.Context, db *sqlx.DB) ([]runtimeDependencyAlertRecipient, error) {
	if db == nil {
		return nil, errors.New("database not configured")
	}
	recipients := make([]runtimeDependencyAlertRecipient, 0)
	query := `
		WITH tenant_admin_users AS (
			SELECT DISTINCT ura.tenant_id, ura.user_id
			FROM user_role_assignments ura
			INNER JOIN rbac_roles rr ON rr.id = ura.role_id
			WHERE REPLACE(LOWER(rr.name), ' ', '_') IN ('owner', 'administrator')
			UNION
			SELECT DISTINCT tg.tenant_id, gm.user_id
			FROM group_members gm
			INNER JOIN tenant_groups tg ON tg.id = gm.group_id
			WHERE tg.status = 'active'
			  AND tg.role_type IN ('owner', 'administrator')
			  AND gm.removed_at IS NULL
		)
		SELECT DISTINCT tau.tenant_id, tau.user_id
		FROM tenant_admin_users tau
		INNER JOIN tenants t ON t.id = tau.tenant_id
		INNER JOIN users u ON u.id = tau.user_id
		WHERE t.status = 'active'
		  AND u.status = 'active'
		ORDER BY tau.tenant_id, tau.user_id`
	if err := db.SelectContext(ctx, &recipients, query); err != nil {
		return nil, err
	}
	return recipients, nil
}

func emitRuntimeDependencyNotification(
	ctx context.Context,
	db *sqlx.DB,
	deliveryRepo interface {
		InsertInAppNotifications(context.Context, []buildnotifications.InAppNotificationRow) error
	},
	wsHub *rest.WebSocketHub,
	title string,
	message string,
	notificationType string,
) (int, error) {
	if deliveryRepo == nil {
		return 0, errors.New("delivery repository not configured")
	}
	recipients, err := listRuntimeDependencyAlertRecipients(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("failed to list runtime dependency alert recipients: %w", err)
	}
	if len(recipients) == 0 {
		return 0, nil
	}
	rows := make([]buildnotifications.InAppNotificationRow, 0, len(recipients))
	for _, recipient := range recipients {
		rows = append(rows, buildnotifications.InAppNotificationRow{
			ID:                  uuid.New(),
			UserID:              recipient.UserID,
			TenantID:            recipient.TenantID,
			Title:               title,
			Message:             message,
			NotificationType:    notificationType,
			RelatedResourceType: "runtime_dependency",
			RelatedResourceID:   nil,
			Channel:             string(buildnotification.ChannelInApp),
		})
	}
	if err := deliveryRepo.InsertInAppNotifications(ctx, rows); err != nil {
		return 0, fmt.Errorf("failed to insert runtime dependency notifications: %w", err)
	}
	if wsHub != nil {
		for _, row := range rows {
			notificationID := row.ID
			wsHub.BroadcastNotificationEvent(
				row.TenantID,
				row.UserID,
				"notification.created",
				&notificationID,
				map[string]interface{}{
					"notification_type":     row.NotificationType,
					"related_resource_type": row.RelatedResourceType,
				},
			)
		}
	}
	return len(rows), nil
}

func main() {
	// Parse command line flags
	var envFile = flag.String("env", "", "Path to environment file (e.g., .env.development, .env.production)")
	var migrateOnly = flag.Bool("migrate-only", false, "Run database migrations only and exit")
	var migrateDown = flag.Int("migrate-down", 0, "Rollback database migrations by specified number of steps and exit")
	var skipMigrations = flag.Bool("skip-migrations", false, "Skip running database migrations on startup")
	var reissueBootstrapAdminPassword = flag.Bool("reissue-bootstrap-admin-password", false, "Reissue first-run local admin password when setup is still required")
	flag.Parse()

	// Load environment file if specified
	if *envFile != "" {
		// Convert to absolute path if relative
		if !filepath.IsAbs(*envFile) {
			if absPath, err := filepath.Abs(*envFile); err == nil {
				*envFile = absPath
			}
		}

		// Check if file exists
		if _, err := os.Stat(*envFile); err != nil {
			log.Fatalf("Environment file not found: %s (%v)", *envFile, err)
		}

		// Load the environment file
		if err := gotenv.Load(*envFile); err != nil {
			log.Fatalf("Failed to load environment file %s: %v", *envFile, err)
		}

		log.Printf("Loaded environment from: %s", *envFile)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger, err := logger.NewLogger(cfg.Logger)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	buildInfo := resolveStartupBuildInfo()
	logger.Info("Starting Image Factory Server",
		zap.String("version", cfg.Server.Version),
		zap.String("build_version", buildInfo.Version),
		zap.String("build_commit", buildInfo.Commit),
		zap.String("build_commit_short", buildInfo.CommitShort),
		zap.String("build_time", buildInfo.BuildTime),
		zap.String("build_dirty", buildInfo.Dirty),
		zap.String("build_source", buildInfo.Source),
		zap.String("build_fingerprint", buildInfo.Fingerprint),
	)
	logStartupBuildVerification(logger, buildInfo)

	// Initialize database
	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Handle migration commands
	if *migrateDown > 0 {
		if err := database.MigrateDown(cfg.Database.URL, *migrateDown, cfg.Database.Schema); err != nil {
			logger.Fatal("Failed to rollback migrations", zap.Error(err))
		}
		logger.Info("Database migrations rolled back successfully", zap.Int("steps", *migrateDown))
		return
	}

	// Run migrations (unless skipped)
	if !*skipMigrations {
		if err := database.Migrate(cfg.Database.URL, cfg.Database.Schema); err != nil {
			logger.Fatal("Failed to run migrations", zap.Error(err))
		}
	} else {
		logger.Info("Skipping database migrations as requested")
	}

	// If migrate-only flag is set, exit after migrations
	if *migrateOnly {
		logger.Info("Database migrations completed successfully. Exiting as requested.")
		return
	}

	logDatabaseSchemaReadiness(context.Background(), db, cfg.Database.Schema, *skipMigrations, logger)

	// Initialize first-run bootstrap state and initial local admin credentials (idempotent).
	bootstrapService := appbootstrap.NewService(db, logger)
	state, generatedPassword, bootstrapErr := bootstrapService.EnsureInitialized(context.Background(), "admin@imgfactory.com")
	if bootstrapErr != nil {
		logger.Fatal("Failed to initialize bootstrap state", zap.Error(bootstrapErr))
	}
	if generatedPassword != "" {
		logger.Warn("Initial local admin credentials generated for first run",
			zap.String("email", "admin@imgfactory.com"),
			zap.String("password", generatedPassword),
		)
	}
	if state != nil {
		logger.Info("Bootstrap state loaded",
			zap.String("status", state.Status),
			zap.Bool("setup_required", state.SetupRequired),
		)
	}
	// Always rotate and print a fresh bootstrap admin password while setup is incomplete.
	// This avoids one-time password loss across restarts in containerized environments.
	if state != nil && state.SetupRequired && generatedPassword == "" {
		reissuedPassword, reissueErr := bootstrapService.ReissueInitialAdminPassword(context.Background(), "admin@imgfactory.com")
		if reissueErr != nil {
			logger.Fatal("Failed to auto-reissue bootstrap admin password", zap.Error(reissueErr))
		}
		logger.Warn("Bootstrap admin password reissued because setup is still required",
			zap.String("email", "admin@imgfactory.com"),
			zap.String("password", reissuedPassword),
		)
	}
	if *reissueBootstrapAdminPassword {
		reissuedPassword, reissueErr := bootstrapService.ReissueInitialAdminPassword(context.Background(), "admin@imgfactory.com")
		if reissueErr != nil {
			logger.Fatal("Failed to reissue bootstrap admin password", zap.Error(reissueErr))
		}
		logger.Warn("Bootstrap admin password reissued via startup flag",
			zap.String("email", "admin@imgfactory.com"),
			zap.String("password", reissuedPassword),
		)
	}

	// Initialize repositories
	tenantRepo := postgres.NewTenantRepository(db, logger)
	buildRepo := postgres.NewBuildRepository(db, logger)
	projectRepo := postgres.NewProjectRepository(db, logger)
	workflowRepo := postgres.NewWorkflowRepository(db, logger)
	userRepo := postgres.NewUserRepository(db, logger)
	userInvitationRepo := postgres.NewUserInvitationRepository(db, logger)
	rbacRepo := postgres.NewRBACRepository(db, logger)
	systemConfigRepo := postgres.NewSystemConfigRepository(db, logger)
	infrastructureRepo := postgres.NewInfrastructureRepository(db, logger)
	notificationTemplateRepo := postgres.NewNotificationRepository(db, logger)
	auditRepo := postgres.NewAuditRepository(db, logger)
	passwordResetTokenRepo := postgres.NewPasswordResetTokenRepository(db, logger)
	// Note: Image repository will be implemented as needed
	// imageRepo := postgres.NewImageRepository(db, logger)

	// Initialize services (application layer)
	systemConfigService := systemconfig.NewService(systemConfigRepo, logger)

	// Ensure a default general system config exists to avoid noisy startup errors.
	ensureDefaultGeneralConfig := func(ctx context.Context) {
		_, err := systemConfigRepo.FindByTypeAndKey(ctx, nil, systemconfig.ConfigTypeGeneral, "general")
		if err == nil {
			return
		}
		if err != systemconfig.ErrConfigNotFound {
			logger.Warn("Failed to check for general system config", zap.Error(err))
			return
		}

		defaultConfig := systemconfig.GeneralConfig{
			SystemName:           "",
			SystemDescription:    "",
			AdminEmail:           "",
			SupportEmail:         "",
			TimeZone:             "UTC",
			DateFormat:           "YYYY-MM-DD",
			DefaultLanguage:      "en",
			MessagingEnableNATS:  nil,
			ProjectRetentionDays: 30,
		}

		createdBy := uuid.Nil
		users, userErr := userRepo.FindAll(ctx)
		if userErr == nil && len(users) > 0 {
			createdBy = users[0].ID()
		}

		valueBytes, marshalErr := json.Marshal(defaultConfig)
		if marshalErr != nil {
			logger.Warn("Failed to marshal default general system config", zap.Error(marshalErr))
			return
		}

		now := time.Now().UTC()
		config, buildErr := systemconfig.NewSystemConfigFromExisting(
			uuid.New(),
			nil,
			systemconfig.ConfigTypeGeneral,
			"general",
			valueBytes,
			systemconfig.ConfigStatusActive,
			"General configuration settings",
			true,
			createdBy,
			createdBy,
			now,
			now,
			1,
		)
		if buildErr != nil {
			logger.Warn("Failed to build default general system config", zap.Error(buildErr))
			return
		}

		if saveErr := systemConfigRepo.Save(ctx, config); saveErr != nil {
			logger.Warn("Failed to seed default general system config", zap.Error(saveErr))
			return
		}

		logger.Info("Seeded default general system config", zap.Bool("user_found", createdBy != uuid.Nil))
	}

	// Ensure a default messaging system config exists so messaging lookups don't log errors on startup.
	ensureDefaultMessagingConfig := func(ctx context.Context) {
		_, err := systemConfigRepo.FindByTypeAndKey(ctx, nil, systemconfig.ConfigTypeMessaging, "messaging")
		if err == nil {
			return
		}
		if err != systemconfig.ErrConfigNotFound {
			logger.Warn("Failed to check for messaging system config", zap.Error(err))
			return
		}

		natsRequired := getBoolEnv("IF_MESSAGING_NATS_REQUIRED", false)
		outboxEnabled := getBoolEnv("IF_MESSAGING_OUTBOX_ENABLED", true)
		externalOnly := getBoolEnv("IF_MESSAGING_EXTERNAL_ONLY", false)
		outboxRelayIntervalSeconds := getIntEnv("IF_MESSAGING_OUTBOX_RELAY_INTERVAL_SECONDS", 5)
		outboxRelayBatchSize := getIntEnv("IF_MESSAGING_OUTBOX_RELAY_BATCH_SIZE", 100)
		outboxClaimLeaseSeconds := getIntEnv("IF_MESSAGING_OUTBOX_CLAIM_LEASE_SECONDS", 30)

		defaultConfig := systemconfig.MessagingConfig{
			EnableNATS:                 cfg.Messaging.EnableNATS,
			NATSRequired:               &natsRequired,
			ExternalOnly:               &externalOnly,
			OutboxEnabled:              &outboxEnabled,
			OutboxRelayIntervalSeconds: &outboxRelayIntervalSeconds,
			OutboxRelayBatchSize:       &outboxRelayBatchSize,
			OutboxClaimLeaseSeconds:    &outboxClaimLeaseSeconds,
		}

		createdBy := uuid.Nil
		users, userErr := userRepo.FindAll(ctx)
		if userErr == nil && len(users) > 0 {
			createdBy = users[0].ID()
		}

		valueBytes, marshalErr := json.Marshal(defaultConfig)
		if marshalErr != nil {
			logger.Warn("Failed to marshal default messaging system config", zap.Error(marshalErr))
			return
		}

		now := time.Now().UTC()
		config, buildErr := systemconfig.NewSystemConfigFromExisting(
			uuid.New(),
			nil,
			systemconfig.ConfigTypeMessaging,
			"messaging",
			valueBytes,
			systemconfig.ConfigStatusActive,
			"Messaging configuration settings",
			true,
			createdBy,
			createdBy,
			now,
			now,
			1,
		)
		if buildErr != nil {
			logger.Warn("Failed to build default messaging system config", zap.Error(buildErr))
			return
		}

		if saveErr := systemConfigRepo.Save(ctx, config); saveErr != nil {
			logger.Warn("Failed to seed default messaging system config", zap.Error(saveErr))
			return
		}

		logger.Info("Seeded default messaging system config", zap.Bool("user_found", createdBy != uuid.Nil))
	}

	ensureDefaultGeneralConfig(context.Background())
	ensureDefaultMessagingConfig(context.Background())

	messagingEnableNATS := cfg.Messaging.EnableNATS
	messagingNATSRequired := getBoolEnv("IF_MESSAGING_NATS_REQUIRED", false)
	messagingOutboxEnabled := getBoolEnv("IF_MESSAGING_OUTBOX_ENABLED", true)
	messagingExternalOnly := getBoolEnv("IF_MESSAGING_EXTERNAL_ONLY", false)
	outboxRelayIntervalSeconds := getIntEnv("IF_MESSAGING_OUTBOX_RELAY_INTERVAL_SECONDS", 5)
	outboxRelayBatchSize := getIntEnv("IF_MESSAGING_OUTBOX_RELAY_BATCH_SIZE", 100)
	outboxClaimLeaseSeconds := getIntEnv("IF_MESSAGING_OUTBOX_CLAIM_LEASE_SECONDS", 30)

	if config, err := systemConfigService.GetConfigByTypeAndKey(context.Background(), nil, systemconfig.ConfigTypeMessaging, "messaging"); err == nil {
		if messagingConfig, err := config.GetMessagingConfig(); err == nil {
			messagingEnableNATS = messagingConfig.EnableNATS
			if messagingConfig.NATSRequired != nil {
				messagingNATSRequired = *messagingConfig.NATSRequired
			}
			if messagingConfig.OutboxEnabled != nil {
				messagingOutboxEnabled = *messagingConfig.OutboxEnabled
			}
			if messagingConfig.ExternalOnly != nil {
				messagingExternalOnly = *messagingConfig.ExternalOnly
			}
			if messagingConfig.OutboxRelayIntervalSeconds != nil {
				outboxRelayIntervalSeconds = *messagingConfig.OutboxRelayIntervalSeconds
			}
			if messagingConfig.OutboxRelayBatchSize != nil {
				outboxRelayBatchSize = *messagingConfig.OutboxRelayBatchSize
			}
			if messagingConfig.OutboxClaimLeaseSeconds != nil {
				outboxClaimLeaseSeconds = *messagingConfig.OutboxClaimLeaseSeconds
			}
		}
	} else if config, err := systemConfigService.GetConfigByTypeAndKey(context.Background(), nil, systemconfig.ConfigTypeGeneral, "general"); err == nil {
		if general, err := config.GetGeneralConfig(); err == nil && general.MessagingEnableNATS != nil {
			messagingEnableNATS = *general.MessagingEnableNATS
		}
	}
	cfg.Messaging.EnableNATS = messagingEnableNATS

	// Allow workflow orchestrator runtime settings to be controlled from general system config.
	if generalCfg, err := systemConfigService.GetConfigByTypeAndKey(context.Background(), nil, systemconfig.ConfigTypeGeneral, "general"); err == nil && generalCfg != nil {
		var generalMap map[string]interface{}
		if unmarshalErr := json.Unmarshal(generalCfg.ConfigValue(), &generalMap); unmarshalErr != nil {
			logger.Warn("Failed to parse general config for workflow settings", zap.Error(unmarshalErr))
		} else {
			if value, ok := generalMap["workflow_enabled"].(bool); ok {
				cfg.Workflow.Enabled = value
			}
			if value, ok := generalMap["workflow_poll_interval"].(string); ok && value != "" {
				if parsed, parseErr := time.ParseDuration(value); parseErr != nil {
					logger.Warn("Invalid workflow_poll_interval in general config; keeping existing value",
						zap.String("value", value),
						zap.Error(parseErr),
					)
				} else {
					cfg.Workflow.PollInterval = parsed
				}
			}
			if value, ok := generalMap["workflow_max_steps_per_tick"].(float64); ok {
				parsed := int(value)
				if parsed > 0 {
					cfg.Workflow.MaxStepsPerTick = parsed
				}
			}
		}
	}

	// Runtime services config can explicitly override orchestrator enablement during initial setup.
	if runtimeCfg, err := systemConfigService.GetConfigByTypeAndKey(context.Background(), nil, systemconfig.ConfigTypeRuntimeServices, "runtime_services"); err == nil && runtimeCfg != nil {
		if runtimeServices, cfgErr := runtimeCfg.GetRuntimeServicesConfig(); cfgErr != nil {
			logger.Warn("Failed to parse runtime services config for workflow settings", zap.Error(cfgErr))
		} else if runtimeServices.WorkflowOrchestratorEnabled != nil {
			cfg.Workflow.Enabled = *runtimeServices.WorkflowOrchestratorEnabled
		}
	}

	// Initialize event bus + publishers
	baseBus := messaging.NewInProcessBus(logger)
	eventSource := cfg.Server.Environment
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		eventSource = fmt.Sprintf("%s:%s", eventSource, hostname)
	}

	if outboxRelayIntervalSeconds <= 0 {
		outboxRelayIntervalSeconds = 5
	}
	if outboxRelayBatchSize <= 0 {
		outboxRelayBatchSize = 100
	}
	if outboxClaimLeaseSeconds <= 0 {
		outboxClaimLeaseSeconds = 30
	}
	logger.Info("Messaging runtime settings resolved",
		zap.Bool("enable_nats", messagingEnableNATS),
		zap.Bool("nats_required", messagingNATSRequired),
		zap.Bool("external_only", messagingExternalOnly),
		zap.Bool("outbox_enabled", messagingOutboxEnabled),
		zap.Int("outbox_relay_interval_seconds", outboxRelayIntervalSeconds),
		zap.Int("outbox_relay_batch_size", outboxRelayBatchSize),
		zap.Int("outbox_claim_lease_seconds", outboxClaimLeaseSeconds),
	)

	var (
		outboxStore *messaging.SQLOutboxStore
		relay       *messaging.OutboxRelay
		natsBus     *messaging.NATSBus
	)
	if messagingExternalOnly && !messagingEnableNATS {
		logger.Fatal("IF_MESSAGING_EXTERNAL_ONLY requires messaging.enable_nats=true")
	}
	if messagingOutboxEnabled {
		outboxStore = messaging.NewSQLOutboxStore(db, logger)
	}

	var bus messaging.EventBus = baseBus
	if messagingEnableNATS {
		natsBus, err = messaging.NewNATSBus(cfg.NATS, logger)
		if err != nil {
			if messagingNATSRequired || messagingExternalOnly {
				logger.Fatal("Failed to initialize NATS event bus", zap.Error(err))
			}
			logger.Warn("NATS event bus unavailable; running in local-only mode", zap.Error(err))
		} else {
			if messagingExternalOnly {
				bus = natsBus
				logger.Info("Messaging configured in external-only mode; local in-process bus disabled")
			} else {
				hybridBus := messaging.NewHybridBusWithOutbox(baseBus, natsBus, outboxStore, eventSource, logger)
				bus = hybridBus
				defer hybridBus.Close()
				if messagingOutboxEnabled && outboxStore != nil {
					relay = messaging.NewOutboxRelay(outboxStore, natsBus, messaging.OutboxRelayConfig{
						Interval:   time.Duration(outboxRelayIntervalSeconds) * time.Second,
						BatchSize:  outboxRelayBatchSize,
						ClaimLease: time.Duration(outboxClaimLeaseSeconds) * time.Second,
					}, logger)
				}
			}
		}
	}

	eventBus := messaging.NewValidatingBus(bus, messaging.ValidationConfig{
		SchemaVersion:  cfg.Messaging.SchemaVersion,
		ValidateEvents: cfg.Messaging.ValidateEvents,
	}, logger)

	buildMonitorEventDrivenEnabled := getBoolEnv("IF_BUILD_MONITOR_EVENT_DRIVEN_ENABLED", true)
	if config, err := systemConfigService.GetConfigByTypeAndKey(context.Background(), nil, systemconfig.ConfigTypeBuild, "build"); err == nil {
		if buildConfig, cfgErr := config.GetBuildConfig(); cfgErr == nil && buildConfig.MonitorEventDrivenEnabled != nil {
			buildMonitorEventDrivenEnabled = *buildConfig.MonitorEventDrivenEnabled
		}
	}
	buildsteps.SetMonitorEventDrivenEnabledOverride(&buildMonitorEventDrivenEnabled)
	build.SetTektonMonitorEventDrivenOverride(&buildMonitorEventDrivenEnabled)
	var monitorSubscriber *buildsteps.BuildMonitorEventSubscriber
	if buildMonitorEventDrivenEnabled {
		monitorSubscriber = buildsteps.NewBuildMonitorEventSubscriber(workflowRepo, logger)
		unsubscribe := buildsteps.RegisterBuildMonitorEventSubscriber(eventBus, monitorSubscriber)
		defer unsubscribe()
		logger.Info("Build monitor event-driven diagnostics subscriber enabled",
			zap.Bool("enabled", buildMonitorEventDrivenEnabled))
	} else {
		logger.Info("Build monitor event-driven diagnostics subscriber disabled",
			zap.Bool("enabled", buildMonitorEventDrivenEnabled))
	}

	tenantEventPublisher := messaging.NewTenantEventPublisher(eventBus, eventSource, cfg.Messaging.SchemaVersion)
	buildEventPublisher := messaging.NewBuildEventPublisher(eventBus, eventSource, cfg.Messaging.SchemaVersion)
	infrastructureEventPublisher := messaging.NewInfrastructureEventPublisher(eventBus, eventSource, cfg.Messaging.SchemaVersion)
	projectEventPublisher := messaging.NewProjectEventPublisher(eventBus, eventSource, cfg.Messaging.SchemaVersion)
	infrastructureService := infrastructure.NewService(infrastructureRepo, infrastructureEventPublisher, logger)
	infrastructureService.SetTektonCoreConfigLookup(func(ctx context.Context) (*systemconfig.TektonCoreConfig, error) {
		// Global-only config (tenant_id = NULL). If missing, fall back to provider/env defaults.
		c, err := systemConfigService.GetConfigByTypeAndKey(ctx, nil, systemconfig.ConfigTypeTekton, "tekton_core")
		if err != nil {
			if errors.Is(err, systemconfig.ErrConfigNotFound) || strings.Contains(err.Error(), "no rows in result set") {
				return nil, nil
			}
			return nil, err
		}
		var out systemconfig.TektonCoreConfig
		if unmarshalErr := json.Unmarshal(c.ConfigValue(), &out); unmarshalErr != nil {
			return nil, fmt.Errorf("failed to unmarshal tekton_core system config: %w", unmarshalErr)
		}
		return &out, nil
	})
	infrastructureService.SetTektonTaskImagesConfigLookup(func(ctx context.Context) (*systemconfig.TektonTaskImagesConfig, error) {
		cfg, err := systemConfigService.GetTektonTaskImagesConfig(ctx)
		if err != nil {
			if errors.Is(err, systemconfig.ErrConfigNotFound) || strings.Contains(err.Error(), "no rows in result set") {
				return nil, nil
			}
			return nil, err
		}
		return cfg, nil
	})
	infrastructureService.SetRuntimeServicesConfigLookup(func(ctx context.Context) (*systemconfig.RuntimeServicesConfig, error) {
		c, err := systemConfigService.GetConfigByTypeAndKey(ctx, nil, systemconfig.ConfigTypeRuntimeServices, "runtime_services")
		if err != nil {
			if errors.Is(err, systemconfig.ErrConfigNotFound) || strings.Contains(err.Error(), "no rows in result set") {
				return nil, nil
			}
			return nil, err
		}
		out, cfgErr := c.GetRuntimeServicesConfig()
		if cfgErr != nil {
			return nil, cfgErr
		}
		return out, nil
	})

	// Initialize build executors
	containerExecutor := build.NewNoOpBuildExecutor(logger)
	packerExecutor := build.NewPackerBuildExecutor(logger, "/tmp/packer-work", "image-factory-builds", "us-east-1", "")

	// Validate JWT secret is set (especially important in production)
	if cfg.Auth.JWTSecret == "" {
		logger.Fatal("JWT_SECRET not configured. Set IF_AUTH_JWT_SECRET environment variable")
	}

	// Initialize WebSocket hub for real-time build events
	wsHub := rest.NewWebSocketHub(logger)
	rest.RegisterBuildEventBusSubscriber(eventBus, wsHub, logger)

	tenantService := tenant.NewService(tenantRepo, tenantEventPublisher, logger)
	projectMemberRepo := postgres.NewProjectMemberRepository(db, logger)
	projectService := project.NewService(projectRepo, projectMemberRepo, projectEventPublisher, logger)
	projectNotificationTriggerRepo := postgres.NewProjectNotificationTriggerRepository(db, logger)
	projectNotificationTriggerService := buildnotification.NewService(projectNotificationTriggerRepo)
	buildNotificationDeliveryRepo := postgres.NewBuildNotificationDeliveryRepository(db, logger)
	sreSmartBotRepo := postgres.NewSRESmartBotRepository(db, logger)
	sreEventPublisher := messaging.NewSREEventPublisher(eventBus, eventSource, cfg.Messaging.SchemaVersion)
	sreSmartBotService := appsresmartbot.NewService(sreSmartBotRepo, sreEventPublisher, logger)
	sreDetectorSuggestionService := appsresmartbot.NewDetectorRuleSuggestionService(sreSmartBotRepo, systemConfigService, logger)
	sreSmartBotService.SetDetectorSuggestionObserver(sreDetectorSuggestionService)
	sreDetectorSubscriber := appsresmartbot.NewDetectorEventSubscriber(sreSmartBotService, logger)
	sreDetectorUnsubscribe := appsresmartbot.RegisterDetectorEventSubscriber(eventBus, sreDetectorSubscriber)
	defer sreDetectorUnsubscribe()

	triggerRepo := postgres.NewTriggerRepository(db, logger)
	buildExecutionRepo := postgres.NewBuildExecutionRepository(db, logger)
	buildStatusBroadcaster := messaging.NewBuildStatusBroadcaster(eventBus, eventSource, cfg.Messaging.SchemaVersion)
	buildExecutionService := build.NewBuildExecutionServiceWithWebSocket(buildExecutionRepo, buildStatusBroadcaster)
	var (
		k8sClient             kubernetes.Interface
		clusterMetricsClient  metricsclient.Interface
		tektonClient          tektonclient.Interface
		namespaceMgr          build.NamespaceManager
		pipelineMgr           build.PipelineManager
		templateEngine        build.TemplateEngine
		buildMethodConfigRepo build.BuildMethodConfigRepository
		tektonClientProvider  build.TektonClientProvider
	)
	if infrastructureService != nil {
		tektonClientProvider = k8sinfra.NewTektonClientProvider(infrastructureService, logger)
	}
	if cfg.Build.TektonEnabled {
		templateEngine = k8sinfra.NewGoTemplateEngine()
		buildMethodConfigRepo = postgres.NewBuildMethodConfigRepository(db, logger)
		if shouldAttemptInClusterOrKubeconfig(cfg.Build.TektonKubeconfig) {
			kubeConfig, err := buildKubeConfig(cfg.Build.TektonKubeconfig)
			if err != nil {
				logger.Warn("Failed to load Kubernetes config for Tekton", zap.Error(err))
			} else {
				k8sClient, err = kubernetes.NewForConfig(kubeConfig)
				if err != nil {
					logger.Warn("Failed to initialize Kubernetes client", zap.Error(err))
				}
				clusterMetricsClient, err = metricsclient.NewForConfig(kubeConfig)
				if err != nil {
					logger.Warn("Failed to initialize Kubernetes metrics client", zap.Error(err))
				}
				tektonClient, err = tektonclient.NewForConfig(kubeConfig)
				if err != nil {
					logger.Warn("Failed to initialize Tekton client", zap.Error(err))
				}
				if k8sClient != nil && tektonClient != nil {
					namespaceMgr = k8sinfra.NewKubernetesNamespaceManager(k8sClient, logger)
					pipelineMgr = k8sinfra.NewKubernetesPipelineManager(k8sClient, tektonClient, logger)
				}
			}
		}
	}
	localExecutorFactory := build.NewBuildMethodExecutorFactory(buildExecutionService)
	var repositoryAuthService *repositoryauth.Service
	var registryAuthService *registryauth.Service
	if encryptor, encErr := crypto.NewAESGCMEncryptorFromEnv(); encErr != nil {
		logger.Warn("Credential encryptor unavailable; docker-config/git-auth reconciliation in Tekton executor will be disabled", zap.Error(encErr))
	} else {
		repositoryAuthRepo := postgres.NewRepositoryAuthRepository(db, logger)
		repositoryAuthService = repositoryauth.NewService(repositoryAuthRepo, encryptor)
		registryAuthRepo := postgres.NewRegistryAuthRepository(db, logger)
		registryAuthService = registryauth.NewService(registryAuthRepo, encryptor)
	}
	var tektonExecutorFactory build.BuildMethodExecutorFactory
	if cfg.Build.TektonEnabled && templateEngine != nil && buildMethodConfigRepo != nil && (tektonClientProvider != nil || (namespaceMgr != nil && pipelineMgr != nil && k8sClient != nil && tektonClient != nil)) {
		tektonExecutorFactory = build.NewTektonExecutorFactory(
			k8sClient,
			tektonClient,
			logger,
			namespaceMgr,
			pipelineMgr,
			templateEngine,
			buildExecutionService,
			buildMethodConfigRepo,
			buildRepo,
			tektonClientProvider,
			registryAuthService,
			repositoryAuthService,
		)
	}
	buildService := build.NewService(buildRepo, triggerRepo, buildEventPublisher, containerExecutor, packerExecutor, buildExecutionService, localExecutorFactory, tektonExecutorFactory, systemConfigService, projectService, logger)
	if registryAuthService != nil {
		buildService.SetRegistryAuthResolver(registryAuthService)
	}
	rbacService := rbac.NewService(rbacRepo, logger)
	auditService := audit.NewService(auditRepo, logger)
	audit.RegisterEventBusSubscriber(eventBus, auditService, logger)

	// Initialize build policy service
	buildPolicyRepo := postgres.NewBuildPolicyRepository(db, logger)
	buildPolicyService := build.NewBuildPolicyService(buildPolicyRepo, logger)
	dispatcherRuntimeStore := postgres.NewDispatcherRuntimeRepository(db, logger)
	processHealthStore := runtimehealth.NewStore()
	releaseComplianceMetrics := releasecompliance.NewMetrics()
	sreLogDetectorBaseURL := strings.TrimSpace(os.Getenv("IF_SRE_LOG_DETECTOR_LOKI_BASE_URL"))
	sreLogDetectorTimeout := time.Duration(getIntEnv("IF_SRE_LOG_DETECTOR_TIMEOUT_SECONDS", 15)) * time.Second
	sreLogDetectorClient := logdetector.NewLokiClient(sreLogDetectorBaseURL, &http.Client{Timeout: sreLogDetectorTimeout})
	appsresmartbot.StartNATSTransportSignalRunner(
		logger,
		processHealthStore,
		natsBus,
		sreSmartBotService,
		appsresmartbot.NATSTransportRunnerConfig{
			Enabled:            getBoolEnv("IF_NATS_TRANSPORT_SIGNAL_RUNNER_ENABLED", true),
			Interval:           time.Duration(getIntEnv("IF_NATS_TRANSPORT_SIGNAL_INTERVAL_SECONDS", 30)) * time.Second,
			ReconnectThreshold: int64(getIntEnv("IF_NATS_TRANSPORT_RECONNECT_THRESHOLD", 3)),
		},
	)
	appsresmartbot.StartLogDetectorRunner(
		logger,
		processHealthStore,
		sreLogDetectorClient,
		eventBus,
		systemConfigService,
		appsresmartbot.LogDetectorRunnerConfig{
			Enabled:       getBoolEnv("IF_SRE_LOG_DETECTOR_ENABLED", false),
			BaseURL:       sreLogDetectorBaseURL,
			Interval:      time.Duration(getIntEnv("IF_SRE_LOG_DETECTOR_INTERVAL_SECONDS", 120)) * time.Second,
			Timeout:       sreLogDetectorTimeout,
			Lookback:      time.Duration(getIntEnv("IF_SRE_LOG_DETECTOR_LOOKBACK_MINUTES", 5)) * time.Minute,
			MaxMatches:    getIntEnv("IF_SRE_LOG_DETECTOR_MAX_MATCHES", 5),
			EventSource:   eventSource,
			SchemaVersion: cfg.Messaging.SchemaVersion,
		},
	)

	if relay != nil {
		processHealthStore.Upsert("messaging_outbox_relay", runtimehealth.ProcessStatus{
			Enabled:      true,
			Running:      true,
			LastActivity: time.Now().UTC(),
			Message:      "outbox relay started",
			Metrics: map[string]int64{
				"messaging_outbox_pending_count":         0,
				"messaging_outbox_replay_success_total":  0,
				"messaging_outbox_replay_failures_total": 0,
			},
		})
		logger.Info("Background process starting",
			zap.String("component", "messaging_outbox_relay"),
			zap.Duration("interval", time.Duration(outboxRelayIntervalSeconds)*time.Second),
			zap.Bool("enabled", messagingOutboxEnabled),
		)
		go func() {
			ticker := time.NewTicker(time.Duration(outboxRelayIntervalSeconds) * time.Second)
			defer ticker.Stop()
			for {
				processed, replayErr := relay.ReplayOnce(context.Background())
				if replayErr != nil {
					snapshot := relay.Snapshot()
					pendingCount := int64(0)
					if outboxStore != nil {
						if pending, pendingErr := outboxStore.PendingCount(context.Background()); pendingErr == nil {
							pendingCount = pending
						}
					}
					processHealthStore.Upsert("messaging_outbox_relay", runtimehealth.ProcessStatus{
						Enabled:      true,
						Running:      true,
						LastActivity: time.Now().UTC(),
						Message:      fmt.Sprintf("outbox replay failed success_total=%d failure_total=%d", snapshot.ReplaySuccessTotal, snapshot.ReplayFailureTotal),
						Metrics: map[string]int64{
							"messaging_outbox_pending_count":         pendingCount,
							"messaging_outbox_replay_success_total":  snapshot.ReplaySuccessTotal,
							"messaging_outbox_replay_failures_total": snapshot.ReplayFailureTotal,
						},
					})
					logger.Warn("Messaging outbox replay failed", zap.Error(replayErr))
				} else {
					pendingCount := int64(0)
					if outboxStore != nil {
						if pending, pendingErr := outboxStore.PendingCount(context.Background()); pendingErr == nil {
							pendingCount = pending
						}
					}
					snapshot := relay.Snapshot()
					processHealthStore.Upsert("messaging_outbox_relay", runtimehealth.ProcessStatus{
						Enabled:      true,
						Running:      true,
						LastActivity: time.Now().UTC(),
						Message:      fmt.Sprintf("replay processed=%d pending=%d success_total=%d failure_total=%d", processed, pendingCount, snapshot.ReplaySuccessTotal, snapshot.ReplayFailureTotal),
						Metrics: map[string]int64{
							"messaging_outbox_pending_count":         pendingCount,
							"messaging_outbox_replay_success_total":  snapshot.ReplaySuccessTotal,
							"messaging_outbox_replay_failures_total": snapshot.ReplayFailureTotal,
						},
					})
				}
				<-ticker.C
			}
		}()
	} else {
		processHealthStore.Upsert("messaging_outbox_relay", runtimehealth.ProcessStatus{
			Enabled:      false,
			Running:      false,
			LastActivity: time.Now().UTC(),
			Message:      "outbox relay disabled",
			Metrics: map[string]int64{
				"messaging_outbox_pending_count":         0,
				"messaging_outbox_replay_success_total":  0,
				"messaging_outbox_replay_failures_total": 0,
			},
		})
	}

	if monitorSubscriber != nil {
		processHealthStore.Upsert("build_monitor_event_subscriber", runtimehealth.ProcessStatus{
			Enabled:      true,
			Running:      true,
			LastActivity: time.Now().UTC(),
			Message:      "monitor event subscriber started",
		})
		go func() {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				snapshot := monitorSubscriber.Snapshot()
				processHealthStore.Upsert("build_monitor_event_subscriber", runtimehealth.ProcessStatus{
					Enabled:      true,
					Running:      true,
					LastActivity: time.Now().UTC(),
					Message: fmt.Sprintf(
						"events=%d transitions=%d noops=%d parse_failures=%d transition_errors=%d",
						snapshot.EventsReceived,
						snapshot.Transitioned,
						snapshot.NoopTerminal,
						snapshot.ParseFailures,
						snapshot.TransitionErrors,
					),
					Metrics: map[string]int64{
						"monitor_event_driven_transitions_total": snapshot.Transitioned,
						"monitor_event_driven_noop_total":        snapshot.NoopTerminal,
						"monitor_event_driven_parse_failures":    snapshot.ParseFailures,
						"monitor_event_driven_transition_errors": snapshot.TransitionErrors,
					},
				})
				<-ticker.C
			}
		}()
	} else {
		processHealthStore.Upsert("build_monitor_event_subscriber", runtimehealth.ProcessStatus{
			Enabled:      false,
			Running:      false,
			LastActivity: time.Now().UTC(),
			Message:      "monitor event subscriber disabled",
			Metrics: map[string]int64{
				"monitor_event_driven_transitions_total": 0,
				"monitor_event_driven_noop_total":        0,
				"monitor_event_driven_parse_failures":    0,
				"monitor_event_driven_transition_errors": 0,
			},
		})
	}

	// Start dispatcher for queued builds
	var dispatcherMetricsProvider rest.DispatcherMetricsProvider
	var dispatcherController rest.DispatcherController
	var orchestratorController rest.WorkflowOrchestratorController
	embeddedDispatcherController, embeddedDispatcherMetricsProvider := appbootstrap.StartDispatcherRunner(
		appbootstrap.DispatcherRunnerDeps{
			ProcessHealthStore:  processHealthStore,
			BuildRepo:           buildRepo,
			BuildService:        buildService,
			SystemConfigService: systemConfigService,
			DispatcherRuntime:   dispatcherRuntimeStore,
			Infrastructure:      infrastructureService,
			Logger:              logger,
		},
		appbootstrap.DispatcherRunnerConfig{
			Enabled:            cfg.Dispatcher.Enabled,
			PollInterval:       cfg.Dispatcher.PollInterval,
			MaxDispatchPerTick: cfg.Dispatcher.MaxDispatchPerTick,
			MaxRetries:         cfg.Dispatcher.MaxRetries,
			RetryBackoff:       cfg.Dispatcher.RetryBackoff,
			RetryBackoffMax:    cfg.Dispatcher.RetryBackoffMax,
		},
	)
	if embeddedDispatcherController != nil {
		dispatcherController = embeddedDispatcherController
	}
	if embeddedDispatcherMetricsProvider != nil {
		dispatcherMetricsProvider = embeddedDispatcherMetricsProvider
	}

	appbootstrap.StartStaleExecutionWatchdog(
		logger,
		processHealthStore,
		db,
		eventBus,
		appbootstrap.StaleExecutionWatchdogConfig{
			Interval:      30 * time.Second,
			BatchSize:     100,
			EventSource:   eventSource,
			SchemaVersion: cfg.Messaging.SchemaVersion,
		},
	)

	buildMonitorSweeperEnabled := getBoolEnv("IF_BUILD_MONITOR_SWEEPER_ENABLED", true)
	buildMonitorSweeperIntervalSeconds := getIntEnv("IF_BUILD_MONITOR_SWEEPER_INTERVAL_SECONDS", 30)
	buildMonitorSweeperBatchSize := getIntEnv("IF_BUILD_MONITOR_SWEEPER_BATCH_SIZE", 200)
	if buildMonitorSweeperIntervalSeconds <= 0 {
		buildMonitorSweeperIntervalSeconds = 30
	}
	if buildMonitorSweeperBatchSize <= 0 {
		buildMonitorSweeperBatchSize = 200
	}
	if buildMonitorSweeperEnabled {
		var sweeperReconciledTotal atomic.Int64
		var sweeperFailureTotal atomic.Int64
		processHealthStore.Upsert("build_monitor_sweeper", runtimehealth.ProcessStatus{
			Enabled:      true,
			Running:      true,
			LastActivity: time.Now().UTC(),
			Message:      "sweeper started",
			Metrics: map[string]int64{
				"monitor_sweeper_reconciled_total": 0,
				"monitor_sweeper_failures_total":   0,
			},
		})
		logger.Info("Background process starting",
			zap.String("component", "build_monitor_sweeper"),
			zap.Duration("interval", time.Duration(buildMonitorSweeperIntervalSeconds)*time.Second),
			zap.Int("batch_limit", buildMonitorSweeperBatchSize),
		)
		go func() {
			ticker := time.NewTicker(time.Duration(buildMonitorSweeperIntervalSeconds) * time.Second)
			defer ticker.Stop()
			for {
				attempted, reconciled, failed, sweepErr := runBuildMonitorSweeperTick(
					context.Background(),
					db,
					workflowRepo,
					logger,
					buildMonitorSweeperBatchSize,
				)
				if sweepErr != nil {
					sweeperFailureTotal.Add(1)
					processHealthStore.Upsert("build_monitor_sweeper", runtimehealth.ProcessStatus{
						Enabled:      true,
						Running:      true,
						LastActivity: time.Now().UTC(),
						Message:      "sweeper tick failed",
						Metrics: map[string]int64{
							"monitor_sweeper_reconciled_total": sweeperReconciledTotal.Load(),
							"monitor_sweeper_failures_total":   sweeperFailureTotal.Load(),
						},
					})
					logger.Warn("Build monitor sweeper tick failed", zap.Error(sweepErr))
				} else {
					sweeperReconciledTotal.Add(int64(reconciled))
					processHealthStore.Upsert("build_monitor_sweeper", runtimehealth.ProcessStatus{
						Enabled:      true,
						Running:      true,
						LastActivity: time.Now().UTC(),
						Message:      fmt.Sprintf("tick attempted=%d reconciled=%d failed=%d", attempted, reconciled, failed),
						Metrics: map[string]int64{
							"monitor_sweeper_reconciled_total": sweeperReconciledTotal.Load(),
							"monitor_sweeper_failures_total":   sweeperFailureTotal.Load(),
						},
					})
					if attempted > 0 || failed > 0 {
						logger.Info("Build monitor sweeper tick completed",
							zap.Int("attempted", attempted),
							zap.Int("reconciled", reconciled),
							zap.Int("failed", failed),
						)
					}
				}
				<-ticker.C
			}
		}()
	} else {
		processHealthStore.Upsert("build_monitor_sweeper", runtimehealth.ProcessStatus{
			Enabled:      false,
			Running:      false,
			LastActivity: time.Now().UTC(),
			Message:      "sweeper disabled",
			Metrics: map[string]int64{
				"monitor_sweeper_reconciled_total": 0,
				"monitor_sweeper_failures_total":   0,
			},
		})
	}

	// Initialize notification service
	notificationDomainService := notification.NewService(notificationTemplateRepo, logger)

	// Start background cleanup job for soft-deleted projects
	go func() {
		logger.Info("Background process starting",
			zap.String("component", "project_retention_cleanup"),
			zap.Duration("interval", 6*time.Hour),
		)
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()

		logRetention := func(ctx context.Context) int {
			retentionDays := 30
			config, err := systemConfigService.GetConfigByTypeAndKey(ctx, nil, systemconfig.ConfigTypeGeneral, "general")
			if err != nil && err != systemconfig.ErrConfigNotFound {
				logger.Warn("Failed to load general system config for retention", zap.Error(err))
				return retentionDays
			}
			if err == nil && config != nil {
				generalConfig, cfgErr := config.GetGeneralConfig()
				if cfgErr != nil {
					logger.Warn("Failed to parse general system config", zap.Error(cfgErr))
					return retentionDays
				}
				if generalConfig.ProjectRetentionDays >= 0 {
					retentionDays = generalConfig.ProjectRetentionDays
				}
			}
			logger.Info("Project retention cleanup configured", zap.Int("retention_days", retentionDays), zap.Duration("interval", 6*time.Hour))
			return retentionDays
		}

		retentionDays := logRetention(context.Background())
		for {
			ctx := context.Background()
			retentionDays = logRetention(ctx)

			if retentionDays > 0 {
				deletedCount, purgeErr := projectService.PurgeDeletedProjects(ctx, retentionDays)
				if purgeErr != nil {
					logger.Warn("Failed to purge deleted projects", zap.Error(purgeErr))
				} else if deletedCount > 0 {
					logger.Info("Purged soft-deleted projects", zap.Int("count", deletedCount), zap.Int("retention_days", retentionDays))
					if auditService != nil {
						_ = auditService.LogGlobalSystemAction(
							ctx,
							audit.AuditEventProjectPurge,
							"projects",
							"purge_deleted",
							"Auto-purged deleted projects",
							map[string]interface{}{
								"retention_days": retentionDays,
								"deleted_count":  deletedCount,
								"trigger":        "scheduled",
							},
						)
					}
				}
			}

			<-ticker.C
		}
	}()

	// Start workflow orchestrator
	defaultProviderReadinessWatcherEnabled := getBoolEnv("IF_PROVIDER_READINESS_WATCHER_ENABLED", true)
	defaultProviderReadinessWatcherIntervalSeconds := getIntEnv("IF_PROVIDER_READINESS_WATCHER_INTERVAL_SECONDS", 180)
	defaultProviderReadinessWatcherTimeoutSeconds := getIntEnv("IF_PROVIDER_READINESS_WATCHER_TIMEOUT_SECONDS", 90)
	defaultProviderReadinessWatcherBatchSize := getIntEnv("IF_PROVIDER_READINESS_WATCHER_BATCH_SIZE", 200)
	if defaultProviderReadinessWatcherIntervalSeconds < 30 {
		defaultProviderReadinessWatcherIntervalSeconds = 180
	}
	if defaultProviderReadinessWatcherTimeoutSeconds < 10 || defaultProviderReadinessWatcherTimeoutSeconds >= defaultProviderReadinessWatcherIntervalSeconds {
		defaultProviderReadinessWatcherTimeoutSeconds = defaultProviderReadinessWatcherIntervalSeconds - 1
		if defaultProviderReadinessWatcherTimeoutSeconds > 90 {
			defaultProviderReadinessWatcherTimeoutSeconds = 90
		}
		if defaultProviderReadinessWatcherTimeoutSeconds < 10 {
			defaultProviderReadinessWatcherTimeoutSeconds = 10
		}
	}
	if defaultProviderReadinessWatcherBatchSize <= 0 {
		defaultProviderReadinessWatcherBatchSize = 200
	}

	appbootstrap.StartProviderReadinessWatcher(
		logger,
		processHealthStore,
		systemConfigService,
		infrastructureService,
		appbootstrap.ProviderReadinessWatcherConfig{
			DefaultEnabled:         defaultProviderReadinessWatcherEnabled,
			DefaultIntervalSeconds: defaultProviderReadinessWatcherIntervalSeconds,
			DefaultTimeoutSeconds:  defaultProviderReadinessWatcherTimeoutSeconds,
			DefaultBatchSize:       defaultProviderReadinessWatcherBatchSize,
			OnTick: func(ctx context.Context, result *infrastructure.ProviderReadinessWatchTickResult, err error, observedAt time.Time) {
				appsresmartbot.ObserveProviderReadinessTick(ctx, sreSmartBotService, logger, result, err, observedAt)
			},
		},
	)

	appbootstrap.StartTenantAssetDriftWatcher(
		logger,
		processHealthStore,
		systemConfigService,
		infrastructureService,
		appbootstrap.TenantAssetDriftWatcherConfig{
			DefaultEnabled:         getBoolEnv("IF_TENANT_ASSET_DRIFT_WATCHER_ENABLED", true),
			DefaultIntervalSeconds: getIntEnv("IF_TENANT_ASSET_DRIFT_WATCHER_INTERVAL_SECONDS", 300),
			DefaultTimeoutSeconds:  getIntEnv("IF_TENANT_ASSET_DRIFT_WATCHER_TIMEOUT_SECONDS", 90),
			DefaultBatchSize:       getIntEnv("IF_TENANT_ASSET_DRIFT_WATCHER_BATCH_SIZE", 200),
			OnTick: func(ctx context.Context, result *infrastructure.TenantAssetDriftWatchTickResult, metrics infrastructure.TenantAssetDriftMetricsSnapshot, err error, observedAt time.Time) {
				appsresmartbot.ObserveTenantAssetDriftTick(ctx, sreSmartBotService, logger, result, metrics, err, observedAt)
			},
		},
	)

	releaseComplianceWatcherEnabled := getBoolEnv("IF_QUARANTINE_RELEASE_COMPLIANCE_WATCHER_ENABLED", true)
	releaseComplianceWatcherIntervalSeconds := getIntEnv("IF_QUARANTINE_RELEASE_COMPLIANCE_WATCHER_INTERVAL_SECONDS", 180)
	if releaseComplianceWatcherIntervalSeconds < 30 {
		releaseComplianceWatcherIntervalSeconds = 180
	}
	releaseComplianceWatcherTimeoutSeconds := getIntEnv("IF_QUARANTINE_RELEASE_COMPLIANCE_WATCHER_TIMEOUT_SECONDS", 20)
	if releaseComplianceWatcherTimeoutSeconds < 10 {
		releaseComplianceWatcherTimeoutSeconds = 20
	}
	if releaseComplianceWatcherTimeoutSeconds >= releaseComplianceWatcherIntervalSeconds {
		releaseComplianceWatcherTimeoutSeconds = releaseComplianceWatcherIntervalSeconds - 1
		if releaseComplianceWatcherTimeoutSeconds < 10 {
			releaseComplianceWatcherTimeoutSeconds = 10
		}
	}
	releaseComplianceWatcherBatchSize := getIntEnv("IF_QUARANTINE_RELEASE_COMPLIANCE_WATCHER_BATCH_SIZE", 500)
	if releaseComplianceWatcherBatchSize < 1 {
		releaseComplianceWatcherBatchSize = 500
	}
	appbootstrap.StartReleaseComplianceWatcher(
		logger,
		processHealthStore,
		releaseComplianceMetrics,
		func(ctx context.Context, limit int) ([]releasecompliance.DriftRecord, error) {
			return listReleaseComplianceDriftCandidates(ctx, db, limit)
		},
		func(ctx context.Context) (int64, error) {
			return countReleaseComplianceReleasedArtifacts(ctx, db)
		},
		func(ctx context.Context, eventType string, record releasecompliance.DriftRecord, stateField string, stateValue string) error {
			return publishReleaseComplianceEvent(
				ctx,
				eventBus,
				eventSource,
				cfg.Messaging.SchemaVersion,
				eventType,
				record,
				stateField,
				stateValue,
			)
		},
		appbootstrap.ReleaseComplianceWatcherConfig{
			Enabled:       releaseComplianceWatcherEnabled,
			Interval:      time.Duration(releaseComplianceWatcherIntervalSeconds) * time.Second,
			Timeout:       time.Duration(releaseComplianceWatcherTimeoutSeconds) * time.Second,
			BatchSize:     releaseComplianceWatcherBatchSize,
			SchemaVersion: cfg.Messaging.SchemaVersion,
			EventSource:   eventSource,
			OnTick: func(ctx context.Context, detected []releasecompliance.DriftRecord, recovered []releasecompliance.DriftRecord, snapshot releasecompliance.Snapshot, err error, observedAt time.Time) {
				appsresmartbot.ObserveReleaseComplianceTick(ctx, sreSmartBotService, logger, detected, recovered, snapshot, err, observedAt)
			},
		},
	)

	if cfg.Workflow.Enabled {
		imageImportRepo := postgres.NewImageImportRepository(db, logger)
		orchestratorController = appbootstrap.StartWorkflowRunner(
			appbootstrap.WorkflowRunnerDeps{
				ProcessHealthStore:  processHealthStore,
				BuildService:        buildService,
				Preflight:           newWorkflowInfrastructurePreflight(infrastructureService, logger),
				WorkflowRepo:        workflowRepo,
				ImageImportRepo:     imageImportRepo,
				PipelineManager:     pipelineMgr,
				Infrastructure:      infrastructureService,
				SystemConfigService: systemConfigService,
				EventBus:            eventBus,
				Logger:              logger,
			},
			appbootstrap.WorkflowRunnerConfig{
				Enabled:         cfg.Workflow.Enabled,
				PollInterval:    cfg.Workflow.PollInterval,
				MaxStepsPerTick: cfg.Workflow.MaxStepsPerTick,
				BuildTektonMode: cfg.Build.TektonEnabled,
			},
		)
	} else {
		processHealthStore.Upsert("workflow_orchestrator", runtimehealth.ProcessStatus{
			Enabled:      false,
			Running:      false,
			LastActivity: time.Now().UTC(),
			Message:      "workflow orchestrator disabled",
		})
		logger.Info("Background process disabled",
			zap.String("component", "workflow_orchestrator"),
		)
	}

	// Initialize email service and notification service for user invitations
	emailRepo := postgres.NewEmailRepository(db, logger)
	emailDomainService := domainEmail.NewServiceWithNotification(
		emailRepo,
		logger,
		os.Getenv("IF_SMTP_HOST"),
		getIntEnv("IF_SMTP_PORT", 1025),
		os.Getenv("IF_SMTP_FROM_EMAIL"),
		notificationDomainService,
	)
	notificationService := email.NewNotificationService(
		emailDomainService,
		eventBus,
		cfg.Messaging.EnableNATS,
		logger,
		os.Getenv("IF_SMTP_FROM_EMAIL"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"), // Default company ID
		systemConfigService,
	)
	buildNotificationEventSubscriber := buildnotifications.NewEventSubscriber(buildRepo, projectNotificationTriggerService, buildNotificationDeliveryRepo, notificationService, logger)
	buildNotificationEventSubscriber.SetRealtimePublisher(wsHub)
	buildNotificationEventUnsubscribe := buildnotifications.RegisterEventSubscriber(eventBus, buildNotificationEventSubscriber)
	defer buildNotificationEventUnsubscribe()
	imageImportNotificationSubscriber := imageimportnotifications.NewEventSubscriber(buildNotificationDeliveryRepo, logger)
	imageImportNotificationSubscriber.SetEmailSender(notificationService)
	imageImportNotificationSubscriber.SetRealtimePublisher(wsHub)
	imageImportNotificationEventUnsubscribe := imageimportnotifications.RegisterEventSubscriber(eventBus, imageImportNotificationSubscriber)
	defer imageImportNotificationEventUnsubscribe()
	buildnotifications.StartHealthReporter(logger, processHealthStore, buildNotificationEventSubscriber)

	runtimeDependencyWatcherEnabled := getBoolEnv("IF_RUNTIME_DEPENDENCY_WATCHER_ENABLED", true)
	runtimeDependencyWatcherIntervalSeconds := getIntEnv("IF_RUNTIME_DEPENDENCY_WATCHER_INTERVAL_SECONDS", 60)
	if runtimeDependencyWatcherIntervalSeconds < 15 {
		runtimeDependencyWatcherIntervalSeconds = 60
	}
	runtimeDependencyAlertCooldownSeconds := getIntEnv("IF_RUNTIME_DEPENDENCY_ALERT_COOLDOWN_SECONDS", 900)
	if runtimeDependencyAlertCooldownSeconds < 0 {
		runtimeDependencyAlertCooldownSeconds = 0
	}
	runtimeDependencyCheckTimeoutSeconds := getIntEnv("IF_RUNTIME_DEPENDENCY_CHECK_TIMEOUT_SECONDS", 5)
	if runtimeDependencyCheckTimeoutSeconds < 1 {
		runtimeDependencyCheckTimeoutSeconds = 5
	}
	internalRegistryGCWorkerEnabled := getBoolEnv("IF_INTERNAL_REGISTRY_TEMP_CLEANUP_ENABLED", true)
	internalRegistryGCWorkerHealthURL := strings.TrimSpace(os.Getenv("IF_INTERNAL_REGISTRY_GC_WORKER_HEALTH_URL"))
	if internalRegistryGCWorkerHealthURL == "" {
		internalRegistryGCWorkerPort := getIntEnv("IF_INTERNAL_REGISTRY_GC_WORKER_PORT", 8085)
		internalRegistryGCWorkerHealthURL = fmt.Sprintf("http://localhost:%d/health", internalRegistryGCWorkerPort)
	}
	internalRegistryGCWorkerHealthTimeoutSeconds := getIntEnv("IF_INTERNAL_REGISTRY_GC_WORKER_HEALTH_TIMEOUT_SECONDS", runtimeDependencyCheckTimeoutSeconds)
	if internalRegistryGCWorkerHealthTimeoutSeconds < 1 {
		internalRegistryGCWorkerHealthTimeoutSeconds = runtimeDependencyCheckTimeoutSeconds
	}
	appsresmartbot.StartRuntimeDependencyWatcher(
		logger,
		db,
		processHealthStore,
		sreSmartBotService,
		dispatcherRuntimeStore,
		monitorSubscriber,
		buildNotificationEventSubscriber,
		relay,
		func(ctx context.Context, title string, message string, notificationType string) (int, error) {
			return emitRuntimeDependencyNotification(
				ctx,
				db,
				buildNotificationDeliveryRepo,
				wsHub,
				title,
				message,
				notificationType,
			)
		},
		appsresmartbot.RuntimeDependencyWatcherConfig{
			Enabled:                         runtimeDependencyWatcherEnabled,
			Interval:                        time.Duration(runtimeDependencyWatcherIntervalSeconds) * time.Second,
			AlertCooldown:                   time.Duration(runtimeDependencyAlertCooldownSeconds) * time.Second,
			CheckTimeout:                    time.Duration(runtimeDependencyCheckTimeoutSeconds) * time.Second,
			InternalRegistryGCWorkerEnabled: internalRegistryGCWorkerEnabled,
			InternalRegistryGCWorkerURL:     internalRegistryGCWorkerHealthURL,
			InternalRegistryGCWorkerTimeout: time.Duration(internalRegistryGCWorkerHealthTimeoutSeconds) * time.Second,
			DispatcherEnabled:               cfg.Dispatcher.Enabled,
			WorkflowEnabled:                 cfg.Workflow.Enabled,
		},
	)

	clusterMetricsSnapshotIngesterEnabled := getBoolEnv("IF_CLUSTER_METRICS_SNAPSHOT_INGESTER_ENABLED", true)
	clusterMetricsSnapshotIntervalSeconds := getIntEnv("IF_CLUSTER_METRICS_SNAPSHOT_INTERVAL_SECONDS", 300)
	if clusterMetricsSnapshotIntervalSeconds < 60 {
		clusterMetricsSnapshotIntervalSeconds = 300
	}
	clusterMetricsSnapshotTimeoutSeconds := getIntEnv("IF_CLUSTER_METRICS_SNAPSHOT_TIMEOUT_SECONDS", 20)
	if clusterMetricsSnapshotTimeoutSeconds < 5 {
		clusterMetricsSnapshotTimeoutSeconds = 20
	}
	clusterMetricsRetentionDays := getIntEnv("IF_CLUSTER_METRICS_RETENTION_DAYS", 14)
	if clusterMetricsRetentionDays < 1 {
		clusterMetricsRetentionDays = 14
	}
	clusterMetricsNodeCPUSaturationPercent := getIntEnv("IF_CLUSTER_METRICS_NODE_CPU_SATURATION_PERCENT", 85)
	if clusterMetricsNodeCPUSaturationPercent < 1 {
		clusterMetricsNodeCPUSaturationPercent = 85
	}
	clusterMetricsNodeMemorySaturationPercent := getIntEnv("IF_CLUSTER_METRICS_NODE_MEMORY_SATURATION_PERCENT", 85)
	if clusterMetricsNodeMemorySaturationPercent < 1 {
		clusterMetricsNodeMemorySaturationPercent = 85
	}
	clusterMetricsPodRestartThreshold := getIntEnv("IF_CLUSTER_METRICS_POD_RESTART_THRESHOLD", 3)
	if clusterMetricsPodRestartThreshold < 1 {
		clusterMetricsPodRestartThreshold = 3
	}
	clusterMetricsClusterName := strings.TrimSpace(os.Getenv("IF_CLUSTER_METRICS_CLUSTER_NAME"))
	if clusterMetricsClusterName == "" {
		clusterMetricsClusterName = "image-factory"
	}
	clusterMetricsRepo := postgres.NewClusterMetricsSnapshotRepository(db, logger)
	clustermetrics.StartSnapshotIngester(
		logger,
		processHealthStore,
		k8sClient,
		clusterMetricsClient,
		clusterMetricsRepo,
		sreSmartBotService,
		clustermetrics.RunnerConfig{
			Enabled:                     clusterMetricsSnapshotIngesterEnabled,
			Interval:                    time.Duration(clusterMetricsSnapshotIntervalSeconds) * time.Second,
			Timeout:                     time.Duration(clusterMetricsSnapshotTimeoutSeconds) * time.Second,
			RetentionDays:               clusterMetricsRetentionDays,
			ClusterName:                 clusterMetricsClusterName,
			NodeCPUSaturationPercent:    clusterMetricsNodeCPUSaturationPercent,
			NodeMemorySaturationPercent: clusterMetricsNodeMemorySaturationPercent,
			PodRestartThreshold:         int32(clusterMetricsPodRestartThreshold),
		},
	)

	imageImportNotificationReceiptCleanupEnabled := getBoolEnv("IF_IMAGE_IMPORT_NOTIFICATION_RECEIPT_CLEANUP_ENABLED", true)
	imageImportNotificationReceiptRetentionDays := getIntEnv("IF_IMAGE_IMPORT_NOTIFICATION_RECEIPT_RETENTION_DAYS", 30)
	if imageImportNotificationReceiptRetentionDays < 1 {
		imageImportNotificationReceiptRetentionDays = 30
	}
	imageImportNotificationReceiptCleanupIntervalHours := getIntEnv("IF_IMAGE_IMPORT_NOTIFICATION_RECEIPT_CLEANUP_INTERVAL_HOURS", 24)
	if imageImportNotificationReceiptCleanupIntervalHours < 1 {
		imageImportNotificationReceiptCleanupIntervalHours = 24
	}
	type imageImportNotificationReceiptCleanupRuntimeConfig struct {
		enabled   bool
		retention int
		interval  time.Duration
		source    string
	}
	loadImageImportNotificationReceiptCleanupConfig := func(ctx context.Context) imageImportNotificationReceiptCleanupRuntimeConfig {
		out := imageImportNotificationReceiptCleanupRuntimeConfig{
			enabled:   imageImportNotificationReceiptCleanupEnabled,
			retention: imageImportNotificationReceiptRetentionDays,
			interval:  time.Duration(imageImportNotificationReceiptCleanupIntervalHours) * time.Hour,
			source:    "env",
		}
		cfg, err := systemConfigService.GetConfigByTypeAndKey(ctx, nil, systemconfig.ConfigTypeRuntimeServices, "runtime_services")
		if err != nil || cfg == nil {
			return out
		}
		runtimeCfg, cfgErr := cfg.GetRuntimeServicesConfig()
		if cfgErr != nil || runtimeCfg == nil {
			return out
		}
		if runtimeCfg.ImageImportNotificationReceiptCleanupEnabled != nil {
			out.enabled = *runtimeCfg.ImageImportNotificationReceiptCleanupEnabled
			out.source = "system_config"
		}
		if runtimeCfg.ImageImportNotificationReceiptRetentionDays >= 1 {
			if runtimeCfg.ImageImportNotificationReceiptRetentionDays <= 3650 {
				out.retention = runtimeCfg.ImageImportNotificationReceiptRetentionDays
				out.source = "system_config"
			} else {
				logger.Warn("Ignoring runtime_services.image_import_notification_receipt_retention_days out of bounds",
					zap.Int("configured", runtimeCfg.ImageImportNotificationReceiptRetentionDays),
					zap.Int("max", 3650),
				)
			}
		}
		if runtimeCfg.ImageImportNotificationReceiptCleanupIntervalHours >= 1 {
			if runtimeCfg.ImageImportNotificationReceiptCleanupIntervalHours <= 168 {
				out.interval = time.Duration(runtimeCfg.ImageImportNotificationReceiptCleanupIntervalHours) * time.Hour
				out.source = "system_config"
			} else {
				logger.Warn("Ignoring runtime_services.image_import_notification_receipt_cleanup_interval_hours out of bounds",
					zap.Int("configured", runtimeCfg.ImageImportNotificationReceiptCleanupIntervalHours),
					zap.Int("max", 168),
				)
			}
		}
		return out
	}
	initialImageImportNotificationReceiptCleanupCfg := loadImageImportNotificationReceiptCleanupConfig(context.Background())
	processHealthStore.Upsert("image_import_notification_receipt_cleanup", runtimehealth.ProcessStatus{
		Enabled:      initialImageImportNotificationReceiptCleanupCfg.enabled,
		Running:      initialImageImportNotificationReceiptCleanupCfg.enabled,
		LastActivity: time.Now().UTC(),
		Message:      "image import notification receipt cleanup initialized",
		Metrics: map[string]int64{
			"receipt_cleanup_runs_total":       0,
			"receipt_cleanup_failures_total":   0,
			"receipt_cleanup_deleted_total":    0,
			"receipt_cleanup_retention_days":   int64(initialImageImportNotificationReceiptCleanupCfg.retention),
			"receipt_cleanup_interval_hours":   int64(initialImageImportNotificationReceiptCleanupCfg.interval / time.Hour),
			"receipt_cleanup_last_deleted":     0,
			"receipt_cleanup_last_run_success": 0,
		},
	})
	logger.Info("Background process starting",
		zap.String("component", "image_import_notification_receipt_cleanup"),
		zap.Int("retention_days", initialImageImportNotificationReceiptCleanupCfg.retention),
		zap.Int64("interval_hours", int64(initialImageImportNotificationReceiptCleanupCfg.interval/time.Hour)),
		zap.String("config_source", initialImageImportNotificationReceiptCleanupCfg.source),
		zap.Bool("enabled", initialImageImportNotificationReceiptCleanupCfg.enabled),
	)
	go func() {
		currentCfg := initialImageImportNotificationReceiptCleanupCfg
		ticker := time.NewTicker(currentCfg.interval)
		defer ticker.Stop()

		var runsTotal int64
		var failuresTotal int64
		var deletedTotal int64

		runCleanup := func(cfg imageImportNotificationReceiptCleanupRuntimeConfig) {
			now := time.Now().UTC()
			if !cfg.enabled {
				processHealthStore.Upsert("image_import_notification_receipt_cleanup", runtimehealth.ProcessStatus{
					Enabled:      false,
					Running:      false,
					LastActivity: now,
					Message:      "image import notification receipt cleanup disabled",
					Metrics: map[string]int64{
						"receipt_cleanup_runs_total":       runsTotal,
						"receipt_cleanup_failures_total":   failuresTotal,
						"receipt_cleanup_deleted_total":    deletedTotal,
						"receipt_cleanup_retention_days":   int64(cfg.retention),
						"receipt_cleanup_interval_hours":   int64(cfg.interval / time.Hour),
						"receipt_cleanup_last_deleted":     0,
						"receipt_cleanup_last_run_success": 0,
					},
				})
				return
			}
			runsTotal++
			cutoff := now.Add(-time.Duration(cfg.retention) * 24 * time.Hour)
			deleted, err := buildNotificationDeliveryRepo.DeleteImageImportNotificationReceiptsOlderThan(context.Background(), cutoff)
			if err != nil {
				failuresTotal++
				processHealthStore.Upsert("image_import_notification_receipt_cleanup", runtimehealth.ProcessStatus{
					Enabled:      true,
					Running:      true,
					LastActivity: now,
					Message:      fmt.Sprintf("cleanup failed: %v", err),
					Metrics: map[string]int64{
						"receipt_cleanup_runs_total":       runsTotal,
						"receipt_cleanup_failures_total":   failuresTotal,
						"receipt_cleanup_deleted_total":    deletedTotal,
						"receipt_cleanup_retention_days":   int64(cfg.retention),
						"receipt_cleanup_interval_hours":   int64(cfg.interval / time.Hour),
						"receipt_cleanup_last_deleted":     0,
						"receipt_cleanup_last_run_success": 0,
					},
				})
				logger.Warn("Image import notification receipt cleanup failed", zap.Error(err))
				return
			}
			deletedTotal += deleted
			processHealthStore.Upsert("image_import_notification_receipt_cleanup", runtimehealth.ProcessStatus{
				Enabled:      true,
				Running:      true,
				LastActivity: now,
				Message:      fmt.Sprintf("cleanup completed deleted=%d retention_days=%d", deleted, cfg.retention),
				Metrics: map[string]int64{
					"receipt_cleanup_runs_total":       runsTotal,
					"receipt_cleanup_failures_total":   failuresTotal,
					"receipt_cleanup_deleted_total":    deletedTotal,
					"receipt_cleanup_retention_days":   int64(cfg.retention),
					"receipt_cleanup_interval_hours":   int64(cfg.interval / time.Hour),
					"receipt_cleanup_last_deleted":     deleted,
					"receipt_cleanup_last_run_success": 1,
				},
			})
			if deleted > 0 {
				logger.Info("Image import notification receipt cleanup deleted stale rows",
					zap.Int64("deleted", deleted),
					zap.Time("cutoff", cutoff),
				)
			}
		}

		runCleanup(currentCfg)
		for {
			<-ticker.C
			nextCfg := loadImageImportNotificationReceiptCleanupConfig(context.Background())
			if nextCfg.interval != currentCfg.interval {
				ticker.Reset(nextCfg.interval)
				logger.Info("Image import notification receipt cleanup interval updated",
					zap.Duration("interval", nextCfg.interval),
					zap.String("config_source", nextCfg.source),
				)
			}
			currentCfg = nextCfg
			runCleanup(currentCfg)
		}
	}()

	userService := user.NewServiceWithInvitationsPasswordResetAndConfig(userRepo, userInvitationRepo, passwordResetTokenRepo, notificationService, systemConfigService, logger, cfg.Auth.JWTSecret, cfg.Frontend.BaseURL)

	// Initialize LDAP client and service
	ldapClient := ldap.NewClient(&ldap.Config{
		Host:         cfg.Auth.LDAPServer,
		Port:         cfg.Auth.LDAPPort,
		BaseDN:       cfg.Auth.LDAPBaseDN,
		BindDN:       cfg.Auth.LDAPBindDN,
		BindPassword: cfg.Auth.LDAPBindPassword,
		UserFilter:   "(uid=%s)",    // Search by uid - works with both standard LDAP and GLauth. If not found, fallback to mail in searchUser
		GroupFilter:  "(member=%s)", // TODO: Get from config
		UseTLS:       cfg.Auth.LDAPUseTLS,
		StartTLS:     cfg.Auth.LDAPStartTLS,
	}, logger)
	ldapService := user.NewLDAPService(userRepo, logger, cfg.Auth.JWTSecret, ldapClient, systemConfigService)

	// Validate JWT secret is set (especially important in production)
	if cfg.Auth.JWTSecret == "" {
		logger.Fatal("JWT_SECRET not configured. Set IF_AUTH_JWT_SECRET environment variable")
	}

	// Initialize HTTP router
	router := rest.NewRouter(cfg, logger, db, auditService)

	httpSignalRunnerEnabled := getBoolEnv("IF_HTTP_SIGNAL_RUNNER_ENABLED", true)
	httpSignalIntervalSeconds := getIntEnv("IF_HTTP_SIGNAL_INTERVAL_SECONDS", 120)
	if httpSignalIntervalSeconds < 30 {
		httpSignalIntervalSeconds = 120
	}
	httpSignalMinRequestCount := int64(getIntEnv("IF_HTTP_SIGNAL_MIN_REQUEST_COUNT", 20))
	if httpSignalMinRequestCount < 1 {
		httpSignalMinRequestCount = 20
	}
	httpSignalErrorRatePercent := getIntEnv("IF_HTTP_SIGNAL_ERROR_RATE_PERCENT", 10)
	if httpSignalErrorRatePercent < 1 {
		httpSignalErrorRatePercent = 10
	}
	httpSignalAverageLatencyThresholdMs := int64(getIntEnv("IF_HTTP_SIGNAL_AVG_LATENCY_THRESHOLD_MS", 800))
	if httpSignalAverageLatencyThresholdMs < 1 {
		httpSignalAverageLatencyThresholdMs = 800
	}
	httpSignalRequestVolumeThreshold := int64(getIntEnv("IF_HTTP_SIGNAL_REQUEST_VOLUME_THRESHOLD", 250))
	if httpSignalRequestVolumeThreshold < 1 {
		httpSignalRequestVolumeThreshold = 250
	}
	httpSignalRetentionDays := getIntEnv("IF_HTTP_SIGNAL_RETENTION_DAYS", 7)
	if httpSignalRetentionDays < 1 {
		httpSignalRetentionDays = 7
	}
	asyncBacklogRunnerEnabled := getBoolEnv("IF_ASYNC_BACKLOG_SIGNAL_RUNNER_ENABLED", true)
	asyncBacklogIntervalSeconds := getIntEnv("IF_ASYNC_BACKLOG_SIGNAL_INTERVAL_SECONDS", 120)
	if asyncBacklogIntervalSeconds < 30 {
		asyncBacklogIntervalSeconds = 120
	}
	asyncBacklogBuildQueueThreshold := int64(getIntEnv("IF_ASYNC_BACKLOG_BUILD_QUEUE_THRESHOLD", 10))
	if asyncBacklogBuildQueueThreshold < 1 {
		asyncBacklogBuildQueueThreshold = 10
	}
	asyncBacklogEmailQueueThreshold := int64(getIntEnv("IF_ASYNC_BACKLOG_EMAIL_QUEUE_THRESHOLD", 20))
	if asyncBacklogEmailQueueThreshold < 1 {
		asyncBacklogEmailQueueThreshold = 20
	}
	asyncBacklogOutboxThreshold := int64(getIntEnv("IF_ASYNC_BACKLOG_OUTBOX_THRESHOLD", 15))
	if asyncBacklogOutboxThreshold < 1 {
		asyncBacklogOutboxThreshold = 15
	}
	appSignalsRepo := postgres.NewAppSignalsRepository(db, logger)
	httpSignalStore := router.HTTPRequestSignalStore()
	appsignals.StartHTTPGoldenSignalRunner(
		logger,
		processHealthStore,
		appsignals.HTTPSignalSnapshotterFunc(func(now time.Time) appsignals.HTTPRequestSignalSnapshot {
			if httpSignalStore == nil {
				return appsignals.HTTPRequestSignalSnapshot{}
			}
			snapshot := httpSignalStore.SnapshotAndReset(now)
			return appsignals.HTTPRequestSignalSnapshot{
				WindowStartedAt:  snapshot.WindowStartedAt,
				WindowEndedAt:    snapshot.WindowEndedAt,
				RequestCount:     snapshot.RequestCount,
				ServerErrorCount: snapshot.ServerErrorCount,
				ClientErrorCount: snapshot.ClientErrorCount,
				TotalLatencyMs:   snapshot.TotalLatencyMs,
				MaxLatencyMs:     snapshot.MaxLatencyMs,
			}
		}),
		appSignalsRepo,
		sreSmartBotService,
		appsignals.RunnerConfig{
			Enabled:                 httpSignalRunnerEnabled,
			Interval:                time.Duration(httpSignalIntervalSeconds) * time.Second,
			RetentionDays:           httpSignalRetentionDays,
			MinRequestCount:         httpSignalMinRequestCount,
			ErrorRatePercent:        httpSignalErrorRatePercent,
			AverageLatencyThreshold: httpSignalAverageLatencyThresholdMs,
			RequestVolumeThreshold:  httpSignalRequestVolumeThreshold,
		},
	)
	asyncsignals.StartAsyncBacklogSignalRunner(
		logger,
		db,
		processHealthStore,
		sreSmartBotService,
		asyncsignals.RunnerConfig{
			Enabled:                  asyncBacklogRunnerEnabled,
			Interval:                 time.Duration(asyncBacklogIntervalSeconds) * time.Second,
			BuildQueueThreshold:      asyncBacklogBuildQueueThreshold,
			EmailQueueThreshold:      asyncBacklogEmailQueueThreshold,
			MessagingOutboxThreshold: asyncBacklogOutboxThreshold,
		},
	)

	// Setup routes
	rest.SetupRoutes(router, tenantRepo, tenantService, buildRepo, buildService, buildExecutionService, projectRepo, projectService, userRepo, userService, rbacRepo, rbacService, systemConfigRepo, logger, ldapService, auditService, notificationTemplateRepo, wsHub, db, cfg, buildPolicyService, dispatcherMetricsProvider, dispatcherController, orchestratorController, dispatcherRuntimeStore, processHealthStore, cfg.Dispatcher.Enabled, infrastructureEventPublisher, releaseComplianceMetrics, eventBus)

	// Log server start event
	if auditService != nil {
		auditService.LogGlobalSystemAction(context.Background(), audit.AuditEventServerStart, "server", "start", "Server started successfully", map[string]interface{}{
			"version":     cfg.Server.Version,
			"port":        cfg.Server.Port,
			"environment": cfg.Server.Environment,
		})
	}

	// Send health check email on startup (disabled for now due to config parsing issues)
	// go func() {
	// 	// Wait a moment for server to fully start
	// 	time.Sleep(2 * time.Second)
	//
	// 	// Get default tenant (get the first available tenant)
	// 	tenants, err := tenantRepo.FindAll(context.Background(), tenant.TenantFilter{Limit: 1})
	// 	if err != nil || len(tenants) == 0 {
	// 		logger.Error("Failed to get default tenant for health check email", zap.Error(err))
	// 		return
	// 	}
	// 	defaultTenant := tenants[0]
	// 	defaultTenantID := defaultTenant.ID()
	// 	defaultCompanyID := defaultTenant.CompanyID()
	//
	// 	// Get SMTP config from system configuration
	// 	smtpConfigs, err := systemConfigRepo.FindActiveByType(context.Background(), defaultTenantID, systemconfig.ConfigTypeSMTP)
	// 	smtpConfig := make(map[string]interface{})
	// 	if err == nil && len(smtpConfigs) > 0 {
	// 		// Use the first active SMTP config
	// 		config := smtpConfigs[0]
	// 		json.Unmarshal(config.ConfigValue(), &smtpConfig)
	// 	}
	//
	// 	// Send health check email to admin
	// 	adminEmail := "admin@image-factory.com" // TODO: Get from config
	// 	if err := emailService.SendHealthCheckEmail(context.Background(), defaultCompanyID, defaultTenantID, adminEmail, nil); err != nil {
	// 		logger.Error("Failed to send health check email", zap.Error(err))
	// 	} else {
	// 		logger.Info("Health check email sent successfully", zap.String("admin_email", adminEmail))
	// 	}
	// }()

	// Create HTTP server with timeouts
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  30 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting HTTP server",
			zap.Int("port", cfg.Server.Port),
			zap.String("address", fmt.Sprintf("0.0.0.0:%d", cfg.Server.Port)),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Received shutdown signal, initiating graceful shutdown...")

	// Give outstanding requests a deadline for completion
	shutdownTimeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Shutdown HTTP server
	logger.Info("Shutting down HTTP server", zap.Duration("timeout", shutdownTimeout))
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	// Close database connection
	logger.Info("Closing database connection")
	if db != nil {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database connection", zap.Error(err))
		} else {
			logger.Info("Database connection closed successfully")
		}
	}

	logger.Info("Server shutdown complete")
}
