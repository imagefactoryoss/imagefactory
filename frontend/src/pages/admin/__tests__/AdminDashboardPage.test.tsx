import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import AdminDashboardPage from '../AdminDashboardPage'

const getSystemStatsMock = vi.fn()
const getSystemComponentsStatusMock = vi.fn()
const getExecutionPipelineHealthMock = vi.fn()
const getReleaseGovernancePolicyMock = vi.fn()
const getSystemConfigsMock = vi.fn()

vi.mock('@/hooks/useBuildStatusWebSocket', () => ({
  __esModule: true,
  default: () => ({ isConnected: false }),
}))

vi.mock('@/store/auth', () => ({
  useAuthStore: () => ({
    user: { id: 'u1', email: 'admin@example.com', name: 'Admin' },
    groups: [{ role_type: 'system_administrator' }],
  }),
}))

vi.mock('@/store/tenant', () => ({
  useTenantStore: () => ({
    selectedTenantId: 'tenant-1',
    userTenants: [{ id: 'tenant-1' }],
  }),
}))

vi.mock('@/services/adminService', () => ({
  adminService: {
    getSystemStats: (...args: any[]) => getSystemStatsMock(...args),
    getSystemComponentsStatus: (...args: any[]) => getSystemComponentsStatusMock(...args),
    getExecutionPipelineHealth: (...args: any[]) => getExecutionPipelineHealthMock(...args),
    getReleaseGovernancePolicy: (...args: any[]) => getReleaseGovernancePolicyMock(...args),
    getSystemConfigs: (...args: any[]) => getSystemConfigsMock(...args),
  },
}))

describe('AdminDashboardPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getSystemStatsMock.mockResolvedValue({
      total_users: 10,
      active_users: 6,
      total_tenants: 3,
      active_tenants: 2,
      total_builds: 0,
      running_builds: 1,
      total_images: 0,
      storage_used_gb: 0,
      critical_vulnerabilities: 0,
      system_health: 'healthy',
      uptime: '99.9%',
      denial_metrics: [],
      release_metrics: {
        requested: 20,
        released: 12,
        failed: 8,
        total: 20,
      },
    })
    getSystemComponentsStatusMock.mockResolvedValue({ components: {} })
    getExecutionPipelineHealthMock.mockResolvedValue({ components: {}, checked_at: '2026-03-02T00:00:00Z' })
    getReleaseGovernancePolicyMock.mockResolvedValue({
      enabled: true,
      failure_ratio_threshold: 0.3,
      consecutive_failures_threshold: 3,
      minimum_samples: 5,
      window_minutes: 60,
    })
    getSystemConfigsMock.mockResolvedValue([])
  })

  it('renders degraded release-governance telemetry when failure ratio breaches threshold', async () => {
    render(
      <MemoryRouter>
        <AdminDashboardPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('Release Governance Telemetry')).toBeInTheDocument()
    expect(await screen.findByText('degraded')).toBeInTheDocument()
    expect(screen.getByText('40.0% (threshold 30.0%)')).toBeInTheDocument()
  })
})

