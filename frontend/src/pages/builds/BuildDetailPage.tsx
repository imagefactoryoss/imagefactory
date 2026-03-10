import ConfirmDialog from '@/components/common/ConfirmDialog'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import Drawer from '@/components/ui/Drawer'
import useBuildStatusWebSocket from '@/hooks/useBuildStatusWebSocket'
import { adminService } from '@/services/adminService'
import { buildService } from '@/services/buildService'
import { projectService } from '@/services/projectService'
import { useAuthStore } from '@/store/auth'
import { useOperationCapabilitiesStore } from '@/store/operationCapabilities'
import { useTenantStore } from '@/store/tenant'
import { Build, BuildExecutionAttempt, BuildStatus, BuildTraceResponse, BuildWorkflowResponse, BuildTraceRuntimeComponent, ProjectSource, WorkflowStepStatus } from '@/types'
import { canCreateBuilds } from '@/utils/permissions'
import { Ban, Check, ChevronDown, ChevronUp, Circle, Clock, Copy, FileText, Info, Play, Plus, RefreshCcw, Trash2, X, XCircle } from 'lucide-react'
import React, { useEffect, useMemo, useRef, useState } from 'react'
import toast from 'react-hot-toast'
import { Link, useNavigate, useParams } from 'react-router-dom'

const WORKFLOW_STEP_LABELS: Record<string, string> = {
    'build.validate': 'Validate Build',
    'build.select_infrastructure': 'Select Infrastructure',
    'build.enqueue': 'Queue Build',
    'build.dispatch': 'Dispatch Build',
    'build.monitor': 'Monitor Execution',
    'build.finalize': 'Finalize Build',
}

const WORKFLOW_STEP_ORDER = [
    'build.validate',
    'build.select_infrastructure',
    'build.enqueue',
    'build.dispatch',
    'build.monitor',
    'build.finalize',
]

const WORKFLOW_DAG_STEPS: Array<{ key: string; label: string; description: string }> = [
    { key: 'build.validate', label: 'Validate', description: 'Input policy + schema checks' },
    { key: 'build.select_infrastructure', label: 'Select Infra', description: 'Provider + namespace selection' },
    { key: 'build.enqueue', label: 'Enqueue', description: 'Queue admission + ordering' },
    { key: 'build.dispatch', label: 'Dispatch', description: 'Launch execution attempt' },
    { key: 'build.monitor', label: 'Monitor', description: 'Track runtime progress' },
    { key: 'build.finalize', label: 'Finalize', description: 'Persist artifacts + status' },
]

const EXECUTION_STAGE_ORDER: Record<string, number> = {
    queued: 0,
    validated: 1,
    dispatched: 2,
    build: 3,
    scan: 4,
    sbom: 5,
    publish: 6,
    complete: 7,
}

const inferStageFromTektonLog = (entry: { message?: string; metadata?: Record<string, any> }): string | undefined => {
    const metadata = entry.metadata || {}
    const taskRun = String(metadata.task_run || metadata.taskRun || '').toLowerCase()
    const step = String(metadata.step || '').toLowerCase()
    const message = String(entry.message || '').toLowerCase()

    const signal = `${taskRun} ${step} ${message}`

    if (signal.includes('sbom')) return 'sbom'
    if (signal.includes('scan')) return 'scan'
    if (signal.includes('push')) return 'publish'

    if (signal.includes('build-and-push') || signal.includes('-build')) return 'build'
    if (signal.includes('clone')) return 'dispatched'
    return undefined
}

const resolveExecutionLogTimestamp = (entry: { timestamp?: string; metadata?: Record<string, any> }): number | undefined => {
    const metadataTs = typeof entry.metadata?.timestamp === 'string' ? entry.metadata.timestamp.trim() : ''
    const ts = metadataTs || (entry.timestamp || '').trim()
    if (!ts) return undefined
    const parsed = Date.parse(ts)
    return Number.isNaN(parsed) ? undefined : parsed
}

const formatCompactDuration = (durationMs?: number): string => {
    if (durationMs === undefined || durationMs < 0) return '-'
    const totalSeconds = Math.floor(durationMs / 1000)
    if (totalSeconds < 60) return `${totalSeconds}s`
    const minutes = Math.floor(totalSeconds / 60)
    const seconds = totalSeconds % 60
    if (minutes < 60) return `${minutes}m ${seconds}s`
    const hours = Math.floor(minutes / 60)
    const remMinutes = minutes % 60
    return `${hours}h ${remMinutes}m`
}

const compactTaskRunName = (taskRunName: string): string => {
    const match = taskRunName.match(/-run-[^-]+-(.+)$/)
    if (match?.[1]) return match[1]
    return taskRunName
}

const parseTimestampMs = (value?: string | null): number | undefined => {
    if (!value) return undefined
    const parsed = Date.parse(value)
    return Number.isNaN(parsed) ? undefined : parsed
}

