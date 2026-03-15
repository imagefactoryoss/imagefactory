import Drawer from '@/components/ui/Drawer'
import { adminService } from '@/services/adminService'
import type { SREActionAttempt, SREAgentDraftResponse, SREAgentInterpretationResponse, SREApproval, SREDemoScenario, SREEvidence, SREFinding, SREIncident, SREIncidentStatus, SREIncidentSeverity, SREIncidentWorkspaceResponse, SREMCPToolDescriptor, SREMCPToolInvocationResponse } from '@/types'
import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'

const severityTone: Record<SREIncidentSeverity, string> = {
    info: 'bg-sky-100 text-sky-800 border-sky-200 dark:bg-sky-500/15 dark:text-sky-200 dark:border-sky-500/30',
    warning: 'bg-amber-100 text-amber-900 border-amber-200 dark:bg-amber-500/15 dark:text-amber-200 dark:border-amber-500/30',
    critical: 'bg-rose-100 text-rose-900 border-rose-200 dark:bg-rose-500/15 dark:text-rose-200 dark:border-rose-500/30',
}

const statusTone: Record<SREIncidentStatus, string> = {
    observed: 'bg-slate-100 text-slate-800 border-slate-200 dark:bg-slate-700/40 dark:text-slate-100 dark:border-slate-600',
    triaged: 'bg-indigo-100 text-indigo-800 border-indigo-200 dark:bg-indigo-500/15 dark:text-indigo-200 dark:border-indigo-500/30',
    contained: 'bg-emerald-100 text-emerald-800 border-emerald-200 dark:bg-emerald-500/15 dark:text-emerald-200 dark:border-emerald-500/30',
    recovering: 'bg-cyan-100 text-cyan-800 border-cyan-200 dark:bg-cyan-500/15 dark:text-cyan-200 dark:border-cyan-500/30',
    resolved: 'bg-green-100 text-green-800 border-green-200 dark:bg-green-500/15 dark:text-green-200 dark:border-green-500/30',
    suppressed: 'bg-zinc-100 text-zinc-800 border-zinc-200 dark:bg-zinc-600/30 dark:text-zinc-100 dark:border-zinc-500',
    escalated: 'bg-fuchsia-100 text-fuchsia-800 border-fuchsia-200 dark:bg-fuchsia-500/15 dark:text-fuchsia-200 dark:border-fuchsia-500/30',
}

const actionTone = (status: string) => {
    switch ((status || '').toLowerCase()) {
        case 'completed':
        case 'succeeded':
        case 'resolved':
            return 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-200'
        case 'failed':
        case 'error':
            return 'border-rose-200 bg-rose-50 text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/30 dark:text-rose-200'
        case 'running':
        case 'started':
            return 'border-cyan-200 bg-cyan-50 text-cyan-800 dark:border-cyan-900/40 dark:bg-cyan-950/30 dark:text-cyan-200'
        default:
            return 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200'
    }
}

const formatDateTime = (value?: string | null) => {
    if (!value) return '—'
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return value
    return date.toLocaleString()
}

const relativeTime = (value?: string | null) => {
    if (!value) return '—'
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return value
    const diffMs = date.getTime() - Date.now()
    const diffMinutes = Math.round(diffMs / 60000)
    const absMinutes = Math.abs(diffMinutes)
    if (absMinutes < 1) return 'just now'
    if (absMinutes < 60) return `${absMinutes}m ${diffMinutes <= 0 ? 'ago' : 'from now'}`
    const diffHours = Math.round(absMinutes / 60)
    if (diffHours < 24) return `${diffHours}h ${diffMinutes <= 0 ? 'ago' : 'from now'}`
    const diffDays = Math.round(diffHours / 24)
    return `${diffDays}d ${diffMinutes <= 0 ? 'ago' : 'from now'}`
}

const prettyJson = (value: unknown) => {
    if (!value) return ''
    try {
        return JSON.stringify(value, null, 2)
    } catch {
        return String(value)
    }
}

const asRecord = (value: unknown): Record<string, any> | null => {
    if (!value || typeof value !== 'object' || Array.isArray(value)) return null
    return value as Record<string, any>
}

const asArrayOfRecords = (value: unknown): Record<string, any>[] => {
    if (!Array.isArray(value)) return []
    return value.map((item) => asRecord(item)).filter((item): item is Record<string, any> => item !== null)
}

const prettifyTrendLabel = (value?: string | null) => {
    if (!value) return 'Unknown'
    return value.replace(/_/g, ' ')
}

const renderAgentEvidenceRefs = (
    refs: { tool_name: string; server_name: string; summary: string }[],
    keyPrefix: string
) => (
    <div className="mt-2 space-y-2">
        {refs.map((evidenceRef, index) => (
            <div
                key={`${keyPrefix}-${evidenceRef.server_name}-${evidenceRef.tool_name}-${index}`}
                className="rounded-lg border border-slate-200 bg-white/80 px-3 py-2 dark:border-slate-800 dark:bg-slate-900/80"
            >
                <div className="flex flex-wrap items-center gap-2 text-[11px] font-medium uppercase tracking-[0.12em] text-slate-500 dark:text-slate-400">
                    <span>{evidenceRef.server_name}</span>
                    <span className="text-slate-300 dark:text-slate-600">/</span>
                    <span>{evidenceRef.tool_name}</span>
                </div>
                <div className="mt-1 text-xs text-slate-600 dark:text-slate-300">{evidenceRef.summary}</div>
            </div>
        ))}
    </div>
)

const renderMCPToolResult = (result: SREMCPToolInvocationResponse) => {
    const payload = asRecord(result.payload)
    if (!payload) {
        return (
            <pre className="overflow-x-auto whitespace-pre-wrap break-words text-xs text-slate-100">
                {prettyJson(result.payload)}
            </pre>
        )
    }

    if (result.tool_name === 'http_signals.recent') {
        const requestCount = typeof payload.request_count === 'number' ? payload.request_count : 0
        const serverErrorCount = typeof payload.server_error_count === 'number' ? payload.server_error_count : 0
        const clientErrorCount = typeof payload.client_error_count === 'number' ? payload.client_error_count : 0
        const errorRatePercent = typeof payload.error_rate_percent === 'number' ? payload.error_rate_percent : 0
        const averageLatencyMs = typeof payload.average_latency_ms === 'number' ? payload.average_latency_ms : 0
        const maxLatencyMs = typeof payload.max_latency_ms === 'number' ? payload.max_latency_ms : 0

        return (
            <div className="space-y-3">
                <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                    <div className="rounded-xl border border-slate-700 bg-slate-900/90 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-slate-400">Requests</div>
                        <div className="mt-2 text-xl font-semibold text-white">{requestCount}</div>
                        <div className="mt-1 text-xs text-slate-400">Latest captured HTTP window</div>
                    </div>
                    <div className="rounded-xl border border-rose-900/50 bg-rose-950/40 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-rose-300">Server Error Rate</div>
                        <div className="mt-2 text-xl font-semibold text-rose-100">{errorRatePercent}%</div>
                        <div className="mt-1 text-xs text-rose-200/80">{serverErrorCount} server errors</div>
                    </div>
                    <div className="rounded-xl border border-cyan-900/50 bg-cyan-950/40 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-cyan-300">Average Latency</div>
                        <div className="mt-2 text-xl font-semibold text-cyan-100">{averageLatencyMs}ms</div>
                        <div className="mt-1 text-xs text-cyan-200/80">Max {maxLatencyMs}ms</div>
                    </div>
                </div>
                <div className="grid gap-3 sm:grid-cols-2">
                    <div className="rounded-xl border border-slate-700 bg-slate-900/70 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-slate-400">Client Errors</div>
                        <div className="mt-2 text-lg font-semibold text-slate-100">{clientErrorCount}</div>
                    </div>
                    <div className="rounded-xl border border-slate-700 bg-slate-900/70 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-slate-400">Runner State</div>
                        <div className="mt-2 text-sm font-medium text-slate-100">
                            {payload.runner_running ? 'Running' : 'Not running'}
                        </div>
                        <div className="mt-1 text-xs text-slate-400">{typeof payload.message === 'string' ? payload.message : 'No runner message available.'}</div>
                    </div>
                </div>
                <details className="rounded-xl border border-slate-700 bg-slate-900/70">
                    <summary className="cursor-pointer list-none px-3 py-2 text-xs font-medium uppercase tracking-[0.14em] text-slate-300">
                        Raw HTTP Signal Payload
                    </summary>
                    <div className="border-t border-slate-700 px-3 py-3">
                        <pre className="overflow-x-auto whitespace-pre-wrap break-words text-xs text-slate-100">
                            {prettyJson(result.payload)}
                        </pre>
                    </div>
                </details>
            </div>
        )
    }

    if (result.tool_name === 'http_signals.history') {
        const windows = Array.isArray(payload.windows) ? payload.windows : []
        const latestWindow = windows[0] && typeof windows[0] === 'object' ? asRecord(windows[0]) : null
        const trend = typeof payload.trend === 'string' ? payload.trend : 'unknown'
        const averageErrorRate = typeof payload.average_error_rate_percent === 'number' ? payload.average_error_rate_percent : 0
        const averageLatencyMs = typeof payload.average_latency_ms === 'number' ? payload.average_latency_ms : 0
        const peakRequestCount = typeof payload.peak_request_count === 'number' ? payload.peak_request_count : 0

        return (
            <div className="space-y-3">
                <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                    <div className="rounded-xl border border-violet-900/50 bg-violet-950/40 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-violet-300">Trend</div>
                        <div className="mt-2 text-lg font-semibold text-violet-100">{prettifyTrendLabel(trend)}</div>
                        <div className="mt-1 text-xs text-violet-200/80">{windows.length} recent windows retained</div>
                    </div>
                    <div className="rounded-xl border border-cyan-900/50 bg-cyan-950/40 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-cyan-300">Avg Latency</div>
                        <div className="mt-2 text-lg font-semibold text-cyan-100">{averageLatencyMs}ms</div>
                        <div className="mt-1 text-xs text-cyan-200/80">Across retained windows</div>
                    </div>
                    <div className="rounded-xl border border-rose-900/50 bg-rose-950/40 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-rose-300">Avg Error Rate</div>
                        <div className="mt-2 text-lg font-semibold text-rose-100">{averageErrorRate}%</div>
                        <div className="mt-1 text-xs text-rose-200/80">Server-side error pressure</div>
                    </div>
                    <div className="rounded-xl border border-slate-700 bg-slate-900/90 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-slate-400">Peak Traffic</div>
                        <div className="mt-2 text-lg font-semibold text-white">{peakRequestCount}</div>
                        <div className="mt-1 text-xs text-slate-400">Highest captured request count</div>
                    </div>
                </div>

                {latestWindow ? (
                    <div className="rounded-xl border border-slate-700 bg-slate-900/70 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-slate-400">Latest Window</div>
                        <div className="mt-2 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                            <div>
                                <div className="text-xs text-slate-400">Requests</div>
                                <div className="mt-1 text-sm font-semibold text-slate-100">{String(latestWindow.request_count ?? 0)}</div>
                            </div>
                            <div>
                                <div className="text-xs text-slate-400">Avg Latency</div>
                                <div className="mt-1 text-sm font-semibold text-slate-100">{String(latestWindow.average_latency_ms ?? 0)}ms</div>
                            </div>
                            <div>
                                <div className="text-xs text-slate-400">Server Errors</div>
                                <div className="mt-1 text-sm font-semibold text-slate-100">{String(latestWindow.server_error_count ?? 0)}</div>
                            </div>
                            <div>
                                <div className="text-xs text-slate-400">Captured</div>
                                <div className="mt-1 text-sm font-semibold text-slate-100">{formatDateTime(String(latestWindow.window_ended_at ?? ''))}</div>
                            </div>
                        </div>
                    </div>
                ) : null}

                <details className="rounded-xl border border-slate-700 bg-slate-900/70">
                    <summary className="cursor-pointer list-none px-3 py-2 text-xs font-medium uppercase tracking-[0.14em] text-slate-300">
                        Raw HTTP Signal History
                    </summary>
                    <div className="border-t border-slate-700 px-3 py-3">
                        <pre className="overflow-x-auto whitespace-pre-wrap break-words text-xs text-slate-100">
                            {prettyJson(result.payload)}
                        </pre>
                    </div>
                </details>
            </div>
        )
    }

    if (result.tool_name === 'async_backlog.recent') {
        const buildQueueDepth = typeof payload.build_queue_depth === 'number' ? payload.build_queue_depth : 0
        const emailQueueDepth = typeof payload.email_queue_depth === 'number' ? payload.email_queue_depth : 0
        const outboxPending = typeof payload.messaging_outbox_pending === 'number' ? payload.messaging_outbox_pending : 0
        const buildQueueThreshold = typeof payload.build_queue_threshold === 'number' ? payload.build_queue_threshold : 0
        const emailQueueThreshold = typeof payload.email_queue_threshold === 'number' ? payload.email_queue_threshold : 0
        const outboxThreshold = typeof payload.messaging_outbox_threshold === 'number' ? payload.messaging_outbox_threshold : 0

        return (
            <div className="space-y-3">
                <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                    <div className="rounded-xl border border-amber-900/50 bg-amber-950/40 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-amber-300">Build Queue</div>
                        <div className="mt-2 text-xl font-semibold text-amber-100">{buildQueueDepth}</div>
                        <div className="mt-1 text-xs text-amber-200/80">Threshold {buildQueueThreshold}</div>
                    </div>
                    <div className="rounded-xl border border-sky-900/50 bg-sky-950/40 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-sky-300">Email Queue</div>
                        <div className="mt-2 text-xl font-semibold text-sky-100">{emailQueueDepth}</div>
                        <div className="mt-1 text-xs text-sky-200/80">Threshold {emailQueueThreshold}</div>
                    </div>
                    <div className="rounded-xl border border-fuchsia-900/50 bg-fuchsia-950/40 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-fuchsia-300">Outbox Pending</div>
                        <div className="mt-2 text-xl font-semibold text-fuchsia-100">{outboxPending}</div>
                        <div className="mt-1 text-xs text-fuchsia-200/80">Threshold {outboxThreshold}</div>
                    </div>
                </div>
                <div className="rounded-xl border border-slate-700 bg-slate-900/70 px-3 py-3">
                    <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-slate-400">Runner State</div>
                    <div className="mt-2 text-sm font-medium text-slate-100">
                        {payload.runner_running ? 'Running' : 'Not running'}
                    </div>
                    <div className="mt-1 text-xs text-slate-400">{typeof payload.message === 'string' ? payload.message : 'No runner message available.'}</div>
                </div>
                <details className="rounded-xl border border-slate-700 bg-slate-900/70">
                    <summary className="cursor-pointer list-none px-3 py-2 text-xs font-medium uppercase tracking-[0.14em] text-slate-300">
                        Raw Async Backlog Payload
                    </summary>
                    <div className="border-t border-slate-700 px-3 py-3">
                        <pre className="overflow-x-auto whitespace-pre-wrap break-words text-xs text-slate-100">
                            {prettyJson(result.payload)}
                        </pre>
                    </div>
                </details>
            </div>
        )
    }

    if (result.tool_name === 'messaging_transport.recent') {
        const reconnects = typeof payload.reconnects === 'number' ? payload.reconnects : 0
        const disconnects = typeof payload.disconnects === 'number' ? payload.disconnects : 0
        const threshold = typeof payload.reconnect_threshold === 'number' ? payload.reconnect_threshold : 0

        return (
            <div className="space-y-3">
                <div className="grid gap-3 sm:grid-cols-3">
                    <div className="rounded-xl border border-cyan-900/50 bg-cyan-950/40 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-cyan-300">Reconnects</div>
                        <div className="mt-2 text-xl font-semibold text-cyan-100">{reconnects}</div>
                        <div className="mt-1 text-xs text-cyan-200/80">Threshold {threshold}</div>
                    </div>
                    <div className="rounded-xl border border-rose-900/50 bg-rose-950/40 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-rose-300">Disconnects</div>
                        <div className="mt-2 text-xl font-semibold text-rose-100">{disconnects}</div>
                        <div className="mt-1 text-xs text-rose-200/80">Recent transport interruptions</div>
                    </div>
                    <div className="rounded-xl border border-slate-700 bg-slate-900/70 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-slate-400">Runner State</div>
                        <div className="mt-2 text-sm font-medium text-slate-100">
                            {payload.runner_running ? 'Running' : 'Not running'}
                        </div>
                        <div className="mt-1 text-xs text-slate-400">{typeof payload.message === 'string' ? payload.message : 'No runner message available.'}</div>
                    </div>
                </div>
            </div>
        )
    }

    return (
        <pre className="overflow-x-auto whitespace-pre-wrap break-words text-xs text-slate-100">
            {prettyJson(result.payload)}
        </pre>
    )
}

