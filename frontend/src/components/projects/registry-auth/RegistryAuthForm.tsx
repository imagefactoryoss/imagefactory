import { registryAuthClient } from '@/api/registryAuthClient'
import { CreateRegistryAuthRequest, RegistryAuthType } from '@/types/registryAuth'
import { RegistryAuth } from '@/types/registryAuth'
import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'

interface RegistryAuthFormProps {
    projectId?: string
    allowProjectScope?: boolean
    authToEdit?: RegistryAuth
    onSuccess: () => void
    onCancel: () => void
}

type Scope = 'project' | 'tenant'

const registryHostSuggestionsByType: Record<string, string[]> = {
    generic: [
        'registry.example.com',
        'harbor.example.com',
        'registry.gitlab.com',
        'gitlab.example.com:5050',
        'quay.io',
    ],
    docker_hub: [
        'docker.io',
        'index.docker.io',
        'registry-1.docker.io',
    ],
    ghcr: [
        'ghcr.io',
    ],
    ecr: [
        'public.ecr.aws',
        '<account-id>.dkr.ecr.us-east-1.amazonaws.com',
    ],
    gcr: [
        'gcr.io',
        'us.gcr.io',
        'eu.gcr.io',
        'asia.gcr.io',
        '<project-id>.pkg.dev',
    ],
    acr: [
        '<registry-name>.azurecr.io',
    ],
    harbor: [
        'harbor.example.com',
    ],
}

