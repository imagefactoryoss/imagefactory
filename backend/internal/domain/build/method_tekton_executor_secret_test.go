package build

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

type stubRepositoryGitAuthResolver struct {
	data map[string][]byte
	err  error
}

func (s *stubRepositoryGitAuthResolver) ResolveGitAuthSecretData(ctx context.Context, projectID uuid.UUID) (map[string][]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func TestUpsertDockerConfigSecret_CreatesAndUpdates(t *testing.T) {
	k8sClient := k8sfake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "image-factory-test"}},
	)
	namespace := "image-factory-test"

	first := []byte(`{"auths":{"ghcr.io":{"auth":"a"}}}`)
	if err := upsertDockerConfigSecret(context.Background(), k8sClient, namespace, first); err != nil {
		t.Fatalf("upsertDockerConfigSecret create returned error: %v", err)
	}

	secret, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), "docker-config", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to load created secret: %v", err)
	}
	if secret.Type != corev1.SecretTypeDockerConfigJson {
		t.Fatalf("expected docker config secret type, got %q", secret.Type)
	}
	if string(secret.Data[corev1.DockerConfigJsonKey]) != string(first) {
		t.Fatalf("unexpected initial docker config content: %s", string(secret.Data[corev1.DockerConfigJsonKey]))
	}
	if string(secret.Data["config.json"]) != string(first) {
		t.Fatalf("unexpected initial config.json content: %s", string(secret.Data["config.json"]))
	}

	second := []byte(`{"auths":{"ghcr.io":{"auth":"b"}}}`)
	if err := upsertDockerConfigSecret(context.Background(), k8sClient, namespace, second); err != nil {
		t.Fatalf("upsertDockerConfigSecret update returned error: %v", err)
	}

	updated, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), "docker-config", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to load updated secret: %v", err)
	}
	if string(updated.Data[corev1.DockerConfigJsonKey]) != string(second) {
		t.Fatalf("expected updated docker config content, got %s", string(updated.Data[corev1.DockerConfigJsonKey]))
	}
	if string(updated.Data["config.json"]) != string(second) {
		t.Fatalf("expected updated config.json content, got %s", string(updated.Data["config.json"]))
	}
}

func TestMethodRequiresDockerConfigSecret(t *testing.T) {
	if !methodRequiresDockerConfigSecret(BuildMethodDocker) {
		t.Fatalf("expected docker to require docker-config secret")
	}
	if !methodRequiresDockerConfigSecret(BuildMethodKaniko) {
		t.Fatalf("expected kaniko to require docker-config secret")
	}
	if !methodRequiresDockerConfigSecret(BuildMethodBuildx) {
		t.Fatalf("expected buildx to require docker-config secret")
	}
	if methodRequiresDockerConfigSecret(BuildMethodPacker) {
		t.Fatalf("did not expect packer to require docker-config secret")
	}
}

func TestUpsertDockerConfigSecret_ProducesValidDockerConfigJSON(t *testing.T) {
	k8sClient := k8sfake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "image-factory-test"}},
	)
	namespace := "image-factory-test"
	payload := []byte(`{"auths":{"registry.example.com":{"username":"u","password":"p","auth":"dTpw"}}}`)
	if err := upsertDockerConfigSecret(context.Background(), k8sClient, namespace, payload); err != nil {
		t.Fatalf("upsertDockerConfigSecret returned error: %v", err)
	}

	secret, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), "docker-config", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to load secret: %v", err)
	}
	var dockerConfig map[string]interface{}
	if err := json.Unmarshal(secret.Data[corev1.DockerConfigJsonKey], &dockerConfig); err != nil {
		t.Fatalf("expected valid docker config json, got error: %v", err)
	}
	if err := json.Unmarshal(secret.Data["config.json"], &dockerConfig); err != nil {
		t.Fatalf("expected valid config.json content, got error: %v", err)
	}
}

func TestUpsertDockerConfigSecret_BackfillsConfigJSONForLegacySecret(t *testing.T) {
	namespace := "image-factory-test"
	payload := []byte(`{"auths":{"ghcr.io":{"auth":"legacy"}}}`)
	k8sClient := k8sfake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "docker-config", Namespace: namespace},
			Type:       corev1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: payload,
			},
		},
	)

	if err := upsertDockerConfigSecret(context.Background(), k8sClient, namespace, payload); err != nil {
		t.Fatalf("upsertDockerConfigSecret returned error: %v", err)
	}

	secret, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), "docker-config", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to load secret: %v", err)
	}
	if string(secret.Data["config.json"]) != string(payload) {
		t.Fatalf("expected config.json to be backfilled, got %s", string(secret.Data["config.json"]))
	}
}

