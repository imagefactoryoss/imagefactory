import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import ExternalImportRequestForm from '../ExternalImportRequestForm'

const createImportRequestMock = vi.fn()
const apiGetMock = vi.fn()
const listRegistryAuthMock = vi.fn()
const listTenantEprRequestsMock = vi.fn()
const paragraphWithText = (text: string) => (_: string, element: Element | null) =>
    element?.tagName === 'P' && (element.textContent || '').includes(text)

vi.mock('@/services/imageImportService', async () => {
    const actual = await vi.importActual<typeof import('@/services/imageImportService')>('@/services/imageImportService')
    return {
        ...actual,
        imageImportService: {
            createImportRequest: (...args: any[]) => createImportRequestMock(...args),
        },
    }
})

vi.mock('@/services/api', () => ({
    __esModule: true,
    default: {
        get: (...args: any[]) => apiGetMock(...args),
    },
}))

vi.mock('@/api/registryAuthClient', () => ({
    registryAuthClient: {
        listRegistryAuth: (...args: any[]) => listRegistryAuthMock(...args),
    },
}))

vi.mock('@/services/eprRegistrationService', () => ({
    eprRegistrationService: {
        listTenantRequests: (...args: any[]) => listTenantEprRequestsMock(...args),
    },
}))

describe('ExternalImportRequestForm', () => {
    const renderForm = (onCreated = vi.fn()) =>
        render(
            <MemoryRouter>
                <ExternalImportRequestForm onCreated={onCreated} />
            </MemoryRouter>
        )

    const unlockFormWithValidEpr = async (eprID = 'EPR-123') => {
        fireEvent.change(screen.getByLabelText(/epr record id/i), { target: { value: eprID } })
        await waitFor(() => {
            expect(screen.getByLabelText(/source registry/i)).not.toBeDisabled()
            expect(screen.getByLabelText(/source image ref/i)).not.toBeDisabled()
        })
    }

    beforeEach(() => {
        vi.clearAllMocks()
        apiGetMock.mockResolvedValue({ data: { runtime_error_mode: 'error' } })
        listRegistryAuthMock.mockResolvedValue({ registry_auth: [], total_count: 0 })
        listTenantEprRequestsMock.mockResolvedValue([
            {
                id: 'epr-1',
                tenant_id: 'tenant-1',
                epr_record_id: 'EPR-123',
                product_name: 'Product A',
                technology_name: 'Tech A',
                requested_by_user_id: 'user-1',
                status: 'approved',
                created_at: '2026-02-20T00:00:00Z',
                updated_at: '2026-02-20T00:00:00Z',
            },
        ])
    })

    it('keeps submit disabled until a valid approved EPR id is selected and required fields are filled', async () => {
        renderForm()

        expect(screen.getByRole('button', { name: /create request/i })).toBeDisabled()
        await unlockFormWithValidEpr()
        expect(screen.getByRole('button', { name: /create request/i })).toBeDisabled()
        fireEvent.change(screen.getByLabelText(/source registry/i), { target: { value: 'registry-1.docker.io' } })
        fireEvent.change(screen.getByLabelText(/source image ref/i), { target: { value: 'library/nginx:latest' } })
        expect(screen.getByRole('button', { name: /create request/i })).not.toBeDisabled()
    })

    it('keeps non-EPR fields disabled until a valid approved EPR id is entered', async () => {
        renderForm()

        const sourceRegistryInput = await screen.findByLabelText(/source registry/i)
        const sourceImageInput = screen.getByLabelText(/source image ref/i)
        expect(sourceRegistryInput).toBeDisabled()
        expect(sourceImageInput).toBeDisabled()

        fireEvent.change(screen.getByLabelText(/epr record id/i), { target: { value: 'EPR-DOES-NOT-EXIST' } })
        expect(await screen.findByText(/EPR ID not found in approved records/i)).toBeInTheDocument()
        expect(sourceRegistryInput).toBeDisabled()
        expect(sourceImageInput).toBeDisabled()

        fireEvent.change(screen.getByLabelText(/epr record id/i), { target: { value: 'EPR-123' } })
        await waitFor(() => {
            expect(sourceRegistryInput).not.toBeDisabled()
            expect(sourceImageInput).not.toBeDisabled()
        })
    })

    it('maps epr_registration_required to actionable message', async () => {
        const { ImageImportApiError } = await import('@/services/imageImportService')
        createImportRequestMock.mockRejectedValue(
            new ImageImportApiError({
                code: 'epr_registration_required',
                message: 'enterprise EPR registration is required',
                status: 412,
            })
        )

        renderForm()

        await unlockFormWithValidEpr('EPR-123')
        fireEvent.change(screen.getByLabelText(/source registry/i), { target: { value: 'registry-1.docker.io' } })
        fireEvent.change(screen.getByLabelText(/source image ref/i), { target: { value: 'library/nginx:latest' } })
        fireEvent.click(screen.getByRole('button', { name: /create request/i }))

        expect(await screen.findByText(/EPR registration is required before requesting quarantine import/i)).toBeInTheDocument()
    })

    it('maps tenant_capability_not_entitled to actionable message', async () => {
        const { ImageImportApiError } = await import('@/services/imageImportService')
        createImportRequestMock.mockRejectedValue(
            new ImageImportApiError({
                code: 'tenant_capability_not_entitled',
                message: 'quarantine capability not entitled',
                status: 403,
            })
        )

        renderForm()

        await unlockFormWithValidEpr('EPR-123')
        fireEvent.change(screen.getByLabelText(/source registry/i), { target: { value: 'registry-1.docker.io' } })
        fireEvent.change(screen.getByLabelText(/source image ref/i), { target: { value: 'library/nginx:latest' } })
        fireEvent.click(screen.getByRole('button', { name: /create request/i }))

        expect(await screen.findByText(/not entitled for quarantine requests/i)).toBeInTheDocument()
        expect(screen.getByText(/Contact your tenant administrator to enable the capability/i)).toBeInTheDocument()
    })

    it('submits create request and calls onCreated', async () => {
        const onCreated = vi.fn()
        createImportRequestMock.mockResolvedValue({
            id: 'import-1',
            tenant_id: 'tenant-1',
            requested_by_user_id: 'user-1',
            epr_record_id: 'EPR-123',
            source_registry: 'registry-1.docker.io',
            source_image_ref: 'library/nginx:latest',
            status: 'pending',
            retryable: true,
            created_at: '2026-02-20T00:00:00Z',
            updated_at: '2026-02-20T00:00:00Z',
        })

        renderForm(onCreated)

        await unlockFormWithValidEpr('EPR-123')
        fireEvent.change(screen.getByLabelText(/source registry/i), { target: { value: 'registry-1.docker.io' } })
        fireEvent.change(screen.getByLabelText(/source image ref/i), { target: { value: 'library/nginx:latest' } })
        fireEvent.click(screen.getByRole('button', { name: /create request/i }))

        await waitFor(() => {
            expect(createImportRequestMock).toHaveBeenCalledWith({
                eprRecordId: 'EPR-123',
                sourceRegistry: 'registry-1.docker.io',
                sourceImageRef: 'library/nginx:latest',
                registryAuthId: undefined,
            })
            expect(onCreated).toHaveBeenCalledTimes(1)
        })
    })

    it('requires registry auth selection when registry is private', async () => {
        renderForm()

        await unlockFormWithValidEpr('EPR-123')
        fireEvent.change(screen.getByLabelText(/source registry/i), { target: { value: 'registry.example.com' } })
        fireEvent.change(screen.getByLabelText(/source image ref/i), { target: { value: 'acme/app:1.0.0' } })
        fireEvent.change(screen.getByLabelText(/registry access/i), { target: { value: 'private' } })
        fireEvent.click(screen.getByRole('button', { name: /create request/i }))

        expect(await screen.findByText(/Registry auth is required for private registries/i)).toBeInTheDocument()
        expect(createImportRequestMock).not.toHaveBeenCalled()
    })

    it('shows EPR runtime policy hint from tenant settings', async () => {
        apiGetMock.mockResolvedValue({ data: { runtime_error_mode: 'allow' } })
        renderForm()

        expect(await screen.findByText(paragraphWithText('EPR runtime policy mode: allow'))).toBeInTheDocument()
        expect(await screen.findByText(paragraphWithText('Effective posture: Permissive'))).toBeInTheDocument()
        expect(await screen.findByText(/can still be admitted/i)).toBeInTheDocument()
        expect(await screen.findByText(/still deny the request when the service responds/i)).toBeInTheDocument()
    })

    it('shows strict deny posture guidance', async () => {
        apiGetMock.mockResolvedValue({ data: { runtime_error_mode: 'deny' } })
        renderForm()

        expect(await screen.findByText(paragraphWithText('EPR runtime policy mode: deny'))).toBeInTheDocument()
        expect(await screen.findByText(paragraphWithText('Effective posture: Strict'))).toBeInTheDocument()
        expect(await screen.findByText(/request is denied as not registered/i)).toBeInTheDocument()
    })

    it('falls back to strict error posture when policy lookup fails', async () => {
        apiGetMock.mockRejectedValue(new Error('network error'))
        renderForm()

        expect(await screen.findByText(paragraphWithText('EPR runtime policy mode: error'))).toBeInTheDocument()
        expect(await screen.findByText(paragraphWithText('Effective posture: Strict'))).toBeInTheDocument()
        expect(await screen.findByText(/showing strict fallback posture/i)).toBeInTheDocument()
    })
})
