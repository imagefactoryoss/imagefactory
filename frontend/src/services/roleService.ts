import { api } from './api'

export interface Role {
    id: string
    name: string
    description: string
    permissions: string[]
    isSystem: boolean
    createdAt: string
}

export interface CreateRoleRequest {
    name: string
    description: string
    permissions: string[]
}

export interface UpdateRoleRequest {
    name?: string
    description?: string
    permissions?: string[]
}

export interface Permission {
    id: string
    name: string
    description: string
    category: string
}

export class RoleService {
    async listRoles(tenantId: string, page: number = 1, limit: number = 10) {
        const response = await api.get(`/roles`, {
            params: {
                tenant_id: tenantId,
                page,
                limit,
            },
        })
        return response.data
    }

    async getRole(roleId: string) {
        const response = await api.get(`/roles/${roleId}`)
        return response.data
    }

    async createRole(role: CreateRoleRequest) {
        const response = await api.post('/roles', role)
        return response.data
    }

    async updateRole(roleId: string, updates: UpdateRoleRequest) {
        const response = await api.put(`/roles/${roleId}`, updates)
        return response.data
    }

    async deleteRole(roleId: string) {
        const response = await api.delete(`/roles/${roleId}`)
        return response.data
    }

    async listPermissions() {
        const response = await api.get(`/permissions`)
        return response.data
    }

    async getRolePermissions(roleId: string) {
        const response = await api.get(`/roles/${roleId}/permissions`)
        return response.data
    }
}

export const roleService = new RoleService()
