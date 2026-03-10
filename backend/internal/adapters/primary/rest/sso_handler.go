package rest

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/sso"
	"github.com/srikarm/image-factory/internal/infrastructure/audit"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// SSOHandler handles SSO (SAML/OpenID Connect) HTTP requests
type SSOHandler struct {
	ssoService   *sso.Service
	auditService *audit.Service
	logger       *zap.Logger
}

// NewSSOHandler creates a new SSO handler
func NewSSOHandler(ssoService *sso.Service, auditService *audit.Service, logger *zap.Logger) *SSOHandler {
	return &SSOHandler{
		ssoService:   ssoService,
		auditService: auditService,
		logger:       logger,
	}
}

// CreateSAMLProviderRequest represents a request to create a SAML provider
type CreateSAMLProviderRequest struct {
	Name        string                 `json:"name"`
	EntityID    string                 `json:"entity_id,omitempty"`
	SSOURL      string                 `json:"sso_url"`
	SLOURL      string                 `json:"slo_url"`
	Certificate string                 `json:"certificate"`
	PrivateKey  string                 `json:"private_key"`
	Position    string                 `json:"position"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	Enabled     *bool                  `json:"enabled,omitempty"`
}

// CreateOIDCProviderRequest represents a request to create an OpenID Connect provider
type CreateOIDCProviderRequest struct {
	Name             string                 `json:"name"`
	Issuer           string                 `json:"issuer"`
	ClientID         string                 `json:"client_id"`
	ClientSecret     string                 `json:"client_secret"`
	AuthorizationURL string                 `json:"authorization_url"`
	TokenURL         string                 `json:"token_url"`
	UserInfoURL      string                 `json:"userinfo_url"`
	JWKSURL          string                 `json:"jwks_url"`
	RedirectURIs     []string               `json:"redirect_uris"`
	Scopes           []string               `json:"scopes"`
	ResponseTypes    []string               `json:"response_types"`
	GrantTypes       []string               `json:"grant_types"`
	Attributes       map[string]interface{} `json:"attributes,omitempty"`
	Enabled          *bool                  `json:"enabled,omitempty"`
}

// SAMLProviderResponse represents the response for SAML provider information
type SAMLProviderResponse struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	EntityID    string                 `json:"entity_id"`
	SSOURL      string                 `json:"sso_url"`
	SLOURL      string                 `json:"slo_url"`
	Certificate string                 `json:"certificate"`
	Position    string                 `json:"position"`
	Attributes  map[string]interface{} `json:"attributes"`
	Enabled     bool                   `json:"enabled"`
	Metadata    string                 `json:"metadata"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

// OIDCProviderResponse represents the response for OIDC provider information
type OIDCProviderResponse struct {
	ID               uuid.UUID              `json:"id"`
	Name             string                 `json:"name"`
	Issuer           string                 `json:"issuer"`
	ClientID         string                 `json:"client_id"`
	AuthorizationURL string                 `json:"authorization_url"`
	TokenURL         string                 `json:"token_url"`
	UserInfoURL      string                 `json:"userinfo_url"`
	JWKSURL          string                 `json:"jwks_url"`
	RedirectURIs     []string               `json:"redirect_uris"`
	Scopes           []string               `json:"scopes"`
	ResponseTypes    []string               `json:"response_types"`
	GrantTypes       []string               `json:"grant_types"`
	Attributes       map[string]interface{} `json:"attributes"`
	Enabled          bool                   `json:"enabled"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
}

type ToggleProviderStatusRequest struct {
	Enabled bool `json:"enabled"`
}

// CreateSAMLProvider handles POST /sso/saml/providers
func (h *SSOHandler) CreateSAMLProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateSAMLProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode SAML provider creation request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert request to service format
	serviceReq := sso.SAMLProviderCreateRequest{
		Name:        req.Name,
		EntityID:    req.EntityID,
		SSOURL:      req.SSOURL,
		SLOURL:      req.SLOURL,
		Certificate: req.Certificate,
		PrivateKey:  req.PrivateKey,
		Position:    sso.SSOPosition(req.Position),
		Attributes:  req.Attributes,
		Enabled:     req.Enabled,
	}

	// Create SAML provider
	provider, err := h.ssoService.CreateSAMLProvider(r.Context(), serviceReq)
	if err != nil {
		h.logger.Error("Failed to create SAML provider", zap.Error(err))
		http.Error(w, "Failed to create SAML provider: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Convert to response format
	response := SAMLProviderResponse{
		ID:          provider.ID,
		Name:        provider.Name,
		EntityID:    provider.EntityID,
		SSOURL:      provider.SSOURL,
		SLOURL:      provider.SLOURL,
		Certificate: provider.Certificate,
		Position:    string(provider.Position),
		Attributes:  provider.Attributes,
		Enabled:     provider.Enabled,
		Metadata:    provider.Metadata,
		CreatedAt:   provider.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   provider.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("SAML provider created successfully",
		zap.String("provider_id", provider.ID.String()),
		zap.String("entity_id", provider.EntityID))

	// Audit SAML provider creation
	if h.auditService != nil {
		authCtx, ok := middleware.GetAuthContext(r)
		if ok {
			h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID,
				audit.AuditEventSSOProviderCreate, "sso", "create_saml_provider",
				"SAML provider created successfully",
				map[string]interface{}{
					"provider_id": provider.ID.String(),
					"entity_id":   provider.EntityID,
					"name":        provider.Name,
				})
		}
	}
}

// CreateOIDCProvider handles POST /sso/oidc/providers
func (h *SSOHandler) CreateOIDCProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateOIDCProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode OIDC provider creation request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert request to service format
	serviceReq := sso.OIDCProviderCreateRequest{
		Name:             req.Name,
		Issuer:           req.Issuer,
		ClientID:         req.ClientID,
		ClientSecret:     req.ClientSecret,
		AuthorizationURL: req.AuthorizationURL,
		TokenURL:         req.TokenURL,
		UserInfoURL:      req.UserInfoURL,
		JWKSURL:          req.JWKSURL,
		RedirectURIs:     req.RedirectURIs,
		Scopes:           req.Scopes,
		ResponseTypes:    req.ResponseTypes,
		GrantTypes:       req.GrantTypes,
		Attributes:       req.Attributes,
		Enabled:          req.Enabled,
	}

	// Create OIDC provider
	provider, err := h.ssoService.CreateOIDCProvider(r.Context(), serviceReq)
	if err != nil {
		h.logger.Error("Failed to create OIDC provider", zap.Error(err))
		http.Error(w, "Failed to create OIDC provider: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Convert to response format
	response := OIDCProviderResponse{
		ID:               provider.ID,
		Name:             provider.Name,
		Issuer:           provider.Issuer,
		ClientID:         provider.ClientID,
		AuthorizationURL: provider.AuthorizationURL,
		TokenURL:         provider.TokenURL,
		UserInfoURL:      provider.UserInfoURL,
		JWKSURL:          provider.JWKSURL,
		RedirectURIs:     provider.RedirectURIs,
		Scopes:           provider.Scopes,
		ResponseTypes:    provider.ResponseTypes,
		GrantTypes:       provider.GrantTypes,
		Attributes:       provider.Attributes,
		Enabled:          provider.Enabled,
		CreatedAt:        provider.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        provider.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	h.logger.Info("OIDC provider created successfully",
		zap.String("provider_id", provider.ID.String()),
		zap.String("issuer", provider.Issuer))

	// Audit OIDC provider creation
	if h.auditService != nil {
		authCtx, ok := middleware.GetAuthContext(r)
		if ok {
			h.auditService.LogUserAction(r.Context(), authCtx.TenantID, authCtx.UserID,
				audit.AuditEventSSOProviderCreate, "sso", "create_oidc_provider",
				"OIDC provider created successfully",
				map[string]interface{}{
					"provider_id": provider.ID.String(),
					"issuer":      provider.Issuer,
					"name":        provider.Name,
				})
		}
	}
}

// GetSAMLProviders handles GET /sso/saml/providers
func (h *SSOHandler) GetSAMLProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all SAML providers
	providers, err := h.ssoService.GetAllSAMLProviders(r.Context())
	if err != nil {
		h.logger.Error("Failed to get SAML providers", zap.Error(err))
		http.Error(w, "Failed to retrieve SAML providers", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	responses := make([]SAMLProviderResponse, len(providers))
	for i, provider := range providers {
		responses[i] = SAMLProviderResponse{
			ID:          provider.ID,
			Name:        provider.Name,
			EntityID:    provider.EntityID,
			SSOURL:      provider.SSOURL,
			SLOURL:      provider.SLOURL,
			Certificate: provider.Certificate,
			Position:    string(provider.Position),
			Attributes:  provider.Attributes,
			Enabled:     provider.Enabled,
			Metadata:    provider.Metadata,
			CreatedAt:   provider.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   provider.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responses)
}

// GetOIDCProviders handles GET /sso/oidc/providers
func (h *SSOHandler) GetOIDCProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all OIDC providers
	providers, err := h.ssoService.GetAllOIDCProviders(r.Context())
	if err != nil {
		h.logger.Error("Failed to get OIDC providers", zap.Error(err))
		http.Error(w, "Failed to retrieve OIDC providers", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	responses := make([]OIDCProviderResponse, len(providers))
	for i, provider := range providers {
		responses[i] = OIDCProviderResponse{
			ID:               provider.ID,
			Name:             provider.Name,
			Issuer:           provider.Issuer,
			ClientID:         provider.ClientID,
			AuthorizationURL: provider.AuthorizationURL,
			TokenURL:         provider.TokenURL,
			UserInfoURL:      provider.UserInfoURL,
			JWKSURL:          provider.JWKSURL,
			RedirectURIs:     provider.RedirectURIs,
			Scopes:           provider.Scopes,
			ResponseTypes:    provider.ResponseTypes,
			GrantTypes:       provider.GrantTypes,
			Attributes:       provider.Attributes,
			Enabled:          provider.Enabled,
			CreatedAt:        provider.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:        provider.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responses)
}

// ValidateSAMLMetadata handles GET /sso/saml/metadata/{provider_id}
func (h *SSOHandler) ValidateSAMLMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get provider ID from URL path
	providerIDStr := r.PathValue("provider_id")
	if providerIDStr == "" {
		http.Error(w, "Provider ID is required", http.StatusBadRequest)
		return
	}

	providerID, err := uuid.Parse(providerIDStr)
	if err != nil {
		h.logger.Warn("Invalid provider ID format", zap.String("provider_id", providerIDStr))
		http.Error(w, "Invalid provider ID format", http.StatusBadRequest)
		return
	}

	// For now, return a mock response
	// In a real implementation, this would fetch the provider and return its metadata
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)

	// Mock SAML metadata response
	metadata := `<?xml version="1.0" encoding="UTF-8"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="` + providerID.String() + `">
  <md:IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat>
    <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://example.com/sso"/>
  </md:IDPSSODescriptor>
</md:EntityDescriptor>`

	w.Write([]byte(metadata))
}

// SSOConfigurationResponse represents the SSO configuration response
type SSOConfigurationResponse struct {
	SAMLProviders []SAMLProviderResponse `json:"saml_providers"`
	OIDCProviders []OIDCProviderResponse `json:"oidc_providers"`
	SSOEnabled    bool                   `json:"sso_enabled"`
	OIDCEnabled   bool                   `json:"oidc_enabled"`
}

// GetSSOConfiguration handles GET /sso/configuration
func (h *SSOHandler) GetSSOConfiguration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all SAML providers
	samlProviders, err := h.ssoService.GetAllSAMLProviders(r.Context())
	if err != nil {
		h.logger.Error("Failed to get SAML providers", zap.Error(err))
		http.Error(w, "Failed to retrieve SAML providers", http.StatusInternalServerError)
		return
	}

	// Get all OIDC providers
	oidcProviders, err := h.ssoService.GetAllOIDCProviders(r.Context())
	if err != nil {
		h.logger.Error("Failed to get OIDC providers", zap.Error(err))
		http.Error(w, "Failed to retrieve OIDC providers", http.StatusInternalServerError)
		return
	}

	// Convert SAML providers to response format
	samlResponses := make([]SAMLProviderResponse, len(samlProviders))
	for i, provider := range samlProviders {
		samlResponses[i] = SAMLProviderResponse{
			ID:          provider.ID,
			Name:        provider.Name,
			EntityID:    provider.EntityID,
			SSOURL:      provider.SSOURL,
			SLOURL:      provider.SLOURL,
			Certificate: provider.Certificate,
			Position:    string(provider.Position),
			Attributes:  provider.Attributes,
			Enabled:     provider.Enabled,
			Metadata:    provider.Metadata,
			CreatedAt:   provider.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   provider.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	// Convert OIDC providers to response format
	oidcResponses := make([]OIDCProviderResponse, len(oidcProviders))
	for i, provider := range oidcProviders {
		oidcResponses[i] = OIDCProviderResponse{
			ID:               provider.ID,
			Name:             provider.Name,
			Issuer:           provider.Issuer,
			ClientID:         provider.ClientID,
			AuthorizationURL: provider.AuthorizationURL,
			TokenURL:         provider.TokenURL,
			UserInfoURL:      provider.UserInfoURL,
			JWKSURL:          provider.JWKSURL,
			RedirectURIs:     provider.RedirectURIs,
			Scopes:           provider.Scopes,
			ResponseTypes:    provider.ResponseTypes,
			GrantTypes:       provider.GrantTypes,
			Attributes:       provider.Attributes,
			Enabled:          provider.Enabled,
			CreatedAt:        provider.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:        provider.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	// Create response
	response := SSOConfigurationResponse{
		SAMLProviders: samlResponses,
		OIDCProviders: oidcResponses,
		SSOEnabled:    hasEnabledSAML(samlProviders),
		OIDCEnabled:   hasEnabledOIDC(oidcProviders),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ToggleSAMLProviderStatus handles PATCH /sso/saml/providers/{provider_id}/status
func (h *SSOHandler) ToggleSAMLProviderStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providerIDStr := r.PathValue("provider_id")
	providerID, err := uuid.Parse(providerIDStr)
	if err != nil {
		http.Error(w, "Invalid provider ID", http.StatusBadRequest)
		return
	}

	var req ToggleProviderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.ssoService.SetSAMLProviderEnabled(r.Context(), providerID, req.Enabled); err != nil {
		h.logger.Error("Failed to toggle SAML provider status", zap.Error(err))
		http.Error(w, "Failed to update SAML provider status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"enabled": req.Enabled,
	})
}

// ToggleOIDCProviderStatus handles PATCH /sso/oidc/providers/{provider_id}/status
func (h *SSOHandler) ToggleOIDCProviderStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providerIDStr := r.PathValue("provider_id")
	providerID, err := uuid.Parse(providerIDStr)
	if err != nil {
		http.Error(w, "Invalid provider ID", http.StatusBadRequest)
		return
	}

	var req ToggleProviderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.ssoService.SetOIDCProviderEnabled(r.Context(), providerID, req.Enabled); err != nil {
		h.logger.Error("Failed to toggle OIDC provider status", zap.Error(err))
		http.Error(w, "Failed to update OIDC provider status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"enabled": req.Enabled,
	})
}

func hasEnabledSAML(providers []*sso.SAMLProvider) bool {
	for _, provider := range providers {
		if provider.Enabled {
			return true
		}
	}
	return false
}

func hasEnabledOIDC(providers []*sso.OpenIDConnectProvider) bool {
	for _, provider := range providers {
		if provider.Enabled {
			return true
		}
	}
	return false
}
