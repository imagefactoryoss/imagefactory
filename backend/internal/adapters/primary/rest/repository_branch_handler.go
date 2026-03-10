package rest

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
    "go.uber.org/zap"

    "github.com/srikarm/image-factory/internal/domain/project"
    "github.com/srikarm/image-factory/internal/domain/repositorybranch"
)

type RepositoryBranchHandler struct {
    branchService *repositorybranch.Service
    projectService *project.Service
    logger        *zap.Logger
}

type RepositoryBranchRequest struct {
    RepositoryURL string `json:"repository_url,omitempty"`
    AuthID        string `json:"auth_id,omitempty"`
    ProviderKey   string `json:"provider_key,omitempty"`
}

type RepositoryBranchResponse struct {
    Branches []string `json:"branches"`
}

func NewRepositoryBranchHandler(branchService *repositorybranch.Service, projectService *project.Service, logger *zap.Logger) *RepositoryBranchHandler {
    return &RepositoryBranchHandler{branchService: branchService, projectService: projectService, logger: logger}
}

func (h *RepositoryBranchHandler) ListBranches(w http.ResponseWriter, r *http.Request) {
    projectIDStr := chi.URLParam(r, "projectId")
    projectID, err := uuid.Parse(projectIDStr)
    if err != nil {
        h.logger.Error("Invalid project ID", zap.Error(err), zap.String("projectId", projectIDStr))
        http.Error(w, "Invalid project ID", http.StatusBadRequest)
        return
    }

    projectEntity, err := h.projectService.GetProject(r.Context(), projectID)
    if err != nil || projectEntity == nil {
        h.logger.Error("Failed to load project", zap.Error(err), zap.String("projectId", projectIDStr))
        http.Error(w, "Project not found", http.StatusNotFound)
        return
    }

    var req RepositoryBranchRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.logger.Error("Failed to decode branch list request", zap.Error(err))
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    repoURL := req.RepositoryURL
    if repoURL == "" {
        repoURL = projectEntity.GitRepo()
    }

    providerKey := req.ProviderKey
    if providerKey == "" {
        providerKey = projectEntity.GitProvider()
    }

    authIDStr := req.AuthID
    if authIDStr == "" && projectEntity.RepositoryAuthID() != nil {
        authIDStr = projectEntity.RepositoryAuthID().String()
    }

    if authIDStr == "" {
        http.Error(w, "Repository auth ID is required", http.StatusBadRequest)
        return
    }

    authID, err := uuid.Parse(authIDStr)
    if err != nil {
        http.Error(w, "Invalid auth ID", http.StatusBadRequest)
        return
    }

    branches, err := h.branchService.ListBranches(r.Context(), repositorybranch.ListBranchesRequest{
        ProjectID:   projectID,
        AuthID:      authID,
        RepoURL:     repoURL,
        ProviderKey: providerKey,
    })
    if err != nil {
        h.logger.Error("Failed to list repository branches", zap.Error(err))
        http.Error(w, "Failed to list repository branches", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(RepositoryBranchResponse{Branches: branches})
}
