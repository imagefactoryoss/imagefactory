package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

type TenantDashboardHandler struct {
	db     *sqlx.DB
	logger *zap.Logger
}

type tenantDashboardSummaryStats struct {
	TotalProjects       int `db:"total_projects" json:"total_projects"`
	ActiveProjects      int `db:"active_projects" json:"active_projects"`
	BuildsToday         int `db:"builds_today" json:"builds_today"`
	SuccessRate         int `json:"success_rate"`
	RunningBuilds       int `db:"running_builds" json:"running_builds"`
	QueuedBuilds        int `db:"queued_builds" json:"queued_builds"`
	CompletedBuilds     int `db:"completed_builds" json:"completed_builds"`
	FailedBuilds        int `db:"failed_builds" json:"failed_builds"`
	UnreadNotifications int `db:"unread_notifications" json:"unread_notifications"`
}

type tenantDashboardSummaryResponse struct {
	Stats         tenantDashboardSummaryStats `json:"stats"`
	LastUpdatedAt string                      `json:"last_updated_at"`
}

type tenantDashboardRecentBuild struct {
	ID            uuid.UUID `db:"id" json:"id"`
	Name          string    `json:"name"`
	ProjectName   string    `db:"project_name" json:"project_name"`
	Status        string    `db:"status" json:"status"`
	CreatedAt     string    `db:"created_at" json:"created_at"`
	DurationLabel string    `json:"duration_label"`
}

type tenantDashboardProjectActivity struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	BuildCount  int       `db:"build_count" json:"build_count"`
	LastBuildAt *string   `db:"last_build_at" json:"last_build_at,omitempty"`
}

type tenantDashboardRecentImage struct {
	ID         uuid.UUID `db:"id" json:"id"`
	Name       string    `db:"name" json:"name"`
	Visibility string    `db:"visibility" json:"visibility"`
	Tags       []string  `json:"tags"`
	CreatedAt  string    `db:"created_at" json:"created_at"`
	UpdatedAt  string    `db:"updated_at" json:"updated_at"`
}

type tenantDashboardActivityResponse struct {
	RecentBuilds       []tenantDashboardRecentBuild     `json:"recent_builds"`
	MostActiveProjects []tenantDashboardProjectActivity `json:"most_active_projects"`
	RecentImages       []tenantDashboardRecentImage     `json:"recent_images"`
	LastUpdatedAt      string                           `json:"last_updated_at"`
}

func NewTenantDashboardHandler(db *sqlx.DB, logger *zap.Logger) *TenantDashboardHandler {
	return &TenantDashboardHandler{db: db, logger: logger}
}

