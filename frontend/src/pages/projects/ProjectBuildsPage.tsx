import ConfirmDialog from '@/components/common/ConfirmDialog'
import { EmptyState } from '@/components/common/EmptyState'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import useBuildStatusWebSocket from '@/hooks/useBuildStatusWebSocket'
import { buildService } from '@/services/buildService'
import { projectService } from '@/services/projectService'
import { useOperationCapabilitiesStore } from '@/store/operationCapabilities'
import { useTenantStore } from '@/store/tenant'
import { Build, BuildStatus, Project, ProjectSource } from '@/types'
import { canCreateBuilds } from '@/utils/permissions'
import { AlertCircle, ChevronRight, Plus, Search } from 'lucide-react'
import React, { useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import { Link, useNavigate, useParams } from 'react-router-dom'

const ProjectBuildsPage: React.FC = () => {
    const { projectId } = useParams<{ projectId: string }>()
    const navigate = useNavigate()
    const { selectedTenantId } = useTenantStore()
    const operationCapabilities = useOperationCapabilitiesStore((state) => state.capabilities)
    const canCreateBuild = canCreateBuilds() && operationCapabilities.build

    const [project, setProject] = useState<Project | null>(null)
    const [projectSources, setProjectSources] = useState<ProjectSource[]>([])
    const [builds, setBuilds] = useState<Build[]>([])
    const [loading, setLoading] = useState(true)
    const [projectLoading, setProjectLoading] = useState(true)
    const [currentPage, setCurrentPage] = useState(1)
    const [totalPages, setTotalPages] = useState(1)
    const [statusFilter, setStatusFilter] = useState<string>('all')
    const [searchQuery, setSearchQuery] = useState('')
    const [wsReloadCounter, setWsReloadCounter] = useState(0)
    const [deleteTarget, setDeleteTarget] = useState<{ id: string; status: string } | null>(null)
    const confirmDialog = useConfirmDialog()

    const itemsPerPage = 20

    const { isConnected: isWsConnected } = useBuildStatusWebSocket({
        enabled: !!projectId,
        filterProjectId: projectId,
        onBuildEvent: () => setWsReloadCounter((current) => current + 1),
    })

    // Load project details
    useEffect(() => {
        if (projectId) {
            loadProject()
            loadProjectSources()
        }
    }, [projectId])

    // Load builds whenever filters or page changes
    useEffect(() => {
        if (projectId) {
            loadBuilds()
        }
    }, [projectId, currentPage, statusFilter, searchQuery, selectedTenantId, wsReloadCounter])

    const loadProject = async () => {
        if (!projectId) return

        try {
            setProjectLoading(true)
            const response = await projectService.getProject(projectId)
            setProject(response)
        } catch (error: any) {
            toast.error('Failed to load project')
            navigate('/projects')
        } finally {
            setProjectLoading(false)
        }
    }

    const loadProjectSources = async () => {
        if (!projectId) return
        try {
            const sources = await projectService.listProjectSources(projectId)
            setProjectSources(sources)
        } catch (error: any) {
            toast.error(error?.message || 'Failed to load project sources')
        }
    }

    const loadBuilds = async () => {
        if (!projectId || !selectedTenantId) {
            setLoading(false)
            return
        }

        try {
            setLoading(true)
            const params: any = {
                page: currentPage,
                limit: itemsPerPage,
                projectId: projectId,
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

    const handleCancelBuild = async (buildId: string, event: React.MouseEvent) => {
        event.preventDefault()
        event.stopPropagation()

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
            loadBuilds()
        } catch (error: any) {
            toast.error(error.message || 'Failed to cancel build')
        }
    }

    const canDeleteBuild = (status: string) => status !== 'running' && status !== 'queued'

    const handleCloneBuild = async (buildId: string, event: React.MouseEvent) => {
        event.preventDefault()
        event.stopPropagation()

        const confirmed = await confirmDialog({
            title: 'Clone Build Configuration',
            message: 'Open this build in the create wizard with prefilled configuration?',
            confirmLabel: 'Open Wizard',
        })
        if (!confirmed) return
        navigate(`/builds/new?cloneFrom=${encodeURIComponent(buildId)}`)
    }

    const handleDeleteBuild = async (buildId: string, status: string, event: React.MouseEvent) => {
        event.preventDefault()
        event.stopPropagation()

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

    const getStatusColor = (status: BuildStatus): string => {
        switch (status) {
            case 'queued':
                return 'bg-yellow-50 dark:bg-slate-800 border-yellow-200 dark:border-yellow-700 text-yellow-800 dark:text-yellow-200'
            case 'running':
                return 'bg-blue-50 dark:bg-slate-800 border-blue-200 dark:border-blue-700 text-blue-800 dark:text-blue-200'
            case 'completed':
                return 'bg-green-50 dark:bg-slate-800 border-green-200 dark:border-green-700 text-green-800 dark:text-green-200'
            case 'failed':
                return 'bg-red-50 dark:bg-slate-800 border-red-200 dark:border-red-700 text-red-800 dark:text-red-200'
            case 'cancelled':
                return 'bg-gray-50 dark:bg-slate-800 border-gray-200 dark:border-gray-700 text-gray-800 dark:text-gray-200'
            default:
                return 'bg-gray-50 dark:bg-slate-800 border-gray-200 dark:border-gray-700 text-gray-800 dark:text-gray-200'
        }
    }

    const getStatusBadgeColor = (status: BuildStatus): string => {
        switch (status) {
            case 'queued':
                return 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200'
            case 'running':
                return 'bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-200'
            case 'completed':
                return 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200'
            case 'failed':
                return 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-200'
            case 'cancelled':
                return 'bg-gray-100 dark:bg-gray-900/30 text-gray-800 dark:text-gray-200'
            default:
                return 'bg-gray-100 dark:bg-gray-900/30 text-gray-800 dark:text-gray-200'
        }
    }

    const formatDate = (date: string | Date): string => {
        const d = new Date(date)
        return d.toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
        })
    }

    const handlePageChange = (newPage: number) => {
        if (newPage >= 1 && newPage <= totalPages) {
            setCurrentPage(newPage)
            window.scrollTo({ top: 0, behavior: 'smooth' })
        }
    }

    const handleSearch = (value: string) => {
        setSearchQuery(value)
        setCurrentPage(1)
    }

    const activeSources = projectSources.filter((source) => source.isActive)
    const defaultSource = projectSources.find((source) => source.isDefault)
    const getRefPolicyLabel = (policy?: string) => {
        if (policy === 'fixed') return 'Fixed ref'
        if (policy === 'event_ref') return 'Webhook ref'
        return 'Source default'
    }
    const resolveSourceName = (sourceId?: string) => {
        if (!sourceId) return 'Not set'
        return projectSources.find((source) => source.id === sourceId)?.name || sourceId.slice(0, 8)
    }

    if (projectLoading) {
        return (
            <div className="flex items-center justify-center min-h-screen">
                <div className="text-center">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
                    <p className="mt-4 text-gray-600">Loading project...</p>
                </div>
            </div>
        )
    }

    if (!project) {
        return (
            <div className="p-8">
                <EmptyState
                    title="Project not found"
                    description="The project you're looking for doesn't exist or you don't have access to it."
                    action={
                        <Link
                            to="/projects"
                            className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
                        >
                            Back to Projects
                        </Link>
                    }
                />
            </div>
        )
    }

    return (
        <div className="min-h-screen bg-gradient-to-br from-gray-50 dark:from-slate-900 to-gray-100 dark:to-slate-800">
            <ConfirmDialog
                isOpen={!!deleteTarget}
                title="Delete Build"
                message="Delete this build and all related execution data? This action cannot be undone."
                confirmLabel="Delete Build"
                destructive
                onConfirm={confirmDeleteBuild}
                onCancel={() => setDeleteTarget(null)}
            />
            {/* Header */}
            <div className="bg-white dark:bg-slate-800 border-b border-gray-200 dark:border-slate-700">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
                    {/* Breadcrumb */}
                    <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-slate-400 mb-4">
                        <Link to="/projects" className="hover:text-gray-900 dark:hover:text-white">
                            Projects
                        </Link>
                        <ChevronRight className="w-4 h-4" />
                        <Link to={`/projects/${projectId}`} className="hover:text-gray-900 dark:hover:text-white">
                            {project.name}
                        </Link>
                        <ChevronRight className="w-4 h-4" />
                        <span className="text-gray-900 dark:text-white font-medium">Builds</span>
                    </div>

                    {/* Title Section */}
                    <div className="flex items-center justify-between">
                        <div>
                            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Project Builds</h1>
                            <p className="mt-2 text-gray-600 dark:text-slate-400">
                                Manage and monitor builds for <span className="font-semibold">{project.name}</span>
                            </p>
                            <p className="mt-2 text-xs text-slate-600 dark:text-slate-300">
                                Sources: <span className="font-semibold">{projectSources.length}</span>
                                {' | '}
                                Active: <span className="font-semibold">{activeSources.length}</span>
                                {' | '}
                                Default: <span className="font-semibold">{defaultSource?.name || 'None'}</span>
                            </p>
                            <p className={`mt-2 inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ${
                                isWsConnected
                                    ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
                                    : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
                            }`}>
                                {isWsConnected ? 'Live status updates on' : 'Live status updates reconnecting'}
                            </p>
                        </div>
                        {canCreateBuild && (
                            <Link
                                to={`/builds/new?projectId=${projectId}`}
                                className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors dark:bg-blue-600 dark:hover:bg-blue-700"
                            >
                                <Plus className="w-4 h-4" />
                                Create Build
                            </Link>
                        )}
                    </div>
                </div>
            </div>

            {/* Main Content */}
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
                {/* Filters */}
                <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-gray-200 dark:border-slate-700 p-6 mb-8">
                    <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:gap-6">
                        {/* Search */}
                        <div className="flex-1">
                            <div className="relative">
                                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400 dark:text-slate-400" />
                                <input
                                    type="text"
                                    placeholder="Search builds by number, branch, or commit..."
                                    value={searchQuery}
                                    onChange={(e) => handleSearch(e.target.value)}
                                    className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-slate-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-slate-700 dark:text-white"
                                />
                            </div>
                        </div>

                        {/* Status Filter */}
                        <div className="flex items-center gap-2">
                            <label htmlFor="status-filter" className="text-sm font-medium text-gray-700 dark:text-slate-300">
                                Status:
                            </label>
                            <select
                                id="status-filter"
                                value={statusFilter}
                                onChange={(e) => {
                                    setStatusFilter(e.target.value)
                                    setCurrentPage(1)
                                }}
                                className="px-3 py-2 border border-gray-300 dark:border-slate-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm dark:bg-slate-700 dark:text-white"
                            >
                                <option value="all">All</option>
                                <option value="queued">Queued</option>
                                <option value="running">Running</option>
                                <option value="completed">Completed</option>
                                <option value="failed">Failed</option>
                                <option value="cancelled">Cancelled</option>
                            </select>
                        </div>
                    </div>
                </div>

                {/* Builds List */}
                {loading ? (
                    <div className="flex items-center justify-center py-12">
                        <div className="text-center">
                            <div className="animate-spin rounded-full h-10 w-10 border-b-2 border-blue-600 mx-auto"></div>
                            <p className="mt-4 text-gray-600 dark:text-slate-400">Loading builds...</p>
                        </div>
                    </div>
                ) : builds.length === 0 ? (
                    searchQuery || statusFilter !== 'all' ? (
                        <EmptyState
                            title="No builds found"
                            description="Try adjusting your filters or search query."
                        />
                    ) : (
                        <div className="rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-sm overflow-hidden">
                            <div className="px-6 py-5 border-b border-slate-200 dark:border-slate-700 bg-gradient-to-r from-blue-50 via-cyan-50 to-emerald-50 dark:from-slate-800 dark:via-slate-800 dark:to-slate-700">
                                <h2 className="text-xl font-semibold text-slate-900 dark:text-slate-100">No Builds For This Project Yet</h2>
                                <p className="mt-1 text-sm text-slate-700 dark:text-slate-300">
                                    Start a build to validate your project configuration and produce publishable image artifacts.
                                </p>
                            </div>
                            <div className="p-6 grid grid-cols-1 lg:grid-cols-2 gap-6">
                                <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/60 p-4">
                                    <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-3">Before You Run</h3>
                                    <ul className="space-y-2 text-sm text-slate-700 dark:text-slate-300">
                                        <li>Ensure at least one active source is configured for this project.</li>
                                        <li>Confirm repository auth and registry auth as needed.</li>
                                        <li>Choose UI-managed config or repository `image-factory.yaml` mode.</li>
                                    </ul>
                                </div>
                                <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/60 p-4">
                                    <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-3">What You Get</h3>
                                    <div className="grid grid-cols-3 gap-2 text-xs font-medium">
                                        <div className="rounded-md bg-cyan-50 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-300 px-2 py-1 text-center border border-cyan-200 dark:border-cyan-800">Layers</div>
                                        <div className="rounded-md bg-violet-50 text-violet-700 dark:bg-violet-900/30 dark:text-violet-300 px-2 py-1 text-center border border-violet-200 dark:border-violet-800">SBOM</div>
                                        <div className="rounded-md bg-rose-50 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300 px-2 py-1 text-center border border-rose-200 dark:border-rose-800">Vulns</div>
                                    </div>
                                    <p className="mt-3 text-xs text-slate-600 dark:text-slate-400">
                                        Build results appear here with status, logs, and trace metadata for each execution.
                                    </p>
                                </div>
                            </div>
                            {canCreateBuild && (
                                <div className="px-6 pb-6">
                                    <Link
                                        to={`/builds/new?projectId=${projectId}`}
                                        className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 dark:bg-blue-600 dark:hover:bg-blue-700"
                                    >
                                        <Plus className="w-4 h-4" />
                                        Create First Build
                                    </Link>
                                </div>
                            )}
                        </div>
                    )
                ) : (
                    <>
                        <div className="grid gap-4">
                            {builds.map((build) => (
                                <Link
                                    key={build.id}
                                    to={`/builds/${build.id}`}
                                    className={`block border rounded-lg p-6 transition-all hover:shadow-md ${getStatusColor(
                                        build.status as BuildStatus
                                    )}`}
                                >
                                    <div className="flex items-start justify-between gap-4">
                                        <div className="flex-1 min-w-0">
                                            <div className="flex items-center gap-3 mb-2">
                                                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                                                    Build #{build.buildNumber || build.id.slice(-8)}
                                                </h3>
                                                <span
                                                    className={`inline-block px-3 py-1 rounded-full text-xs font-medium ${getStatusBadgeColor(
                                                        build.status as BuildStatus
                                                    )}`}
                                                >
                                                    {build.status.replace('_', ' ').toUpperCase()}
                                                </span>
                                            </div>

                                            <div className="flex flex-col gap-2 text-sm text-gray-600 dark:text-slate-400">
                                                <div>
                                                    <span className="font-medium">Source:</span>{' '}
                                                    <span className="inline-flex items-center rounded bg-slate-200 px-2 py-0.5 text-xs text-slate-800 dark:bg-slate-700 dark:text-slate-100">
                                                        {resolveSourceName(build.manifest.buildConfig?.sourceId)}
                                                    </span>
                                                </div>
                                                <div>
                                                    <span className="font-medium">Ref Policy:</span>{' '}
                                                    <span className="inline-flex items-center rounded bg-slate-200 px-2 py-0.5 text-xs text-slate-800 dark:bg-slate-700 dark:text-slate-100">
                                                        {getRefPolicyLabel(build.manifest.buildConfig?.refPolicy)}
                                                    </span>
                                                    {build.manifest.buildConfig?.refPolicy === 'fixed' && build.manifest.buildConfig?.fixedRef ? (
                                                        <code className="ml-2 bg-gray-900 text-cyan-400 px-2 py-1 rounded text-xs">
                                                            {build.manifest.buildConfig.fixedRef}
                                                        </code>
                                                    ) : null}
                                                </div>
                                                {build.gitBranch && (
                                                    <div>
                                                        <span className="font-medium">Branch:</span>{' '}
                                                        <code className="bg-gray-900 text-green-400 px-2 py-1 rounded text-xs">
                                                            {build.gitBranch}
                                                        </code>
                                                    </div>
                                                )}

                                                {build.gitCommit && (
                                                    <div>
                                                        <span className="font-medium">Commit:</span>{' '}
                                                        <code className="bg-gray-900 text-cyan-400 px-2 py-1 rounded text-xs">
                                                            {build.gitCommit.slice(0, 7)}
                                                        </code>
                                                    </div>
                                                )}

                                                <div className="text-xs text-gray-500 dark:text-slate-500">
                                                    {formatDate(build.createdAt)}
                                                </div>
                                            </div>
                                        </div>

                                        {/* Duration Info */}
                                        {build.startedAt && build.completedAt && (
                                            <div className="text-right">
                                                <div className="text-2xl font-bold text-gray-900 dark:text-white">
                                                    {Math.round(
                                                        (new Date(build.completedAt).getTime() -
                                                            new Date(build.startedAt).getTime()) /
                                                        1000
                                                    )}
                                                    <span className="text-sm text-gray-600 dark:text-slate-400">s</span>
                                                </div>
                                                <div className="text-xs text-gray-500 dark:text-slate-500 mt-1">Duration</div>
                                            </div>
                                        )}
                                    </div>

                                    {/* Error Message */}
                                    {build.status === 'failed' && build.failureReason && (
                                        <div className="mt-4 flex gap-3 p-3 bg-red-50 dark:bg-red-900/20 rounded border border-red-200 dark:border-red-700">
                                            <AlertCircle className="w-4 h-4 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
                                            <p className="text-sm text-red-700 dark:text-red-200">{build.failureReason}</p>
                                        </div>
                                    )}

                                    {/* Action Buttons */}
                                    <div className="mt-4 flex gap-2">
                                        {build.status === 'running' && (
                                            <button
                                                onClick={(e) => handleCancelBuild(build.id, e)}
                                                className="inline-flex items-center gap-2 px-3 py-1 bg-red-600 text-white text-sm rounded hover:bg-red-700 transition-colors dark:bg-red-600 dark:hover:bg-red-700"
                                            >
                                                Cancel
                                            </button>
                                        )}
                                        <button
                                            onClick={(e) => handleCloneBuild(build.id, e)}
                                            className="inline-flex items-center gap-2 px-3 py-1 border border-blue-300 text-blue-700 text-sm rounded hover:bg-blue-50 transition-colors dark:border-blue-700 dark:text-blue-300 dark:hover:bg-blue-900/20"
                                        >
                                            Clone
                                        </button>
                                        <button
                                            onClick={(e) => handleDeleteBuild(build.id, build.status, e)}
                                            disabled={!canDeleteBuild(build.status)}
                                            className="inline-flex items-center gap-2 px-3 py-1 border border-red-300 text-red-700 text-sm rounded hover:bg-red-50 transition-colors disabled:opacity-50 disabled:cursor-not-allowed dark:border-red-700 dark:text-red-300 dark:hover:bg-red-900/20"
                                            title={canDeleteBuild(build.status) ? 'Delete build' : 'Cannot delete running or queued build'}
                                        >
                                            Delete
                                        </button>
                                    </div>
                                </Link>
                            ))}
                        </div>

                        {/* Pagination */}
                        {totalPages > 1 && (
                            <div className="mt-8 flex items-center justify-center gap-2">
                                <button
                                    onClick={() => handlePageChange(currentPage - 1)}
                                    disabled={currentPage === 1}
                                    className="px-4 py-2 border border-gray-300 dark:border-slate-600 rounded-lg hover:bg-gray-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors dark:bg-slate-800 dark:text-white"
                                >
                                    Previous
                                </button>

                                <div className="flex items-center gap-1">
                                    {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                                        let pageNum
                                        if (totalPages <= 5) {
                                            pageNum = i + 1
                                        } else if (currentPage <= 3) {
                                            pageNum = i + 1
                                        } else if (currentPage >= totalPages - 2) {
                                            pageNum = totalPages - 4 + i
                                        } else {
                                            pageNum = currentPage - 2 + i
                                        }

                                        return (
                                            <button
                                                key={pageNum}
                                                onClick={() => handlePageChange(pageNum)}
                                                className={`px-3 py-1 rounded-lg transition-colors ${currentPage === pageNum
                                                    ? 'bg-blue-600 text-white'
                                                    : 'border border-gray-300 dark:border-slate-600 hover:bg-gray-50 dark:hover:bg-slate-700 dark:bg-slate-800 dark:text-white'
                                                    }`}
                                            >
                                                {pageNum}
                                            </button>
                                        )
                                    })}
                                </div>

                                <button
                                    onClick={() => handlePageChange(currentPage + 1)}
                                    disabled={currentPage === totalPages}
                                    className="px-4 py-2 border border-gray-300 dark:border-slate-600 rounded-lg hover:bg-gray-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors dark:bg-slate-800 dark:text-white"
                                >
                                    Next
                                </button>
                            </div>
                        )}
                    </>
                )}
            </div>
        </div>
    )
}

export default ProjectBuildsPage
