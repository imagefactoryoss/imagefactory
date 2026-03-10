import type { EPRRegistrationRequest } from '@/types'
import { api } from './api'

type ListResponse = {
  data: EPRRegistrationRequest[]
  pagination?: {
    page: number
    limit: number
  }
}

type ItemResponse = {
  data: EPRRegistrationRequest
}

type BulkResponse = {
  data: EPRRegistrationRequest[]
  count?: number
}

export interface CreateEPRRegistrationRequestInput {
  eprRecordId: string
  productName: string
  technologyName: string
  businessJustification?: string
}

class EPRRegistrationService {
  async createRequest(input: CreateEPRRegistrationRequestInput): Promise<EPRRegistrationRequest> {
    const response = await api.post<ItemResponse>('/epr/registration-requests', {
      epr_record_id: input.eprRecordId,
      product_name: input.productName,
      technology_name: input.technologyName,
      business_justification: input.businessJustification || undefined,
    })
    return response.data.data
  }

  async listTenantRequests(params?: { limit?: number; page?: number; status?: string }): Promise<EPRRegistrationRequest[]> {
    const query = new URLSearchParams({
      limit: String(params?.limit ?? 25),
      page: String(params?.page ?? 1),
    })
    if (params?.status) {
      query.set('status', params.status)
    }
    const response = await api.get<ListResponse>(`/epr/registration-requests?${query.toString()}`)
    return response.data.data || []
  }

  async withdrawRequest(id: string, reason?: string): Promise<EPRRegistrationRequest> {
    const response = await api.post<ItemResponse>(`/epr/registration-requests/${id}/withdraw`, {
      reason: reason?.trim() || undefined,
    })
    return response.data.data
  }

  async listAdminRequests(params?: { limit?: number; page?: number; status?: string }): Promise<EPRRegistrationRequest[]> {
    const query = new URLSearchParams({
      limit: String(params?.limit ?? 50),
      page: String(params?.page ?? 1),
    })
    if (params?.status) {
      query.set('status', params.status)
    }
    const response = await api.get<ListResponse>(`/admin/epr/registration-requests?${query.toString()}`)
    return response.data.data || []
  }

  async approveRequest(id: string, reason?: string): Promise<EPRRegistrationRequest> {
    const response = await api.post<ItemResponse>(`/admin/epr/registration-requests/${id}/approve`, {
      reason: reason?.trim() || undefined,
    })
    return response.data.data
  }

  async rejectRequest(id: string, reason?: string): Promise<EPRRegistrationRequest> {
    const response = await api.post<ItemResponse>(`/admin/epr/registration-requests/${id}/reject`, {
      reason: reason?.trim() || undefined,
    })
    return response.data.data
  }

  async suspendRequest(id: string, reason?: string): Promise<EPRRegistrationRequest> {
    const response = await api.post<ItemResponse>(`/admin/epr/registration-requests/${id}/suspend`, {
      reason: reason?.trim() || undefined,
    })
    return response.data.data
  }

  async reactivateRequest(id: string, reason?: string): Promise<EPRRegistrationRequest> {
    const response = await api.post<ItemResponse>(`/admin/epr/registration-requests/${id}/reactivate`, {
      reason: reason?.trim() || undefined,
    })
    return response.data.data
  }

  async revalidateRequest(id: string, reason?: string): Promise<EPRRegistrationRequest> {
    const response = await api.post<ItemResponse>(`/admin/epr/registration-requests/${id}/revalidate`, {
      reason: reason?.trim() || undefined,
    })
    return response.data.data
  }

  async bulkSuspendRequests(requestIDs: string[], reason?: string): Promise<EPRRegistrationRequest[]> {
    const response = await api.post<BulkResponse>('/admin/epr/registration-requests/bulk/suspend', {
      request_ids: requestIDs,
      reason: reason?.trim() || undefined,
    })
    return response.data.data || []
  }

  async bulkReactivateRequests(requestIDs: string[], reason?: string): Promise<EPRRegistrationRequest[]> {
    const response = await api.post<BulkResponse>('/admin/epr/registration-requests/bulk/reactivate', {
      request_ids: requestIDs,
      reason: reason?.trim() || undefined,
    })
    return response.data.data || []
  }

  async bulkRevalidateRequests(requestIDs: string[], reason?: string): Promise<EPRRegistrationRequest[]> {
    const response = await api.post<BulkResponse>('/admin/epr/registration-requests/bulk/revalidate', {
      request_ids: requestIDs,
      reason: reason?.trim() || undefined,
    })
    return response.data.data || []
  }
}

export const eprRegistrationService = new EPRRegistrationService()
