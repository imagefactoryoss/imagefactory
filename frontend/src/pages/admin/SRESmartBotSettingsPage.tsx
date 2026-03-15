import { adminService } from '@/services/adminService'
import type { SREAgentRuntimeProbeResponse, SRESmartBotAgentRuntimeConfig, SRESmartBotChannelProvider, SRESmartBotDetectorRule, SRESmartBotMCPServer, SRESmartBotOperatorRule, SRESmartBotPolicyConfig } from '@/types'
import React, { useEffect, useState } from 'react'
import toast from 'react-hot-toast'

const DOMAIN_OPTIONS = [
    'infrastructure',
    'runtime_services',
    'application_services',
    'network_ingress',
    'identity_security',
    'release_configuration',
    'operator_channels',
]

const ENVIRONMENT_OPTIONS = ['demo', 'development', 'staging', 'production']
const MCP_KIND_OPTIONS = ['observability', 'kubernetes', 'oci', 'database', 'release', 'chat', 'custom']
const MCP_TRANSPORT_OPTIONS = ['embedded', 'http', 'stdio', 'custom']
const AGENT_PROVIDER_OPTIONS = ['custom', 'ollama', 'openai', 'none']
const DETECTOR_LEARNING_MODE_OPTIONS = ['disabled', 'suggest_only', 'training_auto_create'] as const
const SETTINGS_TABS = [
    { id: 'general', label: 'General', description: 'Identity, automation, and domain scope' },
    { id: 'channels', label: 'Channels', description: 'Operator channel providers' },
    { id: 'mcp-ai', label: 'MCP & AI', description: 'Tool servers and agent runtime' },
    { id: 'rules', label: 'Rules', description: 'Operator-defined policy rules' },
] as const

type SettingsTabId = (typeof SETTINGS_TABS)[number]['id']

