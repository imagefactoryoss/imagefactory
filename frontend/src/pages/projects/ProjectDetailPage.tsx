import { BuildStatsWidget, LastBuildWidget, RecentBuildsWidget } from '@/components/Build/widgets'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import ProjectNotificationTriggerMatrix from '@/components/projects/ProjectNotificationTriggerMatrix'
import ProjectBuildSettingsPanel from '@/components/projects/ProjectBuildSettingsPanel'
import ProjectSourcesPanel from '@/components/projects/ProjectSourcesPanel'
import ProjectWebhooksPanel from '@/components/projects/ProjectWebhooksPanel'
import { ProjectMembersUI } from '@/components/projects/ProjectMembersUI'
import { RegistryAuthList, RegistryAuthModal } from '@/components/projects/registry-auth'
import { RepositoryAuthList, RepositoryAuthModal } from '@/components/projects/repository-auth'
import EditProjectWizardModal from '@/pages/projects/EditProjectWizardModal'
import { buildService } from '@/services/buildService'
import { projectService } from '@/services/projectService'
import { useOperationCapabilitiesStore } from '@/store/operationCapabilities'
import { Project, ProjectSource } from '@/types'
import { RegistryAuth } from '@/types/registryAuth'
import { canCreateBuilds, canDeleteProjects, canEditProjects, canManageMembers, hasPermission } from '@/utils/permissions'
import React, { useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'

type ProjectDetailTab =
    | 'overview'
    | 'sources'
    | 'members'
    | 'builds'
    | 'build-settings'
    | 'webhooks'
    | 'repository-auth'
    | 'registry-auth'
    | 'notifications'

const isProjectDetailTab = (value: string | null): value is ProjectDetailTab => {
    return (
        value === 'overview' ||
        value === 'sources' ||
        value === 'members' ||
        value === 'builds' ||
        value === 'build-settings' ||
        value === 'webhooks' ||
        value === 'repository-auth' ||
        value === 'registry-auth' ||
        value === 'notifications'
    )
}

const ProjectDetailPage: React.FC = () => {
    const { projectId } = useParams<{ projectId: string }>()
    const navigate = useNavigate()
    const [searchParams, setSearchParams] = useSearchParams()
    const [project, setProject] = useState<Project | null>(null)
    const [projectSources, setProjectSources] = useState<ProjectSource[]>([])
    const [projectBuildMetrics, setProjectBuildMetrics] = useState<{ totalBuilds: number; lastBuildAt?: string; lastBuildId?: string } | null>(null)
    const [sourcesLoading, setSourcesLoading] = useState(false)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [activeTab, setActiveTab] = useState<ProjectDetailTab>('overview')
    const [showAuthModal, setShowAuthModal] = useState(false)
    const [authRefreshKey, setAuthRefreshKey] = useState(0)
    const [showRegistryAuthModal, setShowRegistryAuthModal] = useState(false)
    const [registryAuthToEdit, setRegistryAuthToEdit] = useState<RegistryAuth | undefined>(undefined)
    const [registryAuthRefreshKey, setRegistryAuthRefreshKey] = useState(0)
    const [showEditWizard, setShowEditWizard] = useState(false)
    const confirmDialog = useConfirmDialog()
    const operationCapabilities = useOperationCapabilitiesStore((state) => state.capabilities)
    const canUseBuildCapability = operationCapabilities.build
    const canEdit = canEditProjects()
    const canDelete = canDeleteProjects()
    const canManage = canManageMembers()
    const canCreateBuild = canCreateBuilds() && canUseBuildCapability
    const canManageTriggers = hasPermission('build', 'manage_triggers') || hasPermission('build', '*')

    useEffect(() => {
        if (projectId) {
            loadProject()
            loadProjectSources()
            loadProjectBuildMetrics()
            // loadBuildConfigurations() // TODO: Implement build configurations endpoint
        }
    }, [projectId])

    useEffect(() => {
        const tab = searchParams.get('tab')
        if (isProjectDetailTab(tab)) {
            setActiveTab(tab)
        }
    }, [searchParams])

    useEffect(() => {
        if (searchParams.get('edit') === '1' && canEdit) {
            setShowEditWizard(true)
            const next = new URLSearchParams(searchParams)
            next.delete('edit')
            setSearchParams(next, { replace: true })
        }
    }, [searchParams, setSearchParams, canEdit])

    const loadProject = async () => {
        if (!projectId) return

        try {
            setLoading(true)
            setError(null)
            const response = await projectService.getProject(projectId)
            setProject(response)
        } catch (err) {
            const errorMessage = err instanceof Error ? err.message : 'Failed to load project'
            setError(errorMessage)
            toast.error(errorMessage)
        } finally {
            setLoading(false)
        }
    }

    const loadProjectSources = async () => {
        if (!projectId) return

        try {
            setSourcesLoading(true)
            const sources = await projectService.listProjectSources(projectId)
            setProjectSources(sources)
        } catch (err) {
            const errorMessage = err instanceof Error ? err.message : 'Failed to load project sources'
            toast.error(errorMessage)
        } finally {
            setSourcesLoading(false)
        }
    }

    const loadProjectBuildMetrics = async () => {
        if (!projectId) return

        try {
            const response = await buildService.getBuilds({
                projectId,
                page: 1,
                limit: 1,
            })

            const totalBuilds = response.pagination?.total || 0
            const lastBuildAt = response.data?.[0]?.createdAt
            const lastBuildId = response.data?.[0]?.id
            setProjectBuildMetrics({
                totalBuilds,
                lastBuildAt,
                lastBuildId,
            })
        } catch (err) {
            setProjectBuildMetrics(null)
        }
    }

    const handleDeleteProject = async () => {
        if (!project) return

        const confirmed = await confirmDialog({
            title: 'Delete Project',
            message: 'Are you sure you want to delete this project? This action cannot be undone.',
            confirmLabel: 'Delete Project',
            destructive: true,
        })
        if (!confirmed) {
            return
        }

        try {
            await projectService.deleteProject(project.id)
            toast.success('Project deleted successfully!')
            navigate('/projects')
        } catch (error) {
            toast.error('Failed to delete project')
        }
    }

    const formatDate = (dateString: string) => {
        return new Date(dateString).toLocaleDateString()
    }

    const formatDateTime = (dateString: string) => {
        return new Date(dateString).toLocaleString()
    }

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'active':
                return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
            case 'archived':
                return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200'
            case 'suspended':
                return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
            default:
                return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200'
        }
    }

    const activeSources = projectSources.filter((source) => source.isActive)
    const defaultSource = projectSources.find((source) => source.isDefault)
    const totalBuilds = projectBuildMetrics?.totalBuilds ?? project?.buildCount ?? 0
    const lastBuildAt = projectBuildMetrics?.lastBuildAt ?? project?.lastBuildAt
    const lastBuildId = projectBuildMetrics?.lastBuildId

    if (loading) {
        return (
            <div className="flex justify-center items-center h-64">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
            </div>
        )
    }

    if (error || !project) {
        return (
            <div className="text-center py-12">
                <div className="text-red-600 dark:text-red-400 text-lg font-medium">
                    {error || 'Project not found'}
                </div>
                <button
                    onClick={() => navigate('/projects')}
                    className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                >
                    Back to Projects
                </button>
            </div>
        )
    }

    return (
        <div className="px-4 py-6 sm:px-6 lg:px-8">
            {/* Header */}
            <div className="mb-8">
                <div className="flex items-center justify-between">
                    <div>
                        <nav className="flex" aria-label="Breadcrumb">
                            <ol className="flex items-center space-x-2">
                                <li>
                                    <Link to="/projects" className="text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300">
                                        Projects
                                    </Link>
                                </li>
                                <li>
                                    <svg className="flex-shrink-0 h-5 w-5 text-slate-400" viewBox="0 0 20 20" fill="currentColor">
                                        <path fillRule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clipRule="evenodd" />
                                    </svg>
                                </li>
                                <li>
                                    <span className="text-slate-900 dark:text-white font-medium">
                                        {project.name}
                                    </span>
                                </li>
                            </ol>
                        </nav>
                        <div className="flex items-center justify-between">
                            <h1 className="mt-2 text-3xl font-bold text-slate-900 dark:text-white">
                                {project.name}
                            </h1>
                            <div className={`inline-flex items-center px-4 py-2 rounded-full text-sm font-medium ${activeSources.length > 0
                                ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                                : 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
                                }`}>
                                <svg className={`w-5 h-5 mr-2 ${activeSources.length > 0 ? 'text-green-600' : 'text-yellow-600'
                                    }`} fill="currentColor" viewBox="0 0 20 20">
                                    <path fillRule="evenodd" d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
                                </svg>
                                {sourcesLoading ? 'Loading sources...' : `${activeSources.length} active sources`}
                            </div>
                        </div>
                    </div>
                    <div className="flex items-center space-x-3">
                        <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${getStatusColor(project.status)}`}>
                            {project.status}
                        </span>
                    </div>
                </div>
                <div className="mt-4 grid grid-cols-1 md:grid-cols-3 gap-6">
                    <div className="bg-white dark:bg-slate-800 p-6 rounded-lg shadow border border-slate-200 dark:border-slate-700">
                        <div className="flex items-center">
                            <div className="flex-shrink-0">
                                <svg className="h-8 w-8 text-blue-500" viewBox="0 0 20 20" fill="currentColor">
                                    <path fillRule="evenodd" d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
                                </svg>
                            </div>
                            <div className="ml-4">
                                <p className="text-sm font-medium text-slate-600 dark:text-slate-400">
                                    Total Builds
                                </p>
                                <p className="text-2xl font-semibold text-slate-900 dark:text-white">
                                    {totalBuilds}
                                </p>
                            </div>
                        </div>
                    </div>

                    <div className="bg-white dark:bg-slate-800 p-6 rounded-lg shadow border border-slate-200 dark:border-slate-700">
                        <div className="flex items-center">
                            <div className="flex-shrink-0">
                                <svg className="h-8 w-8 text-green-500" viewBox="0 0 20 20" fill="currentColor">
                                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                                </svg>
                            </div>
                            <div className="ml-4">
                                <p className="text-sm font-medium text-slate-600 dark:text-slate-400">
                                    Last Build
                                </p>
                                <p className="text-lg font-semibold text-slate-900 dark:text-white">
                                    {lastBuildAt ? formatDateTime(lastBuildAt) : 'Never'}
                                </p>
                                {lastBuildId && (
                                    <Link
                                        to={`/builds/${lastBuildId}`}
                                        className="inline-flex mt-1 text-xs font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
                                    >
                                        View build details
                                    </Link>
                                )}
                            </div>
                        </div>
                    </div>

                    <div className="bg-white dark:bg-slate-800 p-6 rounded-lg shadow border border-slate-200 dark:border-slate-700">
                        <div className="flex items-center">
                            <div className="flex-shrink-0">
                                <svg className="h-8 w-8 text-purple-500" viewBox="0 0 20 20" fill="currentColor">
                                    <path fillRule="evenodd" d="M4 4a2 2 0 00-2 2v8a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2H4zm2 6a2 2 0 012-2h4a2 2 0 012 2v2a2 2 0 01-2 2H8a2 2 0 01-2-2v-2z" clipRule="evenodd" />
                                </svg>
                            </div>
                            <div className="ml-4">
                                <p className="text-sm font-medium text-slate-600 dark:text-slate-400">
                                    Sources
                                </p>
                                <p className="text-2xl font-semibold text-slate-900 dark:text-white">
                                    {sourcesLoading ? '-' : projectSources.length}
                                </p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Source Details Section */}
            <div className="mb-8">
                {sourcesLoading ? (
                    <div className="bg-slate-50 dark:bg-slate-800/70 border border-slate-200 dark:border-slate-700 rounded-lg p-6">
                        <p className="text-sm text-slate-600 dark:text-slate-300">Loading project sources...</p>
                    </div>
                ) : activeSources.length > 0 ? (
                    <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-6">
                        <div className="flex items-center justify-between gap-4">
                            <div className="flex items-center space-x-4">
                                <div className="flex-shrink-0">
                                    <svg className="h-8 w-8 text-green-600" fill="currentColor" viewBox="0 0 20 20">
                                        <path fillRule="evenodd" d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
                                    </svg>
                                </div>
                                <div>
                                    <h3 className="text-lg font-semibold text-green-800 dark:text-green-200">
                                        Sources Configured
                                    </h3>
                                    <p className="text-sm text-green-700 dark:text-green-300 mt-1">
                                        {activeSources.length} active source{activeSources.length === 1 ? '' : 's'} available for build binding
                                    </p>
                                </div>
                            </div>
                            <div className="flex items-center space-x-4">
                                <button
                                    type="button"
                                    onClick={() => setActiveTab('sources')}
                                    className="px-3 py-2 rounded-md border border-green-300 bg-white text-green-800 text-sm font-medium hover:bg-green-50 dark:border-green-700 dark:bg-slate-900 dark:text-green-200 dark:hover:bg-green-900/20"
                                >
                                    Manage Sources
                                </button>
                            </div>
                        </div>
                        <div className="mt-4 pt-4 border-t border-green-200 dark:border-green-700">
                            {defaultSource ? (
                                <div className="text-sm text-green-800 dark:text-green-200">
                                    Default source:
                                    <span className="ml-2 font-semibold">{defaultSource.name}</span>
                                    <span className="ml-2 font-mono break-all">{defaultSource.repositoryUrl}</span>
                                </div>
                            ) : (
                                <p className="text-sm text-green-800 dark:text-green-200">No default source selected yet.</p>
                            )}
                        </div>
                    </div>
                ) : (
                    <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-6">
                        <div className="flex items-center justify-between">
                            <div className="flex items-center space-x-4">
                                <div className="flex-shrink-0">
                                    <svg className="h-8 w-8 text-yellow-600" fill="currentColor" viewBox="0 0 20 20">
                                        <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                                    </svg>
                                </div>
                                <div>
                                    <h3 className="text-lg font-semibold text-yellow-800 dark:text-yellow-200">
                                        No Sources Configured
                                    </h3>
                                    <p className="text-sm text-yellow-700 dark:text-yellow-300 mt-1">
                                        Configure at least one project source before creating builds
                                    </p>
                                </div>
                            </div>
                            <button
                                type="button"
                                onClick={() => setActiveTab('sources')}
                                className="px-3 py-2 rounded-md border border-yellow-300 bg-white text-yellow-800 text-sm font-medium hover:bg-yellow-50 dark:border-yellow-700 dark:bg-slate-900 dark:text-yellow-200 dark:hover:bg-yellow-900/20"
                            >
                                Add Source
                            </button>
                        </div>
                    </div>
                )}
            </div>

            {/* Tab Navigation */}
            <div className="mb-6 border-b border-slate-200 dark:border-slate-700">
                <div className="flex space-x-8">
                    <button
                        onClick={() => setActiveTab('overview')}
                        className={`py-4 px-1 border-b-2 font-medium text-sm ${activeTab === 'overview'
                            ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                            : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:border-slate-300 dark:hover:border-slate-600'
                            }`}
                    >
                        Overview
                    </button>
                    <button
                        onClick={() => setActiveTab('builds')}
                        className={`py-4 px-1 border-b-2 font-medium text-sm ${activeTab === 'builds'
                            ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                            : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:border-slate-300 dark:hover:border-slate-600'
                            }`}
                    >
                        Builds
                    </button>
                    <button
                        onClick={() => setActiveTab('sources')}
                        className={`py-4 px-1 border-b-2 font-medium text-sm ${activeTab === 'sources'
                            ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                            : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:border-slate-300 dark:hover:border-slate-600'
                            }`}
                    >
                        Sources
                    </button>
                    <button
                        onClick={() => setActiveTab('members')}
                        className={`py-4 px-1 border-b-2 font-medium text-sm ${activeTab === 'members'
                            ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                            : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:border-slate-300 dark:hover:border-slate-600'
                            }`}
                    >
                        Members
                    </button>
                    <button
                        onClick={() => setActiveTab('build-settings')}
                        className={`py-4 px-1 border-b-2 font-medium text-sm ${activeTab === 'build-settings'
                            ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                            : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:border-slate-300 dark:hover:border-slate-600'
                            }`}
                    >
                        Build Config
                    </button>
                    <button
                        onClick={() => setActiveTab('webhooks')}
                        className={`py-4 px-1 border-b-2 font-medium text-sm ${activeTab === 'webhooks'
                            ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                            : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:border-slate-300 dark:hover:border-slate-600'
                            }`}
                    >
                        Webhooks
                    </button>
                    <button
                        onClick={() => setActiveTab('repository-auth')}
                        className={`py-4 px-1 border-b-2 font-medium text-sm ${activeTab === 'repository-auth'
                            ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                            : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:border-slate-300 dark:hover:border-slate-600'
                            }`}
                    >
                        Repository Auth
                    </button>
                    <button
                        onClick={() => setActiveTab('registry-auth')}
                        className={`py-4 px-1 border-b-2 font-medium text-sm ${activeTab === 'registry-auth'
                            ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                            : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:border-slate-300 dark:hover:border-slate-600'
                            }`}
                    >
                        Registry Auth
                    </button>
                    <button
                        onClick={() => setActiveTab('notifications')}
                        className={`py-4 px-1 border-b-2 font-medium text-sm ${activeTab === 'notifications'
                            ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                            : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:border-slate-300 dark:hover:border-slate-600'
                            }`}
                    >
                        Notifications
                    </button>
                </div>
            </div>

            {/* Overview Tab */}
            {activeTab === 'overview' && (
                <>
                    {/* Project Overview */}
                    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                        {/* Project Info Card */}
                        <div className="lg:col-span-2 bg-white dark:bg-slate-800 rounded-lg shadow border border-slate-200 dark:border-slate-700 p-6">
                            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Project Overview</h3>
                            <div className="space-y-6">
                                {/* Project Details */}
                                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                    <div>
                                        <label className="block text-sm font-medium text-gray-600 dark:text-slate-400 mb-1">
                                            Project Name
                                        </label>
                                        <p className="text-gray-900 dark:text-white">{project.name}</p>
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-600 dark:text-slate-400 mb-1">
                                            Project ID
                                        </label>
                                        <p className="text-gray-900 dark:text-white font-mono text-sm">{projectId}</p>
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-600 dark:text-slate-400 mb-1">
                                            Status
                                        </label>
                                        <p className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${getStatusColor(project.status)}`}>
                                            {project.status}
                                        </p>
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-600 dark:text-slate-400 mb-1">
                                            Created At
                                        </label>
                                        <p className="text-gray-900 dark:text-white">{formatDate(project.createdAt)}</p>
                                    </div>
                                </div>

                                {/* Project Description */}
                                {project.description && (
                                    <div>
                                        <label className="block text-sm font-medium text-gray-600 dark:text-slate-400 mb-1">
                                            Description
                                        </label>
                                        <p className="text-gray-900 dark:text-white">{project.description}</p>
                                    </div>
                                )}

                                {/* Source Defaults */}
                                {projectSources.length > 0 && (
                                    <div>
                                        <label className="block text-sm font-medium text-gray-600 dark:text-slate-400 mb-1">
                                            Default Source
                                        </label>
                                        <div className="flex items-center space-x-2">
                                            <span className="text-gray-900 dark:text-white font-medium">
                                                {defaultSource?.name || 'None'}
                                            </span>
                                            {defaultSource && (
                                                <span className="text-slate-500 dark:text-slate-400 font-mono text-xs break-all">
                                                    {defaultSource.repositoryUrl}
                                                </span>
                                            )}
                                        </div>
                                    </div>
                                )}
                            </div>

                            {/* Actions */}
                            {(canEdit || canDelete) && (
                                <div className="mt-6 flex justify-end gap-3">
                                    {canEdit ? (
                                        <button
                                            onClick={() => setShowEditWizard(true)}
                                            className="px-4 py-2 border border-blue-300 dark:border-blue-700 rounded-md shadow-sm text-sm font-medium text-blue-700 dark:text-blue-300 bg-white dark:bg-slate-800 hover:bg-blue-50 dark:hover:bg-blue-900/20 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                                        >
                                            Edit Basic Info
                                        </button>
                                    ) : null}
                                    {canDelete ? (
                                    <button
                                        onClick={handleDeleteProject}
                                        className="px-4 py-2 border border-red-300 dark:border-red-600 rounded-md shadow-sm text-sm font-medium text-red-700 dark:text-red-300 bg-white dark:bg-slate-800 hover:bg-red-50 dark:hover:bg-red-900/20 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
                                    >
                                        Delete Project
                                    </button>
                                    ) : null}
                                </div>
                            )}
                        </div>

                        {/* Quick Stats Card */}
                        <div className="bg-white dark:bg-slate-800 rounded-lg shadow border border-slate-200 dark:border-slate-700 p-6">
                            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Quick Stats</h3>
                            <div className="space-y-4">
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-600 dark:text-slate-400 text-sm">Total Builds</span>
                                    <span className="text-gray-900 dark:text-white font-semibold">{totalBuilds}</span>
                                </div>
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-600 dark:text-slate-400 text-sm">Last Build</span>
                                    <span className="text-gray-900 dark:text-white font-semibold">
                                        {lastBuildAt ? formatDateTime(lastBuildAt) : 'Never'}
                                    </span>
                                </div>
                                {lastBuildId && (
                                    <div className="flex items-center justify-between">
                                        <span className="text-gray-600 dark:text-slate-400 text-sm">Build Detail</span>
                                        <Link
                                            to={`/builds/${lastBuildId}`}
                                            className="text-sm font-semibold text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
                                        >
                                            Open details
                                        </Link>
                                    </div>
                                )}
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-600 dark:text-slate-400 text-sm">Sources</span>
                                    <span className="text-gray-900 dark:text-white font-semibold">{projectSources.length}</span>
                                </div>
                            </div>

                            {/* Quick Actions */}
                            <div className="mt-6 space-y-3">
                                <Link
                                    to={`/projects/${projectId}/builds`}
                                    className="block w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium text-center"
                                >
                                    View Builds
                                </Link>
                                {canCreateBuild && (
                                    <Link
                                        to={`/builds/new?projectId=${projectId}`}
                                        className="block w-full px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors text-sm font-medium text-center"
                                    >
                                        New Build
                                    </Link>
                                )}
                            </div>
                        </div>
                    </div>
                </>
            )}

            {/* Members Tab */}
            {activeTab === 'sources' && projectId && (
                <ProjectSourcesPanel projectId={projectId} canEdit={canEdit} />
            )}

            {/* Members Tab */}
            {activeTab === 'members' && projectId && (
                <div className="bg-white dark:bg-slate-800 rounded-lg shadow border border-slate-200 dark:border-slate-700">
                    <ProjectMembersUI
                        projectId={projectId}
                        canManageMembers={canManage}
                    />
                </div>
            )}

            {/* Builds Tab */}
            {activeTab === 'builds' && projectId && (
                <div className="space-y-6">
                    <div className="flex justify-between items-center">
                        <p className="text-slate-600 dark:text-slate-400">
                            Monitor and manage all builds for this project
                        </p>
                        <div className="flex space-x-2">
                            {canCreateBuild && (
                                <Link
                                    to={`/builds/new?projectId=${projectId}`}
                                    className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors text-sm font-medium"
                                >
                                    New Build
                                </Link>
                            )}
                            <Link
                                to={`/projects/${projectId}/builds`}
                                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium"
                            >
                                View All Builds
                            </Link>
                        </div>
                    </div>

                    {/* Build Stats Grid */}
                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                        <BuildStatsWidget projectId={projectId} />
                        <LastBuildWidget projectId={projectId} />
                    </div>

                    {/* Recent Builds */}
                    <RecentBuildsWidget projectId={projectId} limit={10} />
                </div>
            )}

            {activeTab === 'build-settings' && projectId && (
                <ProjectBuildSettingsPanel projectId={projectId} canEdit={canEdit} />
            )}

            {activeTab === 'webhooks' && projectId && (
                <ProjectWebhooksPanel projectId={projectId} canManage={canManageTriggers} />
            )}

            {/* Repository Auth Tab */}
            {activeTab === 'repository-auth' && projectId && (
                <div className="space-y-6">
                    <div className="flex justify-between items-center">
                        <div>
                            <h2 className="text-xl font-semibold text-slate-900 dark:text-white">
                                Repository Authentication
                            </h2>
                            <p className="text-slate-600 dark:text-slate-400 mt-1">
                                Manage authentication methods for accessing your Git repository
                            </p>
                        </div>
                        <button
                            onClick={() => setShowAuthModal(true)}
                            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium"
                        >
                            Add Authentication
                        </button>
                    </div>

                    <RepositoryAuthList
                        projectId={projectId}
                        repoUrl={defaultSource?.repositoryUrl}
                        refreshKey={authRefreshKey}
                    />
                </div>
            )}

            {activeTab === 'registry-auth' && projectId && (
                <div className="space-y-6">
                    <div className="flex justify-between items-center">
                        <div>
                            <h2 className="text-xl font-semibold text-slate-900 dark:text-white">
                                Registry Authentication
                            </h2>
                            <p className="text-slate-600 dark:text-slate-400 mt-1">
                                Manage project and tenant registry credentials for image build methods
                            </p>
                        </div>
                        <button
                            onClick={() => {
                                setRegistryAuthToEdit(undefined)
                                setShowRegistryAuthModal(true)
                            }}
                            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium"
                        >
                            Add Registry Auth
                        </button>
                    </div>

                    <RegistryAuthList
                        projectId={projectId}
                        refreshKey={registryAuthRefreshKey}
                        onEdit={(auth) => {
                            setRegistryAuthToEdit(auth)
                            setShowRegistryAuthModal(true)
                        }}
                    />
                </div>
            )}

            {activeTab === 'notifications' && projectId && (
                <ProjectNotificationTriggerMatrix
                    projectId={projectId}
                    canEdit={canEdit}
                />
            )}

            {/* Repository Auth Modal */}
            {projectId && (
                <RepositoryAuthModal
                    projectId={projectId}
                    isOpen={showAuthModal}
                    onClose={() => setShowAuthModal(false)}
                    onSuccess={() => {
                        setAuthRefreshKey(prev => prev + 1)
                    }}
                />
            )}

            {projectId && (
                <RegistryAuthModal
                    projectId={projectId}
                    isOpen={showRegistryAuthModal}
                    authToEdit={registryAuthToEdit}
                    onClose={() => {
                        setShowRegistryAuthModal(false)
                        setRegistryAuthToEdit(undefined)
                    }}
                    onSuccess={() => {
                        setRegistryAuthToEdit(undefined)
                        setRegistryAuthRefreshKey(prev => prev + 1)
                    }}
                />
            )}

            {projectId && (
                <EditProjectWizardModal
                    isOpen={showEditWizard}
                    projectId={projectId}
                    project={project}
                    canManageMembers={canManage}
                    restrictToBasics
                    onClose={() => setShowEditWizard(false)}
                    onProjectUpdated={(updatedProject) => {
                        setProject(updatedProject)
                    }}
                />
            )}
        </div>
    )
}

export default ProjectDetailPage
