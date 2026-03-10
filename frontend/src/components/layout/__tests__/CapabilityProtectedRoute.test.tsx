import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import CapabilityProtectedRoute from '../CapabilityProtectedRoute'

const useTenantStoreMock = vi.fn()
const useOperationCapabilitiesStoreMock = vi.fn()
const useCapabilitySurfacesStoreMock = vi.fn()

vi.mock('@/store/tenant', () => ({
    useTenantStore: (selector?: any) => {
        const state = useTenantStoreMock()
        return typeof selector === 'function' ? selector(state) : state
    },
}))

vi.mock('@/store/operationCapabilities', () => ({
    useOperationCapabilitiesStore: (selector?: any) => {
        const state = useOperationCapabilitiesStoreMock()
        return typeof selector === 'function' ? selector(state) : state
    },
}))

vi.mock('@/store/capabilitySurfaces', () => ({
    useCapabilitySurfacesStore: (selector?: any) => {
        const state = useCapabilitySurfacesStoreMock()
        return typeof selector === 'function' ? selector(state) : state
    },
}))

describe('CapabilityProtectedRoute', () => {
    it('renders children when capability is enabled', () => {
        useTenantStoreMock.mockReturnValue({ selectedTenantId: 'tenant-1' })
        useOperationCapabilitiesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-1',
            isLoading: false,
            capabilities: {
                build: true,
                quarantine_request: false,
                ondemand_image_scanning: false,
            },
        })
        useCapabilitySurfacesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-1',
            isLoading: false,
            canAccessRouteKey: () => true,
        })

        render(
            <MemoryRouter>
                <CapabilityProtectedRoute capability="build">
                    <div>Protected Content</div>
                </CapabilityProtectedRoute>
            </MemoryRouter>
        )

        expect(screen.getByText('Protected Content')).toBeInTheDocument()
    })

    it('shows denied state when capability is disabled', () => {
        useTenantStoreMock.mockReturnValue({ selectedTenantId: 'tenant-1' })
        useOperationCapabilitiesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-1',
            isLoading: false,
            capabilities: {
                build: false,
                quarantine_request: false,
                ondemand_image_scanning: false,
            },
        })
        useCapabilitySurfacesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-1',
            isLoading: false,
            canAccessRouteKey: () => false,
        })

        render(
            <MemoryRouter>
                <CapabilityProtectedRoute capability="build">
                    <div>Protected Content</div>
                </CapabilityProtectedRoute>
            </MemoryRouter>
        )

        expect(screen.getByText('Capability Not Entitled')).toBeInTheDocument()
        expect(screen.getByText(/not currently entitled for Image Build/i)).toBeInTheDocument()
        expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
    })

    it('shows pending state while capability data is loading for selected tenant', () => {
        useTenantStoreMock.mockReturnValue({ selectedTenantId: 'tenant-1' })
        useOperationCapabilitiesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-2',
            isLoading: true,
            capabilities: {
                build: true,
                quarantine_request: true,
                ondemand_image_scanning: true,
            },
        })
        useCapabilitySurfacesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-2',
            isLoading: true,
            canAccessRouteKey: () => true,
        })

        render(
            <MemoryRouter>
                <CapabilityProtectedRoute capability="quarantine_request">
                    <div>Protected Content</div>
                </CapabilityProtectedRoute>
            </MemoryRouter>
        )

        expect(screen.getByText('Checking tenant capabilities...')).toBeInTheDocument()
        expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
    })
})
