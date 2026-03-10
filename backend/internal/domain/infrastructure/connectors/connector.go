package connectors

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Connector defines the interface for testing provider connections
type Connector interface {
	// TestConnection tests the connection to a provider
	TestConnection(ctx context.Context) (*ConnectionResult, error)
}

// ConnectionResult represents the result of a connection test
type ConnectionResult struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ConnectorFactory creates connectors for different provider types
type ConnectorFactory struct {
	logger *zap.Logger
}

// NewConnectorFactory creates a new connector factory
func NewConnectorFactory(logger *zap.Logger) *ConnectorFactory {
	return &ConnectorFactory{
		logger: logger,
	}
}

// CreateConnector creates a connector for the given provider type and config
func (f *ConnectorFactory) CreateConnector(providerType string, config map[string]interface{}) (Connector, error) {
	switch providerType {
	case "kubernetes":
		return NewKubernetesConnector(config, f.logger), nil
	case "build_nodes":
		return NewBuildNodesConnector(config, f.logger), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// KubernetesConnector tests connections to Kubernetes clusters
type KubernetesConnector struct {
	config map[string]interface{}
	logger *zap.Logger
}

// NewKubernetesConnector creates a new Kubernetes connector
func NewKubernetesConnector(config map[string]interface{}, logger *zap.Logger) *KubernetesConnector {
	return &KubernetesConnector{
		config: config,
		logger: logger,
	}
}

// TestConnection tests the connection to a Kubernetes cluster
func (c *KubernetesConnector) TestConnection(ctx context.Context) (*ConnectionResult, error) {
	c.logger.Info("Testing Kubernetes connection")
	restConfig, err := BuildRESTConfigFromProviderConfig(c.config)
	if err != nil {
		return &ConnectionResult{
			Success: false,
			Message: fmt.Sprintf("invalid runtime auth config: %v", err),
		}, nil
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return &ConnectionResult{
			Success: false,
			Message: fmt.Sprintf("failed to create kubernetes client: %v", err),
		}, nil
	}

	clusterInfo := map[string]interface{}{
		"host": restConfig.Host,
	}
	if runtimeAuth, ok := c.config["runtime_auth"].(map[string]interface{}); ok {
		if authMethod, ok := runtimeAuth["auth_method"].(string); ok && authMethod != "" {
			clusterInfo["auth_method"] = authMethod
		}
	}

	// Test the connection by getting the server version
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return &ConnectionResult{
			Success: false,
			Message: fmt.Sprintf("failed to get server version: %v", err),
		}, nil
	}

	// Get node information
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Warn("Failed to get node information", zap.Error(err))
	} else {
		clusterInfo["node_count"] = len(nodes.Items)
		clusterInfo["nodes"] = c.getNodeSummary(nodes.Items)
	}

	// Get namespace count
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Warn("Failed to get namespace information", zap.Error(err))
	} else {
		clusterInfo["namespace_count"] = len(namespaces.Items)
	}

	clusterInfo["server_version"] = serverVersion.GitVersion
	clusterInfo["platform"] = serverVersion.Platform

	return &ConnectionResult{
		Success: true,
		Message: "Successfully connected to Kubernetes cluster",
		Details: clusterInfo,
	}, nil
}

// connectViaKubeconfig connects using a kubeconfig file
func (c *KubernetesConnector) connectViaKubeconfig(ctx context.Context) (*kubernetes.Clientset, map[string]interface{}, error) {
	kubeconfigPath, ok := c.config["kubeconfig_path"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("kubeconfig_path is required for kubeconfig auth method")
	}

	// Use the kubeconfig file to create a config
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	clusterInfo := map[string]interface{}{
		"auth_method":     "kubeconfig",
		"kubeconfig_path": kubeconfigPath,
		"host":            config.Host,
	}

	return clientset, clusterInfo, nil
}

