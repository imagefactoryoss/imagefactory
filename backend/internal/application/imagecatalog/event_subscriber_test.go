package imagecatalog

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/image"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type buildRepoStub struct {
	byID map[uuid.UUID]*build.Build
}

func (s *buildRepoStub) FindByID(_ context.Context, id uuid.UUID) (*build.Build, error) {
	return s.byID[id], nil
}

type executionReaderStub struct{}

func (s *executionReaderStub) GetExecution(_ context.Context, _ uuid.UUID) (*build.BuildExecution, error) {
	return nil, nil
}

type executionReaderWithDataStub struct {
	byID    map[uuid.UUID]*build.BuildExecution
	byBuild map[uuid.UUID][]build.BuildExecution
}

func (s *executionReaderWithDataStub) GetExecution(_ context.Context, id uuid.UUID) (*build.BuildExecution, error) {
	if s == nil || s.byID == nil {
		return nil, nil
	}
	return s.byID[id], nil
}

func (s *executionReaderWithDataStub) GetBuildExecutions(_ context.Context, buildID uuid.UUID, _ int, _ int) ([]build.BuildExecution, int64, error) {
	if s == nil || s.byBuild == nil {
		return nil, 0, nil
	}
	rows := s.byBuild[buildID]
	return rows, int64(len(rows)), nil
}

type imageRepoStub struct {
	byTenantName map[string]*image.Image
}

func (s *imageRepoStub) key(tenantID uuid.UUID, name string) string {
	return tenantID.String() + ":" + name
}

func (s *imageRepoStub) FindByTenantAndName(_ context.Context, tenantID uuid.UUID, name string) (*image.Image, error) {
	return s.byTenantName[s.key(tenantID, name)], nil
}

func (s *imageRepoStub) Save(_ context.Context, img *image.Image) error {
	if s.byTenantName == nil {
		s.byTenantName = map[string]*image.Image{}
	}
	s.byTenantName[s.key(img.TenantID(), img.Name())] = img
	return nil
}

func (s *imageRepoStub) Update(_ context.Context, img *image.Image) error {
	if s.byTenantName == nil {
		s.byTenantName = map[string]*image.Image{}
	}
	s.byTenantName[s.key(img.TenantID(), img.Name())] = img
	return nil
}

type versionRepoStub struct {
	byImageVersion map[string]*image.ImageVersion
}

func (s *versionRepoStub) key(imageID uuid.UUID, version string) string {
	return imageID.String() + ":" + version
}

func (s *versionRepoStub) FindByImageAndVersion(_ context.Context, imageID uuid.UUID, version string) (*image.ImageVersion, error) {
	return s.byImageVersion[s.key(imageID, version)], nil
}

func (s *versionRepoStub) Save(_ context.Context, v *image.ImageVersion) error {
	if s.byImageVersion == nil {
		s.byImageVersion = map[string]*image.ImageVersion{}
	}
	s.byImageVersion[s.key(v.ImageID(), v.Version())] = v
	return nil
}

func (s *versionRepoStub) Update(_ context.Context, v *image.ImageVersion) error {
	if s.byImageVersion == nil {
		s.byImageVersion = map[string]*image.ImageVersion{}
	}
	s.byImageVersion[s.key(v.ImageID(), v.Version())] = v
	return nil
}

func makeSuccessfulKanikoBuild(t *testing.T) *build.Build {
	t.Helper()

	tenantID := uuid.New()
	projectID := uuid.New()
	creator := uuid.New()
	manifest := build.BuildManifest{
		Name:         "sample-build",
		Type:         build.BuildTypeKaniko,
		BaseImage:    "alpine:3.20",
		Instructions: []string{"RUN echo ok"},
		Environment:  map[string]string{},
		Tags:         []string{"v1.0.0"},
		Metadata:     map[string]interface{}{},
		BuildConfig: &build.BuildConfig{
			BuildType:         build.BuildTypeKaniko,
			SBOMTool:          build.SBOMToolSyft,
			ScanTool:          build.ScanToolTrivy,
			RegistryType:      build.RegistryTypeS3,
			SecretManagerType: build.SecretManagerAWSSM,
			Dockerfile:        "Dockerfile",
			BuildContext:      ".",
			RegistryRepo:      "registry.gitlab.com/imagefactoryoss/imagefactory/image-factory-backend:v1.0.0",
		},
	}
	b, err := build.NewBuild(tenantID, projectID, manifest, &creator)
	if err != nil {
		t.Fatalf("failed to build fixture: %v", err)
	}
	return b
}

