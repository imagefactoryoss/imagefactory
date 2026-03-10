import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import ImageDetailPage from '../ImageDetailPage'

const getImageDetailsMock = vi.fn()
const triggerOnDemandScanMock = vi.fn()
const getTenantByIdMock = vi.fn()
const useTenantStoreMock = vi.fn()
const useCapabilitySurfacesStoreMock = vi.fn()

vi.mock('@/context/ConfirmDialogContext', () => ({
    useConfirmDialog: () => vi.fn().mockResolvedValue(true),
}))

vi.mock('@/services/imageService', async () => {
    const actual = await vi.importActual<typeof import('@/services/imageService')>('@/services/imageService')
    return {
        ...actual,
        imageService: {
            ...actual.imageService,
            getImageDetails: (...args: any[]) => getImageDetailsMock(...args),
            triggerOnDemandScan: (...args: any[]) => triggerOnDemandScanMock(...args),
        },
    }
})

vi.mock('@/services/adminService', () => ({
    adminService: {
        getTenantById: (...args: any[]) => getTenantByIdMock(...args),
    },
}))

vi.mock('@/store/tenant', () => ({
    useTenantStore: (selector?: any) => {
        const state = useTenantStoreMock()
        return typeof selector === 'function' ? selector(state) : state
    },
}))

vi.mock('@/store/capabilitySurfaces', () => ({
    useCapabilitySurfacesStore: (selector?: any) => {
        const state = useCapabilitySurfacesStoreMock()
        return typeof selector === 'function' ? selector(state) : state
    },
}))

const buildDetailsFixture = () =>
    ({
        image: {
            id: 'img-1',
            tenant_id: 'tenant-1',
            name: 'nginx',
            description: 'Nginx image',
            visibility: 'tenant',
            status: 'published',
            tags: ['latest'],
            metadata: {},
            pull_count: 0,
            created_by: 'user-1',
            updated_by: 'user-1',
            created_at: '2026-02-20T00:00:00Z',
            updated_at: '2026-02-20T00:00:00Z',
        },
        versions: [],
        tags: {
            inline: [],
            normalized: [],
            merged: ['latest'],
        },
        vulnerability_scans: [],
        sbom: {
            id: 'sbom-1',
            image_id: 'img-1',
            sbom_format: 'spdx-json',
            tool: 'syft',
            tool_version: '1.0.0',
            generated_at: '2026-02-20T00:00:00Z',
            total_packages: 0,
            sbom_data: {},
            packages: [],
        },
        layers: [],
        metadata: {
            layers_evidence_status: 'fresh',
            layers_evidence_updated_at: '2026-02-20T00:00:00Z',
            sbom_evidence_status: 'fresh',
            sbom_evidence_updated_at: '2026-02-20T00:00:00Z',
            vulnerability_evidence_status: 'fresh',
            vulnerability_evidence_updated_at: '2026-02-20T00:00:00Z',
            scan_tool: 'trivy',
        },
        stats: {
            total_versions: 0,
            total_downloads: 0,
            recent_downloads: 0,
            latest_scan_status: 'unknown',
            security_score: 0,
            last_updated: '2026-02-20T00:00:00Z',
            vulnerability_scan_count: 0,
            passed_scan_count: 0,
            failed_scan_count: 0,
            sbom_count: 1,
        },
    } as any)

describe('ImageDetailPage on-demand scan capability UX', () => {
    beforeEach(() => {
        vi.clearAllMocks()
        getImageDetailsMock.mockResolvedValue(buildDetailsFixture())
        getTenantByIdMock.mockReturnValue(new Promise(() => {}))
        useTenantStoreMock.mockReturnValue({
            selectedTenantId: 'tenant-1',
            userTenants: [{ id: 'tenant-1', name: 'Tenant One' }],
        })
    })

    const renderPage = () =>
        render(
            <MemoryRouter initialEntries={['/images/img-1']}>
                <Routes>
                    <Route path="/images/:imageId" element={<ImageDetailPage />} />
                </Routes>
            </MemoryRouter>
        )

    it('shows non-entitled label when on-demand scanning capability is disabled', async () => {
        useCapabilitySurfacesStoreMock.mockReturnValue({
            canRunActionKey: () => false,
        })

        renderPage()
        fireEvent.click(await screen.findByRole('button', { name: /security/i }))

        expect(await screen.findByText('On-demand scanning not entitled')).toBeInTheDocument()
        expect(screen.queryByRole('button', { name: /run on-demand scan/i })).not.toBeInTheDocument()
    })

    it('shows actionable denied message when API returns tenant capability denial', async () => {
        useCapabilitySurfacesStoreMock.mockReturnValue({
            canRunActionKey: () => true,
        })
        triggerOnDemandScanMock.mockRejectedValue({
            response: {
                data: {
                    error: {
                        code: 'tenant_capability_not_entitled',
                    },
                },
            },
        })

        renderPage()
        fireEvent.click(await screen.findByRole('button', { name: /security/i }))
        fireEvent.click(await screen.findByRole('button', { name: /run on-demand scan/i }))

        await waitFor(() => {
            expect(screen.getByText('On-demand image scanning is not entitled for this tenant.')).toBeInTheDocument()
            expect(triggerOnDemandScanMock).toHaveBeenCalledWith('img-1')
        })
    })

    it('shows actionable not-found message when scan trigger target is missing', async () => {
        useCapabilitySurfacesStoreMock.mockReturnValue({
            canRunActionKey: () => true,
        })
        triggerOnDemandScanMock.mockRejectedValue({
            response: {
                data: {
                    error: {
                        code: 'not_found',
                    },
                },
            },
        })

        renderPage()
        fireEvent.click(await screen.findByRole('button', { name: /security/i }))
        fireEvent.click(await screen.findByRole('button', { name: /run on-demand scan/i }))

        await waitFor(() => {
            expect(screen.getByText('Image was not found for the current tenant context. Refresh and retry.')).toBeInTheDocument()
            expect(triggerOnDemandScanMock).toHaveBeenCalledWith('img-1')
        })
    })

    it('shows success confirmation when on-demand scan request is accepted', async () => {
        useCapabilitySurfacesStoreMock.mockReturnValue({
            canRunActionKey: () => true,
        })
        triggerOnDemandScanMock.mockResolvedValue({
            scan_request_id: 'scan-1',
            image_id: 'img-1',
            status: 'queued',
            message: 'On-demand scan queued successfully.',
        })

        renderPage()
        fireEvent.click(await screen.findByRole('button', { name: /security/i }))
        fireEvent.click(await screen.findByRole('button', { name: /run on-demand scan/i }))

        await waitFor(() => {
            expect(screen.getByText('On-demand scan queued successfully.')).toBeInTheDocument()
            expect(triggerOnDemandScanMock).toHaveBeenCalledWith('img-1')
        })
    })
})
