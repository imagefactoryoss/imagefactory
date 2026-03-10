import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import QuarantineRequestsPage from '../QuarantineRequestsPage'

const listImportRequestsMock = vi.fn()
const retryImportRequestMock = vi.fn()
const withdrawImportRequestMock = vi.fn()
const listTenantSORRequestsMock = vi.fn()
const createEPRRequestMock = vi.fn()
const withdrawEPRRequestMock = vi.fn()
const confirmDialogMock = vi.fn()
const toastSuccessMock = vi.fn()
const toastErrorMock = vi.fn()

vi.mock('@/services/imageImportService', async () => {
  const actual = await vi.importActual<typeof import('@/services/imageImportService')>('@/services/imageImportService')
  return {
    ...actual,
    imageImportService: {
      listImportRequests: (...args: any[]) => listImportRequestsMock(...args),
      retryImportRequest: (...args: any[]) => retryImportRequestMock(...args),
      withdrawImportRequest: (...args: any[]) => withdrawImportRequestMock(...args),
    },
  }
})

vi.mock('@/components/images/ExternalImportRequestForm', () => ({
  __esModule: true,
  default: ({ initialValues }: any) => (
    <div>
      <div>ExternalImportRequestForm</div>
      <div data-testid="import-form-initial-values">{JSON.stringify(initialValues || {})}</div>
    </div>
  ),
}))

vi.mock('@/services/eprRegistrationService', () => ({
  eprRegistrationService: {
    listTenantRequests: (...args: any[]) => listTenantSORRequestsMock(...args),
    createRequest: (...args: any[]) => createEPRRequestMock(...args),
    withdrawRequest: (...args: any[]) => withdrawEPRRequestMock(...args),
  },
}))

vi.mock('@/context/ConfirmDialogContext', () => ({
  useConfirmDialog: () => confirmDialogMock,
}))

vi.mock('react-hot-toast', () => ({
  __esModule: true,
  default: {
    success: (...args: any[]) => toastSuccessMock(...args),
    error: (...args: any[]) => toastErrorMock(...args),
  },
}))

