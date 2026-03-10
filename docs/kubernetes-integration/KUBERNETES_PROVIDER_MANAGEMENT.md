# Kubernetes Provider Management System

This document is retained as historical reference material. It describes an earlier approach to Kubernetes provider management that has since been replaced by the unified infrastructure provider model.

## Deprecation Notice

The `kubernetes_providers` and `kubernetes_clusters` tables described here are no longer the active model in the published codebase.

**Current implementation:** see the unified `infrastructure_providers` model in the active migrations and backend domain code.

---

## Historical Overview

This document outlines the Kubernetes provider management system that enables Image Factory to connect to and manage builds across different Kubernetes distributions and cloud providers.

### Supported Providers
- **OpenShift** - Red Hat's enterprise Kubernetes platform
- **Rancher** - Multi-cluster Kubernetes management platform
- **AWS EKS** - Amazon Elastic Kubernetes Service
- **GCP GKE** - Google Kubernetes Engine
- **Azure AKS** - Azure Kubernetes Service
- **OCI OKE** - Oracle Cloud Infrastructure Container Engine for Kubernetes
- **VMware vKS** - VMware Kubernetes Service (managed Kubernetes)
- **Standard Kubernetes** - Vanilla Kubernetes (on-premises, self-managed)

---

## Provider Management Architecture

### Core Components

#### 1. Provider Registry
```go
// internal/infrastructure/k8s/providers/registry.go
type ProviderRegistry struct {
    providers map[string]Provider
}

type Provider interface {
    Name() string
    DisplayName() string
    AuthMethods() []AuthMethod
    Capabilities() []Capability
    ValidateConfig(config ProviderConfig) error
    CreateConnector(config ProviderConfig) (ClusterConnector, error)
    DiscoverClusters(ctx context.Context, config ProviderConfig) ([]ClusterInfo, error)
}

func (r *ProviderRegistry) Register(provider Provider) {
    r.providers[provider.Name()] = provider
}

func (r *ProviderRegistry) GetProvider(name string) (Provider, error) {
    provider, exists := r.providers[name]
    if !exists {
        return nil, fmt.Errorf("provider %s not found", name)
    }
    return provider, nil
}
```

#### 2. Provider-Specific Implementations
```go
// Provider interface implementation for each provider
type ProviderConfig struct {
    Name         string                 `json:"name"`
    Provider     string                 `json:"provider"`
    Region       string                 `json:"region,omitempty"`
    Auth         AuthConfig             `json:"auth"`
    Capabilities map[string]interface{} `json:"capabilities,omitempty"`
    Labels       map[string]string      `json:"labels,omitempty"`
}

type AuthConfig struct {
    Method      string            `json:"method"`
    Credentials map[string]string `json:"credentials"`
}
```

---

## Provider-Specific Implementations

