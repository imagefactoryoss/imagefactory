import type {
    SREActionAttempt,
    SREIncident,
    SREIncidentSeverity,
    SREIncidentStatus,
    SREMCPToolInvocationResponse,
} from '@/types'
import React from 'react'

import type {
    AsyncBacklogInsight,
    MessagingConsumerInsight,
    MessagingTransportInsight,
} from './sreSmartBotAsyncSummary'

export const inputClass = 'w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900'

export const severityTone: Record<SREIncidentSeverity, string> = {
    info: 'bg-sky-100 text-sky-800 border-sky-200 dark:bg-sky-500/15 dark:text-sky-200 dark:border-sky-500/30',
    warning: 'bg-amber-100 text-amber-900 border-amber-200 dark:bg-amber-500/15 dark:text-amber-200 dark:border-amber-500/30',
    critical: 'bg-rose-100 text-rose-900 border-rose-200 dark:bg-rose-500/15 dark:text-rose-200 dark:border-rose-500/30',
}

export const statusTone: Record<SREIncidentStatus, string> = {
    observed: 'bg-slate-100 text-slate-800 border-slate-200 dark:bg-slate-700/40 dark:text-slate-100 dark:border-slate-600',
    triaged: 'bg-indigo-100 text-indigo-800 border-indigo-200 dark:bg-indigo-500/15 dark:text-indigo-200 dark:border-indigo-500/30',
    contained: 'bg-emerald-100 text-emerald-800 border-emerald-200 dark:bg-emerald-500/15 dark:text-emerald-200 dark:border-emerald-500/30',
    recovering: 'bg-cyan-100 text-cyan-800 border-cyan-200 dark:bg-cyan-500/15 dark:text-cyan-200 dark:border-cyan-500/30',
    resolved: 'bg-green-100 text-green-800 border-green-200 dark:bg-green-500/15 dark:text-green-200 dark:border-green-500/30',
    suppressed: 'bg-zinc-100 text-zinc-800 border-zinc-200 dark:bg-zinc-600/30 dark:text-zinc-100 dark:border-zinc-500',
    escalated: 'bg-fuchsia-100 text-fuchsia-800 border-fuchsia-200 dark:bg-fuchsia-500/15 dark:text-fuchsia-200 dark:border-fuchsia-500/30',
}

