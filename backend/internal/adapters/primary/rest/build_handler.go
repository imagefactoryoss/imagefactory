package rest

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appbuild "github.com/srikarm/image-factory/internal/application/build"
	"github.com/srikarm/image-factory/internal/application/runtimehealth"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/domain/infrastructure/connectors"
	"github.com/srikarm/image-factory/internal/domain/workflow"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// BuildHandler handles build-related HTTP requests
type BuildHandler struct {
	buildService          *build.Service
	buildAppService       *appbuild.Service
	buildExecutionService build.BuildExecutionService
	workflowRepo          workflow.Repository
	auditService          *audit.Service
	infraService          InfrastructureService
	logger                *zap.Logger
	rbacRepo              interface{} // RBAC repository for system admin checks
	processStatusProvider runtimehealth.Provider
	projectContextLookup  func(ctx context.Context, projectID uuid.UUID) (tenantID uuid.UUID, gitRepo, gitBranch string, err error)
	projectRepoAuthLookup func(ctx context.Context, projectID uuid.UUID) (bool, error)
	projectGitAuthLookup  func(ctx context.Context, projectID uuid.UUID) (map[string][]byte, error)
	publicRepoProbe       func(ctx context.Context, repoURL string) (bool, error)
	readBuildFn           func(ctx context.Context, buildID uuid.UUID) (*build.Build, error)
	readExecutionsFn      func(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]build.BuildExecution, int64, error)
	quarantineAdmission   QuarantineArtifactAdmissionChecker
}

// QuarantineArtifactAdmissionChecker validates whether a tenant can consume a
// quarantine artifact reference (for build/deploy admission guardrails).
type QuarantineArtifactAdmissionChecker interface {
	IsArtifactRefReleased(ctx context.Context, tenantID uuid.UUID, imageRef string) (bool, error)
}

type QuarantineArtifactReleaseStateLookup interface {
	GetArtifactReleaseStateByRef(ctx context.Context, tenantID uuid.UUID, imageRef string) (string, error)
}

// InfrastructureService defines the subset of infrastructure operations needed by the handler.
type InfrastructureService interface {
	GetAvailableProviders(ctx context.Context, tenantID uuid.UUID) ([]*infrastructure.Provider, error)
	GetTenantNamespacePrepareStatus(ctx context.Context, providerID, tenantID uuid.UUID) (*infrastructure.ProviderTenantNamespacePrepare, error)
}

// NewBuildHandler creates a new build handler.
// BuildApplicationService is mandatory; handler startup fails fast if omitted.
func NewBuildHandler(buildService *build.Service, buildAppService *appbuild.Service, buildExecutionService build.BuildExecutionService, workflowRepo workflow.Repository, auditService *audit.Service, infraService InfrastructureService, logger *zap.Logger) *BuildHandler {
	if buildAppService == nil {
		panic("build application service is required")
	}

	h := &BuildHandler{
		buildService:          buildService,
		buildAppService:       buildAppService,
		buildExecutionService: buildExecutionService,
		workflowRepo:          workflowRepo,
		auditService:          auditService,
		infraService:          infraService,
		logger:                logger,
	}

	h.buildAppService.SetCreateBuildPreflight(func(ctx context.Context, tenantID, projectID uuid.UUID, manifest *build.BuildManifest) error {
		if infraErr := h.validateQuarantineArtifactAdmission(ctx, tenantID, manifest); infraErr != nil {
			return &buildHTTPError{status: infraErr.status, message: infraErr.message, code: infraErr.code, details: infraErr.details}
		}
		if infraErr := h.hydrateManifestProjectMetadata(ctx, tenantID, projectID, manifest); infraErr != nil {
			return &buildHTTPError{status: infraErr.status, message: infraErr.message, code: infraErr.code, details: infraErr.details}
		}
		if infraErr := h.validateTektonRepositoryAuth(ctx, projectID, manifest); infraErr != nil {
			return &buildHTTPError{status: infraErr.status, message: infraErr.message, code: infraErr.code, details: infraErr.details}
		}
		if infraErr := h.ensureKubernetesBuildHasProvider(ctx, tenantID, manifest); infraErr != nil {
			return &buildHTTPError{status: infraErr.status, message: infraErr.message, code: infraErr.code, details: infraErr.details}
		}
		if infraErr := h.validateInfrastructureSelection(ctx, tenantID, *manifest); infraErr != nil {
			return &buildHTTPError{status: infraErr.status, message: infraErr.message, code: infraErr.code, details: infraErr.details}
		}
		return nil
	})
	h.buildAppService.SetRetryBuildPreflight(func(ctx context.Context, b *build.Build) error {
		manifest := b.Manifest()
		if infraErr := h.validateQuarantineArtifactAdmission(ctx, b.TenantID(), &manifest); infraErr != nil {
			return &buildHTTPError{status: infraErr.status, message: infraErr.message, code: infraErr.code, details: infraErr.details}
		}
		if infraErr := h.ensureKubernetesBuildHasProvider(ctx, b.TenantID(), &manifest); infraErr != nil {
			return &buildHTTPError{status: infraErr.status, message: infraErr.message, code: infraErr.code, details: infraErr.details}
		}
		if infraErr := h.validateInfrastructureSelection(ctx, b.TenantID(), manifest); infraErr != nil {
			return &buildHTTPError{status: infraErr.status, message: infraErr.message, code: infraErr.code, details: infraErr.details}
		}
		return nil
	})
	return h
}

