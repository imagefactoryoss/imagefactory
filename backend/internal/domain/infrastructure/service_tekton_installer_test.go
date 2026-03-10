package infrastructure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type installerRepoStub struct {
	providers          map[uuid.UUID]*Provider
	installerJobs      map[uuid.UUID]*TektonInstallerJob
	createdJobs        []*TektonInstallerJob
	eventsByJob        map[uuid.UUID][]*TektonInstallerJobEvent
	listJobsByProvider map[uuid.UUID][]*TektonInstallerJob
	nextPendingJob     *TektonInstallerJob
	createJobErr       error
	updatedStatuses    map[uuid.UUID]TektonInstallerJobStatus
	idempotentJobs     map[string]*TektonInstallerJob
}

func (r *installerRepoStub) SaveProvider(ctx context.Context, provider *Provider) error { return nil }
func (r *installerRepoStub) FindProviderByID(ctx context.Context, id uuid.UUID) (*Provider, error) {
	return r.providers[id], nil
}
func (r *installerRepoStub) FindProvidersByTenant(ctx context.Context, tenantID uuid.UUID, opts *ListProvidersOptions) (*ListProvidersResult, error) {
	return &ListProvidersResult{}, nil
}
func (r *installerRepoStub) FindProvidersAll(ctx context.Context, opts *ListProvidersOptions) (*ListProvidersResult, error) {
	return &ListProvidersResult{}, nil
}
func (r *installerRepoStub) UpdateProvider(ctx context.Context, provider *Provider) error { return nil }
func (r *installerRepoStub) DeleteProvider(ctx context.Context, id uuid.UUID) error       { return nil }
func (r *installerRepoStub) ExistsProviderByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	return false, nil
}
func (r *installerRepoStub) SavePermission(ctx context.Context, permission *ProviderPermission) error {
	return nil
}
func (r *installerRepoStub) FindPermissionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*ProviderPermission, error) {
	return nil, nil
}
func (r *installerRepoStub) FindPermissionsByProvider(ctx context.Context, providerID uuid.UUID) ([]*ProviderPermission, error) {
	return nil, nil
}
func (r *installerRepoStub) DeletePermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) error {
	return nil
}
func (r *installerRepoStub) HasPermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) (bool, error) {
	return false, nil
}
func (r *installerRepoStub) UpdateProviderHealth(ctx context.Context, providerID uuid.UUID, health *ProviderHealth) error {
	return nil
}
func (r *installerRepoStub) GetProviderHealth(ctx context.Context, providerID uuid.UUID) (*ProviderHealth, error) {
	return nil, nil
}
func (r *installerRepoStub) UpdateProviderReadiness(ctx context.Context, providerID uuid.UUID, status string, checkedAt time.Time, missingPrereqs []string) error {
	return nil
}

func (r *installerRepoStub) CreateInstallerJob(ctx context.Context, job *TektonInstallerJob) error {
	if r.createJobErr != nil {
		return r.createJobErr
	}
	r.createdJobs = append(r.createdJobs, job)
	return nil
}
func (r *installerRepoStub) FindInstallerJobByProviderAndIdempotencyKey(ctx context.Context, providerID uuid.UUID, operation TektonInstallerOperation, idempotencyKey string) (*TektonInstallerJob, error) {
	if r.idempotentJobs == nil {
		return nil, nil
	}
	return r.idempotentJobs[string(operation)+"|"+providerID.String()+"|"+idempotencyKey], nil
}
func (r *installerRepoStub) ClaimNextPendingInstallerJob(ctx context.Context) (*TektonInstallerJob, error) {
	job := r.nextPendingJob
	r.nextPendingJob = nil
	return job, nil
}
func (r *installerRepoStub) UpdateInstallerJobStatus(ctx context.Context, id uuid.UUID, status TektonInstallerJobStatus, startedAt, completedAt *time.Time, errorMessage *string) error {
	if r.updatedStatuses == nil {
		r.updatedStatuses = make(map[uuid.UUID]TektonInstallerJobStatus)
	}
	r.updatedStatuses[id] = status
	return nil
}
func (r *installerRepoStub) GetInstallerJob(ctx context.Context, id uuid.UUID) (*TektonInstallerJob, error) {
	if r.installerJobs == nil {
		return nil, nil
	}
	return r.installerJobs[id], nil
}
func (r *installerRepoStub) ListInstallerJobsByProvider(ctx context.Context, providerID uuid.UUID, limit, offset int) ([]*TektonInstallerJob, error) {
	return r.listJobsByProvider[providerID], nil
}
func (r *installerRepoStub) AddInstallerJobEvent(ctx context.Context, event *TektonInstallerJobEvent) error {
	if r.eventsByJob == nil {
		r.eventsByJob = make(map[uuid.UUID][]*TektonInstallerJobEvent)
	}
	r.eventsByJob[event.JobID] = append(r.eventsByJob[event.JobID], event)
	return nil
}
func (r *installerRepoStub) ListInstallerJobEvents(ctx context.Context, jobID uuid.UUID, limit, offset int) ([]*TektonInstallerJobEvent, error) {
	return r.eventsByJob[jobID], nil
}

