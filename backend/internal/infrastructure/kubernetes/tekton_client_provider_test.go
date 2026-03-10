package kubernetes

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
)

func TestSelectKubernetesProvider_WhenNoProviderID_UsesGlobalOnly(t *testing.T) {
	globalID := uuid.New()
	tenantID := uuid.New()
	now := time.Now()

	global := &infrastructure.Provider{
		ID:           globalID,
		TenantID:     tenantID,
		IsGlobal:     true,
		ProviderType: infrastructure.ProviderTypeKubernetes,
		Name:         "global-k8s",
		Config: map[string]interface{}{
			"runtime_auth": map[string]interface{}{
				"auth_method": "token",
				"endpoint":    "https://k8s.example",
				"token":       "token",
			},
		},
		CreatedAt: now.Add(-time.Hour),
	}

	nonGlobal := &infrastructure.Provider{
		ID:           uuid.New(),
		TenantID:     tenantID,
		IsGlobal:     false,
		ProviderType: infrastructure.ProviderTypeKubernetes,
		Name:         "tenant-k8s",
		Config: map[string]interface{}{
			"runtime_auth": map[string]interface{}{
				"auth_method": "token",
				"endpoint":    "https://tenant-k8s.example",
				"token":       "token",
			},
		},
		CreatedAt: now.Add(-2 * time.Hour),
	}

	selected, err := selectKubernetesProvider([]*infrastructure.Provider{nonGlobal, global}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected == nil || selected.ID != globalID {
		t.Fatalf("expected global provider to be selected")
	}
}

func TestSelectKubernetesProvider_WhenNoProviderID_AndNoGlobal_ReturnsError(t *testing.T) {
	tenantID := uuid.New()

	nonGlobal := &infrastructure.Provider{
		ID:           uuid.New(),
		TenantID:     tenantID,
		IsGlobal:     false,
		ProviderType: infrastructure.ProviderTypeKubernetes,
		Name:         "tenant-k8s",
		Config: map[string]interface{}{
			"runtime_auth": map[string]interface{}{
				"auth_method": "token",
				"endpoint":    "https://tenant-k8s.example",
				"token":       "token",
			},
		},
		CreatedAt: time.Now(),
	}

	_, err := selectKubernetesProvider([]*infrastructure.Provider{nonGlobal}, nil, nil)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestSelectKubernetesProvider_WhenNoProviderID_PrefersLowerLoadScore(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now()

	olderButHigherLoad := &infrastructure.Provider{
		ID:           uuid.New(),
		TenantID:     tenantID,
		IsGlobal:     true,
		ProviderType: infrastructure.ProviderTypeKubernetes,
		Name:         "global-high-load",
		Config: map[string]interface{}{
			"runtime_auth": map[string]interface{}{
				"auth_method": "token",
				"endpoint":    "https://high-load.example",
				"token":       "token",
			},
			"routing_load_score": 80.0,
		},
		CreatedAt: now.Add(-2 * time.Hour),
	}

	newerButLowerLoad := &infrastructure.Provider{
		ID:           uuid.New(),
		TenantID:     tenantID,
		IsGlobal:     true,
		ProviderType: infrastructure.ProviderTypeKubernetes,
		Name:         "global-low-load",
		Config: map[string]interface{}{
			"runtime_auth": map[string]interface{}{
				"auth_method": "token",
				"endpoint":    "https://low-load.example",
				"token":       "token",
			},
			"routing_load_score": 10.0,
		},
		CreatedAt: now.Add(-time.Hour),
	}

	selected, err := selectKubernetesProvider([]*infrastructure.Provider{olderButHigherLoad, newerButLowerLoad}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected == nil || selected.ID != newerButLowerLoad.ID {
		t.Fatalf("expected lower-load provider to be selected")
	}
}

func TestSelectKubernetesProvider_WhenNoProviderID_UsesRoutingNestedQueueDepth(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now()

	globalA := &infrastructure.Provider{
		ID:           uuid.New(),
		TenantID:     tenantID,
		IsGlobal:     true,
		ProviderType: infrastructure.ProviderTypeKubernetes,
		Name:         "global-a",
		Config: map[string]interface{}{
			"runtime_auth": map[string]interface{}{
				"auth_method": "token",
				"endpoint":    "https://a.example",
				"token":       "token",
			},
			"routing": map[string]interface{}{
				"queue_depth": "42",
			},
		},
		CreatedAt: now.Add(-2 * time.Hour),
	}

	globalB := &infrastructure.Provider{
		ID:           uuid.New(),
		TenantID:     tenantID,
		IsGlobal:     true,
		ProviderType: infrastructure.ProviderTypeKubernetes,
		Name:         "global-b",
		Config: map[string]interface{}{
			"runtime_auth": map[string]interface{}{
				"auth_method": "token",
				"endpoint":    "https://b.example",
				"token":       "token",
			},
			"routing": map[string]interface{}{
				"queue_depth": 2,
			},
		},
		CreatedAt: now.Add(-time.Hour),
	}

	selected, err := selectKubernetesProvider([]*infrastructure.Provider{globalA, globalB}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected == nil || selected.ID != globalB.ID {
		t.Fatalf("expected provider with lower queue depth to be selected")
	}
}

func TestSelectKubernetesProvider_WhenNoProviderID_RequiresCapabilities(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now()

	armProvider := &infrastructure.Provider{
		ID:           uuid.New(),
		TenantID:     tenantID,
		IsGlobal:     true,
		ProviderType: infrastructure.ProviderTypeKubernetes,
		Name:         "arm-provider",
		Capabilities: []string{"arm64"},
		Config: map[string]interface{}{
			"runtime_auth": map[string]interface{}{
				"auth_method": "token",
				"endpoint":    "https://arm.example",
				"token":       "token",
			},
		},
		CreatedAt: now.Add(-2 * time.Hour),
	}

	amdProvider := &infrastructure.Provider{
		ID:           uuid.New(),
		TenantID:     tenantID,
		IsGlobal:     true,
		ProviderType: infrastructure.ProviderTypeKubernetes,
		Name:         "amd-provider",
		Capabilities: []string{"amd64"},
		Config: map[string]interface{}{
			"runtime_auth": map[string]interface{}{
				"auth_method": "token",
				"endpoint":    "https://amd.example",
				"token":       "token",
			},
		},
		CreatedAt: now.Add(-time.Hour),
	}

	selected, err := selectKubernetesProvider([]*infrastructure.Provider{amdProvider, armProvider}, nil, []string{"arm64"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected == nil || selected.ID != armProvider.ID {
		t.Fatalf("expected arm64-capable provider to be selected")
	}
}

func TestSelectKubernetesProvider_WhenNoProviderID_CapabilityRequirementUnsatisfied(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now()

	global := &infrastructure.Provider{
		ID:           uuid.New(),
		TenantID:     tenantID,
		IsGlobal:     true,
		ProviderType: infrastructure.ProviderTypeKubernetes,
		Name:         "global-k8s",
		Capabilities: []string{"amd64"},
		Config: map[string]interface{}{
			"runtime_auth": map[string]interface{}{
				"auth_method": "token",
				"endpoint":    "https://global.example",
				"token":       "token",
			},
		},
		CreatedAt: now,
	}

	_, err := selectKubernetesProvider([]*infrastructure.Provider{global}, nil, []string{"gpu"})
	if err == nil {
		t.Fatalf("expected error for unsatisfied capability requirement")
	}
}

func TestRequiredFallbackCapabilities_IncludesExplicitMetadataCapabilities(t *testing.T) {
	buildEntity := build.NewBuildFromDB(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		build.BuildManifest{
			Metadata: map[string]interface{}{
				"required_provider_capabilities": []interface{}{"gpu", "arm64"},
			},
			BuildConfig: &build.BuildConfig{
				Platforms: []string{"linux/amd64"},
				BuildArgs: map[string]string{},
			},
		},
		build.BuildStatusQueued,
		time.Now().UTC(),
		time.Now().UTC(),
		nil,
	)

	required := requiredFallbackCapabilities(buildEntity)
	if len(required) != 2 {
		t.Fatalf("expected 2 required capabilities, got %d (%v)", len(required), required)
	}
	if required[0] != "arm64" || required[1] != "gpu" {
		t.Fatalf("unexpected capability set/order: %v", required)
	}
}
