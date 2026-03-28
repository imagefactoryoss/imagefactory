import type { SREApprovalQueueItem } from '@/types'
import React from 'react'
import { Link } from 'react-router-dom'

import {
    actionExecutable,
    approvalTone,
    EmptyState,
    formatDateTime,
    inputClass,
    labelClass,
    relativeTime,
    SectionCard,
} from './sreSmartBotApprovalsShared'

export const ApprovalFiltersPanel: React.FC<{
    search: string
    setSearch: (value: string) => void
    statusFilter: string
    setStatusFilter: (value: string) => void
    applyFilters: () => void
}> = ({ search, setSearch, statusFilter, setStatusFilter, applyFilters }) => (
    <SectionCard title="Filters" subtitle="Focus on pending work first, or widen the view to inspect recent approval decisions.">
        <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_220px_160px]">
            <label className="space-y-2">
                <span className={labelClass}>Search</span>
                <input
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                            applyFilters()
                        }
                    }}
                    placeholder="incident, action, request message..."
                    className={inputClass}
                />
            </label>
            <label className="space-y-2">
                <span className={labelClass}>Status</span>
                <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className={inputClass}>
                    <option value="pending">Pending</option>
                    <option value="">All statuses</option>
                    <option value="approved">Approved</option>
                    <option value="rejected">Rejected</option>
                </select>
            </label>
            <div className="flex items-end">
                <button
                    onClick={applyFilters}
                    className="w-full rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-sky-700 dark:bg-sky-500 dark:hover:bg-sky-400"
                >
                    Apply Filters
                </button>
            </div>
        </div>
    </SectionCard>
)

const ApprovalActions: React.FC<{
    item: SREApprovalQueueItem
    mutatingApprovalId: string | null
    mutatingActionId: string | null
    onApprove: (item: SREApprovalQueueItem) => void
    onReject: (item: SREApprovalQueueItem) => void
    onExecute: (item: SREApprovalQueueItem) => void
}> = ({ item, mutatingApprovalId, mutatingActionId, onApprove, onReject, onExecute }) => {
    const pending = (item.approval.status || '').toLowerCase() === 'pending'
    const approved = (item.approval.status || '').toLowerCase() === 'approved'
    const executable = approved && actionExecutable(item) && item.action?.id
    const incidentHref = item.incident?.id
        ? `/admin/operations/sre-smart-bot?incident=${encodeURIComponent(item.incident.id)}`
        : '/admin/operations/sre-smart-bot'

    return (
        <div className="flex flex-wrap items-center gap-2">
            {pending ? (
                <>
                    <button
                        onClick={() => onApprove(item)}
                        disabled={mutatingApprovalId === item.approval.id}
                        className="rounded-lg bg-emerald-600 px-3 py-2 text-xs font-medium text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-emerald-500 dark:hover:bg-emerald-400"
                    >
                        {mutatingApprovalId === item.approval.id ? 'Updating...' : 'Approve'}
                    </button>
                    <button
                        onClick={() => onReject(item)}
                        disabled={mutatingApprovalId === item.approval.id}
                        className="rounded-lg border border-rose-300 bg-white px-3 py-2 text-xs font-medium text-rose-700 transition hover:bg-rose-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-rose-800 dark:bg-slate-950 dark:text-rose-200 dark:hover:bg-rose-950/30"
                    >
                        {mutatingApprovalId === item.approval.id ? 'Updating...' : 'Reject'}
                    </button>
                </>
            ) : null}
            {executable ? (
                <button
                    onClick={() => onExecute(item)}
                    disabled={mutatingActionId === item.action?.id}
                    className="rounded-lg bg-sky-600 px-3 py-2 text-xs font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-sky-500 dark:hover:bg-sky-400"
                >
                    {mutatingActionId === item.action?.id ? 'Executing...' : 'Execute'}
                </button>
            ) : null}
            <Link
                to={incidentHref}
                className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
            >
                Open Incident
            </Link>
        </div>
    )
}

