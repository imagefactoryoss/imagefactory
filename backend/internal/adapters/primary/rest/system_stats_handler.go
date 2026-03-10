package rest

import (
	"context"
	"net/http"

	"github.com/srikarm/image-factory/internal/domain/tenant"
	"github.com/srikarm/image-factory/internal/domain/user"
	"github.com/srikarm/image-factory/internal/infrastructure/denialtelemetry"
	"github.com/srikarm/image-factory/internal/infrastructure/releasecompliance"
	"github.com/srikarm/image-factory/internal/infrastructure/releasetelemetry"
	"go.uber.org/zap"
)

// SystemStatsHandler handles system statistics requests
type SystemStatsHandler struct {
	userService   *user.Service
	tenantService *tenant.Service
	denials       *denialtelemetry.Metrics
	releases      *releasetelemetry.Metrics
	compliance    *releasecompliance.Metrics
	eprStats      eprStatsReader
	logger        *zap.Logger
}

type eprStatsReader interface {
	GetLifecycleMetrics(ctx context.Context) (map[string]int64, error)
}

// NewSystemStatsHandler creates a new system stats handler
func NewSystemStatsHandler(userService *user.Service, tenantService *tenant.Service, denials *denialtelemetry.Metrics, releases *releasetelemetry.Metrics, compliance *releasecompliance.Metrics, logger *zap.Logger) *SystemStatsHandler {
	return &SystemStatsHandler{
		userService:   userService,
		tenantService: tenantService,
		denials:       denials,
		releases:      releases,
		compliance:    compliance,
		logger:        logger,
	}
}

func (h *SystemStatsHandler) SetEPRStatsReader(reader eprStatsReader) {
	if h == nil {
		return
	}
	h.eprStats = reader
}

// SystemStatsResponse represents system statistics
type SystemStatsResponse struct {
	TotalUsers              int                           `json:"total_users"`
	ActiveUsers             int                           `json:"active_users"`
	TotalTenants            int                           `json:"total_tenants"`
	ActiveTenants           int                           `json:"active_tenants"`
	TotalBuilds             int                           `json:"total_builds"`
	RunningBuilds           int                           `json:"running_builds"`
	TotalImages             int                           `json:"total_images"`
	StorageUsedGB           int                           `json:"storage_used_gb"`
	CriticalVulnerabilities int                           `json:"critical_vulnerabilities"`
	SystemHealth            string                        `json:"system_health"`
	Uptime                  string                        `json:"uptime"`
	LastBackup              string                        `json:"last_backup,omitempty"`
	DenialMetrics           []denialtelemetry.SnapshotRow `json:"denial_metrics,omitempty"`
	ReleaseMetrics          *releasetelemetry.Snapshot    `json:"release_metrics,omitempty"`
	ReleaseCompliance       *releasecompliance.Snapshot   `json:"release_compliance,omitempty"`
	EPRLifecycleMetrics     map[string]int64              `json:"epr_lifecycle_metrics,omitempty"`
}

// GetSystemStats handles GET /api/v1/admin/stats
func (h *SystemStatsHandler) GetSystemStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get total users
	totalUsers := 0
	if h.userService != nil {
		var err error
		totalUsers, err = h.userService.GetTotalUserCount(ctx)
		if err != nil {
			h.logger.Error("Failed to get total user count", zap.Error(err))
			totalUsers = 0
		}
	}

	// Get active users (users who logged in within last 30 days)
	activeUsers := 0
	if h.userService != nil {
		var err error
		activeUsers, err = h.userService.GetActiveUserCount(ctx, 30)
		if err != nil {
			h.logger.Error("Failed to get active user count", zap.Error(err))
			activeUsers = 0
		}
	}

	// Get total tenants
	totalTenants := 0
	if h.tenantService != nil {
		var err error
		totalTenants, err = h.tenantService.GetTotalTenantCount(ctx)
		if err != nil {
			h.logger.Error("Failed to get total tenant count", zap.Error(err))
			totalTenants = 0
		}
	}

	// Get active tenants
	activeTenants := 0
	if h.tenantService != nil {
		var err error
		activeTenants, err = h.tenantService.GetActiveTenantCount(ctx)
		if err != nil {
			h.logger.Error("Failed to get active tenant count", zap.Error(err))
			activeTenants = 0
		}
	}

	// For now, return placeholder values for build-related stats
	// TODO: Implement build service integration
	totalBuilds := 0
	runningBuilds := 0
	totalImages := 0
	storageUsedGB := 0
	criticalVulnerabilities := 0

	// System health - for now always healthy, could be enhanced with actual health checks
	systemHealth := "healthy"
	uptime := "99.9%" // TODO: Get actual uptime from health service

	stats := SystemStatsResponse{
		TotalUsers:              totalUsers,
		ActiveUsers:             activeUsers,
		TotalTenants:            totalTenants,
		ActiveTenants:           activeTenants,
		TotalBuilds:             totalBuilds,
		RunningBuilds:           runningBuilds,
		TotalImages:             totalImages,
		StorageUsedGB:           storageUsedGB,
		CriticalVulnerabilities: criticalVulnerabilities,
		SystemHealth:            systemHealth,
		Uptime:                  uptime,
	}
	if h.denials != nil {
		stats.DenialMetrics = h.denials.Snapshot()
	}
	if h.releases != nil {
		snapshot := h.releases.Snapshot()
		stats.ReleaseMetrics = &snapshot
	}
	if h.compliance != nil {
		snapshot := h.compliance.Snapshot()
		stats.ReleaseCompliance = &snapshot
	}
	if h.eprStats != nil {
		metrics, err := h.eprStats.GetLifecycleMetrics(ctx)
		if err != nil {
			h.logger.Error("Failed to get EPR lifecycle metrics", zap.Error(err))
		} else {
			stats.EPRLifecycleMetrics = metrics
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encodeJSON(w, stats)
}
