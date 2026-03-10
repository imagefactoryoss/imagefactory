import { repositoryAuthClient } from '@/api/repositoryAuthClient'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import { RepositoryAuth, RepositoryAuthCreate, RepositoryAuthType, RepositoryAuthUpdate } from '@/types/repositoryAuth'
import { Copy } from 'lucide-react'
import React, { useState } from 'react'
import toast from 'react-hot-toast'

interface RepositoryAuthFormProps {
    projectId: string
    initialAuth?: RepositoryAuth
    onSuccess: () => void
    onCancel: () => void
}

const RepositoryAuthForm: React.FC<RepositoryAuthFormProps> = ({
    projectId,
    initialAuth,
    onSuccess,
    onCancel
}) => {
    const [activeTooltip, setActiveTooltip] = useState<RepositoryAuthType | null>(null)
    const isEdit = Boolean(initialAuth)
    const [changeCredentials, setChangeCredentials] = useState(!isEdit)
    const [formData, setFormData] = useState<RepositoryAuthCreate>({
        project_id: projectId,
        name: initialAuth?.name ?? '',
        description: initialAuth?.description ?? '',
        auth_type: initialAuth?.auth_type ?? RepositoryAuthType.TOKEN,
        username: '',
        ssh_key: '',
        token: '',
        password: '',
    })
    const [loading, setLoading] = useState(false)
    const confirmDialog = useConfirmDialog()

    const handleInputChange = (field: keyof RepositoryAuthCreate, value: string) => {
        setFormData(prev => ({
            ...prev,
            [field]: value
        }))
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()

        const trimmedName = formData.name.trim()

        // Basic validation
        if (!trimmedName) {
            toast.error('Name is required')
            return
        }

        if (!formData.auth_type) {
            toast.error('Authentication type is required')
            return
        }

        const trimmedUsername = formData.username?.trim() ?? ''
        const trimmedSSHKey = formData.ssh_key?.trim() ?? ''
        const trimmedToken = formData.token?.trim() ?? ''
        const trimmedPassword = formData.password?.trim() ?? ''
        const hasCredentialUpdates = Boolean(trimmedSSHKey || trimmedToken || trimmedUsername || trimmedPassword)
        const authTypeChanged = isEdit && formData.auth_type !== initialAuth?.auth_type
        const shouldValidateCredentials = !isEdit || changeCredentials || authTypeChanged || hasCredentialUpdates

        if (shouldValidateCredentials) {
            // Validate based on auth type
            switch (formData.auth_type) {
                case RepositoryAuthType.SSH_KEY:
                    if (!trimmedSSHKey) {
                        toast.error('SSH key is required for SSH authentication')
                        return
                    }
                    break
                case RepositoryAuthType.TOKEN:
                    if (!trimmedToken) {
                        toast.error('Token is required for token authentication')
                        return
                    }
                    break
                case RepositoryAuthType.BASIC_AUTH:
                    if (!trimmedUsername || !trimmedPassword) {
                        toast.error('Username and password are required for basic authentication')
                        return
                    }
                    break
            }
        }

        try {
            setLoading(true)
            if (isEdit && initialAuth) {
                if (authTypeChanged) {
                    const confirmed = await confirmDialog({
                        title: 'Change Authentication Type',
                        message: 'You are changing the authentication type. This will replace the stored credentials for this auth. Do you want to continue?',
                        confirmLabel: 'Continue',
                        destructive: true,
                    })
                    if (!confirmed) {
                        setLoading(false)
                        return
                    }
                }
                const payload: RepositoryAuthUpdate = {}
                if (trimmedName !== initialAuth.name) {
                    payload.name = trimmedName
                }
                if ((formData.description ?? '') !== (initialAuth.description ?? '')) {
                    payload.description = formData.description ?? ''
                }
                if (changeCredentials || authTypeChanged || hasCredentialUpdates) {
                    payload.auth_type = formData.auth_type
                }
                if (trimmedSSHKey) {
                    payload.ssh_key = trimmedSSHKey
                }
                if (trimmedToken) {
                    payload.token = trimmedToken
                }
                if (trimmedUsername) {
                    payload.username = trimmedUsername
                }
                if (trimmedPassword) {
                    payload.password = trimmedPassword
                }

                if (Object.keys(payload).length === 0) {
                    toast.error('No changes to update')
                    return
                }

                await repositoryAuthClient.updateRepositoryAuth(projectId, initialAuth.id, payload)
                toast.success('Repository authentication updated successfully')
            } else {
                await repositoryAuthClient.createRepositoryAuth(projectId, {
                    ...formData,
                    name: trimmedName,
                })
                toast.success('Repository authentication created successfully')
            }
            onSuccess()
        } catch (error) {
            const errorMessage = error instanceof Error
                ? error.message
                : isEdit
                    ? 'Failed to update repository authentication'
                    : 'Failed to create repository authentication'
            toast.error(errorMessage)
        } finally {
            setLoading(false)
        }
    }

    const handleCopyTooltip = async (authType: RepositoryAuthType) => {
        const text = getAuthTypeTooltip(authType)
        try {
            await navigator.clipboard.writeText(text)
            toast.success('Copied tip to clipboard')
        } catch (error) {
            toast.error('Failed to copy tip')
        }
    }

    const getAuthTypeDescription = (authType: RepositoryAuthType): string => {
        switch (authType) {
            case RepositoryAuthType.SSH_KEY:
                return 'Use SSH key authentication for Git repositories'
            case RepositoryAuthType.TOKEN:
                return 'Use personal access token or API token for authentication'
            case RepositoryAuthType.BASIC_AUTH:
                return 'Use username and password for basic authentication'
            case RepositoryAuthType.OAUTH:
                return 'Use OAuth for authentication (configured separately)'
            default:
                return ''
        }
    }

    const getAuthTypeTooltip = (authType: RepositoryAuthType): string => {
        switch (authType) {
            case RepositoryAuthType.SSH_KEY:
                return 'Generate a new SSH key (ed25519 recommended) and paste the private key here. Add the public key to your Git provider.'
            case RepositoryAuthType.TOKEN:
                return 'Create a personal access token with repo read access. Suggested scopes: GitHub (classic) repo:read (or repo for private) and read:org if needed; GitLab read_repository; Bitbucket read:repository. Example (GitHub): Settings → Developer settings → Personal access tokens.'
            case RepositoryAuthType.BASIC_AUTH:
                return 'Use your Git username with a password or app password (GitHub/GitLab may require tokens instead).'
            case RepositoryAuthType.OAUTH:
                return 'OAuth requires provider configuration by an admin. Use this only if OAuth is already set up.'
            default:
                return ''
        }
    }

    return (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow border border-slate-200 dark:border-slate-700 p-6">
            <div className="mb-6">
                <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                    {isEdit ? 'Edit Repository Authentication' : 'Add Repository Authentication'}
                </h3>
                <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                    {isEdit
                        ? 'Update authentication details for accessing your Git repository'
                        : 'Configure authentication method for accessing your Git repository'}
                </p>
            </div>

            <form onSubmit={handleSubmit} className="space-y-6">
                {/* Name */}
                <div>
                    <label htmlFor="repository-auth-name" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        Name *
                    </label>
                    <input
                        type="text"
                        id="repository-auth-name"
                        value={formData.name}
                        onChange={(e) => handleInputChange('name', e.target.value)}
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                        placeholder="e.g., GitHub Token, SSH Key"
                        required
                    />
                </div>

                {/* Description */}
                <div>
                    <label htmlFor="description" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        Description
                    </label>
                    <textarea
                        id="description"
                        value={formData.description}
                        onChange={(e) => handleInputChange('description', e.target.value)}
                        rows={3}
                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                        placeholder="Optional description for this authentication method"
                    />
                </div>

                {/* Auth Type */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        Authentication Type *
                    </label>
                    <div className="space-y-2">
                        {Object.values(RepositoryAuthType).map((authType) => (
                            <div key={authType} className="flex items-start gap-2">
                                <input
                                    type="radio"
                                    id={authType}
                                    name="auth_type"
                                    value={authType}
                                    checked={formData.auth_type === authType}
                                    onChange={(e) => handleInputChange('auth_type', e.target.value as RepositoryAuthType)}
                                    className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-slate-300 dark:border-slate-600"
                                    disabled={isEdit && !changeCredentials}
                                />
                                <label htmlFor={authType} className="ml-3 block text-sm">
                                    <span className="font-medium text-slate-900 dark:text-white capitalize">
                                        {authType.replace('_', ' ')}
                                    </span>
                                    <span className="block text-slate-600 dark:text-slate-400 text-xs">
                                        {getAuthTypeDescription(authType)}
                                    </span>
                                </label>
                                <div className="relative">
                                    <button
                                        type="button"
                                        onClick={() => setActiveTooltip(prev => prev === authType ? null : authType)}
                                        className="inline-flex h-5 w-5 items-center justify-center rounded-full border border-slate-300 dark:border-slate-600 text-xs text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700"
                                        aria-label={`Help for ${authType.replace('_', ' ')} authentication`}
                                    >
                                        i
                                    </button>
                                    {activeTooltip === authType && (
                                        <div className="absolute z-10 mt-2 w-72 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-3 text-xs text-slate-700 dark:text-slate-200 shadow-lg">
                                            <div className="flex items-start justify-between gap-2">
                                                <div className="text-xs leading-relaxed">
                                                    {getAuthTypeTooltip(authType)}
                                                </div>
                                                <button
                                                    type="button"
                                                    onClick={() => handleCopyTooltip(authType)}
                                                    className="inline-flex items-center gap-1 rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 px-2 py-1 text-[11px] text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800"
                                                >
                                                    <Copy className="h-3 w-3" />
                                                    Copy
                                                </button>
                                            </div>
                                        </div>
                                    )}
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
                {isEdit && (
                    <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-3">
                        <label className="flex items-start gap-2 text-sm text-slate-700 dark:text-slate-200">
                            <input
                                type="checkbox"
                                className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-slate-300 dark:border-slate-600"
                                checked={changeCredentials}
                                onChange={(e) => setChangeCredentials(e.target.checked)}
                            />
                            <span>
                                Change credentials
                                <span className="block text-xs text-slate-500 dark:text-slate-400 mt-1">
                                    Keep this off to update only name or description. Turn on to update auth type or secrets.
                                </span>
                                <span className="block text-xs text-slate-500 dark:text-slate-400 mt-1">
                                    This is a safety toggle to prevent accidental secret rotation.
                                </span>
                            </span>
                        </label>
                        {changeCredentials && (
                            <div className="mt-3 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-500/40 dark:bg-amber-900/20 dark:text-amber-200">
                                Changing auth type or credentials will replace the stored secret for this auth. Existing connections using this auth may fail until updated.
                            </div>
                        )}
                    </div>
                )}

                {/* Dynamic Fields based on Auth Type */}
                {changeCredentials && formData.auth_type === RepositoryAuthType.SSH_KEY && (
                    <div>
                        <label htmlFor="ssh_key" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            SSH Private Key *
                        </label>
                        <textarea
                            id="ssh_key"
                            value={formData.ssh_key}
                            onChange={(e) => handleInputChange('ssh_key', e.target.value)}
                            rows={6}
                            className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white font-mono text-sm"
                            placeholder={isEdit ? 'Paste a new key to replace the existing one' : '-----BEGIN OPENSSH PRIVATE KEY-----\n...'}
                            required
                        />
                        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                            Paste your SSH private key. It will be encrypted and stored securely.
                        </p>
                    </div>
                )}

                {changeCredentials && formData.auth_type === RepositoryAuthType.TOKEN && (
                    <div>
                        <label htmlFor="token" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            Access Token *
                        </label>
                        <input
                            type="password"
                            id="token"
                            value={formData.token}
                            onChange={(e) => handleInputChange('token', e.target.value)}
                            className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white font-mono"
                            placeholder={isEdit ? 'Paste a new token to replace the existing one' : 'ghp_xxxxxxxxxxxxxxxxxxxx'}
                            required
                        />
                        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                            Personal access token or API token for repository access.
                        </p>
                    </div>
                )}

                {changeCredentials && formData.auth_type === RepositoryAuthType.BASIC_AUTH && (
                    <div className="space-y-4">
                        <div>
                            <label htmlFor="username" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                Username *
                            </label>
                            <input
                                type="text"
                                id="username"
                                value={formData.username}
                                onChange={(e) => handleInputChange('username', e.target.value)}
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                                placeholder={isEdit ? 'Enter new username' : 'username'}
                                required
                            />
                        </div>
                        <div>
                            <label htmlFor="password" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                                Password *
                            </label>
                            <input
                                type="password"
                                id="password"
                                value={formData.password}
                                onChange={(e) => handleInputChange('password', e.target.value)}
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                                placeholder={isEdit ? 'Enter new password' : 'password'}
                                required
                            />
                        </div>
                    </div>
                )}

                {/* Form Actions */}
                <div className="flex justify-end space-x-3 pt-4 border-t border-slate-200 dark:border-slate-700">
                    <button
                        type="button"
                        onClick={onCancel}
                        className="px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm text-sm font-medium text-slate-700 dark:text-slate-300 bg-white dark:bg-slate-800 hover:bg-slate-50 dark:hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                        disabled={loading}
                    >
                        Cancel
                    </button>
                    <button
                        type="submit"
                        className="px-4 py-2 bg-blue-600 border border-transparent rounded-md shadow-sm text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                        disabled={loading}
                    >
                        {loading ? (
                            <div className="flex items-center">
                                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                                {isEdit ? 'Updating...' : 'Creating...'}
                            </div>
                        ) : (
                            isEdit ? 'Update Authentication' : 'Create Authentication'
                        )}
                    </button>
                </div>
            </form>
        </div>
    )
}

export default RepositoryAuthForm
