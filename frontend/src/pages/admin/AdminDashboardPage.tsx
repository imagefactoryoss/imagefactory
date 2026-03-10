import useBuildStatusWebSocket, { BuildStatusEvent } from '@/hooks/useBuildStatusWebSocket'
import { adminService } from '@/services/adminService'
import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'
import { DenialMetricsRow, ExecutionPipelineComponentHealth, ReleaseGovernancePolicyConfig, ReleaseMetricsSnapshot } from '@/types'
import React, { useCallback, useEffect, useState } from 'react'
import { Link, Navigate } from 'react-router-dom'

interface SystemStats {
    totalUsers: number
    totalTenants: number
    activeBuilds: number
    systemHealth: {
        status: 'healthy' | 'warning' | 'critical'
        uptime: string
        lastHealthCheck: string
    }
}

interface SORDenialSummary {
    total: number
    byMode: Record<string, number>
    byScope: Record<string, number>
}

interface ReleaseAlertSummary {
    level: 'healthy' | 'warning'
    message: string
    ratioText: string
}

interface DashboardComponentStatus {
    name: string
    status: 'healthy' | 'warning' | 'critical'
    lastCheck: string
    message?: string
    endpoint?: string
    httpStatus?: number
    latencyMs?: number
    configured: boolean
    details?: Record<string, any>
}

interface WorkflowControlPlaneDiagnostics {
    subject_type: string
    blocked_step_count: number
    dispatch_blocked: number
    monitor_blocked: number
    finalize_blocked: number
    oldest_blocked_at?: string
    oldest_blocked_age_seconds?: number
    recovery_action?: string
    recovery_hint?: string
}

interface StatCardProps {
    icon: string
    label: string
    value: number | string
    change: string
    color: 'blue' | 'green' | 'purple'
}

const StatCard: React.FC<StatCardProps> = ({ icon, label, value, change, color }) => {
    const colorMap = {
        blue: 'from-blue-500/10 to-blue-500/5 border-blue-200 dark:border-blue-700/50',
        green: 'from-green-500/10 to-green-500/5 border-green-200 dark:border-green-700/50',
        purple: 'from-purple-500/10 to-purple-500/5 border-purple-200 dark:border-purple-700/50',
    }

    return (
        <div className={`bg-gradient-to-br ${colorMap[color]} rounded-lg border p-6`}>
            <div className="flex items-start justify-between">
                <div>
                    <p className="text-sm font-medium text-slate-600 dark:text-slate-400">{label}</p>
                    <p className="mt-2 text-3xl font-bold text-slate-900 dark:text-white">{value}</p>
                    <p className="mt-2 text-xs text-slate-600 dark:text-slate-400">{change}</p>
                </div>
                <span className="text-3xl">{icon}</span>
            </div>
        </div>
    )
}

interface AdminQuickLinkProps {
    icon: string
    title: string
    description: string
    href: string
    color: 'blue' | 'green' | 'purple' | 'orange'
}

const AdminQuickLink: React.FC<AdminQuickLinkProps> = ({ icon, title, description, href, color }) => {
    const colorMap = {
        blue: 'hover:border-blue-300 dark:hover:border-blue-600',
        green: 'hover:border-green-300 dark:hover:border-green-600',
        purple: 'hover:border-purple-300 dark:hover:border-purple-600',
        orange: 'hover:border-orange-300 dark:hover:border-orange-600',
    }

    return (
        <Link
            to={href}
            className={`block p-3 border border-slate-200 dark:border-slate-700 rounded-lg hover:shadow-md transition ${colorMap[color]}`}
        >
            <div className="flex items-start gap-3">
                <div className="w-8 h-8 rounded-md bg-slate-100 dark:bg-slate-700/70 flex items-center justify-center text-base shrink-0">
                    {icon}
                </div>
                <div className="min-w-0">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white leading-5">{title}</h3>
                    <p className="text-xs text-slate-600 dark:text-slate-400 mt-0.5 line-clamp-2">{description}</p>
                </div>
            </div>
        </Link>
    )
}

