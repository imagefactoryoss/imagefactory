package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/config"
	"github.com/srikarm/image-factory/internal/infrastructure/database"
	"github.com/srikarm/image-factory/internal/infrastructure/logger"
	"github.com/subosito/gotenv"
	"go.uber.org/zap"
)

type cleanupCandidate struct {
	BuildID     uuid.UUID       `db:"build_id"`
	TenantID    uuid.UUID       `db:"tenant_id"`
	CompletedAt time.Time       `db:"completed_at"`
	Metadata    json.RawMessage `db:"metadata"`
}

type workerSettings struct {
	Enabled             bool
	RetentionHours      int
	IntervalMinutes     int
	BatchSize           int
	DryRun              bool
	HealthPort          int
	RegistryScheme      string
	RegistryInsecure    bool
	RegistryUsername    string
	RegistryPassword    string
	RegistryBearer      string
	DefaultTempRegistry string
	DefaultTempRepo     string
}

type healthState struct {
	mu                   sync.RWMutex
	LastRunAt            time.Time `json:"last_run_at,omitempty"`
	LastRunDuration      int64     `json:"last_run_duration_ms,omitempty"`
	LastRunCandidates    int       `json:"last_run_candidates,omitempty"`
	LastRunDeleted       int       `json:"last_run_deleted,omitempty"`
	LastRunErrors        int       `json:"last_run_errors,omitempty"`
	LastRunReclaimedByte int64     `json:"last_run_reclaimed_bytes,omitempty"`
	TotalDeleted         int64     `json:"total_deleted,omitempty"`
	TotalReclaimedBytes  int64     `json:"total_reclaimed_bytes,omitempty"`
	Message              string    `json:"message,omitempty"`
}

func (h *healthState) update(candidates, deleted, runErrors int, reclaimedBytes int64, duration time.Duration, msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.LastRunAt = time.Now().UTC()
	h.LastRunDuration = duration.Milliseconds()
	h.LastRunCandidates = candidates
	h.LastRunDeleted = deleted
	h.LastRunErrors = runErrors
	h.LastRunReclaimedByte = reclaimedBytes
	h.TotalDeleted += int64(deleted)
	h.TotalReclaimedBytes += reclaimedBytes
	h.Message = msg
}

func (h *healthState) snapshot() healthState {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return healthState{
		LastRunAt:            h.LastRunAt,
		LastRunDuration:      h.LastRunDuration,
		LastRunCandidates:    h.LastRunCandidates,
		LastRunDeleted:       h.LastRunDeleted,
		LastRunErrors:        h.LastRunErrors,
		LastRunReclaimedByte: h.LastRunReclaimedByte,
		TotalDeleted:         h.TotalDeleted,
		TotalReclaimedBytes:  h.TotalReclaimedBytes,
		Message:              h.Message,
	}
}

func main() {
	var envFile = flag.String("env", "", "Path to environment file (e.g., .env.development, .env.production)")
	flag.Parse()

	if *envFile != "" {
		if !filepath.IsAbs(*envFile) {
			if absPath, err := filepath.Abs(*envFile); err == nil {
				*envFile = absPath
			}
		}
		if _, err := os.Stat(*envFile); err != nil {
			log.Fatalf("Environment file not found: %s (%v)", *envFile, err)
		}
		if err := gotenv.Load(*envFile); err != nil {
			log.Fatalf("Failed to load environment file %s: %v", *envFile, err)
		}
		log.Printf("Loaded environment from: %s", *envFile)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	appLogger, err := logger.NewLogger(cfg.Logger)
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer appLogger.Sync()

	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		appLogger.Fatal("Failed to connect database", zap.Error(err))
	}
	defer db.Close()
	if readiness, readinessErr := database.CheckSchemaReadiness(context.Background(), db); readinessErr != nil {
		appLogger.Warn("Database schema readiness check failed", zap.Error(readinessErr))
	} else {
		if readiness.CurrentSchema != cfg.Database.Schema {
			appLogger.Warn("Database schema mismatch detected",
				zap.String("configured_schema", cfg.Database.Schema),
				zap.String("current_schema", readiness.CurrentSchema),
			)
		}
		appLogger.Info("Database schema readiness check completed",
			zap.String("current_schema", readiness.CurrentSchema),
			zap.Bool("schema_migrations_present", readiness.SchemaMigrations),
			zap.Bool("tenants_domain_column", readiness.TenantsDomainColumn),
			zap.Bool("tenants_domain_not_null", readiness.TenantsDomainNotNull),
		)
	}

	systemConfigRepo := postgres.NewSystemConfigRepository(db, appLogger)

	state := &healthState{Message: "initializing"}
	settings := resolveSettings(context.Background(), systemConfigRepo, appLogger)

	healthServer := startHealthServer(settings.HealthPort, state, appLogger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		runWorkerLoop(ctx, db, systemConfigRepo, state, appLogger)
	}()

	appLogger.Info("Internal registry GC worker started", zap.Int("health_port", settings.HealthPort))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	appLogger.Info("Shutting down internal registry GC worker")
	cancel()
	<-done

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = healthServer.Shutdown(shutdownCtx)
}

