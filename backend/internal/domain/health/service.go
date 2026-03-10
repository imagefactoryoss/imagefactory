package health

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/infrastructure/config"
	"go.uber.org/zap"
)

// Service provides health check functionality
type Service struct {
	db          *sqlx.DB
	config      *config.Config
	logger      *zap.Logger
	startupTime time.Time
	initialized bool
}

// NewService creates a new health check service
func NewService(db *sqlx.DB, config *config.Config, logger *zap.Logger) *Service {
	return &Service{
		db:          db,
		config:      config,
		logger:      logger,
		startupTime: time.Now(),
		initialized: true,
	}
}

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status      string           `json:"status"`
	Service     string           `json:"service"`
	Version     string           `json:"version"`
	Build       BuildInfo        `json:"build"`
	Hostname    string           `json:"hostname"`
	Uptime      time.Duration    `json:"uptime"`
	StartupTime time.Time        `json:"startup_time"`
	Checks      map[string]Check `json:"checks"`
	SystemInfo  SystemInfo       `json:"system_info"`
	Deployment  DeploymentInfo   `json:"deployment"`
}

type BuildInfo struct {
	Revision      string `json:"revision"`
	RevisionShort string `json:"revision_short"`
	Time          string `json:"time"`
	Modified      string `json:"modified"`
	Source        string `json:"source"`
	Fingerprint   string `json:"fingerprint"`
}

var imageTagCommitPattern = regexp.MustCompile(`[0-9a-f]{7,40}`)

// Check represents an individual health check
type Check struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// SystemInfo contains system information
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
}

// DeploymentInfo captures release + component metadata for operational visibility.
type DeploymentInfo struct {
	Environment      string                       `json:"environment"`
	ReleaseName      string                       `json:"release_name"`
	ReleaseNamespace string                       `json:"release_namespace"`
	Components       map[string]ComponentMetadata `json:"components"`
	RuntimeEndpoints map[string]string            `json:"runtime_endpoints,omitempty"`
}

// ComponentMetadata captures image coordinates and optional revision for a component.
type ComponentMetadata struct {
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	Image      string `json:"image,omitempty"`
	Revision   string `json:"revision,omitempty"`
}

// Check performs a comprehensive health check
func (s *Service) Check(ctx context.Context) (*HealthStatus, error) {
	uptime := time.Since(s.startupTime)

	status := &HealthStatus{
		Status:      "healthy",
		Service:     "image-factory",
		Version:     s.config.Server.Version,
		Build:       readBuildInfo(),
		StartupTime: s.startupTime,
		Uptime:      uptime,
		Checks:      make(map[string]Check),
	}
	status.Deployment = readDeploymentInfo(status.Build)

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	status.Hostname = hostname

	// System info
	status.SystemInfo = SystemInfo{
		GoVersion:    runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
	}

	// Database check
	if err := s.checkDatabase(ctx); err != nil {
		status.Checks["database"] = Check{Status: "unhealthy", Message: err.Error()}
		status.Status = "unhealthy"
	} else {
		status.Checks["database"] = Check{Status: "healthy", Message: "Connected"}
	}

	return status, nil
}

// checkDatabase checks database connectivity
func (s *Service) checkDatabase(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Ready checks if the service is fully initialized and ready for traffic
// Returns true only if service is fully initialized and database is accessible
func (s *Service) Ready(ctx context.Context) bool {
	if !s.initialized {
		return false
	}

	// Check database connectivity
	if err := s.checkDatabase(ctx); err != nil {
		return false
	}

	return true
}

// IsHealthy returns true if the overall service health is healthy
func (s *Service) IsHealthy(ctx context.Context) bool {
	status, err := s.Check(ctx)
	if err != nil {
		return false
	}
	return status.Status == "healthy"
}

// GetTemplateData returns health data formatted for email templates
func (s *Service) GetTemplateData(ctx context.Context) (map[string]interface{}, error) {
	health, err := s.Check(ctx)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"hostname":      health.Hostname,
		"version":       s.config.Server.Version,
		"startup_time":  health.StartupTime.UTC().Format(time.RFC3339),
		"uptime":        health.Uptime.String(),
		"status":        health.Status,
		"go_version":    health.SystemInfo.GoVersion,
		"num_cpu":       health.SystemInfo.NumCPU,
		"num_goroutine": health.SystemInfo.NumGoroutine,
	}

	// Add check statuses
	for name, check := range health.Checks {
		data[name+"_status"] = check.Status
		if check.Status == "healthy" {
			data[name+"_status"] = "✅ " + check.Message
		} else {
			data[name+"_status"] = "❌ " + check.Message
		}
	}

	// Add configuration-based values
	data["worker_count"] = getEnvInt("WORKER_COUNT", 5)
	data["poll_interval"] = getEnvString("POLL_INTERVAL", "1s")
	data["max_retries"] = getEnvInt("MAX_RETRIES", 3)
	data["retry_delay"] = getEnvString("RETRY_DELAY", "1s")
	data["retry_max_delay"] = getEnvString("RETRY_MAX_DELAY", "60s")
	data["shutdown_timeout"] = getEnvString("SHUTDOWN_TIMEOUT", "30s")
	data["health_port"] = s.config.Server.Port

	// Add service status checks (these could be expanded to real checks)
	data["queue_status"] = "✅ Operational" // TODO: Add actual queue health check
	data["smtp_status"] = "✅ Connected"    // TODO: Add actual SMTP health check
	data["cache_status"] = "✅ Operational" // TODO: Add actual cache health check

	return data, nil
}

