import Drawer from '@/components/ui/Drawer'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import { registryAuthClient } from '@/api/registryAuthClient'
import { ImageImportApiError, imageImportService } from '@/services/imageImportService'
import { getImportDiagnostic, getImportDiagnosticClasses, getImportProgressLabel, getImportRemediationHint, getImportSyncStateLabel, hasMeaningfulJSONEvidence } from '@/utils/imageImportDiagnostics'
import type { ImageImportRequest } from '@/types'
import type { RegistryAuth } from '@/types/registryAuth'
import { AlertTriangle, CheckCircle2, Clock, ShieldAlert } from 'lucide-react'
import React, { useCallback, useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import { Link, useLocation } from 'react-router-dom'

const OnDemandScansPage: React.FC = () => {
    const location = useLocation()
    const isAdminView = location.pathname.startsWith('/admin/')
    const confirmDialog = useConfirmDialog()
    const [scanRequests, setScanRequests] = useState<ImageImportRequest[]>([])
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [isEntitlementDenied, setIsEntitlementDenied] = useState(false)
    const [selectedRequest, setSelectedRequest] = useState<ImageImportRequest | null>(null)
    const [retryingIds, setRetryingIds] = useState<Record<string, boolean>>({})
    const [retryActionError, setRetryActionError] = useState<string | null>(null)
    const [submitting, setSubmitting] = useState(false)
    const [sourceRegistry, setSourceRegistry] = useState('')
    const [sourceImageRef, setSourceImageRef] = useState('')
    const [registryAuthId, setRegistryAuthId] = useState('')
    const [registryAccessMode, setRegistryAccessMode] = useState<'public' | 'private'>('public')
    const [registryAuthOptions, setRegistryAuthOptions] = useState<RegistryAuth[]>([])
    const [registryAuthLoading, setRegistryAuthLoading] = useState(false)
    const [registryAuthError, setRegistryAuthError] = useState<string | null>(null)
    const [formError, setFormError] = useState<string | null>(null)

    const loadScanRequests = useCallback(async () => {
        try {
            setLoading(true)
            setError(null)
            setIsEntitlementDenied(false)
            const rows = await imageImportService.listScanRequests(25)
            setScanRequests(rows)
        } catch (err) {
            const apiErr = err as Error
            setError(apiErr.message || 'Failed to load on-demand scan requests')
            setIsEntitlementDenied(err instanceof ImageImportApiError && err.code === 'tenant_capability_not_entitled')
            setScanRequests([])
        } finally {
            setLoading(false)
        }
    }, [])

    useEffect(() => {
        loadScanRequests()
    }, [loadScanRequests])

    useEffect(() => {
        setRetryActionError(null)
    }, [selectedRequest?.id])

    useEffect(() => {
        let active = true
        const loadRegistryAuth = async () => {
            try {
                setRegistryAuthLoading(true)
                setRegistryAuthError(null)
                const response = await registryAuthClient.listRegistryAuth(undefined, true)
                if (!active) {
                    return
                }
                setRegistryAuthOptions(response.registry_auth || [])
            } catch {
                if (!active) {
                    return
                }
                setRegistryAuthOptions([])
                setRegistryAuthError('Failed to load registry authentications')
            } finally {
                if (active) {
                    setRegistryAuthLoading(false)
                }
            }
        }
        loadRegistryAuth()
        return () => {
            active = false
        }
    }, [])

    const getImportStatusColor = (status: ImageImportRequest['status']) => {
        switch (status) {
            case 'success': return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-200'
            case 'quarantined': return 'bg-amber-100 text-amber-800 dark:bg-amber-900/50 dark:text-amber-200'
            case 'failed': return 'bg-rose-100 text-rose-800 dark:bg-rose-900/50 dark:text-rose-200'
            case 'importing': return 'bg-blue-100 text-blue-800 dark:bg-blue-900/50 dark:text-blue-200'
            case 'approved': return 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900/50 dark:text-indigo-200'
            case 'pending':
            default: return 'bg-slate-100 text-slate-800 dark:bg-slate-700 dark:text-slate-200'
        }
    }

    const parseJSONSummary = (raw?: string) => {
        if (!raw) return null
        try {
            return JSON.stringify(JSON.parse(raw), null, 2)
        } catch {
            return raw
        }
    }

    const handleSubmit = async (event: React.FormEvent) => {
        event.preventDefault()
        if (!sourceRegistry.trim() || !sourceImageRef.trim()) {
            setFormError('Source registry and source image reference are required.')
            return
        }
        if (registryAccessMode === 'private' && !registryAuthId.trim()) {
            setFormError('Registry auth is required for private registries.')
            return
        }
        setSubmitting(true)
        setFormError(null)
        try {
            const created = await imageImportService.createScanRequest({
                sourceRegistry: sourceRegistry.trim(),
                sourceImageRef: sourceImageRef.trim(),
                registryAuthId: registryAccessMode === 'private' ? registryAuthId.trim() || undefined : undefined,
            })
            setScanRequests((prev) => [created, ...prev.filter((row) => row.id !== created.id)])
            setSourceRegistry('')
            setSourceImageRef('')
            setRegistryAuthId('')
            setRegistryAccessMode('public')
            toast.success('On-demand scan request submitted')
        } catch (err) {
            const apiErr = err as Error
            setFormError(apiErr.message || 'Failed to submit on-demand scan request')
        } finally {
            setSubmitting(false)
        }
    }

    const handleRetry = async (row: ImageImportRequest) => {
        const confirmed = await confirmDialog({
            title: 'Retry On-Demand Scan',
            message: `Retry scan request for ${row.source_image_ref}?`,
            confirmLabel: 'Retry',
        })
        if (!confirmed) return

        try {
            setRetryingIds((prev) => ({ ...prev, [row.id]: true }))
            setRetryActionError(null)
            const retried = await imageImportService.retryScanRequest(row.id)
            setScanRequests((prev) => [retried, ...prev.filter((item) => item.id !== retried.id)])
            setSelectedRequest(retried)
            toast.success('Retry submitted successfully')
        } catch (err) {
            const apiErr = err as ImageImportApiError
            const message = apiErr.message || 'Failed to retry on-demand scan request'
            toast.error(message)
            if (apiErr.code === 'retry_backoff_active' || apiErr.code === 'retry_attempt_limit_reached') {
                setRetryActionError(message)
            }
        } finally {
            setRetryingIds((prev) => ({ ...prev, [row.id]: false }))
        }
    }

    return (
        <div className="space-y-6 px-4 py-6 sm:px-6 lg:px-8">
            <div>
                <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">On-Demand Scans</h1>
                <p className="mt-2 text-sm text-slate-700 dark:text-slate-400">
                    {isAdminView
                        ? 'Review on-demand scan requests and take follow-up actions.'
                        : 'Submit external image references for asynchronous vulnerability and SBOM analysis. Results are tracked here and delivered via notifications.'}
                </p>
            </div>

            <section className="rounded-lg border border-cyan-200 bg-cyan-50/70 p-4 dark:border-cyan-800 dark:bg-cyan-950/20">
                <h2 className="text-sm font-semibold text-cyan-900 dark:text-cyan-200">What To Expect</h2>
                <div className="mt-3 grid grid-cols-1 gap-3 md:grid-cols-3">
                    <div className="rounded-md border border-cyan-200 bg-white/90 p-3 dark:border-cyan-800 dark:bg-slate-900/70">
                        <p className="text-xs font-semibold uppercase tracking-wide text-cyan-700 dark:text-cyan-300">1. Queue Request</p>
                        <p className="mt-1 text-xs text-slate-700 dark:text-slate-300">
                            Submit image reference, mark registry as public/private, and attach auth if private.
                        </p>
                    </div>
                    <div className="rounded-md border border-cyan-200 bg-white/90 p-3 dark:border-cyan-800 dark:bg-slate-900/70">
                        <p className="text-xs font-semibold uppercase tracking-wide text-cyan-700 dark:text-cyan-300">2. Async Processing</p>
                        <p className="mt-1 text-xs text-slate-700 dark:text-slate-300">
                            Worker pipeline pulls, scans, and generates SBOM/vulnerability summaries.
                        </p>
                    </div>
                    <div className="rounded-md border border-cyan-200 bg-white/90 p-3 dark:border-cyan-800 dark:bg-slate-900/70">
                        <p className="text-xs font-semibold uppercase tracking-wide text-cyan-700 dark:text-cyan-300">3. Review Results</p>
                        <p className="mt-1 text-xs text-slate-700 dark:text-slate-300">
                            History rows and notifications show completion status, errors, and evidence summaries.
                        </p>
                    </div>
                </div>
            </section>

            {!isAdminView && !isEntitlementDenied ? (
                <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6">
                    <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-1 flex items-center">
                        <ShieldAlert className="w-5 h-5 mr-2 text-blue-600 dark:text-blue-400" />
                        Submit Scan Request
                    </h3>
                    <p className="text-xs text-slate-600 dark:text-slate-300 mb-4">
                        Provide image location details. The scan is queued and processed asynchronously.
                    </p>
                    {formError ? (
                        <div className="mb-4 flex items-start gap-2 rounded-md border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800 dark:border-rose-800 dark:bg-rose-950/30 dark:text-rose-200">
                            <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                            <span>{formError}</span>
                        </div>
                    ) : null}
                    <form onSubmit={handleSubmit} className="grid grid-cols-1 md:grid-cols-3 gap-3">
                        <input
                            value={sourceRegistry}
                            onChange={(e) => setSourceRegistry(e.target.value)}
                            placeholder="Source registry (e.g. registry-1.docker.io)"
                            className="w-full rounded-md border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30"
                        />
                        <input
                            value={sourceImageRef}
                            onChange={(e) => setSourceImageRef(e.target.value)}
                            placeholder="Source image ref (e.g. library/nginx:1.27)"
                            className="w-full rounded-md border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30"
                        />
                        <select
                            value={registryAccessMode}
                            onChange={(e) => {
                                const mode = e.target.value as 'public' | 'private'
                                setRegistryAccessMode(mode)
                                if (mode === 'public') {
                                    setRegistryAuthId('')
                                }
                            }}
                            className="w-full rounded-md border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30"
                        >
                            <option value="public">Public registry (no auth needed)</option>
                            <option value="private">Private registry (auth required)</option>
                        </select>
                        {registryAccessMode === 'private' ? (
                            <div className="md:col-span-3">
                                <select
                                    value={registryAuthId}
                                    onChange={(e) => setRegistryAuthId(e.target.value)}
                                    disabled={registryAuthLoading}
                                    className="w-full rounded-md border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 disabled:opacity-60"
                                >
                                    <option value="">{registryAuthLoading ? 'Loading registry auth...' : 'Select registry authentication'}</option>
                                    {registryAuthOptions.map((auth) => (
                                        <option key={auth.id} value={auth.id}>
                                            {auth.name} ({auth.scope})
                                        </option>
                                    ))}
                                </select>
                                {registryAuthError ? (
                                    <p className="mt-1 text-xs text-rose-700 dark:text-rose-300">{registryAuthError}</p>
                                ) : null}
                                {!registryAuthLoading && registryAuthOptions.length === 0 ? (
                                    <p className="mt-1 text-xs text-amber-700 dark:text-amber-300">
                                        No registry auth found. <Link to="/settings/auth" className="underline font-medium">Create one first</Link>.
                                    </p>
                                ) : null}
                            </div>
                        ) : null}
                        <button
                            type="submit"
                            disabled={submitting}
                            className="md:col-span-3 inline-flex items-center justify-center rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-slate-400 dark:disabled:bg-slate-600"
                        >
                            {submitting ? 'Submitting...' : 'Queue On-Demand Scan'}
                        </button>
                    </form>
                </div>
            ) : null}

            <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900">
                <div className="mb-4 flex items-center justify-between">
                    <h2 className="text-base font-semibold text-slate-900 dark:text-white">Scan Request History</h2>
                    <button
                        type="button"
                        onClick={() => loadScanRequests()}
                        className="rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                    >
                        Refresh
                    </button>
                </div>

                {loading ? (
                    <p className="text-sm text-slate-500 dark:text-slate-400">Loading scan request activity...</p>
                ) : error ? (
                    <div className="rounded-md border border-rose-200 bg-rose-50 p-3 text-sm text-rose-700 dark:border-rose-700 dark:bg-rose-950/40 dark:text-rose-200">
                        <p>{error}</p>
                        {isEntitlementDenied ? (
                            <p className="mt-2 text-xs text-rose-700 dark:text-rose-300">
                                Review capability status in{' '}
                                <Link to="/help/capability-access" className="underline font-medium">
                                    Capability Matrix
                                </Link>
                                .
                            </p>
                        ) : null}
                    </div>
                ) : scanRequests.length === 0 ? (
                    <p className="text-sm text-slate-500 dark:text-slate-400">No on-demand scan requests yet.</p>
                ) : (
                    <div className="space-y-3">
                        {scanRequests.map((row) => (
                            <div key={row.id} className="rounded-md border border-slate-200 p-3 dark:border-slate-700">
                                <div className="mb-2 flex flex-wrap items-center gap-2">
                                    <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${getImportStatusColor(row.status)}`}>
                                        {row.status}
                                    </span>
                                    <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-700 dark:bg-slate-700 dark:text-slate-200">
                                        {getImportProgressLabel(row)}
                                    </span>
                                    {row.retryable ? (
                                        <span className="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-xs text-amber-800 dark:bg-amber-900/50 dark:text-amber-200">
                                            <Clock className="h-3 w-3" />
                                            Retryable
                                        </span>
                                    ) : (
                                        <span className="inline-flex items-center gap-1 rounded-full bg-emerald-100 px-2 py-0.5 text-xs text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-200">
                                            <CheckCircle2 className="h-3 w-3" />
                                            Stable
                                        </span>
                                    )}
                                </div>
                                <p className="text-sm text-slate-900 dark:text-slate-100">{row.source_image_ref}</p>
                                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                    Registry {row.source_registry} • Updated {new Date(row.updated_at).toLocaleString()}
                                </p>
                                {(() => {
                                    const diagnostic = getImportDiagnostic(row)
                                    return (
                                        <div className={`mt-2 rounded-md border px-2 py-1 text-xs ${getImportDiagnosticClasses(diagnostic.tone)}`}>
                                            <p className="font-semibold">{diagnostic.title}</p>
                                            <p className="mt-0.5">{diagnostic.message}</p>
                                        </div>
                                    )
                                })()}
                                <div className="mt-2 flex items-center gap-2">
                                    <button
                                        type="button"
                                        onClick={() => setSelectedRequest(row)}
                                        className="rounded-md border border-slate-300 px-2.5 py-1 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                                    >
                                        Details
                                    </button>
                                    {row.retryable ? (
                                        <button
                                            type="button"
                                            onClick={() => handleRetry(row)}
                                            disabled={Boolean(retryingIds[row.id])}
                                            className="rounded-md border border-amber-300 bg-amber-50 px-2.5 py-1 text-xs font-medium text-amber-800 hover:bg-amber-100 disabled:opacity-60 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200 dark:hover:bg-amber-900/40"
                                        >
                                            {retryingIds[row.id] ? 'Retrying...' : 'Retry'}
                                        </button>
                                    ) : null}
                                </div>
                                {row.error_message ? (
                                    <div className="mt-2 inline-flex items-center gap-1 rounded-md border border-amber-300 bg-amber-50 px-2 py-1 text-xs text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200">
                                        <AlertTriangle className="h-3 w-3" />
                                        {row.error_message}
                                    </div>
                                ) : null}
                            </div>
                        ))}
                    </div>
                )}
            </section>

            <Drawer
                isOpen={Boolean(selectedRequest)}
                onClose={() => setSelectedRequest(null)}
                title="On-Demand Scan Request Detail"
                description={selectedRequest ? selectedRequest.source_image_ref : undefined}
                width="xl"
            >
                {selectedRequest ? (
                    <div className="space-y-4">
                        <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
                            <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Timeline</p>
                            <ul className="mt-2 list-disc space-y-1 pl-4 text-xs text-slate-600 dark:text-slate-300">
                                <li>Created: {new Date(selectedRequest.created_at).toLocaleString()}</li>
                                <li>Last Updated: {new Date(selectedRequest.updated_at).toLocaleString()}</li>
                                <li>Status: {selectedRequest.status}</li>
                                <li>Sync State: {getImportSyncStateLabel(selectedRequest.sync_state)}</li>
                                {selectedRequest.execution_state && (
                                  <li>Execution State: {getImportProgressLabel(selectedRequest)}</li>
                                )}
                                {selectedRequest.dispatch_queued_at ? <li>Dispatch Queued: {new Date(selectedRequest.dispatch_queued_at).toLocaleString()}</li> : null}
                                {selectedRequest.pipeline_started_at ? <li>Pipeline Started: {new Date(selectedRequest.pipeline_started_at).toLocaleString()}</li> : null}
                                {selectedRequest.evidence_ready_at ? <li>Evidence Ready: {new Date(selectedRequest.evidence_ready_at).toLocaleString()}</li> : null}
                                {selectedRequest.release_ready_at ? <li>Release Ready: {new Date(selectedRequest.release_ready_at).toLocaleString()}</li> : null}
                                {selectedRequest.pipeline_run_name ? (
                                    <li>PipelineRun: {selectedRequest.pipeline_namespace || 'default'}/{selectedRequest.pipeline_run_name}</li>
                                ) : null}
                                {selectedRequest.policy_decision ? <li>Policy Decision: {selectedRequest.policy_decision}</li> : null}
                            </ul>
                        </div>

                            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
                                <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Request Data</p>
                                <div className="mt-2 space-y-1 text-xs text-slate-600 dark:text-slate-300">
                                <p><span className="font-semibold">Source Registry:</span> {selectedRequest.source_registry}</p>
                                <p><span className="font-semibold">Source Image:</span> {selectedRequest.source_image_ref}</p>
                                <p><span className="font-semibold">Retryable:</span> {selectedRequest.retryable ? 'Yes' : 'No'}</p>
                                {selectedRequest.error_message ? <p><span className="font-semibold">Last Error:</span> {selectedRequest.error_message}</p> : null}
                                {selectedRequest.failure_class ? <p><span className="font-semibold">Failure Class:</span> {selectedRequest.failure_class}</p> : null}
                                {selectedRequest.failure_code ? <p><span className="font-semibold">Failure Code:</span> {selectedRequest.failure_code}</p> : null}
                                </div>
                                {retryActionError ? (
                                  <div className="mt-3 rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200">
                                    <span className="font-semibold">Retry Policy:</span> {retryActionError}
                                  </div>
                                ) : null}
                            </div>

                        <div className={`rounded-lg border p-4 ${getImportDiagnosticClasses(getImportDiagnostic(selectedRequest).tone)}`}>
                            <p className="text-xs font-semibold uppercase tracking-wide">Operational Diagnostics</p>
                            <p className="mt-1 text-xs font-semibold">{getImportDiagnostic(selectedRequest).title}</p>
                            <p className="mt-1 text-xs">{getImportDiagnostic(selectedRequest).message}</p>
                            {getImportRemediationHint(selectedRequest) ? (
                              <p className="mt-2 text-xs">
                                <span className="font-semibold">Recommended Action:</span> {getImportRemediationHint(selectedRequest)}
                              </p>
                            ) : null}
                        </div>

                        {hasMeaningfulJSONEvidence(selectedRequest.scan_summary_json) || hasMeaningfulJSONEvidence(selectedRequest.sbom_summary_json) ? (
                            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
                                <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Evidence Summary</p>
                                {hasMeaningfulJSONEvidence(selectedRequest.scan_summary_json) ? (
                                    <>
                                        <p className="mt-2 text-xs font-medium text-slate-700 dark:text-slate-200">Scan Summary JSON</p>
                                        <pre className="mt-1 overflow-x-auto rounded-md border border-slate-200 bg-white p-2 text-[11px] text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">
                                            <code>{parseJSONSummary(selectedRequest.scan_summary_json)}</code>
                                        </pre>
                                    </>
                                ) : null}
                                {hasMeaningfulJSONEvidence(selectedRequest.sbom_summary_json) ? (
                                    <>
                                        <p className="mt-3 text-xs font-medium text-slate-700 dark:text-slate-200">SBOM Summary JSON</p>
                                        <pre className="mt-1 overflow-x-auto rounded-md border border-slate-200 bg-white p-2 text-[11px] text-slate-700 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">
                                            <code>{parseJSONSummary(selectedRequest.sbom_summary_json)}</code>
                                        </pre>
                                    </>
                                ) : null}
                            </div>
                        ) : null}
                    </div>
                ) : null}
            </Drawer>
        </div>
    )
}

export default OnDemandScansPage