### 1. AWS EKS Provider
```go
type AWSProvider struct {
    region string
}

func (p *AWSProvider) Name() string { return "aws-eks" }
func (p *AWSProvider) DisplayName() string { return "Amazon EKS" }

func (p *AWSProvider) AuthMethods() []AuthMethod {
    return []AuthMethod{
        {
            Name:        "iam",
            DisplayName: "IAM Role",
            Description: "Use AWS IAM roles for authentication",
            Fields: []AuthField{
                {
                    Name:        "role_arn",
                    Type:        "text",
                    Label:       "IAM Role ARN",
                    Placeholder: "arn:aws:iam::123456789012:role/image-factory-role",
                    Required:    true,
                },
                {
                    Name:        "external_id",
                    Type:        "text",
                    Label:       "External ID (optional)",
                    Placeholder: "image-factory-external-id",
                    Required:    false,
                },
            },
        },
        {
            Name:        "service-account",
            DisplayName: "Service Account Token",
            Description: "Use Kubernetes service account token",
            Fields: []AuthField{
                {
                    Name:        "token",
                    Type:        "password",
                    Label:       "Service Account Token",
                    Required:    true,
                },
            },
        },
    }
}

func (p *AWSProvider) Capabilities() []Capability {
    return []Capability{
        "gpu", "arm64", "fargate", "private-networking",
        "auto-scaling", "load-balancer", "iam-integration",
    }
}

func (p *AWSProvider) CreateConnector(config ProviderConfig) (ClusterConnector, error) {
    switch config.Auth.Method {
    case "iam":
        return &AWSIAMConnector{
            region:   config.Region,
            roleARN:  config.Auth.Credentials["role_arn"],
            externalID: config.Auth.Credentials["external_id"],
        }, nil
    case "service-account":
        return &ServiceAccountConnector{
            endpoint: config.Auth.Credentials["endpoint"],
            token:    config.Auth.Credentials["token"],
            caData:   []byte(config.Auth.Credentials["ca_data"]),
        }, nil
    default:
        return nil, fmt.Errorf("unsupported auth method: %s", config.Auth.Method)
    }
}

func (p *AWSProvider) DiscoverClusters(ctx context.Context, config ProviderConfig) ([]ClusterInfo, error) {
    // Use AWS SDK to discover EKS clusters
    sess := session.Must(session.NewSession(&aws.Config{
        Region: aws.String(config.Region),
    }))

    eksClient := eks.New(sess)

    result, err := eksClient.ListClusters(&eks.ListClustersInput{})
    if err != nil {
        return nil, fmt.Errorf("failed to list EKS clusters: %w", err)
    }

    var clusters []ClusterInfo
    for _, clusterName := range result.Clusters {
        cluster := &ClusterInfo{
            Name:       *clusterName,
            Provider:   "aws-eks",
            Region:     config.Region,
            AuthMethod: config.Auth.Method,
            Status:     ClusterUnknown,
        }
        clusters = append(clusters, *cluster)
    }

    return clusters, nil
}
```

### 2. OpenShift Provider
```go
type OpenShiftProvider struct{}

func (p *OpenShiftProvider) Name() string { return "openshift" }
func (p *OpenShiftProvider) DisplayName() string { return "Red Hat OpenShift" }

func (p *OpenShiftProvider) AuthMethods() []AuthMethod {
    return []AuthMethod{
        {
            Name:        "service-account",
            DisplayName: "Service Account Token",
            Description: "Use OpenShift service account token",
            Fields: []AuthField{
                {
                    Name:        "token",
                    Type:        "password",
                    Label:       "Service Account Token",
                    Required:    true,
                },
            },
        },
        {
            Name:        "oauth",
            DisplayName: "OAuth Token",
            Description: "Use OpenShift OAuth access token",
            Fields: []AuthField{
                {
                    Name:        "token",
                    Type:        "password",
                    Label:       "OAuth Access Token",
                    Required:    true,
                },
            },
        },
    }
}

func (p *OpenShiftProvider) Capabilities() []Capability {
    return []Capability{
        "security-context-constraints", "routes", "buildconfigs",
        "imagestreams", "deploymentconfigs", "operators",
        "multi-tenancy", "compliance", "enterprise-support",
    }
}

func (p *OpenShiftProvider) CreateConnector(config ProviderConfig) (ClusterConnector, error) {
    return &OpenShiftConnector{
        endpoint: config.Auth.Credentials["endpoint"],
        token:    config.Auth.Credentials["token"],
        // OpenShift-specific configuration
        project: config.Capabilities["project"].(string),
    }, nil
}
```

