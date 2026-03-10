package connectors

import (
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	AuthContextRuntime   = "runtime"
	AuthContextBootstrap = "bootstrap"
)

// BuildRESTConfigFromProviderConfig builds a Kubernetes REST config from provider config fields.
func BuildRESTConfigFromProviderConfig(config map[string]interface{}) (*rest.Config, error) {
	return BuildRESTConfigFromProviderConfigForContext(config, AuthContextRuntime)
}

// BuildRESTConfigFromProviderConfigForContext builds a REST config using a specific auth context.
// Supported contexts:
// - runtime: requires config.runtime_auth
// - bootstrap: requires config.bootstrap_auth
func BuildRESTConfigFromProviderConfigForContext(config map[string]interface{}, authContext string) (*rest.Config, error) {
	authConfig, err := authConfigForContext(config, authContext)
	if err != nil {
		return nil, err
	}

	authMethod := getStringValue(authConfig, "auth_method")
	if authMethod == "" {
		return nil, fmt.Errorf("%s auth config: auth_method is required", authContext)
	}

	switch authMethod {
	case "kubeconfig":
		if kubeconfigPath := getStringValue(authConfig, "kubeconfig_path"); kubeconfigPath != "" {
			return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		}
		if kubeconfigContent := getStringValue(authConfig, "kubeconfig"); kubeconfigContent != "" {
			return clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfigContent))
		}
		return nil, fmt.Errorf("%s auth config: kubeconfig_path is required for kubeconfig auth method", authContext)
	case "token", "service-account", "service_account", "oauth":
		endpoint := firstNonEmpty(
			getStringValue(authConfig, "apiServer"),
			getStringValue(authConfig, "endpoint"),
			getStringValue(authConfig, "cluster_endpoint"),
		)
		if endpoint == "" {
			return nil, fmt.Errorf("%s auth config: apiServer or endpoint is required for token auth method", authContext)
		}
		token := firstNonEmpty(
			getStringValue(authConfig, "token"),
			getStringValue(authConfig, "service_account_token"),
			getStringValue(authConfig, "oauth_token"),
			getStringValue(authConfig, "api_token"),
		)
		if token == "" {
			return nil, fmt.Errorf("%s auth config: token is required for token auth method", authContext)
		}
		return buildTokenConfig(endpoint, token, getStringValue(authConfig, "ca_cert")), nil
	case "client-cert":
		endpoint := firstNonEmpty(
			getStringValue(authConfig, "apiServer"),
			getStringValue(authConfig, "endpoint"),
			getStringValue(authConfig, "cluster_endpoint"),
		)
		if endpoint == "" {
			return nil, fmt.Errorf("%s auth config: apiServer or endpoint is required for client-cert auth method", authContext)
		}
		clientCert := getStringValue(authConfig, "client_cert")
		if clientCert == "" {
			return nil, fmt.Errorf("%s auth config: client_cert is required for client-cert auth method", authContext)
		}
		clientKey := getStringValue(authConfig, "client_key")
		if clientKey == "" {
			return nil, fmt.Errorf("%s auth config: client_key is required for client-cert auth method", authContext)
		}
		cfg := &rest.Config{
			Host: endpoint,
			TLSClientConfig: rest.TLSClientConfig{
				CertData: []byte(clientCert),
				KeyData:  []byte(clientKey),
			},
		}
		if caCert := getStringValue(authConfig, "ca_cert"); caCert != "" {
			cfg.TLSClientConfig.CAData = []byte(caCert)
		} else {
			cfg.TLSClientConfig.Insecure = true
		}
		return cfg, nil
	case "iam", "workload-identity", "managed-identity", "instance-principal", "api-key":
		if inCluster, err := rest.InClusterConfig(); err == nil {
			return inCluster, nil
		}
		if kubeconfigPath := getStringValue(authConfig, "kubeconfig_path"); kubeconfigPath != "" {
			return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		}
		if kubeconfigContent := getStringValue(authConfig, "kubeconfig"); kubeconfigContent != "" {
			return clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfigContent))
		}
		return nil, fmt.Errorf("%s auth config: auth method %s requires in-cluster config or kubeconfig", authContext, authMethod)
	default:
		return nil, fmt.Errorf("%s auth config: unsupported auth method: %s", authContext, authMethod)
	}
}

func authConfigForContext(config map[string]interface{}, authContext string) (map[string]interface{}, error) {
	if config == nil {
		return nil, fmt.Errorf("%s auth config is required", authContext)
	}
	key := ""
	switch authContext {
	case AuthContextRuntime:
		key = "runtime_auth"
	case AuthContextBootstrap:
		key = "bootstrap_auth"
	default:
		return nil, fmt.Errorf("unsupported auth context: %s", authContext)
	}

	raw, exists := config[key]
	if !exists || raw == nil {
		return nil, fmt.Errorf("%s auth config is required (config.%s)", authContext, key)
	}
	authConfig, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s auth config is invalid (config.%s must be an object)", authContext, key)
	}
	return authConfig, nil
}

func buildTokenConfig(endpoint, token, caCert string) *rest.Config {
	cfg := &rest.Config{
		Host:        endpoint,
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}
	if caCert != "" {
		cfg.TLSClientConfig.CAData = []byte(caCert)
		cfg.TLSClientConfig.Insecure = false
	}
	return cfg
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func getStringValue(config map[string]interface{}, key string) string {
	value, ok := config[key]
	if !ok || value == nil {
		return ""
	}
	return normalizeString(value)
}

func normalizeString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}
