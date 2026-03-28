/**
 * Shared constants, helpers, and YAML templates for AdminInfrastructureProviderDetailPage.
 * Extracted to keep the main page focused on orchestration and state management.
 */

import { Server, Zap } from "lucide-react";
import React from "react";

// ---------------------------------------------------------------------------
// Status color maps
// ---------------------------------------------------------------------------

export const statusColors: Record<string, string> = {
    online: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
    offline: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
    maintenance:
        "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
    pending: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
};

export const tektonJobStatusColors: Record<string, string> = {
    pending: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
    running:
        "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
    succeeded:
        "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
    failed: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
    cancelled: "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200",
};

// ---------------------------------------------------------------------------
// Provider type constants
// ---------------------------------------------------------------------------

export const providerTypeIcons: Record<string, React.ReactNode> = {
    kubernetes: <Zap className="h-5 w-5" />,
    "aws-eks": <Server className="h-5 w-5" />,
    "gcp-gke": <Server className="h-5 w-5" />,
    "azure-aks": <Server className="h-5 w-5" />,
    "oci-oke": <Server className="h-5 w-5" />,
    "vmware-vks": <Server className="h-5 w-5" />,
    openshift: <Server className="h-5 w-5" />,
    rancher: <Server className="h-5 w-5" />,
    build_nodes: <Server className="h-5 w-5" />,
};

export const kubernetesProviderTypes = new Set([
    "kubernetes",
    "aws-eks",
    "gcp-gke",
    "azure-aks",
    "oci-oke",
    "vmware-vks",
    "openshift",
    "rancher",
]);

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

export const isPrepareRunActiveStatus = (status?: string | null): boolean =>
    status === "pending" || status === "running";

export const formatAssetVersion = (version?: string | null): string => {
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

export const isLikelyConnectivityIssue = (value: string): boolean => {
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

export const guidanceForReadinessItem = (item: string): string => {
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
    if (
        normalized.includes("access denied") ||
        normalized.includes("forbidden")
    ) {
        return "Expand runtime service account RBAC for namespaces, Tekton resources, pods, and secrets.";
    }
    if (normalized.includes("cluster_capacity")) {
        return "Cluster is reachable but currently unschedulable due to capacity.";
    }
    return "Run Prepare Provider and fix the failing prerequisite shown in the readiness checklist.";
};

export const blockedByLabel = (key: string): string => {
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

export const buildTenantNamespaceName = (tenantId: string): string => {
    const normalized = (tenantId || "").trim();
    if (!normalized) return "";
    return `image-factory-${normalized.slice(0, 8)}`;
};

export const tenantNamespaceTenantIDLabelKey = "imagefactory.io/tenant-id";

// ---------------------------------------------------------------------------
// YAML / shell template strings
// ---------------------------------------------------------------------------

export const bootstrapNamespaceTemplate = `apiVersion: v1
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

export const runtimeNamespaceTemplate = `apiVersion: v1
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

export const bootstrapClusterRoleTemplate = `apiVersion: rbac.authorization.k8s.io/v1
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

export const runtimeRoleTemplate = `apiVersion: rbac.authorization.k8s.io/v1
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

export const serviceAccountTokenTemplate = `# Kubernetes v1.24+ typically does not auto-create long-lived service account token Secrets.
# Use TokenRequest via kubectl (recommended):
#
# 1 year = 8760h (cluster policies may cap max duration)
kubectl -n imagefactory-system create token image-factory-bootstrap-sa --duration=8760h`;

export const runtimeServiceAccountTokenTemplate = `# Kubernetes v1.24+ typically does not auto-create long-lived service account token Secrets.
# Use TokenRequest via kubectl (recommended):
#
# 1 year = 8760h (cluster policies may cap max duration)
kubectl -n imagefactory-system create token image-factory-runtime-sa --duration=8760h`;