const SectionCard: React.FC<{ title: string; subtitle?: string; children: React.ReactNode }> = ({ title, subtitle, children }) => (
    <section className="rounded-2xl border border-slate-200 bg-white/90 p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900/85">
        <div className="mb-4">
            <h2 className="text-sm font-semibold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">{title}</h2>
            {subtitle ? <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">{subtitle}</p> : null}
        </div>
        {children}
    </section>
)

const EmptyState: React.FC<{ title: string; description: string }> = ({ title, description }) => (
    <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-4 py-6 text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-800/40 dark:text-slate-300">
        <p className="font-medium text-slate-800 dark:text-slate-100">{title}</p>
        <p className="mt-1">{description}</p>
    </div>
)

type IncidentDrawerTab = 'summary' | 'ai' | 'signals' | 'actions'

const drawerTabs: Array<{ id: IncidentDrawerTab; label: string; hint: string }> = [
    { id: 'summary', label: 'Summary', hint: 'Snapshot, overview, and email-ready narrative' },
    { id: 'ai', label: 'AI Workspace', hint: 'Grounded MCP, draft hypotheses, and local interpretation' },
    { id: 'signals', label: 'Signals', hint: 'Findings and evidence captured for this thread' },
    { id: 'actions', label: 'Actions', hint: 'Action attempts, approvals, and operator controls' },
]

const SRESmartBotIncidentsPage: React.FC = () => {
    const [searchParams, setSearchParams] = useSearchParams()
    const navigate = useNavigate()
    const [incidents, setIncidents] = useState<SREIncident[]>([])
    const [demoScenarios, setDemoScenarios] = useState<SREDemoScenario[]>([])
    const [total, setTotal] = useState(0)
    const [loading, setLoading] = useState(true)
    const [demoLoading, setDemoLoading] = useState(true)
    const [generatingDemo, setGeneratingDemo] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [statusFilter, setStatusFilter] = useState<string>('')
    const [severityFilter, setSeverityFilter] = useState<string>('')
    const [domainFilter, setDomainFilter] = useState('')
    const [search, setSearch] = useState('')
    const [selectedDemoScenarioId, setSelectedDemoScenarioId] = useState('ldap_timeout')
    const [selectedIncident, setSelectedIncident] = useState<SREIncident | null>(null)
    const [drawerTab, setDrawerTab] = useState<IncidentDrawerTab>('summary')
    const [drawerOpen, setDrawerOpen] = useState(false)
    const [drawerLoading, setDrawerLoading] = useState(false)
    const [mutatingActionId, setMutatingActionId] = useState<string | null>(null)
    const [mutatingApprovalId, setMutatingApprovalId] = useState<string | null>(null)
    const [emailingSummary, setEmailingSummary] = useState(false)
    const [proposingDetectorRule, setProposingDetectorRule] = useState(false)
    const [drawerError, setDrawerError] = useState<string | null>(null)
    const [findings, setFindings] = useState<SREFinding[]>([])
    const [evidence, setEvidence] = useState<SREEvidence[]>([])
    const [actions, setActions] = useState<SREActionAttempt[]>([])
    const [approvals, setApprovals] = useState<SREApproval[]>([])
    const [workspace, setWorkspace] = useState<SREIncidentWorkspaceResponse | null>(null)
    const [mcpTools, setMcpTools] = useState<SREMCPToolDescriptor[]>([])
    const [mcpResults, setMcpResults] = useState<Record<string, SREMCPToolInvocationResponse>>({})
    const [runningMCPToolKey, setRunningMCPToolKey] = useState<string | null>(null)
    const [agentDraft, setAgentDraft] = useState<SREAgentDraftResponse | null>(null)
    const [generatingAgentDraft, setGeneratingAgentDraft] = useState(false)
    const [agentInterpretation, setAgentInterpretation] = useState<SREAgentInterpretationResponse | null>(null)
    const [generatingInterpretation, setGeneratingInterpretation] = useState(false)

    const uniqueDomains = useMemo(() => {
        const values = new Set<string>()
        incidents.forEach((incident) => {
            if (incident.domain) values.add(incident.domain)
        })
        return Array.from(values).sort()
    }, [incidents])

    const actionsById = useMemo(() => {
        return actions.reduce<Record<string, SREActionAttempt>>((acc, action) => {
            acc[action.id] = action
            return acc
        }, {})
    }, [actions])

    const selectedIncidentIdFromUrl = searchParams.get('incident') || ''

    const syncIncidentQuery = (incidentId?: string | null) => {
        const next = new URLSearchParams(searchParams)
        if (incidentId) {
            next.set('incident', incidentId)
        } else {
            next.delete('incident')
        }
        setSearchParams(next, { replace: true })
    }

    const loadIncidents = async () => {
        try {
            setLoading(true)
            setError(null)
            const response = await adminService.getSREIncidents({
                limit: 100,
                offset: 0,
                status: statusFilter || undefined,
                severity: severityFilter || undefined,
                domain: domainFilter || undefined,
                search: search.trim() || undefined,
            })
            setIncidents(response.incidents || [])
            setTotal(response.total || 0)
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load SRE incidents')
            setIncidents([])
            setTotal(0)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        void loadIncidents()
    }, [statusFilter, severityFilter, domainFilter])

    useEffect(() => {
        const loadDemoScenarios = async () => {
            try {
                setDemoLoading(true)
                const response = await adminService.getSREDemoScenarios()
                setDemoScenarios(response.scenarios || [])
                if ((response.scenarios || []).length > 0) {
                    setSelectedDemoScenarioId((current) => current || response.scenarios[0].id)
                }
            } catch (err) {
                toast.error(err instanceof Error ? err.message : 'Failed to load demo scenarios')
            } finally {
                setDemoLoading(false)
            }
        }
        void loadDemoScenarios()
    }, [])

    useEffect(() => {
        if (!selectedIncidentIdFromUrl) {
            return
        }
        if (selectedIncident?.id === selectedIncidentIdFromUrl && drawerOpen) {
            return
        }
        void loadIncidentDetail(
            selectedIncidentIdFromUrl,
            incidents.find((incident) => incident.id === selectedIncidentIdFromUrl),
            true,
        )
    }, [selectedIncidentIdFromUrl, incidents, selectedIncident?.id, drawerOpen])

    const loadIncidentDetail = async (incidentId: string, seedIncident?: SREIncident, openDrawer?: boolean) => {
        if (seedIncident) {
            setSelectedIncident(seedIncident)
        }
        if (openDrawer) {
            setDrawerOpen(true)
        }
        setDrawerTab('summary')
        setDrawerLoading(true)
        setDrawerError(null)
        setFindings([])
        setEvidence([])
        setActions([])
        setApprovals([])
        setWorkspace(null)
        setMcpTools([])
        setMcpResults({})
        setAgentDraft(null)
        setAgentInterpretation(null)
        try {
            const [response, workspaceResponse, mcpToolResponse] = await Promise.all([
                adminService.getSREIncident(incidentId),
                adminService.getSREIncidentWorkspace(incidentId).catch(() => null),
                adminService.getSREIncidentMCPTools(incidentId).catch(() => ({ tools: [] })),
            ])
            setSelectedIncident(response.incident)
            setFindings(response.findings || [])
            setEvidence(response.evidence || [])
            setActions(response.action_attempts || [])
            setApprovals(response.approvals || [])
            setWorkspace(workspaceResponse)
            setMcpTools(mcpToolResponse.tools || [])
        } catch (err) {
            setDrawerError(err instanceof Error ? err.message : 'Failed to load incident details')
        } finally {
            setDrawerLoading(false)
        }
    }

    const openIncident = async (incident: SREIncident) => {
        syncIncidentQuery(incident.id)
        await loadIncidentDetail(incident.id, incident, true)
    }

    const closeDrawer = () => {
        syncIncidentQuery(null)
        setDrawerOpen(false)
        setDrawerTab('summary')
        setSelectedIncident(null)
        setDrawerError(null)
        setFindings([])
        setEvidence([])
        setActions([])
        setApprovals([])
        setWorkspace(null)
        setMcpTools([])
        setMcpResults({})
        setAgentDraft(null)
        setAgentInterpretation(null)
    }

    const handleRequestApproval = async (action: SREActionAttempt) => {
        if (!selectedIncident) return
        try {
            setMutatingActionId(action.id)
            await adminService.requestSREActionApproval(selectedIncident.id, action.id)
            toast.success('Approval requested')
            await loadIncidentDetail(selectedIncident.id)
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to request approval')
        } finally {
            setMutatingActionId(null)
        }
    }

    const handleExecuteAction = async (action: SREActionAttempt) => {
        if (!selectedIncident) return
        try {
            setMutatingActionId(action.id)
            await adminService.executeSREAction(selectedIncident.id, action.id)
            toast.success('Action executed')
            await loadIncidentDetail(selectedIncident.id)
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to execute action')
        } finally {
            setMutatingActionId(null)
        }
    }

    const handleProposeDetectorRule = async () => {
        if (!selectedIncident) return
        try {
            setProposingDetectorRule(true)
            const suggestion = await adminService.proposeSREDetectorRuleSuggestion(selectedIncident.id)
            toast.success('Detector rule suggestion created')
            navigate(`/admin/operations/sre-smart-bot/detector-rules?suggestion=${encodeURIComponent(suggestion.id)}&incident=${encodeURIComponent(selectedIncident.id)}`)
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to propose detector rule')
        } finally {
            setProposingDetectorRule(false)
        }
    }

    const handleApprovalDecision = async (approval: SREApproval, decision: 'approved' | 'rejected') => {
        if (!selectedIncident) return
        try {
            setMutatingApprovalId(approval.id)
            await adminService.decideSREApproval(selectedIncident.id, approval.id, { decision })
            toast.success(decision === 'approved' ? 'Approval granted' : 'Approval rejected')
            await loadIncidentDetail(selectedIncident.id)
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to update approval')
        } finally {
            setMutatingApprovalId(null)
        }
    }

    const handleEmailSummary = async () => {
        if (!selectedIncident) return
        try {
            setEmailingSummary(true)
            await adminService.emailSREIncidentSummary(selectedIncident.id)
            toast.success('Incident summary queued for admin email delivery')
            await loadIncidentDetail(selectedIncident.id)
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to email incident summary')
        } finally {
            setEmailingSummary(false)
        }
    }

    const handleRunMCPTool = async (tool: SREMCPToolDescriptor) => {
        if (!selectedIncident) return
        const toolKey = `${tool.server_id}:${tool.tool_name}`
        try {
            setRunningMCPToolKey(toolKey)
            const response = await adminService.invokeSREIncidentMCPTool(selectedIncident.id, {
                server_id: tool.server_id,
                tool_name: tool.tool_name,
            })
            setMcpResults((current) => ({ ...current, [toolKey]: response }))
            toast.success(`${tool.display_name} completed`)
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to run MCP tool')
        } finally {
            setRunningMCPToolKey(null)
        }
    }

    const handleDismissMCPToolResult = (toolKey: string) => {
        setMcpResults((current) => {
            if (!(toolKey in current)) return current
            const next = { ...current }
            delete next[toolKey]
            return next
        })
    }

    const handleGenerateAgentDraft = async () => {
        if (!selectedIncident) return
        try {
            setGeneratingAgentDraft(true)
            const response = await adminService.getSREIncidentAgentDraft(selectedIncident.id)
            setAgentDraft(response)
            toast.success('Draft hypothesis and investigation plan generated')
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to generate agent draft')
        } finally {
            setGeneratingAgentDraft(false)
        }
    }

    const handleGenerateInterpretation = async () => {
        if (!selectedIncident) return
        try {
            setGeneratingInterpretation(true)
            const response = await adminService.getSREIncidentAgentInterpretation(selectedIncident.id)
            setAgentInterpretation(response)
            toast.success(response.generated ? 'Local model interpretation generated' : 'Interpretation baseline loaded')
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to generate interpretation')
        } finally {
            setGeneratingInterpretation(false)
        }
    }

    const handleGenerateDemoIncident = async () => {
        if (!selectedDemoScenarioId) return
        try {
            setGeneratingDemo(true)
            const response = await adminService.generateSREDemoIncident(selectedDemoScenarioId)
            const incident = response.incident
            toast.success('Demo incident generated')
            await loadIncidents()
            if (incident?.id) {
                syncIncidentQuery(incident.id)
                await loadIncidentDetail(incident.id, incident, true)
            }
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to generate demo incident')
        } finally {
            setGeneratingDemo(false)
        }
    }

    const canExecuteAction = (action: SREActionAttempt) =>
        ['reconcile_tenant_assets', 'review_provider_connectivity'].includes(action.action_key) &&
        ['proposed', 'approved'].includes((action.status || '').toLowerCase())

    const summaryEmailActions = useMemo(
        () => actions.filter((action) => action.action_key === 'email_incident_summary'),
        [actions],
    )

    const httpSignalsRecentTool = useMemo(
        () => mcpTools.find((tool) => tool.tool_name === 'http_signals.recent'),
        [mcpTools],
    )

    const httpSignalsHistoryTool = useMemo(
        () => mcpTools.find((tool) => tool.tool_name === 'http_signals.history'),
        [mcpTools],
    )

    const httpSignalsRecentKey = httpSignalsRecentTool ? `${httpSignalsRecentTool.server_id}:${httpSignalsRecentTool.tool_name}` : ''
    const httpSignalsHistoryKey = httpSignalsHistoryTool ? `${httpSignalsHistoryTool.server_id}:${httpSignalsHistoryTool.tool_name}` : ''

    const httpSignalsRecentResult = httpSignalsRecentKey ? mcpResults[httpSignalsRecentKey] : undefined
    const httpSignalsHistoryResult = httpSignalsHistoryKey ? mcpResults[httpSignalsHistoryKey] : undefined
    const asyncBacklogTool = useMemo(
        () => mcpTools.find((tool) => tool.tool_name === 'async_backlog.recent'),
        [mcpTools],
    )
    const asyncBacklogKey = asyncBacklogTool ? `${asyncBacklogTool.server_id}:${asyncBacklogTool.tool_name}` : ''
    const asyncBacklogResult = asyncBacklogKey ? mcpResults[asyncBacklogKey] : undefined
    const messagingTransportTool = useMemo(
        () => mcpTools.find((tool) => tool.tool_name === 'messaging_transport.recent'),
        [mcpTools],
    )
    const messagingTransportKey = messagingTransportTool ? `${messagingTransportTool.server_id}:${messagingTransportTool.tool_name}` : ''
    const messagingTransportResult = messagingTransportKey ? mcpResults[messagingTransportKey] : undefined

    const summaryDeliveryStats = useMemo(() => {
        const successful = summaryEmailActions.filter((action) => (action.status || '').toLowerCase() === 'completed')
        const latest = summaryEmailActions[0]
        const latestPayload = asRecord(latest?.result_payload)
        return {
            total: summaryEmailActions.length,
            successful: successful.length,
            latest,
            latestRecipients: Array.isArray(latestPayload?.recipients) ? latestPayload?.recipients as string[] : [],
            latestSentCount: typeof latestPayload?.sent_count === 'number' ? latestPayload.sent_count : 0,
        }
    }, [summaryEmailActions])

    const httpSignalsSummary = useMemo(() => {
        const recentPayload = asRecord(httpSignalsRecentResult?.payload)
        const historyPayload = asRecord(httpSignalsHistoryResult?.payload)
        const windows = asArrayOfRecords(historyPayload?.windows)
        return {
            recentRequestCount: typeof recentPayload?.request_count === 'number' ? recentPayload.request_count : 0,
            recentErrorRatePercent: typeof recentPayload?.error_rate_percent === 'number' ? recentPayload.error_rate_percent : 0,
            recentAverageLatencyMs: typeof recentPayload?.average_latency_ms === 'number' ? recentPayload.average_latency_ms : 0,
            historyTrend: typeof historyPayload?.trend === 'string' ? historyPayload.trend : '',
            historyAverageLatencyMs: typeof historyPayload?.average_latency_ms === 'number' ? historyPayload.average_latency_ms : 0,
            historyAverageErrorRatePercent: typeof historyPayload?.average_error_rate_percent === 'number' ? historyPayload.average_error_rate_percent : 0,
            historyPeakRequestCount: typeof historyPayload?.peak_request_count === 'number' ? historyPayload.peak_request_count : 0,
            historyWindowCount: windows.length,
            latestHistoryWindowEndedAt: typeof windows[0]?.window_ended_at === 'string' ? windows[0].window_ended_at : '',
        }
    }, [httpSignalsRecentResult, httpSignalsHistoryResult])

    const asyncBacklogSummary = useMemo(() => {
        const payload = asRecord(asyncBacklogResult?.payload)
        return {
            buildQueueDepth: typeof payload?.build_queue_depth === 'number' ? payload.build_queue_depth : 0,
            emailQueueDepth: typeof payload?.email_queue_depth === 'number' ? payload.email_queue_depth : 0,
            messagingOutboxPending: typeof payload?.messaging_outbox_pending === 'number' ? payload.messaging_outbox_pending : 0,
            buildQueueThreshold: typeof payload?.build_queue_threshold === 'number' ? payload.build_queue_threshold : 0,
            emailQueueThreshold: typeof payload?.email_queue_threshold === 'number' ? payload.email_queue_threshold : 0,
            messagingOutboxThreshold: typeof payload?.messaging_outbox_threshold === 'number' ? payload.messaging_outbox_threshold : 0,
            lastActivity: typeof payload?.last_activity === 'string' ? payload.last_activity : '',
        }
    }, [asyncBacklogResult])

    const messagingTransportSummary = useMemo(() => {
        const payload = asRecord(messagingTransportResult?.payload)
        return {
            reconnects: typeof payload?.reconnects === 'number' ? payload.reconnects : 0,
            disconnects: typeof payload?.disconnects === 'number' ? payload.disconnects : 0,
            reconnectThreshold: typeof payload?.reconnect_threshold === 'number' ? payload.reconnect_threshold : 0,
            lastActivity: typeof payload?.last_activity === 'string' ? payload.last_activity : '',
        }
    }, [messagingTransportResult])

    const executiveSummary = useMemo(() => {
        if (!selectedIncident) return []
        const pendingApprovals = approvals.filter((approval) => !approval.decided_at).length
        const approvedActionsReady = actions.filter((action) =>
            ['reconcile_tenant_assets', 'review_provider_connectivity'].includes(action.action_key) &&
            (action.status || '').toLowerCase() === 'approved',
        ).length
        const topFinding = findings[0]?.title || findings[0]?.message || 'No finding titles recorded yet.'
        const latestAction = actions[0]
        const summary = [
            `${selectedIncident.display_name} is currently ${selectedIncident.status} with ${selectedIncident.severity} severity in ${selectedIncident.domain}.`,
            pendingApprovals > 0
                ? `${pendingApprovals} approval request${pendingApprovals === 1 ? '' : 's'} still need operator attention.`
                : 'There are no pending approval requests on this incident thread.',
            approvedActionsReady > 0
                ? `${approvedActionsReady} approved executable action${approvedActionsReady === 1 ? '' : 's'} can be run now.`
                : 'No approved executable actions are currently waiting to run.',
            `Most recent signal: ${topFinding}`,
            latestAction
                ? `Latest action activity: ${latestAction.action_key} is ${latestAction.status}.`
                : 'No remediation actions have been attempted yet.',
        ]

        if (selectedIncident.incident_type === 'backlog_pressure') {
            if (messagingTransportSummary.reconnects > 0 || messagingTransportSummary.disconnects > 0) {
                summary.push(
                    `Messaging transport instability may be contributing to backlog growth: reconnects=${messagingTransportSummary.reconnects}, disconnects=${messagingTransportSummary.disconnects}.`,
                )
            } else {
                summary.push('Backlog pressure is present without current messaging transport instability, which points more toward downstream processing congestion than bus connectivity.')
            }
        }

        if (selectedIncident.incident_type === 'messaging_transport_degraded') {
            if (asyncBacklogSummary.buildQueueDepth > 0 || asyncBacklogSummary.emailQueueDepth > 0 || asyncBacklogSummary.messagingOutboxPending > 0) {
                summary.push(
                    `Transport instability is occurring alongside async pressure: build queue=${asyncBacklogSummary.buildQueueDepth}, email queue=${asyncBacklogSummary.emailQueueDepth}, outbox pending=${asyncBacklogSummary.messagingOutboxPending}.`,
                )
            } else {
                summary.push('Transport instability is currently visible without major async backlog buildup, which suggests an early-stage messaging issue rather than sustained queue congestion.')
            }
        }

        return summary
    }, [selectedIncident, approvals, actions, findings, asyncBacklogSummary, messagingTransportSummary])

    useEffect(() => {
        const autoLoadGoldenSignalContext = async () => {
            if (!selectedIncident || !drawerOpen || drawerLoading) return
            if (!selectedIncident.domain?.includes('golden_signals') && !selectedIncident.incident_type?.includes('golden_signals')) return

            const shouldLoadAsyncBacklog = selectedIncident.incident_type === 'backlog_pressure'
            const shouldLoadMessagingTransport = selectedIncident.incident_type === 'messaging_transport_degraded'
            const toolsToRun = [httpSignalsRecentTool, httpSignalsHistoryTool, shouldLoadAsyncBacklog ? asyncBacklogTool : undefined, shouldLoadMessagingTransport ? messagingTransportTool : undefined].filter(
                (tool): tool is SREMCPToolDescriptor => !!tool && !mcpResults[`${tool.server_id}:${tool.tool_name}`],
            )
            if (toolsToRun.length === 0) return

            for (const tool of toolsToRun) {
                try {
                    const response = await adminService.invokeSREIncidentMCPTool(selectedIncident.id, {
                        server_id: tool.server_id,
                        tool_name: tool.tool_name,
                    })
                    const toolKey = `${tool.server_id}:${tool.tool_name}`
                    setMcpResults((current) => ({ ...current, [toolKey]: response }))
                } catch {
                    // Keep summary loading resilient; operators can still run the tool manually in AI Workspace.
                }
            }
        }

        void autoLoadGoldenSignalContext()
    }, [
        selectedIncident,
        drawerOpen,
        drawerLoading,
        httpSignalsRecentTool,
        httpSignalsHistoryTool,
        asyncBacklogTool,
        messagingTransportTool,
        mcpResults,
    ])

    const selectedDemoScenario = useMemo(
        () => demoScenarios.find((scenario) => scenario.id === selectedDemoScenarioId) || null,
        [demoScenarios, selectedDemoScenarioId],
    )

    return (
        <div className="min-h-full w-full bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.08),_transparent_30%),linear-gradient(180deg,_#f8fafc_0%,_#eef2ff_100%)] px-4 py-6 text-slate-900 sm:px-6 lg:px-8 dark:bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.16),_transparent_24%),linear-gradient(180deg,_#020617_0%,_#0f172a_100%)] dark:text-slate-100">
            <div className="w-full space-y-6">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                    <div>
                        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">Operations</p>
                        <h1 className="mt-2 text-3xl font-semibold tracking-tight">SRE Smart Bot Incidents</h1>
                        <p className="mt-2 max-w-3xl text-sm text-slate-600 dark:text-slate-400">
                            Review active and recent SRE Smart Bot incidents, inspect evidence, and follow the action and approval trail.
                        </p>
                    </div>
                    <div className="flex items-center gap-3">
                        <Link
                            to="/admin/operations/sre-smart-bot/approvals"
                            className="rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                        >
                            Approvals
                        </Link>
                        <Link
                            to="/admin/operations/sre-smart-bot/settings"
                            className="rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                        >
                            Settings
                        </Link>
                        <div className="rounded-2xl border border-slate-200 bg-white/80 px-4 py-3 text-sm shadow-sm dark:border-slate-800 dark:bg-slate-900/70">
                            <div className="text-slate-500 dark:text-slate-400">Incident threads</div>
                            <div className="mt-1 text-2xl font-semibold text-slate-900 dark:text-white">{total}</div>
                        </div>
                        <button
                            onClick={() => void loadIncidents()}
                            className="rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                        >
                            Refresh
                        </button>
                    </div>
                </div>

                <SectionCard
                    title="Demo Scenarios"
                    subtitle="Generate realistic SRE Smart Bot incidents on demand so you can demo grounded investigation, AI interpretation, and approval-safe action flow."
                >
                    <div className="grid gap-4 xl:grid-cols-[minmax(0,1.4fr)_minmax(320px,0.9fr)]">
                        <div className="space-y-2">
                            <label className="space-y-2">
                                <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Scenario</span>
                                <select
                                    value={selectedDemoScenarioId}
                                    onChange={(e) => setSelectedDemoScenarioId(e.target.value)}
                                    disabled={demoLoading || generatingDemo}
                                    className="w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 disabled:cursor-not-allowed disabled:opacity-70 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900"
                                >
                                    {demoScenarios.map((scenario) => (
                                        <option key={scenario.id} value={scenario.id}>
                                            {scenario.name}
                                        </option>
                                    ))}
                                </select>
                            </label>
                            <div className="flex flex-wrap items-center gap-3">
                                <button
                                    onClick={() => void handleGenerateDemoIncident()}
                                    disabled={demoLoading || generatingDemo || !selectedDemoScenarioId}
                                    className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-70 dark:bg-sky-500 dark:hover:bg-sky-400"
                                >
                                    {generatingDemo ? 'Generating...' : 'Generate Demo Incident'}
                                </button>
                                <p className="text-xs text-slate-500 dark:text-slate-400">
                                    Best flow: generate, open the incident, run MCP tools, then show draft and local interpretation.
                                </p>
                            </div>
                        </div>
                        <div className="rounded-2xl border border-slate-200 bg-slate-50/90 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                            {selectedDemoScenario ? (
                                <>
                                    <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">{selectedDemoScenario.name}</h3>
                                    <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">{selectedDemoScenario.summary}</p>
                                    <div className="mt-4 rounded-xl border border-slate-200 bg-white/80 p-3 dark:border-slate-800 dark:bg-slate-900/70">
                                        <p className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">Suggested Walkthrough</p>
                                        <p className="mt-2 text-sm text-slate-700 dark:text-slate-300">{selectedDemoScenario.recommended_walkthrough}</p>
                                    </div>
                                </>
                            ) : (
                                <EmptyState title="No demo scenarios available" description="The backend demo generator is not exposing any scenarios yet." />
                            )}
                        </div>
                    </div>
                </SectionCard>

                <SectionCard title="Filters" subtitle="Use narrow filters for incident review, or search by summary, type, or source.">
                    <div className="grid gap-4 md:grid-cols-2 2xl:grid-cols-5">
                        <label className="space-y-2">
                            <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Search</span>
                            <input
                                value={search}
                                onChange={(e) => setSearch(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter') {
                                        void loadIncidents()
                                    }
                                }}
                                placeholder="incident, source, summary..."
                                className="w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900"
                            />
                        </label>
                        <label className="space-y-2">
                            <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Status</span>
                            <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className="w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900">
                                <option value="">All statuses</option>
                                <option value="observed">Observed</option>
                                <option value="triaged">Triaged</option>
                                <option value="contained">Contained</option>
                                <option value="recovering">Recovering</option>
                                <option value="resolved">Resolved</option>
                                <option value="suppressed">Suppressed</option>
                                <option value="escalated">Escalated</option>
                            </select>
                        </label>
                        <label className="space-y-2">
                            <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Severity</span>
                            <select value={severityFilter} onChange={(e) => setSeverityFilter(e.target.value)} className="w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900">
                                <option value="">All severities</option>
                                <option value="info">Info</option>
                                <option value="warning">Warning</option>
                                <option value="critical">Critical</option>
                            </select>
                        </label>
                        <label className="space-y-2">
                            <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Domain</span>
                            <select value={domainFilter} onChange={(e) => setDomainFilter(e.target.value)} className="w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900">
                                <option value="">All domains</option>
                                {uniqueDomains.map((domain) => (
                                    <option key={domain} value={domain}>{domain}</option>
                                ))}
                            </select>
                        </label>
                        <div className="flex items-end">
                            <button
                                onClick={() => void loadIncidents()}
                                className="w-full rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-sky-700 dark:bg-sky-500 dark:hover:bg-sky-400"
                            >
                                Apply Filters
                            </button>
                        </div>
                    </div>
                </SectionCard>

                <SectionCard title="Incident Ledger" subtitle="Newest incidents first. Select a row to inspect findings, evidence, actions, and approvals.">
                    {loading ? (
                        <div className="flex items-center justify-center rounded-xl border border-slate-200 bg-slate-50 px-4 py-16 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-900/60 dark:text-slate-300">
                            Loading SRE incidents...
                        </div>
                    ) : error ? (
                        <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-4 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/40 dark:text-rose-200">{error}</div>
                    ) : incidents.length === 0 ? (
                        <EmptyState title="No incidents found" description="Try widening the filters or wait for SRE Smart Bot signal ingestion to create the first incident threads." />
                    ) : (
                        <div className="space-y-4">
                            <div className="grid gap-3 md:hidden">
                                {incidents.map((incident) => (
                                    <button
                                        key={incident.id}
                                        type="button"
                                        className="w-full rounded-2xl border border-slate-200 bg-white/90 p-4 text-left shadow-sm transition hover:border-sky-300 hover:bg-sky-50/50 dark:border-slate-800 dark:bg-slate-950/60 dark:hover:border-sky-700 dark:hover:bg-sky-950/20"
                                        onClick={() => void openIncident(incident)}
                                    >
                                        <div className="flex flex-wrap items-start justify-between gap-3">
                                            <div className="min-w-0 flex-1">
                                                <div className="font-medium text-slate-900 dark:text-white">{incident.display_name}</div>
                                                <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">{incident.summary || incident.incident_type}</div>
                                            </div>
                                            <span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${severityTone[incident.severity]}`}>{incident.severity}</span>
                                        </div>
                                        <div className="mt-3 flex flex-wrap gap-2">
                                            <span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${statusTone[incident.status]}`}>{incident.status}</span>
                                            <span className="inline-flex rounded-full border border-slate-300 px-2.5 py-1 text-xs font-medium text-slate-700 dark:border-slate-700 dark:text-slate-300">{incident.domain}</span>
                                        </div>
                                        <div className="mt-3 grid gap-2 text-xs text-slate-500 dark:text-slate-500 sm:grid-cols-2">
                                            <div>Source: {incident.source || '—'}</div>
                                            <div>Last observed: {relativeTime(incident.last_observed_at)}</div>
                                        </div>
                                    </button>
                                ))}
                            </div>

                            <div className="hidden overflow-hidden rounded-2xl border border-slate-200 dark:border-slate-800 md:block">
                                <div className="overflow-x-auto">
                                <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-800">
                                    <thead className="bg-slate-100/90 dark:bg-slate-900/90">
                                        <tr className="text-left text-xs uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">
                                            <th className="px-4 py-3">Incident</th>
                                            <th className="px-4 py-3">Domain</th>
                                            <th className="px-4 py-3">Severity</th>
                                            <th className="px-4 py-3">Status</th>
                                            <th className="px-4 py-3">Source</th>
                                            <th className="px-4 py-3">Last Observed</th>
                                        </tr>
                                    </thead>
                                    <tbody className="divide-y divide-slate-200 bg-white dark:divide-slate-800 dark:bg-slate-950/50">
                                        {incidents.map((incident) => (
                                            <tr key={incident.id} className="cursor-pointer transition hover:bg-sky-50/60 dark:hover:bg-sky-950/20" onClick={() => void openIncident(incident)}>
                                                <td className="px-4 py-4 align-top">
                                                    <div className="max-w-xl">
                                                        <div className="font-medium text-slate-900 dark:text-white">{incident.display_name}</div>
                                                        <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">{incident.summary || incident.incident_type}</div>
                                                        <div className="mt-2 text-xs text-slate-500 dark:text-slate-500">{incident.incident_type}</div>
                                                    </div>
                                                </td>
                                                <td className="px-4 py-4 text-sm text-slate-700 dark:text-slate-300">{incident.domain}</td>
                                                <td className="px-4 py-4">
                                                    <span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${severityTone[incident.severity]}`}>{incident.severity}</span>
                                                </td>
                                                <td className="px-4 py-4">
                                                    <span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${statusTone[incident.status]}`}>{incident.status}</span>
                                                </td>
                                                <td className="px-4 py-4 text-sm text-slate-700 dark:text-slate-300">{incident.source}</td>
                                                <td className="px-4 py-4 text-sm text-slate-700 dark:text-slate-300">
                                                    <div>{relativeTime(incident.last_observed_at)}</div>
                                                    <div className="mt-1 text-xs text-slate-500 dark:text-slate-500">{formatDateTime(incident.last_observed_at)}</div>
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                        </div>
                    )}
                </SectionCard>
            </div>

            <Drawer
                isOpen={drawerOpen}
                onClose={closeDrawer}
                title={selectedIncident?.display_name || 'Incident Detail'}
                description={selectedIncident ? `${selectedIncident.domain} • ${selectedIncident.incident_type}` : undefined}
                width="60vw"
            >
                {drawerLoading ? (
                    <div className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-10 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-900/60 dark:text-slate-300">Loading incident details...</div>
                ) : drawerError ? (
                    <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-4 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/40 dark:text-rose-200">{drawerError}</div>
                ) : selectedIncident ? (
                    <div className="space-y-6">
                        <div className="rounded-2xl border border-slate-200 bg-white/90 shadow-sm dark:border-slate-800 dark:bg-slate-900/85">
                            <div
                                role="tablist"
                                aria-label="Incident detail sections"
                                className="flex flex-wrap gap-x-1 gap-y-2 border-b border-slate-200 px-3 pt-3 dark:border-slate-800"
                            >
                                {drawerTabs.map((tab) => {
                                    const active = drawerTab === tab.id
                                    return (
                                        <button
                                            key={tab.id}
                                            type="button"
                                            role="tab"
                                            id={`incident-tab-${tab.id}`}
                                            aria-selected={active}
                                            aria-controls={`incident-panel-${tab.id}`}
                                            onClick={() => setDrawerTab(tab.id)}
                                            className={`rounded-t-xl border-b-2 px-3 py-3 text-left transition focus:outline-none focus:ring-2 focus:ring-sky-200 dark:focus:ring-sky-900 ${
                                                active
                                                    ? 'border-sky-500 bg-sky-50/80 text-sky-800 dark:border-sky-400 dark:bg-sky-950/30 dark:text-sky-200'
                                                    : 'border-transparent bg-transparent text-slate-600 hover:border-slate-300 hover:text-slate-900 dark:text-slate-400 dark:hover:border-slate-700 dark:hover:text-slate-100'
                                            }`}
                                        >
                                            <div className="text-sm font-medium">{tab.label}</div>
                                        </button>
                                    )
                                })}
                            </div>
                        </div>

                        {drawerTab === 'summary' ? (
                        <>
                        <div role="tabpanel" id="incident-panel-summary" aria-labelledby="incident-tab-summary" className="space-y-6">
                        <div className="rounded-2xl border border-slate-200 bg-slate-50/90 px-4 py-3 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300">
                            {drawerTabs.find((tab) => tab.id === 'summary')?.hint}
                        </div>
                        <SectionCard title="Operator Snapshot" subtitle="A quick operational read on evidence volume, pending action proposals, and approval activity.">
                            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
                                <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Findings</div>
                                    <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{findings.length}</div>
                                    <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Signals recorded for this incident thread.</div>
                                </div>
                                <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Evidence</div>
                                    <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{evidence.length}</div>
                                    <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Stored snapshots and watcher summaries.</div>
                                </div>
                                <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Action Proposals</div>
                                    <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{actions.length}</div>
                                    <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">{actions.filter((action) => action.approval_required).length} require approval.</div>
                                </div>
                                <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Approvals</div>
                                    <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{approvals.length}</div>
                                    <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">{approvals.filter((approval) => !approval.decided_at).length} still awaiting decision.</div>
                                </div>
                                <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Summary Emails</div>
                                    <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{summaryDeliveryStats.successful}</div>
                                    <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">{summaryDeliveryStats.total} queued or attempted so far.</div>
                                </div>
                            </div>
                        </SectionCard>

                        {httpSignalsRecentTool || httpSignalsHistoryTool ? (
                            <SectionCard
                                title="App Golden Signals"
                                subtitle="Latest HTTP request health and retained trend context, surfaced directly in the incident summary."
                            >
                                {httpSignalsRecentResult || httpSignalsHistoryResult ? (
                                    <div className="space-y-4">
                                        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                                            <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Latest Requests</div>
                                                <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{httpSignalsSummary.recentRequestCount}</div>
                                                <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Most recent captured HTTP window.</div>
                                            </div>
                                            <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 dark:border-rose-900/40 dark:bg-rose-950/30">
                                                <div className="text-xs uppercase tracking-wide text-rose-700 dark:text-rose-300">Latest Error Rate</div>
                                                <div className="mt-2 text-2xl font-semibold text-rose-900 dark:text-rose-100">{httpSignalsSummary.recentErrorRatePercent}%</div>
                                                <div className="mt-1 text-sm text-rose-700/80 dark:text-rose-200/80">Server-side pressure in the latest window.</div>
                                            </div>
                                            <div className="rounded-xl border border-cyan-200 bg-cyan-50 p-4 dark:border-cyan-900/40 dark:bg-cyan-950/30">
                                                <div className="text-xs uppercase tracking-wide text-cyan-700 dark:text-cyan-300">Latest Avg Latency</div>
                                                <div className="mt-2 text-2xl font-semibold text-cyan-900 dark:text-cyan-100">{httpSignalsSummary.recentAverageLatencyMs}ms</div>
                                                <div className="mt-1 text-sm text-cyan-700/80 dark:text-cyan-200/80">Average request duration for the latest slice.</div>
                                            </div>
                                            <div className="rounded-xl border border-violet-200 bg-violet-50 p-4 dark:border-violet-900/40 dark:bg-violet-950/30">
                                                <div className="text-xs uppercase tracking-wide text-violet-700 dark:text-violet-300">Trend Direction</div>
                                                <div className="mt-2 text-2xl font-semibold capitalize text-violet-900 dark:text-violet-100">
                                                    {prettifyTrendLabel(httpSignalsSummary.historyTrend)}
                                                </div>
                                                <div className="mt-1 text-sm text-violet-700/80 dark:text-violet-200/80">
                                                    {httpSignalsSummary.historyWindowCount > 0
                                                        ? `${httpSignalsSummary.historyWindowCount} retained windows available.`
                                                        : 'History has not been captured yet.'}
                                                </div>
                                            </div>
                                        </div>

                                        {httpSignalsSummary.historyWindowCount > 0 ? (
                                            <div className="grid gap-4 md:grid-cols-3">
                                                <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                                                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">History Avg Latency</div>
                                                    <div className="mt-2 text-lg font-semibold text-slate-900 dark:text-slate-100">{httpSignalsSummary.historyAverageLatencyMs}ms</div>
                                                </div>
                                                <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                                                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">History Avg Error Rate</div>
                                                    <div className="mt-2 text-lg font-semibold text-slate-900 dark:text-slate-100">{httpSignalsSummary.historyAverageErrorRatePercent}%</div>
                                                </div>
                                                <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                                                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Peak Traffic</div>
                                                    <div className="mt-2 text-lg font-semibold text-slate-900 dark:text-slate-100">{httpSignalsSummary.historyPeakRequestCount}</div>
                                                    <div className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                                        Latest history sample: {httpSignalsSummary.latestHistoryWindowEndedAt ? formatDateTime(httpSignalsSummary.latestHistoryWindowEndedAt) : '—'}
                                                    </div>
                                                </div>
                                            </div>
                                        ) : null}
                                    </div>
                                ) : (
                                    <EmptyState
                                        title="HTTP signal context is available but not loaded yet"
                                        description="This incident has HTTP golden-signal tooling enabled. Open the AI workspace tools if you want to inspect the raw MCP output directly."
                                    />
                                )}
                            </SectionCard>
                        ) : null}

                        {asyncBacklogTool && selectedIncident.incident_type === 'backlog_pressure' ? (
                            <SectionCard
                                title="Async Backlog Pressure"
                                subtitle="Current queue and relay pressure across background processing paths."
                            >
                                {asyncBacklogResult ? (
                                    <div className="space-y-4">
                                        <div className="grid gap-4 md:grid-cols-3">
                                            <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-900/40 dark:bg-amber-950/30">
                                                <div className="text-xs uppercase tracking-wide text-amber-700 dark:text-amber-300">Build Queue</div>
                                                <div className="mt-2 text-2xl font-semibold text-amber-900 dark:text-amber-100">{asyncBacklogSummary.buildQueueDepth}</div>
                                                <div className="mt-1 text-sm text-amber-700/80 dark:text-amber-200/80">Threshold {asyncBacklogSummary.buildQueueThreshold}</div>
                                            </div>
                                            <div className="rounded-xl border border-sky-200 bg-sky-50 p-4 dark:border-sky-900/40 dark:bg-sky-950/30">
                                                <div className="text-xs uppercase tracking-wide text-sky-700 dark:text-sky-300">Email Queue</div>
                                                <div className="mt-2 text-2xl font-semibold text-sky-900 dark:text-sky-100">{asyncBacklogSummary.emailQueueDepth}</div>
                                                <div className="mt-1 text-sm text-sky-700/80 dark:text-sky-200/80">Threshold {asyncBacklogSummary.emailQueueThreshold}</div>
                                            </div>
                                            <div className="rounded-xl border border-fuchsia-200 bg-fuchsia-50 p-4 dark:border-fuchsia-900/40 dark:bg-fuchsia-950/30">
                                                <div className="text-xs uppercase tracking-wide text-fuchsia-700 dark:text-fuchsia-300">Outbox Pending</div>
                                                <div className="mt-2 text-2xl font-semibold text-fuchsia-900 dark:text-fuchsia-100">{asyncBacklogSummary.messagingOutboxPending}</div>
                                                <div className="mt-1 text-sm text-fuchsia-700/80 dark:text-fuchsia-200/80">Threshold {asyncBacklogSummary.messagingOutboxThreshold}</div>
                                            </div>
                                        </div>
                                        <div className="rounded-xl border border-slate-200 bg-white/90 p-4 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900/70 dark:text-slate-300">
                                            Latest snapshot: {asyncBacklogSummary.lastActivity ? formatDateTime(asyncBacklogSummary.lastActivity) : '—'}
                                        </div>
                                    </div>
                                ) : (
                                    <EmptyState
                                        title="Async backlog context is available but not loaded yet"
                                        description="This backlog-pressure incident has async backlog tooling enabled. Open the AI workspace if you want to inspect the raw MCP output directly."
                                    />
                                )}
                            </SectionCard>
                        ) : null}

                        {messagingTransportTool && selectedIncident.incident_type === 'messaging_transport_degraded' ? (
                            <SectionCard
                                title="Messaging Transport Health"
                                subtitle="Current NATS transport stability for reconnect and disconnect pressure."
                            >
                                {messagingTransportResult ? (
                                    <div className="space-y-4">
                                        <div className="grid gap-4 md:grid-cols-3">
                                            <div className="rounded-xl border border-cyan-200 bg-cyan-50 p-4 dark:border-cyan-900/40 dark:bg-cyan-950/30">
                                                <div className="text-xs uppercase tracking-wide text-cyan-700 dark:text-cyan-300">Reconnects</div>
                                                <div className="mt-2 text-2xl font-semibold text-cyan-900 dark:text-cyan-100">{messagingTransportSummary.reconnects}</div>
                                                <div className="mt-1 text-sm text-cyan-700/80 dark:text-cyan-200/80">Threshold {messagingTransportSummary.reconnectThreshold}</div>
                                            </div>
                                            <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 dark:border-rose-900/40 dark:bg-rose-950/30">
                                                <div className="text-xs uppercase tracking-wide text-rose-700 dark:text-rose-300">Disconnects</div>
                                                <div className="mt-2 text-2xl font-semibold text-rose-900 dark:text-rose-100">{messagingTransportSummary.disconnects}</div>
                                                <div className="mt-1 text-sm text-rose-700/80 dark:text-rose-200/80">Recent transport interruptions</div>
                                            </div>
                                            <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Latest Snapshot</div>
                                                <div className="mt-2 text-sm font-semibold text-slate-900 dark:text-slate-100">
                                                    {messagingTransportSummary.lastActivity ? formatDateTime(messagingTransportSummary.lastActivity) : '—'}
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                ) : (
                                    <EmptyState
                                        title="Messaging transport context is available but not loaded yet"
                                        description="This incident has messaging transport tooling enabled. Open the AI workspace if you want to inspect the raw MCP output directly."
                                    />
                                )}
                            </SectionCard>
                        ) : null}

                        <SectionCard title="Executive Summary" subtitle="A concise operator-ready narrative you can use in triage, handoffs, and email updates.">
                            <div className="space-y-3">
                                {executiveSummary.map((item, index) => (
                                    <div key={`${index}-${item}`} className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-950 dark:text-slate-300">
                                        {item}
                                    </div>
                                ))}
                            </div>
                            <div className="mt-4 flex flex-wrap gap-2">
                                <button
                                    type="button"
                                    onClick={() => void handleEmailSummary()}
                                    disabled={emailingSummary}
                                    className="rounded-lg border border-sky-300 bg-sky-50 px-3 py-1.5 text-xs font-medium text-sky-700 transition hover:bg-sky-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-sky-800 dark:bg-sky-950/30 dark:text-sky-200 dark:hover:bg-sky-950/50"
                                >
                                    {emailingSummary ? 'Queueing email...' : 'Email Summary to Admins'}
                                </button>
                            </div>
                        </SectionCard>

                        <SectionCard title="Summary Delivery" subtitle="Email-summary history for this incident, including the latest delivery attempt and recipients.">
                            {summaryEmailActions.length === 0 ? (
                                <EmptyState title="No summary emails sent yet" description="Use the executive summary action above to queue a summary email for admins." />
                            ) : (
                                <div className="space-y-3">
                                    {summaryEmailActions.map((action) => {
                                        const payload = asRecord(action.result_payload)
                                        const recipients = Array.isArray(payload?.recipients) ? payload.recipients as string[] : []
                                        const sentCount = typeof payload?.sent_count === 'number' ? payload.sent_count : 0
                                        const recipientCount = typeof payload?.recipient_count === 'number' ? payload.recipient_count : recipients.length
                                        return (
                                            <div key={action.id} className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                <div className="flex flex-wrap items-center justify-between gap-3">
                                                    <div className="flex flex-wrap items-center gap-2">
                                                        <div className="font-medium text-slate-900 dark:text-white">Admin summary email</div>
                                                        <span className={`rounded-full border px-2 py-0.5 text-[11px] font-semibold ${actionTone(action.status)}`}>{action.status}</span>
                                                    </div>
                                                    <div className="text-xs text-slate-500 dark:text-slate-400">{formatDateTime(action.completed_at || action.requested_at)}</div>
                                                </div>
                                                <div className="mt-3 grid gap-3 text-sm md:grid-cols-3">
                                                    <div className="rounded-lg border border-slate-200 bg-white px-3 py-3 dark:border-slate-800 dark:bg-slate-900">
                                                        <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Sent</div>
                                                        <div className="mt-1 font-medium text-slate-900 dark:text-slate-100">{sentCount} / {recipientCount}</div>
                                                    </div>
                                                    <div className="rounded-lg border border-slate-200 bg-white px-3 py-3 dark:border-slate-800 dark:bg-slate-900">
                                                        <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Requested by</div>
                                                        <div className="mt-1 font-medium text-slate-900 dark:text-slate-100">{action.actor_id || 'system'}</div>
                                                    </div>
                                                    <div className="rounded-lg border border-slate-200 bg-white px-3 py-3 dark:border-slate-800 dark:bg-slate-900">
                                                        <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Last activity</div>
                                                        <div className="mt-1 font-medium text-slate-900 dark:text-slate-100">{relativeTime(action.completed_at || action.requested_at)}</div>
                                                    </div>
                                                </div>
                                                {recipients.length > 0 ? (
                                                    <div className="mt-3">
                                                        <div className="mb-2 text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Recipients</div>
                                                        <div className="flex flex-wrap gap-2">
                                                            {recipients.map((recipient) => (
                                                                <span key={`${action.id}-${recipient}`} className="inline-flex rounded-full border border-slate-300 bg-white px-2.5 py-1 text-xs font-medium text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">
                                                                    {recipient}
                                                                </span>
                                                            ))}
                                                        </div>
                                                    </div>
                                                ) : null}
                                                {action.error_message ? (
                                                    <div className="mt-3 text-sm font-medium text-rose-700 dark:text-rose-300">{action.error_message}</div>
                                                ) : null}
                                            </div>
                                        )
                                    })}
                                </div>
                            )}
                        </SectionCard>

                        <SectionCard title="Overview">
                            <div className="grid gap-4 md:grid-cols-2">
                                <div>
                                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Status</div>
                                    <div className="mt-2"><span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${statusTone[selectedIncident.status]}`}>{selectedIncident.status}</span></div>
                                </div>
                                <div>
                                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Severity</div>
                                    <div className="mt-2"><span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${severityTone[selectedIncident.severity]}`}>{selectedIncident.severity}</span></div>
                                </div>
                                <div className="text-sm text-slate-700 dark:text-slate-300"><span className="font-medium text-slate-900 dark:text-white">Source:</span> {selectedIncident.source}</div>
                                <div className="text-sm text-slate-700 dark:text-slate-300"><span className="font-medium text-slate-900 dark:text-white">Confidence:</span> {selectedIncident.confidence}</div>
                                <div className="text-sm text-slate-700 dark:text-slate-300"><span className="font-medium text-slate-900 dark:text-white">First observed:</span> {formatDateTime(selectedIncident.first_observed_at)}</div>
                                <div className="text-sm text-slate-700 dark:text-slate-300"><span className="font-medium text-slate-900 dark:text-white">Last observed:</span> {formatDateTime(selectedIncident.last_observed_at)}</div>
                            </div>
                            <div className="mt-4 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-950 dark:text-slate-300">
                                {selectedIncident.summary || 'No summary provided.'}
                            </div>
                            {selectedIncident.metadata ? (
                                <div className="mt-4">
                                    <div className="mb-2 text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Metadata</div>
                                    <pre className="overflow-x-auto rounded-xl border border-slate-200 bg-slate-50 p-4 text-xs text-slate-700 dark:border-slate-800 dark:bg-slate-950 dark:text-slate-300">{prettyJson(selectedIncident.metadata)}</pre>
                                </div>
                            ) : null}
                        </SectionCard>
                        </div>
                        </>
                        ) : null}

                        {drawerTab === 'signals' ? (
                        <>
                        <div role="tabpanel" id="incident-panel-signals" aria-labelledby="incident-tab-signals" className="space-y-6">
                        <div className="rounded-2xl border border-slate-200 bg-slate-50/90 px-4 py-3 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300">
                            {drawerTabs.find((tab) => tab.id === 'signals')?.hint}
                        </div>
                        {evidence.length === 0 ? (
                            <div className="rounded-2xl border border-amber-200 bg-amber-50/90 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200">
                                No evidence snapshots are stored for this incident yet. Findings were recorded, but this thread does not currently have additional evidence rows attached.
                            </div>
                        ) : null}
                        <SectionCard title="Findings" subtitle={`${findings.length} signal observations`}>
                            {findings.length === 0 ? <EmptyState title="No findings recorded" description="This incident has not stored any finding rows yet." /> : (
                                <div className="space-y-3">
                                    {findings.map((finding) => (
                                        <div key={finding.id} className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                            <div className="flex flex-wrap items-center gap-2">
                                                <div className="font-medium text-slate-900 dark:text-white">{finding.title}</div>
                                                <span className={`inline-flex rounded-full border px-2 py-0.5 text-[11px] font-semibold ${severityTone[finding.severity]}`}>{finding.severity}</span>
                                            </div>
                                            <p className="mt-2 text-sm text-slate-700 dark:text-slate-300">{finding.message}</p>
                                            <div className="mt-2 text-xs text-slate-500 dark:text-slate-500">{finding.signal_type} • {finding.signal_key} • {formatDateTime(finding.occurred_at)}</div>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </SectionCard>

                        <SectionCard title="Evidence" subtitle={`${evidence.length} evidence records`}>
                            <div className="mb-4 flex flex-wrap gap-2">
                                <button
                                    type="button"
                                    onClick={() => void handleProposeDetectorRule()}
                                    disabled={proposingDetectorRule}
                                    className="rounded-lg border border-sky-300 bg-sky-50 px-3 py-1.5 text-xs font-medium text-sky-700 transition hover:bg-sky-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-sky-800 dark:bg-sky-950/30 dark:text-sky-200 dark:hover:bg-sky-950/50"
                                >
                                    {proposingDetectorRule ? 'Proposing...' : 'Propose Detector Rule'}
                                </button>
                                <Link
                                    to={selectedIncident ? `/admin/operations/sre-smart-bot/detector-rules?incident=${encodeURIComponent(selectedIncident.id)}` : '/admin/operations/sre-smart-bot/detector-rules'}
                                    className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                                >
                                    Open Detector Rules
                                </Link>
                            </div>
                            {evidence.length === 0 ? <EmptyState title="No evidence recorded" description="Evidence capture is ready but this incident does not yet have stored evidence rows." /> : (
                                <div className="space-y-3">
                                    {evidence.map((item) => (
                                        <div key={item.id} className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                            <div className="font-medium text-slate-900 dark:text-white">{item.evidence_type}</div>
                                            <p className="mt-2 text-sm text-slate-700 dark:text-slate-300">{item.summary}</p>
                                            {item.payload ? <pre className="mt-3 overflow-x-auto rounded-lg border border-slate-200 bg-white p-3 text-xs text-slate-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300">{prettyJson(item.payload)}</pre> : null}
                                        </div>
                                    ))}
                                </div>
                            )}
                        </SectionCard>
                        </div>
                        </>
                        ) : null}

                        {drawerTab === 'actions' ? (
                        <>
                        <div role="tabpanel" id="incident-panel-actions" aria-labelledby="incident-tab-actions" className="space-y-6">
                        <div className="rounded-2xl border border-slate-200 bg-slate-50/90 px-4 py-3 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300">
                            {drawerTabs.find((tab) => tab.id === 'actions')?.hint}
                        </div>
                        <SectionCard title="Actions" subtitle={`${actions.length} action attempts`}>
                            {actions.length === 0 ? <EmptyState title="No actions attempted" description="Remediation action attempts will appear here once the policy engine starts executing or requesting actions." /> : (
                                <div className="space-y-3">
                                    {actions.map((action) => (
                                        <div key={action.id} className={`rounded-xl border p-4 ${actionTone(action.status)}`}>
                                            <div className="flex flex-wrap items-center justify-between gap-3">
                                                <div className="flex flex-wrap items-center gap-2">
                                                    <div className="font-medium">{action.action_key}</div>
                                                    <span className="rounded-full border border-current/20 px-2 py-0.5 text-[11px] font-semibold">{action.status}</span>
                                                    {action.approval_required ? (
                                                        <span className="rounded-full border border-current/20 px-2 py-0.5 text-[11px] font-semibold">approval required</span>
                                                    ) : null}
                                                </div>
                                                <div className="text-xs opacity-80">{formatDateTime(action.requested_at)}</div>
                                            </div>
                                            <div className="mt-3 grid gap-3 text-sm md:grid-cols-2">
                                                <div>
                                                    <div className="text-xs uppercase tracking-wide opacity-70">Target</div>
                                                    <div className="mt-1 font-medium">{action.target_kind}: {action.target_ref || '—'}</div>
                                                </div>
                                                <div>
                                                    <div className="text-xs uppercase tracking-wide opacity-70">Actor</div>
                                                    <div className="mt-1 font-medium">{action.actor_type}{action.actor_id ? ` • ${action.actor_id}` : ''}</div>
                                                </div>
                                            </div>
                                            {action.result_payload ? (
                                                <div className="mt-3 rounded-lg border border-current/15 bg-white/50 p-3 text-xs text-slate-700 dark:bg-slate-950/40 dark:text-slate-200">
                                                    <div className="mb-2 uppercase tracking-wide opacity-70">Stored rationale</div>
                                                    <pre className="overflow-x-auto whitespace-pre-wrap">{prettyJson(action.result_payload)}</pre>
                                                </div>
                                            ) : null}
                                            <div className="mt-3 flex flex-wrap gap-2">
                                                {!action.approval_required && (action.status || '').toLowerCase() === 'proposed' ? (
                                                    <button
                                                        type="button"
                                                        onClick={() => void handleRequestApproval(action)}
                                                        disabled={mutatingActionId === action.id}
                                                        className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                                                    >
                                                        {mutatingActionId === action.id ? 'Requesting...' : 'Request Approval'}
                                                    </button>
                                                ) : null}
                                                {canExecuteAction(action) ? (
                                                    <button
                                                        type="button"
                                                        onClick={() => void handleExecuteAction(action)}
                                                        disabled={mutatingActionId === action.id}
                                                        className="rounded-lg bg-sky-600 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-sky-500 dark:hover:bg-sky-400"
                                                    >
                                                        {mutatingActionId === action.id ? 'Executing...' : 'Execute'}
                                                    </button>
                                                ) : null}
                                            </div>
                                            {action.error_message ? <div className="mt-3 text-sm font-medium text-rose-700 dark:text-rose-300">{action.error_message}</div> : null}
                                        </div>
                                    ))}
                                </div>
                            )}
                        </SectionCard>

                        <SectionCard title="Approvals" subtitle={`${approvals.length} approval records`}>
                            {approvals.length === 0 ? <EmptyState title="No approvals recorded" description="Approval requests will appear here once SRE Smart Bot starts issuing channel or in-app approvals." /> : (
                                <div className="space-y-3">
                                    {approvals.map((approval) => (
                                        <div key={approval.id} className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                            <div className="flex flex-wrap items-center justify-between gap-3">
                                                <div className="font-medium text-slate-900 dark:text-white">
                                                    {approval.action_attempt_id && actionsById[approval.action_attempt_id]
                                                        ? actionsById[approval.action_attempt_id].action_key
                                                        : approval.channel_provider_id}
                                                </div>
                                                <span className={`rounded-full border px-2.5 py-1 text-[11px] font-semibold ${approval.decided_at ? 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-200' : 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200'}`}>{approval.status}</span>
                                            </div>
                                            <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-slate-500">
                                                <span className="rounded-full border border-slate-300 px-2 py-0.5 dark:border-slate-700">{approval.channel_provider_id}</span>
                                                {approval.action_attempt_id ? <span className="rounded-full border border-slate-300 px-2 py-0.5 dark:border-slate-700">linked action</span> : null}
                                            </div>
                                            <p className="mt-2 text-sm text-slate-700 dark:text-slate-300">{approval.request_message}</p>
                                            <div className="mt-2 text-xs text-slate-500 dark:text-slate-500">
                                                Requested {formatDateTime(approval.requested_at)}
                                                {approval.decided_at ? ` • decided ${formatDateTime(approval.decided_at)}` : ''}
                                                {approval.expires_at ? ` • expires ${formatDateTime(approval.expires_at)}` : ''}
                                            </div>
                                            {!approval.decided_at ? (
                                                <div className="mt-3 flex flex-wrap gap-2">
                                                    <button
                                                        type="button"
                                                        onClick={() => void handleApprovalDecision(approval, 'approved')}
                                                        disabled={mutatingApprovalId === approval.id}
                                                        className="rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-emerald-500 dark:hover:bg-emerald-400"
                                                    >
                                                        {mutatingApprovalId === approval.id ? 'Updating...' : 'Approve'}
                                                    </button>
                                                    <button
                                                        type="button"
                                                        onClick={() => void handleApprovalDecision(approval, 'rejected')}
                                                        disabled={mutatingApprovalId === approval.id}
                                                        className="rounded-lg border border-rose-300 bg-white px-3 py-1.5 text-xs font-medium text-rose-700 transition hover:bg-rose-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-rose-800 dark:bg-slate-950 dark:text-rose-300 dark:hover:bg-rose-950/40"
                                                    >
                                                        {mutatingApprovalId === approval.id ? 'Updating...' : 'Reject'}
                                                    </button>
                                                </div>
                                            ) : null}
                                            {approval.decision_comment ? <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">Comment: {approval.decision_comment}</div> : null}
                                        </div>
                                    ))}
                                </div>
                            )}
                        </SectionCard>
                        </div>
                        </>
                        ) : null}

                        {drawerTab === 'ai' ? (
                        <>
                        <div role="tabpanel" id="incident-panel-ai" aria-labelledby="incident-tab-ai" className="space-y-6">
                        <div className="rounded-2xl border border-slate-200 bg-slate-50/90 px-4 py-3 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300">
                            {drawerTabs.find((tab) => tab.id === 'ai')?.hint}
                        </div>
                        <SectionCard title="AI Workspace" subtitle="Structured context for the future MCP and agent-runtime layer, built from the incident ledger and current SRE policy.">
                            {!workspace ? (
                                <EmptyState title="Workspace not available yet" description="The MCP and AI workspace bundle has not loaded for this incident yet." />
                            ) : (
                                <div className="space-y-5">
                                    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                                        <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                            <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Agent Runtime</div>
                                            <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">
                                                {workspace.agent_runtime.enabled ? 'Enabled' : 'Disabled'}
                                            </div>
                                            <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">
                                                {workspace.agent_runtime.provider || 'custom'}
                                                {workspace.agent_runtime.model ? ` • ${workspace.agent_runtime.model}` : ''}
                                            </div>
                                        </div>
                                        <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                            <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Enabled MCP Servers</div>
                                            <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{workspace.enabled_mcp_servers.length}</div>
                                            <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Bounded tool boundaries available under current policy.</div>
                                        </div>
                                        <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                            <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Recommended Questions</div>
                                            <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{workspace.recommended_questions.length}</div>
                                            <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Prompt-ready investigation angles grounded in stored evidence.</div>
                                        </div>
                                    </div>

                                    <div className="grid gap-5 xl:grid-cols-2">
                                        <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                            <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">AI Executive Summary</h4>
                                            <div className="mt-3 space-y-2">
                                                {workspace.executive_summary.map((line, index) => (
                                                    <div key={`${line}-${index}`} className="rounded-xl border border-slate-200 bg-white/90 px-3 py-2 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900/70 dark:text-slate-300">
                                                        {line}
                                                    </div>
                                                ))}
                                            </div>
                                        </div>

                                        <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                            <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Recommended Questions</h4>
                                            <div className="mt-3 space-y-2">
                                                {workspace.recommended_questions.map((line, index) => (
                                                    <div key={`${line}-${index}`} className="rounded-xl border border-slate-200 bg-white/90 px-3 py-2 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900/70 dark:text-slate-300">
                                                        {line}
                                                    </div>
                                                ))}
                                            </div>
                                        </div>
                                    </div>

                                    <div className="grid gap-5 xl:grid-cols-2">
                                        <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                            <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Suggested Tooling</h4>
                                            {workspace.suggested_tooling.length === 0 ? (
                                                <EmptyState title="No tooling guidance yet" description="Enable MCP servers in settings to surface bounded tooling guidance here." />
                                            ) : (
                                                <div className="mt-3 space-y-2">
                                                    {workspace.suggested_tooling.map((line, index) => (
                                                        <div key={`${line}-${index}`} className="rounded-xl border border-slate-200 bg-white/90 px-3 py-2 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900/70 dark:text-slate-300">
                                                            {line}
                                                        </div>
                                                    ))}
                                                </div>
                                            )}
                                        </div>

                                        <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                            <div className="flex flex-wrap items-center justify-between gap-3">
                                                <div>
                                                    <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Grounded Draft</h4>
                                                    <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">Generate deterministic hypotheses and an investigation plan from stored evidence and bounded tools.</p>
                                                </div>
                                                <div className="flex flex-wrap gap-2">
                                                    <button
                                                        type="button"
                                                        onClick={() => void handleGenerateAgentDraft()}
                                                        disabled={generatingAgentDraft}
                                                        className="rounded-lg border border-sky-300 bg-sky-50 px-3 py-2 text-xs font-medium text-sky-700 transition hover:bg-sky-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-sky-800 dark:bg-sky-950/30 dark:text-sky-200 dark:hover:bg-sky-950/50"
                                                    >
                                                        {generatingAgentDraft ? 'Generating...' : 'Generate Draft'}
                                                    </button>
                                                    <button
                                                        type="button"
                                                        onClick={() => void handleGenerateInterpretation()}
                                                        disabled={generatingInterpretation}
                                                        className="rounded-lg border border-violet-300 bg-violet-50 px-3 py-2 text-xs font-medium text-violet-700 transition hover:bg-violet-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-violet-800 dark:bg-violet-950/30 dark:text-violet-200 dark:hover:bg-violet-950/50"
                                                    >
                                                        {generatingInterpretation ? 'Generating...' : 'Local Model Interpretation'}
                                                    </button>
                                                </div>
                                            </div>
                                            <div className="mt-4 space-y-4">
                                                {agentDraft ? (
                                                    <>
                                                        <div className="rounded-xl border border-slate-200 bg-white/90 px-4 py-3 dark:border-slate-800 dark:bg-slate-900/70">
                                                            <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Draft Summary</div>
                                                            <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">{agentDraft.summary}</div>
                                                        </div>
                                                        <div className="space-y-3">
                                                            {agentDraft.hypotheses.map((hypothesis, index) => (
                                                                <div key={`${hypothesis.title}-${index}`} className="rounded-xl border border-slate-200 bg-white/90 px-4 py-3 dark:border-slate-800 dark:bg-slate-900/70">
                                                                    <div className="flex flex-wrap items-center justify-between gap-2">
                                                                        <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">{hypothesis.title}</div>
                                                                        <span className="rounded-full border border-slate-300 px-2 py-0.5 text-[11px] font-medium text-slate-700 dark:border-slate-700 dark:text-slate-300">Rank #{index + 1}</span>
                                                                    </div>
                                                                    <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">{hypothesis.rationale}</div>
                                                                    {hypothesis.evidence_refs?.length ? (
                                                                        <div className="mt-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-3 dark:border-slate-800 dark:bg-slate-950/40">
                                                                            <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Supporting Tool Output</div>
                                                                            {renderAgentEvidenceRefs(hypothesis.evidence_refs, `${hypothesis.title}-evidence`)}
                                                                        </div>
                                                                    ) : null}
                                                                </div>
                                                            ))}
                                                        </div>
                                                        <div className="space-y-3">
                                                            {agentDraft.investigation_plan.map((step, index) => (
                                                                <div key={`${step.title}-${index}`} className="rounded-xl border border-slate-200 bg-white/90 px-4 py-3 dark:border-slate-800 dark:bg-slate-900/70">
                                                                    <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">{index + 1}. {step.title}</div>
                                                                    <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">{step.description}</div>
                                                                    {step.evidence_refs?.length ? (
                                                                        <div className="mt-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-3 dark:border-slate-800 dark:bg-slate-950/40">
                                                                            <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Evidence For This Step</div>
                                                                            {renderAgentEvidenceRefs(step.evidence_refs, `${step.title}-evidence`)}
                                                                        </div>
                                                                    ) : null}
                                                                </div>
                                                            ))}
                                                        </div>
                                                        {agentDraft.tool_runs.length ? (
                                                            <div className="rounded-xl border border-slate-200 bg-slate-950 px-4 py-3 dark:border-slate-700">
                                                                <div className="mb-2 text-xs uppercase tracking-[0.16em] text-slate-400">Tool Evidence Used</div>
                                                                <pre className="overflow-x-auto whitespace-pre-wrap break-words text-xs text-slate-100">{prettyJson(agentDraft.tool_runs)}</pre>
                                                            </div>
                                                        ) : null}
                                                    </>
                                                ) : (
                                                    <EmptyState title="No draft generated yet" description="Use Generate Draft to create a grounded incident hypothesis and investigation plan." />
                                                )}

                                                {agentInterpretation ? (
                                                    <div className="space-y-3">
                                                        <div className="rounded-xl border border-slate-200 bg-white/90 px-4 py-3 dark:border-slate-800 dark:bg-slate-900/70">
                                                            <div className="flex flex-wrap items-center justify-between gap-2">
                                                                <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">Local Model Interpretation</div>
                                                                <span className="rounded-full border border-slate-300 px-2 py-0.5 text-[11px] font-medium text-slate-700 dark:border-slate-700 dark:text-slate-300">
                                                                    {agentInterpretation.generated ? 'Generated' : 'Baseline'}
                                                                </span>
                                                            </div>
                                                            <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">{agentInterpretation.operator_summary || 'No model summary returned.'}</div>
                                                        </div>
                                                        {agentInterpretation.likely_root_cause ? (
                                                            <div className="rounded-xl border border-slate-200 bg-white/90 px-4 py-3 dark:border-slate-800 dark:bg-slate-900/70">
                                                                <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Likely Root Cause</div>
                                                                <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">{agentInterpretation.likely_root_cause}</div>
                                                            </div>
                                                        ) : null}
                                                        {(agentInterpretation.watchouts || []).length > 0 ? (
                                                            <div className="rounded-xl border border-slate-200 bg-white/90 px-4 py-3 dark:border-slate-800 dark:bg-slate-900/70">
                                                                <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Watchouts</div>
                                                                <div className="mt-2 space-y-2">
                                                                    {(agentInterpretation.watchouts || []).map((watchout) => (
                                                                        <div key={watchout} className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700 dark:border-slate-700 dark:bg-slate-950/40 dark:text-slate-300">
                                                                            {watchout}
                                                                        </div>
                                                                    ))}
                                                                </div>
                                                            </div>
                                                        ) : null}
                                                        {agentInterpretation.operator_message_draft ? (
                                                            <div className="rounded-xl border border-slate-200 bg-white/90 px-4 py-3 dark:border-slate-800 dark:bg-slate-900/70">
                                                                <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Operator Message Draft</div>
                                                                <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">{agentInterpretation.operator_message_draft}</div>
                                                            </div>
                                                        ) : null}
                                                        {agentInterpretation.raw_response ? (
                                                            <div className="rounded-xl border border-slate-200 bg-slate-950 px-4 py-3 dark:border-slate-700">
                                                                <div className="mb-2 text-xs uppercase tracking-[0.16em] text-slate-400">Raw Model Response</div>
                                                                <pre className="overflow-x-auto whitespace-pre-wrap break-words text-xs text-slate-100">{agentInterpretation.raw_response}</pre>
                                                            </div>
                                                        ) : null}
                                                    </div>
                                                ) : null}
                                            </div>
                                        </div>
                                    </div>

                                    <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                        <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Read-Only MCP Tools</h4>
                                        {mcpTools.length === 0 ? (
                                            <div className="mt-3">
                                                <EmptyState title="No MCP tools exposed" description="Enable an MCP server with supported read-only tools to run bounded investigation calls here." />
                                            </div>
                                        ) : (
                                            <div className="mt-3 space-y-3">
                                                {mcpTools.map((tool) => {
                                                    const toolKey = `${tool.server_id}:${tool.tool_name}`
                                                    const result = mcpResults[toolKey]
                                                    const isRunning = runningMCPToolKey === toolKey
                                                    return (
                                                        <div key={toolKey} className="rounded-xl border border-slate-200 bg-white/90 px-4 py-4 dark:border-slate-800 dark:bg-slate-900/70">
                                                            <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                                                                <div className="min-w-0 flex-1">
                                                                    <div className="flex flex-wrap items-center gap-2">
                                                                        <div className="text-sm font-medium text-slate-900 dark:text-slate-100">{tool.display_name}</div>
                                                                        <span className="inline-flex rounded-full border border-slate-300 bg-slate-50 px-2.5 py-1 text-[11px] font-medium text-slate-700 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-300">
                                                                            {tool.server_name}
                                                                        </span>
                                                                        <span className="inline-flex rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-1 text-[11px] font-medium text-emerald-800 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-200">
                                                                            Read only
                                                                        </span>
                                                                    </div>
                                                                    <div className="mt-2 text-sm text-slate-600 dark:text-slate-400">{tool.description}</div>
                                                                    <div className="mt-2 text-xs uppercase tracking-[0.14em] text-slate-500 dark:text-slate-500">
                                                                        {tool.server_kind} • {tool.tool_name}
                                                                    </div>
                                                                </div>
                                                                <button
                                                                    type="button"
                                                                    onClick={() => void handleRunMCPTool(tool)}
                                                                    disabled={isRunning}
                                                                    className="rounded-lg border border-sky-300 bg-sky-50 px-3 py-2 text-xs font-medium text-sky-700 transition hover:bg-sky-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-sky-800 dark:bg-sky-950/30 dark:text-sky-200 dark:hover:bg-sky-950/50"
                                                                >
                                                                    {isRunning ? 'Running...' : 'Run Tool'}
                                                                </button>
                                                            </div>
                                                            {result ? (
                                                                <div className="mt-4 rounded-xl border border-slate-200 bg-slate-950 px-4 py-3 dark:border-slate-700">
                                                                    <div className="mb-2 flex items-center justify-between gap-3">
                                                                        <div className="text-xs uppercase tracking-[0.16em] text-slate-400">
                                                                            Output • {formatDateTime(result.executed_at)}
                                                                        </div>
                                                                        <button
                                                                            type="button"
                                                                            onClick={() => handleDismissMCPToolResult(toolKey)}
                                                                            className="rounded-md border border-slate-700 bg-slate-900 px-2 py-1 text-[11px] font-medium text-slate-300 transition hover:border-slate-500 hover:text-white"
                                                                        >
                                                                            Close
                                                                        </button>
                                                                    </div>
                                                                    {renderMCPToolResult(result)}
                                                                </div>
                                                            ) : null}
                                                        </div>
                                                    )
                                                })}
                                            </div>
                                        )}
                                    </div>
                                </div>
                            )}
                        </SectionCard>
                        </div>
                        </>
                        ) : null}
                    </div>
                ) : null}
            </Drawer>
        </div>
    )
}

export default SRESmartBotIncidentsPage
