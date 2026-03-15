import { buildService } from '@/services/buildService'
import { useTenantStore } from '@/store/tenant'
import { Build, BuildStatus } from '@/types'
import { AlertCircle, CheckCircle, Clock, Link as LinkIcon } from 'lucide-react'
import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

interface LastBuildWidgetProps {
    projectId: string
}

export const LastBuildWidget: React.FC<LastBuildWidgetProps> = ({ projectId }) => {
    const [lastBuild, setLastBuild] = useState<Build | null>(null)
    const [loading, setLoading] = useState(true)
    const { selectedTenantId } = useTenantStore()

    useEffect(() => {
        loadLastBuild()
    }, [projectId, selectedTenantId])

    const loadLastBuild = async () => {
        try {
            setLoading(true)
            const response = await buildService.getBuilds({
                projectId,
                limit: 1,
                page: 1,
                tenantId: selectedTenantId || undefined,
            })
            if (response.data.length > 0) {
                setLastBuild(response.data[0])
            }
        } catch (error) {
            console.error('Failed to load last build:', error)
        } finally {
            setLoading(false)
        }
    }

    const getStatusIcon = (status: BuildStatus) => {
        switch (status) {
            case 'completed':
                return <CheckCircle className="w-6 h-6 text-green-600" />
            case 'failed':
                return <AlertCircle className="w-6 h-6 text-red-600" />
            case 'running':
                return <Clock className="w-6 h-6 text-blue-600 animate-spin" />
            case 'queued':
                return <Clock className="w-6 h-6 text-yellow-600" />
            default:
                return <Clock className="w-6 h-6 text-gray-600" />
        }
    }



    const formatDate = (date: string | Date): string => {
        const d = new Date(date)
        return d.toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
        })
    }

    const getDuration = (startedAt?: string, completedAt?: string): string | null => {
        if (!startedAt || !completedAt) return null
        const diff = new Date(completedAt).getTime() - new Date(startedAt).getTime()
        const seconds = Math.round(diff / 1000)
        if (seconds < 60) return `${seconds}s`
        if (seconds < 3600) return `${Math.round(seconds / 60)}m`
        return `${Math.round(seconds / 3600)}h`
    }

    if (loading) {
        return (
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow p-6 border border-slate-200 dark:border-slate-700">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Last Build</h3>
                <div className="h-32 bg-gray-100 dark:bg-slate-700 rounded animate-pulse" />
            </div>
        )
    }

    if (!lastBuild) {
        return (
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow p-6 border border-slate-200 dark:border-slate-700">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Last Build</h3>
                <p className="text-gray-500 dark:text-slate-400 text-center py-8">No builds yet</p>
            </div>
        )
    }

    const getStatusBgColor = (status: BuildStatus) => {
        switch (status) {
            case 'completed':
                return 'bg-green-50 dark:bg-slate-700 border-green-200 dark:border-green-700'
            case 'failed':
                return 'bg-red-50 dark:bg-slate-700 border-red-200 dark:border-red-700'
            case 'running':
                return 'bg-blue-50 dark:bg-slate-700 border-blue-200 dark:border-blue-700'
            case 'queued':
                return 'bg-yellow-50 dark:bg-slate-700 border-yellow-200 dark:border-yellow-700'
            default:
                return 'bg-gray-50 dark:bg-slate-700 border-gray-200 dark:border-gray-700'
        }
    }

    const getStatusTextColor = (status: BuildStatus) => {
        switch (status) {
            case 'completed':
                return 'text-green-900 dark:text-green-200'
            case 'failed':
                return 'text-red-900 dark:text-red-200'
            case 'running':
                return 'text-blue-900 dark:text-blue-200'
            case 'queued':
                return 'text-yellow-900 dark:text-yellow-200'
            default:
                return 'text-gray-900 dark:text-white'
        }
    }

    return (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow p-6 border border-slate-200 dark:border-slate-700">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Last Build</h3>

            <Link
                to={`/builds/${lastBuild.id}`}
                className={`block p-4 rounded-lg border transition-all hover:shadow-md ${getStatusBgColor(
                    lastBuild.status as BuildStatus
                )}`}
            >
                <div className="flex items-start gap-4">
                    {/* Status Icon */}
                    <div className="flex-shrink-0">{getStatusIcon(lastBuild.status as BuildStatus)}</div>

                    {/* Build Info */}
                    <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-2">
                            <h4 className={`font-semibold ${getStatusTextColor(lastBuild.status as BuildStatus)}`}>
                                Build #{lastBuild.buildNumber || lastBuild.id.slice(-8)}
                            </h4>
                        </div>

                        <div className="space-y-1 text-sm text-gray-600 dark:text-slate-400 mb-3">
                            {lastBuild.gitBranch && (
                                <div>
                                    <span className="font-medium">Branch:</span>{' '}
                                    <code className="bg-gray-900 text-green-400 px-2 py-0.5 rounded text-xs">
                                        {lastBuild.gitBranch}
                                    </code>
                                </div>
                            )}

                            {lastBuild.gitCommit && (
                                <div>
                                    <span className="font-medium">Commit:</span>{' '}
                                    <code className="bg-gray-900 text-cyan-400 px-2 py-0.5 rounded text-xs">
                                        {lastBuild.gitCommit.slice(0, 7)}
                                    </code>
                                </div>
                            )}

                            <div className="text-xs text-gray-500 dark:text-slate-500">
                                {formatDate(lastBuild.createdAt)}
                                {getDuration(lastBuild.startedAt, lastBuild.completedAt) && (
                                    <span> • {getDuration(lastBuild.startedAt, lastBuild.completedAt)}</span>
                                )}
                            </div>
                        </div>

                        {/* Failure Reason */}
                        {lastBuild.status === 'failed' && lastBuild.failureReason && (
                            <div className="p-2 bg-red-100 dark:bg-red-900/20 rounded text-xs text-red-800 dark:text-red-200 mb-3">
                                {lastBuild.failureReason}
                            </div>
                        )}

                        {/* View Build Button */}
                        <div className="flex items-center gap-1 text-blue-600 dark:text-blue-400 font-medium text-sm">
                            <LinkIcon className="w-4 h-4" />
                            View Details
                        </div>
                    </div>
                </div>
            </Link>
        </div>
    )
}

export default LastBuildWidget