func startHealthServer(port int, state *healthState, appLogger *zap.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		snapshot := state.snapshot()
		resp := map[string]interface{}{
			"status":                   "healthy",
			"service":                  "internal-registry-gc-worker",
			"timestamp":                time.Now().UTC().Format(time.RFC3339),
			"last_run_at":              "",
			"last_run_duration_ms":     snapshot.LastRunDuration,
			"last_run_candidates":      snapshot.LastRunCandidates,
			"last_run_deleted":         snapshot.LastRunDeleted,
			"last_run_errors":          snapshot.LastRunErrors,
			"last_run_reclaimed_bytes": snapshot.LastRunReclaimedByte,
			"total_deleted":            snapshot.TotalDeleted,
			"total_reclaimed_bytes":    snapshot.TotalReclaimedBytes,
			"message":                  snapshot.Message,
		}
		if !snapshot.LastRunAt.IsZero() {
			resp["last_run_at"] = snapshot.LastRunAt.Format(time.RFC3339)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	go func() {
		appLogger.Info("Starting internal registry GC health server", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Internal registry GC health server failed", zap.Error(err))
		}
	}()
	return server
}

func runWorkerLoop(
	ctx context.Context,
	db DBTX,
	systemConfigRepo *postgres.SystemConfigRepository,
	state *healthState,
	appLogger *zap.Logger,
) {
	for {
		settings := resolveSettings(ctx, systemConfigRepo, appLogger)
		start := time.Now()
		candidates := 0
		deleted := 0
		runErrors := 0
		reclaimedBytes := int64(0)

		if settings.Enabled {
			var err error
			candidates, deleted, runErrors, reclaimedBytes, err = runCleanupCycle(ctx, db, settings, appLogger)
			if err != nil {
				runErrors++
				appLogger.Error("Cleanup cycle failed", zap.Error(err))
			}
			state.update(candidates, deleted, runErrors, reclaimedBytes, time.Since(start), "cleanup cycle complete")
		} else {
			state.update(0, 0, 0, 0, time.Since(start), "cleanup disabled")
		}

		wait := time.Duration(settings.IntervalMinutes) * time.Minute
		if wait < time.Minute {
			wait = time.Minute
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

type DBTX interface {
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

func runCleanupCycle(ctx context.Context, db DBTX, settings workerSettings, appLogger *zap.Logger) (int, int, int, int64, error) {
	candidates, err := loadCleanupCandidates(ctx, db, settings.RetentionHours, settings.BatchSize)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	if len(candidates) == 0 {
		appLogger.Info("No temp scan images eligible for cleanup")
		return 0, 0, 0, 0, nil
	}

	client := &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: settings.RegistryInsecure}, //nolint:gosec
		},
	}

	deleted := 0
	runErrors := 0
	reclaimedBytes := int64(0)
	for _, c := range candidates {
		tempRef, include := resolveTempScanImageRef(c, settings)
		if !include || strings.TrimSpace(tempRef) == "" {
			continue
		}
		if settings.DryRun {
			appLogger.Info("Dry-run cleanup candidate", zap.String("build_id", c.BuildID.String()), zap.String("image_ref", tempRef))
			continue
		}
		estimatedBytes, err := deleteRegistryTagReference(ctx, client, settings, tempRef)
		if err != nil {
			runErrors++
			appLogger.Warn("Failed deleting temp scan image",
				zap.String("build_id", c.BuildID.String()),
				zap.String("image_ref", tempRef),
				zap.Error(err))
			continue
		}
		deleted++
		reclaimedBytes += estimatedBytes
		appLogger.Info("Deleted temp scan image", zap.String("build_id", c.BuildID.String()), zap.String("image_ref", tempRef))
	}

	appLogger.Info("Completed temp scan cleanup cycle",
		zap.Int("candidates", len(candidates)),
		zap.Int("deleted", deleted),
		zap.Int("errors", runErrors),
		zap.Int64("reclaimed_bytes_estimate", reclaimedBytes))
	return len(candidates), deleted, runErrors, reclaimedBytes, nil
}

func loadCleanupCandidates(ctx context.Context, db DBTX, retentionHours, batchSize int) ([]cleanupCandidate, error) {
	if retentionHours < 1 {
		retentionHours = 24
	}
	if batchSize < 1 {
		batchSize = 100
	}

	candidates := make([]cleanupCandidate, 0, batchSize)
	query := `
SELECT b.id AS build_id,
       b.tenant_id,
       b.completed_at,
       COALESCE(bc.metadata, '{}'::jsonb) AS metadata
FROM builds b
LEFT JOIN build_configs bc ON bc.build_id = b.id
WHERE b.completed_at IS NOT NULL
  AND b.status IN ('completed', 'failed', 'cancelled', 'success')
  AND b.completed_at < NOW() - ($1::text || ' hours')::interval
ORDER BY b.completed_at ASC
LIMIT $2
`
	if err := db.SelectContext(ctx, &candidates, query, retentionHours, batchSize); err != nil {
		return nil, fmt.Errorf("load cleanup candidates: %w", err)
	}
	return candidates, nil
}

func resolveTempScanImageRef(c cleanupCandidate, settings workerSettings) (string, bool) {
	meta := map[string]interface{}{}
	_ = json.Unmarshal(c.Metadata, &meta)

	if raw, ok := meta["enable_temp_scan_stage"]; ok {
		enabled, valid := boolFromValue(raw)
		if valid && !enabled {
			return "", false
		}
	}

	for _, key := range []string{"temp_scan_image_name", "tempScanImageName"} {
		if raw, ok := meta[key]; ok {
			if imageRef := strings.TrimSpace(fmt.Sprintf("%v", raw)); imageRef != "" && imageRef != "<nil>" {
				return imageRef, true
			}
		}
	}

	tenant := compactToken(c.TenantID.String())
	build := compactToken(c.BuildID.String())
	if tenant == "" || build == "" {
		return "", false
	}
	if strings.TrimSpace(settings.DefaultTempRegistry) == "" {
		return "", false
	}
	repo := strings.TrimSpace(settings.DefaultTempRepo)
	if repo == "" {
		repo = "quarantine-temp"
	}
	return fmt.Sprintf("%s/%s/%s/%s:scan", strings.TrimSpace(settings.DefaultTempRegistry), repo, tenant, build), true
}

func deleteRegistryTagReference(ctx context.Context, client *http.Client, settings workerSettings, imageRef string) (int64, error) {
	host, repo, tagOrDigest, err := splitImageRef(imageRef)
	if err != nil {
		return 0, err
	}

	manifestURL := fmt.Sprintf("%s://%s/v2/%s/manifests/%s", settings.RegistryScheme, host, repo, tagOrDigest)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
	}, ", "))
	applyRegistryAuth(req, settings)
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("manifest lookup returned HTTP %d", resp.StatusCode)
	}

	var manifestPayload struct {
		Config struct {
			Size int64 `json:"size"`
		} `json:"config"`
		Layers []struct {
			Size int64 `json:"size"`
		} `json:"layers"`
		Manifests []struct {
			Size int64 `json:"size"`
		} `json:"manifests"`
	}
	reclaimedBytesEstimate := int64(0)
	if err := json.NewDecoder(resp.Body).Decode(&manifestPayload); err == nil {
		reclaimedBytesEstimate += manifestPayload.Config.Size
		for _, layer := range manifestPayload.Layers {
			reclaimedBytesEstimate += layer.Size
		}
		for _, m := range manifestPayload.Manifests {
			reclaimedBytesEstimate += m.Size
		}
	}

	digest := strings.TrimSpace(resp.Header.Get("Docker-Content-Digest"))
	if digest == "" {
		if strings.Contains(tagOrDigest, "sha256:") {
			digest = tagOrDigest
		} else {
			return 0, errors.New("registry response missing Docker-Content-Digest")
		}
	}

	deleteURL := fmt.Sprintf("%s://%s/v2/%s/manifests/%s", settings.RegistryScheme, host, repo, digest)
	deleteReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return 0, err
	}
	applyRegistryAuth(deleteReq, settings)
	deleteResp, err := client.Do(deleteReq)
	if err != nil {
		return 0, err
	}
	defer deleteResp.Body.Close()
	if deleteResp.StatusCode == http.StatusNotFound {
		return 0, nil
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		return 0, fmt.Errorf("manifest delete returned HTTP %d", deleteResp.StatusCode)
	}
	return reclaimedBytesEstimate, nil
}

