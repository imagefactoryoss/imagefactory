import { adminService } from '@/services/adminService'
import type { SREDetectorRuleSuggestion, SRESmartBotPolicyConfig } from '@/types'
import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import { Link, useSearchParams } from 'react-router-dom'
import {
    DetectorRuleFiltersPanel,
    DetectorRuleSuggestionInbox,
    LearningPosturePanel,
} from './sreSmartBotDetectorRulesPanels'

const SRESmartBotDetectorRulesPage: React.FC = () => {
    const [searchParams] = useSearchParams()
    const [policy, setPolicy] = useState<SRESmartBotPolicyConfig | null>(null)
    const [suggestions, setSuggestions] = useState<SREDetectorRuleSuggestion[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [statusFilter, setStatusFilter] = useState('pending')
    const [search, setSearch] = useState('')
    const [mutatingSuggestionId, setMutatingSuggestionId] = useState<string | null>(null)
    const focusedSuggestionId = searchParams.get('suggestion') || ''
    const focusedIncidentId = searchParams.get('incident') || ''

    const activeCount = useMemo(() => (policy?.detector_rules || []).filter((rule) => rule.enabled).length, [policy])
    const pendingCount = useMemo(() => suggestions.filter((suggestion) => suggestion.status === 'pending').length, [suggestions])

    const loadData = async (overrides?: { status?: string; search?: string }) => {
        try {
            setLoading(true)
            setError(null)
            const effectiveStatus = overrides?.status ?? statusFilter
            const effectiveSearch = overrides?.search ?? search
            const [policyResponse, suggestionsResponse] = await Promise.all([
                adminService.getSRESmartBotPolicy(),
                adminService.getSREDetectorRuleSuggestions({
                    status: effectiveStatus || undefined,
                    search: effectiveSearch.trim() || undefined,
                    limit: 100,
                    offset: 0,
                }),
            ])
            setPolicy(policyResponse)
            setSuggestions(suggestionsResponse.suggestions || [])
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load detector rules')
            setSuggestions([])
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        if (focusedSuggestionId) {
            setStatusFilter('')
        }
        if (focusedIncidentId) {
            setSearch(focusedIncidentId)
        }
        if (focusedSuggestionId || focusedIncidentId) {
            void loadData({
                status: focusedSuggestionId ? '' : statusFilter,
                search: focusedIncidentId || search,
            })
        }
    }, [focusedIncidentId, focusedSuggestionId])

    useEffect(() => {
        void loadData()
    }, [statusFilter])

    const handleAccept = async (suggestion: SREDetectorRuleSuggestion) => {
        try {
            setMutatingSuggestionId(suggestion.id)
            await adminService.acceptSREDetectorRuleSuggestion(suggestion.id)
            toast.success('Detector rule activated')
            await loadData()
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to accept suggestion')
        } finally {
            setMutatingSuggestionId(null)
        }
    }

    const handleReject = async (suggestion: SREDetectorRuleSuggestion) => {
        const reason = window.prompt('Optional rejection reason', suggestion.reason || '')
        try {
            setMutatingSuggestionId(suggestion.id)
            await adminService.rejectSREDetectorRuleSuggestion(suggestion.id, reason || undefined)
            toast.success('Detector rule suggestion rejected')
            await loadData()
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to reject suggestion')
        } finally {
            setMutatingSuggestionId(null)
        }
    }

    return (
        <div className="min-h-full w-full bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.08),_transparent_30%),linear-gradient(180deg,_#f8fafc_0%,_#eef2ff_100%)] px-4 py-6 text-slate-900 sm:px-6 lg:px-8 dark:bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.16),_transparent_24%),linear-gradient(180deg,_#020617_0%,_#0f172a_100%)] dark:text-slate-100">
            <div className="w-full space-y-6">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                    <div>
                        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">Operations</p>
                        <h1 className="mt-2 text-3xl font-semibold tracking-tight">SRE Smart Bot Detector Rules</h1>
                        <p className="mt-2 max-w-4xl text-sm text-slate-600 dark:text-slate-400">
                            Review learned detector-rule suggestions, promote good ones into active policy, and control whether SRE Smart Bot only suggests or auto-creates rules during training.
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
                        <button
                            onClick={() => void loadData()}
                            className="rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                        >
                            Refresh
                        </button>
                    </div>
                </div>

                <LearningPosturePanel policy={policy} pendingCount={pendingCount} activeCount={activeCount} />

                <DetectorRuleFiltersPanel
                    search={search}
                    setSearch={setSearch}
                    statusFilter={statusFilter}
                    setStatusFilter={setStatusFilter}
                    applyFilters={() => void loadData()}
                />

                <DetectorRuleSuggestionInbox
                    policy={policy}
                    loading={loading}
                    error={error}
                    suggestions={suggestions}
                    focusedSuggestionId={focusedSuggestionId}
                    focusedIncidentId={focusedIncidentId}
                    mutatingSuggestionId={mutatingSuggestionId}
                    onAccept={(suggestion) => void handleAccept(suggestion)}
                    onReject={(suggestion) => void handleReject(suggestion)}
                />
            </div>
        </div>
    )
}

export default SRESmartBotDetectorRulesPage
