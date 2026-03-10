package infrastructure

import (
	"context"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestTriggerTrivyDBWarmupIfNeeded_TriggersManualJobWhenNeverScheduled(t *testing.T) {
	namespace := "image-factory-test"
	client := fake.NewSimpleClientset(
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      trivyDBWarmupCronJobName,
				Namespace: namespace,
			},
			Spec: batchv1.CronJobSpec{
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{},
					},
				},
			},
		},
	)
	service := &Service{}

	result, err := service.triggerTrivyDBWarmupIfNeeded(context.Background(), client, namespace)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "triggered" {
		t.Fatalf("expected status=triggered, got %v", result["status"])
	}
	jobs, err := client.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("failed listing jobs: %v", err)
	}
	if len(jobs.Items) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs.Items))
	}
}

func TestTriggerTrivyDBWarmupIfNeeded_SkipsWhenCronAlreadyScheduled(t *testing.T) {
	namespace := "image-factory-test"
	now := metav1.NewTime(time.Now().UTC())
	client := fake.NewSimpleClientset(
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      trivyDBWarmupCronJobName,
				Namespace: namespace,
			},
			Spec: batchv1.CronJobSpec{
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{},
					},
				},
			},
			Status: batchv1.CronJobStatus{
				LastScheduleTime: &now,
			},
		},
	)
	service := &Service{}

	result, err := service.triggerTrivyDBWarmupIfNeeded(context.Background(), client, namespace)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "skipped" || result["reason"] != "already_scheduled" {
		t.Fatalf("expected skipped/already_scheduled, got status=%v reason=%v", result["status"], result["reason"])
	}
	jobs, err := client.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("failed listing jobs: %v", err)
	}
	if len(jobs.Items) != 0 {
		t.Fatalf("expected 0 jobs, got %d", len(jobs.Items))
	}
}

func TestTriggerTrivyDBWarmupIfNeeded_SkipsWhenCronMissing(t *testing.T) {
	namespace := "image-factory-test"
	client := fake.NewSimpleClientset()
	service := &Service{}

	result, err := service.triggerTrivyDBWarmupIfNeeded(context.Background(), client, namespace)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "skipped" || result["reason"] != "cronjob_not_found" {
		t.Fatalf("expected skipped/cronjob_not_found, got status=%v reason=%v", result["status"], result["reason"])
	}
}

func TestTriggerTrivyDBWarmupIfNeeded_SkipsWhenManualWarmupJobAlreadyActive(t *testing.T) {
	namespace := "image-factory-test"
	client := fake.NewSimpleClientset(
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      trivyDBWarmupCronJobName,
				Namespace: namespace,
			},
			Spec: batchv1.CronJobSpec{
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{},
					},
				},
			},
		},
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "trivy-db-warmup-manual-abcde",
				Namespace: namespace,
				Labels: map[string]string{
					"imagefactory.io/warmup": "manual-bootstrap",
				},
			},
			Status: batchv1.JobStatus{
				Active: 1,
			},
		},
	)
	service := &Service{}

	result, err := service.triggerTrivyDBWarmupIfNeeded(context.Background(), client, namespace)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "skipped" || result["reason"] != "manual_job_active" {
		t.Fatalf("expected skipped/manual_job_active, got status=%v reason=%v", result["status"], result["reason"])
	}
	jobs, err := client.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("failed listing jobs: %v", err)
	}
	if len(jobs.Items) != 1 {
		t.Fatalf("expected no new manual job, got %d job(s)", len(jobs.Items))
	}
}
