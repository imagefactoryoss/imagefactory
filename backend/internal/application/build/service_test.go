package build

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	domainbuild "github.com/srikarm/image-factory/internal/domain/build"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"go.uber.org/zap"
)

type stubDomainService struct {
	buildToReturn *domainbuild.Build
	getBuildErr   error
	createErr     error
	createCalls   int
	lastManifest  *domainbuild.BuildManifest
	retryErr      error
	retryCalls    int
}

type stubWorkflowWriter struct {
	upsertCalls int
	createCalls int
	stepsCalls  int
	lastSteps   []domainworkflow.StepDefinition
}

func (s *stubWorkflowWriter) UpsertDefinition(ctx context.Context, name string, version int, definition map[string]interface{}) (uuid.UUID, error) {
	s.upsertCalls++
	return uuid.New(), nil
}

func (s *stubWorkflowWriter) CreateInstance(ctx context.Context, definitionID uuid.UUID, tenantID *uuid.UUID, subjectType string, subjectID uuid.UUID, status domainworkflow.InstanceStatus) (uuid.UUID, error) {
	s.createCalls++
	return uuid.New(), nil
}

func (s *stubWorkflowWriter) CreateSteps(ctx context.Context, instanceID uuid.UUID, steps []domainworkflow.StepDefinition) error {
	s.stepsCalls++
	s.lastSteps = append([]domainworkflow.StepDefinition(nil), steps...)
	return nil
}

func (s *stubDomainService) CreateBuild(ctx context.Context, tenantID, projectID uuid.UUID, manifest domainbuild.BuildManifest, actorID *uuid.UUID) (*domainbuild.Build, error) {
	s.createCalls++
	cloned := manifest
	s.lastManifest = &cloned
	if s.createErr != nil {
		return nil, s.createErr
	}
	return mustBuildWithStatus(nil, domainbuild.BuildStatusQueued), nil
}

func (s *stubDomainService) RetryBuild(ctx context.Context, buildID uuid.UUID) error {
	s.retryCalls++
	return s.retryErr
}

func (s *stubDomainService) GetBuild(ctx context.Context, id uuid.UUID) (*domainbuild.Build, error) {
	if s.getBuildErr != nil {
		return nil, s.getBuildErr
	}
	return s.buildToReturn, nil
}

func mustBuildWithStatus(t *testing.T, status domainbuild.BuildStatus) *domainbuild.Build {
	if t != nil {
		t.Helper()
	}
	manifest := domainbuild.BuildManifest{
		Name:         "app-service-test",
		Type:         domainbuild.BuildTypeContainer,
		BaseImage:    "alpine:3.19",
		Instructions: []string{"RUN echo ok"},
	}
	b, err := domainbuild.NewBuild(uuid.New(), uuid.New(), manifest, nil)
	if err != nil {
		if t != nil {
			t.Fatalf("failed to create build: %v", err)
		}
		panic(err)
	}
	switch status {
	case domainbuild.BuildStatusPending:
		return b
	case domainbuild.BuildStatusQueued:
		_ = b.Queue()
	case domainbuild.BuildStatusRunning:
		_ = b.Queue()
		_ = b.Start()
	case domainbuild.BuildStatusCompleted:
		_ = b.Queue()
		_ = b.Start()
		_ = b.Complete(domainbuild.BuildResult{})
	case domainbuild.BuildStatusFailed:
		_ = b.Queue()
		_ = b.Start()
		_ = b.Fail("failed")
	case domainbuild.BuildStatusCancelled:
		_ = b.Cancel()
	}
	return b
}

func TestRetryBuild_NotFound(t *testing.T) {
	domainSvc := &stubDomainService{}
	svc := NewService(domainSvc, zap.NewNop())

	err := svc.RetryBuild(context.Background(), uuid.New())
	if !errors.Is(err, domainbuild.ErrBuildNotFound) {
		t.Fatalf("expected ErrBuildNotFound, got %v", err)
	}
}