const formatClockTime = (timestampMs?: number): string => {
    if (timestampMs === undefined) return '-'
    return new Date(timestampMs).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

type TraceSeverity = 'info' | 'success' | 'warn' | 'error'

interface BuildTraceEvent {
    timestamp: string
    title: string
    detail: string
    severity: TraceSeverity
}

// Structured log entry (keeps Tekton metadata if present)
interface LogEntry {
    timestamp?: string
    level?: string
    message: string
    metadata?: Record<string, any>
}

type PipelineRuntimeComponent = BuildTraceRuntimeComponent & {
    available?: boolean
}
type BuildLogTab = 'lifecycle' | 'execution'

const BuildDetailPage: React.FC = () => {
    const { buildId } = useParams<{ buildId: string }>()
    const navigate = useNavigate()
    const { groups, token } = useAuthStore()
    const operationCapabilities = useOperationCapabilitiesStore((state) => state.capabilities)
    const { selectedTenantId } = useTenantStore()
    const canCreateBuild = canCreateBuilds() && operationCapabilities.build
    const [build, setBuild] = useState<Build | null>(null)
    const [loading, setLoading] = useState(true)
    const [lifecycleLogs, setLifecycleLogs] = useState<LogEntry[]>([])
    const [executionLogs, setExecutionLogs] = useState<LogEntry[]>([])
    const [attempts, setAttempts] = useState<BuildExecutionAttempt[]>([])
    const [selectedExecutionId, setSelectedExecutionId] = useState<string | undefined>(undefined)
    const [autoRefresh, setAutoRefresh] = useState(false)
    const [showDockerfileDrawer, setShowDockerfileDrawer] = useState(false)
    const [projectSources, setProjectSources] = useState<ProjectSource[]>([])
    const [workflow, setWorkflow] = useState<BuildWorkflowResponse | null>(null)
    const [traceDiagnostics, setTraceDiagnostics] = useState<BuildTraceResponse['diagnostics'] | undefined>(undefined)
    const [traceCorrelation, setTraceCorrelation] = useState<BuildTraceResponse['correlation'] | undefined>(undefined)
    const [pipelineComponents, setPipelineComponents] = useState<Record<string, PipelineRuntimeComponent>>({})
    const [pipelineCheckedAt, setPipelineCheckedAt] = useState<string | null>(null)
    const [wsReloadCounter, setWsReloadCounter] = useState(0)
    const [isLogStreamConnected, setIsLogStreamConnected] = useState(false)
    const [activeLogTab, setActiveLogTab] = useState<BuildLogTab>('execution')
    const [expandedLogMetadata, setExpandedLogMetadata] = useState<Record<string, boolean>>({})
    const [expandedExecutionLogTasks, setExpandedExecutionLogTasks] = useState<Record<string, boolean>>({})
    const [traceExpanded, setTraceExpanded] = useState(false)
    const [expandedExecutionStages, setExpandedExecutionStages] = useState<Record<string, boolean>>({})
    const [workflowDagView, setWorkflowDagView] = useState<'auto' | 'shown' | 'hidden'>('auto')
    const [dispatcherRecoveryLoading, setDispatcherRecoveryLoading] = useState(false)
    const [orchestratorRecoveryLoading, setOrchestratorRecoveryLoading] = useState(false)
    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
    const confirmDialog = useConfirmDialog()
    const selectedExecutionIdRef = useRef<string | undefined>(undefined)
    const logStreamSocketRef = useRef<WebSocket | null>(null)
    const logStreamReconnectTimerRef = useRef<number | undefined>(undefined)
    const logStreamReconnectAttemptRef = useRef(0)

    useEffect(() => {
        if (buildId) {
            loadBuild()
        }
    }, [buildId, wsReloadCounter])

    useEffect(() => {
        let interval: NodeJS.Timeout
        if (autoRefresh && build && (build.status === 'running' || build.status === 'pending' || build.status === 'queued')) {
            interval = setInterval(() => {
                loadBuild()
            }, 5000) // Refresh every 5 seconds
        }
        return () => {
            if (interval) clearInterval(interval)
        }
    }, [autoRefresh, build])

    const loadExecutionLogTabs = async (targetBuildID: string, executionID: string) => {
        const [lifecycleData, executionData] = await Promise.all([
            buildService.getBuildLogEntries(targetBuildID, executionID, { source: 'lifecycle' }),
            buildService.getBuildLogEntries(targetBuildID, executionID, { source: 'tekton' }),
        ])
        const normalizeEntries = (entries: Array<{ timestamp?: string; level?: string; message?: string; metadata?: Record<string, any> }>): LogEntry[] =>
            entries.map((entry) => ({
                timestamp: entry.timestamp,
                level: entry.level,
                message: entry.message || '',
                metadata: entry.metadata,
            }))
        setLifecycleLogs(normalizeEntries(lifecycleData.entries || []))
        setExecutionLogs(normalizeEntries(executionData.entries || []))
    }

    useEffect(() => {
        const loadSelectedExecution = async () => {
            if (!buildId || !selectedExecutionId) return
            try {
                const [_, workflowData] = await Promise.all([
                    loadExecutionLogTabs(buildId, selectedExecutionId),
                    buildService.getBuildWorkflow(buildId, selectedExecutionId).catch(() => null)
                ])
                setWorkflow(workflowData)
            } catch {
                setLifecycleLogs([])
                setExecutionLogs([])
                setWorkflow(null)
            }
        }
        loadSelectedExecution()
    }, [buildId, selectedExecutionId])

    useEffect(() => {
        selectedExecutionIdRef.current = selectedExecutionId
    }, [selectedExecutionId])

    useEffect(() => {
        if (!buildId || !token || !selectedTenantId) {
            setIsLogStreamConnected(false)
            return
        }

        let active = true
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        const wsUrl = `${protocol}//${window.location.host}/api/v1/builds/${buildId}/logs/stream?token=${encodeURIComponent(token)}&tenant_id=${encodeURIComponent(selectedTenantId)}`
        const appendDedupedEntry = (current: LogEntry[], entry: LogEntry): LogEntry[] => {
            const last = current[current.length - 1]
            if (last && last.timestamp === entry.timestamp && last.message === entry.message) {
                return current
            }
            return [...current, entry]
        }

        const connectLogStream = () => {
            if (!active) return
            const existing = logStreamSocketRef.current
            if (existing && (existing.readyState === WebSocket.OPEN || existing.readyState === WebSocket.CONNECTING)) {
                return
            }

            const socket = new WebSocket(wsUrl)
            logStreamSocketRef.current = socket

            socket.onopen = () => {
                if (!active) return
                logStreamReconnectAttemptRef.current = 0
                setIsLogStreamConnected(true)
            }

            socket.onclose = (event) => {
                if (!active) return
                if (logStreamSocketRef.current === socket) {
                    logStreamSocketRef.current = null
                }
                setIsLogStreamConnected(false)

                if (event.code === 1008 || event.code === 4001 || event.code === 4401 || event.code === 4403) {
                    return
                }

                logStreamReconnectAttemptRef.current += 1
                const delayMs = Math.min(15000, 1000 * Math.max(1, 2 ** (logStreamReconnectAttemptRef.current - 1)))
                if (logStreamReconnectTimerRef.current) {
                    window.clearTimeout(logStreamReconnectTimerRef.current)
                }
                logStreamReconnectTimerRef.current = window.setTimeout(() => {
                    connectLogStream()
                }, delayMs)
            }

            socket.onerror = () => {
                if (!active) return
                setIsLogStreamConnected(false)
            }

            socket.onmessage = (event) => {
                try {
                    const payload = JSON.parse(event.data)
                    if (!payload || payload.build_id !== buildId) return
                    const executionId = payload?.metadata?.execution_id || payload?.metadata?.executionId
                    if (!selectedExecutionIdRef.current && typeof executionId === 'string' && executionId.trim() !== '') {
                        setSelectedExecutionId(executionId)
                    }
                    if (selectedExecutionIdRef.current && executionId && executionId !== selectedExecutionIdRef.current) return

                    const entry = {
                        timestamp: payload.timestamp || undefined,
                        level: payload.level ? String(payload.level).toUpperCase() : undefined,
                        message: payload.message || '',
                        metadata: payload.metadata || undefined,
                    }
                    const source = String(entry.metadata?.source || '').toLowerCase()
                    if (source === 'tekton') {
                        setExecutionLogs((current) => appendDedupedEntry(current, entry))
                    } else {
                        setLifecycleLogs((current) => appendDedupedEntry(current, entry))
                    }
                } catch {
                    // ignore malformed stream payloads
                }
            }
        }

        connectLogStream()

        return () => {
            active = false
            if (logStreamReconnectTimerRef.current) {
                window.clearTimeout(logStreamReconnectTimerRef.current)
                logStreamReconnectTimerRef.current = undefined
            }
            if (logStreamSocketRef.current) {
                logStreamSocketRef.current.close()
                logStreamSocketRef.current = null
            }
            setIsLogStreamConnected(false)
        }
    }, [buildId, token, selectedTenantId])

    const { isConnected: isWsConnected } = useBuildStatusWebSocket({
        enabled: !!buildId,
        filterBuildId: buildId,
        onBuildEvent: () => setWsReloadCounter((current) => current + 1),
    })

    const loadBuild = async (preferredExecutionId?: string) => {
        if (!buildId) return

        try {
            const traceData = await buildService.getBuildTrace(buildId)
            setBuild(traceData.build)
            setAttempts(traceData.executions || [])
            setWorkflow(traceData.workflow || null)
            setTraceDiagnostics(traceData.diagnostics)
            setTraceCorrelation(traceData.correlation)
            setPipelineComponents(traceData.runtime || {})
            setPipelineCheckedAt(traceData.runtime ? new Date().toISOString() : null)
            if (traceData.build?.projectId) {
                try {
                    const sources = await projectService.listProjectSources(traceData.build.projectId)
                    setProjectSources(sources)
                } catch {
                    setProjectSources([])
                }
            } else {
                setProjectSources([])
            }

            const nextExecutionId = (() => {
                if (preferredExecutionId && traceData.executions.some((execution) => execution.id === preferredExecutionId)) {
                    return preferredExecutionId
                }
                if (selectedExecutionId && traceData.executions.some((execution) => execution.id === selectedExecutionId)) {
                    return selectedExecutionId
                }
                return traceData.selectedExecutionId || traceData.executions[0]?.id
            })()
            setSelectedExecutionId(nextExecutionId)

            if (!nextExecutionId) {
                setLifecycleLogs([])
                setExecutionLogs([])
                setWorkflow(null)
                setTraceCorrelation(undefined)
            } else if (nextExecutionId === selectedExecutionId) {
                await loadExecutionLogTabs(buildId, nextExecutionId)
                setTraceCorrelation((prev) => ({
                    ...prev,
                    executionId: nextExecutionId,
                }))
            }
        } catch (error: any) {
            toast.error('Failed to load build details')
        } finally {
            setLoading(false)
        }
    }

    const handleCancelBuild = async () => {
        if (!build) return
        const confirmed = await confirmDialog({
            title: 'Cancel Build',
            message: 'Are you sure you want to cancel this build?',
            confirmLabel: 'Cancel Build',
            destructive: true,
        })
        if (!confirmed) return

        try {
            await buildService.cancelBuild(build.id)
            toast.success('Build cancelled successfully')
            loadBuild()
        } catch (error: any) {
            toast.error(error.message || 'Failed to cancel build')
        }
    }

    const handleStartBuild = async () => {
        if (!build) return
        const confirmed = await confirmDialog({
            title: 'Start Build',
            message: 'Are you sure you want to start this build?',
            confirmLabel: 'Start Build',
        })
        if (!confirmed) return

        try {
            await buildService.startBuild(build.id)
            toast.success('Build started successfully')
            loadBuild()
        } catch (error: any) {
            toast.error(error.message || 'Failed to start build')
        }
    }

    const handleRetryBuild = async () => {
        if (!build) return
        const confirmed = await confirmDialog({
            title: 'Retry Build',
            message: 'Retry this build with the same configuration?',
            confirmLabel: 'Retry Build',
        })
        if (!confirmed) return

        try {
            await buildService.retryBuild(build.id)
            const executionData = await buildService.getBuildExecutions(build.id, { limit: 25, offset: 0 })
            const latestAttemptNumber = executionData.executions.length
            const latestExecutionId = executionData.executions[0]?.id
            if (latestExecutionId) {
                setSelectedExecutionId(latestExecutionId)
                setLifecycleLogs([])
                setExecutionLogs([])
            }
            await loadBuild(latestExecutionId)
            toast.success(`Started attempt #${latestAttemptNumber}`)
        } catch (error: any) {
            toast.error(error.message || 'Failed to retry build')
        }
    }

    const handleCloneBuild = () => {
        if (!build) return
        void (async () => {
            const confirmed = await confirmDialog({
                title: 'Clone Build Config',
                message: 'Open this build in the create wizard with prefilled configuration?',
                confirmLabel: 'Open Wizard',
            })
            if (!confirmed) return
            navigate(`/builds/new?cloneFrom=${encodeURIComponent(build.id)}`)
        })()
    }

    const canDeleteBuild = (status: string) => status !== 'running' && status !== 'queued'

    const handleDeleteBuild = async () => {
        if (!build) return
        if (!canDeleteBuild(build.status)) {
            toast.error('Cannot delete build while execution is running or queued')
            return
        }
        setShowDeleteConfirm(true)
    }

    const confirmDeleteBuild = async () => {
        if (!build) return
        try {
            await buildService.deleteBuild(build.id)
            toast.success('Build deleted successfully')
            navigate('/builds')
        } catch (error: any) {
            toast.error(error.message || 'Failed to delete build')
        } finally {
            setShowDeleteConfirm(false)
        }
    }

    const getStatusColor = (status: BuildStatus): string => {
        switch (status) {
            case 'completed': return 'text-green-700 bg-green-100 dark:text-green-300 dark:bg-green-900/30'
            case 'running': return 'text-blue-700 bg-blue-100 dark:text-blue-300 dark:bg-blue-900/30'
            case 'failed': return 'text-red-700 bg-red-100 dark:text-red-300 dark:bg-red-900/30'
            case 'cancelled': return 'text-slate-700 bg-slate-100 dark:text-slate-300 dark:bg-slate-700/60'
            case 'pending': return 'text-amber-700 bg-amber-100 dark:text-amber-300 dark:bg-amber-900/30'
            case 'queued': return 'text-violet-700 bg-violet-100 dark:text-violet-300 dark:bg-violet-900/30'
            default: return 'text-slate-700 bg-slate-100 dark:text-slate-300 dark:bg-slate-700/60'
        }
    }

    const formatDurationFromTimes = (startedAt?: string, completedAt?: string): string => {
        if (!startedAt) return '-'

        const startTime = new Date(startedAt)
        const endTime = completedAt ? new Date(completedAt) : new Date()

        const durationMs = endTime.getTime() - startTime.getTime()
        const minutes = Math.floor(durationMs / 60000)
        const seconds = Math.floor((durationMs % 60000) / 1000)

        return `${minutes}m ${seconds}s`
    }

    const selectedAttempt = selectedExecutionId
        ? attempts.find((attempt) => attempt.id === selectedExecutionId)
        : attempts[0]
    const visibleLogs = activeLogTab === 'execution' ? executionLogs : lifecycleLogs
    const executionTaskGroups = useMemo(() => {
        const taskMap: Map<string, Map<string, LogEntry[]>> = new Map()
        executionLogs.forEach((entry) => {
            const task = (entry.metadata && (entry.metadata.task_run || entry.metadata.task))
                ? String(entry.metadata.task_run || entry.metadata.task)
                : '__ungrouped__'
            const step = (entry.metadata && entry.metadata.step) ? String(entry.metadata.step) : '__other__'
            if (!taskMap.has(task)) taskMap.set(task, new Map())
            const stepMap = taskMap.get(task) as Map<string, LogEntry[]>
            if (!stepMap.has(step)) stepMap.set(step, [])
            ; (stepMap.get(step) as LogEntry[]).push(entry)
        })
        return Array.from(taskMap.entries())
    }, [executionLogs])
    const isBuildActive = build?.status === 'running' || build?.status === 'queued' || build?.status === 'pending'

    const toggleLogMetadata = (key: string) => {
        setExpandedLogMetadata((current) => ({ ...current, [key]: !current[key] }))
    }
    const toggleExecutionLogTask = (taskRun: string) => {
        setExpandedExecutionLogTasks((current) => ({ ...current, [taskRun]: !current[taskRun] }))
    }
    const expandAllExecutionLogTasks = () => {
        const next: Record<string, boolean> = {}
        executionTaskGroups.forEach(([taskRun]) => {
            next[taskRun] = true
        })
        setExpandedExecutionLogTasks(next)
    }
    const collapseAllExecutionLogTasks = () => {
        const next: Record<string, boolean> = {}
        executionTaskGroups.forEach(([taskRun]) => {
            next[taskRun] = false
        })
        setExpandedExecutionLogTasks(next)
    }
    const toggleExecutionStageDetails = (stageKey: string) => {
        setExpandedExecutionStages((current) => ({ ...current, [stageKey]: !current[stageKey] }))
    }

    useEffect(() => {
        setExpandedExecutionLogTasks((current) => {
            const next: Record<string, boolean> = {}
            executionTaskGroups.forEach(([taskRun], index) => {
                if (Object.prototype.hasOwnProperty.call(current, taskRun)) {
                    next[taskRun] = current[taskRun]
                } else {
                    next[taskRun] = index < 4
                }
            })
            return next
        })
    }, [executionTaskGroups])

    const formatDate = (dateString: string): string => {
        return new Date(dateString).toLocaleString()
    }

    const buildStages = [
        { key: 'queued', label: 'Queued', description: 'Awaiting capacity' },
        { key: 'validated', label: 'Validated', description: 'Config checks' },
        { key: 'dispatched', label: 'Dispatched', description: 'Worker selected' },
        { key: 'build', label: 'Build', description: 'Image build running' },
        { key: 'scan', label: 'Scan', description: 'Security checks' },
        { key: 'sbom', label: 'SBOM', description: 'Metadata capture' },
        { key: 'publish', label: 'Publish', description: 'Registry push' },
        { key: 'complete', label: 'Complete', description: 'Ready to use' },
    ] as const
    const stageTiming = (() => {
        type StageStat = { startAt: number; endAt: number; taskRuns: Set<string> }
        type TaskStat = { stage: string; startAt: number; endAt: number }
        type TaskDetail = {
            taskRun: string
            displayName: string
            startAt: number
            endAt: number
            duration: string
            waitFromPrev?: string
            status: string
        }
        const stats = new Map<string, StageStat>()
        const taskStats = new Map<string, TaskStat>()
        const taskStateByRun = new Map<string, { state: string; at: number }>()

        for (const entry of executionLogs) {
            const stage = inferStageFromTektonLog(entry)
            if (!stage) continue
            const ts = resolveExecutionLogTimestamp(entry)
            if (ts === undefined) continue

            const taskRun = String(entry.metadata?.task_run || entry.metadata?.taskRun || '').trim()
            const current = stats.get(stage)
            if (!current) {
                const initial: StageStat = {
                    startAt: ts,
                    endAt: ts,
                    taskRuns: new Set(taskRun ? [taskRun] : []),
                }
                stats.set(stage, initial)
            } else {
                current.startAt = Math.min(current.startAt, ts)
                current.endAt = Math.max(current.endAt, ts)
                if (taskRun) current.taskRuns.add(taskRun)
            }

            if (taskRun) {
                const taskCurrent = taskStats.get(taskRun)
                if (!taskCurrent) {
                    taskStats.set(taskRun, { stage, startAt: ts, endAt: ts })
                } else {
                    taskCurrent.startAt = Math.min(taskCurrent.startAt, ts)
                    taskCurrent.endAt = Math.max(taskCurrent.endAt, ts)
                }
            }

            if (taskRun) {
                const phase = String(entry.metadata?.phase || '').toLowerCase()
                const state = String(entry.metadata?.state || '').toLowerCase()
                if (phase === 'progress' && state) {
                    const previous = taskStateByRun.get(taskRun)
                    if (!previous || ts >= previous.at) {
                        taskStateByRun.set(taskRun, { state, at: ts })
                    }
                }
            }
        }

        const result: Record<string, { duration?: string; waitFromPrev?: string; tasks?: number; taskDurations?: string[]; taskDetails?: TaskDetail[] }> = {}
        for (let i = 0; i < buildStages.length; i += 1) {
            const stage = buildStages[i]
            const stat = stats.get(stage.key)
            if (!stat) continue

            const durationMs = Math.max(0, stat.endAt - stat.startAt)
            const previousStat = i > 0 ? stats.get(buildStages[i - 1].key) : undefined
            const waitMs = previousStat ? Math.max(0, stat.startAt - previousStat.endAt) : undefined
            const taskDetails = [...taskStats.entries()]
                .filter(([, task]) => task.stage === stage.key)
                .sort((a, b) => a[1].startAt - b[1].startAt)
                .map(([taskRun, task], taskIndex, all) => {
                    const previous = taskIndex > 0 ? all[taskIndex - 1][1] : undefined
                    const state = taskStateByRun.get(taskRun)?.state || 'completed'
                    return {
                        taskRun,
                        displayName: compactTaskRunName(taskRun),
                        startAt: task.startAt,
                        endAt: task.endAt,
                        duration: formatCompactDuration(Math.max(0, task.endAt - task.startAt)),
                        waitFromPrev: previous ? formatCompactDuration(Math.max(0, task.startAt - previous.endAt)) : undefined,
                        status: state,
                    }
                })
            const taskDurations = taskDetails.map((task) => `${task.displayName} (${task.duration})`)

            result[stage.key] = {
                duration: formatCompactDuration(durationMs),
                waitFromPrev: waitMs !== undefined ? formatCompactDuration(waitMs) : undefined,
                tasks: stat.taskRuns.size > 0 ? stat.taskRuns.size : undefined,
                taskDurations: taskDurations.length > 0 ? taskDurations : undefined,
                taskDetails: taskDetails.length > 0 ? taskDetails : undefined,
            }
        }
        return result
    })()

    const statusToStageIndex: Record<BuildStatus, number> = {
        queued: 0,
        pending: 1,
        running: 3,
        completed: buildStages.length - 1,
        failed: 3,
        cancelled: 3,
    }

    const workflowStepMap = new Map(
        (workflow?.steps ?? []).map(step => [step.stepKey, step])
    )
    const latestExecutionStage = (() => {
        let stage: string | undefined
        for (const entry of executionLogs) {
            const inferred = inferStageFromTektonLog(entry)
            if (inferred) stage = inferred
        }
        return stage
    })()
    const orderedWorkflowSteps = [...(workflow?.steps ?? [])].sort((a, b) => {
        const aIndex = WORKFLOW_STEP_ORDER.indexOf(a.stepKey)
        const bIndex = WORKFLOW_STEP_ORDER.indexOf(b.stepKey)
        const aRank = aIndex === -1 ? Number.MAX_SAFE_INTEGER : aIndex
        const bRank = bIndex === -1 ? Number.MAX_SAFE_INTEGER : bIndex
        if (aRank !== bRank) return aRank - bRank
        return String(a.stepKey ?? '').localeCompare(String(b.stepKey ?? ''))
    })
    const isBuildTerminal =
        build?.status === 'completed' ||
        build?.status === 'failed' ||
        build?.status === 'cancelled'
    const effectiveWorkflowStatus = (() => {
        if (build?.status === 'completed') return 'succeeded'
        if (build?.status === 'failed') return 'failed'
        if (build?.status === 'cancelled') return 'cancelled'
        return workflow?.status || 'unknown'
    })()
    const workflowDagTiming = (() => {
        const result: Record<string, { duration?: string; scheduleWait?: string }> = {}
        for (let i = 0; i < WORKFLOW_DAG_STEPS.length; i += 1) {
            const current = workflowStepMap.get(WORKFLOW_DAG_STEPS[i].key)
            if (!current) continue

            const startedMs = parseTimestampMs(current.startedAt) ?? parseTimestampMs(current.createdAt)
            const completedMs = parseTimestampMs(current.completedAt) ?? parseTimestampMs(current.updatedAt)
            const durationMs =
                startedMs !== undefined && completedMs !== undefined
                    ? Math.max(0, completedMs - startedMs)
                    : undefined

            let scheduleWaitMs: number | undefined
            if (i > 0 && startedMs !== undefined) {
                const previous = workflowStepMap.get(WORKFLOW_DAG_STEPS[i - 1].key)
                const previousCompletedMs = previous
                    ? (parseTimestampMs(previous.completedAt) ?? parseTimestampMs(previous.updatedAt))
                    : undefined
                if (previousCompletedMs !== undefined) {
                    scheduleWaitMs = Math.max(0, startedMs - previousCompletedMs)
                }
            }

            result[current.stepKey] = {
                duration: durationMs !== undefined ? formatCompactDuration(durationMs) : undefined,
                scheduleWait: scheduleWaitMs !== undefined ? formatCompactDuration(scheduleWaitMs) : undefined,
            }
        }
        return result
    })()
    const activeWorkflowStep =
        isBuildTerminal
            ? undefined
            : orderedWorkflowSteps.find(step => step.status === 'running') ||
              orderedWorkflowSteps.find(step => step.status === 'blocked') ||
              orderedWorkflowSteps.find(step => step.status === 'failed') ||
              orderedWorkflowSteps.find(step => step.status === 'pending')
    const blockedOrFailedWorkflowSteps = orderedWorkflowSteps.filter(
        step => step.status === 'blocked' || step.status === 'failed'
    )
    const hasControlPlaneIssue =
        build?.status === 'failed' || blockedOrFailedWorkflowSteps.length > 0
    const hasWorkflowDetails = Boolean(
        workflow && (
            (workflow.instanceId && workflow.instanceId.trim().length > 0) ||
            (workflow.status && workflow.status.trim().length > 0) ||
            (workflow.steps && workflow.steps.length > 0)
        )
    )
    const controlPlaneUnavailableMessage = (() => {
        if (traceDiagnostics?.repoConfig?.error) {
            const stageText = traceDiagnostics.repoConfig.stage ? ` during ${traceDiagnostics.repoConfig.stage}` : ''
            return `Build failed before workflow execution${stageText}: ${traceDiagnostics.repoConfig.error}`
        }
        if (build?.status === 'failed') {
            return build.errorMessage
                ? `Build failed before workflow execution: ${build.errorMessage}`
                : 'Build failed before workflow execution. No workflow instance was created.'
        }
        return 'Workflow details are not available for this build yet.'
    })()
    const latestWorkflowUpdateAt = orderedWorkflowSteps.reduce<string | undefined>((latest, step) => {
        if (!latest) return step.updatedAt
        return new Date(step.updatedAt) > new Date(latest) ? step.updatedAt : latest
    }, undefined)
    const pipelineStartedAt = (() => {
        const workflowStarts = orderedWorkflowSteps
            .map((step) => step.startedAt || step.createdAt)
            .filter((value): value is string => Boolean(value))
            .map((value) => ({ raw: value, ts: Date.parse(value) }))
            .filter((item) => !Number.isNaN(item.ts))
            .sort((a, b) => a.ts - b.ts)
        if (workflowStarts.length > 0) return workflowStarts[0].raw
        return selectedAttempt?.startedAt || build?.startedAt || build?.createdAt
    })()
    const pipelineEndedAt = (() => {
        const workflowEnds = orderedWorkflowSteps
            .map((step) => step.completedAt || step.updatedAt)
            .filter((value): value is string => Boolean(value))
            .map((value) => ({ raw: value, ts: Date.parse(value) }))
            .filter((item) => !Number.isNaN(item.ts))
            .sort((a, b) => b.ts - a.ts)
        if (workflowEnds.length > 0 && isBuildTerminal) return workflowEnds[0].raw
        return selectedAttempt?.completedAt || build?.completedAt
    })()
    const pipelineEndToEndDuration = formatDurationFromTimes(pipelineStartedAt, pipelineEndedAt)
    const isSystemAdmin = groups?.some((group: any) => group.role_type === 'system_administrator') ?? false
    const primaryRecoveryStep = blockedOrFailedWorkflowSteps[0]
    const showWorkflowDag =
        workflowDagView === 'shown' ||
        (workflowDagView === 'auto' && (isSystemAdmin || hasControlPlaneIssue))

    const handleStartDispatcherRecovery = async () => {
        try {
            setDispatcherRecoveryLoading(true)
            await adminService.startDispatcher()
            toast.success('Dispatcher start requested')
            await loadBuild()
        } catch (error: any) {
            toast.error(error.message || 'Failed to start dispatcher')
        } finally {
            setDispatcherRecoveryLoading(false)
        }
    }

    const handleStartOrchestratorRecovery = async () => {
        try {
            setOrchestratorRecoveryLoading(true)
            await adminService.startOrchestrator()
            toast.success('Orchestrator start requested')
            await loadBuild()
        } catch (error: any) {
            toast.error(error.message || 'Failed to start orchestrator')
        } finally {
            setOrchestratorRecoveryLoading(false)
        }
    }

    const workflowRecoveryActionsForStep = (stepKey: string): Array<{ key: string; label: string; onClick: () => void; disabled?: boolean }> => {
        if (stepKey === 'build.dispatch') {
            return [
                {
                    key: 'start-dispatcher',
                    label: dispatcherRecoveryLoading ? 'Starting Dispatcher…' : 'Start Dispatcher',
                    onClick: handleStartDispatcherRecovery,
                    disabled: !isSystemAdmin || dispatcherRecoveryLoading || orchestratorRecoveryLoading,
                },
                {
                    key: 'start-orchestrator',
                    label: orchestratorRecoveryLoading ? 'Starting Orchestrator…' : 'Start Orchestrator',
                    onClick: handleStartOrchestratorRecovery,
                    disabled: !isSystemAdmin || orchestratorRecoveryLoading || dispatcherRecoveryLoading,
                },
            ]
        }
        if (stepKey === 'build.monitor' || stepKey === 'build.finalize') {
            return [
                {
                    key: 'start-orchestrator',
                    label: orchestratorRecoveryLoading ? 'Starting Orchestrator…' : 'Start Orchestrator',
                    onClick: handleStartOrchestratorRecovery,
                    disabled: !isSystemAdmin || orchestratorRecoveryLoading || dispatcherRecoveryLoading,
                },
            ]
        }
        return []
    }

    const canShowRetryAction = build?.status === 'failed' || build?.status === 'cancelled'
    const sourceId = build?.manifest?.buildConfig?.sourceId
    const sourceName = sourceId ? (projectSources.find((source) => source.id === sourceId)?.name || sourceId.slice(0, 8)) : 'Not set'
    const refPolicy = build?.manifest?.buildConfig?.refPolicy || 'source_default'
    const refPolicyLabel =
        refPolicy === 'fixed' ? 'Fixed ref'
        : refPolicy === 'event_ref' ? 'Webhook event ref'
        : 'Source default'

    const handleExportTrace = async () => {
        if (!buildId) return
        try {
            await buildService.exportBuildTrace(buildId, selectedExecutionId)
            toast.success('Trace export started')
        } catch (error: any) {
            toast.error(error.message || 'Failed to export trace')
        }
    }

    const handleCopyTraceValue = async (label: string, value?: string) => {
        if (!value) return
        try {
            await navigator.clipboard.writeText(value)
            toast.success(`${label} copied`)
        } catch {
            toast.error(`Failed to copy ${label}`)
        }
    }

    const pushTraceEvent = (events: BuildTraceEvent[], event: BuildTraceEvent) => {
        if (!event.timestamp) return
        const parsed = Date.parse(event.timestamp)
        if (Number.isNaN(parsed)) return
        events.push(event)
    }

    const buildTraceEvents: BuildTraceEvent[] = (() => {
        if (!build) return []

        const events: BuildTraceEvent[] = []
        pushTraceEvent(events, {
            timestamp: build.createdAt,
            title: 'Build Created',
            detail: `Build ${build.manifest.name} was created.`,
            severity: 'info',
        })

        if (build.startedAt) {
            pushTraceEvent(events, {
                timestamp: build.startedAt,
                title: 'Build Started',
                detail: 'Build transitioned to running.',
                severity: 'info',
            })
        }

        if (build.completedAt) {
            pushTraceEvent(events, {
                timestamp: build.completedAt,
                title: 'Build Finished',
                detail: `Build finished with status: ${build.status}.`,
                severity: build.status === 'completed' ? 'success' : 'error',
            })
        }

        if (selectedAttempt) {
            pushTraceEvent(events, {
                timestamp: selectedAttempt.createdAt,
                title: 'Execution Attempt Created',
                detail: `Execution attempt ${selectedAttempt.id} created.`,
                severity: 'info',
            })
            if (selectedAttempt.startedAt) {
                pushTraceEvent(events, {
                    timestamp: selectedAttempt.startedAt,
                    title: 'Execution Attempt Started',
                    detail: `Attempt status: ${selectedAttempt.status}.`,
                    severity: 'info',
                })
            }
            if (selectedAttempt.completedAt) {
                pushTraceEvent(events, {
                    timestamp: selectedAttempt.completedAt,
                    title: 'Execution Attempt Completed',
                    detail: `Attempt finished with status: ${selectedAttempt.status}.`,
                    severity: selectedAttempt.status === 'success' ? 'success' : 'error',
                })
            }
            if (selectedAttempt.errorMessage) {
                pushTraceEvent(events, {
                    timestamp: selectedAttempt.completedAt || selectedAttempt.createdAt,
                    title: 'Execution Error',
                    detail: selectedAttempt.errorMessage,
                    severity: 'error',
                })
            }
        }

        orderedWorkflowSteps.forEach((step) => {
            const stepLabel = WORKFLOW_STEP_LABELS[step.stepKey] || step.stepKey
            if (step.startedAt) {
                pushTraceEvent(events, {
                    timestamp: step.startedAt,
                    title: `Workflow Step Started`,
                    detail: `${stepLabel} started.`,
                    severity: 'info',
                })
            }
            if (step.completedAt) {
                pushTraceEvent(events, {
                    timestamp: step.completedAt,
                    title: `Workflow Step ${step.status === 'succeeded' ? 'Succeeded' : 'Completed'}`,
                    detail: `${stepLabel} status: ${step.status}.`,
                    severity: step.status === 'succeeded' ? 'success' : step.status === 'failed' ? 'error' : 'warn',
                })
            }
            if (step.lastError) {
                pushTraceEvent(events, {
                    timestamp: step.updatedAt,
                    title: 'Workflow Step Error',
                    detail: `${stepLabel}: ${step.lastError}`,
                    severity: step.status === 'failed' ? 'error' : 'warn',
                })
            }
        })

        if (traceDiagnostics?.repoConfig) {
            const repoConfig = traceDiagnostics.repoConfig
            const detailSegments = [
                repoConfig.path ? `path=${repoConfig.path}` : undefined,
                repoConfig.ref ? `ref=${repoConfig.ref}` : undefined,
                repoConfig.stage ? `stage=${repoConfig.stage}` : undefined,
                repoConfig.errorCode ? `code=${repoConfig.errorCode}` : undefined,
                repoConfig.error || undefined,
            ].filter(Boolean)
            pushTraceEvent(events, {
                timestamp: repoConfig.updatedAt || build.updatedAt || build.createdAt,
                title: repoConfig.error ? 'Repository Build Config Error' : 'Repository Build Config Applied',
                detail: detailSegments.join(' | ') || (repoConfig.applied ? 'Repository build config applied.' : 'Repository build config diagnostics recorded.'),
                severity: repoConfig.error ? 'error' : repoConfig.applied ? 'success' : 'warn',
            })
        }

        if (isSystemAdmin && pipelineCheckedAt) {
            const dispatcher = pipelineComponents.dispatcher
            const orchestrator = pipelineComponents.workflow_orchestrator
            if (dispatcher) {
                pushTraceEvent(events, {
                    timestamp: dispatcher.last_activity || pipelineCheckedAt,
                    title: 'Dispatcher Snapshot',
                    detail: `running=${dispatcher.running} available=${dispatcher.available}${dispatcher.message ? ` (${dispatcher.message})` : ''}`,
                    severity: dispatcher.running && dispatcher.available ? 'success' : 'warn',
                })
            }
            if (orchestrator) {
                pushTraceEvent(events, {
                    timestamp: orchestrator.last_activity || pipelineCheckedAt,
                    title: 'Orchestrator Snapshot',
                    detail: `running=${orchestrator.running} available=${orchestrator.available}${orchestrator.message ? ` (${orchestrator.message})` : ''}`,
                    severity: orchestrator.running && orchestrator.available ? 'success' : 'warn',
                })
            }
        }

        return events.sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime())
    })()
    const latestTraceEvent = buildTraceEvents.length > 0 ? buildTraceEvents[buildTraceEvents.length - 1] : null

    const traceSeverityClasses = (severity: TraceSeverity) => {
        switch (severity) {
            case 'success':
                return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-900/20 dark:text-emerald-300'
            case 'warn':
                return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-300'
            case 'error':
                return 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300'
            default:
                return 'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-900/50 dark:bg-blue-900/20 dark:text-blue-300'
        }
    }

    const traceRowActions = (event: BuildTraceEvent) => {
        const detail = event.detail.toLowerCase()
        const actions: Array<{ key: string; label: string; onClick: () => void; disabled?: boolean }> = []

        const canRetryFromEvent =
            canShowRetryAction &&
            (event.title === 'Execution Error' || event.title === 'Build Finished' || event.title === 'Workflow Step Error')

        if (canRetryFromEvent) {
            actions.push({
                key: 'retry-build',
                label: 'Retry Build',
                onClick: handleRetryBuild,
            })
        }

        const isDispatchIssue = detail.includes('dispatch')
        const isOrchestratorIssue = detail.includes('dispatch') || detail.includes('monitor') || detail.includes('finalize') || event.title === 'Workflow Step Error'

        if (isOrchestratorIssue) {
            actions.push({
                key: 'start-orchestrator',
                label: orchestratorRecoveryLoading ? 'Starting Orchestrator…' : 'Start Orchestrator',
                onClick: handleStartOrchestratorRecovery,
                disabled: !isSystemAdmin || orchestratorRecoveryLoading || dispatcherRecoveryLoading,
            })
        }

        if (isDispatchIssue) {
            actions.push({
                key: 'start-dispatcher',
                label: dispatcherRecoveryLoading ? 'Starting Dispatcher…' : 'Start Dispatcher',
                onClick: handleStartDispatcherRecovery,
                disabled: !isSystemAdmin || dispatcherRecoveryLoading || orchestratorRecoveryLoading,
            })
        }

        return actions
    }

    const mapWorkflowStatus = (status: WorkflowStepStatus) => {
        switch (status) {
            case 'succeeded':
                return 'done'
            case 'running':
                return 'active'
            case 'failed':
                return 'failed'
            case 'blocked':
                return 'blocked'
            default:
                return 'upcoming'
        }
    }

    const getWorkflowTaskState = (stepKey: string): string => {
        const workflowStep = workflowStepMap.get(stepKey)
        if (workflowStep) return mapWorkflowStatus(workflowStep.status)
        if (effectiveWorkflowStatus === 'succeeded') return 'done'
        return 'upcoming'
    }

    const getStageState = (index: number, stepKey: string) => {
        const workflowStep = workflowStepMap.get(stepKey)
        if (workflowStep) {
            return mapWorkflowStatus(workflowStep.status)
        }

        const buildStatus: BuildStatus = build?.status ?? 'pending'
        const inferredIndex = latestExecutionStage ? EXECUTION_STAGE_ORDER[latestExecutionStage] : undefined
        const activeIndex =
            (buildStatus === 'failed' || buildStatus === 'cancelled') && inferredIndex !== undefined
                ? inferredIndex
                : (statusToStageIndex[buildStatus] ?? 0)

        if (buildStatus === 'completed') return 'done'
        if (buildStatus === 'failed') {
            if (index < activeIndex) return 'done'
            if (index === activeIndex) return 'failed'
            return 'upcoming'
        }
        if (buildStatus === 'cancelled') {
            if (index < activeIndex) return 'done'
            if (index === activeIndex) return 'cancelled'
            return 'upcoming'
        }
        if (buildStatus === 'running') {
            if (index < activeIndex) return 'done'
            if (index === activeIndex) return 'active'
            return 'upcoming'
        }
        if (index < activeIndex) return 'done'
        if (index === activeIndex) return 'active'
        return 'upcoming'
    }
    const executionStatusLabel = (() => {
        const status = selectedAttempt?.status || build?.status || 'unknown'
        switch (status) {
            case 'success':
            case 'completed':
                return 'succeeded'
            default:
                return String(status)
        }
    })()
    const nextExecutionStage = (() => {
        const stageStates = buildStages.map((stage, index) => ({
            stage,
            state: getStageState(index, stage.key),
        }))
        const active = stageStates.find((entry) => entry.state === 'active' || entry.state === 'blocked' || entry.state === 'failed')
        if (active) return active.stage
        const upcoming = stageStates.find((entry) => entry.state === 'upcoming')
        if (upcoming) return upcoming.stage
        return undefined
    })()
    const latestExecutionWaitHint = (() => {
        for (let i = executionLogs.length - 1; i >= 0; i -= 1) {
            const entry = executionLogs[i]
            const phase = String(entry.metadata?.phase || '').toLowerCase()
            if (phase === 'pipeline_waiting' && entry.message) {
                return entry.message
            }
        }
        return undefined
    })()
    const executionNextStepLabel = nextExecutionStage
        ? `${nextExecutionStage.label}: ${nextExecutionStage.description}`
        : (isBuildTerminal ? 'None (execution complete)' : 'Determining next step...')

    const getStageIcon = (state: string) => {
        switch (state) {
            case 'done':
                return <Check className="h-4 w-4" />
            case 'active':
                return <Clock className="h-4 w-4" />
            case 'failed':
                return <X className="h-4 w-4" />
            case 'cancelled':
                return <Ban className="h-4 w-4" />
            case 'blocked':
                return <XCircle className="h-4 w-4" />
            default:
                return <Circle className="h-3 w-3" />
        }
    }

    const getStageClasses = (state: string) => {
        switch (state) {
            case 'done':
                return 'bg-emerald-500 text-white'
            case 'active':
                return 'bg-blue-600 text-white ring-2 ring-blue-200 dark:ring-blue-900'
            case 'failed':
                return 'bg-red-600 text-white'
            case 'cancelled':
                return 'bg-amber-500 text-white'
            case 'blocked':
                return 'bg-slate-500 text-white'
            default:
                return 'bg-slate-200 text-slate-500 dark:bg-slate-700 dark:text-slate-300'
        }
    }

    const resolveDockerfileContent = () => {
        const dockerfile = build?.manifest.buildConfig?.dockerfile
        if (!dockerfile) return ''
        if (typeof dockerfile === 'string') return dockerfile
        if (typeof dockerfile === 'object') {
            return dockerfile.content || dockerfile.path || ''
        }
        return ''
    }

    const handleCopyDockerfile = async () => {
        const content = resolveDockerfileContent()
        if (!content) return
        try {
            await navigator.clipboard.writeText(content)
            toast.success('Dockerfile copied to clipboard')
        } catch (error) {
            toast.error('Failed to copy Dockerfile')
        }
    }

    const dockerfileContent = resolveDockerfileContent()
    const resolveDestinationImage = (): string | undefined => {
        const imageId = build?.result?.imageId?.trim()
        if (imageId) return imageId

        const artifactRef = build?.result?.artifacts?.find((artifact) => {
            const value = (artifact || '').trim()
            if (!value || value.startsWith('sha256:')) return false
            return value.includes('/') && (value.includes(':') || value.includes('@sha256:'))
        })
        if (artifactRef) return artifactRef.trim()

        const repo = build?.manifest?.buildConfig?.registryRepo?.trim()
        if (!repo) return undefined
        if (repo.includes('@sha256:')) return repo

        const lastSlash = repo.lastIndexOf('/')
        const lastColon = repo.lastIndexOf(':')
        const hasTag = lastColon > lastSlash
        if (hasTag) return repo

        const firstTag = build?.manifest?.tags?.find((tag) => (tag || '').trim().length > 0)?.trim()
        return firstTag ? `${repo}:${firstTag}` : repo
    }
    const destinationImage = resolveDestinationImage()
    const dockerfileUnavailableReason = (() => {
        if (dockerfileContent) return ''
        const repoConfig = traceDiagnostics?.repoConfig
        if (repoConfig?.applied || sourceId) {
            const pathHint = repoConfig?.path ? ` (path: ${repoConfig.path})` : ''
            const refHint = repoConfig?.ref ? ` at ref ${repoConfig.ref}` : ''
            return `Dockerfile content is not stored in this build record when configuration comes from the repository${pathHint}${refHint}.`
        }
        return 'Dockerfile content is not available for this build record.'
    })()

    if (loading) {
        return (
            <div className="px-4 py-6 sm:px-6 lg:px-8">
                <div className="text-center py-8">
                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500 mx-auto"></div>
                    <p className="mt-2 text-sm text-muted-foreground">Loading build details...</p>
                </div>
            </div>
        )
    }

    if (!build) {
        return (
            <div className="px-4 py-6 sm:px-6 lg:px-8">
                <div className="text-center py-8">
                    <p className="text-muted-foreground">Build not found.</p>
                    <Link to="/builds" className="btn btn-primary mt-4">
                        Back to Builds
                    </Link>
                </div>
            </div>
        )
    }

    return (
        <div className="min-h-screen bg-slate-50 dark:bg-slate-900 px-4 py-6 sm:px-6 lg:px-8 space-y-6">
            <ConfirmDialog
                isOpen={showDeleteConfirm}
                title="Delete Build"
                message="Delete this build and all related execution data? This action cannot be undone."
                confirmLabel="Delete Build"
                destructive
                onConfirm={confirmDeleteBuild}
                onCancel={() => setShowDeleteConfirm(false)}
            />
            {/* Header */}
            <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-800 rounded-lg p-5">
                <div className="sm:flex sm:items-center sm:justify-between">
                    <div>
                        <nav className="flex text-sm" aria-label="Breadcrumb">
                            <ol className="flex items-center space-x-2 text-gray-500 dark:text-gray-400">
                                <li>
                                    <Link to="/builds" className="hover:text-gray-700 dark:hover:text-gray-200">
                                        Builds
                                    </Link>
                                </li>
                                <li>
                                    <span className="text-gray-400">/</span>
                                </li>
                                <li>
                                    <span className="text-gray-900 dark:text-white">{build.manifest.name}</span>
                                </li>
                            </ol>
                        </nav>
                        <div className="mt-2 flex items-center gap-3">
                            <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">
                                {build.manifest.name}
                            </h1>
                            <span className={`inline-flex px-2.5 py-1 text-xs font-semibold rounded-full ${getStatusColor(build.status)}`}>
                                {build.status.charAt(0).toUpperCase() + build.status.slice(1)}
                            </span>
                            <span className="inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold bg-slate-100 text-slate-700 dark:bg-slate-700 dark:text-slate-100">
                                <FileText className="mr-1 h-3 w-3" />
                                Build Method: {build.manifest.type.charAt(0).toUpperCase() + build.manifest.type.slice(1)}
                            </span>
                            {selectedAttempt && (
                                <span className="inline-flex px-2.5 py-1 text-xs font-semibold rounded-full bg-slate-100 text-slate-700 dark:bg-slate-700 dark:text-slate-100">
                                    Attempt Status: {selectedAttempt.status}
                                </span>
                            )}
                        </div>
                        <p className="mt-2 text-xs text-gray-500 dark:text-gray-400 font-mono">
                            Build ID: {build.id}
                        </p>
                    </div>
                    <div className="mt-4 sm:mt-0 flex items-center gap-3">
                        <span className={`inline-flex items-center rounded-full px-2 py-1 text-[11px] font-medium ${isWsConnected
                            ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
                            : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
                            }`}>
                            {isWsConnected ? 'Live updates on' : 'Live updates off'}
                        </span>
                        <button
                            onClick={handleDeleteBuild}
                            disabled={!canDeleteBuild(build.status)}
                            className="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs bg-red-600 text-white rounded-lg hover:bg-red-700 transition disabled:bg-slate-400 disabled:cursor-not-allowed dark:disabled:bg-slate-700"
                            title={canDeleteBuild(build.status) ? 'Delete build and related execution records' : 'Cannot delete running or queued build'}
                        >
                            <Trash2 className="h-3.5 w-3.5" />
                            Delete Build
                        </button>
                        {build.status === 'pending' && (
                            <button
                                onClick={handleStartBuild}
                                className="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition dark:bg-blue-500 dark:hover:bg-blue-400"
                            >
                                <Play className="h-3.5 w-3.5" />
                                Start Build
                            </button>
                        )}
                        {build.status === 'running' && (
                            <button
                                onClick={handleCancelBuild}
                                className="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs bg-red-600 text-white rounded-lg hover:bg-red-700 transition dark:bg-red-500 dark:hover:bg-red-400"
                            >
                                <XCircle className="h-3.5 w-3.5" />
                                Cancel Build
                            </button>
                        )}
                        {(build.status === 'failed' || build.status === 'cancelled') && (
                            <button
                                onClick={handleRetryBuild}
                                className="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs bg-amber-600 text-white rounded-lg hover:bg-amber-700 transition dark:bg-amber-500 dark:hover:bg-amber-400"
                                title="Creates a new execution attempt under this same build"
                            >
                                <Play className="h-3.5 w-3.5" />
                                Retry (New Attempt)
                            </button>
                        )}
                        {canCreateBuild ? (
                            <button
                                onClick={handleCloneBuild}
                                className="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition dark:bg-indigo-500 dark:hover:bg-indigo-400"
                                title="Create a new build using this build's configuration"
                            >
                                <Copy className="h-3.5 w-3.5" />
                                Clone Build
                            </button>
                        ) : null}
                        {canCreateBuild ? (
                            <Link
                                to="/builds/new"
                                className="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition dark:bg-blue-500 dark:hover:bg-blue-400"
                            >
                                <Plus className="h-3.5 w-3.5" />
                                New Build
                            </Link>
                        ) : null}
                        <button
                            onClick={() => void loadBuild()}
                            className="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-medium bg-blue-600 text-white border border-transparent rounded-lg hover:bg-blue-700 transition dark:bg-blue-500 dark:hover:bg-blue-400"
                            title="Refresh overall build status, logs, and attempts"
                        >
                            <RefreshCcw className="h-3.5 w-3.5" />
                            Refresh Status
                        </button>
                    </div>
                </div>
            </div>

            {/* Summary */}
            <div className="card bg-white dark:bg-gray-800">
                <div className="card-body pt-5">
                    <p className="inline-flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
                        <Clock className="h-3.5 w-3.5" />
                        Timing
                    </p>
                    <div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
                        <div className="rounded-md border border-gray-200 bg-gray-50 p-2.5 dark:border-gray-700 dark:bg-gray-900/40">
                            <p className="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Duration (Selected Attempt)</p>
                            <p className="text-xs font-medium text-gray-900 dark:text-white">
                                {formatDurationFromTimes(selectedAttempt?.startedAt || build.startedAt, selectedAttempt?.completedAt || build.completedAt)}
                            </p>
                        </div>
                        <div className="rounded-md border border-gray-200 bg-gray-50 p-2.5 dark:border-gray-700 dark:bg-gray-900/40">
                            <p className="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Created</p>
                            <p className="text-xs font-medium text-gray-900 dark:text-white">
                                {formatDate(build.createdAt)}
                            </p>
                        </div>
                        <div className="rounded-md border border-gray-200 bg-gray-50 p-2.5 dark:border-gray-700 dark:bg-gray-900/40">
                            <p className="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Pipeline Start</p>
                            <p className="text-xs font-medium text-gray-900 dark:text-white">
                                {pipelineStartedAt ? formatDate(pipelineStartedAt) : '-'}
                            </p>
                        </div>
                        <div className="rounded-md border border-gray-200 bg-gray-50 p-2.5 dark:border-gray-700 dark:bg-gray-900/40">
                            <p className="text-[11px] uppercase tracking-wide text-gray-500 dark:text-gray-400">Pipeline End</p>
                            <p className="text-xs font-medium text-gray-900 dark:text-white">
                                {pipelineEndedAt ? formatDate(pipelineEndedAt) : (isBuildActive ? 'In progress' : '-')}
                            </p>
                        </div>
                    </div>
                </div>
            </div>

            <div className="card bg-white dark:bg-gray-800">
                <div className="card-header">
                    <div className="flex items-center justify-between">
                        <h3 className="text-lg font-medium text-gray-900 dark:text-white">Build Pipeline</h3>
                        <div className="flex items-center gap-3">
                            <span className="text-xs text-gray-500 dark:text-gray-400">DAG stages</span>
                            <select
                                value={selectedExecutionId || ''}
                                onChange={(e) => setSelectedExecutionId(e.target.value || undefined)}
                                className="text-xs rounded-md border border-slate-300 bg-white px-2 py-1 text-slate-700 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
                            >
                                {attempts.length === 0 && <option value="">No attempts</option>}
                                {attempts.map((attempt, index) => (
                                    <option key={attempt.id} value={attempt.id}>
                                        Attempt #{attempts.length - index} - {attempt.status}
                                    </option>
                                ))}
                            </select>
                        </div>
                    </div>
                </div>
                <div className="card-body space-y-4">
                    <div className="rounded-lg border border-slate-200 bg-slate-50/80 p-2.5 dark:border-slate-700 dark:bg-slate-900/40">
                        <div className="flex items-center justify-between">
                            <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-300">
                                Control Plane Details
                            </p>
                            <button
                                type="button"
                                onClick={() => {
                                    const currentlyVisible =
                                        workflowDagView === 'shown' ||
                                        (workflowDagView === 'auto' && (isSystemAdmin || hasControlPlaneIssue))
                                    setWorkflowDagView(currentlyVisible ? 'hidden' : 'shown')
                                }}
                                className="inline-flex items-center gap-1 rounded-md border border-slate-300 bg-white px-2 py-1 text-[11px] font-medium text-slate-700 hover:bg-slate-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
                            >
                                {showWorkflowDag ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
                                {showWorkflowDag ? 'Hide' : 'Show'}
                            </button>
                        </div>
                        {showWorkflowDag && (
                            <div className="mt-3 rounded-lg border border-blue-200 bg-blue-50/70 p-3 dark:border-blue-900/60 dark:bg-blue-950/30">
                                <div className="mb-3 flex items-center justify-between">
                                    <p className="text-xs font-semibold uppercase tracking-wide text-blue-800 dark:text-blue-300">Workflow Tasks</p>
                                    <p className="text-[11px] text-blue-700 dark:text-blue-300">Control Plane Scheduling</p>
                                </div>
                                <div className="hidden lg:block">
                                    <div className="relative">
                                        <div className="absolute left-4 right-4 top-4 h-px bg-blue-200 dark:bg-blue-900/60" />
                                        <ol className="relative z-10 grid grid-cols-6 gap-3">
                                            {WORKFLOW_DAG_STEPS.map((step, index) => {
                                                const state = getWorkflowTaskState(step.key)
                                                return (
                                                    <li key={step.key} className="flex flex-col items-center text-center gap-2">
                                                        <div className={`flex h-8 w-8 items-center justify-center rounded-full ${getStageClasses(state)}`}>
                                                            {getStageIcon(state)}
                                                        </div>
                                                        <div>
                                                            <p className="text-xs font-semibold text-slate-900 dark:text-white">{step.label}</p>
                                                            <p className="text-[11px] text-slate-600 dark:text-slate-300">{step.description}</p>
                                                            {workflowDagTiming[step.key]?.duration && (
                                                                <p className="text-[10px] text-slate-600 dark:text-slate-300">
                                                                    Task time: {workflowDagTiming[step.key]?.duration}
                                                                </p>
                                                            )}
                                                            {workflowDagTiming[step.key]?.scheduleWait && index > 0 && (
                                                                <p className="text-[10px] text-amber-700 dark:text-amber-300">
                                                                    Schedule wait: {workflowDagTiming[step.key]?.scheduleWait}
                                                                </p>
                                                            )}
                                                        </div>
                                                    </li>
                                                )
                                            })}
                                        </ol>
                                    </div>
                                </div>
                                <div className="space-y-3 lg:hidden">
                                    {WORKFLOW_DAG_STEPS.map((step, index) => {
                                        const state = getWorkflowTaskState(step.key)
                                        return (
                                            <div key={step.key} className="flex items-start gap-3">
                                                <div className={`flex h-7 w-7 items-center justify-center rounded-full ${getStageClasses(state)}`}>
                                                    {getStageIcon(state)}
                                                </div>
                                                <div>
                                                    <p className="text-sm font-medium text-slate-900 dark:text-white">{step.label}</p>
                                                    <p className="text-xs text-slate-500 dark:text-slate-400">{step.description}</p>
                                                    {workflowDagTiming[step.key]?.duration && (
                                                        <p className="text-[11px] text-slate-600 dark:text-slate-300">
                                                            Task time: {workflowDagTiming[step.key]?.duration}
                                                        </p>
                                                    )}
                                                    {workflowDagTiming[step.key]?.scheduleWait && index > 0 && (
                                                        <p className="text-[11px] text-amber-700 dark:text-amber-300">
                                                            Schedule wait: {workflowDagTiming[step.key]?.scheduleWait}
                                                        </p>
                                                    )}
                                                </div>
                                            </div>
                                        )
                                    })}
                                </div>
                            </div>
                        )}
                    </div>

                    <div className="rounded-lg border border-emerald-200 bg-emerald-50/70 p-3 dark:border-emerald-900/60 dark:bg-emerald-950/25">
                        <div className="mb-3 flex items-center justify-between">
                            <p className="text-xs font-semibold uppercase tracking-wide text-emerald-800 dark:text-emerald-300">Execution Tasks</p>
                            <p className="text-[11px] text-emerald-700 dark:text-emerald-300">Runtime Pipeline</p>
                        </div>
                        <div className="hidden lg:block">
                            <div className="relative">
                                <div className="absolute left-4 right-4 top-4 h-px bg-emerald-200 dark:bg-emerald-900/60" />
                                <ol className="relative z-10 grid grid-cols-8 gap-3">
                                    {buildStages.map((stage, index) => {
                                        const state = getStageState(index, stage.key)
                                        return (
                                            <li key={stage.key} className="flex flex-col items-center text-center gap-2">
                                                <div className={`flex h-8 w-8 items-center justify-center rounded-full ${getStageClasses(state)}`}>
                                                    {getStageIcon(state)}
                                                </div>
                                                <div>
                                                    <p className="text-xs font-semibold text-slate-900 dark:text-white">{stage.label}</p>
                                                    <p className="text-[11px] text-slate-600 dark:text-slate-300">{stage.description}</p>
                                                    {stageTiming[stage.key]?.duration && (
                                                        <p className="text-[10px] text-slate-600 dark:text-slate-300">
                                                            Task time: {stageTiming[stage.key]?.duration}
                                                        </p>
                                                    )}
                                                    {stageTiming[stage.key]?.waitFromPrev && index > 0 && (
                                                        <p className="text-[10px] text-amber-700 dark:text-amber-300">
                                                            Wait from prev: {stageTiming[stage.key]?.waitFromPrev}
                                                        </p>
                                                    )}
                                                    {stageTiming[stage.key]?.tasks && (
                                                        <p className="text-[10px] text-slate-500 dark:text-slate-400">
                                                            Task runs: {stageTiming[stage.key]?.tasks}
                                                        </p>
                                                    )}
                                                    {(stageTiming[stage.key]?.taskDetails?.length || 0) > 0 && (
                                                        <button
                                                            type="button"
                                                            onClick={() => toggleExecutionStageDetails(stage.key)}
                                                            className="mt-1 inline-flex items-center gap-1 rounded border border-slate-300 px-1.5 py-0.5 text-[10px] text-slate-600 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
                                                        >
                                                            {expandedExecutionStages[stage.key] ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
                                                            Details ({stageTiming[stage.key]?.taskDetails?.length})
                                                        </button>
                                                    )}
                                                    {expandedExecutionStages[stage.key] && stageTiming[stage.key]?.taskDetails && (
                                                        <div className="mt-2 space-y-1 rounded border border-slate-200 bg-slate-50 p-1.5 text-left dark:border-slate-700 dark:bg-slate-900/40">
                                                            {stageTiming[stage.key]?.taskDetails?.map((task) => (
                                                                <div key={task.taskRun} className="border-b border-slate-200 pb-1 last:border-b-0 dark:border-slate-700">
                                                                    <p className="text-[10px] font-semibold text-slate-700 dark:text-slate-200">{task.displayName}</p>
                                                                    <p className="text-[10px] text-slate-500 dark:text-slate-400">
                                                                        {formatClockTime(task.startAt)} {' -> '} {formatClockTime(task.endAt)} | {task.duration}
                                                                    </p>
                                                                    {task.waitFromPrev && (
                                                                        <p className="text-[10px] text-amber-700 dark:text-amber-300">Gap from previous: {task.waitFromPrev}</p>
                                                                    )}
                                                                    <p className="text-[10px] text-slate-500 dark:text-slate-400">Status: {task.status}</p>
                                                                </div>
                                                            ))}
                                                        </div>
                                                    )}
                                                </div>
                                            </li>
                                        )
                                    })}
                                </ol>
                            </div>
                        </div>
                        <div className="space-y-3 lg:hidden">
                            {buildStages.map((stage, index) => {
                                const state = getStageState(index, stage.key)
                                return (
                                    <div key={stage.key} className="flex items-start gap-3">
                                        <div className={`flex h-7 w-7 items-center justify-center rounded-full ${getStageClasses(state)}`}>
                                            {getStageIcon(state)}
                                        </div>
                                        <div>
                                            <p className="text-sm font-medium text-slate-900 dark:text-white">{stage.label}</p>
                                            <p className="text-xs text-slate-500 dark:text-slate-400">{stage.description}</p>
                                            {stageTiming[stage.key]?.duration && (
                                                <p className="text-[11px] text-slate-600 dark:text-slate-300">
                                                    Task time: {stageTiming[stage.key]?.duration}
                                                </p>
                                            )}
                                            {stageTiming[stage.key]?.waitFromPrev && index > 0 && (
                                                <p className="text-[11px] text-amber-700 dark:text-amber-300">
                                                    Wait from prev: {stageTiming[stage.key]?.waitFromPrev}
                                                </p>
                                            )}
                                        {stageTiming[stage.key]?.tasks && (
                                            <p className="text-[11px] text-slate-500 dark:text-slate-400">
                                                Task runs: {stageTiming[stage.key]?.tasks}
                                            </p>
                                        )}
                                        {(stageTiming[stage.key]?.taskDetails?.length || 0) > 0 && (
                                            <button
                                                type="button"
                                                onClick={() => toggleExecutionStageDetails(stage.key)}
                                                className="mt-1 inline-flex items-center gap-1 rounded border border-slate-300 px-2 py-0.5 text-[11px] text-slate-600 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
                                            >
                                                {expandedExecutionStages[stage.key] ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
                                                Details ({stageTiming[stage.key]?.taskDetails?.length})
                                            </button>
                                        )}
                                        {expandedExecutionStages[stage.key] && stageTiming[stage.key]?.taskDetails && (
                                            <div className="mt-2 space-y-1 rounded border border-slate-200 bg-slate-50 p-2 dark:border-slate-700 dark:bg-slate-900/40">
                                                {stageTiming[stage.key]?.taskDetails?.map((task) => (
                                                    <div key={task.taskRun} className="border-b border-slate-200 pb-1 last:border-b-0 dark:border-slate-700">
                                                        <p className="text-[11px] font-semibold text-slate-700 dark:text-slate-200">{task.displayName}</p>
                                                        <p className="text-[11px] text-slate-500 dark:text-slate-400">
                                                            {formatClockTime(task.startAt)} {' -> '} {formatClockTime(task.endAt)} | {task.duration}
                                                        </p>
                                                        {task.waitFromPrev && (
                                                            <p className="text-[11px] text-amber-700 dark:text-amber-300">Gap from previous: {task.waitFromPrev}</p>
                                                        )}
                                                        <p className="text-[11px] text-slate-500 dark:text-slate-400">Status: {task.status}</p>
                                                    </div>
                                                ))}
                                            </div>
                                        )}
                                    </div>
                                </div>
                            )
                            })}
                        </div>
                    </div>
                </div>
            </div>

            <div className="card bg-white dark:bg-gray-800">
                <div className="card-header">
                    <div className="flex items-center justify-between gap-4">
                        <h3 className="text-lg font-medium text-gray-900 dark:text-white">Build Execution Trace</h3>
                        <div className="flex items-center gap-3">
                            {traceCorrelation?.activeStepKey && (
                                <span className="text-xs rounded-md border border-slate-300 px-2 py-1 text-slate-600 dark:border-slate-600 dark:text-slate-300">
                                    Active: {traceCorrelation.activeStepKey}
                                </span>
                            )}
                            {traceCorrelation?.workflowInstanceId && (
                                <button
                                    type="button"
                                    onClick={() => handleCopyTraceValue('workflow instance ID', traceCorrelation.workflowInstanceId)}
                                    className="text-xs rounded-md border border-slate-300 px-2 py-1 font-mono text-slate-600 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
                                    title={traceCorrelation.workflowInstanceId}
                                >
                                    WF: {traceCorrelation.workflowInstanceId.slice(0, 8)}
                                </button>
                            )}
                            {traceCorrelation?.executionId && (
                                <button
                                    type="button"
                                    onClick={() => handleCopyTraceValue('execution ID', traceCorrelation.executionId)}
                                    className="text-xs rounded-md border border-slate-300 px-2 py-1 font-mono text-slate-600 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
                                    title={traceCorrelation.executionId}
                                >
                                    EX: {traceCorrelation.executionId.slice(0, 8)}
                                </button>
                            )}
                            <span className="text-xs text-slate-500 dark:text-slate-400">
                                {buildTraceEvents.length} events
                            </span>
                            <div className="relative group">
                                <button
                                    type="button"
                                    onClick={() => setTraceExpanded((current) => !current)}
                                    className="inline-flex items-center gap-1 text-xs rounded-md border border-slate-300 px-2 py-1 font-medium text-slate-600 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
                                    title={traceExpanded ? 'Collapse timeline details' : 'Expand to see timeline details'}
                                >
                                    {traceExpanded ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
                                    {traceExpanded ? 'Hide Timeline' : 'Show Timeline'}
                                </button>
                                <div className="pointer-events-none absolute -bottom-9 left-1/2 z-20 hidden -translate-x-1/2 whitespace-nowrap rounded bg-slate-900 px-2 py-1 text-[11px] text-slate-100 shadow-sm group-hover:block dark:bg-slate-100 dark:text-slate-900">
                                    {traceExpanded ? 'Collapse to save space' : 'Expand to see timeline details'}
                                </div>
                            </div>
                            <button
                                type="button"
                                onClick={handleExportTrace}
                                className="text-xs rounded-md border border-slate-300 px-2 py-1 font-medium text-slate-600 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
                            >
                                Export Trace
                            </button>
                            {isSystemAdmin && (
                                <Link
                                    to="/admin"
                                    className="text-xs rounded-md border border-slate-300 px-2 py-1 font-medium text-slate-600 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
                                >
                                    Open Pipeline Health
                                </Link>
                            )}
                        </div>
                    </div>
                </div>
                <div className="card-body">
                    {!traceExpanded && (
                        <p className="mb-3 text-xs text-slate-500 dark:text-slate-400">
                            Expand timeline to see the full execution trace.
                        </p>
                    )}
                    {!traceExpanded ? (
                        <div className="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700 dark:border-slate-700 dark:bg-slate-900/30 dark:text-slate-300">
                            {latestTraceEvent ? (
                                <div className="flex flex-wrap items-center justify-between gap-2">
                                    <span>
                                        Latest: <span className="font-medium">{latestTraceEvent.title}</span>
                                    </span>
                                    <span className="text-xs text-slate-500 dark:text-slate-400">{formatDate(latestTraceEvent.timestamp)}</span>
                                </div>
                            ) : (
                                <span className="text-slate-500 dark:text-slate-400">No trace events available yet.</span>
                            )}
                        </div>
                    ) : buildTraceEvents.length === 0 ? (
                        <p className="text-sm text-slate-500 dark:text-slate-400">No trace events available yet.</p>
                    ) : (
                        <ol className="space-y-3">
                            {buildTraceEvents.map((event, index) => {
                                const actions = traceRowActions(event)
                                return (
                                    <li
                                        key={`${event.timestamp}-${event.title}-${index}`}
                                        className={`rounded-md border px-3 py-2 ${traceSeverityClasses(event.severity)}`}
                                    >
                                        <div className="flex items-start justify-between gap-4">
                                            <div>
                                                <p className="text-sm font-semibold">{event.title}</p>
                                                <p className="mt-1 text-xs opacity-90">{event.detail}</p>
                                                {actions.length > 0 && (
                                                    <div className="mt-2 flex flex-wrap gap-2">
                                                        {actions.map((action) => (
                                                            <button
                                                                key={action.key}
                                                                type="button"
                                                                onClick={action.onClick}
                                                                disabled={action.disabled}
                                                                className="rounded-md border border-current/30 px-2 py-1 text-[11px] font-medium hover:bg-white/40 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-slate-900/40"
                                                            >
                                                                {action.label}
                                                            </button>
                                                        ))}
                                                    </div>
                                                )}
                                            </div>
                                            <span className="whitespace-nowrap text-[11px] opacity-80">
                                                {formatDate(event.timestamp)}
                                            </span>
                                        </div>
                                    </li>
                                )
                            })}
                        </ol>
                    )}
                </div>
            </div>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
                <div className="lg:col-span-2 space-y-6">
                    {/* Build Configuration */}
                    <div className="card bg-white dark:bg-gray-800">
                        <div className="card-header">
                            <div className="flex items-center justify-between">
                                <h3 className="text-lg font-medium text-gray-900 dark:text-white">Build Configuration</h3>
                                <button
                                    onClick={() => setShowDockerfileDrawer(true)}
                                    disabled={!dockerfileContent}
                                    title={dockerfileContent ? 'View Dockerfile content' : dockerfileUnavailableReason}
                                    className="inline-flex items-center gap-2 text-sm text-blue-600 hover:text-blue-700 disabled:text-gray-400 disabled:cursor-not-allowed dark:disabled:text-gray-500"
                                >
                                    <FileText className="h-4 w-4" />
                                    {dockerfileContent ? 'View Dockerfile' : 'Dockerfile Unavailable'}
                                </button>
                            </div>
                        </div>
                        <div className="card-body">
                            {!dockerfileContent && (
                                <div className="mb-4 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-200">
                                    {dockerfileUnavailableReason}
                                </div>
                            )}
                            <dl className="grid grid-cols-1 gap-x-4 gap-y-6 sm:grid-cols-2 lg:grid-cols-3">
                                <div>
                                    <dt className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Base Image</dt>
                                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">{build.manifest.baseImage || 'Not specified'}</dd>
                                </div>
                                <div>
                                    <dt className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Build Context</dt>
                                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">{build.manifest.buildConfig?.buildContext || 'Not specified'}</dd>
                                </div>
                                <div>
                                    <dt className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Source</dt>
                                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">{sourceName}</dd>
                                </div>
                                <div>
                                    <dt className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Ref Policy</dt>
                                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">{refPolicyLabel}</dd>
                                </div>
                                {refPolicy === 'fixed' && (
                                    <div>
                                        <dt className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Fixed Ref</dt>
                                        <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                                            {build.manifest.buildConfig?.fixedRef || 'Not specified'}
                                        </dd>
                                    </div>
                                )}
                                <div>
                                    <dt className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Registry</dt>
                                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                                        {build.manifest.buildConfig?.registryType || 'Not specified'}
                                    </dd>
                                </div>
                                <div>
                                    <dt className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">SBOM Tool</dt>
                                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                                        {build.manifest.buildConfig?.sbomTool || 'Not specified'}
                                    </dd>
                                </div>
                                <div>
                                    <dt className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Security Scanner</dt>
                                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                                        {build.manifest.buildConfig?.scanTool || 'Not specified'}
                                    </dd>
                                </div>
                                <div>
                                    <dt className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Secret Manager</dt>
                                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                                        {build.manifest.buildConfig?.secretManagerType || 'Not specified'}
                                    </dd>
                                </div>
                            </dl>
                        </div>
                    </div>
                    <Drawer
                        isOpen={showDockerfileDrawer}
                        onClose={() => setShowDockerfileDrawer(false)}
                        title="Dockerfile"
                        description={dockerfileContent ? 'Build Dockerfile content' : dockerfileUnavailableReason}
                        width="xl"
                    >
                        <div className="flex items-center justify-between mb-4">
                            <span className="text-sm text-slate-500 dark:text-slate-400">
                                {dockerfileContent ? `${dockerfileContent.split('\n').length} lines` : 'No content'}
                            </span>
                            <button
                                onClick={handleCopyDockerfile}
                                disabled={!dockerfileContent}
                                className="text-sm text-blue-600 hover:text-blue-700 disabled:text-gray-400 disabled:cursor-not-allowed"
                            >
                                Copy
                            </button>
                        </div>
                        {dockerfileContent ? (
                            <pre className="bg-slate-950 text-slate-200 p-4 rounded-lg text-xs overflow-x-auto whitespace-pre-wrap">
                                {dockerfileContent}
                            </pre>
                        ) : (
                            <p className="text-sm text-gray-500 dark:text-gray-400">{dockerfileUnavailableReason}</p>
                        )}
                    </Drawer>

                    {/* Build Logs */}
                    <div className="card bg-white dark:bg-gray-800">
                        <div className="card-header">
                            <div className="flex items-center justify-between">
                                <div className="flex items-center gap-3">
                                    <h3 className="text-lg font-medium text-gray-900 dark:text-white">Build Logs</h3>
                                    <div className="inline-flex rounded-md border border-slate-300 dark:border-slate-700 overflow-hidden">
                                        <button
                                            type="button"
                                            onClick={() => setActiveLogTab('lifecycle')}
                                            className={`px-3 py-1.5 text-xs font-medium ${activeLogTab === 'lifecycle'
                                                ? 'bg-slate-200 text-slate-900 dark:bg-slate-700 dark:text-white'
                                                : 'bg-white text-slate-600 hover:bg-slate-100 dark:bg-slate-900 dark:text-slate-300 dark:hover:bg-slate-800'
                                                }`}
                                        >
                                            Lifecycle ({lifecycleLogs.length})
                                        </button>
                                        <button
                                            type="button"
                                            onClick={() => setActiveLogTab('execution')}
                                            className={`px-3 py-1.5 text-xs font-medium border-l border-slate-300 dark:border-slate-700 ${activeLogTab === 'execution'
                                                ? 'bg-slate-200 text-slate-900 dark:bg-slate-700 dark:text-white'
                                                : 'bg-white text-slate-600 hover:bg-slate-100 dark:bg-slate-900 dark:text-slate-300 dark:hover:bg-slate-800'
                                                }`}
                                        >
                                            Execution ({executionLogs.length})
                                        </button>
                                    </div>
                                </div>
                                <div className="flex items-center gap-3">
                                    <span className={`inline-flex items-center rounded-full px-2 py-1 text-[11px] font-medium ${isLogStreamConnected
                                        ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
                                        : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
                                        }`}>
                                        {isLogStreamConnected ? 'Live log stream on' : 'Live log stream off'}
                                    </span>
                                    {activeLogTab === 'execution' && executionTaskGroups.length > 0 && (
                                        <div className="inline-flex rounded-md border border-slate-300 dark:border-slate-700 overflow-hidden">
                                            <button
                                                type="button"
                                                onClick={expandAllExecutionLogTasks}
                                                className="px-2.5 py-1 text-[11px] bg-white text-slate-700 hover:bg-slate-100 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800"
                                            >
                                                Expand all
                                            </button>
                                            <button
                                                type="button"
                                                onClick={collapseAllExecutionLogTasks}
                                                className="px-2.5 py-1 text-[11px] border-l border-slate-300 bg-white text-slate-700 hover:bg-slate-100 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800"
                                            >
                                                Collapse all
                                            </button>
                                        </div>
                                    )}
                                    <label className="flex items-center text-sm text-gray-600 dark:text-gray-400">
                                        <input
                                            type="checkbox"
                                            checked={autoRefresh}
                                            onChange={(e) => setAutoRefresh(e.target.checked)}
                                            className="rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                                        />
                                        <span className="ml-2">Auto-refresh</span>
                                    </label>
                                    <button
                                        onClick={() => void loadBuild()}
                                        className="inline-flex items-center gap-1.5 rounded-md border border-slate-300 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 transition dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100 dark:hover:bg-slate-800"
                                    >
                                        <RefreshCcw className="h-3.5 w-3.5" />
                                        Refresh All
                                    </button>
                                </div>
                            </div>
                            {selectedAttempt && (
                                <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
                                    Showing logs for attempt {attempts.findIndex((attempt) => attempt.id === selectedAttempt.id) >= 0
                                        ? `#${attempts.length - attempts.findIndex((attempt) => attempt.id === selectedAttempt.id)}`
                                        : '(selected)'} ({selectedAttempt.id})
                                </p>
                            )}
                        </div>
                        <div className="card-body">
                            {isBuildActive && (
                                <div className={`mb-3 rounded-md border px-3 py-2 text-xs ${isLogStreamConnected
                                    ? 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-800 dark:bg-emerald-900/20 dark:text-emerald-200'
                                    : 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-200'
                                    }`}>
                                    {isLogStreamConnected
                                        ? (visibleLogs.length > 0 ? 'Live streaming logs as build progresses.' : 'Connected. Waiting for first log lines...')
                                        : 'Live stream disconnected. Reconnecting or use Refresh All.'}
                                </div>
                            )}
                            <div className="bg-slate-950 text-slate-200 p-4 rounded font-mono text-xs max-h-[36rem] overflow-y-auto">
                                {visibleLogs.length === 0 ? (
                                    <div className="text-gray-500">
                                        {activeLogTab === 'execution'
                                            ? 'No execution logs available yet.'
                                            : 'No lifecycle logs available yet.'}
                                    </div>
                                ) : (
                                    (() => {
                                        const hasTekton = activeLogTab === 'execution'
                                        if (!hasTekton) {
                                            return visibleLogs.map((log, index) => {
                                                const level = (log.level || 'INFO').toUpperCase()
                                                const ts = log.timestamp ? new Date(log.timestamp).toLocaleTimeString() : ''
                                                return (
                                                    <div key={index} className="mb-2">
                                                        <div className="flex items-start space-x-3">
                                                            <div className="text-xs text-gray-400 w-40 monospace break-words">
                                                                {ts && <span className="mr-2">{ts}</span>}
                                                                <span className="font-mono text-[11px] px-2 py-0.5 rounded-md bg-slate-800 text-slate-300">{level}</span>
                                                            </div>
                                                            <div className="flex-1">
                                                                <div className="text-sm text-slate-200 whitespace-pre-wrap">{log.message}</div>
                                                            </div>
                                                        </div>
                                                    </div>
                                                )
                                            })
                                        }

                                        return executionTaskGroups.map(([taskRun, stepMap], tIdx) => {
                                            const totalLines = Array.from(stepMap.values()).reduce((s, arr) => s + arr.length, 0)
                                            const taskLabel = taskRun === '__ungrouped__' ? 'Other' : taskRun
                                            const isExpanded = !!expandedExecutionLogTasks[taskRun]
                                            return (
                                                <div key={"task-" + tIdx} className="mb-3 border border-slate-800 rounded">
                                                    <button
                                                        type="button"
                                                        onClick={() => toggleExecutionLogTask(taskRun)}
                                                        className="w-full px-3 py-2 bg-slate-900 border-b border-slate-800 text-left hover:bg-slate-800 transition"
                                                    >
                                                        <div className="flex items-center justify-between gap-2">
                                                            <div className="text-sm font-medium text-slate-200">
                                                                TaskRun: {taskLabel} <span className="text-xs text-slate-400">({totalLines} lines)</span>
                                                            </div>
                                                            {isExpanded ? <ChevronUp className="h-4 w-4 text-slate-400" /> : <ChevronDown className="h-4 w-4 text-slate-400" />}
                                                        </div>
                                                    </button>
                                                    {isExpanded && (
                                                        <div className="px-3 py-2 space-y-2">
                                                            {Array.from(stepMap.entries()).map(([stepName, entries], sIdx) => (
                                                                <div key={"step-" + sIdx}>
                                                                    <div className="text-xs text-slate-400 mb-1">Step: {stepName}</div>
                                                                    {entries.map((log, i) => {
                                                                        const level = (log.level || 'INFO').toUpperCase()
                                                                        const ts = log.timestamp ? new Date(log.timestamp).toLocaleTimeString() : ''
                                                                        const metadataRowKey = `${taskRun}:${stepName}:${log.timestamp || 'no-ts'}:${i}`
                                                                        const hasExecutionMetadata = Boolean(
                                                                            log.metadata?.pipeline_run || log.metadata?.task_run || log.metadata?.step || log.metadata?.pod
                                                                        )
                                                                        const isMetadataExpanded = !!expandedLogMetadata[metadataRowKey]
                                                                        return (
                                                                            <div key={"entry-" + i} className="mb-1">
                                                                                <div className="flex items-start space-x-3">
                                                                                    <div className="text-xs text-gray-400 w-40 monospace break-words">
                                                                                        {ts && <span className="mr-2">{ts}</span>}
                                                                                        <span className="font-mono text-[11px] px-2 py-0.5 rounded-md bg-slate-800 text-slate-300">{level}</span>
                                                                                    </div>
                                                                                    <div className="flex-1">
                                                                                        <div className="text-sm text-slate-200 whitespace-pre-wrap">{log.message}</div>
                                                                                        {hasExecutionMetadata && (
                                                                                            <div className="mt-1">
                                                                                                <button
                                                                                                    type="button"
                                                                                                    onClick={() => toggleLogMetadata(metadataRowKey)}
                                                                                                    className="inline-flex items-center gap-1 text-[11px] text-slate-300 hover:text-slate-100 transition-colors"
                                                                                                    title={isMetadataExpanded ? 'Hide execution metadata' : 'Show execution metadata'}
                                                                                                >
                                                                                                    <Info size={12} />
                                                                                                    {isMetadataExpanded ? 'Hide metadata' : 'Show metadata'}
                                                                                                </button>
                                                                                                {isMetadataExpanded && (
                                                                                                    <div className="mt-1 text-xs text-slate-400 flex flex-wrap gap-2">
                                                                                                        {log.metadata?.pipeline_run && (<span className="px-2 py-0.5 bg-slate-700 rounded">PR: {String(log.metadata.pipeline_run)}</span>)}
                                                                                                        {log.metadata?.task_run && (<span className="px-2 py-0.5 bg-slate-700 rounded">Task: {String(log.metadata.task_run)}</span>)}
                                                                                                        {log.metadata?.step && (<span className="px-2 py-0.5 bg-slate-700 rounded">Step: {String(log.metadata.step)}</span>)}
                                                                                                        {log.metadata?.pod && (<span className="px-2 py-0.5 bg-slate-700 rounded">Pod: {String(log.metadata.pod)}</span>)}
                                                                                                    </div>
                                                                                                )}
                                                                                            </div>
                                                                                        )}
                                                                                    </div>
                                                                                </div>
                                                                            </div>
                                                                        )
                                                                    })}
                                                                </div>
                                                            ))}
                                                        </div>
                                                    )}
                                                </div>
                                            )
                                        })
                                    })()
                                )}
                            </div>
                        </div>
                    </div>

                    {/* Build Results */}
                    {(build.result || destinationImage) && (
                        <div className="card bg-white dark:bg-gray-800">
                            <div className="card-header">
                                <h3 className="text-lg font-medium text-gray-900 dark:text-white">Build Results</h3>
                            </div>
                            <div className="card-body">
                                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                                    <div className="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 sm:col-span-2 lg:col-span-3 dark:border-slate-700 dark:bg-slate-900/40">
                                        <dt className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Destination Image</dt>
                                        <dd className="mt-1 text-sm text-gray-900 dark:text-white font-mono break-all leading-5">
                                            {destinationImage || 'N/A'}
                                        </dd>
                                    </div>
                                    <div className="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-900/40">
                                        <dt className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Image Size</dt>
                                        <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                                            {build.result?.size ? `${(build.result.size / 1024 / 1024).toFixed(1)} MB` : 'N/A'}
                                        </dd>
                                    </div>
                                    <div className="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-900/40">
                                        <dt className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Duration</dt>
                                        <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                                            {build.result?.duration || 'N/A'}
                                        </dd>
                                    </div>
                                    <div className="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-900/40">
                                        <dt className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Image Digest</dt>
                                        <dd className="mt-1 text-sm text-gray-900 dark:text-white font-mono text-xs">
                                            {build.result?.imageDigest ? build.result.imageDigest.slice(0, 12) + '...' : 'N/A'}
                                        </dd>
                                    </div>
                                </div>

                                {build.result?.artifacts && build.result.artifacts.length > 0 && (
                                    <div className="mt-6">
                                        <h4 className="text-sm font-medium text-slate-900 dark:text-white mb-3">Artifacts</h4>
                                        <div className="space-y-2">
                                            {build.result.artifacts.map((artifact, index) => (
                                                <div key={index} className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded">
                                                    <span className="text-sm text-gray-900 dark:text-white">{artifact}</span>
                                                    <button className="text-blue-600 hover:text-blue-800 text-sm">
                                                        Download
                                                    </button>
                                                </div>
                                            ))}
                                        </div>
                                    </div>
                                )}
                            </div>
                        </div>
                    )}

                    {/* Repository Config Validation Error */}
                    {traceDiagnostics?.repoConfig?.error && (
                        <div className="card border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-900/20">
                            <div className="card-header">
                                <h3 className="text-lg font-medium text-amber-900 dark:text-amber-200">Repository Config Failure</h3>
                            </div>
                            <div className="card-body space-y-2">
                                <p className="text-amber-800 dark:text-amber-300 whitespace-pre-wrap">{traceDiagnostics.repoConfig.error}</p>
                                <div className="text-xs text-amber-900/80 dark:text-amber-200/80">
                                    {traceDiagnostics.repoConfig.path && <span>path={traceDiagnostics.repoConfig.path} </span>}
                                    {traceDiagnostics.repoConfig.ref && <span>ref={traceDiagnostics.repoConfig.ref} </span>}
                                    {traceDiagnostics.repoConfig.stage && <span>stage={traceDiagnostics.repoConfig.stage}</span>}
                                </div>
                            </div>
                        </div>
                    )}

                    {/* Error Message */}
                    {build.errorMessage && (
                        <div className="card border-red-200 bg-red-50 dark:bg-red-900/20">
                            <div className="card-header">
                                <h3 className="text-lg font-medium text-red-800 dark:text-red-200">Build Error</h3>
                            </div>
                            <div className="card-body">
                                <p className="text-red-700 dark:text-red-300 whitespace-pre-wrap">{build.errorMessage}</p>
                            </div>
                        </div>
                    )}
                </div>

                {/* Details Sidebar */}
                <div className="space-y-6">
                    <div className="card bg-white dark:bg-gray-800">
                        <div className="card-header">
                            <h3 className="text-lg font-medium text-gray-900 dark:text-white">Details</h3>
                        </div>
                        <div className="card-body space-y-4">
                            <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Tenant</p>
                                <p className="mt-1 text-sm text-gray-900 dark:text-white font-mono">{build.tenantId}</p>
                            </div>
                            <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Created</p>
                                <p className="mt-1 text-sm text-gray-900 dark:text-white">{formatDate(build.createdAt)}</p>
                            </div>
                            <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Updated</p>
                                <p className="mt-1 text-sm text-gray-900 dark:text-white">{formatDate(build.updatedAt)}</p>
                            </div>
                            <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Status</p>
                                <p className="mt-1 text-sm text-gray-900 dark:text-white">
                                    {build.status.charAt(0).toUpperCase() + build.status.slice(1)}
                                </p>
                            </div>
                            {selectedAttempt && (
                                <>
                                    <div>
                                        <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Selected Attempt</p>
                                        <p className="mt-1 text-sm text-gray-900 dark:text-white font-mono">{selectedAttempt.id}</p>
                                    </div>
                                    <div>
                                        <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Attempt Duration</p>
                                        <p className="mt-1 text-sm text-gray-900 dark:text-white">
                                            {formatDurationFromTimes(selectedAttempt.startedAt, selectedAttempt.completedAt)}
                                        </p>
                                    </div>
                                    <div>
                                        <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Attempt Started</p>
                                        <p className="mt-1 text-sm text-gray-900 dark:text-white">
                                            {selectedAttempt.startedAt ? formatDate(selectedAttempt.startedAt) : '-'}
                                        </p>
                                    </div>
                                    <div>
                                        <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Attempt Ended</p>
                                        <p className="mt-1 text-sm text-gray-900 dark:text-white">
                                            {selectedAttempt.completedAt ? formatDate(selectedAttempt.completedAt) : (isBuildActive ? 'In progress' : '-')}
                                        </p>
                                    </div>
                                </>
                            )}
                            <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Pipeline Started</p>
                                <p className="mt-1 text-sm text-gray-900 dark:text-white">
                                    {pipelineStartedAt ? formatDate(pipelineStartedAt) : '-'}
                                </p>
                            </div>
                            <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Pipeline Ended</p>
                                <p className="mt-1 text-sm text-gray-900 dark:text-white">
                                    {pipelineEndedAt ? formatDate(pipelineEndedAt) : (isBuildActive ? 'In progress' : '-')}
                                </p>
                            </div>
                            <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Pipeline End-to-End</p>
                                <p className="mt-1 text-sm text-gray-900 dark:text-white">{pipelineEndToEndDuration}</p>
                            </div>
                        </div>
                    </div>

                    <div className="card bg-white dark:bg-gray-800">
                        <div className="card-header">
                            <h3 className="text-lg font-medium text-gray-900 dark:text-white">Control Plane</h3>
                        </div>
                        <div className="card-body space-y-4">
                            <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Execution Status</p>
                                <p className="mt-1 text-sm font-medium text-gray-900 dark:text-white">{executionStatusLabel}</p>
                            </div>
                            <div>
                                <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Next Step</p>
                                <p className="mt-1 text-sm font-medium text-gray-900 dark:text-white">{executionNextStepLabel}</p>
                                {latestExecutionWaitHint && (
                                    <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                        {latestExecutionWaitHint}
                                    </p>
                                )}
                            </div>
                            {!hasWorkflowDetails ? (
                                <p className="text-sm text-slate-500 dark:text-slate-400">
                                    {controlPlaneUnavailableMessage}
                                </p>
                            ) : (
                                <>
                                    <div>
                                        <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Workflow Status</p>
                                        <p className="mt-1 text-sm font-medium text-gray-900 dark:text-white">{effectiveWorkflowStatus}</p>
                                    </div>
                                    <div>
                                        <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Workflow Step</p>
                                        <p className="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                                            {activeWorkflowStep ? (WORKFLOW_STEP_LABELS[activeWorkflowStep.stepKey] || activeWorkflowStep.stepKey) : 'None'}
                                        </p>
                                        {activeWorkflowStep && (
                                            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                                Step status: {activeWorkflowStep.status}
                                            </p>
                                        )}
                                    </div>
                                    {blockedOrFailedWorkflowSteps.length > 0 && (
                                        <div className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 dark:border-amber-900/60 dark:bg-amber-900/20">
                                            <p className="text-xs font-semibold uppercase tracking-wide text-amber-800 dark:text-amber-300">Recovery Hint</p>
                                            {blockedOrFailedWorkflowSteps.slice(0, 3).map((step, index) => {
                                                const actions = workflowRecoveryActionsForStep(step.stepKey)
                                                return (
                                                    <div key={`${step.stepKey}-${step.updatedAt}-${index}`} className="mt-2 rounded border border-amber-200/80 bg-amber-100/40 px-2.5 py-2 dark:border-amber-800/70 dark:bg-amber-900/20">
                                                        <p className="text-xs text-amber-800 dark:text-amber-200">
                                                            <span className="font-semibold">{WORKFLOW_STEP_LABELS[step.stepKey] || step.stepKey}</span>: {step.lastError || 'Step requires operator action.'}
                                                        </p>
                                                        {actions.length > 0 && (
                                                            <div className="mt-2 flex flex-wrap gap-2">
                                                                {actions.map((action) => (
                                                                    <button
                                                                        key={action.key}
                                                                        type="button"
                                                                        onClick={action.onClick}
                                                                        disabled={action.disabled}
                                                                        className="rounded-md border border-blue-300 px-2.5 py-1 text-xs font-medium text-blue-700 hover:bg-blue-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-blue-700 dark:text-blue-300 dark:hover:bg-blue-900/40"
                                                                    >
                                                                        {action.label}
                                                                    </button>
                                                                ))}
                                                            </div>
                                                        )}
                                                    </div>
                                                )
                                            })}
                                            <div className="mt-3 flex flex-wrap gap-2">
                                                {canShowRetryAction && (
                                                    <button
                                                        type="button"
                                                        onClick={handleRetryBuild}
                                                        className="rounded-md border border-amber-300 px-2.5 py-1 text-xs font-medium text-amber-700 hover:bg-amber-100 dark:border-amber-700 dark:text-amber-300 dark:hover:bg-amber-900/40"
                                                    >
                                                        Retry Build
                                                    </button>
                                                )}
                                                {isSystemAdmin && (
                                                    <Link
                                                        to="/admin"
                                                        className="rounded-md border border-blue-300 px-2.5 py-1 text-xs font-medium text-blue-700 hover:bg-blue-100 dark:border-blue-700 dark:text-blue-300 dark:hover:bg-blue-900/40"
                                                    >
                                                        Open Pipeline Health
                                                    </Link>
                                                )}
                                                {primaryRecoveryStep && (
                                                    <button
                                                        type="button"
                                                        onClick={handleExportTrace}
                                                        className="rounded-md border border-slate-300 px-2.5 py-1 text-xs font-medium text-slate-700 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
                                                    >
                                                        Export Trace
                                                    </button>
                                                )}
                                            </div>
                                            {!isSystemAdmin && (
                                                <p className="mt-2 text-[11px] text-amber-700 dark:text-amber-200">
                                                    Operator actions require system administrator permissions.
                                                </p>
                                            )}
                                        </div>
                                    )}
                                    {latestWorkflowUpdateAt && (
                                        <div>
                                            <p className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">Last Workflow Update</p>
                                            <p className="mt-1 text-sm text-gray-900 dark:text-white">{formatDate(latestWorkflowUpdateAt)}</p>
                                        </div>
                                    )}
                                </>
                            )}
                        </div>
                    </div>

                    <div className="card bg-white dark:bg-gray-800">
                        <div className="card-header">
                            <h3 className="text-lg font-medium text-gray-900 dark:text-white">Attempts</h3>
                        </div>
                        <div className="card-body">
                            {attempts.length === 0 ? (
                                <p className="text-sm text-slate-500 dark:text-slate-400">No execution attempts yet.</p>
                            ) : (
                                <div className="space-y-3">
                                    {attempts.map((attempt, index) => {
                                        const attemptNumber = attempts.length - index
                                        const isSelected = selectedExecutionId === attempt.id
                                        return (
                                            <button
                                                key={attempt.id}
                                                onClick={() => setSelectedExecutionId(attempt.id)}
                                                className={`w-full text-left rounded-md border px-3 py-2 transition ${isSelected
                                                    ? 'border-blue-500 bg-blue-50 text-blue-900 dark:border-blue-400 dark:bg-blue-900/20 dark:text-blue-100'
                                                    : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800'
                                                    }`}
                                            >
                                                <div className="flex items-center justify-between gap-2">
                                                    <span className="text-xs font-semibold">Attempt #{attemptNumber}</span>
                                                    <span className="text-[11px] uppercase tracking-wide">{attempt.status}</span>
                                                </div>
                                                <p className="mt-1 text-[11px] opacity-80">
                                                    {formatDurationFromTimes(attempt.startedAt, attempt.completedAt)}
                                                </p>
                                            </button>
                                        )
                                    })}
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default BuildDetailPage
