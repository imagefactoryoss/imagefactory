package tektoncompat

import (
	"context"
	"testing"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestDetectAPIVersionNilClient(t *testing.T) {
	_, err := DetectAPIVersion(context.Background(), nil, "default")
	if err == nil {
		t.Fatal("expected error for nil tekton client")
	}
}

func TestDetectAPIVersionDefaultsToV1(t *testing.T) {
	client := tektonfake.NewSimpleClientset()
	v, err := DetectAPIVersion(context.Background(), client, "default")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if v != APIVersionV1 {
		t.Fatalf("expected v1 fallback/preference, got %s", v)
	}
}

func TestClientUnknownVersionErrors(t *testing.T) {
	client := New(tektonfake.NewSimpleClientset(), APIVersion("unknown"))
	if err := client.GetTask(context.Background(), "default", "x"); err == nil {
		t.Fatal("expected unknown version error for GetTask")
	}
	if err := client.GetPipeline(context.Background(), "default", "x"); err == nil {
		t.Fatal("expected unknown version error for GetPipeline")
	}
	if err := client.ListPipelineRuns(context.Background(), "default", 1); err == nil {
		t.Fatal("expected unknown version error for ListPipelineRuns")
	}
	if _, err := client.GetPipelineRun(context.Background(), "default", "x"); err == nil {
		t.Fatal("expected unknown version error for GetPipelineRun")
	}
	if err := client.DeletePipelineRun(context.Background(), "default", "x"); err == nil {
		t.Fatal("expected unknown version error for DeletePipelineRun")
	}
	if _, err := client.GetTaskRun(context.Background(), "default", "x"); err == nil {
		t.Fatal("expected unknown version error for GetTaskRun")
	}
}

func TestCreatePipelineRunNil(t *testing.T) {
	client := New(tektonfake.NewSimpleClientset(), APIVersionV1)
	if _, err := client.CreatePipelineRun(context.Background(), "default", nil); err == nil {
		t.Fatal("expected error for nil pipelinerun")
	}
}

func TestCreateAndGetPipelineRunV1(t *testing.T) {
	raw := tektonfake.NewSimpleClientset()
	client := New(raw, APIVersionV1)

	created, err := client.CreatePipelineRun(context.Background(), "default", &tektonv1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{Name: "pr1"},
	})
	if err != nil {
		t.Fatalf("expected create success, got %v", err)
	}
	if created == nil || created.Name != "pr1" {
		t.Fatalf("unexpected created pipeline run: %+v", created)
	}

	got, err := client.GetPipelineRun(context.Background(), "default", "pr1")
	if err != nil {
		t.Fatalf("expected get success, got %v", err)
	}
	if got.Name != "pr1" {
		t.Fatalf("expected pr1, got %s", got.Name)
	}
}

func TestIsNotFoundHelper(t *testing.T) {
	err := apierrors.NewNotFound(schema.GroupResource{Group: "tekton.dev", Resource: "pipelineruns"}, "pr-x")
	if !IsNotFound(err) {
		t.Fatal("expected IsNotFound true")
	}
	if IsNotFound(nil) {
		t.Fatal("expected IsNotFound false for nil")
	}
}

func TestConvertNil(t *testing.T) {
	out, err := convert[tektonv1.PipelineRun, tektonv1.PipelineRun](nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil output for nil input, got %+v", out)
	}
}
