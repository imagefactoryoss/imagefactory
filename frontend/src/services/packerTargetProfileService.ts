import {
  PackerTargetProfile,
  PackerTargetProvider,
  PackerTargetValidationResult,
} from '@/types'
import api, { getErrorMessage } from './api'

export interface CreatePackerTargetProfileRequest {
  tenant_id?: string
  is_global?: boolean
  name: string
  provider: PackerTargetProvider
  description?: string
  secret_ref: string
  options?: Record<string, any>
}

export interface UpdatePackerTargetProfileRequest {
  name?: string
  is_global?: boolean
  description?: string
  secret_ref?: string
  options?: Record<string, any>
}

export const packerTargetProfileService = {
  async list(params?: {
    provider?: PackerTargetProvider
    tenant_id?: string
    all_tenants?: boolean
  }): Promise<PackerTargetProfile[]> {
    try {
      const query = new URLSearchParams()
      if (params?.provider) query.append('provider', params.provider)
      if (params?.tenant_id) query.append('tenant_id', params.tenant_id)
      if (params?.all_tenants) query.append('all_tenants', 'true')
      const suffix = query.toString() ? `?${query.toString()}` : ''
      const response = await api.get(`/admin/packer-target-profiles${suffix}`)
      return response.data?.profiles || []
    } catch (error: any) {
      throw new Error(getErrorMessage(error) || 'Failed to list Packer target profiles')
    }
  },

  async create(payload: CreatePackerTargetProfileRequest): Promise<PackerTargetProfile> {
    try {
      const response = await api.post('/admin/packer-target-profiles', payload)
      return response.data
    } catch (error: any) {
      throw new Error(getErrorMessage(error) || 'Failed to create Packer target profile')
    }
  },

  async update(id: string, payload: UpdatePackerTargetProfileRequest): Promise<PackerTargetProfile> {
    try {
      const response = await api.put(`/admin/packer-target-profiles/${id}`, payload)
      return response.data
    } catch (error: any) {
      throw new Error(getErrorMessage(error) || 'Failed to update Packer target profile')
    }
  },

  async delete(id: string): Promise<void> {
    try {
      await api.delete(`/admin/packer-target-profiles/${id}`)
    } catch (error: any) {
      throw new Error(getErrorMessage(error) || 'Failed to delete Packer target profile')
    }
  },

  async validate(id: string): Promise<PackerTargetValidationResult> {
    try {
      const response = await api.post(`/admin/packer-target-profiles/${id}/validate`)
      return response.data
    } catch (error: any) {
      throw new Error(getErrorMessage(error) || 'Failed to validate Packer target profile')
    }
  },
}
