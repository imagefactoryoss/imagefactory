package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	appbuild "github.com/srikarm/image-factory/internal/application/build"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/infrastructure"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
)

// Helper to create a test logger
func createTestLogger() *zap.Logger {
	logger, _ := zap.NewProduction()
	return logger
}

// Helper to create test auth context
func createTestAuthContext() *middleware.AuthContext {
	tenantID := uuid.New()
	userID := uuid.New()
	return &middleware.AuthContext{
		TenantID:    tenantID,
		UserID:      userID,
		Email:       "test@example.com",
		UserTenants: []uuid.UUID{tenantID},
	}
}

// Helper to create a request with auth context
func createRequestWithAuth(method, path string, body interface{}, authCtx *middleware.AuthContext) *http.Request {
	var bodyReader *bytes.Buffer
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(bodyBytes)
	} else {
		bodyReader = bytes.NewBuffer([]byte{})
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	// Add auth context to request context
	if authCtx != nil {
		ctx := context.WithValue(req.Context(), "auth", authCtx)
		req = req.WithContext(ctx)
	}

	return req
}

// TestAuthContextExtraction tests that auth context can be properly extracted from requests
func TestAuthContextExtraction(t *testing.T) {
	authCtx := createTestAuthContext()
	require.NotNil(t, authCtx)

	req := createRequestWithAuth("GET", "/test", nil, authCtx)
	require.NotNil(t, req)

	// Extract the context back
	retrieved, ok := middleware.GetAuthContext(req)
	require.True(t, ok)
	assert.Equal(t, authCtx.UserID, retrieved.UserID)
	assert.Equal(t, authCtx.TenantID, retrieved.TenantID)
	assert.Equal(t, authCtx.Email, retrieved.Email)
}

// TestURLVarExtraction tests that URL variables can be extracted from chi routes
func TestURLVarExtraction(t *testing.T) {
	buildID := uuid.New()
	req := httptest.NewRequest("GET", "/api/v1/builds/"+buildID.String(), nil)

	// Set URL parameter using chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", buildID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Extract using chi.URLParam
	extractedID := chi.URLParam(req, "id")
	assert.Equal(t, buildID.String(), extractedID)
}

// TestRequestValidation tests request validation for JSON payloads
func TestRequestValidation_ValidJSON(t *testing.T) {
	payload := map[string]interface{}{
		"project_id": uuid.New().String(),
		"git_branch": "main",
	}

	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/v1/builds", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	var decoded map[string]interface{}
	err := json.NewDecoder(req.Body).Decode(&decoded)
	assert.NoError(t, err)
	assert.NotNil(t, decoded["project_id"])
}

func TestRequestValidation_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/builds", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	var decoded map[string]interface{}
	err := json.NewDecoder(req.Body).Decode(&decoded)
	assert.Error(t, err)
}

// TestBuildIDValidation tests UUID validation logic
func TestBuildIDValidation_Valid(t *testing.T) {
	validID := uuid.New()
	_, err := uuid.Parse(validID.String())
	assert.NoError(t, err)
}

func TestBuildIDValidation_Invalid(t *testing.T) {
	_, err := uuid.Parse("invalid-uuid")
	assert.Error(t, err)
}

// TestResponseFormat tests that responses are properly formatted as JSON
func TestResponseFormat_JSON(t *testing.T) {
	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]string{
			"id": uuid.New().String(),
		},
	}

	json.NewEncoder(w).Encode(response)

	// Verify it can be decoded back
	var decoded map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&decoded)
	assert.NoError(t, err)
	assert.Equal(t, "success", decoded["status"])
}

// TestHTTPStatusCodes tests common HTTP status code scenarios
func TestHTTPStatusCodes_Success(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		expectedCode int
		description  string
	}{
		{"OK", http.StatusOK, http.StatusOK, "200 OK"},
		{"Created", http.StatusCreated, http.StatusCreated, "201 Created"},
		{"BadRequest", http.StatusBadRequest, http.StatusBadRequest, "400 Bad Request"},
		{"Unauthorized", http.StatusUnauthorized, http.StatusUnauthorized, "401 Unauthorized"},
		{"NotFound", http.StatusNotFound, http.StatusNotFound, "404 Not Found"},
		{"Conflict", http.StatusConflict, http.StatusConflict, "409 Conflict"},
		{"InternalServerError", http.StatusInternalServerError, http.StatusInternalServerError, "500 Internal Server Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			w.WriteHeader(tt.status)
			assert.Equal(t, tt.expectedCode, w.Code, tt.description)
		})
	}
}

