package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/imageimport"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	domainworkflow "github.com/srikarm/image-factory/internal/domain/workflow"
	"github.com/srikarm/image-factory/internal/infrastructure/denialtelemetry"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"github.com/srikarm/image-factory/internal/infrastructure/releasetelemetry"
)

type imageImportHandlerTestRepository struct {
	createCalled bool
	byID         map[uuid.UUID]*imageimport.ImportRequest
}

func (r *imageImportHandlerTestRepository) Create(ctx context.Context, req *imageimport.ImportRequest) error {
	r.createCalled = true
	return nil
}

func (r *imageImportHandlerTestRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*imageimport.ImportRequest, error) {
	if r.byID != nil {
		if item, ok := r.byID[id]; ok && item.TenantID == tenantID {
			return item, nil
		}
	}
	return nil, imageimport.ErrImportNotFound
}

func (r *imageImportHandlerTestRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, requestType imageimport.RequestType, limit, offset int) ([]*imageimport.ImportRequest, error) {
	if r.byID == nil {
		return []*imageimport.ImportRequest{}, nil
	}
	rows := make([]*imageimport.ImportRequest, 0, len(r.byID))
	for _, item := range r.byID {
		if item != nil && item.TenantID == tenantID {
			if requestType != "" && item.RequestType != requestType {
				continue
			}
			rows = append(rows, item)
		}
	}
	return rows, nil
}

func (r *imageImportHandlerTestRepository) ListAll(ctx context.Context, requestType imageimport.RequestType, limit, offset int) ([]*imageimport.ImportRequest, error) {
	if r.byID == nil {
		return []*imageimport.ImportRequest{}, nil
	}
	rows := make([]*imageimport.ImportRequest, 0, len(r.byID))
	for _, item := range r.byID {
		if item == nil {
			continue
		}
		if requestType != "" && item.RequestType != requestType {
			continue
		}
		rows = append(rows, item)
	}
	return rows, nil
}

func (r *imageImportHandlerTestRepository) ListReleasedByTenant(ctx context.Context, tenantID uuid.UUID, search string, limit, offset int) ([]*imageimport.ReleasedArtifact, int, error) {
	if r.byID == nil {
		return []*imageimport.ReleasedArtifact{}, 0, nil
	}
	search = strings.ToLower(strings.TrimSpace(search))
	rows := make([]*imageimport.ReleasedArtifact, 0, len(r.byID))
	for _, item := range r.byID {
		if item == nil || item.TenantID != tenantID {
			continue
		}
		projection := imageimport.ResolveReleaseProjection(item)
		if projection.State != imageimport.ReleaseStateReleased {
			continue
		}
		if search != "" {
			haystack := strings.ToLower(strings.TrimSpace(item.SourceImageRef + " " + item.InternalImageRef + " " + item.SourceRegistry + " " + item.SourceImageDigest))
			if !strings.Contains(haystack, search) {
				continue
			}
		}
		rows = append(rows, &imageimport.ReleasedArtifact{
			ID:                 item.ID,
			TenantID:           item.TenantID,
			RequestedByUserID:  item.RequestedByUserID,
			SORRecordID:        item.SORRecordID,
			SourceRegistry:     item.SourceRegistry,
			SourceImageRef:     item.SourceImageRef,
			InternalImageRef:   item.InternalImageRef,
			SourceImageDigest:  item.SourceImageDigest,
			PolicyDecision:     item.PolicyDecision,
			PolicySnapshotJSON: item.PolicySnapshotJSON,
			ReleaseState:       projection.State,
			ReleaseReason:      item.ReleaseReason,
			ReleaseActorUserID: item.ReleaseActorUserID,
			ReleaseRequestedAt: item.ReleaseRequestedAt,
			ReleasedAt:         item.ReleasedAt,
			CreatedAt:          item.CreatedAt,
			UpdatedAt:          item.UpdatedAt,
		})
	}
	total := len(rows)
	return rows, total, nil
}

func (r *imageImportHandlerTestRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status imageimport.Status, errorMessage, internalImageRef string) error {
	return nil
}

func (r *imageImportHandlerTestRepository) UpdatePipelineRefs(ctx context.Context, tenantID, id uuid.UUID, pipelineRunName, pipelineNamespace string) error {
	return nil
}

func (r *imageImportHandlerTestRepository) UpdateEvidence(ctx context.Context, tenantID, id uuid.UUID, evidence imageimport.ImportEvidence) error {
	return nil
}

func (r *imageImportHandlerTestRepository) UpdateReleaseState(ctx context.Context, tenantID, id uuid.UUID, state imageimport.ReleaseState, blockerReason string, actorUserID *uuid.UUID, reason string, requestedAt, releasedAt *time.Time) error {
	if r.byID != nil {
		if item, ok := r.byID[id]; ok && item.TenantID == tenantID {
			item.ReleaseState = state
			item.ReleaseBlockerReason = blockerReason
			item.ReleaseActorUserID = actorUserID
			item.ReleaseReason = reason
			item.ReleaseRequestedAt = requestedAt
			item.ReleasedAt = releasedAt
			return nil
		}
	}
	return imageimport.ErrImportNotFound
}

func (r *imageImportHandlerTestRepository) SyncEvidenceToCatalog(ctx context.Context, tenantID, id uuid.UUID) error {
	return nil
}

type imageImportHandlerTestSORValidator struct {
	ok bool
}

func (v *imageImportHandlerTestSORValidator) ValidateRegistration(ctx context.Context, tenantID uuid.UUID, sorRecordID string) (bool, error) {
	return v.ok, nil
}

type imageImportHandlerTestCapabilityChecker struct {
	entitled        bool
	releaseEntitled bool
}

func (c *imageImportHandlerTestCapabilityChecker) IsImportEntitled(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	return c.entitled, nil
}

func (c *imageImportHandlerTestCapabilityChecker) IsQuarantineReleaseEntitled(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	return c.releaseEntitled, nil
}

type imageImportWorkflowRepoStub struct {
	instance   *domainworkflow.Instance
	steps      []domainworkflow.Step
	updateStep *domainworkflow.Step
}

func (s *imageImportWorkflowRepoStub) ClaimNextRunnableStep(ctx context.Context) (*domainworkflow.Step, error) {
	return nil, nil
}
func (s *imageImportWorkflowRepoStub) UpdateStep(ctx context.Context, step *domainworkflow.Step) error {
	cloned := *step
	s.updateStep = &cloned
	return nil
}
func (s *imageImportWorkflowRepoStub) AppendEvent(ctx context.Context, event *domainworkflow.Event) error {
	return nil
}
func (s *imageImportWorkflowRepoStub) UpsertDefinition(ctx context.Context, name string, version int, definition map[string]interface{}) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (s *imageImportWorkflowRepoStub) CreateInstance(ctx context.Context, definitionID uuid.UUID, tenantID *uuid.UUID, subjectType string, subjectID uuid.UUID, status domainworkflow.InstanceStatus) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (s *imageImportWorkflowRepoStub) CreateSteps(ctx context.Context, instanceID uuid.UUID, steps []domainworkflow.StepDefinition) error {
	return nil
}
func (s *imageImportWorkflowRepoStub) UpdateInstanceStatus(ctx context.Context, instanceID uuid.UUID, status domainworkflow.InstanceStatus) error {
	return nil
}
func (s *imageImportWorkflowRepoStub) UpdateStepStatus(ctx context.Context, instanceID uuid.UUID, stepKey string, status domainworkflow.StepStatus, errMsg *string) error {
	return nil
}
func (s *imageImportWorkflowRepoStub) GetInstanceWithStepsBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) (*domainworkflow.Instance, []domainworkflow.Step, error) {
	return s.instance, s.steps, nil
}
func (s *imageImportWorkflowRepoStub) GetBlockedStepDiagnostics(ctx context.Context, subjectType string) (*domainworkflow.BlockedStepDiagnostics, error) {
	return nil, nil
}

type imageImportNotificationRepoStub struct {
	adminUserIDs  []uuid.UUID
	receiptCounts map[string]int
	inAppCounts   map[string]int
}

type imageImportEventBusStub struct {
	events []messaging.Event
}

func (s *imageImportEventBusStub) Publish(ctx context.Context, event messaging.Event) error {
	s.events = append(s.events, event)
	return nil
}

func (s *imageImportEventBusStub) Subscribe(eventType string, handler messaging.Handler) (unsubscribe func()) {
	return func() {}
}

func (s *imageImportNotificationRepoStub) ListTenantAdminUserIDs(ctx context.Context, tenantID uuid.UUID) ([]uuid.UUID, error) {
	return s.adminUserIDs, nil
}

func (s *imageImportNotificationRepoStub) CountImageImportNotificationReceipts(ctx context.Context, tenantID uuid.UUID, eventType, idempotencyKey string) (int, error) {
	if s.receiptCounts == nil {
		return 0, nil
	}
	return s.receiptCounts[eventType+"|"+idempotencyKey], nil
}

