# Kubernetes Cluster Connectivity Guide

This guide explains how Image Factory connects to Kubernetes clusters for hybrid infrastructure execution, covering both local development and multi-cluster production patterns.

Use it as a technical reference for connectivity patterns and operational concerns. Validate implementation details against the current infrastructure provider code and configuration model.

## Overview

This guide covers how the Image Factory system connects to Kubernetes clusters for hybrid infrastructure execution, supporting both local development and production multi-cluster deployments.

### Connection Methods
- **Local Development**: Direct kubeconfig access
- **Production**: Secure service account tokens, IAM authentication
- **Multi-Cluster**: Dynamic cluster discovery and routing

---

## Local Development Cluster Connection

### Method 1: Kubeconfig File (Most Common)

```go
// internal/infrastructure/k8s/client.go
type K8sClient struct {
    config *rest.Config
    clientset *kubernetes.Clientset
    tektonClient *tektonclientset.Clientset
}

func NewK8sClient(kubeconfigPath string) (*K8sClient, error) {
    // Load kubeconfig
    config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
    }

    // Create clients
    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
    }

    tektonClient, err := tektonclientset.NewForConfig(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create tekton client: %w", err)
    }

    return &K8sClient{
        config: config,
        clientset: clientset,
        tektonClient: tektonClient,
    }, nil
}
```

**Configuration:**
```yaml
# config/local.yaml
kubernetes:
  clusters:
    - name: "local-dev"
      kubeconfig: "~/.kube/config"
      default: true
      context: "docker-desktop"  # or "minikube", "kind"
```

### Method 2: In-Cluster Configuration (When Running in K8s)

```go
func NewInClusterClient() (*K8sClient, error) {
    // Use in-cluster config when running as a pod
    config, err := rest.InClusterConfig()
    if err != nil {
        return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
    }

    // Create clients with in-cluster config
    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
    }

    return &K8sClient{
        config: config,
        clientset: clientset,
    }, nil
}
```

---

## Production Multi-Cluster Connectivity

### Architecture Overview

```
Image Factory Backend
        │
        ├── Cluster Registry API
        │   ├── Cluster Discovery
        │   ├── Health Monitoring
        │   └── Load Balancing
        │
        └── Cluster Connectors
            ├── Service Account Auth
            ├── IAM Authentication
            └── Certificate-Based Auth
```

### Method 1: Service Account Tokens (Recommended for Production)

```go
// internal/infrastructure/k8s/cluster_manager.go
type ClusterManager struct {
    clusters map[string]*ClusterConnection
    registry *ClusterRegistry
}

type ClusterConnection struct {
    Name        string
    Endpoint    string
    CAData      []byte
    Token       string  // Service account token
    Namespace   string
    client      *K8sClient
    lastHealth  time.Time
}

func (cm *ClusterManager) ConnectToCluster(clusterName string) (*K8sClient, error) {
    cluster, exists := cm.clusters[clusterName]
    if !exists {
        return nil, fmt.Errorf("cluster %s not found", clusterName)
    }

    // Create REST config from service account token
    config := &rest.Config{
        Host:        cluster.Endpoint,
        BearerToken: cluster.Token,
        TLSClientConfig: rest.TLSClientConfig{
            CAData: cluster.CAData,
        },
    }

    // Create and cache client
    client, err := NewK8sClientFromConfig(config)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to cluster %s: %w", clusterName, err)
    }

    cluster.client = client
    cluster.lastHealth = time.Now()

    return client, nil
}
```

**Service Account Setup:**
```bash
# Create service account for Image Factory
kubectl create serviceaccount image-factory-sa -n image-factory

# Create cluster role with necessary permissions
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: image-factory-role
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log", "configmaps", "secrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["tekton.dev"]
  resources: ["pipelineruns", "taskruns", "pipelines", "tasks"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
EOF

# Bind role to service account
kubectl create clusterrolebinding image-factory-binding \
  --clusterrole=image-factory-role \
  --serviceaccount=image-factory:image-factory-sa

# Get service account token
kubectl get secret $(kubectl get serviceaccount image-factory-sa -o jsonpath='{.secrets[0].name}') -o jsonpath='{.data.token}' | base64 --decode
```

### Method 2: Cloud Provider IAM Authentication

