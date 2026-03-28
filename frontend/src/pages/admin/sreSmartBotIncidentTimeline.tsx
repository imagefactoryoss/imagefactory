import type { SREActionAttempt, SREApproval, SREEvidence, SREFinding } from '@/types'
import React, { useMemo, useState } from 'react'

import { EmptyState, formatDateTime, relativeTime } from './sreSmartBotIncidentPageShared'

type TimelineActor = 'bot' | 'operator' | 'system'
type TimelineEventKind = 'finding' | 'evidence' | 'action' | 'approval'

type TimelineEntry = {
    id: string
    occurredAt: string
    actor: TimelineActor
    kind: TimelineEventKind
    title: string
    detail: string
    badge?: string
    emphasis?: 'neutral' | 'warning' | 'critical' | 'success'
}

type TimelineActorFilter = 'all' | TimelineActor
type TimelineEventFilter = 'all' | TimelineEventKind

const timelineActorTone: Record<TimelineActor, string> = {
    bot: 'border-sky-200 bg-sky-50 text-sky-800 dark:border-sky-900/40 dark:bg-sky-950/30 dark:text-sky-200',
    operator: 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-200',
    system: 'border-slate-200 bg-slate-50 text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300',
}

const timelineEntryTone = (emphasis: TimelineEntry['emphasis']) => {
    switch (emphasis) {
        case 'critical':
            return 'border-rose-200 bg-rose-50 dark:border-rose-900/40 dark:bg-rose-950/20'
        case 'warning':
            return 'border-amber-200 bg-amber-50 dark:border-amber-900/40 dark:bg-amber-950/20'
        case 'success':
            return 'border-emerald-200 bg-emerald-50 dark:border-emerald-900/40 dark:bg-emerald-950/20'
        default:
            return 'border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-950'
    }
}

const normalizeTimelineTimestamp = (value?: string | null) => {
    if (!value) return 0
    const timestamp = new Date(value).getTime()
    return Number.isNaN(timestamp) ? 0 : timestamp
}

type SREIncidentTimelineProps = {
    findings: SREFinding[]
    evidence: SREEvidence[]
    actions: SREActionAttempt[]
    approvals: SREApproval[]
    actionsById: Record<string, SREActionAttempt>
}