func (s *imageImportNotificationRepoStub) CountImageImportInAppNotifications(ctx context.Context, tenantID, importID uuid.UUID, notificationType string) (int, error) {
	if s.inAppCounts == nil {
		return 0, nil
	}
	return s.inAppCounts[importID.String()+"|"+notificationType], nil
}

type imageImportHandlerTestSystemConfigRepo struct {
	tenantID       uuid.UUID
	tenantConfig   *systemconfig.SystemConfig
	globalConfig   *systemconfig.SystemConfig
	releasePolicy  *systemconfig.SystemConfig
	defaultCfgByID map[uuid.UUID]*systemconfig.SystemConfig
}

func (r *imageImportHandlerTestSystemConfigRepo) Save(ctx context.Context, config *systemconfig.SystemConfig) error {
	return nil
}
func (r *imageImportHandlerTestSystemConfigRepo) SaveAll(ctx context.Context, configs []*systemconfig.SystemConfig) error {
	return nil
}
func (r *imageImportHandlerTestSystemConfigRepo) FindByID(ctx context.Context, id uuid.UUID) (*systemconfig.SystemConfig, error) {
	if cfg, ok := r.defaultCfgByID[id]; ok {
		return cfg, nil
	}
	return nil, systemconfig.ErrConfigNotFound
}
func (r *imageImportHandlerTestSystemConfigRepo) FindByKey(ctx context.Context, tenantID *uuid.UUID, configKey string) (*systemconfig.SystemConfig, error) {
	switch configKey {
	case "sor_registration":
		if tenantID != nil && *tenantID == r.tenantID && r.tenantConfig != nil {
			return r.tenantConfig, nil
		}
		if tenantID == nil && r.globalConfig != nil {
			return r.globalConfig, nil
		}
	case "release_governance_policy":
		if r.releasePolicy != nil {
			return r.releasePolicy, nil
		}
	default:
		return nil, systemconfig.ErrConfigNotFound
	}
	return nil, systemconfig.ErrConfigNotFound
}
func (r *imageImportHandlerTestSystemConfigRepo) FindByTypeAndKey(ctx context.Context, tenantID *uuid.UUID, configType systemconfig.ConfigType, configKey string) (*systemconfig.SystemConfig, error) {
	return nil, systemconfig.ErrConfigNotFound
}
func (r *imageImportHandlerTestSystemConfigRepo) FindByType(ctx context.Context, tenantID *uuid.UUID, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	return []*systemconfig.SystemConfig{}, nil
}
func (r *imageImportHandlerTestSystemConfigRepo) FindAllByType(ctx context.Context, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	return []*systemconfig.SystemConfig{}, nil
}
func (r *imageImportHandlerTestSystemConfigRepo) FindUniversalByType(ctx context.Context, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	return []*systemconfig.SystemConfig{}, nil
}
func (r *imageImportHandlerTestSystemConfigRepo) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*systemconfig.SystemConfig, error) {
	return []*systemconfig.SystemConfig{}, nil
}
func (r *imageImportHandlerTestSystemConfigRepo) FindAll(ctx context.Context) ([]*systemconfig.SystemConfig, error) {
	return []*systemconfig.SystemConfig{}, nil
}
func (r *imageImportHandlerTestSystemConfigRepo) FindActiveByType(ctx context.Context, tenantID uuid.UUID, configType systemconfig.ConfigType) ([]*systemconfig.SystemConfig, error) {
	return []*systemconfig.SystemConfig{}, nil
}
func (r *imageImportHandlerTestSystemConfigRepo) Update(ctx context.Context, config *systemconfig.SystemConfig) error {
	return nil
}
func (r *imageImportHandlerTestSystemConfigRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (r *imageImportHandlerTestSystemConfigRepo) ExistsByKey(ctx context.Context, tenantID *uuid.UUID, configKey string) (bool, error) {
	return false, nil
}
func (r *imageImportHandlerTestSystemConfigRepo) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return 0, nil
}
func (r *imageImportHandlerTestSystemConfigRepo) CountByType(ctx context.Context, tenantID uuid.UUID, configType systemconfig.ConfigType) (int, error) {
	return 0, nil
}

func TestImageImportHandlerCreateImportRequest_Unauthorized(t *testing.T) {
	repo := &imageImportHandlerTestRepository{}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())

	reqBody := map[string]string{
		"epr_record_id":    "APP-123",
		"source_registry":  "ghcr.io",
		"source_image_ref": "ghcr.io/org/app:1.0.0",
	}
	payload, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler.CreateImportRequest(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestImageImportHandlerCreateImportRequest_SORDenied(t *testing.T) {
	repo := &imageImportHandlerTestRepository{}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: false}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())

	reqBody := map[string]string{
		"epr_record_id":    "APP-123",
		"source_registry":  "ghcr.io",
		"source_image_ref": "ghcr.io/org/app:1.0.0",
	}
	payload, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests", bytes.NewReader(payload))
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
	}))
	w := httptest.NewRecorder()

	handler.CreateImportRequest(w, req)
	if w.Code != http.StatusPreconditionFailed {
		t.Fatalf("expected %d, got %d", http.StatusPreconditionFailed, w.Code)
	}
	if repo.createCalled {
		t.Fatalf("expected create not to be called when SOR is denied")
	}
}

func TestImageImportHandlerCreateImportRequest_CapabilityDenied(t *testing.T) {
	repo := &imageImportHandlerTestRepository{}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: false}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())

	reqBody := map[string]string{
		"epr_record_id":    "APP-123",
		"source_registry":  "ghcr.io",
		"source_image_ref": "ghcr.io/org/app:1.0.0",
	}
	payload, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests", bytes.NewReader(payload))
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
	}))
	w := httptest.NewRecorder()

	handler.CreateImportRequest(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, w.Code)
	}
	if repo.createCalled {
		t.Fatalf("expected create not to be called when capability is denied")
	}
}

func TestImageImportHandlerRetryImportRequest_CapabilityDenied(t *testing.T) {
	tenantID := uuid.New()
	existing := &imageimport.ImportRequest{
		ID:                uuid.New(),
		TenantID:          tenantID,
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-123",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/org/app:1.0.0",
		Status:            imageimport.StatusFailed,
	}
	repo := &imageImportHandlerTestRepository{byID: map[uuid.UUID]*imageimport.ImportRequest{existing.ID: existing}}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: false}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests/"+existing.ID.String()+"/retry", nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", existing.ID.String())
	w := httptest.NewRecorder()

	handler.RetryImportRequest(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, w.Code)
	}
	if repo.createCalled {
		t.Fatalf("expected create not to be called when retry capability is denied")
	}
}

func TestImageImportHandlerRetryImportRequest_SORDenied(t *testing.T) {
	tenantID := uuid.New()
	existing := &imageimport.ImportRequest{
		ID:                uuid.New(),
		TenantID:          tenantID,
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-123",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/org/app:1.0.0",
		Status:            imageimport.StatusFailed,
	}
	repo := &imageImportHandlerTestRepository{byID: map[uuid.UUID]*imageimport.ImportRequest{existing.ID: existing}}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: false}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests/"+existing.ID.String()+"/retry", nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", existing.ID.String())
	w := httptest.NewRecorder()

	handler.RetryImportRequest(w, req)
	if w.Code != http.StatusPreconditionFailed {
		t.Fatalf("expected %d, got %d", http.StatusPreconditionFailed, w.Code)
	}
	if repo.createCalled {
		t.Fatalf("expected create not to be called when retry SOR is denied")
	}
}

func TestImageImportHandlerRetryImportRequest_BackoffActive(t *testing.T) {
	tenantID := uuid.New()
	existing := &imageimport.ImportRequest{
		ID:                uuid.New(),
		TenantID:          tenantID,
		RequestedByUserID: uuid.New(),
		RequestType:       imageimport.RequestTypeQuarantine,
		SORRecordID:       "APP-123",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/org/app:1.0.0",
		Status:            imageimport.StatusFailed,
		ErrorMessage:      "dispatch_failed: context deadline exceeded",
		UpdatedAt:         time.Now().UTC(),
	}
	repo := &imageImportHandlerTestRepository{byID: map[uuid.UUID]*imageimport.ImportRequest{existing.ID: existing}}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests/"+existing.ID.String()+"/retry", nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", existing.ID.String())
	w := httptest.NewRecorder()

	handler.RetryImportRequest(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected %d, got %d body=%s", http.StatusTooManyRequests, w.Code, w.Body.String())
	}
	if repo.createCalled {
		t.Fatalf("expected create not to be called while retry backoff is active")
	}
}