func TestStartTektonInstallerJob_DefaultsFromProviderConfig(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	providerID := uuid.New()

	repo := &installerRepoStub{
		providers: map[uuid.UUID]*Provider{
			providerID: {
				ID:       providerID,
				TenantID: tenantID,
				Config: map[string]interface{}{
					"tekton_install_mode":    "gitops",
					"tekton_profile_version": "v2",
				},
			},
		},
	}
	service := NewService(repo, nil, nil)

	job, err := service.StartTektonInstallerJob(context.Background(), providerID, tenantID, userID, StartTektonInstallerJobRequest{
		Operation: TektonInstallerOperationInstall,
	})
	if err != nil {
		t.Fatalf("StartTektonInstallerJob returned error: %v", err)
	}
	if job.Operation != TektonInstallerOperationInstall {
		t.Fatalf("expected operation %q, got %q", TektonInstallerOperationInstall, job.Operation)
	}
	if job.InstallMode != TektonInstallModeGitOps {
		t.Fatalf("expected install mode %q, got %q", TektonInstallModeGitOps, job.InstallMode)
	}
	if job.AssetVersion != "v2" {
		t.Fatalf("expected asset version v2, got %q", job.AssetVersion)
	}
	if job.Status != TektonInstallerJobStatusPending {
		t.Fatalf("expected pending status, got %q", job.Status)
	}
	if len(repo.eventsByJob[job.ID]) != 1 {
		t.Fatalf("expected 1 job event, got %d", len(repo.eventsByJob[job.ID]))
	}
	if repo.eventsByJob[job.ID][0].EventType != "install.requested" {
		t.Fatalf("expected event type install.requested, got %q", repo.eventsByJob[job.ID][0].EventType)
	}
}

func TestStartTektonInstallerJob_ProviderLockConflict(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	providerID := uuid.New()

	repo := &installerRepoStub{
		providers: map[uuid.UUID]*Provider{
			providerID: {
				ID:       providerID,
				TenantID: tenantID,
			},
		},
		createJobErr: errors.New("failed to create installer job: duplicate key value violates unique constraint \"uq_tekton_installer_jobs_provider_active\""),
	}
	service := NewService(repo, nil, nil)

	_, err := service.StartTektonInstallerJob(context.Background(), providerID, tenantID, userID, StartTektonInstallerJobRequest{
		Operation: TektonInstallerOperationUpgrade,
	})
	if !errors.Is(err, ErrTektonInstallerJobInProgress) {
		t.Fatalf("expected ErrTektonInstallerJobInProgress, got %v", err)
	}
}

