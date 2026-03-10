package rest

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// BuildAnalyticsHandler handles admin build analytics endpoints
type BuildAnalyticsHandler struct {
	db *sql.DB
}

// NewBuildAnalyticsHandler creates a new analytics handler
func NewBuildAnalyticsHandler(db *sql.DB) *BuildAnalyticsHandler {
	return &BuildAnalyticsHandler{db: db}
}

// Analytics response structures

type BuildAnalyticsResponse struct {
	TotalBuilds            int64     `json:"total_builds"`
	RunningBuilds          int64     `json:"running_builds"`
	CompletedBuilds        int64     `json:"completed_builds"`
	FailedBuilds           int64     `json:"failed_builds"`
	SuccessRate            float64   `json:"success_rate"`
	AverageDurationSeconds int64     `json:"average_duration_seconds"`
	QueueDepth             int64     `json:"queue_depth"`
	LastUpdated            time.Time `json:"last_updated"`
}

type PerformanceTrendPoint struct {
	Date                    string  `json:"date"`
	AverageDurationSeconds  int64   `json:"average_seconds"`
	SuccessRate             float64 `json:"success_rate"`
	AverageQueueTimeSeconds int64   `json:"average_queue_time_seconds"`
	BuildCount              int64   `json:"count"`
}

type PerformanceTrendsResponse struct {
	DurationTrend []PerformanceTrendPoint `json:"duration_trend"`
	SuccessTrend  []PerformanceTrendPoint `json:"success_trend"`
	QueueTrend    []PerformanceTrendPoint `json:"queue_trend"`
}