func TestImageImportHandlerRetryImportRequest_AttemptLimitReached(t *testing.T) {
	tenantID := uuid.New()
	byID := map[uuid.UUID]*imageimport.ImportRequest{}
	now := time.Now().UTC().Add(-2 * time.Hour)
	var targetID uuid.UUID
	for i := 0; i < 5; i++ {
		id := uuid.New()
		if i == 0 {
			targetID = id
		}
		byID[id] = &imageimport.ImportRequest{
			ID:                id,
			TenantID:          tenantID,
			RequestedByUserID: uuid.New(),
			RequestType:       imageimport.RequestTypeQuarantine,
			SORRecordID:       "APP-123",
			SourceRegistry:    "ghcr.io",
			SourceImageRef:    "ghcr.io/org/app:1.0.0",
			Status:            imageimport.StatusFailed,
			ErrorMessage:      "dispatch_failed: context deadline exceeded",
			UpdatedAt:         now,
		}
	}
	repo := &imageImportHandlerTestRepository{byID: byID}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests/"+targetID.String()+"/retry", nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", targetID.String())
	w := httptest.NewRecorder()

	handler.RetryImportRequest(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected %d, got %d body=%s", http.StatusConflict, w.Code, w.Body.String())
	}
	if repo.createCalled {
		t.Fatalf("expected create not to be called when attempt limit reached")
	}
}

func TestImageImportHandlerCreateImportRequest_SORDenied_RecordsSORPolicyLabels(t *testing.T) {
	tenantID := uuid.New()
	repo := &imageImportHandlerTestRepository{}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: false}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())
	denials := denialtelemetry.NewMetrics()
	handler.SetDenialMetrics(denials)

	tenantPolicy, _ := systemconfig.NewSystemConfig(
		&tenantID,
		systemconfig.ConfigTypeToolSettings,
		"sor_registration",
		systemconfig.SORRegistrationConfig{Enforce: true, RuntimeErrorMode: "deny"},
		"test",
		uuid.New(),
	)
	configRepo := &imageImportHandlerTestSystemConfigRepo{
		tenantID:     tenantID,
		tenantConfig: tenantPolicy,
		defaultCfgByID: map[uuid.UUID]*systemconfig.SystemConfig{
			tenantPolicy.ID(): tenantPolicy,
		},
	}
	handler.SetSystemConfigService(systemconfig.NewService(configRepo, zap.NewNop()))

	reqBody := map[string]string{
		"epr_record_id":    "APP-123",
		"source_registry":  "ghcr.io",
		"source_image_ref": "ghcr.io/org/app:1.0.0",
	}
	payload, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests", bytes.NewReader(payload))
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	w := httptest.NewRecorder()

	handler.CreateImportRequest(w, req)
	if w.Code != http.StatusPreconditionFailed {
		t.Fatalf("expected %d, got %d", http.StatusPreconditionFailed, w.Code)
	}

	rows := denials.Snapshot()
	if len(rows) != 1 {
		t.Fatalf("expected one denial row, got %d", len(rows))
	}
	if rows[0].Reason != "epr_registration_required" {
		t.Fatalf("expected epr_registration_required reason, got %s", rows[0].Reason)
	}
	if rows[0].Labels["epr_runtime_mode"] != "deny" {
		t.Fatalf("expected epr_runtime_mode=deny, got %q", rows[0].Labels["epr_runtime_mode"])
	}
	if rows[0].Labels["epr_policy_scope"] != "tenant" {
		t.Fatalf("expected epr_policy_scope=tenant, got %q", rows[0].Labels["epr_policy_scope"])
	}
}

func TestMapImportResponse_IncludesEvidenceFields(t *testing.T) {
	req := &imageimport.ImportRequest{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-1",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/acme/app:1.0.0",
		Status:            imageimport.StatusQuarantined,
		PolicyDecision:    "quarantine",
		PolicyReasonsJSON: `["critical_count(1) > max_critical(0)"]`,
		ScanSummaryJSON:   `{"vulnerabilities":{"critical":1}}`,
		SBOMSummaryJSON:   `{"package_count":123}`,
		SBOMEvidenceJSON:  `{"bomFormat":"spdx"}`,
		SourceImageDigest: "sha256:abc123",
	}

	got := mapImportResponse(req, nil, nil)
	if got.PolicyDecision != "quarantine" {
		t.Fatalf("expected policy decision in response, got %q", got.PolicyDecision)
	}
	if got.ScanSummaryJSON == "" || got.SBOMSummaryJSON == "" || got.SBOMEvidenceJSON == "" {
		t.Fatalf("expected evidence JSON fields to be populated in response")
	}
	if got.SourceImageDigest == "" {
		t.Fatalf("expected source image digest in response")
	}
	if got.SyncState != "completed" || !got.Retryable {
		t.Fatalf("expected completed retryable state for quarantined import, got state=%q retryable=%v", got.SyncState, got.Retryable)
	}
}

func TestMapImportResponse_ImportingCatalogSyncPending_IsRetryable(t *testing.T) {
	req := &imageimport.ImportRequest{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-2",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/acme/app:2.0.0",
		Status:            imageimport.StatusImporting,
		ErrorMessage:      "catalog image is not ready for evidence sync",
	}

	got := mapImportResponse(req, nil, nil)
	if got.SyncState != "catalog_sync_pending" {
		t.Fatalf("expected sync_state catalog_sync_pending, got %q", got.SyncState)
	}
	if !got.Retryable {
		t.Fatalf("expected retryable=true when catalog sync is pending")
	}
}

func TestMapImportResponse_FailedDispatch_IsRetryable(t *testing.T) {
	req := &imageimport.ImportRequest{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-3",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/acme/app:3.0.0",
		Status:            imageimport.StatusFailed,
		ErrorMessage:      "dispatch_failed: context deadline exceeded",
	}

	got := mapImportResponse(req, nil, nil)
	if got.SyncState != "dispatch_failed" {
		t.Fatalf("expected sync_state dispatch_failed, got %q", got.SyncState)
	}
	if !got.Retryable {
		t.Fatalf("expected retryable=true for dispatch_failed")
	}
}

func TestMapImportResponse_ExecutionContract_AwaitingDispatchIncludesQueuedTimestamp(t *testing.T) {
	now := time.Now().UTC()
	req := &imageimport.ImportRequest{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-K02-1",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/acme/app:k02-1",
		Status:            imageimport.StatusApproved,
		CreatedAt:         now.Add(-5 * time.Minute),
		UpdatedAt:         now,
	}

	got := mapImportResponse(req, nil, nil)
	if got.ExecutionState != "awaiting_dispatch" {
		t.Fatalf("expected execution_state awaiting_dispatch, got %q", got.ExecutionState)
	}
	if got.DispatchQueuedAt == "" {
		t.Fatalf("expected dispatch_queued_at to be populated")
	}
	if got.ExecutionStateUpdatedAt == "" {
		t.Fatalf("expected execution_state_updated_at to be populated")
	}
	if got.PipelineStartedAt != "" || got.EvidenceReadyAt != "" || got.ReleaseReadyAt != "" {
		t.Fatalf("expected downstream timestamps empty for awaiting_dispatch state, got pipeline=%q evidence=%q release=%q", got.PipelineStartedAt, got.EvidenceReadyAt, got.ReleaseReadyAt)
	}
}

func TestMapImportResponse_ExecutionContract_AwaitingApprovalHasNoDispatchTimestamp(t *testing.T) {
	now := time.Now().UTC()
	req := &imageimport.ImportRequest{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-K02-1A",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/acme/app:k02-1a",
		Status:            imageimport.StatusPending,
		CreatedAt:         now.Add(-5 * time.Minute),
		UpdatedAt:         now,
	}

	got := mapImportResponse(req, nil, nil)
	if got.SyncState != "awaiting_approval" {
		t.Fatalf("expected sync_state awaiting_approval, got %q", got.SyncState)
	}
	if got.ExecutionState != "awaiting_approval" {
		t.Fatalf("expected execution_state awaiting_approval, got %q", got.ExecutionState)
	}
	if got.DispatchQueuedAt != "" {
		t.Fatalf("expected dispatch_queued_at to be empty before approval, got %q", got.DispatchQueuedAt)
	}
	if got.Retryable {
		t.Fatalf("expected retryable=false while awaiting approval")
	}
}

func TestMapImportResponse_ExecutionContract_CatalogSyncPendingMapsToEvidencePending(t *testing.T) {
	now := time.Now().UTC()
	req := &imageimport.ImportRequest{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-K02-2",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/acme/app:k02-2",
		Status:            imageimport.StatusImporting,
		ErrorMessage:      "catalog image is not ready for evidence sync",
		CreatedAt:         now.Add(-10 * time.Minute),
		UpdatedAt:         now,
	}

	got := mapImportResponse(req, nil, nil)
	if got.ExecutionState != "evidence_pending" {
		t.Fatalf("expected execution_state evidence_pending, got %q", got.ExecutionState)
	}
	if got.PipelineStartedAt == "" {
		t.Fatalf("expected pipeline_started_at for evidence_pending state")
	}
	if got.EvidenceReadyAt != "" || got.ReleaseReadyAt != "" {
		t.Fatalf("expected evidence/release timestamps empty while pending, got evidence=%q release=%q", got.EvidenceReadyAt, got.ReleaseReadyAt)
	}
}

