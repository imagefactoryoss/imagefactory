import React from 'react'
import { TabbedForm } from './TabbedForm'
import { ProviderConfigFormProps, ProviderFormComponent } from './types'

const BuildNodesFormComponent: React.FC<ProviderConfigFormProps> = ({
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
                            value={formData.config.auth_method || 'ssh-key'}
                            onChange={(e) => setFormData({
                                ...formData,
                                config: { ...formData.config, auth_method: e.target.value }
                            })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-blue-500 focus:border-blue-500"
                            required
                        >
                            <option value="ssh-key">SSH Key</option>
                            <option value="password">Password</option>
                            <option value="api-token">API Token</option>
                        </select>
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            Build nodes support SSH key, password, or API token authentication
                        </p>
                    </div>
                    {(formData.config.auth_method === 'ssh-key' || !formData.config.auth_method) && (
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    SSH Private Key
                                </label>
                                <textarea
                                    value={formData.config.ssh_private_key || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, ssh_private_key: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500 font-mono"
                                    placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                                    rows={6}
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    SSH Public Key (Optional)
                                </label>
                                <input
                                    type="text"
                                    value={formData.config.ssh_public_key || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, ssh_public_key: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500 font-mono"
                                    placeholder="ssh-rsa AAAAB3NzaC1yc2E..."
                                />
                            </div>
                        </div>
                    )}
                    {formData.config.auth_method === 'password' && (
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    Username
                                </label>
                                <input
                                    type="text"
                                    value={formData.config.username || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, username: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="build-user"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    Password
                                </label>
                                <input
                                    type="password"
                                    value={formData.config.password || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, password: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="build-password"
                                />
                            </div>
                        </div>
                    )}
                    {formData.config.auth_method === 'api-token' && (
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    API Token
                                </label>
                                <input
                                    type="password"
                                    value={formData.config.api_token || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, api_token: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="api-token"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                    API Endpoint
                                </label>
                                <input
                                    type="url"
                                    value={formData.config.api_endpoint || ''}
                                    onChange={(e) => setFormData({
                                        ...formData,
                                        config: { ...formData.config, api_endpoint: e.target.value }
                                    })}
                                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="https://build-api.example.com"
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
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Host Address
                        </label>
                        <input
                            type="text"
                            value={formData.config.host || ''}
                            onChange={(e) => setFormData({
                                ...formData,
                                config: { ...formData.config, host: e.target.value }
                            })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                            placeholder="build-node.example.com"
                            required
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Port
                        </label>
                        <input
                            type="number"
                            value={formData.config.port || 22}
                            onChange={(e) => setFormData({
                                ...formData,
                                config: { ...formData.config, port: parseInt(e.target.value) }
                            })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                            placeholder="22"
                            min="1"
                            max="65535"
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Build Tools
                        </label>
                        <div className="space-y-2">
                            {['docker', 'podman', 'buildah', 'kaniko'].map((tool) => (
                                <label key={tool} className="flex items-center">
                                    <input
                                        type="checkbox"
                                        checked={formData.config.build_tools?.includes(tool) || false}
                                        onChange={(e) => {
                                            const currentTools = formData.config.build_tools || []
                                            const newTools = e.target.checked
                                                ? [...currentTools, tool]
                                                : currentTools.filter((t: string) => t !== tool)
                                            setFormData({
                                                ...formData,
                                                config: { ...formData.config, build_tools: newTools }
                                            })
                                        }}
                                        className="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500 dark:bg-gray-700"
                                    />
                                    <span className="ml-2 text-sm text-gray-700 dark:text-gray-300 capitalize">
                                        {tool}
                                    </span>
                                </label>
                            ))}
                        </div>
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            CPU Cores
                        </label>
                        <input
                            type="number"
                            value={formData.config.cpu_cores || ''}
                            onChange={(e) => setFormData({
                                ...formData,
                                config: { ...formData.config, cpu_cores: parseInt(e.target.value) }
                            })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                            placeholder="4"
                            min="1"
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Memory (GB)
                        </label>
                        <input
                            type="number"
                            value={formData.config.memory_gb || ''}
                            onChange={(e) => setFormData({
                                ...formData,
                                config: { ...formData.config, memory_gb: parseInt(e.target.value) }
                            })}
                            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:ring-blue-500 focus:border-blue-500"
                            placeholder="8"
                            min="1"
                        />
                    </div>
                </div>
            )
        }
    ]

    return <TabbedForm tabs={tabs} defaultActiveTab="auth" />
}

export const BuildNodesForm: ProviderFormComponent = {
    component: BuildNodesFormComponent,
    displayName: 'Build Nodes',
    description: 'Custom build nodes for container image building'
}