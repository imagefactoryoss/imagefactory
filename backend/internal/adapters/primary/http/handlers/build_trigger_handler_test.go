package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/build"
	"go.uber.org/zap"
)

func TestTriggerToResponseIncludesScheduleFields(t *testing.T) {
	handler := &BuildTriggerHandler{logger: zap.NewNop()}

	last := time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC)
	next := time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC)
	trigger := &build.BuildTrigger{
		ID:            uuid.New(),
		BuildID:       uuid.New(),
		ProjectID:     uuid.New(),
		Type:          build.TriggerTypeSchedule,
		Name:          "nightly",
		Description:   "nightly build",
		CronExpr:      "0 0 * * *",
		Timezone:      "UTC",
		LastTriggered: &last,
		NextTrigger:   &next,
		IsActive:      true,
		CreatedBy:     uuid.New(),
		CreatedAt:     time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC),
	}

	resp := handler.triggerToResponse(trigger)
	if resp.TriggerType != string(build.TriggerTypeSchedule) {
		t.Fatalf("unexpected trigger type: %s", resp.TriggerType)
	}
	if resp.CronExpression == nil || *resp.CronExpression != "0 0 * * *" {
		t.Fatal("expected cron expression in response")
	}
	if resp.Timezone == nil || *resp.Timezone != "UTC" {
		t.Fatal("expected timezone in response")
	}
	if resp.LastTriggeredAt == nil || resp.NextTriggerAt == nil {
		t.Fatal("expected last/next trigger timestamps")
	}
}

func TestTriggerToResponseIncludesGitFields(t *testing.T) {
	handler := &BuildTriggerHandler{logger: zap.NewNop()}

	trigger := &build.BuildTrigger{
		ID:               uuid.New(),
		BuildID:          uuid.New(),
		ProjectID:        uuid.New(),
		Type:             build.TriggerTypeGitEvent,
		Name:             "git-hook",
		Description:      "repo events",
		GitProvider:      build.GitProviderGitHub,
		GitRepoURL:       "https://github.com/acme/repo",
		GitBranchPattern: "main",
		IsActive:         true,
		CreatedBy:        uuid.New(),
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	resp := handler.triggerToResponse(trigger)
	if resp.GitProvider == nil || *resp.GitProvider != "github" {
		t.Fatal("expected git provider in response")
	}
	if resp.GitRepoURL == nil || *resp.GitRepoURL == "" {
		t.Fatal("expected git repo URL in response")
	}
	if resp.GitBranchPattern == nil || *resp.GitBranchPattern != "main" {
		t.Fatal("expected git branch pattern in response")
	}
}
