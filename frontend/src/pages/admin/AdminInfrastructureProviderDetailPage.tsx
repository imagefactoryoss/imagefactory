import Drawer from "@/components/ui/Drawer.tsx";
import HelpTooltip from "@/components/common/HelpTooltip";
import {
  CopyableCodeBlock,
  TooltipDrawer,
} from "@/components/admin/providers/TooltipDrawer";
import { TenantAssetDriftBadge } from "@/components/admin/providers/TenantAssetDriftBadge";
import { NIL_TENANT_ID } from "@/constants/tenant";
import { useCanManageAdmin } from "@/hooks/useAccess";
import { adminService } from "@/services/adminService";
import { infrastructureService } from "@/services/infrastructureService";
import { userService } from "@/services/userService";
import { useAuthStore } from "@/store/auth";
import { useTenantStore } from "@/store/tenant";
import {
  InfrastructureProvider,
  ProviderPrepareRun,
  ProviderPrepareStatus,
  ProviderTenantNamespacePrepare,
  TektonInstallMode,
  TektonProviderStatus,
} from "@/types";
import {
  AlertCircle,
  ArrowLeft,
  Check,
  Copy,
  Edit2,
  Eye,
  EyeOff,
  HelpCircle,
  RefreshCw,
  Server,
  TestTube,
  Trash2,
  X,
  Zap,
} from "lucide-react";
import React, { useCallback, useEffect, useState } from "react";
import toast from "react-hot-toast";
import { useLocation, useNavigate, useParams } from "react-router-dom";

const statusColors: Record<string, string> = {
  online: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
  offline: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
  maintenance:
    "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
  pending: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
};

const providerTypeIcons: Record<string, React.ReactNode> = {
  kubernetes: <Zap className="h-5 w-5" />,
  "aws-eks": <Server className="h-5 w-5" />, // AWS icon placeholder
  "gcp-gke": <Server className="h-5 w-5" />, // GCP icon placeholder
  "azure-aks": <Server className="h-5 w-5" />, // Azure icon placeholder
  "oci-oke": <Server className="h-5 w-5" />, // Oracle icon placeholder
  "vmware-vks": <Server className="h-5 w-5" />, // VMware icon placeholder
  openshift: <Server className="h-5 w-5" />, // OpenShift icon placeholder
  rancher: <Server className="h-5 w-5" />, // Rancher icon placeholder
  build_nodes: <Server className="h-5 w-5" />,
};

const kubernetesProviderTypes = new Set([
  "kubernetes",
  "aws-eks",
  "gcp-gke",
  "azure-aks",
  "oci-oke",
  "vmware-vks",
  "openshift",
  "rancher",
]);

const tektonJobStatusColors: Record<string, string> = {
  pending: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
  running:
    "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
  succeeded:
    "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
  failed: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
  cancelled: "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200",
};

const isPrepareRunActiveStatus = (status?: string | null): boolean =>
  status === "pending" || status === "running";

const formatAssetVersion = (version?: string | null): string => {
  const value = (version || "").trim();
  if (!value) return "unknown";
  if (value.startsWith("sha256:")) {
    const digest = value.slice("sha256:".length);
    if (digest.length > 8) {
      return `sha256:${digest.slice(0, 4)}...${digest.slice(-4)}`;
    }
  }
  if (value.length > 20) {
    return `${value.slice(0, 10)}...${value.slice(-6)}`;
  }
  return value;
};

const bootstrapNamespaceTemplate = `apiVersion: v1
kind: Namespace
metadata:
  name: imagefactory-system
  labels:
    app: image-factory
    managedBy: image-factory
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: image-factory-bootstrap-sa
  namespace: imagefactory-system`;

const runtimeNamespaceTemplate = `apiVersion: v1
kind: Namespace
metadata:
  name: imagefactory-system
  labels:
    app: image-factory
    managedBy: image-factory
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: image-factory-runtime-sa
  namespace: imagefactory-system`;

const bootstrapClusterRoleTemplate = `apiVersion: rbac.authorization.k8s.io/v1
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
  namespace: imagefactory-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: image-factory-bootstrap-role`;

const runtimeRoleTemplate = `apiVersion: rbac.authorization.k8s.io/v1
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
  namespace: imagefactory-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: image-factory-runtime-role`;

const serviceAccountTokenTemplate = `# Kubernetes v1.24+ typically does not auto-create long-lived service account token Secrets.
# Use TokenRequest via kubectl (recommended):
#
# 1 year = 8760h (cluster policies may cap max duration)
kubectl -n imagefactory-system create token image-factory-bootstrap-sa --duration=8760h`;

const runtimeServiceAccountTokenTemplate = `# Kubernetes v1.24+ typically does not auto-create long-lived service account token Secrets.
# Use TokenRequest via kubectl (recommended):
#
# 1 year = 8760h (cluster policies may cap max duration)
kubectl -n imagefactory-system create token image-factory-runtime-sa --duration=8760h`;

const buildTenantNamespaceName = (tenantId: string) => {
  const normalized = (tenantId || "").trim();
  if (!normalized) return "";
  return `image-factory-${normalized.slice(0, 8)}`;
};

const tenantNamespaceTenantIDLabelKey = "imagefactory.io/tenant-id";

const isLikelyConnectivityIssue = (value: string): boolean => {
  const normalized = value.toLowerCase();
  return (
    normalized.includes("kubeconfig") ||
    normalized.includes("failed to create kubernetes client") ||
    normalized.includes("kubernetes api unreachable") ||
    normalized.includes("namespace probe failed") ||
    normalized.includes("dial tcp") ||
    normalized.includes("i/o timeout") ||
    normalized.includes("connection refused") ||
    normalized.includes("no such host") ||
    normalized.includes("tls handshake timeout") ||
    normalized.includes("context deadline exceeded") ||
    normalized.includes("tekton api preflight failed")
  );
};