func TestEventSubscriber_AutoIngestsSuccessfulBuild(t *testing.T) {
	b := makeSuccessfulKanikoBuild(t)
	buildRepo := &buildRepoStub{byID: map[uuid.UUID]*build.Build{b.ID(): b}}
	imageRepo := &imageRepoStub{byTenantName: map[string]*image.Image{}}
	versionRepo := &versionRepoStub{byImageVersion: map[string]*image.ImageVersion{}}
	subscriber := NewEventSubscriber(buildRepo, &executionReaderStub{}, imageRepo, versionRepo, zap.NewNop())

	subscriber.HandleBuildExecutionCompleted(context.Background(), messaging.Event{
		Type:     messaging.EventTypeBuildExecutionCompleted,
		TenantID: b.TenantID().String(),
		Payload: map[string]interface{}{
			"build_id": b.ID().String(),
			"status":   "success",
			"metadata": map[string]interface{}{
				"execution_id": uuid.New().String(),
			},
		},
	})

	imageKey := b.TenantID().String() + ":image-factory"
	img := imageRepo.byTenantName[imageKey]
	if img == nil {
		t.Fatalf("expected catalog image to be created")
	}
	if img.Visibility() != image.VisibilityTenant {
		t.Fatalf("expected tenant visibility, got %s", img.Visibility())
	}
	if img.Status() != image.StatusPublished {
		t.Fatalf("expected published status, got %s", img.Status())
	}

	versionKey := img.ID().String() + ":v1.0.0"
	if versionRepo.byImageVersion[versionKey] == nil {
		t.Fatalf("expected image version to be created")
	}
}

func TestEventSubscriber_IgnoresNonSuccessEvents(t *testing.T) {
	b := makeSuccessfulKanikoBuild(t)
	buildRepo := &buildRepoStub{byID: map[uuid.UUID]*build.Build{b.ID(): b}}
	imageRepo := &imageRepoStub{byTenantName: map[string]*image.Image{}}
	versionRepo := &versionRepoStub{byImageVersion: map[string]*image.ImageVersion{}}
	subscriber := NewEventSubscriber(buildRepo, &executionReaderStub{}, imageRepo, versionRepo, zap.NewNop())

	subscriber.HandleBuildExecutionCompleted(context.Background(), messaging.Event{
		Type:     messaging.EventTypeBuildExecutionCompleted,
		TenantID: b.TenantID().String(),
		Payload: map[string]interface{}{
			"build_id": b.ID().String(),
			"status":   "failed",
		},
	})

	if len(imageRepo.byTenantName) != 0 {
		t.Fatalf("expected no images to be created for failed events")
	}
	if len(versionRepo.byImageVersion) != 0 {
		t.Fatalf("expected no versions to be created for failed events")
	}
}

func TestEventSubscriber_ResolvesExecutionWhenEventMetadataUsesCamelCase(t *testing.T) {
	b := makeSuccessfulKanikoBuild(t)
	buildRepo := &buildRepoStub{byID: map[uuid.UUID]*build.Build{b.ID(): b}}
	imageRepo := &imageRepoStub{byTenantName: map[string]*image.Image{}}
	versionRepo := &versionRepoStub{byImageVersion: map[string]*image.ImageVersion{}}

	executionID := uuid.New()
	artifacts := []byte(`[
		{"name":"image-digest","type":"pipeline-result","value":"sha256:feedfacefeedfacefeedfacefeedfacefeedfacefeedfacefeedfacefeedface"},
		{"name":"layers-json","type":"pipeline-result","value":"[{\"digest\":\"sha256:aaa\"}]"}
	]`)
	executionReader := &executionReaderWithDataStub{
		byID: map[uuid.UUID]*build.BuildExecution{
			executionID: {
				ID:        executionID,
				BuildID:   b.ID(),
				Status:    build.ExecutionSuccess,
				Artifacts: artifacts,
			},
		},
	}

	subscriber := NewEventSubscriber(buildRepo, executionReader, imageRepo, versionRepo, zap.NewNop())

	subscriber.HandleBuildExecutionCompleted(context.Background(), messaging.Event{
		Type:     messaging.EventTypeBuildExecutionCompleted,
		TenantID: b.TenantID().String(),
		Payload: map[string]interface{}{
			"build_id": b.ID().String(),
			"status":   "success",
			"metadata": map[string]interface{}{
				"executionId": executionID.String(),
			},
		},
	})

	img := imageRepo.byTenantName[b.TenantID().String()+":image-factory"]
	if img == nil {
		t.Fatalf("expected catalog image to be created")
	}
	v := versionRepo.byImageVersion[img.ID().String()+":v1.0.0"]
	if v == nil {
		t.Fatalf("expected image version to be created")
	}
	if got := v.Digest(); got == nil || *got == "" {
		t.Fatalf("expected digest to be populated from execution artifacts")
	}
}

