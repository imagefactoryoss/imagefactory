import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { buildService } from '@/services/buildService'
import { projectService } from '@/services/projectService'
import { useTenantStore } from '@/store/tenant'
import BuildCreationWizard from './BuildCreationWizard'

describe('BuildCreationWizard - tenant-namespace conflict handling', () => {
    const tenantId = '33333333-3333-3333-3333-333333333333'
    const project = {
        id: '11111111-1111-1111-1111-111111111111',
        name: 'Mock Project',
        tenantId,
    }

    beforeEach(() => {
        // Ensure tenant context is set
        useTenantStore.getState().setUserTenants([{ id: tenantId, name: 'T', roles: [] } as any])
        useTenantStore.getState().setSelectedTenant(tenantId)
    })

    afterEach(() => {
        vi.restoreAllMocks()
    })

    it('displays tenant namespace banner when API returns structured 409', async () => {
        // Mock projects API
        vi.spyOn(projectService, 'getProjects').mockResolvedValue({ data: [project], pagination: { page: 1, limit: 20, total: 1, totalPages: 1 } } as any)

        // Mock createBuild to reject with structured 409 payload
        const errorPayload = {
            response: {
                status: 409,
                data: {
                    error: 'selected infrastructure provider is not prepared for this tenant (namespace not provisioned yet)',
                    code: 'tenant_namespace_not_prepared',
                    details: {
                        provider_id: '22222222-2222-2222-2222-222222222222',
                        tenant_id: tenantId,
                        prepare_status: 'missing',
                        namespace: 'image-factory-33333333'
                    }
                }
            }
        }
        vi.spyOn(buildService, 'createBuild').mockRejectedValue(errorPayload)

        render(
            <MemoryRouter>
                <BuildCreationWizard />
            </MemoryRouter>
        )

        // Wait for projects to load and select the project
        await waitFor(() => expect(projectService.getProjects).toHaveBeenCalled())
        const projectSelect = await screen.findByRole('combobox', { name: /Project/i })
        fireEvent.change(projectSelect, { target: { value: project.id } })

        // Fill required fields (build name)
        const buildNameInput = screen.getByPlaceholderText(/e.g., web-app-production-v1.2.3/i)
        fireEvent.change(buildNameInput, { target: { value: 'e2e-test-build' } })

        // Select a build method (Kaniko)
        const kanikoCard = await screen.findByText(/Kaniko - Dockerfile Builds/i)
        fireEvent.click(kanikoCard)

        // Advance to Validation step using Next buttons
        const nextButtons = await screen.findAllByRole('button', { name: /Next/i })
        // Click Next three times to reach Validation step
        fireEvent.click(nextButtons[0])
        await waitFor(() => expect(screen.getByText(/Tool Selection/i)).toBeInTheDocument())
        fireEvent.click(nextButtons[0])
        await waitFor(() => expect(screen.getByText(/Configuration/i)).toBeInTheDocument())
        fireEvent.click(nextButtons[0])
        await waitFor(() => expect(screen.getByText(/Review & Create Build/i)).toBeInTheDocument())

        // Click Create Build (this will trigger the mocked rejection)
        const createBtn = screen.getByRole('button', { name: /Create Build/i })
        fireEvent.click(createBtn)

        // Expect banner to show and include namespace and provider deep-link
        const bannerTitle = await screen.findByText(/Tenant Namespace Not Prepared/i)
        expect(bannerTitle).toBeTruthy()

        const namespaceText = await screen.findByText('image-factory-33333333')
        expect(namespaceText).toBeTruthy()
    })
})
