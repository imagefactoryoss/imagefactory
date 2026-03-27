import { api } from './api'

export interface VMImageCatalogItem {
  execution_id: string
  build_id: string
  project_id: string
  project_name: string
  build_number: number
  build_status: string
  execution_status: string
  created_at: string
  started_at?: string
  completed_at?: string
  target_provider: string
  target_profile_id: string
  provider_artifact_identifiers: Record<string, string[]>
  artifact_values: string[]
  lifecycle_state: string
  lifecycle_last_action_at?: string
  lifecycle_last_action_by?: string
  lifecycle_last_reason?: string
  lifecycle_history?: Array<{
    state: string
    reason?: string
    actor_id?: string
    at?: string
  }>
}

export interface VMImageCatalogListResponse {
  data: VMImageCatalogItem[]
  total_count: number
  limit: number
  offset: number
}

export interface VMImageCatalogDetailResponse {
  data: VMImageCatalogItem
}

export interface VMImageLifecycleActionResponse {
  data: VMImageCatalogItem
  message?: string
}

export const vmImageService = {
  async list(params?: {
    limit?: number
    offset?: number
    provider?: string
    status?: string
    search?: string
  }): Promise<VMImageCatalogListResponse> {
    const response = await api.get('/images/vm', { params })
    return response.data
  },

  async get(executionId: string): Promise<VMImageCatalogDetailResponse> {
    const response = await api.get(`/images/vm/${executionId}`)
    return response.data
  },

  async promote(
    executionId: string,
  ): Promise<VMImageLifecycleActionResponse> {
    const response = await api.post(`/images/vm/${executionId}/promote`)
    return response.data
  },

  async deprecate(
    executionId: string,
    reason?: string,
  ): Promise<VMImageLifecycleActionResponse> {
    const response = await api.post(`/images/vm/${executionId}/deprecate`, {
      reason,
    })
    return response.data
  },

  async remove(
    executionId: string,
    reason?: string,
  ): Promise<VMImageLifecycleActionResponse> {
    const response = await api.delete(`/images/vm/${executionId}`, {
      data: {
        reason,
      },
    })
    return response.data
  },
}
