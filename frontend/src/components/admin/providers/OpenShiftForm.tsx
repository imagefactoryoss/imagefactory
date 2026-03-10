import { HelpCircle } from 'lucide-react'
import React, { useState } from 'react'
import { CommonKubernetesFields } from './CommonKubernetesFields'
import { TabbedForm } from './TabbedForm'
import { CopyableCodeBlock, TooltipDrawer } from './TooltipDrawer'
import { ProviderConfigFormProps, ProviderFormComponent } from './types'

const OpenShiftFormComponent: React.FC<ProviderConfigFormProps> = ({
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
                            <option value="token">Service Account Token</option>
                            <option value="oauth">OAuth Token</option>
                        </select>
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            OpenShift supports multiple authentication methods
                        </p>
                    </div>
                    {formData.config.auth_method === 'token' && (
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    OpenShift API Endpoint
                                </label>
                                <input
                                    type="url"
                                    value={formData.config.endpoint || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, endpoint: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="https://api.openshift.example.com:6443"
                                    required
                                />
                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                    OpenShift API server endpoint
                                </p>
                            </div>
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
                                <textarea
                                    value={formData.config.token || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, token: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    rows={4}
                                    placeholder="eyJhbGciOiJSUzI1NiIsImtpZCI6..."
                                    required
                                />
                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                    Service account token with cluster access permissions
                                </p>
                            </div>
                        </div>
                    )}
                    {formData.config.auth_method === 'oauth' && (
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    OAuth Token
                                </label>
                                <input
                                    type="password"
                                    value={formData.config.oauth_token || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, oauth_token: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="oauth-token"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    OAuth Server URL
                                </label>
                                <input
                                    type="url"
                                    value={formData.config.oauth_server_url || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, oauth_server_url: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="https://oauth-openshift.example.com"
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
                                Use a kubeconfig file for authentication. This works with OpenShift clusters that have been added to your kubeconfig.
                            </p>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    How it works:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Reads authentication credentials from kubeconfig file</li>
                                    <li>Supports OpenShift login tokens and certificates</li>
                                    <li>Can reference multiple OpenShift clusters</li>
                                    <li>Standard Kubernetes/OpenShift authentication</li>
                                </ul>
                            </div>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    Setup:
                                </h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                                    Use <code>oc login</code> to authenticate and update your kubeconfig, or provide the path to an existing kubeconfig file.
                                </p>
                                <CopyableCodeBlock
                                    title="Login to OpenShift"
                                    code="oc login https://api.openshift.example.com:6443"
                                    language="bash"
                                />
                            </div>
                        </>
                    )}

                    {(formData.config.auth_method || 'kubeconfig') === 'token' && (
                        <>
                            <p className="text-gray-600 dark:text-gray-400">
                                Use an OpenShift service account token for authentication. This provides direct API access to OpenShift clusters.
                            </p>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    How it works:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Uses OpenShift service account tokens</li>
                                    <li>Requires API server endpoint and CA certificate</li>
                                    <li>Token must have appropriate SCC and RBAC permissions</li>
                                    <li>Works with OpenShift's security context constraints</li>
                                </ul>
                            </div>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    Setup:
                                </h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                                    You'll need the OpenShift API endpoint, service account token, and CA certificate. Click the help icon next to the Service Account Token field for detailed setup instructions.
                                </p>
                            </div>
                        </>
                    )}

                    {(formData.config.auth_method || 'kubeconfig') === 'oauth' && (
                        <>
                            <p className="text-gray-600 dark:text-gray-400">
                                Use OpenShift OAuth tokens for authentication. This method uses OpenShift's built-in OAuth server for authentication.
                            </p>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    How it works:
                                </h3>
                                <ul className="text-sm text-gray-600 dark:text-gray-400 space-y-1 list-disc list-inside">
                                    <li>Uses OpenShift OAuth access tokens</li>
                                    <li>Requires OAuth server URL and access token</li>
                                    <li>Tokens can be obtained from OpenShift web console</li>
                                    <li>Integrates with OpenShift identity providers</li>
                                </ul>
                            </div>

                            <div>
                                <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
                                    Setup:
                                </h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                                    Obtain an OAuth token from the OpenShift web console or use <code>oc whoami -t</code> after logging in.
                                </p>
                                <CopyableCodeBlock
                                    title="Get current OAuth token"
                                    code="oc whoami -t"
                                    language="bash"
                                />
                            </div>
                        </>
                    )}
                </div>
            </TooltipDrawer>
        </>
    )
}

export const OpenShiftForm: ProviderFormComponent = {
    component: OpenShiftFormComponent,
    displayName: 'OpenShift',
    description: 'Red Hat OpenShift Container Platform'
}