#### AWS EKS with IAM
```go
// internal/infrastructure/k8s/aws_connector.go
type AWSClusterConnector struct {
    region     string
    clusterName string
    roleARN    string
}

func (c *AWSClusterConnector) Connect(ctx context.Context) (*K8sClient, error) {
    // Get EKS cluster details
    eksClient := eks.New(sess.Must(sess.NewSession(&aws.Config{
        Region: aws.String(c.region),
    })))

    cluster, err := eksClient.DescribeCluster(&eks.DescribeClusterInput{
        Name: &c.clusterName,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to describe EKS cluster: %w", err)
    }

    // Create AWS IAM authenticator
    config := &rest.Config{
        Host: cluster.Cluster.Endpoint,
        TLSClientConfig: rest.TLSClientConfig{
            CAData: []byte(*cluster.Cluster.CertificateAuthority.Data),
        },
    }

    // Use AWS IAM authenticator
    config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
        return &awsAuthTransport{
            RoundTripper: rt,
            region:       c.region,
            clusterName:  c.clusterName,
            roleARN:      c.roleARN,
        }
    }

    return NewK8sClientFromConfig(config)
}
```

#### GCP GKE with Workload Identity
```go
// internal/infrastructure/k8s/gcp_connector.go
type GCPClusterConnector struct {
    projectID   string
    clusterName string
    location    string
    serviceAccount string
}

func (c *GCPClusterConnector) Connect(ctx context.Context) (*K8sClient, error) {
    // Use GCP Workload Identity
    config, err := google.DefaultClient(ctx, container.CloudPlatformScope)
    if err != nil {
        return nil, fmt.Errorf("failed to get GCP client: %w", err)
    }

    // Get GKE cluster details
    service, err := container.NewService(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to create GKE service: %w", err)
    }

    cluster, err := service.Projects.Locations.Clusters.Get(
        fmt.Sprintf("projects/%s/locations/%s/clusters/%s",
            c.projectID, c.location, c.clusterName)).Do()
    if err != nil {
        return nil, fmt.Errorf("failed to get GKE cluster: %w", err)
    }

    // Create REST config with GCP auth
    restConfig := &rest.Config{
        Host: fmt.Sprintf("https://%s", cluster.Endpoint),
        TLSClientConfig: rest.TLSClientConfig{
            CAData: []byte(cluster.MasterAuth.ClusterCaCertificate),
        },
    }

    // GCP authentication is handled by the client
    restConfig.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
        return &gcpAuthTransport{
            RoundTripper: rt,
            tokenSource: config,
        }
    }

    return NewK8sClientFromConfig(restConfig)
}
```

#### Azure AKS with Managed Identity
```go
// internal/infrastructure/k8s/azure_connector.go
type AzureClusterConnector struct {
    subscriptionID    string
    resourceGroupName string
    clusterName       string
    clientID          string
}

func (c *AzureClusterConnector) Connect(ctx context.Context) (*K8sClient, error) {
    // Azure authentication using managed identity
    cred, err := azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{
        ID: azidentity.ClientID(c.clientID),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create Azure credential: %w", err)
    }

    // Get AKS cluster details
    clustersClient, err := armcontainerservice.NewManagedClustersClient(c.subscriptionID, cred, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create AKS client: %w", err)
    }

    cluster, err := clustersClient.Get(ctx, c.resourceGroupName, c.clusterName, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to get AKS cluster: %w", err)
    }

    // Create REST config with Azure auth
    config := &rest.Config{
        Host: *cluster.Properties.Fqdn,
        TLSClientConfig: rest.TLSClientConfig{
            CAData: []byte(*cluster.Properties.KubeConfig.CertificateAuthorityData),
        },
    }

    config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
        return &azureAuthTransport{
            RoundTripper: rt,
            cred:         cred,
        }
    }

    return NewK8sClientFromConfig(config)
}
```

### Method 3: Certificate-Based Authentication

```go
// internal/infrastructure/k8s/cert_connector.go
type CertificateClusterConnector struct {
    endpoint    string
    caCert      []byte
    clientCert  []byte
    clientKey   []byte
}

func (c *CertificateClusterConnector) Connect(ctx context.Context) (*K8sClient, error) {
    config := &rest.Config{
        Host: c.endpoint,
        TLSClientConfig: rest.TLSClientConfig{
            CAData:   c.caCert,
            CertData: c.clientCert,
            KeyData:  c.clientKey,
        },
    }

    return NewK8sClientFromConfig(config)
}
```

---

## Cluster Registry And Discovery

### Cluster Registry API

