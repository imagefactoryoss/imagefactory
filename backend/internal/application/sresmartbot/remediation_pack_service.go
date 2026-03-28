package sresmartbot

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
)

type RemediationPack struct {
	Key              string   `json:"key"`
	Version          string   `json:"version"`
	Name             string   `json:"name"`
	Summary          string   `json:"summary"`
	RiskTier         string   `json:"risk_tier"`
	ActionClass      string   `json:"action_class"`
	RequiresApproval bool     `json:"requires_approval"`
	IncidentTypes    []string `json:"incident_types"`
}

type RemediationPackService struct {
	policyReader  robotSREPolicyReader
	fallbackPacks []RemediationPack
}

type robotSREPolicyReader interface {
	GetRobotSREPolicyConfig(ctx context.Context, tenantID *uuid.UUID) (*systemconfig.RobotSREPolicyConfig, error)
}

func NewRemediationPackService(policyReader robotSREPolicyReader) *RemediationPackService {
	return &RemediationPackService{
		policyReader:  policyReader,
		fallbackPacks: defaultRemediationPacks(),
	}
}

func (s *RemediationPackService) ListPacks(ctx context.Context, tenantID *uuid.UUID) []RemediationPack {
	packs := s.resolvePacks(ctx, tenantID)
	if len(packs) == 0 {
		return []RemediationPack{}
	}
	out := make([]RemediationPack, 0, len(packs))
	for _, pack := range packs {
		out = append(out, clonePack(pack))
	}
	return out
}

func (s *RemediationPackService) ListPacksForIncidentType(ctx context.Context, tenantID *uuid.UUID, incidentType string) []RemediationPack {
	normalizedType := strings.TrimSpace(strings.ToLower(incidentType))
	if normalizedType == "" {
		return []RemediationPack{}
	}
	packs := s.resolvePacks(ctx, tenantID)
	if len(packs) == 0 {
		return []RemediationPack{}
	}

	out := make([]RemediationPack, 0)
	for _, pack := range packs {
		for _, raw := range pack.IncidentTypes {
			if strings.ToLower(strings.TrimSpace(raw)) == normalizedType {
				out = append(out, clonePack(pack))
				break
			}
		}
	}
	return out
}

func (s *RemediationPackService) ResolvePackForIncidentType(ctx context.Context, tenantID *uuid.UUID, incidentType string, packKey string) (RemediationPack, bool) {
	key := strings.TrimSpace(packKey)
	if key == "" {
		return RemediationPack{}, false
	}
	packs := s.ListPacksForIncidentType(ctx, tenantID, incidentType)
	for _, pack := range packs {
		if strings.EqualFold(strings.TrimSpace(pack.Key), key) {
			return clonePack(pack), true
		}
	}
	return RemediationPack{}, false
}

func (s *RemediationPackService) resolvePacks(ctx context.Context, tenantID *uuid.UUID) []RemediationPack {
	if s == nil {
		return []RemediationPack{}
	}
	if s.policyReader != nil {
		if cfg, err := s.policyReader.GetRobotSREPolicyConfig(ctx, tenantID); err == nil && cfg != nil {
			packs := packsFromPolicy(cfg)
			if len(packs) > 0 {
				return packs
			}
		}
	}
	return append([]RemediationPack(nil), s.fallbackPacks...)
}

func defaultRemediationPacks() []RemediationPack {
	defaultCfg := systemconfig.DefaultRobotSREPolicyConfig()
	return packsFromPolicy(&defaultCfg)
}

func packsFromPolicy(cfg *systemconfig.RobotSREPolicyConfig) []RemediationPack {
	if cfg == nil || len(cfg.RemediationPacks) == 0 {
		return []RemediationPack{}
	}
	packs := make([]RemediationPack, 0, len(cfg.RemediationPacks))
	for _, pack := range cfg.RemediationPacks {
		packs = append(packs, RemediationPack{
			Key:              pack.Key,
			Version:          pack.Version,
			Name:             pack.Name,
			Summary:          pack.Summary,
			RiskTier:         pack.RiskTier,
			ActionClass:      pack.ActionClass,
			RequiresApproval: pack.RequiresApproval,
			IncidentTypes:    append([]string(nil), pack.IncidentTypes...),
		})
	}
	sort.Slice(packs, func(i, j int) bool {
		return packs[i].Key < packs[j].Key
	})
	return packs
}

func clonePack(pack RemediationPack) RemediationPack {
	copyPack := pack
	copyPack.IncidentTypes = append([]string(nil), pack.IncidentTypes...)
	return copyPack
}