const SectionCard: React.FC<{ title: string; subtitle?: string; children: React.ReactNode }> = ({ title, subtitle, children }) => (
    <section className="rounded-2xl border border-slate-200 bg-white/90 p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900/85">
        <div className="mb-4">
            <h2 className="text-sm font-semibold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">{title}</h2>
            {subtitle ? <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">{subtitle}</p> : null}
        </div>
        {children}
    </section>
)

const inputClass = 'w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900'
const labelClass = 'text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400'

const ToggleField: React.FC<{
    label: string
    description: string
    checked: boolean
    onChange: (value: boolean) => void
}> = ({ label, description, checked, onChange }) => (
    <label className="flex items-start justify-between gap-4 rounded-xl border border-slate-200 bg-slate-50/90 p-4 dark:border-slate-800 dark:bg-slate-950/50">
        <div>
            <div className="text-sm font-medium text-slate-900 dark:text-white">{label}</div>
            <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">{description}</div>
        </div>
        <button
            type="button"
            aria-pressed={checked}
            onClick={() => onChange(!checked)}
            className={`relative mt-0.5 inline-flex h-7 w-12 shrink-0 rounded-full border transition ${checked ? 'border-sky-500 bg-sky-500 dark:border-sky-400 dark:bg-sky-400' : 'border-slate-300 bg-slate-200 dark:border-slate-700 dark:bg-slate-800'}`}
        >
            <span
                className={`inline-block h-5 w-5 translate-y-[3px] rounded-full bg-white shadow transition ${checked ? 'translate-x-6' : 'translate-x-1'}`}
            />
        </button>
    </label>
)

const parseLabelMap = (value: string): Record<string, string> => {
    return value
        .split('\n')
        .map((line) => line.trim())
        .filter(Boolean)
        .reduce<Record<string, string>>((acc, line) => {
            const [key, ...rest] = line.split('=')
            if (key && rest.length > 0) {
                acc[key.trim()] = rest.join('=').trim()
            }
            return acc
        }, {})
}

const stringifyLabelMap = (value?: Record<string, string>) =>
    Object.entries(value || {})
        .map(([key, item]) => `${key}=${item}`)
        .join('\n')

const normalizePolicy = (policy: SRESmartBotPolicyConfig): SRESmartBotPolicyConfig => ({
    ...policy,
    enabled_domains: policy.enabled_domains || [],
    detector_learning_mode: policy.detector_learning_mode || 'suggest_only',
    channel_providers: policy.channel_providers || [],
    mcp_servers: policy.mcp_servers || [],
    agent_runtime: policy.agent_runtime || {
        enabled: false,
        provider: 'ollama',
        model: 'llama3.2:3b',
        base_url: 'http://127.0.0.1:11434',
        system_prompt_ref: 'sre_smart_bot_default',
        operator_summary_enabled: true,
        hypothesis_ranking_enabled: true,
        draft_action_plans_enabled: true,
        conversational_approval_support: false,
        max_tool_calls_per_turn: 6,
        max_incidents_per_summary: 5,
        require_human_confirmation_for_message: true,
    },
    detector_rules: policy.detector_rules || [],
    operator_rules: policy.operator_rules || [],
})

const detectorRuleTone = (rule: SRESmartBotDetectorRule) => {
    if (!rule.enabled) return 'border-slate-200 bg-slate-50 text-slate-700 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300'
    if ((rule.severity || '').toLowerCase() === 'critical') return 'border-rose-200 bg-rose-50 text-rose-900 dark:border-rose-900/40 dark:bg-rose-950/30 dark:text-rose-200'
    if ((rule.severity || '').toLowerCase() === 'warning') return 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200'
    return 'border-sky-200 bg-sky-50 text-sky-900 dark:border-sky-900/40 dark:bg-sky-950/30 dark:text-sky-200'
}

const SRESmartBotSettingsPage: React.FC = () => {
    const [policy, setPolicy] = useState<SRESmartBotPolicyConfig | null>(null)
    const [recommendedPolicy, setRecommendedPolicy] = useState<SRESmartBotPolicyConfig | null>(null)
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [probing, setProbing] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [probeResult, setProbeResult] = useState<SREAgentRuntimeProbeResponse | null>(null)
    const [activeTab, setActiveTab] = useState<SettingsTabId>('general')

    const loadPolicy = async () => {
        try {
            setLoading(true)
            setError(null)
            const [response, defaultsResponse] = await Promise.all([
                adminService.getSRESmartBotPolicy(),
                adminService.getSRESmartBotPolicyDefaults(),
            ])
            setPolicy(normalizePolicy(response))
            setRecommendedPolicy(normalizePolicy(defaultsResponse))
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load SRE Smart Bot settings')
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        void loadPolicy()
    }, [])

    const updatePolicy = <K extends keyof SRESmartBotPolicyConfig>(key: K, value: SRESmartBotPolicyConfig[K]) => {
        setPolicy((current) => (current ? { ...current, [key]: value } : current))
    }

    const updateChannelProvider = (index: number, updates: Partial<SRESmartBotChannelProvider>) => {
        setPolicy((current) => {
            if (!current) return current
            const channelProviders = [...current.channel_providers]
            channelProviders[index] = { ...channelProviders[index], ...updates }
            return { ...current, channel_providers: channelProviders }
        })
    }

    const updateRule = (index: number, updates: Partial<SRESmartBotOperatorRule>) => {
        setPolicy((current) => {
            if (!current) return current
            const operatorRules = [...current.operator_rules]
            operatorRules[index] = { ...operatorRules[index], ...updates }
            return { ...current, operator_rules: operatorRules }
        })
    }

    const updateMCPServer = (index: number, updates: Partial<SRESmartBotMCPServer>) => {
        setPolicy((current) => {
            if (!current) return current
            const mcpServers = [...current.mcp_servers]
            mcpServers[index] = { ...mcpServers[index], ...updates }
            return { ...current, mcp_servers: mcpServers }
        })
    }

    const updateAgentRuntime = (updates: Partial<SRESmartBotAgentRuntimeConfig>) => {
        setPolicy((current) => {
            if (!current) return current
            return { ...current, agent_runtime: { ...current.agent_runtime, ...updates } }
        })
    }

    const savePolicy = async () => {
        if (!policy) return
        try {
            setSaving(true)
            setError(null)
            const response = await adminService.updateSRESmartBotPolicy(policy)
            setPolicy(normalizePolicy(response))
            toast.success('SRE Smart Bot settings saved')
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to save SRE Smart Bot settings'
            setError(message)
            toast.error(message)
        } finally {
            setSaving(false)
        }
    }

    const probeAgentRuntime = async () => {
        if (!policy) return
        try {
            setProbing(true)
            setProbeResult(null)
            setActiveTab('mcp-ai')
            const response = await adminService.probeSREAgentRuntime({ agent_runtime: policy.agent_runtime })
            setProbeResult(response)
            toast.success(response.healthy ? 'Local model probe succeeded' : 'Local model probe completed')
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to probe agent runtime'
            setActiveTab('mcp-ai')
            setProbeResult({
                provider: policy.agent_runtime.provider || '',
                model: policy.agent_runtime.model || '',
                base_url: policy.agent_runtime.base_url || '',
                healthy: false,
                status: 'error',
                message,
                model_installed: false,
            })
            toast.error(message)
        } finally {
            setProbing(false)
        }
    }

    const applyRecommendedAgentRuntime = () => {
        if (!policy || !recommendedPolicy) return
        setPolicy({
            ...policy,
            agent_runtime: {
                ...policy.agent_runtime,
                provider: recommendedPolicy.agent_runtime.provider,
                model: recommendedPolicy.agent_runtime.model,
                base_url: recommendedPolicy.agent_runtime.base_url,
            },
        })
        setActiveTab('mcp-ai')
        toast.success('Applied recommended agent runtime defaults')
    }

    const showRecommendedAgentRuntimeHint = !!policy && !!recommendedPolicy && (
        policy.agent_runtime.provider !== recommendedPolicy.agent_runtime.provider ||
        (policy.agent_runtime.model || '') !== (recommendedPolicy.agent_runtime.model || '') ||
        (policy.agent_runtime.base_url || '') !== (recommendedPolicy.agent_runtime.base_url || '')
    )

    return (
        <div className="min-h-full w-full bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.08),_transparent_30%),linear-gradient(180deg,_#f8fafc_0%,_#eef2ff_100%)] px-4 py-6 text-slate-900 sm:px-6 lg:px-8 dark:bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.16),_transparent_24%),linear-gradient(180deg,_#020617_0%,_#0f172a_100%)] dark:text-slate-100">
            <div className="w-full space-y-6">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                    <div>
                        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">Operations</p>
                        <h1 className="mt-2 text-3xl font-semibold tracking-tight">SRE Smart Bot Settings</h1>
                        <p className="mt-2 max-w-4xl text-sm text-slate-600 dark:text-slate-400">
                            Configure how SRE Smart Bot observes incidents, notifies operators, proposes containment, and routes approvals.
                            Today it runs inside the backend process as an embedded control-plane subsystem. The policy contract is being kept
                            service-safe so we can move it into a standalone worker later without changing the admin surface.
                        </p>
                        <p className="mt-2 text-sm text-slate-500 dark:text-slate-400">
                            Model testing lives in the <span className="font-medium text-slate-700 dark:text-slate-200">MCP &amp; AI</span> tab alongside the agent runtime configuration and probe results.
                        </p>
                    </div>
                    <div className="flex items-center gap-3">
                        <button
                            type="button"
                            onClick={() => void loadPolicy()}
                            className="rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                        >
                            Refresh
                        </button>
                        <button
                            type="button"
                            onClick={() => void savePolicy()}
                            disabled={!policy || saving}
                            className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-sky-500 dark:hover:bg-sky-400"
                        >
                            {saving ? 'Saving...' : 'Save Settings'}
                        </button>
                    </div>
                </div>

                {loading ? (
                    <div className="rounded-2xl border border-slate-200 bg-white/80 px-4 py-16 text-center text-sm text-slate-600 shadow-sm dark:border-slate-800 dark:bg-slate-900/80 dark:text-slate-300">
                        Loading SRE Smart Bot settings...
                    </div>
                ) : error ? (
                    <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-4 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/40 dark:text-rose-200">{error}</div>
                ) : policy ? (
                    <>
                        <div className="rounded-2xl border border-slate-200 bg-white/80 shadow-sm dark:border-slate-800 dark:bg-slate-900/80">
                            <div
                                role="tablist"
                                aria-label="SRE Smart Bot settings sections"
                                className="flex flex-wrap gap-x-1 gap-y-2 border-b border-slate-200 px-3 pt-3 dark:border-slate-800"
                            >
                                {SETTINGS_TABS.map((tab) => {
                                    const active = activeTab === tab.id
                                    return (
                                        <button
                                            key={tab.id}
                                            type="button"
                                            role="tab"
                                            id={`sre-settings-tab-${tab.id}`}
                                            aria-selected={active}
                                            aria-controls={`sre-settings-panel-${tab.id}`}
                                            onClick={() => setActiveTab(tab.id)}
                                            className={`rounded-t-xl border-b-2 px-4 py-3 text-left transition focus:outline-none focus:ring-2 focus:ring-sky-200 dark:focus:ring-sky-900 ${
                                                active
                                                    ? 'border-sky-500 bg-sky-50/80 text-sky-950 dark:border-sky-400 dark:bg-sky-950/40 dark:text-sky-100'
                                                    : 'border-transparent bg-transparent text-slate-700 hover:border-slate-300 hover:text-sky-800 dark:text-slate-300 dark:hover:border-slate-700 dark:hover:text-sky-200'
                                            }`}
                                        >
                                            <div className="text-sm font-semibold">{tab.label}</div>
                                        </button>
                                    )
                                })}
                            </div>
                        </div>

                        {activeTab === 'general' ? (
                            <>
                        <div role="tabpanel" id="sre-settings-panel-general" aria-labelledby="sre-settings-tab-general" className="space-y-6">
                        <div className="rounded-2xl border border-slate-200 bg-slate-50/90 px-4 py-3 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300">
                            {SETTINGS_TABS.find((tab) => tab.id === 'general')?.description}
                        </div>
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

                        <SectionCard title="Domains" subtitle="Choose which incident domains SRE Smart Bot is allowed to observe and act on in this environment.">
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
                        </SectionCard>
                        </div>
                            </>
                        ) : null}

                        {activeTab === 'channels' ? (
                            <>
                        <div role="tabpanel" id="sre-settings-panel-channels" aria-labelledby="sre-settings-tab-channels" className="space-y-6">
                        <div className="rounded-2xl border border-slate-200 bg-slate-50/90 px-4 py-3 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300">
                            {SETTINGS_TABS.find((tab) => tab.id === 'channels')?.description}
                        </div>
                        <SectionCard title="Channel Providers" subtitle="Operator channels are provider-based, so enterprises can use in-app, email, webhooks, or internal integrations without assuming Telegram or WhatsApp.">
                            <div className="space-y-4">
                                {policy.channel_providers.map((provider, index) => (
                                    <div key={provider.id} className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                        <div className="grid gap-4 lg:grid-cols-3">
                                            <label className="space-y-2">
                                                <span className={labelClass}>Name</span>
                                                <input className={inputClass} value={provider.name} onChange={(e) => updateChannelProvider(index, { name: e.target.value })} />
                                            </label>
                                            <label className="space-y-2">
                                                <span className={labelClass}>Kind</span>
                                                <input className={inputClass} value={provider.kind} onChange={(e) => updateChannelProvider(index, { kind: e.target.value })} />
                                            </label>
                                            <label className="space-y-2">
                                                <span className={labelClass}>Config Ref</span>
                                                <input className={inputClass} value={provider.config_ref || ''} onChange={(e) => updateChannelProvider(index, { config_ref: e.target.value })} />
                                            </label>
                                        </div>
                                        <div className="mt-4 grid gap-3 xl:grid-cols-2">
                                            <div className="rounded-2xl border border-slate-200/80 bg-white/80 p-1 dark:border-slate-800 dark:bg-slate-900/70">
                                                <ToggleField
                                                    label="Enabled"
                                                    description="Allow this provider to receive SRE Smart Bot notifications."
                                                    checked={provider.enabled}
                                                    onChange={(value) => updateChannelProvider(index, { enabled: value })}
                                                />
                                            </div>
                                            <div className="rounded-2xl border border-slate-200/80 bg-white/80 p-1 dark:border-slate-800 dark:bg-slate-900/70">
                                                <ToggleField
                                                    label="Interactive Approval"
                                                    description="Mark whether this provider can carry approval decisions."
                                                    checked={!!provider.supports_interactive_approval}
                                                    onChange={(value) => updateChannelProvider(index, { supports_interactive_approval: value })}
                                                />
                                            </div>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </SectionCard>
                        </div>
                            </>
                        ) : null}

                        {activeTab === 'mcp-ai' ? (
                            <>
                        <div role="tabpanel" id="sre-settings-panel-mcp-ai" aria-labelledby="sre-settings-tab-mcp-ai" className="space-y-6">
                        <div className="rounded-2xl border border-slate-200 bg-slate-50/90 px-4 py-3 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300">
                            {SETTINGS_TABS.find((tab) => tab.id === 'mcp-ai')?.description}
                        </div>
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
                                    onClick={() => void probeAgentRuntime()}
                                    disabled={!policy || probing}
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
                            </>
                        ) : null}

                        {activeTab === 'rules' ? (
                        <>
                        <div role="tabpanel" id="sre-settings-panel-rules" aria-labelledby="sre-settings-tab-rules" className="space-y-6">
                        <div className="rounded-2xl border border-slate-200 bg-slate-50/90 px-4 py-3 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300">
                            {SETTINGS_TABS.find((tab) => tab.id === 'rules')?.description}
                        </div>
                        <SectionCard title="Operator Rules" subtitle="Built-in rules stay in code. These operator-defined rules are additive and can tune thresholds, routing, and safe auto-allow behavior.">
                            {policy.operator_rules.length === 0 ? (
                                <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-4 py-6 text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-950/30 dark:text-slate-300">
                                    No operator-defined rules yet. The current watcher behavior is being driven by built-in policy and runtime signals.
                                </div>
                            ) : (
                                <div className="space-y-4">
                                    {policy.operator_rules.map((rule, index) => (
                                        <div key={rule.id} className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                            <div className="grid gap-4 xl:grid-cols-4">
                                                <label className="space-y-2">
                                                    <span className={labelClass}>Rule Name</span>
                                                    <input className={inputClass} value={rule.name} onChange={(e) => updateRule(index, { name: e.target.value })} />
                                                </label>
                                                <label className="space-y-2">
                                                    <span className={labelClass}>Domain</span>
                                                    <input className={inputClass} value={rule.domain} onChange={(e) => updateRule(index, { domain: e.target.value })} />
                                                </label>
                                                <label className="space-y-2">
                                                    <span className={labelClass}>Incident Type</span>
                                                    <input className={inputClass} value={rule.incident_type} onChange={(e) => updateRule(index, { incident_type: e.target.value })} />
                                                </label>
                                                <label className="space-y-2">
                                                    <span className={labelClass}>Source</span>
                                                    <input className={inputClass} value={rule.source} onChange={(e) => updateRule(index, { source: e.target.value })} />
                                                </label>
                                                <label className="space-y-2">
                                                    <span className={labelClass}>Severity</span>
                                                    <input className={inputClass} value={rule.severity} onChange={(e) => updateRule(index, { severity: e.target.value })} />
                                                </label>
                                                <label className="space-y-2">
                                                    <span className={labelClass}>Threshold</span>
                                                    <input className={inputClass} type="number" min={0} value={rule.threshold || 0} onChange={(e) => updateRule(index, { threshold: Number(e.target.value) || 0 })} />
                                                </label>
                                                <label className="space-y-2">
                                                    <span className={labelClass}>For Duration Seconds</span>
                                                    <input className={inputClass} type="number" min={0} value={rule.for_duration_seconds || 0} onChange={(e) => updateRule(index, { for_duration_seconds: Number(e.target.value) || 0 })} />
                                                </label>
                                                <label className="space-y-2">
                                                    <span className={labelClass}>Suggested Action</span>
                                                    <input className={inputClass} value={rule.suggested_action || ''} onChange={(e) => updateRule(index, { suggested_action: e.target.value })} />
                                                </label>
                                            </div>
                                            <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(0,1fr)_20rem]">
                                                <label className="space-y-2">
                                                    <span className={labelClass}>Match Labels</span>
                                                    <textarea
                                                        className={`${inputClass} min-h-28`}
                                                        value={stringifyLabelMap(rule.match_labels)}
                                                        onChange={(e) => updateRule(index, { match_labels: parseLabelMap(e.target.value) })}
                                                        placeholder={'component=dispatcher\nnamespace=image-factory'}
                                                    />
                                                </label>
                                                <div className="grid gap-3">
                                                    <ToggleField label="Rule Enabled" description="Allow this operator rule to influence evaluation." checked={rule.enabled} onChange={(value) => updateRule(index, { enabled: value })} />
                                                    <ToggleField label="Auto Allowed" description="Mark this rule as safe for automatic execution when policy permits." checked={!!rule.auto_allowed} onChange={(value) => updateRule(index, { auto_allowed: value })} />
                                                </div>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </SectionCard>

                        <SectionCard title="Active Detector Rules" subtitle="These are the currently active log-detector rules. Built-in rules live in code; accepted learned rules are stored in policy.">
                            {policy.detector_rules.length === 0 ? (
                                <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-4 py-6 text-sm text-slate-600 dark:border-slate-700 dark:bg-slate-950/30 dark:text-slate-300">
                                    No custom detector rules are active yet. Accepted learned rules will appear here and be merged into the log detector at runtime.
                                </div>
                            ) : (
                                <div className="space-y-3">
                                    {policy.detector_rules.map((rule) => (
                                        <div key={rule.id} className={`rounded-2xl border p-4 ${detectorRuleTone(rule)}`}>
                                            <div className="flex flex-wrap items-center justify-between gap-3">
                                                <div>
                                                    <div className="text-sm font-semibold">{rule.name}</div>
                                                    <div className="mt-1 text-xs opacity-80">{rule.id} • {rule.domain} • {rule.incident_type}</div>
                                                </div>
                                                <div className="flex flex-wrap gap-2 text-[11px] font-semibold">
                                                    <span className="rounded-full border border-current/20 px-2 py-0.5">{rule.enabled ? 'enabled' : 'disabled'}</span>
                                                    <span className="rounded-full border border-current/20 px-2 py-0.5">{rule.severity}</span>
                                                    <span className="rounded-full border border-current/20 px-2 py-0.5">{rule.confidence || 'medium'}</span>
                                                    {rule.auto_created ? <span className="rounded-full border border-current/20 px-2 py-0.5">learned</span> : null}
                                                </div>
                                            </div>
                                            <div className="mt-3 grid gap-3 md:grid-cols-2">
                                                <div className="rounded-xl border border-current/15 bg-white/70 px-3 py-3 text-sm dark:bg-slate-950/20">
                                                    <div className="text-xs uppercase tracking-wide opacity-70">Signal Key</div>
                                                    <div className="mt-1 font-medium">{rule.signal_key || '—'}</div>
                                                </div>
                                                <div className="rounded-xl border border-current/15 bg-white/70 px-3 py-3 text-sm dark:bg-slate-950/20">
                                                    <div className="text-xs uppercase tracking-wide opacity-70">Source</div>
                                                    <div className="mt-1 font-medium">{rule.source || 'operator_defined'}</div>
                                                </div>
                                            </div>
                                            <div className="mt-3 rounded-xl border border-current/15 bg-white/70 p-3 text-xs dark:bg-slate-950/20">
                                                <div className="mb-2 uppercase tracking-wide opacity-70">Loki Query</div>
                                                <code className="whitespace-pre-wrap break-all">{rule.query}</code>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </SectionCard>
                        </div>
                        </>
                        ) : null}
                    </>
                ) : null}
            </div>
        </div>
    )
}

export default SRESmartBotSettingsPage
