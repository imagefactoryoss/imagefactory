import Drawer from '@/components/ui/Drawer'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import { adminService } from '@/services/adminService'
import { eprRegistrationService } from '@/services/eprRegistrationService'
import { ImageImportApiError, imageImportService } from '@/services/imageImportService'
import type { EPRRegistrationRequest, ImageImportRequest, ReleaseGovernancePolicyConfig, ReleaseMetricsSnapshot } from '@/types'
import { getImportDiagnostic, getImportDiagnosticClasses, getImportProgressLabel, getImportRemediationHint, getImportSyncStateLabel, hasMeaningfulJSONEvidence } from '@/utils/imageImportDiagnostics'
import React, { useCallback, useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import { Link, useLocation } from 'react-router-dom'

const mapReviewerErrorMessage = (err: unknown, fallback: string) => {
  if (err instanceof ImageImportApiError) {
    if (err.status === 403) {
      return 'You do not have permission to review quarantine approvals. Contact the platform administrator to verify Security Reviewer access.'
    }
    return err.message || fallback
  }
  if (err instanceof Error && err.message) {
    return err.message
  }
  return fallback
}

interface QuarantineReviewWorkbenchPageProps {
  mode?: 'all' | 'dashboard' | 'requests' | 'epr'
}

const QuarantineReviewWorkbenchPage: React.FC<QuarantineReviewWorkbenchPageProps> = ({ mode = 'all' }) => {
  const location = useLocation()
  const confirmDialog = useConfirmDialog()
  const [rows, setRows] = useState<ImageImportRequest[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [permissionDenied, setPermissionDenied] = useState(false)
  const [selected, setSelected] = useState<ImageImportRequest | null>(null)
  const [selectedEpr, setSelectedEpr] = useState<EPRRegistrationRequest | null>(null)
  const [actionBusy, setActionBusy] = useState<Record<string, boolean>>({})
  const [statusFilter, setStatusFilter] = useState<'all' | ImageImportRequest['status']>('all')
  const [searchTerm, setSearchTerm] = useState('')
  const [eprQueueFilter, setEprQueueFilter] = useState<'all' | 'pending' | 'approved' | 'active' | 'expiring' | 'expired' | 'suspended'>('all')
  const [eprSearchTerm, setEprSearchTerm] = useState('')
  const [selectedEprIds, setSelectedEprIds] = useState<string[]>([])
  const [releaseMetrics, setReleaseMetrics] = useState<ReleaseMetricsSnapshot>({ requested: 0, released: 0, failed: 0, total: 0 })
  const [eprLifecycleMetrics, setEprLifecycleMetrics] = useState<Record<string, number>>({
    total: 0,
    pending: 0,
    approved: 0,
    rejected: 0,
    active: 0,
    expiring: 0,
    expired: 0,
    suspended: 0,
  })
  const [sorRows, setSorRows] = useState<EPRRegistrationRequest[]>([])
  const [releasePolicy, setReleasePolicy] = useState<ReleaseGovernancePolicyConfig>({
    enabled: true,
    failure_ratio_threshold: 0.25,
    consecutive_failures_threshold: 3,
    minimum_samples: 5,
    window_minutes: 60,
  })

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      setPermissionDenied(false)
      if (mode === 'requests') {
        const items = await imageImportService.listAdminImportRequests(50)
        setRows(items)
        setSorRows([])
        setSelectedEprIds([])
      } else {
        const [items, sor] = await Promise.all([
          imageImportService.listAdminImportRequests(50),
          eprRegistrationService.listAdminRequests({
            status: mode === 'epr' ? undefined : 'pending',
            limit: 50,
          }),
        ])
        setRows(items)
        setSorRows(sor)
        setSelectedEprIds([])
      }
    } catch (err) {
      const message = mapReviewerErrorMessage(err, 'Failed to load review queue')
      setError(message)
      setPermissionDenied(err instanceof ImageImportApiError && err.status === 403)
      setRows([])
      setSorRows([])
    } finally {
      setLoading(false)
    }
  }, [mode])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    const loadTelemetry = async () => {
      try {
        const [stats, policy] = await Promise.all([
          adminService.getSystemStats(),
          adminService.getReleaseGovernancePolicy(),
        ])
        setReleaseMetrics(stats.release_metrics || { requested: 0, released: 0, failed: 0, total: 0 })
        setEprLifecycleMetrics(stats.epr_lifecycle_metrics || {
          total: 0,
          pending: 0,
          approved: 0,
          rejected: 0,
          active: 0,
          expiring: 0,
          expired: 0,
          suspended: 0,
        })
        setReleasePolicy(policy)
      } catch {
        setReleaseMetrics({ requested: 0, released: 0, failed: 0, total: 0 })
        setEprLifecycleMetrics({
          total: 0,
          pending: 0,
          approved: 0,
          rejected: 0,
          active: 0,
          expiring: 0,
          expired: 0,
          suspended: 0,
        })
      }
    }
    loadTelemetry()
  }, [])

  const normalizedSearch = searchTerm.trim().toLowerCase()
  const normalizedEprSearch = eprSearchTerm.trim().toLowerCase()
  const filteredRows = useMemo(() => {
    return rows.filter((row) => {
      const matchesStatus = statusFilter === 'all' || row.status === statusFilter
      if (!matchesStatus) return false
      if (!normalizedSearch) return true
      return (
        row.source_image_ref.toLowerCase().includes(normalizedSearch) ||
        (row.epr_record_id || '').toLowerCase().includes(normalizedSearch) ||
        row.status.toLowerCase().includes(normalizedSearch) ||
        (row.sync_state || '').toLowerCase().includes(normalizedSearch)
      )
    })
  }, [rows, statusFilter, normalizedSearch])
  const pendingRows = useMemo(() => filteredRows.filter((row) => row.status === 'pending'), [filteredRows])
  const filteredEprRows = useMemo(() => {
    const byFilter = sorRows.filter((row) => {
      if (eprQueueFilter === 'all') return true
      if (eprQueueFilter === 'pending' || eprQueueFilter === 'approved') {
        return row.status === eprQueueFilter
      }
      return (row.lifecycle_status || 'active') === eprQueueFilter
    })
    if (!normalizedEprSearch) {
      return byFilter
    }
    return byFilter.filter((row) =>
      row.epr_record_id.toLowerCase().includes(normalizedEprSearch) ||
      row.product_name.toLowerCase().includes(normalizedEprSearch) ||
      row.technology_name.toLowerCase().includes(normalizedEprSearch) ||
      row.status.toLowerCase().includes(normalizedEprSearch) ||
      (row.lifecycle_status || 'active').toLowerCase().includes(normalizedEprSearch)
    )
  }, [sorRows, eprQueueFilter, normalizedEprSearch])
  const selectedEprRows = useMemo(
    () => filteredEprRows.filter((row) => selectedEprIds.includes(row.id)),
    [filteredEprRows, selectedEprIds]
  )
  const selectedBulkSuspendRows = useMemo(
    () => selectedEprRows.filter((row) => row.status === 'approved' && row.lifecycle_status !== 'suspended'),
    [selectedEprRows]
  )
  const selectedBulkReactivateRows = useMemo(
    () => selectedEprRows.filter((row) => row.status === 'approved' && row.lifecycle_status === 'suspended'),
    [selectedEprRows]
  )
  const selectedBulkRevalidateRows = useMemo(
    () => selectedEprRows.filter((row) => row.status === 'approved'),
    [selectedEprRows]
  )
  const releaseReadyRows = useMemo(() => filteredRows.filter((row) => row.release_state === 'ready_for_release'), [filteredRows])
  const releaseBlockedRows = useMemo(
    () =>
      filteredRows.filter(
        (row) =>
          row.release_state === 'release_blocked' ||
          (row.release_eligible === false &&
            row.status !== 'pending' &&
            row.release_state !== 'release_approved' &&
            row.release_state !== 'released')
      ),
    [filteredRows]
  )
  const releaseInProgressRows = useMemo(
    () => filteredRows.filter((row) => row.release_state === 'release_approved'),
    [filteredRows]
  )
  const releasedRows = useMemo(() => filteredRows.filter((row) => row.release_state === 'released'), [filteredRows])
  const recentRows = useMemo(() => filteredRows.filter((row) => row.status !== 'pending').slice(0, 10), [filteredRows])
  const releaseFailureRatio = releaseMetrics.total > 0 ? releaseMetrics.failed / releaseMetrics.total : 0
  const releaseFailureThresholdBreached =
    releasePolicy.enabled &&
    releaseMetrics.total >= releasePolicy.minimum_samples &&
    releaseFailureRatio >= releasePolicy.failure_ratio_threshold
  const blockedThresholdBreached =
    releasePolicy.enabled && releaseBlockedRows.length >= releasePolicy.consecutive_failures_threshold
  const eprLifecycleTotal = eprLifecycleMetrics.total || 0
  const eprAtRiskCount = (eprLifecycleMetrics.expiring || 0) + (eprLifecycleMetrics.expired || 0) + (eprLifecycleMetrics.suspended || 0)
  const eprAtRiskRatio = eprLifecycleTotal > 0 ? eprAtRiskCount / eprLifecycleTotal : 0
  const eprAtRiskAbsoluteThreshold = 3
  const eprAtRiskRatioThreshold = 0.3
  const eprLifecycleThresholdBreached =
    eprAtRiskCount >= eprAtRiskAbsoluteThreshold ||
    (eprLifecycleTotal >= 5 && eprAtRiskRatio >= eprAtRiskRatioThreshold)
  const isDashboardMode = mode === 'dashboard'
  const isRequestsMode = mode === 'requests'
  const isEprMode = mode === 'epr'
  const requestDetailBasePath = location.pathname.startsWith('/reviewer/')
    ? '/reviewer/quarantine/requests'
    : '/admin/quarantine/requests'
  const visiblePendingRows = isDashboardMode ? pendingRows.slice(0, 5) : pendingRows

  const parseJSONSummary = (raw?: string) => {
    if (!raw) return null
    try {
      return JSON.stringify(JSON.parse(raw), null, 2)
    } catch {
      return raw
    }
  }

  const handleApprove = async (row: ImageImportRequest) => {
    const confirmed = await confirmDialog({
      title: 'Approve Quarantine Request',
      message: `Approve request for ${row.source_image_ref}?`,
      confirmLabel: 'Approve',
    })
    if (!confirmed) return

    try {
      setActionBusy((prev) => ({ ...prev, [row.id]: true }))
      await imageImportService.approveAdminImportRequest(row.id)
      const now = new Date().toISOString()
      setRows((prev) =>
        prev.map((item) =>
          item.id === row.id
            ? {
              ...item,
              status: 'approved',
              updated_at: now,
            }
            : item
        )
      )
      setSelected((prev) => (prev && prev.id === row.id ? { ...prev, status: 'approved', updated_at: now } : prev))
      toast.success('Approval decision queued')
      window.setTimeout(() => {
        void load()
      }, 2000)
    } catch (err) {
      toast.error(mapReviewerErrorMessage(err, 'Failed to approve request'))
    } finally {
      setActionBusy((prev) => ({ ...prev, [row.id]: false }))
    }
  }

  const handleReject = async (row: ImageImportRequest) => {
    const confirmed = await confirmDialog({
      title: 'Reject Quarantine Request',
      message: `Reject request for ${row.source_image_ref}?`,
      confirmLabel: 'Reject',
    })
    if (!confirmed) return

    try {
      setActionBusy((prev) => ({ ...prev, [row.id]: true }))
      await imageImportService.rejectAdminImportRequest(row.id, 'Rejected by security reviewer')
      const now = new Date().toISOString()
      setRows((prev) =>
        prev.map((item) =>
          item.id === row.id
            ? {
              ...item,
              status: 'failed',
              updated_at: now,
            }
            : item
        )
      )
      setSelected((prev) => (prev && prev.id === row.id ? { ...prev, status: 'failed', updated_at: now } : prev))
      toast.success('Rejection decision queued')
      window.setTimeout(() => {
        void load()
      }, 2000)
    } catch (err) {
      toast.error(mapReviewerErrorMessage(err, 'Failed to reject request'))
    } finally {
      setActionBusy((prev) => ({ ...prev, [row.id]: false }))
    }
  }

  const handleRelease = async (row: ImageImportRequest) => {
    const defaultDestination = (row.internal_image_ref || row.source_image_ref || '').trim()
    const destinationImageRef = window.prompt(
      'Destination image reference (for example: registry.example.com/team/app:tag)',
      defaultDestination
    )?.trim()
    if (!destinationImageRef) {
      return
    }
    const destinationRegistryAuthId = window
      .prompt('Destination registry auth id (UUID)', row.registry_auth_id || '')
      ?.trim()
    if (!destinationRegistryAuthId) {
      toast.error('Destination registry auth id is required')
      return
    }
    const confirmed = await confirmDialog({
      title: 'Release Quarantined Artifact',
      message: `Release ${row.source_image_ref} to ${destinationImageRef}?`,
      confirmLabel: 'Release',
    })
    if (!confirmed) return

    try {
      setActionBusy((prev) => ({ ...prev, [row.id]: true }))
      await imageImportService.releaseImportRequest(row.id, {
        destinationImageRef,
        destinationRegistryAuthId,
      })
      toast.success('Release completed')
      await load()
    } catch (err) {
      if (err instanceof ImageImportApiError && err.code === 'release_not_eligible') {
        toast.error(`Release blocked: ${err.details?.release_blocker_reason || 'artifact is not eligible'}`)
        await load()
        return
      }
      if (err instanceof ImageImportApiError && err.code === 'validation_failed') {
        toast.error(err.message || 'Release request is missing destination details')
        return
      }
      toast.error(mapReviewerErrorMessage(err, 'Failed to release import'))
    } finally {
      setActionBusy((prev) => ({ ...prev, [row.id]: false }))
    }
  }

  const handleSORApprove = async (row: EPRRegistrationRequest) => {
    const confirmed = await confirmDialog({
      title: 'Approve EPR Registration',
      message: `Approve EPR registration ${row.epr_record_id} for ${row.product_name}?`,
      confirmLabel: 'Approve',
    })
    if (!confirmed) return
    try {
      setActionBusy((prev) => ({ ...prev, [row.id]: true }))
      await eprRegistrationService.approveRequest(row.id, 'Approved by security reviewer')
      toast.success('EPR registration approved')
      await load()
    } catch (err) {
      toast.error(mapReviewerErrorMessage(err, 'Failed to approve EPR registration'))
    } finally {
      setActionBusy((prev) => ({ ...prev, [row.id]: false }))
    }
  }

  const handleSORReject = async (row: EPRRegistrationRequest) => {
    const confirmed = await confirmDialog({
      title: 'Reject EPR Registration',
      message: `Reject EPR registration ${row.epr_record_id} for ${row.product_name}?`,
      confirmLabel: 'Reject',
    })
    if (!confirmed) return
    try {
      setActionBusy((prev) => ({ ...prev, [row.id]: true }))
      await eprRegistrationService.rejectRequest(row.id, 'Rejected by security reviewer')
      toast.success('EPR registration rejected')
      await load()
    } catch (err) {
      toast.error(mapReviewerErrorMessage(err, 'Failed to reject EPR registration'))
    } finally {
      setActionBusy((prev) => ({ ...prev, [row.id]: false }))
    }
  }

  const askLifecycleReason = (defaultReason: string) => {
    const value = window.prompt('Enter reason for this lifecycle action:', defaultReason)
    if (value === null) return null
    return value.trim() || defaultReason
  }

  const handleSORSuspend = async (row: EPRRegistrationRequest) => {
    const reason = askLifecycleReason('Suspended by security reviewer')
    if (!reason) return
    try {
      setActionBusy((prev) => ({ ...prev, [row.id]: true }))
      await eprRegistrationService.suspendRequest(row.id, reason)
      toast.success('EPR registration suspended')
      await load()
    } catch (err) {
      toast.error(mapReviewerErrorMessage(err, 'Failed to suspend EPR registration'))
    } finally {
      setActionBusy((prev) => ({ ...prev, [row.id]: false }))
    }
  }

  const handleSORReactivate = async (row: EPRRegistrationRequest) => {
    const reason = askLifecycleReason('Reactivated by security reviewer')
    if (!reason) return
    try {
      setActionBusy((prev) => ({ ...prev, [row.id]: true }))
      await eprRegistrationService.reactivateRequest(row.id, reason)
      toast.success('EPR registration reactivated')
      await load()
    } catch (err) {
      toast.error(mapReviewerErrorMessage(err, 'Failed to reactivate EPR registration'))
    } finally {
      setActionBusy((prev) => ({ ...prev, [row.id]: false }))
    }
  }

  const handleSORRevalidate = async (row: EPRRegistrationRequest) => {
    const reason = askLifecycleReason('Revalidated by security reviewer')
    if (!reason) return
    try {
      setActionBusy((prev) => ({ ...prev, [row.id]: true }))
      await eprRegistrationService.revalidateRequest(row.id, reason)
      toast.success('EPR registration revalidated')
      await load()
    } catch (err) {
      toast.error(mapReviewerErrorMessage(err, 'Failed to revalidate EPR registration'))
    } finally {
      setActionBusy((prev) => ({ ...prev, [row.id]: false }))
    }
  }

  const handleRowKeyActivate = (
    event: React.KeyboardEvent<HTMLTableRowElement>,
    activate: () => void
  ) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      activate()
    }
  }

  const toggleEprRowSelection = (rowID: string) => {
    setSelectedEprIds((prev) => (prev.includes(rowID) ? prev.filter((id) => id !== rowID) : [...prev, rowID]))
  }

  const toggleSelectAllEprRows = () => {
    const visibleIds = filteredEprRows.map((row) => row.id)
    const allSelected = visibleIds.length > 0 && visibleIds.every((id) => selectedEprIds.includes(id))
    if (allSelected) {
      setSelectedEprIds((prev) => prev.filter((id) => !visibleIds.includes(id)))
      return
    }
    setSelectedEprIds((prev) => Array.from(new Set([...prev, ...visibleIds])))
  }

  const handleBulkSORSuspend = async () => {
    if (selectedBulkSuspendRows.length === 0) return
    const reason = askLifecycleReason('Suspended by security reviewer')
    if (!reason) return
    const confirmed = await confirmDialog({
      title: 'Bulk Suspend EPR Registrations',
      message: `Suspend ${selectedBulkSuspendRows.length} selected EPR registrations?`,
      confirmLabel: 'Suspend',
    })
    if (!confirmed) return
    try {
      await eprRegistrationService.bulkSuspendRequests(selectedBulkSuspendRows.map((row) => row.id), reason)
      toast.success(`Suspended ${selectedBulkSuspendRows.length} EPR registration(s)`)
      setSelectedEprIds([])
      await load()
    } catch (err) {
      toast.error(mapReviewerErrorMessage(err, 'Failed to suspend selected EPR registrations'))
    }
  }

  const handleBulkSORReactivate = async () => {
    if (selectedBulkReactivateRows.length === 0) return
    const reason = askLifecycleReason('Reactivated by security reviewer')
    if (!reason) return
    const confirmed = await confirmDialog({
      title: 'Bulk Reactivate EPR Registrations',
      message: `Reactivate ${selectedBulkReactivateRows.length} selected EPR registrations?`,
      confirmLabel: 'Reactivate',
    })
    if (!confirmed) return
    try {
      await eprRegistrationService.bulkReactivateRequests(selectedBulkReactivateRows.map((row) => row.id), reason)
      toast.success(`Reactivated ${selectedBulkReactivateRows.length} EPR registration(s)`)
      setSelectedEprIds([])
      await load()
    } catch (err) {
      toast.error(mapReviewerErrorMessage(err, 'Failed to reactivate selected EPR registrations'))
    }
  }

  const handleBulkSORRevalidate = async () => {
    if (selectedBulkRevalidateRows.length === 0) return
    const reason = askLifecycleReason('Revalidated by security reviewer')
    if (!reason) return
    const confirmed = await confirmDialog({
      title: 'Bulk Revalidate EPR Registrations',
      message: `Revalidate ${selectedBulkRevalidateRows.length} selected EPR registrations?`,
      confirmLabel: 'Revalidate',
    })
    if (!confirmed) return
    try {
      await eprRegistrationService.bulkRevalidateRequests(selectedBulkRevalidateRows.map((row) => row.id), reason)
      toast.success(`Revalidated ${selectedBulkRevalidateRows.length} EPR registration(s)`)
      setSelectedEprIds([])
      await load()
    } catch (err) {
      toast.error(mapReviewerErrorMessage(err, 'Failed to revalidate selected EPR registrations'))
    }
  }

  return (
    <div className="space-y-6 px-4 py-6 sm:px-6 lg:px-8">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">
            {isEprMode ? 'EPR Approval Queue' : 'Security Review Queue'}
          </h1>
          <p className="mt-2 text-sm text-slate-700 dark:text-slate-400">
            {isEprMode
              ? 'Review and decide Enterprise Product Registry registration requests.'
              : 'Review pending quarantine requests and submit approve/reject decisions from one queue.'}
          </p>
          {isDashboardMode ? (
            <p className="mt-1 text-xs font-medium text-slate-500 dark:text-slate-400">
              Summary view: queue health, release governance, and recent decision signals.
            </p>
          ) : null}
        </div>
        <button
          type="button"
          onClick={() => void load()}
          className="rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
        >
          Refresh
        </button>
      </div>

      {!isEprMode ? (
        <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900">
          {!isDashboardMode ? (
            <div className="mb-3 grid grid-cols-1 gap-2 md:grid-cols-3">
              <input
                type="text"
                value={searchTerm}
                onChange={(event) => setSearchTerm(event.target.value)}
                placeholder="Search by image, EPR, status, sync state"
                className="rounded-md border border-slate-300 px-2.5 py-1.5 text-xs text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
              />
              <select
                value={statusFilter}
                onChange={(event) => setStatusFilter(event.target.value as 'all' | ImageImportRequest['status'])}
                className="rounded-md border border-slate-300 px-2.5 py-1.5 text-xs text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
              >
                <option value="all">All statuses</option>
                <option value="pending">Pending</option>
                <option value="approved">Approved</option>
                <option value="importing">Importing</option>
                <option value="success">Success</option>
                <option value="quarantined">Quarantined</option>
                <option value="failed">Failed</option>
              </select>
              <div className="rounded-md border border-slate-200 px-2.5 py-1.5 text-xs text-slate-600 dark:border-slate-700 dark:text-slate-300">
                Filtered rows: <span className="font-semibold">{filteredRows.length}</span>
              </div>
            </div>
          ) : null}
          {!isRequestsMode ? (
            <>
              <div className="flex items-center justify-between gap-3">
                <h2 className="text-base font-semibold text-slate-900 dark:text-white">
                  Pending Decisions ({pendingRows.length})
                </h2>
                {isDashboardMode ? (
                  <Link to="/reviewer/quarantine/requests" className="text-xs font-medium text-blue-700 hover:underline dark:text-blue-300">
                    View all requests
                  </Link>
                ) : null}
              </div>
              {loading ? (
                <p className="mt-3 text-sm text-slate-500 dark:text-slate-400">Loading review queue...</p>
              ) : error ? (
                <div className="mt-3 rounded-md border border-rose-200 bg-rose-50 p-3 text-sm text-rose-700 dark:border-rose-700 dark:bg-rose-950/40 dark:text-rose-200">
                  <p>{error}</p>
                  {permissionDenied ? (
                    <p className="mt-2 text-xs text-rose-700 dark:text-rose-300">
                      Review access requirements in{' '}
                      <Link to="/help/capability-access" className="font-medium underline">
                        Capability Matrix
                      </Link>
                      .
                    </p>
                  ) : null}
                </div>
              ) : visiblePendingRows.length === 0 ? (
                <p className="mt-3 text-sm text-slate-500 dark:text-slate-400">No pending approvals.</p>
              ) : (
                <div className="mt-3 space-y-3">
                  {visiblePendingRows.map((row) => {
                    const diagnostic = getImportDiagnostic(row)
                    return (
                      <div key={row.id} className="rounded-md border border-slate-200 p-3 dark:border-slate-700">
                        <div className="mb-2 flex flex-wrap items-center gap-2">
                          <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
                            {getImportProgressLabel(row)}
                          </span>
                          <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-semibold text-blue-800 dark:bg-blue-900/50 dark:text-blue-200">
                            {row.status}
                          </span>
                        </div>
                        <p className="text-sm text-slate-900 dark:text-slate-100">{row.source_image_ref}</p>
                        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                          EPR {row.epr_record_id} • Updated {new Date(row.updated_at).toLocaleString()}
                        </p>
                        <div className={`mt-2 rounded-md border px-2 py-1 text-xs ${getImportDiagnosticClasses(diagnostic.tone)}`}>
                          <p className="font-semibold">{diagnostic.title}</p>
                          <p className="mt-0.5">{diagnostic.message}</p>
                        </div>
                        <div className="mt-3 flex flex-wrap items-center gap-2">
                          <Link
                            to={`${requestDetailBasePath}/${row.id}`}
                            className="rounded-md border border-blue-300 bg-blue-50 px-2.5 py-1 text-xs font-medium text-blue-800 hover:bg-blue-100 dark:border-blue-700 dark:bg-blue-950/40 dark:text-blue-200 dark:hover:bg-blue-900/40"
                          >
                            View Status
                          </Link>
                          <button
                            type="button"
                            onClick={() => setSelected(row)}
                            className="rounded-md border border-slate-300 px-2.5 py-1 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                          >
                            Details
                          </button>
                          <button
                            type="button"
                            disabled={Boolean(actionBusy[row.id])}
                            onClick={() => void handleApprove(row)}
                            className="rounded-md border border-emerald-300 bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-800 hover:bg-emerald-100 disabled:opacity-60 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-200 dark:hover:bg-emerald-900/40"
                          >
                            {actionBusy[row.id] ? 'Submitting...' : 'Approve'}
                          </button>
                          <button
                            type="button"
                            disabled={Boolean(actionBusy[row.id])}
                            onClick={() => void handleReject(row)}
                            className="rounded-md border border-rose-300 bg-rose-50 px-2.5 py-1 text-xs font-medium text-rose-800 hover:bg-rose-100 disabled:opacity-60 dark:border-rose-700 dark:bg-rose-950/40 dark:text-rose-200 dark:hover:bg-rose-900/40"
                          >
                            {actionBusy[row.id] ? 'Submitting...' : 'Reject'}
                          </button>
                        </div>
                      </div>
                    )
                  })}
                </div>
              )}
            </>
          ) : (
            <>
              <div className="flex items-center justify-between gap-3">
                <h2 className="text-base font-semibold text-slate-900 dark:text-white">
                  Review List ({filteredRows.length})
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400">Click a row or `Details` to inspect full request payload.</p>
              </div>
              {loading ? (
                <p className="mt-3 text-sm text-slate-500 dark:text-slate-400">Loading review queue...</p>
              ) : error ? (
                <div className="mt-3 rounded-md border border-rose-200 bg-rose-50 p-3 text-sm text-rose-700 dark:border-rose-700 dark:bg-rose-950/40 dark:text-rose-200">
                  <p>{error}</p>
                  {permissionDenied ? (
                    <p className="mt-2 text-xs text-rose-700 dark:text-rose-300">
                      Review access requirements in{' '}
                      <Link to="/help/capability-access" className="font-medium underline">
                        Capability Matrix
                      </Link>
                      .
                    </p>
                  ) : null}
                </div>
              ) : filteredRows.length === 0 ? (
                <p className="mt-3 text-sm text-slate-500 dark:text-slate-400">No requests match the current filters.</p>
              ) : (
                <div className="mt-3 overflow-x-auto rounded-md border border-slate-200 dark:border-slate-700">
                  <table className="min-w-full text-xs">
                    <thead className="bg-slate-50 dark:bg-slate-800/70">
                      <tr className="text-left text-slate-600 dark:text-slate-300">
                        <th className="px-3 py-2 font-medium">Status</th>
                        <th className="px-3 py-2 font-medium">Image</th>
                        <th className="px-3 py-2 font-medium">EPR</th>
                        <th className="px-3 py-2 font-medium">Updated</th>
                        <th className="px-3 py-2 font-medium">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                      {filteredRows.map((row) => (
                        <tr
                          key={row.id}
                          className="cursor-pointer bg-white hover:bg-slate-50 dark:bg-slate-900 dark:hover:bg-slate-800/70"
                          onClick={() => setSelected(row)}
                          onKeyDown={(event) => handleRowKeyActivate(event, () => setSelected(row))}
                          tabIndex={0}
                          role="button"
                          aria-label={`Open request ${row.source_image_ref}`}
                        >
                          <td className="px-3 py-2 align-top">
                            <span className="rounded-full bg-blue-100 px-2 py-0.5 font-medium text-blue-800 dark:bg-blue-900/50 dark:text-blue-200">
                              {row.status}
                            </span>
                            <p className="mt-1 text-[11px] text-slate-500 dark:text-slate-400">{getImportProgressLabel(row)}</p>
                          </td>
                          <td className="px-3 py-2 align-top text-slate-900 dark:text-slate-100">
                            <p className="line-clamp-1">{row.source_image_ref}</p>
                            {row.release_blocker_reason ? (
                              <p className="mt-1 line-clamp-1 text-[11px] text-amber-700 dark:text-amber-300">{row.release_blocker_reason}</p>
                            ) : null}
                          </td>
                          <td className="px-3 py-2 align-top text-slate-700 dark:text-slate-300">{row.epr_record_id}</td>
                          <td className="px-3 py-2 align-top text-slate-600 dark:text-slate-400">{new Date(row.updated_at).toLocaleString()}</td>
                          <td className="px-3 py-2 align-top">
                            <div className="flex flex-wrap items-center gap-1.5" onClick={(event) => event.stopPropagation()}>
                              <Link
                                to={`${requestDetailBasePath}/${row.id}`}
                                className="rounded-md border border-blue-300 bg-blue-50 px-2 py-1 text-[11px] font-medium text-blue-800 hover:bg-blue-100 dark:border-blue-700 dark:bg-blue-950/40 dark:text-blue-200 dark:hover:bg-blue-900/40"
                              >
                                View Status
                              </Link>
                              <button
                                type="button"
                                onClick={() => setSelected(row)}
                                className="rounded-md border border-slate-300 px-2 py-1 text-[11px] font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                              >
                                Details
                              </button>
                              {row.status === 'pending' ? (
                                <>
                                  <button
                                    type="button"
                                    disabled={Boolean(actionBusy[row.id])}
                                    onClick={() => void handleApprove(row)}
                                    className="rounded-md border border-emerald-300 bg-emerald-50 px-2 py-1 text-[11px] font-medium text-emerald-800 hover:bg-emerald-100 disabled:opacity-60 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-200 dark:hover:bg-emerald-900/40"
                                  >
                                    {actionBusy[row.id] ? '...' : 'Approve'}
                                  </button>
                                  <button
                                    type="button"
                                    disabled={Boolean(actionBusy[row.id])}
                                    onClick={() => void handleReject(row)}
                                    className="rounded-md border border-rose-300 bg-rose-50 px-2 py-1 text-[11px] font-medium text-rose-800 hover:bg-rose-100 disabled:opacity-60 dark:border-rose-700 dark:bg-rose-950/40 dark:text-rose-200 dark:hover:bg-rose-900/40"
                                  >
                                    {actionBusy[row.id] ? '...' : 'Reject'}
                                  </button>
                                </>
                              ) : null}
                              {row.release_state === 'ready_for_release' ? (
                                <button
                                  type="button"
                                  disabled={Boolean(actionBusy[row.id])}
                                  onClick={() => void handleRelease(row)}
                                  className="rounded-md border border-indigo-300 bg-indigo-50 px-2 py-1 text-[11px] font-medium text-indigo-800 hover:bg-indigo-100 disabled:opacity-60 dark:border-indigo-700 dark:bg-indigo-950/40 dark:text-indigo-200 dark:hover:bg-indigo-900/40"
                                >
                                  {actionBusy[row.id] ? '...' : 'Release'}
                                </button>
                              ) : null}
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}
        </section>
      ) : null}

      {!isRequestsMode ? (
        <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h2 className="text-base font-semibold text-slate-900 dark:text-white">
              EPR Registration Approvals ({filteredEprRows.length})
            </h2>
            <span className="rounded-full border border-slate-200 px-2 py-0.5 text-[11px] font-medium text-slate-600 dark:border-slate-700 dark:text-slate-300">
              Total: {sorRows.length}
            </span>
          </div>
          <div className="mt-3 grid grid-cols-1 gap-2 md:grid-cols-3">
            <input
              type="text"
              value={eprSearchTerm}
              onChange={(event) => setEprSearchTerm(event.target.value)}
              placeholder="Search by EPR, product, lifecycle"
              className="rounded-md border border-slate-300 px-2.5 py-1.5 text-xs text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
            />
            <div className="col-span-2 flex flex-wrap gap-1.5">
              {[
                { key: 'all', label: 'All' },
                { key: 'pending', label: 'Pending' },
                { key: 'active', label: 'Active' },
                { key: 'expiring', label: 'Expiring' },
                { key: 'expired', label: 'Expired' },
                { key: 'suspended', label: 'Suspended' },
              ].map((chip) => (
                <button
                  key={chip.key}
                  type="button"
                  onClick={() => setEprQueueFilter(chip.key as 'all' | 'pending' | 'approved' | 'active' | 'expiring' | 'expired' | 'suspended')}
                  className={`rounded-md border px-2 py-1 text-[11px] font-medium transition-colors ${eprQueueFilter === chip.key
                      ? 'border-blue-300 bg-blue-50 text-blue-800 dark:border-blue-700 dark:bg-blue-900/40 dark:text-blue-200'
                      : 'border-slate-300 bg-white text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800'
                    }`}
                >
                  {chip.label}
                </button>
              ))}
            </div>
          </div>
          <div className="mt-2 rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-800/40">
            <div className="mb-2 flex items-center justify-between">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">EPR Lifecycle Metrics</p>
              <span className={`rounded-full px-2 py-0.5 text-[11px] font-medium ${eprLifecycleThresholdBreached
                  ? 'bg-amber-100 text-amber-800 dark:bg-amber-900/50 dark:text-amber-200'
                  : 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200'
                }`}>
                {eprLifecycleThresholdBreached ? 'degraded' : 'healthy'}
              </span>
            </div>
            <p className="mb-2 text-[11px] text-slate-600 dark:text-slate-300">
              Total {eprLifecycleTotal} • At-risk {eprAtRiskCount} ({(eprAtRiskRatio * 100).toFixed(1)}%)
            </p>
            <div className="grid grid-cols-2 gap-2 text-[11px] md:grid-cols-4">
              <p className="rounded border border-slate-200 bg-white px-2 py-1 text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">Pending: <span className="font-semibold">{eprLifecycleMetrics.pending || 0}</span></p>
              <p className="rounded border border-slate-200 bg-white px-2 py-1 text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">Approved: <span className="font-semibold">{eprLifecycleMetrics.approved || 0}</span></p>
              <p className="rounded border border-slate-200 bg-white px-2 py-1 text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">Active: <span className="font-semibold">{eprLifecycleMetrics.active || 0}</span></p>
              <p className="rounded border border-slate-200 bg-white px-2 py-1 text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">Expiring: <span className="font-semibold">{eprLifecycleMetrics.expiring || 0}</span></p>
              <p className="rounded border border-slate-200 bg-white px-2 py-1 text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">Expired: <span className="font-semibold">{eprLifecycleMetrics.expired || 0}</span></p>
              <p className="rounded border border-slate-200 bg-white px-2 py-1 text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">Suspended: <span className="font-semibold">{eprLifecycleMetrics.suspended || 0}</span></p>
              <p className="rounded border border-slate-200 bg-white px-2 py-1 text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">Rejected: <span className="font-semibold">{eprLifecycleMetrics.rejected || 0}</span></p>
            </div>
            <p className="mt-2 text-[11px] text-slate-500 dark:text-slate-400">
              Posture thresholds: degraded when at-risk count is {eprAtRiskAbsoluteThreshold}+ or ratio reaches {(eprAtRiskRatioThreshold * 100).toFixed(0)}% with at least 5 total.
            </p>
          </div>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={() => void handleBulkSORSuspend()}
              disabled={selectedBulkSuspendRows.length === 0}
              className="rounded-md border border-rose-300 bg-rose-50 px-2 py-1 text-[11px] font-medium text-rose-800 hover:bg-rose-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-rose-700 dark:bg-rose-900/30 dark:text-rose-200 dark:hover:bg-rose-900/50"
            >
              Bulk Suspend ({selectedBulkSuspendRows.length})
            </button>
            <button
              type="button"
              onClick={() => void handleBulkSORReactivate()}
              disabled={selectedBulkReactivateRows.length === 0}
              className="rounded-md border border-emerald-300 bg-emerald-50 px-2 py-1 text-[11px] font-medium text-emerald-800 hover:bg-emerald-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200 dark:hover:bg-emerald-900/50"
            >
              Bulk Reactivate ({selectedBulkReactivateRows.length})
            </button>
            <button
              type="button"
              onClick={() => void handleBulkSORRevalidate()}
              disabled={selectedBulkRevalidateRows.length === 0}
              className="rounded-md border border-indigo-300 bg-indigo-50 px-2 py-1 text-[11px] font-medium text-indigo-800 hover:bg-indigo-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-200 dark:hover:bg-indigo-900/50"
            >
              Bulk Revalidate ({selectedBulkRevalidateRows.length})
            </button>
          </div>
          {filteredEprRows.length === 0 ? (
            <p className="mt-3 text-sm text-slate-500 dark:text-slate-400">No EPR registration requests match current filters.</p>
          ) : (
            <div className="mt-3 overflow-x-auto rounded-md border border-slate-200 dark:border-slate-700">
              <table className="min-w-full text-xs">
                <thead className="bg-slate-50 dark:bg-slate-800/70">
                  <tr className="text-left text-slate-600 dark:text-slate-300">
                    <th className="px-3 py-2 font-medium">
                      <input
                        type="checkbox"
                        checked={filteredEprRows.length > 0 && filteredEprRows.every((row) => selectedEprIds.includes(row.id))}
                        onChange={toggleSelectAllEprRows}
                        className="h-3.5 w-3.5 rounded border-slate-300 text-blue-600 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-900"
                        aria-label="Select all EPR rows"
                      />
                    </th>
                    <th className="px-3 py-2 font-medium">EPR</th>
                    <th className="px-3 py-2 font-medium">Product / Technology</th>
                    <th className="px-3 py-2 font-medium">Status</th>
                    <th className="px-3 py-2 font-medium">Lifecycle</th>
                    <th className="px-3 py-2 font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                  {filteredEprRows.map((row) => (
                    <tr
                      key={row.id}
                      className="cursor-pointer bg-white hover:bg-slate-50 dark:bg-slate-900 dark:hover:bg-slate-800/70"
                      onClick={() => setSelectedEpr(row)}
                      onKeyDown={(event) => handleRowKeyActivate(event, () => setSelectedEpr(row))}
                      tabIndex={0}
                      role="button"
                      aria-label={`Open EPR ${row.epr_record_id}`}
                    >
                      <td className="px-3 py-2 align-top" onClick={(event) => event.stopPropagation()}>
                        <input
                          type="checkbox"
                          checked={selectedEprIds.includes(row.id)}
                          onChange={() => toggleEprRowSelection(row.id)}
                          className="h-3.5 w-3.5 rounded border-slate-300 text-blue-600 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-900"
                          aria-label={`Select ${row.epr_record_id}`}
                        />
                      </td>
                      <td className="px-3 py-2 align-top text-slate-900 dark:text-slate-100">{row.epr_record_id}</td>
                      <td className="px-3 py-2 align-top text-slate-700 dark:text-slate-300">
                        <p>{row.product_name}</p>
                        <p className="mt-0.5 text-[11px] text-slate-500 dark:text-slate-400">{row.technology_name}</p>
                      </td>
                      <td className="px-3 py-2 align-top">
                        <span className="rounded-full bg-amber-100 px-2 py-0.5 font-medium text-amber-800 dark:bg-amber-900/40 dark:text-amber-200">
                          {row.status}
                        </span>
                      </td>
                      <td className="px-3 py-2 align-top">
                        <span className={`rounded-full px-2 py-0.5 font-medium ${row.lifecycle_status === 'suspended'
                            ? 'bg-rose-100 text-rose-800 dark:bg-rose-900/40 dark:text-rose-200'
                            : row.lifecycle_status === 'expired'
                              ? 'bg-slate-200 text-slate-800 dark:bg-slate-700 dark:text-slate-100'
                              : row.lifecycle_status === 'expiring'
                                ? 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200'
                                : 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200'
                          }`}>
                          {row.lifecycle_status || 'active'}
                        </span>
                      </td>
                      <td className="px-3 py-2 align-top">
                        <div className="flex flex-wrap items-center gap-1.5" onClick={(event) => event.stopPropagation()}>
                          <button
                            type="button"
                            onClick={() => setSelectedEpr(row)}
                            className="rounded-md border border-slate-300 px-2 py-1 text-[11px] font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                          >
                            View
                          </button>
                          {row.status === 'pending' ? (
                            <>
                              <button
                                type="button"
                                onClick={() => handleSORApprove(row)}
                                disabled={Boolean(actionBusy[row.id])}
                                className="rounded-md border border-emerald-300 bg-emerald-50 px-2 py-1 text-[11px] font-medium text-emerald-800 hover:bg-emerald-100 disabled:opacity-60 dark:border-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200 dark:hover:bg-emerald-900/50"
                              >
                                {actionBusy[row.id] ? '...' : 'Approve'}
                              </button>
                              <button
                                type="button"
                                onClick={() => handleSORReject(row)}
                                disabled={Boolean(actionBusy[row.id])}
                                className="rounded-md border border-rose-300 bg-rose-50 px-2 py-1 text-[11px] font-medium text-rose-800 hover:bg-rose-100 disabled:opacity-60 dark:border-rose-700 dark:bg-rose-900/30 dark:text-rose-200 dark:hover:bg-rose-900/50"
                              >
                                {actionBusy[row.id] ? '...' : 'Reject'}
                              </button>
                            </>
                          ) : (
                            <>
                              {row.status === 'approved' && row.lifecycle_status !== 'suspended' ? (
                                <>
                                  <button
                                    type="button"
                                    onClick={() => void handleSORSuspend(row)}
                                    disabled={Boolean(actionBusy[row.id])}
                                    className="rounded-md border border-rose-300 bg-rose-50 px-2 py-1 text-[11px] font-medium text-rose-800 hover:bg-rose-100 disabled:opacity-60 dark:border-rose-700 dark:bg-rose-900/30 dark:text-rose-200 dark:hover:bg-rose-900/50"
                                  >
                                    {actionBusy[row.id] ? '...' : 'Suspend'}
                                  </button>
                                  <button
                                    type="button"
                                    onClick={() => void handleSORRevalidate(row)}
                                    disabled={Boolean(actionBusy[row.id])}
                                    className="rounded-md border border-indigo-300 bg-indigo-50 px-2 py-1 text-[11px] font-medium text-indigo-800 hover:bg-indigo-100 disabled:opacity-60 dark:border-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-200 dark:hover:bg-indigo-900/50"
                                  >
                                    {actionBusy[row.id] ? '...' : 'Revalidate'}
                                  </button>
                                </>
                              ) : null}
                              {row.status === 'approved' && row.lifecycle_status === 'suspended' ? (
                                <button
                                  type="button"
                                  onClick={() => void handleSORReactivate(row)}
                                  disabled={Boolean(actionBusy[row.id])}
                                  className="rounded-md border border-emerald-300 bg-emerald-50 px-2 py-1 text-[11px] font-medium text-emerald-800 hover:bg-emerald-100 disabled:opacity-60 dark:border-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200 dark:hover:bg-emerald-900/50"
                                >
                                  {actionBusy[row.id] ? '...' : 'Reactivate'}
                                </button>
                              ) : null}
                              {row.status !== 'approved' ? (
                                <span className="text-[11px] text-slate-500 dark:text-slate-400">Read-only</span>
                              ) : null}
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      ) : (
        <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900">
          <div className="flex items-center justify-between gap-2">
            <h2 className="text-base font-semibold text-slate-900 dark:text-white">EPR Approvals</h2>
            <Link to="/reviewer/epr/approvals" className="text-xs font-medium text-blue-700 hover:underline dark:text-blue-300">
              Open EPR Approvals
            </Link>
          </div>
          <p className="mt-2 text-xs text-slate-600 dark:text-slate-400">
            EPR approvals are managed in the dedicated EPR Approvals workspace.
          </p>
        </section>
      )}

      {!isRequestsMode && !isEprMode ? (
        <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900">
          <div className={`mb-3 rounded-md border px-3 py-2 ${releaseFailureThresholdBreached || blockedThresholdBreached
            ? 'border-amber-200 bg-amber-50 dark:border-amber-700 dark:bg-amber-900/25'
            : 'border-emerald-200 bg-emerald-50 dark:border-emerald-700 dark:bg-emerald-900/20'
            }`}>
            <div className="flex flex-wrap items-center justify-between gap-2">
              <h2 className="text-base font-semibold text-slate-900 dark:text-white">Release Governance</h2>
              <span className={`rounded-full px-2 py-0.5 text-[11px] font-medium ${releaseFailureThresholdBreached || blockedThresholdBreached
                ? 'bg-amber-100 text-amber-800 dark:bg-amber-900/50 dark:text-amber-200'
                : 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-200'
                }`}>
                {releaseFailureThresholdBreached || blockedThresholdBreached ? 'degraded' : 'healthy'}
              </span>
            </div>
            <div className="mt-2 grid grid-cols-2 gap-2 text-[11px] md:grid-cols-4">
              <p className="rounded border border-slate-200 bg-white px-2 py-1 text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">Requested: <span className="font-semibold">{releaseMetrics.requested}</span></p>
              <p className="rounded border border-slate-200 bg-white px-2 py-1 text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">Released: <span className="font-semibold">{releaseMetrics.released}</span></p>
              <p className="rounded border border-slate-200 bg-white px-2 py-1 text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">Failed: <span className="font-semibold">{releaseMetrics.failed}</span></p>
              <p className="rounded border border-slate-200 bg-white px-2 py-1 text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">
                Failure Ratio: <span className="font-semibold">{(releaseFailureRatio * 100).toFixed(1)}%</span>
              </p>
            </div>
            <p className="mt-2 text-[11px] text-slate-600 dark:text-slate-300">
              Thresholds: ratio {(releasePolicy.failure_ratio_threshold * 100).toFixed(1)}% after {releasePolicy.minimum_samples} samples, blocked lane trigger {releasePolicy.consecutive_failures_threshold}, window {releasePolicy.window_minutes}m.
            </p>
          </div>
          <div className="mt-3 grid grid-cols-1 gap-3 lg:grid-cols-2">
            <div className="rounded-md border border-emerald-200 bg-emerald-50/50 p-3 dark:border-emerald-800 dark:bg-emerald-950/20">
              <h3 className="text-sm font-semibold text-emerald-900 dark:text-emerald-200">Ready ({releaseReadyRows.length})</h3>
              {releaseReadyRows.length === 0 ? (
                <p className="mt-2 text-xs text-emerald-800/80 dark:text-emerald-300">No release-ready artifacts.</p>
              ) : (
                <div className="mt-2 space-y-2">
                  {releaseReadyRows.map((row) => (
                    <div key={row.id} className="rounded-md border border-emerald-200 bg-white p-2.5 dark:border-emerald-800 dark:bg-slate-900">
                      <p className="text-xs font-medium text-slate-900 dark:text-slate-100">{row.source_image_ref}</p>
                      <p className="mt-1 text-[11px] text-slate-500 dark:text-slate-400">
                        {row.source_image_digest ? `Digest ${row.source_image_digest}` : 'Digest pending'}
                      </p>
                      <div className="mt-2 flex flex-wrap items-center gap-2">
                        <button
                          type="button"
                          onClick={() => setSelected(row)}
                          className="rounded-md border border-slate-300 px-2 py-1 text-[11px] font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                        >
                          Details
                        </button>
                        <button
                          type="button"
                          disabled={Boolean(actionBusy[row.id])}
                          onClick={() => void handleRelease(row)}
                          className="rounded-md border border-indigo-300 bg-indigo-50 px-2 py-1 text-[11px] font-medium text-indigo-800 hover:bg-indigo-100 disabled:opacity-60 dark:border-indigo-700 dark:bg-indigo-950/40 dark:text-indigo-200 dark:hover:bg-indigo-900/40"
                        >
                          {actionBusy[row.id] ? 'Submitting...' : 'Release'}
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className="rounded-md border border-amber-200 bg-amber-50/60 p-3 dark:border-amber-800 dark:bg-amber-950/20">
              <h3 className="text-sm font-semibold text-amber-900 dark:text-amber-200">Blocked ({releaseBlockedRows.length})</h3>
              {releaseBlockedRows.length === 0 ? (
                <p className="mt-2 text-xs text-amber-800/80 dark:text-amber-300">No blocked artifacts.</p>
              ) : (
                <div className="mt-2 space-y-2">
                  {releaseBlockedRows.map((row) => (
                    <div key={row.id} className="rounded-md border border-amber-200 bg-white p-2.5 dark:border-amber-800 dark:bg-slate-900">
                      <p className="text-xs font-medium text-slate-900 dark:text-slate-100">{row.source_image_ref}</p>
                      <p className="mt-1 text-[11px] text-amber-800 dark:text-amber-300">
                        {row.release_blocker_reason || row.error_message || 'Not eligible for release yet'}
                      </p>
                      <button
                        type="button"
                        onClick={() => setSelected(row)}
                        className="mt-2 rounded-md border border-slate-300 px-2 py-1 text-[11px] font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                      >
                        Details
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className="rounded-md border border-sky-200 bg-sky-50/60 p-3 dark:border-sky-800 dark:bg-sky-950/20">
              <h3 className="text-sm font-semibold text-sky-900 dark:text-sky-200">In Progress ({releaseInProgressRows.length})</h3>
              {releaseInProgressRows.length === 0 ? (
                <p className="mt-2 text-xs text-sky-800/80 dark:text-sky-300">No in-progress releases.</p>
              ) : (
                <div className="mt-2 space-y-2">
                  {releaseInProgressRows.map((row) => (
                    <div key={row.id} className="rounded-md border border-sky-200 bg-white p-2.5 dark:border-sky-800 dark:bg-slate-900">
                      <p className="text-xs font-medium text-slate-900 dark:text-slate-100">{row.source_image_ref}</p>
                      <p className="mt-1 text-[11px] text-sky-800 dark:text-sky-300">Release request accepted and awaiting completion.</p>
                      <button
                        type="button"
                        onClick={() => setSelected(row)}
                        className="mt-2 rounded-md border border-slate-300 px-2 py-1 text-[11px] font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                      >
                        Details
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className="rounded-md border border-indigo-200 bg-indigo-50/60 p-3 dark:border-indigo-800 dark:bg-indigo-950/20">
              <h3 className="text-sm font-semibold text-indigo-900 dark:text-indigo-200">Released ({releasedRows.length})</h3>
              {releasedRows.length === 0 ? (
                <p className="mt-2 text-xs text-indigo-800/80 dark:text-indigo-300">No released artifacts yet.</p>
              ) : (
                <div className="mt-2 space-y-2">
                  {releasedRows.map((row) => (
                    <div key={row.id} className="rounded-md border border-indigo-200 bg-white p-2.5 dark:border-indigo-800 dark:bg-slate-900">
                      <p className="text-xs font-medium text-slate-900 dark:text-slate-100">{row.source_image_ref}</p>
                      <p className="mt-1 text-[11px] text-slate-500 dark:text-slate-400">Available for tenant consumption.</p>
                      <button
                        type="button"
                        onClick={() => setSelected(row)}
                        className="mt-2 rounded-md border border-slate-300 px-2 py-1 text-[11px] font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                      >
                        Details
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </section>
      ) : null}

      {!isRequestsMode && !isEprMode ? (
        <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900">
          <h2 className="text-base font-semibold text-slate-900 dark:text-white">Recent Decisions</h2>
          {recentRows.length === 0 ? (
            <p className="mt-3 text-sm text-slate-500 dark:text-slate-400">No recent completed decisions.</p>
          ) : (
            <div className="mt-3 space-y-2">
              {recentRows.map((row) => (
                <div key={row.id} className="rounded-md border border-slate-200 p-2 text-xs text-slate-700 dark:border-slate-700 dark:text-slate-300">
                  <span className="font-semibold">{row.status}</span> • {row.source_image_ref}
                </div>
              ))}
            </div>
          )}
        </section>
      ) : null}

      <Drawer
        isOpen={Boolean(selected || selectedEpr)}
        onClose={() => {
          setSelected(null)
          setSelectedEpr(null)
        }}
        title={selectedEpr ? 'EPR Registration Detail' : 'Reviewer Request Detail'}
        description={selected ? selected.source_image_ref : selectedEpr ? selectedEpr.epr_record_id : undefined}
        width="xl"
      >
        {selected ? (
          <div className="space-y-4">
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Decision Timeline</p>
              {selected.decision_timeline ? (
                <ul className="mt-2 list-disc space-y-1 pl-4 text-xs text-slate-600 dark:text-slate-300">
                  <li>Decision: {selected.decision_timeline.decision_status || 'n/a'}</li>
                  <li>Step State: {selected.decision_timeline.workflow_step_status || 'n/a'}</li>
                  {selected.decision_timeline.decided_by_user_id ? <li>Decided By: {selected.decision_timeline.decided_by_user_id}</li> : null}
                  {selected.decision_timeline.decided_at ? <li>Decided At: {new Date(selected.decision_timeline.decided_at).toLocaleString()}</li> : null}
                  {selected.decision_timeline.decision_reason ? <li>Reason: {selected.decision_timeline.decision_reason}</li> : null}
                </ul>
              ) : (
                <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">No reviewer decision has been recorded yet.</p>
              )}
            </div>

            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Notification Reconciliation</p>
              {selected.notification_reconciliation ? (
                <ul className="mt-2 list-disc space-y-1 pl-4 text-xs text-slate-600 dark:text-slate-300">
                  <li>Delivery State: {selected.notification_reconciliation.delivery_state || 'pending'}</li>
                  <li>
                    Receipts: {selected.notification_reconciliation.receipt_count} / {selected.notification_reconciliation.expected_recipients}
                  </li>
                  <li>
                    In-App Notifications: {selected.notification_reconciliation.in_app_notification_count} / {selected.notification_reconciliation.expected_recipients}
                  </li>
                  {selected.notification_reconciliation.decision_event_type ? (
                    <li>Decision Event: {selected.notification_reconciliation.decision_event_type}</li>
                  ) : null}
                  {selected.notification_reconciliation.idempotency_key ? (
                    <li>Idempotency Key: {selected.notification_reconciliation.idempotency_key}</li>
                  ) : null}
                </ul>
              ) : (
                <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">No notification reconciliation checkpoints are available yet.</p>
              )}
            </div>

            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Request</p>
              <div className="mt-2 space-y-1 text-xs text-slate-600 dark:text-slate-300">
                <p><span className="font-semibold">Status:</span> {selected.status}</p>
                <p><span className="font-semibold">Sync State:</span> {getImportSyncStateLabel(selected.sync_state)}</p>
                {selected.execution_state && (
                  <p><span className="font-semibold">Execution State:</span> {getImportProgressLabel(selected)}</p>
                )}
                {selected.dispatch_queued_at ? <p><span className="font-semibold">Dispatch Queued:</span> {new Date(selected.dispatch_queued_at).toLocaleString()}</p> : null}
                {selected.pipeline_started_at ? <p><span className="font-semibold">Pipeline Started:</span> {new Date(selected.pipeline_started_at).toLocaleString()}</p> : null}
                {selected.evidence_ready_at ? <p><span className="font-semibold">Evidence Ready:</span> {new Date(selected.evidence_ready_at).toLocaleString()}</p> : null}
                {selected.release_ready_at ? <p><span className="font-semibold">Release Ready:</span> {new Date(selected.release_ready_at).toLocaleString()}</p> : null}
                <p><span className="font-semibold">Release State:</span> {selected.release_state || 'unknown'}</p>
                {selected.release_blocker_reason ? <p><span className="font-semibold">Release Blocker:</span> {selected.release_blocker_reason}</p> : null}
                {selected.failure_class ? <p><span className="font-semibold">Failure Class:</span> {selected.failure_class}</p> : null}
                {selected.failure_code ? <p><span className="font-semibold">Failure Code:</span> {selected.failure_code}</p> : null}
                <p><span className="font-semibold">EPR:</span> {selected.epr_record_id}</p>
                <p><span className="font-semibold">Source:</span> {selected.source_registry}</p>
                <p><span className="font-semibold">Image:</span> {selected.source_image_ref}</p>
              </div>
            </div>

            <div className={`rounded-lg border p-4 ${getImportDiagnosticClasses(getImportDiagnostic(selected).tone)}`}>
              <p className="text-xs font-semibold uppercase tracking-wide">Operational Diagnostics</p>
              <p className="mt-1 text-xs font-semibold">{getImportDiagnostic(selected).title}</p>
              <p className="mt-1 text-xs">{getImportDiagnostic(selected).message}</p>
              {getImportRemediationHint(selected) ? (
                <p className="mt-2 text-xs">
                  <span className="font-semibold">Recommended Action:</span> {getImportRemediationHint(selected)}
                </p>
              ) : null}
            </div>

            {hasMeaningfulJSONEvidence(selected.scan_summary_json) || hasMeaningfulJSONEvidence(selected.sbom_summary_json) ? (
              <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
                <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Evidence Preview</p>
                {hasMeaningfulJSONEvidence(selected.scan_summary_json) ? (
                  <>
                    <p className="mt-2 text-xs font-medium text-slate-700 dark:text-slate-200">Scan Summary JSON</p>
                    <pre className="mt-1 overflow-x-auto rounded-md border border-slate-200 bg-white p-2 text-[11px] text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">
                      <code>{parseJSONSummary(selected.scan_summary_json)}</code>
                    </pre>
                  </>
                ) : null}
                {hasMeaningfulJSONEvidence(selected.sbom_summary_json) ? (
                  <>
                    <p className="mt-3 text-xs font-medium text-slate-700 dark:text-slate-200">SBOM Summary JSON</p>
                    <pre className="mt-1 overflow-x-auto rounded-md border border-slate-200 bg-white p-2 text-[11px] text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">
                      <code>{parseJSONSummary(selected.sbom_summary_json)}</code>
                    </pre>
                  </>
                ) : null}
              </div>
            ) : null}
          </div>
        ) : selectedEpr ? (
          <div className="space-y-4">
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Actions</p>
              <div className="mt-3 flex flex-wrap items-center gap-2">
                {selectedEpr.status === 'pending' ? (
                  <>
                    <button
                      type="button"
                      onClick={() => void handleSORApprove(selectedEpr)}
                      disabled={Boolean(actionBusy[selectedEpr.id])}
                      className="rounded-md border border-emerald-300 bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-800 hover:bg-emerald-100 disabled:opacity-60 dark:border-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200 dark:hover:bg-emerald-900/50"
                    >
                      {actionBusy[selectedEpr.id] ? 'Submitting...' : 'Approve'}
                    </button>
                    <button
                      type="button"
                      onClick={() => void handleSORReject(selectedEpr)}
                      disabled={Boolean(actionBusy[selectedEpr.id])}
                      className="rounded-md border border-rose-300 bg-rose-50 px-2.5 py-1 text-xs font-medium text-rose-800 hover:bg-rose-100 disabled:opacity-60 dark:border-rose-700 dark:bg-rose-900/30 dark:text-rose-200 dark:hover:bg-rose-900/50"
                    >
                      {actionBusy[selectedEpr.id] ? 'Submitting...' : 'Reject'}
                    </button>
                  </>
                ) : null}
                {selectedEpr.status === 'approved' && selectedEpr.lifecycle_status !== 'suspended' ? (
                  <>
                    <button
                      type="button"
                      onClick={() => void handleSORSuspend(selectedEpr)}
                      disabled={Boolean(actionBusy[selectedEpr.id])}
                      className="rounded-md border border-rose-300 bg-rose-50 px-2.5 py-1 text-xs font-medium text-rose-800 hover:bg-rose-100 disabled:opacity-60 dark:border-rose-700 dark:bg-rose-900/30 dark:text-rose-200 dark:hover:bg-rose-900/50"
                    >
                      {actionBusy[selectedEpr.id] ? 'Submitting...' : 'Suspend'}
                    </button>
                    <button
                      type="button"
                      onClick={() => void handleSORRevalidate(selectedEpr)}
                      disabled={Boolean(actionBusy[selectedEpr.id])}
                      className="rounded-md border border-indigo-300 bg-indigo-50 px-2.5 py-1 text-xs font-medium text-indigo-800 hover:bg-indigo-100 disabled:opacity-60 dark:border-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-200 dark:hover:bg-indigo-900/50"
                    >
                      {actionBusy[selectedEpr.id] ? 'Submitting...' : 'Revalidate'}
                    </button>
                  </>
                ) : null}
                {selectedEpr.status === 'approved' && selectedEpr.lifecycle_status === 'suspended' ? (
                  <button
                    type="button"
                    onClick={() => void handleSORReactivate(selectedEpr)}
                    disabled={Boolean(actionBusy[selectedEpr.id])}
                    className="rounded-md border border-emerald-300 bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-800 hover:bg-emerald-100 disabled:opacity-60 dark:border-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200 dark:hover:bg-emerald-900/50"
                  >
                    {actionBusy[selectedEpr.id] ? 'Submitting...' : 'Reactivate'}
                  </button>
                ) : null}
                {selectedEpr.status !== 'pending' && selectedEpr.status !== 'approved' ? (
                  <span className="text-xs text-slate-500 dark:text-slate-400">No lifecycle actions available.</span>
                ) : null}
              </div>
            </div>
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">EPR Registration</p>
              <div className="mt-2 space-y-1 text-xs text-slate-700 dark:text-slate-300">
                <p><span className="font-semibold">EPR ID:</span> {selectedEpr.epr_record_id}</p>
                <p><span className="font-semibold">Status:</span> {selectedEpr.status}</p>
                <p><span className="font-semibold">Lifecycle:</span> {selectedEpr.lifecycle_status || 'active'}</p>
                <p><span className="font-semibold">Product:</span> {selectedEpr.product_name}</p>
                <p><span className="font-semibold">Technology:</span> {selectedEpr.technology_name}</p>
                {selectedEpr.source_registry ? <p><span className="font-semibold">Source Registry:</span> {selectedEpr.source_registry}</p> : null}
                {selectedEpr.source_image_example ? <p><span className="font-semibold">Source Image Example:</span> {selectedEpr.source_image_example}</p> : null}
                <p><span className="font-semibold">Submitted:</span> {new Date(selectedEpr.created_at).toLocaleString()}</p>
                <p><span className="font-semibold">Updated:</span> {new Date(selectedEpr.updated_at).toLocaleString()}</p>
                {selectedEpr.last_reviewed_at ? <p><span className="font-semibold">Last Reviewed:</span> {new Date(selectedEpr.last_reviewed_at).toLocaleString()}</p> : null}
                {selectedEpr.expires_at ? <p><span className="font-semibold">Expires:</span> {new Date(selectedEpr.expires_at).toLocaleString()}</p> : null}
                {selectedEpr.suspension_reason ? <p><span className="font-semibold">Suspension Reason:</span> {selectedEpr.suspension_reason}</p> : null}
              </div>
            </div>
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Business Justification</p>
              <p className="mt-2 text-xs text-slate-600 dark:text-slate-300">
                {selectedEpr.business_justification || 'No justification provided.'}
              </p>
            </div>
            {selectedEpr.additional_notes ? (
              <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
                <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Additional Notes</p>
                <p className="mt-2 text-xs text-slate-600 dark:text-slate-300">{selectedEpr.additional_notes}</p>
              </div>
            ) : null}
          </div>
        ) : null}
      </Drawer>
    </div>
  )
}

export default QuarantineReviewWorkbenchPage
