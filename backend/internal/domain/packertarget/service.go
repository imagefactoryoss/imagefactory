package packertarget

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*Profile, error) {
	if req.TenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	if req.CreatedBy == uuid.Nil {
		return nil, ErrInvalidUser
	}
	if err := validateName(req.Name); err != nil {
		return nil, err
	}
	if err := validateProvider(req.Provider); err != nil {
		return nil, err
	}
	if err := validateSecretRef(req.SecretRef); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	p := &Profile{
		ID:               uuid.New(),
		TenantID:         req.TenantID,
		IsGlobal:         req.IsGlobal,
		Name:             normalizeName(req.Name),
		Provider:         normalizeProvider(req.Provider),
		Description:      strings.TrimSpace(req.Description),
		SecretRef:        normalizeSecretRef(req.SecretRef),
		Options:          copyOptions(req.Options),
		ValidationStatus: ValidationStatusUntested,
		CreatedBy:        req.CreatedBy,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) List(ctx context.Context, tenantID uuid.UUID, allTenants bool, provider string) ([]*Profile, error) {
	if !allTenants && tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	provider = normalizeProvider(provider)
	if provider != "" {
		if err := validateProvider(provider); err != nil {
			return nil, err
		}
	}
	return s.repo.List(ctx, tenantID, allTenants, provider)
}

func (s *Service) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Profile, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *Service) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateRequest) (*Profile, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	current, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		if err := validateName(*req.Name); err != nil {
			return nil, err
		}
		current.Name = normalizeName(*req.Name)
	}
	if req.Description != nil {
		current.Description = strings.TrimSpace(*req.Description)
	}
	if req.SecretRef != nil {
		if err := validateSecretRef(*req.SecretRef); err != nil {
			return nil, err
		}
		current.SecretRef = normalizeSecretRef(*req.SecretRef)
	}
	if req.Options != nil {
		current.Options = copyOptions(*req.Options)
	}
	if req.IsGlobal != nil {
		current.IsGlobal = *req.IsGlobal
	}

	current.ValidationStatus = ValidationStatusUntested
	current.LastValidatedAt = nil
	current.LastValidationMessage = nil
	current.LastRemediationHints = nil
	current.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, current); err != nil {
		return nil, err
	}
	return current, nil
}

func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if tenantID == uuid.Nil {
		return ErrInvalidTenant
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *Service) Validate(ctx context.Context, tenantID, id uuid.UUID) (ValidationResult, error) {
	if tenantID == uuid.Nil {
		return ValidationResult{}, ErrInvalidTenant
	}
	profile, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return ValidationResult{}, err
	}
	result := deterministicValidate(profile)
	if err := s.repo.UpdateValidation(ctx, tenantID, id, result); err != nil {
		return ValidationResult{}, err
	}
	return result, nil
}

func deterministicValidate(profile *Profile) ValidationResult {
	checkedAt := time.Now().UTC()
	checks := make([]ValidationCheck, 0, 6)
	checks = append(checks, makeSecretRefCheck(profile.SecretRef))

	required := requiredProviderOptionKeys(profile.Provider)
	for _, key := range required {
		val, ok := profile.Options[key]
		if !ok || strings.TrimSpace(fmt.Sprintf("%v", val)) == "" {
			checks = append(checks, ValidationCheck{
				Name:            fmt.Sprintf("option:%s", key),
				OK:              false,
				Message:         fmt.Sprintf("missing required option %q", key),
				RemediationHint: fmt.Sprintf("set %q in profile options", key),
			})
			continue
		}
		checks = append(checks, ValidationCheck{
			Name:    fmt.Sprintf("option:%s", key),
			OK:      true,
			Message: fmt.Sprintf("required option %q is set", key),
		})
	}

	status := ValidationStatusValid
	message := "profile passed deterministic validation"
	hints := make([]string, 0)
	for _, check := range checks {
		if check.OK {
			continue
		}
		status = ValidationStatusInvalid
		if check.RemediationHint != "" {
			hints = append(hints, check.RemediationHint)
		}
	}
	if status == ValidationStatusInvalid {
		message = "profile failed deterministic validation"
		hints = uniqueSortedStrings(hints)
	}

	return ValidationResult{
		ProfileID:        profile.ID,
		Provider:         profile.Provider,
		Status:           status,
		CheckedAt:        checkedAt,
		Checks:           checks,
		Message:          message,
		RemediationHints: hints,
	}
}

func requiredProviderOptionKeys(provider string) []string {
	switch normalizeProvider(provider) {
	case ProviderVMware:
		return []string{"vcenter_server", "datacenter", "cluster", "datastore", "network"}
	case ProviderAWS:
		return []string{"region", "source_ami", "instance_type", "subnet_id"}
	case ProviderAzure:
		return []string{"location", "resource_group", "subscription_id", "virtual_network_name"}
	case ProviderGCP:
		return []string{"project_id", "zone", "source_image_family", "network"}
	default:
		return nil
	}
}

func makeSecretRefCheck(secretRef string) ValidationCheck {
	if validateSecretRef(secretRef) != nil {
		return ValidationCheck{
			Name:            "secret_ref",
			OK:              false,
			Message:         "secret_ref must be set as namespaced identifier",
			RemediationHint: "set secret_ref in the format <namespace>/<name>",
		}
	}
	return ValidationCheck{Name: "secret_ref", OK: true, Message: "secret_ref format is valid"}
}

func copyOptions(in map[string]interface{}) map[string]interface{} {
	if len(in) == 0 {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		seen[value] = struct{}{}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
