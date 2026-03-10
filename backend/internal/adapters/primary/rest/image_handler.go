package rest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/image"
	"github.com/srikarm/image-factory/internal/domain/tenant"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/denialtelemetry"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// ImageHandler handles image-related HTTP requests
type ImageHandler struct {
	imageService *image.Service
	versionRepo  image.ImageVersionRepository
	tagRepo      image.ImageTagRepository
	db           *sqlx.DB
	tenantRepo   tenant.Repository
	operationCapabilityChecker imageOperationCapabilityChecker
	denials      *denialtelemetry.Metrics
	auditService interface{} // Will be audit.Service when implemented
	logger       *zap.Logger
}

type imageOperationCapabilityChecker interface {
	IsOnDemandScanEntitled(ctx context.Context, tenantID uuid.UUID) (bool, error)
}

// NewImageHandler creates a new image handler
func NewImageHandler(imageService *image.Service, tenantRepo tenant.Repository, auditService interface{}, logger *zap.Logger) *ImageHandler {
	return &ImageHandler{
		imageService: imageService,
		tenantRepo:   tenantRepo,
		auditService: auditService,
		logger:       logger,
	}
}

// SetVersionRepository configures version data access for image endpoints.
func (h *ImageHandler) SetVersionRepository(repo image.ImageVersionRepository) {
	h.versionRepo = repo
}

// SetTagRepository configures tag data access for image endpoints.
func (h *ImageHandler) SetTagRepository(repo image.ImageTagRepository) {
	h.tagRepo = repo
}

// SetDB sets db handle for aggregate detail queries.
func (h *ImageHandler) SetDB(db *sqlx.DB) {
	h.db = db
}

func (h *ImageHandler) SetOperationCapabilityChecker(checker imageOperationCapabilityChecker) {
	h.operationCapabilityChecker = checker
}

func (h *ImageHandler) SetDenialMetrics(metrics *denialtelemetry.Metrics) {
	h.denials = metrics
}

// CreateImageRequest represents the request payload for creating an image
type CreateImageRequest struct {
	Name        string     `json:"name" validate:"required"`
	Description string     `json:"description,omitempty"`
	Visibility  string     `json:"visibility" validate:"required,oneof=public tenant private"`
	TenantID    *uuid.UUID `json:"tenant_id,omitempty"` // For system admins to specify which tenant
}