func TestEventSubscriber_FallsBackToLatestBuildExecutionWhenExecutionIDMissing(t *testing.T) {
	b := makeSuccessfulKanikoBuild(t)
	buildRepo := &buildRepoStub{byID: map[uuid.UUID]*build.Build{b.ID(): b}}
	imageRepo := &imageRepoStub{byTenantName: map[string]*image.Image{}}
	versionRepo := &versionRepoStub{byImageVersion: map[string]*image.ImageVersion{}}

	artifacts := []byte(`[
		{"name":"push.image-digest","type":"taskrun-result","value":"sha256:cafebabecafebabecafebabecafebabecafebabecafebabecafebabecafebabe"}
	]`)
	successExecution := build.BuildExecution{
		ID:        uuid.New(),
		BuildID:   b.ID(),
		Status:    build.ExecutionSuccess,
		Artifacts: artifacts,
	}
	olderRunning := build.BuildExecution{
		ID:      uuid.New(),
		BuildID: b.ID(),
		Status:  build.ExecutionRunning,
	}
	executionReader := &executionReaderWithDataStub{
		byBuild: map[uuid.UUID][]build.BuildExecution{
			b.ID(): {olderRunning, successExecution},
		},
	}

	subscriber := NewEventSubscriber(buildRepo, executionReader, imageRepo, versionRepo, zap.NewNop())

	subscriber.HandleBuildExecutionCompleted(context.Background(), messaging.Event{
		Type:     messaging.EventTypeBuildExecutionCompleted,
		TenantID: b.TenantID().String(),
		Payload: map[string]interface{}{
			"build_id": b.ID().String(),
			"status":   "success",
		},
	})

	img := imageRepo.byTenantName[b.TenantID().String()+":image-factory"]
	if img == nil {
		t.Fatalf("expected catalog image to be created")
	}
	v := versionRepo.byImageVersion[img.ID().String()+":v1.0.0"]
	if v == nil {
		t.Fatalf("expected image version to be created")
	}
	if got := v.Digest(); got == nil || *got == "" {
		t.Fatalf("expected digest to be populated from fallback execution artifacts")
	}
}

func TestEventSubscriber_StrictExecutionResolutionDisablesFallback(t *testing.T) {
	b := makeSuccessfulKanikoBuild(t)
	buildRepo := &buildRepoStub{byID: map[uuid.UUID]*build.Build{b.ID(): b}}
	imageRepo := &imageRepoStub{byTenantName: map[string]*image.Image{}}
	versionRepo := &versionRepoStub{byImageVersion: map[string]*image.ImageVersion{}}

	executionReader := &executionReaderWithDataStub{
		byBuild: map[uuid.UUID][]build.BuildExecution{
			b.ID(): {
				{
					ID:        uuid.New(),
					BuildID:   b.ID(),
					Status:    build.ExecutionSuccess,
					Artifacts: []byte(`[{"name":"image-digest","value":"sha256:deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"}]`),
				},
			},
		},
	}

	subscriber := NewEventSubscriber(buildRepo, executionReader, imageRepo, versionRepo, zap.NewNop())
	subscriber.SetStrictExecutionResolution(true)
	subscriber.HandleBuildExecutionCompleted(context.Background(), messaging.Event{
		Type:     messaging.EventTypeBuildExecutionCompleted,
		TenantID: b.TenantID().String(),
		Payload: map[string]interface{}{
			"build_id": b.ID().String(),
			"status":   "success",
		},
	})

	img := imageRepo.byTenantName[b.TenantID().String()+":image-factory"]
	if img == nil {
		t.Fatalf("expected catalog image to be created")
	}
	v := versionRepo.byImageVersion[img.ID().String()+":v1.0.0"]
	if v == nil {
		t.Fatalf("expected image version to be created")
	}
	if got := v.Digest(); got != nil && *got != "" {
		t.Fatalf("expected digest to remain empty when strict mode disables fallback execution lookup")
	}

	snapshot := subscriber.Snapshot()
	if snapshot.MissingExecutionID != 1 {
		t.Fatalf("expected MissingExecutionID=1, got %d", snapshot.MissingExecutionID)
	}
	if snapshot.FallbackExecutionLookupSkipped != 1 {
		t.Fatalf("expected FallbackExecutionLookupSkipped=1, got %d", snapshot.FallbackExecutionLookupSkipped)
	}
	if snapshot.FallbackExecutionLookupAttempts != 0 {
		t.Fatalf("expected no fallback attempts in strict mode, got %d", snapshot.FallbackExecutionLookupAttempts)
	}
}