export const ApprovalInboxPanel: React.FC<{
    loading: boolean
    error: string | null
    items: SREApprovalQueueItem[]
    total: number
    mutatingApprovalId: string | null
    mutatingActionId: string | null
    onApprove: (item: SREApprovalQueueItem) => void
    onReject: (item: SREApprovalQueueItem) => void
    onExecute: (item: SREApprovalQueueItem) => void
}> = ({
    loading,
    error,
    items,
    total,
    mutatingApprovalId,
    mutatingActionId,
    onApprove,
    onReject,
    onExecute,
}) => (
        <>
            <SectionCard title="Approval Inbox" subtitle="Pending approvals are floated to the top. Approve in place, then use the incident workspace for deeper action review.">
                {loading ? (
                    <div className="flex items-center justify-center rounded-xl border border-slate-200 bg-slate-50 px-4 py-16 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-900/60 dark:text-slate-300">
                        Loading approvals...
                    </div>
                ) : error ? (
                    <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-4 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/40 dark:text-rose-200">{error}</div>
                ) : items.length === 0 ? (
                    <EmptyState title="No approvals found" description="Once SRE Smart Bot requests operator approval, those requests will show up here." />
                ) : (
                    <div className="space-y-4">
                        {items.map((item) => (
                            <article key={item.approval.id} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-950/70">
                                <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
                                    <div className="min-w-0 space-y-3">
                                        <div className="flex flex-wrap items-center gap-2">
                                            <span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-medium ${approvalTone(item.approval.status)}`}>
                                                {item.approval.status || 'pending'}
                                            </span>
                                            {item.action ? (
                                                <span className="inline-flex rounded-full border border-slate-300 bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200">
                                                    {item.action.action_key}
                                                </span>
                                            ) : null}
                                            {actionExecutable(item) ? (
                                                <span className="inline-flex rounded-full border border-sky-300 bg-sky-50 px-2.5 py-1 text-xs font-medium text-sky-700 dark:border-sky-800 dark:bg-sky-950/30 dark:text-sky-200">
                                                    Executable
                                                </span>
                                            ) : (
                                                <span className="inline-flex rounded-full border border-slate-300 bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-600 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300">
                                                    Recommendation
                                                </span>
                                            )}
                                        </div>

                                        <div>
                                            <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                                                {item.incident?.display_name || item.incident?.incident_type || 'Approval request'}
                                            </h2>
                                            <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
                                                {item.approval.request_message || item.incident?.summary || 'No request message provided.'}
                                            </p>
                                        </div>

                                        <div className="grid gap-3 text-sm sm:grid-cols-2 xl:grid-cols-4">
                                            <div className="rounded-xl border border-slate-200 bg-slate-50 px-3 py-3 dark:border-slate-800 dark:bg-slate-900">
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Requested</div>
                                                <div className="mt-1 font-medium text-slate-900 dark:text-slate-100">{formatDateTime(item.approval.requested_at)}</div>
                                                <div className="mt-1 text-xs text-slate-500 dark:text-slate-400">{relativeTime(item.approval.requested_at)}</div>
                                            </div>
                                            <div className="rounded-xl border border-slate-200 bg-slate-50 px-3 py-3 dark:border-slate-800 dark:bg-slate-900">
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Channel</div>
                                                <div className="mt-1 font-medium text-slate-900 dark:text-slate-100">{item.approval.channel_provider_id || 'in-app-default'}</div>
                                            </div>
                                            <div className="rounded-xl border border-slate-200 bg-slate-50 px-3 py-3 dark:border-slate-800 dark:bg-slate-900">
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Target</div>
                                                <div className="mt-1 font-medium text-slate-900 dark:text-slate-100">{item.action?.target_ref || item.action?.target_kind || '—'}</div>
                                            </div>
                                            <div className="rounded-xl border border-slate-200 bg-slate-50 px-3 py-3 dark:border-slate-800 dark:bg-slate-900">
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Decision</div>
                                                <div className="mt-1 font-medium text-slate-900 dark:text-slate-100">{item.approval.decided_by || 'Awaiting operator'}</div>
                                                <div className="mt-1 text-xs text-slate-500 dark:text-slate-400">{item.approval.decided_at ? formatDateTime(item.approval.decided_at) : 'Pending'}</div>
                                            </div>
                                        </div>

                                        {item.incident?.summary ? (
                                            <div className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300">
                                                <span className="font-medium text-slate-900 dark:text-slate-100">Incident summary:</span> {item.incident.summary}
                                            </div>
                                        ) : null}
                                    </div>

                                    <div className="xl:w-64 xl:flex-shrink-0">
                                        <ApprovalActions
                                            item={item}
                                            mutatingApprovalId={mutatingApprovalId}
                                            mutatingActionId={mutatingActionId}
                                            onApprove={onApprove}
                                            onReject={onReject}
                                            onExecute={onExecute}
                                        />
                                    </div>
                                </div>
                            </article>
                        ))}
                    </div>
                )}
            </SectionCard>

            <div className="text-sm text-slate-500 dark:text-slate-400">
                Showing {items.length} of {total} approval records.
            </div>
        </>
    )