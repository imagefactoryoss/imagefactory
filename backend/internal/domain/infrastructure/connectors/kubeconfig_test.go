package connectors

import (
	"testing"
)

func TestBuildRESTConfigFromProviderConfig_Token(t *testing.T) {
	cfg, err := BuildRESTConfigFromProviderConfig(map[string]interface{}{
		"runtime_auth": map[string]interface{}{
			"auth_method": "token",
			"endpoint":    "https://k8s.example.com:6443",
			"token":       "token-value",
			"ca_cert":     "CA-CERT",
		},
	})
	if err != nil {
		t.Fatalf("expected config, got error: %v", err)
	}
	if cfg.Host != "https://k8s.example.com:6443" {
		t.Fatalf("unexpected host: %s", cfg.Host)
	}
	if cfg.BearerToken != "token-value" {
		t.Fatalf("unexpected token")
	}
	if cfg.TLSClientConfig.Insecure {
		t.Fatalf("expected secure config when ca_cert provided")
	}
	if len(cfg.TLSClientConfig.CAData) == 0 {
		t.Fatalf("expected CA data to be set")
	}
}

func TestBuildRESTConfigFromProviderConfig_ServiceAccount(t *testing.T) {
	cfg, err := BuildRESTConfigFromProviderConfig(map[string]interface{}{
		"runtime_auth": map[string]interface{}{
			"auth_method":           "service-account",
			"cluster_endpoint":      "https://k8s.example.com:6443",
			"service_account_token": "sa-token",
		},
	})
	if err != nil {
		t.Fatalf("expected config, got error: %v", err)
	}
	if cfg.Host != "https://k8s.example.com:6443" {
		t.Fatalf("unexpected host: %s", cfg.Host)
	}
	if cfg.BearerToken != "sa-token" {
		t.Fatalf("unexpected token")
	}
}

func TestBuildRESTConfigFromProviderConfig_MissingAuthMethod(t *testing.T) {
	_, err := BuildRESTConfigFromProviderConfig(map[string]interface{}{
		"runtime_auth": map[string]interface{}{},
	})
	if err == nil {
		t.Fatalf("expected error for missing auth_method")
	}
}

func TestBuildRESTConfigFromProviderConfig_ClientCert(t *testing.T) {
	cfg, err := BuildRESTConfigFromProviderConfig(map[string]interface{}{
		"runtime_auth": map[string]interface{}{
			"auth_method": "client-cert",
			"endpoint":    "https://k8s.example.com:6443",
			"client_cert": "CERT",
			"client_key":  "KEY",
		},
	})
	if err != nil {
		t.Fatalf("expected config, got error: %v", err)
	}
	if cfg.Host != "https://k8s.example.com:6443" {
		t.Fatalf("unexpected host: %s", cfg.Host)
	}
	if len(cfg.TLSClientConfig.CertData) == 0 || len(cfg.TLSClientConfig.KeyData) == 0 {
		t.Fatalf("expected client cert data")
	}
	if !cfg.TLSClientConfig.Insecure {
		t.Fatalf("expected insecure config when ca_cert missing")
	}
}

func TestBuildRESTConfigFromProviderConfigForContext_Bootstrap(t *testing.T) {
	cfg, err := BuildRESTConfigFromProviderConfigForContext(map[string]interface{}{
		"bootstrap_auth": map[string]interface{}{
			"auth_method": "token",
			"apiServer":   "https://k8s.example.com:6443",
			"token":       "bootstrap-token",
		},
	}, AuthContextBootstrap)
	if err != nil {
		t.Fatalf("expected bootstrap config, got error: %v", err)
	}
	if cfg.BearerToken != "bootstrap-token" {
		t.Fatalf("unexpected bootstrap token")
	}
}
