package sso

import (
	"context"
	"encoding/xml"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SSOPosition represents different SSO provider types
type SSOPosition string

const (
	SSOPositionServiceProvider  SSOPosition = "service_provider"
	SSOPositionIdentityProvider SSOPosition = "identity_provider"
)

// SAMLProvider represents SAML 2.0 provider configuration
type SAMLProvider struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	EntityID    string                 `json:"entity_id"`
	MetadataURL string                 `json:"metadata_url"`
	SSOURL      string                 `json:"sso_url"`
	SLOURL      string                 `json:"slo_url"`
	Certificate string                 `json:"certificate"`
	PrivateKey  string                 `json:"private_key"`
	Position    SSOPosition            `json:"position"`
	Metadata    string                 `json:"metadata"`
	Attributes  map[string]interface{} `json:"attributes"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// OpenIDConnectProvider represents OpenID Connect provider configuration
type OpenIDConnectProvider struct {
	ID               uuid.UUID              `json:"id"`
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
	Attributes       map[string]interface{} `json:"attributes"`
	Enabled          bool                   `json:"enabled"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// SAMLMetadata represents SAML metadata
type SAMLMetadata struct {
	XMLName                      xml.Name                         `xml:"urn:oasis:names:tc:SAML:2.0:metadata EntityDescriptor"`
	EntityID                     string                           `xml:"entityID,attr"`
	SSODescriptor                SAMLSSODescriptor                `xml:"IDPSSODescriptor"`
	AttributeAuthorityDescriptor SAMLAttributeAuthorityDescriptor `xml:"AttributeAuthorityDescriptor"`
}

// SAMLSSODescriptor represents SAML SSO descriptor
type SAMLSSODescriptor struct {
	XMLName      xml.Name          `xml:"urn:oasis:names:tc:SAML:2.0:metadata IDPSSODescriptor"`
	NameIDFormat string            `xml:"NameIDFormat"`
	SSOURL       string            `xml:"SingleSignOnService Binding=\"urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect\" Location"`
	SLOService   string            `xml:"SingleLogoutService Binding=\"urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect\" Location"`
	Certificates []SAMLCertificate `xml:"KeyDescriptor KeyInfo X509Data X509Certificate"`
}

// SAMLAttributeAuthorityDescriptor represents SAML attribute authority descriptor
type SAMLAttributeAuthorityDescriptor struct {
	XMLName          xml.Name          `xml:"urn:oasis:names:tc:SAML:2.0:metadata AttributeAuthorityDescriptor"`
	AttributeService string            `xml:"AttributeService Binding=\"urn:oasis:names:tc:SAML:2.0:bindings:SOAP\" Location"`
	AttributeProfile string            `xml:"samlp:AttributeQuery"`
	Certificates     []SAMLCertificate `xml:"KeyDescriptor KeyInfo X509Data X509Certificate"`
}

// SAMLCertificate represents SAML certificate
type SAMLCertificate struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:metadata X509Certificate"`
	Content string   `xml:",innerxml"`
}

// SAMLAssertion represents SAML assertion
type SAMLAssertion struct {
	XMLName            xml.Name               `xml:"urn:oasis:names:tc:SAML:2.0:assertion Assertion"`
	ID                 string                 `xml:"ID,attr"`
	IssueInstant       string                 `xml:"IssueInstant,attr"`
	Version            string                 `xml:"Version,attr"`
	Issuer             SAMLIssuer             `xml:"Issuer"`
	Subject            SAMlSubject            `xml:"Subject"`
	Conditions         SAMLConditions         `xml:"Conditions"`
	AttributeStatement SAMLAttributeStatement `xml:"AttributeStatement"`
}

// SAMLIssuer represents SAML issuer
type SAMLIssuer struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion Issuer"`
	Content string   `xml:",innerhtml"`
}

// SAMlSubject represents SAML subject
type SAMlSubject struct {
	XMLName             xml.Name                `xml:"urn:oasis:names:tc:SAML:2.0:assertion Subject"`
	NameID              SAMLNameID              `xml:"NameID"`
	SubjectConfirmation SAMLSubjectConfirmation `xml:"SubjectConfirmation"`
}

// SAMLNameID represents SAML NameID
type SAMLNameID struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion NameID"`
	Format  string   `xml:"Format,attr"`
	Value   string   `xml:",innerhtml"`
}

