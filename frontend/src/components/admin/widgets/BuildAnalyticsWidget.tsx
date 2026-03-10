import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { buildService } from '@/services/buildService'
import { Activity, AlertCircle, BarChart3, CheckCircle, Clock, Loader2 } from 'lucide-react'
import React, { useEffect, useState } from 'react'

interface BuildAnalytics {
    total_builds: number
    running_builds: number
    completed_builds: number
    failed_builds: number
    success_rate: number
    average_duration_seconds: number
    queue_depth: number
    last_updated: string
}

interface MetricCardProps {
    icon: React.ReactNode
    label: string
    value: string | number
    subtext?: string
    color: 'blue' | 'green' | 'red' | 'orange'
}

const MetricCard: React.FC<MetricCardProps> = ({ icon, label, value, subtext, color }) => {
    const colorClasses = {
        blue: 'from-blue-500/10 to-blue-500/5 border-blue-200 dark:border-blue-700/50',
        green: 'from-green-500/10 to-green-500/5 border-green-200 dark:border-green-700/50',
        red: 'from-red-500/10 to-red-500/5 border-red-200 dark:border-red-700/50',
        orange: 'from-orange-500/10 to-orange-500/5 border-orange-200 dark:border-orange-700/50',
    }

    const iconColorClasses = {
        blue: 'text-blue-600 dark:text-blue-400',
        green: 'text-green-600 dark:text-green-400',
        red: 'text-red-600 dark:text-red-400',
        orange: 'text-orange-600 dark:text-orange-400',
    }

    return (
        <Card className={`bg-gradient-to-br ${colorClasses[color]} border`}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    {label}
                </CardTitle>
                <div className={`${iconColorClasses[color]}`}>{icon}</div>
            </CardHeader>
            <CardContent>
                <div className="text-2xl font-bold text-gray-900 dark:text-white">
                    {value}
                </div>
                {subtext && (
                    <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                        {subtext}
                    </p>
                )}
            </CardContent>
        </Card>
    )
}

const BuildAnalyticsWidget: React.FC = () => {
    const [metrics, setMetrics] = useState<BuildAnalytics | null>(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        loadMetrics()
        // Auto-refresh every 30 seconds
        const interval = setInterval(loadMetrics, 30000)
        return () => clearInterval(interval)
    }, [])

    const loadMetrics = async () => {
        try {
            setLoading(true)
            setError(null)
            const data = await buildService.getAnalytics()
            setMetrics(data)
        } catch (err) {
            console.error('Failed to load build analytics:', err)
            setError('Failed to load build analytics')
        } finally {
            setLoading(false)
        }
    }

    const formatDuration = (seconds: number): string => {
        if (seconds < 60) return `${seconds}s`
        const minutes = Math.floor(seconds / 60)
        const secs = seconds % 60
        return `${minutes}m ${secs}s`
    }

    if (loading && !metrics) {
        return (
            <div className="flex items-center justify-center p-8">
                <Loader2 className="w-6 h-6 animate-spin text-blue-600 mr-3" />
                <span className="text-gray-600 dark:text-gray-400">Loading analytics...</span>
            </div>
        )
    }

    if (error && !metrics) {
        return (
            <Card className="border-red-200 dark:border-red-700/50">
                <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-red-600">
                        <AlertCircle className="w-5 h-5" />
                        Error Loading Analytics
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-sm text-gray-600 dark:text-gray-400">{error}</p>
                    <button
                        onClick={loadMetrics}
                        className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition"
                    >
                        Retry
                    </button>
                </CardContent>
            </Card>
        )
    }

    if (!metrics) {
        return (
            <Card className="border-yellow-200 dark:border-yellow-700/50">
                <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-yellow-600">
                        <AlertCircle className="w-5 h-5" />
                        No Data Available
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                        Build analytics data is not available yet
                    </p>
                </CardContent>
            </Card>
        )
    }

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                    <BarChart3 className="w-5 h-5 text-blue-600" />
                    Build Analytics
                </h3>
                <button
                    onClick={loadMetrics}
                    disabled={loading}
                    className="text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white disabled:opacity-50 transition"
                >
                    {loading ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                        'Refresh'
                    )}
                </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                <MetricCard
                    icon={<Activity className="w-5 h-5" />}
                    label="Total Builds"
                    value={metrics.total_builds}
                    subtext={`Updated ${new Date(metrics.last_updated).toLocaleTimeString()}`}
                    color="blue"
                />

                <MetricCard
                    icon={<CheckCircle className="w-5 h-5" />}
                    label="Success Rate"
                    value={`${metrics.success_rate}%`}
                    subtext={`${metrics.completed_builds} succeeded`}
                    color={metrics.success_rate >= 90 ? 'green' : metrics.success_rate >= 80 ? 'orange' : 'red'}
                />

                <MetricCard
                    icon={<Clock className="w-5 h-5" />}
                    label="Avg Duration"
                    value={formatDuration(metrics.average_duration_seconds)}
                    subtext="Average build time"
                    color="orange"
                />

                <MetricCard
                    icon={<Activity className="w-5 h-5" />}
                    label="Queue Depth"
                    value={metrics.queue_depth}
                    subtext={`${metrics.running_builds} running`}
                    color={metrics.queue_depth <= 5 ? 'green' : metrics.queue_depth <= 10 ? 'orange' : 'red'}
                />
            </div>

            <Card className="border-gray-200 dark:border-gray-700">
                <CardHeader>
                    <CardTitle className="text-sm">Build Summary</CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="grid grid-cols-3 gap-4 text-sm">
                        <div>
                            <p className="text-gray-600 dark:text-gray-400">Completed</p>
                            <p className="text-lg font-semibold text-green-600 dark:text-green-400">
                                {metrics.completed_builds}
                            </p>
                        </div>
                        <div>
                            <p className="text-gray-600 dark:text-gray-400">Failed</p>
                            <p className="text-lg font-semibold text-red-600 dark:text-red-400">
                                {metrics.failed_builds}
                            </p>
                        </div>
                        <div>
                            <p className="text-gray-600 dark:text-gray-400">Running</p>
                            <p className="text-lg font-semibold text-blue-600 dark:text-blue-400">
                                {metrics.running_builds}
                            </p>
                        </div>
                    </div>
                </CardContent>
            </Card>
        </div>
    )
}

export default BuildAnalyticsWidget
