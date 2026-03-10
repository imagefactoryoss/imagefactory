import { registryAuthClient } from '@/api/registryAuthClient'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import { RegistryAuth } from '@/types/registryAuth'
import { Copy } from 'lucide-react'
import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'

interface RegistryAuthListProps {
    projectId?: string
    includeTenant?: boolean
    searchQuery?: string
    refreshKey?: number
    onEdit?: (auth: RegistryAuth) => void
}

const RegistryAuthList: React.FC<RegistryAuthListProps> = ({
    projectId,
    includeTenant = true,
    searchQuery = '',
    refreshKey,
    onEdit,
}) => {
    const [auths, setAuths] = useState<RegistryAuth[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [deletingId, setDeletingId] = useState<string | null>(null)
    const confirmDialog = useConfirmDialog()

    useEffect(() => {
        loadRegistryAuth()
    }, [projectId, includeTenant, refreshKey])

    const loadRegistryAuth = async () => {
        try {
            setLoading(true)
            setError(null)
            const response = await registryAuthClient.listRegistryAuth(projectId, includeTenant)
            setAuths(response.registry_auth || [])
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to load registry authentications'
            setError(message)
            toast.error(message)
        } finally {
            setLoading(false)
        }
    }

    const filteredAuths = useMemo(() => {
        const query = searchQuery.trim().toLowerCase()
        if (!query) return auths

        return auths.filter((auth) =>
            auth.name.toLowerCase().includes(query) ||
            (auth.description || '').toLowerCase().includes(query) ||
            auth.registry_host.toLowerCase().includes(query) ||
            auth.registry_type.toLowerCase().includes(query) ||
            auth.auth_type.toLowerCase().includes(query)
        )
    }, [auths, searchQuery])

    const handleDelete = async (auth: RegistryAuth) => {
        const confirmed = await confirmDialog({
            title: 'Delete Registry Auth',
            message: `Delete registry auth "${auth.name}"? This can impact builds using this credential.`,
            confirmLabel: 'Delete Auth',
            destructive: true,
        })
        if (!confirmed) {
            return
        }

        try {
            setDeletingId(auth.id)
            await registryAuthClient.deleteRegistryAuth(auth.id)
            toast.success('Registry authentication deleted')
            await loadRegistryAuth()
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to delete registry authentication'
            toast.error(message)
        } finally {
            setDeletingId(null)
        }
    }

    const formatDate = (dateString: string): string => {
        return new Date(dateString).toLocaleDateString()
    }

    const handleCopyId = async (id: string) => {
        try {
            await navigator.clipboard.writeText(id)
            toast.success('Registry auth UUID copied')
        } catch {
            toast.error('Failed to copy UUID')
        }
    }

    const scopeColor = (scope: RegistryAuth['scope']) => {
        if (scope === 'tenant') return 'bg-violet-100 text-violet-800 dark:bg-violet-900 dark:text-violet-200'
        return 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-200'
    }

    const authTypeColor = (type: RegistryAuth['auth_type']) => {
        if (type === 'basic_auth') return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
        if (type === 'dockerconfigjson') return 'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200'
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
    }

    if (loading) {
        return (
            <div className="flex justify-center items-center py-8">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
            </div>
        )
    }

    if (error) {
        return (
            <div className="text-center py-8">
                <div className="text-red-600 dark:text-red-400 text-sm">{error}</div>
                <button
                    onClick={loadRegistryAuth}
                    className="mt-2 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 text-sm"
                >
                    Retry
                </button>
            </div>
        )
    }

    if (filteredAuths.length === 0) {
        return (
            <div className="text-center py-12">
                <div className="text-slate-500 dark:text-slate-400">
                    <svg className="mx-auto h-12 w-12 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7h16M4 12h16M4 17h16" />
                    </svg>
                    <h3 className="mt-2 text-sm font-medium text-slate-900 dark:text-white">
                        {searchQuery ? 'No Matching Registry Authentications' : 'No Registry Authentications'}
                    </h3>
                    <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
                        {searchQuery ? 'Try a different search term.' : 'Add a registry auth to enable image push/pull in build methods.'}
                    </p>
                </div>
            </div>
        )
    }

    return (
        <div className="space-y-4">
            {filteredAuths.map((auth) => (
                <div
                    key={auth.id}
                    className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6 shadow-sm"
                >
                    <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
                        <div>
                            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">{auth.name}</h3>
                            {auth.description ? (
                                <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">{auth.description}</p>
                            ) : null}
                            <p className="text-sm text-slate-600 dark:text-slate-400 mt-2">
                                Host: <span className="font-mono text-xs">{auth.registry_host}</span>
                            </p>
                            <div className="text-sm text-slate-600 dark:text-slate-400 mt-1 flex items-center gap-2">
                                <span>UUID: <span className="font-mono text-xs break-all">{auth.id}</span></span>
                                <button
                                    type="button"
                                    onClick={() => handleCopyId(auth.id)}
                                    className="inline-flex items-center justify-center rounded border border-slate-300 dark:border-slate-600 p-1 text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700"
                                    title="Copy UUID"
                                    aria-label="Copy registry auth UUID"
                                >
                                    <Copy className="h-3.5 w-3.5" />
                                </button>
                            </div>
                        </div>

                        <div className="flex items-center gap-2 flex-wrap md:justify-end">
                            <span className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium ${scopeColor(auth.scope)}`}>
                                {auth.scope === 'tenant' ? 'Tenant scope' : 'Project scope'}
                            </span>
                            <span className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium ${authTypeColor(auth.auth_type)}`}>
                                {auth.auth_type}
                            </span>
                            <span className="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-slate-100 text-slate-800 dark:bg-slate-900 dark:text-slate-200">
                                {auth.registry_type}
                            </span>
                            {auth.is_default ? (
                                <span className="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-200">
                                    Default
                                </span>
                            ) : null}
                            {onEdit && (
                                <button
                                    onClick={() => onEdit(auth)}
                                    className="inline-flex items-center px-3 py-1 border border-blue-300 dark:border-blue-700 rounded-md text-sm font-medium text-blue-700 dark:text-blue-300 bg-white dark:bg-slate-800 hover:bg-blue-50 dark:hover:bg-blue-900/20"
                                >
                                    Edit Auth
                                </button>
                            )}
                            <button
                                onClick={() => handleDelete(auth)}
                                disabled={deletingId === auth.id}
                                className="inline-flex items-center px-3 py-1 border border-red-300 dark:border-red-700 rounded-md text-sm font-medium text-red-700 dark:text-red-300 bg-white dark:bg-slate-800 hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-50"
                            >
                                {deletingId === auth.id ? 'Deleting...' : 'Delete'}
                            </button>
                        </div>
                    </div>

                    <div className="mt-4 grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
                        <div>
                            <span className="text-slate-500 dark:text-slate-400">Created:</span>
                            <span className="ml-2 text-slate-900 dark:text-white">{formatDate(auth.created_at)}</span>
                        </div>
                        <div>
                            <span className="text-slate-500 dark:text-slate-400">Updated:</span>
                            <span className="ml-2 text-slate-900 dark:text-white">{formatDate(auth.updated_at)}</span>
                        </div>
                        <div>
                            <span className="text-slate-500 dark:text-slate-400">Status:</span>
                            <span className="ml-2 text-slate-900 dark:text-white">{auth.is_active ? 'Active' : 'Inactive'}</span>
                        </div>
                    </div>
                </div>
            ))}
        </div>
    )
}

export default RegistryAuthList
