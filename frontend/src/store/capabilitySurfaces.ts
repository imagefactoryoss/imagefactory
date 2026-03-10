import { capabilitySurfaceService } from '@/services/capabilitySurfaceService'
import type { CapabilitySurfacesResponse } from '@/types'
import { create } from 'zustand'

interface CapabilitySurfacesState {
    data: CapabilitySurfacesResponse
    isLoading: boolean
    loadedTenantId: string | null
    error: string | null
    refreshForTenant: (tenantId: string) => Promise<void>
    canViewNavKey: (navKey: string) => boolean
    canAccessRouteKey: (routeKey: string) => boolean
    canRunActionKey: (actionKey: string) => boolean
    reset: () => void
}

const defaultData = capabilitySurfaceService.defaultResponse()

export const useCapabilitySurfacesStore = create<CapabilitySurfacesState>((set, get) => ({
    data: defaultData,
    isLoading: false,
    loadedTenantId: null,
    error: null,

    refreshForTenant: async (tenantId: string) => {
        if (!tenantId) {
            set({
                data: capabilitySurfaceService.defaultResponse(),
                loadedTenantId: null,
                isLoading: false,
                error: null,
            })
            return
        }

        set({ isLoading: true, error: null })
        try {
            const data = await capabilitySurfaceService.getTenantCapabilitySurfaces()
            set({
                data,
                loadedTenantId: tenantId,
                isLoading: false,
                error: null,
            })
        } catch (err) {
            set({
                data: capabilitySurfaceService.defaultResponse(),
                loadedTenantId: tenantId,
                isLoading: false,
                error: err instanceof Error ? err.message : 'Failed to load capability surfaces',
            })
        }
    },

    canViewNavKey: (navKey: string) => {
        const state = get()
        return state.data.surfaces.nav_keys.includes(navKey)
    },

    canAccessRouteKey: (routeKey: string) => {
        const state = get()
        return state.data.surfaces.route_keys.includes(routeKey)
    },

    canRunActionKey: (actionKey: string) => {
        const state = get()
        return state.data.surfaces.action_keys.includes(actionKey)
    },

    reset: () => {
        set({
            data: capabilitySurfaceService.defaultResponse(),
            isLoading: false,
            loadedTenantId: null,
            error: null,
        })
    },
}))
