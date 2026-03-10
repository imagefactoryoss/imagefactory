import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import ReleasedArtifactsPage from '../ReleasedArtifactsPage'

const listReleasedArtifactsMock = vi.fn()
const consumeReleasedArtifactMock = vi.fn()
const getProjectsMock = vi.fn()
const navigateMock = vi.fn()

vi.mock('@/services/imageImportService', () => ({
  imageImportService: {
    listReleasedArtifacts: (...args: any[]) => listReleasedArtifactsMock(...args),
    consumeReleasedArtifact: (...args: any[]) => consumeReleasedArtifactMock(...args),
  },
  ImageImportApiError: class extends Error {},
}))

vi.mock('@/services/projectService', () => ({
  projectService: {
    getProjects: (...args: any[]) => getProjectsMock(...args),
  },
}))

vi.mock('@/store/tenant', () => ({
  useTenantStore: () => ({ selectedTenantId: 'tenant-1' }),
}))

vi.mock('react-hot-toast', () => ({
  __esModule: true,
  default: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return {
    ...actual,
    useNavigate: () => navigateMock,
  }
})

describe('ReleasedArtifactsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listReleasedArtifactsMock.mockResolvedValue({
      items: [
        {
          id: 'artifact-1',
          tenant_id: 'tenant-1',
          requested_by_user_id: 'user-1',
          epr_record_id: 'EPR-1',
          source_registry: 'ghcr.io',
          source_image_ref: 'ghcr.io/acme/app:1.0.0',
          internal_image_ref: 'registry.local/quarantine/acme/app:1.0.0',
          source_image_digest: 'sha256:abc123',
          policy_decision: 'pass',
          release_state: 'released',
          released_at: '2026-03-04T00:00:00Z',
          consumption_ready: true,
          created_at: '2026-03-04T00:00:00Z',
          updated_at: '2026-03-04T00:00:00Z',
        },
      ],
      pagination: { page: 1, limit: 50, total: 1 },
    })
    consumeReleasedArtifactMock.mockResolvedValue(undefined)
    getProjectsMock.mockResolvedValue({
      data: [{ id: 'project-1', name: 'Project One' }],
      pagination: { page: 1, limit: 100, total: 1, totalPages: 1 },
    })
  })

  it('renders released artifact rows with use-in-project action', async () => {
    render(
      <MemoryRouter>
        <ReleasedArtifactsPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('ghcr.io/acme/app:1.0.0')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Use in Project' })).toBeEnabled()
    expect(screen.getByText('Ready')).toBeInTheDocument()
  })

  it('consumes artifact and navigates to build wizard with prefill params', async () => {
    render(
      <MemoryRouter>
        <ReleasedArtifactsPage />
      </MemoryRouter>
    )

    expect(await screen.findByText('ghcr.io/acme/app:1.0.0')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Use in Project' }))

    expect(await screen.findByText('Use Released Artifact in Project')).toBeInTheDocument()
    await screen.findByDisplayValue('Project One')
    fireEvent.click(screen.getByRole('button', { name: 'Open Build Wizard' }))

    await waitFor(() => {
      expect(consumeReleasedArtifactMock).toHaveBeenCalledWith(
        'artifact-1',
        'project-1',
        'Selected from released artifacts workspace'
      )
      expect(navigateMock).toHaveBeenCalledWith(
        expect.stringContaining('/builds/new?')
      )
      const target = String(navigateMock.mock.calls[0][0])
      expect(target).toContain('projectId=project-1')
      expect(target).toContain('baseImage=registry.local%2Fquarantine%2Facme%2Fapp%3A1.0.0')
      expect(target).toContain('sourceImageRef=ghcr.io%2Facme%2Fapp%3A1.0.0')
    })
  })
})
