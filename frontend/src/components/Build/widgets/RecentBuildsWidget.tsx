import { buildService } from '@/services/buildService'
import { useTenantStore } from '@/store/tenant'
import { Build, BuildStatus } from '@/types'
import { AlertCircle, CheckCircle, Clock, ExternalLink } from 'lucide-react'
import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

interface RecentBuildsWidgetProps {
    projectId: string
    limit?: number
}

export const RecentBuildsWidget: React.FC<RecentBuildsWidgetProps> = ({ projectId, limit = 5 }) => {
    const [builds, setBuilds] = useState<Build[]>([])
    const [loading, setLoading] = useState(true)
    const { selectedTenantId } = useTenantStore()

    useEffect(() => {
        loadBuilds()
    }, [projectId, selectedTenantId])

    const loadBuilds = async () => {
        try {
            setLoading(true)
            const response = await buildService.getBuilds({
                projectId,
                limit,
                page: 1,
                tenantId: selectedTenantId || undefined,
            })
            setBuilds(response.data.slice(0, limit))
        } catch (error) {
            console.error('Failed to load recent builds:', error)
        } finally {
            setLoading(false)
        }
    }

    const getStatusIcon = (status: BuildStatus) => {
        switch (status) {
            case 'completed':
                return <CheckCircle className="w-4 h-4 text-green-600" />
            case 'failed':
                return <AlertCircle className="w-4 h-4 text-red-600" />
            case 'running':
                return <Clock className="w-4 h-4 text-blue-600 animate-spin" />
            default:
                return <Clock className="w-4 h-4 text-gray-600" />
        }
    }

    const getStatusColor = (status: BuildStatus) => {
        switch (status) {
            case 'completed':
                return 'text-green-700'
            case 'failed':
                return 'text-red-700'
            case 'running':
                return 'text-blue-700'
            case 'queued':
                return 'text-yellow-700'
            default:
                return 'text-gray-700'
        }
    }

    if (loading) {
        return (
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow p-6 border border-slate-200 dark:border-slate-700">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Recent Builds</h3>
                <div className="space-y-3">
                    {[...Array(3)].map((_, i) => (
                        <div key={i} className="h-12 bg-gray-100 dark:bg-slate-700 rounded animate-pulse" />
                    ))}
                </div>
            </div>
        )
    }

    if (builds.length === 0) {
        return (
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow p-6 border border-slate-200 dark:border-slate-700">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Recent Builds</h3>
                <p className="text-gray-500 dark:text-slate-400 text-center py-6">No builds yet</p>
            </div>
        )
    }

    return (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow p-6 border border-slate-200 dark:border-slate-700">
            <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Recent Builds</h3>
                <Link
                    to={`/projects/${projectId}/builds`}
                    className="text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 font-medium"
                >
                    View All
                </Link>
            </div>

            <div className="space-y-3">
                {builds.map((build) => (
                    <Link
                        key={build.id}
                        to={`/builds/${build.id}`}
                        className="flex items-center gap-3 p-3 border border-gray-200 dark:border-slate-600 rounded-lg hover:bg-gray-50 dark:hover:bg-slate-700 transition-colors group"
                    >
                        <div className="flex-shrink-0">{getStatusIcon(build.status as BuildStatus)}</div>
                        <div className="flex-1 min-w-0">
                            <p className={`text-sm font-medium ${getStatusColor(build.status as BuildStatus)}`}>
                                Build #{build.buildNumber || build.id.slice(-8)}
                            </p>
                            <p className="text-xs text-gray-500 dark:text-slate-400">
                                {build.gitBranch && <span>{build.gitBranch}</span>}
                                {build.gitBranch && build.gitCommit && <span> • </span>}
                                {build.gitCommit && (
                                    <code className="text-gray-600 dark:text-gray-400">{build.gitCommit.slice(0, 7)}</code>
                                )}
                            </p>
                        </div>
                        <div className="flex-shrink-0 text-gray-400 group-hover:text-gray-600 dark:text-slate-500 dark:group-hover:text-slate-400">
                            <ExternalLink className="w-4 h-4" />
                        </div>
                    </Link>
                ))}
            </div>
        </div>
    )
}

export default RecentBuildsWidget
