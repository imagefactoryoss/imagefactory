package infrastructure

import (
	"strings"
	"testing"
)

func TestRuntimeTenantRBACYAML_HasCorrectAPIVersionAndKinds(t *testing.T) {
	y := runtimeTenantRBACYAML("system-ns", "tenant-ns")
	if !strings.Contains(y, "apiVersion: rbac.authorization.k8s.io/v1") {
		t.Fatalf("expected Role apiVersion rbac.authorization.k8s.io/v1, got:\n%s", y)
	}
	if !strings.Contains(y, "kind: Role") {
		t.Fatalf("expected Role kind present")
	}
	if !strings.Contains(y, "kind: RoleBinding") {
		t.Fatalf("expected RoleBinding present")
	}
	if !strings.Contains(y, `resources: ["taskruns"]`) {
		t.Fatalf("expected taskruns rule in runtime tenant RBAC")
	}
	if !strings.Contains(y, `resources: ["pipelineruns"]`) {
		t.Fatalf("expected pipelineruns rule in runtime tenant RBAC")
	}
	if !strings.Contains(y, `resources: ["pods", "pods/log"`) {
		t.Fatalf("expected pods and pods/log rule in runtime tenant RBAC")
	}
	if !strings.Contains(y, `verbs: ["create", "get", "list", "watch", "update", "patch", "delete"]`) {
		t.Fatalf("expected broad runtime verbs for core namespace resources")
	}
}