// SAMLSubjectConfirmation represents SAML subject confirmation
type SAMLSubjectConfirmation struct {
	XMLName          xml.Name             `xml:"urn:oasis:names:tc:SAML:2.0:assertion SubjectConfirmation"`
	Method           string               `xml:"Method,attr"`
	ConfirmationData SAMLConfirmationData `xml:"SubjectConfirmationData"`
}

// SAMLConfirmationData represents SAML confirmation data
type SAMLConfirmationData struct {
	XMLName      xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion SubjectConfirmationData"`
	NotBefore    string   `xml:"NotBefore,attr"`
	NotOnOrAfter string   `xml:"NotOnOrAfter,attr"`
	Recipient    string   `xml:"Recipient,attr"`
	// Could include InResponseTo attribute
}

// SAMLConditions represents SAML conditions
type SAMLConditions struct {
	XMLName             xml.Name                  `xml:"urn:oasis:names:tc:SAML:2.0:assertion Conditions"`
	NotBefore           string                    `xml:"NotBefore,attr"`
	NotOnOrAfter        string                    `xml:"NotOnOrAfter,attr"`
	AudienceRestriction []SAMLAudienceRestriction `xml:"AudienceRestriction"`
}

// SAMLAudienceRestriction represents SAML audience restriction
type SAMLAudienceRestriction struct {
	XMLName  xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion AudienceRestriction"`
	Audience string   `xml:"Audience"`
}

// SAMLAttributeStatement represents SAML attribute statement
type SAMLAttributeStatement struct {
	XMLName    xml.Name        `xml:"urn:oasis:names:tc:SAML:2.0:assertion AttributeStatement"`
	Attributes []SAMLAttribute `xml:"Attribute"`
}

// SAMLAttribute represents SAML attribute
type SAMLAttribute struct {
	XMLName    xml.Name             `xml:"urn:oasis:names:tc:SAML:2.0:assertion Attribute"`
	Name       string               `xml:"Name,attr"`
	NameFormat string               `xml:"NameFormat,attr"`
	Values     []SAMLAttributeValue `xml:"AttributeValue"`
}

// SAMLAttributeValue represents SAML attribute value
type SAMLAttributeValue struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion AttributeValue"`
	Value   string   `xml:",innerhtml"`
}

// OpenIDConnectToken represents OpenID Connect token response
type OpenIDConnectToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// OpenIDConnectUserInfo represents OpenID Connect user info response
type OpenIDConnectUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	GivenName     string `json:"given_name,omitempty"`
	FamilyName    string `json:"family_name,omitempty"`
	Profile       string `json:"profile,omitempty"`
	Picture       string `json:"picture,omitempty"`
	Locale        string `json:"locale,omitempty"`
}

// Service represents the SSO service
type Service struct {
	samlRepository SAMLRepository
	oidcRepository OIDCRepository
	logger         *zap.Logger
}

