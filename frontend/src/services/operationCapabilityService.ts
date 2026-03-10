import type { OperationCapabilitiesConfig } from '../types'
import { api } from './api'

const defaultOperationCapabilities: OperationCapabilitiesConfig = {
    build: false,
    quarantine_request: false,
    quarantine_release: false,
    ondemand_image_scanning: false,
}

const normalizeOperationCapabilities = (raw?: Partial<OperationCapabilitiesConfig> | null): OperationCapabilitiesConfig => ({
    build: !!raw?.build,
    quarantine_request: !!raw?.quarantine_request,
    quarantine_release: !!raw?.quarantine_release,
    ondemand_image_scanning: !!raw?.ondemand_image_scanning,
})

export const operationCapabilityService = {
    defaultConfig(): OperationCapabilitiesConfig {
        return { ...defaultOperationCapabilities }
    },

    async getTenantCapabilities(): Promise<OperationCapabilitiesConfig> {
        const response = await api.get<OperationCapabilitiesConfig>('/settings/operation-capabilities')
        return normalizeOperationCapabilities(response.data)
    },

    async getAdminCapabilities(options?: { tenantId?: string; globalDefault?: boolean }): Promise<OperationCapabilitiesConfig> {
        const params: Record<string, string | boolean> = {}
        if (options?.globalDefault) {
            params.all_tenants = true
        } else if (options?.tenantId) {
            params.tenant_id = options.tenantId
        }
        const response = await api.get<OperationCapabilitiesConfig>('/admin/settings/operation-capabilities', { params })
        return normalizeOperationCapabilities(response.data)
    },

    async updateAdminCapabilities(
        config: OperationCapabilitiesConfig,
        options?: { tenantId?: string; globalDefault?: boolean; changeReason?: string }
    ): Promise<OperationCapabilitiesConfig> {
        const params: Record<string, string | boolean> = {}
        if (options?.globalDefault) {
            params.all_tenants = true
        } else if (options?.tenantId) {
            params.tenant_id = options.tenantId
        }
        if (options?.changeReason) {
            params.change_reason = options.changeReason
        }
        const response = await api.put<OperationCapabilitiesConfig>('/admin/settings/operation-capabilities', config, { params })
        return normalizeOperationCapabilities(response.data)
    },
}
