import type { ImageImportExecutionState, ImageImportRequest, ImageImportSyncState } from '@/types'

type DiagnosticTone = 'info' | 'warning' | 'error' | 'success'

export interface ImageImportDiagnostic {
  title: string
  message: string
  tone: DiagnosticTone
}

export const getImportSyncStateLabel = (syncState?: ImageImportSyncState) => {
  switch (syncState) {
    case 'awaiting_approval':
      return 'Awaiting Approval'
    case 'awaiting_dispatch':
      return 'Awaiting Dispatch'
    case 'pipeline_running':
      return 'Pipeline Running'
    case 'catalog_sync_pending':
      return 'Catalog Sync Pending'
    case 'dispatch_failed':
      return 'Dispatch Failed'
    case 'completed':
      return 'Completed'
    case 'failed':
      return 'Failed'
    default:
      return 'Unknown'
  }
}

export const getImportExecutionStateLabel = (executionState?: ImageImportExecutionState) => {
  switch (executionState) {
    case 'awaiting_approval':
      return 'Awaiting Approval'
    case 'awaiting_dispatch':
      return 'Awaiting Dispatch'
    case 'pipeline_running':
      return 'Pipeline Running'
    case 'evidence_pending':
      return 'Evidence Pending'
    case 'ready_for_release':
      return 'Ready For Release'
    case 'completed':
      return 'Completed'
    default:
      return 'Unknown'
  }
}

export const getImportProgressLabel = (request: ImageImportRequest) => {
  if (request.execution_state) {
    return getImportExecutionStateLabel(request.execution_state)
  }
  return getImportSyncStateLabel(request.sync_state)
}

export const getImportDiagnostic = (request: ImageImportRequest): ImageImportDiagnostic => {
  const syncState = request.sync_state
  const executionState = request.execution_state
  const errorMessage = request.error_message?.trim()

  if (syncState === 'dispatch_failed') {
    return {
      tone: 'error',
      title: 'Dispatch Failed',
      message: errorMessage || 'Request failed before pipeline execution. Retry is available.',
    }
  }

  if (syncState === 'catalog_sync_pending') {
    return {
      tone: 'warning',
      title: 'Catalog Sync Pending',
      message: 'Pipeline finished, but catalog projection is pending. Retry can continue the sync.',
    }
  }

  if (executionState === 'awaiting_approval' || syncState === 'awaiting_approval') {
    return {
      tone: 'info',
      title: 'Awaiting Approval',
      message: 'Request is pending reviewer approval. Dispatch begins after approval.',
    }
  }

  if (executionState === 'evidence_pending') {
    return {
      tone: 'warning',
      title: 'Evidence Pending',
      message: 'Pipeline execution completed, but evidence projection is still pending. Refresh for latest state.',
    }
  }

  if (syncState === 'awaiting_dispatch') {
    return {
      tone: 'info',
      title: getImportSyncStateLabel(syncState),
      message: 'No eligible Tekton quarantine provider is currently available. Ensure a provider has tekton_enabled=true and quarantine_dispatch_enabled=true, then refresh.',
    }
  }

  if (syncState === 'pipeline_running') {
    return {
      tone: 'info',
      title: getImportSyncStateLabel(syncState),
      message: 'Request is still processing. Refresh for the latest state.',
    }
  }

  if (request.status === 'failed' || syncState === 'failed') {
    return {
      tone: 'error',
      title: 'Runtime Failed',
      message: errorMessage || 'Request reached a terminal failure state.',
    }
  }

  if (request.status === 'quarantined') {
    return {
      tone: 'warning',
      title: 'Quarantined',
      message: 'Policy blocked release. Review policy decision and evidence summary.',
    }
  }

  if (request.status === 'success') {
    return {
      tone: 'success',
      title: 'Completed',
      message: 'Request completed and evidence is available.',
    }
  }

  return {
    tone: 'info',
    title: 'Pending',
    message: 'Request is queued for approval and processing.',
  }
}

export const getImportRemediationHint = (request: ImageImportRequest): string | null => {
  const failureClass = request.failure_class
  const failureCode = request.failure_code
  const releaseBlocker = request.release_blocker_reason?.trim().toLowerCase()

  if (releaseBlocker) {
    switch (releaseBlocker) {
      case 'evidence_incomplete':
        return 'Capture complete policy, SBOM, and vulnerability evidence before release.'
      case 'evidence_stale':
        return 'Re-run quarantine processing to refresh stale evidence before release.'
      case 'policy_not_passed':
      case 'policy_quarantined':
        return 'Resolve policy findings or exceptions before attempting release.'
      default:
        break
    }
  }

  if (!failureClass && !failureCode) {
    return null
  }

  switch (failureCode) {
    case 'dispatcher_unavailable':
      return 'Enable a provider with tekton_enabled=true and quarantine_dispatch_enabled=true, then retry.'
    case 'dispatch_timeout':
      return 'Check provider/API reachability and Tekton controller health before retrying.'
    case 'auth_error':
      return 'Validate provider runtime_auth credentials and namespace permissions, then retry.'
    case 'connectivity_error':
      return 'Validate network/DNS/connectivity between backend and cluster APIs before retrying.'
    case 'policy_blocked':
    case 'quarantined_by_policy':
      return 'Review policy decision details and remediate findings before requesting another run.'
    case 'runtime_failed':
      return 'Inspect pipeline logs/evidence output and rerun after fixing the failing stage.'
    default:
      break
  }

  switch (failureClass) {
    case 'dispatch':
      return 'Review provider dispatch readiness and Tekton availability, then retry.'
    case 'auth':
      return 'Review provider auth settings and RBAC permissions, then retry.'
    case 'connectivity':
      return 'Review network connectivity and DNS reachability for runtime dependencies.'
    case 'policy':
      return 'Review policy constraints and update request inputs or approvals before retrying.'
    case 'runtime':
      return 'Inspect pipeline execution logs for failing steps before retrying.'
    default:
      return null
  }
}

export const getImportDiagnosticClasses = (tone: DiagnosticTone) => {
  switch (tone) {
    case 'error':
      return 'border-rose-300 bg-rose-50 text-rose-800 dark:border-rose-700 dark:bg-rose-950/40 dark:text-rose-200'
    case 'warning':
      return 'border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200'
    case 'success':
      return 'border-emerald-300 bg-emerald-50 text-emerald-800 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-200'
    case 'info':
    default:
      return 'border-sky-300 bg-sky-50 text-sky-800 dark:border-sky-700 dark:bg-sky-950/40 dark:text-sky-200'
  }
}

export const hasMeaningfulJSONEvidence = (raw?: string) => {
  const trimmed = raw?.trim()
  return Boolean(trimmed && trimmed !== '{}' && trimmed !== '[]')
}
