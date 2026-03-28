import type { SREDetectorRuleSuggestion, SRESmartBotPolicyConfig } from '@/types'
import React from 'react'
import { Link } from 'react-router-dom'

import {
    EmptyState,
    formatDateTime,
    inputClass,
    labelClass,
    relativeTime,
    SectionCard,
    severityTone,
    suggestionTone,
} from './sreSmartBotDetectorRulesShared'

export const LearningPosturePanel: React.FC<{
    policy: SRESmartBotPolicyConfig | null
    pendingCount: number
    activeCount: number
}> = ({ policy, pendingCount, activeCount }) => (
    <SectionCard title="Learning Posture" subtitle="Current detector-learning mode and active detector coverage from policy.">
        <div className="grid gap-4 md:grid-cols-3">
            <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Learning Mode</div>
                <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{policy?.detector_learning_mode || 'suggest_only'}</div>
                <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Configured in SRE Smart Bot settings.</div>
            </div>
            <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Pending Suggestions</div>
                <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{pendingCount}</div>
                <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Awaiting operator review.</div>
            </div>
            <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Active Detector Rules</div>
                <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{activeCount}</div>
                <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Custom detector rules currently merged into runtime policy.</div>
            </div>
        </div>
        <div className="mt-4 rounded-2xl border border-slate-200 bg-slate-50/80 px-4 py-3 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300">
            <span className="font-medium text-slate-900 dark:text-slate-100">Recommendation:</span> keep production on <span className="font-medium">suggest_only</span>. Use <span className="font-medium">training_auto_create</span> only in controlled environments where rapid detector learning is worth the extra noise risk.
        </div>
    </SectionCard>
)

export const DetectorRuleFiltersPanel: React.FC<{
    search: string
    setSearch: (value: string) => void
    statusFilter: string
    setStatusFilter: (value: string) => void
    applyFilters: () => void
}> = ({ search, setSearch, statusFilter, setStatusFilter, applyFilters }) => (
    <SectionCard title="Filters" subtitle="Focus on pending items first, or widen the view to inspect accepted and rejected rule history.">
        <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_220px_160px]">
            <label className="space-y-2">
                <span className={labelClass}>Search</span>
                <input
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    onKeyDown={(e) => {
                        if (e.key === 'Enter') applyFilters()
                    }}
                    placeholder="rule, incident type, signal key..."
                    className={inputClass}
                />
            </label>
            <label className="space-y-2">
                <span className={labelClass}>Status</span>
                <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className={inputClass}>
                    <option value="pending">Pending</option>
                    <option value="">All statuses</option>
                    <option value="accepted">Accepted</option>
                    <option value="rejected">Rejected</option>
                </select>
            </label>
            <div className="flex items-end">
                <button onClick={applyFilters} className="w-full rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-sky-700 dark:bg-sky-500 dark:hover:bg-sky-400">
                    Apply Filters
                </button>
            </div>
        </div>
    </SectionCard>
)

