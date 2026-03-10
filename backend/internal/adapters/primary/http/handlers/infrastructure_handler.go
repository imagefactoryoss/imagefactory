package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/infrastructure/k8s"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// InfrastructureHandler handles infrastructure recommendation and monitoring HTTP requests
type InfrastructureHandler struct {
	buildService *build.Service
	infraService *infrastructure.Service
	selector     *k8s.InfrastructureSelector
	logger       *zap.Logger
}

// NewInfrastructureHandler creates a new infrastructure handler
func NewInfrastructureHandler(buildService *build.Service, infraService *infrastructure.Service, selector *k8s.InfrastructureSelector, logger *zap.Logger) *InfrastructureHandler {
	return &InfrastructureHandler{
		buildService: buildService,
		infraService: infraService,
		selector:     selector,
		logger:       logger,
	}
}

// ============================================================================
// Request/Response Types
// ============================================================================

// InfrastructureRecommendationRequest represents a request for infrastructure recommendation
type InfrastructureRecommendationRequest struct {
	BuildMethod string                 `json:"build_method" validate:"required"`
	ProjectID   uuid.UUID              `json:"project_id" validate:"required"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// InfrastructureRecommendationResponse represents the infrastructure recommendation response
type InfrastructureRecommendationResponse struct {
	RecommendedInfrastructure string                 `json:"recommended_infrastructure"`
	Reason                    string                 `json:"reason"`
	Confidence                float64                `json:"confidence"`
	Requirements              *k8s.BuildRequirements `json:"requirements"`
	Alternatives              []AlternativeOption    `json:"alternatives"`
	Timestamp                 time.Time              `json:"timestamp"`
}

// AlternativeOption represents an alternative infrastructure option
type AlternativeOption struct {
	Infrastructure string  `json:"infrastructure"`
	Reason         string  `json:"reason"`
	Confidence     float64 `json:"confidence"`
}

// InfrastructureUsageResponse represents infrastructure usage metrics
type InfrastructureUsageResponse struct {
	TotalBuilds         int64                 `json:"total_builds"`
	InfrastructureUsage []InfrastructureUsage `json:"infrastructure_usage"`
	TimeRange           string                `json:"time_range"`
	Timestamp           time.Time             `json:"timestamp"`
}

// InfrastructureUsage represents usage for a specific infrastructure type
type InfrastructureUsage struct {
	InfrastructureType string  `json:"infrastructure_type"`
	BuildCount         int64   `json:"build_count"`
	Percentage         float64 `json:"percentage"`
	AvgDuration        float64 `json:"avg_duration_seconds"`
	SuccessRate        float64 `json:"success_rate"`
}

// ============================================================================
// HTTP Handlers
// ============================================================================

// GetInfrastructureRecommendation returns infrastructure recommendation for a build configuration
func (h *InfrastructureHandler) GetInfrastructureRecommendation(w http.ResponseWriter, r *http.Request) {
	var req InfrastructureRecommendationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode infrastructure recommendation request", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.BuildMethod == "" {
		h.respondError(w, http.StatusBadRequest, "build_method is required")
		return
	}
	if req.ProjectID == uuid.Nil {
		h.respondError(w, http.StatusBadRequest, "project_id is required")
		return
	}

	h.logger.Info("Getting infrastructure recommendation",
		zap.String("build_method", req.BuildMethod),
		zap.String("project_id", req.ProjectID.String()))

	// Create a k8s build for analysis
	k8sBuild := h.convertRequestToK8sBuild(req)

	// Get infrastructure recommendation
	decision, err := h.selector.SelectInfrastructure(r.Context(), k8sBuild)
	if err != nil {
		h.logger.Error("Failed to get infrastructure recommendation", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to get infrastructure recommendation")
		return
	}

	// Prefer Kubernetes when any K8s provider is available for this tenant.
	hasK8sProviders := false
	hasSchedulableK8sProviders := false
	hasAvailabilityInfo := false
	kubernetesSupported := false
	hasKubernetesSupportInfo := false
	if h.infraService != nil {
		if authCtx, ok := middleware.GetAuthContext(r); ok {
			if h.buildService != nil {
				if supported, err := h.buildService.SupportsKubernetesInfrastructure(r.Context(), authCtx.TenantID); err != nil {
					h.logger.Warn("Kubernetes infrastructure not supported for tenant",
						zap.Error(err),
						zap.String("tenant_id", authCtx.TenantID.String()))
				} else {
					kubernetesSupported = supported
					hasKubernetesSupportInfo = true
				}
			}

			availableProviders, err := h.infraService.GetAvailableProviders(r.Context(), authCtx.TenantID)
			if err != nil {
				h.logger.Warn("Failed to fetch available infrastructure providers for recommendation",
					zap.Error(err),
					zap.String("tenant_id", authCtx.TenantID.String()))
			} else {
				hasK8sProviders = hasK8sCapableProvider(availableProviders)
				hasSchedulableK8sProviders = hasSchedulableK8sProvider(availableProviders)
				hasAvailabilityInfo = true
				h.logger.Info("Infrastructure provider availability for recommendation",
					zap.String("tenant_id", authCtx.TenantID.String()),
					zap.Bool("has_k8s_provider", hasK8sProviders),
					zap.Bool("has_schedulable_k8s_provider", hasSchedulableK8sProviders),
					zap.Bool("kubernetes_supported", kubernetesSupported),
					zap.Strings("provider_types", summarizeProviderTypes(availableProviders)),
				)
			}
		}
	}

	if hasAvailabilityInfo && hasSchedulableK8sProviders && (!hasKubernetesSupportInfo || kubernetesSupported) && decision.Type == k8s.InfrastructureBuildNodes {
		decision = &k8s.InfrastructureDecision{
			Type:       k8s.InfrastructureKubernetes,
			Reason:     "Schedulable Kubernetes providers are available; prefer Kubernetes over build nodes",
			Confidence: 0.85,
		}
	} else if hasAvailabilityInfo && (!hasSchedulableK8sProviders || (hasKubernetesSupportInfo && !kubernetesSupported)) && decision.Type == k8s.InfrastructureKubernetes {
		reason := "No Kubernetes providers available; using build nodes"
		if hasKubernetesSupportInfo && !kubernetesSupported {
			reason = "Kubernetes execution is not available for this tenant; using build nodes"
		} else if hasK8sProviders && !hasSchedulableK8sProviders {
			reason = "Kubernetes providers exist but are not currently schedulable (health/readiness/pressure); using build nodes"
		}
		decision = &k8s.InfrastructureDecision{
			Type:       k8s.InfrastructureBuildNodes,
			Reason:     reason,
			Confidence: 0.85,
		}
	}

	// Analyze build requirements
	requirements := h.selector.AnalyzeBuildRequirements(k8sBuild)

	// Get alternative options
	alternatives := h.getAlternatives(requirements, k8sBuild.RequiresGPU, hasK8sProviders, hasAvailabilityInfo)

	response := InfrastructureRecommendationResponse{
		RecommendedInfrastructure: string(decision.Type),
		Reason:                    decision.Reason,
		Confidence:                decision.Confidence,
		Requirements:              requirements,
		Alternatives:              alternatives,
		Timestamp:                 time.Now(),
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetInfrastructureUsage returns infrastructure usage metrics for admin dashboard
func (h *InfrastructureHandler) GetInfrastructureUsage(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	scopeTenantID := authCtx.TenantID
	allTenants := false
	if authCtx.IsSystemAdmin {
		if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Tenant-Scope")), "all") {
			allTenants = true
		}
		switch strings.TrimSpace(strings.ToLower(r.URL.Query().Get("all_tenants"))) {
		case "true", "1", "yes":
			allTenants = true
		}
	}
	if tenantIDRaw := strings.TrimSpace(r.URL.Query().Get("tenant_id")); tenantIDRaw != "" {
		parsedTenantID, err := uuid.Parse(tenantIDRaw)
		if err != nil || parsedTenantID == uuid.Nil {
			h.respondError(w, http.StatusBadRequest, "Invalid tenant_id")
			return
		}
		if !authCtx.IsSystemAdmin && parsedTenantID != authCtx.TenantID {
			h.respondError(w, http.StatusForbidden, "Access denied to this tenant")
			return
		}
		scopeTenantID = parsedTenantID
		allTenants = false
	}

	// Get time range from query params (default to last 30 days)
	timeRange := r.URL.Query().Get("range")
	if timeRange == "" {
		timeRange = "30d"
	}

	h.logger.Info("Getting infrastructure usage metrics",
		zap.String("time_range", timeRange),
		zap.String("tenant_id", scopeTenantID.String()),
		zap.Bool("all_tenants", allTenants))

	// Get infrastructure usage from build service
	// Note: This would need to be implemented in the build service/repository
	// For now, return mock data
	usage := []InfrastructureUsage{
		{
			InfrastructureType: "kubernetes",
			BuildCount:         150,
			Percentage:         75.0,
			AvgDuration:        420.5,
			SuccessRate:        92.3,
		},
		{
			InfrastructureType: "build_nodes",
			BuildCount:         50,
			Percentage:         25.0,
			AvgDuration:        380.2,
			SuccessRate:        88.7,
		},
	}

	response := InfrastructureUsageResponse{
		TotalBuilds:         200,
		InfrastructureUsage: usage,
		TimeRange:           timeRange,
		Timestamp:           time.Now(),
	}

	h.respondJSON(w, http.StatusOK, response)
}

// ============================================================================
// Helper Methods
// ============================================================================

// convertRequestToK8sBuild converts a recommendation request to a k8s build
func (h *InfrastructureHandler) convertRequestToK8sBuild(req InfrastructureRecommendationRequest) *k8s.Build {
	// Extract resources (simplified - in production this would be more sophisticated)
	resources := k8s.BuildResources{
		CPU:      2.0,  // default
		MemoryGB: 4.0,  // default
		DiskGB:   20.0, // default
	}

	// Check for GPU requirements (simplified check)
	requiresGPU := false
	if req.Config != nil {
		// Check config for GPU-related keywords
		for key, value := range req.Config {
			if h.hasGPUKeywords(fmt.Sprintf("%v", key), fmt.Sprintf("%v", value)) {
				requiresGPU = true
				break
			}
		}
	}

	return &k8s.Build{
		ID:          uuid.New().String(), // Generate a temporary ID
		Method:      req.BuildMethod,
		Resources:   resources,
		Timeout:     30 * time.Minute,        // default timeout
		Environment: make(map[string]string), // empty for now
		RequiresGPU: requiresGPU,
	}
}

// hasGPUKeywords checks if strings contain GPU-related keywords
func (h *InfrastructureHandler) hasGPUKeywords(values ...string) bool {
	gpuKeywords := []string{"gpu", "cuda", "nvidia", "amd", "radeon"}
	for _, value := range values {
		for _, keyword := range gpuKeywords {
			if strings.Contains(strings.ToLower(value), keyword) {
				return true
			}
		}
	}
	return false
}

// getAlternatives returns alternative infrastructure options based on requirements
func (h *InfrastructureHandler) getAlternatives(requirements *k8s.BuildRequirements, requiresGPU bool, hasK8sProviders bool, hasAvailabilityInfo bool) []AlternativeOption {
	alternatives := []AlternativeOption{}

	// Simple logic for alternatives - in production this would be more sophisticated
	if requiresGPU && hasK8sProviders {
		alternatives = append(alternatives, AlternativeOption{
			Infrastructure: "kubernetes",
			Reason:         "GPU workloads typically run better on Kubernetes clusters",
			Confidence:     0.85,
		})
	} else if hasAvailabilityInfo && !hasK8sProviders {
		alternatives = append(alternatives, AlternativeOption{
			Infrastructure: "build_nodes",
			Reason:         "Kubernetes providers are not available; build nodes are required",
			Confidence:     0.8,
		})
	} else {
		alternatives = append(alternatives, AlternativeOption{
			Infrastructure: "build_nodes",
			Reason:         "Cost-effective for standard workloads without special requirements",
			Confidence:     0.70,
		})
	}

	return alternatives
}

func hasK8sCapableProvider(providers []*infrastructure.Provider) bool {
	for _, provider := range providers {
		if isK8sCapableProviderType(provider.ProviderType) {
			return true
		}
	}
	return false
}

func hasSchedulableK8sProvider(providers []*infrastructure.Provider) bool {
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		if !isK8sCapableProviderType(provider.ProviderType) {
			continue
		}
		if provider.Status != infrastructure.ProviderStatusOnline {
			continue
		}
		if provider.ReadinessStatus != nil && *provider.ReadinessStatus != "ready" {
			continue
		}
		if hasReadinessPrereq(provider.ReadinessMissingPrereqs, "cluster_capacity") {
			continue
		}
		return true
	}
	return false
}

func hasReadinessPrereq(prereqs []string, needle string) bool {
	needleLower := strings.ToLower(needle)
	for _, prereq := range prereqs {
		if strings.Contains(strings.ToLower(prereq), needleLower) {
			return true
		}
	}
	return false
}

func isK8sCapableProviderType(providerType infrastructure.ProviderType) bool {
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

func summarizeProviderTypes(providers []*infrastructure.Provider) []string {
	types := make([]string, 0, len(providers))
	seen := make(map[string]bool, len(providers))
	for _, provider := range providers {
		t := string(provider.ProviderType)
		if !seen[t] {
			seen[t] = true
			types = append(types, t)
		}
	}
	return types
}

// respondJSON sends a JSON response
func (h *InfrastructureHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

// respondError sends an error response
func (h *InfrastructureHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
