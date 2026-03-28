package rest

import "testing"

func TestResolveBootstrapLDAPDefaults_UsesExplicitLDAPEnv(t *testing.T) {
	t.Setenv("IF_AUTH_LDAP_SERVER", "ldaps://ldap.example.com:636")
	t.Setenv("IF_AUTH_LDAP_BASE_DN", "dc=example,dc=com")
	t.Setenv("IF_AUTH_LDAP_BIND_DN", "cn=bind,dc=example,dc=com")
	t.Setenv("IF_AUTH_LDAP_BIND_PASSWORD", "secret")
	t.Setenv("IF_AUTH_LDAP_ENABLED", "true")
	t.Setenv("IF_AUTH_LDAP_ALLOWED_DOMAINS", "example.com")
	t.Setenv("IMAGE_FACTORY_GLAUTH_SERVICE_HOST", "image-factory-glauth")

	defaults := resolveBootstrapLDAPDefaults()

	if defaults.Host != "ldap.example.com" {
		t.Fatalf("expected host ldap.example.com, got %s", defaults.Host)
	}
	if defaults.Port != 636 {
		t.Fatalf("expected port 636, got %d", defaults.Port)
	}
	if defaults.BaseDN != "dc=example,dc=com" {
		t.Fatalf("expected explicit base dn, got %s", defaults.BaseDN)
	}
	if defaults.BindDN != "cn=bind,dc=example,dc=com" {
		t.Fatalf("expected explicit bind dn, got %s", defaults.BindDN)
	}
	if defaults.BindPassword != "secret" {
		t.Fatalf("expected explicit bind password, got %s", defaults.BindPassword)
	}
	if !defaults.Enabled {
		t.Fatalf("expected ldap enabled")
	}
}

func TestResolveBootstrapLDAPDefaults_FallsBackToGLAuth(t *testing.T) {
	t.Setenv("IMAGE_FACTORY_GLAUTH_SERVICE_HOST", "image-factory-glauth")
	t.Setenv("IMAGE_FACTORY_GLAUTH_SERVICE_PORT", "3893")
	t.Setenv("IF_SMTP_FROM_EMAIL", "noreply@imgfactory.com")

	defaults := resolveBootstrapLDAPDefaults()

	if defaults.Host != "image-factory-glauth" {
		t.Fatalf("expected glauth host, got %s", defaults.Host)
	}
	if defaults.Port != 3893 {
		t.Fatalf("expected glauth port 3893, got %d", defaults.Port)
	}
	if defaults.BaseDN != "dc=imgfactory,dc=com" {
		t.Fatalf("expected glauth base dn fallback, got %s", defaults.BaseDN)
	}
	if defaults.BindDN != "cn=ldap_search,ou=svcaccts,dc=imgfactory,dc=com" {
		t.Fatalf("unexpected glauth bind dn fallback: %s", defaults.BindDN)
	}
	if defaults.BindPassword != "search_password" {
		t.Fatalf("expected glauth bind password fallback, got %s", defaults.BindPassword)
	}
	if !defaults.Enabled {
		t.Fatalf("expected ldap enabled from glauth fallback")
	}
	if len(defaults.AllowedDomains) != 1 || defaults.AllowedDomains[0] != "imgfactory.com" {
		t.Fatalf("expected allowed domain fallback from smtp email, got %+v", defaults.AllowedDomains)
	}
}
