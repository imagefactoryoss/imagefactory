import { HelpCircle } from "lucide-react";
import React, { useState } from "react";
import { CommonKubernetesFields } from "./CommonKubernetesFields";
import { TabbedForm } from "./TabbedForm";
import { CopyableCodeBlock, TooltipDrawer } from "./TooltipDrawer";
import { ProviderConfigFormProps, ProviderFormComponent } from "./types";

const StandardKubernetesFormComponent: React.FC<ProviderConfigFormProps> = ({
  formData,
  setFormData,
}) => {
  const [showTokenDrawer, setShowTokenDrawer] = useState(false);
  const [showAuthMethodDrawer, setShowAuthMethodDrawer] = useState(false);
  const runtimeAuth = (formData.config?.runtime_auth || {}) as Record<
    string,
    any
  >;
  const bootstrapAuth = (formData.config?.bootstrap_auth || {}) as Record<
    string,
    any
  >;
  const isManagedBootstrap =
    formData.bootstrap_mode === "image_factory_managed";
  const resolveNamespace = () => {
    const candidates = [
      formData.target_namespace,
      typeof formData.config?.system_namespace === "string"
        ? formData.config.system_namespace
        : undefined,
      "imagefactory-system",
    ];
    for (const c of candidates) {
      const value = typeof c === "string" ? c.trim() : "";
      if (value) return value;
    }
    return "imagefactory-system";
  };
  const targetNamespace = resolveNamespace();
  const tektonCoreInstallSource =
    typeof formData.config?.tekton_core_install_source === "string"
      ? formData.config.tekton_core_install_source
      : "manifest";
  const tektonCoreManifestURLs =
    typeof formData.config?.tekton_core_manifest_urls === "string"
      ? formData.config.tekton_core_manifest_urls
      : "";
  const activeAuthMethod = isManagedBootstrap
    ? bootstrapAuth.auth_method || "token"
    : runtimeAuth.auth_method || "token";

  const setConfigField = (key: string, value: any) => {
    setFormData({
      ...formData,
      config: {
        ...formData.config,
        [key]: value,
      },
    });
  };

  const setAuthConfig = (
    key: "runtime_auth" | "bootstrap_auth",
    value: Record<string, any>,
  ) => {
    setFormData({
      ...formData,
      config: {
        ...formData.config,
        [key]: value,
      },
    });
  };

  const renderAuthFields = (
    authConfig: Record<string, any>,
    setConfig: (next: Record<string, any>) => void,
    title: string,
    helper: string,
  ) => (
    <div className="rounded-md border border-gray-200 dark:border-gray-700 p-4 bg-gray-50 dark:bg-gray-900/40 space-y-4">
      <div>
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white">
          {title}
        </h3>
        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {helper}
        </p>
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          Authentication Method
        </label>
        <select
          value={authConfig.auth_method || "token"}
          onChange={(e) =>
            setConfig({
              ...authConfig,
              auth_method: e.target.value,
            })
          }
          className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-blue-500 focus:border-blue-500"
          required
        >
          <option value="kubeconfig">Kubeconfig File</option>
          <option value="token">Service Account Token</option>
          <option value="client-cert">Client Certificate</option>
        </select>
      </div>
      {(authConfig.auth_method || "token") === "kubeconfig" && (
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Kubeconfig Path
          </label>
          <input
            type="text"
            value={authConfig.kubeconfig_path || ""}
            onChange={(e) =>
              setConfig({
                ...authConfig,
                kubeconfig_path: e.target.value,
              })
            }
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
            placeholder="/path/to/kubeconfig"
            required
          />
        </div>
      )}
      {(authConfig.auth_method || "token") === "token" && (
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              API Server Endpoint
            </label>
            <input
              type="url"
              value={authConfig.endpoint || ""}
              onChange={(e) =>
                setConfig({
                  ...authConfig,
                  endpoint: e.target.value,
                })
              }
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
              placeholder="https://k8s-cluster.example.com:6443"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Service Account Token
            </label>
            <textarea
              value={authConfig.token || ""}
              onChange={(e) =>
                setConfig({
                  ...authConfig,
                  token: e.target.value,
                })
              }
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
              rows={4}
              placeholder="eyJhbGciOiJSUzI1NiIsImtpZCI6..."
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              CA Certificate
            </label>
            <textarea
              value={authConfig.ca_cert || ""}
              onChange={(e) =>
                setConfig({
                  ...authConfig,
                  ca_cert: e.target.value,
                })
              }
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
              rows={3}
              placeholder="-----BEGIN CERTIFICATE-----..."
              required
            />
          </div>
        </div>
      )}
      {(authConfig.auth_method || "token") === "client-cert" && (
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              API Server Endpoint
            </label>
            <input
              type="url"
              value={authConfig.endpoint || ""}
              onChange={(e) =>
                setConfig({
                  ...authConfig,
                  endpoint: e.target.value,
                })
              }
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
              placeholder="https://k8s-cluster.example.com:6443"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Client Certificate
            </label>
            <textarea
              value={authConfig.client_cert || ""}
              onChange={(e) =>
                setConfig({
                  ...authConfig,
                  client_cert: e.target.value,
                })
              }
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
              rows={4}
              placeholder="-----BEGIN CERTIFICATE-----..."
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Client Key
            </label>
            <textarea
              value={authConfig.client_key || ""}
              onChange={(e) =>
                setConfig({
                  ...authConfig,
                  client_key: e.target.value,
                })
              }
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
              rows={4}
              placeholder="-----BEGIN PRIVATE KEY-----..."
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              CA Certificate
            </label>
            <textarea
              value={authConfig.ca_cert || ""}
              onChange={(e) =>
                setConfig({
                  ...authConfig,
                  ca_cert: e.target.value,
                })
              }
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
              rows={3}
              placeholder="-----BEGIN CERTIFICATE-----..."
              required
            />
          </div>
        </div>
      )}
    </div>
  );

  const tabs = [
    {
      id: "auth",
      label: "Authentication",
      content: (
        <div className="space-y-4">
          <div>
            <div className="flex items-center gap-2 mb-1">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                Cluster Authentication
              </label>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  setShowTokenDrawer(true);
                }}
                className="text-xs text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 underline underline-offset-2"
              >
                Token setup guide
              </button>
              <HelpCircle
                className="h-4 w-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 cursor-pointer"
                onClick={(e) => {
                  e.stopPropagation();
                  setShowAuthMethodDrawer(true);
                }}
              />
            </div>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              {isManagedBootstrap
                ? "Configure bootstrap identity only. Runtime identity is generated automatically during Prepare Provider."
                : "Self-managed mode uses runtime identity only. Bootstrap/install operations are expected to be handled outside Image Factory."}
            </p>
          </div>
          {isManagedBootstrap ? (
            <>
              <div className="rounded-md border border-blue-200 bg-blue-50 px-3 py-2 text-xs text-blue-900 dark:border-blue-700 dark:bg-blue-900/20 dark:text-blue-200">
                Runtime service account, role bindings, and runtime token are
                auto-provisioned during{" "}
                <span className="font-semibold">Prepare Provider</span>.
              </div>
              {renderAuthFields(
                bootstrapAuth,
                (next) => setAuthConfig("bootstrap_auth", next),
                "Bootstrap Authentication (Required)",
                "Used for provider prepare/install actions that need elevated access.",
              )}
            </>
          ) : (
            renderAuthFields(
              runtimeAuth,
              (next) => setAuthConfig("runtime_auth", next),
              "Runtime Authentication (Required)",
              "Used for normal build execution and ongoing pipeline operations.",
            )
          )}
        </div>
      ),
    },
    {
      id: "config",
      label: "Configuration",
      content: (
        <div className="space-y-6">
          <CommonKubernetesFields
            formData={formData}
            setFormData={setFormData}
          />

          <div className="rounded-md border border-gray-200 dark:border-gray-700 p-4 bg-gray-50 dark:bg-gray-900/40 space-y-4">
            <div>
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white">
                Tekton Core Bootstrap
              </h3>
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Controls how Tekton core APIs are sourced during managed
                provider preparation for new clusters.
              </p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Tekton Core Source
              </label>
              <select
                value={tektonCoreInstallSource}
                onChange={(e) =>
                  setConfigField("tekton_core_install_source", e.target.value)
                }
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="manifest">Manifest URLs (auto-install)</option>
                <option value="helm">Helm (manual install, tracked)</option>
                <option value="preinstalled">Preinstalled Tekton</option>
              </select>
            </div>

            {tektonCoreInstallSource === "manifest" && (
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Tekton Core Manifest URLs (optional)
                </label>
                <textarea
                  value={tektonCoreManifestURLs}
                  onChange={(e) =>
                    setConfigField("tekton_core_manifest_urls", e.target.value)
                  }
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                  rows={3}
                  placeholder="https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  Leave empty to use default public release manifest. For
                  air-gapped clusters, provide mirrored internal URLs (one per
                  line).
                </p>
              </div>
            )}

            {tektonCoreInstallSource === "helm" && (
              <div className="space-y-4">
                <div className="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-xs text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                  Helm source is tracked for air-gapped workflows. Tekton must
                  be installed via your Helm process before running prepare.
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Helm Repo URL
                  </label>
                  <input
                    type="url"
                    value={
                      typeof formData.config?.tekton_helm_repo_url === "string"
                        ? formData.config.tekton_helm_repo_url
                        : ""
                    }
                    onChange={(e) =>
                      setConfigField("tekton_helm_repo_url", e.target.value)
                    }
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                    placeholder="https://charts.your-company.local/tekton"
                  />
                </div>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Helm Chart
                    </label>
                    <input
                      type="text"
                      value={
                        typeof formData.config?.tekton_helm_chart === "string"
                          ? formData.config.tekton_helm_chart
                          : "tekton-pipeline"
                      }
                      onChange={(e) =>
                        setConfigField("tekton_helm_chart", e.target.value)
                      }
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Release Name
                    </label>
                    <input
                      type="text"
                      value={
                        typeof formData.config?.tekton_helm_release_name ===
                        "string"
                          ? formData.config.tekton_helm_release_name
                          : "tekton-pipelines"
                      }
                      onChange={(e) =>
                        setConfigField(
                          "tekton_helm_release_name",
                          e.target.value,
                        )
                      }
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Helm Namespace
                    </label>
                    <input
                      type="text"
                      value={
                        typeof formData.config?.tekton_helm_namespace ===
                        "string"
                          ? formData.config.tekton_helm_namespace
                          : "tekton-pipelines"
                      }
                      onChange={(e) =>
                        setConfigField("tekton_helm_namespace", e.target.value)
                      }
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                    />
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      ),
    },
  ];

  return (
    <>
      <TabbedForm tabs={tabs} defaultActiveTab="auth" />

      {/* Managed Service Account Token Setup Drawer */}
      <TooltipDrawer
        isOpen={showTokenDrawer && isManagedBootstrap}
        onClose={() => setShowTokenDrawer(false)}
        title="Managed Mode: Bootstrap Token Setup"
      >
        <div className="space-y-4">
          <p className="text-gray-600 dark:text-gray-400">
            Managed mode requires only bootstrap credentials at onboarding.
            Runtime identity is generated automatically by Image Factory during
            provider preparation.
          </p>

          <CopyableCodeBlock
            title="0. Create System Namespace"
            code={`apiVersion: v1
kind: Namespace
metadata:
  name: ${targetNamespace}`}
            language="yaml"
          />

          <div className="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-xs text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
            Bootstrap credentials need elevated permissions for managed
            prepare/install actions. For local/prototype clusters, binding to{" "}
            <code>cluster-admin</code> is simplest. For production, use
            least-privilege and rotate tokens.
          </div>

          <CopyableCodeBlock
            title="1. Create Bootstrap Service Account (System Namespace)"
            code={`apiVersion: v1
kind: ServiceAccount
metadata:
  name: image-factory-bootstrap-sa
  namespace: ${targetNamespace}`}
            language="yaml"
          />
          <CopyableCodeBlock
            title="2. Bind Bootstrap Service Account (Quickstart)"
            code={`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: image-factory-bootstrap-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: image-factory-bootstrap-sa
  namespace: ${targetNamespace}`}
            language="yaml"
          />
          <CopyableCodeBlock
            title="3. Get Bootstrap Token"
            code={`kubectl create token image-factory-bootstrap-sa -n ${targetNamespace} --duration=8760h`}
            language="bash"
          />

          <div className="rounded-md border border-blue-200 bg-blue-50 px-3 py-2 text-xs text-blue-900 dark:border-blue-700 dark:bg-blue-900/20 dark:text-blue-200">
            During <span className="font-semibold">Prepare Provider</span>,
            Image Factory automatically creates the runtime service account,
            runtime role/binding, and runtime token. During{" "}
            <span className="font-semibold">Prepare Tenant Namespace</span>, it
            applies per-tenant runtime RBAC automatically. No manual runtime
            identity or per-tenant RBAC apply is required in managed mode.
          </div>
        </div>
      </TooltipDrawer>

      {/* Self-Managed Service Account Token Setup Drawer */}
      <TooltipDrawer
        isOpen={showTokenDrawer && !isManagedBootstrap}
        onClose={() => setShowTokenDrawer(false)}
        title="Self-Managed Mode: Runtime Token + Tenant RBAC Setup"
      >
        <div className="space-y-4">
          <p className="text-gray-600 dark:text-gray-400">
            Configure a runtime service account for build execution. In
            self-managed mode, you must apply tenant namespace RBAC manually.
          </p>

          <CopyableCodeBlock
            title="0. Create System Namespace"
            code={`apiVersion: v1
kind: Namespace
metadata:
  name: ${targetNamespace}`}
            language="yaml"
          />

          <CopyableCodeBlock
            title="1. Create Runtime Service Account (System Namespace)"
            code={`apiVersion: v1
kind: ServiceAccount
metadata:
  name: image-factory-runtime-sa
  namespace: ${targetNamespace}`}
            language="yaml"
          />

          <div>
            <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
              2. Get Runtime Token
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
              Run this command to retrieve the service account token for
              namespace <span className="font-semibold">{targetNamespace}</span>
              :
            </p>
            <CopyableCodeBlock
              title="Get Service Account Token"
              code={`kubectl create token image-factory-runtime-sa -n ${targetNamespace} --duration=8760h`}
              language="bash"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
              If your cluster does not support <code>kubectl create token</code>
              , or does not auto-create service-account token secrets (K8s
              1.24+), create a bound token secret and read it:
            </p>
            <CopyableCodeBlock
              title="Fallback: Create Token Secret (K8s 1.24+)"
              code={`cat <<'EOF' | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: image-factory-runtime-sa-token
  namespace: ${targetNamespace}
  annotations:
    kubernetes.io/service-account.name: image-factory-runtime-sa
type: kubernetes.io/service-account-token
EOF

kubectl -n ${targetNamespace} get secret image-factory-runtime-sa-token -o jsonpath='{.data.token}' | base64 -d`}
              language="bash"
            />
          </div>

          <div className="rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/40 p-3">
            <div className="text-xs font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-300 mb-2">
              Tenant Namespace Runtime RBAC
            </div>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
              Builds run in per-tenant namespaces (for example{" "}
              <code>image-factory-tenant1234</code>). The runtime role and
              rolebinding must exist in each tenant namespace, and the
              rolebinding should reference the runtime ServiceAccount in the
              system namespace (<code>{targetNamespace}</code>).
            </p>
            <CopyableCodeBlock
              title="Example Role + RoleBinding (per tenant namespace)"
              code={`apiVersion: rbac.authorization.k8s.io/v1
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
  resources: ["pods", "pods/log", "pods/exec", "secrets", "configmaps", "events", "persistentvolumeclaims"]
  verbs: ["create", "get", "list", "watch", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: image-factory-runtime-binding
  namespace: image-factory-tenant1234
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: image-factory-runtime-role
subjects:
- kind: ServiceAccount
  name: image-factory-runtime-sa
  namespace: ${targetNamespace}`}
              language="yaml"
            />
          </div>
        </div>
      </TooltipDrawer>

      {/* Authentication Method Setup Drawer */}
      <TooltipDrawer
        isOpen={showAuthMethodDrawer}
        onClose={() => setShowAuthMethodDrawer(false)}
        title={`${activeAuthMethod} Authentication Setup`}
      >
        <div className="space-y-4">
          {activeAuthMethod === "kubeconfig" && (
            <>
              <p className="text-gray-600 dark:text-gray-400">
                Use a kubeconfig file for authentication. This is the standard
                way to authenticate with Kubernetes clusters.
              </p>

              <div>
                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                  How it works:
                </h3>
                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                  <li>Reads authentication credentials from kubeconfig file</li>
                  <li>Supports certificates, tokens, and other auth methods</li>
                  <li>Can reference multiple clusters and users</li>
                  <li>Standard Kubernetes authentication method</li>
                </ul>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                  Setup:
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                  Provide the path to your kubeconfig file. The file should
                  contain valid cluster credentials and endpoint information.
                </p>
                <CopyableCodeBlock
                  title="Common kubeconfig locations"
                  code={`# Local development
~/.kube/config

# System-wide
/etc/kubernetes/admin.conf

# Custom path
/path/to/your/kubeconfig`}
                  language="bash"
                />
              </div>
            </>
          )}

          {activeAuthMethod === "token" && (
            <>
              <p className="text-gray-600 dark:text-gray-400">
                Use a Kubernetes service account token for authentication. This
                provides direct API access using bearer tokens.
              </p>

              <div>
                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                  How it works:
                </h3>
                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                  <li>Uses Kubernetes service account tokens</li>
                  <li>Requires API server endpoint and CA certificate</li>
                  <li>Token must have appropriate RBAC permissions</li>
                  <li>Secure method for programmatic access</li>
                </ul>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                  Setup:
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                  You'll need the cluster endpoint, service account token, and
                  CA certificate. Click the help icon next to the Service
                  Account Token field for detailed setup instructions.
                </p>
              </div>
            </>
          )}

          {activeAuthMethod === "client-cert" && (
            <>
              <p className="text-gray-600 dark:text-gray-400">
                Use client certificates for mutual TLS authentication. This is a
                secure method that doesn't require storing secrets.
              </p>

              <div>
                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                  How it works:
                </h3>
                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                  <li>Uses X.509 client certificates for authentication</li>
                  <li>
                    Requires client certificate, private key, and CA certificate
                  </li>
                  <li>Mutual TLS authentication</li>
                  <li>No secrets stored in configuration</li>
                </ul>
              </div>

              <div>
                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                  Setup:
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                  You'll need to provide the API server endpoint, client
                  certificate, client private key, and CA certificate. Obtain
                  these from your Kubernetes cluster administrator.
                </p>
              </div>
            </>
          )}
        </div>
      </TooltipDrawer>
    </>
  );
};

export const StandardKubernetesForm: ProviderFormComponent = {
  component: StandardKubernetesFormComponent,
  displayName: "Standard Kubernetes",
  description:
    "Generic Kubernetes cluster with standard authentication methods",
};
