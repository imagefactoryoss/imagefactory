package kubernetes

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// KubernetesNamespaceManager implements NamespaceManager.
type KubernetesNamespaceManager struct {
	k8sClient kubernetes.Interface
	logger    *zap.Logger
}

// NewKubernetesNamespaceManager creates a new namespace manager.
func NewKubernetesNamespaceManager(k8sClient kubernetes.Interface, logger *zap.Logger) *KubernetesNamespaceManager {
	return &KubernetesNamespaceManager{
		k8sClient: k8sClient,
		logger:    logger,
	}
}

// EnsureNamespace ensures a namespace exists for the tenant.
func (m *KubernetesNamespaceManager) EnsureNamespace(ctx context.Context, tenantID uuid.UUID) (string, error) {
	namespaceName := m.GetNamespace(tenantID)

	// Check if namespace exists
	_, err := m.k8sClient.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err == nil {
		return namespaceName, nil
	}

	// Create namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				"tenant-id": tenantID.String(),
				"app":       "image-factory",
			},
		},
	}

	_, err = m.k8sClient.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create namespace %s: %w", namespaceName, err)
	}

	m.logger.Info("Created namespace", zap.String("namespace", namespaceName), zap.String("tenant_id", tenantID.String()))
	return namespaceName, nil
}

// DeleteNamespace deletes a tenant's namespace.
func (m *KubernetesNamespaceManager) DeleteNamespace(ctx context.Context, tenantID uuid.UUID) error {
	namespaceName := m.GetNamespace(tenantID)

	err := m.k8sClient.CoreV1().Namespaces().Delete(ctx, namespaceName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete namespace %s: %w", namespaceName, err)
	}

	m.logger.Info("Deleted namespace", zap.String("namespace", namespaceName))
	return nil
}

// GetNamespace returns the namespace name for a tenant.
func (m *KubernetesNamespaceManager) GetNamespace(tenantID uuid.UUID) string {
	return fmt.Sprintf("image-factory-%s", tenantID.String()[:8])
}
