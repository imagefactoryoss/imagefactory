package build

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/google/uuid"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"
)

type smokeNamespaceManager struct {
	namespace string
}

func (m *smokeNamespaceManager) EnsureNamespace(ctx context.Context, tenantID uuid.UUID) (string, error) {
	return m.namespace, nil
}
func (m *smokeNamespaceManager) DeleteNamespace(ctx context.Context, tenantID uuid.UUID) error {
	return nil
}
func (m *smokeNamespaceManager) GetNamespace(tenantID uuid.UUID) string { return m.namespace }

type smokePipelineManager struct {
	createdYAML string
}

func (m *smokePipelineManager) CreatePipelineRun(ctx context.Context, namespace, yamlContent string) (*tektonv1.PipelineRun, error) {
	m.createdYAML = yamlContent
	var pr tektonv1.PipelineRun
	if err := yaml.Unmarshal([]byte(yamlContent), &pr); err != nil {
		return nil, err
	}
	if pr.Name == "" {
		pr.Name = "if-pr-smoke"
	}
	pr.Namespace = namespace
	return &pr, nil
}
func (m *smokePipelineManager) GetPipelineRun(ctx context.Context, namespace, name string) (*tektonv1.PipelineRun, error) {
	return nil, nil
}
func (m *smokePipelineManager) ListPipelineRuns(ctx context.Context, namespace string, limit int) ([]*tektonv1.PipelineRun, error) {
	return nil, nil
}
func (m *smokePipelineManager) DeletePipelineRun(ctx context.Context, namespace, name string) error {
	return nil
}
func (m *smokePipelineManager) GetLogs(ctx context.Context, namespace, pipelineRunName string) (map[string]string, error) {
	return map[string]string{}, nil
}

type smokeTemplateEngine struct{}

func (e smokeTemplateEngine) Render(tpl string, data interface{}) (string, error) {
	funcMap := template.FuncMap{
		"default": func(defaultValue interface{}, given interface{}) interface{} {
			switch v := given.(type) {
			case string:
				if v == "" {
					return defaultValue
				}
				return v
			case nil:
				return defaultValue
			default:
				return given
			}
		},
	}
	normalized := strings.ReplaceAll(tpl, `\"`, `"`)
	parsed, err := template.New("tekton").Funcs(funcMap).Parse(normalized)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := parsed.Execute(&out, data); err != nil {
		return "", err
	}
	return out.String(), nil
}

type smokeExecutionService struct {
	execution       *BuildExecution
	updatedMetadata []byte
}

func (s *smokeExecutionService) StartBuild(ctx context.Context, configID uuid.UUID, createdBy uuid.UUID) (*BuildExecution, error) {
	return nil, nil
}
func (s *smokeExecutionService) CancelBuild(ctx context.Context, executionID uuid.UUID) error {
	return nil
}
func (s *smokeExecutionService) RetryBuild(ctx context.Context, executionID uuid.UUID, createdBy uuid.UUID) (*BuildExecution, error) {
	return nil, nil
}
func (s *smokeExecutionService) GetExecution(ctx context.Context, executionID uuid.UUID) (*BuildExecution, error) {
	return s.execution, nil
}
func (s *smokeExecutionService) GetBuildExecutions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]BuildExecution, int64, error) {
	return nil, 0, nil
}
func (s *smokeExecutionService) ListRunningExecutions(ctx context.Context) ([]BuildExecution, error) {
	return nil, nil
}
func (s *smokeExecutionService) GetLogs(ctx context.Context, executionID uuid.UUID, limit, offset int) ([]ExecutionLog, int64, error) {
	return nil, 0, nil
}
func (s *smokeExecutionService) AddLog(ctx context.Context, executionID uuid.UUID, level LogLevel, message string, metadata []byte) error {
	return nil
}
func (s *smokeExecutionService) UpdateExecutionStatus(ctx context.Context, executionID uuid.UUID, status ExecutionStatus) error {
	return nil
}
func (s *smokeExecutionService) UpdateExecutionMetadata(ctx context.Context, executionID uuid.UUID, metadata []byte) error {
	s.updatedMetadata = metadata
	return nil
}
func (s *smokeExecutionService) TryAcquireMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
	return false, nil
}
func (s *smokeExecutionService) RenewMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string, ttl time.Duration) (bool, error) {
	return false, nil
}
func (s *smokeExecutionService) ReleaseMonitoringLease(ctx context.Context, executionID uuid.UUID, owner string) error {
	return nil
}
func (s *smokeExecutionService) CompleteExecution(ctx context.Context, executionID uuid.UUID, success bool, errorMsg string, artifacts []byte) error {
	return nil
}
func (s *smokeExecutionService) CleanupOldExecutions(ctx context.Context, olderThan time.Duration) error {
	return nil
}

