package build

import (
	"context"

	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

// TektonClients bundles clients and managers needed for Tekton execution.
type TektonClients struct {
	K8sClient    kubernetes.Interface
	TektonClient tektonclient.Interface
	NamespaceMgr NamespaceManager
	PipelineMgr  PipelineManager
}

// TektonClientProvider resolves clients for a build at execution time.
type TektonClientProvider interface {
	ClientsForBuild(ctx context.Context, build *Build) (*TektonClients, error)
}