export const DetectorRuleSuggestionInbox: React.FC<{
    policy: SRESmartBotPolicyConfig | null
    loading: boolean
    error: string | null
    suggestions: SREDetectorRuleSuggestion[]
    focusedSuggestionId: string
    focusedIncidentId: string
    mutatingSuggestionId: string | null
    onAccept: (suggestion: SREDetectorRuleSuggestion) => void
    onReject: (suggestion: SREDetectorRuleSuggestion) => void
}> = ({
    policy,
    loading,
    error,
    suggestions,
    focusedSuggestionId,
    focusedIncidentId,
    mutatingSuggestionId,
    onAccept,
    onReject,
}) => (
        <SectionCard title="Suggestion Inbox" subtitle="Pending suggestions can be promoted into active detector rules or rejected with a reviewer note.">
            {focusedSuggestionId || focusedIncidentId ? (
                <div className="mb-4 rounded-2xl border border-sky-200 bg-sky-50/90 px-4 py-3 text-sm text-sky-900 dark:border-sky-900/40 dark:bg-sky-950/30 dark:text-sky-200">
                    {focusedSuggestionId ? (
                        <span>
                            Showing a targeted detector-rule suggestion from the incident workspace. Review it here and decide whether it should become active policy.
                        </span>
                    ) : (
                        <span>
                            Filtering suggestions using the selected incident context so you can review learned detector rules tied to that thread.
                        </span>
                    )}
                </div>
            ) : null}
            {loading ? (
                <div className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-16 text-center text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-900/60 dark:text-slate-300">
                    Loading detector-rule suggestions...
                </div>
            ) : error ? (
                <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-4 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/40 dark:text-rose-200">{error}</div>
            ) : suggestions.length === 0 ? (
                <EmptyState title="No detector-rule suggestions found" description="As SRE Smart Bot learns from incidents, new detector-rule suggestions will show up here for review." />
            ) : (
                <div className="space-y-4">
                    {suggestions.map((suggestion) => (
                        <article
                            key={suggestion.id}
                            className={`rounded-2xl border bg-white p-5 shadow-sm dark:bg-slate-950/70 ${suggestion.id === focusedSuggestionId
                                    ? 'border-sky-400 ring-2 ring-sky-200 dark:border-sky-500 dark:ring-sky-900/60'
                                    : 'border-slate-200 dark:border-slate-800'
                                }`}
                        >
                            <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
                                <div className="min-w-0 flex-1 space-y-3">
                                    <div className="flex flex-wrap items-center gap-2">
                                        <span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-medium ${suggestionTone(suggestion)}`}>
                                            {suggestion.status}
                                        </span>
                                        <span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-medium ${severityTone(suggestion.severity)}`}>
                                            {suggestion.severity}
                                        </span>
                                        {suggestion.auto_created ? (
                                            <span className="inline-flex rounded-full border border-slate-300 bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200">
                                                {policy?.detector_learning_mode === 'training_auto_create' ? 'auto-activated' : 'learned'}
                                            </span>
                                        ) : null}
                                    </div>
                                    <div>
                                        <div className="text-lg font-semibold text-slate-900 dark:text-white">{suggestion.name}</div>
                                        <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">{suggestion.description}</div>
                                    </div>
                                    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                                        <div className="rounded-xl border border-slate-200 bg-slate-50 p-3 text-sm dark:border-slate-800 dark:bg-slate-950">
                                            <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Domain</div>
                                            <div className="mt-1 font-medium text-slate-900 dark:text-slate-100">{suggestion.domain}</div>
                                        </div>
                                        <div className="rounded-xl border border-slate-200 bg-slate-50 p-3 text-sm dark:border-slate-800 dark:bg-slate-950">
                                            <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Incident Type</div>
                                            <div className="mt-1 font-medium text-slate-900 dark:text-slate-100">{suggestion.incident_type}</div>
                                        </div>
                                        <div className="rounded-xl border border-slate-200 bg-slate-50 p-3 text-sm dark:border-slate-800 dark:bg-slate-950">
                                            <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Signal Key</div>
                                            <div className="mt-1 font-medium text-slate-900 dark:text-slate-100">{suggestion.signal_key || '—'}</div>
                                        </div>
                                        <div className="rounded-xl border border-slate-200 bg-slate-50 p-3 text-sm dark:border-slate-800 dark:bg-slate-950">
                                            <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Created</div>
                                            <div className="mt-1 font-medium text-slate-900 dark:text-slate-100">{relativeTime(suggestion.created_at)}</div>
                                        </div>
                                    </div>
                                    <div className="rounded-xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                        <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Suggested Loki Query</div>
                                        <pre className="mt-2 overflow-x-auto whitespace-pre-wrap break-all text-xs text-slate-700 dark:text-slate-300">{suggestion.query}</pre>
                                    </div>
                                    {suggestion.reason ? (
                                        <div className="text-sm text-slate-600 dark:text-slate-400">
                                            <span className="font-medium text-slate-900 dark:text-slate-100">Reason:</span> {suggestion.reason}
                                        </div>
                                    ) : null}
                                    {suggestion.auto_created ? (
                                        <div className="rounded-xl border border-violet-200 bg-violet-50/90 px-4 py-3 text-sm text-violet-900 dark:border-violet-900/40 dark:bg-violet-950/30 dark:text-violet-200">
                                            {policy?.detector_learning_mode === 'training_auto_create'
                                                ? 'This rule was auto-activated because detector learning is running in training mode. Review the generated query and keep or refine it before promoting the same posture into production.'
                                                : 'This suggestion was learned from repeated incident signals. It still requires explicit admin acceptance before it becomes active policy.'}
                                        </div>
                                    ) : null}
                                    {suggestion.incident_id ? (
                                        <div>
                                            <Link
                                                to={`/admin/operations/sre-smart-bot?incident=${encodeURIComponent(suggestion.incident_id)}`}
                                                className="inline-flex rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                                            >
                                                Open Incident
                                            </Link>
                                        </div>
                                    ) : null}
                                </div>
                                <div className="w-full shrink-0 xl:w-72">
                                    <div className="rounded-2xl border border-slate-200 bg-slate-50/90 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                        <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Review</div>
                                        <div className="mt-2 space-y-2 text-sm text-slate-600 dark:text-slate-400">
                                            <div>Source: <span className="font-medium text-slate-900 dark:text-slate-100">{suggestion.source}</span></div>
                                            <div>Confidence: <span className="font-medium text-slate-900 dark:text-slate-100">{suggestion.confidence}</span></div>
                                            <div>Reviewed: <span className="font-medium text-slate-900 dark:text-slate-100">{formatDateTime(suggestion.reviewed_at)}</span></div>
                                            <div>Activated Rule: <span className="font-medium text-slate-900 dark:text-slate-100">{suggestion.activated_rule_id || '—'}</span></div>
                                        </div>
                                        {suggestion.status === 'pending' ? (
                                            <div className="mt-4 flex flex-wrap gap-2">
                                                <button
                                                    onClick={() => onAccept(suggestion)}
                                                    disabled={mutatingSuggestionId === suggestion.id}
                                                    className="rounded-lg bg-emerald-600 px-3 py-2 text-xs font-medium text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-emerald-500 dark:hover:bg-emerald-400"
                                                >
                                                    {mutatingSuggestionId === suggestion.id ? 'Updating...' : 'Accept & Activate'}
                                                </button>
                                                <button
                                                    onClick={() => onReject(suggestion)}
                                                    disabled={mutatingSuggestionId === suggestion.id}
                                                    className="rounded-lg border border-rose-300 bg-white px-3 py-2 text-xs font-medium text-rose-700 transition hover:bg-rose-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-rose-800 dark:bg-slate-950 dark:text-rose-300 dark:hover:bg-rose-950/30"
                                                >
                                                    {mutatingSuggestionId === suggestion.id ? 'Updating...' : 'Reject'}
                                                </button>
                                            </div>
                                        ) : null}
                                    </div>
                                </div>
                            </div>
                        </article>
                    ))}
                </div>
            )}
        </SectionCard>
    )