func TestGetTektonInstallerStatus_ActiveJobAndEvents(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	activeJobID := uuid.New()
	now := time.Now().UTC()

	repo := &installerRepoStub{
		providers: map[uuid.UUID]*Provider{
			providerID: {
				ID:       providerID,
				TenantID: tenantID,
			},
		},
		listJobsByProvider: map[uuid.UUID][]*TektonInstallerJob{
			providerID: {
				{
					ID:         activeJobID,
					ProviderID: providerID,
					TenantID:   tenantID,
					Status:     TektonInstallerJobStatusRunning,
					CreatedAt:  now,
					UpdatedAt:  now,
				},
				{
					ID:         uuid.New(),
					ProviderID: providerID,
					TenantID:   tenantID,
					Status:     TektonInstallerJobStatusSucceeded,
					CreatedAt:  now.Add(-time.Hour),
					UpdatedAt:  now.Add(-time.Hour),
				},
			},
		},
		eventsByJob: map[uuid.UUID][]*TektonInstallerJobEvent{
			activeJobID: {
				{
					ID:        uuid.New(),
					JobID:     activeJobID,
					EventType: "upgrade.requested",
					Message:   "requested",
					CreatedAt: now,
				},
			},
		},
	}

	service := NewService(repo, nil, nil)
	status, err := service.GetTektonInstallerStatus(context.Background(), providerID, 10)
	if err != nil {
		t.Fatalf("GetTektonInstallerStatus returned error: %v", err)
	}
	if status.ActiveJob == nil {
		t.Fatalf("expected active job in status response")
	}
	if status.ActiveJob.ID != activeJobID {
		t.Fatalf("expected active job %s, got %s", activeJobID, status.ActiveJob.ID)
	}
	if len(status.ActiveJobEvents) != 1 {
		t.Fatalf("expected 1 active job event, got %d", len(status.ActiveJobEvents))
	}
}

func TestRunNextTektonInstallerJob_NoPendingJob(t *testing.T) {
	service := NewService(&installerRepoStub{}, nil, nil)
	processed, err := service.RunNextTektonInstallerJob(context.Background())
	if err != nil {
		t.Fatalf("RunNextTektonInstallerJob returned error: %v", err)
	}
	if processed {
		t.Fatalf("expected processed=false when there is no pending job")
	}
}

