import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
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

describe('CapabilityProtectedRoute direct URL scenarios', () => {
    const deniedCapabilityState = {
        loadedTenantId: 'tenant-1',
        isLoading: false,
        capabilities: {
            build: false,
            quarantine_request: false,
            ondemand_image_scanning: false,
        },
    }

    it('shows deterministic denied state for direct /builds access when build capability is disabled', () => {
        useTenantStoreMock.mockReturnValue({ selectedTenantId: 'tenant-1' })
        useOperationCapabilitiesStoreMock.mockReturnValue(deniedCapabilityState)
        useCapabilitySurfacesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-1',
            isLoading: false,
            canAccessRouteKey: () => false,
        })

        render(
            <MemoryRouter initialEntries={['/builds']}>
                <Routes>
                    <Route
                        path="/builds"
                        element={(
                            <CapabilityProtectedRoute capability="build">
                                <div>Builds Page Content</div>
                            </CapabilityProtectedRoute>
                        )}
                    />
                </Routes>
            </MemoryRouter>
        )

        expect(screen.getByText('Capability Not Entitled')).toBeInTheDocument()
        expect(screen.getByText(/not currently entitled for Image Build/i)).toBeInTheDocument()
        expect(screen.queryByText('Builds Page Content')).not.toBeInTheDocument()
    })

    it('shows deterministic denied state for direct /projects/:projectId/builds access when build capability is disabled', () => {
        useTenantStoreMock.mockReturnValue({ selectedTenantId: 'tenant-1' })
        useOperationCapabilitiesStoreMock.mockReturnValue(deniedCapabilityState)
        useCapabilitySurfacesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-1',
            isLoading: false,
            canAccessRouteKey: () => false,
        })

        render(
            <MemoryRouter initialEntries={['/projects/proj-1/builds']}>
                <Routes>
                    <Route
                        path="/projects/:projectId/builds"
                        element={(
                            <CapabilityProtectedRoute capability="build">
                                <div>Project Builds Content</div>
                            </CapabilityProtectedRoute>
                        )}
                    />
                </Routes>
            </MemoryRouter>
        )

        expect(screen.getByText('Capability Not Entitled')).toBeInTheDocument()
        expect(screen.getByText(/not currently entitled for Image Build/i)).toBeInTheDocument()
        expect(screen.queryByText('Project Builds Content')).not.toBeInTheDocument()
    })

    it('renders route content for direct /builds access when build capability is enabled', () => {
        useTenantStoreMock.mockReturnValue({ selectedTenantId: 'tenant-1' })
        useOperationCapabilitiesStoreMock.mockReturnValue({
            ...deniedCapabilityState,
            capabilities: {
                ...deniedCapabilityState.capabilities,
                build: true,
            },
        })
        useCapabilitySurfacesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-1',
            isLoading: false,
            canAccessRouteKey: () => true,
        })

        render(
            <MemoryRouter initialEntries={['/builds']}>
                <Routes>
                    <Route
                        path="/builds"
                        element={(
                            <CapabilityProtectedRoute capability="build">
                                <div>Builds Page Content</div>
                            </CapabilityProtectedRoute>
                        )}
                    />
                </Routes>
            </MemoryRouter>
        )

        expect(screen.getByText('Builds Page Content')).toBeInTheDocument()
        expect(screen.queryByText('Capability Not Entitled')).not.toBeInTheDocument()
    })
})
