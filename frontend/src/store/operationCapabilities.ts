import { operationCapabilityService } from '@/services/operationCapabilityService'
import type { OperationCapabilitiesConfig } from '@/types'
import { create } from 'zustand'

interface OperationCapabilitiesState {
    capabilities: OperationCapabilitiesConfig
    isLoading: boolean
    loadedTenantId: string | null
    error: string | null
    refreshForTenant: (tenantId: string) => Promise<void>
    reset: () => void
}

const defaultCapabilities = operationCapabilityService.defaultConfig()

export const useOperationCapabilitiesStore = create<OperationCapabilitiesState>((set) => ({
    capabilities: defaultCapabilities,
    isLoading: false,
    loadedTenantId: null,
    error: null,

    refreshForTenant: async (tenantId: string) => {
        if (!tenantId) {
            set({
                capabilities: operationCapabilityService.defaultConfig(),
                loadedTenantId: null,
                error: null,
                isLoading: false,
            })
            return
        }

        set({ isLoading: true, error: null })
        try {
            const capabilities = await operationCapabilityService.getTenantCapabilities()
            set({
                capabilities,
                loadedTenantId: tenantId,
                isLoading: false,
                error: null,
            })
        } catch (err) {
            set({
                capabilities: operationCapabilityService.defaultConfig(),
                loadedTenantId: tenantId,
                isLoading: false,
                error: err instanceof Error ? err.message : 'Failed to load operation capabilities',
            })
        }
    },

    reset: () =>
        set({
            capabilities: operationCapabilityService.defaultConfig(),
            isLoading: false,
            loadedTenantId: null,
            error: null,
        }),
}))
