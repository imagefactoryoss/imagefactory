import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProjectWebhooksPanel from '../ProjectWebhooksPanel'
import { buildService } from '@/services/buildService'
import { buildTriggerService } from '@/services/buildTriggerService'
import { projectService } from '@/services/projectService'
import toast from 'react-hot-toast'

vi.mock('@/services/buildService', () => ({
  buildService: {
    getBuilds: vi.fn(),
  },
}))

vi.mock('@/services/buildTriggerService', () => ({
  buildTriggerService: {
    createProjectWebhookTrigger: vi.fn(),
    getProjectTriggers: vi.fn(),
    updateProjectWebhookTrigger: vi.fn(),
    deleteTrigger: vi.fn(),
  },
}))

vi.mock('@/services/projectService', () => ({
  projectService: {
    listProjectWebhookReceipts: vi.fn(),
  },
}))

vi.mock('react-hot-toast', () => ({
  default: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

describe('ProjectWebhooksPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(buildService.getBuilds).mockResolvedValue({
      data: [
        {
          id: 'build-1',
          manifest: { name: 'api-image' },
        },
      ],
    } as any)
    vi.mocked(projectService.listProjectWebhookReceipts).mockResolvedValue([])
  })

  it('blocks save when no webhook events are selected', async () => {
    vi.mocked(buildTriggerService.getProjectTriggers).mockResolvedValue([])

    render(<ProjectWebhooksPanel projectId="project-1" canManage />)

    await screen.findByText('Webhook Triggers')
    await userEvent.click(screen.getByRole('button', { name: 'Add Webhook Trigger' }))
    await userEvent.type(screen.getByLabelText('Trigger Name *'), 'GitHub push')
    await userEvent.click(screen.getByLabelText('push'))
    await userEvent.click(screen.getByRole('button', { name: 'Create Trigger' }))

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith('Select at least one webhook event')
    })
    expect(buildTriggerService.createProjectWebhookTrigger).not.toHaveBeenCalled()
  })

  it('refreshes trigger list after successful create', async () => {
    vi.mocked(buildTriggerService.getProjectTriggers)
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce([
        {
          id: 'trigger-1',
          buildId: 'build-1',
          projectId: 'project-1',
          triggerType: 'webhook',
          name: 'GitHub push trigger',
          webhookUrl: 'http://localhost/api/v1/webhooks/github/project-1',
          webhookEvents: ['push'],
          isActive: true,
          createdBy: 'user-1',
          createdAt: '2026-02-19T00:00:00Z',
          updatedAt: '2026-02-19T00:00:00Z',
        },
      ] as any)
    vi.mocked(buildTriggerService.createProjectWebhookTrigger).mockResolvedValue({} as any)

    render(<ProjectWebhooksPanel projectId="project-1" canManage />)

    await screen.findByText('Webhook Triggers')
    await userEvent.click(screen.getByRole('button', { name: 'Add Webhook Trigger' }))
    await userEvent.type(screen.getByLabelText('Trigger Name *'), 'GitHub push trigger')
    await userEvent.click(screen.getByRole('button', { name: 'Create Trigger' }))

    await waitFor(() => {
      expect(buildTriggerService.createProjectWebhookTrigger).toHaveBeenCalled()
    })
    await waitFor(() => {
      expect(buildTriggerService.getProjectTriggers).toHaveBeenCalledTimes(2)
    })
    await screen.findByText('GitHub push trigger')
  })
})
