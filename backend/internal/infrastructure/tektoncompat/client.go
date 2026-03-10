package tektoncompat

import (
	"context"
	"encoding/json"
	"fmt"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type APIVersion string

const (
	APIVersionV1      APIVersion = "v1"
	APIVersionV1Beta1 APIVersion = "v1beta1"
)

// DetectAPIVersion attempts to detect which Tekton API group-version is served by the cluster.
// We prefer v1 and fall back to v1beta1 for older clusters.
func DetectAPIVersion(ctx context.Context, raw tektonclient.Interface, namespace string) (APIVersion, error) {
	if raw == nil {
		return "", fmt.Errorf("tekton client is required")
	}
	// Namespace is used for the probe; if unset, use a safe default.
	if namespace == "" {
		namespace = "default"
	}

	// Prefer discovery if available (typically allowed even with narrow RBAC), since LIST on
	// Tekton resources may be forbidden for runtime identities.
	if raw.Discovery() != nil {
		if _, err := raw.Discovery().ServerResourcesForGroupVersion("tekton.dev/v1"); err == nil {
			return APIVersionV1, nil
		}
		if _, err := raw.Discovery().ServerResourcesForGroupVersion("tekton.dev/v1beta1"); err == nil {
			return APIVersionV1Beta1, nil
		}
	}

	// Fall back to a GET probe: GET is the minimum verb our runtime checks/builds rely on.
	// Use a stable, expected object name; if it's missing we still learn the API exists.
	const probeTask = "git-clone"
	_, errV1 := raw.TektonV1().Tasks(namespace).Get(ctx, probeTask, metav1.GetOptions{})
	if errV1 == nil || errors.IsNotFound(errV1) {
		return APIVersionV1, nil
	}
	_, errBeta := raw.TektonV1beta1().Tasks(namespace).Get(ctx, probeTask, metav1.GetOptions{})
	if errBeta == nil || errors.IsNotFound(errBeta) {
		return APIVersionV1Beta1, nil
	}

	// If both probes are forbidden/unauthorized, pick v1 (modern default) so the caller can
	// continue and report a clearer RBAC error on the actual required GETs/CREATEs.
	if (errors.IsForbidden(errV1) || errors.IsUnauthorized(errV1)) && (errors.IsForbidden(errBeta) || errors.IsUnauthorized(errBeta)) {
		return APIVersionV1, nil
	}

	return "", fmt.Errorf(
		"no supported Tekton API detected (need tekton.dev/v1 or v1beta1). v1 probe error=%v; v1beta1 probe error=%v",
		errV1, errBeta,
	)
}

type Client struct {
	raw     tektonclient.Interface
	version APIVersion
}

func New(raw tektonclient.Interface, version APIVersion) *Client {
	return &Client{raw: raw, version: version}
}

func (c *Client) Version() APIVersion { return c.version }

func (c *Client) GetTask(ctx context.Context, namespace, name string) error {
	switch c.version {
	case APIVersionV1:
		_, err := c.raw.TektonV1().Tasks(namespace).Get(ctx, name, metav1.GetOptions{})
		return err
	case APIVersionV1Beta1:
		_, err := c.raw.TektonV1beta1().Tasks(namespace).Get(ctx, name, metav1.GetOptions{})
		return err
	default:
		return fmt.Errorf("unknown tekton api version: %s", c.version)
	}
}

func (c *Client) GetPipeline(ctx context.Context, namespace, name string) error {
	switch c.version {
	case APIVersionV1:
		_, err := c.raw.TektonV1().Pipelines(namespace).Get(ctx, name, metav1.GetOptions{})
		return err
	case APIVersionV1Beta1:
		_, err := c.raw.TektonV1beta1().Pipelines(namespace).Get(ctx, name, metav1.GetOptions{})
		return err
	default:
		return fmt.Errorf("unknown tekton api version: %s", c.version)
	}
}

func (c *Client) ListPipelineRuns(ctx context.Context, namespace string, limit int64) error {
	opts := metav1.ListOptions{Limit: limit}
	switch c.version {
	case APIVersionV1:
		_, err := c.raw.TektonV1().PipelineRuns(namespace).List(ctx, opts)
		return err
	case APIVersionV1Beta1:
		_, err := c.raw.TektonV1beta1().PipelineRuns(namespace).List(ctx, opts)
		return err
	default:
		return fmt.Errorf("unknown tekton api version: %s", c.version)
	}
}

func (c *Client) CreatePipelineRun(ctx context.Context, namespace string, pr *tektonv1.PipelineRun) (*tektonv1.PipelineRun, error) {
	if pr == nil {
		return nil, fmt.Errorf("pipelinerun is required")
	}
	switch c.version {
	case APIVersionV1:
		return c.raw.TektonV1().PipelineRuns(namespace).Create(ctx, pr, metav1.CreateOptions{})
	case APIVersionV1Beta1:
		beta, err := convert[tektonv1.PipelineRun, tektonv1beta1.PipelineRun](pr)
		if err != nil {
			return nil, err
		}
		created, err := c.raw.TektonV1beta1().PipelineRuns(namespace).Create(ctx, beta, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
		return convert[tektonv1beta1.PipelineRun, tektonv1.PipelineRun](created)
	default:
		return nil, fmt.Errorf("unknown tekton api version: %s", c.version)
	}
}

func (c *Client) GetPipelineRun(ctx context.Context, namespace, name string) (*tektonv1.PipelineRun, error) {
	switch c.version {
	case APIVersionV1:
		return c.raw.TektonV1().PipelineRuns(namespace).Get(ctx, name, metav1.GetOptions{})
	case APIVersionV1Beta1:
		got, err := c.raw.TektonV1beta1().PipelineRuns(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return convert[tektonv1beta1.PipelineRun, tektonv1.PipelineRun](got)
	default:
		return nil, fmt.Errorf("unknown tekton api version: %s", c.version)
	}
}

func (c *Client) DeletePipelineRun(ctx context.Context, namespace, name string) error {
	switch c.version {
	case APIVersionV1:
		return c.raw.TektonV1().PipelineRuns(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case APIVersionV1Beta1:
		return c.raw.TektonV1beta1().PipelineRuns(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	default:
		return fmt.Errorf("unknown tekton api version: %s", c.version)
	}
}

func (c *Client) GetTaskRun(ctx context.Context, namespace, name string) (*tektonv1.TaskRun, error) {
	switch c.version {
	case APIVersionV1:
		return c.raw.TektonV1().TaskRuns(namespace).Get(ctx, name, metav1.GetOptions{})
	case APIVersionV1Beta1:
		got, err := c.raw.TektonV1beta1().TaskRuns(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return convert[tektonv1beta1.TaskRun, tektonv1.TaskRun](got)
	default:
		return nil, fmt.Errorf("unknown tekton api version: %s", c.version)
	}
}

func IsNotFound(err error) bool { return errors.IsNotFound(err) }

func convert[From any, To any](in *From) (*To, error) {
	if in == nil {
		return nil, nil
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tekton object: %w", err)
	}
	var out To
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tekton object: %w", err)
	}
	return &out, nil
}
