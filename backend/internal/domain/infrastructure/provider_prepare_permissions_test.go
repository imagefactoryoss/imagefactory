package infrastructure

import "testing"

func TestPermissionAuditSpecs_ImageFactoryManagedIncludesBootstrapPermissions(t *testing.T) {
	specs := permissionAuditSpecs("image_factory_managed", "image-factory-demo")
	keys := make(map[string]bool, len(specs))
	for _, spec := range specs {
		keys[spec.Key] = true
	}

	required := []string{
		"perm.tasks.get",
		"perm.pipelines.get",
		"perm.pipelineruns.create",
		"perm.secrets.get",
		"perm.pods.create",
		"perm.configmaps.get",
		"perm.persistentvolumeclaims.create",
		"perm.namespaces.create",
		"perm.tasks.patch",
		"perm.pipelines.patch",
		"perm.roles.create",
		"perm.rolebindings.create",
		"perm.serviceaccounts.create",
	}
	for _, key := range required {
		if !keys[key] {
			t.Fatalf("expected managed permission spec %q to be present", key)
		}
	}
}

func TestPermissionAuditSpecs_SelfManagedSkipsBootstrapWritePermissions(t *testing.T) {
	specs := permissionAuditSpecs("self_managed", "image-factory-demo")
	keys := make(map[string]bool, len(specs))
	for _, spec := range specs {
		keys[spec.Key] = true
	}

	// Self-managed should not include bootstrap write permissions that only apply when Image Factory
	// is allowed to perform cluster/namespace provisioning.
	unexpected := []string{
		"perm.namespaces.create",
		"perm.tasks.patch",
		"perm.pipelines.patch",
		"perm.roles.create",
		"perm.rolebindings.create",
		"perm.serviceaccounts.create",
	}
	for _, key := range unexpected {
		if keys[key] {
			t.Fatalf("did not expect self-managed permission spec %q", key)
		}
	}
}
