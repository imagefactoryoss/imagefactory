import { profileService } from '@/services/profileService'
import { LoginResponse, UserResponse, UserRoleWithPermissions } from '@/types'
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { useTenantStore } from './tenant'
import { NIL_TENANT_ID } from '@/constants/tenant'

interface AuthState {
    user: UserResponse | null
    token: string | null
    refreshToken: string | null
    tokenExpiry: number | null // Unix timestamp when access token expires
    isAuthenticated: boolean
    isLoading: boolean
    avatar?: string
    preferences?: Record<string, any>
    roles?: UserRoleWithPermissions[]
    groups?: Array<{
        id: string
        name: string
        role_type: string
        tenant_id: string
        is_admin: boolean
    }>
    rolesByTenant?: Record<string, UserRoleWithPermissions[]>
    isSystemAdmin?: boolean
    canAccessAdmin?: boolean
    defaultLandingRoute?: string
    setupRequired?: boolean
    requiresPasswordChange?: boolean
}

interface AuthActions {
    login: (loginResponse: LoginResponse) => Promise<void>
    logout: () => void
    setLoading: (loading: boolean) => void
    updateUser: (userData: Partial<UserResponse>) => void
    setProfile: (avatar?: string, roles?: any[], groups?: any[]) => void
    setPreferences: (preferences?: Record<string, any>) => void
    refreshProfile: () => Promise<void>
    refreshTenantData: () => Promise<void> // Public method to refresh tenant data
    updateTokens: (token: string, refreshToken: string, tokenExpiry?: number) => void
    setSetupRequired: (required: boolean) => void
    setRequiresPasswordChange: (required: boolean) => void
}

type AuthStore = AuthState & AuthActions

