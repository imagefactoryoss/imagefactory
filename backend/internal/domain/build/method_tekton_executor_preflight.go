package build

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

func (e *MethodTektonExecutor) preflightPipelineRun(
	ctx context.Context,
	namespace string,
	pipelineRunYAML string,
	k8sClient kubernetes.Interface,
	tektonClient tektonclient.Interface,
) error {
	if tektonClient == nil || k8sClient == nil {
		return fmt.Errorf("tekton preflight requires Kubernetes and Tekton clients")
	}

	var pipelineRun tektonv1.PipelineRun
	if err := yaml.Unmarshal([]byte(pipelineRunYAML), &pipelineRun); err != nil {
		return fmt.Errorf("invalid PipelineRun template: %w", err)
	}

	if pipelineRun.Spec.PipelineRef != nil && pipelineRun.Spec.PipelineRef.Name != "" {
		if _, err := tektonClient.TektonV1().Pipelines(namespace).Get(ctx, pipelineRun.Spec.PipelineRef.Name, metav1.GetOptions{}); err != nil {
			if apierrors.IsNotFound(err) {
				return fmt.Errorf("missing tekton pipeline: %s in namespace %s", pipelineRun.Spec.PipelineRef.Name, namespace)
			}
			return fmt.Errorf("failed to verify tekton pipeline %q in namespace %q: %w", pipelineRun.Spec.PipelineRef.Name, namespace, err)
		}
	}

	taskRefs := collectTaskRefs(&pipelineRun)
	if len(taskRefs) == 0 && pipelineRun.Spec.PipelineRef != nil && pipelineRun.Spec.PipelineRef.Name != "" {
		taskRefs = requiredTektonTaskNamesForPipelineRun(&pipelineRun)
	}
	for _, taskName := range taskRefs {
		candidates := compatibleTektonTaskNames(taskName)
		found := false
		var lastErr error
		for _, candidate := range candidates {
			_, err := tektonClient.TektonV1().Tasks(namespace).Get(ctx, candidate, metav1.GetOptions{})
			if err == nil {
				found = true
				break
			}
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to verify tekton task %q in namespace %q: %w", candidate, namespace, err)
			}
			lastErr = err
		}
		if found {
			continue
		}
		if len(candidates) > 1 {
			return fmt.Errorf("missing tekton task: %s (compatible: %s) in namespace %s", taskName, strings.Join(candidates, ","), namespace)
		}
		if apierrors.IsNotFound(lastErr) {
			return fmt.Errorf("missing tekton task: %s in namespace %s", taskName, namespace)
		}
		return fmt.Errorf("missing tekton task: %s in namespace %s", taskName, namespace)
	}

	for _, secretName := range collectRequiredSecrets(&pipelineRun) {
		if _, err := k8sClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{}); err != nil {
			if apierrors.IsNotFound(err) {
				return fmt.Errorf("missing required secret %q in namespace %q", secretName, namespace)
			}
			return fmt.Errorf("failed to verify secret %q in namespace %q: %w", secretName, namespace, err)
		}
	}

	return nil
}

func (e *MethodTektonExecutor) reconcileDockerConfigSecret(
	ctx context.Context,
	executionID uuid.UUID,
	namespace string,
	method BuildMethod,
	registryAuthID *uuid.UUID,
	targetImageRef string,
	k8sClient kubernetes.Interface,
) error {
	if !methodRequiresDockerConfigSecret(method) {
		return nil
	}
	if registryAuthID == nil {
		if shouldUseAnonymousInternalRegistry(targetImageRef) {
			if err := upsertDockerConfigSecret(ctx, k8sClient, namespace, []byte(`{"auths":{}}`)); err != nil {
				return fmt.Errorf("failed to create anonymous internal docker-config secret: %w", err)
			}
			e.service.AddLog(ctx, executionID, LogInfo, "Auto-configured anonymous docker-config secret for internal registry target", nil)
			return nil
		}
		return fmt.Errorf("registry_auth_id is required to materialize docker-config secret")
	}
	if e.registryAuth == nil {
		return fmt.Errorf("registry auth resolver is not configured")
	}

	dockerConfigJSON, err := e.registryAuth.ResolveDockerConfigJSON(ctx, *registryAuthID)
	if err != nil {
		return fmt.Errorf("failed to resolve docker config JSON for registry auth %s: %w", registryAuthID.String(), err)
	}
	if err := upsertDockerConfigSecret(ctx, k8sClient, namespace, dockerConfigJSON); err != nil {
		return err
	}

	e.service.AddLog(ctx, executionID, LogInfo, "Reconciled docker-config secret in namespace", nil)
	return nil
}

