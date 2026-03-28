import type { SREDetectorRuleSuggestion } from '@/types'
import React from 'react'

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

export const suggestionTone = (suggestion: SREDetectorRuleSuggestion) => {
    switch ((suggestion.status || '').toLowerCase()) {
        case 'accepted':
            return 'border-emerald-200 bg-emerald-50 text-emerald-900 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-200'
        case 'rejected':
            return 'border-rose-200 bg-rose-50 text-rose-900 dark:border-rose-900/40 dark:bg-rose-950/30 dark:text-rose-200'
        default:
            return 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200'
    }
}

export const severityTone = (severity: string) => {
    switch ((severity || '').toLowerCase()) {
        case 'critical':
            return 'border-rose-200 bg-rose-50 text-rose-900 dark:border-rose-900/40 dark:bg-rose-950/30 dark:text-rose-200'
        case 'warning':
            return 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200'
        default:
            return 'border-sky-200 bg-sky-50 text-sky-900 dark:border-sky-900/40 dark:bg-sky-950/30 dark:text-sky-200'
    }
}

export const inputClass = 'w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900'
export const labelClass = 'text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400'

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