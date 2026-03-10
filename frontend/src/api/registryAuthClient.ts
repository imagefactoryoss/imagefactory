import { api } from '@/services/api'
import {
    CreateRegistryAuthRequest,
    RegistryAuth,
    RegistryAuthListResponse,
    TestRegistryAuthPermissionsRequest,
    TestRegistryAuthPermissionsResponse,
    UpdateRegistryAuthRequest
} from '@/types/registryAuth'

class RegistryAuthClient {
    async createRegistryAuth(data: CreateRegistryAuthRequest): Promise<RegistryAuth> {
        const response = await api.post('/registry-auth', data)
        return response.data
    }

    async listRegistryAuth(projectId?: string, includeTenant: boolean = true): Promise<RegistryAuthListResponse> {
        const params = new URLSearchParams()
        if (projectId) {
            params.set('project_id', projectId)
            params.set('include_tenant', includeTenant ? 'true' : 'false')
        }
        const suffix = params.toString() ? `?${params.toString()}` : ''
        const response = await api.get(`/registry-auth${suffix}`)
        return response.data
    }

    async deleteRegistryAuth(id: string): Promise<void> {
        await api.delete(`/registry-auth/${id}`)
    }

    async updateRegistryAuth(id: string, data: UpdateRegistryAuthRequest): Promise<RegistryAuth> {
        const response = await api.put(`/registry-auth/${id}`, data)
        return response.data
    }

    async testRegistryAuthPermissions(id: string, data: TestRegistryAuthPermissionsRequest): Promise<TestRegistryAuthPermissionsResponse> {
        const response = await api.post(`/registry-auth/${id}/test-permissions`, data)
        return response.data
    }
}

export const registryAuthClient = new RegistryAuthClient()
