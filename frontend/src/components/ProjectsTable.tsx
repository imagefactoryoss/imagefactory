import { Project } from '@/types'
import { Eye, Pencil, Trash2 } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

interface ProjectsTableProps {
    projects: Project[]
    onEdit: (project: Project) => void
    onDelete: (project: Project) => void
    canEdit?: boolean
    canDelete?: boolean
}

export default function ProjectsTable({
    projects,
    onEdit,
    onDelete,
    canEdit = false,
    canDelete = false,
}: ProjectsTableProps) {
    const navigate = useNavigate()
    const getStatusBadgeColor = (status: string) => {
        switch (status) {
            case 'active':
                return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
            case 'archived':
                return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400'
            case 'suspended':
                return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
            default:
                return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400'
        }
    }

    const formatDate = (dateString: string) => {
        return new Date(dateString).toLocaleDateString()
    }

    const getCreatedAt = (project: Project): string | undefined => {
        const legacy = project as Project & { created_at?: string }
        return project.createdAt || legacy.created_at
    }

    if (projects.length === 0) {
        return (
            <div className="text-center py-8">
                <p className="text-gray-500 dark:text-gray-400">No projects found</p>
            </div>
        )
    }

    return (
        <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                <thead className="bg-gray-50 dark:bg-gray-900">
                    <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Name
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Status
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Builds
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Created
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Actions
                        </th>
                    </tr>
                </thead>
                <tbody className="divide-y divide-gray-200 dark:divide-gray-700 bg-white dark:bg-gray-800">
                    {projects.map((project) => {
                        const createdAt = getCreatedAt(project)
                        return (
                            <tr key={project.id} className="hover:bg-gray-50 dark:hover:bg-gray-700 transition">
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <div>
                                        <p className="text-sm font-medium text-gray-900 dark:text-white">
                                            {project.name}
                                        </p>
                                        {project.description && (
                                            <p className="text-xs text-gray-600 dark:text-gray-400 truncate">
                                                {project.description}
                                            </p>
                                        )}
                                    </div>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusBadgeColor(project.status || 'active')}`}>
                                        {(project.status || 'active').charAt(0).toUpperCase() + (project.status || 'active').slice(1)}
                                    </span>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-400">
                                    {project.buildCount || 0} builds
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-400">
                                    {createdAt ? formatDate(createdAt) : '-'}
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                                    <div className="flex items-center gap-2">
                                        <button
                                            onClick={() => navigate(`/projects/${project.id}`)}
                                            className="inline-flex items-center gap-1 px-3 py-1 rounded-md text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 transition"
                                            title="View project details"
                                        >
                                            <Eye className="h-4 w-4" />
                                            <span className="hidden sm:inline text-xs">View</span>
                                        </button>
                                        {canEdit && (
                                            <button
                                                onClick={() => onEdit(project)}
                                                className="inline-flex items-center gap-1 px-3 py-1 rounded-md text-blue-700 dark:text-blue-400 bg-blue-100 dark:bg-blue-900/30 hover:bg-blue-200 dark:hover:bg-blue-900/50 transition"
                                                title="Edit project"
                                            >
                                                <Pencil className="h-4 w-4" />
                                                <span className="hidden sm:inline text-xs">Edit</span>
                                            </button>
                                        )}
                                        {canDelete && (
                                            <button
                                                onClick={() => onDelete(project)}
                                                className="inline-flex items-center gap-1 px-3 py-1 rounded-md text-red-700 dark:text-red-400 bg-red-100 dark:bg-red-900/30 hover:bg-red-200 dark:hover:bg-red-900/50 transition"
                                                title="Delete project"
                                            >
                                                <Trash2 className="h-4 w-4" />
                                                <span className="hidden sm:inline text-xs">Delete</span>
                                            </button>
                                        )}
                                    </div>
                                </td>
                            </tr>
                        )
                    })}
                </tbody>
            </table>
        </div>
    )
}
