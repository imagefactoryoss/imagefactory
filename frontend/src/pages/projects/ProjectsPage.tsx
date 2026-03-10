import ProjectsTable from '@/components/ProjectsTable'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import { projectService } from '@/services/projectService'
import { useTenantStore } from '@/store/tenant'
import { Project } from '@/types'
import { useCanCreateProject } from '@/hooks/useAccess'
import { canDeleteProjects, canEditProjects } from '@/utils/permissions'
import { AlertCircle, FolderOpen, Loader, Plus, Search } from 'lucide-react'
import React, { useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import { useNavigate } from 'react-router-dom'
import CreateProjectModal from './CreateProjectModal'

const ProjectsPage: React.FC = () => {
    const navigate = useNavigate()
    const canCreateProject = useCanCreateProject()
    const { selectedTenantId } = useTenantStore()
    const [projects, setProjects] = useState<Project[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [searchTerm, setSearchTerm] = useState('')
    const [page, setPage] = useState(1)
    const [total, setTotal] = useState(0)
    const limit = 20
    const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
    const confirmDialog = useConfirmDialog()

    // Fetch projects
    const loadProjects = async () => {
        if (!selectedTenantId) {
            setError('No tenant selected')
            setLoading(false)
            return
        }

        try {
            setLoading(true)
            setError(null)
            const response = await projectService.getProjects({
                tenantId: selectedTenantId,
                search: searchTerm,
                page,
                limit,
                status: ['active']
            })
            setProjects(response.data || [])
            setTotal(response.pagination?.total || 0)
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to load projects'
            setError(message)
            toast.error(message)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        loadProjects()
    }, [page, limit, searchTerm, selectedTenantId])

    const handleProjectCreated = (project: Project) => {
        setIsCreateModalOpen(false)
        toast.success(`Project "${project.name}" created successfully`)
        setPage(1)
        loadProjects()
    }

    // Handle delete project
    const handleDeleteProject = async (project: Project) => {
        const confirmed = await confirmDialog({
            title: 'Delete Project',
            message: `Are you sure you want to delete the project "${project.name}"? This action cannot be undone.`,
            confirmLabel: 'Delete Project',
            destructive: true,
        })
        if (!confirmed) {
            return
        }

        try {
            await projectService.deleteProject(project.id)
            toast.success('Project deleted successfully')
            setProjects(projects.filter(p => p.id !== project.id))
            loadProjects()
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to delete project'
            toast.error(message)
        }
    }

    if (!selectedTenantId) {
        return (
            <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center">
                <div className="text-center">
                    <AlertCircle className="mx-auto h-12 w-12 text-red-500 mb-4" />
                    <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
                        No Tenant Selected
                    </h1>
                    <p className="text-gray-600 dark:text-gray-400">
                        Please select a tenant from the tenants page first.
                    </p>
                </div>
            </div>
        )
    }

    return (
        <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
                {/* Header */}
                <div className="flex justify-between items-center mb-8">
                    <div>
                        <div className="flex items-center gap-3 mb-2">
                            <FolderOpen className="h-8 w-8 text-blue-600 dark:text-blue-400" />
                            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
                                Projects
                            </h1>
                        </div>
                        <p className="text-gray-600 dark:text-gray-400">
                            Manage your container build projects
                        </p>
                    </div>
                    {canCreateProject && (
                        <button
                            onClick={() => {
                                setIsCreateModalOpen(true)
                            }}
                            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
                        >
                            <Plus className="h-5 w-5" />
                            Create Project
                        </button>
                    )}
                </div>

                {/* Search */}
                <div className="mb-6">
                    <div className="relative">
                        <Search className="absolute left-3 top-3 h-5 w-5 text-gray-400" />
                        <input
                            type="text"
                            placeholder="Search projects by name..."
                            value={searchTerm}
                            onChange={(e) => {
                                setSearchTerm(e.target.value)
                                setPage(1)
                            }}
                            className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                    </div>
                </div>

                {/* Error State */}
                {error && (
                    <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                        <div className="flex gap-3">
                            <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
                            <div>
                                <p className="text-sm font-medium text-red-800 dark:text-red-400">
                                    {error}
                                </p>
                            </div>
                        </div>
                    </div>
                )}

                {/* Loading State */}
                {loading && (
                    <div className="flex justify-center items-center py-12">
                        <Loader className="h-8 w-8 text-blue-600 animate-spin" />
                    </div>
                )}

                {/* Projects Table */}
                {!loading && projects.length === 0 ? (
                    searchTerm ? (
                        <div className="text-center py-12">
                            <p className="text-gray-600 dark:text-gray-400">
                                No projects found matching your search.
                            </p>
                        </div>
                    ) : (
                        <div className="rounded-2xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-sm overflow-hidden">
                            <div className="px-6 py-5 border-b border-slate-200 dark:border-slate-700 bg-gradient-to-r from-blue-50 via-cyan-50 to-emerald-50 dark:from-slate-800 dark:via-slate-800 dark:to-slate-700">
                                <h2 className="text-xl font-semibold text-slate-900 dark:text-slate-100">Start Your First Build Project</h2>
                                <p className="mt-1 text-sm text-slate-700 dark:text-slate-300">
                                    Projects define source repositories, build settings, and execution history. Create one project to begin your build pipeline.
                                </p>
                            </div>

                            <div className="p-6 grid grid-cols-1 lg:grid-cols-2 gap-6">
                                <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/60 p-4">
                                    <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-3">Project Setup Flow</h3>
                                    <div className="space-y-3 text-sm text-slate-700 dark:text-slate-300">
                                        <div className="flex items-start gap-3">
                                            <span className="mt-0.5 inline-flex h-6 w-6 items-center justify-center rounded-full bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300 font-semibold">1</span>
                                            <div>
                                                <p className="font-medium text-slate-900 dark:text-slate-100">Create project metadata</p>
                                                <p>Name, description, and default branch establish your build scope.</p>
                                            </div>
                                        </div>
                                        <div className="flex items-start gap-3">
                                            <span className="mt-0.5 inline-flex h-6 w-6 items-center justify-center rounded-full bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300 font-semibold">2</span>
                                            <div>
                                                <p className="font-medium text-slate-900 dark:text-slate-100">Connect source + repository auth</p>
                                                <p>Link your Git repository and private access credentials when needed.</p>
                                            </div>
                                        </div>
                                        <div className="flex items-start gap-3">
                                            <span className="mt-0.5 inline-flex h-6 w-6 items-center justify-center rounded-full bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300 font-semibold">3</span>
                                            <div>
                                                <p className="font-medium text-slate-900 dark:text-slate-100">Choose build configuration mode</p>
                                                <p>Use UI-managed config or repository-driven `image-factory.yaml`.</p>
                                            </div>
                                        </div>
                                    </div>
                                </div>

                                <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/60 p-4">
                                    <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-3">How Builds Work</h3>
                                    <div className="space-y-3 text-sm text-slate-700 dark:text-slate-300">
                                        <p>
                                            Each build runs from the selected source ref and configuration, then emits execution logs and artifact evidence.
                                        </p>
                                        <div className="grid grid-cols-3 gap-2 text-xs font-medium">
                                            <div className="rounded-md bg-cyan-50 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-300 px-2 py-1 text-center border border-cyan-200 dark:border-cyan-800">Layers</div>
                                            <div className="rounded-md bg-violet-50 text-violet-700 dark:bg-violet-900/30 dark:text-violet-300 px-2 py-1 text-center border border-violet-200 dark:border-violet-800">SBOM</div>
                                            <div className="rounded-md bg-rose-50 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300 px-2 py-1 text-center border border-rose-200 dark:border-rose-800">Vulns</div>
                                        </div>
                                        <p className="text-xs text-slate-600 dark:text-slate-400">
                                            After project setup, create builds from the project details page and monitor status in the Builds section.
                                        </p>
                                    </div>
                                </div>
                            </div>

                            <div className="px-6 pb-6 flex flex-wrap items-center gap-3">
                                {canCreateProject ? (
                                    <button
                                        onClick={() => {
                                            setIsCreateModalOpen(true)
                                        }}
                                        className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
                                    >
                                        Create First Project
                                    </button>
                                ) : null}
                                <button
                                    onClick={() => navigate('/builds')}
                                    className="px-4 py-2 border border-slate-300 dark:border-slate-600 text-slate-800 dark:text-slate-200 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700 transition"
                                >
                                    View Build History
                                </button>
                            </div>
                        </div>
                    )
                ) : (
                    <>
                        <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
                            <ProjectsTable
                                projects={projects}
                                onEdit={(project) => {
                                    navigate(`/projects/${project.id}?edit=1`)
                                }}
                                onDelete={handleDeleteProject}
                                canEdit={canEditProjects()}
                                canDelete={canDeleteProjects()}
                            />
                        </div>

                        {/* Pagination */}
                        {total > limit && (
                            <div className="mt-6 flex justify-between items-center">
                                <p className="text-sm text-gray-600 dark:text-gray-400">
                                    Showing {((page - 1) * limit) + 1} to {Math.min(page * limit, total)} of {total} projects
                                </p>
                                <div className="flex gap-2">
                                    <button
                                        onClick={() => setPage(Math.max(1, page - 1))}
                                        disabled={page === 1}
                                        className="px-4 py-2 border border-gray-300 dark:border-gray-700 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100 dark:hover:bg-gray-800 transition"
                                    >
                                        Previous
                                    </button>
                                    <button
                                        onClick={() => setPage(page + 1)}
                                        disabled={page * limit >= total}
                                        className="px-4 py-2 border border-gray-300 dark:border-gray-700 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100 dark:hover:bg-gray-800 transition"
                                    >
                                        Next
                                    </button>
                                </div>
                            </div>
                        )}
                    </>
                )}
            </div>

            {/* Modals */}
            <CreateProjectModal
                isOpen={isCreateModalOpen}
                onClose={() => setIsCreateModalOpen(false)}
                onCreated={handleProjectCreated}
            />
        </div>
    )
}

export default ProjectsPage
