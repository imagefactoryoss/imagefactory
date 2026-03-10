package rest

import "testing"

func TestRoleTypeSetNormalizesKeys(t *testing.T) {
	set := roleTypeSet([]string{
		"Owner",
		"security-reviewer",
		" System Administrator Viewer ",
	})

	expected := []string{
		"owner",
		"security_reviewer",
		"system_administrator_viewer",
	}

	for _, key := range expected {
		if _, ok := set[key]; !ok {
			t.Fatalf("expected normalized key %q in role type set", key)
		}
	}
}

func TestIsTenantRoleAssignable(t *testing.T) {
	allowed := roleTypeSet([]string{"owner", "developer", "viewer"})

	if !isTenantRoleAssignable("Owner", allowed) {
		t.Fatalf("expected owner to be assignable")
	}

	if !isTenantRoleAssignable("  DeVelOper ", allowed) {
		t.Fatalf("expected developer variant to be assignable")
	}

	if isTenantRoleAssignable("Security Reviewer", allowed) {
		t.Fatalf("expected security reviewer to be rejected for tenant role assignment")
	}
}