### 3. Rancher Provider
```go
type RancherProvider struct {
    endpoint string
}

func (p *RancherProvider) Name() string { return "rancher" }
func (p *RancherProvider) DisplayName() string { return "Rancher" }

func (p *RancherProvider) AuthMethods() []AuthMethod {
    return []AuthMethod{
        {
            Name:        "api-token",
            DisplayName: "API Token",
            Description: "Use Rancher API token for authentication",
            Fields: []AuthField{
                {
                    Name:        "token",
                    Type:        "password",
                    Label:       "Rancher API Token",
                    Required:    true,
                },
                {
                    Name:        "cluster_id",
                    Type:        "text",
                    Label:       "Cluster ID",
                    Placeholder: "c-xxxxx",
                    Required:    true,
                },
            },
        },
    }
}

func (p *RancherProvider) Capabilities() []Capability {
    return []Capability{
        "multi-cluster", "cattle", "longhorn", "monitoring",
        "logging", "backup", "security", "policy",
    }
}
```

### 4. Standard Kubernetes Provider
```go
type StandardKubernetesProvider struct{}

func (p *StandardKubernetesProvider) Name() string { return "kubernetes" }
func (p *StandardKubernetesProvider) DisplayName() string { return "Standard Kubernetes" }

func (p *StandardKubernetesProvider) AuthMethods() []AuthMethod {
    return []AuthMethod{
        {
            Name:        "kubeconfig",
            DisplayName: "Kubeconfig",
            Description: "Use kubeconfig file for authentication",
            Fields: []AuthField{
                {
                    Name:        "kubeconfig",
                    Type:        "file",
                    Label:       "Kubeconfig File",
                    Required:    true,
                },
                {
                    Name:        "context",
                    Type:        "text",
                    Label:       "Context (optional)",
                    Placeholder: "my-cluster",
                    Required:    false,
                },
            },
        },
        {
            Name:        "service-account",
            DisplayName: "Service Account",
            Description: "Use service account token and certificate",
            Fields: []AuthField{
                {
                    Name:        "endpoint",
                    Type:        "text",
                    Label:       "API Server Endpoint",
                    Placeholder: "https://api.example.com:6443",
                    Required:    true,
                },
                {
                    Name:        "token",
                    Type:        "password",
                    Label:       "Service Account Token",
                    Required:    true,
                },
                {
                    Name:        "ca_data",
                    Type:        "textarea",
                    Label:       "CA Certificate Data",
                    Required:    true,
                },
            },
        },
        {
            Name:        "client-cert",
            DisplayName: "Client Certificate",
            Description: "Use client certificate for authentication",
            Fields: []AuthField{
                {
                    Name:        "endpoint",
                    Type:        "text",
                    Label:       "API Server Endpoint",
                    Required:    true,
                },
                {
                    Name:        "client_cert",
                    Type:        "textarea",
                    Label:       "Client Certificate",
                    Required:    true,
                },
                {
                    Name:        "client_key",
                    Type:        "textarea",
                    Label:       "Client Key",
                    Required:    true,
                },
                {
                    Name:        "ca_data",
                    Type:        "textarea",
                    Label:       "CA Certificate Data",
                    Required:    true,
                },
            },
        },
    }
}

func (p *StandardKubernetesProvider) Capabilities() []Capability {
    return []Capability{
        "vanilla-k8s", "custom-cni", "custom-scheduler",
        "bare-metal", "on-premises", "self-managed",
    }
}
```