// TestChiRouting tests chi path variable handling
func TestChiRouting_PathVariables(t *testing.T) {
	router := chi.NewRouter()
	buildID := uuid.New()
	projectID := uuid.New()

	// Register route with multiple path variables
	router.Get("/api/v1/projects/{projectId}/builds/{buildId}", func(w http.ResponseWriter, r *http.Request) {
		projectId := chi.URLParam(r, "projectId")
		buildId := chi.URLParam(r, "buildId")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"projectId": projectId,
			"buildId":   buildId,
		})
	})

	// Make request
	req := httptest.NewRequest("GET", "/api/v1/projects/"+projectID.String()+"/builds/"+buildID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify response
	var vars map[string]string
	json.NewDecoder(w.Body).Decode(&vars)
	assert.Equal(t, projectID.String(), vars["projectId"])
	assert.Equal(t, buildID.String(), vars["buildId"])
}

// TestErrorHandling_InvalidContentType tests handling of invalid content types
func TestErrorHandling_InvalidContentType(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/builds", bytes.NewBufferString("some data"))
	req.Header.Set("Content-Type", "text/plain")

	assert.Equal(t, "text/plain", req.Header.Get("Content-Type"))
}

// TestContextPropagation tests that context values are properly propagated
func TestContextPropagation(t *testing.T) {
	authCtx := createTestAuthContext()
	baseReq := createRequestWithAuth("GET", "/test", nil, authCtx)

	// Verify context is propagated
	retrieved, ok := middleware.GetAuthContext(baseReq)
	assert.True(t, ok)
	assert.Equal(t, authCtx.UserID, retrieved.UserID)

	// Create new request from context
	newReq := baseReq.WithContext(baseReq.Context())
	retrieved2, ok2 := middleware.GetAuthContext(newReq)
	assert.True(t, ok2)
	assert.Equal(t, authCtx.UserID, retrieved2.UserID)
}

// Benchmark tests for common operations
func BenchmarkAuthContextCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = createTestAuthContext()
	}
}

func BenchmarkRequestCreation(b *testing.B) {
	authCtx := createTestAuthContext()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = createRequestWithAuth("GET", "/api/v1/builds", nil, authCtx)
	}
}

func BenchmarkJSONEncoding(b *testing.B) {
	payload := map[string]interface{}{
		"id":    uuid.New(),
		"name":  "test",
		"email": "test@example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(payload)
	}
}

type stubInfraService struct {
	providers      []*infrastructure.Provider
	err            error
	prepareStatus  *infrastructure.ProviderTenantNamespacePrepare
	prepareErr     error
	returnNilReady bool
}

func (s *stubInfraService) GetAvailableProviders(ctx context.Context, tenantID uuid.UUID) ([]*infrastructure.Provider, error) {
	return s.providers, s.err
}

func (s *stubInfraService) GetTenantNamespacePrepareStatus(ctx context.Context, providerID, tenantID uuid.UUID) (*infrastructure.ProviderTenantNamespacePrepare, error) {
	if s.prepareErr != nil {
		return nil, s.prepareErr
	}
	if s.returnNilReady {
		return nil, nil
	}
	if s.prepareStatus != nil {
		return s.prepareStatus, nil
	}
	ns := "image-factory-" + tenantID.String()[:8]
	return &infrastructure.ProviderTenantNamespacePrepare{
		ID:         uuid.New(),
		ProviderID: providerID,
		TenantID:   tenantID,
		Namespace:  ns,
		Status:     infrastructure.ProviderTenantNamespacePrepareSucceeded,
	}, nil
}

