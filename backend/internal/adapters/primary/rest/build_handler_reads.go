package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/domain/infrastructure/connectors"
	"github.com/srikarm/image-factory/internal/domain/workflow"
	k8sinfra "github.com/srikarm/image-factory/internal/infrastructure/kubernetes"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
)

func (h *BuildHandler) GetBuild(w http.ResponseWriter, r *http.Request) {
	// Check RBAC permission
	authCtx := h.checkPermission(r)
	if authCtx == nil {
		h.logger.Warn("Unauthorized build read attempt")
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Build ID is required")
		return
	}

	buildID, err := uuid.Parse(id)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid build ID format")
		return
	}

	b, err := h.buildService.GetBuild(r.Context(), buildID)
	if err != nil {
		h.logger.Error("Failed to get build", zap.Error(err), zap.String("build_id", buildID.String()))
		h.respondError(w, http.StatusInternalServerError, "Failed to get build")
		return
	}

	if b == nil {
		h.respondError(w, http.StatusNotFound, "Build not found")
		return
	}

	response := h.buildToResponse(b)
	h.respondJSON(w, http.StatusOK, response)
}

// ListBuilds handles GET /builds
func (h *BuildHandler) ListBuilds(w http.ResponseWriter, r *http.Request) {
	// Check RBAC permission
	authCtx := h.checkPermission(r)
	if authCtx == nil {
		h.logger.Warn("Unauthorized build list attempt")
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Parse query parameters
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		// Also check for camelCase variant for frontend compatibility
		tenantIDStr = r.URL.Query().Get("tenantId")
	}
	projectIDStr := r.URL.Query().Get("projectId")
	if projectIDStr == "" {
		// Also check for snake_case variant for API compatibility
		projectIDStr = r.URL.Query().Get("project_id")
	}
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	pageStr := r.URL.Query().Get("page")
	allTenantsRequested := false
	switch strings.TrimSpace(strings.ToLower(r.URL.Query().Get("all_tenants"))) {
	case "true", "1", "yes":
		allTenantsRequested = true
	}
	allTenants := authCtx.IsSystemAdmin && allTenantsRequested

	// Parse project ID if provided
	var projectID uuid.UUID
	var hasProjectID bool
	if projectIDStr != "" {
		var err error
		projectID, err = uuid.Parse(projectIDStr)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "Invalid project_id format")
			return
		}
		hasProjectID = true
	}

	// Get tenant ID from query parameter or auth context - but not required if projectId is provided
	var tenantID uuid.UUID
	if tenantIDStr != "" {
		var err error
		tenantID, err = uuid.Parse(tenantIDStr)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "Invalid tenant_id format")
			return
		}
		if tenantID == uuid.Nil {
			h.respondError(w, http.StatusBadRequest, "Invalid tenant_id")
			return
		}
	} else if !hasProjectID && !allTenants { // Only require tenantId if no projectId is provided and all-tenant scope is not requested
		// Use tenant ID from auth context (explicit tenant context required)
		tenantID = authCtx.TenantID
	}
	// If projectId is provided and no tenantId is specified, we'll validate project's tenant later

	limit := 20 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Handle pagination (support both offset and page parameters)
	var offset int
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	} else if pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			offset = (page - 1) * limit
		}
	}

	// Fetch builds with appropriate filtering
	var builds []*build.Build
	var totalCount int
	var err error
	if hasProjectID {
		// Validate that the project belongs to the current tenant
		// This ensures that even if someone manipulates the projectId parameter,
		// they won't get access to builds from projects outside their tenant
		var projectTenantID uuid.UUID

		// Check if user is system admin using proper method (not by tenant ID)
		// Get user's system admin status from RBAC
		var isSystemAdmin bool
		if rbacRepo, ok := h.rbacRepo.(*postgres.RBACRepository); ok {
			isSystemAdmin, _ = rbacRepo.IsUserSystemAdmin(r.Context(), authCtx.UserID)
		}

		if !isSystemAdmin {
			// Get project details from project service
			type Project interface {
				ID() uuid.UUID
				TenantID() uuid.UUID
			}

			// Try to get project service from build service
			// Note: This assumes the build service has a reference to the project service
			// In real scenario, we should have the project service injected directly
			projectServiceField := reflect.ValueOf(h.buildService).Elem().FieldByName("projectService")
			if projectServiceField.IsValid() && !projectServiceField.IsNil() {
				getProjectMethod := projectServiceField.MethodByName("GetProject")
				if getProjectMethod.IsValid() {
					// Call GetProject method
					args := []reflect.Value{
						reflect.ValueOf(r.Context()),
						reflect.ValueOf(projectID),
					}
					results := getProjectMethod.Call(args)

					if len(results) > 0 && !results[0].IsNil() {
						project := results[0].Interface()
						tenantIDMethod := reflect.ValueOf(project).MethodByName("TenantID")
						if tenantIDMethod.IsValid() {
							projectTenantID = tenantIDMethod.Call(nil)[0].Interface().(uuid.UUID)

							// Check if project's tenant matches user's tenant
							if projectTenantID != authCtx.TenantID {
								h.logger.Warn("Attempt to access project from different tenant",
									zap.String("user_tenant_id", authCtx.TenantID.String()),
									zap.String("project_tenant_id", projectTenantID.String()),
									zap.String("project_id", projectID.String()))
								h.respondError(w, http.StatusForbidden, "Permission denied: project belongs to a different tenant")
								return
							}
						}
					}
				}
			}
		}

		// Fetch builds by project
		builds, err = h.buildService.ListBuildsByProject(r.Context(), projectID, limit, offset)
		if err != nil {
			h.logger.Error("Failed to list builds by project", zap.Error(err), zap.String("project_id", projectID.String()))
			h.respondError(w, http.StatusInternalServerError, "Failed to list builds")
			return
		}
		totalCount, err = h.buildService.GetBuildCountByProject(r.Context(), projectID)
	} else if allTenants {
		builds, err = h.buildService.ListBuildsAllTenants(r.Context(), limit, offset)
		if err != nil {
			h.logger.Error("Failed to list builds across all tenants", zap.Error(err))
			h.respondError(w, http.StatusInternalServerError, "Failed to list builds")
			return
		}
		totalCount, err = h.buildService.GetBuildCountAllTenants(r.Context())
	} else {
		// Fetch builds by tenant
		builds, err = h.buildService.ListBuilds(r.Context(), tenantID, limit, offset)
		if err != nil {
			h.logger.Error("Failed to list builds", zap.Error(err), zap.String("tenant_id", tenantID.String()))
			h.respondError(w, http.StatusInternalServerError, "Failed to list builds")
			return
		}
		totalCount, err = h.buildService.GetBuildCount(r.Context(), tenantID)
	}
	if err != nil {
		h.logger.Warn("Failed to get build count", zap.Error(err))
		totalCount = len(builds) // fallback
	}

	response := BuildListResponse{
		Builds:     make([]BuildResponse, len(builds)),
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	}

	for i, b := range builds {
		response.Builds[i] = h.buildToResponse(b)
	}

	h.respondJSON(w, http.StatusOK, response)
}

