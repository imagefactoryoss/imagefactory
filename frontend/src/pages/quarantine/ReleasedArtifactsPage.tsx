import Drawer from '@/components/ui/Drawer'
import type { ReleasedArtifact } from '@/types'
import { ImageImportApiError, imageImportService } from '@/services/imageImportService'
import { projectService } from '@/services/projectService'
import { useTenantStore } from '@/store/tenant'
import React, { useCallback, useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import { useNavigate } from 'react-router-dom'
import { Link } from 'react-router-dom'

const ReleasedArtifactsPage: React.FC = () => {
    const navigate = useNavigate()
    const { selectedTenantId } = useTenantStore()
    const [items, setItems] = useState<ReleasedArtifact[]>([])
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [search, setSearch] = useState('')
    const [total, setTotal] = useState(0)
    const [selectedArtifact, setSelectedArtifact] = useState<ReleasedArtifact | null>(null)
    const [projects, setProjects] = useState<Array<{ id: string; name: string }>>([])
    const [selectedProjectId, setSelectedProjectId] = useState('')
    const [projectLoading, setProjectLoading] = useState(false)
    const [consuming, setConsuming] = useState(false)

    const loadReleasedArtifacts = useCallback(async (searchValue: string) => {
        setLoading(true)
        setError(null)
        try {
            const result = await imageImportService.listReleasedArtifacts({
                page: 1,
                limit: 50,
                search: searchValue,
            })
            setItems(result.items)
            setTotal(result.pagination.total)
        } catch (err) {
            if (err instanceof ImageImportApiError) {
                setError(err.message)
            } else {
                setError('Failed to load released artifacts')
            }
            setItems([])
            setTotal(0)
        } finally {
            setLoading(false)
        }
    }, [])

    useEffect(() => {
        loadReleasedArtifacts('')
    }, [loadReleasedArtifacts])

    const handleSearchSubmit = (event: React.FormEvent) => {
        event.preventDefault()
        loadReleasedArtifacts(search)
    }

    const openUseInProjectDrawer = async (artifact: ReleasedArtifact) => {
        setSelectedArtifact(artifact)
        setSelectedProjectId('')
        if (!selectedTenantId) {
            setProjects([])
            return
        }
        try {
            setProjectLoading(true)
            const response = await projectService.getProjects({
                page: 1,
                limit: 100,
                status: ['active'],
                tenantId: selectedTenantId,
            })
            const rows = response.data.map((project) => ({ id: project.id, name: project.name }))
            setProjects(rows)
            if (rows.length > 0) {
                setSelectedProjectId(rows[0].id)
            }
        } catch (err: any) {
            toast.error(err?.message || 'Failed to load tenant projects')
            setProjects([])
        } finally {
            setProjectLoading(false)
        }
    }

    const handleUseInProject = async () => {
        if (!selectedArtifact || !selectedProjectId) return
        try {
            setConsuming(true)
            await imageImportService.consumeReleasedArtifact(selectedArtifact.id, selectedProjectId, 'Selected from released artifacts workspace')
            toast.success('Artifact linked to project build flow')
            const query = new URLSearchParams({
                projectId: selectedProjectId,
                baseImage: selectedArtifact.internal_image_ref || '',
                sourceImageRef: selectedArtifact.source_image_ref || '',
            })
            setSelectedArtifact(null)
            navigate(`/builds/new?${query.toString()}`)
        } catch (err) {
            const message = err instanceof ImageImportApiError ? err.message : 'Failed to use artifact in project'
            toast.error(message)
        } finally {
            setConsuming(false)
        }
    }

    return (
        <div className="space-y-6 px-4 py-6 sm:px-6 lg:px-8">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                    <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">Released Artifacts</h1>
                    <p className="mt-2 text-sm text-slate-700 dark:text-slate-400">
                        Tenant-consumable artifacts promoted from quarantine release workflow.
                    </p>
                </div>
                <Link
                    to="/quarantine/requests"
                    className="inline-flex items-center rounded-md border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                >
                    Back to Requests
                </Link>
            </div>

            <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900">
                <form onSubmit={handleSearchSubmit} className="flex flex-col gap-3 sm:flex-row sm:items-center">
                    <input
                        value={search}
                        onChange={(event) => setSearch(event.target.value)}
                        placeholder="Search by source image, internal ref, registry, digest"
                        className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 placeholder:text-slate-400 focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-200 dark:border-slate-600 dark:bg-slate-950 dark:text-slate-100 dark:placeholder:text-slate-500 dark:focus:border-sky-400 dark:focus:ring-sky-900/50"
                    />
                    <button
                        type="submit"
                        className="rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 dark:bg-sky-500 dark:hover:bg-sky-600"
                    >
                        Search
                    </button>
                </form>
                <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
                    Total released artifacts: {total}
                </p>
            </section>

            <section className="rounded-lg border border-slate-200 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-900">
                {loading ? (
                    <p className="p-4 text-sm text-slate-500 dark:text-slate-400">Loading released artifacts...</p>
                ) : error ? (
                    <p className="p-4 text-sm text-rose-700 dark:text-rose-300">{error}</p>
                ) : items.length === 0 ? (
                    <p className="p-4 text-sm text-slate-500 dark:text-slate-400">No released artifacts found for this tenant.</p>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
                            <thead className="bg-slate-50 dark:bg-slate-800/60">
                                <tr>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Source</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Internal Ref</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Digest</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Released</th>
                                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Posture</th>
                                    <th className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-slate-300">Actions</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                                {items.map((item) => (
                                    <tr key={item.id} className="align-top">
                                        <td className="px-3 py-3 text-xs text-slate-800 dark:text-slate-100">
                                            <p className="font-medium">{item.source_image_ref}</p>
                                            <p className="mt-1 text-slate-500 dark:text-slate-400">{item.source_registry}</p>
                                        </td>
                                        <td className="px-3 py-3 text-xs text-slate-700 dark:text-slate-200">{item.internal_image_ref || '-'}</td>
                                        <td className="px-3 py-3 text-xs text-slate-700 dark:text-slate-200">{item.source_image_digest || '-'}</td>
                                        <td className="px-3 py-3 text-xs text-slate-700 dark:text-slate-200">
                                            {item.released_at ? new Date(item.released_at).toLocaleString() : '-'}
                                        </td>
                                        <td className="px-3 py-3 text-xs text-slate-700 dark:text-slate-200">
                                            {item.consumption_ready ? (
                                                <span className="inline-flex rounded-full bg-emerald-100 px-2 py-0.5 font-semibold text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-200">
                                                    Ready
                                                </span>
                                            ) : (
                                                <span className="inline-flex rounded-full bg-amber-100 px-2 py-0.5 font-semibold text-amber-800 dark:bg-amber-900/50 dark:text-amber-200">
                                                    Blocked
                                                </span>
                                            )}
                                            <p className="mt-1 text-[11px] text-slate-500 dark:text-slate-400">
                                                {item.consumption_ready
                                                    ? `Policy ${item.policy_decision || 'pass'}`
                                                    : item.consumption_blocker_reason || 'Not consumable yet'}
                                            </p>
                                        </td>
                                        <td className="px-3 py-3 text-right text-xs">
                                            <button
                                                type="button"
                                                disabled={!item.consumption_ready}
                                                onClick={() => void openUseInProjectDrawer(item)}
                                                className="rounded-md border border-sky-300 bg-sky-50 px-2.5 py-1 font-medium text-sky-800 hover:bg-sky-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-sky-700 dark:bg-sky-900/30 dark:text-sky-200 dark:hover:bg-sky-900/50"
                                            >
                                                Use in Project
                                            </button>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </section>

            <Drawer
                isOpen={Boolean(selectedArtifact)}
                onClose={() => {
                    if (!consuming) {
                        setSelectedArtifact(null)
                    }
                }}
                title="Use Released Artifact in Project"
                size="md"
            >
                {selectedArtifact ? (
                    <div className="space-y-4">
                        <div className="rounded-md border border-slate-200 bg-slate-50 p-3 text-xs dark:border-slate-700 dark:bg-slate-800/50">
                            <p className="font-semibold text-slate-900 dark:text-slate-100">{selectedArtifact.source_image_ref}</p>
                            <p className="mt-1 text-slate-600 dark:text-slate-300">Internal ref: {selectedArtifact.internal_image_ref || '-'}</p>
                            <p className="mt-1 text-slate-600 dark:text-slate-300">Digest: {selectedArtifact.source_image_digest || '-'}</p>
                        </div>
                        <div>
                            <label className="mb-1 block text-xs font-medium text-slate-700 dark:text-slate-200">Project</label>
                            <select
                                value={selectedProjectId}
                                onChange={(event) => setSelectedProjectId(event.target.value)}
                                disabled={projectLoading || consuming}
                                className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-200 dark:border-slate-600 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900/50"
                            >
                                {projects.length === 0 ? (
                                    <option value="">{projectLoading ? 'Loading projects...' : 'No active projects available'}</option>
                                ) : (
                                    projects.map((project) => (
                                        <option key={project.id} value={project.id}>{project.name}</option>
                                    ))
                                )}
                            </select>
                        </div>
                        <div className="flex justify-end gap-2">
                            <button
                                type="button"
                                onClick={() => setSelectedArtifact(null)}
                                disabled={consuming}
                                className="rounded-md border border-slate-300 px-3 py-2 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                            >
                                Cancel
                            </button>
                            <button
                                type="button"
                                disabled={!selectedProjectId || consuming}
                                onClick={() => void handleUseInProject()}
                                className="rounded-md bg-sky-600 px-3 py-2 text-xs font-medium text-white hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-sky-500 dark:hover:bg-sky-600"
                            >
                                {consuming ? 'Opening...' : 'Open Build Wizard'}
                            </button>
                        </div>
                    </div>
                ) : null}
            </Drawer>
        </div>
    )
}

export default ReleasedArtifactsPage