func TestMapImportResponse_ExecutionContract_SuccessReadyForReleaseIncludesReleaseReadyTimestamp(t *testing.T) {
	now := time.Now().UTC()
	req := &imageimport.ImportRequest{
		ID:                 uuid.New(),
		TenantID:           uuid.New(),
		RequestedByUserID:  uuid.New(),
		SORRecordID:        "APP-K02-3",
		SourceRegistry:     "ghcr.io",
		SourceImageRef:     "ghcr.io/acme/app:k02-3",
		Status:             imageimport.StatusSuccess,
		ReleaseState:       imageimport.ReleaseStateReadyForRelease,
		PolicyDecision:     "pass",
		PolicySnapshotJSON: `{"decision":"pass"}`,
		ScanSummaryJSON:    `{"critical":0}`,
		SBOMSummaryJSON:    `{"packages":42}`,
		SourceImageDigest:  "sha256:k02-3",
		CreatedAt:          now.Add(-15 * time.Minute),
		UpdatedAt:          now,
	}

	got := mapImportResponse(req, nil, nil)
	if got.ExecutionState != "ready_for_release" {
		t.Fatalf("expected execution_state ready_for_release, got %q", got.ExecutionState)
	}
	if got.PipelineStartedAt == "" || got.EvidenceReadyAt == "" || got.ReleaseReadyAt == "" {
		t.Fatalf("expected pipeline/evidence/release timestamps populated, got pipeline=%q evidence=%q release=%q", got.PipelineStartedAt, got.EvidenceReadyAt, got.ReleaseReadyAt)
	}
}

func TestMapImportResponse_FailureClassification_DispatchTimeout(t *testing.T) {
	req := &imageimport.ImportRequest{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-K03-1",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/acme/app:k03-1",
		Status:            imageimport.StatusFailed,
		ErrorMessage:      "dispatch_failed: context deadline exceeded",
	}

	got := mapImportResponse(req, nil, nil)
	if got.FailureClass != "dispatch" || got.FailureCode != "dispatch_timeout" {
		t.Fatalf("expected dispatch_timeout classification, got class=%q code=%q", got.FailureClass, got.FailureCode)
	}
}

func TestMapImportResponse_FailureClassification_RuntimeFallback(t *testing.T) {
	req := &imageimport.ImportRequest{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-K03-2",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/acme/app:k03-2",
		Status:            imageimport.StatusFailed,
		ErrorMessage:      "unexpected terminal error in monitor path",
	}

	got := mapImportResponse(req, nil, nil)
	if got.FailureClass != "runtime" || got.FailureCode != "runtime_failed" {
		t.Fatalf("expected runtime_failed classification, got class=%q code=%q", got.FailureClass, got.FailureCode)
	}
}

func TestMapImportResponse_TerminalEvidenceFallbacks_AreNonEmpty(t *testing.T) {
	req := &imageimport.ImportRequest{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-4",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/acme/app:4.0.0",
		Status:            imageimport.StatusSuccess,
	}

	got := mapImportResponse(req, nil, nil)
	if got.PolicyDecision == "" {
		t.Fatalf("expected non-empty policy decision fallback for terminal import")
	}
	if got.PolicyReasonsJSON == "" || got.PolicySnapshotJSON == "" {
		t.Fatalf("expected non-empty policy JSON fallbacks for terminal import")
	}
	if got.ScanSummaryJSON == "" || got.SBOMSummaryJSON == "" || got.SBOMEvidenceJSON == "" {
		t.Fatalf("expected non-empty scan/sbom JSON fallbacks for terminal import")
	}
	if got.SourceImageDigest == "" {
		t.Fatalf("expected non-empty source image digest fallback for terminal import")
	}
}

func TestMapImportResponse_TerminalEvidenceFallbacks_DoNotOverrideNonEmptyValues(t *testing.T) {
	req := &imageimport.ImportRequest{
		ID:                 uuid.New(),
		TenantID:           uuid.New(),
		RequestedByUserID:  uuid.New(),
		SORRecordID:        "APP-5",
		SourceRegistry:     "ghcr.io",
		SourceImageRef:     "ghcr.io/acme/app:5.0.0",
		Status:             imageimport.StatusQuarantined,
		PolicyDecision:     "quarantine",
		PolicyReasonsJSON:  `["critical_count(2) > max_critical(0)"]`,
		PolicySnapshotJSON: `{"mode":"enforce"}`,
		ScanSummaryJSON:    `{"vulnerabilities":{"critical":2}}`,
		SBOMSummaryJSON:    `{"package_count":321}`,
		SBOMEvidenceJSON:   `{"bomFormat":"spdx"}`,
		SourceImageDigest:  "sha256:feedface",
	}

	got := mapImportResponse(req, nil, nil)
	if got.PolicyDecision != req.PolicyDecision {
		t.Fatalf("expected policy decision to be preserved, got %q", got.PolicyDecision)
	}
	if got.PolicyReasonsJSON != req.PolicyReasonsJSON || got.PolicySnapshotJSON != req.PolicySnapshotJSON {
		t.Fatalf("expected policy JSON values to be preserved")
	}
	if got.ScanSummaryJSON != req.ScanSummaryJSON || got.SBOMSummaryJSON != req.SBOMSummaryJSON || got.SBOMEvidenceJSON != req.SBOMEvidenceJSON {
		t.Fatalf("expected scan/sbom JSON values to be preserved")
	}
	if got.SourceImageDigest != req.SourceImageDigest {
		t.Fatalf("expected source image digest to be preserved, got %q", got.SourceImageDigest)
	}
}

func TestMapImportResponse_NonTerminal_DoesNotApplyEvidenceFallbacks(t *testing.T) {
	req := &imageimport.ImportRequest{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-11",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/acme/app:11.0.0",
		Status:            imageimport.StatusImporting,
	}

	got := mapImportResponse(req, nil, nil)
	if got.PolicyDecision != "" || got.PolicyReasonsJSON != "" || got.PolicySnapshotJSON != "" {
		t.Fatalf("expected no policy fallbacks for non-terminal import, got %+v", got)
	}
	if got.ScanSummaryJSON != "" || got.SBOMSummaryJSON != "" || got.SBOMEvidenceJSON != "" || got.SourceImageDigest != "" {
		t.Fatalf("expected no evidence fallbacks for non-terminal import, got %+v", got)
	}
}

func TestImageImportHandlerApproveImportRequest_QueuesApprovalDecisionStep(t *testing.T) {
	tenantID := uuid.New()
	importID := uuid.New()
	instanceID := uuid.New()
	stepID := uuid.New()
	workflowRepo := &imageImportWorkflowRepoStub{
		instance: &domainworkflow.Instance{
			ID:          instanceID,
			TenantID:    &tenantID,
			SubjectType: "external_image_import",
			SubjectID:   importID,
		},
		steps: []domainworkflow.Step{
			{
				ID:       stepID,
				StepKey:  "approval.decision",
				Status:   domainworkflow.StepStatusBlocked,
				Payload:  map[string]interface{}{"external_image_import_id": importID.String()},
				Attempts: 0,
			},
		},
	}

	repo := &imageImportHandlerTestRepository{}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())
	handler.SetWorkflowRepository(workflowRepo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests/"+importID.String()+"/approve", nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", importID.String())
	w := httptest.NewRecorder()

	handler.ApproveImportRequest(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, w.Code)
	}
	if workflowRepo.updateStep == nil {
		t.Fatalf("expected approval decision step update to be queued")
	}
	if workflowRepo.updateStep.Status != domainworkflow.StepStatusPending {
		t.Fatalf("expected approval decision step status pending, got %s", workflowRepo.updateStep.Status)
	}
	if approved, ok := workflowRepo.updateStep.Payload["approved"].(bool); !ok || !approved {
		t.Fatalf("expected approved=true payload, got %#v", workflowRepo.updateStep.Payload["approved"])
	}
}

func TestImageImportHandlerApproveImportRequest_RejectsTenantContextMismatch(t *testing.T) {
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	importID := uuid.New()
	workflowRepo := &imageImportWorkflowRepoStub{
		instance: &domainworkflow.Instance{
			ID:          uuid.New(),
			TenantID:    &otherTenantID,
			SubjectType: "external_image_import",
			SubjectID:   importID,
		},
		steps: []domainworkflow.Step{
			{ID: uuid.New(), StepKey: "approval.decision", Status: domainworkflow.StepStatusBlocked},
		},
	}

	repo := &imageImportHandlerTestRepository{}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())
	handler.SetWorkflowRepository(workflowRepo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests/"+importID.String()+"/approve", nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", importID.String())
	w := httptest.NewRecorder()

	handler.ApproveImportRequest(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, w.Code)
	}
	if workflowRepo.updateStep != nil {
		t.Fatalf("expected no approval decision update when tenant context mismatches")
	}
}

