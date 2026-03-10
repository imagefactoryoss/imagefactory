import { dashboardService } from '@/services/dashboardService'
import { useTenantStore } from '@/store/tenant'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

export interface DashboardStats {
  totalProjects: number
  activeProjects: number
  buildsToday: number
  successRate: number
  runningBuilds: number
  queuedBuilds: number
  completedBuilds: number
  failedBuilds: number
  unreadNotifications: number
}

export interface DashboardBuildRow {
  id: string
  name: string
  projectName: string
  status: string
  createdAt: string
  durationLabel: string
}

export interface DashboardProjectActivity {
  id: string
  name: string
  buildCount: number
  lastBuildAt?: string
}

export interface DashboardImageRow {
  id: string
  name: string
  visibility: string
  tags: string[]
  createdAt: string
  updatedAt: string
}

export interface TenantDashboardData {
  stats: DashboardStats
  recentBuilds: DashboardBuildRow[]
  mostActiveProjects: DashboardProjectActivity[]
  recentImages: DashboardImageRow[]
  lastUpdatedAt?: string
}

const EMPTY_DATA: TenantDashboardData = {
  stats: {
    totalProjects: 0,
    activeProjects: 0,
    buildsToday: 0,
    successRate: 0,
    runningBuilds: 0,
    queuedBuilds: 0,
    completedBuilds: 0,
    failedBuilds: 0,
    unreadNotifications: 0,
  },
  recentBuilds: [],
  mostActiveProjects: [],
  recentImages: [],
}

export const useTenantDashboardData = () => {
  const selectedTenantId = useTenantStore((state) => state.selectedTenantId)
  const [data, setData] = useState<TenantDashboardData>(EMPTY_DATA)
  const [isLoading, setIsLoading] = useState(true)
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const refreshTimerRef = useRef<number | undefined>(undefined)

  const load = useCallback(
    async (mode: 'initial' | 'refresh' = 'refresh') => {
      if (!selectedTenantId) {
        setData(EMPTY_DATA)
        setError(null)
        setIsLoading(false)
        setIsRefreshing(false)
        return
      }

      if (mode === 'initial') {
        setIsLoading(true)
      } else {
        setIsRefreshing(true)
      }

      try {
        const [summary, activity] = await Promise.all([
          dashboardService.getTenantSummary(),
          dashboardService.getTenantActivity(),
        ])

        setData({
          stats: {
            totalProjects: summary.stats?.total_projects || 0,
            activeProjects: summary.stats?.active_projects || 0,
            buildsToday: summary.stats?.builds_today || 0,
            successRate: summary.stats?.success_rate || 0,
            runningBuilds: summary.stats?.running_builds || 0,
            queuedBuilds: summary.stats?.queued_builds || 0,
            completedBuilds: summary.stats?.completed_builds || 0,
            failedBuilds: summary.stats?.failed_builds || 0,
            unreadNotifications: summary.stats?.unread_notifications || 0,
          },
          recentBuilds: (activity.recent_builds || []).map((row) => ({
            id: row.id,
            name: row.name,
            projectName: row.project_name || 'Unknown project',
            status: row.status,
            createdAt: row.created_at,
            durationLabel: row.duration_label || 'n/a',
          })),
          mostActiveProjects: (activity.most_active_projects || []).map((row) => ({
            id: row.id,
            name: row.name,
            buildCount: row.build_count || 0,
            lastBuildAt: row.last_build_at,
          })),
          recentImages: (activity.recent_images || []).map((row) => ({
            id: row.id,
            name: row.name,
            visibility: row.visibility,
            tags: row.tags || [],
            createdAt: row.created_at,
            updatedAt: row.updated_at,
          })),
          lastUpdatedAt: activity.last_updated_at || summary.last_updated_at || new Date().toISOString(),
        })
        setError(null)
      } catch (loadError) {
        console.error('Failed to load tenant dashboard data', loadError)
        setError('Failed to load tenant dashboard data')
      } finally {
        setIsLoading(false)
        setIsRefreshing(false)
      }
    },
    [selectedTenantId]
  )

  const scheduleRefresh = useCallback(() => {
    if (refreshTimerRef.current) {
      window.clearTimeout(refreshTimerRef.current)
    }
    refreshTimerRef.current = window.setTimeout(() => {
      load('refresh')
    }, 450)
  }, [load])

  useEffect(() => {
    load('initial')
    return () => {
      if (refreshTimerRef.current) {
        window.clearTimeout(refreshTimerRef.current)
        refreshTimerRef.current = undefined
      }
    }
  }, [load])

  useEffect(() => {
    const poll = window.setInterval(() => {
      load('refresh')
    }, 45000)
    return () => window.clearInterval(poll)
  }, [load])

  return useMemo(
    () => ({
      data,
      isLoading,
      isRefreshing,
      error,
      refresh: () => load('refresh'),
      scheduleRefresh,
      tenantReady: Boolean(selectedTenantId),
    }),
    [data, error, isLoading, isRefreshing, load, scheduleRefresh, selectedTenantId]
  )
}

export default useTenantDashboardData
