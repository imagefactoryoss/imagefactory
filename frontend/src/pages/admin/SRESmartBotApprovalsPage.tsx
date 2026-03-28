import { adminService } from '@/services/adminService'
import type { SREApprovalQueueItem } from '@/types'
import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import { Link } from 'react-router-dom'
import { ApprovalFiltersPanel, ApprovalInboxPanel } from './sreSmartBotApprovalsPanels'

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

                <ApprovalFiltersPanel
                    search={search}
                    setSearch={setSearch}
                    statusFilter={statusFilter}
                    setStatusFilter={setStatusFilter}
                    applyFilters={() => void loadApprovals()}
                />

                <ApprovalInboxPanel
                    loading={loading}
                    error={error}
                    items={items}
                    total={total}
                    mutatingApprovalId={mutatingApprovalId}
                    mutatingActionId={mutatingActionId}
                    onApprove={(item) => void handleApprovalDecision(item, 'approved')}
                    onReject={(item) => void handleApprovalDecision(item, 'rejected')}
                    onExecute={(item) => void handleExecuteAction(item)}
                />
            </div>
        </div>
    )
}

export default SRESmartBotApprovalsPage