### 5. VMware vKS Provider
```go
type VMwareProvider struct {
    orgID string
}

func (p *VMwareProvider) Name() string { return "vmware-vks" }
func (p *VMwareProvider) DisplayName() string { return "VMware Kubernetes Service" }

func (p *VMwareProvider) AuthMethods() []AuthMethod {
    return []AuthMethod{
        {
            Name:        "api-token",
            DisplayName: "API Token",
            Description: "Use VMware Cloud Services API token for authentication",
            Fields: []AuthField{
                {
                    Name:        "api_token",
                    Type:        "password",
                    Label:       "VMware API Token",
                    Placeholder: "vmware-api-token-here",
                    Required:    true,
                },
                {
                    Name:        "org_id",
                    Type:        "text",
                    Label:       "Organization ID",
                    Placeholder: "vmware-org-id",
                    Required:    true,
                },
            },
        },
        {
            Name:        "service-account",
            DisplayName: "Service Account Token",
            Description: "Use Kubernetes service account token",
            Fields: []AuthField{
                {
                    Name:        "token",
                    Type:        "password",
                    Label:       "Service Account Token",
                    Required:    true,
                },
            },
        },
    }
}

func (p *VMwareProvider) Capabilities() []Capability {
    return []Capability{
        "vsphere-integration", "tanzu", "nsx", "vrealize",
        "multi-cloud", "disaster-recovery", "compliance",
        "enterprise-support", "hybrid-cloud",
    }
}

func (p *VMwareProvider) CreateConnector(config ProviderConfig) (ClusterConnector, error) {
    switch config.Auth.Method {
    case "api-token":
        return &VMwareAPIConnector{
            apiToken: config.Auth.Credentials["api_token"],
            orgID:    config.Auth.Credentials["org_id"],
        }, nil
    case "service-account":
        return &ServiceAccountConnector{
            endpoint: config.Auth.Credentials["endpoint"],
            token:    config.Auth.Credentials["token"],
            caData:   []byte(config.Auth.Credentials["ca_data"]),
        }, nil
    default:
        return nil, fmt.Errorf("unsupported auth method: %s", config.Auth.Method)
    }
}

func (p *VMwareProvider) DiscoverClusters(ctx context.Context, config ProviderConfig) ([]ClusterInfo, error) {
    // Use VMware Cloud Services API to discover vKS clusters
    client := &http.Client{}
    req, err := http.NewRequest("GET", "https://api.vmware.cloud/vks/orgs/"+config.Auth.Credentials["org_id"]+"/clusters", nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create discovery request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+config.Auth.Credentials["api_token"])
    req.Header.Set("Content-Type", "application/json")

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to discover clusters: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("discovery API returned status: %s", resp.Status)
    }

    var response struct {
        Clusters []struct {
            ID     string `json:"id"`
            Name   string `json:"name"`
            Status string `json:"status"`
            Region string `json:"region"`
        } `json:"clusters"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, fmt.Errorf("failed to decode discovery response: %w", err)
    }

    var clusters []ClusterInfo
    for _, cluster := range response.Clusters {
        clusters = append(clusters, ClusterInfo{
            Name:       cluster.Name,
            Provider:   "vmware-vks",
            Region:     cluster.Region,
            AuthMethod: config.Auth.Method,
            Status:     ClusterStatus(cluster.Status),
        })
    }

    return clusters, nil
}
```

---

## 🎨 Frontend Provider Selection UI

### Provider Selection Component
```tsx
// src/components/admin/kubernetes/ProviderSelector.tsx
const ProviderSelector: React.FC<{
    value: string
    onChange: (provider: string) => void
    onAuthMethodChange: (method: string) => void
}> = ({ value, onChange, onAuthMethodChange }) => {
    const [providers, setProviders] = useState<ProviderInfo[]>([])
    const [selectedProvider, setSelectedProvider] = useState<ProviderInfo | null>(null)

    useEffect(() => {
        // Fetch available providers from backend
        kubernetesService.getProviders().then(setProviders)
    }, [])

    const handleProviderChange = (providerName: string) => {
        const provider = providers.find(p => p.name === providerName)
        setSelectedProvider(provider)
        onChange(providerName)
        onAuthMethodChange(provider?.authMethods[0]?.name || '')
    }

    return (
        <div className="space-y-4">
            {/* Provider Selection */}
            <div>
                <label className="block text-sm font-medium text-gray-700">
                    Kubernetes Provider
                </label>
                <select
                    value={value}
                    onChange={(e) => handleProviderChange(e.target.value)}
                    className="mt-1 block w-full rounded-md border-gray-300 shadow-sm"
                >
                    <option value="">Select Provider</option>
                    {providers.map(provider => (
                        <option key={provider.name} value={provider.name}>
                            {provider.displayName}
                        </option>
                    ))}
                </select>
            </div>

            {/* Provider Description */}
            {selectedProvider && (
                <div className="bg-blue-50 p-4 rounded-md">
                    <h4 className="text-sm font-medium text-blue-800">
                        {selectedProvider.displayName}
                    </h4>
                    <p className="text-sm text-blue-600 mt-1">
                        {selectedProvider.description}
                    </p>
                    <div className="mt-2">
                        <span className="text-xs text-blue-500">Capabilities:</span>
                        <div className="flex flex-wrap gap-1 mt-1">
                            {selectedProvider.capabilities.map(cap => (
                                <span key={cap} className="inline-flex items-center px-2 py-1 rounded-full text-xs bg-blue-100 text-blue-800">
                                    {cap}
                                </span>
                            ))}
                        </div>
                    </div>
                </div>
            )}
        </div>
    )
}
```

### Authentication Configuration Component
```tsx
// src/components/admin/kubernetes/AuthConfigForm.tsx
const AuthConfigForm: React.FC<{
    provider: string
    authMethod: string
    onAuthMethodChange: (method: string) => void
    onConfigChange: (config: AuthConfig) => void
}> = ({ provider, authMethod, onAuthMethodChange, onConfigChange }) => {
    const [authMethods, setAuthMethods] = useState<AuthMethod[]>([])
    const [selectedMethod, setSelectedMethod] = useState<AuthMethod | null>(null)

    useEffect(() => {
        if (provider) {
            kubernetesService.getAuthMethods(provider).then(setAuthMethods)
        }
    }, [provider])

    useEffect(() => {
        const method = authMethods.find(m => m.name === authMethod)
        setSelectedMethod(method)
    }, [authMethod, authMethods])

    const handleMethodChange = (methodName: string) => {
        const method = authMethods.find(m => m.name === methodName)
        setSelectedMethod(method)
        onAuthMethodChange(methodName)
    }

    return (
        <div className="space-y-4">
            {/* Auth Method Selection */}
            <div>
                <label className="block text-sm font-medium text-gray-700">
                    Authentication Method
                </label>
                <select
                    value={authMethod}
                    onChange={(e) => handleMethodChange(e.target.value)}
                    className="mt-1 block w-full rounded-md border-gray-300 shadow-sm"
                >
                    <option value="">Select Method</option>
                    {authMethods.map(method => (
                        <option key={method.name} value={method.name}>
                            {method.displayName}
                        </option>
                    ))}
                </select>
            </div>

            {/* Auth Method Description */}
            {selectedMethod && (
                <div className="bg-gray-50 p-4 rounded-md">
                    <p className="text-sm text-gray-600">
                        {selectedMethod.description}
                    </p>
                </div>
            )}

            {/* Dynamic Auth Fields */}
            {selectedMethod && (
                <div className="space-y-4">
                    {selectedMethod.fields.map(field => (
                        <AuthFieldComponent
                            key={field.name}
                            field={field}
                            onChange={(value) => {
                                const config = { ...selectedMethod }
                                config[field.name] = value
                                onConfigChange(config)
                            }}
                        />
                    ))}
                </div>
            )}
        </div>
    )
}
```

---

## Backend API Endpoints

### Provider Management API
```go
// internal/adapters/http/kubernetes.go
type KubernetesHandler struct {
    providerRegistry *ProviderRegistry
    clusterManager   *ClusterManager
}