func TestImageImportHandlerApproveImportRequestAdmin_AllowsCrossTenantDecision(t *testing.T) {
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	importID := uuid.New()
	workflowRepo := &imageImportWorkflowRepoStub{
		instance: &domainworkflow.Instance{
			ID:          uuid.New(),
			TenantID:    &otherTenantID,
			SubjectType: "external_image_import",
			SubjectID:   importID,
		},
		steps: []domainworkflow.Step{
			{ID: uuid.New(), StepKey: "approval.decision", Status: domainworkflow.StepStatusBlocked},
		},
	}

	repo := &imageImportHandlerTestRepository{}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())
	handler.SetWorkflowRepository(workflowRepo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/images/import-requests/"+importID.String()+"/approve", nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", importID.String())
	w := httptest.NewRecorder()

	handler.ApproveImportRequestAdmin(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, w.Code)
	}
	if workflowRepo.updateStep == nil {
		t.Fatalf("expected approval decision step update to be queued")
	}
}

func TestImageImportHandlerRejectImportRequest_QueuesRejectedDecision(t *testing.T) {
	tenantID := uuid.New()
	importID := uuid.New()
	workflowRepo := &imageImportWorkflowRepoStub{
		instance: &domainworkflow.Instance{
			ID:          uuid.New(),
			TenantID:    &tenantID,
			SubjectType: "external_image_import",
			SubjectID:   importID,
		},
		steps: []domainworkflow.Step{
			{ID: uuid.New(), StepKey: "approval.decision", Status: domainworkflow.StepStatusBlocked},
		},
	}

	repo := &imageImportHandlerTestRepository{}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())
	handler.SetWorkflowRepository(workflowRepo)

	reqBody := map[string]string{"reason": "risk acceptance denied"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests/"+importID.String()+"/reject", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", importID.String())
	w := httptest.NewRecorder()

	handler.RejectImportRequest(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, w.Code)
	}
	if workflowRepo.updateStep == nil {
		t.Fatalf("expected approval decision step update to be queued")
	}
	if approved, ok := workflowRepo.updateStep.Payload["approved"].(bool); !ok || approved {
		t.Fatalf("expected approved=false payload, got %#v", workflowRepo.updateStep.Payload["approved"])
	}
	if reason, _ := workflowRepo.updateStep.Payload["approval_reason"].(string); reason != "risk acceptance denied" {
		t.Fatalf("expected rejection reason to be set, got %q", reason)
	}
}

func TestImageImportHandlerReleaseImportRequest_CapabilityDenied(t *testing.T) {
	tenantID := uuid.New()
	importID := uuid.New()
	repo := &imageImportHandlerTestRepository{
		byID: map[uuid.UUID]*imageimport.ImportRequest{
			importID: {
				ID:                 importID,
				TenantID:           tenantID,
				RequestedByUserID:  uuid.New(),
				RequestType:        imageimport.RequestTypeQuarantine,
				Status:             imageimport.StatusSuccess,
				PolicyDecision:     "pass",
				PolicySnapshotJSON: `{"decision":"pass"}`,
				ScanSummaryJSON:    `{"critical":0}`,
				SBOMSummaryJSON:    `{"packages":42}`,
				SourceImageDigest:  "sha256:release-ok",
				UpdatedAt:          time.Now().UTC(),
			},
		},
	}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())
	handler.SetReleaseCapabilityChecker(&imageImportHandlerTestCapabilityChecker{releaseEntitled: false})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests/"+importID.String()+"/release", strings.NewReader(`{"destination_image_ref":"registry.example.com/released/app:1.0.0","destination_registry_auth_id":"11111111-1111-1111-1111-111111111111"}`))
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", importID.String())
	w := httptest.NewRecorder()

	handler.ReleaseImportRequest(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestImageImportHandlerReleaseImportRequest_RequiresDestinationImageRef(t *testing.T) {
	tenantID := uuid.New()
	importID := uuid.New()
	repo := &imageImportHandlerTestRepository{
		byID: map[uuid.UUID]*imageimport.ImportRequest{
			importID: {
				ID:                 importID,
				TenantID:           tenantID,
				RequestedByUserID:  uuid.New(),
				RequestType:        imageimport.RequestTypeQuarantine,
				Status:             imageimport.StatusSuccess,
				PolicyDecision:     "pass",
				PolicySnapshotJSON: `{"decision":"pass"}`,
				ScanSummaryJSON:    `{"critical":0}`,
				SBOMSummaryJSON:    `{"packages":42}`,
				SourceImageDigest:  "sha256:release-ok",
				UpdatedAt:          time.Now().UTC(),
			},
		},
	}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())
	handler.SetReleaseCapabilityChecker(&imageImportHandlerTestCapabilityChecker{releaseEntitled: true})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests/"+importID.String()+"/release", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", importID.String())
	w := httptest.NewRecorder()

	handler.ReleaseImportRequest(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestImageImportHandlerReleaseImportRequest_NotEligible(t *testing.T) {
	tenantID := uuid.New()
	importID := uuid.New()
	repo := &imageImportHandlerTestRepository{
		byID: map[uuid.UUID]*imageimport.ImportRequest{
			importID: {
				ID:                importID,
				TenantID:          tenantID,
				RequestedByUserID: uuid.New(),
				RequestType:       imageimport.RequestTypeQuarantine,
				Status:            imageimport.StatusPending,
			},
		},
	}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())
	handler.SetReleaseCapabilityChecker(&imageImportHandlerTestCapabilityChecker{releaseEntitled: true})
	releaseMetrics := releasetelemetry.NewMetrics()
	handler.SetReleaseMetrics(releaseMetrics)
	eventBus := &imageImportEventBusStub{}
	handler.SetEventBus(eventBus)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests/"+importID.String()+"/release", strings.NewReader(`{"destination_image_ref":"registry.example.com/released/app:1.0.0","destination_registry_auth_id":"11111111-1111-1111-1111-111111111111"}`))
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", importID.String())
	w := httptest.NewRecorder()

	handler.ReleaseImportRequest(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, w.Code)
	}
	if len(eventBus.events) != 1 {
		t.Fatalf("expected one release event, got %d", len(eventBus.events))
	}
	if eventBus.events[0].Type != messaging.EventTypeQuarantineReleaseFailed {
		t.Fatalf("expected release_failed event, got %q", eventBus.events[0].Type)
	}
	snapshot := releaseMetrics.Snapshot()
	if snapshot.Failed != 1 || snapshot.Total != 1 {
		t.Fatalf("expected failed release metric count=1, got %+v", snapshot)
	}
}

