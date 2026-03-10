package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/image"
)

// ImageHandler handles HTTP requests for image operations
type ImageHandler struct {
	service *image.Service
	logger  *zap.Logger
}

// NewImageHandler creates a new image handler
func NewImageHandler(service *image.Service, logger *zap.Logger) *ImageHandler {
	return &ImageHandler{
		service: service,
		logger:  logger,
	}
}

// CreateImageRequest represents the request payload for creating an image
type CreateImageRequest struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	Visibility       string                 `json:"visibility"`
	RepositoryURL    *string                `json:"repository_url,omitempty"`
	RegistryProvider *string                `json:"registry_provider,omitempty"`
	Architecture     *string                `json:"architecture,omitempty"`
	OS               *string                `json:"os,omitempty"`
	Language         *string                `json:"language,omitempty"`
	Framework        *string                `json:"framework,omitempty"`
	Version          *string                `json:"version,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateImageRequest represents the request payload for updating an image
type UpdateImageRequest struct {
	Description      *string                `json:"description,omitempty"`
	Visibility       *string                `json:"visibility,omitempty"`
	Status           *string                `json:"status,omitempty"`
	RepositoryURL    *string                `json:"repository_url,omitempty"`
	RegistryProvider *string                `json:"registry_provider,omitempty"`
	Architecture     *string                `json:"architecture,omitempty"`
	OS               *string                `json:"os,omitempty"`
	Language         *string                `json:"language,omitempty"`
	Framework        *string                `json:"framework,omitempty"`
	Version          *string                `json:"version,omitempty"`
	Tags             *[]string              `json:"tags,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ImageResponse represents the response payload for an image
type ImageResponse struct {
	ID               uuid.UUID              `json:"id"`
	TenantID         uuid.UUID              `json:"tenant_id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Visibility       string                 `json:"visibility"`
	Status           string                 `json:"status"`
	RepositoryURL    *string                `json:"repository_url,omitempty"`
	RegistryProvider *string                `json:"registry_provider,omitempty"`
	Architecture     *string                `json:"architecture,omitempty"`
	OS               *string                `json:"os,omitempty"`
	Language         *string                `json:"language,omitempty"`
	Framework        *string                `json:"framework,omitempty"`
	Version          *string                `json:"version,omitempty"`
	Tags             []string               `json:"tags"`
	Metadata         map[string]interface{} `json:"metadata"`
	SizeBytes        *int64                 `json:"size_bytes,omitempty"`
	PullCount        int64                  `json:"pull_count"`
	CreatedBy        uuid.UUID              `json:"created_by"`
	UpdatedBy        uuid.UUID              `json:"updated_by"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
	DeprecatedAt     *string                `json:"deprecated_at,omitempty"`
	ArchivedAt       *string                `json:"archived_at,omitempty"`
}

