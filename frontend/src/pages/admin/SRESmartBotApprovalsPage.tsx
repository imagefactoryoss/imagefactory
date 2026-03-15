import { adminService } from '@/services/adminService'
import type { SREApprovalQueueItem } from '@/types'
import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import { Link } from 'react-router-dom'

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

const approvalTone = (status: string) => {
    switch ((status || '').toLowerCase()) {
        case 'approved':
            return 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-200'
        case 'rejected':
            return 'border-rose-200 bg-rose-50 text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/30 dark:text-rose-200'
        default:
            return 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200'
    }
}

const actionExecutable = (item: SREApprovalQueueItem) =>
    ['reconcile_tenant_assets', 'review_provider_connectivity'].includes(item.action?.action_key || '')

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

const SRESmartBotApprovalsPage: React.FC = () => {
    const [items, setItems] = useState<SREApprovalQueueItem[]>([])
    const [total, setTotal] = useState(0)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [statusFilter, setStatusFilter] = useState('pending')
    const [search, setSearch] = useState('')
    const [mutatingApprovalId, setMutatingApprovalId] = useState<string | null>(null)
    const [mutatingActionId, setMutatingActionId] = useState<string | null>(null)

    const pendingCount = useMemo(
        () => items.filter((item) => (item.approval.status || '').toLowerCase() === 'pending').length,
        [items],
    )

    const loadApprovals = async () => {
        try {
            setLoading(true)
            setError(null)
            const response = await adminService.getSREApprovals({
                limit: 100,
                offset: 0,
                status: statusFilter || undefined,
                search: search.trim() || undefined,
            })
            setItems(response.approvals || [])
            setTotal(response.total || 0)
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load SRE approvals')
            setItems([])
            setTotal(0)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        void loadApprovals()
    }, [statusFilter])

    const handleApprovalDecision = async (item: SREApprovalQueueItem, decision: 'approved' | 'rejected') => {
        const incidentId = item.incident?.id || item.approval.incident_id
        if (!incidentId) {
            toast.error('Approval is missing incident context')
            return
        }
        try {
            setMutatingApprovalId(item.approval.id)
            await adminService.decideSREApproval(incidentId, item.approval.id, { decision })
            toast.success(decision === 'approved' ? 'Approval granted' : 'Approval rejected')
            await loadApprovals()
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to update approval')
        } finally {
            setMutatingApprovalId(null)
        }
    }

    const handleExecuteAction = async (item: SREApprovalQueueItem) => {
        const incidentId = item.incident?.id || item.approval.incident_id
        const actionId = item.action?.id
        if (!incidentId || !actionId) {
            toast.error('Action is missing incident context')
            return
        }
        try {
            setMutatingActionId(actionId)
            await adminService.executeSREAction(incidentId, actionId)
            toast.success('Action executed')
            await loadApprovals()
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to execute action')
        } finally {
            setMutatingActionId(null)
        }
    }

    const renderActions = (approval: SREApprovalQueueItem) => {
        const pending = (approval.approval.status || '').toLowerCase() === 'pending'
        const approved = (approval.approval.status || '').toLowerCase() === 'approved'
        const executable = approved && actionExecutable(approval) && approval.action?.id
        const incidentHref = approval.incident?.id
            ? `/admin/operations/sre-smart-bot?incident=${encodeURIComponent(approval.incident.id)}`
            : '/admin/operations/sre-smart-bot'
        return (
            <div className="flex flex-wrap items-center gap-2">
                {pending ? (
                    <>
                        <button
                            onClick={() => void handleApprovalDecision(approval, 'approved')}
                            disabled={mutatingApprovalId === approval.approval.id}
                            className="rounded-lg bg-emerald-600 px-3 py-2 text-xs font-medium text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-emerald-500 dark:hover:bg-emerald-400"
                        >
                            {mutatingApprovalId === approval.approval.id ? 'Updating...' : 'Approve'}
                        </button>
                        <button
                            onClick={() => void handleApprovalDecision(approval, 'rejected')}
                            disabled={mutatingApprovalId === approval.approval.id}
                            className="rounded-lg border border-rose-300 bg-white px-3 py-2 text-xs font-medium text-rose-700 transition hover:bg-rose-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-rose-800 dark:bg-slate-950 dark:text-rose-200 dark:hover:bg-rose-950/30"
                        >
                            {mutatingApprovalId === approval.approval.id ? 'Updating...' : 'Reject'}
                        </button>
                    </>
                ) : null}
                {executable ? (
                    <button
                        onClick={() => void handleExecuteAction(approval)}
                        disabled={mutatingActionId === approval.action?.id}
                        className="rounded-lg bg-sky-600 px-3 py-2 text-xs font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-sky-500 dark:hover:bg-sky-400"
                    >
                        {mutatingActionId === approval.action?.id ? 'Executing...' : 'Execute'}
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

    return (
        <div className="min-h-full w-full bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.08),_transparent_30%),linear-gradient(180deg,_#f8fafc_0%,_#eef2ff_100%)] px-4 py-6 text-slate-900 sm:px-6 lg:px-8 dark:bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.16),_transparent_24%),linear-gradient(180deg,_#020617_0%,_#0f172a_100%)] dark:text-slate-100">
            <div className="w-full space-y-6">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                    <div>
                        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">Operations</p>
                        <h1 className="mt-2 text-3xl font-semibold tracking-tight">SRE Smart Bot Approvals</h1>
                        <p className="mt-2 max-w-3xl text-sm text-slate-600 dark:text-slate-400">
                            Triage pending approvals, grant or reject actions, and keep high-signal recovery work moving without digging through every incident thread.
                        </p>
                    </div>
                    <div className="flex items-center gap-3">
                        <Link
                            to="/admin/operations/sre-smart-bot"
                            className="rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                        >
                            Incidents
                        </Link>
                        <Link
                            to="/admin/operations/sre-smart-bot/settings"
                            className="rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                        >
                            Settings
                        </Link>
                        <div className="rounded-2xl border border-slate-200 bg-white/80 px-4 py-3 text-sm shadow-sm dark:border-slate-800 dark:bg-slate-900/70">
                            <div className="text-slate-500 dark:text-slate-400">Pending approvals</div>
                            <div className="mt-1 text-2xl font-semibold text-slate-900 dark:text-white">{pendingCount}</div>
                        </div>
                        <button
                            onClick={() => void loadApprovals()}
                            className="rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                        >
                            Refresh
                        </button>
                    </div>
                </div>

                <SectionCard title="Filters" subtitle="Focus on pending work first, or widen the view to inspect recent approval decisions.">
                    <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_220px_160px]">
                        <label className="space-y-2">
                            <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Search</span>
                            <input
                                value={search}
                                onChange={(e) => setSearch(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter') {
                                        void loadApprovals()
                                    }
                                }}
                                placeholder="incident, action, request message..."
                                className="w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900"
                            />
                        </label>
                        <label className="space-y-2">
                            <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Status</span>
                            <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className="w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900">
                                <option value="pending">Pending</option>
                                <option value="">All statuses</option>
                                <option value="approved">Approved</option>
                                <option value="rejected">Rejected</option>
                            </select>
                        </label>
                        <div className="flex items-end">
                            <button
                                onClick={() => void loadApprovals()}
                                className="w-full rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-sky-700 dark:bg-sky-500 dark:hover:bg-sky-400"
                            >
                                Apply Filters
                            </button>
                        </div>
                    </div>
                </SectionCard>

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
                                            {renderActions(item)}
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
            </div>
        </div>
    )
}

export default SRESmartBotApprovalsPage
