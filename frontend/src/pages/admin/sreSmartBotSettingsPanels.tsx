import type { SREAgentRuntimeProbeResponse, SRESmartBotChannelProvider, SRESmartBotPolicyConfig } from '@/types'
import React from 'react'
import { Link } from 'react-router-dom'
import { SRESmartBotChannelProviderDrawer } from './sreSmartBotChannelProviderDrawer'
import { SRESmartBotDetectorRuleDrawer } from './sreSmartBotDetectorRuleDrawer'
import { SRESmartBotOperatorRuleDrawer } from './sreSmartBotOperatorRuleDrawer'

import {
    AGENT_PROVIDER_OPTIONS,
    DETECTOR_LEARNING_MODE_OPTIONS,
    DOMAIN_OPTIONS,
    ENVIRONMENT_OPTIONS,
    inputClass,
    labelClass,
    MCP_KIND_OPTIONS,
    MCP_TRANSPORT_OPTIONS,
    SectionCard,
    SETTINGS_TABS,
    ToggleField,
    type AddRule,
    type AddRuleFromJson,
    type RemoveRule,
    type UpdateAgentRuntime,
    type UpdateChannelProvider,
    type UpdateDetectorRule,
    type UpdateMCPServer,
    type UpdateOperatorRuleWithIndex,
    type UpdatePolicy,
    type UpdateRule,
} from './sreSmartBotSettingsShared'

const TabIntro: React.FC<{ tabId: (typeof SETTINGS_TABS)[number]['id'] }> = ({ tabId }) => (
    <div className="rounded-2xl border border-slate-200 bg-slate-50/90 px-4 py-3 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300">
        {SETTINGS_TABS.find((tab) => tab.id === tabId)?.description}
    </div>
)

const AddCustomDomainInput: React.FC<{ onAdd: (domain: string) => void }> = ({ onAdd }) => {
    const [value, setValue] = React.useState('')
    const handleAdd = () => {
        const trimmed = value.trim().toLowerCase().replace(/\s+/g, '_')
        if (trimmed) {
            onAdd(trimmed)
            setValue('')
        }
    }
    return (
        <div className="flex gap-2">
            <input
                className={`${inputClass} flex-1`}
                value={value}
                onChange={(e) => setValue(e.target.value)}
                onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                        e.preventDefault()
                        handleAdd()
                    }
                }}
                placeholder="e.g. data_pipelines"
            />
            <button
                type="button"
                onClick={handleAdd}
                className="shrink-0 rounded-xl border border-sky-300 bg-sky-50 px-3 py-2 text-sm font-medium text-sky-700 transition hover:bg-sky-100 dark:border-sky-700 dark:bg-sky-950/30 dark:text-sky-300 dark:hover:bg-sky-900/40"
            >
                Add
            </button>
        </div>
    )
}

export const GeneralSettingsPanel: React.FC<{
    policy: SRESmartBotPolicyConfig
    updatePolicy: UpdatePolicy
}> = ({ policy, updatePolicy }) => (
    <div role="tabpanel" id="sre-settings-panel-general" aria-labelledby="sre-settings-tab-general" className="space-y-6">
        <TabIntro tabId="general" />
        <SectionCard title="Identity" subtitle="Define the runtime posture and operator-facing identity for the bot.">
            <div className="grid gap-4 lg:grid-cols-4">
                <label className="space-y-2 lg:col-span-2">
                    <span className={labelClass}>Display Name</span>
                    <input className={inputClass} value={policy.display_name} onChange={(e) => updatePolicy('display_name', e.target.value)} />
                </label>
                <label className="space-y-2">
                    <span className={labelClass}>Environment Mode</span>
                    <select className={inputClass} value={policy.environment_mode} onChange={(e) => updatePolicy('environment_mode', e.target.value)}>
                        {ENVIRONMENT_OPTIONS.map((option) => (
                            <option key={option} value={option}>{option}</option>
                        ))}
                    </select>
                </label>
                <label className="space-y-2">
                    <span className={labelClass}>Detector Learning Mode</span>
                    <select className={inputClass} value={policy.detector_learning_mode || 'suggest_only'} onChange={(e) => updatePolicy('detector_learning_mode', e.target.value as SRESmartBotPolicyConfig['detector_learning_mode'])}>
                        {DETECTOR_LEARNING_MODE_OPTIONS.map((option) => (
                            <option key={option} value={option}>{option}</option>
                        ))}
                    </select>
                </label>
                <label className="space-y-2">
                    <span className={labelClass}>Default Provider</span>
                    <select className={inputClass} value={policy.default_channel_provider_id || ''} onChange={(e) => updatePolicy('default_channel_provider_id', e.target.value)}>
                        {policy.channel_providers.map((provider) => (
                            <option key={provider.id} value={provider.id}>{provider.name}</option>
                        ))}
                    </select>
                </label>
            </div>
            <div className="mt-4 grid gap-4 md:grid-cols-2">
                <ToggleField label="Bot Enabled" description="Turn the subsystem on or off without deleting policy or history." checked={policy.enabled} onChange={(value) => updatePolicy('enabled', value)} />
                <ToggleField label="Auto Observe" description="Allow background watchers to automatically open and refresh incident threads." checked={policy.auto_observe_enabled} onChange={(value) => updatePolicy('auto_observe_enabled', value)} />
            </div>
            <div className="mt-4 rounded-2xl border border-slate-200 bg-slate-50/80 p-4 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300">
                <div className="font-medium text-slate-900 dark:text-slate-100">Detector learning behavior</div>
                <div className="mt-2 space-y-2">
                    <div><span className="font-medium">disabled</span>: do not generate learned detector-rule suggestions.</div>
                    <div><span className="font-medium">suggest_only</span>: propose new detector rules for admin review before activation.</div>
                    <div><span className="font-medium">training_auto_create</span>: automatically activate learned rules as part of training mode.</div>
                </div>
            </div>
        </SectionCard>

        <SectionCard title="Automation" subtitle="Control how aggressively the bot notifies, contains, and proposes recovery.">
            <div className="grid gap-4 md:grid-cols-2">
                <ToggleField label="Auto Notify" description="Send notifications automatically when incidents are detected or updated." checked={policy.auto_notify_enabled} onChange={(value) => updatePolicy('auto_notify_enabled', value)} />
                <ToggleField label="Auto Contain" description="Allow containment-oriented actions to be proposed automatically when policy allows." checked={policy.auto_contain_enabled} onChange={(value) => updatePolicy('auto_contain_enabled', value)} />
                <ToggleField label="Auto Recover" description="Allow the bot to move from containment into recovery-oriented actions when enabled by policy." checked={policy.auto_recover_enabled} onChange={(value) => updatePolicy('auto_recover_enabled', value)} />
                <ToggleField label="Approval For Recovery" description="Keep recovery actions behind approval even when automatic recovery is enabled." checked={policy.require_approval_for_recover} onChange={(value) => updatePolicy('require_approval_for_recover', value)} />
                <ToggleField label="Approval For Disruptive" description="Require approval for disruptive actions like restarting, draining, or replacing infrastructure." checked={policy.require_approval_for_disruptive} onChange={(value) => updatePolicy('require_approval_for_disruptive', value)} />
            </div>
            <div className="mt-4 grid gap-4 md:grid-cols-2">
                <label className="space-y-2">
                    <span className={labelClass}>Duplicate Alert Suppression Seconds</span>
                    <input className={inputClass} type="number" min={0} value={policy.duplicate_alert_suppression_seconds} onChange={(e) => updatePolicy('duplicate_alert_suppression_seconds', Number(e.target.value) || 0)} />
                </label>
                <label className="space-y-2">
                    <span className={labelClass}>Action Cooldown Seconds</span>
                    <input className={inputClass} type="number" min={0} value={policy.action_cooldown_seconds} onChange={(e) => updatePolicy('action_cooldown_seconds', Number(e.target.value) || 0)} />
                </label>
            </div>
        </SectionCard>

        <SectionCard title="Domains" subtitle="Choose which incident domains SRE Smart Bot is allowed to observe and act on in this environment. Custom domains can be added below.">
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                {DOMAIN_OPTIONS.map((domain) => {
                    const enabled = policy.enabled_domains.includes(domain)
                    return (
                        <button
                            key={domain}
                            type="button"
                            onClick={() => updatePolicy('enabled_domains', enabled ? policy.enabled_domains.filter((item) => item !== domain) : [...policy.enabled_domains, domain])}
                            className={`rounded-xl border px-4 py-3 text-left transition ${enabled ? 'border-sky-300 bg-sky-50 text-sky-900 dark:border-sky-700 dark:bg-sky-950/30 dark:text-sky-100' : 'border-slate-200 bg-white text-slate-700 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300'}`}
                        >
                            <div className="font-medium">{domain}</div>
                            <div className="mt-1 text-xs opacity-80">{enabled ? 'Enabled for observation and policy decisions' : 'Disabled for this environment'}</div>
                        </button>
                    )
                })}
            </div>
            <div className="mt-5 space-y-3 border-t border-slate-200 pt-4 dark:border-slate-800">
                <div className={labelClass}>Custom Domains</div>
                <div className="flex flex-wrap gap-2 empty:hidden">
                    {policy.enabled_domains
                        .filter((d) => !DOMAIN_OPTIONS.includes(d))
                        .map((domain) => (
                            <span key={domain} className="inline-flex items-center gap-1.5 rounded-full border border-violet-200 bg-violet-50 px-3 py-1 text-sm font-medium text-violet-800 dark:border-violet-700 dark:bg-violet-950/30 dark:text-violet-200">
                                {domain}
                                <button
                                    type="button"
                                    onClick={() => updatePolicy('enabled_domains', policy.enabled_domains.filter((d) => d !== domain))}
                                    className="ml-0.5 rounded-full text-violet-600 transition hover:text-violet-900 dark:text-violet-400 dark:hover:text-violet-100"
                                    aria-label={`Remove domain ${domain}`}
                                >
                                    ×
                                </button>
                            </span>
                        ))}
                </div>
                <AddCustomDomainInput
                    onAdd={(domain) => {
                        if (!policy.enabled_domains.includes(domain)) {
                            updatePolicy('enabled_domains', [...policy.enabled_domains, domain])
                        }
                    }}
                />
            </div>
        </SectionCard>
    </div>
)