func (h *KubernetesHandler) GetProviders(w http.ResponseWriter, r *http.Request) {
    providers := h.providerRegistry.ListProviders()

    response := make([]ProviderInfo, 0, len(providers))
    for _, provider := range providers {
        response = append(response, ProviderInfo{
            Name:         provider.Name(),
            DisplayName:  provider.DisplayName(),
            AuthMethods:  provider.AuthMethods(),
            Capabilities: provider.Capabilities(),
        })
    }

    json.NewEncoder(w).Encode(response)
}

func (h *KubernetesHandler) GetAuthMethods(w http.ResponseWriter, r *http.Request) {
    providerName := r.URL.Query().Get("provider")
    if providerName == "" {
        http.Error(w, "provider parameter required", http.StatusBadRequest)
        return
    }

    provider, err := h.providerRegistry.GetProvider(providerName)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    json.NewEncoder(w).Encode(provider.AuthMethods())
}

func (h *KubernetesHandler) AddCluster(w http.ResponseWriter, r *http.Request) {
    var req AddClusterRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Validate provider configuration
    provider, err := h.providerRegistry.GetProvider(req.Provider)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if err := provider.ValidateConfig(req.Config); err != nil {
        http.Error(w, fmt.Sprintf("invalid config: %v", err), http.StatusBadRequest)
        return
    }

    // Create cluster connector
    connector, err := provider.CreateConnector(req.Config)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Test connection
    if err := h.testClusterConnection(connector); err != nil {
        http.Error(w, fmt.Sprintf("connection test failed: %v", err), http.StatusBadRequest)
        return
    }

    // Save cluster configuration
    cluster := &Cluster{
        Name:     req.Name,
        Provider: req.Provider,
        Config:   req.Config,
        Status:   ClusterHealthy,
    }

    if err := h.clusterManager.AddCluster(cluster); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(cluster)
}

