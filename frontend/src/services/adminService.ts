import {
    BulkOperation,
    CreateSystemConfigRequest,
    InvitationFilters,
    LoginHistory,
    PaginatedResponse,
    Permission,
    SystemConfig,
    SystemStats,
    Tenant,
    TenantManagementFilters,
    UpdateSystemConfigRequest,
    UserActivity,
    UserInvitation,
    UserManagementFilters,
    UserRole,
    UserRoleWithPermissions,
    UserSession,
    UserWithRoles,
    DispatcherMetrics,
    DispatcherStatus,
    ExecutionPipelineHealthResponse,
    SystemComponentsStatusResponse,
    TektonTaskImagesConfig,
    ReleaseGovernancePolicyConfig,
} from '@/types'
import { useTenantStore } from '@/store/tenant'
import { NIL_TENANT_ID } from '@/constants/tenant'
import api from './api'

const getTenantHeader = () => {
    const { selectedTenantId, userTenants } = useTenantStore.getState()
    const tenantId = selectedTenantId || userTenants?.[0]?.id
    return tenantId && tenantId !== NIL_TENANT_ID ? { 'X-Tenant-ID': tenantId } : undefined
}

// Admin service for system management
export const adminService = {
    // System statistics
    async getSystemStats(): Promise<SystemStats> {
        try {
            const response = await api.get('/admin/stats', {
                headers: getTenantHeader(),
            })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getDispatcherMetrics(): Promise<DispatcherMetrics> {
        try {
            const response = await api.get('/admin/dispatcher/metrics', {
                headers: getTenantHeader(),
            })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getDispatcherStatus(): Promise<DispatcherStatus> {
        try {
            const response = await api.get('/admin/dispatcher/status', {
                headers: getTenantHeader(),
            })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getSystemComponentsStatus(): Promise<SystemComponentsStatusResponse> {
        try {
            const response = await api.get('/admin/system/components-status', {
                headers: getTenantHeader(),
            })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getExecutionPipelineHealth(): Promise<ExecutionPipelineHealthResponse> {
        try {
            const response = await api.get('/admin/execution-pipeline/health', {
                headers: getTenantHeader(),
            })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getReleaseGovernancePolicy(): Promise<ReleaseGovernancePolicyConfig> {
        try {
            const response = await api.get('/admin/settings/release-governance-policy', {
                headers: getTenantHeader(),
            })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async startDispatcher(): Promise<DispatcherStatus> {
        try {
            const response = await api.post('/admin/dispatcher/start', undefined, {
                headers: getTenantHeader(),
            })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async stopDispatcher(): Promise<DispatcherStatus> {
        try {
            const response = await api.post('/admin/dispatcher/stop', undefined, {
                headers: getTenantHeader(),
            })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async startOrchestrator(): Promise<{ running: boolean; enabled: boolean; available: boolean; message?: string }> {
        try {
            const response = await api.post('/admin/orchestrator/start', undefined, {
                headers: getTenantHeader(),
            })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async stopOrchestrator(): Promise<{ running: boolean; enabled: boolean; available: boolean; message?: string }> {
        try {
            const response = await api.post('/admin/orchestrator/stop', undefined, {
                headers: getTenantHeader(),
            })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // User management
    async getUsers(filters?: UserManagementFilters): Promise<PaginatedResponse<UserWithRoles>> {
        try {
            const params = new URLSearchParams()
            params.append('all_tenants', 'true')
            if (filters) {
                if (filters.role) params.append('role', filters.role.join(','))
                if (filters.tenantId) params.append('tenantId', filters.tenantId)
                if (filters.mfaEnabled !== undefined) params.append('mfaEnabled', filters.mfaEnabled.toString())
                if (filters.status) params.append('status', filters.status)
                if (filters.search) params.append('search', filters.search)
                if (filters.page) params.append('page', filters.page.toString())
                if (filters.limit) params.append('limit', filters.limit.toString())
            }

            const response = await api.get(`/users?${params.toString()}`)

            // Transform backend response { users: [...], total: N } to frontend format
            const backendData = response.data
            const userResponses = backendData.users || []
            const page = filters?.page || 1
            const limit = filters?.limit || 20
            const total = backendData.total || 0
            const totalPages = Math.ceil(total / limit)

            // Transform user responses to UserWithRoles format
            const userData: UserWithRoles[] = userResponses.map((item: any) => ({
                id: item.user.id,
                email: item.user.email,
                name: `${item.user.first_name} ${item.user.last_name}`,
                role: 'viewer' as UserRole,
                status: item.user.status,
                auth_method: item.user.auth_method,
                isMFAEnabled: false,
                loginCount: 0,
                roles: (item.roles || []).map((role: any) => ({
                    id: role.id,
                    name: role.name,
                    description: role.description,
                    permissions: [], // Backend will provide these if needed
                })),
                rolesByTenant: item.roles_by_tenant || {},
                permissions: [],
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
            }))

            return {
                data: userData,
                pagination: {
                    page,
                    limit,
                    total,
                    totalPages,
                }
            }
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async createUser(data: {
        email: string
        firstName: string
        lastName: string
        password?: string
        status?: string
        tenantIds?: string[]
        roleAssignments?: Array<{ tenantId: string; roleId: string }>
    }): Promise<UserWithRoles> {
        try {
            const response = await api.post('/users', data)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getUserById(id: string): Promise<UserWithRoles> {
        try {
            const response = await api.get(`/users/${id}`)
            const userData = response.data.data || response.data

            // Map roles_by_tenant to rolesByTenant
            const rolesByTenant: Record<string, any[]> = {}
            if (userData.roles_by_tenant) {
                Object.entries(userData.roles_by_tenant).forEach(([tenantId, roles]: [string, any]) => {
                    // Deduplicate roles by ID
                    const uniqueRoles = Array.from(
                        new Map(
                            (roles || []).map((role: any) => [
                                role.id,
                                {
                                    id: role.id,
                                    name: role.name,
                                    description: role.description || '',
                                    permissions: [],
                                }
                            ])
                        ).values()
                    )
                    rolesByTenant[tenantId] = uniqueRoles
                })
            }

            // Map backend response to frontend format
            return {
                id: userData.user.id,
                email: userData.user.email,
                name: `${userData.user.first_name} ${userData.user.last_name}`,
                role: 'viewer' as UserRole,
                status: userData.user.status,
                auth_method: userData.user.auth_method,
                isMFAEnabled: false,
                loginCount: 0,
                roles: (userData.roles || []).map((role: any) => ({
                    id: role.id,
                    name: role.name,
                    description: role.description || '',
                    permissions: [],
                })),
                rolesByTenant,
                permissions: [],
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
            }
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async updateUser(id: string, data: Partial<UserWithRoles>): Promise<UserWithRoles> {
        try {
            const response = await api.put(`/users/${id}`, data)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async deleteUser(id: string): Promise<void> {
        try {
            await api.delete(`/users/${id}`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async assignRoleToUser(userId: string, roleId: string): Promise<void> {
        try {
            await api.post(`/users/${userId}/roles/${roleId}`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async removeRoleFromUser(userId: string, roleId: string): Promise<void> {
        try {
            await api.delete(`/users/${userId}/roles/${roleId}`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async updateUserRoles(userId: string, roleIds: string[]): Promise<void> {
        try {
            await api.patch(`/users/${userId}/roles`, { roleIds })
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async suspendUser(id: string): Promise<void> {
        try {
            await api.post(`/users/${id}/suspend`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async activateUser(id: string): Promise<void> {
        try {
            await api.post(`/users/${id}/activate`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // User Invitations
    async sendInvitation(data: { email: string; tenantId: string; roleId: string; message?: string; isLDAP?: boolean }): Promise<UserInvitation> {
        try {
            const payload = {
                tenant_id: data.tenantId,
                email: data.email,
                role_id: data.roleId,
                message: data.message,
                is_ldap: data.isLDAP,
            }
            const response = await api.post('/invitations', payload)
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async addExistingUserToTenant(data: { userId: string; tenantId: string; roleIds: string[] }): Promise<void> {
        try {
            const payload = {
                userId: data.userId,
                roleIds: data.roleIds,
            }
            await api.post(`/tenants/${data.tenantId}/users`, payload)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async listInvitations(filters?: InvitationFilters): Promise<{ invitations: UserInvitation[], total: number }> {
        try {
            const params = new URLSearchParams()
            if (filters) {
                if (filters.status && filters.status.length > 0) params.append('status', filters.status.join(','))
                // tenant_id is now handled by the X-Tenant-ID header in the API interceptor
                if (filters.invitedBy) params.append('invitedBy', filters.invitedBy)
                if (filters.search) params.append('search', filters.search)
                if (filters.page) params.append('page', filters.page.toString())
                if (filters.limit) params.append('limit', filters.limit.toString())
            }

            const response = await api.get(`/invitations?${params.toString()}`)

            // Transform snake_case API response to camelCase for frontend
            const transformedInvitations = response.data.invitations.map((invitation: any) => ({
                id: invitation.id,
                email: invitation.email,
                tenantId: invitation.tenant_id,
                tenantName: invitation.tenant_name,
                roleId: invitation.role_id,
                roleName: invitation.role_name,
                status: invitation.status,
                token: invitation.token,
                expiresAt: invitation.expires_at,
                acceptedAt: invitation.accepted_at,
                createdAt: invitation.created_at,
                updatedAt: invitation.updated_at,
                invitedBy: invitation.invited_by,
                invitedByName: invitation.invited_by_name,
                message: invitation.message
            }))

            return {
                invitations: transformedInvitations,
                total: response.data.total
            }
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async revokeInvitation(invitationId: string): Promise<void> {
        try {
            await api.delete(`/invitations/${invitationId}`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async resendInvitation(invitationId: string): Promise<UserInvitation> {
        try {
            const response = await api.post(`/invitations/${invitationId}/resend`)
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getRolesByTenant(tenantId: string): Promise<UserRoleWithPermissions[]> {
        try {
            const response = await api.get(`/tenants/${tenantId}/roles`)
            return response.data.data || []
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Bulk Operations
    async importUsers(file: File, tenantId: string, roleId?: string): Promise<BulkOperation> {
        try {
            const formData = new FormData()
            formData.append('file', file)
            formData.append('tenantId', tenantId)
            if (roleId) {
                formData.append('roleId', roleId)
            }

            const response = await api.post('/users/bulk/import', formData, {
                headers: {
                    'Content-Type': 'multipart/form-data',
                },
            })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async bulkDeactivateUsers(userIds: string[], tenantId: string, reason: string): Promise<BulkOperation> {
        try {
            const response = await api.post('/users/bulk/deactivate', {
                userIds,
                tenantId,
                reason,
            })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async bulkAssignRoles(assignments: Array<{ userId: string; roleId: string; tenantId: string }>): Promise<BulkOperation> {
        try {
            const response = await api.post('/users/bulk/assign-roles', { assignments })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getBulkOperationStatus(operationId: string): Promise<BulkOperation> {
        try {
            const response = await api.get(`/bulk-operations/${operationId}`)
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async downloadBulkOperationResults(operationId: string): Promise<Blob> {
        try {
            const response = await api.get(`/bulk-operations/${operationId}/download`, {
                responseType: 'blob',
            })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // User Activity
    async getUserActivity(userId: string, limit?: number): Promise<UserActivity[]> {
        try {
            const params = new URLSearchParams()
            if (limit) params.append('limit', limit.toString())

            const response = await api.get(`/users/${userId}/activity?${params.toString()}`)
            return response.data?.data || []
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getUserLoginHistory(userId: string, limit?: number): Promise<LoginHistory[]> {
        try {
            const params = new URLSearchParams()
            if (limit) params.append('limit', limit.toString())

            const response = await api.get(`/users/${userId}/login-history?${params.toString()}`)
            return response.data?.data || []
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getUserSessions(userId: string): Promise<UserSession[]> {
        try {
            const response = await api.get(`/users/${userId}/sessions`)
            return response.data?.data || []
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Roles and permissions
    async getRoles(filters?: { page?: number; pageSize?: number }): Promise<UserRoleWithPermissions[]> {
        try {
            const params = new URLSearchParams()
            if (filters?.page) params.append('page', filters.page.toString())
            if (filters?.pageSize) params.append('page_size', filters.pageSize.toString())

            const response = await api.get(`/roles?${params.toString()}`)
            // Backend returns paginated response: { data: [...], total, page, page_size, total_pages }
            // Extract the data array and transform to UserRoleWithPermissions format
            const roles = response.data?.data || []

            return roles.map((role: any) => ({
                id: role.id,
                name: role.name,
                description: role.description || '',
                permissions: (role.permissions || []).map((perm: any) => {
                    const key = `${perm.resource}:${perm.action}`
                    return {
                        id: perm.id,  // Use ID from backend directly
                        name: perm.name || key,
                        resource: perm.resource,
                        action: perm.action,
                        description: perm.description || '',
                    }
                }),
            }))
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getRoleById(id: string): Promise<UserRoleWithPermissions> {
        try {
            const response = await api.get(`/roles/${id}`)
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async createRole(data: {
        name: string
        description?: string
        permissions: string[]
        tenantId?: string
        isSystem?: boolean
    }): Promise<UserRoleWithPermissions> {
        try {
            const response = await api.post('/roles', data)
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async updateRole(id: string, data: {
        name?: string
        description?: string
        permissions?: string[]
    }): Promise<UserRoleWithPermissions> {
        try {
            const response = await api.put(`/roles/${id}`, data)
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async deleteRole(id: string): Promise<void> {
        try {
            await api.delete(`/roles/${id}`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Permissions
    /**
     * Fetch all system permissions with optional filtering and pagination
     * Single query - no N+1 issues
     */
    async getPermissions(filters?: {
        resource?: string
        page?: number
        pageSize?: number
    }): Promise<Permission[]> {
        try {
            const params = new URLSearchParams()
            if (filters?.resource) params.append('resource', filters.resource)
            if (filters?.page) params.append('page', filters.page.toString())
            if (filters?.pageSize) params.append('page_size', filters.pageSize.toString())

            const queryString = params.toString()
            const url = queryString ? `/permissions?${queryString}` : '/permissions'
            const response = await api.get(url)

            // Backend returns { data: [...], pagination: {...} }
            const permissions = response.data?.data || []

            // Transform backend format to frontend format
            return permissions.map((perm: any) => ({
                id: perm.id,
                name: `${perm.resource}:${perm.action}`,
                resource: perm.resource,
                action: perm.action,
                description: perm.description || '',
                category: perm.category || '',
                isSystemPermission: perm.is_system_permission || false,
                createdAt: perm.created_at,
                updatedAt: perm.updated_at,
            }))
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async createPermission(data: {
        resource: string
        action: string
        description?: string
        category?: string
    }): Promise<Permission> {
        try {
            const response = await api.post('/permissions', data)
            const perm = response.data
            return {
                id: perm.id,
                name: `${perm.resource}:${perm.action}`,
                resource: perm.resource,
                action: perm.action,
                description: perm.description || '',
                category: perm.category || '',
                isSystemPermission: perm.is_system_permission || false,
                createdAt: perm.created_at,
                updatedAt: perm.updated_at,
            }
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async updatePermission(id: string, data: {
        description?: string
        category?: string
    }): Promise<Permission> {
        try {
            const response = await api.put(`/permissions/${id}`, data)
            const perm = response.data
            return {
                id: perm.id,
                name: `${perm.resource}:${perm.action}`,
                resource: perm.resource,
                action: perm.action,
                description: perm.description || '',
                category: perm.category || '',
                isSystemPermission: perm.is_system_permission || false,
                createdAt: perm.created_at,
                updatedAt: perm.updated_at,
            }
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async deletePermission(id: string): Promise<void> {
        try {
            await api.delete(`/permissions/${id}`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Tenant management
    async getTenants(filters?: TenantManagementFilters): Promise<PaginatedResponse<Tenant>> {
        try {
            const params = new URLSearchParams()
            params.append('all_tenants', 'true')
            if (filters) {
                if (filters.status) params.append('status', filters.status.join(','))
                if (filters.search) params.append('search', filters.search)
                if (filters.page) params.append('page', filters.page.toString())
                if (filters.limit) params.append('limit', filters.limit.toString())
            }

            const response = await api.get(`/tenants?${params.toString()}`)

            // Backend returns paginated response with data and pagination fields
            let tenants: Tenant[] = []
            let pagination = {
                page: filters?.page || 1,
                limit: filters?.limit || 20,
                total: 0,
                totalPages: 0,
            }

            if (response.data) {
                if (Array.isArray(response.data)) {
                    // Plain array response
                    tenants = response.data
                    pagination.total = tenants.length
                    pagination.totalPages = Math.ceil(pagination.total / pagination.limit)
                } else if (response.data.data !== undefined) {
                    // Paginated response with data and pagination fields
                    tenants = response.data.data || []
                    if (response.data.pagination) {
                        pagination = response.data.pagination
                    } else {
                        pagination.total = tenants.length
                        pagination.totalPages = Math.ceil(pagination.total / pagination.limit)
                    }
                } else if (response.data.tenants) {
                    // Legacy format with tenants field
                    tenants = response.data.tenants || []
                    pagination.total = tenants.length
                    pagination.totalPages = Math.ceil(pagination.total / pagination.limit)
                }
            }

            return {
                data: tenants,
                pagination,
            }
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async createTenant(data: {
        name: string
        slug: string
        description?: string
        status?: string
        maxBuilds?: number
        storageLimitGB?: number
    }): Promise<Tenant> {
        try {
            const response = await api.post('/tenants', data)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getTenantById(id: string): Promise<Tenant> {
        try {
            const response = await api.get(`/tenants/${id}`)
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getTenantUsers(tenantId: string): Promise<UserWithRoles[]> {
        try {
            const response = await api.get(`/tenants/${tenantId}/users`)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async assignUserToTenant(userId: string, tenantId: string, role?: string): Promise<void> {
        try {
            await api.post(`/tenants/${tenantId}/users`, { userId, role })
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async removeUserFromTenant(userId: string, tenantId: string): Promise<void> {
        try {
            await api.delete(`/tenants/${tenantId}/users/${userId}`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Tenant Member Management
    async updateTenantMemberRole(tenantId: string, userId: string, roleId: string): Promise<void> {
        try {
            await api.patch(`/tenants/${tenantId}/users/${userId}`, { roleId })
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async removeTenantMember(tenantId: string, userId: string): Promise<void> {
        try {
            await api.delete(`/tenants/${tenantId}/users/${userId}`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async updateTenant(id: string, data: Partial<Tenant>): Promise<Tenant> {
        try {
            const response = await api.patch(`/tenants/${id}`, data)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async deleteTenant(id: string): Promise<void> {
        try {
            await api.delete(`/tenants/${id}`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async suspendTenant(id: string): Promise<void> {
        try {
            await api.post(`/tenants/${id}/suspend`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async activateTenant(id: string): Promise<void> {
        try {
            await api.post(`/tenants/${id}/activate`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // System configuration
    async getSystemConfigs(): Promise<SystemConfig[]> {
        try {
            const response = await api.get('/system-configs')
            return response.data?.configs || response.data?.data || []
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getSystemConfig(id: string): Promise<SystemConfig> {
        try {
            const response = await api.get(`/system-configs/${id}`)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async createSystemConfig(data: CreateSystemConfigRequest): Promise<SystemConfig> {
        try {
            const response = await api.post('/system-configs', data)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async updateSystemConfig(id: string, data: UpdateSystemConfigRequest): Promise<SystemConfig> {
        try {
            const response = await api.patch(`/system-configs/${id}`, data)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async deleteSystemConfig(id: string): Promise<void> {
        try {
            await api.delete(`/system-configs/${id}`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async testSystemConfig(id: string): Promise<{ success: boolean; message: string }> {
        try {
            const response = await api.post(`/system-configs/${id}/test-connection`)
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getTektonTaskImages(): Promise<TektonTaskImagesConfig> {
        try {
            const response = await api.get('/admin/settings/tekton-task-images')
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async updateTektonTaskImages(data: TektonTaskImagesConfig): Promise<TektonTaskImagesConfig> {
        try {
            const response = await api.put('/admin/settings/tekton-task-images', data)
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // System maintenance
    async runSystemMaintenance(): Promise<{ success: boolean; message: string }> {
        try {
            // TODO: Implement system maintenance endpoint in backend
            throw new Error('System maintenance endpoint not yet implemented')
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getSystemLogs(
        // @ts-expect-error filters parameter is used for logging queries in future
        filters?: {
            level?: string[]
            component?: string
            tenantId?: string
            startDate?: string
            endDate?: string
            page?: number
            limit?: number
        }): Promise<PaginatedResponse<any>> {
        try {
            // TODO: Implement system logs endpoint in backend
            throw new Error('System logs endpoint not yet implemented')
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getSystemHealth(): Promise<{
        status: 'healthy' | 'warning' | 'critical'
        components: Record<string, { status: string; lastCheck: string; details?: any }>
    }> {
        try {
            // TODO: Implement system health endpoint in backend
            // For now, return a default healthy state
            return {
                status: 'healthy',
                components: {},
            }
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Group management
    async getGroupsByTenant(tenantId: string): Promise<any[]> {
        try {
            const response = await api.get(`/tenants/${tenantId}/groups`)
            return response.data.data || []
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getGroupMembers(groupId: string): Promise<any[]> {
        try {
            const response = await api.get(`/groups/${groupId}/members`)
            return response.data.data || []
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async addGroupMember(groupId: string, userId: string): Promise<void> {
        try {
            await api.post(`/groups/${groupId}/members`, {
                user_id: userId,
            })
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async removeGroupMember(groupId: string, memberId: string): Promise<void> {
        try {
            await api.delete(`/groups/${groupId}/members/${memberId}`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Build Policies
    async getBuildPolicies(filters?: {
        type?: string
        activeOnly?: boolean
    }): Promise<{ policies: any[]; total: number }> {
        try {
            const params = new URLSearchParams()
            if (filters?.type) params.append('type', filters.type)
            if (filters?.activeOnly) params.append('active_only', 'true')

            const queryString = params.toString()
            const url = queryString ? `/admin/builds/policies?${queryString}` : '/admin/builds/policies'
            const response = await api.get(url)

            return {
                policies: response.data.policies || [],
                total: response.data.total || 0,
            }
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getBuildPolicy(id: string): Promise<any> {
        try {
            const response = await api.get(`/admin/builds/policies/${id}`)
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async createBuildPolicy(data: {
        policy_type: string
        policy_key: string
        policy_value: any
        description?: string
    }): Promise<any> {
        try {
            const response = await api.post('/admin/builds/policies', data)
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async updateBuildPolicy(id: string, data: {
        policy_value?: any
        description?: string
        is_active?: boolean
    }): Promise<any> {
        try {
            const response = await api.put(`/admin/builds/policies/${id}`, data)
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async deleteBuildPolicy(id: string): Promise<void> {
        try {
            await api.delete(`/admin/builds/policies/${id}`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },
}

// Helper function to extract error message from various error formats
function getErrorMessage(error: any): string {
    if (typeof error === 'string') {
        return error
    }
    if (error?.response?.data?.error) {
        return error.response.data.error
    }
    if (error?.response?.data?.message) {
        return error.response.data.message
    }
    if (error?.message) {
        return error.message
    }
    return 'An error occurred'
}