func applyRegistryAuth(req *http.Request, settings workerSettings) {
	if token := strings.TrimSpace(settings.RegistryBearer); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		return
	}
	if user := strings.TrimSpace(settings.RegistryUsername); user != "" {
		req.SetBasicAuth(user, settings.RegistryPassword)
	}
}

func splitImageRef(ref string) (string, string, string, error) {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return "", "", "", errors.New("empty image ref")
	}
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid image ref %q", trimmed)
	}
	host := strings.TrimSpace(parts[0])
	path := strings.TrimSpace(parts[1])
	if host == "" || path == "" {
		return "", "", "", fmt.Errorf("invalid image ref %q", trimmed)
	}
	if strings.Contains(path, "@") {
		s := strings.SplitN(path, "@", 2)
		if len(s) != 2 || strings.TrimSpace(s[0]) == "" || strings.TrimSpace(s[1]) == "" {
			return "", "", "", fmt.Errorf("invalid image ref %q", trimmed)
		}
		return host, strings.TrimSpace(s[0]), strings.TrimSpace(s[1]), nil
	}

	lastSlash := strings.LastIndex(path, "/")
	lastColon := strings.LastIndex(path, ":")
	if lastColon > lastSlash {
		repo := strings.TrimSpace(path[:lastColon])
		tag := strings.TrimSpace(path[lastColon+1:])
		if repo == "" || tag == "" {
			return "", "", "", fmt.Errorf("invalid image ref %q", trimmed)
		}
		return host, repo, tag, nil
	}
	return host, path, "latest", nil
}

