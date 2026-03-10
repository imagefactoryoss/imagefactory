package build

import (
	"context"
	"strings"
	"testing"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestShouldUseAnonymousInternalRegistry_DefaultHost(t *testing.T) {
	t.Setenv("IF_INTERNAL_REGISTRY_HOSTS", "")
	if !shouldUseAnonymousInternalRegistry("image-factory-registry:5000/published/team/app:latest") {
		t.Fatalf("expected default internal registry host to allow anonymous docker-config")
	}
}

func TestShouldUseAnonymousInternalRegistry_EnvOverride(t *testing.T) {
	t.Setenv("IF_INTERNAL_REGISTRY_HOSTS", "registry-a.local:5000, registry-b.local:5000")
	if !shouldUseAnonymousInternalRegistry("registry-b.local:5000/published/team/app:latest") {
		t.Fatalf("expected configured registry host to allow anonymous docker-config")
	}
	if shouldUseAnonymousInternalRegistry("registry-c.local:5000/published/team/app:latest") {
		t.Fatalf("did not expect unconfigured registry host to allow anonymous docker-config")
	}
}

func TestRegistryHostFromImageRef(t *testing.T) {
	if host, ok := registryHostFromImageRef("image-factory-registry:5000/published/team/app:latest"); !ok || host != "image-factory-registry:5000" {
		t.Fatalf("expected host parse success, got host=%q ok=%v", host, ok)
	}
	if _, ok := registryHostFromImageRef("library/nginx:latest"); ok {
		t.Fatalf("did not expect docker hub style ref to be parsed as explicit host")
	}
}

func tektonPrereqObjects(namespace string) []runtime.Object {
	return []runtime.Object{
		&tektonv1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "image-factory-build-v1-kaniko",
				Namespace: namespace,
			},
		},
		&tektonv1.Task{
			ObjectMeta: metav1.ObjectMeta{Name: "git-clone", Namespace: namespace},
		},
		&tektonv1.Task{
			ObjectMeta: metav1.ObjectMeta{Name: "docker-build", Namespace: namespace},
		},
		&tektonv1.Task{
			ObjectMeta: metav1.ObjectMeta{Name: "buildx", Namespace: namespace},
		},
		&tektonv1.Task{
			ObjectMeta: metav1.ObjectMeta{Name: "kaniko-no-push", Namespace: namespace},
		},
		&tektonv1.Task{
			ObjectMeta: metav1.ObjectMeta{Name: "scan-image", Namespace: namespace},
		},
		&tektonv1.Task{
			ObjectMeta: metav1.ObjectMeta{Name: "generate-sbom", Namespace: namespace},
		},
		&tektonv1.Task{
			ObjectMeta: metav1.ObjectMeta{Name: "push-image", Namespace: namespace},
		},
		&tektonv1.Task{
			ObjectMeta: metav1.ObjectMeta{Name: "sign-image", Namespace: namespace},
		},
		&tektonv1.Task{
			ObjectMeta: metav1.ObjectMeta{Name: "packer", Namespace: namespace},
		},
	}
}

func tektonPrereqObjectsLegacyKaniko(namespace string) []runtime.Object {
	return []runtime.Object{
		&tektonv1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "image-factory-build-v1-kaniko",
				Namespace: namespace,
			},
		},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "git-clone", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "docker-build", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "buildx", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "kaniko", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "scan-image", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "generate-sbom", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "push-image", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "sign-image", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "packer", Namespace: namespace}},
	}
}

func TestPreflightPipelineRun_PipelineRefModeFailsWhenRequiredTaskMissing(t *testing.T) {
	executor := &MethodTektonExecutor{}
	namespace := "image-factory-test"
	pipelineRunYAML := `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: if-pr-1
spec:
  pipelineRef:
    name: image-factory-build-v1-kaniko
  workspaces:
  - name: dockerconfig
    secret:
      secretName: docker-config
`

	k8sClient := k8sfake.NewSimpleClientset()
	tektonClient := tektonfake.NewSimpleClientset(
		&tektonv1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "image-factory-build-v1-kaniko",
				Namespace: namespace,
			},
		},
	)

	err := executor.preflightPipelineRun(context.Background(), namespace, pipelineRunYAML, k8sClient, tektonClient)
	if err == nil {
		t.Fatalf("expected preflight error for missing task")
	}
	if !strings.Contains(err.Error(), "missing tekton task: git-clone") {
		t.Fatalf("expected missing git-clone task error, got %v", err)
	}
}

func TestPreflightPipelineRun_FailsWhenRequiredSecretMissing(t *testing.T) {
	executor := &MethodTektonExecutor{}
	namespace := "image-factory-test"
	pipelineRunYAML := `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: if-pr-1
spec:
  pipelineRef:
    name: image-factory-build-v1-kaniko
  workspaces:
  - name: dockerconfig
    secret:
      secretName: docker-config
`

	tektonClient := tektonfake.NewSimpleClientset(tektonPrereqObjects(namespace)...)
	k8sClient := k8sfake.NewSimpleClientset()

	err := executor.preflightPipelineRun(context.Background(), namespace, pipelineRunYAML, k8sClient, tektonClient)
	if err == nil {
		t.Fatalf("expected preflight error for missing secret")
	}
	if !strings.Contains(err.Error(), "missing required secret \"docker-config\"") {
		t.Fatalf("expected missing docker-config secret error, got %v", err)
	}
}

