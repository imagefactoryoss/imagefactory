import { HelpCircle } from 'lucide-react'
import React, { useState } from 'react'
import { CommonKubernetesFields } from './CommonKubernetesFields'
import { TabbedForm } from './TabbedForm'
import { CopyableCodeBlock, TooltipDrawer } from './TooltipDrawer'
import { ProviderConfigFormProps, ProviderFormComponent } from './types'

const AzureAKSFormComponent: React.FC<ProviderConfigFormProps> = ({
    formData,
    setFormData
}) => {
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
                            value={formData.config.auth_method || 'managed-identity'}
                            onChange={(e) => setFormData({
                                ...formData,
                                config: { ...formData.config, auth_method: e.target.value }
                            })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-blue-500 focus:border-blue-500"
                            required
                        >
                            <option value="managed-identity">Managed Identity</option>
                            <option value="service-principal">Service Principal</option>
                            <option value="kubeconfig">Kubeconfig File</option>
                        </select>
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            Azure AKS supports Managed Identity for authentication (recommended)
                        </p>
                    </div>
                    {(formData.config.auth_method === 'managed-identity' || (!formData.config.auth_method)) && (
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                Subscription ID
                            </label>
                            <input
                                type="text"
                                value={formData.config.subscription_id || ''}
                                onChange={(e) => setFormData({
                                    ...formData,
                                    config: { ...formData.config, subscription_id: e.target.value }
                                })}
                                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                placeholder="12345678-1234-1234-1234-123456789012"
                            />
                            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                Azure subscription ID where the AKS cluster is located
                            </p>
                        </div>
                    )}
                    {formData.config.auth_method === 'service-principal' && (
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    Client ID
                                </label>
                                <input
                                    type="text"
                                    value={formData.config.client_id || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, client_id: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="azure-client-id"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    Client Secret
                                </label>
                                <input
                                    type="password"
                                    value={formData.config.client_secret || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, client_secret: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="azure-client-secret"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    Tenant ID
                                </label>
                                <input
                                    type="text"
                                    value={formData.config.tenant_id || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, tenant_id: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="azure-tenant-id"
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
            {showAuthMethodDrawer && (
                <TooltipDrawer
                    isOpen={showAuthMethodDrawer}
                    title={`Setting up ${formData.config.auth_method || 'managed-identity'} authentication for Azure AKS`}
                    onClose={() => setShowAuthMethodDrawer(false)}
                >
                    {formData.config.auth_method === 'managed-identity' && (
                        <div className="space-y-6">
                            <div>
                                <h3 className="text-lg font-semibold mb-2">Managed Identity Setup</h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                                    Use Azure's managed identity for secure, passwordless authentication to your AKS cluster.
                                </p>
                                <div className="space-y-4">
                                    <div>
                                        <h4 className="font-medium mb-2">1. Enable Managed Identity on your VM/VMSS</h4>
                                        <CopyableCodeBlock
                                            title="Enable Managed Identity on VM/VMSS"
                                            code={`# For VM
az vm identity assign --resource-group <resource-group> --name <vm-name>

# For VMSS
az vmss identity assign --resource-group <resource-group> --name <vmss-name>`}
                                            language="bash"
                                        />
                                    </div>
                                    <div>
                                        <h4 className="font-medium mb-2">2. Grant AKS Cluster Access</h4>
                                        <CopyableCodeBlock
                                            title="Grant AKS Cluster Access"
                                            code={`# Get cluster resource ID
CLUSTER_RESOURCE_ID=$(az aks show --resource-group <resource-group> --name <cluster-name> --query id -o tsv)

# Assign Azure Kubernetes Service Cluster User Role
az role assignment create \\
  --assignee <managed-identity-principal-id> \\
  --role "Azure Kubernetes Service Cluster User Role" \\
  --scope $CLUSTER_RESOURCE_ID`}
                                            language="bash"
                                        />
                                    </div>
                                    <div>
                                        <h4 className="font-medium mb-2">3. Configure Provider</h4>
                                        <p className="text-sm mb-2">Set the following values in the provider configuration:</p>
                                        <ul className="list-disc list-inside text-sm space-y-1">
                                            <li><strong>Subscription ID:</strong> Your Azure subscription ID</li>
                                            <li><strong>Resource Group:</strong> The resource group containing your AKS cluster</li>
                                            <li><strong>Cluster Name:</strong> Your AKS cluster name</li>
                                            <li><strong>Auth Method:</strong> managed-identity</li>
                                        </ul>
                                    </div>
                                </div>
                            </div>
                        </div>
                    )}
                    {formData.config.auth_method === 'service-principal' && (
                        <div className="space-y-6">
                            <div>
                                <h3 className="text-lg font-semibold mb-2">Service Principal Setup</h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                                    Create a service principal with appropriate permissions to access your AKS cluster.
                                </p>
                                <div className="space-y-4">
                                    <div>
                                        <h4 className="font-medium mb-2">1. Create Service Principal</h4>
                                        <CopyableCodeBlock
                                            title="Create Service Principal"
                                            code={`# Create service principal
az ad sp create-for-rbac --name "image-factory-sp" --role "Azure Kubernetes Service Cluster User Role" --scopes /subscriptions/<subscription-id>/resourceGroups/<resource-group>/providers/Microsoft.ContainerService/managedClusters/<cluster-name>

# Output will include:
# {
#   "appId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
#   "displayName": "image-factory-sp",
#   "password": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
#   "tenant": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
# }`}
                                            language="bash"
                                        />
                                    </div>
                                    <div>
                                        <h4 className="font-medium mb-2">2. Configure Provider</h4>
                                        <p className="text-sm mb-2">Use the service principal credentials in the provider configuration:</p>
                                        <ul className="list-disc list-inside text-sm space-y-1">
                                            <li><strong>Subscription ID:</strong> Your Azure subscription ID</li>
                                            <li><strong>Client ID:</strong> The appId from the service principal creation</li>
                                            <li><strong>Client Secret:</strong> The password from the service principal creation</li>
                                            <li><strong>Tenant ID:</strong> The tenant from the service principal creation</li>
                                            <li><strong>Auth Method:</strong> service-principal</li>
                                        </ul>
                                    </div>
                                </div>
                            </div>
                        </div>
                    )}
                    {formData.config.auth_method === 'kubeconfig' && (
                        <div className="space-y-6">
                            <div>
                                <h3 className="text-lg font-semibold mb-2">Kubeconfig Setup</h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                                    Use a kubeconfig file for direct cluster access. This method requires downloading and securely storing the kubeconfig.
                                </p>
                                <div className="space-y-4">
                                    <div>
                                        <h4 className="font-medium mb-2">1. Get AKS Kubeconfig</h4>
                                        <CopyableCodeBlock
                                            title="Get AKS Kubeconfig"
                                            code={`# Download kubeconfig for your AKS cluster
az aks get-credentials --resource-group <resource-group> --name <cluster-name> --file kubeconfig-aks

# Verify the kubeconfig
kubectl --kubeconfig=kubeconfig-aks cluster-info`}
                                            language="bash"
                                        />
                                    </div>
                                    <div>
                                        <h4 className="font-medium mb-2">2. Configure Provider</h4>
                                        <p className="text-sm mb-2">Upload the kubeconfig file content in the provider configuration:</p>
                                        <ul className="list-disc list-inside text-sm space-y-1">
                                            <li><strong>Kubeconfig:</strong> Paste the entire content of your kubeconfig file</li>
                                            <li><strong>Auth Method:</strong> kubeconfig</li>
                                        </ul>
                                        <p className="text-sm text-amber-600 dark:text-amber-400 mt-2">
                                            ⚠️ Store the kubeconfig securely and rotate it regularly for security.
                                        </p>
                                    </div>
                                </div>
                            </div>
                        </div>
                    )}
                    {(!formData.config.auth_method || formData.config.auth_method === 'managed-identity') && (
                        <div className="space-y-6">
                            <div>
                                <h3 className="text-lg font-semibold mb-2">Default: Managed Identity Setup</h3>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                                    Use Azure's managed identity for secure, passwordless authentication to your AKS cluster.
                                </p>
                                <div className="space-y-4">
                                    <div>
                                        <h4 className="font-medium mb-2">1. Enable Managed Identity on your VM/VMSS</h4>
                                        <CopyableCodeBlock
                                            title="Enable Managed Identity on VM/VMSS"
                                            code={`# For VM
az vm identity assign --resource-group <resource-group> --name <vm-name>

# For VMSS
az vmss identity assign --resource-group <resource-group> --name <vmss-name>`}
                                            language="bash"
                                        />
                                    </div>
                                    <div>
                                        <h4 className="font-medium mb-2">2. Grant AKS Cluster Access</h4>
                                        <CopyableCodeBlock
                                            title="Grant AKS Cluster Access"
                                            code={`# Get cluster resource ID
CLUSTER_RESOURCE_ID=$(az aks show --resource-group <resource-group> --name <cluster-name> --query id -o tsv)

# Assign Azure Kubernetes Service Cluster User Role
az role assignment create \\
  --assignee <managed-identity-principal-id> \\
  --role "Azure Kubernetes Service Cluster User Role" \\
  --scope $CLUSTER_RESOURCE_ID`}
                                            language="bash"
                                        />
                                    </div>
                                    <div>
                                        <h4 className="font-medium mb-2">3. Configure Provider</h4>
                                        <p className="text-sm mb-2">Set the following values in the provider configuration:</p>
                                        <ul className="list-disc list-inside text-sm space-y-1">
                                            <li><strong>Subscription ID:</strong> Your Azure subscription ID</li>
                                            <li><strong>Resource Group:</strong> The resource group containing your AKS cluster</li>
                                            <li><strong>Cluster Name:</strong> Your AKS cluster name</li>
                                            <li><strong>Auth Method:</strong> managed-identity</li>
                                        </ul>
                                    </div>
                                </div>
                            </div>
                        </div>
                    )}
                </TooltipDrawer>
            )}
        </>
    )
}

export const AzureAKSForm: ProviderFormComponent = {
    component: AzureAKSFormComponent,
    displayName: 'Azure AKS',
    description: 'Azure Kubernetes Service with Managed Identity'
}