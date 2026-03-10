import { Build } from '@/types'
import { AlertCircle, CheckCircle, Clock, Play, Square } from 'lucide-react'

interface BuildsTableProps {
    builds: Build[]
    onViewDetails: (build: Build) => void
    onRetry?: (build: Build) => void
}

export default function BuildsTable({
    builds,
    onViewDetails,
    onRetry,
}: BuildsTableProps) {
    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'completed':
            case 'success':
                return <CheckCircle className="h-5 w-5 text-green-500" />
            case 'failed':
                return <AlertCircle className="h-5 w-5 text-red-500" />
            case 'running':
            case 'in_progress':
                return <Play className="h-5 w-5 text-blue-500 animate-pulse" />
            case 'cancelled':
                return <Square className="h-5 w-5 text-gray-500" />
            default:
                return <Clock className="h-5 w-5 text-gray-500" />
        }
    }

    const getStatusBadgeColor = (status: string) => {
        switch (status) {
            case 'completed':
            case 'success':
                return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
            case 'failed':
                return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
            case 'running':
            case 'in_progress':
                return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400'
            case 'cancelled':
                return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400'
            default:
                return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400'
        }
    }

    const formatDate = (dateString: string) => {
        return new Date(dateString).toLocaleDateString()
    }

    const formatTime = (dateString: string) => {
        return new Date(dateString).toLocaleTimeString()
    }

    const formatDuration = (startTime: string, endTime?: string) => {
        if (!endTime) return '-'
        const start = new Date(startTime).getTime()
        const end = new Date(endTime).getTime()
        const durationMs = end - start
        const seconds = Math.floor(durationMs / 1000)
        const minutes = Math.floor(seconds / 60)
        const hours = Math.floor(minutes / 60)

        if (hours > 0) {
            return `${hours}h ${minutes % 60}m`
        } else if (minutes > 0) {
            return `${minutes}m ${seconds % 60}s`
        } else {
            return `${seconds}s`
        }
    }

    if (builds.length === 0) {
        return (
            <div className="text-center py-8">
                <p className="text-gray-500 dark:text-gray-400">No builds found</p>
            </div>
        )
    }

    return (
        <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                <thead className="bg-gray-50 dark:bg-gray-900">
                    <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Build ID
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Status
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Commit
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Started
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Duration
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Actions
                        </th>
                    </tr>
                </thead>
                <tbody className="divide-y divide-gray-200 dark:divide-gray-700 bg-white dark:bg-gray-800">
                    {builds.map((build) => (
                        <tr key={build.id} className="hover:bg-gray-50 dark:hover:bg-gray-700 transition">
                            <td className="px-6 py-4 whitespace-nowrap">
                                <div className="flex items-center gap-3">
                                    {getStatusIcon(build.status)}
                                    <div>
                                        <p className="text-sm font-medium text-gray-900 dark:text-white">
                                            Build {build.id?.slice(0, 8)}
                                        </p>
                                        <p className="text-xs text-gray-600 dark:text-gray-400">
                                            {build.manifest?.name || 'Unnamed Build'}
                                        </p>
                                    </div>
                                </div>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap">
                                <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusBadgeColor(build.status || 'pending')}`}>
                                    {(build.status || 'pending').charAt(0).toUpperCase() + (build.status || 'pending').slice(1)}
                                </span>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-600 dark:text-gray-400">
                                {build.manifest?.baseImage ? (
                                    <>
                                        <p>{build.manifest.baseImage.slice(0, 20)}...</p>
                                        <p className="text-xs text-gray-500 dark:text-gray-500">{build.manifest.type}</p>
                                    </>
                                ) : (
                                    '-'
                                )}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap">
                                <div className="text-sm text-gray-600 dark:text-gray-400">
                                    {build.createdAt && (
                                        <>
                                            <p>{formatDate(build.createdAt)}</p>
                                            <p className="text-xs text-gray-500 dark:text-gray-500">{formatTime(build.createdAt)}</p>
                                        </>
                                    )}
                                </div>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-400">
                                {formatDuration(build.createdAt || '', build.completedAt)}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                                <div className="flex items-center gap-2">
                                    <button
                                        onClick={() => onViewDetails(build)}
                                        className="inline-flex items-center px-3 py-1 rounded-md text-blue-700 dark:text-blue-400 bg-blue-100 dark:bg-blue-900/30 hover:bg-blue-200 dark:hover:bg-blue-900/50 transition"
                                        title="View build details"
                                    >
                                        <span className="hidden sm:inline text-xs">View</span>
                                    </button>
                                    {onRetry && (build.status === 'failed' || build.status === 'cancelled') && (
                                        <button
                                            onClick={() => onRetry(build)}
                                            className="inline-flex items-center px-3 py-1 rounded-md text-green-700 dark:text-green-400 bg-green-100 dark:bg-green-900/30 hover:bg-green-200 dark:hover:bg-green-900/50 transition"
                                            title="Retry build"
                                        >
                                            <span className="hidden sm:inline text-xs">Retry</span>
                                        </button>
                                    )}
                                </div>
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    )
}
