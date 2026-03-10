package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

type readinessRepoStub struct {
	provider              *infrastructure.Provider
	providersByID         map[uuid.UUID]*infrastructure.Provider
	providersList         []infrastructure.Provider
	readinessStatus       string
	readinessMissing      []string
	activePrepareRun      *infrastructure.ProviderPrepareRun
	prepareRuns           []*infrastructure.ProviderPrepareRun
	prepareChecks         []*infrastructure.ProviderPrepareRunCheck
	prepareRunListCalls   int
	prepareCheckListCalls int
	tenantPrepares        map[string]*infrastructure.ProviderTenantNamespacePrepare
	savedPermissions      []*infrastructure.ProviderPermission
}

type repoWithoutPrepareStub struct {
	provider *infrastructure.Provider
}

func (r *repoWithoutPrepareStub) SaveProvider(ctx context.Context, provider *infrastructure.Provider) error {
	return nil
}

func (r *repoWithoutPrepareStub) FindProviderByID(ctx context.Context, id uuid.UUID) (*infrastructure.Provider, error) {
	if r.provider == nil || r.provider.ID != id {
		return nil, nil
	}
	return r.provider, nil
}

func (r *repoWithoutPrepareStub) FindProvidersByTenant(ctx context.Context, tenantID uuid.UUID, opts *infrastructure.ListProvidersOptions) (*infrastructure.ListProvidersResult, error) {
	return &infrastructure.ListProvidersResult{}, nil
}

func (r *repoWithoutPrepareStub) FindProvidersAll(ctx context.Context, opts *infrastructure.ListProvidersOptions) (*infrastructure.ListProvidersResult, error) {
	return &infrastructure.ListProvidersResult{}, nil
}

func (r *repoWithoutPrepareStub) UpdateProvider(ctx context.Context, provider *infrastructure.Provider) error {
	return nil
}

func (r *repoWithoutPrepareStub) DeleteProvider(ctx context.Context, id uuid.UUID) error { return nil }

func (r *repoWithoutPrepareStub) ExistsProviderByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	return false, nil
}

func (r *repoWithoutPrepareStub) SavePermission(ctx context.Context, permission *infrastructure.ProviderPermission) error {
	return nil
}

func (r *repoWithoutPrepareStub) FindPermissionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*infrastructure.ProviderPermission, error) {
	return nil, nil
}

func (r *repoWithoutPrepareStub) FindPermissionsByProvider(ctx context.Context, providerID uuid.UUID) ([]*infrastructure.ProviderPermission, error) {
	return nil, nil
}

func (r *repoWithoutPrepareStub) DeletePermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) error {
	return nil
}

func (r *repoWithoutPrepareStub) HasPermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) (bool, error) {
	return true, nil
}

func (r *repoWithoutPrepareStub) UpdateProviderHealth(ctx context.Context, providerID uuid.UUID, health *infrastructure.ProviderHealth) error {
	return nil
}

func (r *repoWithoutPrepareStub) GetProviderHealth(ctx context.Context, providerID uuid.UUID) (*infrastructure.ProviderHealth, error) {
	return nil, nil
}

func (r *repoWithoutPrepareStub) UpdateProviderReadiness(ctx context.Context, providerID uuid.UUID, status string, checkedAt time.Time, missingPrereqs []string) error {
	return nil
}

func (r *readinessRepoStub) SaveProvider(ctx context.Context, provider *infrastructure.Provider) error {
	return nil
}

func (r *readinessRepoStub) FindProviderByID(ctx context.Context, id uuid.UUID) (*infrastructure.Provider, error) {
	if r.providersByID != nil {
		if provider, ok := r.providersByID[id]; ok && provider != nil {
			copied := *provider
			return &copied, nil
		}
	}
	for _, provider := range r.providersList {
		if provider.ID == id {
			copied := provider
			return &copied, nil
		}
	}
	if r.provider == nil || r.provider.ID != id {
		return nil, nil
	}
	return r.provider, nil
}

func (r *readinessRepoStub) FindProvidersByTenant(ctx context.Context, tenantID uuid.UUID, opts *infrastructure.ListProvidersOptions) (*infrastructure.ListProvidersResult, error) {
	if len(r.providersList) > 0 {
		return &infrastructure.ListProvidersResult{
			Providers:  r.providersList,
			Page:       1,
			Limit:      len(r.providersList),
			Total:      len(r.providersList),
			TotalPages: 1,
		}, nil
	}
	if r.provider != nil {
		return &infrastructure.ListProvidersResult{
			Providers:  []infrastructure.Provider{*r.provider},
			Page:       1,
			Limit:      1,
			Total:      1,
			TotalPages: 1,
		}, nil
	}
	return &infrastructure.ListProvidersResult{}, nil
}
func (r *readinessRepoStub) FindProvidersAll(ctx context.Context, opts *infrastructure.ListProvidersOptions) (*infrastructure.ListProvidersResult, error) {
	if len(r.providersList) > 0 {
		return &infrastructure.ListProvidersResult{
			Providers:  r.providersList,
			Page:       1,
			Limit:      len(r.providersList),
			Total:      len(r.providersList),
			TotalPages: 1,
		}, nil
	}
	if r.provider != nil {
		return &infrastructure.ListProvidersResult{
			Providers:  []infrastructure.Provider{*r.provider},
			Page:       1,
			Limit:      1,
			Total:      1,
			TotalPages: 1,
		}, nil
	}
	return &infrastructure.ListProvidersResult{}, nil
}

func (r *readinessRepoStub) UpdateProvider(ctx context.Context, provider *infrastructure.Provider) error {
	return nil
}

func (r *readinessRepoStub) DeleteProvider(ctx context.Context, id uuid.UUID) error { return nil }

func (r *readinessRepoStub) ExistsProviderByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	return false, nil
}

func (r *readinessRepoStub) SavePermission(ctx context.Context, permission *infrastructure.ProviderPermission) error {
	r.savedPermissions = append(r.savedPermissions, permission)
	return nil
}

func (r *readinessRepoStub) FindPermissionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*infrastructure.ProviderPermission, error) {
	return nil, nil
}

func (r *readinessRepoStub) FindPermissionsByProvider(ctx context.Context, providerID uuid.UUID) ([]*infrastructure.ProviderPermission, error) {
	return nil, nil
}

func (r *readinessRepoStub) DeletePermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) error {
	return nil
}

func (r *readinessRepoStub) HasPermission(ctx context.Context, providerID, tenantID uuid.UUID, permission string) (bool, error) {
	return true, nil
}

func (r *readinessRepoStub) UpdateProviderHealth(ctx context.Context, providerID uuid.UUID, health *infrastructure.ProviderHealth) error {
	return nil
}

func (r *readinessRepoStub) GetProviderHealth(ctx context.Context, providerID uuid.UUID) (*infrastructure.ProviderHealth, error) {
	return nil, nil
}

func (r *readinessRepoStub) UpdateProviderReadiness(ctx context.Context, providerID uuid.UUID, status string, checkedAt time.Time, missingPrereqs []string) error {
	r.readinessStatus = status
	r.readinessMissing = append([]string{}, missingPrereqs...)
	return nil
}

func (r *readinessRepoStub) UpsertTenantNamespacePrepare(ctx context.Context, prepare *infrastructure.ProviderTenantNamespacePrepare) error {
	if r.tenantPrepares == nil {
		r.tenantPrepares = make(map[string]*infrastructure.ProviderTenantNamespacePrepare)
	}
	key := prepare.ProviderID.String() + ":" + prepare.TenantID.String()
	cp := *prepare
	r.tenantPrepares[key] = &cp
	return nil
}

