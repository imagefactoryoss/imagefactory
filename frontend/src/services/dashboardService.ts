import { api } from './api'

export interface TenantDashboardSummaryStats {
  total_projects: number
  active_projects: number
  builds_today: number
  success_rate: number
  running_builds: number
  queued_builds: number
  completed_builds: number
  failed_builds: number
  unread_notifications: number
}

export interface TenantDashboardSummaryResponse {
  stats: TenantDashboardSummaryStats
  last_updated_at: string
}

export interface TenantDashboardRecentBuild {
  id: string
  name: string
  project_name: string
  status: string
  created_at: string
  duration_label: string
}

export interface TenantDashboardProjectActivity {
  id: string
  name: string
  build_count: number
  last_build_at?: string
}

export interface TenantDashboardRecentImage {
  id: string
  name: string
  visibility: string
  tags: string[]
  created_at: string
  updated_at: string
}

export interface TenantDashboardActivityResponse {
  recent_builds: TenantDashboardRecentBuild[]
  most_active_projects: TenantDashboardProjectActivity[]
  recent_images: TenantDashboardRecentImage[]
  last_updated_at: string
}

export const dashboardService = {
  async getTenantSummary(): Promise<TenantDashboardSummaryResponse> {
    const response = await api.get('/dashboard/tenant/summary')
    return response.data
  },

  async getTenantActivity(): Promise<TenantDashboardActivityResponse> {
    const response = await api.get('/dashboard/tenant/activity')
    return response.data
  },
}