const AdminDashboardPage: React.FC = () => {
    const { user, groups } = useAuthStore()
    const { selectedTenantId, userTenants } = useTenantStore()
    const [stats, setStats] = useState<SystemStats>({
        totalUsers: 0,
        totalTenants: 0,
        activeBuilds: 0,
        systemHealth: {
            status: 'healthy',
            uptime: '99.5%',
            lastHealthCheck: new Date().toLocaleTimeString(),
        },
    })
    const [loading, setLoading] = useState(true)
    const [componentStatus, setComponentStatus] = useState<Record<string, DashboardComponentStatus>>({})
    const [checkingComponents, setCheckingComponents] = useState(false)
    const [executionPipelineHealth, setExecutionPipelineHealth] = useState<Record<string, ExecutionPipelineComponentHealth>>({})
    const [workflowDiagnostics, setWorkflowDiagnostics] = useState<WorkflowControlPlaneDiagnostics | null>(null)
    const [checkingExecutionPipeline, setCheckingExecutionPipeline] = useState(false)
    const [recoveryInProgress, setRecoveryInProgress] = useState(false)
    const [orchestratorActionInProgress, setOrchestratorActionInProgress] = useState(false)
    const [pipelineCheckedAt, setPipelineCheckedAt] = useState<string | null>(null)
    const [autoRefreshEnabled, setAutoRefreshEnabled] = useState(false)
    const [pollIntervalSeconds, setPollIntervalSeconds] = useState(15)
    const [compactExecutionCards, setCompactExecutionCards] = useState(true)
    const [compactSystemComponentCards, setCompactSystemComponentCards] = useState(true)
    const [sorDenialSummary, setSorDenialSummary] = useState<SORDenialSummary>({
        total: 0,
        byMode: {},
        byScope: {},
    })
    const [releaseMetrics, setReleaseMetrics] = useState<ReleaseMetricsSnapshot>({
        requested: 0,
        released: 0,
        failed: 0,
        total: 0,
    })
    const [releasePolicy, setReleasePolicy] = useState<ReleaseGovernancePolicyConfig>({
        enabled: true,
        failure_ratio_threshold: 0.25,
        consecutive_failures_threshold: 3,
        minimum_samples: 5,
        window_minutes: 60,
    })
    const statsLoadedRef = React.useRef(false)
    const pipelineRefreshTimerRef = React.useRef<number | null>(null)
    const orchestratorHealth = executionPipelineHealth.workflow_orchestrator
    const notificationSubscriberHealth = executionPipelineHealth.build_notification_event_subscriber
    const monitorSubscriberHealth = executionPipelineHealth.build_monitor_event_subscriber
    const outboxRelayHealth = executionPipelineHealth.messaging_outbox_relay
    const runtimeDependencyWatcherHealth = executionPipelineHealth.runtime_dependency_watcher

    // Check if user is system administrator
    const groupsLoaded = Array.isArray(groups)
    const isAdmin = groups?.some((group: any) => ['system_administrator', 'system_administrator_viewer'].includes(group.role_type))

    const mapComponentStatus = (components: any): Record<string, DashboardComponentStatus> => {
        const mappedComponents: Record<string, DashboardComponentStatus> = {}
        Object.entries(components || {}).forEach(([key, component]: [string, any]) => {
            mappedComponents[key] = {
                name: component.name,
                status: component.status,
                lastCheck: component.last_check,
                message: component.message,
                endpoint: component.endpoint,
                httpStatus: component.http_status,
                latencyMs: component.latency_ms,
                configured: component.configured,
                details: component.details,
            }
        })
        return mappedComponents
    }

    const formatBytes = (bytes?: number) => {
        const value = Number(bytes || 0)
        if (!Number.isFinite(value) || value <= 0) return '0 B'
        const units = ['B', 'KB', 'MB', 'GB', 'TB']
        let size = value
        let unitIndex = 0
        while (size >= 1024 && unitIndex < units.length - 1) {
            size /= 1024
            unitIndex++
        }
        return `${size.toFixed(unitIndex === 0 ? 0 : 2)} ${units[unitIndex]}`
    }

    const summarizeSORDenials = (rows?: DenialMetricsRow[]): SORDenialSummary => {
        const summary: SORDenialSummary = {
            total: 0,
            byMode: {},
            byScope: {},
        }
        ;(rows || []).forEach((row) => {
            if (row.reason !== 'epr_registration_required') {
                return
            }
            summary.total += row.count || 0
            const mode = row.labels?.epr_runtime_mode || 'unknown'
            const scope = row.labels?.epr_policy_scope || 'unknown'
            summary.byMode[mode] = (summary.byMode[mode] || 0) + (row.count || 0)
            summary.byScope[scope] = (summary.byScope[scope] || 0) + (row.count || 0)
        })
        return summary
    }

    const summarizeReleaseAlert = (
        metrics: ReleaseMetricsSnapshot,
        policy: ReleaseGovernancePolicyConfig
    ): ReleaseAlertSummary => {
        const total = Number(metrics.total || 0)
        const failed = Number(metrics.failed || 0)
        const ratio = total > 0 ? failed / total : 0
        const ratioPercent = `${(ratio * 100).toFixed(1)}%`
        const thresholdPercent = `${(policy.failure_ratio_threshold * 100).toFixed(1)}%`
        if (!policy.enabled) {
            return {
                level: 'healthy',
                message: 'Release alert policy is disabled.',
                ratioText: `${ratioPercent} (alerts disabled)`,
            }
        }
        if (total < policy.minimum_samples) {
            return {
                level: 'healthy',
                message: `Collecting samples (${total}/${policy.minimum_samples}) before alerting starts.`,
                ratioText: `${ratioPercent} (threshold ${thresholdPercent})`,
            }
        }
        if (ratio >= policy.failure_ratio_threshold) {
            return {
                level: 'warning',
                message: `Failure ratio threshold breached: ${ratioPercent} >= ${thresholdPercent}.`,
                ratioText: `${ratioPercent} (threshold ${thresholdPercent})`,
            }
        }
        return {
            level: 'healthy',
            message: `Failure ratio is within threshold (${ratioPercent} < ${thresholdPercent}).`,
            ratioText: `${ratioPercent} (threshold ${thresholdPercent})`,
        }
    }

    const refreshComponentStatus = async () => {
        try {
            setCheckingComponents(true)
            const componentHealth = await adminService.getSystemComponentsStatus()
            setComponentStatus(mapComponentStatus(componentHealth.components))
        } catch (error) {
            setComponentStatus({})
        } finally {
            setCheckingComponents(false)
        }
    }

    const refreshExecutionPipelineHealth = useCallback(async () => {
        try {
            setCheckingExecutionPipeline(true)
            const pipelineHealth = await adminService.getExecutionPipelineHealth()
            setExecutionPipelineHealth(pipelineHealth.components || {})
            setWorkflowDiagnostics(pipelineHealth.workflow_control_plane || null)
            setPipelineCheckedAt(pipelineHealth.checked_at || null)
        } catch (error) {
            setExecutionPipelineHealth({})
            setWorkflowDiagnostics(null)
            setPipelineCheckedAt(null)
        } finally {
            setCheckingExecutionPipeline(false)
        }
    }, [])

    const schedulePipelineRefresh = useCallback((delayMs = 800) => {
        if (pipelineRefreshTimerRef.current !== null) {
            return
        }
        pipelineRefreshTimerRef.current = window.setTimeout(() => {
            pipelineRefreshTimerRef.current = null
            refreshExecutionPipelineHealth()
        }, Math.max(250, delayMs))
    }, [refreshExecutionPipelineHealth])

    const executeRecoveryAction = async () => {
        if (!workflowDiagnostics?.recovery_action) {
            return
        }

        if (workflowDiagnostics.recovery_action === 'start_dispatcher') {
            try {
                setRecoveryInProgress(true)
                await adminService.startDispatcher()
                await refreshExecutionPipelineHealth()
            } catch {
                // noop: panel already shows degraded state and message from API
            } finally {
                setRecoveryInProgress(false)
            }
            return
        }

        if (workflowDiagnostics.recovery_action === 'start_orchestrator') {
            try {
                setRecoveryInProgress(true)
                await adminService.startOrchestrator()
                await refreshExecutionPipelineHealth()
            } catch {
                // noop: panel already shows degraded state and message from API
            } finally {
                setRecoveryInProgress(false)
            }
        }
    }

    const toggleOrchestrator = async () => {
        if (!orchestratorHealth?.enabled) {
            return
        }

        try {
            setOrchestratorActionInProgress(true)
            if (orchestratorHealth.running) {
                await adminService.stopOrchestrator()
            } else {
                await adminService.startOrchestrator()
            }
            await refreshExecutionPipelineHealth()
        } catch {
            // noop: pipeline card already reflects backend state/errors
        } finally {
            setOrchestratorActionInProgress(false)
        }
    }

    // Safe to proceed - we have groups and user is admin

    useEffect(() => {
        // Only load stats if user is admin - this prevents API call for non-admins
        if (!groups || groups.length === 0) {
            // Still loading, don't fetch yet
            return
        }

        // Wait until tenant context is ready, otherwise backend rejects request.
        if (!selectedTenantId && (!userTenants || userTenants.length === 0)) {
            return
        }

        if (!isAdmin) {
            setLoading(false)
            return
        }

        // Prevent multiple calls
        if (statsLoadedRef.current) {
            return
        }
        statsLoadedRef.current = true

        // Fetch actual stats from API only for admins
        const loadStats = async () => {
            try {
                setLoading(true)
                const [backendStats, componentHealth, pipelineHealth, releasePolicyConfig] = await Promise.all([
                    adminService.getSystemStats(),
                    adminService.getSystemComponentsStatus(),
                    adminService.getExecutionPipelineHealth(),
                    adminService.getReleaseGovernancePolicy(),
                ])
                const systemConfigs = await adminService.getSystemConfigs()
                const generalConfig = (systemConfigs || []).find((config) => (
                    (config as any).config_type === 'general' && (config as any).config_key === 'general'
                ))
                const configuredPollInterval = Number((generalConfig as any)?.config_value?.admin_dashboard_poll_interval_seconds)
                if (Number.isFinite(configuredPollInterval) && configuredPollInterval >= 5 && configuredPollInterval <= 300) {
                    setPollIntervalSeconds(configuredPollInterval)
                } else {
                    setPollIntervalSeconds(15)
                }

                setStats({
                    totalUsers: backendStats.total_users,
                    totalTenants: backendStats.total_tenants,
                    activeBuilds: backendStats.running_builds,
                    systemHealth: {
                        status: backendStats.system_health as 'healthy' | 'warning' | 'critical',
                        uptime: backendStats.uptime,
                        lastHealthCheck: new Date().toLocaleTimeString(),
                    },
                })
                setSorDenialSummary(summarizeSORDenials(backendStats.denial_metrics))
                setReleaseMetrics(backendStats.release_metrics || { requested: 0, released: 0, failed: 0, total: 0 })
                setReleasePolicy(releasePolicyConfig)
                setComponentStatus(mapComponentStatus(componentHealth.components))
                setExecutionPipelineHealth(pipelineHealth.components || {})
                setWorkflowDiagnostics(pipelineHealth.workflow_control_plane || null)
                setPipelineCheckedAt(pipelineHealth.checked_at || null)
            } catch (error: any) {
                // Handle 403 Forbidden - user doesn't have system admin permission
                if (error?.response?.status === 403 || error?.status === 403) {
                    // This shouldn't happen if routing is correct, but fallback gracefully
                    setStats({
                        totalUsers: 0,
                        totalTenants: 0,
                        activeBuilds: 0,
                        systemHealth: {
                            status: 'warning',
                            uptime: 'N/A',
                            lastHealthCheck: new Date().toLocaleTimeString(),
                        },
                    })
                } else {
                    // Fallback to default values on error
                    setStats({
                        totalUsers: 0,
                        totalTenants: 0,
                        activeBuilds: 0,
                        systemHealth: {
                            status: 'warning',
                            uptime: 'N/A',
                            lastHealthCheck: new Date().toLocaleTimeString(),
                        },
                    })
                }
                setComponentStatus({})
                setExecutionPipelineHealth({})
                setWorkflowDiagnostics(null)
                setPipelineCheckedAt(null)
                setSorDenialSummary({ total: 0, byMode: {}, byScope: {} })
                setReleaseMetrics({ requested: 0, released: 0, failed: 0, total: 0 })
            } finally {
                setLoading(false)
            }
        }

        loadStats()
    }, [isAdmin, groups, selectedTenantId, userTenants])

    useEffect(() => {
        if (!isAdmin || !autoRefreshEnabled) {
            return
        }
        const interval = window.setInterval(() => {
            refreshExecutionPipelineHealth()
        }, Math.max(5000, pollIntervalSeconds * 1000))
        return () => window.clearInterval(interval)
    }, [isAdmin, autoRefreshEnabled, pollIntervalSeconds])

    useEffect(() => {
        return () => {
            if (pipelineRefreshTimerRef.current !== null) {
                window.clearTimeout(pipelineRefreshTimerRef.current)
                pipelineRefreshTimerRef.current = null
            }
        }
    }, [])

    const { isConnected: isPipelineWsConnected } = useBuildStatusWebSocket({
        enabled: isAdmin,
        onBuildEvent: (event: BuildStatusEvent) => {
            const eventType = String(event?.type || '').toLowerCase()
            if (!eventType) {
                return
            }
            if (eventType.startsWith('build.') || eventType === 'pipeline.health.changed') {
                schedulePipelineRefresh(500)
            }
        },
    })

    if (!groupsLoaded) {
        return (
            <div className="flex items-center justify-center min-h-screen">
                <div className="text-center">
                    <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mb-4"></div>
                    <p className="text-slate-600 dark:text-slate-400">Loading...</p>
                </div>
            </div>
        )
    }

    if (!isAdmin) {
        return <Navigate to="/dashboard" replace />
    }

    // Show loading state only while checking permissions and loading data
    if (loading && isAdmin) {
        return (
            <div className="flex items-center justify-center h-96">
                <div className="text-center">
                    <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mb-4"></div>
                    <p className="text-slate-600 dark:text-slate-400">Loading system data...</p>
                </div>
            </div>
        )
    }

    const releaseAlertSummary = summarizeReleaseAlert(releaseMetrics, releasePolicy)

    return (
        <div className="space-y-8">
            {/* Header */}
            <div>
                <h1 className="text-4xl font-bold text-slate-900 dark:text-white mb-2">Administration Dashboard</h1>
                <p className="text-slate-600 dark:text-slate-400">Welcome, {user?.name || user?.email || 'Administrator'}. Manage your system here.</p>
            </div>

            {/* Stats Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                <StatCard
                    icon="👥"
                    label="Total Users"
                    value={stats.totalUsers}
                    change="+3 this month"
                    color="blue"
                />
                <StatCard
                    icon="🏢"
                    label="Total Tenants"
                    value={stats.totalTenants}
                    change="2 pending approval"
                    color="green"
                />
                <StatCard
                    icon="🔨"
                    label="Active Builds"
                    value={stats.activeBuilds}
                    change="27 completed"
                    color="purple"
                />
                <div className="bg-gradient-to-br from-green-500/10 to-green-500/5 rounded-lg border border-slate-200 dark:border-slate-700/50 p-6">
                    <p className="text-sm font-medium text-slate-600 dark:text-slate-400">System Health</p>
                    <div className="mt-2">
                        <div className="flex items-center gap-2">
                            <span className="w-3 h-3 bg-green-500 rounded-full"></span>
                            <p className="text-2xl font-bold text-slate-900 dark:text-white">{stats.systemHealth.uptime}</p>
                        </div>
                        <p className="text-xs text-slate-600 dark:text-slate-400 mt-3">Uptime</p>
                        <p className="text-xs text-slate-500 dark:text-slate-500 mt-1">
                            Last check: {stats.systemHealth.lastHealthCheck}
                        </p>
                    </div>
                </div>
            </div>

            <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4">
                <div className="flex items-center justify-between gap-3">
                    <div>
                        <h2 className="text-sm font-semibold text-slate-900 dark:text-white">EPR Denial Telemetry</h2>
                        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                            Breakdown of `epr_registration_required` denials by fallback mode and policy scope.
                        </p>
                    </div>
                    <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-700 dark:bg-slate-700 dark:text-slate-200">
                        Total: {sorDenialSummary.total}
                    </span>
                </div>
                <div className="mt-3 grid grid-cols-1 md:grid-cols-2 gap-3 text-xs">
                    <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-3">
                        <p className="font-medium text-slate-800 dark:text-slate-200 mb-2">By Runtime Mode</p>
                        {Object.keys(sorDenialSummary.byMode).length === 0 && (
                            <p className="text-slate-500 dark:text-slate-400">No EPR denial telemetry yet.</p>
                        )}
                        {Object.entries(sorDenialSummary.byMode).map(([mode, count]) => (
                            <div key={mode} className="flex items-center justify-between py-1">
                                <span className="text-slate-600 dark:text-slate-300">{mode}</span>
                                <span className="font-semibold text-slate-900 dark:text-white">{count}</span>
                            </div>
                        ))}
                    </div>
                    <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-3">
                        <p className="font-medium text-slate-800 dark:text-slate-200 mb-2">By Policy Scope</p>
                        {Object.keys(sorDenialSummary.byScope).length === 0 && (
                            <p className="text-slate-500 dark:text-slate-400">No EPR denial telemetry yet.</p>
                        )}
                        {Object.entries(sorDenialSummary.byScope).map(([scope, count]) => (
                            <div key={scope} className="flex items-center justify-between py-1">
                                <span className="text-slate-600 dark:text-slate-300">{scope}</span>
                                <span className="font-semibold text-slate-900 dark:text-white">{count}</span>
                            </div>
                        ))}
                    </div>
                </div>
            </div>

            <div className={`rounded-lg border p-4 ${releaseAlertSummary.level === 'warning'
                ? 'border-amber-200 bg-amber-50 dark:border-amber-700 dark:bg-amber-900/20'
                : 'border-emerald-200 bg-emerald-50 dark:border-emerald-700 dark:bg-emerald-900/20'
                }`}>
                <div className="flex items-center justify-between gap-3">
                    <div>
                        <h2 className="text-sm font-semibold text-slate-900 dark:text-white">Release Governance Telemetry</h2>
                        <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">{releaseAlertSummary.message}</p>
                    </div>
                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${releaseAlertSummary.level === 'warning'
                        ? 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200'
                        : 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200'
                        }`}>
                        {releaseAlertSummary.level === 'warning' ? 'degraded' : 'healthy'}
                    </span>
                </div>
                <div className="mt-3 grid grid-cols-2 md:grid-cols-4 gap-3 text-xs">
                    <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-white/80 dark:bg-slate-900/50 p-2">
                        <p className="text-slate-500 dark:text-slate-400">Requested</p>
                        <p className="mt-1 text-base font-semibold text-slate-900 dark:text-white">{releaseMetrics.requested}</p>
                    </div>
                    <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-white/80 dark:bg-slate-900/50 p-2">
                        <p className="text-slate-500 dark:text-slate-400">Released</p>
                        <p className="mt-1 text-base font-semibold text-slate-900 dark:text-white">{releaseMetrics.released}</p>
                    </div>
                    <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-white/80 dark:bg-slate-900/50 p-2">
                        <p className="text-slate-500 dark:text-slate-400">Failed</p>
                        <p className="mt-1 text-base font-semibold text-slate-900 dark:text-white">{releaseMetrics.failed}</p>
                    </div>
                    <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-white/80 dark:bg-slate-900/50 p-2">
                        <p className="text-slate-500 dark:text-slate-400">Failure Ratio</p>
                        <p className="mt-1 text-base font-semibold text-slate-900 dark:text-white">{releaseAlertSummary.ratioText}</p>
                    </div>
                </div>
                <p className="mt-3 text-[11px] text-slate-600 dark:text-slate-300">
                    Policy window: {releasePolicy.window_minutes}m • Minimum samples: {releasePolicy.minimum_samples} • Consecutive failure threshold: {releasePolicy.consecutive_failures_threshold}
                </p>
            </div>

            <div>
                <div className="mb-4 flex items-center justify-between">
                    <div>
                        <h2 className="text-xl font-bold text-slate-900 dark:text-white">Execution Pipeline Health</h2>
                        {pipelineCheckedAt && (
                            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                Last update: {new Date(pipelineCheckedAt).toLocaleTimeString()}
                            </p>
                        )}
                    </div>
                    <div className="flex items-center gap-2">
                        <button
                            type="button"
                            onClick={() => setAutoRefreshEnabled((value) => !value)}
                            className={`px-3 py-2 text-sm font-medium rounded-md border transition-colors ${autoRefreshEnabled
                                    ? 'border-green-300 bg-green-50 text-green-800 hover:bg-green-100 dark:border-green-700/60 dark:bg-green-900/30 dark:text-green-300 dark:hover:bg-green-900/40'
                                    : 'border-slate-300 text-slate-700 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-700'
                                }`}
                        >
                            Auto refresh: {autoRefreshEnabled ? 'On' : 'Off'}
                        </button>

                        <button
                            type="button"
                            onClick={() => setCompactExecutionCards((v) => !v)}
                            aria-pressed={compactExecutionCards}
                            className={`px-3 py-2 text-sm font-medium rounded-md border transition-colors ${compactExecutionCards
                                    ? 'border-slate-300 bg-slate-50 text-slate-800 dark:bg-slate-900/30 dark:text-slate-200'
                                    : 'border-slate-300 text-slate-700 dark:border-slate-600 dark:text-slate-200'
                                }`}
                        >
                            Compact: {compactExecutionCards ? 'On' : 'Off'}
                        </button>

                        <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-700 dark:bg-slate-700 dark:text-slate-200">
                            {pollIntervalSeconds}s
                        </span>
                        <span
                            className={`px-2 py-0.5 rounded-full text-xs font-medium ${isPipelineWsConnected
                                    ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
                                    : 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300'
                                }`}
                        >
                            Push: {isPipelineWsConnected ? 'Connected' : 'Polling'}
                        </span>
                        <button
                            type="button"
                            onClick={refreshExecutionPipelineHealth}
                            disabled={checkingExecutionPipeline}
                            className="px-3 py-2 text-sm font-medium rounded-md border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-60"
                        >
                            {checkingExecutionPipeline ? 'Refreshing...' : 'Refresh'}
                        </button>
                    </div>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    {runtimeDependencyWatcherHealth && ((runtimeDependencyWatcherHealth.metrics?.runtime_dependency_critical_count ?? 0) > 0 || (runtimeDependencyWatcherHealth.metrics?.runtime_dependency_degraded_count ?? 0) > 0) && (
                        <div className="md:col-span-3 rounded-lg border border-red-200 dark:border-red-700/70 bg-red-50 dark:bg-red-900/20 p-4">
                            <div className="flex items-start justify-between gap-3">
                                <div>
                                    <h3 className="text-sm font-semibold text-red-800 dark:text-red-200">Runtime Dependency Alert</h3>
                                    <p className="mt-1 text-xs text-red-700 dark:text-red-300">
                                        Critical: {runtimeDependencyWatcherHealth.metrics?.runtime_dependency_critical_count ?? 0} | Degraded: {runtimeDependencyWatcherHealth.metrics?.runtime_dependency_degraded_count ?? 0}
                                    </p>
                                    {runtimeDependencyWatcherHealth.message && (
                                        <p className="mt-1 text-xs text-red-700 dark:text-red-300">
                                            {runtimeDependencyWatcherHealth.message}
                                        </p>
                                    )}
                                </div>
                                <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-200">
                                    attention
                                </span>
                            </div>
                        </div>
                    )}
                    {Object.entries(executionPipelineHealth).length === 0 && (
                        <div className="md:col-span-3 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4 text-sm text-slate-600 dark:text-slate-300">
                            Execution pipeline health is unavailable.
                        </div>
                    )}
                    {Object.entries(executionPipelineHealth).map(([key, component]) => {
                        const badgeClass = component.running && component.available
                            ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
                            : component.enabled
                                ? 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300'
                                : 'bg-slate-100 text-slate-700 dark:bg-slate-700 dark:text-slate-200'
                        const statusText = component.running && component.available ? 'running' : component.enabled ? 'degraded' : 'disabled'

                        return (
                            <details key={key} open={!compactExecutionCards} className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
                                <summary className="cursor-pointer list-none px-4 py-3 flex items-center justify-between">
                                    <div className="flex items-center gap-3">
                                        <span className={`w-2 h-2 rounded-full ${component.running && component.available ? 'bg-green-500' : component.enabled ? 'bg-amber-500' : 'bg-slate-400'}`} />
                                        <div>
                                            <h3 className="text-sm font-semibold text-slate-900 dark:text-white">{key.split('_').join(' ')}</h3>
                                            <p className="text-xs text-slate-500 dark:text-slate-400">
                                                {component.last_activity ? new Date(component.last_activity).toLocaleTimeString() : (component.message || '')}
                                            </p>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-3">
                                        <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${badgeClass}`}>{statusText}</span>
                                        <svg xmlns="http://www.w3.org/2000/svg" className="w-4 h-4 text-slate-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                                            <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.292l3.71-4.06a.75.75 0 011.13.98l-4.25 4.653a.75.75 0 01-1.08 0L5.21 8.27a.75.75 0 01.02-1.06z" clipRule="evenodd" />
                                        </svg>
                                    </div>
                                </summary>

                                <div className="px-4 pb-4 pt-2 text-xs text-slate-600 dark:text-slate-400 space-y-1">
                                    <p>Enabled: {component.enabled ? 'Yes' : 'No'}</p>
                                    <p>Available: {component.available ? 'Yes' : 'No'}</p>
                                    {component.mode && <p>Mode: {component.mode}</p>}
                                    {component.last_activity && <p>Last activity: {new Date(component.last_activity).toLocaleString()}</p>}
                                    {component.message && <p className="text-slate-500 dark:text-slate-400">{component.message}</p>}
                                </div>
                            </details>
                        )
                    })}
                </div>
                {(notificationSubscriberHealth || monitorSubscriberHealth || outboxRelayHealth) && (
                    <div className="mt-4 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4">
                        <div className="flex items-center justify-between gap-3">
                            <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Build Notification Diagnostics</h3>
                            <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-700 dark:bg-slate-700 dark:text-slate-200">
                                runtime
                            </span>
                        </div>
                        <div className="mt-3 grid grid-cols-1 md:grid-cols-3 gap-3 text-xs">
                            {notificationSubscriberHealth && (
                                <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-3">
                                    <p className="text-slate-500 dark:text-slate-400">Notification Subscriber</p>
                                    <p className="mt-1 font-semibold text-slate-900 dark:text-white">
                                        {notificationSubscriberHealth.running ? 'Running' : 'Stopped'}
                                    </p>
                                    <p className="mt-1 text-slate-600 dark:text-slate-300">
                                        Events: {notificationSubscriberHealth.metrics?.build_notification_events_received_total ?? 0}
                                    </p>
                                    <p className="text-slate-600 dark:text-slate-300">
                                        In-app: {notificationSubscriberHealth.metrics?.build_notification_in_app_delivered_total ?? 0}
                                    </p>
                                    <p className="text-slate-600 dark:text-slate-300">
                                        Email queued: {notificationSubscriberHealth.metrics?.build_notification_email_queued_total ?? 0}
                                    </p>
                                    <p className="text-slate-600 dark:text-slate-300">
                                        Failures: {notificationSubscriberHealth.metrics?.build_notification_failures_total ?? 0}
                                    </p>
                                </div>
                            )}
                            {monitorSubscriberHealth && (
                                <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-3">
                                    <p className="text-slate-500 dark:text-slate-400">Build Monitor Subscriber</p>
                                    <p className="mt-1 font-semibold text-slate-900 dark:text-white">
                                        {monitorSubscriberHealth.running ? 'Running' : 'Stopped'}
                                    </p>
                                    <p className="mt-1 text-slate-600 dark:text-slate-300">
                                        Transitions: {monitorSubscriberHealth.metrics?.monitor_event_driven_transitions_total ?? 0}
                                    </p>
                                    <p className="text-slate-600 dark:text-slate-300">
                                        Parse failures: {monitorSubscriberHealth.metrics?.monitor_event_driven_parse_failures ?? 0}
                                    </p>
                                </div>
                            )}
                            {outboxRelayHealth && (
                                <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-3">
                                    <p className="text-slate-500 dark:text-slate-400">Messaging Outbox Relay</p>
                                    <p className="mt-1 font-semibold text-slate-900 dark:text-white">
                                        {outboxRelayHealth.running ? 'Running' : 'Stopped'}
                                    </p>
                                    <p className="mt-1 text-slate-600 dark:text-slate-300">
                                        Pending (lag): {outboxRelayHealth.metrics?.messaging_outbox_pending_count ?? 0}
                                    </p>
                                    <p className="text-slate-600 dark:text-slate-300">
                                        Replay failures: {outboxRelayHealth.metrics?.messaging_outbox_replay_failures_total ?? 0}
                                    </p>
                                </div>
                            )}
                        </div>
                    </div>
                )}
                {workflowDiagnostics && (
                    <div className="mt-4 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4">
                        <div className="flex items-center justify-between gap-3">
                            <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Build Workflow Control Plane</h3>
                            <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${workflowDiagnostics.blocked_step_count > 0
                                    ? 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300'
                                    : 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
                                }`}>
                                {workflowDiagnostics.blocked_step_count > 0 ? 'attention' : 'healthy'}
                            </span>
                        </div>
                        <div className="mt-3 grid grid-cols-1 md:grid-cols-4 gap-3 text-xs">
                            <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-2">
                                <p className="text-slate-500 dark:text-slate-400">Blocked Steps</p>
                                <p className="text-lg font-semibold text-slate-900 dark:text-white">{workflowDiagnostics.blocked_step_count}</p>
                            </div>
                            <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-2">
                                <p className="text-slate-500 dark:text-slate-400">Dispatch</p>
                                <p className="text-lg font-semibold text-slate-900 dark:text-white">{workflowDiagnostics.dispatch_blocked}</p>
                            </div>
                            <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-2">
                                <p className="text-slate-500 dark:text-slate-400">Monitor</p>
                                <p className="text-lg font-semibold text-slate-900 dark:text-white">{workflowDiagnostics.monitor_blocked}</p>
                            </div>
                            <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-2">
                                <p className="text-slate-500 dark:text-slate-400">Finalize</p>
                                <p className="text-lg font-semibold text-slate-900 dark:text-white">{workflowDiagnostics.finalize_blocked}</p>
                            </div>
                        </div>
                        <div className="mt-3 text-xs text-slate-600 dark:text-slate-300 space-y-1">
                            {workflowDiagnostics.oldest_blocked_at && (
                                <p>Oldest blocked at: {new Date(workflowDiagnostics.oldest_blocked_at).toLocaleString()}</p>
                            )}
                            {workflowDiagnostics.oldest_blocked_age_seconds !== undefined && workflowDiagnostics.oldest_blocked_age_seconds > 0 && (
                                <p>Oldest blocked age: {Math.floor(workflowDiagnostics.oldest_blocked_age_seconds / 60)} min</p>
                            )}
                            {workflowDiagnostics.recovery_action && (
                                <p>
                                    Recovery action: <span className="font-semibold text-slate-900 dark:text-white">{workflowDiagnostics.recovery_action}</span>
                                </p>
                            )}
                            {workflowDiagnostics.recovery_hint && (
                                <p className="text-amber-700 dark:text-amber-300">{workflowDiagnostics.recovery_hint}</p>
                            )}
                            {workflowDiagnostics.recovery_action === 'start_dispatcher' && (
                                <div className="pt-1">
                                    <button
                                        type="button"
                                        onClick={executeRecoveryAction}
                                        disabled={recoveryInProgress}
                                        className="px-3 py-1.5 text-xs font-medium rounded-md border border-blue-300 dark:border-blue-600 text-blue-700 dark:text-blue-300 hover:bg-blue-50 dark:hover:bg-blue-900/30 disabled:opacity-60"
                                    >
                                        {recoveryInProgress ? 'Starting dispatcher...' : 'Start Dispatcher'}
                                    </button>
                                </div>
                            )}
                            {workflowDiagnostics.recovery_action === 'start_orchestrator' && (
                                <div className="pt-1">
                                    <button
                                        type="button"
                                        onClick={executeRecoveryAction}
                                        disabled={recoveryInProgress}
                                        className="px-3 py-1.5 text-xs font-medium rounded-md border border-blue-300 dark:border-blue-600 text-blue-700 dark:text-blue-300 hover:bg-blue-50 dark:hover:bg-blue-900/30 disabled:opacity-60"
                                    >
                                        {recoveryInProgress ? 'Starting orchestrator...' : 'Start Orchestrator'}
                                    </button>
                                </div>
                            )}
                        </div>
                    </div>
                )}
                {orchestratorHealth && (
                    <div className="mt-4 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4">
                        <div className="flex items-center justify-between gap-3">
                            <div>
                                <h3 className="text-sm font-semibold text-slate-900 dark:text-white">Workflow Orchestrator Control</h3>
                                <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">
                                    {orchestratorHealth.enabled
                                        ? orchestratorHealth.running
                                            ? 'Orchestrator is running.'
                                            : 'Orchestrator is enabled but not running.'
                                        : 'Orchestrator is disabled by configuration.'}
                                </p>
                            </div>
                            <button
                                type="button"
                                onClick={toggleOrchestrator}
                                disabled={!orchestratorHealth.enabled || orchestratorActionInProgress}
                                className="px-3 py-1.5 text-xs font-medium rounded-md border border-blue-300 dark:border-blue-600 text-blue-700 dark:text-blue-300 hover:bg-blue-50 dark:hover:bg-blue-900/30 disabled:opacity-60"
                            >
                                {orchestratorActionInProgress
                                    ? orchestratorHealth.running
                                        ? 'Stopping...'
                                        : 'Starting...'
                                    : orchestratorHealth.running
                                        ? 'Stop Orchestrator'
                                        : 'Start Orchestrator'}
                            </button>
                        </div>
                    </div>
                )}
            </div>

            <div>
                <div className="mb-4 flex items-center justify-between">
                    <h2 className="text-xl font-bold text-slate-900 dark:text-white">System Components Status</h2>
                    <div className="flex items-center gap-2">
                        <button
                            type="button"
                            onClick={() => setCompactSystemComponentCards((v) => !v)}
                            aria-pressed={compactSystemComponentCards}
                            className={`px-3 py-2 text-sm font-medium rounded-md border transition-colors ${compactSystemComponentCards
                                    ? 'border-slate-300 bg-slate-50 text-slate-800 dark:bg-slate-900/30 dark:text-slate-200'
                                    : 'border-slate-300 text-slate-700 dark:border-slate-600 dark:text-slate-200'
                                }`}
                        >
                            Compact: {compactSystemComponentCards ? 'On' : 'Off'}
                        </button>
                        <button
                            type="button"
                            onClick={refreshComponentStatus}
                            disabled={checkingComponents}
                            className="px-3 py-2 text-sm font-medium rounded-md border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-60"
                        >
                            {checkingComponents ? 'Checking...' : 'Check now'}
                        </button>
                    </div>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                    {Object.entries(componentStatus).length === 0 && (
                        <div className="col-span-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4 text-sm text-slate-600 dark:text-slate-300">
                            Component status is unavailable.
                        </div>
                    )}
                    {Object.entries(componentStatus).map(([key, component]) => {
                        const badgeClass = component.status === 'healthy'
                            ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
                            : component.status === 'critical'
                                ? 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300'
                                : 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300'

                        return (
                            <details key={key} open={!compactSystemComponentCards} className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
                                <summary className="cursor-pointer list-none px-4 py-3 flex items-center justify-between">
                                    <div className="flex items-center gap-3">
                                        <span className={`w-2 h-2 rounded-full ${component.status === 'healthy' ? 'bg-green-500' : component.status === 'critical' ? 'bg-red-500' : 'bg-amber-500'}`} />
                                        <div>
                                            <h3 className="text-sm font-semibold text-slate-900 dark:text-white">{component.name}</h3>
                                            <p className="text-xs text-slate-500 dark:text-slate-400">
                                                {component.message || (component.configured ? 'Configured' : 'Not configured')}
                                            </p>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-3">
                                        <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${badgeClass}`}>{component.status}</span>
                                        <svg xmlns="http://www.w3.org/2000/svg" className="w-4 h-4 text-slate-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                                            <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.292l3.71-4.06a.75.75 0 011.13.98l-4.25 4.653a.75.75 0 01-1.08 0L5.21 8.27a.75.75 0 01.02-1.06z" clipRule="evenodd" />
                                        </svg>
                                    </div>
                                </summary>
                                <div className="px-4 pb-4 pt-2 text-xs text-slate-600 dark:text-slate-400 space-y-2">
                                    <p>{component.message || (component.configured ? 'Configured' : 'Not configured')}</p>
                                    {component.endpoint && (
                                        <p className="break-all">
                                            Endpoint: {component.endpoint}
                                        </p>
                                    )}
                                    <div className="flex flex-wrap gap-3 text-slate-500 dark:text-slate-500">
                                        {component.httpStatus !== undefined && <span>HTTP: {component.httpStatus}</span>}
                                        {component.latencyMs !== undefined && <span>Latency: {component.latencyMs}ms</span>}
                                        <span>Last Check: {new Date(component.lastCheck).toLocaleTimeString()}</span>
                                    </div>
                                    {key === 'internal_registry_gc_worker' && component.details && (
                                        <div className="rounded-md border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/40 p-2 text-slate-600 dark:text-slate-300">
                                            <p>Last run deleted: {Number(component.details.last_run_deleted || 0)}</p>
                                            <p>Last reclaimed: ~{formatBytes(Number(component.details.last_run_reclaimed_bytes || 0))}</p>
                                            <p>Total deleted: {Number(component.details.total_deleted || 0)}</p>
                                            <p>Total reclaimed: ~{formatBytes(Number(component.details.total_reclaimed_bytes || 0))}</p>
                                        </div>
                                    )}
                                </div>
                            </details>
                        )
                    })}
                </div>
            </div>

            {/* Quick Actions */}
            <div>
                <h2 className="text-xl font-bold text-slate-900 dark:text-white mb-4">Quick Actions</h2>
                <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 2xl:grid-cols-5 gap-3">
                    <AdminQuickLink
                        icon="👥"
                        title="User Management"
                        description="Add, edit, or manage users and their roles"
                        href="/admin/users"
                        color="blue"
                    />
                    <AdminQuickLink
                        icon="🏢"
                        title="Tenant Management"
                        description="Configure and manage tenants"
                        href="/admin/tenants"
                        color="green"
                    />
                    <AdminQuickLink
                        icon="🔐"
                        title="Roles & Permissions"
                        description="Define and manage system roles"
                        href="/admin/access/roles"
                        color="purple"
                    />
                    <AdminQuickLink
                        icon="⚙️"
                        title="System Configuration"
                        description="Configure system settings and parameters"
                        href="/admin/system-config"
                        color="orange"
                    />
                    <AdminQuickLink
                        icon="📋"
                        title="Audit Logs"
                        description="Review system audit and activity logs"
                        href="/admin/audit-logs"
                        color="blue"
                    />
                    <AdminQuickLink
                        icon="�"
                        title="Tool Management"
                        description="Configure available build tools and services"
                        href="/admin/tools"
                        color="orange"
                    />
                    <AdminQuickLink
                        icon="⚙️"
                        title="Build Management"
                        description="Monitor and manage all system builds"
                        href="/admin/builds"
                        color="purple"
                    />
                </div>
            </div>

            {/* Recent Activity Section */}
            <div>
                <h2 className="text-xl font-bold text-slate-900 dark:text-white mb-4">Recent Activity</h2>
                <div className="bg-slate-50 dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700/50 p-6">
                    <div className="text-center text-slate-600 dark:text-slate-400 py-8">
                        <p className="text-sm">Activity data will appear here as soon as there is user activity.</p>
                    </div>
                </div>
            </div>

            {/* System Alerts */}
            <div>
                <h2 className="text-xl font-bold text-slate-900 dark:text-white mb-4">System Alerts</h2>
                <div className="bg-green-500/10 border border-green-500/30 rounded-lg p-4 flex items-start gap-4">
                    <span className="text-2xl">✓</span>
                    <div>
                        <h3 className="font-semibold text-green-300">System Healthy</h3>
                        <p className="text-sm text-green-200/80 mt-1">All systems are operating normally.</p>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default AdminDashboardPage