type smokeRegistryAuthResolver struct {
	dockerConfig []byte
}

func (r *smokeRegistryAuthResolver) ResolveDockerConfigJSON(ctx context.Context, registryAuthID uuid.UUID) ([]byte, error) {
	return r.dockerConfig, nil
}

type smokeRepositoryAuthResolver struct {
	data map[string][]byte
}

func (r *smokeRepositoryAuthResolver) ResolveGitAuthSecretData(ctx context.Context, projectID uuid.UUID) (map[string][]byte, error) {
	return r.data, nil
}

func tektonPrereqRuntimeObjects(namespace string) []runtime.Object {
	return []runtime.Object{
		&tektonv1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{Name: "image-factory-build-v1-kaniko", Namespace: namespace},
		},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "git-clone", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "docker-build", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "buildx", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "kaniko-no-push", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "scan-image", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "generate-sbom", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "push-image", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "sign-image", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "packer", Namespace: namespace}},
	}
}

func makeSmokeBuild(t *testing.T, registryAuthID uuid.UUID) *Build {
	t.Helper()
	tenantID := uuid.New()
	projectID := uuid.New()
	buildID := uuid.New()
	manifest := BuildManifest{
		Name: "tekton-smoke",
		Type: BuildTypeKaniko,
		Metadata: map[string]interface{}{
			"git_url": "https://github.com/acme/private-repo.git",
		},
		BuildConfig: &BuildConfig{
			BuildType:         BuildTypeKaniko,
			SBOMTool:          SBOMToolSyft,
			ScanTool:          ScanToolTrivy,
			RegistryType:      RegistryTypeHarbor,
			SecretManagerType: SecretManagerVault,
			Dockerfile:        "Dockerfile",
			BuildContext:      ".",
			RegistryRepo:      "ghcr.io/acme/private-repo:latest",
			RegistryAuthID:    &registryAuthID,
		},
	}
	return NewBuildFromDB(buildID, tenantID, projectID, manifest, BuildStatusQueued, time.Now().UTC(), time.Now().UTC(), nil)
}

func makeSmokeExecutor(namespace string, service BuildExecutionService, registryAuth RegistryDockerConfigResolver, repoAuth RepositoryGitAuthResolver, k8sClient *k8sfake.Clientset, tektonClient *tektonfake.Clientset, pipelineMgr PipelineManager) *MethodTektonExecutor {
	return &MethodTektonExecutor{
		k8sClient:      k8sClient,
		tektonClient:   tektonClient,
		logger:         zap.NewNop(),
		namespaceMgr:   &smokeNamespaceManager{namespace: namespace},
		pipelineMgr:    pipelineMgr,
		templateEngine: smokeTemplateEngine{},
		service:        service,
		registryAuth:   registryAuth,
		repositoryAuth: repoAuth,
		instanceID:     uuid.NewString(),
		running:        map[string]tektonExecutionRuntime{},
	}
}

