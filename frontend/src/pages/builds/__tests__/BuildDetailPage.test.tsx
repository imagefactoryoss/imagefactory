import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import BuildDetailPage from '../BuildDetailPage'

const getBuildTraceMock = vi.fn()
const getBuildLogsMock = vi.fn()
const getBuildLogEntriesMock = vi.fn()
const getBuildWorkflowMock = vi.fn()
const getBuildExecutionsMock = vi.fn()
const retryBuildMock = vi.fn()
const exportBuildTraceMock = vi.fn()
const startDispatcherMock = vi.fn()
const startOrchestratorMock = vi.fn()
const useAuthStoreMock = vi.fn()
const useTenantStoreMock = vi.fn()

vi.mock('@/services/buildService', () => ({
    buildService: {
        getBuildTrace: (...args: any[]) => getBuildTraceMock(...args),
        getBuildLogs: (...args: any[]) => getBuildLogsMock(...args),
        getBuildLogEntries: (...args: any[]) => getBuildLogEntriesMock(...args),
        getBuildWorkflow: (...args: any[]) => getBuildWorkflowMock(...args),
        getBuildExecutions: (...args: any[]) => getBuildExecutionsMock(...args),
        retryBuild: (...args: any[]) => retryBuildMock(...args),
        exportBuildTrace: (...args: any[]) => exportBuildTraceMock(...args),
    },
}))

vi.mock('@/services/adminService', () => ({
    adminService: {
        startDispatcher: (...args: any[]) => startDispatcherMock(...args),
        startOrchestrator: (...args: any[]) => startOrchestratorMock(...args),
    },
}))

vi.mock('@/hooks/useBuildStatusWebSocket', () => ({
    default: () => ({ isConnected: true }),
}))

vi.mock('@/store/auth', () => ({
    useAuthStore: Object.assign((selector?: any) => {
        const state = useAuthStoreMock()
        return typeof selector === 'function' ? selector(state) : state
    }, {
        getState: () => useAuthStoreMock(),
    }),
}))

vi.mock('@/store/tenant', () => ({
    useTenantStore: Object.assign((selector?: any) => {
        const state = useTenantStoreMock()
        return typeof selector === 'function' ? selector(state) : state
    }, {
        getState: () => useTenantStoreMock(),
    }),
}))

vi.mock('react-hot-toast', () => ({
    default: {
        success: vi.fn(),
        error: vi.fn(),
    },
}))

const baseTraceResponse = {
    build: {
        id: 'build-1',
        tenant_id: 'tenant-1',
        tenantId: 'tenant-1',
        status: 'failed',
        manifest: {
            name: 'nginx-build',
            type: 'container',
            base_image: 'nginx:alpine',
            baseImage: 'nginx:alpine',
            instructions: [],
            environment: {},
            tags: ['latest'],
            metadata: {},
            build_config: {
                dockerfile: 'FROM nginx:alpine',
            },
            buildConfig: {
                dockerfile: 'FROM nginx:alpine',
            },
        },
        created_at: '2026-02-14T07:27:20Z',
        createdAt: '2026-02-14T07:27:20Z',
        updated_at: '2026-02-14T07:27:30Z',
        updatedAt: '2026-02-14T07:27:30Z',
        version: 1,
    },
    executions: [
        {
            id: 'exec-1',
            status: 'failed',
            created_at: '2026-02-14T07:27:21Z',
            createdAt: '2026-02-14T07:27:21Z',
            completed_at: '2026-02-14T07:27:29Z',
            completedAt: '2026-02-14T07:27:29Z',
            error_message: 'dispatch failed',
            errorMessage: 'dispatch failed',
        },
    ],
    selected_execution_id: 'exec-1',
    selectedExecutionId: 'exec-1',
    workflow: {
        status: 'failed',
        steps: [
            {
                step_key: 'build.dispatch',
                stepKey: 'build.dispatch',
                status: 'failed',
                attempts: 1,
                last_error: 'dispatcher unavailable',
                lastError: 'dispatcher unavailable',
                created_at: '2026-02-14T07:27:22Z',
                createdAt: '2026-02-14T07:27:22Z',
                updated_at: '2026-02-14T07:27:24Z',
                updatedAt: '2026-02-14T07:27:24Z',
            },
        ],
    },
    runtime: {
        dispatcher: { enabled: true, running: false, available: false, message: 'down' },
        workflow_orchestrator: { enabled: true, running: false, available: false, message: 'down' },
    },
    correlation: {
        workflow_instance_id: 'wf-12345678-abcd',
        execution_id: 'exec-12345678-abcd',
        active_step_key: 'build.dispatch',
        workflowInstanceId: 'wf-12345678-abcd',
        executionId: 'exec-12345678-abcd',
        activeStepKey: 'build.dispatch',
    },
}

const renderPage = () => {
    return render(
        <MemoryRouter initialEntries={['/builds/build-1']}>
            <Routes>
                <Route path="/builds/:buildId" element={<BuildDetailPage />} />
            </Routes>
        </MemoryRouter>
    )
}