type stubDomainBuildServiceForHandlerTests struct {
	createErr    error
	createdBuild *build.Build
}

type stubQuarantineAdmissionChecker struct {
	allow        bool
	err          error
	releaseState string
}

func (s *stubQuarantineAdmissionChecker) IsArtifactRefReleased(ctx context.Context, tenantID uuid.UUID, imageRef string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.allow, nil
}

func (s *stubQuarantineAdmissionChecker) GetArtifactReleaseStateByRef(ctx context.Context, tenantID uuid.UUID, imageRef string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.releaseState, nil
}

func (s *stubDomainBuildServiceForHandlerTests) CreateBuild(ctx context.Context, tenantID, projectID uuid.UUID, manifest build.BuildManifest, actorID *uuid.UUID) (*build.Build, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	if s.createdBuild != nil {
		return s.createdBuild, nil
	}
	return nil, build.ErrBuildNotFound
}

func (s *stubDomainBuildServiceForHandlerTests) RetryBuild(ctx context.Context, buildID uuid.UUID) error {
	return build.ErrBuildNotFound
}

func (s *stubDomainBuildServiceForHandlerTests) GetBuild(ctx context.Context, id uuid.UUID) (*build.Build, error) {
	return nil, build.ErrBuildNotFound
}

func TestCreateBuild_InfrastructureProviderRequired(t *testing.T) {
	logger := zap.NewNop()
	handler := NewBuildHandler(nil, appbuild.NewService(&stubDomainBuildServiceForHandlerTests{}, logger), nil, nil, nil, nil, logger)

	tenantID := uuid.New()
	projectID := uuid.New()
	reqBody := CreateBuildRequest{
		TenantID:  tenantID.String(),
		ProjectID: projectID.String(),
		Manifest: build.BuildManifest{
			Name:               "Build",
			Type:               build.BuildTypeKaniko,
			InfrastructureType: "kubernetes",
			BuildConfig: &build.BuildConfig{
				BuildType:         build.BuildTypeKaniko,
				SBOMTool:          build.SBOMToolSyft,
				ScanTool:          build.ScanToolTrivy,
				RegistryType:      build.RegistryTypeHarbor,
				SecretManagerType: build.SecretManagerVault,
				Dockerfile:        "FROM alpine",
				BuildContext:      ".",
				RegistryRepo:      "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
			},
		},
	}

	authCtx := createTestAuthContext()
	req := createRequestWithAuth("POST", "/api/v1/builds", reqBody, authCtx)
	w := httptest.NewRecorder()

	handler.CreateBuild(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateBuild_InfrastructureProviderUnauthorized(t *testing.T) {
	logger := zap.NewNop()
	providerID := uuid.New()
	handler := NewBuildHandler(nil, appbuild.NewService(&stubDomainBuildServiceForHandlerTests{}, logger), nil, nil, nil, &stubInfraService{providers: []*infrastructure.Provider{}}, logger)

	tenantID := uuid.New()
	projectID := uuid.New()
	reqBody := CreateBuildRequest{
		TenantID:  tenantID.String(),
		ProjectID: projectID.String(),
		Manifest: build.BuildManifest{
			Name:                     "Build",
			Type:                     build.BuildTypeKaniko,
			InfrastructureType:       "kubernetes",
			InfrastructureProviderID: &providerID,
			BuildConfig: &build.BuildConfig{
				BuildType:         build.BuildTypeKaniko,
				SBOMTool:          build.SBOMToolSyft,
				ScanTool:          build.ScanToolTrivy,
				RegistryType:      build.RegistryTypeHarbor,
				SecretManagerType: build.SecretManagerVault,
				Dockerfile:        "FROM alpine",
				BuildContext:      ".",
				RegistryRepo:      "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
			},
		},
	}

	authCtx := createTestAuthContext()
	req := createRequestWithAuth("POST", "/api/v1/builds", reqBody, authCtx)
	w := httptest.NewRecorder()

	handler.CreateBuild(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCreateBuild_KubernetesProviderTektonDisabled_FailsFast(t *testing.T) {
	logger := zap.NewNop()
	providerID := uuid.New()
	tenantID := uuid.New()

	handler := NewBuildHandler(nil, appbuild.NewService(&stubDomainBuildServiceForHandlerTests{}, logger), nil, nil, nil, &stubInfraService{providers: []*infrastructure.Provider{
		{
			ID:           providerID,
			TenantID:     tenantID,
			IsGlobal:     true,
			ProviderType: infrastructure.ProviderTypeKubernetes,
			Name:         "k8s-provider",
			Config: map[string]interface{}{
				"tekton_enabled": false,
				"runtime_auth": map[string]interface{}{
					"auth_method": "token",
					"endpoint":    "https://k8s.example",
					"token":       "token",
				},
			},
		},
	}}, logger)

	projectID := uuid.New()
	reqBody := CreateBuildRequest{
		TenantID:  tenantID.String(),
		ProjectID: projectID.String(),
		Manifest: build.BuildManifest{
			Name:                     "Build",
			Type:                     build.BuildTypeKaniko,
			InfrastructureType:       "kubernetes",
			InfrastructureProviderID: &providerID,
			BuildConfig: &build.BuildConfig{
				BuildType:         build.BuildTypeKaniko,
				SBOMTool:          build.SBOMToolSyft,
				ScanTool:          build.ScanToolTrivy,
				RegistryType:      build.RegistryTypeHarbor,
				SecretManagerType: build.SecretManagerVault,
				Dockerfile:        "FROM alpine",
				BuildContext:      ".",
				RegistryRepo:      "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
			},
		},
	}

	authCtx := createTestAuthContext()
	req := createRequestWithAuth("POST", "/api/v1/builds", reqBody, authCtx)
	w := httptest.NewRecorder()

	handler.CreateBuild(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "tekton_enabled=true")
}

func TestCreateBuild_KubernetesManagedProviderTenantNamespaceNotPrepared_ReturnsStructuredConflict(t *testing.T) {
	logger := zap.NewNop()
	providerID := uuid.New()
	tenantID := uuid.New()

	handler := NewBuildHandler(nil, appbuild.NewService(&stubDomainBuildServiceForHandlerTests{}, logger), nil, nil, nil, &stubInfraService{
		providers: []*infrastructure.Provider{
			{
				ID:            providerID,
				TenantID:      tenantID,
				IsGlobal:      true,
				ProviderType:  infrastructure.ProviderTypeKubernetes,
				Name:          "k8s-provider",
				BootstrapMode: "image_factory_managed",
				Config: map[string]interface{}{
					"tekton_enabled": true,
					"runtime_auth": map[string]interface{}{
						"auth_method": "token",
						"endpoint":    "https://k8s.example",
						"token":       "token",
					},
				},
			},
		},
		returnNilReady: true,
	}, logger)

	projectID := uuid.New()
	reqBody := CreateBuildRequest{
		TenantID:  tenantID.String(),
		ProjectID: projectID.String(),
		Manifest: build.BuildManifest{
			Name:                     "Build",
			Type:                     build.BuildTypeKaniko,
			InfrastructureType:       "kubernetes",
			InfrastructureProviderID: &providerID,
			BuildConfig: &build.BuildConfig{
				BuildType:         build.BuildTypeKaniko,
				SBOMTool:          build.SBOMToolSyft,
				ScanTool:          build.ScanToolTrivy,
				RegistryType:      build.RegistryTypeHarbor,
				SecretManagerType: build.SecretManagerVault,
				Dockerfile:        "FROM alpine",
				BuildContext:      ".",
				RegistryRepo:      "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
			},
		},
	}

	authCtx := createTestAuthContext()
	req := createRequestWithAuth("POST", "/api/v1/builds", reqBody, authCtx)
	w := httptest.NewRecorder()

	handler.CreateBuild(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	assert.Equal(t, "tenant_namespace_not_prepared", payload["code"])
	assert.NotEmpty(t, payload["error"])

	details, ok := payload["details"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, providerID.String(), details["provider_id"])
	assert.Equal(t, tenantID.String(), details["tenant_id"])
	assert.Equal(t, "missing", details["prepare_status"])
}

func TestCreateBuild_ValidationErrorFromAppService_ReturnsBadRequest(t *testing.T) {
	logger := zap.NewNop()
	stubDomain := &stubDomainBuildServiceForHandlerTests{
		createErr: errors.New("build name is required"),
	}
	handler := NewBuildHandler(nil, appbuild.NewService(stubDomain, logger), nil, nil, nil, nil, logger)

	tenantID := uuid.New()
	projectID := uuid.New()
	reqBody := CreateBuildRequest{
		TenantID:  tenantID.String(),
		ProjectID: projectID.String(),
		Manifest: build.BuildManifest{
			Name:         "Build",
			Type:         build.BuildTypeContainer,
			BaseImage:    "alpine:3.19",
			Instructions: []string{"RUN echo ok"},
		},
	}

	authCtx := createTestAuthContext()
	req := createRequestWithAuth("POST", "/api/v1/builds", reqBody, authCtx)
	w := httptest.NewRecorder()

	handler.CreateBuild(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateBuild_ToolAvailabilityDenied_ReturnsBadRequest(t *testing.T) {
	logger := zap.NewNop()
	providerID := uuid.New()
	tenantID := uuid.New()
	stubDomain := &stubDomainBuildServiceForHandlerTests{
		createErr: errors.New("kaniko build method is not available for this tenant"),
	}
	handler := NewBuildHandler(nil, appbuild.NewService(stubDomain, logger), nil, nil, nil, &stubInfraService{providers: []*infrastructure.Provider{
		{
			ID:            providerID,
			TenantID:      tenantID,
			IsGlobal:      true,
			ProviderType:  infrastructure.ProviderTypeKubernetes,
			Name:          "k8s-provider",
			BootstrapMode: "image_factory_managed",
			Config: map[string]interface{}{
				"tekton_enabled": true,
				"runtime_auth": map[string]interface{}{
					"auth_method": "token",
					"endpoint":    "https://k8s.example",
					"token":       "token",
				},
			},
		},
	}}, logger)

	projectID := uuid.New()
	reqBody := CreateBuildRequest{
		TenantID:  tenantID.String(),
		ProjectID: projectID.String(),
		Manifest: build.BuildManifest{
			Name:                     "Build",
			Type:                     build.BuildTypeKaniko,
			InfrastructureType:       "kubernetes",
			InfrastructureProviderID: &providerID,
			BuildConfig: &build.BuildConfig{
				BuildType:         build.BuildTypeKaniko,
				SBOMTool:          build.SBOMToolSyft,
				ScanTool:          build.ScanToolTrivy,
				RegistryType:      build.RegistryTypeHarbor,
				SecretManagerType: build.SecretManagerVault,
				Dockerfile:        "FROM alpine",
				BuildContext:      ".",
				RegistryRepo:      "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
			},
		},
	}

	authCtx := createTestAuthContext()
	req := createRequestWithAuth("POST", "/api/v1/builds", reqBody, authCtx)
	w := httptest.NewRecorder()

	handler.CreateBuild(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "not available for this tenant")
}

func TestCreateBuild_Success_ReturnsCreatedResponse(t *testing.T) {
	logger := zap.NewNop()
	providerID := uuid.New()
	tenantID := uuid.New()
	projectID := uuid.New()
	manifest := build.BuildManifest{
		Name:                     "Build Success",
		Type:                     build.BuildTypeKaniko,
		InfrastructureType:       "kubernetes",
		InfrastructureProviderID: &providerID,
		BuildConfig: &build.BuildConfig{
			BuildType:         build.BuildTypeKaniko,
			SBOMTool:          build.SBOMToolSyft,
			ScanTool:          build.ScanToolTrivy,
			RegistryType:      build.RegistryTypeHarbor,
			SecretManagerType: build.SecretManagerVault,
			Dockerfile:        "FROM alpine",
			BuildContext:      ".",
			RegistryRepo:      "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
		},
	}
	created, err := build.NewBuild(tenantID, projectID, manifest, nil)
	require.NoError(t, err)

	stubDomain := &stubDomainBuildServiceForHandlerTests{
		createdBuild: created,
	}
	handler := NewBuildHandler(nil, appbuild.NewService(stubDomain, logger), nil, nil, nil, &stubInfraService{providers: []*infrastructure.Provider{
		{
			ID:            providerID,
			TenantID:      tenantID,
			IsGlobal:      true,
			ProviderType:  infrastructure.ProviderTypeKubernetes,
			Name:          "k8s-provider",
			BootstrapMode: "image_factory_managed",
			Config: map[string]interface{}{
				"tekton_enabled": true,
				"runtime_auth": map[string]interface{}{
					"auth_method": "token",
					"endpoint":    "https://k8s.example",
					"token":       "token",
				},
			},
		},
	}}, logger)

	reqBody := CreateBuildRequest{
		TenantID:  tenantID.String(),
		ProjectID: projectID.String(),
		Manifest:  manifest,
	}

	authCtx := createTestAuthContext()
	req := createRequestWithAuth("POST", "/api/v1/builds", reqBody, authCtx)
	w := httptest.NewRecorder()

	handler.CreateBuild(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, created.ID().String(), resp["id"])
	assert.Equal(t, tenantID.String(), resp["tenant_id"])
	assert.Equal(t, "Build Success", resp["name"])
	assert.Equal(t, string(build.BuildTypeKaniko), resp["type"])
}

func TestCreateBuild_QuarantineBaseImageNotReleased_ReturnsConflict(t *testing.T) {
	logger := zap.NewNop()
	providerID := uuid.New()
	tenantID := uuid.New()
	projectID := uuid.New()
	handler := NewBuildHandler(nil, appbuild.NewService(&stubDomainBuildServiceForHandlerTests{}, logger), nil, nil, nil, &stubInfraService{providers: []*infrastructure.Provider{
		{
			ID:            providerID,
			TenantID:      tenantID,
			IsGlobal:      true,
			ProviderType:  infrastructure.ProviderTypeKubernetes,
			Name:          "k8s-provider",
			BootstrapMode: "image_factory_managed",
			Config: map[string]interface{}{
				"tekton_enabled": true,
				"runtime_auth": map[string]interface{}{
					"auth_method": "token",
					"endpoint":    "https://k8s.example",
					"token":       "token",
				},
			},
		},
	}}, logger)
	handler.SetQuarantineArtifactAdmissionChecker(&stubQuarantineAdmissionChecker{allow: false})

	reqBody := CreateBuildRequest{
		TenantID:  tenantID.String(),
		ProjectID: projectID.String(),
		Manifest: build.BuildManifest{
			Name:                     "Build Unreleased Quarantine Artifact",
			Type:                     build.BuildTypeKaniko,
			BaseImage:                "registry.local/quarantine/team-a/import-1@sha256:1111111111111111111111111111111111111111111111111111111111111111",
			InfrastructureType:       "kubernetes",
			InfrastructureProviderID: &providerID,
			BuildConfig: &build.BuildConfig{
				BuildType:         build.BuildTypeKaniko,
				SBOMTool:          build.SBOMToolSyft,
				ScanTool:          build.ScanToolTrivy,
				RegistryType:      build.RegistryTypeHarbor,
				SecretManagerType: build.SecretManagerVault,
				Dockerfile:        "FROM registry.local/quarantine/team-a/import-1@sha256:1111111111111111111111111111111111111111111111111111111111111111",
				BuildContext:      ".",
				RegistryRepo:      "registry.local/team-a/output:latest",
			},
		},
	}
	authCtx := createTestAuthContext()
	req := createRequestWithAuth("POST", "/api/v1/builds", reqBody, authCtx)
	w := httptest.NewRecorder()

	handler.CreateBuild(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	assert.Equal(t, "quarantine_artifact_not_released", payload["code"])
	details, ok := payload["details"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "registry.local/quarantine/team-a/import-1@sha256:1111111111111111111111111111111111111111111111111111111111111111", details["image_ref"])
}

func TestCreateBuild_QuarantineBaseImageReleased_Allowed(t *testing.T) {
	logger := zap.NewNop()
	providerID := uuid.New()
	tenantID := uuid.New()
	projectID := uuid.New()
	manifest := build.BuildManifest{
		Name:                     "Build Released Quarantine Artifact",
		Type:                     build.BuildTypeKaniko,
		BaseImage:                "registry.local/quarantine/team-a/import-2@sha256:2222222222222222222222222222222222222222222222222222222222222222",
		InfrastructureType:       "kubernetes",
		InfrastructureProviderID: &providerID,
		BuildConfig: &build.BuildConfig{
			BuildType:         build.BuildTypeKaniko,
			SBOMTool:          build.SBOMToolSyft,
			ScanTool:          build.ScanToolTrivy,
			RegistryType:      build.RegistryTypeHarbor,
			SecretManagerType: build.SecretManagerVault,
			Dockerfile:        "FROM registry.local/quarantine/team-a/import-2@sha256:2222222222222222222222222222222222222222222222222222222222222222",
			BuildContext:      ".",
			RegistryRepo:      "registry.local/team-a/output:latest",
		},
	}
	created, err := build.NewBuild(tenantID, projectID, manifest, nil)
	require.NoError(t, err)
	handler := NewBuildHandler(nil, appbuild.NewService(&stubDomainBuildServiceForHandlerTests{createdBuild: created}, logger), nil, nil, nil, &stubInfraService{providers: []*infrastructure.Provider{
		{
			ID:            providerID,
			TenantID:      tenantID,
			IsGlobal:      true,
			ProviderType:  infrastructure.ProviderTypeKubernetes,
			Name:          "k8s-provider",
			BootstrapMode: "image_factory_managed",
			Config: map[string]interface{}{
				"tekton_enabled": true,
				"runtime_auth": map[string]interface{}{
					"auth_method": "token",
					"endpoint":    "https://k8s.example",
					"token":       "token",
				},
			},
		},
	}}, logger)
	handler.SetQuarantineArtifactAdmissionChecker(&stubQuarantineAdmissionChecker{allow: true})

	reqBody := CreateBuildRequest{
		TenantID:  tenantID.String(),
		ProjectID: projectID.String(),
		Manifest:  manifest,
	}
	authCtx := createTestAuthContext()
	req := createRequestWithAuth("POST", "/api/v1/builds", reqBody, authCtx)
	w := httptest.NewRecorder()

	handler.CreateBuild(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateBuild_QuarantineBaseImageDigestNotPinned_ReturnsConflict(t *testing.T) {
	logger := zap.NewNop()
	providerID := uuid.New()
	tenantID := uuid.New()
	projectID := uuid.New()
	handler := NewBuildHandler(nil, appbuild.NewService(&stubDomainBuildServiceForHandlerTests{}, logger), nil, nil, nil, &stubInfraService{providers: []*infrastructure.Provider{
		{
			ID:            providerID,
			TenantID:      tenantID,
			IsGlobal:      true,
			ProviderType:  infrastructure.ProviderTypeKubernetes,
			Name:          "k8s-provider",
			BootstrapMode: "image_factory_managed",
			Config: map[string]interface{}{
				"tekton_enabled": true,
				"runtime_auth": map[string]interface{}{
					"auth_method": "token",
					"endpoint":    "https://k8s.example",
					"token":       "token",
				},
			},
		},
	}}, logger)
	handler.SetQuarantineArtifactAdmissionChecker(&stubQuarantineAdmissionChecker{allow: true})

	reqBody := CreateBuildRequest{
		TenantID:  tenantID.String(),
		ProjectID: projectID.String(),
		Manifest: build.BuildManifest{
			Name:                     "Build Quarantine Artifact Not Pinned",
			Type:                     build.BuildTypeKaniko,
			BaseImage:                "registry.local/quarantine/team-a/import-3:latest",
			InfrastructureType:       "kubernetes",
			InfrastructureProviderID: &providerID,
			BuildConfig: &build.BuildConfig{
				BuildType:         build.BuildTypeKaniko,
				SBOMTool:          build.SBOMToolSyft,
				ScanTool:          build.ScanToolTrivy,
				RegistryType:      build.RegistryTypeHarbor,
				SecretManagerType: build.SecretManagerVault,
				Dockerfile:        "FROM registry.local/quarantine/team-a/import-3:latest",
				BuildContext:      ".",
				RegistryRepo:      "registry.local/team-a/output:latest",
			},
		},
	}
	authCtx := createTestAuthContext()
	req := createRequestWithAuth("POST", "/api/v1/builds", reqBody, authCtx)
	w := httptest.NewRecorder()

	handler.CreateBuild(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	assert.Equal(t, "quarantine_artifact_digest_required", payload["code"])
}

func TestCreateBuild_QuarantineBaseImageWithdrawn_ReturnsConflict(t *testing.T) {
	logger := zap.NewNop()
	providerID := uuid.New()
	tenantID := uuid.New()
	projectID := uuid.New()
	handler := NewBuildHandler(nil, appbuild.NewService(&stubDomainBuildServiceForHandlerTests{}, logger), nil, nil, nil, &stubInfraService{providers: []*infrastructure.Provider{
		{
			ID:            providerID,
			TenantID:      tenantID,
			IsGlobal:      true,
			ProviderType:  infrastructure.ProviderTypeKubernetes,
			Name:          "k8s-provider",
			BootstrapMode: "image_factory_managed",
			Config: map[string]interface{}{
				"tekton_enabled": true,
				"runtime_auth": map[string]interface{}{
					"auth_method": "token",
					"endpoint":    "https://k8s.example",
					"token":       "token",
				},
			},
		},
	}}, logger)
	handler.SetQuarantineArtifactAdmissionChecker(&stubQuarantineAdmissionChecker{
		allow:        false,
		releaseState: "withdrawn",
	})

	reqBody := CreateBuildRequest{
		TenantID:  tenantID.String(),
		ProjectID: projectID.String(),
		Manifest: build.BuildManifest{
			Name:                     "Build Quarantine Artifact Withdrawn",
			Type:                     build.BuildTypeKaniko,
			BaseImage:                "registry.local/quarantine/team-a/import-4@sha256:4444444444444444444444444444444444444444444444444444444444444444",
			InfrastructureType:       "kubernetes",
			InfrastructureProviderID: &providerID,
			BuildConfig: &build.BuildConfig{
				BuildType:         build.BuildTypeKaniko,
				SBOMTool:          build.SBOMToolSyft,
				ScanTool:          build.ScanToolTrivy,
				RegistryType:      build.RegistryTypeHarbor,
				SecretManagerType: build.SecretManagerVault,
				Dockerfile:        "FROM registry.local/quarantine/team-a/import-4@sha256:4444444444444444444444444444444444444444444444444444444444444444",
				BuildContext:      ".",
				RegistryRepo:      "registry.local/team-a/output:latest",
			},
		},
	}
	authCtx := createTestAuthContext()
	req := createRequestWithAuth("POST", "/api/v1/builds", reqBody, authCtx)
	w := httptest.NewRecorder()

	handler.CreateBuild(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	assert.Equal(t, "quarantine_artifact_withdrawn", payload["code"])
	details, ok := payload["details"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "withdrawn", details["release_state"])
}

func TestIsDigestPinnedQuarantineRef(t *testing.T) {
	assert.True(t, isDigestPinnedQuarantineRef("registry.local/quarantine/team-a/import@sha256:abc123"))
	assert.True(t, isDigestPinnedQuarantineRef("registry.local/quarantine/team-a/import@SHA256:abc123"))
	assert.False(t, isDigestPinnedQuarantineRef("registry.local/quarantine/team-a/import:latest"))
	assert.False(t, isDigestPinnedQuarantineRef(""))
}

func BenchmarkUUIDParsing(b *testing.B) {
	id := uuid.New().String()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = uuid.Parse(id)
	}
}

func BenchmarkChiVariables(b *testing.B) {
	buildID := uuid.New().String()
	req := httptest.NewRequest("GET", "/api/v1/builds/"+buildID, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Set URL parameter using chi context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", buildID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Extract using chi.URLParam
		_ = chi.URLParam(req, "id")
	}
}