export const SREIncidentTimeline: React.FC<SREIncidentTimelineProps> = ({ findings, evidence, actions, approvals, actionsById }) => {
    const [timelineActorFilter, setTimelineActorFilter] = useState<TimelineActorFilter>('all')
    const [timelineEventFilter, setTimelineEventFilter] = useState<TimelineEventFilter>('all')

    const entries = useMemo<TimelineEntry[]>(() => {
        const findingEntries: TimelineEntry[] = findings.map((finding) => ({
            id: `finding-${finding.id}`,
            occurredAt: finding.occurred_at,
            actor: 'bot',
            kind: 'finding',
            title: finding.title || finding.signal_type,
            detail: finding.message,
            badge: `${finding.signal_type} • ${finding.severity}`,
            emphasis: finding.severity === 'critical' ? 'critical' : finding.severity === 'warning' ? 'warning' : 'neutral',
        }))

        const evidenceEntries: TimelineEntry[] = evidence.map((item) => ({
            id: `evidence-${item.id}`,
            occurredAt: item.captured_at,
            actor: 'bot',
            kind: 'evidence',
            title: `Evidence captured: ${item.evidence_type}`,
            detail: item.summary,
            badge: item.evidence_type,
            emphasis: 'neutral',
        }))

        const actionEntries: TimelineEntry[] = actions.map((action) => {
            const actor: TimelineActor = action.actor_type === 'operator' || action.actor_type === 'human'
                ? 'operator'
                : action.actor_type === 'system'
                    ? 'system'
                    : 'bot'
            const status = (action.status || '').toLowerCase()
            const verb = status === 'completed'
                ? 'completed'
                : status === 'approved'
                    ? 'approved for execution'
                    : status === 'failed'
                        ? 'failed'
                        : 'proposed'
            const target = action.target_ref ? `${action.target_kind} ${action.target_ref}` : action.target_kind
            return {
                id: `action-${action.id}`,
                occurredAt: action.completed_at || action.started_at || action.requested_at,
                actor,
                kind: 'action',
                title: `Action ${verb}: ${action.action_key}`,
                detail: `${action.action_key} targeted ${target}.${action.error_message ? ` ${action.error_message}` : ''}`.trim(),
                badge: action.status,
                emphasis: status === 'completed' ? 'success' : status === 'failed' ? 'critical' : 'warning',
            }
        })

        const approvalEntries: TimelineEntry[] = approvals.flatMap((approval) => {
            const linkedAction = approval.action_attempt_id ? actionsById[approval.action_attempt_id] : undefined
            const actionName = linkedAction?.action_key || approval.channel_provider_id
            const requested: TimelineEntry = {
                id: `approval-request-${approval.id}`,
                occurredAt: approval.requested_at,
                actor: approval.requested_by ? 'operator' : 'bot',
                kind: 'approval',
                title: `Approval requested for ${actionName}`,
                detail: approval.request_message,
                badge: approval.channel_provider_id,
                emphasis: 'warning',
            }
            const decided: TimelineEntry[] = approval.decided_at ? [{
                id: `approval-decision-${approval.id}`,
                occurredAt: approval.decided_at,
                actor: 'operator',
                kind: 'approval',
                title: `${approval.status === 'approved' ? 'Approval granted' : 'Approval rejected'} for ${actionName}`,
                detail: approval.decision_comment || `Decision recorded by ${approval.decided_by || 'operator'}.`,
                badge: approval.status,
                emphasis: approval.status === 'approved' ? 'success' : 'critical',
            }] : []
            return [requested, ...decided]
        })

        return [...findingEntries, ...evidenceEntries, ...actionEntries, ...approvalEntries]
            .filter((entry) => normalizeTimelineTimestamp(entry.occurredAt) > 0)
            .sort((left, right) => normalizeTimelineTimestamp(right.occurredAt) - normalizeTimelineTimestamp(left.occurredAt))
    }, [findings, evidence, actions, approvals, actionsById])

    const actorCounts = useMemo<Record<TimelineActor, number>>(() => {
        return entries.reduce<Record<TimelineActor, number>>((acc, entry) => {
            acc[entry.actor] += 1
            return acc
        }, { bot: 0, operator: 0, system: 0 })
    }, [entries])

    const eventCounts = useMemo<Record<TimelineEventKind, number>>(() => {
        return entries.reduce<Record<TimelineEventKind, number>>((acc, entry) => {
            acc[entry.kind] += 1
            return acc
        }, { finding: 0, evidence: 0, action: 0, approval: 0 })
    }, [entries])

    const filteredEntries = useMemo(() => {
        return entries.filter((entry) => {
            const actorMatch = timelineActorFilter === 'all' || entry.actor === timelineActorFilter
            const eventMatch = timelineEventFilter === 'all' || entry.kind === timelineEventFilter
            return actorMatch && eventMatch
        })
    }, [entries, timelineActorFilter, timelineEventFilter])

    return (
        <>
            <div className="mb-4 flex flex-wrap items-center gap-2">
                {([
                    { id: 'all', label: 'All activity', count: entries.length },
                    { id: 'bot', label: 'Bot', count: actorCounts.bot },
                    { id: 'operator', label: 'Operator', count: actorCounts.operator },
                    { id: 'system', label: 'System', count: actorCounts.system },
                ] as Array<{ id: TimelineActorFilter; label: string; count: number }>).map((option) => {
                    const active = timelineActorFilter === option.id
                    return (
                        <button
                            key={option.id}
                            type="button"
                            onClick={() => setTimelineActorFilter(option.id)}
                            className={`inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-medium transition ${active
                                ? 'border-sky-400 bg-sky-50 text-sky-800 dark:border-sky-500 dark:bg-sky-950/40 dark:text-sky-200'
                                : 'border-slate-300 bg-white text-slate-700 hover:border-slate-400 hover:text-slate-900 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-300 dark:hover:border-slate-600 dark:hover:text-slate-100'
                                }`}
                        >
                            <span>{option.label}</span>
                            <span className="rounded-full border border-current/20 px-1.5 py-0.5 text-[11px]">{option.count}</span>
                        </button>
                    )
                })}
            </div>

            <div className="mb-4 flex flex-wrap items-center gap-2">
                {([
                    { id: 'all', label: 'All events', count: entries.length },
                    { id: 'finding', label: 'Findings', count: eventCounts.finding },
                    { id: 'evidence', label: 'Evidence', count: eventCounts.evidence },
                    { id: 'action', label: 'Actions', count: eventCounts.action },
                    { id: 'approval', label: 'Approvals', count: eventCounts.approval },
                ] as Array<{ id: TimelineEventFilter; label: string; count: number }>).map((option) => {
                    const active = timelineEventFilter === option.id
                    return (
                        <button
                            key={option.id}
                            type="button"
                            onClick={() => setTimelineEventFilter(option.id)}
                            className={`inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-medium transition ${active
                                ? 'border-violet-400 bg-violet-50 text-violet-800 dark:border-violet-500 dark:bg-violet-950/40 dark:text-violet-200'
                                : 'border-slate-300 bg-white text-slate-700 hover:border-slate-400 hover:text-slate-900 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-300 dark:hover:border-slate-600 dark:hover:text-slate-100'
                                }`}
                        >
                            <span>{option.label}</span>
                            <span className="rounded-full border border-current/20 px-1.5 py-0.5 text-[11px]">{option.count}</span>
                        </button>
                    )
                })}
            </div>

            {filteredEntries.length === 0 ? (
                <EmptyState title="No activity matches the current filters" description="Broaden the actor or event filters to see the full incident thread again." />
            ) : (
                <div className="space-y-3">
                    {filteredEntries.map((entry) => (
                        <div key={entry.id} className={`rounded-xl border p-4 ${timelineEntryTone(entry.emphasis)}`}>
                            <div className="flex flex-wrap items-start justify-between gap-3">
                                <div className="min-w-0 flex-1">
                                    <div className="flex flex-wrap items-center gap-2">
                                        <span className={`inline-flex rounded-full border px-2.5 py-1 text-[11px] font-semibold uppercase tracking-[0.14em] ${timelineActorTone[entry.actor]}`}>
                                            {entry.actor}
                                        </span>
                                        <span className="inline-flex rounded-full border border-slate-300 px-2 py-0.5 text-[11px] font-medium uppercase tracking-[0.12em] text-slate-700 dark:border-slate-700 dark:text-slate-300">
                                            {entry.kind}
                                        </span>
                                        {entry.badge ? (
                                            <span className="inline-flex rounded-full border border-slate-300 px-2 py-0.5 text-[11px] font-medium text-slate-700 dark:border-slate-700 dark:text-slate-300">
                                                {entry.badge}
                                            </span>
                                        ) : null}
                                    </div>
                                    <div className="mt-2 text-sm font-semibold text-slate-900 dark:text-slate-100">{entry.title}</div>
                                    <div className="mt-1 text-sm text-slate-700 dark:text-slate-300">{entry.detail}</div>
                                </div>
                                <div className="shrink-0 text-right text-xs text-slate-500 dark:text-slate-400">
                                    <div>{formatDateTime(entry.occurredAt)}</div>
                                    <div className="mt-1">{relativeTime(entry.occurredAt)}</div>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </>
    )
}