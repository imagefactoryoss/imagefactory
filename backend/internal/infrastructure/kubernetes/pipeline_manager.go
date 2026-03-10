package kubernetes

import (
	"context"
	"fmt"
	"io"
	"strings"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

// KubernetesPipelineManager implements PipelineManager
type KubernetesPipelineManager struct {
	k8sClient    kubernetes.Interface
	tektonClient tektonclient.Interface
	logger       *zap.Logger
}

// NewKubernetesPipelineManager creates a new pipeline manager
func NewKubernetesPipelineManager(
	k8sClient kubernetes.Interface,
	tektonClient tektonclient.Interface,
	logger *zap.Logger,
) *KubernetesPipelineManager {
	return &KubernetesPipelineManager{
		k8sClient:    k8sClient,
		tektonClient: tektonClient,
		logger:       logger,
	}
}

// CreatePipelineRun creates a new PipelineRun from YAML
func (m *KubernetesPipelineManager) CreatePipelineRun(ctx context.Context, namespace, yamlContent string) (*tektonv1.PipelineRun, error) {
	// Parse YAML into PipelineRun
	var pipelineRun tektonv1.PipelineRun
	err := yaml.Unmarshal([]byte(yamlContent), &pipelineRun)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PipelineRun YAML: %w", err)
	}

	// Ensure namespace is set
	pipelineRun.Namespace = namespace

	// Create the PipelineRun
	created, err := m.tektonClient.TektonV1().PipelineRuns(namespace).Create(ctx, &pipelineRun, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create PipelineRun: %w", err)
	}

	m.logger.Info("Created PipelineRun",
		zap.String("name", created.Name),
		zap.String("namespace", namespace))

	return created, nil
}

// GetPipelineRun gets a PipelineRun by name
func (m *KubernetesPipelineManager) GetPipelineRun(ctx context.Context, namespace, name string) (*tektonv1.PipelineRun, error) {
	return m.tektonClient.TektonV1().PipelineRuns(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ListPipelineRuns lists PipelineRuns in a namespace
func (m *KubernetesPipelineManager) ListPipelineRuns(ctx context.Context, namespace string, limit int) ([]*tektonv1.PipelineRun, error) {
	listOptions := metav1.ListOptions{}
	if limit > 0 {
		listOptions.Limit = int64(limit)
	}

	list, err := m.tektonClient.TektonV1().PipelineRuns(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	result := make([]*tektonv1.PipelineRun, len(list.Items))
	for i := range list.Items {
		result[i] = &list.Items[i]
	}

	return result, nil
}

// DeletePipelineRun deletes a PipelineRun
func (m *KubernetesPipelineManager) DeletePipelineRun(ctx context.Context, namespace, name string) error {
	err := m.tektonClient.TektonV1().PipelineRuns(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete PipelineRun %s: %w", name, err)
	}

	m.logger.Info("Deleted PipelineRun", zap.String("name", name), zap.String("namespace", namespace))
	return nil
}

// GetLogs retrieves logs from a PipelineRun
func (m *KubernetesPipelineManager) GetLogs(ctx context.Context, namespace, pipelineRunName string) (map[string]string, error) {
	logs := make(map[string]string)

	// Tekton v1 no longer exposes a TaskRuns map on PipelineRun status. Resolve TaskRuns via label selector.
	taskRuns, err := m.tektonClient.TektonV1().TaskRuns(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/pipelineRun=%s", pipelineRunName),
	})
	if err != nil {
		return nil, err
	}

	for _, taskRun := range taskRuns.Items {
		taskRunName := taskRun.Name
		podName := strings.TrimSpace(taskRun.Status.PodName)
		if podName == "" {
			podName = fmt.Sprintf("%s-pod", taskRunName)
		}

		for _, step := range taskRun.Status.Steps {
			containerName := strings.TrimSpace(step.Container)
			if containerName == "" {
				containerName = strings.TrimSpace(step.Name)
			}
			if containerName == "" {
				continue
			}

			req := m.k8sClient.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
				Container:  containerName,
				Timestamps: true,
			})
			logStream, err := req.Stream(ctx)
			if err != nil {
				m.logger.Warn("Failed to get logs for step",
					zap.String("taskrun", taskRunName),
					zap.String("pod", podName),
					zap.String("container", containerName),
					zap.Error(err))
				continue
			}
			buf, readErr := io.ReadAll(logStream)
			_ = logStream.Close()
			if readErr != nil || len(buf) == 0 {
				continue
			}
			logs[fmt.Sprintf("%s/%s", taskRunName, containerName)] = string(buf)
		}
	}

	return logs, nil
}