export const ChannelProvidersPanel: React.FC<{
    policy: SRESmartBotPolicyConfig
    updateChannelProvider: UpdateChannelProvider
    addChannelProvider: (kind: string) => void
    removeChannelProvider: (index: number) => void
    onChannelProviderSave?: (index: number, provider: SRESmartBotChannelProvider) => Promise<void> | void
    isMutating?: boolean
}> = ({ policy, updateChannelProvider, addChannelProvider, removeChannelProvider, onChannelProviderSave, isMutating = false }) => {
    const [selectedProviderIndex, setSelectedProviderIndex] = React.useState<number | null>(null)

    const providerKinds = ['in_app', 'email', 'webhook', 'slack', 'teams', 'telegram', 'whatsapp', 'custom']
    const providerConfigRefHint: Record<string, string> = {
        in_app: 'in_app://default',
        email: 'email://ops-oncall',
        webhook: 'webhook://incident-hook',
        slack: 'slack://workspace/channel',
        teams: 'teams://tenant/channel',
        telegram: 'telegram://bot/channel',
        whatsapp: 'whatsapp://business/template',
        custom: 'custom://provider/config',
    }

    const selectedProvider = selectedProviderIndex !== null ? policy.channel_providers[selectedProviderIndex] || null : null

    const saveSelectedProvider = async (provider: SRESmartBotChannelProvider) => {
        if (selectedProviderIndex === null) return
        if (onChannelProviderSave) {
            await onChannelProviderSave(selectedProviderIndex, provider)
            return
        }
        updateChannelProvider(selectedProviderIndex, provider)
    }

    const removeProviderAndUpdateSelection = (index: number) => {
        removeChannelProvider(index)
        if (selectedProviderIndex === index) {
            setSelectedProviderIndex(null)
            return
        }
        if (selectedProviderIndex !== null && selectedProviderIndex > index) {
            setSelectedProviderIndex(selectedProviderIndex - 1)
        }
    }

    return (
        <div role="tabpanel" id="sre-settings-panel-channels" aria-labelledby="sre-settings-tab-channels" className="space-y-6">
            <TabIntro tabId="channels" />
            <SectionCard title="Channel Providers" subtitle="Operator channels are provider-based, so enterprises can use in-app, email, webhooks, or internal integrations without assuming Telegram or WhatsApp.">
                {isMutating ? (
                    <div className="mb-4 rounded-2xl border border-sky-200 bg-sky-50/80 px-4 py-3 text-sm text-sky-800 dark:border-sky-900/50 dark:bg-sky-950/30 dark:text-sky-200">
                        Saving channel provider changes...
                    </div>
                ) : null}
                <div className="mb-4 space-y-3 rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/30">
                    <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">Add Provider</div>
                    <div className="flex flex-wrap items-center gap-2">
                        <button type="button" onClick={() => addChannelProvider('email')} className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300">+ Email</button>
                        <button type="button" onClick={() => addChannelProvider('slack')} className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300">+ Slack</button>
                        <button type="button" onClick={() => addChannelProvider('teams')} className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300">+ Teams</button>
                        <button type="button" onClick={() => addChannelProvider('telegram')} className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300">+ Telegram</button>
                        <button type="button" onClick={() => addChannelProvider('whatsapp')} className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300">+ WhatsApp</button>
                        <button type="button" onClick={() => addChannelProvider('webhook')} className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300">+ Webhook</button>
                        <button type="button" onClick={() => addChannelProvider('in_app')} className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300">+ In-App</button>
                        <button type="button" onClick={() => addChannelProvider('custom')} className="rounded-lg border border-dashed border-sky-400 bg-sky-50/60 px-3 py-1.5 text-xs font-medium text-sky-700 transition hover:border-sky-500 hover:bg-sky-100 dark:border-sky-700 dark:bg-sky-950/20 dark:text-sky-300 dark:hover:border-sky-500 dark:hover:bg-sky-950/40">+ Custom</button>
                    </div>
                </div>
                {policy.channel_providers.length === 0 ? (
                    <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-4 py-6 text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-950/30 dark:text-slate-300">
                        No channel providers configured yet. Add one of the provider templates above.
                    </div>
                ) : (
                    <div className="space-y-3">
                        <div className="overflow-x-auto rounded-2xl border border-slate-200 dark:border-slate-800">
                            <table className="min-w-full divide-y divide-slate-200 text-sm dark:divide-slate-800">
                                <thead className="bg-slate-50 dark:bg-slate-900/60">
                                    <tr>
                                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Name</th>
                                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Kind</th>
                                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Config Ref</th>
                                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Settings</th>
                                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Status</th>
                                        <th className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Action</th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y divide-slate-100 bg-white dark:divide-slate-800 dark:bg-slate-950/30">
                                    {policy.channel_providers.map((provider, index) => {
                                        const settingsCount = Object.keys(provider.settings || {}).length
                                        return (
                                            <tr key={provider.id} className="align-top hover:bg-slate-50/70 dark:hover:bg-slate-900/40">
                                                <td className="px-3 py-3 text-sm font-medium text-slate-900 dark:text-slate-100">{provider.name}</td>
                                                <td className="px-3 py-3">
                                                    <span className="rounded-full border border-slate-300 bg-slate-100 px-2 py-0.5 text-[11px] font-medium uppercase tracking-wide text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300">
                                                        {provider.kind}
                                                    </span>
                                                </td>
                                                <td className="max-w-sm truncate px-3 py-3 text-xs text-slate-600 dark:text-slate-300">{provider.config_ref || 'Not set'}</td>
                                                <td className="px-3 py-3 text-xs text-slate-600 dark:text-slate-300">{settingsCount > 0 ? `${settingsCount} configured` : 'No settings'}</td>
                                                <td className="px-3 py-3">
                                                    <div className="flex items-center gap-2">
                                                        <span
                                                            className={`rounded-full px-2 py-0.5 text-[11px] font-medium ${provider.enabled
                                                                ? 'border border-emerald-300 bg-emerald-100 text-emerald-800 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-200'
                                                                : 'border border-slate-300 bg-slate-100 text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300'
                                                                }`}
                                                        >
                                                            {provider.enabled ? 'Enabled' : 'Disabled'}
                                                        </span>
                                                        {provider.supports_interactive_approval ? (
                                                            <span className="rounded-full border border-sky-300 bg-sky-100 px-2 py-0.5 text-[11px] font-medium text-sky-800 dark:border-sky-700 dark:bg-sky-950/40 dark:text-sky-200">
                                                                Interactive
                                                            </span>
                                                        ) : null}
                                                    </div>
                                                </td>
                                                <td className="px-3 py-3 text-right">
                                                    <div className="flex justify-end gap-2">
                                                        <button
                                                            type="button"
                                                            onClick={() => setSelectedProviderIndex(index)}
                                                            className="rounded-lg border border-sky-300 bg-sky-50 px-2.5 py-1 text-xs font-medium text-sky-700 transition hover:border-sky-400 hover:bg-sky-100 dark:border-sky-700 dark:bg-sky-950/40 dark:text-sky-300 dark:hover:border-sky-600"
                                                        >
                                                            View
                                                        </button>
                                                        <button
                                                            type="button"
                                                            onClick={() => removeProviderAndUpdateSelection(index)}
                                                            className="rounded-lg border border-rose-300 bg-white px-2.5 py-1 text-xs font-medium text-rose-700 transition hover:border-rose-400 hover:bg-rose-50 dark:border-rose-800 dark:bg-slate-900 dark:text-rose-300 dark:hover:border-rose-700 dark:hover:bg-rose-950/30"
                                                        >
                                                            Remove
                                                        </button>
                                                    </div>
                                                </td>
                                            </tr>
                                        )
                                    })}
                                </tbody>
                            </table>
                        </div>
                        <div className="text-xs text-slate-500 dark:text-slate-400">Add, remove, and drawer edits all persist immediately. Use the inline saving notice above to track provider-only writes separately from page-level saves.</div>
                    </div>
                )}
            </SectionCard>

            <SRESmartBotChannelProviderDrawer
                isOpen={selectedProviderIndex !== null}
                onClose={() => setSelectedProviderIndex(null)}
                provider={selectedProvider}
                providerKinds={providerKinds}
                providerConfigRefHint={providerConfigRefHint}
                onSave={saveSelectedProvider}
            />
        </div>
    )
}