func (h *BuildHandler) SetProjectContextLookup(lookup func(ctx context.Context, projectID uuid.UUID) (tenantID uuid.UUID, gitRepo, gitBranch string, err error)) {
	h.projectContextLookup = lookup
}

func (h *BuildHandler) SetProjectRepoAuthLookup(lookup func(ctx context.Context, projectID uuid.UUID) (bool, error)) {
	h.projectRepoAuthLookup = lookup
}

func (h *BuildHandler) SetProjectGitAuthLookup(lookup func(ctx context.Context, projectID uuid.UUID) (map[string][]byte, error)) {
	h.projectGitAuthLookup = lookup
}

func (h *BuildHandler) SetPublicRepoProbe(probe func(ctx context.Context, repoURL string) (bool, error)) {
	h.publicRepoProbe = probe
}

func (h *BuildHandler) SetQuarantineArtifactAdmissionChecker(checker QuarantineArtifactAdmissionChecker) {
	h.quarantineAdmission = checker
}

// SetRBACService sets the RBAC service for permission checks
// This is called from the router after handler initialization
func (h *BuildHandler) SetRBACService(rbacService interface{}) {
	h.rbacRepo = rbacService
}

// SetProcessStatusProvider sets optional runtime process status provider for trace endpoint.
func (h *BuildHandler) SetProcessStatusProvider(processStatusProvider runtimehealth.Provider) {
	h.processStatusProvider = processStatusProvider
}

// SetTraceReadOverrides injects read hooks for tests.
func (h *BuildHandler) SetTraceReadOverrides(
	readBuildFn func(ctx context.Context, buildID uuid.UUID) (*build.Build, error),
	readExecutionsFn func(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]build.BuildExecution, int64, error),
) {
	h.readBuildFn = readBuildFn
	h.readExecutionsFn = readExecutionsFn
}

// checkPermission verifies that the request has the required permission
// Returns the auth context if user is authorized, nil otherwise
func (h *BuildHandler) checkPermission(r *http.Request) *middleware.AuthContext {
	// Extract auth context (set by auth middleware)
	authCtx, ok := r.Context().Value("auth").(*middleware.AuthContext)
	if !ok {
		h.logger.Debug("no auth context in request")
		return nil
	}

	if authCtx == nil || authCtx.UserID.String() == "" {
		h.logger.Debug("invalid auth context")
		return nil
	}

	return authCtx
}

// CreateBuildRequest represents the request payload for creating a build
type CreateBuildRequest struct {
	TenantID  string              `json:"tenant_id" validate:"required,uuid"`
	ProjectID string              `json:"project_id" validate:"required,uuid"`
	Manifest  build.BuildManifest `json:"manifest" validate:"required"`
}

