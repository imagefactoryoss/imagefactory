import React from 'react'
import { CommonKubernetesFields } from './CommonKubernetesFields'
import { TabbedForm } from './TabbedForm'
import { ProviderConfigFormProps, ProviderFormComponent } from './types'

const VMwareVKSFormComponent: React.FC<ProviderConfigFormProps> = ({
    formData,
    setFormData
}) => {
    const tabs = [
        {
            id: 'auth',
            label: 'Authentication',
            content: (
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Authentication Method
                        </label>
                        <select
                            value={formData.config.auth_method || 'api-token'}
                            onChange={(e) => setFormData({
                                ...formData,
                                config: { ...formData.config, auth_method: e.target.value }
                            })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-blue-500 focus:border-blue-500"
                            required
                        >
                            <option value="api-token">API Token</option>
                            <option value="kubeconfig">Kubeconfig File</option>
                        </select>
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            VMware vKS supports API tokens for authentication
                        </p>
                    </div>
                    {(formData.config.auth_method === 'api-token' || !formData.config.auth_method) && (
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    VMware Cloud Server
                                </label>
                                <input
                                    type="url"
                                    value={formData.config.server || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, server: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="https://vmc.vmware.com"
                                />
                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                    VMware Cloud server endpoint
                                </p>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    VMware Cloud API Token
                                </label>
                                <input
                                    type="password"
                                    value={formData.config.api_token || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, api_token: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="vmware-api-token"
                                />
                                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                    VMware Cloud API token with vKS cluster access
                                </p>
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

    return <TabbedForm tabs={tabs} defaultActiveTab="auth" />
}

export const VMwareVKSForm: ProviderFormComponent = {
    component: VMwareVKSFormComponent,
    displayName: 'VMware vKS',
    description: 'VMware Tanzu Kubernetes Service'
}