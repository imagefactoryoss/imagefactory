import ConfirmDialog from '@/components/common/ConfirmDialog'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import useBuildStatusWebSocket from '@/hooks/useBuildStatusWebSocket'
import { buildService } from '@/services/buildService'
import { projectService } from '@/services/projectService'
import { useOperationCapabilitiesStore } from '@/store/operationCapabilities'
import { useTenantStore } from '@/store/tenant'
import { Build, BuildStatus } from '@/types'
import { canCreateBuilds } from '@/utils/permissions'
import { AlertCircle, Plus } from 'lucide-react'
import React, { useEffect, useRef, useState } from 'react'
import toast from 'react-hot-toast'
import { Link, useNavigate } from 'react-router-dom'

const BuildsPage: React.FC = () => {
    const navigate = useNavigate()
    const { selectedTenantId } = useTenantStore()
    const operationCapabilities = useOperationCapabilitiesStore((state) => state.capabilities)
    const canUseBuildCapability = operationCapabilities.build
    const canCreateBuild = canCreateBuilds() && canUseBuildCapability
    const [builds, setBuilds] = useState<Build[]>([])
    const [loading, setLoading] = useState(true)
    const [currentPage, setCurrentPage] = useState(1)
    const [totalPages, setTotalPages] = useState(1)
    const [statusFilter, setStatusFilter] = useState<string>('all')
    const [searchQuery, setSearchQuery] = useState('')
    const [hasProjects, setHasProjects] = useState(false)
    const [projectsLoading, setProjectsLoading] = useState(true)
    const [wsReloadCounter, setWsReloadCounter] = useState(0)
    const [deleteTarget, setDeleteTarget] = useState<{ id: string; status: string } | null>(null)
    const [sourceNameById, setSourceNameById] = useState<Record<string, string>>({})
    const sourceLookupRequestSeq = useRef(0)
    const confirmDialog = useConfirmDialog()

    const itemsPerPage = 20

    const { isConnected: isWsConnected } = useBuildStatusWebSocket({
        enabled: hasProjects,
        onBuildEvent: () => setWsReloadCounter((current) => current + 1),
    })

    useEffect(() => {
        checkProjects()
    }, [selectedTenantId])

    useEffect(() => {
        if (hasProjects) {
            loadBuilds()
        }
    }, [currentPage, statusFilter, searchQuery, selectedTenantId, hasProjects, wsReloadCounter])

    useEffect(() => {
        if (!hasProjects || !selectedTenantId || builds.length === 0) {
            setSourceNameById({})
            return
        }

        const projectIDs = Array.from(new Set(builds.map((build) => build.projectId).filter(Boolean)))
        if (projectIDs.length === 0) {
            setSourceNameById({})
            return
        }

        const requestSeq = sourceLookupRequestSeq.current + 1
        sourceLookupRequestSeq.current = requestSeq
        let cancelled = false

        const loadSourceNames = async () => {
            const sourcesByID: Record<string, string> = {}
            await Promise.all(
                projectIDs.map(async (projectID) => {
                    try {
                        const sources = await projectService.listProjectSources(projectID)
                        sources.forEach((source) => {
                            if (!sourcesByID[source.id]) {
                                sourcesByID[source.id] = source.name || source.repositoryUrl || source.id.slice(0, 8)
                            }
                        })
                    } catch {
                        // Source list lookup is best-effort for display labels.
                    }
                })
            )
            if (cancelled || sourceLookupRequestSeq.current !== requestSeq) {
                return
            }
            setSourceNameById(sourcesByID)
        }

        loadSourceNames()
        return () => {
            cancelled = true
        }
    }, [builds, hasProjects, selectedTenantId])

    const checkProjects = async () => {
        if (!selectedTenantId) {
            setProjectsLoading(false)
            setHasProjects(false)
            return
        }

        try {
            setProjectsLoading(true)
            const response = await projectService.getProjects({
                tenantId: selectedTenantId,
                limit: 1,
            })
            setHasProjects((response.data && response.data.length > 0) || false)
        } catch (error: any) {
            setHasProjects(false)
        } finally {
            setProjectsLoading(false)
        }
    }

    const loadBuilds = async () => {
        if (!selectedTenantId) {
            setLoading(false)
            return
        }

        try {
            setLoading(true)
            const params: any = {
                page: currentPage,
                limit: itemsPerPage,
                tenantId: selectedTenantId,
            }

            if (statusFilter !== 'all') {
                params.status = [statusFilter]
            }

            if (searchQuery.trim()) {
                params.search = searchQuery.trim()
            }

            const response = await buildService.getBuilds(params)
            setBuilds(response.data)
            setTotalPages(response.pagination.totalPages)
        } catch (error: any) {
            toast.error('Failed to load builds')
        } finally {
            setLoading(false)
        }
    }

    const handleCancelBuild = async (buildId: string) => {
        const confirmed = await confirmDialog({
            title: 'Cancel Build',
            message: 'Are you sure you want to cancel this build?',
            confirmLabel: 'Cancel Build',
            destructive: true,
        })
        if (!confirmed) return

        try {
            await buildService.cancelBuild(buildId)
            toast.success('Build cancelled successfully')
            loadBuilds() // Refresh the list
        } catch (error: any) {
            toast.error(error.message || 'Failed to cancel build')
        }
    }

    const canDeleteBuild = (status: string) => status !== 'running' && status !== 'queued'

    const handleDeleteBuild = async (buildId: string, status: string) => {
        if (!canDeleteBuild(status)) {
            toast.error('Cannot delete build while execution is running or queued')
            return
        }
        setDeleteTarget({ id: buildId, status })
    }

    const confirmDeleteBuild = async () => {
        if (!deleteTarget) return
        try {
            await buildService.deleteBuild(deleteTarget.id)
            toast.success('Build deleted successfully')
            loadBuilds()
        } catch (error: any) {
            toast.error(error.message || 'Failed to delete build')
        } finally {
            setDeleteTarget(null)
        }
    }

    const handleCloneBuild = async (buildId: string) => {
        const confirmed = await confirmDialog({
            title: 'Clone Build Configuration',
            message: 'Open this build in the create wizard with prefilled configuration?',
            confirmLabel: 'Open Wizard',
        })
        if (!confirmed) return
        navigate(`/builds/new?cloneFrom=${encodeURIComponent(buildId)}`)
    }

    const getStatusColor = (status: BuildStatus): string => {
        switch (status) {
            case 'completed': return 'text-green-600 bg-green-100'
            case 'running': return 'text-blue-600 bg-blue-100'
            case 'failed': return 'text-red-600 bg-red-100'
            case 'cancelled': return 'text-gray-600 bg-gray-100'
            case 'pending': return 'text-yellow-600 bg-yellow-100'
            case 'queued': return 'text-purple-600 bg-purple-100'
            default: return 'text-gray-600 bg-gray-100'
        }
    }

    const formatDuration = (build: Build): string => {
        if (!build.startedAt) return '-'

        const startTime = new Date(build.startedAt)
        const endTime = build.completedAt ? new Date(build.completedAt) : new Date()

        const durationMs = endTime.getTime() - startTime.getTime()
        const minutes = Math.floor(durationMs / 60000)
        const seconds = Math.floor((durationMs % 60000) / 1000)

        return `${minutes}m ${seconds}s`
    }

    const formatDate = (dateString: string): string => {
        return new Date(dateString).toLocaleString()
    }

    return (
        <div className="min-h-screen bg-slate-50 dark:bg-slate-900 px-4 py-6 sm:px-6 lg:px-8">
            <ConfirmDialog
                isOpen={!!deleteTarget}
                title="Delete Build"
                message="Delete this build and all related execution data? This action cannot be undone."
                confirmLabel="Delete Build"
                destructive
                onConfirm={confirmDeleteBuild}
                onCancel={() => setDeleteTarget(null)}
            />
            <div className="sm:flex sm:items-center">
                <div className="sm:flex-auto">
                    <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">Builds</h1>
                    <p className="mt-2 text-sm text-slate-700 dark:text-slate-400">
                        Manage your container builds and deployments.
                    </p>
                    {hasProjects && (
                        <p className={`mt-2 inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ${
                            isWsConnected
                                ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
                                : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
                        }`}>
                            {isWsConnected ? 'Live status updates on' : 'Live status updates reconnecting'}
                        </p>
                    )}
                </div>
                <div className="mt-4 sm:mt-0 sm:ml-16 sm:flex-none">
                    {canCreateBuild && (
                        <button
                            onClick={() => window.location.href = '/builds/new'}
                            disabled={!hasProjects || projectsLoading}
                            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition disabled:bg-gray-400 disabled:cursor-not-allowed"
                        >
                            <Plus className="h-5 w-5" />
                            New Build
                        </button>
                    )}
                </div>
            </div>

            {/* No Projects Guidance */}
            {!projectsLoading && !hasProjects && (
                <div className="mt-6 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 overflow-hidden">
                    <div className="px-5 py-4 border-b border-slate-200 dark:border-slate-700 bg-gradient-to-r from-amber-50 via-orange-50 to-yellow-50 dark:from-slate-800 dark:via-slate-800 dark:to-slate-700 flex items-start gap-3">
                        <AlertCircle className="h-5 w-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
                        <div>
                            <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
                                Set up a project to start building images
                            </h3>
                            <p className="mt-1 text-sm text-slate-700 dark:text-slate-300">
                                Builds run from project sources and settings. Create your first project, connect a repository, then trigger your first build.
                            </p>
                        </div>
                    </div>
                    <div className="px-5 py-4">
                        <ol className="list-decimal pl-5 text-sm text-slate-700 dark:text-slate-300 space-y-1">
                            <li>Create a project in the Projects page.</li>
                            <li>Add source repository and auth (if needed).</li>
                            <li>Choose build config mode and run your first build.</li>
                        </ol>
                        <div className="mt-4">
                            <Link
                                to="/projects"
                                className="inline-flex items-center rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700"
                            >
                                Go to Projects
                            </Link>
                        </div>
                    </div>
                </div>
            )}

            {/* Filters */}
            {hasProjects && (
                <div className="mt-6 flex flex-col sm:flex-row gap-4">
                    <div className="flex-1">
                        <input
                            type="text"
                            placeholder="Search builds..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            className="input w-full"
                        />
                    </div>
                    <div className="sm:w-48">
                        <select
                            value={statusFilter}
                            onChange={(e) => setStatusFilter(e.target.value)}
                            className="input w-full"
                        >
                            <option value="all">All Status</option>
                            <option value="pending">Pending</option>
                            <option value="queued">Queued</option>
                            <option value="running">Running</option>
                            <option value="completed">Completed</option>
                            <option value="failed">Failed</option>
                            <option value="cancelled">Cancelled</option>
                        </select>
                    </div>
                </div>
            )}

            {/* Builds Table */}
            {hasProjects && (
                <div className="mt-8">
                    <div className="card">
                        <div className="card-body p-0">
                            {loading ? (
                                <div className="text-center py-8">
                                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500 mx-auto"></div>
                                    <p className="mt-2 text-sm text-muted-foreground">Loading builds...</p>
                                </div>
                            ) : builds.length === 0 ? (
                                searchQuery.trim() || statusFilter !== 'all' ? (
                                    <div className="text-center py-12">
                                        <p className="text-sm text-slate-600 dark:text-slate-400">
                                            No builds found for the current search or status filter.
                                        </p>
                                    </div>
                                ) : (
                                    <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 m-6 overflow-hidden">
                                        <div className="px-5 py-4 border-b border-slate-200 dark:border-slate-700 bg-gradient-to-r from-blue-50 via-cyan-50 to-emerald-50 dark:from-slate-800 dark:via-slate-800 dark:to-slate-700">
                                            <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100">No Builds Yet</h3>
                                            <p className="mt-1 text-sm text-slate-700 dark:text-slate-300">
                                                Builds execute your project configuration and generate image evidence for traceability and security review.
                                            </p>
                                        </div>
                                        <div className="p-5 grid grid-cols-1 lg:grid-cols-2 gap-5">
                                            <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/60 p-4">
                                                <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-2">Build Flow</h4>
                                                <ol className="space-y-2 text-sm text-slate-700 dark:text-slate-300">
                                                    <li><span className="font-medium">1.</span> Pick project + source ref.</li>
                                                    <li><span className="font-medium">2.</span> Resolve build config (UI or `image-factory.yaml`).</li>
                                                    <li><span className="font-medium">3.</span> Execute and stream logs.</li>
                                                    <li><span className="font-medium">4.</span> Persist output evidence.</li>
                                                </ol>
                                            </div>
                                            <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/60 p-4">
                                                <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-2">Evidence Produced</h4>
                                                <div className="grid grid-cols-3 gap-2 text-xs font-medium">
                                                    <div className="rounded-md bg-cyan-50 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-300 px-2 py-1 text-center border border-cyan-200 dark:border-cyan-800">Layers</div>
                                                    <div className="rounded-md bg-violet-50 text-violet-700 dark:bg-violet-900/30 dark:text-violet-300 px-2 py-1 text-center border border-violet-200 dark:border-violet-800">SBOM</div>
                                                    <div className="rounded-md bg-rose-50 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300 px-2 py-1 text-center border border-rose-200 dark:border-rose-800">Vulns</div>
                                                </div>
                                            </div>
                                        </div>
                                        {canCreateBuild && (
                                            <div className="px-5 pb-5">
                                                <Link
                                                    to="/builds/new"
                                                    className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
                                                >
                                                    <Plus className="h-4 w-4" />
                                                    Create First Build
                                                </Link>
                                            </div>
                                        )}
                                    </div>
                                )
                            ) : (
                                <div className="overflow-x-auto">
                                    <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                                        <thead className="bg-gray-50 dark:bg-gray-900">
                                            <tr>
                                                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                                                    Build
                                                </th>
                                                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                                                    Status
                                                </th>
                                                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                                                    Type
                                                </th>
                                                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                                                    Duration
                                                </th>
                                                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                                                    Created
                                                </th>
                                                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                                                    Actions
                                                </th>
                                            </tr>
                                        </thead>
                                        <tbody className="divide-y divide-gray-200 dark:divide-gray-700 bg-white dark:bg-gray-800">
                                            {builds.map((build) => {
                                                const metadata = (build.manifest.metadata || {}) as Record<string, any>
                                                const sourceId = build.manifest.buildConfig?.sourceId || metadata.source_id || metadata.sourceId
                                                const refPolicy = build.manifest.buildConfig?.refPolicy || metadata.ref_policy || metadata.refPolicy
                                                const sourceLabel = sourceId
                                                    ? sourceNameById[sourceId] || sourceId.slice(0, 8)
                                                    : 'Not set'

                                                return (
                                                    <tr key={build.id} className="hover:bg-gray-50 dark:hover:bg-gray-700 transition">
                                                    <td className="px-6 py-4 whitespace-nowrap">
                                                        <div>
                                                            <Link
                                                                to={`/builds/${build.id}`}
                                                                className="text-sm font-medium text-gray-900 dark:text-white hover:text-blue-600"
                                                            >
                                                                {build.manifest.name}
                                                            </Link>
                                                            <div className="text-sm text-gray-500 dark:text-gray-400">
                                                                {build.id.slice(0, 8)}...
                                                            </div>
                                                            <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                                                                <span className="rounded bg-slate-200 px-2 py-0.5 text-slate-700 dark:bg-slate-700 dark:text-slate-200">
                                                                    Source: {sourceLabel}
                                                                </span>
                                                                <span className="rounded bg-slate-200 px-2 py-0.5 text-slate-700 dark:bg-slate-700 dark:text-slate-200">
                                                                    Ref: {refPolicy === 'fixed'
                                                                        ? 'Fixed'
                                                                        : refPolicy === 'event_ref'
                                                                            ? 'Webhook'
                                                                            : 'Default'}
                                                                </span>
                                                            </div>
                                                        </div>
                                                    </td>
                                                    <td className="px-6 py-4 whitespace-nowrap">
                                                        <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getStatusColor(build.status)}`}>
                                                            {build.status.charAt(0).toUpperCase() + build.status.slice(1)}
                                                        </span>
                                                    </td>
                                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                                                        {build.manifest.type.charAt(0).toUpperCase() + build.manifest.type.slice(1)}
                                                    </td>
                                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-400">
                                                        {formatDuration(build)}
                                                    </td>
                                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-400">
                                                        {formatDate(build.createdAt)}
                                                    </td>
                                                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                                        <div className="flex justify-end space-x-2">
                                                            <Link
                                                                to={`/builds/${build.id}`}
                                                                className="text-blue-600 hover:text-blue-900"
                                                            >
                                                                View
                                                            </Link>
                                                            {build.status === 'running' && (
                                                                <button
                                                                    onClick={() => handleCancelBuild(build.id)}
                                                                    className="text-red-600 hover:text-red-900"
                                                                >
                                                                    Cancel
                                                                </button>
                                                            )}
                                                            <button
                                                                onClick={() => handleCloneBuild(build.id)}
                                                                className="text-indigo-700 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-300"
                                                            >
                                                                Clone
                                                            </button>
                                                            <button
                                                                onClick={() => handleDeleteBuild(build.id, build.status)}
                                                                disabled={!canDeleteBuild(build.status)}
                                                                className="text-red-700 hover:text-red-900 disabled:text-slate-400 disabled:cursor-not-allowed dark:text-red-400 dark:hover:text-red-300 dark:disabled:text-slate-500"
                                                                title={canDeleteBuild(build.status) ? 'Delete build' : 'Cannot delete running or queued build'}
                                                            >
                                                                Delete
                                                            </button>
                                                        </div>
                                                    </td>
                                                    </tr>
                                                )
                                            })}
                                        </tbody>
                                    </table>
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            )}

            {/* Pagination */}
            {hasProjects && totalPages > 1 && (
                <div className="mt-6 flex items-center justify-between">
                    <div className="text-sm text-slate-700 dark:text-slate-400">
                        Page {currentPage} of {totalPages}
                    </div>
                    <div className="flex space-x-2">
                        <button
                            onClick={() => setCurrentPage(prev => Math.max(1, prev - 1))}
                            disabled={currentPage === 1}
                            className="btn btn-secondary disabled:opacity-50"
                        >
                            Previous
                        </button>
                        <button
                            onClick={() => setCurrentPage(prev => Math.min(totalPages, prev + 1))}
                            disabled={currentPage === totalPages}
                            className="btn btn-secondary disabled:opacity-50"
                        >
                            Next
                        </button>
                    </div>
                </div>
            )}
        </div>
    )
}

export default BuildsPage
