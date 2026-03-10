package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appbuild "github.com/srikarm/image-factory/internal/application/build"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	systemconfig "github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

type stubInfraServiceDefaultProvider struct {
	providers []*infrastructure.Provider
}

func (s *stubInfraServiceDefaultProvider) GetAvailableProviders(ctx context.Context, tenantID uuid.UUID) ([]*infrastructure.Provider, error) {
	return s.providers, nil
}

func (s *stubInfraServiceDefaultProvider) GetTenantNamespacePrepareStatus(ctx context.Context, providerID, tenantID uuid.UUID) (*infrastructure.ProviderTenantNamespacePrepare, error) {
	ns := "image-factory-" + tenantID.String()[:8]
	return &infrastructure.ProviderTenantNamespacePrepare{
		ID:         uuid.New(),
		ProviderID: providerID,
		TenantID:   tenantID,
		Namespace:  ns,
		Status:     infrastructure.ProviderTenantNamespacePrepareSucceeded,
	}, nil
}

type stubBuildRepo struct {
	lastSaved *build.Build
}

func (s *stubBuildRepo) Save(ctx context.Context, b *build.Build) error { s.lastSaved = b; return nil }
func (s *stubBuildRepo) FindByID(ctx context.Context, id uuid.UUID) (*build.Build, error) {
	return nil, nil
}
func (s *stubBuildRepo) FindByIDsBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*build.Build, error) {
	return map[uuid.UUID]*build.Build{}, nil
}
func (s *stubBuildRepo) FindByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*build.Build, error) {
	return nil, nil
}
func (s *stubBuildRepo) FindByProjectID(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*build.Build, error) {
	return nil, nil
}
func (s *stubBuildRepo) FindByStatus(ctx context.Context, status build.BuildStatus, limit, offset int) ([]*build.Build, error) {
	return nil, nil
}
func (s *stubBuildRepo) Update(ctx context.Context, b *build.Build) error { return nil }
func (s *stubBuildRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status build.BuildStatus, startedAt, completedAt *time.Time, errorMessage *string) error {
	return nil
}
func (s *stubBuildRepo) ClaimNextQueuedBuild(ctx context.Context) (*build.Build, error) {
	return nil, nil
}
func (s *stubBuildRepo) RequeueBuild(ctx context.Context, id uuid.UUID, nextRunAt time.Time, errorMessage *string) error {
	return nil
}
func (s *stubBuildRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (s *stubBuildRepo) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return 0, nil
}
func (s *stubBuildRepo) CountByStatus(ctx context.Context, tenantID uuid.UUID, status build.BuildStatus) (int, error) {
	return 0, nil
}
func (s *stubBuildRepo) CountByProjectID(ctx context.Context, projectID uuid.UUID) (int, error) {
	return 0, nil
}
func (s *stubBuildRepo) FindRunningBuilds(ctx context.Context) ([]*build.Build, error) {
	return nil, nil
}
func (s *stubBuildRepo) SaveBuildConfig(ctx context.Context, config *build.BuildConfigData) error {
	return nil
}
func (s *stubBuildRepo) GetBuildConfig(ctx context.Context, buildID uuid.UUID) (*build.BuildConfigData, error) {
	return nil, nil
}
func (s *stubBuildRepo) UpdateBuildConfig(ctx context.Context, config *build.BuildConfigData) error {
	return nil
}
func (s *stubBuildRepo) DeleteBuildConfig(ctx context.Context, buildID uuid.UUID) error { return nil }
func (s *stubBuildRepo) UpdateInfrastructureSelection(ctx context.Context, b *build.Build) error {
	return nil
}

type stubTriggerRepo struct{}

func (s *stubTriggerRepo) SaveTrigger(ctx context.Context, trigger *build.BuildTrigger) error {
	return nil
}
func (s *stubTriggerRepo) GetTrigger(ctx context.Context, triggerID uuid.UUID) (*build.BuildTrigger, error) {
	return nil, nil
}
func (s *stubTriggerRepo) GetTriggersByBuild(ctx context.Context, buildID uuid.UUID) ([]*build.BuildTrigger, error) {
	return nil, nil
}
func (s *stubTriggerRepo) GetTriggersByProject(ctx context.Context, projectID uuid.UUID) ([]*build.BuildTrigger, error) {
	return nil, nil
}
func (s *stubTriggerRepo) GetActiveScheduledTriggers(ctx context.Context, tenantID uuid.UUID) ([]*build.BuildTrigger, error) {
	return nil, nil
}
func (s *stubTriggerRepo) UpdateTrigger(ctx context.Context, trigger *build.BuildTrigger) error {
	return nil
}
func (s *stubTriggerRepo) DeleteTrigger(ctx context.Context, triggerID uuid.UUID) error { return nil }

type stubEventPublisher struct{}

func (s *stubEventPublisher) PublishBuildCreated(ctx context.Context, event *build.BuildCreated) error {
	return nil
}
func (s *stubEventPublisher) PublishBuildStarted(ctx context.Context, event *build.BuildStarted) error {
	return nil
}
func (s *stubEventPublisher) PublishBuildCompleted(ctx context.Context, event *build.BuildCompleted) error {
	return nil
}

func (s *stubEventPublisher) PublishBuildStatusUpdated(ctx context.Context, event *build.BuildStatusUpdated) error {
	return nil
}

type stubSystemConfigService struct{}