func (r *readinessRepoStub) GetTenantNamespacePrepare(ctx context.Context, providerID, tenantID uuid.UUID) (*infrastructure.ProviderTenantNamespacePrepare, error) {
	if r.tenantPrepares == nil {
		return nil, nil
	}
	key := providerID.String() + ":" + tenantID.String()
	if prepare, ok := r.tenantPrepares[key]; ok && prepare != nil {
		cp := *prepare
		return &cp, nil
	}
	return nil, nil
}

func (r *readinessRepoStub) ListTenantNamespacePreparesByProvider(ctx context.Context, providerID uuid.UUID) ([]*infrastructure.ProviderTenantNamespacePrepare, error) {
	if r.tenantPrepares == nil {
		return nil, nil
	}
	out := make([]*infrastructure.ProviderTenantNamespacePrepare, 0, len(r.tenantPrepares))
	for _, prep := range r.tenantPrepares {
		if prep == nil || prep.ProviderID != providerID {
			continue
		}
		cp := *prep
		out = append(out, &cp)
	}
	return out, nil
}

func (r *readinessRepoStub) CreateProviderPrepareRun(ctx context.Context, run *infrastructure.ProviderPrepareRun) error {
	copied := *run
	r.prepareRuns = append([]*infrastructure.ProviderPrepareRun{&copied}, r.prepareRuns...)
	return nil
}

func (r *readinessRepoStub) UpdateProviderPrepareRunStatus(ctx context.Context, id uuid.UUID, status infrastructure.ProviderPrepareRunStatus, startedAt, completedAt *time.Time, errorMessage *string, resultSummary map[string]interface{}) error {
	for _, run := range r.prepareRuns {
		if run.ID == id {
			run.Status = status
			run.StartedAt = startedAt
			run.CompletedAt = completedAt
			run.ErrorMessage = errorMessage
			run.ResultSummary = resultSummary
			run.UpdatedAt = time.Now().UTC()
			if status == infrastructure.ProviderPrepareRunStatusRunning || status == infrastructure.ProviderPrepareRunStatusPending {
				r.activePrepareRun = run
			} else if r.activePrepareRun != nil && r.activePrepareRun.ID == id {
				r.activePrepareRun = nil
			}
			return nil
		}
	}
	return nil
}

func (r *readinessRepoStub) GetProviderPrepareRun(ctx context.Context, id uuid.UUID) (*infrastructure.ProviderPrepareRun, error) {
	for _, run := range r.prepareRuns {
		if run.ID == id {
			copied := *run
			return &copied, nil
		}
	}
	return nil, nil
}

func (r *readinessRepoStub) FindActiveProviderPrepareRunByProvider(ctx context.Context, providerID uuid.UUID) (*infrastructure.ProviderPrepareRun, error) {
	if r.activePrepareRun == nil || r.activePrepareRun.ProviderID != providerID {
		return nil, nil
	}
	copied := *r.activePrepareRun
	return &copied, nil
}

func (r *readinessRepoStub) ListProviderPrepareRunsByProvider(ctx context.Context, providerID uuid.UUID, limit, offset int) ([]*infrastructure.ProviderPrepareRun, error) {
	r.prepareRunListCalls++
	out := make([]*infrastructure.ProviderPrepareRun, 0, len(r.prepareRuns))
	for _, run := range r.prepareRuns {
		if run.ProviderID == providerID {
			copied := *run
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *readinessRepoStub) AddProviderPrepareRunCheck(ctx context.Context, check *infrastructure.ProviderPrepareRunCheck) error {
	copied := *check
	r.prepareChecks = append(r.prepareChecks, &copied)
	return nil
}

func (r *readinessRepoStub) ListProviderPrepareRunChecks(ctx context.Context, runID uuid.UUID, limit, offset int) ([]*infrastructure.ProviderPrepareRunCheck, error) {
	r.prepareCheckListCalls++
	out := make([]*infrastructure.ProviderPrepareRunCheck, 0, len(r.prepareChecks))
	for _, check := range r.prepareChecks {
		if check.RunID == runID {
			copied := *check
			out = append(out, &copied)
		}
	}
	return out, nil
}

func readinessTestAuthContext(tenantID uuid.UUID) *middleware.AuthContext {
	return &middleware.AuthContext{
		TenantID:    tenantID,
		UserID:      uuid.New(),
		Email:       "readiness-test@example.com",
		UserTenants: []uuid.UUID{tenantID},
	}
}

func readinessSystemAdminAuthContext(tenantID uuid.UUID) *middleware.AuthContext {
	return &middleware.AuthContext{
		TenantID:      tenantID,
		UserID:        uuid.New(),
		Email:         "sysadmin@example.com",
		UserTenants:   []uuid.UUID{tenantID},
		IsSystemAdmin: true,
	}
}

func withURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func withURLParams(req *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func withAuth(req *http.Request, authCtx *middleware.AuthContext) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), "auth", authCtx))
}

func newReadinessAPIServer(t *testing.T, namespace string, taskExists map[string]bool, pipelineExists map[string]bool, secretExists bool) *httptest.Server {
	t.Helper()

	nodeList := corev1.NodeList{
		Items: []corev1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
					},
				},
			},
		},
	}

	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/version":
			_, _ = w.Write([]byte(`{"major":"1","minor":"29","gitVersion":"v1.29.0","gitCommit":"x","gitTreeState":"clean","buildDate":"2026-01-01T00:00:00Z","goVersion":"go1.22.0","compiler":"gc","platform":"linux/amd64"}`))
			return

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/nodes":
			_ = json.NewEncoder(w).Encode(nodeList)
			return

		case r.Method == http.MethodGet && r.URL.Path == "/apis/tekton.dev/v1/namespaces/"+namespace+"/pipelineruns":
			_, _ = w.Write([]byte(`{"apiVersion":"tekton.dev/v1","kind":"PipelineRunList","items":[]}`))
			return

		case r.Method == http.MethodGet && r.URL.Path == "/apis/tekton.dev/v1":
			_, _ = w.Write([]byte(`{"groupVersion":"tekton.dev/v1","resources":[{"name":"tasks","namespaced":true},{"name":"pipelines","namespaced":true},{"name":"pipelineruns","namespaced":true}]}`))
			return

		case r.Method == http.MethodGet && r.URL.Path == "/apis/tekton.dev/v1beta1":
			_, _ = w.Write([]byte(`{"groupVersion":"tekton.dev/v1beta1","resources":[{"name":"tasks","namespaced":true},{"name":"pipelines","namespaced":true},{"name":"pipelineruns","namespaced":true}]}`))
			return

		case r.Method == http.MethodGet && r.URL.Path == "/apis/tekton.dev/v1/namespaces/"+namespace+"/tasks":
			_, _ = w.Write([]byte(`{"apiVersion":"tekton.dev/v1","kind":"TaskList","items":[]}`))
			return

		case r.Method == http.MethodGet && r.URL.Path == "/apis/tekton.dev/v1beta1/namespaces/"+namespace+"/tasks":
			_, _ = w.Write([]byte(`{"apiVersion":"tekton.dev/v1beta1","kind":"TaskList","items":[]}`))
			return

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/apis/tekton.dev/v1/namespaces/"+namespace+"/tasks/"):
			taskName := strings.TrimPrefix(r.URL.Path, "/apis/tekton.dev/v1/namespaces/"+namespace+"/tasks/")
			if taskExists[taskName] {
				_, _ = w.Write([]byte(`{"apiVersion":"tekton.dev/v1","kind":"Task","metadata":{"name":"` + taskName + `","namespace":"` + namespace + `"}}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","message":"not found","reason":"NotFound","code":404}`))
			return

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/apis/tekton.dev/v1/namespaces/"+namespace+"/pipelines/"):
			pipelineName := strings.TrimPrefix(r.URL.Path, "/apis/tekton.dev/v1/namespaces/"+namespace+"/pipelines/")
			if pipelineExists[pipelineName] {
				_, _ = w.Write([]byte(`{"apiVersion":"tekton.dev/v1","kind":"Pipeline","metadata":{"name":"` + pipelineName + `","namespace":"` + namespace + `"}}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","message":"not found","reason":"NotFound","code":404}`))
			return

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/namespaces/"+namespace+"/secrets/docker-config":
			if secretExists {
				_, _ = w.Write([]byte(`{"apiVersion":"v1","kind":"Secret","metadata":{"name":"docker-config","namespace":"` + namespace + `"}}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","message":"not found","reason":"NotFound","code":404}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","message":"unhandled test endpoint","reason":"NotFound","code":404}`))
	}))
}

