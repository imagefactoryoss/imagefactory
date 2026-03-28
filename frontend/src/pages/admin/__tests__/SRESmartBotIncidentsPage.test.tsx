import { render, screen, waitFor, within } from '@testing-library/react'
import { act } from 'react'
import { MemoryRouter } from 'react-router-dom'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import SRESmartBotIncidentsPage from '../SRESmartBotIncidentsPage'

const getSREIncidentsMock = vi.fn()
const getSREDemoScenariosMock = vi.fn()
const getSRESmartBotPolicyMock = vi.fn()
const getSREIncidentMock = vi.fn()
const getSREIncidentWorkspaceMock = vi.fn()
const getSREIncidentMCPToolsMock = vi.fn()
const getSREIncidentRemediationPacksMock = vi.fn()
const getSREIncidentRemediationPackRunsMock = vi.fn()
const dryRunSREIncidentRemediationPackMock = vi.fn()
const executeSREIncidentRemediationPackMock = vi.fn()
const toastSuccessMock = vi.fn()
const toastErrorMock = vi.fn()

vi.mock('@/components/ui/Drawer', () => ({
    __esModule: true,
    default: ({ isOpen, children }: any) => (isOpen ? <div>{children}</div> : null),
}))

vi.mock('@/services/adminService', () => ({
    adminService: {
        getSREIncidents: (...args: any[]) => getSREIncidentsMock(...args),
        getSREDemoScenarios: (...args: any[]) => getSREDemoScenariosMock(...args),
        getSRESmartBotPolicy: (...args: any[]) => getSRESmartBotPolicyMock(...args),
        getSREIncident: (...args: any[]) => getSREIncidentMock(...args),
        getSREIncidentWorkspace: (...args: any[]) => getSREIncidentWorkspaceMock(...args),
        getSREIncidentMCPTools: (...args: any[]) => getSREIncidentMCPToolsMock(...args),
        getSREIncidentRemediationPacks: (...args: any[]) => getSREIncidentRemediationPacksMock(...args),
        getSREIncidentRemediationPackRuns: (...args: any[]) => getSREIncidentRemediationPackRunsMock(...args),
        dryRunSREIncidentRemediationPack: (...args: any[]) => dryRunSREIncidentRemediationPackMock(...args),
        executeSREIncidentRemediationPack: (...args: any[]) => executeSREIncidentRemediationPackMock(...args),
        requestSREActionApproval: vi.fn(),
        executeSREAction: vi.fn(),
        proposeSREDetectorRuleSuggestion: vi.fn(),
        decideSREApproval: vi.fn(),
        emailSREIncidentSummary: vi.fn(),
        invokeSREIncidentMCPTool: vi.fn(),
        getSREIncidentAgentDraft: vi.fn(),
        getSREIncidentAgentInterpretation: vi.fn(),
        generateSREDemoIncident: vi.fn(),
    },
}))

vi.mock('react-hot-toast', () => ({
    default: {
        success: (...args: any[]) => toastSuccessMock(...args),
        error: (...args: any[]) => toastErrorMock(...args),
    },
}))