export const actionTone = (status: string) => {
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

export const formatDateTime = (value?: string | null) => {
    if (!value) return '—'
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return value
    return date.toLocaleString()
}

export const relativeTime = (value?: string | null) => {
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

export const prettyJson = (value: unknown) => {
    if (!value) return ''
    try {
        return JSON.stringify(value, null, 2)
    } catch {
        return String(value)
    }
}

export const buildApprovalRequestMessage = (incident: SREIncident, action: SREActionAttempt) => {
    const target = action.target_ref ? `${action.target_kind} ${action.target_ref}` : action.target_kind
    const summary = incident.summary?.trim() || incident.display_name || incident.incident_type
    return `Requesting operator approval for ${action.action_key} on ${target} while investigating ${summary}.`
}

export const asRecord = (value: unknown): Record<string, any> | null => {
    if (!value || typeof value !== 'object' || Array.isArray(value)) return null
    return value as Record<string, any>
}

export const asArrayOfRecords = (value: unknown): Record<string, any>[] => {
    if (!Array.isArray(value)) return []
    return value.map((item) => asRecord(item)).filter((item): item is Record<string, any> => item !== null)
}

export const prettifyTrendLabel = (value?: string | null) => {
    if (!value) return 'Unknown'
    return value.replace(/_/g, ' ')
}

export const renderAgentEvidenceRefs = (
    refs: { tool_name: string; server_name: string; summary: string }[],
    keyPrefix: string,
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

export const renderMCPToolResult = (result: SREMCPToolInvocationResponse) => {
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

    if (result.tool_name === 'messaging_consumers.recent') {
        const visibleConsumers = typeof payload.count === 'number' ? payload.count : 0
        const laggingCount = typeof payload.lagging_count === 'number' ? payload.lagging_count : 0
        const maxPendingCount = typeof payload.max_pending_count === 'number' ? payload.max_pending_count : 0
        const consumers = asArrayOfRecords(payload.consumers)
        const topConsumer = consumers[0]

        return (
            <div className="space-y-3">
                <div className="grid gap-3 sm:grid-cols-3">
                    <div className="rounded-xl border border-amber-900/50 bg-amber-950/40 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-amber-300">Visible Consumers</div>
                        <div className="mt-2 text-xl font-semibold text-amber-100">{visibleConsumers}</div>
                        <div className="mt-1 text-xs text-amber-200/80">JetStream consumers visible to the snapshot</div>
                    </div>
                    <div className="rounded-xl border border-rose-900/50 bg-rose-950/40 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-rose-300">Lagging Consumers</div>
                        <div className="mt-2 text-xl font-semibold text-rose-100">{laggingCount}</div>
                        <div className="mt-1 text-xs text-rose-200/80">Consumers with pending or ack-pending pressure</div>
                    </div>
                    <div className="rounded-xl border border-cyan-900/50 bg-cyan-950/40 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-cyan-300">Max Pending</div>
                        <div className="mt-2 text-xl font-semibold text-cyan-100">{maxPendingCount}</div>
                        <div className="mt-1 text-xs text-cyan-200/80">Highest pending count across visible consumers</div>
                    </div>
                </div>

                {topConsumer ? (
                    <div className="rounded-xl border border-slate-700 bg-slate-900/70 px-3 py-3">
                        <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-slate-400">Top Consumer Snapshot</div>
                        <div className="mt-2 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                            <div>
                                <div className="text-xs text-slate-400">Target</div>
                                <div className="mt-1 text-sm font-semibold text-slate-100">{String(topConsumer.stream ?? 'unknown')}/{String(topConsumer.consumer ?? 'unknown')}</div>
                            </div>
                            <div>
                                <div className="text-xs text-slate-400">Pending</div>
                                <div className="mt-1 text-sm font-semibold text-slate-100">{String(topConsumer.pending_count ?? 0)}</div>
                            </div>
                            <div>
                                <div className="text-xs text-slate-400">Ack Pending</div>
                                <div className="mt-1 text-sm font-semibold text-slate-100">{String(topConsumer.ack_pending_count ?? 0)}</div>
                            </div>
                            <div>
                                <div className="text-xs text-slate-400">Last Active</div>
                                <div className="mt-1 text-sm font-semibold text-slate-100">{formatDateTime(typeof topConsumer.last_active === 'string' ? topConsumer.last_active : '')}</div>
                            </div>
                        </div>
                    </div>
                ) : null}
            </div>
        )
    }

    return (
        <pre className="overflow-x-auto whitespace-pre-wrap break-words text-xs text-slate-100">
            {prettyJson(result.payload)}
        </pre>
    )
}

export const SectionCard: React.FC<{ title: string; subtitle?: string; children: React.ReactNode }> = ({ title, subtitle, children }) => (
    <section className="rounded-2xl border border-slate-200 bg-white/90 p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900/85">
        <div className="mb-4">
            <h2 className="text-sm font-semibold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">{title}</h2>
            {subtitle ? <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">{subtitle}</p> : null}
        </div>
        {children}
    </section>
)

export const EmptyState: React.FC<{ title: string; description: string }> = ({ title, description }) => (
    <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-4 py-6 text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-800/40 dark:text-slate-300">
        <p className="font-medium text-slate-800 dark:text-slate-100">{title}</p>
        <p className="mt-1">{description}</p>
    </div>
)

export const AsyncBacklogInsightContent: React.FC<{ insight: AsyncBacklogInsight }> = ({ insight }) => {
    const recentObservations = Object.entries(insight.recentObservations)
    const correlationHints = Object.entries(insight.correlationHints)

    return (
        <div className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-900/40 dark:bg-amber-950/30">
                    <div className="text-xs uppercase tracking-wide text-amber-700 dark:text-amber-300">Queue Path</div>
                    <div className="mt-2 text-lg font-semibold text-amber-900 dark:text-amber-100">{insight.displayName}</div>
                    <div className="mt-1 text-sm text-amber-700/80 dark:text-amber-200/80">{insight.queueKind} via {insight.subsystem}</div>
                </div>
                <div className="rounded-xl border border-sky-200 bg-sky-50 p-4 dark:border-sky-900/40 dark:bg-sky-950/30">
                    <div className="text-xs uppercase tracking-wide text-sky-700 dark:text-sky-300">Current Depth</div>
                    <div className="mt-2 text-2xl font-semibold text-sky-900 dark:text-sky-100">{insight.count}</div>
                    <div className="mt-1 text-sm text-sky-700/80 dark:text-sky-200/80">{insight.operatorStatus}</div>
                </div>
                <div className="rounded-xl border border-fuchsia-200 bg-fuchsia-50 p-4 dark:border-fuchsia-900/40 dark:bg-fuchsia-950/30">
                    <div className="text-xs uppercase tracking-wide text-fuchsia-700 dark:text-fuchsia-300">Threshold</div>
                    <div className="mt-2 text-2xl font-semibold text-fuchsia-900 dark:text-fuchsia-100">{insight.threshold}</div>
                    <div className="mt-1 text-sm text-fuchsia-700/80 dark:text-fuchsia-200/80">Delta {insight.thresholdDelta} • {insight.thresholdRatioPercent}%</div>
                </div>
                <div className="rounded-xl border border-violet-200 bg-violet-50 p-4 dark:border-violet-900/40 dark:bg-violet-950/30">
                    <div className="text-xs uppercase tracking-wide text-violet-700 dark:text-violet-300">Trend</div>
                    <div className="mt-2 text-2xl font-semibold capitalize text-violet-900 dark:text-violet-100">{prettifyTrendLabel(insight.trend)}</div>
                    <div className="mt-1 text-sm text-violet-700/80 dark:text-violet-200/80">{insight.source === 'workspace' ? 'Workspace-projected signal' : 'Tool-derived fallback'}</div>
                </div>
            </div>

            <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Latest Summary</div>
                <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">{insight.latestSummary}</div>
                <div className="mt-2 text-xs text-slate-500 dark:text-slate-400">Captured {insight.latestCapturedAt ? formatDateTime(insight.latestCapturedAt) : '—'}</div>
            </div>

            {recentObservations.length > 0 ? (
                <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Recent Observations</div>
                    <div className="mt-3 flex flex-wrap gap-2">
                        {recentObservations.map(([key, value]) => (
                            <span key={key} className="inline-flex rounded-full border border-slate-300 bg-slate-50 px-3 py-1 text-xs font-medium text-slate-700 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-300">
                                {key.replace(/_/g, ' ')}: {value}
                            </span>
                        ))}
                    </div>
                </div>
            ) : null}

            {correlationHints.length > 0 ? (
                <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Correlation Hints</div>
                    <div className="mt-3 flex flex-wrap gap-2">
                        {correlationHints.map(([key, value]) => (
                            <span key={key} className="inline-flex rounded-full border border-cyan-300 bg-cyan-50 px-3 py-1 text-xs font-medium text-cyan-700 dark:border-cyan-800 dark:bg-cyan-950/30 dark:text-cyan-200">
                                {key.replace(/_/g, ' ')}: {value}
                            </span>
                        ))}
                    </div>
                </div>
            ) : null}
        </div>
    )
}

export const MessagingTransportInsightContent: React.FC<{ insight: MessagingTransportInsight }> = ({ insight }) => (
    <div className="space-y-4">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Status</div>
                <div className="mt-2 text-2xl font-semibold capitalize text-slate-900 dark:text-slate-100">{insight.status}</div>
                <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">{insight.operatorStatus}</div>
            </div>
            <div className="rounded-xl border border-cyan-200 bg-cyan-50 p-4 dark:border-cyan-900/40 dark:bg-cyan-950/30">
                <div className="text-xs uppercase tracking-wide text-cyan-700 dark:text-cyan-300">Reconnects</div>
                <div className="mt-2 text-2xl font-semibold text-cyan-900 dark:text-cyan-100">{insight.reconnects}</div>
                <div className="mt-1 text-sm text-cyan-700/80 dark:text-cyan-200/80">Threshold {insight.reconnectThreshold}</div>
            </div>
            <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 dark:border-rose-900/40 dark:bg-rose-950/30">
                <div className="text-xs uppercase tracking-wide text-rose-700 dark:text-rose-300">Disconnects</div>
                <div className="mt-2 text-2xl font-semibold text-rose-900 dark:text-rose-100">{insight.disconnects}</div>
                <div className="mt-1 text-sm text-rose-700/80 dark:text-rose-200/80">Transport interruptions in the latest window</div>
            </div>
            <div className="rounded-xl border border-violet-200 bg-violet-50 p-4 dark:border-violet-900/40 dark:bg-violet-950/30">
                <div className="text-xs uppercase tracking-wide text-violet-700 dark:text-violet-300">Snapshot Source</div>
                <div className="mt-2 text-lg font-semibold text-violet-900 dark:text-violet-100">{insight.source === 'workspace' ? 'Workspace summary' : 'Tool fallback'}</div>
                <div className="mt-1 text-sm text-violet-700/80 dark:text-violet-200/80">Captured {insight.latestCapturedAt ? formatDateTime(insight.latestCapturedAt) : '—'}</div>
            </div>
        </div>

        <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
            <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Latest Summary</div>
            <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">{insight.latestSummary}</div>
        </div>
    </div>
)

export const MessagingConsumerInsightContent: React.FC<{ insight: MessagingConsumerInsight }> = ({ insight }) => {
    const correlationHints = Object.entries(insight.correlationHints)

    return (
        <div className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-900/40 dark:bg-amber-950/30">
                    <div className="text-xs uppercase tracking-wide text-amber-700 dark:text-amber-300">Consumer Path</div>
                    <div className="mt-2 text-lg font-semibold text-amber-900 dark:text-amber-100">{insight.stream}/{insight.consumer}</div>
                    <div className="mt-1 text-sm text-amber-700/80 dark:text-amber-200/80">{insight.kind.replace(/_/g, ' ')}</div>
                </div>
                <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 dark:border-rose-900/40 dark:bg-rose-950/30">
                    <div className="text-xs uppercase tracking-wide text-rose-700 dark:text-rose-300">Current Pressure</div>
                    <div className="mt-2 text-2xl font-semibold text-rose-900 dark:text-rose-100">{insight.count}</div>
                    <div className="mt-1 text-sm text-rose-700/80 dark:text-rose-200/80">{insight.operatorStatus}</div>
                </div>
                <div className="rounded-xl border border-cyan-200 bg-cyan-50 p-4 dark:border-cyan-900/40 dark:bg-cyan-950/30">
                    <div className="text-xs uppercase tracking-wide text-cyan-700 dark:text-cyan-300">Threshold</div>
                    <div className="mt-2 text-2xl font-semibold text-cyan-900 dark:text-cyan-100">{insight.threshold}</div>
                    <div className="mt-1 text-sm text-cyan-700/80 dark:text-cyan-200/80">Delta {insight.thresholdDelta} • {insight.thresholdRatioPercent}%</div>
                </div>
                <div className="rounded-xl border border-violet-200 bg-violet-50 p-4 dark:border-violet-900/40 dark:bg-violet-950/30">
                    <div className="text-xs uppercase tracking-wide text-violet-700 dark:text-violet-300">Trend</div>
                    <div className="mt-2 text-2xl font-semibold capitalize text-violet-900 dark:text-violet-100">{prettifyTrendLabel(insight.trend)}</div>
                    <div className="mt-1 text-sm text-violet-700/80 dark:text-violet-200/80">{insight.source === 'workspace' ? 'Workspace-projected signal' : 'Tool-derived fallback'}</div>
                </div>
            </div>

            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Pending</div>
                    <div className="mt-2 text-lg font-semibold text-slate-900 dark:text-slate-100">{insight.pendingCount}</div>
                </div>
                <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Ack Pending</div>
                    <div className="mt-2 text-lg font-semibold text-slate-900 dark:text-slate-100">{insight.ackPendingCount}</div>
                </div>
                <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Waiting</div>
                    <div className="mt-2 text-lg font-semibold text-slate-900 dark:text-slate-100">{insight.waitingCount}</div>
                </div>
                <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Last Active</div>
                    <div className="mt-2 text-sm font-semibold text-slate-900 dark:text-slate-100">{insight.lastActive ? formatDateTime(insight.lastActive) : '—'}</div>
                </div>
            </div>

            <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Latest Summary</div>
                <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">{insight.latestSummary}</div>
                <div className="mt-2 text-xs text-slate-500 dark:text-slate-400">Captured {insight.latestCapturedAt ? formatDateTime(insight.latestCapturedAt) : '—'}</div>
            </div>

            {correlationHints.length > 0 ? (
                <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Correlation Hints</div>
                    <div className="mt-3 flex flex-wrap gap-2">
                        {correlationHints.map(([key, value]) => (
                            <span key={key} className="inline-flex rounded-full border border-cyan-300 bg-cyan-50 px-3 py-1 text-xs font-medium text-cyan-700 dark:border-cyan-800 dark:bg-cyan-950/30 dark:text-cyan-200">
                                {key.replace(/_/g, ' ')}: {value}
                            </span>
                        ))}
                    </div>
                </div>
            ) : null}
        </div>
    )
}