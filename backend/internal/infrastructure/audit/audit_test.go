package audit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type captureRepo struct {
	savedEvent      *AuditEvent
	queryTenantID   *uuid.UUID
	queryFilter     AuditEventFilter
	queryLimit      int
	queryOffset     int
	countTenantID   *uuid.UUID
	countFilter     AuditEventFilter
}

func (r *captureRepo) SaveEvent(_ context.Context, event *AuditEvent) error {
	r.savedEvent = event
	return nil
}

func (r *captureRepo) QueryEvents(_ context.Context, tenantID *uuid.UUID, filter AuditEventFilter, limit, offset int) ([]*AuditEvent, error) {
	r.queryTenantID = tenantID
	r.queryFilter = filter
	r.queryLimit = limit
	r.queryOffset = offset
	return []*AuditEvent{}, nil
}

func (r *captureRepo) DeleteOldEvents(context.Context, uuid.UUID, time.Time) error {
	return nil
}

func (r *captureRepo) CountEvents(_ context.Context, tenantID *uuid.UUID, filter AuditEventFilter) (int, error) {
	r.countTenantID = tenantID
	r.countFilter = filter
	return 0, nil
}

func TestLogEventSetsDefaults(t *testing.T) {
	repo := &captureRepo{}
	svc := NewService(repo, zap.NewNop())

	event := &AuditEvent{
		EventType: AuditEventLoginSuccess,
		Severity:  AuditSeverityInfo,
		Resource:  "auth",
		Action:    "login",
		Message:   "ok",
	}

	if err := svc.LogEvent(context.Background(), event); err != nil {
		t.Fatalf("LogEvent returned error: %v", err)
	}

	if repo.savedEvent == nil {
		t.Fatal("expected event to be persisted")
	}
	if repo.savedEvent.ID == uuid.Nil {
		t.Fatal("expected event ID to be generated")
	}
	if repo.savedEvent.Timestamp.IsZero() {
		t.Fatal("expected timestamp to be set")
	}
}

func TestLogUserActionWithNilTenant(t *testing.T) {
	repo := &captureRepo{}
	svc := NewService(repo, zap.NewNop())

	userID := uuid.New()
	if err := svc.LogUserAction(
		context.Background(),
		uuid.Nil,
		userID,
		AuditEventUserUpdate,
		"user",
		"update",
		"updated",
		map[string]interface{}{"field": "email"},
	); err != nil {
		t.Fatalf("LogUserAction returned error: %v", err)
	}

	if repo.savedEvent == nil {
		t.Fatal("expected event to be saved")
	}
	if repo.savedEvent.TenantID != nil {
		t.Fatal("expected nil tenant for global user action")
	}
	if repo.savedEvent.UserID == nil || *repo.savedEvent.UserID != userID {
		t.Fatal("expected user_id to be set")
	}
}

func TestGetUserSessionsBuildsExpectedFilter(t *testing.T) {
	repo := &captureRepo{}
	svc := NewService(repo, zap.NewNop())
	userID := uuid.New()

	start := time.Now().UTC()
	_, err := svc.GetUserSessions(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserSessions returned error: %v", err)
	}

	if repo.queryFilter.UserID == nil || *repo.queryFilter.UserID != userID {
		t.Fatal("expected user_id filter")
	}
	if repo.queryFilter.EventType == nil || *repo.queryFilter.EventType != AuditEventLoginSuccess {
		t.Fatal("expected login_success event filter")
	}
	if repo.queryFilter.StartTime == nil {
		t.Fatal("expected start_time filter")
	}
	if repo.queryFilter.StartTime.After(start) {
		t.Fatal("expected start_time in the past")
	}
	if repo.queryLimit != 1000 || repo.queryOffset != 0 {
		t.Fatalf("unexpected pagination: limit=%d offset=%d", repo.queryLimit, repo.queryOffset)
	}
}
