package build

import (
	"context"
	"time"

	"github.com/google/uuid"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

// NamespaceManager handles tenant-specific Kubernetes namespaces
type NamespaceManager interface {
	// EnsureNamespace ensures a namespace exists for the tenant
	EnsureNamespace(ctx context.Context, tenantID uuid.UUID) (string, error)

	// DeleteNamespace deletes a tenant's namespace
	DeleteNamespace(ctx context.Context, tenantID uuid.UUID) error

	// GetNamespace returns the namespace name for a tenant
	GetNamespace(tenantID uuid.UUID) string
}

// PipelineManager handles Tekton PipelineRun operations
type PipelineManager interface {
	// CreatePipelineRun creates a new PipelineRun from YAML
	CreatePipelineRun(ctx context.Context, namespace, yamlContent string) (*tektonv1.PipelineRun, error)

	// GetPipelineRun gets a PipelineRun by name
	GetPipelineRun(ctx context.Context, namespace, name string) (*tektonv1.PipelineRun, error)

	// ListPipelineRuns lists PipelineRuns in a namespace
	ListPipelineRuns(ctx context.Context, namespace string, limit int) ([]*tektonv1.PipelineRun, error)

	// DeletePipelineRun deletes a PipelineRun
	DeletePipelineRun(ctx context.Context, namespace, name string) error

	// GetLogs retrieves logs from a PipelineRun
	GetLogs(ctx context.Context, namespace, pipelineRunName string) (map[string]string, error)
}

// TemplateEngine handles template rendering for pipeline definitions
type TemplateEngine interface {
	// Render renders a template with the given data
	Render(template string, data interface{}) (string, error)
}

// ExecutionResult represents the result of a build execution
type ExecutionResult struct {
	BuildID    uuid.UUID `json:"build_id"`
	ExecutorID string    `json:"executor_id"`
	Success    bool      `json:"success"`
	Message    string    `json:"message"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Artifacts  []Artifact `json:"artifacts,omitempty"`
}

// Artifact represents a build artifact
type Artifact struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	Path  string `json:"path,omitempty"`
}