func (h *KubernetesHandler) DiscoverClusters(w http.ResponseWriter, r *http.Request) {
    var req DiscoverClustersRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    provider, err := h.providerRegistry.GetProvider(req.Provider)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    clusters, err := provider.DiscoverClusters(r.Context(), req.Config)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(clusters)
}
```

---

## Database Schema

### Provider and Cluster Tables
```sql
-- Kubernetes providers
CREATE TABLE kubernetes_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    capabilities JSONB DEFAULT '[]',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Kubernetes clusters
CREATE TABLE kubernetes_clusters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    provider_id UUID NOT NULL REFERENCES kubernetes_providers(id),
    region VARCHAR(50),
    endpoint VARCHAR(255),
    config JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'unknown',
    capabilities JSONB DEFAULT '[]',
    labels JSONB DEFAULT '{}',
    last_health_check TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Authentication configurations
CREATE TABLE cluster_auth_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL REFERENCES kubernetes_clusters(id),
    auth_method VARCHAR(50) NOT NULL,
    credentials JSONB NOT NULL DEFAULT '{}', -- Encrypted
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE UNIQUE INDEX idx_clusters_name ON kubernetes_clusters(name);
CREATE INDEX idx_clusters_provider ON kubernetes_clusters(provider_id);
CREATE INDEX idx_clusters_status ON kubernetes_clusters(status);
CREATE INDEX idx_auth_cluster ON cluster_auth_configs(cluster_id);
```

---

## Security Considerations

### Credential Encryption
```go
// internal/infrastructure/security/encryption.go
type CredentialEncryptor struct {
    key []byte
}

func (e *CredentialEncryptor) Encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }

    // Use GCM for authenticated encryption
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }

    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *CredentialEncryptor) Decrypt(ciphertext string) (string, error) {
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }

    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return "", fmt.Errorf("ciphertext too short")
    }

    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}
