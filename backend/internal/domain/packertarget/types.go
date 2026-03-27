package packertarget

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ProviderVMware = "vmware"
	ProviderAWS    = "aws"
	ProviderAzure  = "azure"
	ProviderGCP    = "gcp"
)

const (
	ValidationStatusUntested = "untested"
	ValidationStatusValid    = "valid"
	ValidationStatusInvalid  = "invalid"
)

var (
	ErrNotFound         = errors.New("packer target profile not found")
	ErrInvalidTenant    = errors.New("invalid tenant id")
	ErrInvalidUser      = errors.New("invalid user id")
	ErrInvalidName      = errors.New("invalid profile name")
	ErrInvalidProvider  = errors.New("invalid provider")
	ErrInvalidSecretRef = errors.New("invalid secret reference")
)

type Profile struct {
	ID                    uuid.UUID              `json:"id"`
	TenantID              uuid.UUID              `json:"tenant_id"`
	IsGlobal              bool                   `json:"is_global"`
	Name                  string                 `json:"name"`
	Provider              string                 `json:"provider"`
	Description           string                 `json:"description,omitempty"`
	SecretRef             string                 `json:"secret_ref"`
	Options               map[string]interface{} `json:"options,omitempty"`
	ValidationStatus      string                 `json:"validation_status"`
	LastValidatedAt       *time.Time             `json:"last_validated_at,omitempty"`
	LastValidationMessage *string                `json:"last_validation_message,omitempty"`
	LastRemediationHints  []string               `json:"last_remediation_hints,omitempty"`
	CreatedBy             uuid.UUID              `json:"created_by"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

type CreateRequest struct {
	TenantID    uuid.UUID
	IsGlobal    bool
	Name        string
	Provider    string
	Description string
	SecretRef   string
	Options     map[string]interface{}
	CreatedBy   uuid.UUID
}

type UpdateRequest struct {
	Name        *string
	Description *string
	SecretRef   *string
	Options     *map[string]interface{}
	IsGlobal    *bool
}

type ValidationCheck struct {
	Name            string `json:"name"`
	OK              bool   `json:"ok"`
	Message         string `json:"message,omitempty"`
	RemediationHint string `json:"remediation_hint,omitempty"`
}

type ValidationResult struct {
	ProfileID        uuid.UUID         `json:"profile_id"`
	Provider         string            `json:"provider"`
	Status           string            `json:"status"`
	CheckedAt        time.Time         `json:"checked_at"`
	Checks           []ValidationCheck `json:"checks"`
	Message          string            `json:"message"`
	RemediationHints []string          `json:"remediation_hints"`
}

func normalizeProvider(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func normalizeName(raw string) string {
	return strings.TrimSpace(raw)
}

func normalizeSecretRef(raw string) string {
	return strings.TrimSpace(raw)
}

func validateProvider(provider string) error {
	switch normalizeProvider(provider) {
	case ProviderVMware, ProviderAWS, ProviderAzure, ProviderGCP:
		return nil
	default:
		return ErrInvalidProvider
	}
}

func validateName(name string) error {
	name = normalizeName(name)
	if name == "" || len(name) > 120 {
		return ErrInvalidName
	}
	return nil
}

func validateSecretRef(secretRef string) error {
	secretRef = normalizeSecretRef(secretRef)
	if secretRef == "" || len(secretRef) > 255 || !strings.Contains(secretRef, "/") {
		return ErrInvalidSecretRef
	}
	return nil
}
