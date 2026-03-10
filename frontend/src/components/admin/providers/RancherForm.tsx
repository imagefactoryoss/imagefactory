import { HelpCircle } from 'lucide-react'
import React, { useState } from 'react'
import { CommonKubernetesFields } from './CommonKubernetesFields'
import { TabbedForm } from './TabbedForm'
import { CopyableCodeBlock, TooltipDrawer } from './TooltipDrawer'
import { ProviderConfigFormProps, ProviderFormComponent } from './types'

const RancherFormComponent: React.FC<ProviderConfigFormProps> = ({
    formData,
    setFormData
}) => {
    const [showTokenDrawer, setShowTokenDrawer] = useState(false)
    const [showAuthMethodDrawer, setShowAuthMethodDrawer] = useState(false)
    const tabs = [
        {
            id: 'auth',
            label: 'Authentication',
            content: (
                <div className="space-y-4">
                    <div>
                        <div className="flex items-center gap-2 mb-1">
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                Authentication Method
                            </label>
                            <HelpCircle
                                className="h-4 w-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 cursor-pointer"
                                onClick={(e) => { e.stopPropagation(); setShowAuthMethodDrawer(true); }}
                            />
                        </div>
                        <select
                            value={formData.config.auth_method || 'kubeconfig'}
                            onChange={(e) => setFormData({
                                ...formData,
                                config: { ...formData.config, auth_method: e.target.value }
                            })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-blue-500 focus:border-blue-500"
                            required
                        >
                            <option value="kubeconfig">Kubeconfig File</option>
                            <option value="api-key">API Key</option>
                            <option value="service-account">Service Account Token</option>
                        </select>
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            Rancher supports multiple authentication methods
                        </p>
                    </div>
                    {formData.config.auth_method === 'api-key' && (
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    Rancher Server Endpoint
                                </label>
                                <input
                                    type="url"
                                    value={formData.config.endpoint || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, endpoint: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="https://rancher.example.com"
                                    required
                                />
                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                    Rancher server API endpoint
                                </p>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    Rancher API Key
                                </label>
                                <input
                                    type="password"
                                    value={formData.config.api_key || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, api_key: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="token-abcde..."
                                    required
                                />
                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                    Rancher API key with cluster access
                                </p>
                            </div>
                        </div>
                    )}
                    {formData.config.auth_method === 'service-account' && (
                        <div className="space-y-4">
                            <div>
                                <div className="flex items-center gap-2 mb-1">
                                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                        Service Account Token
                                    </label>
                                    <div className="group relative">
                                        <HelpCircle
                                            className="h-4 w-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 cursor-pointer"
                                            onClick={(e) => { e.stopPropagation(); setShowTokenDrawer(true); }}
                                        />
                                    </div>
                                </div>
                                <input
                                    type="password"
                                    value={formData.config.service_account_token || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, service_account_token: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="service-account-token"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    Service Account CA Certificate
                                </label>
                                <textarea
                                    value={formData.config.service_account_ca_cert || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, service_account_ca_cert: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="-----BEGIN CERTIFICATE-----"
                                    rows={4}
                                />
                            </div>
                        </div>
                    )}
                </div>
            )
        },
        {
            id: 'config',
            label: 'Configuration',
            content: (
                <CommonKubernetesFields formData={formData} setFormData={setFormData} />
            )
        }
    ]

    return (
        <>
            <TabbedForm tabs={tabs} defaultActiveTab="auth" />

            {/* Service Account Token Setup Drawer */}
            <TooltipDrawer
                isOpen={showTokenDrawer}
                onClose={() => setShowTokenDrawer(false)}
                title="Service Account Token Setup"
            >
                <div className="space-y-4">
                    <p className="text-gray-600 dark:text-gray-400">
                        To get a service account token with the required permissions, create the following resources in your cluster:
                    </p>

                    <CopyableCodeBlock
                        title="1. Create the Service Account"
                        code={`apiVersion: v1
kind: ServiceAccount
metadata:
  name: image-factory-sa
  namespace: default`}
                        language="yaml"
                    />

                    <CopyableCodeBlock
                        title="2. Create the Cluster Role"
                        code={`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: image-factory-role
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log", "pods/exec", "services", "configmaps", "secrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets", "statefulsets", "daemonsets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["batch"]
  resources: ["jobs", "cronjobs"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["networking.k8s.io"]
  resources: ["ingresses", "networkpolicies"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]`}
                        language="yaml"
                    />

                    <CopyableCodeBlock
                        title="3. Create the Cluster Role Binding"
                        code={`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: image-factory-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: image-factory-role
subjects:
- kind: ServiceAccount
  name: image-factory-sa
  namespace: default`}
                        language="yaml"
                    />

                    <div>
                        <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                            4. Get the Token
                        </h3>
                        <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
                            Run this command to retrieve the service account token:
                        </p>
                        <CopyableCodeBlock
                            title="Get Service Account Token"
                            code="kubectl get secret $(kubectl get sa image-factory-sa -o jsonpath='{.secrets[0].name}') -o jsonpath='{.data.token}' | base64 -d"
                            language="bash"
                        />
                    </div>
                </div>
            </TooltipDrawer>

            {/* Authentication Method Setup Drawer */}
            <TooltipDrawer
                isOpen={showAuthMethodDrawer}
                onClose={() => setShowAuthMethodDrawer(false)}
                title={`${formData.config.auth_method || 'kubeconfig'} Authentication Setup`}
            >
                <div className="space-y-4">
                    {(formData.config.auth_method || 'kubeconfig') === 'kubeconfig' && (
                        <>
                            <p className="text-gray-600 dark:text-gray-400">
                                Use a kubeconfig file for authentication. This works with Rancher-managed clusters that have been imported into your kubeconfig.
                            </p>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    How it works:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Reads authentication credentials from kubeconfig file</li>
                                    <li>Supports Rancher-generated kubeconfig files</li>
                                    <li>Can reference multiple Rancher clusters</li>
                                    <li>Standard Kubernetes authentication method</li>
                                </ul>
                            </div>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    Setup:
                                </h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                                    Download the kubeconfig file from the Rancher UI for the cluster you want to access, or use Rancher CLI to generate one.
                                </p>
                                <CopyableCodeBlock
                                    title="Download kubeconfig with Rancher CLI"
                                    code="rancher kubectl config view --raw"
                                    language="bash"
                                />
                            </div>
                        </>
                    )}

                    {(formData.config.auth_method || 'kubeconfig') === 'api-key' && (
                        <>
                            <p className="text-gray-600 dark:text-gray-400">
                                Use Rancher API keys for authentication. This method provides access to Rancher-managed clusters through the Rancher API.
                            </p>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    How it works:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Uses Rancher API keys (Bearer tokens)</li>
                                    <li>Requires Rancher server endpoint</li>
                                    <li>API key must have appropriate permissions</li>
                                    <li>Provides access to all Rancher-managed clusters</li>
                                </ul>
                            </div>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    Setup:
                                </h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                                    Create an API key in the Rancher UI under your user account. The key will be in the format "token-xxx".
                                </p>
                                <CopyableCodeBlock
                                    title="API Key Format"
                                    code="token-abc123def456..."
                                    language="text"
                                />
                            </div>
                        </>
                    )}

                    {(formData.config.auth_method || 'kubeconfig') === 'service-account' && (
                        <>
                            <p className="text-gray-600 dark:text-gray-400">
                                Use a Kubernetes service account token for direct cluster access. This bypasses Rancher and connects directly to the downstream cluster.
                            </p>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    How it works:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Uses Kubernetes service account tokens</li>
                                    <li>Requires direct cluster endpoint (not Rancher URL)</li>
                                    <li>Token must have appropriate RBAC permissions</li>
                                    <li>Bypasses Rancher API for direct cluster access</li>
                                </ul>
                            </div>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    Setup:
                                </h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                                    You'll need the direct cluster endpoint, service account token, and CA certificate. Click the help icon next to the Service Account Token field for detailed setup instructions.
                                </p>
                            </div>
                        </>
                    )}
                </div>
            </TooltipDrawer>
        </>
    )
}

export const RancherForm: ProviderFormComponent = {
    component: RancherFormComponent,
    displayName: 'Rancher',
    description: 'SUSE Rancher Kubernetes Management Platform'
}