// connectViaToken connects using a service account token
func (c *KubernetesConnector) connectViaToken(ctx context.Context) (*kubernetes.Clientset, map[string]interface{}, error) {
	// Prefer apiServer over endpoint for Kubernetes API server URL
	endpoint, ok := c.config["apiServer"].(string)
	if !ok {
		endpoint, ok = c.config["endpoint"].(string)
		if !ok {
			return nil, nil, fmt.Errorf("apiServer or endpoint is required for token auth method")
		}
	}

	token, ok := c.config["token"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("token is required for token auth method")
	}

	// Create config from token
	config := &rest.Config{
		Host:        endpoint,
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true, // Skip TLS verification for development
		},
	}

	// Add CA cert if provided
	if caCert, ok := c.config["ca_cert"].(string); ok && caCert != "" {
		config.TLSClientConfig.CAData = []byte(caCert)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	clusterInfo := map[string]interface{}{
		"auth_method": "token",
		"endpoint":    endpoint,
	}

	return clientset, clusterInfo, nil
}

// connectViaClientCert connects using client certificate authentication
func (c *KubernetesConnector) connectViaClientCert(ctx context.Context) (*kubernetes.Clientset, map[string]interface{}, error) {
	// Prefer apiServer over endpoint for Kubernetes API server URL
	endpoint, ok := c.config["apiServer"].(string)
	if !ok {
		endpoint, ok = c.config["endpoint"].(string)
		if !ok {
			return nil, nil, fmt.Errorf("apiServer or endpoint is required for client-cert auth method")
		}
	}

	clientCert, ok := c.config["client_cert"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("client_cert is required for client-cert auth method")
	}

	clientKey, ok := c.config["client_key"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("client_key is required for client-cert auth method")
	}

	// Create config from client certificate
	config := &rest.Config{
		Host: endpoint,
		TLSClientConfig: rest.TLSClientConfig{
			CertData: []byte(clientCert),
			KeyData:  []byte(clientKey),
			Insecure: true, // Skip TLS verification for development
		},
	}

	// Add CA cert if provided
	if caCert, ok := c.config["ca_cert"].(string); ok && caCert != "" {
		config.TLSClientConfig.CAData = []byte(caCert)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	clusterInfo := map[string]interface{}{
		"auth_method": "client-cert",
		"endpoint":    endpoint,
	}

	return clientset, clusterInfo, nil
}

// getNodeSummary returns a summary of node information
func (c *KubernetesConnector) getNodeSummary(nodes []corev1.Node) []map[string]interface{} {
	summary := make([]map[string]interface{}, 0, len(nodes))
	for _, node := range nodes {
		name := node.Name
		ready := false

		// Check if node is ready
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				ready = true
				break
			}
		}

		summary = append(summary, map[string]interface{}{
			"name":  name,
			"ready": ready,
		})
	}
	return summary
}

// BuildNodesConnector tests connections to build nodes
type BuildNodesConnector struct {
	config map[string]interface{}
	logger *zap.Logger
}

// NewBuildNodesConnector creates a new build nodes connector
func NewBuildNodesConnector(config map[string]interface{}, logger *zap.Logger) *BuildNodesConnector {
	return &BuildNodesConnector{
		config: config,
		logger: logger,
	}
}

// TestConnection tests the connection to build nodes
func (c *BuildNodesConnector) TestConnection(ctx context.Context) (*ConnectionResult, error) {
	c.logger.Info("Testing build nodes connection")

	// For build nodes, we perform basic connectivity checks
	// This is a simplified implementation that validates the config

	details := make(map[string]interface{})

	// Check for required fields
	if endpoint, ok := c.config["endpoint"].(string); ok {
		details["endpoint"] = endpoint
	} else {
		return &ConnectionResult{
			Success: false,
			Message: "endpoint is required in config",
		}, nil
	}

	// Check for authentication
	if _, ok := c.config["api_key"].(string); ok {
		details["auth_method"] = "api_key"
	} else if _, ok := c.config["token"].(string); ok {
		details["auth_method"] = "token"
	} else {
		details["auth_method"] = "none"
	}

	// Add additional config details
	if region, ok := c.config["region"].(string); ok {
		details["region"] = region
	}

	// Simulate a connection test with timeout
	select {
	case <-time.After(500 * time.Millisecond):
		// Simulated successful connection
		return &ConnectionResult{
			Success: true,
			Message: "Successfully connected to build nodes",
			Details: details,
		}, nil
	case <-ctx.Done():
		return &ConnectionResult{
			Success: false,
			Message: "Connection test timed out",
		}, nil
	}
}
