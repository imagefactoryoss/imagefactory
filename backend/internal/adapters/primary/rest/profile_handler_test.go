package rest

import "testing"

func TestDeriveDefaultLandingRoute(t *testing.T) {
	cases := []struct {
		name                 string
		canAccessAdmin       bool
		hasSecurityReviewer  bool
		hasAnyTenantAccess   bool
		expectedLandingRoute string
	}{
		{
			name:                 "admin landing takes precedence",
			canAccessAdmin:       true,
			hasSecurityReviewer:  true,
			hasAnyTenantAccess:   true,
			expectedLandingRoute: "/admin/dashboard",
		},
		{
			name:                 "security reviewer landing",
			canAccessAdmin:       false,
			hasSecurityReviewer:  true,
			hasAnyTenantAccess:   true,
			expectedLandingRoute: "/reviewer/dashboard",
		},
		{
			name:                 "tenant landing",
			canAccessAdmin:       false,
			hasSecurityReviewer:  false,
			hasAnyTenantAccess:   true,
			expectedLandingRoute: "/dashboard",
		},
		{
			name:                 "no-access landing",
			canAccessAdmin:       false,
			hasSecurityReviewer:  false,
			hasAnyTenantAccess:   false,
			expectedLandingRoute: "/no-access",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveDefaultLandingRoute(tc.canAccessAdmin, tc.hasSecurityReviewer, tc.hasAnyTenantAccess)
			if got != tc.expectedLandingRoute {
				t.Fatalf("expected %s, got %s", tc.expectedLandingRoute, got)
			}
		})
	}
}

func TestHasSecurityReviewerRole(t *testing.T) {
	if !hasSecurityReviewerRole([]GroupProfileResponse{{RoleType: "security-reviewer"}}, nil) {
		t.Fatalf("expected security reviewer from groups")
	}

	if !hasSecurityReviewerRole(nil, map[string][]RoleResponse{
		"tenant-a": {
			{Name: "Security Reviewer"},
		},
	}) {
		t.Fatalf("expected security reviewer from roles by tenant")
	}

	if hasSecurityReviewerRole([]GroupProfileResponse{{RoleType: "owner"}}, map[string][]RoleResponse{
		"tenant-a": {
			{Name: "Developer"},
		},
	}) {
		t.Fatalf("expected non-security roles to return false")
	}
}

func TestNormalizeRoleKey(t *testing.T) {
	cases := map[string]string{
		"Security Reviewer":  "security_reviewer",
		"security-reviewer":  "security_reviewer",
		"security_reviewer":  "security_reviewer",
		"  Security  Viewer ": "security_viewer",
	}

	for input, expected := range cases {
		if got := normalizeRoleKey(input); got != expected {
			t.Fatalf("input %q: expected %q, got %q", input, expected, got)
		}
	}
}
