import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import QuarantineReviewWorkbenchPage from '../QuarantineReviewWorkbenchPage'

const listAdminImportRequestsMock = vi.fn()
const approveImportRequestMock = vi.fn()
const rejectImportRequestMock = vi.fn()
const releaseImportRequestMock = vi.fn()
const getSystemStatsMock = vi.fn()
const getReleaseGovernancePolicyMock = vi.fn()
const listAdminSORRequestsMock = vi.fn()
const approveSORRequestMock = vi.fn()
const rejectSORRequestMock = vi.fn()
const suspendSORRequestMock = vi.fn()
const reactivateSORRequestMock = vi.fn()
const revalidateSORRequestMock = vi.fn()
const bulkSuspendSORRequestsMock = vi.fn()
const bulkReactivateSORRequestsMock = vi.fn()
const bulkRevalidateSORRequestsMock = vi.fn()
const confirmDialogMock = vi.fn()
const toastSuccessMock = vi.fn()
const toastErrorMock = vi.fn()

vi.mock('@/services/imageImportService', async () => {
  const actual = await vi.importActual<typeof import('@/services/imageImportService')>('@/services/imageImportService')
  return {
    ...actual,
    imageImportService: {
      listImportRequests: (...args: any[]) => listAdminImportRequestsMock(...args),
      listAdminImportRequests: (...args: any[]) => listAdminImportRequestsMock(...args),
      approveImportRequest: (...args: any[]) => approveImportRequestMock(...args),
      approveAdminImportRequest: (...args: any[]) => approveImportRequestMock(...args),
      rejectImportRequest: (...args: any[]) => rejectImportRequestMock(...args),
      rejectAdminImportRequest: (...args: any[]) => rejectImportRequestMock(...args),
      releaseImportRequest: (...args: any[]) => releaseImportRequestMock(...args),
    },
  }
})

vi.mock('@/context/ConfirmDialogContext', () => ({
  useConfirmDialog: () => confirmDialogMock,
}))

vi.mock('@/services/adminService', () => ({
  adminService: {
    getSystemStats: (...args: any[]) => getSystemStatsMock(...args),
    getReleaseGovernancePolicy: (...args: any[]) => getReleaseGovernancePolicyMock(...args),
  },
}))

vi.mock('@/services/eprRegistrationService', () => ({
  eprRegistrationService: {
    listAdminRequests: (...args: any[]) => listAdminSORRequestsMock(...args),
    approveRequest: (...args: any[]) => approveSORRequestMock(...args),
    rejectRequest: (...args: any[]) => rejectSORRequestMock(...args),
    suspendRequest: (...args: any[]) => suspendSORRequestMock(...args),
    reactivateRequest: (...args: any[]) => reactivateSORRequestMock(...args),
    revalidateRequest: (...args: any[]) => revalidateSORRequestMock(...args),
    bulkSuspendRequests: (...args: any[]) => bulkSuspendSORRequestsMock(...args),
    bulkReactivateRequests: (...args: any[]) => bulkReactivateSORRequestsMock(...args),
    bulkRevalidateRequests: (...args: any[]) => bulkRevalidateSORRequestsMock(...args),
  },
}))

vi.mock('react-hot-toast', () => ({
  __esModule: true,
  default: {
    success: (...args: any[]) => toastSuccessMock(...args),
    error: (...args: any[]) => toastErrorMock(...args),
  },
}))