func TestUpsertOpaqueSecret_CreatesAndUpdates(t *testing.T) {
	k8sClient := k8sfake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "image-factory-test"}},
	)
	namespace := "image-factory-test"

	first := map[string][]byte{
		"auth_type": []byte("token"),
		"username":  []byte("token"),
		"token":     []byte("abc123"),
	}
	if err := upsertOpaqueSecret(context.Background(), k8sClient, namespace, "git-auth", first, map[string]string{"kind": "repository-auth"}); err != nil {
		t.Fatalf("upsertOpaqueSecret create returned error: %v", err)
	}

	secret, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), "git-auth", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to load created secret: %v", err)
	}
	if secret.Type != corev1.SecretTypeOpaque {
		t.Fatalf("expected opaque secret type, got %q", secret.Type)
	}
	if string(secret.Data["token"]) != "abc123" {
		t.Fatalf("unexpected initial token content")
	}

	second := map[string][]byte{
		"auth_type":      []byte("ssh_key"),
		"ssh-privatekey": []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----"),
	}
	if err := upsertOpaqueSecret(context.Background(), k8sClient, namespace, "git-auth", second, map[string]string{"kind": "repository-auth"}); err != nil {
		t.Fatalf("upsertOpaqueSecret update returned error: %v", err)
	}

	updated, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), "git-auth", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to load updated secret: %v", err)
	}
	if string(updated.Data["auth_type"]) != "ssh_key" {
		t.Fatalf("expected updated auth_type, got %q", string(updated.Data["auth_type"]))
	}
	if len(updated.Data["ssh-privatekey"]) == 0 {
		t.Fatalf("expected ssh-privatekey data after update")
	}
}

func TestOpaqueSecretDataEqual(t *testing.T) {
	base := map[string][]byte{
		"a": []byte("1"),
		"b": []byte("2"),
	}
	same := map[string][]byte{
		"a": []byte("1"),
		"b": []byte("2"),
	}
	diff := map[string][]byte{
		"a": []byte("1"),
		"b": []byte("3"),
	}
	if !opaqueSecretDataEqual(base, same) {
		t.Fatalf("expected equal maps to return true")
	}
	if opaqueSecretDataEqual(base, diff) {
		t.Fatalf("expected different maps to return false")
	}
}

func TestReconcileGitAuthSecret_NoResolver(t *testing.T) {
	k8sClient := k8sfake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "image-factory-test"}},
	)
	executor := &MethodTektonExecutor{}
	include, err := executor.reconcileGitAuthSecret(context.Background(), uuid.New(), "image-factory-test", uuid.New(), k8sClient)
	if err != nil {
		t.Fatalf("reconcileGitAuthSecret returned error: %v", err)
	}
	if include {
		t.Fatalf("expected includeGitAuth=false when resolver is not configured")
	}
}

func TestReconcileGitAuthSecret_ResolverError(t *testing.T) {
	k8sClient := k8sfake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "image-factory-test"}},
	)
	executor := &MethodTektonExecutor{
		repositoryAuth: &stubRepositoryGitAuthResolver{err: errors.New("boom")},
	}
	include, err := executor.reconcileGitAuthSecret(context.Background(), uuid.New(), "image-factory-test", uuid.New(), k8sClient)
	if err == nil {
		t.Fatalf("expected error when resolver fails")
	}
	if include {
		t.Fatalf("expected includeGitAuth=false when resolver returns error")
	}
}

func TestReconcileGitAuthSecret_CreatesSecretAndReturnsInclude(t *testing.T) {
	namespace := "image-factory-test"
	k8sClient := k8sfake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}},
	)
	executor := &MethodTektonExecutor{
		repositoryAuth: &stubRepositoryGitAuthResolver{
			data: map[string][]byte{
				"auth_type": []byte("token"),
				"username":  []byte("token"),
				"token":     []byte("abc123"),
			},
		},
		service: &stubExecutionService{},
	}
	include, err := executor.reconcileGitAuthSecret(context.Background(), uuid.New(), namespace, uuid.New(), k8sClient)
	if err != nil {
		t.Fatalf("reconcileGitAuthSecret returned error: %v", err)
	}
	if !include {
		t.Fatalf("expected includeGitAuth=true when resolver returns secret data")
	}
	secret, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), "git-auth", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected git-auth secret to be created: %v", err)
	}
	if string(secret.Data["token"]) != "abc123" {
		t.Fatalf("unexpected git-auth token data")
	}
}