func resolveSettings(ctx context.Context, repo *postgres.SystemConfigRepository, appLogger *zap.Logger) workerSettings {
	settings := workerSettings{
		Enabled:             envBool("IF_INTERNAL_REGISTRY_TEMP_CLEANUP_ENABLED", true),
		RetentionHours:      envInt("IF_INTERNAL_REGISTRY_TEMP_CLEANUP_RETENTION_HOURS", 72),
		IntervalMinutes:     envInt("IF_INTERNAL_REGISTRY_TEMP_CLEANUP_INTERVAL_MINUTES", 60),
		BatchSize:           envInt("IF_INTERNAL_REGISTRY_TEMP_CLEANUP_BATCH_SIZE", 100),
		DryRun:              envBool("IF_INTERNAL_REGISTRY_TEMP_CLEANUP_DRY_RUN", false),
		HealthPort:          envInt("IF_INTERNAL_REGISTRY_GC_WORKER_PORT", 8085),
		RegistryInsecure:    envBool("IF_INTERNAL_REGISTRY_GC_INSECURE_SKIP_TLS_VERIFY", false),
		RegistryUsername:    strings.TrimSpace(os.Getenv("IF_INTERNAL_REGISTRY_GC_USERNAME")),
		RegistryPassword:    strings.TrimSpace(os.Getenv("IF_INTERNAL_REGISTRY_GC_PASSWORD")),
		RegistryBearer:      strings.TrimSpace(os.Getenv("IF_INTERNAL_REGISTRY_GC_BEARER_TOKEN")),
		DefaultTempRegistry: strings.TrimSpace(os.Getenv("IF_INTERNAL_TEMP_SCAN_REGISTRY")),
		DefaultTempRepo:     strings.TrimSpace(os.Getenv("IF_INTERNAL_TEMP_SCAN_REPOSITORY")),
	}

	if envBool("IF_INTERNAL_REGISTRY_GC_TLS_ENABLED", false) {
		settings.RegistryScheme = "https"
	} else {
		settings.RegistryScheme = "http"
	}

	cfg, err := repo.FindByTypeAndKey(ctx, nil, systemconfig.ConfigTypeRuntimeServices, "runtime_services")
	if err != nil || cfg == nil {
		return clampSettings(settings)
	}
	runtimeCfg, err := cfg.GetRuntimeServicesConfig()
	if err != nil || runtimeCfg == nil {
		return clampSettings(settings)
	}

	if runtimeCfg.InternalRegistryTempCleanupEnabled != nil {
		settings.Enabled = *runtimeCfg.InternalRegistryTempCleanupEnabled
	}
	if runtimeCfg.InternalRegistryTempCleanupRetentionHours > 0 {
		settings.RetentionHours = runtimeCfg.InternalRegistryTempCleanupRetentionHours
	}
	if runtimeCfg.InternalRegistryTempCleanupIntervalMinutes > 0 {
		settings.IntervalMinutes = runtimeCfg.InternalRegistryTempCleanupIntervalMinutes
	}
	if runtimeCfg.InternalRegistryTempCleanupBatchSize > 0 {
		settings.BatchSize = runtimeCfg.InternalRegistryTempCleanupBatchSize
	}
	if runtimeCfg.InternalRegistryTempCleanupDryRun != nil {
		settings.DryRun = *runtimeCfg.InternalRegistryTempCleanupDryRun
	}
	if runtimeCfg.InternalRegistryGCWorkerPort > 0 {
		settings.HealthPort = runtimeCfg.InternalRegistryGCWorkerPort
	}

	appLogger.Debug("Loaded internal registry cleanup settings",
		zap.Bool("enabled", settings.Enabled),
		zap.Int("retention_hours", settings.RetentionHours),
		zap.Int("interval_minutes", settings.IntervalMinutes),
		zap.Int("batch_size", settings.BatchSize),
		zap.Bool("dry_run", settings.DryRun))

	return clampSettings(settings)
}

func clampSettings(in workerSettings) workerSettings {
	if in.RetentionHours < 1 {
		in.RetentionHours = 24
	}
	if in.IntervalMinutes < 1 {
		in.IntervalMinutes = 60
	}
	if in.BatchSize < 1 {
		in.BatchSize = 100
	}
	if in.HealthPort < 1 {
		in.HealthPort = 8085
	}
	if strings.TrimSpace(in.DefaultTempRegistry) == "" {
		in.DefaultTempRegistry = "image-factory-registry:5000"
	}
	if strings.TrimSpace(in.DefaultTempRepo) == "" {
		in.DefaultTempRepo = "quarantine-temp"
	}
	if strings.TrimSpace(in.RegistryScheme) == "" {
		in.RegistryScheme = "http"
	}
	return in
}

func envBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch raw {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func boolFromValue(v interface{}) (bool, bool) {
	switch t := v.(type) {
	case bool:
		return t, true
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "1", "true", "yes", "y", "on":
			return true, true
		case "0", "false", "no", "n", "off":
			return false, true
		}
	case float64:
		return t != 0, true
	}
	return false, false
}

func compactToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "")
	if len(value) > 12 {
		return value[:12]
	}
	return value
}
