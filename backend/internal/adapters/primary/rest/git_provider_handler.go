package rest

import (
    "encoding/json"
    "net/http"

    "go.uber.org/zap"

    "github.com/srikarm/image-factory/internal/domain/gitprovider"
)

type GitProviderHandler struct {
    service *gitprovider.Service
    logger  *zap.Logger
}

type GitProviderResponse struct {
    Key         string `json:"key"`
    DisplayName string `json:"display_name"`
    ProviderType string `json:"provider_type"`
    APIBaseURL  string `json:"api_base_url,omitempty"`
    SupportsAPI bool   `json:"supports_api"`
}

func NewGitProviderHandler(service *gitprovider.Service, logger *zap.Logger) *GitProviderHandler {
    return &GitProviderHandler{service: service, logger: logger}
}

func (h *GitProviderHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
    providers, err := h.service.ListActiveProviders(r.Context())
    if err != nil {
        h.logger.Error("Failed to list git providers", zap.Error(err))
        http.Error(w, "Failed to list git providers", http.StatusInternalServerError)
        return
    }

    responses := make([]GitProviderResponse, len(providers))
    for i, provider := range providers {
        responses[i] = GitProviderResponse{
            Key:          provider.Key(),
            DisplayName:  provider.DisplayName(),
            ProviderType: string(provider.ProviderType()),
            APIBaseURL:   provider.APIBaseURL(),
            SupportsAPI:  provider.SupportsAPI(),
        }
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(map[string]interface{}{"providers": responses}); err != nil {
        h.logger.Error("Failed to encode git providers response", zap.Error(err))
    }
}
