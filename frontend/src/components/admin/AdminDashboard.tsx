import {
    Activity,
    AlertTriangle,
    CheckCircle,
    Server,
    Shield,
    Users,
    Wrench
} from 'lucide-react'
import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { buildService } from '../../services/buildService'
import { ToolAvailabilityConfig } from '../../types'
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card'
import { Progress } from '../ui/progress'
import { DispatcherMetricsPanel, InfrastructurePanel } from './panels'
import {
    BuildAnalyticsWidget,
    BuildPerformanceWidget,
    FailureAnalysisWidget
} from './widgets'

interface SystemStats {
    totalBuilds: number
    runningBuilds: number
    completedBuilds: number
    failedBuilds: number
    totalTenants: number
    activeTenants: number
    systemHealth: 'healthy' | 'warning' | 'critical'
}

const AdminDashboard: React.FC = () => {
    const [stats, setStats] = useState<SystemStats | null>(null)
    const [toolAvailability, setToolAvailability] = useState<ToolAvailabilityConfig | null>(null)
    const [isLoading, setIsLoading] = useState(true)

    useEffect(() => {
        loadDashboardData()
    }, [])

    const loadDashboardData = async () => {
        try {
            setIsLoading(true)

            // Load tool availability
            const tools = await buildService.getToolAvailability()
            setToolAvailability(tools)

            // Load system stats from API
            // For now, keep basic stats - they'll be superseded by widgets below
            const mockStats: SystemStats = {
                totalBuilds: 0,
                runningBuilds: 0,
                completedBuilds: 0,
                failedBuilds: 0,
                totalTenants: 45,
                activeTenants: 42,
                systemHealth: 'healthy'
            }
            setStats(mockStats)

        } catch (error) {
            console.error('Failed to load dashboard data:', error)
        } finally {
            setIsLoading(false)
        }
    }

    const getHealthColor = (health: string) => {
        switch (health) {
            case 'healthy':
                return 'text-green-600'
            case 'warning':
                return 'text-yellow-600'
            case 'critical':
                return 'text-red-600'
            default:
                return 'text-gray-600'
        }
    }

    const getHealthIcon = (health: string) => {
        switch (health) {
            case 'healthy':
                return <CheckCircle className="w-5 h-5" />
            case 'warning':
                return <AlertTriangle className="w-5 h-5" />
            case 'critical':
                return <AlertTriangle className="w-5 h-5" />
            default:
                return <Server className="w-5 h-5" />
        }
    }

    if (isLoading) {
        return (
            <div className="flex items-center justify-center p-8">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
                <span className="ml-2">Loading dashboard...</span>
            </div>
        )
    }

    const enabledTools = toolAvailability ? Object.values(toolAvailability).reduce((acc, category) => {
        return acc + Object.values(category).filter(Boolean).length
    }, 0) : 0

    const totalTools = toolAvailability ? Object.values(toolAvailability).reduce((acc, category) => {
        return acc + Object.keys(category).length
    }, 0) : 0

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
                    Admin Dashboard
                </h1>
                <p className="mt-2 text-gray-600 dark:text-gray-400">
                    System overview and administrative controls
                </p>
            </div>

            {/* Build Analytics Widget - Real-time metrics from API */}
            <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
                <BuildAnalyticsWidget />
            </div>

            {/* Infrastructure Panel - Node management and resource monitoring */}
            <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700">
                <InfrastructurePanel />
            </div>

            {/* Dispatcher Metrics */}
            <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
                <DispatcherMetricsPanel />
            </div>

            {/* System Health */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">System Health</CardTitle>
                        {stats && getHealthIcon(stats.systemHealth)}
                    </CardHeader>
                    <CardContent>
                        <div className={`text-2xl font-bold ${getHealthColor(stats?.systemHealth || 'healthy')}`}>
                            {stats?.systemHealth === 'healthy' ? 'Healthy' :
                                stats?.systemHealth === 'warning' ? 'Warning' : 'Critical'}
                        </div>
                        <p className="text-xs text-muted-foreground">
                            All systems operational
                        </p>
                    </CardContent>
                </Card>

                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Active Tenants</CardTitle>
                        <Users className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">{stats?.activeTenants || 0}</div>
                        <p className="text-xs text-muted-foreground">
                            of {stats?.totalTenants || 0} total
                        </p>
                    </CardContent>
                </Card>

                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Tool Availability</CardTitle>
                        <Wrench className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">{enabledTools}/{totalTools}</div>
                        <p className="text-xs text-muted-foreground">
                            Tools enabled
                        </p>
                    </CardContent>
                </Card>
            </div>

            {/* Build Performance Trends Widget */}
            <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
                <BuildPerformanceWidget />
            </div>

            {/* Failure Analysis Widget */}
            <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
                <FailureAnalysisWidget />
            </div>

            {/* Tool Availability Overview */}
            {toolAvailability && (
                <Card>
                    <CardHeader>
                        <CardTitle className="flex items-center space-x-2">
                            <Shield className="w-5 h-5" />
                            <span>Tool Availability Overview</span>
                        </CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="space-y-3">
                            {Object.entries(toolAvailability).map(([category, tools]) => {
                                const enabled = Object.values(tools).filter(Boolean).length
                                const total = Object.keys(tools).length
                                const percentage = (enabled / total) * 100

                                return (
                                    <div key={category} className="space-y-1">
                                        <div className="flex items-center justify-between text-sm">
                                            <span className="font-medium capitalize">
                                                {category.replace('_', ' ')}
                                            </span>
                                            <span className="text-muted-foreground">
                                                {enabled}/{total}
                                            </span>
                                        </div>
                                        <Progress value={percentage} className="h-2" />
                                    </div>
                                )
                            })}
                        </div>
                    </CardContent>
                </Card>
            )}

            {/* Quick Actions */}
            <Card>
                <CardHeader>
                    <CardTitle>Quick Actions</CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                        <Link to="/admin/tools">
                            <div className="p-3 rounded-lg transition-all border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-700/50 text-slate-700 dark:text-slate-300 hover:text-slate-900 dark:hover:text-slate-100 cursor-pointer h-20 flex flex-col items-center justify-center space-y-2">
                                <Wrench className="w-6 h-6" />
                                <span className="text-sm font-medium">Manage Tools</span>
                            </div>
                        </Link>

                        <Link to="/admin/builds">
                            <div className="p-3 rounded-lg transition-all border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-700/50 text-slate-700 dark:text-slate-300 hover:text-slate-900 dark:hover:text-slate-100 cursor-pointer h-20 flex flex-col items-center justify-center space-y-2">
                                <Activity className="w-6 h-6" />
                                <span className="text-sm font-medium">Build Management</span>
                            </div>
                        </Link>

                        <Link to="/admin/users">
                            <div className="p-3 rounded-lg transition-all border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-700/50 text-slate-700 dark:text-slate-300 hover:text-slate-900 dark:hover:text-slate-100 cursor-pointer h-20 flex flex-col items-center justify-center space-y-2">
                                <Users className="w-6 h-6" />
                                <span className="text-sm font-medium">User Management</span>
                            </div>
                        </Link>
                    </div>
                </CardContent>
            </Card>
        </div>
    )
}

export default AdminDashboard