const guidanceForReadinessItem = (item: string): string => {
  const normalized = item.toLowerCase();
  if (normalized.includes("missing tekton task")) {
    return "Apply latest Tekton tasks in target namespace (kubectl apply -k backend/tekton).";
  }
  if (normalized.includes("missing tekton pipeline")) {
    return "Apply latest Tekton pipelines in target namespace (kubectl apply -k backend/tekton).";
  }
  if (normalized.includes("missing namespace")) {
    return "Prepare provider/tenant namespace and ensure target namespace exists.";
  }
  if (normalized.includes("access denied") || normalized.includes("forbidden")) {
    return "Expand runtime service account RBAC for namespaces, Tekton resources, pods, and secrets.";
  }
  if (normalized.includes("cluster_capacity")) {
    return "Cluster is reachable but currently unschedulable due to capacity.";
  }
  return "Run Prepare Provider and fix the failing prerequisite shown in the readiness checklist.";
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

/**
 * AdminInfrastructureProviderDetailPage - Admin component for viewing infrastructure provider details
 * Shows detailed information about a specific infrastructure provider
 */
const AdminInfrastructureProviderDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const location = useLocation();
  const navigate = useNavigate();
  const { groups = [], token } = useAuthStore();
  const selectedTenantId = useTenantStore((state) => state.selectedTenantId);
  const canManageAdmin = useCanManageAdmin();

  // Check if user is system admin
  const isSystemAdmin = groups.some(
    (g: any) => g.role_type === "system_administrator",
  );

  const [provider, setProvider] = useState<InfrastructureProvider | null>(null);
  const [loading, setLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [pageError, setPageError] = useState<string | null>(null);
  const [testingConnection, setTestingConnection] = useState(false);
  const [showTestDrawer, setShowTestDrawer] = useState(false);
  const [testProgress, setTestProgress] = useState<
    "initializing" | "connecting" | "receiving" | "completed" | "failed" | null
  >(null);
  const [testError, setTestError] = useState<string | null>(null);
  const [creatorName, setCreatorName] = useState<string | null>(null);
  const [visibleSensitiveFields, setVisibleSensitiveFields] = useState<
    Record<string, boolean>
  >({});
  const [permissionsLoading, setPermissionsLoading] = useState(false);
  const [tenantPermissions, setTenantPermissions] = useState<
    Record<string, boolean>
  >({});
  const [tenants, setTenants] = useState<Array<{ id: string; name: string }>>(
    [],
  );
  const [savingPermissions, setSavingPermissions] = useState(false);
  const [showAccessDrawer, setShowAccessDrawer] = useState(false);
  const [draftIsGlobal, setDraftIsGlobal] = useState<boolean>(false);
  const [draftTenantPermissions, setDraftTenantPermissions] = useState<
    Record<string, boolean>
  >({});
  const [tektonStatus, setTektonStatus] = useState<TektonProviderStatus | null>(
    null,
  );
  const [tektonLoading, setTektonLoading] = useState(false);
  const [tektonError, setTektonError] = useState<string | null>(null);
  const [tektonActionLoading, setTektonActionLoading] = useState<
    "install" | "upgrade" | "validate" | "retry" | null
  >(null);
  const [tektonInstallMode, setTektonInstallMode] = useState<TektonInstallMode>(
    "image_factory_installer",
  );
  const [tektonAssetVersion, setTektonAssetVersion] = useState("v1");
  const [prepareStatus, setPrepareStatus] =
    useState<ProviderPrepareStatus | null>(null);
  const [prepareRuns, setPrepareRuns] = useState<ProviderPrepareRun[]>([]);
  const [prepareLoading, setPrepareLoading] = useState(false);
  const [prepareError, setPrepareError] = useState<string | null>(null);
  const [prepareActionLoading, setPrepareActionLoading] = useState(false);
  const [pendingPrepareCompletionRefresh, setPendingPrepareCompletionRefresh] =
    useState(false);
  const [showPrepareDrawer, setShowPrepareDrawer] = useState(false);
  const [prepareWsConnected, setPrepareWsConnected] = useState(false);
  const [prepareWsRetryNonce, setPrepareWsRetryNonce] = useState(0);
  const [selectedPrepareRunStatus, setSelectedPrepareRunStatus] =
    useState<ProviderPrepareStatus | null>(null);
  const [selectedPrepareRunLoading, setSelectedPrepareRunLoading] =
    useState(false);
  const [selectedPrepareRunError, setSelectedPrepareRunError] = useState<
    string | null
  >(null);
  const [copiedText, setCopiedText] = useState<string | null>(null);
  const [showBootstrapRBACDrawer, setShowBootstrapRBACDrawer] = useState(false);
  const [tenantNamespacePrepare, setTenantNamespacePrepare] =
    useState<ProviderTenantNamespacePrepare | null>(null);
  const [namespaceTenantId, setNamespaceTenantId] = useState("");
  const [tenantNamespacePrepareLoading, setTenantNamespacePrepareLoading] =
    useState(false);
  const [tenantNamespacePrepareError, setTenantNamespacePrepareError] =
    useState<string | null>(null);
  const [
    tenantNamespacePrepareActionLoading,
    setTenantNamespacePrepareActionLoading,
  ] = useState(false);
  const [tenantNamespaceReconcileLoading, setTenantNamespaceReconcileLoading] =
    useState<"stale" | "selected" | null>(null);
  const [
    showTenantNamespacePrepareDrawer,
    setShowTenantNamespacePrepareDrawer,
  ] = useState(false);
  const [
    tenantNamespacePrepareWsConnected,
    setTenantNamespacePrepareWsConnected,
  ] = useState(false);
  const effectiveTenantNamespaceId =
    isSystemAdmin && namespaceTenantId
      ? namespaceTenantId
      : selectedTenantId || "";
  const selectedNamespaceTenant = tenants.find(
    (t) => t.id === effectiveTenantNamespaceId,
  );
  const providerProfileVersion =
    typeof provider?.config?.tekton_profile_version === "string" &&
    provider.config.tekton_profile_version.trim() !== ""
      ? provider.config.tekton_profile_version.trim().toLowerCase()
      : "v1";

  const loadTektonStatus = useCallback(
    async (options?: { showLoading?: boolean }) => {
      const showLoading = options?.showLoading ?? true;
      if (!id) return;
      try {
        if (showLoading) {
          setTektonLoading(true);
        }
        setTektonError(null);
        const status = await infrastructureService.getTektonStatus(id, 25);
        setTektonStatus(status);
      } catch (err) {
        setTektonError(
          err instanceof Error ? err.message : "Failed to load Tekton status",
        );
      } finally {
        if (showLoading) {
          setTektonLoading(false);
        }
      }
    },
    [id],
  );

  const loadPrepareStatus = useCallback(
    async (options?: { showLoading?: boolean }) => {
      const showLoading = options?.showLoading ?? true;
      if (!id) return;
      try {
        if (showLoading) {
          setPrepareLoading(true);
        }
        setPrepareError(null);
        const status = await infrastructureService.getProviderPrepareStatus(id);
        setPrepareStatus(status);
        const runs = await infrastructureService.listProviderPrepareRuns(id, {
          limit: 10,
          offset: 0,
        });
        setPrepareRuns(runs);
      } catch (err) {
        setPrepareError(
          err instanceof Error
            ? err.message
            : "Failed to load provider preparation status",
        );
      } finally {
        if (showLoading) {
          setPrepareLoading(false);
        }
      }
    },
    [id],
  );

  const loadPrepareSnapshot = useCallback(async () => {
    if (!id) return;
    try {
      setPrepareError(null);
      const [status, runs] = await Promise.all([
        infrastructureService.getProviderPrepareStatus(id),
        infrastructureService.listProviderPrepareRuns(id, {
          limit: 1,
          offset: 0,
        }),
      ]);
      setPrepareStatus(status);
      if (runs.length > 0) {
        setPrepareRuns(runs);
      } else if (status?.active_run) {
        const activeRun = status.active_run;
        setPrepareRuns([activeRun]);
      } else {
        setPrepareRuns([]);
      }
    } catch (err) {
      setPrepareError(
        err instanceof Error
          ? err.message
          : "Failed to load provider preparation status",
      );
    }
  }, [id]);

  const loadPrepareRunDetails = useCallback(
    async (runId: string) => {
      if (!id || !runId) return;
      try {
        setSelectedPrepareRunLoading(true);
        setSelectedPrepareRunError(null);
        const status = await infrastructureService.getProviderPrepareRun(
          id,
          runId,
          {
            limit: 500,
            offset: 0,
          },
        );
        setSelectedPrepareRunStatus(status);
      } catch (err) {
        setSelectedPrepareRunError(
          err instanceof Error
            ? err.message
            : "Failed to load provider prepare run details",
        );
      } finally {
        setSelectedPrepareRunLoading(false);
      }
    },
    [id],
  );

  const loadTenantNamespacePrepareStatus = useCallback(async () => {
    if (!id) return;
    if (
      !effectiveTenantNamespaceId ||
      effectiveTenantNamespaceId === NIL_TENANT_ID
    ) {
      setTenantNamespacePrepare(null);
      return;
    }
    if (!provider) return;
    if (!isSystemAdmin) return;
    if (!kubernetesProviderTypes.has(provider.provider_type)) return;
    if (
      (provider.bootstrap_mode || "image_factory_managed") !==
      "image_factory_managed"
    ) {
      setTenantNamespacePrepare(null);
      return;
    }

    try {
      setTenantNamespacePrepareLoading(true);
      setTenantNamespacePrepareError(null);
      const status =
        await infrastructureService.getTenantNamespaceProvisionStatus(
          id,
          effectiveTenantNamespaceId,
        );
      setTenantNamespacePrepare(status);
    } catch (err) {
      setTenantNamespacePrepareError(
        err instanceof Error
          ? err.message
          : "Failed to load tenant namespace provisioning status",
      );
    } finally {
      setTenantNamespacePrepareLoading(false);
    }
  }, [id, effectiveTenantNamespaceId, provider, isSystemAdmin]);

  const refreshProviderReadinessAfterPrepare = useCallback(async () => {
    if (!id) return;
    try {
      const response = await infrastructureService.getProvider(id);
      setProvider(response);
      await loadTektonStatus();
      if (isSystemAdmin) {
        await loadTenantNamespacePrepareStatus();
      }
    } catch {
      // Best-effort post-prepare refresh; normal polling and manual refresh remain available.
    }
  }, [id, loadTektonStatus, loadTenantNamespacePrepareStatus, isSystemAdmin]);

  const handleProvisionTenantNamespace = useCallback(async () => {
    if (!id) return;
    if (
      !effectiveTenantNamespaceId ||
      effectiveTenantNamespaceId === NIL_TENANT_ID
    ) {
      toast.error("Select a tenant to provision its namespace.");
      return;
    }
    if (!provider) return;
    if (!isSystemAdmin) {
      toast.error("Permission denied.");
      return;
    }
    if (!kubernetesProviderTypes.has(provider.provider_type)) return;
    if (
      (provider.bootstrap_mode || "image_factory_managed") !==
      "image_factory_managed"
    ) {
      toast.error(
        "Tenant namespace provisioning is only available in managed mode.",
      );
      return;
    }

    try {
      setTenantNamespacePrepareActionLoading(true);
      setTenantNamespacePrepareError(null);
      setShowTenantNamespacePrepareDrawer(true);
      const prepare = await infrastructureService.provisionTenantNamespace(
        id,
        effectiveTenantNamespaceId,
      );
      setTenantNamespacePrepare(prepare);
      if (prepare.status === "succeeded") {
        toast.success("Tenant namespace provisioned.");
      } else if (prepare.status === "failed") {
        toast.error("Tenant namespace provisioning failed.");
      } else {
        toast.success("Tenant namespace provisioning started.");
      }
    } catch (err) {
      const message =
        err instanceof Error
          ? err.message
          : "Failed to provision tenant namespace";
      setTenantNamespacePrepareError(message);
      toast.error(message);
    } finally {
      setTenantNamespacePrepareActionLoading(false);
      await loadTenantNamespacePrepareStatus();
    }
  }, [
    id,
    effectiveTenantNamespaceId,
    provider,
    isSystemAdmin,
    loadTenantNamespacePrepareStatus,
  ]);

  const handleDeprovisionTenantNamespace = useCallback(async () => {
    if (!id) return;
    if (
      !effectiveTenantNamespaceId ||
      effectiveTenantNamespaceId === NIL_TENANT_ID
    ) {
      toast.error("Select a tenant to deprovision its namespace.");
      return;
    }
    if (!provider) return;
    if (!isSystemAdmin) {
      toast.error("Permission denied.");
      return;
    }
    if (!kubernetesProviderTypes.has(provider.provider_type)) return;
    if (
      (provider.bootstrap_mode || "image_factory_managed") !==
      "image_factory_managed"
    ) {
      toast.error(
        "Tenant namespace deprovision is only available in managed mode.",
      );
      return;
    }

    const confirmed = window.confirm(
      "Deprovision will delete the selected tenant namespace and all resources inside it. Continue?",
    );
    if (!confirmed) return;

    try {
      setTenantNamespacePrepareActionLoading(true);
      setTenantNamespacePrepareError(null);
      setShowTenantNamespacePrepareDrawer(true);
      const prepare = await infrastructureService.deprovisionTenantNamespace(
        id,
        effectiveTenantNamespaceId,
      );
      setTenantNamespacePrepare(prepare);
      if (prepare.status === "succeeded") {
        toast.success("Tenant namespace deprovisioned.");
      } else if (prepare.status === "failed") {
        toast.error("Tenant namespace deprovision failed.");
      } else {
        toast.success("Tenant namespace deprovision started.");
      }
    } catch (err) {
      const message =
        err instanceof Error
          ? err.message
          : "Failed to deprovision tenant namespace";
      setTenantNamespacePrepareError(message);
      toast.error(message);
    } finally {
      setTenantNamespacePrepareActionLoading(false);
      await loadTenantNamespacePrepareStatus();
    }
  }, [
    id,
    effectiveTenantNamespaceId,
    provider,
    isSystemAdmin,
    loadTenantNamespacePrepareStatus,
  ]);

  const handleReconcileStaleTenantNamespaces = useCallback(async () => {
    if (!id) return;
    if (!provider || !isSystemAdmin) {
      toast.error("Permission denied.");
      return;
    }
    if (!kubernetesProviderTypes.has(provider.provider_type)) return;
    if (
      (provider.bootstrap_mode || "image_factory_managed") !==
      "image_factory_managed"
    ) {
      toast.error(
        "Tenant namespace reconcile is only available in managed mode.",
      );
      return;
    }
    try {
      setTenantNamespaceReconcileLoading("stale");
      setTenantNamespacePrepareError(null);
      const summary =
        await infrastructureService.reconcileStaleTenantNamespaces(id);
      toast.success(
        `Reconcile stale completed: targeted ${summary?.targeted ?? 0}, applied ${summary?.applied ?? 0}, failed ${summary?.failed ?? 0}.`,
      );
      await loadTenantNamespacePrepareStatus();
    } catch (err) {
      const message =
        err instanceof Error
          ? err.message
          : "Failed to reconcile stale tenant namespaces";
      setTenantNamespacePrepareError(message);
      toast.error(message);
    } finally {
      setTenantNamespaceReconcileLoading(null);
    }
  }, [id, provider, isSystemAdmin, loadTenantNamespacePrepareStatus]);

  const handleReconcileSelectedTenantNamespace = useCallback(async () => {
    if (!id) return;
    if (
      !effectiveTenantNamespaceId ||
      effectiveTenantNamespaceId === NIL_TENANT_ID
    ) {
      toast.error("Select a tenant to reconcile.");
      return;
    }
    if (!provider || !isSystemAdmin) {
      toast.error("Permission denied.");
      return;
    }
    if (!kubernetesProviderTypes.has(provider.provider_type)) return;
    if (
      (provider.bootstrap_mode || "image_factory_managed") !==
      "image_factory_managed"
    ) {
      toast.error(
        "Tenant namespace reconcile is only available in managed mode.",
      );
      return;
    }
    try {
      setTenantNamespaceReconcileLoading("selected");
      setTenantNamespacePrepareError(null);
      const summary =
        await infrastructureService.reconcileSelectedTenantNamespaces(id, [
          effectiveTenantNamespaceId,
        ]);
      toast.success(
        `Reconcile selected completed: targeted ${summary?.targeted ?? 0}, applied ${summary?.applied ?? 0}, failed ${summary?.failed ?? 0}.`,
      );
      await loadTenantNamespacePrepareStatus();
    } catch (err) {
      const message =
        err instanceof Error
          ? err.message
          : "Failed to reconcile selected tenant namespace";
      setTenantNamespacePrepareError(message);
      toast.error(message);
    } finally {
      setTenantNamespaceReconcileLoading(null);
    }
  }, [
    id,
    effectiveTenantNamespaceId,
    provider,
    isSystemAdmin,
    loadTenantNamespacePrepareStatus,
  ]);

  // Load provider details
  const loadProvider = useCallback(async () => {
    if (!id) return;

    try {
      setLoading(true);
      setPageError(null);
      const response = await infrastructureService.getProvider(id);
      setProvider(response);
      setDraftIsGlobal(!!response.is_global);
      if (
        response.config?.tekton_install_mode === "gitops" ||
        response.config?.tekton_install_mode === "image_factory_installer"
      ) {
        setTektonInstallMode(response.config.tekton_install_mode);
      }
      if (
        response.config?.tekton_profile_version &&
        typeof response.config.tekton_profile_version === "string"
      ) {
        setTektonAssetVersion(response.config.tekton_profile_version);
      }

      // Fetch creator name if provider has created_by and user is system admin
      if (response.created_by && isSystemAdmin) {
        try {
          const userResponse = await userService.getUser(response.created_by);
          const fullName =
            `${userResponse.user.first_name} ${userResponse.user.last_name}`.trim();
          setCreatorName(fullName);
        } catch (userErr) {
          console.warn("Failed to fetch creator name:", userErr);
          setCreatorName(null);
        }
      } else {
        setCreatorName(null);
      }

      const tenantResponse = await adminService.getTenants({ limit: 200 });
      const filteredTenants = (tenantResponse.data || [])
        .filter((tenant) => tenant.id !== NIL_TENANT_ID)
        .map((tenant) => ({ id: tenant.id, name: tenant.name }));
      setTenants(filteredTenants);

      setPermissionsLoading(true);
      const perms = await infrastructureService.getProviderPermissions(id);
      const nextPermissions: Record<string, boolean> = {};
      perms.forEach((perm) => {
        if (perm.permission === "infrastructure:select") {
          nextPermissions[perm.tenant_id] = true;
        }
      });
      setTenantPermissions(nextPermissions);
      setDraftTenantPermissions(nextPermissions);
      await loadTektonStatus();
    } catch (err) {
      setPageError(
        err instanceof Error ? err.message : "Failed to load provider details",
      );
    } finally {
      setPermissionsLoading(false);
      setLoading(false);
    }
  }, [id, isSystemAdmin, loadTektonStatus]);

  useEffect(() => {
    loadProvider();
  }, [loadProvider]);

  useEffect(() => {
    if (!showPrepareDrawer) return;
    void loadPrepareStatus();
  }, [showPrepareDrawer, loadPrepareStatus]);

  useEffect(() => {
    void loadPrepareSnapshot();
  }, [loadPrepareSnapshot]);

  const handleRefreshPage = useCallback(async () => {
    if (!id) return;
    try {
      setRefreshing(true);
      setPageError(null);

      // Lightweight refresh: fetch provider/readiness state only.
      // Avoid prepare-runs/status endpoints on manual refresh.
      const response = await infrastructureService.getProvider(id);
      setProvider(response);
      setDraftIsGlobal(!!response.is_global);

      if (
        response.config?.tekton_install_mode === "gitops" ||
        response.config?.tekton_install_mode === "image_factory_installer"
      ) {
        setTektonInstallMode(response.config.tekton_install_mode);
      }
      if (
        response.config?.tekton_profile_version &&
        typeof response.config.tekton_profile_version === "string"
      ) {
        setTektonAssetVersion(response.config.tekton_profile_version);
      }

      toast.success("Provider status refreshed");
    } catch (err) {
      setPageError(
        err instanceof Error ? err.message : "Failed to refresh provider status",
      );
    } finally {
      setRefreshing(false);
    }
  }, [id]);

  useEffect(() => {
    if (!isSystemAdmin) {
      setNamespaceTenantId(
        selectedTenantId && selectedTenantId !== NIL_TENANT_ID
          ? selectedTenantId
          : "",
      );
      return;
    }
    if (namespaceTenantId) return;
    if (selectedTenantId && selectedTenantId !== NIL_TENANT_ID) {
      setNamespaceTenantId(selectedTenantId);
      return;
    }
    if (tenants.length > 0) {
      setNamespaceTenantId(tenants[0].id);
    }
  }, [isSystemAdmin, namespaceTenantId, selectedTenantId, tenants]);

  useEffect(() => {
    loadTenantNamespacePrepareStatus();
  }, [loadTenantNamespacePrepareStatus]);

  useEffect(() => {
    if (!id) return;
    const activeStatus = tektonStatus?.active_job?.status;
    if (activeStatus !== "pending" && activeStatus !== "running") {
      return;
    }
    const interval = window.setInterval(() => {
      loadTektonStatus({ showLoading: false });
    }, 5000);
    return () => {
      window.clearInterval(interval);
    };
  }, [id, loadTektonStatus, tektonStatus?.active_job?.status]);

  useEffect(() => {
    if (!id) return;
    if (!showPrepareDrawer) return;
    if (prepareWsConnected) return;
    const activeRun = prepareStatus?.active_run;
    const latestRun = prepareRuns[0];
    const trackedRun = latestRun || activeRun;
    if (!isPrepareRunActiveStatus(trackedRun?.status)) {
      return;
    }

    // Defensive stop condition: if status endpoint reports an older "active" run
    // but the latest run list entry is terminal, do not keep polling forever.
    if (
      latestRun &&
      activeRun &&
      latestRun.id !== activeRun.id &&
      !isPrepareRunActiveStatus(latestRun.status)
    ) {
      return;
    }
    const interval = window.setInterval(() => {
      loadPrepareStatus({ showLoading: false });
    }, 5000);
    return () => {
      window.clearInterval(interval);
    };
  }, [
    id,
    showPrepareDrawer,
    loadPrepareStatus,
    prepareStatus?.active_run?.status,
    prepareStatus?.active_run?.id,
    prepareRuns,
    prepareWsConnected,
  ]);

  useEffect(() => {
    if (!pendingPrepareCompletionRefresh) return;
    const latestRun = prepareStatus?.active_run || prepareRuns[0];
    if (!latestRun?.id) return;
    if (latestRun.status === "pending" || latestRun.status === "running") {
      return;
    }

    // Flip the flag first to avoid re-entrant refresh loops when this effect's
    // own network calls update prepare/provider state.
    setPendingPrepareCompletionRefresh(false);
    void refreshProviderReadinessAfterPrepare();
  }, [
    pendingPrepareCompletionRefresh,
    prepareStatus?.active_run,
    prepareRuns,
    refreshProviderReadinessAfterPrepare,
  ]);

  useEffect(() => {
    if (!showPrepareDrawer || !id || !token) {
      setPrepareWsConnected(false);
      return;
    }
    if (selectedPrepareRunStatus?.active_run?.id) {
      setPrepareWsConnected(false);
      return;
    }

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const query = new URLSearchParams({
      token: token,
    });
    if (selectedTenantId && selectedTenantId !== NIL_TENANT_ID) {
      query.set("tenant_id", selectedTenantId);
    }
    const wsUrl = `${protocol}//${window.location.host}/api/v1/admin/infrastructure/providers/${id}/prepare/stream?${query.toString()}`;
    const ws = new WebSocket(wsUrl);
    let reconnectTimer: number | null = null;
    let closedByEffect = false;

    ws.onopen = () => {
      setPrepareWsConnected(true);
    };

    ws.onclose = () => {
      setPrepareWsConnected(false);
      if (!closedByEffect) {
        reconnectTimer = window.setTimeout(() => {
          setPrepareWsRetryNonce((current) => current + 1);
        }, 1500);
      }
    };

    ws.onerror = () => {
      setPrepareWsConnected(false);
    };

    ws.onmessage = (messageEvent) => {
      try {
        const payload = JSON.parse(messageEvent.data || "{}");
        if (payload?.type !== "prepare_status") {
          return;
        }
        if (payload.status && typeof payload.status === "object") {
          setPrepareStatus(payload.status as ProviderPrepareStatus);
        }
        if (Array.isArray(payload.runs)) {
          setPrepareRuns(payload.runs as ProviderPrepareRun[]);
        }
      } catch {
        // ignore malformed stream payloads
      }
    };

    return () => {
      closedByEffect = true;
      if (reconnectTimer !== null) {
        window.clearTimeout(reconnectTimer);
      }
      setPrepareWsConnected(false);
      ws.close();
    };
  }, [
    showPrepareDrawer,
    id,
    token,
    selectedTenantId,
    selectedPrepareRunStatus,
    prepareWsRetryNonce,
  ]);

  useEffect(() => {
    if (
      !id ||
      !effectiveTenantNamespaceId ||
      effectiveTenantNamespaceId === NIL_TENANT_ID
    )
      return;
    if (tenantNamespacePrepareWsConnected) return;
    const prepareStatus = tenantNamespacePrepare?.status;
    if (prepareStatus !== "pending" && prepareStatus !== "running") return;
    const interval = window.setInterval(() => {
      loadTenantNamespacePrepareStatus();
    }, 5000);
    return () => {
      window.clearInterval(interval);
    };
  }, [
    id,
    effectiveTenantNamespaceId,
    tenantNamespacePrepare?.status,
    tenantNamespacePrepareWsConnected,
    loadTenantNamespacePrepareStatus,
  ]);

  useEffect(() => {
    if (
      !showTenantNamespacePrepareDrawer ||
      !id ||
      !token ||
      !effectiveTenantNamespaceId ||
      effectiveTenantNamespaceId === NIL_TENANT_ID
    ) {
      setTenantNamespacePrepareWsConnected(false);
      return;
    }

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/api/v1/admin/infrastructure/providers/${id}/tenants/${effectiveTenantNamespaceId}/provision-namespace/stream?token=${encodeURIComponent(token)}&tenant_id=${encodeURIComponent(effectiveTenantNamespaceId)}`;
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      setTenantNamespacePrepareWsConnected(true);
    };

    ws.onclose = () => {
      setTenantNamespacePrepareWsConnected(false);
    };

    ws.onerror = () => {
      setTenantNamespacePrepareWsConnected(false);
    };

    ws.onmessage = (messageEvent) => {
      try {
        const payload = JSON.parse(messageEvent.data || "{}");
        if (payload?.type !== "tenant_namespace_prepare_status") {
          return;
        }
        setTenantNamespacePrepare(
          (payload?.prepare as ProviderTenantNamespacePrepare | null) || null,
        );
      } catch {
        // ignore malformed stream payloads
      }
    };

    return () => {
      setTenantNamespacePrepareWsConnected(false);
      ws.close();
    };
  }, [
    showTenantNamespacePrepareDrawer,
    id,
    token,
    effectiveTenantNamespaceId,
  ]);

  // Handle test connection
  const handleTestConnection = async () => {
    if (!provider) return;

    try {
      setTestingConnection(true);
      setPageError(null);
      setTestError(null);
      setShowTestDrawer(true);
      setTestProgress("initializing");

      // Simulate initialization delay
      await new Promise((resolve) => setTimeout(resolve, 500));

      setTestProgress("connecting");

      const result = await infrastructureService.testProviderConnection(
        provider.id,
      );

      setTestProgress("receiving");

      if (result.success) {
        setTestProgress("completed");
        setPageError(null); // Clear any previous error
        setTestError(null);
        // Refresh provider data to get updated status
        await loadProvider();
        toast.success("Connection test successful!");
        // Keep drawer open to show success results - user can close manually
      } else {
        setTestProgress("failed");
        setTestError(result.message || "Connection test failed");
        // Keep drawer open to show error
      }
    } catch (err) {
      setTestProgress("failed");
      setTestError(
        err instanceof Error ? err.message : "Failed to test connection",
      );
    } finally {
      setTestingConnection(false);
    }
  };

  // Handle close test drawer
  const handleCloseTestDrawer = () => {
    setShowTestDrawer(false);
    setTestProgress(null);
    setTestError(null);
  };

  // Handle toggle status
  const handleToggleStatus = async () => {
    if (!provider) return;
    if (!canManageAdmin) {
      toast.error("Read-only mode.");
      return;
    }

    try {
      const enabled = provider.status !== "online";
      await infrastructureService.toggleProviderStatus(provider.id, enabled);
      // Refresh provider data
      await loadProvider();
    } catch (err) {
      setPageError(
        err instanceof Error ? err.message : "Failed to update provider status",
      );
    }
  };

  const handleTektonAction = async (
    action: "install" | "upgrade" | "validate",
  ) => {
    if (!provider) return;
    if (!canManageAdmin) {
      toast.error("Read-only mode.");
      return;
    }
    if (
      (provider.bootstrap_mode || "image_factory_managed") !==
      "image_factory_managed"
    ) {
      toast.error(
        "Tekton install/upgrade/validate is disabled for self-managed providers",
      );
      return;
    }

    try {
      setTektonActionLoading(action);
      setTektonError(null);

      const payload = {
        install_mode: tektonInstallMode,
        asset_version: tektonAssetVersion.trim() || "v1",
        idempotency_key: `${action}-${provider.id}-${tektonAssetVersion.trim() || "v1"}-${Date.now()}`,
      };

      if (action === "install") {
        await infrastructureService.installTekton(provider.id, payload);
      } else if (action === "upgrade") {
        await infrastructureService.upgradeTekton(provider.id, payload);
      } else {
        await infrastructureService.validateTekton(provider.id, payload);
      }

      toast.success(`Tekton ${action} job started`);
      await loadTektonStatus();
    } catch (err) {
      const message =
        err instanceof Error ? err.message : `Failed to ${action} Tekton`;
      setTektonError(message);
      toast.error(message);
    } finally {
      setTektonActionLoading(null);
    }
  };

  const handleRetryTektonJob = async (jobId: string) => {
    if (!provider) return;
    if (!canManageAdmin) {
      toast.error("Read-only mode.");
      return;
    }

    try {
      setTektonActionLoading("retry");
      setTektonError(null);
      await infrastructureService.retryTektonJob(provider.id, jobId);
      toast.success("Tekton retry job started");
      await loadTektonStatus();
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to retry Tekton job";
      setTektonError(message);
      toast.error(message);
    } finally {
      setTektonActionLoading(null);
    }
  };

  const handlePrepareProvider = async () => {
    if (!provider) return;
    if (!canManageAdmin) {
      toast.error("Read-only mode.");
      return;
    }
    try {
      setPrepareActionLoading(true);
      setPrepareError(null);
      setSelectedPrepareRunStatus(null);
      setSelectedPrepareRunError(null);
      setShowPrepareDrawer(true);
      await infrastructureService.prepareProvider(provider.id, {
        requested_actions: {
          connectivity: true,
          bootstrap: provider.bootstrap_mode === "image_factory_managed",
          readiness: true,
        },
      });
      setPendingPrepareCompletionRefresh(true);
      toast.success("Provider preparation started");
      await loadPrepareStatus();
    } catch (err) {
      const message =
        err instanceof Error
          ? err.message
          : "Failed to start provider preparation";
      setPrepareError(message);
      toast.error(message);
    } finally {
      setPrepareActionLoading(false);
    }
  };

  const handleRegenerateRuntimeAuth = async () => {
    if (!provider) return;
    if (!canManageAdmin) {
      toast.error("Read-only mode.");
      return;
    }
    if (provider.bootstrap_mode !== "image_factory_managed") {
      toast.error("Runtime auth regeneration is only available for managed providers");
      return;
    }
    try {
      setPrepareActionLoading(true);
      setPrepareError(null);
      setSelectedPrepareRunStatus(null);
      setSelectedPrepareRunError(null);
      setShowPrepareDrawer(true);
      await infrastructureService.prepareProvider(provider.id, {
        requested_actions: {
          connectivity: true,
          bootstrap: true,
          readiness: true,
          runtime_auth_regeneration: true,
        },
      });
      setPendingPrepareCompletionRefresh(true);
      toast.success("Runtime auth regeneration started");
      await loadPrepareStatus();
    } catch (err) {
      const message =
        err instanceof Error
          ? err.message
          : "Failed to start runtime auth regeneration";
      setPrepareError(message);
      toast.error(message);
    } finally {
      setPrepareActionLoading(false);
    }
  };

  const handleClosePrepareDrawer = () => {
    setShowPrepareDrawer(false);
    setSelectedPrepareRunStatus(null);
    setSelectedPrepareRunError(null);
  };

  const handleViewPrepareRun = async (runId: string) => {
    await loadPrepareRunDetails(runId);
    setShowPrepareDrawer(true);
  };

  const copyToClipboard = async (text: string, id: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedText(id);
      setTimeout(() => setCopiedText(null), 1500);
    } catch {
      toast.error("Failed to copy");
    }
  };

  const handleToggleGlobal = async (value: boolean) => {
    if (!provider) return;
    setDraftIsGlobal(value);
  };

  const handleTenantAccessChange = async (
    tenantId: string,
    enabled: boolean,
  ) => {
    setDraftTenantPermissions((prev) => ({ ...prev, [tenantId]: enabled }));
  };

  const handleCancelAccessEdit = () => {
    if (!provider) return;
    setDraftIsGlobal(!!provider.is_global);
    setDraftTenantPermissions(tenantPermissions);
  };

  const openAccessDrawer = () => {
    setShowAccessDrawer(true);
  };

  const closeAccessDrawer = () => {
    if (isSystemAdmin) {
      handleCancelAccessEdit();
    }
    setShowAccessDrawer(false);
  };

  const handleSaveAccess = async () => {
    if (!provider) return;
    if (!canManageAdmin) {
      toast.error("Read-only mode.");
      return;
    }
    try {
      setSavingPermissions(true);
      setPageError(null);

      const updates: Promise<any>[] = [];

      if (!!provider.is_global !== draftIsGlobal) {
        updates.push(
          infrastructureService.updateProvider(provider.id, {
            is_global: draftIsGlobal,
          }),
        );
      }

      const tenantIds = new Set([
        ...Object.keys(tenantPermissions || {}),
        ...Object.keys(draftTenantPermissions || {}),
      ]);

      tenantIds.forEach((tenantId) => {
        const current = !!tenantPermissions[tenantId];
        const next = !!draftTenantPermissions[tenantId];
        if (current === next) return;
        if (next) {
          updates.push(
            infrastructureService.grantProviderPermission(
              provider.id,
              tenantId,
            ),
          );
        } else {
          updates.push(
            infrastructureService.revokeProviderPermission(
              provider.id,
              tenantId,
            ),
          );
        }
      });

      if (updates.length > 0) {
        await Promise.all(updates);
      }

      setTenantPermissions(draftTenantPermissions);
      setProvider({ ...provider, is_global: draftIsGlobal });
      setShowAccessDrawer(false);
      toast.success("Access settings saved");
    } catch (err) {
      setPageError(
        err instanceof Error ? err.message : "Failed to update access settings",
      );
    } finally {
      setSavingPermissions(false);
    }
  };

  // Handle edit
  const handleEdit = () => {
    if (!provider) return;
    if (!canManageAdmin) {
      toast.error("Read-only mode.");
      return;
    }
    // Navigate back to providers page and request edit by id so the target page
    // always loads the freshest provider payload before opening the form.
    navigate("/admin/infrastructure", {
      state: { editingProviderId: provider.id },
    });
  };

  // Handle delete
  const handleDelete = async () => {
    if (!provider) return;
    if (!canManageAdmin) {
      toast.error("Read-only mode.");
      return;
    }

    if (
      !confirm(
        `Are you sure you want to delete the provider "${provider.display_name}"? This action cannot be undone.`,
      )
    ) {
      return;
    }

    try {
      await infrastructureService.deleteProvider(provider.id);
      navigate("/admin/infrastructure");
    } catch (err) {
      setPageError(
        err instanceof Error ? err.message : "Failed to delete provider",
      );
    }
  };

  useEffect(() => {
    if (location.hash !== "#prepare-checks-section") {
      return;
    }
    const timer = window.setTimeout(() => {
      const section = document.getElementById("prepare-checks-section");
      if (section) {
        section.scrollIntoView({ behavior: "smooth", block: "start" });
        return;
      }
      const fallback = document.getElementById("provider-preparation-section");
      if (fallback) {
        fallback.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    }, 50);
    return () => window.clearTimeout(timer);
  }, [location.hash, prepareStatus?.checks?.length]);

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <RefreshCw className="h-8 w-8 animate-spin mx-auto text-gray-400" />
          <p className="mt-2 text-gray-500">Loading provider details...</p>
        </div>
      </div>
    );
  }

  if (pageError || !provider) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <AlertCircle className="h-12 w-12 mx-auto text-red-400" />
          <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">
            {pageError || "Provider not found"}
          </h3>
          <button
            onClick={() => navigate("/admin/infrastructure")}
            className="mt-4 text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-200"
          >
            ← Back to Providers
          </button>
        </div>
      </div>
    );
  }

  const accessChangesDirty = (() => {
    if (!!provider.is_global !== draftIsGlobal) return true;
    const tenantIds = new Set([
      ...Object.keys(tenantPermissions || {}),
      ...Object.keys(draftTenantPermissions || {}),
    ]);
    for (const tenantId of tenantIds) {
      if (
        !!tenantPermissions[tenantId] !== !!draftTenantPermissions[tenantId]
      ) {
        return true;
      }
    }
    return false;
  })();
  const selectedTenants = tenants.filter(
    (tenant) => !!draftTenantPermissions[tenant.id],
  );
  const selectedTenantCount = selectedTenants.length;

  const activeEventTypes = new Set(
    (tektonStatus?.active_job_events || []).map((event) => event.event_type),
  );
  const hasActiveInstallerJob =
    tektonStatus?.active_job?.status === "pending" ||
    tektonStatus?.active_job?.status === "running";
  const hasActivePrepareRun = isPrepareRunActiveStatus(
    prepareStatus?.active_run?.status,
  );
  const isViewingHistoricalPrepareRun =
    !!selectedPrepareRunStatus?.active_run?.id;
  const displayedPrepareStatus = isViewingHistoricalPrepareRun
    ? selectedPrepareRunStatus
    : prepareStatus;
  const displayedPrepareRun = displayedPrepareStatus?.active_run;
  const displayedPrepareChecks = displayedPrepareStatus?.checks || [];
  const runtimeAuthRegenerationRequested =
    displayedPrepareRun?.requested_actions?.runtime_auth_regeneration === true;
  const runtimeAuthRegenerationStatus =
    typeof displayedPrepareRun?.result_summary?.runtime_auth_regeneration ===
    "string"
      ? displayedPrepareRun.result_summary.runtime_auth_regeneration
      : null;
  const runtimeAuthRegenerationApplied =
    runtimeAuthRegenerationStatus === "applied";
  const hasActiveDisplayedPrepareRun =
    !isViewingHistoricalPrepareRun &&
    isPrepareRunActiveStatus(displayedPrepareRun?.status);
  const hasActiveTenantNamespacePrepare = isPrepareRunActiveStatus(
    tenantNamespacePrepare?.status,
  );
  const tenantPrepareSummary = tenantNamespacePrepare?.result_summary || {};
  const tenantPrepareSteps: Array<{
    key: string;
    label: string;
    complete: boolean;
    value?: unknown;
  }> = [
    {
      key: "tekton_resources",
      label: "Tekton API preflight",
      complete: Object.prototype.hasOwnProperty.call(
        tenantPrepareSummary,
        "tekton_resources",
      ),
      value: tenantPrepareSummary.tekton_resources,
    },
    {
      key: "rbac_applied_objects",
      label: "Runtime RBAC applied",
      complete: Object.prototype.hasOwnProperty.call(
        tenantPrepareSummary,
        "rbac_applied_objects",
      ),
      value: tenantPrepareSummary.rbac_applied_objects,
    },
    {
      key: "assets_applied_objects",
      label: "Tekton assets applied",
      complete: Object.prototype.hasOwnProperty.call(
        tenantPrepareSummary,
        "assets_applied_objects",
      ),
      value: tenantPrepareSummary.assets_applied_objects,
    },
  ];
  const readinessStatus =
    tektonStatus?.readiness_status || provider.readiness_status || "unknown";
  const readinessMissingPrereqs =
    tektonStatus?.readiness_missing_prereqs ||
    provider.readiness_missing_prereqs ||
    [];
  const requiredTektonTasksForReadiness =
    tektonStatus?.required_tasks && tektonStatus.required_tasks.length > 0
      ? tektonStatus.required_tasks
      : ["git-clone", "docker-build", "buildx", "kaniko", "packer"];
  const requiredTektonPipelinesForReadiness =
    tektonStatus?.required_pipelines && tektonStatus.required_pipelines.length > 0
      ? tektonStatus.required_pipelines
      : [
          "image-factory-build-v1-docker",
          "image-factory-build-v1-buildx",
          "image-factory-build-v1-kaniko",
          "image-factory-build-v1-packer",
        ];
  const isProviderSchedulable =
    provider.status === "online" && provider.is_schedulable;
  const schedulableReason =
    typeof provider.schedulable_reason === "string"
      ? provider.schedulable_reason.trim()
      : "";
  const blockedBy = Array.isArray(provider.blocked_by) ? provider.blocked_by : [];
  const isManagedBootstrapProvider =
    (provider.bootstrap_mode || "image_factory_managed") ===
    "image_factory_managed";
  const visibleAuthConfig =
    provider?.config?.[
      isManagedBootstrapProvider ? "bootstrap_auth" : "runtime_auth"
    ];
  const visibleAuthLabel = isManagedBootstrapProvider
    ? "Bootstrap Authentication"
    : "Runtime Authentication";
  const configEntries = Object.entries(provider?.config || {}).filter(
    ([key]) =>
      key !== "bootstrap_auth" &&
      key !== "runtime_auth" &&
      !(
        key === "namespace" &&
        provider &&
        kubernetesProviderTypes.has(provider.provider_type)
      ),
  );
  const isSensitiveKey = (key: string): boolean => {
    const normalized = key.toLowerCase();
    return (
      normalized.includes("token") ||
      normalized.includes("password") ||
      normalized.includes("secret") ||
      normalized.includes("client_key") ||
      normalized.includes("private_key") ||
      normalized.endsWith("_key")
    );
  };
  const toggleSensitiveField = (fieldKey: string) => {
    setVisibleSensitiveFields((prev) => ({
      ...prev,
      [fieldKey]: !prev[fieldKey],
    }));
  };

  const hasEventPrefix = (prefix: string) =>
    Array.from(activeEventTypes).some((eventType) =>
      eventType.startsWith(prefix),
    );
  const scrollToPrepareChecks = () => {
    const section = document.getElementById("prepare-checks-section");
    if (section) {
      section.scrollIntoView({ behavior: "smooth", block: "start" });
      return;
    }
    const fallback = document.getElementById("provider-preparation-section");
    if (fallback) {
      fallback.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  };
  const prepareChecks = displayedPrepareChecks;
  const resolvePrepareStageStatus = (
    category: string,
  ): "pending" | "running" | "failed" | "passed" => {
    const checks = prepareChecks.filter((check) => check.category === category);
    if (checks.length === 0) {
      if (hasActiveDisplayedPrepareRun) return "running";
      return "pending";
    }
    if (checks.some((check) => check.ok === false)) return "failed";
    if (checks.every((check) => check.ok === true)) return "passed";
    return "running";
  };
  const prepareStageCards: Array<{
    key: string;
    label: string;
    status: "pending" | "running" | "failed" | "passed";
  }> = [
    {
      key: "connectivity",
      label: "Connectivity",
      status: resolvePrepareStageStatus("connectivity"),
    },
    {
      key: "permission_audit",
      label: "Permission Audit",
      status: resolvePrepareStageStatus("permission_audit"),
    },
    {
      key: "bootstrap",
      label: "Bootstrap",
      status: resolvePrepareStageStatus("bootstrap"),
    },
    {
      key: "readiness",
      label: "Readiness",
      status: resolvePrepareStageStatus("readiness"),
    },
  ];
  const prepareStageColor = (
    status: "pending" | "running" | "failed" | "passed",
  ) => {
    if (status === "passed")
      return "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200";
    if (status === "failed")
      return "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200";
    if (status === "running")
      return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200";
    return "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-200";
  };
  const sortedPrepareChecks = [...prepareChecks].sort(
    (a, b) =>
      new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
  );
  const connectivityChecks = [...sortedPrepareChecks]
    .filter((check) => check.category === "connectivity")
    .sort(
      (a, b) =>
        new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
    );
  const latestConnectivityCheck =
    connectivityChecks.length > 0
      ? connectivityChecks[connectivityChecks.length - 1]
      : undefined;
  const connectivityIssueDetected =
    readinessMissingPrereqs.some((item) => isLikelyConnectivityIssue(item)) ||
    sortedPrepareChecks.some(
      (check) =>
        !check.ok &&
        isLikelyConnectivityIssue(`${check.check_key} ${check.message}`),
    );
  const providerLooksReachableNow =
    readinessStatus === "ready" ||
    provider.health_status === "healthy" ||
    provider.health_status === "warning";
  const clusterConnectivityStatus: "reachable" | "unreachable" | "unknown" =
    providerLooksReachableNow
      ? "reachable"
      : latestConnectivityCheck != null
        ? latestConnectivityCheck.ok
          ? "reachable"
          : "unreachable"
        : connectivityIssueDetected
          ? "unreachable"
          : "unknown";
  const clusterConnectivityClass =
    clusterConnectivityStatus === "reachable"
      ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
      : clusterConnectivityStatus === "unreachable"
        ? "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
        : "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200";
  const healthIsUnhealthy =
    !!provider.health_status && provider.health_status !== "healthy";
  const healthTooltipText = (() => {
    if (!healthIsUnhealthy) {
      return "Health is derived from provider readiness and connectivity checks.";
    }
    const firstIssue = readinessMissingPrereqs[0];
    const reason = schedulableReason
      ? `Cause: ${schedulableReason}.`
      : "Cause: provider readiness checks are failing.";
    const issueText = firstIssue
      ? ` Primary blocker: ${firstIssue}.`
      : " No detailed blocker is currently available.";
    const guidance = firstIssue
      ? ` Guidance: ${guidanceForReadinessItem(firstIssue)}`
      : " Guidance: run Prepare Provider to refresh failing checks and remediation details.";
    return `${reason}${issueText}${guidance}`;
  })();
  const readinessTooltipText = (() => {
    if (readinessStatus === "ready") {
      return "Readiness checks passed. Provider is eligible for scheduling when status is online.";
    }
    if (readinessMissingPrereqs.length === 0) {
      return "Readiness is not ready, but prerequisites are not listed yet. Run Prepare Provider for details.";
    }
    const first = readinessMissingPrereqs[0];
    return `Not ready because: ${first}. Guidance: ${guidanceForReadinessItem(first)}`;
  })();
  const findLatestCheckByKey = (checkKey: string) => {
    const matches = sortedPrepareChecks.filter(
      (check) => check.check_key === checkKey,
    );
    return matches.length > 0 ? matches[matches.length - 1] : undefined;
  };
  const hasCheckFailureContaining = (needle: string) =>
    sortedPrepareChecks.some(
      (check) =>
        !check.ok &&
        `${check.message} ${check.check_key}`
          .toLowerCase()
          .includes(needle.toLowerCase()),
    );
  const readinessCheck = findLatestCheckByKey("readiness_eval");
  const readinessMissingFromCheck = Array.isArray(
    readinessCheck?.details?.missing_prereqs,
  )
    ? (readinessCheck?.details?.missing_prereqs as string[])
    : [];
  const readinessMissing =
    readinessMissingFromCheck.length > 0
      ? readinessMissingFromCheck
      : readinessMissingPrereqs;
  const hasMissingPrereq = (needle: string) =>
    readinessMissing.some((item) =>
      item.toLowerCase().includes(needle.toLowerCase()),
    );
  const providerConfigCheck = findLatestCheckByKey("provider_config");
  const kubernetesAPICheck = findLatestCheckByKey("kubernetes_api");
  const bootstrapApplyCheck = findLatestCheckByKey("bootstrap.apply");
  const permissionAuditChecks = sortedPrepareChecks.filter(
    (check) => check.category === "permission_audit",
  );
  const permissionAuditFailed = permissionAuditChecks.some(
    (check) => !check.ok,
  );
  const permissionAuditPassed =
    permissionAuditChecks.length > 0 &&
    permissionAuditChecks.every((check) => check.ok);
  const tektonAPICheckFailed = hasCheckFailureContaining(
    "tekton api preflight failed",
  );
  const tektonAPIReady =
    !!bootstrapApplyCheck?.ok ||
    (!!readinessCheck &&
      !hasCheckFailureContaining(
        "the server could not find the requested resource",
      ) &&
      !hasCheckFailureContaining("tekton api preflight failed"));
  const namespaceFailed = hasMissingPrereq("missing namespace");
  const tasksFailed = hasMissingPrereq("missing tekton task");
  const pipelinesFailed = hasMissingPrereq("missing tekton pipeline");
  const registrySecretFailed =
    hasMissingPrereq("docker-config") ||
    hasMissingPrereq("docker config") ||
    hasMissingPrereq("registry secret");
  const latestPrepareRun = prepareStatus?.active_run || prepareRuns[0];
  const latestPrepareRunFailed = latestPrepareRun?.status === "failed";
  const latestPrepareRunSucceeded = latestPrepareRun?.status === "succeeded";
  const latestPrepareRunRunning =
    latestPrepareRun?.status === "pending" ||
    latestPrepareRun?.status === "running";
  const hasPrepareRunHistory = !!latestPrepareRun;
  const sevenStepStatusColor = (
    status: "pending" | "running" | "failed" | "passed" | "blocked",
  ) => {
    if (status === "blocked")
      return "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200";
    return prepareStageColor(status);
  };
  const sevenStepState = (
    passed: boolean,
    failed: boolean,
    hasSignal: boolean,
  ): "pending" | "running" | "failed" | "passed" | "blocked" => {
    if (failed) return "failed";
    if (passed) return "passed";
    if (hasSignal && hasActiveDisplayedPrepareRun) return "running";
    return hasActiveDisplayedPrepareRun ? "running" : "pending";
  };
  const baseTektonWorkSequence: Array<{
    key: string;
    label: string;
    detail: string;
    status: "pending" | "running" | "failed" | "passed" | "blocked";
  }> = [
    {
      key: "cluster_connectivity",
      label: "1. Cluster Connectivity",
      detail: "Provider config resolves and Kubernetes API is reachable.",
      status: sevenStepState(
        !!providerConfigCheck?.ok && !!kubernetesAPICheck?.ok,
        providerConfigCheck?.ok === false || kubernetesAPICheck?.ok === false,
        !!providerConfigCheck || !!kubernetesAPICheck,
      ),
    },
    {
      key: "tekton_api",
      label: "2. Tekton APIs Available",
      detail:
        "Tekton API resources are discoverable and usable by checks/bootstrap.",
      status: sevenStepState(
        tektonAPIReady,
        tektonAPICheckFailed,
        !!bootstrapApplyCheck || !!readinessCheck,
      ),
    },
    {
      key: "target_namespace",
      label: "3. Target Namespace Exists",
      detail: "The provider target namespace is present.",
      status: sevenStepState(
        !!readinessCheck && !namespaceFailed,
        namespaceFailed,
        !!readinessCheck,
      ),
    },
    {
      key: "required_tasks",
      label: "4. Required Tasks Present",
      detail:
        `${requiredTektonTasksForReadiness.join(", ")} are available.`,
      status: sevenStepState(
        !!readinessCheck && !tasksFailed,
        tasksFailed,
        !!readinessCheck,
      ),
    },
    {
      key: "required_pipelines",
      label: "5. Required Pipelines Present",
      detail: `${requiredTektonPipelinesForReadiness.join(", ")} are present in the target namespace.`,
      status: sevenStepState(
        !!readinessCheck && !pipelinesFailed,
        pipelinesFailed,
        !!readinessCheck,
      ),
    },
    {
      key: "runtime_permissions",
      label: "6. Runtime Permissions Valid",
      detail: "Permission audit validates required runtime/build verbs.",
      status: sevenStepState(
        permissionAuditPassed,
        permissionAuditFailed,
        permissionAuditChecks.length > 0,
      ),
    },
    {
      key: "registry_secret_flow",
      label: "7. Registry Secret Flow Ready",
      detail: "Registry docker-config secret prerequisite is satisfied.",
      status: sevenStepState(
        !!readinessCheck && !registrySecretFailed,
        registrySecretFailed,
        !!readinessCheck,
      ),
    },
  ];
  const tektonWorkSequence = (() => {
    // Post-process statuses so "blocked" only appears for steps after the first real failure
    // when a run has completed as failed. This prevents Step 1 from ever being shown as blocked.
    const steps = baseTektonWorkSequence.map((step) => ({ ...step }));
    if (!hasActivePrepareRun && latestPrepareRunFailed) {
      const firstFailureIdx = steps.findIndex(
        (step) => step.status === "failed",
      );
      if (firstFailureIdx >= 0) {
        for (let i = firstFailureIdx + 1; i < steps.length; i++) {
          if (steps[i].status === "pending" || steps[i].status === "running") {
            steps[i].status = "blocked";
          }
        }
      } else if (steps.every((step) => step.status === "pending")) {
        // If the run failed but no check signals were captured, surface it at the start.
        steps[0].status = "failed";
      }
    }
    return steps;
  })();
  const tektonProgressSteps = [
    {
      key: "requested",
      label: "Requested",
      done:
        hasEventPrefix("install.requested") ||
        hasEventPrefix("upgrade.requested") ||
        hasEventPrefix("validate.requested"),
    },
    {
      key: "started",
      label: "Started",
      done: activeEventTypes.has("job.started"),
    },
    {
      key: "apply",
      label: "Assets Applied",
      done:
        hasEventPrefix("install.apply.completed") ||
        hasEventPrefix("upgrade.apply.completed") ||
        activeEventTypes.has("validate.completed"),
    },
    {
      key: "done",
      label: "Completed",
      done: activeEventTypes.has("job.succeeded"),
    },
  ];
  type OnboardingStepStatus = "pending" | "running" | "blocked" | "done" | "na";
  const onboardingStepClass = (status: OnboardingStepStatus) => {
    if (status === "done") {
      return "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200";
    }
    if (status === "blocked") {
      return "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200";
    }
    if (status === "running") {
      return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200";
    }
    if (status === "na") {
      return "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200";
    }
    return "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-200";
  };
  const connectivityStepStatus: OnboardingStepStatus = hasActivePrepareRun
    ? "running"
    : providerConfigCheck?.ok === false || kubernetesAPICheck?.ok === false
      ? "blocked"
      : providerConfigCheck?.ok && kubernetesAPICheck?.ok
        ? "done"
        : provider.health_status === "healthy" ||
            provider.health_status === "warning" ||
            readinessStatus === "ready"
          ? "done"
          : provider.health_status === "unhealthy"
            ? "blocked"
            : "pending";
  const prepareStepStatus: OnboardingStepStatus = hasActivePrepareRun
    ? "running"
    : latestPrepareRunSucceeded
      ? "done"
      : latestPrepareRunFailed
        ? "blocked"
        : readinessStatus === "ready" || isProviderSchedulable
          ? "done"
          : "pending";
  const readinessStepStatus: OnboardingStepStatus = latestPrepareRunRunning
    ? "running"
    : isProviderSchedulable
      ? "done"
      : hasPrepareRunHistory
        ? "blocked"
        : "pending";
  const tenantProvisionRequired =
    isManagedBootstrapProvider && isSystemAdmin && !!effectiveTenantNamespaceId;
  const tenantProvisionStepStatus: OnboardingStepStatus = !isManagedBootstrapProvider
    ? "na"
    : !isSystemAdmin
      ? "blocked"
      : !effectiveTenantNamespaceId
        ? "pending"
        : hasActiveTenantNamespacePrepare
          ? "running"
          : tenantNamespacePrepare?.status === "succeeded"
            ? "done"
            : tenantNamespacePrepare?.status === "failed"
              ? "blocked"
              : "pending";
  const onboardingSteps: Array<{
    key: string;
    label: string;
    detail: string;
    status: OnboardingStepStatus;
  }> = [
    {
      key: "connectivity",
      label: "1. Connectivity + Auth",
      detail: "Validate provider config and Kubernetes API reachability.",
      status: connectivityStepStatus,
    },
    {
      key: "prepare",
      label: "2. Prepare Provider",
      detail: "Run the orchestration flow for checks, bootstrap, and readiness.",
      status: prepareStepStatus,
    },
    {
      key: "readiness",
      label: "3. Scheduling Gate",
      detail: "Provider must be online, ready, and schedulable.",
      status: readinessStepStatus,
    },
    {
      key: "tenant-namespace",
      label: "4. Tenant Namespace",
      detail: !isManagedBootstrapProvider
        ? "Not required for self-managed providers."
        : !isSystemAdmin
          ? "System admin access is required for tenant namespace provisioning."
          : "Prepare namespace assets for the selected tenant before builds.",
      status: tenantProvisionStepStatus,
    },
  ];
  const onboardingComplete =
    connectivityStepStatus === "done" &&
    prepareStepStatus === "done" &&
    readinessStepStatus === "done" &&
    (!tenantProvisionRequired || tenantProvisionStepStatus === "done");

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <button
            onClick={() => navigate("/admin/infrastructure")}
            className="text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200"
          >
            <ArrowLeft className="h-5 w-5" />
          </button>
          <div>
            <div className="flex items-center space-x-3">
              {providerTypeIcons[provider.provider_type]}
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                {provider.display_name}
              </h1>
              <span
                className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColors[provider.status]}`}
              >
                {provider.status}
              </span>
            </div>
            <p className="mt-1 text-gray-600 dark:text-gray-400">
              {provider.name} • {provider.provider_type}
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <button
            onClick={handleRefreshPage}
            disabled={loading || refreshing}
            className="border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 px-4 py-2 rounded-md hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-50 flex items-center gap-2"
          >
            <RefreshCw className={`h-4 w-4 ${refreshing ? "animate-spin" : ""}`} />
            Refresh
          </button>
          <button
            onClick={handleTestConnection}
            disabled={testingConnection}
            className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"
          >
            {testingConnection ? (
              <RefreshCw className="h-4 w-4 animate-spin" />
            ) : (
              <TestTube className="h-4 w-4" />
            )}
            Test Connection
          </button>
          {canManageAdmin && (
            <button
              onClick={handleToggleStatus}
              className={`px-4 py-2 rounded-md flex items-center gap-2 ${
                provider.status === "online"
                  ? "bg-red-600 text-white hover:bg-red-700"
                  : "bg-green-600 text-white hover:bg-green-700"
              }`}
            >
              {provider.status === "online" ? (
                <X className="h-4 w-4" />
              ) : (
                <Check className="h-4 w-4" />
              )}
              {provider.status === "online" ? "Disable" : "Enable"}
            </button>
          )}
          {canManageAdmin && (
            <button
              onClick={handleEdit}
              className="bg-gray-600 text-white px-4 py-2 rounded-md hover:bg-gray-700 flex items-center gap-2"
            >
              <Edit2 className="h-4 w-4" />
              Edit
            </button>
          )}
          {canManageAdmin && (
            <button
              onClick={handleDelete}
              className="bg-red-600 text-white px-4 py-2 rounded-md hover:bg-red-700 flex items-center gap-2"
            >
              <Trash2 className="h-4 w-4" />
              Delete
            </button>
          )}
        </div>
      </div>

      {/* Error Display */}
      {pageError && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md p-4">
          <div className="flex items-center">
            <AlertCircle className="h-5 w-5 text-red-600 mr-2" />
            <span className="text-red-800 dark:text-red-200">{pageError}</span>
          </div>
        </div>
      )}

      {/* Test Connection Drawer */}
      <Drawer
        isOpen={showTestDrawer}
        onClose={handleCloseTestDrawer}
        title="Testing Connection"
        description="Testing connectivity to the infrastructure provider"
      >
        <div className="space-y-6">
          {/* Progress Steps */}
          <div className="space-y-4">
            <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-3">
              Test Progress
            </h3>

            <div className="space-y-3">
              <div
                className={`flex items-center space-x-3 ${testProgress === "initializing" ? "text-blue-600 dark:text-blue-400" : "text-gray-400 dark:text-gray-500"}`}
              >
                <div
                  className={`w-6 h-6 rounded-full flex items-center justify-center ${testProgress === "initializing" ? "bg-blue-100 dark:bg-blue-900" : "bg-gray-100 dark:bg-gray-800"}`}
                >
                  {testProgress === "initializing" && (
                    <RefreshCw className="h-4 w-4 animate-spin" />
                  )}
                  {testProgress === "connecting" && (
                    <RefreshCw className="h-4 w-4 animate-spin" />
                  )}
                  {testProgress === "receiving" && (
                    <RefreshCw className="h-4 w-4 animate-spin" />
                  )}
                  {testProgress === "completed" && (
                    <Check className="h-4 w-4 text-green-600 dark:text-green-400" />
                  )}
                  {testProgress === "failed" && (
                    <X className="h-4 w-4 text-red-600 dark:text-red-400" />
                  )}
                </div>
                <span className="text-sm">
                  {testProgress === "initializing" && "Initializing test..."}
                  {testProgress === "connecting" && "Connecting to provider..."}
                  {testProgress === "receiving" && "Receiving response..."}
                  {testProgress === "completed" && "Connection test completed"}
                  {testProgress === "failed" && "Connection test failed"}
                </span>
              </div>
            </div>

            {/* Provider Info */}
            <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
              <h4 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                Provider Information
              </h4>
              <dl className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <dt className="text-gray-500 dark:text-gray-400">
                    Provider Type:
                  </dt>
                  <dd className="text-gray-900 dark:text-white font-medium">
                    {provider?.provider_type}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-gray-500 dark:text-gray-400">
                    Provider Name:
                  </dt>
                  <dd className="text-gray-900 dark:text-white font-medium">
                    {provider?.display_name}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-gray-500 dark:text-gray-400">Status:</dt>
                  <dd className="text-gray-900 dark:text-white font-medium">
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColors[provider?.status || "pending"]}`}
                    >
                      {provider?.status}
                    </span>
                  </dd>
                </div>
                {provider?.health_status && (
                  <div className="flex justify-between">
                    <dt className="text-gray-500 dark:text-gray-400">
                      Health Status:
                    </dt>
                    <dd className="text-gray-900 dark:text-white font-medium">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                          provider.health_status === "healthy"
                            ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                            : provider.health_status === "warning"
                              ? "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
                              : "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                        }`}
                      >
                        {provider.health_status}
                      </span>
                    </dd>
                  </div>
                )}
                {provider?.last_health_check && (
                  <div className="flex justify-between">
                    <dt className="text-gray-500 dark:text-gray-400">
                      Last Health Check:
                    </dt>
                    <dd className="text-gray-900 dark:text-white font-medium">
                      {new Date(provider.last_health_check).toLocaleString()}
                    </dd>
                  </div>
                )}
              </dl>
            </div>

            {/* Configuration Details */}
            <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
              <h4 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                Configuration
              </h4>
              <dl className="space-y-2 text-sm">
                {provider &&
                  provider?.provider_type === "kubernetes" &&
                  (visibleAuthConfig?.auth_method ||
                    provider?.config?.auth_method) && (
                    <div className="flex justify-between">
                      <dt className="text-gray-500 dark:text-gray-400">
                        Authentication Method:
                      </dt>
                      <dd className="text-gray-900 dark:text-white font-medium capitalize">
                        {(
                          visibleAuthConfig?.auth_method ||
                          provider.config?.auth_method ||
                          "unknown"
                        ).replace("_", " ")}
                      </dd>
                    </div>
                  )}
                {provider?.config?.apiServer && (
                  <div className="flex justify-between">
                    <dt className="text-gray-500 dark:text-gray-400">
                      API Server Endpoint:
                    </dt>
                    <dd className="text-gray-900 dark:text-white font-medium font-mono text-xs">
                      {provider.config.apiServer}
                    </dd>
                  </div>
                )}
                {provider?.config?.endpoint &&
                  provider?.provider_type !== "kubernetes" && (
                    <div className="flex justify-between">
                      <dt className="text-gray-500 dark:text-gray-400">
                        API Endpoint:
                      </dt>
                      <dd className="text-gray-900 dark:text-white font-medium font-mono text-xs">
                        {provider.config.endpoint}
                      </dd>
                    </div>
                  )}
                {provider?.config?.cluster_endpoint && (
                  <div className="flex justify-between">
                    <dt className="text-gray-500 dark:text-gray-400">
                      Cluster Endpoint:
                    </dt>
                    <dd className="text-gray-900 dark:text-white font-medium font-mono text-xs">
                      {provider.config.cluster_endpoint}
                    </dd>
                  </div>
                )}
                {provider?.config?.namespace &&
                  !kubernetesProviderTypes.has(provider.provider_type) && (
                    <div className="flex justify-between">
                      <dt className="text-gray-500 dark:text-gray-400">
                        Namespace:
                      </dt>
                      <dd className="text-gray-900 dark:text-white font-medium">
                        {provider.config.namespace}
                      </dd>
                    </div>
                  )}
                {provider?.config?.region && (
                  <div className="flex justify-between">
                    <dt className="text-gray-500 dark:text-gray-400">
                      Region:
                    </dt>
                    <dd className="text-gray-900 dark:text-white font-medium">
                      {provider.config.region}
                    </dd>
                  </div>
                )}
                {provider?.config?.kubeconfig_path && (
                  <div className="flex justify-between">
                    <dt className="text-gray-500 dark:text-gray-400">
                      Kubeconfig Path:
                    </dt>
                    <dd className="text-gray-900 dark:text-white font-medium font-mono text-xs">
                      {provider.config.kubeconfig_path}
                    </dd>
                  </div>
                )}
                {provider?.capabilities && provider.capabilities.length > 0 && (
                  <div className="flex justify-between">
                    <dt className="text-gray-500 dark:text-gray-400">
                      Capabilities:
                    </dt>
                    <dd className="text-gray-900 dark:text-white font-medium">
                      <div className="flex flex-wrap gap-1">
                        {provider.capabilities.map((capability, index) => (
                          <span
                            key={index}
                            className="inline-flex items-center px-2 py-1 rounded-md text-xs bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
                          >
                            {capability}
                          </span>
                        ))}
                      </div>
                    </dd>
                  </div>
                )}
              </dl>
            </div>

            {/* Test Details */}
            {testProgress === "completed" && (
              <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
                <h4 className="text-sm font-medium text-green-900 dark:text-green-200 mb-2">
                  Test Results
                </h4>
                <div className="flex items-center space-x-2">
                  <Check className="h-5 w-5 text-green-600" />
                  <span className="text-sm text-green-800 dark:text-green-200">
                    Successfully connected to provider
                  </span>
                </div>
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">
                  The provider is now online and ready to use.
                </p>
              </div>
            )}

            {testProgress === "failed" && (
              <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                <h4 className="text-sm font-medium text-red-900 dark:text-red-200 mb-2">
                  Test Failed
                </h4>
                <div className="flex items-center space-x-2">
                  <X className="h-5 w-5 text-red-600" />
                  <span className="text-sm text-red-800 dark:text-red-200">
                    Connection test failed
                  </span>
                </div>
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">
                  {testError ||
                    "Please check your provider configuration and try again."}
                </p>
              </div>
            )}
          </div>

          {/* Close Button */}
          <div className="flex justify-end pt-4 border-t border-gray-200 dark:border-gray-700">
            <button
              onClick={handleCloseTestDrawer}
              className="px-4 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-700 dark:bg-gray-700 dark:hover:bg-gray-600 font-medium"
            >
              Close
            </button>
          </div>
        </div>
      </Drawer>

      <TooltipDrawer
        isOpen={showBootstrapRBACDrawer}
        onClose={() => setShowBootstrapRBACDrawer(false)}
        title={
          isManagedBootstrapProvider
            ? "Image Factory Managed Bootstrap RBAC"
            : "Self-Managed Runtime RBAC"
        }
      >
        <div className="space-y-4 text-sm text-gray-700 dark:text-gray-300">
          {isManagedBootstrapProvider ? (
            <p>
              Image Factory managed mode only needs bootstrap credentials here.
              During provider/tenant prepare, Image Factory creates and manages
              runtime service accounts and RBAC automatically.
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

          {isManagedBootstrapProvider ? (
            <>
              <CopyableCodeBlock
                title="1) System Namespace + Service Accounts"
                code={bootstrapNamespaceTemplate}
                language="yaml"
              />
              <CopyableCodeBlock
                title="2) Bootstrap ClusterRole + Binding (Managed Prepare)"
                code={bootstrapClusterRoleTemplate}
                language="yaml"
              />
              <CopyableCodeBlock
                title="3) Bootstrap ServiceAccount Token (Recommended)"
                code={serviceAccountTokenTemplate}
                language="bash"
              />
            </>
          ) : (
            <>
              <CopyableCodeBlock
                title="1) System Namespace + Runtime Service Account"
                code={runtimeNamespaceTemplate}
                language="yaml"
              />
              <CopyableCodeBlock
                title="2) Tenant Namespace Runtime Role + Binding (Least Privilege)"
                code={runtimeRoleTemplate}
                language="yaml"
              />
              <CopyableCodeBlock
                title="3) Runtime ServiceAccount Token Setup"
                code={runtimeServiceAccountTokenTemplate}
                language="bash"
              />
            </>
          )}
        </div>
      </TooltipDrawer>

      {/* Provider Details */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Basic Information */}
        <div className="bg-white dark:bg-gray-800 shadow rounded-lg lg:col-span-2">
          <div className="px-4 py-5 sm:p-6">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
              Basic Information
            </h3>
            <dl className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-x-8 gap-y-4">
              <div className="min-w-0">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
                  Internal Name
                </dt>
                <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                  {provider.name}
                </dd>
              </div>
              <div className="min-w-0">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
                  Display Name
                </dt>
                <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                  {provider.display_name}
                </dd>
              </div>
              <div className="min-w-0">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
                  Provider Type
                </dt>
                <dd className="mt-1 text-sm text-gray-900 dark:text-white capitalize">
                  {provider.provider_type.replace("_", " ")}
                </dd>
              </div>
              <div className="min-w-0">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
                  Status
                </dt>
                <dd className="mt-1">
                  <span
                    className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColors[provider.status]}`}
                  >
                    {provider.status}
                  </span>
                </dd>
              </div>
              <div className="min-w-0">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
                  <span className="inline-flex items-center gap-2">
                    Health Status
                    <HelpTooltip
                      text={healthTooltipText}
                      sticky
                      trigger={
                        healthIsUnhealthy ? (
                          <AlertCircle className="h-3 w-3" />
                        ) : undefined
                      }
                      buttonClassName={
                        healthIsUnhealthy
                          ? "border-red-300 text-red-600 dark:border-red-700 dark:text-red-300"
                          : ""
                      }
                    />
                  </span>
                </dt>
                <dd className="mt-1">
                  <span
                    className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                      provider.health_status === "healthy"
                        ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                        : provider.health_status === "warning"
                          ? "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
                          : provider.health_status === "unhealthy"
                            ? "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                            : "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200"
                    }`}
                  >
                    {provider.health_status || "unknown"}
                  </span>
                </dd>
              </div>
              <div className="min-w-0">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
                  Cluster Connectivity
                </dt>
                <dd className="mt-1">
                  <span
                    className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${clusterConnectivityClass}`}
                  >
                    {clusterConnectivityStatus}
                  </span>
                </dd>
              </div>
              <div className="min-w-0">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
                  Last Health Check
                </dt>
                <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                  {provider.last_health_check
                    ? new Date(provider.last_health_check).toLocaleString()
                    : "Never checked"}
                </dd>
              </div>
              <div className="min-w-0">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
                  Access
                </dt>
                <dd className="mt-1 flex items-center justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${draftIsGlobal ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" : "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"}`}
                    >
                      {draftIsGlobal ? "Global" : "Tenant-restricted"}
                    </span>
                    {!draftIsGlobal && (
                      <span className="text-xs text-gray-600 dark:text-gray-300">
                        {selectedTenantCount} tenant
                        {selectedTenantCount === 1 ? "" : "s"}
                      </span>
                    )}
                  </div>
                  <button
                    onClick={openAccessDrawer}
                    className="inline-flex items-center px-2.5 py-1 text-xs font-medium text-blue-700 dark:text-blue-300 bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-800 rounded hover:bg-blue-100 dark:hover:bg-blue-900/50"
                  >
                    {isSystemAdmin ? "Manage" : "View"}
                  </button>
                </dd>
              </div>
              {kubernetesProviderTypes.has(provider.provider_type) && (
                <div className="min-w-0">
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
                    Tekton Enabled
                  </dt>
                  <dd className="mt-1">
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        provider.config?.tekton_enabled === true
                          ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                          : provider.config?.tekton_enabled === false
                            ? "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                            : "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200"
                      }`}
                    >
                      {provider.config?.tekton_enabled === true
                        ? "Enabled"
                        : provider.config?.tekton_enabled === false
                          ? "Disabled"
                          : "Not set"}
                    </span>
                  </dd>
                </div>
              )}
              {kubernetesProviderTypes.has(provider.provider_type) && (
                <div className="min-w-0">
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
                    Quarantine Dispatch
                  </dt>
                  <dd className="mt-1">
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        provider.config?.quarantine_dispatch_enabled === true
                          ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                          : provider.config?.quarantine_dispatch_enabled ===
                              false
                            ? "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                            : "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200"
                      }`}
                    >
                      {provider.config?.quarantine_dispatch_enabled === true
                        ? "Enabled"
                        : provider.config?.quarantine_dispatch_enabled === false
                          ? "Disabled"
                          : "Not set"}
                    </span>
                  </dd>
                </div>
              )}
              {kubernetesProviderTypes.has(provider.provider_type) && (
                <div className="min-w-0">
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
                    Bootstrap Mode
                  </dt>
                  <dd className="mt-1 flex items-center gap-2">
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">
                      {(
                        provider.bootstrap_mode || "image_factory_managed"
                      ).replace("_", " ")}
                    </span>
                    <button
                      type="button"
                      onClick={() => setShowBootstrapRBACDrawer(true)}
                      className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                      title="Recommended service account and RBAC manifests"
                    >
                      <HelpCircle className="h-4 w-4" />
                    </button>
                  </dd>
                </div>
              )}
              <div className="min-w-0">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
                  Created By
                </dt>
                <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                  {creatorName || provider.created_by}
                </dd>
              </div>
              <div className="min-w-0">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
                  Created At
                </dt>
                <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                  {new Date(provider.created_at).toLocaleString()}
                </dd>
              </div>
              <div className="min-w-0">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
                  Last Updated
                </dt>
                <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                  {new Date(provider.updated_at).toLocaleString()}
                </dd>
              </div>
              {provider.capabilities && provider.capabilities.length > 0 && (
                <div className="min-w-0 md:col-span-2 lg:col-span-3">
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">
                    Capabilities
                  </dt>
                  <dd className="mt-1">
                    <div className="flex flex-wrap gap-2">
                      {provider.capabilities.map((capability, index) => (
                        <span
                          key={index}
                          className="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
                        >
                          {capability}
                        </span>
                      ))}
                    </div>
                  </dd>
                </div>
              )}
            </dl>
          </div>
        </div>

        <Drawer
          isOpen={showAccessDrawer}
          onClose={closeAccessDrawer}
          title="Manage Provider Access"
          description="Define whether this provider is global or tenant-restricted."
          width="md"
        >
          <div className="space-y-4">
            <div className="flex items-start gap-3 rounded-md border border-gray-200 dark:border-gray-700 p-3 bg-gray-50 dark:bg-gray-900/40">
              <input
                id="provider_is_global_detail"
                type="checkbox"
                checked={draftIsGlobal}
                onChange={(e) => handleToggleGlobal(e.target.checked)}
                disabled={!isSystemAdmin}
                className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <div>
                <label
                  htmlFor="provider_is_global_detail"
                  className="text-sm font-medium text-gray-700 dark:text-gray-200"
                >
                  Global provider (available to all tenants)
                </label>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  Disable this to grant access only to selected tenants.
                </p>
              </div>
            </div>

            {!draftIsGlobal && (
              <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                <div className="flex items-center justify-between mb-2">
                  <div>
                    <div className="text-sm font-medium text-gray-700 dark:text-gray-200">
                      Tenant Access
                    </div>
                    <div className="text-xs text-gray-500 dark:text-gray-400">
                      Select tenants that can use this provider.
                    </div>
                  </div>
                  {permissionsLoading && (
                    <span className="text-xs text-gray-500 dark:text-gray-400">
                      Loading…
                    </span>
                  )}
                </div>
                <div className="space-y-2 max-h-64 overflow-y-auto">
                  {tenants.length === 0 && (
                    <div className="text-xs text-gray-500 dark:text-gray-400">
                      No tenants found.
                    </div>
                  )}
                  {tenants.map((tenant) => (
                    <label
                      key={tenant.id}
                      className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200"
                    >
                      <input
                        type="checkbox"
                        checked={!!draftTenantPermissions[tenant.id]}
                        onChange={(e) =>
                          handleTenantAccessChange(tenant.id, e.target.checked)
                        }
                        disabled={!isSystemAdmin || savingPermissions}
                        className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                      />
                      <span>{tenant.name}</span>
                    </label>
                  ))}
                </div>
              </div>
            )}
          </div>

          <div className="flex justify-end gap-2 pt-4 border-t border-gray-200 dark:border-gray-700 mt-4">
            <button
              onClick={closeAccessDrawer}
              className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded hover:bg-gray-200 dark:hover:bg-gray-600"
              disabled={savingPermissions}
            >
              Close
            </button>
            {isSystemAdmin && (
              <button
                onClick={handleSaveAccess}
                className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-white bg-blue-600 border border-blue-600 rounded hover:bg-blue-700 disabled:opacity-50"
                disabled={savingPermissions || !accessChangesDirty}
              >
                {savingPermissions ? "Saving..." : "Save Access"}
              </button>
            )}
          </div>
        </Drawer>

        {/* Tekton Operations */}
        <Drawer
          isOpen={showPrepareDrawer}
          onClose={handleClosePrepareDrawer}
          title="Provider Preparation Progress"
          description={
            isViewingHistoricalPrepareRun
              ? "Read-only details for a previous prepare run."
              : "Live execution status for connectivity, bootstrap, and readiness checks."
          }
          width="lg"
        >
          <div className="space-y-4">
            <div className="rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/40 p-3">
              <div className="flex flex-wrap items-center gap-2 text-sm">
                <span
                  className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                    isViewingHistoricalPrepareRun
                      ? "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
                      : prepareWsConnected
                        ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                        : "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-200"
                  }`}
                >
                  {isViewingHistoricalPrepareRun
                    ? "Historical"
                    : prepareWsConnected
                      ? "Live"
                      : "Polling"}
                </span>
                <span className="text-gray-600 dark:text-gray-300">Run</span>
                <span className="font-mono text-xs text-gray-700 dark:text-gray-200">
                  {displayedPrepareRun?.id || "No run"}
                </span>
                <span className="hidden md:inline text-gray-400 dark:text-gray-500">
                  •
                </span>
                <span className="text-gray-600 dark:text-gray-300">Status</span>
                <span
                  className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                    displayedPrepareRun?.status === "succeeded"
                      ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                      : displayedPrepareRun?.status === "failed"
                        ? "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                        : displayedPrepareRun?.status
                          ? "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
                          : "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-200"
                  }`}
                >
                  {displayedPrepareRun?.status || "idle"}
                </span>
                {runtimeAuthRegenerationRequested && (
                  <span
                    className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                      runtimeAuthRegenerationApplied
                        ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                        : "bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-200"
                    }`}
                  >
                    {runtimeAuthRegenerationApplied
                      ? "Runtime Auth Regenerated"
                      : "Runtime Auth Regeneration Requested"}
                  </span>
                )}
                {hasActiveDisplayedPrepareRun && (
                  <RefreshCw className="h-4 w-4 animate-spin text-blue-500 dark:text-blue-300" />
                )}
              </div>
              {selectedPrepareRunError && (
                <div className="mt-2 text-xs text-red-700 dark:text-red-300">
                  {selectedPrepareRunError}
                </div>
              )}
              {displayedPrepareRun?.error_message && (
                <div className="mt-2 text-xs text-red-700 dark:text-red-300">
                  {displayedPrepareRun.error_message}
                </div>
              )}
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
              {prepareStageCards.map((stage) => (
                <div
                  key={stage.key}
                  className="rounded-md border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-2"
                >
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-700 dark:text-gray-200">
                      {stage.label}
                    </span>
                    <span
                      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${prepareStageColor(stage.status)}`}
                    >
                      {stage.status}
                    </span>
                  </div>
                </div>
              ))}
            </div>

            <div className="rounded-md border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
              <div className="px-3 py-2 border-b border-gray-200 dark:border-gray-700 text-sm font-medium text-gray-700 dark:text-gray-200">
                Tekton Runtime Sequence (7 Steps)
              </div>
              <div className="p-3 space-y-2">
                {tektonWorkSequence.map((step) => (
                  <div
                    key={step.key}
                    className="rounded-md border border-gray-200 dark:border-gray-700 p-2"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <div className="text-sm font-medium text-gray-800 dark:text-gray-100">
                          {step.label}
                        </div>
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {step.detail}
                        </div>
                        {step.status === "blocked" && (
                          <div className="mt-1 text-xs text-orange-700 dark:text-orange-300">
                            Not executed because an earlier prerequisite step
                            failed.
                          </div>
                        )}
                      </div>
                      <span
                        className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${sevenStepStatusColor(step.status)}`}
                      >
                        {step.status}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div className="rounded-md border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
              <div className="px-3 py-2 border-b border-gray-200 dark:border-gray-700 text-sm font-medium text-gray-700 dark:text-gray-200">
                Running Log
              </div>
              <div className="max-h-96 overflow-y-auto p-3 space-y-2">
                {sortedPrepareChecks.length === 0 ? (
                  <div className="text-xs text-gray-500 dark:text-gray-400">
                    {hasActiveDisplayedPrepareRun
                      ? "Preparing provider… waiting for first check entries."
                      : selectedPrepareRunLoading
                        ? "Loading run details..."
                        : "No check log entries yet."}
                  </div>
                ) : (
                  sortedPrepareChecks.map((check) => (
                    <div
                      key={check.id}
                      className="rounded-md border border-gray-200 dark:border-gray-700 p-2"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <div className="text-xs font-medium text-gray-800 dark:text-gray-100">
                          {check.category.replace("_", " ")} · {check.check_key}
                        </div>
                        <div className="flex items-center gap-2">
                          <span
                            className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium ${
                              check.ok
                                ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                                : "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                            }`}
                          >
                            {check.ok ? "OK" : "Fail"}
                          </span>
                          <span className="text-[11px] text-gray-500 dark:text-gray-400">
                            {new Date(check.created_at).toLocaleTimeString()}
                          </span>
                        </div>
                      </div>
                      <div className="mt-1 text-xs text-gray-600 dark:text-gray-300">
                        {check.message}
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>

            <div className="flex justify-end gap-2">
              {(hasActivePrepareRun || isViewingHistoricalPrepareRun) && (
                <button
                  type="button"
                  onClick={() => {
                    if (isViewingHistoricalPrepareRun) {
                      const runID = selectedPrepareRunStatus?.active_run?.id;
                      if (runID) {
                        void loadPrepareRunDetails(runID);
                      }
                      return;
                    }
                    void loadPrepareStatus({ showLoading: true });
                  }}
                  className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-blue-700 dark:text-blue-300 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700 rounded hover:bg-blue-100 dark:hover:bg-blue-900/30"
                >
                  <RefreshCw
                    className={`h-4 w-4 mr-1 ${prepareLoading || selectedPrepareRunLoading ? "animate-spin" : ""}`}
                  />
                  Refresh
                </button>
              )}
              <button
                type="button"
                onClick={handleClosePrepareDrawer}
                className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded hover:bg-gray-200 dark:hover:bg-gray-600"
              >
                Close
              </button>
            </div>
          </div>
        </Drawer>

        <Drawer
          isOpen={showTenantNamespacePrepareDrawer}
          onClose={() => setShowTenantNamespacePrepareDrawer(false)}
          title="Tenant Namespace Provisioning"
          description="Provision per-tenant namespace RBAC and Tekton assets (managed providers only)."
          width="lg"
        >
          <div className="space-y-4">
            {!effectiveTenantNamespaceId ||
            effectiveTenantNamespaceId === NIL_TENANT_ID ? (
              <div className="rounded-md border border-amber-300 bg-amber-50 dark:border-amber-700 dark:bg-amber-900/20 p-3 text-sm text-amber-900 dark:text-amber-200">
                Select a tenant to view or provision its namespace.
              </div>
            ) : (
              <>
                {isSystemAdmin && (
                  <div>
                    <label className="block text-xs font-medium text-gray-600 dark:text-gray-300 mb-1">
                      Tenant
                    </label>
                    <select
                      value={namespaceTenantId}
                      onChange={(event) =>
                        setNamespaceTenantId(event.target.value)
                      }
                      className="w-full md:w-80 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-sm text-gray-900 dark:text-gray-100 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      {tenants.map((tenant) => (
                        <option key={tenant.id} value={tenant.id}>
                          {tenant.name}
                        </option>
                      ))}
                    </select>
                  </div>
                )}
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="text-sm text-gray-700 dark:text-gray-200">
                    Tenant:{" "}
                    <span className="font-semibold">
                      {selectedNamespaceTenant?.name ||
                        effectiveTenantNamespaceId}
                    </span>
                  </div>
                  <div className="flex flex-wrap items-stretch gap-2 md:justify-end">
                    <button
                      type="button"
                      onClick={loadTenantNamespacePrepareStatus}
                      disabled={tenantNamespacePrepareLoading}
                      className="inline-flex min-h-10 w-full sm:w-auto sm:min-w-[12rem] items-center justify-center rounded-md border border-gray-200 px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600 bg-gray-100"
                    >
                      <RefreshCw
                        className={`h-4 w-4 mr-1 ${tenantNamespacePrepareLoading ? "animate-spin" : ""}`}
                      />
                      Refresh
                    </button>
                    <button
                      type="button"
                      onClick={handleProvisionTenantNamespace}
                      disabled={
                        tenantNamespacePrepareActionLoading ||
                        hasActiveTenantNamespacePrepare
                      }
                      className="inline-flex min-h-10 w-full sm:w-auto sm:min-w-[12rem] items-center justify-center rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      {tenantNamespacePrepareActionLoading ||
                      hasActiveTenantNamespacePrepare
                        ? "Provisioning…"
                        : "Prepare"}
                    </button>
                    <button
                      type="button"
                      onClick={handleDeprovisionTenantNamespace}
                      disabled={
                        tenantNamespacePrepareActionLoading ||
                        hasActiveTenantNamespacePrepare
                      }
                      className="inline-flex min-h-10 w-full sm:w-auto sm:min-w-[12rem] items-center justify-center rounded-md border border-red-200 bg-red-100 px-3 py-2 text-sm font-medium text-red-700 transition-colors hover:bg-red-200 disabled:cursor-not-allowed disabled:opacity-50 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200 dark:hover:bg-red-900/50"
                    >
                      {tenantNamespacePrepareActionLoading ||
                      hasActiveTenantNamespacePrepare
                        ? "Working…"
                        : "Deprovision"}
                    </button>
                  </div>
                </div>

                <div className="rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/40 px-3 py-2">
                  <div className="flex items-center justify-between gap-3">
                    <div className="text-xs font-medium text-gray-700 dark:text-gray-300">
                      Live Stream
                    </div>
                    <span
                      className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium ${
                        tenantNamespacePrepareWsConnected
                          ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                          : "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-200"
                      }`}
                    >
                      {tenantNamespacePrepareWsConnected
                        ? "connected"
                        : "polling fallback"}
                    </span>
                  </div>
                </div>

                {tenantNamespacePrepareError && (
                  <div className="rounded-md border border-red-200 dark:border-red-700 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-800 dark:text-red-200">
                    {tenantNamespacePrepareError}
                  </div>
                )}

                {tenantNamespacePrepare?.asset_drift_status === "stale" && (
                  <div className="rounded-md border border-amber-200 dark:border-amber-700 bg-amber-50 dark:bg-amber-900/20 p-3">
                    <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                      <div className="text-sm text-amber-800 dark:text-amber-200">
                        Tenant namespace assets are stale. Reconcile this tenant
                        or trigger stale-only reconcile across this provider.
                      </div>
                      <div className="flex flex-wrap items-stretch gap-2">
                        <button
                          type="button"
                          onClick={handleReconcileSelectedTenantNamespace}
                          disabled={
                            tenantNamespaceReconcileLoading !== null ||
                            hasActiveTenantNamespacePrepare
                          }
                          className="inline-flex min-h-10 w-full sm:w-auto sm:min-w-[12rem] items-center justify-center rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          {tenantNamespaceReconcileLoading === "selected"
                            ? "Reconciling selected…"
                            : "Reconcile Selected"}
                        </button>
                        <button
                          type="button"
                          onClick={handleReconcileStaleTenantNamespaces}
                          disabled={
                            tenantNamespaceReconcileLoading !== null ||
                            hasActiveTenantNamespacePrepare
                          }
                          className="inline-flex min-h-10 w-full sm:w-auto sm:min-w-[12rem] items-center justify-center rounded-md border border-amber-300 bg-amber-200 px-3 py-2 text-sm font-medium text-amber-900 transition-colors hover:bg-amber-300 disabled:cursor-not-allowed disabled:opacity-50 dark:border-amber-700 dark:bg-amber-800 dark:text-amber-100 dark:hover:bg-amber-700"
                        >
                          {tenantNamespaceReconcileLoading === "stale"
                            ? "Reconciling stale…"
                            : "Reconcile Stale"}
                        </button>
                      </div>
                    </div>
                  </div>
                )}

                <div className="grid grid-cols-1 md:grid-cols-5 gap-3">
                  <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                    <div className="text-xs font-medium text-gray-600 dark:text-gray-300">
                      Namespace
                    </div>
                    <div className="mt-1 font-mono text-xs text-gray-900 dark:text-white break-all">
                      {tenantNamespacePrepare?.namespace ||
                        `image-factory-${effectiveTenantNamespaceId.slice(0, 8)}`}
                    </div>
                  </div>
                  <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                    <div className="text-xs font-medium text-gray-600 dark:text-gray-300">
                      Status
                    </div>
                    <div className="mt-1">
                      <span
                        className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                          tenantNamespacePrepare?.status === "succeeded"
                            ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                            : tenantNamespacePrepare?.status === "failed"
                              ? "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                              : tenantNamespacePrepare?.status
                                ? "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
                                : "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-200"
                        }`}
                      >
                        {tenantNamespacePrepare?.status ||
                          "no tenant namespace prepare run yet"}
                      </span>
                    </div>
                  </div>
                  <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                    <div className="text-xs font-medium text-gray-600 dark:text-gray-300">
                      Drift Status
                    </div>
                    <div className="mt-1">
                      <TenantAssetDriftBadge
                        status={tenantNamespacePrepare?.asset_drift_status}
                      />
                    </div>
                  </div>
                  <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                    <div className="text-xs font-medium text-gray-600 dark:text-gray-300">
                      Desired Asset Version
                    </div>
                    <div className="mt-1 group flex items-center gap-2">
                      <div
                        className="font-mono text-xs text-gray-900 dark:text-white break-all"
                        title={tenantNamespacePrepare?.desired_asset_version || ""}
                      >
                        {formatAssetVersion(
                          tenantNamespacePrepare?.desired_asset_version,
                        )}
                      </div>
                      {!!tenantNamespacePrepare?.desired_asset_version && (
                        <button
                          type="button"
                          onClick={() =>
                            void copyToClipboard(
                              tenantNamespacePrepare.desired_asset_version || "",
                              "tenant-asset-desired",
                            )
                          }
                          className="opacity-0 group-hover:opacity-100 transition-opacity inline-flex items-center justify-center h-5 w-5 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
                          title="Copy desired asset version"
                        >
                          {copiedText === "tenant-asset-desired" ? (
                            <Check className="h-3 w-3" />
                          ) : (
                            <Copy className="h-3 w-3" />
                          )}
                        </button>
                      )}
                    </div>
                    <div className="mt-1 text-[11px] text-gray-500 dark:text-gray-400">
                      profile: {providerProfileVersion}
                    </div>
                  </div>
                  <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                    <div className="text-xs font-medium text-gray-600 dark:text-gray-300">
                      Installed Asset Version
                    </div>
                    <div className="mt-1 group flex items-center gap-2">
                      <div
                        className="font-mono text-xs text-gray-900 dark:text-white break-all"
                        title={
                          tenantNamespacePrepare?.installed_asset_version || ""
                        }
                      >
                        {formatAssetVersion(
                          tenantNamespacePrepare?.installed_asset_version,
                        )}
                      </div>
                      {!!tenantNamespacePrepare?.installed_asset_version && (
                        <button
                          type="button"
                          onClick={() =>
                            void copyToClipboard(
                              tenantNamespacePrepare.installed_asset_version ||
                                "",
                              "tenant-asset-installed",
                            )
                          }
                          className="opacity-0 group-hover:opacity-100 transition-opacity inline-flex items-center justify-center h-5 w-5 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
                          title="Copy installed asset version"
                        >
                          {copiedText === "tenant-asset-installed" ? (
                            <Check className="h-3 w-3" />
                          ) : (
                            <Copy className="h-3 w-3" />
                          )}
                        </button>
                      )}
                    </div>
                  </div>
                  <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                    <div className="text-xs font-medium text-gray-600 dark:text-gray-300">
                      Last Updated
                    </div>
                    <div className="mt-1 text-sm text-gray-900 dark:text-white">
                      {tenantNamespacePrepare?.updated_at
                        ? new Date(
                            tenantNamespacePrepare.updated_at,
                          ).toLocaleString()
                        : "never"}
                    </div>
                  </div>
                </div>

                {tenantNamespacePrepare?.error_message && (
                  <div className="rounded-md border border-red-200 dark:border-red-700 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-800 dark:text-red-200">
                    <div className="font-semibold mb-1">Error</div>
                    <div className="text-xs whitespace-pre-wrap">
                      {tenantNamespacePrepare.error_message}
                    </div>
                  </div>
                )}

                <div className="rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/40 p-3">
                  <div className="text-xs font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-300 mb-2">
                    Result Summary
                  </div>
                  <pre className="text-xs text-gray-800 dark:text-gray-200 overflow-x-auto whitespace-pre-wrap">
                    {JSON.stringify(
                      tenantNamespacePrepare?.result_summary || {},
                      null,
                      2,
                    )}
                  </pre>
                </div>

                <div className="rounded-md border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-3">
                  <div className="text-xs font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-300 mb-2">
                    Provisioning Steps
                  </div>
                  <div className="space-y-2">
                    {tenantPrepareSteps.map((step) => (
                      <div
                        key={step.key}
                        className="rounded border border-gray-200 dark:border-gray-700 p-2"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="text-xs font-medium text-gray-700 dark:text-gray-200">
                            {step.label}
                          </div>
                          <span
                            className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium ${
                              step.complete
                                ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                                : hasActiveTenantNamespacePrepare
                                  ? "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
                                  : "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-200"
                            }`}
                          >
                            {step.complete
                              ? "done"
                              : hasActiveTenantNamespacePrepare
                                ? "running"
                                : "pending"}
                          </span>
                        </div>
                        {step.complete && step.value !== undefined && (
                          <pre className="mt-1 text-[11px] text-gray-600 dark:text-gray-300 overflow-x-auto whitespace-pre-wrap">
                            {JSON.stringify(step.value, null, 2)}
                          </pre>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              </>
            )}
          </div>
        </Drawer>

        {kubernetesProviderTypes.has(provider.provider_type) && (
          <div className="bg-white dark:bg-gray-800 shadow rounded-lg lg:col-span-2">
            <div className="px-4 py-5 sm:p-6 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                  Provider Runtime Status
                </h3>
                <div className="flex flex-wrap items-center gap-2">
                  {canManageAdmin &&
                    provider.bootstrap_mode === "image_factory_managed" && (
                    <button
                      type="button"
                      onClick={handleRegenerateRuntimeAuth}
                      disabled={prepareActionLoading || hasActivePrepareRun}
                      className="px-3 py-1.5 text-sm font-medium text-blue-700 dark:text-blue-300 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700 rounded-md hover:bg-blue-100 dark:hover:bg-blue-900/30 disabled:opacity-50"
                    >
                      Regenerate Runtime Auth
                    </button>
                  )}
                  {canManageAdmin && (
                    <button
                      type="button"
                      onClick={handlePrepareProvider}
                      disabled={prepareActionLoading || hasActivePrepareRun}
                      className="px-3 py-1.5 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50"
                    >
                      {prepareActionLoading || hasActivePrepareRun
                        ? "Preparation In Progress"
                        : "Prepare Provider"}
                    </button>
                  )}
                  {(prepareActionLoading || hasActivePrepareRun) && (
                    <button
                      type="button"
                      onClick={() => {
                        setSelectedPrepareRunStatus(null);
                        setSelectedPrepareRunError(null);
                        setShowPrepareDrawer(true);
                      }}
                      className="px-3 py-1.5 text-sm font-medium text-blue-700 dark:text-blue-300 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700 rounded-md hover:bg-blue-100 dark:hover:bg-blue-900/30"
                    >
                      View Progress
                    </button>
                  )}
                  <button
                    type="button"
                    onClick={loadTektonStatus}
                    className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded hover:bg-gray-200 dark:hover:bg-gray-600"
                  >
                    <RefreshCw
                      className={`h-4 w-4 mr-1 ${tektonLoading ? "animate-spin" : ""}`}
                    />
                    Refresh
                  </button>
                </div>
              </div>

              {tektonError && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md p-3 text-sm text-red-800 dark:text-red-200">
                  {tektonError}
                </div>
              )}

              <div className="rounded-md border border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-900/20 p-3">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <div className="text-sm font-semibold text-blue-900 dark:text-blue-100">
                      Onboarding Flow
                    </div>
                    <div className="text-xs text-blue-800 dark:text-blue-200">
                      Single path for provider onboarding. Follow steps in order
                      and use the action bar above.
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {onboardingComplete && (
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
                        Complete
                      </span>
                    )}
                  </div>
                </div>

                <div className="mt-3 grid grid-cols-1 md:grid-cols-2 gap-2">
                  {onboardingSteps.map((step) => (
                    <div
                      key={step.key}
                      className="rounded-md border border-blue-100 dark:border-blue-900 bg-white dark:bg-gray-800 p-2"
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div>
                          <div className="text-xs font-semibold text-gray-800 dark:text-gray-100">
                            {step.label}
                          </div>
                          <div className="text-[11px] text-gray-600 dark:text-gray-300">
                            {step.detail}
                          </div>
                        </div>
                        <span
                          className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium ${onboardingStepClass(step.status)}`}
                        >
                          {step.status}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>

                {prepareError && (
                  <div className="mt-3 rounded-md border border-red-200 dark:border-red-700 bg-red-50 dark:bg-red-900/20 p-2 text-xs text-red-700 dark:text-red-300">
                    {prepareError}
                  </div>
                )}

                <div className="mt-3 grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="rounded-md border border-blue-100 dark:border-blue-900 bg-white dark:bg-gray-800 p-3">
                    <div className="text-xs font-medium text-gray-700 dark:text-gray-200 mb-2">
                      Active Run
                    </div>
                    {prepareStatus?.active_run ? (
                      <div className="space-y-1 text-xs text-gray-600 dark:text-gray-300">
                        <div>
                          Status:{" "}
                          <span
                            className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                              prepareStatus.active_run.status === "succeeded"
                                ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                                : prepareStatus.active_run.status === "failed"
                                  ? "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                                  : "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
                            }`}
                          >
                            {prepareStatus.active_run.status}
                          </span>
                        </div>
                        <div>
                          ID:{" "}
                          <span className="font-mono">
                            {prepareStatus.active_run.id}
                          </span>
                        </div>
                        <div>
                          Updated:{" "}
                          {new Date(
                            prepareStatus.active_run.updated_at,
                          ).toLocaleString()}
                        </div>
                        {prepareStatus.active_run.error_message && (
                          <div className="text-red-700 dark:text-red-300">
                            {prepareStatus.active_run.error_message}
                          </div>
                        )}
                      </div>
                    ) : (
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        {prepareLoading ? "Loading…" : "No active run"}
                      </div>
                    )}
                  </div>

                  <div className="rounded-md border border-blue-100 dark:border-blue-900 bg-white dark:bg-gray-800 p-3">
                    <div className="text-xs font-medium text-gray-700 dark:text-gray-200 mb-2">
                      Recent Runs
                    </div>
                    {prepareRuns.length > 0 ? (
                      <div className="max-h-36 overflow-y-auto space-y-1 text-xs">
                        {prepareRuns.slice(0, 5).map((run) => (
                          <div
                            key={run.id}
                            className="flex items-center justify-between gap-2 border-b border-gray-100 dark:border-gray-700 pb-1"
                          >
                            <button
                              type="button"
                              onClick={() => void handleViewPrepareRun(run.id)}
                              className="font-mono text-left text-blue-700 dark:text-blue-300 hover:underline"
                              title="Open run details"
                            >
                              {run.id.slice(0, 8)}
                            </button>
                            <span className="text-gray-500 dark:text-gray-400">
                              {new Date(run.created_at).toLocaleString()}
                            </span>
                            <span
                              className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                                run.status === "succeeded"
                                  ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                                  : run.status === "failed"
                                    ? "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                                    : "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
                              }`}
                            >
                              {run.status}
                            </span>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        {prepareLoading ? "Loading…" : "No prepare runs yet"}
                      </div>
                    )}
                  </div>
                </div>
              </div>

              <div
                id="provider-preparation-section"
                className="rounded-md border border-gray-200 dark:border-gray-700 p-3 bg-white dark:bg-gray-800"
              >
                <div className="mb-3">
                  <div className="text-sm font-medium text-gray-800 dark:text-gray-100">
                    Preparation Check Details
                  </div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">
                    Diagnostic timeline from the latest prepare run.
                  </div>
                </div>

                {prepareStatus?.checks && prepareStatus.checks.length > 0 ? (
                  <details
                    id="prepare-checks-section"
                    className="mt-3 rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/40"
                    open={!latestPrepareRunSucceeded || latestPrepareRunRunning}
                  >
                    <summary className="cursor-pointer list-none px-3 py-2">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="text-xs font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-300">
                          Preparation Checks
                        </div>
                        <div className="flex items-center gap-2">
                          {latestPrepareRunSucceeded && (
                            <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
                              all checks passed
                            </span>
                          )}
                          {latestPrepareRun?.updated_at && (
                            <span className="text-[11px] text-gray-500 dark:text-gray-400">
                              {new Date(
                                latestPrepareRun.updated_at,
                              ).toLocaleString()}
                            </span>
                          )}
                        </div>
                      </div>
                      {latestPrepareRunSucceeded && (
                        <div className="mt-2 flex flex-wrap items-center gap-2 text-[11px]">
                          <span className="text-gray-600 dark:text-gray-300">
                            Latest run:
                          </span>
                          <button
                            type="button"
                            onClick={(e) => {
                              e.preventDefault();
                              if (latestPrepareRun?.id) {
                                void handleViewPrepareRun(latestPrepareRun.id);
                              }
                            }}
                            className="font-mono text-blue-700 dark:text-blue-300 hover:underline"
                          >
                            {latestPrepareRun?.id?.slice(0, 8)}
                          </button>
                        </div>
                      )}
                    </summary>
                    <div className="border-t border-gray-200 dark:border-gray-700 p-3 space-y-3 max-h-96 overflow-y-auto">
                      {(
                        [
                          "connectivity",
                          "permission_audit",
                          "bootstrap",
                          "readiness",
                        ] as const
                      ).map((category) => {
                        const checks =
                          prepareStatus.checks?.filter(
                            (c) => c.category === category,
                          ) || [];
                        if (checks.length === 0) return null;
                        return (
                          <div
                            key={category}
                            className="rounded-md border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-2"
                          >
                            <div className="text-[11px] font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wide mb-2">
                              {category.replace("_", " ")}
                            </div>
                            <div className="space-y-2">
                              {checks.slice(-5).map((check) => {
                                const remediationCommands = Array.isArray(
                                  check.details?.remediation_commands,
                                )
                                  ? (check.details
                                      ?.remediation_commands as string[])
                                  : [];
                                const remediation =
                                  typeof check.details?.remediation === "string"
                                    ? check.details.remediation
                                    : null;
                                return (
                                  <div
                                    key={check.id}
                                    className="rounded border border-gray-100 dark:border-gray-700 p-2"
                                  >
                                    <div className="flex items-start justify-between gap-2">
                                      <div className="min-w-0">
                                        <div className="text-xs font-medium text-gray-700 dark:text-gray-200">
                                          {check.check_key}
                                        </div>
                                        <div className="text-xs text-gray-500 dark:text-gray-400">
                                          {check.message}
                                        </div>
                                      </div>
                                      <span
                                        className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium ${check.ok ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" : "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"}`}
                                      >
                                        {check.ok ? "OK" : "Fail"}
                                      </span>
                                    </div>

                                    {!check.ok && remediation && (
                                      <div className="mt-2 text-[11px] text-amber-700 dark:text-amber-300">
                                        Remediation: {remediation}
                                      </div>
                                    )}

                                    {!check.ok &&
                                      remediationCommands.length > 0 && (
                                        <div className="mt-2 rounded bg-gray-100 dark:bg-gray-900 border border-gray-200 dark:border-gray-700 p-2">
                                          <div className="flex items-center justify-between mb-1">
                                            <span className="text-[11px] font-medium text-gray-700 dark:text-gray-300">
                                              Suggested Commands
                                            </span>
                                            <button
                                              type="button"
                                              onClick={() =>
                                                copyToClipboard(
                                                  remediationCommands.join(
                                                    "\n",
                                                  ),
                                                  `rem-${check.id}`,
                                                )
                                              }
                                              className="inline-flex items-center gap-1 text-[11px] px-2 py-1 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700"
                                            >
                                              <Copy className="h-3 w-3" />
                                              {copiedText === `rem-${check.id}`
                                                ? "Copied"
                                                : "Copy"}
                                            </button>
                                          </div>
                                          <pre className="text-[11px] text-gray-700 dark:text-gray-300 overflow-x-auto whitespace-pre-wrap">
                                            {remediationCommands.join("\n")}
                                          </pre>
                                        </div>
                                      )}
                                  </div>
                                );
                              })}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </details>
                ) : (
                  <div className="mt-3 rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/40 p-3">
                    <div className="text-xs text-gray-600 dark:text-gray-300">
                      {prepareLoading
                        ? "Loading prepare checks..."
                        : hasPrepareRunHistory
                          ? "No checks available for the latest run yet. Open run details for full status."
                          : "No prepare run history yet. Use Prepare Provider from the action bar to generate diagnostics."}
                    </div>
                    {hasPrepareRunHistory && latestPrepareRun?.id && (
                      <button
                        type="button"
                        onClick={() => void handleViewPrepareRun(latestPrepareRun.id)}
                        className="mt-2 inline-flex items-center px-2 py-1 text-xs font-medium text-blue-700 dark:text-blue-300 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700 rounded hover:bg-blue-100 dark:hover:bg-blue-900/30"
                      >
                        View Latest Run
                      </button>
                    )}
                  </div>
                )}
              </div>

              {isManagedBootstrapProvider &&
                isSystemAdmin &&
                effectiveTenantNamespaceId &&
                effectiveTenantNamespaceId !== NIL_TENANT_ID && (
                  <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3 bg-white dark:bg-gray-800">
                    <div>
                      <div className="text-sm font-medium text-gray-800 dark:text-gray-100">
                        Tenant Namespace Provisioning
                      </div>
                      <div className="text-xs text-gray-500 dark:text-gray-400">
                        Managed mode uses per-tenant namespaces. Provision RBAC
                        and Tekton assets for the selected tenant before
                        scheduling builds.
                      </div>
                      <div className="mt-2">
                        <label className="block text-xs font-medium text-gray-600 dark:text-gray-300 mb-1">
                          Tenant
                        </label>
                        <select
                          value={namespaceTenantId}
                          onChange={(event) =>
                            setNamespaceTenantId(event.target.value)
                          }
                          className="w-full md:w-80 rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-sm text-gray-900 dark:text-gray-100 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                        >
                          {tenants.map((tenant) => (
                            <option key={tenant.id} value={tenant.id}>
                              {tenant.name}
                            </option>
                          ))}
                        </select>
                      </div>
                    </div>

                    <div className="mt-3 border-t border-gray-200 dark:border-gray-700 pt-3">
                      <div className="mb-2 text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                        Actions
                      </div>
                      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-4">
                        <button
                          type="button"
                          onClick={loadTenantNamespacePrepareStatus}
                          disabled={tenantNamespacePrepareLoading}
                          className="inline-flex min-h-10 w-full items-center justify-center rounded-md border border-gray-200 bg-gray-100 px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600"
                        >
                          <RefreshCw
                            className={`h-4 w-4 mr-1 ${tenantNamespacePrepareLoading ? "animate-spin" : ""}`}
                          />
                          Refresh
                        </button>
                        <button
                          type="button"
                          onClick={() => setShowTenantNamespacePrepareDrawer(true)}
                          className="inline-flex min-h-10 w-full items-center justify-center rounded-md border border-blue-200 bg-blue-50 px-3 py-2 text-sm font-medium text-blue-700 transition-colors hover:bg-blue-100 dark:border-blue-700 dark:bg-blue-900/20 dark:text-blue-300 dark:hover:bg-blue-900/30"
                        >
                          View Details
                        </button>
                        {canManageAdmin && (
                          <button
                            type="button"
                            onClick={handleProvisionTenantNamespace}
                            disabled={
                              tenantNamespacePrepareActionLoading ||
                              hasActiveTenantNamespacePrepare
                            }
                            className="inline-flex min-h-10 w-full items-center justify-center rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
                          >
                            {tenantNamespacePrepareActionLoading ||
                            hasActiveTenantNamespacePrepare
                              ? "Provisioning…"
                              : "Provision Tenant Namespace"}
                          </button>
                        )}
                        {canManageAdmin && (
                          <button
                            type="button"
                            onClick={handleDeprovisionTenantNamespace}
                            disabled={
                              tenantNamespacePrepareActionLoading ||
                              hasActiveTenantNamespacePrepare
                            }
                            className="inline-flex min-h-10 w-full items-center justify-center rounded-md border border-red-200 bg-red-100 px-3 py-2 text-sm font-medium text-red-700 transition-colors hover:bg-red-200 disabled:cursor-not-allowed disabled:opacity-50 dark:border-red-800 dark:bg-red-900/30 dark:text-red-200 dark:hover:bg-red-900/50"
                          >
                            {tenantNamespacePrepareActionLoading ||
                            hasActiveTenantNamespacePrepare
                              ? "Working…"
                              : "Deprovision Tenant Namespace"}
                          </button>
                        )}
                      </div>
                    </div>

                    {tenantNamespacePrepareError && (
                      <div className="mt-3 rounded-md border border-red-200 dark:border-red-700 bg-red-50 dark:bg-red-900/20 p-2 text-xs text-red-700 dark:text-red-300">
                        {tenantNamespacePrepareError}
                      </div>
                    )}

                    {tenantNamespacePrepare?.asset_drift_status === "stale" && (
                      <div className="mt-3 rounded-md border border-amber-200 dark:border-amber-700 bg-amber-50 dark:bg-amber-900/20 p-3">
                        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                          <div className="text-sm text-amber-800 dark:text-amber-200">
                            Tenant namespace assets are stale. Reconcile this
                            tenant or trigger stale-only reconcile for this
                            provider.
                          </div>
                          <div className="flex flex-wrap items-stretch gap-2">
                            {canManageAdmin && (
                              <button
                                type="button"
                                onClick={handleReconcileSelectedTenantNamespace}
                                disabled={
                                  tenantNamespaceReconcileLoading !== null ||
                                  hasActiveTenantNamespacePrepare
                                }
                                className="inline-flex min-h-10 w-full sm:w-auto sm:min-w-[12rem] items-center justify-center rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
                              >
                                {tenantNamespaceReconcileLoading === "selected"
                                  ? "Reconciling selected…"
                                  : "Reconcile Selected"}
                              </button>
                            )}
                            {canManageAdmin && (
                              <button
                                type="button"
                                onClick={handleReconcileStaleTenantNamespaces}
                                disabled={
                                  tenantNamespaceReconcileLoading !== null ||
                                  hasActiveTenantNamespacePrepare
                                }
                                className="inline-flex min-h-10 w-full sm:w-auto sm:min-w-[12rem] items-center justify-center rounded-md border border-amber-300 bg-amber-200 px-3 py-2 text-sm font-medium text-amber-900 transition-colors hover:bg-amber-300 disabled:cursor-not-allowed disabled:opacity-50 dark:border-amber-700 dark:bg-amber-800 dark:text-amber-100 dark:hover:bg-amber-700"
                              >
                                {tenantNamespaceReconcileLoading === "stale"
                                  ? "Reconciling stale…"
                                  : "Reconcile Stale"}
                              </button>
                            )}
                          </div>
                        </div>
                      </div>
                    )}

                    <div className="mt-3 grid grid-cols-1 md:grid-cols-5 gap-3 text-sm">
                      <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                        <div className="text-xs font-medium text-gray-600 dark:text-gray-300">
                          Tenant
                        </div>
                        <div className="mt-1 text-gray-900 dark:text-white">
                          {selectedNamespaceTenant?.name ||
                            effectiveTenantNamespaceId}
                        </div>
                      </div>
                      <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                        <div className="text-xs font-medium text-gray-600 dark:text-gray-300">
                          Namespace
                        </div>
                        <div className="mt-1 font-mono text-xs text-gray-900 dark:text-white break-all">
                          {tenantNamespacePrepare?.namespace ||
                            `image-factory-${effectiveTenantNamespaceId.slice(0, 8)}`}
                        </div>
                      </div>
                      <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                        <div className="text-xs font-medium text-gray-600 dark:text-gray-300">
                          Status
                        </div>
                        <div className="mt-1">
                          <span
                            className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                              tenantNamespacePrepare?.status === "succeeded"
                                ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                                : tenantNamespacePrepare?.status === "failed"
                                  ? "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
                                  : tenantNamespacePrepare?.status
                                    ? "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
                                    : "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-200"
                            }`}
                          >
                            {tenantNamespacePrepare?.status ||
                              "no tenant namespace prepare run yet"}
                          </span>
                        </div>
                      </div>
                      <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                        <div className="text-xs font-medium text-gray-600 dark:text-gray-300">
                          Drift
                        </div>
                        <div className="mt-1">
                          <TenantAssetDriftBadge
                            status={tenantNamespacePrepare?.asset_drift_status}
                          />
                        </div>
                      </div>
                      <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                        <div className="text-xs font-medium text-gray-600 dark:text-gray-300">
                          Desired / Installed
                        </div>
                        <div className="mt-1 text-[11px] text-gray-800 dark:text-gray-200">
                          <div className="group flex items-center gap-2">
                            <div
                              className="font-mono break-all"
                              title={
                                tenantNamespacePrepare?.desired_asset_version || ""
                              }
                            >
                              d:{" "}
                              {formatAssetVersion(
                                tenantNamespacePrepare?.desired_asset_version,
                              )}
                            </div>
                            {!!tenantNamespacePrepare?.desired_asset_version && (
                              <button
                                type="button"
                                onClick={() =>
                                  void copyToClipboard(
                                    tenantNamespacePrepare.desired_asset_version ||
                                      "",
                                    "tenant-asset-desired-inline",
                                  )
                                }
                                className="opacity-0 group-hover:opacity-100 transition-opacity inline-flex items-center justify-center h-4 w-4 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
                                title="Copy desired asset version"
                              >
                                {copiedText === "tenant-asset-desired-inline" ? (
                                  <Check className="h-3 w-3" />
                                ) : (
                                  <Copy className="h-3 w-3" />
                                )}
                              </button>
                            )}
                          </div>
                          <div className="group flex items-center gap-2">
                            <div
                              className="font-mono break-all"
                              title={
                                tenantNamespacePrepare?.installed_asset_version ||
                                ""
                              }
                            >
                              i:{" "}
                              {formatAssetVersion(
                                tenantNamespacePrepare?.installed_asset_version,
                              )}
                            </div>
                            {!!tenantNamespacePrepare?.installed_asset_version && (
                              <button
                                type="button"
                                onClick={() =>
                                  void copyToClipboard(
                                    tenantNamespacePrepare.installed_asset_version ||
                                      "",
                                    "tenant-asset-installed-inline",
                                  )
                                }
                                className="opacity-0 group-hover:opacity-100 transition-opacity inline-flex items-center justify-center h-4 w-4 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
                                title="Copy installed asset version"
                              >
                                {copiedText === "tenant-asset-installed-inline" ? (
                                  <Check className="h-3 w-3" />
                                ) : (
                                  <Copy className="h-3 w-3" />
                                )}
                              </button>
                            )}
                          </div>
                          <div className="mt-1 text-gray-500 dark:text-gray-400">
                            profile: {providerProfileVersion}
                          </div>
                        </div>
                      </div>
                    </div>

                    <div className="mt-3 rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/40 p-3">
                      <div className="text-xs font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-300 mb-2">
                        Provisioning Model
                      </div>
                      <div className="space-y-2 text-xs text-gray-700 dark:text-gray-300">
                        <div>
                          Namespace naming rule:
                          <span className="ml-1 font-mono bg-white dark:bg-gray-800 px-1.5 py-0.5 rounded border border-gray-200 dark:border-gray-700">
                            image-factory-&lt;tenant-id-first-8&gt;
                          </span>
                        </div>
                        <div>
                          Selected tenant namespace:
                          <span className="ml-1 font-mono bg-white dark:bg-gray-800 px-1.5 py-0.5 rounded border border-gray-200 dark:border-gray-700 break-all">
                            {buildTenantNamespaceName(
                              effectiveTenantNamespaceId,
                            )}
                          </span>
                        </div>
                        <div>
                          Tenant label applied:
                          <span className="ml-1 font-mono bg-white dark:bg-gray-800 px-1.5 py-0.5 rounded border border-gray-200 dark:border-gray-700 break-all">
                            {tenantNamespaceTenantIDLabelKey}=
                            {effectiveTenantNamespaceId}
                          </span>
                        </div>
                        <ol className="list-decimal ml-4 space-y-1">
                          <li>Create or ensure tenant namespace exists.</li>
                          <li>
                            Apply runtime Role/RoleBinding in that namespace.
                          </li>
                          <li>
                            Apply tenant-scoped Tekton tasks/pipelines in that
                            namespace.
                          </li>
                        </ol>
                      </div>
                    </div>
                  </div>
                )}

              <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3 bg-gray-50 dark:bg-gray-900/40">
                <div className="text-xs font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-300 mb-2">
                  Scheduling Gate
                </div>
                <div className="flex flex-wrap items-center gap-3 text-sm">
                  <div className="flex items-center gap-2">
                    <span className="text-gray-600 dark:text-gray-300">
                      Provider status
                    </span>
                    <span
                      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${statusColors[provider.status]}`}
                    >
                      {provider.status}
                    </span>
                  </div>
                  <div className="hidden md:block h-4 w-px bg-gray-300 dark:bg-gray-600" />
                  <div className="flex items-center gap-2">
                    <span className="text-gray-600 dark:text-gray-300">
                      Readiness
                    </span>
                    <span
                      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${readinessStatus === "ready" ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" : "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"}`}
                    >
                      {readinessStatus}
                    </span>
                  </div>
                  <div className="hidden md:block h-4 w-px bg-gray-300 dark:bg-gray-600" />
                  <div className="flex items-center gap-2">
                    <span className="text-gray-600 dark:text-gray-300">
                      Schedulable
                    </span>
                    <span
                      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold ${isProviderSchedulable ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" : "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"}`}
                    >
                      {isProviderSchedulable ? "Yes" : "No"}
                    </span>
                  </div>
                </div>
                {!isProviderSchedulable && (
                  <div className="mt-3 rounded-md border border-amber-300 bg-amber-50 dark:border-amber-700 dark:bg-amber-900/20 p-2 text-xs text-amber-900 dark:text-amber-200">
                    <div>
                      <span className="font-semibold">Reason:</span>{" "}
                      {schedulableReason ||
                        "Provider is blocked by readiness, status, or policy gates."}
                    </div>
                    {blockedBy.length > 0 && (
                      <div className="mt-2 flex flex-wrap gap-1.5">
                        {blockedBy.map((gate) => (
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
                      onClick={scrollToPrepareChecks}
                      className="mt-2 text-xs font-semibold text-amber-800 hover:text-amber-900 dark:text-amber-300 dark:hover:text-amber-200 underline underline-offset-2"
                    >
                      View prepare checks
                    </button>
                  </div>
                )}
              </div>

              {isManagedBootstrapProvider ? (
                hasPrepareRunHistory ? (
                  <details
                    className="rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/40"
                    open={latestPrepareRun?.status !== "succeeded"}
                  >
                    <summary className="cursor-pointer list-none px-3 py-2">
                      <div className="flex items-center justify-between gap-3">
                        <div>
                          <div className="text-sm font-medium text-gray-800 dark:text-gray-100">
                            Advanced Installer Actions
                          </div>
                          <div className="text-xs text-gray-500 dark:text-gray-400">
                            Use only for recovery or explicit
                            re-install/upgrade.
                          </div>
                        </div>
                        {latestPrepareRun?.status === "succeeded" && (
                          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
                            Prepared
                          </span>
                        )}
                      </div>
                    </summary>
                    <div className="border-t border-gray-200 dark:border-gray-700 p-3">
                      <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Install Mode
                          </label>
                          <select
                            value={tektonInstallMode}
                            onChange={(e) =>
                              setTektonInstallMode(
                                e.target.value as TektonInstallMode,
                              )
                            }
                            className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white"
                          >
                            <option value="image_factory_installer">
                              image_factory_installer
                            </option>
                            <option value="gitops">gitops</option>
                          </select>
                        </div>
                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Asset Version
                          </label>
                          <input
                            value={tektonAssetVersion}
                            onChange={(e) =>
                              setTektonAssetVersion(e.target.value)
                            }
                            placeholder="v1"
                            className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-3 py-2 text-sm text-gray-900 dark:text-white"
                          />
                        </div>
                        <div className="flex items-end gap-2">
                          {canManageAdmin && (
                            <button
                              type="button"
                              onClick={() => handleTektonAction("install")}
                              disabled={
                                tektonActionLoading !== null ||
                                hasActivePrepareRun
                              }
                              className="px-3 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50"
                            >
                              {tektonActionLoading === "install"
                                ? "Starting…"
                                : "Reinstall Assets"}
                            </button>
                          )}
                          {canManageAdmin && (
                            <button
                              type="button"
                              onClick={() => handleTektonAction("upgrade")}
                              disabled={
                                tektonActionLoading !== null ||
                                hasActivePrepareRun
                              }
                              className="px-3 py-2 text-sm font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700 disabled:opacity-50"
                            >
                              {tektonActionLoading === "upgrade"
                                ? "Starting…"
                                : "Upgrade Assets"}
                            </button>
                          )}
                          {canManageAdmin && (
                            <button
                              type="button"
                              onClick={() => handleTektonAction("validate")}
                              disabled={
                                tektonActionLoading !== null ||
                                hasActivePrepareRun
                              }
                              className="px-3 py-2 text-sm font-medium text-white bg-emerald-600 rounded-md hover:bg-emerald-700 disabled:opacity-50"
                            >
                              {tektonActionLoading === "validate"
                                ? "Starting…"
                                : "Validate Only"}
                            </button>
                          )}
                        </div>
                      </div>
                    </div>
                  </details>
                ) : (
                  <div className="rounded-md border border-blue-200 dark:border-blue-700 bg-blue-50 dark:bg-blue-900/20 p-3 text-sm text-blue-800 dark:text-blue-200">
                    Advanced installer actions appear after the first{" "}
                    <span className="font-semibold">Prepare Provider</span> run.
                  </div>
                )
              ) : (
                <div className="rounded-md border border-amber-300 bg-amber-50 dark:border-amber-700 dark:bg-amber-900/20 p-3 text-sm text-amber-900 dark:text-amber-200">
                  This provider is in{" "}
                  <span className="font-semibold">self-managed</span> mode.
                  Tekton install/upgrade/validate actions are intentionally
                  disabled. Run cluster bootstrap and Tekton asset management
                  outside Image Factory, then use{" "}
                  <span className="font-semibold">Prepare Provider</span> for
                  connectivity/readiness checks.
                </div>
              )}

              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                  <div className="text-sm font-medium text-gray-700 dark:text-gray-200 mb-2">
                    Readiness
                  </div>
                  <div className="text-sm text-gray-600 dark:text-gray-300 space-y-1">
                    <div>
                      Status:{" "}
                      <span
                        className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${readinessStatus === "ready" ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" : "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"}`}
                      >
                        {readinessStatus}
                      </span>
                      <span className="ml-2 inline-flex align-middle">
                        <HelpTooltip text={readinessTooltipText} sticky />
                      </span>
                    </div>
                    <div>
                      Last checked:{" "}
                      {tektonStatus?.readiness_last_checked
                        ? new Date(
                            tektonStatus.readiness_last_checked,
                          ).toLocaleString()
                        : provider.readiness_last_checked
                          ? new Date(
                              provider.readiness_last_checked,
                            ).toLocaleString()
                          : "never"}
                    </div>
                    {readinessMissingPrereqs.length > 0 && (
                      <div className="pt-1">
                        <div className="font-medium text-red-700 dark:text-red-300">
                          Missing prerequisites
                        </div>
                        <ul className="mt-1 space-y-1">
                          {readinessMissingPrereqs
                            .slice(0, 5)
                            .map((item, index) => (
                              <li
                                key={`${item}-${index}`}
                                className="text-xs text-red-700 dark:text-red-300"
                              >
                                {item}
                              </li>
                            ))}
                        </ul>
                      </div>
                    )}
                  </div>
                </div>

                <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                  <div className="text-sm font-medium text-gray-700 dark:text-gray-200 mb-2">
                    Active Job
                  </div>
                  {tektonStatus?.active_job ? (
                    <div className="text-sm text-gray-600 dark:text-gray-300 space-y-1">
                      <div>
                        ID:{" "}
                        <span className="font-mono text-xs">
                          {tektonStatus.active_job.id}
                        </span>
                      </div>
                      <div>Mode: {tektonStatus.active_job.install_mode}</div>
                      <div>
                        Version: {tektonStatus.active_job.asset_version}
                      </div>
                      <div>
                        Status:{" "}
                        <span
                          className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${tektonJobStatusColors[tektonStatus.active_job.status] || "bg-gray-100 text-gray-800"}`}
                        >
                          {tektonStatus.active_job.status}
                        </span>
                      </div>
                      {tektonStatus.active_job.error_message && (
                        <div className="text-xs text-red-700 dark:text-red-300">
                          {tektonStatus.active_job.error_message}
                        </div>
                      )}
                    </div>
                  ) : (
                    <div className="text-sm text-gray-500 dark:text-gray-400">
                      No active installer job
                    </div>
                  )}
                </div>

                <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                  <div className="text-sm font-medium text-gray-700 dark:text-gray-200 mb-2">
                    Progress
                  </div>
                  {tektonStatus?.active_job ? (
                    <div className="space-y-2">
                      {tektonProgressSteps.map((step) => (
                        <div
                          key={step.key}
                          className="flex items-center justify-between text-sm"
                        >
                          <span className="text-gray-700 dark:text-gray-300">
                            {step.label}
                          </span>
                          <span
                            className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${step.done ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" : "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-200"}`}
                          >
                            {step.done ? "Done" : "Pending"}
                          </span>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="text-sm text-gray-500 dark:text-gray-400">
                      No active installer job
                    </div>
                  )}
                </div>
              </div>

              <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                <div className="text-sm font-medium text-gray-700 dark:text-gray-200 mb-2">
                  Job Events
                </div>
                {tektonStatus?.active_job_events &&
                tektonStatus.active_job_events.length > 0 ? (
                  <div className="max-h-56 overflow-y-auto space-y-2">
                    {tektonStatus.active_job_events.map((event) => (
                      <div
                        key={event.id}
                        className="rounded border border-gray-100 dark:border-gray-700 p-2"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="text-xs font-medium text-gray-800 dark:text-gray-100">
                            {event.event_type}
                          </div>
                          <div className="text-xs text-gray-500 dark:text-gray-400">
                            {new Date(event.created_at).toLocaleString()}
                          </div>
                        </div>
                        <div className="text-sm text-gray-700 dark:text-gray-300 mt-1">
                          {event.message}
                        </div>
                        {event.details &&
                          Object.keys(event.details).length > 0 && (
                            <pre className="mt-2 text-xs bg-gray-50 dark:bg-gray-900 p-2 rounded overflow-x-auto text-gray-700 dark:text-gray-300">
                              {JSON.stringify(event.details, null, 2)}
                            </pre>
                          )}
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-sm text-gray-500 dark:text-gray-400">
                    No active job events
                  </div>
                )}
              </div>

              <div className="rounded-md border border-gray-200 dark:border-gray-700 p-3">
                <div className="text-sm font-medium text-gray-700 dark:text-gray-200 mb-2">
                  Recent Jobs
                </div>
                {tektonStatus?.recent_jobs &&
                tektonStatus.recent_jobs.length > 0 ? (
                  <div className="overflow-x-auto">
                    <table className="min-w-full text-sm">
                      <thead>
                        <tr className="text-left text-gray-500 dark:text-gray-400">
                          <th className="py-2 pr-3">Created</th>
                          <th className="py-2 pr-3">Mode</th>
                          <th className="py-2 pr-3">Version</th>
                          <th className="py-2 pr-3">Status</th>
                          <th className="py-2 pr-3">Job ID</th>
                          <th className="py-2 pr-3">Action</th>
                        </tr>
                      </thead>
                      <tbody>
                        {tektonStatus.recent_jobs.slice(0, 10).map((job) => (
                          <tr
                            key={job.id}
                            className="border-t border-gray-100 dark:border-gray-700"
                          >
                            <td className="py-2 pr-3 text-gray-700 dark:text-gray-300">
                              {new Date(job.created_at).toLocaleString()}
                            </td>
                            <td className="py-2 pr-3 text-gray-700 dark:text-gray-300">
                              {job.install_mode}
                            </td>
                            <td className="py-2 pr-3 text-gray-700 dark:text-gray-300">
                              {job.asset_version}
                            </td>
                            <td className="py-2 pr-3">
                              <span
                                className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${tektonJobStatusColors[job.status] || "bg-gray-100 text-gray-800"}`}
                              >
                                {job.status}
                              </span>
                            </td>
                            <td className="py-2 pr-3 text-xs font-mono text-gray-600 dark:text-gray-400">
                              {job.id}
                            </td>
                            <td className="py-2 pr-3">
                              {canManageAdmin && job.status === "failed" ? (
                                <button
                                  type="button"
                                  onClick={() => handleRetryTektonJob(job.id)}
                                  disabled={
                                    tektonActionLoading !== null ||
                                    hasActiveInstallerJob
                                  }
                                  className="px-2 py-1 text-xs font-medium text-white bg-amber-600 rounded hover:bg-amber-700 disabled:opacity-50"
                                >
                                  {tektonActionLoading === "retry"
                                    ? "Starting…"
                                    : "Retry"}
                                </button>
                              ) : (
                                <span className="text-xs text-gray-400 dark:text-gray-500">
                                  -
                                </span>
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <div className="text-sm text-gray-500 dark:text-gray-400">
                    No recent jobs
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Configuration */}
        <div className="bg-white dark:bg-gray-800 shadow rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
              Configuration
            </h3>
            {configEntries.length > 0 ||
            (visibleAuthConfig &&
              typeof visibleAuthConfig === "object" &&
              Object.keys(visibleAuthConfig).length > 0) ? (
              <dl className="space-y-4">
                {visibleAuthConfig &&
                  typeof visibleAuthConfig === "object" &&
                  Object.entries(visibleAuthConfig).length > 0 && (
                    <div className="rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/40 p-3">
                      <dt className="text-sm font-medium text-gray-700 dark:text-gray-200 mb-2">
                        {visibleAuthLabel}
                      </dt>
                      <div className="space-y-2">
                        {Object.entries(visibleAuthConfig).map(
                          ([authKey, authValue]) => {
                            const fieldPath = `${isManagedBootstrapProvider ? "bootstrap_auth" : "runtime_auth"}.${authKey}`;
                            const isSensitive = isSensitiveKey(authKey);
                            const isVisible = visibleSensitiveFields[fieldPath];
                            return (
                              <div key={fieldPath}>
                                <div className="text-sm font-medium text-gray-500 dark:text-gray-400 capitalize flex items-center justify-between">
                                  <span>
                                    {authKey
                                      .replace(/([A-Z])/g, " $1")
                                      .replace(/^./, (str) =>
                                        str.toUpperCase(),
                                      )}
                                  </span>
                                  {isSensitive && (
                                    <button
                                      onClick={() =>
                                        toggleSensitiveField(fieldPath)
                                      }
                                      className="text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 p-1 rounded"
                                      title={
                                        isVisible
                                          ? "Hide sensitive data"
                                          : "Show sensitive data"
                                      }
                                    >
                                      {isVisible ? (
                                        <EyeOff className="h-4 w-4" />
                                      ) : (
                                        <Eye className="h-4 w-4" />
                                      )}
                                    </button>
                                  )}
                                </div>
                                <div className="mt-1 text-sm text-gray-900 dark:text-white font-mono bg-white dark:bg-gray-800 p-2 rounded break-all">
                                  {isSensitive && !isVisible
                                    ? "••••••••••••••••••••••••••••••••"
                                    : typeof authValue === "object"
                                      ? JSON.stringify(authValue, null, 2)
                                      : String(authValue)}
                                </div>
                              </div>
                            );
                          },
                        )}
                      </div>
                    </div>
                  )}
                {configEntries.map(([key, value]) => (
                  <div key={key}>
                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400 capitalize flex items-center justify-between">
                      <span>
                        {key
                          .replace(/([A-Z])/g, " $1")
                          .replace(/^./, (str) => str.toUpperCase())}
                      </span>
                      {isSensitiveKey(key) && (
                        <button
                          onClick={() => toggleSensitiveField(key)}
                          className="text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 p-1 rounded"
                          title={
                            visibleSensitiveFields[key]
                              ? "Hide sensitive data"
                              : "Show sensitive data"
                          }
                        >
                          {visibleSensitiveFields[key] ? (
                            <EyeOff className="h-4 w-4" />
                          ) : (
                            <Eye className="h-4 w-4" />
                          )}
                        </button>
                      )}
                    </dt>
                    <dd className="mt-1 text-sm text-gray-900 dark:text-white font-mono bg-gray-50 dark:bg-gray-700 p-2 rounded break-all">
                      {isSensitiveKey(key) && !visibleSensitiveFields[key]
                        ? "••••••••••••••••••••••••••••••••"
                        : typeof value === "object"
                          ? JSON.stringify(value, null, 2)
                          : String(value)}
                    </dd>
                  </div>
                ))}
              </dl>
            ) : (
              <p className="text-sm text-gray-500 dark:text-gray-400">
                No configuration details available
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default AdminInfrastructureProviderDetailPage;
