import Drawer from '@/components/ui/Drawer'
import type { SRESmartBotOperatorRule } from '@/types'
import React from 'react'

import { parseLabelMap, stringifyLabelMap } from './sreSmartBotSettingsShared'

type SaveOperatorRule = (rule: SRESmartBotOperatorRule) => void

export const SRESmartBotOperatorRuleDrawer: React.FC<{
    isOpen: boolean
    onClose: () => void
    rule: SRESmartBotOperatorRule | null
    domainOptions: string[]
    onSave: SaveOperatorRule
}> = ({ isOpen, onClose, rule, domainOptions, onSave }) => {
    const [draftRule, setDraftRule] = React.useState<SRESmartBotOperatorRule | null>(rule)
    const [isEditing, setIsEditing] = React.useState(false)
    const [activeTab, setActiveTab] = React.useState<'details' | 'json'>('details')

    React.useEffect(() => {
        setDraftRule(rule ? { ...rule } : null)
        setIsEditing(false)
        setActiveTab('details')
    }, [rule, isOpen])

    if (!rule || !draftRule) return null

    const hasUnsavedChanges = JSON.stringify(draftRule) !== JSON.stringify(rule)

    const closeWithGuard = () => {
        if (isEditing && hasUnsavedChanges && !window.confirm('Discard unsaved changes?')) {
            return
        }
        onClose()
    }

    const cancelEdits = () => {
        if (hasUnsavedChanges && !window.confirm('Discard unsaved changes?')) {
            return
        }
        setDraftRule({ ...rule })
        setIsEditing(false)
    }

    const updateDraft = (updates: Partial<SRESmartBotOperatorRule>) => {
        setDraftRule((current) => (current ? { ...current, ...updates } : current))
    }

    const fieldLabel = 'text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400'
    const inputClass = 'w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900'
    const tabClass = (tab: 'details' | 'json') => `rounded-xl border px-3 py-2 text-sm font-medium transition ${activeTab === tab ? 'border-sky-300 bg-sky-50 text-sky-800 dark:border-sky-700 dark:bg-sky-950/30 dark:text-sky-200' : 'border-slate-300 bg-white text-slate-700 hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300'}`

    return (
        <Drawer
            isOpen={isOpen}
            onClose={closeWithGuard}
            title="View Operator Rule"
            description="Review details first, then switch to edit mode. Save updates in this drawer and use Save Settings on the page to persist to backend."
            width="2xl"
        >
            <div className="space-y-4">
                <div className="rounded-xl border border-slate-200 bg-slate-50/80 px-3 py-2 text-xs text-slate-600 dark:border-slate-800 dark:bg-slate-950/30 dark:text-slate-300">
                    Rule ID: <span className="font-mono">{draftRule.id}</span>
                </div>

                {!isEditing ? (
                    <div className="rounded-xl border border-sky-200 bg-sky-50/80 px-3 py-2 text-xs font-medium text-sky-800 dark:border-sky-900/50 dark:bg-sky-950/30 dark:text-sky-200">
                        View mode. Click Edit to modify this rule.
                    </div>
                ) : hasUnsavedChanges ? (
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
                                <input className={inputClass} disabled={!isEditing} value={draftRule.name} onChange={(e) => updateDraft({ name: e.target.value })} />
                            </label>
                            <label className="space-y-2">
                                <span className={fieldLabel}>Domain</span>
                                <select className={inputClass} disabled={!isEditing} value={draftRule.domain} onChange={(e) => updateDraft({ domain: e.target.value })}>
                                    {domainOptions.map((domain) => (
                                        <option key={domain} value={domain}>{domain}</option>
                                    ))}
                                </select>
                            </label>
                            <label className="space-y-2">
                                <span className={fieldLabel}>Incident Type</span>
                                <input className={inputClass} disabled={!isEditing} value={draftRule.incident_type} onChange={(e) => updateDraft({ incident_type: e.target.value })} />
                            </label>
                            <label className="space-y-2">
                                <span className={fieldLabel}>Severity</span>
                                <select className={inputClass} disabled={!isEditing} value={draftRule.severity || 'warning'} onChange={(e) => updateDraft({ severity: e.target.value })}>
                                    <option value="info">info</option>
                                    <option value="warning">warning</option>
                                    <option value="critical">critical</option>
                                </select>
                            </label>
                            <label className="space-y-2">
                                <span className={fieldLabel}>Threshold</span>
                                <input className={inputClass} disabled={!isEditing} type="number" min={0} value={draftRule.threshold || 0} onChange={(e) => updateDraft({ threshold: Number(e.target.value) || 0 })} />
                            </label>
                            <label className="space-y-2">
                                <span className={fieldLabel}>For Duration Seconds</span>
                                <input className={inputClass} disabled={!isEditing} type="number" min={0} value={draftRule.for_duration_seconds || 0} onChange={(e) => updateDraft({ for_duration_seconds: Number(e.target.value) || 0 })} />
                            </label>
                            <label className="space-y-2">
                                <span className={fieldLabel}>Source</span>
                                <input className={inputClass} disabled={!isEditing} value={draftRule.source || ''} onChange={(e) => updateDraft({ source: e.target.value })} />
                            </label>
                            <label className="space-y-2 md:col-span-2">
                                <span className={fieldLabel}>Suggested Action</span>
                                <input className={inputClass} disabled={!isEditing} value={draftRule.suggested_action || ''} onChange={(e) => updateDraft({ suggested_action: e.target.value })} />
                            </label>
                            <label className="space-y-2 md:col-span-2">
                                <span className={fieldLabel}>Match Labels</span>
                                <textarea
                                    className={`${inputClass} min-h-28`}
                                    disabled={!isEditing}
                                    value={stringifyLabelMap(draftRule.match_labels)}
                                    onChange={(e) => updateDraft({ match_labels: parseLabelMap(e.target.value) })}
                                    placeholder={'component=dispatcher\nnamespace=image-factory'}
                                />
                            </label>
                        </div>

                        <div className="grid gap-3 md:grid-cols-2">
                            <label className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white/80 px-3 py-2 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-950/30 dark:text-slate-300">
                                <input disabled={!isEditing} type="checkbox" checked={draftRule.enabled} onChange={(e) => updateDraft({ enabled: e.target.checked })} />
                                Rule enabled
                            </label>
                            <label className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white/80 px-3 py-2 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-950/30 dark:text-slate-300">
                                <input disabled={!isEditing} type="checkbox" checked={!!draftRule.auto_allowed} onChange={(e) => updateDraft({ auto_allowed: e.target.checked })} />
                                Auto allowed
                            </label>
                        </div>
                    </>
                ) : (
                    <div className="rounded-2xl border border-slate-200 bg-slate-950 p-0 dark:border-slate-800">
                        <div className="border-b border-slate-800 px-4 py-2 text-xs font-medium uppercase tracking-wide text-slate-400">
                            {isEditing ? 'Draft JSON' : 'Rule JSON'}
                        </div>
                        <pre className="max-h-[28rem] overflow-auto p-4 text-xs text-slate-100">
                            <code>{JSON.stringify(draftRule, null, 2)}</code>
                        </pre>
                    </div>
                )}

                <div className="flex items-center justify-end gap-2 border-t border-slate-200 pt-3 dark:border-slate-800">
                    {isEditing ? (
                        <>
                            <button
                                type="button"
                                onClick={cancelEdits}
                                className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 transition hover:border-slate-400 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200"
                            >
                                Cancel
                            </button>
                            <button
                                type="button"
                                disabled={!hasUnsavedChanges}
                                onClick={() => {
                                    onSave(draftRule)
                                    setIsEditing(false)
                                }}
                                className="rounded-lg bg-sky-600 px-3 py-1.5 text-sm font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-sky-500 dark:hover:bg-sky-400"
                            >
                                Save
                            </button>
                        </>
                    ) : (
                        <button
                            type="button"
                            onClick={() => setIsEditing(true)}
                            className="rounded-lg bg-sky-600 px-3 py-1.5 text-sm font-medium text-white transition hover:bg-sky-700 dark:bg-sky-500 dark:hover:bg-sky-400"
                        >
                            Edit
                        </button>
                    )}
                </div>
            </div>
        </Drawer>
    )
}
