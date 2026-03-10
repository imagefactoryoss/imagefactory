import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import Layout from '../Layout'

const useAuthStoreMock = vi.fn()
const useOperationCapabilitiesStoreMock = vi.fn()
const useCapabilitySurfacesStoreMock = vi.fn()
const useTenantStoreMock = vi.fn()
const useThemeStoreMock = vi.fn()

vi.mock('@/context/RefreshContext', () => ({
    useRefresh: () => ({ isRefreshing: false, triggerRefresh: vi.fn() }),
}))

vi.mock('@/context/ConfirmDialogContext', () => ({
    useConfirmDialog: () => vi.fn().mockResolvedValue(true),
}))

vi.mock('@/services/api', () => ({
    api: { get: vi.fn() },
}))

vi.mock('@/services/authService', () => ({
    authService: { logout: vi.fn() },
}))

vi.mock('@/services/profileService', () => ({
    profileService: {
        getInitials: () => 'TA',
        getAvatarColor: () => 'bg-blue-500',
    },
}))

vi.mock('@/store/auth', () => ({
    useAuthStore: (selector?: any) => {
        const state = useAuthStoreMock()
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

vi.mock('@/store/tenant', () => ({
    useTenantStore: (selector?: any) => {
        const state = useTenantStoreMock()
        return typeof selector === 'function' ? selector(state) : state
    },
}))

vi.mock('@/store/theme', () => ({
    useThemeStore: (selector?: any) => {
        const state = useThemeStoreMock()
        return typeof selector === 'function' ? selector(state) : state
    },
}))

vi.mock('@/utils/permissions', () => ({
    canCreateBuilds: () => false,
    canManageMembers: () => false,
    canManageTenants: () => false,
    canViewBuilds: () => true,
    canViewMembers: () => false,
    canViewTenants: () => false,
}))

vi.mock('@/components/auth/PostLoginContextSelector', () => ({
    __esModule: true,
    default: () => null,
}))

vi.mock('@/components/common/ContextSwitcher', () => ({
    __esModule: true,
    default: () => <div>ContextSwitcher</div>,
}))

vi.mock('@/components/notifications/UserNotificationCenter', () => ({
    __esModule: true,
    default: () => <div>UserNotificationCenter</div>,
}))

vi.mock('@/components/onboarding/TenantOwnerWelcomeTour', () => ({
    __esModule: true,
    default: () => null,
}))

vi.mock('@/components/TokenExpirationWarning', () => ({
    TokenExpirationWarning: () => null,
}))

describe('Layout capability-driven navigation', () => {
    beforeEach(() => {
        vi.clearAllMocks()
        window.localStorage.setItem('if_layout_open_nav_section', 'Build & Delivery')

        useAuthStoreMock.mockReturnValue({
            user: { id: 'user-1', name: 'Tenant Admin', email: 'admin@example.com' },
            logout: vi.fn(),
            avatar: null,
            groups: [{ tenant_id: 'tenant-1', role_type: 'tenant_administrator' }],
        })

        useTenantStoreMock.mockReturnValue({
            selectedTenantId: 'tenant-1',
            userTenants: [{ id: 'tenant-1', name: 'Tenant One', roles: [{ name: 'Tenant Admin' }] }],
            getCurrentContext: () => ({ role: { name: 'Tenant Admin' } }),
        })

        useThemeStoreMock.mockReturnValue({
            isDark: false,
            toggleTheme: vi.fn(),
        })

        useCapabilitySurfacesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-1',
            isLoading: false,
            data: {
                surfaces: {
                    nav_keys: ['dashboard', 'notifications', 'images'],
                    route_keys: ['dashboard.view', 'images.list', 'images.detail'],
                    action_keys: ['images.view_catalog'],
                },
            },
        })
    })

    it('shows Builds nav link when build capability is enabled', () => {
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
            data: {
                surfaces: {
                    nav_keys: ['dashboard', 'notifications', 'images', 'projects', 'builds', 'auth_management'],
                    route_keys: ['dashboard.view', 'images.list', 'images.detail', 'projects.list', 'builds.list', 'settings.auth'],
                    action_keys: ['images.view_catalog', 'builds.create', 'settings.auth.manage'],
                },
            },
        })

        render(
            <MemoryRouter>
                <Layout>
                    <div>Page Content</div>
                </Layout>
            </MemoryRouter>
        )

        expect(screen.getByRole('link', { name: /Builds/i })).toBeInTheDocument()
    })

    it('hides Builds nav link when build capability is disabled', () => {
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
            data: {
                surfaces: {
                    nav_keys: ['dashboard', 'notifications', 'images'],
                    route_keys: ['dashboard.view', 'images.list', 'images.detail'],
                    action_keys: ['images.view_catalog'],
                },
            },
        })

        render(
            <MemoryRouter>
                <Layout>
                    <div>Page Content</div>
                </Layout>
            </MemoryRouter>
        )

        expect(screen.queryByRole('link', { name: /Builds/i })).not.toBeInTheDocument()
        expect(screen.queryByRole('link', { name: /Projects/i })).not.toBeInTheDocument()
        expect(screen.queryByRole('link', { name: /Registry Auth/i })).not.toBeInTheDocument()
    })

    it('hides Builds nav link when capability state is stale for another tenant', () => {
        useOperationCapabilitiesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-2',
            isLoading: false,
            capabilities: {
                build: true,
                quarantine_request: false,
                ondemand_image_scanning: false,
            },
        })
        useCapabilitySurfacesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-2',
            isLoading: false,
            data: {
                surfaces: {
                    nav_keys: ['dashboard', 'notifications', 'images', 'projects', 'builds'],
                    route_keys: ['dashboard.view', 'images.list', 'images.detail', 'projects.list', 'builds.list'],
                    action_keys: ['images.view_catalog', 'builds.create'],
                },
            },
        })

        render(
            <MemoryRouter>
                <Layout>
                    <div>Page Content</div>
                </Layout>
            </MemoryRouter>
        )

        expect(screen.queryByRole('link', { name: /Builds/i })).not.toBeInTheDocument()
    })

    it('shows Quarantine Requests when quarantine request capability is enabled', () => {
        window.localStorage.setItem('if_layout_open_nav_section', 'Quarantine')
        useOperationCapabilitiesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-1',
            isLoading: false,
            capabilities: {
                build: false,
                quarantine_request: true,
                ondemand_image_scanning: false,
            },
        })
        useCapabilitySurfacesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-1',
            isLoading: false,
            data: {
                surfaces: {
                    nav_keys: ['dashboard', 'notifications', 'images', 'quarantine_requests'],
                    route_keys: ['dashboard.view', 'images.list', 'images.detail', 'quarantine.request.list', 'quarantine.request.create'],
                    action_keys: ['images.view_catalog', 'quarantine.request.create'],
                },
            },
        })

        render(
            <MemoryRouter>
                <Layout>
                    <div>Page Content</div>
                </Layout>
            </MemoryRouter>
        )

        expect(screen.getByRole('link', { name: /Quarantine Requests/i })).toBeInTheDocument()
        expect(screen.queryByRole('link', { name: /On-Demand Scans/i })).not.toBeInTheDocument()
    })

    it('shows Registry Auth under Security when quarantine request capability is enabled', () => {
        window.localStorage.setItem('if_layout_open_nav_section', 'Security')
        useOperationCapabilitiesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-1',
            isLoading: false,
            capabilities: {
                build: false,
                quarantine_request: true,
                ondemand_image_scanning: false,
            },
        })
        useCapabilitySurfacesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-1',
            isLoading: false,
            data: {
                surfaces: {
                    nav_keys: ['dashboard', 'notifications', 'images', 'quarantine_requests', 'auth_management'],
                    route_keys: ['dashboard.view', 'images.list', 'images.detail', 'quarantine.request.list', 'settings.auth'],
                    action_keys: ['images.view_catalog', 'settings.auth.manage'],
                },
            },
        })

        render(
            <MemoryRouter>
                <Layout>
                    <div>Page Content</div>
                </Layout>
            </MemoryRouter>
        )

        expect(screen.getByRole('link', { name: /Registry Auth/i })).toBeInTheDocument()
    })

    it('shows On-Demand Scans in its own Image Scanning section when scan route is entitled', () => {
        window.localStorage.setItem('if_layout_open_nav_section', 'Image Scanning')
        useOperationCapabilitiesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-1',
            isLoading: false,
            capabilities: {
                build: false,
                quarantine_request: false,
                ondemand_image_scanning: true,
            },
        })
        useCapabilitySurfacesStoreMock.mockReturnValue({
            loadedTenantId: 'tenant-1',
            isLoading: false,
            data: {
                surfaces: {
                    nav_keys: ['dashboard', 'notifications', 'images'],
                    route_keys: ['dashboard.view', 'images.list', 'images.detail', 'images.scan.ondemand'],
                    action_keys: ['images.view_catalog', 'images.scan.ondemand'],
                },
            },
        })

        render(
            <MemoryRouter>
                <Layout>
                    <div>Page Content</div>
                </Layout>
            </MemoryRouter>
        )

        expect(screen.getByRole('button', { name: /Image Scanning/i })).toBeInTheDocument()
        const scanLink = screen.getByRole('link', { name: /On-Demand Scans/i })
        expect(scanLink).toBeInTheDocument()
        expect(scanLink).toHaveAttribute('href', '/images/scans')
        expect(screen.queryByRole('link', { name: /Quarantine Requests/i })).not.toBeInTheDocument()
    })

    it('routes Capability Access nav item to help page', () => {
        window.localStorage.setItem('if_layout_open_nav_section', 'Help')
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
            data: {
                surfaces: {
                    nav_keys: ['dashboard', 'notifications', 'images'],
                    route_keys: ['dashboard.view', 'images.list', 'images.detail'],
                    action_keys: ['images.view_catalog'],
                },
            },
        })

        render(
            <MemoryRouter>
                <Layout>
                    <div>Page Content</div>
                </Layout>
            </MemoryRouter>
        )

        const link = screen.getByRole('link', { name: /Capability Access/i })
        expect(link).toHaveAttribute('href', '/help/capabilities')
    })
})
