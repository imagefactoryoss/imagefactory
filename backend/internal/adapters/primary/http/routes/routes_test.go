package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/srikarm/image-factory/internal/domain/repositoryauth"
	"go.uber.org/zap"
)

func TestImageRoutesRegistersPaths(t *testing.T) {
	r := chi.NewRouter()
	ImageRoutes(r, nil, zap.NewNop())

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/images/search", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for registered path with wrong method, got %d", rr.Code)
	}
}

func TestInfrastructureRoutesRegistersPaths(t *testing.T) {
	r := chi.NewRouter()
	InfrastructureRoutes(r, nil, nil, nil, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/builds/infrastructure-recommendation", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for registered path with wrong method, got %d", rr.Code)
	}
}

func TestRepositoryAuthRoutesRegistersPaths(t *testing.T) {
	r := chi.NewRouter()
	var service repositoryauth.ServiceInterface
	RepositoryAuthRoutes(r, service, nil, zap.NewNop())

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/p1/repository-auth", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for registered path with wrong method, got %d", rr.Code)
	}
}