// SAMLRepository represents the interface for SAML provider persistence
type SAMLRepository interface {
	Save(ctx context.Context, provider *SAMLProvider) error
	Update(ctx context.Context, provider *SAMLProvider) error
	FindByID(ctx context.Context, id uuid.UUID) (*SAMLProvider, error)
	FindByEntityID(ctx context.Context, entityID string) (*SAMLProvider, error)
	FindAll(ctx context.Context) ([]*SAMLProvider, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// OIDCRepository represents the interface for OpenID Connect provider persistence
type OIDCRepository interface {
	Save(ctx context.Context, provider *OpenIDConnectProvider) error
	Update(ctx context.Context, provider *OpenIDConnectProvider) error
	FindByID(ctx context.Context, id uuid.UUID) (*OpenIDConnectProvider, error)
	FindByIssuer(ctx context.Context, issuer string) (*OpenIDConnectProvider, error)
	FindAll(ctx context.Context) ([]*OpenIDConnectProvider, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// NewService creates a new SSO service
func NewService(
	samlRepository SAMLRepository,
	oidcRepository OIDCRepository,
	logger *zap.Logger,
) *Service {
	return &Service{
		samlRepository: samlRepository,
		oidcRepository: oidcRepository,
		logger:         logger,
	}
}

// SAMLProviderCreateRequest represents a request to create a SAML provider
type SAMLProviderCreateRequest struct {
	Name        string                 `json:"name"`
	EntityID    string                 `json:"entity_id"`
	SSOURL      string                 `json:"sso_url"`
	SLOURL      string                 `json:"slo_url"`
	Certificate string                 `json:"certificate"`
	PrivateKey  string                 `json:"private_key"`
	Position    SSOPosition            `json:"position"`
	Attributes  map[string]interface{} `json:"attributes"`
	Enabled     *bool                  `json:"enabled,omitempty"`
}

// OIDCProviderCreateRequest represents a request to create an OpenID Connect provider
type OIDCProviderCreateRequest struct {
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
	Attributes       map[string]interface{} `json:"attributes"`
	Enabled          *bool                  `json:"enabled,omitempty"`
}

// CreateSAMLProvider creates a new SAML provider
func (s *Service) CreateSAMLProvider(ctx context.Context, req SAMLProviderCreateRequest) (*SAMLProvider, error) {
	// Validate required fields
	if err := s.validateSAMLProviderRequest(req); err != nil {
		return nil, err
	}

	// Generate metadata if needed
	metadata, err := s.generateSAMLMetadata(req)
	if err != nil {
		s.logger.Error("Failed to generate SAML metadata", zap.Error(err))
		return nil, err
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	provider := &SAMLProvider{
		ID:          uuid.New(),
		Name:        req.Name,
		EntityID:    req.EntityID,
		SSOURL:      req.SSOURL,
		SLOURL:      req.SLOURL,
		Certificate: req.Certificate,
		PrivateKey:  req.PrivateKey,
		Position:    req.Position,
		Metadata:    metadata,
		Attributes:  req.Attributes,
		Enabled:     enabled,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.samlRepository.Save(ctx, provider); err != nil {
		s.logger.Error("Failed to save SAML provider", zap.Error(err))
		return nil, err
	}

	if provider.Enabled {
		if err := s.SetSAMLProviderEnabled(ctx, provider.ID, true); err != nil {
			return nil, err
		}
	}

	s.logger.Info("SAML provider created successfully",
		zap.String("provider_id", provider.ID.String()),
		zap.String("entity_id", provider.EntityID))

	return provider, nil
}

// CreateOIDCProvider creates a new OpenID Connect provider
func (s *Service) CreateOIDCProvider(ctx context.Context, req OIDCProviderCreateRequest) (*OpenIDConnectProvider, error) {
	// Validate required fields
	if err := s.validateOIDCProviderRequest(req); err != nil {
		return nil, err
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	provider := &OpenIDConnectProvider{
		ID:               uuid.New(),
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
		Enabled:          enabled,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	if err := s.oidcRepository.Save(ctx, provider); err != nil {
		s.logger.Error("Failed to save OIDC provider", zap.Error(err))
		return nil, err
	}

	if provider.Enabled {
		if err := s.SetOIDCProviderEnabled(ctx, provider.ID, true); err != nil {
			return nil, err
		}
	}

	s.logger.Info("OIDC provider created successfully",
		zap.String("provider_id", provider.ID.String()),
		zap.String("issuer", provider.Issuer))

	return provider, nil
}

// GenerateSAMLMetadata generates SAML metadata for a provider
func (s *Service) generateSAMLMetadata(req SAMLProviderCreateRequest) (string, error) {
	// Generate unique entity ID if not provided
	entityID := req.EntityID
	if entityID == "" {
		entityID = "urn:image-factory:saml:" + uuid.New().String()
	}

	metadata := SAMLMetadata{
		EntityID: entityID,
		SSODescriptor: SAMLSSODescriptor{
			NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
			SSOURL:       req.SSOURL,
			SLOService:   req.SLOURL,
			Certificates: []SAMLCertificate{
				{
					Content: req.Certificate,
				},
			},
		},
		AttributeAuthorityDescriptor: SAMLAttributeAuthorityDescriptor{
			AttributeService: req.SSOURL,
			AttributeProfile: "urn:oasis:names:tc:SAML:2.0:attrname-format:basic",
			Certificates: []SAMLCertificate{
				{
					Content: req.Certificate,
				},
			},
		},
	}

	// Convert to XML
	xmlData, err := xml.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", err
	}

	return string(xmlData), nil
}

// validateSAMLProviderRequest validates SAML provider creation request
func (s *Service) validateSAMLProviderRequest(req SAMLProviderCreateRequest) error {
	if req.Name == "" {
		return errors.New("name is required")
	}
	if req.SSOURL == "" {
		return errors.New("SSO URL is required")
	}
	if req.Certificate == "" {
		return errors.New("certificate is required")
	}

	// Validate URLs
	if _, err := url.Parse(req.SSOURL); err != nil {
		return errors.New("invalid SSO URL")
	}
	if req.SLOURL != "" {
		if _, err := url.Parse(req.SLOURL); err != nil {
			return errors.New("invalid SLO URL")
		}
	}

	// Validate certificate format (basic check)
	if !strings.Contains(req.Certificate, "BEGIN CERTIFICATE") {
		return errors.New("invalid certificate format")
	}

	return nil
}

// validateOIDCProviderRequest validates OpenID Connect provider creation request
func (s *Service) validateOIDCProviderRequest(req OIDCProviderCreateRequest) error {
	if req.Name == "" {
		return errors.New("name is required")
	}
	if req.Issuer == "" {
		return errors.New("issuer is required")
	}
	if req.ClientID == "" {
		return errors.New("client ID is required")
	}
	if req.AuthorizationURL == "" {
		return errors.New("authorization URL is required")
	}
	if req.TokenURL == "" {
		return errors.New("token URL is required")
	}
	if len(req.RedirectURIs) == 0 {
		return errors.New("at least one redirect URI is required")
	}

	// Validate URLs
	if _, err := url.Parse(req.Issuer); err != nil {
		return errors.New("invalid issuer URL")
	}
	if _, err := url.Parse(req.AuthorizationURL); err != nil {
		return errors.New("invalid authorization URL")
	}
	if _, err := url.Parse(req.TokenURL); err != nil {
		return errors.New("invalid token URL")
	}

	for _, uri := range req.RedirectURIs {
		if _, err := url.Parse(uri); err != nil {
			return errors.New("invalid redirect URI: " + uri)
		}
	}

	return nil
}

// GetAllSAMLProviders returns all SAML providers
func (s *Service) GetAllSAMLProviders(ctx context.Context) ([]*SAMLProvider, error) {
	return s.samlRepository.FindAll(ctx)
}

// GetAllOIDCProviders returns all OpenID Connect providers
func (s *Service) GetAllOIDCProviders(ctx context.Context) ([]*OpenIDConnectProvider, error) {
	return s.oidcRepository.FindAll(ctx)
}

// SetSAMLProviderEnabled toggles a SAML provider and enforces single-active-provider across SSO.
func (s *Service) SetSAMLProviderEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	provider, err := s.samlRepository.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if !enabled {
		provider.Enabled = false
		provider.UpdatedAt = time.Now().UTC()
		return s.samlRepository.Update(ctx, provider)
	}

	samlProviders, err := s.samlRepository.FindAll(ctx)
	if err != nil {
		return err
	}
	for _, p := range samlProviders {
		p.Enabled = p.ID == id
		p.UpdatedAt = time.Now().UTC()
		if err := s.samlRepository.Update(ctx, p); err != nil {
			return err
		}
	}

	oidcProviders, err := s.oidcRepository.FindAll(ctx)
	if err != nil {
		return err
	}
	for _, p := range oidcProviders {
		if p.Enabled {
			p.Enabled = false
			p.UpdatedAt = time.Now().UTC()
			if err := s.oidcRepository.Update(ctx, p); err != nil {
				return err
			}
		}
	}

	return nil
}

// SetOIDCProviderEnabled toggles an OIDC provider and enforces single-active-provider across SSO.
func (s *Service) SetOIDCProviderEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	provider, err := s.oidcRepository.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if !enabled {
		provider.Enabled = false
		provider.UpdatedAt = time.Now().UTC()
		return s.oidcRepository.Update(ctx, provider)
	}

	oidcProviders, err := s.oidcRepository.FindAll(ctx)
	if err != nil {
		return err
	}
	for _, p := range oidcProviders {
		p.Enabled = p.ID == id
		p.UpdatedAt = time.Now().UTC()
		if err := s.oidcRepository.Update(ctx, p); err != nil {
			return err
		}
	}

	samlProviders, err := s.samlRepository.FindAll(ctx)
	if err != nil {
		return err
	}
	for _, p := range samlProviders {
		if p.Enabled {
			p.Enabled = false
			p.UpdatedAt = time.Now().UTC()
			if err := s.samlRepository.Update(ctx, p); err != nil {
				return err
			}
		}
	}

	return nil
}
