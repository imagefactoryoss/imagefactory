import Drawer from '@/components/ui/Drawer'
import type { SRESmartBotChannelProvider } from '@/types'
import React from 'react'

import { parseLabelMap, stringifyLabelMap } from './sreSmartBotSettingsShared'

type SaveChannelProvider = (provider: SRESmartBotChannelProvider) => void

export const SRESmartBotChannelProviderDrawer: React.FC<{
    isOpen: boolean
    onClose: () => void
    provider: SRESmartBotChannelProvider | null
    providerKinds: string[]
    providerConfigRefHint: Record<string, string>
    onSave: SaveChannelProvider
}> = ({ isOpen, onClose, provider, providerKinds, providerConfigRefHint, onSave }) => {
    const [draftProvider, setDraftProvider] = React.useState<SRESmartBotChannelProvider | null>(provider)
    const [isEditing, setIsEditing] = React.useState(false)
    const [isSaving, setIsSaving] = React.useState(false)

    React.useEffect(() => {
        setDraftProvider(provider ? { ...provider, settings: { ...(provider.settings || {}) } } : null)
        setIsEditing(false)
        setIsSaving(false)
    }, [provider, isOpen])

    if (!provider || !draftProvider) return null

    const hasUnsavedChanges = JSON.stringify(draftProvider) !== JSON.stringify(provider)

    const closeWithGuard = () => {
        if (isSaving) {
            return
        }
        if (isEditing && hasUnsavedChanges && !window.confirm('Discard unsaved changes?')) {
            return
        }
        onClose()
    }

    const cancelEdits = () => {
        if (isSaving) {
            return
        }
        if (hasUnsavedChanges && !window.confirm('Discard unsaved changes?')) {
            return
        }
        setDraftProvider({ ...provider, settings: { ...(provider.settings || {}) } })
        setIsEditing(false)
    }

    const updateDraft = (updates: Partial<SRESmartBotChannelProvider>) => {
        setDraftProvider((current) => (current ? { ...current, ...updates } : current))
    }

    const getSetting = (key: string) => (draftProvider.settings || {})[key] || ''

    const updateSetting = (key: string, value: string) => {
        setDraftProvider((current) => {
            if (!current) return current
            const nextSettings = { ...(current.settings || {}) }
            const trimmed = value.trim()
            if (trimmed === '') {
                delete nextSettings[key]
            } else {
                nextSettings[key] = value
            }
            return { ...current, settings: nextSettings }
        })
    }

    const fieldLabel = 'text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400'
    const inputClass = 'w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900'

    return (
        <Drawer
            isOpen={isOpen}
            onClose={closeWithGuard}
            title="Channel Provider"
            description="Review and edit provider configuration. Saving in this drawer persists the provider immediately."
            width="2xl"
        >
            <div className="space-y-4">
                {!isEditing ? (
                    <div className="rounded-xl border border-sky-200 bg-sky-50/80 px-3 py-2 text-xs font-medium text-sky-800 dark:border-sky-900/50 dark:bg-sky-950/30 dark:text-sky-200">
                        View mode. Click Edit to modify this provider.
                    </div>
                ) : hasUnsavedChanges ? (
                    <div className="rounded-xl border border-amber-200 bg-amber-50/80 px-3 py-2 text-xs font-medium text-amber-800 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-200">
                        Unsaved changes
                    </div>
                ) : null}

                <div className="grid gap-4 md:grid-cols-2">
                    <label className="space-y-2">
                        <span className={fieldLabel}>Name</span>
                        <input className={inputClass} disabled={!isEditing} value={draftProvider.name} onChange={(e) => updateDraft({ name: e.target.value })} />
                    </label>
                    <label className="space-y-2">
                        <span className={fieldLabel}>Kind</span>
                        <select className={inputClass} disabled={!isEditing} value={draftProvider.kind} onChange={(e) => updateDraft({ kind: e.target.value })}>
                            {providerKinds.map((kind) => (
                                <option key={kind} value={kind}>{kind}</option>
                            ))}
                        </select>
                    </label>
                    <label className="space-y-2 md:col-span-2">
                        <span className={fieldLabel}>Config Ref</span>
                        <input
                            className={inputClass}
                            disabled={!isEditing}
                            value={draftProvider.config_ref || ''}
                            onChange={(e) => updateDraft({ config_ref: e.target.value })}
                            placeholder={providerConfigRefHint[draftProvider.kind] || providerConfigRefHint.custom}
                        />
                    </label>
                </div>

                <div className="grid gap-3 md:grid-cols-2">
                    <label className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white/80 px-3 py-2 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-950/30 dark:text-slate-300">
                        <input disabled={!isEditing} type="checkbox" checked={draftProvider.enabled} onChange={(e) => updateDraft({ enabled: e.target.checked })} />
                        Enabled
                    </label>
                    <label className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white/80 px-3 py-2 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-950/30 dark:text-slate-300">
                        <input disabled={!isEditing} type="checkbox" checked={!!draftProvider.supports_interactive_approval} onChange={(e) => updateDraft({ supports_interactive_approval: e.target.checked })} />
                        Interactive Approval
                    </label>
                </div>

                {draftProvider.kind === 'webhook' ? (
                    <div className="space-y-3 rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/30">
                        <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">Webhook Settings</div>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Webhook URL</span>
                            <input className={inputClass} disabled={!isEditing} value={getSetting('webhook_url')} onChange={(e) => updateSetting('webhook_url', e.target.value)} placeholder="https://hooks.example.com/sre" />
                        </label>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Auth Header</span>
                            <input className={inputClass} disabled={!isEditing} value={getSetting('auth_header')} onChange={(e) => updateSetting('auth_header', e.target.value)} placeholder="Bearer <token>" />
                        </label>
                    </div>
                ) : null}

                {draftProvider.kind === 'slack' ? (
                    <div className="space-y-3 rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/30">
                        <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">Slack Settings</div>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Channel</span>
                            <input className={inputClass} disabled={!isEditing} value={getSetting('channel')} onChange={(e) => updateSetting('channel', e.target.value)} placeholder="#ops-alerts" />
                        </label>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Bot Token Ref</span>
                            <input className={inputClass} disabled={!isEditing} value={getSetting('bot_token_ref')} onChange={(e) => updateSetting('bot_token_ref', e.target.value)} placeholder="secret://slack/bot_token" />
                        </label>
                    </div>
                ) : null}

                {draftProvider.kind === 'email' ? (
                    <div className="space-y-3 rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/30">
                        <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">Email Settings</div>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Recipient List</span>
                            <input className={inputClass} disabled={!isEditing} value={getSetting('recipients')} onChange={(e) => updateSetting('recipients', e.target.value)} placeholder="ops@acme.io,oncall@acme.io" />
                        </label>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Subject Prefix</span>
                            <input className={inputClass} disabled={!isEditing} value={getSetting('subject_prefix')} onChange={(e) => updateSetting('subject_prefix', e.target.value)} placeholder="[SRE Smart Bot]" />
                        </label>
                    </div>
                ) : null}

                {draftProvider.kind === 'telegram' ? (
                    <div className="space-y-3 rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/30">
                        <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">Telegram Settings</div>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Chat ID</span>
                            <input className={inputClass} disabled={!isEditing} value={getSetting('chat_id')} onChange={(e) => updateSetting('chat_id', e.target.value)} placeholder="-1001234567890" />
                        </label>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Bot Token Ref</span>
                            <input className={inputClass} disabled={!isEditing} value={getSetting('bot_token_ref')} onChange={(e) => updateSetting('bot_token_ref', e.target.value)} placeholder="secret://telegram/bot_token" />
                        </label>
                    </div>
                ) : null}

                {draftProvider.kind === 'whatsapp' ? (
                    <div className="space-y-3 rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/30">
                        <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">WhatsApp Settings</div>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Phone Number ID</span>
                            <input className={inputClass} disabled={!isEditing} value={getSetting('phone_number_id')} onChange={(e) => updateSetting('phone_number_id', e.target.value)} placeholder="1234567890" />
                        </label>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Template Namespace</span>
                            <input className={inputClass} disabled={!isEditing} value={getSetting('template_namespace')} onChange={(e) => updateSetting('template_namespace', e.target.value)} placeholder="incident_alerts" />
                        </label>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Access Token Ref</span>
                            <input className={inputClass} disabled={!isEditing} value={getSetting('access_token_ref')} onChange={(e) => updateSetting('access_token_ref', e.target.value)} placeholder="secret://whatsapp/access_token" />
                        </label>
                    </div>
                ) : null}

                {draftProvider.kind === 'teams' ? (
                    <div className="space-y-3 rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/30">
                        <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">Teams Settings</div>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Channel</span>
                            <input className={inputClass} disabled={!isEditing} value={getSetting('channel')} onChange={(e) => updateSetting('channel', e.target.value)} placeholder="SRE Alerts" />
                        </label>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Webhook URL</span>
                            <input className={inputClass} disabled={!isEditing} value={getSetting('webhook_url')} onChange={(e) => updateSetting('webhook_url', e.target.value)} placeholder="https://outlook.office.com/webhook/..." />
                        </label>
                    </div>
                ) : null}

                {draftProvider.kind === 'in_app' ? (
                    <div className="space-y-3 rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/30">
                        <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">In-App Settings</div>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Channel</span>
                            <input className={inputClass} disabled={!isEditing} value={getSetting('channel')} onChange={(e) => updateSetting('channel', e.target.value)} placeholder="incident-feed" />
                        </label>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Audience</span>
                            <input className={inputClass} disabled={!isEditing} value={getSetting('audience')} onChange={(e) => updateSetting('audience', e.target.value)} placeholder="ops-oncall" />
                        </label>
                    </div>
                ) : null}

                {draftProvider.kind === 'custom' ? (
                    <div className="space-y-3 rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/30">
                        <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">Custom Settings</div>
                        <label className="space-y-2">
                            <span className={fieldLabel}>Settings (key=value)</span>
                            <textarea
                                className={`${inputClass} min-h-28`}
                                disabled={!isEditing}
                                value={stringifyLabelMap(draftProvider.settings)}
                                onChange={(e) => updateDraft({ settings: parseLabelMap(e.target.value) })}
                                placeholder={'endpoint=https://custom-provider.local\ncredential_ref=secret://custom/provider'}
                            />
                        </label>
                    </div>
                ) : null}

                <div className="flex items-center justify-end gap-2 border-t border-slate-200 pt-3 dark:border-slate-800">
                    {isEditing ? (
                        <>
                            <button
                                type="button"
                                onClick={cancelEdits}
                                disabled={isSaving}
                                className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 transition hover:border-slate-400 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200"
                            >
                                Cancel
                            </button>
                            <button
                                type="button"
                                disabled={!hasUnsavedChanges || isSaving}
                                onClick={async () => {
                                    try {
                                        setIsSaving(true)
                                        await onSave(draftProvider)
                                        setIsEditing(false)
                                    } finally {
                                        setIsSaving(false)
                                    }
                                }}
                                className="rounded-lg bg-sky-600 px-3 py-1.5 text-sm font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-sky-500 dark:hover:bg-sky-400"
                            >
                                {isSaving ? 'Saving...' : 'Save'}
                            </button>
                        </>
                    ) : (
                        <button
                            type="button"
                            onClick={() => setIsEditing(true)}
                            disabled={isSaving}
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