func TestImageImportHandlerReleaseImportRequest_NotEligibleWhenEvidenceIncomplete(t *testing.T) {
	tenantID := uuid.New()
	importID := uuid.New()
	repo := &imageImportHandlerTestRepository{
		byID: map[uuid.UUID]*imageimport.ImportRequest{
			importID: {
				ID:                 importID,
				TenantID:           tenantID,
				RequestedByUserID:  uuid.New(),
				RequestType:        imageimport.RequestTypeQuarantine,
				Status:             imageimport.StatusSuccess,
				PolicyDecision:     "pass",
				PolicySnapshotJSON: `{"decision":"pass"}`,
				ScanSummaryJSON:    `{}`,
				SBOMSummaryJSON:    `{"packages":42}`,
				SourceImageDigest:  "sha256:evidence-missing",
				UpdatedAt:          time.Now().UTC(),
			},
		},
	}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())
	handler.SetReleaseCapabilityChecker(&imageImportHandlerTestCapabilityChecker{releaseEntitled: true})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests/"+importID.String()+"/release", strings.NewReader(`{"destination_image_ref":"registry.example.com/released/app:1.0.0","destination_registry_auth_id":"11111111-1111-1111-1111-111111111111"}`))
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", importID.String())
	w := httptest.NewRecorder()

	handler.ReleaseImportRequest(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, w.Code)
	}
	var decoded struct {
		Error struct {
			Code    string                 `json:"code"`
			Message string                 `json:"message"`
			Details map[string]interface{} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if decoded.Error.Code != "release_not_eligible" {
		t.Fatalf("expected release_not_eligible code, got %q", decoded.Error.Code)
	}
	if decoded.Error.Details["release_blocker_reason"] != "evidence_incomplete" {
		t.Fatalf("expected release_blocker_reason evidence_incomplete, got %+v", decoded.Error.Details["release_blocker_reason"])
	}
}

func TestImageImportHandlerReleaseImportRequest_AcceptsEligibleImport(t *testing.T) {
	tenantID := uuid.New()
	importID := uuid.New()
	repo := &imageImportHandlerTestRepository{
		byID: map[uuid.UUID]*imageimport.ImportRequest{
			importID: {
				ID:                 importID,
				TenantID:           tenantID,
				RequestedByUserID:  uuid.New(),
				RequestType:        imageimport.RequestTypeQuarantine,
				Status:             imageimport.StatusSuccess,
				PolicyDecision:     "pass",
				PolicySnapshotJSON: `{"decision":"pass"}`,
				ScanSummaryJSON:    `{"critical":0}`,
				SBOMSummaryJSON:    `{"packages":42}`,
				SourceImageDigest:  "sha256:release-ok",
				UpdatedAt:          time.Now().UTC(),
			},
		},
	}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())
	handler.SetReleaseCapabilityChecker(&imageImportHandlerTestCapabilityChecker{releaseEntitled: true})
	releaseMetrics := releasetelemetry.NewMetrics()
	handler.SetReleaseMetrics(releaseMetrics)
	eventBus := &imageImportEventBusStub{}
	handler.SetEventBus(eventBus)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/import-requests/"+importID.String()+"/release", strings.NewReader(`{"destination_image_ref":"registry.example.com/released/app:1.0.0","destination_registry_auth_id":"11111111-1111-1111-1111-111111111111"}`))
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", importID.String())
	w := httptest.NewRecorder()

	handler.ReleaseImportRequest(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, w.Code)
	}

	var decoded struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if decoded.Data["release_state"] != "released" {
		t.Fatalf("expected release_state released, got %+v", decoded.Data["release_state"])
	}
	if len(eventBus.events) != 2 {
		t.Fatalf("expected two release events, got %d", len(eventBus.events))
	}
	if eventBus.events[0].Type != messaging.EventTypeQuarantineReleaseRequested {
		t.Fatalf("expected release_requested event, got %q", eventBus.events[0].Type)
	}
	if eventBus.events[1].Type != messaging.EventTypeQuarantineReleased {
		t.Fatalf("expected released event, got %q", eventBus.events[1].Type)
	}
	if eventBus.events[0].Payload["external_image_import_id"] != importID.String() {
		t.Fatalf("expected payload import id %q, got %#v", importID.String(), eventBus.events[0].Payload["external_image_import_id"])
	}
	snapshot := releaseMetrics.Snapshot()
	if snapshot.Requested != 1 || snapshot.Released != 1 || snapshot.Total != 2 {
		t.Fatalf("expected requested/released telemetry counts to be tracked, got %+v", snapshot)
	}
}

func TestImageImportHandlerConsumeReleasedArtifact_AcceptsReleasedArtifact(t *testing.T) {
	tenantID := uuid.New()
	importID := uuid.New()
	projectID := uuid.New().String()
	repo := &imageImportHandlerTestRepository{
		byID: map[uuid.UUID]*imageimport.ImportRequest{
			importID: {
				ID:                importID,
				TenantID:          tenantID,
				RequestedByUserID: uuid.New(),
				RequestType:       imageimport.RequestTypeQuarantine,
				Status:            imageimport.StatusSuccess,
				ReleaseState:      imageimport.ReleaseStateReleased,
				InternalImageRef:  "registry.local/quarantine/acme/nginx:1.0.0",
				SourceImageDigest: "sha256:abc123",
			},
		},
	}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())
	releaseMetrics := releasetelemetry.NewMetrics()
	handler.SetReleaseMetrics(releaseMetrics)
	eventBus := &imageImportEventBusStub{}
	handler.SetEventBus(eventBus)

	reqBody := map[string]string{
		"project_id": projectID,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/released-artifacts/"+importID.String()+"/consume", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", importID.String())
	w := httptest.NewRecorder()

	handler.ConsumeReleasedArtifact(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, w.Code)
	}

	if len(eventBus.events) != 1 {
		t.Fatalf("expected one consumption event, got %d", len(eventBus.events))
	}
	if eventBus.events[0].Type != messaging.EventTypeQuarantineReleaseConsumed {
		t.Fatalf("expected release_consumed event, got %q", eventBus.events[0].Type)
	}
	if eventBus.events[0].Payload["project_id"] != projectID {
		t.Fatalf("expected project_id %q in payload, got %#v", projectID, eventBus.events[0].Payload["project_id"])
	}

	snapshot := releaseMetrics.Snapshot()
	if snapshot.Consumed != 1 {
		t.Fatalf("expected consumed release metric count=1, got %+v", snapshot)
	}
}

func TestImageImportHandlerConsumeReleasedArtifact_RejectsNonReleased(t *testing.T) {
	tenantID := uuid.New()
	importID := uuid.New()
	repo := &imageImportHandlerTestRepository{
		byID: map[uuid.UUID]*imageimport.ImportRequest{
			importID: {
				ID:                importID,
				TenantID:          tenantID,
				RequestedByUserID: uuid.New(),
				RequestType:       imageimport.RequestTypeQuarantine,
				Status:            imageimport.StatusSuccess,
				ReleaseState:      imageimport.ReleaseStateReadyForRelease,
			},
		},
	}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())
	eventBus := &imageImportEventBusStub{}
	handler.SetEventBus(eventBus)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/released-artifacts/"+importID.String()+"/consume", nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", importID.String())
	w := httptest.NewRecorder()

	handler.ConsumeReleasedArtifact(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, w.Code)
	}
	if len(eventBus.events) != 0 {
		t.Fatalf("expected no event on rejected consume, got %d", len(eventBus.events))
	}
}

func TestImageImportHandlerPublishReleaseEvent_EmitsAlertTransitionsWithDedupe(t *testing.T) {
	tenantID := uuid.New()
	importID := uuid.New()
	actorID := uuid.New()

	releasePolicyConfig, _ := systemconfig.NewSystemConfig(
		nil,
		systemconfig.ConfigTypeToolSettings,
		"release_governance_policy",
		systemconfig.ReleaseGovernancePolicyConfig{
			Enabled:                      true,
			FailureRatioThreshold:        0.90,
			ConsecutiveFailuresThreshold: 2,
			MinimumSamples:               1,
			WindowMinutes:                60,
		},
		"test",
		uuid.New(),
	)
	configRepo := &imageImportHandlerTestSystemConfigRepo{
		releasePolicy: releasePolicyConfig,
		defaultCfgByID: map[uuid.UUID]*systemconfig.SystemConfig{
			releasePolicyConfig.ID(): releasePolicyConfig,
		},
	}

	handler := NewImageImportHandler(nil, zap.NewNop())
	handler.SetSystemConfigService(systemconfig.NewService(configRepo, zap.NewNop()))
	handler.SetReleaseMetrics(releasetelemetry.NewMetrics())
	handler.SetEventBus(&imageImportEventBusStub{})

	req := &imageimport.ImportRequest{
		ID:                importID,
		TenantID:          tenantID,
		RequestedByUserID: actorID,
		RequestType:       imageimport.RequestTypeQuarantine,
		Status:            imageimport.StatusSuccess,
	}

	// 1) failure => degraded transition alert
	req.Status = imageimport.StatusPending
	handler.publishReleaseEvent(context.Background(), actorID, req, messaging.EventTypeQuarantineReleaseFailed, "not eligible", nil)
	// 2) failure => still degraded, no duplicate alert
	handler.publishReleaseEvent(context.Background(), actorID, req, messaging.EventTypeQuarantineReleaseFailed, "not eligible", nil)
	// 3) release success => recovery transition alert
	req.Status = imageimport.StatusSuccess
	handler.publishReleaseEvent(context.Background(), actorID, req, messaging.EventTypeQuarantineReleased, "", nil)

	eventBus, ok := handler.eventBus.(*imageImportEventBusStub)
	if !ok {
		t.Fatalf("expected image import event bus stub")
	}

	var degradedCount, recoveredCount int
	for _, event := range eventBus.events {
		switch event.Type {
		case messaging.EventTypeQuarantineReleaseAlert:
			degradedCount++
		case messaging.EventTypeQuarantineReleaseRecovered:
			recoveredCount++
		}
	}
	if degradedCount != 1 {
		t.Fatalf("expected exactly one degraded alert transition, got %d", degradedCount)
	}
	if recoveredCount != 1 {
		t.Fatalf("expected exactly one recovered alert transition, got %d", recoveredCount)
	}
}