// SearchImagesRequest represents search query parameters
type SearchImagesRequest struct {
	Query            string   `json:"query,omitempty"`
	Status           string   `json:"status,omitempty"`
	RegistryProvider string   `json:"registry_provider,omitempty"`
	Architecture     string   `json:"architecture,omitempty"`
	OS               string   `json:"os,omitempty"`
	Language         string   `json:"language,omitempty"`
	Framework        string   `json:"framework,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	SortBy           string   `json:"sort_by,omitempty"`
	SortOrder        string   `json:"sort_order,omitempty"`
	Limit            int      `json:"limit,omitempty"`
	Offset           int      `json:"offset,omitempty"`
}

// CreateImage handles POST /api/v1/images
func (h *ImageHandler) CreateImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tenant ID from context (set by middleware)
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		h.logger.Error("Tenant ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user ID from context
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		h.logger.Error("User ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		http.Error(w, "Image name is required", http.StatusBadRequest)
		return
	}

	if req.Visibility == "" {
		req.Visibility = string(image.VisibilityPrivate)
	}

	// Create image
	img, err := h.service.CreateImage(ctx, tenantID, userID, req.Name, req.Description, image.ImageVisibility(req.Visibility))
	if err != nil {
		h.logger.Error("Failed to create image", zap.Error(err))
		if err == image.ErrImageExists {
			http.Error(w, "Image with this name already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create image", http.StatusInternalServerError)
		return
	}

	// Get updated image
	updatedImg, err := h.service.GetImage(ctx, img.ID(), &userID, &tenantID)
	if err != nil {
		h.logger.Error("Failed to get updated image", zap.Error(err))
	}

	response := h.imageToResponse(updatedImg)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetImage handles GET /api/v1/images/{id}
func (h *ImageHandler) GetImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tenant ID from context
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		h.logger.Error("Tenant ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user ID from context
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		h.logger.Error("User ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	img, err := h.service.GetImage(ctx, id, &userID, &tenantID)
	if err != nil {
		h.logger.Error("Failed to get image", zap.String("image_id", id.String()), zap.Error(err))
		if err == image.ErrImageNotFound {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		if err == image.ErrPermissionDenied {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		http.Error(w, "Failed to get image", http.StatusInternalServerError)
		return
	}

	response := h.imageToResponse(img)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateImage handles PUT /api/v1/images/{id}
func (h *ImageHandler) UpdateImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tenant ID and user ID from context
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		h.logger.Error("Tenant ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		h.logger.Error("User ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	var req UpdateImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build image updates
	updates := image.ImageUpdates{
		Description:  req.Description,
		Visibility:   (*image.ImageVisibility)(req.Visibility),
		Status:       (*image.ImageStatus)(req.Status),
		Metadata:     req.Metadata,
	}

	if req.Tags != nil {
		updates.TagsToAdd = *req.Tags
	}

	// Update image
	if err := h.service.UpdateImage(ctx, id, userID, tenantID, updates); err != nil {
		h.logger.Error("Failed to update image", zap.String("image_id", id.String()), zap.Error(err))
		if err == image.ErrImageNotFound {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		if err == image.ErrPermissionDenied {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		http.Error(w, "Failed to update image", http.StatusInternalServerError)
		return
	}

	// Get updated image
	img, err := h.service.GetImage(ctx, id, &userID, &tenantID)
	if err != nil {
		h.logger.Error("Failed to get updated image", zap.String("image_id", id.String()), zap.Error(err))
		http.Error(w, "Failed to get updated image", http.StatusInternalServerError)
		return
	}

	response := h.imageToResponse(img)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteImage handles DELETE /api/v1/images/{id}
func (h *ImageHandler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		h.logger.Error("User ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get tenant ID from context
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		h.logger.Error("Tenant ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteImage(ctx, id, userID, tenantID); err != nil {
		h.logger.Error("Failed to delete image", zap.String("image_id", id.String()), zap.Error(err))
		if err == image.ErrImageNotFound {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		if err == image.ErrPermissionDenied {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		http.Error(w, "Failed to delete image", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SearchImages handles GET /api/v1/images/search
func (h *ImageHandler) SearchImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tenant ID from context
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		h.logger.Error("Tenant ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user ID from context
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		h.logger.Error("User ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	req := SearchImagesRequest{
		Query:     r.URL.Query().Get("q"),
		Status:    r.URL.Query().Get("status"),
		SortBy:    r.URL.Query().Get("sort_by"),
		SortOrder: r.URL.Query().Get("sort_order"),
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			req.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			req.Offset = offset
		}
	}

	// Parse tags
	if tagsStr := r.URL.Query().Get("tags"); tagsStr != "" {
		req.Tags = strings.Split(tagsStr, ",")
		for i, tag := range req.Tags {
			req.Tags[i] = strings.TrimSpace(tag)
		}
	}

	// Set default limit
	if req.Limit == 0 {
		req.Limit = 50
	}

	// Build search filters
	filters := image.SearchFilters{
		Status:    (*image.ImageStatus)(stringPtrToStringPtr(&req.Status)),
		Tags:      req.Tags,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}

	images, err := h.service.SearchImages(ctx, req.Query, &userID, &tenantID, filters)
	if err != nil {
		h.logger.Error("Failed to search images", zap.Error(err))
		http.Error(w, "Failed to search images", http.StatusInternalServerError)
		return
	}

	var responses []ImageResponse
	for _, img := range images {
		responses = append(responses, h.imageToResponse(img))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"images": responses,
		"total":  len(responses),
	})
}

// GetPopularImages handles GET /api/v1/images/popular
func (h *ImageHandler) GetPopularImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tenant ID from context
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		h.logger.Error("Tenant ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user ID from context
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		h.logger.Error("User ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	images, err := h.service.GetPopularImages(ctx, &userID, &tenantID, limit)
	if err != nil {
		h.logger.Error("Failed to get popular images", zap.Error(err))
		http.Error(w, "Failed to get popular images", http.StatusInternalServerError)
		return
	}

	var responses []ImageResponse
	for _, img := range images {
		responses = append(responses, h.imageToResponse(img))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"images": responses,
	})
}

// GetRecentImages handles GET /api/v1/images/recent
func (h *ImageHandler) GetRecentImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tenant ID from context
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		h.logger.Error("Tenant ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user ID from context
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		h.logger.Error("User ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	images, err := h.service.GetRecentImages(ctx, &userID, &tenantID, limit)
	if err != nil {
		h.logger.Error("Failed to get recent images", zap.Error(err))
		http.Error(w, "Failed to get recent images", http.StatusInternalServerError)
		return
	}

	var responses []ImageResponse
	for _, img := range images {
		responses = append(responses, h.imageToResponse(img))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"images": responses,
	})
}

// Helper functions

func (h *ImageHandler) imageToResponse(img *image.Image) ImageResponse {
	var deprecatedAt, archivedAt *string
	if img.DeprecatedAt() != nil {
		t := img.DeprecatedAt().Format("2006-01-02T15:04:05Z07:00")
		deprecatedAt = &t
	}
	if img.ArchivedAt() != nil {
		t := img.ArchivedAt().Format("2006-01-02T15:04:05Z07:00")
		archivedAt = &t
	}

	return ImageResponse{
		ID:               img.ID(),
		TenantID:         img.TenantID(),
		Name:             img.Name(),
		Description:      img.Description(),
		Visibility:       string(img.Visibility()),
		Status:           string(img.Status()),
		RepositoryURL:    img.RepositoryURL(),
		RegistryProvider: img.RegistryProvider(),
		Architecture:     img.Architecture(),
		OS:               img.OS(),
		Language:         img.Language(),
		Framework:        img.Framework(),
		Version:          img.Version(),
		Tags:             img.Tags(),
		Metadata:         img.Metadata(),
		SizeBytes:        img.SizeBytes(),
		PullCount:        img.PullCount(),
		CreatedBy:        img.CreatedBy(),
		UpdatedBy:        img.UpdatedBy(),
		CreatedAt:        img.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        img.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
		DeprecatedAt:     deprecatedAt,
		ArchivedAt:       archivedAt,
	}
}

func stringPtrToStringPtr(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}