func TestCreateBuild_PreflightError(t *testing.T) {
	domainSvc := &stubDomainService{}
	svc := NewService(domainSvc, zap.NewNop())
	svc.SetCreateBuildPreflight(func(ctx context.Context, tenantID, projectID uuid.UUID, manifest *domainbuild.BuildManifest) error {
		return errors.New("create preflight failed")
	})

	err := func() error {
		_, createErr := svc.CreateBuild(context.Background(), uuid.New(), uuid.New(), domainbuild.BuildManifest{}, nil)
		return createErr
	}()
	if err == nil || err.Error() != "create preflight failed" {
		t.Fatalf("expected create preflight failed error, got %v", err)
	}
	if domainSvc.createCalls != 0 {
		t.Fatalf("expected no domain create call when preflight fails, got %d", domainSvc.createCalls)
	}
}

func TestCreateBuild_PreflightCanMutateManifest(t *testing.T) {
	domainSvc := &stubDomainService{}
	svc := NewService(domainSvc, zap.NewNop())
	mutatedProviderID := uuid.New()
	svc.SetCreateBuildPreflight(func(ctx context.Context, tenantID, projectID uuid.UUID, manifest *domainbuild.BuildManifest) error {
		manifest.InfrastructureType = "kubernetes"
		manifest.InfrastructureProviderID = &mutatedProviderID
		return nil
	})

	manifest := domainbuild.BuildManifest{
		Name:         "create-preflight-test",
		Type:         domainbuild.BuildTypeContainer,
		BaseImage:    "alpine:3.19",
		Instructions: []string{"RUN echo ok"},
	}

	if _, err := svc.CreateBuild(context.Background(), uuid.New(), uuid.New(), manifest, nil); err != nil {
		t.Fatalf("expected create success, got %v", err)
	}
	if domainSvc.createCalls != 1 {
		t.Fatalf("expected one domain create call, got %d", domainSvc.createCalls)
	}
	if domainSvc.lastManifest == nil || domainSvc.lastManifest.InfrastructureProviderID == nil {
		t.Fatal("expected preflight-mutation to be passed to domain create")
	}
	if *domainSvc.lastManifest.InfrastructureProviderID != mutatedProviderID {
		t.Fatalf("expected provider id %s, got %s", mutatedProviderID, *domainSvc.lastManifest.InfrastructureProviderID)
	}
}

func TestCreateBuild_SeedsBuildWorkflow(t *testing.T) {
	domainSvc := &stubDomainService{}
	workflowWriter := &stubWorkflowWriter{}
	svc := NewService(domainSvc, zap.NewNop())
	svc.SetWorkflowWriter(workflowWriter)

	manifest := domainbuild.BuildManifest{
		Name:         "create-workflow-seed-test",
		Type:         domainbuild.BuildTypeContainer,
		BaseImage:    "alpine:3.19",
		Instructions: []string{"RUN echo ok"},
	}

	if _, err := svc.CreateBuild(context.Background(), uuid.New(), uuid.New(), manifest, nil); err != nil {
		t.Fatalf("expected create success, got %v", err)
	}

	if workflowWriter.upsertCalls != 1 || workflowWriter.createCalls != 1 || workflowWriter.stepsCalls != 1 {
		t.Fatalf("expected workflow seed calls (1/1/1), got (%d/%d/%d)", workflowWriter.upsertCalls, workflowWriter.createCalls, workflowWriter.stepsCalls)
	}
	if len(workflowWriter.lastSteps) == 0 {
		t.Fatal("expected workflow steps to be seeded")
	}
}

func TestCreateBuild_OrchestratorDisabled_StillCreatesBuild(t *testing.T) {
	domainSvc := &stubDomainService{}
	svc := NewService(domainSvc, zap.NewNop())
	// Intentionally do not configure workflow writer (orchestrator disabled path).

	manifest := domainbuild.BuildManifest{
		Name:         "create-no-orchestrator",
		Type:         domainbuild.BuildTypeContainer,
		BaseImage:    "alpine:3.19",
		Instructions: []string{"RUN echo ok"},
	}

	created, err := svc.CreateBuild(context.Background(), uuid.New(), uuid.New(), manifest, nil)
	if err != nil {
		t.Fatalf("expected create success when orchestrator is disabled, got %v", err)
	}
	if created == nil {
		t.Fatal("expected created build")
	}
	if domainSvc.createCalls != 1 {
		t.Fatalf("expected one domain create call, got %d", domainSvc.createCalls)
	}
}