describe('SRESmartBotIncidentsPage remediation packs', () => {
    const incident = {
        id: 'incident-1',
        tenant_id: 'tenant-1',
        correlation_key: 'corr-1',
        domain: 'runtime_services',
        incident_type: 'nats_transport_disconnect_storm',
        display_name: 'NATS reconnect storm',
        summary: 'Transport instability observed',
        severity: 'warning',
        confidence: 'high',
        status: 'observed',
        source: 'detector',
        first_observed_at: '2026-03-16T10:00:00Z',
        last_observed_at: '2026-03-16T10:05:00Z',
        created_at: '2026-03-16T10:00:00Z',
        updated_at: '2026-03-16T10:05:00Z',
    }

    const flushAsyncState = async () => {
        await act(async () => {
            await Promise.resolve()
        })
    }

    const renderPage = async () => {
        render(
            <MemoryRouter initialEntries={[`/admin/operations/sre-smart-bot?incident=${incident.id}`]}>
                <SRESmartBotIncidentsPage />
            </MemoryRouter>
        )

        await waitFor(() => expect(getSREIncidentsMock).toHaveBeenCalled())
        await waitFor(() => expect(getSREDemoScenariosMock).toHaveBeenCalled())
        await waitFor(() => expect(getSREIncidentMock).toHaveBeenCalledWith(incident.id))

        await screen.findByText('Guided Remediation Packs')
        await flushAsyncState()
        await flushAsyncState()
    }

    beforeEach(() => {
        vi.clearAllMocks()

        getSREIncidentsMock.mockResolvedValue({ incidents: [incident], total: 1, limit: 100, offset: 0 })
        getSREDemoScenariosMock.mockResolvedValue({ scenarios: [] })
        getSRESmartBotPolicyMock.mockResolvedValue({ channel_providers: [], default_channel_provider_id: '' })

        getSREIncidentMock.mockResolvedValue({
            incident,
            findings: [],
            evidence: [],
            action_attempts: [],
            approvals: [],
        })
        getSREIncidentWorkspaceMock.mockResolvedValue(null)
        getSREIncidentMCPToolsMock.mockResolvedValue({ tools: [] })
        getSREIncidentRemediationPacksMock.mockResolvedValue({
            packs: [
                {
                    key: 'nats_transport_stability_pack',
                    version: 'v1',
                    name: 'NATS Transport Stability Pack',
                    summary: 'Validates transport health preconditions and guides bounded recovery actions.',
                    risk_tier: 'medium',
                    action_class: 'guided_remediation',
                    requires_approval: true,
                    incident_types: ['nats_transport_disconnect_storm'],
                },
            ],
        })
        getSREIncidentRemediationPackRunsMock.mockResolvedValue({ runs: [] })
        dryRunSREIncidentRemediationPackMock.mockResolvedValue({ run: { id: 'run-1' } })
        executeSREIncidentRemediationPackMock.mockResolvedValue({ run: { id: 'run-2' } })
    })

    it('renders remediation packs in actions tab and runs dry-run', async () => {
        const user = userEvent.setup()
        await renderPage()
        expect(await screen.findByText('NATS Transport Stability Pack')).toBeInTheDocument()

        const dryRunButton = screen.getByRole('button', { name: 'Dry Run' })
        await user.click(dryRunButton)

        await waitFor(() => {
            expect(dryRunSREIncidentRemediationPackMock).toHaveBeenCalledWith(incident.id, 'nats_transport_stability_pack')
        })
        await flushAsyncState()
    })

    it('blocks approval-required execute when no approved approval exists', async () => {
        const user = userEvent.setup()
        await renderPage()

        const executeButton = await screen.findByRole('button', { name: 'Execute' })
        await user.click(executeButton)

        await waitFor(() => {
            expect(toastErrorMock).toHaveBeenCalledWith('This remediation pack requires an approved approval decision before execution')
        })
        expect(executeSREIncidentRemediationPackMock).not.toHaveBeenCalled()
        await flushAsyncState()
    })

    it('executes approval-required remediation pack with selected approved approval id', async () => {
        const user = userEvent.setup()
        getSREIncidentMock.mockResolvedValue({
            incident,
            findings: [],
            evidence: [],
            action_attempts: [],
            approvals: [
                {
                    id: 'approval-1',
                    incident_id: incident.id,
                    status: 'approved',
                    request_message: 'ok',
                    channel_provider_id: 'in_app',
                    requested_at: '2026-03-16T10:00:00Z',
                    decided_at: '2026-03-16T10:01:00Z',
                    created_at: '2026-03-16T10:00:00Z',
                    updated_at: '2026-03-16T10:01:00Z',
                },
            ],
        })

        await renderPage()

        const packName = await screen.findByText('NATS Transport Stability Pack')
        const packCard = packName.closest('div.rounded-xl.border')
        expect(packCard).toBeTruthy()
        const select = within(packCard as HTMLElement).getByRole('combobox')
        await user.selectOptions(select, 'approval-1')

        const executeButton = await screen.findByRole('button', { name: 'Execute' })
        await user.click(executeButton)

        await waitFor(() => {
            expect(executeSREIncidentRemediationPackMock).toHaveBeenCalledWith(incident.id, 'nats_transport_stability_pack', {
                approval_id: 'approval-1',
            })
        })
        await flushAsyncState()
    })
})