```go
// internal/infrastructure/k8s/registry.go
type ClusterRegistry struct {
    clusters   map[string]*ClusterInfo
    discovery  ClusterDiscovery
    healthCheck *HealthChecker
}

type ClusterInfo struct {
    Name         string
    Provider     string  // aws, gcp, azure, onprem
    Region       string
    Endpoint     string
    AuthMethod   string  // service-account, iam, cert
    Status       ClusterStatus
    Capabilities []string  // gpu, arm64, windows, etc.
    LastHealth   time.Time
}

func (r *ClusterRegistry) DiscoverClusters(ctx context.Context) error {
    // Discover clusters based on configuration
    discovered, err := r.discovery.Discover(ctx)
    if err != nil {
        return fmt.Errorf("failed to discover clusters: %w", err)
    }

    // Update registry
    for _, cluster := range discovered {
        r.clusters[cluster.Name] = cluster
    }

    return nil
}

func (r *ClusterRegistry) GetHealthyClusters() []*ClusterInfo {
    var healthy []*ClusterInfo
    for _, cluster := range r.clusters {
        if cluster.Status == ClusterHealthy {
            healthy = append(healthy, cluster)
        }
    }
    return healthy
}
```

### Dynamic Cluster Discovery

```go
// internal/infrastructure/k8s/discovery.go
type ClusterDiscovery interface {
    Discover(ctx context.Context) ([]*ClusterInfo, error)
}

// AWS EKS Discovery
type AWSEKSDiscovery struct {
    regions []string
}

func (d *AWSEKSDiscovery) Discover(ctx context.Context) ([]*ClusterInfo, error) {
    var clusters []*ClusterInfo

    for _, region := range d.regions {
        sess := session.Must(session.NewSession(&aws.Config{
            Region: aws.String(region),
        }))

        eksClient := eks.New(sess)

        // List all EKS clusters in region
        result, err := eksClient.ListClusters(&eks.ListClustersInput{})
        if err != nil {
            continue // Skip region if error
        }

        for _, clusterName := range result.Clusters {
            cluster := &ClusterInfo{
                Name:       *clusterName,
                Provider:   "aws",
                Region:     region,
                AuthMethod: "iam",
                Status:     ClusterUnknown,
            }
            clusters = append(clusters, cluster)
        }
    }

    return clusters, nil
}
```

---

## Connection Management And Health Monitoring

### Connection Pooling

```go
// internal/infrastructure/k8s/connection_pool.go
type ConnectionPool struct {
    clients    map[string]*PooledClient
    maxIdle    time.Duration
    maxClients int
    mu         sync.RWMutex
}

type PooledClient struct {
    client     *K8sClient
    lastUsed   time.Time
    clusterName string
}

func (p *ConnectionPool) Get(clusterName string) (*K8sClient, error) {
    p.mu.Lock()
    defer p.mu.Unlock()

    if client, exists := p.clients[clusterName]; exists {
        // Check if client is still valid
        if time.Since(client.lastUsed) < p.maxIdle {
            client.lastUsed = time.Now()
            return client.client, nil
        }
        // Remove stale client
        delete(p.clients, clusterName)
    }

    // Create new client
    client, err := p.createClient(clusterName)
    if err != nil {
        return nil, err
    }

    p.clients[clusterName] = &PooledClient{
        client:      client,
        lastUsed:    time.Now(),
        clusterName: clusterName,
    }

    return client, nil
}

func (p *ConnectionPool) HealthCheck(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            p.performHealthChecks(ctx)
        }
    }
}
```

### Health Monitoring

```go
// internal/infrastructure/k8s/health.go
type HealthChecker struct {
    timeout time.Duration
}

func (h *HealthChecker) CheckClusterHealth(ctx context.Context, client *K8sClient) ClusterStatus {
    ctx, cancel := context.WithTimeout(ctx, h.timeout)
    defer cancel()

    // Check API server connectivity
    _, err := client.clientset.ServerVersion()
    if err != nil {
        return ClusterUnhealthy
    }

    // Check Tekton installation
    _, err = client.tektonClient.TektonV1beta1().Pipelines("").List(ctx, metav1.ListOptions{Limit: 1})
    if err != nil {
        return ClusterDegraded
    }

    // Check node capacity
    nodes, err := client.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
    if err != nil {
        return ClusterDegraded
    }

    // Check if there are ready nodes
    readyNodes := 0
    for _, node := range nodes.Items {
        for _, condition := range node.Status.Conditions {
            if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
                readyNodes++
                break
            }
        }
    }

    if readyNodes == 0 {
        return ClusterUnhealthy
    }

    return ClusterHealthy
}
```

