package rest

import (
	"os"
	"strings"
	"testing"
)

func TestRouter_SetupRoutes_DelegatesToRouteModules(t *testing.T) {
	content := readFile(t, "router.go")

	required := []string{
		"initializePlatformWiring(",
		"configureImageCatalogSubscriber(",
		"initializeSSOWiring(",
		"initializeRepositoryAuthWiring(",
		"setupTenantRoutes(",
		"setupUserRoutes(",
		"registerCoreAPIRoutes(",
		"registerIdentitySystemAdminRoutes(",
		"registerSecurityCatalogRepoRoutes(",
	}
	for _, snippet := range required {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected SetupRoutes to delegate via %q", snippet)
		}
	}
}

func TestRouter_CoreModule_OwnsBuildProjectAndInfraRoutes(t *testing.T) {
	content := readFile(t, "router_route_registration.go")

	required := []string{
		`router.Post("/api/v1/builds",`,
		`router.Get("/api/v1/projects",`,
		`router.Get("/api/notifications/events",`,
		`router.Get("/api/v1/admin/infrastructure/providers",`,
		`router.Get("/api/v1/auth/login-options",`,
	}
	for _, snippet := range required {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected core route module snippet not found: %s", snippet)
		}
	}
}

func TestRouter_ExtendedModule_OwnsAdminSecurityCatalogAndRepoAuthRoutes(t *testing.T) {
	content := readFile(t, "router_route_registration_extended.go")

	required := []string{
		`router.Get("/api/v1/profile",`,
		`router.Get("/api/v1/admin/settings/tools",`,
		`router.Get("/api/v1/audit-events",`,
		`router.Post("/api/v1/sso/saml/providers",`,
		`router.Get("/api/v1/images/{id}",`,
		`router.Post("/api/v1/repository-auth",`,
	}
	for _, snippet := range required {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected extended route module snippet not found: %s", snippet)
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(bytes)
}