describe('QuarantineRequestsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    confirmDialogMock.mockResolvedValue(true)
    listImportRequestsMock.mockResolvedValue([
      {
        id: 'req-1',
        tenant_id: 'tenant-1',
        requested_by_user_id: 'user-1',
        epr_record_id: 'EPR-1',
        source_registry: 'ghcr.io',
        source_image_ref: 'ghcr.io/acme/app:1.0.0',
        status: 'failed',
        retryable: true,
        sync_state: 'failed',
        error_message: 'pipeline timeout',
        created_at: '2026-02-20T00:00:00Z',
        updated_at: '2026-02-20T00:05:00Z',
      },
    ])
    retryImportRequestMock.mockResolvedValue({
      id: 'req-2',
      tenant_id: 'tenant-1',
      requested_by_user_id: 'user-1',
      epr_record_id: 'EPR-1',
      source_registry: 'ghcr.io',
      source_image_ref: 'ghcr.io/acme/app:1.0.0',
      status: 'pending',
      retryable: true,
      sync_state: 'awaiting_dispatch',
      created_at: '2026-02-20T00:10:00Z',
      updated_at: '2026-02-20T00:10:00Z',
    })
    withdrawImportRequestMock.mockResolvedValue({
      id: 'req-1',
      tenant_id: 'tenant-1',
      requested_by_user_id: 'user-1',
      epr_record_id: 'EPR-1',
      source_registry: 'ghcr.io',
      source_image_ref: 'ghcr.io/acme/app:1.0.0',
      status: 'failed',
      retryable: false,
      sync_state: 'failed',
      error_message: 'Withdrawn: Withdrawn by tenant user',
      created_at: '2026-02-20T00:00:00Z',
      updated_at: '2026-02-20T00:05:00Z',
    })
    listTenantSORRequestsMock.mockResolvedValue([])
    createEPRRequestMock.mockResolvedValue({
      id: 'epr-1',
      tenant_id: 'tenant-1',
      epr_record_id: 'EPR-1',
      product_name: 'Prod',
      technology_name: 'Tech',
      requested_by_user_id: 'user-1',
      status: 'pending',
      created_at: '2026-02-20T00:00:00Z',
      updated_at: '2026-02-20T00:00:00Z',
    })
    withdrawEPRRequestMock.mockResolvedValue({
      id: 'epr-1',
      tenant_id: 'tenant-1',
      epr_record_id: 'EPR-1',
      product_name: 'Prod',
      technology_name: 'Tech',
      requested_by_user_id: 'user-1',
      status: 'withdrawn',
      decision_reason: 'Withdrawn by tenant user',
      created_at: '2026-02-20T00:00:00Z',
      updated_at: '2026-02-20T00:00:00Z',
    })
  })

  it('opens details drawer for a request', async () => {
    render(
      <MemoryRouter>
        <QuarantineRequestsPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('ghcr.io/acme/app:1.0.0')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Details' }))

    expect(await screen.findByText('Quarantine Request Detail')).toBeInTheDocument()
    expect(screen.getByText(/Timeline/i)).toBeInTheDocument()
    expect(screen.getByText(/Request Data/i)).toBeInTheDocument()
  })

  it('retries a retryable request and refreshes card state', async () => {
    render(
      <MemoryRouter>
        <QuarantineRequestsPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('ghcr.io/acme/app:1.0.0')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))

    await waitFor(() => {
      expect(confirmDialogMock).toHaveBeenCalled()
      expect(retryImportRequestMock).toHaveBeenCalledWith('req-1')
      expect(toastSuccessMock).toHaveBeenCalledWith('Retry submitted successfully')
    })
  })

  it('shows error toast when retry fails', async () => {
    retryImportRequestMock.mockRejectedValueOnce(new Error('retry failed'))
    render(
      <MemoryRouter>
        <QuarantineRequestsPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('ghcr.io/acme/app:1.0.0')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))

    await waitFor(() => {
      expect(retryImportRequestMock).toHaveBeenCalledWith('req-1')
      expect(toastErrorMock).toHaveBeenCalledWith('retry failed')
    })
  })

  it('does not retry when confirmation is cancelled', async () => {
    confirmDialogMock.mockResolvedValueOnce(false)
    render(
      <MemoryRouter>
        <QuarantineRequestsPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('ghcr.io/acme/app:1.0.0')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))

    await waitFor(() => {
      expect(confirmDialogMock).toHaveBeenCalled()
    })
    expect(retryImportRequestMock).not.toHaveBeenCalled()
    expect(toastSuccessMock).not.toHaveBeenCalled()
    expect(toastErrorMock).not.toHaveBeenCalled()
  })

  it('shows entitlement guidance when list API is denied by capability', async () => {
    const { ImageImportApiError } = await import('@/services/imageImportService')
    listImportRequestsMock.mockRejectedValueOnce(
      new ImageImportApiError({
        code: 'tenant_capability_not_entitled',
        message: 'denied',
        status: 403,
      })
    )

    render(
      <MemoryRouter>
        <QuarantineRequestsPage />
      </MemoryRouter>
    )

    expect(await screen.findByText(/not entitled for quarantine requests/i)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /capability matrix/i })).toHaveAttribute('href', '/help/capability-access')
    expect(screen.queryByText('ExternalImportRequestForm')).not.toBeInTheDocument()
  })

  it('maps import_not_retryable to actionable retry message', async () => {
    const { ImageImportApiError } = await import('@/services/imageImportService')
    retryImportRequestMock.mockRejectedValueOnce(
      new ImageImportApiError({
        code: 'import_not_retryable',
        message: 'cannot retry',
        status: 409,
      })
    )
    render(
      <MemoryRouter>
        <QuarantineRequestsPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('ghcr.io/acme/app:1.0.0')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))

    await waitFor(() => {
      expect(toastErrorMock).toHaveBeenCalledWith(
        'This request is not currently retryable. Refresh to confirm latest status or create a new request.'
      )
    })
  })

  it('shows dispatch_failed sync state and diagnostics in details', async () => {
    listImportRequestsMock.mockResolvedValueOnce([
      {
        id: 'req-3',
        tenant_id: 'tenant-1',
        requested_by_user_id: 'user-1',
        epr_record_id: 'EPR-3',
        source_registry: 'ghcr.io',
        source_image_ref: 'ghcr.io/acme/app:3.0.0',
        status: 'failed',
        retryable: true,
        sync_state: 'dispatch_failed',
        error_message: 'dispatch_failed: context deadline exceeded',
        created_at: '2026-02-20T00:00:00Z',
        updated_at: '2026-02-20T00:05:00Z',
      },
    ])

    render(
      <MemoryRouter>
        <QuarantineRequestsPage />
      </MemoryRouter>
    )

    expect((await screen.findAllByText('Dispatch Failed')).length).toBeGreaterThan(0)
    fireEvent.click(screen.getByRole('button', { name: 'Details' }))

    expect(await screen.findByText('Operational Diagnostics')).toBeInTheDocument()
    expect(screen.getAllByText('Dispatch Failed').length).toBeGreaterThan(0)
    expect(screen.getAllByText(/context deadline exceeded/i).length).toBeGreaterThan(0)
  })

  it('shows EPR empty state with create action', async () => {
    listImportRequestsMock.mockResolvedValueOnce([])

    render(
      <MemoryRouter>
        <QuarantineRequestsPage />
      </MemoryRouter>
    )

    expect(await screen.findByText(/No EPR registration requests yet/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Create EPR Registration/i })).toBeInTheDocument()
  })

  it('opens create quarantine request drawer from empty state', async () => {
    listImportRequestsMock.mockResolvedValueOnce([])

    render(
      <MemoryRouter>
        <QuarantineRequestsPage />
      </MemoryRouter>
    )

    expect(await screen.findByText(/No quarantine requests yet/i)).toBeInTheDocument()
    fireEvent.click(screen.getAllByRole('button', { name: /Create Quarantine Request/i })[0])
    expect(await screen.findByText('ExternalImportRequestForm')).toBeInTheDocument()
  })

  it('submits EPR registration from drawer and reloads EPR list', async () => {
    listImportRequestsMock.mockResolvedValueOnce([])
    listTenantSORRequestsMock.mockResolvedValueOnce([
      {
        id: 'epr-existing-1',
        tenant_id: 'tenant-1',
        epr_record_id: 'EPR-777',
        product_name: 'Existing Product',
        technology_name: 'Existing Tech',
        requested_by_user_id: 'user-1',
        status: 'approved',
        created_at: '2026-03-01T00:00:00Z',
        updated_at: '2026-03-01T00:00:00Z',
      },
    ])

    render(
      <MemoryRouter>
        <QuarantineRequestsPage />
      </MemoryRouter>
    )

    fireEvent.click(await screen.findByRole('button', { name: /Create EPR Registration/i }))
    fireEvent.change(await screen.findByPlaceholderText(/Product Name/i), { target: { value: 'Prod' } })
    fireEvent.change(screen.getByPlaceholderText(/Technology Name/i), { target: { value: 'Tech' } })
    fireEvent.click(screen.getByRole('button', { name: /Submit EPR Registration/i }))

    await waitFor(() => {
      expect(createEPRRequestMock).toHaveBeenCalled()
      expect(toastSuccessMock).toHaveBeenCalledWith('EPR registration request submitted for security review')
    })
  })

  it('prefills generated EPR record ID in registration drawer', async () => {
    render(
      <MemoryRouter>
        <QuarantineRequestsPage />
      </MemoryRouter>
    )

    fireEvent.click(await screen.findByRole('button', { name: /Create EPR Registration/i }))
    const input = await screen.findByPlaceholderText(/EPR Record ID/i)
    expect((input as HTMLInputElement).value).toMatch(/^EPR-\d{8}-[A-Z0-9]{8}$/)
  })

  it('opens clone import drawer with prefilled values', async () => {
    render(
      <MemoryRouter>
        <QuarantineRequestsPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('ghcr.io/acme/app:1.0.0')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Clone' }))
    expect(await screen.findByText('ExternalImportRequestForm')).toBeInTheDocument()
    expect(screen.getByTestId('import-form-initial-values').textContent).toContain('"eprRecordId":"EPR-1"')
    expect(screen.getByTestId('import-form-initial-values').textContent).toContain('"sourceRegistry":"ghcr.io"')
  })

  it('withdraws pending import request', async () => {
    listImportRequestsMock.mockResolvedValueOnce([
      {
        id: 'req-pending-1',
        tenant_id: 'tenant-1',
        requested_by_user_id: 'user-1',
        epr_record_id: 'EPR-1',
        source_registry: 'ghcr.io',
        source_image_ref: 'ghcr.io/acme/app:1.0.0',
        status: 'pending',
        retryable: false,
        sync_state: 'awaiting_dispatch',
        created_at: '2026-02-20T00:00:00Z',
        updated_at: '2026-02-20T00:05:00Z',
      },
    ])
    withdrawImportRequestMock.mockResolvedValueOnce({
      id: 'req-pending-1',
      tenant_id: 'tenant-1',
      requested_by_user_id: 'user-1',
      epr_record_id: 'EPR-1',
      source_registry: 'ghcr.io',
      source_image_ref: 'ghcr.io/acme/app:1.0.0',
      status: 'failed',
      retryable: false,
      sync_state: 'failed',
      error_message: 'Withdrawn: Withdrawn by tenant user',
      created_at: '2026-02-20T00:00:00Z',
      updated_at: '2026-02-20T00:06:00Z',
    })

    render(
      <MemoryRouter>
        <QuarantineRequestsPage />
      </MemoryRouter>
    )

    expect(await screen.findByRole('button', { name: 'Withdraw' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Withdraw' }))

    await waitFor(() => {
      expect(withdrawImportRequestMock).toHaveBeenCalledWith('req-pending-1', 'Withdrawn by tenant user')
      expect(toastSuccessMock).toHaveBeenCalledWith('Quarantine request withdrawn')
    })
  })

  it('does not show Retry action for pending requests even when retryable=true', async () => {
    listImportRequestsMock.mockResolvedValueOnce([
      {
        id: 'req-pending-2',
        tenant_id: 'tenant-1',
        requested_by_user_id: 'user-1',
        epr_record_id: 'EPR-1',
        source_registry: 'ghcr.io',
        source_image_ref: 'ghcr.io/acme/app:2.0.0',
        status: 'pending',
        retryable: true,
        sync_state: 'awaiting_dispatch',
        created_at: '2026-02-20T00:00:00Z',
        updated_at: '2026-02-20T00:05:00Z',
      },
    ])

    render(
      <MemoryRouter>
        <QuarantineRequestsPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('ghcr.io/acme/app:2.0.0')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Retry' })).not.toBeInTheDocument()
  })
})
