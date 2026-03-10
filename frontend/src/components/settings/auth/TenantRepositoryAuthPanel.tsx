import { repositoryAuthClient } from '@/api/repositoryAuthClient'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import Drawer from '@/components/ui/Drawer'
import { RepositoryAuth, RepositoryAuthType } from '@/types/repositoryAuth'
import { getErrorMessage } from '@/services/api'
import axios from 'axios'
import { Copy } from 'lucide-react'
import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'

type FormState = {
    name: string
    description: string
    auth_type: RepositoryAuthType
    username: string
    password: string
    token: string
    ssh_key: string
}

type ScopeFilter = 'tenant' | 'project'

interface TenantRepositoryAuthPanelProps {
    scope: ScopeFilter
    selectedProjectId?: string
    searchQuery?: string
}

const emptyForm: FormState = {
    name: '',
    description: '',
    auth_type: RepositoryAuthType.TOKEN,
    username: '',
    password: '',
    token: '',
    ssh_key: '',
}

type DeleteInUseErrorPayload = {
    details?: {
        active_project_names?: string[]
    }
}

const TenantRepositoryAuthPanel: React.FC<TenantRepositoryAuthPanelProps> = ({
    scope,
    selectedProjectId,
    searchQuery = '',
}) => {
    const [auths, setAuths] = useState<RepositoryAuth[]>([])
    const [loading, setLoading] = useState(true)
    const [submitting, setSubmitting] = useState(false)
    const [deletingId, setDeletingId] = useState<string | null>(null)
    const [editingAuthId, setEditingAuthId] = useState<string | null>(null)
    const [showForm, setShowForm] = useState(false)
    const [form, setForm] = useState<FormState>(emptyForm)
    const confirmDialog = useConfirmDialog()

    const loadAuths = async () => {
        try {
            setLoading(true)
            if (scope === 'project') {
                if (!selectedProjectId) {
                    setAuths([])
                    return
                }
                const response = await repositoryAuthClient.listScopedRepositoryAuth(selectedProjectId, false)
                setAuths(response.repository_auths || [])
                return
            }

            const response = await repositoryAuthClient.listScopedRepositoryAuth()
            setAuths(response.repository_auths || [])
        } catch (error) {
            toast.error(error instanceof Error ? error.message : 'Failed to load repository auth')
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        loadAuths()
    }, [scope, selectedProjectId])

    const filteredAuths = useMemo(() => {
        const query = searchQuery.trim().toLowerCase()
        if (!query) return auths
        return auths.filter((auth) =>
            auth.name.toLowerCase().includes(query) ||
            (auth.description || '').toLowerCase().includes(query) ||
            auth.auth_type.toLowerCase().includes(query)
        )
    }, [auths, searchQuery])

    const buildPayload = () => {
        if (!form.name.trim()) {
            toast.error('Name is required')
            return null
        }

        const isEditing = Boolean(editingAuthId)
        const payload: any = {
            name: form.name.trim(),
            description: form.description.trim(),
        }
        if (!isEditing) {
            payload.auth_type = form.auth_type
            payload.project_id = scope === 'project' ? selectedProjectId : undefined
        }

        if (!isEditing && scope === 'project' && !selectedProjectId) {
            toast.error('Select a project first')
            return null
        }

        if (form.auth_type === RepositoryAuthType.BASIC_AUTH) {
            const username = form.username.trim()
            const password = form.password.trim()
            if (!isEditing && (!username || !password)) {
                toast.error('Username and password are required for basic auth')
                return null
            }
            if (isEditing && (username || password) && (!username || !password)) {
                toast.error('Provide both username and password to rotate basic auth credentials')
                return null
            }
            if (username && password) {
                payload.username = username
                payload.password = password
            }
        }

        if (form.auth_type === RepositoryAuthType.TOKEN) {
            const token = form.token.trim()
            if (!isEditing && !token) {
                toast.error('Token is required for token auth')
                return null
            }
            if (token) {
                payload.token = token
            }
        }

        if (form.auth_type === RepositoryAuthType.SSH_KEY) {
            const sshKey = form.ssh_key.trim()
            if (!isEditing && !sshKey) {
                toast.error('SSH key is required for SSH auth')
                return null
            }
            if (sshKey) {
                payload.ssh_key = sshKey
            }
        }

        return payload
    }

    const handleCreate = async (e: React.FormEvent) => {
        e.preventDefault()
        const payload = buildPayload()
        if (!payload) return

        try {
            setSubmitting(true)
            if (editingAuthId) {
                await repositoryAuthClient.updateScopedRepositoryAuth(editingAuthId, payload)
                toast.success('Repository auth updated')
            } else {
                await repositoryAuthClient.createScopedRepositoryAuth(payload)
                toast.success(`${scope === 'tenant' ? 'Tenant' : 'Project'} repository auth created`)
            }
            setForm(emptyForm)
            setEditingAuthId(null)
            setShowForm(false)
            await loadAuths()
        } catch (error) {
            toast.error(error instanceof Error ? error.message : `Failed to ${editingAuthId ? 'update' : 'create'} repository auth`)
        } finally {
            setSubmitting(false)
        }
    }

    const handleEdit = (auth: RepositoryAuth) => {
        setEditingAuthId(auth.id)
        setShowForm(true)
        setForm({
            name: auth.name,
            description: auth.description || '',
            auth_type: auth.auth_type,
            username: '',
            password: '',
            token: '',
            ssh_key: '',
        })
    }

    const cancelEdit = () => {
        setEditingAuthId(null)
        setShowForm(false)
        setForm(emptyForm)
    }

    const startCreate = () => {
        setEditingAuthId(null)
        setForm(emptyForm)
        setShowForm(true)
    }

    const handleDelete = async (auth: RepositoryAuth) => {
        const confirmMessage = scope === 'project'
            ? `Delete project-scoped repository auth "${auth.name}"?\n\nThis will remove credentials used by this project.`
            : `Delete tenant-scoped repository auth "${auth.name}"?`
        const confirmed = await confirmDialog({
            title: 'Delete Repository Auth',
            message: confirmMessage,
            confirmLabel: 'Delete Auth',
            destructive: true,
        })
        if (!confirmed) return

        try {
            setDeletingId(auth.id)
            await repositoryAuthClient.deleteScopedRepositoryAuth(auth.id)
            toast.success('Repository auth deleted')
            await loadAuths()
        } catch (error) {
            if (axios.isAxiosError<DeleteInUseErrorPayload>(error) && error.response?.status === 409) {
                const projectNames = error.response.data?.details?.active_project_names || []
                const message = projectNames.length > 0
                    ? `Cannot delete "${auth.name}". It is still used by active projects: ${projectNames.join(', ')}. Remove it from those projects first.`
                    : `Cannot delete "${auth.name}". It is still used by active projects. Remove it from projects first.`
                toast.error(message, { duration: 7000 })
            } else {
                toast.error(getErrorMessage(error))
            }
        } finally {
            setDeletingId(null)
        }
    }

    const handleCopyId = async (id: string) => {
        try {
            await navigator.clipboard.writeText(id)
            toast.success('Repository auth UUID copied')
        } catch {
            toast.error('Failed to copy UUID')
        }
    }

    return (
        <div className="space-y-6">
            <Drawer
                isOpen={showForm}
                onClose={cancelEdit}
                title={editingAuthId ? 'Edit Repository Auth' : `Add ${scope === 'tenant' ? 'Tenant' : 'Project'} Repository Auth`}
                description={editingAuthId
                    ? 'Update name/description or provide new credentials for this auth type.'
                    : scope === 'tenant'
                        ? 'Tenant-scoped credentials can be reused across projects.'
                        : 'Project-scoped credentials are specific to one project.'}
                width="xl"
            >
                <form className="space-y-4" onSubmit={handleCreate}>
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Name *</label>
                                <input
                                    value={form.name}
                                    onChange={(e) => setForm(prev => ({ ...prev, name: e.target.value }))}
                                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md dark:bg-slate-700 dark:text-white"
                                    placeholder="e.g. Shared GitHub Token"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Auth Type *</label>
                                <select
                                    value={form.auth_type}
                                    onChange={(e) => setForm(prev => ({ ...prev, auth_type: e.target.value as RepositoryAuthType }))}
                                    disabled={Boolean(editingAuthId)}
                                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md dark:bg-slate-700 dark:text-white disabled:opacity-60"
                                >
                                    <option value={RepositoryAuthType.TOKEN}>Token</option>
                                    <option value={RepositoryAuthType.BASIC_AUTH}>Basic Auth</option>
                                    <option value={RepositoryAuthType.SSH_KEY}>SSH Key</option>
                                </select>
                            </div>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Description</label>
                            <input
                                value={form.description}
                                onChange={(e) => setForm(prev => ({ ...prev, description: e.target.value }))}
                                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md dark:bg-slate-700 dark:text-white"
                                placeholder="Optional"
                            />
                        </div>

                        {form.auth_type === RepositoryAuthType.TOKEN && (
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Token *</label>
                                <input
                                    type="password"
                                    value={form.token}
                                    onChange={(e) => setForm(prev => ({ ...prev, token: e.target.value }))}
                                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md dark:bg-slate-700 dark:text-white"
                                />
                            </div>
                        )}

                        {form.auth_type === RepositoryAuthType.BASIC_AUTH && (
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Username *</label>
                                    <input
                                        value={form.username}
                                        onChange={(e) => setForm(prev => ({ ...prev, username: e.target.value }))}
                                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md dark:bg-slate-700 dark:text-white"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Password *</label>
                                    <input
                                        type="password"
                                        value={form.password}
                                        onChange={(e) => setForm(prev => ({ ...prev, password: e.target.value }))}
                                        className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md dark:bg-slate-700 dark:text-white"
                                    />
                                </div>
                            </div>
                        )}

                        {form.auth_type === RepositoryAuthType.SSH_KEY && (
                            <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">SSH Key *</label>
                                <textarea
                                    rows={5}
                                    value={form.ssh_key}
                                    onChange={(e) => setForm(prev => ({ ...prev, ssh_key: e.target.value }))}
                                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md font-mono text-xs dark:bg-slate-700 dark:text-white"
                                />
                            </div>
                        )}

                        <div className="flex justify-end gap-2">
                            <button
                                type="button"
                                onClick={cancelEdit}
                                className="px-4 py-2 border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-200 rounded-md hover:bg-slate-50 dark:hover:bg-slate-700"
                            >
                                Cancel
                            </button>
                            <button
                                type="submit"
                                disabled={submitting || (scope === 'project' && !selectedProjectId)}
                                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-60"
                            >
                                {submitting
                                    ? (editingAuthId ? 'Saving...' : 'Creating...')
                                    : (editingAuthId ? 'Save Changes' : `Create ${scope === 'tenant' ? 'Tenant' : 'Project'} Auth`)}
                            </button>
                        </div>
                </form>
            </Drawer>

            <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
                <div className="flex items-center justify-between gap-3">
                    <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                        {scope === 'tenant' ? 'Tenant Repository Auth' : 'Project Repository Auth'}
                    </h3>
                    <button
                        type="button"
                        onClick={startCreate}
                        disabled={scope === 'project' && !selectedProjectId}
                        className="px-3 py-1.5 bg-blue-600 text-white rounded-md hover:bg-blue-700 text-sm disabled:opacity-60"
                    >
                        Add Repository Auth
                    </button>
                </div>
                {loading ? (
                    <div className="py-6 text-sm text-slate-500 dark:text-slate-400">Loading...</div>
                ) : scope === 'project' && !selectedProjectId ? (
                    <div className="py-6 text-sm text-slate-500 dark:text-slate-400">
                        Select a project to view and manage project-scoped repository auth.
                    </div>
                ) : filteredAuths.length === 0 ? (
                    <div className="py-6 text-sm text-slate-500 dark:text-slate-400">
                        {searchQuery ? 'No matching repository auth found.' : `No ${scope}-scoped repository auth configured yet.`}
                    </div>
                ) : (
                    <div className="mt-4 space-y-3">
                        {filteredAuths.map((auth) => (
                            <div key={auth.id} className="border border-slate-200 dark:border-slate-700 rounded-md p-4 flex items-center justify-between">
                                <div>
                                    <div className="font-medium text-slate-900 dark:text-white">{auth.name}</div>
                                    <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">{auth.auth_type}</div>
                                    <div className="mt-1 text-xs text-slate-500 dark:text-slate-400 flex items-center gap-2">
                                        <span>UUID: <span className="font-mono break-all">{auth.id}</span></span>
                                        <button
                                            type="button"
                                            onClick={() => handleCopyId(auth.id)}
                                            className="inline-flex items-center justify-center rounded border border-slate-300 dark:border-slate-600 p-1 text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700"
                                            title="Copy UUID"
                                            aria-label="Copy repository auth UUID"
                                        >
                                            <Copy className="h-3.5 w-3.5" />
                                        </button>
                                    </div>
                                </div>
                                <div className="flex items-center gap-2">
                                    <button
                                        onClick={() => handleEdit(auth)}
                                        className="px-3 py-1 border border-blue-300 dark:border-blue-700 rounded-md text-sm text-blue-700 dark:text-blue-300 hover:bg-blue-50 dark:hover:bg-blue-900/20"
                                    >
                                        Edit
                                    </button>
                                    <button
                                        onClick={() => handleDelete(auth)}
                                        disabled={deletingId === auth.id}
                                        className="px-3 py-1 border border-red-300 dark:border-red-700 rounded-md text-sm text-red-700 dark:text-red-300 hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-50"
                                    >
                                        {deletingId === auth.id ? 'Deleting...' : 'Delete'}
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    )
}

export default TenantRepositoryAuthPanel