func TestTektonExecuteWithConfig_Smoke_PrivateRepoAuthEnabled(t *testing.T) {
	namespace := "image-factory-test"
	k8sClient := k8sfake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}},
	)
	tektonClient := tektonfake.NewSimpleClientset(tektonPrereqRuntimeObjects(namespace)...)
	pipelineMgr := &smokePipelineManager{}

	executionID := uuid.New()
	service := &smokeExecutionService{
		execution: &BuildExecution{
			ID:      executionID,
			BuildID: uuid.New(),
			Status:  ExecutionPending,
			Metadata: func() json.RawMessage {
				return json.RawMessage(`{}`)
			}(),
		},
	}
	registryAuthID := uuid.New()
	build := makeSmokeBuild(t, registryAuthID)
	cfg, err := NewKanikoConfig(build.ID(), "Dockerfile", ".", build.Manifest().BuildConfig.RegistryRepo)
	if err != nil {
		t.Fatalf("failed to create kaniko config: %v", err)
	}
	executor := makeSmokeExecutor(
		namespace,
		service,
		&smokeRegistryAuthResolver{dockerConfig: []byte(`{"auths":{"ghcr.io":{"auth":"dGVzdDp0ZXN0"}}}`)},
		&smokeRepositoryAuthResolver{data: map[string][]byte{
			"auth_type": []byte("token"),
			"username":  []byte("token"),
			"token":     []byte("ghp_test_token"),
		}},
		k8sClient,
		tektonClient,
		pipelineMgr,
	)

	out, err := executor.executeWithConfig(context.Background(), executionID, build, cfg, BuildMethodKaniko)
	if err != nil {
		t.Fatalf("executeWithConfig returned error: %v", err)
	}
	if out == nil || out.Status != ExecutionRunning {
		t.Fatalf("expected running output, got %#v", out)
	}
	if !strings.Contains(pipelineMgr.createdYAML, "secretName: git-auth") {
		t.Fatalf("expected rendered PipelineRun to include git-auth secret workspace")
	}
	if !strings.Contains(pipelineMgr.createdYAML, "- name: enable-temp-scan-stage") {
		t.Fatalf("expected rendered PipelineRun to include enable-temp-scan-stage param")
	}
	if !strings.Contains(pipelineMgr.createdYAML, "- name: sbom-source") {
		t.Fatalf("expected rendered PipelineRun to include sbom-source param")
	}

	if _, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), "docker-config", metav1.GetOptions{}); err != nil {
		t.Fatalf("expected docker-config secret to be reconciled: %v", err)
	}
	if _, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), "git-auth", metav1.GetOptions{}); err != nil {
		t.Fatalf("expected git-auth secret to be reconciled: %v", err)
	}
	if len(service.updatedMetadata) == 0 {
		t.Fatalf("expected execution metadata to be updated with tekton refs")
	}
}

func TestTektonExecuteWithConfig_MetadataNull_DoesNotPanic(t *testing.T) {
	namespace := "image-factory-test-null-metadata"
	k8sClient := k8sfake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}},
	)
	tektonClient := tektonfake.NewSimpleClientset(tektonPrereqRuntimeObjects(namespace)...)
	pipelineMgr := &smokePipelineManager{}

	executionID := uuid.New()
	service := &smokeExecutionService{
		execution: &BuildExecution{
			ID:       executionID,
			BuildID:  uuid.New(),
			Status:   ExecutionPending,
			Metadata: json.RawMessage(`null`),
		},
	}

	registryAuthID := uuid.New()
	build := makeSmokeBuild(t, registryAuthID)
	cfg, err := NewKanikoConfig(build.ID(), "Dockerfile", ".", build.Manifest().BuildConfig.RegistryRepo)
	if err != nil {
		t.Fatalf("failed to create kaniko config: %v", err)
	}

	executor := makeSmokeExecutor(
		namespace,
		service,
		&smokeRegistryAuthResolver{dockerConfig: []byte(`{"auths":{"ghcr.io":{"auth":"dGVzdDp0ZXN0"}}}`)},
		&smokeRepositoryAuthResolver{data: map[string][]byte{
			"auth_type": []byte("token"),
			"username":  []byte("token"),
			"token":     []byte("ghp_test_token"),
		}},
		k8sClient,
		tektonClient,
		pipelineMgr,
	)

	out, err := executor.executeWithConfig(context.Background(), executionID, build, cfg, BuildMethodKaniko)
	if err != nil {
		t.Fatalf("executeWithConfig returned error: %v", err)
	}
	if out == nil || out.Status != ExecutionRunning {
		t.Fatalf("expected running execution, got %#v", out)
	}
	if len(service.updatedMetadata) == 0 {
		t.Fatalf("expected execution metadata update payload")
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(service.updatedMetadata, &payload); err != nil {
		t.Fatalf("failed to parse updated metadata payload: %v", err)
	}
	if _, ok := payload["tekton"]; !ok {
		t.Fatalf("expected tekton metadata in updated payload, got: %v", payload)
	}
}

