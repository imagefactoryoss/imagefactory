import { providerRegistry } from "@/components/admin/providers/ProviderRegistry";
import {
  CopyableCodeBlock,
  TooltipDrawer,
} from "@/components/admin/providers/TooltipDrawer";
import { useCanManageAdmin } from "@/hooks/useAccess";
import { infrastructureService } from "@/services/infrastructureService";
import {
  InfrastructureProvider,
  InfrastructureProviderType,
  ProviderPrepareSummaryBatchMetrics,
} from "@/types";
import {
  AlertCircle,
  Check,
  Edit2,
  HelpCircle,
  Plus,
  RefreshCw,
  Server,
  TestTube,
  Trash2,
  X,
  Zap,
} from "lucide-react";
import React, { useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";

interface ProviderFormData {
  provider_type: InfrastructureProviderType;
  name: string;
  display_name: string;
  config: Record<string, any>;
  capabilities: string[];
  is_global: boolean;
  bootstrap_mode: "image_factory_managed" | "self_managed";
  credential_scope:
    | "cluster_admin"
    | "namespace_admin"
    | "read_only"
    | "unknown";
  target_namespace: string;
}

type InfrastructureNavigationState = {
  editingProvider?: InfrastructureProvider;
  editingProviderId?: string;
};

const normalizeBootstrapModeForm = (
  value: string | undefined,
): ProviderFormData["bootstrap_mode"] => {
  return value === "self_managed" ? "self_managed" : "image_factory_managed";
};

const normalizeCredentialScopeForm = (
  value: string | undefined,
): ProviderFormData["credential_scope"] => {
  if (
    value === "cluster_admin" ||
    value === "namespace_admin" ||
    value === "read_only" ||
    value === "unknown"
  ) {
    return value;
  }
  return "unknown";
};

const kubernetesNamespacePattern = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;

const statusColors: Record<string, string> = {
  online: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
  offline: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
  maintenance:
    "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
  pending: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
};

const prepareSeverityClasses: Record<string, string> = {
  error: "ring-1 ring-red-500/40 dark:ring-red-400/40",
  warn: "ring-1 ring-amber-500/40 dark:ring-amber-400/40",
  info: "ring-1 ring-blue-500/35 dark:ring-blue-400/35",
};

const providerTypeIcons: Record<InfrastructureProviderType, React.ReactNode> = {
  kubernetes: <Zap className="h-4 w-4" />,
  "aws-eks": <Server className="h-4 w-4" />, // AWS icon placeholder
  "gcp-gke": <Server className="h-4 w-4" />, // GCP icon placeholder
  "azure-aks": <Server className="h-4 w-4" />, // Azure icon placeholder
  "oci-oke": <Server className="h-4 w-4" />, // Oracle icon placeholder
  "vmware-vks": <Server className="h-4 w-4" />, // VMware icon placeholder
  openshift: <Server className="h-4 w-4" />, // OpenShift icon placeholder
  rancher: <Server className="h-4 w-4" />, // Rancher icon placeholder
  build_nodes: <Server className="h-4 w-4" />,
};

const kubernetesProviderTypes: InfrastructureProviderType[] = [
  "kubernetes",
  "aws-eks",
  "gcp-gke",
  "azure-aks",
  "oci-oke",
  "vmware-vks",
  "openshift",
  "rancher",
];

const isKubernetesProviderType = (
  providerType: InfrastructureProviderType,
): boolean => {
  return kubernetesProviderTypes.includes(providerType);
};

const normalizeProviderType = (
  value: string | undefined,
): InfrastructureProviderType => {
  const normalized = (value || "").trim().toLowerCase().replace(/_/g, "-");
  return providerRegistry.isProviderTypeSupported(normalized)
    ? normalized
    : "kubernetes";
};

const blockedByLabel = (key: string): string => {
  switch (key) {
    case "cluster_capacity":
      return "Cluster capacity unavailable";
    case "provider_not_ready":
      return "Provider readiness check failed";
    case "provider_status_offline":
      return "Provider is offline";
    case "provider_status_maintenance":
      return "Provider in maintenance";
    case "provider_status_pending":
      return "Provider still pending";
    default:
      return key.replace(/_/g, " ");
  }
};

const systemNamespaceDefault = "imagefactory-system";
const authConfigKeys = new Set([
  "auth_method",
  "apiServer",
  "endpoint",
  "cluster_endpoint",
  "token",
  "service_account_token",
  "oauth_token",
  "api_token",
  "ca_cert",
  "client_cert",
  "client_key",
  "kubeconfig_path",
  "kubeconfig",
]);

const flattenRuntimeAuthForForm = (
  config: Record<string, any> | undefined,
): Record<string, any> => {
  const source = config || {};
  const runtimeAuth =
    source.runtime_auth && typeof source.runtime_auth === "object"
      ? source.runtime_auth
      : {};
  const bootstrapAuth =
    source.bootstrap_auth && typeof source.bootstrap_auth === "object"
      ? source.bootstrap_auth
      : {};
  const legacyAuth: Record<string, any> = {};
  authConfigKeys.forEach((key) => {
    const value = source[key];
    if (value !== undefined && value !== null && value !== "") {
      legacyAuth[key] = value;
    }
  });
  const normalizedRuntimeAuth = {
    auth_method: "token",
    ...legacyAuth,
    ...runtimeAuth,
  };
  const normalizedBootstrapAuth = {
    ...normalizedRuntimeAuth,
    ...bootstrapAuth,
  };
  return {
    ...source,
    runtime_auth: normalizedRuntimeAuth,
    bootstrap_auth: normalizedBootstrapAuth,
  };
};

const normalizeKubernetesProviderConfig = (
  rawConfig: Record<string, any>,
  bootstrapMode: ProviderFormData["bootstrap_mode"],
): Record<string, any> => {
  const source = { ...(rawConfig || {}) };
  const explicitRuntimeAuth =
    source.runtime_auth && typeof source.runtime_auth === "object"
      ? { ...(source.runtime_auth as Record<string, any>) }
      : {};
  const runtimeAuth: Record<string, any> = { ...explicitRuntimeAuth };
  authConfigKeys.forEach((key) => {
    const value = source[key];
    if (value !== undefined && value !== null && value !== "") {
      runtimeAuth[key] = value;
    }
  });
  runtimeAuth.auth_method = runtimeAuth.auth_method || "token";
  const hasMeaningfulAuthConfig = (auth: Record<string, any>): boolean => {
    const method = (auth?.auth_method || "").trim();
    if (!method) return false;
    if (method === "kubeconfig") {
      return Boolean(auth?.kubeconfig_path || auth?.kubeconfig);
    }
    if (
      method === "token" ||
      method === "service-account" ||
      method === "service_account" ||
      method === "oauth"
    ) {
      return Boolean(
        (auth?.endpoint || auth?.apiServer || auth?.cluster_endpoint) &&
          (auth?.token ||
            auth?.service_account_token ||
            auth?.oauth_token ||
            auth?.api_token),
      );
    }
    if (method === "client-cert") {
      return Boolean(
        (auth?.endpoint || auth?.apiServer || auth?.cluster_endpoint) &&
          auth?.client_cert &&
          auth?.client_key,
      );
    }
    return Object.keys(auth || {}).some(
      (key) => key !== "auth_method" && auth[key],
    );
  };
  const runtimeAuthConfigured = hasMeaningfulAuthConfig(runtimeAuth);

  const normalized: Record<string, any> = { ...source };
  authConfigKeys.forEach((key) => {
    delete normalized[key];
  });
  delete normalized.runtime_auth;
  delete normalized.bootstrap_auth;

  normalized.system_namespace =
    typeof source.system_namespace === "string" &&
    source.system_namespace.trim()
      ? source.system_namespace.trim()
      : systemNamespaceDefault;

  if (bootstrapMode === "image_factory_managed") {
    const explicitBootstrapAuth =
      source.bootstrap_auth && typeof source.bootstrap_auth === "object"
        ? { ...(source.bootstrap_auth as Record<string, any>) }
        : {};
    const bootstrapAuth = {
      auth_method: explicitBootstrapAuth.auth_method || "token",
      ...explicitBootstrapAuth,
    };
    normalized.bootstrap_auth = bootstrapAuth;
    if (runtimeAuthConfigured) {
      normalized.runtime_auth = runtimeAuth;
    }
  } else {
    normalized.runtime_auth = runtimeAuth;
  }

  return normalized;
};

const bootstrapNamespaceTemplate = (namespace: string) => `apiVersion: v1
kind: Namespace
metadata:
  name: ${namespace}
  labels:
    app: image-factory
    managedBy: image-factory
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: image-factory-bootstrap-sa
  namespace: ${namespace}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: image-factory-runtime-sa
  namespace: ${namespace}`;

const bootstrapClusterRoleTemplate = (
  namespace: string,
) => `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: image-factory-bootstrap-role
rules:
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list", "create", "patch", "update"]
- apiGroups: ["rbac.authorization.k8s.io"]
  resources: ["roles", "rolebindings"]
  verbs: ["get", "list", "watch", "create", "patch", "update"]
- apiGroups: ["tekton.dev"]
  resources: ["tasks", "pipelines"]
  verbs: ["get", "list", "watch", "create", "patch", "update"]
- apiGroups: ["tekton.dev"]
  resources: ["pipelineruns"]
  verbs: ["create", "get", "list", "watch", "patch", "update"]
- apiGroups: [""]
  resources: ["pods", "secrets", "configmaps", "persistentvolumeclaims"]
  verbs: ["create", "get", "list", "watch", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: image-factory-bootstrap-binding
subjects:
- kind: ServiceAccount
  name: image-factory-bootstrap-sa
  namespace: ${namespace}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: image-factory-bootstrap-role`;

const runtimeRoleTemplate = (
  systemNamespace: string,
) => `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: image-factory-runtime-role
  namespace: image-factory-tenant1234
rules:
- apiGroups: ["tekton.dev"]
  resources: ["tasks", "pipelines"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["tekton.dev"]
  resources: ["taskruns"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["tekton.dev"]
  resources: ["pipelineruns"]
  verbs: ["create", "get", "list", "watch", "delete"]
- apiGroups: [""]
  resources: ["pods", "pods/log", "pods/exec", "secrets", "configmaps", "serviceaccounts", "events", "persistentvolumeclaims"]
  verbs: ["create", "get", "list", "watch", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: image-factory-runtime-binding
  namespace: image-factory-tenant1234
subjects:
- kind: ServiceAccount
  name: image-factory-runtime-sa
  namespace: ${systemNamespace}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: image-factory-runtime-role`;

const serviceAccountTokenTemplate = (namespace: string) => `# Kubernetes v1.24+ typically does not auto-create long-lived service account token Secrets.
# Use TokenRequest via kubectl (recommended):
#
# 1 year = 8760h (cluster policies may cap max duration)
kubectl -n ${namespace} create token image-factory-bootstrap-sa --duration=8760h
kubectl -n ${namespace} create token image-factory-runtime-sa --duration=8760h`;

/**
 * AdminInfrastructureProvidersPage - Admin component for managing infrastructure providers
 * Allows admins to create, configure, test, and manage infrastructure providers
 */
const AdminInfrastructureProvidersPage: React.FC = () => {
  const canManageAdmin = useCanManageAdmin();
  const navigate = useNavigate();
  const location = useLocation();
  const [providers, setProviders] = useState<InfrastructureProvider[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editingProvider, setEditingProvider] =
    useState<InfrastructureProvider | null>(null);
  const [formData, setFormData] = useState<ProviderFormData>({
    provider_type: "kubernetes",
    name: "",
    display_name: "",
    config: {
      runtime_auth: { auth_method: "token" },
      bootstrap_auth: { auth_method: "token" },
      tekton_enabled: true,
      quarantine_dispatch_enabled: false,
    },
    capabilities: [],
    is_global: false,
    bootstrap_mode: "image_factory_managed",
    credential_scope: "unknown",
    target_namespace: systemNamespaceDefault,
  });
  const [formError, setFormError] = useState<string | null>(null);
  const [formSuccess, setFormSuccess] = useState<string | null>(null);
  const [testingConnection, setTestingConnection] = useState<string | null>(
    null,
  );
  const [providerComponent, setProviderComponent] =
    useState<React.ComponentType<any> | null>(null);
  const [loadingProviderForm, setLoadingProviderForm] = useState(false);
  const [prepareBatchMetrics, setPrepareBatchMetrics] =
    useState<ProviderPrepareSummaryBatchMetrics | null>(null);
  const [showBootstrapRBACDrawer, setShowBootstrapRBACDrawer] = useState(false);
  const effectiveTargetNamespace =
    (formData.target_namespace || "").trim() || systemNamespaceDefault;

  const validateAuthConfig = (
    auth: Record<string, any> | undefined,
    label: string,
  ): string | null => {
    const method = (auth?.auth_method || "").trim();
    if (!method) return `${label} authentication method is required`;
    if (method === "kubeconfig") {
      if (!auth?.kubeconfig_path && !auth?.kubeconfig) {
        return `${label} kubeconfig path or inline kubeconfig is required`;
      }
      return null;
    }
    if (
      method === "token" ||
      method === "service-account" ||
      method === "service_account" ||
      method === "oauth"
    ) {
      if (!auth?.endpoint && !auth?.apiServer && !auth?.cluster_endpoint) {
        return `${label} endpoint is required`;
      }
      if (
        !auth?.token &&
        !auth?.service_account_token &&
        !auth?.oauth_token &&
        !auth?.api_token
      ) {
        return `${label} token is required`;
      }
      return null;
    }
    if (method === "client-cert") {
      if (!auth?.endpoint && !auth?.apiServer && !auth?.cluster_endpoint) {
        return `${label} endpoint is required`;
      }
      if (!auth?.client_cert) return `${label} client certificate is required`;
      if (!auth?.client_key) return `${label} client key is required`;
      return null;
    }
    return null;
  };

  const validateTargetNamespace = (
    value: string | undefined,
  ): string | null => {
    const ns = (value || "").trim();
    if (!ns) return "Target namespace is required";
    if (ns.length > 63)
      return "Target namespace must be 63 characters or fewer";
    if (!kubernetesNamespacePattern.test(ns)) {
      return "Target namespace must be lowercase alphanumeric and '-' only";
    }
    return null;
  };

  // Load providers
  const loadProviders = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await infrastructureService.getProviders();
      const loadedProviders = response.data.map((provider) => ({
        ...provider,
        provider_type: normalizeProviderType(provider.provider_type),
      }));
      setProviders(loadedProviders);
      if (loadedProviders.length === 0) {
        setPrepareBatchMetrics(null);
        return;
      }
      try {
        const providerIds = loadedProviders
          .slice(0, 200)
          .map((provider) => provider.id);
        const summaryPayload =
          await infrastructureService.getProviderPrepareSummariesWithMetrics(
            providerIds,
            true,
          );
        setPrepareBatchMetrics(summaryPayload.batch_metrics || null);
      } catch (metricsError) {
        console.warn(
          "Failed to load provider prepare batch metrics",
          metricsError,
        );
        setPrepareBatchMetrics(null);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load providers");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadProviders();
  }, [loadProviders]);

  const openEditProvider = useCallback(
    (provider: InfrastructureProvider) => {
      const normalizedProvider: InfrastructureProvider = {
        ...provider,
        provider_type: normalizeProviderType(provider.provider_type),
      };
      setEditingProvider(normalizedProvider);
      setFormData({
        provider_type: normalizedProvider.provider_type,
        name: normalizedProvider.name,
        display_name: normalizedProvider.display_name,
        config: flattenRuntimeAuthForForm(normalizedProvider.config || {}),
        capabilities: normalizedProvider.capabilities || [],
        is_global: normalizedProvider.is_global || false,
        bootstrap_mode: normalizeBootstrapModeForm(normalizedProvider.bootstrap_mode),
        credential_scope: normalizeCredentialScopeForm(normalizedProvider.credential_scope),
        target_namespace: normalizedProvider.target_namespace || systemNamespaceDefault,
      });
      setShowForm(true);
    },
    [],
  );

  // Check for editing provider from navigation state
  useEffect(() => {
    const navState = (location.state || {}) as InfrastructureNavigationState;
    if (!navState.editingProvider && !navState.editingProviderId) {
      return;
    }

    const openFromNavState = async () => {
      try {
        if (navState.editingProviderId) {
          const provider = await infrastructureService.getProvider(
            navState.editingProviderId,
          );
          openEditProvider(provider);
          return;
        }
        if (navState.editingProvider) {
          openEditProvider(navState.editingProvider);
        }
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : "Failed to load provider for edit form",
        );
      } finally {
        navigate(location.pathname, { replace: true, state: null });
      }
    };

    void openFromNavState();
  }, [location.pathname, location.state, navigate, openEditProvider]);

  // Load provider component when provider type changes
  useEffect(() => {
    const loadProviderComponent = async () => {
      if (formData.provider_type) {
        setLoadingProviderForm(true);
        try {
          const component = await providerRegistry.getProviderForm(
            formData.provider_type,
          );
          setProviderComponent(() => component.component);
        } catch (error) {
          console.error("Failed to load provider component:", error);
          setProviderComponent(null);
        } finally {
          setLoadingProviderForm(false);
        }
      }
    };

    loadProviderComponent();
  }, [formData.provider_type]);

  // Handle form submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);
    setFormSuccess(null);

    try {
      if (!formData.name.trim()) {
        setFormError("Provider name is required");
        return;
      }
      if (!formData.display_name.trim()) {
        setFormError("Display name is required");
        return;
      }
      if (isKubernetesProviderType(formData.provider_type)) {
        if (
          formData.bootstrap_mode === "image_factory_managed" &&
          formData.credential_scope === "read_only"
        ) {
          setFormError(
            "Image Factory Managed bootstrap requires credential scope of namespace_admin or cluster_admin",
          );
          return;
        }
        if (formData.bootstrap_mode === "image_factory_managed") {
          const bootstrapErr = validateAuthConfig(
            formData.config?.bootstrap_auth,
            "Bootstrap",
          );
          if (bootstrapErr) {
            setFormError(bootstrapErr);
            return;
          }
        } else {
          const runtimeErr = validateAuthConfig(
            formData.config?.runtime_auth,
            "Runtime",
          );
          if (runtimeErr) {
            setFormError(runtimeErr);
            return;
          }
        }
        const nsErr = validateTargetNamespace(formData.target_namespace);
        if (nsErr) {
          setFormError(nsErr);
          return;
        }
      }

      if (editingProvider) {
        const preparedConfig = isKubernetesProviderType(formData.provider_type)
          ? normalizeKubernetesProviderConfig(
              formData.config,
              formData.bootstrap_mode,
            )
          : formData.config;
        await infrastructureService.updateProvider(editingProvider.id, {
          display_name: formData.display_name,
          config: preparedConfig,
          capabilities: formData.capabilities,
          is_global: formData.is_global,
          bootstrap_mode: formData.bootstrap_mode,
          credential_scope: formData.credential_scope,
          target_namespace: formData.target_namespace.trim(),
        });
        setFormSuccess("Provider updated successfully!");
        setTimeout(() => setFormSuccess(null), 5000);
      } else {
        const preparedConfig = isKubernetesProviderType(formData.provider_type)
          ? normalizeKubernetesProviderConfig(
              formData.config,
              formData.bootstrap_mode,
            )
          : formData.config;
        await infrastructureService.createProvider({
          ...formData,
          target_namespace: formData.target_namespace.trim(),
          config: preparedConfig,
        });
        setFormSuccess("Provider created successfully!");
        setTimeout(() => setFormSuccess(null), 5000);
      }

      setShowForm(false);
      setEditingProvider(null);
      resetForm();
      await loadProviders();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "An error occurred");
    }
  };

  // Handle delete
  const handleDelete = async (provider: InfrastructureProvider) => {
    if (
      !window.confirm(
        `Delete provider "${provider.display_name}"? This action cannot be undone.`,
      )
    ) {
      return;
    }

    try {
      setError(null);
      await infrastructureService.deleteProvider(provider.id);
      await loadProviders();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to delete provider",
      );
    }
  };

  // Handle edit
  const handleEdit = async (provider: InfrastructureProvider) => {
    setFormSuccess(null);
    setError(null);
    try {
      const freshProvider = await infrastructureService.getProvider(provider.id);
      openEditProvider(freshProvider);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to load provider for edit form",
      );
    }
  };

  // Handle copy YAML
  // Handle test connection
  const handleTestConnection = async (providerId: string) => {
    setTestingConnection(providerId);
    try {
      const result =
        await infrastructureService.testProviderConnection(providerId);
      if (result.success) {
        alert(`✅ Connection test successful: ${result.message}`);
      } else {
        alert(`❌ Connection test failed: ${result.message}`);
      }
    } catch (err) {
      alert(
        `❌ Connection test failed: ${err instanceof Error ? err.message : "Unknown error"}`,
      );
    } finally {
      setTestingConnection(null);
    }
  };

  // Handle toggle status
  const handleToggleStatus = async (provider: InfrastructureProvider) => {
    try {
      const newStatus = provider.status === "online" ? "offline" : "online";
      await infrastructureService.updateProvider(provider.id, {
        status: newStatus,
      });
      await loadProviders();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to update provider status",
      );
    }
  };

  // Reset form
  const resetForm = () => {
    setFormData({
      provider_type: "kubernetes",
      name: "",
      display_name: "",
      config: {
        runtime_auth: { auth_method: "token" },
        bootstrap_auth: { auth_method: "token" },
        tekton_enabled: true,
        quarantine_dispatch_enabled: false,
      },
      capabilities: [],
      is_global: false,
      bootstrap_mode: "image_factory_managed",
      credential_scope: "unknown",
      target_namespace: systemNamespaceDefault,
    });
    setFormError(null);
    setProviderComponent(null);
    setLoadingProviderForm(false);
    // Don't clear formSuccess here - it should persist at the top level
  };

  // Handle cancel
  const handleCancel = () => {
    setShowForm(false);
    setEditingProvider(null);
    resetForm();
  };

  // Render provider config form based on type
  const renderProviderConfigForm = () => {
    if (loadingProviderForm) {
      return (
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <span className="ml-2 text-gray-600 dark:text-gray-400">
            Loading provider form...
          </span>
        </div>
      );
    }

    if (!providerComponent) {
      return (
        <div className="text-center py-8 text-red-600 dark:text-red-400">
          Failed to load provider form. Please try selecting a different
          provider type.
        </div>
      );
    }

    const ProviderFormComponent = providerComponent;
    return (
      <ProviderFormComponent formData={formData} setFormData={setFormData} />
    );
  };
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            Infrastructure Providers
          </h1>
          <p className="mt-2 text-gray-600 dark:text-gray-400">
            Configure and manage infrastructure providers for build execution
          </p>
        </div>
        {canManageAdmin && (
          <button
            onClick={() => {
              setFormSuccess(null);
              setShowForm(true);
            }}
            className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 flex items-center gap-2"
          >
            <Plus className="h-4 w-4" />
            Add Provider
          </button>
        )}
      </div>

      {!canManageAdmin && (
        <div className="rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
          Read-only mode: provider create, edit, delete, and enable/disable actions are hidden for System Administrator Viewer.
        </div>
      )}

      {/* Error Display */}
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md p-4">
          <div className="flex items-center">
            <AlertCircle className="h-5 w-5 text-red-600 mr-2" />
            <span className="text-red-800 dark:text-red-200">{error}</span>
          </div>
        </div>
      )}

      {/* Success Display */}
      {formSuccess && (
        <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-md p-4">
          <div className="flex items-center">
            <Check className="h-5 w-5 text-green-600 mr-2" />
            <span className="text-green-800 dark:text-green-200">
              {formSuccess}
            </span>
          </div>
        </div>
      )}

      {/* Providers List */}
      <div className="bg-white dark:bg-gray-800 shadow rounded-lg">
        <div className="px-4 py-5 sm:p-6">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">
              Configured Providers
            </h3>
            <button
              onClick={loadProviders}
              disabled={loading}
              className="text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200"
            >
              <RefreshCw
                className={`h-5 w-5 ${loading ? "animate-spin" : ""}`}
              />
            </button>
          </div>

          {prepareBatchMetrics && (
            <div className="mb-4 rounded-lg border border-indigo-200 bg-indigo-50 p-3 dark:border-indigo-700 dark:bg-indigo-900/20">
              <div className="flex items-center justify-between gap-2">
                <p className="text-sm font-semibold text-indigo-900 dark:text-indigo-100">
                  Prepare Summary Batch Diagnostics
                </p>
                <p className="text-xs text-indigo-700 dark:text-indigo-300">
                  Based on provider list enrichment requests
                </p>
              </div>
              <div className="mt-2 grid grid-cols-2 gap-2 text-xs md:grid-cols-5">
                <div className="rounded border border-indigo-200 bg-white px-2 py-1 dark:border-indigo-700 dark:bg-gray-900/60">
                  <div className="text-indigo-700 dark:text-indigo-300">
                    Batches
                  </div>
                  <div className="font-semibold text-indigo-900 dark:text-indigo-100">
                    {prepareBatchMetrics.batch_count}
                  </div>
                </div>
                <div className="rounded border border-indigo-200 bg-white px-2 py-1 dark:border-indigo-700 dark:bg-gray-900/60">
                  <div className="text-indigo-700 dark:text-indigo-300">
                    Providers
                  </div>
                  <div className="font-semibold text-indigo-900 dark:text-indigo-100">
                    {prepareBatchMetrics.providers_total}
                  </div>
                </div>
                <div className="rounded border border-indigo-200 bg-white px-2 py-1 dark:border-indigo-700 dark:bg-gray-900/60">
                  <div className="text-indigo-700 dark:text-indigo-300">
                    Avg / Max ms
                  </div>
                  <div className="font-semibold text-indigo-900 dark:text-indigo-100">
                    {Math.round(prepareBatchMetrics.batch_avg_ms)} /{" "}
                    {prepareBatchMetrics.batch_max_ms}
                  </div>
                </div>
                <div className="rounded border border-indigo-200 bg-white px-2 py-1 dark:border-indigo-700 dark:bg-gray-900/60">
                  <div className="text-indigo-700 dark:text-indigo-300">
                    Repo / Fallback
                  </div>
                  <div className="font-semibold text-indigo-900 dark:text-indigo-100">
                    {prepareBatchMetrics.repository_batches} /{" "}
                    {prepareBatchMetrics.fallback_batches}
                  </div>
                </div>
                <div className="rounded border border-indigo-200 bg-white px-2 py-1 dark:border-indigo-700 dark:bg-gray-900/60">
                  <div className="text-indigo-700 dark:text-indigo-300">
                    Errors
                  </div>
                  <div className="font-semibold text-indigo-900 dark:text-indigo-100">
                    {prepareBatchMetrics.batch_errors}
                  </div>
                </div>
              </div>
            </div>
          )}

          {loading ? (
            <div className="text-center py-8">
              <RefreshCw className="h-8 w-8 animate-spin mx-auto text-gray-400" />
              <p className="mt-2 text-gray-500">Loading providers...</p>
            </div>
          ) : providers.length === 0 ? (
            <div className="text-center py-8">
              <Server className="h-12 w-12 mx-auto text-gray-400" />
              <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">
                No providers configured
              </h3>
              <p className="mt-1 text-sm text-gray-500">
                Get started by adding your first infrastructure provider.
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              {providers.map((provider) => (
                <div
                  key={provider.id}
                  className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                  onClick={() =>
                    navigate(`/admin/infrastructure/${provider.id}`)
                  }
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-3">
                      {providerTypeIcons[provider.provider_type]}
                      <div>
                        <h4 className="text-sm font-medium text-gray-900 dark:text-white">
                          {provider.display_name}
                        </h4>
                        <p className="text-sm text-gray-500 dark:text-gray-400">
                          {provider.name} • {provider.provider_type}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColors[provider.status]}`}
                      >
                        {provider.status}
                      </span>
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                          provider.is_schedulable &&
                          provider.status === "online"
                            ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                            : "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200"
                        }`}
                      >
                        {provider.is_schedulable && provider.status === "online"
                          ? "schedulable"
                          : "blocked"}
                      </span>
                      {provider.latest_prepare_status && (
                        <span
                          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                            provider.latest_prepare_status === "succeeded"
                              ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                              : provider.latest_prepare_status === "failed"
                                ? "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                                : "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
                          } ${provider.latest_prepare_check_severity ? prepareSeverityClasses[provider.latest_prepare_check_severity] || "" : ""}`}
                          title={(() => {
                            if (
                              provider.latest_prepare_status === "failed" &&
                              provider.latest_prepare_error
                            ) {
                              const checkSuffix =
                                provider.latest_prepare_check_category ||
                                provider.latest_prepare_check_severity
                                  ? ` (${provider.latest_prepare_check_category || "check"}${provider.latest_prepare_check_severity ? `/${provider.latest_prepare_check_severity}` : ""})`
                                  : "";
                              const hintSuffix =
                                provider.latest_prepare_remediation_hint
                                  ? ` - Hint: ${provider.latest_prepare_remediation_hint}`
                                  : "";
                              return `${provider.latest_prepare_error}${checkSuffix}${hintSuffix}`;
                            }
                            if (
                              provider.latest_prepare_check_category ||
                              provider.latest_prepare_check_severity
                            ) {
                              return `Latest check: ${provider.latest_prepare_check_category || "unknown"}${provider.latest_prepare_check_severity ? ` (${provider.latest_prepare_check_severity})` : ""}${provider.latest_prepare_remediation_hint ? ` - Hint: ${provider.latest_prepare_remediation_hint}` : ""}`;
                            }
                            return undefined;
                          })()}
                        >
                          prepare:{provider.latest_prepare_status}
                        </span>
                      )}
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleTestConnection(provider.id);
                        }}
                        disabled={testingConnection === provider.id}
                        className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-200"
                        title="Test Connection"
                      >
                        {testingConnection === provider.id ? (
                          <RefreshCw className="h-4 w-4 animate-spin" />
                        ) : (
                          <TestTube className="h-4 w-4" />
                        )}
                      </button>
                      {canManageAdmin && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleToggleStatus(provider);
                          }}
                          className={`${
                            provider.status === "online"
                              ? "text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-200"
                              : "text-green-600 hover:text-green-800 dark:text-green-400 dark:hover:text-green-200"
                          }`}
                          title={
                            provider.status === "online"
                              ? "Disable Provider"
                              : "Enable Provider"
                          }
                        >
                          {provider.status === "online" ? (
                            <X className="h-4 w-4" />
                          ) : (
                            <Check className="h-4 w-4" />
                          )}
                        </button>
                      )}
                      {canManageAdmin && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleEdit(provider);
                          }}
                          className="text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200"
                          title="Edit Provider"
                        >
                          <Edit2 className="h-4 w-4" />
                        </button>
                      )}
                      {canManageAdmin && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDelete(provider);
                          }}
                          className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-200"
                          title="Delete Provider"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      )}
                    </div>
                  </div>

                  {provider.capabilities &&
                    provider.capabilities.length > 0 && (
                      <div className="mt-2">
                        <div className="flex flex-wrap gap-1">
                          {provider.capabilities.map((capability, index) => (
                            <span
                              key={index}
                              className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200"
                            >
                              {capability}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}

                  <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
                    Created {new Date(provider.created_at).toLocaleDateString()}
                    {provider.last_health_check && (
                      <span className="ml-4">
                        Last health check:{" "}
                        {new Date(provider.last_health_check).toLocaleString()}
                      </span>
                    )}
                  </div>
                  {provider.status === "online" && !provider.is_schedulable && (
                    <div className="mt-2 rounded-md border border-amber-300 bg-amber-50 dark:border-amber-700 dark:bg-amber-900/20 px-3 py-2 text-xs text-amber-900 dark:text-amber-200">
                      <div>
                        <span className="font-semibold">Blocked:</span>{" "}
                        {provider.schedulable_reason ||
                          "Provider is not schedulable due to readiness or policy gates."}
                      </div>
                      {provider.blocked_by && provider.blocked_by.length > 0 && (
                        <div className="mt-2 flex flex-wrap gap-1.5">
                          {provider.blocked_by.map((gate) => (
                            <span
                              key={`${provider.id}-${gate}`}
                              className="inline-flex items-center rounded-full border border-amber-400/60 bg-amber-100/80 px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide text-amber-900 dark:border-amber-600 dark:bg-amber-900/40 dark:text-amber-100"
                            >
                              {blockedByLabel(gate)}
                            </span>
                          ))}
                        </div>
                      )}
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          navigate(
                            `/admin/infrastructure/${provider.id}#prepare-checks-section`,
                          );
                        }}
                        className="mt-2 text-xs font-semibold text-amber-800 hover:text-amber-900 dark:text-amber-300 dark:hover:text-amber-200 underline underline-offset-2"
                      >
                        View prepare checks
                      </button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Add/Edit Provider Modal */}
      {showForm && (
        <div
          className="fixed inset-0 z-50 h-full w-full overflow-y-auto bg-gray-600/50"
          onMouseDown={(e) => {
            // Keep dialog open on backdrop interaction; close only via explicit actions.
            e.stopPropagation();
          }}
          onClick={(e) => {
            e.stopPropagation();
          }}
        >
          <div
            className="relative top-20 mx-auto w-11/12 max-w-2xl rounded-md border bg-white p-5 shadow-lg dark:bg-gray-800"
            onMouseDown={(e) => e.stopPropagation()}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mt-3">
              <div className="mb-4 flex items-center justify-between">
                <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                  {editingProvider
                    ? "Edit Provider"
                    : "Add Infrastructure Provider"}
                </h3>
                <button
                  type="button"
                  onClick={handleCancel}
                  className="rounded-md p-1 text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-gray-200"
                  aria-label="Close provider form"
                >
                  <X className="h-5 w-5" />
                </button>
              </div>

              <form onSubmit={handleSubmit} className="space-y-4" noValidate>
                {/* Provider Type */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Provider Type
                  </label>
                  <select
                    value={formData.provider_type}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        provider_type: e.target
                          .value as InfrastructureProviderType,
                        config: isKubernetesProviderType(
                          e.target.value as InfrastructureProviderType,
                        )
                          ? {
                              runtime_auth: { auth_method: "token" },
                              bootstrap_auth: { auth_method: "token" },
                              tekton_enabled: true,
                            }
                          : {}, // Reset config when type changes
                      })
                    }
                    disabled={!!editingProvider}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-100 dark:disabled:bg-gray-600 disabled:cursor-not-allowed"
                  >
                    <option value="kubernetes">Kubernetes Cluster</option>
                    <option value="aws-eks">Amazon EKS</option>
                    <option value="gcp-gke">Google GKE</option>
                    <option value="azure-aks">Azure AKS</option>
                    <option value="oci-oke">Oracle OKE</option>
                    <option value="vmware-vks">VMware vKS</option>
                    <option value="openshift">Red Hat OpenShift</option>
                    <option value="rancher">Rancher</option>
                    <option value="build_nodes">Build Nodes</option>
                  </select>
                </div>

                {/* Name */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Internal Name
                  </label>
                  <input
                    type="text"
                    value={formData.name}
                    onChange={(e) =>
                      setFormData({ ...formData, name: e.target.value })
                    }
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                    placeholder="prod-k8s-cluster"
                    disabled={!!editingProvider}
                    required
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Internal identifier, cannot be changed after creation
                  </p>
                </div>

                {/* Display Name */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Display Name
                  </label>
                  <input
                    type="text"
                    value={formData.display_name}
                    onChange={(e) =>
                      setFormData({ ...formData, display_name: e.target.value })
                    }
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                    placeholder="Production Kubernetes Cluster"
                    required
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    User-friendly name shown in selection interfaces
                  </p>
                </div>

                {isKubernetesProviderType(formData.provider_type) && (
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Tekton Integration
                    </label>
                    <div className="flex items-start gap-3 rounded-md border border-gray-200 dark:border-gray-700 p-3 bg-gray-50 dark:bg-gray-900/40">
                      <input
                        id="provider_tekton_enabled"
                        type="checkbox"
                        checked={formData.config?.tekton_enabled === true}
                        onChange={(e) =>
                          setFormData({
                            ...formData,
                            config: {
                              ...formData.config,
                              tekton_enabled: e.target.checked,
                            },
                          })
                        }
                        className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                      />
                      <div>
                        <label
                          htmlFor="provider_tekton_enabled"
                          className="text-sm font-medium text-gray-700 dark:text-gray-200"
                        >
                          Enable Tekton for this Kubernetes provider
                        </label>
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                          Required for Kubernetes build execution and
                          PipelineRun creation.
                        </p>
                      </div>
                    </div>
                    <div className="mt-3 flex items-start gap-3 rounded-md border border-gray-200 dark:border-gray-700 p-3 bg-gray-50 dark:bg-gray-900/40">
                      <input
                        id="provider_quarantine_dispatch_enabled"
                        type="checkbox"
                        checked={
                          formData.config?.quarantine_dispatch_enabled === true
                        }
                        onChange={(e) =>
                          setFormData({
                            ...formData,
                            config: {
                              ...formData.config,
                              quarantine_dispatch_enabled: e.target.checked,
                            },
                          })
                        }
                        className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                      />
                      <div>
                        <label
                          htmlFor="provider_quarantine_dispatch_enabled"
                          className="text-sm font-medium text-gray-700 dark:text-gray-200"
                        >
                          Allow quarantine pipeline dispatch
                        </label>
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                          Enables this provider to execute quarantine intake
                          pipeline jobs.
                        </p>
                      </div>
                    </div>
                  </div>
                )}

                {/* Global Availability */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Availability
                  </label>
                  <div className="flex items-start gap-3 rounded-md border border-gray-200 dark:border-gray-700 p-3 bg-gray-50 dark:bg-gray-900/40">
                    <input
                      id="provider_is_global"
                      type="checkbox"
                      checked={formData.is_global}
                      onChange={(e) =>
                        setFormData({
                          ...formData,
                          is_global: e.target.checked,
                        })
                      }
                      className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                    />
                    <div>
                      <label
                        htmlFor="provider_is_global"
                        className="text-sm font-medium text-gray-700 dark:text-gray-200"
                      >
                        Make this provider available to all tenants
                      </label>
                      <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                        If unchecked, you can grant access to specific tenants
                        from the provider detail page.
                      </p>
                    </div>
                  </div>
                </div>

                {isKubernetesProviderType(formData.provider_type) && (
                  <>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <div className="mb-1 flex items-center gap-2">
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                            Bootstrap Mode
                          </label>
                          <button
                            type="button"
                            onClick={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                              setShowBootstrapRBACDrawer(true);
                            }}
                            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                            title="Recommended service account and RBAC manifests"
                          >
                            <HelpCircle className="h-4 w-4" />
                          </button>
                        </div>
                        <select
                          value={formData.bootstrap_mode}
                          onChange={(e) =>
                            setFormData({
                              ...formData,
                              bootstrap_mode: e.target
                                .value as ProviderFormData["bootstrap_mode"],
                            })
                          }
                          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-blue-500 focus:border-blue-500"
                        >
                          <option value="image_factory_managed">
                            Image Factory Managed
                          </option>
                          <option value="self_managed">Self Managed</option>
                        </select>
                        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                          Managed mode: provide bootstrap auth only; runtime auth
                          is generated during Prepare Provider. Self-managed:
                          provide runtime auth and manage cluster/tenant RBAC
                          yourself.
                        </p>
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                          Credential Scope
                        </label>
                        <select
                          value={formData.credential_scope}
                          onChange={(e) =>
                            setFormData({
                              ...formData,
                              credential_scope: e.target
                                .value as ProviderFormData["credential_scope"],
                            })
                          }
                          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-blue-500 focus:border-blue-500"
                        >
                          <option value="unknown">Unknown</option>
                          <option value="cluster_admin">Cluster Admin</option>
                          <option value="namespace_admin">
                            Namespace Admin
                          </option>
                          <option value="read_only">Read Only</option>
                        </select>
                      </div>
                    </div>
                    <div className="rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/40 p-3 text-xs text-gray-700 dark:text-gray-300">
                      <p className="font-semibold mb-1">Mode behavior</p>
                      <p>
                        <span className="font-medium">
                          Image Factory Managed:
                        </span>{" "}
                        Image Factory runs provider/tenant prepare workflows,
                        creates runtime identity, and applies tenant namespace
                        runtime RBAC automatically.
                      </p>
                      <p className="mt-1">
                        <span className="font-medium">Self Managed:</span> you
                        keep control of runtime credentials and RBAC lifecycle.
                        Image Factory uses your provided runtime auth and does
                        not bootstrap cluster/tenant RBAC for you.
                      </p>
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        Target Namespace
                      </label>
                      <input
                        type="text"
                        value={formData.target_namespace}
                        onChange={(e) =>
                          setFormData({
                            ...formData,
                            target_namespace: e.target.value,
                          })
                        }
                        className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                        placeholder={systemNamespaceDefault}
                      />
                      {effectiveTargetNamespace !== systemNamespaceDefault && (
                        <div className="mt-2 rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-xs text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                          You changed the namespace from recommended default.
                          Ensure RBAC and Tekton assets are applied in{" "}
                          <span className="font-semibold">
                            {effectiveTargetNamespace}
                          </span>
                          .
                          <button
                            type="button"
                            onClick={() =>
                              setFormData({
                                ...formData,
                                target_namespace: systemNamespaceDefault,
                              })
                            }
                            className="ml-2 underline underline-offset-2 text-amber-900 hover:text-amber-950 dark:text-amber-200 dark:hover:text-amber-100"
                          >
                            Reset to {systemNamespaceDefault}
                          </button>
                        </div>
                      )}
                    </div>
                  </>
                )}

                {/* Provider-specific configuration */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Configuration
                  </label>
                  {renderProviderConfigForm()}
                </div>

                {/* Capabilities */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Capabilities (optional)
                  </label>
                  <input
                    type="text"
                    value={formData.capabilities.join(", ")}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        capabilities: e.target.value
                          .split(",")
                          .map((s) => s.trim())
                          .filter((s) => s),
                      })
                    }
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                    placeholder="gpu, arm64, high-memory"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Comma-separated list of capabilities (e.g., gpu, arm64,
                    high-memory)
                  </p>
                </div>

                {/* Error Display */}
                {formError && (
                  <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md p-3">
                    <div className="flex items-center">
                      <AlertCircle className="h-4 w-4 text-red-600 mr-2" />
                      <span className="text-red-800 dark:text-red-200 text-sm">
                        {formError}
                      </span>
                    </div>
                  </div>
                )}

                {/* Form Actions */}
                <div className="flex justify-end space-x-3 pt-4">
                  <button
                    type="button"
                    onClick={handleCancel}
                    className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 border border-gray-300 rounded-md hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-600"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700"
                  >
                    {editingProvider ? "Update Provider" : "Create Provider"}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}

      <TooltipDrawer
        isOpen={showBootstrapRBACDrawer}
        onClose={() => setShowBootstrapRBACDrawer(false)}
        title={
          formData.bootstrap_mode === "image_factory_managed"
            ? "Image Factory Managed Bootstrap RBAC"
            : "Self-Managed Runtime RBAC"
        }
      >
        <div className="space-y-4 text-sm text-gray-700 dark:text-gray-300">
          {formData.bootstrap_mode === "image_factory_managed" ? (
            <p>
              Recommended approach is split identity: an elevated bootstrap
              service account for prepare/install, and a least-privilege runtime
              service account for normal build execution.
            </p>
          ) : (
            <p>
              Self-managed mode assumes cluster bootstrap is handled outside of
              Image Factory. You still need a runtime service account and
              namespace-scoped RBAC so builds can run.
            </p>
          )}
          <div className="rounded-md border border-amber-300 bg-amber-50 p-3 text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
            Update namespace and object names before applying. These are secure
            defaults to get started, not mandatory final production policy.
          </div>
          <CopyableCodeBlock
            title="1) System Namespace + Service Accounts"
            code={bootstrapNamespaceTemplate(effectiveTargetNamespace)}
            language="yaml"
          />
          {formData.bootstrap_mode === "image_factory_managed" && (
            <CopyableCodeBlock
              title="2) Bootstrap ClusterRole + Binding (Managed Prepare)"
              code={bootstrapClusterRoleTemplate(effectiveTargetNamespace)}
              language="yaml"
            />
          )}
          <CopyableCodeBlock
            title="3) Tenant Namespace Runtime Role + Binding (Applied Per Tenant)"
            code={runtimeRoleTemplate(effectiveTargetNamespace)}
            language="yaml"
          />
          <CopyableCodeBlock
            title="4) ServiceAccount Tokens (Recommended)"
            code={serviceAccountTokenTemplate(effectiveTargetNamespace)}
            language="bash"
          />
        </div>
      </TooltipDrawer>
    </div>
  );
};

export default AdminInfrastructureProvidersPage;