func (s *stubSystemConfigService) GetBuildConfig(ctx context.Context, tenantID uuid.UUID) (*systemconfig.BuildConfig, error) {
	return &systemconfig.BuildConfig{MaxConcurrentJobs: 10, MaxQueueSize: 100, TektonEnabled: true}, nil
}
func (s *stubSystemConfigService) GetToolAvailabilityConfig(ctx context.Context, tenantID *uuid.UUID) (*systemconfig.ToolAvailabilityConfig, error) {
	return &systemconfig.ToolAvailabilityConfig{
		BuildMethods:   systemconfig.BuildMethodAvailability{Kaniko: true},
		SBOMTools:      systemconfig.SBOMToolAvailability{Syft: true, Trivy: true},
		ScanTools:      systemconfig.ScanToolAvailability{Trivy: true},
		RegistryTypes:  systemconfig.RegistryTypeAvailability{Harbor: true},
		SecretManagers: systemconfig.SecretManagerAvailability{Vault: true},
	}, nil
}

type stubDomainBuildServiceNoop struct{}

func (s *stubDomainBuildServiceNoop) CreateBuild(ctx context.Context, tenantID, projectID uuid.UUID, manifest build.BuildManifest, actorID *uuid.UUID) (*build.Build, error) {
	return nil, build.ErrBuildNotFound
}

func (s *stubDomainBuildServiceNoop) RetryBuild(ctx context.Context, buildID uuid.UUID) error {
	return build.ErrBuildNotFound
}

func (s *stubDomainBuildServiceNoop) GetBuild(ctx context.Context, id uuid.UUID) (*build.Build, error) {
	return nil, build.ErrBuildNotFound
}

func TestCreateBuild_WhenKubernetesProviderOmitted_DefaultsToGlobalProvider(t *testing.T) {
	logger := zap.NewNop()

	repo := &stubBuildRepo{}
	localFactory := build.NewBuildMethodExecutorFactory(nil)
	noopExecutor := build.NewNoOpBuildExecutor(logger)
	buildService := build.NewService(
		repo,
		&stubTriggerRepo{},
		&stubEventPublisher{},
		noopExecutor,
		noopExecutor,
		nil,
		localFactory,
		nil,
		&stubSystemConfigService{},
		nil,
		logger,
	)

	tenantID := uuid.New()
	projectID := uuid.New()
	globalProviderID := uuid.New()
	infraSvc := &stubInfraServiceDefaultProvider{providers: []*infrastructure.Provider{
		{
			ID:           globalProviderID,
			TenantID:     tenantID,
			IsGlobal:     true,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Name:         "global-k8s",
			Config: map[string]interface{}{
				"tekton_enabled": true,
				"runtime_auth": map[string]interface{}{
					"auth_method": "token",
					"endpoint":    "https://k8s.example",
					"token":       "token",
				},
			},
		},
	}}

	h := NewBuildHandler(buildService, appbuild.NewService(buildService, logger), nil, nil, nil, infraSvc, logger)

	body := CreateBuildRequest{
		TenantID:  tenantID.String(),
		ProjectID: projectID.String(),
		Manifest: build.BuildManifest{
			Name:               "Build",
			Type:               build.BuildTypeKaniko,
			InfrastructureType: "kubernetes",
			BuildConfig: &build.BuildConfig{
				BuildType:         build.BuildTypeKaniko,
				SBOMTool:          build.SBOMToolSyft,
				ScanTool:          build.ScanToolTrivy,
				RegistryType:      build.RegistryTypeHarbor,
				SecretManagerType: build.SecretManagerVault,
				Dockerfile:        "FROM alpine",
				BuildContext:      ".",
				RegistryRepo:      "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
			},
		},
	}

	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/builds", bytes.NewReader(buf))
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{UserID: uuid.New(), TenantID: tenantID}))
	w := httptest.NewRecorder()

	h.CreateBuild(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}
	if repo.lastSaved == nil {
		t.Fatalf("expected build to be saved")
	}
	if repo.lastSaved.InfrastructureProviderID() == nil || *repo.lastSaved.InfrastructureProviderID() != globalProviderID {
		t.Fatalf("expected global provider ID to be persisted")
	}
}

func TestCreateBuild_WhenKubernetesProviderOmitted_GlobalProviderNotTektonReady_FailsFast(t *testing.T) {
	logger := zap.NewNop()

	tenantID := uuid.New()
	projectID := uuid.New()
	globalProviderID := uuid.New()
	infraSvc := &stubInfraServiceDefaultProvider{providers: []*infrastructure.Provider{
		{
			ID:           globalProviderID,
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
		},
	}}

	h := NewBuildHandler(nil, appbuild.NewService(&stubDomainBuildServiceNoop{}, logger), nil, nil, nil, infraSvc, logger)

	body := CreateBuildRequest{
		TenantID:  tenantID.String(),
		ProjectID: projectID.String(),
		Manifest: build.BuildManifest{
			Name:               "Build",
			Type:               build.BuildTypeKaniko,
			InfrastructureType: "kubernetes",
			BuildConfig: &build.BuildConfig{
				BuildType:         build.BuildTypeKaniko,
				SBOMTool:          build.SBOMToolSyft,
				ScanTool:          build.ScanToolTrivy,
				RegistryType:      build.RegistryTypeHarbor,
				SecretManagerType: build.SecretManagerVault,
				Dockerfile:        "FROM alpine",
				BuildContext:      ".",
				RegistryRepo:      "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
			},
		},
	}

	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/builds", bytes.NewReader(buf))
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{UserID: uuid.New(), TenantID: tenantID}))
	w := httptest.NewRecorder()

	h.CreateBuild(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}