func TestRunNextTektonInstallerJob_GitOpsInstallSucceeds(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	jobID := uuid.New()
	now := time.Now().UTC()

	repo := &installerRepoStub{
		providers: map[uuid.UUID]*Provider{
			providerID: {
				ID:       providerID,
				TenantID: tenantID,
				Config: map[string]interface{}{
					"tekton_install_mode": "gitops",
				},
			},
		},
		nextPendingJob: &TektonInstallerJob{
			ID:          jobID,
			ProviderID:  providerID,
			TenantID:    tenantID,
			InstallMode: TektonInstallModeGitOps,
			Status:      TektonInstallerJobStatusRunning,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		eventsByJob: map[uuid.UUID][]*TektonInstallerJobEvent{
			jobID: {
				{
					ID:        uuid.New(),
					JobID:     jobID,
					EventType: "install.requested",
					Details: map[string]interface{}{
						"operation": "install",
					},
					CreatedAt: now,
				},
			},
		},
	}

	service := NewService(repo, nil, nil)
	processed, err := service.RunNextTektonInstallerJob(context.Background())
	if err != nil {
		t.Fatalf("RunNextTektonInstallerJob returned error: %v", err)
	}
	if !processed {
		t.Fatalf("expected processed=true")
	}
	if repo.updatedStatuses[jobID] != TektonInstallerJobStatusSucceeded {
		t.Fatalf("expected job status %q, got %q", TektonInstallerJobStatusSucceeded, repo.updatedStatuses[jobID])
	}
}

func TestExecuteInstallOrUpgradeInstallerJob_EmptyInstallModeDefaultsToInstallerMode(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	jobID := uuid.New()

	repo := &installerRepoStub{
		providers: map[uuid.UUID]*Provider{
			providerID: {
				ID:       providerID,
				TenantID: tenantID,
				// Intentionally missing bootstrap auth so apply path fails after mode resolution.
				Config: map[string]interface{}{},
			},
		},
	}
	service := NewService(repo, nil, nil)
	job := &TektonInstallerJob{
		ID:          jobID,
		ProviderID:  providerID,
		TenantID:    tenantID,
		InstallMode: "",
	}

	err := service.executeInstallOrUpgradeInstallerJob(context.Background(), job, TektonInstallerOperationInstall)
	if err == nil {
		t.Fatal("expected error for invalid bootstrap auth config")
	}
	if strings.Contains(err.Error(), "not implemented for install_mode") {
		t.Fatalf("expected empty install mode to resolve to installer mode, got: %v", err)
	}
}

func TestResolveTektonTargetNamespace(t *testing.T) {
	tenantID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	unsetTenantID := uuid.Nil

	withConfig := &Provider{
		Config: map[string]interface{}{
			"tekton_target_namespace": "custom-tekton",
		},
	}
	if ns := resolveTektonTargetNamespace(withConfig, tenantID); ns != "custom-tekton" {
		t.Fatalf("expected custom namespace, got %q", ns)
	}

	if ns := resolveTektonTargetNamespace(&Provider{}, unsetTenantID); ns != "image-factory-default" {
		t.Fatalf("expected default namespace image-factory-default, got %q", ns)
	}

	if ns := resolveTektonTargetNamespace(&Provider{}, tenantID); ns != "image-factory-11111111" {
		t.Fatalf("expected tenant namespace image-factory-11111111, got %q", ns)
	}
}

func TestStartTektonInstallerJob_IdempotencyReplayReturnsExistingJob(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	providerID := uuid.New()
	existingJobID := uuid.New()

	existing := &TektonInstallerJob{
		ID:          existingJobID,
		ProviderID:  providerID,
		TenantID:    tenantID,
		RequestedBy: userID,
		Operation:   TektonInstallerOperationInstall,
		Status:      TektonInstallerJobStatusRunning,
		CreatedAt:   time.Now().UTC().Add(-time.Minute),
		UpdatedAt:   time.Now().UTC().Add(-time.Minute),
	}

	repo := &installerRepoStub{
		providers: map[uuid.UUID]*Provider{
			providerID: {
				ID:       providerID,
				TenantID: tenantID,
			},
		},
		idempotentJobs: map[string]*TektonInstallerJob{
			string(TektonInstallerOperationInstall) + "|" + providerID.String() + "|abc-123": existing,
		},
	}
	service := NewService(repo, nil, nil)

	job, err := service.StartTektonInstallerJob(context.Background(), providerID, tenantID, userID, StartTektonInstallerJobRequest{
		Operation:      TektonInstallerOperationInstall,
		IdempotencyKey: "abc-123",
	})
	if err != nil {
		t.Fatalf("StartTektonInstallerJob returned error: %v", err)
	}
	if job.ID != existingJobID {
		t.Fatalf("expected existing job %s, got %s", existingJobID, job.ID)
	}
	if len(repo.createdJobs) != 0 {
		t.Fatalf("expected no new jobs to be created, got %d", len(repo.createdJobs))
	}
}

func TestRetryTektonInstallerJob_CreatesPendingJobFromFailedSource(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	providerID := uuid.New()
	sourceJobID := uuid.New()

	repo := &installerRepoStub{
		providers: map[uuid.UUID]*Provider{
			providerID: {
				ID:       providerID,
				TenantID: tenantID,
			},
		},
		installerJobs: map[uuid.UUID]*TektonInstallerJob{
			sourceJobID: {
				ID:           sourceJobID,
				ProviderID:   providerID,
				TenantID:     tenantID,
				RequestedBy:  userID,
				Operation:     TektonInstallerOperationUpgrade,
				InstallMode:  TektonInstallModeImageFactoryInstaller,
				AssetVersion: "v3",
				Status:       TektonInstallerJobStatusFailed,
				CreatedAt:    time.Now().UTC().Add(-10 * time.Minute),
				UpdatedAt:    time.Now().UTC().Add(-5 * time.Minute),
			},
		},
		eventsByJob: map[uuid.UUID][]*TektonInstallerJobEvent{
			sourceJobID: {
				{
					ID:        uuid.New(),
					JobID:     sourceJobID,
					EventType: "upgrade.requested",
					Details: map[string]interface{}{
						"operation": "upgrade",
					},
					CreatedAt: time.Now().UTC().Add(-10 * time.Minute),
				},
			},
		},
	}

	service := NewService(repo, nil, nil)
	job, err := service.RetryTektonInstallerJob(context.Background(), providerID, sourceJobID, tenantID, userID)
	if err != nil {
		t.Fatalf("RetryTektonInstallerJob returned error: %v", err)
	}
	if job.Status != TektonInstallerJobStatusPending {
		t.Fatalf("expected pending status, got %q", job.Status)
	}
	if job.InstallMode != TektonInstallModeImageFactoryInstaller {
		t.Fatalf("expected install mode %q, got %q", TektonInstallModeImageFactoryInstaller, job.InstallMode)
	}
	if job.AssetVersion != "v3" {
		t.Fatalf("expected asset version v3, got %q", job.AssetVersion)
	}
	if len(repo.createdJobs) != 1 {
		t.Fatalf("expected 1 created job, got %d", len(repo.createdJobs))
	}
	if repo.createdJobs[0].Operation != TektonInstallerOperationUpgrade {
		t.Fatalf("expected created job operation %q, got %q", TektonInstallerOperationUpgrade, repo.createdJobs[0].Operation)
	}
	retryEvents := repo.eventsByJob[job.ID]
	if len(retryEvents) < 2 {
		t.Fatalf("expected retry job to have at least 2 events, got %d", len(retryEvents))
	}
}

func TestRetryTektonInstallerJob_RejectsNonFailedSource(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	providerID := uuid.New()
	sourceJobID := uuid.New()

	repo := &installerRepoStub{
		providers: map[uuid.UUID]*Provider{
			providerID: {
				ID:       providerID,
				TenantID: tenantID,
			},
		},
		installerJobs: map[uuid.UUID]*TektonInstallerJob{
			sourceJobID: {
				ID:         sourceJobID,
				ProviderID: providerID,
				TenantID:   tenantID,
				Status:     TektonInstallerJobStatusSucceeded,
				CreatedAt:  time.Now().UTC().Add(-10 * time.Minute),
				UpdatedAt:  time.Now().UTC().Add(-5 * time.Minute),
			},
		},
	}

	service := NewService(repo, nil, nil)
	_, err := service.RetryTektonInstallerJob(context.Background(), providerID, sourceJobID, tenantID, userID)
	if !errors.Is(err, ErrTektonInstallerJobNotRetryable) {
		t.Fatalf("expected ErrTektonInstallerJobNotRetryable, got %v", err)
	}
	if len(repo.createdJobs) != 0 {
		t.Fatalf("expected 0 created jobs, got %d", len(repo.createdJobs))
	}
}

func TestSummarizeTektonResources(t *testing.T) {
	required := []string{"tasks", "pipelines", "pipelineruns"}
	resourceList := &metav1.APIResourceList{
		GroupVersion: "tekton.dev/v1",
		APIResources: []metav1.APIResource{
			{Name: "tasks"},
			{Name: "pipelineruns"},
			{Name: "customruns"},
		},
	}

	present, missing := summarizeTektonResources(resourceList, required)
	if len(present) != 2 {
		t.Fatalf("expected 2 present resources, got %d", len(present))
	}
	if len(missing) != 1 {
		t.Fatalf("expected 1 missing resource, got %d", len(missing))
	}
	if missing[0] != "pipelines" {
		t.Fatalf("expected missing resource pipelines, got %q", missing[0])
	}
}

func TestRequiredTektonPipelineNamesForProvider_UsesProfileVersion(t *testing.T) {
	provider := &Provider{
		Config: map[string]interface{}{
			"tekton_profile_version": "v2",
		},
	}
	names := requiredTektonPipelineNamesForProvider(provider)
	if len(names) != 4 {
		t.Fatalf("expected 4 pipeline names, got %d", len(names))
	}
	for _, name := range names {
		if !strings.Contains(name, "image-factory-build-v2-") {
			t.Fatalf("expected v2 pipeline naming, got %q", name)
		}
	}
}

func TestResolveTektonAssetVersion_NormalizesCaseAndWhitespace(t *testing.T) {
	provider := &Provider{
		Config: map[string]interface{}{
			"tekton_profile_version": " V2 ",
		},
	}

	if got := resolveTektonAssetVersion(provider, "  V3 "); got != "v3" {
		t.Fatalf("expected requested version normalized to v3, got %q", got)
	}
	if got := resolveTektonAssetVersion(provider, ""); got != "v2" {
		t.Fatalf("expected provider config version normalized to v2, got %q", got)
	}
}

func TestLoadKustomizationResourceFilesForProfile_RewritesVersionedAssets(t *testing.T) {
	assetRoot := t.TempDir()

	writeFile := func(rel string) {
		path := filepath.Join(assetRoot, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("failed to create directory for %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n"), 0o644); err != nil {
			t.Fatalf("failed to write file %s: %v", rel, err)
		}
	}

	if err := os.WriteFile(filepath.Join(assetRoot, "kustomization.yaml"), []byte(`resources:
  - tasks/v1/git-clone-task.yaml
  - pipelines/v1/image-factory-build-kaniko.yaml
  - jobs/v1/tekton-history-cleanup-cronjob.yaml
`), 0o644); err != nil {
		t.Fatalf("failed to write kustomization file: %v", err)
	}
	writeFile("tasks/v1/git-clone-task.yaml")
	writeFile("tasks/v2/git-clone-task.yaml")
	writeFile("pipelines/v1/image-factory-build-kaniko.yaml")
	writeFile("pipelines/v2/image-factory-build-kaniko.yaml")
	writeFile("jobs/v1/tekton-history-cleanup-cronjob.yaml")

	files, err := loadKustomizationResourceFilesForProfile(assetRoot, "v2")
	if err != nil {
		t.Fatalf("loadKustomizationResourceFilesForProfile returned error: %v", err)
	}
	joined := strings.Join(files, "\n")
	if !strings.Contains(joined, filepath.ToSlash(filepath.Join(assetRoot, "tasks/v2/git-clone-task.yaml"))) {
		t.Fatalf("expected tasks/v2 file in resolved list, got: %v", files)
	}
	if !strings.Contains(joined, filepath.ToSlash(filepath.Join(assetRoot, "pipelines/v2/image-factory-build-kaniko.yaml"))) {
		t.Fatalf("expected pipelines/v2 file in resolved list, got: %v", files)
	}
	if !strings.Contains(joined, filepath.ToSlash(filepath.Join(assetRoot, "jobs/v1/tekton-history-cleanup-cronjob.yaml"))) {
		t.Fatalf("expected jobs/v1 file to remain unchanged, got: %v", files)
	}
}

func TestLoadKustomizationResourceFilesForProfile_ErrorsWhenVersionAssetMissing(t *testing.T) {
	assetRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(assetRoot, "kustomization.yaml"), []byte(`resources:
  - tasks/v1/git-clone-task.yaml
`), 0o644); err != nil {
		t.Fatalf("failed to write kustomization file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(assetRoot, "tasks", "v1"), 0o755); err != nil {
		t.Fatalf("failed to create tasks/v1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetRoot, "tasks", "v1", "git-clone-task.yaml"), []byte("kind: Task\n"), 0o644); err != nil {
		t.Fatalf("failed to write tasks/v1 file: %v", err)
	}

	_, err := loadKustomizationResourceFilesForProfile(assetRoot, "v3")
	if err == nil {
		t.Fatal("expected error when requested profile version assets are missing")
	}
	if !strings.Contains(err.Error(), "tekton profile version v3 requested but resource does not exist") {
		t.Fatalf("expected explicit missing profile version error, got: %v", err)
	}
}
