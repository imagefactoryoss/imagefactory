package rest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRouter_AdminAndSystemRoutes_AreChecklistTrackedAndPermissionProtected(t *testing.T) {
	routerContent := readRouterRegistrationSources(t)

	checklistPath := resolveRepoPath(t, "docs", "implementation", "ADMIN_TENANT_SCOPE_ROUTE_CHECKLIST.md")
	checklistBytes, err := os.ReadFile(checklistPath)
	if err != nil {
		t.Fatalf("failed to read checklist %s: %v", checklistPath, err)
	}
	checklistContent := string(checklistBytes)

	// Capture route registration lines in router registration sources.
	lineExpr := regexp.MustCompile(`router\.(Get|Post|Put|Delete|Patch)\("(/api/v1/(admin[^"]*|system-configs[^"]*|audit-events[^"]*))"`)
	lines := strings.Split(routerContent, "\n")
	type routeLine struct {
		method string
		route  string
		line   string
	}
	var routeLines []routeLine
	for _, line := range lines {
		m := lineExpr.FindStringSubmatch(line)
		if len(m) == 0 {
			continue
		}
		routeLines = append(routeLines, routeLine{
			method: strings.ToUpper(m[1]),
			route:  m[2],
			line:   strings.TrimSpace(line),
		})
	}

	if len(routeLines) == 0 {
		t.Fatal("expected admin/system routes in router registration sources, found none")
	}

	skipChecklist := map[string]bool{
		"/api/v1/system-configs/{id}":             true, // covered by system-configs section conventions
		"/api/v1/system-configs/activate/{id}":    true,
		"/api/v1/system-configs/deactivate/{id}":  true,
		"/api/v1/admin/infrastructure/providers/{id}/permissions": true, // verb-paired row in checklist
		"/api/v1/admin/external-services/{name}":  true, // verb-paired row in checklist
		"/api/v1/admin/ldap/{id}":                 true, // verb-paired row in checklist
		"/api/v1/audit-events/{id}":               true, // paired with list route
	}

	for _, reg := range routeLines {
		method := reg.method
		route := reg.route
		regLine := reg.line

		// Enforce route-level permission middleware (no bare authenticate) for admin/system/audit routes.
		if !strings.Contains(regLine, "RequirePermission(") {
			t.Fatalf("route %s %s is missing route-level RequirePermission middleware: %s", method, route, regLine)
		}

		if skipChecklist[route] {
			continue
		}
		checklistEntry := "`" + method + " " + route + "`"
		if !strings.Contains(checklistContent, checklistEntry) {
			t.Fatalf("route %s %s is not tracked in ADMIN_TENANT_SCOPE_ROUTE_CHECKLIST.md", method, route)
		}
	}
}

func readRouterRegistrationSources(t *testing.T) string {
	t.Helper()
	paths := []string{
		"router.go",
		"router_route_registration.go",
		"router_route_registration_extended.go",
	}

	var content strings.Builder
	for _, path := range paths {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		content.Write(bytes)
		content.WriteString("\n")
	}
	return content.String()
}

func resolveRepoPath(t *testing.T, parts ...string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	dir := wd
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, ".git")
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Join(append([]string{dir}, parts...)...)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("failed to locate repository root from %s", wd)
	return ""
}
