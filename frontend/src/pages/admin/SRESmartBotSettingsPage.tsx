import { adminService } from '@/services/adminService'
import type { SREAgentRuntimeProbeResponse, SRESmartBotAgentRuntimeConfig, SRESmartBotChannelProvider, SRESmartBotMCPServer, SRESmartBotPolicyConfig } from '@/types'
import React, { useEffect, useState } from 'react'
import toast from 'react-hot-toast'

import {
    ChannelProvidersPanel,
    GeneralSettingsPanel,
    MCPAISettingsPanel,
    RulesSettingsOverviewPanel,
} from './sreSmartBotSettingsPanels'
import { normalizePolicy, SETTINGS_TABS, type SettingsTabId } from './sreSmartBotSettingsShared'

const SRESmartBotSettingsPage: React.FC = () => {
    const [policy, setPolicy] = useState<SRESmartBotPolicyConfig | null>(null)
    const [recommendedPolicy, setRecommendedPolicy] = useState<SRESmartBotPolicyConfig | null>(null)
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [channelMutationInFlight, setChannelMutationInFlight] = useState(false)
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

    const addChannelProvider = async (kind: string) => {
        let nextPolicy: SRESmartBotPolicyConfig | null = null
        setPolicy((current) => {
            if (!current) return current
            const normalizedKind = (kind || '').trim().toLowerCase() || 'custom'
            const provider: SRESmartBotChannelProvider = {
                id: crypto.randomUUID(),
                name: normalizedKind === 'in_app' ? 'In-App Channel' : `${normalizedKind.charAt(0).toUpperCase()}${normalizedKind.slice(1)} Provider`,
                kind: normalizedKind,
                enabled: true,
                supports_interactive_approval: normalizedKind === 'in_app' || normalizedKind === 'slack' || normalizedKind === 'teams' || normalizedKind === 'telegram' || normalizedKind === 'whatsapp',
                config_ref: `${normalizedKind}://default`,
            }
            nextPolicy = { ...current, channel_providers: [...current.channel_providers, provider] }
            return nextPolicy
        })
        setActiveTab('channels')

        if (!nextPolicy) return

        try {
            setChannelMutationInFlight(true)
            setSaving(true)
            setError(null)
            const response = await adminService.updateSRESmartBotPolicy(nextPolicy)
            setPolicy(normalizePolicy(response))
            toast.success('Channel provider added')
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to add channel provider'
            setError(message)
            toast.error(message)
            void loadPolicy()
        } finally {
            setChannelMutationInFlight(false)
            setSaving(false)
        }
    }

    const removeChannelProvider = async (index: number) => {
        let nextPolicy: SRESmartBotPolicyConfig | null = null
        setPolicy((current) => {
            if (!current) return current
            const channelProviders = [...current.channel_providers]
            channelProviders.splice(index, 1)
            nextPolicy = { ...current, channel_providers: channelProviders }
            return nextPolicy
        })

        if (!nextPolicy) return

        try {
            setChannelMutationInFlight(true)
            setSaving(true)
            setError(null)
            const response = await adminService.updateSRESmartBotPolicy(nextPolicy)
            setPolicy(normalizePolicy(response))
            toast.success('Channel provider removed')
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to remove channel provider'
            setError(message)
            toast.error(message)
            void loadPolicy()
        } finally {
            setChannelMutationInFlight(false)
            setSaving(false)
        }
    }

    const persistChannelProvider = async (index: number, provider: SRESmartBotChannelProvider) => {
        let nextPolicy: SRESmartBotPolicyConfig | null = null
        setPolicy((current) => {
            if (!current) return current
            const channelProviders = [...current.channel_providers]
            channelProviders[index] = provider
            nextPolicy = { ...current, channel_providers: channelProviders }
            return nextPolicy
        })

        if (!nextPolicy) return

        try {
            setChannelMutationInFlight(true)
            setSaving(true)
            setError(null)
            const response = await adminService.updateSRESmartBotPolicy(nextPolicy)
            setPolicy(normalizePolicy(response))
            toast.success('Channel provider saved')
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to persist channel provider update'
            setError(message)
            toast.error(message)
            void loadPolicy()
            throw err
        } finally {
            setChannelMutationInFlight(false)
            setSaving(false)
        }
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
                                            className={`rounded-t-xl border-b-2 px-4 py-3 text-left transition focus:outline-none focus:ring-2 focus:ring-sky-200 dark:focus:ring-sky-900 ${active
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

                        {activeTab === 'general' ? <GeneralSettingsPanel policy={policy} updatePolicy={updatePolicy} /> : null}

                        {activeTab === 'channels' ? <ChannelProvidersPanel policy={policy} updateChannelProvider={updateChannelProvider} addChannelProvider={(kind) => void addChannelProvider(kind)} removeChannelProvider={(index) => void removeChannelProvider(index)} onChannelProviderSave={(index, provider) => persistChannelProvider(index, provider)} isMutating={channelMutationInFlight} /> : null}

                        {activeTab === 'mcp-ai' ? (
                            <MCPAISettingsPanel
                                policy={policy}
                                recommendedPolicy={recommendedPolicy}
                                showRecommendedAgentRuntimeHint={showRecommendedAgentRuntimeHint}
                                applyRecommendedAgentRuntime={applyRecommendedAgentRuntime}
                                probeAgentRuntime={() => void probeAgentRuntime()}
                                probing={probing}
                                probeResult={probeResult}
                                updateMCPServer={updateMCPServer}
                                updateAgentRuntime={updateAgentRuntime}
                            />
                        ) : null}

                        {activeTab === 'rules' ? <RulesSettingsOverviewPanel /> : null}
                    </>
                ) : null}
            </div>
        </div>
    )
}

export default SRESmartBotSettingsPage
