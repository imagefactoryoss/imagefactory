import { buildService } from '@/services/buildService'
import { useTenantStore } from '@/store/tenant'
import { AlertTriangle, TrendingUp } from 'lucide-react'
import React, { useEffect, useState } from 'react'

interface BuildStatsWidgetProps {
    projectId: string
}

export const BuildStatsWidget: React.FC<BuildStatsWidgetProps> = ({ projectId }) => {
    const [stats, setStats] = useState({
        total: 0,
        successful: 0,
        failed: 0,
        inProgress: 0,
        successRate: 0,
    })
    const [loading, setLoading] = useState(true)
    const { selectedTenantId } = useTenantStore()
    const normalizeBuildStatus = (status: string) => {
        if (status === 'success') return 'completed'
        if (status === 'in_progress') return 'running'
        return status
    }

    useEffect(() => {
        loadStats()
    }, [projectId, selectedTenantId])

    const loadStats = async () => {
        try {
            setLoading(true)
            // Load builds with higher limit to calculate stats
            const response = await buildService.getBuilds({
                projectId,
                limit: 100,
                page: 1,
                tenantId: selectedTenantId ?? undefined,
            })

            const builds = response.data
            const total = builds.length
            const successful = builds.filter((b) => normalizeBuildStatus(String(b.status)) === 'completed').length
            const failed = builds.filter((b) => normalizeBuildStatus(String(b.status)) === 'failed').length
            const inProgress = builds.filter((b) => {
                const status = normalizeBuildStatus(String(b.status))
                return status === 'pending' || status === 'queued' || status === 'running'
            }).length
            const terminalTotal = successful + failed

            setStats({
                total,
                successful,
                failed,
                inProgress,
                successRate: terminalTotal > 0 ? Math.round((successful / terminalTotal) * 100) : 0,
            })
        } catch (error) {
            console.error('Failed to load build stats:', error)
        } finally {
            setLoading(false)
        }
    }

    if (loading) {
        return (
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow p-6 border border-slate-200 dark:border-slate-700">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Build Statistics</h3>
                <div className="grid grid-cols-2 gap-4">
                    {[...Array(4)].map((_, i) => (
                        <div key={i} className="h-20 bg-gray-100 dark:bg-slate-700 rounded animate-pulse" />
                    ))}
                </div>
            </div>
        )
    }

    return (
        <div className="bg-white dark:bg-slate-800 rounded-lg shadow p-6 border border-slate-200 dark:border-slate-700">
            <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Build Statistics</h3>
                <TrendingUp className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            </div>

            <div className="grid grid-cols-2 gap-4">
                {/* Total Builds */}
                <div className="bg-gray-50 dark:bg-slate-700 rounded-lg p-4">
                    <p className="text-xs font-medium text-gray-600 dark:text-slate-400 uppercase tracking-wider">Total Builds</p>
                    <p className="text-2xl font-bold text-gray-900 dark:text-white mt-2">{stats.total}</p>
                </div>

                {/* Success Rate */}
                <div className="bg-green-50 dark:bg-slate-700 rounded-lg p-4">
                    <p className="text-xs font-medium text-green-600 dark:text-green-400 uppercase tracking-wider">Success Rate</p>
                    <p className="text-2xl font-bold text-green-900 dark:text-green-200 mt-2">{stats.successRate}%</p>
                    {stats.total > 0 && (
                        <div className="mt-2 w-full bg-gray-300 dark:bg-slate-600 rounded-full h-1">
                            <div
                                className="bg-green-600 dark:bg-green-500 h-1 rounded-full transition-all duration-300"
                                style={{ width: `${stats.successRate}%` }}
                            />
                        </div>
                    )}
                </div>

                {/* Successful Builds */}
                <div className="bg-blue-50 dark:bg-slate-700 rounded-lg p-4">
                    <p className="text-xs font-medium text-blue-600 dark:text-blue-400 uppercase tracking-wider">Successful</p>
                    <p className="text-2xl font-bold text-blue-900 dark:text-blue-200 mt-2">{stats.successful}</p>
                </div>

                {/* Failed Builds */}
                <div className="bg-red-50 dark:bg-slate-700 rounded-lg p-4">
                    <div className="flex items-center gap-2">
                        <p className="text-xs font-medium text-red-600 dark:text-red-400 uppercase tracking-wider">Failed</p>
                        {stats.failed > 0 && <AlertTriangle className="w-3 h-3 text-red-600 dark:text-red-400" />}
                    </div>
                    <p className="text-2xl font-bold text-red-900 dark:text-red-200 mt-2">{stats.failed}</p>
                </div>
            </div>

            {/* In Progress Info */}
            {stats.inProgress > 0 && (
                <div className="mt-4 p-3 bg-blue-50 dark:bg-slate-700 border border-blue-200 dark:border-slate-600 rounded-lg">
                    <p className="text-sm text-blue-800 dark:text-blue-200">
                        <span className="font-semibold">{stats.inProgress}</span> build
                        {stats.inProgress !== 1 ? 's are' : ' is'} currently in progress
                    </p>
                </div>
            )}
        </div>
    )
}

export default BuildStatsWidget