export const useAuthStore = create<AuthStore>()(
    persist(
        (set, get) => ({
            // State
            user: null,
            token: null,
            refreshToken: null,
            tokenExpiry: null,
            isAuthenticated: false,
            isLoading: false,
            avatar: undefined,
            preferences: undefined,
            roles: undefined,
            isSystemAdmin: undefined,
            canAccessAdmin: undefined,
            defaultLandingRoute: undefined,
            setupRequired: false,
            requiresPasswordChange: false,

            // Actions
            login: async (loginResponse: LoginResponse) => {
                set({
                    user: loginResponse.user,
                    token: loginResponse.access_token,
                    refreshToken: loginResponse.refresh_token,
                    tokenExpiry: loginResponse.access_token_expiry || null,
                    isAuthenticated: true,
                        isLoading: true, // Keep loading true while fetching profile
                    setupRequired: !!loginResponse.setup_required,
                    requiresPasswordChange: !!loginResponse.requires_password_change,
                })
                // Fetch full user profile with roles and groups
                try {
                    const profile = await profileService.getProfile()

                    // Populate tenant store for all users with tenant assignments
                    // Build from both RBAC roles and group-based tenant assignments
                    const tenantMap = new Map<string, any>()

                    // First, add tenants from roles_by_tenant (RBAC role assignments)
                    if (profile.roles_by_tenant) {
                        const tenantIds = Object.keys(profile.roles_by_tenant)
                        tenantIds.forEach(id => {
                            if (!id || id === NIL_TENANT_ID) return
                            const name = profile.tenant_names?.[id] || `Tenant ${id.slice(0, 8)}...`
                            const roles = profile.roles_by_tenant![id].map(role => ({
                                id: role.id,
                                name: role.name,
                                permissions: [],
                                is_admin: role.name === 'Owner'
                            }))
                            if (!tenantMap.has(id)) {
                                tenantMap.set(id, { id, name, roles })
                            }
                        })
                    }

                    // Then, add/merge tenants from groups (group-based role assignments)
                    if (profile.groups && profile.groups.length > 0) {
                        profile.groups.forEach((group: any) => {
                            if (group.tenant_id && group.tenant_id !== NIL_TENANT_ID) {
                                if (!tenantMap.has(group.tenant_id)) {
                                    // Tenant not in RBAC roles, add from groups
                                    // Use tenant name from profile.tenant_names if available, otherwise fallback
                                    const tenantName = profile.tenant_names?.[group.tenant_id] || `Tenant ${group.tenant_id.slice(0, 8)}...`
                                    tenantMap.set(group.tenant_id, {
                                        id: group.tenant_id,
                                        name: tenantName,
                                        roles: [{
                                            id: group.id,
                                            name: group.role_type.charAt(0).toUpperCase() + group.role_type.slice(1),
                                            permissions: [],
                                            is_admin: group.role_type === 'owner'
                                        }]
                                    })
                                }
                            }
                        })
                    }

                    const userTenants = Array.from(tenantMap.values())
                    useTenantStore.getState().setUserTenants(userTenants)

                    set({
                        user: {
                            ...loginResponse.user,
                            name: [loginResponse.user.first_name, loginResponse.user.last_name].filter(Boolean).join(' ') || loginResponse.user.email,
                            roles: profile.roles,
                        },
                        avatar: profile.avatar,
                        preferences: profile.preferences,
                        roles: profile.roles,
                        groups: profile.groups,
                        rolesByTenant: profile.roles_by_tenant,
                        isSystemAdmin: profile.is_system_admin,
                        canAccessAdmin: profile.can_access_admin,
                        defaultLandingRoute: profile.default_landing_route,
                        isLoading: false, // Loading complete after profile is loaded
                    })
                } catch (error) {
                    // Fail closed: profile contract/data is mandatory after authentication.
                    set({
                        user: null,
                        token: null,
                        refreshToken: null,
                        tokenExpiry: null,
                        isAuthenticated: false,
                        isLoading: false,
                        avatar: undefined,
                        preferences: undefined,
                        roles: undefined,
                        groups: undefined,
                        rolesByTenant: undefined,
                        isSystemAdmin: undefined,
                        canAccessAdmin: undefined,
                        defaultLandingRoute: undefined,
                        setupRequired: false,
                        requiresPasswordChange: false,
                    })
                    useTenantStore.getState().setUserTenants([])
                    throw error
                }
            },

            logout: () => {
                set({
                    user: null,
                    token: null,
                    refreshToken: null,
                    tokenExpiry: null,
                    isAuthenticated: false,
                    isLoading: false,
                    avatar: undefined,
                    preferences: undefined,
                    roles: undefined,
                    groups: undefined,
                    isSystemAdmin: undefined,
                    canAccessAdmin: undefined,
                    defaultLandingRoute: undefined,
                    setupRequired: false,
                    requiresPasswordChange: false,
                })
                // Clear tenant store when logging out
                useTenantStore.getState().setUserTenants([])
            },

            setLoading: (loading: boolean) => {
                set({ isLoading: loading })
            },

            updateUser: (userData: Partial<UserResponse>) => {
                const { user } = get()
                if (user) {
                    set({
                        user: { ...user, ...userData },
                    })
                }
            },

            setProfile: (avatar?: string, roles?: any[], groups?: any[]) => {
                set({
                    avatar,
                    roles,
                    groups,
                })
            },
            setPreferences: (preferences?: Record<string, any>) => {
                set({ preferences })
            },

            refreshProfile: async () => {
                set({ isLoading: true })
                try {
                    const profile = await profileService.getProfile()

                    // Populate tenant store similar to login
                    const tenantMap = new Map<string, any>()

                    // First, add tenants from roles_by_tenant (RBAC role assignments)
                    if (profile.roles_by_tenant) {
                        const tenantIds = Object.keys(profile.roles_by_tenant)
                        tenantIds.forEach(id => {
                            if (!id || id === NIL_TENANT_ID) return
                            const name = profile.tenant_names?.[id] || `Tenant ${id.slice(0, 8)}...`
                            const roles = profile.roles_by_tenant![id].map(role => ({
                                id: role.id,
                                name: role.name,
                                permissions: [],
                                is_admin: role.name === 'Owner'
                            }))
                            if (!tenantMap.has(id)) {
                                tenantMap.set(id, { id, name, roles })
                            }
                        })
                    }

                    // Then, add/merge tenants from groups (group-based role assignments)
                    if (profile.groups && profile.groups.length > 0) {
                        profile.groups.forEach((group: any) => {
                            if (group.tenant_id && group.tenant_id !== NIL_TENANT_ID) {
                                if (!tenantMap.has(group.tenant_id)) {
                                    const tenantName = profile.tenant_names?.[group.tenant_id] || `Tenant ${group.tenant_id.slice(0, 8)}...`
                                    tenantMap.set(group.tenant_id, {
                                        id: group.tenant_id,
                                        name: tenantName,
                                        roles: [{
                                            id: group.id,
                                            name: group.role_type.charAt(0).toUpperCase() + group.role_type.slice(1),
                                            permissions: [],
                                            is_admin: group.role_type === 'owner'
                                        }]
                                    })
                                }
                            }
                        })
                    }

                    const userTenants = Array.from(tenantMap.values())

                    // Set auth store data FIRST (groups, roles) before marking loading as false
                    // This ensures hasTenantAccess() hook has the groups data ready when routes evaluate
                    set({
                        avatar: profile.avatar,
                        preferences: profile.preferences,
                        roles: profile.roles,
                        groups: profile.groups,
                        rolesByTenant: profile.roles_by_tenant,
                        isSystemAdmin: profile.is_system_admin,
                        canAccessAdmin: profile.can_access_admin,
                        defaultLandingRoute: profile.default_landing_route,
                    })

                    // Then set tenant context AFTER auth data is set
                    useTenantStore.getState().setUserTenants(userTenants)

                    // Finally mark loading as false - at this point all data is ready
                    set({ isLoading: false })
                } catch (error) {
                    set({ isLoading: false })
                    throw error
                }
            },

            refreshTenantData: async () => {
                // This is a public alias for refreshProfile that focuses on tenant data refresh
                await get().refreshProfile()
            },
            updateTokens: (token: string, refreshToken: string, tokenExpiry?: number) => {
                set({
                    token,
                    refreshToken,
                    tokenExpiry: tokenExpiry || null,
                    isAuthenticated: true,
                })
            },
            setSetupRequired: (required: boolean) => {
                set({ setupRequired: required })
            },
            setRequiresPasswordChange: (required: boolean) => {
                set({ requiresPasswordChange: required })
            },
        }),
        {
            name: 'auth-storage',
            partialize: (state) => ({
                user: state.user,
                token: state.token,
                refreshToken: state.refreshToken,
                tokenExpiry: state.tokenExpiry,
                isAuthenticated: state.isAuthenticated,
                avatar: state.avatar,
                preferences: state.preferences,
                roles: state.roles,
                isSystemAdmin: state.isSystemAdmin,
                canAccessAdmin: state.canAccessAdmin,
                defaultLandingRoute: state.defaultLandingRoute,
                setupRequired: state.setupRequired,
                requiresPasswordChange: state.requiresPasswordChange,
                // NOTE: Do NOT persist groups - they must be fetched fresh on every login
                // This prevents stale group data from being used
            }),
        }
    )
)