func TestImageImportHandlerLoadDecisionTimeline_FromApprovalStepPayload(t *testing.T) {
	importID := uuid.New()
	workflowRepo := &imageImportWorkflowRepoStub{
		instance: &domainworkflow.Instance{
			ID:          uuid.New(),
			SubjectType: "external_image_import",
			SubjectID:   importID,
		},
		steps: []domainworkflow.Step{
			{
				ID:      uuid.New(),
				StepKey: approvalDecisionStepKey,
				Status:  domainworkflow.StepStatusSucceeded,
				Payload: map[string]interface{}{
					"approval_status":     "approved",
					"approval_reason":     "",
					"approved_by_user_id": "user-123",
					"approved_at":         "2026-03-01T10:00:00Z",
				},
			},
		},
	}

	handler := NewImageImportHandler(nil, zap.NewNop())
	handler.SetWorkflowRepository(workflowRepo)
	got := handler.loadDecisionTimeline(context.Background(), importID)
	if got == nil {
		t.Fatalf("expected decision timeline to be populated")
	}
	if got.DecisionStatus != "approved" {
		t.Fatalf("expected decision status approved, got %q", got.DecisionStatus)
	}
	if got.DecidedByUserID != "user-123" {
		t.Fatalf("expected decided_by_user_id user-123, got %q", got.DecidedByUserID)
	}
	if got.DecidedAt != "2026-03-01T10:00:00Z" {
		t.Fatalf("expected decided_at to be preserved, got %q", got.DecidedAt)
	}
	if got.WorkflowStepStatus != "succeeded" {
		t.Fatalf("expected workflow step status succeeded, got %q", got.WorkflowStepStatus)
	}
}

func TestMapImportResponse_IncludesDecisionTimeline(t *testing.T) {
	req := &imageimport.ImportRequest{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-12",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/acme/app:12.0.0",
		Status:            imageimport.StatusPending,
	}
	timeline := &imageImportDecisionTimeline{
		DecisionStatus:     "approved",
		DecisionReason:     "",
		DecidedByUserID:    "user-321",
		DecidedAt:          "2026-03-01T10:00:00Z",
		WorkflowStepStatus: "succeeded",
	}

	got := mapImportResponse(req, timeline, nil)
	if got.DecisionTimeline == nil {
		t.Fatalf("expected decision timeline in response")
	}
	if got.DecisionTimeline.DecisionStatus != "approved" {
		t.Fatalf("expected decision timeline status approved, got %q", got.DecisionTimeline.DecisionStatus)
	}
}

func TestImageImportHandlerLoadNotificationReconciliation_Delivered(t *testing.T) {
	tenantID := uuid.New()
	requesterID := uuid.New()
	importID := uuid.New()
	repo := &imageImportNotificationRepoStub{
		adminUserIDs: []uuid.UUID{uuid.New()},
		receiptCounts: map[string]int{
			"external.image.import.approved|" + importID.String() + ":approved": 2,
		},
		inAppCounts: map[string]int{
			importID.String() + "|external_image_import_approved": 2,
		},
	}

	handler := NewImageImportHandler(nil, zap.NewNop())
	handler.SetNotificationReconciliationRepository(repo)
	req := &imageimport.ImportRequest{
		ID:                importID,
		TenantID:          tenantID,
		RequestedByUserID: requesterID,
	}
	timeline := &imageImportDecisionTimeline{DecisionStatus: "approved"}
	got := handler.loadNotificationReconciliation(context.Background(), req, timeline)
	if got == nil {
		t.Fatalf("expected notification reconciliation data")
	}
	if got.DeliveryState != "delivered" {
		t.Fatalf("expected delivered state, got %q", got.DeliveryState)
	}
	if got.ExpectedRecipients != 2 || got.ReceiptCount != 2 || got.InAppNotificationCount != 2 {
		t.Fatalf("unexpected reconciliation counts: %+v", got)
	}
}

func TestMapImportResponse_IncludesNotificationReconciliation(t *testing.T) {
	req := &imageimport.ImportRequest{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		RequestedByUserID: uuid.New(),
		SORRecordID:       "APP-13",
		SourceRegistry:    "ghcr.io",
		SourceImageRef:    "ghcr.io/acme/app:13.0.0",
		Status:            imageimport.StatusPending,
	}
	reconciliation := &imageImportNotificationReconciliation{
		DecisionEventType:      "external.image.import.approved",
		IdempotencyKey:         req.ID.String() + ":approved",
		ExpectedRecipients:     2,
		ReceiptCount:           1,
		InAppNotificationCount: 1,
		DeliveryState:          "partial",
	}

	got := mapImportResponse(req, nil, reconciliation)
	if got.NotificationReconciliation == nil {
		t.Fatalf("expected notification reconciliation in response")
	}
	if got.NotificationReconciliation.DeliveryState != "partial" {
		t.Fatalf("expected partial delivery state, got %q", got.NotificationReconciliation.DeliveryState)
	}
}

