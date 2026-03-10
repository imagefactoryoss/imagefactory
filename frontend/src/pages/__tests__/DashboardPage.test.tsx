import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import DashboardPage from '../DashboardPage'

const mockDashboardHook = vi.fn()
const useCapabilitySurfacesStoreMock = vi.fn()

vi.mock('@/hooks/useTenantDashboardData', () => ({
  __esModule: true,
  default: () => mockDashboardHook(),
}))

vi.mock('@/hooks/useBuildStatusWebSocket', () => ({
  __esModule: true,
  default: () => ({ isConnected: true }),
}))

vi.mock('@/hooks/useNotificationWebSocket', () => ({
  __esModule: true,
  default: () => ({ isConnected: true }),
}))

vi.mock('@/store/auth', () => ({
  useAuthStore: (selector?: any) => {
    const state = { user: { name: 'Tenant Admin', email: 'admin@example.com' } }
    return typeof selector === 'function' ? selector(state) : state
  },
}))

vi.mock('@/store/tenant', () => ({
  useTenantStore: (selector?: any) => {
    const state = {
      userTenants: [{ id: 'tenant-1', name: 'Tenant One' }],
      selectedTenantId: 'tenant-1',
      selectedRoleId: 'role-1',
    }
    return typeof selector === 'function' ? selector(state) : state
  },
}))

vi.mock('@/store/capabilitySurfaces', () => ({
  useCapabilitySurfacesStore: (selector?: any) => {
    const state = useCapabilitySurfacesStoreMock()
    return typeof selector === 'function' ? selector(state) : state
  },
}))

vi.mock('@/components/auth/PostLoginContextSelector', () => ({
  __esModule: true,
  default: () => null,
}))

vi.mock('@/components/common/ContextIndicator', () => ({
  __esModule: true,
  default: () => <div>Context Indicator</div>,
}))

describe('DashboardPage', () => {
  it('renders live tenant metrics and widgets', () => {
    useCapabilitySurfacesStoreMock.mockReturnValue({
      loadedTenantId: 'tenant-1',
      isLoading: false,
      data: {
        capabilities: {
          build: true,
          quarantine_request: false,
          quarantine_release: false,
          ondemand_image_scanning: false,
        },
      },
      canAccessRouteKey: (routeKey: string) => routeKey === 'builds.list',
      canRunActionKey: (actionKey: string) => actionKey === 'builds.create',
    })

    mockDashboardHook.mockReturnValue({
      data: {
        stats: {
          totalProjects: 5,
          activeProjects: 4,
          buildsToday: 9,
          successRate: 88,
          runningBuilds: 2,
          queuedBuilds: 1,
          completedBuilds: 42,
          failedBuilds: 3,
          unreadNotifications: 7,
        },
        recentBuilds: [
          {
            id: 'build-1',
            name: 'Backend API Build',
            projectName: 'Backend',
            status: 'completed',
            createdAt: new Date().toISOString(),
            durationLabel: '2m 12s',
          },
        ],
        mostActiveProjects: [
          {
            id: 'project-1',
            name: 'Backend',
            buildCount: 11,
            lastBuildAt: new Date().toISOString(),
          },
        ],
        recentImages: [
          {
            id: 'img-1',
            tenant_id: 'tenant-1',
            name: 'backend-api',
            description: '',
            visibility: 'tenant',
            status: 'published',
            tags: ['latest'],
            metadata: {},
            pull_count: 0,
            created_by: 'user-1',
            updated_by: 'user-1',
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          },
        ],
        lastUpdatedAt: new Date().toISOString(),
      },
      isLoading: false,
      isRefreshing: false,
      error: null,
      refresh: vi.fn(),
      scheduleRefresh: vi.fn(),
      tenantReady: true,
    })

    render(
      <MemoryRouter>
        <DashboardPage />
      </MemoryRouter>
    )

    expect(screen.getByText('Tenant Control Tower')).toBeInTheDocument()
    expect(screen.getByText('5')).toBeInTheDocument()
    expect(screen.getByText('Backend API Build')).toBeInTheDocument()
    expect(screen.getByText('backend-api')).toBeInTheDocument()
    expect(screen.getByText('Live updates connected')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /new build/i })).toBeInTheDocument()
  })

  it('shows tenant context prompt when no tenant is selected', () => {
    useCapabilitySurfacesStoreMock.mockReturnValue({
      loadedTenantId: 'tenant-1',
      isLoading: false,
      data: {
        capabilities: {
          build: true,
          quarantine_request: false,
          quarantine_release: false,
          ondemand_image_scanning: false,
        },
      },
      canAccessRouteKey: (routeKey: string) => routeKey === 'builds.list',
      canRunActionKey: (actionKey: string) => actionKey === 'builds.create',
    })

    mockDashboardHook.mockReturnValue({
      data: {
        stats: {
          totalProjects: 0,
          activeProjects: 0,
          buildsToday: 0,
          successRate: 0,
          runningBuilds: 0,
          queuedBuilds: 0,
          completedBuilds: 0,
          failedBuilds: 0,
          unreadNotifications: 0,
        },
        recentBuilds: [],
        mostActiveProjects: [],
        recentImages: [],
        lastUpdatedAt: undefined,
      },
      isLoading: false,
      isRefreshing: false,
      error: null,
      refresh: vi.fn(),
      scheduleRefresh: vi.fn(),
      tenantReady: false,
    })

    render(
      <MemoryRouter>
        <DashboardPage />
      </MemoryRouter>
    )

    expect(screen.getByText('Select a tenant context to view realtime dashboard data.')).toBeInTheDocument()
    expect(screen.getByText('No tenant context')).toBeInTheDocument()
  })

  it('hides new build CTA when build capability is disabled', () => {
    useCapabilitySurfacesStoreMock.mockReturnValue({
      loadedTenantId: 'tenant-1',
      isLoading: false,
      data: {
        capabilities: {
          build: false,
          quarantine_request: false,
          quarantine_release: false,
          ondemand_image_scanning: false,
        },
      },
      canAccessRouteKey: () => false,
      canRunActionKey: () => false,
    })

    mockDashboardHook.mockReturnValue({
      data: {
        stats: {
          totalProjects: 1,
          activeProjects: 1,
          buildsToday: 1,
          successRate: 100,
          runningBuilds: 0,
          queuedBuilds: 0,
          completedBuilds: 1,
          failedBuilds: 0,
          unreadNotifications: 0,
        },
        recentBuilds: [],
        mostActiveProjects: [],
        recentImages: [],
        lastUpdatedAt: new Date().toISOString(),
      },
      isLoading: false,
      isRefreshing: false,
      error: null,
      refresh: vi.fn(),
      scheduleRefresh: vi.fn(),
      tenantReady: true,
    })

    render(
      <MemoryRouter>
        <DashboardPage />
      </MemoryRouter>
    )

    expect(screen.queryByRole('link', { name: /new build/i })).not.toBeInTheDocument()
    expect(screen.getByText(/No operational capabilities are enabled for this tenant/i)).toBeInTheDocument()
    expect(screen.getByText(/Build history is unavailable because this tenant is not entitled/i)).toBeInTheDocument()
    expect(screen.getByText(/Project activity is hidden because Image Build capability is not enabled/i)).toBeInTheDocument()
  })
})