// UpdateImageRequest represents the request payload for updating an image
type UpdateImageRequest struct {
	Description      *string                `json:"description,omitempty"`
	Visibility       *string                `json:"visibility,omitempty"`
	Status           *string                `json:"status,omitempty"`
	RepositoryURL    string                 `json:"repository_url,omitempty"`
	RegistryProvider string                 `json:"registry_provider,omitempty"`
	Architecture     string                 `json:"architecture,omitempty"`
	OS               string                 `json:"os,omitempty"`
	Language         string                 `json:"language,omitempty"`
	Framework        string                 `json:"framework,omitempty"`
	Version          string                 `json:"version,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ImageResponse represents the image response
type ImageResponse struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	Visibility       string                 `json:"visibility"`
	Status           string                 `json:"status"`
	RepositoryURL    string                 `json:"repository_url,omitempty"`
	RegistryProvider string                 `json:"registry_provider,omitempty"`
	Architecture     string                 `json:"architecture,omitempty"`
	OS               string                 `json:"os,omitempty"`
	Language         string                 `json:"language,omitempty"`
	Framework        string                 `json:"framework,omitempty"`
	Version          string                 `json:"version,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	PullCount        int                    `json:"pull_count"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
	CreatedBy        string                 `json:"created_by"`
	TenantID         string                 `json:"tenant_id"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// SearchImagesResponse represents the search response
type SearchImagesResponse struct {
	Images     []ImageResponse `json:"images"`
	Pagination struct {
		Page       int `json:"page"`
		Limit      int `json:"limit"`
		Total      int `json:"total"`
		TotalPages int `json:"total_pages"`
	} `json:"pagination"`
}

// PopularImagesResponse represents the popular images response
type PopularImagesResponse struct {
	Images []ImageResponse `json:"images"`
}

// RecentImagesResponse represents the recent images response
type RecentImagesResponse struct {
	Images []ImageResponse `json:"images"`
}

// ImageVersionResponse represents the image version response
type ImageVersionResponse struct {
	ID          string                 `json:"id"`
	Version     string                 `json:"version"`
	Description string                 `json:"description,omitempty"`
	Digest      string                 `json:"digest,omitempty"`
	SizeBytes   int64                  `json:"size_bytes,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	PublishedAt string                 `json:"published_at"`
	CreatedBy   string                 `json:"created_by"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ImageVersionsResponse represents the image versions response
type ImageVersionsResponse struct {
	Versions []ImageVersionResponse `json:"versions"`
}

// ImageTagListResponse represents normalized tags response.
type ImageTagListResponse struct {
	Tags []string `json:"tags"`
}

// ImageStatsResponse represents stats for image detail screen.
type ImageStatsResponse struct {
	PullCount             int    `json:"pull_count"`
	VersionCount          int    `json:"version_count"`
	LayerCount            int    `json:"layer_count"`
	VulnerabilityScans    int    `json:"vulnerability_scan_count"`
	LastUpdated           string `json:"last_updated"`
	LatestScanStatus      string `json:"latest_scan_status,omitempty"`
	LatestHighVulnCount   int    `json:"latest_high_vulnerabilities,omitempty"`
	LatestMediumVulnCount int    `json:"latest_medium_vulnerabilities,omitempty"`
}

type ImageDetailTag struct {
	Tag      string `json:"tag"`
	Category string `json:"category,omitempty"`
}

type CatalogImageMetadataResponse struct {
	DockerConfigDigest             string `json:"docker_config_digest,omitempty"`
	DockerManifestDigest           string `json:"docker_manifest_digest,omitempty"`
	TotalLayerCount                int    `json:"total_layer_count,omitempty"`
	CompressedSizeBytes            int64  `json:"compressed_size_bytes,omitempty"`
	UncompressedSizeBytes          int64  `json:"uncompressed_size_bytes,omitempty"`
	PackagesCount                  int    `json:"packages_count,omitempty"`
	VulnerabilitiesHighCount       int    `json:"vulnerabilities_high_count,omitempty"`
	VulnerabilitiesMediumCount     int    `json:"vulnerabilities_medium_count,omitempty"`
	VulnerabilitiesLowCount        int    `json:"vulnerabilities_low_count,omitempty"`
	Entrypoint                     string `json:"entrypoint,omitempty"`
	Cmd                            string `json:"cmd,omitempty"`
	EnvVars                        string `json:"env_vars,omitempty"`
	WorkingDir                     string `json:"working_dir,omitempty"`
	Labels                         string `json:"labels,omitempty"`
	LastScannedAt                  string `json:"last_scanned_at,omitempty"`
	ScanTool                       string `json:"scan_tool,omitempty"`
	LayersEvidenceStatus           string `json:"layers_evidence_status,omitempty"`
	LayersEvidenceBuildID          string `json:"layers_evidence_build_id,omitempty"`
	LayersEvidenceUpdatedAt        string `json:"layers_evidence_updated_at,omitempty"`
	SBOMEvidenceStatus             string `json:"sbom_evidence_status,omitempty"`
	SBOMEvidenceBuildID            string `json:"sbom_evidence_build_id,omitempty"`
	SBOMEvidenceUpdatedAt          string `json:"sbom_evidence_updated_at,omitempty"`
	VulnerabilityEvidenceStatus    string `json:"vulnerability_evidence_status,omitempty"`
	VulnerabilityEvidenceBuildID   string `json:"vulnerability_evidence_build_id,omitempty"`
	VulnerabilityEvidenceUpdatedAt string `json:"vulnerability_evidence_updated_at,omitempty"`
}

type CatalogImageLayerResponse struct {
	LayerNumber        int                                      `json:"layer_number"`
	LayerDigest        string                                   `json:"layer_digest"`
	LayerSizeBytes     int64                                    `json:"layer_size_bytes,omitempty"`
	MediaType          string                                   `json:"media_type,omitempty"`
	IsBaseLayer        bool                                     `json:"is_base_layer"`
	BaseImageName      string                                   `json:"base_image_name,omitempty"`
	BaseImageTag       string                                   `json:"base_image_tag,omitempty"`
	UsedInBuildsCount  int                                      `json:"used_in_builds_count,omitempty"`
	LastUsedInBuildAt  string                                   `json:"last_used_in_build_at,omitempty"`
	HistoryCreatedBy   string                                   `json:"history_created_by,omitempty"`
	SourceCommand      string                                   `json:"source_command,omitempty"`
	DiffID             string                                   `json:"diff_id,omitempty"`
	PackageCount       int                                      `json:"package_count,omitempty"`
	VulnerabilityCount int                                      `json:"vulnerability_count,omitempty"`
	Packages           []CatalogImageLayerPackageResponse       `json:"packages,omitempty"`
	Vulnerabilities    []CatalogImageLayerVulnerabilityResponse `json:"vulnerabilities,omitempty"`
}

type CatalogImageLayerPackageResponse struct {
	PackageName                  string `json:"package_name"`
	PackageVersion               string `json:"package_version,omitempty"`
	PackageType                  string `json:"package_type,omitempty"`
	PackagePath                  string `json:"package_path,omitempty"`
	SourceCommand                string `json:"source_command,omitempty"`
	KnownVulnerabilitiesCount    int    `json:"known_vulnerabilities_count,omitempty"`
	CriticalVulnerabilitiesCount int    `json:"critical_vulnerabilities_count,omitempty"`
}

type CatalogImageLayerVulnerabilityResponse struct {
	CVEID          string  `json:"cve_id"`
	Severity       string  `json:"severity"`
	CVSSV3Score    float64 `json:"cvss_v3_score,omitempty"`
	PackageName    string  `json:"package_name,omitempty"`
	PackageVersion string  `json:"package_version,omitempty"`
	ReferenceURL   string  `json:"reference_url,omitempty"`
}

type CatalogImageSBOMPackageResponse struct {
	PackageName                  string                                  `json:"package_name"`
	PackageVersion               string                                  `json:"package_version,omitempty"`
	PackageType                  string                                  `json:"package_type,omitempty"`
	LayerDigest                  string                                  `json:"layer_digest,omitempty"`
	PackagePath                  string                                  `json:"package_path,omitempty"`
	SourceCommand                string                                  `json:"source_command,omitempty"`
	KnownVulnerabilitiesCount    int                                     `json:"known_vulnerabilities_count,omitempty"`
	CriticalVulnerabilitiesCount int                                     `json:"critical_vulnerabilities_count,omitempty"`
	HighSeverityVulnerabilities  []CatalogImageSBOMVulnerabilityResponse `json:"high_severity_vulnerabilities,omitempty"`
}

type CatalogImageSBOMVulnerabilityResponse struct {
	CVEID         string  `json:"cve_id"`
	Severity      string  `json:"severity"`
	CVSSV3Score   float64 `json:"cvss_v3_score,omitempty"`
	Description   string  `json:"description,omitempty"`
	PublishedDate string  `json:"published_date,omitempty"`
	ReferenceURL  string  `json:"reference_url,omitempty"`
}

type CatalogImageSBOMResponse struct {
	Format              string                            `json:"format"`
	Version             string                            `json:"version,omitempty"`
	Status              string                            `json:"status,omitempty"`
	GeneratedByTool     string                            `json:"generated_by_tool,omitempty"`
	ToolVersion         string                            `json:"tool_version,omitempty"`
	ScanTimestamp       string                            `json:"scan_timestamp,omitempty"`
	ScanDurationSeconds int                               `json:"scan_duration_seconds,omitempty"`
	Packages            []CatalogImageSBOMPackageResponse `json:"packages"`
}

type CatalogImageVulnerabilityScanResponse struct {
	ID                        string `json:"id"`
	BuildID                   string `json:"build_id,omitempty"`
	ScanTool                  string `json:"scan_tool"`
	ToolVersion               string `json:"tool_version,omitempty"`
	ScanStatus                string `json:"scan_status"`
	StartedAt                 string `json:"started_at,omitempty"`
	CompletedAt               string `json:"completed_at,omitempty"`
	DurationSeconds           int    `json:"duration_seconds,omitempty"`
	VulnerabilitiesCritical   int    `json:"vulnerabilities_critical,omitempty"`
	VulnerabilitiesHigh       int    `json:"vulnerabilities_high,omitempty"`
	VulnerabilitiesMedium     int    `json:"vulnerabilities_medium,omitempty"`
	VulnerabilitiesLow        int    `json:"vulnerabilities_low,omitempty"`
	VulnerabilitiesNegligible int    `json:"vulnerabilities_negligible,omitempty"`
	VulnerabilitiesUnknown    int    `json:"vulnerabilities_unknown,omitempty"`
	PassFailResult            string `json:"pass_fail_result,omitempty"`
	ComplianceCheckPassed     bool   `json:"compliance_check_passed"`
	ScanReportLocation        string `json:"scan_report_location,omitempty"`
	ErrorMessage              string `json:"error_message,omitempty"`
}

type ImageDetailsResponse struct {
	Image    *ImageResponse         `json:"image"`
	Versions []ImageVersionResponse `json:"versions"`
	Tags     struct {
		Inline     []string         `json:"inline"`
		Normalized []ImageDetailTag `json:"normalized"`
		Merged     []string         `json:"merged"`
	} `json:"tags"`
	Metadata           *CatalogImageMetadataResponse           `json:"metadata,omitempty"`
	Layers             []CatalogImageLayerResponse             `json:"layers"`
	SBOM               *CatalogImageSBOMResponse               `json:"sbom,omitempty"`
	VulnerabilityScans []CatalogImageVulnerabilityScanResponse `json:"vulnerability_scans"`
	Stats              ImageStatsResponse                      `json:"stats"`
}

type mutateTagsRequest struct {
	Tags []string `json:"tags"`
}

// CreateImage handles POST /api/v1/images
func (h *ImageHandler) CreateImage(w http.ResponseWriter, r *http.Request) {
	// Get auth context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in CreateImage")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req CreateImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode create image request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Use explicit tenant context from auth. System admins can override with tenant_id.
	tenantID := authCtx.TenantID
	h.logger.Info("CreateImage request",
		zap.Bool("is_system_admin", authCtx.IsSystemAdmin),
		zap.String("auth_tenant_id", authCtx.TenantID.String()),
		zap.String("user_id", authCtx.UserID.String()),
		zap.String("request_name", req.Name))

	if authCtx.IsSystemAdmin && req.TenantID != nil {
		h.logger.Info("System admin specified tenant_id",
			zap.String("specified_tenant_id", req.TenantID.String()))
		// Validate that the specified tenant exists.
		if tenant, err := h.tenantRepo.FindByID(r.Context(), *req.TenantID); err != nil || tenant == nil {
			h.logger.Warn("System admin specified invalid tenant_id",
				zap.String("specified_tenant_id", req.TenantID.String()),
				zap.Error(err))
			http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
			return
		}
		tenantID = *req.TenantID
	} else if !authCtx.IsSystemAdmin {
		h.logger.Info("Non-admin user using auth tenant_id",
			zap.String("tenant_id", tenantID.String()))
	}

	h.logger.Info("Creating image with tenant_id",
		zap.String("final_tenant_id", tenantID.String()),
		zap.String("image_name", req.Name))

	// Create the image
	img, err := h.imageService.CreateImage(r.Context(), tenantID, authCtx.UserID, req.Name, req.Description, image.ImageVisibility(req.Visibility))
	if err != nil {
		h.logger.Error("Failed to create image", zap.Error(err))
		if err == image.ErrInvalidTenantID {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log audit event
	if auditSvc, ok := h.auditService.(*audit.Service); ok {
		auditSvc.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventUserCreate, "image", "create", "Image created", map[string]interface{}{
			"image_id":   img.ID(),
			"image_name": img.Name(),
			"tenant_id":  authCtx.TenantID,
			"user_id":    authCtx.UserID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": h.imageToResponse(img),
	})
}

// GetImage handles GET /api/v1/images/{id}
func (h *ImageHandler) GetImage(w http.ResponseWriter, r *http.Request) {
	// Get auth context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in GetImage")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	// Prepare tenant ID pointer
	tenantIDPtr := &authCtx.TenantID

	img, err := h.imageService.GetImage(r.Context(), id, &authCtx.UserID, tenantIDPtr)
	if err != nil {
		h.logger.Error("Failed to get image", zap.Error(err))
		if err == image.ErrImageNotFound {
			http.Error(w, "Image not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": h.imageToResponse(img),
	})
}

// TriggerOnDemandScan handles POST /api/v1/images/{id}/scan
func (h *ImageHandler) TriggerOnDemandScan(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		writeImageScanError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	if authCtx.TenantID == uuid.Nil {
		writeImageScanError(w, http.StatusBadRequest, "invalid_tenant", "tenant context is required", nil)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeImageScanError(w, http.StatusBadRequest, "invalid_id", "invalid image id", nil)
		return
	}

	entitled := false
	if h.operationCapabilityChecker != nil {
		entitled, err = h.operationCapabilityChecker.IsOnDemandScanEntitled(r.Context(), authCtx.TenantID)
		if err != nil {
			h.logger.Error("Failed to evaluate on-demand scan entitlement", zap.Error(err), zap.String("tenant_id", authCtx.TenantID.String()))
			writeImageScanError(w, http.StatusInternalServerError, "internal_error", "failed to evaluate scan entitlement", nil)
			return
		}
	}
	if !entitled {
		if h.denials != nil {
			h.denials.RecordDenied(authCtx.TenantID, "ondemand_image_scanning", "tenant_capability_not_entitled")
		}
		if auditSvc, ok := h.auditService.(*audit.Service); ok {
			_ = auditSvc.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID, audit.AuditEventCapabilityDenied, "images", "scan", "On-demand scan denied: capability not entitled", map[string]interface{}{
				"tenant_id":      authCtx.TenantID.String(),
				"capability_key": "ondemand_image_scanning",
				"reason":         "tenant_capability_not_entitled",
				"image_id":       id.String(),
			})
		}
		writeImageScanError(w, http.StatusForbidden, "tenant_capability_not_entitled", "on-demand image scanning is not enabled for this tenant", map[string]interface{}{
			"tenant_id":      authCtx.TenantID.String(),
			"capability_key": "ondemand_image_scanning",
		})
		return
	}

	tenantIDPtr := &authCtx.TenantID
	img, err := h.imageService.GetImage(r.Context(), id, &authCtx.UserID, tenantIDPtr)
	if err != nil {
		if err == image.ErrImageNotFound {
			writeImageScanError(w, http.StatusNotFound, "not_found", "image not found", nil)
			return
		}
		h.logger.Error("Failed to load image for on-demand scan", zap.Error(err), zap.String("image_id", id.String()))
		writeImageScanError(w, http.StatusInternalServerError, "internal_error", "failed to queue on-demand scan", nil)
		return
	}
	if img == nil {
		writeImageScanError(w, http.StatusNotFound, "not_found", "image not found", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"scan_request_id": uuid.New().String(),
			"image_id":        id.String(),
			"status":          "queued",
			"message":         "on-demand scan has been queued",
		},
	})
}

// UpdateImage handles PUT /api/v1/images/{id}
func (h *ImageHandler) UpdateImage(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in UpdateImage")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	var req UpdateImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode update image request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updates := image.ImageUpdates{
		Metadata: req.Metadata,
	}
	if req.Description != nil {
		updates.Description = req.Description
	}
	if req.Visibility != nil {
		v := image.ImageVisibility(*req.Visibility)
		updates.Visibility = &v
	}
	if req.Status != nil {
		s := image.ImageStatus(*req.Status)
		updates.Status = &s
	}
	if len(req.Tags) > 0 {
		updates.TagsToAdd = req.Tags
	}

	if err := h.imageService.UpdateImage(r.Context(), id, authCtx.UserID, authCtx.TenantID, updates); err != nil {
		h.logger.Error("Failed to update image", zap.Error(err), zap.String("image_id", id.String()))
		switch err {
		case image.ErrImageNotFound:
			http.Error(w, "Image not found", http.StatusNotFound)
		case image.ErrPermissionDenied:
			http.Error(w, "Permission denied", http.StatusForbidden)
		case image.ErrInvalidVisibility, image.ErrInvalidStatus:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "Failed to update image", http.StatusInternalServerError)
		}
		return
	}

	tenantIDPtr := &authCtx.TenantID
	img, err := h.imageService.GetImage(r.Context(), id, &authCtx.UserID, tenantIDPtr)
	if err != nil {
		http.Error(w, "Failed to fetch updated image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": h.imageToResponse(img),
	})
}

// DeleteImage handles DELETE /api/v1/images/{id}
func (h *ImageHandler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in DeleteImage")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	if err := h.imageService.DeleteImage(r.Context(), id, authCtx.UserID, authCtx.TenantID); err != nil {
		h.logger.Error("Failed to delete image", zap.Error(err), zap.String("image_id", id.String()))
		switch err {
		case image.ErrImageNotFound:
			http.Error(w, "Image not found", http.StatusNotFound)
		case image.ErrPermissionDenied:
			http.Error(w, "Permission denied", http.StatusForbidden)
		default:
			http.Error(w, "Failed to delete image", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SearchImages handles GET /api/v1/images/search
func (h *ImageHandler) SearchImages(w http.ResponseWriter, r *http.Request) {
	// Get auth context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in SearchImages")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	query := r.URL.Query().Get("query")
	status := r.URL.Query().Get("status")
	registryProvider := r.URL.Query().Get("registry_provider")
	architecture := r.URL.Query().Get("architecture")
	os := r.URL.Query().Get("os")
	language := r.URL.Query().Get("language")
	framework := r.URL.Query().Get("framework")
	tagsParam := r.URL.Query().Get("tags")
	sortBy := r.URL.Query().Get("sort_by")
	sortOrder := strings.ToLower(r.URL.Query().Get("sort_order"))

	var tags []string
	if tagsParam != "" {
		tags = strings.Split(tagsParam, ",")
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offsetStr := r.URL.Query().Get("offset")
	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Create search filters - only set non-empty values
	filters := image.SearchFilters{
		Tags:   tags,
		Limit:  limit,
		Offset: offset,
	}

	if status != "" {
		statusEnum := image.ImageStatus(status)
		filters.Status = &statusEnum
	}

	if registryProvider != "" {
		filters.RegistryProvider = &registryProvider
	}

	if architecture != "" {
		filters.Architecture = &architecture
	}

	if os != "" {
		filters.OS = &os
	}

	if language != "" {
		filters.Language = &language
	}

	if framework != "" {
		filters.Framework = &framework
	}

	if sortBy != "" {
		switch sortBy {
		case "name", "created_at", "updated_at", "pull_count", "size_bytes":
			filters.SortBy = sortBy
		default:
			http.Error(w, "Invalid sort_by. Allowed: name, created_at, updated_at, pull_count, size_bytes", http.StatusBadRequest)
			return
		}
	}
	if sortOrder != "" {
		if sortOrder != "asc" && sortOrder != "desc" {
			http.Error(w, "Invalid sort_order. Allowed: asc, desc", http.StatusBadRequest)
			return
		}
		filters.SortOrder = sortOrder
	}

	// Always scope to the selected tenant context.
	tenantIDPtr := &authCtx.TenantID

	// Perform search
	images, err := h.imageService.SearchImages(r.Context(), query, &authCtx.UserID, tenantIDPtr, filters)
	if err != nil {
		h.logger.Error("Failed to search images", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response
	var imageResponses []ImageResponse
	for _, img := range images {
		imageResponses = append(imageResponses, *h.imageToResponse(img))
	}

	response := SearchImagesResponse{
		Images: imageResponses,
	}
	response.Pagination.Page = (offset / limit) + 1
	response.Pagination.Limit = limit
	response.Pagination.Total = len(images) // This is approximate since we don't have total count
	response.Pagination.TotalPages = (len(images) + limit - 1) / limit

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPopularImages handles GET /api/v1/images/popular
func (h *ImageHandler) GetPopularImages(w http.ResponseWriter, r *http.Request) {
	// Get auth context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in GetPopularImages")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	// Always scope to the selected tenant context.
	tenantIDPtr := &authCtx.TenantID

	images, err := h.imageService.GetPopularImages(r.Context(), &authCtx.UserID, tenantIDPtr, limit)
	if err != nil {
		h.logger.Error("Failed to get popular images", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response
	var imageResponses []ImageResponse
	for _, img := range images {
		imageResponses = append(imageResponses, *h.imageToResponse(img))
	}

	response := PopularImagesResponse{
		Images: imageResponses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRecentImages handles GET /api/v1/images/recent
func (h *ImageHandler) GetRecentImages(w http.ResponseWriter, r *http.Request) {
	// Get auth context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in GetRecentImages")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	// Always scope to the selected tenant context.
	tenantIDPtr := &authCtx.TenantID

	images, err := h.imageService.GetRecentImages(r.Context(), &authCtx.UserID, tenantIDPtr, limit)
	if err != nil {
		h.logger.Error("Failed to get recent images", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response
	var imageResponses []ImageResponse
	for _, img := range images {
		imageResponses = append(imageResponses, *h.imageToResponse(img))
	}

	response := RecentImagesResponse{
		Images: imageResponses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetImageVersions handles GET /api/v1/images/{id}/versions
func (h *ImageHandler) GetImageVersions(w http.ResponseWriter, r *http.Request) {
	// Get auth context
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		h.logger.Error("No auth context found in GetImageVersions")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	// Verify image access first
	tenantIDPtr := &authCtx.TenantID
	if _, err := h.imageService.GetImage(r.Context(), id, &authCtx.UserID, tenantIDPtr); err != nil {
		if err == image.ErrImageNotFound || err == image.ErrPermissionDenied {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to load image", http.StatusInternalServerError)
		return
	}

	versions := []*image.ImageVersion{}
	if h.versionRepo != nil {
		versions, err = h.versionRepo.FindByImageID(r.Context(), id)
		if err != nil {
			h.logger.Error("Failed to fetch image versions", zap.Error(err))
			http.Error(w, "Failed to fetch image versions", http.StatusInternalServerError)
			return
		}
	}

	// Convert to response
	var versionResponses []ImageVersionResponse
	for _, version := range versions {
		versionResponses = append(versionResponses, *h.versionToResponse(version))
	}

	response := ImageVersionsResponse{
		Versions: versionResponses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": response,
	})
}

// GetImageTags handles GET /api/v1/images/{id}/tags
func (h *ImageHandler) GetImageTags(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	tenantIDPtr := &authCtx.TenantID
	img, err := h.imageService.GetImage(r.Context(), id, &authCtx.UserID, tenantIDPtr)
	if err != nil {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	tagSet := map[string]struct{}{}
	for _, t := range img.Tags() {
		if strings.TrimSpace(t) != "" {
			tagSet[t] = struct{}{}
		}
	}
	if h.tagRepo != nil {
		normalizedTags, err := h.tagRepo.FindByImageID(r.Context(), id)
		if err != nil {
			h.logger.Warn("Failed to fetch normalized tags", zap.Error(err))
		} else {
			for _, t := range normalizedTags {
				if strings.TrimSpace(t.Tag) != "" {
					tagSet[t.Tag] = struct{}{}
				}
			}
		}
	}

	tags := make([]string, 0, len(tagSet))
	for t := range tagSet {
		tags = append(tags, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": ImageTagListResponse{Tags: tags},
	})
}

// AddImageTags handles POST /api/v1/images/{id}/tags
func (h *ImageHandler) AddImageTags(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	var req mutateTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updates := image.ImageUpdates{TagsToAdd: req.Tags}
	if err := h.imageService.UpdateImage(r.Context(), id, authCtx.UserID, authCtx.TenantID, updates); err != nil {
		if err == image.ErrImageNotFound {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		if err == image.ErrPermissionDenied {
			http.Error(w, "Permission denied", http.StatusForbidden)
			return
		}
		h.logger.Error("Failed to add image tags", zap.Error(err))
		http.Error(w, "Failed to add tags", http.StatusInternalServerError)
		return
	}

	h.GetImageTags(w, r)
}

// RemoveImageTags handles DELETE /api/v1/images/{id}/tags
func (h *ImageHandler) RemoveImageTags(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	var req mutateTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updates := image.ImageUpdates{TagsToRemove: req.Tags}
	if err := h.imageService.UpdateImage(r.Context(), id, authCtx.UserID, authCtx.TenantID, updates); err != nil {
		if err == image.ErrImageNotFound {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		if err == image.ErrPermissionDenied {
			http.Error(w, "Permission denied", http.StatusForbidden)
			return
		}
		h.logger.Error("Failed to remove image tags", zap.Error(err))
		http.Error(w, "Failed to remove tags", http.StatusInternalServerError)
		return
	}

	h.GetImageTags(w, r)
}

// GetImageStats handles GET /api/v1/images/{id}/stats
func (h *ImageHandler) GetImageStats(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	tenantIDPtr := &authCtx.TenantID
	img, err := h.imageService.GetImage(r.Context(), id, &authCtx.UserID, tenantIDPtr)
	if err != nil {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	stats := ImageStatsResponse{
		PullCount:   int(img.PullCount()),
		LastUpdated: img.UpdatedAt().Format(time.RFC3339),
	}

	if h.versionRepo != nil {
		versions, err := h.versionRepo.FindByImageID(r.Context(), id)
		if err == nil {
			stats.VersionCount = len(versions)
		}
	}

	if h.db != nil {
		_ = h.db.GetContext(r.Context(), &stats.LayerCount, `SELECT COUNT(*) FROM catalog_image_layers WHERE image_id = $1`, id)
		_ = h.db.GetContext(r.Context(), &stats.VulnerabilityScans, `SELECT COUNT(*) FROM catalog_image_vulnerability_scans WHERE image_id = $1`, id)

		var latest struct {
			ScanStatus            string `db:"scan_status"`
			VulnerabilitiesHigh   int    `db:"vulnerabilities_high"`
			VulnerabilitiesMedium int    `db:"vulnerabilities_medium"`
		}
		if err := h.db.GetContext(r.Context(), &latest, `
			SELECT scan_status, vulnerabilities_high, vulnerabilities_medium
			FROM catalog_image_vulnerability_scans
			WHERE image_id = $1
			ORDER BY COALESCE(completed_at, started_at) DESC
			LIMIT 1
		`, id); err == nil {
			stats.LatestScanStatus = latest.ScanStatus
			stats.LatestHighVulnCount = latest.VulnerabilitiesHigh
			stats.LatestMediumVulnCount = latest.VulnerabilitiesMedium
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": stats,
	})
}

// GetImageDetails handles GET /api/v1/images/{id}/details
func (h *ImageHandler) GetImageDetails(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	tenantIDPtr := &authCtx.TenantID
	img, err := h.imageService.GetImage(r.Context(), id, &authCtx.UserID, tenantIDPtr)
	if err != nil {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	details := ImageDetailsResponse{
		Image:              h.imageToResponse(img),
		Versions:           []ImageVersionResponse{},
		Layers:             []CatalogImageLayerResponse{},
		VulnerabilityScans: []CatalogImageVulnerabilityScanResponse{},
	}
	details.Tags.Inline = img.Tags()

	tagSet := map[string]struct{}{}
	for _, t := range img.Tags() {
		if strings.TrimSpace(t) != "" {
			tagSet[t] = struct{}{}
		}
	}

	if h.versionRepo != nil {
		versions, err := h.versionRepo.FindByImageID(r.Context(), id)
		if err == nil {
			for _, v := range versions {
				details.Versions = append(details.Versions, *h.versionToResponse(v))
			}
			details.Stats.VersionCount = len(details.Versions)
		}
	}

	if h.tagRepo != nil {
		tags, err := h.tagRepo.FindByImageID(r.Context(), id)
		if err == nil {
			for _, t := range tags {
				details.Tags.Normalized = append(details.Tags.Normalized, ImageDetailTag{
					Tag:      t.Tag,
					Category: t.Category,
				})
				if strings.TrimSpace(t.Tag) != "" {
					tagSet[t.Tag] = struct{}{}
				}
			}
		}
	}
	for t := range tagSet {
		details.Tags.Merged = append(details.Tags.Merged, t)
	}

	details.Stats.PullCount = int(img.PullCount())
	details.Stats.LastUpdated = img.UpdatedAt().Format(time.RFC3339)

	if h.db != nil {
		if metadata, err := h.loadCatalogImageMetadata(r.Context(), id); err == nil {
			details.Metadata = metadata
		}
		if layers, err := h.loadCatalogImageLayers(r.Context(), id); err == nil {
			details.Layers = layers
			details.Stats.LayerCount = len(layers)
		}
		if sbom, err := h.loadCatalogImageSBOM(r.Context(), id); err == nil {
			details.SBOM = sbom
		}
		if scans, err := h.loadCatalogImageVulnerabilityScans(r.Context(), id); err == nil {
			details.VulnerabilityScans = scans
			details.Stats.VulnerabilityScans = len(scans)
			if len(scans) > 0 {
				details.Stats.LatestScanStatus = scans[0].ScanStatus
				details.Stats.LatestHighVulnCount = scans[0].VulnerabilitiesHigh
				details.Stats.LatestMediumVulnCount = scans[0].VulnerabilitiesMedium
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": details,
	})
}

// Helper method to convert domain image to response
func (h *ImageHandler) imageToResponse(img *image.Image) *ImageResponse {
	return &ImageResponse{
		ID:               img.ID().String(),
		Name:             img.Name(),
		Description:      img.Description(),
		Visibility:       string(img.Visibility()),
		Status:           string(img.Status()),
		RepositoryURL:    stringPtrToString(img.RepositoryURL()),
		RegistryProvider: stringPtrToString(img.RegistryProvider()),
		Architecture:     stringPtrToString(img.Architecture()),
		OS:               stringPtrToString(img.OS()),
		Language:         stringPtrToString(img.Language()),
		Framework:        stringPtrToString(img.Framework()),
		Version:          stringPtrToString(img.Version()),
		Tags:             img.Tags(),
		PullCount:        int(img.PullCount()),
		CreatedAt:        img.CreatedAt().Format(time.RFC3339),
		UpdatedAt:        img.UpdatedAt().Format(time.RFC3339),
		CreatedBy:        img.CreatedBy().String(),
		TenantID:         img.TenantID().String(),
		Metadata:         img.Metadata(),
	}
}

// Helper method to convert domain image version to response
func (h *ImageHandler) versionToResponse(version *image.ImageVersion) *ImageVersionResponse {
	return &ImageVersionResponse{
		ID:          version.ID().String(),
		Version:     version.Version(),
		Description: stringPtrToString(version.Description()),
		Digest:      stringPtrToString(version.Digest()),
		SizeBytes:   int64PtrToInt64(version.SizeBytes()),
		Tags:        version.Tags(),
		CreatedAt:   version.CreatedAt().Format(time.RFC3339),
		PublishedAt: version.PublishedAt().Format(time.RFC3339),
		CreatedBy:   version.CreatedBy().String(),
		Metadata:    version.Metadata(),
	}
}

type catalogImageMetadataModel struct {
	DockerConfigDigest             sql.NullString `db:"docker_config_digest"`
	DockerManifestDigest           sql.NullString `db:"docker_manifest_digest"`
	TotalLayerCount                sql.NullInt64  `db:"total_layer_count"`
	CompressedSizeBytes            sql.NullInt64  `db:"compressed_size_bytes"`
	UncompressedSizeBytes          sql.NullInt64  `db:"uncompressed_size_bytes"`
	PackagesCount                  sql.NullInt64  `db:"packages_count"`
	VulnerabilitiesHighCount       sql.NullInt64  `db:"vulnerabilities_high_count"`
	VulnerabilitiesMediumCount     sql.NullInt64  `db:"vulnerabilities_medium_count"`
	VulnerabilitiesLowCount        sql.NullInt64  `db:"vulnerabilities_low_count"`
	Entrypoint                     sql.NullString `db:"entrypoint"`
	Cmd                            sql.NullString `db:"cmd"`
	EnvVars                        sql.NullString `db:"env_vars"`
	WorkingDir                     sql.NullString `db:"working_dir"`
	Labels                         sql.NullString `db:"labels"`
	LastScannedAt                  sql.NullTime   `db:"last_scanned_at"`
	ScanTool                       sql.NullString `db:"scan_tool"`
	LayersEvidenceStatus           sql.NullString `db:"layers_evidence_status"`
	LayersEvidenceBuildID          sql.NullString `db:"layers_evidence_build_id"`
	LayersEvidenceUpdatedAt        sql.NullTime   `db:"layers_evidence_updated_at"`
	SBOMEvidenceStatus             sql.NullString `db:"sbom_evidence_status"`
	SBOMEvidenceBuildID            sql.NullString `db:"sbom_evidence_build_id"`
	SBOMEvidenceUpdatedAt          sql.NullTime   `db:"sbom_evidence_updated_at"`
	VulnerabilityEvidenceStatus    sql.NullString `db:"vulnerability_evidence_status"`
	VulnerabilityEvidenceBuildID   sql.NullString `db:"vulnerability_evidence_build_id"`
	VulnerabilityEvidenceUpdatedAt sql.NullTime   `db:"vulnerability_evidence_updated_at"`
}

type catalogImageLayerModel struct {
	LayerNumber       int            `db:"layer_number"`
	LayerDigest       string         `db:"layer_digest"`
	LayerSizeBytes    sql.NullInt64  `db:"layer_size_bytes"`
	MediaType         sql.NullString `db:"media_type"`
	IsBaseLayer       bool           `db:"is_base_layer"`
	BaseImageName     sql.NullString `db:"base_image_name"`
	BaseImageTag      sql.NullString `db:"base_image_tag"`
	UsedInBuildsCount sql.NullInt64  `db:"used_in_builds_count"`
	LastUsedInBuildAt sql.NullTime   `db:"last_used_in_build_at"`
}

type catalogImageSBOMModel struct {
	ID                  uuid.UUID      `db:"id"`
	SBOMFormat          string         `db:"sbom_format"`
	SBOMVersion         sql.NullString `db:"sbom_version"`
	GeneratedByTool     sql.NullString `db:"generated_by_tool"`
	ToolVersion         sql.NullString `db:"tool_version"`
	ScanTimestamp       sql.NullTime   `db:"scan_timestamp"`
	ScanDurationSeconds sql.NullInt64  `db:"scan_duration_seconds"`
	Status              sql.NullString `db:"status"`
}

type catalogImageSBOMPackageModel struct {
	PackageName                  string         `db:"package_name"`
	PackageVersion               sql.NullString `db:"package_version"`
	PackageType                  sql.NullString `db:"package_type"`
	LayerDigest                  sql.NullString `db:"layer_digest"`
	PackagePath                  sql.NullString `db:"package_path"`
	SourceCommand                sql.NullString `db:"source_command"`
	KnownVulnerabilitiesCount    sql.NullInt64  `db:"known_vulnerabilities_count"`
	CriticalVulnerabilitiesCount sql.NullInt64  `db:"critical_vulnerabilities_count"`
}

type catalogImageSBOMHighVulnModel struct {
	PackageName    string          `db:"package_name"`
	PackageVersion sql.NullString  `db:"package_version"`
	PackageType    sql.NullString  `db:"package_type"`
	CVEID          string          `db:"cve_id"`
	Severity       sql.NullString  `db:"cvss_v3_severity"`
	CVSSV3Score    sql.NullFloat64 `db:"cvss_v3_score"`
	Description    sql.NullString  `db:"cve_description"`
	PublishedDate  sql.NullTime    `db:"published_date"`
	References     sql.NullString  `db:"references"`
}

type catalogImageVulnerabilityScanModel struct {
	ID                        uuid.UUID      `db:"id"`
	BuildID                   sql.NullString `db:"build_id"`
	ScanTool                  string         `db:"scan_tool"`
	ToolVersion               sql.NullString `db:"tool_version"`
	ScanStatus                string         `db:"scan_status"`
	StartedAt                 sql.NullTime   `db:"started_at"`
	CompletedAt               sql.NullTime   `db:"completed_at"`
	DurationSeconds           sql.NullInt64  `db:"duration_seconds"`
	VulnerabilitiesCritical   sql.NullInt64  `db:"vulnerabilities_critical"`
	VulnerabilitiesHigh       sql.NullInt64  `db:"vulnerabilities_high"`
	VulnerabilitiesMedium     sql.NullInt64  `db:"vulnerabilities_medium"`
	VulnerabilitiesLow        sql.NullInt64  `db:"vulnerabilities_low"`
	VulnerabilitiesNegligible sql.NullInt64  `db:"vulnerabilities_negligible"`
	VulnerabilitiesUnknown    sql.NullInt64  `db:"vulnerabilities_unknown"`
	PassFailResult            sql.NullString `db:"pass_fail_result"`
	ComplianceCheckPassed     sql.NullBool   `db:"compliance_check_passed"`
	ScanReportLocation        sql.NullString `db:"scan_report_location"`
	ErrorMessage              sql.NullString `db:"error_message"`
}

type catalogImageLayerEvidenceModel struct {
	LayerDigest      string         `db:"layer_digest"`
	HistoryCreatedBy sql.NullString `db:"history_created_by"`
	SourceCommand    sql.NullString `db:"source_command"`
	DiffID           sql.NullString `db:"diff_id"`
}

type catalogImageLayerPackageModel struct {
	LayerDigest                  string         `db:"layer_digest"`
	PackageName                  string         `db:"package_name"`
	PackageVersion               sql.NullString `db:"package_version"`
	PackageType                  sql.NullString `db:"package_type"`
	PackagePath                  sql.NullString `db:"package_path"`
	SourceCommand                sql.NullString `db:"source_command"`
	KnownVulnerabilitiesCount    sql.NullInt64  `db:"known_vulnerabilities_count"`
	CriticalVulnerabilitiesCount sql.NullInt64  `db:"critical_vulnerabilities_count"`
}

type catalogImageLayerVulnerabilityModel struct {
	LayerDigest    string          `db:"layer_digest"`
	PackageName    string          `db:"package_name"`
	PackageVersion sql.NullString  `db:"package_version"`
	CVEID          string          `db:"cve_id"`
	Severity       sql.NullString  `db:"severity"`
	CVSSV3Score    sql.NullFloat64 `db:"cvss_v3_score"`
	ReferenceURL   sql.NullString  `db:"reference_url"`
}

func (h *ImageHandler) loadCatalogImageMetadata(ctx context.Context, imageID uuid.UUID) (*CatalogImageMetadataResponse, error) {
	var model catalogImageMetadataModel
	err := h.db.GetContext(ctx, &model, `
		SELECT docker_config_digest, docker_manifest_digest, total_layer_count,
		       compressed_size_bytes, uncompressed_size_bytes, packages_count,
		       vulnerabilities_high_count, vulnerabilities_medium_count, vulnerabilities_low_count,
		       entrypoint, cmd, env_vars, working_dir, labels, last_scanned_at, scan_tool,
		       layers_evidence_status, layers_evidence_build_id::text AS layers_evidence_build_id, layers_evidence_updated_at,
		       sbom_evidence_status, sbom_evidence_build_id::text AS sbom_evidence_build_id, sbom_evidence_updated_at,
		       vulnerability_evidence_status, vulnerability_evidence_build_id::text AS vulnerability_evidence_build_id, vulnerability_evidence_updated_at
		FROM catalog_image_metadata
		WHERE image_id = $1
	`, imageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return catalogImageMetadataModelToResponse(model), nil
}

func catalogImageMetadataModelToResponse(model catalogImageMetadataModel) *CatalogImageMetadataResponse {
	resp := &CatalogImageMetadataResponse{
		DockerConfigDigest:   nullString(model.DockerConfigDigest),
		DockerManifestDigest: nullString(model.DockerManifestDigest),
		Entrypoint:           nullString(model.Entrypoint),
		Cmd:                  nullString(model.Cmd),
		EnvVars:              nullString(model.EnvVars),
		WorkingDir:           nullString(model.WorkingDir),
		Labels:               nullString(model.Labels),
		ScanTool:             nullString(model.ScanTool),
	}
	if model.TotalLayerCount.Valid {
		resp.TotalLayerCount = int(model.TotalLayerCount.Int64)
	}
	if model.CompressedSizeBytes.Valid {
		resp.CompressedSizeBytes = model.CompressedSizeBytes.Int64
	}
	if model.UncompressedSizeBytes.Valid {
		resp.UncompressedSizeBytes = model.UncompressedSizeBytes.Int64
	}
	if model.PackagesCount.Valid {
		resp.PackagesCount = int(model.PackagesCount.Int64)
	}
	if model.VulnerabilitiesHighCount.Valid {
		resp.VulnerabilitiesHighCount = int(model.VulnerabilitiesHighCount.Int64)
	}
	if model.VulnerabilitiesMediumCount.Valid {
		resp.VulnerabilitiesMediumCount = int(model.VulnerabilitiesMediumCount.Int64)
	}
	if model.VulnerabilitiesLowCount.Valid {
		resp.VulnerabilitiesLowCount = int(model.VulnerabilitiesLowCount.Int64)
	}
	if model.LastScannedAt.Valid {
		resp.LastScannedAt = model.LastScannedAt.Time.Format(time.RFC3339)
	}
	resp.LayersEvidenceStatus = nullString(model.LayersEvidenceStatus)
	resp.LayersEvidenceBuildID = nullString(model.LayersEvidenceBuildID)
	if model.LayersEvidenceUpdatedAt.Valid {
		resp.LayersEvidenceUpdatedAt = model.LayersEvidenceUpdatedAt.Time.Format(time.RFC3339)
	}
	resp.SBOMEvidenceStatus = nullString(model.SBOMEvidenceStatus)
	resp.SBOMEvidenceBuildID = nullString(model.SBOMEvidenceBuildID)
	if model.SBOMEvidenceUpdatedAt.Valid {
		resp.SBOMEvidenceUpdatedAt = model.SBOMEvidenceUpdatedAt.Time.Format(time.RFC3339)
	}
	resp.VulnerabilityEvidenceStatus = nullString(model.VulnerabilityEvidenceStatus)
	resp.VulnerabilityEvidenceBuildID = nullString(model.VulnerabilityEvidenceBuildID)
	if model.VulnerabilityEvidenceUpdatedAt.Valid {
		resp.VulnerabilityEvidenceUpdatedAt = model.VulnerabilityEvidenceUpdatedAt.Time.Format(time.RFC3339)
	}
	return resp
}

func (h *ImageHandler) loadCatalogImageLayers(ctx context.Context, imageID uuid.UUID) ([]CatalogImageLayerResponse, error) {
	var models []catalogImageLayerModel
	if err := h.db.SelectContext(ctx, &models, `
		SELECT layer_number, layer_digest, layer_size_bytes, media_type, is_base_layer,
		       base_image_name, base_image_tag, used_in_builds_count, last_used_in_build_at
		FROM catalog_image_layers
		WHERE image_id = $1
		ORDER BY layer_number ASC
	`, imageID); err != nil {
		return nil, err
	}

	layerEvidenceByDigest := map[string]catalogImageLayerEvidenceModel{}
	var layerEvidenceRows []catalogImageLayerEvidenceModel
	if err := h.db.SelectContext(ctx, &layerEvidenceRows, `
		SELECT layer_digest, history_created_by, source_command, diff_id
		FROM catalog_image_layer_evidence
		WHERE image_id = $1
	`, imageID); err == nil {
		for _, row := range layerEvidenceRows {
			layerEvidenceByDigest[row.LayerDigest] = row
		}
	}

	layerPackagesByDigest := map[string][]CatalogImageLayerPackageResponse{}
	var layerPackageRows []catalogImageLayerPackageModel
	if err := h.db.SelectContext(ctx, &layerPackageRows, `
		SELECT layer_digest, package_name, package_version, package_type, package_path, source_command,
		       known_vulnerabilities_count, critical_vulnerabilities_count
		FROM catalog_image_layer_packages
		WHERE image_id = $1
		ORDER BY layer_digest ASC, known_vulnerabilities_count DESC, package_name ASC
	`, imageID); err == nil {
		for _, row := range layerPackageRows {
			item := CatalogImageLayerPackageResponse{
				PackageName:    row.PackageName,
				PackageVersion: nullString(row.PackageVersion),
				PackageType:    nullString(row.PackageType),
				PackagePath:    nullString(row.PackagePath),
				SourceCommand:  nullString(row.SourceCommand),
			}
			if row.KnownVulnerabilitiesCount.Valid {
				item.KnownVulnerabilitiesCount = int(row.KnownVulnerabilitiesCount.Int64)
			}
			if row.CriticalVulnerabilitiesCount.Valid {
				item.CriticalVulnerabilitiesCount = int(row.CriticalVulnerabilitiesCount.Int64)
			}
			layerPackagesByDigest[row.LayerDigest] = append(layerPackagesByDigest[row.LayerDigest], item)
		}
	}

	layerVulnerabilitiesByDigest := map[string][]CatalogImageLayerVulnerabilityResponse{}
	var layerVulnRows []catalogImageLayerVulnerabilityModel
	if err := h.db.SelectContext(ctx, &layerVulnRows, `
		SELECT layer_digest, package_name, package_version, cve_id, severity, cvss_v3_score, reference_url
		FROM catalog_image_layer_vulnerabilities
		WHERE image_id = $1
		ORDER BY layer_digest ASC, cvss_v3_score DESC NULLS LAST, cve_id ASC
	`, imageID); err == nil {
		for _, row := range layerVulnRows {
			item := CatalogImageLayerVulnerabilityResponse{
				CVEID:          row.CVEID,
				Severity:       strings.ToUpper(nullString(row.Severity)),
				PackageName:    row.PackageName,
				PackageVersion: nullString(row.PackageVersion),
				ReferenceURL:   nullString(row.ReferenceURL),
			}
			if row.CVSSV3Score.Valid {
				item.CVSSV3Score = row.CVSSV3Score.Float64
			}
			layerVulnerabilitiesByDigest[row.LayerDigest] = append(layerVulnerabilitiesByDigest[row.LayerDigest], item)
		}
	}

	out := make([]CatalogImageLayerResponse, 0, len(models))
	for _, m := range models {
		row := CatalogImageLayerResponse{
			LayerNumber:   m.LayerNumber,
			LayerDigest:   m.LayerDigest,
			MediaType:     nullString(m.MediaType),
			IsBaseLayer:   m.IsBaseLayer,
			BaseImageName: nullString(m.BaseImageName),
			BaseImageTag:  nullString(m.BaseImageTag),
		}
		if m.LayerSizeBytes.Valid {
			row.LayerSizeBytes = m.LayerSizeBytes.Int64
		}
		if m.UsedInBuildsCount.Valid {
			row.UsedInBuildsCount = int(m.UsedInBuildsCount.Int64)
		}
		if m.LastUsedInBuildAt.Valid {
			row.LastUsedInBuildAt = m.LastUsedInBuildAt.Time.Format(time.RFC3339)
		}
		if evidence, ok := layerEvidenceByDigest[m.LayerDigest]; ok {
			row.HistoryCreatedBy = nullString(evidence.HistoryCreatedBy)
			row.SourceCommand = nullString(evidence.SourceCommand)
			row.DiffID = nullString(evidence.DiffID)
		}
		if packages := layerPackagesByDigest[m.LayerDigest]; len(packages) > 0 {
			row.Packages = packages
			row.PackageCount = len(packages)
		}
		if vulns := layerVulnerabilitiesByDigest[m.LayerDigest]; len(vulns) > 0 {
			row.Vulnerabilities = vulns
			row.VulnerabilityCount = len(vulns)
		}
		out = append(out, row)
	}
	return out, nil
}

func (h *ImageHandler) loadCatalogImageSBOM(ctx context.Context, imageID uuid.UUID) (*CatalogImageSBOMResponse, error) {
	var sbom catalogImageSBOMModel
	err := h.db.GetContext(ctx, &sbom, `
		SELECT id, sbom_format, sbom_version, generated_by_tool, tool_version,
		       scan_timestamp, scan_duration_seconds, status
		FROM catalog_image_sbom
		WHERE image_id = $1
	`, imageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	response := &CatalogImageSBOMResponse{
		Format:          sbom.SBOMFormat,
		Version:         nullString(sbom.SBOMVersion),
		Status:          nullString(sbom.Status),
		GeneratedByTool: nullString(sbom.GeneratedByTool),
		ToolVersion:     nullString(sbom.ToolVersion),
		Packages:        []CatalogImageSBOMPackageResponse{},
	}
	if sbom.ScanTimestamp.Valid {
		response.ScanTimestamp = sbom.ScanTimestamp.Time.Format(time.RFC3339)
	}
	if sbom.ScanDurationSeconds.Valid {
		response.ScanDurationSeconds = int(sbom.ScanDurationSeconds.Int64)
	}

	var packages []catalogImageSBOMPackageModel
	if err := h.db.SelectContext(ctx, &packages, `
		SELECT package_name, package_version, package_type,
		       known_vulnerabilities_count, critical_vulnerabilities_count
		FROM sbom_packages
		WHERE image_sbom_id = $1
		ORDER BY package_name ASC
	`, sbom.ID); err != nil {
		return nil, err
	}

	layerPackageByKey := map[string]catalogImageLayerPackageModel{}
	var layerPackageRows []catalogImageLayerPackageModel
	if err := h.db.SelectContext(ctx, &layerPackageRows, `
		SELECT layer_digest, package_name, package_version, package_type, package_path, source_command,
		       known_vulnerabilities_count, critical_vulnerabilities_count
		FROM catalog_image_layer_packages
		WHERE image_id = $1
	`, imageID); err == nil {
		for _, row := range layerPackageRows {
			key := fmt.Sprintf("%s|%s|%s", row.PackageName, nullString(row.PackageType), nullString(row.PackageVersion))
			if _, exists := layerPackageByKey[key]; exists {
				continue
			}
			layerPackageByKey[key] = row
		}
	}

	var highVulnRows []catalogImageSBOMHighVulnModel
	if err := h.db.SelectContext(ctx, &highVulnRows, `
		SELECT sp.package_name, sp.package_version, sp.package_type,
		       cve.cve_id, cve.cvss_v3_severity, cve.cvss_v3_score, cve.cve_description,
		       cve.published_date::timestamp AS published_date, cve.references
		FROM sbom_packages sp
		JOIN package_vulnerabilities pv
		  ON pv.package_name = sp.package_name
		 AND COALESCE(pv.package_version, '') = COALESCE(sp.package_version, '')
		JOIN cve_database cve ON cve.cve_id = pv.cve_id
		WHERE sp.image_sbom_id = $1
		  AND UPPER(COALESCE(cve.cvss_v3_severity, '')) IN ('HIGH', 'CRITICAL')
		ORDER BY cve.cvss_v3_score DESC NULLS LAST, cve.cve_id ASC
	`, sbom.ID); err != nil {
		return nil, err
	}

	keyForPkg := func(name string, typ, version sql.NullString) string {
		return fmt.Sprintf("%s|%s|%s", name, nullString(typ), nullString(version))
	}
	highVulnByPackage := map[string][]CatalogImageSBOMVulnerabilityResponse{}
	for _, v := range highVulnRows {
		item := CatalogImageSBOMVulnerabilityResponse{
			CVEID:        v.CVEID,
			Severity:     strings.ToUpper(nullString(v.Severity)),
			Description:  nullString(v.Description),
			ReferenceURL: cveReferenceURL(v.CVEID, v.References),
		}
		if v.CVSSV3Score.Valid {
			item.CVSSV3Score = v.CVSSV3Score.Float64
		}
		if v.PublishedDate.Valid {
			item.PublishedDate = v.PublishedDate.Time.Format("2006-01-02")
		}
		k := keyForPkg(v.PackageName, v.PackageType, v.PackageVersion)
		highVulnByPackage[k] = append(highVulnByPackage[k], item)
	}

	for _, p := range packages {
		item := CatalogImageSBOMPackageResponse{
			PackageName:    p.PackageName,
			PackageVersion: nullString(p.PackageVersion),
			PackageType:    nullString(p.PackageType),
		}
		if p.KnownVulnerabilitiesCount.Valid {
			item.KnownVulnerabilitiesCount = int(p.KnownVulnerabilitiesCount.Int64)
		}
		if p.CriticalVulnerabilitiesCount.Valid {
			item.CriticalVulnerabilitiesCount = int(p.CriticalVulnerabilitiesCount.Int64)
		}
		if layerPackage, ok := layerPackageByKey[keyForPkg(p.PackageName, p.PackageType, p.PackageVersion)]; ok {
			item.LayerDigest = layerPackage.LayerDigest
			item.PackagePath = nullString(layerPackage.PackagePath)
			item.SourceCommand = nullString(layerPackage.SourceCommand)
		}
		item.HighSeverityVulnerabilities = highVulnByPackage[keyForPkg(p.PackageName, p.PackageType, p.PackageVersion)]
		response.Packages = append(response.Packages, item)
	}

	return response, nil
}

func (h *ImageHandler) loadCatalogImageVulnerabilityScans(ctx context.Context, imageID uuid.UUID) ([]CatalogImageVulnerabilityScanResponse, error) {
	var scans []catalogImageVulnerabilityScanModel
	if err := h.db.SelectContext(ctx, &scans, `
		SELECT id, build_id::text AS build_id, scan_tool, tool_version, scan_status, started_at, completed_at, duration_seconds,
		       vulnerabilities_critical, vulnerabilities_high, vulnerabilities_medium,
		       vulnerabilities_low, vulnerabilities_negligible, vulnerabilities_unknown,
		       pass_fail_result, compliance_check_passed, scan_report_location, error_message
		FROM catalog_image_vulnerability_scans
		WHERE image_id = $1
		ORDER BY COALESCE(completed_at, started_at) DESC
	`, imageID); err != nil {
		return nil, err
	}
	return catalogImageVulnerabilityScanModelsToResponse(scans), nil
}

func catalogImageVulnerabilityScanModelsToResponse(scans []catalogImageVulnerabilityScanModel) []CatalogImageVulnerabilityScanResponse {
	out := make([]CatalogImageVulnerabilityScanResponse, 0, len(scans))
	for _, s := range scans {
		row := CatalogImageVulnerabilityScanResponse{
			ID:                    s.ID.String(),
			BuildID:               nullString(s.BuildID),
			ScanTool:              s.ScanTool,
			ToolVersion:           nullString(s.ToolVersion),
			ScanStatus:            s.ScanStatus,
			PassFailResult:        nullString(s.PassFailResult),
			ComplianceCheckPassed: s.ComplianceCheckPassed.Valid && s.ComplianceCheckPassed.Bool,
			ScanReportLocation:    nullString(s.ScanReportLocation),
			ErrorMessage:          nullString(s.ErrorMessage),
		}
		if s.StartedAt.Valid {
			row.StartedAt = s.StartedAt.Time.Format(time.RFC3339)
		}
		if s.CompletedAt.Valid {
			row.CompletedAt = s.CompletedAt.Time.Format(time.RFC3339)
		}
		if s.DurationSeconds.Valid {
			row.DurationSeconds = int(s.DurationSeconds.Int64)
		}
		if s.VulnerabilitiesCritical.Valid {
			row.VulnerabilitiesCritical = int(s.VulnerabilitiesCritical.Int64)
		}
		if s.VulnerabilitiesHigh.Valid {
			row.VulnerabilitiesHigh = int(s.VulnerabilitiesHigh.Int64)
		}
		if s.VulnerabilitiesMedium.Valid {
			row.VulnerabilitiesMedium = int(s.VulnerabilitiesMedium.Int64)
		}
		if s.VulnerabilitiesLow.Valid {
			row.VulnerabilitiesLow = int(s.VulnerabilitiesLow.Int64)
		}
		if s.VulnerabilitiesNegligible.Valid {
			row.VulnerabilitiesNegligible = int(s.VulnerabilitiesNegligible.Int64)
		}
		if s.VulnerabilitiesUnknown.Valid {
			row.VulnerabilitiesUnknown = int(s.VulnerabilitiesUnknown.Int64)
		}
		out = append(out, row)
	}
	return out
}

func nullString(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

func cveReferenceURL(cveID string, refs sql.NullString) string {
	if refs.Valid && strings.TrimSpace(refs.String) != "" {
		var urls []string
		if err := json.Unmarshal([]byte(refs.String), &urls); err == nil {
			for _, u := range urls {
				if strings.TrimSpace(u) != "" {
					return u
				}
			}
		}
	}
	if strings.TrimSpace(cveID) == "" {
		return ""
	}
	return fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cveID)
}

func writeImageScanError(w http.ResponseWriter, status int, code, message string, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"details": details,
		},
	})
}

// Helper function to convert string pointer to string
func stringPtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Helper function to convert int64 pointer to int64
func int64PtrToInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}