describe('BuildDetailPage trace controls', () => {
    beforeEach(() => {
        vi.clearAllMocks()
            ; (window as any).confirm = vi.fn(() => true)
        Object.defineProperty(navigator, 'clipboard', {
            value: { writeText: vi.fn().mockResolvedValue(undefined) },
            configurable: true,
        })

        getBuildTraceMock.mockResolvedValue(baseTraceResponse)
        getBuildLogsMock.mockResolvedValue({ executionId: 'exec-1', lines: ['log line'] })
        getBuildLogEntriesMock.mockResolvedValue({ executionId: 'exec-1', entries: [{ timestamp: '2026-02-14T07:27:22Z', level: 'INFO', message: 'log line', metadata: { source: 'tekton', task_run: 'task-1', step: 'clone', pipeline_run: 'pr-1' } }] })
        getBuildWorkflowMock.mockResolvedValue(baseTraceResponse.workflow)
        getBuildExecutionsMock.mockResolvedValue({ executions: [{ id: 'exec-2' }, { id: 'exec-1' }] })
        retryBuildMock.mockResolvedValue({ status: 'queued', source_build_id: 'build-1' })
        exportBuildTraceMock.mockResolvedValue(undefined)
        startDispatcherMock.mockResolvedValue(undefined)
        startOrchestratorMock.mockResolvedValue(undefined)
        useAuthStoreMock.mockReturnValue({ user: { id: 'user-1' }, groups: [{ role_type: 'system_administrator' }] })
        useTenantStoreMock.mockReturnValue({ selectedTenantId: 'tenant-1' })
    })

    it('runs retry flow and starts a new attempt', async () => {
        renderPage()

        await screen.findByText('Build Execution Trace')

        fireEvent.click(screen.getByRole('button', { name: 'Retry (New Attempt)' }))

        await waitFor(() => {
            expect(retryBuildMock).toHaveBeenCalledWith('build-1')
            expect(getBuildExecutionsMock).toHaveBeenCalledWith('build-1', { limit: 25, offset: 0 })
        })
    })

    it('exports trace and copies correlation IDs', async () => {
        renderPage()

        await screen.findByText('Build Execution Trace')

        // historical log entry should be shown (structured entries)
        await waitFor(() => expect(screen.getByText('log line')).toBeInTheDocument())
        // metadata is hidden by default behind a toggle
        await waitFor(() => expect(screen.getByRole('button', { name: /show metadata/i })).toBeInTheDocument())
        expect(screen.queryByText(/Task: task-1/)).not.toBeInTheDocument()
        fireEvent.click(screen.getByRole('button', { name: /show metadata/i }))
        await waitFor(() => expect(screen.getByText(/Task: task-1/)).toBeInTheDocument())
        // grouped TaskRun header should appear
        await waitFor(() => expect(screen.getByText(/TaskRun: task-1/)).toBeInTheDocument())

        fireEvent.click(screen.getAllByRole('button', { name: 'Export Trace' })[0])
        await waitFor(() => {
            expect(exportBuildTraceMock).toHaveBeenCalledWith('build-1', 'exec-1')
        })

        const wfButtons = screen.getAllByRole('button').filter((button) => button.textContent?.startsWith('WF:'))
        fireEvent.click(wfButtons[0])
        await waitFor(() => {
            expect(navigator.clipboard.writeText).toHaveBeenCalledWith('wf-12345678-abcd')
        })

        const exButtons = screen.getAllByRole('button').filter((button) => button.textContent?.startsWith('EX:'))
        fireEvent.click(exButtons[0])
        await waitFor(() => {
            expect(navigator.clipboard.writeText).toHaveBeenCalledWith('exec-12345678-abcd')
        })
    })

    it('shows and executes recovery actions for system admin', async () => {
        renderPage()

        await screen.findByText('Recovery Hint')

        expect(screen.getAllByRole('button', { name: 'Start Orchestrator' }).length).toBeGreaterThan(0)
        expect(screen.getAllByRole('button', { name: 'Start Dispatcher' }).length).toBeGreaterThan(0)

        fireEvent.click(screen.getAllByRole('button', { name: 'Start Dispatcher' })[0])

        await waitFor(() => {
            expect(startDispatcherMock).toHaveBeenCalled()
        })

        expect(screen.getAllByRole('link', { name: 'Open Pipeline Health' }).length).toBeGreaterThan(0)
    })

    it('hides admin-only visibility and disables operator actions for non-admin users', async () => {
        useAuthStoreMock.mockReturnValue({ groups: [{ role_type: 'tenant_developer' }] })
        renderPage()

        await screen.findByText('Recovery Hint')

        expect(screen.queryByRole('link', { name: 'Open Pipeline Health' })).not.toBeInTheDocument()
        expect(screen.getByText('Operator actions require system administrator permissions.')).toBeInTheDocument()

        const orchestratorButtons = screen.getAllByRole('button', { name: 'Start Orchestrator' })
        const dispatcherButtons = screen.getAllByRole('button', { name: 'Start Dispatcher' })

        orchestratorButtons.forEach((button) => expect(button).toBeDisabled())
        dispatcherButtons.forEach((button) => expect(button).toBeDisabled())
    })
})
