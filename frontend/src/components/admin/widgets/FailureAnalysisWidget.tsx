import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { buildService } from '@/services/buildService'
import { AlertCircle, AlertTriangle, Loader2 } from 'lucide-react'
import React, { useEffect, useState } from 'react'
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'

interface SllowestBuild {
    id: string
    project_id: string
    project_name: string
    duration_seconds: number
    status: string
    created_at: string
}

interface FailureReason {
    reason: string
    count: number
    percentage: number
}

interface ProjectFailureRate {
    project_id: string
    project_name: string
    total_builds: number
    failed_builds: number
    failure_rate: number
}

interface FailureAnalysis {
    slowest_builds: SllowestBuild[]
    failure_reasons: FailureReason[]
    failure_rate_by_project: ProjectFailureRate[]
}

const FailureAnalysisWidget: React.FC = () => {
    const [failures, setFailures] = useState<FailureAnalysis | null>(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        loadFailureData()
        // Auto-refresh every 120 seconds for failure data
        const interval = setInterval(loadFailureData, 120000)
        return () => clearInterval(interval)
    }, [])

    const loadFailureData = async () => {
        try {
            setLoading(true)
            setError(null)
            const data = await buildService.getFailures()
            setFailures(data)
        } catch (err) {
            console.error('Failed to load failure analysis:', err)
            setError('Failed to load failure analysis')
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

    const formatDate = (dateStr: string): string => {
        const date = new Date(dateStr)
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    }

    if (loading && !failures) {
        return (
            <div className="flex items-center justify-center p-8">
                <Loader2 className="w-6 h-6 animate-spin text-blue-600 mr-3" />
                <span className="text-gray-600 dark:text-gray-400">Loading failure analysis...</span>
            </div>
        )
    }

    if (error && !failures) {
        return (
            <Card className="border-red-200 dark:border-red-700/50">
                <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-red-600">
                        <AlertCircle className="w-5 h-5" />
                        Error Loading Failure Data
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-sm text-gray-600 dark:text-gray-400">{error}</p>
                    <button
                        onClick={loadFailureData}
                        className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition"
                    >
                        Retry
                    </button>
                </CardContent>
            </Card>
        )
    }

    if (!failures) {
        return (
            <Card className="border-yellow-200 dark:border-yellow-700/50">
                <CardHeader>
                    <CardTitle className="flex items-center gap-2 text-yellow-600">
                        <AlertCircle className="w-5 h-5" />
                        No Failure Data
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                        No failure data available yet
                    </p>
                </CardContent>
            </Card>
        )
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                    <AlertTriangle className="w-5 h-5 text-red-600" />
                    Failure Analysis
                </h3>
                <button
                    onClick={loadFailureData}
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

            {/* Slowest Builds Table */}
            {failures.slowest_builds && failures.slowest_builds.length > 0 && (
                <Card className="border-gray-200 dark:border-gray-700">
                    <CardHeader>
                        <CardTitle className="text-sm">Slowest Builds (Last 30 Days)</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="overflow-x-auto">
                            <Table>
                                <TableHeader>
                                    <TableRow>
                                        <TableHead>Project</TableHead>
                                        <TableHead>Build ID</TableHead>
                                        <TableHead>Duration</TableHead>
                                        <TableHead>Status</TableHead>
                                        <TableHead>Created</TableHead>
                                    </TableRow>
                                </TableHeader>
                                <TableBody>
                                    {failures.slowest_builds.slice(0, 10).map((build) => (
                                        <TableRow key={build.id}>
                                            <TableCell className="font-medium">
                                                {build.project_name || 'Unknown'}
                                            </TableCell>
                                            <TableCell>
                                                <code className="text-xs bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded">
                                                    {build.id.slice(0, 8)}
                                                </code>
                                            </TableCell>
                                            <TableCell className="font-medium">
                                                {formatDuration(build.duration_seconds)}
                                            </TableCell>
                                            <TableCell>
                                                <Badge
                                                    variant={
                                                        build.status === 'completed' || build.status === 'success' ? 'default' :
                                                            build.status === 'failed' ? 'destructive' :
                                                                'secondary'
                                                    }
                                                >
                                                    {build.status}
                                                </Badge>
                                            </TableCell>
                                            <TableCell className="text-sm text-gray-600 dark:text-gray-400">
                                                {formatDate(build.created_at)}
                                            </TableCell>
                                        </TableRow>
                                    ))}
                                </TableBody>
                            </Table>
                        </div>
                    </CardContent>
                </Card>
            )}

            {/* Failure Reasons Chart */}
            {failures.failure_reasons && failures.failure_reasons.length > 0 && (
                <Card className="border-gray-200 dark:border-gray-700">
                    <CardHeader>
                        <CardTitle className="text-sm">Top Failure Reasons (Last 30 Days)</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <ResponsiveContainer width="100%" height={300}>
                            <BarChart data={failures.failure_reasons.slice(0, 10)}>
                                <CartesianGrid strokeDasharray="3 3" stroke="rgba(0,0,0,0.1)" />
                                <XAxis
                                    dataKey="reason"
                                    stroke="rgba(0,0,0,0.5)"
                                    style={{ fontSize: '12px' }}
                                    angle={-45}
                                    textAnchor="end"
                                    height={100}
                                />
                                <YAxis
                                    label={{ value: 'Count', angle: -90, position: 'insideLeft' }}
                                    stroke="rgba(0,0,0,0.5)"
                                    style={{ fontSize: '12px' }}
                                />
                                <Tooltip
                                    formatter={(value) => [`${value}`, 'Count']}
                                    contentStyle={{
                                        backgroundColor: 'rgba(255, 255, 255, 0.95)',
                                        border: '1px solid #ccc',
                                        borderRadius: '4px'
                                    }}
                                />
                                <Bar
                                    dataKey="count"
                                    fill="#ef4444"
                                    name="Failure Count"
                                    radius={[4, 4, 0, 0]}
                                />
                            </BarChart>
                        </ResponsiveContainer>
                    </CardContent>
                </Card>
            )}

            {/* Failure Rate by Project Chart */}
            {failures.failure_rate_by_project && failures.failure_rate_by_project.length > 0 && (
                <Card className="border-gray-200 dark:border-gray-700">
                    <CardHeader>
                        <CardTitle className="text-sm">Failure Rate by Project</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <ResponsiveContainer width="100%" height={300}>
                            <BarChart data={failures.failure_rate_by_project.slice(0, 10)}>
                                <CartesianGrid strokeDasharray="3 3" stroke="rgba(0,0,0,0.1)" />
                                <XAxis
                                    dataKey="project_name"
                                    stroke="rgba(0,0,0,0.5)"
                                    style={{ fontSize: '12px' }}
                                    angle={-45}
                                    textAnchor="end"
                                    height={100}
                                />
                                <YAxis
                                    domain={[0, 100]}
                                    label={{ value: 'Failure Rate (%)', angle: -90, position: 'insideLeft' }}
                                    stroke="rgba(0,0,0,0.5)"
                                    style={{ fontSize: '12px' }}
                                />
                                <Tooltip
                                    formatter={(value) => [`${Number(value).toFixed(2)}%`, 'Failure Rate']}
                                    contentStyle={{
                                        backgroundColor: 'rgba(255, 255, 255, 0.95)',
                                        border: '1px solid #ccc',
                                        borderRadius: '4px'
                                    }}
                                />
                                <Bar
                                    dataKey="failure_rate"
                                    fill="#f59e0b"
                                    name="Failure Rate (%)"
                                    radius={[4, 4, 0, 0]}
                                />
                            </BarChart>
                        </ResponsiveContainer>
                    </CardContent>
                </Card>
            )}

            {(!failures.slowest_builds || failures.slowest_builds.length === 0) &&
                (!failures.failure_reasons || failures.failure_reasons.length === 0) &&
                (!failures.failure_rate_by_project || failures.failure_rate_by_project.length === 0) && (
                    <Card className="border-green-200 dark:border-green-700/50">
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2 text-green-600">
                                ✓ All Systems Healthy
                            </CardTitle>
                        </CardHeader>
                        <CardContent>
                            <p className="text-sm text-gray-600 dark:text-gray-400">
                                No failures to report. All recent builds have been successful.
                            </p>
                        </CardContent>
                    </Card>
                )}
        </div>
    )
}

export default FailureAnalysisWidget
