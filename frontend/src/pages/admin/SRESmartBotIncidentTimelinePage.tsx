import { adminService } from '@/services/adminService'
import type { SREActionAttempt, SREApproval, SREEvidence, SREFinding, SREIncident } from '@/types'
import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import { Link, useParams } from 'react-router-dom'

import { EmptyState, SectionCard } from './sreSmartBotIncidentPageShared'
import { SREIncidentTimeline } from './sreSmartBotIncidentTimeline'

const SRESmartBotIncidentTimelinePage: React.FC = () => {
    const { incidentId = '' } = useParams<{ incidentId: string }>()
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [incident, setIncident] = useState<SREIncident | null>(null)
    const [findings, setFindings] = useState<SREFinding[]>([])
    const [evidence, setEvidence] = useState<SREEvidence[]>([])
    const [actions, setActions] = useState<SREActionAttempt[]>([])
    const [approvals, setApprovals] = useState<SREApproval[]>([])

    useEffect(() => {
        if (!incidentId) return
        const loadIncidentTimeline = async () => {
            try {
                setLoading(true)
                setError(null)
                const response = await adminService.getSREIncident(incidentId)
                setIncident(response.incident)
                setFindings(response.findings || [])
                setEvidence(response.evidence || [])
                setActions(response.action_attempts || [])
                setApprovals(response.approvals || [])
            } catch (err) {
                const message = err instanceof Error ? err.message : 'Failed to load incident timeline'
                setError(message)
                toast.error(message)
            } finally {
                setLoading(false)
            }
        }
        void loadIncidentTimeline()
    }, [incidentId])

    const actionsById = useMemo(() => {
        return actions.reduce<Record<string, SREActionAttempt>>((acc, action) => {
            acc[action.id] = action
            return acc
        }, {})
    }, [actions])

    return (
        <div className="min-h-full w-full bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.08),_transparent_30%),linear-gradient(180deg,_#f8fafc_0%,_#eef2ff_100%)] px-4 py-6 text-slate-900 sm:px-6 lg:px-8 dark:bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.16),_transparent_24%),linear-gradient(180deg,_#020617_0%,_#0f172a_100%)] dark:text-slate-100">
            <div className="w-full space-y-6">
                <div className="flex flex-wrap items-center justify-between gap-3">
                    <div>
                        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">Operations</p>
                        <h1 className="mt-2 text-3xl font-semibold tracking-tight">Incident Timeline</h1>
                        <p className="mt-2 max-w-3xl text-sm text-slate-600 dark:text-slate-400">
                            Full timeline view for {incident?.display_name || incidentId}, including findings, evidence, actions, and approvals.
                        </p>
                    </div>
                    <Link
                        to={`/admin/operations/sre-smart-bot?incident=${encodeURIComponent(incidentId)}`}
                        className="inline-flex h-10 items-center justify-center rounded-xl border border-slate-300 bg-white px-4 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                    >
                        Back To Incident
                    </Link>
                </div>

                <SectionCard title="Timeline Activity" subtitle="Filter by actor or event type to narrow the incident history.">
                    {loading ? (
                        <div className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-10 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-900/60 dark:text-slate-300">
                            Loading timeline...
                        </div>
                    ) : error ? (
                        <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-4 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/40 dark:text-rose-200">
                            {error}
                        </div>
                    ) : incident ? (
                        <SREIncidentTimeline
                            findings={findings}
                            evidence={evidence}
                            actions={actions}
                            approvals={approvals}
                            actionsById={actionsById}
                        />
                    ) : (
                        <EmptyState title="Incident not found" description="The requested incident could not be loaded." />
                    )}
                </SectionCard>
            </div>
        </div>
    )
}

export default SRESmartBotIncidentTimelinePage
