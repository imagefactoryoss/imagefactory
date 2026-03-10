import type { CapabilitySurfacesResponse, OperationCapabilitiesConfig } from '@/types'
import { api } from './api'

const defaultResponse: CapabilitySurfacesResponse = {
    tenant_id: '',
    version: '2026-02-20',
    capabilities: {
        build: false,
        quarantine_request: false,
        quarantine_release: false,
        ondemand_image_scanning: false,
    },
    surfaces: {
        nav_keys: [],
        route_keys: [],
        action_keys: [],
    },
    denials: {},
}

export const capabilitySurfaceService = {
    defaultResponse(): CapabilitySurfacesResponse {
        return {
            ...defaultResponse,
            capabilities: { ...defaultResponse.capabilities },
            surfaces: {
                nav_keys: [...defaultResponse.surfaces.nav_keys],
                route_keys: [...defaultResponse.surfaces.route_keys],
                action_keys: [...defaultResponse.surfaces.action_keys],
            },
            denials: { ...defaultResponse.denials },
        }
    },

    async getTenantCapabilitySurfaces(): Promise<CapabilitySurfacesResponse> {
        try {
            const response = await api.get<CapabilitySurfacesResponse>('/settings/capability-surfaces')
            return response.data
        } catch (err: any) {
            // Compatibility path: older backend may not expose capability-surfaces yet.
            if (err?.response?.status === 404) {
                const fallback = await api.get<OperationCapabilitiesConfig>('/settings/operation-capabilities')
                return deriveFromOperationCapabilities(fallback.data)
            }
            throw err
        }
    },

    async getAdminCapabilitySurfaces(tenantId: string): Promise<CapabilitySurfacesResponse> {
        try {
            const response = await api.get<CapabilitySurfacesResponse>('/admin/settings/capability-surfaces', {
                params: {
                    tenant_id: tenantId,
                },
            })
            return response.data
        } catch (err: any) {
            if (err?.response?.status === 404) {
                const fallback = await api.get<OperationCapabilitiesConfig>('/admin/settings/operation-capabilities', {
                    params: { tenant_id: tenantId },
                })
                return deriveFromOperationCapabilities(fallback.data, tenantId)
            }
            throw err
        }
    },
}

function deriveFromOperationCapabilities(
    cfg: OperationCapabilitiesConfig,
    tenantId: string = ''
): CapabilitySurfacesResponse {
    const capabilities: OperationCapabilitiesConfig = {
        build: !!cfg?.build,
        quarantine_request: !!cfg?.quarantine_request,
        quarantine_release: !!cfg?.quarantine_release,
        ondemand_image_scanning: !!cfg?.ondemand_image_scanning,
    }

    const navKeys = ['dashboard', 'notifications', 'images']
    const routeKeys = ['dashboard.view', 'images.list', 'images.detail', 'profile.view', 'notifications.view']
    const actionKeys = ['images.view_catalog']

    if (capabilities.build) {
        navKeys.push('projects', 'builds')
        routeKeys.push(
            'projects.list',
            'projects.create',
            'projects.detail',
            'projects.edit',
            'builds.list',
            'builds.create',
            'builds.detail'
        )
        actionKeys.push('builds.create', 'projects.manage')
    }

    if (capabilities.build || capabilities.quarantine_request) {
        navKeys.push('auth_management')
        routeKeys.push('settings.auth')
        actionKeys.push('settings.auth.manage')
    }

    if (capabilities.quarantine_request) {
        navKeys.push('quarantine_requests')
        routeKeys.push('quarantine.request.list', 'quarantine.request.create')
        actionKeys.push('quarantine.request.create')
    }

    if (capabilities.quarantine_release) {
        routeKeys.push('quarantine.release')
        actionKeys.push('quarantine.release')
    }

    if (capabilities.ondemand_image_scanning) {
        routeKeys.push('images.scan.ondemand')
        actionKeys.push('images.scan.ondemand')
    }

    return {
        tenant_id: tenantId,
        version: 'compat-derive-v1',
        capabilities,
        surfaces: {
            nav_keys: [...new Set(navKeys)].sort(),
            route_keys: [...new Set(routeKeys)].sort(),
            action_keys: [...new Set(actionKeys)].sort(),
        },
        denials: {},
    }
}