func TestGetProviderReadiness_MissingTaskIsReported(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	namespace := "image-factory-" + tenantID.String()[:8]
	server := newReadinessAPIServer(t, namespace, map[string]bool{
		"git-clone":      false,
		"docker-build":   true,
		"buildx":         true,
		"kaniko-no-push": true,
		"scan-image":     true,
		"generate-sbom":  true,
		"push-image":     true,
		"packer":         true,
	}, nil, true)
	defer server.Close()

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:       providerID,
			TenantID: tenantID,
			Config: map[string]interface{}{
				"tekton_enabled": true,
				"runtime_auth": map[string]interface{}{
					"auth_method": "token",
					"apiServer":   server.URL,
					"token":       "test-token",
				},
			},
			Status:      infrastructure.ProviderStatusOnline,
			Name:        "k8s-1",
			DisplayName: "K8s 1",
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/readiness", nil)
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.GetProviderReadiness(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response ProviderReadinessResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Status != "not_ready" {
		t.Fatalf("expected status not_ready, got %q", response.Status)
	}

	found := false
	for _, item := range response.MissingPrereqs {
		if strings.Contains(item, "missing tekton task: git-clone") && strings.Contains(item, namespace) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing prereq to include git-clone task in namespace %s, got %+v", namespace, response.MissingPrereqs)
	}
}

func TestGetProviderReadiness_MissingDockerConfigSecretIsReported(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	namespace := "image-factory-" + tenantID.String()[:8]
	server := newReadinessAPIServer(t, namespace, map[string]bool{
		"git-clone":      true,
		"docker-build":   true,
		"buildx":         true,
		"kaniko-no-push": true,
		"scan-image":     true,
		"generate-sbom":  true,
		"push-image":     true,
		"packer":         true,
	}, nil, false)
	defer server.Close()

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:       providerID,
			TenantID: tenantID,
			Config: map[string]interface{}{
				"tekton_enabled": true,
				"runtime_auth": map[string]interface{}{
					"auth_method": "token",
					"apiServer":   server.URL,
					"token":       "test-token",
				},
			},
			Status:      infrastructure.ProviderStatusOnline,
			Name:        "k8s-1",
			DisplayName: "K8s 1",
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/readiness", nil)
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.GetProviderReadiness(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response ProviderReadinessResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Status != "not_ready" {
		t.Fatalf("expected status not_ready, got %q", response.Status)
	}

	found := false
	for _, item := range response.MissingPrereqs {
		if strings.Contains(item, "missing required secret docker-config") && strings.Contains(item, namespace) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing prereq to include docker-config secret in namespace %s, got %+v", namespace, response.MissingPrereqs)
	}
}

func TestGetProviderReadiness_PipelineRefModeMissingPipelineIsReported(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	namespace := "image-factory-" + tenantID.String()[:8]
	server := newReadinessAPIServer(t, namespace, map[string]bool{
		"git-clone":      true,
		"docker-build":   true,
		"buildx":         true,
		"kaniko-no-push": true,
		"scan-image":     true,
		"generate-sbom":  true,
		"push-image":     true,
		"packer":         true,
	}, map[string]bool{
		"image-factory-build-v1-docker": false,
		"image-factory-build-v1-buildx": true,
		"image-factory-build-v1-kaniko": true,
		"image-factory-build-v1-packer": true,
	}, true)
	defer server.Close()

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:       providerID,
			TenantID: tenantID,
			Config: map[string]interface{}{
				"tekton_enabled": true,
				"runtime_auth": map[string]interface{}{
					"auth_method": "token",
					"apiServer":   server.URL,
					"token":       "test-token",
				},
				"tekton_use_pipeline_ref": true,
				"tekton_profile_version":  "v1",
			},
			Status:      infrastructure.ProviderStatusOnline,
			Name:        "k8s-1",
			DisplayName: "K8s 1",
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/readiness", nil)
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.GetProviderReadiness(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response ProviderReadinessResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Status != "not_ready" {
		t.Fatalf("expected status not_ready, got %q", response.Status)
	}

	found := false
	for _, item := range response.MissingPrereqs {
		if strings.Contains(item, "missing tekton pipeline: image-factory-build-v1-docker") && strings.Contains(item, namespace) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing prereq to include missing pipeline in namespace %s, got %+v", namespace, response.MissingPrereqs)
	}
}

func TestGetProviderReadiness_AllPrereqsPresentReturnsReady(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	namespace := "image-factory-" + tenantID.String()[:8]
	server := newReadinessAPIServer(t, namespace, map[string]bool{
		"git-clone":      true,
		"docker-build":   true,
		"buildx":         true,
		"kaniko-no-push": true,
		"scan-image":     true,
		"generate-sbom":  true,
		"push-image":     true,
		"packer":         true,
	}, map[string]bool{
		"image-factory-build-v1-docker": true,
		"image-factory-build-v1-buildx": true,
		"image-factory-build-v1-kaniko": true,
		"image-factory-build-v1-packer": true,
	}, true)
	defer server.Close()

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:       providerID,
			TenantID: tenantID,
			Config: map[string]interface{}{
				"tekton_enabled": true,
				"runtime_auth": map[string]interface{}{
					"auth_method": "token",
					"apiServer":   server.URL,
					"token":       "test-token",
				},
				"tekton_use_pipeline_ref": true,
				"tekton_profile_version":  "v1",
			},
			Status:      infrastructure.ProviderStatusOnline,
			Name:        "k8s-1",
			DisplayName: "K8s 1",
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/readiness", nil)
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.GetProviderReadiness(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response ProviderReadinessResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Status != "ready" {
		t.Fatalf("expected status ready, got %q", response.Status)
	}
	if len(response.MissingPrereqs) != 0 {
		t.Fatalf("expected no missing prereqs, got %+v", response.MissingPrereqs)
	}
}

func TestGetProviderQuarantineDispatchReadiness_Ready(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	readinessStatus := "ready"

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:              providerID,
			TenantID:        tenantID,
			ProviderType:    infrastructure.ProviderTypeKubernetes,
			Status:          infrastructure.ProviderStatusOnline,
			Name:            "k8s-dispatch",
			DisplayName:     "K8s Dispatch",
			ReadinessStatus: &readinessStatus,
			IsSchedulable:   true,
			Config: map[string]interface{}{
				"tekton_enabled":              true,
				"quarantine_dispatch_enabled": true,
				"runtime_auth": map[string]interface{}{
					"auth_method": "token",
					"apiServer":   "https://kubernetes.example.test",
					"token":       "test-token",
				},
			},
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/quarantine-dispatch-readiness", nil)
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.GetProviderQuarantineDispatchReadiness(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response ProviderQuarantineDispatchReadinessResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Status != "ready" || !response.DispatchReady {
		t.Fatalf("expected ready dispatch response, got status=%q dispatch_ready=%v", response.Status, response.DispatchReady)
	}
	if !response.TektonEnabled || !response.QuarantineDispatchEnabled {
		t.Fatalf("expected both dispatch flags true, got tekton=%v quarantine_dispatch=%v", response.TektonEnabled, response.QuarantineDispatchEnabled)
	}
	if len(response.MissingPrereqs) != 0 {
		t.Fatalf("expected no missing prereqs, got %+v", response.MissingPrereqs)
	}
}

func TestGetProviderQuarantineDispatchReadiness_NotReadyIncludesReasons(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	readinessStatus := "not_ready"
	schedulableReason := "provider blocked by maintenance policy"

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:                      providerID,
			TenantID:                tenantID,
			ProviderType:            infrastructure.ProviderTypeBuildNodes,
			Status:                  infrastructure.ProviderStatusMaintenance,
			Name:                    "build-nodes-1",
			DisplayName:             "Build Nodes 1",
			ReadinessStatus:         &readinessStatus,
			ReadinessMissingPrereqs: []string{"tekton tasks missing"},
			IsSchedulable:           false,
			SchedulableReason:       &schedulableReason,
			Config: map[string]interface{}{
				"tekton_enabled": false,
			},
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/quarantine-dispatch-readiness", nil)
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.GetProviderQuarantineDispatchReadiness(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response ProviderQuarantineDispatchReadinessResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Status != "not_ready" || response.DispatchReady {
		t.Fatalf("expected not_ready dispatch response, got status=%q dispatch_ready=%v", response.Status, response.DispatchReady)
	}
	if len(response.MissingPrereqs) == 0 {
		t.Fatalf("expected missing prereqs to include dispatch blockers")
	}

	expectedSubstrings := []string{
		"provider type",
		"must be online",
		"tekton_enabled=false",
		"missing quarantine_dispatch_enabled=true",
		"runtime auth config invalid",
		"provider readiness status is not_ready",
		"provider blocked by maintenance policy",
	}
	for _, expected := range expectedSubstrings {
		found := false
		for _, item := range response.MissingPrereqs {
			if strings.Contains(item, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected missing prereqs to include %q, got %+v", expected, response.MissingPrereqs)
		}
	}
}

func TestPrepareProvider_RunInProgressReturnsConflict(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	activeRunID := uuid.New()

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:           providerID,
			TenantID:     tenantID,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Config:       map[string]interface{}{"tekton_enabled": true},
			Status:       infrastructure.ProviderStatusOnline,
			Name:         "k8s-1",
			DisplayName:  "K8s 1",
		},
		activePrepareRun: &infrastructure.ProviderPrepareRun{
			ID:         activeRunID,
			ProviderID: providerID,
			TenantID:   tenantID,
			Status:     infrastructure.ProviderPrepareRunStatusRunning,
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/prepare", strings.NewReader(`{}`))
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.PrepareProvider(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPrepareProvider_DifferentTenantReturnsForbidden(t *testing.T) {
	providerTenantID := uuid.New()
	authTenantID := uuid.New()
	providerID := uuid.New()

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:           providerID,
			TenantID:     providerTenantID,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Config:       map[string]interface{}{"tekton_enabled": true},
			Status:       infrastructure.ProviderStatusOnline,
			Name:         "k8s-1",
			DisplayName:  "K8s 1",
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/prepare", strings.NewReader(`{}`))
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessTestAuthContext(authTenantID))
	rec := httptest.NewRecorder()

	handler.PrepareProvider(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPrepareProvider_NotFoundReturnsNotFound(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()

	repo := &readinessRepoStub{}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/prepare", strings.NewReader(`{}`))
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.PrepareProvider(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetProviderPrepareStatus_ReturnsActiveRunAndChecks(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	runID := uuid.New()
	checkID := uuid.New()
	now := time.Now().UTC()

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:           providerID,
			TenantID:     tenantID,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Config:       map[string]interface{}{"tekton_enabled": true},
			Status:       infrastructure.ProviderStatusOnline,
			Name:         "k8s-1",
			DisplayName:  "K8s 1",
		},
		activePrepareRun: &infrastructure.ProviderPrepareRun{
			ID:               runID,
			ProviderID:       providerID,
			TenantID:         tenantID,
			RequestedBy:      uuid.New(),
			Status:           infrastructure.ProviderPrepareRunStatusRunning,
			RequestedActions: map[string]interface{}{"bootstrap": true},
			ResultSummary:    map[string]interface{}{},
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		prepareChecks: []*infrastructure.ProviderPrepareRunCheck{
			{
				ID:       checkID,
				RunID:    runID,
				CheckKey: "connectivity.kubernetes_api",
				Category: "connectivity",
				Severity: "error",
				OK:       false,
				Message:  "kubernetes api unreachable",
				Details: map[string]interface{}{
					"remediation":          "verify kube-api endpoint and credentials",
					"remediation_commands": []string{"kubectl cluster-info", "kubectl auth can-i get nodes"},
				},
				CreatedAt: now,
			},
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/prepare/status", nil)
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.GetProviderPrepareStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var response ProviderPrepareStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.ProviderID != providerID.String() {
		t.Fatalf("expected provider_id %s, got %s", providerID, response.ProviderID)
	}
	if response.ActiveRun == nil || response.ActiveRun.ID != runID.String() {
		t.Fatalf("expected active run %s, got %+v", runID, response.ActiveRun)
	}
	if len(response.Checks) != 1 || response.Checks[0].ID != checkID.String() {
		t.Fatalf("expected one check %s, got %+v", checkID, response.Checks)
	}
	details := response.Checks[0].Details
	if details == nil {
		t.Fatalf("expected remediation details in check payload")
	}
	if remediation, ok := details["remediation"].(string); !ok || remediation == "" {
		t.Fatalf("expected remediation string in details, got %+v", details["remediation"])
	}
	commandsRaw, ok := details["remediation_commands"].([]interface{})
	if !ok || len(commandsRaw) != 2 {
		t.Fatalf("expected remediation_commands array of size 2, got %+v", details["remediation_commands"])
	}
}

func TestListProviderPrepareRuns_ReturnsRuns(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	runID := uuid.New()
	now := time.Now().UTC()

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:           providerID,
			TenantID:     tenantID,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Config:       map[string]interface{}{"tekton_enabled": true},
			Status:       infrastructure.ProviderStatusOnline,
			Name:         "k8s-1",
			DisplayName:  "K8s 1",
		},
		prepareRuns: []*infrastructure.ProviderPrepareRun{
			{
				ID:               runID,
				ProviderID:       providerID,
				TenantID:         tenantID,
				RequestedBy:      uuid.New(),
				Status:           infrastructure.ProviderPrepareRunStatusSucceeded,
				RequestedActions: map[string]interface{}{"bootstrap": true},
				ResultSummary:    map[string]interface{}{"stage": "done"},
				CreatedAt:        now,
				UpdatedAt:        now,
			},
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/prepare/runs?limit=10&offset=0", nil)
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.ListProviderPrepareRuns(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Runs []ProviderPrepareRunResponse `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload.Runs) != 1 || payload.Runs[0].ID != runID.String() {
		t.Fatalf("expected one run %s, got %+v", runID, payload.Runs)
	}
	if payload.Runs[0].Status != string(infrastructure.ProviderPrepareRunStatusSucceeded) {
		t.Fatalf("expected run status succeeded, got %q", payload.Runs[0].Status)
	}
	if payload.Runs[0].RequestedActions == nil || payload.Runs[0].RequestedActions["bootstrap"] != true {
		t.Fatalf("expected requested_actions.bootstrap=true, got %+v", payload.Runs[0].RequestedActions)
	}
	if payload.Runs[0].ResultSummary == nil || payload.Runs[0].ResultSummary["stage"] != "done" {
		t.Fatalf("expected result_summary.stage=done, got %+v", payload.Runs[0].ResultSummary)
	}
	if payload.Runs[0].CreatedAt == "" || payload.Runs[0].UpdatedAt == "" {
		t.Fatalf("expected created_at and updated_at to be present, got created_at=%q updated_at=%q", payload.Runs[0].CreatedAt, payload.Runs[0].UpdatedAt)
	}
}

func TestListProviderPrepareRuns_InvalidLimitReturnsBadRequest(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:           providerID,
			TenantID:     tenantID,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Config:       map[string]interface{}{"tekton_enabled": true},
			Status:       infrastructure.ProviderStatusOnline,
			Name:         "k8s-1",
			DisplayName:  "K8s 1",
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/prepare/runs?limit=0", nil)
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.ListProviderPrepareRuns(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetProviderPrepareStatus_NotConfiguredReturnsNotImplemented(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()

	repo := &repoWithoutPrepareStub{
		provider: &infrastructure.Provider{
			ID:           providerID,
			TenantID:     tenantID,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Config:       map[string]interface{}{"tekton_enabled": true},
			Status:       infrastructure.ProviderStatusOnline,
			Name:         "k8s-1",
			DisplayName:  "K8s 1",
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/prepare/status", nil)
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.GetProviderPrepareStatus(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetProviderPrepareSummaries_ReturnsLatestStatus(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	runID := uuid.New()
	errMessage := "bootstrap failed: missing permissions"
	checkID := uuid.New()
	now := time.Now().UTC()

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:           providerID,
			TenantID:     tenantID,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Config:       map[string]interface{}{"tekton_enabled": true},
			Status:       infrastructure.ProviderStatusOnline,
			Name:         "k8s-1",
			DisplayName:  "K8s 1",
		},
		prepareRuns: []*infrastructure.ProviderPrepareRun{
			{
				ID:               runID,
				ProviderID:       providerID,
				TenantID:         tenantID,
				RequestedBy:      uuid.New(),
				Status:           infrastructure.ProviderPrepareRunStatusFailed,
				RequestedActions: map[string]interface{}{"bootstrap": true},
				ResultSummary:    map[string]interface{}{"stage": "bootstrap"},
				ErrorMessage:     &errMessage,
				CreatedAt:        now,
				UpdatedAt:        now,
			},
		},
		prepareChecks: []*infrastructure.ProviderPrepareRunCheck{
			{
				ID:       checkID,
				RunID:    runID,
				CheckKey: "bootstrap.apply",
				Category: "bootstrap",
				Severity: "error",
				OK:       false,
				Message:  "apply failed",
				Details: map[string]interface{}{
					"remediation": "grant patch/update permissions for Tekton resources",
				},
				CreatedAt: now,
			},
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/prepare/summary?provider_ids="+providerID.String(), nil)
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.GetProviderPrepareSummaries(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Summaries []ProviderPrepareSummaryResponse `json:"summaries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload.Summaries) != 1 {
		t.Fatalf("expected 1 summary, got %+v", payload.Summaries)
	}
	if payload.Summaries[0].ProviderID != providerID.String() {
		t.Fatalf("expected provider_id=%s got %s", providerID.String(), payload.Summaries[0].ProviderID)
	}
	if payload.Summaries[0].Status == nil || *payload.Summaries[0].Status != string(infrastructure.ProviderPrepareRunStatusFailed) {
		t.Fatalf("expected failed status, got %+v", payload.Summaries[0].Status)
	}
	if payload.Summaries[0].Error == nil || *payload.Summaries[0].Error != errMessage {
		t.Fatalf("expected error_message=%q got %+v", errMessage, payload.Summaries[0].Error)
	}
	if payload.Summaries[0].LatestCheckCategory == nil || *payload.Summaries[0].LatestCheckCategory != "bootstrap" {
		t.Fatalf("expected latest_prepare_check_category=bootstrap got %+v", payload.Summaries[0].LatestCheckCategory)
	}
	if payload.Summaries[0].LatestCheckSeverity == nil || *payload.Summaries[0].LatestCheckSeverity != "error" {
		t.Fatalf("expected latest_prepare_check_severity=error got %+v", payload.Summaries[0].LatestCheckSeverity)
	}
	if payload.Summaries[0].LatestRemediationHint == nil || *payload.Summaries[0].LatestRemediationHint == "" {
		t.Fatalf("expected latest_prepare_remediation_hint to be present, got %+v", payload.Summaries[0].LatestRemediationHint)
	}
}

func TestGetProviderPrepareSummaries_RequiresProviderIDs(t *testing.T) {
	tenantID := uuid.New()
	repo := &readinessRepoStub{}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/prepare/summary", nil)
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.GetProviderPrepareSummaries(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetProviderPrepareSummaries_MixedTenantDeniedWithoutAllTenantScope(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	providerA := uuid.New()
	providerB := uuid.New()

	repo := &readinessRepoStub{
		providersByID: map[uuid.UUID]*infrastructure.Provider{
			providerA: {
				ID:           providerA,
				TenantID:     tenantA,
				ProviderType: infrastructure.ProviderTypeKubernetes,
				Status:       infrastructure.ProviderStatusOnline,
			},
			providerB: {
				ID:           providerB,
				TenantID:     tenantB,
				ProviderType: infrastructure.ProviderTypeKubernetes,
				Status:       infrastructure.ProviderStatusOnline,
			},
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/prepare/summary?provider_ids="+providerA.String()+","+providerB.String(), nil)
	req = withAuth(req, readinessTestAuthContext(tenantA))
	rec := httptest.NewRecorder()

	handler.GetProviderPrepareSummaries(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetProviderPrepareSummaries_SystemAdminAllTenantsAllowsMixedTenant(t *testing.T) {
	tenantAdmin := uuid.New()
	tenantA := uuid.New()
	tenantB := uuid.New()
	providerA := uuid.New()
	providerB := uuid.New()
	runA := uuid.New()
	now := time.Now().UTC()

	repo := &readinessRepoStub{
		providersByID: map[uuid.UUID]*infrastructure.Provider{
			providerA: {
				ID:           providerA,
				TenantID:     tenantA,
				ProviderType: infrastructure.ProviderTypeKubernetes,
				Status:       infrastructure.ProviderStatusOnline,
			},
			providerB: {
				ID:           providerB,
				TenantID:     tenantB,
				ProviderType: infrastructure.ProviderTypeKubernetes,
				Status:       infrastructure.ProviderStatusOnline,
			},
		},
		prepareRuns: []*infrastructure.ProviderPrepareRun{
			{
				ID:          runA,
				ProviderID:  providerA,
				TenantID:    tenantA,
				RequestedBy: uuid.New(),
				Status:      infrastructure.ProviderPrepareRunStatusSucceeded,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/prepare/summary?all_tenants=true&provider_ids="+providerA.String()+","+providerB.String(), nil)
	req = withAuth(req, readinessSystemAdminAuthContext(tenantAdmin))
	rec := httptest.NewRecorder()

	handler.GetProviderPrepareSummaries(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Summaries []ProviderPrepareSummaryResponse `json:"summaries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload.Summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %+v", payload.Summaries)
	}
}

func TestGetProviderPrepareSummaries_IncludeBatchMetrics(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:           providerID,
			TenantID:     tenantID,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Config:       map[string]interface{}{"tekton_enabled": true},
			Status:       infrastructure.ProviderStatusOnline,
			Name:         "k8s-metrics",
			DisplayName:  "K8s Metrics",
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/prepare/summary?provider_ids="+providerID.String()+"&include_batch_metrics=true", nil)
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.GetProviderPrepareSummaries(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Summaries    []ProviderPrepareSummaryResponse            `json:"summaries"`
		BatchMetrics *ProviderPrepareSummaryBatchMetricsResponse `json:"batch_metrics"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload.Summaries) != 1 {
		t.Fatalf("expected 1 summary, got %+v", payload.Summaries)
	}
	if payload.BatchMetrics == nil {
		t.Fatalf("expected batch_metrics to be present, got nil")
	}
	if payload.BatchMetrics.BatchCount < 1 {
		t.Fatalf("expected batch_count >= 1, got %d", payload.BatchMetrics.BatchCount)
	}
	if payload.BatchMetrics.ProvidersTotal < 1 {
		t.Fatalf("expected providers_total >= 1, got %d", payload.BatchMetrics.ProvidersTotal)
	}
	if payload.BatchMetrics.FallbackBatches < 1 {
		t.Fatalf("expected fallback_batches >= 1, got %d", payload.BatchMetrics.FallbackBatches)
	}
}

func TestListProviders_EmbedsLatestPrepareSummary(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	runID := uuid.New()
	checkID := uuid.New()
	now := time.Now().UTC()
	errMessage := "tekton apply failed"

	repo := &readinessRepoStub{
		providersList: []infrastructure.Provider{
			{
				ID:           providerID,
				TenantID:     tenantID,
				ProviderType: infrastructure.ProviderTypeKubernetes,
				Status:       infrastructure.ProviderStatusOnline,
				Name:         "k8s-1",
				DisplayName:  "K8s 1",
				Config:       map[string]interface{}{"tekton_enabled": true},
			},
		},
		prepareRuns: []*infrastructure.ProviderPrepareRun{
			{
				ID:           runID,
				ProviderID:   providerID,
				TenantID:     tenantID,
				RequestedBy:  uuid.New(),
				Status:       infrastructure.ProviderPrepareRunStatusFailed,
				ErrorMessage: &errMessage,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		},
		prepareChecks: []*infrastructure.ProviderPrepareRunCheck{
			{
				ID:       checkID,
				RunID:    runID,
				CheckKey: "permission.namespaces.create",
				Category: "permission_audit",
				Severity: "warn",
				OK:       false,
				Message:  "missing namespaces create",
				Details: map[string]interface{}{
					"remediation": "grant create/get/list on namespaces for managed bootstrap",
				},
				CreatedAt: now,
			},
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers", nil)
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.ListProviders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload PaginatedProvidersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("expected 1 provider, got %+v", payload.Data)
	}
	if payload.Data[0].LatestPrepareStatus == nil || *payload.Data[0].LatestPrepareStatus != string(infrastructure.ProviderPrepareRunStatusFailed) {
		t.Fatalf("expected latest_prepare_status=failed got %+v", payload.Data[0].LatestPrepareStatus)
	}
	if payload.Data[0].LatestPrepareError == nil || *payload.Data[0].LatestPrepareError != errMessage {
		t.Fatalf("expected latest_prepare_error=%q got %+v", errMessage, payload.Data[0].LatestPrepareError)
	}
	if payload.Data[0].LatestPrepareCheckCategory == nil || *payload.Data[0].LatestPrepareCheckCategory != "permission_audit" {
		t.Fatalf("expected latest_prepare_check_category=permission_audit got %+v", payload.Data[0].LatestPrepareCheckCategory)
	}
	if payload.Data[0].LatestPrepareCheckSeverity == nil || *payload.Data[0].LatestPrepareCheckSeverity != "warn" {
		t.Fatalf("expected latest_prepare_check_severity=warn got %+v", payload.Data[0].LatestPrepareCheckSeverity)
	}
	if payload.Data[0].LatestPrepareRemediationHint == nil || *payload.Data[0].LatestPrepareRemediationHint == "" {
		t.Fatalf("expected latest_prepare_remediation_hint to be present, got %+v", payload.Data[0].LatestPrepareRemediationHint)
	}
}

func TestListProviders_CanDisablePrepareSummaryEnrichment(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	runID := uuid.New()
	now := time.Now().UTC()
	errMessage := "tekton apply failed"

	repo := &readinessRepoStub{
		providersList: []infrastructure.Provider{
			{
				ID:           providerID,
				TenantID:     tenantID,
				ProviderType: infrastructure.ProviderTypeKubernetes,
				Status:       infrastructure.ProviderStatusOnline,
				Name:         "k8s-1",
				DisplayName:  "K8s 1",
				Config:       map[string]interface{}{"tekton_enabled": true},
			},
		},
		prepareRuns: []*infrastructure.ProviderPrepareRun{
			{
				ID:           runID,
				ProviderID:   providerID,
				TenantID:     tenantID,
				RequestedBy:  uuid.New(),
				Status:       infrastructure.ProviderPrepareRunStatusFailed,
				ErrorMessage: &errMessage,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers?include_prepare_summary=false", nil)
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.ListProviders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload PaginatedProvidersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("expected 1 provider, got %+v", payload.Data)
	}
	if payload.Data[0].LatestPrepareStatus != nil {
		t.Fatalf("expected latest_prepare_status to be omitted when enrichment disabled, got %+v", payload.Data[0].LatestPrepareStatus)
	}
	if payload.Data[0].LatestPrepareError != nil {
		t.Fatalf("expected latest_prepare_error to be omitted when enrichment disabled, got %+v", payload.Data[0].LatestPrepareError)
	}
}

func TestListProviders_PrepareSummaryFlagScalesOnLargeProviderSet(t *testing.T) {
	tenantID := uuid.New()
	providers := make([]infrastructure.Provider, 0, 120)
	for i := 0; i < 120; i++ {
		providers = append(providers, infrastructure.Provider{
			ID:           uuid.New(),
			TenantID:     tenantID,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Status:       infrastructure.ProviderStatusOnline,
			Name:         "k8s-" + uuid.NewString()[:8],
			DisplayName:  "K8s Provider",
			Config:       map[string]interface{}{"tekton_enabled": true},
		})
	}

	repo := &readinessRepoStub{
		providersList: providers,
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	reqNoSummary := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers?include_prepare_summary=false", nil)
	reqNoSummary = withAuth(reqNoSummary, readinessTestAuthContext(tenantID))
	recNoSummary := httptest.NewRecorder()
	handler.ListProviders(recNoSummary, reqNoSummary)
	if recNoSummary.Code != http.StatusOK {
		t.Fatalf("expected 200 with summary disabled, got %d body=%s", recNoSummary.Code, recNoSummary.Body.String())
	}
	if repo.prepareRunListCalls != 0 || repo.prepareCheckListCalls != 0 {
		t.Fatalf("expected zero prepare repository calls when summary disabled, got run_calls=%d check_calls=%d", repo.prepareRunListCalls, repo.prepareCheckListCalls)
	}

	reqWithSummary := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers?include_prepare_summary=true", nil)
	reqWithSummary = withAuth(reqWithSummary, readinessTestAuthContext(tenantID))
	recWithSummary := httptest.NewRecorder()
	handler.ListProviders(recWithSummary, reqWithSummary)
	if recWithSummary.Code != http.StatusOK {
		t.Fatalf("expected 200 with summary enabled, got %d body=%s", recWithSummary.Code, recWithSummary.Body.String())
	}
	if repo.prepareRunListCalls != len(providers) {
		t.Fatalf("expected one prepare run lookup per provider when enabled, got %d expected %d", repo.prepareRunListCalls, len(providers))
	}
}

func TestListProviders_PrepareSummaryCapSkipsEnrichmentOnVeryLargeList(t *testing.T) {
	tenantID := uuid.New()
	providers := make([]infrastructure.Provider, 0, 550)
	for i := 0; i < 550; i++ {
		providers = append(providers, infrastructure.Provider{
			ID:           uuid.New(),
			TenantID:     tenantID,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Status:       infrastructure.ProviderStatusOnline,
			Name:         "k8s-" + uuid.NewString()[:8],
			DisplayName:  "K8s Provider",
			Config:       map[string]interface{}{"tekton_enabled": true},
		})
	}

	repo := &readinessRepoStub{providersList: providers}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers?include_prepare_summary=true", nil)
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()
	handler.ListProviders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	if repo.prepareRunListCalls != 0 || repo.prepareCheckListCalls != 0 {
		t.Fatalf("expected zero prepare enrichment calls due to cap, got run_calls=%d check_calls=%d", repo.prepareRunListCalls, repo.prepareCheckListCalls)
	}
}

func TestListProviders_IncludesBlockedByGates(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	reason := "provider status is offline"
	repo := &readinessRepoStub{
		providersList: []infrastructure.Provider{
			{
				ID:                providerID,
				TenantID:          tenantID,
				ProviderType:      infrastructure.ProviderTypeKubernetes,
				Status:            infrastructure.ProviderStatusOffline,
				Name:              "k8s-gated",
				DisplayName:       "K8s Gated",
				Config:            map[string]interface{}{"tekton_enabled": true},
				IsSchedulable:     false,
				SchedulableReason: &reason,
				BlockedBy:         []string{"provider_not_ready", "provider_status_offline"},
			},
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers", nil)
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.ListProviders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Data []struct {
			ID        string   `json:"id"`
			BlockedBy []string `json:"blocked_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("expected one provider row, got %+v", payload.Data)
	}
	if payload.Data[0].ID != providerID.String() {
		t.Fatalf("expected provider id %s, got %s", providerID.String(), payload.Data[0].ID)
	}
	if len(payload.Data[0].BlockedBy) != 2 {
		t.Fatalf("expected 2 blocked_by entries, got %+v", payload.Data[0].BlockedBy)
	}
	if payload.Data[0].BlockedBy[0] != "provider_not_ready" || payload.Data[0].BlockedBy[1] != "provider_status_offline" {
		t.Fatalf("unexpected blocked_by entries: %+v", payload.Data[0].BlockedBy)
	}
}

func TestStreamTenantNamespacePrepareStatus_InvalidTenantID(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:           providerID,
			TenantID:     tenantID,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Status:       infrastructure.ProviderStatusOnline,
			Name:         "k8s",
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/tenants/not-a-uuid/prepare-namespace/stream", nil)
	req = withURLParams(req, map[string]string{
		"id":        providerID.String(),
		"tenant_id": "not-a-uuid",
	})
	req = withAuth(req, readinessSystemAdminAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.StreamTenantNamespacePrepareStatus(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestStreamTenantNamespacePrepareStatus_RequiresSystemAdmin(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:           providerID,
			TenantID:     tenantID,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Status:       infrastructure.ProviderStatusOnline,
			Name:         "k8s",
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/tenants/"+tenantID.String()+"/prepare-namespace/stream", nil)
	req = withURLParams(req, map[string]string{
		"id":        providerID.String(),
		"tenant_id": tenantID.String(),
	})
	req = withAuth(req, readinessTestAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.StreamTenantNamespacePrepareStatus(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetTenantNamespacePrepareStatus_IncludesAssetDriftFields(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	desired := "sha256:desired"
	installed := "sha256:installed"
	now := time.Now().UTC()

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:           providerID,
			TenantID:     tenantID,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Status:       infrastructure.ProviderStatusOnline,
			Name:         "k8s",
		},
		tenantPrepares: map[string]*infrastructure.ProviderTenantNamespacePrepare{
			providerID.String() + ":" + tenantID.String(): {
				ID:                    uuid.New(),
				ProviderID:            providerID,
				TenantID:              tenantID,
				Namespace:             "image-factory-" + tenantID.String()[:8],
				Status:                infrastructure.ProviderTenantNamespacePrepareSucceeded,
				DesiredAssetVersion:   &desired,
				InstalledAssetVersion: &installed,
				AssetDriftStatus:      infrastructure.TenantAssetDriftStatusStale,
				CreatedAt:             now,
				UpdatedAt:             now,
			},
		},
	}

	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/tenants/"+tenantID.String()+"/prepare-namespace/status", nil)
	req = withURLParams(req, map[string]string{
		"id":        providerID.String(),
		"tenant_id": tenantID.String(),
	})
	req = withAuth(req, readinessSystemAdminAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.GetTenantNamespacePrepareStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Prepare struct {
			DesiredAssetVersion   *string `json:"desired_asset_version"`
			InstalledAssetVersion *string `json:"installed_asset_version"`
			AssetDriftStatus      string  `json:"asset_drift_status"`
		} `json:"prepare"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Prepare.DesiredAssetVersion == nil || *payload.Prepare.DesiredAssetVersion != desired {
		t.Fatalf("expected desired_asset_version=%q, got %#v", desired, payload.Prepare.DesiredAssetVersion)
	}
	if payload.Prepare.InstalledAssetVersion == nil || *payload.Prepare.InstalledAssetVersion != installed {
		t.Fatalf("expected installed_asset_version=%q, got %#v", installed, payload.Prepare.InstalledAssetVersion)
	}
	if payload.Prepare.AssetDriftStatus == "" {
		t.Fatalf("expected non-empty asset_drift_status")
	}
}

func TestGetTenantNamespacePrepareStatus_DefaultsAssetDriftStatusToUnknown(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	now := time.Now().UTC()

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:           providerID,
			TenantID:     tenantID,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Status:       infrastructure.ProviderStatusOnline,
			Name:         "k8s",
		},
		tenantPrepares: map[string]*infrastructure.ProviderTenantNamespacePrepare{
			providerID.String() + ":" + tenantID.String(): {
				ID:               uuid.New(),
				ProviderID:       providerID,
				TenantID:         tenantID,
				Namespace:        "image-factory-" + tenantID.String()[:8],
				Status:           infrastructure.ProviderTenantNamespacePrepareSucceeded,
				AssetDriftStatus: "",
				CreatedAt:        now,
				UpdatedAt:        now,
			},
		},
	}

	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/tenants/"+tenantID.String()+"/prepare-namespace/status", nil)
	req = withURLParams(req, map[string]string{
		"id":        providerID.String(),
		"tenant_id": tenantID.String(),
	})
	req = withAuth(req, readinessSystemAdminAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.GetTenantNamespacePrepareStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Prepare struct {
			AssetDriftStatus string `json:"asset_drift_status"`
		} `json:"prepare"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Prepare.AssetDriftStatus != string(infrastructure.TenantAssetDriftStatusUnknown) {
		t.Fatalf("expected asset_drift_status=unknown, got %q", payload.Prepare.AssetDriftStatus)
	}
}

func TestReconcileSelectedTenantNamespaces_InvalidRequestBody(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:           providerID,
			TenantID:     tenantID,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Status:       infrastructure.ProviderStatusOnline,
			Name:         "k8s",
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/tenants/reconcile-selected", strings.NewReader(`{"tenant_ids":["bad-uuid"]}`))
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessSystemAdminAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.ReconcileSelectedTenantNamespaces(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestReconcileSelectedTenantNamespaces_ReturnsConflictSummaryWhenFailuresOccur(t *testing.T) {
	tenantID := uuid.New()
	targetTenantID := uuid.New()
	providerID := uuid.New()

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:            providerID,
			TenantID:      tenantID,
			ProviderType:  infrastructure.ProviderTypeKubernetes,
			Status:        infrastructure.ProviderStatusOnline,
			Name:          "k8s",
			BootstrapMode: "image_factory_managed",
			Config:        map[string]interface{}{},
		},
		tenantPrepares: map[string]*infrastructure.ProviderTenantNamespacePrepare{},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	reqBody := strings.NewReader(`{"tenant_ids":["` + targetTenantID.String() + `"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/tenants/reconcile-selected", reqBody)
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessSystemAdminAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.ReconcileSelectedTenantNamespaces(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Summary struct {
			Mode     string `json:"mode"`
			Targeted int    `json:"targeted"`
			Failed   int    `json:"failed"`
		} `json:"summary"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Summary.Mode != "selected" {
		t.Fatalf("expected mode=selected got %q", payload.Summary.Mode)
	}
	if payload.Summary.Targeted != 1 {
		t.Fatalf("expected targeted=1 got %d", payload.Summary.Targeted)
	}
	if payload.Summary.Failed != 1 {
		t.Fatalf("expected failed=1 got %d", payload.Summary.Failed)
	}
	if strings.TrimSpace(payload.Error) == "" {
		t.Fatalf("expected error message in conflict response")
	}
}

func TestReconcileStaleTenantNamespaces_NoStaleTargetsReturnsOK(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()

	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:            providerID,
			TenantID:      tenantID,
			ProviderType:  infrastructure.ProviderTypeKubernetes,
			Status:        infrastructure.ProviderStatusOnline,
			Name:          "k8s",
			BootstrapMode: "image_factory_managed",
			Config:        map[string]interface{}{},
		},
		tenantPrepares: map[string]*infrastructure.ProviderTenantNamespacePrepare{
			providerID.String() + ":" + tenantID.String(): {
				ID:               uuid.New(),
				ProviderID:       providerID,
				TenantID:         tenantID,
				Namespace:        "image-factory-" + tenantID.String()[:8],
				Status:           infrastructure.ProviderTenantNamespacePrepareSucceeded,
				AssetDriftStatus: infrastructure.TenantAssetDriftStatusUnknown,
				CreatedAt:        time.Now().UTC(),
				UpdatedAt:        time.Now().UTC(),
			},
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/tenants/reconcile-stale", nil)
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessSystemAdminAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.ReconcileStaleTenantNamespaces(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Summary struct {
			Mode     string `json:"mode"`
			Targeted int    `json:"targeted"`
			Applied  int    `json:"applied"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Summary.Mode != "stale_only" {
		t.Fatalf("expected mode=stale_only got %q", payload.Summary.Mode)
	}
	if payload.Summary.Targeted != 0 {
		t.Fatalf("expected targeted=0 got %d", payload.Summary.Targeted)
	}
	if payload.Summary.Applied != 0 {
		t.Fatalf("expected applied=0 got %d", payload.Summary.Applied)
	}
}

func TestGrantProviderPermission_SystemAdminAutoTriggersTenantPrepareAsync(t *testing.T) {
	tenantID := uuid.New()
	targetTenantID := uuid.New()
	providerID := uuid.New()
	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:            providerID,
			TenantID:      tenantID,
			ProviderType:  infrastructure.ProviderTypeKubernetes,
			Status:        infrastructure.ProviderStatusOnline,
			Name:          "k8s",
			BootstrapMode: "image_factory_managed",
			Config: map[string]interface{}{
				"tekton_enabled": true,
			},
		},
		tenantPrepares: make(map[string]*infrastructure.ProviderTenantNamespacePrepare),
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	reqBody := strings.NewReader(`{"tenant_id":"` + targetTenantID.String() + `","permission":"infrastructure:select"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/infrastructure/providers/"+providerID.String()+"/permissions", reqBody)
	req = withURLParam(req, "id", providerID.String())
	req = withAuth(req, readinessSystemAdminAuthContext(tenantID))
	rec := httptest.NewRecorder()

	handler.GrantProviderPermission(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rec.Code, rec.Body.String())
	}

	deadline := time.Now().Add(2 * time.Second)
	key := providerID.String() + ":" + targetTenantID.String()
	for time.Now().Before(deadline) {
		if prepare, ok := repo.tenantPrepares[key]; ok && prepare != nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("expected async tenant namespace prepare row for provider=%s tenant=%s", providerID, targetTenantID)
}

func TestStreamTenantNamespacePrepareStatus_WebSocketPayloadContract(t *testing.T) {
	tenantID := uuid.New()
	providerID := uuid.New()
	namespacePrepare := &infrastructure.ProviderTenantNamespacePrepare{
		ID:         uuid.New(),
		ProviderID: providerID,
		TenantID:   tenantID,
		Namespace:  "image-factory-" + tenantID.String()[:8],
		Status:     infrastructure.ProviderTenantNamespacePrepareRunning,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	repo := &readinessRepoStub{
		provider: &infrastructure.Provider{
			ID:            providerID,
			TenantID:      tenantID,
			ProviderType:  infrastructure.ProviderTypeKubernetes,
			Status:        infrastructure.ProviderStatusOnline,
			Name:          "k8s",
			BootstrapMode: "image_factory_managed",
			Config: map[string]interface{}{
				"tekton_enabled": true,
			},
		},
		tenantPrepares: map[string]*infrastructure.ProviderTenantNamespacePrepare{
			providerID.String() + ":" + tenantID.String(): namespacePrepare,
		},
	}
	svc := infrastructure.NewService(repo, nil, zap.NewNop())
	handler := NewInfrastructureProviderHandler(svc, zap.NewNop())

	router := chi.NewRouter()
	router.Get("/api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/prepare-namespace/stream", func(w http.ResponseWriter, r *http.Request) {
		reqWithAuth := withAuth(r, readinessSystemAdminAuthContext(tenantID))
		handler.StreamTenantNamespacePrepareStatus(w, reqWithAuth)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}
	wsURL.Scheme = "ws"
	wsURL.Path = "/api/v1/admin/infrastructure/providers/" + providerID.String() + "/tenants/" + tenantID.String() + "/prepare-namespace/stream"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		t.Fatalf("failed to connect websocket: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read websocket message: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("failed to parse websocket payload: %v", err)
	}
	if payload["type"] != "tenant_namespace_prepare_status" {
		t.Fatalf("expected payload type tenant_namespace_prepare_status, got %v", payload["type"])
	}
	if payload["provider_id"] != providerID.String() {
		t.Fatalf("expected provider_id %s, got %v", providerID.String(), payload["provider_id"])
	}
	if payload["tenant_id"] != tenantID.String() {
		t.Fatalf("expected tenant_id %s, got %v", tenantID.String(), payload["tenant_id"])
	}
	prepareObj, ok := payload["prepare"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected prepare object, got %#v", payload["prepare"])
	}
	if prepareObj["status"] != string(infrastructure.ProviderTenantNamespacePrepareRunning) {
		t.Fatalf("expected prepare status running, got %v", prepareObj["status"])
	}
}
