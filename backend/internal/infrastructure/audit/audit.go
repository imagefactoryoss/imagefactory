package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	AuditEventLoginSuccess      AuditEventType = "login_success"
	AuditEventLoginFailure      AuditEventType = "login_failure"
	AuditEventLogout            AuditEventType = "logout"
	AuditEventPasswordChange    AuditEventType = "password_change"
	AuditEventUserCreate        AuditEventType = "user_create"
	AuditEventUserUpdate        AuditEventType = "user_update"
	AuditEventUserDelete        AuditEventType = "user_delete"
	AuditEventTenantCreate      AuditEventType = "tenant_create"
	AuditEventTenantUpdate      AuditEventType = "tenant_update"
	AuditEventTenantActivate    AuditEventType = "tenant_activate"
	AuditEventTenantDelete      AuditEventType = "tenant_delete"
	AuditEventRoleAssign        AuditEventType = "role_assign"
	AuditEventRoleRemove        AuditEventType = "role_remove"
	AuditEventPermissionCheck   AuditEventType = "permission_check"
	AuditEventConfigChange      AuditEventType = "config_change"
	AuditEventBuildCreate       AuditEventType = "build_create"
	AuditEventBuildStart        AuditEventType = "build_start"
	AuditEventBuildCancel       AuditEventType = "build_cancel"
	AuditEventBuildComplete     AuditEventType = "build_complete"
	AuditEventBuildFail         AuditEventType = "build_fail"
	AuditEventServerStart       AuditEventType = "server_start"
	AuditEventServerRestart     AuditEventType = "server_restart"
	AuditEventPermissionDenied  AuditEventType = "permission_denied"
	AuditEventAPICall           AuditEventType = "api_call"
	AuditEventBulkImport        AuditEventType = "bulk_import"
	AuditEventUserInvite        AuditEventType = "user_invite"
	AuditEventPasswordReset     AuditEventType = "password_reset"
	AuditEventSSOLogin          AuditEventType = "sso_login"
	AuditEventSSOFailure        AuditEventType = "sso_failure"
	AuditEventMFAEnable         AuditEventType = "mfa_enable"
	AuditEventMFADisable        AuditEventType = "mfa_disable"
	AuditEventBulkOperation     AuditEventType = "bulk_operation"
	AuditEventSSOProviderCreate AuditEventType = "sso_provider_create"
	AuditEventSSOProviderUpdate AuditEventType = "sso_provider_update"
	AuditEventSSOProviderDelete AuditEventType = "sso_provider_delete"
	AuditEventUserInviteCreate  AuditEventType = "user_invite_create"
	AuditEventUserInviteAccept  AuditEventType = "user_invite_accept"
	AuditEventUserInviteCancel  AuditEventType = "user_invite_cancel"
	AuditEventUserInviteResend  AuditEventType = "user_invite_resend"
	AuditEventProjectCreate     AuditEventType = "project_create"
	AuditEventProjectUpdate     AuditEventType = "project_update"
	AuditEventProjectDelete     AuditEventType = "project_delete"
	AuditEventProjectArchive    AuditEventType = "project_archive"
	AuditEventProjectActivate   AuditEventType = "project_activate"
	AuditEventProjectPurge      AuditEventType = "project_purge"
	AuditEventCapabilityDenied  AuditEventType = "capability_denied"
	AuditEventSORDenied         AuditEventType = "sor_denied"
	AuditEventMemberAdded       AuditEventType = "member_added"
	AuditEventMemberRemoved     AuditEventType = "member_removed"
	AuditEventMemberUpdated     AuditEventType = "member_updated"
	AuditEventGroupCreate       AuditEventType = "group_create"
	AuditEventGroupUpdate       AuditEventType = "group_update"
	AuditEventGroupDelete       AuditEventType = "group_delete"
	AuditEventGroupMemberAdd    AuditEventType = "group_member_add"
	AuditEventGroupMemberRemove AuditEventType = "group_member_remove"
	AuditEventProfileUpdate     AuditEventType = "profile_update"
)

// AuditEventSeverity represents the severity level of an audit event
type AuditEventSeverity string

const (
	AuditSeverityInfo     AuditEventSeverity = "info"
	AuditSeverityWarning  AuditEventSeverity = "warning"
	AuditSeverityError    AuditEventSeverity = "error"
	AuditSeverityCritical AuditEventSeverity = "critical"
)