// CreateBuildResponse represents the response for build creation
type CreateBuildResponse struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// BuildResponse represents a build in API responses
type BuildResponse struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenant_id"`
	ProjectID   string      `json:"project_id"`
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Status      string      `json:"status"`
	Manifest    interface{} `json:"manifest,omitempty"`
	Result      interface{} `json:"result,omitempty"`
	ErrorMsg    string      `json:"error_message,omitempty"`
	CreatedAt   string      `json:"created_at"`
	StartedAt   string      `json:"started_at,omitempty"`
	CompletedAt string      `json:"completed_at,omitempty"`
	UpdatedAt   string      `json:"updated_at"`
}

// BuildListResponse represents a paginated list of builds
type BuildListResponse struct {
	Builds     []BuildResponse `json:"builds"`
	TotalCount int             `json:"total_count"`
	Limit      int             `json:"limit"`
	Offset     int             `json:"offset"`
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// GetLogsResponse represents the response for getting build logs
type GetLogsResponse struct {
	BuildID     string     `json:"build_id"`
	ExecutionID string     `json:"execution_id,omitempty"`
	Logs        []LogEntry `json:"logs"`
	Total       int        `json:"total"`
	HasMore     bool       `json:"has_more"`
}

type buildLogSourceFilter string

const (
	buildLogSourceAll       buildLogSourceFilter = "all"
	buildLogSourceTekton    buildLogSourceFilter = "tekton"
	buildLogSourceLifecycle buildLogSourceFilter = "lifecycle"
)

type buildLogsFilter struct {
	source   buildLogSourceFilter
	minLevel build.LogLevel
}

// ProgressInfo represents build execution progress
type ProgressInfo struct {
	Current int    `json:"current"`
	Total   int    `json:"total"`
	Stage   string `json:"stage"`
}

// GetStatusResponse represents the response for getting build status
type GetStatusResponse struct {
	BuildID     string       `json:"build_id"`
	ExecutionID string       `json:"execution_id,omitempty"`
	Status      string       `json:"status"`
	Progress    ProgressInfo `json:"progress"`
	StartedAt   string       `json:"started_at,omitempty"`
	CompletedAt string       `json:"completed_at,omitempty"`
	Duration    *int64       `json:"duration,omitempty"` // seconds
}

type BuildWorkflowStepResponse struct {
	StepKey     string     `json:"step_key"`
	Status      string     `json:"status"`
	Attempts    int        `json:"attempts"`
	LastError   *string    `json:"last_error,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type BuildWorkflowResponse struct {
	InstanceID  string                      `json:"instance_id,omitempty"`
	ExecutionID string                      `json:"execution_id,omitempty"`
	Status      string                      `json:"status,omitempty"`
	Steps       []BuildWorkflowStepResponse `json:"steps"`
}

type BuildExecutionAttemptResponse struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
	StartedAt       string `json:"started_at,omitempty"`
	CompletedAt     string `json:"completed_at,omitempty"`
	DurationSeconds *int   `json:"duration_seconds,omitempty"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

type BuildExecutionListResponse struct {
	BuildID    string                          `json:"build_id"`
	Executions []BuildExecutionAttemptResponse `json:"executions"`
	Total      int64                           `json:"total"`
	Limit      int                             `json:"limit"`
	Offset     int                             `json:"offset"`
}

type BuildContextPathSuggestion struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
	Score  int    `json:"score"`
}

type BuildDockerfilePathSuggestion struct {
	Path    string `json:"path"`
	Context string `json:"context"`
	Score   int    `json:"score"`
}

type BuildContextSuggestionsResponse struct {
	ProjectID   string                          `json:"project_id"`
	RepoURL     string                          `json:"repo_url"`
	Ref         string                          `json:"ref,omitempty"`
	Contexts    []BuildContextPathSuggestion    `json:"contexts"`
	Dockerfiles []BuildDockerfilePathSuggestion `json:"dockerfiles"`
	Note        string                          `json:"note,omitempty"`
}

type BuildTraceRuntimeComponent struct {
	Enabled      bool      `json:"enabled"`
	Running      bool      `json:"running"`
	LastActivity time.Time `json:"last_activity,omitempty"`
	Message      string    `json:"message,omitempty"`
}

type BuildTraceResponse struct {
	Build               BuildResponse                         `json:"build"`
	Executions          []BuildExecutionAttemptResponse       `json:"executions"`
	SelectedExecutionID string                                `json:"selected_execution_id,omitempty"`
	Workflow            BuildWorkflowResponse                 `json:"workflow"`
	Diagnostics         *BuildTraceDiagnostics                `json:"diagnostics,omitempty"`
	Runtime             map[string]BuildTraceRuntimeComponent `json:"runtime,omitempty"`
	Correlation         *BuildTraceCorrelation                `json:"correlation,omitempty"`
}

type BuildTraceCorrelation struct {
	WorkflowInstanceID string `json:"workflow_instance_id,omitempty"`
	ExecutionID        string `json:"execution_id,omitempty"`
	ActiveStepKey      string `json:"active_step_key,omitempty"`
}

type BuildTraceDiagnostics struct {
	RepoConfig *BuildTraceRepoConfigDiagnostics `json:"repo_config,omitempty"`
}

type BuildTraceRepoConfigDiagnostics struct {
	Applied   bool   `json:"applied"`
	Path      string `json:"path,omitempty"`
	Ref       string `json:"ref,omitempty"`
	Stage     string `json:"stage,omitempty"`
	Error     string `json:"error,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// CreateBuild handles POST /builds
type infraValidationError struct {
	status  int
	message string
	code    string
	details map[string]interface{}
}

type buildHTTPError struct {
	status  int
	message string
	code    string
	details map[string]interface{}
}

func (e *buildHTTPError) Error() string { return e.message }

func (h *BuildHandler) ensureKubernetesBuildHasProvider(ctx context.Context, tenantID uuid.UUID, manifest *build.BuildManifest) *infraValidationError {
	if manifest == nil {
		return &infraValidationError{status: http.StatusBadRequest, message: "build manifest is required"}
	}
	if manifest.InfrastructureType != "kubernetes" {
		return nil
	}
	if manifest.InfrastructureProviderID != nil {
		return nil
	}
	if h.infraService == nil {
		// Can't auto-select a global provider without infra service; keep existing validation
		// behavior (explicit provider required).
		return nil
	}

	providers, err := h.infraService.GetAvailableProviders(ctx, tenantID)
	if err != nil {
		h.logger.Error("Failed to fetch infrastructure providers", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return &infraValidationError{status: http.StatusInternalServerError, message: "failed to select default infrastructure provider"}
	}

	globalProviders := make([]*infrastructure.Provider, 0, len(providers))
	for i := range providers {
		provider := providers[i]
		if provider == nil || !provider.IsGlobal {
			continue
		}
		if !isKubernetesCapableProviderType(provider.ProviderType) {
			continue
		}
		if !isTektonEnabled(provider) {
			continue
		}
		if _, err := connectors.BuildRESTConfigFromProviderConfig(provider.Config); err != nil {
			continue
		}
		globalProviders = append(globalProviders, provider)
	}

	sort.SliceStable(globalProviders, func(i, j int) bool {
		return globalProviders[i].CreatedAt.Before(globalProviders[j].CreatedAt)
	})

	var selected *infrastructure.Provider
	if len(globalProviders) > 0 {
		selected = globalProviders[0]
	}
	if selected == nil {
		return &infraValidationError{status: http.StatusBadRequest, message: "infrastructure_provider_id is required for kubernetes builds (no global tekton-ready provider available)"}
	}

	providerID := selected.ID
	manifest.InfrastructureProviderID = &providerID
	return nil
}

func (h *BuildHandler) hydrateManifestProjectMetadata(ctx context.Context, tenantID, projectID uuid.UUID, manifest *build.BuildManifest) *infraValidationError {
	if manifest == nil {
		return &infraValidationError{status: http.StatusBadRequest, message: "build manifest is required"}
	}
	if h.projectContextLookup == nil {
		return nil
	}

	projectTenantID, projectGitRepo, projectGitBranch, err := h.projectContextLookup(ctx, projectID)
	if err != nil {
		h.logger.Error("Failed to load project context for build", zap.Error(err), zap.String("project_id", projectID.String()))
		return &infraValidationError{status: http.StatusInternalServerError, message: "failed to resolve project configuration"}
	}
	if projectTenantID != tenantID {
		return &infraValidationError{status: http.StatusForbidden, message: "project belongs to a different tenant"}
	}

	if manifest.Metadata == nil {
		manifest.Metadata = map[string]interface{}{}
	}

	if !hasManifestStringMetadata(manifest.Metadata, "git_url", "gitUrl", "repo_url", "repoUrl", "repository_url", "repositoryUrl") && strings.TrimSpace(projectGitRepo) != "" {
		manifest.Metadata["git_url"] = projectGitRepo
	}
	if !hasManifestStringMetadata(manifest.Metadata, "git_branch", "gitBranch", "branch") && strings.TrimSpace(projectGitBranch) != "" {
		manifest.Metadata["git_branch"] = projectGitBranch
	}

	if requiresTektonGitContext(manifest) &&
		!hasManifestStringMetadata(manifest.Metadata, "git_url", "gitUrl", "repo_url", "repoUrl", "repository_url", "repositoryUrl") {
		return &infraValidationError{status: http.StatusBadRequest, message: "project git repository URL is required for Tekton builds"}
	}

	return nil
}

func (h *BuildHandler) validateTektonRepositoryAuth(ctx context.Context, projectID uuid.UUID, manifest *build.BuildManifest) *infraValidationError {
	if manifest == nil {
		return &infraValidationError{status: http.StatusBadRequest, message: "build manifest is required"}
	}
	if !requiresTektonGitContext(manifest) {
		return nil
	}
	repoURL := manifestStringMetadata(manifest.Metadata, "git_url", "gitUrl", "repo_url", "repoUrl", "repository_url", "repositoryUrl")
	if strings.TrimSpace(repoURL) == "" {
		// In production wiring project context lookup is configured and should hydrate git_url.
		// Keep lightweight/unit handler scenarios backward compatible.
		if h.projectContextLookup == nil {
			return nil
		}
		return &infraValidationError{status: http.StatusBadRequest, message: "project git repository URL is required for Tekton builds"}
	}
	if h.projectRepoAuthLookup == nil {
		// If lookup is unavailable, avoid introducing false-negative blocks.
		return nil
	}

	hasAuth, err := h.projectRepoAuthLookup(ctx, projectID)
	if err != nil {
		h.logger.Error("Failed to resolve project repository auth", zap.Error(err), zap.String("project_id", projectID.String()))
		return &infraValidationError{status: http.StatusInternalServerError, message: "failed to validate project repository authentication"}
	}
	if hasAuth {
		return nil
	}

	if h.publicRepoProbe == nil {
		return &infraValidationError{
			status:  http.StatusBadRequest,
			message: "project repository authentication is required for Tekton builds",
		}
	}

	probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	publiclyReachable, probeErr := h.publicRepoProbe(probeCtx, repoURL)
	if probeErr != nil {
		h.logger.Warn("Anonymous repository probe failed",
			zap.Error(probeErr),
			zap.String("project_id", projectID.String()))
	}
	if publiclyReachable {
		return nil
	}
	return &infraValidationError{
		status:  http.StatusBadRequest,
		message: "project repository authentication is required for private or inaccessible repositories",
	}
}

func isKubernetesCapableProviderType(providerType infrastructure.ProviderType) bool {
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

func (h *BuildHandler) validateInfrastructureSelection(ctx context.Context, tenantID uuid.UUID, manifest build.BuildManifest) *infraValidationError {
	if manifest.InfrastructureType == "" || manifest.InfrastructureType == "auto" {
		return &infraValidationError{status: http.StatusBadRequest, message: "infrastructure selection is required"}
	}
	if manifest.InfrastructureProviderID == nil {
		return &infraValidationError{status: http.StatusBadRequest, message: "infrastructure_provider_id is required when infrastructure_type is set"}
	}
	if h.infraService == nil {
		return &infraValidationError{status: http.StatusInternalServerError, message: "infrastructure service not configured"}
	}

	providers, err := h.infraService.GetAvailableProviders(ctx, tenantID)
	if err != nil {
		h.logger.Error("Failed to fetch infrastructure providers", zap.Error(err), zap.String("tenant_id", tenantID.String()))
		return &infraValidationError{status: http.StatusInternalServerError, message: "failed to validate infrastructure provider"}
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
		return &infraValidationError{status: http.StatusForbidden, message: "selected infrastructure provider is not available for this tenant"}
	}

	if manifest.InfrastructureType == "kubernetes" {
		if !isKubernetesCapableProviderType(selected.ProviderType) {
			return &infraValidationError{status: http.StatusBadRequest, message: "selected infrastructure provider does not match infrastructure_type"}
		}
		if !isTektonEnabled(selected) {
			return &infraValidationError{status: http.StatusBadRequest, message: "selected infrastructure provider must have tekton_enabled=true"}
		}
		if _, err := connectors.BuildRESTConfigFromProviderConfig(selected.Config); err != nil {
			return &infraValidationError{status: http.StatusBadRequest, message: "selected infrastructure provider has invalid kubernetes configuration"}
		}

		// Managed mode uses per-tenant namespaces with namespace-scoped Tekton assets + RBAC.
		// Fail fast during build creation when per-tenant provisioning has not completed yet.
		if (selected.BootstrapMode == "" || selected.BootstrapMode == "image_factory_managed") && h.infraService != nil {
			prepare, err := h.infraService.GetTenantNamespacePrepareStatus(ctx, selected.ID, tenantID)
			if err != nil {
				h.logger.Error("Failed to check tenant namespace provisioning status",
					zap.Error(err),
					zap.String("tenant_id", tenantID.String()),
					zap.String("provider_id", selected.ID.String()),
				)
				return &infraValidationError{status: http.StatusInternalServerError, message: "failed to validate tenant namespace provisioning status"}
			}
			expectedNS := ""
			if tenantID != uuid.Nil {
				expectedNS = fmt.Sprintf("image-factory-%s", tenantID.String()[:8])
			}
			if prepare == nil {
				msg := "selected infrastructure provider is not prepared for this tenant (namespace not provisioned yet)"
				if expectedNS != "" {
					msg = fmt.Sprintf("%s: expected namespace %s", msg, expectedNS)
				}
				return &infraValidationError{
					status:  http.StatusConflict,
					message: msg + ". Ask a system administrator to prepare the tenant namespace on the provider details page.",
					code:    "tenant_namespace_not_prepared",
					details: map[string]interface{}{
						"provider_id":    selected.ID.String(),
						"tenant_id":      tenantID.String(),
						"prepare_status": "missing",
						"namespace":      expectedNS,
					},
				}
			}
			if prepare.Status != infrastructure.ProviderTenantNamespacePrepareSucceeded {
				msg := fmt.Sprintf("selected infrastructure provider tenant namespace is not ready (status=%s)", prepare.Status)
				if expectedNS != "" {
					msg = fmt.Sprintf("%s: namespace %s", msg, prepare.Namespace)
				}
				if prepare.ErrorMessage != nil && strings.TrimSpace(*prepare.ErrorMessage) != "" {
					msg = fmt.Sprintf("%s: %s", msg, strings.TrimSpace(*prepare.ErrorMessage))
				}
				details := map[string]interface{}{
					"provider_id":    selected.ID.String(),
					"tenant_id":      tenantID.String(),
					"prepare_status": string(prepare.Status),
				}
				if prepare.Namespace != "" {
					details["namespace"] = prepare.Namespace
				}
				if prepare.ErrorMessage != nil && strings.TrimSpace(*prepare.ErrorMessage) != "" {
					details["prepare_error"] = strings.TrimSpace(*prepare.ErrorMessage)
				}
				return &infraValidationError{
					status:  http.StatusConflict,
					message: msg,
					code:    "tenant_namespace_not_prepared",
					details: details,
				}
			}
		}
	} else if string(selected.ProviderType) != manifest.InfrastructureType {
		return &infraValidationError{status: http.StatusBadRequest, message: "selected infrastructure provider does not match infrastructure_type"}
	}

	return nil
}

func (h *BuildHandler) validateQuarantineArtifactAdmission(ctx context.Context, tenantID uuid.UUID, manifest *build.BuildManifest) *infraValidationError {
	if manifest == nil || h.quarantineAdmission == nil {
		return nil
	}
	refs := quarantineCandidateRefs(manifest)
	for _, ref := range refs {
		if !isDigestPinnedQuarantineRef(ref) {
			return &infraValidationError{
				status:  http.StatusConflict,
				message: "quarantine artifact references must be immutable digest-pinned refs",
				code:    "quarantine_artifact_digest_required",
				details: map[string]interface{}{
					"image_ref": ref,
					"tenant_id": tenantID.String(),
				},
			}
		}
		allowed, err := h.quarantineAdmission.IsArtifactRefReleased(ctx, tenantID, ref)
		if err != nil {
			h.logger.Error("Failed to validate quarantine artifact admission",
				zap.Error(err),
				zap.String("tenant_id", tenantID.String()),
				zap.String("image_ref", ref))
			return &infraValidationError{
				status:  http.StatusInternalServerError,
				message: "failed to validate quarantine artifact release state",
				code:    "quarantine_artifact_admission_error",
			}
		}
		if !allowed {
			releaseState := ""
			if stateLookup, ok := h.quarantineAdmission.(QuarantineArtifactReleaseStateLookup); ok {
				stateValue, stateErr := stateLookup.GetArtifactReleaseStateByRef(ctx, tenantID, ref)
				if stateErr != nil {
					h.logger.Warn("Failed to load quarantine artifact release state for deny contract",
						zap.Error(stateErr),
						zap.String("tenant_id", tenantID.String()),
						zap.String("image_ref", ref))
				} else {
					releaseState = strings.TrimSpace(strings.ToLower(stateValue))
				}
			}

			message := "selected quarantine artifact is not released for tenant consumption"
			code := "quarantine_artifact_not_released"
			switch releaseState {
			case "withdrawn":
				message = "selected quarantine artifact has been withdrawn and can no longer be consumed"
				code = "quarantine_artifact_withdrawn"
			case "superseded":
				message = "selected quarantine artifact has been superseded and can no longer be consumed"
				code = "quarantine_artifact_superseded"
			}

			return &infraValidationError{
				status:  http.StatusConflict,
				message: message,
				code:    code,
				details: map[string]interface{}{
					"image_ref":     ref,
					"tenant_id":     tenantID.String(),
					"release_state": releaseState,
				},
			}
		}
	}
	return nil
}

func quarantineCandidateRefs(manifest *build.BuildManifest) []string {
	if manifest == nil {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, 4)
	appendRef := func(v string) {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return
		}
		if !strings.Contains(strings.ToLower(trimmed), "/quarantine/") {
			return
		}
		if _, exists := seen[trimmed]; exists {
			return
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}

	appendRef(manifest.BaseImage)
	if manifest.Metadata != nil {
		appendRef(manifestStringMetadata(manifest.Metadata, "base_image", "baseImage"))
		appendRef(manifestStringMetadata(manifest.Metadata, "source_image_ref", "sourceImageRef"))
		appendRef(manifestStringMetadata(manifest.Metadata, "image_ref", "imageRef"))
	}
	return out
}

func isDigestPinnedQuarantineRef(imageRef string) bool {
	trimmed := strings.TrimSpace(imageRef)
	if trimmed == "" {
		return false
	}
	return strings.Contains(strings.ToLower(trimmed), "@sha256:")
}

// TODO: Implement RBAC permission checking
// func (h *BuildHandler) checkPermission(ctx context.Context, permission string) bool {
//     // Check if user has the required permission
//     return true // Placeholder
// }