func TestCreateBuild_ClassifiesValidationErrors(t *testing.T) {
	domainSvc := &stubDomainService{createErr: errors.New("build name is required")}
	svc := NewService(domainSvc, zap.NewNop())

	_, err := svc.CreateBuild(context.Background(), uuid.New(), uuid.New(), domainbuild.BuildManifest{}, nil)
	if err == nil {
		t.Fatal("expected create build error")
	}

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T (%v)", err, err)
	}
}

func TestCreateBuild_DoesNotClassifyInternalErrors(t *testing.T) {
	internalErr := errors.New("database is unavailable")
	domainSvc := &stubDomainService{createErr: internalErr}
	svc := NewService(domainSvc, zap.NewNop())

	_, err := svc.CreateBuild(context.Background(), uuid.New(), uuid.New(), domainbuild.BuildManifest{}, nil)
	if err == nil {
		t.Fatal("expected create build error")
	}

	if !errors.Is(err, internalErr) {
		t.Fatalf("expected original internal error, got %v", err)
	}

	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		t.Fatalf("did not expect ValidationError for internal error: %v", err)
	}
}

func TestRetryBuild_NotRetriable(t *testing.T) {
	domainSvc := &stubDomainService{buildToReturn: mustBuildWithStatus(t, domainbuild.BuildStatusRunning)}
	svc := NewService(domainSvc, zap.NewNop())

	err := svc.RetryBuild(context.Background(), uuid.New())
	var notRetriable ErrBuildNotRetriable
	if !errors.As(err, &notRetriable) {
		t.Fatalf("expected ErrBuildNotRetriable, got %v", err)
	}
}

func TestRetryBuild_PreflightError(t *testing.T) {
	domainSvc := &stubDomainService{buildToReturn: mustBuildWithStatus(t, domainbuild.BuildStatusFailed)}
	svc := NewService(domainSvc, zap.NewNop())
	svc.SetRetryBuildPreflight(func(ctx context.Context, b *domainbuild.Build) error {
		return errors.New("preflight failed")
	})

	err := svc.RetryBuild(context.Background(), uuid.New())
	if err == nil || err.Error() != "preflight failed" {
		t.Fatalf("expected preflight failed error, got %v", err)
	}
	if domainSvc.retryCalls != 0 {
		t.Fatalf("expected no retry call when preflight fails, got %d", domainSvc.retryCalls)
	}
}

func TestRetryBuild_Success(t *testing.T) {
	domainSvc := &stubDomainService{buildToReturn: mustBuildWithStatus(t, domainbuild.BuildStatusFailed)}
	workflowWriter := &stubWorkflowWriter{}
	svc := NewService(domainSvc, zap.NewNop())
	svc.SetWorkflowWriter(workflowWriter)

	err := svc.RetryBuild(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("expected retry success, got %v", err)
	}
	if domainSvc.retryCalls != 1 {
		t.Fatalf("expected one retry call, got %d", domainSvc.retryCalls)
	}
	if workflowWriter.stepsCalls != 1 {
		t.Fatalf("expected workflow seed on retry, got %d", workflowWriter.stepsCalls)
	}
	if len(workflowWriter.lastSteps) >= 4 && workflowWriter.lastSteps[3].Status != domainworkflow.StepStatusSucceeded {
		t.Fatalf("expected dispatch step succeeded for retry seed, got %s", workflowWriter.lastSteps[3].Status)
	}
}

func TestRetryBuild_OrchestratorDisabled_StillRetriesBuild(t *testing.T) {
	domainSvc := &stubDomainService{buildToReturn: mustBuildWithStatus(t, domainbuild.BuildStatusFailed)}
	svc := NewService(domainSvc, zap.NewNop())
	// Intentionally do not configure workflow writer (orchestrator disabled path).

	err := svc.RetryBuild(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("expected retry success when orchestrator is disabled, got %v", err)
	}
	if domainSvc.retryCalls != 1 {
		t.Fatalf("expected one retry call, got %d", domainSvc.retryCalls)
	}
}