func TestTektonExecuteWithConfig_Smoke_PrivateRepoAuthDisabled(t *testing.T) {
	namespace := "image-factory-test"
	k8sClient := k8sfake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}},
	)
	tektonClient := tektonfake.NewSimpleClientset(tektonPrereqRuntimeObjects(namespace)...)
	pipelineMgr := &smokePipelineManager{}

	executionID := uuid.New()
	service := &smokeExecutionService{
		execution: &BuildExecution{
			ID:      executionID,
			BuildID: uuid.New(),
			Status:  ExecutionPending,
			Metadata: func() json.RawMessage {
				return json.RawMessage(`{}`)
			}(),
		},
	}
	registryAuthID := uuid.New()
	build := makeSmokeBuild(t, registryAuthID)
	cfg, err := NewKanikoConfig(build.ID(), "Dockerfile", ".", build.Manifest().BuildConfig.RegistryRepo)
	if err != nil {
		t.Fatalf("failed to create kaniko config: %v", err)
	}
	executor := makeSmokeExecutor(
		namespace,
		service,
		&smokeRegistryAuthResolver{dockerConfig: []byte(`{"auths":{"ghcr.io":{"auth":"dGVzdDp0ZXN0"}}}`)},
		&smokeRepositoryAuthResolver{data: nil},
		k8sClient,
		tektonClient,
		pipelineMgr,
	)

	if _, err := executor.executeWithConfig(context.Background(), executionID, build, cfg, BuildMethodKaniko); err != nil {
		t.Fatalf("executeWithConfig returned error: %v", err)
	}
	if strings.Contains(pipelineMgr.createdYAML, "secretName: git-auth") {
		t.Fatalf("did not expect rendered PipelineRun to include git-auth secret workspace")
	}
	if _, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), "git-auth", metav1.GetOptions{}); err == nil {
		t.Fatalf("did not expect git-auth secret when repository auth resolver returns no data")
	}
}

func TestTektonExecuteWithConfig_Smoke_SigningEnabledWithKeySecret(t *testing.T) {
	namespace := "image-factory-test-signing"
	k8sClient := k8sfake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "cosign-key", Namespace: namespace}},
	)
	tektonClient := tektonfake.NewSimpleClientset(tektonPrereqRuntimeObjects(namespace)...)
	pipelineMgr := &smokePipelineManager{}

	executionID := uuid.New()
	service := &smokeExecutionService{
		execution: &BuildExecution{
			ID:      executionID,
			BuildID: uuid.New(),
			Status:  ExecutionPending,
			Metadata: func() json.RawMessage {
				return json.RawMessage(`{}`)
			}(),
		},
	}

	registryAuthID := uuid.New()
	build := makeSmokeBuild(t, registryAuthID)
	build.manifest.Metadata["enable_sign"] = true
	build.manifest.Metadata["sign_key_secret"] = "cosign-key"
	build.manifest.Metadata["enable_scan"] = false
	build.manifest.Metadata["enable_sbom"] = false
	cfg, err := NewKanikoConfig(build.ID(), "Dockerfile", ".", build.Manifest().BuildConfig.RegistryRepo)
	if err != nil {
		t.Fatalf("failed to create kaniko config: %v", err)
	}
	executor := makeSmokeExecutor(
		namespace,
		service,
		&smokeRegistryAuthResolver{dockerConfig: []byte(`{"auths":{"ghcr.io":{"auth":"dGVzdDp0ZXN0"}}}`)},
		&smokeRepositoryAuthResolver{data: nil},
		k8sClient,
		tektonClient,
		pipelineMgr,
	)

	if _, err := executor.executeWithConfig(context.Background(), executionID, build, cfg, BuildMethodKaniko); err != nil {
		t.Fatalf("executeWithConfig returned error: %v", err)
	}
	if !strings.Contains(pipelineMgr.createdYAML, "- name: enable-sign") || !strings.Contains(pipelineMgr.createdYAML, "value: \"true\"") {
		t.Fatalf("expected rendered PipelineRun to include enable-sign=true param")
	}
	if !strings.Contains(pipelineMgr.createdYAML, "name: signing-key") || !strings.Contains(pipelineMgr.createdYAML, "secretName: \"cosign-key\"") {
		t.Fatalf("expected rendered PipelineRun to include signing-key secret workspace")
	}
}