```

### RBAC for Cluster Access
```yaml
# Cluster role for provider-specific access
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: image-factory-provider-access
rules:
# Base Kubernetes permissions
- apiGroups: [""]
  resources: ["pods", "pods/log", "configmaps", "secrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
# Tekton permissions
- apiGroups: ["tekton.dev"]
  resources: ["pipelineruns", "taskruns", "pipelines", "tasks"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
# Provider-specific permissions
- apiGroups: ["route.openshift.io"]  # OpenShift routes
  resources: ["routes"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["management.cattle.io"]  # Rancher management
  resources: ["clusters"]
  verbs: ["get", "list", "watch"]
```

---

## Testing Strategy

### Provider Unit Tests
```go
func TestAWSProvider(t *testing.T) {
    provider := &AWSProvider{region: "us-east-1"}

    // Test provider metadata
    assert.Equal(t, "aws-eks", provider.Name())
    assert.Equal(t, "Amazon EKS", provider.DisplayName())

    // Test auth methods
    authMethods := provider.AuthMethods()
    assert.Contains(t, authMethods, AuthMethod{Name: "iam"})
    assert.Contains(t, authMethods, AuthMethod{Name: "service-account"})

    // Test capabilities
    capabilities := provider.Capabilities()
    assert.Contains(t, capabilities, "gpu")
    assert.Contains(t, capabilities, "iam-integration")
}

func TestProviderValidation(t *testing.T) {
    provider := &AWSProvider{}

    tests := []struct {
        name    string
        config  ProviderConfig
        wantErr bool
    }{
        {
            name: "valid IAM config",
            config: ProviderConfig{
                Auth: AuthConfig{
                    Method: "iam",
                    Credentials: map[string]string{
                        "role_arn": "arn:aws:iam::123456789012:role/test-role",
                    },
                },
            },
            wantErr: false,
        },
        {
            name: "invalid IAM config - missing role",
            config: ProviderConfig{
                Auth: AuthConfig{
                    Method: "iam",
                    Credentials: map[string]string{},
                },
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func t) {
            err := provider.ValidateConfig(tt.config)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Integration Tests
```go
func TestProviderConnectivity(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Test AWS EKS connectivity
    awsProvider := &AWSProvider{region: "us-east-1"}
    config := ProviderConfig{
        Auth: AuthConfig{
            Method: "iam",
            Credentials: map[string]string{
                "role_arn": os.Getenv("AWS_ROLE_ARN"),
            },
        },
    }

    connector, err := awsProvider.CreateConnector(config)
    assert.NoError(t, err)

    // Test connection
    client, err := connector.Connect(context.Background())
    assert.NoError(t, err)
    assert.NotNil(t, client)

    // Test basic operations
    version, err := client.clientset.ServerVersion()
    assert.NoError(t, err)
    assert.NotEmpty(t, version.GitVersion)
}
```

## API Specification

### Provider Endpoints
```yaml
paths:
  /api/v1/kubernetes/providers:
    get:
      summary: Get available Kubernetes providers
      responses:
        '200':
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Provider'

  /api/v1/kubernetes/providers/{provider}/auth-methods:
    get:
      summary: Get authentication methods for a provider
      parameters:
        - name: provider
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/AuthMethod'

  /api/v1/kubernetes/clusters:
    post:
      summary: Add a new Kubernetes cluster
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/AddClusterRequest'
      responses:
        '201':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Cluster'

  /api/v1/kubernetes/clusters/discover:
    post:
      summary: Discover clusters for a provider
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/DiscoverClustersRequest'
      responses:
        '200':
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/ClusterInfo'
```

---

## Summary
This historical provider management design enabled Image Factory to:

1. **Support Multiple Kubernetes Distributions**: OpenShift, Rancher, AWS EKS, GCP GKE, Azure AKS, OCI OKE, VMware vKS, Standard Kubernetes
2. **Provider-Specific Authentication**: IAM for cloud providers, service accounts for on-premises, OAuth for OpenShift
3. **Dynamic Configuration**: UI adapts based on selected provider and auth method
4. **Secure Credential Management**: Encrypted storage with proper access controls
5. **Cluster Discovery**: Automatic discovery of clusters in cloud environments
6. **Capability Awareness**: Provider-specific features and limitations

The approach provided a unified interface for managing diverse Kubernetes environments while maintaining security and usability. For the active model, refer to the current infrastructure provider documentation and code.