export const MCPAISettingsPanel: React.FC<{
    policy: SRESmartBotPolicyConfig
    recommendedPolicy: SRESmartBotPolicyConfig | null
    showRecommendedAgentRuntimeHint: boolean
    applyRecommendedAgentRuntime: () => void
    probeAgentRuntime: () => void
    probing: boolean
    probeResult: SREAgentRuntimeProbeResponse | null
    updateMCPServer: UpdateMCPServer
    updateAgentRuntime: UpdateAgentRuntime
}> = ({
    policy,
    recommendedPolicy,
    showRecommendedAgentRuntimeHint,
    applyRecommendedAgentRuntime,
    probeAgentRuntime,
    probing,
    probeResult,
    updateMCPServer,
    updateAgentRuntime,
}) => (
        <div role="tabpanel" id="sre-settings-panel-mcp-ai" aria-labelledby="sre-settings-tab-mcp-ai" className="space-y-6">
            <TabIntro tabId="mcp-ai" />
            <SectionCard title="MCP Servers" subtitle="Define the tool boundaries the future standalone SRE Smart Bot runtime can use for investigation and bounded tool calls.">
                <div className="space-y-4">
                    {policy.mcp_servers.map((server, index) => (
                        <div key={server.id} className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                            <div className="grid gap-4 xl:grid-cols-4">
                                <label className="space-y-2">
                                    <span className={labelClass}>Name</span>
                                    <input className={inputClass} value={server.name} onChange={(e) => updateMCPServer(index, { name: e.target.value })} />
                                </label>
                                <label className="space-y-2">
                                    <span className={labelClass}>Kind</span>
                                    <select className={inputClass} value={server.kind} onChange={(e) => updateMCPServer(index, { kind: e.target.value })}>
                                        {MCP_KIND_OPTIONS.map((option) => (
                                            <option key={option} value={option}>{option}</option>
                                        ))}
                                    </select>
                                </label>
                                <label className="space-y-2">
                                    <span className={labelClass}>Transport</span>
                                    <select className={inputClass} value={server.transport} onChange={(e) => updateMCPServer(index, { transport: e.target.value })}>
                                        {MCP_TRANSPORT_OPTIONS.map((option) => (
                                            <option key={option} value={option}>{option}</option>
                                        ))}
                                    </select>
                                </label>
                                <label className="space-y-2">
                                    <span className={labelClass}>Config Ref</span>
                                    <input className={inputClass} value={server.config_ref || ''} onChange={(e) => updateMCPServer(index, { config_ref: e.target.value })} />
                                </label>
                                <label className="space-y-2 xl:col-span-2">
                                    <span className={labelClass}>Endpoint</span>
                                    <input className={inputClass} value={server.endpoint || ''} onChange={(e) => updateMCPServer(index, { endpoint: e.target.value })} placeholder="http://ops.example.com/mcp/observability" />
                                </label>
                                <label className="space-y-2 xl:col-span-2">
                                    <span className={labelClass}>Allowed Tools</span>
                                    <textarea
                                        className={`${inputClass} min-h-24`}
                                        value={(server.allowed_tools || []).join('\n')}
                                        onChange={(e) => updateMCPServer(index, { allowed_tools: e.target.value.split('\n').map((item) => item.trim()).filter(Boolean) })}
                                        placeholder={'incidents.list\nevidence.list\nruntime_health.get'}
                                    />
                                </label>
                            </div>
                            <div className="mt-4 grid gap-3 lg:grid-cols-3">
                                <ToggleField label="Enabled" description="Allow this MCP server to be used by the agent/runtime." checked={server.enabled} onChange={(value) => updateMCPServer(index, { enabled: value })} />
                                <ToggleField label="Read Only" description="Mark this server as evidence-only with no mutation authority." checked={!!server.read_only} onChange={(value) => updateMCPServer(index, { read_only: value })} />
                                <ToggleField label="Approval Required" description="Require an approval path before this server can be used for sensitive tool flows." checked={!!server.approval_required} onChange={(value) => updateMCPServer(index, { approval_required: value })} />
                            </div>
                        </div>
                    ))}
                </div>
            </SectionCard>

            <SectionCard title="Agent Runtime" subtitle="Control the AI feature layer for summaries, hypotheses, and draft plans while keeping action authority deterministic.">
                {showRecommendedAgentRuntimeHint ? (
                    <div className="mb-4 flex flex-col gap-3 rounded-2xl border border-amber-200 bg-amber-50/90 p-4 lg:flex-row lg:items-center lg:justify-between dark:border-amber-900/50 dark:bg-amber-950/30">
                        <div>
                            <div className="text-sm font-semibold text-amber-950 dark:text-amber-100">Deployment Runtime Recommendation</div>
                            <div className="mt-1 text-sm text-amber-800 dark:text-amber-200">
                                This deployment recommends <span className="font-medium">{recommendedPolicy?.agent_runtime.provider}</span> at <span className="font-medium">{recommendedPolicy?.agent_runtime.base_url}</span> using <span className="font-medium">{recommendedPolicy?.agent_runtime.model}</span>, but the saved policy currently differs.
                            </div>
                        </div>
                        <button
                            type="button"
                            onClick={applyRecommendedAgentRuntime}
                            className="rounded-xl border border-amber-300 bg-white px-4 py-2 text-sm font-medium text-amber-800 transition hover:border-amber-500 hover:text-amber-900 dark:border-amber-700 dark:bg-slate-900 dark:text-amber-300 dark:hover:border-amber-500 dark:hover:text-amber-200"
                        >
                            Use Recommended Defaults
                        </button>
                    </div>
                ) : null}
                <div className="mb-4 flex flex-col gap-3 rounded-2xl border border-emerald-200 bg-emerald-50/80 p-4 sm:flex-row sm:items-center sm:justify-between dark:border-emerald-900/50 dark:bg-emerald-950/30">
                    <div>
                        <div className="text-sm font-semibold text-emerald-950 dark:text-emerald-100">Model Connectivity Check</div>
                        <div className="mt-1 text-sm text-emerald-800 dark:text-emerald-200">
                            Test the configured provider, base URL, and model inventory from here. Results appear below in this section.
                        </div>
                    </div>
                    <button
                        type="button"
                        onClick={probeAgentRuntime}
                        disabled={probing}
                        className="rounded-xl border border-emerald-300 bg-white px-4 py-2 text-sm font-medium text-emerald-700 transition hover:border-emerald-500 hover:text-emerald-800 disabled:cursor-not-allowed disabled:opacity-60 dark:border-emerald-700 dark:bg-slate-900 dark:text-emerald-300 dark:hover:border-emerald-500 dark:hover:text-emerald-200"
                    >
                        {probing ? 'Testing Model...' : 'Test Model'}
                    </button>
                </div>
                <div className="grid gap-4 lg:grid-cols-4">
                    <label className="space-y-2">
                        <span className={labelClass}>Provider</span>
                        <select className={inputClass} value={policy.agent_runtime.provider || 'ollama'} onChange={(e) => updateAgentRuntime({ provider: e.target.value })}>
                            {AGENT_PROVIDER_OPTIONS.map((option) => (
                                <option key={option} value={option}>{option}</option>
                            ))}
                        </select>
                    </label>
                    <label className="space-y-2 lg:col-span-2">
                        <span className={labelClass}>Model</span>
                        <input className={inputClass} value={policy.agent_runtime.model || ''} onChange={(e) => updateAgentRuntime({ model: e.target.value })} placeholder="llama3.2:3b" />
                    </label>
                    <label className="space-y-2">
                        <span className={labelClass}>Base URL</span>
                        <input className={inputClass} value={policy.agent_runtime.base_url || ''} onChange={(e) => updateAgentRuntime({ base_url: e.target.value })} placeholder="http://127.0.0.1:11434" />
                    </label>
                    <label className="space-y-2">
                        <span className={labelClass}>System Prompt Ref</span>
                        <input className={inputClass} value={policy.agent_runtime.system_prompt_ref || ''} onChange={(e) => updateAgentRuntime({ system_prompt_ref: e.target.value })} />
                    </label>
                    <label className="space-y-2">
                        <span className={labelClass}>Max Tool Calls Per Turn</span>
                        <input className={inputClass} type="number" min={1} value={policy.agent_runtime.max_tool_calls_per_turn} onChange={(e) => updateAgentRuntime({ max_tool_calls_per_turn: Number(e.target.value) || 1 })} />
                    </label>
                    <label className="space-y-2">
                        <span className={labelClass}>Max Incidents Per Summary</span>
                        <input className={inputClass} type="number" min={1} value={policy.agent_runtime.max_incidents_per_summary} onChange={(e) => updateAgentRuntime({ max_incidents_per_summary: Number(e.target.value) || 1 })} />
                    </label>
                </div>
                <p className="mt-4 text-sm text-slate-600 dark:text-slate-400">
                    Recommended local profile: <span className="font-medium text-slate-900 dark:text-slate-100">Ollama + <code className="rounded bg-slate-100 px-1 py-0.5 text-xs dark:bg-slate-800">llama3.2:3b</code></span>. It is small enough for local interpretation work while staying useful for operator summaries and grounded incident explanation.
                </p>
                {probeResult ? (
                    <div className={`mt-4 rounded-2xl border p-4 ${probeResult.healthy ? 'border-emerald-300 bg-emerald-50 text-emerald-950 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-100' : probeResult.status === 'skipped' ? 'border-amber-300 bg-amber-50 text-amber-950 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-100' : 'border-rose-300 bg-rose-50 text-rose-950 dark:border-rose-800 dark:bg-rose-950/30 dark:text-rose-100'}`}>
                        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                            <div>
                                <div className="text-sm font-semibold">Model Probe: {probeResult.status}</div>
                                <div className="mt-1 text-sm opacity-90">{probeResult.message}</div>
                            </div>
                            {probeResult.latency_ms ? <div className="text-sm font-medium opacity-80">{probeResult.latency_ms} ms</div> : null}
                        </div>
                        <div className="mt-3 grid gap-3 md:grid-cols-3">
                            <div className="rounded-xl border border-current/15 bg-white/60 p-3 dark:bg-slate-950/30">
                                <div className="text-xs uppercase tracking-wide opacity-70">Provider</div>
                                <div className="mt-1 text-sm font-medium">{probeResult.provider || 'n/a'}</div>
                            </div>
                            <div className="rounded-xl border border-current/15 bg-white/60 p-3 dark:bg-slate-950/30">
                                <div className="text-xs uppercase tracking-wide opacity-70">Model</div>
                                <div className="mt-1 text-sm font-medium">{probeResult.model || 'n/a'}</div>
                            </div>
                            <div className="rounded-xl border border-current/15 bg-white/60 p-3 dark:bg-slate-950/30">
                                <div className="text-xs uppercase tracking-wide opacity-70">Base URL</div>
                                <div className="mt-1 truncate text-sm font-medium">{probeResult.base_url || 'n/a'}</div>
                            </div>
                        </div>
                        <div className="mt-3 grid gap-3 md:grid-cols-2">
                            <div className="rounded-xl border border-current/15 bg-white/60 p-3 dark:bg-slate-950/30">
                                <div className="text-xs uppercase tracking-wide opacity-70">Model Installed</div>
                                <div className="mt-1 text-sm font-medium">{probeResult.model_installed ? 'Yes' : 'No / Unknown'}</div>
                            </div>
                            <div className="rounded-xl border border-current/15 bg-white/60 p-3 dark:bg-slate-950/30">
                                <div className="text-xs uppercase tracking-wide opacity-70">Installed Models</div>
                                <div className="mt-1 text-sm font-medium">{probeResult.installed_models?.length ? probeResult.installed_models.join(', ') : 'No inventory returned'}</div>
                            </div>
                        </div>
                        {probeResult.sample_response ? (
                            <div className="mt-3 rounded-xl border border-current/15 bg-white/60 p-3 dark:bg-slate-950/30">
                                <div className="text-xs uppercase tracking-wide opacity-70">Sample Response</div>
                                <div className="mt-1 text-sm">{probeResult.sample_response}</div>
                            </div>
                        ) : null}
                        {probeResult.guidance?.length ? (
                            <div className="mt-3 rounded-xl border border-current/15 bg-white/60 p-3 dark:bg-slate-950/30">
                                <div className="text-xs uppercase tracking-wide opacity-70">Guidance</div>
                                <div className="mt-2 space-y-2 text-sm">
                                    {probeResult.guidance.map((item, index) => (
                                        <div key={`${item}-${index}`}>- {item}</div>
                                    ))}
                                </div>
                            </div>
                        ) : null}
                    </div>
                ) : null}
                <div className="mt-4 grid gap-4 md:grid-cols-2">
                    <ToggleField label="Agent Runtime Enabled" description="Enable the AI feature layer for evidence summarization and guided investigation." checked={policy.agent_runtime.enabled} onChange={(value) => updateAgentRuntime({ enabled: value })} />
                    <ToggleField label="Operator Summaries" description="Allow the agent layer to generate operator-facing incident summaries." checked={policy.agent_runtime.operator_summary_enabled} onChange={(value) => updateAgentRuntime({ operator_summary_enabled: value })} />
                    <ToggleField label="Hypothesis Ranking" description="Allow the agent layer to rank likely causes based on structured evidence." checked={policy.agent_runtime.hypothesis_ranking_enabled} onChange={(value) => updateAgentRuntime({ hypothesis_ranking_enabled: value })} />
                    <ToggleField label="Draft Action Plans" description="Allow the agent layer to draft step-by-step plans without executing them directly." checked={policy.agent_runtime.draft_action_plans_enabled} onChange={(value) => updateAgentRuntime({ draft_action_plans_enabled: value })} />
                    <ToggleField label="Conversational Approval Support" description="Prepare the agent layer to assist with future provider-based approval flows." checked={policy.agent_runtime.conversational_approval_support} onChange={(value) => updateAgentRuntime({ conversational_approval_support: value })} />
                    <ToggleField label="Human Confirmation For Messages" description="Require a human confirmation checkpoint before outbound AI-authored operator messages are sent." checked={policy.agent_runtime.require_human_confirmation_for_message} onChange={(value) => updateAgentRuntime({ require_human_confirmation_for_message: value })} />
                </div>
            </SectionCard>
        </div>
    )

