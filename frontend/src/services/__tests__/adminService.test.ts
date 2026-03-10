import { beforeEach, describe, expect, it, vi } from 'vitest'
import { adminService } from '../adminService'
import api from '../api'

// Mock the api module
vi.mock('../api')

describe('adminService.getPermissions', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('should fetch permissions without pagination parameters', async () => {
        const mockPermissions = [
            {
                id: '1',
                resource: 'user',
                action: 'create',
                description: 'Create a user',
                category: 'user_management',
                is_system_permission: true,
            },
            {
                id: '2',
                resource: 'user',
                action: 'read',
                description: 'Read user details',
                category: 'user_management',
                is_system_permission: true,
            },
            {
                id: '3',
                resource: 'role',
                action: 'update',
                description: 'Update role',
                category: 'role_management',
                is_system_permission: true,
            },
        ]

        vi.mocked(api.get).mockResolvedValue({
            data: {
                data: mockPermissions,
                pagination: { page: 1, page_size: 50, total: 3, total_pages: 1 },
            },
        })

        const result = await adminService.getPermissions()

        expect(api.get).toHaveBeenCalledWith('/permissions')
        expect(result).toHaveLength(3)
        expect(result[0]).toEqual({
            id: '1',
            name: 'user:create',
            resource: 'user',
            action: 'create',
            description: 'Create a user',
        })
    })

    it('should support pagination parameters', async () => {
        const mockPermissions = [
            {
                id: '1',
                resource: 'user',
                action: 'create',
                description: 'Create a user',
                category: 'user_management',
                is_system_permission: true,
            },
        ]

        vi.mocked(api.get).mockResolvedValue({
            data: {
                data: mockPermissions,
                pagination: { page: 2, page_size: 10, total: 25, total_pages: 3 },
            },
        })

        const result = await adminService.getPermissions({ page: 2, pageSize: 10 })

        expect(api.get).toHaveBeenCalledWith('/permissions?page=2&page_size=10')
        expect(result).toHaveLength(1)
    })

    it('should filter permissions by resource', async () => {
        const mockPermissions = [
            {
                id: '1',
                resource: 'user',
                action: 'create',
                description: 'Create a user',
                category: 'user_management',
                is_system_permission: true,
            },
            {
                id: '2',
                resource: 'user',
                action: 'read',
                description: 'Read user',
                category: 'user_management',
                is_system_permission: true,
            },
        ]

        vi.mocked(api.get).mockResolvedValue({
            data: {
                data: mockPermissions,
                pagination: { page: 1, page_size: 50, total: 2, total_pages: 1 },
            },
        })

        const result = await adminService.getPermissions({ resource: 'user' })

        expect(api.get).toHaveBeenCalledWith('/permissions?resource=user')
        expect(result).toHaveLength(2)
        expect(result.every((p) => p.resource === 'user')).toBe(true)
    })

    it('should handle empty permission list', async () => {
        vi.mocked(api.get).mockResolvedValue({
            data: {
                data: [],
                pagination: { page: 1, page_size: 50, total: 0, total_pages: 0 },
            },
        })

        const result = await adminService.getPermissions()

        expect(result).toEqual([])
    })

    it('should throw error on API failure', async () => {
        const errorMessage = 'Failed to fetch permissions'
        vi.mocked(api.get).mockRejectedValue(new Error(errorMessage))

        await expect(adminService.getPermissions()).rejects.toThrow(errorMessage)
    })

    it('should transform permission format correctly', async () => {
        const mockPermission = {
            id: 'perm-123',
            resource: 'build',
            action: 'cancel',
            description: 'Cancel a build',
            category: 'build_management',
            is_system_permission: true,
        }

        vi.mocked(api.get).mockResolvedValue({
            data: {
                data: [mockPermission],
                pagination: { page: 1, page_size: 50, total: 1, total_pages: 1 },
            },
        })

        const result = await adminService.getPermissions()

        expect(result[0]).toEqual({
            id: 'perm-123',
            name: 'build:cancel',
            resource: 'build',
            action: 'cancel',
            description: 'Cancel a build',
        })
    })
})

describe('adminService.tektonTaskImages', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('should fetch tekton task images config', async () => {
        const mockConfig = {
            git_clone: 'registry.local/alpine-git:2.45.2',
            kaniko_executor: 'registry.local/kaniko:v1.23.2',
            buildkit: 'registry.local/buildkit:v0.13.2',
            skopeo: 'registry.local/skopeo:v1.15.0',
            trivy: 'registry.local/trivy:0.57.1',
            syft: 'registry.local/syft:v1.18.1',
            cosign: 'registry.local/cosign:v2.4.1',
            packer: 'registry.local/packer:1.10.2',
            python_alpine: 'registry.local/python:3.12-alpine',
            alpine: 'registry.local/alpine:3.20',
            cleanup_kubectl: 'registry.local/kubectl:latest',
        }
        vi.mocked(api.get).mockResolvedValue({ data: mockConfig })

        const result = await adminService.getTektonTaskImages()

        expect(api.get).toHaveBeenCalledWith('/admin/settings/tekton-task-images')
        expect(result).toEqual(mockConfig)
    })

    it('should update tekton task images config', async () => {
        const payload = {
            git_clone: 'registry.local/alpine-git:2.45.2',
            kaniko_executor: 'registry.local/kaniko:v1.23.2',
            buildkit: 'registry.local/buildkit:v0.13.2',
            skopeo: 'registry.local/skopeo:v1.15.0',
            trivy: 'registry.local/trivy:0.57.1',
            syft: 'registry.local/syft:v1.18.1',
            cosign: 'registry.local/cosign:v2.4.1',
            packer: 'registry.local/packer:1.10.2',
            python_alpine: 'registry.local/python:3.12-alpine',
            alpine: 'registry.local/alpine:3.20',
            cleanup_kubectl: 'registry.local/kubectl:latest',
        }
        vi.mocked(api.put).mockResolvedValue({ data: payload })

        const result = await adminService.updateTektonTaskImages(payload)

        expect(api.put).toHaveBeenCalledWith('/admin/settings/tekton-task-images', payload)
        expect(result).toEqual(payload)
    })
})
