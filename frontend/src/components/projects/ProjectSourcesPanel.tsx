import Drawer from '@/components/ui/Drawer'
import HelpTooltip from '@/components/common/HelpTooltip'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import { gitProviderClient } from '@/api/gitProviderClient'
import { repositoryAuthClient } from '@/api/repositoryAuthClient'
import { repositoryBranchClient } from '@/api/repositoryBranchClient'
import RepositoryAuthModal from '@/components/projects/repository-auth/RepositoryAuthModal'
import { projectService } from '@/services/projectService'
import { CreateProjectSourceRequest, ProjectSource } from '@/types'
import { GitProvider } from '@/types/gitProvider'
import { RepositoryAuth } from '@/types/repositoryAuth'
import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'

interface ProjectSourcesPanelProps {
    projectId: string
    canEdit: boolean
}

const emptyForm: CreateProjectSourceRequest = {
    name: '',
    provider: 'generic',
    repositoryUrl: '',
    defaultBranch: 'main',
    isDefault: false,
    isActive: true,
}

const normalizeSourceKeyPart = (value?: string): string => (value || '').trim().toLowerCase()

const ProjectSourcesPanel: React.FC<ProjectSourcesPanelProps> = ({ projectId, canEdit }) => {
    const [sources, setSources] = useState<ProjectSource[]>([])
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [drawerOpen, setDrawerOpen] = useState(false)
    const [showAuthModal, setShowAuthModal] = useState(false)
    const [authRefreshKey, setAuthRefreshKey] = useState(0)
    const [repositoryAuths, setRepositoryAuths] = useState<RepositoryAuth[]>([])
    const [authsLoading, setAuthsLoading] = useState(false)
    const [gitProviders, setGitProviders] = useState<GitProvider[]>([])
    const [providersLoading, setProvidersLoading] = useState(false)
    const [branches, setBranches] = useState<string[]>([])
    const [branchesLoading, setBranchesLoading] = useState(false)
    const [branchesError, setBranchesError] = useState<string | null>(null)
    const [editing, setEditing] = useState<ProjectSource | null>(null)
    const [form, setForm] = useState<CreateProjectSourceRequest>(emptyForm)
    const confirmDialog = useConfirmDialog()

    const hasSources = useMemo(() => sources.length > 0, [sources.length])

    const loadSources = async () => {
        try {
            setLoading(true)
            const data = await projectService.listProjectSources(projectId)
            setSources(data)
        } catch (error: any) {
            toast.error(error?.message || 'Failed to load sources')
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        loadSources()
    }, [projectId])

    const loadRepositoryAuths = async () => {
        try {
            setAuthsLoading(true)
            const response = await repositoryAuthClient.getRepositoryAuths(projectId, true)
            setRepositoryAuths(response.repository_auths || [])
        } catch (error: any) {
            toast.error(error?.message || 'Failed to load repository auth')
        } finally {
            setAuthsLoading(false)
        }
    }

    useEffect(() => {
        loadRepositoryAuths()
    }, [projectId, authRefreshKey])

    const loadGitProviders = async () => {
        try {
            setProvidersLoading(true)
            const response = await gitProviderClient.getGitProviders()
            setGitProviders(response.providers || [])
        } catch (error: any) {
            toast.error(error?.message || 'Failed to load git providers')
        } finally {
            setProvidersLoading(false)
        }
    }

    useEffect(() => {
        loadGitProviders()
    }, [projectId])

    useEffect(() => {
        setBranches([])
        setBranchesError(null)
    }, [form.repositoryUrl, form.repositoryAuthId, form.provider])

    const resetForm = () => {
        setEditing(null)
        setForm({ ...emptyForm })
    }

    const closeDrawer = () => {
        setDrawerOpen(false)
        resetForm()
    }

    const openCreate = () => {
        resetForm()
        setDrawerOpen(true)
    }

    const handleFetchBranches = async () => {
        if (!form.repositoryUrl?.trim()) {
            setBranchesError('Repository URL is required to load branches')
            return
        }

        try {
            setBranchesLoading(true)
            setBranchesError(null)
            const response = await repositoryBranchClient.listBranches(projectId, {
                repository_url: form.repositoryUrl,
                auth_id: form.repositoryAuthId || undefined,
                provider_key: form.provider || 'generic',
            })
            setBranches(response.branches || [])
            if (!response.branches?.length) {
                setBranchesError('No branches were returned for this repository')
            }
        } catch (error: any) {
            const message = error?.response?.data?.error || error?.message || 'Failed to load branches'
            setBranchesError(message)
        } finally {
            setBranchesLoading(false)
        }
    }

    const onSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!canEdit) return
        if (!form.name?.trim() || !form.repositoryUrl?.trim()) {
            toast.error('Name and repository URL are required')
            return
        }
        const provider = normalizeSourceKeyPart(form.provider || 'generic')
        const repository = normalizeSourceKeyPart(form.repositoryUrl)
        const branch = normalizeSourceKeyPart(form.defaultBranch || 'main')
        const duplicate = sources.some((source) => {
            if (editing && source.id === editing.id) return false
            return (
                normalizeSourceKeyPart(source.provider || 'generic') === provider &&
                normalizeSourceKeyPart(source.repositoryUrl) === repository &&
                normalizeSourceKeyPart(source.defaultBranch || 'main') === branch
            )
        })
        if (duplicate) {
            toast.error('A source with the same provider, repository, and default branch already exists in this project')
            return
        }
        try {
            setSaving(true)
            if (editing) {
                await projectService.updateProjectSource(projectId, editing.id, form)
                toast.success('Source updated')
            } else {
                await projectService.createProjectSource(projectId, form)
                toast.success('Source created')
            }
            closeDrawer()
            await loadSources()
        } catch (error: any) {
            toast.error(error?.message || 'Failed to save source')
        } finally {
            setSaving(false)
        }
    }

    const onEdit = (source: ProjectSource) => {
        setEditing(source)
        setForm({
            name: source.name,
            provider: source.provider,
            repositoryUrl: source.repositoryUrl,
            defaultBranch: source.defaultBranch,
            repositoryAuthId: source.repositoryAuthId,
            isDefault: source.isDefault,
            isActive: source.isActive,
        })
        setDrawerOpen(true)
    }

    const onDelete = async (source: ProjectSource) => {
        if (!canEdit) return
        const confirmed = await confirmDialog({
            title: 'Delete Source',
            message: `Delete source "${source.name}"?`,
            confirmLabel: 'Delete Source',
            destructive: true,
        })
        if (!confirmed) return
        try {
            await projectService.deleteProjectSource(projectId, source.id)
            toast.success('Source deleted')
            await loadSources()
        } catch (error: any) {
            toast.error(error?.message || 'Failed to delete source')
        }
    }

    if (loading) {
        return (
            <div className="rounded-lg border border-slate-200 bg-white p-6 dark:border-slate-700 dark:bg-slate-800">
                <p className="text-sm text-slate-600 dark:text-slate-300">Loading project sources...</p>
            </div>
        )
    }

    return (
        <div className="space-y-6">
            <div className="rounded-lg border border-slate-200 bg-white p-6 dark:border-slate-700 dark:bg-slate-800">
                <div className="mb-4 flex items-center justify-between">
                    <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100">Project Sources</h3>
                    <div className="flex items-center gap-3">
                        <span className="rounded-full bg-blue-100 px-3 py-1 text-xs font-medium text-blue-700 dark:bg-blue-900/40 dark:text-blue-200">
                            {sources.length} configured
                        </span>
                        {canEdit ? (
                            <button
                                type="button"
                                onClick={openCreate}
                                className="rounded-md bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-400"
                            >
                                Add Source
                            </button>
                        ) : null}
                    </div>
                </div>
                <div className="mb-4 rounded-md border border-indigo-200 bg-indigo-50 p-3 dark:border-indigo-700 dark:bg-indigo-900/25">
                    <p className="text-xs font-medium text-indigo-900 dark:text-indigo-200">Repository Auth Precedence</p>
                    <p className="mt-1 text-xs text-indigo-800 dark:text-indigo-300">
                        Build-as-code fetch uses source auth first, then project active repository auth, then anonymous clone.
                    </p>
                </div>

                {!hasSources ? (
                    <p className="text-sm text-slate-600 dark:text-slate-300">No sources configured yet.</p>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
                            <thead className="bg-slate-50 dark:bg-slate-900/50">
                                <tr>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Name</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Provider</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Repository</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Default Branch</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">State</th>
                                    {canEdit ? <th className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Actions</th> : null}
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                                {sources.map((source) => (
                                    <tr key={source.id}>
                                        <td className="px-3 py-3 text-sm font-medium text-slate-900 dark:text-slate-100">
                                            {source.name}
                                            {source.isDefault ? (
                                                <span className="ml-2 rounded bg-emerald-100 px-2 py-0.5 text-xs text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-200">Default</span>
                                            ) : null}
                                        </td>
                                        <td className="px-3 py-3 text-sm text-slate-700 dark:text-slate-300">{source.provider}</td>
                                        <td className="px-3 py-3 text-sm text-slate-700 dark:text-slate-300">
                                            <span className="font-mono break-all">{source.repositoryUrl}</span>
                                        </td>
                                        <td className="px-3 py-3 text-sm text-slate-700 dark:text-slate-300">{source.defaultBranch}</td>
                                        <td className="px-3 py-3 text-sm">
                                            <span className={`rounded-full px-2 py-1 text-xs font-medium ${source.isActive ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-200' : 'bg-slate-200 text-slate-700 dark:bg-slate-700 dark:text-slate-200'}`}>
                                                {source.isActive ? 'Active' : 'Inactive'}
                                            </span>
                                        </td>
                                        {canEdit ? (
                                            <td className="px-3 py-3 text-right">
                                                <button
                                                    onClick={() => onEdit(source)}
                                                    className="mr-2 rounded-md border border-blue-300 px-2 py-1 text-xs text-blue-700 hover:bg-blue-50 dark:border-blue-700 dark:text-blue-200 dark:hover:bg-blue-900/30"
                                                >
                                                    Edit
                                                </button>
                                                <button
                                                    onClick={() => onDelete(source)}
                                                    className="rounded-md border border-red-300 px-2 py-1 text-xs text-red-700 hover:bg-red-50 dark:border-red-700 dark:text-red-200 dark:hover:bg-red-900/30"
                                                >
                                                    Delete
                                                </button>
                                            </td>
                                        ) : null}
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            {canEdit ? (
                <Drawer
                    isOpen={drawerOpen}
                    onClose={closeDrawer}
                    title={editing ? `Edit Source: ${editing.name}` : 'Add Source'}
                    description="Configure repository source details for this project."
                    width="lg"
                >
                <form onSubmit={onSubmit} className="rounded-lg border border-slate-200 bg-white p-6 dark:border-slate-700 dark:bg-slate-800">
                    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                        <label className="block text-sm text-slate-700 dark:text-slate-300">
                            Name
                            <input
                                value={form.name || ''}
                                onChange={(e) => setForm({ ...form, name: e.target.value })}
                                className="mt-1 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100 dark:focus:border-blue-400"
                                placeholder="primary"
                            />
                        </label>
                        <label className="block text-sm text-slate-700 dark:text-slate-300">
                            Provider
                            <select
                                value={form.provider || 'generic'}
                                onChange={(e) => setForm({ ...form, provider: e.target.value })}
                                className="mt-1 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100 dark:focus:border-blue-400"
                            >
                                <option value="generic">Generic</option>
                                {gitProviders.map((provider) => (
                                    <option key={provider.key} value={provider.key}>
                                        {provider.display_name}
                                    </option>
                                ))}
                                {!!form.provider && form.provider !== 'generic' && !gitProviders.some((provider) => provider.key === form.provider) && (
                                    <option value={form.provider}>{form.provider}</option>
                                )}
                            </select>
                            {providersLoading ? (
                                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">Loading providers...</p>
                            ) : null}
                        </label>
                        <label className="block text-sm text-slate-700 dark:text-slate-300 md:col-span-2">
                            Repository URL
                            <input
                                value={form.repositoryUrl || ''}
                                onChange={(e) => setForm({ ...form, repositoryUrl: e.target.value })}
                                className="mt-1 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100 dark:focus:border-blue-400"
                                placeholder="https://github.com/org/repo.git"
                            />
                        </label>
                        <label className="block text-sm text-slate-700 dark:text-slate-300">
                            Default Branch
                            <div className="mt-1 flex items-center gap-2">
                                {branches.length > 0 ? (
                                    <div className="flex-1 space-y-2">
                                        <select
                                            value={branches.includes(form.defaultBranch || '') ? form.defaultBranch : '__custom__'}
                                            onChange={(e) => {
                                                const next = e.target.value
                                                setForm({ ...form, defaultBranch: next === '__custom__' ? '' : next })
                                            }}
                                            className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100 dark:focus:border-blue-400"
                                        >
                                            <option value="" disabled>Select a branch</option>
                                            {branches.map((branch) => (
                                                <option key={branch} value={branch}>
                                                    {branch}
                                                </option>
                                            ))}
                                            <option value="__custom__">Custom branch...</option>
                                        </select>
                                        {(!form.defaultBranch || !branches.includes(form.defaultBranch)) && (
                                            <input
                                                value={form.defaultBranch || ''}
                                                onChange={(e) => setForm({ ...form, defaultBranch: e.target.value })}
                                                className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100 dark:focus:border-blue-400"
                                                placeholder="main"
                                            />
                                        )}
                                    </div>
                                ) : (
                                    <input
                                        value={form.defaultBranch || ''}
                                        onChange={(e) => setForm({ ...form, defaultBranch: e.target.value })}
                                        className="flex-1 rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100 dark:focus:border-blue-400"
                                        placeholder="main"
                                    />
                                )}
                                <button
                                    type="button"
                                    onClick={handleFetchBranches}
                                    disabled={branchesLoading || !form.repositoryUrl?.trim()}
                                    className="whitespace-nowrap rounded-md border border-slate-300 px-3 py-2 text-xs font-medium text-slate-700 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                                >
                                    {branchesLoading ? 'Loading...' : 'Load Branches'}
                                </button>
                            </div>
                            {branchesError ? (
                                <p className="mt-1 text-xs text-red-600 dark:text-red-400">{branchesError}</p>
                            ) : null}
                        </label>
                        <label className="block text-sm text-slate-700 dark:text-slate-300 md:col-span-2">
                            <span className="inline-flex items-center gap-2">
                                Repository Auth (optional)
                                <HelpTooltip text="Source auth precedence: source repository auth first, then project active repository auth, then anonymous clone." />
                            </span>
                            <div className="mt-1 grid grid-cols-1 gap-2 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
                                <select
                                    value={form.repositoryAuthId || ''}
                                    onChange={(e) => setForm({ ...form, repositoryAuthId: e.target.value || undefined })}
                                    disabled={authsLoading}
                                    className="w-full min-w-0 rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100 dark:focus:border-blue-400"
                                >
                                    <option value="">
                                        {authsLoading ? 'Loading auth...' : 'No repository auth selected'}
                                    </option>
                                    {repositoryAuths.map((auth) => (
                                        <option key={auth.id} value={auth.id}>
                                            {auth.name} — {auth.id}
                                        </option>
                                    ))}
                                </select>
                                <button
                                    type="button"
                                    onClick={() => setShowAuthModal(true)}
                                    className="whitespace-nowrap rounded-md border border-blue-300 px-3 py-2 text-xs font-medium text-blue-700 hover:bg-blue-50 dark:border-blue-700 dark:text-blue-300 dark:hover:bg-blue-900/20"
                                >
                                    Create New Auth
                                </button>
                            </div>
                            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                Choose existing credentials or create a new one. Leave empty for public repositories.
                            </p>
                            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                The UUID shown here is the repository auth identifier used by source settings.
                            </p>
                            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                This source-level credential takes precedence over project-level repository auth for source and `image-factory.yaml` fetch.
                            </p>
                        </label>
                    </div>
                    <div className="mt-4 flex flex-wrap items-center gap-4">
                        <label className="inline-flex items-center text-sm text-slate-700 dark:text-slate-300">
                            <input
                                type="checkbox"
                                checked={!!form.isDefault}
                                onChange={(e) => setForm({ ...form, isDefault: e.target.checked })}
                                className="mr-2 h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-900 dark:focus:ring-blue-400"
                            />
                            Set as default
                        </label>
                        <label className="inline-flex items-center text-sm text-slate-700 dark:text-slate-300">
                            <input
                                type="checkbox"
                                checked={form.isActive !== false}
                                onChange={(e) => setForm({ ...form, isActive: e.target.checked })}
                                className="mr-2 h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-900 dark:focus:ring-blue-400"
                            />
                            Active
                        </label>
                    </div>
                    <div className="mt-6 flex items-center gap-3">
                        <button
                            type="submit"
                            disabled={saving}
                            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-blue-500 dark:hover:bg-blue-400"
                        >
                            {saving ? 'Saving...' : editing ? 'Update Source' : 'Create Source'}
                        </button>
                        {editing ? (
                            <button
                                type="button"
                                onClick={closeDrawer}
                                className="rounded-md border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-700"
                            >
                                Cancel
                            </button>
                        ) : null}
                    </div>
                </form>
                </Drawer>
            ) : null}
            {canEdit ? (
                <RepositoryAuthModal
                    projectId={projectId}
                    isOpen={showAuthModal}
                    onClose={() => setShowAuthModal(false)}
                    onSuccess={() => {
                        setAuthRefreshKey((prev) => prev + 1)
                    }}
                />
            ) : null}
        </div>
    )
}

export default ProjectSourcesPanel
