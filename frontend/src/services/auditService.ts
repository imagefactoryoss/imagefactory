import { AuditEvent } from '@/types/audit'
import { api } from './api'

export interface AuditEventsResponse {
    events: AuditEvent[]
    total: number
    limit: number
    offset: number
}

export interface AuditEventsFilters {
    user_id?: string
    event_type?: string
    severity?: string
    resource?: string
    action?: string
    start_time?: string
    end_time?: string
    search?: string
    limit?: number
    offset?: number
}

export const auditService = {
    async getAuditEvents(filters: AuditEventsFilters = {}): Promise<AuditEventsResponse> {
        const params = new URLSearchParams()
        // Admin audit page should include cross-tenant visibility for system admins.
        // Backend enforces authorization and ignores this for non-system-admin users.
        params.append('all_tenants', 'true')

        // Add filters to query params
        Object.entries(filters).forEach(([key, value]) => {
            if (value !== undefined && value !== null && value !== '') {
                params.append(key, value.toString())
            }
        })

        const response = await api.get(`/audit-events?${params.toString()}`)
        return response.data
    },

    async getAuditEvent(id: string): Promise<AuditEvent> {
        const response = await api.get(`/audit-events/${id}?all_tenants=true`)
        return response.data
    }
}