// StartBuild handles POST /builds/{id}/start
func (h *BuildHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	// Check RBAC permission
	authCtx := h.checkPermission(r)
	if authCtx == nil {
		h.logger.Warn("Unauthorized build logs read attempt")
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Build ID is required")
		return
	}

	buildID, err := uuid.Parse(id)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid build ID format")
		return
	}

	// Verify build exists
	b, err := h.fetchBuild(r.Context(), buildID)
	if err != nil || b == nil {
		h.respondError(w, http.StatusNotFound, "Build not found")
		return
	}

	// Parse pagination parameters
	limit := 100 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	logFilter, err := parseBuildLogsFilter(r.URL.Query())
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	executionID, resolved, err := h.resolveRequestedExecutionID(r.Context(), buildID, r.URL.Query().Get("execution_id"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !resolved {
		response := GetLogsResponse{
			BuildID: buildID.String(),
			Logs:    []LogEntry{},
			Total:   0,
			HasMore: false,
		}
		h.respondJSON(w, http.StatusOK, response)
		return
	}

	executionLogs, _, err := h.buildExecutionService.GetLogs(r.Context(), executionID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get execution logs", zap.Error(err), zap.String("execution_id", executionID.String()))
		h.respondError(w, http.StatusInternalServerError, "Failed to get execution logs")
		return
	}

	logs := make([]LogEntry, len(executionLogs))
	foundTekton := false
	for i, log := range executionLogs {
		var metadata map[string]interface{}
		if len(log.Metadata) > 0 {
			_ = json.Unmarshal(log.Metadata, &metadata)
			if src, ok := metadata["source"].(string); ok && src == "tekton" {
				foundTekton = true
			}
		}
		logs[i] = LogEntry{
			Timestamp: log.Timestamp.Format(time.RFC3339),
			Level:     string(log.Level),
			Message:   log.Message,
			Metadata:  metadata,
		}
	}

	// Fallback: if there are no persisted Tekton step logs for this execution but
	// execution metadata references a PipelineRun, try to fetch step logs directly
	// from the infrastructure provider (best-effort, does not fail the request).
	needsTektonLogs := logFilter.source == buildLogSourceAll || logFilter.source == buildLogSourceTekton
	if needsTektonLogs && !foundTekton && h.infraService != nil {
		execObj, execErr := h.buildExecutionService.GetExecution(r.Context(), executionID)
		if execErr == nil && execObj != nil && len(execObj.Metadata) > 0 {
			var execMeta map[string]json.RawMessage
			if err := json.Unmarshal(execObj.Metadata, &execMeta); err == nil {
				if tekRaw, ok := execMeta["tekton"]; ok && len(tekRaw) > 0 {
					var tek struct {
						Namespace   string `json:"namespace"`
						PipelineRun string `json:"pipeline_run"`
						ProviderID  string `json:"provider_id"`
					}
					if err := json.Unmarshal(tekRaw, &tek); err == nil && tek.Namespace != "" && tek.PipelineRun != "" {
						provList, pErr := h.infraService.GetAvailableProviders(r.Context(), b.TenantID())
						if pErr == nil {
							var selected *infrastructure.Provider
							if tek.ProviderID != "" {
								if pid, perr := uuid.Parse(tek.ProviderID); perr == nil {
									for _, p := range provList {
										if p != nil && p.ID == pid {
											selected = p
											break
										}
									}
								}
							}
							if selected == nil && b.InfrastructureProviderID() != nil {
								for _, p := range provList {
									if p != nil && p.ID == *b.InfrastructureProviderID() {
										selected = p
										break
									}
								}
							}

							if selected != nil {
								restCfg, rErr := connectors.BuildRESTConfigFromProviderConfig(selected.Config)
								if rErr == nil {
									k8sClient, kerr := k8sclient.NewForConfig(restCfg)
									tektonClient, terr := tektonclient.NewForConfig(restCfg)
									if kerr == nil && terr == nil {
										pipelineMgr := k8sinfra.NewKubernetesPipelineManager(k8sClient, tektonClient, h.logger)
										logsMap, gErr := pipelineMgr.GetLogs(r.Context(), tek.Namespace, tek.PipelineRun)
										if gErr == nil && len(logsMap) > 0 {
											for key, content := range logsMap {
												parts := strings.SplitN(key, "/", 2)
												taskRunName := parts[0]
												containerName := ""
												if len(parts) > 1 {
													containerName = parts[1]
												}

												taskRunObj, trErr := tektonClient.TektonV1().TaskRuns(tek.Namespace).Get(r.Context(), taskRunName, metav1.GetOptions{})
												podName := ""
												isFailed := false
												if trErr == nil {
													podName = strings.TrimSpace(taskRunObj.Status.PodName)
													for _, c := range taskRunObj.Status.Conditions {
														if strings.EqualFold(string(c.Type), "Succeeded") && c.Status == "False" {
															isFailed = true
															break
														}
													}
												}

												meta := map[string]interface{}{
													"source":       "tekton",
													"pipeline_run": tek.PipelineRun,
													"task_run":     taskRunName,
													"step":         containerName,
												}
												if podName != "" {
													meta["pod"] = podName
												}

												mdata, _ := json.Marshal(meta)
												lvl := build.LogInfo
												if isFailed {
													lvl = build.LogError
												}
												_ = h.buildExecutionService.AddLog(r.Context(), executionID, lvl, content, mdata)

												logs = append(logs, LogEntry{
													Timestamp: time.Now().UTC().Format(time.RFC3339),
													Level:     string(lvl),
													Message:   content,
													Metadata:  meta,
												})
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	filteredLogs := applyBuildLogsFilter(logs, logFilter)
	filteredTotal := len(filteredLogs)

	// Truncate to requested limit if necessary
	if len(filteredLogs) > limit {
		filteredLogs = filteredLogs[:limit]
	}

	hasMore := int64(offset+limit) < int64(filteredTotal)
	response := GetLogsResponse{
		BuildID:     buildID.String(),
		ExecutionID: executionID.String(),
		Logs:        filteredLogs,
		Total:       filteredTotal,
		HasMore:     hasMore,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetStatus handles GET /api/v1/builds/{id}/status
func (h *BuildHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Build ID is required")
		return
	}

	buildID, err := uuid.Parse(id)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid build ID format")
		return
	}

	b, err := h.buildService.GetBuild(r.Context(), buildID)
	if err != nil || b == nil {
		h.respondError(w, http.StatusNotFound, "Build not found")
		return
	}

	// Calculate duration if started and completed
	var duration *int64
	if b.StartedAt() != nil && b.CompletedAt() != nil {
		d := int64(b.CompletedAt().Sub(*b.StartedAt()).Seconds())
		duration = &d
	}

	// Get progress (0-100)
	progress := ProgressInfo{
		Current: 0,
		Total:   100,
		Stage:   string(b.Status()),
	}

	response := GetStatusResponse{
		BuildID:  buildID.String(),
		Status:   string(b.Status()),
		Progress: progress,
		Duration: duration,
	}

	if b.StartedAt() != nil {
		response.StartedAt = b.StartedAt().Format("2006-01-02T15:04:05Z07:00")
	}
	if b.CompletedAt() != nil {
		response.CompletedAt = b.CompletedAt().Format("2006-01-02T15:04:05Z07:00")
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetWorkflow returns workflow steps for a build.
// Source of truth: workflow subject_type=build, subject_id={build_id}.
func (h *BuildHandler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	if h.checkPermission(r) == nil {
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	buildIDStr := chi.URLParam(r, "id")
	buildID, err := uuid.Parse(buildIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid build ID")
		return
	}

	if h.workflowRepo == nil {
		h.respondJSON(w, http.StatusOK, BuildWorkflowResponse{Steps: []BuildWorkflowStepResponse{}})
		return
	}

	// Primary: build-subject workflow.
	instance, steps, err := h.workflowRepo.GetInstanceWithStepsBySubject(r.Context(), "build", buildID)
	if err == nil && instance != nil {
		h.respondJSON(w, http.StatusOK, h.workflowToResponse(instance, steps, nil))
		return
	}

	h.respondJSON(w, http.StatusOK, BuildWorkflowResponse{Steps: []BuildWorkflowStepResponse{}})
}

// GetExecutions handles GET /api/v1/builds/{id}/executions
func (h *BuildHandler) GetExecutions(w http.ResponseWriter, r *http.Request) {
	if h.checkPermission(r) == nil {
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Build ID is required")
		return
	}

	buildID, err := uuid.Parse(id)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid build ID format")
		return
	}

	// Verify build exists
	b, err := h.buildService.GetBuild(r.Context(), buildID)
	if err != nil || b == nil {
		h.respondError(w, http.StatusNotFound, "Build not found")
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, convErr := strconv.Atoi(limitStr); convErr == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, convErr := strconv.Atoi(offsetStr); convErr == nil && o >= 0 {
			offset = o
		}
	}

	executions, total, err := h.fetchExecutions(r.Context(), buildID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get build executions", zap.Error(err), zap.String("build_id", buildID.String()))
		h.respondError(w, http.StatusInternalServerError, "Failed to get build executions")
		return
	}

	items := h.buildExecutionsToResponse(executions)

	h.respondJSON(w, http.StatusOK, BuildExecutionListResponse{
		BuildID:    buildID.String(),
		Executions: items,
		Total:      total,
		Limit:      limit,
		Offset:     offset,
	})
}

// GetTrace handles GET /api/v1/builds/{id}/trace.
func (h *BuildHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
	authCtx := h.checkPermission(r)
	if authCtx == nil {
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Build ID is required")
		return
	}

	buildID, err := uuid.Parse(id)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid build ID format")
		return
	}

	trace, statusCode, errMsg := h.buildTraceResponse(r.Context(), buildID, r.URL.Query().Get("execution_id"), authCtx)
	if trace == nil {
		h.respondError(w, statusCode, errMsg)
		return
	}
	h.respondJSON(w, http.StatusOK, trace)
}

// GetTraceExport handles GET /api/v1/builds/{id}/trace/export.
func (h *BuildHandler) GetTraceExport(w http.ResponseWriter, r *http.Request) {
	authCtx := h.checkPermission(r)
	if authCtx == nil {
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	buildIDStr := chi.URLParam(r, "id")
	buildID, err := uuid.Parse(buildIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid build ID format")
		return
	}

	trace, statusCode, errMsg := h.buildTraceResponse(r.Context(), buildID, r.URL.Query().Get("execution_id"), authCtx)
	if trace == nil {
		h.respondError(w, statusCode, errMsg)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"build-trace-%s.json\"", buildID.String()))
	w.WriteHeader(http.StatusOK)
	if err := encodeJSON(w, trace); err != nil {
		h.logger.Error("Failed to encode trace export response", zap.Error(err), zap.String("build_id", buildID.String()))
	}
}

func (h *BuildHandler) resolveRequestedExecutionID(ctx context.Context, buildID uuid.UUID, executionIDParam string) (uuid.UUID, bool, error) {
	if strings.TrimSpace(executionIDParam) != "" {
		executionID, err := uuid.Parse(strings.TrimSpace(executionIDParam))
		if err != nil {
			return uuid.Nil, false, fmt.Errorf("invalid execution_id format")
		}

		execution, err := h.buildExecutionService.GetExecution(ctx, executionID)
		if err != nil || execution == nil {
			return uuid.Nil, false, fmt.Errorf("execution not found")
		}
		if execution.BuildID != buildID {
			return uuid.Nil, false, fmt.Errorf("execution does not belong to this build")
		}
		return executionID, true, nil
	}

	executions, _, err := h.fetchExecutions(ctx, buildID, 1, 0)
	if err != nil {
		h.logger.Error("Failed to get build executions", zap.Error(err), zap.String("build_id", buildID.String()))
		return uuid.Nil, false, fmt.Errorf("failed to get build executions")
	}
	if len(executions) == 0 {
		return uuid.Nil, false, nil
	}
	return executions[0].ID, true, nil
}

func (h *BuildHandler) buildExecutionsToResponse(executions []build.BuildExecution) []BuildExecutionAttemptResponse {
	items := make([]BuildExecutionAttemptResponse, 0, len(executions))
	for _, execution := range executions {
		item := BuildExecutionAttemptResponse{
			ID:           execution.ID.String(),
			Status:       string(execution.Status),
			CreatedAt:    execution.CreatedAt.Format(time.RFC3339),
			ErrorMessage: execution.ErrorMessage,
		}
		if execution.StartedAt != nil {
			item.StartedAt = execution.StartedAt.Format(time.RFC3339)
		}
		if execution.CompletedAt != nil {
			item.CompletedAt = execution.CompletedAt.Format(time.RFC3339)
		}
		item.DurationSeconds = execution.DurationSeconds
		items = append(items, item)
	}
	return items
}

func (h *BuildHandler) buildTraceResponse(ctx context.Context, buildID uuid.UUID, executionIDParam string, authCtx *middleware.AuthContext) (*BuildTraceResponse, int, string) {
	b, err := h.fetchBuild(ctx, buildID)
	if err != nil || b == nil {
		return nil, http.StatusNotFound, "Build not found"
	}

	executions, _, err := h.fetchExecutions(ctx, buildID, 50, 0)
	if err != nil {
		h.logger.Error("Failed to get build executions", zap.Error(err), zap.String("build_id", buildID.String()))
		return nil, http.StatusInternalServerError, "Failed to get build executions"
	}
	items := h.buildExecutionsToResponse(executions)

	selectedExecutionID, hasExecution, resolveErr := h.resolveRequestedExecutionID(ctx, buildID, executionIDParam)
	if resolveErr != nil {
		return nil, http.StatusBadRequest, resolveErr.Error()
	}

	workflowResp := BuildWorkflowResponse{Steps: []BuildWorkflowStepResponse{}}
	if h.workflowRepo != nil {
		instance, steps, workflowErr := h.workflowRepo.GetInstanceWithStepsBySubject(ctx, "build", buildID)
		if workflowErr == nil && instance != nil {
			if hasExecution {
				workflowResp = h.workflowToResponse(instance, steps, &selectedExecutionID)
			} else {
				workflowResp = h.workflowToResponse(instance, steps, nil)
			}
		}
	}

	trace := &BuildTraceResponse{
		Build:       h.buildToResponse(b),
		Executions:  items,
		Workflow:    workflowResp,
		Diagnostics: buildTraceDiagnosticsFromBuild(b),
	}
	if hasExecution {
		trace.SelectedExecutionID = selectedExecutionID.String()
	}
	trace.Correlation = &BuildTraceCorrelation{
		WorkflowInstanceID: workflowResp.InstanceID,
		ExecutionID:        trace.SelectedExecutionID,
		ActiveStepKey:      activeWorkflowStepKey(workflowResp.Steps),
	}

	if authCtx != nil && authCtx.IsSystemAdmin && h.processStatusProvider != nil {
		trace.Runtime = map[string]BuildTraceRuntimeComponent{}
		for _, component := range []string{"dispatcher", "workflow_orchestrator"} {
			if status, ok := h.processStatusProvider.GetStatus(component); ok {
				trace.Runtime[component] = BuildTraceRuntimeComponent{
					Enabled:      status.Enabled,
					Running:      status.Running,
					LastActivity: status.LastActivity,
					Message:      status.Message,
				}
			}
		}
	}

	return trace, 0, ""
}

func activeWorkflowStepKey(steps []BuildWorkflowStepResponse) string {
	if len(steps) == 0 {
		return ""
	}
	priorities := []string{"running", "blocked", "failed", "pending"}
	for _, status := range priorities {
		for _, step := range steps {
			if step.Status == status {
				return step.StepKey
			}
		}
	}
	return steps[len(steps)-1].StepKey
}

func buildTraceDiagnosticsFromBuild(b *build.Build) *BuildTraceDiagnostics {
	if b == nil {
		return nil
	}
	manifest := b.Manifest()
	metadata := manifest.Metadata

	path := traceMetadataString(metadata, "repo_config_path")
	ref := traceMetadataString(metadata, "repo_config_ref")
	stage := traceMetadataString(metadata, "repo_config_error_stage")
	errorAt := traceMetadataString(metadata, "repo_config_error_at")
	errorMsg := traceMetadataString(metadata, "repo_config_error")
	if errorMsg == "" {
		errorMsg = strings.TrimSpace(b.ErrorMessage())
	}
	applied, appliedSet := traceMetadataBool(metadata, "repo_config_applied")

	if !appliedSet && path == "" && ref == "" && stage == "" && errorAt == "" && !strings.Contains(strings.ToLower(errorMsg), "repo build config") {
		return nil
	}

	repoDiag := &BuildTraceRepoConfigDiagnostics{
		Applied:   applied,
		Path:      path,
		Ref:       ref,
		Stage:     stage,
		Error:     errorMsg,
		UpdatedAt: errorAt,
	}
	if repoDiag.Error != "" {
		if classified := classifyBuildLifecycleError(errors.New(repoDiag.Error)); classified != nil {
			repoDiag.ErrorCode = classified.code
		}
	}
	return &BuildTraceDiagnostics{RepoConfig: repoDiag}
}

func traceMetadataString(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func traceMetadataBool(metadata map[string]interface{}, key string) (bool, bool) {
	if metadata == nil {
		return false, false
	}
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return false, false
	}
	switch v := raw.(type) {
	case bool:
		return v, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

func (h *BuildHandler) workflowToResponse(instance *workflow.Instance, steps []workflow.Step, executionID *uuid.UUID) BuildWorkflowResponse {
	response := BuildWorkflowResponse{
		InstanceID: instance.ID.String(),
		Status:     string(instance.Status),
	}
	if executionID != nil {
		response.ExecutionID = executionID.String()
	}
	for _, step := range steps {
		response.Steps = append(response.Steps, BuildWorkflowStepResponse{
			StepKey:     step.StepKey,
			Status:      string(step.Status),
			Attempts:    step.Attempts,
			LastError:   step.LastError,
			StartedAt:   step.StartedAt,
			CompletedAt: step.CompletedAt,
			CreatedAt:   step.CreatedAt,
			UpdatedAt:   step.UpdatedAt,
		})
	}
	return response
}

func (h *BuildHandler) fetchBuild(ctx context.Context, buildID uuid.UUID) (*build.Build, error) {
	if h.readBuildFn != nil {
		return h.readBuildFn(ctx, buildID)
	}
	return h.buildService.GetBuild(ctx, buildID)
}

func (h *BuildHandler) fetchExecutions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]build.BuildExecution, int64, error) {
	if h.readExecutionsFn != nil {
		return h.readExecutionsFn(ctx, buildID, limit, offset)
	}
	return h.buildExecutionService.GetBuildExecutions(ctx, buildID, limit, offset)
}

// RetryBuild handles POST /api/v1/builds/{id}/retry
func (h *BuildHandler) GetBuildContextSuggestions(w http.ResponseWriter, r *http.Request) {
	authCtx := h.checkPermission(r)
	if authCtx == nil {
		h.respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	projectIDStr := chi.URLParam(r, "projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid project ID format")
		return
	}
	if h.projectContextLookup == nil {
		h.respondError(w, http.StatusNotImplemented, "Project context lookup not configured")
		return
	}

	projectTenantID, repoURL, defaultRef, err := h.projectContextLookup(r.Context(), projectID)
	if err != nil {
		h.logger.Error("Failed to load project context for build context suggestions", zap.Error(err), zap.String("project_id", projectID.String()))
		h.respondError(w, http.StatusInternalServerError, "Failed to load project context")
		return
	}
	if projectTenantID != authCtx.TenantID {
		h.respondError(w, http.StatusForbidden, "Project not found in selected tenant context")
		return
	}
	if strings.TrimSpace(repoURL) == "" {
		h.respondError(w, http.StatusBadRequest, "Project repository URL is required")
		return
	}

	ref := strings.TrimSpace(r.URL.Query().Get("ref"))
	if ref == "" {
		ref = strings.TrimSpace(defaultRef)
	}

	var gitAuthSecret map[string][]byte
	if h.projectGitAuthLookup != nil {
		secretData, secretErr := h.projectGitAuthLookup(r.Context(), projectID)
		if secretErr != nil {
			h.logger.Warn("Failed to resolve project repository auth for repo inspection; proceeding without credentials", zap.Error(secretErr), zap.String("project_id", projectID.String()))
		} else {
			gitAuthSecret = secretData
		}
	}

	contexts, dockerfiles, scanErr := scanRepositoryBuildStructure(r.Context(), repoURL, ref, gitAuthSecret)
	if scanErr != nil {
		h.logger.Warn("Failed to inspect repository structure for build context suggestions", zap.Error(scanErr), zap.String("project_id", projectID.String()))
		h.respondJSON(w, http.StatusOK, BuildContextSuggestionsResponse{
			ProjectID: projectID.String(),
			RepoURL:   repoURL,
			Ref:       ref,
			Contexts: []BuildContextPathSuggestion{
				{Path: ".", Reason: "Repository root", Score: 10},
			},
			Dockerfiles: []BuildDockerfilePathSuggestion{},
			Note:        "Could not inspect repository structure. You can still enter Build Context and Dockerfile path manually.",
		})
		return
	}

	h.respondJSON(w, http.StatusOK, BuildContextSuggestionsResponse{
		ProjectID:   projectID.String(),
		RepoURL:     repoURL,
		Ref:         ref,
		Contexts:    contexts,
		Dockerfiles: dockerfiles,
	})
}

func (h *BuildHandler) buildToResponse(b *build.Build) BuildResponse {
	response := BuildResponse{
		ID:        b.ID().String(),
		TenantID:  b.TenantID().String(),
		ProjectID: b.ProjectID().String(),
		Name:      b.Manifest().Name,
		Type:      string(b.Manifest().Type),
		Status:    string(b.Status()),
		ErrorMsg:  b.ErrorMessage(),
		CreatedAt: b.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: b.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	if b.StartedAt() != nil {
		response.StartedAt = b.StartedAt().Format("2006-01-02T15:04:05Z07:00")
	}

	if b.CompletedAt() != nil {
		response.CompletedAt = b.CompletedAt().Format("2006-01-02T15:04:05Z07:00")
	}

	// Include manifest for detailed view (consider security implications)
	response.Manifest = b.Manifest()

	// Include result if available
	if result := b.Result(); result != nil {
		response.Result = *result
	}

	return response
}

func (h *BuildHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *BuildHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

func (h *BuildHandler) respondBuildHTTPError(w http.ResponseWriter, err *buildHTTPError) {
	if err == nil {
		h.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if err.code == "" && len(err.details) == 0 {
		h.respondError(w, err.status, err.message)
		return
	}
	payload := map[string]interface{}{
		"error": err.message,
	}
	if err.code != "" {
		payload["code"] = err.code
	}
	if len(err.details) > 0 {
		payload["details"] = err.details
	}
	h.respondJSON(w, err.status, payload)
}