func shouldUseAnonymousInternalRegistry(targetImageRef string) bool {
	host, ok := registryHostFromImageRef(targetImageRef)
	if !ok {
		return false
	}
	for _, allowed := range internalRegistryHosts() {
		if strings.EqualFold(strings.TrimSpace(host), strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}

func internalRegistryHosts() []string {
	raw := strings.TrimSpace(os.Getenv("IF_INTERNAL_REGISTRY_HOSTS"))
	if raw == "" {
		return []string{"image-factory-registry:5000"}
	}
	parts := strings.Split(raw, ",")
	hosts := make([]string, 0, len(parts))
	for _, part := range parts {
		host := strings.TrimSpace(part)
		if host == "" {
			continue
		}
		hosts = append(hosts, host)
	}
	if len(hosts) == 0 {
		return []string{"image-factory-registry:5000"}
	}
	return hosts
}

func registryHostFromImageRef(ref string) (string, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", false
	}
	firstSlash := strings.IndexRune(ref, '/')
	if firstSlash <= 0 {
		return "", false
	}
	candidate := ref[:firstSlash]
	if strings.Contains(candidate, ".") || strings.Contains(candidate, ":") || strings.EqualFold(candidate, "localhost") {
		return candidate, true
	}
	return "", false
}

func (e *MethodTektonExecutor) reconcileGitAuthSecret(
	ctx context.Context,
	executionID uuid.UUID,
	namespace string,
	projectID uuid.UUID,
	k8sClient kubernetes.Interface,
) (bool, error) {
	if e.repositoryAuth == nil {
		return false, nil
	}
	secretData, err := e.repositoryAuth.ResolveGitAuthSecretData(ctx, projectID)
	if err != nil {
		return false, fmt.Errorf("failed to resolve git auth secret data for project %s: %w", projectID.String(), err)
	}
	if len(secretData) == 0 {
		return false, nil
	}
	if err := upsertOpaqueSecret(ctx, k8sClient, namespace, "git-auth", secretData, map[string]string{
		"app":  "image-factory",
		"kind": "repository-auth",
	}); err != nil {
		return false, err
	}
	e.service.AddLog(ctx, executionID, LogInfo, "Reconciled git-auth secret in namespace", nil)
	return true, nil
}

func methodRequiresDockerConfigSecret(method BuildMethod) bool {
	switch method {
	case BuildMethodDocker, BuildMethodBuildx, BuildMethodKaniko:
		return true
	default:
		return false
	}
}

func upsertDockerConfigSecret(ctx context.Context, k8sClient kubernetes.Interface, namespace string, dockerConfigJSON []byte) error {
	const secretName = "docker-config"
	const dockerConfigFileKey = "config.json"
	existing, err := k8sClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, createErr := k8sClient.CoreV1().Secrets(namespace).Create(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
					Labels: map[string]string{
						"app":  "image-factory",
						"kind": "registry-auth",
					},
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: dockerConfigJSON,
					dockerConfigFileKey:        dockerConfigJSON,
				},
			}, metav1.CreateOptions{})
			if createErr != nil {
				return fmt.Errorf("failed to create docker-config secret in namespace %s: %w", namespace, createErr)
			}
			return nil
		}
		return fmt.Errorf("failed to read docker-config secret in namespace %s: %w", namespace, err)
	}

	current := existing.Data[corev1.DockerConfigJsonKey]
	currentConfigJSON := existing.Data[dockerConfigFileKey]
	if existing.Type == corev1.SecretTypeDockerConfigJson &&
		bytes.Equal(current, dockerConfigJSON) &&
		bytes.Equal(currentConfigJSON, dockerConfigJSON) {
		return nil
	}

	existing.Type = corev1.SecretTypeDockerConfigJson
	if existing.Data == nil {
		existing.Data = make(map[string][]byte)
	}
	existing.Data[corev1.DockerConfigJsonKey] = dockerConfigJSON
	existing.Data[dockerConfigFileKey] = dockerConfigJSON
	if _, err := k8sClient.CoreV1().Secrets(namespace).Update(ctx, existing, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update docker-config secret in namespace %s: %w", namespace, err)
	}
	return nil
}