---

## Security Considerations

### Authentication Best Practices

1. **Service Accounts for Production**
   - Use dedicated service accounts with minimal permissions
   - Rotate tokens regularly
   - Store tokens securely (vault, secrets manager)

2. **IAM Authentication**
   - Use cloud provider IAM roles
   - Implement least privilege access
   - Enable audit logging

3. **Certificate Management**
   - Use short-lived certificates
   - Implement certificate rotation
   - Store certificates in secure locations

### Network Security

```yaml
# Network policies for cluster access
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: image-factory-access
  namespace: image-factory
spec:
  podSelector:
    matchLabels:
      app: image-factory
  policyTypes:
  - Egress
  egress:
  - to:
    - ipBlock:
        cidr: 10.0.0.0/8  # Cluster internal
    ports:
    - protocol: TCP
      port: 443
  - to:
    - ipBlock:
        cidr: 172.16.0.0/12  # Cluster internal
    ports:
    - protocol: TCP
      port: 6443  # API server
```

### RBAC Configuration

```yaml
# Cluster role for Image Factory
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: image-factory-cluster-role
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log", "pods/exec", "configmaps", "secrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["tekton.dev"]
  resources: ["pipelineruns", "taskruns", "pipelines", "tasks", "conditions"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["tekton.dev"]
  resources: ["clustertasks"]
  verbs: ["get", "list", "watch"]
```

---

## Configuration Management

### Cluster Configuration Schema

```go
// internal/config/kubernetes.go
type KubernetesConfig struct {
    Clusters []ClusterConfig `yaml:"clusters"`
    DefaultCluster string    `yaml:"default_cluster"`
    ConnectionPool ConnectionPoolConfig `yaml:"connection_pool"`
    HealthCheck   HealthCheckConfig   `yaml:"health_check"`
}

type ClusterConfig struct {
    Name         string            `yaml:"name"`
    Provider     string            `yaml:"provider"`      // aws, gcp, azure, local
    Region       string            `yaml:"region"`
    Endpoint     string            `yaml:"endpoint"`
    Auth         AuthConfig        `yaml:"auth"`
    Capabilities []string          `yaml:"capabilities"`
    Labels       map[string]string `yaml:"labels"`
}

type AuthConfig struct {
    Method      string `yaml:"method"`       // service-account, iam, cert, kubeconfig
    Kubeconfig  string `yaml:"kubeconfig"`   // for local development
    TokenPath   string `yaml:"token_path"`   // for service account tokens
    RoleARN     string `yaml:"role_arn"`     // for AWS IAM
    ServiceAccount string `yaml:"service_account"` // for GCP
    ClientID    string `yaml:"client_id"`    // for Azure
    CertPath    string `yaml:"cert_path"`    // for certificates
    KeyPath     string `yaml:"key_path"`     // for certificates
}
```

### Example Configuration

```yaml
# config/production.yaml
kubernetes:
  default_cluster: "prod-us-east-1"
  connection_pool:
    max_idle: "5m"
    max_clients: 10
  health_check:
    interval: "30s"
    timeout: "10s"

  clusters:
    - name: "prod-us-east-1"
      provider: "aws"
      region: "us-east-1"
      auth:
        method: "iam"
        role_arn: "arn:aws:iam::123456789012:role/image-factory-role"
      capabilities: ["gpu", "arm64"]
      labels:
        environment: "production"
        region: "us-east-1"

    - name: "staging-eu-west-1"
      provider: "aws"
      region: "eu-west-1"
      auth:
        method: "service-account"
        token_path: "/secrets/staging-token"
      capabilities: ["amd64"]
      labels:
        environment: "staging"
        region: "eu-west-1"

    - name: "local-dev"
      provider: "local"
      auth:
        method: "kubeconfig"
        kubeconfig: "~/.kube/config"
      capabilities: ["amd64", "arm64"]
      labels:
        environment: "development"
```

---

## Testing Cluster Connections

### Unit Tests

