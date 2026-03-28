import Drawer from '@/components/ui/Drawer'
import { adminService } from '@/services/adminService'
import type { SREActionAttempt, SREAgentDraftResponse, SREAgentIncidentScorecardResponse, SREAgentIncidentSnapshotResponse, SREAgentInterpretationResponse, SREAgentSeverityResponse, SREAgentSuggestedActionResponse, SREAgentTriageResponse, SREApproval, SREDemoScenario, SREEvidence, SREFinding, SREIncident, SREIncidentWorkspaceResponse, SREMCPToolDescriptor, SREMCPToolInvocationResponse, SRERemediationPack, SRERemediationPackRun, SRESmartBotChannelProvider, SRESmartBotPolicyConfig } from '@/types'
import React, { useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'

import { deriveAsyncBacklogInsight, deriveMessagingConsumerInsight, deriveMessagingTransportInsight, isAsyncBacklogIncidentType, isMessagingConsumerIncidentType } from './sreSmartBotAsyncSummary'
import {
    actionTone,
    asArrayOfRecords,
    asRecord,
    AsyncBacklogInsightContent,
    buildApprovalRequestMessage,
    EmptyState,
    formatDateTime,
    inputClass,
    MessagingConsumerInsightContent,
    MessagingTransportInsightContent,
    prettifyTrendLabel,
    prettyJson,
    relativeTime,
    renderAgentEvidenceRefs,
    renderMCPToolResult,
    SectionCard,
    severityTone,
    statusTone,
} from './sreSmartBotIncidentPageShared'
import { SREIncidentTimeline } from './sreSmartBotIncidentTimeline'

type IncidentDrawerTab = 'summary' | 'ai' | 'signals' | 'actions'

const drawerTabs: Array<{ id: IncidentDrawerTab; label: string; hint: string }> = [
    { id: 'summary', label: 'Summary', hint: 'Snapshot, overview, and email-ready narrative' },
    { id: 'ai', label: 'AI Workspace', hint: 'Grounded MCP, draft hypotheses, and local interpretation' },
    { id: 'signals', label: 'Signals', hint: 'Findings and evidence captured for this thread' },
    { id: 'actions', label: 'Actions', hint: 'Action attempts, approvals, and operator controls' },
]

const SRESmartBotIncidentsPage: React.FC = () => {
    const [searchParams, setSearchParams] = useSearchParams()
    const navigate = useNavigate()
    const [incidents, setIncidents] = useState<SREIncident[]>([])
    const [demoScenarios, setDemoScenarios] = useState<SREDemoScenario[]>([])
    const [total, setTotal] = useState(0)
    const [loading, setLoading] = useState(true)
    const [demoLoading, setDemoLoading] = useState(true)
    const [generatingDemo, setGeneratingDemo] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [statusFilter, setStatusFilter] = useState<string>('')
    const [severityFilter, setSeverityFilter] = useState<string>('')
    const [domainFilter, setDomainFilter] = useState('')
    const [search, setSearch] = useState('')
    const [selectedDemoScenarioId, setSelectedDemoScenarioId] = useState('ldap_timeout')
    const [selectedIncident, setSelectedIncident] = useState<SREIncident | null>(null)
    const [drawerTab, setDrawerTab] = useState<IncidentDrawerTab>('summary')
    const [drawerOpen, setDrawerOpen] = useState(false)
    const [drawerLoading, setDrawerLoading] = useState(false)
    const [demoDrawerOpen, setDemoDrawerOpen] = useState(false)
    const [mutatingActionId, setMutatingActionId] = useState<string | null>(null)
    const [mutatingApprovalId, setMutatingApprovalId] = useState<string | null>(null)
    const [approvalComments, setApprovalComments] = useState<Record<string, string>>({})
    const [policy, setPolicy] = useState<SRESmartBotPolicyConfig | null>(null)
    const [approvalRequestActionId, setApprovalRequestActionId] = useState<string | null>(null)
    const [approvalRequestMessage, setApprovalRequestMessage] = useState('')
    const [approvalRequestChannelProviderId, setApprovalRequestChannelProviderId] = useState('')
    const [emailingSummary, setEmailingSummary] = useState(false)
    const [proposingDetectorRule, setProposingDetectorRule] = useState(false)
    const [drawerError, setDrawerError] = useState<string | null>(null)
    const [findings, setFindings] = useState<SREFinding[]>([])
    const [evidence, setEvidence] = useState<SREEvidence[]>([])
    const [actions, setActions] = useState<SREActionAttempt[]>([])
    const [approvals, setApprovals] = useState<SREApproval[]>([])
    const [workspace, setWorkspace] = useState<SREIncidentWorkspaceResponse | null>(null)
    const [mcpTools, setMcpTools] = useState<SREMCPToolDescriptor[]>([])
    const [mcpResults, setMcpResults] = useState<Record<string, SREMCPToolInvocationResponse>>({})
    const [runningMCPToolKey, setRunningMCPToolKey] = useState<string | null>(null)
    const [agentDraft, setAgentDraft] = useState<SREAgentDraftResponse | null>(null)
    const [agentTriage, setAgentTriage] = useState<SREAgentTriageResponse | null>(null)
    const [agentSeverity, setAgentSeverity] = useState<SREAgentSeverityResponse | null>(null)
    const [agentScorecard, setAgentScorecard] = useState<SREAgentIncidentScorecardResponse | null>(null)
    const [agentSnapshot, setAgentSnapshot] = useState<SREAgentIncidentSnapshotResponse | null>(null)
    const [agentSuggestedAction, setAgentSuggestedAction] = useState<SREAgentSuggestedActionResponse | null>(null)
    const [generatingAgentSnapshot, setGeneratingAgentSnapshot] = useState(false)
    const [generatingAgentScorecard, setGeneratingAgentScorecard] = useState(false)
    const [generatingAgentSeverity, setGeneratingAgentSeverity] = useState(false)
    const [generatingAgentSuggestedAction, setGeneratingAgentSuggestedAction] = useState(false)
    const [generatingAgentTriage, setGeneratingAgentTriage] = useState(false)
    const [generatingAgentDraft, setGeneratingAgentDraft] = useState(false)
    const [agentInterpretation, setAgentInterpretation] = useState<SREAgentInterpretationResponse | null>(null)
    const [generatingInterpretation, setGeneratingInterpretation] = useState(false)
    const [remediationPacks, setRemediationPacks] = useState<SRERemediationPack[]>([])
    const [remediationRuns, setRemediationRuns] = useState<SRERemediationPackRun[]>([])
    const [mutatingRemediationPackKey, setMutatingRemediationPackKey] = useState<string | null>(null)
    const [selectedRemediationApprovalByPack, setSelectedRemediationApprovalByPack] = useState<Record<string, string>>({})

    const uniqueDomains = useMemo(() => {
        const values = new Set<string>()
        incidents.forEach((incident) => {
            if (incident.domain) values.add(incident.domain)
        })
        return Array.from(values).sort()
    }, [incidents])

    const actionsById = useMemo(() => {
        return actions.reduce<Record<string, SREActionAttempt>>((acc, action) => {
            acc[action.id] = action
            return acc
        }, {})
    }, [actions])

    const selectedIncidentIdFromUrl = searchParams.get('incident') || ''

    const syncIncidentQuery = (incidentId?: string | null) => {
        const next = new URLSearchParams(searchParams)
        if (incidentId) {
            next.set('incident', incidentId)
        } else {
            next.delete('incident')
        }
        setSearchParams(next, { replace: true })
    }

    const loadIncidents = async () => {
        try {
            setLoading(true)
            setError(null)
            const response = await adminService.getSREIncidents({
                limit: 100,
                offset: 0,
                status: statusFilter || undefined,
                severity: severityFilter || undefined,
                domain: domainFilter || undefined,
                search: search.trim() || undefined,
            })
            setIncidents(response.incidents || [])
            setTotal(response.total || 0)
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load SRE incidents')
            setIncidents([])
            setTotal(0)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        void loadIncidents()
    }, [statusFilter, severityFilter, domainFilter])

    useEffect(() => {
        const loadDemoScenarios = async () => {
            try {
                setDemoLoading(true)
                const response = await adminService.getSREDemoScenarios()
                setDemoScenarios(response.scenarios || [])
                if ((response.scenarios || []).length > 0) {
                    setSelectedDemoScenarioId((current) => current || response.scenarios[0].id)
                }
            } catch (err) {
                toast.error(err instanceof Error ? err.message : 'Failed to load demo scenarios')
            } finally {
                setDemoLoading(false)
            }
        }
        void loadDemoScenarios()
    }, [])

    useEffect(() => {
        const loadPolicy = async () => {
            try {
                const response = await adminService.getSRESmartBotPolicy()
                setPolicy(response)
            } catch {
                // Keep the incident workspace usable even if policy metadata is unavailable.
            }
        }
        void loadPolicy()
    }, [])

    useEffect(() => {
        if (!selectedIncidentIdFromUrl) {
            return
        }
        if (selectedIncident?.id === selectedIncidentIdFromUrl && drawerOpen) {
            return
        }
        void loadIncidentDetail(
            selectedIncidentIdFromUrl,
            incidents.find((incident) => incident.id === selectedIncidentIdFromUrl),
            true,
        )
    }, [selectedIncidentIdFromUrl, incidents, selectedIncident?.id, drawerOpen])

    const loadIncidentDetail = async (incidentId: string, seedIncident?: SREIncident, openDrawer?: boolean) => {
        if (seedIncident) {
            setSelectedIncident(seedIncident)
        }
        if (openDrawer) {
            setDrawerOpen(true)
        }
        setDrawerTab('summary')
        setDrawerLoading(true)
        setDrawerError(null)
        setFindings([])
        setEvidence([])
        setActions([])
        setApprovals([])
        setWorkspace(null)
        setMcpTools([])
        setMcpResults({})
        setRemediationPacks([])
        setRemediationRuns([])
        setSelectedRemediationApprovalByPack({})
        setAgentDraft(null)
        setAgentTriage(null)
        setAgentSeverity(null)
        setAgentScorecard(null)
        setAgentSnapshot(null)
        setAgentSuggestedAction(null)
        setAgentInterpretation(null)
        setApprovalComments({})
        setApprovalRequestActionId(null)
        setApprovalRequestMessage('')
        setApprovalRequestChannelProviderId('')
        try {
            const [response, workspaceResponse, mcpToolResponse, remediationPackResponse, remediationRunResponse] = await Promise.all([
                adminService.getSREIncident(incidentId),
                adminService.getSREIncidentWorkspace(incidentId).catch(() => null),
                adminService.getSREIncidentMCPTools(incidentId).catch(() => ({ tools: [] })),
                adminService.getSREIncidentRemediationPacks(incidentId).catch(() => ({ packs: [] })),
                adminService.getSREIncidentRemediationPackRuns(incidentId).catch(() => ({ runs: [] })),
            ])
            setSelectedIncident(response.incident)
            setFindings(response.findings || [])
            setEvidence(response.evidence || [])
            setActions(response.action_attempts || [])
            setApprovals(response.approvals || [])
            setWorkspace(workspaceResponse)
            setMcpTools(mcpToolResponse.tools || [])
            setRemediationPacks(remediationPackResponse.packs || [])
            setRemediationRuns(remediationRunResponse.runs || [])
        } catch (err) {
            setDrawerError(err instanceof Error ? err.message : 'Failed to load incident details')
        } finally {
            setDrawerLoading(false)
        }
    }

    const openIncident = async (incident: SREIncident) => {
        syncIncidentQuery(incident.id)
        await loadIncidentDetail(incident.id, incident, true)
    }

    const closeDrawer = () => {
        syncIncidentQuery(null)
        setDrawerOpen(false)
        setDrawerTab('summary')
        setSelectedIncident(null)
        setDrawerError(null)
        setFindings([])
        setEvidence([])
        setActions([])
        setApprovals([])
        setWorkspace(null)
        setMcpTools([])
        setMcpResults({})
        setRemediationPacks([])
        setRemediationRuns([])
        setSelectedRemediationApprovalByPack({})
        setAgentDraft(null)
        setAgentTriage(null)
        setAgentSeverity(null)
        setAgentScorecard(null)
        setAgentSnapshot(null)
        setAgentSuggestedAction(null)
        setAgentInterpretation(null)
        setApprovalComments({})
        setApprovalRequestActionId(null)
        setApprovalRequestMessage('')
        setApprovalRequestChannelProviderId('')
    }

    const closeDemoDrawer = () => {
        setDemoDrawerOpen(false)
    }

    const handleStartApprovalRequest = (action: SREActionAttempt) => {
        if (!selectedIncident) return
        const enabledProviders = (policy?.channel_providers || []).filter((provider) => provider.enabled)
        const preferredProviders = enabledProviders.filter((provider) => provider.supports_interactive_approval)
        const selectedProvider = preferredProviders[0] || enabledProviders.find((provider) => provider.id === policy?.default_channel_provider_id) || enabledProviders[0]
        setApprovalRequestActionId(action.id)
        setApprovalRequestMessage(buildApprovalRequestMessage(selectedIncident, action))
        setApprovalRequestChannelProviderId(selectedProvider?.id || policy?.default_channel_provider_id || '')
    }

    const handleCancelApprovalRequest = () => {
        setApprovalRequestActionId(null)
        setApprovalRequestMessage('')
        setApprovalRequestChannelProviderId('')
    }

    const handleSubmitApprovalRequest = async (action: SREActionAttempt) => {
        if (!selectedIncident) return
        try {
            setMutatingActionId(action.id)
            await adminService.requestSREActionApproval(selectedIncident.id, action.id, {
                channel_provider_id: approvalRequestChannelProviderId || undefined,
                request_message: approvalRequestMessage.trim() || undefined,
            })
            toast.success('Approval requested')
            handleCancelApprovalRequest()
            await loadIncidentDetail(selectedIncident.id)
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to request approval')
        } finally {
            setMutatingActionId(null)
        }
    }

    const handleExecuteAction = async (action: SREActionAttempt) => {
        if (!selectedIncident) return
        try {
            setMutatingActionId(action.id)
            await adminService.executeSREAction(selectedIncident.id, action.id)
            toast.success('Action executed')
            await loadIncidentDetail(selectedIncident.id)
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to execute action')
        } finally {
            setMutatingActionId(null)
        }
    }

    const handleProposeDetectorRule = async () => {
        if (!selectedIncident) return
        try {
            setProposingDetectorRule(true)
            const suggestion = await adminService.proposeSREDetectorRuleSuggestion(selectedIncident.id)
            toast.success('Detector rule suggestion created')
            navigate(`/admin/operations/sre-smart-bot/detector-rules?suggestion=${encodeURIComponent(suggestion.id)}&incident=${encodeURIComponent(selectedIncident.id)}`)
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to propose detector rule')
        } finally {
            setProposingDetectorRule(false)
        }
    }

    const handleApprovalDecision = async (approval: SREApproval, decision: 'approved' | 'rejected') => {
        if (!selectedIncident) return
        try {
            setMutatingApprovalId(approval.id)
            const comment = (approvalComments[approval.id] || '').trim()
            await adminService.decideSREApproval(selectedIncident.id, approval.id, {
                decision,
                comment: comment || undefined,
            })
            toast.success(decision === 'approved' ? 'Approval granted' : 'Approval rejected')
            setApprovalComments((current) => {
                if (!(approval.id in current)) return current
                const next = { ...current }
                delete next[approval.id]
                return next
            })
            await loadIncidentDetail(selectedIncident.id)
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to update approval')
        } finally {
            setMutatingApprovalId(null)
        }
    }

    const handleEmailSummary = async () => {
        if (!selectedIncident) return
        try {
            setEmailingSummary(true)
            await adminService.emailSREIncidentSummary(selectedIncident.id)
            toast.success('Incident summary queued for admin email delivery')
            await loadIncidentDetail(selectedIncident.id)
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to email incident summary')
        } finally {
            setEmailingSummary(false)
        }
    }

    const handleRunMCPTool = async (tool: SREMCPToolDescriptor) => {
        if (!selectedIncident) return
        const toolKey = `${tool.server_id}:${tool.tool_name}`
        try {
            setRunningMCPToolKey(toolKey)
            const response = await adminService.invokeSREIncidentMCPTool(selectedIncident.id, {
                server_id: tool.server_id,
                tool_name: tool.tool_name,
            })
            setMcpResults((current) => ({ ...current, [toolKey]: response }))
            toast.success(`${tool.display_name} completed`)
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to run MCP tool')
        } finally {
            setRunningMCPToolKey(null)
        }
    }

    const handleDismissMCPToolResult = (toolKey: string) => {
        setMcpResults((current) => {
            if (!(toolKey in current)) return current
            const next = { ...current }
            delete next[toolKey]
            return next
        })
    }

    const handleGenerateAgentDraft = async () => {
        if (!selectedIncident) return
        try {
            setGeneratingAgentDraft(true)
            const response = await adminService.getSREIncidentAgentDraft(selectedIncident.id)
            setAgentDraft(response)
            toast.success('Draft hypothesis and investigation plan generated')
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to generate agent draft')
        } finally {
            setGeneratingAgentDraft(false)
        }
    }

    const handleGenerateAgentTriage = async () => {
        if (!selectedIncident) return
        try {
            setGeneratingAgentTriage(true)
            const response = await adminService.getSREIncidentAgentTriage(selectedIncident.id)
            setAgentTriage(response)
            toast.success('Triage snapshot generated')
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to generate triage snapshot')
        } finally {
            setGeneratingAgentTriage(false)
        }
    }

    const handleGenerateAgentSeverity = async () => {
        if (!selectedIncident) return
        try {
            setGeneratingAgentSeverity(true)
            const response = await adminService.getSREIncidentAgentSeverity(selectedIncident.id)
            setAgentSeverity(response)
            toast.success('Severity correlation generated')
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to generate severity correlation')
        } finally {
            setGeneratingAgentSeverity(false)
        }
    }

    const handleGenerateAgentScorecard = async () => {
        if (!selectedIncident) return
        try {
            setGeneratingAgentScorecard(true)
            const response = await adminService.getSREIncidentAgentScorecard(selectedIncident.id)
            setAgentScorecard(response)
            toast.success('Incident scorecard generated')
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to generate incident scorecard')
        } finally {
            setGeneratingAgentScorecard(false)
        }
    }

    const handleGenerateAgentSnapshot = async () => {
        if (!selectedIncident) return
        try {
            setGeneratingAgentSnapshot(true)
            const response = await adminService.getSREIncidentAgentSnapshot(selectedIncident.id)
            setAgentSnapshot(response)
            if (response.triage) setAgentTriage(response.triage)
            if (response.severity) setAgentSeverity(response.severity)
            if (response.scorecard) setAgentScorecard(response.scorecard)
            if (response.suggested_action) setAgentSuggestedAction(response.suggested_action)
            toast.success('AI snapshot generated')
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to generate AI snapshot')
        } finally {
            setGeneratingAgentSnapshot(false)
        }
    }

    const handleGenerateAgentSuggestedAction = async () => {
        if (!selectedIncident) return
        try {
            setGeneratingAgentSuggestedAction(true)
            const response = await adminService.getSREIncidentAgentSuggestedAction(selectedIncident.id)
            setAgentSuggestedAction(response)
            toast.success('Advisory suggested action generated')
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to generate advisory suggested action')
        } finally {
            setGeneratingAgentSuggestedAction(false)
        }
    }

    const handleGenerateInterpretation = async () => {
        if (!selectedIncident) return
        try {
            setGeneratingInterpretation(true)
            const response = await adminService.getSREIncidentAgentInterpretation(selectedIncident.id)
            setAgentInterpretation(response)
            if (response.generated && response.cache_hit) {
                toast.success('Local model summaries loaded from cache')
            } else if (response.generated) {
                toast.success('Local model interpretation generated')
            } else {
                toast.success('Grounded fallback interpretation loaded')
            }
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to generate interpretation')
        } finally {
            setGeneratingInterpretation(false)
        }
    }

    const handleDryRunRemediationPack = async (pack: SRERemediationPack) => {
        if (!selectedIncident) return
        try {
            setMutatingRemediationPackKey(pack.key)
            await adminService.dryRunSREIncidentRemediationPack(selectedIncident.id, pack.key)
            toast.success(`Dry run completed for ${pack.name}`)
            await loadIncidentDetail(selectedIncident.id)
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to run remediation pack dry run')
        } finally {
            setMutatingRemediationPackKey(null)
        }
    }

    const handleExecuteRemediationPack = async (pack: SRERemediationPack) => {
        if (!selectedIncident) return
        try {
            setMutatingRemediationPackKey(pack.key)
            const approvedApproval = approvals.find((approval) => (
                (approval.status || '').toLowerCase() === 'approved' && !!approval.decided_at
            ))
            const selectedApprovalId = selectedRemediationApprovalByPack[pack.key] || approvedApproval?.id
            if (pack.requires_approval && !selectedApprovalId) {
                toast.error('This remediation pack requires an approved approval decision before execution')
                return
            }
            await adminService.executeSREIncidentRemediationPack(selectedIncident.id, pack.key, {
                approval_id: pack.requires_approval ? selectedApprovalId : undefined,
            })
            toast.success(`Execution recorded for ${pack.name}`)
            await loadIncidentDetail(selectedIncident.id)
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to execute remediation pack')
        } finally {
            setMutatingRemediationPackKey(null)
        }
    }

    const handleGenerateDemoIncident = async () => {
        if (!selectedDemoScenarioId) return
        try {
            setGeneratingDemo(true)
            const response = await adminService.generateSREDemoIncident(selectedDemoScenarioId)
            const incident = response.incident
            toast.success('Demo incident generated')
            await loadIncidents()
            if (incident?.id) {
                syncIncidentQuery(incident.id)
                await loadIncidentDetail(incident.id, incident, true)
            }
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to generate demo incident')
        } finally {
            setGeneratingDemo(false)
        }
    }

    const handleCopyOperatorMessage = async () => {
        const message = (agentInterpretation?.operator_handoff_note || agentInterpretation?.operator_message_draft || '').trim()
        if (!message) {
            toast.error('No operator message is available to copy')
            return
        }
        try {
            await navigator.clipboard.writeText(message)
            toast.success('Operator message copied')
        } catch {
            toast.error('Failed to copy operator message')
        }
    }

    const canExecuteAction = (action: SREActionAttempt) =>
        ['reconcile_tenant_assets', 'review_provider_connectivity'].includes(action.action_key) &&
        ['proposed', 'approved'].includes((action.status || '').toLowerCase())

    const agentSeverityTone = (level?: string) => {
        switch ((level || '').toLowerCase()) {
        case 'critical':
            return 'border-rose-200 bg-rose-50 text-rose-900 dark:border-rose-900/40 dark:bg-rose-950/30 dark:text-rose-200'
        case 'high':
            return 'border-orange-200 bg-orange-50 text-orange-900 dark:border-orange-900/40 dark:bg-orange-950/30 dark:text-orange-200'
        case 'medium':
            return 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200'
        default:
            return 'border-emerald-200 bg-emerald-50 text-emerald-900 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-200'
        }
    }

    const pendingApprovalsCount = useMemo(
        () => approvals.filter((approval) => !approval.decided_at).length,
        [approvals],
    )

    const runnableActionsCount = useMemo(
        () => actions.filter((action) => canExecuteAction(action)).length,
        [actions],
    )

    const approvalCapableChannelProviders = useMemo<SRESmartBotChannelProvider[]>(() => {
        const enabledProviders = (policy?.channel_providers || []).filter((provider) => provider.enabled)
        const interactiveProviders = enabledProviders.filter((provider) => provider.supports_interactive_approval)
        return interactiveProviders.length > 0 ? interactiveProviders : enabledProviders
    }, [policy])

    const summaryEmailActions = useMemo(
        () => actions.filter((action) => action.action_key === 'email_incident_summary'),
        [actions],
    )

    const approvedApprovals = useMemo(
        () => approvals.filter((approval) => (approval.status || '').toLowerCase() === 'approved' && !!approval.decided_at),
        [approvals],
    )

    const latestRemediationRunByPack = useMemo(() => {
        return remediationRuns.reduce<Record<string, SRERemediationPackRun>>((acc, run) => {
            if (!acc[run.pack_key]) {
                acc[run.pack_key] = run
                return acc
            }
            if (new Date(run.created_at).getTime() > new Date(acc[run.pack_key].created_at).getTime()) {
                acc[run.pack_key] = run
            }
            return acc
        }, {})
    }, [remediationRuns])

    const httpSignalsRecentTool = useMemo(
        () => mcpTools.find((tool) => tool.tool_name === 'http_signals.recent'),
        [mcpTools],
    )

    const httpSignalsHistoryTool = useMemo(
        () => mcpTools.find((tool) => tool.tool_name === 'http_signals.history'),
        [mcpTools],
    )

    const httpSignalsRecentKey = httpSignalsRecentTool ? `${httpSignalsRecentTool.server_id}:${httpSignalsRecentTool.tool_name}` : ''
    const httpSignalsHistoryKey = httpSignalsHistoryTool ? `${httpSignalsHistoryTool.server_id}:${httpSignalsHistoryTool.tool_name}` : ''

    const httpSignalsRecentResult = httpSignalsRecentKey ? mcpResults[httpSignalsRecentKey] : undefined
    const httpSignalsHistoryResult = httpSignalsHistoryKey ? mcpResults[httpSignalsHistoryKey] : undefined
    const asyncBacklogTool = useMemo(
        () => mcpTools.find((tool) => tool.tool_name === 'async_backlog.recent'),
        [mcpTools],
    )
    const asyncBacklogKey = asyncBacklogTool ? `${asyncBacklogTool.server_id}:${asyncBacklogTool.tool_name}` : ''
    const asyncBacklogResult = asyncBacklogKey ? mcpResults[asyncBacklogKey] : undefined
    const messagingTransportTool = useMemo(
        () => mcpTools.find((tool) => tool.tool_name === 'messaging_transport.recent'),
        [mcpTools],
    )
    const messagingTransportKey = messagingTransportTool ? `${messagingTransportTool.server_id}:${messagingTransportTool.tool_name}` : ''
    const messagingTransportResult = messagingTransportKey ? mcpResults[messagingTransportKey] : undefined
    const messagingConsumersTool = useMemo(
        () => mcpTools.find((tool) => tool.tool_name === 'messaging_consumers.recent'),
        [mcpTools],
    )
    const messagingConsumersKey = messagingConsumersTool ? `${messagingConsumersTool.server_id}:${messagingConsumersTool.tool_name}` : ''
    const messagingConsumersResult = messagingConsumersKey ? mcpResults[messagingConsumersKey] : undefined

    const summaryDeliveryStats = useMemo(() => {
        const successful = summaryEmailActions.filter((action) => (action.status || '').toLowerCase() === 'completed')
        const latest = summaryEmailActions[0]
        const latestPayload = asRecord(latest?.result_payload)
        return {
            total: summaryEmailActions.length,
            successful: successful.length,
            latest,
            latestRecipients: Array.isArray(latestPayload?.recipients) ? latestPayload?.recipients as string[] : [],
            latestSentCount: typeof latestPayload?.sent_count === 'number' ? latestPayload.sent_count : 0,
        }
    }, [summaryEmailActions])

    const httpSignalsSummary = useMemo(() => {
        const recentPayload = asRecord(httpSignalsRecentResult?.payload)
        const historyPayload = asRecord(httpSignalsHistoryResult?.payload)
        const windows = asArrayOfRecords(historyPayload?.windows)
        return {
            recentRequestCount: typeof recentPayload?.request_count === 'number' ? recentPayload.request_count : 0,
            recentErrorRatePercent: typeof recentPayload?.error_rate_percent === 'number' ? recentPayload.error_rate_percent : 0,
            recentAverageLatencyMs: typeof recentPayload?.average_latency_ms === 'number' ? recentPayload.average_latency_ms : 0,
            historyTrend: typeof historyPayload?.trend === 'string' ? historyPayload.trend : '',
            historyAverageLatencyMs: typeof historyPayload?.average_latency_ms === 'number' ? historyPayload.average_latency_ms : 0,
            historyAverageErrorRatePercent: typeof historyPayload?.average_error_rate_percent === 'number' ? historyPayload.average_error_rate_percent : 0,
            historyPeakRequestCount: typeof historyPayload?.peak_request_count === 'number' ? historyPayload.peak_request_count : 0,
            historyWindowCount: windows.length,
            latestHistoryWindowEndedAt: typeof windows[0]?.window_ended_at === 'string' ? windows[0].window_ended_at : '',
        }
    }, [httpSignalsRecentResult, httpSignalsHistoryResult])

    const asyncBacklogInsight = useMemo(
        () => deriveAsyncBacklogInsight(selectedIncident?.incident_type, workspace, asyncBacklogResult),
        [selectedIncident?.incident_type, workspace, asyncBacklogResult],
    )

    const messagingTransportInsight = useMemo(
        () => deriveMessagingTransportInsight(workspace, messagingTransportResult),
        [workspace, messagingTransportResult],
    )

    const messagingConsumerInsight = useMemo(
        () => deriveMessagingConsumerInsight(selectedIncident?.incident_type, workspace, messagingConsumersResult),
        [selectedIncident?.incident_type, workspace, messagingConsumersResult],
    )

    const executiveSummary = useMemo(() => {
        if (!selectedIncident) return []
        const pendingApprovals = approvals.filter((approval) => !approval.decided_at).length
        const approvedActionsReady = actions.filter((action) =>
            ['reconcile_tenant_assets', 'review_provider_connectivity'].includes(action.action_key) &&
            (action.status || '').toLowerCase() === 'approved',
        ).length
        const topFinding = findings[0]?.title || findings[0]?.message || 'No finding titles recorded yet.'
        const latestAction = actions[0]
        const summary = [
            `${selectedIncident.display_name} is currently ${selectedIncident.status} with ${selectedIncident.severity} severity in ${selectedIncident.domain}.`,
            pendingApprovals > 0
                ? `${pendingApprovals} approval request${pendingApprovals === 1 ? '' : 's'} still need operator attention.`
                : 'There are no pending approval requests on this incident thread.',
            approvedActionsReady > 0
                ? `${approvedActionsReady} approved executable action${approvedActionsReady === 1 ? '' : 's'} can be run now.`
                : 'No approved executable actions are currently waiting to run.',
            `Most recent signal: ${topFinding}`,
            latestAction
                ? `Latest action activity: ${latestAction.action_key} is ${latestAction.status}.`
                : 'No remediation actions have been attempted yet.',
        ]

        if (isAsyncBacklogIncidentType(selectedIncident.incident_type) && asyncBacklogInsight) {
            if (messagingTransportInsight && (messagingTransportInsight.reconnects > 0 || messagingTransportInsight.disconnects > 0)) {
                summary.push(
                    `Messaging transport instability may be contributing to ${asyncBacklogInsight.displayName.toLowerCase()}: reconnects=${messagingTransportInsight.reconnects}, disconnects=${messagingTransportInsight.disconnects}.`,
                )
            } else {
                summary.push(`${asyncBacklogInsight.displayName} is above threshold without current messaging transport instability, which points more toward downstream processing congestion than bus connectivity.`)
            }
        }

        if (isMessagingConsumerIncidentType(selectedIncident.incident_type) && messagingConsumerInsight) {
            if (messagingTransportInsight && (messagingTransportInsight.reconnects > 0 || messagingTransportInsight.disconnects > 0)) {
                summary.push(
                    `${messagingConsumerInsight.displayName} is rising alongside transport instability: reconnects=${messagingTransportInsight.reconnects}, disconnects=${messagingTransportInsight.disconnects}, trend=${prettifyTrendLabel(messagingConsumerInsight.trend)}.`,
                )
            } else {
                summary.push(`${messagingConsumerInsight.displayName} is above threshold without current messaging transport instability, which points to localized consumer pressure rather than a broad bus fault.`)
            }
        }

        if (selectedIncident.incident_type === 'messaging_transport_degraded') {
            if (messagingConsumerInsight && messagingConsumerInsight.count > 0) {
                summary.push(
                    `Transport instability is occurring alongside consumer pressure in ${messagingConsumerInsight.targetRef}: count=${messagingConsumerInsight.count}, threshold=${messagingConsumerInsight.threshold}, trend=${prettifyTrendLabel(messagingConsumerInsight.trend)}.`,
                )
            } else if (asyncBacklogInsight && asyncBacklogInsight.count > 0) {
                summary.push(
                    `Transport instability is occurring alongside async pressure in ${asyncBacklogInsight.displayName.toLowerCase()}: count=${asyncBacklogInsight.count}, threshold=${asyncBacklogInsight.threshold}, trend=${prettifyTrendLabel(asyncBacklogInsight.trend)}.`,
                )
            } else {
                summary.push('Transport instability is currently visible without major async backlog buildup, which suggests an early-stage messaging issue rather than sustained queue congestion.')
            }
        }

        return summary
    }, [selectedIncident, approvals, actions, findings, asyncBacklogInsight, messagingConsumerInsight, messagingTransportInsight])

    useEffect(() => {
        const autoLoadGoldenSignalContext = async () => {
            if (!selectedIncident || !drawerOpen || drawerLoading) return
            if (!selectedIncident.domain?.includes('golden_signals') && !selectedIncident.incident_type?.includes('golden_signals')) return

            const shouldLoadAsyncBacklog = isAsyncBacklogIncidentType(selectedIncident.incident_type)
            const shouldLoadMessagingConsumers = isMessagingConsumerIncidentType(selectedIncident.incident_type)
            const shouldLoadMessagingTransport = selectedIncident.incident_type === 'messaging_transport_degraded' || shouldLoadAsyncBacklog || shouldLoadMessagingConsumers
            const toolsToRun = [httpSignalsRecentTool, httpSignalsHistoryTool, shouldLoadAsyncBacklog ? asyncBacklogTool : undefined, shouldLoadMessagingConsumers ? messagingConsumersTool : undefined, shouldLoadMessagingTransport ? messagingTransportTool : undefined].filter(
                (tool): tool is SREMCPToolDescriptor => !!tool && !mcpResults[`${tool.server_id}:${tool.tool_name}`],
            )
            if (toolsToRun.length === 0) return

            for (const tool of toolsToRun) {
                try {
                    const response = await adminService.invokeSREIncidentMCPTool(selectedIncident.id, {
                        server_id: tool.server_id,
                        tool_name: tool.tool_name,
                    })
                    const toolKey = `${tool.server_id}:${tool.tool_name}`
                    setMcpResults((current) => ({ ...current, [toolKey]: response }))
                } catch {
                    // Keep summary loading resilient; operators can still run the tool manually in AI Workspace.
                }
            }
        }

        void autoLoadGoldenSignalContext()
    }, [
        selectedIncident,
        drawerOpen,
        drawerLoading,
        httpSignalsRecentTool,
        httpSignalsHistoryTool,
        asyncBacklogTool,
        messagingConsumersTool,
        messagingTransportTool,
        mcpResults,
    ])

    const selectedDemoScenario = useMemo(
        () => demoScenarios.find((scenario) => scenario.id === selectedDemoScenarioId) || null,
        [demoScenarios, selectedDemoScenarioId],
    )

    return (
        <div className="min-h-full w-full bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.08),_transparent_30%),linear-gradient(180deg,_#f8fafc_0%,_#eef2ff_100%)] px-4 py-6 text-slate-900 sm:px-6 lg:px-8 dark:bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.16),_transparent_24%),linear-gradient(180deg,_#020617_0%,_#0f172a_100%)] dark:text-slate-100">
            <div className="w-full space-y-6">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                    <div>
                        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">Operations</p>
                        <h1 className="mt-2 text-3xl font-semibold tracking-tight">SRE Smart Bot Incidents</h1>
                        <p className="mt-2 max-w-3xl text-sm text-slate-600 dark:text-slate-400">
                            Review active and recent SRE Smart Bot incidents, inspect evidence, and follow the action and approval trail.
                        </p>
                    </div>
                    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-end">
                        <div className="flex flex-wrap items-center gap-2 sm:flex-nowrap">
                            <Link
                                to="/admin/operations/sre-smart-bot/approvals"
                                className="inline-flex h-10 items-center justify-center rounded-xl border border-slate-300 bg-white px-4 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                            >
                                Approvals
                            </Link>
                            <Link
                                to="/admin/operations/sre-smart-bot/settings"
                                className="inline-flex h-10 items-center justify-center rounded-xl border border-slate-300 bg-white px-4 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                            >
                                Settings
                            </Link>
                            <button
                                onClick={() => setDemoDrawerOpen(true)}
                                className="inline-flex h-10 items-center justify-center rounded-xl border border-slate-300 bg-white px-4 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                            >
                                Demo Scenarios
                            </button>
                        </div>
                        <div className="flex flex-wrap items-center gap-2 sm:flex-nowrap">
                            <div className="inline-flex h-10 items-center gap-3 rounded-xl border border-slate-200 bg-white/80 px-4 text-sm shadow-sm dark:border-slate-800 dark:bg-slate-900/70">
                                <div className="text-xs uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">Incidents</div>
                                <div className="text-lg font-semibold text-slate-900 dark:text-white">{total}</div>
                            </div>
                            <button
                                onClick={() => void loadIncidents()}
                                className="inline-flex h-10 items-center justify-center rounded-xl border border-slate-300 bg-white px-4 text-sm font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                            >
                                Refresh
                            </button>
                        </div>
                    </div>
                </div>

                <SectionCard title="Filters" subtitle="Use narrow filters for incident review, or search by summary, type, or source.">
                    <div className="grid gap-4 md:grid-cols-2 2xl:grid-cols-5">
                        <label className="space-y-2">
                            <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Search</span>
                            <input
                                value={search}
                                onChange={(e) => setSearch(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter') {
                                        void loadIncidents()
                                    }
                                }}
                                placeholder="incident, source, summary..."
                                className="w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900"
                            />
                        </label>
                        <label className="space-y-2">
                            <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Status</span>
                            <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className="w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900">
                                <option value="">All statuses</option>
                                <option value="observed">Observed</option>
                                <option value="triaged">Triaged</option>
                                <option value="contained">Contained</option>
                                <option value="recovering">Recovering</option>
                                <option value="resolved">Resolved</option>
                                <option value="suppressed">Suppressed</option>
                                <option value="escalated">Escalated</option>
                            </select>
                        </label>
                        <label className="space-y-2">
                            <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Severity</span>
                            <select value={severityFilter} onChange={(e) => setSeverityFilter(e.target.value)} className="w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900">
                                <option value="">All severities</option>
                                <option value="info">Info</option>
                                <option value="warning">Warning</option>
                                <option value="critical">Critical</option>
                            </select>
                        </label>
                        <label className="space-y-2">
                            <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Domain</span>
                            <select value={domainFilter} onChange={(e) => setDomainFilter(e.target.value)} className="w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900">
                                <option value="">All domains</option>
                                {uniqueDomains.map((domain) => (
                                    <option key={domain} value={domain}>{domain}</option>
                                ))}
                            </select>
                        </label>
                        <div className="flex items-end">
                            <button
                                onClick={() => void loadIncidents()}
                                className="w-full rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-sky-700 dark:bg-sky-500 dark:hover:bg-sky-400"
                            >
                                Apply Filters
                            </button>
                        </div>
                    </div>
                </SectionCard>

                <SectionCard title="Incident Ledger" subtitle="Newest incidents first. Select a row to inspect findings, evidence, actions, and approvals.">
                    {loading ? (
                        <div className="flex items-center justify-center rounded-xl border border-slate-200 bg-slate-50 px-4 py-16 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-900/60 dark:text-slate-300">
                            Loading SRE incidents...
                        </div>
                    ) : error ? (
                        <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-4 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/40 dark:text-rose-200">{error}</div>
                    ) : incidents.length === 0 ? (
                        <EmptyState title="No incidents found" description="Try widening the filters or wait for SRE Smart Bot signal ingestion to create the first incident threads." />
                    ) : (
                        <div className="space-y-4">
                            <div className="grid gap-3 md:hidden">
                                {incidents.map((incident) => (
                                    <button
                                        key={incident.id}
                                        type="button"
                                        className="w-full rounded-2xl border border-slate-200 bg-white/90 p-4 text-left shadow-sm transition hover:border-sky-300 hover:bg-sky-50/50 dark:border-slate-800 dark:bg-slate-950/60 dark:hover:border-sky-700 dark:hover:bg-sky-950/20"
                                        onClick={() => void openIncident(incident)}
                                    >
                                        <div className="flex flex-wrap items-start justify-between gap-3">
                                            <div className="min-w-0 flex-1">
                                                <div className="font-medium text-slate-900 dark:text-white">{incident.display_name}</div>
                                                <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">{incident.summary || incident.incident_type}</div>
                                            </div>
                                            <span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${severityTone[incident.severity]}`}>{incident.severity}</span>
                                        </div>
                                        <div className="mt-3 flex flex-wrap gap-2">
                                            <span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${statusTone[incident.status]}`}>{incident.status}</span>
                                            <span className="inline-flex rounded-full border border-slate-300 px-2.5 py-1 text-xs font-medium text-slate-700 dark:border-slate-700 dark:text-slate-300">{incident.domain}</span>
                                        </div>
                                        <div className="mt-3 grid gap-2 text-xs text-slate-500 dark:text-slate-500 sm:grid-cols-2">
                                            <div>Source: {incident.source || '—'}</div>
                                            <div>Last observed: {relativeTime(incident.last_observed_at)}</div>
                                        </div>
                                    </button>
                                ))}
                            </div>

                            <div className="hidden overflow-hidden rounded-2xl border border-slate-200 dark:border-slate-800 md:block">
                                <div className="overflow-x-auto">
                                    <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-800">
                                        <thead className="bg-slate-100/90 dark:bg-slate-900/90">
                                            <tr className="text-left text-xs uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">
                                                <th className="px-4 py-3">Incident</th>
                                                <th className="px-4 py-3">Domain</th>
                                                <th className="px-4 py-3">Severity</th>
                                                <th className="px-4 py-3">Status</th>
                                                <th className="px-4 py-3">Source</th>
                                                <th className="px-4 py-3">Last Observed</th>
                                            </tr>
                                        </thead>
                                        <tbody className="divide-y divide-slate-200 bg-white dark:divide-slate-800 dark:bg-slate-950/50">
                                            {incidents.map((incident) => (
                                                <tr key={incident.id} className="cursor-pointer transition hover:bg-sky-50/60 dark:hover:bg-sky-950/20" onClick={() => void openIncident(incident)}>
                                                    <td className="px-4 py-4 align-top">
                                                        <div className="max-w-xl">
                                                            <div className="font-medium text-slate-900 dark:text-white">{incident.display_name}</div>
                                                            <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">{incident.summary || incident.incident_type}</div>
                                                            <div className="mt-2 text-xs text-slate-500 dark:text-slate-500">{incident.incident_type}</div>
                                                        </div>
                                                    </td>
                                                    <td className="px-4 py-4 text-sm text-slate-700 dark:text-slate-300">{incident.domain}</td>
                                                    <td className="px-4 py-4">
                                                        <span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${severityTone[incident.severity]}`}>{incident.severity}</span>
                                                    </td>
                                                    <td className="px-4 py-4">
                                                        <span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${statusTone[incident.status]}`}>{incident.status}</span>
                                                    </td>
                                                    <td className="px-4 py-4 text-sm text-slate-700 dark:text-slate-300">{incident.source}</td>
                                                    <td className="px-4 py-4 text-sm text-slate-700 dark:text-slate-300">
                                                        <div>{relativeTime(incident.last_observed_at)}</div>
                                                        <div className="mt-1 text-xs text-slate-500 dark:text-slate-500">{formatDateTime(incident.last_observed_at)}</div>
                                                    </td>
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </div>
                            </div>
                        </div>
                    )}
                </SectionCard>
            </div>

            <Drawer
                isOpen={drawerOpen}
                onClose={closeDrawer}
                title={selectedIncident?.display_name || 'Incident Detail'}
                description={selectedIncident ? `${selectedIncident.domain} • ${selectedIncident.incident_type}` : undefined}
                width="60vw"
            >
                {drawerLoading ? (
                    <div className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-10 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-900/60 dark:text-slate-300">Loading incident details...</div>
                ) : drawerError ? (
                    <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-4 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/40 dark:text-rose-200">{drawerError}</div>
                ) : selectedIncident ? (
                    <div className="space-y-6">
                        <div className="rounded-2xl border border-slate-200 bg-white/90 shadow-sm dark:border-slate-800 dark:bg-slate-900/85">
                            <div
                                role="tablist"
                                aria-label="Incident detail sections"
                                className="flex flex-wrap gap-x-1 gap-y-2 border-b border-slate-200 px-3 pt-3 dark:border-slate-800"
                            >
                                {drawerTabs.map((tab) => {
                                    const active = drawerTab === tab.id
                                    return (
                                        <button
                                            key={tab.id}
                                            type="button"
                                            role="tab"
                                            id={`incident-tab-${tab.id}`}
                                            aria-selected={active}
                                            aria-controls={`incident-panel-${tab.id}`}
                                            onClick={() => setDrawerTab(tab.id)}
                                            className={`rounded-t-xl border-b-2 px-3 py-3 text-left transition focus:outline-none focus:ring-2 focus:ring-sky-200 dark:focus:ring-sky-900 ${active
                                                ? 'border-sky-500 bg-sky-50/80 text-sky-800 dark:border-sky-400 dark:bg-sky-950/30 dark:text-sky-200'
                                                : 'border-transparent bg-transparent text-slate-600 hover:border-slate-300 hover:text-slate-900 dark:text-slate-400 dark:hover:border-slate-700 dark:hover:text-slate-100'
                                                }`}
                                        >
                                            <div className="text-sm font-medium">{tab.label}</div>
                                        </button>
                                    )
                                })}
                            </div>
                        </div>

                        {drawerTab === 'summary' ? (
                            <>
                                <div role="tabpanel" id="incident-panel-summary" aria-labelledby="incident-tab-summary" className="space-y-6">
                                    <div className="rounded-2xl border border-slate-200 bg-slate-50/90 px-4 py-3 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300">
                                        {drawerTabs.find((tab) => tab.id === 'summary')?.hint}
                                    </div>
                                    <SectionCard title="Operator Snapshot" subtitle="A quick operational read on evidence volume, pending action proposals, and approval activity.">
                                        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
                                            <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Findings</div>
                                                <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{findings.length}</div>
                                                <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Signals recorded for this incident thread.</div>
                                            </div>
                                            <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Evidence</div>
                                                <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{evidence.length}</div>
                                                <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Stored snapshots and watcher summaries.</div>
                                            </div>
                                            <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Action Proposals</div>
                                                <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{actions.length}</div>
                                                <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">{actions.filter((action) => action.approval_required).length} require approval.</div>
                                            </div>
                                            <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Approvals</div>
                                                <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{approvals.length}</div>
                                                <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">{approvals.filter((approval) => !approval.decided_at).length} still awaiting decision.</div>
                                            </div>
                                            <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Summary Emails</div>
                                                <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{summaryDeliveryStats.successful}</div>
                                                <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">{summaryDeliveryStats.total} queued or attempted so far.</div>
                                            </div>
                                        </div>
                                    </SectionCard>

                                    <SectionCard title="Guided Remediation Packs" subtitle={`${remediationPacks.length} packs available for this incident type`}>
                                        {remediationPacks.length === 0 ? (
                                            <EmptyState title="No remediation packs available" description="No pack definitions are currently mapped to this incident type." />
                                        ) : (
                                            <div className="space-y-3">
                                                {remediationPacks.map((pack) => {
                                                    const latestRun = latestRemediationRunByPack[pack.key]
                                                    return (
                                                        <div key={pack.key} className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                            <div className="flex flex-wrap items-center justify-between gap-2">
                                                                <div>
                                                                    <div className="flex flex-wrap items-center gap-2">
                                                                        <div className="font-medium text-slate-900 dark:text-slate-100">{pack.name}</div>
                                                                        <span className="rounded-full border border-slate-300 px-2 py-0.5 text-[11px] font-semibold text-slate-700 dark:border-slate-700 dark:text-slate-300">{pack.key}</span>
                                                                        <span className="rounded-full border border-slate-300 px-2 py-0.5 text-[11px] font-semibold text-slate-700 dark:border-slate-700 dark:text-slate-300">risk: {pack.risk_tier}</span>
                                                                        {pack.requires_approval ? <span className="rounded-full border border-amber-300 bg-amber-50 px-2 py-0.5 text-[11px] font-semibold text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-200">approval required</span> : null}
                                                                    </div>
                                                                    <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">{pack.summary}</div>
                                                                </div>
                                                                <div className="flex flex-wrap gap-2">
                                                                    <button
                                                                        type="button"
                                                                        onClick={() => void handleDryRunRemediationPack(pack)}
                                                                        disabled={mutatingRemediationPackKey === pack.key}
                                                                        className="rounded-lg border border-sky-300 bg-sky-50 px-3 py-1.5 text-xs font-medium text-sky-700 transition hover:bg-sky-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-sky-800 dark:bg-sky-950/30 dark:text-sky-200 dark:hover:bg-sky-950/50"
                                                                    >
                                                                        {mutatingRemediationPackKey === pack.key ? 'Running...' : 'Dry Run'}
                                                                    </button>
                                                                    <button
                                                                        type="button"
                                                                        onClick={() => void handleExecuteRemediationPack(pack)}
                                                                        disabled={mutatingRemediationPackKey === pack.key}
                                                                        className="rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-emerald-500 dark:hover:bg-emerald-400"
                                                                    >
                                                                        {mutatingRemediationPackKey === pack.key ? 'Executing...' : 'Execute'}
                                                                    </button>
                                                                </div>
                                                            </div>
                                                            {pack.requires_approval ? (
                                                                <div className="mt-3">
                                                                    <label className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Approved Approval Record</label>
                                                                    <select
                                                                        value={selectedRemediationApprovalByPack[pack.key] || ''}
                                                                        onChange={(event) => setSelectedRemediationApprovalByPack((current) => ({ ...current, [pack.key]: event.target.value }))}
                                                                        className={`${inputClass} mt-2`}
                                                                    >
                                                                        <option value="">{approvedApprovals.length > 0 ? 'Use most recent approved record' : 'No approved records available'}</option>
                                                                        {approvedApprovals.map((approval) => (
                                                                            <option key={approval.id} value={approval.id}>
                                                                                {approval.id} • {approval.channel_provider_id} • {formatDateTime(approval.decided_at || approval.updated_at)}
                                                                            </option>
                                                                        ))}
                                                                    </select>
                                                                    <div className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                                                        This pack requires approval. Select an approved record explicitly, or keep default to use the latest approved one.
                                                                    </div>
                                                                </div>
                                                            ) : null}
                                                            {latestRun ? (
                                                                <div className="mt-3 rounded-lg border border-slate-200 bg-white/80 px-3 py-2 text-xs text-slate-600 dark:border-slate-800 dark:bg-slate-900/70 dark:text-slate-300">
                                                                    Latest run: {latestRun.run_kind} • {latestRun.status} • {formatDateTime(latestRun.created_at)}
                                                                    {latestRun.summary ? ` • ${latestRun.summary}` : ''}
                                                                </div>
                                                            ) : (
                                                                <div className="mt-3 rounded-lg border border-dashed border-slate-300 bg-white/70 px-3 py-2 text-xs text-slate-500 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-400">
                                                                    No runs recorded yet for this pack.
                                                                </div>
                                                            )}
                                                        </div>
                                                    )
                                                })}
                                            </div>
                                        )}
                                    </SectionCard>

                                    <SectionCard title="Operator And Bot Timeline" subtitle="One chronological thread of what the bot observed, what evidence it stored, and how operators responded.">
                                        <SREIncidentTimeline
                                            findings={findings}
                                            evidence={evidence}
                                            actions={actions}
                                            approvals={approvals}
                                            actionsById={actionsById}
                                        />
                                    </SectionCard>

                                    {httpSignalsRecentTool || httpSignalsHistoryTool ? (
                                        <SectionCard
                                            title="App Golden Signals"
                                            subtitle="Latest HTTP request health and retained trend context, surfaced directly in the incident summary."
                                        >
                                            {httpSignalsRecentResult || httpSignalsHistoryResult ? (
                                                <div className="space-y-4">
                                                    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                                                        <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                            <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Latest Requests</div>
                                                            <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{httpSignalsSummary.recentRequestCount}</div>
                                                            <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Most recent captured HTTP window.</div>
                                                        </div>
                                                        <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 dark:border-rose-900/40 dark:bg-rose-950/30">
                                                            <div className="text-xs uppercase tracking-wide text-rose-700 dark:text-rose-300">Latest Error Rate</div>
                                                            <div className="mt-2 text-2xl font-semibold text-rose-900 dark:text-rose-100">{httpSignalsSummary.recentErrorRatePercent}%</div>
                                                            <div className="mt-1 text-sm text-rose-700/80 dark:text-rose-200/80">Server-side pressure in the latest window.</div>
                                                        </div>
                                                        <div className="rounded-xl border border-cyan-200 bg-cyan-50 p-4 dark:border-cyan-900/40 dark:bg-cyan-950/30">
                                                            <div className="text-xs uppercase tracking-wide text-cyan-700 dark:text-cyan-300">Latest Avg Latency</div>
                                                            <div className="mt-2 text-2xl font-semibold text-cyan-900 dark:text-cyan-100">{httpSignalsSummary.recentAverageLatencyMs}ms</div>
                                                            <div className="mt-1 text-sm text-cyan-700/80 dark:text-cyan-200/80">Average request duration for the latest slice.</div>
                                                        </div>
                                                        <div className="rounded-xl border border-violet-200 bg-violet-50 p-4 dark:border-violet-900/40 dark:bg-violet-950/30">
                                                            <div className="text-xs uppercase tracking-wide text-violet-700 dark:text-violet-300">Trend Direction</div>
                                                            <div className="mt-2 text-2xl font-semibold capitalize text-violet-900 dark:text-violet-100">
                                                                {prettifyTrendLabel(httpSignalsSummary.historyTrend)}
                                                            </div>
                                                            <div className="mt-1 text-sm text-violet-700/80 dark:text-violet-200/80">
                                                                {httpSignalsSummary.historyWindowCount > 0
                                                                    ? `${httpSignalsSummary.historyWindowCount} retained windows available.`
                                                                    : 'History has not been captured yet.'}
                                                            </div>
                                                        </div>
                                                    </div>

                                                    {httpSignalsSummary.historyWindowCount > 0 ? (
                                                        <div className="grid gap-4 md:grid-cols-3">
                                                            <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">History Avg Latency</div>
                                                                <div className="mt-2 text-lg font-semibold text-slate-900 dark:text-slate-100">{httpSignalsSummary.historyAverageLatencyMs}ms</div>
                                                            </div>
                                                            <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">History Avg Error Rate</div>
                                                                <div className="mt-2 text-lg font-semibold text-slate-900 dark:text-slate-100">{httpSignalsSummary.historyAverageErrorRatePercent}%</div>
                                                            </div>
                                                            <div className="rounded-xl border border-slate-200 bg-white/90 p-4 dark:border-slate-800 dark:bg-slate-900/70">
                                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Peak Traffic</div>
                                                                <div className="mt-2 text-lg font-semibold text-slate-900 dark:text-slate-100">{httpSignalsSummary.historyPeakRequestCount}</div>
                                                                <div className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                                                    Latest history sample: {httpSignalsSummary.latestHistoryWindowEndedAt ? formatDateTime(httpSignalsSummary.latestHistoryWindowEndedAt) : '—'}
                                                                </div>
                                                            </div>
                                                        </div>
                                                    ) : null}
                                                </div>
                                            ) : (
                                                <EmptyState
                                                    title="HTTP signal context is available but not loaded yet"
                                                    description="This incident has HTTP golden-signal tooling enabled. Open the AI workspace tools if you want to inspect the raw MCP output directly."
                                                />
                                            )}
                                        </SectionCard>
                                    ) : null}

                                    {asyncBacklogInsight && isAsyncBacklogIncidentType(selectedIncident.incident_type) ? (
                                        <SectionCard
                                            title="Async Backlog Pressure"
                                            subtitle="Normalized backlog pressure for the affected async path, projected from stored evidence and tool snapshots."
                                        >
                                            <AsyncBacklogInsightContent insight={asyncBacklogInsight} />
                                        </SectionCard>
                                    ) : null}

                                    {messagingConsumerInsight && isMessagingConsumerIncidentType(selectedIncident.incident_type) ? (
                                        <SectionCard
                                            title="Messaging Consumer Pressure"
                                            subtitle="Normalized NATS consumer lag, stalled progress, or pending-ack pressure projected from stored evidence and tool snapshots."
                                        >
                                            <MessagingConsumerInsightContent insight={messagingConsumerInsight} />
                                        </SectionCard>
                                    ) : null}

                                    {messagingTransportInsight && (selectedIncident.incident_type === 'messaging_transport_degraded' || isAsyncBacklogIncidentType(selectedIncident.incident_type) || isMessagingConsumerIncidentType(selectedIncident.incident_type)) ? (
                                        <SectionCard
                                            title="Messaging Transport Health"
                                            subtitle="Current NATS transport stability for reconnect and disconnect pressure, correlated against async backlog context when available."
                                        >
                                            <MessagingTransportInsightContent insight={messagingTransportInsight} />
                                        </SectionCard>
                                    ) : null}

                                    <SectionCard title="Executive Summary" subtitle="A concise operator-ready narrative you can use in triage, handoffs, and email updates.">
                                        <div className="space-y-3">
                                            {executiveSummary.map((item, index) => (
                                                <div key={`${index}-${item}`} className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-950 dark:text-slate-300">
                                                    {item}
                                                </div>
                                            ))}
                                        </div>
                                        <div className="mt-4 flex flex-wrap gap-2">
                                            <button
                                                type="button"
                                                onClick={() => void handleEmailSummary()}
                                                disabled={emailingSummary}
                                                className="rounded-lg border border-sky-300 bg-sky-50 px-3 py-1.5 text-xs font-medium text-sky-700 transition hover:bg-sky-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-sky-800 dark:bg-sky-950/30 dark:text-sky-200 dark:hover:bg-sky-950/50"
                                            >
                                                {emailingSummary ? 'Queueing email...' : 'Email Summary to Admins'}
                                            </button>
                                        </div>
                                    </SectionCard>

                                    <SectionCard title="Summary Delivery" subtitle="Email-summary history for this incident, including the latest delivery attempt and recipients.">
                                        {summaryEmailActions.length === 0 ? (
                                            <EmptyState title="No summary emails sent yet" description="Use the executive summary action above to queue a summary email for admins." />
                                        ) : (
                                            <div className="space-y-3">
                                                {summaryEmailActions.map((action) => {
                                                    const payload = asRecord(action.result_payload)
                                                    const recipients = Array.isArray(payload?.recipients) ? payload.recipients as string[] : []
                                                    const sentCount = typeof payload?.sent_count === 'number' ? payload.sent_count : 0
                                                    const recipientCount = typeof payload?.recipient_count === 'number' ? payload.recipient_count : recipients.length
                                                    return (
                                                        <div key={action.id} className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                            <div className="flex flex-wrap items-center justify-between gap-3">
                                                                <div className="flex flex-wrap items-center gap-2">
                                                                    <div className="font-medium text-slate-900 dark:text-white">Admin summary email</div>
                                                                    <span className={`rounded-full border px-2 py-0.5 text-[11px] font-semibold ${actionTone(action.status)}`}>{action.status}</span>
                                                                </div>
                                                                <div className="text-xs text-slate-500 dark:text-slate-400">{formatDateTime(action.completed_at || action.requested_at)}</div>
                                                            </div>
                                                            <div className="mt-3 grid gap-3 text-sm md:grid-cols-3">
                                                                <div className="rounded-lg border border-slate-200 bg-white px-3 py-3 dark:border-slate-800 dark:bg-slate-900">
                                                                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Sent</div>
                                                                    <div className="mt-1 font-medium text-slate-900 dark:text-slate-100">{sentCount} / {recipientCount}</div>
                                                                </div>
                                                                <div className="rounded-lg border border-slate-200 bg-white px-3 py-3 dark:border-slate-800 dark:bg-slate-900">
                                                                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Requested by</div>
                                                                    <div className="mt-1 font-medium text-slate-900 dark:text-slate-100">{action.actor_id || 'system'}</div>
                                                                </div>
                                                                <div className="rounded-lg border border-slate-200 bg-white px-3 py-3 dark:border-slate-800 dark:bg-slate-900">
                                                                    <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Last activity</div>
                                                                    <div className="mt-1 font-medium text-slate-900 dark:text-slate-100">{relativeTime(action.completed_at || action.requested_at)}</div>
                                                                </div>
                                                            </div>
                                                            {recipients.length > 0 ? (
                                                                <div className="mt-3">
                                                                    <div className="mb-2 text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Recipients</div>
                                                                    <div className="flex flex-wrap gap-2">
                                                                        {recipients.map((recipient) => (
                                                                            <span key={`${action.id}-${recipient}`} className="inline-flex rounded-full border border-slate-300 bg-white px-2.5 py-1 text-xs font-medium text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">
                                                                                {recipient}
                                                                            </span>
                                                                        ))}
                                                                    </div>
                                                                </div>
                                                            ) : null}
                                                            {action.error_message ? (
                                                                <div className="mt-3 text-sm font-medium text-rose-700 dark:text-rose-300">{action.error_message}</div>
                                                            ) : null}
                                                        </div>
                                                    )
                                                })}
                                            </div>
                                        )}
                                    </SectionCard>

                                    <SectionCard title="Overview">
                                        <div className="grid gap-4 md:grid-cols-2">
                                            <div>
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Status</div>
                                                <div className="mt-2"><span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${statusTone[selectedIncident.status]}`}>{selectedIncident.status}</span></div>
                                            </div>
                                            <div>
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Severity</div>
                                                <div className="mt-2"><span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${severityTone[selectedIncident.severity]}`}>{selectedIncident.severity}</span></div>
                                            </div>
                                            <div className="text-sm text-slate-700 dark:text-slate-300"><span className="font-medium text-slate-900 dark:text-white">Source:</span> {selectedIncident.source}</div>
                                            <div className="text-sm text-slate-700 dark:text-slate-300"><span className="font-medium text-slate-900 dark:text-white">Confidence:</span> {selectedIncident.confidence}</div>
                                            <div className="text-sm text-slate-700 dark:text-slate-300"><span className="font-medium text-slate-900 dark:text-white">First observed:</span> {formatDateTime(selectedIncident.first_observed_at)}</div>
                                            <div className="text-sm text-slate-700 dark:text-slate-300"><span className="font-medium text-slate-900 dark:text-white">Last observed:</span> {formatDateTime(selectedIncident.last_observed_at)}</div>
                                        </div>
                                        <div className="mt-4 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-950 dark:text-slate-300">
                                            {selectedIncident.summary || 'No summary provided.'}
                                        </div>
                                        {selectedIncident.metadata ? (
                                            <div className="mt-4">
                                                <div className="mb-2 text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Metadata</div>
                                                <pre className="overflow-x-auto rounded-xl border border-slate-200 bg-slate-50 p-4 text-xs text-slate-700 dark:border-slate-800 dark:bg-slate-950 dark:text-slate-300">{prettyJson(selectedIncident.metadata)}</pre>
                                            </div>
                                        ) : null}
                                    </SectionCard>
                                </div>
                            </>
                        ) : null}

                        {drawerTab === 'signals' ? (
                            <>
                                <div role="tabpanel" id="incident-panel-signals" aria-labelledby="incident-tab-signals" className="space-y-6">
                                    <div className="rounded-2xl border border-slate-200 bg-slate-50/90 px-4 py-3 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300">
                                        {drawerTabs.find((tab) => tab.id === 'signals')?.hint}
                                    </div>
                                    {evidence.length === 0 ? (
                                        <div className="rounded-2xl border border-amber-200 bg-amber-50/90 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200">
                                            No evidence snapshots are stored for this incident yet. Findings were recorded, but this thread does not currently have additional evidence rows attached.
                                        </div>
                                    ) : null}
                                    <SectionCard title="Findings" subtitle={`${findings.length} signal observations`}>
                                        {findings.length === 0 ? <EmptyState title="No findings recorded" description="This incident has not stored any finding rows yet." /> : (
                                            <div className="space-y-3">
                                                {findings.map((finding) => (
                                                    <div key={finding.id} className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                        <div className="flex flex-wrap items-center gap-2">
                                                            <div className="font-medium text-slate-900 dark:text-white">{finding.title}</div>
                                                            <span className={`inline-flex rounded-full border px-2 py-0.5 text-[11px] font-semibold ${severityTone[finding.severity]}`}>{finding.severity}</span>
                                                        </div>
                                                        <p className="mt-2 text-sm text-slate-700 dark:text-slate-300">{finding.message}</p>
                                                        <div className="mt-2 text-xs text-slate-500 dark:text-slate-500">{finding.signal_type} • {finding.signal_key} • {formatDateTime(finding.occurred_at)}</div>
                                                    </div>
                                                ))}
                                            </div>
                                        )}
                                    </SectionCard>

                                    <SectionCard title="Evidence" subtitle={`${evidence.length} evidence records`}>
                                        <div className="mb-4 flex flex-wrap gap-2">
                                            <button
                                                type="button"
                                                onClick={() => void handleProposeDetectorRule()}
                                                disabled={proposingDetectorRule}
                                                className="rounded-lg border border-sky-300 bg-sky-50 px-3 py-1.5 text-xs font-medium text-sky-700 transition hover:bg-sky-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-sky-800 dark:bg-sky-950/30 dark:text-sky-200 dark:hover:bg-sky-950/50"
                                            >
                                                {proposingDetectorRule ? 'Proposing...' : 'Propose Detector Rule'}
                                            </button>
                                            <Link
                                                to={selectedIncident ? `/admin/operations/sre-smart-bot/detector-rules?incident=${encodeURIComponent(selectedIncident.id)}` : '/admin/operations/sre-smart-bot/detector-rules'}
                                                className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                                            >
                                                Open Detector Rules
                                            </Link>
                                        </div>
                                        {evidence.length === 0 ? <EmptyState title="No evidence recorded" description="Evidence capture is ready but this incident does not yet have stored evidence rows." /> : (
                                            <div className="space-y-3">
                                                {evidence.map((item) => (
                                                    <div key={item.id} className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                        <div className="font-medium text-slate-900 dark:text-white">{item.evidence_type}</div>
                                                        <p className="mt-2 text-sm text-slate-700 dark:text-slate-300">{item.summary}</p>
                                                        {item.payload ? <pre className="mt-3 overflow-x-auto rounded-lg border border-slate-200 bg-white p-3 text-xs text-slate-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300">{prettyJson(item.payload)}</pre> : null}
                                                    </div>
                                                ))}
                                            </div>
                                        )}
                                    </SectionCard>
                                </div>
                            </>
                        ) : null}

                        {drawerTab === 'actions' ? (
                            <>
                                <div role="tabpanel" id="incident-panel-actions" aria-labelledby="incident-tab-actions" className="space-y-6">
                                    <div className="rounded-2xl border border-slate-200 bg-slate-50/90 px-4 py-3 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300">
                                        {drawerTabs.find((tab) => tab.id === 'actions')?.hint}
                                    </div>
                                    <SectionCard title="Operator Control Center" subtitle="Drive the bot from one place: synthesize context, decide on actions, communicate updates, and route follow-up work.">
                                        <div className="grid gap-3 md:grid-cols-3">
                                            <div className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-4 dark:border-slate-800 dark:bg-slate-950">
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Pending approvals</div>
                                                <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-slate-100">{pendingApprovalsCount}</div>
                                                <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Approval requests awaiting an operator decision.</div>
                                            </div>
                                            <div className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-4 dark:border-slate-800 dark:bg-slate-950">
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Runnable actions</div>
                                                <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-slate-100">{runnableActionsCount}</div>
                                                <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Approved or executable recovery actions that can run now.</div>
                                            </div>
                                            <div className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-4 dark:border-slate-800 dark:bg-slate-950">
                                                <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Recommended tools</div>
                                                <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-slate-100">{workspace?.default_tool_bundle?.length || 0}</div>
                                                <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Bounded MCP calls suggested for this incident workspace.</div>
                                            </div>
                                        </div>

                                        <div className="mt-4 flex flex-wrap gap-2">
                                            <button
                                                type="button"
                                                onClick={() => void handleGenerateAgentSnapshot()}
                                                disabled={generatingAgentSnapshot}
                                                className="rounded-lg border border-emerald-300 bg-emerald-50 px-3 py-2 text-xs font-medium text-emerald-800 transition hover:bg-emerald-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-200 dark:hover:bg-emerald-950/50"
                                            >
                                                {generatingAgentSnapshot ? 'Generating...' : 'Generate AI Snapshot'}
                                            </button>
                                            <button
                                                type="button"
                                                onClick={() => void handleGenerateAgentSeverity()}
                                                disabled={generatingAgentSeverity}
                                                className="rounded-lg border border-amber-300 bg-amber-50 px-3 py-2 text-xs font-medium text-amber-900 transition hover:bg-amber-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-200 dark:hover:bg-amber-950/50"
                                            >
                                                {generatingAgentSeverity ? 'Generating...' : 'Generate Severity'}
                                            </button>
                                            <button
                                                type="button"
                                                onClick={() => void handleGenerateAgentScorecard()}
                                                disabled={generatingAgentScorecard}
                                                className="rounded-lg border border-rose-300 bg-rose-50 px-3 py-2 text-xs font-medium text-rose-800 transition hover:bg-rose-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-rose-800 dark:bg-rose-950/30 dark:text-rose-200 dark:hover:bg-rose-950/50"
                                            >
                                                {generatingAgentScorecard ? 'Generating...' : 'Generate Scorecard'}
                                            </button>
                                            <button
                                                type="button"
                                                onClick={() => void handleGenerateAgentTriage()}
                                                disabled={generatingAgentTriage}
                                                className="rounded-lg border border-cyan-300 bg-cyan-50 px-3 py-2 text-xs font-medium text-cyan-800 transition hover:bg-cyan-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-cyan-800 dark:bg-cyan-950/30 dark:text-cyan-200 dark:hover:bg-cyan-950/50"
                                            >
                                                {generatingAgentTriage ? 'Generating...' : 'Generate Triage'}
                                            </button>
                                            <button
                                                type="button"
                                                onClick={() => void handleGenerateAgentSuggestedAction()}
                                                disabled={generatingAgentSuggestedAction}
                                                className="rounded-lg border border-fuchsia-300 bg-fuchsia-50 px-3 py-2 text-xs font-medium text-fuchsia-800 transition hover:bg-fuchsia-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-fuchsia-800 dark:bg-fuchsia-950/30 dark:text-fuchsia-200 dark:hover:bg-fuchsia-950/50"
                                            >
                                                {generatingAgentSuggestedAction ? 'Generating...' : 'Generate Suggested Action'}
                                            </button>
                                            <button
                                                type="button"
                                                onClick={() => void handleGenerateAgentDraft()}
                                                disabled={generatingAgentDraft}
                                                className="rounded-lg border border-sky-300 bg-sky-50 px-3 py-2 text-xs font-medium text-sky-700 transition hover:bg-sky-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-sky-800 dark:bg-sky-950/30 dark:text-sky-200 dark:hover:bg-sky-950/50"
                                            >
                                                {generatingAgentDraft ? 'Generating...' : 'Generate Draft'}
                                            </button>
                                            <button
                                                type="button"
                                                onClick={() => void handleGenerateInterpretation()}
                                                disabled={generatingInterpretation}
                                                className="rounded-lg border border-violet-300 bg-violet-50 px-3 py-2 text-xs font-medium text-violet-700 transition hover:bg-violet-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-violet-800 dark:bg-violet-950/30 dark:text-violet-200 dark:hover:bg-violet-950/50"
                                            >
                                                {generatingInterpretation ? 'Generating...' : 'Local Model Interpretation'}
                                            </button>
                                            <button
                                                type="button"
                                                onClick={() => void handleEmailSummary()}
                                                disabled={emailingSummary}
                                                className="rounded-lg border border-emerald-300 bg-emerald-50 px-3 py-2 text-xs font-medium text-emerald-700 transition hover:bg-emerald-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-200 dark:hover:bg-emerald-950/50"
                                            >
                                                {emailingSummary ? 'Queueing email...' : 'Email Summary to Admins'}
                                            </button>
                                            <Link
                                                to="/admin/operations/sre-smart-bot/approvals"
                                                className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                                            >
                                                Open Approval Queue
                                            </Link>
                                            <Link
                                                to={selectedIncident ? `/admin/operations/sre-smart-bot/detector-rules?incident=${encodeURIComponent(selectedIncident.id)}` : '/admin/operations/sre-smart-bot/detector-rules'}
                                                className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                                            >
                                                Open Detector Rules
                                            </Link>
                                        </div>

                                        {agentDraft || agentInterpretation || agentSuggestedAction || agentScorecard || agentSnapshot ? (
                                            <div className="mt-4 grid gap-4 xl:grid-cols-2">
                                                <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                    <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Bot Guidance</div>
                                                    <div className="mt-3 space-y-3">
                                                        {agentSnapshot ? (
                                                            <div className="rounded-lg border border-emerald-200 bg-emerald-50/70 px-3 py-3 text-sm text-emerald-900 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-100">
                                                                <div className="text-xs font-semibold uppercase tracking-[0.16em] text-emerald-700 dark:text-emerald-300">AI Snapshot</div>
                                                                <div className="mt-2">{agentSnapshot.summary}</div>
                                                                <div className="mt-2 rounded-md border border-emerald-200 bg-white/80 px-2.5 py-2 text-xs dark:border-emerald-900/40 dark:bg-emerald-950/30">
                                                                    <div><span className="font-semibold">Probable cause:</span> {agentSnapshot.triage?.probable_cause || 'n/a'}</div>
                                                                    <div className="mt-1"><span className="font-semibold">Severity:</span> {agentSnapshot.severity?.score ?? 'n/a'} ({agentSnapshot.severity?.level || 'n/a'})</div>
                                                                    <div className="mt-1"><span className="font-semibold">Action:</span> {agentSnapshot.suggested_action?.action_key || 'n/a'}</div>
                                                                </div>
                                                            </div>
                                                        ) : null}
                                                        {agentScorecard ? (
                                                            <div className={`rounded-lg border px-3 py-3 text-sm ${agentSeverityTone(agentScorecard.severity_level)}`}>
                                                                <div className="text-xs font-semibold uppercase tracking-[0.16em]">Incident Scorecard</div>
                                                                <div className="mt-2 font-medium">Score {agentScorecard.severity_score} ({agentScorecard.severity_level})</div>
                                                                <div className="mt-1">{agentScorecard.summary}</div>
                                                                <div className="mt-2 rounded-md border border-current/25 bg-white/60 px-2.5 py-2 text-xs dark:bg-slate-950/30">
                                                                    <div><span className="font-semibold">Probable cause:</span> {agentScorecard.probable_cause}</div>
                                                                    <div className="mt-1"><span className="font-semibold">Confidence:</span> {agentScorecard.confidence}</div>
                                                                    <div className="mt-1"><span className="font-semibold">Action key:</span> {agentScorecard.action_key}</div>
                                                                    <div className="mt-1"><span className="font-semibold">Blast radius:</span> {agentScorecard.blast_radius}</div>
                                                                </div>
                                                                {(agentScorecard.why_severe_cards || []).length > 0 ? (
                                                                    <div className="mt-2 space-y-2">
                                                                        {(agentScorecard.why_severe_cards || []).map((card) => (
                                                                            <div key={card.key} className="rounded-md border border-current/25 bg-white/60 px-2.5 py-2 text-xs dark:bg-slate-950/30">
                                                                                <span className="font-semibold">{card.label}</span> (+{card.contribution}): {card.reason}
                                                                            </div>
                                                                        ))}
                                                                    </div>
                                                                ) : null}
                                                            </div>
                                                        ) : null}
                                                        {agentSuggestedAction ? (
                                                            <div className="rounded-lg border border-fuchsia-200 bg-fuchsia-50/70 px-3 py-3 text-sm text-fuchsia-900 dark:border-fuchsia-900/40 dark:bg-fuchsia-950/30 dark:text-fuchsia-100">
                                                                <div className="flex flex-wrap items-center justify-between gap-2">
                                                                    <div className="text-xs font-semibold uppercase tracking-[0.16em] text-fuchsia-700 dark:text-fuchsia-300">Advisory Suggested Action</div>
                                                                    <span className="rounded-full border border-fuchsia-300 bg-white/80 px-2 py-0.5 text-[11px] font-semibold text-fuchsia-800 dark:border-fuchsia-800 dark:bg-fuchsia-950/40 dark:text-fuchsia-200">
                                                                        advisory only
                                                                    </span>
                                                                </div>
                                                                <div className="mt-2"><span className="font-semibold">Action:</span> {agentSuggestedAction.action_key}</div>
                                                                <div className="mt-1">{agentSuggestedAction.action_summary}</div>
                                                                <div className="mt-2 rounded-md border border-fuchsia-200 bg-white/80 px-2.5 py-2 text-xs dark:border-fuchsia-900/40 dark:bg-fuchsia-950/30">
                                                                    <div><span className="font-semibold">Justification:</span> {agentSuggestedAction.justification}</div>
                                                                    <div className="mt-1"><span className="font-semibold">Blast radius:</span> {agentSuggestedAction.blast_radius}</div>
                                                                    <div className="mt-1"><span className="font-semibold">Guardrail:</span> {agentSuggestedAction.execution_guardrail}</div>
                                                                </div>
                                                            </div>
                                                        ) : null}
                                                        {agentSeverity ? (
                                                            <div className={`rounded-lg border px-3 py-3 text-sm ${agentSeverityTone(agentSeverity.level)}`}>
                                                                <div className="text-xs font-semibold uppercase tracking-[0.16em]">Why This Is Severe</div>
                                                                <div className="mt-2 font-medium">Score {agentSeverity.score} ({agentSeverity.level})</div>
                                                                <div className="mt-1">{agentSeverity.summary}</div>
                                                                {(agentSeverity.factors || []).length > 0 ? (
                                                                    <div className="mt-2 space-y-2">
                                                                        {agentSeverity.factors.map((factor) => (
                                                                            <div key={factor.key} className="rounded-md border border-current/25 bg-white/60 px-2.5 py-2 text-xs dark:bg-slate-950/30">
                                                                                <span className="font-semibold">{factor.label}</span> (+{factor.contribution}): {factor.reason}
                                                                            </div>
                                                                        ))}
                                                                    </div>
                                                                ) : null}
                                                            </div>
                                                        ) : null}
                                                        {agentTriage ? (
                                                            <div className="rounded-lg border border-cyan-200 bg-cyan-50/70 px-3 py-3 text-sm text-cyan-900 dark:border-cyan-900/40 dark:bg-cyan-950/30 dark:text-cyan-100">
                                                                <div className="text-xs font-semibold uppercase tracking-[0.16em] text-cyan-700 dark:text-cyan-300">Triage Snapshot</div>
                                                                <div className="mt-2"><span className="font-semibold">Probable cause:</span> {agentTriage.probable_cause}</div>
                                                                <div className="mt-1"><span className="font-semibold">Confidence:</span> {agentTriage.confidence}</div>
                                                                <div className="mt-2"><span className="font-semibold">Recommended action:</span> {agentTriage.recommended_action}</div>
                                                            </div>
                                                        ) : null}
                                                        {agentDraft ? (
                                                            <div className="rounded-lg border border-slate-200 bg-white px-3 py-3 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300">
                                                                <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Grounded Draft</div>
                                                                <div className="mt-2">{agentDraft.summary}</div>
                                                            </div>
                                                        ) : (
                                                            <EmptyState title="No draft generated yet" description="Generate a grounded draft to get bounded hypotheses and an investigation plan before deciding on recovery actions." />
                                                        )}
                                                        {agentInterpretation?.timeline_summary || agentInterpretation?.operator_summary ? (
                                                            <div className="rounded-lg border border-slate-200 bg-white px-3 py-3 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300">
                                                                <div className="flex flex-wrap items-center justify-between gap-2">
                                                                    <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Timeline Summary</div>
                                                                    {agentInterpretation?.cache_hit ? (
                                                                        <span className="rounded-full border border-emerald-300 bg-emerald-50 px-2 py-0.5 text-[11px] font-medium text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-200">Cache Hit</span>
                                                                    ) : null}
                                                                </div>
                                                                <div className="mt-2">{agentInterpretation.timeline_summary || agentInterpretation.operator_summary}</div>
                                                                {agentInterpretation?.fallback_reason ? (
                                                                    <div className="mt-2 rounded-md border border-amber-200 bg-amber-50 px-2.5 py-2 text-xs text-amber-800 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200">
                                                                        {agentInterpretation.fallback_reason}
                                                                    </div>
                                                                ) : null}
                                                            </div>
                                                        ) : null}
                                                        {agentInterpretation?.change_detection_15m ? (
                                                            <div className="rounded-lg border border-indigo-200 bg-indigo-50 px-3 py-3 text-sm text-indigo-900 dark:border-indigo-900/40 dark:bg-indigo-950/30 dark:text-indigo-100">
                                                                <div className="text-xs font-semibold uppercase tracking-[0.16em] text-indigo-700 dark:text-indigo-300">15m Change Detection</div>
                                                                <div className="mt-2">{agentInterpretation.change_detection_15m}</div>
                                                            </div>
                                                        ) : null}
                                                        {(agentInterpretation?.citations || []).length > 0 ? (
                                                            <div className="rounded-lg border border-slate-200 bg-white px-3 py-3 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300">
                                                                <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Grounding Citations</div>
                                                                <div className="mt-2 space-y-2">
                                                                    {(agentInterpretation?.citations || []).map((citation, index) => (
                                                                        <div key={`${citation.kind}-${citation.source}-${index}`} className="rounded-md border border-slate-200 bg-slate-50 px-2.5 py-2 text-xs dark:border-slate-700 dark:bg-slate-950/40">
                                                                            <span className="font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">{citation.kind}</span> · <span className="font-medium text-slate-800 dark:text-slate-200">{citation.source}</span>
                                                                            {citation.section ? <span> · {citation.section}</span> : null}
                                                                            <div className="mt-1 text-slate-600 dark:text-slate-300">{citation.note}</div>
                                                                        </div>
                                                                    ))}
                                                                </div>
                                                            </div>
                                                        ) : null}
                                                        {agentInterpretation?.likely_root_cause ? (
                                                            <div className="rounded-lg border border-slate-200 bg-white px-3 py-3 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300">
                                                                <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Likely Root Cause</div>
                                                                <div className="mt-2">{agentInterpretation.likely_root_cause}</div>
                                                            </div>
                                                        ) : null}
                                                    </div>
                                                </div>

                                                <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                    <div className="flex flex-wrap items-center justify-between gap-2">
                                                        <div>
                                                            <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Operator Handoff Note</div>
                                                            <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Use this as a handoff, status update, or approval context message.</div>
                                                        </div>
                                                        <button
                                                            type="button"
                                                            onClick={() => void handleCopyOperatorMessage()}
                                                            disabled={!(agentInterpretation?.operator_handoff_note || agentInterpretation?.operator_message_draft)}
                                                            className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                                                        >
                                                            Copy Message
                                                        </button>
                                                    </div>
                                                    {(agentInterpretation?.operator_handoff_note || agentInterpretation?.operator_message_draft) ? (
                                                        <div className="mt-3 rounded-lg border border-slate-200 bg-white px-3 py-3 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300">
                                                            {agentInterpretation.operator_handoff_note || agentInterpretation.operator_message_draft}
                                                        </div>
                                                    ) : (
                                                        <div className="mt-3">
                                                            <EmptyState title="No message draft yet" description="Generate a local interpretation to have the bot draft operator-facing communication for this incident." />
                                                        </div>
                                                    )}
                                                    {(agentInterpretation?.watchouts || []).length > 0 ? (
                                                        <div className="mt-3 space-y-2">
                                                            {(agentInterpretation?.watchouts || []).map((watchout) => (
                                                                <div key={watchout} className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200">
                                                                    {watchout}
                                                                </div>
                                                            ))}
                                                        </div>
                                                    ) : null}
                                                </div>
                                            </div>
                                        ) : null}
                                    </SectionCard>

                                    <SectionCard title="Actions" subtitle={`${actions.length} action attempts`}>
                                        {actions.length === 0 ? <EmptyState title="No actions attempted" description="Remediation action attempts will appear here once the policy engine starts executing or requesting actions." /> : (
                                            <div className="space-y-3">
                                                {actions.map((action) => (
                                                    <div key={action.id} className={`rounded-xl border p-4 ${actionTone(action.status)}`}>
                                                        <div className="flex flex-wrap items-center justify-between gap-3">
                                                            <div className="flex flex-wrap items-center gap-2">
                                                                <div className="font-medium">{action.action_key}</div>
                                                                <span className="rounded-full border border-current/20 px-2 py-0.5 text-[11px] font-semibold">{action.status}</span>
                                                                {action.approval_required ? (
                                                                    <span className="rounded-full border border-current/20 px-2 py-0.5 text-[11px] font-semibold">approval required</span>
                                                                ) : null}
                                                            </div>
                                                            <div className="text-xs opacity-80">{formatDateTime(action.requested_at)}</div>
                                                        </div>
                                                        <div className="mt-3 grid gap-3 text-sm md:grid-cols-2">
                                                            <div>
                                                                <div className="text-xs uppercase tracking-wide opacity-70">Target</div>
                                                                <div className="mt-1 font-medium">{action.target_kind}: {action.target_ref || '—'}</div>
                                                            </div>
                                                            <div>
                                                                <div className="text-xs uppercase tracking-wide opacity-70">Actor</div>
                                                                <div className="mt-1 font-medium">{action.actor_type}{action.actor_id ? ` • ${action.actor_id}` : ''}</div>
                                                            </div>
                                                        </div>
                                                        {action.result_payload ? (
                                                            <div className="mt-3 rounded-lg border border-current/15 bg-white/50 p-3 text-xs text-slate-700 dark:bg-slate-950/40 dark:text-slate-200">
                                                                <div className="mb-2 uppercase tracking-wide opacity-70">Stored rationale</div>
                                                                <pre className="overflow-x-auto whitespace-pre-wrap">{prettyJson(action.result_payload)}</pre>
                                                            </div>
                                                        ) : null}
                                                        <div className="mt-3 flex flex-wrap gap-2">
                                                            {!action.approval_required && (action.status || '').toLowerCase() === 'proposed' ? (
                                                                <button
                                                                    type="button"
                                                                    onClick={() => handleStartApprovalRequest(action)}
                                                                    disabled={mutatingActionId === action.id}
                                                                    className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                                                                >
                                                                    {approvalRequestActionId === action.id ? 'Approval Request Open' : 'Request Approval'}
                                                                </button>
                                                            ) : null}
                                                            {canExecuteAction(action) ? (
                                                                <button
                                                                    type="button"
                                                                    onClick={() => void handleExecuteAction(action)}
                                                                    disabled={mutatingActionId === action.id}
                                                                    className="rounded-lg bg-sky-600 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-sky-500 dark:hover:bg-sky-400"
                                                                >
                                                                    {mutatingActionId === action.id ? 'Executing...' : 'Execute'}
                                                                </button>
                                                            ) : null}
                                                        </div>
                                                        {approvalRequestActionId === action.id ? (
                                                            <div className="mt-4 rounded-xl border border-slate-200 bg-white/70 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                                                <div className="flex flex-wrap items-center justify-between gap-2">
                                                                    <div>
                                                                        <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Approval Request Composer</div>
                                                                        <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Choose where the request goes and tailor the message operators will see.</div>
                                                                    </div>
                                                                    <button
                                                                        type="button"
                                                                        onClick={() => handleCancelApprovalRequest()}
                                                                        className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:border-slate-400 hover:text-slate-900 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-slate-600 dark:hover:text-white"
                                                                    >
                                                                        Cancel
                                                                    </button>
                                                                </div>

                                                                <div className="mt-4 grid gap-4 md:grid-cols-2">
                                                                    <div>
                                                                        <label className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Channel Provider</label>
                                                                        <select
                                                                            value={approvalRequestChannelProviderId}
                                                                            onChange={(event) => setApprovalRequestChannelProviderId(event.target.value)}
                                                                            className={`${inputClass} mt-2`}
                                                                        >
                                                                            <option value="">Use default in-app routing</option>
                                                                            {approvalCapableChannelProviders.map((provider) => (
                                                                                <option key={provider.id} value={provider.id}>
                                                                                    {provider.name} ({provider.kind})
                                                                                </option>
                                                                            ))}
                                                                        </select>
                                                                        <div className="mt-2 text-xs text-slate-500 dark:text-slate-400">
                                                                            {approvalCapableChannelProviders.length > 0
                                                                                ? 'Only enabled approval-capable providers are listed here.'
                                                                                : 'No explicit approval-capable provider is configured, so the default in-app path will be used.'}
                                                                        </div>
                                                                    </div>
                                                                    <div>
                                                                        <label className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Target Action</label>
                                                                        <div className="mt-2 rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-950 dark:text-slate-300">
                                                                            {action.action_key} on {action.target_kind}: {action.target_ref || '—'}
                                                                        </div>
                                                                    </div>
                                                                </div>

                                                                <div className="mt-4">
                                                                    <label className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Request Message</label>
                                                                    <textarea
                                                                        value={approvalRequestMessage}
                                                                        onChange={(event) => setApprovalRequestMessage(event.target.value)}
                                                                        rows={4}
                                                                        placeholder="Explain the action, why approval is needed, and any operational risk or rollback context."
                                                                        className={`${inputClass} mt-2`}
                                                                    />
                                                                </div>

                                                                <div className="mt-4 flex flex-wrap gap-2">
                                                                    <button
                                                                        type="button"
                                                                        onClick={() => void handleSubmitApprovalRequest(action)}
                                                                        disabled={mutatingActionId === action.id || !approvalRequestMessage.trim()}
                                                                        className="rounded-lg bg-sky-600 px-3 py-2 text-xs font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-sky-500 dark:hover:bg-sky-400"
                                                                    >
                                                                        {mutatingActionId === action.id ? 'Requesting...' : 'Send Approval Request'}
                                                                    </button>
                                                                </div>
                                                            </div>
                                                        ) : null}
                                                        {action.error_message ? <div className="mt-3 text-sm font-medium text-rose-700 dark:text-rose-300">{action.error_message}</div> : null}
                                                    </div>
                                                ))}
                                            </div>
                                        )}
                                    </SectionCard>

                                    <SectionCard title="Approvals" subtitle={`${approvals.length} approval records`}>
                                        {approvals.length === 0 ? <EmptyState title="No approvals recorded" description="Approval requests will appear here once SRE Smart Bot starts issuing channel or in-app approvals." /> : (
                                            <div className="space-y-3">
                                                {approvals.map((approval) => (
                                                    <div key={approval.id} className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                        <div className="flex flex-wrap items-center justify-between gap-3">
                                                            <div className="font-medium text-slate-900 dark:text-white">
                                                                {approval.action_attempt_id && actionsById[approval.action_attempt_id]
                                                                    ? actionsById[approval.action_attempt_id].action_key
                                                                    : approval.channel_provider_id}
                                                            </div>
                                                            <span className={`rounded-full border px-2.5 py-1 text-[11px] font-semibold ${approval.decided_at ? 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-200' : 'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200'}`}>{approval.status}</span>
                                                        </div>
                                                        <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-slate-500">
                                                            <span className="rounded-full border border-slate-300 px-2 py-0.5 dark:border-slate-700">{approval.channel_provider_id}</span>
                                                            {approval.action_attempt_id ? <span className="rounded-full border border-slate-300 px-2 py-0.5 dark:border-slate-700">linked action</span> : null}
                                                        </div>
                                                        <p className="mt-2 text-sm text-slate-700 dark:text-slate-300">{approval.request_message}</p>
                                                        <div className="mt-2 text-xs text-slate-500 dark:text-slate-500">
                                                            Requested {formatDateTime(approval.requested_at)}
                                                            {approval.decided_at ? ` • decided ${formatDateTime(approval.decided_at)}` : ''}
                                                            {approval.expires_at ? ` • expires ${formatDateTime(approval.expires_at)}` : ''}
                                                        </div>
                                                        {!approval.decided_at ? (
                                                            <div className="mt-3 space-y-3">
                                                                <div>
                                                                    <label className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">
                                                                        Decision Comment
                                                                    </label>
                                                                    <textarea
                                                                        value={approvalComments[approval.id] || ''}
                                                                        onChange={(event) => setApprovalComments((current) => ({ ...current, [approval.id]: event.target.value }))}
                                                                        rows={3}
                                                                        placeholder="Optional context for why you approved or rejected this request."
                                                                        className="mt-2 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-700 outline-none transition focus:border-sky-400 focus:ring-2 focus:ring-sky-200 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:focus:border-sky-500 dark:focus:ring-sky-900/40"
                                                                    />
                                                                </div>
                                                                <div className="flex flex-wrap gap-2">
                                                                    <button
                                                                        type="button"
                                                                        onClick={() => void handleApprovalDecision(approval, 'approved')}
                                                                        disabled={mutatingApprovalId === approval.id}
                                                                        className="rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-emerald-500 dark:hover:bg-emerald-400"
                                                                    >
                                                                        {mutatingApprovalId === approval.id ? 'Updating...' : 'Approve'}
                                                                    </button>
                                                                    <button
                                                                        type="button"
                                                                        onClick={() => void handleApprovalDecision(approval, 'rejected')}
                                                                        disabled={mutatingApprovalId === approval.id}
                                                                        className="rounded-lg border border-rose-300 bg-white px-3 py-1.5 text-xs font-medium text-rose-700 transition hover:bg-rose-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-rose-800 dark:bg-slate-950 dark:text-rose-300 dark:hover:bg-rose-950/40"
                                                                    >
                                                                        {mutatingApprovalId === approval.id ? 'Updating...' : 'Reject'}
                                                                    </button>
                                                                </div>
                                                            </div>
                                                        ) : null}
                                                        {approval.decision_comment ? <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">Comment: {approval.decision_comment}</div> : null}
                                                    </div>
                                                ))}
                                            </div>
                                        )}
                                    </SectionCard>
                                </div>
                            </>
                        ) : null}

                        {drawerTab === 'ai' ? (
                            <>
                                <div role="tabpanel" id="incident-panel-ai" aria-labelledby="incident-tab-ai" className="space-y-6">
                                    <div className="rounded-2xl border border-slate-200 bg-slate-50/90 px-4 py-3 text-sm text-slate-600 dark:border-slate-800 dark:bg-slate-950/40 dark:text-slate-300">
                                        {drawerTabs.find((tab) => tab.id === 'ai')?.hint}
                                    </div>
                                    <SectionCard title="AI Workspace" subtitle="Structured context for the future MCP and agent-runtime layer, built from the incident ledger and current SRE policy.">
                                        {!workspace ? (
                                            <EmptyState title="Workspace not available yet" description="The MCP and AI workspace bundle has not loaded for this incident yet." />
                                        ) : (
                                            <div className="space-y-5">
                                                <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                                                    <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                        <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Agent Runtime</div>
                                                        <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">
                                                            {workspace.agent_runtime.enabled ? 'Enabled' : 'Disabled'}
                                                        </div>
                                                        <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">
                                                            {workspace.agent_runtime.provider || 'custom'}
                                                            {workspace.agent_runtime.model ? ` • ${workspace.agent_runtime.model}` : ''}
                                                        </div>
                                                    </div>
                                                    <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                        <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Enabled MCP Servers</div>
                                                        <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{workspace.enabled_mcp_servers.length}</div>
                                                        <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Bounded tool boundaries available under current policy.</div>
                                                    </div>
                                                    <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-950">
                                                        <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-500">Recommended Questions</div>
                                                        <div className="mt-2 text-2xl font-semibold text-slate-900 dark:text-white">{workspace.recommended_questions.length}</div>
                                                        <div className="mt-1 text-sm text-slate-600 dark:text-slate-400">Prompt-ready investigation angles grounded in stored evidence.</div>
                                                    </div>
                                                </div>

                                                <div className="grid gap-5 xl:grid-cols-2">
                                                    <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                                        <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Default Tool Bundle</h4>
                                                        {!workspace.default_tool_bundle?.length ? (
                                                            <div className="mt-3">
                                                                <EmptyState title="No default tool bundle yet" description="The workspace did not project a default bounded MCP bundle for this incident." />
                                                            </div>
                                                        ) : (
                                                            <div className="mt-3 flex flex-wrap gap-2">
                                                                {workspace.default_tool_bundle.map((toolName) => (
                                                                    <span key={toolName} className="inline-flex rounded-full border border-slate-300 bg-white/90 px-3 py-1 text-xs font-medium text-slate-700 dark:border-slate-700 dark:bg-slate-900/80 dark:text-slate-300">
                                                                        {toolName}
                                                                    </span>
                                                                ))}
                                                            </div>
                                                        )}
                                                    </div>

                                                    <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                                        <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Async Backlog Context</h4>
                                                        <div className="mt-3">
                                                            {asyncBacklogInsight ? (
                                                                <AsyncBacklogInsightContent insight={asyncBacklogInsight} />
                                                            ) : (
                                                                <EmptyState title="No backlog summary projected" description="No normalized async backlog snapshot is attached to this workspace yet." />
                                                            )}
                                                        </div>
                                                    </div>
                                                </div>

                                                <div className="grid gap-5 xl:grid-cols-2">
                                                    <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                                        <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Messaging Transport Correlation</h4>
                                                        <div className="mt-3">
                                                            {messagingTransportInsight ? (
                                                                <MessagingTransportInsightContent insight={messagingTransportInsight} />
                                                            ) : (
                                                                <EmptyState title="No transport summary projected" description="No normalized messaging transport snapshot is attached to this workspace yet." />
                                                            )}
                                                        </div>
                                                    </div>

                                                    <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                                        <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Messaging Consumer Context</h4>
                                                        <div className="mt-3">
                                                            {messagingConsumerInsight ? (
                                                                <MessagingConsumerInsightContent insight={messagingConsumerInsight} />
                                                            ) : (
                                                                <EmptyState title="No consumer summary projected" description="No normalized messaging consumer snapshot is attached to this workspace yet." />
                                                            )}
                                                        </div>
                                                    </div>

                                                    <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                                        <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">AI Executive Summary</h4>
                                                        <div className="mt-3 space-y-2">
                                                            {workspace.executive_summary.map((line, index) => (
                                                                <div key={`${line}-${index}`} className="rounded-xl border border-slate-200 bg-white/90 px-3 py-2 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900/70 dark:text-slate-300">
                                                                    {line}
                                                                </div>
                                                            ))}
                                                        </div>
                                                    </div>

                                                    <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                                        <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Recommended Questions</h4>
                                                        <div className="mt-3 space-y-2">
                                                            {workspace.recommended_questions.map((line, index) => (
                                                                <div key={`${line}-${index}`} className="rounded-xl border border-slate-200 bg-white/90 px-3 py-2 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900/70 dark:text-slate-300">
                                                                    {line}
                                                                </div>
                                                            ))}
                                                        </div>
                                                    </div>
                                                </div>

                                                <div className="grid gap-5 xl:grid-cols-2">
                                                    <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                                        <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Suggested Tooling</h4>
                                                        {workspace.suggested_tooling.length === 0 ? (
                                                            <EmptyState title="No tooling guidance yet" description="Enable MCP servers in settings to surface bounded tooling guidance here." />
                                                        ) : (
                                                            <div className="mt-3 space-y-2">
                                                                {workspace.suggested_tooling.map((line, index) => (
                                                                    <div key={`${line}-${index}`} className="rounded-xl border border-slate-200 bg-white/90 px-3 py-2 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900/70 dark:text-slate-300">
                                                                        {line}
                                                                    </div>
                                                                ))}
                                                            </div>
                                                        )}
                                                    </div>

                                                    <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                                        <div className="flex flex-wrap items-center justify-between gap-3">
                                                            <div>
                                                                <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Grounded Draft</h4>
                                                                <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">Generate deterministic hypotheses and an investigation plan from stored evidence and bounded tools.</p>
                                                            </div>
                                                            <div className="flex flex-wrap gap-2">
                                                                <button
                                                                    type="button"
                                                                    onClick={() => void handleGenerateAgentSnapshot()}
                                                                    disabled={generatingAgentSnapshot}
                                                                    className="rounded-lg border border-emerald-300 bg-emerald-50 px-3 py-2 text-xs font-medium text-emerald-800 transition hover:bg-emerald-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-200 dark:hover:bg-emerald-950/50"
                                                                >
                                                                    {generatingAgentSnapshot ? 'Generating...' : 'Generate AI Snapshot'}
                                                                </button>
                                                                <button
                                                                    type="button"
                                                                    onClick={() => void handleGenerateAgentSeverity()}
                                                                    disabled={generatingAgentSeverity}
                                                                    className="rounded-lg border border-amber-300 bg-amber-50 px-3 py-2 text-xs font-medium text-amber-900 transition hover:bg-amber-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-200 dark:hover:bg-amber-950/50"
                                                                >
                                                                    {generatingAgentSeverity ? 'Generating...' : 'Generate Severity'}
                                                                </button>
                                                                <button
                                                                    type="button"
                                                                    onClick={() => void handleGenerateAgentScorecard()}
                                                                    disabled={generatingAgentScorecard}
                                                                    className="rounded-lg border border-rose-300 bg-rose-50 px-3 py-2 text-xs font-medium text-rose-800 transition hover:bg-rose-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-rose-800 dark:bg-rose-950/30 dark:text-rose-200 dark:hover:bg-rose-950/50"
                                                                >
                                                                    {generatingAgentScorecard ? 'Generating...' : 'Generate Scorecard'}
                                                                </button>
                                                                <button
                                                                    type="button"
                                                                    onClick={() => void handleGenerateAgentTriage()}
                                                                    disabled={generatingAgentTriage}
                                                                    className="rounded-lg border border-cyan-300 bg-cyan-50 px-3 py-2 text-xs font-medium text-cyan-800 transition hover:bg-cyan-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-cyan-800 dark:bg-cyan-950/30 dark:text-cyan-200 dark:hover:bg-cyan-950/50"
                                                                >
                                                                    {generatingAgentTriage ? 'Generating...' : 'Generate Triage'}
                                                                </button>
                                                                <button
                                                                    type="button"
                                                                    onClick={() => void handleGenerateAgentSuggestedAction()}
                                                                    disabled={generatingAgentSuggestedAction}
                                                                    className="rounded-lg border border-fuchsia-300 bg-fuchsia-50 px-3 py-2 text-xs font-medium text-fuchsia-800 transition hover:bg-fuchsia-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-fuchsia-800 dark:bg-fuchsia-950/30 dark:text-fuchsia-200 dark:hover:bg-fuchsia-950/50"
                                                                >
                                                                    {generatingAgentSuggestedAction ? 'Generating...' : 'Generate Suggested Action'}
                                                                </button>
                                                                <button
                                                                    type="button"
                                                                    onClick={() => void handleGenerateAgentDraft()}
                                                                    disabled={generatingAgentDraft}
                                                                    className="rounded-lg border border-sky-300 bg-sky-50 px-3 py-2 text-xs font-medium text-sky-700 transition hover:bg-sky-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-sky-800 dark:bg-sky-950/30 dark:text-sky-200 dark:hover:bg-sky-950/50"
                                                                >
                                                                    {generatingAgentDraft ? 'Generating...' : 'Generate Draft'}
                                                                </button>
                                                                <button
                                                                    type="button"
                                                                    onClick={() => void handleGenerateInterpretation()}
                                                                    disabled={generatingInterpretation}
                                                                    className="rounded-lg border border-violet-300 bg-violet-50 px-3 py-2 text-xs font-medium text-violet-700 transition hover:bg-violet-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-violet-800 dark:bg-violet-950/30 dark:text-violet-200 dark:hover:bg-violet-950/50"
                                                                >
                                                                    {generatingInterpretation ? 'Generating...' : 'Local Model Interpretation'}
                                                                </button>
                                                            </div>
                                                        </div>
                                                        <div className="mt-4 space-y-4">
                                                            {agentSnapshot ? (
                                                                <div className="rounded-xl border border-emerald-200 bg-emerald-50/70 px-4 py-3 dark:border-emerald-900/40 dark:bg-emerald-950/30">
                                                                    <div className="text-xs font-semibold uppercase tracking-[0.16em] text-emerald-700 dark:text-emerald-300">AI Snapshot</div>
                                                                    <div className="mt-2 text-sm text-emerald-900 dark:text-emerald-100">{agentSnapshot.summary}</div>
                                                                    <div className="mt-3 rounded-lg border border-emerald-200 bg-white/80 px-3 py-2 text-sm text-emerald-900 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-100">
                                                                        <div><span className="font-semibold">Probable cause:</span> {agentSnapshot.triage?.probable_cause || 'n/a'}</div>
                                                                        <div className="mt-1"><span className="font-semibold">Severity:</span> {agentSnapshot.severity?.score ?? 'n/a'} ({agentSnapshot.severity?.level || 'n/a'})</div>
                                                                        <div className="mt-1"><span className="font-semibold">Action:</span> {agentSnapshot.suggested_action?.action_key || 'n/a'}</div>
                                                                    </div>
                                                                </div>
                                                            ) : (
                                                                <EmptyState title="No AI snapshot yet" description="Generate AI Snapshot to build triage, severity, scorecard, and suggested action in one deterministic call." />
                                                            )}
                                                            {agentScorecard ? (
                                                                <div className={`rounded-xl border px-4 py-3 ${agentSeverityTone(agentScorecard.severity_level)}`}>
                                                                    <div className="text-xs font-semibold uppercase tracking-[0.16em]">Incident Scorecard</div>
                                                                    <div className="mt-2 text-sm font-medium">Score {agentScorecard.severity_score} ({agentScorecard.severity_level})</div>
                                                                    <div className="mt-1 text-sm">{agentScorecard.summary}</div>
                                                                    <div className="mt-3 rounded-lg border border-current/25 bg-white/70 px-3 py-2 text-sm dark:bg-slate-950/30">
                                                                        <div><span className="font-semibold">Probable cause:</span> {agentScorecard.probable_cause}</div>
                                                                        <div className="mt-1"><span className="font-semibold">Confidence:</span> {agentScorecard.confidence}</div>
                                                                        <div className="mt-1"><span className="font-semibold">Action key:</span> {agentScorecard.action_key}</div>
                                                                        <div className="mt-1"><span className="font-semibold">Blast radius:</span> {agentScorecard.blast_radius}</div>
                                                                    </div>
                                                                    {(agentScorecard.why_severe_cards || []).length > 0 ? (
                                                                        <div className="mt-3 grid gap-2">
                                                                            {(agentScorecard.why_severe_cards || []).map((card) => (
                                                                                <div key={card.key} className="rounded-lg border border-current/25 bg-white/70 px-3 py-2 text-xs dark:bg-slate-950/30">
                                                                                    <span className="font-semibold">{card.label}</span> (+{card.contribution}): {card.reason}
                                                                                </div>
                                                                            ))}
                                                                        </div>
                                                                    ) : null}
                                                                </div>
                                                            ) : (
                                                                <EmptyState title="No incident scorecard yet" description="Generate Scorecard for a compact incident view: probable cause, confidence, severity score, top why-severe cards, and approval-safe action guidance." />
                                                            )}
                                                            {agentSuggestedAction ? (
                                                                <div className="rounded-xl border border-fuchsia-200 bg-fuchsia-50/70 px-4 py-3 dark:border-fuchsia-900/40 dark:bg-fuchsia-950/30">
                                                                    <div className="flex flex-wrap items-center justify-between gap-2">
                                                                        <div className="text-xs font-semibold uppercase tracking-[0.16em] text-fuchsia-700 dark:text-fuchsia-300">Advisory Suggested Action</div>
                                                                        <span className="rounded-full border border-fuchsia-300 bg-white/80 px-2 py-0.5 text-[11px] font-semibold text-fuchsia-800 dark:border-fuchsia-800 dark:bg-fuchsia-950/40 dark:text-fuchsia-200">
                                                                            advisory only
                                                                        </span>
                                                                    </div>
                                                                    <div className="mt-2 text-sm text-fuchsia-900 dark:text-fuchsia-100"><span className="font-semibold">Action:</span> {agentSuggestedAction.action_key}</div>
                                                                    <div className="mt-1 text-sm text-fuchsia-900 dark:text-fuchsia-100">{agentSuggestedAction.action_summary}</div>
                                                                    <div className="mt-2 rounded-lg border border-fuchsia-200 bg-white/80 px-3 py-2 text-sm text-fuchsia-900 dark:border-fuchsia-900/40 dark:bg-fuchsia-950/30 dark:text-fuchsia-100">
                                                                        <div><span className="font-semibold">Justification:</span> {agentSuggestedAction.justification}</div>
                                                                        <div className="mt-1"><span className="font-semibold">Blast radius:</span> {agentSuggestedAction.blast_radius}</div>
                                                                        <div className="mt-1"><span className="font-semibold">Execution guardrail:</span> {agentSuggestedAction.execution_guardrail}</div>
                                                                    </div>
                                                                </div>
                                                            ) : (
                                                                <EmptyState title="No advisory suggested action yet" description="Generate Suggested Action to get action + justification + blast radius guidance. This is advisory-only and does not execute anything." />
                                                            )}
                                                            {agentSeverity ? (
                                                                <div className={`rounded-xl border px-4 py-3 ${agentSeverityTone(agentSeverity.level)}`}>
                                                                    <div className="text-xs font-semibold uppercase tracking-[0.16em]">Why This Is Severe</div>
                                                                    <div className="mt-2 text-sm font-medium">Score {agentSeverity.score} ({agentSeverity.level})</div>
                                                                    <div className="mt-1 text-sm">{agentSeverity.summary}</div>
                                                                    {(agentSeverity.factors || []).length > 0 ? (
                                                                        <div className="mt-3 grid gap-2">
                                                                            {agentSeverity.factors.map((factor) => (
                                                                                <div key={factor.key} className="rounded-lg border border-current/25 bg-white/70 px-3 py-2 text-xs dark:bg-slate-950/30">
                                                                                    <span className="font-semibold">{factor.label}</span> (+{factor.contribution}): {factor.reason}
                                                                                </div>
                                                                            ))}
                                                                        </div>
                                                                    ) : null}
                                                                </div>
                                                            ) : (
                                                                <EmptyState title="No severity correlation yet" description="Generate Severity to correlate logs, HTTP signals, async backlog, and transport pressure into one operator score." />
                                                            )}
                                                            {agentTriage ? (
                                                                <div className="rounded-xl border border-cyan-200 bg-cyan-50/70 px-4 py-3 dark:border-cyan-900/40 dark:bg-cyan-950/30">
                                                                    <div className="text-xs font-semibold uppercase tracking-[0.16em] text-cyan-700 dark:text-cyan-300">Triage Snapshot</div>
                                                                    <div className="mt-2 text-sm text-cyan-900 dark:text-cyan-100">{agentTriage.summary}</div>
                                                                    <div className="mt-3 rounded-lg border border-cyan-200 bg-white/80 px-3 py-3 text-sm text-cyan-900 dark:border-cyan-900/40 dark:bg-cyan-950/30 dark:text-cyan-100">
                                                                        <div><span className="font-semibold">Probable cause:</span> {agentTriage.probable_cause}</div>
                                                                        <div className="mt-1"><span className="font-semibold">Confidence:</span> {agentTriage.confidence}</div>
                                                                        <div className="mt-2"><span className="font-semibold">Recommended action:</span> {agentTriage.recommended_action}</div>
                                                                    </div>
                                                                    <div className="mt-3">
                                                                        <div className="text-xs font-semibold uppercase tracking-[0.16em] text-cyan-700 dark:text-cyan-300">Next 3 Checks</div>
                                                                        <div className="mt-2 space-y-2">
                                                                            {agentTriage.next_checks.map((check, index) => (
                                                                                <div key={`${check}-${index}`} className="rounded-lg border border-cyan-200 bg-white/80 px-3 py-2 text-sm text-cyan-900 dark:border-cyan-900/40 dark:bg-cyan-950/30 dark:text-cyan-100">
                                                                                    {index + 1}. {check}
                                                                                </div>
                                                                            ))}
                                                                        </div>
                                                                    </div>
                                                                </div>
                                                            ) : (
                                                                <EmptyState title="No triage snapshot yet" description="Generate Triage to get a quick probable cause, confidence, and next-3-checks view before deeper investigation." />
                                                            )}
                                                            {agentDraft ? (
                                                                <>
                                                                    <div className="rounded-xl border border-slate-200 bg-white/90 px-4 py-3 dark:border-slate-800 dark:bg-slate-900/70">
                                                                        <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Draft Summary</div>
                                                                        <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">{agentDraft.summary}</div>
                                                                    </div>
                                                                    <div className="space-y-3">
                                                                        {agentDraft.hypotheses.map((hypothesis, index) => (
                                                                            <div key={`${hypothesis.title}-${index}`} className="rounded-xl border border-slate-200 bg-white/90 px-4 py-3 dark:border-slate-800 dark:bg-slate-900/70">
                                                                                <div className="flex flex-wrap items-center justify-between gap-2">
                                                                                    <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">{hypothesis.title}</div>
                                                                                    <span className="rounded-full border border-slate-300 px-2 py-0.5 text-[11px] font-medium text-slate-700 dark:border-slate-700 dark:text-slate-300">Rank #{index + 1}</span>
                                                                                </div>
                                                                                <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">{hypothesis.rationale}</div>
                                                                                {hypothesis.evidence_refs?.length ? (
                                                                                    <div className="mt-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-3 dark:border-slate-800 dark:bg-slate-950/40">
                                                                                        <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Supporting Tool Output</div>
                                                                                        {renderAgentEvidenceRefs(hypothesis.evidence_refs, `${hypothesis.title}-evidence`)}
                                                                                    </div>
                                                                                ) : null}
                                                                            </div>
                                                                        ))}
                                                                    </div>
                                                                    <div className="space-y-3">
                                                                        {agentDraft.investigation_plan.map((step, index) => (
                                                                            <div key={`${step.title}-${index}`} className="rounded-xl border border-slate-200 bg-white/90 px-4 py-3 dark:border-slate-800 dark:bg-slate-900/70">
                                                                                <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">{index + 1}. {step.title}</div>
                                                                                <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">{step.description}</div>
                                                                                {step.evidence_refs?.length ? (
                                                                                    <div className="mt-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-3 dark:border-slate-800 dark:bg-slate-950/40">
                                                                                        <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Evidence For This Step</div>
                                                                                        {renderAgentEvidenceRefs(step.evidence_refs, `${step.title}-evidence`)}
                                                                                    </div>
                                                                                ) : null}
                                                                            </div>
                                                                        ))}
                                                                    </div>
                                                                    {agentDraft.tool_runs.length ? (
                                                                        <div className="rounded-xl border border-slate-200 bg-slate-950 px-4 py-3 dark:border-slate-700">
                                                                            <div className="mb-2 text-xs uppercase tracking-[0.16em] text-slate-400">Tool Evidence Used</div>
                                                                            <pre className="overflow-x-auto whitespace-pre-wrap break-words text-xs text-slate-100">{prettyJson(agentDraft.tool_runs)}</pre>
                                                                        </div>
                                                                    ) : null}
                                                                </>
                                                            ) : (
                                                                <EmptyState title="No draft generated yet" description="Use Generate Draft to create a grounded incident hypothesis and investigation plan." />
                                                            )}

                                                            {agentInterpretation ? (
                                                                <div className="space-y-3">
                                                                    <div className="rounded-xl border border-slate-200 bg-white/90 px-4 py-3 dark:border-slate-800 dark:bg-slate-900/70">
                                                                        <div className="flex flex-wrap items-center justify-between gap-2">
                                                                            <div className="text-sm font-semibold text-slate-900 dark:text-slate-100">Local Model Interpretation</div>
                                                                            <div className="flex flex-wrap items-center gap-2">
                                                                                <span className="rounded-full border border-slate-300 px-2 py-0.5 text-[11px] font-medium text-slate-700 dark:border-slate-700 dark:text-slate-300">
                                                                                    {agentInterpretation.generated ? 'Generated' : 'Fallback'}
                                                                                </span>
                                                                                {agentInterpretation.cache_hit ? (
                                                                                    <span className="rounded-full border border-emerald-300 bg-emerald-50 px-2 py-0.5 text-[11px] font-medium text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-200">
                                                                                        Cache Hit
                                                                                    </span>
                                                                                ) : null}
                                                                            </div>
                                                                        </div>
                                                                        <div className="mt-2 text-xs uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Timeline Summary</div>
                                                                        <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">{agentInterpretation.timeline_summary || agentInterpretation.operator_summary || 'No model summary returned.'}</div>
                                                                        {agentInterpretation.summary_mode ? (
                                                                            <div className="mt-2 text-xs text-slate-500 dark:text-slate-400">
                                                                                Mode: {agentInterpretation.summary_mode}{agentInterpretation.evidence_hash ? ` • Evidence hash ${agentInterpretation.evidence_hash.slice(0, 12)}` : ''}
                                                                            </div>
                                                                        ) : null}
                                                                        {agentInterpretation.fallback_reason ? (
                                                                            <div className="mt-2 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200">
                                                                                {agentInterpretation.fallback_reason}
                                                                            </div>
                                                                        ) : null}
                                                                    </div>
                                                                    {agentInterpretation.change_detection_15m ? (
                                                                        <div className="rounded-xl border border-indigo-200 bg-indigo-50/70 px-4 py-3 dark:border-indigo-900/40 dark:bg-indigo-950/30">
                                                                            <div className="text-xs font-semibold uppercase tracking-[0.16em] text-indigo-700 dark:text-indigo-300">15m Change Detection</div>
                                                                            <div className="mt-2 text-sm text-indigo-900 dark:text-indigo-100">{agentInterpretation.change_detection_15m}</div>
                                                                        </div>
                                                                    ) : null}
                                                                    {agentInterpretation.likely_root_cause ? (
                                                                        <div className="rounded-xl border border-slate-200 bg-white/90 px-4 py-3 dark:border-slate-800 dark:bg-slate-900/70">
                                                                            <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Likely Root Cause</div>
                                                                            <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">{agentInterpretation.likely_root_cause}</div>
                                                                        </div>
                                                                    ) : null}
                                                                    {(agentInterpretation.watchouts || []).length > 0 ? (
                                                                        <div className="rounded-xl border border-slate-200 bg-white/90 px-4 py-3 dark:border-slate-800 dark:bg-slate-900/70">
                                                                            <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Watchouts</div>
                                                                            <div className="mt-2 space-y-2">
                                                                                {(agentInterpretation.watchouts || []).map((watchout) => (
                                                                                    <div key={watchout} className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700 dark:border-slate-700 dark:bg-slate-950/40 dark:text-slate-300">
                                                                                        {watchout}
                                                                                    </div>
                                                                                ))}
                                                                            </div>
                                                                        </div>
                                                                    ) : null}
                                                                    {(agentInterpretation.citations || []).length > 0 ? (
                                                                        <div className="rounded-xl border border-slate-200 bg-white/90 px-4 py-3 dark:border-slate-800 dark:bg-slate-900/70">
                                                                            <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Grounding Citations</div>
                                                                            <div className="mt-2 space-y-2">
                                                                                {(agentInterpretation.citations || []).map((citation, index) => (
                                                                                    <div key={`${citation.kind}-${citation.source}-${index}`} className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700 dark:border-slate-700 dark:bg-slate-950/40 dark:text-slate-300">
                                                                                        <div className="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">{citation.kind}</div>
                                                                                        <div className="mt-1 font-medium text-slate-900 dark:text-slate-100">
                                                                                            {citation.source}{citation.section ? ` • ${citation.section}` : ''}
                                                                                        </div>
                                                                                        <div className="mt-1">{citation.note}</div>
                                                                                    </div>
                                                                                ))}
                                                                            </div>
                                                                        </div>
                                                                    ) : null}
                                                                    {(agentInterpretation.operator_handoff_note || agentInterpretation.operator_message_draft) ? (
                                                                        <div className="rounded-xl border border-slate-200 bg-white/90 px-4 py-3 dark:border-slate-800 dark:bg-slate-900/70">
                                                                            <div className="flex flex-wrap items-center justify-between gap-2">
                                                                                <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">Operator Handoff Note</div>
                                                                                <button
                                                                                    type="button"
                                                                                    onClick={() => void handleCopyOperatorMessage()}
                                                                                    className="rounded-lg border border-slate-300 bg-white px-2.5 py-1 text-[11px] font-medium text-slate-700 transition hover:border-sky-400 hover:text-sky-700 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-200 dark:hover:border-sky-500 dark:hover:text-sky-300"
                                                                                >
                                                                                    Copy
                                                                                </button>
                                                                            </div>
                                                                            <div className="mt-2 text-sm text-slate-700 dark:text-slate-300">{agentInterpretation.operator_handoff_note || agentInterpretation.operator_message_draft}</div>
                                                                        </div>
                                                                    ) : null}
                                                                    {agentInterpretation.raw_response ? (
                                                                        <div className="rounded-xl border border-slate-200 bg-slate-950 px-4 py-3 dark:border-slate-700">
                                                                            <div className="mb-2 text-xs uppercase tracking-[0.16em] text-slate-400">Raw Model Response</div>
                                                                            <pre className="overflow-x-auto whitespace-pre-wrap break-words text-xs text-slate-100">{agentInterpretation.raw_response}</pre>
                                                                        </div>
                                                                    ) : null}
                                                                </div>
                                                            ) : null}
                                                        </div>
                                                    </div>
                                                </div>

                                                <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                                                    <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Read-Only MCP Tools</h4>
                                                    {mcpTools.length === 0 ? (
                                                        <div className="mt-3">
                                                            <EmptyState title="No MCP tools exposed" description="Enable an MCP server with supported read-only tools to run bounded investigation calls here." />
                                                        </div>
                                                    ) : (
                                                        <div className="mt-3 space-y-3">
                                                            {mcpTools.map((tool) => {
                                                                const toolKey = `${tool.server_id}:${tool.tool_name}`
                                                                const result = mcpResults[toolKey]
                                                                const isRunning = runningMCPToolKey === toolKey
                                                                return (
                                                                    <div key={toolKey} className="rounded-xl border border-slate-200 bg-white/90 px-4 py-4 dark:border-slate-800 dark:bg-slate-900/70">
                                                                        <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                                                                            <div className="min-w-0 flex-1">
                                                                                <div className="flex flex-wrap items-center gap-2">
                                                                                    <div className="text-sm font-medium text-slate-900 dark:text-slate-100">{tool.display_name}</div>
                                                                                    <span className="inline-flex rounded-full border border-slate-300 bg-slate-50 px-2.5 py-1 text-[11px] font-medium text-slate-700 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-300">
                                                                                        {tool.server_name}
                                                                                    </span>
                                                                                    <span className="inline-flex rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-1 text-[11px] font-medium text-emerald-800 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-200">
                                                                                        Read only
                                                                                    </span>
                                                                                </div>
                                                                                <div className="mt-2 text-sm text-slate-600 dark:text-slate-400">{tool.description}</div>
                                                                                <div className="mt-2 text-xs uppercase tracking-[0.14em] text-slate-500 dark:text-slate-500">
                                                                                    {tool.server_kind} • {tool.tool_name}
                                                                                </div>
                                                                            </div>
                                                                            <button
                                                                                type="button"
                                                                                onClick={() => void handleRunMCPTool(tool)}
                                                                                disabled={isRunning}
                                                                                className="rounded-lg border border-sky-300 bg-sky-50 px-3 py-2 text-xs font-medium text-sky-700 transition hover:bg-sky-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-sky-800 dark:bg-sky-950/30 dark:text-sky-200 dark:hover:bg-sky-950/50"
                                                                            >
                                                                                {isRunning ? 'Running...' : 'Run Tool'}
                                                                            </button>
                                                                        </div>
                                                                        {result ? (
                                                                            <div className="mt-4 rounded-xl border border-slate-200 bg-slate-950 px-4 py-3 dark:border-slate-700">
                                                                                <div className="mb-2 flex items-center justify-between gap-3">
                                                                                    <div className="text-xs uppercase tracking-[0.16em] text-slate-400">
                                                                                        Output • {formatDateTime(result.executed_at)}
                                                                                    </div>
                                                                                    <button
                                                                                        type="button"
                                                                                        onClick={() => handleDismissMCPToolResult(toolKey)}
                                                                                        className="rounded-md border border-slate-700 bg-slate-900 px-2 py-1 text-[11px] font-medium text-slate-300 transition hover:border-slate-500 hover:text-white"
                                                                                    >
                                                                                        Close
                                                                                    </button>
                                                                                </div>
                                                                                {renderMCPToolResult(result)}
                                                                            </div>
                                                                        ) : null}
                                                                    </div>
                                                                )
                                                            })}
                                                        </div>
                                                    )}
                                                </div>
                                            </div>
                                        )}
                                    </SectionCard>
                                </div>
                            </>
                        ) : null}
                    </div>
                ) : null}
            </Drawer>

            <Drawer
                isOpen={demoDrawerOpen}
                onClose={closeDemoDrawer}
                title="Demo Scenarios"
                description="Generate realistic SRE Smart Bot incidents on demand so you can demo grounded investigation, AI interpretation, and approval-safe action flow."
                width="60vw"
            >
                <div className="grid gap-4 xl:grid-cols-[minmax(0,1.4fr)_minmax(320px,0.9fr)]">
                    <div className="space-y-2">
                        <label className="space-y-2">
                            <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Scenario</span>
                            <select
                                value={selectedDemoScenarioId}
                                onChange={(e) => setSelectedDemoScenarioId(e.target.value)}
                                disabled={demoLoading || generatingDemo}
                                className="w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-200 disabled:cursor-not-allowed disabled:opacity-70 dark:border-slate-700 dark:bg-slate-950 dark:text-slate-100 dark:focus:border-sky-400 dark:focus:ring-sky-900"
                            >
                                {demoScenarios.map((scenario) => (
                                    <option key={scenario.id} value={scenario.id}>
                                        {scenario.name}
                                    </option>
                                ))}
                            </select>
                        </label>
                        <div className="flex flex-wrap items-center gap-3">
                            <button
                                onClick={() => void handleGenerateDemoIncident()}
                                disabled={demoLoading || generatingDemo || !selectedDemoScenarioId}
                                className="rounded-xl bg-sky-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-70 dark:bg-sky-500 dark:hover:bg-sky-400"
                            >
                                {generatingDemo ? 'Generating...' : 'Generate Demo Incident'}
                            </button>
                            <p className="text-xs text-slate-500 dark:text-slate-400">
                                Best flow: generate, open the incident, run MCP tools, then show draft and local interpretation.
                            </p>
                        </div>
                    </div>
                    <div className="rounded-2xl border border-slate-200 bg-slate-50/90 p-4 dark:border-slate-800 dark:bg-slate-950/40">
                        {selectedDemoScenario ? (
                            <>
                                <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">{selectedDemoScenario.name}</h3>
                                <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">{selectedDemoScenario.summary}</p>
                                <div className="mt-4 rounded-xl border border-slate-200 bg-white/80 p-3 dark:border-slate-800 dark:bg-slate-900/70">
                                    <p className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">Suggested Walkthrough</p>
                                    <p className="mt-2 text-sm text-slate-700 dark:text-slate-300">{selectedDemoScenario.recommended_walkthrough}</p>
                                </div>
                            </>
                        ) : (
                            <EmptyState title="No demo scenarios available" description="The backend demo generator is not exposing any scenarios yet." />
                        )}
                    </div>
                </div>
            </Drawer>
        </div>
    )
}

export default SRESmartBotIncidentsPage