// getEnvString gets a string value from environment with fallback
func getEnvString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvInt gets an int value from environment with fallback
func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return fallback
}

func readBuildInfo() BuildInfo {
	info := BuildInfo{
		Revision: strings.TrimSpace(os.Getenv("IF_BUILD_COMMIT")),
		Time:     strings.TrimSpace(os.Getenv("IF_BUILD_TIME")),
		Modified: strings.TrimSpace(os.Getenv("IF_BUILD_DIRTY")),
		Source:   "env",
	}
	if info.Revision == "" {
		info.Revision = "unknown"
	}
	if info.Time == "" {
		info.Time = "unknown"
	}
	if info.Modified == "" {
		info.Modified = "unknown"
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if strings.TrimSpace(s.Value) != "" {
					info.Revision = strings.TrimSpace(s.Value)
				}
			case "vcs.time":
				if strings.TrimSpace(s.Value) != "" {
					info.Time = strings.TrimSpace(s.Value)
				}
			case "vcs.modified":
				if strings.TrimSpace(s.Value) != "" {
					info.Modified = strings.TrimSpace(s.Value)
				}
			}
		}
		info.Source = "go_build_info"
	}

	if info.Revision == "unknown" {
		if tagCommit := parseCommitFromImageTag(strings.TrimSpace(os.Getenv("IF_BACKEND_IMAGE_TAG"))); tagCommit != "" {
			info.Revision = tagCommit
			info.Source = "image_tag"
		}
	}
	info.RevisionShort = info.Revision
	if len(info.RevisionShort) > 12 {
		info.RevisionShort = info.RevisionShort[:12]
	}
	sum := sha1.Sum([]byte(strings.Join([]string{info.Revision, info.Time, info.Modified}, "|")))
	info.Fingerprint = fmt.Sprintf("%x", sum[:6])
	return info
}

func parseCommitFromImageTag(tag string) string {
	normalized := strings.TrimSpace(strings.ToLower(tag))
	if normalized == "" {
		return ""
	}
	return imageTagCommitPattern.FindString(normalized)
}

func readDeploymentInfo(build BuildInfo) DeploymentInfo {
	components := map[string]ComponentMetadata{
		"backend": componentMetadataFromEnv("IF_BACKEND_IMAGE_REPOSITORY", "IF_BACKEND_IMAGE_TAG"),
		"frontend": componentMetadataFromEnv(
			"IF_FRONTEND_IMAGE_REPOSITORY",
			"IF_FRONTEND_IMAGE_TAG",
		),
		"dispatcher": componentMetadataFromEnv("IF_DISPATCHER_IMAGE_REPOSITORY", "IF_DISPATCHER_IMAGE_TAG"),
		"notification_worker": componentMetadataFromEnv(
			"IF_NOTIFICATION_WORKER_IMAGE_REPOSITORY",
			"IF_NOTIFICATION_WORKER_IMAGE_TAG",
		),
		"email_worker": componentMetadataFromEnv("IF_EMAIL_WORKER_IMAGE_REPOSITORY", "IF_EMAIL_WORKER_IMAGE_TAG"),
		"internal_registry_gc_worker": componentMetadataFromEnv(
			"IF_INTERNAL_REGISTRY_GC_WORKER_IMAGE_REPOSITORY",
			"IF_INTERNAL_REGISTRY_GC_WORKER_IMAGE_TAG",
		),
		"external_tenant_service": componentMetadataFromEnv(
			"IF_EXTERNAL_TENANT_SERVICE_IMAGE_REPOSITORY",
			"IF_EXTERNAL_TENANT_SERVICE_IMAGE_TAG",
		),
	}

	// Current process is backend; expose backend revision from Go build info.
	if backend := components["backend"]; backend.Revision == "" {
		backend.Revision = build.Revision
		components["backend"] = backend
	}

	return DeploymentInfo{
		Environment:      strings.TrimSpace(os.Getenv("IF_SERVER_ENVIRONMENT")),
		ReleaseName:      strings.TrimSpace(os.Getenv("IF_HELM_RELEASE_NAME")),
		ReleaseNamespace: strings.TrimSpace(os.Getenv("IF_HELM_RELEASE_NAMESPACE")),
		Components:       components,
		RuntimeEndpoints: map[string]string{
			"dispatcher_url":             strings.TrimSpace(os.Getenv("IF_DISPATCHER_URL")),
			"email_worker_url":           strings.TrimSpace(os.Getenv("IF_EMAIL_WORKER_URL")),
			"notification_worker_url":    strings.TrimSpace(os.Getenv("IF_NOTIFICATION_WORKER_URL")),
			"external_tenant_service_url": strings.TrimSpace(os.Getenv("EXTERNAL_TENANT_SERVICE_URL")),
		},
	}
}

func componentMetadataFromEnv(repositoryKey, tagKey string) ComponentMetadata {
	repository := strings.TrimSpace(os.Getenv(repositoryKey))
	tag := strings.TrimSpace(os.Getenv(tagKey))
	image := strings.TrimSpace(strings.Trim(repository+":"+tag, ":"))
	if repository == "" && tag == "" {
		image = ""
	}
	return ComponentMetadata{
		Repository: repository,
		Tag:        tag,
		Image:      image,
	}
}
