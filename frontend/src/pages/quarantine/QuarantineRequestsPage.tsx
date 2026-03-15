import Drawer from '@/components/ui/Drawer'
import ExternalImportRequestForm from '@/components/images/ExternalImportRequestForm'
import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import { ImageImportApiError, imageImportService } from '@/services/imageImportService'
import { eprRegistrationService } from '@/services/eprRegistrationService'
import { mapQuarantineImportErrorMessage } from '@/utils/quarantineErrorMessages'
import { getImportDiagnostic, getImportDiagnosticClasses, getImportProgressLabel, getImportRemediationHint, getImportSyncStateLabel, hasMeaningfulJSONEvidence } from '@/utils/imageImportDiagnostics'
import type { ImageImportRequest, EPRRegistrationRequest } from '@/types'
import { AlertTriangle, CheckCircle2, Clock } from 'lucide-react'
import React, { useCallback, useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import { Link } from 'react-router-dom'

const EPRTerm: React.FC = () => (
    <abbr
        title="Enterprise Product/Technology Registry"
        className="cursor-help border-b border-dotted border-slate-400 no-underline dark:border-slate-500"
    >
        EPR
    </abbr>
)

const generateEPRRecordID = () => {
    const date = new Date()
    const y = date.getUTCFullYear()
    const m = String(date.getUTCMonth() + 1).padStart(2, '0')
    const d = String(date.getUTCDate()).padStart(2, '0')
    const suffix = Math.random().toString(36).slice(2, 10).toUpperCase()
    return `EPR-${y}${m}${d}-${suffix}`
}

interface QuarantineRequestsPageProps {
    mode?: 'all' | 'requests' | 'epr'
}

const QuarantineRequestsPage: React.FC<QuarantineRequestsPageProps> = ({ mode = 'all' }) => {
    const confirmDialog = useConfirmDialog()
    const [importRequests, setImportRequests] = useState<ImageImportRequest[]>([])
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [isEntitlementDenied, setIsEntitlementDenied] = useState(false)
    const [selectedRequest, setSelectedRequest] = useState<ImageImportRequest | null>(null)
    const [selectedEprRequest, setSelectedEprRequest] = useState<EPRRegistrationRequest | null>(null)
    const [retryingIds, setRetryingIds] = useState<Record<string, boolean>>({})
    const [retryActionError, setRetryActionError] = useState<string | null>(null)
    const [eprRequests, setEprRequests] = useState<EPRRegistrationRequest[]>([])
    const [eprLoading, setEprLoading] = useState(false)
    const [eprSubmitting, setEprSubmitting] = useState(false)
    const [createEprDrawerOpen, setCreateEprDrawerOpen] = useState(false)
    const [createImportDrawerOpen, setCreateImportDrawerOpen] = useState(false)
    const [importDraftValues, setImportDraftValues] = useState<{
        eprRecordId: string
        sourceRegistry: string
        sourceImageRef: string
        registryAuthId?: string
    } | null>(null)
    const [withdrawingImportIds, setWithdrawingImportIds] = useState<Record<string, boolean>>({})
    const [withdrawingEprIds, setWithdrawingEprIds] = useState<Record<string, boolean>>({})
    const [queueTab, setQueueTab] = useState<'all' | 'pending' | 'in_progress' | 'completed' | 'failed'>('all')
    const [queueSearch, setQueueSearch] = useState('')
    const [queueSyncState, setQueueSyncState] = useState<'all' | string>('all')
    const [queueRetryable, setQueueRetryable] = useState<'all' | 'retryable' | 'stable'>('all')
    const [queueSort, setQueueSort] = useState<'updated_desc' | 'updated_asc' | 'created_desc' | 'created_asc'>('updated_desc')
    const [queuePage, setQueuePage] = useState(1)
    const [eprForm, setEprForm] = useState({
        eprRecordId: generateEPRRecordID(),
        productName: '',
        technologyName: '',
        businessJustification: '',
    })
    const [eprFormError, setEprFormError] = useState<string | null>(null)
    const isRequestsMode = mode === 'requests'
    const isEprMode = mode === 'epr'

    const loadImportRequests = useCallback(async () => {
        try {
            setLoading(true)
            setError(null)
            setIsEntitlementDenied(false)
            const rows = await imageImportService.listImportRequests(200)
            setImportRequests(rows)
        } catch (err) {
            const message = mapQuarantineImportErrorMessage(err, 'Failed to load quarantine requests')
            setError(message)
            setIsEntitlementDenied(err instanceof ImageImportApiError && err.code === 'tenant_capability_not_entitled')
            setImportRequests([])
        } finally {
            setLoading(false)
        }
    }, [])

    useEffect(() => {
        loadImportRequests()
    }, [loadImportRequests])

    useEffect(() => {
        setRetryActionError(null)
    }, [selectedRequest?.id])

    const loadEPRRequests = useCallback(async () => {
        try {
            setEprLoading(true)
            const rows = await eprRegistrationService.listTenantRequests({ limit: 20, page: 1 })
            setEprRequests(rows)
        } catch {
            setEprRequests([])
        } finally {
            setEprLoading(false)
        }
    }, [])

    useEffect(() => {
        loadEPRRequests()
    }, [loadEPRRequests])

    const latestEPRRequest = useMemo(() => eprRequests[0] || null, [eprRequests])
    const hasApprovedEPR = useMemo(
        () => eprRequests.some((row) => row.status === 'approved'),
        [eprRequests]
    )
    const hasPendingEPR = useMemo(
        () => eprRequests.some((row) => row.status === 'pending'),
        [eprRequests]
    )
    const hasRejectedEPR = useMemo(
        () => eprRequests.some((row) => row.status === 'rejected'),
        [eprRequests]
    )
    const hasWithdrawnEPR = useMemo(
        () => eprRequests.some((row) => row.status === 'withdrawn'),
        [eprRequests]
    )

    const intakeStateMessage = useMemo(() => {
        if (hasApprovedEPR) {
            return {
                tone: 'emerald',
                title: 'EPR approved',
                message: 'Approved EPR registrations exist. Create a quarantine request for the source image.',
            }
        }
        if (hasPendingEPR) {
            return {
                tone: 'amber',
                title: 'Awaiting security review',
                message: 'EPR registration is pending approval. You can track status in the EPR list below.',
            }
        }
        if (hasRejectedEPR) {
            return {
                tone: 'rose',
                title: 'EPR rejected',
                message: 'Latest EPR registration was rejected. Submit an updated EPR registration request.',
            }
        }
        if (hasWithdrawnEPR) {
            return {
                tone: 'amber',
                title: 'EPR withdrawn',
                message: 'Latest EPR registration was withdrawn. Submit a new EPR registration request when ready.',
            }
        }
        return {
            tone: 'sky',
            title: 'No EPR registration requests yet',
            message: 'Create an EPR registration if product/technology is not yet in the enterprise registry, or create a quarantine request if it is already registered.',
        }
    }, [hasApprovedEPR, hasPendingEPR, hasRejectedEPR, hasWithdrawnEPR])

    const intakeBannerClasses = intakeStateMessage.tone === 'emerald'
        ? 'border-emerald-300 bg-emerald-50 text-emerald-900 dark:border-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-200'
        : intakeStateMessage.tone === 'rose'
            ? 'border-rose-300 bg-rose-50 text-rose-900 dark:border-rose-700 dark:bg-rose-900/20 dark:text-rose-200'
            : intakeStateMessage.tone === 'amber'
                ? 'border-amber-300 bg-amber-50 text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200'
                : 'border-sky-300 bg-sky-50 text-sky-900 dark:border-sky-700 dark:bg-sky-900/20 dark:text-sky-200'

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

    const queueCounts = useMemo(() => {
        return importRequests.reduce(
            (acc, row) => {
                acc.all += 1
                if (row.status === 'pending' || row.status === 'approved') {
                    acc.pending += 1
                } else if (row.status === 'importing') {
                    acc.in_progress += 1
                } else if (row.status === 'success' || row.status === 'quarantined') {
                    acc.completed += 1
                } else if (row.status === 'failed') {
                    acc.failed += 1
                }
                return acc
            },
            { all: 0, pending: 0, in_progress: 0, completed: 0, failed: 0 }
        )
    }, [importRequests])

    const availableSyncStates = useMemo(() => {
        return Array.from(new Set(importRequests.map((row) => row.sync_state).filter(Boolean)))
    }, [importRequests])

    const filteredImportRequests = useMemo(() => {
        const normalizedSearch = queueSearch.trim().toLowerCase()
        const rows = importRequests.filter((row) => {
            if (queueTab === 'pending' && row.status !== 'pending' && row.status !== 'approved') {
                return false
            }
            if (queueTab === 'in_progress' && row.status !== 'importing') {
                return false
            }
            if (queueTab === 'completed' && row.status !== 'success' && row.status !== 'quarantined') {
                return false
            }
            if (queueTab === 'failed' && row.status !== 'failed') {
                return false
            }
            if (queueSyncState !== 'all' && row.sync_state !== queueSyncState) {
                return false
            }
            if (queueRetryable === 'retryable' && !row.retryable) {
                return false
            }
            if (queueRetryable === 'stable' && row.retryable) {
                return false
            }
            if (normalizedSearch) {
                const haystack = [
                    row.source_image_ref,
                    row.epr_record_id,
                    row.source_registry,
                    row.status,
                    row.sync_state,
                ]
                    .filter(Boolean)
                    .join(' ')
                    .toLowerCase()
                if (!haystack.includes(normalizedSearch)) {
                    return false
                }
            }
            return true
        })

        return rows.sort((a, b) => {
            const updatedA = new Date(a.updated_at).getTime()
            const updatedB = new Date(b.updated_at).getTime()
            const createdA = new Date(a.created_at).getTime()
            const createdB = new Date(b.created_at).getTime()
            switch (queueSort) {
                case 'updated_asc':
                    return updatedA - updatedB
                case 'created_desc':
                    return createdB - createdA
                case 'created_asc':
                    return createdA - createdB
                case 'updated_desc':
                default:
                    return updatedB - updatedA
            }
        })
    }, [importRequests, queueRetryable, queueSearch, queueSort, queueSyncState, queueTab])

    const queuePageSize = 10
    const queueTotalPages = Math.max(1, Math.ceil(filteredImportRequests.length / queuePageSize))
    const safeQueuePage = Math.min(queuePage, queueTotalPages)
    const pagedImportRequests = useMemo(() => {
        const start = (safeQueuePage - 1) * queuePageSize
        return filteredImportRequests.slice(start, start + queuePageSize)
    }, [filteredImportRequests, safeQueuePage])

    useEffect(() => {
        setQueuePage(1)
    }, [queueTab, queueSearch, queueSyncState, queueRetryable, queueSort])

    useEffect(() => {
        if (queuePage > queueTotalPages) {
            setQueuePage(queueTotalPages)
        }
    }, [queuePage, queueTotalPages])

    const hasQueueFilters =
        queueTab !== 'all' ||
        queueSearch.trim().length > 0 ||
        queueSyncState !== 'all' ||
        queueRetryable !== 'all' ||
        queueSort !== 'updated_desc'

    const handleRetry = async (row: ImageImportRequest) => {
        const confirmed = await confirmDialog({
            title: 'Retry Quarantine Request',
            message: `Retry request for ${row.source_image_ref}?`,
            confirmLabel: 'Retry',
        })
        if (!confirmed) return

        try {
            setRetryingIds((prev) => ({ ...prev, [row.id]: true }))
            setRetryActionError(null)
            const retried = await imageImportService.retryImportRequest(row.id)
            setImportRequests((prev) => [retried, ...prev.filter((item) => item.id !== retried.id)])
            setSelectedRequest(retried)
            toast.success('Retry submitted successfully')
        } catch (err) {
            const message = mapQuarantineImportErrorMessage(err, 'Failed to retry quarantine request')
            toast.error(message)
            if (err instanceof ImageImportApiError && (err.code === 'retry_backoff_active' || err.code === 'retry_attempt_limit_reached')) {
                setRetryActionError(message)
            }
        } finally {
            setRetryingIds((prev) => ({ ...prev, [row.id]: false }))
        }
    }

    const handleCreateEPRRequest = async (event: React.FormEvent) => {
        event.preventDefault()
        setEprFormError(null)
        if (!eprForm.eprRecordId.trim() || !eprForm.productName.trim() || !eprForm.technologyName.trim()) {
            setEprFormError('EPR Record ID, Product Name, and Technology Name are required.')
            return
        }
        try {
            setEprSubmitting(true)
            const created = await eprRegistrationService.createRequest({
                eprRecordId: eprForm.eprRecordId.trim(),
                productName: eprForm.productName.trim(),
                technologyName: eprForm.technologyName.trim(),
                businessJustification: eprForm.businessJustification.trim() || undefined,
            })
            setEprRequests((prev) => [created, ...prev.filter((row) => row.id !== created.id)])
            setEprForm({
                eprRecordId: generateEPRRecordID(),
                productName: '',
                technologyName: '',
                businessJustification: '',
            })
            setCreateEprDrawerOpen(false)
            toast.success('EPR registration request submitted for security review')
            await loadEPRRequests()
        } catch (err: any) {
            const message = err?.response?.data?.error?.message || err?.message || 'Failed to submit EPR registration request'
            setEprFormError(message)
            toast.error(message)
        } finally {
            setEprSubmitting(false)
        }
    }

    const closeCreateEPRDrawer = async () => {
        setCreateEprDrawerOpen(false)
        setEprFormError(null)
        setEprForm((prev) => ({
            ...prev,
            eprRecordId: prev.eprRecordId.trim() || generateEPRRecordID(),
        }))
        await loadEPRRequests()
    }

    const closeCreateImportDrawer = async () => {
        setCreateImportDrawerOpen(false)
        setImportDraftValues(null)
        await loadImportRequests()
    }

    const handleCloneImportRequest = (row: ImageImportRequest) => {
        setImportDraftValues({
            eprRecordId: row.epr_record_id || '',
            sourceRegistry: row.source_registry,
            sourceImageRef: row.source_image_ref,
            registryAuthId: row.registry_auth_id || undefined,
        })
        setCreateImportDrawerOpen(true)
    }

    const handleWithdrawImportRequest = async (row: ImageImportRequest) => {
        const confirmed = await confirmDialog({
            title: 'Withdraw Quarantine Request',
            message: `Withdraw pending request for ${row.source_image_ref}?`,
            confirmLabel: 'Withdraw',
        })
        if (!confirmed) return

        try {
            setWithdrawingImportIds((prev) => ({ ...prev, [row.id]: true }))
            const updated = await imageImportService.withdrawImportRequest(row.id, 'Withdrawn by tenant user')
            setImportRequests((prev) => [updated, ...prev.filter((item) => item.id !== updated.id)])
            if (selectedRequest?.id === updated.id) {
                setSelectedRequest(updated)
            }
            toast.success('Quarantine request withdrawn')
        } catch (err) {
            toast.error(mapQuarantineImportErrorMessage(err, 'Failed to withdraw quarantine request'))
        } finally {
            setWithdrawingImportIds((prev) => ({ ...prev, [row.id]: false }))
        }
    }

    const handleCloneEPRRequest = (row: EPRRegistrationRequest) => {
        setEprForm({
            eprRecordId: generateEPRRecordID(),
            productName: row.product_name || '',
            technologyName: row.technology_name || '',
            businessJustification: row.business_justification || '',
        })
        setEprFormError(null)
        setCreateEprDrawerOpen(true)
    }

    const handleWithdrawEPRRequest = async (row: EPRRegistrationRequest) => {
        const confirmed = await confirmDialog({
            title: 'Withdraw EPR Registration',
            message: `Withdraw pending EPR registration ${row.epr_record_id}?`,
            confirmLabel: 'Withdraw',
        })
        if (!confirmed) return

        try {
            setWithdrawingEprIds((prev) => ({ ...prev, [row.id]: true }))
            const updated = await eprRegistrationService.withdrawRequest(row.id, 'Withdrawn by tenant user')
            setEprRequests((prev) => [updated, ...prev.filter((item) => item.id !== updated.id)])
            if (selectedEprRequest?.id === updated.id) {
                setSelectedEprRequest(updated)
            }
            toast.success('EPR registration withdrawn')
        } catch (err: any) {
            const message = err?.response?.data?.error?.message || err?.message || 'Failed to withdraw EPR registration'
            toast.error(message)
        } finally {
            setWithdrawingEprIds((prev) => ({ ...prev, [row.id]: false }))
        }
    }

    return (
        <div className="space-y-6 px-4 py-6 sm:px-6 lg:px-8">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                    <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">
                        {isEprMode ? 'EPR Registration Requests' : 'Quarantine Requests'}
                    </h1>
                    <p className="mt-2 text-sm text-slate-700 dark:text-slate-400">
                        {isEprMode
                            ? 'Register product/technology entries and track approval status.'
                            : 'Submit and track external image quarantine requests and import/scan pipeline status.'}
                    </p>
                    <p className="mt-1 text-xs text-slate-600 dark:text-slate-400">
                        <EPRTerm /> stands for Enterprise Product/Technology Registry.
                    </p>
                </div>
                {!isEprMode ? (
                    <Link
                        to="/quarantine/releases"
                        className="inline-flex items-center rounded-md border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                    >
                        View Released Artifacts
                    </Link>
                ) : null}
            </div>

            {!isEprMode ? (
            <section className="rounded-lg border border-sky-200 bg-sky-50/70 p-4 dark:border-sky-800 dark:bg-sky-950/20">
                <h2 className="text-sm font-semibold text-sky-900 dark:text-sky-200">How Quarantine Requests Work</h2>
                <div className="mt-3 grid grid-cols-1 gap-3 md:grid-cols-3">
                    <div className="rounded-md border border-sky-200 bg-white/90 p-3 dark:border-sky-800 dark:bg-slate-900/70">
                        <p className="text-xs font-semibold uppercase tracking-wide text-sky-700 dark:text-sky-300">1. Submit</p>
                        <p className="mt-1 text-xs text-slate-700 dark:text-slate-300">
                            Provide <EPRTerm /> + image details and choose whether registry auth is needed.
                        </p>
                    </div>
                    <div className="rounded-md border border-sky-200 bg-white/90 p-3 dark:border-sky-800 dark:bg-slate-900/70">
                        <p className="text-xs font-semibold uppercase tracking-wide text-sky-700 dark:text-sky-300">2. Approve + Process</p>
                        <p className="mt-1 text-xs text-slate-700 dark:text-slate-300">
                            Request is reviewed, then import + scan pipeline runs with policy evaluation.
                        </p>
                    </div>
                    <div className="rounded-md border border-sky-200 bg-white/90 p-3 dark:border-sky-800 dark:bg-slate-900/70">
                        <p className="text-xs font-semibold uppercase tracking-wide text-sky-700 dark:text-sky-300">3. Outcome</p>
                        <p className="mt-1 text-xs text-slate-700 dark:text-slate-300">
                            Status and evidence appear in history; notifications are sent when state changes.
                        </p>
                    </div>
                </div>
            </section>
            ) : null}

            {!isEntitlementDenied ? (
                <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900">
                    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                        <div>
                            <h2 className="text-base font-semibold text-slate-900 dark:text-white">
                                {isEprMode ? 'EPR Intake' : 'Quarantine Intake'}
                            </h2>
                            <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">
                                {isEprMode
                                    ? <>Use this page to register product/technology in <EPRTerm />.</>
                                    : <>Use this page to register product/technology in <EPRTerm /> and submit quarantine requests.</>}
                            </p>
                        </div>
                        <div className="flex flex-wrap items-center gap-2">
                            {!isRequestsMode ? (
                                <button
                                    type="button"
                                    onClick={() => setCreateEprDrawerOpen(true)}
                                    className="inline-flex items-center rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm font-medium text-amber-900 hover:bg-amber-100 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200 dark:hover:bg-amber-900/35"
                                >
                                    Create EPR Registration
                                </button>
                            ) : null}
                            {!isEprMode ? (
                            <button
                                type="button"
                                onClick={() => {
                                    setImportDraftValues(null)
                                    setCreateImportDrawerOpen(true)
                                }}
                                className="inline-flex items-center rounded-md border border-emerald-300 bg-emerald-50 px-3 py-2 text-sm font-medium text-emerald-900 hover:bg-emerald-100 dark:border-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-200 dark:hover:bg-emerald-900/30"
                            >
                                    Create Quarantine Request
                                </button>
                            ) : null}
                        </div>
                    </div>
                    <div className={`mt-3 rounded-md border px-3 py-2 text-xs ${intakeBannerClasses}`}>
                        <p className="font-semibold">{intakeStateMessage.title}</p>
                        <p className="mt-1">{intakeStateMessage.message}</p>
                    </div>
                    {latestEPRRequest ? (
                        <p className="mt-3 text-xs text-slate-600 dark:text-slate-300">
                            Latest EPR request: <span className="font-semibold">{latestEPRRequest.epr_record_id}</span> ({latestEPRRequest.status})
                        </p>
                    ) : null}
                    {isRequestsMode ? (
                        <p className="mt-3 text-xs text-slate-600 dark:text-slate-300">
                            Need to register product/technology first? Go to{' '}
                            <Link to="/quarantine/epr" className="font-medium underline">
                                EPR Registrations
                            </Link>
                            .
                        </p>
                    ) : null}
                </section>
            ) : null}

            {!isRequestsMode ? (
            <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900">
                <div className="mb-3 flex items-center justify-between">
                    <h2 className="text-base font-semibold text-slate-900 dark:text-white">EPR Registration Requests</h2>
                    <button
                        type="button"
                        onClick={() => loadEPRRequests()}
                        className="rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                    >
                        Refresh
                    </button>
                </div>
                {eprLoading ? (
                    <p className="text-sm text-slate-500 dark:text-slate-400">Loading EPR requests...</p>
                ) : eprRequests.length === 0 ? (
                    <div className="rounded-md border border-dashed border-slate-300 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
                        <p className="text-sm font-medium text-slate-900 dark:text-slate-100">No EPR registration requests yet.</p>
                        <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">
                            If the product/technology is not in the enterprise registry, submit an EPR registration request first.
                        </p>
                        <button
                            type="button"
                            onClick={() => {
                                setEprForm({
                                    eprRecordId: generateEPRRecordID(),
                                    productName: '',
                                    technologyName: '',
                                    businessJustification: '',
                                })
                                setEprFormError(null)
                                setCreateEprDrawerOpen(true)
                            }}
                            className="mt-3 inline-flex items-center rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-xs font-medium text-amber-900 hover:bg-amber-100 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200 dark:hover:bg-amber-900/35"
                        >
                            Create EPR Registration
                        </button>
                    </div>
                ) : (
                    <div className="overflow-x-auto rounded-md border border-slate-200 dark:border-slate-700">
                        <table className="min-w-full text-xs">
                            <thead className="bg-slate-50 dark:bg-slate-800/70">
                                <tr className="text-left text-slate-600 dark:text-slate-300">
                                    <th className="px-3 py-2 font-medium">EPR</th>
                                    <th className="px-3 py-2 font-medium">Product / Technology</th>
                                    <th className="px-3 py-2 font-medium">Status</th>
                                    <th className="px-3 py-2 font-medium">Actions</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                                {eprRequests.map((row) => (
                                    <tr
                                        key={row.id}
                                        className="cursor-pointer bg-white hover:bg-slate-50 dark:bg-slate-900 dark:hover:bg-slate-800/70"
                                        onClick={() => setSelectedEprRequest(row)}
                                    >
                                        <td className="px-3 py-2 align-top text-slate-900 dark:text-slate-100">{row.epr_record_id}</td>
                                        <td className="px-3 py-2 align-top text-slate-700 dark:text-slate-300">
                                            <p>{row.product_name}</p>
                                            <p className="mt-0.5 text-[11px] text-slate-500 dark:text-slate-400">{row.technology_name}</p>
                                        </td>
                                        <td className="px-3 py-2 align-top">
                                            <span className={`rounded-full px-2 py-0.5 font-semibold ${row.status === 'approved'
                                                ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200'
                                                : row.status === 'rejected'
                                                    ? 'bg-rose-100 text-rose-800 dark:bg-rose-900/40 dark:text-rose-200'
                                                    : row.status === 'withdrawn'
                                                        ? 'bg-slate-100 text-slate-800 dark:bg-slate-700 dark:text-slate-200'
                                                        : 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200'
                                                }`}>
                                                {row.status}
                                            </span>
                                        </td>
                                        <td className="px-3 py-2 align-top">
                                            <div className="flex items-center gap-1.5" onClick={(event) => event.stopPropagation()}>
                                                <button
                                                    type="button"
                                                    onClick={() => setSelectedEprRequest(row)}
                                                    className="rounded-md border border-slate-300 px-2 py-1 text-[11px] font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                                                >
                                                    View
                                                </button>
                                                <button
                                                    type="button"
                                                    onClick={() => handleCloneEPRRequest(row)}
                                                    className="rounded-md border border-slate-300 px-2 py-1 text-[11px] font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                                                >
                                                    Clone
                                                </button>
                                                {row.status === 'pending' ? (
                                                    <button
                                                        type="button"
                                                        onClick={() => handleWithdrawEPRRequest(row)}
                                                        disabled={Boolean(withdrawingEprIds[row.id])}
                                                        className="rounded-md border border-rose-300 bg-rose-50 px-2 py-1 text-[11px] font-medium text-rose-700 hover:bg-rose-100 disabled:opacity-60 dark:border-rose-700 dark:bg-rose-950/30 dark:text-rose-200 dark:hover:bg-rose-900/30"
                                                    >
                                                        {withdrawingEprIds[row.id] ? 'Withdrawing...' : 'Withdraw'}
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
            </section>
            ) : null}

            {!isEprMode ? (
            <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-900">
                <div className="mb-4 flex items-center justify-between">
                    <h2 className="text-base font-semibold text-slate-900 dark:text-white">Quarantine Request Queue</h2>
                    <div className="flex items-center gap-2">
                        <button
                            type="button"
                            onClick={() => {
                                setImportDraftValues(null)
                                setCreateImportDrawerOpen(true)
                            }}
                            className="rounded-md border border-emerald-300 bg-emerald-50 px-3 py-1.5 text-xs font-medium text-emerald-900 hover:bg-emerald-100 dark:border-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-200 dark:hover:bg-emerald-900/30"
                        >
                            Create Quarantine Request
                        </button>
                        <button
                            type="button"
                            onClick={() => loadImportRequests()}
                            className="rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                        >
                            Refresh
                        </button>
                    </div>
                </div>

                {!loading && !error ? (
                    <div className="mb-4 grid grid-cols-2 gap-2 sm:grid-cols-5">
                        <div className="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200">
                            <p className="font-semibold">All</p>
                            <p className="mt-0.5 text-lg font-bold">{queueCounts.all}</p>
                        </div>
                        <div className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                            <p className="font-semibold">Pending</p>
                            <p className="mt-0.5 text-lg font-bold">{queueCounts.pending}</p>
                        </div>
                        <div className="rounded-md border border-blue-200 bg-blue-50 px-3 py-2 text-xs text-blue-800 dark:border-blue-700 dark:bg-blue-900/20 dark:text-blue-200">
                            <p className="font-semibold">In Progress</p>
                            <p className="mt-0.5 text-lg font-bold">{queueCounts.in_progress}</p>
                        </div>
                        <div className="rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-xs text-emerald-800 dark:border-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-200">
                            <p className="font-semibold">Completed</p>
                            <p className="mt-0.5 text-lg font-bold">{queueCounts.completed}</p>
                        </div>
                        <div className="rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-xs text-rose-800 dark:border-rose-700 dark:bg-rose-900/20 dark:text-rose-200">
                            <p className="font-semibold">Failed</p>
                            <p className="mt-0.5 text-lg font-bold">{queueCounts.failed}</p>
                        </div>
                    </div>
                ) : null}

                {!loading && !error ? (
                    <div className="mb-3 flex flex-wrap items-center gap-2">
                        {[
                            { key: 'all', label: 'All', count: queueCounts.all },
                            { key: 'pending', label: 'Pending', count: queueCounts.pending },
                            { key: 'in_progress', label: 'In Progress', count: queueCounts.in_progress },
                            { key: 'completed', label: 'Completed', count: queueCounts.completed },
                            { key: 'failed', label: 'Failed', count: queueCounts.failed },
                        ].map((tab) => (
                            <button
                                key={tab.key}
                                type="button"
                                onClick={() => setQueueTab(tab.key as 'all' | 'pending' | 'in_progress' | 'completed' | 'failed')}
                                className={`rounded-full border px-3 py-1 text-xs font-medium ${
                                    queueTab === tab.key
                                        ? 'border-blue-500 bg-blue-50 text-blue-700 dark:border-blue-500 dark:bg-blue-900/30 dark:text-blue-200'
                                        : 'border-slate-300 bg-white text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700'
                                }`}
                            >
                                {tab.label} ({tab.count})
                            </button>
                        ))}
                    </div>
                ) : null}

                {!loading && !error ? (
                    <div className="mb-4 grid grid-cols-1 gap-2 lg:grid-cols-4">
                        <input
                            value={queueSearch}
                            onChange={(event) => setQueueSearch(event.target.value)}
                            placeholder="Search image, EPR, registry, state"
                            className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
                        />
                        <select
                            value={queueSyncState}
                            onChange={(event) => setQueueSyncState(event.target.value)}
                            className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
                        >
                            <option value="all">All Sync States</option>
                            {availableSyncStates.map((state) => (
                                <option key={state} value={state}>
                                    {getImportSyncStateLabel(state)}
                                </option>
                            ))}
                        </select>
                        <select
                            value={queueRetryable}
                            onChange={(event) => setQueueRetryable(event.target.value as 'all' | 'retryable' | 'stable')}
                            className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
                        >
                            <option value="all">All Retry States</option>
                            <option value="retryable">Retryable</option>
                            <option value="stable">Stable</option>
                        </select>
                        <div className="flex items-center gap-2">
                            <select
                                value={queueSort}
                                onChange={(event) => setQueueSort(event.target.value as 'updated_desc' | 'updated_asc' | 'created_desc' | 'created_asc')}
                                className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
                            >
                                <option value="updated_desc">Updated (Newest)</option>
                                <option value="updated_asc">Updated (Oldest)</option>
                                <option value="created_desc">Created (Newest)</option>
                                <option value="created_asc">Created (Oldest)</option>
                            </select>
                            {hasQueueFilters ? (
                                <button
                                    type="button"
                                    onClick={() => {
                                        setQueueTab('all')
                                        setQueueSearch('')
                                        setQueueSyncState('all')
                                        setQueueRetryable('all')
                                        setQueueSort('updated_desc')
                                    }}
                                    className="rounded-md border border-slate-300 px-2 py-2 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                                >
                                    Reset
                                </button>
                            ) : null}
                        </div>
                    </div>
                ) : null}

                {loading ? (
                    <p className="text-sm text-slate-500 dark:text-slate-400">Loading request activity...</p>
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
                ) : importRequests.length === 0 ? (
                    <div className="rounded-md border border-dashed border-slate-300 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
                        <p className="text-sm font-medium text-slate-900 dark:text-slate-100">No quarantine requests yet.</p>
                        <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">
                            Create a request when product/technology already exists in <EPRTerm /> and you are ready to import and scan.
                        </p>
                        <button
                            type="button"
                            onClick={() => {
                                setImportDraftValues(null)
                                setCreateImportDrawerOpen(true)
                            }}
                            className="mt-3 inline-flex items-center rounded-md border border-emerald-300 bg-emerald-50 px-3 py-2 text-xs font-medium text-emerald-900 hover:bg-emerald-100 dark:border-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-200 dark:hover:bg-emerald-900/30"
                        >
                            Create Quarantine Request
                        </button>
                    </div>
                ) : pagedImportRequests.length === 0 ? (
                    <div className="rounded-md border border-dashed border-slate-300 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
                        <p className="text-sm font-medium text-slate-900 dark:text-slate-100">No requests match current filters.</p>
                        <p className="mt-1 text-xs text-slate-600 dark:text-slate-300">
                            Adjust search/filters or reset the queue view.
                        </p>
                    </div>
                ) : (
                    <div className="space-y-3">
                        <div className="text-xs text-slate-600 dark:text-slate-300">
                            Showing {(safeQueuePage - 1) * queuePageSize + 1}-
                            {Math.min(safeQueuePage * queuePageSize, filteredImportRequests.length)} of {filteredImportRequests.length}
                        </div>
                        {pagedImportRequests.map((row) => (
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
                                    EPR {row.epr_record_id} • Updated {new Date(row.updated_at).toLocaleString()}
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
                                    <Link
                                        to={`/quarantine/requests/${row.id}`}
                                        className="rounded-md border border-blue-300 bg-blue-50 px-2.5 py-1 text-xs font-medium text-blue-800 hover:bg-blue-100 dark:border-blue-700 dark:bg-blue-950/30 dark:text-blue-200 dark:hover:bg-blue-900/40"
                                    >
                                        View Status
                                    </Link>
                                    <button
                                        type="button"
                                        onClick={() => setSelectedRequest(row)}
                                        className="rounded-md border border-slate-300 px-2.5 py-1 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                                    >
                                        Details
                                    </button>
                                    {row.retryable && row.status === 'failed' ? (
                                        <button
                                            type="button"
                                            onClick={() => handleRetry(row)}
                                            disabled={Boolean(retryingIds[row.id])}
                                            className="rounded-md border border-amber-300 bg-amber-50 px-2.5 py-1 text-xs font-medium text-amber-800 hover:bg-amber-100 disabled:opacity-60 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200 dark:hover:bg-amber-900/40"
                                        >
                                            {retryingIds[row.id] ? 'Retrying...' : 'Retry'}
                                        </button>
                                    ) : null}
                                    <button
                                        type="button"
                                        onClick={() => handleCloneImportRequest(row)}
                                        className="rounded-md border border-slate-300 px-2.5 py-1 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                                    >
                                        Clone
                                    </button>
                                    {row.status === 'pending' ? (
                                        <button
                                            type="button"
                                            onClick={() => handleWithdrawImportRequest(row)}
                                            disabled={Boolean(withdrawingImportIds[row.id])}
                                            className="rounded-md border border-rose-300 bg-rose-50 px-2.5 py-1 text-xs font-medium text-rose-700 hover:bg-rose-100 disabled:opacity-60 dark:border-rose-700 dark:bg-rose-950/30 dark:text-rose-200 dark:hover:bg-rose-900/30"
                                        >
                                            {withdrawingImportIds[row.id] ? 'Withdrawing...' : 'Withdraw'}
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
                        <div className="flex items-center justify-between border-t border-slate-200 pt-3 dark:border-slate-700">
                            <button
                                type="button"
                                disabled={safeQueuePage <= 1}
                                onClick={() => setQueuePage((prev) => Math.max(1, prev - 1))}
                                className="rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                            >
                                Previous
                            </button>
                            <p className="text-xs text-slate-600 dark:text-slate-300">
                                Page {safeQueuePage} of {queueTotalPages}
                            </p>
                            <button
                                type="button"
                                disabled={safeQueuePage >= queueTotalPages}
                                onClick={() => setQueuePage((prev) => Math.min(queueTotalPages, prev + 1))}
                                className="rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                            >
                                Next
                            </button>
                        </div>
                    </div>
                )}
            </section>
            ) : null}

            <Drawer
                isOpen={Boolean(selectedRequest)}
                onClose={() => setSelectedRequest(null)}
                title="Quarantine Request Detail"
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
                                <p><span className="font-semibold">EPR Record:</span> {selectedRequest.epr_record_id}</p>
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

            <Drawer
                isOpen={Boolean(selectedEprRequest)}
                onClose={() => setSelectedEprRequest(null)}
                title="EPR Registration Detail"
                description={selectedEprRequest ? selectedEprRequest.epr_record_id : undefined}
                width="lg"
            >
                {selectedEprRequest ? (
                    <div className="space-y-4">
                        <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
                            <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">EPR Registration</p>
                            <div className="mt-2 space-y-1 text-xs text-slate-700 dark:text-slate-300">
                                <p><span className="font-semibold">EPR ID:</span> {selectedEprRequest.epr_record_id}</p>
                                <p><span className="font-semibold">Status:</span> {selectedEprRequest.status}</p>
                                <p><span className="font-semibold">Product:</span> {selectedEprRequest.product_name}</p>
                                <p><span className="font-semibold">Technology:</span> {selectedEprRequest.technology_name}</p>
                                {selectedEprRequest.decision_reason ? <p><span className="font-semibold">Decision Reason:</span> {selectedEprRequest.decision_reason}</p> : null}
                                <p><span className="font-semibold">Submitted:</span> {new Date(selectedEprRequest.created_at).toLocaleString()}</p>
                                <p><span className="font-semibold">Updated:</span> {new Date(selectedEprRequest.updated_at).toLocaleString()}</p>
                            </div>
                        </div>
                        {selectedEprRequest.business_justification ? (
                            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
                                <p className="text-xs font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-200">Business Justification</p>
                                <p className="mt-2 text-xs text-slate-600 dark:text-slate-300">{selectedEprRequest.business_justification}</p>
                            </div>
                        ) : null}
                    </div>
                ) : null}
            </Drawer>

            <Drawer
                isOpen={createEprDrawerOpen}
                onClose={closeCreateEPRDrawer}
                title="Create EPR Registration"
                description="Register product/technology for security review before quarantine request approval."
                width="lg"
            >
                <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
                    Submit this when product/technology is not yet present in <EPRTerm />.
                </div>
                {eprFormError ? (
                    <p className="mt-3 rounded border border-rose-300 bg-rose-50 px-2 py-1 text-xs text-rose-700 dark:border-rose-700 dark:bg-rose-950/30 dark:text-rose-200">
                        {eprFormError}
                    </p>
                ) : null}
                <form onSubmit={handleCreateEPRRequest} className="mt-3 space-y-3">
                    <div className="space-y-1">
                        <label className="text-xs font-medium text-slate-700 dark:text-slate-300">EPR Record ID</label>
                        <input
                            value={eprForm.eprRecordId}
                            onChange={(e) => setEprForm((prev) => ({ ...prev, eprRecordId: e.target.value }))}
                            placeholder="EPR Record ID (for example: EPR-00123)"
                            className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
                        />
                        <div className="flex justify-end">
                            <button
                                type="button"
                                onClick={() => setEprForm((prev) => ({ ...prev, eprRecordId: generateEPRRecordID() }))}
                                className="rounded-md border border-slate-300 px-2 py-1 text-xs font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-800"
                            >
                                Regenerate ID
                            </button>
                        </div>
                    </div>
                    <div className="space-y-1">
                        <label className="text-xs font-medium text-slate-700 dark:text-slate-300">Product Name</label>
                        <input
                            value={eprForm.productName}
                            onChange={(e) => setEprForm((prev) => ({ ...prev, productName: e.target.value }))}
                            placeholder="Product Name"
                            className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
                        />
                    </div>
                    <div className="space-y-1">
                        <label className="text-xs font-medium text-slate-700 dark:text-slate-300">Technology Name</label>
                        <input
                            value={eprForm.technologyName}
                            onChange={(e) => setEprForm((prev) => ({ ...prev, technologyName: e.target.value }))}
                            placeholder="Technology Name"
                            className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
                        />
                    </div>
                    <div className="space-y-1">
                        <label className="text-xs font-medium text-slate-700 dark:text-slate-300">Business Justification (optional)</label>
                        <textarea
                            value={eprForm.businessJustification}
                            onChange={(e) => setEprForm((prev) => ({ ...prev, businessJustification: e.target.value }))}
                            placeholder="Business justification"
                            rows={3}
                            className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100"
                        />
                    </div>
                    <button
                        type="submit"
                        disabled={eprSubmitting}
                        className="inline-flex items-center rounded-md bg-amber-600 px-3 py-2 text-sm font-medium text-white hover:bg-amber-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500/50 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                        {eprSubmitting ? 'Submitting...' : 'Submit EPR Registration'}
                    </button>
                </form>
            </Drawer>

            <Drawer
                isOpen={createImportDrawerOpen}
                onClose={closeCreateImportDrawer}
                title="Create Quarantine Request"
                description="Submit source image details for import, scan, and policy evaluation."
                width="xl"
            >
                <ExternalImportRequestForm
                    initialValues={importDraftValues || undefined}
                    onCreated={(created) => {
                        setImportRequests((prev) => [created, ...prev.filter((row) => row.id !== created.id)])
                        setCreateImportDrawerOpen(false)
                        setImportDraftValues(null)
                        loadImportRequests()
                    }}
                />
            </Drawer>
        </div>
    )
}

export default QuarantineRequestsPage