describe('QuarantineReviewWorkbenchPage', () => {
  const pendingRow = {
    id: 'req-pending-1',
    tenant_id: 'tenant-1',
    requested_by_user_id: 'user-1',
    epr_record_id: 'EPR-1',
    source_registry: 'ghcr.io',
    source_image_ref: 'ghcr.io/acme/app:1.0.0',
    status: 'pending',
    retryable: false,
    sync_state: 'awaiting_dispatch',
    scan_summary_json: '{"critical":1}',
    sbom_summary_json: '{"packages":120}',
    created_at: '2026-02-28T00:00:00Z',
    updated_at: '2026-02-28T00:05:00Z',
  }

  beforeEach(() => {
    vi.clearAllMocks()
    let promptCall = 0
    vi.spyOn(window, 'prompt').mockImplementation(() => {
      promptCall += 1
      if (promptCall % 2 === 1) {
        return 'registry.example.com/released/app:1.0.0'
      }
      return '11111111-1111-1111-1111-111111111111'
    })
    confirmDialogMock.mockResolvedValue(true)
    listAdminImportRequestsMock.mockResolvedValue([pendingRow])
    listAdminImportRequestsMock.mockResolvedValue([pendingRow])
    approveImportRequestMock.mockResolvedValue(undefined)
    rejectImportRequestMock.mockResolvedValue(undefined)
    releaseImportRequestMock.mockResolvedValue(undefined)
    getSystemStatsMock.mockResolvedValue({
      release_metrics: {
        requested: 4,
        released: 3,
        failed: 1,
        total: 4,
      },
      epr_lifecycle_metrics: {
        total: 6,
        pending: 2,
        approved: 4,
        rejected: 0,
        active: 3,
        expiring: 1,
        expired: 1,
        suspended: 1,
      },
    })
    getReleaseGovernancePolicyMock.mockResolvedValue({
      enabled: true,
      failure_ratio_threshold: 0.5,
      consecutive_failures_threshold: 3,
      minimum_samples: 3,
      window_minutes: 60,
    })
    listAdminSORRequestsMock.mockResolvedValue([])
    approveSORRequestMock.mockResolvedValue(undefined)
    rejectSORRequestMock.mockResolvedValue(undefined)
    suspendSORRequestMock.mockResolvedValue(undefined)
    reactivateSORRequestMock.mockResolvedValue(undefined)
    revalidateSORRequestMock.mockResolvedValue(undefined)
    bulkSuspendSORRequestsMock.mockResolvedValue([])
    bulkReactivateSORRequestsMock.mockResolvedValue([])
    bulkRevalidateSORRequestsMock.mockResolvedValue([])
  })

  it('renders pending review rows and opens details drawer', async () => {
    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('ghcr.io/acme/app:1.0.0')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Details' }))

    expect(await screen.findByText('Reviewer Request Detail')).toBeInTheDocument()
    expect(screen.getByText('No reviewer decision has been recorded yet.')).toBeInTheDocument()
    expect(screen.getByText('No notification reconciliation checkpoints are available yet.')).toBeInTheDocument()
    expect(screen.getByText('Release State:')).toBeInTheDocument()
    expect(screen.getByText('Operational Diagnostics')).toBeInTheDocument()
    expect(screen.getByText('Scan Summary JSON')).toBeInTheDocument()
    expect(screen.getByText('SBOM Summary JSON')).toBeInTheDocument()
  })

  it('submits approve decision and updates queue immediately', async () => {
    listAdminImportRequestsMock.mockResolvedValueOnce([pendingRow])
    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('ghcr.io/acme/app:1.0.0')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Approve' }))

    await waitFor(() => {
      expect(confirmDialogMock).toHaveBeenCalled()
      expect(approveImportRequestMock).toHaveBeenCalledWith('req-pending-1')
      expect(toastSuccessMock).toHaveBeenCalledWith('Approval decision queued')
    })
    await waitFor(() => {
      expect(screen.queryByText('ghcr.io/acme/app:1.0.0')).not.toBeInTheDocument()
    })
  })

  it('submits reject decision and updates queue immediately', async () => {
    listAdminImportRequestsMock.mockResolvedValueOnce([pendingRow])
    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('ghcr.io/acme/app:1.0.0')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Reject' }))

    await waitFor(() => {
      expect(confirmDialogMock).toHaveBeenCalled()
      expect(rejectImportRequestMock).toHaveBeenCalledWith('req-pending-1', 'Rejected by security reviewer')
      expect(toastSuccessMock).toHaveBeenCalledWith('Rejection decision queued')
    })
    await waitFor(() => {
      expect(screen.queryByText('ghcr.io/acme/app:1.0.0')).not.toBeInTheDocument()
    })
  })

  it('shows toast error when approve action fails', async () => {
    approveImportRequestMock.mockRejectedValueOnce(new Error('approve failed'))
    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('ghcr.io/acme/app:1.0.0')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Approve' }))

    await waitFor(() => {
      expect(approveImportRequestMock).toHaveBeenCalledWith('req-pending-1')
      expect(toastErrorMock).toHaveBeenCalledWith('approve failed')
    })
  })

  it('shows explicit permission guidance when queue load is forbidden', async () => {
    const { ImageImportApiError } = await import('@/services/imageImportService')
    listAdminImportRequestsMock.mockRejectedValueOnce(
      new ImageImportApiError({
        code: 'unknown_error',
        message: 'forbidden',
        status: 403,
      })
    )

    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage />
      </MemoryRouter>
    )

    expect(await screen.findByText(/do not have permission to review quarantine approvals/i)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /capability matrix/i })).toHaveAttribute('href', '/help/capability-access')
  })

  it('maps forbidden approve action errors to reviewer guidance', async () => {
    const { ImageImportApiError } = await import('@/services/imageImportService')
    approveImportRequestMock.mockRejectedValueOnce(
      new ImageImportApiError({
        code: 'unknown_error',
        message: 'forbidden',
        status: 403,
      })
    )

    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('ghcr.io/acme/app:1.0.0')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Approve' }))

    await waitFor(() => {
      expect(approveImportRequestMock).toHaveBeenCalledWith('req-pending-1')
      expect(toastErrorMock).toHaveBeenCalledWith(
        'You do not have permission to review quarantine approvals. Contact the platform administrator to verify Security Reviewer access.'
      )
    })
  })

  it('filters queue by search term and status', async () => {
    listAdminImportRequestsMock.mockResolvedValueOnce([
      pendingRow,
      {
        ...pendingRow,
        id: 'req-done-1',
        status: 'success',
        source_image_ref: 'ghcr.io/acme/other:2.0.0',
        epr_record_id: 'EPR-2',
      },
    ])

    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('ghcr.io/acme/app:1.0.0')).toBeInTheDocument()
    expect((await screen.findAllByText(/ghcr\.io\/acme\/other:2\.0\.0/i)).length).toBeGreaterThan(0)

    fireEvent.change(screen.getByPlaceholderText(/search by image, epr, status, sync state/i), {
      target: { value: 'other' },
    })
    expect(screen.queryByText('ghcr.io/acme/app:1.0.0')).not.toBeInTheDocument()
    expect(screen.getByText(/ghcr\.io\/acme\/other:2\.0\.0/i)).toBeInTheDocument()

    fireEvent.change(screen.getByRole('combobox'), {
      target: { value: 'pending' },
    })
    expect(screen.queryByText(/ghcr\.io\/acme\/other:2\.0\.0/i)).not.toBeInTheDocument()
    expect(screen.getByText('No pending approvals.')).toBeInTheDocument()
  })

  it('shows degraded release-governance banner when thresholds are breached', async () => {
    getSystemStatsMock.mockResolvedValue({
      release_metrics: {
        requested: 8,
        released: 2,
        failed: 6,
        total: 8,
      },
      epr_lifecycle_metrics: {
        total: 5,
        pending: 1,
        approved: 4,
        rejected: 0,
        active: 1,
        expiring: 1,
        expired: 2,
        suspended: 1,
      },
    })
    getReleaseGovernancePolicyMock.mockResolvedValue({
      enabled: true,
      failure_ratio_threshold: 0.5,
      consecutive_failures_threshold: 1,
      minimum_samples: 5,
      window_minutes: 60,
    })
    listAdminImportRequestsMock.mockResolvedValueOnce([
      {
        ...pendingRow,
        id: 'req-blocked-telemetry',
        status: 'failed',
        release_state: 'release_blocked',
        release_eligible: false,
      },
    ])

    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('Release Governance')).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getAllByText('degraded').length).toBeGreaterThan(0)
      expect(screen.getByText('75.0%')).toBeInTheDocument()
    })
  })

  it('renders epr lifecycle metrics from admin stats', async () => {
    getSystemStatsMock.mockImplementation(() => Promise.resolve({
      release_metrics: {
        requested: 1,
        released: 1,
        failed: 0,
        total: 1,
      },
      epr_lifecycle_metrics: {
        total: 9,
        pending: 1,
        approved: 8,
        rejected: 0,
        active: 5,
        expiring: 1,
        expired: 2,
        suspended: 1,
      },
    }))
    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage mode="epr" />
      </MemoryRouter>
    )

    expect(await screen.findByText('EPR Lifecycle Metrics')).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByText((_, element) => element?.textContent === 'Active: 5')).toBeInTheDocument()
      expect(screen.getByText((_, element) => element?.textContent === 'Expiring: 1')).toBeInTheDocument()
      expect(screen.getByText((_, element) => element?.textContent === 'Expired: 2')).toBeInTheDocument()
      expect(screen.getByText((_, element) => element?.textContent === 'Suspended: 1')).toBeInTheDocument()
      expect(screen.getByText((_, element) => element?.textContent === 'Total 9 • At-risk 4 (44.4%)')).toBeInTheDocument()
      expect(screen.getAllByText(/degraded/i).length).toBeGreaterThan(0)
    })
  })

  it('filters EPR queue by lifecycle tabs in epr mode', async () => {
    listAdminSORRequestsMock.mockResolvedValueOnce([
      {
        id: 'epr-1',
        tenant_id: 'tenant-1',
        epr_record_id: 'EPR-ACTIVE',
        product_name: 'Product One',
        technology_name: 'Tech One',
        requested_by_user_id: 'user-1',
        status: 'approved',
        lifecycle_status: 'active',
        created_at: '2026-03-04T00:00:00Z',
        updated_at: '2026-03-04T00:00:00Z',
      },
      {
        id: 'epr-2',
        tenant_id: 'tenant-1',
        epr_record_id: 'EPR-SUSPENDED',
        product_name: 'Product Two',
        technology_name: 'Tech Two',
        requested_by_user_id: 'user-1',
        status: 'approved',
        lifecycle_status: 'suspended',
        created_at: '2026-03-04T00:00:00Z',
        updated_at: '2026-03-04T00:00:00Z',
      },
      {
        id: 'epr-3',
        tenant_id: 'tenant-1',
        epr_record_id: 'EPR-PENDING',
        product_name: 'Product Three',
        technology_name: 'Tech Three',
        requested_by_user_id: 'user-1',
        status: 'pending',
        lifecycle_status: 'active',
        created_at: '2026-03-04T00:00:00Z',
        updated_at: '2026-03-04T00:00:00Z',
      },
    ])
    listAdminImportRequestsMock.mockResolvedValueOnce([])

    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage mode="epr" />
      </MemoryRouter>
    )

    expect(await screen.findByText('EPR Registration Approvals (3)')).toBeInTheDocument()
    expect(screen.getByText('EPR-ACTIVE')).toBeInTheDocument()
    expect(screen.getByText('EPR-SUSPENDED')).toBeInTheDocument()
    expect(screen.getByText('EPR-PENDING')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Suspended' }))
    expect(await screen.findByText('EPR Registration Approvals (1)')).toBeInTheDocument()
    expect(screen.queryByText('EPR-ACTIVE')).not.toBeInTheDocument()
    expect(screen.getByText('EPR-SUSPENDED')).toBeInTheDocument()
    expect(screen.queryByText('EPR-PENDING')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Pending' }))
    expect(await screen.findByText('EPR Registration Approvals (1)')).toBeInTheDocument()
    expect(screen.queryByText('EPR-ACTIVE')).not.toBeInTheDocument()
    expect(screen.queryByText('EPR-SUSPENDED')).not.toBeInTheDocument()
    expect(screen.getByText('EPR-PENDING')).toBeInTheDocument()
  })

  it('executes bulk suspend for selected approved EPR rows', async () => {
    listAdminSORRequestsMock.mockResolvedValueOnce([
      {
        id: 'epr-10',
        tenant_id: 'tenant-1',
        epr_record_id: 'EPR-BULK-1',
        product_name: 'Bulk Product One',
        technology_name: 'Bulk Tech One',
        requested_by_user_id: 'user-1',
        status: 'approved',
        lifecycle_status: 'active',
        created_at: '2026-03-04T00:00:00Z',
        updated_at: '2026-03-04T00:00:00Z',
      },
      {
        id: 'epr-11',
        tenant_id: 'tenant-1',
        epr_record_id: 'EPR-BULK-2',
        product_name: 'Bulk Product Two',
        technology_name: 'Bulk Tech Two',
        requested_by_user_id: 'user-1',
        status: 'approved',
        lifecycle_status: 'expiring',
        created_at: '2026-03-04T00:00:00Z',
        updated_at: '2026-03-04T00:00:00Z',
      },
    ])
    listAdminImportRequestsMock.mockResolvedValueOnce([])
    const promptSpy = vi.spyOn(window, 'prompt').mockReturnValue('Bulk reason')

    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage mode="epr" />
      </MemoryRouter>
    )

    expect(await screen.findByText('EPR-BULK-1')).toBeInTheDocument()
    fireEvent.click(screen.getByLabelText('Select EPR-BULK-1'))
    fireEvent.click(screen.getByLabelText('Select EPR-BULK-2'))
    fireEvent.click(screen.getByRole('button', { name: /Bulk Suspend \(2\)/i }))

    await waitFor(() => {
      expect(promptSpy).toHaveBeenCalled()
      expect(confirmDialogMock).toHaveBeenCalled()
      expect(bulkSuspendSORRequestsMock).toHaveBeenCalledWith(['epr-10', 'epr-11'], 'Bulk reason')
      expect(toastSuccessMock).toHaveBeenCalledWith('Suspended 2 EPR registration(s)')
    })
    promptSpy.mockRestore()
  })

  it('shows EPR drawer action controls and executes approve from drawer', async () => {
    listAdminSORRequestsMock.mockResolvedValueOnce([
      {
        id: 'epr-drawer-1',
        tenant_id: 'tenant-1',
        epr_record_id: 'EPR-DRAWER-1',
        product_name: 'Drawer Product',
        technology_name: 'Drawer Tech',
        requested_by_user_id: 'user-1',
        status: 'pending',
        lifecycle_status: 'active',
        created_at: '2026-03-04T00:00:00Z',
        updated_at: '2026-03-04T00:00:00Z',
      },
    ])
    listAdminImportRequestsMock.mockResolvedValueOnce([])

    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage mode="epr" />
      </MemoryRouter>
    )

    expect(await screen.findByText('EPR-DRAWER-1')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'View' }))
    expect(await screen.findByText('EPR Registration Detail')).toBeInTheDocument()
    expect(screen.getAllByText('Actions').length).toBeGreaterThan(0)
    const approveButtons = screen.getAllByRole('button', { name: 'Approve' })
    fireEvent.click(approveButtons[approveButtons.length - 1] as HTMLElement)

    await waitFor(() => {
      expect(confirmDialogMock).toHaveBeenCalled()
      expect(approveSORRequestMock).toHaveBeenCalledWith('epr-drawer-1', 'Approved by security reviewer')
      expect(toastSuccessMock).toHaveBeenCalledWith('EPR registration approved')
    })
  })

  it('renders decision timeline metadata when present', async () => {
    listAdminImportRequestsMock.mockResolvedValueOnce([
      {
        ...pendingRow,
        id: 'req-done-2',
        status: 'pending',
        source_image_ref: 'ghcr.io/acme/reviewed:3.0.0',
        decision_timeline: {
          decision_status: 'approved',
          workflow_step_status: 'succeeded',
          decided_by_user_id: 'user-42',
          decided_at: '2026-03-01T10:00:00Z',
        },
        notification_reconciliation: {
          decision_event_type: 'external.image.import.approved',
          idempotency_key: 'req-done-2:approved',
          expected_recipients: 2,
          receipt_count: 2,
          in_app_notification_count: 2,
          delivery_state: 'delivered',
        },
      },
    ])
    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage />
      </MemoryRouter>
    )

    expect(await screen.findByText(/ghcr\.io\/acme\/reviewed:3\.0\.0/i)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Details' }))

    expect(await screen.findByText('Decision Timeline')).toBeInTheDocument()
    expect(screen.getByText(/Decision: approved/i)).toBeInTheDocument()
    expect(screen.getByText(/Step State: succeeded/i)).toBeInTheDocument()
    expect(screen.getByText(/Decided By: user-42/i)).toBeInTheDocument()
    expect(screen.getByText(/Delivery State: delivered/i)).toBeInTheDocument()
    expect(screen.getByText(/Receipts: 2 \/ 2/i)).toBeInTheDocument()
    expect(screen.getByText(/In-App Notifications: 2 \/ 2/i)).toBeInTheDocument()
  })

  it('releases release-ready artifacts and reloads queue', async () => {
    listAdminImportRequestsMock
      .mockResolvedValueOnce([
        {
          ...pendingRow,
          id: 'req-release-1',
          status: 'success',
          source_image_ref: 'ghcr.io/acme/released:4.0.0',
          release_state: 'ready_for_release',
          release_eligible: true,
        },
      ])
      .mockResolvedValueOnce([])

    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage />
      </MemoryRouter>
    )

    expect(await screen.findByRole('button', { name: 'Release' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Release' }))

    await waitFor(() => {
      expect(confirmDialogMock).toHaveBeenCalled()
      expect(releaseImportRequestMock).toHaveBeenCalledWith('req-release-1', {
        destinationImageRef: 'registry.example.com/released/app:1.0.0',
        destinationRegistryAuthId: '11111111-1111-1111-1111-111111111111',
      })
      expect(toastSuccessMock).toHaveBeenCalledWith('Release completed')
    })
    expect(listAdminImportRequestsMock).toHaveBeenCalledTimes(2)
  })

  it('renders compact release governance lanes for blocked, in-progress and released items', async () => {
    listAdminImportRequestsMock.mockResolvedValueOnce([
      {
        ...pendingRow,
        id: 'req-blocked-1',
        status: 'quarantined',
        source_image_ref: 'ghcr.io/acme/blocked:1.0.0',
        release_state: 'release_blocked',
        release_eligible: false,
        release_blocker_reason: 'policy violation: critical vulnerabilities',
      },
      {
        ...pendingRow,
        id: 'req-failed-1',
        status: 'failed',
        source_image_ref: 'ghcr.io/acme/release-failed:1.0.0',
        release_state: 'release_blocked',
        release_eligible: false,
        error_message: 'release event publish failed',
      },
      {
        ...pendingRow,
        id: 'req-progress-1',
        status: 'success',
        source_image_ref: 'ghcr.io/acme/progress:1.0.0',
        release_state: 'release_approved',
        release_eligible: false,
      },
      {
        ...pendingRow,
        id: 'req-released-1',
        status: 'success',
        source_image_ref: 'ghcr.io/acme/released:1.0.0',
        release_state: 'released',
        release_eligible: false,
      },
    ])

    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage />
      </MemoryRouter>
    )

    expect(await screen.findByText(/policy violation: critical vulnerabilities/i)).toBeInTheDocument()
    expect(screen.getByText(/blocked \(2\)/i)).toBeInTheDocument()
    expect(screen.getByText(/in progress \(1\)/i)).toBeInTheDocument()
    expect(screen.getByText(/released \(1\)/i)).toBeInTheDocument()
    expect(screen.getByText(/release event publish failed/i)).toBeInTheDocument()
    expect(screen.getByText(/policy violation: critical vulnerabilities/i)).toBeInTheDocument()
    expect(screen.getByText(/release request accepted and awaiting completion/i)).toBeInTheDocument()
    expect(screen.getByText(/available for tenant consumption/i)).toBeInTheDocument()
  })

  it('surfaces release blocker reason when backend returns release_not_eligible', async () => {
    const { ImageImportApiError } = await import('@/services/imageImportService')
    listAdminImportRequestsMock.mockResolvedValueOnce([
      {
        ...pendingRow,
        id: 'req-release-2',
        status: 'success',
        source_image_ref: 'ghcr.io/acme/released:5.0.0',
        release_state: 'ready_for_release',
        release_eligible: true,
      },
    ])
    releaseImportRequestMock.mockRejectedValueOnce(
      new ImageImportApiError({
        code: 'release_not_eligible',
        message: 'not eligible',
        status: 409,
        details: {
          release_blocker_reason: 'policy snapshot stale',
        },
      })
    )
    listAdminImportRequestsMock.mockResolvedValueOnce([])

    render(
      <MemoryRouter>
        <QuarantineReviewWorkbenchPage />
      </MemoryRouter>
    )

    expect(await screen.findByRole('button', { name: 'Release' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Release' }))

    await waitFor(() => {
      expect(releaseImportRequestMock).toHaveBeenCalledWith('req-release-2', {
        destinationImageRef: 'registry.example.com/released/app:1.0.0',
        destinationRegistryAuthId: '11111111-1111-1111-1111-111111111111',
      })
      expect(toastErrorMock).toHaveBeenCalledWith('Release blocked: policy snapshot stale')
    })
    expect(listAdminImportRequestsMock).toHaveBeenCalledTimes(2)
  })
})
