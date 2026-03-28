import type {
    SRESmartBotAgentRuntimeConfig,
    SRESmartBotChannelProvider,
    SRESmartBotDetectorRule,
    SRESmartBotMCPServer,
    SRESmartBotOperatorRule,
    SRESmartBotPolicyConfig,
} from '@/types'
import React from 'react'

export const DOMAIN_OPTIONS = [
    'infrastructure',
    'runtime_services',
    'application_services',
    'network_ingress',
    'identity_security',
    'release_configuration',
    'operator_channels',
]

export const ENVIRONMENT_OPTIONS = ['demo', 'development', 'staging', 'production']
export const MCP_KIND_OPTIONS = ['observability', 'kubernetes', 'oci', 'database', 'release', 'chat', 'custom']
export const MCP_TRANSPORT_OPTIONS = ['embedded', 'http', 'stdio', 'custom']
export const AGENT_PROVIDER_OPTIONS = ['custom', 'ollama', 'openai', 'none']
export const DETECTOR_LEARNING_MODE_OPTIONS = ['disabled', 'suggest_only', 'training_auto_create'] as const
export const SETTINGS_TABS = [
    { id: 'general', label: 'General', description: 'Identity, automation, and domain scope' },
    { id: 'channels', label: 'Channels', description: 'Operator channel providers' },
    { id: 'mcp-ai', label: 'MCP & AI', description: 'Tool servers and agent runtime' },
    { id: 'rules', label: 'Rules', description: 'Operator-defined policy rules' },
] as const

export type SettingsTabId = (typeof SETTINGS_TABS)[number]['id']

export const SectionCard: React.FC<{ title: string; subtitle?: string; children: React.ReactNode }> = ({ title, subtitle, children }) => (
    <section className="rounded-2xl border border-slate-200 bg-white/90 p-5 shadow-sm dark:border-slate-800 dark:bg-slate-900/85">
        <div className="mb-4">
            <h2 className="text-sm font-semibold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">{title}</h2>
            {subtitle ? <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">{subtitle}</p> : null}
        </div>
        {children}
    </section>
)

export const inputClass = 'w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900'
export const labelClass = 'text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400'

export const ToggleField: React.FC<{
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

export const parseLabelMap = (value: string): Record<string, string> => {
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

export const stringifyLabelMap = (value?: Record<string, string>) =>
    Object.entries(value || {})
        .map(([key, item]) => `${key}=${item}`)
        .join('\n')

export const normalizePolicy = (policy: SRESmartBotPolicyConfig): SRESmartBotPolicyConfig => ({
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

export const detectorRuleTone = (rule: SRESmartBotDetectorRule) => {
    if (!rule.enabled) return 'border-slate-200 bg-slate-50 text-slate-700 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300'
    if ((rule.severity || '').toLowerCase() === 'critical') return 'border-rose-200 bg-rose-50 text-rose-900 dark:border-rose-900/40 dark:bg-rose-950/30 dark:text-rose-200'
    if ((rule.severity || '').toLowerCase() === 'warning') return 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200'
    return 'border-sky-200 bg-sky-50 text-sky-900 dark:border-sky-900/40 dark:bg-sky-950/30 dark:text-sky-200'
}

export type UpdatePolicy = <K extends keyof SRESmartBotPolicyConfig>(key: K, value: SRESmartBotPolicyConfig[K]) => void
export type UpdateChannelProvider = (index: number, updates: Partial<SRESmartBotChannelProvider>) => void
export type UpdateRule = (index: number, updates: Partial<SRESmartBotOperatorRule>) => void
export type UpdateDetectorRule = (index: number, updates: Partial<SRESmartBotDetectorRule>) => void
export type AddRule = () => void
export type AddRuleFromJson = (rule: Partial<SRESmartBotOperatorRule>) => void
export type RemoveRule = (index: number) => void
export type UpdateOperatorRuleWithIndex = (index: number, updates: Partial<SRESmartBotOperatorRule>) => void
export type UpdateMCPServer = (index: number, updates: Partial<SRESmartBotMCPServer>) => void
export type UpdateAgentRuntime = (updates: Partial<SRESmartBotAgentRuntimeConfig>) => void