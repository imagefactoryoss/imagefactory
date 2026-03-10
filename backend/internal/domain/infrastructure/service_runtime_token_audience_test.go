package infrastructure

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseJWTAudienceClaim_String(t *testing.T) {
	token := makeUnsignedTestJWT(t, map[string]interface{}{
		"aud": "https://kubernetes.default.svc.cluster.local",
	})

	audiences, err := parseJWTAudienceClaim(token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(audiences) != 1 || audiences[0] != "https://kubernetes.default.svc.cluster.local" {
		t.Fatalf("unexpected audiences: %#v", audiences)
	}
}

func TestParseJWTAudienceClaim_Array(t *testing.T) {
	token := makeUnsignedTestJWT(t, map[string]interface{}{
		"aud": []interface{}{"k3s", "https://kubernetes.default.svc"},
	})

	audiences, err := parseJWTAudienceClaim(token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(audiences) != 2 {
		t.Fatalf("unexpected audience count: %#v", audiences)
	}
	if audiences[0] != "k3s" || audiences[1] != "https://kubernetes.default.svc" {
		t.Fatalf("unexpected audiences: %#v", audiences)
	}
}

func TestResolveRuntimeTokenAudiences_Rancher(t *testing.T) {
	bootstrapToken := makeUnsignedTestJWT(t, map[string]interface{}{
		"aud": []interface{}{"https://kubernetes.default.svc.cluster.local"},
	})
	provider := &Provider{
		ProviderType: ProviderTypeRancher,
		Config: map[string]interface{}{
			"bootstrap_auth": map[string]interface{}{
				"token": bootstrapToken,
			},
		},
	}

	audiences := resolveRuntimeTokenAudiences(provider, "https://kubernetes.default.svc.cluster.local:6443")

	assertContainsAudience(t, audiences, "https://kubernetes.default.svc.cluster.local")
	assertContainsAudience(t, audiences, "https://kubernetes.default.svc")
	assertContainsAudience(t, audiences, "k3s")
}

func TestResolveRuntimeTokenAudiences_OpenShift(t *testing.T) {
	provider := &Provider{ProviderType: ProviderTypeOpenShift}

	audiences := resolveRuntimeTokenAudiences(provider, "https://api.cluster.example.com:6443")

	assertContainsAudience(t, audiences, "https://api.cluster.example.com:6443")
	assertContainsAudience(t, audiences, "https://api.cluster.example.com")
	assertContainsAudience(t, audiences, "https://kubernetes.default.svc")
	assertContainsAudience(t, audiences, "https://kubernetes.default.svc.cluster.local")
}

func TestManagedRuntimeAuthNeedsRegeneration_AudienceMismatch(t *testing.T) {
	bootstrapToken := makeUnsignedTestJWT(t, map[string]interface{}{
		"aud": []interface{}{"https://kubernetes.default.svc.cluster.local", "k3s"},
	})
	staleRuntimeToken := makeUnsignedTestJWT(t, map[string]interface{}{
		"aud": []interface{}{"https://kubernetes.default.svc"},
	})

	provider := &Provider{
		ProviderType: ProviderTypeRancher,
		Config: map[string]interface{}{
			"bootstrap_auth": map[string]interface{}{
				"token": bootstrapToken,
			},
			"runtime_auth": map[string]interface{}{
				"endpoint": "https://localhost:6443",
				"token":    staleRuntimeToken,
			},
		},
	}

	needsRegenerate, reason := managedRuntimeAuthNeedsRegeneration(provider, "https://localhost:6443", resolveRuntimeTokenAudiences(provider, "https://localhost:6443"))
	if !needsRegenerate {
		t.Fatalf("expected regeneration for stale runtime token audiences")
	}
	if reason == "" {
		t.Fatalf("expected non-empty regeneration reason")
	}
}

func TestManagedRuntimeAuthNeedsRegeneration_ValidIntersection(t *testing.T) {
	bootstrapToken := makeUnsignedTestJWT(t, map[string]interface{}{
		"aud": []interface{}{"https://kubernetes.default.svc.cluster.local", "k3s"},
	})
	runtimeToken := makeUnsignedTestJWT(t, map[string]interface{}{
		"aud": []interface{}{"k3s"},
	})

	provider := &Provider{
		ProviderType: ProviderTypeRancher,
		Config: map[string]interface{}{
			"bootstrap_auth": map[string]interface{}{
				"token": bootstrapToken,
			},
			"runtime_auth": map[string]interface{}{
				"endpoint": "https://localhost:6443",
				"token":    runtimeToken,
			},
		},
	}

	needsRegenerate, reason := managedRuntimeAuthNeedsRegeneration(provider, "https://localhost:6443", resolveRuntimeTokenAudiences(provider, "https://localhost:6443"))
	if needsRegenerate {
		t.Fatalf("did not expect regeneration, reason=%q", reason)
	}
}

func assertContainsAudience(t *testing.T, audiences []string, expected string) {
	t.Helper()
	for _, aud := range audiences {
		if aud == expected {
			return
		}
	}
	t.Fatalf("expected audience %q in %#v", expected, audiences)
}

func makeUnsignedTestJWT(t *testing.T, claims map[string]interface{}) string {
	t.Helper()

	header := map[string]string{"alg": "none", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}

	encode := func(raw []byte) string {
		return strings.TrimRight(base64.URLEncoding.EncodeToString(raw), "=")
	}

	return encode(headerJSON) + "." + encode(claimsJSON) + "."
}