func TestImageImportHandlerGetImportRequest_TerminalEvidenceFallbacksInAPIResponse(t *testing.T) {
	tenantID := uuid.New()
	importID := uuid.New()
	repo := &imageImportHandlerTestRepository{
		byID: map[uuid.UUID]*imageimport.ImportRequest{
			importID: {
				ID:                importID,
				TenantID:          tenantID,
				RequestedByUserID: uuid.New(),
				RequestType:       imageimport.RequestTypeQuarantine,
				SORRecordID:       "APP-6",
				SourceRegistry:    "ghcr.io",
				SourceImageRef:    "ghcr.io/acme/app:6.0.0",
				Status:            imageimport.StatusFailed,
				ErrorMessage:      "image pull failed",
			},
		},
	}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/import-requests/"+importID.String(), nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	req = withImageImportURLParam(req, "id", importID.String())
	w := httptest.NewRecorder()

	handler.GetImportRequest(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var decoded struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	data := decoded.Data
	if data["policy_decision"] == "" || data["policy_reasons_json"] == "" || data["policy_snapshot_json"] == "" {
		t.Fatalf("expected non-empty policy evidence fields in API response, got %+v", data)
	}
	if data["scan_summary_json"] == "" || data["sbom_summary_json"] == "" || data["sbom_evidence_json"] == "" {
		t.Fatalf("expected non-empty scan/sbom evidence fields in API response, got %+v", data)
	}
	if data["source_image_digest"] == "" {
		t.Fatalf("expected non-empty source_image_digest in API response")
	}
}

func TestImageImportHandlerListImportRequests_TerminalEvidenceFallbacksInAPIResponse(t *testing.T) {
	tenantID := uuid.New()
	importID := uuid.New()
	repo := &imageImportHandlerTestRepository{
		byID: map[uuid.UUID]*imageimport.ImportRequest{
			importID: {
				ID:                importID,
				TenantID:          tenantID,
				RequestedByUserID: uuid.New(),
				RequestType:       imageimport.RequestTypeQuarantine,
				SORRecordID:       "APP-7",
				SourceRegistry:    "ghcr.io",
				SourceImageRef:    "ghcr.io/acme/app:7.0.0",
				Status:            imageimport.StatusSuccess,
			},
		},
	}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/import-requests", nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	w := httptest.NewRecorder()

	handler.ListImportRequests(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var decoded struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(decoded.Data) != 1 {
		t.Fatalf("expected one row, got %d", len(decoded.Data))
	}
	row := decoded.Data[0]
	if row["policy_decision"] == "" || row["scan_summary_json"] == "" || row["sbom_summary_json"] == "" || row["sbom_evidence_json"] == "" || row["source_image_digest"] == "" {
		t.Fatalf("expected non-empty terminal evidence fields in list response, got %+v", row)
	}
}

func TestMapImportResponse_IncludesReleaseProjection(t *testing.T) {
	req := &imageimport.ImportRequest{
		ID:                 uuid.New(),
		TenantID:           uuid.New(),
		RequestedByUserID:  uuid.New(),
		SORRecordID:        "APP-REL-1",
		SourceRegistry:     "ghcr.io",
		SourceImageRef:     "ghcr.io/acme/app:release",
		Status:             imageimport.StatusSuccess,
		PolicyDecision:     "pass",
		PolicySnapshotJSON: `{"decision":"pass"}`,
		ScanSummaryJSON:    `{"critical":0}`,
		SBOMSummaryJSON:    `{"packages":42}`,
		SourceImageDigest:  "sha256:rel-1",
		UpdatedAt:          time.Now().UTC(),
	}

	got := mapImportResponse(req, nil, nil)
	if got.ReleaseState != "ready_for_release" {
		t.Fatalf("expected release_state ready_for_release, got %q", got.ReleaseState)
	}
	if !got.ReleaseEligible {
		t.Fatalf("expected release_eligible true")
	}
	if got.ReleaseBlockerReason != "" {
		t.Fatalf("expected empty release_blocker_reason, got %q", got.ReleaseBlockerReason)
	}
}

func TestMapImportResponse_ReleaseProjectionBlocksIncompleteEvidence(t *testing.T) {
	req := &imageimport.ImportRequest{
		ID:                 uuid.New(),
		TenantID:           uuid.New(),
		RequestedByUserID:  uuid.New(),
		SORRecordID:        "APP-REL-2",
		SourceRegistry:     "ghcr.io",
		SourceImageRef:     "ghcr.io/acme/app:release-incomplete",
		Status:             imageimport.StatusSuccess,
		PolicyDecision:     "pass",
		PolicySnapshotJSON: `{"decision":"pass"}`,
		ScanSummaryJSON:    `{}`,
		SBOMSummaryJSON:    `{"packages":42}`,
		SourceImageDigest:  "sha256:rel-2",
		UpdatedAt:          time.Now().UTC(),
	}

	got := mapImportResponse(req, nil, nil)
	if got.ReleaseState != "release_blocked" {
		t.Fatalf("expected release_state release_blocked, got %q", got.ReleaseState)
	}
	if got.ReleaseEligible {
		t.Fatalf("expected release_eligible false")
	}
	if got.ReleaseBlockerReason != "evidence_incomplete" {
		t.Fatalf("expected release_blocker_reason evidence_incomplete, got %q", got.ReleaseBlockerReason)
	}
}

func TestImageImportHandlerListReleasedArtifacts_Success(t *testing.T) {
	tenantID := uuid.New()
	releaseActor := uuid.New()
	now := time.Now().UTC()
	repo := &imageImportHandlerTestRepository{
		byID: map[uuid.UUID]*imageimport.ImportRequest{
			uuid.New(): {
				ID:                 uuid.New(),
				TenantID:           tenantID,
				RequestedByUserID:  uuid.New(),
				RequestType:        imageimport.RequestTypeQuarantine,
				SORRecordID:        "APP-REL-1",
				SourceRegistry:     "ghcr.io",
				SourceImageRef:     "ghcr.io/acme/nginx:1.0.0",
				Status:             imageimport.StatusSuccess,
				InternalImageRef:   "registry.local/quarantine/acme/nginx:1.0.0",
				SourceImageDigest:  "sha256:abc123",
				PolicyDecision:     "pass",
				PolicySnapshotJSON: "{}",
				ReleaseState:       imageimport.ReleaseStateReleased,
				ReleaseReason:      "approved",
				ReleaseActorUserID: &releaseActor,
				ReleaseRequestedAt: &now,
				ReleasedAt:         &now,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
			uuid.New(): {
				ID:                uuid.New(),
				TenantID:          tenantID,
				RequestedByUserID: uuid.New(),
				RequestType:       imageimport.RequestTypeQuarantine,
				SORRecordID:       "APP-REL-2",
				SourceRegistry:    "ghcr.io",
				SourceImageRef:    "ghcr.io/acme/blocked:2.0.0",
				Status:            imageimport.StatusQuarantined,
				ReleaseState:      imageimport.ReleaseStateReleaseBlocked,
			},
		},
	}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/released-artifacts?page=1&limit=25&search=nginx", nil)
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		UserID:   uuid.New(),
		TenantID: tenantID,
	}))
	w := httptest.NewRecorder()

	handler.ListReleasedArtifacts(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var decoded struct {
		Data       []map[string]interface{} `json:"data"`
		Pagination map[string]interface{}   `json:"pagination"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(decoded.Data) != 1 {
		t.Fatalf("expected one released artifact row, got %d", len(decoded.Data))
	}
	row := decoded.Data[0]
	if row["release_state"] != "released" {
		t.Fatalf("expected release_state released, got %+v", row["release_state"])
	}
	if row["source_image_ref"] != "ghcr.io/acme/nginx:1.0.0" {
		t.Fatalf("unexpected source_image_ref: %+v", row["source_image_ref"])
	}
	if row["consumption_ready"] != true {
		t.Fatalf("expected consumption_ready=true, got %+v", row["consumption_ready"])
	}
	if blocker, ok := row["consumption_blocker_reason"]; ok && blocker != "" {
		t.Fatalf("expected empty consumption_blocker_reason for ready artifact, got %+v", blocker)
	}
	if decoded.Pagination["total"] != float64(1) {
		t.Fatalf("expected pagination.total=1, got %+v", decoded.Pagination["total"])
	}
}

func TestImageImportHandlerListReleasedArtifacts_RequiresAuth(t *testing.T) {
	repo := &imageImportHandlerTestRepository{}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/released-artifacts", nil)
	w := httptest.NewRecorder()

	handler.ListReleasedArtifacts(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestImageImportHandlerGetImportRequestWorkflow_ReturnsWorkflowSteps(t *testing.T) {
	tenantID := uuid.New()
	importID := uuid.New()
	now := time.Now().UTC()
	repo := &imageImportHandlerTestRepository{
		byID: map[uuid.UUID]*imageimport.ImportRequest{
			importID: {
				ID:                importID,
				TenantID:          tenantID,
				RequestedByUserID: uuid.New(),
				RequestType:       imageimport.RequestTypeQuarantine,
				SORRecordID:       "sor-1",
				SourceRegistry:    "ghcr.io",
				SourceImageRef:    "ghcr.io/acme/api:1.2.3",
				Status:            imageimport.StatusPending,
				CreatedAt:         now,
				UpdatedAt:         now,
			},
		},
	}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())
	handler.SetWorkflowRepository(&imageImportWorkflowRepoStub{
		instance: &domainworkflow.Instance{
			ID:     uuid.New(),
			Status: domainworkflow.InstanceStatusRunning,
		},
		steps: []domainworkflow.Step{
			{
				StepKey:   "approval.request",
				Status:    domainworkflow.StepStatusSucceeded,
				Attempts:  1,
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				StepKey:   "approval.decision",
				Status:    domainworkflow.StepStatusRunning,
				Attempts:  1,
				CreatedAt: now.Add(time.Second),
				UpdatedAt: now.Add(time.Second),
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/import-requests/"+importID.String()+"/workflow", nil)
	req = withImageImportURLParam(req, "id", importID.String())
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		TenantID: tenantID,
		UserID:   uuid.New(),
	}))
	w := httptest.NewRecorder()

	handler.GetImportRequestWorkflow(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}
	var decoded BuildWorkflowResponse
	if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if decoded.InstanceID == "" {
		t.Fatalf("expected instance id in workflow response")
	}
	if len(decoded.Steps) != 2 {
		t.Fatalf("expected 2 workflow steps, got %d", len(decoded.Steps))
	}
	if decoded.Steps[0].StepKey != "approval.request" {
		t.Fatalf("expected first step approval.request, got %s", decoded.Steps[0].StepKey)
	}
}

func TestImageImportHandlerGetImportRequestLogs_ReturnsLifecycleLogs(t *testing.T) {
	tenantID := uuid.New()
	importID := uuid.New()
	now := time.Now().UTC()
	releasedAt := now.Add(2 * time.Minute)
	repo := &imageImportHandlerTestRepository{
		byID: map[uuid.UUID]*imageimport.ImportRequest{
			importID: {
				ID:                 importID,
				TenantID:           tenantID,
				RequestedByUserID:  uuid.New(),
				RequestType:        imageimport.RequestTypeQuarantine,
				SORRecordID:        "sor-2",
				SourceRegistry:     "ghcr.io",
				SourceImageRef:     "ghcr.io/acme/api:2.0.0",
				Status:             imageimport.StatusFailed,
				PipelineRunName:    "import-run-123",
				PipelineNamespace:  "tenant-a",
				ErrorMessage:       "dispatch_failed: no eligible provider",
				ReleaseRequestedAt: &now,
				ReleasedAt:         &releasedAt,
				CreatedAt:          now.Add(-5 * time.Minute),
				UpdatedAt:          now,
			},
		},
	}
	svc := imageimport.NewService(repo, &imageImportHandlerTestSORValidator{ok: true}, &imageImportHandlerTestCapabilityChecker{entitled: true}, nil, zap.NewNop())
	handler := NewImageImportHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/import-requests/"+importID.String()+"/logs?source=lifecycle", nil)
	req = withImageImportURLParam(req, "id", importID.String())
	req = req.WithContext(context.WithValue(req.Context(), "auth", &middleware.AuthContext{
		TenantID: tenantID,
		UserID:   uuid.New(),
	}))
	w := httptest.NewRecorder()

	handler.GetImportRequestLogs(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	var decoded struct {
		ImportRequestID string     `json:"import_request_id"`
		Logs            []LogEntry `json:"logs"`
		Total           int        `json:"total"`
		HasMore         bool       `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode logs response: %v", err)
	}
	if decoded.ImportRequestID != importID.String() {
		t.Fatalf("expected import_request_id=%s, got %s", importID.String(), decoded.ImportRequestID)
	}
	if len(decoded.Logs) == 0 {
		t.Fatalf("expected lifecycle logs in response")
	}
	hasLifecycleSource := false
	for _, entry := range decoded.Logs {
		if entry.Metadata != nil && entry.Metadata["source"] == "lifecycle" {
			hasLifecycleSource = true
			break
		}
	}
	if !hasLifecycleSource {
		t.Fatalf("expected at least one lifecycle log source entry")
	}
}

func withImageImportURLParam(req *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}
