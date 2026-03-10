package rest

import "testing"

func TestNormalizeRoleTypeFromName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "simple", input: "Viewer", expected: "viewer"},
		{name: "spaces", input: "Security   Reviewer", expected: "security_reviewer"},
		{name: "trim", input: "  System Administrator Viewer  ", expected: "system_administrator_viewer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeRoleTypeFromName(tt.input)
			if got != tt.expected {
				t.Fatalf("normalizeRoleTypeFromName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestShouldExposeRoleForTenant(t *testing.T) {
	allowed := map[string]struct{}{
		"owner":     {},
		"developer": {},
		"viewer":    {},
	}

	tests := []struct {
		name     string
		roleName string
		allowed  map[string]struct{}
		want     bool
	}{
		{name: "allowed role", roleName: "Owner", allowed: allowed, want: true},
		{name: "disallowed role", roleName: "Security Reviewer", allowed: allowed, want: false},
		{name: "exclude system admin", roleName: "System Administrator", allowed: allowed, want: false},
		{name: "exclude system admin viewer", roleName: "System Administrator Viewer", allowed: allowed, want: false},
		{name: "empty allowlist", roleName: "Owner", allowed: map[string]struct{}{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldExposeRoleForTenant(tt.roleName, tt.allowed)
			if got != tt.want {
				t.Fatalf("shouldExposeRoleForTenant(%q) = %v, want %v", tt.roleName, got, tt.want)
			}
		})
	}
}
