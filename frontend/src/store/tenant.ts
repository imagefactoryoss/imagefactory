import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface UserRole {
    id: string
    name: string
    permissions: string[]
    is_admin: boolean
}

export interface TenantContext {
    id: string
    name: string
    roles: UserRole[]
}

interface ContextState {
    selectedTenantId: string | null
    selectedRoleId: string | null
    userTenants: TenantContext[]
    contextValidated: boolean
    setSelectedTenant: (tenantId: string) => void
    setSelectedRole: (roleId: string) => void
    setContext: (tenantId: string, roleId: string) => void
    setUserTenants: (tenants: TenantContext[]) => void
    clearContext: () => void
    validateContext: () => boolean
    getCurrentContext: () => { tenant: TenantContext | null, role: UserRole | null }
}

export const useTenantStore = create<ContextState>()(
    persist(
        (set, get) => ({
            selectedTenantId: null,
            selectedRoleId: null,
            userTenants: [],
            contextValidated: false,

            setSelectedTenant: (tenantId: string) => {
                const state = get()
                const tenant = state.userTenants.find(t => t.id === tenantId)

                // If tenant has roles, auto-select first role or clear if no roles
                let newRoleId = state.selectedRoleId
                if (tenant) {
                    if (tenant.roles.length > 0 && !tenant.roles.find(r => r.id === newRoleId)) {
                        newRoleId = tenant.roles[0].id
                    }
                } else {
                    newRoleId = null
                }

                set({
                    selectedTenantId: tenantId,
                    selectedRoleId: newRoleId,
                    contextValidated: false
                })
            },

            setSelectedRole: (roleId: string) => {
                set({
                    selectedRoleId: roleId,
                    contextValidated: false
                })
            },

            setContext: (tenantId: string, roleId: string) => {
                set({
                    selectedTenantId: tenantId,
                    selectedRoleId: roleId,
                    contextValidated: true
                })
            },

            setUserTenants: (tenants: TenantContext[]) => {
                const state = get()
                let newSelectedTenantId = state.selectedTenantId
                let newSelectedRoleId = state.selectedRoleId

                // Validate current selection is still valid
                const currentTenant = tenants.find(t => t.id === newSelectedTenantId)
                if (!currentTenant) {
                    // Current tenant not available, select first available
                    newSelectedTenantId = tenants.length > 0 ? tenants[0].id : null
                    newSelectedRoleId = null
                }

                // Now get the tenant we're actually using (either current or newly selected)
                const tenantToUse = tenants.find(t => t.id === newSelectedTenantId)

                if (tenantToUse && newSelectedRoleId) {
                    // Check if current role is still available for this tenant
                    const roleExists = tenantToUse.roles.find(r => r.id === newSelectedRoleId)
                    if (!roleExists) {
                        newSelectedRoleId = tenantToUse.roles.length > 0 ? tenantToUse.roles[0].id : null
                    }
                } else if (tenantToUse && tenantToUse.roles.length > 0) {
                    // No role selected but tenant has roles, select first
                    newSelectedRoleId = tenantToUse.roles[0].id
                }

                set({
                    userTenants: tenants,
                    selectedTenantId: newSelectedTenantId,
                    selectedRoleId: newSelectedRoleId,
                    contextValidated: false
                })
            },

            clearContext: () => {
                set({
                    selectedTenantId: null,
                    selectedRoleId: null,
                    contextValidated: false
                })
            },

            validateContext: () => {
                const state = get()
                if (!state.selectedTenantId || !state.selectedRoleId) {
                    return false
                }

                const tenant = state.userTenants.find(t => t.id === state.selectedTenantId)
                if (!tenant) return false

                const role = tenant.roles.find(r => r.id === state.selectedRoleId)
                if (!role) return false

                set({ contextValidated: true })
                return true
            },

            getCurrentContext: () => {
                const state = get()
                const tenant = state.userTenants.find(t => t.id === state.selectedTenantId) || null
                const role = tenant?.roles.find(r => r.id === state.selectedRoleId) || null
                return { tenant, role }
            },
        }),
        {
            name: 'tenant-context-store',
            partialize: (state) => ({
                selectedTenantId: state.selectedTenantId,
                selectedRoleId: state.selectedRoleId,
                contextValidated: state.contextValidated,
                // Don't persist userTenants - it should always be loaded fresh from profile
            }),
        }
    )
)

// Legacy compatibility - redirect old methods to new interface
export const useLegacyTenantStore = () => {
    const store = useTenantStore()
    return {
        selectedTenantId: store.selectedTenantId,
        userTenants: store.userTenants.map(t => ({ id: t.id, name: t.name })),
        setSelectedTenant: store.setSelectedTenant,
        setUserTenants: (tenants: Array<{ id: string; name: string }>) => {
            // Convert legacy format to new format
            const convertedTenants: TenantContext[] = tenants.map(t => ({
                id: t.id,
                name: t.name,
                roles: [] // Will be populated by auth store
            }))
            store.setUserTenants(convertedTenants)
        }
    }
}
