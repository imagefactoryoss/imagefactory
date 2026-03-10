package buildnotification

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListProjectTriggerPreferences(ctx context.Context, tenantID, projectID uuid.UUID) ([]ProjectTriggerPreference, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if projectID == uuid.Nil {
		return nil, ErrInvalidProjectID
	}

	stored, err := s.repo.ListProjectTriggerPreferences(ctx, tenantID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list project trigger preferences: %w", err)
	}
	tenantDefaults, err := s.repo.ListTenantTriggerPreferences(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant trigger preferences: %w", err)
	}

	byTrigger := make(map[TriggerID]ProjectTriggerPreference, len(stored))
	for _, pref := range stored {
		byTrigger[pref.TriggerID] = pref
	}
	tenantByTrigger := make(map[TriggerID]TenantTriggerPreference, len(tenantDefaults))
	for _, pref := range tenantDefaults {
		tenantByTrigger[pref.TriggerID] = pref
	}

	result := make([]ProjectTriggerPreference, 0, len(AllTriggerIDs))
	for _, triggerID := range AllTriggerIDs {
		if pref, ok := byTrigger[triggerID]; ok {
			pref.Source = TriggerPreferenceSourceProject
			result = append(result, pref)
			continue
		}
		if tenantPref, ok := tenantByTrigger[triggerID]; ok {
			pref := ProjectTriggerPreference{
				ID:                   uuid.Nil,
				TenantID:             tenantID,
				ProjectID:            projectID,
				TriggerID:            triggerID,
				Source:               TriggerPreferenceSourceTenant,
				Enabled:              tenantPref.Enabled,
				Channels:             tenantPref.Channels,
				RecipientPolicy:      tenantPref.RecipientPolicy,
				CustomRecipientUsers: tenantPref.CustomRecipientUsers,
				SeverityOverride:     tenantPref.SeverityOverride,
			}
			result = append(result, pref)
			continue
		}
		pref := DefaultProjectTriggerPreference(tenantID, projectID, triggerID)
		pref.Source = TriggerPreferenceSourceSystem
		result = append(result, pref)
	}

	return result, nil
}

func (s *Service) ListTenantTriggerPreferences(ctx context.Context, tenantID uuid.UUID) ([]TenantTriggerPreference, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}

	stored, err := s.repo.ListTenantTriggerPreferences(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant trigger preferences: %w", err)
	}
	byTrigger := make(map[TriggerID]TenantTriggerPreference, len(stored))
	for _, pref := range stored {
		byTrigger[pref.TriggerID] = pref
	}

	result := make([]TenantTriggerPreference, 0, len(AllTriggerIDs))
	for _, triggerID := range AllTriggerIDs {
		if pref, ok := byTrigger[triggerID]; ok {
			pref.Source = TriggerPreferenceSourceTenant
			result = append(result, pref)
			continue
		}
		pref := DefaultTenantTriggerPreference(tenantID, triggerID)
		pref.Source = TriggerPreferenceSourceSystem
		result = append(result, pref)
	}
	return result, nil
}

func (s *Service) UpsertTenantTriggerPreferences(ctx context.Context, tenantID, actorID uuid.UUID, prefs []TenantTriggerPreference) error {
	if tenantID == uuid.Nil {
		return ErrInvalidTenantID
	}
	if actorID == uuid.Nil {
		return ErrInvalidUserID
	}

	for i := range prefs {
		prefs[i].TenantID = tenantID
		if err := validatePreferenceFields(
			prefs[i].TriggerID,
			prefs[i].Enabled,
			prefs[i].Channels,
			prefs[i].RecipientPolicy,
			prefs[i].CustomRecipientUsers,
			prefs[i].SeverityOverride,
		); err != nil {
			return err
		}
	}
	if err := s.repo.UpsertTenantTriggerPreferences(ctx, tenantID, actorID, prefs); err != nil {
		return fmt.Errorf("failed to persist tenant trigger preferences: %w", err)
	}
	return nil
}

func (s *Service) UpsertProjectTriggerPreferences(ctx context.Context, tenantID, projectID, actorID uuid.UUID, prefs []ProjectTriggerPreference) error {
	if tenantID == uuid.Nil {
		return ErrInvalidTenantID
	}
	if projectID == uuid.Nil {
		return ErrInvalidProjectID
	}
	if actorID == uuid.Nil {
		return ErrInvalidUserID
	}

	for i := range prefs {
		prefs[i].TenantID = tenantID
		prefs[i].ProjectID = projectID
		if err := validatePreference(prefs[i]); err != nil {
			return err
		}
	}

	if err := s.repo.UpsertProjectTriggerPreferences(ctx, tenantID, projectID, actorID, prefs); err != nil {
		return fmt.Errorf("failed to persist project trigger preferences: %w", err)
	}
	return nil
}

func (s *Service) DeleteProjectTriggerPreference(ctx context.Context, tenantID, projectID uuid.UUID, triggerID TriggerID) error {
	if tenantID == uuid.Nil {
		return ErrInvalidTenantID
	}
	if projectID == uuid.Nil {
		return ErrInvalidProjectID
	}
	if !isValidTriggerID(triggerID) {
		return ErrInvalidTriggerID
	}
	if err := s.repo.DeleteProjectTriggerPreference(ctx, tenantID, projectID, triggerID); err != nil {
		return fmt.Errorf("failed to delete project trigger preference: %w", err)
	}
	return nil
}

func validatePreference(pref ProjectTriggerPreference) error {
	return validatePreferenceFields(pref.TriggerID, pref.Enabled, pref.Channels, pref.RecipientPolicy, pref.CustomRecipientUsers, pref.SeverityOverride)
}

func validatePreferenceFields(triggerID TriggerID, enabled bool, channels []Channel, recipientPolicy RecipientPolicy, customRecipientUsers []uuid.UUID, severityOverride *Severity) error {
	if !isValidTriggerID(triggerID) {
		return ErrInvalidTriggerID
	}

	for _, channel := range channels {
		if channel != ChannelInApp && channel != ChannelEmail {
			return fmt.Errorf("invalid channel: %s", channel)
		}
	}
	if enabled && len(channels) == 0 {
		return errors.New("at least one channel is required when trigger is enabled")
	}

	switch recipientPolicy {
	case RecipientInitiator, RecipientProjectMember, RecipientTenantAdmins, RecipientCustomUsers:
	default:
		return fmt.Errorf("invalid recipient policy: %s", recipientPolicy)
	}

	if severityOverride != nil {
		s := *severityOverride
		if s != SeverityLow && s != SeverityNormal && s != SeverityHigh {
			return fmt.Errorf("invalid severity override: %s", s)
		}
	}

	if recipientPolicy == RecipientCustomUsers && len(customRecipientUsers) == 0 {
		return errors.New("at least one custom recipient user is required for custom_users recipient policy")
	}

	return nil
}

func isValidTriggerID(triggerID TriggerID) bool {
	for _, known := range AllTriggerIDs {
		if known == triggerID {
			return true
		}
	}
	return false
}