// AuditEvent represents an audit log entry
type AuditEvent struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	TenantID  *uuid.UUID             `json:"tenant_id,omitempty" db:"tenant_id"`
	UserID    *uuid.UUID             `json:"user_id,omitempty" db:"user_id"`
	UserName  string                 `json:"user_name,omitempty" db:"user_name"`
	EventType AuditEventType         `json:"event_type" db:"event_type"`
	Severity  AuditEventSeverity     `json:"severity" db:"severity"`
	Resource  string                 `json:"resource" db:"resource"`
	Action    string                 `json:"action" db:"action"`
	IPAddress string                 `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent string                 `json:"user_agent,omitempty" db:"user_agent"`
	Details   map[string]interface{} `json:"details,omitempty" db:"details"`
	Message   string                 `json:"message" db:"message"`
	Timestamp time.Time              `json:"timestamp" db:"timestamp"`
}

// AuditService defines the interface for audit logging
type AuditService interface {
	LogEvent(ctx context.Context, event *AuditEvent) error
	LogUserAction(ctx context.Context, tenantID, userID uuid.UUID, eventType AuditEventType, resource, action, message string, details map[string]interface{}) error
	LogSystemAction(ctx context.Context, tenantID uuid.UUID, eventType AuditEventType, resource, action, message string, details map[string]interface{}) error
	QueryEvents(ctx context.Context, tenantID *uuid.UUID, filter AuditEventFilter, limit, offset int) ([]*AuditEvent, error)
	CountEvents(ctx context.Context, tenantID *uuid.UUID, filter AuditEventFilter) (int, error)
}

// AuditEventFilter represents filters for querying audit events
type AuditEventFilter struct {
	UserID    *uuid.UUID          `json:"user_id,omitempty"`
	EventType *AuditEventType     `json:"event_type,omitempty"`
	Severity  *AuditEventSeverity `json:"severity,omitempty"`
	Resource  *string             `json:"resource,omitempty"`
	Action    *string             `json:"action,omitempty"`
	Search    *string             `json:"search,omitempty"`
	StartTime *time.Time          `json:"start_time,omitempty"`
	EndTime   *time.Time          `json:"end_time,omitempty"`
}

// Service implements the AuditService interface
type Service struct {
	repo   Repository
	logger *zap.Logger
}

// NewService creates a new audit service
func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// LogEvent logs an audit event
func (s *Service) LogEvent(ctx context.Context, event *AuditEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Log to structured logger as well
	tenantIDStr := "null"
	if event.TenantID != nil {
		tenantIDStr = event.TenantID.String()
	}
	s.logger.Info("Audit event",
		zap.String("event_id", event.ID.String()),
		zap.String("tenant_id", tenantIDStr),
		zap.String("event_type", string(event.EventType)),
		zap.String("severity", string(event.Severity)),
		zap.String("resource", event.Resource),
		zap.String("action", event.Action),
		zap.String("message", event.Message),
		zap.Any("details", event.Details),
	)

	return s.repo.SaveEvent(ctx, event)
}

// LogUserAction logs a user-initiated action
func (s *Service) LogUserAction(ctx context.Context, tenantID, userID uuid.UUID, eventType AuditEventType, resource, action, message string, details map[string]interface{}) error {
	var tenantIDPtr *uuid.UUID
	if tenantID != uuid.Nil {
		tenantIDPtr = &tenantID
	}
	event := &AuditEvent{
		TenantID:  tenantIDPtr,
		UserID:    &userID,
		EventType: eventType,
		Severity:  AuditSeverityInfo,
		Resource:  resource,
		Action:    action,
		Message:   message,
		Details:   details,
	}

	return s.LogEvent(ctx, event)
}

// LogSystemAction logs a system-initiated action
func (s *Service) LogSystemAction(ctx context.Context, tenantID uuid.UUID, eventType AuditEventType, resource, action, message string, details map[string]interface{}) error {
	var tenantIDPtr *uuid.UUID
	if tenantID != uuid.Nil {
		tenantIDPtr = &tenantID
	}
	event := &AuditEvent{
		TenantID:  tenantIDPtr,
		EventType: eventType,
		Severity:  AuditSeverityInfo,
		Resource:  resource,
		Action:    action,
		Message:   message,
		Details:   details,
	}

	return s.LogEvent(ctx, event)
}

// LogGlobalSystemAction logs a system-initiated action that is not tenant-specific
func (s *Service) LogGlobalSystemAction(ctx context.Context, eventType AuditEventType, resource, action, message string, details map[string]interface{}) error {
	event := &AuditEvent{
		TenantID:  nil, // System events are not tenant-specific
		EventType: eventType,
		Severity:  AuditSeverityInfo,
		Resource:  resource,
		Action:    action,
		Message:   message,
		Details:   details,
	}

	return s.LogEvent(ctx, event)
}

// QueryEvents queries audit events with filtering
func (s *Service) QueryEvents(ctx context.Context, tenantID *uuid.UUID, filter AuditEventFilter, limit, offset int) ([]*AuditEvent, error) {
	return s.repo.QueryEvents(ctx, tenantID, filter, limit, offset)
}

// CountEvents counts audit events matching the filter
func (s *Service) CountEvents(ctx context.Context, tenantID *uuid.UUID, filter AuditEventFilter) (int, error) {
	return s.repo.CountEvents(ctx, tenantID, filter)
}

// GetUserActivity retrieves audit events for a specific user
func (s *Service) GetUserActivity(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*AuditEvent, error) {
	filter := AuditEventFilter{
		UserID: &userID,
	}
	return s.repo.QueryEvents(ctx, nil, filter, limit, offset)
}

// GetLoginHistory retrieves login events for a specific user
func (s *Service) GetLoginHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*AuditEvent, error) {
	eventType := AuditEventLoginSuccess
	filter := AuditEventFilter{
		UserID:    &userID,
		EventType: &eventType,
	}
	return s.repo.QueryEvents(ctx, nil, filter, limit, offset)
}

// GetUserSessions retrieves active sessions for a user
func (s *Service) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*AuditEvent, error) {
	eventType := AuditEventLoginSuccess
	startTime := time.Now().Add(-24 * time.Hour) // Last 24 hours
	filter := AuditEventFilter{
		UserID:    &userID,
		EventType: &eventType,
		StartTime: &startTime,
	}
	return s.repo.QueryEvents(ctx, nil, filter, 1000, 0)
}
