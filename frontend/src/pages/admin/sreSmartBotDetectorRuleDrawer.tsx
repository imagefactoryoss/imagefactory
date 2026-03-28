import Drawer from '@/components/ui/Drawer'
import type { SRESmartBotDetectorRule } from '@/types'
import React from 'react'

type SaveDetectorRule = (rule: SRESmartBotDetectorRule) => void

export const SRESmartBotDetectorRuleDrawer: React.FC<{
    isOpen: boolean
    onClose: () => void
    rule: SRESmartBotDetectorRule | null
    domainOptions: string[]
    onSave: SaveDetectorRule
}> = ({ isOpen, onClose, rule, domainOptions, onSave }) => {
    const [draftRule, setDraftRule] = React.useState<SRESmartBotDetectorRule | null>(rule)
    const [activeTab, setActiveTab] = React.useState<'details' | 'json'>('details')

    React.useEffect(() => {
        setDraftRule(rule ? { ...rule } : null)
        setActiveTab('details')
    }, [rule, isOpen])

    if (!rule || !draftRule) return null

    const hasUnsavedChanges = JSON.stringify(draftRule) !== JSON.stringify(rule)

    const closeWithGuard = () => {
        if (hasUnsavedChanges && !window.confirm('Discard unsaved changes?')) {
            return
        }
        onClose()
    }

    const updateDraft = (updates: Partial<SRESmartBotDetectorRule>) => {
        setDraftRule((current) => (current ? { ...current, ...updates } : current))
    }

    const fieldLabel = 'text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400'
    const inputClass = 'w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900'
    const tabClass = (tab: 'details' | 'json') => `rounded-xl border px-3 py-2 text-sm font-medium transition ${activeTab === tab ? 'border-sky-300 bg-sky-50 text-sky-800 dark:border-sky-700 dark:bg-sky-950/30 dark:text-sky-200' : 'border-slate-300 bg-white text-slate-700 hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300'}`

    return (
        <Drawer
            isOpen={isOpen}
            onClose={closeWithGuard}
            title="Edit Detector Rule"
            description="Edit detector behavior in this drawer, then click Save Changes. Use Save Settings on the page to persist to backend."
            width="2xl"
        >
            <div className="space-y-4">
                <div className="rounded-xl border border-slate-200 bg-slate-50/80 px-3 py-2 text-xs text-slate-600 dark:border-slate-800 dark:bg-slate-950/30 dark:text-slate-300">
                    Rule ID: <span className="font-mono">{draftRule.id}</span>
                </div>

                {hasUnsavedChanges ? (
                    <div className="rounded-xl border border-amber-200 bg-amber-50/80 px-3 py-2 text-xs font-medium text-amber-800 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-200">
                        Unsaved changes
                    </div>
                ) : null}

                <div className="flex flex-wrap items-center gap-2">
                    <button type="button" className={tabClass('details')} onClick={() => setActiveTab('details')}>
                        Details
                    </button>
                    <button type="button" className={tabClass('json')} onClick={() => setActiveTab('json')}>
                        JSON
                    </button>
                </div>

                {activeTab === 'details' ? (
                    <>
                        <div className="grid gap-4 md:grid-cols-2">
                            <label className="space-y-2">
                                <span className={fieldLabel}>Name</span>
                                <input className={inputClass} value={draftRule.name} onChange={(e) => updateDraft({ name: e.target.value })} />
                            </label>
                            <label className="space-y-2">
                                <span className={fieldLabel}>Domain</span>
                                <select className={inputClass} value={draftRule.domain} onChange={(e) => updateDraft({ domain: e.target.value })}>
                                    {domainOptions.map((domain) => (
                                        <option key={domain} value={domain}>{domain}</option>
                                    ))}
                                </select>
                            </label>
                            <label className="space-y-2">
                                <span className={fieldLabel}>Incident Type</span>
                                <input className={inputClass} value={draftRule.incident_type} onChange={(e) => updateDraft({ incident_type: e.target.value })} />
                            </label>
                            <label className="space-y-2">
                                <span className={fieldLabel}>Severity</span>
                                <select className={inputClass} value={draftRule.severity} onChange={(e) => updateDraft({ severity: e.target.value })}>
                                    <option value="info">info</option>
                                    <option value="warning">warning</option>
                                    <option value="critical">critical</option>
                                </select>
                            </label>
                            <label className="space-y-2">
                                <span className={fieldLabel}>Confidence</span>
                                <select className={inputClass} value={draftRule.confidence || ''} onChange={(e) => updateDraft({ confidence: e.target.value || undefined })}>
                                    <option value="">(none)</option>
                                    <option value="low">low</option>
                                    <option value="medium">medium</option>
                                    <option value="high">high</option>
                                </select>
                            </label>
                            <label className="space-y-2">
                                <span className={fieldLabel}>Threshold</span>
                                <input className={inputClass} type="number" min={0} value={draftRule.threshold || 0} onChange={(e) => updateDraft({ threshold: Number(e.target.value) || 0 })} />
                            </label>
                            <label className="space-y-2">
                                <span className={fieldLabel}>Signal Key</span>
                                <input className={inputClass} value={draftRule.signal_key || ''} onChange={(e) => updateDraft({ signal_key: e.target.value })} />
                            </label>
                            <label className="space-y-2">
                                <span className={fieldLabel}>Source</span>
                                <input className={inputClass} value={draftRule.source || ''} onChange={(e) => updateDraft({ source: e.target.value })} />
                            </label>
                            <label className="space-y-2 md:col-span-2">
                                <span className={fieldLabel}>Suggested Action</span>
                                <input className={inputClass} value={draftRule.suggested_action || ''} onChange={(e) => updateDraft({ suggested_action: e.target.value })} />
                            </label>
                            <label className="space-y-2 md:col-span-2">
                                <span className={fieldLabel}>Loki Query</span>
                                <textarea className={`${inputClass} min-h-28`} value={draftRule.query} onChange={(e) => updateDraft({ query: e.target.value })} />
                            </label>
                        </div>

                        <div className="rounded-xl border border-slate-200 bg-white/80 p-3 dark:border-slate-800 dark:bg-slate-950/30">
                            <label className="inline-flex items-center gap-2 text-sm text-slate-700 dark:text-slate-300">
                                <input type="checkbox" checked={draftRule.enabled} onChange={(e) => updateDraft({ enabled: e.target.checked })} />
                                Rule enabled
                            </label>
                        </div>
                    </>
                ) : (
                    <div className="rounded-2xl border border-slate-200 bg-slate-950 p-0 dark:border-slate-800">
                        <div className="border-b border-slate-800 px-4 py-2 text-xs font-medium uppercase tracking-wide text-slate-400">
                            {hasUnsavedChanges ? 'Draft JSON' : 'Rule JSON'}
                        </div>
                        <pre className="max-h-[28rem] overflow-auto p-4 text-xs text-slate-100">
                            <code>{JSON.stringify(draftRule, null, 2)}</code>
                        </pre>
                    </div>
                )}

                <div className="flex items-center justify-end gap-2 border-t border-slate-200 pt-3 dark:border-slate-800">
                    <button
                        type="button"
                        onClick={closeWithGuard}
                        className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 transition hover:border-slate-400 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200"
                    >
                        Cancel
                    </button>
                    <button
                        type="button"
                        disabled={!hasUnsavedChanges}
                        onClick={() => {
                            onSave(draftRule)
                            onClose()
                        }}
                        className="rounded-lg bg-sky-600 px-3 py-1.5 text-sm font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-sky-500 dark:hover:bg-sky-400"
                    >
                        Save Changes
                    </button>
                </div>
            </div>
        </Drawer>
    )
}
