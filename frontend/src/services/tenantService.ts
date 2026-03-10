import {
    ApiResponse,
    CreateTenantRequest,
    PaginatedResponse,
    Tenant,
    TenantFilters,
    UpdateTenantRequest
} from '@/types'
import api from './api'

export class TenantService {
    private static baseUrl = '/tenants'

    // Get all tenants with optional filtering and pagination
    static async getTenants(
        page = 1,
        limit = 20,
        filters?: TenantFilters
    ): Promise<PaginatedResponse<Tenant>> {
        const params = new URLSearchParams()
        params.append('page', page.toString())
        params.append('limit', limit.toString())

        if (filters?.status?.length) {
            filters.status.forEach(status => params.append('status', status))
        }
        if (filters?.search) {
            params.append('search', filters.search)
        }

        const response = await api.get<PaginatedResponse<Tenant>>(
            `${this.baseUrl}?${params.toString()}`
        )
        return response.data
    }

    // Get a specific tenant by ID
    static async getTenant(id: string): Promise<Tenant> {
        const response = await api.get<ApiResponse<Tenant>>(`${this.baseUrl}/${id}`)
        return response.data.data
    }

    // Get tenant by slug
    static async getTenantBySlug(slug: string): Promise<Tenant> {
        const response = await api.get<ApiResponse<Tenant>>(`${this.baseUrl}/slug/${slug}`)
        return response.data.data
    }

    // Create a new tenant
    static async createTenant(data: CreateTenantRequest): Promise<Tenant> {
        const response = await api.post<ApiResponse<Tenant>>(this.baseUrl, data)
        return response.data.data
    }

    // Update an existing tenant
    static async updateTenant(id: string, data: UpdateTenantRequest): Promise<Tenant> {
        const response = await api.put<ApiResponse<Tenant>>(`${this.baseUrl}/${id}`, data)
        return response.data.data
    }

    // Activate a tenant
    static async activateTenant(id: string): Promise<void> {
        await api.post(`${this.baseUrl}/${id}/activate`)
    }

    // Suspend a tenant
    static async suspendTenant(id: string): Promise<void> {
        await api.post(`${this.baseUrl}/${id}/suspend`)
    }

    // Delete a tenant
    static async deleteTenant(id: string): Promise<void> {
        await api.delete(`${this.baseUrl}/${id}`)
    }

    // Check if tenant slug is available
    static async checkSlugAvailability(slug: string): Promise<boolean> {
        try {
            const response = await api.get<ApiResponse<{ available: boolean }>>(
                `${this.baseUrl}/check-slug/${slug}`
            )
            return response.data.data.available
        } catch (error) {
            return false
        }
    }

    // Get tenant statistics
    static async getTenantStats(id: string): Promise<any> {
        const response = await api.get<ApiResponse<any>>(`${this.baseUrl}/${id}/stats`)
        return response.data.data
    }

    // Get tenant usage
    static async getTenantUsage(id: string): Promise<any> {
        const response = await api.get<ApiResponse<any>>(`${this.baseUrl}/${id}/usage`)
        return response.data.data
    }
}