```go
func TestClusterConnection(t *testing.T) {
    tests := []struct {
        name        string
        cluster     ClusterConfig
        expectError bool
    }{
        {
            name: "valid AWS cluster",
            cluster: ClusterConfig{
                Name:     "test-aws",
                Provider: "aws",
                Region:   "us-east-1",
                Auth: AuthConfig{
                    Method:  "iam",
                    RoleARN: "arn:aws:iam::123456789012:role/test-role",
                },
            },
            expectError: false,
        },
        {
            name: "invalid cluster config",
            cluster: ClusterConfig{
                Name: "invalid",
                Auth: AuthConfig{
                    Method: "unknown",
                },
            },
            expectError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            connector := NewClusterConnector(tt.cluster)
            client, err := connector.Connect(context.Background())

            if tt.expectError {
                assert.Error(t, err)
                assert.Nil(t, client)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, client)
            }
        })
    }
}
```

### Integration Tests

```go
func TestMultiClusterConnectivity(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Setup test clusters
    clusters := []ClusterConfig{
        {
            Name:     "local-kind",
            Provider: "local",
            Auth: AuthConfig{
                Method:     "kubeconfig",
                Kubeconfig: "~/.kube/config",
            },
        },
        // Add more test clusters as available
    }

    manager := NewClusterManager(clusters)

    for _, cluster := range clusters {
        t.Run(fmt.Sprintf("connect-%s", cluster.Name), func(t *testing.T) {
            client, err := manager.ConnectToCluster(cluster.Name)
            assert.NoError(t, err)
            assert.NotNil(t, client)

            // Test basic connectivity
            version, err := client.clientset.ServerVersion()
            assert.NoError(t, err)
            assert.NotEmpty(t, version.GitVersion)
        })
    }
}
```

---

## Monitoring And Observability

### Connection Metrics

```go
// internal/infrastructure/k8s/metrics.go
type ClusterMetrics struct {
    ConnectionAttempts   prometheus.Counter
    ConnectionSuccesses  prometheus.Counter
    ConnectionFailures   prometheus.Counter
    ConnectionLatency    prometheus.Histogram
    ActiveConnections    prometheus.Gauge
    ClusterHealthStatus  prometheus.Gauge
}

func (m *ClusterMetrics) RecordConnection(clusterName string, success bool, duration time.Duration) {
    m.ConnectionAttempts.WithLabelValues(clusterName).Inc()
    m.ConnectionLatency.WithLabelValues(clusterName).Observe(duration.Seconds())

    if success {
        m.ConnectionSuccesses.WithLabelValues(clusterName).Inc()
    } else {
        m.ConnectionFailures.WithLabelValues(clusterName).Inc()
    }
}
```

### Logging

```go
// Structured logging for cluster connections
type ClusterLogger struct {
    logger *zap.Logger
}

func (l *ClusterLogger) LogConnectionAttempt(clusterName, method string) {
    l.logger.Info("attempting cluster connection",
        zap.String("cluster", clusterName),
        zap.String("method", method),
        zap.Time("timestamp", time.Now()),
    )
}

func (l *ClusterLogger) LogConnectionResult(clusterName string, success bool, duration time.Duration, err error) {
    fields := []zap.Field{
        zap.String("cluster", clusterName),
        zap.Bool("success", success),
        zap.Duration("duration", duration),
    }

    if err != nil {
        fields = append(fields, zap.Error(err))
    }

    if success {
        l.logger.Info("cluster connection successful", fields...)
    } else {
        l.logger.Error("cluster connection failed", fields...)
    }
}
```

## Quick Start For Local Development

1. **Install kubectl and configure kubeconfig:**
   ```bash
   # For Docker Desktop
   kubectl config use-context docker-desktop

   # For minikube
   minikube start
   kubectl config use-context minikube

   # For kind
   kind create cluster
   kubectl config use-context kind-kind
   ```

2. **Install Tekton:**
   ```bash
   kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
   ```

3. **Configure Image Factory:**
   ```yaml
   # config/local.yaml
   kubernetes:
     clusters:
       - name: "local-dev"
         provider: "local"
         auth:
           method: "kubeconfig"
           kubeconfig: "~/.kube/config"
   ```

4. **Test Connection:**
   ```go
   client, err := NewK8sClient("~/.kube/config")
   if err != nil {
       log.Fatal(err)
   }

   version, err := client.clientset.ServerVersion()
   fmt.Printf("Connected to Kubernetes %s\n", version.GitVersion)
   ```

---

## Summary

This guide documents connectivity patterns for both local development and production multi-cluster deployments. It is most useful as implementation reference when designing secure, scalable Kubernetes access for hybrid infrastructure execution.