export const RulesSettingsOverviewPanel: React.FC = () => (
    <div className="space-y-6">
        <TabIntro tabId="rules" />
        <SectionCard title="Rules Management" subtitle="Manage operator and detector rules on dedicated pages built for larger rule sets.">
            <div className="grid gap-3 md:grid-cols-2">
                <Link
                    to="/admin/operations/sre-smart-bot/settings/operator-rules"
                    className="rounded-xl border border-slate-200 bg-white px-4 py-4 transition hover:border-sky-300 hover:bg-sky-50/60 dark:border-slate-800 dark:bg-slate-950/40 dark:hover:border-sky-700 dark:hover:bg-sky-950/20"
                >
                    <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">Operator Rules</div>
                    <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Create and tune operator-defined escalation and routing rules.</div>
                </Link>
                <Link
                    to="/admin/operations/sre-smart-bot/settings/detector-rules"
                    className="rounded-xl border border-slate-200 bg-white px-4 py-4 transition hover:border-sky-300 hover:bg-sky-50/60 dark:border-slate-800 dark:bg-slate-950/40 dark:hover:border-sky-700 dark:hover:bg-sky-950/20"
                >
                    <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">Active Detector Rules</div>
                    <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Search, filter, and edit active detector rules at runtime scale.</div>
                </Link>
            </div>
        </SectionCard>
    </div>
)

