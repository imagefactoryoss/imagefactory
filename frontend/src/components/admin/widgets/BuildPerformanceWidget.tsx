import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { buildService } from '@/services/buildService'
import { AlertCircle, Loader2, TrendingUp } from 'lucide-react'
import React, { useEffect, useState } from 'react'
import { Bar, BarChart, CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'

interface TrendPoint {
    date: string
    average_seconds?: number
    success_rate?: number
    average_queue_time_seconds?: number
    count?: number
}

interface PerformanceTrends {
    duration_trend: TrendPoint[] | null
    success_trend: TrendPoint[] | null
    queue_trend: TrendPoint[] | null
}

const BuildPerformanceWidget: React.FC = () => {
    const [trends, setTrends] = useState<PerformanceTrends | null>(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        loadPerformance()
        // Auto-refresh every 60 seconds for trends
        const interval = setInterval(loadPerformance, 60000)
        return () => clearInterval(interval)
    }, [])

    const loadPerformance = async () => {
        try {
            setLoading(true)
            setError(null)
            const data = await buildService.getPerformance()
            setTrends(data)
        } catch (err) {
            console.error('Failed to load performance trends:', err)
            setError('Failed to load performance trends')
        } finally {
            setLoading(false)
        }
    }

    if (loading && !trends) {
        return (
            <div className="flex items-center justify-center p-8">
                <Loader2 className="w-6 h-6 animate-spin text-blue-600 mr-3" />
                <span className="text-gray-600 dark:text-gray-400">Loading performance data...</span>
            </div>
        )
    }

    if (error && !trends) {
        return (
            <Card className="border-red-200 dark:border-red-700/50">
                <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-red-600">
                        <AlertCircle className="w-5 h-5" />
                        Error Loading Performance Data
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-sm text-gray-600 dark:text-gray-400">{error}</p>
                    <button
                        onClick={loadPerformance}
                        className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition"
                    >
                        Retry
                    </button>
                </CardContent>
            </Card>
        )
    }

    if (!trends || !trends.duration_trend || trends.duration_trend.length === 0) {
        return (
            <Card className="border-yellow-200 dark:border-yellow-700/50">
                <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-yellow-600">
                        <AlertCircle className="w-5 h-5" />
                        Insufficient Data
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                        Not enough historical data to display trends yet
                    </p>
                </CardContent>
            </Card>
        )
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                    <TrendingUp className="w-5 h-5 text-blue-600" />
                    Performance Trends (7 Days)
                </h3>
                <button
                    onClick={loadPerformance}
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

            {/* Duration Trend Chart */}
            <Card className="border-gray-200 dark:border-gray-700">
                <CardHeader>
                    <CardTitle className="text-sm">Average Build Duration</CardTitle>
                </CardHeader>
                <CardContent>
                    <ResponsiveContainer width="100%" height={300}>
                        <LineChart data={trends.duration_trend || []}>
                            <CartesianGrid strokeDasharray="3 3" stroke="rgba(0,0,0,0.1)" />
                            <XAxis
                                dataKey="date"
                                stroke="rgba(0,0,0,0.5)"
                                style={{ fontSize: '12px' }}
                            />
                            <YAxis
                                label={{ value: 'Seconds', angle: -90, position: 'insideLeft' }}
                                stroke="rgba(0,0,0,0.5)"
                                style={{ fontSize: '12px' }}
                            />
                            <Tooltip
                                formatter={(value) => [`${value}s`, 'Duration']}
                                contentStyle={{
                                    backgroundColor: 'rgba(255, 255, 255, 0.95)',
                                    border: '1px solid #ccc',
                                    borderRadius: '4px'
                                }}
                            />
                            <Legend />
                            <Line
                                type="monotone"
                                dataKey="average_seconds"
                                stroke="#3b82f6"
                                name="Avg Duration (seconds)"
                                strokeWidth={2}
                                dot={{ fill: '#3b82f6', r: 4 }}
                                activeDot={{ r: 6 }}
                            />
                        </LineChart>
                    </ResponsiveContainer>
                </CardContent>
            </Card>

            {/* Success Rate Trend Chart */}
            <Card className="border-gray-200 dark:border-gray-700">
                <CardHeader>
                    <CardTitle className="text-sm">Build Success Rate</CardTitle>
                </CardHeader>
                <CardContent>
                    <ResponsiveContainer width="100%" height={300}>
                        <LineChart data={trends.success_trend || []}>
                            <CartesianGrid strokeDasharray="3 3" stroke="rgba(0,0,0,0.1)" />
                            <XAxis
                                dataKey="date"
                                stroke="rgba(0,0,0,0.5)"
                                style={{ fontSize: '12px' }}
                            />
                            <YAxis
                                domain={[0, 100]}
                                label={{ value: 'Percentage (%)', angle: -90, position: 'insideLeft' }}
                                stroke="rgba(0,0,0,0.5)"
                                style={{ fontSize: '12px' }}
                            />
                            <Tooltip
                                formatter={(value) => [`${value}%`, 'Success Rate']}
                                contentStyle={{
                                    backgroundColor: 'rgba(255, 255, 255, 0.95)',
                                    border: '1px solid #ccc',
                                    borderRadius: '4px'
                                }}
                            />
                            <Legend />
                            <Line
                                type="monotone"
                                dataKey="success_rate"
                                stroke="#10b981"
                                name="Success Rate (%)"
                                strokeWidth={2}
                                dot={{ fill: '#10b981', r: 4 }}
                                activeDot={{ r: 6 }}
                            />
                        </LineChart>
                    </ResponsiveContainer>
                </CardContent>
            </Card>

            {/* Queue Depth Trend Chart */}
            <Card className="border-gray-200 dark:border-gray-700">
                <CardHeader>
                    <CardTitle className="text-sm">Average Queue Depth</CardTitle>
                </CardHeader>
                <CardContent>
                    <ResponsiveContainer width="100%" height={300}>
                        <BarChart data={trends.queue_trend || []}>
                            <CartesianGrid strokeDasharray="3 3" stroke="rgba(0,0,0,0.1)" />
                            <XAxis
                                dataKey="date"
                                stroke="rgba(0,0,0,0.5)"
                                style={{ fontSize: '12px' }}
                            />
                            <YAxis
                                label={{ value: 'Queue Depth', angle: -90, position: 'insideLeft' }}
                                stroke="rgba(0,0,0,0.5)"
                                style={{ fontSize: '12px' }}
                            />
                            <Tooltip
                                formatter={(value) => [Math.round(value as number * 10) / 10, 'Queue Depth']}
                                contentStyle={{
                                    backgroundColor: 'rgba(255, 255, 255, 0.95)',
                                    border: '1px solid #ccc',
                                    borderRadius: '4px'
                                }}
                            />
                            <Legend />
                            <Bar
                                dataKey="average_queue_time_seconds"
                                fill="#f59e0b"
                                name="Avg Queue Depth"
                                radius={[4, 4, 0, 0]}
                            />
                        </BarChart>
                    </ResponsiveContainer>
                </CardContent>
            </Card>
        </div>
    )
}

export default BuildPerformanceWidget
