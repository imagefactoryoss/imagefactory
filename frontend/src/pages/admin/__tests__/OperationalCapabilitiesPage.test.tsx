import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OperationalCapabilitiesPage from '../OperationalCapabilitiesPage'

const getTenantsMock = vi.fn()
const getAdminCapabilitiesMock = vi.fn()
const updateAdminCapabilitiesMock = vi.fn()
const toastSuccessMock = vi.fn()
const toastErrorMock = vi.fn()
const canManageAdminMock = vi.fn()

vi.mock('@/services/adminService', () => ({
    adminService: {
        getTenants: (...args: any[]) => getTenantsMock(...args),
    },
}))

vi.mock('@/services/operationCapabilityService', () => ({
    operationCapabilityService: {
        defaultConfig: () => ({
            build: false,
            quarantine_request: false,
            quarantine_release: false,
            ondemand_image_scanning: false,
        }),
        getAdminCapabilities: (...args: any[]) => getAdminCapabilitiesMock(...args),
        updateAdminCapabilities: (...args: any[]) => updateAdminCapabilitiesMock(...args),
    },
}))

vi.mock('react-hot-toast', () => ({
    __esModule: true,
    default: {
        success: (...args: any[]) => toastSuccessMock(...args),
        error: (...args: any[]) => toastErrorMock(...args),
    },
}))

vi.mock('@/hooks/useAccess', () => ({
    useCanManageAdmin: () => canManageAdminMock(),
}))