func TestPreflightPipelineRun_FailsWhenGitAuthSecretMissing(t *testing.T) {
	executor := &MethodTektonExecutor{}
	namespace := "image-factory-test"
	pipelineRunYAML := `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: if-pr-1
spec:
  pipelineRef:
    name: image-factory-build-v1-kaniko
  workspaces:
  - name: dockerconfig
    secret:
      secretName: docker-config
  - name: git-auth
    secret:
      secretName: git-auth
`

	tektonClient := tektonfake.NewSimpleClientset(tektonPrereqObjects(namespace)...)
	k8sClient := k8sfake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: namespace},
			Type:       corev1.SecretTypeDockerConfigJson,
		},
	)

	err := executor.preflightPipelineRun(context.Background(), namespace, pipelineRunYAML, k8sClient, tektonClient)
	if err == nil {
		t.Fatalf("expected preflight error for missing git-auth secret")
	}
	if !strings.Contains(err.Error(), "missing required secret \"git-auth\"") {
		t.Fatalf("expected missing git-auth secret error, got %v", err)
	}
}

func TestPreflightPipelineRun_PassesWithLegacyKanikoTaskAlias(t *testing.T) {
	executor := &MethodTektonExecutor{}
	namespace := "image-factory-test"
	pipelineRunYAML := `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: if-pr-1
spec:
  pipelineRef:
    name: image-factory-build-v1-kaniko
  workspaces:
  - name: dockerconfig
    secret:
      secretName: docker-config
`

	tektonClient := tektonfake.NewSimpleClientset(tektonPrereqObjectsLegacyKaniko(namespace)...)
	k8sClient := k8sfake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: namespace},
			Type:       corev1.SecretTypeDockerConfigJson,
		},
	)

	if err := executor.preflightPipelineRun(context.Background(), namespace, pipelineRunYAML, k8sClient, tektonClient); err != nil {
		t.Fatalf("expected preflight to pass with legacy kaniko task alias, got: %v", err)
	}
}

func TestPreflightPipelineRun_PassesWhenScanAndSBOMDisabled(t *testing.T) {
	executor := &MethodTektonExecutor{}
	namespace := "image-factory-test"
	pipelineRunYAML := `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: if-pr-1
spec:
  pipelineRef:
    name: image-factory-build-v1-kaniko
  params:
  - name: enable-scan
    value: "false"
  - name: enable-sbom
    value: "false"
  workspaces:
  - name: dockerconfig
    secret:
      secretName: docker-config
`

	tektonClient := tektonfake.NewSimpleClientset(
		&tektonv1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "image-factory-build-v1-kaniko",
				Namespace: namespace,
			},
		},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "git-clone", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "kaniko-no-push", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "push-image", Namespace: namespace}},
	)
	k8sClient := k8sfake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: namespace},
			Type:       corev1.SecretTypeDockerConfigJson,
		},
	)

	if err := executor.preflightPipelineRun(context.Background(), namespace, pipelineRunYAML, k8sClient, tektonClient); err != nil {
		t.Fatalf("expected preflight to pass without scan/sbom tasks, got: %v", err)
	}
}

func TestPreflightPipelineRun_FailsWhenSignEnabledAndTaskMissing(t *testing.T) {
	executor := &MethodTektonExecutor{}
	namespace := "image-factory-test"
	pipelineRunYAML := `
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: if-pr-1
spec:
  pipelineRef:
    name: image-factory-build-v1-kaniko
  params:
  - name: enable-scan
    value: "false"
  - name: enable-sbom
    value: "false"
  - name: enable-sign
    value: "true"
  workspaces:
  - name: dockerconfig
    secret:
      secretName: docker-config
`

	tektonClient := tektonfake.NewSimpleClientset(
		&tektonv1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "image-factory-build-v1-kaniko",
				Namespace: namespace,
			},
		},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "git-clone", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "kaniko-no-push", Namespace: namespace}},
		&tektonv1.Task{ObjectMeta: metav1.ObjectMeta{Name: "push-image", Namespace: namespace}},
	)
	k8sClient := k8sfake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: namespace},
			Type:       corev1.SecretTypeDockerConfigJson,
		},
	)

	err := executor.preflightPipelineRun(context.Background(), namespace, pipelineRunYAML, k8sClient, tektonClient)
	if err == nil {
		t.Fatalf("expected preflight error for missing sign-image task")
	}
	if !strings.Contains(err.Error(), "missing tekton task: sign-image") {
		t.Fatalf("expected missing sign-image task error, got %v", err)
	}
}