const RegistryAuthForm: React.FC<RegistryAuthFormProps> = ({ projectId, authToEdit, onSuccess, onCancel }) => {
    const isEditMode = Boolean(authToEdit)
    const [scope, setScope] = useState<Scope>(authToEdit?.scope === 'tenant' ? 'tenant' : (projectId ? 'project' : 'tenant'))
    const [loading, setLoading] = useState(false)
    const [showHostSuggestions, setShowHostSuggestions] = useState(false)
    const [hostInputLocked, setHostInputLocked] = useState(true)
    const [tokenInputLocked, setTokenInputLocked] = useState(true)
    const [formData, setFormData] = useState({
        name: authToEdit?.name || '',
        description: authToEdit?.description || '',
        registry_type: authToEdit?.registry_type || 'generic',
        auth_type: (authToEdit?.auth_type || 'token') as RegistryAuthType,
        registry_host: authToEdit?.registry_host || '',
        is_default: authToEdit?.is_default || false,
        username: '',
        password: '',
        token: '',
        dockerconfigjson: '',
    })

    useEffect(() => {
        if (!authToEdit) return
        setScope(authToEdit.scope === 'tenant' ? 'tenant' : 'project')
        setFormData({
            name: authToEdit.name || '',
            description: authToEdit.description || '',
            registry_type: authToEdit.registry_type || 'generic',
            auth_type: (authToEdit.auth_type || 'token') as RegistryAuthType,
            registry_host: authToEdit.registry_host || '',
            is_default: authToEdit.is_default || false,
            username: '',
            password: '',
            token: '',
            dockerconfigjson: '',
        })
    }, [authToEdit])

    useEffect(() => {
        if (formData.auth_type === 'token') {
            setTokenInputLocked(true)
        }
    }, [formData.auth_type])

    const helperText = useMemo(() => {
        if (formData.auth_type === 'basic_auth') {
            return 'Use username/password for private registries.'
        }
        if (formData.auth_type === 'dockerconfigjson') {
            return 'Paste the full Docker config JSON for this registry.'
        }
        return 'Use an access token from your registry provider.'
    }, [formData.auth_type])

    const registryHostSuggestions = useMemo(() => {
        const type = formData.registry_type || 'generic'
        const typeSpecific = registryHostSuggestionsByType[type] || []
        const generic = registryHostSuggestionsByType.generic || []
        return Array.from(new Set([...typeSpecific, ...generic]))
    }, [formData.registry_type])

    const filteredRegistryHostSuggestions = useMemo(() => {
        const query = (formData.registry_host || '').trim().toLowerCase()
        if (!query) return registryHostSuggestions
        return registryHostSuggestions.filter((host) => host.toLowerCase().includes(query))
    }, [formData.registry_host, registryHostSuggestions])

    const handleChange = (field: string, value: string | boolean) => {
        setFormData(prev => ({
            ...prev,
            [field]: value,
        }))
    }

    const buildCredentials = (): Record<string, string> | null => {
        const username = formData.username.trim()
        const password = formData.password.trim()
        const token = formData.token.trim()
        const dockerconfigjson = formData.dockerconfigjson.trim()
        const allowEmptyCredentials = isEditMode

        switch (formData.auth_type) {
            case 'basic_auth':
                if (!username || !password) {
                    if (allowEmptyCredentials) return {}
                    toast.error('Username and password are required for basic auth')
                    return null
                }
                return { username, password }
            case 'token':
                if (!token) {
                    if (allowEmptyCredentials) return {}
                    toast.error('Token is required for token auth')
                    return null
                }
                return { token }
            case 'dockerconfigjson':
                if (!dockerconfigjson) {
                    if (allowEmptyCredentials) return {}
                    toast.error('Docker config JSON is required')
                    return null
                }
                return { dockerconfigjson }
            default:
                toast.error('Unsupported auth type')
                return null
        }
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()

        const name = formData.name.trim()
        const registryHost = formData.registry_host.trim()

        if (!name) {
            toast.error('Name is required')
            return
        }
        if (!registryHost) {
            toast.error('Registry host is required')
            return
        }
        if (scope === 'project' && !projectId) {
            toast.error('Select a project to create project-scoped registry auth')
            return
        }

        const credentials = buildCredentials()
        if (!credentials) {
            return
        }

        const payload: CreateRegistryAuthRequest = {
            name,
            description: formData.description.trim(),
            registry_type: formData.registry_type,
            auth_type: formData.auth_type,
            registry_host: registryHost,
            is_default: formData.is_default,
            credentials,
            project_id: scope === 'project' ? projectId : undefined,
        }

        try {
            setLoading(true)
            if (authToEdit) {
                await registryAuthClient.updateRegistryAuth(authToEdit.id, {
                    name: payload.name,
                    description: payload.description,
                    registry_type: payload.registry_type,
                    auth_type: payload.auth_type,
                    registry_host: payload.registry_host,
                    is_default: payload.is_default,
                    credentials: Object.keys(payload.credentials).length > 0 ? payload.credentials : undefined,
                })
                toast.success('Registry authentication updated successfully')
            } else {
                await registryAuthClient.createRegistryAuth(payload)
                toast.success('Registry authentication created successfully')
            }
            onSuccess()
        } catch (error) {
            const message = error instanceof Error
                ? error.message
                : isEditMode
                    ? 'Failed to update registry authentication'
                    : 'Failed to create registry authentication'
            toast.error(message)
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow border border-slate-200 dark:border-slate-700 p-6">
            <div className="mb-6">
                <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                    {isEditMode ? 'Edit Registry Authentication' : 'Add Registry Authentication'}
                </h3>
                <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                    {isEditMode
                        ? 'Update metadata and rotate credentials as needed.'
                        : 'Configure credentials for pushing and pulling images.'}
                </p>
            </div>

            <form onSubmit={handleSubmit} className="space-y-6" autoComplete="off">
                {/* Autofill decoys: reduce browser/password-manager credential hijacking in non-credential fields. */}
                <input
                    type="text"
                    name="username"
                    autoComplete="username"
                    tabIndex={-1}
                    aria-hidden="true"
                    className="absolute -left-[9999px] h-0 w-0 opacity-0 pointer-events-none"
                />
                <input
                    type="password"
                    name="password"
                    autoComplete="current-password"
                    tabIndex={-1}
                    aria-hidden="true"
                    className="absolute -left-[9999px] h-0 w-0 opacity-0 pointer-events-none"
                />
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Scope *</label>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                        <label className="flex items-center border border-slate-300 dark:border-slate-600 rounded-md px-3 py-2 cursor-pointer">
                            <input
                                type="radio"
                                name="scope"
                                checked={scope === 'project'}
                                disabled={isEditMode || !projectId}
                                onChange={() => setScope('project')}
                                className="mr-2"
                            />
                            <span className="text-sm text-slate-700 dark:text-slate-200">Project scope</span>
                        </label>
                        <label className="flex items-center border border-slate-300 dark:border-slate-600 rounded-md px-3 py-2 cursor-pointer">
                            <input
                                type="radio"
                                name="scope"
                                checked={scope === 'tenant'}
                                disabled={isEditMode}
                                onChange={() => setScope('tenant')}
                                className="mr-2"
                            />
                            <span className="text-sm text-slate-700 dark:text-slate-200">Tenant scope</span>
                        </label>
                    </div>
                    {isEditMode && (
                        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                            Scope cannot be changed while editing an existing auth.
                        </p>
                    )}
                    {!projectId && !isEditMode && (
                        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                            Select a project in filters above to enable project scope.
                        </p>
                    )}
                </div>

                <div>
                    <label htmlFor="registry-auth-name" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        Name *
                    </label>
                    <input
                        id="registry-auth-name"
                        type="text"
                        value={formData.name}
                        onChange={(e) => handleChange('name', e.target.value)}
                        placeholder="e.g. GHCR Team Token"
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-slate-700 dark:text-white"
                        required
                    />
                </div>

                <div>
                    <label htmlFor="registry-auth-description" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        Description
                    </label>
                    <textarea
                        id="registry-auth-description"
                        rows={2}
                        value={formData.description}
                        onChange={(e) => handleChange('description', e.target.value)}
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-slate-700 dark:text-white"
                        placeholder="Optional description"
                    />
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                        <label htmlFor="registry-type" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            Registry Type *
                        </label>
                        <select
                            id="registry-type"
                            value={formData.registry_type}
                            onChange={(e) => handleChange('registry_type', e.target.value)}
                            className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-slate-700 dark:text-white"
                        >
                            <option value="generic">Generic</option>
                            <option value="docker_hub">Docker Hub</option>
                            <option value="ghcr">GHCR</option>
                            <option value="ecr">AWS ECR</option>
                            <option value="gcr">Google GCR/Artifact Registry</option>
                            <option value="acr">Azure ACR</option>
                            <option value="harbor">Harbor</option>
                        </select>
                    </div>

                    <div>
                        <label htmlFor="registry-endpoint" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            Registry Host *
                        </label>
                        <div className="relative">
                            <input
                                id="registry-endpoint"
                                name="registry_endpoint"
                                type="text"
                                value={formData.registry_host}
                                onChange={(e) => {
                                    handleChange('registry_host', e.target.value)
                                    setShowHostSuggestions(true)
                                }}
                                onFocus={() => {
                                    setHostInputLocked(false)
                                    setShowHostSuggestions(true)
                                }}
                                onBlur={() => {
                                    window.setTimeout(() => setShowHostSuggestions(false), 120)
                                }}
                                placeholder="e.g. ghcr.io"
                                readOnly={hostInputLocked}
                                autoComplete="section-registry no-autofill"
                                autoCapitalize="none"
                                autoCorrect="off"
                                spellCheck={false}
                                data-lpignore="true"
                                data-1p-ignore="true"
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-slate-700 dark:text-white"
                                required
                            />
                            {showHostSuggestions && filteredRegistryHostSuggestions.length > 0 && (
                                <div className="absolute z-20 mt-1 max-h-44 w-full overflow-y-auto rounded-md border border-slate-200 bg-white shadow-lg dark:border-slate-700 dark:bg-slate-800">
                                    {filteredRegistryHostSuggestions.map((host) => (
                                        <button
                                            key={host}
                                            type="button"
                                            onMouseDown={() => {
                                                handleChange('registry_host', host)
                                                setShowHostSuggestions(false)
                                            }}
                                            className="block w-full px-3 py-2 text-left text-sm text-slate-700 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700"
                                        >
                                            {host}
                                        </button>
                                    ))}
                                </div>
                            )}
                        </div>
                    </div>
                </div>

                <div>
                    <label htmlFor="registry-auth-type" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        Auth Type *
                    </label>
                    <select
                        id="registry-auth-type"
                        value={formData.auth_type}
                        onChange={(e) => handleChange('auth_type', e.target.value as RegistryAuthType)}
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-slate-700 dark:text-white"
                    >
                        <option value="token">Token</option>
                        <option value="basic_auth">Basic Auth</option>
                        <option value="dockerconfigjson">Docker Config JSON</option>
                    </select>
                    <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{helperText}</p>
                    {isEditMode && (
                        <p className="mt-1 text-xs text-amber-700 dark:text-amber-300">
                            Leave credential fields blank to keep existing stored credentials.
                        </p>
                    )}
                </div>

                {formData.auth_type === 'basic_auth' && (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                            <label htmlFor="registry-username" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                Username *
                            </label>
                            {isEditMode && (
                                <div className="mb-2">
                                    <input
                                        type="text"
                                        value="*****"
                                        readOnly
                                        aria-label="Stored username"
                                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-slate-50 dark:bg-slate-800 text-slate-600 dark:text-slate-300"
                                    />
                                </div>
                            )}
                            <input
                                id="registry-username"
                                type="text"
                                value={formData.username}
                                onChange={(e) => handleChange('username', e.target.value)}
                                placeholder={isEditMode ? 'Enter new username to rotate' : undefined}
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-slate-700 dark:text-white"
                            />
                        </div>
                        <div>
                            <label htmlFor="registry-password" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                Password *
                            </label>
                            {isEditMode && (
                                <div className="mb-2">
                                    <input
                                        type="password"
                                        value="*****"
                                        readOnly
                                        aria-label="Stored password"
                                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-slate-50 dark:bg-slate-800 text-slate-600 dark:text-slate-300"
                                    />
                                </div>
                            )}
                            <input
                                id="registry-password"
                                type="password"
                                value={formData.password}
                                onChange={(e) => handleChange('password', e.target.value)}
                                placeholder={isEditMode ? 'Enter new password to rotate' : undefined}
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-slate-700 dark:text-white"
                            />
                        </div>
                    </div>
                )}

                {formData.auth_type === 'token' && (
                    <div>
                        <label htmlFor="registry-token" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            Token *
                        </label>
                        {isEditMode && (
                            <div className="mb-2">
                                <input
                                    type="password"
                                    value="*****"
                                    readOnly
                                    aria-label="Stored token"
                                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-slate-50 dark:bg-slate-800 text-slate-600 dark:text-slate-300"
                                />
                            </div>
                        )}
                        <input
                            id="registry-token"
                            name="registry_access_token"
                            type="password"
                            value={formData.token}
                            onChange={(e) => handleChange('token', e.target.value)}
                            onFocus={() => setTokenInputLocked(false)}
                            readOnly={tokenInputLocked}
                            autoComplete="section-registry no-autofill"
                            data-lpignore="true"
                            data-1p-ignore="true"
                            placeholder={isEditMode ? 'Enter new token to rotate' : undefined}
                            className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-slate-700 dark:text-white"
                        />
                    </div>
                )}

                {formData.auth_type === 'dockerconfigjson' && (
                    <div>
                        <label htmlFor="registry-dockerconfigjson" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            Docker Config JSON *
                        </label>
                        {isEditMode && (
                            <div className="mb-2">
                                <textarea
                                    rows={2}
                                    value="*****"
                                    readOnly
                                    aria-label="Stored docker config json"
                                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-slate-50 dark:bg-slate-800 text-slate-600 dark:text-slate-300 font-mono text-xs"
                                />
                            </div>
                        )}
                        <textarea
                            id="registry-dockerconfigjson"
                            rows={6}
                            value={formData.dockerconfigjson}
                            onChange={(e) => handleChange('dockerconfigjson', e.target.value)}
                            className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-xs dark:bg-slate-700 dark:text-white"
                            placeholder={isEditMode ? 'Paste new Docker config JSON to rotate' : '{"auths": {"ghcr.io": {"auth": "..."}}}'}
                        />
                    </div>
                )}

                <div className="flex items-center">
                    <input
                        id="registry-default"
                        type="checkbox"
                        checked={formData.is_default}
                        onChange={(e) => handleChange('is_default', e.target.checked)}
                        className="h-4 w-4 text-blue-600 border-slate-300 rounded"
                    />
                    <label htmlFor="registry-default" className="ml-2 text-sm text-slate-700 dark:text-slate-300">
                        Mark as default for this {scope}
                    </label>
                </div>

                <div className="flex justify-end space-x-3 pt-4 border-t border-slate-200 dark:border-slate-700">
                    <button
                        type="button"
                        onClick={onCancel}
                        disabled={loading}
                        className="px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-slate-700"
                    >
                        Cancel
                    </button>
                    <button
                        type="submit"
                        disabled={loading}
                        className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-60"
                    >
                        {loading
                            ? (isEditMode ? 'Saving...' : 'Creating...')
                            : (isEditMode ? 'Save Changes' : 'Create Authentication')}
                    </button>
                </div>
            </form>
        </div>
    )
}

export default RegistryAuthForm