describe('OperationalCapabilitiesPage', () => {
    beforeEach(() => {
        vi.clearAllMocks()
        canManageAdminMock.mockReturnValue(true)
        getTenantsMock.mockResolvedValue({
            data: [
                { id: 'tenant-1', name: 'Tenant One', slug: 'tenant-one', status: 'active' },
                { id: 'tenant-2', name: 'Tenant Two', slug: 'tenant-two', status: 'active' },
            ],
            pagination: { page: 1, limit: 20, total: 2, totalPages: 1 },
        })
        getAdminCapabilitiesMock.mockResolvedValue({
            build: true,
            quarantine_request: true,
            quarantine_release: false,
            ondemand_image_scanning: false,
        })
        updateAdminCapabilitiesMock.mockResolvedValue({
            build: false,
            quarantine_request: true,
            quarantine_release: false,
            ondemand_image_scanning: false,
        })
    })

    it('loads tenants and opens drawer to fetch capabilities', async () => {
        render(<OperationalCapabilitiesPage />)

        await waitFor(() => {
            expect(getTenantsMock).toHaveBeenCalledTimes(1)
            expect(getTenantsMock).toHaveBeenCalledWith({
                page: 1,
                limit: 20,
                search: undefined,
                status: undefined,
            })
        })

        fireEvent.click(await screen.findAllByRole('button', { name: /edit capabilities/i }).then((buttons) => buttons[0]))

        await waitFor(() => {
            expect(getAdminCapabilitiesMock).toHaveBeenCalledWith({ tenantId: 'tenant-1' })
        })

        expect(await screen.findByRole('checkbox', { name: /image build/i })).toBeInTheDocument()
    })

    it('loads capabilities for the selected tenant row', async () => {
        render(<OperationalCapabilitiesPage />)

        await waitFor(() => {
            expect(getTenantsMock).toHaveBeenCalled()
        })

        fireEvent.click(await screen.findAllByRole('button', { name: /edit capabilities/i }).then((buttons) => buttons[1]))

        await waitFor(() => {
            expect(getAdminCapabilitiesMock).toHaveBeenCalledWith({ tenantId: 'tenant-2' })
        })
    })

    it('requires entitlement change reason when capability values changed', async () => {
        render(<OperationalCapabilitiesPage />)

        await waitFor(() => {
            expect(getTenantsMock).toHaveBeenCalled()
        })

        fireEvent.click(await screen.findAllByRole('button', { name: /edit capabilities/i }).then((buttons) => buttons[0]))
        await waitFor(() => {
            expect(getAdminCapabilitiesMock).toHaveBeenCalled()
        })
        fireEvent.click(await screen.findByRole('checkbox', { name: /image build/i }))
        fireEvent.change(screen.getByPlaceholderText(/reason for capability entitlement changes/i), { target: { value: '' } })
        fireEvent.click(screen.getByRole('button', { name: /save capabilities/i }))

        expect(await screen.findByText(/entitlement change reason is required/i)).toBeInTheDocument()
        expect(updateAdminCapabilitiesMock).not.toHaveBeenCalled()
    })

    it('saves updated capabilities with tenant id and reason', async () => {
        render(<OperationalCapabilitiesPage />)

        await waitFor(() => {
            expect(getTenantsMock).toHaveBeenCalled()
        })

        fireEvent.click(await screen.findAllByRole('button', { name: /edit capabilities/i }).then((buttons) => buttons[0]))
        await waitFor(() => {
            expect(getAdminCapabilitiesMock).toHaveBeenCalled()
        })
        await waitFor(() => {
            expect((screen.getByRole('checkbox', { name: /image build/i }) as HTMLInputElement).checked).toBe(true)
        })
        fireEvent.click(await screen.findByRole('checkbox', { name: /image build/i }))
        fireEvent.change(screen.getByPlaceholderText(/reason for capability entitlement changes/i), { target: { value: 'Quarterly policy review update' } })
        fireEvent.click(screen.getByRole('button', { name: /save capabilities/i }))

        await waitFor(() => {
            expect(updateAdminCapabilitiesMock).toHaveBeenCalledWith(
                {
                    build: false,
                    quarantine_request: true,
                    quarantine_release: false,
                    ondemand_image_scanning: false,
                },
                {
                    tenantId: 'tenant-1',
                    changeReason: 'Quarterly policy review update',
                }
            )
            expect(toastSuccessMock).toHaveBeenCalled()
        })
    })

    it('applies bulk capability update for selected tenants', async () => {
        render(<OperationalCapabilitiesPage />)

        await waitFor(() => {
            expect(getTenantsMock).toHaveBeenCalled()
        })

        await screen.findAllByRole('button', { name: /edit capabilities/i })
        fireEvent.click(screen.getByRole('checkbox', { name: /select tenant one/i }))
        fireEvent.change(screen.getByRole('combobox', { name: /action/i }), { target: { value: 'disable' } })
        fireEvent.change(screen.getByPlaceholderText(/reason for bulk capability changes/i), {
            target: { value: 'Bulk policy adjustment' },
        })
        fireEvent.click(screen.getByRole('button', { name: /apply bulk update/i }))

        await waitFor(() => {
            expect(getAdminCapabilitiesMock).toHaveBeenCalledWith({ tenantId: 'tenant-1' })
            expect(updateAdminCapabilitiesMock).toHaveBeenCalledWith(
                {
                    build: true,
                    quarantine_request: false,
                    quarantine_release: false,
                    ondemand_image_scanning: false,
                },
                {
                    tenantId: 'tenant-1',
                    changeReason: 'Bulk policy adjustment',
                }
            )
            expect(toastSuccessMock).toHaveBeenCalled()
        })
    })

    it('caps bulk selection at 50 tenants', async () => {
        getTenantsMock.mockResolvedValueOnce({
            data: Array.from({ length: 60 }, (_, index) => ({
                id: `tenant-${index + 1}`,
                name: `Tenant ${index + 1}`,
                slug: `tenant-${index + 1}`,
                status: 'active',
            })),
            pagination: { page: 1, limit: 20, total: 60, totalPages: 3 },
        })

        render(<OperationalCapabilitiesPage />)

        await screen.findAllByRole('button', { name: /edit capabilities/i })
        fireEvent.click(screen.getByRole('checkbox', { name: /select all tenants on this page/i }))

        expect(screen.getByText(/50 tenant\(s\) selected/i)).toBeInTheDocument()
        expect(toastErrorMock).toHaveBeenCalledWith('Bulk updates support up to 50 tenants per run.')
    })

    it('hides tenant edit and bulk actions in read-only mode', async () => {
        canManageAdminMock.mockReturnValue(false)
        render(<OperationalCapabilitiesPage />)

        await waitFor(() => {
            expect(getTenantsMock).toHaveBeenCalled()
        })

        expect(
            screen.getByText(/Read-only mode: capability entitlement edit actions are hidden/i)
        ).toBeInTheDocument()
        expect(screen.queryByRole('button', { name: /edit capabilities/i })).not.toBeInTheDocument()
        expect(screen.queryByRole('button', { name: /apply bulk update/i })).not.toBeInTheDocument()
    })
})
