import { UserRoleWithPermissions } from '@/types'
import { api } from './api'

interface ProfileResponse {
    id: string
    email: string
    first_name: string
    last_name: string
    status: string
    is_active: boolean
    is_system_admin: boolean
    can_access_admin: boolean
    default_landing_route: string
    roles: UserRoleWithPermissions[]
    roles_by_tenant?: Record<string, UserRoleWithPermissions[]>
    tenant_names?: Record<string, string> // Maps tenant ID to tenant name
    groups: Array<{
        id: string
        name: string
        role_type: string
        tenant_id: string
        is_admin: boolean
    }>
    preferences?: Record<string, any>
    avatar?: string
    created_at?: string
    has_multi_tenant: boolean
}

export const profileService = {
    validateProfileContract(data: any): void {
        if (typeof data?.is_system_admin !== 'boolean') {
            throw new Error('Invalid profile contract: is_system_admin is required')
        }
        if (typeof data?.can_access_admin !== 'boolean') {
            throw new Error('Invalid profile contract: can_access_admin is required')
        }
        if (typeof data?.default_landing_route !== 'string' || data.default_landing_route.trim() === '') {
            throw new Error('Invalid profile contract: default_landing_route is required')
        }
    },

    /**
     * Get current user's profile
     */
    async getProfile(): Promise<ProfileResponse> {
        const response = await api.get('/profile')
        profileService.validateProfileContract(response.data)
        return response.data
    },

    /**
     * Update user profile
     */
    async updateProfile(data: {
        first_name?: string
        last_name?: string
        avatar?: string
        preferences?: Record<string, any>
    }): Promise<ProfileResponse> {
        const response = await api.put('/profile', data)
        profileService.validateProfileContract(response.data)
        return response.data
    },

    /**
     * Get initials from user name
     */
    getInitials(firstName: string, lastName: string): string {
        const first = firstName?.charAt(0)?.toUpperCase() || ''
        const last = lastName?.charAt(0)?.toUpperCase() || ''
        return `${first}${last}` || 'U'
    },

    /**
     * Generate avatar color based on user ID
     */
    getAvatarColor(userId: string): string {
        const colors = [
            'bg-red-500',
            'bg-blue-500',
            'bg-green-500',
            'bg-yellow-500',
            'bg-purple-500',
            'bg-pink-500',
            'bg-indigo-500',
            'bg-cyan-500',
        ]
        const hash = userId.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0)
        return colors[hash % colors.length]
    },

    /**
     * Get full name from first and last name
     */
    getFullName(firstName: string, lastName: string): string {
        const name = `${firstName} ${lastName}`.trim()
        return name || 'Unknown User'
    },
}