func (h *TenantDashboardHandler) GetTenantSummary(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil || authCtx.TenantID == uuid.Nil || authCtx.UserID == uuid.Nil {
		writeTenantDashboardError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	query := `
		SELECT
			COALESCE((SELECT COUNT(*) FROM projects p WHERE p.tenant_id = $1 AND p.deleted_at IS NULL), 0) AS total_projects,
			COALESCE((SELECT COUNT(*) FROM projects p WHERE p.tenant_id = $1 AND p.deleted_at IS NULL AND COALESCE(p.is_draft, false) = false), 0) AS active_projects,
			COALESCE((
				SELECT COUNT(*)
				FROM builds b
				WHERE b.tenant_id = $1
				  AND COALESCE(b.completed_at, b.started_at, b.created_at) >= (date_trunc('day', NOW() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC')
			), 0) AS builds_today,
			COALESCE((SELECT COUNT(*) FROM builds b WHERE b.tenant_id = $1 AND b.status = 'running'), 0) AS running_builds,
			COALESCE((SELECT COUNT(*) FROM builds b WHERE b.tenant_id = $1 AND b.status = 'queued'), 0) AS queued_builds,
			COALESCE((SELECT COUNT(*) FROM builds b WHERE b.tenant_id = $1 AND b.status = 'completed'), 0) AS completed_builds,
			COALESCE((SELECT COUNT(*) FROM builds b WHERE b.tenant_id = $1 AND b.status = 'failed'), 0) AS failed_builds,
			COALESCE((SELECT COUNT(*) FROM notifications n WHERE n.tenant_id = $1 AND n.user_id = $2 AND n.channel = 'in_app' AND n.is_read = false), 0) AS unread_notifications
	`

	var stats tenantDashboardSummaryStats
	if err := h.db.QueryRowxContext(r.Context(), query, authCtx.TenantID, authCtx.UserID).StructScan(&stats); err != nil {
		h.logger.Error("Failed to load tenant dashboard summary", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()))
		writeTenantDashboardError(w, http.StatusInternalServerError, "failed to load tenant dashboard summary")
		return
	}

	terminalBuilds := stats.CompletedBuilds + stats.FailedBuilds
	if terminalBuilds > 0 {
		stats.SuccessRate = int(float64(stats.CompletedBuilds) / float64(terminalBuilds) * 100.0)
	}

	writeTenantDashboardJSON(w, http.StatusOK, tenantDashboardSummaryResponse{
		Stats:         stats,
		LastUpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *TenantDashboardHandler) GetTenantActivity(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok || authCtx == nil || authCtx.TenantID == uuid.Nil {
		writeTenantDashboardError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	recentBuilds := make([]tenantDashboardRecentBuild, 0, 8)
	type recentBuildRow struct {
		ID          uuid.UUID  `db:"id"`
		BuildNumber int        `db:"build_number"`
		ProjectName string     `db:"project_name"`
		Status      string     `db:"status"`
		CreatedAt   time.Time  `db:"created_at"`
		StartedAt   *time.Time `db:"started_at"`
		CompletedAt *time.Time `db:"completed_at"`
	}
	buildRows := make([]recentBuildRow, 0, 8)
	buildQuery := `
		SELECT
			b.id,
			b.build_number,
			COALESCE(NULLIF(TRIM(p.name), ''), 'Unknown project') AS project_name,
			b.status,
			b.created_at,
			b.started_at,
			b.completed_at
		FROM builds b
		JOIN projects p ON p.id = b.project_id
		WHERE b.tenant_id = $1
		  AND p.deleted_at IS NULL
		ORDER BY b.created_at DESC
		LIMIT 8
	`
	if err := h.db.SelectContext(r.Context(), &buildRows, buildQuery, authCtx.TenantID); err != nil {
		h.logger.Error("Failed to load tenant dashboard recent builds", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()))
		writeTenantDashboardError(w, http.StatusInternalServerError, "failed to load tenant dashboard activity")
		return
	}
	for _, row := range buildRows {
		recentBuilds = append(recentBuilds, tenantDashboardRecentBuild{
			ID:            row.ID,
			Name:          fmt.Sprintf("Build #%d", row.BuildNumber),
			ProjectName:   row.ProjectName,
			Status:        row.Status,
			CreatedAt:     row.CreatedAt.UTC().Format(time.RFC3339),
			DurationLabel: formatDashboardDuration(row.StartedAt, row.CompletedAt),
		})
	}

	mostActiveProjects := make([]tenantDashboardProjectActivity, 0, 6)
	activityQuery := `
		SELECT
			p.id,
			COALESCE(NULLIF(TRIM(p.name), ''), 'Unknown project') AS name,
			COUNT(b.id)::int AS build_count,
			CASE
				WHEN MAX(b.created_at) IS NULL THEN NULL
				ELSE to_char(MAX(b.created_at) AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
			END AS last_build_at
		FROM projects p
		JOIN builds b ON b.project_id = p.id
		WHERE p.tenant_id = $1
		  AND p.deleted_at IS NULL
		GROUP BY p.id, p.name
		ORDER BY build_count DESC, MAX(b.created_at) DESC
		LIMIT 6
	`
	if err := h.db.SelectContext(r.Context(), &mostActiveProjects, activityQuery, authCtx.TenantID); err != nil {
		h.logger.Error("Failed to load tenant dashboard project activity", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()))
		writeTenantDashboardError(w, http.StatusInternalServerError, "failed to load tenant dashboard activity")
		return
	}

	type recentImageRow struct {
		ID         uuid.UUID       `db:"id"`
		Name       string          `db:"name"`
		Visibility string          `db:"visibility"`
		TagsRaw    json.RawMessage `db:"tags"`
		CreatedAt  time.Time       `db:"created_at"`
		UpdatedAt  time.Time       `db:"updated_at"`
	}
	imageRows := make([]recentImageRow, 0, 6)
	imageQuery := `
		SELECT
			id,
			name,
			visibility::text AS visibility,
			COALESCE(tags, '[]'::jsonb) AS tags,
			created_at,
			updated_at
		FROM catalog_images
		WHERE tenant_id = $1
		  AND archived_at IS NULL
		ORDER BY updated_at DESC NULLS LAST, created_at DESC
		LIMIT 6
	`
	if err := h.db.SelectContext(r.Context(), &imageRows, imageQuery, authCtx.TenantID); err != nil {
		h.logger.Error("Failed to load tenant dashboard recent images", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()))
		writeTenantDashboardError(w, http.StatusInternalServerError, "failed to load tenant dashboard activity")
		return
	}

	recentImages := make([]tenantDashboardRecentImage, 0, len(imageRows))
	for _, row := range imageRows {
		tags := make([]string, 0)
		if len(row.TagsRaw) > 0 {
			_ = json.Unmarshal(row.TagsRaw, &tags)
		}
		recentImages = append(recentImages, tenantDashboardRecentImage{
			ID:         row.ID,
			Name:       row.Name,
			Visibility: row.Visibility,
			Tags:       tags,
			CreatedAt:  row.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:  row.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	writeTenantDashboardJSON(w, http.StatusOK, tenantDashboardActivityResponse{
		RecentBuilds:       recentBuilds,
		MostActiveProjects: mostActiveProjects,
		RecentImages:       recentImages,
		LastUpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	})
}

func formatDashboardDuration(startedAt, completedAt *time.Time) string {
	if startedAt == nil {
		return "n/a"
	}
	end := time.Now().UTC()
	if completedAt != nil {
		end = completedAt.UTC()
	}
	delta := end.Sub(startedAt.UTC())
	if delta <= 0 {
		return "n/a"
	}
	totalSeconds := int(delta.Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	if minutes == 0 {
		return fmt.Sprintf("%ds", seconds)
	}
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

func writeTenantDashboardJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeTenantDashboardError(w http.ResponseWriter, status int, message string) {
	writeTenantDashboardJSON(w, status, map[string]string{"error": message})
}