func upsertOpaqueSecret(
	ctx context.Context,
	k8sClient kubernetes.Interface,
	namespace, name string,
	data map[string][]byte,
	labels map[string]string,
) error {
	existing, err := k8sClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, createErr := k8sClient.CoreV1().Secrets(namespace).Create(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
					Labels:    labels,
				},
				Type: corev1.SecretTypeOpaque,
				Data: data,
			}, metav1.CreateOptions{})
			if createErr != nil {
				return fmt.Errorf("failed to create secret %s in namespace %s: %w", name, namespace, createErr)
			}
			return nil
		}
		return fmt.Errorf("failed to read secret %s in namespace %s: %w", name, namespace, err)
	}

	if existing.Type == corev1.SecretTypeOpaque && opaqueSecretDataEqual(existing.Data, data) {
		return nil
	}
	existing.Type = corev1.SecretTypeOpaque
	existing.Data = data
	if existing.Labels == nil {
		existing.Labels = map[string]string{}
	}
	for k, v := range labels {
		existing.Labels[k] = v
	}
	if _, err := k8sClient.CoreV1().Secrets(namespace).Update(ctx, existing, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update secret %s in namespace %s: %w", name, namespace, err)
	}
	return nil
}

func opaqueSecretDataEqual(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for key, aval := range a {
		bval, ok := b[key]
		if !ok || !bytes.Equal(aval, bval) {
			return false
		}
	}
	return true
}

func requiredTektonTaskNamesForPipelineRun(pipelineRun *tektonv1.PipelineRun) []string {
	if pipelineRun == nil {
		return []string{"git-clone"}
	}
	pipelineName := ""
	if pipelineRun.Spec.PipelineRef != nil {
		pipelineName = strings.TrimSpace(pipelineRun.Spec.PipelineRef.Name)
	}
	enableScan := pipelineRunParamBoolDefault(pipelineRun, "enable-scan", true)
	enableSBOM := pipelineRunParamBoolDefault(pipelineRun, "enable-sbom", true)
	enableSign := pipelineRunParamBoolDefault(pipelineRun, "enable-sign", false)

	switch pipelineName {
	case "image-factory-build-v1-docker":
		names := []string{"git-clone", "docker-build", "push-image"}
		if enableScan {
			names = append(names, "scan-image")
		}
		if enableSBOM {
			names = append(names, "generate-sbom")
		}
		if enableSign {
			names = append(names, "sign-image")
		}
		return names
	case "image-factory-build-v1-buildx":
		names := []string{"git-clone", "buildx", "push-image"}
		if enableScan {
			names = append(names, "scan-image")
		}
		if enableSBOM {
			names = append(names, "generate-sbom")
		}
		if enableSign {
			names = append(names, "sign-image")
		}
		return names
	case "image-factory-build-v1-kaniko":
		names := []string{"git-clone", "kaniko-no-push", "push-image"}
		if enableScan {
			names = append(names, "scan-image")
		}
		if enableSBOM {
			names = append(names, "generate-sbom")
		}
		if enableSign {
			names = append(names, "sign-image")
		}
		return names
	case "image-factory-build-v1-packer":
		return []string{"git-clone", "packer"}
	default:
		return []string{"git-clone"}
	}
}

func pipelineRunParamBoolDefault(pipelineRun *tektonv1.PipelineRun, name string, fallback bool) bool {
	if pipelineRun == nil {
		return fallback
	}
	target := strings.ToLower(strings.TrimSpace(name))
	for _, param := range pipelineRun.Spec.Params {
		if strings.ToLower(strings.TrimSpace(param.Name)) != target {
			continue
		}
		value := strings.ToLower(strings.TrimSpace(param.Value.StringVal))
		switch value {
		case "true", "1", "yes", "y", "on":
			return true
		case "false", "0", "no", "n", "off":
			return false
		default:
			return fallback
		}
	}
	return fallback
}

func compatibleTektonTaskNames(taskName string) []string {
	switch taskName {
	case "kaniko-no-push":
		return []string{"kaniko-no-push", "kaniko"}
	default:
		return []string{taskName}
	}
}

func collectTaskRefs(pipelineRun *tektonv1.PipelineRun) []string {
	if pipelineRun == nil || pipelineRun.Spec.PipelineSpec == nil {
		return nil
	}

	seen := map[string]struct{}{}
	refs := make([]string, 0, len(pipelineRun.Spec.PipelineSpec.Tasks))
	for _, task := range pipelineRun.Spec.PipelineSpec.Tasks {
		if task.TaskRef == nil || task.TaskRef.Name == "" {
			continue
		}
		name := task.TaskRef.Name
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		refs = append(refs, name)
	}
	return refs
}

func collectRequiredSecrets(pipelineRun *tektonv1.PipelineRun) []string {
	if pipelineRun == nil {
		return nil
	}

	seen := map[string]struct{}{}
	names := make([]string, 0, len(pipelineRun.Spec.Workspaces))
	for _, workspace := range pipelineRun.Spec.Workspaces {
		if workspace.Secret == nil || workspace.Secret.SecretName == "" {
			continue
		}
		secretName := workspace.Secret.SecretName
		if _, exists := seen[secretName]; exists {
			continue
		}
		seen[secretName] = struct{}{}
		names = append(names, secretName)
	}
	return names
}