export const RulesSettingsPanel: React.FC<{
    policy: SRESmartBotPolicyConfig
    updateRule: UpdateRule
    updateDetectorRule: UpdateDetectorRule
    addRule: AddRule
    addRuleFromJson: AddRuleFromJson
    removeRule: RemoveRule
    showOperatorRules?: boolean
    showDetectorRules?: boolean
    showTabIntro?: boolean
    onOperatorRuleSave?: UpdateOperatorRuleWithIndex
    onDetectorRuleSave?: UpdateDetectorRule
    operatorMutationInFlight?: boolean
    detectorMutationInFlight?: boolean
}> = ({
    policy,
    updateRule,
    updateDetectorRule,
    addRule,
    addRuleFromJson,
    removeRule,
    showOperatorRules = true,
    showDetectorRules = true,
    showTabIntro = true,
    onOperatorRuleSave,
    onDetectorRuleSave,
    operatorMutationInFlight = false,
    detectorMutationInFlight = false,
}) => {
        const [detectorSearch, setDetectorSearch] = React.useState('')
        const [statusFilter, setStatusFilter] = React.useState<'all' | 'enabled' | 'disabled'>('all')
        const [severityFilter, setSeverityFilter] = React.useState<'all' | 'info' | 'warning' | 'critical'>('all')
        const [domainFilter, setDomainFilter] = React.useState('all')
        const [sourceFilter, setSourceFilter] = React.useState('all')
        const [sortBy, setSortBy] = React.useState<'name' | 'severity' | 'domain' | 'source'>('severity')
        const [pageSize, setPageSize] = React.useState<25 | 50 | 100 | 250>(50)
        const [page, setPage] = React.useState(1)
        const [editingOperatorRuleIndex, setEditingOperatorRuleIndex] = React.useState<number | null>(null)
        const [editingDetectorRuleIndex, setEditingDetectorRuleIndex] = React.useState<number | null>(null)
        const [operatorRuleJson, setOperatorRuleJson] = React.useState('')
        const [operatorRuleJsonError, setOperatorRuleJsonError] = React.useState<string | null>(null)

        const operatorDomainOptions = React.useMemo(() => {
            return Array.from(new Set([...DOMAIN_OPTIONS, ...policy.enabled_domains])).sort((a, b) => a.localeCompare(b))
        }, [policy.enabled_domains])

        const activeDetectorFilterCount = [
            detectorSearch.trim() !== '',
            statusFilter !== 'all',
            severityFilter !== 'all',
            domainFilter !== 'all',
            sourceFilter !== 'all',
            sortBy !== 'severity',
        ].filter(Boolean).length

        const detectorDomainOptions = React.useMemo(() => {
            return Array.from(new Set(policy.detector_rules.map((rule) => (rule.domain || '').trim()).filter(Boolean))).sort((a, b) => a.localeCompare(b))
        }, [policy.detector_rules])

        const detectorSourceOptions = React.useMemo(() => {
            return Array.from(new Set(policy.detector_rules.map((rule) => (rule.source || '').trim()).filter(Boolean))).sort((a, b) => a.localeCompare(b))
        }, [policy.detector_rules])

        const detectorEditorDomainOptions = React.useMemo(() => {
            return Array.from(new Set([...policy.enabled_domains, ...detectorDomainOptions])).sort((a, b) => a.localeCompare(b))
        }, [policy.enabled_domains, detectorDomainOptions])

        const filteredDetectorRules = React.useMemo(() => {
            const normalizedSearch = detectorSearch.trim().toLowerCase()
            const severityRank: Record<string, number> = { critical: 3, warning: 2, info: 1 }

            const filtered = policy.detector_rules.filter((rule) => {
                if (statusFilter === 'enabled' && !rule.enabled) return false
                if (statusFilter === 'disabled' && rule.enabled) return false
                if (severityFilter !== 'all' && (rule.severity || '').toLowerCase() !== severityFilter) return false
                if (domainFilter !== 'all' && (rule.domain || '') !== domainFilter) return false
                if (sourceFilter !== 'all' && (rule.source || '') !== sourceFilter) return false
                if (!normalizedSearch) return true

                const haystack = [
                    rule.name,
                    rule.id,
                    rule.domain,
                    rule.incident_type,
                    rule.source,
                    rule.signal_key,
                    rule.severity,
                    rule.query,
                ]
                    .map((item) => (item || '').toLowerCase())
                    .join(' ')

                return haystack.includes(normalizedSearch)
            })

            return [...filtered].sort((a, b) => {
                if (sortBy === 'severity') {
                    const left = severityRank[(a.severity || '').toLowerCase()] || 0
                    const right = severityRank[(b.severity || '').toLowerCase()] || 0
                    if (right !== left) return right - left
                    return (a.name || '').localeCompare(b.name || '')
                }
                if (sortBy === 'domain') {
                    const byDomain = (a.domain || '').localeCompare(b.domain || '')
                    if (byDomain !== 0) return byDomain
                    return (a.name || '').localeCompare(b.name || '')
                }
                if (sortBy === 'source') {
                    const bySource = (a.source || '').localeCompare(b.source || '')
                    if (bySource !== 0) return bySource
                    return (a.name || '').localeCompare(b.name || '')
                }
                return (a.name || '').localeCompare(b.name || '')
            })
        }, [policy.detector_rules, detectorSearch, statusFilter, severityFilter, domainFilter, sourceFilter, sortBy])

        const totalPages = Math.max(1, Math.ceil(filteredDetectorRules.length / pageSize))
        const currentPage = Math.min(page, totalPages)
        const pagedDetectorRules = React.useMemo(() => {
            const start = (currentPage - 1) * pageSize
            return filteredDetectorRules.slice(start, start + pageSize).map((rule) => ({
                rule,
                index: policy.detector_rules.findIndex((item) => item.id === rule.id),
            }))
        }, [filteredDetectorRules, currentPage, pageSize, policy.detector_rules])

        React.useEffect(() => {
            setPage(1)
        }, [detectorSearch, statusFilter, severityFilter, domainFilter, sourceFilter, sortBy, pageSize])

        return (
            <div className="space-y-6">
                {showTabIntro ? <TabIntro tabId="rules" /> : null}
                {showOperatorRules ? (
                    <SectionCard title="Operator Rules" subtitle="Built-in rules stay in code. These operator-defined rules are additive and can tune thresholds, routing, and safe auto-allow behavior.">
                        {operatorMutationInFlight ? (
                            <div className="mb-4 rounded-2xl border border-sky-200 bg-sky-50/80 px-4 py-3 text-sm text-sky-800 dark:border-sky-900/50 dark:bg-sky-950/30 dark:text-sky-200">
                                Saving operator rule changes...
                            </div>
                        ) : null}
                        {policy.operator_rules.length === 0 ? (
                            <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-4 py-6 text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-950/30 dark:text-slate-300">
                                No operator-defined rules yet. The current watcher behavior is being driven by built-in policy and runtime signals.
                            </div>
                        ) : (
                            <div className="overflow-x-auto rounded-2xl border border-slate-200 dark:border-slate-800">
                                <table className="min-w-full divide-y divide-slate-200 text-sm dark:divide-slate-800">
                                    <thead className="bg-slate-50 dark:bg-slate-900/60">
                                        <tr>
                                            <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Rule</th>
                                            <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Status</th>
                                            <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Severity</th>
                                            <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Domain</th>
                                            <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Incident Type</th>
                                            <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Source</th>
                                            <th className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Action</th>
                                        </tr>
                                    </thead>
                                    <tbody className="divide-y divide-slate-100 bg-white dark:divide-slate-800 dark:bg-slate-950/30">
                                        {policy.operator_rules.map((rule, index) => (
                                            <tr key={rule.id} className="align-top hover:bg-slate-50/70 dark:hover:bg-slate-900/40">
                                                <td className="px-3 py-3">
                                                    <div className="font-medium text-slate-900 dark:text-slate-100">{rule.name || `Rule ${index + 1}`}</div>
                                                    <div className="mt-1 text-xs text-slate-500 dark:text-slate-400">{rule.id}</div>
                                                </td>
                                                <td className="px-3 py-3">
                                                    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-semibold ${rule.enabled ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300' : 'bg-slate-200 text-slate-700 dark:bg-slate-800 dark:text-slate-300'}`}>{rule.enabled ? 'enabled' : 'disabled'}</span>
                                                </td>
                                                <td className="px-3 py-3">
                                                    <span className="inline-flex rounded-full border border-slate-300 px-2 py-0.5 text-xs font-semibold text-slate-700 dark:border-slate-700 dark:text-slate-300">{rule.severity || 'warning'}</span>
                                                </td>
                                                <td className="px-3 py-3 text-slate-700 dark:text-slate-300">{rule.domain || '—'}</td>
                                                <td className="px-3 py-3 text-slate-700 dark:text-slate-300">{rule.incident_type || '—'}</td>
                                                <td className="px-3 py-3 text-slate-700 dark:text-slate-300">{rule.source || 'operator_defined'}</td>
                                                <td className="px-3 py-3 text-right">
                                                    <div className="flex justify-end gap-2">
                                                        <button
                                                            type="button"
                                                            className="rounded-lg border border-slate-300 bg-white px-2.5 py-1 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                                                            onClick={() => setEditingOperatorRuleIndex(index)}
                                                        >
                                                            View
                                                        </button>
                                                        <button
                                                            type="button"
                                                            onClick={() => removeRule(index)}
                                                            className="rounded-lg border border-rose-300 bg-white px-2.5 py-1 text-xs font-medium text-rose-700 transition hover:border-rose-400 hover:bg-rose-50 dark:border-rose-800 dark:bg-slate-900 dark:text-rose-300 dark:hover:border-rose-700 dark:hover:bg-rose-950/30"
                                                        >
                                                            Remove
                                                        </button>
                                                    </div>
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        )}
                        <div className="mt-4">
                            <div className="flex flex-wrap items-center gap-2">
                                <button
                                    type="button"
                                    onClick={addRule}
                                    className="inline-flex w-fit rounded-xl border border-dashed border-sky-400 bg-sky-50/60 px-4 py-2 text-sm font-medium text-sky-700 transition hover:border-sky-500 hover:bg-sky-100 dark:border-sky-700 dark:bg-sky-950/20 dark:text-sky-300 dark:hover:border-sky-500 dark:hover:bg-sky-950/40"
                                >
                                    + Add Operator Rule
                                </button>
                            </div>
                            <details className="mt-3 rounded-xl border border-slate-200 bg-slate-50/80 p-3 dark:border-slate-800 dark:bg-slate-950/30">
                                <summary className="inline-flex w-fit list-none rounded-xl border border-dashed border-sky-400 bg-sky-50/60 px-4 py-2 text-sm font-medium text-sky-700 transition marker:content-none hover:border-sky-500 hover:bg-sky-100 dark:border-sky-700 dark:bg-sky-950/20 dark:text-sky-300 dark:hover:border-sky-500 dark:hover:bg-sky-950/40 cursor-pointer">
                                    + Add Operator Rule From JSON
                                </summary>
                                <div className="mt-3 space-y-2">
                                    <textarea
                                        className={`${inputClass} min-h-32 font-mono text-xs`}
                                        value={operatorRuleJson}
                                        onChange={(e) => {
                                            setOperatorRuleJson(e.target.value)
                                            setOperatorRuleJsonError(null)
                                        }}
                                        placeholder={"{\n  \"name\": \"Transport lag spike\",\n  \"domain\": \"runtime_services\",\n  \"incident_type\": \"consumer_lag\",\n  \"severity\": \"warning\",\n  \"threshold\": 3,\n  \"for_duration_seconds\": 600,\n  \"enabled\": true\n}"}
                                    />
                                    {operatorRuleJsonError ? <div className="text-xs text-rose-600 dark:text-rose-400">{operatorRuleJsonError}</div> : null}
                                    <div className="flex items-center gap-2">
                                        <button
                                            type="button"
                                            className="rounded-lg bg-slate-900 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-slate-700 dark:bg-slate-100 dark:text-slate-900 dark:hover:bg-slate-300"
                                            onClick={() => {
                                                try {
                                                    const parsed = JSON.parse(operatorRuleJson)
                                                    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
                                                        setOperatorRuleJsonError('JSON must be a single rule object.')
                                                        return
                                                    }
                                                    addRuleFromJson(parsed)
                                                    setOperatorRuleJson('')
                                                    setOperatorRuleJsonError(null)
                                                } catch {
                                                    setOperatorRuleJsonError('Invalid JSON. Please fix syntax and try again.')
                                                }
                                            }}
                                        >
                                            Add From JSON
                                        </button>
                                        <button
                                            type="button"
                                            className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:border-slate-400 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200"
                                            onClick={() => {
                                                setOperatorRuleJson('')
                                                setOperatorRuleJsonError(null)
                                            }}
                                        >
                                            Clear
                                        </button>
                                    </div>
                                </div>
                            </details>
                        </div>
                    </SectionCard>
                ) : null}

                {showDetectorRules ? (
                    <SectionCard title="Active Detector Rules" subtitle="Search, filter, and page through active detector rules at runtime scale.">
                        {detectorMutationInFlight ? (
                            <div className="mb-4 rounded-2xl border border-sky-200 bg-sky-50/80 px-4 py-3 text-sm text-sky-800 dark:border-sky-900/50 dark:bg-sky-950/30 dark:text-sky-200">
                                Saving detector rule changes...
                            </div>
                        ) : null}
                        {policy.detector_rules.length === 0 ? (
                            <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-4 py-6 text-center text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-950/30 dark:text-slate-300">
                                <p>No custom detector rules are active yet.</p>
                                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                    Accepted learned rules from the{' '}
                                    <Link to="/admin/operations/sre-smart-bot/detector-rules" className="font-medium text-sky-600 underline-offset-2 hover:underline dark:text-sky-400">
                                        Detector Rules
                                    </Link>{' '}
                                    page will appear here and be merged into the log detector at runtime.
                                </p>
                            </div>
                        ) : (
                            <div className="space-y-4">
                                <details className="group rounded-xl border border-slate-200 bg-slate-50/80 p-3 dark:border-slate-800 dark:bg-slate-950/30">
                                    <summary className="flex cursor-pointer list-none flex-wrap items-center justify-between gap-2 rounded-lg px-1 py-1 text-sm font-medium text-slate-700 marker:content-none dark:text-slate-200">
                                        <span>Search & Filters</span>
                                        <span className="inline-flex items-center gap-3">
                                            <span className="text-xs text-slate-500 dark:text-slate-400">
                                                {activeDetectorFilterCount > 0 ? `${activeDetectorFilterCount} filter(s) active` : 'No active filters'}
                                            </span>
                                            <span className="text-lg leading-none text-slate-500 dark:text-slate-300 group-open:hidden">▼</span>
                                            <span className="hidden text-lg leading-none text-slate-500 dark:text-slate-300 group-open:inline">▲</span>
                                        </span>
                                    </summary>
                                    <div className="mt-3 grid gap-3 lg:grid-cols-2 xl:grid-cols-3">
                                        <label className="space-y-2 xl:col-span-2">
                                            <span className={labelClass}>Search</span>
                                            <input
                                                className={inputClass}
                                                value={detectorSearch}
                                                onChange={(e) => setDetectorSearch(e.target.value)}
                                                placeholder="Search by name, domain, signal key, query, id"
                                            />
                                        </label>
                                        <label className="space-y-2">
                                            <span className={labelClass}>Sort</span>
                                            <select className={inputClass} value={sortBy} onChange={(e) => setSortBy(e.target.value as 'name' | 'severity' | 'domain' | 'source')}>
                                                <option value="severity">Severity</option>
                                                <option value="name">Name</option>
                                                <option value="domain">Domain</option>
                                                <option value="source">Source</option>
                                            </select>
                                        </label>
                                        <label className="space-y-2">
                                            <span className={labelClass}>Status</span>
                                            <select className={inputClass} value={statusFilter} onChange={(e) => setStatusFilter(e.target.value as 'all' | 'enabled' | 'disabled')}>
                                                <option value="all">All</option>
                                                <option value="enabled">Enabled</option>
                                                <option value="disabled">Disabled</option>
                                            </select>
                                        </label>
                                        <label className="space-y-2">
                                            <span className={labelClass}>Severity</span>
                                            <select className={inputClass} value={severityFilter} onChange={(e) => setSeverityFilter(e.target.value as 'all' | 'info' | 'warning' | 'critical')}>
                                                <option value="all">All</option>
                                                <option value="critical">Critical</option>
                                                <option value="warning">Warning</option>
                                                <option value="info">Info</option>
                                            </select>
                                        </label>
                                        <label className="space-y-2">
                                            <span className={labelClass}>Domain</span>
                                            <select className={inputClass} value={domainFilter} onChange={(e) => setDomainFilter(e.target.value)}>
                                                <option value="all">All</option>
                                                {detectorDomainOptions.map((domain) => (
                                                    <option key={domain} value={domain}>{domain}</option>
                                                ))}
                                            </select>
                                        </label>
                                        <label className="space-y-2">
                                            <span className={labelClass}>Source</span>
                                            <select className={inputClass} value={sourceFilter} onChange={(e) => setSourceFilter(e.target.value)}>
                                                <option value="all">All</option>
                                                {detectorSourceOptions.map((source) => (
                                                    <option key={source} value={source}>{source}</option>
                                                ))}
                                            </select>
                                        </label>
                                        <div className="xl:col-span-3">
                                            <button
                                                type="button"
                                                className="rounded-lg border border-slate-300 bg-white px-2 py-1 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                                                onClick={() => {
                                                    setDetectorSearch('')
                                                    setStatusFilter('all')
                                                    setSeverityFilter('all')
                                                    setDomainFilter('all')
                                                    setSourceFilter('all')
                                                    setSortBy('severity')
                                                }}
                                            >
                                                Reset Filters
                                            </button>
                                        </div>
                                    </div>
                                </details>

                                <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50/80 px-3 py-2 text-sm dark:border-slate-800 dark:bg-slate-950/30">
                                    <div className="text-slate-600 dark:text-slate-300">
                                        Showing <span className="font-semibold text-slate-900 dark:text-slate-100">{pagedDetectorRules.length}</span> of <span className="font-semibold text-slate-900 dark:text-slate-100">{filteredDetectorRules.length}</span> filtered rules ({policy.detector_rules.length} total)
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <span className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">Rows</span>
                                        <select className="rounded-lg border border-slate-300 bg-white px-2 py-1 text-sm dark:border-slate-700 dark:bg-slate-900" value={pageSize} onChange={(e) => setPageSize(Number(e.target.value) as 25 | 50 | 100 | 250)}>
                                            <option value={25}>25</option>
                                            <option value={50}>50</option>
                                            <option value={100}>100</option>
                                            <option value={250}>250</option>
                                        </select>
                                    </div>
                                </div>

                                {filteredDetectorRules.length === 0 ? (
                                    <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-4 py-6 text-center text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-950/30 dark:text-slate-300">
                                        No detector rules match the current search and filters.
                                    </div>
                                ) : (
                                    <>
                                        <div className="overflow-x-auto rounded-2xl border border-slate-200 dark:border-slate-800">
                                            <table className="min-w-full divide-y divide-slate-200 text-sm dark:divide-slate-800">
                                                <thead className="bg-slate-50 dark:bg-slate-900/60">
                                                    <tr>
                                                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Rule</th>
                                                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Status</th>
                                                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Severity</th>
                                                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Domain</th>
                                                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Incident Type</th>
                                                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Signal</th>
                                                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Source</th>
                                                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Query</th>
                                                        <th className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Action</th>
                                                    </tr>
                                                </thead>
                                                <tbody className="divide-y divide-slate-100 bg-white dark:divide-slate-800 dark:bg-slate-950/30">
                                                    {pagedDetectorRules.map(({ rule, index }) => (
                                                        <tr key={rule.id} className="align-top hover:bg-slate-50/70 dark:hover:bg-slate-900/40">
                                                            <td className="px-3 py-3">
                                                                <div className="font-medium text-slate-900 dark:text-slate-100">{rule.name}</div>
                                                                <div className="mt-1 text-xs text-slate-500 dark:text-slate-400">{rule.id}</div>
                                                            </td>
                                                            <td className="px-3 py-3">
                                                                <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-semibold ${rule.enabled ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300' : 'bg-slate-200 text-slate-700 dark:bg-slate-800 dark:text-slate-300'}`}>{rule.enabled ? 'enabled' : 'disabled'}</span>
                                                            </td>
                                                            <td className="px-3 py-3">
                                                                <span className="inline-flex rounded-full border border-slate-300 px-2 py-0.5 text-xs font-semibold text-slate-700 dark:border-slate-700 dark:text-slate-300">{rule.severity || 'info'}</span>
                                                            </td>
                                                            <td className="px-3 py-3 text-slate-700 dark:text-slate-300">{rule.domain}</td>
                                                            <td className="px-3 py-3 text-slate-700 dark:text-slate-300">{rule.incident_type}</td>
                                                            <td className="px-3 py-3 text-slate-700 dark:text-slate-300">{rule.signal_key || '—'}</td>
                                                            <td className="px-3 py-3 text-slate-700 dark:text-slate-300">{rule.source || 'operator_defined'}</td>
                                                            <td className="max-w-[28rem] px-3 py-3 text-xs text-slate-600 dark:text-slate-400">
                                                                <code className="line-clamp-2 whitespace-pre-wrap break-all">{rule.query}</code>
                                                            </td>
                                                            <td className="px-3 py-3 text-right">
                                                                <button
                                                                    type="button"
                                                                    className="rounded-lg border border-slate-300 bg-white px-2.5 py-1 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                                                                    onClick={() => {
                                                                        if (index >= 0) {
                                                                            setEditingDetectorRuleIndex(index)
                                                                        }
                                                                    }}
                                                                >
                                                                    Edit
                                                                </button>
                                                            </td>
                                                        </tr>
                                                    ))}
                                                </tbody>
                                            </table>
                                        </div>

                                        <div className="flex flex-wrap items-center justify-between gap-3">
                                            <div className="text-sm text-slate-600 dark:text-slate-400">Page {currentPage} of {totalPages}</div>
                                            <div className="flex items-center gap-2">
                                                <button
                                                    type="button"
                                                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                                                    disabled={currentPage <= 1}
                                                    className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                                                >
                                                    Previous
                                                </button>
                                                <button
                                                    type="button"
                                                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                                                    disabled={currentPage >= totalPages}
                                                    className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                                                >
                                                    Next
                                                </button>
                                            </div>
                                        </div>
                                    </>
                                )}
                            </div>
                        )}
                    </SectionCard>
                ) : null}

                {showDetectorRules ? (
                    <SRESmartBotDetectorRuleDrawer
                        isOpen={editingDetectorRuleIndex !== null}
                        onClose={() => setEditingDetectorRuleIndex(null)}
                        rule={editingDetectorRuleIndex !== null ? policy.detector_rules[editingDetectorRuleIndex] || null : null}
                        domainOptions={detectorEditorDomainOptions}
                        onSave={(nextRule) => {
                            if (editingDetectorRuleIndex !== null) {
                                if (onDetectorRuleSave) {
                                    onDetectorRuleSave(editingDetectorRuleIndex, nextRule)
                                } else {
                                    updateDetectorRule(editingDetectorRuleIndex, nextRule)
                                }
                            }
                        }}
                    />
                ) : null}
                {showOperatorRules ? (
                    <SRESmartBotOperatorRuleDrawer
                        isOpen={editingOperatorRuleIndex !== null}
                        onClose={() => setEditingOperatorRuleIndex(null)}
                        rule={editingOperatorRuleIndex !== null ? policy.operator_rules[editingOperatorRuleIndex] || null : null}
                        domainOptions={operatorDomainOptions}
                        onSave={(nextRule) => {
                            if (editingOperatorRuleIndex !== null) {
                                if (onOperatorRuleSave) {
                                    onOperatorRuleSave(editingOperatorRuleIndex, nextRule)
                                } else {
                                    updateRule(editingOperatorRuleIndex, nextRule)
                                }
                            }
                        }}
                    />
                ) : null}
            </div>
        )
    }