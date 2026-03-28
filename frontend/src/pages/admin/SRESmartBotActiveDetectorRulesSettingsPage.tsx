import { adminService } from '@/services/adminService'
import type { SRESmartBotDetectorRule, SRESmartBotOperatorRule, SRESmartBotPolicyConfig } from '@/types'
import React, { useEffect, useState } from 'react'
import toast from 'react-hot-toast'

import SRESmartBotRuleSettingsNav from './sreSmartBotRuleSettingsNav'
import { RulesSettingsPanel } from './sreSmartBotSettingsPanels'

import { DOMAIN_OPTIONS, normalizePolicy } from './sreSmartBotSettingsShared'

const SRESmartBotActiveDetectorRulesSettingsPage: React.FC = () => {
    const [policy, setPolicy] = useState<SRESmartBotPolicyConfig | null>(null)
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [ruleMutationInFlight, setRuleMutationInFlight] = useState(false)
    const [error, setError] = useState<string | null>(null)

    const loadPolicy = async () => {
        try {
            setLoading(true)
            setError(null)
            const response = await adminService.getSRESmartBotPolicy()
            setPolicy(normalizePolicy(response))
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load detector rules settings')
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        void loadPolicy()
    }, [])

    const savePolicy = async () => {
        if (!policy) return
        try {
            setSaving(true)
            setError(null)
            const response = await adminService.updateSRESmartBotPolicy(policy)
            setPolicy(normalizePolicy(response))
            toast.success('Detector rules saved')
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to save detector rules'
            setError(message)
            toast.error(message)
        } finally {
            setSaving(false)
        }
    }

    const updateRule = (index: number, updates: Partial<SRESmartBotOperatorRule>) => {
        setPolicy((current) => {
            if (!current) return current
            const operatorRules = [...current.operator_rules]
            operatorRules[index] = { ...operatorRules[index], ...updates }
            return { ...current, operator_rules: operatorRules }
        })
    }

    const updateDetectorRule = (index: number, updates: Partial<SRESmartBotDetectorRule>) => {
        setPolicy((current) => {
            if (!current) return current
            const detectorRules = [...current.detector_rules]
            detectorRules[index] = { ...detectorRules[index], ...updates }
            return { ...current, detector_rules: detectorRules }
        })
    }

    const persistDetectorRule = async (index: number, updates: Partial<SRESmartBotDetectorRule>) => {
        let nextPolicy: SRESmartBotPolicyConfig | null = null
        setPolicy((current) => {
            if (!current) return current
            const detectorRules = [...current.detector_rules]
            detectorRules[index] = { ...detectorRules[index], ...updates }
            nextPolicy = { ...current, detector_rules: detectorRules }
            return nextPolicy
        })

        if (!nextPolicy) return

        try {
            setRuleMutationInFlight(true)
            setSaving(true)
            setError(null)
            const response = await adminService.updateSRESmartBotPolicy(nextPolicy)
            setPolicy(normalizePolicy(response))
            toast.success('Detector rule saved')
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to persist detector rule update'
            setError(message)
            toast.error(message)
            void loadPolicy()
        } finally {
            setRuleMutationInFlight(false)
            setSaving(false)
        }
    }

    const addRule = () => {
        setPolicy((current) => {
            if (!current) return current
            const newRule: SRESmartBotOperatorRule = {
                id: crypto.randomUUID(),
                name: '',
                domain: DOMAIN_OPTIONS[0],
                incident_type: '',
                severity: 'warning',
                enabled: true,
                source: 'operator_defined',
            }
            return { ...current, operator_rules: [...current.operator_rules, newRule] }
        })
    }

    const addRuleFromJson = (rule: Partial<SRESmartBotOperatorRule>) => {
        setPolicy((current) => {
            if (!current) return current
            const severity = (rule.severity || '').toLowerCase()
            const normalized: SRESmartBotOperatorRule = {
                id: (rule.id || '').trim() || crypto.randomUUID(),
                name: (rule.name || '').trim(),
                domain: (rule.domain || '').trim() || DOMAIN_OPTIONS[0],
                incident_type: (rule.incident_type || '').trim(),
                severity: severity === 'info' || severity === 'warning' || severity === 'critical' ? severity : 'warning',
                enabled: rule.enabled ?? true,
                source: (rule.source || '').trim() || 'operator_defined',
                match_labels: rule.match_labels,
                threshold: Number.isFinite(Number(rule.threshold)) ? Number(rule.threshold) : undefined,
                for_duration_seconds: Number.isFinite(Number(rule.for_duration_seconds)) ? Number(rule.for_duration_seconds) : undefined,
                suggested_action: rule.suggested_action,
                auto_allowed: rule.auto_allowed,
            }
            return { ...current, operator_rules: [...current.operator_rules, normalized] }
        })
    }

    const removeRule = (index: number) => {
        setPolicy((current) => {
            if (!current) return current
            const operatorRules = [...current.operator_rules]
            operatorRules.splice(index, 1)
            return { ...current, operator_rules: operatorRules }
        })
    }

    return (
        <div className="min-h-full w-full bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.08),_transparent_30%),linear-gradient(180deg,_#f8fafc_0%,_#eef2ff_100%)] px-4 py-6 text-slate-900 sm:px-6 lg:px-8 dark:bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.16),_transparent_24%),linear-gradient(180deg,_#020617_0%,_#0f172a_100%)] dark:text-slate-100">
            <div className="w-full space-y-6">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                    <div>
                        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">Operations</p>
                        <h1 className="mt-2 text-3xl font-semibold tracking-tight">SRE Smart Bot Active Detector Rules</h1>
                        <p className="mt-2 max-w-4xl text-sm text-slate-600 dark:text-slate-400">
                            Manage active detector rules on a dedicated page with search, filters, pagination, and drawer-based editing.
                        </p>
                    </div>
                    <div className="flex items-center gap-3">
                        <button
                            type="button"
                            onClick={() => void loadPolicy()}
                            disabled={saving || ruleMutationInFlight}
                            className="rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                        >
                            Refresh
                        </button>
                        <button
                            type="button"
                            onClick={() => void savePolicy()}
                            disabled={!policy || saving || ruleMutationInFlight}
                            className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-sky-500 dark:hover:bg-sky-400"
                        >
                            {saving ? 'Saving...' : ruleMutationInFlight ? 'Rule Save In Progress...' : 'Save Settings'}
                        </button>
                    </div>
                </div>

                <SRESmartBotRuleSettingsNav active="detector" />

                {loading ? (
                    <div className="rounded-2xl border border-slate-200 bg-white/80 px-4 py-16 text-center text-sm text-slate-600 shadow-sm dark:border-slate-800 dark:bg-slate-900/80 dark:text-slate-300">
                        Loading active detector rules...
                    </div>
                ) : error ? (
                    <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-4 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/40 dark:text-rose-200">{error}</div>
                ) : policy ? (
                    <RulesSettingsPanel
                        policy={policy}
                        updateRule={updateRule}
                        updateDetectorRule={updateDetectorRule}
                        addRule={addRule}
                        addRuleFromJson={addRuleFromJson}
                        removeRule={removeRule}
                        showOperatorRules={false}
                        showDetectorRules
                        showTabIntro={false}
                        detectorMutationInFlight={ruleMutationInFlight}
                        onDetectorRuleSave={(index, updates) => {
                            void persistDetectorRule(index, updates)
                        }}
                    />
                ) : null}
            </div>
        </div>
    )
}

export default SRESmartBotActiveDetectorRulesSettingsPage
