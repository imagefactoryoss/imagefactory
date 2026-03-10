import { api } from './api'

export interface User {
    id: string
    email: string
    first_name: string
    last_name: string
    status: string
    is_active: boolean
    auth_method: string
}

export interface UserResponse {
    user: User
    roles: Array<{
        id: string
        name: string
        description: string
        is_system: boolean
    }>
    roles_by_tenant: Record<string, Array<{
        id: string
        name: string
        description: string
        is_system: boolean
    }>>
}

export interface CreateUserRequest {
    email: string
    name: string
    password: string
}

export interface UpdateUserRequest {
    name?: string
    status?: string
}

export class UserService {
    async listUsers(tenantId: string, page: number = 1, limit: number = 10) {
        const response = await api.get(`/users`, {
            params: {
                tenantId,
                page,
                limit,
            },
        })
        return response.data
    }

    async getUser(userId: string): Promise<UserResponse> {
        const response = await api.get(`/users/${userId}`)
        return response.data
    }

    async createUser(user: CreateUserRequest) {
        const response = await api.post('/users', user)
        return response.data
    }

    async updateUser(userId: string, updates: UpdateUserRequest) {
        const response = await api.put(`/users/${userId}`, updates)
        return response.data
    }

    async deleteUser(userId: string) {
        const response = await api.delete(`/users/${userId}`)
        return response.data
    }

    async resetPassword(userId: string) {
        const response = await api.post(`/users/${userId}/reset-password`, {})
        return response.data
    }

    async assignGroup(userId: string, groupId: string) {
        const response = await api.post(`/users/${userId}/groups`, {
            group_id: groupId,
        })
        return response.data
    }

    async removeGroup(userId: string, groupId: string) {
        const response = await api.delete(`/users/${userId}/groups/${groupId}`)
        return response.data
    }
}

export const userService = new UserService()
