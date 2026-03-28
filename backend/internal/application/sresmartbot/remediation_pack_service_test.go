package sresmartbot

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
)

type stubRobotSREPolicyReader struct {
	config *systemconfig.RobotSREPolicyConfig
	err    error
}

func (s *stubRobotSREPolicyReader) GetRobotSREPolicyConfig(ctx context.Context, tenantID *uuid.UUID) (*systemconfig.RobotSREPolicyConfig, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.config, nil
}

func TestRemediationPackServiceListPacksSorted(t *testing.T) {
	service := NewRemediationPackService(nil)
	packs := service.ListPacks(context.Background(), nil)

	if len(packs) < 3 {
		t.Fatalf("expected at least 3 remediation packs, got %d", len(packs))
	}

	for i := 1; i < len(packs); i++ {
		if packs[i-1].Key > packs[i].Key {
			t.Fatalf("expected sorted pack keys, got %q before %q", packs[i-1].Key, packs[i].Key)
		}
	}
}

func TestRemediationPackServiceListPacksForIncidentType(t *testing.T) {
	service := NewRemediationPackService(nil)

	tests := []struct {
		name         string
		incidentType string
		expectedKey  string
	}{
		{name: "nats incident type", incidentType: "nats_transport_disconnect_storm", expectedKey: "nats_transport_stability_pack"},
		{name: "async incident type", incidentType: "email_queue_backlog_pressure", expectedKey: "async_backlog_pressure_pack"},
		{name: "provider incident type", incidentType: "provider_connectivity_degraded", expectedKey: "provider_connectivity_drift_pack"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packs := service.ListPacksForIncidentType(context.Background(), nil, tt.incidentType)
			if len(packs) != 1 {
				t.Fatalf("expected 1 remediation pack for %q, got %d", tt.incidentType, len(packs))
			}
			if packs[0].Key != tt.expectedKey {
				t.Fatalf("expected pack %q, got %q", tt.expectedKey, packs[0].Key)
			}
		})
	}

	t.Run("unknown type returns empty", func(t *testing.T) {
		packs := service.ListPacksForIncidentType(context.Background(), nil, "does_not_exist")
		if len(packs) != 0 {
			t.Fatalf("expected 0 packs for unknown incident type, got %d", len(packs))
		}
	})
}

func TestRemediationPackServiceUsesPolicyConfigWhenAvailable(t *testing.T) {
	service := NewRemediationPackService(&stubRobotSREPolicyReader{
		config: &systemconfig.RobotSREPolicyConfig{
			RemediationPacks: []systemconfig.RobotSRERemediationPack{
				{
					Key:           "custom_pack",
					Version:       "v2",
					Name:          "Custom",
					Summary:       "Custom summary",
					RiskTier:      "high",
					ActionClass:   "guided_remediation",
					IncidentTypes: []string{"custom_incident"},
				},
			},
		},
	})

	packs := service.ListPacks(context.Background(), nil)
	if len(packs) != 1 || packs[0].Key != "custom_pack" {
		t.Fatalf("expected custom policy pack, got %#v", packs)
	}
}

func TestRemediationPackServiceFallsBackWhenPolicyReadFails(t *testing.T) {
	service := NewRemediationPackService(&stubRobotSREPolicyReader{err: errors.New("read failed")})

	packs := service.ListPacks(context.Background(), nil)
	if len(packs) < 3 {
		t.Fatalf("expected fallback packs when policy read fails, got %d", len(packs))
	}
}