type SlowestBuild struct {
	ID              string    `json:"id"`
	ProjectID       string    `json:"project_id"`
	ProjectName     string    `json:"project_name"`
	DurationSeconds int64     `json:"duration_seconds"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

type FailureReason struct {
	Reason     string  `json:"reason"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

type ProjectFailureRate struct {
	ProjectID    string  `json:"project_id"`
	ProjectName  string  `json:"project_name"`
	TotalBuilds  int64   `json:"total_builds"`
	FailedBuilds int64   `json:"failed_builds"`
	FailureRate  float64 `json:"failure_rate"`
}

type FailureAnalysisResponse struct {
	SlowestBuilds        []SlowestBuild       `json:"slowest_builds"`
	FailureReasons       []FailureReason      `json:"failure_reasons"`
	FailureRateByProject []ProjectFailureRate `json:"failure_rate_by_project"`
}

// GetAnalytics handles GET /api/v1/admin/builds/analytics
// Returns current build metrics summary
func (h *BuildAnalyticsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	scopeTenantID, allTenants, status, message := resolveTenantScopeFromRequest(r, authCtx, true)
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	var response BuildAnalyticsResponse

	query := `
		SELECT
			COALESCE(SUM(total_builds), 0) AS total_builds,
			COALESCE(SUM(running_builds), 0) AS running_builds,
			COALESCE(SUM(completed_builds), 0) AS completed_builds,
			COALESCE(SUM(failed_builds), 0) AS failed_builds,
			COALESCE(
				ROUND(
					(COALESCE(SUM(completed_builds), 0)::NUMERIC / NULLIF(COALESCE(SUM(total_builds), 0), 0) * 100)::NUMERIC,
					2
				),
				0
			) AS success_rate,
			COALESCE(
				ROUND(
					(COALESCE(SUM(average_duration_seconds * total_builds), 0)::NUMERIC / NULLIF(COALESCE(SUM(total_builds), 0), 0))::NUMERIC,
					0
				)::INTEGER,
				0
			) AS average_duration_seconds,
			COALESCE(SUM(queue_depth), 0) AS queue_depth,
			COALESCE(MAX(last_updated), NOW()) AS last_updated
		FROM v_build_analytics
	`
	args := []interface{}{}
	if !allTenants {
		query += ` WHERE tenant_id = $1`
		args = append(args, scopeTenantID)
	}

	row := h.db.QueryRow(query, args...)

	err := row.Scan(
		&response.TotalBuilds,
		&response.RunningBuilds,
		&response.CompletedBuilds,
		&response.FailedBuilds,
		&response.SuccessRate,
		&response.AverageDurationSeconds,
		&response.QueueDepth,
		&response.LastUpdated,
	)

	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Failed to fetch analytics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPerformance handles GET /api/v1/admin/builds/performance
// Returns 7-day performance trends
func (h *BuildAnalyticsHandler) GetPerformance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	scopeTenantID, allTenants, status, message := resolveTenantScopeFromRequest(r, authCtx, true)
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	query := `
		SELECT
			trend_date,
			COALESCE(
				ROUND(
					(SUM(average_duration_seconds * build_count)::NUMERIC / NULLIF(SUM(build_count), 0))::NUMERIC,
					0
				)::INTEGER,
				0
			) AS average_duration_seconds,
			COALESCE(
				ROUND(
					(SUM(success_rate * build_count)::NUMERIC / NULLIF(SUM(build_count), 0))::NUMERIC,
					2
				),
				0
			) AS success_rate,
			COALESCE(
				ROUND(
					(SUM(average_queue_time_seconds * build_count)::NUMERIC / NULLIF(SUM(build_count), 0))::NUMERIC,
					0
				)::INTEGER,
				0
			) AS average_queue_time_seconds,
			COALESCE(SUM(build_count), 0) AS build_count
		FROM v_build_performance_trends
	`
	args := []interface{}{}
	if allTenants {
		query += ` GROUP BY trend_date`
	} else {
		query += ` WHERE tenant_id = $1 GROUP BY trend_date`
		args = append(args, scopeTenantID)
	}
	query += ` ORDER BY trend_date DESC LIMIT 7`

	rows, err := h.db.Query(query, args...)

	if err != nil {
		http.Error(w, "Failed to fetch performance trends", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var trends []PerformanceTrendPoint
	for rows.Next() {
		var trend PerformanceTrendPoint
		var date time.Time

		err := rows.Scan(
			&date,
			&trend.AverageDurationSeconds,
			&trend.SuccessRate,
			&trend.AverageQueueTimeSeconds,
			&trend.BuildCount,
		)

		if err != nil {
			continue
		}

		trend.Date = date.Format("2006-01-02")
		trends = append(trends, trend)
	}

	// Reverse to show oldest first
	for i, j := 0, len(trends)-1; i < j; i, j = i+1, j-1 {
		trends[i], trends[j] = trends[j], trends[i]
	}

	response := PerformanceTrendsResponse{
		DurationTrend: trends,
		SuccessTrend:  trends,
		QueueTrend:    trends,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetFailures handles GET /api/v1/admin/builds/failures
// Returns failure analysis data
func (h *BuildAnalyticsHandler) GetFailures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	scopeTenantID, allTenants, status, message := resolveTenantScopeFromRequest(r, authCtx, true)
	if status != 0 {
		http.Error(w, message, status)
		return
	}

	response := FailureAnalysisResponse{
		SlowestBuilds:        []SlowestBuild{},
		FailureReasons:       []FailureReason{},
		FailureRateByProject: []ProjectFailureRate{},
	}

	slowestQuery := `
		SELECT
			id,
			project_id,
			COALESCE(project_name, 'Unknown'),
			COALESCE(duration_seconds, 0),
			status,
			created_at
		FROM v_build_slowest_builds
	`
	slowestArgs := []interface{}{}
	if !allTenants {
		slowestQuery += ` WHERE tenant_id = $1`
		slowestArgs = append(slowestArgs, scopeTenantID)
	}
	slowestQuery += ` LIMIT 10`
	slowestRows, err := h.db.Query(slowestQuery, slowestArgs...)

	if err == nil {
		defer slowestRows.Close()
		for slowestRows.Next() {
			var build SlowestBuild
			err := slowestRows.Scan(
				&build.ID,
				&build.ProjectID,
				&build.ProjectName,
				&build.DurationSeconds,
				&build.Status,
				&build.CreatedAt,
			)
			if err == nil {
				response.SlowestBuilds = append(response.SlowestBuilds, build)
			}
		}
	}

	reasonsQuery := `
		SELECT
			failure_reason,
			SUM(failure_count) AS failure_count,
			ROUND((SUM(failure_count)::NUMERIC / NULLIF(SUM(SUM(failure_count)) OVER (), 0) * 100)::NUMERIC, 2) AS percentage
		FROM v_build_failure_reasons
	`
	reasonsArgs := []interface{}{}
	if !allTenants {
		reasonsQuery += ` WHERE tenant_id = $1`
		reasonsArgs = append(reasonsArgs, scopeTenantID)
	}
	reasonsQuery += `
		GROUP BY failure_reason
		ORDER BY failure_count DESC
		LIMIT 10
	`
	reasonsRows, err := h.db.Query(reasonsQuery, reasonsArgs...)

	if err == nil {
		defer reasonsRows.Close()
		for reasonsRows.Next() {
			var reason FailureReason
			err := reasonsRows.Scan(
				&reason.Reason,
				&reason.Count,
				&reason.Percentage,
			)
			if err == nil {
				response.FailureReasons = append(response.FailureReasons, reason)
			}
		}
	}

	projectQuery := `
		SELECT
			id,
			project_name,
			total_builds,
			failed_builds,
			failure_rate
		FROM v_build_failure_rate_by_project
		WHERE total_builds > 0
	`
	projectArgs := []interface{}{}
	if !allTenants {
		projectQuery += ` AND tenant_id = $1`
		projectArgs = append(projectArgs, scopeTenantID)
	}
	projectQuery += ` LIMIT 20`
	projectRows, err := h.db.Query(projectQuery, projectArgs...)

	if err == nil {
		defer projectRows.Close()
		for projectRows.Next() {
			var project ProjectFailureRate
			err := projectRows.Scan(
				&project.ProjectID,
				&project.ProjectName,
				&project.TotalBuilds,
				&project.FailedBuilds,
				&project.FailureRate,
			)
			if err == nil {
				response.FailureRateByProject = append(response.FailureRateByProject, project)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
