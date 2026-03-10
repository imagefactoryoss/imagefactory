import PostLoginContextSelector from '@/components/auth/PostLoginContextSelector'
import ContextIndicator from '@/components/common/ContextIndicator'
import useBuildStatusWebSocket from '@/hooks/useBuildStatusWebSocket'
import useNotificationWebSocket from '@/hooks/useNotificationWebSocket'
import useTenantDashboardData from '@/hooks/useTenantDashboardData'
import { useAuthStore } from '@/store/auth'
import { useCapabilitySurfacesStore } from '@/store/capabilitySurfaces'
import { useTenantStore } from '@/store/tenant'
import {
  Activity,
  AlertTriangle,
  Bell,
  Boxes,
  CheckCircle2,
  Clock3,
  Image as ImageIcon,
  RefreshCw,
  Rocket,
  Wifi,
  WifiOff,
} from 'lucide-react'
import React, { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'

const formatRelativeTime = (value?: string) => {
  if (!value) return 'n/a'
  const ts = Date.parse(value)
  if (!Number.isFinite(ts)) return 'n/a'
  const deltaSeconds = Math.max(0, Math.floor((Date.now() - ts) / 1000))
  if (deltaSeconds < 60) return `${deltaSeconds}s ago`
  const minutes = Math.floor(deltaSeconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}

const badgeClass = (status: string) => {
  switch (status) {
    case 'completed':
      return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-200'
    case 'failed':
      return 'bg-rose-100 text-rose-800 dark:bg-rose-900/30 dark:text-rose-200'
    case 'running':
      return 'bg-sky-100 text-sky-800 dark:bg-sky-900/30 dark:text-sky-200'
    case 'queued':
      return 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-200'
    case 'cancelled':
      return 'bg-slate-200 text-slate-800 dark:bg-slate-700 dark:text-slate-200'
    default:
      return 'bg-slate-100 text-slate-800 dark:bg-slate-800 dark:text-slate-200'
  }
}

const DashboardPage: React.FC = () => {
  const { user } = useAuthStore()
  const capabilitySurfaceData = useCapabilitySurfacesStore((state) => state.data)
  const capabilitiesLoadedTenantId = useCapabilitySurfacesStore((state) => state.loadedTenantId)
  const capabilitiesLoading = useCapabilitySurfacesStore((state) => state.isLoading)
  const canAccessRouteKey = useCapabilitySurfacesStore((state) => state.canAccessRouteKey)
  const canRunActionKey = useCapabilitySurfacesStore((state) => state.canRunActionKey)
  const { userTenants, selectedTenantId, selectedRoleId } = useTenantStore()
  const capabilityStateReadyForTenant =
    !!selectedTenantId && !capabilitiesLoading && capabilitiesLoadedTenantId === selectedTenantId
  const effectiveCapabilities = capabilityStateReadyForTenant
    ? capabilitySurfaceData.capabilities
    : {
        build: false,
        quarantine_request: false,
        quarantine_release: false,
        ondemand_image_scanning: false,
      }
  const canUseBuildCapability = capabilityStateReadyForTenant && canAccessRouteKey('builds.list')
  const canCreateBuildCapability = capabilityStateReadyForTenant && canRunActionKey('builds.create')
  const hasAnyOperationalCapability = Object.values(effectiveCapabilities).some(Boolean)
  const capabilityLabelMap: Record<string, string> = {
    build: 'Image Build',
    quarantine_request: 'Quarantine Request',
    quarantine_release: 'Quarantine Release (Admin)',
    ondemand_image_scanning: 'On-Demand Scanning',
  }
  const [showContextSelector, setShowContextSelector] = useState(false)
  const { data, isLoading, isRefreshing, error, refresh, scheduleRefresh, tenantReady } = useTenantDashboardData()

  useEffect(() => {
    if (userTenants && userTenants.length > 1 && (!selectedTenantId || !selectedRoleId)) {
      setShowContextSelector(true)
    }
  }, [selectedRoleId, selectedTenantId, userTenants])

  const buildSocket = useBuildStatusWebSocket({
    enabled: tenantReady,
    onBuildEvent: () => scheduleRefresh(),
  })
  const notificationSocket = useNotificationWebSocket({
    enabled: tenantReady,
    onEvent: () => scheduleRefresh(),
  })

  const socketHealth = useMemo(() => {
    if (!tenantReady) return { label: 'No tenant context', connected: false }
    if (buildSocket.isConnected || notificationSocket.isConnected) {
      return { label: 'Live updates connected', connected: true }
    }
    return { label: 'Live updates reconnecting', connected: false }
  }, [buildSocket.isConnected, notificationSocket.isConnected, tenantReady])

  const hasValidContext = selectedTenantId && selectedRoleId

  return (
    <>
      <PostLoginContextSelector isOpen={showContextSelector} onClose={() => setShowContextSelector(false)} />

      <div className="min-h-screen bg-[radial-gradient(circle_at_top_right,#e0f2fe_0%,#f8fafc_45%,#f8fafc_100%)] px-4 py-6 dark:bg-[radial-gradient(circle_at_top_right,#172554_0%,#020617_45%,#020617_100%)] sm:px-6 lg:px-8">
        <div className="mx-auto max-w-7xl space-y-6">
          <section className="overflow-hidden rounded-2xl border border-slate-200 bg-white/90 p-6 shadow-xl shadow-sky-100/60 backdrop-blur dark:border-slate-700 dark:bg-slate-900/80 dark:shadow-sky-950/20">
            <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
              <div>
                <p className="text-xs font-semibold uppercase tracking-[0.25em] text-sky-600 dark:text-sky-300">Tenant Control Tower</p>
                <h1 className="mt-2 text-3xl font-black tracking-tight text-slate-900 dark:text-white">Dashboard</h1>
                <p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
                  Welcome back, {user?.name || user?.email || 'User'}.
                </p>
                {hasValidContext ? (
                  <div className="mt-3">
                    <ContextIndicator compact />
                  </div>
                ) : null}
              </div>

              <div className="flex flex-wrap items-center gap-2">
                <span
                  className={`inline-flex items-center gap-2 rounded-full px-3 py-1 text-xs font-semibold ${
                    socketHealth.connected
                      ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-200'
                      : 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-200'
                  }`}
                >
                  {socketHealth.connected ? <Wifi className="h-3.5 w-3.5" /> : <WifiOff className="h-3.5 w-3.5" />}
                  {socketHealth.label}
                </span>
                <button
                  onClick={refresh}
                  className="inline-flex items-center gap-2 rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-100 disabled:opacity-60 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
                  disabled={isRefreshing || isLoading}
                >
                  <RefreshCw className={`h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
                  Refresh
                </button>
                {canCreateBuildCapability ? (
                  <Link
                    to="/builds"
                    className="inline-flex items-center gap-2 rounded-lg bg-sky-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-sky-500"
                  >
                    <Rocket className="h-4 w-4" />
                    New Build
                  </Link>
                ) : null}
              </div>
            </div>
          </section>

          {!tenantReady ? (
            <section className="rounded-xl border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-950/30 dark:text-amber-200">
              Select a tenant context to view realtime dashboard data.
            </section>
          ) : null}

          {error ? (
            <section className="rounded-xl border border-rose-300 bg-rose-50 px-4 py-3 text-sm text-rose-900 dark:border-rose-700 dark:bg-rose-950/30 dark:text-rose-200">
              {error}
            </section>
          ) : null}

          <section className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-900">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Operational Entitlements</h2>
              {!capabilityStateReadyForTenant ? (
                <span className="text-xs text-slate-500 dark:text-slate-400">Checking capabilities...</span>
              ) : null}
            </div>
            <div className="mt-3 grid grid-cols-1 gap-2 md:grid-cols-2 xl:grid-cols-5">
              {(Object.keys(effectiveCapabilities) as Array<keyof typeof effectiveCapabilities>).map((key) => {
                const enabled = Boolean(effectiveCapabilities[key])
                return (
                  <div key={key} className="rounded-lg border border-slate-200 px-3 py-2 dark:border-slate-700">
                    <p className="text-xs font-medium text-slate-700 dark:text-slate-300">{capabilityLabelMap[key] || key}</p>
                    <p
                      className={`mt-1 text-xs font-semibold ${
                        enabled
                          ? 'text-emerald-700 dark:text-emerald-300'
                          : 'text-amber-700 dark:text-amber-300'
                      }`}
                    >
                      {enabled ? 'Enabled' : 'Not entitled'}
                    </p>
                  </div>
                )
              })}
            </div>
            {capabilityStateReadyForTenant && !hasAnyOperationalCapability ? (
              <div className="mt-3 space-y-2 rounded-lg border border-amber-300 bg-amber-50 px-3 py-3 text-xs text-amber-900 dark:border-amber-700 dark:bg-amber-950/30 dark:text-amber-100">
                <p className="font-semibold">No operational capabilities are enabled for this tenant.</p>
                <p>Build, project, and quarantine request workflows are currently unavailable. Contact your tenant administrator to request access.</p>
                <p className="text-amber-800 dark:text-amber-200">You can still use: Image Catalog, Notifications, and Profile.</p>
              </div>
            ) : null}
          </section>

          <section className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
            {canUseBuildCapability ? (
              <div className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-900">
                <div className="flex items-center justify-between">
                  <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Projects</p>
                  <Boxes className="h-4 w-4 text-sky-600 dark:text-sky-300" />
                </div>
                <p className="mt-2 text-3xl font-bold text-slate-900 dark:text-white">{isLoading ? '-' : data.stats.totalProjects}</p>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{data.stats.activeProjects} active projects</p>
              </div>
            ) : (
              <div className="rounded-xl border border-dashed border-slate-300 bg-white p-4 dark:border-slate-700 dark:bg-slate-900">
                <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Projects</p>
                <p className="mt-2 text-sm font-medium text-slate-700 dark:text-slate-300">Not entitled</p>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">Enable Image Build capability to view project metrics.</p>
              </div>
            )}

            {canUseBuildCapability ? (
              <div className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-900">
                <div className="flex items-center justify-between">
                  <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Builds Today</p>
                  <Activity className="h-4 w-4 text-violet-600 dark:text-violet-300" />
                </div>
                <p className="mt-2 text-3xl font-bold text-slate-900 dark:text-white">{isLoading ? '-' : data.stats.buildsToday}</p>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{data.stats.successRate}% success rate</p>
              </div>
            ) : (
              <div className="rounded-xl border border-dashed border-slate-300 bg-white p-4 dark:border-slate-700 dark:bg-slate-900">
                <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Builds</p>
                <p className="mt-2 text-sm font-medium text-slate-700 dark:text-slate-300">Not entitled</p>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">Enable Image Build capability to view build metrics.</p>
              </div>
            )}

            {canUseBuildCapability ? (
              <div className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-900">
                <div className="flex items-center justify-between">
                  <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Execution Pressure</p>
                  <Clock3 className="h-4 w-4 text-amber-600 dark:text-amber-300" />
                </div>
                <p className="mt-2 text-3xl font-bold text-slate-900 dark:text-white">{isLoading ? '-' : data.stats.runningBuilds + data.stats.queuedBuilds}</p>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                  {data.stats.runningBuilds} running, {data.stats.queuedBuilds} queued
                </p>
              </div>
            ) : (
              <div className="rounded-xl border border-dashed border-slate-300 bg-white p-4 dark:border-slate-700 dark:bg-slate-900">
                <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Execution Pressure</p>
                <p className="mt-2 text-sm font-medium text-slate-700 dark:text-slate-300">Not entitled</p>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">Build capability required.</p>
              </div>
            )}

            <div className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-900">
              <div className="flex items-center justify-between">
                <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">Unread Notifications</p>
                <Bell className="h-4 w-4 text-emerald-600 dark:text-emerald-300" />
              </div>
              <p className="mt-2 text-3xl font-bold text-slate-900 dark:text-white">{isLoading ? '-' : data.stats.unreadNotifications}</p>
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">User-specific unread feed count</p>
            </div>
          </section>

          <section className="grid grid-cols-1 gap-6 xl:grid-cols-3">
            {canUseBuildCapability ? (
              <article className="xl:col-span-2 rounded-xl border border-slate-200 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-900">
              <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-slate-700">
                <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Recent Builds</h2>
                <Link to="/builds" className="text-xs font-semibold text-sky-600 hover:text-sky-500 dark:text-sky-300 dark:hover:text-sky-200">
                  View all
                </Link>
              </div>
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
                  <thead className="bg-slate-50 dark:bg-slate-800/60">
                    <tr>
                      <th className="px-4 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-300">Build</th>
                      <th className="px-4 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-300">Project</th>
                      <th className="px-4 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-300">Status</th>
                      <th className="px-4 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-300">Duration</th>
                      <th className="px-4 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-300">Created</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                    {isLoading ? (
                      <tr>
                        <td colSpan={5} className="px-4 py-8 text-center text-sm text-slate-500 dark:text-slate-400">
                          Loading recent builds...
                        </td>
                      </tr>
                    ) : data.recentBuilds.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="px-4 py-8 text-center text-sm text-slate-500 dark:text-slate-400">
                          No builds found for this tenant.
                        </td>
                      </tr>
                    ) : (
                      data.recentBuilds.map((build) => (
                        <tr key={build.id} className="hover:bg-slate-50/70 dark:hover:bg-slate-800/50">
                          <td className="px-4 py-3 text-sm font-medium text-slate-900 dark:text-white">
                            <Link to={`/builds/${build.id}`} className="hover:text-sky-600 dark:hover:text-sky-300">
                              {build.name}
                            </Link>
                          </td>
                          <td className="px-4 py-3 text-sm text-slate-600 dark:text-slate-300">{build.projectName}</td>
                          <td className="px-4 py-3 text-sm">
                            <span className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold ${badgeClass(build.status)}`}>
                              {build.status}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-sm text-slate-600 dark:text-slate-300">{build.durationLabel}</td>
                          <td className="px-4 py-3 text-sm text-slate-500 dark:text-slate-400">{formatRelativeTime(build.createdAt)}</td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </article>
            ) : (
              <article className="xl:col-span-2 rounded-xl border border-dashed border-slate-300 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-900">
                <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Recent Builds</h2>
                <p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
                  Build history is unavailable because this tenant is not entitled for Image Build capability.
                </p>
              </article>
            )}

            <div className="space-y-6">
              <article className="rounded-xl border border-slate-200 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-900">
                <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-slate-700">
                  <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Most Active Projects</h2>
                  {canUseBuildCapability ? (
                    <Link to="/projects" className="text-xs font-semibold text-sky-600 hover:text-sky-500 dark:text-sky-300 dark:hover:text-sky-200">
                      View all
                    </Link>
                  ) : null}
                </div>
                <div className="space-y-2 p-3">
                  {!canUseBuildCapability ? (
                    <p className="px-2 py-6 text-center text-sm text-slate-500 dark:text-slate-400">
                      Project activity is hidden because Image Build capability is not enabled.
                    </p>
                  ) : isLoading ? (
                    <p className="px-2 py-6 text-center text-sm text-slate-500 dark:text-slate-400">Loading projects...</p>
                  ) : data.mostActiveProjects.length === 0 ? (
                    <p className="px-2 py-6 text-center text-sm text-slate-500 dark:text-slate-400">No project activity yet.</p>
                  ) : (
                    data.mostActiveProjects.map((project) => (
                      <Link
                        key={project.id}
                        to={`/projects/${project.id}`}
                        className="block rounded-lg border border-slate-200 p-3 transition hover:border-sky-300 hover:bg-sky-50 dark:border-slate-700 dark:hover:border-sky-700 dark:hover:bg-sky-950/20"
                      >
                        <div className="flex items-center justify-between gap-3">
                          <p className="truncate text-sm font-semibold text-slate-900 dark:text-white">{project.name}</p>
                          <span className="text-xs font-semibold text-slate-500 dark:text-slate-400">{project.buildCount} builds</span>
                        </div>
                        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                          Last build {formatRelativeTime(project.lastBuildAt)}
                        </p>
                      </Link>
                    ))
                  )}
                </div>
              </article>

              <article className="rounded-xl border border-slate-200 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-900">
                <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-slate-700">
                  <h2 className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">
                    <ImageIcon className="h-4 w-4" />
                    Recent Images
                  </h2>
                  <Link to="/images" className="text-xs font-semibold text-sky-600 hover:text-sky-500 dark:text-sky-300 dark:hover:text-sky-200">
                    Catalog
                  </Link>
                </div>
                <div className="space-y-2 p-3">
                  {isLoading ? (
                    <p className="px-2 py-6 text-center text-sm text-slate-500 dark:text-slate-400">Loading images...</p>
                  ) : data.recentImages.length === 0 ? (
                    <p className="px-2 py-6 text-center text-sm text-slate-500 dark:text-slate-400">No recently published images.</p>
                  ) : (
                    data.recentImages.map((image) => (
                      <Link
                        key={image.id}
                        to={`/images/${image.id}`}
                        className="block rounded-lg border border-slate-200 p-3 transition hover:border-emerald-300 hover:bg-emerald-50 dark:border-slate-700 dark:hover:border-emerald-700 dark:hover:bg-emerald-950/20"
                      >
                        <p className="truncate text-sm font-semibold text-slate-900 dark:text-white">{image.name}</p>
                        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                          {image.visibility} • {image.tags?.slice(0, 2).join(', ') || 'untagged'}
                        </p>
                      </Link>
                    ))
                  )}
                </div>
              </article>
            </div>
          </section>

          <section className="grid grid-cols-1 gap-4 lg:grid-cols-3">
            {canUseBuildCapability ? (
              <>
                <article className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-900">
                  <h3 className="text-sm font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Failure Watch</h3>
                  <p className="mt-2 flex items-center gap-2 text-2xl font-bold text-rose-600 dark:text-rose-300">
                    <AlertTriangle className="h-5 w-5" />
                    {isLoading ? '-' : data.stats.failedBuilds}
                  </p>
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">Failed builds in current dataset window.</p>
                </article>

                <article className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-900">
                  <h3 className="text-sm font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Completion Pulse</h3>
                  <p className="mt-2 flex items-center gap-2 text-2xl font-bold text-emerald-600 dark:text-emerald-300">
                    <CheckCircle2 className="h-5 w-5" />
                    {isLoading ? '-' : data.stats.completedBuilds}
                  </p>
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">Completed builds in current dataset window.</p>
                </article>
              </>
            ) : null}

            <article className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-900">
              <h3 className="text-sm font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Last Sync</h3>
              <p className="mt-2 text-lg font-bold text-slate-900 dark:text-white">{formatRelativeTime(data.lastUpdatedAt)}</p>
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">Realtime + 45s fallback refresh while dashboard is open.</p>
            </article>
          </section>
        </div>
      </div>
    </>
  )
}

export default DashboardPage
