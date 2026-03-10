package build

import (
	"context"
	"errors"
	"testing"
	"time"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestResolveTaskRunPodName_SelectsRunningNewestPod(t *testing.T) {
	now := time.Now().UTC()
	client := k8sfake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "taskrun-pod-old-pending",
				Namespace:         "tenant-a",
				Labels:            map[string]string{"tekton.dev/taskRun": "tr-1"},
				CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Minute)),
			},
			Status: corev1.PodStatus{Phase: corev1.PodPending},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "taskrun-pod-new-running",
				Namespace:         "tenant-a",
				Labels:            map[string]string{"tekton.dev/taskRun": "tr-1"},
				CreationTimestamp: metav1.NewTime(now.Add(-30 * time.Second)),
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
	)

	got := resolveTaskRunPodName(context.Background(), client, "tenant-a", "tr-1", nil)
	if got != "taskrun-pod-new-running" {
		t.Fatalf("expected running pod fallback, got %q", got)
	}
}

func TestResolveTaskRunPodName_ReturnsEmptyWhenNoMatchingPod(t *testing.T) {
	client := k8sfake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other-taskrun-pod",
				Namespace: "tenant-a",
				Labels:    map[string]string{"tekton.dev/taskRun": "tr-other"},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
	)

	got := resolveTaskRunPodName(context.Background(), client, "tenant-a", "tr-1", nil)
	if got != "" {
		t.Fatalf("expected empty fallback when no matching pod, got %q", got)
	}
}

func TestIsTransientTektonLogStreamError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "pod initializing",
			err:  errors.New("container \"step-push\" in pod \"abc\" is waiting to start: PodInitializing"),
			want: true,
		},
		{
			name: "container creating",
			err:  errors.New("container is waiting to start: ContainerCreating"),
			want: true,
		},
		{
			name: "generic failure",
			err:  errors.New("forbidden"),
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isTransientTektonLogStreamError(tc.err)
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestTektonStepProgressState(t *testing.T) {
	waiting := tektonv1.StepState{
		ContainerState: corev1.ContainerState{
			Waiting: &corev1.ContainerStateWaiting{Reason: "PodInitializing", Message: "pulling"},
		},
	}
	state, reason, msg, level := tektonStepProgressState(waiting)
	if state != "waiting" || reason != "PodInitializing" || msg != "pulling" || level != LogInfo {
		t.Fatalf("unexpected waiting state: state=%q reason=%q msg=%q level=%q", state, reason, msg, level)
	}

	running := tektonv1.StepState{
		ContainerState: corev1.ContainerState{
			Running: &corev1.ContainerStateRunning{},
		},
	}
	state, reason, msg, level = tektonStepProgressState(running)
	if state != "running" || reason != "" || msg != "" || level != LogInfo {
		t.Fatalf("unexpected running state: state=%q reason=%q msg=%q level=%q", state, reason, msg, level)
	}

	failed := tektonv1.StepState{
		ContainerState: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{ExitCode: 2, Reason: "Error", Message: "boom"},
		},
	}
	state, reason, msg, level = tektonStepProgressState(failed)
	if state != "failed" || reason != "Error" || msg != "boom" || level != LogError {
		t.Fatalf("unexpected failed state: state=%q reason=%q msg=%q level=%q", state, reason, msg, level)
	}
}
